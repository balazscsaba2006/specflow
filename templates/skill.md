---
name: specflow
description: Use when implementing stories, creating artifacts, or doing spec-driven development with specflow MCP tools.
---

# specflow Workflow

You have access to specflow MCP tools (`sf_*`) for managing development artifacts. This skill defines how to use them together.

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

1. Call `sf_execution_start` to record the baseline git ref.
2. Implement according to the plan and acceptance criteria from context.
3. Call `sf_execution_complete` when implementation is done to capture the end state.
4. Record any assumptions made during implementation via `sf_story_update` (add to the assumptions field).

## Verifying

1. Run tests and check acceptance criteria.
2. Use `sf_diff_check` to detect spec drift — compare what was implemented against what was specified.
3. Save verification results with `sf_verify_save`:
   - Include the result (`pass`, `partial`, `fail`), findings, and per-criterion checks.

## Completing

- **Pass:** Call `sf_story_update` to set status to `done`.
- **Partial:** Surface findings to the user. Fix what can be fixed, then re-verify.
- **Fail:** Discuss with the user. Do not silently mark as done.

## Working Style

- **Present trade-offs, don't decide unilaterally.** When there are multiple valid approaches, lay out options with pros/cons and let the user choose.
- **Stop on contradictions.** If context build reveals conflicting requirements (e.g., PRD says X but story says Y), surface it immediately. Don't pick a side.
- **Record decisions.** When a non-trivial technical choice is made during implementation, record it via `sf_decision_record`.
- **Check scope when uncertain.** Use `sf_scope_check` against the PRD to determine whether something is in or out of scope. Don't gold-plate.
- **Use assumptions field.** Anything you assumed but couldn't verify goes into the story's assumptions list.
