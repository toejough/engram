//go:build targ && integration

package dev

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a19a1a_AProposedApproachIsRecalledBeforeItIsEndorsed reads the recall guidance the repository ships: a
// proposed approach is recalled with glance before it is endorsed or ranked, and the skill draws the phrases from
// the situation.
func Test792a19a1a_AProposedApproachIsRecalledBeforeItIsEndorsed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1RecallGuidance)

	g.Expect(text).To(ContainSubstring("**run `/recall glance` before you proceed** at these cues:"))
	g.Expect(text).To(ContainSubstring("**Even just chatting about a proposed approach** — before you endorse or " +
		"rank it — **`/recall glance` first** (the action is recalling, not judging what's in front of you)"))

	skill := shippedText(t, ag1RecallSkill)

	g.Expect(skill).To(ContainSubstring("### Step 1 — Phrase queries from your plan and situation"))
}

// Test792a19a1b_WorkIsRecalledBeforeItIsDeclaredDone reads the recall guidance the repository ships: glance runs
// before work is declared done, then the agent verifies.
func Test792a19a1b_WorkIsRecalledBeforeItIsDeclaredDone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1RecallGuidance)

	g.Expect(text).To(ContainSubstring("**Before declaring work done** — **`/recall glance` first** (the action is " +
		"recalling — not a substitute self-check or re-inspection), then verify;"))
}

// Test792a19a1c_AnUnexplainedFailureIsRecalledOnce reads the recall guidance the repository ships: glance runs once
// before guessing at an unexplained failure and not on every retry.
func Test792a19a1c_AnUnexplainedFailureIsRecalledOnce(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1RecallGuidance)

	g.Expect(text).To(ContainSubstring("**After a failure you can't immediately explain** — " +
		"**`/recall glance` once before you start guessing** (not on every retry)"))
}

// Test792a19a1d_ANewApproachIsRecalledBeforeItIsBuilt reads the recall guidance the repository ships: glance runs for
// prior decisions and standards before a new approach is built.
func Test792a19a1d_ANewApproachIsRecalledBeforeItIsBuilt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1RecallGuidance)

	g.Expect(text).To(ContainSubstring("**Before you start building a new approach** — " +
		"**`/recall glance` prior decisions and standards first**, while the path is still cheap to change."))
}

// Test792a19b1a_AWeightyDecisionEscalates reads the recall guidance the repository ships: a weighty or irreversible
// decision escalates to deep.
func Test792a19b1a_AWeightyDecisionEscalates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1RecallGuidance)

	g.Expect(text).To(ContainSubstring(ag1EscalateSentence))
	g.Expect(text).To(ContainSubstring("when the decision is weighty or irreversible"))
}

// Test792a19b1b_AnUnresolvedGapEscalates reads the recall guidance the repository ships: a glance that flags a gap
// the agent cannot resolve escalates to deep.
func Test792a19b1b_AnUnresolvedGapEscalates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1RecallGuidance)

	g.Expect(text).To(ContainSubstring(ag1EscalateSentence))
	g.Expect(text).To(ContainSubstring("or when `glance` flags a gap it can't resolve."))
}

// Test792a19c1a_ARejectedApproachIsCapturedBeforeTheFix reads the learn guidance the repository ships: a rejected
// approach is captured with the learn skill before the fix is applied.
func Test792a19c1a_ARejectedApproachIsCapturedBeforeTheFix(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("**A review or the user rejected your approach** — \"no, do X not Y\", " +
		"\"the convention here is Z\", \"that's wrong because W\" — `/learn` the confirmed correction " +
		"**before you apply the fix**."))
	g.Expect(text).To(ContainSubstring("It is: **invoke the engram `/learn` skill to crystallize the confirmed " +
		"correction, then fix.**"))
}

// Test792a19c1b_AFailingCheckWithAKnownCauseIsCaptured reads the learn guidance the repository ships: a failed check
// whose non-obvious cause is now known is captured with the learn skill.
func Test792a19c1b_AFailingCheckWithAKnownCauseIsCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("**A build/test/check failed and revealed a non-obvious cause** — " +
		"once you actually know the cause, `/learn` it so the next session doesn't re-hit it."))
}

// Test792a19c1c_ASelfDiscoveredReversalIsCaptured reads the learn guidance the repository ships: the root cause of a
// self-discovered reversal is captured at the moment it is caught.
func Test792a19c1c_ASelfDiscoveredReversalIsCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("**You caught your own approach being wrong mid-task** — " +
		"a self-discovered reversal is the same cue; `/learn` the root cause at the moment you catch it."))
}

// Test792a19c1d_AOneLineFixStillEarnsTheNote reads the learn guidance the repository ships: a one-line fix is not
// below the capture threshold.
func Test792a19c1d_AOneLineFixStillEarnsTheNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("**Even a one-line fix earns the note.** Small is not below the capture " +
		"threshold — the value is in the *rule you were just corrected on*, which is identical whether the fix is " +
		"one line or ten."))
}

// Test792a19d1a_TheLearnSkillIsInvokedNotANativeMemoryWrite reads the learn guidance the repository ships: the
// capture is the learn skill, which writes a vault note, and not a plain file or the harness's own memory.
func Test792a19d1a_TheLearnSkillIsInvokedNotANativeMemoryWrite(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("**Fire the engram `/learn` skill specifically** — the `Skill` tool with " +
		"`skill=learn`, which writes a vault note (via `engram learn`). NOT a plain note file, NOT Claude Code's " +
		"native project memory (`memory/…`, `MEMORY.md`) — those are a different store engram can't recall from."))
}

// Test792a19d1b_NoSweepRunsMidTask reads the learn guidance the repository ships: the capture is the one lesson on
// the fast path and runs no ingest sweep mid-task.
func Test792a19d1b_NoSweepRunsMidTask(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("crystallize the one confirmed lesson now (the `/learn` skill's mid-cycle " +
		"fast path — skip its Step 1 sweep, go straight to the crystallize step)."))
	g.Expect(text).To(ContainSubstring("Do NOT run `engram ingest --auto` mid-task"))
}

// Test792a19d1c_ARejectionIsCapturedWithoutTheRightAnswer reads the learn guidance the repository ships: the
// confirmed negative is captured on its own, without a guess at the rule.
func Test792a19d1c_ARejectionIsCapturedWithoutTheRightAnswer(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("Capture the **confirmed negative**, not a guess:"))
	g.Expect(text).To(ContainSubstring("You know what was REJECTED with confidence long before the full right " +
		"answer; the rejection is the safely-capturable half, and it is enough."))
}

// Test792a19d1d_AHunchIsNotCaptured reads the learn guidance the repository ships: an unconfirmed hunch is not
// crystallized.
func Test792a19d1d_AHunchIsNotCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("Do NOT crystallize an unconfirmed hunch " +
		"(\"maybe it's X\") — that rots the vault."))
}

// Test792a19e1a_AConfirmedCorrectionIsCapturedBeforeTheNextUnit reads the learn guidance the repository ships: a
// returned lesson that is a confirmed correction is captured on the fast path, with no sweep, before the next
// dispatch.
func Test792a19e1a_AConfirmedCorrectionIsCapturedBeforeTheNextUnit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring(ag1ReturnedBulletStart))
	g.Expect(text).To(ContainSubstring("if it is a confirmed correction, reversal, explicit save-request, or " +
		"confirmed approach that states a reusable rule, `/learn` it on the fast path — one note per lesson — " +
		"**before your next dispatch**"))
	g.Expect(text).To(ContainSubstring("Do NOT run `engram ingest --auto` mid-task"))
}

// Test792a19e1b_AConfirmedApproachIsCapturedAsSuch reads the learn guidance the repository ships: a confirmed
// approach is one of the kinds captured on the fast path before the next dispatch.
func Test792a19e1b_AConfirmedApproachIsCapturedAsSuch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("a confirmed correction, reversal, explicit save-request, or confirmed " +
		"approach that states a reusable rule, `/learn` it on the fast path"))
	g.Expect(text).To(ContainSubstring("**before your next dispatch**"))
}

// Test792a19e1c_TwoReturnsGiveTwoNotes reads the learn guidance the repository ships: each lesson is judged as the
// report is read and gets one note.
func Test792a19e1c_TwoReturnsGiveTwoNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("Judge each lesson as you read the report:"))
	g.Expect(text).To(ContainSubstring("`/learn` it on the fast path — one note per lesson — " +
		"**before your next dispatch**"))
}

// Test792a19e1d_AnAdHocDispatchCarriesTheCue reads the learn guidance the repository ships: the cue lives in the
// ambient guidance and its condition is a subagent return, not a route or please skill.
func Test792a19e1d_AnAdHocDispatchCarriesTheCue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring(ag1ReturnedBulletStart))

	bullet := ag1Between(text, ag1ReturnedBulletStart, ag1ReturnedBulletEnd)

	g.Expect(bullet).NotTo(BeEmpty())
	g.Expect(bullet).NotTo(ContainSubstring("route"))
	g.Expect(bullet).NotTo(ContainSubstring("please"))
	g.Expect(bullet).NotTo(ContainSubstring("skill mediat"))
}

// Test792a19e1e_TheLastReturnIsCapturedBeforeTheTurnEnds reads the learn guidance the repository ships: with no
// dispatch left, the lesson is captured before the turn ends.
func Test792a19e1e_TheLastReturnIsCapturedBeforeTheTurnEnds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("**before your next dispatch** (or before you end your turn, " +
		"if no dispatch is left)."))
}

// Test792a19f1a_LessonsNoneIsSkipped reads the learn guidance the repository ships: a LESSONS none line is skipped.
func Test792a19f1a_LessonsNoneIsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring("its report's `LESSONS:` line isn't `none`"))
	g.Expect(text).To(ContainSubstring(ag1SkipSentence))
}

// Test792a19f1b_ARoutineNarrativeIsSkipped reads the learn guidance the repository ships: a bare success line such as
// "done, tests pass" is skipped.
func Test792a19f1b_ARoutineNarrativeIsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring(ag1SkipSentence))
	g.Expect(text).To(ContainSubstring("Skip `none`, \"done, tests pass\","))
}

// Test792a19f1c_AnUnconfirmedHunchIsSkipped reads the learn guidance the repository ships: a lesson the report does
// not confirm is skipped.
func Test792a19f1c_AnUnconfirmedHunchIsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag1LearnGuidance)

	g.Expect(text).To(ContainSubstring(ag1SkipSentence))
	g.Expect(text).To(ContainSubstring("anything the report doesn't confirm (\"might be a race\")"))
}

// unexported constants.
const (
	ag1EscalateSentence = "Escalate to **`/recall deep`** — the thorough rung, which also crystallizes new lessons — " +
		"when the decision is weighty or irreversible, or when `glance` flags a gap it can't resolve."
	ag1LearnGuidance       = "agent-instructions/guidance/learn.md"
	ag1RecallGuidance      = "agent-instructions/guidance/recall.md"
	ag1RecallSkill         = "agent-instructions/skills/recall/SKILL.md"
	ag1ReturnedBulletEnd   = "**Even a one-line fix earns the note.**"
	ag1ReturnedBulletStart = "**A subagent just returned, and its report's `LESSONS:` line isn't `none`**"
	ag1SkipSentence        = "Skip `none`, \"done, tests pass\", and anything the report doesn't confirm " +
		"(\"might be a race\") — those are not lessons, and the closing learn will discard them anyway."
)

// ag1Between returns the text from the first occurrence of start up to the next occurrence of end, or empty when
// either marker is absent.
func ag1Between(text, start, end string) string {
	_, rest, found := strings.Cut(text, start)
	if !found {
		return ""
	}

	section, _, found := strings.Cut(rest, end)
	if !found {
		return ""
	}

	return section
}
