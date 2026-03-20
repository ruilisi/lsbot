package cmd

import "testing"

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
