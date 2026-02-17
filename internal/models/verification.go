package models

import "time"

// Verification results
const (
	VerificationPass    = "pass"
	VerificationFail    = "fail"
	VerificationPartial = "partial"
)

// Finding severities
const (
	SeverityCritical = "critical"
	SeverityMajor    = "major"
	SeverityMinor    = "minor"
)

// Finding categories
const (
	CategoryMissing     = "missing"
	CategoryBug         = "bug"
	CategoryPerformance = "performance"
	CategorySecurity    = "security"
	CategoryClarity     = "clarity"
	CategoryQuality     = "quality"
)

type Finding struct {
	Severity    string `yaml:"severity"`
	Category    string `yaml:"category"`
	File        string `yaml:"file,omitempty"`
	Description string `yaml:"description"`
	Suggestion  string `yaml:"suggestion,omitempty"`
}

type AcceptanceCheck struct {
	Criteria string `yaml:"criteria"`
	Met      bool   `yaml:"met"`
}

type VerificationStats struct {
	Critical int `yaml:"critical"`
	Major    int `yaml:"major"`
	Minor    int `yaml:"minor"`
}

type Verification struct {
	ID              string            `yaml:"id"`
	Execution       string            `yaml:"execution"`
	Story           string            `yaml:"story"`
	Result          string            `yaml:"result"`
	Created         time.Time         `yaml:"created"`
	Stats           VerificationStats `yaml:"stats"`
	Findings        []Finding         `yaml:"findings,omitempty"`
	AcceptanceCheck []AcceptanceCheck `yaml:"acceptance_check,omitempty"`
	Assumptions     []string          `yaml:"assumptions,omitempty"`

	Body string `yaml:"-"`
}
