package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/balazscsaba2006/specflow/internal/models"
)

// setupArchiveTest creates a temporary .specflow directory with an epic,
// stories, and optionally execution directories.
func setupArchiveTest(t *testing.T, epicStatus string, storyStatuses map[string]string) *Store {
	t.Helper()

	root := filepath.Join(t.TempDir(), ".specflow")
	s := New(root)
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}

	epic := &models.Epic{
		Slug:   "test-epic",
		Title:  "Test Epic",
		Status: epicStatus,
		Phases: []models.Phase{
			{Label: "P0", Stories: []string{"story-a", "story-b"}},
		},
		Body: "# Epic body\n\nThis should be stripped.",
	}
	if err := s.CreateEpic(epic); err != nil {
		t.Fatal(err)
	}

	for slug, status := range storyStatuses {
		st := &models.Story{
			Slug:       slug,
			Title:      "Story " + slug,
			Status:     models.StoryStatusDraft,
			Priority:   models.PriorityMedium,
			Epic:       "test-epic",
			Acceptance: []string{"AC1 for " + slug, "AC2 for " + slug},
			Body:       "# Story body\n\nThis should be stripped.",
		}
		if err := s.CreateStory(st); err != nil {
			t.Fatal(err)
		}
		// Transition through valid states to reach target status.
		transitions := transitionPath(status)
		for _, ts := range transitions {
			if err := s.UpdateStoryStatus(slug, "test-epic", ts); err != nil {
				t.Fatalf("transitioning %s to %s: %v", slug, ts, err)
			}
		}
	}

	return s
}

// transitionPath returns the sequence of transitions needed to reach a target status from draft.
func transitionPath(target string) []string {
	switch target {
	case models.StoryStatusDraft:
		return nil
	case models.StoryStatusPlanned:
		return []string{models.StoryStatusPlanned}
	case models.StoryStatusInProgress:
		return []string{models.StoryStatusPlanned, models.StoryStatusInProgress}
	case models.StoryStatusDone:
		return []string{models.StoryStatusPlanned, models.StoryStatusInProgress, models.StoryStatusDone}
	default:
		return nil
	}
}

// createTestExecutions creates dummy execution directories for a story.
func createTestExecutions(t *testing.T, s *Store, storySlug string) {
	t.Helper()
	execDir := s.StoryExecutionsDir(storySlug)
	subDir := filepath.Join(execDir, "x_test123")
	if err := os.MkdirAll(subDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "meta.yaml"), []byte("id: x_test123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestArchiveEpic(t *testing.T) {
	tests := []struct {
		name           string
		epicStatus     string
		storyStatuses  map[string]string
		force          bool
		withExecs      bool
		wantErr        string
		wantStoryCount int
		wantExecCount  int
	}{
		{
			name:       "successful archive of completed epic",
			epicStatus: models.EpicStatusCompleted,
			storyStatuses: map[string]string{
				"story-a": models.StoryStatusDone,
				"story-b": models.StoryStatusDone,
			},
			withExecs:      true,
			wantStoryCount: 2,
			wantExecCount:  2,
		},
		{
			name:       "completed epic without executions",
			epicStatus: models.EpicStatusCompleted,
			storyStatuses: map[string]string{
				"story-a": models.StoryStatusDone,
			},
			wantStoryCount: 1,
			wantExecCount:  0,
		},
		{
			name:       "non-completed epic without force",
			epicStatus: models.EpicStatusActive,
			storyStatuses: map[string]string{
				"story-a": models.StoryStatusDone,
			},
			wantErr: `epic "test-epic" has status "active"`,
		},
		{
			name:       "non-done stories without force",
			epicStatus: models.EpicStatusCompleted,
			storyStatuses: map[string]string{
				"story-a": models.StoryStatusDone,
				"story-b": models.StoryStatusInProgress,
			},
			wantErr: `story "story-b" has status "in_progress"`,
		},
		{
			name:       "force overrides status checks",
			epicStatus: models.EpicStatusActive,
			storyStatuses: map[string]string{
				"story-a": models.StoryStatusInProgress,
			},
			force:          true,
			wantStoryCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := setupArchiveTest(t, tt.epicStatus, tt.storyStatuses)

			if tt.withExecs {
				for slug := range tt.storyStatuses {
					createTestExecutions(t, s, slug)
				}
			}

			summary, err := s.ArchiveEpic("test-epic", tt.force)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if summary.StoryCount != tt.wantStoryCount {
				t.Errorf("story count: got %d, want %d", summary.StoryCount, tt.wantStoryCount)
			}
			if summary.ExecutionCount != tt.wantExecCount {
				t.Errorf("execution count: got %d, want %d", summary.ExecutionCount, tt.wantExecCount)
			}

			// Original epic dir should be gone.
			if _, statErr := os.Stat(s.EpicDir("test-epic")); !os.IsNotExist(statErr) {
				t.Error("original epic dir still exists")
			}

			// Archived epic should exist.
			if !s.IsArchived("test-epic") {
				t.Error("IsArchived returned false after archive")
			}

			// Archived epic should have no body and status=archived.
			archivedEpic, loadErr := s.LoadArchivedEpic("test-epic")
			if loadErr != nil {
				t.Fatalf("loading archived epic: %v", loadErr)
			}
			if archivedEpic.Body != "" {
				t.Errorf("archived epic body should be empty, got %q", archivedEpic.Body)
			}
			if archivedEpic.Status != models.EpicStatusArchived {
				t.Errorf("archived epic status: got %q, want %q", archivedEpic.Status, models.EpicStatusArchived)
			}
			if len(archivedEpic.Phases) == 0 {
				t.Error("archived epic phases should be preserved")
			}

			// Archived stories should have no body but keep acceptance criteria.
			for slug := range tt.storyStatuses {
				archivedSt, stErr := s.LoadArchivedStory(slug, "test-epic")
				if stErr != nil {
					t.Fatalf("loading archived story %q: %v", slug, stErr)
				}
				if archivedSt.Body != "" {
					t.Errorf("archived story %q body should be empty, got %q", slug, archivedSt.Body)
				}
				if len(archivedSt.Acceptance) == 0 {
					t.Errorf("archived story %q acceptance criteria should be preserved", slug)
				}
			}

			// Executions should be moved.
			if tt.withExecs {
				for slug := range tt.storyStatuses {
					archiveExecDir := s.ArchiveStoryExecutionsDir(slug)
					if _, statErr := os.Stat(archiveExecDir); os.IsNotExist(statErr) {
						t.Errorf("archived execution dir for %q not found", slug)
					}
					origExecDir := s.StoryExecutionsDir(slug)
					if _, statErr := os.Stat(origExecDir); !os.IsNotExist(statErr) {
						t.Errorf("original execution dir for %q still exists", slug)
					}
				}
			}
		})
	}
}

func TestArchiveEpic_NonExistent(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".specflow")
	s := New(root)
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}

	_, err := s.ArchiveEpic("nonexistent", false)
	if err == nil {
		t.Fatal("expected error for non-existent epic")
	}
}

func TestArchiveEpic_AlreadyArchived(t *testing.T) {
	s := setupArchiveTest(t, models.EpicStatusCompleted, map[string]string{
		"story-a": models.StoryStatusDone,
	})

	if _, err := s.ArchiveEpic("test-epic", false); err != nil {
		t.Fatalf("first archive: %v", err)
	}

	_, err := s.ArchiveEpic("test-epic", false)
	if err == nil {
		t.Fatal("expected error for already-archived epic")
	}
	if !contains(err.Error(), "already archived") {
		t.Fatalf("expected 'already archived' error, got %q", err.Error())
	}
}

func TestArchiveEpic_DocsPreserved(t *testing.T) {
	s := setupArchiveTest(t, models.EpicStatusCompleted, map[string]string{
		"story-a": models.StoryStatusDone,
	})

	// Create a doc in the epic.
	doc := &models.Document{
		Slug:  "test-prd",
		Type:  "prd",
		Title: "Test PRD",
		Epic:  "test-epic",
		Body:  "# PRD Content\n\nThis should be preserved.",
	}
	if err := s.CreateDoc(doc); err != nil {
		t.Fatal(err)
	}

	if _, err := s.ArchiveEpic("test-epic", false); err != nil {
		t.Fatalf("archive: %v", err)
	}

	// Doc should exist in archive with body preserved.
	docPath := filepath.Join(s.ArchiveEpicDocsDir("test-epic"), "test-prd.md")
	if _, statErr := os.Stat(docPath); os.IsNotExist(statErr) {
		t.Fatal("archived doc not found")
	}

	var d models.Document
	body, parseErr := ParseFile(docPath, &d)
	if parseErr != nil {
		t.Fatalf("parsing archived doc: %v", parseErr)
	}
	if body == "" {
		t.Error("archived doc body should be preserved (not compacted)")
	}
}

func TestListArchivedEpics(t *testing.T) {
	s := setupArchiveTest(t, models.EpicStatusCompleted, map[string]string{
		"story-a": models.StoryStatusDone,
	})

	// Before archive: no archived epics.
	epics, err := s.ListArchivedEpics()
	if err != nil {
		t.Fatal(err)
	}
	if len(epics) != 0 {
		t.Errorf("expected 0 archived epics, got %d", len(epics))
	}

	if _, archErr := s.ArchiveEpic("test-epic", false); archErr != nil {
		t.Fatal(archErr)
	}

	epics, err = s.ListArchivedEpics()
	if err != nil {
		t.Fatal(err)
	}
	if len(epics) != 1 {
		t.Fatalf("expected 1 archived epic, got %d", len(epics))
	}
	if epics[0].Slug != "test-epic" {
		t.Errorf("expected slug %q, got %q", "test-epic", epics[0].Slug)
	}
}

func TestListArchivedStories(t *testing.T) {
	s := setupArchiveTest(t, models.EpicStatusCompleted, map[string]string{
		"story-a": models.StoryStatusDone,
		"story-b": models.StoryStatusDone,
	})

	if _, err := s.ArchiveEpic("test-epic", false); err != nil {
		t.Fatal(err)
	}

	stories, err := s.ListArchivedStories("test-epic")
	if err != nil {
		t.Fatal(err)
	}
	if len(stories) != 2 {
		t.Fatalf("expected 2 archived stories, got %d", len(stories))
	}
}

func TestIsArchived(t *testing.T) {
	s := setupArchiveTest(t, models.EpicStatusCompleted, map[string]string{
		"story-a": models.StoryStatusDone,
	})

	if s.IsArchived("test-epic") {
		t.Error("should not be archived before archive")
	}
	if s.IsArchived("nonexistent") {
		t.Error("nonexistent epic should not be archived")
	}

	if _, err := s.ArchiveEpic("test-epic", false); err != nil {
		t.Fatal(err)
	}

	if !s.IsArchived("test-epic") {
		t.Error("should be archived after archive")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
