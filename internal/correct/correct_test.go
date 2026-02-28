package correct_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"
	"pgregory.net/rapid"

	"engram/internal/corpus"
	"engram/internal/correct"
)

func TestT19_NoMatchReturnsEmpty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any message, nil patterns (empty corpus)
		message := rapid.StringMatching(`^[A-Z][a-z]+ [a-z]+ [a-z]+\.$`).Draw(rt, "message")
		ctx := context.Background()

		mockReconciler, _ := MockReconciler(t)
		// When test calls DetectCorrection with (store, gate, nil, any ctx, message)
		call := StartDetectCorrection(
			t,
			correct.DetectCorrection,
			ctx,
			mockReconciler,
			nil,
			message,
		)

		// Then returns ("", empty recordings, any string, nil error)
		call.ReturnsShould(BeEmpty(), BeEmpty(), match.BeAny, Not(HaveOccurred()))
	})
}

func TestT20_MatchTriggersReconciliation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Given patterns including `^no,`
	mockReconciler, reconcilerExp := MockReconciler(t)
	// When test calls DetectCorrection with (store, gate, patterns, any ctx, "no, use specific files not git add -A")
	call := StartDetectCorrection(
		t,
		correct.DetectCorrection,
		ctx,
		mockReconciler,
		defaultPatterns(),
		"no, use specific files not git add -A",
	)

	// Given Reconcile responds with (created, nil) — store.FindSimilar → store.Create path
	reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
		Return(correct.ReconcileResult{Action: "created", MemoryID: "m_0001"}, nil)

	// Then returns (non-empty reminder, 1 recording, non-empty audit, nil error)
	call.ReturnsShould(Not(BeEmpty()), HaveLen(1), Not(BeEmpty()), Not(HaveOccurred()))
}

func TestT22_CorrectionRecordedToSessionLog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Given patterns including `^no,`
	mockReconciler, reconcilerExp := MockReconciler(t)
	// When test calls DetectCorrection with (store, gate, patterns, any ctx, "no, that's not right")
	call := StartDetectCorrection(
		t,
		correct.DetectCorrection,
		ctx,
		mockReconciler,
		defaultPatterns(),
		"no, that's not right",
	)

	// Given Reconcile responds with (created, nil)
	reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
		Return(correct.ReconcileResult{Action: "created", MemoryID: "m_0002"}, nil)

	// Then returns (any string, recordings with len 1, any string, nil error)
	call.ReturnsShould(match.BeAny, HaveLen(1), match.BeAny, Not(HaveOccurred()))
}

func TestT23_EnrichedExistingMemoryReminderSaysEnriched(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Given patterns including `^no,`, existing Memory with Title "Use git add specific files"
	mockReconciler, reconcilerExp := MockReconciler(t)
	// When test calls DetectCorrection with (store, gate, patterns, any ctx, "no, don't use git add -A")
	call := StartDetectCorrection(
		t,
		correct.DetectCorrection,
		ctx,
		mockReconciler,
		defaultPatterns(),
		"no, don't use git add -A",
	)

	// Given Reconcile responds with (enriched, nil) — overlap path triggered
	reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
		Return(correct.ReconcileResult{Action: "enriched", MemoryID: "m_0001", Title: "Use git add specific files"}, nil)

	// Then reminder contains "Enriched:", "Use git add specific files", "Correction captured"
	call.ReturnsShould(
		And(
			ContainSubstring("Enriched:"),
			ContainSubstring("Use git add specific files"),
			ContainSubstring("Correction captured"),
		),
		match.BeAny,
		match.BeAny,
		Not(HaveOccurred()),
	)
}

func TestT24_CreatedNewMemoryReminderSaysCreated(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Given patterns including `^wait`
	mockReconciler, reconcilerExp := MockReconciler(t)
	// When test calls DetectCorrection with (store, gate, patterns, any ctx, "wait, this project uses bun not npm")
	call := StartDetectCorrection(
		t,
		correct.DetectCorrection,
		ctx,
		mockReconciler,
		defaultPatterns(),
		"wait, this project uses bun not npm",
	)

	// Given Reconcile responds with (created, nil) — no overlap, new memory
	reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
		Return(correct.ReconcileResult{Action: "created", MemoryID: "m_0003"}, nil)

	// Then returns (reminder containing "Created:", unchecked recordings, unchecked audit, nil error)
	call.ReturnsShould(ContainSubstring("Created:"), match.BeAny, match.BeAny, Not(HaveOccurred()))
}

func TestT25_FalsePositiveCapturedAnyway(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Given patterns including `\bremember\s+(that|to)` — false positive per DES-5
	mockReconciler, reconcilerExp := MockReconciler(t)
	// When test calls DetectCorrection with (store, gate, patterns, any ctx, "remember to run tests before committing")
	call := StartDetectCorrection(
		t,
		correct.DetectCorrection,
		ctx,
		mockReconciler,
		defaultPatterns(),
		"remember to run tests before committing",
	)

	// Given Reconcile responds with (created, nil) — captured without confirmation
	reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
		Return(correct.ReconcileResult{Action: "created", MemoryID: "m_0004"}, nil)

	// Then returns (non-empty reminder, unchecked recordings, unchecked audit, nil error)
	call.ReturnsShould(Not(BeEmpty()), match.BeAny, match.BeAny, Not(HaveOccurred()))
}

func TestT26_EndToEndCorrectionDetection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Given patterns including `^no,`
	mockReconciler, reconcilerExp := MockReconciler(t)
	// When test calls DetectCorrection with (store, gate, patterns, any ctx, "no, use targ test not go test")
	call := StartDetectCorrection(
		t,
		correct.DetectCorrection,
		ctx,
		mockReconciler,
		defaultPatterns(),
		"no, use targ test not go test",
	)

	// Given Reconcile responds with (created, nil)
	reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
		Return(correct.ReconcileResult{Action: "created", MemoryID: "m_0005"}, nil)

	// Then returns (non-empty reminder, 1 recording, non-empty audit, nil error)
	call.ReturnsShould(Not(BeEmpty()), HaveLen(1), Not(BeEmpty()), Not(HaveOccurred()))
}

func TestT27_LongMessageTitleTruncated(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockReconciler, reconcilerExp := MockReconciler(t)
	call := StartDetectCorrection(
		t,
		correct.DetectCorrection,
		ctx,
		mockReconciler,
		defaultPatterns(),
		"no, you should use specific file paths instead of git add dash capital A for staging",
	)

	reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
		Return(correct.ReconcileResult{Action: "created", MemoryID: "m_0006"}, nil)

	call.ReturnsShould(Not(BeEmpty()), HaveLen(1), Not(BeEmpty()), Not(HaveOccurred()))
}

func defaultPatterns() *corpus.Corpus {
	return corpus.New(corpus.DefaultPatterns())
}
