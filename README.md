# specflow

A spec-driven development CLI that acts as a structured memory and context layer for Claude Code.

## What It Is

specflow sits between the human (architect/PM/tech lead) and Claude Code (the AI agent), providing:

1. **Structured artifact management** -- initiatives, epics, stories, specs, decisions stored as markdown with YAML frontmatter
2. **Rich context assembly** -- layered prompt building from project state for Claude Code
3. **Progress tracking** -- status rollup across the full hierarchy
4. **Verification support** -- plan-vs-implementation comparison data
5. **Working style enforcement** -- behavioral instructions embedded in MCP tool descriptions
6. **Archiving** -- move completed epics, stories, and initiatives out of day-to-day workflows while preserving cross-references

**Core principle:** specflow has zero AI logic. No API keys. No LLM calls. Claude Code IS the AI. specflow manages structured state and assembles context. All intelligence comes from Claude Code interacting through MCP tools.

## What It Is Not

- **Not an AI agent** -- Claude Code is the agent; specflow manages state and context
- **Not a replacement for git** -- artifacts are plain markdown files that can optionally be version-controlled
- **Not a team collaboration tool** -- built for personal use
- **Not a GUI** -- CLI and MCP only

## Architecture

```
+-----------------------------------------------------+
|                  Human (CLI)                        |
|   specflow epic new, specflow status, etc.          |
+-------------------+---------------------------------+
                     |
                     v
+-----------------------------------------------------+
|            specflow (Go binary)                     |
|                                                     |
|   CLI mode         |    MCP server mode (stdio)     |
|   (human ops)      |    (Claude Code ops)           |
|                    |                                |
|   +--------------+----------------------------+     |
|   |          Shared Core                      |     |
|   |  Store - Context Builder - Git - HardQ    |     |
|   +-------------------------------------------+     |
|                    |                                |
|   +--------------+----------------------------+     |
|   |      Filesystem Store (.specflow/)        |     |
|   +-------------------------------------------+     |
+-----------------------------------------------------+
                     | MCP (stdio)
                     v
+-----------------------------------------------------+
|               Claude Code                           |
|                                                     |
|  Reads context -> implements code -> writes results |
+-----------------------------------------------------+
```

Two modes, one binary:

```sh
specflow [command]     # CLI mode -- human creates/manages artifacts
specflow mcp           # MCP mode -- Claude Code reads/writes via stdio
```

## Installation

- **From source:** `go install github.com/balazscsaba2006/specflow/cmd/specflow@latest`
- **Pre-built binaries:** Download from [GitHub Releases](https://github.com/balazscsaba2006/specflow/releases) (Linux, macOS, Windows -- amd64 and arm64)

See [Getting Started](docs/getting-started.md) for detailed setup instructions including MCP configuration and skill installation.

## The Hierarchy

```
Project (.specflow/)
  +-- Initiative (optional -- groups epics toward a strategic goal)
    +-- Epic (optional -- a shippable feature/capability)
      +-- Story (the atomic work unit)
```

Everything upward is optional:
- `initiative > epic > story` -- full hierarchy
- `epic > story` -- no initiative
- Standalone story -- no epic, no initiative

## CLI Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `specflow init` | | Initialize a new project (`.specflow/` directory) |
| `specflow config` | | Get, set, or list project configuration |
| `specflow initiative` | `i` | Create, list, show, edit, and update initiatives |
| `specflow epic` | `e` | Create, list, show, edit, and update epics |
| `specflow story` | `s` | Create, list, show, edit, update stories; find next story |
| `specflow doc` | `d` | Create, list, show, and edit documents |
| `specflow decision` | `dec` | Create, list, and show architectural decisions |
| `specflow status` | | Project-wide or per-entity status rollup with progress bars |
| `specflow questions` | | List all open questions across the project |
| `specflow blocked` | | Show all blocked stories and their blockers |
| `specflow assumptions` | | List all recorded assumptions |
| `specflow log` | | Show the activity log |
| `specflow search` | | Full-text search across all artifacts |
| `specflow import` | | Import an existing markdown file as an artifact |
| `specflow mode` | | Show or set project mode (`fast` / `careful`) |
| `specflow template` | `tmpl` | List, override, and reset templates |
| `specflow sync` | | Update the installed Claude Code skill from the current binary |
| `specflow mcp` | | Start MCP server on stdio for Claude Code |
| `specflow version` | | Print version information |

Each command group supports subcommands like `new`, `ls`, `show`, `edit`, and `set`. See the [CLI Reference](docs/cli-reference.md) for full details.

### Document Types

Documents are formal artifacts that live in the docs hierarchy and can be scoped to an epic.

| Type | Flag | Purpose |
|------|------|---------|
| PRD | `--type prd` | Problem, users, goals, scope, requirements, risks |
| Tech Spec | `--type tech-spec` | Architecture, data model, API changes, constraints |
| API Spec | `--type api-spec` | Endpoint contracts, auth, rate limiting, versioning |
| Design Spec | `--type design-spec` | Problem, goals, non-goals, component design, trade-offs |
| ADR | `--type adr` | Architecture Decision Record -- formal, with alternatives considered |
| One-Pager | `--type one-pager` | Lightweight proposal: TL;DR, problem, solution, trade-offs |

Each type has a dedicated template with type-specific sections. See `specflow template ls` for the full list.

### Decisions vs ADR Docs

specflow has two ways to record decisions:

- **`specflow decision new`** -- A lightweight decision record. Lives in `.specflow/decisions/`. Use for quick choices made during planning or implementation (e.g., "use JWT over sessions", "store files in S3"). No epic scope, no open questions tracking.

- **`specflow doc new --type adr`** -- A full Architecture Decision Record. Lives in the docs hierarchy, can be scoped to an epic, tracks open questions, and has a full status lifecycle (`draft` → `review` → `approved`). Use for significant architectural choices that need team visibility or formal review.

## MCP Tools

specflow exposes 33 MCP tools prefixed with `sf_` for Claude Code integration. Start the MCP server:

```sh
specflow mcp
```

The MCP server is configured automatically during `specflow init`.

### Read Tools

| Tool | Purpose |
|------|---------|
| `sf_status` | Project-wide status rollup |
| `sf_initiative_show` | Full initiative detail |
| `sf_epic_show` | Epic detail with phases and stories |
| `sf_story_show` | Story detail with acceptance criteria |
| `sf_story_next` | Next recommended story to work on |
| `sf_story_ls` | List stories with filters |
| `sf_doc_read` | Read a document by slug |
| `sf_plan_read` | Read implementation plan for a story |
| `sf_verify_read` | Read latest verification results |
| `sf_context_build` | Build full execution context for a story |
| `sf_questions` | All open questions grouped by source |
| `sf_blocked` | All blocked stories with their blockers |
| `sf_decisions` | Decision log |
| `sf_log` | Activity log entries |
| `sf_assumptions` | All recorded assumptions |
| `sf_hard_questions` | Contextual hard questions for any entity |
| `sf_review_prompt` | Coaching/review prompt for a document |
| `sf_diff` | Git diff between refs or for a story's execution |
| `sf_scope_check` | Cross-reference stories against PRD scope |
| `sf_diff_check` | Detect drift between docs and stories by timestamps |
| `sf_scope_drift` | Compare planned vs actual files changed in execution |
| `sf_unstuck` | Diagnostic tool for debugging stuck implementations |
| `sf_export` | Export any entity (initiative, epic, story, doc, decision) as YAML, markdown, or HTML |

### Write Tools

| Tool | Purpose |
|------|---------|
| `sf_initiative_create` | Create a new initiative |
| `sf_epic_create` | Create a new epic |
| `sf_story_create` | Create a new story |
| `sf_story_update` | Update story fields (status, priority, etc.) |
| `sf_doc_write` | Create or update a document |
| `sf_decision_record` | Record an architectural decision |
| `sf_plan_save` | Save an implementation plan |
| `sf_execution_start` | Start execution tracking (records git ref) |
| `sf_execution_complete` | Complete execution (captures diff) |
| `sf_execution_pause` | Pause execution with handover notes for session continuity |
| `sf_verify_save` | Save verification results |
| `sf_question_resolve` | Mark an open question as resolved |
| `sf_epic_archive` | Archive an epic (move to archive, optional compaction) |
| `sf_story_archive` | Archive a standalone story (move to archive, optional compaction) |
| `sf_initiative_archive` | Archive an initiative (move to archive, optional compaction) |
| `sf_epic_unarchive` | Restore an archived epic to active state |
| `sf_story_unarchive` | Restore an archived standalone story to active state |
| `sf_initiative_unarchive` | Restore an archived initiative to active state |

See the [MCP Tools Reference](docs/mcp-tools.md) for parameters and response formats.

### Context Builder

The core value of specflow. When Claude Code calls `sf_context_build`, it assembles a six-layer context document:

| Layer | Content |
|-------|---------|
| 1. Project Conventions | CLAUDE.md, AGENTS.md, project config |
| 2. Initiative/Epic Context | Goal, phase map, completed stories, decisions |
| 3. Spec Requirements | Referenced docs (full content), acceptance criteria |
| 4. Implementation Plan | Approved plan with file-level detail, handover notes from paused sessions |
| 5. Referenced Files | Files from plan, pattern exemplars |
| 6. Open Items | Open questions, assumptions, blockers |

## Modes

specflow supports two modes that control the level of ceremony:

```sh
specflow mode           # Show current mode
specflow mode fast      # Switch to fast mode
specflow mode careful   # Switch to careful mode (default)
```

| Aspect | Fast | Careful |
|--------|------|---------|
| Story templates | Title + acceptance only | Full template with all fields |
| Hard questions | Suppressed | Always included |
| Verification | Only on explicit request | Prompted after every execution |

## Storage

All data lives in `.specflow/` as markdown with YAML frontmatter:

```
.specflow/
+-- config.yaml                          # Project configuration
+-- log.jsonl                            # Append-only activity log
+-- templates/                           # Per-project template overrides
+-- initiatives/{slug}/
|   +-- initiative.md
+-- epics/{slug}/
|   +-- epic.md
|   +-- docs/                            # Specs scoped to this epic
|   +-- stories/                         # Stories under this epic
+-- stories/                             # Standalone stories
+-- docs/                                # Project-level documents
+-- decisions/                           # Decision log
+-- executions/{story-slug}/
|   +-- latest/
|   |   +-- plan.md                      # Implementation plan (latest)
|   +-- {exec-id}/
|       +-- verification.md
|       +-- meta.yaml                    # Git refs, timestamps, status
+-- archive/                             # Archived (completed) items
    +-- initiatives/{slug}/             # Compacted initiative tombstones
    +-- epics/{slug}/                    # Compacted epic + stories + docs
    +-- stories/                         # Compacted standalone stories
    +-- executions/{story-slug}/         # Moved execution data
```

All artifacts use the same format: YAML frontmatter for structured metadata, markdown body for human-written content. IDs are ULID-based (time-sortable) with type prefixes (`i_`, `e_`, `s_`, `d_`, `dec_`, `p_`, `x_`, `v_`).

### Git Integration

The `.specflow/` directory works in two ways:

**Committed to git (default)** -- Artifacts are version-controlled alongside your code. Changes to specs, stories, and decisions are tracked in git history. This is the recommended approach for most projects.

**Local-only** -- If you prefer to keep specflow state out of version control, exclude it:

```sh
# Add to .git/info/exclude (local to your clone, not committed)
echo ".specflow/" >> .git/info/exclude

# Or add to .gitignore (shared with team)
echo ".specflow/" >> .gitignore
```

This is useful when specflow is purely a personal planning layer and you don't want to add metadata files to the repository.

### Template Customization

specflow ships with built-in templates for all artifact types. Every doc type has a dedicated template with type-specific sections -- no generic fallback needed for supported types.

Override any template per-project using the `template` command:

```sh
specflow template ls                    # List all templates and override status
specflow template override tech-spec    # Copy to .specflow/templates/doc_tech-spec.md
specflow template reset tech-spec       # Remove override, revert to default
```

Available templates:

| Template | Used by |
|----------|---------|
| `initiative` | `specflow initiative new` |
| `epic` | `specflow epic new` |
| `story` | `specflow story new` (careful mode) |
| `story_fast` | `specflow story new` (fast mode) |
| `decision` | `specflow decision new` |
| `doc_prd` | `specflow doc new --type prd` |
| `doc_tech-spec` | `specflow doc new --type tech-spec` |
| `doc_api-spec` | `specflow doc new --type api-spec` |
| `doc_design-spec` | `specflow doc new --type design-spec` |
| `doc_adr` | `specflow doc new --type adr` |
| `doc_one-pager` | `specflow doc new --type one-pager` |
| `doc_generic` | Fallback for unknown doc types |
Resolution order: `.specflow/templates/<name>.md` (project override) → embedded default.

The `skill` template is not overridable — it ships with the binary and is installed globally.

### Claude Code Skill

`specflow init` installs a Claude Code skill globally to `~/.claude/skills/specflow/SKILL.md`. The skill encodes the spec-driven development workflow: context building, quality gates, verification loops, and working style conventions. It teaches Claude Code how to orchestrate `sf_*` tools in the right sequence.

The skill source is embedded in the binary (`templates/skill.md`). It is installed on `specflow init` and updated on `specflow update`. You can also sync manually with `specflow sync`.

## Documentation

- [Getting Started](docs/getting-started.md) -- Installation, MCP setup, and first workflow
- [CLI Reference](docs/cli-reference.md) -- Full command reference with flags and examples
- [MCP Tools Reference](docs/mcp-tools.md) -- All MCP tool parameters and response formats
- [Architecture](docs/architecture.md) -- System design, data models, and storage format

## Development

```sh
make help          # Show all available targets
make build         # Build the binary
make test          # Run tests
make quality       # Run all checks (format + vet + lint + test)
make fix           # Auto-fix formatting and lint issues
```

Requires Go 1.24+ and [golangci-lint](https://golangci-lint.run/) for linting.

## License

MIT
