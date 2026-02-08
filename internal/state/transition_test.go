package state_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"pgregory.net/rapid"
)

func TestTransition(t *testing.T) {
	t.Run("legal transition updates phase and appends history", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Transition(dir, "pm_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("pm_produce"))
		g.Expect(s.History).To(HaveLen(2))
		g.Expect(s.History[1].Phase).To(Equal("pm_produce"))
	})

	t.Run("illegal transition returns error with legal targets", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))
		g.Expect(err.Error()).To(ContainSubstring("init"))
		g.Expect(err.Error()).To(ContainSubstring("complete"))
		g.Expect(err.Error()).To(ContainSubstring("pm_produce"))
	})

	t.Run("transition with task and subphase opts", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Transition(dir, "pm_produce", state.TransitionOpts{
			Task:     "TASK-001",
			Subphase: "interview",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Progress.CurrentTask).To(Equal("TASK-001"))
		g.Expect(s.Progress.CurrentSubphase).To(Equal("interview"))
	})

	t.Run("transition persists atomically to disk", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Read back from disk
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("pm_produce"))
		g.Expect(s.History).To(HaveLen(2))
	})

	t.Run("multiple sequential transitions build history", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Transition(dir, "pm_qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.History).To(HaveLen(3))
		g.Expect(s.History[0].Phase).To(Equal("init"))
		g.Expect(s.History[1].Phase).To(Equal("pm_produce"))
		g.Expect(s.History[2].Phase).To(Equal("pm_qa"))
	})

	t.Run("errors on nonexistent state file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Transition(dir, "pm_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
	})
}

func TestIsLegalTransition(t *testing.T) {
	t.Run("new project workflow transitions", func(t *testing.T) {
		g := NewWithT(t)
		// PM phase
		g.Expect(state.IsLegalTransition("pm_produce", "pm_qa", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("pm_qa", "pm_decide", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("pm_decide", "pm_produce", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("pm_decide", "pm_commit", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("pm_commit", "design_produce", "new")).To(BeTrue())
		// Design phase
		g.Expect(state.IsLegalTransition("design_produce", "design_qa", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("design_decide", "design_commit", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("design_commit", "arch_produce", "new")).To(BeTrue())
		// Architecture phase
		g.Expect(state.IsLegalTransition("arch_produce", "arch_qa", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("arch_decide", "arch_commit", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("arch_commit", "breakdown_produce", "new")).To(BeTrue())
		// Breakdown
		g.Expect(state.IsLegalTransition("breakdown_decide", "breakdown_commit", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("breakdown_commit", "item_select", "new")).To(BeTrue())
	})

	t.Run("TDD loop transitions", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("tdd_red_produce", "tdd_red_qa", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_red_qa", "tdd_red_decide", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_red_decide", "tdd_red_produce", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_red_decide", "tdd_green_produce", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_green_produce", "tdd_green_qa", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_green_decide", "tdd_refactor_produce", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_refactor_decide", "tdd_commit", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd_commit", "merge_acquire", "new")).To(BeTrue())
	})

	t.Run("main flow ending transitions", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("alignment_produce", "alignment_qa", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("alignment_decide", "alignment_commit", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("alignment_commit", "retro_produce", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("retro_decide", "retro_commit", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("retro_commit", "summary_produce", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("summary_decide", "summary_commit", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("summary_commit", "issue_update", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("issue_update", "next_steps", "new")).To(BeTrue())
		g.Expect(state.IsLegalTransition("next_steps", "complete", "new")).To(BeTrue())
	})

	t.Run("align workflow transitions", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("align_infer_tests_produce", "align_infer_arch_produce", "align")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align_infer_arch_produce", "align_infer_design_produce", "align")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align_infer_design_produce", "align_infer_reqs_produce", "align")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align_infer_reqs_produce", "align_crosscut_qa", "align")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align_crosscut_decide", "align_artifact_commit", "align")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align_artifact_commit", "alignment_produce", "align")).To(BeTrue())
	})

	t.Run("scoped workflow transitions", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("item_select", "item_fork", "scoped")).To(BeTrue())
		g.Expect(state.IsLegalTransition("items_done", "documentation_produce", "scoped")).To(BeTrue())
		g.Expect(state.IsLegalTransition("documentation_commit", "alignment_produce", "scoped")).To(BeTrue())
	})

	t.Run("known illegal transitions", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("pm_produce", "design_produce", "new")).To(BeFalse())
		g.Expect(state.IsLegalTransition("complete", "pm_produce", "new")).To(BeFalse())
	})

	t.Run("unknown state returns false", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("nonexistent", "pm_produce", "new")).To(BeFalse())
	})
}

func TestIsLegalTransitionProperty(t *testing.T) {
	cfg := state.TransitionsForWorkflow("new")
	phases := make([]string, 0, len(cfg))
	for k := range cfg {
		phases = append(phases, k)
	}

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		from := rapid.SampledFrom(phases).Draw(rt, "from")
		to := rapid.SampledFrom(phases).Draw(rt, "to")

		result := state.IsLegalTransition(from, to, "new")
		targets := state.LegalTargets(from, "new")

		// Result should be true iff `to` is in targets
		found := false
		for _, tgt := range targets {
			if tgt == to {
				found = true
				break
			}
		}

		g.Expect(result).To(Equal(found),
			"IsLegalTransition(%s, %s) = %v, but %s targets are %v",
			from, to, result, from, targets)
	})
}

func TestTransitionMapCompleteness(t *testing.T) {
	g := NewWithT(t)

	transitions := state.TransitionsForWorkflow("new")

	// Every target phase should also be either a source or a known terminal state
	terminalStates := map[string]bool{
		"complete":      true,
		"phase_blocked": true,
	}

	for from, targets := range transitions {
		for _, to := range targets {
			// Target should either be a source in transitions OR a terminal state
			_, isSource := transitions[to]
			if !isSource {
				g.Expect(terminalStates).To(HaveKey(to),
					"state %q (target of %q) is not a source in transitions and not a known terminal state", to, from)
			}
		}
	}
}

// Test that illegal transitions provide helpful error messages
func TestTransition_HelpfulErrorMessages(t *testing.T) {
	t.Run("explains what phases must complete first", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		// Try to jump directly to design_produce from init
		_, err = state.Transition(dir, "design_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		// Should list legal target (pm_produce for "new" workflow)
		g.Expect(err.Error()).To(ContainSubstring("pm_produce"))
	})
}

// Test that --force flag allows override of preconditions (but not transition graph)
func TestTransitionWithChecker_ForceFlag(t *testing.T) {
	t.Run("force cannot bypass illegal transitions", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		// Even with force, can't skip directly to design_produce
		opts := state.TransitionOpts{Force: true}
		_, err = state.TransitionWithChecker(dir, "design_produce", opts, nowFunc(), nil)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))
	})
}

// Test failed transition captures error details in state.toml.
func TestTransition_CapturesError(t *testing.T) {
	t.Run("captures error on illegal transition", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		// Attempt illegal transition
		_, err = state.Transition(dir, "complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())

		// Read state and verify error section exists
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Error).ToNot(BeNil())
		g.Expect(s.Error.ErrorType).To(Equal("illegal_transition"))
	})
}

// Test retry count increments on repeated failures.
func TestTransition_RetryCountIncrements(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
	g.Expect(err).ToNot(HaveOccurred())

	// Attempt same illegal transition twice
	_, err = state.Transition(dir, "complete", state.TransitionOpts{}, nowFunc())
	g.Expect(err).To(HaveOccurred())

	s, _ := state.Get(dir)
	g.Expect(s.Error.RetryCount).To(Equal(1))

	_, err = state.Transition(dir, "complete", state.TransitionOpts{}, nowFunc())
	g.Expect(err).To(HaveOccurred())

	s, _ = state.Get(dir)
	g.Expect(s.Error.RetryCount).To(Equal(2))
}

// Test error cleared on successful transition.
func TestTransition_ClearsErrorOnSuccess(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
	g.Expect(err).ToNot(HaveOccurred())

	// Cause a failure first
	_, err = state.Transition(dir, "complete", state.TransitionOpts{}, nowFunc())
	g.Expect(err).To(HaveOccurred())

	s, _ := state.Get(dir)
	g.Expect(s.Error).ToNot(BeNil())

	// Now succeed with valid transition
	s, err = state.Transition(dir, "pm_produce", state.TransitionOpts{}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.Error).To(BeNil())
}

// mockPreconditionChecker implements state.PreconditionChecker for testing.
type mockPreconditionChecker struct {
	requirementsExists           bool
	requirementsHasIDs           bool
	designExists                 bool
	designHasIDs                 bool
	traceValidationPasses        bool
	testsExist                   bool
	testsFail                    bool
	testsPass                    bool
	acceptanceCriteriaComplete   bool
	incompleteAcceptanceCriteria []string
	unblockedTasks               []string
	currentTaskID                string
	retroExists                  bool
	summaryExists                bool
	issueACComplete              bool
	incompleteIssueAC            []string
}

func (m *mockPreconditionChecker) RequirementsExist(dir string) bool {
	return m.requirementsExists
}

func (m *mockPreconditionChecker) RequirementsHaveIDs(dir string) bool {
	return m.requirementsHasIDs
}

func (m *mockPreconditionChecker) DesignExists(dir string) bool {
	return m.designExists
}

func (m *mockPreconditionChecker) DesignHasIDs(dir string) bool {
	return m.designHasIDs
}

func (m *mockPreconditionChecker) TraceValidationPasses(dir string, phase string) bool {
	return m.traceValidationPasses
}

func (m *mockPreconditionChecker) TestsExist(dir string) bool {
	return m.testsExist
}

func (m *mockPreconditionChecker) TestsFail(dir string) bool {
	return m.testsFail
}

func (m *mockPreconditionChecker) TestsPass(dir string) bool {
	return m.testsPass
}

func (m *mockPreconditionChecker) AcceptanceCriteriaComplete(dir, taskID string) bool {
	m.currentTaskID = taskID
	return m.acceptanceCriteriaComplete
}

func (m *mockPreconditionChecker) IncompleteAcceptanceCriteria(dir, taskID string) []string {
	return m.incompleteAcceptanceCriteria
}

func (m *mockPreconditionChecker) UnblockedTasks(dir, failedTask string) []string {
	return m.unblockedTasks
}

func (m *mockPreconditionChecker) RetroExists(dir string) bool {
	return m.retroExists
}

func (m *mockPreconditionChecker) SummaryExists(dir string) bool {
	return m.summaryExists
}

func (m *mockPreconditionChecker) IssueACComplete(repoDir, issueID string) bool {
	return m.issueACComplete
}

func (m *mockPreconditionChecker) IncompleteIssueAC(repoDir, issueID string) []string {
	return m.incompleteIssueAC
}
