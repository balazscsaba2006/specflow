---
name: specflow
description: Use when implementing stories, creating artifacts, or doing spec-driven development with specflow MCP tools.
---

# specflow Workflow

You have access to specflow MCP tools (`sf_*`) for managing development artifacts. This skill defines how to use them together.

## Integration with superpowers Skills

**specflow is the source of truth for planning artifacts.** When both specflow and superpowers skills are available:

### Discovery & Planning (`superpowers:brainstorming` → specflow artifacts)

When the user describes a new feature or system:
1. **Brainstorming drives discovery** — explore, question, propose approaches, iterate, get approval.
2. **After design approval, create specflow artifacts** — initiative/epic/stories, NOT `docs/plans/` files.
3. **Per-story plans:**
   - **Complex multi-step stories:** Invoke `writing-plans` to produce granular TDD steps, then save the output via `sf_plan_save` (not to `docs/plans/`).
   - **Simple stories:** Write the plan directly via `sf_plan_save` using writing-plans style (TDD steps) without invoking the skill.
4. **Get plan approval** before implementation (see Plan Approval section).

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

## Discovery Phase — Discuss Before Creating

**When a user describes a new feature, system change, or problem to solve — DO NOT immediately create specflow artifacts.** Creating initiatives, epics, stories, or docs is the OUTPUT of discovery, not the starting point.

### When Discovery Applies

- User describes a feature, problem, or system change in natural language
- User shares context about what they want to build (even if detailed)
- User asks to "plan", "design", or "figure out" something

### When to Skip Discovery

- User explicitly asks to create a specific artifact ("create an epic for X")
- User is continuing work on an already-planned story
- The task is a trivial fix with obvious scope

### Discovery Flow

1. **Explore** — Read relevant code, understand current state, identify constraints
2. **Ask questions** — One at a time, clarify intent, constraints, success criteria
3. **Propose approaches** — 2-3 options with trade-offs and your recommendation
4. **Iterate** — Refine based on user feedback, go back and forth as needed
5. **Align** — Present final design summary, get explicit user approval
6. **THEN create artifacts** — Break down into specflow hierarchy (initiative > epic > stories) only after alignment

**The user saying "let's build X" is the START of discovery, not a signal to create artifacts.**

### Integration with Brainstorming Skill

If the `superpowers:brainstorming` skill is available, it drives steps 1-5. The specflow-specific addition is step 6: instead of saving a design doc to `docs/plans/`, create specflow artifacts:

1. Brainstorming completes → user approves design
2. Create initiative (if scope warrants it) via `sf_initiative_create`
3. Create epic(s) via `sf_epic_create`
4. Create stories with acceptance criteria via `sf_story_create`
5. Optionally create tech-spec/PRD docs via `sf_doc_write` (scoped to epic)
6. For complex multi-step stories: invoke `writing-plans` to produce granular implementation steps, then save via `sf_plan_save`. For simple stories: write the plan directly via `sf_plan_save`.

When work has natural phases (e.g., OIDC first, SAML second), create **separate epics per phase** with stories scoped to each. This allows independent tracking, estimation, and completion.

**Do NOT save to `docs/plans/`.** The brainstorming output becomes specflow artifacts directly.

If brainstorming is NOT available, follow the same flow yourself — the discovery steps are not optional just because the skill isn't loaded.

### Resolving Open Questions

**When open questions exist — whether from discovery, from `sf_questions`, or from the user asking you to resolve them — use `AskUserQuestion` to ask them interactively.** Do NOT just list questions as text output. Each question should be asked via the tool so the user can answer directly.

- Ask questions in batches of 1-4 (the tool's limit) grouped by topic
- After each batch of answers, resolve them in specflow via `sf_question_resolve`
- Continue until all questions are resolved before creating artifacts or starting implementation
- If the user explicitly says "ask the open questions", this is your cue to use `AskUserQuestion` — not to print a markdown list

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
