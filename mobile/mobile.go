// Package mobile exposes a gomobile-bindable API for embedding lsbot
// in iOS (XCFramework) and Android (AAR) apps.
//
// Build for iOS:
//
//	gomobile bind -target ios -o dist/lsbot.xcframework ./mobile/
//
// Build for Android:
//
//	gomobile bind -target android -o dist/lsbot.aar ./mobile/
package mobile

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ruilisi/lsbot/internal/agent"
	"github.com/ruilisi/lsbot/internal/agent/mcpclient"
	"github.com/ruilisi/lsbot/internal/config"
	cronpkg "github.com/ruilisi/lsbot/internal/cron"
	"github.com/ruilisi/lsbot/internal/e2e"
	"github.com/ruilisi/lsbot/internal/mcp"
	"github.com/ruilisi/lsbot/internal/platforms/relay"
	"github.com/ruilisi/lsbot/internal/platforms/webapp"
	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/skills"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// OutputCallback receives log lines from the running lsbot instance.
// gomobile exports this as a protocol/interface.
type OutputCallback interface {
	OnLine(line string)
}

var (
	mu             sync.Mutex
	cancelFn       context.CancelFunc
	running        bool
	outputCallback OutputCallback
	logWriter      io.Writer = os.Stderr
)

// SetDataDir redirects all lsbot data paths (skills, cron DB, E2E keys) to dir.
// Call this before Start() on iOS to use the writable Documents directory.
func SetDataDir(dir string) {
	config.SetDataDir(dir)
}

// GatewayPort is the port used when running in gateway (local) mode.
// Set before calling Start if you need a specific port; 0 = auto-assign.
var GatewayPort = 18080

// GetGatewayPort returns the actual port the gateway is listening on (useful after auto-assign).
func GetGatewayPort() int {
	return GatewayPort
}

// Version returns the lsbot version string (set via ldflags at build time).
func Version() string {
	return mcp.ServerVersion
}

// IsRunning returns true if lsbot is currently running.
func IsRunning() bool {
	mu.Lock()
	defer mu.Unlock()
	return running
}

// SetOutputCallback registers a callback that receives each log/output line.
// Call before Start to capture output. Pass nil to unregister.
func SetOutputCallback(cb OutputCallback) {
	mu.Lock()
	defer mu.Unlock()
	outputCallback = cb
	if cb != nil {
		// Redirect the standard logger to our callback writer
		logWriter = &callbackWriter{cb: cb}
		log.SetOutput(logWriter)
	} else {
		log.SetOutput(os.Stderr)
		logWriter = os.Stderr
	}
}

// Start starts lsbot using the config file at configPath.
// It reads the mode from the config (relay or gateway) and starts accordingly.
// This call blocks until Stop() is called or a fatal error occurs.
// Call from a goroutine / background thread — never from the main thread.
//
// On iOS the configPath should be an absolute path inside the app sandbox, e.g.:
//
//	NSHomeDirectory() + "/.lsbot.yaml"
func Start(configPath string) error {
	mu.Lock()
	if running {
		mu.Unlock()
		return fmt.Errorf("lsbot is already running")
	}
	running = true
	mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			emit("[lsbot] panic: %v", r)
		}
		mu.Lock()
		running = false
		cancelFn = nil
		mu.Unlock()
	}()

	// Point config at the provided path
	if configPath != "" {
		config.SetConfigPath(configPath)
	}

	cfg, err := config.Load()
	if err != nil {
		// No config yet — run in bot-page-only relay mode (no platform needed)
		cfg = &config.Config{}
	}

	// Ensure bot_id is set (required for relay E2EE and bot page)
	if cfg.BotID == "" {
		cfg.BotID = uuid.New().String()
		_ = cfg.Save()
	}

	// Determine run mode
	mode := strings.ToLower(cfg.Mode)
	if mode == "" {
		mode = "relay" // default on mobile
	}

	ctx, cancel := context.WithCancel(context.Background())
	mu.Lock()
	cancelFn = cancel
	mu.Unlock()
	defer cancel()

	emit("[lsbot] starting in %s mode (version %s)", mode, mcp.ServerVersion)

	switch mode {
	case "relay":
		emit("[lsbot] calling startRelay...")
		err := startRelay(ctx, cfg)
		emit("[lsbot] startRelay returned: %v", err)
		return err
	case "gateway", "local":
		return startGateway(ctx, cfg)
	default:
		return fmt.Errorf("unsupported mode %q on mobile (use relay or gateway)", mode)
	}
}

// Stop gracefully stops the running lsbot instance.
// Safe to call even if lsbot is not running.
func Stop() {
	mu.Lock()
	cancel := cancelFn
	mu.Unlock()
	if cancel != nil {
		emit("[lsbot] stopping...")
		cancel()
	}
}

// RunCommand runs an lsbot subcommand and returns stdout.
// args is a space-separated string, e.g. "skills list --json".
// This is synchronous and may block for a few seconds.
func RunCommand(args string) (string, error) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty args")
	}

	cfg, _ := config.Load()

	switch parts[0] {
	case "skills":
		return runSkillsCommand(parts[1:], cfg)
	case "version":
		return fmt.Sprintf("lsbot %s", mcp.ServerVersion), nil
	case "db":
		return runDBCommand(parts[1:])
	default:
		return "", fmt.Errorf("command %q not supported in mobile mode", parts[0])
	}
}

// runDBCommand tests SQLite connectivity.
// Usage: db test  — creates a test table, inserts a row, queries it.
func runDBCommand(args []string) (string, error) {
	if len(args) == 0 || args[0] != "test" {
		return "", fmt.Errorf("usage: db test")
	}
	dbDir := filepath.Join(config.HubDir(), "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dbDir, err)
	}
	dbPath := filepath.Join(dbDir, "_test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", fmt.Errorf("sql.Open: %w", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _test (id INTEGER PRIMARY KEY, val TEXT)`); err != nil {
		return "", fmt.Errorf("create table: %w", err)
	}
	if _, err := db.Exec(`INSERT INTO _test (val) VALUES ('ok')`); err != nil {
		return "", fmt.Errorf("insert: %w", err)
	}
	var val string
	if err := db.QueryRow(`SELECT val FROM _test LIMIT 1`).Scan(&val); err != nil {
		return "", fmt.Errorf("query: %w", err)
	}
	os.Remove(dbPath)
	return fmt.Sprintf("SQLite OK — path=%s val=%s", dbPath, val), nil
}

// --- relay startup ---

func startRelay(ctx context.Context, cfg *config.Config) error {
	emit("[lsbot] resolving AI config...")
	provider, apiKey, baseURL, model := resolveAI(cfg)
	emit("[lsbot] provider=%s hasKey=%v", provider, apiKey != "")

	if apiKey == "" {
		return fmt.Errorf("AI API key not configured — edit .lsbot.yaml and set api_key")
	}

	emit("[lsbot] setting up E2E key...")
	e2eKeyFile := cfg.E2EKeyFile
	if e2eKeyFile == "" {
		e2eKeyFile = filepath.Join(config.HubDir(), "e2e.pem")
	}
	// Auto-generate key on first run
	if _, err := os.Stat(e2eKeyFile); os.IsNotExist(err) {
		if _, err2 := e2e.GenerateOrLoadKeyPair(e2eKeyFile); err2 != nil {
			emit("[lsbot] warning: E2EE key generation failed: %v", err2)
			e2eKeyFile = ""
		} else {
			emit("[lsbot] E2EE key generated at %s", e2eKeyFile)
			cfg.E2EKeyFile = e2eKeyFile
			_ = cfg.Save()
		}
	}

	emit("[lsbot] creating agent...")
	// MCP servers from config
	var mcpServers []mcpclient.ServerConfig
	for _, s := range cfg.AI.MCPServers {
		mcpServers = append(mcpServers, mcpclient.ServerConfig{
			Name: s.Name, Command: s.Command, Args: s.Args, Env: s.Env, URL: s.URL,
		})
	}

	agentCfg := agent.Config{
		Provider:         provider,
		APIKey:           apiKey,
		BaseURL:          baseURL,
		Model:            model,
		AllowedPaths:     cfg.Security.AllowedPaths,
		DisableFileTools: cfg.Security.DisableFileTools,
		MaxToolRounds:    cfg.AI.MaxRounds,
		CallTimeoutSecs:  cfg.AI.CallTimeoutSecs,
		MCPServers:       mcpServers,
	}

	aiAgent, err := agent.New(agentCfg)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}
	emit("[lsbot] agent created, setting up router...")

	pool := agent.NewAgentPool(aiAgent, agentCfg, cfg)
	r := router.New(pool.HandleMessage)

	emit("[lsbot] setting up cron...")
	// Cron scheduler
	cronPath := filepath.Join(config.HubDir(), "cron.db")
	if cronStore, err := cronpkg.NewStore(cronPath); err == nil {
		cronNotifier := agent.NewRouterCronNotifier(r)
		cronScheduler := cronpkg.NewScheduler(cronStore, aiAgent, aiAgent, cronNotifier)
		aiAgent.SetCronScheduler(cronScheduler)
		_ = cronScheduler.Start()
		defer cronScheduler.Stop()
	}

	emit("[lsbot] connecting relay...")
	// Relay platform
	relayUserID := cfg.Relay.UserID
	relayPlatform := cfg.Relay.Platform
	if relayUserID == "" {
		relayUserID = cfg.BotID
	}

	relayInst, err := relay.New(relay.Config{
		UserID:     relayUserID,
		Platform:   relayPlatform,
		ServerURL:  cfg.Relay.ServerURL,
		WebhookURL: cfg.Relay.WebhookURL,
		AIProvider: provider,
		AIModel:    model,
		BotID:      cfg.BotID,
		E2EKeyFile: e2eKeyFile,
	})
	if err != nil {
		return fmt.Errorf("failed to create relay: %w", err)
	}
	r.Register(relayInst)

	if err := r.Start(ctx); err != nil {
		return fmt.Errorf("relay start failed: %w", err)
	}

	emit("[lsbot] relay connected — bot ID: %s", cfg.BotID)
	emit("[lsbot:bot_id] %s", cfg.BotID)
	// Emit public key so the mobile app can do in-process E2EE chat
	if e2eKeyFile != "" {
		if privKey, err2 := e2e.GenerateOrLoadKeyPair(e2eKeyFile); err2 == nil {
			pubB64 := e2e.PublicKeyToBase64(privKey.PublicKey())
			emit("[lsbot:pubkey] %s", pubB64)
		}
	}

	// Block until context cancelled (Stop() called)
	<-ctx.Done()
	emit("[lsbot] shutting down...")
	r.Stop()
	return nil
}

// --- skills commands ---

func runSkillsCommand(args []string, cfg *config.Config) (string, error) {
	if len(args) == 0 || args[0] == "list" {
		isJSON := false
		for _, a := range args {
			if a == "--json" {
				isJSON = true
			}
		}
		report := skills.BuildStatusReport(cfg.Skills.Disabled, cfg.Skills.ExtraDirs)
		return skills.FormatList(report, skills.FormatListOptions{JSON: isJSON}), nil
	}

	if args[0] == "enable" && len(args) >= 2 {
		return "", toggleSkill(args[1], true, cfg)
	}
	if args[0] == "disable" && len(args) >= 2 {
		return "", toggleSkill(args[1], false, cfg)
	}
	if args[0] == "download" {
		emit("[lsbot] fetching skills from GitHub...")
		count, err := skills.DownloadBundledSkills("", func(skill string) {
			emit("[lsbot] installed skill: %s", skill)
		})
		if err != nil {
			return "", err
		}
		msg := fmt.Sprintf("[lsbot] downloaded %d skills to %s", count, config.SkillsDir())
		emit(msg)
		return msg, nil
	}

	return "", fmt.Errorf("unknown skills subcommand: %s", args[0])
}

func toggleSkill(name string, enable bool, cfg *config.Config) error {
	if enable {
		// Remove from disabled list
		newDisabled := []string{}
		for _, d := range cfg.Skills.Disabled {
			if d != name {
				newDisabled = append(newDisabled, d)
			}
		}
		cfg.Skills.Disabled = newDisabled
	} else {
		// Add to disabled list if not already there
		found := false
		for _, d := range cfg.Skills.Disabled {
			if d == name {
				found = true
				break
			}
		}
		if !found {
			cfg.Skills.Disabled = append(cfg.Skills.Disabled, name)
		}
	}
	return cfg.Save()
}

// --- helpers ---

func resolveAI(cfg *config.Config) (provider, apiKey, baseURL, model string) {
	// 1. Try named provider / relay.provider / ai.provider
	if resolved, found := cfg.ResolveProvider(""); found {
		provider = resolved.Provider
		apiKey = resolved.APIKey
		baseURL = resolved.BaseURL
		model = resolved.Model
	}
	// 2. If still no key, use the default agent directly
	if apiKey == "" {
		if id := cfg.DefaultAgentID(); id != "" {
			if resolved, found := cfg.ResolveProvider(id); found {
				provider = resolved.Provider
				apiKey = resolved.APIKey
				baseURL = resolved.BaseURL
				model = resolved.Model
			}
		}
	}
	// 3. Fall back to flat ai: block
	if provider == "" {
		provider = cfg.AI.Provider
	}
	if apiKey == "" {
		apiKey = cfg.AI.APIKey
	}
	if baseURL == "" {
		baseURL = cfg.AI.BaseURL
	}
	if model == "" {
		model = cfg.AI.Model
	}
	// 4. Fall back to env vars
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	// Trim whitespace/newlines from key (YAML multiline quirk)
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)
	return
}

func emit(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	log.Println(line)
	mu.Lock()
	cb := outputCallback
	mu.Unlock()
	if cb != nil {
		cb.OnLine(line)
	}
}

// callbackWriter forwards written bytes line-by-line to the OutputCallback.
type callbackWriter struct {
	cb  OutputCallback
	buf bytes.Buffer
}

func (w *callbackWriter) Write(p []byte) (n int, err error) {
	w.buf.Write(p)
	for {
		line, rest, found := strings.Cut(w.buf.String(), "\n")
		if !found {
			break
		}
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			w.cb.OnLine(trimmed)
		}
		w.buf.Reset()
		w.buf.WriteString(rest)
	}
	return len(p), nil
}

// --- gateway startup ---

func startGateway(ctx context.Context, cfg *config.Config) error {
	provider, apiKey, baseURL, model := resolveAI(cfg)
	if apiKey == "" {
		return fmt.Errorf("AI API key not configured — edit .lsbot.yaml and set api_key")
	}

	agentCfg := agent.Config{
		Provider:         provider,
		APIKey:           apiKey,
		BaseURL:          baseURL,
		Model:            model,
		AllowedPaths:     cfg.Security.AllowedPaths,
		DisableFileTools: cfg.Security.DisableFileTools,
		MaxToolRounds:    cfg.AI.MaxRounds,
		CallTimeoutSecs:  cfg.AI.CallTimeoutSecs,
	}

	aiAgent, err := agent.New(agentCfg)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	pool := agent.NewAgentPool(aiAgent, agentCfg, cfg)
	r := router.New(pool.HandleMessage)

	port := GatewayPort
	wa, err := webapp.New(webapp.Config{Port: port})
	if err != nil {
		return fmt.Errorf("failed to create webapp: %w", err)
	}
	r.Register(wa)

	if err := r.Start(ctx); err != nil {
		return fmt.Errorf("gateway start failed: %w", err)
	}

	emit("[lsbot:gateway_port] %d", port)
	emit("[lsbot] gateway ready on ws://localhost:%d/ws", port)

	<-ctx.Done()
	emit("[lsbot] shutting down gateway...")
	r.Stop()
	return nil
}
