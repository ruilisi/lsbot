package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage agents and routing bindings",
}

// agents add flags
var (
	agentWorkspace    string
	agentModel        string
	agentProvider     string
	agentAPIKey       string
	agentInstructions string
	agentDefault      bool
	agentAllowTools   string
	agentDenyTools    string
)

// agents list flags
var agentsListBindings bool

// agents bind/unbind flags
var (
	bindAgentID string
	bindSpecs   []string
	unbindAll   bool
)

var agentsAddCmd = &cobra.Command{
	Use:   "add [id]",
	Short: "Add a new agent to ~/.lsbot.yaml",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := ""
		if len(args) == 1 {
			id = args[0]
		}

		interactive := id == "" && !cmd.Flags().Changed("provider") &&
			!cmd.Flags().Changed("model") && !cmd.Flags().Changed("api-key") &&
			!cmd.Flags().Changed("instructions") && !cmd.Flags().Changed("workspace")

		if interactive || id == "" {
			initScanner()
		}

		if id == "" {
			fmt.Println("  Agent name — a short nickname you choose, e.g. \"mybot\", \"work-assistant\"")
			id = promptText("Agent name", "")
			if id == "" {
				return fmt.Errorf("agent name is required")
			}
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Find existing agent (if any) to use as defaults
		existingIdx := -1
		var existing config.AgentEntry
		for i, a := range cfg.Agents {
			if a.ID == id {
				existingIdx = i
				existing = a
				break
			}
		}

		if interactive {
			// progressively prompt for optional fields, defaulting to existing values
			providerOpts := []string{"(inherit from base config)"}
			for _, p := range providers {
				providerOpts = append(providerOpts, p.label)
			}
			defProviderIdx := 0
			for i, p := range providers {
				if p.name == existing.Provider {
					defProviderIdx = i + 1
					break
				}
			}
			idx := promptSelect("AI Provider:", providerOpts, defProviderIdx)
			if idx > 0 {
				agentProvider = providers[idx-1].name
			}

			agentModel = promptText("Model (blank = inherit)", existing.Model)
			agentAPIKey = promptText("API Key (blank = inherit)", existing.APIKey)
			agentInstructions = promptText("Instructions (text or file path, optional)", existing.Instructions)
			agentDefault = promptYesNo("Mark as default agent?", existing.Default)

			home, _ := os.UserHomeDir()
			defaultWS := existing.Workspace
			if defaultWS == "" {
				defaultWS = filepath.Join(home, ".lsbot", "agents", id)
			}
			agentWorkspace = promptText("Workspace directory", defaultWS)
		} else if existingIdx >= 0 {
			// Non-interactive update: only override fields that were explicitly passed
			if !cmd.Flags().Changed("provider") {
				agentProvider = existing.Provider
			}
			if !cmd.Flags().Changed("model") {
				agentModel = existing.Model
			}
			if !cmd.Flags().Changed("api-key") {
				agentAPIKey = existing.APIKey
			}
			if !cmd.Flags().Changed("instructions") {
				agentInstructions = existing.Instructions
			}
			if !cmd.Flags().Changed("default") {
				agentDefault = existing.Default
			}
			if !cmd.Flags().Changed("workspace") {
				agentWorkspace = existing.Workspace
			}
			if agentAllowTools == "" && len(existing.AllowTools) > 0 {
				agentAllowTools = strings.Join(existing.AllowTools, ",")
			}
			if agentDenyTools == "" && len(existing.DenyTools) > 0 {
				agentDenyTools = strings.Join(existing.DenyTools, ",")
			}
		}

		workspace := agentWorkspace
		if workspace == "" {
			home, _ := os.UserHomeDir()
			workspace = filepath.Join(home, ".lsbot", "agents", id)
		}
		// Expand ~ manually
		if strings.HasPrefix(workspace, "~/") {
			home, _ := os.UserHomeDir()
			workspace = filepath.Join(home, workspace[2:])
		}
		if err := os.MkdirAll(workspace, 0755); err != nil {
			return fmt.Errorf("failed to create workspace dir %s: %w", workspace, err)
		}

		entry := config.AgentEntry{
			ID:           id,
			Default:      agentDefault,
			Workspace:    workspace,
			Model:        agentModel,
			Provider:     agentProvider,
			APIKey:       agentAPIKey,
			Instructions: agentInstructions,
		}

		if agentAllowTools != "" {
			for _, t := range strings.Split(agentAllowTools, ",") {
				if t = strings.TrimSpace(t); t != "" {
					entry.AllowTools = append(entry.AllowTools, t)
				}
			}
		}
		if agentDenyTools != "" {
			for _, t := range strings.Split(agentDenyTools, ",") {
				if t = strings.TrimSpace(t); t != "" {
					entry.DenyTools = append(entry.DenyTools, t)
				}
			}
		}

		// If this is marked default, clear default on others
		if agentDefault {
			for i := range cfg.Agents {
				cfg.Agents[i].Default = false
			}
		}

		action := "added"
		if existingIdx >= 0 {
			cfg.Agents[existingIdx] = entry
			action = "updated"
		} else {
			cfg.Agents = append(cfg.Agents, entry)
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Agent %q %s (workspace: %s)\n", id, action, workspace)
		return nil
	},
}

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents (and optionally bindings)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.Agents) == 0 {
			fmt.Println("No agents configured. Use 'lsbot agents add' to add one.")
			return nil
		}

		fmt.Printf("%-12s  %-8s  %-20s  %-20s  %s\n", "ID", "DEFAULT", "MODEL", "PROVIDER", "WORKSPACE")
		fmt.Printf("%-12s  %-8s  %-20s  %-20s  %s\n", "--", "-------", "-----", "--------", "---------")
		for _, a := range cfg.Agents {
			def := ""
			if a.Default {
				def = "✓"
			}
			model := a.Model
			if model == "" {
				model = "(inherited)"
			}
			provider := a.Provider
			if provider == "" {
				provider = "(inherited)"
			}
			fmt.Printf("%-12s  %-8s  %-20s  %-20s  %s\n", a.ID, def, model, provider, a.Workspace)
		}

		if agentsListBindings && len(cfg.Bindings) > 0 {
			fmt.Println()
			fmt.Printf("%-12s  %-12s  %s\n", "AGENT", "PLATFORM", "CHANNEL_ID")
			fmt.Printf("%-12s  %-12s  %s\n", "-----", "--------", "----------")
			for _, b := range cfg.Bindings {
				fmt.Printf("%-12s  %-12s  %s\n", b.AgentID, b.Match.Platform, b.Match.ChannelID)
			}
		} else if agentsListBindings {
			fmt.Println("\nNo bindings configured.")
		}

		return nil
	},
}

var agentsBindCmd = &cobra.Command{
	Use:   "bind",
	Short: "Add routing bindings for an agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(bindSpecs) == 0 {
			return fmt.Errorf("--bind is required")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		agentID := bindAgentID
		if agentID == "" {
			agentID = cfg.DefaultAgentID()
		}
		if agentID == "" {
			return fmt.Errorf("no default agent found; specify --agent <id>")
		}
		if _, found := cfg.FindAgent(agentID); !found {
			return fmt.Errorf("agent %q not found", agentID)
		}

		for _, spec := range bindSpecs {
			parts := strings.SplitN(spec, ":", 2)
			platform := parts[0]
			channelID := ""
			if len(parts) == 2 {
				channelID = parts[1]
			}

			// Check for duplicate
			duplicate := false
			for _, b := range cfg.Bindings {
				if b.AgentID == agentID && b.Match.Platform == platform && b.Match.ChannelID == channelID {
					duplicate = true
					break
				}
			}
			if duplicate {
				fmt.Printf("Binding %q → %s already exists, skipping\n", agentID, spec)
				continue
			}

			cfg.Bindings = append(cfg.Bindings, config.AgentBinding{
				AgentID: agentID,
				Match: config.AgentBindingMatch{
					Platform:  platform,
					ChannelID: channelID,
				},
			})
			fmt.Printf("Bound agent %q to %s\n", agentID, spec)
		}

		return cfg.Save()
	},
}

var agentsUnbindCmd = &cobra.Command{
	Use:   "unbind",
	Short: "Remove routing bindings for an agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		agentID := bindAgentID
		if agentID == "" {
			agentID = cfg.DefaultAgentID()
		}
		if agentID == "" {
			return fmt.Errorf("no default agent found; specify --agent <id>")
		}

		if unbindAll {
			before := len(cfg.Bindings)
			var kept []config.AgentBinding
			for _, b := range cfg.Bindings {
				if b.AgentID != agentID {
					kept = append(kept, b)
				}
			}
			cfg.Bindings = kept
			fmt.Printf("Removed %d binding(s) for agent %q\n", before-len(kept), agentID)
			return cfg.Save()
		}

		if len(bindSpecs) == 0 {
			return fmt.Errorf("--bind or --all is required")
		}

		for _, spec := range bindSpecs {
			parts := strings.SplitN(spec, ":", 2)
			platform := parts[0]
			channelID := ""
			if len(parts) == 2 {
				channelID = parts[1]
			}

			before := len(cfg.Bindings)
			var kept []config.AgentBinding
			for _, b := range cfg.Bindings {
				if b.AgentID == agentID && b.Match.Platform == platform && b.Match.ChannelID == channelID {
					continue
				}
				kept = append(kept, b)
			}
			cfg.Bindings = kept
			removed := before - len(kept)
			if removed > 0 {
				fmt.Printf("Removed binding %q → %s\n", agentID, spec)
			} else {
				fmt.Printf("No binding found for %q → %s\n", agentID, spec)
			}
		}

		return cfg.Save()
	},
}

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsAddCmd)
	agentsCmd.AddCommand(agentsListCmd)
	agentsCmd.AddCommand(agentsBindCmd)
	agentsCmd.AddCommand(agentsUnbindCmd)

	// agents add flags
	agentsAddCmd.Flags().StringVar(&agentWorkspace, "workspace", "", "Workspace directory (default: ~/.lsbot/agents/<id>)")
	agentsAddCmd.Flags().StringVar(&agentModel, "model", "", "Model override for this agent")
	agentsAddCmd.Flags().StringVar(&agentProvider, "provider", "", "AI provider override")
	agentsAddCmd.Flags().StringVar(&agentAPIKey, "api-key", "", "API key override")
	agentsAddCmd.Flags().StringVar(&agentInstructions, "instructions", "", "Inline instructions text or path to file")
	agentsAddCmd.Flags().BoolVar(&agentDefault, "default", false, "Mark as default agent")
	agentsAddCmd.Flags().StringVar(&agentAllowTools, "allow-tools", "", "Comma-separated tool whitelist")
	agentsAddCmd.Flags().StringVar(&agentDenyTools, "deny-tools", "", "Comma-separated tool blacklist")

	// agents list flags
	agentsListCmd.Flags().BoolVarP(&agentsListBindings, "bindings", "b", false, "Also show routing bindings")

	// agents bind/unbind flags
	agentsBindCmd.Flags().StringVar(&bindAgentID, "agent", "", "Agent ID (default: config default agent)")
	agentsBindCmd.Flags().StringArrayVar(&bindSpecs, "bind", nil, "Binding spec: <platform> or <platform>:<channelID> (repeatable)")

	agentsUnbindCmd.Flags().StringVar(&bindAgentID, "agent", "", "Agent ID (default: config default agent)")
	agentsUnbindCmd.Flags().StringArrayVar(&bindSpecs, "bind", nil, "Binding spec to remove (repeatable)")
	agentsUnbindCmd.Flags().BoolVar(&unbindAll, "all", false, "Remove all bindings for this agent")
}
