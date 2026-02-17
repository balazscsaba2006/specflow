# Getting Started

This guide walks you through installing specflow, connecting it to Claude Code, and running your first spec-driven workflow.

---

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and working
- A project directory with git initialized

---

## Installation

### Option 1: Pre-built binary (recommended)

Download the latest release for your platform from [GitHub Releases](https://github.com/balazscsaba2006/specflow/releases).

**macOS / Linux:**

```sh
# Download (replace OS and ARCH with your platform, e.g. darwin/arm64, linux/amd64)
curl -Lo specflow.tar.gz https://github.com/balazscsaba2006/specflow/releases/latest/download/specflow_OS_ARCH.tar.gz

# Extract and move to PATH
tar xzf specflow.tar.gz
sudo mv specflow /usr/local/bin/
rm specflow.tar.gz
```

### Option 2: From source

Requires Go 1.24+:

```sh
go install github.com/balazscsaba2006/specflow/cmd/specflow@latest
```

### Verify

```sh
specflow version
```

---

## Setting Up in Your Project

From your project root:

```sh
specflow init --with-claude
```

This does two things:

1. **Creates `.specflow/`** -- the directory where all artifacts (epics, stories, docs, decisions, plans) are stored as markdown files with YAML frontmatter.

2. **Creates `.claude/settings.local.json`** -- registers specflow as an MCP server so Claude Code can use `sf_*` tools automatically.

The generated MCP entry looks like:

```json
{
  "mcpServers": {
    "specflow": {
      "command": "specflow",
      "args": ["mcp"]
    }
  }
}
```

**Important:** `specflow` must be on your `$PATH` for Claude Code to find it. If you installed to a non-standard location, use the full path in the settings file.

---

## Manual MCP Setup

If you already ran `specflow init` without `--with-claude`, or need to configure MCP manually:

1. Create `.claude/settings.local.json` in your project root (or edit the existing one):

```json
{
  "mcpServers": {
    "specflow": {
      "command": "specflow",
      "args": ["mcp"]
    }
  }
}
```

2. Restart Claude Code so it picks up the new MCP server.

3. Verify by asking Claude Code: *"What MCP tools do you have?"* -- you should see `sf_*` tools listed.

---

## Workflow Skill

`specflow init --with-claude` also installs a Claude Code skill to `.claude/skills/specflow/SKILL.md`. The skill encodes the spec-driven development workflow: when to build context, how to gate on open questions, when to verify, and how to record decisions. Without the skill, Claude Code has the MCP tools but no encoded knowledge of the expected workflow sequence.

**What the skill encodes:**
- Always call `sf_context_build` before starting implementation
- Gate on open questions and blockers before writing code
- Use `sf_hard_questions` before finalizing artifacts (in careful mode)
- Record assumptions and decisions during implementation
- Run verification after every execution

The skill source is embedded in the specflow binary at `templates/skill.md`. You can edit the installed skill directly at `.claude/skills/specflow/SKILL.md` to customize the workflow. To re-install the default skill, run `specflow init --with-claude` again (the skill file is idempotently overwritten).

See the [Claude Code skills docs](https://code.claude.com/docs/en/skills) for more on how skills work.

---

## Adopting specflow in an Existing Project

If you have an existing project with markdown specs, PRDs, or design docs:

```sh
# Initialize specflow in your project root
specflow init --with-claude

# Import existing markdown files as specflow artifacts
specflow import path/to/existing-prd.md --type prd --epic my-epic
specflow import path/to/design-doc.md --type tech-spec --epic my-epic
```

The `import` command detects YAML frontmatter in existing files. If present, it preserves the metadata. If not, it creates appropriate frontmatter based on the `--type` flag.

**Practical tips:**
- Start small -- create one epic for your current focus area
- Import only the docs that are actively relevant
- You don't need to model your entire project history; specflow is forward-looking

---

## Your First Workflow

Here's an end-to-end walkthrough from creating an epic to completing a story with Claude Code.

### 1. Create an epic

```sh
specflow epic new auth-system
```

This opens your `$EDITOR` with a template. Fill in the description, phases, and any open questions, then save.

### 2. Create a PRD

```sh
specflow doc new prd --type prd --epic auth-system
```

Write out the problem statement, target users, requirements, and scope.

### 3. Create stories

```sh
specflow story new jwt-middleware --epic auth-system
specflow story new api-key-store --epic auth-system --blocked-by jwt-middleware
```

Each story gets acceptance criteria, priority, and optional doc refs.

### 4. Check status

```sh
specflow status
```

Shows a progress rollup across all epics with story counts and completion bars.

### 5. Work with Claude Code

Now the MCP workflow kicks in. Tell Claude Code something like *"What should I work on next?"* and it will use specflow tools:

```
Claude calls sf_story_next()
  -> specflow returns "jwt-middleware" (highest priority, no blockers)

Claude calls sf_context_build("jwt-middleware")
  -> specflow assembles 6 layers: conventions, epic context, PRD, plan, files, open items

Claude calls sf_execution_start("jwt-middleware")
  -> specflow records the current git ref as baseline

... Claude implements the code ...

Claude calls sf_execution_complete("x_01ABC...")
  -> specflow captures the git ref after implementation and records changed files

Claude calls sf_verify_save("jwt-middleware", result="pass", ...)
  -> specflow stores verification results

Claude calls sf_story_update("jwt-middleware", status="done")
  -> story is marked complete, progress updates
```

### 6. Review status

```sh
specflow status --epic auth-system
```

You'll see `jwt-middleware` marked as done, and `api-key-store` is now unblocked and ready.

---

## Modes

specflow has two modes that control the level of ceremony:

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

**When to use fast mode:** Prototyping, small fixes, personal projects where you want less overhead.

**When to use careful mode:** Production features, complex multi-story epics, anything where you want verification and hard questions to catch blind spots.

---

## What Goes Where

### Entity Types

| Entity | Use case | Created via | Example |
|--------|----------|-------------|---------|
| **Initiative** | Strategic goal grouping multiple epics | `specflow initiative new` | "Launch v2.0" |
| **Epic** | A shippable feature or capability | `specflow epic new` | "Auth system", "Payment integration" |
| **Story** | Atomic work unit with acceptance criteria | `specflow story new` | "Create JWT middleware" |
| **Decision** | Lightweight choice record | `specflow decision new` | "Use JWT over sessions" |

### Document Types

Documents are formal artifacts with a status lifecycle (`draft` → `review` → `approved`) and can be scoped to an epic.

| Type | Use case | Created via | Example |
|------|----------|-------------|---------|
| **PRD** | Problem, users, goals, scope, requirements | `specflow doc new --type prd` | Product spec for auth system |
| **Tech Spec** | Architecture, data model, API design, constraints | `specflow doc new --type tech-spec` | How JWT tokens flow through the system |
| **API Spec** | Endpoint contracts, auth, rate limiting | `specflow doc new --type api-spec` | REST API v2 contract |
| **Design Spec** | Component design, interactions, trade-offs | `specflow doc new --type design-spec` | Token refresh design |
| **ADR** | Architecture Decision Record (formal) | `specflow doc new --type adr` | "Why JWT over session cookies" |
| **One-Pager** | Lightweight proposal | `specflow doc new --type one-pager` | "Should we add WebSocket support?" |

### Decision vs ADR

Both record choices, but at different levels of formality:

- **Decision** (`specflow decision new`) -- Quick, lightweight. Lives in `.specflow/decisions/`. No epic scope, no open questions. Use during planning or implementation for choices that need to be recorded but don't need formal review. Created via CLI or MCP (`sf_decision_record`).

- **ADR** (`specflow doc new --type adr`) -- Formal Architecture Decision Record. Lives in the docs hierarchy, can be scoped to an epic, tracks open questions, supports status workflow. Use for significant architectural choices that affect the system long-term and benefit from structured alternatives analysis.

---

## Next Steps

- [CLI Reference](cli-reference.md) -- Full command reference with flags and examples
- [MCP Tools Reference](mcp-tools.md) -- All `sf_*` tool parameters and response formats
- [Architecture](architecture.md) -- System design, data models, and storage format
