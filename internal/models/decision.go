package models

// Decision statuses
const (
	DecisionStatusProposed   = "proposed"
	DecisionStatusAccepted   = "accepted"
	DecisionStatusSuperseded = "superseded"
	DecisionStatusDeprecated = "deprecated"
)

var ValidDecisionStatuses = []string{
	DecisionStatusProposed,
	DecisionStatusAccepted,
	DecisionStatusSuperseded,
	DecisionStatusDeprecated,
}

type Decision struct {
	ID          string   `yaml:"id"`
	Slug        string   `yaml:"slug"`
	Date        string   `yaml:"date"`
	Title       string   `yaml:"title"`
	Status      string   `yaml:"status"`
	ContextRefs []string `yaml:"context_refs,omitempty"`

	Body string `yaml:"-"`
}
