package userprofile

import (
	"os"
	"path/filepath"

	"github.com/ruilisi/lsbot/internal/config"
)

func MemoryPath() string {
	return filepath.Join(config.HubDir(), "memory", "MEMORY.md")
}

// LoadMemory returns the content of MEMORY.md; empty string if not found.
func LoadMemory() string {
	data, err := os.ReadFile(MemoryPath())
	if err != nil {
		return ""
	}
	return string(data)
}

// WriteMemory replaces MEMORY.md with the given content (creates dirs as needed).
func WriteMemory(content string) error {
	dir := filepath.Dir(MemoryPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(MemoryPath(), []byte(content), 0644)
}
