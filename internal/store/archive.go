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

// ArchiveEpic moves an epic and its stories/executions to the archive directory.
// If compact is true, story and epic bodies are stripped (frontmatter-only tombstones).
// If force is false, the epic must be completed and all stories must be done.
func (s *Store) ArchiveEpic(slug string, force, compact bool) (*ArchiveSummary, error) {
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

	if stErr := s.archiveStories(slug, stories, compact); stErr != nil {
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

	if epErr := s.archiveEpicFile(slug, epic, compact); epErr != nil {
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

// archiveStories writes stories to the archive. If compact is true, bodies are stripped.
func (s *Store) archiveStories(epicSlug string, stories []*models.Story, compact bool) error {
	for _, st := range stories {
		body := st.Body
		if compact {
			body = ""
		}
		dst := s.archiveStoryFile(epicSlug, st.Slug)
		if err := WriteFile(dst, st, body); err != nil {
			return fmt.Errorf("archiving story %q: %w", st.Slug, err)
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

// archiveEpicFile writes the epic file to the archive. If compact is true, the body is stripped.
func (s *Store) archiveEpicFile(slug string, epic *models.Epic, compact bool) error {
	body := epic.Body
	if compact {
		body = ""
	}
	epic.Status = models.EpicStatusArchived
	if err := WriteFile(s.ArchiveEpicFile(slug), epic, body); err != nil {
		return fmt.Errorf("archiving epic %q: %w", slug, err)
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

// unarchiveExecutions moves execution directories back from the archive for the given stories.
func (s *Store) unarchiveExecutions(stories []*models.Story) (int, error) {
	var count int
	for _, st := range stories {
		srcExec := s.ArchiveStoryExecutionsDir(st.Slug)
		if _, err := os.Stat(srcExec); err != nil {
			continue
		}
		if err := s.EnsureDir(s.ExecutionsDir()); err != nil {
			return 0, fmt.Errorf("creating executions dir: %w", err)
		}
		dstExec := s.StoryExecutionsDir(st.Slug)
		if err := os.Rename(srcExec, dstExec); err != nil {
			return 0, fmt.Errorf("restoring executions for %q: %w", st.Slug, err)
		}
		count++
	}
	return count, nil
}

// archiveStoryFile returns the archive path for a story file.
func (s *Store) archiveStoryFile(epicSlug, storySlug string) string {
	return fmt.Sprintf("%s/%s.md", s.ArchiveEpicStoriesDir(epicSlug), storySlug)
}

// InitiativeArchiveSummary reports what was moved during an initiative archive.
type InitiativeArchiveSummary struct {
	Slug      string
	Title     string
	EpicCount int // number of linked epics (informational)
}

// ArchiveInitiative moves an initiative to the archive directory.
// If compact is true, the body is stripped (frontmatter-only tombstone).
// If force is false, the initiative must be completed and all linked epics must be archived or completed.
func (s *Store) ArchiveInitiative(slug string, force, compact bool) (*InitiativeArchiveSummary, error) {
	if s.IsInitiativeArchived(slug) {
		return nil, fmt.Errorf("initiative %q is already archived", slug)
	}

	ini, err := s.LoadInitiative(slug)
	if err != nil {
		return nil, fmt.Errorf("archiving initiative: %w", err)
	}

	if !force && ini.Status != models.InitiativeStatusCompleted {
		return nil, fmt.Errorf(
			"initiative %q has status %q (expected %q); use force to override",
			slug, ini.Status, models.InitiativeStatusCompleted,
		)
	}

	if !force {
		for _, epicSlug := range ini.Epics {
			ep, loadErr := s.LoadEpic(epicSlug)
			if loadErr != nil {
				// If we can't load it, check if it's already archived.
				if !s.IsArchived(epicSlug) {
					return nil, fmt.Errorf("linked epic %q not found and not archived; use force to override", epicSlug)
				}
				continue
			}
			if ep.Status != models.EpicStatusCompleted && !s.IsArchived(epicSlug) {
				return nil, fmt.Errorf(
					"linked epic %q has status %q (expected completed or archived); use force to override",
					epicSlug, ep.Status,
				)
			}
		}
	}

	if err := s.EnsureDir(s.ArchiveInitiativeDir(slug)); err != nil {
		return nil, fmt.Errorf("creating archive directory: %w", err)
	}

	// Write to archive (strip body only if compact).
	body := ini.Body
	if compact {
		body = ""
	}
	ini.Status = models.InitiativeStatusArchived
	if err := WriteFile(s.ArchiveInitiativeFile(slug), ini, body); err != nil {
		return nil, fmt.Errorf("archiving initiative %q: %w", slug, err)
	}

	if err := os.RemoveAll(s.InitiativeDir(slug)); err != nil {
		return nil, fmt.Errorf("removing original initiative dir: %w", err)
	}

	return &InitiativeArchiveSummary{
		Slug:      slug,
		Title:     ini.Title,
		EpicCount: len(ini.Epics),
	}, nil
}

// IsInitiativeArchived reports whether an initiative exists in the archive.
func (s *Store) IsInitiativeArchived(slug string) bool {
	_, err := os.Stat(s.ArchiveInitiativeFile(slug))
	return err == nil
}

// LoadArchivedInitiative reads an initiative from the archive directory.
func (s *Store) LoadArchivedInitiative(slug string) (*models.Initiative, error) {
	path := s.ArchiveInitiativeFile(slug)
	var i models.Initiative
	body, err := ParseFile(path, &i)
	if err != nil {
		return nil, fmt.Errorf("loading archived initiative %q: %w", slug, err)
	}
	i.Body = body
	return &i, nil
}

// ListArchivedInitiatives returns all initiatives in the archive directory.
func (s *Store) ListArchivedInitiatives() ([]*models.Initiative, error) {
	dir := s.ArchiveInitiativesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading archived initiatives directory: %w", err)
	}

	var initiatives []*models.Initiative
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		i, loadErr := s.LoadArchivedInitiative(entry.Name())
		if loadErr != nil {
			return nil, fmt.Errorf("listing archived initiatives: %w", loadErr)
		}
		initiatives = append(initiatives, i)
	}
	return initiatives, nil
}

// StoryArchiveSummary reports what was moved during a standalone story archive.
type StoryArchiveSummary struct {
	Slug           string
	Title          string
	ExecutionCount int
}

// ArchiveStory moves a standalone story and its executions to the archive.
// If compact is true, the body is stripped. If force is false, the story must have status done.
func (s *Store) ArchiveStory(slug string, force, compact bool) (*StoryArchiveSummary, error) {
	if s.IsStoryArchived(slug) {
		return nil, fmt.Errorf("story %q is already archived", slug)
	}

	st, err := s.LoadStory(slug, "")
	if err != nil {
		return nil, fmt.Errorf("loading story: %w", err)
	}
	if st.Epic != "" {
		return nil, fmt.Errorf("story %q belongs to epic %q — archive the epic instead", slug, st.Epic)
	}
	if !force && st.Status != models.StoryStatusDone {
		return nil, fmt.Errorf(
			"story %q has status %q (expected %q); use force to override",
			slug, st.Status, models.StoryStatusDone,
		)
	}

	if err := s.EnsureDir(s.ArchiveStoriesDir()); err != nil {
		return nil, fmt.Errorf("creating archive stories dir: %w", err)
	}

	// Write to archive (strip body only if compact).
	body := st.Body
	if compact {
		body = ""
	}
	dst := s.ArchiveStandaloneStoryFile(slug)
	if err := WriteFile(dst, st, body); err != nil {
		return nil, fmt.Errorf("archiving story %q: %w", slug, err)
	}

	// Move executions.
	summary := &StoryArchiveSummary{Slug: slug, Title: st.Title}
	execCount, execErr := s.archiveExecutions([]*models.Story{st})
	if execErr != nil {
		return nil, execErr
	}
	summary.ExecutionCount = execCount

	// Remove original.
	if err := os.Remove(s.StoryFile(slug, "")); err != nil {
		return nil, fmt.Errorf("removing original story file: %w", err)
	}

	return summary, nil
}

// IsStoryArchived reports whether a standalone story exists in the archive.
func (s *Store) IsStoryArchived(slug string) bool {
	_, err := os.Stat(s.ArchiveStandaloneStoryFile(slug))
	return err == nil
}

// LoadArchivedStandaloneStory reads a standalone story from the archive.
func (s *Store) LoadArchivedStandaloneStory(slug string) (*models.Story, error) {
	path := s.ArchiveStandaloneStoryFile(slug)
	var st models.Story
	body, err := ParseFile(path, &st)
	if err != nil {
		return nil, fmt.Errorf("loading archived story %q: %w", slug, err)
	}
	st.Body = body
	return &st, nil
}

// ListArchivedStandaloneStories returns all standalone stories in the archive.
func (s *Store) ListArchivedStandaloneStories() ([]*models.Story, error) {
	dir := s.ArchiveStoriesDir()
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
		st, loadErr := s.LoadArchivedStandaloneStory(slug)
		if loadErr != nil {
			return nil, fmt.Errorf("listing archived standalone stories: %w", loadErr)
		}
		stories = append(stories, st)
	}
	return stories, nil
}

// UnarchiveEpic restores an epic and its stories/executions from the archive.
// The epic is set to on_hold status; stories keep their original status.
func (s *Store) UnarchiveEpic(slug string) (*ArchiveSummary, error) {
	if !s.IsArchived(slug) {
		return nil, fmt.Errorf("epic %q is not in the archive", slug)
	}
	if _, err := os.Stat(s.EpicDir(slug)); err == nil {
		return nil, fmt.Errorf("epic %q already exists in active epics — cannot unarchive", slug)
	}

	epic, err := s.LoadArchivedEpic(slug)
	if err != nil {
		return nil, fmt.Errorf("unarchiving epic: %w", err)
	}
	epic.Status = models.EpicStatusOnHold

	stories, err := s.ListArchivedStories(slug)
	if err != nil {
		return nil, fmt.Errorf("unarchiving epic: listing stories: %w", err)
	}

	for _, d := range []string{s.EpicDir(slug), s.EpicStoriesDir(slug)} {
		if err := s.EnsureDir(d); err != nil {
			return nil, fmt.Errorf("creating epic directory: %w", err)
		}
	}

	if err := WriteFile(s.EpicFile(slug), epic, epic.Body); err != nil {
		return nil, fmt.Errorf("restoring epic %q: %w", slug, err)
	}

	for _, st := range stories {
		dst := s.StoryFile(st.Slug, slug)
		if err := WriteFile(dst, st, st.Body); err != nil {
			return nil, fmt.Errorf("restoring story %q: %w", st.Slug, err)
		}
	}

	// Move docs back if they exist.
	srcDocs := s.ArchiveEpicDocsDir(slug)
	if info, statErr := os.Stat(srcDocs); statErr == nil && info.IsDir() {
		if err := os.Rename(srcDocs, s.EpicDocsDir(slug)); err != nil {
			return nil, fmt.Errorf("restoring docs for epic %q: %w", slug, err)
		}
	}

	// Move executions back.
	execCount, execErr := s.unarchiveExecutions(stories)
	if execErr != nil {
		return nil, execErr
	}

	if err := os.RemoveAll(s.ArchiveEpicDir(slug)); err != nil {
		return nil, fmt.Errorf("removing archive epic dir: %w", err)
	}

	return &ArchiveSummary{
		EpicSlug:       slug,
		EpicTitle:      epic.Title,
		StoryCount:     len(stories),
		ExecutionCount: execCount,
	}, nil
}

// UnarchiveStory restores a standalone story from the archive.
// The story status is set to planned.
func (s *Store) UnarchiveStory(slug string) (*StoryArchiveSummary, error) {
	if !s.IsStoryArchived(slug) {
		return nil, fmt.Errorf("story %q is not in the archive", slug)
	}
	if _, err := os.Stat(s.StoryFile(slug, "")); err == nil {
		return nil, fmt.Errorf("story %q already exists in active stories — cannot unarchive", slug)
	}

	st, err := s.LoadArchivedStandaloneStory(slug)
	if err != nil {
		return nil, fmt.Errorf("unarchiving story: %w", err)
	}
	st.Status = models.StoryStatusPlanned

	if err := s.EnsureDir(s.StandaloneStoriesDir()); err != nil {
		return nil, fmt.Errorf("creating stories dir: %w", err)
	}

	if err := WriteFile(s.StoryFile(slug, ""), st, st.Body); err != nil {
		return nil, fmt.Errorf("restoring story %q: %w", slug, err)
	}

	execCount, execErr := s.unarchiveExecutions([]*models.Story{st})
	if execErr != nil {
		return nil, execErr
	}

	if err := os.Remove(s.ArchiveStandaloneStoryFile(slug)); err != nil {
		return nil, fmt.Errorf("removing archived story file: %w", err)
	}

	return &StoryArchiveSummary{
		Slug:           slug,
		Title:          st.Title,
		ExecutionCount: execCount,
	}, nil
}

// UnarchiveInitiative restores an initiative from the archive.
// The initiative status is set to on_hold.
func (s *Store) UnarchiveInitiative(slug string) (*InitiativeArchiveSummary, error) {
	if !s.IsInitiativeArchived(slug) {
		return nil, fmt.Errorf("initiative %q is not in the archive", slug)
	}
	if _, err := os.Stat(s.InitiativeDir(slug)); err == nil {
		return nil, fmt.Errorf("initiative %q already exists in active initiatives — cannot unarchive", slug)
	}

	ini, err := s.LoadArchivedInitiative(slug)
	if err != nil {
		return nil, fmt.Errorf("unarchiving initiative: %w", err)
	}
	ini.Status = models.InitiativeStatusOnHold

	if err := s.EnsureDir(s.InitiativeDir(slug)); err != nil {
		return nil, fmt.Errorf("creating initiative dir: %w", err)
	}

	if err := WriteFile(s.InitiativeFile(slug), ini, ini.Body); err != nil {
		return nil, fmt.Errorf("restoring initiative %q: %w", slug, err)
	}

	if err := os.RemoveAll(s.ArchiveInitiativeDir(slug)); err != nil {
		return nil, fmt.Errorf("removing archive initiative dir: %w", err)
	}

	return &InitiativeArchiveSummary{
		Slug:      slug,
		Title:     ini.Title,
		EpicCount: len(ini.Epics),
	}, nil
}
