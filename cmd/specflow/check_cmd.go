package main

import (
	"fmt"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/spf13/cobra"
)

func newScopeCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scope-check <epic-slug>",
		Short: "Compare stories against PRD scope definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			epicSlug := args[0]

			// Find PRD for this epic.
			prd, err := findPRD(epicSlug)
			if err != nil {
				return fmt.Errorf("no PRD found for epic %q: %w", epicSlug, err)
			}

			stories, err := appStore.ListStories(epicSlug)
			if err != nil {
				return fmt.Errorf("listing stories: %w", err)
			}

			inScope, outScope, userStories := parseScopeSections(prd.Body)
			untraced := findUntracedStories(stories, userStories, inScope)
			outOfScope := findOutOfScopeStories(stories, outScope)

			fmt.Printf("Scope Check for epic %q (PRD: %s)\n\n", epicSlug, prd.Slug)

			if len(untraced) == 0 && len(outOfScope) == 0 {
				fmt.Println("All stories are traceable to the PRD scope. No issues found.")
				return nil
			}

			if len(untraced) > 0 {
				fmt.Println("Stories not traceable to PRD:")
				for _, st := range untraced {
					fmt.Printf("  - %s — %s [%s]\n", st.Slug, st.Title, st.Status)
				}
				fmt.Println()
			}

			if len(outOfScope) > 0 {
				fmt.Println("Stories matching out-of-scope:")
				for _, st := range outOfScope {
					fmt.Printf("  - %s — %s [%s]\n", st.Slug, st.Title, st.Status)
				}
			}

			return nil
		},
	}
}

func newDiffCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff-check <epic-slug>",
		Short: "Detect drift between specs and stories",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			epicSlug := args[0]

			stories, err := appStore.ListStories(epicSlug)
			if err != nil {
				return fmt.Errorf("listing stories: %w", err)
			}

			// Load all docs for the epic + project-level.
			docMap := make(map[string]*models.Document)
			if epicDocs, listErr := appStore.ListDocs(epicSlug); listErr == nil {
				for _, d := range epicDocs {
					docMap[d.Slug] = d
				}
			}
			if projDocs, listErr := appStore.ListDocs(""); listErr == nil {
				for _, d := range projDocs {
					docMap[d.Slug] = d
				}
			}

			type driftItem struct {
				storySlug string
				docSlug   string
				docType   string
				storyTS   string
				docTS     string
			}
			var drifted []driftItem

			for _, st := range stories {
				for _, ref := range st.DocRefs {
					doc, exists := docMap[ref]
					if !exists {
						continue
					}
					if doc.Updated.After(st.Updated) {
						drifted = append(drifted, driftItem{
							storySlug: st.Slug,
							docSlug:   doc.Slug,
							docType:   doc.Type,
							storyTS:   st.Updated.Format("2006-01-02 15:04"),
							docTS:     doc.Updated.Format("2006-01-02 15:04"),
						})
					}
				}
			}

			fmt.Printf("Drift Check for epic %q\n\n", epicSlug)

			if len(drifted) == 0 {
				fmt.Println("No drift detected. All stories are up-to-date with their referenced documents.")
				return nil
			}

			fmt.Println("Stories with potential drift:")
			for _, d := range drifted {
				fmt.Printf("  - %s references %s (%s) — story: %s, doc: %s\n",
					d.storySlug, d.docSlug, d.docType, d.storyTS, d.docTS)
			}
			fmt.Printf("\n%d potentially drifted stories\n", len(drifted))

			return nil
		},
	}
}

// --- Helpers (same logic as MCP handlers) ---

func findPRD(epicSlug string) (*models.Document, error) {
	docs, err := appStore.ListDocs(epicSlug)
	if err == nil {
		for _, d := range docs {
			if d.Type == models.DocTypePRD {
				return d, nil
			}
		}
	}
	docs, err = appStore.ListDocs("")
	if err != nil {
		return nil, fmt.Errorf("listing project docs: %w", err)
	}
	for _, d := range docs {
		if d.Type == models.DocTypePRD {
			return d, nil
		}
	}
	return nil, fmt.Errorf("no PRD document found")
}

func parseScopeSections(body string) (inScope, outScope string, userStories []string) {
	lines := strings.Split(body, "\n")
	var currentSection string
	var sectionLines []string

	flushSection := func() {
		content := strings.TrimSpace(strings.Join(sectionLines, "\n"))
		switch {
		case strings.Contains(strings.ToLower(currentSection), "in scope") && !strings.Contains(strings.ToLower(currentSection), "out"):
			inScope = content
		case strings.Contains(strings.ToLower(currentSection), "out of scope") || strings.Contains(strings.ToLower(currentSection), "out-of-scope"):
			outScope = content
		case strings.Contains(strings.ToLower(currentSection), "user stor"):
			for _, line := range sectionLines {
				trimmed := strings.TrimSpace(line)
				trimmed = strings.TrimLeft(trimmed, "-*• ")
				if trimmed != "" {
					userStories = append(userStories, trimmed)
				}
			}
		}
		sectionLines = nil
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			flushSection()
			currentSection = line
			continue
		}
		sectionLines = append(sectionLines, line)
	}
	flushSection()

	return inScope, outScope, userStories
}

func findUntracedStories(stories []*models.Story, userStories []string, inScope string) []*models.Story {
	inScopeLower := strings.ToLower(inScope)
	var untraced []*models.Story
	for _, st := range stories {
		titleLower := strings.ToLower(st.Title)
		matched := false
		for _, us := range userStories {
			usLower := strings.ToLower(us)
			if strings.Contains(usLower, titleLower) || strings.Contains(titleLower, usLower) {
				matched = true
				break
			}
		}
		if !matched && !strings.Contains(inScopeLower, titleLower) {
			untraced = append(untraced, st)
		}
	}
	return untraced
}

func findOutOfScopeStories(stories []*models.Story, outScope string) []*models.Story {
	if outScope == "" {
		return nil
	}
	outLower := strings.ToLower(outScope)
	var result []*models.Story
	for _, st := range stories {
		if strings.Contains(outLower, strings.ToLower(st.Title)) {
			result = append(result, st)
		}
	}
	return result
}
