package models

import "time"

// Plan statuses
const (
	PlanStatusDraft     = "draft"
	PlanStatusApproved  = "approved"
	PlanStatusExecuting = "executing"
	PlanStatusVerified  = "verified"
)

var ValidPlanStatuses = []string{
	PlanStatusDraft,
	PlanStatusApproved,
	PlanStatusExecuting,
	PlanStatusVerified,
}

type Plan struct {
	ID             string    `yaml:"id"`
	Story          string    `yaml:"story"`
	Status         string    `yaml:"status"`
	Created        time.Time `yaml:"created"`
	GitRefBaseline string    `yaml:"git_ref_baseline,omitempty"`
	EstimatedFiles int       `yaml:"estimated_files,omitempty"`

	Body string `yaml:"-"`
}
