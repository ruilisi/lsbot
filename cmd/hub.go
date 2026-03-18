package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/ruilisi/lsbot/internal/clawhub"
	"github.com/spf13/cobra"
)

var hubCmd = &cobra.Command{
	Use:   "hub",
	Short: "Manage skills from ClawHub (clawhub.ai)",
	Long:  `Search, install, update, and remove skills from the ClawHub skill marketplace.`,
}

var hubSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search ClawHub for skills",
	Args:  cobra.ExactArgs(1),
	Run:   runHubSearch,
}

var hubInstallCmd = &cobra.Command{
	Use:   "install <slug>",
	Short: "Install a skill from ClawHub",
	Args:  cobra.ExactArgs(1),
	Run:   runHubInstall,
}

var (
	hubUpdateAll bool
)

var hubUpdateCmd = &cobra.Command{
	Use:   "update [slug]",
	Short: "Update installed skill(s)",
	Args:  cobra.MaximumNArgs(1),
	Run:   runHubUpdate,
}

var hubRemoveCmd = &cobra.Command{
	Use:   "remove <slug>",
	Short: "Remove an installed skill",
	Args:  cobra.ExactArgs(1),
	Run:   runHubRemove,
}

var hubListCmd = &cobra.Command{
	Use:   "list",
	Short: "List hub-installed skills and their versions",
	Run:   runHubList,
}

func init() {
	rootCmd.AddCommand(hubCmd)
	hubCmd.AddCommand(hubSearchCmd)
	hubCmd.AddCommand(hubInstallCmd)
	hubCmd.AddCommand(hubUpdateCmd)
	hubCmd.AddCommand(hubRemoveCmd)
	hubCmd.AddCommand(hubListCmd)

	hubUpdateCmd.Flags().BoolVar(&hubUpdateAll, "all", false, "Update all installed skills")
}

func runHubSearch(_ *cobra.Command, args []string) {
	client := clawhub.NewClient()
	results, err := client.Search(context.Background(), args[0], 20)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No skills found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SLUG\tNAME\tVERSION\tSUMMARY")
	for _, r := range results {
		summary := r.Summary
		if len(summary) > 60 {
			summary = summary[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Slug, r.DisplayName, r.Version, summary)
	}
	w.Flush()
}

func runHubInstall(_ *cobra.Command, args []string) {
	slug := args[0]
	client := clawhub.NewClient()

	fmt.Printf("Installing %s...\n", slug)
	version, err := clawhub.Install(context.Background(), client, slug, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if version != "" {
		fmt.Printf("Installed %s@%s\n", slug, version)
	} else {
		fmt.Printf("Installed %s\n", slug)
	}
}

func runHubUpdate(_ *cobra.Command, args []string) {
	client := clawhub.NewClient()
	ctx := context.Background()

	if hubUpdateAll || len(args) == 0 {
		lock, err := clawhub.LoadLock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading lock file: %v\n", err)
			os.Exit(1)
		}
		if len(lock.Skills) == 0 {
			fmt.Println("No hub-installed skills to update.")
			return
		}
		for slug := range lock.Skills {
			updateSkill(ctx, client, slug)
		}
		return
	}

	updateSkill(ctx, client, args[0])
}

func updateSkill(ctx context.Context, client *clawhub.Client, slug string) {
	lock, err := clawhub.LoadLock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	entry, ok := lock.Get(slug)
	if !ok {
		fmt.Fprintf(os.Stderr, "%s is not installed\n", slug)
		return
	}

	result, err := client.Resolve(ctx, slug, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking %s: %v\n", slug, err)
		return
	}

	if result.LatestVersion != "" && result.LatestVersion == entry.Version {
		fmt.Printf("%s is already up to date (%s)\n", slug, entry.Version)
		return
	}

	fmt.Printf("Updating %s...\n", slug)
	version, err := clawhub.Install(ctx, client, slug, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating %s: %v\n", slug, err)
		return
	}

	if version != "" {
		fmt.Printf("Updated %s to %s\n", slug, version)
	} else {
		fmt.Printf("Updated %s\n", slug)
	}
}

func runHubRemove(_ *cobra.Command, args []string) {
	slug := args[0]
	if err := clawhub.Remove(slug); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Removed %s\n", slug)
}

func runHubList(_ *cobra.Command, _ []string) {
	lock, err := clawhub.LoadLock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(lock.Skills) == 0 {
		fmt.Println("No hub-installed skills. Use 'lsbot hub install <slug>' to install one.")
		return
	}

	slugs := make([]string, 0, len(lock.Skills))
	for slug := range lock.Skills {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SLUG\tVERSION\tINSTALLED AT")
	for _, slug := range slugs {
		e := lock.Skills[slug]
		fmt.Fprintf(w, "%s\t%s\t%s\n", e.Slug, e.Version, e.InstalledAt.Format("2006-01-02 15:04:05"))
	}
	w.Flush()
}
