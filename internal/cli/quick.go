// Package cli — quick subcommand: writes a fleeting note file.
package cli

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	errSlugEmpty   = errors.New("quick: slug is required")
	errSlugInvalid = errors.New("quick: slug must match [a-z0-9-]+")

	slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// validateSlug returns nil if slug is non-empty kebab-case lowercase.
func validateSlug(slug string) error {
	if slug == "" {
		return errSlugEmpty
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("%w: got %q", errSlugInvalid, slug)
	}
	return nil
}
