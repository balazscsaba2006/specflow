package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// CurrentRef returns the current HEAD commit hash (short form).
func CurrentRef() (string, error) {
	out, err := run("rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting current ref: %w", err)
	}
	return out, nil
}

// Diff returns the git diff between two refs. If "to" is empty, diffs against HEAD.
func Diff(from, to string) (string, error) {
	if to == "" {
		to = "HEAD"
	}
	out, err := runRaw("diff", from+".."+to)
	if err != nil {
		return "", fmt.Errorf("getting diff %s..%s: %w", from, to, err)
	}
	return out, nil
}

// FileChange represents a single changed file with its action.
type FileChange struct {
	Path   string
	Action string // A=added, M=modified, D=deleted, R=renamed, C=copied
}

// FileChanges returns the list of files changed between two refs.
// If "to" is empty, compares against HEAD.
func FileChanges(from, to string) ([]FileChange, error) {
	if to == "" {
		to = "HEAD"
	}
	out, err := run("diff", "--name-status", from+".."+to)
	if err != nil {
		return nil, fmt.Errorf("getting file changes %s..%s: %w", from, to, err)
	}
	if out == "" {
		return nil, nil
	}

	var changes []FileChange
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		changes = append(changes, FileChange{
			Action: parts[0],
			Path:   parts[1],
		})
	}
	return changes, nil
}

// Status returns the short-form git status output.
func Status() (string, error) {
	out, err := run("status", "--short")
	if err != nil {
		return "", fmt.Errorf("getting git status: %w", err)
	}
	return out, nil
}

// run executes a git command and returns trimmed stdout.
func run(args ...string) (string, error) {
	out, err := runRaw(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// runRaw executes a git command and returns raw stdout.
func runRaw(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return "", err
	}
	return string(out), nil
}
