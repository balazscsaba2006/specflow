package store

import (
	"os"
	"testing"

	"github.com/balazscsaba2006/specflow/internal/models"
)

func makeEpic(slug, title string) *models.Epic {
	return &models.Epic{
		Slug:       slug,
		Title:      title,
		Status:     models.EpicStatusDraft,
		Initiative: "i_parent",
		Phases: []models.Phase{
			{Label: "Phase 1", Stories: []string{"s_aaa", "s_bbb"}},
			{Label: "Phase 2", Stories: []string{"s_ccc"}},
		},
		OpenQuestions: []string{"How does X work?"},
		Decisions:     []string{"dec_001"},
		Body:          "# Epic Overview\n\nDetailed description here.",
	}
}

func TestEpicCreateAndLoad(t *testing.T) {
	s := newTestStore(t)
	e := makeEpic("my-epic", "My Epic")

	if err := s.CreateEpic(e); err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	// ID should be set with correct prefix
	if e.ID == "" {
		t.Fatal("expected ID to be set")
	}
	if e.ID[:2] != models.PrefixEpic {
		t.Errorf("ID prefix = %q, want %q", e.ID[:2], models.PrefixEpic)
	}

	// Timestamps should be set
	if e.Created.IsZero() {
		t.Error("expected Created to be set")
	}
	if e.Updated.IsZero() {
		t.Error("expected Updated to be set")
	}

	// File should exist
	if _, err := os.Stat(s.EpicFile(e.Slug)); err != nil {
		t.Fatalf("epic file does not exist: %v", err)
	}

	// Load and verify roundtrip
	loaded, err := s.LoadEpic("my-epic")
	if err != nil {
		t.Fatalf("LoadEpic() error = %v", err)
	}

	if loaded.ID != e.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, e.ID)
	}
	if loaded.Slug != e.Slug {
		t.Errorf("Slug = %q, want %q", loaded.Slug, e.Slug)
	}
	if loaded.Title != e.Title {
		t.Errorf("Title = %q, want %q", loaded.Title, e.Title)
	}
	if loaded.Status != e.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, e.Status)
	}
	if loaded.Initiative != e.Initiative {
		t.Errorf("Initiative = %q, want %q", loaded.Initiative, e.Initiative)
	}
	if len(loaded.Phases) != len(e.Phases) {
		t.Fatalf("Phases len = %d, want %d", len(loaded.Phases), len(e.Phases))
	}
	for i, p := range loaded.Phases {
		if p.Label != e.Phases[i].Label {
			t.Errorf("Phases[%d].Label = %q, want %q", i, p.Label, e.Phases[i].Label)
		}
		if len(p.Stories) != len(e.Phases[i].Stories) {
			t.Errorf("Phases[%d].Stories len = %d, want %d", i, len(p.Stories), len(e.Phases[i].Stories))
		}
	}
	if len(loaded.OpenQuestions) != len(e.OpenQuestions) {
		t.Errorf("OpenQuestions len = %d, want %d", len(loaded.OpenQuestions), len(e.OpenQuestions))
	}
	if len(loaded.Decisions) != len(e.Decisions) {
		t.Errorf("Decisions len = %d, want %d", len(loaded.Decisions), len(e.Decisions))
	}
	if loaded.Body != e.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, e.Body)
	}
	if !loaded.Created.Equal(e.Created) {
		t.Errorf("Created = %v, want %v", loaded.Created, e.Created)
	}
	if !loaded.Updated.Equal(e.Updated) {
		t.Errorf("Updated = %v, want %v", loaded.Updated, e.Updated)
	}
}

func TestEpicCreateSubdirectories(t *testing.T) {
	s := newTestStore(t)
	e := makeEpic("subdirs-epic", "Subdirs Epic")

	if err := s.CreateEpic(e); err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	dirs := []struct {
		name string
		path string
	}{
		{"docs", s.EpicDocsDir(e.Slug)},
		{"stories", s.EpicStoriesDir(e.Slug)},
	}

	for _, d := range dirs {
		info, err := os.Stat(d.path)
		if err != nil {
			t.Errorf("%s directory does not exist: %v", d.name, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s path is not a directory", d.name)
		}
	}
}

func TestEpicCreateInvalidSlug(t *testing.T) {
	s := newTestStore(t)

	tests := []struct {
		name string
		slug string
	}{
		{"empty", ""},
		{"uppercase", "My-Epic"},
		{"spaces", "my epic"},
		{"special chars", "my_epic!"},
		{"leading hyphen", "-my-epic"},
		{"trailing hyphen", "my-epic-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := makeEpic(tt.slug, "Title")
			if err := s.CreateEpic(e); err == nil {
				t.Errorf("CreateEpic(%q) expected error, got nil", tt.slug)
			}
		})
	}
}

func TestEpicCreateDuplicateSlug(t *testing.T) {
	s := newTestStore(t)

	e := makeEpic("dup-epic", "First")
	if err := s.CreateEpic(e); err != nil {
		t.Fatalf("first CreateEpic() error = %v", err)
	}

	dup := makeEpic("dup-epic", "Second")
	if err := s.CreateEpic(dup); err == nil {
		t.Error("expected error for duplicate slug, got nil")
	}
}

func TestEpicList(t *testing.T) {
	s := newTestStore(t)

	slugs := []string{"alpha", "beta", "gamma"}
	for _, slug := range slugs {
		e := makeEpic(slug, "Title "+slug)
		if err := s.CreateEpic(e); err != nil {
			t.Fatalf("CreateEpic(%q) error = %v", slug, err)
		}
	}

	list, err := s.ListEpics()
	if err != nil {
		t.Fatalf("ListEpics() error = %v", err)
	}
	if len(list) != len(slugs) {
		t.Fatalf("ListEpics() returned %d, want %d", len(list), len(slugs))
	}

	found := make(map[string]bool)
	for _, e := range list {
		found[e.Slug] = true
	}
	for _, slug := range slugs {
		if !found[slug] {
			t.Errorf("slug %q not found in list", slug)
		}
	}
}

func TestEpicListEmpty(t *testing.T) {
	s := newTestStore(t)

	list, err := s.ListEpics()
	if err != nil {
		t.Fatalf("ListEpics() error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("ListEpics() returned %d, want 0", len(list))
	}
}

func TestEpicSave(t *testing.T) {
	s := newTestStore(t)

	e := makeEpic("save-test", "Original Title")
	if err := s.CreateEpic(e); err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	originalCreated := e.Created
	originalUpdated := e.Updated

	// Mutate fields
	e.Title = "Updated Title"
	e.Status = models.EpicStatusActive
	e.Body = "# Updated\n\nNew epic content."

	if err := s.SaveEpic(e); err != nil {
		t.Fatalf("SaveEpic() error = %v", err)
	}

	// Updated timestamp should have changed (or at least not be before original)
	if e.Updated.Before(originalUpdated) {
		t.Errorf("Updated went backwards: %v < %v", e.Updated, originalUpdated)
	}

	// Reload and verify
	loaded, err := s.LoadEpic("save-test")
	if err != nil {
		t.Fatalf("LoadEpic() error = %v", err)
	}

	if loaded.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Updated Title")
	}
	if loaded.Status != models.EpicStatusActive {
		t.Errorf("Status = %q, want %q", loaded.Status, models.EpicStatusActive)
	}
	if loaded.Body != "# Updated\n\nNew epic content." {
		t.Errorf("Body = %q, want %q", loaded.Body, "# Updated\n\nNew epic content.")
	}
	if !loaded.Created.Equal(originalCreated) {
		t.Errorf("Created changed: got %v, want %v", loaded.Created, originalCreated)
	}
}

func TestEpicDelete(t *testing.T) {
	s := newTestStore(t)

	e := makeEpic("to-delete", "Delete Me")
	if err := s.CreateEpic(e); err != nil {
		t.Fatalf("CreateEpic() error = %v", err)
	}

	// Verify it exists
	if _, err := s.LoadEpic("to-delete"); err != nil {
		t.Fatalf("epic should exist before delete: %v", err)
	}

	if err := s.DeleteEpic("to-delete"); err != nil {
		t.Fatalf("DeleteEpic() error = %v", err)
	}

	// Directory should be gone (including docs/ and stories/ subdirs)
	if _, err := os.Stat(s.EpicDir("to-delete")); !os.IsNotExist(err) {
		t.Error("expected epic directory to be removed")
	}

	// Load should fail
	if _, err := s.LoadEpic("to-delete"); err == nil {
		t.Error("expected error loading deleted epic, got nil")
	}
}

func TestEpicLoadNonExistent(t *testing.T) {
	s := newTestStore(t)

	_, err := s.LoadEpic("does-not-exist")
	if err == nil {
		t.Error("expected error loading non-existent epic, got nil")
	}
}

func TestEpicDeleteNonExistent(t *testing.T) {
	s := newTestStore(t)

	err := s.DeleteEpic("ghost")
	if err == nil {
		t.Error("expected error deleting non-existent epic, got nil")
	}
}

func TestEpicSaveNonExistent(t *testing.T) {
	s := newTestStore(t)

	e := makeEpic("no-such-epic", "Nope")
	e.ID = "e_fake"
	err := s.SaveEpic(e)
	if err == nil {
		t.Error("expected error saving non-existent epic, got nil")
	}
}
