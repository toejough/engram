package correct_test

// Tests for ARCH-4: Correction Detection
// Defines DetectCorrection target function with MemoryStore/OverlapGate I/O mocks.
// Pure deps wired internally: PatternCorpus, Reconciler, SessionRecorder, audit.Logger.
// Won't compile yet — RED phase.

//go:generate impgen store.MemoryStore --dependency
//go:generate impgen reconcile.OverlapGate --dependency
//go:generate impgen correct.DetectCorrection --target

import (
	"context"
	"testing"

	"engram/internal"
	"engram/internal/correct"
	"engram/internal/corpus"
	_ "engram/internal/reconcile"
	_ "engram/internal/store"
	"github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// --- Generators ---

func genPatterns() *rapid.Generator[[]corpus.Pattern] {
	return rapid.Custom(func(t *rapid.T) []corpus.Pattern {
		return []corpus.Pattern{
			{Regex: `^no,`, Label: "no,"},
			{Regex: `^wait`, Label: "wait"},
			{Regex: `\bremember\s+(that|to)`, Label: "remember"},
		}
	})
}

// T-19: No correction pattern match → empty string.
func TestCorrectionDetector_NoMatchReturnsEmpty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockStore, _ := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		// Empty patterns → no match possible
		message := rapid.String().Draw(t, "message")

		wrapper := StartDetectCorrection(t, correct.DetectCorrection,
			mockStore, mockGate, nil, context.Background(), message)

		// DetectCorrection returns (reminder, recordings, auditOutput, error)
		reminder, recordings, _, err := wrapper.ReturnsAs()
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(reminder).To(gomega.BeEmpty())
		g.Expect(recordings).To(gomega.BeEmpty())
	})
}

// T-20: Pattern match triggers reconciliation.
func TestCorrectionDetector_MatchTriggersReconciliation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)
		patterns := genPatterns().Draw(t, "patterns")

		wrapper := StartDetectCorrection(t, correct.DetectCorrection,
			mockStore, mockGate, patterns,
			context.Background(), "no, use specific files not git add -A")

		// Reconciler wired internally calls FindSimilar → Create
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
			gomega.Not(gomega.BeEmpty()),  // reminder
			gomega.HaveLen(1),             // recordings
			gomega.Not(gomega.BeEmpty()),   // audit
			gomega.BeNil(),                // error
		)
	})
}

// T-21: Each of the 15 initial patterns matches expected input.
// This is a pure corpus test — moved to corpus/corpus_test.go.

// T-22: Correction recorded to session log for dedup.
func TestCorrectionDetector_MatchRecordsToSessionLog(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)
		patterns := genPatterns().Draw(t, "patterns")

		wrapper := StartDetectCorrection(t, correct.DetectCorrection,
			mockStore, mockGate, patterns,
			context.Background(), "no, that's not right")

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return(nil, nil)

		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		_, recordings, _, err := wrapper.ReturnsAs()
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(recordings).To(gomega.HaveLen(1))
	})
}

// T-23: Enriched existing memory → system reminder says "Enriched:".
func TestCorrectionDetector_EnrichedExistingMemory_SystemReminder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, expectGate := MockOverlapGate(t)
		patterns := genPatterns().Draw(t, "patterns")

		existing := internal.Memory{
			ID:       "m_0001",
			Title:    "Use git add specific files",
			Keywords: []string{"git", "staging", "no-git-add-A"},
		}

		wrapper := StartDetectCorrection(t, correct.DetectCorrection,
			mockStore, mockGate, patterns,
			context.Background(), "no, don't use git add -A")

		// Reconciler finds overlap → enriches
		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return([]internal.ScoredMemory{{Memory: existing, Score: 0.9}}, nil)

		expectGate.Check.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(internal.Learning{}),
			gomega.Equal(existing),
		).Return(true, "overlapping", nil)

		expectStore.Update.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		reminder, _, _, _ := wrapper.ReturnsAs()
		g.Expect(reminder).To(gomega.ContainSubstring("Enriched:"))
		g.Expect(reminder).To(gomega.ContainSubstring("Use git add specific files"))
		g.Expect(reminder).To(gomega.ContainSubstring("Correction captured"))
	})
}

// T-24: New memory → system reminder says "Created:".
func TestCorrectionDetector_CreatedNewMemory_SystemReminder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)
		patterns := genPatterns().Draw(t, "patterns")

		wrapper := StartDetectCorrection(t, correct.DetectCorrection,
			mockStore, mockGate, patterns,
			context.Background(), "wait, this project uses bun not npm")

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return(nil, nil)

		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		reminder, _, _, _ := wrapper.ReturnsAs()
		g.Expect(reminder).To(gomega.ContainSubstring("Created:"))
	})
}

// T-25: False positive captured anyway (no confirmation).
func TestCorrectionDetector_FalsePositive_CapturedAnyway(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)
		patterns := genPatterns().Draw(t, "patterns")

		// "remember to run tests" matches \bremember\s+(that|to) — false positive
		wrapper := StartDetectCorrection(t, correct.DetectCorrection,
			mockStore, mockGate, patterns,
			context.Background(), "remember to run tests before committing")

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return(nil, nil)

		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		reminder, _, _, err := wrapper.ReturnsAs()
		g.Expect(err).ToNot(gomega.HaveOccurred())
		// Should still capture — no confirmation prompt
		g.Expect(reminder).ToNot(gomega.BeEmpty())
	})
}

// T-26: End-to-end correction detection.
func TestCorrectionDetector_EndToEnd(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)
		patterns := genPatterns().Draw(t, "patterns")

		wrapper := StartDetectCorrection(t, correct.DetectCorrection,
			mockStore, mockGate, patterns,
			context.Background(), "no, use targ test not go test")

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return(nil, nil)

		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		reminder, recordings, auditOutput, err := wrapper.ReturnsAs()
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(reminder).ToNot(gomega.BeEmpty())
		g.Expect(recordings).To(gomega.HaveLen(1))
		g.Expect(auditOutput).ToNot(gomega.BeEmpty())
	})
}
