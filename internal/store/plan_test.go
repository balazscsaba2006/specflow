package store

import (
	"os"
	"testing"

	"github.com/balazscsaba2006/specflow/internal/models"
)

func makePlan(storySlug string) *models.Plan {
	return &models.Plan{
		Story:          storySlug,
		Status:         models.PlanStatusDraft,
		GitRefBaseline: "abc1234",
		EstimatedFiles: 5,
		Body:           "## Steps\n\n1. Create the handler\n2. Write tests",
	}
}

func TestPlanSaveAndLoad(t *testing.T) {
	s := newTestStore(t)
	storySlug := "implement-auth"
	p := makePlan(storySlug)

	if err := s.SavePlan(p, storySlug); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// ID should be set with correct prefix
	if p.ID == "" {
		t.Fatal("expected ID to be set")
	}
	if p.ID[:2] != models.PrefixPlan {
		t.Errorf("ID prefix = %q, want %q", p.ID[:2], models.PrefixPlan)
	}

	// Created should be set
	if p.Created.IsZero() {
		t.Error("expected Created to be set")
	}

	// File should exist
	if _, err := os.Stat(s.PlanFile(storySlug, "latest")); err != nil {
		t.Fatalf("plan file does not exist: %v", err)
	}

	// Load and verify roundtrip
	loaded, err := s.LoadPlan(storySlug)
	if err != nil {
		t.Fatalf("LoadPlan() error = %v", err)
	}

	if loaded.ID != p.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, p.ID)
	}
	if loaded.Story != p.Story {
		t.Errorf("Story = %q, want %q", loaded.Story, p.Story)
	}
	if loaded.Status != p.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, p.Status)
	}
	if loaded.GitRefBaseline != p.GitRefBaseline {
		t.Errorf("GitRefBaseline = %q, want %q", loaded.GitRefBaseline, p.GitRefBaseline)
	}
	if loaded.EstimatedFiles != p.EstimatedFiles {
		t.Errorf("EstimatedFiles = %d, want %d", loaded.EstimatedFiles, p.EstimatedFiles)
	}
	if loaded.Body != p.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, p.Body)
	}
	if !loaded.Created.Equal(p.Created) {
		t.Errorf("Created = %v, want %v", loaded.Created, p.Created)
	}
}

func TestPlanLoadNonExistent(t *testing.T) {
	s := newTestStore(t)

	_, err := s.LoadPlan("no-such-story")
	if err == nil {
		t.Error("expected error loading non-existent plan, got nil")
	}
}

func TestPlanDelete(t *testing.T) {
	s := newTestStore(t)
	storySlug := "delete-plan-story"
	p := makePlan(storySlug)

	if err := s.SavePlan(p, storySlug); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	// Verify it exists
	if _, err := s.LoadPlan(storySlug); err != nil {
		t.Fatalf("plan should exist before delete: %v", err)
	}

	if err := s.DeletePlan(storySlug); err != nil {
		t.Fatalf("DeletePlan() error = %v", err)
	}

	// File should be gone
	if _, err := os.Stat(s.PlanFile(storySlug, "latest")); !os.IsNotExist(err) {
		t.Error("expected plan file to be removed")
	}

	// Load should fail
	if _, err := s.LoadPlan(storySlug); err == nil {
		t.Error("expected error loading deleted plan, got nil")
	}
}

func TestPlanDeleteNonExistent(t *testing.T) {
	s := newTestStore(t)

	err := s.DeletePlan("ghost-story")
	if err == nil {
		t.Error("expected error deleting non-existent plan, got nil")
	}
}

func TestPlanSaveOverwrite(t *testing.T) {
	s := newTestStore(t)
	storySlug := "overwrite-story"
	p := makePlan(storySlug)

	if err := s.SavePlan(p, storySlug); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}

	originalID := p.ID

	// Update and save again
	p.Status = models.PlanStatusApproved
	p.Body = "## Updated Plan\n\nRevised steps."

	if err := s.SavePlan(p, storySlug); err != nil {
		t.Fatalf("SavePlan() second call error = %v", err)
	}

	// ID should not change (already set)
	if p.ID != originalID {
		t.Errorf("ID changed on re-save: got %q, want %q", p.ID, originalID)
	}

	loaded, err := s.LoadPlan(storySlug)
	if err != nil {
		t.Fatalf("LoadPlan() error = %v", err)
	}
	if loaded.Status != models.PlanStatusApproved {
		t.Errorf("Status = %q, want %q", loaded.Status, models.PlanStatusApproved)
	}
	if loaded.Body != "## Updated Plan\n\nRevised steps." {
		t.Errorf("Body = %q, want %q", loaded.Body, "## Updated Plan\n\nRevised steps.")
	}
}
