package state_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"pgregory.net/rapid"
)

func fixedTime() time.Time {
	return time.Date(2026, 1, 27, 12, 0, 0, 0, time.UTC)
}

func nowFunc() func() time.Time {
	return func() time.Time { return fixedTime() }
}

func TestInit(t *testing.T) {
	t.Run("creates state file with correct initial state", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Name).To(Equal("test-project"))
		g.Expect(s.Project.Phase).To(Equal("init"))
		g.Expect(s.Project.Created).To(Equal(fixedTime()))
		g.Expect(s.History).To(HaveLen(1))
		g.Expect(s.History[0].Phase).To(Equal("init"))
		g.Expect(s.History[0].Timestamp).To(Equal(fixedTime()))

		// Verify file exists on disk
		_, err = os.Stat(filepath.Join(dir, state.StateFile))
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("errors if state file already exists", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create initial state
		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Try again — should fail
		_, err = state.Init(dir, "test-project", nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("already exists"))
	})

	t.Run("errors if directory does not exist", func(t *testing.T) {
		g := NewWithT(t)

		_, err := state.Init("/nonexistent/path/that/does/not/exist", "test-project", nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	})

	t.Run("state file is valid TOML readable by Get", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		original, err := state.Init(dir, "roundtrip-test", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Name).To(Equal(original.Project.Name))
		g.Expect(loaded.Project.Phase).To(Equal(original.Project.Phase))
		g.Expect(loaded.History).To(HaveLen(1))
	})

	t.Run("defaults workflow to new when no options provided", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Workflow).To(Equal("new"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Workflow).To(Equal("new"))
	})

	t.Run("accepts workflow option", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Workflow).To(Equal("scoped"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Workflow).To(Equal("scoped"))
	})

	t.Run("accepts issue option", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-42",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Issue).To(Equal("ISSUE-42"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Issue).To(Equal("ISSUE-42"))
	})

	t.Run("accepts both workflow and issue options", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "scoped",
			Issue:    "ISSUE-99",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Workflow).To(Equal("scoped"))
		g.Expect(s.Project.Issue).To(Equal("ISSUE-99"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Workflow).To(Equal("scoped"))
		g.Expect(loaded.Project.Issue).To(Equal("ISSUE-99"))
	})

	t.Run("accepts repo_dir option", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			RepoDir: "/path/to/repo",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.RepoDir).To(Equal("/path/to/repo"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.RepoDir).To(Equal("/path/to/repo"))
	})

	t.Run("repo_dir defaults to empty when not provided", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.RepoDir).To(Equal(""))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.RepoDir).To(Equal(""))
	})
}

func TestInitProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		name := rapid.StringMatching(`[a-z][a-z0-9-]{1,30}`).Draw(rt, "name")
		dir := t.TempDir()

		s, err := state.Init(dir, name, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Name).To(Equal(name))
		g.Expect(s.Project.Phase).To(Equal("init"))

		// Roundtrip: read back should match
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Name).To(Equal(name))
	})
}

func TestSet(t *testing.T) {
	t.Run("sets issue without transitioning", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Set(dir, state.SetOpts{Issue: "ISSUE-42"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Issue).To(Equal("ISSUE-42"))
		g.Expect(s.Project.Phase).To(Equal("init")) // No transition

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Issue).To(Equal("ISSUE-42"))
	})

	t.Run("sets task without transitioning", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Set(dir, state.SetOpts{Task: "TASK-007"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Progress.CurrentTask).To(Equal("TASK-007"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Progress.CurrentTask).To(Equal("TASK-007"))
	})

	t.Run("sets workflow without transitioning", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Set(dir, state.SetOpts{Workflow: "scoped"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Workflow).To(Equal("scoped"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Workflow).To(Equal("scoped"))
	})

	t.Run("sets multiple fields at once", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Set(dir, state.SetOpts{
			Issue:    "ISSUE-99",
			Task:     "TASK-001",
			Workflow: "scoped",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Issue).To(Equal("ISSUE-99"))
		g.Expect(s.Progress.CurrentTask).To(Equal("TASK-001"))
		g.Expect(s.Project.Workflow).To(Equal("scoped"))
	})

	t.Run("ignores empty string values", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-1",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Set only task, issue should remain unchanged
		s, err := state.Set(dir, state.SetOpts{Task: "TASK-001"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Issue).To(Equal("ISSUE-1")) // Unchanged
		g.Expect(s.Progress.CurrentTask).To(Equal("TASK-001"))
	})
}

func TestGet(t *testing.T) {
	t.Run("errors if state file does not exist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Get(dir)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("failed to read"))
	})

	t.Run("reads valid state file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "read-test", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Name).To(Equal("read-test"))
		g.Expect(s.Project.Phase).To(Equal("init"))
	})
}

func TestPairLoopTracking(t *testing.T) {
	t.Run("tracks pair loop state for phases", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.SetPair(dir, "pm", state.PairState{
			Iteration:          2,
			MaxIterations:      3,
			ProducerComplete:   true,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "REQ-003 acceptance criteria are not measurable",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs).To(HaveKey("pm"))
		g.Expect(s.Pairs["pm"].Iteration).To(Equal(2))
		g.Expect(s.Pairs["pm"].QAVerdict).To(Equal("improvement-request"))

		// Verify persistence
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Pairs).To(HaveKey("pm"))
		g.Expect(loaded.Pairs["pm"].Iteration).To(Equal(2))
	})

	t.Run("tracks pair loop state for tasks", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.SetPair(dir, "TASK-007", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "approved",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs).To(HaveKey("TASK-007"))
		g.Expect(s.Pairs["TASK-007"].QAVerdict).To(Equal("approved"))
	})

	t.Run("updates existing pair loop state", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:        1,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "improvement-request",
		})
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.SetPair(dir, "pm", state.PairState{
			Iteration:        2,
			MaxIterations:    3,
			ProducerComplete: false,
			QAVerdict:        "",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["pm"].Iteration).To(Equal(2))
		g.Expect(s.Pairs["pm"].ProducerComplete).To(BeFalse())
	})

	t.Run("ClearPair removes pair loop state", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration: 2,
		})
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.ClearPair(dir, "pm")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs).ToNot(HaveKey("pm"))
	})
}

func TestCompletedTasks(t *testing.T) {
	t.Run("Progress has CompletedTasks field", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Progress.CompletedTasks).To(BeEmpty())

		// Verify persistence - empty slice persists correctly
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Progress.CompletedTasks).To(BeEmpty())
	})

	t.Run("MarkTaskComplete appends to slice and persists", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.MarkTaskComplete(dir, "TASK-001")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Progress.CompletedTasks).To(Equal([]string{"TASK-001"}))

		// Verify persistence
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Progress.CompletedTasks).To(Equal([]string{"TASK-001"}))
	})

	t.Run("MarkTaskComplete appends multiple tasks in order", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.MarkTaskComplete(dir, "TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.MarkTaskComplete(dir, "TASK-003")
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.MarkTaskComplete(dir, "TASK-002")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Progress.CompletedTasks).To(Equal([]string{"TASK-001", "TASK-003", "TASK-002"}))
	})

	t.Run("MarkTaskComplete is idempotent - same task not added twice", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.MarkTaskComplete(dir, "TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.MarkTaskComplete(dir, "TASK-001")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Progress.CompletedTasks).To(Equal([]string{"TASK-001"}))
	})

	t.Run("MarkTaskComplete errors if state file does not exist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.MarkTaskComplete(dir, "TASK-001")
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("failed to read"))
	})

	t.Run("IsTaskComplete returns true for completed task", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.MarkTaskComplete(dir, "TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		complete, err := state.IsTaskComplete(dir, "TASK-001")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(complete).To(BeTrue())
	})

	t.Run("IsTaskComplete returns false for incomplete task", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		complete, err := state.IsTaskComplete(dir, "TASK-001")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(complete).To(BeFalse())
	})

	t.Run("IsTaskComplete errors if state file does not exist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.IsTaskComplete(dir, "TASK-001")
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("failed to read"))
	})
}

func TestCompletedTasksProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Generate a set of task IDs
		taskCount := rapid.IntRange(1, 10).Draw(rt, "taskCount")
		taskIDs := make([]string, taskCount)
		for i := 0; i < taskCount; i++ {
			taskIDs[i] = rapid.StringMatching(`TASK-[0-9]{3}`).Draw(rt, "taskID")
		}

		// Mark all tasks complete
		for _, taskID := range taskIDs {
			_, err := state.MarkTaskComplete(dir, taskID)
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Verify all tasks are complete
		for _, taskID := range taskIDs {
			complete, err := state.IsTaskComplete(dir, taskID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(complete).To(BeTrue(), "task %s should be complete", taskID)
		}

		// Verify a random non-completed task is not complete
		nonExistent := "TASK-999"
		found := false
		for _, taskID := range taskIDs {
			if taskID == nonExistent {
				found = true
				break
			}
		}
		if !found {
			complete, err := state.IsTaskComplete(dir, nonExistent)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(complete).To(BeFalse())
		}
	})
}

// Precondition tests removed during TOML state machine migration.
// Old adopt-* phase paths no longer exist. Precondition logic is still in place
// and tested via the new flat state names in transition_test.go.

func TestWorktreeTracking(t *testing.T) {
	t.Run("State has Worktrees map", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Worktrees).To(BeNil()) // nil by default, not empty map

		// Verify persistence - nil map persists correctly
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Worktrees).To(BeNil())
	})

	t.Run("SetWorktree adds a worktree and persists", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		created := fixedTime()
		s, err := state.SetWorktree(dir, "TASK-007", state.WorktreeState{
			Path:    "/path/to/worktree/TASK-007",
			Branch:  "task/TASK-007",
			Created: created,
			Status:  "active",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Worktrees).To(HaveKey("TASK-007"))
		g.Expect(s.Worktrees["TASK-007"].Path).To(Equal("/path/to/worktree/TASK-007"))
		g.Expect(s.Worktrees["TASK-007"].Branch).To(Equal("task/TASK-007"))
		g.Expect(s.Worktrees["TASK-007"].Created).To(Equal(created))
		g.Expect(s.Worktrees["TASK-007"].Status).To(Equal("active"))

		// Verify persistence
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Worktrees).To(HaveKey("TASK-007"))
		g.Expect(loaded.Worktrees["TASK-007"].Path).To(Equal("/path/to/worktree/TASK-007"))
		g.Expect(loaded.Worktrees["TASK-007"].Branch).To(Equal("task/TASK-007"))
		g.Expect(loaded.Worktrees["TASK-007"].Status).To(Equal("active"))
	})

	t.Run("SetWorktree updates existing worktree", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		created := fixedTime()
		_, err = state.SetWorktree(dir, "TASK-007", state.WorktreeState{
			Path:    "/path/to/worktree/TASK-007",
			Branch:  "task/TASK-007",
			Created: created,
			Status:  "active",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Update to merged
		s, err := state.SetWorktree(dir, "TASK-007", state.WorktreeState{
			Path:    "/path/to/worktree/TASK-007",
			Branch:  "task/TASK-007",
			Created: created,
			Status:  "merged",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Worktrees["TASK-007"].Status).To(Equal("merged"))

		// Verify persistence
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Worktrees["TASK-007"].Status).To(Equal("merged"))
	})

	t.Run("ClearWorktree removes worktree", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetWorktree(dir, "TASK-007", state.WorktreeState{
			Path:    "/path/to/worktree/TASK-007",
			Branch:  "task/TASK-007",
			Created: fixedTime(),
			Status:  "active",
		})
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.ClearWorktree(dir, "TASK-007")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Worktrees).ToNot(HaveKey("TASK-007"))

		// Verify persistence
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Worktrees).ToNot(HaveKey("TASK-007"))
	})

	t.Run("multiple worktrees can coexist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		created := fixedTime()
		_, err = state.SetWorktree(dir, "TASK-007", state.WorktreeState{
			Path:    "/path/to/worktree/TASK-007",
			Branch:  "task/TASK-007",
			Created: created,
			Status:  "active",
		})
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.SetWorktree(dir, "TASK-008", state.WorktreeState{
			Path:    "/path/to/worktree/TASK-008",
			Branch:  "task/TASK-008",
			Created: created,
			Status:  "active",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Worktrees).To(HaveLen(2))
		g.Expect(s.Worktrees).To(HaveKey("TASK-007"))
		g.Expect(s.Worktrees).To(HaveKey("TASK-008"))

		// Verify persistence
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Worktrees).To(HaveLen(2))
	})

	t.Run("WorktreeState status tracks active/merged/failed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		created := fixedTime()

		// Test all status values
		for _, status := range []string{"active", "merged", "failed"} {
			taskID := "TASK-" + status
			s, err := state.SetWorktree(dir, taskID, state.WorktreeState{
				Path:    "/path/to/worktree/" + taskID,
				Branch:  "task/" + taskID,
				Created: created,
				Status:  status,
			})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(s.Worktrees[taskID].Status).To(Equal(status))
		}

		// Verify persistence
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Worktrees["TASK-active"].Status).To(Equal("active"))
		g.Expect(loaded.Worktrees["TASK-merged"].Status).To(Equal("merged"))
		g.Expect(loaded.Worktrees["TASK-failed"].Status).To(Equal("failed"))
	})
}

func TestWorktreeTrackingProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Generate random worktree data
		taskID := rapid.StringMatching(`TASK-[0-9]{3}`).Draw(rt, "taskID")
		path := rapid.StringMatching(`/[a-z]+/[a-z]+/`+taskID).Draw(rt, "path")
		branch := rapid.StringMatching(`task/`+taskID).Draw(rt, "branch")
		status := rapid.SampledFrom([]string{"active", "merged", "failed"}).Draw(rt, "status")

		// Set worktree
		s, err := state.SetWorktree(dir, taskID, state.WorktreeState{
			Path:    path,
			Branch:  branch,
			Created: fixedTime(),
			Status:  status,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Worktrees).To(HaveKey(taskID))
		g.Expect(s.Worktrees[taskID].Path).To(Equal(path))
		g.Expect(s.Worktrees[taskID].Branch).To(Equal(branch))
		g.Expect(s.Worktrees[taskID].Status).To(Equal(status))

		// Roundtrip: read back should match
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Worktrees[taskID].Path).To(Equal(path))
		g.Expect(loaded.Worktrees[taskID].Branch).To(Equal(branch))
		g.Expect(loaded.Worktrees[taskID].Status).To(Equal(status))
	})
}
