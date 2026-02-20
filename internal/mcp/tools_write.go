package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/balazscsaba2006/specflow/internal/git"
	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerWriteTools registers all write/mutation MCP tools on the server.
func (s *Server) registerWriteTools() {
	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_initiative_create",
		Description: `Create a new initiative. An initiative is a top-level goal that groups epics.

Before creating, challenge the scope:
- Is this actually one initiative or multiple?
- What's the success criteria? If you can't measure it, push back.
- What happens if this takes 3x longer than expected?
- What's the minimum viable version?
- What dependencies does this create across the project?

Ask these questions conversationally before writing the initiative. Be direct, not diplomatic. If the scope is too vague, say so.`,
	}, s.handleInitiativeCreate)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_epic_create",
		Description: `Create a new epic. An epic groups related stories under an optional initiative.

Before creating:
- Does this epic have a clear boundary? Where does it end?
- What's the failure mode if we ship half of this?
- Is this over-engineered for current needs or under-engineered for where we're heading?
- Flag architectural implications the user might be overlooking.
- If there are multiple valid approaches, present options with trade-offs.

An epic should be shippable independently. If it's not, it might need to be split or reconsidered.`,
	}, s.handleEpicCreate)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_story_create",
		Description: `Create a new story. A story is the primary unit of work with acceptance criteria.

Before creating:
- Is this actually one story or should it be split?
- Are the acceptance criteria specific and testable?
- Does this story have clear "done" criteria?
- If the acceptance criteria need a paragraph to explain, the story is too big.

Stories can be standalone, under an epic, or under an initiative>epic. Require: title, acceptance criteria. Everything else is optional.`,
	}, s.handleStoryCreate)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_story_update",
		Description: `Update an existing story's status, priority, labels, blocked_by, assumptions, or open_questions. Only provided fields are updated. Assumptions and open_questions are appended to existing values. Validates state transitions (e.g., can't go from draft to done directly).`,
	}, s.handleStoryUpdate)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_doc_write",
		Description: `Create or update a document. If a doc with the given slug exists, it is updated; otherwise a new doc is created.

When creating a PRD:
- Start with the PROBLEM, not the solution. Push back if the user jumps to solutions.
- Success metrics must be measurable. "Improve UX" is not a metric.
- The "What If" section is mandatory — what breaks if requirements change?
- Open questions are a feature, not a bug. Capture what you don't know.
- Challenge scope: is this MVP or gold-plating?
- Flag risks the user hasn't mentioned.

When creating a tech spec:
- Ask the hard questions: What at 10x scale? Failure mode? Migration path?
- Flag when a pattern choice has non-obvious downstream consequences.
- Constraints section must be explicit — implicit constraints cause drift.

Act as a CPO/principal engineer reviewing the document. Be direct about gaps.`,
	}, s.handleDocWrite)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_decision_record",
		Description: `Record an architectural or design decision. The body is assembled from context, decision, and consequences sections.

Use this when a choice has been made during planning or implementation that future work should know about. Keep decisions concise.`,
	}, s.handleDecisionRecord)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_plan_save",
		Description: `Save an implementation plan for a story. Plans are markdown documents describing how to implement a story.

Captures current git ref as baseline. Plans should include file-level detail: which files to create/modify, what pattern to follow, and reference files for each step.`,
	}, s.handlePlanSave)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_execution_start",
		Description: `Start a new execution for a story. Captures current git ref as baseline, creates an execution record, and sets story status to in_progress.

The story must be in planned or in_progress status. If the story is in draft, transition it to planned first via sf_story_update.

Call this BEFORE starting to implement a story.`,
	}, s.handleExecutionStart)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_execution_pause",
		Description: `Pause an in-progress execution with handover notes for multi-session work. Captures what was done, what remains, and any gotchas for the next session.

Call this when ending a session mid-implementation. The handover notes will be surfaced in sf_context_build when work resumes.`,
	}, s.handleExecutionPause)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_execution_complete",
		Description: `Mark an execution as completed. Captures current git ref, computes file changes since execution start, and records them. Provide story slug to avoid scanning all stories.

Call this AFTER finishing implementation (code written, tests run).`,
	}, s.handleExecutionComplete)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_verify_save",
		Description: `Save verification results for a story's latest execution. Records pass/fail/partial result with findings and acceptance checks.

Verification should check:
- Were all acceptance criteria met?
- Were all planned files actually touched?
- Were any unexpected files modified?
- Are there assumptions baked in that should be documented?
- What will break if requirements change? (the "what if" list)

Be a principal engineer — direct, not diplomatic. If something is fine but the user would regret it in 6 months, flag it now.

Category values: missing | bug | performance | security | clarity | quality`,
	}, s.handleVerifySave)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name: "sf_question_resolve",
		Description: `Resolve an open question on any entity (initiative, epic, story, or doc) by removing it from open_questions. Records the resolution in the activity log.

The question is moved from open_questions to resolved_questions with the answer attached, preserving the Q&A pair as contextual knowledge on the entity.`,
	}, s.handleQuestionResolve)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_epic_archive",
		Description: `Archive an epic. Moves the epic tree to the archive directory and moves execution directories.

IMPORTANT: Do NOT use force unless the user explicitly asks for it. Without force, archiving requires the epic to be completed and all stories done — if it refuses, surface the reason to the user and let them decide. Using force silently can archive incomplete work.`,
	}, s.handleEpicArchive)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_story_archive",
		Description: `Archive a standalone story. Moves it to the archive directory and moves execution directories. Only works for standalone stories (not under an epic).

IMPORTANT: Do NOT use force unless the user explicitly asks for it. Without force, archiving requires the story to be done — if it refuses, surface the reason to the user and let them decide. Using force silently can archive incomplete work.`,
	}, s.handleStoryArchive)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_initiative_archive",
		Description: `Archive an initiative. Moves it to the archive directory. All linked epics must be archived or completed.

IMPORTANT: Do NOT use force unless the user explicitly asks for it. Without force, archiving requires all linked epics to be archived or completed — if it refuses, surface the reason to the user and let them decide. Using force silently can archive initiatives with active work.`,
	}, s.handleInitiativeArchive)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_epic_unarchive",
		Description: `Restore an archived epic back to active state. Moves the epic, its stories, docs, and executions from the archive back to the active directory. The epic status is set to on_hold; stories keep their original status.`,
	}, s.handleEpicUnarchive)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_story_unarchive",
		Description: `Restore an archived standalone story back to active state. Moves it from the archive back to the active directory with status set to planned. Only works for standalone stories.`,
	}, s.handleStoryUnarchive)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_initiative_unarchive",
		Description: `Restore an archived initiative back to active state. Moves it from the archive back to the active directory with status set to on_hold. Does NOT automatically unarchive linked epics — those must be unarchived separately if needed.`,
	}, s.handleInitiativeUnarchive)
}

// --- Input structs ---

type initiativeCreateInput struct {
	Slug            string   `json:"slug" jsonschema:"the URL-friendly identifier for the initiative"`
	Title           string   `json:"title" jsonschema:"human-readable title"`
	Goal            string   `json:"goal" jsonschema:"the initiative's goal"`
	SuccessCriteria []string `json:"success_criteria,omitempty" jsonschema:"measurable success criteria"`
	OpenQuestions   []string `json:"open_questions,omitempty" jsonschema:"unresolved questions"`
	Body            string   `json:"body,omitempty" jsonschema:"markdown body content"`
}

type epicCreateInput struct {
	Slug          string         `json:"slug" jsonschema:"the URL-friendly identifier for the epic"`
	Title         string         `json:"title" jsonschema:"human-readable title"`
	Initiative    string         `json:"initiative,omitempty" jsonschema:"parent initiative slug"`
	Fidelity      string         `json:"fidelity,omitempty" jsonschema:"target fidelity: prototype, personal-tool, alpha, beta, production"`
	Phases        []models.Phase `json:"phases,omitempty" jsonschema:"ordered phases with story references"`
	Body          string         `json:"body,omitempty" jsonschema:"markdown body content"`
	OpenQuestions []string       `json:"open_questions,omitempty" jsonschema:"unresolved questions"`
	NonGoals      []string       `json:"non_goals,omitempty" jsonschema:"explicit non-goals or out-of-scope items"`
}

type storyCreateInput struct {
	Slug          string   `json:"slug" jsonschema:"the URL-friendly identifier for the story"`
	Title         string   `json:"title" jsonschema:"human-readable title"`
	Epic          string   `json:"epic,omitempty" jsonschema:"parent epic slug"`
	Priority      string   `json:"priority,omitempty" jsonschema:"priority level: critical, high, medium, low"`
	Fidelity      string   `json:"fidelity,omitempty" jsonschema:"target fidelity (for standalone stories without epic): prototype, personal-tool, alpha, beta, production"`
	Acceptance    []string `json:"acceptance" jsonschema:"acceptance criteria"`
	Labels        []string `json:"labels,omitempty" jsonschema:"classification labels"`
	BlockedBy     []string `json:"blocked_by,omitempty" jsonschema:"slugs of blocking stories"`
	DocRefs       []string `json:"doc_refs,omitempty" jsonschema:"referenced document slugs"`
	OpenQuestions []string `json:"open_questions,omitempty" jsonschema:"unresolved questions"`
	NonGoals      []string `json:"non_goals,omitempty" jsonschema:"explicit non-goals or out-of-scope items"`
	Body          string   `json:"body,omitempty" jsonschema:"markdown body content"`
}

type storyUpdateInput struct {
	Slug          string   `json:"slug" jsonschema:"story slug to update"`
	Status        string   `json:"status,omitempty" jsonschema:"new status (validated transition)"`
	Priority      string   `json:"priority,omitempty" jsonschema:"new priority"`
	Labels        []string `json:"labels,omitempty" jsonschema:"replacement labels"`
	BlockedBy     []string `json:"blocked_by,omitempty" jsonschema:"replacement blocked_by"`
	Assumptions   []string `json:"assumptions,omitempty" jsonschema:"assumptions to append"`
	OpenQuestions []string `json:"open_questions,omitempty" jsonschema:"open questions to append"`
}

type docWriteInput struct {
	Slug          string   `json:"slug" jsonschema:"the URL-friendly identifier for the doc"`
	Type          string   `json:"type" jsonschema:"doc type: prd, tech-spec, api-spec, design-spec, adr, one-pager"`
	Title         string   `json:"title" jsonschema:"human-readable title"`
	Body          string   `json:"body" jsonschema:"markdown body content"`
	Epic          string   `json:"epic,omitempty" jsonschema:"parent epic slug"`
	OpenQuestions []string `json:"open_questions,omitempty" jsonschema:"unresolved questions"`
}

type decisionRecordInput struct {
	Slug         string   `json:"slug" jsonschema:"the URL-friendly identifier for the decision"`
	Title        string   `json:"title" jsonschema:"human-readable title"`
	Context      string   `json:"context" jsonschema:"context/background for the decision"`
	Decision     string   `json:"decision" jsonschema:"the decision that was made"`
	Consequences string   `json:"consequences" jsonschema:"consequences of this decision"`
	ContextRefs  []string `json:"context_refs,omitempty" jsonschema:"references to related entities"`
}

type planSaveInput struct {
	Story   string `json:"story" jsonschema:"story slug this plan belongs to"`
	Content string `json:"content" jsonschema:"markdown content of the plan"`
	Status  string `json:"status,omitempty" jsonschema:"plan status (default: draft)"`
}

type executionStartInput struct {
	Story string `json:"story" jsonschema:"story slug to start execution for"`
}

type executionPauseInput struct {
	ExecutionID   string `json:"execution_id" jsonschema:"the execution ID to pause"`
	Story         string `json:"story,omitempty" jsonschema:"story slug (avoids O(N) scan if provided)"`
	HandoverNotes string `json:"handover_notes" jsonschema:"markdown notes for the next session: what was done, what remains, blockers, gotchas"`
}

type executionCompleteInput struct {
	ExecutionID string `json:"execution_id" jsonschema:"the execution ID to complete"`
	Story       string `json:"story,omitempty" jsonschema:"story slug (avoids O(N) scan if provided)"`
}

type findingInput struct {
	Severity    string `json:"severity" jsonschema:"critical, major, or minor"`
	Category    string `json:"category" jsonschema:"missing, bug, performance, security, clarity, quality"`
	File        string `json:"file,omitempty" jsonschema:"file path related to the finding"`
	Description string `json:"description" jsonschema:"description of the finding"`
	Suggestion  string `json:"suggestion,omitempty" jsonschema:"suggested fix"`
}

type acceptanceItemInput struct {
	Criteria string `json:"criteria" jsonschema:"the acceptance criteria text"`
	Met      bool   `json:"met" jsonschema:"whether the criteria was met"`
}

type verifySaveInput struct {
	Story           string                `json:"story" jsonschema:"story slug"`
	Result          string                `json:"result" jsonschema:"pass, fail, or partial"`
	Summary         string                `json:"summary" jsonschema:"summary of verification results"`
	Findings        []findingInput        `json:"findings,omitempty" jsonschema:"list of findings"`
	AcceptanceCheck []acceptanceItemInput `json:"acceptance_check,omitempty" jsonschema:"acceptance criteria check results"`
	Assumptions     []string              `json:"assumptions,omitempty" jsonschema:"assumptions made during verification"`
}

type questionResolveInput struct {
	Entity   string `json:"entity" jsonschema:"entity slug (initiative, epic, story, or doc)"`
	Question string `json:"question" jsonschema:"the exact open question text to resolve"`
	Answer   string `json:"answer" jsonschema:"the answer or resolution"`
}

type epicArchiveInput struct {
	Slug    string `json:"slug" jsonschema:"epic slug to archive"`
	Force   bool   `json:"force,omitempty" jsonschema:"archive even if not completed/done. Only use when the user explicitly requests it."`
	Compact bool   `json:"compact,omitempty" jsonschema:"strip markdown bodies (compact to frontmatter-only tombstones). Default false preserves full content."`
}

type storyArchiveInput struct {
	Slug    string `json:"slug" jsonschema:"standalone story slug to archive"`
	Force   bool   `json:"force,omitempty" jsonschema:"archive even if not done. Only use when the user explicitly requests it."`
	Compact bool   `json:"compact,omitempty" jsonschema:"strip markdown body (compact to frontmatter-only tombstone). Default false preserves full content."`
}

type initiativeArchiveInput struct {
	Slug    string `json:"slug" jsonschema:"initiative slug to archive"`
	Force   bool   `json:"force,omitempty" jsonschema:"archive even if not completed. Only use when the user explicitly requests it."`
	Compact bool   `json:"compact,omitempty" jsonschema:"strip markdown body (compact to frontmatter-only tombstone). Default false preserves full content."`
}

type epicUnarchiveInput struct {
	Slug string `json:"slug" jsonschema:"epic slug to unarchive"`
}

type storyUnarchiveInput struct {
	Slug string `json:"slug" jsonschema:"standalone story slug to unarchive"`
}

type initiativeUnarchiveInput struct {
	Slug string `json:"slug" jsonschema:"initiative slug to unarchive"`
}

// --- Handlers ---

func (s *Server) handleInitiativeCreate(_ context.Context, _ *mcp.CallToolRequest, input initiativeCreateInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" || input.Title == "" || input.Goal == "" {
		return errResult("slug, title, and goal are required"), nil, nil
	}

	i := &models.Initiative{
		Slug:            input.Slug,
		Title:           input.Title,
		Status:          models.InitiativeStatusActive,
		Goal:            input.Goal,
		SuccessCriteria: input.SuccessCriteria,
		OpenQuestions:   input.OpenQuestions,
		Body:            input.Body,
	}

	if err := s.store.CreateInitiative(i); err != nil {
		return errResultf("creating initiative: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogInitiativeCreated,
		Entity: i.Slug,
	})

	return textResult(fmt.Sprintf("Created initiative **%s** (`%s`)\nID: `%s`", i.Title, i.Slug, i.ID)), nil, nil
}

func (s *Server) handleEpicCreate(_ context.Context, _ *mcp.CallToolRequest, input epicCreateInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" || input.Title == "" {
		return errResult("slug and title are required"), nil, nil
	}

	e := &models.Epic{
		Slug:          input.Slug,
		Title:         input.Title,
		Status:        models.EpicStatusDraft,
		Initiative:    input.Initiative,
		Fidelity:      input.Fidelity,
		Phases:        input.Phases,
		OpenQuestions: input.OpenQuestions,
		NonGoals:      input.NonGoals,
		Body:          input.Body,
	}

	if err := s.store.CreateEpic(e); err != nil {
		return errResultf("creating epic: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogEpicCreated,
		Entity: e.Slug,
	})

	return textResult(fmt.Sprintf("Created epic **%s** (`%s`)\nID: `%s`", e.Title, e.Slug, e.ID)), nil, nil
}

func (s *Server) handleStoryCreate(_ context.Context, _ *mcp.CallToolRequest, input storyCreateInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" || input.Title == "" || len(input.Acceptance) == 0 {
		return errResult("slug, title, and acceptance are required"), nil, nil
	}

	priority := input.Priority
	if priority == "" {
		priority = models.PriorityMedium
	}

	st := &models.Story{
		Slug:          input.Slug,
		Title:         input.Title,
		Epic:          input.Epic,
		Priority:      priority,
		Fidelity:      input.Fidelity,
		Acceptance:    input.Acceptance,
		Labels:        input.Labels,
		BlockedBy:     input.BlockedBy,
		DocRefs:       input.DocRefs,
		OpenQuestions: input.OpenQuestions,
		NonGoals:      input.NonGoals,
		Body:          input.Body,
	}

	if err := s.store.CreateStory(st); err != nil {
		return errResultf("creating story: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogStoryCreated,
		Entity: st.Slug,
		Epic:   st.Epic,
	})

	return textResult(fmt.Sprintf("Created story **%s** (`%s`)\nID: `%s` | Priority: %s | Status: %s",
		st.Title, st.Slug, st.ID, st.Priority, st.Status)), nil, nil
}

func (s *Server) handleStoryUpdate(_ context.Context, _ *mcp.CallToolRequest, input storyUpdateInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" {
		return errResult("slug is required"), nil, nil
	}

	// Try to find the story — could be standalone or under an epic.
	st, err := s.findStory(input.Slug)
	if err != nil {
		return errResultf("loading story %q: %v", input.Slug, err), nil, nil
	}

	oldStatus := st.Status

	if input.Status != "" {
		st.Status = input.Status
	}
	if input.Priority != "" {
		st.Priority = input.Priority
	}
	if input.Labels != nil {
		st.Labels = input.Labels
	}
	if input.BlockedBy != nil {
		st.BlockedBy = input.BlockedBy
	}
	if len(input.Assumptions) > 0 {
		st.Assumptions = append(st.Assumptions, input.Assumptions...)
	}
	if len(input.OpenQuestions) > 0 {
		st.OpenQuestions = append(st.OpenQuestions, input.OpenQuestions...)
	}

	if err := s.store.SaveStory(st); err != nil {
		return errResultf("saving story %q: %v", input.Slug, err), nil, nil
	}

	if oldStatus != st.Status {
		_ = s.store.AppendLog(models.LogEntry{
			Type:   models.LogStoryStatusChanged,
			Entity: st.Slug,
			Epic:   st.Epic,
			From:   oldStatus,
			To:     st.Status,
		})
	}

	msg := fmt.Sprintf("Updated story **%s** (`%s`)\nStatus: %s | Priority: %s",
		st.Title, st.Slug, st.Status, st.Priority)

	// Auto-complete epic and initiative when all stories are done.
	if st.Status == models.StoryStatusDone && st.Epic != "" {
		if cascadeMsg := s.cascadeCompletion(st.Epic); cascadeMsg != "" {
			msg += "\n\n" + cascadeMsg
		}
	}

	return textResult(msg), nil, nil
}

func (s *Server) handleDocWrite(_ context.Context, _ *mcp.CallToolRequest, input docWriteInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" || input.Type == "" || input.Title == "" || input.Body == "" {
		return errResult("slug, type, title, and body are required"), nil, nil
	}

	// Try loading existing doc first.
	existing, err := s.store.LoadDoc(input.Slug, input.Epic)
	if err != nil && input.Epic == "" {
		// Doc not found at project level — search under epics in case it lives there.
		if found, _, findErr := s.findDoc(input.Slug); findErr == nil {
			existing = found
			err = nil
		}
	}
	if err == nil {
		// Update existing doc, preserving its original epic scope.
		existing.Title = input.Title
		existing.Type = input.Type
		existing.Body = input.Body
		existing.OpenQuestions = input.OpenQuestions

		if err := s.store.SaveDoc(existing); err != nil {
			return errResultf("updating doc: %v", err), nil, nil
		}

		_ = s.store.AppendLog(models.LogEntry{
			Type:   models.LogDocUpdated,
			Entity: existing.Slug,
			Epic:   existing.Epic,
		})

		return textResult(fmt.Sprintf("Updated doc **%s** (`%s`)\nID: `%s` | Type: %s",
			existing.Title, existing.Slug, existing.ID, existing.Type)), nil, nil
	}

	// Create new doc.
	d := &models.Document{
		Slug:          input.Slug,
		Type:          input.Type,
		Title:         input.Title,
		Epic:          input.Epic,
		OpenQuestions: input.OpenQuestions,
		Body:          input.Body,
	}

	if err := s.store.CreateDoc(d); err != nil {
		return errResultf("creating doc: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogDocCreated,
		Entity: d.Slug,
		Epic:   d.Epic,
	})

	return textResult(fmt.Sprintf("Created doc **%s** (`%s`)\nID: `%s` | Type: %s",
		d.Title, d.Slug, d.ID, d.Type)), nil, nil
}

func (s *Server) handleDecisionRecord(_ context.Context, _ *mcp.CallToolRequest, input decisionRecordInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" || input.Title == "" || input.Context == "" || input.Decision == "" || input.Consequences == "" {
		return errResult("slug, title, context, decision, and consequences are required"), nil, nil
	}

	body := fmt.Sprintf("## Context\n\n%s\n\n## Decision\n\n%s\n\n## Consequences\n\n%s",
		input.Context, input.Decision, input.Consequences)

	d := &models.Decision{
		Slug:        input.Slug,
		Title:       input.Title,
		ContextRefs: input.ContextRefs,
		Body:        body,
	}

	if err := s.store.CreateDecision(d); err != nil {
		return errResultf("creating decision: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogDecisionRecorded,
		Entity: d.Slug,
	})

	return textResult(fmt.Sprintf("Recorded decision **%s** (`%s`)\nID: `%s` | Status: %s",
		d.Title, d.Slug, d.ID, d.Status)), nil, nil
}

func (s *Server) handlePlanSave(_ context.Context, _ *mcp.CallToolRequest, input planSaveInput) (*mcp.CallToolResult, any, error) {
	if input.Story == "" || input.Content == "" {
		return errResult("story and content are required"), nil, nil
	}

	status := input.Status
	if status == "" {
		if s.cfg.Mode == "fast" {
			status = models.PlanStatusApproved
		} else {
			status = models.PlanStatusDraft
		}
	}

	gitRef, _ := git.CurrentRef() // best-effort; non-fatal if not in a git repo

	p := &models.Plan{
		Story:          input.Story,
		Status:         status,
		GitRefBaseline: gitRef,
		Body:           input.Content,
	}

	if err := s.store.SavePlan(p, input.Story); err != nil {
		return errResultf("saving plan: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogPlanSaved,
		Entity: p.ID,
		Story:  input.Story,
	})

	return textResult(fmt.Sprintf("Saved plan for story `%s`\nID: `%s` | Status: %s",
		input.Story, p.ID, p.Status)), nil, nil
}

func (s *Server) handleExecutionStart(_ context.Context, _ *mcp.CallToolRequest, input executionStartInput) (*mcp.CallToolResult, any, error) {
	if input.Story == "" {
		return errResult("story is required"), nil, nil
	}

	// Validate story status before creating execution.
	st, err := s.findStory(input.Story)
	if err != nil {
		return errResultf("story %q not found: %v", input.Story, err), nil, nil
	}
	if st.Status != models.StoryStatusPlanned && st.Status != models.StoryStatusInProgress {
		return errResultf(
			"story %q has status %q — must be `planned` or `in_progress` to start execution. "+
				"Transition with sf_story_update(status=\"planned\") first.",
			input.Story, st.Status,
		), nil, nil
	}

	gitRef, _ := git.CurrentRef() // best-effort; non-fatal if not in a git repo

	e := &models.Execution{
		Story:        input.Story,
		GitRefBefore: gitRef,
	}

	if err := s.store.CreateExecution(e); err != nil {
		return errResultf("creating execution: %v", err), nil, nil
	}

	// Link plan if one exists for this story (best-effort).
	if plan, planErr := s.store.LoadPlan(input.Story); planErr == nil {
		e.Plan = plan.ID
		_ = s.store.SaveExecution(e)
	}

	// Set story status to in_progress if not already.
	if st.Status != models.StoryStatusInProgress {
		st.Status = models.StoryStatusInProgress
		_ = s.store.SaveStory(st)
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogExecutionStarted,
		Entity: e.ID,
		Story:  input.Story,
	})

	return textResult(fmt.Sprintf("Started execution for story `%s`\nExecution ID: `%s`",
		input.Story, e.ID)), nil, nil
}

func (s *Server) handleExecutionComplete(_ context.Context, _ *mcp.CallToolRequest, input executionCompleteInput) (*mcp.CallToolResult, any, error) {
	if input.ExecutionID == "" {
		return errResult("execution_id is required"), nil, nil
	}

	var exec *models.Execution

	if input.Story != "" {
		// Direct load when story slug is provided.
		e, loadErr := s.store.LoadExecution(input.Story, input.ExecutionID)
		if loadErr != nil {
			return errResultf("execution %q not found for story %q", input.ExecutionID, input.Story), nil, nil
		}
		exec = e
	} else {
		// Fall back to scanning all stories for backwards compat.
		stories, err := s.store.ListAllStories()
		if err != nil {
			return errResultf("listing stories to find execution: %v", err), nil, nil
		}
		for _, st := range stories {
			e, loadErr := s.store.LoadExecution(st.Slug, input.ExecutionID)
			if loadErr == nil {
				exec = e
				break
			}
		}
	}

	if exec == nil {
		return errResultf("execution %q not found", input.ExecutionID), nil, nil
	}

	if err := s.completeExecution(exec); err != nil {
		return errResultf("saving execution: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:         models.LogExecutionCompleted,
		Entity:       exec.ID,
		Story:        exec.Story,
		FilesChanged: len(exec.FilesChanged),
	})

	msg := fmt.Sprintf("Completed execution `%s` for story `%s` (%d files changed)", exec.ID, exec.Story, len(exec.FilesChanged))
	if s.cfg.Mode != "fast" {
		msg += "\n\nPlease run verification and save results with `sf_verify_save`."
	}
	return textResult(msg), nil, nil
}

func (s *Server) handleExecutionPause(_ context.Context, _ *mcp.CallToolRequest, input executionPauseInput) (*mcp.CallToolResult, any, error) {
	if input.ExecutionID == "" || input.HandoverNotes == "" {
		return errResult("execution_id and handover_notes are required"), nil, nil
	}

	var exec *models.Execution

	if input.Story != "" {
		e, loadErr := s.store.LoadExecution(input.Story, input.ExecutionID)
		if loadErr != nil {
			return errResultf("execution %q not found for story %q", input.ExecutionID, input.Story), nil, nil
		}
		exec = e
	} else {
		stories, err := s.store.ListAllStories()
		if err != nil {
			return errResultf("listing stories to find execution: %v", err), nil, nil
		}
		for _, st := range stories {
			e, loadErr := s.store.LoadExecution(st.Slug, input.ExecutionID)
			if loadErr == nil {
				exec = e
				break
			}
		}
	}

	if exec == nil {
		return errResultf("execution %q not found", input.ExecutionID), nil, nil
	}

	if exec.Status != models.ExecutionStatusStarted {
		return errResultf("execution %q is %s, can only pause started executions", input.ExecutionID, exec.Status), nil, nil
	}

	// Pause the execution.
	exec.Status = models.ExecutionStatusPaused
	exec.HandoverNotes = input.HandoverNotes
	if err := s.store.SaveExecution(exec); err != nil {
		return errResultf("saving execution: %v", err), nil, nil
	}

	// Write handover notes to file.
	if err := s.store.SaveHandover(input.HandoverNotes, exec.Story, exec.ID); err != nil {
		return errResultf("saving handover notes: %v", err), nil, nil
	}

	// Set story back to planned.
	if st, stErr := s.findStory(exec.Story); stErr == nil && st.Status == models.StoryStatusInProgress {
		st.Status = models.StoryStatusPlanned
		_ = s.store.SaveStory(st)
		_ = s.store.AppendLog(models.LogEntry{
			Type:   models.LogStoryStatusChanged,
			Entity: st.Slug,
			Epic:   st.Epic,
			From:   models.StoryStatusInProgress,
			To:     models.StoryStatusPlanned,
		})
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogExecutionPaused,
		Entity: exec.ID,
		Story:  exec.Story,
	})

	return textResult(fmt.Sprintf("Paused execution `%s` for story `%s`.\nHandover notes saved. They will appear in `sf_context_build` when work resumes.",
		exec.ID, exec.Story)), nil, nil
}

func (s *Server) handleVerifySave(_ context.Context, _ *mcp.CallToolRequest, input verifySaveInput) (*mcp.CallToolResult, any, error) {
	if input.Story == "" || input.Result == "" || input.Summary == "" {
		return errResult("story, result, and summary are required"), nil, nil
	}

	// Find the latest execution for the story.
	latest, err := s.store.LatestExecution(input.Story)
	if err != nil {
		return errResultf("finding latest execution for story %q: %v", input.Story, err), nil, nil
	}

	// Convert input findings to model findings.
	var findings []models.Finding
	for _, f := range input.Findings {
		findings = append(findings, models.Finding{
			Severity:    f.Severity,
			Category:    f.Category,
			File:        f.File,
			Description: f.Description,
			Suggestion:  f.Suggestion,
		})
	}

	// Convert input acceptance checks to model acceptance checks.
	var checks []models.AcceptanceCheck
	for _, ac := range input.AcceptanceCheck {
		checks = append(checks, models.AcceptanceCheck{
			Criteria: ac.Criteria,
			Met:      ac.Met,
		})
	}

	// Compute stats from findings.
	var stats models.VerificationStats
	for _, f := range findings {
		switch f.Severity {
		case models.SeverityCritical:
			stats.Critical++
		case models.SeverityMajor:
			stats.Major++
		case models.SeverityMinor:
			stats.Minor++
		}
	}

	v := &models.Verification{
		Execution:       latest.ID,
		Story:           input.Story,
		Result:          input.Result,
		Stats:           stats,
		Findings:        findings,
		AcceptanceCheck: checks,
		Assumptions:     input.Assumptions,
		Body:            input.Summary,
	}

	if err := s.store.SaveVerification(v, input.Story, latest.ID); err != nil {
		return errResultf("saving verification: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:     models.LogVerificationSaved,
		Entity:   v.ID,
		Story:    input.Story,
		Result:   input.Result,
		Critical: stats.Critical,
		Major:    stats.Major,
		Minor:    stats.Minor,
	})

	return textResult(fmt.Sprintf("Saved verification for story `%s` (execution `%s`)\nID: `%s` | Result: **%s** | Critical: %d | Major: %d | Minor: %d",
		input.Story, latest.ID, v.ID, v.Result, stats.Critical, stats.Major, stats.Minor)), nil, nil
}

func (s *Server) handleQuestionResolve(_ context.Context, _ *mcp.CallToolRequest, input questionResolveInput) (*mcp.CallToolResult, any, error) {
	if input.Entity == "" || input.Question == "" || input.Answer == "" {
		return errResult("entity, question, and answer are required"), nil, nil
	}

	// Try each entity type until we find one with the matching slug.

	// Try initiative.
	if ini, err := s.store.LoadInitiative(input.Entity); err == nil {
		ini.ResolvedQuestions = append(ini.ResolvedQuestions, models.ResolvedQuestion{
			Question: input.Question,
			Answer:   input.Answer,
		})
		ini.OpenQuestions = removeQuestion(ini.OpenQuestions, input.Question)
		if err := s.store.SaveInitiative(ini); err != nil {
			return errResultf("saving initiative: %v", err), nil, nil
		}
		_ = s.store.AppendLog(models.LogEntry{
			Type:   models.LogQuestionResolved,
			Entity: input.Entity,
		})
		return textResult(fmt.Sprintf("Resolved question on initiative `%s`\n\n**Q:** %s\n**A:** %s",
			input.Entity, input.Question, input.Answer)), nil, nil
	}

	// Try epic.
	if ep, err := s.store.LoadEpic(input.Entity); err == nil {
		ep.ResolvedQuestions = append(ep.ResolvedQuestions, models.ResolvedQuestion{
			Question: input.Question,
			Answer:   input.Answer,
		})
		ep.OpenQuestions = removeQuestion(ep.OpenQuestions, input.Question)
		if err := s.store.SaveEpic(ep); err != nil {
			return errResultf("saving epic: %v", err), nil, nil
		}
		_ = s.store.AppendLog(models.LogEntry{
			Type:   models.LogQuestionResolved,
			Entity: input.Entity,
		})
		return textResult(fmt.Sprintf("Resolved question on epic `%s`\n\n**Q:** %s\n**A:** %s",
			input.Entity, input.Question, input.Answer)), nil, nil
	}

	// Try story (search all stories since we don't know the epic).
	if st, err := s.findStory(input.Entity); err == nil {
		st.ResolvedQuestions = append(st.ResolvedQuestions, models.ResolvedQuestion{
			Question: input.Question,
			Answer:   input.Answer,
		})
		st.OpenQuestions = removeQuestion(st.OpenQuestions, input.Question)
		if err := s.store.SaveStory(st); err != nil {
			return errResultf("saving story: %v", err), nil, nil
		}
		_ = s.store.AppendLog(models.LogEntry{
			Type:   models.LogQuestionResolved,
			Entity: input.Entity,
		})
		return textResult(fmt.Sprintf("Resolved question on story `%s`\n\n**Q:** %s\n**A:** %s",
			input.Entity, input.Question, input.Answer)), nil, nil
	}

	// Try doc (project-level first, then under each epic would require listing epics).
	if doc, err := s.store.LoadDoc(input.Entity, ""); err == nil {
		doc.ResolvedQuestions = append(doc.ResolvedQuestions, models.ResolvedQuestion{
			Question: input.Question,
			Answer:   input.Answer,
		})
		doc.OpenQuestions = removeQuestion(doc.OpenQuestions, input.Question)
		if err := s.store.SaveDoc(doc); err != nil {
			return errResultf("saving doc: %v", err), nil, nil
		}
		_ = s.store.AppendLog(models.LogEntry{
			Type:   models.LogQuestionResolved,
			Entity: input.Entity,
		})
		return textResult(fmt.Sprintf("Resolved question on doc `%s`\n\n**Q:** %s\n**A:** %s",
			input.Entity, input.Question, input.Answer)), nil, nil
	}

	// Try docs under epics.
	if doc, epicSlug, err := s.findDoc(input.Entity); err == nil {
		doc.ResolvedQuestions = append(doc.ResolvedQuestions, models.ResolvedQuestion{
			Question: input.Question,
			Answer:   input.Answer,
		})
		doc.OpenQuestions = removeQuestion(doc.OpenQuestions, input.Question)
		if err := s.store.SaveDoc(doc); err != nil {
			return errResultf("saving doc: %v", err), nil, nil
		}
		_ = s.store.AppendLog(models.LogEntry{
			Type:   models.LogQuestionResolved,
			Entity: input.Entity,
			Epic:   epicSlug,
		})
		return textResult(fmt.Sprintf("Resolved question on doc `%s` (epic `%s`)\n\n**Q:** %s\n**A:** %s",
			input.Entity, epicSlug, input.Question, input.Answer)), nil, nil
	}

	return errResultf("entity %q not found", input.Entity), nil, nil
}

func (s *Server) handleEpicArchive(_ context.Context, _ *mcp.CallToolRequest, input epicArchiveInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" {
		return errResult("slug is required"), nil, nil
	}

	summary, err := s.store.ArchiveEpic(input.Slug, input.Force, input.Compact)
	if err != nil {
		return errResultf("archiving epic: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogEpicArchived,
		Entity: input.Slug,
	})

	return textResult(fmt.Sprintf("Archived epic **%s** (`%s`)\n%d stories, %d executions moved",
		summary.EpicTitle, summary.EpicSlug, summary.StoryCount, summary.ExecutionCount)), nil, nil
}

func (s *Server) handleStoryArchive(_ context.Context, _ *mcp.CallToolRequest, input storyArchiveInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" {
		return errResult("slug is required"), nil, nil
	}

	summary, err := s.store.ArchiveStory(input.Slug, input.Force, input.Compact)
	if err != nil {
		return errResultf("archiving story: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogStoryArchived,
		Entity: input.Slug,
	})

	return textResult(fmt.Sprintf("Archived story **%s** (`%s`)\n%d executions moved",
		summary.Title, summary.Slug, summary.ExecutionCount)), nil, nil
}

func (s *Server) handleInitiativeArchive(_ context.Context, _ *mcp.CallToolRequest, input initiativeArchiveInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" {
		return errResult("slug is required"), nil, nil
	}

	summary, err := s.store.ArchiveInitiative(input.Slug, input.Force, input.Compact)
	if err != nil {
		return errResultf("archiving initiative: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogInitiativeArchived,
		Entity: input.Slug,
	})

	return textResult(fmt.Sprintf("Archived initiative **%s** (`%s`)\n%d linked epics",
		summary.Title, summary.Slug, summary.EpicCount)), nil, nil
}

func (s *Server) handleEpicUnarchive(_ context.Context, _ *mcp.CallToolRequest, input epicUnarchiveInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" {
		return errResult("slug is required"), nil, nil
	}

	summary, err := s.store.UnarchiveEpic(input.Slug)
	if err != nil {
		return errResultf("unarchiving epic: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogEpicUnarchived,
		Entity: input.Slug,
	})

	return textResult(fmt.Sprintf("Unarchived epic **%s** (`%s`)\n%d stories, %d executions restored\nEpic status set to **on_hold**",
		summary.EpicTitle, summary.EpicSlug, summary.StoryCount, summary.ExecutionCount)), nil, nil
}

func (s *Server) handleStoryUnarchive(_ context.Context, _ *mcp.CallToolRequest, input storyUnarchiveInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" {
		return errResult("slug is required"), nil, nil
	}

	summary, err := s.store.UnarchiveStory(input.Slug)
	if err != nil {
		return errResultf("unarchiving story: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogStoryUnarchived,
		Entity: input.Slug,
	})

	return textResult(fmt.Sprintf("Unarchived story **%s** (`%s`)\n%d executions restored\nStatus set to **planned**",
		summary.Title, summary.Slug, summary.ExecutionCount)), nil, nil
}

func (s *Server) handleInitiativeUnarchive(_ context.Context, _ *mcp.CallToolRequest, input initiativeUnarchiveInput) (*mcp.CallToolResult, any, error) {
	if input.Slug == "" {
		return errResult("slug is required"), nil, nil
	}

	summary, err := s.store.UnarchiveInitiative(input.Slug)
	if err != nil {
		return errResultf("unarchiving initiative: %v", err), nil, nil
	}

	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogInitiativeUnarchived,
		Entity: input.Slug,
	})

	return textResult(fmt.Sprintf("Unarchived initiative **%s** (`%s`)\n%d linked epics\nStatus set to **on_hold**",
		summary.Title, summary.Slug, summary.EpicCount)), nil, nil
}

// --- Helpers ---

// completeExecution finalizes an execution: sets status, captures git state, and auto-transitions story to verifying.
func (s *Server) completeExecution(exec *models.Execution) error {
	now := time.Now().UTC().Truncate(time.Second)
	exec.Status = models.ExecutionStatusCompleted
	exec.CompletedAt = &now

	if gitRef, gitErr := git.CurrentRef(); gitErr == nil {
		exec.GitRefAfter = gitRef
	}
	if exec.GitRefBefore != "" {
		if changes, gitErr := git.FileChanges(exec.GitRefBefore, ""); gitErr == nil {
			for _, c := range changes {
				exec.FilesChanged = append(exec.FilesChanged, models.FileChange{
					Path:   c.Path,
					Action: c.Action,
				})
			}
		}
	}

	if err := s.store.SaveExecution(exec); err != nil {
		return err
	}

	// Auto-transition story to verifying.
	if st, stErr := s.findStory(exec.Story); stErr == nil && st.Status == models.StoryStatusInProgress {
		st.Status = models.StoryStatusVerifying
		_ = s.store.SaveStory(st)
		_ = s.store.AppendLog(models.LogEntry{
			Type:   models.LogStoryStatusChanged,
			Entity: st.Slug,
			Epic:   st.Epic,
			From:   models.StoryStatusInProgress,
			To:     models.StoryStatusVerifying,
		})
	}

	return nil
}

// cascadeCompletion checks if all stories in an epic are done and auto-completes
// the epic. If the epic has an initiative, checks if all epics in that initiative
// are done and auto-completes the initiative. Returns a message describing what was auto-completed.
func (s *Server) cascadeCompletion(epicSlug string) string {
	msg := s.tryCompleteEpic(epicSlug)
	if msg == "" {
		return ""
	}

	ep, err := s.store.LoadEpic(epicSlug)
	if err != nil || ep.Initiative == "" {
		return msg
	}

	if iniMsg := s.tryCompleteInitiative(ep.Initiative); iniMsg != "" {
		return msg + "\n" + iniMsg
	}
	return msg
}

// tryCompleteEpic checks if all stories in an epic are done and auto-completes it.
func (s *Server) tryCompleteEpic(epicSlug string) string {
	ep, err := s.store.LoadEpic(epicSlug)
	if err != nil || ep.Status == models.EpicStatusCompleted {
		return ""
	}

	stories, err := s.store.ListStories(epicSlug)
	if err != nil || len(stories) == 0 {
		return ""
	}

	for _, st := range stories {
		if st.Status != models.StoryStatusDone {
			return ""
		}
	}

	oldStatus := ep.Status
	ep.Status = models.EpicStatusCompleted
	if saveErr := s.store.SaveEpic(ep); saveErr != nil {
		return ""
	}
	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogStoryStatusChanged,
		Entity: ep.Slug,
		From:   oldStatus,
		To:     models.EpicStatusCompleted,
	})

	return fmt.Sprintf("Auto-completed epic **%s** (all stories done)", ep.Slug)
}

// tryCompleteInitiative checks if all epics in an initiative are completed and auto-completes it.
func (s *Server) tryCompleteInitiative(iniSlug string) string {
	ini, err := s.store.LoadInitiative(iniSlug)
	if err != nil || ini.Status == models.InitiativeStatusCompleted {
		return ""
	}

	for _, linkedEpic := range ini.Epics {
		linkedEp, loadErr := s.store.LoadEpic(linkedEpic)
		if loadErr != nil || linkedEp.Status != models.EpicStatusCompleted {
			return ""
		}
	}

	oldStatus := ini.Status
	ini.Status = models.InitiativeStatusCompleted
	if saveErr := s.store.SaveInitiative(ini); saveErr != nil {
		return ""
	}
	_ = s.store.AppendLog(models.LogEntry{
		Type:   models.LogStoryStatusChanged,
		Entity: ini.Slug,
		From:   oldStatus,
		To:     models.InitiativeStatusCompleted,
	})

	return fmt.Sprintf("Auto-completed initiative **%s** (all epics done)", ini.Slug)
}

// findStory searches for a story by slug across all epics and standalone stories.
func (s *Server) findStory(slug string) (*models.Story, error) {
	// Try standalone first.
	if st, err := s.store.LoadStory(slug, ""); err == nil {
		return st, nil
	}

	// Search across all active stories.
	stories, err := s.store.ListAllStories()
	if err != nil {
		return nil, fmt.Errorf("searching for story %q: %w", slug, err)
	}

	for _, st := range stories {
		if st.Slug == slug {
			return st, nil
		}
	}

	// Fall back to archived epics.
	archivedEpics, archErr := s.store.ListArchivedEpics()
	if archErr == nil {
		for _, ep := range archivedEpics {
			if st, loadErr := s.store.LoadArchivedStory(slug, ep.Slug); loadErr == nil {
				return st, nil
			}
		}
	}

	return nil, fmt.Errorf("story %q not found", slug)
}

// findDoc searches for a doc by slug across all epics, including archived.
func (s *Server) findDoc(slug string) (*models.Document, string, error) {
	// Search active epics.
	epics, err := s.store.ListEpics()
	if err != nil {
		return nil, "", fmt.Errorf("listing epics: %w", err)
	}
	for _, ep := range epics {
		if doc, loadErr := s.store.LoadDoc(slug, ep.Slug); loadErr == nil {
			return doc, ep.Slug, nil
		}
	}

	// Fall back to archived epics.
	archivedEpics, archErr := s.store.ListArchivedEpics()
	if archErr == nil {
		for _, ep := range archivedEpics {
			docPath := filepath.Join(s.store.ArchiveEpicDocsDir(ep.Slug), slug+".md")
			var doc models.Document
			body, parseErr := store.ParseFile(docPath, &doc)
			if parseErr == nil {
				doc.Body = body
				return &doc, ep.Slug, nil
			}
		}
	}

	return nil, "", fmt.Errorf("doc %q not found under any epic", slug)
}

// removeQuestion filters out the first occurrence of question from the slice.
func removeQuestion(questions []string, question string) []string {
	result := make([]string, 0, len(questions))
	removed := false
	for _, q := range questions {
		if !removed && q == question {
			removed = true
			continue
		}
		result = append(result, q)
	}
	return result
}
