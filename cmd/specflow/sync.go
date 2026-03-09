package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Update the Claude Code skill from the current binary",
		Long:  "Re-writes ~/.claude/skills/specflow/SKILL.md from the embedded template in this binary version.",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := installSkillGlobal(); err != nil {
				return fmt.Errorf("syncing skill: %w", err)
			}

			home, _ := os.UserHomeDir()
			fmt.Printf("Updated %s/.claude/skills/specflow/SKILL.md (specflow %s)\n", home, version)

			// Hint about per-project leftovers.
			if cwd, err := os.Getwd(); err == nil {
				perProject := cwd + "/.claude/skills/specflow/SKILL.md"
				if _, err := os.Stat(perProject); err == nil {
					fmt.Println()
					fmt.Println("Note: Found per-project skill at .claude/skills/specflow/SKILL.md — this is no longer used.")
					fmt.Println("The skill is now installed globally. You can safely delete the per-project copy.")
				}
			}

			return nil
		},
	}
}
