package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/ruilisi/lsbot/internal/termui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var skinCmd = &cobra.Command{
	Use:   "skin",
	Short: "Manage CLI skins (visual themes)",
	Long: `Skins customise the lsbot terminal appearance: spinner frames, colours,
tool prefix, agent name, and prompt symbol.

Built-in skins: default, mono, hacker, warm.

Custom skins: drop a YAML file into ~/.lsbot/skins/<name>.yaml and
activate it with 'lsbot skin set <name>' or LSBOT_SKIN=<name>.`,
}

var skinListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available skins",
	RunE: func(cmd *cobra.Command, args []string) error {
		hubDir := config.HubDir()
		active := os.Getenv("LSBOT_SKIN")
		if active == "" {
			active = "default"
		}

		builtins := termui.ListBuiltinSkins()
		sort.Slice(builtins, func(i, j int) bool { return builtins[i].Name < builtins[j].Name })

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tSOURCE\tACTIVE")
		for _, s := range builtins {
			marker := ""
			if s.Name == active {
				marker = "✓"
			}
			fmt.Fprintf(w, "%s\t%s\tbuilt-in\t%s\n", s.Name, s.Description, marker)
		}

		// User skins
		skinsDir := fmt.Sprintf("%s/skins", hubDir)
		entries, err := os.ReadDir(skinsDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() || len(e.Name()) < 6 {
					continue
				}
				name := e.Name()[:len(e.Name())-5] // strip .yaml
				marker := ""
				if name == active {
					marker = "✓"
				}
				fmt.Fprintf(w, "%s\t(user skin)\tuser\t%s\n", name, marker)
			}
		}
		return w.Flush()
	},
}

var skinSetCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "Set the active skin (writes LSBOT_SKIN to shell config hint)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		skin := termui.LoadSkin(config.HubDir(), name)
		fmt.Printf("✓ Skin: %s — %s\n", skin.Name, skin.Description)
		fmt.Printf("\nTo make this permanent, add to your shell profile:\n")
		fmt.Printf("  export LSBOT_SKIN=%s\n", name)
		fmt.Printf("\nOr set it per-command:\n")
		fmt.Printf("  LSBOT_SKIN=%s lsbot relay ...\n", name)
		return nil
	},
}

var skinNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new user skin YAML in ~/.lsbot/skins/",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dir := fmt.Sprintf("%s/skins", config.HubDir())
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		path := fmt.Sprintf("%s/%s.yaml", dir, name)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("skin %q already exists at %s", name, path)
		}

		scaffold := termui.Skin{
			Name:          name,
			Description:   "My custom skin",
			SpinnerFrames: []string{"⠋", "⠙", "⠹", "⠸"},
			ThinkingVerbs: []string{"thinking", "working"},
			ToolPrefix:    "┊ ",
			Colors: map[string]string{
				"banner":          "\033[36m",
				"response_border": "\033[36m",
				"tool_name":       "\033[90m",
				"prompt_symbol":   "\033[32m",
			},
		}
		scaffold.Branding.AgentName = "lsbot"
		scaffold.Branding.PromptSymbol = "❯"
		scaffold.Branding.WelcomeMsg = "Hello!"

		data, err := yaml.Marshal(scaffold)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return err
		}
		fmt.Printf("✓ Skin scaffold created at %s\n", path)
		fmt.Printf("  Edit it, then activate with: lsbot skin set %s\n", name)
		return nil
	},
}

func init() {
	skinCmd.AddCommand(skinListCmd)
	skinCmd.AddCommand(skinSetCmd)
	skinCmd.AddCommand(skinNewCmd)
	rootCmd.AddCommand(skinCmd)
}
