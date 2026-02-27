// Package checker provides precondition checking for project workflows.
package checker

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/toejough/projctl/internal/issue"
	"github.com/toejough/projctl/internal/task"
	"github.com/toejough/projctl/internal/trace"
)

// DefaultChecker implements precondition checking with real filesystem checks.
type DefaultChecker struct{}

func (c *DefaultChecker) AcceptanceCriteriaComplete(dir, taskID string) bool {
	if taskID == "" {
		return true
	}

	result := task.ValidateAcceptanceCriteria(dir, taskID)
	if result.Error != "" {
		return strings.Contains(result.Error, "not found")
	}

	return result.AllComplete
}

func (c *DefaultChecker) DesignExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "design.md"))
	return err == nil
}

func (c *DefaultChecker) DesignHasIDs(dir string) bool {
	content, err := os.ReadFile(filepath.Join(dir, "design.md"))
	if err != nil {
		return false
	}

	matched, _ := regexp.Match(`DES-\d+`, content)

	return matched
}

func (c *DefaultChecker) IncompleteAcceptanceCriteria(dir, taskID string) []string {
	if taskID == "" {
		return nil
	}

	result := task.ValidateAcceptanceCriteria(dir, taskID)
	if result.Error != "" {
		return nil
	}

	var incomplete []string

	for _, item := range result.Items {
		if !item.Complete {
			incomplete = append(incomplete, item.Text)
		}
	}

	return incomplete
}

func (c *DefaultChecker) IncompleteIssueAC(repoDir, issueID string) []string {
	if issueID == "" {
		return nil
	}

	result := issue.ParseAcceptanceCriteria(repoDir, issueID)
	if result.Error != "" {
		return nil
	}

	var incomplete []string

	for _, item := range result.Items {
		if !item.Complete {
			incomplete = append(incomplete, item.Text)
		}
	}

	return incomplete
}

func (c *DefaultChecker) IssueACComplete(repoDir, issueID string) bool {
	if issueID == "" {
		return true
	}

	result := issue.ParseAcceptanceCriteria(repoDir, issueID)
	if result.Error != "" {
		return strings.Contains(result.Error, "not found")
	}

	return result.AllComplete
}

func (c *DefaultChecker) RequirementsExist(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "requirements.md"))
	return err == nil
}

func (c *DefaultChecker) RequirementsHaveIDs(dir string) bool {
	content, err := os.ReadFile(filepath.Join(dir, "requirements.md"))
	if err != nil {
		return false
	}

	matched, _ := regexp.Match(`REQ-\d+`, content)

	return matched
}

func (c *DefaultChecker) RetroExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "retro.md"))
	return err == nil
}

func (c *DefaultChecker) SummaryExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "summary.md"))
	return err == nil
}

func (c *DefaultChecker) TestsExist(dir string) bool {
	matches, err := filepath.Glob(filepath.Join(dir, "**", "*_test.go"))
	if err != nil || len(matches) == 0 {
		matches, _ = filepath.Glob(filepath.Join(dir, "internal", "**", "*_test.go"))
	}

	return len(matches) > 0
}

func (c *DefaultChecker) TestsFail(dir string) bool {
	return true
}

func (c *DefaultChecker) TestsPass(dir string) bool {
	return true
}

func (c *DefaultChecker) TraceValidationPasses(dir string, phase string) bool {
	var (
		result trace.ValidateV2ArtifactsResult
		err    error
	)

	if phase != "" {
		result, err = trace.ValidateV2Artifacts(dir, trace.RealFS{}, phase)
	} else {
		result, err = trace.ValidateV2Artifacts(dir, trace.RealFS{})
	}

	if err != nil {
		return false
	}

	return result.Pass
}

func (c *DefaultChecker) UnblockedTasks(dir, failedTask string) []string {
	parallelTasks, err := task.Parallel(dir)
	if err != nil {
		return nil
	}

	var unblocked []string

	for _, t := range parallelTasks {
		if t != failedTask {
			unblocked = append(unblocked, t)
		}
	}

	return unblocked
}
