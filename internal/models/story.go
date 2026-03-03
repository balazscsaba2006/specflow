package models

import (
	"fmt"
	"time"
)

// Story statuses
const (
	StoryStatusDraft      = "draft"
	StoryStatusPlanned    = "planned"
	StoryStatusInProgress = "in_progress"
	StoryStatusVerifying  = "verifying"
	StoryStatusDone       = "done"
	StoryStatusBlocked    = "blocked"
	StoryStatusCancelled  = "cancelled"
)

var ValidStoryStatuses = []string{
	StoryStatusDraft,
	StoryStatusPlanned,
	StoryStatusInProgress,
	StoryStatusVerifying,
	StoryStatusDone,
	StoryStatusBlocked,
	StoryStatusCancelled,
}

// Story priorities
const (
	PriorityCritical = "critical"
	PriorityHigh     = "high"
	PriorityMedium   = "medium"
	PriorityLow      = "low"
)

var ValidPriorities = []string{
	PriorityCritical,
	PriorityHigh,
	PriorityMedium,
	PriorityLow,
}

// validTransitions defines allowed status transitions.
// Any status can transition to "blocked".
var validTransitions = map[string][]string{
	StoryStatusDraft:      {StoryStatusPlanned, StoryStatusBlocked, StoryStatusCancelled},
	StoryStatusPlanned:    {StoryStatusInProgress, StoryStatusBlocked, StoryStatusCancelled},
	StoryStatusInProgress: {StoryStatusVerifying, StoryStatusDone, StoryStatusBlocked, StoryStatusCancelled},
	StoryStatusVerifying:  {StoryStatusDone, StoryStatusInProgress, StoryStatusBlocked, StoryStatusCancelled},
	StoryStatusDone:       {StoryStatusCancelled},
	StoryStatusBlocked:    {StoryStatusDraft, StoryStatusPlanned, StoryStatusInProgress, StoryStatusCancelled},
	StoryStatusCancelled:  {StoryStatusDraft, StoryStatusPlanned},
}

type Story struct {
	ID            string    `yaml:"id"`
	Slug          string    `yaml:"slug"`
	Title         string    `yaml:"title"`
	Status        string    `yaml:"status"`
	Priority      string    `yaml:"priority"`
	Epic          string    `yaml:"epic,omitempty"`
	BlockedBy     []string  `yaml:"blocked_by,omitempty"`
	Labels        []string  `yaml:"labels,omitempty"`
	Acceptance    []string  `yaml:"acceptance,omitempty"`
	DocRefs       []string  `yaml:"doc_refs,omitempty"`
	Fidelity      string    `yaml:"fidelity,omitempty"`
	OpenQuestions     []string           `yaml:"open_questions,omitempty"`
	ResolvedQuestions []ResolvedQuestion `yaml:"resolved_questions,omitempty"`
	NonGoals          []string           `yaml:"non_goals,omitempty"`
	Assumptions       []string           `yaml:"assumptions,omitempty"`
	Created       time.Time `yaml:"created"`
	Updated       time.Time `yaml:"updated"`

	Body string `yaml:"-"`
}

// ValidateTransition checks if a status transition is allowed.
func ValidateTransition(from, to string) error {
	allowed, ok := validTransitions[from]
	if !ok {
		return fmt.Errorf("unknown status: %s", from)
	}
	for _, s := range allowed {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("invalid transition: %s → %s", from, to)
}
