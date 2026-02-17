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
| `internal/hardq` | Hard questions template engine. Returns deterministic, template-based questions per entity type (initiative, epic, story, PRD, tech spec). No AI involved. |
| `internal/ui` | Terminal output rendering. Tables via lipgloss, markdown rendering via glamour, progress bars, and colored status badges. |

### Entry Points

| File | Role |
|------|------|
| `cmd/specflow/main.go` | Cobra root command setup. No business logic. |
| `cmd/specflow/mcp.go` | Starts the MCP server on stdio. |
| `cmd/specflow/*.go` | One file per CLI command group (initiative, epic, story, doc, decision, status, etc.). |

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
+-- templates/                               # User-customizable template overrides
|   +-- prd.md.tmpl
|   +-- tech-spec.md.tmpl
|   +-- api-spec.md.tmpl
|   +-- design-spec.md.tmpl
|   +-- adr.md.tmpl
|   +-- one-pager.md.tmpl
|   +-- story.md.tmpl
|   +-- epic.md.tmpl
|   +-- initiative.md.tmpl
|   +-- review-prd.md.tmpl
|   +-- review-tech-spec.md.tmpl
|   +-- review-story.md.tmpl
|   +-- decompose.md.tmpl
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
    +-- {story-slug}/
        +-- {exec-id}/
            +-- plan.md                      # Implementation plan
            +-- verification.md              # Verification findings
            +-- meta.yaml                    # Git refs, timestamps, status
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

**Statuses**: `active`, `completed`, `on_hold`, `archived`

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
| `decisions` | []string | Decision strings recorded inline |

**Statuses**: `draft`, `active`, `completed`, `on_hold`, `archived`

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
| `assumptions` | []string | Assumptions discovered during execution |

**Statuses**: `draft`, `planned`, `in_progress`, `verifying`, `done`, `blocked`

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
```

Transitions are validated in `models.ValidateTransition()`. The `done` status is terminal. Any status can transition to `blocked`. From `blocked`, a story can return to `draft`, `planned`, or `in_progress`. The `verifying` status can transition back to `in_progress` (rework needed) or forward to `done`.

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

**Types**: `prd`, `tech-spec`, `api-spec`, `design-spec`, `adr`, `one-pager`

**Statuses**: `draft`, `review`, `approved`, `superseded`

### Decision

Lightweight ADR. Records choices made during planning or implementation.

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

**Statuses**: `started`, `completed`, `failed`

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

**Event types**: `story.status_changed`, `execution.started`, `execution.completed`, `verification.saved`, `doc.created`, `doc.updated`, `decision.recorded`, `initiative.created`, `epic.created`, `story.created`, `plan.saved`, `question.resolved`

---

## 5. Context Builder

The context builder (`internal/context/builder.go`) is the core value proposition of specflow. When Claude Code calls `sf_context_build`, it assembles a 6-layer context document from project state.

### Layer Architecture

| Layer | Name | What It Provides |
|-------|------|------------------|
| 1 | Project Conventions | `CLAUDE.md` from the consuming project, `AGENTS.md` if it exists, `.specflow/config.yaml` project-specific rules |
| 2 | Initiative/Epic Context | Initiative goal + success criteria, epic description + phase map, completed stories (title + summary only), in-progress stories (title + what's happening), decisions made so far |
| 3 | Spec Requirements | All docs referenced by the story (`doc_refs`) in full content, acceptance criteria extracted and highlighted |
| 4 | Implementation Plan | Approved plan with file-level detail, or a "no plan yet" prompt if none exists |
| 5 | Referenced Files | Files mentioned in the plan (current content), pattern exemplar files from config, files created by completed predecessor stories |
| 6 | Open Items | Open questions that might affect implementation, assumptions from related stories, blockers (should be empty if story is ready to work) |

### Assembly Algorithm

1. Load story, follow `epic` ref, follow `initiative` ref.
2. Load all docs referenced by `story.doc_refs`.
3. Load plan if it exists.
4. Load completed sibling stories (title + summary only, not full content).
5. Load decisions for the epic.
6. Collect open questions from all layers (story, docs, epic, initiative).
7. Collect assumptions from completed stories.
8. Read `CLAUDE.md` and `AGENTS.md` from the project root.
9. Read pattern exemplar files from config.
10. Read files referenced in the plan.
11. Render through `context.md.tmpl` template.

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
