package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show activity log",
		Long:  "Reads the last N entries from the activity log (log.jsonl) and displays them as a timeline.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n, _ := cmd.Flags().GetInt("last")
			return showLog(n)
		},
	}

	cmd.Flags().IntP("last", "n", 20, "Number of recent entries to show")

	return cmd
}

func showLog(last int) error {
	entries, err := appStore.ReadLog(last)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No log entries found.")
		return nil
	}

	for idx := range entries {
		e := &entries[idx]
		ts := e.Timestamp.Format("2006-01-02 15:04:05")

		switch e.Type {
		case "story.status_changed":
			fmt.Printf("[%s] %s: %s -> %s (%s)\n", ts, e.Type, e.From, e.To, e.Entity)
		case "execution.started":
			fmt.Printf("[%s] %s: story=%s ref=%s\n", ts, e.Type, e.Story, e.GitRef)
		case "execution.completed":
			fmt.Printf("[%s] %s: story=%s files_changed=%d ref=%s\n", ts, e.Type, e.Story, e.FilesChanged, e.GitRef)
		case "verification.saved":
			fmt.Printf("[%s] %s: story=%s result=%s (critical:%d major:%d minor:%d)\n",
				ts, e.Type, e.Story, e.Result, e.Critical, e.Major, e.Minor)
		default:
			line := fmt.Sprintf("[%s] %s", ts, e.Type)
			if e.Entity != "" {
				line += fmt.Sprintf(": %s", e.Entity)
			}
			if e.Epic != "" {
				line += fmt.Sprintf(" (epic:%s)", e.Epic)
			}
			if e.Story != "" {
				line += fmt.Sprintf(" (story:%s)", e.Story)
			}
			fmt.Println(line)
		}
	}

	fmt.Printf("\nShowing %d entries\n", len(entries))

	return nil
}
