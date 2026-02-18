package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/balazscsaba2006/specflow/internal/ui"
	"github.com/balazscsaba2006/specflow/templates"
	"github.com/spf13/cobra"
)

func newEpicCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "epic",
		Aliases: []string{"e"},
		Short:   "Manage epics",
		Long:    "Create, list, show, edit, and update epics.",
	}

	cmd.AddCommand(newEpicNewCmd())
	cmd.AddCommand(newEpicLsCmd())
	cmd.AddCommand(newEpicShowCmd())
	cmd.AddCommand(newEpicEditCmd())
	cmd.AddCommand(newEpicSetCmd())
	cmd.AddCommand(newEpicArchiveCmd())

	return cmd
}

func newEpicNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <slug>",
		Short: "Create a new epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if err := models.ValidateSlug(slug); err != nil {
				return err
			}

			initiative, _ := cmd.Flags().GetString("initiative")

			tmpl, err := templates.Load(appStore.Root(), "epic")
			if err != nil {
				return fmt.Errorf("loading template: %w", err)
			}
			if initiative != "" {
				tmpl = strings.Replace(tmpl, `initiative: ""`, fmt.Sprintf("initiative: %q", initiative), 1)
			}

			edited, err := getContent(cmd, tmpl)
			if err != nil {
				return fmt.Errorf("editing epic: %w", err)
			}

			var e models.Epic
			body, err := store.Parse([]byte(edited), &e)
			if err != nil {
				return fmt.Errorf("parsing epic: %w", err)
			}
			e.Slug = slug
			e.Body = body

			if err := appStore.CreateEpic(&e); err != nil {
				return err
			}

			fmt.Printf("Created epic %q (%s)\n", e.Title, e.Slug)
			return nil
		},
	}

	cmd.Flags().String("initiative", "", "Parent initiative slug")

	return cmd
}

func newEpicLsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List epics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			epics, err := appStore.ListEpics()
			if err != nil {
				return err
			}

			includeArchived, _ := cmd.Flags().GetBool("include-archived")
			if includeArchived {
				archived, archErr := appStore.ListArchivedEpics()
				if archErr == nil {
					epics = append(epics, archived...)
				}
			}

			initiative, _ := cmd.Flags().GetString("initiative")
			if initiative != "" {
				var filtered []*models.Epic
				for _, e := range epics {
					if e.Initiative == initiative {
						filtered = append(filtered, e)
					}
				}
				epics = filtered
			}

			headers := []string{"SLUG", "TITLE", "STATUS", "INITIATIVE"}
			rows := make([][]string, len(epics))
			for idx, e := range epics {
				rows[idx] = []string{e.Slug, e.Title, ui.StatusBadge(e.Status), e.Initiative}
			}

			printTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().String("initiative", "", "Filter by initiative slug")
	cmd.Flags().Bool("include-archived", false, "Include archived epics")

	return cmd
}

func newEpicShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <slug>",
		Short: "Show epic details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			e, err := appStore.LoadEpic(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("%s  %s\n", ui.Label("ID:"), e.ID)
			fmt.Printf("%s  %s\n", ui.Label("Slug:"), e.Slug)
			fmt.Printf("%s  %s\n", ui.Label("Title:"), e.Title)
			fmt.Printf("%s  %s\n", ui.Label("Status:"), ui.StatusBadge(e.Status))
			fmt.Printf("%s  %s\n", ui.Label("Initiative:"), e.Initiative)
			if e.Fidelity != "" {
				fmt.Printf("%s  %s\n", ui.Label("Fidelity:"), e.Fidelity)
			}
			fmt.Printf("%s  %s\n", ui.Label("Created:"), e.Created.Format("2006-01-02 15:04:05"))
			fmt.Printf("%s  %s\n", ui.Label("Updated:"), e.Updated.Format("2006-01-02 15:04:05"))

			if len(e.Phases) > 0 {
				fmt.Println("Phases:")
				for _, p := range e.Phases {
					fmt.Printf("  - %s\n", p.Label)
					for _, s := range p.Stories {
						fmt.Printf("      - %s\n", s)
					}
				}
			}
			if len(e.NonGoals) > 0 {
				fmt.Println("Non-Goals:")
				for _, ng := range e.NonGoals {
					fmt.Printf("  - %s\n", ng)
				}
			}
			if len(e.OpenQuestions) > 0 {
				fmt.Println("Open Questions:")
				for _, q := range e.OpenQuestions {
					fmt.Printf("  - %s\n", q)
				}
			}
			if len(e.Decisions) > 0 {
				fmt.Println("Decisions:")
				for _, d := range e.Decisions {
					fmt.Printf("  - %s\n", d)
				}
			}
			if e.Body != "" {
				fmt.Print(ui.RenderMarkdown(e.Body))
			}

			return nil
		},
	}
}

func newEpicEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <slug>",
		Short: "Edit an epic in $EDITOR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]

			e, err := appStore.LoadEpic(slug)
			if err != nil {
				return err
			}

			data, err := store.Marshal(e, e.Body)
			if err != nil {
				return fmt.Errorf("marshaling epic: %w", err)
			}

			edited, err := getContent(cmd, string(data))
			if err != nil {
				return fmt.Errorf("editing epic: %w", err)
			}

			var updated models.Epic
			body, err := store.Parse([]byte(edited), &updated)
			if err != nil {
				return fmt.Errorf("parsing epic: %w", err)
			}

			// Preserve immutable fields from the original.
			updated.ID = e.ID
			updated.Slug = e.Slug
			updated.Created = e.Created
			updated.Body = body

			if err := checkPRDGate(slug, e.Status, updated.Status); err != nil {
				return err
			}

			if err := appStore.SaveEpic(&updated); err != nil {
				return err
			}

			fmt.Printf("Updated epic %q\n", updated.Slug)
			return nil
		},
	}
}

func newEpicSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <slug> <field> <value>",
		Short: "Quick-update a field (status, title, initiative)",
		Args:  cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			slug, field, value := args[0], args[1], args[2]

			e, err := appStore.LoadEpic(slug)
			if err != nil {
				return err
			}

			switch field {
			case "status":
				if !slices.Contains(models.ValidEpicStatuses, value) {
					return fmt.Errorf("invalid status %q: must be one of [%s]", value, strings.Join(models.ValidEpicStatuses, ", "))
				}
				if err := checkPRDGate(slug, e.Status, value); err != nil {
					return err
				}
				e.Status = value
			case "title":
				e.Title = value
			case "initiative":
				e.Initiative = value
			default:
				return fmt.Errorf("unsupported field %q: must be one of [status, title, initiative]", field)
			}

			if err := appStore.SaveEpic(e); err != nil {
				return err
			}

			fmt.Printf("Set %s = %q on epic %q\n", field, value, slug)
			return nil
		},
	}
}

func newEpicArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive <slug>",
		Short: "Archive a completed epic (move to archive, compact files)",
		Long:  "Moves the epic tree to .specflow/archive/, compacts story and epic files to frontmatter-only tombstones, and moves execution directories.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			force, _ := cmd.Flags().GetBool("force")

			summary, err := appStore.ArchiveEpic(slug, force)
			if err != nil {
				return err
			}

			fmt.Printf("Archived epic %q (%d stories, %d executions compacted)\n",
				summary.EpicTitle, summary.StoryCount, summary.ExecutionCount)
			return nil
		},
	}

	cmd.Flags().Bool("force", false, "Archive even if epic/stories aren't in completed/done status")

	return cmd
}

// checkPRDGate returns an error if careful mode requires a PRD before activating an epic.
func checkPRDGate(epicSlug, oldStatus, newStatus string) error {
	if appConfig.Mode != "careful" {
		return nil
	}
	if newStatus != models.EpicStatusActive || oldStatus == models.EpicStatusActive {
		return nil
	}

	docs, err := appStore.ListDocs(epicSlug)
	if err != nil {
		return fmt.Errorf("checking PRD gate: %w", err)
	}
	for _, d := range docs {
		if d.Type == models.DocTypePRD {
			return nil
		}
	}

	return fmt.Errorf("careful mode requires a PRD document for epic %q before activating it.\nCreate one with: specflow doc new <prd-slug> --epic %s --type prd", epicSlug, epicSlug)
}
