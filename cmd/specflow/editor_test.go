package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("file", "", "")
	return cmd
}

func TestGetContent_FileFlag(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "input.md")
	if err := os.WriteFile(tmp, []byte("---\ntitle: from-file\n---\n# Body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCmd()
	if err := cmd.Flags().Set("file", tmp); err != nil {
		t.Fatal(err)
	}

	got, err := getContent(cmd, "fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "---\ntitle: from-file\n---\n# Body\n" {
		t.Errorf("got %q, want file content", got)
	}
}

func TestGetContent_FileFlagEmptyFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "empty.md")
	if err := os.WriteFile(tmp, []byte("   \n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCmd()
	if err := cmd.Flags().Set("file", tmp); err != nil {
		t.Fatal(err)
	}

	_, err := getContent(cmd, "fallback")
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestGetContent_FileFlagMissing(t *testing.T) {
	cmd := newTestCmd()
	if err := cmd.Flags().Set("file", "/nonexistent/path.md"); err != nil {
		t.Fatal(err)
	}

	_, err := getContent(cmd, "fallback")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestGetContent_PipedStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	content := "---\ntitle: from-stdin\n---\n# Piped\n"
	_, writeErr := w.WriteString(content)
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	cmd := newTestCmd()
	got, err := getContent(cmd, "fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestGetContent_PipedStdinEmpty(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	cmd := newTestCmd()
	_, err = getContent(cmd, "fallback")
	if err == nil {
		t.Fatal("expected error for empty stdin")
	}
}

func TestGetContent_FileFlagPrecedenceOverStdin(t *testing.T) {
	// Set up piped stdin with different content.
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatal(pipeErr)
	}
	_, writeErr := w.WriteString("stdin-content")
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	// Set up --file with different content.
	tmp := filepath.Join(t.TempDir(), "input.md")
	require(t, os.WriteFile(tmp, []byte("file-content"), 0o644))

	cmd := newTestCmd()
	require(t, cmd.Flags().Set("file", tmp))

	got, err := getContent(cmd, "fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "file-content" {
		t.Errorf("got %q, want %q (--file should win over stdin)", got, "file-content")
	}
}

func require(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
