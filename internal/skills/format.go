package skills

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
	colorCyan   = "\033[36m"
)

// FormatListOptions controls the list output
type FormatListOptions struct {
	JSON     bool
	Eligible bool
	Verbose  bool
}

// FormatList formats the skills list for terminal output
func FormatList(report StatusReport, opts FormatListOptions) string {
	skills := report.Skills
	if opts.Eligible {
		skills = report.EligibleSkills()
	}

	if opts.JSON {
		data, _ := json.MarshalIndent(report, "", "  ")
		return string(data)
	}

	if len(skills) == 0 {
		if opts.Eligible {
			return "No eligible skills found. Run `lsbot skills list` to see all skills."
		}
		return "No skills found.\n\nSkills are loaded from:\n" +
			fmt.Sprintf("  Managed:   %s\n", ShortenHomePath(report.ManagedDir)) +
			fmt.Sprintf("  Workspace: %s\n", ShortenHomePath(report.WorkspaceDir)) +
			"\nRun `lsbot skills download` to download bundled skills from GitHub."
	}

	eligible, _, _ := report.CountByStatus()

	var b strings.Builder

	fmt.Fprintf(&b, "%sSkills%s %s(%d/%d ready)%s\n\n",
		colorBold, colorReset, colorGray, eligible, len(skills), colorReset)

	// Column widths
	statusW := 12
	nameW := 20
	descW := 36
	sourceW := 10

	// Header
	fmt.Fprintf(&b, "  %-*s %-*s %-*s %-*s",
		statusW, "Status", nameW, "Skill", descW, "Description", sourceW, "Source")
	if opts.Verbose {
		b.WriteString("  Missing")
	}
	b.WriteString("\n")

	for _, skill := range skills {
		status := formatStatus(skill.Status)
		name := formatSkillName(skill.SkillEntry)
		desc := truncate(skill.Description, descW)
		source := string(skill.Source)

		fmt.Fprintf(&b, "  %-*s %-*s %s%-*s%s %-*s",
			statusW+colorLen(status), status,
			nameW+colorLen(name), name,
			colorGray, descW, desc, colorReset,
			sourceW, source)

		if opts.Verbose {
			missing := formatMissingSummary(skill.Missing)
			if missing != "" {
				fmt.Fprintf(&b, "  %s%s%s", colorYellow, missing, colorReset)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// FormatInfo formats detailed info for a single skill
func FormatInfo(report StatusReport, name string, asJSON bool) string {
	var skill *SkillStatus
	for i := range report.Skills {
		if report.Skills[i].Name == name {
			skill = &report.Skills[i]
			break
		}
	}

	if skill == nil {
		if asJSON {
			data, _ := json.MarshalIndent(map[string]string{"error": "not found", "skill": name}, "", "  ")
			return string(data)
		}
		return fmt.Sprintf("Skill %q not found. Run `lsbot skills list` to see available skills.", name)
	}

	if asJSON {
		data, _ := json.MarshalIndent(skill, "", "  ")
		return string(data)
	}

	var b strings.Builder

	emoji := skill.Metadata.Emoji
	if emoji == "" {
		emoji = "📦"
	}

	statusStr := ""
	switch skill.Status {
	case StatusReady:
		statusStr = colorGreen + "✓ Ready" + colorReset
	case StatusDisabled:
		statusStr = colorYellow + "⏸ Disabled" + colorReset
	case StatusMissing:
		statusStr = colorRed + "✗ Missing requirements" + colorReset
	}

	fmt.Fprintf(&b, "%s %s%s%s %s\n\n", emoji, colorBold, skill.Name, colorReset, statusStr)
	b.WriteString(skill.Description + "\n\n")

	// Details
	b.WriteString(colorBold + "Details:" + colorReset + "\n")
	fmt.Fprintf(&b, "  %sSource:%s   %s\n", colorGray, colorReset, skill.Source)
	fmt.Fprintf(&b, "  %sPath:%s     %s\n", colorGray, colorReset, ShortenHomePath(skill.FilePath))
	if skill.Homepage != "" {
		fmt.Fprintf(&b, "  %sHomepage:%s %s\n", colorGray, colorReset, skill.Homepage)
	}

	// Requirements
	hasReqs := len(skill.Metadata.Requires.Bins) > 0 ||
		len(skill.Metadata.Requires.AnyBins) > 0 ||
		len(skill.Metadata.Requires.Env) > 0 ||
		len(skill.Metadata.OS) > 0

	if hasReqs {
		b.WriteString("\n" + colorBold + "Requirements:" + colorReset + "\n")

		if len(skill.Metadata.Requires.Bins) > 0 {
			var items []string
			for _, bin := range skill.Metadata.Requires.Bins {
				if slices.Contains(skill.Missing.Bins, bin) {
					items = append(items, colorRed+"✗ "+bin+colorReset)
				} else {
					items = append(items, colorGreen+"✓ "+bin+colorReset)
				}
			}
			fmt.Fprintf(&b, "  %sBinaries:%s  %s\n", colorGray, colorReset, strings.Join(items, ", "))
		}

		if len(skill.Metadata.Requires.AnyBins) > 0 {
			var items []string
			anyMissing := len(skill.Missing.AnyBins) > 0
			for _, bin := range skill.Metadata.Requires.AnyBins {
				if anyMissing {
					items = append(items, colorRed+"✗ "+bin+colorReset)
				} else {
					items = append(items, colorGreen+"✓ "+bin+colorReset)
				}
			}
			fmt.Fprintf(&b, "  %sAny of:%s    %s\n", colorGray, colorReset, strings.Join(items, ", "))
		}

		if len(skill.Metadata.Requires.Env) > 0 {
			var items []string
			for _, env := range skill.Metadata.Requires.Env {
				if slices.Contains(skill.Missing.Env, env) {
					items = append(items, colorRed+"✗ "+env+colorReset)
				} else {
					items = append(items, colorGreen+"✓ "+env+colorReset)
				}
			}
			fmt.Fprintf(&b, "  %sEnv vars:%s  %s\n", colorGray, colorReset, strings.Join(items, ", "))
		}

		if len(skill.Metadata.OS) > 0 {
			var items []string
			for _, osName := range skill.Metadata.OS {
				if slices.Contains(skill.Missing.OS, osName) {
					items = append(items, colorRed+"✗ "+osName+colorReset)
				} else {
					items = append(items, colorGreen+"✓ "+osName+colorReset)
				}
			}
			fmt.Fprintf(&b, "  %sOS:%s        %s\n", colorGray, colorReset, strings.Join(items, ", "))
		}
	}

	// Install options (only if missing)
	if skill.Status == StatusMissing && len(skill.Metadata.Install) > 0 {
		b.WriteString("\n" + colorBold + "Install options:" + colorReset + "\n")
		for _, inst := range skill.Metadata.Install {
			label := inst.Label
			if label == "" {
				label = fmt.Sprintf("%s (%s)", inst.Kind, inst.ID)
			}
			fmt.Fprintf(&b, "  %s→%s %s\n", colorYellow, colorReset, label)
		}
	}

	return b.String()
}

// FormatCheck formats a summary check of all skills
func FormatCheck(report StatusReport, asJSON bool) string {
	eligible, disabled, missing := report.CountByStatus()

	if asJSON {
		result := map[string]any{
			"summary": map[string]int{
				"total":                len(report.Skills),
				"eligible":             eligible,
				"disabled":             disabled,
				"missing_requirements": missing,
			},
			"eligible":             skillNames(report.EligibleSkills()),
			"disabled":             skillNames(report.DisabledSkills()),
			"missing_requirements": report.MissingSkills(),
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		return string(data)
	}

	var b strings.Builder

	b.WriteString(colorBold + "Skills Status Check" + colorReset + "\n\n")
	fmt.Fprintf(&b, "%sTotal:%s                  %d\n", colorGray, colorReset, len(report.Skills))
	fmt.Fprintf(&b, "%s✓%s %sEligible:%s             %d\n", colorGreen, colorReset, colorGray, colorReset, eligible)
	fmt.Fprintf(&b, "%s⏸%s %sDisabled:%s             %d\n", colorYellow, colorReset, colorGray, colorReset, disabled)
	fmt.Fprintf(&b, "%s✗%s %sMissing requirements:%s %d\n", colorRed, colorReset, colorGray, colorReset, missing)

	if eligible > 0 {
		b.WriteString("\n" + colorBold + "Ready to use:" + colorReset + "\n")
		for _, skill := range report.EligibleSkills() {
			emoji := skill.Metadata.Emoji
			if emoji == "" {
				emoji = "📦"
			}
			fmt.Fprintf(&b, "  %s %s\n", emoji, skill.Name)
		}
	}

	if missing > 0 {
		b.WriteString("\n" + colorBold + "Missing requirements:" + colorReset + "\n")
		for _, skill := range report.MissingSkills() {
			emoji := skill.Metadata.Emoji
			if emoji == "" {
				emoji = "📦"
			}
			summary := formatMissingSummary(skill.Missing)
			fmt.Fprintf(&b, "  %s %s %s(%s)%s\n", emoji, skill.Name, colorGray, summary, colorReset)
		}
	}

	return b.String()
}

// --- helpers ---

func formatStatus(status EligibilityStatus) string {
	switch status {
	case StatusReady:
		return colorGreen + "✓ ready" + colorReset
	case StatusDisabled:
		return colorYellow + "⏸ disabled" + colorReset
	case StatusMissing:
		return colorRed + "✗ missing" + colorReset
	default:
		return string(status)
	}
}

func formatSkillName(entry SkillEntry) string {
	emoji := entry.Metadata.Emoji
	if emoji == "" {
		emoji = "📦"
	}
	return fmt.Sprintf("%s %s%s%s", emoji, colorCyan, entry.Name, colorReset)
}

func formatMissingSummary(m MissingRequirements) string {
	var parts []string
	if len(m.Bins) > 0 {
		parts = append(parts, "bins: "+strings.Join(m.Bins, ", "))
	}
	if len(m.AnyBins) > 0 {
		parts = append(parts, "anyBins: "+strings.Join(m.AnyBins, ", "))
	}
	if len(m.Env) > 0 {
		parts = append(parts, "env: "+strings.Join(m.Env, ", "))
	}
	if len(m.OS) > 0 {
		parts = append(parts, "os: "+strings.Join(m.OS, ", "))
	}
	return strings.Join(parts, "; ")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// colorLen returns the number of extra bytes from ANSI escape codes in a string,
// so we can adjust column widths for alignment.
func colorLen(s string) int {
	stripped := stripANSI(s)
	return len(s) - len(stripped)
}

func stripANSI(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until 'm'
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			i = j + 1
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

func skillNames(skills []SkillStatus) []string {
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	return names
}
