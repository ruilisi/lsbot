package cmd

import (
	"fmt"
	"strings"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/spf13/cobra"
)

var channelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "Manage platform channel credentials",
}

// channels add flags — shared across platforms since only one --channel is given per run.
var (
	channelName string

	// generic credential flags (reused across platforms by name similarity)
	caToken           string // telegram, discord, twitch, mattermost, matrix, nostr
	caBotToken        string // slack
	caAppToken        string // slack
	caAppID           string // feishu, teams, zalo, googlechat
	caAppSecret       string // feishu
	caAppPassword     string // teams
	caTenantID        string // teams
	caClientID        string // dingtalk
	caClientSecret    string // dingtalk
	caCorpID          string // wecom
	caAgentID         string // wecom
	caSecret          string // wecom, zalo
	caAESKey          string // wecom
	caPort            int    // wecom, webapp
	caPhoneID         string // whatsapp
	caAccessToken     string // whatsapp, matrix, zalo
	caVerifyToken     string // whatsapp
	caChannelSecret   string // line
	caChannelToken    string // line
	caHomeserverURL   string // matrix
	caUserID          string // matrix, nextcloud
	caServerURL       string // mattermost, nextcloud, signal
	caTeamName        string // mattermost
	caAPIURL          string // signal
	caPhoneNumber     string // signal
	caBlueBubblesURL  string // imessage
	caBlueBubblesPass string // imessage
	caChannelNameFlag string // twitch
	caBotName         string // twitch
	caPrivateKey      string // nostr
	caRelays          string // nostr
	caSecretKey       string // zalo
	caUsername        string // nextcloud
	caPassword        string // nextcloud
	caRoomToken       string // nextcloud
	caProjectID       string // googlechat
	caCredentialsFile string // googlechat
	caAuthToken       string // webapp
)

// anyFlagChanged returns true if any of the named flags were explicitly set.
func anyFlagChanged(cmd *cobra.Command, names ...string) bool {
	for _, n := range names {
		if cmd.Flags().Changed(n) {
			return true
		}
	}
	return false
}

var channelsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add or update channel credentials in ~/.lsbot.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		if channelName == "" {
			// interactive: pick platform
			initScanner()
			options := make([]string, len(platformOptions))
			for i, p := range platformOptions {
				options[i] = p.label
			}
			idx := promptSelect("Select platform to configure:", options, 0)
			channelName = platformOptions[idx].name
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// ensure scanner is ready for any interactive step* calls
		if scanner == nil {
			initScanner()
		}

		switch strings.ToLower(channelName) {
		case "telegram":
			if anyFlagChanged(cmd, "token") {
				cfg.Platforms.Telegram.Token = caToken
			} else {
				stepTelegram(cfg)
			}
		case "slack":
			if anyFlagChanged(cmd, "bot-token", "app-token") {
				if caBotToken != "" {
					cfg.Platforms.Slack.BotToken = caBotToken
				}
				if caAppToken != "" {
					cfg.Platforms.Slack.AppToken = caAppToken
				}
			} else {
				stepSlack(cfg)
			}
		case "discord":
			if anyFlagChanged(cmd, "token") {
				cfg.Platforms.Discord.Token = caToken
			} else {
				stepDiscord(cfg)
			}
		case "feishu":
			if anyFlagChanged(cmd, "app-id", "app-secret") {
				if caAppID != "" {
					cfg.Platforms.Feishu.AppID = caAppID
				}
				if caAppSecret != "" {
					cfg.Platforms.Feishu.AppSecret = caAppSecret
				}
			} else {
				stepFeishu(cfg)
			}
		case "dingtalk":
			if anyFlagChanged(cmd, "client-id", "client-secret") {
				if caClientID != "" {
					cfg.Platforms.DingTalk.ClientID = caClientID
				}
				if caClientSecret != "" {
					cfg.Platforms.DingTalk.ClientSecret = caClientSecret
				}
			} else {
				stepDingTalk(cfg)
			}
		case "wecom":
			if anyFlagChanged(cmd, "corp-id", "agent-id", "secret", "token", "aes-key", "port") {
				if caCorpID != "" {
					cfg.Platforms.WeCom.CorpID = caCorpID
				}
				if caAgentID != "" {
					cfg.Platforms.WeCom.AgentID = caAgentID
				}
				if caSecret != "" {
					cfg.Platforms.WeCom.Secret = caSecret
				}
				if caToken != "" {
					cfg.Platforms.WeCom.Token = caToken
				}
				if caAESKey != "" {
					cfg.Platforms.WeCom.AESKey = caAESKey
				}
				if caPort != 0 {
					cfg.Platforms.WeCom.CallbackPort = caPort
				}
			} else {
				stepWecom(cfg)
			}
		case "whatsapp":
			if anyFlagChanged(cmd, "phone-id", "access-token", "verify-token") {
				if caPhoneID != "" {
					cfg.Platforms.WhatsApp.PhoneNumberID = caPhoneID
				}
				if caAccessToken != "" {
					cfg.Platforms.WhatsApp.AccessToken = caAccessToken
				}
				if caVerifyToken != "" {
					cfg.Platforms.WhatsApp.VerifyToken = caVerifyToken
				}
			} else {
				stepWhatsApp(cfg)
			}
		case "line":
			if anyFlagChanged(cmd, "channel-secret", "channel-token") {
				if caChannelSecret != "" {
					cfg.Platforms.LINE.ChannelSecret = caChannelSecret
				}
				if caChannelToken != "" {
					cfg.Platforms.LINE.ChannelToken = caChannelToken
				}
			} else {
				stepLINE(cfg)
			}
		case "teams":
			if anyFlagChanged(cmd, "app-id", "app-password", "tenant-id") {
				if caAppID != "" {
					cfg.Platforms.Teams.AppID = caAppID
				}
				if caAppPassword != "" {
					cfg.Platforms.Teams.AppPassword = caAppPassword
				}
				if caTenantID != "" {
					cfg.Platforms.Teams.TenantID = caTenantID
				}
			} else {
				stepTeams(cfg)
			}
		case "matrix":
			if anyFlagChanged(cmd, "homeserver-url", "user-id", "access-token") {
				if caHomeserverURL != "" {
					cfg.Platforms.Matrix.HomeserverURL = caHomeserverURL
				}
				if caUserID != "" {
					cfg.Platforms.Matrix.UserID = caUserID
				}
				if caAccessToken != "" {
					cfg.Platforms.Matrix.AccessToken = caAccessToken
				}
			} else {
				stepMatrix(cfg)
			}
		case "mattermost":
			if anyFlagChanged(cmd, "server-url", "token", "team-name") {
				if caServerURL != "" {
					cfg.Platforms.Mattermost.ServerURL = caServerURL
				}
				if caToken != "" {
					cfg.Platforms.Mattermost.Token = caToken
				}
				if caTeamName != "" {
					cfg.Platforms.Mattermost.TeamName = caTeamName
				}
			} else {
				stepMattermost(cfg)
			}
		case "signal":
			if anyFlagChanged(cmd, "api-url", "phone-number") {
				if caAPIURL != "" {
					cfg.Platforms.Signal.APIURL = caAPIURL
				}
				if caPhoneNumber != "" {
					cfg.Platforms.Signal.PhoneNumber = caPhoneNumber
				}
			} else {
				stepSignal(cfg)
			}
		case "imessage":
			if anyFlagChanged(cmd, "bluebubbles-url", "bluebubbles-password") {
				if caBlueBubblesURL != "" {
					cfg.Platforms.IMessage.BlueBubblesURL = caBlueBubblesURL
				}
				if caBlueBubblesPass != "" {
					cfg.Platforms.IMessage.BlueBubblesPassword = caBlueBubblesPass
				}
			} else {
				stepIMessage(cfg)
			}
		case "twitch":
			if anyFlagChanged(cmd, "token", "channel-name", "bot-name") {
				if caToken != "" {
					cfg.Platforms.Twitch.Token = caToken
				}
				if caChannelNameFlag != "" {
					cfg.Platforms.Twitch.Channel = caChannelNameFlag
				}
				if caBotName != "" {
					cfg.Platforms.Twitch.BotName = caBotName
				}
			} else {
				stepTwitch(cfg)
			}
		case "nostr":
			if anyFlagChanged(cmd, "private-key", "relays") {
				if caPrivateKey != "" {
					cfg.Platforms.NOSTR.PrivateKey = caPrivateKey
				}
				if caRelays != "" {
					cfg.Platforms.NOSTR.Relays = caRelays
				}
			} else {
				stepNOSTR(cfg)
			}
		case "zalo":
			if anyFlagChanged(cmd, "app-id", "secret-key", "access-token") {
				if caAppID != "" {
					cfg.Platforms.Zalo.AppID = caAppID
				}
				if caSecretKey != "" {
					cfg.Platforms.Zalo.SecretKey = caSecretKey
				}
				if caAccessToken != "" {
					cfg.Platforms.Zalo.AccessToken = caAccessToken
				}
			} else {
				stepZalo(cfg)
			}
		case "nextcloud":
			if anyFlagChanged(cmd, "server-url", "username", "password", "room-token") {
				if caServerURL != "" {
					cfg.Platforms.Nextcloud.ServerURL = caServerURL
				}
				if caUsername != "" {
					cfg.Platforms.Nextcloud.Username = caUsername
				}
				if caPassword != "" {
					cfg.Platforms.Nextcloud.Password = caPassword
				}
				if caRoomToken != "" {
					cfg.Platforms.Nextcloud.RoomToken = caRoomToken
				}
			} else {
				stepNextcloud(cfg)
			}
		case "googlechat":
			if anyFlagChanged(cmd, "project-id", "credentials-file") {
				if caProjectID != "" {
					cfg.Platforms.GoogleChat.ProjectID = caProjectID
				}
				if caCredentialsFile != "" {
					cfg.Platforms.GoogleChat.CredentialsFile = caCredentialsFile
				}
			} else {
				stepGoogleChat(cfg)
			}
		case "webapp":
			if anyFlagChanged(cmd, "port", "auth-token") {
				if caPort != 0 {
					cfg.Platforms.Webapp.Port = caPort
				}
				if caAuthToken != "" {
					cfg.Platforms.Webapp.Token = caAuthToken
				}
			} else {
				if scanner == nil {
					initScanner()
				}
				portStr := promptText("Port", "8080")
				fmt.Sscan(portStr, &cfg.Platforms.Webapp.Port)
				cfg.Platforms.Webapp.Token = promptText("Auth Token (optional)", cfg.Platforms.Webapp.Token)
				fmt.Println("\n  > Webapp configured")
			}
		case "skip":
			fmt.Println("  > Channel configuration skipped")
			return nil
		case "wechat":
			stepWeChat(cfg)
		default:
			return fmt.Errorf("unknown channel: %q", channelName)
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Channel %q saved to %s\n", channelName, config.ConfigPath())
		return nil
	},
}

var channelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured channels",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		type row struct {
			name       string
			configured bool
			detail     string
		}

		rows := []row{
			{"telegram", cfg.Platforms.Telegram.Token != "", cfg.Platforms.Telegram.Token},
			{"slack", cfg.Platforms.Slack.BotToken != "" && cfg.Platforms.Slack.AppToken != "", cfg.Platforms.Slack.BotToken},
			{"discord", cfg.Platforms.Discord.Token != "", cfg.Platforms.Discord.Token},
			{"feishu", cfg.Platforms.Feishu.AppID != "", cfg.Platforms.Feishu.AppID},
			{"dingtalk", cfg.Platforms.DingTalk.ClientID != "", cfg.Platforms.DingTalk.ClientID},
			{"wecom", cfg.Platforms.WeCom.CorpID != "", cfg.Platforms.WeCom.CorpID},
			{"whatsapp", cfg.Platforms.WhatsApp.PhoneNumberID != "", cfg.Platforms.WhatsApp.PhoneNumberID},
			{"line", cfg.Platforms.LINE.ChannelSecret != "", cfg.Platforms.LINE.ChannelSecret},
			{"teams", cfg.Platforms.Teams.AppID != "", cfg.Platforms.Teams.AppID},
			{"matrix", cfg.Platforms.Matrix.HomeserverURL != "", cfg.Platforms.Matrix.HomeserverURL},
			{"mattermost", cfg.Platforms.Mattermost.ServerURL != "", cfg.Platforms.Mattermost.ServerURL},
			{"signal", cfg.Platforms.Signal.APIURL != "", cfg.Platforms.Signal.APIURL},
			{"imessage", cfg.Platforms.IMessage.BlueBubblesURL != "", cfg.Platforms.IMessage.BlueBubblesURL},
			{"twitch", cfg.Platforms.Twitch.Token != "", cfg.Platforms.Twitch.Channel},
			{"nostr", cfg.Platforms.NOSTR.PrivateKey != "", cfg.Platforms.NOSTR.Relays},
			{"zalo", cfg.Platforms.Zalo.AppID != "", cfg.Platforms.Zalo.AppID},
			{"nextcloud", cfg.Platforms.Nextcloud.ServerURL != "", cfg.Platforms.Nextcloud.ServerURL},
			{"googlechat", cfg.Platforms.GoogleChat.ProjectID != "", cfg.Platforms.GoogleChat.ProjectID},
			{"webapp", cfg.Platforms.Webapp.Port != 0, fmt.Sprintf("port=%d", cfg.Platforms.Webapp.Port)},
		}

		fmt.Printf("%-12s  %-6s  %s\n", "CHANNEL", "STATUS", "DETAIL")
		fmt.Printf("%-12s  %-6s  %s\n", "-------", "------", "------")
		for _, r := range rows {
			status := "✗"
			if r.configured {
				status = "✓"
			}
			detail := r.detail
			if len(detail) > 40 {
				detail = detail[:37] + "..."
			}
			fmt.Printf("%-12s  %-6s  %s\n", r.name, status, detail)
		}
		return nil
	},
}

var channelsRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove channel credentials from ~/.lsbot.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		if channelName == "" {
			return fmt.Errorf("--channel is required")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		switch strings.ToLower(channelName) {
		case "telegram":
			cfg.Platforms.Telegram = config.TelegramConfig{}
		case "slack":
			cfg.Platforms.Slack = config.SlackConfig{}
		case "discord":
			cfg.Platforms.Discord = config.DiscordConfig{}
		case "feishu":
			cfg.Platforms.Feishu = config.FeishuConfig{}
		case "dingtalk":
			cfg.Platforms.DingTalk = config.DingTalkConfig{}
		case "wecom":
			cfg.Platforms.WeCom = config.WeComConfig{}
		case "whatsapp":
			cfg.Platforms.WhatsApp = config.WhatsAppConfig{}
		case "line":
			cfg.Platforms.LINE = config.LINEConfig{}
		case "teams":
			cfg.Platforms.Teams = config.TeamsConfig{}
		case "matrix":
			cfg.Platforms.Matrix = config.MatrixConfig{}
		case "mattermost":
			cfg.Platforms.Mattermost = config.MattermostConfig{}
		case "signal":
			cfg.Platforms.Signal = config.SignalConfig{}
		case "imessage":
			cfg.Platforms.IMessage = config.IMessageConfig{}
		case "twitch":
			cfg.Platforms.Twitch = config.TwitchConfig{}
		case "nostr":
			cfg.Platforms.NOSTR = config.NOSTRConfig{}
		case "zalo":
			cfg.Platforms.Zalo = config.ZaloConfig{}
		case "nextcloud":
			cfg.Platforms.Nextcloud = config.NextcloudConfig{}
		case "googlechat":
			cfg.Platforms.GoogleChat = config.GoogleChatConfig{}
		case "webapp":
			cfg.Platforms.Webapp = config.WebappConfig{}
		default:
			return fmt.Errorf("unknown channel: %q", channelName)
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Channel %q removed from %s\n", channelName, config.ConfigPath())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(channelsCmd)
	channelsCmd.AddCommand(channelsAddCmd)
	channelsCmd.AddCommand(channelsListCmd)
	channelsCmd.AddCommand(channelsRemoveCmd)

	// --channel flag shared by add and remove
	channelsAddCmd.Flags().StringVar(&channelName, "channel", "", "Channel name (required)")
	channelsRemoveCmd.Flags().StringVar(&channelName, "channel", "", "Channel name (required)")

	f := channelsAddCmd.Flags()

	// Generic credential flags — reused across platforms since only one --channel per run
	f.StringVar(&caToken, "token", "", "Bot/OAuth token (telegram, discord, wecom, mattermost, twitch, nostr)")
	f.StringVar(&caBotToken, "bot-token", "", "Bot token (slack)")
	f.StringVar(&caAppToken, "app-token", "", "App-level token (slack)")
	f.StringVar(&caAppID, "app-id", "", "App ID (feishu, teams, zalo, googlechat)")
	f.StringVar(&caAppSecret, "app-secret", "", "App secret (feishu)")
	f.StringVar(&caAppPassword, "app-password", "", "App password (teams)")
	f.StringVar(&caTenantID, "tenant-id", "", "Tenant ID (teams)")
	f.StringVar(&caClientID, "client-id", "", "Client ID (dingtalk)")
	f.StringVar(&caClientSecret, "client-secret", "", "Client secret (dingtalk)")
	f.StringVar(&caCorpID, "corp-id", "", "Corp ID (wecom)")
	f.StringVar(&caAgentID, "agent-id", "", "Agent ID (wecom)")
	f.StringVar(&caSecret, "secret", "", "Secret (wecom)")
	f.StringVar(&caAESKey, "aes-key", "", "EncodingAESKey (wecom)")
	f.IntVar(&caPort, "port", 0, "Callback/listen port (wecom, webapp)")
	f.StringVar(&caPhoneID, "phone-id", "", "Phone number ID (whatsapp)")
	f.StringVar(&caAccessToken, "access-token", "", "Access token (whatsapp, matrix, zalo)")
	f.StringVar(&caVerifyToken, "verify-token", "", "Verify token (whatsapp)")
	f.StringVar(&caChannelSecret, "channel-secret", "", "Channel secret (line)")
	f.StringVar(&caChannelToken, "channel-token", "", "Channel token (line)")
	f.StringVar(&caHomeserverURL, "homeserver-url", "", "Homeserver URL (matrix)")
	f.StringVar(&caUserID, "user-id", "", "User ID (matrix, nextcloud)")
	f.StringVar(&caServerURL, "server-url", "", "Server URL (mattermost, nextcloud, signal)")
	f.StringVar(&caTeamName, "team-name", "", "Team name (mattermost)")
	f.StringVar(&caAPIURL, "api-url", "", "API URL (signal)")
	f.StringVar(&caPhoneNumber, "phone-number", "", "Phone number (signal)")
	f.StringVar(&caBlueBubblesURL, "bluebubbles-url", "", "BlueBubbles server URL (imessage)")
	f.StringVar(&caBlueBubblesPass, "bluebubbles-password", "", "BlueBubbles password (imessage)")
	f.StringVar(&caChannelNameFlag, "channel-name", "", "Channel name (twitch)")
	f.StringVar(&caBotName, "bot-name", "", "Bot name (twitch)")
	f.StringVar(&caPrivateKey, "private-key", "", "Private key (nostr)")
	f.StringVar(&caRelays, "relays", "", "Relay URLs comma-separated (nostr)")
	f.StringVar(&caSecretKey, "secret-key", "", "Secret key (zalo)")
	f.StringVar(&caUsername, "username", "", "Username (nextcloud)")
	f.StringVar(&caPassword, "password", "", "Password (nextcloud)")
	f.StringVar(&caRoomToken, "room-token", "", "Room token (nextcloud)")
	f.StringVar(&caProjectID, "project-id", "", "Project ID (googlechat)")
	f.StringVar(&caCredentialsFile, "credentials-file", "", "Credentials JSON file path (googlechat)")
	f.StringVar(&caAuthToken, "auth-token", "", "Auth token (webapp)")
}
