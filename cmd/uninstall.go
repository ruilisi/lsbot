package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/ruilisi/lsbot/internal/service"
	"github.com/spf13/cobra"
)

var uninstallForce bool

var topLevelUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Fully remove lsbot from this system",
	Long: `Fully remove lsbot from this system, including:
  - System service (launchd/systemd)
  - Installed binary
  - Configuration files
  - Skills directory
  - Log files

Requires confirmation unless --force is used.`,
	Run: runUninstall,
}

func init() {
	rootCmd.AddCommand(topLevelUninstallCmd)
	topLevelUninstallCmd.Flags().BoolVarP(&uninstallForce, "force", "f", false, "Skip confirmation prompt")
}

func runUninstall(cmd *cobra.Command, args []string) {
	if !uninstallForce {
		fmt.Println("This will fully remove lsbot from your system:")
		fmt.Println()

		if service.IsInstalled() {
			binaryPath, configPath := service.Paths()
			fmt.Printf("  - Service binary:  %s\n", binaryPath)
			fmt.Printf("  - Service config:  %s\n", configPath)
		}
		fmt.Printf("  - Config dir:      %s\n", config.ConfigDir())
		fmt.Printf("  - Skills dir:      %s\n", config.SkillsDir())
		fmt.Printf("  - Log file:        /tmp/lsbot.log\n")
		fmt.Println()
		fmt.Print("Are you sure? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return
		}
	}

	var errors []string

	// 1. Stop and remove service
	if service.IsInstalled() {
		fmt.Println("Removing system service...")
		if err := service.Uninstall(); err != nil {
			errors = append(errors, fmt.Sprintf("service: %v", err))
		}
	}

	// 2. Remove config directory
	configDir := config.ConfigDir()
	if _, err := os.Stat(configDir); err == nil {
		fmt.Printf("Removing config dir: %s\n", configDir)
		if err := os.RemoveAll(configDir); err != nil {
			errors = append(errors, fmt.Sprintf("config dir: %v", err))
		}
	}

	// 3. Remove skills directory
	skillsDir := config.SkillsDir()
	if _, err := os.Stat(skillsDir); err == nil {
		fmt.Printf("Removing skills dir: %s\n", skillsDir)
		if err := os.RemoveAll(skillsDir); err != nil {
			errors = append(errors, fmt.Sprintf("skills dir: %v", err))
		}
	}

	// 4. Remove log file
	logFile := "/tmp/lsbot.log"
	if _, err := os.Stat(logFile); err == nil {
		fmt.Printf("Removing log file: %s\n", logFile)
		if err := os.Remove(logFile); err != nil {
			errors = append(errors, fmt.Sprintf("log file: %v", err))
		}
	}

	fmt.Println()
	if len(errors) > 0 {
		fmt.Println("Completed with errors:")
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Println("lsbot has been fully uninstalled.")

	// Check if we're running from a non-service path (user's own binary)
	execPath, err := os.Executable()
	if err == nil {
		binaryPath, _ := service.Paths()
		if execPath != binaryPath {
			fmt.Printf("\nNote: this binary was not removed: %s\n", execPath)
			fmt.Println("You can delete it manually.")
		}
	}
}
