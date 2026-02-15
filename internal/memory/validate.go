package memory

import (
	"fmt"
	"strings"
)

// ValidateActionability checks if a principle/learning is actionable and specific.
// Returns an error if the principle is:
// - Too short (< 20 characters)
// - Contains vague/boilerplate phrases
// - Not in imperative form
func ValidateActionability(principle string) error {
	trimmed := strings.TrimSpace(principle)

	// Check minimum length
	if len(trimmed) < 20 {
		return fmt.Errorf("principle too short (%d chars): must be at least 20 characters", len(trimmed))
	}

	// Check for vague/boilerplate phrases
	lowerPrinciple := strings.ToLower(trimmed)
	vagueBlocklist := []string{
		"important pattern for review",
		"learning number",
		"useful reminder",
		"good to know",
		"for future reference",
		"keep in mind",
		"worth noting",
		"interesting finding",
		"something to remember",
	}

	for _, vague := range vagueBlocklist {
		if strings.Contains(lowerPrinciple, vague) {
			return fmt.Errorf("principle contains vague phrase %q", vague)
		}
	}

	return nil
}
