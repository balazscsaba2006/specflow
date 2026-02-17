package context

import (
	"regexp"
	"strings"
)

// fileRefPatterns matches common ways files are referenced in plan markdown:
// - `path/to/file.go` (backtick-quoted)
// - file: path/to/file.go (YAML-style)
// - - path/to/file.go (list item that looks like a file path)
var fileRefPatterns = []*regexp.Regexp{
	regexp.MustCompile("`([^`]+\\.[a-zA-Z]+)`"),
	regexp.MustCompile(`(?:file|path|pattern_reference):\s*["']?([^\s"']+\.[a-zA-Z]+)["']?`),
}

// pathLikePattern matches strings that look like file paths (contain / and end with extension).
var pathLikePattern = regexp.MustCompile(`(?:^|\s)((?:[a-zA-Z0-9_\-./]+/)+[a-zA-Z0-9_\-]+\.[a-zA-Z]+)`)

// ExtractFileRefs extracts file paths from plan markdown content.
// It looks for backtick-quoted paths, YAML-style file references,
// and path-like strings (containing / with file extension).
func ExtractFileRefs(content string) []string {
	seen := make(map[string]struct{})
	var refs []string

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try explicit patterns first.
		for _, pat := range fileRefPatterns {
			matches := pat.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				path := m[1]
				if isLikelyFilePath(path) {
					if _, ok := seen[path]; !ok {
						seen[path] = struct{}{}
						refs = append(refs, path)
					}
				}
			}
		}

		// Try path-like pattern.
		matches := pathLikePattern.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			path := m[1]
			if _, ok := seen[path]; !ok {
				seen[path] = struct{}{}
				refs = append(refs, path)
			}
		}
	}

	return refs
}

// isLikelyFilePath checks if a string looks like a file path
// (has a directory separator and a common extension).
func isLikelyFilePath(s string) bool {
	if !strings.Contains(s, "/") && !strings.Contains(s, ".") {
		return false
	}

	// Must have a file extension.
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}

	ext := parts[len(parts)-1]
	// Common source file extensions.
	validExts := map[string]bool{
		"go": true, "py": true, "js": true, "ts": true, "tsx": true,
		"jsx": true, "rs": true, "java": true, "rb": true, "php": true,
		"yaml": true, "yml": true, "json": true, "toml": true,
		"md": true, "txt": true, "sql": true, "sh": true,
		"css": true, "html": true, "xml": true, "proto": true,
		"mod": true, "sum": true, "lock": true, "tmpl": true,
	}
	return validExts[ext]
}
