package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage lsbot profiles (isolated data directories)",
	Long: `Profiles let you run multiple fully isolated lsbot instances, each with
its own config, memory, session history, skills, cron jobs, and E2E keys.

Data is stored under ~/.lsbot/profiles/<name>/.

Switch profiles with the global --profile / -p flag or by setting
LSBOT_PROFILE in your environment.`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		root := config.ProfilesDir()
		entries, err := os.ReadDir(root)
		if os.IsNotExist(err) {
			fmt.Println("No profiles found. Create one with: lsbot profile create <name>")
			return nil
		}
		if err != nil {
			return err
		}

		active := config.ActiveProfile()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PROFILE\tPATH\tACTIVE")
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			marker := ""
			if e.Name() == active {
				marker = "✓"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", e.Name(), filepath.Join(root, e.Name()), marker)
		}
		return w.Flush()
	},
}

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new profile directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dir := filepath.Join(config.ProfilesDir(), name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create profile: %w", err)
		}
		fmt.Printf("Profile %q created at %s\n", name, dir)
		fmt.Printf("Use it with: lsbot -p %s <command>\n", name)
		fmt.Printf("Or set:      export LSBOT_PROFILE=%s\n", name)
		return nil
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile and all its data",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dir := filepath.Join(config.ProfilesDir(), name)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("profile %q does not exist", name)
		}
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("This will permanently delete profile %q (%s).\n", name, dir)
			fmt.Print("Are you sure? [y/N] ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Aborted.")
				return nil
			}
		}
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("delete profile: %w", err)
		}
		fmt.Printf("Profile %q deleted.\n", name)
		return nil
	},
}

func init() {
	profileDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	rootCmd.AddCommand(profileCmd)
}
