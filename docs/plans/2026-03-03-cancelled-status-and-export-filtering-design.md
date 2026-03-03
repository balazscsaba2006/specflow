# Cancelled Status + Export Filtering

## Problem

specflow has no `cancelled` status for any entity type. Stories, epics, and initiatives that are abandoned have no proper lifecycle state. Users resort to editing YAML frontmatter directly (as seen in the b2b-reporting-app project with 6 cancelled stories).

Additionally, export filtering is hardcoded to a single boolean (`--exclude-done` / `IncludeDone`) that only filters stories with status `done`. There's no way to exclude cancelled items, and filtering doesn't apply to epics or initiatives.

## Decision

### 1. Add `cancelled` status to all entity types

**Stories:**
- New constant: `StoryStatusCancelled = "cancelled"`
- Transitions in: any active status -> cancelled (reachable from draft, planned, in_progress, verifying, blocked)
- Transitions out: cancelled -> draft, cancelled -> planned (reversible)
- Added to `ValidStoryStatuses`

**Epics:**
- New constant: `EpicStatusCancelled = "cancelled"`
- Added to `ValidEpicStatuses`
- Reversible: cancelled -> draft, cancelled -> active

**Initiatives:**
- New constant: `InitiativeStatusCancelled = "cancelled"`
- Added to `ValidInitiativeStatuses`
- Reversible: cancelled -> active, cancelled -> on_hold

### 2. Replace `--exclude-done` with `--exclude-status`

**CLI:**
```
specflow export my-epic --tree --exclude-status done,cancelled
```
- `--exclude-done` retained as hidden deprecated alias (maps to `--exclude-status done`)

**Internal options:**
- `IncludeDone bool` replaced with `ExcludeStatuses map[string]bool` in both `ExtractOptions` and `RenderOptions`

**Filtering scope:**
- Applied to all node types: stories, epics, initiatives
- When an epic/initiative is excluded, its entire subtree is skipped (no point exporting children of a cancelled epic)

**MCP tool:**
- New parameter: `exclude_status` (string array)
- Backwards compat: `include_done=false` with empty `exclude_status` translates to `exclude_status: ["done"]`

## Files Changed

| Layer | Files | Change |
|-------|-------|--------|
| Models | `internal/models/story.go` | Add cancelled constant, update transitions, update valid list |
| Models | `internal/models/epic.go` | Add cancelled constant, update valid list |
| Models | `internal/models/initiative.go` | Add cancelled constant, update valid list |
| Export | `internal/export/extract.go` | Replace `IncludeDone` with `ExcludeStatuses`, filter all node types |
| Export | `internal/export/render_md.go` | Replace `IncludeDone` with `ExcludeStatuses` |
| Export | `internal/export/render_yaml.go` | Replace `IncludeDone` with `ExcludeStatuses` |
| Export | `internal/export/render_html.go` | Replace `IncludeDone` with `ExcludeStatuses` |
| CLI | `cmd/specflow/export.go` | New `--exclude-status` flag, deprecate `--exclude-done` |
| MCP | `internal/mcp/tools_read.go` | New `exclude_status` param, backwards compat for `include_done` |
| Tests | `internal/export/extract_test.go` | Update tests for new filtering |
| Tests | `internal/export/render_*_test.go` | Update tests for new filtering |
| Docs | Multiple | mcp-tools.md, architecture.md, cli-reference.md, README.md |

## What-If List

- **New statuses added later:** `--exclude-status` accepts any string, no code changes needed
- **Cancelled epic with in-progress stories:** Epic + all children excluded from export. Stories still exist in specflow.
- **MCP clients using `include_done`:** Backwards compat translation keeps them working
- **Status validation on existing files:** Files with `status: cancelled` already in YAML will now pass validation instead of being silently accepted
