package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// Store provides filesystem CRUD for all specflow entities.
type Store struct {
	root string // Path to .specflow/ directory
}

// New creates a Store rooted at the given .specflow/ directory.
func New(root string) *Store {
	return &Store{root: root}
}

// Root returns the .specflow/ directory path.
func (s *Store) Root() string {
	return s.root
}

// ProjectRoot returns the parent of .specflow/ (the project root).
func (s *Store) ProjectRoot() string {
	return filepath.Dir(s.root)
}

// EnsureDir creates a directory and all parents if they don't exist.
func (s *Store) EnsureDir(path string) error {
	return os.MkdirAll(path, 0o750)
}

// Path helpers for each entity type

func (s *Store) InitiativesDir() string {
	return filepath.Join(s.root, "initiatives")
}

func (s *Store) InitiativeDir(slug string) string {
	return filepath.Join(s.InitiativesDir(), slug)
}

func (s *Store) InitiativeFile(slug string) string {
	return filepath.Join(s.InitiativeDir(slug), "initiative.md")
}

func (s *Store) EpicsDir() string {
	return filepath.Join(s.root, "epics")
}

func (s *Store) EpicDir(slug string) string {
	return filepath.Join(s.EpicsDir(), slug)
}

func (s *Store) EpicFile(slug string) string {
	return filepath.Join(s.EpicDir(slug), "epic.md")
}

func (s *Store) EpicDocsDir(epicSlug string) string {
	return filepath.Join(s.EpicDir(epicSlug), "docs")
}

func (s *Store) EpicStoriesDir(epicSlug string) string {
	return filepath.Join(s.EpicDir(epicSlug), "stories")
}

func (s *Store) StandaloneStoriesDir() string {
	return filepath.Join(s.root, "stories")
}

func (s *Store) StoryFile(slug, epicSlug string) string {
	if epicSlug != "" {
		return filepath.Join(s.EpicStoriesDir(epicSlug), slug+".md")
	}
	return filepath.Join(s.StandaloneStoriesDir(), slug+".md")
}

func (s *Store) ProjectDocsDir() string {
	return filepath.Join(s.root, "docs")
}

func (s *Store) DocFile(slug, epicSlug string) string {
	if epicSlug != "" {
		return filepath.Join(s.EpicDocsDir(epicSlug), slug+".md")
	}
	return filepath.Join(s.ProjectDocsDir(), slug+".md")
}

func (s *Store) DecisionsDir() string {
	return filepath.Join(s.root, "decisions")
}

func (s *Store) DecisionFile(slug string) string {
	return filepath.Join(s.DecisionsDir(), slug+".md")
}

func (s *Store) ExecutionsDir() string {
	return filepath.Join(s.root, "executions")
}

func (s *Store) StoryExecutionsDir(storySlug string) string {
	return filepath.Join(s.ExecutionsDir(), storySlug)
}

func (s *Store) ExecutionDir(storySlug, execID string) string {
	return filepath.Join(s.StoryExecutionsDir(storySlug), execID)
}

func (s *Store) PlanFile(storySlug, execID string) string {
	return filepath.Join(s.ExecutionDir(storySlug, execID), "plan.md")
}

func (s *Store) VerificationFile(storySlug, execID string) string {
	return filepath.Join(s.ExecutionDir(storySlug, execID), "verification.md")
}

func (s *Store) ExecutionMetaFile(storySlug, execID string) string {
	return filepath.Join(s.ExecutionDir(storySlug, execID), "meta.yaml")
}

func (s *Store) HandoverFile(storySlug, execID string) string {
	return filepath.Join(s.ExecutionDir(storySlug, execID), "handover.md")
}

func (s *Store) LogFile() string {
	return filepath.Join(s.root, "log.jsonl")
}

func (s *Store) ConfigFile() string {
	return filepath.Join(s.root, "config.yaml")
}

func (s *Store) TemplatesDir() string {
	return filepath.Join(s.root, "templates")
}

// Archive path helpers

func (s *Store) ArchiveDir() string {
	return filepath.Join(s.root, "archive")
}

func (s *Store) ArchiveEpicsDir() string {
	return filepath.Join(s.ArchiveDir(), "epics")
}

func (s *Store) ArchiveEpicDir(slug string) string {
	return filepath.Join(s.ArchiveEpicsDir(), slug)
}

func (s *Store) ArchiveEpicFile(slug string) string {
	return filepath.Join(s.ArchiveEpicDir(slug), "epic.md")
}

func (s *Store) ArchiveEpicStoriesDir(slug string) string {
	return filepath.Join(s.ArchiveEpicDir(slug), "stories")
}

func (s *Store) ArchiveEpicDocsDir(slug string) string {
	return filepath.Join(s.ArchiveEpicDir(slug), "docs")
}

func (s *Store) ArchiveStoriesDir() string {
	return filepath.Join(s.ArchiveDir(), "stories")
}

func (s *Store) ArchiveStandaloneStoryFile(slug string) string {
	return filepath.Join(s.ArchiveStoriesDir(), slug+".md")
}

func (s *Store) ArchiveExecutionsDir() string {
	return filepath.Join(s.ArchiveDir(), "executions")
}

func (s *Store) ArchiveStoryExecutionsDir(storySlug string) string {
	return filepath.Join(s.ArchiveExecutionsDir(), storySlug)
}

func (s *Store) ArchiveInitiativesDir() string {
	return filepath.Join(s.ArchiveDir(), "initiatives")
}

func (s *Store) ArchiveInitiativeDir(slug string) string {
	return filepath.Join(s.ArchiveInitiativesDir(), slug)
}

func (s *Store) ArchiveInitiativeFile(slug string) string {
	return filepath.Join(s.ArchiveInitiativeDir(slug), "initiative.md")
}

// Init creates the base .specflow/ directory structure.
func (s *Store) Init() error {
	dirs := []string{
		s.root,
		s.InitiativesDir(),
		s.EpicsDir(),
		s.StandaloneStoriesDir(),
		s.ProjectDocsDir(),
		s.DecisionsDir(),
		s.ExecutionsDir(),
		s.TemplatesDir(),
	}
	for _, dir := range dirs {
		if err := s.EnsureDir(dir); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}
	return nil
}
