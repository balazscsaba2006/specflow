package models

import "time"

// Document types
const (
	DocTypePRD        = "prd"
	DocTypeTechSpec   = "tech-spec"
	DocTypeAPISpec    = "api-spec"
	DocTypeDesignSpec = "design-spec"
	DocTypeADR        = "adr"
	DocTypeOnePager   = "one-pager"
)

var ValidDocTypes = []string{
	DocTypePRD,
	DocTypeTechSpec,
	DocTypeAPISpec,
	DocTypeDesignSpec,
	DocTypeADR,
	DocTypeOnePager,
}

// Document statuses
const (
	DocStatusDraft      = "draft"
	DocStatusReview     = "review"
	DocStatusApproved   = "approved"
	DocStatusSuperseded = "superseded"
)

var ValidDocStatuses = []string{
	DocStatusDraft,
	DocStatusReview,
	DocStatusApproved,
	DocStatusSuperseded,
}

type Document struct {
	ID            string    `yaml:"id"`
	Slug          string    `yaml:"slug"`
	Type          string    `yaml:"type"`
	Title         string    `yaml:"title"`
	Status        string    `yaml:"status"`
	Epic          string    `yaml:"epic,omitempty"`
	Created       time.Time `yaml:"created"`
	Updated       time.Time `yaml:"updated"`
	OpenQuestions     []string           `yaml:"open_questions,omitempty"`
	ResolvedQuestions []ResolvedQuestion `yaml:"resolved_questions,omitempty"`

	Body string `yaml:"-"`
}
