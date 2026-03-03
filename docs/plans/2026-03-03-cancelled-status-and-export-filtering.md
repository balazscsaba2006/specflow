# Cancelled Status + Export Filtering Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `cancelled` status to all entity types (stories, epics, initiatives) and replace the narrow `--exclude-done` export flag with a generic `--exclude-status` flag that can filter any combination of statuses across all entity types.

**Architecture:** Three-layer change: (1) model constants + transitions, (2) replace `IncludeDone bool` with `ExcludeStatuses map[string]bool` across extract/render/export options, (3) CLI flag + MCP parameter update. A nil map lookup returns `false`, so `opts.ExcludeStatuses[status]` is safe without nil checks.

**Tech Stack:** Go 1.24+, Cobra CLI, mcp-go SDK

---

### Task 1: Add cancelled status to story model

**Files:**
- Modify: `internal/models/story.go:8-51`

**Step 1: Write the failing test**

No separate test file for model constants — validation is tested via store integration tests. Skip test-first for pure constant addition.

**Step 2: Add the constant and update valid list**

In `internal/models/story.go`, add `StoryStatusCancelled` to the const block and `ValidStoryStatuses`:

```go
const (
	StoryStatusDraft      = "draft"
	StoryStatusPlanned    = "planned"
	StoryStatusInProgress = "in_progress"
	StoryStatusVerifying  = "verifying"
	StoryStatusDone       = "done"
	StoryStatusBlocked    = "blocked"
	StoryStatusCancelled  = "cancelled"
)

var ValidStoryStatuses = []string{
	StoryStatusDraft,
	StoryStatusPlanned,
	StoryStatusInProgress,
	StoryStatusVerifying,
	StoryStatusDone,
	StoryStatusBlocked,
	StoryStatusCancelled,
}
```

**Step 3: Update transition map**

Any status can transition TO cancelled. Cancelled can transition to draft or planned (reversible).

```go
var validTransitions = map[string][]string{
	StoryStatusDraft:      {StoryStatusPlanned, StoryStatusBlocked, StoryStatusCancelled},
	StoryStatusPlanned:    {StoryStatusInProgress, StoryStatusBlocked, StoryStatusCancelled},
	StoryStatusInProgress: {StoryStatusVerifying, StoryStatusDone, StoryStatusBlocked, StoryStatusCancelled},
	StoryStatusVerifying:  {StoryStatusDone, StoryStatusInProgress, StoryStatusBlocked, StoryStatusCancelled},
	StoryStatusDone:       {StoryStatusCancelled},
	StoryStatusBlocked:    {StoryStatusDraft, StoryStatusPlanned, StoryStatusInProgress, StoryStatusCancelled},
	StoryStatusCancelled:  {StoryStatusDraft, StoryStatusPlanned},
}
```

**Step 4: Verify compilation**

Run: `cd /Users/csababalazs/Projects/specflow && go build ./...`
Expected: compiles without errors

---

### Task 2: Add cancelled status to epic and initiative models

**Files:**
- Modify: `internal/models/epic.go:8-23`
- Modify: `internal/models/initiative.go:8-21`

**Step 1: Add to epic model**

In `internal/models/epic.go`:

```go
const (
	EpicStatusDraft     = "draft"
	EpicStatusActive    = "active"
	EpicStatusCompleted = "completed"
	EpicStatusOnHold    = "on_hold"
	EpicStatusArchived  = "archived"
	EpicStatusCancelled = "cancelled"
)

var ValidEpicStatuses = []string{
	EpicStatusDraft,
	EpicStatusActive,
	EpicStatusCompleted,
	EpicStatusOnHold,
	EpicStatusArchived,
	EpicStatusCancelled,
}
```

**Step 2: Add to initiative model**

In `internal/models/initiative.go`:

```go
const (
	InitiativeStatusActive    = "active"
	InitiativeStatusCompleted = "completed"
	InitiativeStatusOnHold    = "on_hold"
	InitiativeStatusArchived  = "archived"
	InitiativeStatusCancelled = "cancelled"
)

var ValidInitiativeStatuses = []string{
	InitiativeStatusActive,
	InitiativeStatusCompleted,
	InitiativeStatusOnHold,
	InitiativeStatusArchived,
	InitiativeStatusCancelled,
}
```

**Step 3: Verify compilation**

Run: `cd /Users/csababalazs/Projects/specflow && go build ./...`
Expected: compiles without errors

**Step 4: Commit**

```bash
git add internal/models/story.go internal/models/epic.go internal/models/initiative.go
git commit -m "feat: add cancelled status to story, epic, and initiative models"
```

---

### Task 3: Replace IncludeDone with ExcludeStatuses in option structs

**Files:**
- Modify: `internal/export/node.go:52-57` (RenderOptions)
- Modify: `internal/export/extract.go:12-17` (ExtractOptions)
- Modify: `internal/export/export.go:13-17` (ExportOptions)

**Step 1: Update RenderOptions**

In `internal/export/node.go`, replace `IncludeDone bool` with `ExcludeStatuses map[string]bool`:

```go
// RenderOptions controls renderer behavior.
type RenderOptions struct {
	IncludeBody    bool
	ExcludeStatuses map[string]bool
	Title          string // override document title
}
```

**Step 2: Update ExtractOptions**

In `internal/export/extract.go`:

```go
// ExtractOptions controls what data is included when extracting entities into ExportNodes.
type ExtractOptions struct {
	ExcludeStatuses map[string]bool
	IncludeBody     bool
	Tree            bool // include full subtree (children, docs, decisions)
}
```

**Step 3: Update ExportOptions**

In `internal/export/export.go`:

```go
// ExportOptions controls what data is included in the export.
type ExportOptions struct {
	ExcludeStatuses map[string]bool
	IncludeBody     bool
}
```

**Step 4: Do NOT compile yet** — this will break consumers. Continue to Task 4.

---

### Task 4: Update extraction layer filtering

**Files:**
- Modify: `internal/export/extract.go:86-91,117-128,216-228`

**Step 1: Update ExtractEpicNode story filtering (line 86-89)**

Replace:
```go
if !opts.IncludeDone && st.Status == models.StoryStatusDone {
```
With:
```go
if opts.ExcludeStatuses[st.Status] {
```

**Step 2: Update ExtractStoryNode filtering (line 123-124)**

Replace:
```go
if !opts.IncludeDone && st.Status == models.StoryStatusDone {
	return nil, fmt.Errorf("story %q has status done and include_done is false", slug)
}
```
With:
```go
if opts.ExcludeStatuses[st.Status] {
	return nil, fmt.Errorf("story %q has excluded status %q", slug, st.Status)
}
```

**Step 3: Update extractStandaloneStories filtering (line 222)**

Replace:
```go
if !opts.IncludeDone && st.Status == models.StoryStatusDone {
```
With:
```go
if opts.ExcludeStatuses[st.Status] {
```

**Step 4: Add filtering in ExtractInitiative for cancelled epics (line 43-49)**

When iterating linked epics, skip excluded epics:

```go
for _, epicSlug := range init.Epics {
	epicNode, err := ExtractEpicNode(s, epicSlug, opts)
	if err != nil {
		return nil, fmt.Errorf("extracting epic %q for initiative %q: %w", epicSlug, slug, err)
	}
	if opts.ExcludeStatuses[epicNode.Status] {
		continue
	}
	node.Children = append(node.Children, epicNode)
}
```

**Step 5: Add filtering in extractStandaloneEpics**

In `extractStandaloneEpics` (line 203-213), after extracting the epic node, skip if excluded:

```go
epicNode, epicErr := ExtractEpicNode(s, e.Slug, opts)
if epicErr != nil {
	return fmt.Errorf("extracting epic %q: %w", e.Slug, epicErr)
}
if opts.ExcludeStatuses[epicNode.Status] {
	continue
}
root.Children = append(root.Children, epicNode)
```

**Step 6: Add filtering in extractAllInitiatives**

In `extractAllInitiatives` (line 185-194), skip excluded initiatives:

```go
initNode, initErr := ExtractInitiative(s, init.Slug, opts)
if initErr != nil {
	return nil, fmt.Errorf("extracting initiative %q: %w", init.Slug, initErr)
}
if opts.ExcludeStatuses[initNode.Status] {
	continue
}
root.Children = append(root.Children, initNode)
```

---

### Task 5: Update legacy export.go filtering

**Files:**
- Modify: `internal/export/export.go:59-68,139-141`

**Step 1: Update ExportEpic filtering (line 60-68)**

Replace:
```go
if !opts.IncludeDone {
	filtered := make([]*models.Story, 0, len(stories))
	for _, st := range stories {
		if st.Status != models.StoryStatusDone {
			filtered = append(filtered, st)
		}
	}
	stories = filtered
}
```
With:
```go
if len(opts.ExcludeStatuses) > 0 {
	filtered := make([]*models.Story, 0, len(stories))
	for _, st := range stories {
		if !opts.ExcludeStatuses[st.Status] {
			filtered = append(filtered, st)
		}
	}
	stories = filtered
}
```

**Step 2: Update ExportStory filtering (line 139-141)**

Replace:
```go
if !opts.IncludeDone && st.Status == models.StoryStatusDone {
	return nil, fmt.Errorf("story %q has status done and include_done is false", storySlug)
}
```
With:
```go
if opts.ExcludeStatuses[st.Status] {
	return nil, fmt.Errorf("story %q has excluded status %q", storySlug, st.Status)
}
```

---

### Task 6: Update renderers

**Files:**
- Modify: `internal/export/render_md.go:84-92`
- Modify: `internal/export/render_yaml.go:89-95,153-156`
- Modify: `internal/export/render_html.go:125-149`

**Step 1: Update MarkdownRenderer.renderChildren (line 86-88)**

Replace:
```go
if !opts.IncludeDone && child.Type == NodeStory && child.Status == "done" {
	continue
}
```
With:
```go
if opts.ExcludeStatuses[child.Status] {
	continue
}
```

**Step 2: Update YAMLRenderer.renderLegacyEpic (line 93-95)**

Replace:
```go
if !opts.IncludeDone && child.Status == "done" {
	continue
}
```
With:
```go
if opts.ExcludeStatuses[child.Status] {
	continue
}
```

**Step 3: Update YAMLRenderer.nodeToYAML (line 154-156)**

Replace:
```go
if !opts.IncludeDone && c.Type == NodeStory && c.Status == "done" {
	continue
}
```
With:
```go
if opts.ExcludeStatuses[c.Status] {
	continue
}
```

**Step 4: Update HTML status badges (line 126-127)**

Add "cancelled" to the statuses list:
```go
statuses := []string{"done", "active", "in_progress", "planned", "draft", "on_hold", "blocked",
	"completed", "verifying", "cancelled"}
```

**Step 5: Update badgeClass (line 138-149)**

Add cancelled case — use the same styling as blocked (muted/negative):
```go
case "blocked", "cancelled":
	return "badge-blocked"
```

---

### Task 7: Update CLI export command

**Files:**
- Modify: `cmd/specflow/export.go:28-59`

**Step 1: Replace the flag**

Remove line 33 (`--exclude-done`), add `--exclude-status`:

```go
cmd.Flags().StringSlice("exclude-status", nil, "Skip entities with these statuses (comma-separated, e.g. done,cancelled)")

// Deprecated alias for backwards compatibility.
cmd.Flags().Bool("exclude-done", false, "Skip stories with status done (deprecated: use --exclude-status done)")
_ = cmd.Flags().MarkHidden("exclude-done")
```

**Step 2: Update runExport to build ExcludeStatuses map**

Replace lines 44 and 50-59 with:

```go
excludeStatusSlice, _ := cmd.Flags().GetStringSlice("exclude-status")
excludeDone, _ := cmd.Flags().GetBool("exclude-done")

// Backwards compat: --exclude-done maps to --exclude-status done.
if excludeDone && len(excludeStatusSlice) == 0 {
	excludeStatusSlice = []string{"done"}
}

excludeStatuses := make(map[string]bool, len(excludeStatusSlice))
for _, s := range excludeStatusSlice {
	excludeStatuses[s] = true
}

extOpts := export.ExtractOptions{
	ExcludeStatuses: excludeStatuses,
	IncludeBody:     !noBody,
	Tree:            tree || all,
}

renderOpts := export.RenderOptions{
	IncludeBody:     !noBody,
	ExcludeStatuses: excludeStatuses,
}
```

---

### Task 8: Update MCP export tool

**Files:**
- Modify: `internal/mcp/tools_read.go:77-87,1644-1671`

**Step 1: Update exportInput struct**

Add `ExcludeStatus` field, keep `IncludeDone` for backwards compat:

```go
type exportInput struct {
	Epic          string   `json:"epic,omitempty" jsonschema:"Epic slug to export"`
	Story         string   `json:"story,omitempty" jsonschema:"Standalone story slug to export"`
	Initiative    string   `json:"initiative,omitempty" jsonschema:"Initiative slug to export"`
	Doc           string   `json:"doc,omitempty" jsonschema:"Document slug to export"`
	Decision      string   `json:"decision,omitempty" jsonschema:"Decision slug to export"`
	Format        string   `json:"format,omitempty" jsonschema:"Output format: yaml (default), md, html"`
	Tree          *bool    `json:"tree,omitempty" jsonschema:"Include full subtree with children, docs, decisions (default false)"`
	IncludeDone   *bool    `json:"include_done,omitempty" jsonschema:"Include stories with status done (default true)"`
	IncludeBody   *bool    `json:"include_body,omitempty" jsonschema:"Include markdown body content (default true)"`
	ExcludeStatus []string `json:"exclude_status,omitempty" jsonschema:"Statuses to exclude from export (e.g. done, cancelled)"`
}
```

**Step 2: Update handleExport to build ExcludeStatuses map**

Replace the options construction in handleExport:

```go
func (s *Server) handleExport(_ context.Context, _ *mcp.CallToolRequest, input exportInput) (*mcp.CallToolResult, any, error) {
	if err := validateExportInput(input); err != nil {
		return errResult(err.Error()), nil, nil
	}

	excludeStatuses := buildExcludeStatuses(input.ExcludeStatus, input.IncludeDone)

	extOpts := export.ExtractOptions{
		ExcludeStatuses: excludeStatuses,
		IncludeBody:     boolDefault(input.IncludeBody, true),
		Tree:            boolDefault(input.Tree, false),
	}

	node, err := s.extractExportEntity(input, extOpts)
	if err != nil {
		return errResultf("extracting entity: %v", err), nil, nil
	}

	format := input.Format
	if format == "" {
		format = "yaml"
	}

	renderOpts := export.RenderOptions{
		ExcludeStatuses: excludeStatuses,
		IncludeBody:     boolDefault(input.IncludeBody, true),
	}

	return s.renderExportResult(node, format, renderOpts)
}

// buildExcludeStatuses constructs the ExcludeStatuses map from the new exclude_status
// parameter and the legacy include_done parameter for backwards compatibility.
func buildExcludeStatuses(excludeStatus []string, includeDone *bool) map[string]bool {
	result := make(map[string]bool, len(excludeStatus))
	for _, s := range excludeStatus {
		result[s] = true
	}
	// Backwards compat: include_done=false adds "done" to excludes (if not already set via exclude_status).
	if includeDone != nil && !*includeDone && !result["done"] {
		result["done"] = true
	}
	return result
}
```

---

### Task 9: Update all tests

**Files:**
- Modify: `internal/export/extract_test.go`
- Modify: `internal/export/export_test.go`
- Modify: `internal/export/render_test.go`

**Step 1: Replace all `IncludeDone: true` with `ExcludeStatuses: nil`**

In all three test files, find-and-replace:
- `IncludeDone: true` → remove (nil map is the default, means include everything)
- `IncludeDone: false` → `ExcludeStatuses: map[string]bool{"done": true}`

Specifically in `extract_test.go`, update all `ExtractOptions` constructors:

```go
// Before:
opts: ExtractOptions{IncludeDone: true, IncludeBody: true, Tree: false},
// After:
opts: ExtractOptions{IncludeBody: true, Tree: false},

// Before:
opts: ExtractOptions{IncludeDone: false, IncludeBody: true, Tree: true},
// After:
opts: ExtractOptions{ExcludeStatuses: map[string]bool{"done": true}, IncludeBody: true, Tree: true},
```

Same pattern in `export_test.go` for `ExportOptions` and `render_test.go` for `RenderOptions`.

**Step 2: Add cancelled story filtering test in extract_test.go**

Add a new test case in `TestExtractEpicNode`:

```go
{
	name: "epic tree with ExcludeStatuses filters cancelled stories",
	setup: func(t *testing.T, s *store.Store) {
		createEpic(t, s, "cancel-epic", "Cancel", "beta", "", nil)
		createStory(t, s, "active-s", "Active", "cancel-epic", "planned", "high", "", nil, nil)
		createStory(t, s, "cancel-s", "Cancelled", "cancel-epic", "cancelled", "low", "", nil, nil)
	},
	slug: "cancel-epic",
	opts: ExtractOptions{ExcludeStatuses: map[string]bool{"cancelled": true}, IncludeBody: true, Tree: true},
	check: func(t *testing.T, node *ExportNode) {
		if len(node.Children) != 1 {
			t.Fatalf("Children = %d, want 1 (cancelled filtered)", len(node.Children))
		}
		if node.Children[0].Slug != "active-s" {
			t.Errorf("Children[0].Slug = %q, want active-s", node.Children[0].Slug)
		}
	},
},
```

**Step 3: Add multi-status exclusion test**

```go
{
	name: "epic tree excludes both done and cancelled",
	setup: func(t *testing.T, s *store.Store) {
		createEpic(t, s, "multi-epic", "Multi", "beta", "", nil)
		createStory(t, s, "active-m", "Active", "multi-epic", "planned", "high", "", nil, nil)
		createStory(t, s, "done-m", "Done", "multi-epic", "done", "low", "", nil, nil)
		createStory(t, s, "cancel-m", "Cancelled", "multi-epic", "cancelled", "low", "", nil, nil)
	},
	slug: "multi-epic",
	opts: ExtractOptions{ExcludeStatuses: map[string]bool{"done": true, "cancelled": true}, IncludeBody: true, Tree: true},
	check: func(t *testing.T, node *ExportNode) {
		if len(node.Children) != 1 {
			t.Fatalf("Children = %d, want 1 (done+cancelled filtered)", len(node.Children))
		}
		if node.Children[0].Slug != "active-m" {
			t.Errorf("Children[0].Slug = %q, want active-m", node.Children[0].Slug)
		}
	},
},
```

**Step 4: Add cancelled story rendering test in render_test.go**

Add to `makeTestNode()` a cancelled story child, or add a new test case:

```go
{
	name: "excludes cancelled stories",
	node: func() *ExportNode {
		n := makeTestNode()
		n.Children = append(n.Children, &ExportNode{
			Type:     NodeStory,
			Slug:     "story-c",
			Title:    "Story C",
			Status:   "cancelled",
			Priority: "low",
		})
		return n
	}(),
	opts: RenderOptions{IncludeBody: true, ExcludeStatuses: map[string]bool{"cancelled": true}},
	contains: []string{
		"## Story A",
		"## Story B",
	},
	excludes: []string{
		"## Story C",
	},
},
```

**Step 5: Run tests**

Run: `cd /Users/csababalazs/Projects/specflow && go test ./internal/export/... -count=1`
Expected: all tests pass

**Step 6: Commit**

```bash
git add internal/export/ cmd/specflow/export.go internal/mcp/tools_read.go
git commit -m "feat: replace --exclude-done with --exclude-status for generic status filtering"
```

---

### Task 10: Run full test suite and lint

**Step 1: Run all tests**

Run: `cd /Users/csababalazs/Projects/specflow && go test ./... -count=1`
Expected: all tests pass

**Step 2: Run linter**

Run: `cd /Users/csababalazs/Projects/specflow && ~/go/bin/golangci-lint run`
Expected: no new warnings

**Step 3: Build binary**

Run: `cd /Users/csababalazs/Projects/specflow && go build -o specflow ./cmd/specflow`
Expected: binary builds successfully

---

### Task 11: Update documentation

**Files:**
- Modify: `docs/mcp-tools.md` — update sf_export tool parameters
- Modify: `docs/architecture.md` — add cancelled to status values for all entity types
- Modify: `docs/cli-reference.md` — update export command flags
- Modify: `README.md` — update if export section exists

**Step 1: Update each doc**

Key changes:
- **mcp-tools.md**: Add `exclude_status` parameter to sf_export, note `include_done` is deprecated
- **architecture.md**: Add `cancelled` to story/epic/initiative status values, document transitions
- **cli-reference.md**: Replace `--exclude-done` with `--exclude-status` in export command docs

**Step 2: Commit**

```bash
git add docs/ README.md
git commit -m "docs: update docs for cancelled status and --exclude-status flag"
```
