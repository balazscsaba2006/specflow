package store

import (
	"fmt"
	"os"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
)

// SaveVerification generates an ID if empty, sets Created if zero, and writes
// the verification as frontmatter+body to executions/{storySlug}/{execID}/verification.md.
func (s *Store) SaveVerification(v *models.Verification, storySlug, execID string) error {
	if v.ID == "" {
		v.ID = models.NewID(models.PrefixVerification)
	}
	if v.Created.IsZero() {
		v.Created = time.Now().UTC().Truncate(time.Second)
	}
	if v.Story == "" {
		v.Story = storySlug
	}
	if v.Execution == "" {
		v.Execution = execID
	}

	dir := s.ExecutionDir(storySlug, execID)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("saving verification: execution directory %q does not exist", dir)
	}

	path := s.VerificationFile(storySlug, execID)
	if err := WriteFile(path, v, v.Body); err != nil {
		return fmt.Errorf("writing verification file: %w", err)
	}

	return nil
}

// LoadVerification reads and parses the verification from
// executions/{storySlug}/{execID}/verification.md.
func (s *Store) LoadVerification(storySlug, execID string) (*models.Verification, error) {
	path := s.VerificationFile(storySlug, execID)

	var v models.Verification
	body, err := ParseFile(path, &v)
	if err != nil {
		return nil, fmt.Errorf("loading verification for %q/%q: %w", storySlug, execID, err)
	}
	v.Body = body

	return &v, nil
}

// LatestVerification finds the latest execution for a story and loads its
// verification. Returns an error if no executions or no verification exists.
func (s *Store) LatestVerification(storySlug string) (*models.Verification, error) {
	latest, err := s.LatestExecution(storySlug)
	if err != nil {
		return nil, fmt.Errorf("finding latest verification: %w", err)
	}

	v, err := s.LoadVerification(storySlug, latest.ID)
	if err != nil {
		return nil, fmt.Errorf("loading latest verification: %w", err)
	}

	return v, nil
}
