# CLAUDE.md - Architect & Tech Lead Mode

## Who I Am
I am a software architect and technical lead. I make strategic technical decisions, design systems, and guide teams. I don't need hand-holding — I need a sparring partner who challenges my thinking, catches my blind spots, and helps me move fast without cutting corners.

## Rules for Code Generation

### Before Writing Code
- If I describe a problem, discuss trade-offs before jumping to implementation
- Flag architectural implications I might be overlooking
- If there are multiple valid approaches, present them as options with trade-offs — don't pick for me
- Skip the basics. DO flag when a design choice has non-obvious downstream consequences for the MCP protocol, CLI UX, or storage format

### During Code Generation
- Optimize for maintainability and readability
- Follow existing project conventions unless I explicitly say we're refactoring
- Flag any deviation from established patterns in the codebase
- If something feels over-engineered for a personal tool, say so
- Every public function needs a clear purpose — no "just in case" APIs

### After Writing Code
- Summarize decisions made and assumptions baked in
- Call out what will break if requirements change (the "what if" list)
- If tests are missing or insufficient, flag it — don't silently ship untested code

### Code Review Mode
When I share code for review:
- Review like a principal engineer — be direct, not diplomatic
- Focus on: correctness, edge cases, CLI UX, MCP protocol compliance
- Don't nitpick formatting or style unless it hurts readability
- If something is fine but I'd regret it in 6 months, tell me now

### Architecture Decisions
- When I'm designing specflow features, act as devil's advocate by default
- Ask: What happens with 50 epics and 500 stories? Does the filesystem layout still work? Does the MCP tool response get too large?
- If I'm bikeshedding on CLI flag names, call it out
- If I'm gold-plating a personal tool, tell me to ship it

### Commit Messages
- Conventional commits (feat:, fix:, refactor:, chore:)
- NEVER add "Co-Authored-By" or any AI attribution
- NEVER add long AI-generated descriptions
- Keep them concise — if it needs a paragraph, it should be in the PR description

### General Rules
- Be opinionated. "It depends" without follow-up is useless
- If I'm about to introduce accidental complexity, block me
- If I'm gold-plating, tell me to ship it
- Assume I know Go fundamentals — teach me what I don't know I don't know
- When I'm prototyping, match my pace — skip ceremony
- When I'm building the MCP server (user-facing), slow me down if I'm rushing

---

## Project Overview

**specflow** is a personal spec-driven development CLI tool that acts as a structured memory and context layer for Claude Code. It manages the lifecycle of software development artifacts (initiatives, epics, stories, specs, decisions) and provides Claude Code with rich, assembled context via MCP.

**Core principle:** specflow has ZERO AI logic. Claude Code IS the AI. specflow manages state, assembles context, and tracks progress. All intelligence comes from Claude Code reading and writing through MCP tools.

### What specflow IS
- A Go CLI binary for managing development artifacts (initiatives, epics, stories, docs, decisions)
- An MCP server that Claude Code connects to for reading/writing those artifacts
- A context assembler that builds rich, layered prompts from project state
- A progress tracker with status rollup across the hierarchy

### What specflow is NOT
- An AI agent or orchestrator (Claude Code is the agent)
- A replacement for git (everything is stored as git-friendly markdown)
- A team collaboration tool (personal use)
- A VS Code extension or GUI (CLI + MCP only)

---

## Technology Stack

- **Go 1.24+** — latest stable Go
- **Cobra** — CLI framework
- **mcp-go** (`github.com/mark3labs/mcp-go`) — MCP server SDK
- **Charm libs** — lipgloss (terminal styling), glamour (markdown rendering)
- **go:embed** — embedded default templates
- **adrg/frontmatter** — YAML frontmatter parsing
- **gopkg.in/yaml.v3** — YAML marshal/unmarshal
- **google/uuid** or **oklog/ulid** — ID generation

### Command Preferences
- `go test ./...` for testing
- `golangci-lint run` for linting
- No CGO — pure Go, single static binary
- No external database — filesystem only (.specflow/ directory)
- No Anthropic SDK — specflow never calls AI APIs directly

---

## Coding Standards

### Go Conventions
- Follow standard Go conventions (gofmt, effective Go)
- Use `internal/` for non-exported packages
- Interfaces defined where consumed, not where implemented
- Errors are values — wrap with context using `fmt.Errorf("doing X: %w", err)`
- No `panic` except in truly unrecoverable init situations
- Table-driven tests
- No global state — pass dependencies explicitly

### Project Structure
```
specflow/
├── cmd/specflow/
│   └── main.go                     # Cobra root command setup, no business logic
├── internal/
│   ├── config/                     # Config loading (.specflow/config.yaml + ~/.specflow/config.yaml)
│   ├── store/                      # Filesystem CRUD for all entity types
│   ├── context/                    # Context builder — assembles layered prompts
│   ├── git/                        # Git operations (diff, rev-parse, status)
│   ├── mcp/                        # MCP stdio server, tool registration
│   ├── hardq/                      # Hard questions template engine
│   └── ui/                         # Terminal output (tables, markdown, progress)
├── templates/                      # go:embed default templates
├── testdata/                       # Test fixtures
├── go.mod
├── go.sum
└── Makefile
```

### Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Packages | lowercase, single word | `store`, `context`, `mcp` |
| Interfaces | descriptive noun or `-er` suffix | `Store`, `ContextBuilder` |
| Structs | PascalCase, noun | `Epic`, `Story`, `Plan` |
| Functions | PascalCase, verb+noun | `BuildContext`, `SavePlan` |
| CLI commands | noun + verb (cobra style) | `specflow epic new`, `specflow story ls` |
| Go files | snake_case.go | `context_builder.go`, `epic_store.go` |
| Test files | snake_case_test.go | `context_builder_test.go` |

### Error Handling
- Return errors, don't log-and-continue
- CLI layer (cmd/) handles user-facing error formatting
- Internal packages return structured errors
- MCP layer translates errors to MCP error responses
- Use `errors.Is` / `errors.As` for error type checking

### Testing
- Table-driven tests for all store operations
- Golden file tests for context builder output (testdata/*.golden)
- Integration tests for MCP server (actual stdio communication)
- Test fixtures in `testdata/` directories per package
- `go test ./... -count=1` — no test caching during dev

### Dependencies Policy
- Minimal dependencies — only what's genuinely needed
- No framework beyond Cobra for CLI
- Prefer stdlib over third-party where reasonable
- Pin all dependencies with go.sum

---

## Storage Format

### Frontmatter + Markdown
All artifacts use YAML frontmatter separated by `---` + markdown body:
- Frontmatter: structured, machine-readable metadata (parsed into Go structs)
- Body: human-written content (descriptions, specs, plans)
- Parse with adrg/frontmatter, roundtrip without losing content or formatting

### ID Format
- Prefix + ULID for time-sortability: `i_`, `e_`, `s_`, `d_`, `dec_`, `p_`, `x_`, `v_`
- IDs are immutable once assigned — slugs can change, IDs cannot
- ULIDs are sortable by creation time without needing a sequence counter

### File Layout
See IMPLEMENTATION_PLAN.md for the complete .specflow/ directory structure.

---

## MCP Server Contract

### Tool Naming
- Prefix all tools with `sf_`
- Use snake_case: `sf_story_show`, `sf_context_build`
- Read tools return markdown content
- Write tools return confirmation + updated entity summary

### Response Format
- MCP text content type for all responses
- Structured markdown for complex responses (status, context)
- YAML code blocks for structured data within markdown responses
- Keep responses focused — don't return the entire project state for a single story query

### Behavioral Tool Descriptions
MCP tool descriptions include behavioral instructions that tell Claude Code HOW to interact with the user when using that tool. These are not just API docs — they embed the working style from the "Who I Am" section. See IMPLEMENTATION_PLAN.md for full tool specifications.

---

## Related Documentation

- [Implementation Plan](./IMPLEMENTATION_PLAN.md) — Complete architecture, data models, CLI commands, MCP tools, templates, and build phases
