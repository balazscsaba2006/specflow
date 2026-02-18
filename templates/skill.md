---
name: specflow
description: Use when implementing stories, creating artifacts, or doing spec-driven development with specflow MCP tools.
---

# specflow Workflow

You have access to specflow MCP tools (`sf_*`) for managing development artifacts. This skill defines how to use them together.

## Integration with superpowers Skills

**specflow is the source of truth for planning artifacts.** When both specflow and superpowers skills are available:

### Planning (`superpowers:writing-plans`)

When the user asks to plan a feature or the writing-plans skill is invoked:
1. **Create the epic and stories in specflow** via `sf_epic_create` and `sf_story_create` — NOT a standalone plan file in `docs/plans/`.
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

## Starting Work

1. **Identify the story.** Use `sf_story_next` to get the highest-priority unblocked story, or work on the story the user specifies.
2. **Build context.** ALWAYS call `sf_context_build` before starting implementation. This assembles conventions, epic context, PRD content, plan, and open items into a single layered prompt.
3. **Check gates before implementing:**
   - **Open questions?** Resolve them with the user. Update via `sf_question_resolve`. Do not proceed with unresolved questions that affect implementation.
   - **Blockers?** If `blocked_by` stories aren't done, surface this to the user. Don't work around blockers silently.
   - **No plan?** Create one with `sf_plan_save` before writing code. Plans should include approach, key decisions, and what's NOT being done.

## Creating Artifacts

When creating epics, stories, docs, or decisions via `sf_*_create` / `sf_doc_write` / `sf_decision_record`:

- In **careful** mode: call `sf_hard_questions` before finalizing the artifact. Review the questions with the user and incorporate answers into the artifact.
- In **fast** mode: skip hard questions unless the user asks for them.
- Always fill in open_questions for anything you're uncertain about rather than making silent assumptions.

## Implementing

**CRITICAL: Every story MUST go through this lifecycle. Do NOT skip steps.**

```
sf_execution_start → write code → sf_execution_complete → sf_verify_save → sf_story_update(done)
```

1. **BEFORE writing any code:** Call `sf_execution_start`. This records the git baseline and auto-sets the story to `in_progress`. If you haven't called this, stop and call it now.
2. Implement according to the plan and acceptance criteria from context.
3. **AFTER implementation is complete:** Call `sf_execution_complete`. This captures the git diff and auto-sets the story to `verifying`.
4. Record any assumptions made during implementation via `sf_story_update` (add to the assumptions field).

**Never leave a story in `in_progress` or `verifying` at the end of a session.** Either complete the full cycle or explicitly note what's unfinished.

## Verifying

1. Run tests and check acceptance criteria.
2. Use `sf_diff_check` to detect spec drift — compare what was implemented against what was specified.
3. **MUST call** `sf_verify_save` with the result (`pass`, `partial`, `fail`), findings, and per-criterion checks. This is not optional.

## Completing

- **Pass:** Call `sf_story_update` to set status to `done`. This is mandatory after a passing verification.
- **Partial:** Surface findings to the user. Fix what can be fixed, then re-verify.
- **Fail:** Discuss with the user. Do not silently mark as done.

**A story is NOT done until `sf_story_update status=done` has been called.** Implemented code without a status update means the tracking is broken.

## Working Style

- **Present trade-offs, don't decide unilaterally.** When there are multiple valid approaches, lay out options with pros/cons and let the user choose.
- **Stop on contradictions.** If context build reveals conflicting requirements (e.g., PRD says X but story says Y), surface it immediately. Don't pick a side.
- **Record decisions.** When a non-trivial technical choice is made during implementation, record it via `sf_decision_record`.
- **Check scope when uncertain.** Use `sf_scope_check` against the PRD to determine whether something is in or out of scope. Don't gold-plate.
- **Use assumptions field.** Anything you assumed but couldn't verify goes into the story's assumptions list.
