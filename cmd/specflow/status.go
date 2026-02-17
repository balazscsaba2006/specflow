package main

import (
	"fmt"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/ui"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [slug]",
		Short: "Show project or entity status",
		Long: `Without arguments, shows an aggregate status rollup across all epics and stories.
With a slug argument, auto-detects the entity type (initiative, epic, or story) and shows its detail.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 1 {
				return showEntityStatus(args[0])
			}
			return showProjectStatus()
		},
	}
}

func showProjectStatus() error {
	epics, err := appStore.ListEpics()
	if err != nil {
		return err
	}

	allStories, err := appStore.ListAllStories()
	if err != nil {
		return err
	}

	if len(epics) == 0 && len(allStories) == 0 {
		fmt.Println("No epics or stories found.")
		return nil
	}

	// Overall story counts.
	total := len(allStories)
	counts := countByStatus(allStories)
	done := counts[models.StoryStatusDone]

	fmt.Println(ui.Header("Project Status"))
	fmt.Println()

	if total > 0 {
		fmt.Printf("%s  %s\n", ui.Label("Stories:"), fmt.Sprintf("%d total, %d done", total, done))
		fmt.Printf("%s  %s\n", ui.Label("Progress:"), ui.ProgressBar(done, total, 20))
		printStatusCounts(counts)
		fmt.Println()
	}

	// Per-epic breakdown.
	if len(epics) > 0 {
		headers := []string{"EPIC", "STATUS", "STORIES", "DONE", "PROGRESS"}
		rows := make([][]string, 0, len(epics))

		for _, e := range epics {
			epicStories, listErr := appStore.ListStories(e.Slug)
			if listErr != nil {
				return listErr
			}
			epicTotal := len(epicStories)
			epicCounts := countByStatus(epicStories)
			epicDone := epicCounts[models.StoryStatusDone]

			rows = append(rows, []string{
				e.Slug,
				ui.StatusBadge(e.Status),
				fmt.Sprintf("%d", epicTotal),
				fmt.Sprintf("%d", epicDone),
				ui.ProgressBar(epicDone, epicTotal, 15),
			})
		}

		printTable(headers, rows)
	}

	// Standalone stories summary.
	standalone, err := appStore.ListStories("")
	if err != nil {
		return err
	}
	if len(standalone) > 0 {
		standaloneCount := countByStatus(standalone)
		standaloneDone := standaloneCount[models.StoryStatusDone]
		fmt.Printf("\n%s  %d total, %d done  %s\n", ui.Label("Standalone:"), len(standalone), standaloneDone, ui.ProgressBar(standaloneDone, len(standalone), 15))
	}

	return nil
}

func showEntityStatus(slug string) error {
	// Try initiative first.
	if i, err := appStore.LoadInitiative(slug); err == nil {
		return printInitiativeStatus(i)
	}

	// Try epic.
	if e, err := appStore.LoadEpic(slug); err == nil {
		return printEpicStatus(e)
	}

	// Try story — search all epics + standalone.
	return printStoryStatus(slug)
}

func printInitiativeStatus(i *models.Initiative) error {
	fmt.Printf("%s  Initiative\n", ui.Label("Type:"))
	fmt.Printf("%s  %s\n", ui.Label("Slug:"), i.Slug)
	fmt.Printf("%s  %s\n", ui.Label("Title:"), i.Title)
	fmt.Printf("%s  %s\n", ui.Label("Status:"), ui.StatusBadge(i.Status))
	fmt.Printf("%s  %s\n", ui.Label("Goal:"), i.Goal)
	if len(i.Epics) > 0 {
		fmt.Printf("%s  %s\n", ui.Label("Epics:"), strings.Join(i.Epics, ", "))
	}
	if len(i.OpenQuestions) > 0 {
		fmt.Printf("%s  %d\n", ui.Label("Open Qs:"), len(i.OpenQuestions))
	}
	return nil
}

func printEpicStatus(e *models.Epic) error {
	stories, err := appStore.ListStories(e.Slug)
	if err != nil {
		return err
	}
	counts := countByStatus(stories)
	done := counts[models.StoryStatusDone]

	fmt.Printf("%s  Epic\n", ui.Label("Type:"))
	fmt.Printf("%s  %s\n", ui.Label("Slug:"), e.Slug)
	fmt.Printf("%s  %s\n", ui.Label("Title:"), e.Title)
	fmt.Printf("%s  %s\n", ui.Label("Status:"), ui.StatusBadge(e.Status))
	if e.Initiative != "" {
		fmt.Printf("%s  %s\n", ui.Label("Initiative:"), e.Initiative)
	}

	total := len(stories)
	if total > 0 {
		fmt.Printf("%s  %d total, %d done\n", ui.Label("Stories:"), total, done)
		fmt.Printf("%s  %s\n", ui.Label("Progress:"), ui.ProgressBar(done, total, 20))
		printStatusCounts(counts)
	} else {
		fmt.Printf("%s  none\n", ui.Label("Stories:"))
	}

	if len(e.OpenQuestions) > 0 {
		fmt.Printf("%s  %d\n", ui.Label("Open Qs:"), len(e.OpenQuestions))
	}
	return nil
}

func printStoryStatus(slug string) error {
	allStories, err := appStore.ListAllStories()
	if err != nil {
		return err
	}

	for _, st := range allStories {
		if st.Slug != slug {
			continue
		}

		fmt.Printf("%s  Story\n", ui.Label("Type:"))
		fmt.Printf("%s  %s\n", ui.Label("Slug:"), st.Slug)
		fmt.Printf("%s  %s\n", ui.Label("Title:"), st.Title)
		fmt.Printf("%s  %s\n", ui.Label("Status:"), ui.StatusBadge(st.Status))
		fmt.Printf("%s  %s\n", ui.Label("Priority:"), ui.PriorityBadge(st.Priority))
		if st.Epic != "" {
			fmt.Printf("%s  %s\n", ui.Label("Epic:"), st.Epic)
		}
		if len(st.BlockedBy) > 0 {
			fmt.Printf("%s  %s\n", ui.Label("Blocked By:"), strings.Join(st.BlockedBy, ", "))
		}
		if len(st.OpenQuestions) > 0 {
			fmt.Printf("%s  %d\n", ui.Label("Open Qs:"), len(st.OpenQuestions))
		}
		if len(st.Assumptions) > 0 {
			fmt.Printf("%s  %d\n", ui.Label("Assumptions:"), len(st.Assumptions))
		}
		return nil
	}

	return fmt.Errorf("no initiative, epic, or story found with slug %q", slug)
}

func countByStatus(stories []*models.Story) map[string]int {
	counts := make(map[string]int)
	for _, st := range stories {
		counts[st.Status]++
	}
	return counts
}

func printStatusCounts(counts map[string]int) {
	order := []string{
		models.StoryStatusDraft,
		models.StoryStatusPlanned,
		models.StoryStatusInProgress,
		models.StoryStatusVerifying,
		models.StoryStatusDone,
		models.StoryStatusBlocked,
	}

	parts := make([]string, 0, len(order))
	for _, s := range order {
		if c, ok := counts[s]; ok && c > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", ui.StatusBadge(s), c))
		}
	}

	if len(parts) > 0 {
		fmt.Printf("  %s\n", strings.Join(parts, "  "))
	}
}
