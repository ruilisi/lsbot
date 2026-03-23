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
		wantBaseURL  string
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
			wantBaseURL:  "https://api.minimax.chat/v1",
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
			wantBaseURL:  "https://api.minimax.chat/v1",
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
		{
			// Regression: kimi has no base_url; default agent (minimax) has one.
			// resolveRelayAI must NOT inherit minimax's base_url for kimi.
			name: "kimi provider must not inherit minimax base_url from default agent",
			cfg: &config.Config{
				Relay:  config.RelayConfig{Provider: "kimi"},
				Agents: []config.AgentEntry{defaultAgent, kimiAgent},
			},
			wantProvider: "kimi",
			wantAPIKey:   "kimi-key",
			wantBaseURL:  "", // kimi has no base_url; must not get minimax's URL
			wantModel:    "kimi-k2.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProvider, gotAPIKey, gotBaseURL, gotModel := resolveRelayAI(tt.cfg, tt.inProvider, tt.inAPIKey, tt.inBaseURL, tt.inModel)
			if gotProvider != tt.wantProvider {
				t.Errorf("provider = %q, want %q", gotProvider, tt.wantProvider)
			}
			if gotAPIKey != tt.wantAPIKey {
				t.Errorf("apiKey = %q, want %q", gotAPIKey, tt.wantAPIKey)
			}
			if gotBaseURL != tt.wantBaseURL {
				t.Errorf("baseURL = %q, want %q", gotBaseURL, tt.wantBaseURL)
			}
			if gotModel != tt.wantModel {
				t.Errorf("model = %q, want %q", gotModel, tt.wantModel)
			}
		})
	}
}

func TestApplyResolvedAIToConfig(t *testing.T) {
	makeConfig := func() *config.Config {
		return &config.Config{
			Agents: []config.AgentEntry{
				{ID: "default", Default: true, Provider: "minimax", APIKey: "minimax-key", Model: "MiniMax-M2.5"},
				{ID: "kimi", Provider: "kimi", APIKey: "kimi-key", Model: "kimi-k2.5"},
			},
		}
	}

	t.Run("relay.provider updates only default agent", func(t *testing.T) {
		cfg := makeConfig()
		applyResolvedAIToConfig(cfg, "kimi", "kimi-key", "", "kimi-k2.5", false)

		// savedCfg.AI must reflect kimi
		if cfg.AI.Provider != "kimi" {
			t.Errorf("AI.Provider = %q, want kimi", cfg.AI.Provider)
		}
		if cfg.AI.Model != "kimi-k2.5" {
			t.Errorf("AI.Model = %q, want kimi-k2.5", cfg.AI.Model)
		}

		// default agent entry must now be kimi
		def, _ := cfg.FindAgent("default")
		if def.Provider != "kimi" {
			t.Errorf("default agent Provider = %q, want kimi", def.Provider)
		}
		if def.Model != "kimi-k2.5" {
			t.Errorf("default agent Model = %q, want kimi-k2.5", def.Model)
		}

		// non-default agent must be untouched
		kimiEntry, _ := cfg.FindAgent("kimi")
		if kimiEntry.Provider != "kimi" || kimiEntry.APIKey != "kimi-key" {
			t.Errorf("kimi agent was unexpectedly modified: %+v", kimiEntry)
		}
	})

	t.Run("CLI override clears all agent provider fields", func(t *testing.T) {
		cfg := makeConfig()
		applyResolvedAIToConfig(cfg, "deepseek", "ds-key", "https://api.deepseek.com/v1", "deepseek-chat", true)

		for _, a := range cfg.Agents {
			if a.Provider != "" || a.APIKey != "" || a.Model != "" {
				t.Errorf("agent %q not cleared after CLI override: provider=%q apiKey=%q model=%q",
					a.ID, a.Provider, a.APIKey, a.Model)
			}
		}
		if cfg.AI.Provider != "deepseek" {
			t.Errorf("AI.Provider = %q, want deepseek", cfg.AI.Provider)
		}
	})

	t.Run("ResolveAgentAI on default agent returns kimi after relay.provider applied", func(t *testing.T) {
		cfg := makeConfig()
		applyResolvedAIToConfig(cfg, "kimi", "kimi-key", "", "kimi-k2.5", false)

		entry, _ := cfg.FindAgent("default")
		ai := cfg.ResolveAgentAI(entry)
		if ai.Provider != "kimi" {
			t.Errorf("ResolveAgentAI provider = %q, want kimi", ai.Provider)
		}
		if ai.Model != "kimi-k2.5" {
			t.Errorf("ResolveAgentAI model = %q, want kimi-k2.5", ai.Model)
		}
		if ai.APIKey != "kimi-key" {
			t.Errorf("ResolveAgentAI apiKey = %q, want kimi-key", ai.APIKey)
		}
	})
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
