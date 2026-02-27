package memory

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

// ParseResultFile parses a result protocol TOML file from raw bytes.
// It uses BurntSushi/toml for unmarshaling and performs strict schema
// validation that fails fast on the first error.
//
// Required fields:
//   - status.result: The result status (e.g., "success", "failure", "error")
//   - status.timestamp: RFC3339 timestamp when result was created
//
// Returns SchemaValidationError when required fields are missing or empty.
// Returns a wrapped parse error when TOML syntax is invalid.
func ParseResultFile(data []byte) (*ResultFile, error) {
	var resultFile ResultFile

	err := toml.Unmarshal(data, &resultFile)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Validate required fields - fail fast on first error
	err = validateRequired("status.result", resultFile.Status.Result, "non-empty string")
	if err != nil {
		return nil, err
	}

	err = validateRequired("status.timestamp", resultFile.Status.Timestamp, "non-empty string (RFC3339 format)")
	if err != nil {
		return nil, err
	}

	return &resultFile, nil
}

// validateRequired creates a SchemaValidationError if the field is empty.
// Returns nil if the field is non-empty.
func validateRequired(field, value, expectedDesc string) error {
	if value == "" {
		return &SchemaValidationError{
			Field:    field,
			Expected: expectedDesc,
			Actual:   "empty or missing",
			Line:     0,
		}
	}

	return nil
}
