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
specflow init [flags]
```

Initialize a new specflow project. Creates the `.specflow/` directory structure and default config in the current directory. Fails if `.specflow/` already exists.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--with-claude` | bool | `false` | Also configure `.claude/settings.json` with the specflow MCP server entry |

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
specflow initiative ls
```

List all initiatives. Outputs a table with columns: SLUG, TITLE, STATUS.

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

**Valid statuses:** `active`, `completed`, `on_hold`, `archived`

### initiative archive

```
specflow initiative archive <slug>
```

Archive an initiative. Shortcut for `specflow initiative set <slug> status archived`.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the initiative to archive |

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

### epic show

```
specflow epic show <slug>
```

Show full details of an epic, including ID, slug, title, status, initiative, timestamps, phases with their stories, open questions, decisions, and body content.

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

**Valid statuses:** `draft`, `active`, `completed`, `on_hold`, `archived`

### epic archive

```
specflow epic archive <slug>
```

Archive an epic. Shortcut for `specflow epic set <slug> status archived`.

**Arguments:**

| Argument | Required | Description |
|----------|----------|-------------|
| `slug` | yes | Slug of the epic to archive |

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

Create a new story. Opens `$EDITOR` with a template. If `--epic` is provided, it is pre-filled in the template.

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

### story show

```
specflow story show <slug> [flags]
```

Show full details of a story, including ID, slug, title, status, priority, epic, timestamps, blocked-by list, labels, acceptance criteria, doc refs, open questions, assumptions, and body content.

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

**Valid statuses:** `draft`, `planned`, `in_progress`, `verifying`, `done`, `blocked`

**Status transitions:**

| From | Allowed transitions |
|------|---------------------|
| `draft` | `planned`, `blocked` |
| `planned` | `in_progress`, `blocked` |
| `in_progress` | `verifying`, `done`, `blocked` |
| `verifying` | `done`, `in_progress`, `blocked` |
| `done` | (none) |
| `blocked` | `draft`, `planned`, `in_progress` |

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

Create a new document. Opens `$EDITOR` with a type-specific template (PRDs and ADRs have structured templates; other types use a generic template). The `--type` flag is required.

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

Create, list, and show architectural decisions.

**Alias:** `dec`

### decision new

```
specflow decision new <slug>
```

Create a new decision. Opens `$EDITOR` with a template containing Context, Decision, and Consequences sections.

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
