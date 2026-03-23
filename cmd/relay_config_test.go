package cmd

import (
	"testing"

	"github.com/ruilisi/lsbot/internal/config"
)

func TestResolveRelayAI(t *testing.T) {
	kimiAgent := config.AgentEntry{
		ID:       "kimi",
		Provider: "kimi",
		APIKey:   "kimi-key",
		Model:    "kimi-k2.5",
	}
	defaultAgent := config.AgentEntry{
		ID:       "default",
		Default:  true,
		Provider: "minimax",
		APIKey:   "minimax-key",
		BaseURL:  "https://api.minimax.chat/v1",
		Model:    "MiniMax-M2.5",
	}

	tests := []struct {
		name         string
		cfg          *config.Config
		inProvider   string
		inAPIKey     string
		inBaseURL    string
		inModel      string
		wantProvider string
		wantAPIKey   string
		wantModel    string
	}{
		{
			name: "relay.provider=kimi overrides default agent",
			cfg: &config.Config{
				Relay:  config.RelayConfig{Provider: "kimi"},
				Agents: []config.AgentEntry{defaultAgent, kimiAgent},
			},
			wantProvider: "kimi",
			wantAPIKey:   "kimi-key",
			wantModel:    "kimi-k2.5",
		},
		{
			name: "no relay.provider falls back to default agent",
			cfg: &config.Config{
				Agents: []config.AgentEntry{defaultAgent, kimiAgent},
			},
			wantProvider: "minimax",
			wantAPIKey:   "minimax-key",
			wantModel:    "MiniMax-M2.5",
		},
		{
			name: "CLI --provider overrides relay.provider",
			cfg: &config.Config{
				Relay:  config.RelayConfig{Provider: "kimi"},
				Agents: []config.AgentEntry{defaultAgent, kimiAgent},
			},
			inProvider:   "minimax",
			wantProvider: "minimax",
			wantAPIKey:   "minimax-key",
			wantModel:    "MiniMax-M2.5",
		},
		{
			name: "relay.provider=kimi but CLI --model wins",
			cfg: &config.Config{
				Relay:  config.RelayConfig{Provider: "kimi"},
				Agents: []config.AgentEntry{defaultAgent, kimiAgent},
			},
			inModel:      "custom-model",
			wantProvider: "kimi",
			wantAPIKey:   "kimi-key",
			wantModel:    "custom-model",
		},
		{
			name: "relay.provider matches agent by provider type not just ID",
			cfg: &config.Config{
				Relay: config.RelayConfig{Provider: "kimi"},
				Agents: []config.AgentEntry{
					defaultAgent,
					{ID: "my-kimi", Provider: "kimi", APIKey: "kimi-key2", Model: "kimi-k2.5"},
				},
			},
			wantProvider: "kimi",
			wantAPIKey:   "kimi-key2",
			wantModel:    "kimi-k2.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProvider, gotAPIKey, _, gotModel := resolveRelayAI(tt.cfg, tt.inProvider, tt.inAPIKey, tt.inBaseURL, tt.inModel)
			if gotProvider != tt.wantProvider {
				t.Errorf("provider = %q, want %q", gotProvider, tt.wantProvider)
			}
			if gotAPIKey != tt.wantAPIKey {
				t.Errorf("apiKey = %q, want %q", gotAPIKey, tt.wantAPIKey)
			}
			if gotModel != tt.wantModel {
				t.Errorf("model = %q, want %q", gotModel, tt.wantModel)
			}
		})
	}
}

func TestApplyRelayPlatformFallback(t *testing.T) {
	tests := []struct {
		name        string
		platform    string // explicit CLI/env value
		userID      string // explicit CLI/env value
		cfgPlatform string // relay.platform from ~/.lsbot.yaml
		botID       string // bot_id from ~/.lsbot.yaml
		wantPlatform string
	}{
		{
			name:         "bot-page-only: botID set, no platform/userID — config platform must NOT apply",
			platform:     "",
			userID:       "",
			cfgPlatform:  "feishu",
			botID:        "some-bot-id",
			wantPlatform: "",
		},
		{
			name:         "normal relay: no botID — config platform applies",
			platform:     "",
			userID:       "",
			cfgPlatform:  "feishu",
			botID:        "",
			wantPlatform: "feishu",
		},
		{
			name:         "explicit platform always wins over config",
			platform:     "slack",
			userID:       "u123",
			cfgPlatform:  "feishu",
			botID:        "some-bot-id",
			wantPlatform: "slack",
		},
		{
			name:         "botID set but userID also set — config platform applies (user specified a platform relay)",
			platform:     "",
			userID:       "u123",
			cfgPlatform:  "feishu",
			botID:        "some-bot-id",
			wantPlatform: "feishu",
		},
		{
			name:         "no botID, no config platform — nothing changes",
			platform:     "",
			userID:       "",
			cfgPlatform:  "",
			botID:        "",
			wantPlatform: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPlatform, _ := applyRelayPlatformFallback(tt.platform, tt.userID, tt.cfgPlatform, tt.botID)
			if gotPlatform != tt.wantPlatform {
				t.Errorf("platform = %q, want %q", gotPlatform, tt.wantPlatform)
			}
		})
	}
}
