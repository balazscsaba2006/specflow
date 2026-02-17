package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/balazscsaba2006/specflow/internal/ui"
	"github.com/spf13/cobra"
)

const storyTemplate = `---
title: ""
status: draft
priority: medium
epic: ""
blocked_by: []
labels: []
acceptance: []
doc_refs: []
open_questions: []
assumptions: []
---
# Story Title

Description of this story...
`

func newStoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "story",
		Aliases: []string{"s"},
		Short:   "Manage stories",
		Long:    "Create, list, show, edit, and update stories.",
	}

	cmd.AddCommand(newStoryNewCmd())
	cmd.AddCommand(newStoryLsCmd())
	cmd.AddCommand(newStoryShowCmd())
	cmd.AddCommand(newStoryEditCmd())
	cmd.AddCommand(newStorySetCmd())
	cmd.AddCommand(newStoryNextCmd())

	return cmd
}

func newStoryNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <slug>",
		Short: "Create a new story",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if err := models.ValidateSlug(slug); err != nil {
				return err
			}

			epicSlug, _ := cmd.Flags().GetString("epic")

			tmpl := storyTemplate
			if epicSlug != "" {
				tmpl = strings.Replace(tmpl, `epic: ""`, fmt.Sprintf("epic: %q", epicSlug), 1)
			}

			edited, err := openInEditor(tmpl)
			if err != nil {
				return fmt.Errorf("editing story: %w", err)
			}

			var st models.Story
			body, err := store.Parse([]byte(edited), &st)
			if err != nil {
				return fmt.Errorf("parsing story: %w", err)
			}
			st.Slug = slug
			st.Body = body

			if err := appStore.CreateStory(&st); err != nil {
				return err
			}

			fmt.Printf("Created story %q (%s)\n", st.Title, st.Slug)
			return nil
		},
	}

	cmd.Flags().String("epic", "", "Parent epic slug")

	return cmd
}

func newStoryLsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List stories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			epicSlug, _ := cmd.Flags().GetString("epic")
			statusFilter, _ := cmd.Flags().GetString("status")
			labelFilter, _ := cmd.Flags().GetString("label")
			blockedOnly, _ := cmd.Flags().GetBool("blocked")

			var stories []*models.Story
			var err error

			if epicSlug != "" {
				stories, err = appStore.ListStories(epicSlug)
			} else {
				stories, err = appStore.ListStories("")
			}
			if err != nil {
				return err
			}

			// Apply filters.
			var filtered []*models.Story
			for _, st := range stories {
				if statusFilter != "" && st.Status != statusFilter {
					continue
				}
				if labelFilter != "" && !slices.Contains(st.Labels, labelFilter) {
					continue
				}
				if blockedOnly && len(st.BlockedBy) == 0 {
					continue
				}
				filtered = append(filtered, st)
			}

			headers := []string{"SLUG", "TITLE", "STATUS", "PRIORITY", "EPIC", "LABELS"}
			rows := make([][]string, len(filtered))
			for idx, st := range filtered {
				rows[idx] = []string{
					st.Slug,
					st.Title,
					ui.StatusBadge(st.Status),
					ui.PriorityBadge(st.Priority),
					st.Epic,
					strings.Join(st.Labels, ", "),
				}
			}

			printTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().String("epic", "", "Filter by epic slug")
	cmd.Flags().String("status", "", "Filter by status")
	cmd.Flags().String("label", "", "Filter by label")
	cmd.Flags().Bool("blocked", false, "Only show stories with non-empty blocked_by")

	return cmd
}

func newStoryShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <slug>",
		Short: "Show story details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			epicSlug, _ := cmd.Flags().GetString("epic")

			st, err := appStore.LoadStory(args[0], epicSlug)
			if err != nil {
				return err
			}

			fmt.Printf("%s  %s\n", ui.Label("ID:"), st.ID)
			fmt.Printf("%s  %s\n", ui.Label("Slug:"), st.Slug)
			fmt.Printf("%s  %s\n", ui.Label("Title:"), st.Title)
			fmt.Printf("%s  %s\n", ui.Label("Status:"), ui.StatusBadge(st.Status))
			fmt.Printf("%s  %s\n", ui.Label("Priority:"), ui.PriorityBadge(st.Priority))
			fmt.Printf("%s  %s\n", ui.Label("Epic:"), st.Epic)
			fmt.Printf("%s  %s\n", ui.Label("Created:"), st.Created.Format("2006-01-02 15:04:05"))
			fmt.Printf("%s  %s\n", ui.Label("Updated:"), st.Updated.Format("2006-01-02 15:04:05"))

			if len(st.BlockedBy) > 0 {
				fmt.Println("Blocked By:")
				for _, b := range st.BlockedBy {
					fmt.Printf("  - %s\n", b)
				}
			}
			if len(st.Labels) > 0 {
				fmt.Println("Labels:")
				for _, l := range st.Labels {
					fmt.Printf("  - %s\n", l)
				}
			}
			if len(st.Acceptance) > 0 {
				fmt.Println("Acceptance Criteria:")
				for _, a := range st.Acceptance {
					fmt.Printf("  - %s\n", a)
				}
			}
			if len(st.DocRefs) > 0 {
				fmt.Println("Doc Refs:")
				for _, d := range st.DocRefs {
					fmt.Printf("  - %s\n", d)
				}
			}
			if len(st.OpenQuestions) > 0 {
				fmt.Println("Open Questions:")
				for _, q := range st.OpenQuestions {
					fmt.Printf("  - %s\n", q)
				}
			}
			if len(st.Assumptions) > 0 {
				fmt.Println("Assumptions:")
				for _, a := range st.Assumptions {
					fmt.Printf("  - %s\n", a)
				}
			}
			if st.Body != "" {
				fmt.Printf("\n%s\n", st.Body)
			}

			return nil
		},
	}

	cmd.Flags().String("epic", "", "Epic slug (for epic-scoped stories)")

	return cmd
}

func newStoryEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <slug>",
		Short: "Edit a story in $EDITOR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			epicSlug, _ := cmd.Flags().GetString("epic")

			st, err := appStore.LoadStory(slug, epicSlug)
			if err != nil {
				return err
			}

			data, err := store.Marshal(st, st.Body)
			if err != nil {
				return fmt.Errorf("marshaling story: %w", err)
			}

			edited, err := openInEditor(string(data))
			if err != nil {
				return fmt.Errorf("editing story: %w", err)
			}

			var updated models.Story
			body, err := store.Parse([]byte(edited), &updated)
			if err != nil {
				return fmt.Errorf("parsing story: %w", err)
			}

			// Preserve immutable fields from the original.
			updated.ID = st.ID
			updated.Slug = st.Slug
			updated.Created = st.Created
			updated.Body = body

			if err := appStore.SaveStory(&updated); err != nil {
				return err
			}

			fmt.Printf("Updated story %q\n", updated.Slug)
			return nil
		},
	}

	cmd.Flags().String("epic", "", "Epic slug (for epic-scoped stories)")

	return cmd
}

func newStorySetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <slug> <field> <value>",
		Short: "Quick-update a field (status, priority, title)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, field, value := args[0], args[1], args[2]
			epicSlug, _ := cmd.Flags().GetString("epic")

			switch field {
			case "status":
				if !slices.Contains(models.ValidStoryStatuses, value) {
					return fmt.Errorf("invalid status %q: must be one of [%s]", value, strings.Join(models.ValidStoryStatuses, ", "))
				}
				if err := appStore.UpdateStoryStatus(slug, epicSlug, value); err != nil {
					return err
				}
			case "priority":
				if !slices.Contains(models.ValidPriorities, value) {
					return fmt.Errorf("invalid priority %q: must be one of [%s]", value, strings.Join(models.ValidPriorities, ", "))
				}
				st, err := appStore.LoadStory(slug, epicSlug)
				if err != nil {
					return err
				}
				st.Priority = value
				if err := appStore.SaveStory(st); err != nil {
					return err
				}
			case "title":
				st, err := appStore.LoadStory(slug, epicSlug)
				if err != nil {
					return err
				}
				st.Title = value
				if err := appStore.SaveStory(st); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported field %q: must be one of [status, priority, title]", field)
			}

			fmt.Printf("Set %s = %q on story %q\n", field, value, slug)
			return nil
		},
	}

	cmd.Flags().String("epic", "", "Epic slug (for epic-scoped stories)")

	return cmd
}

// priorityRank maps priority strings to sort order (lower = higher priority).
var priorityRank = map[string]int{
	models.PriorityCritical: 0,
	models.PriorityHigh:     1,
	models.PriorityMedium:   2,
	models.PriorityLow:      3,
}

func newStoryNextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next",
		Short: "Recommend the next story to work on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			epicSlug, _ := cmd.Flags().GetString("epic")

			var stories []*models.Story
			var err error

			if epicSlug != "" {
				stories, err = appStore.ListStories(epicSlug)
			} else {
				stories, err = appStore.ListAllStories()
			}
			if err != nil {
				return err
			}

			// Filter to planned stories with no blockers (or all blockers done).
			var candidates []*models.Story
			for _, st := range stories {
				if st.Status != models.StoryStatusPlanned {
					continue
				}
				if len(st.BlockedBy) > 0 && !allBlockersDone(st.BlockedBy, stories) {
					continue
				}
				candidates = append(candidates, st)
			}

			if len(candidates) == 0 {
				fmt.Println("No actionable stories found.")
				return nil
			}

			// Sort by priority (critical > high > medium > low).
			slices.SortFunc(candidates, func(a, b *models.Story) int {
				return priorityRank[a.Priority] - priorityRank[b.Priority]
			})

			next := candidates[0]
			fmt.Printf("Slug:      %s\n", next.Slug)
			fmt.Printf("Title:     %s\n", next.Title)
			fmt.Printf("Priority:  %s\n", next.Priority)
			if next.Epic != "" {
				fmt.Printf("Epic:      %s\n", next.Epic)
			}
			if len(next.Labels) > 0 {
				fmt.Printf("Labels:    %s\n", strings.Join(next.Labels, ", "))
			}

			return nil
		},
	}

	cmd.Flags().String("epic", "", "Scope to a specific epic")

	return cmd
}

// allBlockersDone checks if all slugs in blockedBy correspond to stories
// with status "done" in the provided stories list. If a blocker slug is not
// found among the loaded stories, it's considered not done.
func allBlockersDone(blockedBy []string, stories []*models.Story) bool {
	storyBySlug := make(map[string]*models.Story, len(stories))
	for _, st := range stories {
		storyBySlug[st.Slug] = st
	}

	for _, slug := range blockedBy {
		blocker, ok := storyBySlug[slug]
		if !ok || blocker.Status != models.StoryStatusDone {
			return false
		}
	}
	return true
}
