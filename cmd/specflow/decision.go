package main

import (
	"fmt"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/spf13/cobra"
)

const decisionTemplate = `---
title: ""
status: accepted
context_refs: []
---
# Decision Title

## Context

What is the issue that we're seeing that is motivating this decision?

## Decision

What is the change that we're proposing and/or doing?

## Consequences

What becomes easier or more difficult to do because of this change?
`

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

			edited, err := openInEditor(decisionTemplate)
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
				rows[idx] = []string{d.Slug, d.Date, d.Title, d.Status}
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

			fmt.Printf("ID:       %s\n", d.ID)
			fmt.Printf("Slug:     %s\n", d.Slug)
			fmt.Printf("Date:     %s\n", d.Date)
			fmt.Printf("Title:    %s\n", d.Title)
			fmt.Printf("Status:   %s\n", d.Status)

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
