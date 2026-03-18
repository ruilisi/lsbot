package clawhub

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ruilisi/lsbot/internal/config"
)

// Install downloads a skill from ClawHub and extracts it to ~/.lsbot/skills/<slug>/
func Install(ctx context.Context, client *Client, slug, version string) (string, error) {
	// 1. Determine installed version
	installedVersion := version

	// 2. Download ZIP
	rc, err := client.Download(ctx, slug, version)
	if err != nil {
		return "", fmt.Errorf("download error: %w", err)
	}
	defer rc.Close()

	// 3. Buffer to temp file (zip.NewReader needs io.ReaderAt + size)
	tmp, err := os.CreateTemp("", "clawhub-*.zip")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := io.Copy(tmp, rc); err != nil {
		return "", fmt.Errorf("download write error: %w", err)
	}

	size, err := tmp.Seek(0, io.SeekEnd)
	if err != nil {
		return "", err
	}

	// 4. Extract ZIP
	zr, err := zip.NewReader(tmp, size)
	if err != nil {
		return "", fmt.Errorf("zip open error: %w", err)
	}

	destDir := filepath.Join(config.HubSkillsDir(), slug)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	// Detect top-level directory to strip (common in GitHub archives)
	topDir := detectTopDir(zr.File)

	for _, f := range zr.File {
		relPath := f.Name
		if topDir != "" {
			relPath = strings.TrimPrefix(relPath, topDir)
		}
		if relPath == "" || relPath == "/" {
			continue
		}

		destPath := filepath.Join(destDir, relPath)
		// Security: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(destDir)) {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return "", err
			}
			continue
		}

		if err := extractFile(f, destPath); err != nil {
			return "", err
		}
	}

	// 5. If version was unknown, try to resolve from API
	if installedVersion == "" {
		detail, err := client.GetSkill(ctx, slug)
		if err == nil {
			installedVersion = detail.LatestVersion.Version
		}
	}

	// 6. Update lock file
	lock, err := LoadLock()
	if err != nil {
		return "", err
	}
	lock.Add(slug, installedVersion)
	if err := lock.Save(); err != nil {
		return "", err
	}

	return installedVersion, nil
}

// Remove deletes ~/.lsbot/skills/<slug>/ and removes from lock file
func Remove(slug string) error {
	destDir := filepath.Join(config.HubSkillsDir(), slug)
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("remove directory error: %w", err)
	}

	lock, err := LoadLock()
	if err != nil {
		return err
	}
	lock.Remove(slug)
	return lock.Save()
}

func detectTopDir(files []*zip.File) string {
	if len(files) == 0 {
		return ""
	}
	// Check if every file shares the same top-level directory
	first := files[0].Name
	parts := strings.SplitN(first, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	prefix := parts[0] + "/"
	for _, f := range files {
		if !strings.HasPrefix(f.Name, prefix) {
			return ""
		}
	}
	return prefix
}

func extractFile(f *zip.File, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
