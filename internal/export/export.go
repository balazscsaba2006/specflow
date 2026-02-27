package export

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
)

// ExportOptions controls what data is included in the export.
type ExportOptions struct {
	IncludeDone bool
	IncludeBody bool
}

// ExportData holds the export-ready representation of an epic and its stories.
type ExportData struct {
	Epic    EpicExport
	Stories []StoryExport
}

// EpicExport is the export-ready representation of an epic.
type EpicExport struct {
	Slug     string
	Title    string
	Status   string
	Fidelity string
	Body     string
	Phases   []models.Phase
}

// StoryExport is the export-ready representation of a story.
type StoryExport struct {
	Slug        string
	Title       string
	Status      string
	Priority    string
	Labels      []string
	Acceptance  []string
	Description string
}

// ExportEpic loads an epic and its stories, returning an export-ready structure.
// Stories are ordered by phase order; stories not in any phase appear at the end.
func ExportEpic(s *store.Store, epicSlug string, opts ExportOptions) (*ExportData, error) {
	epic, err := s.LoadEpic(epicSlug)
	if err != nil {
		return nil, fmt.Errorf("loading epic %q: %w", epicSlug, err)
	}

	stories, err := s.ListStories(epicSlug)
	if err != nil {
		return nil, fmt.Errorf("listing stories for epic %q: %w", epicSlug, err)
	}

	// Filter out done stories if requested.
	if !opts.IncludeDone {
		filtered := make([]*models.Story, 0, len(stories))
		for _, st := range stories {
			if st.Status != models.StoryStatusDone {
				filtered = append(filtered, st)
			}
		}
		stories = filtered
	}

	// Build slug→position map from phase order.
	slugPos := buildPhasePositionMap(epic.Phases)

	// Sort stories by phase position, preserving load order for ties.
	sort.SliceStable(stories, func(i, j int) bool {
		return phasePosition(slugPos, stories[i].Slug) < phasePosition(slugPos, stories[j].Slug)
	})

	// Build story exports.
	storyExports := make([]StoryExport, 0, len(stories))
	for _, st := range stories {
		storyExports = append(storyExports, StoryExport{
			Slug:        st.Slug,
			Title:       st.Title,
			Status:      st.Status,
			Priority:    st.Priority,
			Labels:      st.Labels,
			Acceptance:  st.Acceptance,
			Description: assembleDescription(st.Body, st.Acceptance, opts.IncludeBody),
		})
	}

	// Build epic export.
	body := ""
	if opts.IncludeBody {
		body = epic.Body
	}

	return &ExportData{
		Epic: EpicExport{
			Slug:     epic.Slug,
			Title:    epic.Title,
			Status:   epic.Status,
			Fidelity: epic.Fidelity,
			Body:     body,
			Phases:   epic.Phases,
		},
		Stories: storyExports,
	}, nil
}

// buildPhasePositionMap assigns a sequential position to each story slug based on phase order.
func buildPhasePositionMap(phases []models.Phase) map[string]int {
	pos := make(map[string]int)
	idx := 0
	for _, phase := range phases {
		for _, slug := range phase.Stories {
			pos[slug] = idx
			idx++
		}
	}
	return pos
}

// phasePosition returns the position for a slug, defaulting to MaxInt for unphased stories.
func phasePosition(pos map[string]int, slug string) int {
	if p, ok := pos[slug]; ok {
		return p
	}
	return math.MaxInt
}

// ExportStory loads a single standalone story and returns it as a StoryExport.
func ExportStory(s *store.Store, storySlug string, opts ExportOptions) (*StoryExport, error) {
	st, err := s.LoadStory(storySlug, "")
	if err != nil {
		return nil, fmt.Errorf("loading story %q: %w", storySlug, err)
	}

	if !opts.IncludeDone && st.Status == models.StoryStatusDone {
		return nil, fmt.Errorf("story %q has status done and include_done is false", storySlug)
	}

	return &StoryExport{
		Slug:        st.Slug,
		Title:       st.Title,
		Status:      st.Status,
		Priority:    st.Priority,
		Labels:      st.Labels,
		Acceptance:  st.Acceptance,
		Description: assembleDescription(st.Body, st.Acceptance, opts.IncludeBody),
	}, nil
}

// assembleDescription builds a pre-assembled description from body and acceptance criteria.
func assembleDescription(body string, acceptance []string, includeBody bool) string {
	var parts []string

	if includeBody && body != "" {
		parts = append(parts, strings.TrimSpace(body))
	}

	if len(acceptance) > 0 {
		var b strings.Builder
		b.WriteString("## Acceptance Criteria")
		for _, ac := range acceptance {
			b.WriteString("\n- [ ] ")
			b.WriteString(ac)
		}
		parts = append(parts, b.String())
	}

	return strings.Join(parts, "\n\n")
}
