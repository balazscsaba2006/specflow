package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search across all specflow artifacts",
		Long:  "Walks all .md files in .specflow/ and searches for the query string, returning matches with context.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contextLines, _ := cmd.Flags().GetInt("context")
			return runSearch(args[0], contextLines)
		},
	}

	cmd.Flags().IntP("context", "C", 1, "Number of context lines around each match")

	return cmd
}

type searchMatch struct {
	File    string
	Line    int
	Content string
	Context []string // lines around the match
}

func runSearch(query string, contextLines int) error {
	root := appStore.Root()
	queryLower := strings.ToLower(query)

	var matches []searchMatch

	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip files we can't access
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		fileMatches, err := searchFile(path, queryLower, contextLines)
		if err != nil {
			return nil // skip files we can't read
		}

		matches = append(matches, fileMatches...)
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("walking .specflow directory: %w", walkErr)
	}

	if len(matches) == 0 {
		fmt.Printf("No matches found for %q\n", query)
		return nil
	}

	// Print matches grouped by file.
	currentFile := ""
	for _, m := range matches {
		relPath, _ := filepath.Rel(root, m.File)
		if relPath == "" {
			relPath = m.File
		}

		if relPath != currentFile {
			if currentFile != "" {
				fmt.Println()
			}
			fmt.Printf("--- %s ---\n", relPath)
			currentFile = relPath
		}

		for _, line := range m.Context {
			fmt.Println(line)
		}
		if len(m.Context) > 0 {
			fmt.Println()
		}
	}

	fmt.Printf("%d match(es) in %d file(s)\n", len(matches), countUniqueFiles(matches))

	return nil
}

func searchFile(path, queryLower string, contextLines int) ([]searchMatch, error) {
	f, err := os.Open(path) //nolint:gosec // reading own project files
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var matches []searchMatch
	for i, line := range lines {
		if !strings.Contains(strings.ToLower(line), queryLower) {
			continue
		}

		// Build context window.
		start := i - contextLines
		if start < 0 {
			start = 0
		}
		end := i + contextLines + 1
		if end > len(lines) {
			end = len(lines)
		}

		ctx := make([]string, 0, end-start)
		for j := start; j < end; j++ {
			prefix := "  "
			if j == i {
				prefix = "> "
			}
			ctx = append(ctx, fmt.Sprintf("%s%4d: %s", prefix, j+1, lines[j]))
		}

		matches = append(matches, searchMatch{
			File:    path,
			Line:    i + 1,
			Content: line,
			Context: ctx,
		})
	}

	return matches, nil
}

func countUniqueFiles(matches []searchMatch) int {
	seen := make(map[string]struct{})
	for _, m := range matches {
		seen[m.File] = struct{}{}
	}
	return len(seen)
}
