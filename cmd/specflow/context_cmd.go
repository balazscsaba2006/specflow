package main

import (
	"fmt"

	sfcontext "github.com/balazscsaba2006/specflow/internal/context"
	"github.com/spf13/cobra"
)

func newContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "context <story-slug>",
		Short: "Preview the assembled 6-layer context for a story",
		Long:  "Prints the same context that sf_context_build returns via MCP. Useful for debugging what Claude Code sees.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			builder := sfcontext.New(appStore, appConfig)
			assembled, err := builder.Build(args[0])
			if err != nil {
				return fmt.Errorf("building context for %q: %w", args[0], err)
			}

			fmt.Print(assembled)
			return nil
		},
	}
}
