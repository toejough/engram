package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Exported variables.
var (
	ErrLearnInvalidType = errors.New(
		"learn: type must be 'feedback' or 'fact'",
	)
	ErrLearnMissingField = errors.New("learn: missing required field")
)

// ValidateLearnMessage validates a learn message's JSON text.
// For type="feedback": requires situation, behavior, impact, action.
// For type="fact": requires situation, subject, predicate, object.
func ValidateLearnMessage(text string) error {
	var parsed map[string]string

	jsonErr := json.Unmarshal([]byte(text), &parsed)
	if jsonErr != nil {
		return fmt.Errorf("learn: invalid JSON: %w", jsonErr)
	}

	switch parsed["type"] {
	case "feedback":
		return requireFields(
			parsed,
			"feedback",
			"situation", "behavior", "impact", "action",
		)
	case "fact":
		return requireFields(
			parsed,
			"fact",
			"situation", "subject", "predicate", "object",
		)
	default:
		return fmt.Errorf(
			"%w, got %q", ErrLearnInvalidType, parsed["type"],
		)
	}
}

// isLearnMessage returns true if text looks like a learn message JSON payload.
func isLearnMessage(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "{") &&
		strings.Contains(text, `"type":`)
}

// requireFields checks that all named fields are non-empty in the parsed map.
func requireFields(
	parsed map[string]string,
	kind string,
	fields ...string,
) error {
	for _, field := range fields {
		if parsed[field] == "" {
			return fmt.Errorf(
				"%w: %s %s", ErrLearnMissingField, kind, field,
			)
		}
	}

	return nil
}
