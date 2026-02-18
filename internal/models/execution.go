package models

import "time"

// Execution statuses
const (
	ExecutionStatusStarted   = "started"
	ExecutionStatusCompleted = "completed"
	ExecutionStatusFailed    = "failed"
	ExecutionStatusPaused    = "paused"
)

var ValidExecutionStatuses = []string{
	ExecutionStatusStarted,
	ExecutionStatusCompleted,
	ExecutionStatusFailed,
	ExecutionStatusPaused,
}

type FileChange struct {
	Path   string `yaml:"path"`
	Action string `yaml:"action"`
}

type Execution struct {
	ID           string       `yaml:"id"`
	Story        string       `yaml:"story"`
	Plan         string       `yaml:"plan,omitempty"`
	Status       string       `yaml:"status"`
	StartedAt    time.Time    `yaml:"started_at"`
	CompletedAt  *time.Time   `yaml:"completed_at,omitempty"`
	GitRefBefore string       `yaml:"git_ref_before"`
	GitRefAfter  string       `yaml:"git_ref_after,omitempty"`
	FilesChanged  []FileChange `yaml:"files_changed,omitempty"`
	HandoverNotes string       `yaml:"handover_notes,omitempty"`
}
