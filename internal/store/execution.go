package store

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/balazscsaba2006/specflow/internal/models"
	"gopkg.in/yaml.v3"
)

// CreateExecution generates an ID, sets StartedAt and status, creates the
// execution directory, and writes meta.yaml.
func (s *Store) CreateExecution(e *models.Execution) error {
	e.ID = models.NewID(models.PrefixExecution)
	e.StartedAt = time.Now().UTC().Truncate(time.Second)
	e.Status = models.ExecutionStatusStarted

	dir := s.ExecutionDir(e.Story, e.ID)
	if err := s.EnsureDir(dir); err != nil {
		return fmt.Errorf("creating execution directory: %w", err)
	}

	if err := s.writeExecutionMeta(e); err != nil {
		return fmt.Errorf("creating execution: %w", err)
	}

	return nil
}

// LoadExecution reads and unmarshals meta.yaml for a specific execution.
func (s *Store) LoadExecution(storySlug, execID string) (*models.Execution, error) {
	path := s.ExecutionMetaFile(storySlug, execID)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading execution %q/%q: %w", storySlug, execID, err)
	}

	var e models.Execution
	if err := yaml.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("parsing execution %q/%q: %w", storySlug, execID, err)
	}

	return &e, nil
}

// SaveExecution writes the execution's meta.yaml at its known path.
func (s *Store) SaveExecution(e *models.Execution) error {
	path := s.ExecutionMetaFile(e.Story, e.ID)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("saving execution %q/%q: file does not exist", e.Story, e.ID)
	}

	if err := s.writeExecutionMeta(e); err != nil {
		return fmt.Errorf("saving execution: %w", err)
	}

	return nil
}

// ListExecutions reads execution dirs for a story and loads each meta.yaml.
func (s *Store) ListExecutions(storySlug string) ([]*models.Execution, error) {
	dir := s.StoryExecutionsDir(storySlug)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading executions directory for %q: %w", storySlug, err)
	}

	var executions []*models.Execution
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "latest" {
			continue // latest/ is a mutable workspace for plans/verifications, not an execution record
		}
		e, err := s.LoadExecution(storySlug, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("listing executions: %w", err)
		}
		executions = append(executions, e)
	}

	return executions, nil
}

// LatestExecution lists executions for a story and returns the most recent one
// by StartedAt. Returns an error if no executions exist.
func (s *Store) LatestExecution(storySlug string) (*models.Execution, error) {
	execs, err := s.ListExecutions(storySlug)
	if err != nil {
		return nil, fmt.Errorf("finding latest execution: %w", err)
	}
	if len(execs) == 0 {
		return nil, fmt.Errorf("no executions found for story %q", storySlug)
	}

	sort.Slice(execs, func(i, j int) bool {
		return execs[i].StartedAt.After(execs[j].StartedAt)
	})

	return execs[0], nil
}

// SaveHandover writes handover notes as raw markdown to the execution directory.
func (s *Store) SaveHandover(notes, storySlug, execID string) error {
	path := s.HandoverFile(storySlug, execID)
	if err := os.WriteFile(path, []byte(notes), 0o600); err != nil {
		return fmt.Errorf("writing handover notes: %w", err)
	}
	return nil
}

// LoadHandover reads handover notes from the execution directory.
func (s *Store) LoadHandover(storySlug, execID string) (string, error) {
	path := s.HandoverFile(storySlug, execID)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading handover notes: %w", err)
	}
	return string(data), nil
}

// writeExecutionMeta marshals an Execution to YAML and writes it to meta.yaml.
func (s *Store) writeExecutionMeta(e *models.Execution) error {
	data, err := yaml.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshaling execution meta: %w", err)
	}

	path := s.ExecutionMetaFile(e.Story, e.ID)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing execution meta: %w", err)
	}

	return nil
}
