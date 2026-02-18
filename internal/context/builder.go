package context

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/balazscsaba2006/specflow/internal/config"
	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
)

// ContextData holds all assembled data for the context template.
type ContextData struct {
	// Layer 1: Project conventions
	ClaudeMD  string
	AgentsMD  string
	Config    config.Config
	HasConfig bool

	// Layer 2: Epic/initiative context
	Story             *models.Story
	Epic              *models.Epic
	Initiative        *models.Initiative
	CompletedStories  []*models.Story
	InProgressStories []*models.Story
	Decisions         []*models.Decision

	// Layer 3: Spec requirements
	Docs []*models.Document

	// Layer 4: Implementation plan
	Plan    *models.Plan
	HasPlan bool

	// Layer 5: Referenced files
	ReferencedFiles []FileContent

	// Layer 6: Open items
	OpenQuestions []QuestionItem
	Assumptions   []AssumptionItem
	Blockers      []string
}

// QuestionItem holds an open question with its source.
type QuestionItem struct {
	Source   string
	Question string
}

// AssumptionItem holds an assumption with its source story.
type AssumptionItem struct {
	Story      string
	Assumption string
}

// FileContent holds a referenced file's path and content.
type FileContent struct {
	Path    string
	Content string
	Exists  bool
}

// Builder assembles context for story execution.
type Builder struct {
	store       *store.Store
	cfg         config.Config
	projectRoot string
}

// New creates a context Builder.
func New(s *store.Store, cfg config.Config) *Builder {
	return &Builder{
		store:       s,
		cfg:         cfg,
		projectRoot: s.ProjectRoot(),
	}
}

// Build assembles the full 6-layer context for a story.
func (b *Builder) Build(storySlug string) (string, error) {
	data, err := b.assemble(storySlug)
	if err != nil {
		return "", fmt.Errorf("building context for %q: %w", storySlug, err)
	}

	return b.render(data)
}

func (b *Builder) assemble(storySlug string) (*ContextData, error) {
	data := &ContextData{
		Config:    b.cfg,
		HasConfig: true,
	}

	// Load the story.
	story, err := b.findStory(storySlug)
	if err != nil {
		return nil, fmt.Errorf("loading story: %w", err)
	}
	data.Story = story

	// Layer 1: Project conventions.
	b.assembleConventions(data)

	// Layer 2: Epic/initiative context.
	b.assembleEpicContext(data, story)

	// Layer 3: Specs / referenced docs.
	b.assembleDocs(data, story)

	// Layer 4: Implementation plan.
	b.assemblePlan(data, storySlug)

	// Layer 5: Referenced files.
	b.assembleReferencedFiles(data)

	// Layer 6: Open items.
	data.OpenQuestions = b.collectQuestions(data)
	data.Assumptions = b.collectAssumptions(data)
	data.Blockers = story.BlockedBy

	return data, nil
}

// assembleConventions loads Layer 1: project conventions files.
func (b *Builder) assembleConventions(data *ContextData) {
	data.ClaudeMD = b.readProjectFile(b.cfg.ConventionsFile)
	if b.cfg.AgentsFile != "" {
		data.AgentsMD = b.readProjectFile(b.cfg.AgentsFile)
	}
}

// assembleEpicContext loads Layer 2: epic, initiative, sibling stories, decisions.
func (b *Builder) assembleEpicContext(data *ContextData, story *models.Story) {
	if story.Epic == "" {
		return
	}

	epic, epicErr := b.store.LoadEpic(story.Epic)
	if epicErr != nil {
		return
	}
	data.Epic = epic

	// Load initiative if referenced.
	if epic.Initiative != "" {
		initiative, initErr := b.store.LoadInitiative(epic.Initiative)
		if initErr == nil {
			data.Initiative = initiative
		}
	}

	// Load sibling stories.
	siblings, sibErr := b.store.ListStories(story.Epic)
	if sibErr == nil {
		for _, s := range siblings {
			if s.Slug == story.Slug {
				continue
			}
			switch s.Status {
			case models.StoryStatusDone:
				data.CompletedStories = append(data.CompletedStories, s)
			case models.StoryStatusInProgress:
				data.InProgressStories = append(data.InProgressStories, s)
			}
		}
	}

	// Load decisions referencing this epic.
	decisions, decErr := b.store.ListDecisions()
	if decErr != nil {
		return
	}
	for _, d := range decisions {
		for _, ref := range d.ContextRefs {
			if ref == story.Epic {
				data.Decisions = append(data.Decisions, d)
				break
			}
		}
	}
	// If no decisions have context refs matching the epic, include all.
	if len(data.Decisions) == 0 {
		data.Decisions = decisions
	}
}

// assembleDocs loads Layer 3: documents referenced by the story.
func (b *Builder) assembleDocs(data *ContextData, story *models.Story) {
	for _, docRef := range story.DocRefs {
		doc, docErr := b.store.LoadDoc(docRef, story.Epic)
		if docErr != nil {
			// Try project-level docs if epic-scoped failed.
			doc, docErr = b.store.LoadDoc(docRef, "")
			if docErr != nil {
				// Try archived epic docs as last resort.
				doc = b.findArchivedDoc(docRef)
				if doc == nil {
					continue
				}
			}
		}
		data.Docs = append(data.Docs, doc)
	}
}

// findArchivedDoc searches for a doc in archived epics.
func (b *Builder) findArchivedDoc(slug string) *models.Document {
	archivedEpics, err := b.store.ListArchivedEpics()
	if err != nil {
		return nil
	}
	for _, ep := range archivedEpics {
		docPath := filepath.Join(b.store.ArchiveEpicDocsDir(ep.Slug), slug+".md")
		var doc models.Document
		body, parseErr := store.ParseFile(docPath, &doc)
		if parseErr == nil {
			doc.Body = body
			return &doc
		}
	}
	return nil
}

// assemblePlan loads Layer 4: implementation plan.
func (b *Builder) assemblePlan(data *ContextData, storySlug string) {
	plan, planErr := b.store.LoadPlan(storySlug)
	if planErr == nil {
		data.Plan = plan
		data.HasPlan = true
	}
}

// assembleReferencedFiles loads Layer 5: files referenced in plan and config.
func (b *Builder) assembleReferencedFiles(data *ContextData) {
	var filePaths []string
	if data.HasPlan {
		filePaths = append(filePaths, ExtractFileRefs(data.Plan.Body)...)
	}
	for _, path := range b.cfg.PatternExemplars {
		filePaths = append(filePaths, path)
	}
	for _, path := range filePaths {
		data.ReferencedFiles = append(data.ReferencedFiles, b.readReferencedFile(path))
	}
}

// findStory searches for a story across all epics, standalone stories, and archived epics.
func (b *Builder) findStory(slug string) (*models.Story, error) {
	all, err := b.store.ListAllStories()
	if err != nil {
		return nil, err
	}
	for _, st := range all {
		if st.Slug == slug {
			return st, nil
		}
	}

	// Fall back to archived epics.
	archivedEpics, archErr := b.store.ListArchivedEpics()
	if archErr == nil {
		for _, ep := range archivedEpics {
			if st, loadErr := b.store.LoadArchivedStory(slug, ep.Slug); loadErr == nil {
				return st, nil
			}
		}
	}

	return nil, fmt.Errorf("story %q not found", slug)
}

// readProjectFile reads a file relative to the project root.
func (b *Builder) readProjectFile(name string) string {
	if name == "" {
		return ""
	}
	path := filepath.Join(b.projectRoot, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// readReferencedFile reads a file and returns its content info.
func (b *Builder) readReferencedFile(path string) FileContent {
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(b.projectRoot, path)
	}

	data, err := os.ReadFile(absPath) //nolint:gosec // reading project files referenced in plan
	if err != nil {
		return FileContent{Path: path, Exists: false}
	}
	return FileContent{Path: path, Content: string(data), Exists: true}
}

// collectQuestions gathers open questions from all layers.
func (b *Builder) collectQuestions(data *ContextData) []QuestionItem {
	var items []QuestionItem

	if data.Initiative != nil {
		for _, q := range data.Initiative.OpenQuestions {
			items = append(items, QuestionItem{Source: "initiative:" + data.Initiative.Slug, Question: q})
		}
	}
	if data.Epic != nil {
		for _, q := range data.Epic.OpenQuestions {
			items = append(items, QuestionItem{Source: "epic:" + data.Epic.Slug, Question: q})
		}
	}
	for _, q := range data.Story.OpenQuestions {
		items = append(items, QuestionItem{Source: "story:" + data.Story.Slug, Question: q})
	}
	for _, doc := range data.Docs {
		for _, q := range doc.OpenQuestions {
			items = append(items, QuestionItem{Source: "doc:" + doc.Slug, Question: q})
		}
	}

	return items
}

// collectAssumptions gathers assumptions from completed sibling stories.
func (b *Builder) collectAssumptions(data *ContextData) []AssumptionItem {
	var items []AssumptionItem
	for _, st := range data.CompletedStories {
		for _, a := range st.Assumptions {
			items = append(items, AssumptionItem{Story: st.Slug, Assumption: a})
		}
	}
	return items
}

func (b *Builder) render(data *ContextData) (string, error) {
	tmpl, err := template.New("context").Parse(contextTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing context template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering context template: %w", err)
	}

	return buf.String(), nil
}
