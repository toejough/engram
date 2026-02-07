package step_test

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestFullTDDWorkflow verifies TASK-17 acceptance criteria:
// Integration test for full TDD workflow driven by state machine.
// Tests that the orchestrator loop (step next → spawn → step complete) drives the full TDD cycle:
// tdd-red → tdd-red-qa → commit-red → commit-red-qa → tdd-green → tdd-green-qa →
// commit-green → commit-green-qa → tdd-refactor → tdd-refactor-qa → commit-refactor →
// commit-refactor-qa → task-audit
func TestFullTDDWorkflow(t *testing.T) {
	t.Run("state machine drives full TDD cycle", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Initialize state in task workflow
		_, err := state.Init(dir, "test-tdd-workflow", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Transition to task-implementation → task-start → tdd-red
		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: tdd-red ===

		// Step 1: Next should spawn tdd-red-producer
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("tdd-red-producer"))
		g.Expect(result.Model).To(Equal("sonnet"))
		g.Expect(result.Phase).To(Equal("tdd-red"))

		// Step 2: Complete producer spawn
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Step 3: Next should spawn qa
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))
		g.Expect(result.Model).To(Equal("haiku"))

		// Step 4: Complete QA with approved verdict
		err = step.Complete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Step 5: Next should trigger commit
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))

		// Step 6: Complete commit
		err = step.Complete(dir, step.CompleteResult{
			Action: "commit",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Step 7: Next should transition to tdd-red-qa (per state machine graph)
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd-red-qa"))

		// Step 8: Execute transition
		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "tdd-red-qa",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: tdd-red-qa (QA-only phase) ===

		// Step 9: Next should spawn QA (no producer in QA-only phase)
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.Skill).To(Equal("qa"))

		// Step 10: Complete QA
		err = step.Complete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Step 11: Next should transition to commit-red (per state machine graph)
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("commit-red"))

		err = step.Complete(dir, step.CompleteResult{
			Action: "transition",
			Phase:  "commit-red",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: commit-red (producer + QA + commit) ===

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("commit-producer"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))

		err = step.Complete(dir, step.CompleteResult{Action: "commit", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("commit-red-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "commit-red-qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: commit-red-qa (QA-only) ===

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd-green"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd-green"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: tdd-green ===
		// Following same pattern: producer → qa → commit → transition

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("tdd-green-producer"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))

		err = step.Complete(dir, step.CompleteResult{Action: "commit", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd-green-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd-green-qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: tdd-green-qa (QA-only) ===

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("commit-green"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "commit-green"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: commit-green (producer + QA + commit) ===

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))

		err = step.Complete(dir, step.CompleteResult{Action: "commit", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("commit-green-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "commit-green-qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: commit-green-qa ===

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd-refactor"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd-refactor"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: tdd-refactor (producer + QA + commit) ===

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))

		err = step.Complete(dir, step.CompleteResult{Action: "commit", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd-refactor-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd-refactor-qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: tdd-refactor-qa (QA-only) ===

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("commit-refactor"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "commit-refactor"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: commit-refactor (producer + QA + commit) ===

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))

		err = step.Complete(dir, step.CompleteResult{Action: "commit", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("commit-refactor-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "commit-refactor-qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Phase: commit-refactor-qa (QA-only) ===

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-qa", Status: "done", QAVerdict: "approved"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Final: Next should transition to task-audit
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("task-audit"))

		// Verify no composite skill (tdd-producer) was referenced anywhere
		// This is verified implicitly - all spawned skills were atomic (tdd-red-producer, tdd-green-producer, tdd-refactor-producer, commit-producer, qa)
	})
}

// TestQAIterationWithFeedback verifies TASK-18 acceptance criteria:
// Integration test for QA iteration with feedback.
// Tests that improvement-request verdicts cause re-spawning with feedback,
// iteration counter increments, max iteration triggers escalation,
// and approved verdict advances with iteration reset.
func TestQAIterationWithFeedback(t *testing.T) {
	t.Run("QA improvement-request triggers iteration loop", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Initialize and navigate to tdd-red
		_, err := state.Init(dir, "test-qa-iteration", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// === Iteration 1: Producer → QA → improvement-request ===

		// Spawn producer
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))

		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Spawn QA
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		// QA requests improvement
		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "Add edge case tests for empty input",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify iteration incremented
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].Iteration).To(Equal(2))
		g.Expect(s.Pairs["tdd-red"].ImprovementRequest).To(Equal("Add edge case tests for empty input"))
		g.Expect(s.Pairs["tdd-red"].ProducerComplete).To(BeFalse())

		// === Iteration 2: Producer re-spawns with QA feedback ===

		// Next should spawn producer with feedback in context
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Context.QAFeedback).To(Equal("Add edge case tests for empty input"))

		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Spawn QA again
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		// QA requests another improvement
		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "Add boundary tests for max values",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify iteration incremented again
		s, err = state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].Iteration).To(Equal(3))
		g.Expect(s.Pairs["tdd-red"].MaxIterations).To(Equal(3))

		// === Iteration 3: Max iteration reached, one more attempt allowed ===

		// Next should spawn producer (last attempt at max iteration)
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Context.QAFeedback).To(Equal("Add boundary tests for max values"))

		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		// QA requests yet another improvement (this will push iteration to 4, exceeding max)
		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "One more thing...",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify iteration is now beyond max
		s, err = state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].Iteration).To(Equal(4))
		g.Expect(s.Pairs["tdd-red"].MaxIterations).To(Equal(3))

		// === Escalation: iteration 4 > max 3, should escalate ===

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("escalate-user"))
		g.Expect(result.Details).To(ContainSubstring("max iterations (4) exceeded"))
		g.Expect(result.Details).To(ContainSubstring("tdd-red"))
	})

	t.Run("QA approved verdict advances phase with iteration reset", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Initialize and navigate to tdd-red
		_, err := state.Init(dir, "test-qa-approved", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "task-start", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "tdd-red", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// First iteration with improvement-request
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		err = step.Complete(dir, step.CompleteResult{
			Action:     "spawn-qa",
			Status:     "done",
			QAVerdict:  "improvement-request",
			QAFeedback: "Improve tests",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify iteration = 2
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].Iteration).To(Equal(2))

		// Second iteration with approved
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))

		err = step.Complete(dir, step.CompleteResult{Action: "spawn-producer", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		// QA approves this time
		err = step.Complete(dir, step.CompleteResult{
			Action:    "spawn-qa",
			Status:    "done",
			QAVerdict: "approved",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify approved verdict is stored, iteration is still 2
		s, err = state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Pairs["tdd-red"].QAVerdict).To(Equal("approved"))
		g.Expect(s.Pairs["tdd-red"].Iteration).To(Equal(2))

		// Next should trigger commit
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("commit"))

		err = step.Complete(dir, step.CompleteResult{Action: "commit", Status: "done"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Next should transition to next phase
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("transition"))
		g.Expect(result.Phase).To(Equal("tdd-red-qa"))

		err = step.Complete(dir, step.CompleteResult{Action: "transition", Phase: "tdd-red-qa"}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify tdd-red pair state is cleared (iteration reset happens implicitly via clearing)
		s, err = state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		_, exists := s.Pairs["tdd-red"]
		g.Expect(exists).To(BeFalse(), "pair state should be cleared on phase transition")

		// Verify we're now in tdd-red-qa
		g.Expect(s.Project.Phase).To(Equal("tdd-red-qa"))
	})
}

// TestBackwardCompatibilityMigration verifies TASK-19 acceptance criteria:
// Integration test for backward compatibility migration.
// Tests that legacy phase="tdd" is auto-migrated to phase="tdd-red" on state load,
// and that the workflow continues normally after migration.
func TestBackwardCompatibilityMigration(t *testing.T) {
	t.Run("auto-migrates legacy tdd phase to tdd-red", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create a state with legacy phase="tdd" by manually writing state.toml
		_, err := state.Init(dir, "test-migration", nowFunc(), state.InitOpts{
			Workflow: "task",
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Manually set phase to legacy "tdd" using state.SetPair
		// (We can't use Transition because "tdd" is not in the transition graph)
		// Instead, we'll transition to task-implementation, then manually edit the file
		_, err = state.Transition(dir, "task-implementation", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Manually write legacy phase to state file
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		s.Project.Phase = "tdd" // Set legacy phase

		// Write state back to file (using internal knowledge - we'll just re-init with the right phase)
		// Actually, let's use a different approach: read the state.toml file and modify it
		stateFilePath := dir + "/state.toml"
		stateContent := `[project]
name = "test-migration"
created = 2026-01-27T12:00:00Z
phase = "tdd"
workflow = "task"

[[history]]
timestamp = 2026-01-27T12:00:00Z
phase = "init"

[[history]]
timestamp = 2026-01-27T12:00:00Z
phase = "task-implementation"

[progress]
current_task = ""
current_subphase = ""
tasks_complete = 0
tasks_total = 0
tasks_escalated = 0

[conflicts]
open = 0
blocking_tasks = []

[meta]
corrections_since_last_audit = 0
`
		err = os.WriteFile(stateFilePath, []byte(stateContent), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Now read the state - migration should happen automatically
		s, err = state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify migration occurred
		g.Expect(s.Project.Phase).To(Equal("tdd-red"), "legacy tdd should be migrated to tdd-red")

		// Verify pair state was initialized for tdd-red
		g.Expect(s.Pairs).ToNot(BeNil())
		pair, exists := s.Pairs["tdd-red"]
		g.Expect(exists).To(BeTrue(), "tdd-red pair state should be initialized")
		g.Expect(pair.Iteration).To(Equal(0))
		g.Expect(pair.MaxIterations).To(Equal(3))

		// Verify workflow continues normally after migration
		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.Skill).To(Equal("tdd-red-producer"))
		g.Expect(result.Phase).To(Equal("tdd-red"))

		// Complete the producer spawn to verify state machine continues
		err = step.Complete(dir, step.CompleteResult{
			Action: "spawn-producer",
			Status: "done",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		// Verify next action continues normally
		result, err = step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))

		// Verify migration is logged (happens during Get, should be persisted to disk)
		// Re-read state to ensure migration was persisted
		s2, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s2.Project.Phase).To(Equal("tdd-red"), "migrated phase should persist")
	})

	t.Run("migration is idempotent", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create legacy state
		stateFilePath := dir + "/state.toml"
		stateContent := `[project]
name = "test-idempotent"
created = 2026-01-27T12:00:00Z
phase = "tdd"
workflow = "task"

[[history]]
timestamp = 2026-01-27T12:00:00Z
phase = "init"

[progress]
current_task = ""
current_subphase = ""
tasks_complete = 0
tasks_total = 0
tasks_escalated = 0

[conflicts]
open = 0
blocking_tasks = []

[meta]
corrections_since_last_audit = 0
`
		err := os.WriteFile(stateFilePath, []byte(stateContent), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// First migration
		s1, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s1.Project.Phase).To(Equal("tdd-red"))

		// Second Get should not re-migrate or error
		s2, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s2.Project.Phase).To(Equal("tdd-red"))

		// Pair state should still be correct
		pair, exists := s2.Pairs["tdd-red"]
		g.Expect(exists).To(BeTrue())
		g.Expect(pair.Iteration).To(Equal(0))
		g.Expect(pair.MaxIterations).To(Equal(3))
	})
}
