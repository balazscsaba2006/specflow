package store

import (
	"testing"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

func makeVerification(execID, storySlug string) *models.Verification {
	return &models.Verification{
		Execution: execID,
		Story:     storySlug,
		Result:    models.VerificationPass,
		Stats: models.VerificationStats{
			Critical: 0,
			Major:    1,
			Minor:    2,
		},
		Findings: []models.Finding{
			{
				Severity:    models.SeverityMajor,
				Category:    models.CategoryBug,
				File:        "handler.go",
				Description: "Missing nil check on input",
				Suggestion:  "Add guard clause",
			},
			{
				Severity:    models.SeverityMinor,
				Category:    models.CategoryClarity,
				Description: "Variable name could be clearer",
			},
			{
				Severity:    models.SeverityMinor,
				Category:    models.CategoryQuality,
				Description: "Missing doc comment on exported function",
			},
		},
		AcceptanceCheck: []models.AcceptanceCheck{
			{Criteria: "Handler returns 200", Met: true},
			{Criteria: "Tests pass", Met: true},
			{Criteria: "Error cases handled", Met: false},
		},
		Assumptions: []string{"No concurrent access needed"},
		Body:        "## Verification Notes\n\nOverall looks good with minor issues.",
	}
}

func TestVerificationSaveAndLoad(t *testing.T) {
	s := newTestStore(t)
	storySlug := "verify-story"

	// Create an execution first (verification needs an execution dir)
	e := &models.Execution{
		Story:        storySlug,
		GitRefBefore: "abc123",
	}
	if err := s.CreateExecution(e); err != nil {
		t.Fatalf("CreateExecution() error = %v", err)
	}

	v := makeVerification(e.ID, storySlug)

	if err := s.SaveVerification(v, storySlug, e.ID); err != nil {
		t.Fatalf("SaveVerification() error = %v", err)
	}

	// ID should be set with correct prefix
	if v.ID == "" {
		t.Fatal("expected ID to be set")
	}
	if v.ID[:2] != models.PrefixVerification {
		t.Errorf("ID prefix = %q, want %q", v.ID[:2], models.PrefixVerification)
	}

	// Created should be set
	if v.Created.IsZero() {
		t.Error("expected Created to be set")
	}

	// Load and verify roundtrip
	loaded, err := s.LoadVerification(storySlug, e.ID)
	if err != nil {
		t.Fatalf("LoadVerification() error = %v", err)
	}

	if loaded.ID != v.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, v.ID)
	}
	if loaded.Execution != v.Execution {
		t.Errorf("Execution = %q, want %q", loaded.Execution, v.Execution)
	}
	if loaded.Story != v.Story {
		t.Errorf("Story = %q, want %q", loaded.Story, v.Story)
	}
	if loaded.Result != v.Result {
		t.Errorf("Result = %q, want %q", loaded.Result, v.Result)
	}
	if !loaded.Created.Equal(v.Created) {
		t.Errorf("Created = %v, want %v", loaded.Created, v.Created)
	}

	// Stats
	if loaded.Stats.Critical != 0 {
		t.Errorf("Stats.Critical = %d, want 0", loaded.Stats.Critical)
	}
	if loaded.Stats.Major != 1 {
		t.Errorf("Stats.Major = %d, want 1", loaded.Stats.Major)
	}
	if loaded.Stats.Minor != 2 {
		t.Errorf("Stats.Minor = %d, want 2", loaded.Stats.Minor)
	}

	// Findings
	if len(loaded.Findings) != 3 {
		t.Fatalf("Findings len = %d, want 3", len(loaded.Findings))
	}
	if loaded.Findings[0].Severity != models.SeverityMajor {
		t.Errorf("Findings[0].Severity = %q, want %q", loaded.Findings[0].Severity, models.SeverityMajor)
	}
	if loaded.Findings[0].File != "handler.go" {
		t.Errorf("Findings[0].File = %q, want %q", loaded.Findings[0].File, "handler.go")
	}
	if loaded.Findings[0].Suggestion != "Add guard clause" {
		t.Errorf("Findings[0].Suggestion = %q, want %q", loaded.Findings[0].Suggestion, "Add guard clause")
	}

	// Acceptance checks
	if len(loaded.AcceptanceCheck) != 3 {
		t.Fatalf("AcceptanceCheck len = %d, want 3", len(loaded.AcceptanceCheck))
	}
	if loaded.AcceptanceCheck[0].Criteria != "Handler returns 200" {
		t.Errorf("AcceptanceCheck[0].Criteria = %q, want %q", loaded.AcceptanceCheck[0].Criteria, "Handler returns 200")
	}
	if !loaded.AcceptanceCheck[0].Met {
		t.Error("AcceptanceCheck[0].Met = false, want true")
	}
	if loaded.AcceptanceCheck[2].Met {
		t.Error("AcceptanceCheck[2].Met = true, want false")
	}

	// Assumptions
	if len(loaded.Assumptions) != 1 {
		t.Fatalf("Assumptions len = %d, want 1", len(loaded.Assumptions))
	}
	if loaded.Assumptions[0] != "No concurrent access needed" {
		t.Errorf("Assumptions[0] = %q, want %q", loaded.Assumptions[0], "No concurrent access needed")
	}

	// Body
	if loaded.Body != v.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, v.Body)
	}
}

func TestVerificationLoadNonExistent(t *testing.T) {
	s := newTestStore(t)

	_, err := s.LoadVerification("no-story", "x_fake")
	if err == nil {
		t.Error("expected error loading non-existent verification, got nil")
	}
}

func TestVerificationSaveNoExecutionDir(t *testing.T) {
	s := newTestStore(t)

	v := makeVerification("x_fake", "no-story")
	err := s.SaveVerification(v, "no-story", "x_fake")
	if err == nil {
		t.Error("expected error saving verification without execution dir, got nil")
	}
}

func TestVerificationLatest(t *testing.T) {
	s := newTestStore(t)
	storySlug := "latest-verify-story"

	baseTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	// Create two executions with distinct StartedAt
	e1 := &models.Execution{Story: storySlug, GitRefBefore: "ref1"}
	if err := s.CreateExecution(e1); err != nil {
		t.Fatalf("CreateExecution() #1 error = %v", err)
	}
	e1.StartedAt = baseTime
	if err := s.SaveExecution(e1); err != nil {
		t.Fatalf("SaveExecution() #1 error = %v", err)
	}

	e2 := &models.Execution{Story: storySlug, GitRefBefore: "ref2"}
	if err := s.CreateExecution(e2); err != nil {
		t.Fatalf("CreateExecution() #2 error = %v", err)
	}
	e2.StartedAt = baseTime.Add(time.Minute)
	if err := s.SaveExecution(e2); err != nil {
		t.Fatalf("SaveExecution() #2 error = %v", err)
	}

	// Save verification only on the latest execution
	v := makeVerification(e2.ID, storySlug)
	if err := s.SaveVerification(v, storySlug, e2.ID); err != nil {
		t.Fatalf("SaveVerification() error = %v", err)
	}

	latest, err := s.LatestVerification(storySlug)
	if err != nil {
		t.Fatalf("LatestVerification() error = %v", err)
	}
	if latest.Execution != e2.ID {
		t.Errorf("LatestVerification().Execution = %q, want %q", latest.Execution, e2.ID)
	}
}
