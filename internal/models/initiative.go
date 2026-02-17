package models

import "time"

// Initiative statuses
const (
	InitiativeStatusActive    = "active"
	InitiativeStatusCompleted = "completed"
	InitiativeStatusOnHold    = "on_hold"
	InitiativeStatusArchived  = "archived"
)

var ValidInitiativeStatuses = []string{
	InitiativeStatusActive,
	InitiativeStatusCompleted,
	InitiativeStatusOnHold,
	InitiativeStatusArchived,
}

type Initiative struct {
	ID              string    `yaml:"id"`
	Slug            string    `yaml:"slug"`
	Title           string    `yaml:"title"`
	Status          string    `yaml:"status"`
	Created         time.Time `yaml:"created"`
	Updated         time.Time `yaml:"updated"`
	Epics           []string  `yaml:"epics,omitempty"`
	Goal            string    `yaml:"goal"`
	SuccessCriteria []string  `yaml:"success_criteria,omitempty"`
	OpenQuestions   []string  `yaml:"open_questions,omitempty"`

	Body string `yaml:"-"`
}
