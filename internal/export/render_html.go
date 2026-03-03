package export

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	goldhtml "github.com/yuin/goldmark/renderer/html"
)

//go:embed templates/export.html
var templateFS embed.FS

// HTMLRenderer renders an ExportNode tree as a self-contained HTML file.
type HTMLRenderer struct{}

type htmlTemplateData struct {
	Title   string
	Content template.HTML
	Date    string
}

var mermaidBlockRe = regexp.MustCompile("(?s)```mermaid\n(.*?)```")

// Render produces a self-contained HTML file from an ExportNode tree.
func (r *HTMLRenderer) Render(node *ExportNode, opts RenderOptions) ([]byte, error) {
	tmplBytes, err := templateFS.ReadFile("templates/export.html")
	if err != nil {
		return nil, fmt.Errorf("reading HTML template: %w", err)
	}

	tmpl, err := template.New("export").Parse(string(tmplBytes))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML template: %w", err)
	}

	// Build markdown content first, then convert to HTML.
	mdRenderer := &MarkdownRenderer{}
	mdBytes, err := mdRenderer.Render(node, opts)
	if err != nil {
		return nil, fmt.Errorf("rendering markdown for HTML: %w", err)
	}

	// Extract mermaid blocks before goldmark processing.
	mdStr := string(mdBytes)
	var mermaidBlocks []string
	placeholder := "MERMAID_PLACEHOLDER_%d"
	mdStr = mermaidBlockRe.ReplaceAllStringFunc(mdStr, func(match string) string {
		// Extract content between ```mermaid and ```.
		content := mermaidBlockRe.FindStringSubmatch(match)[1]
		idx := len(mermaidBlocks)
		mermaidBlocks = append(mermaidBlocks, strings.TrimSpace(content))
		return fmt.Sprintf(placeholder, idx)
	})

	// Convert markdown to HTML using goldmark.
	md := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(),
			),
		),
		goldmark.WithRendererOptions(
			goldhtml.WithUnsafe(),
		),
	)

	var htmlBuf bytes.Buffer
	if err := md.Convert([]byte(mdStr), &htmlBuf); err != nil {
		return nil, fmt.Errorf("converting markdown to HTML: %w", err)
	}

	// Restore mermaid blocks as proper divs.
	htmlContent := htmlBuf.String()
	for i, content := range mermaidBlocks {
		ph := fmt.Sprintf("<p>"+placeholder+"</p>", i)
		mermaidDiv := fmt.Sprintf(`<pre class="mermaid">%s</pre>`, html.EscapeString(content))
		htmlContent = strings.Replace(htmlContent, ph, mermaidDiv, 1)

		// Also try without <p> wrapper in case goldmark didn't wrap it.
		phRaw := fmt.Sprintf(placeholder, i)
		htmlContent = strings.Replace(htmlContent, phRaw, mermaidDiv, 1)
	}

	// Add badge classes to status spans if present.
	htmlContent = addStatusBadges(htmlContent)

	// Build TOC from the node tree.
	toc := buildTOC(node)

	// Combine TOC + content.
	var fullContent strings.Builder
	if toc != "" {
		fullContent.WriteString(toc)
	}
	fullContent.WriteString(htmlContent)

	title := opts.Title
	if title == "" {
		title = node.Title
	}

	data := htmlTemplateData{
		Title:   title,
		Content: template.HTML(fullContent.String()), //nolint:gosec // we control the content
		Date:    time.Now().Format("2006-01-02"),
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return nil, fmt.Errorf("executing HTML template: %w", err)
	}

	return out.Bytes(), nil
}

func addStatusBadges(content string) string {
	statuses := []string{"done", "active", "in_progress", "planned", "draft", "on_hold", "blocked",
		"completed", "verifying", "cancelled"}
	for _, status := range statuses {
		// Match bold status text like **Status:** done.
		old := fmt.Sprintf("<strong>Status:</strong> %s", status)
		badge := badgeClass(status)
		replacement := fmt.Sprintf(`<strong>Status:</strong> <span class="badge %s">%s</span>`, badge, status)
		content = strings.ReplaceAll(content, old, replacement)
	}
	return content
}

func badgeClass(status string) string {
	switch status {
	case "done", "completed":
		return "badge-done"
	case "active", "in_progress", "planned":
		return "badge-active"
	case "blocked", "cancelled":
		return "badge-blocked"
	default:
		return "badge-draft"
	}
}

func buildTOC(node *ExportNode) string {
	var b strings.Builder
	b.WriteString(`<details class="toc" open>`)
	b.WriteString("<summary>Table of Contents</summary>\n<ul>\n")
	buildTOCItems(&b, node, true)
	b.WriteString("</ul>\n</details>\n")
	return b.String()
}

func buildTOCItems(b *strings.Builder, node *ExportNode, isRoot bool) {
	if !isRoot && node.Title != "" {
		slug := slugify(node.Title)
		fmt.Fprintf(b, `<li><a href="#%s">%s</a>`, slug, html.EscapeString(node.Title))
		if node.Status != "" {
			fmt.Fprintf(b, ` <span class="badge %s">%s</span>`, badgeClass(node.Status), node.Status)
		}
	}

	hasChildren := len(node.Children) > 0 || len(node.Docs) > 0 || len(node.Decisions) > 0
	if hasChildren {
		if !isRoot {
			b.WriteString("\n<ul>\n")
		}
		for _, c := range node.Children {
			buildTOCItems(b, c, false)
		}
		for _, d := range node.Docs {
			buildTOCItems(b, d, false)
		}
		for _, d := range node.Decisions {
			buildTOCItems(b, d, false)
		}
		if !isRoot {
			b.WriteString("</ul>\n")
		}
	}

	if !isRoot && node.Title != "" {
		b.WriteString("</li>\n")
	}
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		if r == ' ' || r == '_' {
			return '-'
		}
		return -1
	}, s)
	// Collapse multiple hyphens.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}
