package store

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

// ParseFile reads a frontmatter+markdown file and unmarshals the frontmatter
// into dst (must be a pointer to struct). Returns the markdown body.
func ParseFile(path string, dst any) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return Parse(data, dst)
}

// Parse splits frontmatter from body and unmarshals the frontmatter into dst.
func Parse(data []byte, dst any) (string, error) {
	r := bytes.NewReader(data)
	body, err := frontmatter.Parse(r, dst)
	if err != nil {
		return "", fmt.Errorf("parsing frontmatter: %w", err)
	}
	return strings.TrimSpace(string(body)), nil
}

// WriteFile writes a struct as YAML frontmatter + markdown body to a file.
func WriteFile(path string, meta any, body string) error {
	data, err := Marshal(meta, body)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Marshal serializes a struct as YAML frontmatter + markdown body.
func Marshal(meta any, body string) ([]byte, error) {
	yamlData, err := yaml.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshaling frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(yamlData)
	buf.WriteString("---\n")
	if body != "" {
		buf.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			buf.WriteString("\n")
		}
	}
	return buf.Bytes(), nil
}
