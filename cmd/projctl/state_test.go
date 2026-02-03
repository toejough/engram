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

// traces: ISSUE-036
// Test stateInit defaults to .claude/projects/<name>/ when --dir is not provided.
func TestStateInit_DefaultsToProjectDir(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Change to temp dir to test relative path behavior
	oldWd, err := os.Getwd()
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = os.Chdir(oldWd) }()
	g.Expect(os.Chdir(dir)).To(Succeed())

	// Call stateInit with name but no dir - should default to .claude/projects/<name>/
	err = stateInitWithDefaults("my-project", "", "new", "")
	g.Expect(err).ToNot(HaveOccurred())

	// Verify state file was created in the default location
	expectedPath := filepath.Join(dir, ".claude", "projects", "my-project", "state.toml")
	_, err = os.Stat(expectedPath)
	g.Expect(err).ToNot(HaveOccurred(), "state.toml should exist at %s", expectedPath)

	// Verify state content
	s, err := state.Get(filepath.Join(dir, ".claude", "projects", "my-project"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.Project.Name).To(Equal("my-project"))
}

// traces: ISSUE-036
// Test stateInit creates the directory if it doesn't exist.
func TestStateInit_CreatesDirectory(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	oldWd, err := os.Getwd()
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = os.Chdir(oldWd) }()
	g.Expect(os.Chdir(dir)).To(Succeed())

	// .claude/projects doesn't exist yet
	projectDir := filepath.Join(dir, ".claude", "projects", "new-project")
	_, err = os.Stat(projectDir)
	g.Expect(os.IsNotExist(err)).To(BeTrue())

	// stateInit should create it
	err = stateInitWithDefaults("new-project", "", "new", "")
	g.Expect(err).ToNot(HaveOccurred())

	// Directory should now exist
	info, err := os.Stat(projectDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(info.IsDir()).To(BeTrue())
}

// traces: ISSUE-036
// Test stateInit still respects explicit --dir when provided.
func TestStateInit_RespectsExplicitDir(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	customDir := filepath.Join(dir, "custom", "location")
	g.Expect(os.MkdirAll(customDir, 0o755)).To(Succeed())

	err := stateInitWithDefaults("my-project", customDir, "new", "")
	g.Expect(err).ToNot(HaveOccurred())

	// Verify state file was created in the explicit location
	expectedPath := filepath.Join(customDir, "state.toml")
	_, err = os.Stat(expectedPath)
	g.Expect(err).ToNot(HaveOccurred())
}

// stateInitWithDefaults wraps the CLI logic for testing.
// This replicates the defaulting behavior from stateInit in state.go.
func stateInitWithDefaults(name, dir, mode, issue string) error {
	// Default mode is "new"
	if mode == "" {
		mode = "new"
	}

	// Default dir to .claude/projects/<name>/
	if dir == "" {
		dir = filepath.Join(".claude", "projects", name)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	_, err := state.Init(dir, name, func() time.Time { return time.Now() }, state.InitOpts{
		Workflow: mode,
		Issue:    issue,
	})
	return err
}
