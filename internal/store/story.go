package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

// CreateStory validates the slug, generates an ID, sets timestamps and defaults,
// then writes the story file. If s.Epic is set, the file goes under
// epics/{epic}/stories/{slug}.md; otherwise under stories/{slug}.md.
func (s *Store) CreateStory(st *models.Story) error {
	if err := models.ValidateSlug(st.Slug); err != nil {
		return fmt.Errorf("creating story: %w", err)
	}

	// Check for duplicate slug at the target location
	path := s.StoryFile(st.Slug, st.Epic)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("creating story: slug %q already exists", st.Slug)
	}

	st.ID = models.NewID(models.PrefixStory)
	now := time.Now().UTC().Truncate(time.Second)
	st.Created = now
	st.Updated = now

	if st.Status == "" {
		st.Status = models.StoryStatusDraft
	}
	if st.Priority == "" {
		st.Priority = models.PriorityMedium
	}

	// Ensure the parent directory exists
	dir := filepath.Dir(path)
	if err := s.EnsureDir(dir); err != nil {
		return fmt.Errorf("creating story directory: %w", err)
	}

	if err := WriteFile(path, st, st.Body); err != nil {
		return fmt.Errorf("writing story file: %w", err)
	}

	return nil
}

// LoadStory reads and parses the story file for the given slug.
// If epicSlug is non-empty, it looks under that epic's stories/ dir.
func (s *Store) LoadStory(slug, epicSlug string) (*models.Story, error) {
	path := s.StoryFile(slug, epicSlug)

	var st models.Story
	body, err := ParseFile(path, &st)
	if err != nil {
		return nil, fmt.Errorf("loading story %q: %w", slug, err)
	}
	st.Body = body

	return &st, nil
}

// SaveStory validates any status transition, updates the Updated timestamp,
// and writes the story file. The story must already exist on disk.
func (s *Store) SaveStory(st *models.Story) error {
	path := s.StoryFile(st.Slug, st.Epic)

	var old models.Story
	if _, err := ParseFile(path, &old); err != nil {
		return fmt.Errorf("saving story %q: file does not exist", st.Slug)
	}

	if old.Status != st.Status {
		if err := models.ValidateTransition(old.Status, st.Status); err != nil {
			return fmt.Errorf("saving story %q: %w", st.Slug, err)
		}
	}

	st.Updated = time.Now().UTC().Truncate(time.Second)

	if err := WriteFile(path, st, st.Body); err != nil {
		return fmt.Errorf("writing story file: %w", err)
	}

	return nil
}

// ListStories lists stories in the given scope. If epicSlug is provided,
// lists stories under that epic's stories/ dir; otherwise lists standalone stories.
func (s *Store) ListStories(epicSlug string) ([]*models.Story, error) {
	var dir string
	if epicSlug != "" {
		dir = s.EpicStoriesDir(epicSlug)
	} else {
		dir = s.StandaloneStoriesDir()
	}

	return s.loadStoriesFromDir(dir, epicSlug)
}

// ListAllStories returns all stories across all epics plus standalone stories.
func (s *Store) ListAllStories() ([]*models.Story, error) {
	var all []*models.Story

	// Standalone stories
	standalone, err := s.ListStories("")
	if err != nil {
		return nil, fmt.Errorf("listing all stories: %w", err)
	}
	all = append(all, standalone...)

	// Epic-scoped stories
	epicsDir := s.EpicsDir()
	entries, err := os.ReadDir(epicsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return all, nil
		}
		return nil, fmt.Errorf("listing all stories: reading epics directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		epicSlug := entry.Name()
		stories, err := s.ListStories(epicSlug)
		if err != nil {
			return nil, fmt.Errorf("listing all stories: %w", err)
		}
		all = append(all, stories...)
	}

	return all, nil
}

// DeleteStory removes the story file for the given slug.
func (s *Store) DeleteStory(slug, epicSlug string) error {
	path := s.StoryFile(slug, epicSlug)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("deleting story %q: %w", slug, err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting story %q: %w", slug, err)
	}

	return nil
}

// UpdateStoryStatus is a convenience method that loads a story, validates
// the transition, updates the status, and saves it.
func (s *Store) UpdateStoryStatus(slug, epicSlug, newStatus string) error {
	st, err := s.LoadStory(slug, epicSlug)
	if err != nil {
		return fmt.Errorf("updating story status: %w", err)
	}

	st.Status = newStatus

	if err := s.SaveStory(st); err != nil {
		return fmt.Errorf("updating story status: %w", err)
	}

	return nil
}

// loadStoriesFromDir reads all .md files in a directory and parses them as stories.
func (s *Store) loadStoriesFromDir(dir, epicSlug string) ([]*models.Story, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading stories directory %s: %w", dir, err)
	}

	var stories []*models.Story
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), ".md")
		st, err := s.LoadStory(slug, epicSlug)
		if err != nil {
			return nil, fmt.Errorf("listing stories: %w", err)
		}
		stories = append(stories, st)
	}

	return stories, nil
}
