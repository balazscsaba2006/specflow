package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

// getContent resolves input from three sources in priority order:
// 1. --file flag (explicit path)
// 2. Piped stdin (auto-detected via non-TTY)
// 3. $EDITOR (interactive terminal fallback)
func getContent(cmd *cobra.Command, fallback string) (string, error) {
	// Priority 1: --file flag.
	filePath, _ := cmd.Flags().GetString("file")
	if filePath != "" {
		data, err := os.ReadFile(filePath) //nolint:gosec // user-provided path is intentional
		if err != nil {
			return "", fmt.Errorf("reading file %q: %w", filePath, err)
		}
		if strings.TrimSpace(string(data)) == "" {
			return "", fmt.Errorf("file %q is empty", filePath)
		}
		return string(data), nil
	}

	// Priority 2: piped stdin (non-TTY).
	if !term.IsTerminal(int(os.Stdin.Fd())) { //nolint:gosec // stdin fd fits in int on all supported platforms
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		if strings.TrimSpace(string(data)) == "" {
			return "", fmt.Errorf("stdin is empty")
		}
		return string(data), nil
	}

	// Priority 3: interactive editor.
	return openInEditor(fallback)
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
