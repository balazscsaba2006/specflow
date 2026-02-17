package models

import "time"

type LogEntry struct {
	Timestamp    time.Time `json:"ts"`
	Type         string    `json:"type"`
	Entity       string    `json:"entity"`
	From         string    `json:"from,omitempty"`
	To           string    `json:"to,omitempty"`
	Epic         string    `json:"epic,omitempty"`
	Story        string    `json:"story,omitempty"`
	GitRef       string    `json:"git_ref,omitempty"`
	FilesChanged int       `json:"files_changed,omitempty"`
	Result       string    `json:"result,omitempty"`
	Critical     int       `json:"critical,omitempty"`
	Major        int       `json:"major,omitempty"`
	Minor        int       `json:"minor,omitempty"`
}

// Log event types
const (
	LogStoryStatusChanged   = "story.status_changed"
	LogExecutionStarted     = "execution.started"
	LogExecutionCompleted   = "execution.completed"
	LogVerificationSaved    = "verification.saved"
	LogDocCreated           = "doc.created"
	LogDocUpdated           = "doc.updated"
	LogDecisionRecorded     = "decision.recorded"
	LogInitiativeCreated    = "initiative.created"
	LogEpicCreated          = "epic.created"
	LogStoryCreated         = "story.created"
	LogPlanSaved            = "plan.saved"
	LogQuestionResolved     = "question.resolved"
)
