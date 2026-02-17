package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/balazscsaba2006/specflow/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".specflow")
	s := New(root)
	if err := s.Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	return s
}

func makeInitiative(slug, title string) *models.Initiative {
	return &models.Initiative{
		Slug:            slug,
		Title:           title,
		Status:          models.InitiativeStatusActive,
		Goal:            "Ship it",
		SuccessCriteria: []string{"works", "tests pass"},
		Body:            "# Overview\n\nSome description here.",
	}
}

func TestInitiativeCreateAndLoad(t *testing.T) {
	s := newTestStore(t)
	i := makeInitiative("my-project", "My Project")

	if err := s.CreateInitiative(i); err != nil {
		t.Fatalf("CreateInitiative() error = %v", err)
	}

	// ID should be set with correct prefix
	if i.ID == "" {
		t.Fatal("expected ID to be set")
	}
	if i.ID[:2] != models.PrefixInitiative {
		t.Errorf("ID prefix = %q, want %q", i.ID[:2], models.PrefixInitiative)
	}

	// Timestamps should be set
	if i.Created.IsZero() {
		t.Error("expected Created to be set")
	}
	if i.Updated.IsZero() {
		t.Error("expected Updated to be set")
	}

	// Directory and file should exist
	if _, err := os.Stat(s.InitiativeFile(i.Slug)); err != nil {
		t.Fatalf("initiative file does not exist: %v", err)
	}

	// Load and verify roundtrip
	loaded, err := s.LoadInitiative("my-project")
	if err != nil {
		t.Fatalf("LoadInitiative() error = %v", err)
	}

	if loaded.ID != i.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, i.ID)
	}
	if loaded.Slug != i.Slug {
		t.Errorf("Slug = %q, want %q", loaded.Slug, i.Slug)
	}
	if loaded.Title != i.Title {
		t.Errorf("Title = %q, want %q", loaded.Title, i.Title)
	}
	if loaded.Status != i.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, i.Status)
	}
	if loaded.Goal != i.Goal {
		t.Errorf("Goal = %q, want %q", loaded.Goal, i.Goal)
	}
	if len(loaded.SuccessCriteria) != len(i.SuccessCriteria) {
		t.Errorf("SuccessCriteria len = %d, want %d", len(loaded.SuccessCriteria), len(i.SuccessCriteria))
	}
	if loaded.Body != i.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, i.Body)
	}
	if !loaded.Created.Equal(i.Created) {
		t.Errorf("Created = %v, want %v", loaded.Created, i.Created)
	}
	if !loaded.Updated.Equal(i.Updated) {
		t.Errorf("Updated = %v, want %v", loaded.Updated, i.Updated)
	}
}

func TestInitiativeCreateInvalidSlug(t *testing.T) {
	s := newTestStore(t)

	tests := []struct {
		name string
		slug string
	}{
		{"empty", ""},
		{"uppercase", "My-Project"},
		{"spaces", "my project"},
		{"special chars", "my_project!"},
		{"leading hyphen", "-my-project"},
		{"trailing hyphen", "my-project-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := makeInitiative(tt.slug, "Title")
			if err := s.CreateInitiative(i); err == nil {
				t.Errorf("CreateInitiative(%q) expected error, got nil", tt.slug)
			}
		})
	}
}

func TestInitiativeCreateDuplicateSlug(t *testing.T) {
	s := newTestStore(t)

	i := makeInitiative("dup-project", "First")
	if err := s.CreateInitiative(i); err != nil {
		t.Fatalf("first CreateInitiative() error = %v", err)
	}

	dup := makeInitiative("dup-project", "Second")
	if err := s.CreateInitiative(dup); err == nil {
		t.Error("expected error for duplicate slug, got nil")
	}
}

func TestInitiativeList(t *testing.T) {
	s := newTestStore(t)

	slugs := []string{"alpha", "beta", "gamma"}
	for _, slug := range slugs {
		i := makeInitiative(slug, "Title "+slug)
		if err := s.CreateInitiative(i); err != nil {
			t.Fatalf("CreateInitiative(%q) error = %v", slug, err)
		}
	}

	list, err := s.ListInitiatives()
	if err != nil {
		t.Fatalf("ListInitiatives() error = %v", err)
	}
	if len(list) != len(slugs) {
		t.Fatalf("ListInitiatives() returned %d, want %d", len(list), len(slugs))
	}

	found := make(map[string]bool)
	for _, i := range list {
		found[i.Slug] = true
	}
	for _, slug := range slugs {
		if !found[slug] {
			t.Errorf("slug %q not found in list", slug)
		}
	}
}

func TestInitiativeListEmpty(t *testing.T) {
	s := newTestStore(t)

	list, err := s.ListInitiatives()
	if err != nil {
		t.Fatalf("ListInitiatives() error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("ListInitiatives() returned %d, want 0", len(list))
	}
}

func TestInitiativeSave(t *testing.T) {
	s := newTestStore(t)

	i := makeInitiative("save-test", "Original Title")
	if err := s.CreateInitiative(i); err != nil {
		t.Fatalf("CreateInitiative() error = %v", err)
	}

	originalCreated := i.Created
	originalUpdated := i.Updated

	// Mutate fields
	i.Title = "Updated Title"
	i.Status = models.InitiativeStatusCompleted
	i.Body = "# Updated\n\nNew content."

	if err := s.SaveInitiative(i); err != nil {
		t.Fatalf("SaveInitiative() error = %v", err)
	}

	// Updated timestamp should have changed (or at least not be before original)
	if i.Updated.Before(originalUpdated) {
		t.Errorf("Updated went backwards: %v < %v", i.Updated, originalUpdated)
	}

	// Reload and verify
	loaded, err := s.LoadInitiative("save-test")
	if err != nil {
		t.Fatalf("LoadInitiative() error = %v", err)
	}

	if loaded.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Updated Title")
	}
	if loaded.Status != models.InitiativeStatusCompleted {
		t.Errorf("Status = %q, want %q", loaded.Status, models.InitiativeStatusCompleted)
	}
	if loaded.Body != "# Updated\n\nNew content." {
		t.Errorf("Body = %q, want %q", loaded.Body, "# Updated\n\nNew content.")
	}
	if !loaded.Created.Equal(originalCreated) {
		t.Errorf("Created changed: got %v, want %v", loaded.Created, originalCreated)
	}
}

func TestInitiativeDelete(t *testing.T) {
	s := newTestStore(t)

	i := makeInitiative("to-delete", "Delete Me")
	if err := s.CreateInitiative(i); err != nil {
		t.Fatalf("CreateInitiative() error = %v", err)
	}

	// Verify it exists
	if _, err := s.LoadInitiative("to-delete"); err != nil {
		t.Fatalf("initiative should exist before delete: %v", err)
	}

	if err := s.DeleteInitiative("to-delete"); err != nil {
		t.Fatalf("DeleteInitiative() error = %v", err)
	}

	// Directory should be gone
	if _, err := os.Stat(s.InitiativeDir("to-delete")); !os.IsNotExist(err) {
		t.Error("expected initiative directory to be removed")
	}

	// Load should fail
	if _, err := s.LoadInitiative("to-delete"); err == nil {
		t.Error("expected error loading deleted initiative, got nil")
	}
}

func TestInitiativeLoadNonExistent(t *testing.T) {
	s := newTestStore(t)

	_, err := s.LoadInitiative("does-not-exist")
	if err == nil {
		t.Error("expected error loading non-existent initiative, got nil")
	}
}

func TestInitiativeDeleteNonExistent(t *testing.T) {
	s := newTestStore(t)

	err := s.DeleteInitiative("ghost")
	if err == nil {
		t.Error("expected error deleting non-existent initiative, got nil")
	}
}

func TestInitiativeSaveNonExistent(t *testing.T) {
	s := newTestStore(t)

	i := makeInitiative("no-such-thing", "Nope")
	i.ID = "i_fake"
	err := s.SaveInitiative(i)
	if err == nil {
		t.Error("expected error saving non-existent initiative, got nil")
	}
}
