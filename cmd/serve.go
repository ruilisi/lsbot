package cmd

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/ruilisi/lsbot/internal/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long:  `Start the MCP server and listen for requests via stdio.`,
	Run: func(cmd *cobra.Command, args []string) {
		s := mcp.NewServer(loadSecurityOptions())
		defer s.Stop()

		// Serve over stdio (default MCP transport)
		if err := server.ServeStdio(s.GetMCPServer()); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
