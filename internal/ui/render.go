package ui

import (
	"github.com/charmbracelet/glamour"
)

// RenderMarkdown renders markdown for terminal display using glamour.
// Falls back to the raw input if rendering fails.
func RenderMarkdown(md string) string {
	rendered, err := glamour.Render(md, "auto")
	if err != nil {
		return md
	}
	return rendered
}
