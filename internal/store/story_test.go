package store

import (
	"os"
	"testing"

	"github.com/balazscsaba2006/specflow/internal/models"
)

func makeStory(slug, title string) *models.Story {
	return &models.Story{
		Slug:       slug,
		Title:      title,
		Acceptance: []string{"it works", "tests pass"},
		Body:       "# Story\n\nDetailed description here.",
	}
}

func makeEpicScopedStory(slug, title, epicSlug string) *models.Story {
	st := makeStory(slug, title)
	st.Epic = epicSlug
	return st
}

// createEpicDir sets up the epic stories directory so epic-scoped story tests work.
func createEpicDir(t *testing.T, s *Store, epicSlug string) {
	t.Helper()
	if err := s.EnsureDir(s.EpicStoriesDir(epicSlug)); err != nil {
		t.Fatalf("creating epic stories dir: %v", err)
	}
}

func TestStoryCreateAndLoad(t *testing.T) {
	s := newTestStore(t)
	st := makeStory("implement-auth", "Implement Auth")

	if err := s.CreateStory(st); err != nil {
		t.Fatalf("CreateStory() error = %v", err)
	}

	// ID should be set with correct prefix
	if st.ID == "" {
		t.Fatal("expected ID to be set")
	}
	if st.ID[:2] != models.PrefixStory {
		t.Errorf("ID prefix = %q, want %q", st.ID[:2], models.PrefixStory)
	}

	// Timestamps should be set
	if st.Created.IsZero() {
		t.Error("expected Created to be set")
	}
	if st.Updated.IsZero() {
		t.Error("expected Updated to be set")
	}

	// File should exist
	if _, err := os.Stat(s.StoryFile(st.Slug, "")); err != nil {
		t.Fatalf("story file does not exist: %v", err)
	}

	// Load and verify roundtrip
	loaded, err := s.LoadStory("implement-auth", "")
	if err != nil {
		t.Fatalf("LoadStory() error = %v", err)
	}

	if loaded.ID != st.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, st.ID)
	}
	if loaded.Slug != st.Slug {
		t.Errorf("Slug = %q, want %q", loaded.Slug, st.Slug)
	}
	if loaded.Title != st.Title {
		t.Errorf("Title = %q, want %q", loaded.Title, st.Title)
	}
	if loaded.Status != st.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, st.Status)
	}
	if loaded.Priority != st.Priority {
		t.Errorf("Priority = %q, want %q", loaded.Priority, st.Priority)
	}
	if len(loaded.Acceptance) != len(st.Acceptance) {
		t.Errorf("Acceptance len = %d, want %d", len(loaded.Acceptance), len(st.Acceptance))
	}
	if loaded.Body != st.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, st.Body)
	}
	if !loaded.Created.Equal(st.Created) {
		t.Errorf("Created = %v, want %v", loaded.Created, st.Created)
	}
	if !loaded.Updated.Equal(st.Updated) {
		t.Errorf("Updated = %v, want %v", loaded.Updated, st.Updated)
	}
}

func TestStoryCreateEpicScopedAndLoad(t *testing.T) {
	s := newTestStore(t)
	epicSlug := "auth-epic"
	createEpicDir(t, s, epicSlug)

	st := makeEpicScopedStory("001-create-model", "Create Auth Model", epicSlug)

	if err := s.CreateStory(st); err != nil {
		t.Fatalf("CreateStory() error = %v", err)
	}

	// File should be under epic stories dir
	if _, err := os.Stat(s.StoryFile(st.Slug, epicSlug)); err != nil {
		t.Fatalf("epic-scoped story file does not exist: %v", err)
	}

	// Load and verify roundtrip
	loaded, err := s.LoadStory("001-create-model", epicSlug)
	if err != nil {
		t.Fatalf("LoadStory() error = %v", err)
	}

	if loaded.ID != st.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, st.ID)
	}
	if loaded.Slug != st.Slug {
		t.Errorf("Slug = %q, want %q", loaded.Slug, st.Slug)
	}
	if loaded.Title != st.Title {
		t.Errorf("Title = %q, want %q", loaded.Title, st.Title)
	}
	if loaded.Epic != epicSlug {
		t.Errorf("Epic = %q, want %q", loaded.Epic, epicSlug)
	}
	if loaded.Body != st.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, st.Body)
	}
}

func TestStoryDefaults(t *testing.T) {
	s := newTestStore(t)
	st := makeStory("defaults-test", "Defaults Test")

	// No status or priority set
	if st.Status != "" {
		t.Fatal("precondition failed: status should be empty before create")
	}
	if st.Priority != "" {
		t.Fatal("precondition failed: priority should be empty before create")
	}

	if err := s.CreateStory(st); err != nil {
		t.Fatalf("CreateStory() error = %v", err)
	}

	if st.Status != models.StoryStatusDraft {
		t.Errorf("default Status = %q, want %q", st.Status, models.StoryStatusDraft)
	}
	if st.Priority != models.PriorityMedium {
		t.Errorf("default Priority = %q, want %q", st.Priority, models.PriorityMedium)
	}

	// Verify on reload
	loaded, err := s.LoadStory("defaults-test", "")
	if err != nil {
		t.Fatalf("LoadStory() error = %v", err)
	}
	if loaded.Status != models.StoryStatusDraft {
		t.Errorf("loaded Status = %q, want %q", loaded.Status, models.StoryStatusDraft)
	}
	if loaded.Priority != models.PriorityMedium {
		t.Errorf("loaded Priority = %q, want %q", loaded.Priority, models.PriorityMedium)
	}
}

func TestStoryCreateInvalidSlug(t *testing.T) {
	s := newTestStore(t)

	tests := []struct {
		name string
		slug string
	}{
		{"empty", ""},
		{"uppercase", "My-Story"},
		{"spaces", "my story"},
		{"special chars", "my_story!"},
		{"leading hyphen", "-my-story"},
		{"trailing hyphen", "my-story-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := makeStory(tt.slug, "Title")
			if err := s.CreateStory(st); err == nil {
				t.Errorf("CreateStory(%q) expected error, got nil", tt.slug)
			}
		})
	}
}

func TestStoryListStandalone(t *testing.T) {
	s := newTestStore(t)

	slugs := []string{"alpha", "beta", "gamma"}
	for _, slug := range slugs {
		st := makeStory(slug, "Title "+slug)
		if err := s.CreateStory(st); err != nil {
			t.Fatalf("CreateStory(%q) error = %v", slug, err)
		}
	}

	list, err := s.ListStories("")
	if err != nil {
		t.Fatalf("ListStories() error = %v", err)
	}
	if len(list) != len(slugs) {
		t.Fatalf("ListStories() returned %d, want %d", len(list), len(slugs))
	}

	found := make(map[string]bool)
	for _, st := range list {
		found[st.Slug] = true
	}
	for _, slug := range slugs {
		if !found[slug] {
			t.Errorf("slug %q not found in list", slug)
		}
	}
}

func TestStoryListEpicScoped(t *testing.T) {
	s := newTestStore(t)
	epicSlug := "my-epic"
	createEpicDir(t, s, epicSlug)

	slugs := []string{"001-first", "002-second", "003-third"}
	for _, slug := range slugs {
		st := makeEpicScopedStory(slug, "Title "+slug, epicSlug)
		if err := s.CreateStory(st); err != nil {
			t.Fatalf("CreateStory(%q) error = %v", slug, err)
		}
	}

	list, err := s.ListStories(epicSlug)
	if err != nil {
		t.Fatalf("ListStories(%q) error = %v", epicSlug, err)
	}
	if len(list) != len(slugs) {
		t.Fatalf("ListStories(%q) returned %d, want %d", epicSlug, len(list), len(slugs))
	}

	found := make(map[string]bool)
	for _, st := range list {
		found[st.Slug] = true
	}
	for _, slug := range slugs {
		if !found[slug] {
			t.Errorf("slug %q not found in epic-scoped list", slug)
		}
	}
}

func TestStoryListAllStories(t *testing.T) {
	s := newTestStore(t)

	// Create standalone stories
	for _, slug := range []string{"standalone-a", "standalone-b"} {
		st := makeStory(slug, "Title "+slug)
		if err := s.CreateStory(st); err != nil {
			t.Fatalf("CreateStory(%q) error = %v", slug, err)
		}
	}

	// Create epic-scoped stories across two epics
	for _, epicSlug := range []string{"epic-one", "epic-two"} {
		createEpicDir(t, s, epicSlug)
		st := makeEpicScopedStory("story-in-"+epicSlug, "Title", epicSlug)
		if err := s.CreateStory(st); err != nil {
			t.Fatalf("CreateStory() error = %v", err)
		}
	}

	all, err := s.ListAllStories()
	if err != nil {
		t.Fatalf("ListAllStories() error = %v", err)
	}

	// 2 standalone + 2 epic-scoped = 4
	if len(all) != 4 {
		t.Fatalf("ListAllStories() returned %d, want 4", len(all))
	}

	expected := map[string]bool{
		"standalone-a":       false,
		"standalone-b":       false,
		"story-in-epic-one":  false,
		"story-in-epic-two":  false,
	}
	for _, st := range all {
		if _, ok := expected[st.Slug]; !ok {
			t.Errorf("unexpected slug %q in ListAllStories()", st.Slug)
		}
		expected[st.Slug] = true
	}
	for slug, found := range expected {
		if !found {
			t.Errorf("slug %q not found in ListAllStories()", slug)
		}
	}
}

func TestStoryStatusTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{"draft to planned", models.StoryStatusDraft, models.StoryStatusPlanned, false},
		{"planned to in_progress", models.StoryStatusPlanned, models.StoryStatusInProgress, false},
		{"in_progress to done", models.StoryStatusInProgress, models.StoryStatusDone, false},
		{"in_progress to verifying", models.StoryStatusInProgress, models.StoryStatusVerifying, false},
		{"verifying to done", models.StoryStatusVerifying, models.StoryStatusDone, false},
		{"draft to done (invalid)", models.StoryStatusDraft, models.StoryStatusDone, true},
		{"done to planned (invalid)", models.StoryStatusDone, models.StoryStatusPlanned, true},
		{"draft to blocked", models.StoryStatusDraft, models.StoryStatusBlocked, false},
		{"blocked to planned", models.StoryStatusBlocked, models.StoryStatusPlanned, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)

			st := makeStory("transition-test", "Transition Test")
			st.Status = tt.from
			if err := s.CreateStory(st); err != nil {
				t.Fatalf("CreateStory() error = %v", err)
			}

			st.Status = tt.to
			err := s.SaveStory(st)
			if tt.wantErr && err == nil {
				t.Errorf("SaveStory() transition %s -> %s: expected error, got nil", tt.from, tt.to)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("SaveStory() transition %s -> %s: unexpected error = %v", tt.from, tt.to, err)
			}
		})
	}
}

func TestStoryUpdateStoryStatus(t *testing.T) {
	s := newTestStore(t)
	st := makeStory("status-update", "Status Update Test")
	if err := s.CreateStory(st); err != nil {
		t.Fatalf("CreateStory() error = %v", err)
	}

	// draft -> planned should work
	if err := s.UpdateStoryStatus("status-update", "", models.StoryStatusPlanned); err != nil {
		t.Fatalf("UpdateStoryStatus() error = %v", err)
	}

	loaded, err := s.LoadStory("status-update", "")
	if err != nil {
		t.Fatalf("LoadStory() error = %v", err)
	}
	if loaded.Status != models.StoryStatusPlanned {
		t.Errorf("Status = %q, want %q", loaded.Status, models.StoryStatusPlanned)
	}

	// planned -> done should fail (invalid transition)
	if err := s.UpdateStoryStatus("status-update", "", models.StoryStatusDone); err == nil {
		t.Error("UpdateStoryStatus() planned -> done: expected error, got nil")
	}
}

func TestStorySavePreservesBody(t *testing.T) {
	s := newTestStore(t)
	st := makeStory("body-test", "Body Test")
	st.Body = "# Original Body\n\nWith some content."
	if err := s.CreateStory(st); err != nil {
		t.Fatalf("CreateStory() error = %v", err)
	}

	// Update title, keep body
	st.Title = "Updated Title"
	if err := s.SaveStory(st); err != nil {
		t.Fatalf("SaveStory() error = %v", err)
	}

	loaded, err := s.LoadStory("body-test", "")
	if err != nil {
		t.Fatalf("LoadStory() error = %v", err)
	}
	if loaded.Body != "# Original Body\n\nWith some content." {
		t.Errorf("Body = %q, want %q", loaded.Body, "# Original Body\n\nWith some content.")
	}
	if loaded.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Updated Title")
	}
}

func TestStoryDelete(t *testing.T) {
	s := newTestStore(t)
	st := makeStory("to-delete", "Delete Me")
	if err := s.CreateStory(st); err != nil {
		t.Fatalf("CreateStory() error = %v", err)
	}

	// Verify it exists
	if _, err := s.LoadStory("to-delete", ""); err != nil {
		t.Fatalf("story should exist before delete: %v", err)
	}

	if err := s.DeleteStory("to-delete", ""); err != nil {
		t.Fatalf("DeleteStory() error = %v", err)
	}

	// File should be gone
	if _, err := os.Stat(s.StoryFile("to-delete", "")); !os.IsNotExist(err) {
		t.Error("expected story file to be removed")
	}

	// Load should fail
	if _, err := s.LoadStory("to-delete", ""); err == nil {
		t.Error("expected error loading deleted story, got nil")
	}
}

func TestStoryLoadNonExistent(t *testing.T) {
	s := newTestStore(t)

	_, err := s.LoadStory("does-not-exist", "")
	if err == nil {
		t.Error("expected error loading non-existent story, got nil")
	}
}

func TestStoryDeleteNonExistent(t *testing.T) {
	s := newTestStore(t)

	err := s.DeleteStory("ghost", "")
	if err == nil {
		t.Error("expected error deleting non-existent story, got nil")
	}
}

func TestStoryCreateDuplicateSlug(t *testing.T) {
	s := newTestStore(t)

	st := makeStory("dup-story", "First")
	if err := s.CreateStory(st); err != nil {
		t.Fatalf("first CreateStory() error = %v", err)
	}

	dup := makeStory("dup-story", "Second")
	if err := s.CreateStory(dup); err == nil {
		t.Error("expected error for duplicate slug, got nil")
	}
}

func TestStorySaveNonExistent(t *testing.T) {
	s := newTestStore(t)

	st := makeStory("no-such-story", "Nope")
	st.ID = "s_fake"
	st.Status = models.StoryStatusDraft
	err := s.SaveStory(st)
	if err == nil {
		t.Error("expected error saving non-existent story, got nil")
	}
}
