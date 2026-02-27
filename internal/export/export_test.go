package export

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".specflow")
	s := store.New(root)
	if err := s.Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	return s
}

func createEpic(t *testing.T, s *store.Store, slug, title, fidelity, body string, phases []models.Phase) {
	t.Helper()
	e := &models.Epic{
		Slug:     slug,
		Title:    title,
		Status:   models.EpicStatusActive,
		Fidelity: fidelity,
		Phases:   phases,
		Body:     body,
	}
	if err := s.CreateEpic(e); err != nil {
		t.Fatalf("CreateEpic(%q) error: %v", slug, err)
	}
}

func createStory(t *testing.T, s *store.Store, slug, title, epic, status, priority, body string, labels, acceptance []string) {
	t.Helper()
	st := &models.Story{
		Slug:       slug,
		Title:      title,
		Epic:       epic,
		Status:     status,
		Priority:   priority,
		Labels:     labels,
		Acceptance: acceptance,
		Body:       body,
	}
	if err := s.CreateStory(st); err != nil {
		t.Fatalf("CreateStory(%q) error: %v", slug, err)
	}
}

func TestExportEpic(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, s *store.Store)
		epicSlug   string
		opts       ExportOptions
		wantErr    bool
		checkData  func(t *testing.T, data *ExportData)
	}{
		{
			name: "happy path with phased stories",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "my-epic", "My Epic", "beta", "Epic body here.", []models.Phase{
					{Label: "Phase 1", Stories: []string{"story-a", "story-b"}},
					{Label: "Phase 2", Stories: []string{"story-c"}},
				})
				createStory(t, s, "story-a", "Story A", "my-epic", "planned", "high", "Body A", []string{"backend"}, []string{"AC1"})
				createStory(t, s, "story-b", "Story B", "my-epic", "planned", "medium", "", nil, []string{"AC2", "AC3"})
				createStory(t, s, "story-c", "Story C", "my-epic", "in_progress", "low", "Body C", []string{"frontend"}, nil)
			},
			epicSlug: "my-epic",
			opts:     ExportOptions{IncludeDone: true, IncludeBody: true},
			checkData: func(t *testing.T, data *ExportData) {
				// Epic fields
				if data.Epic.Slug != "my-epic" {
					t.Errorf("Epic.Slug = %q, want %q", data.Epic.Slug, "my-epic")
				}
				if data.Epic.Title != "My Epic" {
					t.Errorf("Epic.Title = %q, want %q", data.Epic.Title, "My Epic")
				}
				if data.Epic.Status != "active" {
					t.Errorf("Epic.Status = %q, want %q", data.Epic.Status, "active")
				}
				if data.Epic.Fidelity != "beta" {
					t.Errorf("Epic.Fidelity = %q, want %q", data.Epic.Fidelity, "beta")
				}
				if data.Epic.Body != "Epic body here." {
					t.Errorf("Epic.Body = %q, want %q", data.Epic.Body, "Epic body here.")
				}
				if len(data.Epic.Phases) != 2 {
					t.Fatalf("Epic.Phases len = %d, want 2", len(data.Epic.Phases))
				}

				// Story count and order
				if len(data.Stories) != 3 {
					t.Fatalf("Stories len = %d, want 3", len(data.Stories))
				}
				if data.Stories[0].Slug != "story-a" {
					t.Errorf("Stories[0].Slug = %q, want %q", data.Stories[0].Slug, "story-a")
				}
				if data.Stories[1].Slug != "story-b" {
					t.Errorf("Stories[1].Slug = %q, want %q", data.Stories[1].Slug, "story-b")
				}
				if data.Stories[2].Slug != "story-c" {
					t.Errorf("Stories[2].Slug = %q, want %q", data.Stories[2].Slug, "story-c")
				}

				// Story fields
				if data.Stories[0].Priority != "high" {
					t.Errorf("Stories[0].Priority = %q, want %q", data.Stories[0].Priority, "high")
				}
				if len(data.Stories[0].Labels) != 1 || data.Stories[0].Labels[0] != "backend" {
					t.Errorf("Stories[0].Labels = %v, want [backend]", data.Stories[0].Labels)
				}

				// Description: body + acceptance
				if !strings.Contains(data.Stories[0].Description, "Body A") {
					t.Error("Stories[0].Description should contain body")
				}
				if !strings.Contains(data.Stories[0].Description, "- [ ] AC1") {
					t.Error("Stories[0].Description should contain acceptance criteria")
				}
			},
		},
		{
			name: "empty epic with no stories",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "empty-epic", "Empty", "alpha", "Some body.", nil)
			},
			epicSlug: "empty-epic",
			opts:     ExportOptions{IncludeDone: true, IncludeBody: true},
			checkData: func(t *testing.T, data *ExportData) {
				if data.Epic.Slug != "empty-epic" {
					t.Errorf("Epic.Slug = %q, want %q", data.Epic.Slug, "empty-epic")
				}
				if len(data.Stories) != 0 {
					t.Errorf("Stories len = %d, want 0", len(data.Stories))
				}
			},
		},
		{
			name: "IncludeDone=false filters done stories",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "filter-epic", "Filter", "beta", "", []models.Phase{
					{Label: "Phase 1", Stories: []string{"active-story", "done-story"}},
				})
				createStory(t, s, "active-story", "Active", "filter-epic", "planned", "high", "", nil, []string{"AC"})
				createStory(t, s, "done-story", "Done", "filter-epic", "done", "medium", "", nil, []string{"AC"})
			},
			epicSlug: "filter-epic",
			opts:     ExportOptions{IncludeDone: false, IncludeBody: true},
			checkData: func(t *testing.T, data *ExportData) {
				if len(data.Stories) != 1 {
					t.Fatalf("Stories len = %d, want 1", len(data.Stories))
				}
				if data.Stories[0].Slug != "active-story" {
					t.Errorf("Stories[0].Slug = %q, want %q", data.Stories[0].Slug, "active-story")
				}
			},
		},
		{
			name: "IncludeBody=false omits body",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "nobody-epic", "No Body", "beta", "Epic body.", []models.Phase{
					{Label: "P1", Stories: []string{"body-story"}},
				})
				createStory(t, s, "body-story", "Has Body", "nobody-epic", "planned", "high", "Story body.", nil, []string{"AC1"})
			},
			epicSlug: "nobody-epic",
			opts:     ExportOptions{IncludeDone: true, IncludeBody: false},
			checkData: func(t *testing.T, data *ExportData) {
				if data.Epic.Body != "" {
					t.Errorf("Epic.Body = %q, want empty", data.Epic.Body)
				}
				if strings.Contains(data.Stories[0].Description, "Story body") {
					t.Error("Story description should not contain body when IncludeBody=false")
				}
				// Should still have acceptance criteria
				if !strings.Contains(data.Stories[0].Description, "- [ ] AC1") {
					t.Error("Story description should still contain acceptance criteria")
				}
			},
		},
		{
			name: "unphased stories appear after phased",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "mixed-epic", "Mixed", "beta", "", []models.Phase{
					{Label: "P1", Stories: []string{"phased"}},
				})
				createStory(t, s, "unphased", "Unphased", "mixed-epic", "planned", "low", "", nil, []string{"AC"})
				createStory(t, s, "phased", "Phased", "mixed-epic", "planned", "high", "", nil, []string{"AC"})
			},
			epicSlug: "mixed-epic",
			opts:     ExportOptions{IncludeDone: true, IncludeBody: true},
			checkData: func(t *testing.T, data *ExportData) {
				if len(data.Stories) != 2 {
					t.Fatalf("Stories len = %d, want 2", len(data.Stories))
				}
				if data.Stories[0].Slug != "phased" {
					t.Errorf("Stories[0].Slug = %q, want %q (phased first)", data.Stories[0].Slug, "phased")
				}
				if data.Stories[1].Slug != "unphased" {
					t.Errorf("Stories[1].Slug = %q, want %q (unphased last)", data.Stories[1].Slug, "unphased")
				}
			},
		},
		{
			name: "story with no body uses acceptance as description",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "ac-epic", "AC Only", "beta", "", nil)
				createStory(t, s, "ac-story", "AC Story", "ac-epic", "planned", "medium", "", nil, []string{"Must do X", "Must do Y"})
			},
			epicSlug: "ac-epic",
			opts:     ExportOptions{IncludeDone: true, IncludeBody: true},
			checkData: func(t *testing.T, data *ExportData) {
				desc := data.Stories[0].Description
				if !strings.HasPrefix(desc, "## Acceptance Criteria") {
					t.Errorf("Description should start with acceptance header, got: %q", desc)
				}
				if !strings.Contains(desc, "- [ ] Must do X") {
					t.Error("Description should contain first AC")
				}
				if !strings.Contains(desc, "- [ ] Must do Y") {
					t.Error("Description should contain second AC")
				}
			},
		},
		{
			name: "story with no body and no acceptance has empty description",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "bare-epic", "Bare", "beta", "", nil)
				createStory(t, s, "bare-story", "Bare Story", "bare-epic", "planned", "medium", "", nil, nil)
			},
			epicSlug: "bare-epic",
			opts:     ExportOptions{IncludeDone: true, IncludeBody: true},
			checkData: func(t *testing.T, data *ExportData) {
				if data.Stories[0].Description != "" {
					t.Errorf("Description = %q, want empty", data.Stories[0].Description)
				}
			},
		},
		{
			name: "mixed phased unphased and done stories",
			setup: func(t *testing.T, s *store.Store) {
				createEpic(t, s, "full-epic", "Full", "production", "Full epic body.", []models.Phase{
					{Label: "Phase 1", Stories: []string{"p1-story"}},
					{Label: "Phase 2", Stories: []string{"p2-story"}},
				})
				createStory(t, s, "p1-story", "Phase 1 Story", "full-epic", "planned", "high", "P1 body", nil, []string{"AC1"})
				createStory(t, s, "p2-story", "Phase 2 Story", "full-epic", "done", "medium", "", nil, []string{"AC2"})
				createStory(t, s, "orphan", "Orphan Story", "full-epic", "planned", "low", "", nil, nil)
			},
			epicSlug: "full-epic",
			opts:     ExportOptions{IncludeDone: false, IncludeBody: true},
			checkData: func(t *testing.T, data *ExportData) {
				// Done story should be filtered out
				if len(data.Stories) != 2 {
					t.Fatalf("Stories len = %d, want 2 (done filtered)", len(data.Stories))
				}
				// Phase 1 story first, orphan last
				if data.Stories[0].Slug != "p1-story" {
					t.Errorf("Stories[0].Slug = %q, want %q", data.Stories[0].Slug, "p1-story")
				}
				if data.Stories[1].Slug != "orphan" {
					t.Errorf("Stories[1].Slug = %q, want %q", data.Stories[1].Slug, "orphan")
				}
			},
		},
		{
			name:     "nonexistent epic returns error",
			setup:    func(t *testing.T, s *store.Store) {},
			epicSlug: "does-not-exist",
			opts:     ExportOptions{IncludeDone: true, IncludeBody: true},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			tt.setup(t, s)

			data, err := ExportEpic(s, tt.epicSlug, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ExportEpic() error: %v", err)
			}
			if tt.checkData != nil {
				tt.checkData(t, data)
			}
		})
	}
}

func TestExportStory(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, s *store.Store)
		slug      string
		opts      ExportOptions
		wantErr   bool
		checkData func(t *testing.T, st *StoryExport)
	}{
		{
			name: "standalone story with body and acceptance",
			setup: func(t *testing.T, s *store.Store) {
				createStory(t, s, "solo-story", "Solo Story", "", "planned", "high", "Story body.", []string{"backend"}, []string{"AC1", "AC2"})
			},
			slug: "solo-story",
			opts: ExportOptions{IncludeDone: true, IncludeBody: true},
			checkData: func(t *testing.T, st *StoryExport) {
				if st.Slug != "solo-story" {
					t.Errorf("Slug = %q, want %q", st.Slug, "solo-story")
				}
				if st.Title != "Solo Story" {
					t.Errorf("Title = %q, want %q", st.Title, "Solo Story")
				}
				if st.Priority != "high" {
					t.Errorf("Priority = %q, want %q", st.Priority, "high")
				}
				if st.Description == "" {
					t.Error("Description should not be empty")
				}
			},
		},
		{
			name: "done story with IncludeDone=false returns error",
			setup: func(t *testing.T, s *store.Store) {
				createStory(t, s, "done-solo", "Done Solo", "", "done", "low", "", nil, nil)
			},
			slug:    "done-solo",
			opts:    ExportOptions{IncludeDone: false, IncludeBody: true},
			wantErr: true,
		},
		{
			name:    "nonexistent story returns error",
			setup:   func(t *testing.T, s *store.Store) {},
			slug:    "nope",
			opts:    ExportOptions{IncludeDone: true, IncludeBody: true},
			wantErr: true,
		},
		{
			name: "IncludeBody=false omits body from description",
			setup: func(t *testing.T, s *store.Store) {
				createStory(t, s, "nb-story", "No Body Export", "", "planned", "medium", "Secret body.", nil, []string{"AC1"})
			},
			slug: "nb-story",
			opts: ExportOptions{IncludeDone: true, IncludeBody: false},
			checkData: func(t *testing.T, st *StoryExport) {
				if strings.Contains(st.Description, "Secret body") {
					t.Error("Description should not contain body when IncludeBody=false")
				}
				if !strings.Contains(st.Description, "- [ ] AC1") {
					t.Error("Description should contain acceptance criteria")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			tt.setup(t, s)

			st, err := ExportStory(s, tt.slug, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ExportStory() error: %v", err)
			}
			if tt.checkData != nil {
				tt.checkData(t, st)
			}
		})
	}
}

func TestAssembleDescription(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		acceptance  []string
		includeBody bool
		want        string
	}{
		{
			name:        "body and acceptance",
			body:        "Some body text.",
			acceptance:  []string{"AC1", "AC2"},
			includeBody: true,
			want:        "Some body text.\n\n## Acceptance Criteria\n- [ ] AC1\n- [ ] AC2",
		},
		{
			name:        "body only",
			body:        "Just body.",
			acceptance:  nil,
			includeBody: true,
			want:        "Just body.",
		},
		{
			name:        "acceptance only",
			body:        "",
			acceptance:  []string{"AC1"},
			includeBody: true,
			want:        "## Acceptance Criteria\n- [ ] AC1",
		},
		{
			name:        "body excluded",
			body:        "Hidden body.",
			acceptance:  []string{"AC1"},
			includeBody: false,
			want:        "## Acceptance Criteria\n- [ ] AC1",
		},
		{
			name:        "nothing",
			body:        "",
			acceptance:  nil,
			includeBody: true,
			want:        "",
		},
		{
			name:        "body with whitespace trimmed",
			body:        "  Body text.  \n\n",
			acceptance:  nil,
			includeBody: true,
			want:        "Body text.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assembleDescription(tt.body, tt.acceptance, tt.includeBody)
			if got != tt.want {
				t.Errorf("assembleDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}
