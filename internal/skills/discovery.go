package skills

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ruilisi/lsbot/internal/config"
	"gopkg.in/yaml.v3"
)

// SkillSource represents where a skill was loaded from
type SkillSource string

const (
	SourceBundled   SkillSource = "bundled"
	SourceManaged   SkillSource = "managed"
	SourceHub       SkillSource = "hub"
	SourceWorkspace SkillSource = "workspace"
	SourceExtra     SkillSource = "extra"
)

// SkillEntry is a discovered skill with parsed metadata
type SkillEntry struct {
	Name        string        `json:"name" yaml:"name"`
	Description string        `json:"description" yaml:"description"`
	Homepage    string        `json:"homepage,omitempty" yaml:"homepage,omitempty"`
	FilePath    string        `json:"file_path" yaml:"-"`
	BaseDir     string        `json:"base_dir" yaml:"-"`
	Source      SkillSource   `json:"source" yaml:"-"`
	Content     string        `json:"-" yaml:"-"` // Markdown body after frontmatter
	Metadata    SkillMetadata `json:"metadata" yaml:"metadata"`
	Enabled     bool          `json:"enabled" yaml:"-"`
}

// SkillMetadata holds gating and display metadata
type SkillMetadata struct {
	Emoji    string        `json:"emoji,omitempty" yaml:"emoji,omitempty"`
	OS       []string      `json:"os,omitempty" yaml:"os,omitempty"`
	Always   bool          `json:"always,omitempty" yaml:"always,omitempty"`
	Default  bool          `json:"default,omitempty" yaml:"default,omitempty"`
	Requires Requirements  `json:"requires,omitempty" yaml:"requires,omitempty"`
	Install  []InstallSpec `json:"install,omitempty" yaml:"install,omitempty"`
}

// Requirements defines what a skill needs to be eligible
type Requirements struct {
	Bins    []string `json:"bins,omitempty" yaml:"bins,omitempty"`
	AnyBins []string `json:"any_bins,omitempty" yaml:"any_bins,omitempty"`
	Env     []string `json:"env,omitempty" yaml:"env,omitempty"`
}

// InstallSpec describes how to install a missing requirement
type InstallSpec struct {
	ID      string   `json:"id,omitempty" yaml:"id,omitempty"`
	Kind    string   `json:"kind" yaml:"kind"` // brew, apt, go, npm, download
	Formula string   `json:"formula,omitempty" yaml:"formula,omitempty"`
	Package string   `json:"package,omitempty" yaml:"package,omitempty"`
	Module  string   `json:"module,omitempty" yaml:"module,omitempty"`
	URL     string   `json:"url,omitempty" yaml:"url,omitempty"`
	Label   string   `json:"label,omitempty" yaml:"label,omitempty"`
	Bins    []string `json:"bins,omitempty" yaml:"bins,omitempty"`
}

// skillFrontmatter is the raw YAML structure in SKILL.md frontmatter.
// Supports both our flat format and openclaw's nested {"openclaw": {...}} format.
type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Homepage    string `yaml:"homepage,omitempty"`
	Metadata    any    `yaml:"metadata,omitempty"` // Can be SkillMetadata or {"openclaw": SkillMetadata}
}

// ParseSkillMD parses a SKILL.md file into a SkillEntry
func ParseSkillMD(path string) (*SkillEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	frontmatter, body, err := splitFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter in %s: %w", path, err)
	}

	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatter), &fm); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", path, err)
	}

	if fm.Name == "" {
		return nil, fmt.Errorf("skill in %s is missing required 'name' field", path)
	}

	metadata := resolveMetadata(fm.Metadata)

	entry := &SkillEntry{
		Name:        fm.Name,
		Description: fm.Description,
		Homepage:    fm.Homepage,
		FilePath:    path,
		BaseDir:     filepath.Dir(path),
		Content:     body,
		Metadata:    metadata,
		Enabled:     true,
	}

	return entry, nil
}

// resolveMetadata handles both flat metadata and openclaw's nested {"openclaw": {...}} format.
func resolveMetadata(raw any) SkillMetadata {
	if raw == nil {
		return SkillMetadata{}
	}

	// Re-marshal and try to unmarshal as our SkillMetadata directly
	data, err := yaml.Marshal(raw)
	if err != nil {
		return SkillMetadata{}
	}

	// Try openclaw format: {"openclaw": {...}}
	var openclawWrapper struct {
		OpenClaw SkillMetadata `yaml:"openclaw"`
	}
	if err := yaml.Unmarshal(data, &openclawWrapper); err == nil && openclawWrapper.OpenClaw.Emoji != "" {
		return openclawWrapper.OpenClaw
	}

	// Try flat format
	var meta SkillMetadata
	if err := yaml.Unmarshal(data, &meta); err == nil {
		return meta
	}

	return SkillMetadata{}
}

// splitFrontmatter splits a document into YAML frontmatter and markdown body.
// Frontmatter is delimited by "---" lines.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(content))

	// Find opening ---
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			break
		}
		// If we hit non-empty content before ---, there's no frontmatter
		if line != "" {
			return "", content, nil
		}
	}

	// Collect frontmatter lines until closing ---
	var fmLines []string
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			found = true
			break
		}
		fmLines = append(fmLines, line)
	}

	if !found {
		return "", content, fmt.Errorf("no closing --- found for frontmatter")
	}

	frontmatter = strings.Join(fmLines, "\n")

	// Rest is body
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}
	body = strings.Join(bodyLines, "\n")

	return frontmatter, strings.TrimSpace(body), nil
}

// DiscoverSkills loads skills from all sources in precedence order.
// Later sources override earlier ones (workspace > managed > bundled).
func DiscoverSkills(disabledList []string, extraDirs []string) []SkillEntry {
	disabled := make(map[string]bool, len(disabledList))
	for _, name := range disabledList {
		disabled[name] = true
	}

	skills := make(map[string]SkillEntry)

	// 1. Bundled skills (lowest precedence)
	if dir := resolveBundledSkillsDir(); dir != "" {
		loadSkillsFromDir(dir, SourceBundled, disabled, skills)
	}

	// 2. Extra directories
	for _, dir := range extraDirs {
		loadSkillsFromDir(dir, SourceExtra, disabled, skills)
	}

	// 3. Managed skills (~/.lsbot/skills/)
	managedDir := managedSkillsDir()
	loadSkillsFromDir(managedDir, SourceManaged, disabled, skills)

	// 4. Hub skills (~/.lsbot/skills/)
	hubDir := config.HubSkillsDir()
	loadSkillsFromDir(hubDir, SourceHub, disabled, skills)

	// 5. Workspace skills (highest precedence)
	if cwd, err := os.Getwd(); err == nil {
		workspaceDir := filepath.Join(cwd, "skills")
		loadSkillsFromDir(workspaceDir, SourceWorkspace, disabled, skills)
	}

	// Convert map to sorted slice
	result := make([]SkillEntry, 0, len(skills))
	for _, entry := range skills {
		result = append(result, entry)
	}

	// Sort by name for stable output
	sortSkillEntries(result)

	return result
}

// loadSkillsFromDir scans a directory for SKILL.md files (one level deep)
// and also loads legacy JSON files for backward compatibility.
func loadSkillsFromDir(dir string, source SkillSource, disabled map[string]bool, skills map[string]SkillEntry) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // Directory doesn't exist or can't be read — skip silently
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Look for <dir>/<name>/SKILL.md
			skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				skill, err := ParseSkillMD(skillPath)
				if err != nil {
					continue
				}
				skill.Source = source
				if disabled[skill.Name] {
					skill.Enabled = false
				}
				skills[skill.Name] = *skill
			}
		}
	}
}

// resolveBundledSkillsDir finds the bundled skills directory relative to the executable.
func resolveBundledSkillsDir() string {
	// Check environment variable override
	if dir := os.Getenv("LINGTI_BUNDLED_SKILLS_DIR"); dir != "" {
		if looksLikeSkillsDir(dir) {
			return dir
		}
	}

	// Try relative to executable
	execPath, err := os.Executable()
	if err == nil {
		execPath, _ = filepath.EvalSymlinks(execPath)
		execDir := filepath.Dir(execPath)

		// <exec>/../skills/
		candidate := filepath.Join(execDir, "..", "skills")
		if looksLikeSkillsDir(candidate) {
			return candidate
		}

		// <exec>/skills/ (dev mode)
		candidate = filepath.Join(execDir, "skills")
		if looksLikeSkillsDir(candidate) {
			return candidate
		}
	}

	// Try relative to working directory (dev mode)
	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, "bundled-skills")
		if looksLikeSkillsDir(candidate) {
			return candidate
		}
	}

	return ""
}

// looksLikeSkillsDir checks if a directory contains skill files
func looksLikeSkillsDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				return true
			}
		}
	}
	return false
}

// managedSkillsDir returns the user-level skills directory
func managedSkillsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lsbot", "skills")
}

// HasBinary checks if a binary exists in PATH
func HasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// RuntimeOS returns the current OS in the format used by skill metadata
func RuntimeOS() string {
	return runtime.GOOS
}

// ShortenHomePath replaces the home directory with ~ for display
func ShortenHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// sortSkillEntries sorts skills by name
func sortSkillEntries(entries []SkillEntry) {
	for i := range len(entries) {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Name > entries[j].Name {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}
