# MCP Tools Reference

specflow exposes MCP tools prefixed with `sf_` for Claude Code integration. Start the MCP server with `specflow mcp` or configure it via `specflow init --with-claude`.

All tools return MCP text content. Read tools return markdown-formatted responses. Write tools return confirmation messages with entity summaries.

---

## Read Tools

### sf_status

Show project status with progress per epic.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `scope` | string | no | Epic slug to scope status to, or empty for project-wide |
| `include_archived` | bool | no | Include archived epics in the status rollup |

**Response:** Markdown table with per-epic breakdown (stories, done count, progress), plus standalone stories summary. When `include_archived` is true, adds a separate archived epics section.

---

### sf_initiative_show

Show initiative details including goal, success criteria, linked epics, and open questions.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Initiative slug |

**Response:** Full initiative detail with all fields and body content.

---

### sf_epic_show

Show epic details including phases, story statuses, and open questions.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Epic slug |

**Response:** Epic detail with phase map showing each story's status, open questions, and decisions.

---

### sf_story_show

Show story details including acceptance criteria, doc refs, assumptions, and open questions.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Story slug |

**Response:** Full story detail with all fields and body content. Auto-searches across all epics and standalone stories.

---

### sf_story_next

Suggest the next story to work on. Returns the highest-priority planned story with no unresolved blockers.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `epic` | string | no | Filter to stories under this epic |

**Response:** Recommended story with title, slug, priority, and acceptance criteria.

---

### sf_story_ls

List stories with optional filters.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `epic` | string | no | Filter by epic slug |
| `status` | string | no | Filter by status |
| `label` | string | no | Filter by label |
| `blocked` | bool | no | Only show stories with blockers |
| `include_archived` | bool | no | Include stories from archived epics |

**Response:** Markdown table of matching stories with slug, title, status, priority, epic, and labels.

---

### sf_doc_read

Read a document by slug. Returns frontmatter summary and full body.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Document slug |
| `epic` | string | no | Epic slug (for epic-scoped docs) |

**Response:** Full document content with metadata header and body.

---

### sf_plan_read

Read the implementation plan for a story.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `story` | string | yes | Story slug |

**Response:** Plan content with metadata (ID, status, estimated files) and body.

---

### sf_verify_read

Read the latest verification result for a story.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `story` | string | yes | Story slug |

**Response:** Verification result with findings summary, acceptance check results, and severity stats.

---

### sf_context_build

Build the full 6-layer assembled context for a story. This is the core value of specflow -- use before starting implementation.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `story` | string | yes | Story slug |

**Response:** Multi-section markdown document with:

1. **Project Conventions** -- CLAUDE.md, AGENTS.md, config
2. **Initiative/Epic Context** -- goal, phase map, sibling stories, decisions
3. **Spec Requirements** -- referenced documents (full content), acceptance criteria
4. **Implementation Plan** -- approved plan or "no plan yet" prompt
5. **Referenced Files** -- files from plan, pattern exemplars from config
6. **Open Items** -- open questions, assumptions, blockers

---

### sf_questions

List all open questions across initiatives, epics, stories, and docs, grouped by source.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `epic` | string | no | Filter to questions from this epic and its stories |

**Response:** Questions grouped by source entity with source type annotation.

---

### sf_blocked

List all stories that have unresolved blockers.

No parameters.

**Response:** Markdown table of blocked stories with their blockers and blocker statuses.

---

### sf_decisions

List all decisions, optionally filtered by epic context ref.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `epic` | string | no | Filter to decisions referencing this epic |

**Response:** Markdown list of decisions with date, title, status, and body excerpt.

---

### sf_log

Show recent activity log entries.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `last` | int | no | Number of entries to show (default: 20) |

**Response:** Timeline of recent events with timestamps and event-specific details.

---

### sf_assumptions

List all assumptions across stories, grouped by epic.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `epic` | string | no | Filter to stories under this epic |

**Response:** Assumptions grouped by epic with story attribution.

---

### sf_diff

Git diff for a story's execution or between explicit refs.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `story` | string | no | Story slug (uses latest execution's git refs) |
| `refs` | string | no | Explicit ref range, e.g. `abc123..HEAD` |

Provide either `story` or `refs`. If `story` is provided, the diff is computed between the execution's start and end git refs.

**Response:** Git diff output.

---

### sf_hard_questions

Contextual hard questions for any entity. Returns deterministic, template-based questions to challenge thinking before finalizing an artifact.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `entity` | string | no | Any entity slug (initiative, epic, story, or doc) |

In fast mode, hard questions are suppressed.

**Response:** List of challenging questions tailored to the entity type (initiative strategy, epic scope, story implementation, PRD requirements, tech spec architecture).

---

### sf_review_prompt

Coaching/review prompt for a document. Returns a structured review prompt with the document content embedded.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `doc` | string | no | Document slug |
| `epic` | string | no | Epic slug (for epic-scoped docs) |

**Response:** Review prompt tailored to the document type (PRD review, tech spec review, or general decompose prompt).

---

### sf_scope_check

Cross-reference stories against PRD scope. Flags stories that aren't traceable to PRD user stories or are explicitly out-of-scope.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `epic` | string | yes | Epic slug |

**Response:** Scope analysis showing matched stories, unmatched stories, and potential out-of-scope items.

---

### sf_diff_check

Detect drift between specs and stories. Checks if documents were updated more recently than stories that reference them.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `epic` | string | yes | Epic slug |

**Response:** Drift report showing stories that may need updating based on more recently modified documents.

---

## Write Tools

### sf_initiative_create

Create a new initiative.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Unique slug identifier |
| `title` | string | yes | Initiative title |
| `goal` | string | yes | Strategic goal statement |
| `success_criteria` | []string | no | Measurable success conditions |
| `open_questions` | []string | no | Initial open questions |
| `body` | string | no | Markdown body content |

**Response:** Confirmation with title, slug, and generated ID.

---

### sf_epic_create

Create a new epic.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Unique slug identifier |
| `title` | string | yes | Epic title |
| `initiative` | string | no | Parent initiative slug |
| `phases` | []Phase | no | Phases with label and story slugs |
| `body` | string | no | Markdown body content |
| `open_questions` | []string | no | Initial open questions |

**Phase structure:** `{"label": "Phase 1", "stories": ["story-slug-1", "story-slug-2"]}`

**Response:** Confirmation with title, slug, and generated ID.

---

### sf_story_create

Create a new story.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Unique slug identifier |
| `title` | string | yes | Story title |
| `acceptance` | []string | yes | Acceptance criteria |
| `epic` | string | no | Parent epic slug |
| `priority` | string | no | Priority: `critical`, `high`, `medium` (default), `low` |
| `labels` | []string | no | Freeform labels |
| `blocked_by` | []string | no | Story slugs that block this story |
| `doc_refs` | []string | no | Document slugs referenced by this story |
| `open_questions` | []string | no | Initial open questions |
| `body` | string | no | Markdown body content |

**Response:** Confirmation with title, slug, ID, priority, and status.

---

### sf_story_update

Update an existing story. Only provided fields are updated. Assumptions and open_questions are appended (not replaced).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Story slug |
| `status` | string | no | New status (validated against transition rules) |
| `priority` | string | no | New priority |
| `labels` | []string | no | Replace labels |
| `blocked_by` | []string | no | Replace blocked_by list |
| `assumptions` | []string | no | Append to assumptions |
| `open_questions` | []string | no | Append to open_questions |

Auto-searches across all epics and standalone stories.

**Response:** Confirmation with updated status and priority.

---

### sf_doc_write

Create or update a document. If a doc with the given slug exists, it is updated; otherwise a new doc is created.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Document slug |
| `type` | string | yes | Document type: `prd`, `tech-spec`, `api-spec`, `design-spec`, `adr`, `one-pager` |
| `title` | string | yes | Document title |
| `body` | string | yes | Markdown body content |
| `epic` | string | no | Parent epic slug |
| `open_questions` | []string | no | Open questions |

**Response:** Confirmation with title, slug, ID, and type.

---

### sf_decision_record

Record an architectural or design decision. The body is assembled from context, decision, and consequences sections.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Unique slug identifier |
| `title` | string | yes | Decision title |
| `context` | string | yes | Context section (problem/situation) |
| `decision` | string | yes | Decision section (what was decided) |
| `consequences` | string | yes | Consequences section (implications) |
| `context_refs` | []string | no | Related epic/doc slugs |

**Response:** Confirmation with title, slug, ID, and status.

---

### sf_plan_save

Save an implementation plan for a story.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `story` | string | yes | Story slug |
| `content` | string | yes | Plan content (markdown) |
| `status` | string | no | Plan status: `draft` (default), `approved`, `executing`, `verified` |

**Response:** Confirmation with story slug, plan ID, and status.

---

### sf_execution_start

Start a new execution for a story. Captures current git ref as baseline, creates an execution record, and sets story status to `in_progress`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `story` | string | yes | Story slug |

**Response:** Confirmation with story slug and execution ID.

---

### sf_execution_complete

Mark an execution as completed. Captures current git ref, computes file changes since execution start.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `execution_id` | string | yes | Execution ID (from `sf_execution_start`) |
| `story` | string | no | Story slug (avoids scanning all stories if provided) |

In careful mode, the response includes a prompt to run verification.

**Response:** Confirmation with execution ID, story slug, and files changed count.

---

### sf_verify_save

Save verification results for a story's latest execution.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `story` | string | yes | Story slug |
| `result` | string | yes | Overall result: `pass`, `fail`, `partial` |
| `summary` | string | yes | Verification summary text |
| `findings` | []Finding | no | Individual findings |
| `acceptance_check` | []AcceptanceCheck | no | Per-criteria pass/fail |
| `assumptions` | []string | no | Assumptions discovered |

**Finding structure:** `{"severity": "critical|major|minor", "category": "missing|bug|performance|security|clarity|quality", "file": "path/to/file.go", "description": "...", "suggestion": "..."}`

**AcceptanceCheck structure:** `{"criteria": "User can log in with JWT", "met": true}`

**Response:** Confirmation with verification ID, result, and severity stats.

---

### sf_question_resolve

Resolve an open question on any entity by removing it from the `open_questions` list.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `entity` | string | yes | Entity slug (initiative, epic, story, or doc) |
| `question` | string | yes | Exact text of the question to resolve |
| `answer` | string | yes | Answer/resolution |

Auto-detects entity type by trying initiative, epic, story, project-level doc, then epic-scoped doc.

**Response:** Confirmation with entity type, question, and answer.
