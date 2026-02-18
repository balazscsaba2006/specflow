package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	sfcontext "github.com/balazscsaba2006/specflow/internal/context"
	"github.com/spf13/cobra"
)

func newHandoffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "handoff <story-slug>",
		Short: "Render context + plan as a clipboard-ready prompt for any AI agent",
		Long: `Assembles the story context and outputs a plain-text prompt suitable for
pasting into any AI coding agent (Cursor, Cline, Copilot, etc.).

By default copies to clipboard if pbcopy (macOS) or xclip (Linux) is available.
Use --no-copy to print to stdout instead.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			noCopy, _ := cmd.Flags().GetBool("no-copy")

			builder := sfcontext.New(appStore, appConfig)
			assembled, err := builder.Build(args[0])
			if err != nil {
				return fmt.Errorf("building context for %q: %w", args[0], err)
			}

			// Wrap in a handoff header/footer.
			var b strings.Builder
			b.WriteString("# Implementation Task\n\n")
			b.WriteString("You are implementing this task in an existing codebase. Follow the context, plan, and acceptance criteria below.\n\n")
			b.WriteString("---\n\n")
			b.WriteString(assembled)
			b.WriteString("\n---\n\n")
			b.WriteString("Implement this task. Follow the acceptance criteria exactly. Ask clarifying questions if anything is ambiguous.\n")

			prompt := b.String()

			if noCopy {
				fmt.Print(prompt)
				return nil
			}

			if err := copyToClipboard(prompt); err != nil {
				// Fallback to stdout if clipboard unavailable.
				fmt.Print(prompt)
				fmt.Fprintf(cmd.ErrOrStderr(), "\n(clipboard not available: %v — printed to stdout instead)\n", err)
				return nil
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Copied handoff prompt for %q to clipboard (%d chars)\n", args[0], len(prompt))
			return nil
		},
	}

	cmd.Flags().Bool("no-copy", false, "Print to stdout instead of copying to clipboard")

	return cmd
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip or xsel)")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
