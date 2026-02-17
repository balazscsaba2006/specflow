package models

import (
	"fmt"
	"regexp"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidateSlug checks that a slug is lowercase alphanumeric with hyphens.
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("invalid slug %q: must be lowercase alphanumeric with hyphens (e.g., my-feature-name)", slug)
	}
	return nil
}
