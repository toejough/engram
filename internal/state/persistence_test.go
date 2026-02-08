package state_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"pgregory.net/rapid"
)

// TestStatePersistence verifies TASK-11 acceptance criteria:
// State persistence with TOML roundtrip, validation, and error handling
//
// Traces to: REQ-105-002, ARCH-105-002, ARCH-105-003, TASK-4, TASK-11, ISSUE-105

func TestStateSchemaFields(t *testing.T) {
	t.Run("State contains all required fields", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Initialize state
		s, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Workflow: "new",
			Issue:    "ISSUE-42",
			RepoDir:  "/repo/path",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Verify Project fields
		g.Expect(s.Project.Name).To(Equal("test-project"))
		g.Expect(s.Project.Created).To(Equal(fixedTime()))
		g.Expect(s.Project.Phase).To(Equal("init"))
		g.Expect(s.Project.Workflow).To(Equal("new"))
		g.Expect(s.Project.Issue).To(Equal("ISSUE-42"))
		g.Expect(s.Project.RepoDir).To(Equal("/repo/path"))

		// Verify Progress fields exist (zero-valued initially)
		g.Expect(s.Progress.CurrentTask).To(Equal(""))
		g.Expect(s.Progress.CurrentSubphase).To(Equal(""))
		g.Expect(s.Progress.TasksComplete).To(Equal(0))
		g.Expect(s.Progress.TasksTotal).To(Equal(0))
		g.Expect(s.Progress.TasksEscalated).To(Equal(0))
		g.Expect(s.Progress.CompletedTasks).To(BeEmpty())

		// Verify Conflicts fields exist (zero-valued initially)
		g.Expect(s.Conflicts.Open).To(Equal(0))
		g.Expect(s.Conflicts.BlockingTasks).To(BeNil())

		// Verify Meta fields exist (zero-valued initially)
		g.Expect(s.Meta.CorrectionsSinceLastAudit).To(Equal(0))

		// Verify History exists with init entry
		g.Expect(s.History).To(HaveLen(1))
		g.Expect(s.History[0].Phase).To(Equal("init"))
		g.Expect(s.History[0].Timestamp).To(Equal(fixedTime()))

		// Verify Error is nil initially
		g.Expect(s.Error).To(BeNil())

		// Verify Pairs is nil initially
		g.Expect(s.Pairs).To(BeNil())

		// Verify Worktrees is nil initially
		g.Expect(s.Worktrees).To(BeNil())
	})
}

func TestTOMLRoundtrip(t *testing.T) {
	t.Run("TOML serialization roundtrips correctly", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create a state with all fields populated
		_, err := state.Init(dir, "roundtrip-test", nowFunc(), state.InitOpts{
			Workflow: "scoped",
			Issue:    "ISSUE-99",
			RepoDir:  "/some/repo",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Transition to init_state for scoped workflow
		_, err = state.Transition(dir, "item_select", state.TransitionOpts{
			Task:     "TASK-001",
			Subphase: "red",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Add pair state
		_, err = state.SetPair(dir, "tdd-red", state.PairState{
			Iteration:          2,
			MaxIterations:      3,
			ProducerComplete:   true,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "Need more test coverage",
			SpawnAttempts:      1,
			FailedModels:       []string{"haiku"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Add worktree state
		_, err = state.SetWorktree(dir, "TASK-001", state.WorktreeState{
			Path:    "/worktrees/task-001",
			Branch:  "task/TASK-001",
			Created: fixedTime(),
			Status:  "active",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Mark a task complete
		_, err = state.MarkTaskComplete(dir, "TASK-000")
		g.Expect(err).ToNot(HaveOccurred())

		// Load state back
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify all fields match
		g.Expect(loaded.Project.Name).To(Equal("roundtrip-test"))
		g.Expect(loaded.Project.Workflow).To(Equal("scoped"))
		g.Expect(loaded.Project.Issue).To(Equal("ISSUE-99"))
		g.Expect(loaded.Project.RepoDir).To(Equal("/some/repo"))
		g.Expect(loaded.Progress.CurrentTask).To(Equal("TASK-001"))
		g.Expect(loaded.Progress.CurrentSubphase).To(Equal("red"))
		g.Expect(loaded.History).To(HaveLen(2))
		g.Expect(loaded.Pairs).To(HaveKey("tdd-red"))
		g.Expect(loaded.Pairs["tdd-red"].Iteration).To(Equal(2))
		g.Expect(loaded.Pairs["tdd-red"].QAVerdict).To(Equal("improvement-request"))
		g.Expect(loaded.Pairs["tdd-red"].ImprovementRequest).To(Equal("Need more test coverage"))
		g.Expect(loaded.Pairs["tdd-red"].FailedModels).To(Equal([]string{"haiku"}))
		g.Expect(loaded.Worktrees).To(HaveKey("TASK-001"))
		g.Expect(loaded.Worktrees["TASK-001"].Path).To(Equal("/worktrees/task-001"))
		g.Expect(loaded.Worktrees["TASK-001"].Status).To(Equal("active"))
		g.Expect(loaded.Progress.CompletedTasks).To(Equal([]string{"TASK-000"}))
	})

	t.Run("State file is valid TOML that can be parsed externally", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "toml-valid", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify the file exists and is readable TOML
		statePath := filepath.Join(dir, state.StateFile)
		content, err := os.ReadFile(statePath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(content).ToNot(BeEmpty())
		g.Expect(string(content)).To(ContainSubstring("name = \"toml-valid\""))
		g.Expect(string(content)).To(ContainSubstring("phase = \"init\""))
	})
}

func TestStateValidation(t *testing.T) {
	t.Run("Get rejects malformed TOML", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create an invalid TOML file
		statePath := filepath.Join(dir, state.StateFile)
		err := os.WriteFile(statePath, []byte("invalid {{{ toml"), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Get(dir)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("failed to read"))
	})

	t.Run("Get handles missing state file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Get(dir)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("failed to read"))
	})

	t.Run("Init rejects non-existent directory", func(t *testing.T) {
		g := NewWithT(t)

		_, err := state.Init("/nonexistent/invalid/path", "test", nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	})

	t.Run("Init rejects file as directory", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create a file instead of directory
		filePath := filepath.Join(dir, "not-a-dir")
		err := os.WriteFile(filePath, []byte("test"), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Init(filePath, "test", nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	})
}

func TestAtomicWrites(t *testing.T) {
	t.Run("Transition writes atomically", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "atomic-test", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		// Perform transition
		_, err = state.Transition(dir, "pm_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify no .tmp files left behind
		files, err := os.ReadDir(dir)
		g.Expect(err).ToNot(HaveOccurred())

		for _, file := range files {
			g.Expect(file.Name()).ToNot(ContainSubstring(".tmp"),
				"no temporary files should remain after atomic write")
		}

		// Verify state file exists and is readable
		statePath := filepath.Join(dir, state.StateFile)
		_, err = os.Stat(statePath)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("Set writes atomically", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "atomic-test", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Set(dir, state.SetOpts{Issue: "ISSUE-123"})
		g.Expect(err).ToNot(HaveOccurred())

		// Verify no .tmp files
		files, err := os.ReadDir(dir)
		g.Expect(err).ToNot(HaveOccurred())

		for _, file := range files {
			g.Expect(file.Name()).ToNot(ContainSubstring(".tmp"))
		}
	})

	t.Run("SetPair writes atomically", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "atomic-test", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetPair(dir, "pm", state.PairState{Iteration: 1})
		g.Expect(err).ToNot(HaveOccurred())

		// Verify no .tmp files
		files, err := os.ReadDir(dir)
		g.Expect(err).ToNot(HaveOccurred())

		for _, file := range files {
			g.Expect(file.Name()).ToNot(ContainSubstring(".tmp"))
		}
	})
}

func TestErrorStateHandling(t *testing.T) {
	t.Run("Error state persists across Get calls", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "error-test", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Cause an error — try illegal transition from pm_produce
		_, err = state.Transition(dir, "complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())

		// Verify error is persisted
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Error).ToNot(BeNil())
		g.Expect(s.Error.ErrorType).To(Equal("illegal_transition"))
		g.Expect(s.Error.LastPhase).To(Equal("pm_produce"))
		g.Expect(s.Error.TargetPhase).To(Equal("complete"))
	})

	t.Run("Error with TargetPhase persists correctly", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "error-test", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		// Navigate to pm_decide, then try pm_commit with precondition failure
		navigateToState(t, dir, "pm_decide", "new")

		_, err = state.TransitionWithChecker(dir, "pm_commit", state.TransitionOpts{}, nowFunc(), &mockPreconditionChecker{
			requirementsExists: false,
		})
		g.Expect(err).To(HaveOccurred())

		// Verify TargetPhase is set
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Error).ToNot(BeNil())
		g.Expect(s.Error.TargetPhase).To(Equal("pm_commit"))
		g.Expect(s.Error.ErrorType).To(Equal("precondition_failed"))
	})
}

func TestHistoryTracking(t *testing.T) {
	t.Run("History accumulates all transitions", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "history-test", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		phases := []string{"pm_produce", "pm_qa"}
		for _, phase := range phases {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.History).To(HaveLen(3)) // init + pm_produce + pm_qa
		g.Expect(s.History[0].Phase).To(Equal("init"))
		g.Expect(s.History[1].Phase).To(Equal("pm_produce"))
		g.Expect(s.History[2].Phase).To(Equal("pm_qa"))
	})

	t.Run("History timestamps persist correctly", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "history-test", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.History).To(HaveLen(2))
		g.Expect(s.History[0].Timestamp).To(Equal(fixedTime()))
		g.Expect(s.History[1].Timestamp).To(Equal(fixedTime()))
	})
}

func TestPairStateComplexity(t *testing.T) {
	t.Run("Multiple pair states coexist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "multi-pair", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Add multiple pair states
		pairs := map[string]state.PairState{
			"pm": {
				Iteration:     1,
				MaxIterations: 3,
				QAVerdict:     "approved",
			},
			"design": {
				Iteration:          2,
				MaxIterations:      3,
				QAVerdict:          "improvement-request",
				ImprovementRequest: "DES-002 needs clarification",
			},
			"TASK-007": {
				Iteration:     3,
				MaxIterations: 3,
				QAVerdict:     "escalate-user",
			},
		}

		for key, ps := range pairs {
			_, err = state.SetPair(dir, key, ps)
			g.Expect(err).ToNot(HaveOccurred())
		}

		// Verify all persist correctly
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs).To(HaveLen(3))
		g.Expect(s.Pairs["pm"].QAVerdict).To(Equal("approved"))
		g.Expect(s.Pairs["design"].ImprovementRequest).To(Equal("DES-002 needs clarification"))
		g.Expect(s.Pairs["TASK-007"].Iteration).To(Equal(3))
	})

	t.Run("FailedModels array persists correctly", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "failed-models", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetPair(dir, "pm", state.PairState{
			Iteration:     2,
			SpawnAttempts: 3,
			FailedModels:  []string{"haiku", "sonnet", "opus"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["pm"].FailedModels).To(Equal([]string{"haiku", "sonnet", "opus"}))
	})
}

func TestStateRoundtripProperty(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Generate random valid state data
		name := rapid.StringMatching(`[a-z][a-z0-9-]{1,20}`).Draw(rt, "name")
		workflow := rapid.SampledFrom([]string{"new", "scoped", "align"}).Draw(rt, "workflow")
		issue := rapid.StringMatching(`ISSUE-[0-9]{1,3}`).Draw(rt, "issue")

		// Initialize state
		s, err := state.Init(dir, name, nowFunc(), state.InitOpts{
			Workflow: workflow,
			Issue:    issue,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Verify roundtrip
		loaded, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(loaded.Project.Name).To(Equal(s.Project.Name))
		g.Expect(loaded.Project.Workflow).To(Equal(s.Project.Workflow))
		g.Expect(loaded.Project.Issue).To(Equal(s.Project.Issue))
		g.Expect(loaded.Project.Phase).To(Equal("init"))
	})
}

func TestTransitionOptsPersistence(t *testing.T) {
	t.Run("Task and Subphase opts persist through transition", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "opts-test", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		// Transition with opts
		_, err = state.Transition(dir, "pm_produce", state.TransitionOpts{
			Task:     "TASK-042",
			Subphase: "interview",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify opts persisted
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Progress.CurrentTask).To(Equal("TASK-042"))
		g.Expect(s.Progress.CurrentSubphase).To(Equal("interview"))
	})

	t.Run("RepoDir from state propagates to transition opts", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "repo-test", nowFunc(), state.InitOpts{
			Workflow: "new",
			RepoDir:  "/my/repo",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Transition without explicit RepoDir
		s, err := state.Transition(dir, "pm_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify RepoDir still accessible
		g.Expect(s.Project.RepoDir).To(Equal("/my/repo"))
	})
}
