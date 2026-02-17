# specflow

A spec-driven development CLI that acts as a structured memory and context layer for Claude Code.

## What It Is

specflow sits between the human (architect/PM/tech lead) and Claude Code (the AI agent), providing:

1. **Structured artifact management** -- initiatives, epics, stories, specs, decisions stored as git-friendly markdown
2. **Rich context assembly** -- layered prompt building from project state for Claude Code
3. **Progress tracking** -- status rollup across the full hierarchy
4. **Verification support** -- plan-vs-implementation comparison data
5. **Working style enforcement** -- behavioral instructions embedded in MCP tool descriptions

**Core principle:** specflow has zero AI logic. No API keys. No LLM calls. Claude Code IS the AI. specflow manages structured state and assembles context. All intelligence comes from Claude Code interacting through MCP tools.

## What It Is Not

- **Not an AI agent** -- Claude Code is the agent; specflow manages state and context
- **Not a replacement for git** -- everything is stored as plain markdown, designed to be committed
- **Not a team collaboration tool** -- built for personal use
- **Not a GUI** -- CLI and MCP only

## Architecture

```
+-------------------------------------------------+
|                  Human (CLI)                     |
|   specflow epic new, specflow status, etc.       |
+-------------------+-----------------------------+
                    |
                    v
+-------------------------------------------------+
|            specflow (Go binary)                  |
|                                                  |
|   CLI mode         |    MCP server mode (stdio)  |
|   (human ops)      |    (Claude Code ops)        |
|                    |                              |
|   +--------------+----------------------------+  |
|   |          Shared Core                      |  |
|   |  Store - Context Builder - Git - HardQ    |  |
|   +-------------------------------------------+  |
|                    |                              |
|   +--------------+----------------------------+  |
|   |      Filesystem Store (.specflow/)        |  |
|   +-------------------------------------------+  |
+-------------------------------------------------+
                    | MCP (stdio)
                    v
+-------------------------------------------------+
|               Claude Code                        |
|                                                  |
|  Reads context -> implements code -> writes results |
+-------------------------------------------------+
```

Two modes, one binary:

```sh
specflow [command]     # CLI mode -- human creates/manages artifacts
specflow mcp           # MCP mode -- Claude Code reads/writes via stdio
```

## Installation

### From source

```sh
go install github.com/balazscsaba2006/specflow/cmd/specflow@latest
```

### Pre-built binaries

Download from [GitHub Releases](https://github.com/balazscsaba2006/specflow/releases). Binaries are available for Linux, macOS, and Windows on amd64 and arm64.

## Quick Start

```sh
# Initialize a new specflow project
specflow init

# Optionally configure Claude Code MCP integration
specflow init --with-claude

# Create an epic
specflow epic new auth-system

# Create stories under that epic
specflow story new jwt-middleware --epic auth-system
specflow story new api-key-store --epic auth-system

# Create a PRD for the epic
specflow doc new prd --type prd --epic auth-system

# See what to work on next
specflow story next --epic auth-system
```

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

## How It Works with Claude Code

**Human workflow (CLI):**
```
specflow epic new "auth-system"
  -> CLI creates .specflow/epics/auth-system/epic.md
  -> Opens $EDITOR for human to write description
  -> Human saves and closes editor
  -> CLI confirms creation
```

**Claude Code workflow (MCP):**
```
Human tells Claude: "What should I work on next?"
  -> Claude calls sf_story_next()
  -> specflow returns the next unblocked story
  -> Claude calls sf_context_build("jwt-middleware")
  -> specflow assembles: project conventions + epic context + specs + plan
  -> Claude reads the context, implements the code
  -> Claude calls sf_execution_start("jwt-middleware")
  -> specflow records git ref baseline
  -> Claude finishes implementation
  -> Claude calls sf_execution_complete("x_01...")
  -> specflow captures git ref after + diff
  -> Claude calls sf_verify_save("jwt-middleware", findings)
  -> specflow stores verification results
  -> Claude calls sf_story_update("jwt-middleware", status="done")
```

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
| `specflow version` | | Print version information |

Each command group supports subcommands like `new`, `ls`, `show`, `edit`, and `set`. Run `specflow <command> --help` for details.

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

specflow exposes MCP tools prefixed with `sf_` for Claude Code integration. Start the MCP server:

```sh
specflow mcp
```

Or configure it automatically:

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
| `sf_assumptions` | All recorded assumptions |
| `sf_hard_questions` | Contextual hard questions for any entity |
| `sf_review_prompt` | Coaching/review prompt for a document |

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

### Context Builder

The core value of specflow. When Claude Code calls `sf_context_build`, it assembles a six-layer context document:

| Layer | Content |
|-------|---------|
| 1. Project Conventions | CLAUDE.md, AGENTS.md, project config |
| 2. Initiative/Epic Context | Goal, phase map, completed stories, decisions |
| 3. Spec Requirements | Referenced docs (full content), acceptance criteria |
| 4. Implementation Plan | Approved plan with file-level detail |
| 5. Referenced Files | Files from plan, pattern exemplars, predecessor outputs |
| 6. Open Items | Open questions, assumptions, blockers |

## Storage

All data lives in `.specflow/` as markdown with YAML frontmatter:

```
.specflow/
+-- config.yaml
+-- log.jsonl                         # Append-only activity log
+-- templates/                        # User-customizable templates
+-- initiatives/{slug}/
|   +-- initiative.md
+-- epics/{slug}/
|   +-- epic.md
|   +-- docs/                         # Specs scoped to this epic
|   +-- stories/                      # Stories under this epic
+-- stories/                          # Standalone stories
+-- docs/                             # Project-level documents
+-- decisions/                        # Decision log
+-- executions/{story-slug}/{exec-id}/
    +-- plan.md
    +-- verification.md
    +-- meta.yaml                     # Git refs, timestamps, status
```

All artifacts use the same format: YAML frontmatter for structured metadata, markdown body for human-written content. IDs are ULID-based (time-sortable) with type prefixes (`i_`, `e_`, `s_`, `d_`, `dec_`, `p_`, `x_`, `v_`).

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
