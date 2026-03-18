package clawhub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/ruilisi/lsbot/internal/config"
)

type LockFile struct {
	Skills map[string]LockEntry `json:"skills"`
}

type LockEntry struct {
	Slug        string    `json:"slug"`
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
}

func lockFilePath() string {
	return filepath.Join(config.HubDir(), "hub-lock.json")
}

func LoadLock() (*LockFile, error) {
	data, err := os.ReadFile(lockFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &LockFile{Skills: make(map[string]LockEntry)}, nil
		}
		return nil, err
	}

	var lf LockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, err
	}
	if lf.Skills == nil {
		lf.Skills = make(map[string]LockEntry)
	}
	return &lf, nil
}

func (l *LockFile) Save() error {
	if err := os.MkdirAll(config.HubDir(), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}

	path := lockFilePath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (l *LockFile) Add(slug, version string) {
	l.Skills[slug] = LockEntry{
		Slug:        slug,
		Version:     version,
		InstalledAt: time.Now(),
	}
}

func (l *LockFile) Remove(slug string) {
	delete(l.Skills, slug)
}

func (l *LockFile) Get(slug string) (LockEntry, bool) {
	e, ok := l.Skills[slug]
	return e, ok
}
