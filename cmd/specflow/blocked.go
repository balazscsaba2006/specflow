package main

import (
	"fmt"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/spf13/cobra"
)

func newBlockedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "blocked",
		Short: "List all blocked stories",
		Long:  "Shows stories that have non-empty blocked_by where at least one blocker is not done.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return showBlocked()
		},
	}
}

func showBlocked() error {
	stories, err := appStore.ListAllStories()
	if err != nil {
		return err
	}

	// Build slug→status map for blocker resolution.
	statusBySlug := make(map[string]string, len(stories))
	for _, st := range stories {
		statusBySlug[st.Slug] = st.Status
	}

	// Collect stories that are effectively blocked.
	var blocked []*models.Story
	for _, st := range stories {
		if len(st.BlockedBy) == 0 {
			continue
		}
		// Check if any blocker is not done.
		for _, dep := range st.BlockedBy {
			depStatus, ok := statusBySlug[dep]
			if !ok || depStatus != models.StoryStatusDone {
				blocked = append(blocked, st)
				break
			}
		}
	}

	if len(blocked) == 0 {
		fmt.Println("No blocked stories.")
		return nil
	}

	headers := []string{"SLUG", "TITLE", "STATUS", "EPIC", "BLOCKED BY"}
	rows := make([][]string, len(blocked))
	for idx, st := range blocked {
		// Annotate each blocker with its status.
		blockers := make([]string, len(st.BlockedBy))
		for i, dep := range st.BlockedBy {
			if status, ok := statusBySlug[dep]; ok {
				blockers[i] = fmt.Sprintf("%s (%s)", dep, status)
			} else {
				blockers[i] = fmt.Sprintf("%s (unknown)", dep)
			}
		}
		rows[idx] = []string{
			st.Slug,
			st.Title,
			st.Status,
			st.Epic,
			strings.Join(blockers, ", "),
		}
	}

	printTable(headers, rows)
	fmt.Printf("\n%d blocked story/stories\n", len(blocked))

	return nil
}
