package main

import (
	"fmt"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
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

	fmt.Println("=== Project Status ===")
	fmt.Println()

	if total > 0 {
		pct := float64(done) / float64(total) * 100
		fmt.Printf("Stories: %d total, %d done (%.0f%%)\n", total, done, pct)
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

			progress := "—"
			if epicTotal > 0 {
				pct := float64(epicDone) / float64(epicTotal) * 100
				progress = fmt.Sprintf("%.0f%%", pct)
			}

			rows = append(rows, []string{
				e.Slug,
				e.Status,
				fmt.Sprintf("%d", epicTotal),
				fmt.Sprintf("%d", epicDone),
				progress,
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
		fmt.Printf("\nStandalone stories: %d total, %d done\n", len(standalone), standaloneDone)
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
	fmt.Printf("Type:       Initiative\n")
	fmt.Printf("Slug:       %s\n", i.Slug)
	fmt.Printf("Title:      %s\n", i.Title)
	fmt.Printf("Status:     %s\n", i.Status)
	fmt.Printf("Goal:       %s\n", i.Goal)
	if len(i.Epics) > 0 {
		fmt.Printf("Epics:      %s\n", strings.Join(i.Epics, ", "))
	}
	if len(i.OpenQuestions) > 0 {
		fmt.Printf("Open Qs:    %d\n", len(i.OpenQuestions))
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

	fmt.Printf("Type:       Epic\n")
	fmt.Printf("Slug:       %s\n", e.Slug)
	fmt.Printf("Title:      %s\n", e.Title)
	fmt.Printf("Status:     %s\n", e.Status)
	if e.Initiative != "" {
		fmt.Printf("Initiative: %s\n", e.Initiative)
	}

	total := len(stories)
	if total > 0 {
		pct := float64(done) / float64(total) * 100
		fmt.Printf("Stories:    %d total, %d done (%.0f%%)\n", total, done, pct)
		printStatusCounts(counts)
	} else {
		fmt.Println("Stories:    none")
	}

	if len(e.OpenQuestions) > 0 {
		fmt.Printf("Open Qs:    %d\n", len(e.OpenQuestions))
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

		fmt.Printf("Type:       Story\n")
		fmt.Printf("Slug:       %s\n", st.Slug)
		fmt.Printf("Title:      %s\n", st.Title)
		fmt.Printf("Status:     %s\n", st.Status)
		fmt.Printf("Priority:   %s\n", st.Priority)
		if st.Epic != "" {
			fmt.Printf("Epic:       %s\n", st.Epic)
		}
		if len(st.BlockedBy) > 0 {
			fmt.Printf("Blocked By: %s\n", strings.Join(st.BlockedBy, ", "))
		}
		if len(st.OpenQuestions) > 0 {
			fmt.Printf("Open Qs:    %d\n", len(st.OpenQuestions))
		}
		if len(st.Assumptions) > 0 {
			fmt.Printf("Assumptions: %d\n", len(st.Assumptions))
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
			parts = append(parts, fmt.Sprintf("%s:%d", s, c))
		}
	}

	if len(parts) > 0 {
		fmt.Printf("  %s\n", strings.Join(parts, "  "))
	}
}
