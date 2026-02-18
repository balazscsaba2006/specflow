package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/git"
	"github.com/balazscsaba2006/specflow/internal/hardq"
	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// priorityRank maps priority strings to sort order (lower = higher priority).
var priorityRank = map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}

// --- Input structs ---

type statusInput struct {
	Scope           string `json:"scope,omitempty" jsonschema:"optional scope filter (epic slug or 'all')"`
	IncludeArchived bool   `json:"include_archived,omitempty" jsonschema:"include archived epics in results"`
}

type slugInput struct {
	Slug string `json:"slug" jsonschema:"the slug identifier"`
}

type storyNextInput struct {
	Epic string `json:"epic,omitempty" jsonschema:"optional epic slug to filter by"`
}

type storyLsInput struct {
	Epic            string `json:"epic,omitempty" jsonschema:"filter by epic slug"`
	Status          string `json:"status,omitempty" jsonschema:"filter by status"`
	Label           string `json:"label,omitempty" jsonschema:"filter by label"`
	Blocked         bool   `json:"blocked,omitempty" jsonschema:"show only blocked stories"`
	IncludeArchived bool   `json:"include_archived,omitempty" jsonschema:"include stories from archived epics"`
}

type docReadInput struct {
	Slug string `json:"slug" jsonschema:"document slug"`
	Epic string `json:"epic,omitempty" jsonschema:"optional epic slug"`
}

type storyRefInput struct {
	Story string `json:"story" jsonschema:"story slug"`
}

type epicOptInput struct {
	Epic string `json:"epic,omitempty" jsonschema:"optional epic slug to filter by"`
}

type logInput struct {
	Last int `json:"last,omitempty" jsonschema:"number of recent entries to return (default 20)"`
}

type diffInput struct {
	Story string `json:"story,omitempty" jsonschema:"story slug (uses latest execution baseline)"`
	Refs  string `json:"refs,omitempty" jsonschema:"explicit git ref range e.g. abc123..HEAD"`
}

type hardQuestionsInput struct {
	Entity string `json:"entity" jsonschema:"any entity slug (initiative, epic, story, or doc)"`
}

type reviewPromptInput struct {
	Doc  string `json:"doc" jsonschema:"document slug"`
	Epic string `json:"epic,omitempty" jsonschema:"parent epic slug for epic-scoped docs"`
}

type epicRequiredInput struct {
	Epic string `json:"epic" jsonschema:"epic slug"`
}

// registerReadTools registers all read-only MCP tools on the server.
func (s *Server) registerReadTools() {
	mcp.AddTool[statusInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_status",
		Description: "Show project status with progress per epic. Scope can be an epic slug or empty for all.",
	}, s.handleStatus)

	mcp.AddTool[slugInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_initiative_show",
		Description: "Show initiative details including goal, success criteria, linked epics, and open questions.",
	}, s.handleInitiativeShow)

	mcp.AddTool[slugInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_epic_show",
		Description: "Show epic details including phases, story statuses, and open questions.",
	}, s.handleEpicShow)

	mcp.AddTool[slugInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_story_show",
		Description: "Show story details including acceptance criteria, doc refs, assumptions, and open questions.",
	}, s.handleStoryShow)

	mcp.AddTool[storyNextInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_story_next",
		Description: "Suggest the next story to work on. Returns the highest-priority planned story with no unresolved blockers.",
	}, s.handleStoryNext)

	mcp.AddTool[storyLsInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_story_ls",
		Description: "List stories with optional filters for epic, status, label, and blocked state.",
	}, s.handleStoryLs)

	mcp.AddTool[docReadInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_doc_read",
		Description: "Read a document by slug. Returns frontmatter summary and full body.",
	}, s.handleDocRead)

	mcp.AddTool[storyRefInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_plan_read",
		Description: "Read the implementation plan for a story.",
	}, s.handlePlanRead)

	mcp.AddTool[storyRefInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_verify_read",
		Description: "Read the latest verification result for a story.",
	}, s.handleVerifyRead)

	mcp.AddTool[storyRefInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_context_build",
		Description: "Build the full 6-layer assembled context for a story. Use this before starting implementation.",
	}, s.handleContextBuild)

	mcp.AddTool[epicOptInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_questions",
		Description: "List all open questions across initiatives, epics, stories, and docs, grouped by source.",
	}, s.handleQuestions)

	mcp.AddTool[any, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_blocked",
		Description: "List all stories that have unresolved blockers.",
	}, s.handleBlocked)

	mcp.AddTool[epicOptInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_decisions",
		Description: "List all decisions, optionally filtered by epic context ref.",
	}, s.handleDecisions)

	mcp.AddTool[logInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_log",
		Description: "Show recent activity log entries as a timeline.",
	}, s.handleLog)

	mcp.AddTool[epicOptInput, any](s.mcpSrv, &mcp.Tool{
		Name:        "sf_assumptions",
		Description: "List all assumptions across stories, grouped by epic.",
	}, s.handleAssumptions)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_diff",
		Description: "Returns git diff for a story's execution. Defaults to diff between execution start and current HEAD. Provide either a story slug or explicit ref range.",
	}, s.handleDiff)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_hard_questions",
		Description: "Returns contextual hard questions for any entity based on its type. These are deterministic template-based questions to challenge thinking before finalizing an artifact.",
	}, s.handleHardQuestions)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_review_prompt",
		Description: "Assembles a coaching/review prompt for a document. Returns a structured prompt with the document content embedded, tailored to the document type (PRD, tech-spec, etc.).",
	}, s.handleReviewPrompt)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_scope_check",
		Description: "Compares current stories against the PRD's scope definition. Flags stories that aren't traceable to PRD user stories or are explicitly out-of-scope.",
	}, s.handleScopeCheck)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_diff_check",
		Description: "Detects drift between specs and their stories. Checks if documents were updated more recently than stories that reference them, indicating potential drift.",
	}, s.handleDiffCheck)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_scope_drift",
		Description: "Assembles a structured prompt to evaluate plan-vs-reality drift. Loads the implementation plan and actual git diff, then asks for a semantic comparison: what was implemented as planned, what deviated, what's missing, and what's unplanned.",
	}, s.handleScopeDrift)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_unstuck",
		Description: "Assembles debugging-focused context when stuck during implementation. Gathers story, plan, current diff, execution state, handover notes, and suggests debugging steps.",
	}, s.handleUnstuck)

	mcp.AddTool(s.mcpSrv, &mcp.Tool{
		Name:        "sf_code_review",
		Description: "Assembles a structured code review prompt for a story's latest execution. Loads the execution diff, story acceptance criteria, and implementation plan into a review template that checks: criteria met, planned files touched, unexpected changes, security issues, test coverage.",
	}, s.handleCodeReview)
}

// --- Handlers ---

func (s *Server) handleStatus(_ context.Context, _ *mcp.CallToolRequest, input statusInput) (*mcp.CallToolResult, any, error) {
	epics, err := s.store.ListEpics()
	if err != nil {
		return errResultf("listing epics: %v", err), nil, nil
	}

	allStories, err := s.store.ListAllStories()
	if err != nil {
		return errResultf("listing stories: %v", err), nil, nil
	}

	// Group stories by epic.
	storyByEpic := make(map[string][]*models.Story)
	for _, st := range allStories {
		storyByEpic[st.Epic] = append(storyByEpic[st.Epic], st)
	}

	var b strings.Builder
	b.WriteString("## Project Status\n\n")

	for _, epic := range epics {
		if input.Scope != "" && input.Scope != "all" && input.Scope != epic.Slug {
			continue
		}
		writeStatusBlock(&b, epic.Title+" ("+epic.Status+")", storyByEpic[epic.Slug])
		b.WriteString("\n")
	}

	// Standalone stories.
	if standalone := storyByEpic[""]; len(standalone) > 0 && (input.Scope == "" || input.Scope == "all") {
		writeStatusBlock(&b, "Standalone Stories", standalone)
	}

	if input.IncludeArchived {
		s.writeArchivedStatus(&b)
	}

	return textResult(b.String()), nil, nil
}

// writeArchivedStatus appends archived epic and standalone story status blocks to the builder.
func (s *Server) writeArchivedStatus(b *strings.Builder) {
	archivedEpics, _ := s.store.ListArchivedEpics()
	archivedStandalone, _ := s.store.ListArchivedStandaloneStories()

	if len(archivedEpics) == 0 && len(archivedStandalone) == 0 {
		return
	}
	b.WriteString("\n## Archived\n\n")
	for _, ep := range archivedEpics {
		archStories, stErr := s.store.ListArchivedStories(ep.Slug)
		if stErr != nil {
			continue
		}
		writeStatusBlock(b, ep.Title+" [archived]", archStories)
		b.WriteString("\n")
	}
	if len(archivedStandalone) > 0 {
		writeStatusBlock(b, "Standalone Stories [archived]", archivedStandalone)
		b.WriteString("\n")
	}
}

// writeStatusBlock writes a status summary block for a group of stories.
func writeStatusBlock(b *strings.Builder, label string, stories []*models.Story) {
	counts := countStatuses(stories)
	total := len(stories)
	done := counts[models.StoryStatusDone]
	pct := 0
	if total > 0 {
		pct = done * 100 / total
	}
	fmt.Fprintf(b, "### %s — %d%% complete\n", label, pct)
	fmt.Fprintf(b, "- Stories: %d total", total)
	for _, status := range models.ValidStoryStatuses {
		if c := counts[status]; c > 0 {
			fmt.Fprintf(b, ", %d %s", c, status)
		}
	}
	b.WriteString("\n")
}

func (s *Server) handleInitiativeShow(_ context.Context, _ *mcp.CallToolRequest, input slugInput) (*mcp.CallToolResult, any, error) {
	ini, err := s.store.LoadInitiative(input.Slug)
	if err != nil {
		return errResultf("loading initiative: %v", err), nil, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Initiative: %s\n\n", ini.Title)
	fmt.Fprintf(&b, "- **ID:** %s\n", ini.ID)
	fmt.Fprintf(&b, "- **Slug:** %s\n", ini.Slug)
	fmt.Fprintf(&b, "- **Status:** %s\n", ini.Status)
	fmt.Fprintf(&b, "- **Goal:** %s\n", ini.Goal)

	if len(ini.SuccessCriteria) > 0 {
		b.WriteString("\n### Success Criteria\n")
		for _, sc := range ini.SuccessCriteria {
			fmt.Fprintf(&b, "- %s\n", sc)
		}
	}

	// Dynamically discover linked epics instead of reading ini.Epics (which is never populated).
	if epics, listErr := s.store.ListEpics(); listErr == nil {
		var linked []*models.Epic
		for _, ep := range epics {
			if ep.Initiative == ini.Slug {
				linked = append(linked, ep)
			}
		}
		if len(linked) > 0 {
			b.WriteString("\n### Linked Epics\n")
			for _, ep := range linked {
				fmt.Fprintf(&b, "- %s — %s [%s]\n", ep.Slug, ep.Title, ep.Status)
			}
		}
	}

	if len(ini.OpenQuestions) > 0 {
		b.WriteString("\n### Open Questions\n")
		for _, q := range ini.OpenQuestions {
			fmt.Fprintf(&b, "- %s\n", q)
		}
	}

	if ini.Body != "" {
		b.WriteString("\n---\n\n")
		b.WriteString(ini.Body)
	}

	return textResult(b.String()), nil, nil
}

func (s *Server) handleEpicShow(_ context.Context, _ *mcp.CallToolRequest, input slugInput) (*mcp.CallToolResult, any, error) {
	epic, err := s.store.LoadEpic(input.Slug)
	if err != nil {
		return errResultf("loading epic: %v", err), nil, nil
	}

	stories, err := s.store.ListStories(epic.Slug)
	if err != nil {
		return errResultf("listing epic stories: %v", err), nil, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Epic: %s\n\n", epic.Title)
	fmt.Fprintf(&b, "- **ID:** %s\n", epic.ID)
	fmt.Fprintf(&b, "- **Slug:** %s\n", epic.Slug)
	fmt.Fprintf(&b, "- **Status:** %s\n", epic.Status)
	if epic.Initiative != "" {
		fmt.Fprintf(&b, "- **Initiative:** %s\n", epic.Initiative)
	}
	if epic.Fidelity != "" {
		fmt.Fprintf(&b, "- **Fidelity:** %s\n", epic.Fidelity)
	}

	writeEpicPhases(&b, epic.Phases, stories)
	writeMermaidDiagram(&b, epic.Phases, stories)

	if len(stories) > 0 {
		b.WriteString("\n### Stories\n")
		for _, st := range stories {
			fmt.Fprintf(&b, "- %s — %s [%s] (%s)\n", st.Slug, st.Title, st.Status, st.Priority)
		}
	}

	writeStringSection(&b, "Open Questions", epic.OpenQuestions)
	writeStringSection(&b, "Non-Goals", epic.NonGoals)
	writeStringSection(&b, "Decisions", epic.Decisions)

	if epic.Body != "" {
		b.WriteString("\n---\n\n")
		b.WriteString(epic.Body)
	}

	return textResult(b.String()), nil, nil
}

// writeEpicPhases writes the phases section with story status lookup.
func writeEpicPhases(b *strings.Builder, phases []models.Phase, stories []*models.Story) {
	if len(phases) == 0 {
		return
	}
	storyStatus := make(map[string]string, len(stories))
	for _, st := range stories {
		storyStatus[st.Slug] = st.Status
	}
	b.WriteString("\n### Phases\n")
	for _, phase := range phases {
		fmt.Fprintf(b, "\n**%s:**\n", phase.Label)
		for _, slug := range phase.Stories {
			status := storyStatus[slug]
			if status == "" {
				status = "?"
			}
			fmt.Fprintf(b, "- %s [%s]\n", slug, status)
		}
	}
}

// writeMermaidDiagram appends a Mermaid flowchart showing phases and story statuses.
func writeMermaidDiagram(b *strings.Builder, phases []models.Phase, stories []*models.Story) {
	if len(phases) == 0 {
		return
	}

	storyStatus := make(map[string]string, len(stories))
	for _, st := range stories {
		storyStatus[st.Slug] = st.Status
	}

	b.WriteString("\n### Phase Diagram\n\n```mermaid\nflowchart LR\n")

	// Style classes for status colors.
	b.WriteString("    classDef done fill:#2ea44f,color:#fff\n")
	b.WriteString("    classDef in_progress fill:#dbab09,color:#000\n")
	b.WriteString("    classDef planned fill:#8b949e,color:#fff\n")
	b.WriteString("    classDef draft fill:#8b949e,color:#fff\n")
	b.WriteString("    classDef blocked fill:#f85149,color:#fff\n")

	// Emit subgraphs per phase.
	for i, phase := range phases {
		fmt.Fprintf(b, "    subgraph P%d[\"%s\"]\n", i, phase.Label)
		for _, slug := range phase.Stories {
			status := storyStatus[slug]
			if status == "" {
				status = "planned"
			}
			// Use slug as node ID, display slug + status.
			nodeID := mermaidNodeID(slug)
			fmt.Fprintf(b, "        %s[\"%s\\n[%s]\"]\n", nodeID, slug, status)
			fmt.Fprintf(b, "        class %s %s\n", nodeID, status)
		}
		b.WriteString("    end\n")
	}

	// Connect phases sequentially.
	for i := 0; i < len(phases)-1; i++ {
		if len(phases[i].Stories) > 0 && len(phases[i+1].Stories) > 0 {
			from := mermaidNodeID(phases[i].Stories[len(phases[i].Stories)-1])
			to := mermaidNodeID(phases[i+1].Stories[0])
			fmt.Fprintf(b, "    %s --> %s\n", from, to)
		}
	}

	b.WriteString("```\n")
}

// mermaidNodeID converts a slug to a valid Mermaid node identifier by replacing hyphens.
func mermaidNodeID(slug string) string {
	return strings.ReplaceAll(slug, "-", "_")
}

// writeStringSection writes a titled markdown section of string items.
func writeStringSection(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "\n### %s\n", title)
	for _, item := range items {
		fmt.Fprintf(b, "- %s\n", item)
	}
}

// writeChecklistSection writes a titled markdown section with checkbox items.
func writeChecklistSection(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "\n### %s\n", title)
	for _, item := range items {
		fmt.Fprintf(b, "- [ ] %s\n", item)
	}
}

func (s *Server) handleStoryShow(_ context.Context, _ *mcp.CallToolRequest, input slugInput) (*mcp.CallToolResult, any, error) {
	story, err := s.findStory(input.Slug)
	if err != nil {
		return errResultf("finding story: %v", err), nil, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Story: %s\n\n", story.Title)
	fmt.Fprintf(&b, "- **ID:** %s\n", story.ID)
	fmt.Fprintf(&b, "- **Slug:** %s\n", story.Slug)
	fmt.Fprintf(&b, "- **Status:** %s\n", story.Status)
	fmt.Fprintf(&b, "- **Priority:** %s\n", story.Priority)
	if story.Epic != "" {
		fmt.Fprintf(&b, "- **Epic:** %s\n", story.Epic)
	}
	if story.Fidelity != "" {
		fmt.Fprintf(&b, "- **Fidelity:** %s\n", story.Fidelity)
	}
	if len(story.Labels) > 0 {
		fmt.Fprintf(&b, "- **Labels:** %s\n", strings.Join(story.Labels, ", "))
	}

	writeStringSection(&b, "Blocked By", story.BlockedBy)
	writeChecklistSection(&b, "Acceptance Criteria", story.Acceptance)
	writeStringSection(&b, "Doc References", story.DocRefs)
	writeStringSection(&b, "Open Questions", story.OpenQuestions)
	writeStringSection(&b, "Non-Goals", story.NonGoals)
	writeStringSection(&b, "Assumptions", story.Assumptions)

	if story.Body != "" {
		b.WriteString("\n---\n\n")
		b.WriteString(story.Body)
	}

	return textResult(b.String()), nil, nil
}

func (s *Server) handleStoryNext(_ context.Context, _ *mcp.CallToolRequest, input storyNextInput) (*mcp.CallToolResult, any, error) {
	allStories, err := s.store.ListAllStories()
	if err != nil {
		return errResultf("listing stories: %v", err), nil, nil
	}

	statusMap := buildStatusMap(allStories)
	archivedSlugs := s.buildArchivedSlugSet()

	var candidates []*models.Story
	for _, st := range allStories {
		if st.Status != models.StoryStatusPlanned {
			continue
		}
		if input.Epic != "" && st.Epic != input.Epic {
			continue
		}
		if len(st.BlockedBy) > 0 && !allBlockersDone(st.BlockedBy, statusMap, archivedSlugs) {
			continue
		}
		candidates = append(candidates, st)
	}

	if len(candidates) == 0 {
		return textResult("No planned stories available. All stories are either in progress, blocked, or done."), nil, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		ri := priorityRank[candidates[i].Priority]
		rj := priorityRank[candidates[j].Priority]
		if ri != rj {
			return ri < rj
		}
		// Tie-break by creation time (earlier first).
		return candidates[i].Created.Before(candidates[j].Created)
	})

	st := candidates[0]
	var b strings.Builder
	fmt.Fprintf(&b, "## Next Story: %s\n\n", st.Title)
	fmt.Fprintf(&b, "- **Slug:** %s\n", st.Slug)
	fmt.Fprintf(&b, "- **Priority:** %s\n", st.Priority)
	if st.Epic != "" {
		fmt.Fprintf(&b, "- **Epic:** %s\n", st.Epic)
	}
	if len(st.Acceptance) > 0 {
		b.WriteString("\n### Acceptance Criteria\n")
		for _, ac := range st.Acceptance {
			fmt.Fprintf(&b, "- [ ] %s\n", ac)
		}
	}

	if len(candidates) > 1 {
		fmt.Fprintf(&b, "\n_(%d other planned stories in queue)_\n", len(candidates)-1)
	}

	return textResult(b.String()), nil, nil
}

func (s *Server) handleStoryLs(_ context.Context, _ *mcp.CallToolRequest, input storyLsInput) (*mcp.CallToolResult, any, error) {
	var stories []*models.Story
	var err error

	if input.Epic != "" {
		stories, err = s.store.ListStories(input.Epic)
	} else {
		stories, err = s.store.ListAllStories()
	}
	if err != nil {
		return errResultf("listing stories: %v", err), nil, nil
	}

	if input.IncludeArchived {
		stories = append(stories, s.collectArchivedStories(input.Epic)...)
	}

	// Apply filters.
	var filtered []*models.Story
	for _, st := range stories {
		if input.Status != "" && st.Status != input.Status {
			continue
		}
		if input.Label != "" && !containsString(st.Labels, input.Label) {
			continue
		}
		if input.Blocked && len(st.BlockedBy) == 0 {
			continue
		}
		filtered = append(filtered, st)
	}

	if len(filtered) == 0 {
		return textResult("No stories match the given filters."), nil, nil
	}

	var b strings.Builder
	b.WriteString("## Stories\n\n")
	b.WriteString("| Slug | Title | Status | Priority | Epic |\n")
	b.WriteString("|------|-------|--------|----------|------|\n")
	for _, st := range filtered {
		epic := st.Epic
		if epic == "" {
			epic = "-"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", st.Slug, st.Title, st.Status, st.Priority, epic)
	}

	fmt.Fprintf(&b, "\n_%d stories_\n", len(filtered))

	return textResult(b.String()), nil, nil
}

func (s *Server) handleDocRead(_ context.Context, _ *mcp.CallToolRequest, input docReadInput) (*mcp.CallToolResult, any, error) {
	doc, err := s.store.LoadDoc(input.Slug, input.Epic)
	if err != nil {
		return errResultf("loading doc: %v", err), nil, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Document: %s\n\n", doc.Title)
	fmt.Fprintf(&b, "- **ID:** %s\n", doc.ID)
	fmt.Fprintf(&b, "- **Slug:** %s\n", doc.Slug)
	fmt.Fprintf(&b, "- **Type:** %s\n", doc.Type)
	fmt.Fprintf(&b, "- **Status:** %s\n", doc.Status)
	if doc.Epic != "" {
		fmt.Fprintf(&b, "- **Epic:** %s\n", doc.Epic)
	}

	if len(doc.OpenQuestions) > 0 {
		b.WriteString("\n### Open Questions\n")
		for _, q := range doc.OpenQuestions {
			fmt.Fprintf(&b, "- %s\n", q)
		}
	}

	if doc.Body != "" {
		b.WriteString("\n---\n\n")
		b.WriteString(doc.Body)
	}

	return textResult(b.String()), nil, nil
}

func (s *Server) handlePlanRead(_ context.Context, _ *mcp.CallToolRequest, input storyRefInput) (*mcp.CallToolResult, any, error) {
	plan, err := s.store.LoadPlan(input.Story)
	if err != nil {
		return textResult(fmt.Sprintf("No plan exists for story %q.", input.Story)), nil, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Plan for %s\n\n", plan.Story)
	fmt.Fprintf(&b, "- **ID:** %s\n", plan.ID)
	fmt.Fprintf(&b, "- **Status:** %s\n", plan.Status)
	if plan.EstimatedFiles > 0 {
		fmt.Fprintf(&b, "- **Estimated files:** %d\n", plan.EstimatedFiles)
	}
	if plan.GitRefBaseline != "" {
		fmt.Fprintf(&b, "- **Git baseline:** %s\n", plan.GitRefBaseline)
	}

	if plan.Body != "" {
		b.WriteString("\n---\n\n")
		b.WriteString(plan.Body)
	}

	return textResult(b.String()), nil, nil
}

func (s *Server) handleVerifyRead(_ context.Context, _ *mcp.CallToolRequest, input storyRefInput) (*mcp.CallToolResult, any, error) {
	v, err := s.store.LatestVerification(input.Story)
	if err != nil {
		return textResult(fmt.Sprintf("No verification exists for story %q.", input.Story)), nil, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Verification for %s\n\n", v.Story)
	fmt.Fprintf(&b, "- **ID:** %s\n", v.ID)
	fmt.Fprintf(&b, "- **Execution:** %s\n", v.Execution)
	fmt.Fprintf(&b, "- **Result:** %s\n", v.Result)
	fmt.Fprintf(&b, "- **Stats:** critical=%d, major=%d, minor=%d\n", v.Stats.Critical, v.Stats.Major, v.Stats.Minor)

	if len(v.AcceptanceCheck) > 0 {
		b.WriteString("\n### Acceptance Check\n")
		for _, ac := range v.AcceptanceCheck {
			check := "[ ]"
			if ac.Met {
				check = "[x]"
			}
			fmt.Fprintf(&b, "- %s %s\n", check, ac.Criteria)
		}
	}

	if len(v.Findings) > 0 {
		b.WriteString("\n### Findings\n")
		for _, f := range v.Findings {
			loc := ""
			if f.File != "" {
				loc = fmt.Sprintf(" (%s)", f.File)
			}
			fmt.Fprintf(&b, "- **%s/%s**%s: %s\n", f.Severity, f.Category, loc, f.Description)
			if f.Suggestion != "" {
				fmt.Fprintf(&b, "  - Suggestion: %s\n", f.Suggestion)
			}
		}
	}

	if v.Body != "" {
		b.WriteString("\n---\n\n")
		b.WriteString(v.Body)
	}

	return textResult(b.String()), nil, nil
}

func (s *Server) handleContextBuild(_ context.Context, _ *mcp.CallToolRequest, input storyRefInput) (*mcp.CallToolResult, any, error) {
	assembled, err := s.builder.Build(input.Story)
	if err != nil {
		return errResultf("building context: %v", err), nil, nil
	}

	return textResult(assembled), nil, nil
}

type sourceQuestion struct {
	source   string
	question string
}

func (s *Server) handleQuestions(_ context.Context, _ *mcp.CallToolRequest, input epicOptInput) (*mcp.CallToolResult, any, error) {
	questions, err := s.collectQuestions(input.Epic)
	if err != nil {
		return errResultf("%v", err), nil, nil
	}

	if len(questions) == 0 {
		return textResult("No open questions found."), nil, nil
	}

	return textResult(formatGroupedQuestions(questions)), nil, nil
}

func (s *Server) collectQuestions(epicFilter string) ([]sourceQuestion, error) {
	var questions []sourceQuestion

	initiatives, err := s.store.ListInitiatives()
	if err != nil {
		return nil, fmt.Errorf("listing initiatives: %w", err)
	}
	for _, ini := range initiatives {
		for _, q := range ini.OpenQuestions {
			questions = append(questions, sourceQuestion{source: "initiative:" + ini.Slug, question: q})
		}
	}

	epics, err := s.store.ListEpics()
	if err != nil {
		return nil, fmt.Errorf("listing epics: %w", err)
	}
	for _, epic := range epics {
		if epicFilter != "" && epic.Slug != epicFilter {
			continue
		}
		for _, q := range epic.OpenQuestions {
			questions = append(questions, sourceQuestion{source: "epic:" + epic.Slug, question: q})
		}
	}

	allStories, err := s.store.ListAllStories()
	if err != nil {
		return nil, fmt.Errorf("listing stories: %w", err)
	}
	for _, st := range allStories {
		if epicFilter != "" && st.Epic != epicFilter {
			continue
		}
		for _, q := range st.OpenQuestions {
			questions = append(questions, sourceQuestion{source: "story:" + st.Slug, question: q})
		}
	}

	return questions, nil
}

func formatGroupedQuestions(questions []sourceQuestion) string {
	grouped := make(map[string][]string)
	var sources []string
	for _, q := range questions {
		if _, exists := grouped[q.source]; !exists {
			sources = append(sources, q.source)
		}
		grouped[q.source] = append(grouped[q.source], q.question)
	}

	var b strings.Builder
	b.WriteString("## Open Questions\n\n")
	for _, src := range sources {
		fmt.Fprintf(&b, "### %s\n", src)
		for _, q := range grouped[src] {
			fmt.Fprintf(&b, "- %s\n", q)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "_%d questions total_\n", len(questions))
	return b.String()
}

func (s *Server) handleBlocked(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	allStories, err := s.store.ListAllStories()
	if err != nil {
		return errResultf("listing stories: %v", err), nil, nil
	}

	statusMap := buildStatusMap(allStories)
	archivedSlugs := s.buildArchivedSlugSet()

	var blocked []*models.Story
	for _, st := range allStories {
		if len(st.BlockedBy) > 0 && !allBlockersDone(st.BlockedBy, statusMap, archivedSlugs) {
			blocked = append(blocked, st)
		}
	}

	if len(blocked) == 0 {
		return textResult("No blocked stories."), nil, nil
	}

	var b strings.Builder
	b.WriteString("## Blocked Stories\n\n")
	for _, st := range blocked {
		fmt.Fprintf(&b, "### %s — %s\n", st.Slug, st.Title)
		if st.Epic != "" {
			fmt.Fprintf(&b, "- **Epic:** %s\n", st.Epic)
		}
		fmt.Fprintf(&b, "- **Status:** %s\n", st.Status)
		b.WriteString("- **Blocked by:**\n")
		for _, blocker := range st.BlockedBy {
			status, ok := statusMap[blocker]
			if !ok {
				status = "unknown"
			}
			fmt.Fprintf(&b, "  - %s (%s)\n", blocker, status)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "_%d blocked stories_\n", len(blocked))

	return textResult(b.String()), nil, nil
}

func (s *Server) handleDecisions(_ context.Context, _ *mcp.CallToolRequest, input epicOptInput) (*mcp.CallToolResult, any, error) {
	decisions, err := s.store.ListDecisions()
	if err != nil {
		return errResultf("listing decisions: %v", err), nil, nil
	}

	if input.Epic != "" {
		var filtered []*models.Decision
		for _, d := range decisions {
			for _, ref := range d.ContextRefs {
				if ref == input.Epic {
					filtered = append(filtered, d)
					break
				}
			}
		}
		decisions = filtered
	}

	if len(decisions) == 0 {
		return textResult("No decisions found."), nil, nil
	}

	var b strings.Builder
	b.WriteString("## Decisions\n\n")
	for _, d := range decisions {
		fmt.Fprintf(&b, "### %s — %s\n", d.Slug, d.Title)
		fmt.Fprintf(&b, "- **ID:** %s\n", d.ID)
		fmt.Fprintf(&b, "- **Date:** %s\n", d.Date)
		fmt.Fprintf(&b, "- **Status:** %s\n", d.Status)
		if len(d.ContextRefs) > 0 {
			fmt.Fprintf(&b, "- **Context refs:** %s\n", strings.Join(d.ContextRefs, ", "))
		}
		if d.Body != "" {
			b.WriteString("\n")
			b.WriteString(d.Body)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return textResult(b.String()), nil, nil
}

func (s *Server) handleLog(_ context.Context, _ *mcp.CallToolRequest, input logInput) (*mcp.CallToolResult, any, error) {
	last := input.Last
	if last <= 0 {
		last = 20
	}

	entries, err := s.store.ReadLog(last)
	if err != nil {
		return errResultf("reading log: %v", err), nil, nil
	}

	if len(entries) == 0 {
		return textResult("No log entries."), nil, nil
	}

	var b strings.Builder
	b.WriteString("## Activity Log\n\n")
	for idx := range entries {
		e := &entries[idx]
		ts := e.Timestamp.Format("2006-01-02 15:04:05")
		fmt.Fprintf(&b, "- **%s** `%s` %s", ts, e.Type, e.Entity)
		if e.From != "" || e.To != "" {
			fmt.Fprintf(&b, " (%s -> %s)", e.From, e.To)
		}
		if e.Result != "" {
			fmt.Fprintf(&b, " [%s]", e.Result)
		}
		if e.GitRef != "" {
			fmt.Fprintf(&b, " @%s", e.GitRef)
		}
		if e.FilesChanged > 0 {
			fmt.Fprintf(&b, " (%d files)", e.FilesChanged)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "\n_%d entries_\n", len(entries))

	return textResult(b.String()), nil, nil
}

func (s *Server) handleAssumptions(_ context.Context, _ *mcp.CallToolRequest, input epicOptInput) (*mcp.CallToolResult, any, error) {
	allStories, err := s.store.ListAllStories()
	if err != nil {
		return errResultf("listing stories: %v", err), nil, nil
	}

	type assumption struct {
		epic       string
		storySlug  string
		assumption string
	}
	var assumptions []assumption

	for _, st := range allStories {
		if input.Epic != "" && st.Epic != input.Epic {
			continue
		}
		for _, a := range st.Assumptions {
			epic := st.Epic
			if epic == "" {
				epic = "(standalone)"
			}
			assumptions = append(assumptions, assumption{epic: epic, storySlug: st.Slug, assumption: a})
		}
	}

	if len(assumptions) == 0 {
		return textResult("No assumptions found."), nil, nil
	}

	// Group by epic.
	grouped := make(map[string][]assumption)
	var epicOrder []string
	for _, a := range assumptions {
		if _, exists := grouped[a.epic]; !exists {
			epicOrder = append(epicOrder, a.epic)
		}
		grouped[a.epic] = append(grouped[a.epic], a)
	}

	var b strings.Builder
	b.WriteString("## Assumptions\n\n")
	for _, epic := range epicOrder {
		fmt.Fprintf(&b, "### Epic: %s\n", epic)
		for _, a := range grouped[epic] {
			fmt.Fprintf(&b, "- [%s] %s\n", a.storySlug, a.assumption)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "_%d assumptions total_\n", len(assumptions))

	return textResult(b.String()), nil, nil
}

func (s *Server) handleDiff(_ context.Context, _ *mcp.CallToolRequest, input diffInput) (*mcp.CallToolResult, any, error) {
	var from, to string

	switch {
	case input.Refs != "":
		// Explicit ref range provided.
		parts := strings.SplitN(input.Refs, "..", 2)
		if len(parts) != 2 {
			return errResult("refs must be in the format 'from..to'"), nil, nil
		}
		from, to = parts[0], parts[1]
	case input.Story != "":
		// Look up the latest execution's git baseline.
		latest, err := s.store.LatestExecution(input.Story)
		if err != nil {
			return errResultf("finding execution for story %q: %v", input.Story, err), nil, nil
		}
		if latest.GitRefBefore == "" {
			return errResult("execution has no git baseline recorded"), nil, nil
		}
		from = latest.GitRefBefore
		to = "HEAD"
	default:
		return errResult("either story or refs is required"), nil, nil
	}

	diff, err := git.Diff(from, to)
	if err != nil {
		return errResultf("getting diff: %v", err), nil, nil
	}

	if diff == "" {
		return textResult(fmt.Sprintf("No changes between %s and %s.", from, to)), nil, nil
	}

	return textResult(fmt.Sprintf("```diff\n%s```", diff)), nil, nil
}

func (s *Server) handleHardQuestions(_ context.Context, _ *mcp.CallToolRequest, input hardQuestionsInput) (*mcp.CallToolResult, any, error) {
	if input.Entity == "" {
		return errResult("entity is required"), nil, nil
	}

	if s.cfg.Mode == "fast" {
		return textResult("Hard questions suppressed in fast mode. Switch to careful mode to enable."), nil, nil
	}

	entityType := s.detectEntityType(input.Entity)
	if entityType == "" {
		return errResultf("entity %q not found", input.Entity), nil, nil
	}

	questions := hardq.Questions(entityType)
	if len(questions) == 0 {
		return textResult(fmt.Sprintf("No hard questions defined for %s entities.", entityType)), nil, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Hard Questions for %s (`%s`)\n\n", entityType, input.Entity)
	for i, q := range questions {
		fmt.Fprintf(&b, "%d. %s\n", i+1, q)
	}
	return textResult(b.String()), nil, nil
}

func (s *Server) handleReviewPrompt(_ context.Context, _ *mcp.CallToolRequest, input reviewPromptInput) (*mcp.CallToolResult, any, error) {
	if input.Doc == "" {
		return errResult("doc is required"), nil, nil
	}

	doc, err := s.store.LoadDoc(input.Doc, input.Epic)
	if err != nil {
		// Try searching under epics if no epic was provided.
		if input.Epic == "" {
			if found, _, findErr := s.findDoc(input.Doc); findErr == nil {
				doc = found
			} else {
				return errResultf("loading doc %q: %v", input.Doc, err), nil, nil
			}
		} else {
			return errResultf("loading doc %q: %v", input.Doc, err), nil, nil
		}
	}

	docType := hardq.EntityType(doc.Type)
	prompt := hardq.ReviewPrompt(docType)
	if prompt == "" {
		// Fall back to decompose prompt for specs without a specific review template.
		prompt = hardq.DecomposePrompt
		prompt = strings.ReplaceAll(prompt, "{{spec_content}}", doc.Body)
	} else {
		prompt = strings.ReplaceAll(prompt, "{{doc_content}}", doc.Body)
	}

	return textResult(prompt), nil, nil
}

func (s *Server) handleScopeDrift(_ context.Context, _ *mcp.CallToolRequest, input storyRefInput) (*mcp.CallToolResult, any, error) {
	if input.Story == "" {
		return errResult("story is required"), nil, nil
	}

	plan, err := s.store.LoadPlan(input.Story)
	if err != nil {
		return errResultf("no plan found for story %q", input.Story), nil, nil
	}

	latest, err := s.store.LatestExecution(input.Story)
	if err != nil {
		return errResultf("no execution found for story %q", input.Story), nil, nil
	}

	diffContent := executionDiffContent(latest)

	prompt := hardq.ScopeDriftPrompt
	prompt = strings.ReplaceAll(prompt, "{{plan_content}}", plan.Body)
	prompt = strings.ReplaceAll(prompt, "{{diff_content}}", diffContent)

	return textResult(prompt), nil, nil
}

// executionDiffContent returns the git diff for an execution.
// Falls back to HEAD when git_ref_after is missing or equals git_ref_before.
func executionDiffContent(exec *models.Execution) string {
	if exec.GitRefBefore == "" {
		return "(no diff available — execution has no git baseline)"
	}
	to := "HEAD"
	if exec.GitRefAfter != "" && exec.GitRefAfter != exec.GitRefBefore {
		to = exec.GitRefAfter
	}
	if d, err := git.Diff(exec.GitRefBefore, to); err == nil && d != "" {
		return d
	}
	return "(no diff available)"
}

func (s *Server) handleUnstuck(_ context.Context, _ *mcp.CallToolRequest, input storyRefInput) (*mcp.CallToolResult, any, error) {
	if input.Story == "" {
		return errResult("story is required"), nil, nil
	}

	story, err := s.findStory(input.Story)
	if err != nil {
		return errResultf("finding story: %v", err), nil, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Debugging Context for `%s`\n\n", story.Title)
	fmt.Fprintf(&b, "**Story:** %s | **Status:** %s | **Priority:** %s\n\n", story.Slug, story.Status, story.Priority)

	writeChecklistSection(&b, "What We're Trying to Achieve", story.Acceptance)
	s.writeUnstuckPlan(&b, input.Story)
	s.writeUnstuckExecution(&b, input.Story)
	writeStringSection(&b, "Open Questions", story.OpenQuestions)
	writeStringSection(&b, "Assumptions", story.Assumptions)
	s.writeUnstuckLog(&b)
	writeDebuggingSteps(&b)

	return textResult(b.String()), nil, nil
}

func (s *Server) writeUnstuckPlan(b *strings.Builder, storySlug string) {
	plan, err := s.store.LoadPlan(storySlug)
	if err != nil {
		return
	}
	b.WriteString("\n### Implementation Plan\n")
	b.WriteString(plan.Body)
	b.WriteString("\n\n")
}

func (s *Server) writeUnstuckExecution(b *strings.Builder, storySlug string) {
	latest, err := s.store.LatestExecution(storySlug)
	if err != nil {
		return
	}

	fmt.Fprintf(b, "\n### Current Execution\n")
	fmt.Fprintf(b, "- **ID:** %s\n", latest.ID)
	fmt.Fprintf(b, "- **Status:** %s\n", latest.Status)
	fmt.Fprintf(b, "- **Files changed:** %d\n\n", len(latest.FilesChanged))

	if latest.GitRefBefore != "" {
		if diff, diffErr := git.Diff(latest.GitRefBefore, "HEAD"); diffErr == nil && diff != "" {
			b.WriteString("### Changes So Far\n\n```diff\n")
			const maxDiff = 5000
			if len(diff) > maxDiff {
				b.WriteString(diff[:maxDiff])
				b.WriteString("\n... (truncated)\n")
			} else {
				b.WriteString(diff)
			}
			b.WriteString("```\n\n")
		}
	}

	if notes, hErr := s.store.LoadHandover(storySlug, latest.ID); hErr == nil {
		b.WriteString("### Handover Notes from Previous Session\n")
		b.WriteString(notes)
		b.WriteString("\n\n")
	}
}

func (s *Server) writeUnstuckLog(b *strings.Builder) {
	entries, err := s.store.ReadLog(10)
	if err != nil || len(entries) == 0 {
		return
	}
	b.WriteString("\n### Recent Activity\n")
	for idx := range entries {
		e := &entries[idx]
		ts := e.Timestamp.Format("15:04:05")
		fmt.Fprintf(b, "- %s `%s` %s", ts, e.Type, e.Entity)
		if e.From != "" || e.To != "" {
			fmt.Fprintf(b, " (%s -> %s)", e.From, e.To)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func writeDebuggingSteps(b *strings.Builder) {
	b.WriteString("\n### Suggested Debugging Steps\n\n")
	b.WriteString("1. **Reproduce** — Can you trigger the issue reliably? Write a failing test if possible.\n")
	b.WriteString("2. **Isolate** — Binary search: which change introduced the problem? Comment out recent changes.\n")
	b.WriteString("3. **Hypothesize** — Based on the diff and error, what's the most likely root cause?\n")
	b.WriteString("4. **Verify** — Test your hypothesis with the smallest possible change.\n")
	b.WriteString("5. **Fix** — Apply the fix, run tests, verify acceptance criteria.\n")
}

func (s *Server) handleCodeReview(_ context.Context, _ *mcp.CallToolRequest, input storyRefInput) (*mcp.CallToolResult, any, error) {
	if input.Story == "" {
		return errResult("story is required"), nil, nil
	}

	// Load story for acceptance criteria.
	story, err := s.findStory(input.Story)
	if err != nil {
		return errResultf("finding story: %v", err), nil, nil
	}

	// Load latest execution for the diff.
	latest, err := s.store.LatestExecution(input.Story)
	if err != nil {
		return errResultf("no execution found for story %q: %v", input.Story, err), nil, nil
	}

	// Build acceptance criteria text.
	var acText strings.Builder
	for _, ac := range story.Acceptance {
		fmt.Fprintf(&acText, "- [ ] %s\n", ac)
	}
	if acText.Len() == 0 {
		acText.WriteString("(none specified)\n")
	}

	// Load plan if available.
	planContent := "(no plan exists)"
	if plan, planErr := s.store.LoadPlan(input.Story); planErr == nil {
		planContent = plan.Body
	}

	// Get diff.
	diffContent := executionDiffContent(latest)

	// Assemble the review prompt.
	prompt := hardq.CodeReviewPrompt
	prompt = strings.ReplaceAll(prompt, "{{acceptance_criteria}}", acText.String())
	prompt = strings.ReplaceAll(prompt, "{{plan_content}}", planContent)
	prompt = strings.ReplaceAll(prompt, "{{diff_content}}", diffContent)

	return textResult(prompt), nil, nil
}

// detectEntityType probes the store to determine what type an entity slug is.
func (s *Server) detectEntityType(slug string) hardq.EntityType {
	if _, err := s.store.LoadInitiative(slug); err == nil {
		return hardq.Initiative
	}
	if _, err := s.store.LoadEpic(slug); err == nil {
		return hardq.Epic
	}
	if _, err := s.findStory(slug); err == nil {
		return hardq.Story
	}
	// Try doc — project-level first, then under epics.
	if doc, err := s.store.LoadDoc(slug, ""); err == nil {
		return hardq.EntityType(doc.Type)
	}
	if doc, _, err := s.findDoc(slug); err == nil {
		return hardq.EntityType(doc.Type)
	}
	return ""
}

func (s *Server) handleScopeCheck(_ context.Context, _ *mcp.CallToolRequest, input epicRequiredInput) (*mcp.CallToolResult, any, error) {
	if input.Epic == "" {
		return errResult("epic is required"), nil, nil
	}

	// Find the PRD for this epic.
	prd, err := s.findPRD(input.Epic)
	if err != nil {
		return errResultf("no PRD found for epic %q: %v", input.Epic, err), nil, nil
	}

	stories, err := s.store.ListStories(input.Epic)
	if err != nil {
		return errResultf("listing stories: %v", err), nil, nil
	}

	// Extract scope sections from PRD body.
	inScope, outScope, userStories := parseScopeSections(prd.Body)

	untraced := findUntracedStories(stories, userStories, inScope)
	outOfScope := findOutOfScopeStories(stories, outScope)

	var b strings.Builder
	fmt.Fprintf(&b, "## Scope Check for epic `%s`\n\n", input.Epic)
	fmt.Fprintf(&b, "**PRD:** %s (`%s`)\n\n", prd.Title, prd.Slug)

	if len(untraced) == 0 && len(outOfScope) == 0 {
		b.WriteString("All stories are traceable to the PRD scope. No issues found.\n")
		return textResult(b.String()), nil, nil
	}

	if len(untraced) > 0 {
		b.WriteString("### Stories not traceable to PRD\n\n")
		b.WriteString("These stories couldn't be matched to PRD user stories or in-scope items:\n\n")
		for _, st := range untraced {
			fmt.Fprintf(&b, "- **%s** — %s [%s]\n", st.Slug, st.Title, st.Status)
		}
		b.WriteString("\n")
	}

	if len(outOfScope) > 0 {
		b.WriteString("### Stories matching out-of-scope\n\n")
		b.WriteString("These stories appear to match items explicitly marked as out-of-scope:\n\n")
		for _, st := range outOfScope {
			fmt.Fprintf(&b, "- **%s** — %s [%s]\n", st.Slug, st.Title, st.Status)
		}
		b.WriteString("\n")
	}

	return textResult(b.String()), nil, nil
}

// findUntracedStories returns stories whose titles can't be matched to PRD user stories or in-scope text.
func findUntracedStories(stories []*models.Story, userStories []string, inScope string) []*models.Story {
	inScopeLower := strings.ToLower(inScope)
	var untraced []*models.Story
	for _, st := range stories {
		titleLower := strings.ToLower(st.Title)
		if storyMatchesScope(titleLower, userStories, inScopeLower) {
			continue
		}
		untraced = append(untraced, st)
	}
	return untraced
}

func storyMatchesScope(titleLower string, userStories []string, inScopeLower string) bool {
	for _, us := range userStories {
		usLower := strings.ToLower(us)
		if strings.Contains(usLower, titleLower) || strings.Contains(titleLower, usLower) {
			return true
		}
	}
	return strings.Contains(inScopeLower, titleLower)
}

// findOutOfScopeStories returns stories whose titles match the out-of-scope text.
func findOutOfScopeStories(stories []*models.Story, outScope string) []*models.Story {
	if outScope == "" {
		return nil
	}
	outLower := strings.ToLower(outScope)
	var result []*models.Story
	for _, st := range stories {
		if strings.Contains(outLower, strings.ToLower(st.Title)) {
			result = append(result, st)
		}
	}
	return result
}

// findPRD looks for a PRD document under the given epic, then project-level.
func (s *Server) findPRD(epicSlug string) (*models.Document, error) {
	docs, err := s.store.ListDocs(epicSlug)
	if err == nil {
		for _, d := range docs {
			if d.Type == models.DocTypePRD {
				return d, nil
			}
		}
	}
	// Try project-level docs.
	docs, err = s.store.ListDocs("")
	if err != nil {
		return nil, fmt.Errorf("listing project docs: %w", err)
	}
	for _, d := range docs {
		if d.Type == models.DocTypePRD {
			return d, nil
		}
	}
	return nil, fmt.Errorf("no PRD document found")
}

// parseScopeSections extracts In Scope, Out of Scope, and User Stories sections from PRD markdown.
func parseScopeSections(body string) (inScope, outScope string, userStories []string) {
	lines := strings.Split(body, "\n")
	var currentSection string
	var sectionLines []string

	flushSection := func() {
		content := strings.TrimSpace(strings.Join(sectionLines, "\n"))
		switch {
		case strings.Contains(strings.ToLower(currentSection), "in scope") && !strings.Contains(strings.ToLower(currentSection), "out"):
			inScope = content
		case strings.Contains(strings.ToLower(currentSection), "out of scope") || strings.Contains(strings.ToLower(currentSection), "out-of-scope"):
			outScope = content
		case strings.Contains(strings.ToLower(currentSection), "user stor"):
			for _, line := range sectionLines {
				trimmed := strings.TrimSpace(line)
				trimmed = strings.TrimLeft(trimmed, "-*• ")
				if trimmed != "" {
					userStories = append(userStories, trimmed)
				}
			}
		}
		sectionLines = nil
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			flushSection()
			currentSection = line
			continue
		}
		sectionLines = append(sectionLines, line)
	}
	flushSection()

	return inScope, outScope, userStories
}

// buildStatusMap creates a slug→status lookup from a list of stories.
func buildStatusMap(stories []*models.Story) map[string]string {
	m := make(map[string]string, len(stories))
	for _, st := range stories {
		m[st.Slug] = st.Status
	}
	return m
}

// collectArchivedStories returns stories from all archived epics and standalone archive, optionally filtered by epic slug.
func (s *Server) collectArchivedStories(epicFilter string) []*models.Story {
	var result []*models.Story
	archivedEpics, err := s.store.ListArchivedEpics()
	if err == nil {
		for _, ep := range archivedEpics {
			if epicFilter != "" && ep.Slug != epicFilter {
				continue
			}
			archStories, stErr := s.store.ListArchivedStories(ep.Slug)
			if stErr == nil {
				result = append(result, archStories...)
			}
		}
	}
	// Include archived standalone stories when no epic filter.
	if epicFilter == "" {
		standalone, stErr := s.store.ListArchivedStandaloneStories()
		if stErr == nil {
			result = append(result, standalone...)
		}
	}
	return result
}

// buildArchivedSlugSet returns a set of story slugs from all archived epics and standalone archive.
func (s *Server) buildArchivedSlugSet() map[string]bool {
	slugs := make(map[string]bool)
	archivedEpics, err := s.store.ListArchivedEpics()
	if err == nil {
		for _, ep := range archivedEpics {
			stories, stErr := s.store.ListArchivedStories(ep.Slug)
			if stErr != nil {
				continue
			}
			for _, st := range stories {
				slugs[st.Slug] = true
			}
		}
	}
	standalone, stErr := s.store.ListArchivedStandaloneStories()
	if stErr == nil {
		for _, st := range standalone {
			slugs[st.Slug] = true
		}
	}
	return slugs
}

// allBlockersDone returns true if every slug in blockedBy maps to "done" status.
// Stories not in the status map are checked against the archive — archived stories are treated as done.
func allBlockersDone(blockedBy []string, statusMap map[string]string, archivedSlugs map[string]bool) bool {
	for _, slug := range blockedBy {
		status, ok := statusMap[slug]
		if ok {
			if status != models.StoryStatusDone {
				return false
			}
			continue
		}
		// Not in active stories — check archive.
		if !archivedSlugs[slug] {
			return false
		}
	}
	return true
}

func (s *Server) handleDiffCheck(_ context.Context, _ *mcp.CallToolRequest, input epicRequiredInput) (*mcp.CallToolResult, any, error) {
	if input.Epic == "" {
		return errResult("epic is required"), nil, nil
	}

	stories, err := s.store.ListStories(input.Epic)
	if err != nil {
		return errResultf("listing stories: %v", err), nil, nil
	}

	// Load all docs for the epic + project-level.
	docMap := make(map[string]*models.Document)
	if epicDocs, listErr := s.store.ListDocs(input.Epic); listErr == nil {
		for _, d := range epicDocs {
			docMap[d.Slug] = d
		}
	}
	if projDocs, listErr := s.store.ListDocs(""); listErr == nil {
		for _, d := range projDocs {
			docMap[d.Slug] = d
		}
	}

	type driftItem struct {
		story   *models.Story
		doc     *models.Document
		storyTS string
		docTS   string
	}
	var drifted []driftItem

	for _, st := range stories {
		for _, ref := range st.DocRefs {
			doc, exists := docMap[ref]
			if !exists {
				continue
			}
			if doc.Updated.After(st.Updated) {
				drifted = append(drifted, driftItem{
					story:   st,
					doc:     doc,
					storyTS: st.Updated.Format("2006-01-02 15:04"),
					docTS:   doc.Updated.Format("2006-01-02 15:04"),
				})
			}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Drift Check for epic `%s`\n\n", input.Epic)

	if len(drifted) == 0 {
		b.WriteString("No drift detected. All stories are up-to-date with their referenced documents.\n")
		return textResult(b.String()), nil, nil
	}

	b.WriteString("The following stories reference documents that were updated after the story:\n\n")
	b.WriteString("| Story | Doc | Story Updated | Doc Updated |\n")
	b.WriteString("|-------|-----|---------------|-------------|\n")
	for _, d := range drifted {
		fmt.Fprintf(&b, "| %s | %s (%s) | %s | %s |\n",
			d.story.Slug, d.doc.Slug, d.doc.Type, d.storyTS, d.docTS)
	}
	fmt.Fprintf(&b, "\n_%d potentially drifted stories_\n", len(drifted))

	return textResult(b.String()), nil, nil
}
