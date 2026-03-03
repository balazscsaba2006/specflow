package export

import (
	"testing"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
)

func createInitiative(t *testing.T, s *store.Store, slug, title, goal string, epics []string) {
	t.Helper()
	i := &models.Initiative{
		Slug:  slug,
		Title: title,
		Goal:  goal,
		Epics: epics,
	}
	if err := s.CreateInitiative(i); err != nil {
		t.Fatalf("CreateInitiative(%q) error: %v", slug, err)
	}
}

func createDoc(t *testing.T, s *store.Store, slug, title, docType, epicSlug, body string) {
	t.Helper()
	d := &models.Document{
		Slug:  slug,
		Type:  docType,
		Title: title,
		Epic:  epicSlug,
		Body:  body,
	}
	if err := s.CreateDoc(d); err != nil {
		t.Fatalf("CreateDoc(%q) error: %v", slug, err)
	}
}

func createDecision(t *testing.T, s *store.Store, slug, title, body string, contextRefs []string) {
	t.Helper()
	d := &models.Decision{
		Slug:        slug,
		Title:       title,
		ContextRefs: contextRefs,
		Body:        body,
	}
	if err := s.CreateDecision(d); err != nil {
		t.Fatalf("CreateDecision(%q) error: %v", slug, err)
	}
}

func TestExtractEpicNode(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, s *store.Store)
		slug    string
		opts    ExtractOptions
		wantErr bool
		check   func(t *testing.T, node *ExportNode)
	}{
		{
			name: "epic without tree returns leaf node",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "my-epic", "My Epic", "beta", "Epic body.", nil)
				createStory(t, s, "s1", "Story 1", "my-epic", "planned", "high", "", nil, nil)
			},
			slug: "my-epic",
			opts: ExtractOptions{IncludeBody: true, Tree: false},
			check: func(t *testing.T, node *ExportNode) {
				if node.Type != NodeEpic {
					t.Errorf("Type = %q, want %q", node.Type, NodeEpic)
				}
				if node.Title != "My Epic" {
					t.Errorf("Title = %q, want %q", node.Title, "My Epic")
				}
				if len(node.Children) != 0 {
					t.Errorf("Children = %d, want 0 (tree=false)", len(node.Children))
				}
			},
		},
		{
			name: "epic with tree includes stories, docs, decisions",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "full-epic", "Full Epic", "beta", "Body.", []models.Phase{
					{Label: "P1", Stories: []string{"s1", "s2"}},
				})
				createStory(t, s, "s1", "Story 1", "full-epic", "planned", "high", "S1 body", nil, []string{"AC1"})
				createStory(t, s, "s2", "Story 2", "full-epic", "done", "medium", "", nil, nil)
				createDoc(t, s, "spec1", "Spec 1", "tech-spec", "full-epic", "Spec body.")
				createDecision(t, s, "dec1", "Decision 1", "## Context\n\nCtx\n\n## Decision\n\nDec", []string{"full-epic"})
			},
			slug: "full-epic",
			opts: ExtractOptions{IncludeBody: true, Tree: true},
			check: func(t *testing.T, node *ExportNode) {
				if len(node.Children) != 2 {
					t.Fatalf("Children = %d, want 2", len(node.Children))
				}
				if node.Children[0].Slug != "s1" {
					t.Errorf("Children[0].Slug = %q, want s1", node.Children[0].Slug)
				}
				if len(node.Docs) != 1 {
					t.Fatalf("Docs = %d, want 1", len(node.Docs))
				}
				if node.Docs[0].DocType != "tech-spec" {
					t.Errorf("Docs[0].DocType = %q, want tech-spec", node.Docs[0].DocType)
				}
				if len(node.Decisions) != 1 {
					t.Fatalf("Decisions = %d, want 1", len(node.Decisions))
				}
			},
		},
		{
			name: "epic tree with ExcludeStatuses filters done stories",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "filter-epic", "Filter", "beta", "", nil)
				createStory(t, s, "active", "Active", "filter-epic", "planned", "high", "", nil, nil)
				createStory(t, s, "done-s", "Done", "filter-epic", "done", "low", "", nil, nil)
			},
			slug: "filter-epic",
			opts: ExtractOptions{ExcludeStatuses: map[string]bool{"done": true}, IncludeBody: true, Tree: true},
			check: func(t *testing.T, node *ExportNode) {
				if len(node.Children) != 1 {
					t.Fatalf("Children = %d, want 1 (done filtered)", len(node.Children))
				}
				if node.Children[0].Slug != "active" {
					t.Errorf("Children[0].Slug = %q, want active", node.Children[0].Slug)
				}
			},
		},
		{
			name: "epic tree with IncludeBody=false omits body",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "nb-epic", "No Body", "beta", "Epic body.", nil)
				createStory(t, s, "nb-s", "NB Story", "nb-epic", "planned", "high", "Story body.", nil, nil)
			},
			slug: "nb-epic",
			opts: ExtractOptions{IncludeBody: false, Tree: true},
			check: func(t *testing.T, node *ExportNode) {
				if node.Body != "" {
					t.Errorf("Epic Body = %q, want empty", node.Body)
				}
				if len(node.Children) > 0 && node.Children[0].Body != "" {
					t.Errorf("Story Body = %q, want empty", node.Children[0].Body)
				}
			},
		},
		{
			name:    "nonexistent epic returns error",
			setup:   func(t *testing.T, s *store.Store) {},
			slug:    "nope",
			opts:    ExtractOptions{IncludeBody: true, Tree: true},
			wantErr: true,
		},
		{
			name: "epic tree with ExcludeStatuses filters cancelled stories",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "cancel-epic", "Cancel", "beta", "", nil)
				createStory(t, s, "active-s", "Active", "cancel-epic", "planned", "high", "", nil, nil)
				createStory(t, s, "cancel-s", "Cancelled", "cancel-epic", "cancelled", "low", "", nil, nil)
			},
			slug: "cancel-epic",
			opts: ExtractOptions{ExcludeStatuses: map[string]bool{"cancelled": true}, IncludeBody: true, Tree: true},
			check: func(t *testing.T, node *ExportNode) {
				if len(node.Children) != 1 {
					t.Fatalf("Children = %d, want 1 (cancelled filtered)", len(node.Children))
				}
				if node.Children[0].Slug != "active-s" {
					t.Errorf("Children[0].Slug = %q, want active-s", node.Children[0].Slug)
				}
			},
		},
		{
			name: "epic tree excludes both done and cancelled",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "multi-epic", "Multi", "beta", "", nil)
				createStory(t, s, "active-m", "Active", "multi-epic", "planned", "high", "", nil, nil)
				createStory(t, s, "done-m", "Done", "multi-epic", "done", "low", "", nil, nil)
				createStory(t, s, "cancel-m", "Cancelled", "multi-epic", "cancelled", "low", "", nil, nil)
			},
			slug: "multi-epic",
			opts: ExtractOptions{ExcludeStatuses: map[string]bool{"done": true, "cancelled": true}, IncludeBody: true, Tree: true},
			check: func(t *testing.T, node *ExportNode) {
				if len(node.Children) != 1 {
					t.Fatalf("Children = %d, want 1 (done+cancelled filtered)", len(node.Children))
				}
				if node.Children[0].Slug != "active-m" {
					t.Errorf("Children[0].Slug = %q, want active-m", node.Children[0].Slug)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			tt.setup(t, s)

			node, err := ExtractEpicNode(s, tt.slug, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ExtractEpicNode() error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, node)
			}
		})
	}
}

func TestExtractInitiative(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, s *store.Store)
		slug    string
		opts    ExtractOptions
		wantErr bool
		check   func(t *testing.T, node *ExportNode)
	}{
		{
			name: "initiative without tree",
			setup: func(t *testing.T, s *store.Store) {
				createInitiative(t, s, "init1", "Initiative 1", "Build things", nil)
			},
			slug: "init1",
			opts: ExtractOptions{IncludeBody: true, Tree: false},
			check: func(t *testing.T, node *ExportNode) {
				if node.Type != NodeInitiative {
					t.Errorf("Type = %q, want initiative", node.Type)
				}
				if node.Goal != "Build things" {
					t.Errorf("Goal = %q, want %q", node.Goal, "Build things")
				}
				if len(node.Children) != 0 {
					t.Errorf("Children = %d, want 0", len(node.Children))
				}
			},
		},
		{
			name: "initiative with tree loads epics",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "ep1", "Epic 1", "beta", "", nil)
				createEpic(t, s, "ep2", "Epic 2", "alpha", "", nil)
				createInitiative(t, s, "init2", "Initiative 2", "Goal", []string{"ep1", "ep2"})
			},
			slug: "init2",
			opts: ExtractOptions{IncludeBody: true, Tree: true},
			check: func(t *testing.T, node *ExportNode) {
				if len(node.Children) != 2 {
					t.Fatalf("Children = %d, want 2", len(node.Children))
				}
				if node.Children[0].Type != NodeEpic {
					t.Errorf("Children[0].Type = %q, want epic", node.Children[0].Type)
				}
			},
		},
		{
			name:    "nonexistent initiative",
			setup:   func(t *testing.T, s *store.Store) {},
			slug:    "nope",
			opts:    ExtractOptions{IncludeBody: true, Tree: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			tt.setup(t, s)

			node, err := ExtractInitiative(s, tt.slug, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ExtractInitiative() error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, node)
			}
		})
	}
}

func TestExtractStoryNode(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, s *store.Store)
		slug    string
		opts    ExtractOptions
		wantErr bool
		check   func(t *testing.T, node *ExportNode)
	}{
		{
			name: "standalone story",
			setup: func(t *testing.T, s *store.Store) {
				createStory(t, s, "solo", "Solo", "", "planned", "high", "Body.", []string{"be"}, []string{"AC1"})
			},
			slug: "solo",
			opts: ExtractOptions{IncludeBody: true},
			check: func(t *testing.T, node *ExportNode) {
				if node.Type != NodeStory {
					t.Errorf("Type = %q, want story", node.Type)
				}
				if node.Priority != "high" {
					t.Errorf("Priority = %q, want high", node.Priority)
				}
				if len(node.Acceptance) != 1 {
					t.Errorf("Acceptance len = %d, want 1", len(node.Acceptance))
				}
			},
		},
		{
			name: "done story with ExcludeStatuses excludes done",
			setup: func(t *testing.T, s *store.Store) {
				createStory(t, s, "done-s", "Done", "", "done", "low", "", nil, nil)
			},
			slug:    "done-s",
			opts:    ExtractOptions{ExcludeStatuses: map[string]bool{"done": true}, IncludeBody: true},
			wantErr: true,
		},
		{
			name:    "nonexistent story",
			setup:   func(t *testing.T, s *store.Store) {},
			slug:    "nope",
			opts:    ExtractOptions{IncludeBody: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			tt.setup(t, s)

			node, err := ExtractStoryNode(s, tt.slug, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ExtractStoryNode() error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, node)
			}
		})
	}
}

func TestExtractDoc(t *testing.T) {
	s := newTestStore(t)
	createDoc(t, s, "my-doc", "My Doc", "prd", "", "Doc body.")

	node, err := ExtractDoc(s, "my-doc", "", ExtractOptions{IncludeBody: true})
	if err != nil {
		t.Fatalf("ExtractDoc() error: %v", err)
	}
	if node.Type != NodeDoc {
		t.Errorf("Type = %q, want doc", node.Type)
	}
	if node.DocType != "prd" {
		t.Errorf("DocType = %q, want prd", node.DocType)
	}
	if node.Body != "Doc body." {
		t.Errorf("Body = %q, want %q", node.Body, "Doc body.")
	}

	// Verify IncludeBody=false omits body.
	node2, err := ExtractDoc(s, "my-doc", "", ExtractOptions{IncludeBody: false})
	if err != nil {
		t.Fatalf("ExtractDoc() error: %v", err)
	}
	if node2.Body != "" {
		t.Errorf("Body = %q, want empty", node2.Body)
	}
}

func TestExtractDecision(t *testing.T) {
	s := newTestStore(t)
	createDecision(t, s, "dec1", "Decision 1", "## Context\n\nSome context.", nil)

	node, err := ExtractDecision(s, "dec1", ExtractOptions{IncludeBody: true})
	if err != nil {
		t.Fatalf("ExtractDecision() error: %v", err)
	}
	if node.Type != NodeDecision {
		t.Errorf("Type = %q, want decision", node.Type)
	}
	if node.Title != "Decision 1" {
		t.Errorf("Title = %q, want %q", node.Title, "Decision 1")
	}
}

func TestExtractAll(t *testing.T) {
	s := newTestStore(t)

	// Create an initiative with an epic.
	createEpic(t, s, "ep1", "Epic 1", "beta", "", nil)
	createStory(t, s, "s1", "Story 1", "ep1", "planned", "high", "", nil, nil)
	createInitiative(t, s, "init1", "Init 1", "Goal", []string{"ep1"})

	// Create a standalone epic.
	createEpic(t, s, "ep2", "Epic 2", "alpha", "", nil)

	// Create a standalone story.
	createStory(t, s, "solo", "Solo Story", "", "planned", "medium", "", nil, nil)

	// Create a project-level doc.
	createDoc(t, s, "proj-doc", "Project Doc", "prd", "", "Doc body.")

	// Create an unscoped decision.
	createDecision(t, s, "dec1", "Decision 1", "Body.", nil)

	root, err := ExtractAll(s, ExtractOptions{IncludeBody: true})
	if err != nil {
		t.Fatalf("ExtractAll() error: %v", err)
	}

	// Should have: 1 initiative + 1 standalone epic + 1 standalone story = 3 children.
	if len(root.Children) != 3 {
		t.Errorf("Children = %d, want 3 (1 initiative + 1 standalone epic + 1 standalone story)", len(root.Children))
	}

	// Should have 1 project-level doc.
	if len(root.Docs) != 1 {
		t.Errorf("Docs = %d, want 1", len(root.Docs))
	}

	// Should have 1 unscoped decision.
	if len(root.Decisions) != 1 {
		t.Errorf("Decisions = %d, want 1", len(root.Decisions))
	}

	// Initiative child should have the epic under it.
	var initNode *ExportNode
	for _, c := range root.Children {
		if c.Type == NodeInitiative {
			initNode = c
			break
		}
	}
	if initNode == nil {
		t.Fatal("expected initiative node in children")
	}
	if len(initNode.Children) != 1 {
		t.Errorf("Initiative children = %d, want 1 (ep1)", len(initNode.Children))
	}
}
