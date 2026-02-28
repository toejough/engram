package catchup_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"
	"pgregory.net/rapid"

	"engram/internal/catchup"
)

func TestT32_NoMissedCorrectionsNoNewMemories(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript, empty capturedEvents
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		ctx := context.Background()

		mockEval, evalExp := MockEvaluator(t)
		mockReconciler, _ := MockReconciler(t)

		// When CatchupRun is called
		call := StartRun(t, catchup.Run, ctx, mockEval, mockReconciler, nil, transcript)

		// Then evaluator.FindMissed called; Given nil missed, nil error
		evalExp.FindMissed.ArgsShould(match.BeAny, Equal(transcript), match.BeAny).
			Return(nil, nil)

		// Then CatchupRun returns empty candidates, nil error
		call.ReturnsShould(BeEmpty(), match.BeAny, Not(HaveOccurred()))
	})
}

func TestT33_MissedCorrectionReconciled(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		ctx := context.Background()
		missedCorrection := catchup.MissedCorrection{
			Content: rapid.StringMatching(`[A-Za-z ]{10,50}`).Draw(rt, "content"),
			Context: rapid.StringMatching(`[A-Za-z ]{5,30}`).Draw(rt, "context"),
			Phrase:  rapid.StringMatching(`\\b[a-z]+\\b`).Draw(rt, "phrase"),
		}

		mockEval, evalExp := MockEvaluator(t)
		mockReconciler, reconcilerExp := MockReconciler(t)

		// When CatchupRun is called
		call := StartRun(t, catchup.Run, ctx, mockEval, mockReconciler, nil, transcript)

		// Then evaluator.FindMissed called; Given one MissedCorrection, nil error
		evalExp.FindMissed.ArgsShould(match.BeAny, Equal(transcript), match.BeAny).
			Return([]catchup.MissedCorrection{missedCorrection}, nil)

		// Then reconciler.Reconcile called; Given "created" result, nil error
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(catchup.ReconcileResult{Action: "created", MemoryID: "m_0001"}, nil)

		// Then CatchupRun returns non-empty candidates, non-empty audit, nil error
		call.ReturnsShould(Not(BeEmpty()), Not(BeEmpty()), Not(HaveOccurred()))
	})
}

func TestT34_NewPatternAddedAsCandidate(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any transcript, any phrase
		transcript := []byte(rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(rt, "transcript"))
		phrase := rapid.StringMatching(`\\b[a-z]+\\b`).Draw(rt, "phrase")
		ctx := context.Background()

		missedCorrection := catchup.MissedCorrection{
			Content: "missed correction",
			Context: "context",
			Phrase:  phrase,
		}

		mockEval, evalExp := MockEvaluator(t)
		mockReconciler, reconcilerExp := MockReconciler(t)

		// When CatchupRun is called
		call := StartRun(t, catchup.Run, ctx, mockEval, mockReconciler, nil, transcript)

		// Then evaluator.FindMissed called; Given [{missed correction, context, phrase}], nil error
		evalExp.FindMissed.ArgsShould(match.BeAny, Equal(transcript), match.BeAny).
			Return([]catchup.MissedCorrection{missedCorrection}, nil)

		// Then reconciler.Reconcile called; Given "created" result, nil error
		reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
			Return(catchup.ReconcileResult{Action: "created"}, nil)

		// Then candidates has len 1, candidates[0].Regex equals phrase, nil error
		call.ReturnsShould(
			And(HaveLen(1), ContainElement(HaveField("Regex", Equal(phrase)))),
			match.BeAny,
			Not(HaveOccurred()),
		)
	})
}

func TestT35_FullScenarioMissedCorrectionMemoryCandidateAudit(t *testing.T) {
	t.Parallel()

	// Given any transcript, capturedEvents = [{m_other, ^no,, "no, use bun"}]
	ctx := context.Background()
	transcript := []byte("session transcript with orphaned teammates discussion")
	captured := []catchup.CapturedEvent{
		{MemoryID: "m_other", Pattern: `^no,`, Message: "no, use bun"},
	}

	mockEval, evalExp := MockEvaluator(t)
	mockReconciler, reconcilerExp := MockReconciler(t)

	// When CatchupRun is called with captured events
	call := StartRun(t, catchup.Run, ctx, mockEval, mockReconciler, captured, transcript)

	// Then evaluator.FindMissed called with transcript and capturedEvents; Given missed correction, nil error
	evalExp.FindMissed.ArgsShould(match.BeAny, Equal(transcript), Equal(captured)).
		Return([]catchup.MissedCorrection{
			{
				Content: "you didn't shut them down",
				Context: "orphaned teammates",
				Phrase:  `\byou didn't\b`,
			},
		}, nil)

	// Then reconciler.Reconcile called; Given "created" result, nil error
	reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
		Return(catchup.ReconcileResult{Action: "created", MemoryID: "m_0010"}, nil)

	// Then candidates[0].Regex equals `\byou didn't\b`, audit non-empty, nil error
	call.ReturnsShould(
		And(HaveLen(1), ContainElement(HaveField("Regex", Equal(`\byou didn't\b`)))),
		Not(BeEmpty()),
		Not(HaveOccurred()),
	)
}

func TestT36_LongContentTruncatedInTitle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	transcript := []byte("session transcript")
	longContent := "one two three four five six seven eight nine ten eleven twelve"

	mockEval, evalExp := MockEvaluator(t)
	mockReconciler, reconcilerExp := MockReconciler(t)

	call := StartRun(t, catchup.Run, ctx, mockEval, mockReconciler, nil, transcript)

	evalExp.FindMissed.ArgsShould(match.BeAny, Equal(transcript), match.BeAny).
		Return([]catchup.MissedCorrection{
			{Content: longContent, Context: "ctx", Phrase: "pattern"},
		}, nil)

	reconcilerExp.Reconcile.ArgsShould(match.BeAny, match.BeAny).
		Return(catchup.ReconcileResult{Action: "created"}, nil)

	call.ReturnsShould(HaveLen(1), Not(BeEmpty()), Not(HaveOccurred()))
}
