package export

import "github.com/balazscsaba2006/specflow/internal/models"

// NodeType identifies the kind of entity an ExportNode represents.
type NodeType string

const (
	NodeInitiative NodeType = "initiative"
	NodeEpic       NodeType = "epic"
	NodeStory      NodeType = "story"
	NodeDoc        NodeType = "doc"
	NodeDecision   NodeType = "decision"
)

// ExportNode is the unified tree data model for exporting any specflow entity.
// It supports recursive nesting via Children, Docs, and Decisions fields.
type ExportNode struct {
	Type     NodeType
	Slug     string
	Title    string
	Status   string
	Priority string
	Fidelity string
	Labels   []string
	Body     string

	// Story-specific
	Acceptance []string

	// Epic-specific
	Phases []models.Phase

	// Initiative-specific
	Goal            string
	SuccessCriteria []string

	// Doc-specific
	DocType string // prd, tech-spec, etc.

	// Recursive children
	Children  []*ExportNode // nested entities (epics under initiative, stories under epic)
	Docs      []*ExportNode // attached documents
	Decisions []*ExportNode // attached decisions
}

// Renderer transforms an ExportNode tree into output bytes.
type Renderer interface {
	Render(node *ExportNode, opts RenderOptions) ([]byte, error)
}

// RenderOptions controls renderer behavior.
type RenderOptions struct {
	IncludeBody bool
	IncludeDone bool
	Title       string // override document title
}
