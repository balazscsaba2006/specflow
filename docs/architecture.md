# Architecture

Reference documentation for the specflow system architecture, storage format, data models, and key design decisions.

---

## 1. System Overview

specflow is a single Go binary that operates in two modes: CLI mode for human interaction, and MCP server mode for Claude Code integration. Both modes share the same core libraries and filesystem store.

The system has zero AI logic. It manages structured state, assembles context, and tracks progress. All intelligence comes from Claude Code interacting through MCP tools.

### Architecture Diagram

```
+------------------------------------------------------+
|                    Human (CLI)                       |
|   specflow epic new, specflow status, etc.           |
+---------------+--------------------------------------+
                |
                v
+------------------------------------------------------+
|              specflow (Go binary)                    |
|                                                      |
|   CLI mode          |    MCP server mode (stdio)     |
|   (human ops)       |    (Claude Code ops)           |
|                     |                                |
|   +-----------------+----------------------------+   |
|   |            Shared Core                       |   |
|   |  Store - Context Builder - Git - HardQ       |   |
|   +----------------------------------------------+   |
|                     |                                |
|   +----------------------------------------------+   |
|   |        Filesystem Store (.specflow/)         |   |
|   +----------------------------------------------+   |
+------------------------------------------------------+
                      | MCP (stdio)
                      v
+------------------------------------------------------+
|                 Claude Code                          |
|                                                      |
|  Reads context -> implements code -> writes results  |
+------------------------------------------------------+
```

**CLI mode** (`specflow [command]`) -- Human creates and manages artifacts, views status, runs reports.

**MCP mode** (`specflow mcp`) -- Claude Code reads context, writes execution results, and updates story state over stdio.

### Entity Hierarchy

```
Project (.specflow/)
  +-- Initiative (optional -- groups epics toward a strategic goal)
       +-- Epic (optional -- a shippable feature/capability)
            +-- Story (the atomic work unit)
```

Everything is optional upward. A story can exist standalone, under an epic, or under a full initiative > epic > story hierarchy.

---

## 2. Package Structure

All non-exported packages live under `internal/`.

| Package | Purpose |
|---------|---------|
| `internal/models` | Pure data structs for all entity types (Initiative, Epic, Story, Document, Decision, Plan, Execution, Verification, LogEntry). Includes status constants, valid status lists, and story status transition validation. No I/O. |
| `internal/store` | Filesystem CRUD for all entity types. Owns the `.specflow/` directory layout. Provides path helpers, frontmatter parsing/writing, and the activity log. All methods operate on model structs. |
| `internal/config` | Config loading from two sources: project-level (`.specflow/config.yaml`) and global (`~/.specflow/config.yaml`). Merges with project config taking precedence. |
| `internal/context` | Context builder -- the core value proposition. Assembles 6-layer context documents from project state for Claude Code consumption. Includes file reference resolution from plans and pattern exemplars. |
| `internal/git` | Git operations via `os/exec` shell-outs. Provides `CurrentRef`, `Diff`, `Status`, and `FileChanges`. No git library dependency. |
| `internal/mcp` | MCP stdio server using `mark3labs/mcp-go`. Registers all `sf_*` tools with behavioral descriptions. Splits handlers into read tools (`tools_read.go`) and write tools (`tools_write.go`). |
| `internal/hardq` | Hard questions template engine. Returns deterministic, template-based questions per entity type (initiative, epic, story, PRD, tech spec, API spec, design spec, ADR, one-pager). No AI involved. |
| `internal/ui` | Terminal output rendering. Tables via lipgloss, markdown rendering via glamour, progress bars, and colored status badges. |

### Entry Points

| File | Role |
|------|------|
| `cmd/specflow/main.go` | Cobra root command setup. No business logic. |
| `cmd/specflow/mcp.go` | Starts the MCP server on stdio. |
| `cmd/specflow/*.go` | One file per CLI command group (initiative, epic, story, doc, decision, status, template, etc.). |

---

## 3. Storage Format

### Frontmatter + Markdown

All artifacts (initiatives, epics, stories, documents, decisions, plans, verifications) use the same format: YAML frontmatter delimited by `---`, followed by a markdown body.

```
---
id: s_01JMXYZ345678
slug: 001-jwt-middleware
title: "Create JWT Auth Middleware"
status: planned
---
# Create JWT Auth Middleware

Implementation details...
```

- **Frontmatter**: structured, machine-readable metadata parsed into Go structs.
- **Body**: human-written content (descriptions, specs, plans).
- Parsed with `adrg/frontmatter`. Roundtrip preserves body formatting.

### Execution Metadata

Executions use pure YAML (`meta.yaml`) rather than frontmatter+markdown, since they are machine-generated with no human-authored body.

### ID Format

All entity IDs use a prefix + ULID scheme:

| Prefix | Entity |
|--------|--------|
| `i_` | Initiative |
| `e_` | Epic |
| `s_` | Story |
| `d_` | Document |
| `dec_` | Decision |
| `p_` | Plan |
| `x_` | Execution |
| `v_` | Verification |

ULIDs provide time-sortability without a sequence counter. IDs are immutable once assigned. Slugs are the human-readable identifier and can change; IDs cannot.

### Directory Layout

```
.specflow/
+-- config.yaml                              # Project configuration
+-- log.jsonl                                # Activity log (append-only JSONL)
+-- templates/                               # Per-project template overrides
|   +-- initiative.md                        # Override initiative template
|   +-- epic.md                              # Override epic template
|   +-- story.md                             # Override story template (careful mode)
|   +-- story_fast.md                        # Override story template (fast mode)
|   +-- decision.md                          # Override decision template
|   +-- doc_prd.md                           # Override PRD template
|   +-- doc_tech-spec.md                     # Override tech spec template
|   +-- doc_api-spec.md                      # Override API spec template
|   +-- doc_design-spec.md                   # Override design spec template
|   +-- doc_adr.md                           # Override ADR template
|   +-- doc_one-pager.md                     # Override one-pager template
|   +-- doc_generic.md                       # Override fallback doc template
|   +-- skill.md                             # Override Claude Code skill
+-- initiatives/
|   +-- {slug}/
|       +-- initiative.md                    # Frontmatter + description
+-- epics/
|   +-- {slug}/
|       +-- epic.md                          # Frontmatter + description
|       +-- docs/                            # Specs/PRDs/ADRs scoped to this epic
|       |   +-- prd.md
|       |   +-- tech-spec.md
|       |   +-- adr-001-some-decision.md
|       +-- stories/
|           +-- 001-create-model.md
|           +-- 002-create-repo.md
+-- stories/                                 # Standalone stories (no epic)
|   +-- fix-timezone-bug.md
|   +-- upgrade-dependency.md
+-- docs/                                    # Project-level docs (no epic scope)
|   +-- adr-001-multi-tenancy.md
|   +-- api-spec-v1.md
+-- decisions/                               # Lightweight project-level decision log
|   +-- 001-use-go-for-cli.md
|   +-- 002-filesystem-over-sqlite.md
+-- executions/                              # Flat, indexed by story slug
|   +-- {story-slug}/
|       +-- latest/
|       |   +-- plan.md                      # Implementation plan (latest)
|       +-- {exec-id}/
|           +-- verification.md              # Verification findings
|           +-- meta.yaml                    # Git refs, timestamps, status
+-- archive/                                 # Archived (completed) items
    +-- initiatives/
    |   +-- {slug}/
    |       +-- initiative.md               # Compacted (frontmatter-only)
    +-- epics/
    |   +-- {slug}/
    |       +-- epic.md                      # Compacted (frontmatter-only)
    |       +-- docs/                        # Docs preserved as-is
    |       +-- stories/
    |           +-- {story-slug}.md          # Compacted (frontmatter-only)
    +-- stories/                             # Archived standalone stories
    |   +-- {slug}.md                        # Compacted (frontmatter-only)
    +-- executions/
        +-- {story-slug}/
            +-- {exec-id}/
                +-- verification.md
                +-- meta.yaml
```

### Path Resolution (store.go)

The `Store` struct holds the root `.specflow/` path and exposes typed path helpers for every entity location:

- `StoryFile(slug, epicSlug)` -- returns the path under `epics/{epic}/stories/` if an epic is provided, otherwise under `stories/`.
- `DocFile(slug, epicSlug)` -- returns the path under `epics/{epic}/docs/` if an epic is provided, otherwise under `docs/`.
- `ExecutionDir(storySlug, execID)` -- returns `executions/{storySlug}/{execID}/`.
- `PlanFile`, `VerificationFile`, `ExecutionMetaFile` -- nested under the execution directory.

---

## 4. Data Models

All model structs live in `internal/models/`. Each entity has YAML struct tags for frontmatter serialization and a `Body string` field (tagged `yaml:"-"`) for the markdown content that is not included in frontmatter.

### Initiative

Groups epics toward a strategic goal. Optional -- epics can exist without one.

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | `i_` + ULID |
| `slug` | string | URL-safe identifier |
| `title` | string | Display name |
| `status` | string | Current status |
| `epics` | []string | Ordered list of epic slugs |
| `goal` | string | Strategic goal statement |
| `success_criteria` | []string | Measurable success conditions |
| `open_questions` | []string | Unresolved questions |
| `resolved_questions` | []ResolvedQuestion | Resolved Q&A pairs (question + answer) |

**Statuses**: `active`, `completed`, `on_hold`, `cancelled`, `archived`

### Epic

A shippable feature or capability. Contains phases that group stories.

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | `e_` + ULID |
| `slug` | string | URL-safe identifier |
| `title` | string | Display name |
| `status` | string | Current status |
| `initiative` | string | Parent initiative slug (optional) |
| `phases` | []Phase | Ordered phases, each with a label and story slug list |
| `open_questions` | []string | Unresolved questions |
| `resolved_questions` | []ResolvedQuestion | Resolved Q&A pairs (question + answer) |
| `decisions` | []string | Decision strings recorded inline |
| `fidelity` | string | Quality target: `prototype`, `personal-tool`, `alpha`, `beta`, `production` |
| `non_goals` | []string | Explicit non-goals to prevent scope creep |

**Statuses**: `draft`, `active`, `completed`, `on_hold`, `cancelled`, `archived`

### Story

The atomic work unit. The only entity that goes through execution and verification.

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | `s_` + ULID |
| `slug` | string | URL-safe identifier |
| `title` | string | Display name |
| `status` | string | Current status |
| `priority` | string | `critical`, `high`, `medium`, `low` |
| `epic` | string | Parent epic slug (optional) |
| `blocked_by` | []string | Story slugs that block this story |
| `labels` | []string | Freeform labels |
| `acceptance` | []string | Acceptance criteria |
| `doc_refs` | []string | Slugs of referenced docs |
| `open_questions` | []string | Unresolved questions |
| `resolved_questions` | []ResolvedQuestion | Resolved Q&A pairs (question + answer) |
| `assumptions` | []string | Assumptions discovered during execution |
| `fidelity` | string | Quality target (inherits from epic context if not set) |
| `non_goals` | []string | Explicit non-goals for this story |

**Statuses**: `draft`, `planned`, `in_progress`, `verifying`, `done`, `blocked`, `cancelled`

**Status transition diagram:**

```
draft --> planned --> in_progress --> verifying --> done
  |          |            |     \         |
  |          |            |      +--------+
  |          |            |      (can skip verifying)
  v          v            v
blocked <----+------------+
  |
  +--> draft | planned | in_progress
       (unblocked, returns to prior state)

any --> cancelled
cancelled --> draft | planned
```

Transitions are validated in `models.ValidateTransition()`. Any status can transition to `blocked` or `cancelled`. `done` can only transition to `cancelled`. From `blocked`, a story can return to `draft`, `planned`, or `in_progress`. From `cancelled`, a story can return to `draft` or `planned`. The `verifying` status can transition back to `in_progress` (rework needed) or forward to `done`.

### Document

Specs, PRDs, ADRs, and other project documentation. Can be scoped to an epic or project-level.

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | `d_` + ULID |
| `slug` | string | URL-safe identifier |
| `type` | string | Document type |
| `title` | string | Display name |
| `status` | string | Current status |
| `epic` | string | Parent epic slug (optional) |
| `open_questions` | []string | Unresolved questions |
| `resolved_questions` | []ResolvedQuestion | Resolved Q&A pairs (question + answer) |

**Types**: `prd`, `tech-spec`, `api-spec`, `design-spec`, `adr`, `one-pager`

**Statuses**: `draft`, `review`, `approved`, `superseded`

### Decision

Lightweight choice record. Records decisions made during planning or implementation. Lives in `.specflow/decisions/` (project-level, not scoped to an epic). For formal Architecture Decision Records with epic scope, open questions, and document status tracking, use a Document with `type: adr` instead.

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | `dec_` + ULID |
| `slug` | string | URL-safe identifier |
| `date` | string | Date of decision (YYYY-MM-DD) |
| `title` | string | Decision title |
| `status` | string | Current status |
| `context_refs` | []string | Related epic/doc slugs |

**Statuses**: `proposed`, `accepted`, `superseded`, `deprecated`

### Plan

Implementation plan generated by Claude Code, stored by specflow. Linked to a story.

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | `p_` + ULID |
| `story` | string | Story slug |
| `status` | string | Current status |
| `git_ref_baseline` | string | Commit SHA when plan was created |
| `estimated_files` | int | Number of files expected to change |

**Statuses**: `draft`, `approved`, `executing`, `verified`

### Execution

Tracks a single execution attempt of a story. Stored as `meta.yaml` (pure YAML, no markdown body).

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | `x_` + ULID |
| `story` | string | Story slug |
| `plan` | string | Plan ID (optional) |
| `status` | string | Current status |
| `started_at` | time.Time | When execution began |
| `completed_at` | *time.Time | When execution ended (nil if in progress) |
| `git_ref_before` | string | Commit SHA at start |
| `git_ref_after` | string | Commit SHA at completion |
| `files_changed` | []FileChange | List of changed files with action (added/modified) |
| `handover_notes` | string | Markdown notes written when execution is paused (for session handover) |

**Statuses**: `started`, `completed`, `paused`, `failed`

### Verification

Post-execution comparison of implementation against plan and acceptance criteria.

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | `v_` + ULID |
| `execution` | string | Execution ID |
| `story` | string | Story slug |
| `result` | string | Overall result |
| `stats` | VerificationStats | Counts by severity (critical, major, minor) |
| `findings` | []Finding | Individual findings with severity, category, file, description, suggestion |
| `acceptance_check` | []AcceptanceCheck | Per-criteria pass/fail |
| `assumptions` | []string | Assumptions discovered during verification |

**Results**: `pass`, `fail`, `partial`

**Finding severities**: `critical`, `major`, `minor`

**Finding categories**: `missing`, `bug`, `performance`, `security`, `clarity`, `quality`

### LogEntry

Append-only activity log entry stored in `log.jsonl`. Uses JSON tags (not YAML) since the log is JSONL format.

| Field | Type | Description |
|-------|------|-------------|
| `ts` | time.Time | Timestamp |
| `type` | string | Event type |
| `entity` | string | Entity ID or slug |
| `from` / `to` | string | Status transition (for status changes) |
| `epic` / `story` | string | Context references |
| `git_ref` | string | Git reference (for execution events) |
| `files_changed` | int | Count of changed files |
| `result` | string | Verification result |
| `critical` / `major` / `minor` | int | Finding severity counts |

**Event types**: `story.status_changed`, `execution.started`, `execution.completed`, `execution.paused`, `verification.saved`, `doc.created`, `doc.updated`, `decision.recorded`, `initiative.created`, `epic.created`, `story.created`, `plan.saved`, `question.resolved`, `epic.archived`, `story.archived`, `initiative.archived`, `epic.unarchived`, `story.unarchived`, `initiative.unarchived`

---

## 5. Context Builder

The context builder (`internal/context/builder.go`) is the core value proposition of specflow. When Claude Code calls `sf_context_build`, it assembles a 6-layer context document from project state.

### Layer Architecture

| Layer | Name | What It Provides |
|-------|------|------------------|
| 1 | Project Conventions | `CLAUDE.md` from the consuming project, `AGENTS.md` if it exists, `.specflow/config.yaml` project-specific rules |
| 2 | Initiative/Epic Context | Initiative goal + success criteria, epic description + phase map, completed stories (title + summary only), in-progress stories (title + what's happening), decisions made so far |
| 3 | Spec Requirements | All docs referenced by the story (`doc_refs`) in full content, acceptance criteria extracted and highlighted |
| 4 | Implementation Plan | Approved plan with file-level detail, handover notes from paused executions, or a "no plan yet" prompt if none exists |
| 5 | Referenced Files | Files mentioned in the plan (current content), pattern exemplar files from config, files created by completed predecessor stories |
| 6 | Open Items | Open questions that might affect implementation, assumptions from related stories, blockers (should be empty if story is ready to work) |

### Assembly Algorithm

1. Load story, follow `epic` ref, follow `initiative` ref.
2. Load all docs referenced by `story.doc_refs`.
3. Load plan if it exists.
4. If the latest execution for the story is paused, load handover notes.
5. Load completed sibling stories (title + summary only, not full content).
6. Load decisions for the epic.
7. Collect open questions from all layers (story, docs, epic, initiative).
8. Collect assumptions from completed stories.
9. Read `CLAUDE.md` and `AGENTS.md` from the project root.
10. Read pattern exemplar files from config.
11. Read files referenced in the plan.
12. Render through `context.md.tmpl` template.

Files that do not yet exist are noted as "planned but not yet created."

---

## 6. Design Decisions

### Filesystem over SQLite

All state lives in `.specflow/` as markdown files and YAML. No database.

- Git-friendly: every artifact is diffable, mergeable, and versioned alongside code.
- Human-readable: artifacts can be browsed and edited with any text editor.
- Zero dependencies: no database driver, no CGO, single static binary.
- Trade-off: no indexed queries. Full-text search walks all files. Acceptable at personal-tool scale (dozens of epics, hundreds of stories).

### ULID IDs

ULIDs instead of UUIDs or sequential integers.

- Time-sortable: entities sort by creation time without a sequence counter.
- Collision-resistant: safe for concurrent creation without coordination.
- Prefixed: `s_`, `e_`, `i_`, etc. allow type identification from an ID alone.
- Trade-off: longer than sequential integers. Acceptable since slugs are the primary human-facing identifier.

### Frontmatter Format

YAML frontmatter + markdown body, parsed with `adrg/frontmatter`.

- Two concerns in one file: structured metadata for machines, rich content for humans.
- Roundtrip safe: body formatting is preserved through parse/write cycles.
- Standard format: widely understood by developers, supported by many editors.
- Trade-off: parsing is slightly more complex than pure YAML or pure markdown. The `adrg/frontmatter` library handles this cleanly.

### Zero AI Logic

specflow never calls AI APIs. No Anthropic SDK. No prompt engineering in Go code.

- Separation of concerns: specflow manages state, Claude Code provides intelligence.
- No API keys to manage, no rate limits to handle, no token costs.
- MCP tool descriptions carry behavioral instructions that shape how Claude Code interacts with users -- this is the only place "prompt engineering" exists, and it is static text.
- Trade-off: specflow cannot function standalone as an intelligent tool. It requires Claude Code (or another MCP client) to provide the reasoning layer.

### Append-Only Activity Log

All state changes are appended to `log.jsonl` as one JSON object per line.

- Auditable: complete history of all state transitions.
- Simple: append is the only write operation, no file locking concerns for reads.
- Greppable: JSONL is trivially searchable with standard Unix tools.
- Trade-off: log grows unbounded. At personal-tool scale, this is negligible (a few MB after thousands of operations).

### Epics Own Their Stories and Docs

Stories and docs scoped to an epic live inside the epic's directory, not in a flat global directory.

- Colocation: browsing an epic directory shows everything related to it.
- Isolation: deleting or archiving an epic is a single directory operation.
- Trade-off: standalone stories and project-level docs require separate top-level directories. Moving a story between epics requires a file move.

### Executions Are Flat

Executions are indexed by story slug under a top-level `executions/` directory, not nested under epics.

- Stories can move between epics without breaking execution references.
- A story can have multiple execution attempts, each with its own plan, verification, and git refs.
- Trade-off: requires a lookup by story slug rather than browsing within an epic directory.

### Archiving

Completed epics, standalone stories, and initiatives can be archived to `.specflow/archive/`. Archiving moves files out of day-to-day directories.

- **Filesystem separation**: archived items are excluded from all default listings (CLI and MCP) without filtering logic. Standard `ListEpics()` / `ListStories()` / `ListInitiatives()` only read active directories.
- **Optional compaction**: by default, archived files preserve their full markdown body. When `--compact` (CLI) or `compact: true` (MCP) is used, story, epic, and initiative bodies are stripped to frontmatter-only tombstones. Compaction is lossy but git preserves the full content in history.
- **Docs preserved**: epic-scoped documents are moved as-is (never compacted) since they may be referenced by future stories via `doc_refs`.
- **Cross-reference resolution**: `sf_context_build`, `sf_story_next`, and `sf_blocked` fall back to archived epics and standalone stories when resolving blocker references, story lookups, and doc refs.
- **Opt-in visibility**: `--include-archived` (CLI) and `include_archived` (MCP) flags allow listing archived epics, stories, and initiatives when needed.
- **Standalone stories**: archived via `specflow story archive <slug>` or `sf_story_archive` to `.specflow/archive/stories/{slug}.md`. Only standalone stories (no epic) can be archived individually; epic-scoped stories are archived with their epic.
- **Initiatives**: archived via `specflow initiative archive <slug>` or `sf_initiative_archive` to `.specflow/archive/initiatives/{slug}/initiative.md`. Requires all linked epics to be archived or completed (unless force is used). Initiative archiving does not cascade to linked epics.
- **Unarchiving**: archived items can be restored via `unarchive` commands (CLI) or `sf_*_unarchive` MCP tools. Epics and initiatives are restored with status `on_hold`; standalone stories are restored with status `planned`. Docs and executions are moved back to their original locations.
