package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/ruilisi/lsbot/internal/termui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and configuration",
	Long:  `Run diagnostic checks on configuration, credentials, connectivity, and required tools.`,
	Run:   runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type checkResult struct {
	name   string
	ok     bool
	detail string
}

func runDoctor(cmd *cobra.Command, args []string) {
	fmt.Println("lsbot doctor")
	fmt.Println("=================")
	fmt.Printf("OS: %s/%s, Go: %s\n\n", runtime.GOOS, runtime.GOARCH, runtime.Version())

	var results []checkResult

	// 1. Config file
	cfg, err := config.Load()
	if err != nil {
		results = append(results, checkResult{"Config file (~/.lsbot.yaml)", false, err.Error()})
	} else {
		_, statErr := os.Stat(config.ConfigPath())
		if os.IsNotExist(statErr) {
			results = append(results, checkResult{"Config file (~/.lsbot.yaml)", true, "not found, using defaults"})
		} else {
			results = append(results, checkResult{"Config file (~/.lsbot.yaml)", true, "loaded"})
		}
	}

	// 2. AI provider API key
	apiKey := os.Getenv("AI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_OAUTH_TOKEN")
	}
	if apiKey == "" && cfg != nil {
		apiKey = cfg.AI.APIKey
	}
	if apiKey == "" {
		results = append(results, checkResult{"AI API key", false, "not set (AI_API_KEY, ANTHROPIC_API_KEY, or config)"})
	} else {
		masked := apiKey[:min(8, len(apiKey))] + "..."
		provider := cfg.AI.Provider
		if provider == "" {
			provider = "claude"
		}
		results = append(results, checkResult{"AI API key", true, fmt.Sprintf("set (%s, provider: %s)", masked, provider)})
	}

	// 3. AI provider connectivity
	if apiKey != "" && cfg != nil {
		provider := cfg.AI.Provider
		if provider == "" {
			provider = "claude"
		}
		ok, detail := checkAIConnectivity(provider, apiKey, cfg.AI.BaseURL)
		results = append(results, checkResult{"AI provider connectivity", ok, detail})
	} else {
		results = append(results, checkResult{"AI provider connectivity", false, "skipped (no API key)"})
	}

	// 4. Platform credentials
	if cfg != nil {
		platforms := detectPlatforms(cfg)
		if len(platforms) == 0 {
			results = append(results, checkResult{"Platform credentials", false, "none configured"})
		} else {
			results = append(results, checkResult{"Platform credentials", true, strings.Join(platforms, ", ")})
		}
	}

	// 5. Required binaries
	binaries := []string{"gh", "chrome", "claude"}
	for _, bin := range binaries {
		if _, err := exec.LookPath(bin); err != nil {
			results = append(results, checkResult{fmt.Sprintf("Binary: %s", bin), false, "not found in PATH"})
		} else {
			results = append(results, checkResult{fmt.Sprintf("Binary: %s", bin), true, "found"})
		}
	}

	// 6. Browser CDP connectivity
	if cfg != nil && cfg.Browser.CDPURL != "" {
		conn, err := net.DialTimeout("tcp", cfg.Browser.CDPURL, 3*time.Second)
		if err != nil {
			results = append(results, checkResult{"Browser CDP", false, fmt.Sprintf("%s unreachable", cfg.Browser.CDPURL)})
		} else {
			conn.Close()
			results = append(results, checkResult{"Browser CDP", true, cfg.Browser.CDPURL})
		}
	}

	// 7. MCP servers
	if cfg != nil && len(cfg.AI.MCPServers) > 0 {
		for _, s := range cfg.AI.MCPServers {
			if s.URL != "" {
				ok := checkURL(s.URL)
				status := "reachable"
				if !ok {
					status = "unreachable"
				}
				results = append(results, checkResult{fmt.Sprintf("MCP server: %s", s.Name), ok, status})
			} else if s.Command != "" {
				if _, err := exec.LookPath(s.Command); err != nil {
					results = append(results, checkResult{fmt.Sprintf("MCP server: %s", s.Name), false, fmt.Sprintf("command %q not found", s.Command)})
				} else {
					results = append(results, checkResult{fmt.Sprintf("MCP server: %s", s.Name), true, fmt.Sprintf("command %q available", s.Command)})
				}
			}
		}
	}

	// 8. Temp dir writable
	tmpFile, err := os.CreateTemp("", "lingti-doctor-*")
	if err != nil {
		results = append(results, checkResult{"Temp directory", false, err.Error()})
	} else {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		results = append(results, checkResult{"Temp directory", true, "writable"})
	}

	// Print results
	fmt.Println("Checks:")
	passed, failed := 0, 0
	for _, r := range results {
		icon := termui.Colorize(termui.Green, "✓")
		if !r.ok {
			icon = termui.Colorize(termui.Red, "✗")
			failed++
		} else {
			passed++
		}
		fmt.Printf("  %s %s — %s\n", icon, r.name, r.detail)
	}

	fmt.Printf("\n%d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func detectPlatforms(cfg *config.Config) []string {
	var platforms []string
	if cfg.Platforms.Slack.BotToken != "" {
		platforms = append(platforms, "slack")
	}
	if cfg.Platforms.Telegram.Token != "" {
		platforms = append(platforms, "telegram")
	}
	if cfg.Platforms.Discord.Token != "" {
		platforms = append(platforms, "discord")
	}
	if cfg.Platforms.Feishu.AppID != "" {
		platforms = append(platforms, "feishu")
	}
	if cfg.Platforms.DingTalk.ClientID != "" {
		platforms = append(platforms, "dingtalk")
	}
	if cfg.Platforms.WeCom.CorpID != "" {
		platforms = append(platforms, "wecom")
	}
	if cfg.Platforms.WeChat.AppID != "" {
		platforms = append(platforms, "wechat")
	}
	if cfg.Platforms.WhatsApp.PhoneNumberID != "" {
		platforms = append(platforms, "whatsapp")
	}
	if cfg.Platforms.LINE.ChannelToken != "" {
		platforms = append(platforms, "line")
	}
	if cfg.Platforms.Teams.AppID != "" {
		platforms = append(platforms, "teams")
	}
	if cfg.Platforms.Matrix.AccessToken != "" {
		platforms = append(platforms, "matrix")
	}
	if cfg.Platforms.GoogleChat.ProjectID != "" {
		platforms = append(platforms, "googlechat")
	}
	if cfg.Platforms.Mattermost.Token != "" {
		platforms = append(platforms, "mattermost")
	}
	if cfg.Platforms.IMessage.BlueBubblesURL != "" {
		platforms = append(platforms, "imessage")
	}
	if cfg.Platforms.Signal.APIURL != "" {
		platforms = append(platforms, "signal")
	}
	if cfg.Platforms.Twitch.Token != "" {
		platforms = append(platforms, "twitch")
	}
	if cfg.Platforms.NOSTR.PrivateKey != "" {
		platforms = append(platforms, "nostr")
	}
	if cfg.Platforms.Zalo.AppID != "" {
		platforms = append(platforms, "zalo")
	}
	if cfg.Platforms.Nextcloud.ServerURL != "" {
		platforms = append(platforms, "nextcloud")
	}
	return platforms
}

func checkAIConnectivity(provider, apiKey, baseURL string) (bool, string) {
	var url string
	switch strings.ToLower(provider) {
	case "claude", "anthropic", "":
		url = "https://api.anthropic.com/v1/messages"
		if baseURL != "" {
			url = strings.TrimRight(baseURL, "/") + "/messages"
		}
	case "deepseek":
		url = "https://api.deepseek.com/v1/models"
		if baseURL != "" {
			url = strings.TrimRight(baseURL, "/") + "/models"
		}
	case "openai":
		url = "https://api.openai.com/v1/models"
		if baseURL != "" {
			url = strings.TrimRight(baseURL, "/") + "/models"
		}
	default:
		return true, "skipped (unknown provider)"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if strings.ToLower(provider) == "claude" || strings.ToLower(provider) == "anthropic" || provider == "" {
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	// 401/403 means we reached the API but key is invalid
	// 200/405 means connectivity works
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return false, fmt.Sprintf("reachable but auth failed (HTTP %d)", resp.StatusCode)
	}
	return true, fmt.Sprintf("reachable (HTTP %d)", resp.StatusCode)
}

func checkURL(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}
