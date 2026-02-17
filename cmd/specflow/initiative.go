package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/spf13/cobra"
)

const initiativeTemplate = `---
title: ""
status: active
goal: ""
success_criteria: []
open_questions: []
---
# Initiative Title

Description of this initiative...
`

func newInitiativeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "initiative",
		Aliases: []string{"i"},
		Short:   "Manage initiatives",
		Long:    "Create, list, show, edit, and update initiatives.",
	}

	cmd.AddCommand(newInitiativeNewCmd())
	cmd.AddCommand(newInitiativeLsCmd())
	cmd.AddCommand(newInitiativeShowCmd())
	cmd.AddCommand(newInitiativeEditCmd())
	cmd.AddCommand(newInitiativeSetCmd())
	cmd.AddCommand(newInitiativeArchiveCmd())

	return cmd
}

func newInitiativeNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <slug>",
		Short: "Create a new initiative",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			slug := args[0]
			if err := models.ValidateSlug(slug); err != nil {
				return err
			}

			edited, err := openInEditor(initiativeTemplate)
			if err != nil {
				return fmt.Errorf("editing initiative: %w", err)
			}

			var i models.Initiative
			body, err := store.Parse([]byte(edited), &i)
			if err != nil {
				return fmt.Errorf("parsing initiative: %w", err)
			}
			i.Slug = slug
			i.Body = body

			if err := appStore.CreateInitiative(&i); err != nil {
				return err
			}

			fmt.Printf("Created initiative %q (%s)\n", i.Title, i.Slug)
			return nil
		},
	}
}

func newInitiativeLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all initiatives",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			initiatives, err := appStore.ListInitiatives()
			if err != nil {
				return err
			}

			headers := []string{"SLUG", "TITLE", "STATUS"}
			rows := make([][]string, len(initiatives))
			for idx, i := range initiatives {
				rows[idx] = []string{i.Slug, i.Title, i.Status}
			}

			printTable(headers, rows)
			return nil
		},
	}
}

func newInitiativeShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <slug>",
		Short: "Show initiative details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			i, err := appStore.LoadInitiative(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("ID:       %s\n", i.ID)
			fmt.Printf("Slug:     %s\n", i.Slug)
			fmt.Printf("Title:    %s\n", i.Title)
			fmt.Printf("Status:   %s\n", i.Status)
			fmt.Printf("Goal:     %s\n", i.Goal)
			fmt.Printf("Created:  %s\n", i.Created.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated:  %s\n", i.Updated.Format("2006-01-02 15:04:05"))

			if len(i.SuccessCriteria) > 0 {
				fmt.Println("Success Criteria:")
				for _, sc := range i.SuccessCriteria {
					fmt.Printf("  - %s\n", sc)
				}
			}
			if len(i.OpenQuestions) > 0 {
				fmt.Println("Open Questions:")
				for _, q := range i.OpenQuestions {
					fmt.Printf("  - %s\n", q)
				}
			}
			if len(i.Epics) > 0 {
				fmt.Println("Epics:")
				for _, e := range i.Epics {
					fmt.Printf("  - %s\n", e)
				}
			}
			if i.Body != "" {
				fmt.Printf("\n%s\n", i.Body)
			}

			return nil
		},
	}
}

func newInitiativeEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <slug>",
		Short: "Edit an initiative in $EDITOR",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			slug := args[0]

			i, err := appStore.LoadInitiative(slug)
			if err != nil {
				return err
			}

			data, err := store.Marshal(i, i.Body)
			if err != nil {
				return fmt.Errorf("marshaling initiative: %w", err)
			}

			edited, err := openInEditor(string(data))
			if err != nil {
				return fmt.Errorf("editing initiative: %w", err)
			}

			var updated models.Initiative
			body, err := store.Parse([]byte(edited), &updated)
			if err != nil {
				return fmt.Errorf("parsing initiative: %w", err)
			}

			// Preserve immutable fields from the original.
			updated.ID = i.ID
			updated.Slug = i.Slug
			updated.Created = i.Created
			updated.Body = body

			if err := appStore.SaveInitiative(&updated); err != nil {
				return err
			}

			fmt.Printf("Updated initiative %q\n", updated.Slug)
			return nil
		},
	}
}

func newInitiativeSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <slug> <field> <value>",
		Short: "Quick-update a field (status, title, goal)",
		Args:  cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			slug, field, value := args[0], args[1], args[2]

			i, err := appStore.LoadInitiative(slug)
			if err != nil {
				return err
			}

			switch field {
			case "status":
				if !slices.Contains(models.ValidInitiativeStatuses, value) {
					return fmt.Errorf("invalid status %q: must be one of [%s]", value, strings.Join(models.ValidInitiativeStatuses, ", "))
				}
				i.Status = value
			case "title":
				i.Title = value
			case "goal":
				i.Goal = value
			default:
				return fmt.Errorf("unsupported field %q: must be one of [status, title, goal]", field)
			}

			if err := appStore.SaveInitiative(i); err != nil {
				return err
			}

			fmt.Printf("Set %s = %q on initiative %q\n", field, value, slug)
			return nil
		},
	}
}

func newInitiativeArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive <slug>",
		Short: "Archive an initiative (shortcut for set <slug> status archived)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			slug := args[0]

			i, err := appStore.LoadInitiative(slug)
			if err != nil {
				return err
			}

			i.Status = models.InitiativeStatusArchived

			if err := appStore.SaveInitiative(i); err != nil {
				return err
			}

			fmt.Printf("Archived initiative %q\n", slug)
			return nil
		},
	}
}
