package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/ruilisi/lsbot/internal/agent"
	"github.com/ruilisi/lsbot/internal/browser"
	"github.com/ruilisi/lsbot/internal/config"
	cronpkg "github.com/ruilisi/lsbot/internal/cron"
	"github.com/ruilisi/lsbot/internal/gateway"
	"github.com/ruilisi/lsbot/internal/logger"
	"github.com/ruilisi/lsbot/internal/platforms/dingtalk"
	"github.com/ruilisi/lsbot/internal/platforms/discord"
	"github.com/ruilisi/lsbot/internal/platforms/feishu"
	"github.com/ruilisi/lsbot/internal/platforms/googlechat"
	"github.com/ruilisi/lsbot/internal/platforms/imessage"
	"github.com/ruilisi/lsbot/internal/platforms/line"
	"github.com/ruilisi/lsbot/internal/platforms/matrix"
	"github.com/ruilisi/lsbot/internal/platforms/mattermost"
	"github.com/ruilisi/lsbot/internal/platforms/nextcloud"
	"github.com/ruilisi/lsbot/internal/platforms/nostr"
	signalplatform "github.com/ruilisi/lsbot/internal/platforms/signal"
	"github.com/ruilisi/lsbot/internal/platforms/slack"
	"github.com/ruilisi/lsbot/internal/platforms/zalo"
	"github.com/ruilisi/lsbot/internal/platforms/teams"
	"github.com/ruilisi/lsbot/internal/platforms/telegram"
	"github.com/ruilisi/lsbot/internal/platforms/twitch"
	"github.com/ruilisi/lsbot/internal/platforms/webapp"
	"github.com/ruilisi/lsbot/internal/platforms/wecom"
	"github.com/ruilisi/lsbot/internal/platforms/whatsapp"
	"github.com/ruilisi/lsbot/internal/router"
	"github.com/spf13/cobra"
)

var (
	gatewayAddr          string
	gatewayAuthToken     string
	gatewayAuthTokens    []string
	gatewayNoWS          bool
	gatewayRefreshBotID  bool
)

// Platform credential vars — used by gateway and the deprecated router alias.
var (
	slackBotToken        string
	slackAppToken        string
	feishuAppID          string
	feishuAppSecret      string
	telegramToken        string
	discordToken         string
	wecomCorpID          string
	wecomAgentID         string
	wecomSecret          string
	wecomToken           string
	wecomAESKey          string
	wecomPort            int
	dingtalkClientID     string
	dingtalkClientSecret string
	lineChannelSecret    string
	lineChannelToken     string
	teamsAppID           string
	teamsAppPassword     string
	teamsTenantID        string
	matrixHomeserverURL  string
	matrixUserID         string
	matrixAccessToken    string
	googlechatProjectID       string
	googlechatCredentialsFile string
	mattermostServerURL  string
	mattermostToken      string
	mattermostTeamName   string
	blueBubblesURL       string
	blueBubblesPassword  string
	signalAPIURL         string
	signalPhoneNumber    string
	twitchToken          string
	twitchChannel        string
	twitchBotName        string
	nostrPrivateKey      string
	nostrRelays          string
	zaloAppID            string
	zaloSecretKey        string
	zaloAccessToken      string
	nextcloudServerURL   string
	nextcloudUsername    string
	nextcloudPassword    string
	nextcloudRoomToken   string
	whatsappPhoneID      string
	whatsappAccessToken  string
	whatsappVerifyToken  string
	aiProvider           string
	aiAPIKey             string
	aiBaseURL            string
	aiModel              string
	aiInstructions       string
	aiCallTimeout        int
	browserDebugDir      string
	webappPort           int
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway [restart]",
	Short: "Start all platform bots and the WebSocket server",
	Long: `Start all configured platform bots and the WebSocket gateway server.

The gateway is the unified run command that:
  - Starts all platform bots configured in ~/.lingti.yaml (telegram, slack, discord, etc.)
  - Starts the WebSocket server on :18789 by default (use --no-ws to disable)
  - Optionally serves the web chat UI (use --webapp-port)

Subcommands:
  restart   Send SIGHUP to a running gateway to reload config

Environment variables:
  GATEWAY_ADDR        Address for WebSocket server (default: :18789)
  GATEWAY_AUTH_TOKEN  Single authentication token
  GATEWAY_AUTH_TOKENS Comma-separated authentication tokens`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && args[0] == "restart" {
			return gatewayRestart()
		}
		runGateway(cmd, args)
		return nil
	},
}

func gatewayRestart() error {
	home, _ := os.UserHomeDir()
	pidFile := filepath.Join(home, ".lingti", "gateway.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("could not read PID file %s: %w", pidFile, err)
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID in %s: %w", pidFile, err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send SIGHUP to process %d: %w", pid, err)
	}
	fmt.Printf("Sent SIGHUP to gateway (PID %d)\n", pid)
	return nil
}

func writePIDFile() {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".lingti")
	_ = os.MkdirAll(dir, 0755)
	pidFile := filepath.Join(dir, "gateway.pid")
	_ = os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func removePIDFile() {
	home, _ := os.UserHomeDir()
	_ = os.Remove(filepath.Join(home, ".lingti", "gateway.pid"))
}

func init() {
	rootCmd.AddCommand(gatewayCmd)

	gatewayCmd.Flags().StringVar(&gatewayAddr, "addr", "", "WebSocket gateway address (or GATEWAY_ADDR env, default: :18789)")
	gatewayCmd.Flags().StringVar(&gatewayAuthToken, "auth-token", "", "Single authentication token (or GATEWAY_AUTH_TOKEN env)")
	gatewayCmd.Flags().StringSliceVar(&gatewayAuthTokens, "auth-tokens", nil, "Multiple authentication tokens (or GATEWAY_AUTH_TOKENS env)")
	gatewayCmd.Flags().BoolVar(&gatewayNoWS, "no-ws", false, "Disable WebSocket server")
	gatewayCmd.Flags().BoolVar(&gatewayRefreshBotID, "refresh-bot-id", false, "Generate a new bot ID (invalidates existing share links)")

	gatewayCmd.Flags().StringVar(&aiProvider, "provider", "", "AI provider: claude, deepseek, kimi, qwen (or AI_PROVIDER env)")
	gatewayCmd.Flags().StringVar(&aiAPIKey, "api-key", "", "AI API Key (or AI_API_KEY env)")
	gatewayCmd.Flags().StringVar(&aiBaseURL, "base-url", "", "AI API base URL (or AI_BASE_URL env)")
	gatewayCmd.Flags().StringVar(&aiModel, "model", "", "Model name (or AI_MODEL env)")
	gatewayCmd.Flags().StringVar(&aiInstructions, "instructions", "", "Path to custom instructions file")
	gatewayCmd.Flags().IntVar(&aiCallTimeout, "call-timeout", 0, "Base timeout in seconds for each AI API call (or AI_CALL_TIMEOUT env)")
	gatewayCmd.Flags().StringVar(&browserDebugDir, "debug-dir", "", "Directory for debug screenshots (or BROWSER_DEBUG_DIR env)")
	gatewayCmd.Flags().IntVar(&webappPort, "webapp-port", 0, "Web chat UI port (0 = disabled, or WEBAPP_PORT env)")

	// Platform credential flags (overrides for CI/Docker — config file is preferred)
	gatewayCmd.Flags().StringVar(&slackBotToken, "slack-bot-token", "", "Slack Bot Token (or SLACK_BOT_TOKEN env)")
	gatewayCmd.Flags().StringVar(&slackAppToken, "slack-app-token", "", "Slack App Token (or SLACK_APP_TOKEN env)")
	gatewayCmd.Flags().StringVar(&feishuAppID, "feishu-app-id", "", "Feishu App ID (or FEISHU_APP_ID env)")
	gatewayCmd.Flags().StringVar(&feishuAppSecret, "feishu-app-secret", "", "Feishu App Secret (or FEISHU_APP_SECRET env)")
	gatewayCmd.Flags().StringVar(&telegramToken, "telegram-token", "", "Telegram Bot Token (or TELEGRAM_BOT_TOKEN env)")
	gatewayCmd.Flags().StringVar(&discordToken, "discord-token", "", "Discord Bot Token (or DISCORD_BOT_TOKEN env)")
	gatewayCmd.Flags().StringVar(&wecomCorpID, "wecom-corp-id", "", "WeCom Corp ID (or WECOM_CORP_ID env)")
	gatewayCmd.Flags().StringVar(&wecomAgentID, "wecom-agent-id", "", "WeCom Agent ID (or WECOM_AGENT_ID env)")
	gatewayCmd.Flags().StringVar(&wecomSecret, "wecom-secret", "", "WeCom Secret (or WECOM_SECRET env)")
	gatewayCmd.Flags().StringVar(&wecomToken, "wecom-token", "", "WeCom Callback Token (or WECOM_TOKEN env)")
	gatewayCmd.Flags().StringVar(&wecomAESKey, "wecom-aes-key", "", "WeCom EncodingAESKey (or WECOM_AES_KEY env)")
	gatewayCmd.Flags().IntVar(&wecomPort, "wecom-port", 0, "WeCom Callback Port (or WECOM_PORT env)")
	gatewayCmd.Flags().StringVar(&dingtalkClientID, "dingtalk-client-id", "", "DingTalk AppKey (or DINGTALK_CLIENT_ID env)")
	gatewayCmd.Flags().StringVar(&dingtalkClientSecret, "dingtalk-client-secret", "", "DingTalk AppSecret (or DINGTALK_CLIENT_SECRET env)")
	gatewayCmd.Flags().StringVar(&lineChannelSecret, "line-channel-secret", "", "LINE Channel Secret (or LINE_CHANNEL_SECRET env)")
	gatewayCmd.Flags().StringVar(&lineChannelToken, "line-channel-token", "", "LINE Channel Token (or LINE_CHANNEL_TOKEN env)")
	gatewayCmd.Flags().StringVar(&teamsAppID, "teams-app-id", "", "Teams App ID (or TEAMS_APP_ID env)")
	gatewayCmd.Flags().StringVar(&teamsAppPassword, "teams-app-password", "", "Teams App Password (or TEAMS_APP_PASSWORD env)")
	gatewayCmd.Flags().StringVar(&teamsTenantID, "teams-tenant-id", "", "Teams Tenant ID (or TEAMS_TENANT_ID env)")
	gatewayCmd.Flags().StringVar(&matrixHomeserverURL, "matrix-homeserver-url", "", "Matrix Homeserver URL (or MATRIX_HOMESERVER_URL env)")
	gatewayCmd.Flags().StringVar(&matrixUserID, "matrix-user-id", "", "Matrix User ID (or MATRIX_USER_ID env)")
	gatewayCmd.Flags().StringVar(&matrixAccessToken, "matrix-access-token", "", "Matrix Access Token (or MATRIX_ACCESS_TOKEN env)")
	gatewayCmd.Flags().StringVar(&googlechatProjectID, "googlechat-project-id", "", "Google Chat Project ID (or GOOGLE_CHAT_PROJECT_ID env)")
	gatewayCmd.Flags().StringVar(&googlechatCredentialsFile, "googlechat-credentials-file", "", "Google Chat Credentials File (or GOOGLE_CHAT_CREDENTIALS_FILE env)")
	gatewayCmd.Flags().StringVar(&mattermostServerURL, "mattermost-server-url", "", "Mattermost Server URL (or MATTERMOST_SERVER_URL env)")
	gatewayCmd.Flags().StringVar(&mattermostToken, "mattermost-token", "", "Mattermost Token (or MATTERMOST_TOKEN env)")
	gatewayCmd.Flags().StringVar(&mattermostTeamName, "mattermost-team-name", "", "Mattermost Team Name (or MATTERMOST_TEAM_NAME env)")
	gatewayCmd.Flags().StringVar(&blueBubblesURL, "bluebubbles-url", "", "BlueBubbles Server URL (or BLUEBUBBLES_URL env)")
	gatewayCmd.Flags().StringVar(&blueBubblesPassword, "bluebubbles-password", "", "BlueBubbles Password (or BLUEBUBBLES_PASSWORD env)")
	gatewayCmd.Flags().StringVar(&signalAPIURL, "signal-api-url", "", "Signal API URL (or SIGNAL_API_URL env)")
	gatewayCmd.Flags().StringVar(&signalPhoneNumber, "signal-phone-number", "", "Signal Phone Number (or SIGNAL_PHONE_NUMBER env)")
	gatewayCmd.Flags().StringVar(&twitchToken, "twitch-token", "", "Twitch OAuth Token (or TWITCH_TOKEN env)")
	gatewayCmd.Flags().StringVar(&twitchChannel, "twitch-channel", "", "Twitch Channel (or TWITCH_CHANNEL env)")
	gatewayCmd.Flags().StringVar(&twitchBotName, "twitch-bot-name", "", "Twitch Bot Name (or TWITCH_BOT_NAME env)")
	gatewayCmd.Flags().StringVar(&nostrPrivateKey, "nostr-private-key", "", "NOSTR Private Key (or NOSTR_PRIVATE_KEY env)")
	gatewayCmd.Flags().StringVar(&nostrRelays, "nostr-relays", "", "NOSTR Relay URLs (or NOSTR_RELAYS env)")
	gatewayCmd.Flags().StringVar(&zaloAppID, "zalo-app-id", "", "Zalo App ID (or ZALO_APP_ID env)")
	gatewayCmd.Flags().StringVar(&zaloSecretKey, "zalo-secret-key", "", "Zalo Secret Key (or ZALO_SECRET_KEY env)")
	gatewayCmd.Flags().StringVar(&zaloAccessToken, "zalo-access-token", "", "Zalo Access Token (or ZALO_ACCESS_TOKEN env)")
	gatewayCmd.Flags().StringVar(&nextcloudServerURL, "nextcloud-server-url", "", "Nextcloud Server URL (or NEXTCLOUD_SERVER_URL env)")
	gatewayCmd.Flags().StringVar(&nextcloudUsername, "nextcloud-username", "", "Nextcloud Username (or NEXTCLOUD_USERNAME env)")
	gatewayCmd.Flags().StringVar(&nextcloudPassword, "nextcloud-password", "", "Nextcloud Password (or NEXTCLOUD_PASSWORD env)")
	gatewayCmd.Flags().StringVar(&nextcloudRoomToken, "nextcloud-room-token", "", "Nextcloud Room Token (or NEXTCLOUD_ROOM_TOKEN env)")
	gatewayCmd.Flags().StringVar(&whatsappPhoneID, "whatsapp-phone-id", "", "WhatsApp Phone Number ID (or WHATSAPP_PHONE_NUMBER_ID env)")
	gatewayCmd.Flags().StringVar(&whatsappAccessToken, "whatsapp-access-token", "", "WhatsApp Access Token (or WHATSAPP_ACCESS_TOKEN env)")
	gatewayCmd.Flags().StringVar(&whatsappVerifyToken, "whatsapp-verify-token", "", "WhatsApp Verify Token (or WHATSAPP_VERIFY_TOKEN env)")
}

func runGateway(cmd *cobra.Command, args []string) {
	// --- Resolve env vars (3-tier: flags > env > config file) ---
	resolveRouterEnvVars()

	if gatewayAddr == "" {
		gatewayAddr = os.Getenv("GATEWAY_ADDR")
		if gatewayAddr == "" {
			gatewayAddr = ":18789"
		}
	}
	if gatewayAuthToken == "" {
		gatewayAuthToken = os.Getenv("GATEWAY_AUTH_TOKEN")
	}
	if len(gatewayAuthTokens) == 0 {
		if v := os.Getenv("GATEWAY_AUTH_TOKENS"); v != "" {
			for _, t := range strings.Split(v, ",") {
				if t = strings.TrimSpace(t); t != "" {
					gatewayAuthTokens = append(gatewayAuthTokens, t)
				}
			}
		}
	}

	// Load ~/.lingti.yaml and apply config-file fallbacks
	savedCfg, cfgErr := config.Load()
	if cfgErr == nil {
		applyRouterConfigFallbacks(savedCfg)
	}

	// Generate or refresh bot ID
	if cfgErr == nil {
		if gatewayRefreshBotID || savedCfg.BotID == "" {
			savedCfg.BotID = uuid.New().String()
			if err := savedCfg.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save bot ID: %v\n", err)
			}
		}
		fmt.Printf("[Gateway] Your bot page: https://lsbot.org/bots/%s\n", savedCfg.BotID)
	}

	debugEnabled := logger.IsDebug()
	if !debugEnabled {
		if os.Getenv("BROWSER_DEBUG") == "1" || os.Getenv("BROWSER_DEBUG") == "true" {
			debugEnabled = true
		}
	}
	if browserDebugDir == "" {
		browserDebugDir = os.Getenv("BROWSER_DEBUG_DIR")
	}

	if aiAPIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: AI_API_KEY is required")
		os.Exit(1)
	}

	var customInstructions string
	if aiInstructions != "" {
		data, err := os.ReadFile(aiInstructions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading instructions file: %v\n", err)
			os.Exit(1)
		}
		customInstructions = string(data)
		logger.Info("Loaded custom instructions from %s (%d bytes)", aiInstructions, len(data))
	}

	agentCfg := agent.Config{
		Provider:           aiProvider,
		APIKey:             aiAPIKey,
		BaseURL:            aiBaseURL,
		Model:              aiModel,
		AutoApprove:        IsAutoApprove(),
		CustomInstructions: customInstructions,
		AllowedPaths:       loadAllowedPaths(),
		DisableFileTools:   loadDisableFileTools(),
		CallTimeoutSecs:    aiCallTimeout,
	}
	aiAgent, err := agent.New(agentCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	if debugEnabled {
		if err := setupBrowserDebug(browserDebugDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to setup browser debug: %v\n", err)
		} else {
			logger.Info("Browser debug mode enabled, screenshots: %s", browserDebugDir)
		}
	}

	pool := agent.NewAgentPool(aiAgent, agentCfg, savedCfg)
	r := router.New(pool.HandleMessage)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}
	cronPath := filepath.Join(homeDir, ".lingti.db")
	cronStore, err := cronpkg.NewStore(cronPath)
	if err != nil {
		logger.Error("Failed to open cron store: %v", err)
		os.Exit(1)
	}
	cronNotifier := agent.NewRouterCronNotifier(r)
	cronScheduler := cronpkg.NewScheduler(cronStore, aiAgent, aiAgent, cronNotifier)
	aiAgent.SetCronScheduler(cronScheduler)
	if err := cronScheduler.Start(); err != nil {
		logger.Warn("Failed to start cron scheduler: %v", err)
	}

	registerPlatforms(r)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := r.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting router: %v\n", err)
		os.Exit(1)
	}

	providerName := aiProvider
	if providerName == "" {
		providerName = "claude"
	}
	modelName := aiModel
	if modelName == "" {
		modelName = "(default)"
	}
	logger.Info("[Gateway] Platform bots started. AI Provider: %s, Model: %s", providerName, modelName)

	// Write PID file for `gateway restart`
	writePIDFile()
	defer removePIDFile()

	// Start WebSocket server unless --no-ws
	var gw *gateway.Gateway
	if !gatewayNoWS {
		gw = gateway.New(gateway.Config{
			Addr:       gatewayAddr,
			AuthToken:  gatewayAuthToken,
			AuthTokens: gatewayAuthTokens,
		})

		gw.SetMessageHandler(func(ctx context.Context, clientID, sessionID, text string) (<-chan gateway.ResponsePayload, error) {
			respChan := make(chan gateway.ResponsePayload, 1)
			go func() {
				defer close(respChan)
				msg := router.Message{
					ID:        sessionID,
					Platform:  "gateway",
					ChannelID: clientID,
					UserID:    clientID,
					Username:  "gateway-user",
					Text:      text,
					Metadata:  map[string]string{"session_id": sessionID},
				}
				response, err := aiAgent.HandleMessage(ctx, msg)
				if err != nil {
					respChan <- gateway.ResponsePayload{
						Text:      fmt.Sprintf("Error: %v", err),
						SessionID: sessionID,
						Done:      true,
					}
					return
				}
				respChan <- gateway.ResponsePayload{
					Text:      response.Text,
					SessionID: sessionID,
					Done:      true,
				}
			}()
			return respChan, nil
		})

		go func() {
			if err := gw.Start(ctx); err != nil {
				logger.Error("Gateway WebSocket error: %v", err)
			}
		}()

		logger.Info("[Gateway] WebSocket server started on %s", gatewayAddr)
		total := len(gatewayAuthTokens)
		if gatewayAuthToken != "" {
			total++
		}
		if total > 0 {
			logger.Info("[Gateway] Authentication enabled (%d token(s))", total)
		}
	}

	logger.Info("Press Ctrl+C to stop.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigCh
		if sig == syscall.SIGHUP {
			logger.Info("[Gateway] Received SIGHUP, reloading config...")
			// TODO: implement config reload (re-register platforms)
			continue
		}
		break
	}

	logger.Info("Shutting down...")
	cronScheduler.Stop()
	r.Stop()
	if gw != nil {
		gw.Stop()
	}
}

// resolveRouterEnvVars fills platform credential vars from environment (flags take priority).
func resolveRouterEnvVars() {
	if slackBotToken == "" {
		slackBotToken = os.Getenv("SLACK_BOT_TOKEN")
	}
	if slackAppToken == "" {
		slackAppToken = os.Getenv("SLACK_APP_TOKEN")
	}
	if feishuAppID == "" {
		feishuAppID = os.Getenv("FEISHU_APP_ID")
	}
	if feishuAppSecret == "" {
		feishuAppSecret = os.Getenv("FEISHU_APP_SECRET")
	}
	if telegramToken == "" {
		telegramToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}
	if discordToken == "" {
		discordToken = os.Getenv("DISCORD_BOT_TOKEN")
	}
	if wecomCorpID == "" {
		wecomCorpID = os.Getenv("WECOM_CORP_ID")
	}
	if wecomAgentID == "" {
		wecomAgentID = os.Getenv("WECOM_AGENT_ID")
	}
	if wecomSecret == "" {
		wecomSecret = os.Getenv("WECOM_SECRET")
	}
	if wecomToken == "" {
		wecomToken = os.Getenv("WECOM_TOKEN")
	}
	if wecomAESKey == "" {
		wecomAESKey = os.Getenv("WECOM_AES_KEY")
	}
	if wecomPort == 0 {
		if port := os.Getenv("WECOM_PORT"); port != "" {
			fmt.Sscanf(port, "%d", &wecomPort)
		}
	}
	if dingtalkClientID == "" {
		dingtalkClientID = os.Getenv("DINGTALK_CLIENT_ID")
	}
	if dingtalkClientSecret == "" {
		dingtalkClientSecret = os.Getenv("DINGTALK_CLIENT_SECRET")
	}
	if nextcloudServerURL == "" {
		nextcloudServerURL = os.Getenv("NEXTCLOUD_SERVER_URL")
	}
	if nextcloudUsername == "" {
		nextcloudUsername = os.Getenv("NEXTCLOUD_USERNAME")
	}
	if nextcloudPassword == "" {
		nextcloudPassword = os.Getenv("NEXTCLOUD_PASSWORD")
	}
	if nextcloudRoomToken == "" {
		nextcloudRoomToken = os.Getenv("NEXTCLOUD_ROOM_TOKEN")
	}
	if zaloAppID == "" {
		zaloAppID = os.Getenv("ZALO_APP_ID")
	}
	if zaloSecretKey == "" {
		zaloSecretKey = os.Getenv("ZALO_SECRET_KEY")
	}
	if zaloAccessToken == "" {
		zaloAccessToken = os.Getenv("ZALO_ACCESS_TOKEN")
	}
	if nostrPrivateKey == "" {
		nostrPrivateKey = os.Getenv("NOSTR_PRIVATE_KEY")
	}
	if nostrRelays == "" {
		nostrRelays = os.Getenv("NOSTR_RELAYS")
	}
	if twitchToken == "" {
		twitchToken = os.Getenv("TWITCH_TOKEN")
	}
	if twitchChannel == "" {
		twitchChannel = os.Getenv("TWITCH_CHANNEL")
	}
	if twitchBotName == "" {
		twitchBotName = os.Getenv("TWITCH_BOT_NAME")
	}
	if signalAPIURL == "" {
		signalAPIURL = os.Getenv("SIGNAL_API_URL")
	}
	if signalPhoneNumber == "" {
		signalPhoneNumber = os.Getenv("SIGNAL_PHONE_NUMBER")
	}
	if blueBubblesURL == "" {
		blueBubblesURL = os.Getenv("BLUEBUBBLES_URL")
	}
	if blueBubblesPassword == "" {
		blueBubblesPassword = os.Getenv("BLUEBUBBLES_PASSWORD")
	}
	if mattermostServerURL == "" {
		mattermostServerURL = os.Getenv("MATTERMOST_SERVER_URL")
	}
	if mattermostToken == "" {
		mattermostToken = os.Getenv("MATTERMOST_TOKEN")
	}
	if mattermostTeamName == "" {
		mattermostTeamName = os.Getenv("MATTERMOST_TEAM_NAME")
	}
	if googlechatProjectID == "" {
		googlechatProjectID = os.Getenv("GOOGLE_CHAT_PROJECT_ID")
	}
	if googlechatCredentialsFile == "" {
		googlechatCredentialsFile = os.Getenv("GOOGLE_CHAT_CREDENTIALS_FILE")
	}
	if matrixHomeserverURL == "" {
		matrixHomeserverURL = os.Getenv("MATRIX_HOMESERVER_URL")
	}
	if matrixUserID == "" {
		matrixUserID = os.Getenv("MATRIX_USER_ID")
	}
	if matrixAccessToken == "" {
		matrixAccessToken = os.Getenv("MATRIX_ACCESS_TOKEN")
	}
	if teamsAppID == "" {
		teamsAppID = os.Getenv("TEAMS_APP_ID")
	}
	if teamsAppPassword == "" {
		teamsAppPassword = os.Getenv("TEAMS_APP_PASSWORD")
	}
	if teamsTenantID == "" {
		teamsTenantID = os.Getenv("TEAMS_TENANT_ID")
	}
	if lineChannelSecret == "" {
		lineChannelSecret = os.Getenv("LINE_CHANNEL_SECRET")
	}
	if lineChannelToken == "" {
		lineChannelToken = os.Getenv("LINE_CHANNEL_TOKEN")
	}
	if whatsappPhoneID == "" {
		whatsappPhoneID = os.Getenv("WHATSAPP_PHONE_NUMBER_ID")
	}
	if whatsappAccessToken == "" {
		whatsappAccessToken = os.Getenv("WHATSAPP_ACCESS_TOKEN")
	}
	if whatsappVerifyToken == "" {
		whatsappVerifyToken = os.Getenv("WHATSAPP_VERIFY_TOKEN")
	}
	if webappPort == 0 {
		if port := os.Getenv("WEBAPP_PORT"); port != "" {
			fmt.Sscanf(port, "%d", &webappPort)
		}
	}
	if aiProvider == "" {
		aiProvider = os.Getenv("AI_PROVIDER")
	}
	if aiAPIKey == "" {
		aiAPIKey = os.Getenv("AI_API_KEY")
		if aiAPIKey == "" {
			aiAPIKey = os.Getenv("ANTHROPIC_OAUTH_TOKEN")
		}
		if aiAPIKey == "" {
			aiAPIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}
	if aiBaseURL == "" {
		aiBaseURL = os.Getenv("AI_BASE_URL")
		if aiBaseURL == "" {
			aiBaseURL = os.Getenv("ANTHROPIC_BASE_URL")
		}
	}
	if aiModel == "" {
		aiModel = os.Getenv("AI_MODEL")
		if aiModel == "" {
			aiModel = os.Getenv("ANTHROPIC_MODEL")
		}
	}
	if aiCallTimeout == 0 {
		if v := os.Getenv("AI_CALL_TIMEOUT"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				aiCallTimeout = n
			}
		}
	}
}

// applyRouterConfigFallbacks fills remaining empty vars from the config file.
func applyRouterConfigFallbacks(savedCfg *config.Config) {
	// When agents are configured and no provider was explicitly set via CLI/env,
	// prefer the default agent's AI config over relay.provider / ai.provider.
	if aiProvider == "" && len(savedCfg.Agents) > 0 {
		defaultID := savedCfg.DefaultAgentID()
		if entry, ok := savedCfg.FindAgent(defaultID); ok {
			ai := savedCfg.ResolveAgentAI(entry)
			if ai.Provider != "" || ai.APIKey != "" {
				if ai.Provider != "" {
					aiProvider = ai.Provider
				}
				if aiAPIKey == "" {
					aiAPIKey = ai.APIKey
				}
				if aiBaseURL == "" {
					aiBaseURL = ai.BaseURL
				}
				if aiModel == "" {
					aiModel = ai.Model
				}
				if aiCallTimeout == 0 && savedCfg.AI.CallTimeoutSecs > 0 {
					aiCallTimeout = savedCfg.AI.CallTimeoutSecs
				}
				return
			}
		}
	}

	resolved, found := savedCfg.ResolveProvider(aiProvider)
	if found {
		if aiProvider == "" {
			aiProvider = resolved.Provider
		}
		if aiAPIKey == "" {
			aiAPIKey = resolved.APIKey
		}
		if aiBaseURL == "" {
			aiBaseURL = resolved.BaseURL
		}
		if aiModel == "" {
			aiModel = resolved.Model
		}
	}
	if aiCallTimeout == 0 && savedCfg.AI.CallTimeoutSecs > 0 {
		aiCallTimeout = savedCfg.AI.CallTimeoutSecs
	}

	p := savedCfg.Platforms
	if slackBotToken == "" {
		slackBotToken = p.Slack.BotToken
	}
	if slackAppToken == "" {
		slackAppToken = p.Slack.AppToken
	}
	if feishuAppID == "" {
		feishuAppID = p.Feishu.AppID
	}
	if feishuAppSecret == "" {
		feishuAppSecret = p.Feishu.AppSecret
	}
	if telegramToken == "" {
		telegramToken = p.Telegram.Token
	}
	if discordToken == "" {
		discordToken = p.Discord.Token
	}
	if wecomCorpID == "" {
		wecomCorpID = p.WeCom.CorpID
	}
	if wecomAgentID == "" {
		wecomAgentID = p.WeCom.AgentID
	}
	if wecomSecret == "" {
		wecomSecret = p.WeCom.Secret
	}
	if wecomToken == "" {
		wecomToken = p.WeCom.Token
	}
	if wecomAESKey == "" {
		wecomAESKey = p.WeCom.AESKey
	}
	if wecomPort == 0 && p.WeCom.CallbackPort != 0 {
		wecomPort = p.WeCom.CallbackPort
	}
	if dingtalkClientID == "" {
		dingtalkClientID = p.DingTalk.ClientID
	}
	if dingtalkClientSecret == "" {
		dingtalkClientSecret = p.DingTalk.ClientSecret
	}
	if nextcloudServerURL == "" {
		nextcloudServerURL = p.Nextcloud.ServerURL
	}
	if nextcloudUsername == "" {
		nextcloudUsername = p.Nextcloud.Username
	}
	if nextcloudPassword == "" {
		nextcloudPassword = p.Nextcloud.Password
	}
	if nextcloudRoomToken == "" {
		nextcloudRoomToken = p.Nextcloud.RoomToken
	}
	if zaloAppID == "" {
		zaloAppID = p.Zalo.AppID
	}
	if zaloSecretKey == "" {
		zaloSecretKey = p.Zalo.SecretKey
	}
	if zaloAccessToken == "" {
		zaloAccessToken = p.Zalo.AccessToken
	}
	if nostrPrivateKey == "" {
		nostrPrivateKey = p.NOSTR.PrivateKey
	}
	if nostrRelays == "" {
		nostrRelays = p.NOSTR.Relays
	}
	if twitchToken == "" {
		twitchToken = p.Twitch.Token
	}
	if twitchChannel == "" {
		twitchChannel = p.Twitch.Channel
	}
	if twitchBotName == "" {
		twitchBotName = p.Twitch.BotName
	}
	if signalAPIURL == "" {
		signalAPIURL = p.Signal.APIURL
	}
	if signalPhoneNumber == "" {
		signalPhoneNumber = p.Signal.PhoneNumber
	}
	if blueBubblesURL == "" {
		blueBubblesURL = p.IMessage.BlueBubblesURL
	}
	if blueBubblesPassword == "" {
		blueBubblesPassword = p.IMessage.BlueBubblesPassword
	}
	if mattermostServerURL == "" {
		mattermostServerURL = p.Mattermost.ServerURL
	}
	if mattermostToken == "" {
		mattermostToken = p.Mattermost.Token
	}
	if mattermostTeamName == "" {
		mattermostTeamName = p.Mattermost.TeamName
	}
	if googlechatProjectID == "" {
		googlechatProjectID = p.GoogleChat.ProjectID
	}
	if googlechatCredentialsFile == "" {
		googlechatCredentialsFile = p.GoogleChat.CredentialsFile
	}
	if matrixHomeserverURL == "" {
		matrixHomeserverURL = p.Matrix.HomeserverURL
	}
	if matrixUserID == "" {
		matrixUserID = p.Matrix.UserID
	}
	if matrixAccessToken == "" {
		matrixAccessToken = p.Matrix.AccessToken
	}
	if teamsAppID == "" {
		teamsAppID = p.Teams.AppID
	}
	if teamsAppPassword == "" {
		teamsAppPassword = p.Teams.AppPassword
	}
	if teamsTenantID == "" {
		teamsTenantID = p.Teams.TenantID
	}
	if lineChannelSecret == "" {
		lineChannelSecret = p.LINE.ChannelSecret
	}
	if lineChannelToken == "" {
		lineChannelToken = p.LINE.ChannelToken
	}
	if whatsappPhoneID == "" {
		whatsappPhoneID = p.WhatsApp.PhoneNumberID
	}
	if whatsappAccessToken == "" {
		whatsappAccessToken = p.WhatsApp.AccessToken
	}
	if whatsappVerifyToken == "" {
		whatsappVerifyToken = p.WhatsApp.VerifyToken
	}
	if webappPort == 0 && p.Webapp.Port != 0 {
		webappPort = p.Webapp.Port
	}
}

// registerPlatforms registers all configured platforms with the router.
func registerPlatforms(r *router.Router) {
	if slackBotToken != "" && slackAppToken != "" {
		p, err := slack.New(slack.Config{BotToken: slackBotToken, AppToken: slackAppToken})
		if err != nil {
			logger.Warn("Error creating Slack platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Slack tokens not provided, skipping Slack integration")
	}

	if feishuAppID != "" && feishuAppSecret != "" {
		p, err := feishu.New(feishu.Config{AppID: feishuAppID, AppSecret: feishuAppSecret})
		if err != nil {
			logger.Warn("Error creating Feishu platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Feishu tokens not provided, skipping Feishu integration")
	}

	if telegramToken != "" {
		p, err := telegram.New(telegram.Config{Token: telegramToken})
		if err != nil {
			logger.Warn("Error creating Telegram platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Telegram token not provided, skipping Telegram integration")
	}

	if discordToken != "" {
		p, err := discord.New(discord.Config{Token: discordToken})
		if err != nil {
			logger.Warn("Error creating Discord platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Discord token not provided, skipping Discord integration")
	}

	if wecomCorpID != "" && wecomAgentID != "" && wecomSecret != "" && wecomToken != "" && wecomAESKey != "" {
		p, err := wecom.New(wecom.Config{
			CorpID: wecomCorpID, AgentID: wecomAgentID, Secret: wecomSecret,
			Token: wecomToken, EncodingAESKey: wecomAESKey, CallbackPort: wecomPort,
		})
		if err != nil {
			logger.Warn("Error creating WeCom platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("WeCom tokens not provided, skipping WeCom integration")
	}

	if dingtalkClientID != "" && dingtalkClientSecret != "" {
		p, err := dingtalk.New(dingtalk.Config{ClientID: dingtalkClientID, ClientSecret: dingtalkClientSecret})
		if err != nil {
			logger.Warn("Error creating DingTalk platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("DingTalk tokens not provided, skipping DingTalk integration")
	}

	if nextcloudServerURL != "" && nextcloudUsername != "" && nextcloudPassword != "" && nextcloudRoomToken != "" {
		p, err := nextcloud.New(nextcloud.Config{
			ServerURL: nextcloudServerURL, Username: nextcloudUsername,
			Password: nextcloudPassword, RoomToken: nextcloudRoomToken,
		})
		if err != nil {
			logger.Warn("Error creating Nextcloud Talk platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Nextcloud Talk tokens not provided, skipping Nextcloud Talk integration")
	}

	if zaloAppID != "" && zaloAccessToken != "" {
		p, err := zalo.New(zalo.Config{AppID: zaloAppID, SecretKey: zaloSecretKey, AccessToken: zaloAccessToken})
		if err != nil {
			logger.Warn("Error creating Zalo platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Zalo tokens not provided, skipping Zalo integration")
	}

	if nostrPrivateKey != "" && nostrRelays != "" {
		p, err := nostr.New(nostr.Config{PrivateKey: nostrPrivateKey, Relays: nostrRelays})
		if err != nil {
			logger.Warn("Error creating NOSTR platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("NOSTR tokens not provided, skipping NOSTR integration")
	}

	if twitchToken != "" && twitchChannel != "" && twitchBotName != "" {
		p, err := twitch.New(twitch.Config{Token: twitchToken, Channel: twitchChannel, BotName: twitchBotName})
		if err != nil {
			logger.Warn("Error creating Twitch platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Twitch tokens not provided, skipping Twitch integration")
	}

	if signalAPIURL != "" && signalPhoneNumber != "" {
		p, err := signalplatform.New(signalplatform.Config{APIURL: signalAPIURL, PhoneNumber: signalPhoneNumber})
		if err != nil {
			logger.Warn("Error creating Signal platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Signal tokens not provided, skipping Signal integration")
	}

	if blueBubblesURL != "" && blueBubblesPassword != "" {
		p, err := imessage.New(imessage.Config{BlueBubblesURL: blueBubblesURL, BlueBubblesPassword: blueBubblesPassword})
		if err != nil {
			logger.Warn("Error creating iMessage platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("iMessage tokens not provided, skipping iMessage integration")
	}

	if mattermostServerURL != "" && mattermostToken != "" {
		p, err := mattermost.New(mattermost.Config{ServerURL: mattermostServerURL, Token: mattermostToken, TeamName: mattermostTeamName})
		if err != nil {
			logger.Warn("Error creating Mattermost platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Mattermost tokens not provided, skipping Mattermost integration")
	}

	if googlechatProjectID != "" {
		p, err := googlechat.New(googlechat.Config{ProjectID: googlechatProjectID, CredentialsFile: googlechatCredentialsFile})
		if err != nil {
			logger.Warn("Error creating Google Chat platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Google Chat tokens not provided, skipping Google Chat integration")
	}

	if matrixHomeserverURL != "" && matrixAccessToken != "" {
		p, err := matrix.New(matrix.Config{HomeserverURL: matrixHomeserverURL, UserID: matrixUserID, AccessToken: matrixAccessToken})
		if err != nil {
			logger.Warn("Error creating Matrix platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Matrix tokens not provided, skipping Matrix integration")
	}

	if teamsAppID != "" && teamsAppPassword != "" {
		p, err := teams.New(teams.Config{AppID: teamsAppID, AppPassword: teamsAppPassword, TenantID: teamsTenantID})
		if err != nil {
			logger.Warn("Error creating Teams platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("Teams tokens not provided, skipping Teams integration")
	}

	if lineChannelSecret != "" && lineChannelToken != "" {
		p, err := line.New(line.Config{ChannelSecret: lineChannelSecret, ChannelToken: lineChannelToken})
		if err != nil {
			logger.Warn("Error creating LINE platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("LINE tokens not provided, skipping LINE integration")
	}

	if whatsappPhoneID != "" && whatsappAccessToken != "" {
		p, err := whatsapp.New(whatsapp.Config{PhoneNumberID: whatsappPhoneID, AccessToken: whatsappAccessToken, VerifyToken: whatsappVerifyToken})
		if err != nil {
			logger.Warn("Error creating WhatsApp platform: %v, skipping", err)
			return
		}
		r.Register(p)
	} else {
		logger.Info("WhatsApp tokens not provided, skipping WhatsApp integration")
	}

	if webappPort > 0 {
		p, err := webapp.New(webapp.Config{Port: webappPort})
		if err != nil {
			logger.Warn("Error creating webapp platform: %v, skipping", err)
			return
		}
		r.Register(p)
	}
}

func setupBrowserDebug(debugDir string) error {
	if debugDir == "" {
		debugDir = filepath.Join(os.TempDir(), "lsbot")
	}
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return fmt.Errorf("failed to create debug directory: %w", err)
	}
	b := browser.Instance()
	b.EnableDebug(debugDir)
	return nil
}
