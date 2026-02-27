package export

import (
	"strings"
	"testing"

	"github.com/balazscsaba2006/specflow/internal/models"
)

func makeTestNode() *ExportNode {
	return &ExportNode{
		Type:     NodeEpic,
		Slug:     "test-epic",
		Title:    "Test Epic",
		Status:   "active",
		Fidelity: "beta",
		Body:     "Epic description here.",
		Phases: []models.Phase{
			{Label: "Phase 1", Stories: []string{"story-a", "story-b"}},
		},
		Children: []*ExportNode{
			{
				Type:       NodeStory,
				Slug:       "story-a",
				Title:      "Story A",
				Status:     "planned",
				Priority:   "high",
				Labels:     []string{"backend"},
				Acceptance: []string{"AC1", "AC2"},
				Body:       "Story A body.",
			},
			{
				Type:       NodeStory,
				Slug:       "story-b",
				Title:      "Story B",
				Status:     "done",
				Priority:   "medium",
				Acceptance: []string{"AC3"},
			},
		},
		Docs: []*ExportNode{
			{
				Type:    NodeDoc,
				Slug:    "spec-doc",
				Title:   "Spec Doc",
				Status:  "approved",
				DocType: "tech-spec",
				Body:    "Spec body.",
			},
		},
		Decisions: []*ExportNode{
			{
				Type:   NodeDecision,
				Slug:   "dec-1",
				Title:  "Decision 1",
				Status: "accepted",
				Body:   "## Context\n\nSome context.\n\n## Decision\n\nWe chose X.",
			},
		},
	}
}

func TestMarkdownRenderer(t *testing.T) {
	tests := []struct {
		name     string
		node     *ExportNode
		opts     RenderOptions
		contains []string
		excludes []string
	}{
		{
			name: "full epic with body",
			node: makeTestNode(),
			opts: RenderOptions{IncludeBody: true, IncludeDone: true},
			contains: []string{
				"# Test Epic",
				"**Status:** active",
				"**Fidelity:** beta",
				"Epic description here.",
				"## Story A",
				"**Priority:** high",
				"- [ ] AC1",
				"- [ ] AC2",
				"Story A body.",
				"## Story B",
				"## Documents",
				"### Spec Doc",
				"Spec body.",
				"## Decisions",
				"<details>",
				"Decision 1",
			},
		},
		{
			name: "excludes done stories",
			node: makeTestNode(),
			opts: RenderOptions{IncludeBody: true, IncludeDone: false},
			contains: []string{
				"## Story A",
			},
			excludes: []string{
				"## Story B",
			},
		},
		{
			name: "excludes body when IncludeBody=false",
			node: makeTestNode(),
			opts: RenderOptions{IncludeBody: false, IncludeDone: true},
			excludes: []string{
				"Epic description here.",
				"Story A body.",
				"Spec body.",
			},
			contains: []string{
				"# Test Epic",
				"- [ ] AC1", // acceptance criteria still shown
			},
		},
		{
			name: "title override",
			node: makeTestNode(),
			opts: RenderOptions{IncludeBody: true, IncludeDone: true, Title: "Custom Title"},
			contains: []string{
				"# Custom Title",
			},
		},
		{
			name: "standalone story",
			node: &ExportNode{
				Type:       NodeStory,
				Slug:       "solo",
				Title:      "Solo Story",
				Status:     "planned",
				Priority:   "high",
				Acceptance: []string{"AC1"},
				Body:       "Story body.",
			},
			opts: RenderOptions{IncludeBody: true, IncludeDone: true},
			contains: []string{
				"# Solo Story",
				"**Priority:** high",
				"- [ ] AC1",
				"Story body.",
			},
		},
		{
			name: "initiative node",
			node: &ExportNode{
				Type:   NodeInitiative,
				Slug:   "init-1",
				Title:  "My Initiative",
				Status: "active",
				Goal:   "Build great things",
			},
			opts: RenderOptions{IncludeBody: true, IncludeDone: true},
			contains: []string{
				"# My Initiative",
				"**Goal:** Build great things",
			},
		},
	}

	r := &MarkdownRenderer{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := r.Render(tt.node, tt.opts)
			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}
			content := string(out)

			for _, s := range tt.contains {
				if !strings.Contains(content, s) {
					t.Errorf("output should contain %q\n\nGot:\n%s", s, content)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(content, s) {
					t.Errorf("output should NOT contain %q", s)
				}
			}
		})
	}
}

func TestYAMLRenderer(t *testing.T) {
	tests := []struct {
		name     string
		node     *ExportNode
		opts     RenderOptions
		contains []string
		excludes []string
	}{
		{
			name: "epic produces legacy format",
			node: makeTestNode(),
			opts: RenderOptions{IncludeBody: true, IncludeDone: true},
			contains: []string{
				"epic:",
				"slug: test-epic",
				"title: Test Epic",
				"stories:",
				"slug: story-a",
				"slug: story-b",
				"acceptance:",
				"- AC1",
			},
		},
		{
			name: "epic excludes done stories",
			node: makeTestNode(),
			opts: RenderOptions{IncludeBody: true, IncludeDone: false},
			contains: []string{
				"slug: story-a",
			},
			excludes: []string{
				"slug: story-b",
			},
		},
		{
			name: "standalone story produces flat YAML",
			node: &ExportNode{
				Type:       NodeStory,
				Slug:       "solo",
				Title:      "Solo Story",
				Status:     "planned",
				Priority:   "high",
				Acceptance: []string{"AC1"},
			},
			opts: RenderOptions{IncludeBody: true, IncludeDone: true},
			contains: []string{
				"slug: solo",
				"title: Solo Story",
				"priority: high",
			},
		},
		{
			name: "initiative uses generic format",
			node: &ExportNode{
				Type:   NodeInitiative,
				Slug:   "init-1",
				Title:  "My Initiative",
				Status: "active",
				Goal:   "Build things",
				Children: []*ExportNode{
					{Type: NodeEpic, Slug: "ep1", Title: "Epic 1", Status: "active"},
				},
			},
			opts: RenderOptions{IncludeBody: true, IncludeDone: true},
			contains: []string{
				"type: initiative",
				"slug: init-1",
				"goal: Build things",
				"children:",
			},
		},
	}

	r := &YAMLRenderer{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := r.Render(tt.node, tt.opts)
			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}
			content := string(out)

			for _, s := range tt.contains {
				if !strings.Contains(content, s) {
					t.Errorf("output should contain %q\n\nGot:\n%s", s, content)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(content, s) {
					t.Errorf("output should NOT contain %q", s)
				}
			}
		})
	}
}

func TestHTMLRenderer(t *testing.T) {
	tests := []struct {
		name     string
		node     *ExportNode
		opts     RenderOptions
		contains []string
	}{
		{
			name: "produces valid HTML with mermaid and hljs",
			node: makeTestNode(),
			opts: RenderOptions{IncludeBody: true, IncludeDone: true},
			contains: []string{
				"<!DOCTYPE html>",
				"<title>Test Epic</title>",
				"mermaid",
				"highlight.js",
				"Table of Contents",
				"Test Epic",
				"Story A",
				"Generated by specflow",
			},
		},
		{
			name: "includes mermaid diagram",
			node: &ExportNode{
				Type:   NodeEpic,
				Slug:   "mermaid-test",
				Title:  "Mermaid Test",
				Status: "active",
				Body:   "```mermaid\nflowchart LR\n    A --> B\n```",
			},
			opts: RenderOptions{IncludeBody: true, IncludeDone: true},
			contains: []string{
				`class="mermaid"`,
				"A --&gt; B",
			},
		},
		{
			name: "title override in HTML",
			node: makeTestNode(),
			opts: RenderOptions{IncludeBody: true, IncludeDone: true, Title: "Custom Title"},
			contains: []string{
				"<title>Custom Title</title>",
			},
		},
	}

	r := &HTMLRenderer{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := r.Render(tt.node, tt.opts)
			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}
			content := string(out)

			for _, s := range tt.contains {
				if !strings.Contains(content, s) {
					t.Errorf("output should contain %q\n\nGot (first 2000 chars):\n%.2000s", s, content)
				}
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Story A (planned)", "story-a-planned"},
		{"test--double", "test-double"},
		{"  leading/trailing  ", "leadingtrailing"},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
