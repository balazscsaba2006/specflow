package store

import (
	"fmt"
	"os"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
)

// ArchiveSummary reports what was moved during an archive operation.
type ArchiveSummary struct {
	EpicSlug       string
	EpicTitle      string
	StoryCount     int
	ExecutionCount int
}

// ArchiveEpic moves an epic and its stories/executions to the archive directory,
// compacting story and epic files to frontmatter-only tombstones.
// If force is false, the epic must be completed and all stories must be done.
func (s *Store) ArchiveEpic(slug string, force bool) (*ArchiveSummary, error) {
	if s.IsArchived(slug) {
		return nil, fmt.Errorf("epic %q is already archived", slug)
	}

	epic, stories, valErr := s.validateArchive(slug, force)
	if valErr != nil {
		return nil, valErr
	}

	if prepErr := s.prepareArchiveDirs(slug); prepErr != nil {
		return nil, prepErr
	}

	summary := &ArchiveSummary{
		EpicSlug:   slug,
		EpicTitle:  epic.Title,
		StoryCount: len(stories),
	}

	if stErr := s.archiveStories(slug, stories); stErr != nil {
		return nil, stErr
	}

	execCount, execErr := s.archiveExecutions(stories)
	if execErr != nil {
		return nil, execErr
	}
	summary.ExecutionCount = execCount

	if docErr := s.archiveDocs(slug); docErr != nil {
		return nil, docErr
	}

	if epErr := s.archiveEpicFile(slug, epic); epErr != nil {
		return nil, epErr
	}

	if rmErr := os.RemoveAll(s.EpicDir(slug)); rmErr != nil {
		return nil, fmt.Errorf("removing original epic dir: %w", rmErr)
	}

	return summary, nil
}

// validateArchive loads and validates the epic and stories for archiving.
func (s *Store) validateArchive(slug string, force bool) (*models.Epic, []*models.Story, error) {
	epic, err := s.LoadEpic(slug)
	if err != nil {
		return nil, nil, fmt.Errorf("archiving epic: %w", err)
	}

	if !force && epic.Status != models.EpicStatusCompleted {
		return nil, nil, fmt.Errorf(
			"epic %q has status %q (expected %q); use force to override",
			slug, epic.Status, models.EpicStatusCompleted,
		)
	}

	stories, err := s.ListStories(slug)
	if err != nil {
		return nil, nil, fmt.Errorf("archiving epic: listing stories: %w", err)
	}

	if !force {
		for _, st := range stories {
			if st.Status != models.StoryStatusDone {
				return nil, nil, fmt.Errorf(
					"story %q has status %q (expected %q); use force to override",
					st.Slug, st.Status, models.StoryStatusDone,
				)
			}
		}
	}

	return epic, stories, nil
}

// prepareArchiveDirs creates the archive directory structure for an epic.
func (s *Store) prepareArchiveDirs(slug string) error {
	for _, d := range []string{s.ArchiveEpicDir(slug), s.ArchiveEpicStoriesDir(slug)} {
		if err := s.EnsureDir(d); err != nil {
			return fmt.Errorf("creating archive directory: %w", err)
		}
	}
	return nil
}

// archiveStories compacts and writes stories to the archive.
func (s *Store) archiveStories(epicSlug string, stories []*models.Story) error {
	for _, st := range stories {
		st.Body = ""
		dst := s.archiveStoryFile(epicSlug, st.Slug)
		if err := WriteFile(dst, st, ""); err != nil {
			return fmt.Errorf("compacting story %q: %w", st.Slug, err)
		}
	}
	return nil
}

// archiveExecutions moves execution directories to the archive.
func (s *Store) archiveExecutions(stories []*models.Story) (int, error) {
	var count int
	for _, st := range stories {
		srcExec := s.StoryExecutionsDir(st.Slug)
		if _, err := os.Stat(srcExec); err != nil {
			continue
		}
		if err := s.EnsureDir(s.ArchiveExecutionsDir()); err != nil {
			return 0, fmt.Errorf("creating archive executions dir: %w", err)
		}
		dstExec := s.ArchiveStoryExecutionsDir(st.Slug)
		if err := os.Rename(srcExec, dstExec); err != nil {
			return 0, fmt.Errorf("moving executions for %q: %w", st.Slug, err)
		}
		count++
	}
	return count, nil
}

// archiveDocs moves the epic's docs directory as-is (no compaction).
func (s *Store) archiveDocs(slug string) error {
	srcDocs := s.EpicDocsDir(slug)
	info, err := os.Stat(srcDocs)
	if err != nil || !info.IsDir() {
		return nil
	}
	dstDocs := s.ArchiveEpicDocsDir(slug)
	if err := os.Rename(srcDocs, dstDocs); err != nil {
		return fmt.Errorf("moving docs for epic %q: %w", slug, err)
	}
	return nil
}

// archiveEpicFile compacts and writes the epic file to the archive.
func (s *Store) archiveEpicFile(slug string, epic *models.Epic) error {
	epic.Body = ""
	epic.Status = models.EpicStatusArchived
	if err := WriteFile(s.ArchiveEpicFile(slug), epic, ""); err != nil {
		return fmt.Errorf("compacting epic %q: %w", slug, err)
	}
	return nil
}

// IsArchived reports whether an epic exists in the archive.
func (s *Store) IsArchived(slug string) bool {
	_, err := os.Stat(s.ArchiveEpicFile(slug))
	return err == nil
}

// LoadArchivedEpic reads an epic from the archive directory.
func (s *Store) LoadArchivedEpic(slug string) (*models.Epic, error) {
	path := s.ArchiveEpicFile(slug)
	var e models.Epic
	body, err := ParseFile(path, &e)
	if err != nil {
		return nil, fmt.Errorf("loading archived epic %q: %w", slug, err)
	}
	e.Body = body
	return &e, nil
}

// LoadArchivedStory reads a story from the archive directory.
func (s *Store) LoadArchivedStory(slug, epicSlug string) (*models.Story, error) {
	path := s.archiveStoryFile(epicSlug, slug)
	var st models.Story
	body, err := ParseFile(path, &st)
	if err != nil {
		return nil, fmt.Errorf("loading archived story %q: %w", slug, err)
	}
	st.Body = body
	return &st, nil
}

// ListArchivedEpics returns all epics in the archive directory.
func (s *Store) ListArchivedEpics() ([]*models.Epic, error) {
	dir := s.ArchiveEpicsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading archived epics directory: %w", err)
	}

	var epics []*models.Epic
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		e, loadErr := s.LoadArchivedEpic(entry.Name())
		if loadErr != nil {
			return nil, fmt.Errorf("listing archived epics: %w", loadErr)
		}
		epics = append(epics, e)
	}
	return epics, nil
}

// ListArchivedStories returns all stories for an archived epic.
func (s *Store) ListArchivedStories(epicSlug string) ([]*models.Story, error) {
	dir := s.ArchiveEpicStoriesDir(epicSlug)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading archived stories directory: %w", err)
	}

	var stories []*models.Story
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), ".md")
		st, loadErr := s.LoadArchivedStory(slug, epicSlug)
		if loadErr != nil {
			return nil, fmt.Errorf("listing archived stories: %w", loadErr)
		}
		stories = append(stories, st)
	}
	return stories, nil
}

// archiveStoryFile returns the archive path for a story file.
func (s *Store) archiveStoryFile(epicSlug, storySlug string) string {
	return fmt.Sprintf("%s/%s.md", s.ArchiveEpicStoriesDir(epicSlug), storySlug)
}
