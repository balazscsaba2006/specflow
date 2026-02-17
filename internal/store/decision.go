package store

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

// CreateDecision validates the slug, generates an ID, sets defaults,
// and writes the decision file under decisions/{slug}.md.
func (s *Store) CreateDecision(d *models.Decision) error {
	if err := models.ValidateSlug(d.Slug); err != nil {
		return fmt.Errorf("creating decision: %w", err)
	}

	// Check for duplicate slug
	path := s.DecisionFile(d.Slug)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("creating decision: slug %q already exists", d.Slug)
	}

	d.ID = models.NewID(models.PrefixDecision)

	if d.Date == "" {
		d.Date = time.Now().UTC().Format("2006-01-02")
	}

	if d.Status == "" {
		d.Status = models.DecisionStatusAccepted
	}

	if err := WriteFile(path, d, d.Body); err != nil {
		return fmt.Errorf("writing decision file: %w", err)
	}

	return nil
}

// LoadDecision reads and parses the decision file for the given slug.
func (s *Store) LoadDecision(slug string) (*models.Decision, error) {
	path := s.DecisionFile(slug)

	var d models.Decision
	body, err := ParseFile(path, &d)
	if err != nil {
		return nil, fmt.Errorf("loading decision %q: %w", slug, err)
	}
	d.Body = body

	return &d, nil
}

// SaveDecision writes the decision file. The decision must already exist on disk.
func (s *Store) SaveDecision(d *models.Decision) error {
	path := s.DecisionFile(d.Slug)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("saving decision %q: file does not exist", d.Slug)
	}

	if err := WriteFile(path, d, d.Body); err != nil {
		return fmt.Errorf("writing decision file: %w", err)
	}

	return nil
}

// ListDecisions reads the decisions directory and loads each decision.
func (s *Store) ListDecisions() ([]*models.Decision, error) {
	dir := s.DecisionsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading decisions directory: %w", err)
	}

	var decisions []*models.Decision
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), ".md")
		d, err := s.LoadDecision(slug)
		if err != nil {
			return nil, fmt.Errorf("listing decisions: %w", err)
		}
		decisions = append(decisions, d)
	}

	return decisions, nil
}

// DeleteDecision removes the decision file for the given slug.
func (s *Store) DeleteDecision(slug string) error {
	path := s.DecisionFile(slug)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("deleting decision %q: %w", slug, err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting decision %q: %w", slug, err)
	}

	return nil
}
