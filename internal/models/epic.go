package models

import "time"

// Epic statuses
const (
	EpicStatusDraft     = "draft"
	EpicStatusActive    = "active"
	EpicStatusCompleted = "completed"
	EpicStatusOnHold    = "on_hold"
	EpicStatusArchived  = "archived"
)

var ValidEpicStatuses = []string{
	EpicStatusDraft,
	EpicStatusActive,
	EpicStatusCompleted,
	EpicStatusOnHold,
	EpicStatusArchived,
}

type Phase struct {
	Label   string   `yaml:"label"`
	Stories []string `yaml:"stories"`
}

type Epic struct {
	ID            string    `yaml:"id"`
	Slug          string    `yaml:"slug"`
	Title         string    `yaml:"title"`
	Status        string    `yaml:"status"`
	Initiative    string    `yaml:"initiative,omitempty"`
	Created       time.Time `yaml:"created"`
	Updated       time.Time `yaml:"updated"`
	Phases        []Phase   `yaml:"phases,omitempty"`
	Fidelity      string    `yaml:"fidelity,omitempty"`
	OpenQuestions     []string           `yaml:"open_questions,omitempty"`
	ResolvedQuestions []ResolvedQuestion `yaml:"resolved_questions,omitempty"`
	NonGoals          []string           `yaml:"non_goals,omitempty"`
	Decisions         []string           `yaml:"decisions,omitempty"`

	Body string `yaml:"-"`
}
