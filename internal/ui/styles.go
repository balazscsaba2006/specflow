package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Status badge colors.
var (
	colorDone       = lipgloss.Color("2")  // green
	colorInProgress = lipgloss.Color("3")  // yellow
	colorBlocked    = lipgloss.Color("1")  // red
	colorDraft      = lipgloss.Color("8")  // gray
	colorPlanned    = lipgloss.Color("6")  // cyan
	colorVerifying  = lipgloss.Color("5")  // magenta
	colorActive     = lipgloss.Color("2")  // green
)

var statusColors = map[string]lipgloss.Color{
	"done":        colorDone,
	"in_progress": colorInProgress,
	"blocked":     colorBlocked,
	"draft":       colorDraft,
	"planned":     colorPlanned,
	"verifying":   colorVerifying,
	"active":      colorActive,
	"review":      colorVerifying,
	"approved":    colorDone,
	"superseded":  colorDraft,
	"started":     colorInProgress,
	"completed":   colorDone,
	"failed":      colorBlocked,
}

// StatusBadge returns a colored status string.
func StatusBadge(status string) string {
	color, ok := statusColors[status]
	if !ok {
		return status
	}
	style := lipgloss.NewStyle().Foreground(color).Bold(true)
	return style.Render(status)
}

// PriorityBadge returns a styled priority string.
func PriorityBadge(priority string) string {
	switch priority {
	case "critical":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Render(priority)
	case "high":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(priority)
	case "low":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(priority)
	default:
		return priority
	}
}

// Header renders a styled section header.
func Header(text string) string {
	style := lipgloss.NewStyle().Bold(true).Underline(true)
	return style.Render(text)
}

// Label renders a dimmed label for key-value pairs.
func Label(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(text)
}

// ProgressBar renders a simple text-based progress bar.
func ProgressBar(done, total, width int) string {
	if total == 0 {
		return Label("—")
	}
	pct := float64(done) / float64(total)
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pctStr := fmt.Sprintf("%3.0f%%", pct*100)

	greenStyle := lipgloss.NewStyle().Foreground(colorDone)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	return greenStyle.Render(bar[:filled]) + dimStyle.Render(bar[filled:]) + " " + pctStr
}

// Table renders a formatted table with headers and rows, with column alignment.
func Table(headers []string, rows [][]string) string {
	if len(rows) == 0 {
		return Label("No results.")
	}

	// Calculate column widths.
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			// Strip ANSI for width calculation.
			plain := stripAnsi(cell)
			if i < len(widths) && len(plain) > widths[i] {
				widths[i] = len(plain)
			}
		}
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var b strings.Builder

	// Header row.
	for i, h := range headers {
		fmt.Fprintf(&b, "%-*s  ", widths[i], headerStyle.Render(h))
	}
	b.WriteString("\n")

	// Separator.
	for i := range headers {
		b.WriteString(sepStyle.Render(strings.Repeat("─", widths[i])))
		b.WriteString("  ")
	}
	b.WriteString("\n")

	// Data rows.
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				continue
			}
			// Pad based on plain text width to handle ANSI escape codes.
			plain := stripAnsi(cell)
			padding := widths[i] - len(plain)
			if padding < 0 {
				padding = 0
			}
			b.WriteString(cell)
			b.WriteString(strings.Repeat(" ", padding))
			b.WriteString("  ")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// stripAnsi removes ANSI escape sequences for width calculation.
func stripAnsi(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
