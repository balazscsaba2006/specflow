package export

import (
	"fmt"
	"strings"
)

// MarkdownRenderer renders an ExportNode tree as human-readable markdown.
type MarkdownRenderer struct{}

// Render produces markdown output from an ExportNode tree.
func (r *MarkdownRenderer) Render(node *ExportNode, opts RenderOptions) ([]byte, error) {
	var b strings.Builder

	title := opts.Title
	if title == "" {
		title = node.Title
	}

	// For ExtractAll root nodes, render as project export.
	if node.Slug == "" && node.Type == NodeInitiative {
		fmt.Fprintf(&b, "# %s\n\n", title)
		r.renderChildren(&b, node, opts, 2)
		r.renderAttachments(&b, node, opts, 2)
		return []byte(b.String()), nil
	}

	// Apply title override.
	if opts.Title != "" {
		node = shallowCopyNode(node)
		node.Title = opts.Title
	}

	r.renderNode(&b, node, opts, 1)
	return []byte(b.String()), nil
}

func (r *MarkdownRenderer) renderNode(b *strings.Builder, node *ExportNode, opts RenderOptions, depth int) {
	heading := strings.Repeat("#", depth)

	fmt.Fprintf(b, "%s %s\n\n", heading, node.Title)
	r.renderMetadata(b, node)

	if opts.IncludeBody && node.Body != "" {
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(node.Body))
		b.WriteString("\n")
	}

	// Story-specific: acceptance criteria.
	if node.Type == NodeStory && len(node.Acceptance) > 0 {
		fmt.Fprintf(b, "\n%s Acceptance Criteria\n", strings.Repeat("#", depth+1))
		for _, ac := range node.Acceptance {
			fmt.Fprintf(b, "- [ ] %s\n", ac)
		}
	}

	// Epic-specific: phases overview.
	if node.Type == NodeEpic && len(node.Phases) > 0 {
		fmt.Fprintf(b, "\n%s Phases\n", strings.Repeat("#", depth+1))
		childMap := make(map[string]*ExportNode)
		for _, c := range node.Children {
			childMap[c.Slug] = c
		}
		for _, phase := range node.Phases {
			fmt.Fprintf(b, "\n**%s**\n", phase.Label)
			for _, slug := range phase.Stories {
				if c, ok := childMap[slug]; ok {
					fmt.Fprintf(b, "- %s (%s, %s)\n", c.Title, c.Status, c.Priority)
				} else {
					fmt.Fprintf(b, "- %s (filtered)\n", slug)
				}
			}
		}
	}

	b.WriteString("\n")

	// Render children (stories under epic, epics under initiative).
	r.renderChildren(b, node, opts, depth+1)
	r.renderAttachments(b, node, opts, depth+1)
}

func (r *MarkdownRenderer) renderChildren(b *strings.Builder, node *ExportNode, opts RenderOptions, depth int) {
	for _, child := range node.Children {
		if opts.ExcludeStatuses[child.Status] {
			continue
		}
		b.WriteString("---\n\n")
		r.renderNode(b, child, opts, depth)
	}
}

func (r *MarkdownRenderer) renderAttachments(b *strings.Builder, node *ExportNode, opts RenderOptions, depth int) {
	if len(node.Docs) > 0 {
		heading := strings.Repeat("#", depth)
		fmt.Fprintf(b, "%s Documents\n\n", heading)
		for _, doc := range node.Docs {
			r.renderNode(b, doc, opts, depth+1)
		}
	}

	if len(node.Decisions) > 0 {
		heading := strings.Repeat("#", depth)
		fmt.Fprintf(b, "%s Decisions\n\n", heading)
		for _, dec := range node.Decisions {
			fmt.Fprintf(b, "<details>\n<summary>%s</summary>\n\n", dec.Title)
			if opts.IncludeBody && dec.Body != "" {
				b.WriteString(strings.TrimSpace(dec.Body))
				b.WriteString("\n")
			}
			b.WriteString("\n</details>\n\n")
		}
	}
}

// shallowCopyNode creates a shallow copy of an ExportNode (slices shared).
func shallowCopyNode(n *ExportNode) *ExportNode {
	cp := *n
	return &cp
}

func (r *MarkdownRenderer) renderMetadata(b *strings.Builder, node *ExportNode) {
	var parts []string

	if node.Status != "" {
		parts = append(parts, fmt.Sprintf("**Status:** %s", node.Status))
	}
	if node.Priority != "" {
		parts = append(parts, fmt.Sprintf("**Priority:** %s", node.Priority))
	}
	if node.Fidelity != "" {
		parts = append(parts, fmt.Sprintf("**Fidelity:** %s", node.Fidelity))
	}
	if len(node.Labels) > 0 {
		parts = append(parts, fmt.Sprintf("**Labels:** %s", strings.Join(node.Labels, ", ")))
	}
	if node.DocType != "" {
		parts = append(parts, fmt.Sprintf("**Type:** %s", node.DocType))
	}
	if node.Goal != "" {
		parts = append(parts, fmt.Sprintf("**Goal:** %s", node.Goal))
	}

	if len(parts) > 0 {
		b.WriteString(strings.Join(parts, " | "))
		b.WriteString("\n")
	}

	if len(node.SuccessCriteria) > 0 {
		b.WriteString("\n**Success Criteria:**\n")
		for _, sc := range node.SuccessCriteria {
			fmt.Fprintf(b, "- %s\n", sc)
		}
	}
}
