package skills

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// GitHub tarball URL for the default branch
	repoTarballURL = "https://github.com/ruilisi/lsbot/archive/refs/heads/master.tar.gz"
	// Prefix inside the tarball for bundled skills
	bundledSkillsPrefix = "bundled-skills/"
)

// DownloadBundledSkills downloads bundled skills from GitHub into the managed skills directory.
// It overwrites existing skills with the same name.
// progress is called with each skill name as it is installed (may be nil).
func DownloadBundledSkills(destDir string, progress func(skill string)) (int, error) {
	if destDir == "" {
		destDir = managedSkillsDir()
	}

	resp, err := http.Get(repoTarballURL)
	if err != nil {
		return 0, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to decompress: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	count := 0
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, fmt.Errorf("tar read error: %w", err)
		}

		// Tarball entries look like: lsbot-master/bundled-skills/weather/SKILL.md
		// Strip the repo prefix (first path component) to get: bundled-skills/weather/SKILL.md
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 {
			continue
		}
		relPath := parts[1]

		// Only extract bundled-skills/
		if !strings.HasPrefix(relPath, bundledSkillsPrefix) {
			continue
		}

		// Map bundled-skills/weather/SKILL.md -> <destDir>/weather/SKILL.md
		skillRelPath := strings.TrimPrefix(relPath, bundledSkillsPrefix)
		if skillRelPath == "" {
			continue
		}

		target := filepath.Join(destDir, skillRelPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return count, fmt.Errorf("failed to create dir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return count, fmt.Errorf("failed to create parent dir for %s: %w", target, err)
			}
			f, err := os.Create(target)
			if err != nil {
				return count, fmt.Errorf("failed to create file %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return count, fmt.Errorf("failed to write %s: %w", target, err)
			}
			f.Close()

			if filepath.Base(target) == "SKILL.md" {
				count++
				if progress != nil {
					skillName := filepath.Base(filepath.Dir(target))
					progress(skillName)
				}
			}
		}
	}

	return count, nil
}
