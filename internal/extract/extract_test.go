package extract_test

import (
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"
	"pgregory.net/rapid"

	"engram/internal/extract"
)

func TestT10_EmptyTranscriptProducesNoMemories(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript t
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		ctx := context.Background()

		mockEnricher, enricherExp := MockEnricher(t)
		mockClassifier, _ := MockClassifier(t)
		mockReconciler, _ := MockReconciler(t)

		// When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
		call := StartRun(
			t,
			extract.Run,
			ctx,
			mockEnricher,
			mockClassifier,
			mockReconciler,
			nil,
			transcript,
		)

		// Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)
		// Given empty learnings, nil error; When enricher.Enrich responds with (empty, nil)
		enricherExp.Enrich.ArgsShould(match.BeAny, Equal(transcript)).
			Return(nil, nil)

		// Then ExtractRun returns (any string, nil error)
		// And ExtractRun never calls classifier, store, or gate
		call.ReturnsShould(match.BeAny, Not(HaveOccurred()))
	})
}

func TestT11_QualityGateRejectsContentBelowTokenThreshold(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript t, any RawLearning with content shorter than 10 tokens
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		short := genShortLearning().Draw(rt, "short")
		ctx := context.Background()

		mockEnricher, enricherExp := MockEnricher(t)
		mockClassifier, _ := MockClassifier(t)
		mockReconciler, _ := MockReconciler(t)

		// When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
		call := StartRun(
			t,
			extract.Run,
			ctx,
			mockEnricher,
			mockClassifier,
			mockReconciler,
			nil,
			transcript,
		)

		// Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)
		// Given [shortLearning], nil error; When enricher.Enrich responds with ([shortLearning], nil)
		enricherExp.Enrich.ArgsShould(match.BeAny, Equal(transcript)).
			Return([]extract.RawLearning{short}, nil)

		// Then ExtractRun returns (string containing "rejected", any error)
		// And ExtractRun never calls classifier, store, or gate
		call.ReturnsShould(ContainSubstring("rejected"), Not(HaveOccurred()))
	})
}

func TestT12_EveryCreatedMemoryHasConfidenceTier(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript t, any two RawLearnings learning1 learning2, any tier in {A, B, C}
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		learning1 := genRawLearning().Draw(rt, "l1")
		learning2 := genRawLearning().Draw(rt, "l2")
		tier := rapid.SampledFrom([]string{"A", "B", "C"}).Draw(rt, "tier")
		ctx := context.Background()

		mockEnricher, enricherExp := MockEnricher(t)
		mockClassifier, classifierExp := MockClassifier(t)
		mockReconciler, reconcilerExp := MockReconciler(t)

		// When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
		call := StartRun(
			t,
			extract.Run,
			ctx,
			mockEnricher,
			mockClassifier,
			mockReconciler,
			nil,
			transcript,
		)

		// Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)
		// Given [learning1, learning2], nil error
		enricherExp.Enrich.ArgsShould(match.BeAny, Equal(transcript)).
			Return([]extract.RawLearning{learning1, learning2}, nil)

		// Then for each learning, ExtractRun calls classifier.Classify with (any ctx, learning, t)
		// Given tier, nil error; When classifier.Classify responds with (tier, nil)
		classifierExp.Classify.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
			Return(tier, nil)
		// Then ExtractRun calls reconciler.Reconcile; Given "created", nil
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(extract.ReconcileResult{Action: "created"}, nil)

		// Then continues to next learning (same sequence repeats for learning2)
		classifierExp.Classify.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
			Return(tier, nil)
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(extract.ReconcileResult{Action: "created"}, nil)

		// Then ExtractRun returns (string containing "confidence=", nil error)
		call.ReturnsShould(ContainSubstring("confidence="), Not(HaveOccurred()))
	})
}

func TestT13_DedupSkipsMidSessionCorrections(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript t, any two RawLearnings: overlapping and fresh
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		overlapping := genRawLearning().Draw(rt, "overlapping")
		fresh := genRawLearning().Draw(rt, "fresh")
		ctx := context.Background()

		mockEnricher, enricherExp := MockEnricher(t)
		mockClassifier, classifierExp := MockClassifier(t)
		mockReconciler, reconcilerExp := MockReconciler(t)
		mockOverlaps, overlapsExp := MockSessionOverlaps(t)

		// When test calls ExtractRun with (enricher, classifier, store, gate, sessionOverlaps, any ctx, t)
		call := StartRun(
			t,
			extract.Run,
			ctx,
			mockEnricher,
			mockClassifier,
			mockReconciler,
			mockOverlaps,
			transcript,
		)

		// Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)
		// Given [overlapping, fresh], nil error; When enricher.Enrich responds with ([overlapping, fresh], nil)
		enricherExp.Enrich.ArgsShould(match.BeAny, Equal(transcript)).
			Return([]extract.RawLearning{overlapping, fresh}, nil)

		// Then ExtractRun skips overlapping (sessionOverlaps marks overlapping.Content as already captured)
		overlapsExp.HasOverlap.ArgsEqual(overlapping.Content).Return(true)

		// Then ExtractRun calls classifier.Classify with (any ctx, fresh, t)
		overlapsExp.HasOverlap.ArgsEqual(fresh.Content).Return(false)

		// Given "B", nil error; When classifier.Classify responds with ("B", nil)
		classifierExp.Classify.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
			Return("B", nil)
		// Then ExtractRun calls reconciler.Reconcile; Given "created", nil
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(extract.ReconcileResult{Action: "created"}, nil)

		// Then ExtractRun returns (string containing "skipped", nil error)
		call.ReturnsShould(ContainSubstring("skipped"), Not(HaveOccurred()))
	})
}

func TestT14_ReconciliationEnrichesOnOverlap(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript t, any RawLearning
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		learning := genRawLearning().Draw(rt, "learning")
		ctx := context.Background()

		mockEnricher, enricherExp := MockEnricher(t)
		mockClassifier, classifierExp := MockClassifier(t)
		mockReconciler, reconcilerExp := MockReconciler(t)

		// When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
		call := StartRun(
			t,
			extract.Run,
			ctx,
			mockEnricher,
			mockClassifier,
			mockReconciler,
			nil,
			transcript,
		)

		// Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)
		// Given [learning], nil error; When enricher.Enrich responds with ([learning], nil)
		enricherExp.Enrich.ArgsShould(match.BeAny, Equal(transcript)).
			Return([]extract.RawLearning{learning}, nil)

		// Then ExtractRun calls classifier.Classify with (any ctx, learning, t)
		// Given "B", nil error; When classifier.Classify responds with ("B", nil)
		classifierExp.Classify.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
			Return("B", nil)
		// Then ExtractRun calls reconciler; Given "enriched", nil (overlap detected internally)
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(extract.ReconcileResult{Action: "enriched"}, nil)

		// Then ExtractRun returns (string containing "enriched", nil error)
		call.ReturnsShould(ContainSubstring("enriched"), Not(HaveOccurred()))
	})
}

func TestT15_ReconciliationCreatesOnNoOverlap(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript t, any RawLearning
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		learning := genRawLearning().Draw(rt, "learning")
		ctx := context.Background()

		mockEnricher, enricherExp := MockEnricher(t)
		mockClassifier, classifierExp := MockClassifier(t)
		mockReconciler, reconcilerExp := MockReconciler(t)

		// When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
		call := StartRun(
			t,
			extract.Run,
			ctx,
			mockEnricher,
			mockClassifier,
			mockReconciler,
			nil,
			transcript,
		)

		// Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)
		// Given [learning], nil error; When enricher.Enrich responds with ([learning], nil)
		enricherExp.Enrich.ArgsShould(match.BeAny, Equal(transcript)).
			Return([]extract.RawLearning{learning}, nil)

		// Then ExtractRun calls classifier.Classify with (any ctx, learning, t)
		// Given "C", nil error; When classifier.Classify responds with ("C", nil)
		classifierExp.Classify.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
			Return("C", nil)
		// Then ExtractRun calls reconciler; Given "created", nil (no overlap found)
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(extract.ReconcileResult{Action: "created"}, nil)

		// Then ExtractRun returns (string containing "created", nil error)
		call.ReturnsShould(ContainSubstring("created"), Not(HaveOccurred()))
	})
}

func TestT16_RealSessionScenario4Learnings1Dedup3Reconciled(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript t, any four RawLearnings learning1 learning2 learning3 learning4
		// sessionOverlaps marks learning2.Content as already captured
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		learning1 := genRawLearning().Draw(rt, "l1")
		learning2 := genRawLearning().Draw(rt, "l2")
		learning3 := genRawLearning().Draw(rt, "l3")
		learning4 := genRawLearning().Draw(rt, "l4")
		ctx := context.Background()

		mockEnricher, enricherExp := MockEnricher(t)
		mockClassifier, classifierExp := MockClassifier(t)
		mockReconciler, reconcilerExp := MockReconciler(t)
		mockOverlaps, overlapsExp := MockSessionOverlaps(t)

		// When test calls ExtractRun with (enricher, classifier, store, gate, sessionOverlaps, any ctx, t)
		call := StartRun(
			t,
			extract.Run,
			ctx,
			mockEnricher,
			mockClassifier,
			mockReconciler,
			mockOverlaps,
			transcript,
		)

		// Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)
		// Given all 4 learnings, nil error
		enricherExp.Enrich.ArgsShould(match.BeAny, Equal(transcript)).
			Return([]extract.RawLearning{learning1, learning2, learning3, learning4}, nil)

		// Then for each of learning1, learning3, learning4 (learning2 skipped by dedup):
		// learning1: fresh — classify -> "B" -> reconcile -> created
		overlapsExp.HasOverlap.ArgsEqual(learning1.Content).Return(false)
		classifierExp.Classify.ArgsShould(match.BeAny, match.BeAny, match.BeAny).Return("B", nil)
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(extract.ReconcileResult{Action: "created"}, nil)

		// learning2: dedup — sessionOverlaps marks as already captured
		overlapsExp.HasOverlap.ArgsEqual(learning2.Content).Return(true)

		// learning3: fresh — classify -> "B" -> reconcile -> created
		overlapsExp.HasOverlap.ArgsEqual(learning3.Content).Return(false)
		classifierExp.Classify.ArgsShould(match.BeAny, match.BeAny, match.BeAny).Return("B", nil)
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(extract.ReconcileResult{Action: "created"}, nil)

		// learning4: fresh — classify -> "B" -> reconcile -> created
		overlapsExp.HasOverlap.ArgsEqual(learning4.Content).Return(false)
		classifierExp.Classify.ArgsShould(match.BeAny, match.BeAny, match.BeAny).Return("B", nil)
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(extract.ReconcileResult{Action: "created"}, nil)

		// Then ExtractRun returns (string containing "skipped" and "created", nil error)
		call.ReturnsShould(
			And(ContainSubstring("skipped"), ContainSubstring("created")),
			Not(HaveOccurred()),
		)
	})
}

func TestT17_AllRejectedByQualityGate(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript t, any two RawLearnings learning1 learning2 each with content shorter than 10 tokens
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		learning1 := genShortLearning().Draw(rt, "l1")
		learning2 := genShortLearning().Draw(rt, "l2")
		ctx := context.Background()

		mockEnricher, enricherExp := MockEnricher(t)
		mockClassifier, _ := MockClassifier(t)
		mockReconciler, _ := MockReconciler(t)

		// When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
		call := StartRun(
			t,
			extract.Run,
			ctx,
			mockEnricher,
			mockClassifier,
			mockReconciler,
			nil,
			transcript,
		)

		// Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)
		// Given [learning1, learning2], nil error; When enricher.Enrich responds with ([learning1, learning2], nil)
		enricherExp.Enrich.ArgsShould(match.BeAny, Equal(transcript)).
			Return([]extract.RawLearning{learning1, learning2}, nil)

		// Then ExtractRun returns (string containing "rejected", any error)
		// And ExtractRun never calls classifier, store, or gate
		call.ReturnsShould(ContainSubstring("rejected"), Not(HaveOccurred()))
	})
}

func TestT18_PipelineEndToEnd(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript t, any RawLearning
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		learning := genRawLearning().Draw(rt, "learning")
		ctx := context.Background()

		mockEnricher, enricherExp := MockEnricher(t)
		mockClassifier, classifierExp := MockClassifier(t)
		mockReconciler, reconcilerExp := MockReconciler(t)

		// When test calls ExtractRun with (enricher, classifier, store, gate, nil, any ctx, t)
		call := StartRun(
			t,
			extract.Run,
			ctx,
			mockEnricher,
			mockClassifier,
			mockReconciler,
			nil,
			transcript,
		)

		// Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)
		// Given [learning], nil error; When enricher.Enrich responds with ([learning], nil)
		enricherExp.Enrich.ArgsShould(match.BeAny, Equal(transcript)).
			Return([]extract.RawLearning{learning}, nil)

		// Then ExtractRun calls classifier.Classify with (any ctx, learning, t)
		// Given "A", nil error; When classifier.Classify responds with ("A", nil)
		classifierExp.Classify.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
			Return("A", nil)
		// Then ExtractRun calls reconciler; Given "created", nil
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(extract.ReconcileResult{Action: "created"}, nil)

		// Then ExtractRun returns (non-empty string, nil error)
		call.ReturnsShould(Not(BeEmpty()), Not(HaveOccurred()))
	})
}

func genRawLearning() *rapid.Generator[extract.RawLearning] {
	return rapid.Custom(func(t *rapid.T) extract.RawLearning {
		// Generate content with at least 10 words
		words := make([]string, rapid.IntRange(10, 20).Draw(t, "wordCount"))
		for i := range words {
			words[i] = rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "word")
		}

		return extract.RawLearning{
			Content:  strings.Join(words, " "),
			Title:    rapid.StringMatching(`[A-Za-z ]{5,30}`).Draw(t, "title"),
			Keywords: []string{rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "keyword")},
		}
	})
}

func genShortLearning() *rapid.Generator[extract.RawLearning] {
	return rapid.Custom(func(t *rapid.T) extract.RawLearning {
		// Generate content with fewer than 10 tokens
		words := make([]string, rapid.IntRange(1, 9).Draw(t, "wordCount"))
		for i := range words {
			words[i] = rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "word")
		}

		return extract.RawLearning{
			Content:  strings.Join(words, " "),
			Title:    rapid.StringMatching(`[A-Za-z ]{5,30}`).Draw(t, "title"),
			Keywords: []string{rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "keyword")},
		}
	})
}
