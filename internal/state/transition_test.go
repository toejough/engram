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

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("pm"))
		g.Expect(s.History).To(HaveLen(2))
		g.Expect(s.History[1].Phase).To(Equal("pm"))
	})

	t.Run("illegal transition returns error with legal targets", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))
		g.Expect(err.Error()).To(ContainSubstring("init"))
		g.Expect(err.Error()).To(ContainSubstring("complete"))
		g.Expect(err.Error()).To(ContainSubstring("pm"))
	})

	t.Run("transition with task and subphase opts", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Transition(dir, "pm", state.TransitionOpts{
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

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Read back from disk
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("pm"))
		g.Expect(s.History).To(HaveLen(2))
	})

	t.Run("multiple sequential transitions build history", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Transition(dir, "pm-complete", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.History).To(HaveLen(3))
		g.Expect(s.History[0].Phase).To(Equal("init"))
		g.Expect(s.History[1].Phase).To(Equal("pm"))
		g.Expect(s.History[2].Phase).To(Equal("pm-complete"))
	})

	t.Run("errors on nonexistent state file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
	})
}

func TestIsLegalTransition(t *testing.T) {
	t.Run("new project workflow transitions", func(t *testing.T) {
		g := NewWithT(t)
		// Init to PM
		g.Expect(state.IsLegalTransition("init", "pm")).To(BeTrue())
		// PM phase
		g.Expect(state.IsLegalTransition("pm", "pm-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("pm-complete", "design")).To(BeTrue())
		// Design phase
		g.Expect(state.IsLegalTransition("design", "design-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("design-complete", "architect")).To(BeTrue())
		// Architecture phase
		g.Expect(state.IsLegalTransition("architect", "architect-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("architect-complete", "breakdown")).To(BeTrue())
		// Breakdown phase
		g.Expect(state.IsLegalTransition("breakdown", "breakdown-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("breakdown-complete", "implementation")).To(BeTrue())
		// Implementation to Documentation
		g.Expect(state.IsLegalTransition("implementation-complete", "documentation")).To(BeTrue())
		g.Expect(state.IsLegalTransition("documentation", "documentation-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("documentation-complete", "alignment")).To(BeTrue())
	})

	t.Run("TDD loop transitions", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("implementation", "task-start")).To(BeTrue())
		g.Expect(state.IsLegalTransition("task-start", "tdd-red")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd-red", "commit-red")).To(BeTrue())
		g.Expect(state.IsLegalTransition("commit-red", "tdd-green")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd-green", "commit-green")).To(BeTrue())
		g.Expect(state.IsLegalTransition("commit-green", "tdd-refactor")).To(BeTrue())
		g.Expect(state.IsLegalTransition("tdd-refactor", "commit-refactor")).To(BeTrue())
		g.Expect(state.IsLegalTransition("commit-refactor", "task-audit")).To(BeTrue())
		g.Expect(state.IsLegalTransition("task-audit", "task-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("task-audit", "task-retry")).To(BeTrue())
		g.Expect(state.IsLegalTransition("task-audit", "task-escalated")).To(BeTrue())
		g.Expect(state.IsLegalTransition("task-complete", "task-start")).To(BeTrue())
		g.Expect(state.IsLegalTransition("task-complete", "implementation-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("task-retry", "tdd-red")).To(BeTrue())
	})

	t.Run("main flow ending transitions", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("alignment", "alignment-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("alignment-complete", "retro")).To(BeTrue())
		g.Expect(state.IsLegalTransition("retro", "retro-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("retro-complete", "summary")).To(BeTrue())
		g.Expect(state.IsLegalTransition("summary", "summary-complete")).To(BeTrue())
		g.Expect(state.IsLegalTransition("summary-complete", "issue-update")).To(BeTrue())
		g.Expect(state.IsLegalTransition("issue-update", "next-steps")).To(BeTrue())
		g.Expect(state.IsLegalTransition("next-steps", "complete")).To(BeTrue())
	})

	t.Run("adopt workflow transitions (bottom-up)", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("init", "adopt-explore")).To(BeTrue())
		g.Expect(state.IsLegalTransition("adopt-explore", "adopt-infer-tests")).To(BeTrue())
		g.Expect(state.IsLegalTransition("adopt-infer-tests", "adopt-infer-arch")).To(BeTrue())
		g.Expect(state.IsLegalTransition("adopt-infer-arch", "adopt-infer-design")).To(BeTrue())
		g.Expect(state.IsLegalTransition("adopt-infer-design", "adopt-infer-reqs")).To(BeTrue())
		g.Expect(state.IsLegalTransition("adopt-infer-reqs", "adopt-escalations")).To(BeTrue())
		g.Expect(state.IsLegalTransition("adopt-escalations", "adopt-documentation")).To(BeTrue())
		g.Expect(state.IsLegalTransition("adopt-documentation", "alignment")).To(BeTrue())
	})

	t.Run("align workflow transitions", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("init", "align-explore")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align-explore", "align-infer-tests")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align-infer-tests", "align-infer-arch")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align-infer-arch", "align-infer-design")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align-infer-design", "align-infer-reqs")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align-infer-reqs", "align-escalations")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align-escalations", "align-documentation")).To(BeTrue())
		g.Expect(state.IsLegalTransition("align-documentation", "alignment")).To(BeTrue())
	})

	t.Run("task workflow transitions", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("init", "task-implementation")).To(BeTrue())
		g.Expect(state.IsLegalTransition("task-implementation", "task-start")).To(BeTrue())
		g.Expect(state.IsLegalTransition("task-complete", "task-documentation")).To(BeTrue())
		g.Expect(state.IsLegalTransition("task-documentation", "alignment")).To(BeTrue())
	})

	t.Run("known illegal transitions", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("init", "complete")).To(BeFalse())
		g.Expect(state.IsLegalTransition("init", "tdd-red")).To(BeFalse())
		g.Expect(state.IsLegalTransition("pm", "design")).To(BeFalse()) // Must go through pm-complete
		g.Expect(state.IsLegalTransition("complete", "init")).To(BeFalse())
	})

	t.Run("unknown phase returns false", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(state.IsLegalTransition("nonexistent", "init")).To(BeFalse())
	})
}

func TestIsLegalTransitionProperty(t *testing.T) {
	phases := make([]string, 0, len(state.LegalTransitions))
	for k := range state.LegalTransitions {
		phases = append(phases, k)
	}

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		from := rapid.SampledFrom(phases).Draw(rt, "from")
		to := rapid.SampledFrom(phases).Draw(rt, "to")

		result := state.IsLegalTransition(from, to)
		targets := state.LegalTargets(from)

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

	// Every target phase should also be a key in the map (no dangling references)
	for from, targets := range state.LegalTransitions {
		for _, to := range targets {
			_, exists := state.LegalTransitions[to]
			g.Expect(exists).To(BeTrue(),
				"phase %q (target of %q) is not a key in LegalTransitions", to, from)
		}
	}
}

// Test precondition: pm-complete requires requirements.md with REQ-NNN IDs
func TestTransitionPrecondition_PMComplete(t *testing.T) {
	t.Run("fails without requirements.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		checker := &mockPreconditionChecker{
			requirementsExists: false,
		}
		_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("requirements.md"))
	})

	t.Run("fails without REQ-NNN IDs in requirements.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		checker := &mockPreconditionChecker{
			requirementsExists: true,
			requirementsHasIDs: false,
		}
		_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("REQ-"))
	})

	t.Run("succeeds with valid requirements.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		checker := &mockPreconditionChecker{
			requirementsExists: true,
			requirementsHasIDs: true,
		}
		s, err := state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("pm-complete"))
	})
}

// Test precondition: design-complete requires design.md with DES-NNN IDs
func TestTransitionPrecondition_DesignComplete(t *testing.T) {
	t.Run("fails without design.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhase(t, dir, "design")

		checker := &mockPreconditionChecker{
			designExists: false,
		}
		_, err = state.TransitionWithChecker(dir, "design-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("design.md"))
	})

	t.Run("succeeds with valid design.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhase(t, dir, "design")

		checker := &mockPreconditionChecker{
			designExists: true,
			designHasIDs: true,
		}
		s, err := state.TransitionWithChecker(dir, "design-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("design-complete"))
	})
}

// Test precondition: architect-complete requires trace validate to pass
func TestTransitionPrecondition_ArchitectComplete(t *testing.T) {
	t.Run("fails when trace validation fails", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhase(t, dir, "architect")

		checker := &mockPreconditionChecker{
			traceValidationPasses: false,
		}
		_, err = state.TransitionWithChecker(dir, "architect-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("trace"))
	})

	t.Run("succeeds when trace validation passes", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhase(t, dir, "architect")

		checker := &mockPreconditionChecker{
			traceValidationPasses: true,
		}
		s, err := state.TransitionWithChecker(dir, "architect-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("architect-complete"))
	})
}

// Test precondition: task-complete requires trace validation to pass
func TestTransitionPrecondition_TaskComplete(t *testing.T) {
	t.Run("fails when trace validation fails", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhase(t, dir, "task-audit")

		checker := &mockPreconditionChecker{
			traceValidationPasses: false,
		}
		_, err = state.TransitionWithChecker(dir, "task-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("trace"))
	})
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

func (m *mockPreconditionChecker) TraceValidationPasses(dir string) bool {
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

// Test that illegal transitions provide helpful error messages
func TestTransition_HelpfulErrorMessages(t *testing.T) {
	t.Run("explains what phases must complete first", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Try to jump directly to implementation from init
		_, err = state.Transition(dir, "implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).To(HaveOccurred())
		// Should list legal targets
		g.Expect(err.Error()).To(ContainSubstring("pm"))
	})
}

// Test that --force flag allows override of transitions
func TestTransitionWithChecker_ForceFlag(t *testing.T) {
	t.Run("force bypasses preconditions", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Without force, precondition fails
		checker := &mockPreconditionChecker{
			requirementsExists: false,
		}
		_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())

		// With force, precondition is bypassed
		opts := state.TransitionOpts{Force: true}
		s, err := state.TransitionWithChecker(dir, "pm-complete", opts, nowFunc(), checker)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("pm-complete"))
	})

	t.Run("force cannot bypass illegal transitions", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Even with force, can't skip directly to implementation
		opts := state.TransitionOpts{Force: true}
		_, err = state.TransitionWithChecker(dir, "implementation", opts, nowFunc(), nil)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("illegal transition"))
	})
}

// Test task-complete transition checks acceptance criteria
func TestTransitionWithChecker_TaskCompleteChecksAC(t *testing.T) {
	t.Run("blocks when AC incomplete", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhase(t, dir, "task-audit")

		checker := &mockPreconditionChecker{
			traceValidationPasses:      true,
			acceptanceCriteriaComplete: false,
		}
		opts := state.TransitionOpts{Task: "TASK-001"}
		_, err = state.TransitionWithChecker(dir, "task-complete", opts, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("acceptance criteria"))
	})

	t.Run("succeeds when AC complete", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhase(t, dir, "task-audit")

		checker := &mockPreconditionChecker{
			traceValidationPasses:      true,
			acceptanceCriteriaComplete: true,
		}
		opts := state.TransitionOpts{Task: "TASK-001"}
		s, err := state.TransitionWithChecker(dir, "task-complete", opts, nowFunc(), checker)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("task-complete"))
	})
}

// Test state next returns validation_failed when AC incomplete at task-audit
func TestNextWithChecker_ValidationFailed(t *testing.T) {
	t.Run("returns validation_failed when AC incomplete at task-audit", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhaseWithTask(t, dir, "task-audit", "TASK-001")

		checker := &mockPreconditionChecker{
			acceptanceCriteriaComplete:   false,
			incompleteAcceptanceCriteria: []string{"First AC item", "Second AC item"},
		}
		result := state.NextWithChecker(dir, checker)
		g.Expect(result.Action).To(Equal("stop"))
		g.Expect(result.Reason).To(Equal("validation_failed"))
		g.Expect(result.Details).To(ContainSubstring("acceptance criteria for TASK-001 are incomplete"))
	})

	t.Run("returns continue when AC complete at task-audit", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhaseWithTask(t, dir, "task-audit", "TASK-001")

		checker := &mockPreconditionChecker{
			acceptanceCriteriaComplete: true,
		}
		result := state.NextWithChecker(dir, checker)
		g.Expect(result.Action).To(Equal("continue"))
		g.Expect(result.NextPhase).To(Equal("task-complete"))
	})
}

// Test state next returns action: continue when unblocked work exists
func TestNext_Continue(t *testing.T) {
	t.Run("returns continue when in tdd-red", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhase(t, dir, "tdd-red")

		result := state.Next(dir)
		g.Expect(result.Action).To(Equal("continue"))
		g.Expect(result.NextPhase).To(Equal("commit-red"))
	})

	t.Run("continues across phases", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhase(t, dir, "commit-green")

		result := state.Next(dir)
		g.Expect(result.Action).To(Equal("continue"))
		g.Expect(result.NextPhase).To(Equal("tdd-refactor"))
	})
}

// Test state next returns action: stop when all complete
func TestNext_Stop(t *testing.T) {
	t.Run("returns stop at terminal states", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		walkToPhase(t, dir, "next-steps")
		_, err = state.TransitionWithChecker(dir, "complete", state.TransitionOpts{}, nowFunc(), nil)
		g.Expect(err).ToNot(HaveOccurred())

		result := state.Next(dir)
		g.Expect(result.Action).To(Equal("stop"))
		g.Expect(result.Reason).To(Equal("all_complete"))
	})
}

// Test NextResult contains required fields
func TestNextResult_Fields(t *testing.T) {
	g := NewWithT(t)

	result := state.NextResult{
		Action:    "continue",
		NextPhase: "tdd-green",
		NextTask:  "TASK-001",
		Reason:    "",
	}

	g.Expect(result.Action).To(Equal("continue"))
	g.Expect(result.NextPhase).To(Equal("tdd-green"))
	g.Expect(result.NextTask).To(Equal("TASK-001"))
}

// Test failed transition captures error details in state.toml.
func TestTransition_CapturesError(t *testing.T) {
	t.Run("captures error on precondition failure", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{
			Task: "TASK-001",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Attempt transition that fails precondition
		checker := &mockPreconditionChecker{
			requirementsExists: false,
		}
		_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())

		// Read state and verify error section exists
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Error).ToNot(BeNil())
		g.Expect(s.Error.LastPhase).To(Equal("pm"))
		g.Expect(s.Error.LastTask).To(Equal("TASK-001"))
		g.Expect(s.Error.ErrorType).To(Equal("precondition_failed"))
		g.Expect(s.Error.Message).To(ContainSubstring("requirements.md"))
	})

	t.Run("captures error on illegal transition", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
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

	_, err := state.Init(dir, "test-project", nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	checker := &mockPreconditionChecker{requirementsExists: false}

	// First failure
	_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), checker)
	g.Expect(err).To(HaveOccurred())

	s, _ := state.Get(dir)
	g.Expect(s.Error.RetryCount).To(Equal(1))

	// Second failure
	_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), checker)
	g.Expect(err).To(HaveOccurred())

	s, _ = state.Get(dir)
	g.Expect(s.Error.RetryCount).To(Equal(2))
}

// Test error cleared on successful transition.
func TestTransition_ClearsErrorOnSuccess(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := state.Init(dir, "test-project", nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())

	// Cause a failure first
	checker := &mockPreconditionChecker{requirementsExists: false}
	_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), checker)
	g.Expect(err).To(HaveOccurred())

	s, _ := state.Get(dir)
	g.Expect(s.Error).ToNot(BeNil())

	// Now succeed with force
	_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{Force: true}, nowFunc(), checker)
	g.Expect(err).ToNot(HaveOccurred())

	s, _ = state.Get(dir)
	g.Expect(s.Error).To(BeNil())
}

// Test GetRecovery returns recovery info after failure.
func TestGetRecovery_AfterFailure(t *testing.T) {
	t.Run("shows available actions after failure", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Cause a failure
		checker := &mockPreconditionChecker{requirementsExists: false}
		_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())

		recovery := state.GetRecovery(dir)
		g.Expect(recovery.HasError).To(BeTrue())
		g.Expect(recovery.AvailableActions).To(ContainElements("retry", "skip", "escalate"))
	})

	t.Run("no recovery info when no error", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		recovery := state.GetRecovery(dir)
		g.Expect(recovery.HasError).To(BeFalse())
		g.Expect(recovery.AvailableActions).To(BeEmpty())
	})
}

// Test Retry re-attempts the last failed transition.
func TestRetry(t *testing.T) {
	t.Run("retry succeeds when precondition fixed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Fail first
		failChecker := &mockPreconditionChecker{requirementsExists: false}
		_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), failChecker)
		g.Expect(err).To(HaveOccurred())

		// Retry with fixed precondition
		successChecker := &mockPreconditionChecker{requirementsExists: true, requirementsHasIDs: true}
		s, err := state.Retry(dir, nowFunc(), successChecker)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("pm-complete"))
		g.Expect(s.Error).To(BeNil())
	})

	t.Run("retry errors when no previous failure", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Retry(dir, nowFunc(), nil)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("no previous failure"))
	})
}

// Test Next considers error state.
func TestNext_ConsidersError(t *testing.T) {
	t.Run("reports error when transition failed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "pm", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Cause a failure
		checker := &mockPreconditionChecker{requirementsExists: false}
		_, err = state.TransitionWithChecker(dir, "pm-complete", state.TransitionOpts{}, nowFunc(), checker)
		g.Expect(err).To(HaveOccurred())

		result := state.Next(dir)
		g.Expect(result.Action).To(Equal("stop"))
		g.Expect(result.Reason).To(Equal("error_pending"))
	})
}

// Test Next() filters completed tasks from suggestions
func TestNextFiltersCompletedTasks(t *testing.T) {
	t.Run("Next excludes completed tasks from task suggestions", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Walk to task-complete, simulating finishing TASK-001
		walkToPhaseWithTask(t, dir, "task-complete", "TASK-001")

		// Mark TASK-001 as complete
		_, err = state.MarkTaskComplete(dir, "TASK-001")
		g.Expect(err).ToNot(HaveOccurred())

		// Checker reports TASK-001 and TASK-002 as unblocked
		checker := &mockPreconditionChecker{
			unblockedTasks: []string{"TASK-001", "TASK-002"},
		}

		// Next() should filter out TASK-001 since it's complete
		result := state.NextWithChecker(dir, checker)
		g.Expect(result.Action).To(Equal("continue"))
		g.Expect(result.NextTask).To(Equal("TASK-002"))
	})

	t.Run("Next returns all_complete when all unblocked tasks are completed", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Walk to task-complete
		walkToPhaseWithTask(t, dir, "task-complete", "TASK-001")

		// Mark both tasks as complete
		_, err = state.MarkTaskComplete(dir, "TASK-001")
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.MarkTaskComplete(dir, "TASK-002")
		g.Expect(err).ToNot(HaveOccurred())

		// Checker reports only completed tasks as unblocked
		checker := &mockPreconditionChecker{
			unblockedTasks: []string{"TASK-001", "TASK-002"},
		}

		// Since both suggested tasks are complete, should suggest implementation-complete
		result := state.NextWithChecker(dir, checker)
		// At task-complete, if no remaining tasks, should suggest implementation-complete
		g.Expect(result.Action).To(Equal("continue"))
		g.Expect(result.NextPhase).To(Equal("implementation-complete"))
	})
}

// walkToPhase transitions through phases to reach the target phase.
func walkToPhaseWithTask(t *testing.T, dir, target, taskID string) {
	t.Helper()
	g := NewWithT(t)

	paths := map[string][]string{
		"task-audit": {
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
			"commit-red", "tdd-green", "commit-green", "tdd-refactor",
			"commit-refactor", "task-audit",
		},
		"task-complete": {
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
			"commit-red", "tdd-green", "commit-green", "tdd-refactor",
			"commit-refactor", "task-audit", "task-complete",
		},
	}

	phases, ok := paths[target]
	if !ok {
		t.Fatalf("unknown target phase: %s", target)
	}

	passChecker := &mockPreconditionChecker{
		requirementsExists:         true,
		requirementsHasIDs:         true,
		designExists:               true,
		designHasIDs:               true,
		traceValidationPasses:      true,
		testsExist:                 true,
		testsFail:                  true,
		testsPass:                  true,
		acceptanceCriteriaComplete: true,
	}

	for _, phase := range phases {
		opts := state.TransitionOpts{}
		if phase == "task-start" || phase == "tdd-red" || phase == "commit-red" ||
			phase == "tdd-green" || phase == "commit-green" || phase == "tdd-refactor" ||
			phase == "commit-refactor" || phase == "task-audit" || phase == "task-complete" {
			opts.Task = taskID
		}
		_, err := state.TransitionWithChecker(dir, phase, opts, nowFunc(), passChecker)
		g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", phase)
	}
}

func walkToPhase(t *testing.T, dir, target string) {
	t.Helper()
	g := NewWithT(t)

	paths := map[string][]string{
		"pm":          {"pm"},
		"pm-complete": {"pm", "pm-complete"},
		"design": {
			"pm", "pm-complete", "design",
		},
		"design-complete": {
			"pm", "pm-complete", "design", "design-complete",
		},
		"architect": {
			"pm", "pm-complete", "design", "design-complete", "architect",
		},
		"architect-complete": {
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete",
		},
		"breakdown": {
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown",
		},
		"tdd-red": {
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
		},
		"commit-green": {
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
			"commit-red", "tdd-green", "commit-green",
		},
		"task-audit": {
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
			"commit-red", "tdd-green", "commit-green", "tdd-refactor",
			"commit-refactor", "task-audit",
		},
		"next-steps": {
			"pm", "pm-complete", "design", "design-complete",
			"architect", "architect-complete", "breakdown", "breakdown-complete",
			"implementation", "task-start", "tdd-red",
			"commit-red", "tdd-green", "commit-green", "tdd-refactor",
			"commit-refactor", "task-audit", "task-complete",
			"implementation-complete", "documentation", "documentation-complete",
			"alignment", "alignment-complete", "retro", "retro-complete",
			"summary", "summary-complete", "issue-update", "next-steps",
		},
	}

	phases, ok := paths[target]
	if !ok {
		t.Fatalf("unknown target phase: %s", target)
	}

	passChecker := &mockPreconditionChecker{
		requirementsExists:         true,
		requirementsHasIDs:         true,
		designExists:               true,
		designHasIDs:               true,
		traceValidationPasses:      true,
		testsExist:                 true,
		testsFail:                  true,
		testsPass:                  true,
		acceptanceCriteriaComplete: true,
		retroExists:                true,
		summaryExists:              true,
	}

	for _, phase := range phases {
		_, err := state.TransitionWithChecker(dir, phase, state.TransitionOpts{}, nowFunc(), passChecker)
		g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", phase)
	}
}
