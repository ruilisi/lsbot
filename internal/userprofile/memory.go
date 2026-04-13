package userprofile

import (
	"os"
	"path/filepath"
	"strings"

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

const memoryDelimiter = "§"
const maxMemoryChars = 2200

// AppendMemory adds a new entry to MEMORY.md (§-delimited), trimming oldest if over limit.
func AppendMemory(entry string) error {
	current := LoadMemory()
	var entries []string
	if current != "" {
		for _, e := range strings.Split(current, memoryDelimiter) {
			if t := strings.TrimSpace(e); t != "" {
				entries = append(entries, t)
			}
		}
	}
	entries = append(entries, strings.TrimSpace(entry))
	for {
		joined := strings.Join(entries, "\n"+memoryDelimiter+"\n")
		if len(joined) <= maxMemoryChars || len(entries) <= 1 {
			return WriteMemory(joined)
		}
		entries = entries[1:] // drop oldest
	}
}

// RemoveMemoryEntry removes the first entry containing the given substring.
func RemoveMemoryEntry(substring string) error {
	current := LoadMemory()
	if current == "" {
		return nil
	}
	var kept []string
	removed := false
	for _, e := range strings.Split(current, memoryDelimiter) {
		t := strings.TrimSpace(e)
		if t == "" {
			continue
		}
		if !removed && strings.Contains(t, substring) {
			removed = true
			continue
		}
		kept = append(kept, t)
	}
	return WriteMemory(strings.Join(kept, "\n"+memoryDelimiter+"\n"))
}
