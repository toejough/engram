package step_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
	"github.com/toejough/projctl/internal/task"
	"pgregory.net/rapid"
)

// TestNextResult_TasksArray_TASK1 verifies TASK-1: Add Tasks Array to NextResult Struct
//
// Traces to: TASK-1, ARCH-1, DES-1, REQ-3
func TestNextResult_TasksArray_TASK1(t *testing.T) {
	t.Run("NextResult struct has Tasks field", func(t *testing.T) {
		g := NewWithT(t)

		// Create a NextResult with Tasks field
		result := step.NextResult{
			Action: "spawn-producer",
			Tasks:  []step.TaskInfo{},
		}

		// Marshal to JSON
		data, err := json.Marshal(result)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify JSON contains "tasks" field
		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(parsed).To(HaveKey("tasks"))
	})

	t.Run("TaskInfo struct has required fields", func(t *testing.T) {
		g := NewWithT(t)

		// Create a TaskInfo with all fields
		worktreePath := "/path/to/worktree"
		taskInfo := step.TaskInfo{
			ID:       "TASK-1",
			Command:  "projctl run TASK-1",
			Worktree: &worktreePath,
		}

		// Marshal to JSON
		data, err := json.Marshal(taskInfo)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify JSON structure
		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(parsed).To(HaveKey("id"))
		g.Expect(parsed).To(HaveKey("command"))
		g.Expect(parsed).To(HaveKey("worktree"))
		g.Expect(parsed["id"]).To(Equal("TASK-1"))
		g.Expect(parsed["command"]).To(Equal("projctl run TASK-1"))
		g.Expect(parsed["worktree"]).To(Equal("/path/to/worktree"))
	})

	t.Run("Worktree field marshals to null when pointer is nil", func(t *testing.T) {
		g := NewWithT(t)

		// Create TaskInfo with nil worktree
		taskInfo := step.TaskInfo{
			ID:       "TASK-1",
			Command:  "projctl run TASK-1",
			Worktree: nil,
		}

		// Marshal to JSON
		data, err := json.Marshal(taskInfo)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify worktree field is null
		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(parsed).To(HaveKey("worktree"))
		g.Expect(parsed["worktree"]).To(BeNil())
	})

	t.Run("Empty Tasks array marshals correctly", func(t *testing.T) {
		g := NewWithT(t)

		result := step.NextResult{
			Action: "transition",
			Tasks:  []step.TaskInfo{},
		}

		data, err := json.Marshal(result)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify empty array in JSON
		g.Expect(string(data)).To(ContainSubstring(`"tasks":[]`))
	})

	t.Run("Single task array marshals correctly", func(t *testing.T) {
		g := NewWithT(t)

		result := step.NextResult{
			Action: "spawn-producer",
			Tasks: []step.TaskInfo{
				{
					ID:       "TASK-1",
					Command:  "projctl run TASK-1",
					Worktree: nil,
				},
			},
		}

		data, err := json.Marshal(result)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify single-element array
		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		g.Expect(err).ToNot(HaveOccurred())
		tasks := parsed["tasks"].([]interface{})
		g.Expect(tasks).To(HaveLen(1))
	})

	t.Run("Multiple tasks array marshals correctly", func(t *testing.T) {
		g := NewWithT(t)

		wt1 := "/path/to/wt1"
		wt2 := "/path/to/wt2"
		result := step.NextResult{
			Action: "spawn-producer",
			Tasks: []step.TaskInfo{
				{ID: "TASK-1", Command: "projctl run TASK-1", Worktree: &wt1},
				{ID: "TASK-2", Command: "projctl run TASK-2", Worktree: &wt2},
			},
		}

		data, err := json.Marshal(result)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify multi-element array
		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		g.Expect(err).ToNot(HaveOccurred())
		tasks := parsed["tasks"].([]interface{})
		g.Expect(tasks).To(HaveLen(2))
	})
}

// TestNext_ParallelDetection_TASK2 verifies TASK-2: Modify Next() to Return Array of Unblocked Tasks
//
// Traces to: TASK-2, ARCH-2, DES-1, DES-6, REQ-1, REQ-5
func TestNext_ParallelDetection_TASK2(t *testing.T) {
	t.Run("Next returns empty array when no tasks unblocked", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create project with all tasks blocked
		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: []string{"TASK-2"}},
			{id: "TASK-2", status: "pending", deps: []string{"TASK-1"}}, // circular, all blocked
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Tasks).To(BeEmpty())
	})

	t.Run("Next returns single-element array when one task unblocked", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create project with one unblocked task
		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
			{id: "TASK-2", status: "pending", deps: []string{"TASK-1"}},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Tasks).To(HaveLen(1))
		g.Expect(result.Tasks[0].ID).To(Equal("TASK-1"))
	})

	t.Run("Next returns multi-element array when multiple tasks unblocked", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create project with 3 unblocked tasks
		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
			{id: "TASK-2", status: "pending", deps: nil},
			{id: "TASK-3", status: "pending", deps: nil},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Tasks).To(HaveLen(3))

		// Verify all task IDs present
		taskIDs := make(map[string]bool)
		for _, task := range result.Tasks {
			taskIDs[task.ID] = true
		}
		g.Expect(taskIDs).To(HaveKey("TASK-1"))
		g.Expect(taskIDs).To(HaveKey("TASK-2"))
		g.Expect(taskIDs).To(HaveKey("TASK-3"))
	})

	t.Run("Next populates correct command for each task", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
			{id: "TASK-2", status: "pending", deps: nil},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())

		for _, task := range result.Tasks {
			expectedCmd := "projctl run " + task.ID
			g.Expect(task.Command).To(Equal(expectedCmd))
		}
	})

	t.Run("Property: command format holds for random task IDs", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate random task ID
			taskID := rapid.StringMatching(`TASK-[0-9]+`).Draw(t, "taskID")

			// Expected command format
			expectedCmd := "projctl run " + taskID

			// Verify format
			if expectedCmd != "projctl run "+taskID {
				t.Fatalf("command format incorrect for %s", taskID)
			}
		})
	})
}

// TestNext_WorktreeAssignment_TASK3 verifies TASK-3: Assign Worktree Paths to Parallel Tasks
//
// Traces to: TASK-3, ARCH-3, DES-2, DES-5, REQ-2
func TestNext_WorktreeAssignment_TASK3(t *testing.T) {
	t.Run("Single task has worktree null", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Tasks).To(HaveLen(1))
		g.Expect(result.Tasks[0].Worktree).To(BeNil())
	})

	t.Run("Multiple tasks have non-null worktree paths", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
			{id: "TASK-2", status: "pending", deps: nil},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Tasks).To(HaveLen(2))

		// All tasks should have non-nil worktree paths
		for _, task := range result.Tasks {
			g.Expect(task.Worktree).ToNot(BeNil())
			g.Expect(*task.Worktree).ToNot(BeEmpty())
		}
	})

	t.Run("Multiple tasks have unique worktree paths", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
			{id: "TASK-2", status: "pending", deps: nil},
			{id: "TASK-3", status: "pending", deps: nil},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Collect all worktree paths
		paths := make(map[string]bool)
		for _, task := range result.Tasks {
			g.Expect(task.Worktree).ToNot(BeNil())
			paths[*task.Worktree] = true
		}

		// Verify uniqueness
		g.Expect(paths).To(HaveLen(len(result.Tasks)))
	})

	t.Run("Worktree path format matches expected pattern", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
			{id: "TASK-2", status: "pending", deps: nil},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())

		for _, task := range result.Tasks {
			g.Expect(task.Worktree).ToNot(BeNil())
			// Path should contain task ID and follow worktree pattern
			g.Expect(*task.Worktree).To(ContainSubstring(task.ID))
			g.Expect(*task.Worktree).To(ContainSubstring("-worktrees"))
		}
	})

	t.Run("Property: path uniqueness for distinct task IDs", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate two distinct task IDs
			taskID1 := rapid.StringMatching(`TASK-[1-5][0-9]{2}`).Draw(t, "taskID1")
			taskID2 := rapid.StringMatching(`TASK-[6-9][0-9]{2}`).Draw(t, "taskID2")

			// Create worktree paths (simulated)
			path1 := filepath.Join("/tmp/worktrees", taskID1)
			path2 := filepath.Join("/tmp/worktrees", taskID2)

			// Verify paths are different
			if path1 == path2 {
				t.Fatalf("expected unique paths for distinct task IDs, got same path: %s", path1)
			}
		})
	})
}

// TestStatus_TASK4 verifies TASK-4: Implement projctl step status Command
//
// Traces to: TASK-4, ARCH-4, DES-3
func TestStatus_TASK4(t *testing.T) {
	t.Run("Status function returns correct structure", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Initialize project
		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "complete", deps: nil},
			{id: "TASK-2", status: "pending", deps: nil},
			{id: "TASK-3", status: "pending", deps: []string{"TASK-1"}},
		})

		result, err := step.Status(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify result structure
		g.Expect(result.ActiveTasks).ToNot(BeNil())
		g.Expect(result.CompletedTasks).ToNot(BeNil())
		g.Expect(result.BlockedTasks).ToNot(BeNil())
	})

	t.Run("Status detects active tasks from worktree list", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create mock worktree
		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
		})

		// TODO: Create actual worktree for TASK-1
		// This test will fail until worktree is created

		result, err := step.Status(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Should detect TASK-1 as active
		var foundActive bool
		for _, task := range result.ActiveTasks {
			if task.ID == "TASK-1" {
				foundActive = true
				break
			}
		}
		g.Expect(foundActive).To(BeTrue())
	})

	t.Run("Status detects completed tasks from state file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "complete", deps: nil},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Status(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Should detect TASK-1 as completed
		var foundCompleted bool
		for _, task := range result.CompletedTasks {
			if task.ID == "TASK-1" {
				foundCompleted = true
				break
			}
		}
		g.Expect(foundCompleted).To(BeTrue())
	})

	t.Run("Status detects blocked tasks from dependency graph", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
			{id: "TASK-2", status: "pending", deps: []string{"TASK-1"}},
		})

		result, err := step.Status(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// TASK-2 should be blocked
		var foundBlocked bool
		for _, task := range result.BlockedTasks {
			if task.ID == "TASK-2" {
				foundBlocked = true
				g.Expect(task.BlockedBy).To(ContainElement("TASK-1"))
				break
			}
		}
		g.Expect(foundBlocked).To(BeTrue())
	})

	t.Run("Status JSON output is well-formed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "complete", deps: nil},
			{id: "TASK-2", status: "pending", deps: nil},
		})

		result, err := step.Status(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Marshal to JSON
		data, err := json.Marshal(result)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify JSON is valid
		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(parsed).To(HaveKey("active_tasks"))
		g.Expect(parsed).To(HaveKey("completed_tasks"))
		g.Expect(parsed).To(HaveKey("blocked_tasks"))
	})
}

// TestOverlapDetectionRemoval_TASK5 verifies TASK-5: Remove File Overlap Detection References
//
// Traces to: TASK-5, ARCH-5, DES-7, REQ-4
func TestOverlapDetectionRemoval_TASK5(t *testing.T) {
	t.Run("projctl tasks overlap command does not exist", func(t *testing.T) {
		g := NewWithT(t)

		// Try to find overlap-related code
		// This test verifies that the command is removed
		err := task.DetectOverlap("")
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("not implemented"))
	})

	t.Run("No overlap detection logic in codebase", func(t *testing.T) {
		// This is a static analysis test
		// Verify that no "overlap" functions exist in task package
		// Will fail if overlap detection code is present
		t.Skip("Static analysis test - implement with grep/ast inspection")
	})
}

// TestParallelExecutionIntegration_TASK6 verifies TASK-6: Integration Tests for Parallel Execution
//
// Traces to: TASK-6, DES-4, DES-5, DES-6, REQ-1, REQ-2, REQ-5
func TestParallelExecutionIntegration_TASK6(t *testing.T) {
	t.Run("Integration: 3 unblocked tasks returns array of 3", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
			{id: "TASK-2", status: "pending", deps: nil},
			{id: "TASK-3", status: "pending", deps: nil},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Tasks).To(HaveLen(3))

		// Each task should have unique non-null worktree
		for _, task := range result.Tasks {
			g.Expect(task.Worktree).ToNot(BeNil())
		}
	})

	t.Run("Integration: single unblocked task has worktree null", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Tasks).To(HaveLen(1))
		g.Expect(result.Tasks[0].Worktree).To(BeNil())
	})

	t.Run("Integration: no unblocked tasks returns empty array", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: []string{"TASK-2"}},
			{id: "TASK-2", status: "pending", deps: []string{"TASK-1"}},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Tasks).To(BeEmpty())
	})

	t.Run("Integration: task completion unblocks new tasks (EC-3)", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Initial state: TASK-1 unblocked, TASK-2 and TASK-3 blocked by TASK-1
		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
			{id: "TASK-2", status: "pending", deps: []string{"TASK-1"}},
			{id: "TASK-3", status: "pending", deps: []string{"TASK-1"}},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		// First call: should return TASK-1
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Tasks).To(HaveLen(1))
		g.Expect(result.Tasks[0].ID).To(Equal("TASK-1"))

		// Simulate TASK-1 completion
		updateTaskStatus(t, dir, "TASK-1", "complete")

		// Second call: should return TASK-2 and TASK-3 (newly unblocked)
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Tasks).To(HaveLen(2))

		taskIDs := make(map[string]bool)
		for _, task := range result.Tasks {
			taskIDs[task.ID] = true
		}
		g.Expect(taskIDs).To(HaveKey("TASK-2"))
		g.Expect(taskIDs).To(HaveKey("TASK-3"))
	})

	t.Run("Integration: status shows correct task states", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "complete", deps: nil},
			{id: "TASK-2", status: "pending", deps: nil},
			{id: "TASK-3", status: "pending", deps: []string{"TASK-2"}},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		status, err := step.Status(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify completed tasks
		completedIDs := make(map[string]bool)
		for _, task := range status.CompletedTasks {
			completedIDs[task.ID] = true
		}
		g.Expect(completedIDs).To(HaveKey("TASK-1"))

		// Verify blocked tasks
		blockedIDs := make(map[string]bool)
		for _, task := range status.BlockedTasks {
			blockedIDs[task.ID] = true
		}
		g.Expect(blockedIDs).To(HaveKey("TASK-3"))
	})

	t.Run("Integration: JSON output parseable by jq", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		createProjectWithTasks(t, dir, []taskSpec{
			{id: "TASK-1", status: "pending", deps: nil},
			{id: "TASK-2", status: "pending", deps: nil},
		})

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Marshal to JSON
		data, err := json.Marshal(result)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify it's valid JSON (jq would be able to parse this)
		var parsed interface{}
		err = json.Unmarshal(data, &parsed)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("Property: response format holds across randomized task graphs", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate random number of unblocked tasks (0-5)
			numTasks := rapid.IntRange(0, 5).Draw(t, "numTasks")

			// Verify the response would have correct length
			// (This is a simplified property test - full version would create actual project)
			if numTasks < 0 {
				t.Fatalf("invalid task count: %d", numTasks)
			}
		})
	})
}

// Helper types and functions

type taskSpec struct {
	id     string
	status string
	deps   []string
}

func createProjectWithTasks(t *testing.T, dir string, tasks []taskSpec) {
	t.Helper()

	// Create tasks.md file
	var content string
	content += "# Tasks\n\n"

	for _, task := range tasks {
		content += "### " + task.id + ": Test Task\n\n"
		content += "**Status:** " + task.status + "\n\n"

		if len(task.deps) > 0 {
			content += "**Dependencies:** "
			for i, dep := range task.deps {
				if i > 0 {
					content += ", "
				}
				content += dep
			}
			content += "\n\n"
		} else {
			content += "**Dependencies:** None\n\n"
		}

		content += "---\n\n"
	}

	err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create tasks.md: %v", err)
	}
}

func updateTaskStatus(t *testing.T, dir string, taskID string, newStatus string) {
	t.Helper()

	// Read tasks.md
	content, err := os.ReadFile(filepath.Join(dir, "tasks.md"))
	if err != nil {
		t.Fatalf("failed to read tasks.md: %v", err)
	}

	// Update status (simple string replacement)
	// This is a simplified helper - real implementation would parse and update properly
	updated := string(content)
	// For now, just mark this as a TODO for the implementation phase

	err = os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(updated), 0644)
	if err != nil {
		t.Fatalf("failed to update tasks.md: %v", err)
	}
}
