package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/ruilisi/lsbot/internal/router"
)

func TestCreateProvider_ValidProviders(t *testing.T) {
	providers := []string{"claude", "deepseek", "kimi", "moonshot", "qwen", "qianwen", "tongyi", "openai", "zhipu", "gemini"}
	for _, p := range providers {
		_, err := createProvider(Config{Provider: p, APIKey: "test-key"})
		if err != nil {
			t.Errorf("createProvider(%q) failed: %v", p, err)
		}
	}
}

func TestCreateProvider_Aliases(t *testing.T) {
	aliases := []string{"glm", "gpt", "chatgpt", "google", "xai"}
	for _, alias := range aliases {
		_, err := createProvider(Config{Provider: alias, APIKey: "test-key"})
		if err != nil {
			t.Errorf("createProvider(%q) failed: %v", alias, err)
		}
	}
}

func TestCreateProvider_EmptyDefaultsClaude(t *testing.T) {
	p, err := createProvider(Config{Provider: "", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("empty provider should default to claude: %v", err)
	}
	if p.Name() != "claude" {
		t.Errorf("expected claude, got %s", p.Name())
	}
}

// TestCreateProvider_InferFromModel ensures that when provider is empty but a
// model name is set, the correct provider is inferred and used. This guards
// against the regression where e.g. model=kimi-k2.5 with no provider would
// send requests to Anthropic instead of Kimi.
func TestCreateProvider_InferFromModel(t *testing.T) {
	cases := []struct {
		model    string
		wantName string
	}{
		{"kimi-k2.5", "kimi"},
		{"moonshot-v1-8k", "kimi"},
		{"deepseek-chat", "deepseek"},
		{"deepseek-r1", "deepseek"},
		{"qwen-plus", "qwen"},
		{"glm-4-flash", "zhipu"},
		{"gpt-4o", "openai"},
		{"gemini-2.0-flash", "gemini"},
		{"claude-sonnet-4-20250514", "claude"},
		{"grok-2-latest", "grok"},
	}
	for _, tc := range cases {
		p, err := createProvider(Config{Provider: "", APIKey: "test-key", Model: tc.model})
		if err != nil {
			t.Errorf("model=%q: createProvider failed: %v", tc.model, err)
			continue
		}
		if p.Name() != tc.wantName {
			t.Errorf("model=%q: expected provider %q, got %q", tc.model, tc.wantName, p.Name())
		}
	}
}

func TestCreateProvider_Unknown(t *testing.T) {
	_, err := createProvider(Config{Provider: "nonexistent", APIKey: "test-key"})
	if err == nil {
		t.Error("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateProvider_OllamaNoKey(t *testing.T) {
	_, err := createProvider(Config{Provider: "ollama"})
	if err != nil {
		t.Errorf("ollama should not require API key: %v", err)
	}
}

func TestHandleBuiltinCommand(t *testing.T) {
	agent, err := New(Config{Provider: "claude", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	tests := []struct {
		text    string
		handled bool
	}{
		{"/help", true},
		{"/new", true},
		{"/status", true},
		{"/whoami", true},
		{"/model", true},
		{"/tools", true},
		{"/think high", true},
		{"/verbose on", true},
		{"hello world", false},
	}
	for _, tt := range tests {
		msg := router.Message{Text: tt.text, Platform: "test", ChannelID: "c1", UserID: "u1", Username: "tester"}
		_, handled := agent.handleBuiltinCommand(context.Background(), msg)
		if handled != tt.handled {
			t.Errorf("handleBuiltinCommand(%q): got handled=%v, want %v", tt.text, handled, tt.handled)
		}
	}
}
