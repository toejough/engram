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

// Text-level: each test below reads the learn skill the repository ships and asserts the sentences that state the rule
// of its scenario. It verifies that the shipped rule is stated, not that an agent follows it. The scenarios whose Then
// has a prohibition half also scan every shipped SKILL.md, references file and guidance document for an instruction
// that contradicts it. The rules of the guidance document learn.md (792a19c, 792a19d) and the save-request for status
// (855a3r1a) have their own tests in skillrules_ambient_guidance_part1_integration_internal_test.go and
// skillrules_integration_internal_test.go, and are not repeated here.

// Test792a15a1a_ASessionWithNoLessonWritesNothing reads the learn skill the repository ships: Step 1 runs the sweep
// and reports its tally, and a session with none of the four moments writes no note.
func Test792a15a1a_ASessionWithNoLessonWritesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("## Step 1 — Sweep (first; the closing learn always sweeps) ```bash " +
		"engram ingest --auto ```"))
	g.Expect(text).To(ContainSubstring("Report the one-line tally it prints (or \"memory index up to date\")."))
	g.Expect(text).To(ContainSubstring("**No moments of any kind → write nothing.**"))
	g.Expect(text).To(ContainSubstring("a session with no corrections, no save-requests, no reversals, and no " +
		"confirmed approaches is a two-command learn (sweep + report)."))
	g.Expect(learnPart1Instructions(t, learnPart1WriteWithoutMomentPattern, nil)).To(BeEmpty())
}

// Test792a15a1b_NoEpisodeOrSummaryIsWritten reads the learn skill the repository ships: the old episode workflow is
// gone, no summary or episode note is written, and neither engram transcript nor engram learn episode is run.
func Test792a15a1b_NoEpisodeOrSummaryIsWritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("**Raw event memory is AUTOMATIC.** `engram ingest` chunks and embeds session " +
		"transcripts and markdown itself — no summaries, no episode notes, no arc detection."))
	g.Expect(text).To(ContainSubstring("Do not reconstruct the old episode workflow (`engram transcript`, " +
		"`engram learn episode`) — it is gone."))
	g.Expect(text).To(ContainSubstring("| You ran `engram transcript` anything |"))
	g.Expect(text).To(ContainSubstring("| You're writing an episode or summarizing the session into a note | " +
		"Don't — raw chunks already hold it |"))
	g.Expect(learnPart1Instructions(t, learnPart1EpisodePattern, nil)).To(BeEmpty())
	g.Expect(learnPart1Instructions(t, learnPart1TranscriptPattern, nil)).To(BeEmpty())
}

// Test792a15b1b_AFailedSweepDoesNotBlockTheLessons reads the learn skill the repository ships: a failed sweep is
// surfaced and the procedure continues to Step 2, so the lessons are not lost.
func Test792a15b1b_AFailedSweepDoesNotBlockTheLessons(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("If the command fails, surface the error and continue to Step 2 — explicit " +
		"lessons must not be lost to an ingest hiccup."))
	g.Expect(learnPart1Before(text, "If the command fails, surface the error", "## Step 2 — Crystallize")).To(BeTrue())
}

// Test792a15b1c_AnEarlierSweepIsNotRepeated reads the learn skill the repository ships: a sweep that already ran in the
// session is skipped, even under a worry that something changed in another terminal.
func Test792a15b1c_AnEarlierSweepIsNotRepeated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("Skip the sweep if one already ran earlier in THIS session, UNLESS this is the " +
		"closing learn of a work cycle"))
	g.Expect(text).To(ContainSubstring("This holds even under a \"something might have changed outside this session " +
		"(another terminal, another agent)\" worry — that gets picked up by the next sweep (yours or theirs), not by " +
		"re-sweeping now on suspicion."))
	g.Expect(learnPart1Instructions(t, learnPart1SweepAgainPattern, learnPart1ClosingLearnPattern)).To(BeEmpty())
}

// Test792a15b1d_TheClosingLearnAlwaysSweeps reads the learn skill the repository ships: the closing learn of a work
// cycle always sweeps, to capture the tail of the session, whether or not a sweep ran earlier.
func Test792a15b1d_TheClosingLearnAlwaysSweeps(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("## Step 1 — Sweep (first; the closing learn always sweeps)"))
	g.Expect(text).To(ContainSubstring("UNLESS this is the closing learn of a work cycle — the closing learn ALWAYS " +
		"sweeps (it captures the session's tail for future sessions)."))
	g.Expect(text).To(ContainSubstring("The closing learn still sweeps."))
}

// Test792a15c1a_ARejectedApproachIsCapturedAtOnce reads the learn skill the repository ships: at a correction moment
// the mid-cycle fast path skips the sweep and the vocabulary check and crystallizes the one correction through
// write-memory. The same rule in guidance learn.md is Test792a19c1a_ARejectedApproachIsCapturedBeforeTheFix.
func Test792a15c1a_ARejectedApproachIsCapturedAtOnce(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("When you fire at a CORRECTION moment mid-task — a review, a failing check, or " +
		"the user just rejected your approach or named a different convention"))
	g.Expect(text).To(ContainSubstring("this is a focused single-note capture, NOT the open/close of a cycle."))
	g.Expect(text).To(ContainSubstring("**SKIP Step 1 (sweep)"))
	g.Expect(text).To(ContainSubstring("and Step 1.5 (vocab); go straight to Step 2**"))
	g.Expect(text).To(ContainSubstring("crystallize the ONE confirmed correction (hand off to write-memory as always)."))
	g.Expect(learnPart1Instructions(t, learnPart1CorrectionCheckPattern, nil)).To(BeEmpty())
}

// Test792a15c1b_NoMidTaskSweep reads the learn skill the repository ships: engram ingest --auto is not run in the
// middle of a task, for it is a cycle-boundary job that can block and lose the capture. The same rule in guidance
// learn.md is Test792a19d1b_NoSweepRunsMidTask.
func Test792a15c1b_NoMidTaskSweep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("Do NOT run `engram ingest --auto` mid-task — the sweep is a cycle-boundary " +
		"job, it is wasteful here and can block on a large corpus, and losing the capture to a hung sweep is the " +
		"failure mode."))
	g.Expect(learnPart1Instructions(t, learnPart1MidTaskSweepPattern, nil)).To(BeEmpty())
}

// Test792a15d1a_AHealthyVocabularyNeedsNothing reads the learn skill the repository ships: Step 1.5 runs engram vocab
// stats and, on verdict: OK, continues to Step 2 with no further vocabulary action.
func Test792a15d1a_AHealthyVocabularyNeedsNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("## Step 1.5 — Vocab liveness check Run `engram vocab stats`."))
	g.Expect(text).To(ContainSubstring("If the verdict is `verdict: OK`, continue to Step 2 with no further vocab " +
		"action."))
	g.Expect(learnPart1Instructions(t, learnPart1HealthyVocabularyPattern, nil)).To(BeEmpty())
}

// Test792a15d1b_APendingRefitIsRunAutonomously reads the learn skill the repository ships: on verdict: REFIT_PENDING
// the refit is run autonomously, without deferring to the user, starting with engram vocab refit.
func Test792a15d1b_APendingRefitIsRunAutonomously(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("If the output includes a line matching `verdict: REFIT_PENDING (<reason>)`, " +
		"run the refit autonomously — do not defer to the user:"))
	g.Expect(text).To(ContainSubstring("1. Run `engram vocab refit`. The binary derives the clusters itself — there " +
		"is no plan to author."))
	g.Expect(learnPart1Instructions(t, learnPart1AskBeforeRefitPattern, nil)).To(BeEmpty())
}

// Test792a15d1c_NoStructureFoundIsReportedAsSuch reads the learn skill the repository ships: when the refit prints that
// no structure was found, learn reports exactly that and does not claim an applied refit.
func Test792a15d1c_NoStructureFoundIsReportedAsSuch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("If it prints `vocab refit: no structure found; vocabulary unchanged`, " +
		"nothing was written and there is no version bump"))
	g.Expect(text).To(ContainSubstring("report exactly that (\"Vocab refit ran: no structure found, vocabulary " +
		"unchanged.\") and continue to Step 2."))
	g.Expect(text).To(ContainSubstring("Do NOT claim an applied refit."))
	g.Expect(learnPart1Instructions(t, learnPart1NoStructureAppliedPattern, nil)).To(BeEmpty())
}

// Test792a15d1d_NewClustersAreNamedAndTheRefitRerun reads the learn skill the repository ships: a refit that prints
// naming requests and a fingerprint exits without writing; learn names every new cluster once, echoes the fingerprint
// verbatim in the answers file and re-runs the refit with --names.
func Test792a15d1d_NewClustersAreNamedAndTheRefitRerun(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("If there are new clusters, the command prints a JSON payload " +
		"(`naming_requests` + `fingerprint`) and exits WITHOUT writing."))
	g.Expect(text).To(ContainSubstring("Name each new cluster from its exemplars, then write an answer file:"))
	g.Expect(text).To(ContainSubstring("Cover every new cluster exactly once."))
	g.Expect(text).To(ContainSubstring("\"fingerprint\": \"<echoed verbatim from the request payload>\"}"))
	g.Expect(text).To(ContainSubstring("Re-run: `engram vocab refit --names /path/answer.json`."))
}

// Test792a15d1e_AStaleAnswerRestartsTheRefit reads the learn skill the repository ships: a re-run that fails with a
// stale-names error voids the requests; learn re-runs the refit from scratch and answers the regenerated requests.
func Test792a15d1e_AStaleAnswerRestartsTheRefit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("If the re-run fails with a stale-names error (the vault changed between " +
		"runs), the requests are void"))
	g.Expect(text).To(ContainSubstring("re-run `engram vocab refit` from scratch to regenerate them, and answer " +
		"those."))
}

// Test792a15d1f_AnAppliedRefitIsReportedLoudly reads the learn skill the repository ships: an applied refit is reported
// loudly in the stated form, whether the run applied directly or after the re-run.
func Test792a15d1f_AnAppliedRefitIsReportedLoudly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("If every cluster matched an existing term, the run applies directly — skip " +
		"to step 4."))
	g.Expect(text).To(ContainSubstring("4. **Report loudly:** \"Vocab refit applied: <version bump>. Triggered by: " +
		"<reason>.\""))
}

// Test792a15d1g_TheQARound2GateIsReportedAndNotRun reads the learn skill the repository ships: a READY round-2 gate is
// reported to the user with the gates to schedule, and the validation is not run autonomously.
func Test792a15d1g_TheQARound2GateIsReportedAndNotRun(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("Also check the QA round-2 gate line. If the output includes " +
		"`qa round-2 gate: READY (...)`, report to"))
	g.Expect(text).To(ContainSubstring("\"QA round-2 validation is due (≥20 pairs captured). Please schedule the " +
		"round-2 gates recorded in docs/ROADMAP.md"))
	g.Expect(text).To(ContainSubstring("P2′ attribution fidelity, P3′ distribution, Arm V larger-n.\""))
	g.Expect(text).To(ContainSubstring("Do NOT run round-2 validation autonomously — it requires"))
	g.Expect(text).To(ContainSubstring(" oversight."))
	g.Expect(learnPart1Instructions(t, learnPart1Round2RunPattern, nil)).To(BeEmpty())
}

// Test792a15e1a_ACorrectionBecomesAFeedbackNote reads the learn skill the repository ships: a correction is handed to
// write-memory as kind feedback with the situation, the behavior, the impact and the action.
func Test792a15e1a_ACorrectionBecomesAFeedbackNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)
	block := learnPart1Between(t, text, "1. **Corrections**", "2. **Explicit save-requests**")
	handoff := learnPart1Between(t, block, "with this handoff —", "write-memory composes")

	g.Expect(block).To(ContainSubstring("the user corrected your approach or behavior"))
	g.Expect(handoff).To(ContainSubstring("kind=feedback,"))
	g.Expect(handoff).To(ContainSubstring(" situation (retrieval-shaped),"))
	g.Expect(handoff).To(ContainSubstring(" behavior,"))
	g.Expect(handoff).To(ContainSubstring(" impact,"))
	g.Expect(handoff).To(ContainSubstring(" action;"))
}

// Test792a15e1b_ASaveRequestBecomesAFactNoteAtOnce reads the learn skill the repository ships: an explicit
// save-request is handed to write-memory as kind fact with the situation, the subject, the predicate and the object,
// immediately and before anything else.
func Test792a15e1b_ASaveRequestBecomesAFactNoteAtOnce(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)
	block := learnPart1Between(t, text, "2. **Explicit save-requests**", "3. **Reversals**")
	handoff := learnPart1Between(t, block, "with this handoff —", "write-memory composes")

	g.Expect(block).To(ContainSubstring("the user said \"remember this/that X\""))
	g.Expect(handoff).To(ContainSubstring("kind=fact,"))
	g.Expect(handoff).To(ContainSubstring(" situation (retrieval-shaped),"))
	g.Expect(handoff).To(ContainSubstring(" subject,"))
	g.Expect(handoff).To(ContainSubstring(" predicate,"))
	g.Expect(handoff).To(ContainSubstring(" object;"))
	g.Expect(text).To(ContainSubstring("An explicit save-request for a durable lesson ALWAYS gets its note, " +
		"immediately — \"remember this\" means stop and write before anything else."))
}

// Test792a15e1c_AReversalBecomesAFeedbackNoteAboutTheRootCause reads the learn skill the repository ships: a reversal
// is handed to write-memory as kind feedback whose behavior is what the original reasoning did wrong and whose action
// is the guard that would have prevented it.
func Test792a15e1c_AReversalBecomesAFeedbackNoteAboutTheRootCause(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)
	block := learnPart1Between(t, text, "3. **Reversals**", "4. **Confirmed approaches (positive reinforcement)**")
	handoff := learnPart1Between(t, block, "with this handoff —", "plus supersedes details")

	g.Expect(block).To(ContainSubstring("PRESENTED (to the user, a review gate, or a committed plan) and later " +
		"OVERTURNED — by you, a reviewer, or an instrument"))
	g.Expect(handoff).To(ContainSubstring("kind=feedback,"))
	g.Expect(handoff).To(ContainSubstring("behavior = what the original reasoning did wrong,"))
	g.Expect(handoff).To(ContainSubstring("action = the guard that would have prevented it"))
}

// Test792a15e1d_AConfirmedApproachBecomesAFeedbackNote reads the learn skill the repository ships: a confirmed
// approach of no step structure is handed to write-memory as kind feedback with the situation of its next use, what
// worked, the confirming evidence and a keep-doing action.
func Test792a15e1d_AConfirmedApproachBecomesAFeedbackNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)
	block := learnPart1Between(t, text, "4. **Confirmed approaches (positive reinforcement)**", "Rules: - **State the")
	handoff := learnPart1Between(t, block, "For **feedback**,", "plus supersedes details")

	g.Expect(block).To(ContainSubstring("validated as good, either by the user or by the outcome"))
	g.Expect(handoff).To(ContainSubstring("kind=feedback,"))
	g.Expect(handoff).To(ContainSubstring("situation (retrieval-shaped: when would this approach apply again),"))
	g.Expect(handoff).To(ContainSubstring("behavior = what worked,"))
	g.Expect(handoff).To(ContainSubstring("impact = the confirming evidence (the user's quote for 4a, or the observed " +
		"outcome that resolved the uncertainty for 4b),"))
	g.Expect(handoff).To(ContainSubstring("action = keep doing it + its trigger conditions;"))
}

// Test792a15e1e_ACorrectionOfAnExistingNoteCarriesASupersession reads the learn skill the repository ships: a lesson
// that corrects, narrows or refutes an existing vault note carries that note's basename, type and claim in the handoff,
// and each of the four kinds of handoff names the supersedes details.
func Test792a15e1e_ACorrectionOfAnExistingNoteCarriesASupersession(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)
	rule := learnPart1Between(t, text, "- If the new lesson CORRECTS", "- **No moments of any kind")

	g.Expect(rule).To(ContainSubstring("If the new lesson CORRECTS, narrows, or refutes an existing vault note, "))
	g.Expect(rule).To(ContainSubstring("include the superseded note's basename,"))
	g.Expect(rule).To(ContainSubstring(" type,"))
	g.Expect(rule).To(ContainSubstring(" and claim in the handoff."))

	for _, kind := range []struct{ start, end string }{
		{"1. **Corrections**", "2. **Explicit save-requests**"},
		{"2. **Explicit save-requests**", "3. **Reversals**"},
		{"3. **Reversals**", "4. **Confirmed approaches (positive reinforcement)**"},
		{"4. **Confirmed approaches (positive reinforcement)**", "Rules: - **State the"},
	} {
		g.Expect(learnPart1Between(t, text, kind.start, kind.end)).To(ContainSubstring("plus supersedes details if"),
			kind.start)
	}
}

// Test792a15f1a_ASelfDiscoveredReversalIsCaptured reads the learn skill the repository ships: a reversal qualifies
// when nobody said the correction aloud, an instrument may overturn it, and the note is for the root cause.
func Test792a15f1a_ASelfDiscoveredReversalIsCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)
	block := learnPart1Between(t, text, "3. **Reversals**", "4. **Confirmed approaches (positive reinforcement)**")

	g.Expect(block).To(ContainSubstring("OVERTURNED — by you, a reviewer, or an instrument"))
	g.Expect(block).To(ContainSubstring("Nobody needs to have SAID the correction — self-discovered reversals qualify"))
	g.Expect(block).To(ContainSubstring("the ROOT CAUSE, not a narrative of the flip"))
}

// Test792a15f1b_ARepoDocCorrectionDoesNotCount reads the learn skill the repository ships: a CORRECTION section or
// postscript in a repository document is not the capture, and the vault note for the reversal is still written.
func Test792a15f1b_ARepoDocCorrectionDoesNotCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := learnPart1Text(t)

	g.Expect(text).To(ContainSubstring("a repo-doc CORRECTION section or postscript does NOT count as capture " +
		"(record-correction ≠ lesson-capture)"))
	g.Expect(text).To(ContainSubstring("| You corrected a repo doc (CORRECTION/postscript) and skipped the vault note | " +
		"Write the vault note for the reversal's root cause — record-correction is not capture |"))
}

// TestLearnPart1Hits supports the absence scans of this file: a match is an instruction unless the clause it stands in
// negates it, and the clause ends at a sentence end, a colon, a table cell border and a line break.
func TestLearnPart1Hits(t *testing.T) {
	t.Parallel()

	episode := regexp.MustCompile(`(?i)\b(?:write|create)\b[^.\n]{0,40}\bepisode\b`)
	sweep := regexp.MustCompile(`(?i)\b(?:re-?sweep\w*|sweeps? again)\b`)

	for name, scenario := range map[string]struct {
		content string
		pattern *regexp.Regexp
		exempt  *regexp.Regexp
		want    int
	}{
		"a negated sentence is a prohibition": {
			"Do not write an episode note.", episode, nil, 0,
		},
		"an instruction is a hit": {
			"Write an episode note.", episode, nil, 1,
		},
		"a table row after a negated row is a hit": {
			"| Tiers | Tiers are not part of learn anymore |\n| Session end | Write an episode note |\n", episode, nil, 1,
		},
		"a cell after a negated cell is a hit": {
			"| No tiers | Write an episode note |", episode, nil, 1,
		},
		"a list item after a negated item is a hit": {
			"- Never summarize\n- Write an episode note\n", episode, nil, 1,
		},
		"a clause after a colon is a hit": {
			"Never skip the check: write an episode note.", episode, nil, 1,
		},
		"a negation in a wrapped line of the same sentence is a prohibition": {
			"Do NOT write\nan episode note.", episode, nil, 0,
		},
		"a heading before the sentence is another block": {
			"## Not now\nWrite an episode note.", episode, nil, 1,
		},
		"a sweep instruction is a hit": {
			"Sweep again after every ten tool calls.", sweep, learnPart1ClosingLearnPattern, 1,
		},
		"the closing learn sweeping again is exempt": {
			"The closing learn sweeps again.", sweep, learnPart1ClosingLearnPattern, 0,
		},
		"a sweep again that names the closing learn after the match is exempt": {
			"Sweep again only at the closing learn.", sweep, learnPart1ClosingLearnPattern, 0,
		},
		"a sweep again in the row after the closing learn row is a hit": {
			"| Closing learn | Sweeps |\n| Mid-task | Sweep again |\n", sweep, learnPart1ClosingLearnPattern, 1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(learnPart1Hits(scenario.content, scenario.pattern, scenario.exempt)).To(HaveLen(scenario.want))
		})
	}
}

// unexported constants.
const (
	learnPart1NegationWindow = 60
	learnPart1SkillPath      = "agent-instructions/skills/learn/SKILL.md"
)

// unexported variables.
var (
	// learnPart1AskBeforeRefitPattern finds an instruction to ask or defer to the user about the refit.
	learnPart1AskBeforeRefitPattern = regexp.MustCompile(
		`(?i)\b(?:ask|confirm|check with|wait for|defer to)\b[^.\n]{0,40}\b(?:user|owner|human)\b[^.\n]{0,60}\brefit\b`)
	// learnPart1BlockStartPattern finds a line that starts a new block: a table row, a list item, a heading or a fence.
	learnPart1BlockStartPattern = regexp.MustCompile("^(?:\\||[-*+] |#|\\d+\\. |```)")
	// learnPart1ClosingLearnPattern finds a sentence that is about the closing learn, which always sweeps.
	learnPart1ClosingLearnPattern = regexp.MustCompile(`(?i)\bclosing learn\b`)
	// learnPart1CorrectionCheckPattern finds an instruction to sweep or check the vocabulary at a correction moment.
	learnPart1CorrectionCheckPattern = regexp.MustCompile(
		`(?i)\b(?:run|perform|start|execute)\b[^.\n]{0,40}\b(?:sweep|vocab check|vocab stats|engram ingest --auto)\b` +
			`[^.\n]{0,60}\b(?:mid-task|mid-cycle|correction)\b`)
	// learnPart1EpisodePattern finds an instruction to write an episode or a session summary.
	learnPart1EpisodePattern = regexp.MustCompile(
		`(?i)\b(?:write|create|add|record)\b[^.\n]{0,40}\b(?:episode|session[- ]summary|summary note)\b`)
	// learnPart1HealthyVocabularyPattern finds an instruction to act on the vocabulary after a verdict of OK.
	learnPart1HealthyVocabularyPattern = regexp.MustCompile(
		`(?i)verdict: OK\W{0,3}[^.\n]{0,60}\b(?:run|refit|apply|propose|bootstrap)\b`)
	// learnPart1MidTaskSweepPattern finds an instruction to run the sweep in the middle of a task.
	learnPart1MidTaskSweepPattern = regexp.MustCompile(
		`(?i)\b(?:run|execute|start|launch)\b[^.\n]{0,20}engram ingest --auto[^.\n]{0,40}` +
			`\b(?:mid-task|mid-cycle|correction|at once)\b`)
	// learnPart1NegationPattern finds a negation in the text that leads up to a match.
	learnPart1NegationPattern = regexp.MustCompile(`(?i)\b(?:not|never|no|neither|nor|without|skip)\b|n't\b`)
	// learnPart1NoStructureAppliedPattern finds an instruction to report an applied refit when no structure was found.
	learnPart1NoStructureAppliedPattern = regexp.MustCompile(
		`(?i)no structure found[^.\n]{0,80}\b(?:report|say|state|claim)\b[^.\n]{0,40}\bapplied\b`)
	// learnPart1Round2RunPattern finds an instruction to run the round-2 validation.
	learnPart1Round2RunPattern = regexp.MustCompile(`(?i)\b(?:run|start|launch|execute|perform)\b[^.\n]{0,40}round-2`)
	// learnPart1SweepAgainPattern finds an instruction to sweep again.
	learnPart1SweepAgainPattern = regexp.MustCompile(`(?i)\b(?:re-?sweep\w*|sweeps? again)\b`)
	// learnPart1TranscriptPattern finds an instruction to run engram transcript or engram learn episode.
	learnPart1TranscriptPattern = regexp.MustCompile(
		`(?i)\b(?:run|use|invoke|call|execute)\b[^.\n]{0,20}\bengram (?:transcript|learn episode)\b`)
	// learnPart1WriteWithoutMomentPattern finds an instruction to write a note whether or not a moment occurred.
	learnPart1WriteWithoutMomentPattern = regexp.MustCompile(
		`(?i)\b(?:write|crystalli[sz]e|record)\b[^.\n]{0,40}\b(?:note|summary)\b[^.\n]{0,60}` +
			`\b(?:even (?:when|if)|regardless|anyway)\b`)
)

// learnPart1Before reports whether both fragments occur in the text and the first occurs before the second.
func learnPart1Before(text, first, second string) bool {
	firstAt := strings.Index(text, first)
	secondAt := strings.Index(text, second)

	return firstAt >= 0 && secondAt >= 0 && firstAt < secondAt
}

// learnPart1Between returns the text from the start fragment up to the end fragment, and fails the test when either is
// missing or out of order.
func learnPart1Between(t *testing.T, text, start, end string) string {
	t.Helper()

	startAt := strings.Index(text, start)
	if startAt < 0 {
		t.Fatalf("the text has no %q", start)

		return ""
	}

	endAt := strings.Index(text[startAt:], end)
	if endAt < 0 {
		t.Fatalf("the text has no %q after %q", end, start)

		return ""
	}

	return text[startAt : startAt+endAt]
}

// learnPart1Blocks returns the text of a shipped file with the lines of one block (a paragraph, a list item, a table
// row or a heading) joined by single spaces and the blocks separated by a line break, so that the sentence, the cell
// and the line a match stands in can be told apart.
func learnPart1Blocks(content string) string {
	var builder strings.Builder

	separator := "\n"

	for raw := range strings.SplitSeq(content, "\n") {
		line := strings.Join(strings.Fields(strings.TrimLeft(strings.TrimSpace(raw), "> ")), " ")
		if line == "" {
			separator = "\n"

			continue
		}

		if learnPart1BlockStartPattern.MatchString(line) {
			separator = "\n"
		}

		if builder.Len() > 0 {
			builder.WriteString(separator)
		}

		builder.WriteString(line)

		separator = " "
		if strings.HasPrefix(line, "#") {
			separator = "\n"
		}
	}

	return builder.String()
}

// learnPart1Boundary returns the offset in the lead where the clause that precedes a match begins: after the last
// sentence end, colon, table cell border or line break.
func learnPart1Boundary(lead string) int {
	start := 0

	for _, border := range []string{". ", "? ", "! ", ": ", "| ", "\n"} {
		if at := strings.LastIndex(lead, border); at >= 0 {
			start = max(start, at+len(border))
		}
	}

	return start
}

// learnPart1Hits returns every match of the pattern in the content of a shipped file, except a match whose clause
// negates it before the match (a prohibition is not an instruction) and a match whose clause the exempt pattern
// matches (a sentence about another rule); a nil exempt pattern exempts nothing.
func learnPart1Hits(content string, pattern, exempt *regexp.Regexp) []string {
	text := learnPart1Blocks(content)
	hits := make([]string, 0)

	for _, location := range pattern.FindAllStringIndex(text, -1) {
		lead := text[max(0, location[0]-learnPart1NegationWindow):location[0]]
		lead = lead[learnPart1Boundary(lead):]

		if learnPart1NegationPattern.MatchString(lead) {
			continue
		}

		matched := text[location[0]:location[1]]
		trail := text[location[1]:min(len(text), location[1]+learnPart1NegationWindow)]
		trail = trail[:learnPart1TrailEnd(trail)]

		if exempt != nil && exempt.MatchString(lead+matched+trail) {
			continue
		}

		hits = append(hits, matched)
	}

	return hits
}

// learnPart1Instructions returns every match of the pattern in a shipped SKILL.md, references file or guidance
// document that learnPart1Hits keeps.
func learnPart1Instructions(t *testing.T, pattern, exempt *regexp.Regexp) []string {
	t.Helper()

	hits := make([]string, 0)

	for _, path := range aibInstructionPaths(t) {
		content, err := os.ReadFile(filepath.Join(repositoryRoot, path))
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)

			return nil
		}

		for _, hit := range learnPart1Hits(string(content), pattern, exempt) {
			hits = append(hits, path+": "+hit)
		}
	}

	return hits
}

// learnPart1Text returns the text of the learn skill with white space collapsed and the quote markers of its
// blockquotes removed.
func learnPart1Text(t *testing.T) string {
	t.Helper()

	return strings.ReplaceAll(shippedText(t, learnPart1SkillPath), " > ", " ")
}

// learnPart1TrailEnd returns the offset in the trail where the clause that follows a match ends.
func learnPart1TrailEnd(trail string) int {
	end := len(trail)

	for _, border := range []string{". ", "? ", "! ", " |", "\n"} {
		if at := strings.Index(trail, border); at >= 0 {
			end = min(end, at)
		}
	}

	return end
}
