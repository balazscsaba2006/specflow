package export

import (
	"fmt"
	"sort"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
)

// ExtractOptions controls what data is included when extracting entities into ExportNodes.
type ExtractOptions struct {
	IncludeDone bool
	IncludeBody bool
	Tree        bool // include full subtree (children, docs, decisions)
}

// ExtractInitiative loads an initiative and optionally its full subtree.
func ExtractInitiative(s *store.Store, slug string, opts ExtractOptions) (*ExportNode, error) {
	init, err := s.LoadInitiative(slug)
	if err != nil {
		return nil, fmt.Errorf("loading initiative %q: %w", slug, err)
	}

	node := &ExportNode{
		Type:            NodeInitiative,
		Slug:            init.Slug,
		Title:           init.Title,
		Status:          init.Status,
		Goal:            init.Goal,
		SuccessCriteria: init.SuccessCriteria,
	}
	if opts.IncludeBody {
		node.Body = init.Body
	}

	if !opts.Tree {
		return node, nil
	}

	// Load linked epics as children.
	for _, epicSlug := range init.Epics {
		epicNode, err := ExtractEpicNode(s, epicSlug, opts)
		if err != nil {
			return nil, fmt.Errorf("extracting epic %q for initiative %q: %w", epicSlug, slug, err)
		}
		node.Children = append(node.Children, epicNode)
	}

	return node, nil
}

// ExtractEpicNode loads an epic and optionally its full subtree as an ExportNode.
func ExtractEpicNode(s *store.Store, slug string, opts ExtractOptions) (*ExportNode, error) {
	epic, err := s.LoadEpic(slug)
	if err != nil {
		return nil, fmt.Errorf("loading epic %q: %w", slug, err)
	}

	node := &ExportNode{
		Type:     NodeEpic,
		Slug:     epic.Slug,
		Title:    epic.Title,
		Status:   epic.Status,
		Fidelity: epic.Fidelity,
		Phases:   epic.Phases,
	}
	if opts.IncludeBody {
		node.Body = epic.Body
	}

	if !opts.Tree {
		return node, nil
	}

	// Load stories ordered by phase.
	stories, err := s.ListStories(slug)
	if err != nil {
		return nil, fmt.Errorf("listing stories for epic %q: %w", slug, err)
	}

	slugPos := buildPhasePositionMap(epic.Phases)
	sortStoriesByPhase(stories, slugPos)

	for _, st := range stories {
		if !opts.IncludeDone && st.Status == models.StoryStatusDone {
			continue
		}
		node.Children = append(node.Children, storyToNode(st, opts))
	}

	// Load docs scoped to this epic.
	docs, err := s.ListDocs(slug)
	if err != nil {
		return nil, fmt.Errorf("listing docs for epic %q: %w", slug, err)
	}
	for _, d := range docs {
		node.Docs = append(node.Docs, docToNode(d, opts))
	}

	// Load decisions (project-level, filtered by context_refs to this epic).
	decisions, err := s.ListDecisions()
	if err != nil {
		return nil, fmt.Errorf("listing decisions: %w", err)
	}
	for _, d := range decisions {
		if hasContextRef(d.ContextRefs, slug) {
			node.Decisions = append(node.Decisions, decisionToNode(d, opts))
		}
	}

	return node, nil
}

// ExtractStoryNode loads a standalone or epic-scoped story as an ExportNode.
func ExtractStoryNode(s *store.Store, slug string, opts ExtractOptions) (*ExportNode, error) {
	st, err := s.LoadStory(slug, "")
	if err != nil {
		return nil, fmt.Errorf("loading story %q: %w", slug, err)
	}

	if !opts.IncludeDone && st.Status == models.StoryStatusDone {
		return nil, fmt.Errorf("story %q has status done and include_done is false", slug)
	}

	return storyToNode(st, opts), nil
}

// ExtractDoc loads a document as an ExportNode.
func ExtractDoc(s *store.Store, slug, epicSlug string, opts ExtractOptions) (*ExportNode, error) {
	d, err := s.LoadDoc(slug, epicSlug)
	if err != nil {
		return nil, fmt.Errorf("loading doc %q: %w", slug, err)
	}
	return docToNode(d, opts), nil
}

// ExtractDecision loads a decision as an ExportNode.
func ExtractDecision(s *store.Store, slug string, opts ExtractOptions) (*ExportNode, error) {
	d, err := s.LoadDecision(slug)
	if err != nil {
		return nil, fmt.Errorf("loading decision %q: %w", slug, err)
	}
	return decisionToNode(d, opts), nil
}

// ExtractAll builds a root ExportNode containing the entire project hierarchy.
func ExtractAll(s *store.Store, opts ExtractOptions) (*ExportNode, error) {
	treeOpts := opts
	treeOpts.Tree = true

	root := &ExportNode{
		Type:  NodeInitiative,
		Title: "Project Export",
	}

	epicInInitiative, err := extractAllInitiatives(s, root, treeOpts)
	if err != nil {
		return nil, err
	}

	if err := extractStandaloneEpics(s, root, epicInInitiative, treeOpts); err != nil {
		return nil, err
	}

	if err := extractStandaloneStories(s, root, opts); err != nil {
		return nil, err
	}

	if err := extractProjectDocsAndDecisions(s, root, opts); err != nil {
		return nil, err
	}

	return root, nil
}

func extractAllInitiatives(s *store.Store, root *ExportNode, opts ExtractOptions) (map[string]bool, error) {
	initiatives, err := s.ListInitiatives()
	if err != nil {
		return nil, fmt.Errorf("listing initiatives: %w", err)
	}

	epicInInitiative := make(map[string]bool)
	for _, init := range initiatives {
		initNode, initErr := ExtractInitiative(s, init.Slug, opts)
		if initErr != nil {
			return nil, fmt.Errorf("extracting initiative %q: %w", init.Slug, initErr)
		}
		root.Children = append(root.Children, initNode)
		for _, e := range init.Epics {
			epicInInitiative[e] = true
		}
	}
	return epicInInitiative, nil
}

func extractStandaloneEpics(s *store.Store, root *ExportNode, epicInInitiative map[string]bool, opts ExtractOptions) error {
	epics, err := s.ListEpics()
	if err != nil {
		return fmt.Errorf("listing epics: %w", err)
	}
	for _, e := range epics {
		if epicInInitiative[e.Slug] {
			continue
		}
		epicNode, epicErr := ExtractEpicNode(s, e.Slug, opts)
		if epicErr != nil {
			return fmt.Errorf("extracting epic %q: %w", e.Slug, epicErr)
		}
		root.Children = append(root.Children, epicNode)
	}
	return nil
}

func extractStandaloneStories(s *store.Store, root *ExportNode, opts ExtractOptions) error {
	stories, err := s.ListStories("")
	if err != nil {
		return fmt.Errorf("listing standalone stories: %w", err)
	}
	for _, st := range stories {
		if !opts.IncludeDone && st.Status == models.StoryStatusDone {
			continue
		}
		root.Children = append(root.Children, storyToNode(st, opts))
	}
	return nil
}

func extractProjectDocsAndDecisions(s *store.Store, root *ExportNode, opts ExtractOptions) error {
	docs, err := s.ListDocs("")
	if err != nil {
		return fmt.Errorf("listing project docs: %w", err)
	}
	for _, d := range docs {
		root.Docs = append(root.Docs, docToNode(d, opts))
	}

	decisions, err := s.ListDecisions()
	if err != nil {
		return fmt.Errorf("listing decisions: %w", err)
	}
	for _, d := range decisions {
		if len(d.ContextRefs) == 0 {
			root.Decisions = append(root.Decisions, decisionToNode(d, opts))
		}
	}
	return nil
}

// --- helpers ---

func storyToNode(st *models.Story, opts ExtractOptions) *ExportNode {
	node := &ExportNode{
		Type:       NodeStory,
		Slug:       st.Slug,
		Title:      st.Title,
		Status:     st.Status,
		Priority:   st.Priority,
		Labels:     st.Labels,
		Acceptance: st.Acceptance,
	}
	if opts.IncludeBody {
		node.Body = st.Body
	}
	return node
}

func docToNode(d *models.Document, opts ExtractOptions) *ExportNode {
	node := &ExportNode{
		Type:    NodeDoc,
		Slug:    d.Slug,
		Title:   d.Title,
		Status:  d.Status,
		DocType: d.Type,
	}
	if opts.IncludeBody {
		node.Body = d.Body
	}
	return node
}

func decisionToNode(d *models.Decision, opts ExtractOptions) *ExportNode {
	node := &ExportNode{
		Type:   NodeDecision,
		Slug:   d.Slug,
		Title:  d.Title,
		Status: d.Status,
	}
	if opts.IncludeBody {
		node.Body = d.Body
	}
	return node
}

func hasContextRef(refs []string, target string) bool {
	for _, ref := range refs {
		if strings.EqualFold(ref, target) {
			return true
		}
	}
	return false
}

func sortStoriesByPhase(stories []*models.Story, slugPos map[string]int) {
	sort.SliceStable(stories, func(i, j int) bool {
		return phasePosition(slugPos, stories[i].Slug) < phasePosition(slugPos, stories[j].Slug)
	})
}
