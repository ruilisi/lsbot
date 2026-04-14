package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/ruilisi/lsbot/internal/skills"
	"github.com/spf13/cobra"
)

var (
	skillsJSON     bool
	skillsEligible bool
	skillsVerbose  bool
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "List and inspect available skills",
	Long: `Discover, inspect, and manage skills.

Skills are loaded from multiple directories in precedence order:
  1. Bundled skills  (shipped with binary)
  2. Managed skills  (~/.lsbot/skills/)
  3. Workspace skills (./skills/)

Each skill is a directory containing a SKILL.md file with YAML frontmatter
that declares requirements (binaries, env vars, OS) and metadata.`,
	Run: func(cmd *cobra.Command, args []string) {
		runSkillsList(cmd, args)
	},
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available skills",
	Run:   runSkillsList,
}

var skillsInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed information about a skill",
	Args:  cobra.ExactArgs(1),
	Run:   runSkillsInfo,
}

var skillsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check which skills are ready vs missing requirements",
	Run:   runSkillsCheck,
}

var skillsEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a disabled skill",
	Args:  cobra.ExactArgs(1),
	Run:   runSkillsEnable,
}

var skillsDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a skill",
	Args:  cobra.ExactArgs(1),
	Run:   runSkillsDisable,
}

var skillsDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download bundled skills from GitHub",
	Long:  `Download the latest bundled skills from the lsbot GitHub repository into ~/.lsbot/skills/.`,
	Run:   runSkillsDownload,
}

func init() {
	rootCmd.AddCommand(skillsCmd)

	// Subcommands
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsInfoCmd)
	skillsCmd.AddCommand(skillsCheckCmd)
	skillsCmd.AddCommand(skillsEnableCmd)
	skillsCmd.AddCommand(skillsDisableCmd)
	skillsCmd.AddCommand(skillsDownloadCmd)

	// Flags shared across subcommands
	for _, cmd := range []*cobra.Command{skillsCmd, skillsListCmd, skillsInfoCmd, skillsCheckCmd} {
		cmd.Flags().BoolVar(&skillsJSON, "json", false, "Output as JSON")
	}

	skillsListCmd.Flags().BoolVar(&skillsEligible, "eligible", false, "Show only eligible (ready-to-use) skills")
	skillsListCmd.Flags().BoolVarP(&skillsVerbose, "verbose", "v", false, "Show missing requirements details")
}

func loadSkillsConfig() ([]string, []string) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil
	}
	return cfg.Skills.Disabled, cfg.Skills.ExtraDirs
}

func runSkillsList(_ *cobra.Command, _ []string) {
	disabled, extraDirs := loadSkillsConfig()
	report := skills.BuildStatusReport(disabled, extraDirs)
	fmt.Println(skills.FormatList(report, skills.FormatListOptions{
		JSON:     skillsJSON,
		Eligible: skillsEligible,
		Verbose:  skillsVerbose,
	}))
}

func runSkillsInfo(_ *cobra.Command, args []string) {
	disabled, extraDirs := loadSkillsConfig()
	report := skills.BuildStatusReport(disabled, extraDirs)
	fmt.Println(skills.FormatInfo(report, args[0], skillsJSON))
}

func runSkillsCheck(_ *cobra.Command, _ []string) {
	disabled, extraDirs := loadSkillsConfig()
	report := skills.BuildStatusReport(disabled, extraDirs)
	fmt.Println(skills.FormatCheck(report, skillsJSON))
}

func runSkillsDownload(_ *cobra.Command, _ []string) {
	fmt.Println("Downloading bundled skills from GitHub...")
	count, err := skills.DownloadBundledSkills("", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Downloaded %d skills to %s\n", count, config.SkillsDir())
}

func runSkillsEnable(_ *cobra.Command, args []string) {
	name := args[0]
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	idx := slices.Index(cfg.Skills.Disabled, name)
	if idx == -1 {
		fmt.Printf("Skill %q is not disabled.\n", name)
		return
	}

	cfg.Skills.Disabled = slices.Delete(cfg.Skills.Disabled, idx, idx+1)
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Skill %q enabled.\n", name)
}

func runSkillsDisable(_ *cobra.Command, args []string) {
	name := args[0]
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if slices.Contains(cfg.Skills.Disabled, name) {
		fmt.Printf("Skill %q is already disabled.\n", name)
		return
	}

	cfg.Skills.Disabled = append(cfg.Skills.Disabled, name)
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Skill %q disabled.\n", name)
}
