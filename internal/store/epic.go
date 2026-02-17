package store

import (
	"fmt"
	"os"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

// CreateEpic validates the slug, generates an ID, sets timestamps,
// creates the epic directory with docs/ and stories/ subdirs, and writes epic.md.
func (s *Store) CreateEpic(e *models.Epic) error {
	if err := models.ValidateSlug(e.Slug); err != nil {
		return fmt.Errorf("creating epic: %w", err)
	}

	// Check for duplicate slug
	dir := s.EpicDir(e.Slug)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("creating epic: slug %q already exists", e.Slug)
	}

	e.ID = models.NewID(models.PrefixEpic)
	now := time.Now().UTC().Truncate(time.Second)
	e.Created = now
	e.Updated = now

	// Create epic directory and subdirectories
	for _, d := range []string{dir, s.EpicDocsDir(e.Slug), s.EpicStoriesDir(e.Slug)} {
		if err := s.EnsureDir(d); err != nil {
			return fmt.Errorf("creating epic directory: %w", err)
		}
	}

	path := s.EpicFile(e.Slug)
	if err := WriteFile(path, e, e.Body); err != nil {
		return fmt.Errorf("writing epic file: %w", err)
	}

	return nil
}

// LoadEpic reads and parses the epic.md file for the given slug.
func (s *Store) LoadEpic(slug string) (*models.Epic, error) {
	path := s.EpicFile(slug)

	var e models.Epic
	body, err := ParseFile(path, &e)
	if err != nil {
		return nil, fmt.Errorf("loading epic %q: %w", slug, err)
	}
	e.Body = body

	return &e, nil
}

// SaveEpic updates the Updated timestamp and writes the epic file.
// The epic must already exist on disk.
func (s *Store) SaveEpic(e *models.Epic) error {
	path := s.EpicFile(e.Slug)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("saving epic %q: file does not exist", e.Slug)
	}

	e.Updated = time.Now().UTC().Truncate(time.Second)

	if err := WriteFile(path, e, e.Body); err != nil {
		return fmt.Errorf("writing epic file: %w", err)
	}

	return nil
}

// ListEpics reads the epics directory and loads each epic.
func (s *Store) ListEpics() ([]*models.Epic, error) {
	dir := s.EpicsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading epics directory: %w", err)
	}

	var epics []*models.Epic
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		e, err := s.LoadEpic(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("listing epics: %w", err)
		}
		epics = append(epics, e)
	}

	return epics, nil
}

// DeleteEpic removes the epic directory for the given slug.
func (s *Store) DeleteEpic(slug string) error {
	dir := s.EpicDir(slug)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("deleting epic %q: %w", slug, err)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("deleting epic %q: %w", slug, err)
	}

	return nil
}
