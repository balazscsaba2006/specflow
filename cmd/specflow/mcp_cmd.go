package main

import (
	"fmt"
	"os"
	"path/filepath"

	sfmcp "github.com/balazscsaba2006/specflow/internal/mcp"
	"github.com/balazscsaba2006/specflow/templates"
	"github.com/spf13/cobra"
)

// syncSkillOnStartup overwrites .claude/skills/specflow/SKILL.md with the
// embedded template if the file already exists. This keeps the installed skill
// in sync with the binary version without manual intervention.
func syncSkillOnStartup(cmd *cobra.Command) {
	root := appStore.Root()
	projectRoot := filepath.Dir(root)
	skillPath := filepath.Join(projectRoot, ".claude", "skills", "specflow", "SKILL.md")

	// Only sync if the skill was previously installed.
	if _, err := os.Stat(skillPath); err != nil {
		return
	}

	content, err := templates.Load(root, "skill")
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "skill sync: load template: %v\n", err)
		return
	}

	if err := os.WriteFile(skillPath, []byte(content), 0o600); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "skill sync: write: %v\n", err)
		return
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "skill synced from binary")
}

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server on stdio",
		Long:  "Starts the specflow MCP server for Claude Code integration over stdio.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			syncSkillOnStartup(cmd)

			srv := sfmcp.NewServer(appStore, appConfig, version)
			fmt.Fprintln(cmd.ErrOrStderr(), "specflow MCP server started")
			return srv.Run(cmd.Context())
		},
	}
}
