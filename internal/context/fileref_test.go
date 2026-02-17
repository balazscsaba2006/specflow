package context

import (
	"testing"
)

func TestExtractFileRefs(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name:    "backtick-quoted paths",
			content: "1. Create `internal/middleware/jwt.go`\n2. Write tests in `internal/middleware/jwt_test.go`",
			expected: []string{
				"internal/middleware/jwt.go",
				"internal/middleware/jwt_test.go",
			},
		},
		{
			name:    "yaml-style file references",
			content: "file: internal/api/handler.go\npattern_reference: internal/models/user.go",
			expected: []string{
				"internal/api/handler.go",
				"internal/models/user.go",
			},
		},
		{
			name:    "path-like strings in text",
			content: "Look at internal/store/story.go for the pattern.\nAlso check cmd/specflow/main.go.",
			expected: []string{
				"internal/store/story.go",
				"cmd/specflow/main.go",
			},
		},
		{
			name:    "deduplicates paths",
			content: "Create `internal/api/handler.go` based on internal/api/handler.go pattern",
			expected: []string{
				"internal/api/handler.go",
			},
		},
		{
			name:     "ignores non-file strings",
			content:  "Use JWT tokens with RS256 algorithm. The API should return 401.",
			expected: nil,
		},
		{
			name:    "mixed content",
			content: "## Plan\n\n- Edit `internal/middleware/auth.go`\n- Update config in `config/app.yaml`\n\nReferences:\nfile: internal/models/user.go",
			expected: []string{
				"internal/middleware/auth.go",
				"config/app.yaml",
				"internal/models/user.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFileRefs(tt.content)

			if len(got) != len(tt.expected) {
				t.Errorf("ExtractFileRefs() returned %d refs, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.expected), got, tt.expected)
				return
			}

			for i, path := range got {
				if path != tt.expected[i] {
					t.Errorf("ref[%d] = %q, want %q", i, path, tt.expected[i])
				}
			}
		})
	}
}
