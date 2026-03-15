package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/pltanton/lingti-bot/internal/agent"
	"github.com/google/uuid"
	"github.com/pltanton/lingti-bot/internal/agent/mcpclient"
	"github.com/pltanton/lingti-bot/internal/config"
	cronpkg "github.com/pltanton/lingti-bot/internal/cron"
	"github.com/pltanton/lingti-bot/internal/platforms/relay"
	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/spf13/cobra"
)

var (
	relayUserID        string
	relayPlatform      string
	relayHost          string
	relayServerURL     string
	relayWebhookURL    string
	relayRefreshBotID  bool
	relayInsecure      bool
	relayE2EKeyFile    string
	relayAIProvider    string
	relayAPIKey        string
	relayBaseURL       string
	relayModel         string
	relayInstructions  string
	relayMaxRounds     int
	relayCallTimeout   int
	// WeCom credentials for cloud relay
	relayWeComCorpID  string
	relayWeComAgentID string
	relayWeComSecret  string
	relayWeComToken   string
	relayWeComAESKey  string
	// WeChat OA credentials
	relayWeChatAppID     string
	relayWeChatAppSecret string
)

var relayCmd = &cobra.Command{
	Use:   "relay",
	Short: "Connect to the cloud relay service",
	Long: `Connect to the lingti-bot cloud relay service to process messages
using your local AI API key.

This allows you to use the official lingti-bot service on Feishu/Slack/WeChat
without registering your own bot application.

User Flow (Feishu/Slack/WeChat):
  1. Follow the official lingti-bot on Feishu/Slack/WeChat
  2. Send /whoami to get your user ID
  3. Run: lingti-bot relay --user-id <ID> --platform feishu
  4. Messages are processed locally with your AI API key
  5. Responses are sent back through the relay service

WeCom Cloud Relay:
  For WeCom, no user-id is needed - just provide your credentials.
  This command handles both callback verification AND message processing.

  lingti-bot relay --platform wecom \
    --wecom-corp-id YOUR_CORP_ID \
    --wecom-agent-id YOUR_AGENT_ID \
    --wecom-secret YOUR_SECRET \
    --wecom-token YOUR_TOKEN \
    --wecom-aes-key YOUR_AES_KEY \
    --provider deepseek \
    --api-key YOUR_API_KEY

  1. Run this command first
  2. Configure callback URL in WeCom: https://bot.lingti.com/wecom
  3. Save config in WeCom - verification will succeed automatically
  4. Messages will be processed with your AI provider

Required:
  --user-id     Your user ID from /whoami (not needed for WeCom)
  --platform    Platform type: feishu, slack, wechat, or wecom
  --api-key     AI API key (or AI_API_KEY env)

WeCom Required (when platform=wecom):
  --wecom-corp-id    WeCom Corp ID (or WECOM_CORP_ID env)
  --wecom-agent-id   WeCom Agent ID (or WECOM_AGENT_ID env)
  --wecom-secret     WeCom Secret (or WECOM_SECRET env)
  --wecom-token      WeCom Callback Token (or WECOM_TOKEN env)
  --wecom-aes-key    WeCom Encoding AES Key (or WECOM_AES_KEY env)

Environment variables:
  RELAY_USER_ID        Alternative to --user-id
  RELAY_PLATFORM       Alternative to --platform
  RELAY_SERVER_URL     Custom WebSocket server URL
  RELAY_WEBHOOK_URL    Custom webhook URL
  AI_PROVIDER          AI provider: claude, deepseek, kimi, qwen (default: claude)
  AI_API_KEY           AI API key
  AI_BASE_URL          Custom API base URL
  AI_MODEL             Model name`,
	Run: runRelay,
}

func init() {
	rootCmd.AddCommand(relayCmd)

	relayCmd.Flags().StringVar(&relayUserID, "user-id", "", "User ID from /whoami (required, or RELAY_USER_ID env)")
	relayCmd.Flags().StringVar(&relayPlatform, "platform", "", "Platform: feishu, slack, wechat, or wecom (required, or RELAY_PLATFORM env)")
	relayCmd.Flags().StringVar(&relayHost, "host", "", "Base URL of the relay server (e.g. http://localhost:8080 or https://bot.lingti.com); sets --server and --webhook")
	relayCmd.Flags().StringVar(&relayServerURL, "server", "", "WebSocket URL (default: wss://bot.lingti.com/ws, or RELAY_SERVER_URL env)")
	relayCmd.Flags().StringVar(&relayWebhookURL, "webhook", "", "Webhook URL (default: https://bot.lingti.com/webhook, or RELAY_WEBHOOK_URL env)")
	relayCmd.Flags().StringVar(&relayAIProvider, "provider", "", "AI provider: claude, deepseek, kimi, qwen (or AI_PROVIDER env)")
	relayCmd.Flags().StringVar(&relayAPIKey, "api-key", "", "AI API key (or AI_API_KEY env)")
	relayCmd.Flags().StringVar(&relayBaseURL, "base-url", "", "Custom API base URL (or AI_BASE_URL env)")
	relayCmd.Flags().StringVar(&relayModel, "model", "", "Model name (or AI_MODEL env)")
	relayCmd.Flags().StringVar(&relayInstructions, "instructions", "", "Path to custom instructions file appended to system prompt")
	relayCmd.Flags().IntVar(&relayMaxRounds, "max-rounds", 0, "Max tool-call iterations per message (default 100, or AI_MAX_ROUNDS env)")
	relayCmd.Flags().IntVar(&relayCallTimeout, "call-timeout", 0, "Base timeout in seconds for each AI API call (default 90, or AI_CALL_TIMEOUT env)")
	relayCmd.Flags().BoolVar(&relayRefreshBotID, "refresh-bot-id", false, "Generate a new bot ID (invalidates existing bot page links)")
	relayCmd.Flags().BoolVar(&relayInsecure, "insecure", false, "Skip TLS certificate verification (use when server has self-signed cert)")
	relayCmd.Flags().StringVar(&relayE2EKeyFile, "e2e-key-file", "", "Path to E2E PEM key file (default: ~/.lingti-e2e.pem)")

	// WeCom credentials for cloud relay
	relayCmd.Flags().StringVar(&relayWeComCorpID, "wecom-corp-id", "", "WeCom Corp ID (or WECOM_CORP_ID env)")
	relayCmd.Flags().StringVar(&relayWeComAgentID, "wecom-agent-id", "", "WeCom Agent ID (or WECOM_AGENT_ID env)")
	relayCmd.Flags().StringVar(&relayWeComSecret, "wecom-secret", "", "WeCom Secret (or WECOM_SECRET env)")
	relayCmd.Flags().StringVar(&relayWeComToken, "wecom-token", "", "WeCom Callback Token (or WECOM_TOKEN env)")
	relayCmd.Flags().StringVar(&relayWeComAESKey, "wecom-aes-key", "", "WeCom Encoding AES Key (or WECOM_AES_KEY env)")

	// WeChat OA credentials
	relayCmd.Flags().StringVar(&relayWeChatAppID, "wechat-app-id", "", "WeChat OA App ID (or WECHAT_APP_ID env)")
	relayCmd.Flags().StringVar(&relayWeChatAppSecret, "wechat-app-secret", "", "WeChat OA App Secret (or WECHAT_APP_SECRET env)")
}

func runRelay(cmd *cobra.Command, args []string) {
	// Get values from flags or environment
	if relayUserID == "" {
		relayUserID = os.Getenv("RELAY_USER_ID")
	}
	if relayPlatform == "" {
		relayPlatform = os.Getenv("RELAY_PLATFORM")
	}
	if relayServerURL == "" {
		relayServerURL = os.Getenv("RELAY_SERVER_URL")
	}
	if relayWebhookURL == "" {
		relayWebhookURL = os.Getenv("RELAY_WEBHOOK_URL")
	}

	// --host sets both server and webhook URLs (explicit --server/--webhook override it)
	if relayHost != "" {
		// Derive WebSocket scheme from HTTP scheme
		wsBase := strings.TrimRight(relayHost, "/")
		httpBase := wsBase
		if strings.HasPrefix(wsBase, "https://") {
			wsBase = "wss://" + wsBase[len("https://"):]
		} else if strings.HasPrefix(wsBase, "http://") {
			wsBase = "ws://" + wsBase[len("http://"):]
		} else {
			// bare host, assume secure
			wsBase = "wss://" + wsBase
			httpBase = "https://" + httpBase
		}
		if relayServerURL == "" {
			relayServerURL = wsBase + "/ws"
		}
		if relayWebhookURL == "" {
			relayWebhookURL = httpBase + "/webhook"
		}
	}
	if relayAIProvider == "" {
		relayAIProvider = os.Getenv("AI_PROVIDER")
	}
	if relayAPIKey == "" {
		relayAPIKey = os.Getenv("AI_API_KEY")
		// Fallback: ANTHROPIC_OAUTH_TOKEN (setup token) > ANTHROPIC_API_KEY
		if relayAPIKey == "" {
			relayAPIKey = os.Getenv("ANTHROPIC_OAUTH_TOKEN")
		}
		if relayAPIKey == "" {
			relayAPIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}
	if relayBaseURL == "" {
		relayBaseURL = os.Getenv("AI_BASE_URL")
		if relayBaseURL == "" {
			relayBaseURL = os.Getenv("ANTHROPIC_BASE_URL")
		}
	}
	if relayModel == "" {
		relayModel = os.Getenv("AI_MODEL")
		if relayModel == "" {
			relayModel = os.Getenv("ANTHROPIC_MODEL")
		}
	}
	if relayMaxRounds == 0 {
		if v := os.Getenv("AI_MAX_ROUNDS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				relayMaxRounds = n
			}
		}
	}
	if relayCallTimeout == 0 {
		if v := os.Getenv("AI_CALL_TIMEOUT"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				relayCallTimeout = n
			}
		}
	}

	// Get WeCom credentials from flags or environment
	if relayWeComCorpID == "" {
		relayWeComCorpID = os.Getenv("WECOM_CORP_ID")
	}
	if relayWeComAgentID == "" {
		relayWeComAgentID = os.Getenv("WECOM_AGENT_ID")
	}
	if relayWeComSecret == "" {
		relayWeComSecret = os.Getenv("WECOM_SECRET")
	}
	if relayWeComToken == "" {
		relayWeComToken = os.Getenv("WECOM_TOKEN")
	}
	if relayWeComAESKey == "" {
		relayWeComAESKey = os.Getenv("WECOM_AES_KEY")
	}

	// Get WeChat OA credentials from flags or environment
	if relayWeChatAppID == "" {
		relayWeChatAppID = os.Getenv("WECHAT_APP_ID")
	}
	if relayWeChatAppSecret == "" {
		relayWeChatAppSecret = os.Getenv("WECHAT_APP_SECRET")
	}

	// Fallback to saved config file
	savedCfg, cfgErr := config.Load()
	if cfgErr == nil {
		// When agents are configured and no provider was explicitly set via CLI/env,
		// prefer the default agent's AI config over relay.provider / ai.provider.
		if relayAIProvider == "" && len(savedCfg.Agents) > 0 {
			defaultID := savedCfg.DefaultAgentID()
			if entry, ok := savedCfg.FindAgent(defaultID); ok {
				ai := savedCfg.ResolveAgentAI(entry)
				if ai.Provider != "" || ai.APIKey != "" {
					if ai.Provider != "" {
						relayAIProvider = ai.Provider
					}
					if relayAPIKey == "" {
						relayAPIKey = ai.APIKey
					}
					if relayBaseURL == "" {
						relayBaseURL = ai.BaseURL
					}
					if relayModel == "" {
						relayModel = ai.Model
					}
				}
			}
		}

		// Resolve named provider: CLI --provider > env > relay.provider > ai.provider
		providerRef := relayAIProvider
		resolved, found := savedCfg.ResolveProvider(providerRef)
		if found {
			if relayAIProvider == "" {
				relayAIProvider = resolved.Provider
			}
			if relayAPIKey == "" {
				relayAPIKey = resolved.APIKey
			}
			if relayBaseURL == "" {
				relayBaseURL = resolved.BaseURL
			}
			if relayModel == "" {
				relayModel = resolved.Model
			}
		}
		if relayMaxRounds == 0 && savedCfg.AI.MaxRounds > 0 {
			relayMaxRounds = savedCfg.AI.MaxRounds
		}
		if relayCallTimeout == 0 && savedCfg.AI.CallTimeoutSecs > 0 {
			relayCallTimeout = savedCfg.AI.CallTimeoutSecs
		}
		// Read relay-specific config (platform, user-id) from saved config
		if relayPlatform == "" && savedCfg.Relay.Platform != "" {
			relayPlatform = savedCfg.Relay.Platform
		}
		if relayUserID == "" && savedCfg.Relay.UserID != "" {
			relayUserID = savedCfg.Relay.UserID
		}
		if relayPlatform == "" && savedCfg.Mode == "relay" {
			// Infer platform from saved platform credentials
			if savedCfg.Platforms.WeCom.CorpID != "" {
				relayPlatform = "wecom"
			}
		}
		if relayWeComCorpID == "" {
			relayWeComCorpID = savedCfg.Platforms.WeCom.CorpID
		}
		if relayWeComAgentID == "" {
			relayWeComAgentID = savedCfg.Platforms.WeCom.AgentID
		}
		if relayWeComSecret == "" {
			relayWeComSecret = savedCfg.Platforms.WeCom.Secret
		}
		if relayWeComToken == "" {
			relayWeComToken = savedCfg.Platforms.WeCom.Token
		}
		if relayWeComAESKey == "" {
			relayWeComAESKey = savedCfg.Platforms.WeCom.AESKey
		}
		if relayWeChatAppID == "" {
			relayWeChatAppID = savedCfg.Platforms.WeChat.AppID
		}
		if relayWeChatAppSecret == "" {
			relayWeChatAppSecret = savedCfg.Platforms.WeChat.AppSecret
		}

		// Generate or refresh bot ID
		if relayRefreshBotID || savedCfg.BotID == "" {
			savedCfg.BotID = uuid.New().String()
			if err := savedCfg.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save bot ID: %v\n", err)
			}
		}
		if relayRefreshBotID {
			botBase := "https://bot.lingti.com"
			if relayHost != "" {
				botBase = strings.TrimRight(relayHost, "/")
				if !strings.HasPrefix(botBase, "http") {
					botBase = "https://" + botBase
				}
			}
			fmt.Printf("[Relay] Bot ID refreshed. New bot page: %s/bots/%s\n", botBase, savedCfg.BotID)
			return
		}
	}

	// Validate required parameters
	// Platform is optional when a bot_id is configured (bot-page-only mode)
	hasBotID := cfgErr == nil && savedCfg.BotID != ""
	if relayPlatform == "" && !hasBotID {
		fmt.Fprintln(os.Stderr, "Error: --platform is required (feishu, slack, wechat, or wecom)")
		os.Exit(1)
	}
	if relayPlatform != "" && relayPlatform != "feishu" && relayPlatform != "slack" && relayPlatform != "wechat" && relayPlatform != "wecom" {
		fmt.Fprintln(os.Stderr, "Error: --platform must be 'feishu', 'slack', 'wechat', or 'wecom'")
		os.Exit(1)
	}
	if relayAPIKey == "" && strings.ToLower(relayAIProvider) != "ollama" {
		fmt.Fprintln(os.Stderr, "Error: AI API key is required (--api-key or AI_API_KEY env)")
		os.Exit(1)
	}

	// For WeCom, user-id is optional - auto-generate from corp_id
	// For bot-page-only mode (no platform), user-id is optional - use bot ID as fallback
	// For other platforms, user-id is required
	if relayUserID == "" {
		if relayPlatform == "wecom" && relayWeComCorpID != "" {
			relayUserID = "wecom-" + relayWeComCorpID
		} else if relayPlatform == "" && hasBotID {
			relayUserID = savedCfg.BotID
		} else if relayPlatform != "wecom" && relayPlatform != "" {
			fmt.Fprintln(os.Stderr, "Error: --user-id is required (get it from /whoami)")
			os.Exit(1)
		}
	}

	// Validate WeCom credentials when platform is wecom
	if relayPlatform == "wecom" {
		missing := []string{}
		if relayWeComCorpID == "" {
			missing = append(missing, "--wecom-corp-id")
		}
		if relayWeComAgentID == "" {
			missing = append(missing, "--wecom-agent-id")
		}
		if relayWeComSecret == "" {
			missing = append(missing, "--wecom-secret")
		}
		if relayWeComToken == "" {
			missing = append(missing, "--wecom-token")
		}
		if relayWeComAESKey == "" {
			missing = append(missing, "--wecom-aes-key")
		}
		if len(missing) > 0 {
			fmt.Fprintf(os.Stderr, "Error: WeCom credentials required for cloud relay: %v\n", missing)
			fmt.Fprintln(os.Stderr, "Configure callback URL in WeCom: https://bot.lingti.com/wecom")
			os.Exit(1)
		}
	}

	// Load custom instructions if specified
	var customInstructions string
	if relayInstructions != "" {
		data, err := os.ReadFile(relayInstructions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading instructions file: %v\n", err)
			os.Exit(1)
		}
		customInstructions = string(data)
		log.Printf("Loaded custom instructions from %s (%d bytes)", relayInstructions, len(data))
	}

	// Load MCP server configs from yaml config file
	var mcpServers []mcpclient.ServerConfig
	if cfgErr == nil {
		for _, s := range savedCfg.AI.MCPServers {
			mcpServers = append(mcpServers, mcpclient.ServerConfig{
				Name:    s.Name,
				Command: s.Command,
				Args:    s.Args,
				Env:     s.Env,
				URL:     s.URL,
			})
		}
	}

	// Create the AI agent
	agentCfg := agent.Config{
		Provider:           relayAIProvider,
		APIKey:             relayAPIKey,
		BaseURL:            relayBaseURL,
		Model:              relayModel,
		CustomInstructions: customInstructions,
		AllowedPaths:       loadAllowedPaths(),
		DisableFileTools:   loadDisableFileTools(),
		MaxToolRounds:      relayMaxRounds,
		CallTimeoutSecs:    relayCallTimeout,
		MCPServers:         mcpServers,
	}
	aiAgent, err := agent.New(agentCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// Resolve provider and model names for display
	providerName := relayAIProvider
	if providerName == "" {
		providerName = "claude"
	}
	modelName := relayModel
	if modelName == "" {
		modelName = "(default)"
	}

	// Create agent pool for per-platform/channel model overrides
	pool := agent.NewAgentPool(aiAgent, agentCfg, savedCfg)

	// Create the router with the pool as message handler
	r := router.New(pool.HandleMessage)

	// Initialize cron scheduler
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}
	cronPath := filepath.Join(homeDir, ".lingti.db")
	cronStore, err := cronpkg.NewStore(cronPath)
	if err != nil {
		log.Fatalf("Failed to open cron store: %v", err)
	}
	cronNotifier := agent.NewRouterCronNotifier(r)
	cronScheduler := cronpkg.NewScheduler(cronStore, aiAgent, aiAgent, cronNotifier)
	aiAgent.SetCronScheduler(cronScheduler)
	if err := cronScheduler.Start(); err != nil {
		log.Printf("Warning: Failed to start cron scheduler: %v", err)
	}

	// Create and register relay platform
	var relayBotID string
	if cfgErr == nil {
		relayBotID = savedCfg.BotID
	}
	relayPlatformInstance, err := relay.New(relay.Config{
		UserID:       relayUserID,
		Platform:     relayPlatform,
		ServerURL:    relayServerURL,
		WebhookURL:   relayWebhookURL,
		AIProvider:   providerName,
		AIModel:      modelName,
		BotID:        relayBotID,
		InsecureTLS:  relayInsecure,
		E2EKeyFile:   relayE2EKeyFile,
		WeComCorpID:     relayWeComCorpID,
		WeComAgentID:    relayWeComAgentID,
		WeComSecret:     relayWeComSecret,
		WeComToken:      relayWeComToken,
		WeComAESKey:     relayWeComAESKey,
		WeChatAppID:     relayWeChatAppID,
		WeChatAppSecret: relayWeChatAppSecret,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating relay platform: %v\n", err)
		os.Exit(1)
	}
	r.Register(relayPlatformInstance)

	// Start the router
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := r.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting relay: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Relay connected. User: %s, Platform: %s", relayUserID, relayPlatform)
	log.Printf("AI Provider: %s, Model: %s", providerName, modelName)
	if relayBotID != "" {
		botBase := "https://bot.lingti.com"
		if relayHost != "" {
			botBase = strings.TrimRight(relayHost, "/")
			if !strings.HasPrefix(botBase, "http") {
				botBase = "https://" + botBase
			}
		}
		fmt.Printf("[Relay] Your bot page: %s/bots/%s\n", botBase, relayBotID)
	}
	log.Println("Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cronScheduler.Stop()
	r.Stop()
}
