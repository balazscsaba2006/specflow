# specflow CLI Reference

## Global

```
specflow [command]
```

Spec-driven development CLI -- a structured memory and context layer for Claude Code.

specflow automatically locates the `.specflow/` directory by walking up from the current working directory.

### version

```
specflow version
```

Print version information (version, commit, build date).

---

## init

```
specflow init
```

Initialize a new specflow project. Creates the `.specflow/` directory structure and default config in the current directory. Also configures `.mcp.json` with the MCP server entry and installs the Claude Code workflow skill globally to `~/.claude/skills/specflow/SKILL.md`. Fails if `.specflow/` already exists.

---

## config

```
specflow config [subcommand]
```

Manage project configuration. Get, set, or list specflow project configuration values.

### config get

```
specflow config get <key>
```

Get a single config value by key. Prints the value to stdout.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `key` | yes | The configuration key to retrieve |

### config set

```
specflow config set <key> <value>
```

Set a config value and persist it to `.specflow/config.yaml`.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `key` | yes | The configuration key to set |
| `value` | yes | The value to assign |

### config ls

```
specflow config ls
```

List all config values. Outputs the full configuration as YAML.

### Configuration Keys

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `mode` | string | `careful` | Project mode: `fast` or `careful` |
| `conventions_file` | string | `CLAUDE.md` | Path to project conventions file (used by context builder) |
| `agents_file` | string | `""` | Path to agents file (used by context builder) |
| `default_priority` | string | `medium` | Default priority for new stories |

---

## initiative

```
specflow initiative [subcommand]
specflow i [subcommand]
```

Create, list, show, edit, and update initiatives.

**Alias:** `i`

### initiative new

```
specflow initiative new <slug>
```

Create a new initiative. Opens `$EDITOR` with a template for editing frontmatter and body content. The slug must be valid (lowercase, hyphens, no spaces).

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Unique slug identifier for the initiative |

### initiative ls

```
specflow initiative ls [flags]
```

List all initiatives. Outputs a table with columns: SLUG, TITLE, STATUS.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--include-archived` | bool | `false` | Include archived initiatives |

### initiative show

```
specflow initiative show <slug>
```

Show full details of an initiative, including ID, slug, title, status, goal, timestamps, success criteria, open questions, linked epics, and body content.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the initiative to display |

### initiative edit

```
specflow initiative edit <slug>
```

Open an existing initiative in `$EDITOR` for full editing. Preserves immutable fields (ID, slug, created timestamp) after saving.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the initiative to edit |

### initiative set

```
specflow initiative set <slug> <field> <value>
```

Quick-update a single field on an initiative without opening an editor.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the initiative to update |
| `field` | yes | Field to update: `status`, `title`, or `goal` |
| `value` | yes | New value for the field |

**Valid statuses:** `active`, `completed`, `on_hold`, `cancelled`, `archived`

### initiative archive

```
specflow initiative archive <slug> [flags]
```

Archive an initiative. Moves the initiative to `.specflow/archive/initiatives/`. All linked epics must be archived or completed unless `--force` is used. Body is preserved by default.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the initiative to archive |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Archive even if initiative/epics aren't in completed/archived status |
| `--compact` | bool | `false` | Strip markdown body (compact to frontmatter-only tombstone) |

---

### initiative unarchive

```
specflow initiative unarchive <slug>
```

Restore an archived initiative back to active state. Moves it from `.specflow/archive/initiatives/` back to `.specflow/initiatives/`, setting status to `on_hold`. Does not automatically unarchive linked epics.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the initiative to unarchive |

---

## epic

```
specflow epic [subcommand]
specflow e [subcommand]
```

Create, list, show, edit, and update epics.

**Alias:** `e`

### epic new

```
specflow epic new <slug> [flags]
```

Create a new epic. Opens `$EDITOR` with a template. If `--initiative` is provided, it is pre-filled in the template.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Unique slug identifier for the epic |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--initiative` | string | `""` | Parent initiative slug |

### epic ls

```
specflow epic ls [flags]
```

List epics. Outputs a table with columns: SLUG, TITLE, STATUS, INITIATIVE.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--initiative` | string | `""` | Filter by initiative slug |
| `--include-archived` | bool | `false` | Include archived epics in the listing |

### epic show

```
specflow epic show <slug>
```

Show full details of an epic, including ID, slug, title, status, initiative, fidelity, timestamps, phases with their stories, non-goals, open questions, decisions, and body content.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the epic to display |

### epic edit

```
specflow epic edit <slug>
```

Open an existing epic in `$EDITOR` for full editing. Preserves immutable fields (ID, slug, created timestamp) after saving.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the epic to edit |

### epic set

```
specflow epic set <slug> <field> <value>
```

Quick-update a single field on an epic without opening an editor.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the epic to update |
| `field` | yes | Field to update: `status`, `title`, or `initiative` |
| `value` | yes | New value for the field |

**Valid statuses:** `draft`, `active`, `completed`, `on_hold`, `cancelled`, `archived`

### epic archive

```
specflow epic archive <slug> [flags]
```

Archive a completed epic. Moves the epic tree to `.specflow/archive/` and moves execution directories. Bodies are preserved by default.

By default, requires the epic to have status `completed` and all stories to have status `done`. Use `--force` to bypass these checks.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the epic to archive |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Archive even if epic/stories aren't in completed/done status |
| `--compact` | bool | `false` | Strip markdown bodies (compact to frontmatter-only tombstones) |

---

### epic unarchive

```
specflow epic unarchive <slug>
```

Restore an archived epic back to active state. Moves the epic tree from `.specflow/archive/` back to `.specflow/epics/`, setting status to `on_hold`. Stories keep their original status. Docs and executions are moved back to their original locations.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the epic to unarchive |

---

## story

```
specflow story [subcommand]
specflow s [subcommand]
```

Create, list, show, edit, and update stories.

**Alias:** `s`

### story new

```
specflow story new <slug> [flags]
```

Create a new story. Opens `$EDITOR` with a template. The template varies by project mode: careful mode uses the full template with all fields; fast mode uses a minimal template with just title and acceptance criteria. If `--epic` is provided, it is pre-filled in the template.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Unique slug identifier for the story |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--epic` | string | `""` | Parent epic slug |

### story ls

```
specflow story ls [flags]
```

List stories. Outputs a table with columns: SLUG, TITLE, STATUS, PRIORITY, EPIC, LABELS. Multiple filters can be combined.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--epic` | string | `""` | Filter by epic slug |
| `--status` | string | `""` | Filter by status |
| `--label` | string | `""` | Filter by label |
| `--blocked` | bool | `false` | Only show stories with non-empty `blocked_by` |
| `--include-archived` | bool | `false` | Include stories from archived epics |

### story show

```
specflow story show <slug> [flags]
```

Show full details of a story, including ID, slug, title, status, priority, epic, fidelity, timestamps, blocked-by list, labels, acceptance criteria, doc refs, non-goals, open questions, assumptions, and body content.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the story to display |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--epic` | string | `""` | Epic slug (for epic-scoped stories) |

### story edit

```
specflow story edit <slug> [flags]
```

Open an existing story in `$EDITOR` for full editing. Preserves immutable fields (ID, slug, created timestamp) after saving.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the story to edit |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--epic` | string | `""` | Epic slug (for epic-scoped stories) |

### story set

```
specflow story set <slug> <field> <value> [flags]
```

Quick-update a single field on a story without opening an editor. Status changes are validated against allowed transitions.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the story to update |
| `field` | yes | Field to update: `status`, `priority`, or `title` |
| `value` | yes | New value for the field |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--epic` | string | `""` | Epic slug (for epic-scoped stories) |

**Valid statuses:** `draft`, `planned`, `in_progress`, `verifying`, `done`, `blocked`, `cancelled`

**Status transitions:**

| From | Allowed transitions |
|------|---------------------|
| `draft` | `planned`, `blocked`, `cancelled` |
| `planned` | `in_progress`, `blocked`, `cancelled` |
| `in_progress` | `verifying`, `done`, `blocked`, `cancelled` |
| `verifying` | `done`, `in_progress`, `blocked`, `cancelled` |
| `done` | `cancelled` |
| `blocked` | `draft`, `planned`, `in_progress`, `cancelled` |
| `cancelled` | `draft`, `planned` |

**Valid priorities:** `critical`, `high`, `medium`, `low`

### story next

```
specflow story next [flags]
```

Recommend the next story to work on. Filters to stories with status `planned` whose blockers (if any) are all `done`, then sorts by priority (critical > high > medium > low) and returns the top candidate.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--epic` | string | `""` | Scope to a specific epic |

### story archive

```
specflow story archive <slug> [flags]
```

Archive a standalone story. Moves the story file to `.specflow/archive/stories/` and moves execution directories to `.specflow/archive/executions/`. Body is preserved by default.

By default, requires the story to have status `done`. Use `--force` to bypass. Only works on standalone stories (not epic-scoped).

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the standalone story to archive |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Archive even if story isn't in done status |
| `--compact` | bool | `false` | Strip markdown body (compact to frontmatter-only tombstone) |

---

### story unarchive

```
specflow story unarchive <slug>
```

Restore an archived standalone story back to active state. Moves it from `.specflow/archive/stories/` back to `.specflow/stories/`, setting status to `planned`. Executions are moved back to their original location.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the standalone story to unarchive |

---

## doc

```
specflow doc [subcommand]
specflow d [subcommand]
```

Create, list, show, and edit documents (PRDs, tech specs, ADRs, etc.).

**Alias:** `d`

### doc new

```
specflow doc new <slug> --type <type> [flags]
```

Create a new document. Opens `$EDITOR` with a type-specific template. Every supported type (PRD, tech-spec, api-spec, design-spec, ADR, one-pager) has a dedicated template with relevant sections. Unknown types fall back to a generic template. The `--type` flag is required.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Unique slug identifier for the document |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | `""` | Document type (required): `prd`, `tech-spec`, `api-spec`, `design-spec`, `adr`, `one-pager` |
| `--epic` | string | `""` | Parent epic slug |

### doc ls

```
specflow doc ls [flags]
```

List documents. Outputs a table with columns: SLUG, TYPE, TITLE, STATUS, EPIC. Filters can be combined.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--epic` | string | `""` | Filter by epic slug |
| `--type` | string | `""` | Filter by document type |

### doc show

```
specflow doc show <slug> [flags]
```

Show full details of a document, including ID, slug, type, title, status, epic, timestamps, open questions, and body content.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the document to display |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--epic` | string | `""` | Epic slug (for epic-scoped docs) |

### doc edit

```
specflow doc edit <slug> [flags]
```

Open an existing document in `$EDITOR` for full editing. Preserves immutable fields (ID, slug, created timestamp) after saving.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the document to edit |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--epic` | string | `""` | Epic slug (for epic-scoped docs) |

---

## decision

```
specflow decision [subcommand]
specflow dec [subcommand]
```

Create, list, and show lightweight decision records.

**Alias:** `dec`

Decisions are quick, lightweight choice records that live in `.specflow/decisions/`. For formal Architecture Decision Records with epic scope, open questions, and status tracking, use `specflow doc new --type adr` instead.

### decision new

```
specflow decision new <slug>
```

Create a new decision. Opens `$EDITOR` with a template containing Context, Alternatives Considered, Decision, and Consequences sections. The date is auto-set to today if not specified in the frontmatter.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Unique slug identifier for the decision |

### decision ls

```
specflow decision ls
```

List decisions. Outputs a table with columns: SLUG, DATE, TITLE, STATUS.

### decision show

```
specflow decision show <slug>
```

Show full details of a decision, including ID, slug, date, title, status, context refs, and body content.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the decision to display |

**Valid statuses:** `proposed`, `accepted`, `superseded`, `deprecated`

---

## status

```
specflow status [slug]
```

Show project or entity status.

Without arguments, shows an aggregate status rollup across all epics and stories with a table (columns: EPIC, STATUS, STORIES, DONE, PROGRESS) and a summary of standalone stories.

With a slug argument, auto-detects the entity type (initiative, epic, or story) and shows its details.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | no | Entity slug to show detail for (auto-detects type) |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--include-archived` | bool | `false` | Include archived epics in the project-wide status rollup |

---

## questions

```
specflow questions
```

List all open questions across the project. Walks all initiatives, epics, and stories to collect open questions, grouped by source.

Outputs a table with columns: SOURCE, QUESTION. Prints total count at the bottom.

---

## blocked

```
specflow blocked
```

List all blocked stories. Shows stories that have non-empty `blocked_by` where at least one blocker is not done.

Outputs a table with columns: SLUG, TITLE, STATUS, EPIC, BLOCKED BY. Each blocker is annotated with its current status.

---

## assumptions

```
specflow assumptions
```

List all assumptions across stories, grouped by epic. Walks all stories and collects the `assumptions` field.

Outputs a table with columns: EPIC, STORY, ASSUMPTION. Prints total count at the bottom.

---

## log

```
specflow log [flags]
```

Show the activity log. Reads the most recent entries from `log.jsonl` and displays them as a timeline.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--last`, `-n` | int | `20` | Number of recent entries to show |

### Event Types

| Event | Fields Shown |
|-------|-------------|
| `story.status_changed` | entity, from/to status |
| `execution.started` | story, git ref |
| `execution.completed` | story, files changed, git ref |
| `verification.saved` | story, result, severity counts |
| Other events | type, entity, optional epic/story context |

---

## search

```
specflow search <query> [flags]
```

Full-text search across all specflow artifacts. Performs case-insensitive search through all `.md` files under `.specflow/`.

Results are grouped by file with matching lines highlighted and context lines shown around each match.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `query` | yes | The search string |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--context`, `-C` | int | `1` | Number of context lines around each match |

---

## import

```
specflow import <file> [flags]
```

Import an existing markdown file as a specflow artifact.

If the file has YAML frontmatter, its fields are used. If not, metadata is generated from the filename (the filename is converted to a slug).

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `file` | yes | Path to the markdown file to import |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | `story` | Artifact type: `story`, `doc`, `epic`, `initiative`, `decision` |
| `--epic` | string | `""` | Parent epic slug (for stories and docs) |

---

## mode

```
specflow mode [fast|careful]
```

Show or set the project mode.

Without arguments, shows the current mode. With an argument, sets the mode and persists it to config.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `mode` | no | Mode to set: `fast` or `careful` |

### Mode Differences

| Aspect | Fast | Careful (default) |
|--------|------|-------------------|
| Story templates | Title + acceptance only | Full template with all fields |
| Hard questions | Suppressed via MCP | Always included |
| Verification prompt | Only on explicit request | Prompted after every execution |

---

## template

```
specflow template [subcommand]
specflow tmpl [subcommand]
```

Manage specflow templates. List, override, and reset templates.

**Alias:** `tmpl`

### template ls

```
specflow template ls
```

List all available templates and whether a project override exists for each. Outputs a table with columns: TEMPLATE, OVERRIDE.

### template override

```
specflow template override <name>
```

Copy an embedded default template to `.specflow/templates/` for per-project customization.

For doc types, use the short name (e.g., `tech-spec` instead of `doc_tech-spec`). For entity types, use the name directly (e.g., `story`, `epic`, `initiative`).

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | yes | Template name: `story`, `story_fast`, `epic`, `initiative`, `decision`, `prd`, `tech-spec`, `api-spec`, `design-spec`, `adr`, `one-pager`, `generic` |

Note: The `skill` template is not overridable — it ships with the binary and is installed globally via `~/.claude/skills/specflow/SKILL.md`.

**Examples:**

```sh
specflow template override tech-spec    # → .specflow/templates/doc_tech-spec.md
specflow template override story        # → .specflow/templates/story.md
```

### template reset

```
specflow template reset <name>
```

Delete a template override, reverting to the embedded default.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | yes | Template name (same values as `override`) |

---

## update

```
specflow update [flags]
```

Download the latest release from GitHub and replace the current binary.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Re-download even if already on the latest version |

---

## export

```
specflow export [slug] [flags]
```

Export any specflow entity (initiative, epic, story, doc, decision) to markdown, HTML, or YAML. Auto-detects entity type from the slug. Use `--all` to export the entire project hierarchy.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-f`, `--format` | string | `md` | Output format: `md`, `html`, `yaml` |
| `-o`, `--output` | string | `<slug>-export.<ext>` | Output file path |
| `-t`, `--tree` | bool | `false` | Include full subtree (children, docs, decisions) |
| `--all` | bool | `false` | Export entire project hierarchy |
| `--no-body` | bool | `false` | Omit markdown body content |
| `--exclude-status` | string | `""` | Comma-separated statuses to exclude (e.g. `done,cancelled`). `--exclude-done` still works as a deprecated hidden alias. |

### Examples

```bash
# Export epic as markdown (backward compatible)
specflow export user-auth
# → Exported 1 epic (user-auth) to user-auth-export.md

# Export epic with full subtree (stories, docs, decisions)
specflow export user-auth --tree

# Export as self-contained HTML with Mermaid diagrams and code highlighting
specflow export user-auth --tree --format html

# Export entire project as HTML
specflow export --all --format html

# Export as YAML
specflow export user-auth --tree --format yaml

# Export without body content, excluding done and cancelled stories
specflow export user-auth -o ~/exports/auth.md --exclude-status done,cancelled --no-body
```

---

## sync

```
specflow sync
```

Update the globally installed Claude Code skill (`~/.claude/skills/specflow/SKILL.md`) from the embedded template in the current binary. The skill is also synced automatically on `specflow update`. Use this for manual reset after editing or to verify the installed version matches the binary.

---

## mcp

```
specflow mcp
```

Start the MCP server on stdio for Claude Code integration. This command is typically not run directly -- instead, it is configured via `specflow init` which sets up the MCP server entry in `.mcp.json`.

The MCP server exposes 33 tools prefixed with `sf_` for reading and writing specflow artifacts. See the [MCP Tools Reference](mcp-tools.md) for the full tool catalog.
