package main

import (
	"fmt"
	"os"

	"github.com/balazscsaba2006/specflow/internal/export"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [slug]",
		Short: "Export entities to markdown, HTML, or YAML",
		Long: `Export specflow entities (initiatives, epics, stories, docs, decisions) to various formats.

Auto-detects the entity type from the slug. Use --all to export the entire project.
Use --exclude-status to skip entities with specific statuses (e.g. done, cancelled).

Examples:
  specflow export my-epic                                   # markdown export of an epic
  specflow export my-epic --format html                     # HTML export with Mermaid + code highlighting
  specflow export my-epic --tree                            # include full subtree (stories, docs, decisions)
  specflow export my-epic --tree --exclude-status done      # subtree without done stories
  specflow export --all --format html                       # export entire project as HTML
  specflow export --all --exclude-status done,cancelled     # skip done and cancelled entities`,
		Args: cobra.MaximumNArgs(1),
		RunE: runExport,
	}

	cmd.Flags().StringP("format", "f", "md", "Output format: md, html, yaml")
	cmd.Flags().StringP("output", "o", "", "Output file path (default: <slug>-export.<ext>)")
	cmd.Flags().BoolP("tree", "t", false, "Include full subtree (children, docs, decisions)")
	cmd.Flags().Bool("all", false, "Export entire project hierarchy")
	cmd.Flags().Bool("no-body", false, "Omit markdown body content")
	cmd.Flags().StringSlice("exclude-status", nil, "Skip entities with these statuses (comma-separated, e.g. done,cancelled)")
	cmd.Flags().Bool("exclude-done", false, "Skip stories with status done (deprecated: use --exclude-status done)")
	_ = cmd.Flags().MarkHidden("exclude-done")

	return cmd
}

func runExport(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	output, _ := cmd.Flags().GetString("output")
	tree, _ := cmd.Flags().GetBool("tree")
	all, _ := cmd.Flags().GetBool("all")
	noBody, _ := cmd.Flags().GetBool("no-body")
	excludeStatusSlice, _ := cmd.Flags().GetStringSlice("exclude-status")
	excludeDone, _ := cmd.Flags().GetBool("exclude-done")

	// Backwards compat: --exclude-done maps to --exclude-status done.
	if excludeDone && len(excludeStatusSlice) == 0 {
		excludeStatusSlice = []string{"done"}
	}

	excludeStatuses := make(map[string]bool, len(excludeStatusSlice))
	for _, s := range excludeStatusSlice {
		excludeStatuses[s] = true
	}

	if !all && len(args) == 0 {
		return fmt.Errorf("provide a slug or use --all to export the entire project")
	}

	extOpts := export.ExtractOptions{
		ExcludeStatuses: excludeStatuses,
		IncludeBody:     !noBody,
		Tree:            tree || all, // --all implies tree
	}

	renderOpts := export.RenderOptions{
		IncludeBody:     !noBody,
		ExcludeStatuses: excludeStatuses,
	}

	var node *export.ExportNode
	var slug string
	var err error

	if all {
		slug = "project"
		node, err = export.ExtractAll(appStore, extOpts)
	} else {
		slug = args[0]
		node, err = resolveAndExtract(slug, extOpts)
	}
	if err != nil {
		return err
	}

	renderer, ext := getRenderer(format)
	if renderer == nil {
		return fmt.Errorf("unsupported format %q (use md, html, or yaml)", format)
	}

	data, err := renderer.Render(node, renderOpts)
	if err != nil {
		return fmt.Errorf("rendering export: %w", err)
	}

	if output == "" {
		output = slug + "-export" + ext
	}

	if err := os.WriteFile(output, data, 0o600); err != nil {
		return fmt.Errorf("writing export file: %w", err)
	}

	summary := buildExportSummary(node)
	fmt.Printf("Exported %s to %s\n", summary, output)
	return nil
}

// resolveAndExtract auto-detects the entity type from the slug and extracts it.
func resolveAndExtract(slug string, opts export.ExtractOptions) (*export.ExportNode, error) {
	// Try initiative first.
	if node, err := export.ExtractInitiative(appStore, slug, opts); err == nil {
		return node, nil
	}

	// Try epic.
	if node, err := export.ExtractEpicNode(appStore, slug, opts); err == nil {
		return node, nil
	}

	// Try standalone story.
	if node, err := export.ExtractStoryNode(appStore, slug, opts); err == nil {
		return node, nil
	}

	// Try doc (project-level, then epic-scoped would need epic context).
	if node, err := export.ExtractDoc(appStore, slug, "", opts); err == nil {
		return node, nil
	}

	// Try decision.
	if node, err := export.ExtractDecision(appStore, slug, opts); err == nil {
		return node, nil
	}

	return nil, fmt.Errorf("entity %q not found (checked: initiative, epic, story, doc, decision)", slug)
}

func getRenderer(format string) (renderer export.Renderer, ext string) {
	switch format {
	case "md", "markdown":
		return &export.MarkdownRenderer{}, ".md"
	case "html":
		return &export.HTMLRenderer{}, ".html"
	case "yaml", "yml":
		return &export.YAMLRenderer{}, ".yaml"
	default:
		return nil, ""
	}
}

func buildExportSummary(node *export.ExportNode) string {
	if node.Type == export.NodeInitiative && node.Slug == "" {
		// Full project export.
		return fmt.Sprintf("project (%d top-level entities)", len(node.Children))
	}

	childCount := countChildren(node)
	if childCount > 0 {
		return fmt.Sprintf("1 %s + %d children", node.Type, childCount)
	}
	return fmt.Sprintf("1 %s (%s)", node.Type, node.Slug)
}

func countChildren(node *export.ExportNode) int {
	count := len(node.Children)
	for _, c := range node.Children {
		count += countChildren(c)
	}
	return count
}

