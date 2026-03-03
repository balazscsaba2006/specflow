package models

import (
	"fmt"
	"time"
)

// Initiative statuses
const (
	InitiativeStatusActive    = "active"
	InitiativeStatusCompleted = "completed"
	InitiativeStatusOnHold    = "on_hold"
	InitiativeStatusArchived   = "archived"
	InitiativeStatusCancelled  = "cancelled"
)

var ValidInitiativeStatuses = []string{
	InitiativeStatusActive,
	InitiativeStatusCompleted,
	InitiativeStatusOnHold,
	InitiativeStatusArchived,
	InitiativeStatusCancelled,
}

// ValidateInitiativeStatus checks if the given status is a valid initiative status.
func ValidateInitiativeStatus(status string) error {
	for _, s := range ValidInitiativeStatuses {
		if s == status {
			return nil
		}
	}
	return fmt.Errorf("invalid initiative status %q", status)
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
	OpenQuestions     []string           `yaml:"open_questions,omitempty"`
	ResolvedQuestions []ResolvedQuestion `yaml:"resolved_questions,omitempty"`

	Body string `yaml:"-"`
}
