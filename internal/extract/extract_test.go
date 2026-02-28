package extract_test

// Tests for ARCH-3: Extraction Pipeline
// Defines ExtractRun target function, Enricher/Classifier I/O mocks,
// and MemoryStore/OverlapGate I/O mocks (through wired-real Reconciler).
// Won't compile yet — RED phase.

//go:generate impgen extract.Enricher --dependency
//go:generate impgen extract.Classifier --dependency
//go:generate impgen store.MemoryStore --dependency
//go:generate impgen reconcile.OverlapGate --dependency
//go:generate impgen extract.ExtractRun --target

import (
	"context"
	"testing"

	"engram/internal"
	"engram/internal/extract"
	_ "engram/internal/reconcile"
	_ "engram/internal/store"
	"github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// --- Generators ---

func genRawLearning() *rapid.Generator[extract.RawLearning] {
	return rapid.Custom(func(t *rapid.T) extract.RawLearning {
		return extract.RawLearning{
			Title:    rapid.String().Draw(t, "title"),
			Content:  rapid.String().Draw(t, "content"),
			Keywords: rapid.SliceOfN(rapid.String(), 1, 5).Draw(t, "keywords"),
		}
	})
}

// T-10: Empty transcript → zero memories, no errors.
func TestExtractor_EmptyTranscriptProducesNoMemories(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockEnricher, expectEnricher := MockEnricher(t)
		mockClassifier, _ := MockClassifier(t)
		mockStore, _ := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")

		wrapper := StartExtractRun(t, extract.ExtractRun,
			mockEnricher, mockClassifier, mockStore, mockGate,
			nil, context.Background(), transcript)

		// Enricher returns nothing → pipeline stops, no further calls
		expectEnricher.Enrich.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
		).Return(nil, nil)

		// No Classifier/Store/Gate calls expected (imptest fails on unexpected calls)
		wrapper.ReturnsShould(gomega.BeAssignableToTypeOf(""), gomega.BeNil())
	})
}

// T-11: Quality gate rejects vague content → no reconciliation.
func TestExtractor_QualityGateRejectsVagueContent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockEnricher, expectEnricher := MockEnricher(t)
		mockClassifier, _ := MockClassifier(t)
		mockStore, _ := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		vagueLearning := extract.RawLearning{
			Title:   "Vague",
			Content: rapid.String().Draw(t, "vagueContent"),
		}

		wrapper := StartExtractRun(t, extract.ExtractRun,
			mockEnricher, mockClassifier, mockStore, mockGate,
			nil, context.Background(), transcript)

		// Enricher returns a learning → gate will reject it
		expectEnricher.Enrich.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
		).Return([]extract.RawLearning{vagueLearning}, nil)

		// Gate rejects → no Classifier/Store/Gate calls
		// Audit should record rejection
		auditOutput, _ := wrapper.ReturnsAs()
		g.Expect(auditOutput).To(gomega.ContainSubstring("rejected"))
	})
}

// T-12: Every created memory has a confidence tier.
func TestExtractor_EveryMemoryHasConfidenceTier(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockEnricher, expectEnricher := MockEnricher(t)
		mockClassifier, expectClassifier := MockClassifier(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		l1 := genRawLearning().Draw(t, "l1")
		l2 := genRawLearning().Draw(t, "l2")
		tier := rapid.SampledFrom([]string{"A", "B", "C"}).Draw(t, "tier")

		wrapper := StartExtractRun(t, extract.ExtractRun,
			mockEnricher, mockClassifier, mockStore, mockGate,
			nil, context.Background(), transcript)

		expectEnricher.Enrich.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
		).Return([]extract.RawLearning{l1, l2}, nil)

		// Classifier called for each learning
		for _, l := range []extract.RawLearning{l1, l2} {
			expectClassifier.Classify.ArgsShould(
				gomega.BeAssignableToTypeOf(context.Background()),
				gomega.Equal(l),
				gomega.Equal(transcript),
			).Return(tier, nil)

			expectStore.FindSimilar.ArgsShould(
				gomega.BeAssignableToTypeOf(context.Background()),
				gomega.BeAssignableToTypeOf(""),
				gomega.BeNumerically(">", 0),
			).Return(nil, nil)

			expectStore.Create.ArgsShould(
				gomega.BeAssignableToTypeOf(context.Background()),
				gomega.Not(gomega.BeNil()),
			).Return(nil)
		}

		auditOutput, _ := wrapper.ReturnsAs()
		g.Expect(auditOutput).To(gomega.ContainSubstring("confidence="))
	})
}

// T-13: Learnings matching session log are skipped (dedup).
func TestExtractor_DedupSkipsMidSessionCorrections(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockEnricher, expectEnricher := MockEnricher(t)
		mockClassifier, expectClassifier := MockClassifier(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		overlapping := genRawLearning().Draw(t, "overlapping")
		fresh := genRawLearning().Draw(t, "fresh")

		// Session log marks the overlapping learning's content as already captured
		sessionOverlaps := map[string]bool{overlapping.Content: true}

		wrapper := StartExtractRun(t, extract.ExtractRun,
			mockEnricher, mockClassifier, mockStore, mockGate,
			sessionOverlaps, context.Background(), transcript)

		expectEnricher.Enrich.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
		).Return([]extract.RawLearning{overlapping, fresh}, nil)

		// Only the fresh learning goes through classifier + reconciler
		expectClassifier.Classify.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(fresh),
			gomega.Equal(transcript),
		).Return("B", nil)

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return(nil, nil)

		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		auditOutput, _ := wrapper.ReturnsAs()
		g.Expect(auditOutput).To(gomega.ContainSubstring("skipped"))
	})
}

// T-14: Overlap → existing memory enriched.
func TestExtractor_ReconciliationEnrichesOnOverlap(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockEnricher, expectEnricher := MockEnricher(t)
		mockClassifier, expectClassifier := MockClassifier(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, expectGate := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		learning := genRawLearning().Draw(t, "learning")
		existing := internal.Memory{
			ID:       "m_existing",
			Title:    rapid.String().Draw(t, "existingTitle"),
			Content:  rapid.String().Draw(t, "existingContent"),
			Keywords: rapid.SliceOfN(rapid.String(), 1, 3).Draw(t, "existingKW"),
		}

		wrapper := StartExtractRun(t, extract.ExtractRun,
			mockEnricher, mockClassifier, mockStore, mockGate,
			nil, context.Background(), transcript)

		expectEnricher.Enrich.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
		).Return([]extract.RawLearning{learning}, nil)

		expectClassifier.Classify.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(learning),
			gomega.Equal(transcript),
		).Return("B", nil)

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

		auditOutput, _ := wrapper.ReturnsAs()
		g.Expect(auditOutput).To(gomega.ContainSubstring("enriched"))
	})
}

// T-15: No overlap → new memory created.
func TestExtractor_ReconciliationCreatesOnNoOverlap(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockEnricher, expectEnricher := MockEnricher(t)
		mockClassifier, expectClassifier := MockClassifier(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		learning := genRawLearning().Draw(t, "learning")

		wrapper := StartExtractRun(t, extract.ExtractRun,
			mockEnricher, mockClassifier, mockStore, mockGate,
			nil, context.Background(), transcript)

		expectEnricher.Enrich.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
		).Return([]extract.RawLearning{learning}, nil)

		expectClassifier.Classify.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(learning),
			gomega.Equal(transcript),
		).Return("C", nil)

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return(nil, nil)

		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		auditOutput, _ := wrapper.ReturnsAs()
		g.Expect(auditOutput).To(gomega.ContainSubstring("created"))
	})
}

// T-16: 4 learnings, 1 dedup → 3 reconciled.
func TestExtractor_RealSessionScenario_ThreeLearningsOneDedup(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockEnricher, expectEnricher := MockEnricher(t)
		mockClassifier, expectClassifier := MockClassifier(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		l1 := genRawLearning().Draw(t, "l1")
		l2 := genRawLearning().Draw(t, "l2")
		l3 := genRawLearning().Draw(t, "l3")
		l4 := genRawLearning().Draw(t, "l4")

		// l2 is already captured mid-session
		sessionOverlaps := map[string]bool{l2.Content: true}

		wrapper := StartExtractRun(t, extract.ExtractRun,
			mockEnricher, mockClassifier, mockStore, mockGate,
			sessionOverlaps, context.Background(), transcript)

		expectEnricher.Enrich.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
		).Return([]extract.RawLearning{l1, l2, l3, l4}, nil)

		// 3 learnings go through (l2 skipped)
		for _, l := range []extract.RawLearning{l1, l3, l4} {
			expectClassifier.Classify.ArgsShould(
				gomega.BeAssignableToTypeOf(context.Background()),
				gomega.Equal(l),
				gomega.Equal(transcript),
			).Return("B", nil)

			expectStore.FindSimilar.ArgsShould(
				gomega.BeAssignableToTypeOf(context.Background()),
				gomega.BeAssignableToTypeOf(""),
				gomega.BeNumerically(">", 0),
			).Return(nil, nil)

			expectStore.Create.ArgsShould(
				gomega.BeAssignableToTypeOf(context.Background()),
				gomega.Not(gomega.BeNil()),
			).Return(nil)
		}

		auditOutput, _ := wrapper.ReturnsAs()
		g.Expect(auditOutput).To(gomega.ContainSubstring("skipped"))
		g.Expect(auditOutput).To(gomega.ContainSubstring("created"))
	})
}

// T-17: All rejected by gate → zero memories, audit records reasons.
func TestExtractor_AllRejected_AuditLogRecordsReasons(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockEnricher, expectEnricher := MockEnricher(t)
		mockClassifier, _ := MockClassifier(t)
		mockStore, _ := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		l1 := genRawLearning().Draw(t, "l1")
		l2 := genRawLearning().Draw(t, "l2")

		wrapper := StartExtractRun(t, extract.ExtractRun,
			mockEnricher, mockClassifier, mockStore, mockGate,
			nil, context.Background(), transcript)

		// Enricher returns learnings, but the internal quality gate rejects all
		expectEnricher.Enrich.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
		).Return([]extract.RawLearning{l1, l2}, nil)

		// No Classifier/Store/Gate calls — gate rejected everything
		auditOutput, _ := wrapper.ReturnsAs()
		g.Expect(auditOutput).To(gomega.ContainSubstring("rejected"))
	})
}

// T-18: End-to-end: enricher → gate → classifier → reconciler → audit.
func TestExtractor_PipelineEndToEnd(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockEnricher, expectEnricher := MockEnricher(t)
		mockClassifier, expectClassifier := MockClassifier(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)

		transcript := rapid.SliceOf(rapid.Byte()).Draw(t, "transcript")
		learning := genRawLearning().Draw(t, "learning")

		wrapper := StartExtractRun(t, extract.ExtractRun,
			mockEnricher, mockClassifier, mockStore, mockGate,
			nil, context.Background(), transcript)

		expectEnricher.Enrich.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(transcript),
		).Return([]extract.RawLearning{learning}, nil)

		expectClassifier.Classify.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(learning),
			gomega.Equal(transcript),
		).Return("A", nil)

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.BeNumerically(">", 0),
		).Return(nil, nil)

		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		auditOutput, err := wrapper.ReturnsAs()
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(auditOutput).ToNot(gomega.BeEmpty())
	})
}
