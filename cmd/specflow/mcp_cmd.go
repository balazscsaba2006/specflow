package main

import (
	"fmt"

	sfmcp "github.com/balazscsaba2006/specflow/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server on stdio",
		Long:  "Starts the specflow MCP server for Claude Code integration over stdio.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			srv := sfmcp.NewServer(appStore, appConfig, version)
			fmt.Fprintln(cmd.ErrOrStderr(), "specflow MCP server started")
			return srv.Run(cmd.Context())
		},
	}
}
