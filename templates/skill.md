---
name: specflow
description: Use when implementing stories, creating artifacts, or doing spec-driven development with specflow MCP tools.
---

# specflow Workflow

You have access to specflow MCP tools (`sf_*`) for managing development artifacts. This skill defines how to use them together.

## Integration with superpowers Skills

**specflow is the source of truth for planning artifacts.** When both specflow and superpowers skills are available:

### Planning (`superpowers:writing-plans`)

When the user asks to plan a multi-story feature or the writing-plans skill is invoked:
1. **Create the epic and stories in specflow** via `sf_epic_create` and `sf_story_create` — NOT a standalone plan file in `docs/plans/`. For single, self-contained work, create a standalone story via `sf_story_create` (no epic needed — see "When to Use Epics vs Standalone Stories").
2. **Save per-story implementation plans** via `sf_plan_save` — this is where the bite-sized steps, TDD structure, and file-level detail from writing-plans lives.
3. **Use writing-plans' task granularity** (test → verify fail → implement → verify pass → commit) inside the sf_plan_save content.
4. **Do NOT save to `docs/plans/`** — specflow artifacts replace standalone plan files.

The epic/stories define WHAT to build (acceptance criteria, priorities, phases, dependencies). The plan defines HOW to build each story (step-by-step implementation).

### Executing (`superpowers:executing-plans`)

When implementing stories, combine specflow's tracking with superpowers' execution discipline:
1. **Source context from specflow:** Call `sf_context_build` before starting each story — this gives you conventions, epic context, specs, plan, and open items in one assembled prompt.
2. **Track with specflow:** Use `sf_execution_start` / `sf_execution_complete` for git baseline tracking.
3. **Execute with superpowers discipline:** Follow the batched execution model — implement in batches, report for review between batches, stop on blockers.
4. **Verify with specflow:** Use `sf_verify_save` to record pass/fail/partial with per-criterion checks.
5. **Complete with specflow:** `sf_story_update` to mark stories as `done`.

### Finishing (`superpowers:finishing-a-development-branch`)

After all stories in an epic are done, use the finishing skill for commit/PR/release decisions. specflow tracks the what; superpowers handles the git workflow.

## When to Use Epics vs Standalone Stories

Not every piece of work needs an epic. Use this guide:

**Create an epic** when:
- The work spans 3+ stories with dependencies
- There's a PRD or tech spec driving the work
- You need phased delivery

**Use a standalone story** when:
- It's a single, well-scoped unit of work (bug fix, small feature, refactor)
- There's no broader context needed beyond the story itself
- The acceptance criteria are self-contained

Standalone stories live in `.specflow/stories/` (not under an epic). The full lifecycle still applies:
```
sf_execution_start → write code → sf_execution_complete → sf_verify_save → sf_story_update(done)
```

When working on a standalone story, `sf_context_build` assembles project conventions + story details (no epic/spec layers). This is sufficient for focused, self-contained work.

## Starting Work

1. **Identify the story.** Use `sf_story_next` to get the highest-priority unblocked story, or work on the story the user specifies.
2. **Build context.** ALWAYS call `sf_context_build` before starting implementation. This assembles conventions, epic context, PRD content, plan, and open items into a single layered prompt.
3. **Check gates before implementing:**
   - **Open questions?** Resolve them with the user. Update via `sf_question_resolve`. Do not proceed with unresolved questions that affect implementation.
   - **Blockers?** If `blocked_by` stories aren't done, surface this to the user. Don't work around blockers silently.
   - **No plan?** Create one with `sf_plan_save` before writing code. Plans should include approach, key decisions, and what's NOT being done. **Then get user approval** (see Plan Approval section).

## Creating Artifacts

When creating epics, stories, docs, or decisions via `sf_*_create` / `sf_doc_write` / `sf_decision_record`:

- In **careful** mode: call `sf_hard_questions` before finalizing the artifact. Review the questions with the user and incorporate answers into the artifact.
- In **fast** mode: skip hard questions unless the user asks for them.
- Always fill in open_questions for anything you're uncertain about rather than making silent assumptions.

## Plan Approval

**CRITICAL: Never start implementation without user approval of the plan.**

After saving a plan via `sf_plan_save`:

1. **Present the plan** to the user — summarize the approach, key files, and what's NOT being done.
2. **Ask for approval** using `AskUserQuestion` with options:
   - **Approve** — proceed to implementation
   - **Request changes** — user provides feedback, update the plan, ask again
   - **Reject** — stop, discuss alternative approaches
3. **Only after approval:** proceed to `sf_execution_start` and implementation.

This gate applies to ALL stories — epic-scoped and standalone. Skip only if the user explicitly says "just do it" or the change is trivial (< 5 lines, obvious fix).

## Implementing

**Pre-condition: Plan must be approved by the user** (see Plan Approval section above).

**CRITICAL: Every story MUST go through this lifecycle. Do NOT skip steps.**

```
sf_execution_start → write code → run tests → sf_execution_complete → sf_verify_save → sf_story_update(done) → commit
```

1. **BEFORE writing any code:** Call `sf_execution_start`. This records the git baseline and auto-sets the story to `in_progress`. **Note the execution_id from the response** — you'll need it for completion.
2. Implement according to the plan and acceptance criteria from context.
3. Run tests and check acceptance criteria in the foreground.
4. **AFTER verification passes:** Complete the lifecycle, then commit everything together (see below).

**Never leave a story in `in_progress` or `verifying` at the end of a session.** Either complete the full cycle or explicitly note what's unfinished.

## Story Completion (Background)

To reduce terminal noise, batch the post-implementation lifecycle calls into a **single background Task agent**. This keeps the bookkeeping out of the main conversation.

Complete the specflow lifecycle **before committing**, so code changes and specflow metadata (execution, verification, story status) land in a single commit. This avoids a two-commit dance when `.specflow/` is tracked in git.

**When verification passes:**

Launch a single background Task (subagent_type: `general-purpose`, run_in_background: true) with a prompt that instructs it to run these three calls in sequence:
1. `sf_execution_complete` with the execution_id and story slug
2. `sf_verify_save` with result=pass, summary, and acceptance_check array
3. `sf_story_update` with status=done

Include ALL required parameters in the task prompt — the background agent has no conversation context. Example prompt structure:
```
Complete specflow lifecycle for story "my-story":
1. Call sf_execution_complete with execution_id="x_01ABC" and story="my-story"
2. Call sf_verify_save with story="my-story", result="pass", summary="All criteria met", acceptance_check=[{"criteria":"...", "met":true}, ...]
3. Call sf_story_update with slug="my-story", status="done"
```

After the background task completes, commit all changes (code + `.specflow/` metadata) together.

**When verification fails or is partial:**

Do NOT background — keep everything in the foreground so findings are visible to the user:
1. Call `sf_execution_complete` directly (foreground)
2. Call `sf_verify_save` with findings (foreground)
3. Discuss failures with the user before deciding next steps

**A story is NOT done until the background task completes successfully.** If you need to confirm completion, read the background task output file.

## Working Style

- **Present trade-offs, don't decide unilaterally.** When there are multiple valid approaches, lay out options with pros/cons and let the user choose.
- **Stop on contradictions.** If context build reveals conflicting requirements (e.g., PRD says X but story says Y), surface it immediately. Don't pick a side.
- **Record decisions.** When a non-trivial technical choice is made during implementation, record it via `sf_decision_record`.
- **Check scope when uncertain.** Use `sf_scope_check` against the PRD to determine whether something is in or out of scope. Don't gold-plate.
- **Use assumptions field.** Anything you assumed but couldn't verify goes into the story's assumptions list.
