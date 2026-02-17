package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

// CreateDoc validates the slug, generates an ID, sets timestamps and defaults,
// then writes the doc file. If d.Epic is set, the file goes under
// epics/{epic}/docs/{slug}.md; otherwise under docs/{slug}.md.
func (s *Store) CreateDoc(d *models.Document) error {
	if err := models.ValidateSlug(d.Slug); err != nil {
		return fmt.Errorf("creating doc: %w", err)
	}

	// Check for duplicate slug at the target location
	path := s.DocFile(d.Slug, d.Epic)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("creating doc: slug %q already exists", d.Slug)
	}

	d.ID = models.NewID(models.PrefixDoc)
	now := time.Now().UTC().Truncate(time.Second)
	d.Created = now
	d.Updated = now

	if d.Status == "" {
		d.Status = models.DocStatusDraft
	}

	// Ensure the parent directory exists
	dir := filepath.Dir(path)
	if err := s.EnsureDir(dir); err != nil {
		return fmt.Errorf("creating doc directory: %w", err)
	}

	if err := WriteFile(path, d, d.Body); err != nil {
		return fmt.Errorf("writing doc file: %w", err)
	}

	return nil
}

// LoadDoc reads and parses the doc file for the given slug.
// If epicSlug is non-empty, it looks under that epic's docs/ dir.
func (s *Store) LoadDoc(slug, epicSlug string) (*models.Document, error) {
	path := s.DocFile(slug, epicSlug)

	var d models.Document
	body, err := ParseFile(path, &d)
	if err != nil {
		return nil, fmt.Errorf("loading doc %q: %w", slug, err)
	}
	d.Body = body

	return &d, nil
}

// SaveDoc updates the Updated timestamp and writes the doc file.
// The doc must already exist on disk.
func (s *Store) SaveDoc(d *models.Document) error {
	path := s.DocFile(d.Slug, d.Epic)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("saving doc %q: file does not exist", d.Slug)
	}

	d.Updated = time.Now().UTC().Truncate(time.Second)

	if err := WriteFile(path, d, d.Body); err != nil {
		return fmt.Errorf("writing doc file: %w", err)
	}

	return nil
}

// ListDocs lists docs in the given scope. If epicSlug is provided,
// lists docs under that epic's docs/ dir; otherwise lists project-level docs.
func (s *Store) ListDocs(epicSlug string) ([]*models.Document, error) {
	var dir string
	if epicSlug != "" {
		dir = s.EpicDocsDir(epicSlug)
	} else {
		dir = s.ProjectDocsDir()
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading docs directory %s: %w", dir, err)
	}

	var docs []*models.Document
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), ".md")
		d, err := s.LoadDoc(slug, epicSlug)
		if err != nil {
			return nil, fmt.Errorf("listing docs: %w", err)
		}
		docs = append(docs, d)
	}

	return docs, nil
}

// DeleteDoc removes the doc file for the given slug.
func (s *Store) DeleteDoc(slug, epicSlug string) error {
	path := s.DocFile(slug, epicSlug)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("deleting doc %q: %w", slug, err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting doc %q: %w", slug, err)
	}

	return nil
}
