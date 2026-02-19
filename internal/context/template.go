package context

// contextTemplate is the embedded Go template for rendering context output.
// Using a Go constant instead of go:embed for simplicity — this can be
// moved to an embedded template file later if user-customizable templates
// are needed.
const contextTemplate = `# Execution Context: {{ .Story.Title }}

**Story:** {{ .Story.Slug }} | **Status:** {{ .Story.Status }} | **Priority:** {{ .Story.Priority }}
{{- if .Story.Epic }}
**Epic:** {{ .Story.Epic }}
{{- end }}

---

## Layer 1: Project Conventions
{{ if .ClaudeMD }}
### CLAUDE.md
{{ .ClaudeMD }}
{{ end }}
{{- if .AgentsMD }}
### AGENTS.md
{{ .AgentsMD }}
{{ end }}
---

## Layer 2: Epic & Initiative Context
{{ if .Initiative }}
### Initiative: {{ .Initiative.Title }}
**Goal:** {{ .Initiative.Goal }}
{{ if .Initiative.SuccessCriteria }}
**Success Criteria:**
{{ range .Initiative.SuccessCriteria }}- {{ . }}
{{ end }}{{ end }}{{ end }}
{{- if .Epic }}
### Epic: {{ .Epic.Title }}
**Status:** {{ .Epic.Status }}
{{- if .Epic.Fidelity }}
**Fidelity:** {{ .Epic.Fidelity }}
{{- end }}
{{ if .Epic.NonGoals }}
#### Non-Goals
{{ range .Epic.NonGoals }}- {{ . }}
{{ end }}{{ end }}
{{ if .Epic.Body }}{{ .Epic.Body }}{{ end }}
{{ if .Epic.Phases }}
#### Phases
{{ range .Epic.Phases }}
- **{{ .Label }}:** {{ range .Stories }}{{ . }} {{ end }}
{{- end }}{{ end }}{{ end }}
{{ if .CompletedStories }}
### Completed Stories
{{ range .CompletedStories }}- [{{ .Status }}] **{{ .Slug }}**: {{ .Title }}
{{ end }}{{ end }}
{{- if .InProgressStories }}
### In-Progress Stories
{{ range .InProgressStories }}- **{{ .Slug }}**: {{ .Title }}
{{ end }}{{ end }}
{{- if .Decisions }}
### Decisions
{{ range .Decisions }}- **{{ .Title }}** ({{ .Date }}, {{ .Status }})
{{ end }}{{ end }}
---

## Layer 3: Spec Requirements
{{- if .Story.Fidelity }}
**Fidelity:** {{ .Story.Fidelity }}
{{- end }}
{{ if .Story.NonGoals }}
### Non-Goals
{{ range .Story.NonGoals }}- {{ . }}
{{ end }}{{ end }}
{{ if .Story.Acceptance }}
### Acceptance Criteria
{{ range .Story.Acceptance }}- [ ] {{ . }}
{{ end }}{{ end }}
{{- if .Docs }}{{ range .Docs }}
### Doc: {{ .Title }} ({{ .Type }})
{{ .Body }}
{{ end }}{{ end }}
{{- if not .Docs }}{{ if not .Story.Acceptance }}
*No specs or acceptance criteria defined for this story.*
{{ end }}{{ end }}
---

## Layer 4: Implementation Plan
{{ if .HandoverNotes }}
### Handover from Previous Session
{{ .HandoverNotes }}
{{ end }}
{{ if .HasPlan }}
**Status:** {{ .Plan.Status }}
{{ .Plan.Body }}
{{ else }}
*No implementation plan exists yet. Create one before starting implementation.*
{{ end }}
---

## Layer 5: Referenced Files
{{ if .ReferencedFiles }}{{ range .ReferencedFiles }}
### {{ .Path }}
{{ if .Exists }}` + "```" + `
{{ .Content }}` + "```" + `
{{ else }}*File does not exist yet (planned but not created).*
{{ end }}{{ end }}{{ else }}
*No referenced files.*
{{ end }}
---

## Layer 6: Open Items
{{ if .OpenQuestions }}
### Open Questions
{{ range .OpenQuestions }}- [{{ .Source }}] {{ .Question }}
{{ end }}{{ end }}
{{- if .ResolvedQuestions }}
### Resolved Questions
{{ range .ResolvedQuestions }}- [{{ .Source }}] **Q:** {{ .Question }}
  **A:** {{ .Answer }}
{{ end }}{{ end }}
{{- if .Assumptions }}
### Assumptions (from completed stories)
{{ range .Assumptions }}- [{{ .Story }}] {{ .Assumption }}
{{ end }}{{ end }}
{{- if .Blockers }}
### Blockers
{{ range .Blockers }}- {{ . }}
{{ end }}{{ end }}
{{- if not .OpenQuestions }}{{ if not .Assumptions }}{{ if not .Blockers }}
*No open items.*
{{ end }}{{ end }}{{ end }}
`
