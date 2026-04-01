package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Transport string                    `yaml:"transport"` // "stdio" or "sse"
	Port      int                       `yaml:"port"`
	Security  SecurityConfig            `yaml:"security"`
	Logging   LoggingConfig             `yaml:"logging"`
	AI        AIConfig                  `yaml:"ai,omitempty"`
	Providers map[string]ProviderEntry  `yaml:"providers,omitempty"`
	Platforms PlatformConfig            `yaml:"platforms,omitempty"`
	Mode      string                    `yaml:"mode,omitempty"` // "relay" or "router"
	Relay     RelayConfig               `yaml:"relay,omitempty"`
	Skills    SkillsConfig              `yaml:"skills,omitempty"`
	Browser   BrowserConfig             `yaml:"browser,omitempty"`
	Agents    []AgentEntry              `yaml:"agents,omitempty"`
	Bindings  []AgentBinding            `yaml:"bindings,omitempty"`
	BotID      string                    `yaml:"bot_id,omitempty"`
	E2EKeyFile string                    `yaml:"e2e_key_file,omitempty"` // path to PEM key file
}

// ProviderEntry defines a named AI provider configuration.
type ProviderEntry struct {
	Provider  string   `yaml:"provider,omitempty"`
	APIKey    string   `yaml:"api_key,omitempty"`
	BaseURL   string   `yaml:"base_url,omitempty"`
	Model     string   `yaml:"model,omitempty"`
	Fallbacks []string `yaml:"fallbacks,omitempty"` // ordered list of fallback provider names when quota is exhausted
}

// BrowserConfig configures browser automation.
type BrowserConfig struct {
	// ScreenSize controls the browser window size.
	// Use "fullscreen" for fullscreen mode, or "WIDTHxHEIGHT" (e.g. "1024x768").
	// Default: "fullscreen"
	ScreenSize string `yaml:"screen_size,omitempty"`

	// CDPURL is the Chrome DevTools Protocol address of a running Chrome instance
	// (e.g. "127.0.0.1:9222"). When set, EnsureRunning connects to this Chrome
	// instead of launching a new one. The Chrome must be started with
	// --remote-debugging-port=<port>.
	CDPURL string `yaml:"cdp_url,omitempty"`
}

type RelayConfig struct {
	UserID     string `yaml:"user_id,omitempty"`
	Platform   string `yaml:"platform,omitempty"`   // "feishu", "slack", "wechat", "wecom"
	Provider   string `yaml:"provider,omitempty"`   // references a named provider in Providers map
	ServerURL  string `yaml:"server_url,omitempty"` // WebSocket URL override
	WebhookURL string `yaml:"webhook_url,omitempty"` // Webhook URL override
}

type SkillsConfig struct {
	Disabled  []string `yaml:"disabled,omitempty"`
	ExtraDirs []string `yaml:"extra_dirs,omitempty"`
}

// SkillsDir returns the managed skills directory path
func SkillsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lsbot", "skills")
}

// HubDir returns ~/.lsbot/ — the lsbot data directory
func HubDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lsbot")
}

// HubSkillsDir returns ~/.lsbot/skills/ — where hub-installed skills live
func HubSkillsDir() string {
	return filepath.Join(HubDir(), "skills")
}

// MCPServerConfig describes one external MCP server to connect to.
type MCPServerConfig struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command,omitempty"`
	Args    []string `yaml:"args,omitempty"`
	Env     []string `yaml:"env,omitempty"`
	URL     string   `yaml:"url,omitempty"`
}

// AIOverride allows per-platform or per-channel AI provider settings.
type AIOverride struct {
	Platform  string `yaml:"platform,omitempty"`   // e.g. "telegram", "discord"
	ChannelID string `yaml:"channel_id,omitempty"` // optional: specific channel
	Provider  string `yaml:"provider,omitempty"`
	APIKey    string `yaml:"api_key,omitempty"`
	BaseURL   string `yaml:"base_url,omitempty"`
	Model     string `yaml:"model,omitempty"`
}

// AgentEntry defines a named agent with its own AI provider, model, and instructions.
type AgentEntry struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name,omitempty"`
	Default      bool     `yaml:"default,omitempty"`
	Provider     string   `yaml:"provider,omitempty"`
	APIKey       string   `yaml:"api_key,omitempty"`
	BaseURL      string   `yaml:"base_url,omitempty"`
	Model        string   `yaml:"model,omitempty"`
	Instructions string   `yaml:"instructions,omitempty"` // inline text or path to a file
	Workspace    string   `yaml:"workspace,omitempty"`    // workspace directory for this agent
	AllowTools   []string `yaml:"allow_tools,omitempty"`  // whitelist; empty = allow all
	DenyTools    []string `yaml:"deny_tools,omitempty"`   // blacklist; checked after allowlist
}

// AgentBindingMatch holds the filter criteria for a binding.
// All non-empty fields must match for the binding to apply.
type AgentBindingMatch struct {
	Platform  string `yaml:"platform,omitempty"`
	ChannelID string `yaml:"channel_id,omitempty"`
	UserID    string `yaml:"user_id,omitempty"`
}

// AgentBinding maps a match pattern to an agent ID.
type AgentBinding struct {
	AgentID string            `yaml:"agent_id"`
	Comment string            `yaml:"comment,omitempty"`
	Match   AgentBindingMatch `yaml:"match"`
}

type AIConfig struct {
	Provider   string            `yaml:"provider,omitempty"`
	APIKey     string            `yaml:"api_key,omitempty"`
	BaseURL    string            `yaml:"base_url,omitempty"`
	Model      string            `yaml:"model,omitempty"`
	MaxRounds       int               `yaml:"max_rounds,omitempty"`
	CallTimeoutSecs int               `yaml:"call_timeout_secs,omitempty"`
	MCPServers []MCPServerConfig `yaml:"mcp_servers,omitempty"`
	Overrides  []AIOverride      `yaml:"overrides,omitempty"`
}

// ResolveAI returns the AI settings for a given platform and channel,
// checking overrides from most specific (platform+channel) to least (platform only).
func (c *Config) ResolveAI(platform, channelID string) AIConfig {
	base := c.AI
	// Check for channel-specific override first, then platform-only
	for _, o := range c.AI.Overrides {
		if o.Platform == platform && o.ChannelID != "" && o.ChannelID == channelID {
			return applyOverride(base, o)
		}
	}
	for _, o := range c.AI.Overrides {
		if o.Platform == platform && o.ChannelID == "" {
			return applyOverride(base, o)
		}
	}
	return base
}

// ResolveProvider looks up a named provider. Resolution order:
//  1. Exact key match in Providers map
//  2. Scan Providers for entry whose .Provider field matches name
//  3. If Providers map is empty, construct from ai: block (backward compat)
//  4. If name is empty, fall back: relay.provider → ai.provider
func (c *Config) ResolveProvider(name string) (ProviderEntry, bool) {
	if name == "" {
		name = c.Relay.Provider
	}
	if name == "" {
		name = c.AI.Provider
	}
	if name == "" {
		return ProviderEntry{}, false
	}

	if len(c.Providers) > 0 {
		// 1. Exact key match
		if e, ok := c.Providers[name]; ok {
			if e.Provider == "" {
				e.Provider = name
			}
			return e, true
		}
		// 2. Match by provider type
		for _, e := range c.Providers {
			if strings.EqualFold(e.Provider, name) {
				return e, true
			}
		}
		return ProviderEntry{}, false
	}

	// 3. Backward compat: construct from ai: block
	if strings.EqualFold(c.AI.Provider, name) || name == "" {
		return ProviderEntry{
			Provider: c.AI.Provider,
			APIKey:   c.AI.APIKey,
			BaseURL:  c.AI.BaseURL,
			Model:    c.AI.Model,
		}, true
	}

	// 4. Search agents by ID or provider type
	for _, a := range c.Agents {
		if strings.EqualFold(a.ID, name) || strings.EqualFold(a.Provider, name) {
			ai := c.ResolveAgentAI(a)
			return ProviderEntry{
				Provider: ai.Provider,
				APIKey:   ai.APIKey,
				BaseURL:  ai.BaseURL,
				Model:    ai.Model,
			}, true
		}
	}

	// Provider name doesn't match anything — return entry with just the provider name
	// so CLI overrides (--api-key, --model) can fill in the rest
	return ProviderEntry{Provider: name}, true
}

func applyOverride(base AIConfig, o AIOverride) AIConfig {
	if o.Provider != "" && o.Provider != base.Provider {
		// Provider changed — clear base_url and model so the new provider's
		// defaults are used (e.g. don't pass deepseek's base_url to kimi).
		base.Provider = o.Provider
		base.BaseURL = ""
		base.Model = ""
	}
	if o.APIKey != "" {
		base.APIKey = o.APIKey
	}
	if o.BaseURL != "" {
		base.BaseURL = o.BaseURL
	}
	if o.Model != "" {
		base.Model = o.Model
	}
	base.Overrides = nil // don't propagate
	return base
}

// ResolveAgentAI returns the effective AIConfig for a named agent entry,
// inheriting from the base AI config for any fields left empty in the entry.
func (c *Config) ResolveAgentAI(entry AgentEntry) AIConfig {
	base := c.AI
	if entry.Provider != "" && entry.Provider != base.Provider {
		base.Provider = entry.Provider
		base.BaseURL = ""
		base.Model = ""
	}
	if entry.APIKey != "" {
		base.APIKey = entry.APIKey
	}
	if entry.BaseURL != "" {
		base.BaseURL = entry.BaseURL
	}
	if entry.Model != "" {
		base.Model = entry.Model
	}
	base.Overrides = nil
	return base
}

// FindAgent looks up an AgentEntry by ID.
func (c *Config) FindAgent(id string) (AgentEntry, bool) {
	for _, a := range c.Agents {
		if a.ID == id {
			return a, true
		}
	}
	return AgentEntry{}, false
}

// DefaultAgentID returns the ID of the agent marked default:true,
// falling back to the first agent in the list, or "" if the list is empty.
func (c *Config) DefaultAgentID() string {
	for _, a := range c.Agents {
		if a.Default {
			return a.ID
		}
	}
	if len(c.Agents) > 0 {
		return c.Agents[0].ID
	}
	return ""
}

type PlatformConfig struct {
	WeCom    WeComConfig    `yaml:"wecom,omitempty"`
	Slack    SlackConfig    `yaml:"slack,omitempty"`
	Telegram TelegramConfig `yaml:"telegram,omitempty"`
	Discord  DiscordConfig  `yaml:"discord,omitempty"`
	WeChat   WeChatConfig   `yaml:"wechat,omitempty"`
	Feishu   FeishuConfig   `yaml:"feishu,omitempty"`
	DingTalk DingTalkConfig `yaml:"dingtalk,omitempty"`
	WhatsApp WhatsAppConfig `yaml:"whatsapp,omitempty"`
	LINE     LINEConfig     `yaml:"line,omitempty"`
	Teams    TeamsConfig    `yaml:"teams,omitempty"`
	Matrix   MatrixConfig   `yaml:"matrix,omitempty"`
	GoogleChat GoogleChatConfig `yaml:"googlechat,omitempty"`
	Mattermost MattermostConfig `yaml:"mattermost,omitempty"`
	IMessage   IMessageConfig   `yaml:"imessage,omitempty"`
	Signal     SignalConfig     `yaml:"signal,omitempty"`
	Twitch     TwitchConfig     `yaml:"twitch,omitempty"`
	NOSTR      NOSTRConfig      `yaml:"nostr,omitempty"`
	Zalo       ZaloConfig       `yaml:"zalo,omitempty"`
	Nextcloud  NextcloudConfig  `yaml:"nextcloud,omitempty"`
	Webapp     WebappConfig     `yaml:"webapp,omitempty"`
}

type WeComConfig struct {
	CorpID       string `yaml:"corp_id,omitempty"`
	AgentID      string `yaml:"agent_id,omitempty"`
	Secret       string `yaml:"secret,omitempty"`
	Token        string `yaml:"token,omitempty"`
	AESKey       string `yaml:"aes_key,omitempty"`
	CallbackPort int    `yaml:"callback_port,omitempty"`
}

type SlackConfig struct {
	BotToken string `yaml:"bot_token,omitempty"`
	AppToken string `yaml:"app_token,omitempty"`
}

type TelegramConfig struct {
	Token string `yaml:"token,omitempty"`
}

type DiscordConfig struct {
	Token string `yaml:"token,omitempty"`
}

type WeChatConfig struct {
	AppID     string `yaml:"app_id,omitempty"`
	AppSecret string `yaml:"app_secret,omitempty"`
}

type FeishuConfig struct {
	AppID     string `yaml:"app_id,omitempty"`
	AppSecret string `yaml:"app_secret,omitempty"`
}

type DingTalkConfig struct {
	ClientID     string `yaml:"client_id,omitempty"`
	ClientSecret string `yaml:"client_secret,omitempty"`
}

type WhatsAppConfig struct {
	PhoneNumberID string `yaml:"phone_number_id,omitempty"`
	AccessToken   string `yaml:"access_token,omitempty"`
	VerifyToken   string `yaml:"verify_token,omitempty"`
}

type LINEConfig struct {
	ChannelSecret string `yaml:"channel_secret,omitempty"`
	ChannelToken  string `yaml:"channel_token,omitempty"`
}

type TeamsConfig struct {
	AppID       string `yaml:"app_id,omitempty"`
	AppPassword string `yaml:"app_password,omitempty"`
	TenantID    string `yaml:"tenant_id,omitempty"`
}

type MatrixConfig struct {
	HomeserverURL string `yaml:"homeserver_url,omitempty"`
	UserID        string `yaml:"user_id,omitempty"`
	AccessToken   string `yaml:"access_token,omitempty"`
}

type GoogleChatConfig struct {
	ProjectID       string `yaml:"project_id,omitempty"`
	CredentialsFile string `yaml:"credentials_file,omitempty"`
}

type MattermostConfig struct {
	ServerURL string `yaml:"server_url,omitempty"`
	Token     string `yaml:"token,omitempty"`
	TeamName  string `yaml:"team_name,omitempty"`
}

type IMessageConfig struct {
	BlueBubblesURL      string `yaml:"bluebubbles_url,omitempty"`
	BlueBubblesPassword string `yaml:"bluebubbles_password,omitempty"`
}

type SignalConfig struct {
	APIURL      string `yaml:"api_url,omitempty"`
	PhoneNumber string `yaml:"phone_number,omitempty"`
}

type TwitchConfig struct {
	Token   string `yaml:"token,omitempty"`
	Channel string `yaml:"channel,omitempty"`
	BotName string `yaml:"bot_name,omitempty"`
}

type NOSTRConfig struct {
	PrivateKey string `yaml:"private_key,omitempty"`
	Relays     string `yaml:"relays,omitempty"`
}

type ZaloConfig struct {
	AppID       string `yaml:"app_id,omitempty"`
	SecretKey   string `yaml:"secret_key,omitempty"`
	AccessToken string `yaml:"access_token,omitempty"`
}

type NextcloudConfig struct {
	ServerURL string `yaml:"server_url,omitempty"`
	Username  string `yaml:"username,omitempty"`
	Password  string `yaml:"password,omitempty"`
	RoomToken string `yaml:"room_token,omitempty"`
}

type WebappConfig struct {
	Port  int    `yaml:"port,omitempty"`
	Token string `yaml:"token,omitempty"`
}

type SecurityConfig struct {
	AllowedPaths        []string `yaml:"allowed_paths"`
	BlockedCommands     []string `yaml:"blocked_commands"`
	RequireConfirmation []string `yaml:"require_confirmation"`
	DisableFileTools    bool     `yaml:"disable_file_tools"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

func DefaultConfig() *Config {
	return &Config{
		Transport: "stdio",
		Port:      8686,
		Security: SecurityConfig{
			AllowedPaths:        []string{},
			BlockedCommands:     []string{"rm -rf /", "mkfs", "dd if="},
			RequireConfirmation: []string{},
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  "/tmp/lsbot.log",
		},
	}
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lsbot")
}

// overridePath is set via SetConfigPath to use a custom config file location.
var overridePath string

// SetConfigPath overrides the default ~/.lsbot.yaml path for all Load() calls.
func SetConfigPath(path string) {
	overridePath = path
}

func ConfigPath() string {
	if overridePath != "" {
		return overridePath
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lsbot.yaml")
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0600)
}
