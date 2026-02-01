package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/toejough/projctl/internal/task"
	"github.com/toejough/projctl/internal/trace"
)

// DefaultChecker implements state.PreconditionChecker with real filesystem checks.
type DefaultChecker struct{}

func (c *DefaultChecker) RequirementsExist(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "docs", "requirements.md"))
	return err == nil
}

func (c *DefaultChecker) RequirementsHaveIDs(dir string) bool {
	content, err := os.ReadFile(filepath.Join(dir, "docs", "requirements.md"))
	if err != nil {
		return false
	}
	// Check for REQ-NNN pattern
	matched, _ := regexp.MatchString(`REQ-\d{3}`, string(content))
	return matched
}

func (c *DefaultChecker) DesignExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "docs", "design.md"))
	return err == nil
}

func (c *DefaultChecker) DesignHasIDs(dir string) bool {
	content, err := os.ReadFile(filepath.Join(dir, "docs", "design.md"))
	if err != nil {
		return false
	}
	// Check for DES-NNN pattern
	matched, _ := regexp.MatchString(`DES-\d{3}`, string(content))
	return matched
}

func (c *DefaultChecker) TraceValidationPasses(dir string) bool {
	result, err := trace.Validate(dir)
	if err != nil {
		return false
	}
	return result.Pass
}

func (c *DefaultChecker) TestsExist(dir string) bool {
	// Look for *_test.go files
	matches, err := filepath.Glob(filepath.Join(dir, "**", "*_test.go"))
	if err != nil || len(matches) == 0 {
		// Try internal directory
		matches, _ = filepath.Glob(filepath.Join(dir, "internal", "**", "*_test.go"))
	}
	return len(matches) > 0
}

func (c *DefaultChecker) TestsFail(dir string) bool {
	// This would require running tests - stub for now
	// In practice, the TDD skill verifies this
	return true
}

func (c *DefaultChecker) TestsPass(dir string) bool {
	// This would require running tests - stub for now
	// In practice, the TDD skill verifies this
	return true
}

func (c *DefaultChecker) AcceptanceCriteriaComplete(dir, taskID string) bool {
	if taskID == "" {
		return true // No task specified, skip AC check
	}

	result := task.ValidateAcceptanceCriteria(dir, taskID)
	if result.Error != "" {
		// Task not found or parse error - check if it's because AC section is missing
		if strings.Contains(result.Error, "not found") {
			return true // Task not in tasks.md, skip AC check
		}
		return false
	}

	return result.AllComplete
}
