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

// ValidGapSizes are valid gap_size values.
var ValidGapSizes = []string{
	"small",
	"medium",
	"large",
}

// RequiredGapAnalysisFields are required fields in gap_analysis section.
var RequiredGapAnalysisFields = []string{
	"total_key_questions",
	"questions_answered",
	"coverage_percent",
	"gap_size",
	"question_count",
	"sources",
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

// GapAnalysis holds gap assessment metadata.
type GapAnalysis struct {
	TotalKeyQuestions  int      `toml:"total_key_questions"`
	QuestionsAnswered  int      `toml:"questions_answered"`
	CoveragePercent    float64  `toml:"coverage_percent"`
	GapSize            string   `toml:"gap_size"`
	QuestionCount      int      `toml:"question_count"`
	Sources            []string `toml:"sources"`
	UnansweredCritical []string `toml:"unanswered_critical"`
}

// ContextSection holds the [context] section.
type ContextSection struct {
	Phase       string        `toml:"phase"`
	Subphase    string        `toml:"subphase"`
	Iteration   int           `toml:"iteration"`
	Task        string        `toml:"task"`
	Awaiting    string        `toml:"awaiting"`
	Role        string        `toml:"role"`
	GapAnalysis *GapAnalysis  `toml:"gap_analysis"`
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

// ParseContent parses yield content into a YieldFile struct.
func ParseContent(content string) (*YieldFile, error) {
	var y YieldFile
	_, err := toml.Decode(content, &y)
	if err != nil {
		return nil, fmt.Errorf("TOML parse error: %w", err)
	}
	return &y, nil
}

// ValidateContent checks yield content for validity.
func ValidateContent(content string) (ValidationResult, error) {
	var result ValidationResult

	// Decode to map to check field presence
	var raw map[string]any
	if _, err := toml.Decode(content, &raw); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("TOML parse error: %v", err))
		return result, nil
	}

	// Decode to struct for type checking
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

	// Validate gap_analysis section if present
	if y.Context.GapAnalysis != nil {
		validateGapAnalysisWithMap(y.Context.GapAnalysis, raw, &result)
	}

	result.Valid = len(result.Errors) == 0
	return result, nil
}

func validateGapAnalysisWithMap(ga *GapAnalysis, raw map[string]any, result *ValidationResult) {
	// Extract gap_analysis map
	contextMap, ok := raw["context"].(map[string]any)
	if !ok {
		return
	}
	gaMap, ok := contextMap["gap_analysis"].(map[string]any)
	if !ok {
		return
	}

	// Check required fields
	for _, field := range RequiredGapAnalysisFields {
		if _, exists := gaMap[field]; !exists {
			result.Errors = append(result.Errors, fmt.Sprintf("missing required field: [context.gap_analysis].%s", field))
		}
	}

	// Validate gap_size enum
	if ga.GapSize != "" {
		valid := false
		for _, validSize := range ValidGapSizes {
			if ga.GapSize == validSize {
				valid = true
				break
			}
		}
		if !valid {
			result.Errors = append(result.Errors,
				fmt.Sprintf("invalid gap_size: must be one of: %s", strings.Join(ValidGapSizes, ", ")))
		}
	}

	// Validate coverage_percent range
	if ga.CoveragePercent < 0 || ga.CoveragePercent > 100 {
		result.Errors = append(result.Errors, "coverage_percent must be between 0 and 100")
	}

	// Validate question_count is non-negative
	if ga.QuestionCount < 0 {
		result.Errors = append(result.Errors, "question_count must be non-negative")
	}
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
