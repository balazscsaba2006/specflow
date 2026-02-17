package store

import (
	"testing"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

func TestExecutionCreateAndLoad(t *testing.T) {
	s := newTestStore(t)
	storySlug := "auth-handler"

	e := &models.Execution{
		Story:        storySlug,
		Plan:         "p_someplain",
		GitRefBefore: "deadbeef",
	}

	if err := s.CreateExecution(e); err != nil {
		t.Fatalf("CreateExecution() error = %v", err)
	}

	// ID should be set with correct prefix
	if e.ID == "" {
		t.Fatal("expected ID to be set")
	}
	if e.ID[:2] != models.PrefixExecution {
		t.Errorf("ID prefix = %q, want %q", e.ID[:2], models.PrefixExecution)
	}

	// StartedAt should be set
	if e.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}

	// Status should be started
	if e.Status != models.ExecutionStatusStarted {
		t.Errorf("Status = %q, want %q", e.Status, models.ExecutionStatusStarted)
	}

	// Load and verify roundtrip
	loaded, err := s.LoadExecution(storySlug, e.ID)
	if err != nil {
		t.Fatalf("LoadExecution() error = %v", err)
	}

	if loaded.ID != e.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, e.ID)
	}
	if loaded.Story != e.Story {
		t.Errorf("Story = %q, want %q", loaded.Story, e.Story)
	}
	if loaded.Plan != e.Plan {
		t.Errorf("Plan = %q, want %q", loaded.Plan, e.Plan)
	}
	if loaded.Status != e.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, e.Status)
	}
	if loaded.GitRefBefore != e.GitRefBefore {
		t.Errorf("GitRefBefore = %q, want %q", loaded.GitRefBefore, e.GitRefBefore)
	}
	if !loaded.StartedAt.Equal(e.StartedAt) {
		t.Errorf("StartedAt = %v, want %v", loaded.StartedAt, e.StartedAt)
	}
}

func TestExecutionListMultiple(t *testing.T) {
	s := newTestStore(t)
	storySlug := "multi-exec-story"

	for i := 0; i < 3; i++ {
		e := &models.Execution{
			Story:        storySlug,
			GitRefBefore: "ref" + string(rune('a'+i)),
		}
		if err := s.CreateExecution(e); err != nil {
			t.Fatalf("CreateExecution() #%d error = %v", i, err)
		}
		// Small delay so StartedAt differs (ULID is ms-resolution)
		time.Sleep(2 * time.Millisecond)
	}

	list, err := s.ListExecutions(storySlug)
	if err != nil {
		t.Fatalf("ListExecutions() error = %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("ListExecutions() returned %d, want 3", len(list))
	}
}

func TestExecutionLatest(t *testing.T) {
	s := newTestStore(t)
	storySlug := "latest-exec-story"

	baseTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	var lastID string
	for i := 0; i < 3; i++ {
		e := &models.Execution{
			Story:        storySlug,
			GitRefBefore: "ref" + string(rune('a'+i)),
		}
		if err := s.CreateExecution(e); err != nil {
			t.Fatalf("CreateExecution() #%d error = %v", i, err)
		}
		// Override StartedAt to guarantee distinct, ordered timestamps
		e.StartedAt = baseTime.Add(time.Duration(i) * time.Minute)
		if err := s.SaveExecution(e); err != nil {
			t.Fatalf("SaveExecution() #%d error = %v", i, err)
		}
		lastID = e.ID
	}

	latest, err := s.LatestExecution(storySlug)
	if err != nil {
		t.Fatalf("LatestExecution() error = %v", err)
	}
	if latest.ID != lastID {
		t.Errorf("LatestExecution().ID = %q, want %q", latest.ID, lastID)
	}
}

func TestExecutionSaveUpdate(t *testing.T) {
	s := newTestStore(t)
	storySlug := "update-exec-story"

	e := &models.Execution{
		Story:        storySlug,
		GitRefBefore: "before123",
	}
	if err := s.CreateExecution(e); err != nil {
		t.Fatalf("CreateExecution() error = %v", err)
	}

	// Complete the execution
	now := time.Now().UTC().Truncate(time.Second)
	e.Status = models.ExecutionStatusCompleted
	e.CompletedAt = &now
	e.GitRefAfter = "after456"
	e.FilesChanged = []models.FileChange{
		{Path: "handler.go", Action: "modified"},
		{Path: "handler_test.go", Action: "created"},
	}

	if err := s.SaveExecution(e); err != nil {
		t.Fatalf("SaveExecution() error = %v", err)
	}

	loaded, err := s.LoadExecution(storySlug, e.ID)
	if err != nil {
		t.Fatalf("LoadExecution() error = %v", err)
	}

	if loaded.Status != models.ExecutionStatusCompleted {
		t.Errorf("Status = %q, want %q", loaded.Status, models.ExecutionStatusCompleted)
	}
	if loaded.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
	if !loaded.CompletedAt.Equal(now) {
		t.Errorf("CompletedAt = %v, want %v", loaded.CompletedAt, now)
	}
	if loaded.GitRefAfter != "after456" {
		t.Errorf("GitRefAfter = %q, want %q", loaded.GitRefAfter, "after456")
	}
	if len(loaded.FilesChanged) != 2 {
		t.Fatalf("FilesChanged len = %d, want 2", len(loaded.FilesChanged))
	}
	if loaded.FilesChanged[0].Path != "handler.go" {
		t.Errorf("FilesChanged[0].Path = %q, want %q", loaded.FilesChanged[0].Path, "handler.go")
	}
	if loaded.FilesChanged[1].Action != "created" {
		t.Errorf("FilesChanged[1].Action = %q, want %q", loaded.FilesChanged[1].Action, "created")
	}
}

func TestExecutionLoadNonExistent(t *testing.T) {
	s := newTestStore(t)

	_, err := s.LoadExecution("no-story", "x_fake")
	if err == nil {
		t.Error("expected error loading non-existent execution, got nil")
	}
}

func TestExecutionLatestNoExecutions(t *testing.T) {
	s := newTestStore(t)

	_, err := s.LatestExecution("empty-story")
	if err == nil {
		t.Error("expected error for LatestExecution on story with no executions, got nil")
	}
}

func TestExecutionListEmpty(t *testing.T) {
	s := newTestStore(t)

	list, err := s.ListExecutions("no-such-story")
	if err != nil {
		t.Fatalf("ListExecutions() error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("ListExecutions() returned %d, want 0", len(list))
	}
}
