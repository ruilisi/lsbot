package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const routerDeprecationNotice = `
WARNING: 'router' is deprecated and will be removed in a future release.
Use the new 'channels', 'agents', and 'gateway' commands instead:

  # 1. Save your platform credentials to ~/.lingti.yaml
  lsbot channels add --channel telegram --token YOUR_BOT_TOKEN
  lsbot channels add --channel slack --bot-token xoxb-... --app-token xapp-...

  # 2. Add an agent (optional — skip if using env vars / flags only)
  lsbot agents add main --default

  # 3. Start everything
  lsbot gateway --api-key YOUR_API_KEY

  # Or keep passing flags/env vars directly (all router flags work on gateway):
  lsbot gateway --telegram-token YOUR_BOT_TOKEN --api-key YOUR_API_KEY

`

// routerCmd is a deprecated alias for gatewayCmd.
// All functionality has been merged into `gateway`.
var routerCmd = &cobra.Command{
	Use:                "router",
	Hidden:             true,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprint(os.Stderr, routerDeprecationNotice)
		os.Exit(1)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(routerCmd)
}
