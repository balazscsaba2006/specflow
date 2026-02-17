package main

import (
	"fmt"

	"github.com/balazscsaba2006/specflow/internal/ui"
	"github.com/spf13/cobra"
)

func newModeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mode [fast|careful]",
		Short: "Show or set the project mode",
		Long: `Without arguments, shows the current mode.
With an argument, sets the mode to 'fast' or 'careful'.

Fast mode:     minimal ceremony — lighter templates, fewer prompts
Careful mode:  full ceremony — complete templates, hard questions, verification prompts`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Printf("%s  %s\n", ui.Label("Mode:"), appConfig.Mode)
				return nil
			}

			value := args[0]
			if err := appConfig.Set("mode", value); err != nil {
				return err
			}

			if err := saveConfig(); err != nil {
				return err
			}

			fmt.Printf("Mode set to %q\n", value)
			return nil
		},
	}
}
