package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newQuestionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "questions",
		Short: "List all open questions across the project",
		Long:  "Walks all initiatives, epics, and stories to collect open questions, grouped by source.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return showQuestions()
		},
	}
}

type questionEntry struct {
	Source   string // e.g. "epic:auth-system" or "story:jwt-middleware"
	Question string
}

func showQuestions() error {
	var entries []questionEntry

	// Initiatives.
	initiatives, err := appStore.ListInitiatives()
	if err != nil {
		return err
	}
	for _, i := range initiatives {
		for _, q := range i.OpenQuestions {
			entries = append(entries, questionEntry{
				Source:   "initiative:" + i.Slug,
				Question: q,
			})
		}
	}

	// Epics.
	epics, err := appStore.ListEpics()
	if err != nil {
		return err
	}
	for _, e := range epics {
		for _, q := range e.OpenQuestions {
			entries = append(entries, questionEntry{
				Source:   "epic:" + e.Slug,
				Question: q,
			})
		}
	}

	// Stories (all).
	stories, err := appStore.ListAllStories()
	if err != nil {
		return err
	}
	for _, st := range stories {
		for _, q := range st.OpenQuestions {
			entries = append(entries, questionEntry{
				Source:   "story:" + st.Slug,
				Question: q,
			})
		}
	}

	if len(entries) == 0 {
		fmt.Println("No open questions found.")
		return nil
	}

	headers := []string{"SOURCE", "QUESTION"}
	rows := make([][]string, len(entries))
	for idx, e := range entries {
		rows[idx] = []string{e.Source, e.Question}
	}

	printTable(headers, rows)
	fmt.Printf("\n%d open question(s)\n", len(entries))

	return nil
}
