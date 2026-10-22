//go:build targ && integration

package dev

import (
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a15g1a_SpecificPraiseIsCaptured reads the learn skill the repository ships: a user who explicitly praised or
// thanked a specific behavior that restates as a reusable tactic yields a confirmed approach.
func Test792a15g1a_SpecificPraiseIsCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**4a — user-confirmed:** the user explicitly praised or thanked a SPECIFIC " +
		"behavior you could restate as a reusable tactic"))
	g.Expect(text).To(ContainSubstring("(\"thanks for attaching the file — easier to read from my phone\" → " +
		"\"attach deliverable files instead of inlining them\")"))
}

// Test792a15g1b_ABareThanksIsNot reads the learn skill the repository ships: a bare pleasantry that names no behavior
// does not qualify, a bare thanks is a red flag, and no shipped instruction says a pleasantry qualifies.
func Test792a15g1b_ABareThanksIsNot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("A bare pleasantry naming no behavior (\"thanks!\", \"great work\") does NOT " +
		"qualify."))
	g.Expect(text).To(ContainSubstring("| You're writing a confirmed-approach note for a routine success or a bare " +
		"\"thanks!\" | Kind 4 needs a genuine bet with observable confirmation, or explicit praise of a SPECIFIC " +
		"behavior — never mere success |"))
	g.Expect(learnPart2Contradictions(t, learnPart2PleasantryScan)).To(BeEmpty())
}

// Test792a15g1c_ASelfValidatedBetIsCaptured reads the learn skill the repository ships: a genuine guess that was acted
// on and then confirmed by an observable outcome yields a confirmed approach, and the targeted search that checked a
// design assumption is the worked example.
func Test792a15g1c_ASelfValidatedBetIsCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**4b — self-validated bet:** you made a genuine guess, or were uncertain about " +
		"a plan, concept, or idea, ACTED on it"))
	g.Expect(text).To(ContainSubstring("an OBSERVABLE, session-recorded outcome then confirmed it worked"))
	g.Expect(text).To(ContainSubstring("Unsure whether a design assumption held; ran a targeted code search to check " +
		"it before finalizing the design; the search confirmed it → confirmed approach (the check-before-you- commit " +
		"method)"))
}

// Test792a15g1d_ARoutineSuccessIsNot reads the learn skill the repository ships: a routine success with no bet behind
// it is never the signal for a confirmed approach, and no shipped instruction says a routine success qualifies.
func Test792a15g1d_ARoutineSuccessIsNot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("a routine success with no bet behind it, or an unconfirmed guess is never the " +
		"signal"))
	g.Expect(text).To(ContainSubstring("a routine success does not count as reinforcement-capture"))
	g.Expect(text).To(ContainSubstring("the signal is a resolved uncertainty or an explicit specific confirmation, " +
		"never \"it worked\""))
	g.Expect(learnPart2Contradictions(t, learnPart2RoutineSuccessScan)).To(BeEmpty())
}

// Test792a15g1e_AProcedureBecomesARunbook reads the learn skill the repository ships: a reusable, ordered, multi-step
// procedure is handed to write-memory as kind runbook with a situation, done_when and a body of numbered steps, and the
// release sequence is the worked example.
func Test792a15g1e_AProcedureBecomesARunbook(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("a confirmed approach that is a reusable, ordered, multi-step procedure for a " +
		"recurring task (e.g. \"run tests → tag the release → push the tag → update the changelog\" for releasing a " +
		"Go module"))
	g.Expect(text).To(ContainSubstring("→ **kind=runbook**"))
	g.Expect(text).To(ContainSubstring("handoff — kind=runbook, slug, source"))
	g.Expect(text).To(ContainSubstring("situation (retrieval-shaped: when would this approach apply again), " +
		"done_when = what should be true"))
	g.Expect(text).To(ContainSubstring("done_when = what should be true once the procedure is complete"))
	g.Expect(text).To(ContainSubstring("body = the numbered steps"))
}

// Test792a15g1f_ASingleTweakStaysFeedback reads the learn skill the repository ships: a confirmed behavior that is a
// single tweak with no natural step structure is handed to write-memory as kind feedback.
func Test792a15g1f_ASingleTweakStaysFeedback(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("A confirmed approach that is a single behavioral tweak with no natural step " +
		"structure (e.g. \"attach deliverable files instead of inlining them\") → **kind=feedback**, as before."))
	g.Expect(text).To(ContainSubstring("For **feedback**, **REQUIRED SUB-SKILL:** invoke the **write-memory** skill " +
		"with this handoff — kind=feedback, slug"))
}

// Test792a15g1g_ARunbookIsNotDuplicatedAsASaveRequest reads the learn skill the repository ships: when the user also
// says to do the runbook going forward, no kind=fact save-request is additionally fired for the same content, and no
// shipped instruction says to fire one.
func Test792a15g1g_ARunbookIsNotDuplicatedAsASaveRequest(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("The runbook note itself already is the \"remember to do it this way\" capture"))
	g.Expect(text).To(ContainSubstring("if the user also instructs \"do that going forward,\" do NOT additionally fire " +
		"a kind=fact save-request (kind 2) for the same content"))
	g.Expect(text).To(ContainSubstring("that redundancy-avoidance is specific to the runbook case"))
	g.Expect(learnPart2Contradictions(t, learnPart2RunbookFactScan)).To(BeEmpty())
}

// Test792a15h1a_ASubPointIsAContinuation reads the learn skill the repository ships: a note that develops one specific
// sub-point raised inside an in-session note takes position continuation with that note's ID as the target, and the
// position and target go into the write-memory handoff.
func Test792a15h1a_ASubPointIsAContinuation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("include the resulting `position`/`target` in the write-memory handoff"))
	g.Expect(text).To(ContainSubstring("1. Does the new note develop **one specific sub-point** raised inside an " +
		"in-session note?"))
	g.Expect(text).To(ContainSubstring("`position=continuation`, `target=<that note's ID>`."))
}

// Test792a15h1b_TheSameThoughtAtTheSameLevelIsASibling reads the learn skill the repository ships: a note that extends
// the same overall thought as an in-session note at the same level, and is not a sub-point of it, takes position
// sibling with that note's ID as the target.
func Test792a15h1b_TheSameThoughtAtTheSameLevelIsASibling(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("2. Else, does it continue/extend the **same overall thought** as an in-session " +
		"note, at the same level (not a sub-point of it)?"))
	g.Expect(text).To(ContainSubstring("`position=sibling`, `target=<that note's ID>`."))
}

// Test792a15h1c_AnUnrelatedNoteIsTopLevel reads the learn skill the repository ships: a note that matches neither test
// takes position top with no target.
func Test792a15h1c_AnUnrelatedNoteIsTopLevel(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("3. Else → `position=top` (no target)."))
	g.Expect(text).To(ContainSubstring("relative to notes written or recalled earlier in THIS session"))
}

// Test792a15h1d_ContinuationIsCheckedBeforeSibling reads the learn skill the repository ships: the placement tests are
// applied in order, the continuation test first, and each later test starts with "Else", so the first match decides.
func Test792a15h1d_ContinuationIsCheckedBeforeSibling(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Apply this test in order:"))

	first := "1. Does the new note develop **one specific sub-point**"
	second := "2. Else, does it continue/extend the **same overall thought**"
	third := "3. Else → `position=top` (no target)."

	g.Expect(learnPart2Before(text, "Apply this test in order:", first)).To(BeTrue())
	g.Expect(learnPart2Before(text, first, second)).To(BeTrue())
	g.Expect(learnPart2Before(text, second, third)).To(BeTrue())
}

// Test792a15h1e_TheVaultIsNotSearched reads the learn skill the repository ships: with no in-session candidate the
// placement is always top and the rest of the vault is not searched for a target, and no shipped instruction says to
// search the vault for a placement target.
func Test792a15h1e_TheVaultIsNotSearched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("No in-session candidate note exists → always `position=top`; do not search " +
		"the rest of the vault for a placement target."))
	g.Expect(text).To(ContainSubstring("(not a full-vault search)"))
	g.Expect(learnPart2Contradictions(t, learnPart2VaultSearchScan)).To(BeEmpty())
}

// Test792a15i1a_ThePrincipleNotTheInstance reads the learn skill the repository ships: a note states the general
// principle and not the session-specific instance or narrative, and no shipped instruction says to record the
// instance.
func Test792a15i1a_ThePrincipleNotTheInstance(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**State the general principle**, not the session-specific instance — " +
		"future-you recalls by situation similarity, not by remembering this session."))
	g.Expect(text).To(ContainSubstring("a general reusable principle, not a session-specific narrative"))
	g.Expect(learnPart2Contradictions(t, learnPart2InstanceScan)).To(BeEmpty())
}

// Test792a15i1b_OneNotePerPrinciple reads the learn skill the repository ships: each distinct principle gets its own
// note, and a second claim is its own lesson and is never folded into the note beside it.
func Test792a15i1b_OneNotePerPrinciple(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("One note per distinct principle."))
	g.Expect(text).To(ContainSubstring("**Atomic** — one claim per note. A second claim is its own lesson and clears " +
		"both tests itself, or is dropped; it is never folded into the note beside it."))
	g.Expect(text).To(ContainSubstring("| One note holds two claims, or you're about to fold a second claim into a note " +
		"| Atomic — split it: each claim clears both tests alone, or is dropped |"))
}

// Test792a15j1a_AnUnreachableParentIsNotAFailure reads the learn skill the repository ships: the warning "parent
// unreachable ... offer(s) queued" means the write is done, the note is reported as saved and its offer as queued, the
// write is not re-run, and no shipped instruction says to re-run it.
func Test792a15j1a_AnUnreachableParentIsNotAFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("A warning `engram: parent unreachable (retry after …); N offer(s) queued` " +
		"means the write is **done**"))
	g.Expect(text).To(ContainSubstring("Report the note as saved, and that its offer is queued for the parent."))
	g.Expect(text).To(ContainSubstring("Do not re-run the write"))
	g.Expect(text).To(ContainSubstring("| A write printed `parent unreachable … offer(s) queued` and you're " +
		"retrying it, calling it failed,"))
	g.Expect(text).To(ContainSubstring("| The note is saved locally and the offer is queued — report it as done |"))
	g.Expect(learnPart2Contradictions(t, learnPart2RewriteScan)).To(BeEmpty())
}

// Test792a15j1b_EngramServerIsNeverSet reads the learn skill the repository ships: ENGRAM_SERVER is never set, because
// that mode is gone and setting it is a hard error, and no shipped instruction says to set it.
func Test792a15j1b_EngramServerIsNeverSet(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("never set `ENGRAM_SERVER` (that mode is gone; setting it is a hard error)"))
	g.Expect(text).To(ContainSubstring("or reaching for `ENGRAM_SERVER` |"))
	g.Expect(learnPart2Contradictions(t, learnPart2EngramServerScan)).To(BeEmpty())
}

// Test792a15k1a_ACitedAnswerGetsAPair reads the learn skill the repository ships: an answer whose body cites a
// [[full-basename]] wikilink is handed to write-memory as kind qa with the verbatim question, the copied answer body
// and the contributors taken from the wikilinks.
func Test792a15k1a_ACitedAnswerGetsAPair(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("the answer body contains ≥1 `[[full-basename]]` wikilink"))
	g.Expect(text).To(ContainSubstring("with this handoff — kind=qa, slug,"))
	g.Expect(text).To(ContainSubstring("kind=qa, slug, verbatim question, answer body"))
	g.Expect(text).To(ContainSubstring("verbatim question, answer body (copy; no re-derive), contributor basenames"))
	g.Expect(text).To(ContainSubstring("(copy; no re-derive), contributor basenames, certainty, source " +
		"(\"ad-hoc capture, learn session <date>\")."))
	g.Expect(text).To(ContainSubstring("Contributors come ONLY from `[[full-basename]]` wikilinks in the written " +
		"answer — never free-listed."))
}

// Test792a15k1b_GenericAdviceGetsNone reads the learn skill the repository ships: a question answered with generic
// advice, no wikilink and no new note is skipped, and no shipped instruction says to write a pair for generic advice.
func Test792a15k1b_GenericAdviceGetsNone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Skip questions answered with generic advice or without `[[...]]` wikilinks."))
	g.Expect(text).To(ContainSubstring("If no `[[...]]` wikilinks appear in the answer and no note was crystallized, " +
		"skip (D2 bar not met)."))
	g.Expect(learnPart2Contradictions(t, learnPart2GenericAdviceScan)).To(BeEmpty())
}

// Test792a15k1c_APairAlreadyWrittenIsNotWrittenAgain reads the learn skill the repository ships: a QA pair already
// written this session, for example by recall's Step 4, is not written again, and no shipped instruction says to write
// it again.
func Test792a15k1c_APairAlreadyWrittenIsNotWrittenAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**Gate — do not duplicate:** if a QA pair was already written (e.g. by " +
		"recall's Step 4 during this session), do not write it again here."))
	g.Expect(learnPart2Contradictions(t, learnPart2PairAgainScan)).To(BeEmpty())
}

// Test792a15k1d_TheAgentsOwnNoteIsTheCapture reads the learn skill the repository ships: a question answered by a
// Step 2 note written this turn gets no QA pair, whatever wikilinks the note cites, because that note is the capture,
// and no shipped instruction says to write one.
func Test792a15k1d_TheAgentsOwnNoteIsTheCapture(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("if a question was answered by a Step 2 note YOU wrote this turn, that note " +
		"already IS the capture (whatever wikilinks it cites) — do not additionally write a QA pair for the same " +
		"question"))
	g.Expect(text).To(ContainSubstring("| At Step 2.5 you're about to write a QA pair for a question you just answered " +
		"with your OWN Step 2 note this turn | Don't — that note is already the capture"))
	g.Expect(learnPart2Contradictions(t, learnPart2OwnNotePairScan)).To(BeEmpty())
}

// Test792a15k1e_APriorNoteStillEarnsAPair reads the learn skill the repository ships: a question answered by citing an
// existing prior note, with no new Step 2 note written for it this turn, still gets its QA pair.
func Test792a15k1e_APriorNoteStillEarnsAPair(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("The wikilink-based trigger stays live on its own: a question answered by " +
		"citing an EXISTING prior note, with no new Step 2 note written for it this turn, still gets its QA pair."))
}

// Test792a15k1f_ARejectedContributorIsSurfaced reads the learn skill the repository ships: learn does not pre-validate
// contributors, and when write-memory reports a contributor rejection learn surfaces it.
func Test792a15k1f_ARejectedContributorIsSurfaced(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Do NOT pre-validate whether contributors exist in the vault"))
	g.Expect(text).To(ContainSubstring("Validation happens at write time; if write-memory reports a contributor " +
		"rejection, surface it."))
}

// Test792a15l1a_EveryCandidateNoteIsAnsweredOnce reads the learn skill the repository ships: in the re-parent batch
// mode every candidate's note gets exactly one entry, and a note that is a candidate for several targets is answered
// with the one target and position judged most correct.
func Test792a15l1a_EveryCandidateNoteIsAnsweredOnce(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("every candidate's `note` MUST get exactly one entry in the answers file — " +
		"there is no \"skip this one\" option."))
	g.Expect(text).To(ContainSubstring("If a note appears as `note` in more than one candidate (multiple targets above " +
		"the floor), judge each candidate independently and answer with the ONE `target`/`position` you find most " +
		"correct; still exactly one output entry per distinct `note`."))
}

// Test792a15l1b_AHighScoreDoesNotDecide reads the learn skill the repository ships: a candidate is judged by the
// content of its two excerpts and not by the similarity score alone, and an unrelated pair despite a high score takes
// position top.
func Test792a15l1b_AHighScoreDoesNotDecide(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("read `note_excerpt` and `target_excerpt` and judge the content relationship, " +
		"not the `similarity` score alone (a high score can still be an unrelated false positive: two notes can " +
		"share vocabulary while addressing different problems)"))
	g.Expect(text).To(ContainSubstring("3. Else (unrelated despite the similarity score) → `position=top`."))
	g.Expect(text).To(ContainSubstring("a `top` verdict still needs a full answer entry"))
}

// Test792a15l1c_TheFingerprintIsEchoedAndTheNextStepIsTheUsers reads the learn skill the repository ships: the answers
// file echoes the payload's fingerprint verbatim, the user is told to run the command with --dry-run first and then
// without it, learn runs neither command, and no shipped instruction has learn run either.
func Test792a15l1c_TheFingerprintIsEchoedAndTheNextStepIsTheUsers(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, learnPart2SkillPath)

	g.Expect(text).To(ContainSubstring("\"fingerprint\": \"<echoed VERBATIM from the input payload's fingerprint " +
		"field>\""))
	g.Expect(text).To(ContainSubstring("tell the user to run: ``` engram update --reparent-luhmann --answers <file> " +
		"--dry-run ```"))
	g.Expect(text).To(ContainSubstring("to review the resulting renames, then re-run the same command without " +
		"`--dry-run` to apply."))
	g.Expect(text).To(ContainSubstring("This skill does not invoke either command itself — that is the acting " +
		"agent's (or user's) next step."))
	g.Expect(learnPart2Contradictions(t, learnPart2ReparentRunScan)).To(BeEmpty())
}

// TestLearnPart2Contradictions shows the scanner of the absence checks of this file finds an instruction that
// contradicts each rule, and passes the shipped wording that states the rule or forbids the opposite.
func TestLearnPart2Contradictions(t *testing.T) {
	t.Parallel()

	for _, scan := range learnPart2Scans() {
		t.Run(scan.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(learnPart2UnitHits(scan, scan.contradiction)).NotTo(BeEmpty())
			g.Expect(learnPart2UnitHits(scan, scan.conditional)).NotTo(BeEmpty())
			g.Expect(learnPart2UnitHits(scan, scan.stated)).To(BeEmpty())
		})
	}
}

// unexported constants.
const (
	learnPart2SkillPath = "agent-instructions/skills/learn/SKILL.md"
)

// unexported variables.
var (
	// learnPart2ClauseBoundary marks where a clause of a sentence ends, so that a negation in an earlier clause does not
	// reach a later one.
	learnPart2ClauseBoundary   = regexp.MustCompile(`[,;(—]`)
	learnPart2EngramServerScan = learnPart2Scan{
		name: "setting ENGRAM_SERVER",
		pattern: regexp.MustCompile(
			`(?i)\b(?:set|export|use|add|pass|prefix)\b[^.]*\bENGRAM_SERVER\b|ENGRAM_SERVER=`),
		negation:      learnPart2NegationPattern,
		contradiction: "When the parent is unreachable, set ENGRAM_SERVER and run the write again.",
		conditional:   "If the parent can't be reached, set ENGRAM_SERVER and run the write again.",
		stated:        "Report the note as saved, and never set `ENGRAM_SERVER` (that mode is gone).",
	}
	learnPart2GenericAdviceScan = learnPart2Scan{
		name:          "a pair for generic advice",
		pattern:       regexp.MustCompile(`(?i)\b(?:write|capture|hand off)\b[^.]*\bgeneric advice\b`),
		negation:      learnPart2NegationPattern,
		contradiction: "Write a QA pair for a question answered with generic advice as well.",
		conditional:   "When the user isn't satisfied, write a QA pair for a question answered with generic advice.",
		stated:        "Skip questions answered with generic advice or without `[[...]]` wikilinks.",
	}
	learnPart2InstanceScan = learnPart2Scan{
		name: "recording the instance",
		pattern: regexp.MustCompile(
			`(?i)\b(?:state|record|write|capture)\b[^.]*\b(?:narrative|session-specific instance)\b`),
		negation:      learnPart2NegationPattern,
		contradiction: "Record the narrative of the file you fixed in the note.",
		conditional:   "If the note isn't clear enough, record the narrative of the file you fixed in the note.",
		stated:        "State the general principle, not the session-specific instance.",
	}
	// learnPart2NegationPattern marks a sentence that forbids or excludes what it names.
	learnPart2NegationPattern = regexp.MustCompile(`(?i)\b(?:not|never|neither|nor)\b|n't\b`)
	learnPart2OwnNotePairScan = learnPart2Scan{
		name:          "a pair beside the agent's own note",
		pattern:       regexp.MustCompile(`(?i)\b(?:additionally|also|still)\s+write\s+a\s+QA\s+pair\b`),
		negation:      learnPart2NegationPattern,
		contradiction: "Also write a QA pair for a question answered by your own Step 2 note.",
		conditional:   "If the note doesn't cite a wikilink, also write a QA pair for the same question.",
		stated:        "That note already IS the capture — do not additionally write a QA pair for the same question.",
	}
	learnPart2PairAgainScan = learnPart2Scan{
		name: "writing a pair again",
		pattern: regexp.MustCompile(
			`(?i)\bwrite\s+(?:it|the\s+pair|a\s+(?:second|duplicate)\s+(?:QA\s+)?pair)\s+again\b`),
		negation:      learnPart2NegationPattern,
		contradiction: "Even if the pair was already written this session, write it again here.",
		conditional:   "If the first write isn't confirmed, write it again here.",
		stated:        "If a QA pair was already written, do not write it again here.",
	}
	learnPart2PleasantryScan = learnPart2Scan{
		name: "a bare pleasantry qualifying",
		pattern: regexp.MustCompile(
			`(?i)(?:bare\s+(?:pleasantry|thanks)|"thanks!"|great work)[^.]*\b(?:qualif(?:y|ies)|is\s+captured)\b`),
		negation:      learnPart2NegationPattern,
		contradiction: "A bare pleasantry (\"thanks!\", \"great work\") qualifies as a confirmed approach.",
		conditional:   "When the user isn't specific, a bare pleasantry qualifies as a confirmed approach.",
		stated:        "A bare pleasantry naming no behavior (\"thanks!\", \"great work\") does NOT qualify.",
	}
	learnPart2ReparentRunScan = learnPart2Scan{
		name: "learn running the re-parent command",
		pattern: regexp.MustCompile(
			`(?i)\b(?:run|invoke|execute)\b[^.]*\bengram\s+update\s+--reparent-luhmann\b`),
		negation: regexp.MustCompile(
			`(?i)\b(?:not|never|neither|nor)\b|n't\b|\btell\s+the\s+user\s+to\b`),
		contradiction: "Then run `engram update --reparent-luhmann --answers <file>` yourself to apply the answers.",
		conditional:   "If the user isn't available, run `engram update --reparent-luhmann --answers <file>` yourself.",
		stated:        "Tell the user to run: engram update --reparent-luhmann --answers <file> --dry-run",
	}
	learnPart2RewriteScan = learnPart2Scan{
		name:          "re-running a write that queued its offer",
		pattern:       regexp.MustCompile(`(?i)\b(?:re-?run|retry|repeat)\s+the\s+write\b`),
		negation:      learnPart2NegationPattern,
		contradiction: "If the parent is unreachable, re-run the write until the offer goes through.",
		conditional:   "When the write does not reach the parent, re-run the write.",
		stated:        "Do not re-run the write, and never set `ENGRAM_SERVER`.",
	}
	learnPart2RoutineSuccessScan = learnPart2Scan{
		name: "a routine success qualifying",
		pattern: regexp.MustCompile(
			`(?i)routine\s+success[^.]*\b(?:qualif(?:y|ies)|counts?|is\s+captured|gets\s+a\s+note)\b`),
		negation:      learnPart2NegationPattern,
		contradiction: "A routine success also qualifies for a confirmed-approach note.",
		conditional:   "If the user isn't sure, a routine success also qualifies for a confirmed-approach note.",
		stated:        "A routine success does not count as reinforcement-capture.",
	}
	learnPart2RunbookFactScan = learnPart2Scan{
		name:          "a fact note beside a runbook",
		pattern:       regexp.MustCompile(`(?i)\badditionally\b[^.]*\bkind=fact\b`),
		negation:      learnPart2NegationPattern,
		contradiction: "If the user says to do that going forward, additionally fire a kind=fact save-request.",
		conditional:   "If the runbook isn't enough, additionally fire a kind=fact save-request.",
		stated:        "Do NOT additionally fire a kind=fact save-request (kind 2) for the same content.",
	}
	// learnPart2UnitSplit splits collapsed text into sentences and table cells.
	learnPart2UnitSplit       = regexp.MustCompile(`[.!?]\s+|\s\|\s`)
	learnPart2VaultSearchScan = learnPart2Scan{
		name:          "searching the vault for a placement target",
		pattern:       regexp.MustCompile(`(?i)\bsearch\b[^.]*\bvault\b[^.]*\b(?:placement|target)\b`),
		negation:      learnPart2NegationPattern,
		contradiction: "When no in-session candidate exists, search the vault for a placement target.",
		conditional:   "If the candidate isn't found, search the vault for a placement target.",
		stated:        "Do not search the rest of the vault for a placement target.",
	}
)

// learnPart2Scan is one absence scan: a pattern for an instruction that contradicts a rule, the negation pattern that
// excludes a match that forbids it, a contradicting instruction to insert, the same instruction under a negated
// condition and the shipped wording that must pass.
type learnPart2Scan struct {
	name          string
	pattern       *regexp.Regexp
	negation      *regexp.Regexp
	contradiction string
	conditional   string
	stated        string
}

// learnPart2Before reports whether both fragments occur in the text and the first occurs before the second.
func learnPart2Before(text, first, second string) bool {
	firstAt := strings.Index(text, first)
	secondAt := strings.Index(text, second)

	return firstAt >= 0 && secondAt >= 0 && firstAt < secondAt
}

// learnPart2Contradictions returns, by path, the sentences of every shipped SKILL.md, references file and guidance
// document that the scan finds.
func learnPart2Contradictions(t *testing.T, scan learnPart2Scan) map[string][]string {
	t.Helper()

	found := make(map[string][]string)

	for _, path := range aibInstructionPaths(t) {
		hits := learnPart2UnitHits(scan, shippedText(t, path))
		if len(hits) > 0 {
			found[path] = hits
		}
	}

	return found
}

// learnPart2HasUnnegatedMatch reports whether the unit holds a match of the scan's pattern whose own clause, from the
// last clause boundary before the match to the end of the match, carries no negation. A negation elsewhere in the
// sentence, such as a negated condition before the instruction, does not excuse it.
func learnPart2HasUnnegatedMatch(scan learnPart2Scan, unit string) bool {
	for _, span := range scan.pattern.FindAllStringIndex(unit, -1) {
		clauseStart := 0

		for _, boundary := range learnPart2ClauseBoundary.FindAllStringIndex(unit[:span[0]], -1) {
			clauseStart = boundary[1]
		}

		if !scan.negation.MatchString(unit[clauseStart:span[1]]) {
			return true
		}
	}

	return false
}

// learnPart2Scans returns every absence scan of this file.
func learnPart2Scans() []learnPart2Scan {
	return []learnPart2Scan{
		learnPart2EngramServerScan, learnPart2GenericAdviceScan, learnPart2InstanceScan, learnPart2OwnNotePairScan,
		learnPart2PairAgainScan, learnPart2PleasantryScan, learnPart2ReparentRunScan, learnPart2RewriteScan,
		learnPart2RoutineSuccessScan, learnPart2RunbookFactScan, learnPart2VaultSearchScan,
	}
}

// learnPart2UnitHits returns every sentence of the text with a match of the scan's pattern that is not negated.
func learnPart2UnitHits(scan learnPart2Scan, text string) []string {
	hits := make([]string, 0)

	for _, unit := range learnPart2UnitSplit.Split(text, -1) {
		if learnPart2HasUnnegatedMatch(scan, unit) {
			hits = append(hits, unit)
		}
	}

	return hits
}
