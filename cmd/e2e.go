package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/ruilisi/lsbot/internal/e2e"
	"github.com/spf13/cobra"
)

var e2eCmd = &cobra.Command{
	Use:   "e2e",
	Short: "Manage end-to-end encryption keys",
	Long:  `Commands for managing E2EE key pairs used for secure bot-page chat.`,
}

var (
	e2eKeygenPath string
	e2eKeygenSave bool
	e2ePubkeyFile string
	e2eRegenPath  string
	e2eRegenSave  bool
)

var e2eKeygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generate a new E2EE key pair",
	Long: `Generate a P-256 key pair and save it as a PEM file.
Use --save to record the key file path in ~/.lsbot.yaml so it is
loaded automatically when running "lsbot relay".`,
	Run: func(cmd *cobra.Command, args []string) {
		keyFile := e2eKeygenPath
		if keyFile == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: cannot determine home dir: %v\n", err)
				os.Exit(1)
			}
			keyFile = filepath.Join(homeDir, ".lsbot-e2e.pem")
		}

		// Don't overwrite an existing key without explicit path
		if _, err := os.Stat(keyFile); err == nil {
			fmt.Fprintf(os.Stderr, "Error: key file already exists: %s\n", keyFile)
			fmt.Fprintln(os.Stderr, "Use --path to specify a different location.")
			os.Exit(1)
		}

		priv, err := e2e.GenerateOrLoadKeyPair(keyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Key generated:   %s\n", keyFile)
		fmt.Printf("Public key:      %s\n", e2e.PublicKeyToBase64(priv.PublicKey()))
		fmt.Printf("Fingerprint:     %s\n", e2e.Fingerprint(priv.PublicKey()))

		if e2eKeygenSave {
			cfg, err := config.Load()
			if err != nil {
				cfg = config.DefaultConfig()
			}
			cfg.E2EKeyFile = keyFile
			if err := cfg.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save key path to config: %v\n", err)
			} else {
				fmt.Printf("Saved to config: %s\n", config.ConfigPath())
			}
		}
	},
}

var e2eRegenCmd = &cobra.Command{
	Use:   "regen",
	Short: "Regenerate the E2EE key pair",
	Long: `Delete the existing E2EE key pair and generate a new one.
The new public key must be re-pasted into the browser's Secure mode setup panel.`,
	Run: func(cmd *cobra.Command, args []string) {
		keyFile := e2eRegenPath
		if keyFile == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: cannot determine home dir: %v\n", err)
				os.Exit(1)
			}
			keyFile = filepath.Join(homeDir, ".lsbot-e2e.pem")
		}

		if _, err := os.Stat(keyFile); err == nil {
			if err := os.Remove(keyFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to remove existing key: %v\n", err)
				os.Exit(1)
			}
		}

		priv, err := e2e.GenerateOrLoadKeyPair(keyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Key regenerated: %s\n", keyFile)
		fmt.Printf("Public key:      %s\n", e2e.PublicKeyToBase64(priv.PublicKey()))
		fmt.Printf("Fingerprint:     %s\n", e2e.Fingerprint(priv.PublicKey()))
		fmt.Println("Re-paste the public key into the browser's Secure mode setup panel.")

		if e2eRegenSave {
			cfg, err := config.Load()
			if err != nil {
				cfg = config.DefaultConfig()
			}
			cfg.E2EKeyFile = keyFile
			if err := cfg.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save key path to config: %v\n", err)
			} else {
				fmt.Printf("Saved to config: %s\n", config.ConfigPath())
			}
		}
	},
}

var e2ePubkeyCmd = &cobra.Command{
	Use:   "pubkey",
	Short: "Print the bot's E2EE public key",
	Long: `Print the base64-encoded public key and fingerprint for the bot's E2EE key pair.
Paste the public key into the browser's Secure mode setup panel, then verify
the fingerprint matches what is shown in the browser.`,
	Run: func(cmd *cobra.Command, args []string) {
		keyFile := e2ePubkeyFile
		if keyFile == "" {
			// Resolve: env → saved config → default
			keyFile = os.Getenv("E2E_KEY_FILE")
		}
		if keyFile == "" {
			cfg, err := config.Load()
			if err == nil && cfg.E2EKeyFile != "" {
				keyFile = cfg.E2EKeyFile
			}
		}
		if keyFile == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: cannot determine home dir: %v\n", err)
				os.Exit(1)
			}
			newPath := filepath.Join(homeDir, ".lsbot-e2e.pem")
			// Migrate from legacy path if new path doesn't exist yet
			legacyPath := filepath.Join(homeDir, ".lingti-e2e.pem")
			if _, statErr := os.Stat(newPath); os.IsNotExist(statErr) {
				if _, statErr2 := os.Stat(legacyPath); statErr2 == nil {
					_ = os.Rename(legacyPath, newPath)
				}
			}
			keyFile = newPath
		}

		priv, err := e2e.LoadKeyPair(keyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintln(os.Stderr, "Run 'lsbot e2e keygen' to create a key pair first.")
			os.Exit(1)
		}

		fmt.Printf("Key file:    %s\n", keyFile)
		fmt.Printf("Public key:  %s\n", e2e.PublicKeyToBase64(priv.PublicKey()))
		fmt.Printf("Fingerprint: %s\n", e2e.Fingerprint(priv.PublicKey()))
	},
}

func init() {
	rootCmd.AddCommand(e2eCmd)

	e2eCmd.AddCommand(e2eKeygenCmd)
	e2eKeygenCmd.Flags().StringVar(&e2eKeygenPath, "path", "", "Path to save the PEM key file (default: ~/.lsbot-e2e.pem)")
	e2eKeygenCmd.Flags().BoolVar(&e2eKeygenSave, "save", false, "Save key file path to ~/.lsbot.yaml")

	e2eCmd.AddCommand(e2eRegenCmd)
	e2eRegenCmd.Flags().StringVar(&e2eRegenPath, "path", "", "Path to the PEM key file (default: ~/.lsbot-e2e.pem)")
	e2eRegenCmd.Flags().BoolVar(&e2eRegenSave, "save", false, "Save key file path to ~/.lsbot.yaml")

	e2eCmd.AddCommand(e2ePubkeyCmd)
	e2ePubkeyCmd.Flags().StringVar(&e2ePubkeyFile, "key-file", "", "Path to the PEM key file (default: ~/.lsbot-e2e.pem)")
}
