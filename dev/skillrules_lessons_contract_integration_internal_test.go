//go:build targ && integration

package dev

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a16a1a_LessonsContract_ASubagentWithLessonsReportsThem reads the route skill the repository ships: the
// completion report ends with a LESSONS line of one to three comma-separated one-line lessons, and a confirmed
// correction and a surprising finding are among the kinds of lesson.
func Test792a16a1a_LessonsContract_ASubagentWithLessonsReportsThem(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	contract := lessonsContractRouteContract(t)

	g.Expect(contract).To(ContainSubstring(lessonsContractEndsWithLine))
	g.Expect(contract).To(ContainSubstring("1–3 comma-separated one-line lessons"))
	g.Expect(contract).To(ContainSubstring("confirmed corrections"))
	g.Expect(contract).To(ContainSubstring("surprising findings"))
}

// Test792a16a1b_LessonsContract_ASubagentWithNothingToReportSaysSo reads the route skill the repository ships: the
// LESSONS line is either `LESSONS: none` or the lessons.
func Test792a16a1b_LessonsContract_ASubagentWithNothingToReportSaysSo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	contract := lessonsContractRouteContract(t)

	g.Expect(contract).To(ContainSubstring(lessonsContractEndsWithLine + " — `LESSONS: none`, or"))
}

// Test792a16a1c_LessonsContract_AtMostThreeLessons reads the route skill the repository ships: the line carries one
// to three one-line lessons and no more.
func Test792a16a1c_LessonsContract_AtMostThreeLessons(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	contract := lessonsContractRouteContract(t)

	g.Expect(contract).To(ContainSubstring("`LESSONS: none`, or 1–3 comma-separated one-line lessons"))
}

// Test792a16b1a_LessonsContract_AFreshContextDispatchCarriesTheMandate reads the route skill the repository ships:
// every dispatch hands the subagent the completion-report contract, which carries the LESSONS line.
func Test792a16b1a_LessonsContract_AFreshContextDispatchCarriesTheMandate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, lessonsContractRoutePath)

	g.Expect(text).To(ContainSubstring("Every dispatch MUST hand the subagent:"))
	g.Expect(text).To(ContainSubstring(lessonsContractContractStart + " " + lessonsContractEndsWithLine))
	g.Expect(lessonsContractBeforeContract(text, "Every dispatch MUST hand the subagent:")).
		To(BeTrue())
}

// Test792a16b1b_LessonsContract_AForksHandoffCarriesItToo reads the route skill the repository ships: the rule is for
// every dispatch with no kind of dispatch exempt, and it is a MUST. No shipped instruction exempts a dispatch from
// the contract.
func Test792a16b1b_LessonsContract_AForksHandoffCarriesItToo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, lessonsContractRoutePath)

	g.Expect(text).To(ContainSubstring("Every dispatch MUST hand the subagent:"))
	g.Expect(lessonsContractRouteContract(t)).To(ContainSubstring(lessonsContractEndsWithLine))
	g.Expect(lessonsContractContradictions(t, lessonsContractExemptionScan)).To(BeEmpty())
}

// Test792a16b1c_LessonsContract_ItFollowsTheTaskDescription reads the route skill the repository ships: the contract
// is the last item of the handoff, after the exact files and the acceptance checks that describe the task.
func Test792a16b1c_LessonsContract_ItFollowsTheTaskDescription(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, lessonsContractRoutePath)

	g.Expect(lessonsContractBeforeContract(text, "- **Exact files and paths**")).To(BeTrue())
	g.Expect(lessonsContractBeforeContract(text, "- **Acceptance checks**")).To(BeTrue())
	g.Expect(lessonsContractBeforeContract(text, "- **Explicit do-NOT-touch bounds**")).
		To(BeTrue())
}

// Test792a16c1a_LessonsContract_AMissingLineIsReAsked reads the route skill the repository ships: a returned report
// without the line gets one targeted follow-up asking for it, in the format of the contract.
func Test792a16c1a_LessonsContract_AMissingLineIsReAsked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	contract := lessonsContractRouteContract(t)

	g.Expect(contract).To(ContainSubstring("If a returned report is missing the line, send ONE targeted follow-up " +
		"asking for it"))
	g.Expect(contract).To(ContainSubstring("`LESSONS: none`, or 1–3 comma-separated one-line lessons"))
}

// Test792a16c1c_LessonsContract_ThereIsNoSecondReAsk reads the route skill the repository ships: a line still
// missing after the one follow-up is recorded as `LESSONS: none`, the pipeline moves on, and no second re-ask is
// asked for. No shipped instruction asks again.
func Test792a16c1c_LessonsContract_ThereIsNoSecondReAsk(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	contract := lessonsContractRouteContract(t)

	g.Expect(contract).To(ContainSubstring("never a second re-ask"))
	g.Expect(contract).To(ContainSubstring("if it's still missing after that one follow-up, record `LESSONS: none`"))
	g.Expect(contract).To(ContainSubstring("move on rather than blocking the pipeline"))
	g.Expect(lessonsContractContradictions(t, lessonsContractReaskScan)).To(BeEmpty())
}

// Test792a16d1a_LessonsContract_LinesAreKeptInOrderAndVerbatim reads the please skill the repository ships: the moment
// a unit's result comes back, before the next unit, its line is recorded verbatim in a running list.
func Test792a16d1a_LessonsContract_LinesAreKeptInOrderAndVerbatim(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, lessonsContractPleasePath)

	g.Expect(text).To(ContainSubstring("Every dispatched unit's completion report ends with a `LESSONS:` line"))
	g.Expect(text).To(ContainSubstring("The moment a unit's result comes back — before moving to the next unit — " +
		"record that line verbatim in a running list for this cycle"))
}

// Test792a16d1b_LessonsContract_ANoneIsStillRecorded reads the please skill and the learn guidance the repository
// ships: a `LESSONS: none` line is an entry of the list, and the guidance skips it, so no learn fires for it. No
// shipped instruction captures a none line.
func Test792a16d1b_LessonsContract_ANoneIsStillRecorded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(shippedText(t, lessonsContractPleasePath)).To(ContainSubstring("including `LESSONS: none` entries"))

	guidance := shippedText(t, lessonsContractGuidancePath)

	g.Expect(guidance).To(ContainSubstring("its report's `LESSONS:` line isn't `none`"))
	g.Expect(guidance).To(ContainSubstring("Skip `none`,"))
	g.Expect(lessonsContractContradictions(t, lessonsContractNoneScan)).To(BeEmpty())
}

// Test792a16d1c_LessonsContract_AGateDoesNotFilterTheList reads the please skill the repository ships: every
// dispatched unit's line is recorded the moment its result comes back, with no condition on a gate, and step 7 drops
// no line. No shipped instruction drops the line of a rejected unit.
func Test792a16d1c_LessonsContract_AGateDoesNotFilterTheList(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, lessonsContractPleasePath)

	g.Expect(text).To(ContainSubstring("Every dispatched unit's completion report ends with a `LESSONS:` line"))
	g.Expect(text).To(ContainSubstring("The moment a unit's result comes back — before moving to the next unit — " +
		"record that line verbatim"))
	g.Expect(text).To(ContainSubstring("step 7 drops no line"))
	g.Expect(lessonsContractContradictions(t, lessonsContractGateDropScan)).To(BeEmpty())
}

// Test792a16e1a_LessonsContract_AConfirmedCorrectionIsCapturedAtTheReturn reads the learn guidance and the learn skill
// the repository ships: a lesson of a returned line that is a confirmed correction is learned on the fast path, one
// note per lesson, before the next dispatch.
func Test792a16e1a_LessonsContract_AConfirmedCorrectionIsCapturedAtTheReturn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	guidance := shippedText(t, lessonsContractGuidancePath)

	g.Expect(guidance).To(ContainSubstring("Judge each lesson as you read the report: if it is a confirmed correction, " +
		"reversal, explicit save-request, or confirmed approach that states a reusable rule, `/learn` it on the " +
		"fast path — one note per lesson — **before your next dispatch**"))
	g.Expect(shippedText(t, lessonsContractLearnPath)).To(ContainSubstring("one note per lesson that clears the " +
		"bar, no sweep."))
}

// Test792a16e1b_LessonsContract_NoneAndBareSuccessAreLeftAlone reads the learn guidance the repository ships: a none
// line and a routine "done, tests pass" are skipped. No shipped instruction captures either.
func Test792a16e1b_LessonsContract_NoneAndBareSuccessAreLeftAlone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	guidance := shippedText(t, lessonsContractGuidancePath)

	g.Expect(guidance).To(ContainSubstring("Skip `none`,"))
	g.Expect(guidance).To(ContainSubstring("\"done, tests pass\""))
	g.Expect(guidance).To(ContainSubstring("those are not lessons"))
	g.Expect(lessonsContractContradictions(t, lessonsContractNoneScan)).To(BeEmpty())
	g.Expect(lessonsContractContradictions(t, lessonsContractSuccessNoteScan)).To(BeEmpty())
}

// Test792a16e1c_LessonsContract_AnUnconfirmedLessonIsLeftAlone reads the learn guidance and the learn skill the
// repository ships: anything the report does not confirm is skipped, and a lesson must be confirmed and not
// hypothesized. No shipped instruction captures an unconfirmed lesson.
func Test792a16e1c_LessonsContract_AnUnconfirmedLessonIsLeftAlone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(shippedText(t, lessonsContractGuidancePath)).To(ContainSubstring("anything the report doesn't confirm " +
		"(\"might be a race\")"))
	g.Expect(shippedText(t, lessonsContractLearnPath)).To(ContainSubstring("(confirmed, not hypothesized; a general " +
		"reusable principle, not a session-specific narrative)"))
	g.Expect(lessonsContractContradictions(t, lessonsContractUnconfirmedScan)).To(BeEmpty())
}

// Test792a16e1d_LessonsContract_TheLastReturnIsCapturedBeforeTheTurnEnds reads the learn guidance the repository
// ships: the capture is before the next dispatch, or before the turn ends when no dispatch is left.
func Test792a16e1d_LessonsContract_TheLastReturnIsCapturedBeforeTheTurnEnds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	guidance := shippedText(t, lessonsContractGuidancePath)

	g.Expect(guidance).To(ContainSubstring("**before your next dispatch** (or before you end your turn, if no " +
		"dispatch is left)"))
}

// Test792a16f1a_LessonsContract_TheClosingLearnReceivesTheList reads the please skill and the learn skill the
// repository ships: the closing learn is handed the full list of LESSONS lines, none entries included, and step 7
// drops no line.
func Test792a16f1a_LessonsContract_TheClosingLearnReceivesTheList(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, lessonsContractPleasePath)

	g.Expect(text).To(ContainSubstring("handing it the full list of `LESSONS:` lines collected across step 4's " +
		"dispatches (including any `none` entries) alongside the lessons-audit map above"))
	g.Expect(text).To(ContainSubstring("step 7 drops no line"))
	g.Expect(shippedText(t, lessonsContractLearnPath)).To(ContainSubstring("every line, `none` entries included"))
}

// Test792a16f1b_LessonsContract_NoDispatchEmptyCollection reads the please skill and the learn skill the repository
// ships: the closing learn is not skipped for a session with nothing to document and is handed the lines collected
// across the dispatches, and the learn skill crystallizes only the lessons that exist.
func Test792a16f1b_LessonsContract_NoDispatchEmptyCollection(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, lessonsContractPleasePath)

	g.Expect(text).To(ContainSubstring("A step is N/A only when the *mechanism* doesn't exist in this environment"))
	g.Expect(text).To(ContainSubstring("or step 7's closing `/learn` when the engram binary itself is absent"))
	g.Expect(text).To(ContainSubstring("\"nothing to document\""))
	g.Expect(text).To(ContainSubstring("run the step; if it produces nothing, that's fine"))
	g.Expect(text).To(ContainSubstring("the full list of `LESSONS:` lines collected across step 4's dispatches"))
	g.Expect(shippedText(t, lessonsContractLearnPath)).To(ContainSubstring("## Step 2 — Crystallize explicit " +
		"lessons (only when they exist)"))
}

// Test792a16f1c_LessonsContract_ALineCapturedAtItsReturnIsSkipped reads the please skill and the learn skill the
// repository ships: a line a note written earlier this session covers is skipped and never written a second time,
// and every other line is judged. No shipped instruction writes a captured line again.
func Test792a16f1c_LessonsContract_ALineCapturedAtItsReturnIsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	learn := shippedText(t, lessonsContractLearnPath)

	g.Expect(learn).To(ContainSubstring("A line is **captured** when a note written earlier this session covers it: " +
		"skip captured lines — never write one a second time."))
	g.Expect(learn).To(ContainSubstring("Judge every other line against the same four kinds and the same quality bar " +
		"below"))
	g.Expect(shippedText(t, lessonsContractPleasePath)).To(ContainSubstring("the closing `/learn` skips any lesson a " +
		"vault note written earlier this session already covers"))
	g.Expect(lessonsContractContradictions(t, lessonsContractRewriteScan)).To(BeEmpty())
}

// Test792a16f1d_LessonsContract_AMissedLineIsCaught reads the learn skill and the please skill the repository ships:
// the closing learn scans every collected line, judges each line no note covers against the bar, and crystallizes
// the lessons that clear it.
func Test792a16f1d_LessonsContract_AMissedLineIsCaught(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	learn := shippedText(t, lessonsContractLearnPath)

	g.Expect(learn).To(ContainSubstring("At the closing learn, scan the session's collected `LESSONS:` lines from " +
		"dispatched work"))
	g.Expect(learn).To(ContainSubstring("Judge every other line against the same four kinds and the same quality bar " +
		"below"))
	g.Expect(learn).To(ContainSubstring("## Step 2 — Crystallize explicit lessons (only when they exist)"))
	g.Expect(shippedText(t, lessonsContractPleasePath)).To(ContainSubstring("and curates the rest against its own " +
		"quality bar"))
}

// Test792a16f1e_LessonsContract_AMixedBatch reads the learn skill the repository ships: a line is judged against
// exactly four kinds, a correction and a confirmed approach are two of them, a routine success is not a signal, and
// a line that does not clear the bar is discarded silently, with no note and no log. No shipped instruction logs a
// discard.
func Test792a16f1e_LessonsContract_AMixedBatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	learn := shippedText(t, lessonsContractLearnPath)

	g.Expect(learn).To(ContainSubstring("Scan THIS session for exactly four kinds of moments."))
	g.Expect(learn).To(ContainSubstring("1. **Corrections** — the user corrected your approach or behavior"))
	g.Expect(learn).To(ContainSubstring("4. **Confirmed approaches (positive reinforcement)**"))
	g.Expect(learn).To(ContainSubstring("a routine success does not count as reinforcement-capture"))
	g.Expect(learn).To(ContainSubstring("(confirmed, not hypothesized; a general reusable principle, not a " +
		"session-specific narrative)"))
	g.Expect(learn).To(ContainSubstring("A line that doesn't clear the bar is discarded silently — no vault note, no " +
		"logged discard"))
	g.Expect(lessonsContractContradictions(t, lessonsContractDiscardLogScan)).To(BeEmpty())
}

// Test792a16g1a_LessonsContract_ACompletionAnchorsTheScan reads the learn skill the repository ships: the scan for a
// confirmed approach is anchored on the moment a unit of work's outcome is confirmed, which includes reading a
// dispatched subagent's completion report and its LESSONS line.
func Test792a16g1a_LessonsContract_ACompletionAnchorsTheScan(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	learn := shippedText(t, lessonsContractLearnPath)

	g.Expect(learn).To(ContainSubstring("**Completion-moment anchor:** scan for kind-4 specifically at the moment a " +
		"unit of work's outcome is CONFIRMED — a review or gate verdict lands, a check or test passes, or the user " +
		"explicitly confirms something."))
	g.Expect(learn).To(ContainSubstring("That includes the moment you read a dispatched subagent's completion report " +
		"(and its `LESSONS:` line) once its outcome is known"))
	g.Expect(learn).To(ContainSubstring("Anchor on that concrete event, not on \"did the session feel successful " +
		"overall.\""))
}

// Test792a16g1b_LessonsContract_AnExemplarShapedLineIsRecognized reads the learn skill the repository ships: a defect
// confirmed by a targeted reproduction test is an exemplar of a confirmed approach.
func Test792a16g1b_LessonsContract_AnExemplarShapedLineIsRecognized(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	learn := shippedText(t, lessonsContractLearnPath)

	g.Expect(learn).To(ContainSubstring("**Real-world exemplars** (kind-4 under-fires more than any other kind in " +
		"practice"))
	g.Expect(learn).To(ContainSubstring("Suspected a defect existed; wrote and ran a targeted reproduction test that " +
		"confirmed it → confirmed approach (the targeted-reproduction method), not \"just found a bug.\""))
}

// Test792a16g1c_LessonsContract_ABareSuccessIsStillNotKind4 reads the learn skill the repository ships: a routine
// success is never the signal of a confirmed approach. No shipped instruction counts a bare success as one.
func Test792a16g1c_LessonsContract_ABareSuccessIsStillNotKind4(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	learn := shippedText(t, lessonsContractLearnPath)

	g.Expect(learn).To(ContainSubstring("a routine success does not count as reinforcement-capture; the signal is a " +
		"resolved uncertainty or an explicit specific confirmation, never \"it worked\"."))
	g.Expect(learn).To(ContainSubstring("A bare pleasantry, a routine success with no bet behind it, or an " +
		"unconfirmed guess is never the signal"))
	g.Expect(lessonsContractContradictions(t, lessonsContractSuccessKindScan)).To(BeEmpty())
}

// TestLessonsContractHits supports the absence scans of this file: each scan finds a contradicting instruction in
// three forms, a paraphrase, an imperative and the imperative under a negated condition, and passes the shipped
// wording that states the rule.
func TestLessonsContractHits(t *testing.T) {
	t.Parallel()

	for _, scan := range lessonsContractScans() {
		t.Run(scan.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(lessonsContractHits(scan.paraphrase, scan)).NotTo(BeEmpty())
			g.Expect(lessonsContractHits(scan.imperative, scan)).NotTo(BeEmpty())
			g.Expect(lessonsContractHits(scan.conditional, scan)).NotTo(BeEmpty())
			g.Expect(lessonsContractHits(scan.stated, scan)).To(BeEmpty())
		})
	}
}

// TestLessonsContractHitsClauses supports the absence scans of this file: a negation excuses a match only in the
// clause of the match, and a clause ends at a sentence end, a colon, a table cell border, a list item or line break,
// a comma, a semicolon, a parenthesis and a dash, with blockquote markers stripped.
func TestLessonsContractHitsClauses(t *testing.T) {
	t.Parallel()

	scan := lessonsContractNoneScan

	for name, scenario := range map[string]struct {
		content string
		want    int
	}{
		"an imperative is a hit":                            {"Capture a note for a `none` line.", 1},
		"a negated imperative is a prohibition":             {"Never capture a note for a `none` line.", 0},
		"a negation in a wrapped line of the same clause":   {"Do not\ncapture a note for a `none` line.", 0},
		"a negated earlier sentence does not excuse":        {"Do not wait. Capture a note for a `none` line.", 1},
		"a negated earlier list item does not excuse":       {"- Never wait\n- Capture a note for a `none` line.\n", 1},
		"a negated earlier table cell does not excuse":      {"| No wait | Capture a note for a `none` line |", 1},
		"a negated clause before a colon does not excuse":   {"Never skip this: capture a note for a `none` line.", 1},
		"a negated condition before a comma does not":       {"If it isn't clear, capture a note for a `none` line.", 1},
		"a negation behind a quote marker is still seen":    {"> Never capture a note for a `none` line.", 0},
		"a quote marker does not hide an imperative":        {"> Capture a note for a `none` line.", 1},
		"a negation after a dash does not excuse before it": {"Capture a note for a `none` line — never mind.", 1},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(lessonsContractHits(scenario.content, scan)).To(HaveLen(scenario.want))
		})
	}
}

// unexported constants.
const (
	lessonsContractContractEnd   = "Vague handoffs are why"
	lessonsContractContractStart = "**The completion-report contract:**"
	lessonsContractEndsWithLine  = "the subagent's final report MUST end with a `LESSONS:` line"
	lessonsContractGuidancePath  = "agent-instructions/guidance/learn.md"
	lessonsContractLearnPath     = "agent-instructions/skills/learn/SKILL.md"
	lessonsContractPleasePath    = "agent-instructions/skills/please/SKILL.md"
	lessonsContractRoutePath     = "agent-instructions/skills/route/SKILL.md"
)

// unexported variables.
var (
	// lessonsContractBorders are the fragments of the collapsed blocks after which a new clause starts.
	lessonsContractBorders = []string{". ", "? ", "! ", ": ", "| ", "\n", ",", ";", "(", "—"}
	// lessonsContractDiscardLogScan finds an instruction to log or announce a discarded line.
	lessonsContractDiscardLogScan = lessonsContractScan{
		name: "logging a discard",
		pattern: regexp.MustCompile(
			`(?i)\b(?:log|logs|logged|logging|report|reports|announce|announces)\b[^.\n]{0,40}?` +
				`\bdiscard\w*|\bdiscard\w*\b[^.\n]{0,60}?\b(?:log\w*|with\s+a\s+note|and\s+(?:report|announce|tell|say))\b`),
		negation:    lessonsContractNegation,
		paraphrase:  "Each discarded line is written to the session log.",
		imperative:  "Log each discarded line in the closing report.",
		conditional: "If the line isn't captured, log the discard in the closing report.",
		stated: "A line that doesn't clear the bar is discarded silently — no vault note, no logged discard — " +
			"the line is a low-bar offer.",
	}
	// lessonsContractExemptionScan finds an instruction that exempts a dispatch from the completion-report contract.
	lessonsContractExemptionScan = lessonsContractScan{
		name: "exempting a dispatch from the contract",
		pattern: regexp.MustCompile(`(?i)` + lessonsContractOrdered(
			`\b(?:omit\w*|skip\w*|leave\s+out|left\s+out|drop\w*|waive\w*|unnecessary|optional|redundant)\b`,
			`(?:completion-report\s+contract|\b(?-i:LESSONS)(?::|\s+line|\s+mandate)|\bmandate\b)`)),
		negation:    lessonsContractNegationKeepingSkip,
		paraphrase:  "A fork inherits the session context, so the handoff can leave out the LESSONS mandate.",
		imperative:  "Omit the completion-report contract from the handoff of a fork.",
		conditional: "If the unit is small, skip the LESSONS line in its handoff.",
		stated:      "Every dispatch MUST hand the subagent the completion-report contract.",
	}
	// lessonsContractGateDropScan finds an instruction to drop the line of a unit that a gate rejected.
	lessonsContractGateDropScan = lessonsContractScan{
		name: "dropping the line of a rejected unit",
		pattern: regexp.MustCompile(`(?i)` + lessonsContractOrdered(
			`\b(?:reject\w*|gate|FAIL\w*)\b`,
			`\b(?-i:LESSONS)\b`,
			`\b(?:drop\w*|discard\w*|omit\w*|exclud\w+|filter\w*|leave\s+out|left\s+out|remov\w+|skip\w*|ignor\w+)\b`)),
		negation:    lessonsContractNegationKeepingSkip,
		paraphrase:  "A unit that fails a gate has its LESSONS line left out of the running list.",
		imperative:  "Drop the LESSONS line of a unit that a gate rejected.",
		conditional: "If the work isn't accepted, omit its LESSONS line when a gate fails.",
		stated:      "A gate FAIL verdict never removes the LESSONS line of the unit from the running list.",
	}
	// lessonsContractNegation marks a clause that forbids or excludes what it names.
	lessonsContractNegation = regexp.MustCompile(`(?i)\b(?:not|never|no|neither|nor|without|skip|skips)\b|n't\b`)
	// lessonsContractNegationKeepingSkip is the negation of a scan whose forbidden act can be to skip.
	lessonsContractNegationKeepingSkip = regexp.MustCompile(`(?i)\b(?:not|never|no|neither|nor|without)\b|n't\b`)
	// lessonsContractNoneReference matches a mention of a none line.
	lessonsContractNoneReference = `(?:LESSONS:\s*none|\x60none\x60|\bnone\s+(?:lines?|entr(?:y|ies)|lessons?))`
	// lessonsContractNoneScan finds an instruction to capture a none line with a learn.
	lessonsContractNoneScan = lessonsContractScan{
		name: "capturing a none line",
		pattern: regexp.MustCompile(
			`(?i)(?:\b(?:fir(?:e|es|ing)|invok(?:e|es|ing)|captur(?:e|es|ing)|crystalli[sz](?:e|es|ing)|fast-path)\b` +
				`|/learn\b)[^.\n]{0,80}?` + lessonsContractNoneReference + `|` + lessonsContractNoneReference +
				`[^.\n]{0,60}?\b(?:gets?|earns?|warrants?|deserves?|needs?|receives?)\b[^.\n]{0,30}\b(?:learn|note|capture)\b`),
		negation:    lessonsContractNegation,
		paraphrase:  "A `none` line earns a fast-path learn too.",
		imperative:  "Fire a fast-path /learn for each `LESSONS: none` line.",
		conditional: "If the report isn't clear, capture a note for the `none` line.",
		stated:      "Judge each lesson as you read the report. Skip `none`, \"done, tests pass\", and the rest.",
	}
	// lessonsContractReaskScan finds an instruction to ask for the LESSONS line a second time.
	lessonsContractReaskScan = lessonsContractScan{
		name: "asking for the line again",
		pattern: regexp.MustCompile(
			`(?i)\b(?:(?:re-?ask|ask|follow\s+up|chase|nudge)\w*\s+(?:the\s+subagent\s+)?` +
				`(?:again|a\s+second\s+time|twice|more\s+than\s+once)|(?:a\s+)?second\s+(?:re-?ask|follow-?up|request)|` +
				`(?:another|further|repeated)\s+(?:re-?ask|follow-?up|request)|follow-?up\s+again)\b`),
		negation:    lessonsContractNegation,
		paraphrase:  "A subagent whose line is still missing is asked a second time.",
		imperative:  "Re-ask the subagent again for the line.",
		conditional: "If the follow-up isn't answered, send another follow-up.",
		stated:      "Send ONE targeted follow-up asking for it — never a second re-ask.",
	}
	// lessonsContractRewriteScan finds an instruction to write a captured line again.
	lessonsContractRewriteScan = lessonsContractScan{
		name: "writing a captured line again",
		pattern: regexp.MustCompile(
			`(?i)\b(?:writ(?:e|es|ing|ten)|re-?writ\w+|captur(?:e|es|ing|ed)|crystalli[sz]\w+|duplicat\w+)\b` +
				`[^.\n]{0,60}?\b(?:line|lesson)s?\b[^.\n]{0,60}?` +
				`\b(?:again|a\s+second\s+time|twice|already\s+(?:captured|covered|written))\b|` +
				`\b(?:line|lesson)s?\b[^.\n]{0,60}?\b(?:is|are|gets?)\s+(?:writ\w+|captured|crystalli[sz]ed)\s+` +
				`(?:again|a\s+second\s+time|twice)\b`),
		negation:    lessonsContractNegation,
		paraphrase:  "A line captured at its return is written again by the closing learn.",
		imperative:  "Write the line again at the closing learn.",
		conditional: "If the note isn't found, write the lesson again at the closing learn.",
		stated:      "A line is captured when a note covers it: skip captured lines — never write one a second time.",
	}
	// lessonsContractSuccessKindScan finds an instruction that counts a bare success as a confirmed approach.
	lessonsContractSuccessKindScan = lessonsContractScan{
		name: "a bare success as kind 4",
		pattern: regexp.MustCompile(
			`(?i)\b(?:bare\s+success|routine\s+success|(?:the\s+)?task\s+is\s+done|"?it\s+worked"?)\b` +
				`[^.\n]{0,60}?\b(?:qualif(?:y|ies)|counts?|is\s+kind\s+4|is\s+a\s+confirmed\s+approach|is\s+captured|` +
				`earns?|gets?)\b|\b(?:treat|classif(?:y|ies)|count|record|capture)\w*\b[^.\n]{0,40}?` +
				`\b(?:bare|routine)\s+success\b[^.\n]{0,40}?` +
				`\b(?:as\s+(?:a\s+)?(?:kind\s+4|confirmed\s+approach|reinforcement)|kind\s+4)`),
		negation:    lessonsContractNegation,
		paraphrase:  "A bare success is a confirmed approach too.",
		imperative:  "Classify a bare success as kind 4.",
		conditional: "If the user isn't specific, a bare success counts as a confirmed approach.",
		stated:      "A routine success does not count as reinforcement-capture.",
	}
	// lessonsContractSuccessNoteScan finds an instruction to capture a note for a routine success.
	lessonsContractSuccessNoteScan = lessonsContractScan{
		name: "capturing a routine success",
		pattern: regexp.MustCompile(
			`(?i)(?:\b(?:fir(?:e|es|ing)|invok(?:e|es|ing)|captur(?:e|es|ing)|crystalli[sz](?:e|es|ing))\b|/learn\b)` +
				`[^.\n]{0,80}?(?:done,\s*tests\s+pass|we\s+did\s+the\s+work|(?:bare|routine)\s+success)|` +
				`(?:done,\s*tests\s+pass|we\s+did\s+the\s+work|(?:bare|routine)\s+success)[^.\n]{0,40}?` +
				`\b(?:gets?|earns?|warrants?|deserves?|qualif(?:y|ies))\b[^.\n]{0,30}\b(?:learn|note|capture)\b`),
		negation:    lessonsContractNegation,
		paraphrase:  "A routine success such as \"done, tests pass\" earns a note too.",
		imperative:  "Capture a note for each \"done, tests pass\" report.",
		conditional: "If the work isn't reviewed, fire a fast-path /learn for a routine success.",
		stated:      "Skip `none`, \"done, tests pass\", and anything the report doesn't confirm.",
	}
	// lessonsContractUnconfirmedScan finds an instruction to capture an unconfirmed lesson.
	lessonsContractUnconfirmedScan = lessonsContractScan{
		name: "capturing an unconfirmed lesson",
		pattern: regexp.MustCompile(
			`(?i)(?:\b(?:fir(?:e|es|ing)|invok(?:e|es|ing)|captur(?:e|es|ing)|crystalli[sz](?:e|es|ing))\b|/learn\b)` +
				`[^.\n]{0,80}?\b(?:unconfirmed|hypothesi[sz]\w*|hunch(?:es)?|guess(?:es)?|speculat\w+|might\s+be)\b|` +
				`\b(?:unconfirmed|hypothesi[sz]\w*|speculative|hunch)\s+(?:lessons?|lines?|offers?)[^.\n]{0,40}?` +
				`\b(?:gets?|earns?|qualif(?:y|ies)|is\s+captured|counts?)\b`),
		negation:    lessonsContractNegation,
		paraphrase:  "An unconfirmed lesson earns a note as well.",
		imperative:  "Capture a hunch from the report as a note.",
		conditional: "If the report isn't long, crystallize an unconfirmed guess as a note.",
		stated:      "Do NOT crystallize an unconfirmed hunch — that rots the vault.",
	}
)

// lessonsContractScan is one absence scan: a pattern for an instruction that contradicts a rule, the negation pattern
// that excuses a match in a clause that forbids it, the three forms of a contradicting instruction to insert (a
// paraphrase, an imperative and the imperative under a negated condition) and the wording that must pass.
type lessonsContractScan struct {
	name        string
	pattern     *regexp.Regexp
	negation    *regexp.Regexp
	paraphrase  string
	imperative  string
	conditional string
	stated      string
}

// lessonsContractBeforeContract reports whether the fragment occurs in the text before the completion-report contract.
func lessonsContractBeforeContract(text, fragment string) bool {
	fragmentAt := strings.Index(text, fragment)
	contractAt := strings.Index(text, lessonsContractContractStart)

	return fragmentAt >= 0 && contractAt >= 0 && fragmentAt < contractAt
}

// lessonsContractClauseStart returns the offset in the lead where the clause that precedes a match begins.
func lessonsContractClauseStart(lead string) int {
	start := 0

	for _, border := range lessonsContractBorders {
		if at := strings.LastIndex(lead, border); at >= 0 {
			start = max(start, at+len(border))
		}
	}

	return start
}

// lessonsContractContradictions returns, by path, the clauses of every shipped SKILL.md, references file and guidance
// document that the scan finds. It fails the test when the scan did not cover the four files of the contract, so an
// empty result is not vacuous.
func lessonsContractContradictions(t *testing.T, scan lessonsContractScan) map[string][]string {
	t.Helper()

	found := make(map[string][]string)
	covered := make(map[string]bool)

	for _, path := range aibInstructionPaths(t) {
		covered[path] = true

		hits := lessonsContractHits(lessonsContractRaw(t, path), scan)
		if len(hits) > 0 {
			found[path] = hits
		}
	}

	for _, path := range []string{
		lessonsContractGuidancePath, lessonsContractLearnPath, lessonsContractPleasePath, lessonsContractRoutePath,
	} {
		if !covered[path] {
			t.Fatalf("the instruction scan did not cover %s", path)
		}
	}

	return found
}

// lessonsContractHits returns the clause of every match of the scan's pattern in the content of a shipped file whose
// clause, from the last border before the match to the end of the match, carries no negation. The content is read by
// block (a paragraph, a list item, a table row or a heading is one line, and blockquote markers are stripped), so a
// negation in an earlier block does not excuse a match in a later one.
func lessonsContractHits(content string, scan lessonsContractScan) []string {
	text := learnPart1Blocks(content)
	hits := make([]string, 0)

	for _, span := range scan.pattern.FindAllStringIndex(text, -1) {
		clause := text[lessonsContractClauseStart(text[:span[0]]):span[1]]
		if !scan.negation.MatchString(clause) {
			hits = append(hits, strings.TrimSpace(clause))
		}
	}

	return hits
}

// lessonsContractOrdered returns a regular expression that matches the parts in any order within one sentence.
func lessonsContractOrdered(parts ...string) string {
	if len(parts) == 1 {
		return parts[0]
	}

	alternatives := make([]string, 0, len(parts))

	for index := range parts {
		rest := make([]string, 0, len(parts)-1)
		rest = append(rest, parts[:index]...)
		rest = append(rest, parts[index+1:]...)

		alternatives = append(alternatives, parts[index]+`[^.\n]{0,60}?(?:`+lessonsContractOrdered(rest...)+`)`)
	}

	return strings.Join(alternatives, "|")
}

// lessonsContractRaw returns the content of a shipped file as it is on disk.
func lessonsContractRaw(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(repositoryRoot, path))
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)

		return ""
	}

	return string(content)
}

// lessonsContractRouteContract returns the completion-report contract bullet of the route skill's handoff.
func lessonsContractRouteContract(t *testing.T) string {
	t.Helper()

	text := shippedText(t, lessonsContractRoutePath)

	_, contract, found := strings.Cut(text, lessonsContractContractStart)
	if !found {
		t.Fatalf("the route skill has no %q", lessonsContractContractStart)

		return ""
	}

	contract, _, found = strings.Cut(contract, lessonsContractContractEnd)
	if !found {
		t.Fatalf("the route skill has no %q after the contract", lessonsContractContractEnd)

		return ""
	}

	return lessonsContractContractStart + contract
}

// lessonsContractScans returns every absence scan of this file.
func lessonsContractScans() []lessonsContractScan {
	return []lessonsContractScan{
		lessonsContractDiscardLogScan, lessonsContractExemptionScan, lessonsContractGateDropScan,
		lessonsContractNoneScan, lessonsContractReaskScan, lessonsContractRewriteScan,
		lessonsContractSuccessKindScan, lessonsContractSuccessNoteScan, lessonsContractUnconfirmedScan,
	}
}
