package context

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/balazscsaba2006/specflow/internal/config"
	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
)

var update = flag.Bool("update", false, "update golden files")

// setupFullProject creates a .specflow/ directory with a complete hierarchy:
// initiative -> epic -> stories (one completed, one target), doc, decision, plan.
func setupFullProject(t *testing.T) (*store.Store, config.Config) {
	t.Helper()

	root := t.TempDir()
	specflowDir := filepath.Join(root, ".specflow")
	s := store.New(specflowDir)
	if err := s.Init(); err != nil {
		t.Fatalf("init store: %v", err)
	}

	// Write a CLAUDE.md in the project root.
	claudeMD := "# Project Conventions\n\n- Use Go 1.24\n- Follow standard Go conventions\n"
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(claudeMD), 0o600); err != nil {
		t.Fatalf("writing CLAUDE.md: %v", err)
	}

	// Create initiative.
	initiative := &models.Initiative{
		Slug:            "platform-mvp",
		Title:           "Platform MVP",
		Status:          models.InitiativeStatusActive,
		Goal:            "Ship the initial platform version with auth and core API",
		SuccessCriteria: []string{"All auth stories done", "API documented"},
		OpenQuestions:   []string{"Which OAuth provider to use?"},
	}
	if err := s.CreateInitiative(initiative); err != nil {
		t.Fatalf("creating initiative: %v", err)
	}

	// Create epic.
	epic := &models.Epic{
		Slug:          "auth-system",
		Title:         "Authentication System",
		Status:        models.EpicStatusActive,
		Initiative:    "platform-mvp",
		OpenQuestions: []string{"Should we support refresh tokens in v1?"},
		Phases: []models.Phase{
			{Label: "Phase 1", Stories: []string{"api-key-store", "jwt-middleware"}},
		},
	}
	if err := s.CreateEpic(epic); err != nil {
		t.Fatalf("creating epic: %v", err)
	}

	// Create completed story.
	completedStory := &models.Story{
		Slug:        "api-key-store",
		Title:       "API Key Storage",
		Status:      models.StoryStatusDraft,
		Priority:    models.PriorityHigh,
		Epic:        "auth-system",
		Assumptions: []string{"Keys are stored hashed, not plaintext"},
	}
	if err := s.CreateStory(completedStory); err != nil {
		t.Fatalf("creating completed story: %v", err)
	}
	// Transition to done: draft -> planned -> in_progress -> done.
	for _, status := range []string{models.StoryStatusPlanned, models.StoryStatusInProgress, models.StoryStatusDone} {
		if err := s.UpdateStoryStatus("api-key-store", "auth-system", status); err != nil {
			t.Fatalf("transitioning api-key-store to %s: %v", status, err)
		}
	}

	// Create target story.
	targetStory := &models.Story{
		Slug:          "jwt-middleware",
		Title:         "JWT Authentication Middleware",
		Status:        models.StoryStatusDraft,
		Priority:      models.PriorityCritical,
		Epic:          "auth-system",
		Acceptance:    []string{"Validate JWT tokens on protected endpoints", "Return 401 for invalid tokens"},
		DocRefs:       []string{"auth-prd"},
		OpenQuestions: []string{"Which JWT library to use?"},
		Labels:        []string{"auth", "middleware"},
	}
	if err := s.CreateStory(targetStory); err != nil {
		t.Fatalf("creating target story: %v", err)
	}
	if err := s.UpdateStoryStatus("jwt-middleware", "auth-system", models.StoryStatusPlanned); err != nil {
		t.Fatalf("transitioning jwt-middleware to planned: %v", err)
	}

	// Create doc under epic.
	doc := &models.Document{
		Slug:   "auth-prd",
		Type:   models.DocTypePRD,
		Title:  "Authentication PRD",
		Status: models.DocStatusApproved,
		Epic:   "auth-system",
		Body:   "## Problem\n\nWe need authentication for the API.\n\n## Requirements\n\n- JWT-based auth\n- API key support",
	}
	if err := s.CreateDoc(doc); err != nil {
		t.Fatalf("creating doc: %v", err)
	}

	// Create decision.
	dec := &models.Decision{
		Slug:        "use-jwt",
		Title:       "Use JWT for API Authentication",
		Status:      models.DecisionStatusAccepted,
		Date:        "2025-01-10",
		ContextRefs: []string{"auth-system"},
		Body:        "## Context\n\nNeed stateless auth.\n\n## Decision\n\nUse JWT with RS256.",
	}
	if err := s.CreateDecision(dec); err != nil {
		t.Fatalf("creating decision: %v", err)
	}

	// Create plan.
	plan := &models.Plan{
		Story:  "jwt-middleware",
		Status: models.PlanStatusApproved,
		Body:   "## Steps\n\n1. Create `internal/middleware/jwt.go`\n2. Add token validation logic\n3. Write tests in `internal/middleware/jwt_test.go`",
	}
	if err := s.SavePlan(plan, "jwt-middleware"); err != nil {
		t.Fatalf("saving plan: %v", err)
	}

	cfg := config.Config{
		Mode:            "careful",
		ConventionsFile: "CLAUDE.md",
		DefaultPriority: "medium",
	}

	if err := config.Save(s.ConfigFile(), cfg); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	return s, cfg
}

// setupMinimalProject creates a .specflow/ with a standalone story (no epic, no plan).
func setupMinimalProject(t *testing.T) (*store.Store, config.Config) {
	t.Helper()

	root := t.TempDir()
	specflowDir := filepath.Join(root, ".specflow")
	s := store.New(specflowDir)
	if err := s.Init(); err != nil {
		t.Fatalf("init store: %v", err)
	}

	story := &models.Story{
		Slug:     "quick-fix",
		Title:    "Fix Login Bug",
		Status:   models.StoryStatusDraft,
		Priority: models.PriorityMedium,
	}
	if err := s.CreateStory(story); err != nil {
		t.Fatalf("creating story: %v", err)
	}

	cfg := config.DefaultConfig()

	return s, cfg
}

func TestBuildContext_FullProject(t *testing.T) {
	s, cfg := setupFullProject(t)

	builder := New(s, cfg)
	result, err := builder.Build("jwt-middleware")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	compareGolden(t, result, filepath.Join("testdata", "golden", "context-full.md"))
}

func TestBuildContext_MinimalProject(t *testing.T) {
	s, cfg := setupMinimalProject(t)

	builder := New(s, cfg)
	result, err := builder.Build("quick-fix")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	compareGolden(t, result, filepath.Join("testdata", "golden", "context-minimal.md"))
}

func TestBuildContext_StoryNotFound(t *testing.T) {
	s, cfg := setupMinimalProject(t)

	builder := New(s, cfg)
	_, err := builder.Build("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent story")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

// compareGolden compares the result against a golden file, updating it if -update flag is set.
func compareGolden(t *testing.T, result, goldenFile string) {
	t.Helper()

	if *update {
		mkdirErr := os.MkdirAll(filepath.Dir(goldenFile), 0o750)
		if mkdirErr != nil {
			t.Fatalf("creating golden dir: %v", mkdirErr)
		}
		writeErr := os.WriteFile(goldenFile, []byte(result), 0o600)
		if writeErr != nil {
			t.Fatalf("writing golden file: %v", writeErr)
		}
		t.Log("Updated golden file:", goldenFile)
		return
	}

	expected, readErr := os.ReadFile(goldenFile)
	if readErr != nil {
		t.Fatalf("Reading golden file (run with -update to create): %v", readErr)
	}

	normalizedResult := normalizeDynamic(result)
	normalizedExpected := normalizeDynamic(string(expected))

	if normalizedResult != normalizedExpected {
		t.Errorf("Context output does not match golden file.\nGot:\n%s\n\nExpected:\n%s", result, string(expected))
	}
}

// normalizeDynamic replaces dynamic values (ULIDs, timestamps) with placeholders
// so golden file comparison works across runs.
func normalizeDynamic(s string) string {
	// Replace ULID-based IDs (e.g., s_01XXXXXXXXXXXXXXXXXXXXXXXXXX).
	s = ulidPattern.ReplaceAllString(s, "<ID>")
	// Replace ISO timestamps (e.g., 2025-01-15T10:00:00Z or 2025-01-15 10:00:00).
	s = timestampPattern.ReplaceAllString(s, "<TIMESTAMP>")
	return s
}

var (
	ulidPattern      = regexp.MustCompile(`[a-z]+_[0-9A-HJKMNP-TV-Z]{26}`)
	timestampPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}Z?`)
)
