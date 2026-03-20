package userprofile

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ruilisi/lsbot/internal/config"
	"gopkg.in/yaml.v3"
)

type UserProfile struct {
	Nickname  string    `yaml:"nickname,omitempty"`
	Timezone  string    `yaml:"timezone,omitempty"`
	CreatedAt time.Time `yaml:"created_at,omitempty"`
}

func profilePath() string {
	return filepath.Join(config.HubDir(), "profile.yaml")
}

// Load reads ~/.lsbot/profile.yaml; returns empty profile if not found.
func Load() (*UserProfile, error) {
	data, err := os.ReadFile(profilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &UserProfile{}, nil
		}
		return nil, err
	}
	var p UserProfile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Save writes the profile atomically to ~/.lsbot/profile.yaml
func (p *UserProfile) Save() error {
	if err := os.MkdirAll(config.HubDir(), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	tmp := profilePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, profilePath())
}

// IsOnboarded returns true if the user has completed the onboarding flow.
func (p *UserProfile) IsOnboarded() bool {
	return p.Nickname != ""
}
