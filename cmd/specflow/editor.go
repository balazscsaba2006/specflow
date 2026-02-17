package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/config"
)

// getEditor returns the editor command to use, checking:
// 1. Global specflow config (editor field)
// 2. $VISUAL env var
// 3. $EDITOR env var
// 4. Falls back to "vi"
func getEditor() string {
	gc, err := config.LoadGlobal()
	if err == nil && gc.Editor != "" {
		return gc.Editor
	}
	if v := os.Getenv("VISUAL"); v != "" {
		return v
	}
	if v := os.Getenv("EDITOR"); v != "" {
		return v
	}
	return "vi"
}

// openInEditor writes content to a temp file, opens it in $EDITOR,
// and returns the edited content after the editor closes.
func openInEditor(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "specflow-*.md")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) //nolint:gosec // temp file created by us

	_, writeErr := tmpFile.WriteString(content)
	tmpFile.Close()
	if writeErr != nil {
		return "", fmt.Errorf("writing temp file: %w", writeErr)
	}

	editor := getEditor()
	parts := strings.Fields(editor)
	parts = append(parts, tmpPath)

	editorCmd := exec.Command(parts[0], parts[1:]...) //nolint:gosec // editor is user-configured
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	runErr := editorCmd.Run()
	if runErr != nil {
		return "", fmt.Errorf("editor exited with error: %w", runErr)
	}

	edited, err := os.ReadFile(tmpPath) //nolint:gosec // reading back our own temp file
	if err != nil {
		return "", fmt.Errorf("reading edited file: %w", err)
	}

	return string(edited), nil
}
