# specflow

A spec-driven development CLI that acts as a structured memory and context layer for Claude Code.

## What It Is

specflow sits between the human (architect/PM/tech lead) and Claude Code (the AI agent), providing:

1. **Structured artifact management** -- initiatives, epics, stories, specs, decisions stored as markdown with YAML frontmatter
2. **Rich context assembly** -- layered prompt building from project state for Claude Code
3. **Progress tracking** -- status rollup across the full hierarchy
4. **Verification support** -- plan-vs-implementation comparison data
5. **Working style enforcement** -- behavioral instructions embedded in MCP tool descriptions

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

See [Getting Started](docs/getting-started.md) for detailed setup instructions including MCP configuration.

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
| `specflow mcp` | | Start MCP server on stdio for Claude Code |
| `specflow version` | | Print version information |

Each command group supports subcommands like `new`, `ls`, `show`, `edit`, and `set`. See the [CLI Reference](docs/cli-reference.md) for full details.

### Document Types

| Type | Flag | Purpose |
|------|------|---------|
| PRD | `--type prd` | Problem, users, goals, scope, requirements, risks |
| Tech Spec | `--type tech-spec` | Architecture, data model, API changes, testing strategy |
| API Spec | `--type api-spec` | Endpoint contracts and schemas |
| Design Spec | `--type design-spec` | UI/UX design specifications |
| ADR | `--type adr` | Architecture Decision Record (context, decision, consequences) |
| One-Pager | `--type one-pager` | Lightweight proposal (problem, solution, metrics) |

## MCP Tools

specflow exposes 30 MCP tools prefixed with `sf_` for Claude Code integration. Start the MCP server:

```sh
specflow mcp
```

Or configure it automatically during init:

```sh
specflow init --with-claude
```

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
| `sf_verify_save` | Save verification results |
| `sf_question_resolve` | Mark an open question as resolved |

See the [MCP Tools Reference](docs/mcp-tools.md) for parameters and response formats.

### Context Builder

The core value of specflow. When Claude Code calls `sf_context_build`, it assembles a six-layer context document:

| Layer | Content |
|-------|---------|
| 1. Project Conventions | CLAUDE.md, AGENTS.md, project config |
| 2. Initiative/Epic Context | Goal, phase map, completed stories, decisions |
| 3. Spec Requirements | Referenced docs (full content), acceptance criteria |
| 4. Implementation Plan | Approved plan with file-level detail |
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
    +-- latest/
    |   +-- plan.md                      # Implementation plan (latest)
    +-- {exec-id}/
        +-- verification.md
        +-- meta.yaml                    # Git refs, timestamps, status
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

specflow ships with built-in templates for all artifact types. You can override any template per-project by placing a file in `.specflow/templates/`:

```
.specflow/templates/
+-- initiative.md      # Override initiative creation template
+-- epic.md            # Override epic creation template
+-- story.md           # Override story creation template (careful mode)
+-- story_fast.md      # Override story creation template (fast mode)
+-- decision.md        # Override decision creation template
+-- doc_prd.md         # Override PRD template
+-- doc_adr.md         # Override ADR template
+-- doc_generic.md     # Override fallback doc template
```

When you run `specflow story new`, for example, specflow checks `.specflow/templates/story.md` first. If it exists, that template is used. Otherwise, the built-in default is used.

This lets you tailor templates per project -- for example, adapting story templates to match your team's ticket format or adding project-specific sections to PRDs.

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
