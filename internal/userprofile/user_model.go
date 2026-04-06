package userprofile

import (
	"os"
	"path/filepath"

	"github.com/ruilisi/lsbot/internal/config"
)

// UserModelPath returns the path to USER.md – the agent's evolving model of the user.
// This is distinct from MEMORY.md (general notes) and profile.yaml (structured data).
// USER.md captures personality, communication style, preferences, and workflow habits.
func UserModelPath() string {
	return filepath.Join(config.HubDir(), "memory", "USER.md")
}

// LoadUserModel returns the content of USER.md; empty string if not found.
func LoadUserModel() string {
	data, err := os.ReadFile(UserModelPath())
	if err != nil {
		return ""
	}
	return string(data)
}

// WriteUserModel replaces USER.md with the given content (creates dirs as needed).
func WriteUserModel(content string) error {
	dir := filepath.Dir(UserModelPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(UserModelPath(), []byte(content), 0644)
}
