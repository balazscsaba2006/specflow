// Package templates provides embedded default templates with user-override support.
//
// Resolution order for template loading:
//  1. .specflow/templates/<name>.md  (user override)
//  2. Embedded default from go:embed
package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed *.md
var embedded embed.FS

// Load returns the template content for the given name (e.g. "story", "doc_prd").
// It checks specflowRoot/templates/<name>.md first; if not found, falls back to
// the embedded default.
func Load(specflowRoot, name string) (string, error) {
	// Try user override first.
	if specflowRoot != "" {
		overridePath := filepath.Join(specflowRoot, "templates", name+".md")
		if data, err := os.ReadFile(overridePath); err == nil {
			return string(data), nil
		}
	}

	// Fall back to embedded.
	data, err := embedded.ReadFile(name + ".md")
	if err != nil {
		return "", fmt.Errorf("template %q not found", name)
	}
	return string(data), nil
}

// LoadDoc returns the template for a document type. It maps known types (prd, adr)
// to their specific templates, and falls back to the generic doc template for others.
// For the generic template, the type placeholder is filled in.
func LoadDoc(specflowRoot, docType string) (string, error) {
	name := "doc_" + docType
	tmpl, err := Load(specflowRoot, name)
	if err != nil {
		// Unknown doc type — use generic template with type filled in.
		tmpl, err = Load(specflowRoot, "doc_generic")
		if err != nil {
			return "", err
		}
		tmpl = strings.Replace(tmpl, `type: ""`, fmt.Sprintf("type: %s", docType), 1)
	}
	return tmpl, nil
}
