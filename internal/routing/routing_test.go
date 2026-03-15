package routing

import (
	"testing"

	"github.com/ruilisi/lsbot/internal/config"
)

func TestResolveRoute_NoBindings(t *testing.T) {
	cfg := &config.Config{}
	r := ResolveRoute(cfg, "slack", "C123", "U456")
	if r.AgentID != "" {
		t.Errorf("expected empty agent ID, got %q", r.AgentID)
	}
}

func TestResolveRoute_PlatformOnly(t *testing.T) {
	cfg := &config.Config{
		Bindings: []config.AgentBinding{
			{AgentID: "support", Match: config.AgentBindingMatch{Platform: "slack"}},
		},
	}
	r := ResolveRoute(cfg, "slack", "C123", "U456")
	if r.AgentID != "support" {
		t.Errorf("expected support, got %q", r.AgentID)
	}
	// Different platform should not match
	r2 := ResolveRoute(cfg, "telegram", "C123", "U456")
	if r2.AgentID != "" {
		t.Errorf("expected no match for telegram, got %q", r2.AgentID)
	}
}

func TestResolveRoute_SpecificityWins(t *testing.T) {
	cfg := &config.Config{
		Bindings: []config.AgentBinding{
			{AgentID: "general", Match: config.AgentBindingMatch{Platform: "slack"}},
			{AgentID: "vip", Match: config.AgentBindingMatch{Platform: "slack", UserID: "U_VIP"}},
			{AgentID: "channel-bot", Match: config.AgentBindingMatch{Platform: "slack", ChannelID: "C_SALES"}},
			{AgentID: "exact", Match: config.AgentBindingMatch{Platform: "slack", ChannelID: "C_SALES", UserID: "U_VIP"}},
		},
	}

	// Most specific: platform+channel+user
	r := ResolveRoute(cfg, "slack", "C_SALES", "U_VIP")
	if r.AgentID != "exact" {
		t.Errorf("expected exact, got %q", r.AgentID)
	}

	// platform+user beats platform+channel and platform-only
	r = ResolveRoute(cfg, "slack", "C_OTHER", "U_VIP")
	if r.AgentID != "vip" {
		t.Errorf("expected vip, got %q", r.AgentID)
	}

	// platform+channel beats platform-only
	r = ResolveRoute(cfg, "slack", "C_SALES", "U_ANON")
	if r.AgentID != "channel-bot" {
		t.Errorf("expected channel-bot, got %q", r.AgentID)
	}

	// platform-only fallback
	r = ResolveRoute(cfg, "slack", "C_OTHER", "U_ANON")
	if r.AgentID != "general" {
		t.Errorf("expected general, got %q", r.AgentID)
	}
}

func TestResolveRoute_WildcardBinding(t *testing.T) {
	// Empty match fields act as wildcard
	cfg := &config.Config{
		Bindings: []config.AgentBinding{
			{AgentID: "catchall", Match: config.AgentBindingMatch{}},
		},
	}
	r := ResolveRoute(cfg, "telegram", "any", "anyone")
	if r.AgentID != "catchall" {
		t.Errorf("expected catchall, got %q", r.AgentID)
	}
}
