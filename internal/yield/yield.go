// Package yield provides functionality for validating yield TOML files.
package yield

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// ValidProducerTypes are valid producer yield types.
var ValidProducerTypes = []string{
	"complete",
	"need-user-input",
	"need-context",
	"need-decision",
	"need-agent",
	"blocked",
	"error",
}

// ValidQATypes are valid QA yield types.
var ValidQATypes = []string{
	"approved",
	"improvement-request",
	"escalate-phase",
	"escalate-user",
}

// AllValidTypes returns all valid yield types.
func AllValidTypes() []string {
	all := make([]string, 0, len(ValidProducerTypes)+len(ValidQATypes))
	all = append(all, ValidProducerTypes...)
	all = append(all, ValidQATypes...)
	return all
}

// ResumableTypes are yield types that require a context section.
var ResumableTypes = []string{
	"need-user-input",
	"need-context",
	"need-decision",
	"need-agent",
	"blocked",
	"improvement-request",
	"escalate-phase",
	"escalate-user",
}

// YieldSection holds the [yield] section.
type YieldSection struct {
	Type      string    `toml:"type"`
	Timestamp time.Time `toml:"timestamp"`
}

// ContextSection holds the [context] section.
type ContextSection struct {
	Phase     string `toml:"phase"`
	Subphase  string `toml:"subphase"`
	Iteration int    `toml:"iteration"`
	Task      string `toml:"task"`
	Awaiting  string `toml:"awaiting"`
	Role      string `toml:"role"`
}

// YieldFile represents a parsed yield file.
type YieldFile struct {
	Yield   YieldSection   `toml:"yield"`
	Payload map[string]any `toml:"payload"`
	Context ContextSection `toml:"context"`
}

// ValidationResult holds the result of validating a yield file.
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// Validate checks a yield file for validity.
func Validate(path string) (ValidationResult, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("failed to read file: %w", err)
	}

	return ValidateContent(string(content))
}

// ValidateContent checks yield content for validity.
func ValidateContent(content string) (ValidationResult, error) {
	var result ValidationResult

	var y YieldFile
	_, err := toml.Decode(content, &y)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("TOML parse error: %v", err))
		return result, nil
	}

	// Check required field: type
	if y.Yield.Type == "" {
		result.Errors = append(result.Errors, "missing required field: [yield].type")
	} else if !isValidType(y.Yield.Type) {
		result.Errors = append(result.Errors,
			fmt.Sprintf("invalid yield type %q, valid types: %s",
				y.Yield.Type, strings.Join(AllValidTypes(), ", ")))
	}

	// Check required field: timestamp
	if y.Yield.Timestamp.IsZero() {
		result.Errors = append(result.Errors, "missing required field: [yield].timestamp")
	}

	// Check context section for resumable types
	if isResumableType(y.Yield.Type) {
		if y.Context.Phase == "" && y.Context.Subphase == "" &&
			y.Context.Task == "" && y.Context.Awaiting == "" {
			result.Errors = append(result.Errors,
				fmt.Sprintf("yield type %q requires [context] section", y.Yield.Type))
		}
	}

	result.Valid = len(result.Errors) == 0
	return result, nil
}

func isValidType(t string) bool {
	for _, valid := range AllValidTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

func isResumableType(t string) bool {
	for _, rt := range ResumableTypes {
		if t == rt {
			return true
		}
	}
	return false
}
