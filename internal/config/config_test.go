package config

import "testing"

func TestResolveAI_NoOverride(t *testing.T) {
	cfg := &Config{AI: AIConfig{Provider: "claude", APIKey: "key1", Model: "sonnet"}}
	resolved := cfg.ResolveAI("telegram", "chan1")
	if resolved.Provider != "claude" || resolved.APIKey != "key1" {
		t.Errorf("expected base config, got provider=%s key=%s", resolved.Provider, resolved.APIKey)
	}
}

func TestResolveAI_PlatformOverride(t *testing.T) {
	cfg := &Config{AI: AIConfig{
		Provider: "claude",
		APIKey:   "key1",
		Overrides: []AIOverride{
			{Platform: "telegram", Provider: "deepseek", APIKey: "key2"},
		},
	}}
	resolved := cfg.ResolveAI("telegram", "any-channel")
	if resolved.Provider != "deepseek" || resolved.APIKey != "key2" {
		t.Errorf("expected override, got provider=%s key=%s", resolved.Provider, resolved.APIKey)
	}
}

func TestResolveAI_ChannelOverride(t *testing.T) {
	cfg := &Config{AI: AIConfig{
		Provider: "claude",
		APIKey:   "key1",
		Overrides: []AIOverride{
			{Platform: "telegram", APIKey: "platform-key"},
			{Platform: "telegram", ChannelID: "special", APIKey: "channel-key"},
		},
	}}
	resolved := cfg.ResolveAI("telegram", "special")
	if resolved.APIKey != "channel-key" {
		t.Errorf("channel override should win, got key=%s", resolved.APIKey)
	}
}

func TestResolveProvider_ByAgentID(t *testing.T) {
	cfg := &Config{
		Agents: []AgentEntry{
			{ID: "coder", Provider: "kimi", APIKey: "ak-xxx", Model: "kimi-k2.5"},
		},
	}
	e, ok := cfg.ResolveProvider("coder")
	if !ok || e.Provider != "kimi" || e.APIKey != "ak-xxx" {
		t.Errorf("expected match by agent ID, got ok=%v entry=%+v", ok, e)
	}
}

func TestResolveProvider_ByProviderType(t *testing.T) {
	cfg := &Config{
		Agents: []AgentEntry{
			{ID: "coder", Provider: "kimi", APIKey: "ak-xxx", Model: "kimi-k2.5"},
		},
	}
	e, ok := cfg.ResolveProvider("kimi")
	if !ok || e.APIKey != "ak-xxx" {
		t.Errorf("expected match by provider type, got ok=%v entry=%+v", ok, e)
	}
}

func TestResolveProvider_BackwardCompat(t *testing.T) {
	cfg := &Config{
		AI: AIConfig{Provider: "deepseek", APIKey: "sk-xxx", Model: "deepseek-chat"},
	}
	e, ok := cfg.ResolveProvider("deepseek")
	if !ok || e.APIKey != "sk-xxx" || e.Model != "deepseek-chat" {
		t.Errorf("expected backward compat from ai: block, got ok=%v entry=%+v", ok, e)
	}
}

func TestResolveProvider_EmptyName(t *testing.T) {
	cfg := &Config{
		Relay: RelayConfig{Provider: "kimi"},
		Agents: []AgentEntry{
			{ID: "default", Default: true, Provider: "kimi", APIKey: "ak-xxx"},
		},
	}
	e, ok := cfg.ResolveProvider("")
	if !ok || e.Provider != "kimi" {
		t.Errorf("expected fallback to relay.provider → agent, got ok=%v entry=%+v", ok, e)
	}
}

func TestResolveProvider_EmptyFallbackToAI(t *testing.T) {
	cfg := &Config{
		AI: AIConfig{Provider: "deepseek", APIKey: "sk-xxx"},
	}
	e, ok := cfg.ResolveProvider("")
	if !ok || e.Provider != "deepseek" {
		t.Errorf("expected fallback to ai.provider, got ok=%v entry=%+v", ok, e)
	}
}

func TestResolveProvider_DifferentProviderNoMap(t *testing.T) {
	cfg := &Config{
		AI: AIConfig{Provider: "deepseek", APIKey: "sk-xxx"},
	}
	e, ok := cfg.ResolveProvider("kimi")
	if !ok || e.Provider != "kimi" {
		t.Errorf("expected provider-only entry for CLI override, got ok=%v entry=%+v", ok, e)
	}
	if e.APIKey != "" {
		t.Errorf("expected empty api_key for different provider, got %s", e.APIKey)
	}
}

// TestResolveAgentAI_APIKeyOnlyAgent ensures that an agent with only api_key set
// (no provider) still returns the key so callers can use it without a provider.
// This guards against the bug where gateway/relay required ai.Provider != "" before
// reading the api_key, causing "AI API key is required" even when an agent was configured.
func TestResolveAgentAI_APIKeyOnlyAgent(t *testing.T) {
	cfg := &Config{
		Agents: []AgentEntry{
			{ID: "lingti", Default: true, APIKey: "sk-kimi-abc", Model: "kimi-k2.5"},
		},
	}
	defaultID := cfg.DefaultAgentID()
	if defaultID != "lingti" {
		t.Fatalf("expected default agent 'lingti', got %q", defaultID)
	}
	entry, ok := cfg.FindAgent(defaultID)
	if !ok {
		t.Fatal("agent 'lingti' not found")
	}
	ai := cfg.ResolveAgentAI(entry)
	if ai.APIKey != "sk-kimi-abc" {
		t.Errorf("expected api_key 'sk-kimi-abc', got %q", ai.APIKey)
	}
	if ai.Model != "kimi-k2.5" {
		t.Errorf("expected model 'kimi-k2.5', got %q", ai.Model)
	}
	// Provider should be empty (caller infers it from model name)
	if ai.Provider != "" {
		t.Errorf("expected empty provider, got %q", ai.Provider)
	}
}

func TestApplyOverride_ProviderChange(t *testing.T) {
	base := AIConfig{Provider: "claude", BaseURL: "https://api.anthropic.com", Model: "sonnet"}
	result := applyOverride(base, AIOverride{Provider: "deepseek", APIKey: "dk"})
	if result.Provider != "deepseek" {
		t.Errorf("expected deepseek, got %s", result.Provider)
	}
	if result.BaseURL != "" {
		t.Errorf("base_url should be cleared on provider change, got %s", result.BaseURL)
	}
	if result.Model != "" {
		t.Errorf("model should be cleared on provider change, got %s", result.Model)
	}
}
