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
			Workflow: "adopt",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Workflow).To(Equal("adopt"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Workflow).To(Equal("adopt"))
	})

	t.Run("accepts issue option", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-042",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Issue).To(Equal("ISSUE-042"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Issue).To(Equal("ISSUE-042"))
	})

	t.Run("accepts both workflow and issue options", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		s, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "task",
			Issue:    "ISSUE-099",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Workflow).To(Equal("task"))
		g.Expect(s.Project.Issue).To(Equal("ISSUE-099"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Workflow).To(Equal("task"))
		g.Expect(loaded.Project.Issue).To(Equal("ISSUE-099"))
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

		s, err := state.Set(dir, state.SetOpts{Issue: "ISSUE-042"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Issue).To(Equal("ISSUE-042"))
		g.Expect(s.Project.Phase).To(Equal("init")) // No transition

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Issue).To(Equal("ISSUE-042"))
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

		s, err := state.Set(dir, state.SetOpts{Workflow: "adopt"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Workflow).To(Equal("adopt"))

		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Workflow).To(Equal("adopt"))
	})

	t.Run("sets multiple fields at once", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Set(dir, state.SetOpts{
			Issue:    "ISSUE-099",
			Task:     "TASK-001",
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Issue).To(Equal("ISSUE-099"))
		g.Expect(s.Progress.CurrentTask).To(Equal("TASK-001"))
		g.Expect(s.Project.Workflow).To(Equal("task"))
	})

	t.Run("ignores empty string values", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-001",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Set only task, issue should remain unchanged
		s, err := state.Set(dir, state.SetOpts{Task: "TASK-001"})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Issue).To(Equal("ISSUE-001")) // Unchanged
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
			Iteration:        2,
			MaxIterations:    3,
			ProducerComplete: true,
			QAVerdict:        "improvement-request",
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

// traces: ISSUE-037
// Test that retro-complete requires retro.md to exist.
func TestArtifactPreconditions(t *testing.T) {
	// Use adopt workflow which has a simpler path to main flow ending
	adoptPathToRetro := []string{
		"adopt-explore", "adopt-infer-tests", "adopt-infer-arch",
		"adopt-infer-design", "adopt-infer-reqs", "adopt-escalations",
		"adopt-documentation", "alignment", "alignment-complete", "retro",
	}

	t.Run("retro-complete fails without retro.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Fast-forward to retro phase using adopt workflow
		for _, phase := range adoptPathToRetro {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", phase)
		}

		// Now try to transition to retro-complete without retro.md
		checker := &mockArtifactChecker{retroExists: false}
		_, err = state.TransitionWithChecker(dir, "retro-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("retro.md"))
	})

	t.Run("retro-complete succeeds with retro.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Fast-forward to retro phase using adopt workflow
		for _, phase := range adoptPathToRetro {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", phase)
		}

		// Now try to transition to retro-complete with retro.md existing
		checker := &mockArtifactChecker{retroExists: true}
		_, err = state.TransitionWithChecker(dir, "retro-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("summary-complete fails without summary.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Fast-forward to summary phase using adopt workflow
		adoptPathToSummary := append(adoptPathToRetro, "retro-complete", "summary")
		for _, phase := range adoptPathToSummary {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", phase)
		}

		// Now try to transition to summary-complete without summary.md
		checker := &mockArtifactChecker{summaryExists: false}
		_, err = state.TransitionWithChecker(dir, "summary-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("summary.md"))
	})

	t.Run("summary-complete succeeds with summary.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Fast-forward to summary phase using adopt workflow
		adoptPathToSummary := append(adoptPathToRetro, "retro-complete", "summary")
		for _, phase := range adoptPathToSummary {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", phase)
		}

		// Now try to transition to summary-complete with summary.md existing
		checker := &mockArtifactChecker{summaryExists: true}
		_, err = state.TransitionWithChecker(dir, "summary-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).ToNot(HaveOccurred())
	})
}

// mockArtifactChecker implements PreconditionChecker for testing artifact preconditions.
type mockArtifactChecker struct {
	retroExists   bool
	summaryExists bool
}

func (m *mockArtifactChecker) RequirementsExist(dir string) bool              { return true }
func (m *mockArtifactChecker) RequirementsHaveIDs(dir string) bool            { return true }
func (m *mockArtifactChecker) DesignExists(dir string) bool                   { return true }
func (m *mockArtifactChecker) DesignHasIDs(dir string) bool                   { return true }
func (m *mockArtifactChecker) TraceValidationPasses(dir string) bool          { return true }
func (m *mockArtifactChecker) TestsExist(dir string) bool                     { return true }
func (m *mockArtifactChecker) TestsFail(dir string) bool                      { return true }
func (m *mockArtifactChecker) TestsPass(dir string) bool                      { return true }
func (m *mockArtifactChecker) AcceptanceCriteriaComplete(dir, taskID string) bool { return true }
func (m *mockArtifactChecker) IncompleteAcceptanceCriteria(dir, taskID string) []string { return nil }
func (m *mockArtifactChecker) UnblockedTasks(dir string, failedTask string) []string { return nil }
func (m *mockArtifactChecker) RetroExists(dir string) bool                    { return m.retroExists }
func (m *mockArtifactChecker) SummaryExists(dir string) bool                  { return m.summaryExists }

func TestYieldTracking(t *testing.T) {
	t.Run("tracks pending yield", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.SetYield(dir, &state.YieldState{
			Pending:     true,
			Type:        "need-user-input",
			Agent:       "pm",
			ContextFile: ".claude/agents/pm-state.toml",
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Yield).ToNot(BeNil())
		g.Expect(s.Yield.Pending).To(BeTrue())
		g.Expect(s.Yield.Type).To(Equal("need-user-input"))
		g.Expect(s.Yield.Agent).To(Equal("pm"))
		g.Expect(s.Yield.ContextFile).To(Equal(".claude/agents/pm-state.toml"))

		// Verify persistence
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Yield).ToNot(BeNil())
		g.Expect(loaded.Yield.Type).To(Equal("need-user-input"))
	})

	t.Run("clears yield by setting to nil", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetYield(dir, &state.YieldState{
			Pending: true,
			Type:    "need-context",
			Agent:   "arch",
		})
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.SetYield(dir, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Yield).To(BeNil())

		// Verify persistence
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Yield).To(BeNil())
	})

	t.Run("ClearYield clears pending yield", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetYield(dir, &state.YieldState{
			Pending: true,
			Type:    "blocked",
			Agent:   "tdd-green",
		})
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.ClearYield(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Yield).To(BeNil())
	})
}
