package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/export"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <epic-slug>",
		Short: "Export an epic and its stories to a markdown file",
		Long:  "Generates a single human-readable markdown file from a specflow epic and all its stories.",
		Args:  cobra.ExactArgs(1),
		RunE:  runExport,
	}

	cmd.Flags().StringP("output", "o", "", "Output file path (default: <epic-slug>-export.md)")
	cmd.Flags().Bool("no-body", false, "Omit markdown body content from stories")
	cmd.Flags().Bool("exclude-done", false, "Skip stories with status done")

	return cmd
}

func runExport(cmd *cobra.Command, args []string) error {
	epicSlug := args[0]
	output, _ := cmd.Flags().GetString("output")
	noBody, _ := cmd.Flags().GetBool("no-body")
	excludeDone, _ := cmd.Flags().GetBool("exclude-done")

	opts := export.ExportOptions{
		IncludeDone: !excludeDone,
		IncludeBody: !noBody,
	}

	data, err := export.ExportEpic(appStore, epicSlug, opts)
	if err != nil {
		return err
	}

	md := renderExportMarkdown(data)

	if output == "" {
		output = epicSlug + "-export.md"
	}

	if err := os.WriteFile(output, []byte(md), 0o600); err != nil {
		return fmt.Errorf("writing export file: %w", err)
	}

	fmt.Printf("Exported 1 epic + %d stories to %s\n", len(data.Stories), output)
	return nil
}

func renderExportMarkdown(data *export.ExportData) string {
	var b strings.Builder

	// Epic header.
	fmt.Fprintf(&b, "# %s\n\n", data.Epic.Title)
	fmt.Fprintf(&b, "**Status:** %s", data.Epic.Status)
	if data.Epic.Fidelity != "" {
		fmt.Fprintf(&b, " | **Fidelity:** %s", data.Epic.Fidelity)
	}
	b.WriteString("\n")

	// Epic body.
	if data.Epic.Body != "" {
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(data.Epic.Body))
		b.WriteString("\n")
	}

	// Phases overview.
	if len(data.Epic.Phases) > 0 {
		b.WriteString("\n## Phases\n")
		// Build a slug→story map for quick lookup.
		storyMap := make(map[string]export.StoryExport)
		for i := range data.Stories {
			storyMap[data.Stories[i].Slug] = data.Stories[i]
		}
		for _, phase := range data.Epic.Phases {
			fmt.Fprintf(&b, "\n### %s\n", phase.Label)
			for _, slug := range phase.Stories {
				if st, ok := storyMap[slug]; ok {
					fmt.Fprintf(&b, "- %s (%s, %s)\n", st.Title, st.Status, st.Priority)
				} else {
					fmt.Fprintf(&b, "- %s (filtered)\n", slug)
				}
			}
		}
	}

	// Stories.
	if len(data.Stories) > 0 {
		b.WriteString("\n---\n\n## Stories\n")
		for i := range data.Stories {
			st := &data.Stories[i]
			fmt.Fprintf(&b, "\n### %s\n\n", st.Title)
			fmt.Fprintf(&b, "**Status:** %s | **Priority:** %s", st.Status, st.Priority)
			if len(st.Labels) > 0 {
				fmt.Fprintf(&b, " | **Labels:** %s", strings.Join(st.Labels, ", "))
			}
			b.WriteString("\n")

			if st.Description != "" {
				b.WriteString("\n")
				b.WriteString(st.Description)
				b.WriteString("\n")
			}

			if i < len(data.Stories)-1 {
				b.WriteString("\n---\n")
			}
		}
	}

	b.WriteString("\n")
	return b.String()
}
