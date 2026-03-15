package cmd

import (
	"fmt"

	"github.com/ruilisi/lsbot/internal/mcp"
	"github.com/spf13/cobra"
)

var build = "unknown"

// SetBuild sets the build string from main
func SetBuild(b string) {
	build = b
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("lsbot %s (%s)\n", mcp.ServerVersion, build)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
