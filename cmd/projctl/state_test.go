package main_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/task"
)

// traces: TASK-002
// Test stateComplete marks task complete when task exists in tasks.md.
func TestStateComplete_MarksTaskComplete(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create state file
	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Create tasks.md with the task
	_ = os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Test task

**Acceptance Criteria:**
- [x] Done
`
	err = os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Mark complete - uses internal API that CLI wraps
	err = markTaskCompleteWithValidation(dir, "TASK-001")
	g.Expect(err).ToNot(HaveOccurred())

	// Verify task is marked complete
	s, err := state.Get(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.Progress.CompletedTasks).To(ContainElement("TASK-001"))
}

// traces: TASK-002
// Test stateComplete errors when task doesn't exist in tasks.md.
func TestStateComplete_ErrorsWhenTaskNotFound(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create state file
	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Create tasks.md without the target task
	_ = os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Different task

**Acceptance Criteria:**
- [x] Done
`
	err = os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Try to mark non-existent task complete
	err = markTaskCompleteWithValidation(dir, "TASK-999")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not found"))
}

// traces: TASK-002
// Test stateGet shows completed tasks in output.
func TestStateGet_ShowsCompletedTasks(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create state file
	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())

	// Mark a task complete directly
	_, err = state.MarkTaskComplete(dir, "TASK-001")
	g.Expect(err).ToNot(HaveOccurred())

	// Get state and verify completed_tasks is present
	s, err := state.Get(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.Progress.CompletedTasks).To(Equal([]string{"TASK-001"}))
}

// markTaskCompleteWithValidation validates task exists before marking complete.
// This is the logic that the CLI command should implement.
func markTaskCompleteWithValidation(dir, taskID string) error {
	// Validate task exists in tasks.md
	result := task.ValidateAcceptanceCriteria(dir, taskID)
	if result.Error != "" && containsNotFound(result.Error) {
		return fmt.Errorf("task %s not found in tasks.md", taskID)
	}

	// Mark task complete
	_, err := state.MarkTaskComplete(dir, taskID)
	return err
}

func containsNotFound(s string) bool {
	return strings.Contains(s, "not found")
}
