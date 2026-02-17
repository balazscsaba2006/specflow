package store

import (
	"fmt"
	"os"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

// SavePlan generates an ID if empty, sets Created if zero, and writes the plan
// as frontmatter+body to executions/{storySlug}/latest/plan.md.
func (s *Store) SavePlan(p *models.Plan, storySlug string) error {
	if p.ID == "" {
		p.ID = models.NewID(models.PrefixPlan)
	}
	if p.Created.IsZero() {
		p.Created = time.Now().UTC().Truncate(time.Second)
	}
	if p.Story == "" {
		p.Story = storySlug
	}

	dir := s.ExecutionDir(storySlug, "latest")
	if err := s.EnsureDir(dir); err != nil {
		return fmt.Errorf("creating plan directory: %w", err)
	}

	path := s.PlanFile(storySlug, "latest")
	if err := WriteFile(path, p, p.Body); err != nil {
		return fmt.Errorf("writing plan file: %w", err)
	}

	return nil
}

// LoadPlan reads and parses the plan from executions/{storySlug}/latest/plan.md.
func (s *Store) LoadPlan(storySlug string) (*models.Plan, error) {
	path := s.PlanFile(storySlug, "latest")

	var p models.Plan
	body, err := ParseFile(path, &p)
	if err != nil {
		return nil, fmt.Errorf("loading plan for story %q: %w", storySlug, err)
	}
	p.Body = body

	return &p, nil
}

// DeletePlan removes the plan file at executions/{storySlug}/latest/plan.md.
func (s *Store) DeletePlan(storySlug string) error {
	path := s.PlanFile(storySlug, "latest")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("deleting plan for story %q: %w", storySlug, err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting plan for story %q: %w", storySlug, err)
	}

	return nil
}
