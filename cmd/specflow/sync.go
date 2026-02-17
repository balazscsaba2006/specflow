package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/balazscsaba2006/specflow/templates"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Update the Claude Code skill from the current binary",
		Long:  "Re-writes .claude/skills/specflow/SKILL.md from the embedded template in this binary version.",
		RunE: func(_ *cobra.Command, _ []string) error {
			root := appStore.Root()
			projectRoot := filepath.Dir(root)

			content, err := templates.Load(root, "skill")
			if err != nil {
				return fmt.Errorf("loading skill template: %w", err)
			}

			skillDir := filepath.Join(projectRoot, ".claude", "skills", "specflow")
			if err := os.MkdirAll(skillDir, 0o750); err != nil {
				return fmt.Errorf("creating skill directory: %w", err)
			}

			skillPath := filepath.Join(skillDir, "SKILL.md")
			if err := os.WriteFile(skillPath, []byte(content), 0o600); err != nil {
				return fmt.Errorf("writing skill: %w", err)
			}

			fmt.Printf("Updated %s (specflow %s)\n", skillPath, version)
			return nil
		},
	}
}
