package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Show usage modes and getting started guide",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(`
Usage modes:

  1. Gateway (recommended — web-based setup at bot.lingti.com):
     lsbot gateway       # Opens bot.lingti.com/bots/xxxx to connect platforms

  2. MCP Server (for Claude Desktop / Cursor / Windsurf):
     Add to your MCP config (claude_desktop_config.json):

     {
       "mcpServers": {
         "lsbot": {
           "command": "/usr/local/bin/lsbot",
           "args": ["serve"]
         }
       }
     }


  3. Cloud Relay (connect to Lingti cloud for Feishu/Slack bots):
     lsbot relay         # Connect to cloud relay service

For more information:
  lsbot help                       # Show all commands
  lsbot <command> --help           # Help for specific command
  https://bot.lingti.com               # Official website & docs
  https://github.com/ruilisi/lsbot # Source code

`)
	},
}

func init() {
	rootCmd.AddCommand(quickstartCmd)
}
