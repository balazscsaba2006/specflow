package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAssumptionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "assumptions",
		Short: "List all assumptions across stories",
		Long:  "Walks all stories and collects assumptions, grouped by epic.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return showAssumptions()
		},
	}
}

func showAssumptions() error {
	stories, err := appStore.ListAllStories()
	if err != nil {
		return err
	}

	type assumptionEntry struct {
		Epic       string
		Story      string
		Assumption string
	}

	var entries []assumptionEntry
	for _, st := range stories {
		epic := st.Epic
		if epic == "" {
			epic = "(standalone)"
		}
		for _, a := range st.Assumptions {
			entries = append(entries, assumptionEntry{
				Epic:       epic,
				Story:      st.Slug,
				Assumption: a,
			})
		}
	}

	if len(entries) == 0 {
		fmt.Println("No assumptions found.")
		return nil
	}

	headers := []string{"EPIC", "STORY", "ASSUMPTION"}
	rows := make([][]string, len(entries))
	for idx, e := range entries {
		rows[idx] = []string{e.Epic, e.Story, e.Assumption}
	}

	printTable(headers, rows)
	fmt.Printf("\n%d assumption(s)\n", len(entries))

	return nil
}
