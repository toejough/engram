package catchup_test

// Tests for ARCH-6: Session-End Catch-Up Processor
// Defines CatchupRun target function with Evaluator/MemoryStore/OverlapGate I/O mocks.
// Pure deps wired internally: Reconciler, PatternCorpus, SessionLog, audit.Logger.
// Won't compile yet — RED phase.

//go:generate impgen catchup.Evaluator --dependency
//go:generate impgen store.MemoryStore --dependency
//go:generate impgen reconcile.OverlapGate --dependency
//go:generate impgen catchup.CatchupRun --target

import (
	"context"
	"testing"

	"engram/internal"
	"engram/internal/catchup"
	_ "engram/internal/reconcile"
	_ "engram/internal/store"
	"github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// T-32: When the evaluator finds no missed corrections, no new memories are created.
func TestCatchupProcessor_NoMissedCorrections_NoNewMemories(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockEvaluator, expectEvaluator := MockEvaluator(t)
		mockStore, _ := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		capturedEvents := []internal.CorrectionEvent{}

		wrapper := StartCatchupRun(t, catchup.CatchupRun,
			mockEvaluator, mockStore, mockGate, capturedEvents,
			context.Background(), transcript)

		// Evaluator finds nothing missed
		expectEvaluator.FindMissed.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
			gomega.Equal(capturedEvents),
		).Return(nil, nil)

		// CatchupRun returns (candidates, auditOutput, error)
		wrapper.ReturnsShould(
			gomega.BeEmpty(),              // no candidates added
			gomega.BeAssignableToTypeOf(""), // audit output
			gomega.BeNil(),                // no error
		)
	})
}

// T-33: A missed correction goes through the reconciler and produces a memory.
func TestCatchupProcessor_MissedCorrectionReconciled(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockEvaluator, expectEvaluator := MockEvaluator(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")

		wrapper := StartCatchupRun(t, catchup.CatchupRun,
			mockEvaluator, mockStore, mockGate, nil,
			context.Background(), transcript)

		// Evaluator finds one missed correction
		expectEvaluator.FindMissed.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
			gomega.BeAssignableToTypeOf([]internal.CorrectionEvent{}),
		).Return([]internal.MissedCorrection{
			{Content: "you didn't shut them down", Context: "teammate cleanup", Phrase: `\byou didn't\b`},
		}, nil)

		// Reconciler (wired real) calls FindSimilar → Create
		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return(nil, nil)

		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		wrapper.ReturnsShould(
			gomega.Not(gomega.BeEmpty()), // candidates added
			gomega.Not(gomega.BeEmpty()), // audit output
			gomega.BeNil(),              // no error
		)
	})
}

// T-34: A correction phrase from a missed correction is appended to the corpus as a candidate.
func TestCatchupProcessor_NewPatternAddedAsCandidate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockEvaluator, expectEvaluator := MockEvaluator(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		phrase := rapid.String().Draw(t, "phrase")

		wrapper := StartCatchupRun(t, catchup.CatchupRun,
			mockEvaluator, mockStore, mockGate, nil,
			context.Background(), transcript)

		expectEvaluator.FindMissed.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
			gomega.BeAssignableToTypeOf([]internal.CorrectionEvent{}),
		).Return([]internal.MissedCorrection{
			{Content: "missed correction", Context: "context", Phrase: phrase},
		}, nil)

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return(nil, nil)

		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		candidates, _, err := wrapper.ReturnsAs()
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(candidates).To(gomega.HaveLen(1))
		g.Expect(candidates[0].Regex).To(gomega.Equal(phrase))
	})
}

// T-35: Full scenario — missed correction → memory + candidate pattern + audit.
func TestCatchupProcessor_CorpusGrowth_Scenario(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockEvaluator, expectEvaluator := MockEvaluator(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		capturedEvents := []internal.CorrectionEvent{
			{MemoryID: "m_other", Pattern: `^no,`, Message: "no, use bun"},
		}

		wrapper := StartCatchupRun(t, catchup.CatchupRun,
			mockEvaluator, mockStore, mockGate, capturedEvents,
			context.Background(), transcript)

		expectEvaluator.FindMissed.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
			gomega.Equal(capturedEvents),
		).Return([]internal.MissedCorrection{
			{
				Content: "you didn't shut them down before ending",
				Context: "user corrected about orphaned teammates",
				Phrase:  `\byou didn't\b`,
			},
		}, nil)

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return(nil, nil)

		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		candidates, auditOutput, err := wrapper.ReturnsAs()
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(candidates).To(gomega.HaveLen(1))
		g.Expect(candidates[0].Regex).To(gomega.Equal(`\byou didn't\b`))
		g.Expect(auditOutput).ToNot(gomega.BeEmpty())
	})
}
