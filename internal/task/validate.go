// Package task provides task validation and management.
package task

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidationResult holds the result of task validation.
type ValidationResult struct {
	Valid   bool
	Error   string
	Warning string
}

// ValidateOpts holds options for task validation.
type ValidateOpts struct {
	ManualVisualVerified bool // Bypass visual evidence requirement with manual verification
}

// Validate checks if a task meets validation requirements.
// For UI tasks, visual evidence is required.
func Validate(dir, taskID string) ValidationResult {
	return ValidateWithOpts(dir, taskID, ValidateOpts{})
}

// ValidateWithOpts checks if a task meets validation requirements with options.
func ValidateWithOpts(dir, taskID string, opts ValidateOpts) ValidationResult {
	tasksPath := filepath.Join(dir, "docs", "tasks.md")

	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return ValidationResult{Valid: false, Error: "could not read tasks.md: " + err.Error()}
	}

	taskContent := extractTask(string(content), taskID)
	if taskContent == "" {
		return ValidationResult{Valid: false, Error: "task not found: " + taskID}
	}

	isUI := parseUIFlag(taskContent)
	hasVisualEvidence := parseVisualEvidence(taskContent)

	if isUI && !hasVisualEvidence {
		if opts.ManualVisualVerified {
			return ValidationResult{
				Valid:   true,
				Warning: "visual evidence bypassed via manual verification flag",
			}
		}

		return ValidationResult{Valid: false, Error: "UI task requires visual evidence (add **Visual evidence:** field)"}
	}

	return ValidationResult{Valid: true}
}

// extractTask extracts the content for a specific task from tasks.md.
func extractTask(content, taskID string) string {
	// Find the task section by ID
	pattern := regexp.MustCompile(`(?m)^###\s+` + regexp.QuoteMeta(taskID) + `:.*$`)
	loc := pattern.FindStringIndex(content)
	if loc == nil {
		return ""
	}

	start := loc[0]

	// Find the next task section or end of file
	nextPattern := regexp.MustCompile(`(?m)^###\s+TASK-\d+:`)
	rest := content[loc[1]:]
	nextLoc := nextPattern.FindStringIndex(rest)

	var end int
	if nextLoc == nil {
		end = len(content)
	} else {
		end = loc[1] + nextLoc[0]
	}

	return content[start:end]
}

// parseUIFlag extracts the UI flag value from task content.
func parseUIFlag(content string) bool {
	pattern := regexp.MustCompile(`(?m)^\*\*UI:\*\*\s*(true|false)`)
	match := pattern.FindStringSubmatch(content)
	if match == nil {
		return false
	}

	return strings.ToLower(match[1]) == "true"
}

// parseVisualEvidence checks if visual evidence is present.
func parseVisualEvidence(content string) bool {
	pattern := regexp.MustCompile(`(?m)^\*\*Visual evidence:\*\*\s*\S+`)
	return pattern.MatchString(content)
}
