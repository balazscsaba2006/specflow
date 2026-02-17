package store

import (
	"os"
	"testing"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

func makeDecision(slug, title string) *models.Decision {
	return &models.Decision{
		Slug:        slug,
		Title:       title,
		ContextRefs: []string{"e_auth-epic", "d_api-spec"},
		Body:        "# Decision\n\nWe decided to use JWT tokens.",
	}
}

func TestDecisionCreateAndLoad(t *testing.T) {
	s := newTestStore(t)
	d := makeDecision("use-jwt", "Use JWT for Auth")

	if err := s.CreateDecision(d); err != nil {
		t.Fatalf("CreateDecision() error = %v", err)
	}

	// ID should be set with correct prefix
	if d.ID == "" {
		t.Fatal("expected ID to be set")
	}
	if d.ID[:4] != models.PrefixDecision {
		t.Errorf("ID prefix = %q, want %q", d.ID[:4], models.PrefixDecision)
	}

	// File should exist
	if _, err := os.Stat(s.DecisionFile(d.Slug)); err != nil {
		t.Fatalf("decision file does not exist: %v", err)
	}

	// Load and verify roundtrip
	loaded, err := s.LoadDecision("use-jwt")
	if err != nil {
		t.Fatalf("LoadDecision() error = %v", err)
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
	if loaded.Status != d.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, d.Status)
	}
	if loaded.Date != d.Date {
		t.Errorf("Date = %q, want %q", loaded.Date, d.Date)
	}
	if len(loaded.ContextRefs) != len(d.ContextRefs) {
		t.Fatalf("ContextRefs len = %d, want %d", len(loaded.ContextRefs), len(d.ContextRefs))
	}
	for i, ref := range loaded.ContextRefs {
		if ref != d.ContextRefs[i] {
			t.Errorf("ContextRefs[%d] = %q, want %q", i, ref, d.ContextRefs[i])
		}
	}
	if loaded.Body != d.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, d.Body)
	}
}

func TestDecisionDefaults(t *testing.T) {
	s := newTestStore(t)
	d := makeDecision("defaults-test", "Defaults Test")

	// No status or date set
	if d.Status != "" {
		t.Fatal("precondition failed: status should be empty before create")
	}
	if d.Date != "" {
		t.Fatal("precondition failed: date should be empty before create")
	}

	if err := s.CreateDecision(d); err != nil {
		t.Fatalf("CreateDecision() error = %v", err)
	}

	if d.Status != models.DecisionStatusAccepted {
		t.Errorf("default Status = %q, want %q", d.Status, models.DecisionStatusAccepted)
	}

	expectedDate := time.Now().UTC().Format("2006-01-02")
	if d.Date != expectedDate {
		t.Errorf("default Date = %q, want %q", d.Date, expectedDate)
	}

	// Verify on reload
	loaded, err := s.LoadDecision("defaults-test")
	if err != nil {
		t.Fatalf("LoadDecision() error = %v", err)
	}
	if loaded.Status != models.DecisionStatusAccepted {
		t.Errorf("loaded Status = %q, want %q", loaded.Status, models.DecisionStatusAccepted)
	}
	if loaded.Date != expectedDate {
		t.Errorf("loaded Date = %q, want %q", loaded.Date, expectedDate)
	}
}

func TestDecisionList(t *testing.T) {
	s := newTestStore(t)

	slugs := []string{"dec-alpha", "dec-beta", "dec-gamma"}
	for _, slug := range slugs {
		d := makeDecision(slug, "Title "+slug)
		if err := s.CreateDecision(d); err != nil {
			t.Fatalf("CreateDecision(%q) error = %v", slug, err)
		}
	}

	list, err := s.ListDecisions()
	if err != nil {
		t.Fatalf("ListDecisions() error = %v", err)
	}
	if len(list) != len(slugs) {
		t.Fatalf("ListDecisions() returned %d, want %d", len(list), len(slugs))
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

func TestDecisionSavePreservesBody(t *testing.T) {
	s := newTestStore(t)
	d := makeDecision("body-test", "Body Test")
	d.Body = "# Original Decision\n\nWith rationale."
	if err := s.CreateDecision(d); err != nil {
		t.Fatalf("CreateDecision() error = %v", err)
	}

	// Update title, keep body
	d.Title = "Updated Title"
	if err := s.SaveDecision(d); err != nil {
		t.Fatalf("SaveDecision() error = %v", err)
	}

	loaded, err := s.LoadDecision("body-test")
	if err != nil {
		t.Fatalf("LoadDecision() error = %v", err)
	}
	if loaded.Body != "# Original Decision\n\nWith rationale." {
		t.Errorf("Body = %q, want %q", loaded.Body, "# Original Decision\n\nWith rationale.")
	}
	if loaded.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Updated Title")
	}
}

func TestDecisionDelete(t *testing.T) {
	s := newTestStore(t)
	d := makeDecision("to-delete", "Delete Me")
	if err := s.CreateDecision(d); err != nil {
		t.Fatalf("CreateDecision() error = %v", err)
	}

	// Verify it exists
	if _, err := s.LoadDecision("to-delete"); err != nil {
		t.Fatalf("decision should exist before delete: %v", err)
	}

	if err := s.DeleteDecision("to-delete"); err != nil {
		t.Fatalf("DeleteDecision() error = %v", err)
	}

	// File should be gone
	if _, err := os.Stat(s.DecisionFile("to-delete")); !os.IsNotExist(err) {
		t.Error("expected decision file to be removed")
	}

	// Load should fail
	if _, err := s.LoadDecision("to-delete"); err == nil {
		t.Error("expected error loading deleted decision, got nil")
	}
}

func TestDecisionLoadNonExistent(t *testing.T) {
	s := newTestStore(t)

	_, err := s.LoadDecision("does-not-exist")
	if err == nil {
		t.Error("expected error loading non-existent decision, got nil")
	}
}

func TestDecisionCreateDuplicateSlug(t *testing.T) {
	s := newTestStore(t)

	d := makeDecision("dup-dec", "First")
	if err := s.CreateDecision(d); err != nil {
		t.Fatalf("first CreateDecision() error = %v", err)
	}

	dup := makeDecision("dup-dec", "Second")
	if err := s.CreateDecision(dup); err == nil {
		t.Error("expected error for duplicate slug, got nil")
	}
}

func TestDecisionSaveNonExistent(t *testing.T) {
	s := newTestStore(t)

	d := makeDecision("no-such-dec", "Nope")
	d.ID = "dec_fake"
	d.Status = models.DecisionStatusAccepted
	err := s.SaveDecision(d)
	if err == nil {
		t.Error("expected error saving non-existent decision, got nil")
	}
}

func TestDecisionDeleteNonExistent(t *testing.T) {
	s := newTestStore(t)

	err := s.DeleteDecision("ghost")
	if err == nil {
		t.Error("expected error deleting non-existent decision, got nil")
	}
}
