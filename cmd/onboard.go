package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/spf13/cobra"
)

var (
	onboardProvider    string
	onboardAPIKey      string
	onboardBaseURL     string
	onboardModel       string
	onboardPlatform    string
	onboardMode        string
	onboardReset       bool
	onboardRelayUserID string
	// WeCom
	onboardWeComCorpID  string
	onboardWeComAgentID string
	onboardWeComSecret  string
	onboardWeComToken   string
	onboardWeComAESKey  string
	// Slack
	onboardSlackBotToken string
	onboardSlackAppToken string
	// Telegram
	onboardTelegramToken string
	// Discord
	onboardDiscordToken string
	// Feishu
	onboardFeishuAppID     string
	onboardFeishuAppSecret string
	// DingTalk
	onboardDingTalkClientID     string
	onboardDingTalkClientSecret string
	// WhatsApp
	onboardWhatsAppPhoneID     string
	onboardWhatsAppAccessToken string
	onboardWhatsAppVerifyToken string
	// LINE
	onboardLINEChannelSecret string
	onboardLINEChannelToken  string
	// Teams
	onboardTeamsAppID       string
	onboardTeamsAppPassword string
	onboardTeamsTenantID    string
	// Matrix
	onboardMatrixHomeserverURL string
	onboardMatrixUserID        string
	onboardMatrixAccessToken   string
	// Google Chat
	onboardGoogleChatProjectID       string
	onboardGoogleChatCredentialsFile string
	// Mattermost
	onboardMattermostServerURL string
	onboardMattermostToken     string
	onboardMattermostTeamName  string
	// iMessage
	onboardBlueBubblesURL      string
	onboardBlueBubblesPassword string
	// Signal
	onboardSignalAPIURL      string
	onboardSignalPhoneNumber string
	// Twitch
	onboardTwitchToken   string
	onboardTwitchChannel string
	onboardTwitchBotName string
	// NOSTR
	onboardNOSTRPrivateKey string
	onboardNOSTRRelays     string
	// Zalo
	onboardZaloAppID       string
	onboardZaloSecretKey   string
	onboardZaloAccessToken string
	// Nextcloud
	onboardNextcloudServerURL string
	onboardNextcloudUsername  string
	onboardNextcloudPassword  string
	onboardNextcloudRoomToken string
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Interactive setup wizard for first-time configuration",
	Long: `Interactive setup wizard that saves AI provider and platform credentials
to a config file. After running onboard once, you can use 'relay' or 'router'
without passing any flags.

Usage:
  lsbot onboard              # Interactive wizard
  lsbot onboard --reset      # Clear config and start fresh
  lsbot onboard --provider deepseek --api-key sk-xxx  # Non-interactive

Config saved to:
  ~/.lsbot.yaml`,
	Run: runOnboard,
}

func init() {
	rootCmd.AddCommand(onboardCmd)

	onboardCmd.Flags().StringVar(&onboardProvider, "provider", "", "AI provider: deepseek, qwen, claude, kimi, minimax, doubao, zhipu, openai, gemini, yi, stepfun, baichuan, spark, siliconflow, grok")
	onboardCmd.Flags().StringVar(&onboardAPIKey, "api-key", "", "AI API key")
	onboardCmd.Flags().StringVar(&onboardBaseURL, "base-url", "", "Custom API base URL")
	onboardCmd.Flags().StringVar(&onboardModel, "model", "", "Model name")
	onboardCmd.Flags().StringVar(&onboardPlatform, "platform", "", "Platform: wecom, wechat, dingtalk, feishu, slack, telegram, discord")
	onboardCmd.Flags().StringVar(&onboardMode, "mode", "", "Connection mode: relay, router")
	onboardCmd.Flags().BoolVar(&onboardReset, "reset", false, "Clear existing config and start fresh")
	onboardCmd.Flags().StringVar(&onboardRelayUserID, "relay-user-id", "", "Relay user ID (from /whoami)")

	// WeCom
	onboardCmd.Flags().StringVar(&onboardWeComCorpID, "wecom-corp-id", "", "WeCom Corp ID")
	onboardCmd.Flags().StringVar(&onboardWeComAgentID, "wecom-agent-id", "", "WeCom Agent ID")
	onboardCmd.Flags().StringVar(&onboardWeComSecret, "wecom-secret", "", "WeCom Secret")
	onboardCmd.Flags().StringVar(&onboardWeComToken, "wecom-token", "", "WeCom Callback Token")
	onboardCmd.Flags().StringVar(&onboardWeComAESKey, "wecom-aes-key", "", "WeCom EncodingAESKey")
	// Slack
	onboardCmd.Flags().StringVar(&onboardSlackBotToken, "slack-bot-token", "", "Slack Bot Token")
	onboardCmd.Flags().StringVar(&onboardSlackAppToken, "slack-app-token", "", "Slack App Token")
	// Telegram
	onboardCmd.Flags().StringVar(&onboardTelegramToken, "telegram-token", "", "Telegram Bot Token")
	// Discord
	onboardCmd.Flags().StringVar(&onboardDiscordToken, "discord-token", "", "Discord Bot Token")
	// Feishu
	onboardCmd.Flags().StringVar(&onboardFeishuAppID, "feishu-app-id", "", "Feishu App ID")
	onboardCmd.Flags().StringVar(&onboardFeishuAppSecret, "feishu-app-secret", "", "Feishu App Secret")
	// DingTalk
	onboardCmd.Flags().StringVar(&onboardDingTalkClientID, "dingtalk-client-id", "", "DingTalk AppKey")
	onboardCmd.Flags().StringVar(&onboardDingTalkClientSecret, "dingtalk-client-secret", "", "DingTalk AppSecret")
	// WhatsApp
	onboardCmd.Flags().StringVar(&onboardWhatsAppPhoneID, "whatsapp-phone-id", "", "WhatsApp Phone Number ID")
	onboardCmd.Flags().StringVar(&onboardWhatsAppAccessToken, "whatsapp-access-token", "", "WhatsApp Access Token")
	onboardCmd.Flags().StringVar(&onboardWhatsAppVerifyToken, "whatsapp-verify-token", "", "WhatsApp Verify Token")
	// LINE
	onboardCmd.Flags().StringVar(&onboardLINEChannelSecret, "line-channel-secret", "", "LINE Channel Secret")
	onboardCmd.Flags().StringVar(&onboardLINEChannelToken, "line-channel-token", "", "LINE Channel Token")
	// Teams
	onboardCmd.Flags().StringVar(&onboardTeamsAppID, "teams-app-id", "", "Teams App ID")
	onboardCmd.Flags().StringVar(&onboardTeamsAppPassword, "teams-app-password", "", "Teams App Password")
	onboardCmd.Flags().StringVar(&onboardTeamsTenantID, "teams-tenant-id", "", "Teams Tenant ID")
	// Matrix
	onboardCmd.Flags().StringVar(&onboardMatrixHomeserverURL, "matrix-homeserver-url", "", "Matrix Homeserver URL")
	onboardCmd.Flags().StringVar(&onboardMatrixUserID, "matrix-user-id", "", "Matrix User ID")
	onboardCmd.Flags().StringVar(&onboardMatrixAccessToken, "matrix-access-token", "", "Matrix Access Token")
	// Google Chat
	onboardCmd.Flags().StringVar(&onboardGoogleChatProjectID, "googlechat-project-id", "", "Google Chat Project ID")
	onboardCmd.Flags().StringVar(&onboardGoogleChatCredentialsFile, "googlechat-credentials-file", "", "Google Chat Credentials File")
	// Mattermost
	onboardCmd.Flags().StringVar(&onboardMattermostServerURL, "mattermost-server-url", "", "Mattermost Server URL")
	onboardCmd.Flags().StringVar(&onboardMattermostToken, "mattermost-token", "", "Mattermost Token")
	onboardCmd.Flags().StringVar(&onboardMattermostTeamName, "mattermost-team-name", "", "Mattermost Team Name")
	// iMessage
	onboardCmd.Flags().StringVar(&onboardBlueBubblesURL, "bluebubbles-url", "", "BlueBubbles Server URL")
	onboardCmd.Flags().StringVar(&onboardBlueBubblesPassword, "bluebubbles-password", "", "BlueBubbles Password")
	// Signal
	onboardCmd.Flags().StringVar(&onboardSignalAPIURL, "signal-api-url", "", "Signal API URL")
	onboardCmd.Flags().StringVar(&onboardSignalPhoneNumber, "signal-phone-number", "", "Signal Phone Number")
	// Twitch
	onboardCmd.Flags().StringVar(&onboardTwitchToken, "twitch-token", "", "Twitch OAuth Token")
	onboardCmd.Flags().StringVar(&onboardTwitchChannel, "twitch-channel", "", "Twitch Channel")
	onboardCmd.Flags().StringVar(&onboardTwitchBotName, "twitch-bot-name", "", "Twitch Bot Name")
	// NOSTR
	onboardCmd.Flags().StringVar(&onboardNOSTRPrivateKey, "nostr-private-key", "", "NOSTR Private Key")
	onboardCmd.Flags().StringVar(&onboardNOSTRRelays, "nostr-relays", "", "NOSTR Relay URLs (comma-separated)")
	// Zalo
	onboardCmd.Flags().StringVar(&onboardZaloAppID, "zalo-app-id", "", "Zalo App ID")
	onboardCmd.Flags().StringVar(&onboardZaloSecretKey, "zalo-secret-key", "", "Zalo Secret Key")
	onboardCmd.Flags().StringVar(&onboardZaloAccessToken, "zalo-access-token", "", "Zalo Access Token")
	// Nextcloud
	onboardCmd.Flags().StringVar(&onboardNextcloudServerURL, "nextcloud-server-url", "", "Nextcloud Server URL")
	onboardCmd.Flags().StringVar(&onboardNextcloudUsername, "nextcloud-username", "", "Nextcloud Username")
	onboardCmd.Flags().StringVar(&onboardNextcloudPassword, "nextcloud-password", "", "Nextcloud Password")
	onboardCmd.Flags().StringVar(&onboardNextcloudRoomToken, "nextcloud-room-token", "", "Nextcloud Talk Room Token")
}

var scanner *bufio.Scanner

func initScanner() {
	scanner = bufio.NewScanner(os.Stdin)
}

func promptText(prompt string, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s (default: %s):\n  > ", prompt, defaultVal)
	} else {
		fmt.Printf("  %s:\n  > ", prompt)
	}
	scanner.Scan()
	val := strings.TrimSpace(scanner.Text())
	if val == "" {
		return defaultVal
	}
	return val
}

func promptYesNo(question string, defaultYes bool) bool {
	hint := "y/N"
	if defaultYes {
		hint = "Y/n"
	}
	fmt.Printf("  %s [%s]: ", question, hint)
	scanner.Scan()
	val := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if val == "" {
		return defaultYes
	}
	return val == "y" || val == "yes"
}

func promptSelect(prompt string, options []string, defaultIdx int) int {
	fmt.Printf("\n  %s\n", prompt)
	for i, opt := range options {
		fmt.Printf("    %d. %s\n", i+1, opt)
	}
	fmt.Printf("  Choice [%d]: ", defaultIdx+1)
	scanner.Scan()
	val := strings.TrimSpace(scanner.Text())
	if val == "" {
		return defaultIdx
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 || n > len(options) {
		return defaultIdx
	}
	return n - 1
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func runOnboard(cmd *cobra.Command, args []string) {
	if onboardReset {
		path := config.ConfigPath()
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error removing config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Config cleared: %s\n", path)
	}

	cfg, _ := config.Load()

	if hasOnboardFlags(cmd) {
		applyOnboardFlags(cfg)
	} else {
		initScanner()
		runInteractiveWizard(cfg)
	}

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	printOnboardSummary(cfg)
}

func hasOnboardFlags(_ *cobra.Command) bool {
	return onboardProvider != "" || onboardAPIKey != "" || onboardPlatform != ""
}

func applyOnboardFlags(cfg *config.Config) {
	if onboardProvider != "" {
		// Write to named providers map (new format)
		entry := config.ProviderEntry{Provider: onboardProvider}
		if onboardAPIKey != "" {
			entry.APIKey = onboardAPIKey
		}
		if onboardBaseURL != "" {
			entry.BaseURL = onboardBaseURL
		}
		if onboardModel != "" {
			entry.Model = onboardModel
		}
		name := onboardProvider
		if cfg.Providers == nil {
			cfg.Providers = make(map[string]config.ProviderEntry)
		}
		cfg.Providers[name] = entry
		cfg.Relay.Provider = name
		// Also keep ai: block for backward compat
		cfg.AI.Provider = onboardProvider
		if onboardAPIKey != "" {
			cfg.AI.APIKey = onboardAPIKey
		}
		if onboardBaseURL != "" {
			cfg.AI.BaseURL = onboardBaseURL
		}
		if onboardModel != "" {
			cfg.AI.Model = onboardModel
		}
	} else {
		// No provider flag — just update individual ai: fields
		if onboardAPIKey != "" {
			cfg.AI.APIKey = onboardAPIKey
		}
		if onboardBaseURL != "" {
			cfg.AI.BaseURL = onboardBaseURL
		}
		if onboardModel != "" {
			cfg.AI.Model = onboardModel
		}
	}
	if onboardMode != "" {
		cfg.Mode = onboardMode
	}

	// Relay user ID
	if onboardRelayUserID != "" {
		cfg.Relay.UserID = onboardRelayUserID
	}

	// Platform credentials
	switch onboardPlatform {
	case "wecom":
		if onboardWeComCorpID != "" {
			cfg.Platforms.WeCom.CorpID = onboardWeComCorpID
		}
		if onboardWeComAgentID != "" {
			cfg.Platforms.WeCom.AgentID = onboardWeComAgentID
		}
		if onboardWeComSecret != "" {
			cfg.Platforms.WeCom.Secret = onboardWeComSecret
		}
		if onboardWeComToken != "" {
			cfg.Platforms.WeCom.Token = onboardWeComToken
		}
		if onboardWeComAESKey != "" {
			cfg.Platforms.WeCom.AESKey = onboardWeComAESKey
		}
	case "wechat":
		cfg.Relay.Platform = "wechat"
		cfg.Mode = "relay"
	case "slack":
		if onboardSlackBotToken != "" {
			cfg.Platforms.Slack.BotToken = onboardSlackBotToken
		}
		if onboardSlackAppToken != "" {
			cfg.Platforms.Slack.AppToken = onboardSlackAppToken
		}
	case "telegram":
		if onboardTelegramToken != "" {
			cfg.Platforms.Telegram.Token = onboardTelegramToken
		}
	case "discord":
		if onboardDiscordToken != "" {
			cfg.Platforms.Discord.Token = onboardDiscordToken
		}
	case "feishu":
		if onboardFeishuAppID != "" {
			cfg.Platforms.Feishu.AppID = onboardFeishuAppID
		}
		if onboardFeishuAppSecret != "" {
			cfg.Platforms.Feishu.AppSecret = onboardFeishuAppSecret
		}
	case "dingtalk":
		if onboardDingTalkClientID != "" {
			cfg.Platforms.DingTalk.ClientID = onboardDingTalkClientID
		}
		if onboardDingTalkClientSecret != "" {
			cfg.Platforms.DingTalk.ClientSecret = onboardDingTalkClientSecret
		}
	case "whatsapp":
		if onboardWhatsAppPhoneID != "" {
			cfg.Platforms.WhatsApp.PhoneNumberID = onboardWhatsAppPhoneID
		}
		if onboardWhatsAppAccessToken != "" {
			cfg.Platforms.WhatsApp.AccessToken = onboardWhatsAppAccessToken
		}
		if onboardWhatsAppVerifyToken != "" {
			cfg.Platforms.WhatsApp.VerifyToken = onboardWhatsAppVerifyToken
		}
	case "line":
		if onboardLINEChannelSecret != "" {
			cfg.Platforms.LINE.ChannelSecret = onboardLINEChannelSecret
		}
		if onboardLINEChannelToken != "" {
			cfg.Platforms.LINE.ChannelToken = onboardLINEChannelToken
		}
	case "teams":
		if onboardTeamsAppID != "" {
			cfg.Platforms.Teams.AppID = onboardTeamsAppID
		}
		if onboardTeamsAppPassword != "" {
			cfg.Platforms.Teams.AppPassword = onboardTeamsAppPassword
		}
		if onboardTeamsTenantID != "" {
			cfg.Platforms.Teams.TenantID = onboardTeamsTenantID
		}
	case "matrix":
		if onboardMatrixHomeserverURL != "" {
			cfg.Platforms.Matrix.HomeserverURL = onboardMatrixHomeserverURL
		}
		if onboardMatrixUserID != "" {
			cfg.Platforms.Matrix.UserID = onboardMatrixUserID
		}
		if onboardMatrixAccessToken != "" {
			cfg.Platforms.Matrix.AccessToken = onboardMatrixAccessToken
		}
	case "googlechat":
		if onboardGoogleChatProjectID != "" {
			cfg.Platforms.GoogleChat.ProjectID = onboardGoogleChatProjectID
		}
		if onboardGoogleChatCredentialsFile != "" {
			cfg.Platforms.GoogleChat.CredentialsFile = onboardGoogleChatCredentialsFile
		}
	case "mattermost":
		if onboardMattermostServerURL != "" {
			cfg.Platforms.Mattermost.ServerURL = onboardMattermostServerURL
		}
		if onboardMattermostToken != "" {
			cfg.Platforms.Mattermost.Token = onboardMattermostToken
		}
		if onboardMattermostTeamName != "" {
			cfg.Platforms.Mattermost.TeamName = onboardMattermostTeamName
		}
	case "imessage":
		if onboardBlueBubblesURL != "" {
			cfg.Platforms.IMessage.BlueBubblesURL = onboardBlueBubblesURL
		}
		if onboardBlueBubblesPassword != "" {
			cfg.Platforms.IMessage.BlueBubblesPassword = onboardBlueBubblesPassword
		}
	case "signal":
		if onboardSignalAPIURL != "" {
			cfg.Platforms.Signal.APIURL = onboardSignalAPIURL
		}
		if onboardSignalPhoneNumber != "" {
			cfg.Platforms.Signal.PhoneNumber = onboardSignalPhoneNumber
		}
	case "twitch":
		if onboardTwitchToken != "" {
			cfg.Platforms.Twitch.Token = onboardTwitchToken
		}
		if onboardTwitchChannel != "" {
			cfg.Platforms.Twitch.Channel = onboardTwitchChannel
		}
		if onboardTwitchBotName != "" {
			cfg.Platforms.Twitch.BotName = onboardTwitchBotName
		}
	case "nostr":
		if onboardNOSTRPrivateKey != "" {
			cfg.Platforms.NOSTR.PrivateKey = onboardNOSTRPrivateKey
		}
		if onboardNOSTRRelays != "" {
			cfg.Platforms.NOSTR.Relays = onboardNOSTRRelays
		}
	case "zalo":
		if onboardZaloAppID != "" {
			cfg.Platforms.Zalo.AppID = onboardZaloAppID
		}
		if onboardZaloSecretKey != "" {
			cfg.Platforms.Zalo.SecretKey = onboardZaloSecretKey
		}
		if onboardZaloAccessToken != "" {
			cfg.Platforms.Zalo.AccessToken = onboardZaloAccessToken
		}
	case "nextcloud":
		if onboardNextcloudServerURL != "" {
			cfg.Platforms.Nextcloud.ServerURL = onboardNextcloudServerURL
		}
		if onboardNextcloudUsername != "" {
			cfg.Platforms.Nextcloud.Username = onboardNextcloudUsername
		}
		if onboardNextcloudPassword != "" {
			cfg.Platforms.Nextcloud.Password = onboardNextcloudPassword
		}
		if onboardNextcloudRoomToken != "" {
			cfg.Platforms.Nextcloud.RoomToken = onboardNextcloudRoomToken
		}
	}
}

func runInteractiveWizard(cfg *config.Config) {
	fmt.Println()
	fmt.Println("  lsbot -- Interactive Setup")
	fmt.Println("  ───────────────────────────────────")

	// Show existing config if present
	if cfg.AI.Provider != "" || len(cfg.Providers) > 0 {
		displayProvider := cfg.AI.Provider
		displayKey := cfg.AI.APIKey
		if displayProvider == "" && len(cfg.Providers) > 0 {
			for name, e := range cfg.Providers {
				displayProvider = name
				displayKey = e.APIKey
				break
			}
		}
		fmt.Printf("\n  Existing config found: %s / %s\n", displayProvider, maskKey(displayKey))
		idx := promptSelect("What would you like to do?", []string{
			"Update existing config",
			"Start fresh",
			"Keep and exit",
		}, 0)
		if idx == 2 {
			return
		}
		if idx == 1 {
			*cfg = *config.DefaultConfig()
		}
	}

	stepAIProvider(cfg)
	stepPlatform(cfg)
	stepConnectionMode(cfg)
}

type providerInfo struct {
	name     string
	label    string
	keyURL   string
	defModel string
}

var providers = []providerInfo{
	{"deepseek", "deepseek     (recommended)", "https://platform.deepseek.com/api_keys", "deepseek-chat"},
	{"qwen", "qwen         (tongyi qianwen)", "https://bailian.console.aliyun.com/", "qwen-plus"},
	{"claude", "claude       (Anthropic)", "https://console.anthropic.com/", "claude-sonnet-4-20250514"},
	{"kimi", "kimi         (Moonshot)", "https://platform.moonshot.cn/", "kimi-k2.5"},
	{"minimax", "minimax      (MiniMax/海螺)", "https://platform.minimaxi.com/", "MiniMax-Text-01"},
	{"doubao", "doubao       (ByteDance/豆包)", "https://console.volcengine.com/ark", "doubao-pro-32k"},
	{"zhipu", "zhipu        (GLM/智谱)", "https://open.bigmodel.cn/", "glm-4-flash"},
	{"openai", "openai       (GPT)", "https://platform.openai.com/api-keys", "gpt-4o"},
	{"gemini", "gemini       (Google)", "https://aistudio.google.com/apikey", "gemini-2.0-flash"},
	{"yi", "yi           (Lingyiwanwu/零一)", "https://platform.lingyiwanwu.com/", "yi-large"},
	{"stepfun", "stepfun      (StepFun/阶跃)", "https://platform.stepfun.com/", "step-2-16k"},
	{"baichuan", "baichuan     (Baichuan/百川)", "https://platform.baichuan-ai.com/", "Baichuan4"},
	{"spark", "spark        (iFlytek/讯飞星火)", "https://console.xfyun.cn/", "generalv3.5"},
	{"siliconflow", "siliconflow  (aggregator/硅基流动)", "https://cloud.siliconflow.cn/", "Qwen/Qwen2.5-72B-Instruct"},
	{"grok", "grok         (xAI)", "https://console.x.ai/", "grok-2-latest"},
}

// detectClaudeOAuthToken tries to find an existing Claude OAuth token from env vars or macOS Keychain.
func detectClaudeOAuthToken() string {
	// 1. Check ANTHROPIC_OAUTH_TOKEN env var
	if tok := os.Getenv("ANTHROPIC_OAUTH_TOKEN"); tok != "" && strings.HasPrefix(tok, "sk-ant-oat") {
		return tok
	}

	// 2. macOS Keychain: Claude Code stores credentials under "Claude Code-credentials"
	if runtime.GOOS == "darwin" {
		if tok := readClaudeKeychain(); tok != "" {
			return tok
		}
	}

	// 3. Check ANTHROPIC_API_KEY if it looks like an OAuth token
	if tok := os.Getenv("ANTHROPIC_API_KEY"); tok != "" && strings.HasPrefix(tok, "sk-ant-oat") {
		return tok
	}

	return ""
}

// detectClaudeAPIKey tries to find an existing Anthropic API key from env vars.
func detectClaudeAPIKey() string {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" && strings.HasPrefix(key, "sk-ant-") && !strings.HasPrefix(key, "sk-ant-oat") {
		return key
	}
	return ""
}

// readClaudeKeychain reads the Claude Code OAuth token from macOS Keychain.
func readClaudeKeychain() string {
	out, err := exec.Command("security", "find-generic-password", "-s", "Claude Code-credentials", "-w").Output()
	if err != nil {
		return ""
	}

	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(out, &creds); err != nil {
		return ""
	}

	tok := creds.ClaudeAiOauth.AccessToken
	if strings.HasPrefix(tok, "sk-ant-oat") {
		return tok
	}
	return ""
}

func stepAIProvider(cfg *config.Config) {
	fmt.Println("\n  Step 1/3: AI Provider")

	options := make([]string, len(providers))
	for i, p := range providers {
		options[i] = p.label
	}

	defIdx := 0
	for i, p := range providers {
		if p.name == cfg.AI.Provider {
			defIdx = i
			break
		}
	}

	idx := promptSelect("Select AI provider:", options, defIdx)
	p := providers[idx]
	cfg.AI.Provider = p.name

	if p.name == "claude" {
		// Auto-detect existing tokens to suggest as defaults
		detectedOAuth := detectClaudeOAuthToken()
		detectedAPIKey := detectClaudeAPIKey()

		// If existing config or detected token is OAuth, default to Setup Token auth
		defAuth := 0
		if detectedOAuth != "" || strings.HasPrefix(cfg.AI.APIKey, "sk-ant-oat") {
			defAuth = 1
		}

		authIdx := promptSelect("Auth method:", []string{
			"API Key       (from console.anthropic.com)",
			"Setup Token   (from 'claude setup-token', requires Claude subscription)",
		}, defAuth)
		if authIdx == 0 {
			defKey := cfg.AI.APIKey
			if defKey == "" && detectedAPIKey != "" {
				defKey = detectedAPIKey
				fmt.Printf("\n  Detected existing API key: %s\n", maskKey(defKey))
			}
			fmt.Printf("\n  Claude API Key (%s)\n", p.keyURL)
			cfg.AI.APIKey = promptText("API Key", defKey)
		} else {
			// Pick best default: prefer detected (freshest), fall back to config
			defToken := detectedOAuth
			if defToken == "" {
				defToken = cfg.AI.APIKey
			}

			if defToken != "" {
				fmt.Printf("\n  Detected existing Claude token: %s\n", maskKey(defToken))
				fmt.Println("  Press Enter to use it, or paste a different token.")
			} else {
				fmt.Println("\n  Run 'claude setup-token' in another terminal, then paste the token here.")
				fmt.Println("  (Requires Claude Code CLI and an active Claude subscription)")
			}
			cfg.AI.APIKey = promptText("Setup Token (sk-ant-oat01-...)", defToken)
			if cfg.AI.APIKey != "" && !strings.HasPrefix(cfg.AI.APIKey, "sk-ant-oat") {
				fmt.Println("  Warning: expected token starting with sk-ant-oat01-")
			}
		}
	} else {
		displayName := strings.ToUpper(p.name[:1]) + p.name[1:]
		fmt.Printf("\n  %s API Key (%s)\n", displayName, p.keyURL)
		cfg.AI.APIKey = promptText("API Key", cfg.AI.APIKey)
	}

	model := promptText("Model", p.defModel)
	cfg.AI.Model = model

	// Also write to named providers map
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderEntry)
	}
	cfg.Providers[p.name] = config.ProviderEntry{
		Provider: cfg.AI.Provider,
		APIKey:   cfg.AI.APIKey,
		BaseURL:  cfg.AI.BaseURL,
		Model:    cfg.AI.Model,
	}
	cfg.Relay.Provider = p.name

	fmt.Printf("\n  > AI provider configured: %s / %s\n", cfg.AI.Provider, cfg.AI.Model)
}

type platformInfo struct {
	name  string
	label string
}

var platformOptions = []platformInfo{
	{"wecom", "wecom     (WeCom/企业微信)"},
	{"wechat", "wechat    (WeChat/微信, relay only)"},
	{"dingtalk", "dingtalk  (DingTalk/钉钉)"},
	{"feishu", "feishu    (Feishu/Lark/飞书)"},
	{"slack", "slack"},
	{"telegram", "telegram"},
	{"discord", "discord"},
	{"whatsapp", "whatsapp  (WhatsApp Business)"},
	{"line", "line      (LINE)"},
	{"teams", "teams     (Microsoft Teams)"},
	{"matrix", "matrix    (Matrix/Element)"},
	{"googlechat", "googlechat (Google Chat)"},
	{"mattermost", "mattermost (Mattermost)"},
	{"imessage", "imessage  (iMessage/BlueBubbles)"},
	{"signal", "signal    (Signal)"},
	{"twitch", "twitch    (Twitch)"},
	{"nostr", "nostr     (NOSTR)"},
	{"zalo", "zalo      (Zalo)"},
	{"nextcloud", "nextcloud (Nextcloud Talk)"},
	{"skip", "skip      (configure later)"},
}

func stepPlatform(cfg *config.Config) {
	fmt.Println("\n  Step 2/3: Platform")

	options := make([]string, len(platformOptions))
	for i, p := range platformOptions {
		options[i] = p.label
	}

	idx := promptSelect("Select messaging platform:", options, 0)
	platform := platformOptions[idx].name

	switch platform {
	case "wecom":
		stepWecom(cfg)
	case "wechat":
		stepWeChat(cfg)
	case "dingtalk":
		stepDingTalk(cfg)
	case "feishu":
		stepFeishu(cfg)
	case "slack":
		stepSlack(cfg)
	case "telegram":
		stepTelegram(cfg)
	case "discord":
		stepDiscord(cfg)
	case "whatsapp":
		stepWhatsApp(cfg)
	case "line":
		stepLINE(cfg)
	case "teams":
		stepTeams(cfg)
	case "matrix":
		stepMatrix(cfg)
	case "googlechat":
		stepGoogleChat(cfg)
	case "mattermost":
		stepMattermost(cfg)
	case "imessage":
		stepIMessage(cfg)
	case "signal":
		stepSignal(cfg)
	case "twitch":
		stepTwitch(cfg)
	case "nostr":
		stepNOSTR(cfg)
	case "zalo":
		stepZalo(cfg)
	case "nextcloud":
		stepNextcloud(cfg)
	case "skip":
		fmt.Println("\n  > Platform configuration skipped")
	}
}

func stepWecom(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.WeCom.CorpID = promptText("WeCom Corp ID", cfg.Platforms.WeCom.CorpID)
	cfg.Platforms.WeCom.AgentID = promptText("WeCom Agent ID", cfg.Platforms.WeCom.AgentID)
	cfg.Platforms.WeCom.Secret = promptText("WeCom Secret", cfg.Platforms.WeCom.Secret)
	cfg.Platforms.WeCom.Token = promptText("WeCom Token", cfg.Platforms.WeCom.Token)
	cfg.Platforms.WeCom.AESKey = promptText("WeCom AES Key (EncodingAESKey)", cfg.Platforms.WeCom.AESKey)
	fmt.Println("\n  > WeCom configured")
}

func stepDingTalk(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.DingTalk.ClientID = promptText("DingTalk AppKey (ClientID)", cfg.Platforms.DingTalk.ClientID)
	cfg.Platforms.DingTalk.ClientSecret = promptText("DingTalk AppSecret (ClientSecret)", cfg.Platforms.DingTalk.ClientSecret)
	fmt.Println("\n  > DingTalk configured")
}

func stepFeishu(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Feishu.AppID = promptText("Feishu App ID", cfg.Platforms.Feishu.AppID)
	cfg.Platforms.Feishu.AppSecret = promptText("Feishu App Secret", cfg.Platforms.Feishu.AppSecret)
	fmt.Println("\n  > Feishu configured")
}

func stepSlack(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Slack.BotToken = promptText("Slack Bot Token (xoxb-...)", cfg.Platforms.Slack.BotToken)
	cfg.Platforms.Slack.AppToken = promptText("Slack App Token (xapp-...)", cfg.Platforms.Slack.AppToken)
	fmt.Println("\n  > Slack configured")
}

func stepTelegram(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Telegram.Token = promptText("Telegram Bot Token", cfg.Platforms.Telegram.Token)
	fmt.Println("\n  > Telegram configured")
}

func stepDiscord(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Discord.Token = promptText("Discord Bot Token", cfg.Platforms.Discord.Token)
	fmt.Println("\n  > Discord configured")
}

func stepWhatsApp(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.WhatsApp.PhoneNumberID = promptText("WhatsApp Phone Number ID", cfg.Platforms.WhatsApp.PhoneNumberID)
	cfg.Platforms.WhatsApp.AccessToken = promptText("WhatsApp Access Token", cfg.Platforms.WhatsApp.AccessToken)
	cfg.Platforms.WhatsApp.VerifyToken = promptText("WhatsApp Verify Token", cfg.Platforms.WhatsApp.VerifyToken)
	fmt.Println("\n  > WhatsApp configured")
}

func stepLINE(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.LINE.ChannelSecret = promptText("LINE Channel Secret", cfg.Platforms.LINE.ChannelSecret)
	cfg.Platforms.LINE.ChannelToken = promptText("LINE Channel Token", cfg.Platforms.LINE.ChannelToken)
	fmt.Println("\n  > LINE configured")
}

func stepTeams(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Teams.AppID = promptText("Teams App ID", cfg.Platforms.Teams.AppID)
	cfg.Platforms.Teams.AppPassword = promptText("Teams App Password", cfg.Platforms.Teams.AppPassword)
	cfg.Platforms.Teams.TenantID = promptText("Teams Tenant ID", cfg.Platforms.Teams.TenantID)
	fmt.Println("\n  > Teams configured")
}

func stepMatrix(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Matrix.HomeserverURL = promptText("Matrix Homeserver URL", cfg.Platforms.Matrix.HomeserverURL)
	cfg.Platforms.Matrix.UserID = promptText("Matrix User ID (@bot:server)", cfg.Platforms.Matrix.UserID)
	cfg.Platforms.Matrix.AccessToken = promptText("Matrix Access Token", cfg.Platforms.Matrix.AccessToken)
	fmt.Println("\n  > Matrix configured")
}

func stepGoogleChat(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.GoogleChat.ProjectID = promptText("Google Chat Project ID", cfg.Platforms.GoogleChat.ProjectID)
	cfg.Platforms.GoogleChat.CredentialsFile = promptText("Google Chat Credentials File", cfg.Platforms.GoogleChat.CredentialsFile)
	fmt.Println("\n  > Google Chat configured")
}

func stepMattermost(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Mattermost.ServerURL = promptText("Mattermost Server URL", cfg.Platforms.Mattermost.ServerURL)
	cfg.Platforms.Mattermost.Token = promptText("Mattermost Token", cfg.Platforms.Mattermost.Token)
	cfg.Platforms.Mattermost.TeamName = promptText("Mattermost Team Name", cfg.Platforms.Mattermost.TeamName)
	fmt.Println("\n  > Mattermost configured")
}

func stepIMessage(cfg *config.Config) {
	fmt.Println()
	fmt.Println("  iMessage requires BlueBubbles server running on macOS.")
	cfg.Platforms.IMessage.BlueBubblesURL = promptText("BlueBubbles Server URL", cfg.Platforms.IMessage.BlueBubblesURL)
	cfg.Platforms.IMessage.BlueBubblesPassword = promptText("BlueBubbles Password", cfg.Platforms.IMessage.BlueBubblesPassword)
	fmt.Println("\n  > iMessage configured")
}

func stepSignal(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Signal.APIURL = promptText("Signal API URL (signal-cli REST)", cfg.Platforms.Signal.APIURL)
	cfg.Platforms.Signal.PhoneNumber = promptText("Signal Phone Number", cfg.Platforms.Signal.PhoneNumber)
	fmt.Println("\n  > Signal configured")
}

func stepTwitch(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Twitch.Token = promptText("Twitch OAuth Token", cfg.Platforms.Twitch.Token)
	cfg.Platforms.Twitch.Channel = promptText("Twitch Channel", cfg.Platforms.Twitch.Channel)
	cfg.Platforms.Twitch.BotName = promptText("Twitch Bot Name", cfg.Platforms.Twitch.BotName)
	fmt.Println("\n  > Twitch configured")
}

func stepNOSTR(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.NOSTR.PrivateKey = promptText("NOSTR Private Key (hex or nsec)", cfg.Platforms.NOSTR.PrivateKey)
	cfg.Platforms.NOSTR.Relays = promptText("NOSTR Relay URLs (comma-separated)", cfg.Platforms.NOSTR.Relays)
	fmt.Println("\n  > NOSTR configured")
}

func stepZalo(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Zalo.AppID = promptText("Zalo App ID", cfg.Platforms.Zalo.AppID)
	cfg.Platforms.Zalo.SecretKey = promptText("Zalo Secret Key", cfg.Platforms.Zalo.SecretKey)
	cfg.Platforms.Zalo.AccessToken = promptText("Zalo Access Token", cfg.Platforms.Zalo.AccessToken)
	fmt.Println("\n  > Zalo configured")
}

func stepNextcloud(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Nextcloud.ServerURL = promptText("Nextcloud Server URL", cfg.Platforms.Nextcloud.ServerURL)
	cfg.Platforms.Nextcloud.Username = promptText("Nextcloud Username", cfg.Platforms.Nextcloud.Username)
	cfg.Platforms.Nextcloud.Password = promptText("Nextcloud Password", cfg.Platforms.Nextcloud.Password)
	cfg.Platforms.Nextcloud.RoomToken = promptText("Nextcloud Talk Room Token", cfg.Platforms.Nextcloud.RoomToken)
	fmt.Println("\n  > Nextcloud Talk configured")
}

func stepWeChat(cfg *config.Config) {
	fmt.Println()
	fmt.Println("  WeChat works via the cloud relay service.")
	fmt.Println("  1. Follow the official lsbot on WeChat")
	fmt.Println("  2. Send /whoami to get your user ID")
	fmt.Println("  3. Enter the user ID below")
	fmt.Println()
	cfg.Relay.UserID = promptText("WeChat User ID (from /whoami)", cfg.Relay.UserID)
	cfg.Relay.Platform = "wechat"
	cfg.Mode = "relay"
	fmt.Println("\n  > WeChat configured (relay mode auto-selected)")
}

func stepConnectionMode(cfg *config.Config) {
	fmt.Println("\n  Step 3/3: Connection Mode")

	// If mode was already set by platform (e.g., wechat forces relay), skip
	if cfg.Mode == "relay" && cfg.Relay.Platform != "" {
		fmt.Printf("\n  > Connection mode: relay (set by platform)\n")
		return
	}

	defIdx := 0
	if cfg.Mode == "router" {
		defIdx = 1
	}

	idx := promptSelect("Select connection mode:", []string{
		"relay   (cloud relay, recommended, no public server needed)",
		"router  (self-hosted, requires public IP)",
	}, defIdx)

	if idx == 0 {
		cfg.Mode = "relay"
		stepRelayConfig(cfg)
	} else {
		cfg.Mode = "router"
	}

	fmt.Printf("\n  > Connection mode: %s\n", cfg.Mode)
}

// stepRelayConfig prompts for relay-specific settings when relay mode is selected.
func stepRelayConfig(cfg *config.Config) {
	// If relay platform already set (e.g., wechat), skip
	if cfg.Relay.Platform != "" {
		return
	}

	// Determine relay platform from configured platform credentials
	relayPlatforms := []string{}
	if cfg.Platforms.Feishu.AppID != "" {
		relayPlatforms = append(relayPlatforms, "feishu")
	}
	if cfg.Platforms.Slack.BotToken != "" {
		relayPlatforms = append(relayPlatforms, "slack")
	}
	if cfg.Platforms.WeCom.CorpID != "" {
		relayPlatforms = append(relayPlatforms, "wecom")
	}

	// For feishu/slack relay, prompt for user ID
	if len(relayPlatforms) == 1 {
		cfg.Relay.Platform = relayPlatforms[0]
	}

	if cfg.Relay.Platform == "feishu" || cfg.Relay.Platform == "slack" {
		fmt.Println()
		fmt.Println("  Relay mode requires a user ID from the official bot.")
		fmt.Println("  Send /whoami to the bot to get your user ID.")
		cfg.Relay.UserID = promptText("Relay User ID (from /whoami, or leave empty)", cfg.Relay.UserID)
	}
}

func printOnboardSummary(cfg *config.Config) {
	fmt.Println()
	fmt.Println("  ───────────────────────────────────")
	fmt.Printf("  > Configuration saved to %s\n", config.ConfigPath())
	fmt.Println()

	if cfg.Mode == "relay" {
		if cfg.Relay.UserID != "" && cfg.Relay.Platform != "" {
			fmt.Println("  To start the bot, run:")
			fmt.Println("    lsbot relay")
		} else if cfg.Relay.Platform == "wecom" {
			fmt.Println("  To start the bot, run:")
			fmt.Println("    lsbot relay --platform wecom")
		} else {
			fmt.Println("  To start the bot, run:")
			fmt.Println("    lsbot relay --platform <platform> --user-id <your-id>")
			fmt.Println()
			fmt.Println("  Get your user ID by sending /whoami to the official bot.")
		}
	} else {
		fmt.Println("  To start the bot, run:")
		fmt.Println("    lsbot gateway")
	}
	fmt.Println()
	fmt.Println("  To reconfigure:")
	fmt.Println("    lsbot onboard")
	fmt.Println()
}
