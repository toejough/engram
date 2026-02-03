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
	tasksPath := filepath.Join(dir, "tasks.md")

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

// ACItem represents a single acceptance criterion item.
type ACItem struct {
	Text     string
	Complete bool
}

// ACResult holds the result of acceptance criteria validation.
type ACResult struct {
	Complete    int
	Incomplete  int
	AllComplete bool
	Items       []ACItem
	Error       string
}

// ValidateAcceptanceCriteria checks acceptance criteria completion status.
func ValidateAcceptanceCriteria(dir, taskID string) ACResult {
	tasksPath := filepath.Join(dir, "tasks.md")

	content, err := os.ReadFile(tasksPath)
	if err != nil {
		return ACResult{Error: "could not read tasks.md: " + err.Error()}
	}

	taskContent := extractTask(string(content), taskID)
	if taskContent == "" {
		return ACResult{Error: "task not found: " + taskID}
	}

	// Parse acceptance criteria
	items := parseAcceptanceCriteria(taskContent)

	complete := 0
	incomplete := 0
	for _, item := range items {
		if item.Complete {
			complete++
		} else {
			incomplete++
		}
	}

	return ACResult{
		Complete:    complete,
		Incomplete:  incomplete,
		AllComplete: incomplete == 0,
		Items:       items,
	}
}

// parseAcceptanceCriteria extracts AC items from task content.
func parseAcceptanceCriteria(content string) []ACItem {
	// Find AC section
	acPattern := regexp.MustCompile(`(?m)^\*\*Acceptance Criteria:\*\*\s*$`)
	loc := acPattern.FindStringIndex(content)
	if loc == nil {
		return nil
	}

	// Get content after AC header
	rest := content[loc[1]:]

	// Find next section (any **Field:**)
	nextPattern := regexp.MustCompile(`(?m)^\*\*[^*]+:\*\*`)
	nextLoc := nextPattern.FindStringIndex(rest)

	var acContent string
	if nextLoc == nil {
		acContent = rest
	} else {
		acContent = rest[:nextLoc[0]]
	}

	// Parse checkboxes
	var items []ACItem
	checkboxPattern := regexp.MustCompile(`(?m)^-\s+\[([ x])\]\s+(.+)$`)
	matches := checkboxPattern.FindAllStringSubmatch(acContent, -1)

	for _, m := range matches {
		items = append(items, ACItem{
			Text:     strings.TrimSpace(m[2]),
			Complete: m[1] == "x",
		})
	}

	return items
}

// PreconditionChecker is an interface for checking preconditions.
type PreconditionChecker interface {
	// Can be used for custom precondition checking logic
}

// DefaultPreconditionChecker is the default implementation.
type DefaultPreconditionChecker struct{}

// TaskCompleteOpts holds options for task completion validation.
type TaskCompleteOpts struct {
	Force bool
}

// ValidateTaskComplete checks if a task can be marked complete.
func ValidateTaskComplete(dir, taskID string, checker PreconditionChecker) error {
	return ValidateTaskCompleteWithOpts(dir, taskID, checker, TaskCompleteOpts{})
}

// ValidateTaskCompleteWithOpts checks if a task can be marked complete with options.
func ValidateTaskCompleteWithOpts(dir, taskID string, checker PreconditionChecker, opts TaskCompleteOpts) error {
	// If force is enabled, bypass validation
	if opts.Force {
		return nil
	}

	result := ValidateAcceptanceCriteria(dir, taskID)
	if result.Error != "" {
		return &ValidationError{Message: result.Error}
	}

	if !result.AllComplete {
		return &ValidationError{Message: buildIncompleteACError(result.Items)}
	}

	return nil
}

// buildIncompleteACError constructs an error message for incomplete acceptance criteria.
func buildIncompleteACError(items []ACItem) string {
	var incompleteItems []string
	for _, item := range items {
		if !item.Complete {
			incompleteItems = append(incompleteItems, item.Text)
		}
	}

	errMsg := "acceptance criteria unmet. Incomplete items:\n"
	for _, item := range incompleteItems {
		errMsg += "- " + item + "\n"
	}

	return errMsg
}

// ValidationError is returned when validation fails.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
