package store

import (
	"os"
	"testing"

	"github.com/balazscsaba2006/specflow/internal/models"
)

func makeDoc(slug, title string) *models.Document {
	return &models.Document{
		Slug:          slug,
		Title:         title,
		Type:          models.DocTypeTechSpec,
		OpenQuestions: []string{"How to handle auth?"},
		Body:          "# Tech Spec\n\nDetailed spec content here.",
	}
}

func makeEpicScopedDoc(slug, title, epicSlug string) *models.Document {
	d := makeDoc(slug, title)
	d.Epic = epicSlug
	return d
}

// createEpicDocsDir sets up the epic docs directory so epic-scoped doc tests work.
func createEpicDocsDir(t *testing.T, s *Store, epicSlug string) {
	t.Helper()
	if err := s.EnsureDir(s.EpicDocsDir(epicSlug)); err != nil {
		t.Fatalf("creating epic docs dir: %v", err)
	}
}

func TestDocCreateAndLoad(t *testing.T) {
	s := newTestStore(t)
	d := makeDoc("api-design", "API Design")

	if err := s.CreateDoc(d); err != nil {
		t.Fatalf("CreateDoc() error = %v", err)
	}

	// ID should be set with correct prefix
	if d.ID == "" {
		t.Fatal("expected ID to be set")
	}
	if d.ID[:2] != models.PrefixDoc {
		t.Errorf("ID prefix = %q, want %q", d.ID[:2], models.PrefixDoc)
	}

	// Timestamps should be set
	if d.Created.IsZero() {
		t.Error("expected Created to be set")
	}
	if d.Updated.IsZero() {
		t.Error("expected Updated to be set")
	}

	// File should exist
	if _, err := os.Stat(s.DocFile(d.Slug, "")); err != nil {
		t.Fatalf("doc file does not exist: %v", err)
	}

	// Load and verify roundtrip
	loaded, err := s.LoadDoc("api-design", "")
	if err != nil {
		t.Fatalf("LoadDoc() error = %v", err)
	}

	if loaded.ID != d.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, d.ID)
	}
	if loaded.Slug != d.Slug {
		t.Errorf("Slug = %q, want %q", loaded.Slug, d.Slug)
	}
	if loaded.Title != d.Title {
		t.Errorf("Title = %q, want %q", loaded.Title, d.Title)
	}
	if loaded.Type != d.Type {
		t.Errorf("Type = %q, want %q", loaded.Type, d.Type)
	}
	if loaded.Status != d.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, d.Status)
	}
	if len(loaded.OpenQuestions) != len(d.OpenQuestions) {
		t.Errorf("OpenQuestions len = %d, want %d", len(loaded.OpenQuestions), len(d.OpenQuestions))
	}
	if loaded.Body != d.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, d.Body)
	}
	if !loaded.Created.Equal(d.Created) {
		t.Errorf("Created = %v, want %v", loaded.Created, d.Created)
	}
	if !loaded.Updated.Equal(d.Updated) {
		t.Errorf("Updated = %v, want %v", loaded.Updated, d.Updated)
	}
}

func TestDocCreateEpicScopedAndLoad(t *testing.T) {
	s := newTestStore(t)
	epicSlug := "auth-epic"
	createEpicDocsDir(t, s, epicSlug)

	d := makeEpicScopedDoc("auth-spec", "Auth Spec", epicSlug)

	if err := s.CreateDoc(d); err != nil {
		t.Fatalf("CreateDoc() error = %v", err)
	}

	// File should be under epic docs dir
	if _, err := os.Stat(s.DocFile(d.Slug, epicSlug)); err != nil {
		t.Fatalf("epic-scoped doc file does not exist: %v", err)
	}

	// Load and verify roundtrip
	loaded, err := s.LoadDoc("auth-spec", epicSlug)
	if err != nil {
		t.Fatalf("LoadDoc() error = %v", err)
	}

	if loaded.ID != d.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, d.ID)
	}
	if loaded.Slug != d.Slug {
		t.Errorf("Slug = %q, want %q", loaded.Slug, d.Slug)
	}
	if loaded.Title != d.Title {
		t.Errorf("Title = %q, want %q", loaded.Title, d.Title)
	}
	if loaded.Epic != epicSlug {
		t.Errorf("Epic = %q, want %q", loaded.Epic, epicSlug)
	}
	if loaded.Body != d.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, d.Body)
	}
}

func TestDocDefaultStatus(t *testing.T) {
	s := newTestStore(t)
	d := makeDoc("defaults-test", "Defaults Test")

	// No status set
	if d.Status != "" {
		t.Fatal("precondition failed: status should be empty before create")
	}

	if err := s.CreateDoc(d); err != nil {
		t.Fatalf("CreateDoc() error = %v", err)
	}

	if d.Status != models.DocStatusDraft {
		t.Errorf("default Status = %q, want %q", d.Status, models.DocStatusDraft)
	}

	// Verify on reload
	loaded, err := s.LoadDoc("defaults-test", "")
	if err != nil {
		t.Fatalf("LoadDoc() error = %v", err)
	}
	if loaded.Status != models.DocStatusDraft {
		t.Errorf("loaded Status = %q, want %q", loaded.Status, models.DocStatusDraft)
	}
}

func TestDocListProjectLevel(t *testing.T) {
	s := newTestStore(t)

	slugs := []string{"alpha-doc", "beta-doc", "gamma-doc"}
	for _, slug := range slugs {
		d := makeDoc(slug, "Title "+slug)
		if err := s.CreateDoc(d); err != nil {
			t.Fatalf("CreateDoc(%q) error = %v", slug, err)
		}
	}

	list, err := s.ListDocs("")
	if err != nil {
		t.Fatalf("ListDocs() error = %v", err)
	}
	if len(list) != len(slugs) {
		t.Fatalf("ListDocs() returned %d, want %d", len(list), len(slugs))
	}

	found := make(map[string]bool)
	for _, d := range list {
		found[d.Slug] = true
	}
	for _, slug := range slugs {
		if !found[slug] {
			t.Errorf("slug %q not found in list", slug)
		}
	}
}

func TestDocListEpicScoped(t *testing.T) {
	s := newTestStore(t)
	epicSlug := "my-epic"
	createEpicDocsDir(t, s, epicSlug)

	slugs := []string{"spec-one", "spec-two", "spec-three"}
	for _, slug := range slugs {
		d := makeEpicScopedDoc(slug, "Title "+slug, epicSlug)
		if err := s.CreateDoc(d); err != nil {
			t.Fatalf("CreateDoc(%q) error = %v", slug, err)
		}
	}

	list, err := s.ListDocs(epicSlug)
	if err != nil {
		t.Fatalf("ListDocs(%q) error = %v", epicSlug, err)
	}
	if len(list) != len(slugs) {
		t.Fatalf("ListDocs(%q) returned %d, want %d", epicSlug, len(list), len(slugs))
	}

	found := make(map[string]bool)
	for _, d := range list {
		found[d.Slug] = true
	}
	for _, slug := range slugs {
		if !found[slug] {
			t.Errorf("slug %q not found in epic-scoped list", slug)
		}
	}
}

func TestDocSavePreservesBody(t *testing.T) {
	s := newTestStore(t)
	d := makeDoc("body-test", "Body Test")
	d.Body = "# Original Body\n\nWith some content."
	if err := s.CreateDoc(d); err != nil {
		t.Fatalf("CreateDoc() error = %v", err)
	}

	originalCreated := d.Created
	originalUpdated := d.Updated

	// Update title, keep body
	d.Title = "Updated Title"
	if err := s.SaveDoc(d); err != nil {
		t.Fatalf("SaveDoc() error = %v", err)
	}

	// Updated timestamp should have changed (or at least not be before original)
	if d.Updated.Before(originalUpdated) {
		t.Errorf("Updated went backwards: %v < %v", d.Updated, originalUpdated)
	}

	loaded, err := s.LoadDoc("body-test", "")
	if err != nil {
		t.Fatalf("LoadDoc() error = %v", err)
	}
	if loaded.Body != "# Original Body\n\nWith some content." {
		t.Errorf("Body = %q, want %q", loaded.Body, "# Original Body\n\nWith some content.")
	}
	if loaded.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Updated Title")
	}
	if !loaded.Created.Equal(originalCreated) {
		t.Errorf("Created changed: got %v, want %v", loaded.Created, originalCreated)
	}
}

func TestDocDelete(t *testing.T) {
	s := newTestStore(t)
	d := makeDoc("to-delete", "Delete Me")
	if err := s.CreateDoc(d); err != nil {
		t.Fatalf("CreateDoc() error = %v", err)
	}

	// Verify it exists
	if _, err := s.LoadDoc("to-delete", ""); err != nil {
		t.Fatalf("doc should exist before delete: %v", err)
	}

	if err := s.DeleteDoc("to-delete", ""); err != nil {
		t.Fatalf("DeleteDoc() error = %v", err)
	}

	// File should be gone
	if _, err := os.Stat(s.DocFile("to-delete", "")); !os.IsNotExist(err) {
		t.Error("expected doc file to be removed")
	}

	// Load should fail
	if _, err := s.LoadDoc("to-delete", ""); err == nil {
		t.Error("expected error loading deleted doc, got nil")
	}
}

func TestDocLoadNonExistent(t *testing.T) {
	s := newTestStore(t)

	_, err := s.LoadDoc("does-not-exist", "")
	if err == nil {
		t.Error("expected error loading non-existent doc, got nil")
	}
}

func TestDocCreateDuplicateSlug(t *testing.T) {
	s := newTestStore(t)

	d := makeDoc("dup-doc", "First")
	if err := s.CreateDoc(d); err != nil {
		t.Fatalf("first CreateDoc() error = %v", err)
	}

	dup := makeDoc("dup-doc", "Second")
	if err := s.CreateDoc(dup); err == nil {
		t.Error("expected error for duplicate slug, got nil")
	}
}

func TestDocSaveNonExistent(t *testing.T) {
	s := newTestStore(t)

	d := makeDoc("no-such-doc", "Nope")
	d.ID = "d_fake"
	d.Status = models.DocStatusDraft
	err := s.SaveDoc(d)
	if err == nil {
		t.Error("expected error saving non-existent doc, got nil")
	}
}

func TestDocDeleteNonExistent(t *testing.T) {
	s := newTestStore(t)

	err := s.DeleteDoc("ghost", "")
	if err == nil {
		t.Error("expected error deleting non-existent doc, got nil")
	}
}
