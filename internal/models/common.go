package models

// ResolvedQuestion holds a question and its answer after resolution.
type ResolvedQuestion struct {
	Question string `yaml:"question"`
	Answer   string `yaml:"answer"`
}
