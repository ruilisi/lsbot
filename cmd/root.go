package cmd

import (
	"fmt"
	"os"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/ruilisi/lsbot/internal/logger"
	"github.com/ruilisi/lsbot/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	logLevel         string
	autoApprove      bool
	disableFileTools bool
	configFile       string
)

var rootCmd = &cobra.Command{
	Use:   "lsbot",
	Short: "Lingti Secure Bot — private AI on your own machine",
	Long: `lsbot (Lingti Secure Bot) is a lean, secure AI bot that runs entirely
on your machine. Your data never leaves your computer.

All relay traffic is end-to-end encrypted. No chat server can read
your messages or access files on your machine.

It provides tools for:
  - File operations (read, write, list, search)
  - Shell command execution
  - System information (CPU, memory, disk)
  - Process management
  - Browser automation`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if configFile != "" {
			config.SetConfigPath(configFile)
		}
		// Parse and set log level
		level, err := logger.ParseLevel(logLevel)
		if err != nil {
			return err
		}
		logger.SetLevel(level)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "",
		"Config file path (default: ~/.lingti.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info",
		"Log level: trace, debug, info, warn, error, fatal, panic")
	rootCmd.PersistentFlags().BoolVarP(&autoApprove, "yes", "y", false,
		"Automatically approve all operations without prompting (skip security checks)")
	rootCmd.PersistentFlags().BoolVar(&disableFileTools, "no-files", false,
		"Disable all file operation tools")
}

// IsAutoApprove returns true if auto-approve mode is enabled globally
func IsAutoApprove() bool {
	return autoApprove
}

// loadAllowedPaths returns security allowed_paths from config file.
func loadAllowedPaths() []string {
	if cfg, err := config.Load(); err == nil {
		return cfg.Security.AllowedPaths
	}
	return nil
}

// loadDisableFileTools returns true if file tools are disabled via flag or config.
func loadDisableFileTools() bool {
	if disableFileTools {
		return true
	}
	if cfg, err := config.Load(); err == nil {
		return cfg.Security.DisableFileTools
	}
	return false
}

// loadSecurityOptions returns MCP security options from config file.
func loadSecurityOptions() mcp.SecurityOptions {
	cfg, err := config.Load()
	if err != nil {
		return mcp.SecurityOptions{}
	}
	return mcp.SecurityOptions{
		AllowedPaths:     cfg.Security.AllowedPaths,
		DisableFileTools: cfg.Security.DisableFileTools,
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
