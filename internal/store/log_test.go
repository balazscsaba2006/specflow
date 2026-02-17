package store

import (
	"testing"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

func TestLogAppendAndRead(t *testing.T) {
	s := newTestStore(t)

	entry := models.LogEntry{
		Type:   models.LogStoryCreated,
		Entity: "s_abc123",
		Story:  "implement-auth",
	}

	if err := s.AppendLog(entry); err != nil {
		t.Fatalf("AppendLog() error = %v", err)
	}

	entries, err := s.ReadLog(0)
	if err != nil {
		t.Fatalf("ReadLog() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("ReadLog() returned %d entries, want 1", len(entries))
	}

	if entries[0].Type != models.LogStoryCreated {
		t.Errorf("Type = %q, want %q", entries[0].Type, models.LogStoryCreated)
	}
	if entries[0].Entity != "s_abc123" {
		t.Errorf("Entity = %q, want %q", entries[0].Entity, "s_abc123")
	}
	if entries[0].Story != "implement-auth" {
		t.Errorf("Story = %q, want %q", entries[0].Story, "implement-auth")
	}
	// Timestamp should be set automatically
	if entries[0].Timestamp.IsZero() {
		t.Error("expected Timestamp to be set")
	}
}

func TestLogReadLastN(t *testing.T) {
	s := newTestStore(t)

	types := []string{
		models.LogStoryCreated,
		models.LogPlanSaved,
		models.LogExecutionStarted,
		models.LogExecutionCompleted,
		models.LogVerificationSaved,
	}

	for _, typ := range types {
		entry := models.LogEntry{
			Type:   typ,
			Entity: "s_test",
		}
		if err := s.AppendLog(entry); err != nil {
			t.Fatalf("AppendLog(%q) error = %v", typ, err)
		}
	}

	// Read last 3
	entries, err := s.ReadLog(3)
	if err != nil {
		t.Fatalf("ReadLog(3) error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("ReadLog(3) returned %d entries, want 3", len(entries))
	}

	// Should be the last 3 entries
	if entries[0].Type != models.LogExecutionStarted {
		t.Errorf("entries[0].Type = %q, want %q", entries[0].Type, models.LogExecutionStarted)
	}
	if entries[1].Type != models.LogExecutionCompleted {
		t.Errorf("entries[1].Type = %q, want %q", entries[1].Type, models.LogExecutionCompleted)
	}
	if entries[2].Type != models.LogVerificationSaved {
		t.Errorf("entries[2].Type = %q, want %q", entries[2].Type, models.LogVerificationSaved)
	}
}

func TestLogReadEmptyReturnsNil(t *testing.T) {
	s := newTestStore(t)

	entries, err := s.ReadLog(0)
	if err != nil {
		t.Fatalf("ReadLog() error = %v", err)
	}
	if entries != nil {
		t.Errorf("ReadLog() on empty = %v, want nil", entries)
	}
}

func TestLogMultipleAppendsInOrder(t *testing.T) {
	s := newTestStore(t)

	for i := 0; i < 5; i++ {
		entry := models.LogEntry{
			Timestamp: time.Date(2025, 1, 1, 0, 0, i, 0, time.UTC),
			Type:      models.LogStoryStatusChanged,
			Entity:    "s_test",
			From:      "draft",
			To:        "ready",
		}
		if err := s.AppendLog(entry); err != nil {
			t.Fatalf("AppendLog() #%d error = %v", i, err)
		}
	}

	entries, err := s.ReadLog(0)
	if err != nil {
		t.Fatalf("ReadLog() error = %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("ReadLog() returned %d entries, want 5", len(entries))
	}

	// Verify ordering is preserved
	for i := 0; i < 5; i++ {
		expected := time.Date(2025, 1, 1, 0, 0, i, 0, time.UTC)
		if !entries[i].Timestamp.Equal(expected) {
			t.Errorf("entries[%d].Timestamp = %v, want %v", i, entries[i].Timestamp, expected)
		}
	}
}

func TestLogReadAllWithLastZero(t *testing.T) {
	s := newTestStore(t)

	for i := 0; i < 3; i++ {
		entry := models.LogEntry{
			Type:   models.LogEpicCreated,
			Entity: "e_test",
		}
		if err := s.AppendLog(entry); err != nil {
			t.Fatalf("AppendLog() #%d error = %v", i, err)
		}
	}

	// last=0 means return all
	entries, err := s.ReadLog(0)
	if err != nil {
		t.Fatalf("ReadLog(0) error = %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("ReadLog(0) returned %d entries, want 3", len(entries))
	}
}

func TestLogReadLastLargerThanTotal(t *testing.T) {
	s := newTestStore(t)

	entry := models.LogEntry{
		Type:   models.LogDocCreated,
		Entity: "d_test",
	}
	if err := s.AppendLog(entry); err != nil {
		t.Fatalf("AppendLog() error = %v", err)
	}

	// Requesting last 100 when only 1 exists should return all
	entries, err := s.ReadLog(100)
	if err != nil {
		t.Fatalf("ReadLog(100) error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("ReadLog(100) returned %d entries, want 1", len(entries))
	}
}

func TestLogEntryWithOptionalFields(t *testing.T) {
	s := newTestStore(t)

	entry := models.LogEntry{
		Type:         models.LogVerificationSaved,
		Entity:       "v_test",
		Story:        "auth-story",
		Epic:         "auth-epic",
		GitRef:       "deadbeef",
		FilesChanged: 5,
		Result:       models.VerificationPass,
		Critical:     0,
		Major:        1,
		Minor:        3,
	}

	if err := s.AppendLog(entry); err != nil {
		t.Fatalf("AppendLog() error = %v", err)
	}

	entries, err := s.ReadLog(0)
	if err != nil {
		t.Fatalf("ReadLog() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ReadLog() returned %d entries, want 1", len(entries))
	}

	e := entries[0]
	if e.Epic != "auth-epic" {
		t.Errorf("Epic = %q, want %q", e.Epic, "auth-epic")
	}
	if e.GitRef != "deadbeef" {
		t.Errorf("GitRef = %q, want %q", e.GitRef, "deadbeef")
	}
	if e.FilesChanged != 5 {
		t.Errorf("FilesChanged = %d, want 5", e.FilesChanged)
	}
	if e.Result != models.VerificationPass {
		t.Errorf("Result = %q, want %q", e.Result, models.VerificationPass)
	}
	if e.Critical != 0 {
		t.Errorf("Critical = %d, want 0", e.Critical)
	}
	if e.Major != 1 {
		t.Errorf("Major = %d, want 1", e.Major)
	}
	if e.Minor != 3 {
		t.Errorf("Minor = %d, want 3", e.Minor)
	}
}
