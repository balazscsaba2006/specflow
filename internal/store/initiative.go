package store

import (
	"fmt"
	"os"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

// CreateInitiative validates the slug, generates an ID, sets timestamps,
// creates the initiative directory, and writes the frontmatter file.
func (s *Store) CreateInitiative(i *models.Initiative) error {
	if err := models.ValidateSlug(i.Slug); err != nil {
		return fmt.Errorf("creating initiative: %w", err)
	}

	// Check for duplicate slug
	dir := s.InitiativeDir(i.Slug)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("creating initiative: slug %q already exists", i.Slug)
	}

	i.ID = models.NewID(models.PrefixInitiative)
	now := time.Now().UTC().Truncate(time.Second)
	i.Created = now
	i.Updated = now

	if err := s.EnsureDir(dir); err != nil {
		return fmt.Errorf("creating initiative directory: %w", err)
	}

	path := s.InitiativeFile(i.Slug)
	if err := WriteFile(path, i, i.Body); err != nil {
		return fmt.Errorf("writing initiative file: %w", err)
	}

	return nil
}

// LoadInitiative reads and parses the initiative.md file for the given slug.
func (s *Store) LoadInitiative(slug string) (*models.Initiative, error) {
	path := s.InitiativeFile(slug)

	var i models.Initiative
	body, err := ParseFile(path, &i)
	if err != nil {
		return nil, fmt.Errorf("loading initiative %q: %w", slug, err)
	}
	i.Body = body

	return &i, nil
}

// SaveInitiative updates the Updated timestamp and writes the initiative file.
// The initiative must already exist on disk.
func (s *Store) SaveInitiative(i *models.Initiative) error {
	path := s.InitiativeFile(i.Slug)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("saving initiative %q: file does not exist", i.Slug)
	}

	i.Updated = time.Now().UTC().Truncate(time.Second)

	if err := WriteFile(path, i, i.Body); err != nil {
		return fmt.Errorf("writing initiative file: %w", err)
	}

	return nil
}

// ListInitiatives reads the initiatives directory and loads each initiative.
func (s *Store) ListInitiatives() ([]*models.Initiative, error) {
	dir := s.InitiativesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading initiatives directory: %w", err)
	}

	var initiatives []*models.Initiative
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		i, err := s.LoadInitiative(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("listing initiatives: %w", err)
		}
		initiatives = append(initiatives, i)
	}

	return initiatives, nil
}

// DeleteInitiative removes the initiative directory for the given slug.
func (s *Store) DeleteInitiative(slug string) error {
	dir := s.InitiativeDir(slug)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("deleting initiative %q: %w", slug, err)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("deleting initiative %q: %w", slug, err)
	}

	return nil
}
