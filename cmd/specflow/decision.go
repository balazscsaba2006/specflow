package main

import (
	"fmt"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/balazscsaba2006/specflow/internal/ui"
	"github.com/balazscsaba2006/specflow/templates"
	"github.com/spf13/cobra"
)

func newDecisionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "decision",
		Aliases: []string{"dec"},
		Short:   "Manage decisions",
		Long:    "Create, list, and show architectural decisions.",
	}

	cmd.AddCommand(newDecisionNewCmd())
	cmd.AddCommand(newDecisionLsCmd())
	cmd.AddCommand(newDecisionShowCmd())

	return cmd
}

func newDecisionNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <slug>",
		Short: "Create a new decision",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			slug := args[0]
			if err := models.ValidateSlug(slug); err != nil {
				return err
			}

			tmpl, err := templates.Load(appStore.Root(), "decision")
			if err != nil {
				return fmt.Errorf("loading template: %w", err)
			}

			edited, err := openInEditor(tmpl)
			if err != nil {
				return fmt.Errorf("editing decision: %w", err)
			}

			var d models.Decision
			body, err := store.Parse([]byte(edited), &d)
			if err != nil {
				return fmt.Errorf("parsing decision: %w", err)
			}
			d.Slug = slug
			d.Body = body

			if err := appStore.CreateDecision(&d); err != nil {
				return err
			}

			fmt.Printf("Created decision %q (%s)\n", d.Title, d.Slug)
			return nil
		},
	}
}

func newDecisionLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List decisions",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			decisions, err := appStore.ListDecisions()
			if err != nil {
				return err
			}

			headers := []string{"SLUG", "DATE", "TITLE", "STATUS"}
			rows := make([][]string, len(decisions))
			for idx, d := range decisions {
				rows[idx] = []string{d.Slug, d.Date, d.Title, ui.StatusBadge(d.Status)}
			}

			printTable(headers, rows)
			return nil
		},
	}
}

func newDecisionShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <slug>",
		Short: "Show decision details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := appStore.LoadDecision(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("%s  %s\n", ui.Label("ID:"), d.ID)
			fmt.Printf("%s  %s\n", ui.Label("Slug:"), d.Slug)
			fmt.Printf("%s  %s\n", ui.Label("Date:"), d.Date)
			fmt.Printf("%s  %s\n", ui.Label("Title:"), d.Title)
			fmt.Printf("%s  %s\n", ui.Label("Status:"), ui.StatusBadge(d.Status))

			if len(d.ContextRefs) > 0 {
				fmt.Println("Context Refs:")
				for _, ref := range d.ContextRefs {
					fmt.Printf("  - %s\n", ref)
				}
			}
			if d.Body != "" {
				fmt.Printf("\n%s\n", d.Body)
			}

			return nil
		},
	}
}
