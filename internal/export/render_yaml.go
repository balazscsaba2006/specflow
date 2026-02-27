package export

import (
	"fmt"

	"github.com/balazscsaba2006/specflow/internal/models"
	"gopkg.in/yaml.v3"
)

// YAMLRenderer renders an ExportNode tree as YAML.
// For epic nodes it produces the same structure as the original sf_export tool.
type YAMLRenderer struct{}

// yamlExport is the top-level YAML structure for epic exports.
type yamlExport struct {
	Epic    yamlEpic    `yaml:"epic"`
	Stories []yamlStory `yaml:"stories"`
}

type yamlEpic struct {
	Slug     string         `yaml:"slug"`
	Title    string         `yaml:"title"`
	Status   string         `yaml:"status"`
	Fidelity string         `yaml:"fidelity,omitempty"`
	Body     string         `yaml:"body,omitempty"`
	Phases   []models.Phase `yaml:"phases,omitempty"`
}

type yamlStory struct {
	Slug        string   `yaml:"slug"`
	Title       string   `yaml:"title"`
	Status      string   `yaml:"status"`
	Priority    string   `yaml:"priority"`
	Labels      []string `yaml:"labels,omitempty"`
	Acceptance  []string `yaml:"acceptance,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

// yamlNode is a generic YAML representation for non-epic node types.
type yamlNode struct {
	Type        string      `yaml:"type"`
	Slug        string      `yaml:"slug"`
	Title       string      `yaml:"title"`
	Status      string      `yaml:"status,omitempty"`
	Priority    string      `yaml:"priority,omitempty"`
	Fidelity    string      `yaml:"fidelity,omitempty"`
	Labels      []string    `yaml:"labels,omitempty"`
	Body        string      `yaml:"body,omitempty"`
	Acceptance  []string    `yaml:"acceptance,omitempty"`
	DocType     string      `yaml:"doc_type,omitempty"`
	Goal        string      `yaml:"goal,omitempty"`
	Children    []yamlNode  `yaml:"children,omitempty"`
	Docs        []yamlNode  `yaml:"docs,omitempty"`
	Decisions   []yamlNode  `yaml:"decisions,omitempty"`
}

// Render produces YAML output from an ExportNode tree.
// For epic nodes without children (tree=false from old API), it produces the legacy format.
func (r *YAMLRenderer) Render(node *ExportNode, opts RenderOptions) ([]byte, error) {
	// Legacy epic format: when the node is an epic, produce the backward-compatible structure.
	if node.Type == NodeEpic {
		return r.renderLegacyEpic(node, opts)
	}

	// Single story (leaf): produce the same flat YAML as the legacy story export.
	if node.Type == NodeStory && len(node.Children) == 0 {
		return r.renderLegacyStory(node, opts)
	}

	// Generic tree rendering for all other cases.
	yn := r.nodeToYAML(node, opts)
	return yaml.Marshal(yn)
}

func (r *YAMLRenderer) renderLegacyEpic(node *ExportNode, opts RenderOptions) ([]byte, error) {
	data := yamlExport{
		Epic: yamlEpic{
			Slug:     node.Slug,
			Title:    node.Title,
			Status:   node.Status,
			Fidelity: node.Fidelity,
			Phases:   node.Phases,
		},
	}
	if opts.IncludeBody {
		data.Epic.Body = node.Body
	}

	for _, child := range node.Children {
		if child.Type != NodeStory {
			continue
		}
		if !opts.IncludeDone && child.Status == "done" {
			continue
		}
		data.Stories = append(data.Stories, yamlStory{
			Slug:        child.Slug,
			Title:       child.Title,
			Status:      child.Status,
			Priority:    child.Priority,
			Labels:      child.Labels,
			Acceptance:  child.Acceptance,
			Description: assembleDescription(child.Body, child.Acceptance, opts.IncludeBody),
		})
	}

	// If no children were loaded (tree=false), the stories slice is nil — that's fine,
	// it matches the legacy behavior where an epic with no stories returns an empty list.
	return yaml.Marshal(data)
}

func (r *YAMLRenderer) renderLegacyStory(node *ExportNode, opts RenderOptions) ([]byte, error) {
	st := yamlStory{
		Slug:       node.Slug,
		Title:      node.Title,
		Status:     node.Status,
		Priority:   node.Priority,
		Labels:     node.Labels,
		Acceptance: node.Acceptance,
	}

	desc := assembleDescription(node.Body, node.Acceptance, opts.IncludeBody)
	if desc != "" {
		st.Description = desc
	}

	return yaml.Marshal(st)
}

func (r *YAMLRenderer) nodeToYAML(node *ExportNode, opts RenderOptions) yamlNode {
	yn := yamlNode{
		Type:     string(node.Type),
		Slug:     node.Slug,
		Title:    node.Title,
		Status:   node.Status,
		Priority: node.Priority,
		Fidelity: node.Fidelity,
		Labels:   node.Labels,
		DocType:  node.DocType,
		Goal:     node.Goal,
	}
	if opts.IncludeBody {
		yn.Body = node.Body
	}
	if node.Type == NodeStory {
		yn.Acceptance = node.Acceptance
		desc := assembleDescription(node.Body, node.Acceptance, opts.IncludeBody)
		if desc != "" {
			yn.Body = fmt.Sprintf("%s\n\n---\n\n%s", yn.Body, desc)
		}
	}

	for _, c := range node.Children {
		if !opts.IncludeDone && c.Type == NodeStory && c.Status == "done" {
			continue
		}
		yn.Children = append(yn.Children, r.nodeToYAML(c, opts))
	}
	for _, d := range node.Docs {
		yn.Docs = append(yn.Docs, r.nodeToYAML(d, opts))
	}
	for _, d := range node.Decisions {
		yn.Decisions = append(yn.Decisions, r.nodeToYAML(d, opts))
	}

	return yn
}
