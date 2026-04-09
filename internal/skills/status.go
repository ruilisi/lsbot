package skills

import (
	"os"
	"slices"
)

// EligibilityStatus represents whether a skill is ready to use
type EligibilityStatus string

const (
	StatusReady    EligibilityStatus = "ready"
	StatusMissing  EligibilityStatus = "missing"
	StatusDisabled EligibilityStatus = "disabled"
)

// MissingRequirements tracks what a skill is missing
type MissingRequirements struct {
	Bins    []string `json:"bins,omitempty"`
	AnyBins []string `json:"any_bins,omitempty"`
	Env     []string `json:"env,omitempty"`
	OS      []string `json:"os,omitempty"`
}

// SkillStatus is the full eligibility report for one skill
type SkillStatus struct {
	SkillEntry
	Status  EligibilityStatus   `json:"status"`
	Missing MissingRequirements `json:"missing"`
}

// StatusReport is the full report for all discovered skills
type StatusReport struct {
	Skills       []SkillStatus `json:"skills"`
	BundledDir   string        `json:"bundled_dir"`
	ManagedDir   string        `json:"managed_dir"`
	WorkspaceDir string        `json:"workspace_dir"`
}

// BuildStatusReport discovers all skills and checks eligibility for each.
func BuildStatusReport(disabledList []string, extraDirs []string) StatusReport {
	entries := DiscoverSkills(disabledList, extraDirs)

	statuses := make([]SkillStatus, 0, len(entries))
	for _, entry := range entries {
		statuses = append(statuses, checkEligibility(entry))
	}

	workspaceDir := ""
	if cwd, err := os.Getwd(); err == nil {
		workspaceDir = cwd + "/skills"
	}

	return StatusReport{
		Skills:       statuses,
		BundledDir:   resolveBundledSkillsDir(),
		ManagedDir:   managedSkillsDir(),
		WorkspaceDir: workspaceDir,
	}
}

// checkEligibility evaluates all gating rules for a skill entry.
func checkEligibility(entry SkillEntry) SkillStatus {
	status := SkillStatus{
		SkillEntry: entry,
		Status:     StatusReady,
	}

	// Gate 1: explicitly disabled
	if !entry.Enabled {
		status.Status = StatusDisabled
		return status
	}

	// Gate 2: OS requirement
	if len(entry.Metadata.OS) > 0 {
		if !slices.Contains(entry.Metadata.OS, RuntimeOS()) {
			status.Status = StatusMissing
			status.Missing.OS = entry.Metadata.OS
			return status
		}
	}

	// Gate 3: always or default flag — skip remaining gates
	if entry.Metadata.Always || entry.Metadata.Default {
		status.Status = StatusReady
		return status
	}

	// Gate 4: required binaries (all must exist)
	for _, bin := range entry.Metadata.Requires.Bins {
		if !HasBinary(bin) {
			status.Missing.Bins = append(status.Missing.Bins, bin)
		}
	}

	// Gate 5: any binaries (at least one must exist)
	if len(entry.Metadata.Requires.AnyBins) > 0 {
		if !slices.ContainsFunc(entry.Metadata.Requires.AnyBins, HasBinary) {
			status.Missing.AnyBins = entry.Metadata.Requires.AnyBins
		}
	}

	// Gate 6: required environment variables
	for _, envVar := range entry.Metadata.Requires.Env {
		if os.Getenv(envVar) == "" {
			status.Missing.Env = append(status.Missing.Env, envVar)
		}
	}

	// If anything is missing, mark as missing
	if len(status.Missing.Bins) > 0 || len(status.Missing.AnyBins) > 0 ||
		len(status.Missing.Env) > 0 || len(status.Missing.OS) > 0 {
		status.Status = StatusMissing
	}

	return status
}

// CountByStatus returns counts of eligible, disabled, and missing skills
func (r *StatusReport) CountByStatus() (eligible, disabled, missing int) {
	for _, s := range r.Skills {
		switch s.Status {
		case StatusReady:
			eligible++
		case StatusDisabled:
			disabled++
		case StatusMissing:
			missing++
		}
	}
	return
}

// EligibleSkills returns only skills that are ready to use
func (r *StatusReport) EligibleSkills() []SkillStatus {
	var result []SkillStatus
	for _, s := range r.Skills {
		if s.Status == StatusReady {
			result = append(result, s)
		}
	}
	return result
}

// MissingSkills returns skills that have unmet requirements
func (r *StatusReport) MissingSkills() []SkillStatus {
	var result []SkillStatus
	for _, s := range r.Skills {
		if s.Status == StatusMissing {
			result = append(result, s)
		}
	}
	return result
}

// DisabledSkills returns explicitly disabled skills
func (r *StatusReport) DisabledSkills() []SkillStatus {
	var result []SkillStatus
	for _, s := range r.Skills {
		if s.Status == StatusDisabled {
			result = append(result, s)
		}
	}
	return result
}
