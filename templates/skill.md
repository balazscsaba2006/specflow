---
name: specflow
description: Use when implementing stories, creating artifacts, or doing spec-driven development with specflow MCP tools.
---

# specflow Workflow

You have access to specflow MCP tools (`sf_*`) for managing development artifacts (initiatives, epics, stories, docs, decisions, plans, executions). This skill defines how to use them together as a self-contained workflow.

## Discovery Phase

**When a user describes a new feature, system change, or problem — DO NOT immediately create specflow artifacts.** Artifacts are the OUTPUT of discovery, not the starting point.

### When Discovery Applies
- User describes a feature, problem, or system change in natural language
- User shares context about what they want to build
- User asks to "plan", "design", or "figure out" something

### When to Skip Discovery
- User explicitly asks to create a specific artifact ("create an epic for X")
- User is continuing work on an already-planned story
- The task is a trivial fix with obvious scope

### Discovery Flow

Follow this sequence — do not skip steps:

1. **Explore** — Read relevant code, understand current state, identify constraints.
2. **Question** — Ask questions **one at a time** via `AskUserQuestion`. Group related questions in batches of 1-4. Do NOT dump a markdown list of questions — use the tool so the user can answer interactively.
3. **Propose** — Present 2-3 approaches with trade-offs and your recommendation. Be opinionated but let the user choose.
4. **Iterate** — Refine based on feedback. Go back and forth as needed.
5. **Align** — Present final design summary, get **explicit user approval**.
6. **Create artifacts** — Only after approval. Break down into specflow hierarchy:
   - Create initiative (if scope warrants) via `sf_initiative_create`
   - Create epic(s) via `sf_epic_create`
   - Create stories with acceptance criteria via `sf_story_create` (status: `draft`)
   - Transition stories to `planned` via `sf_story_update(status="planned")` — don't leave in `draft` unless genuinely incomplete
   - Optionally create docs via `sf_doc_write` (scoped to epic)

**Hard gate: NO artifacts until design is approved.** "Let's build X" is the START of discovery.

When work has natural phases, create **separate epics per phase** with stories scoped to each.

**Do NOT save to `docs/plans/`.** Discovery output becomes specflow artifacts directly.

### Resolving Open Questions

When open questions exist — from discovery, `sf_questions`, or the user asking to resolve them:
- Use `AskUserQuestion` to ask interactively (batches of 1-4, grouped by topic)
- After each batch, resolve via `sf_question_resolve`
- Continue until all resolved before creating artifacts or starting implementation

## Artifact Guidelines

### Epic vs Standalone Story

**Create an epic** when: 3+ stories with dependencies, a PRD/spec drives the work, or phased delivery needed.

**Use a standalone story** when: single well-scoped unit (bug fix, small feature, refactor) with self-contained acceptance criteria. Lives in `.specflow/stories/`. Full lifecycle still applies.

### Hard Questions

- In **careful mode**: call `sf_hard_questions` before finalizing artifacts. Review with user and incorporate answers.
- In **fast mode**: skip unless user asks.
- Always fill `open_questions` for anything uncertain rather than making silent assumptions.

### Story Status Lifecycle

```
draft → planned → in_progress → done
```

- `sf_story_create` → `draft`
- `sf_story_update(status="planned")` → ready for implementation
- `sf_execution_start` → auto-sets `in_progress` (only from `planned` or `in_progress`)
- `sf_story_update(status="done")` → complete

The state machine enforces valid transitions. Always check and transition status before starting execution.

## Planning Phase

**CRITICAL: Never start implementation without user approval of the plan.**

### Flow

1. **Enter plan mode** via `EnterPlanMode`.
2. **Design the plan** — explore codebase, design approach, write full plan to plan file using this format:

```
## Plan: [Story Title]

### Tasks
1. [ ] **Task name** (~2-5 min)
   - File: `path/to/file.go`
   - What: concise description of the change
   - Test: `go test ./internal/pkg/ -run TestName`
   - Expected: what passing looks like

2. [ ] **Next task** (~2-5 min)
   ...

### Verification
- Command: `go test ./... -count=1`
- Acceptance criteria check: [list from story]
```

Each task should be a bite-sized unit (2-5 min), with exact file paths, exact commands, and expected output. TDD style: write test first where applicable.

3. **Exit plan mode** via `ExitPlanMode` — user sees the full plan and can approve, request changes, or reject.
4. **After approval:** save to specflow via `sf_plan_save` — persists for `sf_context_build`, `sf_scope_drift`, and verification across sessions.
5. **Proceed** to execution.

### Dual Storage

- **Claude plan file** — visible for approval, survives context compaction. Ephemeral (current session). Written BEFORE approval.
- **specflow plan** (via `sf_plan_save`) — persistent across sessions. Powers `sf_context_build` (Layer 4), `sf_scope_drift`, and verification. Written AFTER approval.

### When to Skip Plan Mode

- Trivial change (< 5 lines, obvious fix)
- User explicitly says "just do it" or "skip the plan"

This gate applies to ALL non-trivial stories — epic-scoped and standalone.

## Execution Phase

**Pre-conditions:** Plan approved. Story in `planned` status.

### Starting Work

1. **Identify the story.** Use `sf_story_next` to get the highest-priority unblocked story, or work on the story the user specifies.
2. **Build context.** ALWAYS call `sf_context_build` before starting implementation. This assembles conventions, epic context, PRD content, plan, and open items into a single layered prompt. Read the full response — it contains everything you need.
3. **Ensure story is `planned`.** Stories are created in `draft` status. Before implementation can start, the story must be in `planned`. If the story is in `draft`, transition it: `sf_story_update(slug, status="planned")`. The state machine enforces `draft → planned → in_progress` — you cannot skip `planned`.
4. **Check gates before implementing:**
   - **Open questions?** Resolve them with the user. Use `AskUserQuestion` to ask interactively, then update via `sf_question_resolve`. Do not proceed with unresolved questions that affect implementation.
   - **Blockers?** If `blocked_by` stories aren't done, surface this to the user. Don't work around blockers silently.
   - **No plan?** Enter plan mode (`EnterPlanMode`), design the plan, get user approval via `ExitPlanMode`, then save with `sf_plan_save` (see Planning Phase above).

### Implementing

1. **Call `sf_execution_start`.** This records the git baseline and auto-sets the story to `in_progress`. **Note the execution_id from the response** — you'll need it for `sf_execution_complete`.
2. **Create TodoWrite checklist** from the plan tasks for visual progress tracking. Each plan task becomes a todo item.
3. **Implement in batches of ~3 tasks.** After each batch:
   - Run relevant tests to verify the batch
   - Update TodoWrite items as completed
   - Report progress to user briefly
   - Stop on blockers — don't push through silently. Use `sf_blocked` if a blocker is discovered.
4. **Run full test suite** and check each acceptance criterion from the story.
5. **After verification passes:** complete the lifecycle, then commit everything together (see Completion Phase).

**Never leave a story in `in_progress` at the end of a session.** Either complete the full cycle or use `sf_execution_pause` with handover notes describing what was done, what remains, and any blockers.

### Resuming Paused Work

When `sf_execution_pause` was used previously:
1. Call `sf_context_build` — surfaces handover notes (what was done, what remains, blockers).
2. Call `sf_execution_start` — creates new execution. Paused execution stays as history.
3. Continue from where handover notes left off.

### Retroactive Completion

When work is already done (committed before the story was created, or done in a previous session without lifecycle tracking):

1. **Transition through statuses** — you must still follow valid transitions: `draft → planned`, then `planned → in_progress`, then `in_progress → done`. Call `sf_story_update` for each step.
2. **Skip execution/verification** — no need to create an execution or verify if the work is already committed and tested.
3. **Don't pretend** — this is bookkeeping. The point is to get the status accurate, not to retroactively fabricate an execution trail.

## Completion Phase

### Verification

**Evidence before claims.** Run tests, check each acceptance criterion, demonstrate correctness. A story is NOT done until verified.

### Background Lifecycle (Verification Passes)

To reduce terminal noise, batch the post-implementation lifecycle calls into a **single background agent**. Complete the specflow lifecycle **before committing**, so code changes and specflow metadata (execution, verification, story status) land in a single commit.

Launch a single background agent (subagent_type: `general-purpose`, run_in_background: true) with a prompt that instructs it to run these three calls in sequence. Include ALL required parameters in the prompt — the background agent has no conversation context:

```
Complete specflow lifecycle for story "{slug}":
1. Call sf_execution_complete with execution_id="{exec_id}" and story="{slug}"
2. Call sf_verify_save with story="{slug}", result="pass", summary="{summary}",
   acceptance_check=[{"criteria":"first criterion text", "met":true},
                     {"criteria":"second criterion text", "met":true}, ...]
3. Call sf_story_update with slug="{slug}", status="done"
```

**A story is NOT done until the background task completes successfully.** After it completes, commit all changes (code + `.specflow/`) together.

### Foreground Lifecycle (Verification Fails)

Do NOT background — keep visible to user:
1. `sf_execution_complete` (foreground)
2. `sf_verify_save` with findings (foreground)
3. Discuss failures with user before deciding next steps

### Committing

Commit code changes and `.specflow/` metadata together in a single commit. This avoids a two-commit dance when `.specflow/` is tracked in git.

## Working Style

- **Present trade-offs, don't decide unilaterally.** Lay out options with pros/cons and let the user choose.
- **Stop on contradictions.** If context reveals conflicting requirements, surface immediately. Don't pick a side.
- **Record decisions.** Non-trivial technical choices → `sf_decision_record`.
- **Check scope when uncertain.** Use `sf_scope_check` against the PRD. Don't gold-plate.
- **Use assumptions field.** Anything assumed but not verified → story's assumptions list.
- **Log significant events.** Use `sf_log` for important milestones, decisions, or deviations.

## Exporting to Jira

When the user asks to export an epic (or stories) to Jira, follow this workflow. Requires the Atlassian MCP server to be configured.

### Prerequisites

Check that Atlassian MCP tools are available (`getAccessibleAtlassianResources`, `createJiraIssue`, etc.). If not available, tell the user to configure the Atlassian MCP server and fall back to markdown export via `specflow export <epic-slug>`.

### Export Flow

1. **Get export data.** Call `sf_export` with the epic slug (or story slug for standalone stories). Review the YAML output to understand what will be created.

2. **Get Jira target.**
   - Call `getAccessibleAtlassianResources` to get the `cloudId`.
   - Call `getVisibleJiraProjects` to list available projects.
   - Ask the user which project to export to via `AskUserQuestion`.

3. **Confirm before creating.** Show the user a summary of what will be created:
   - 1 Epic issue + N Story issues (or just 1 Story for standalone)
   - Target project
   - Ask for confirmation before proceeding.

4. **Create the Jira epic.**
   - Use `createJiraIssue` with `issueTypeName: "Epic"`.
   - Map: `title → summary`, `description (body + acceptance) → description`.
   - Note the returned issue key (e.g., `PROJ-123`).

5. **Create stories as children.**
   - For each story in the export, call `createJiraIssue` with:
     - `issueTypeName: "Story"` (or ask user for preferred type)
     - `parent: <epic-issue-key>`
     - `summary: story.title`
     - `description: story.description` (pre-assembled by sf_export)
   - **Priority mapping:**

     | specflow | Jira |
     |----------|------|
     | critical | Highest |
     | high | High |
     | medium | Medium |
     | low | Low |

   - If a story fails, log the error and continue with remaining stories. Do NOT stop on first failure.

6. **Report results.** After all issues are created, show:
   - Each created issue with its Jira key and link
   - Any failures with error details
   - Summary: "Created 1 epic + N stories in PROJECT"

### Markdown Export Fallback

If the user prefers markdown export (or Atlassian MCP is unavailable):
- Run `specflow export <epic-slug>` via Bash, or
- Call `sf_export` and write the YAML data to a file directly.

### Edge Cases

- **Duplicate export:** Warn the user if they're exporting an epic that may have already been exported (ask before proceeding).
- **Empty epic:** Create the Jira epic without child stories. Inform the user.
- **Standalone story export:** Use `sf_export` with `story` parameter instead of `epic`. Create a single Jira Story (no parent epic).
