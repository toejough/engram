//go:build targ && integration

package dev

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a14n1a_TheAskedTaskStaysFirst reads the recall skill the repository ships: a recommendation the work
// generated is reconciled against the asked task, defaults to the asked task, and deviates only by naming a new fact
// and stating a reversal; no shipped instruction puts a prerequisite or a better test ahead of the asked task.
func Test792a14n1a_TheAskedTaskStaysFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	synthesis := recallPart3Section(t, text, recallPart3Step3Heading, recallPart3Step35Heading)

	g.Expect(synthesis).To(ContainSubstring("Before stating any option, lever, or recommendation the work generated, " +
		"reconcile it against the ASKED TASK and decisions already in context."))
	g.Expect(synthesis).To(ContainSubstring("Default to the asked task;"))
	g.Expect(synthesis).To(ContainSubstring("deviate only by (a) naming a genuinely NEW fact"))
	g.Expect(synthesis).To(ContainSubstring("and (b) stating you are reversing direction."))
	g.Expect(synthesis).To(ContainSubstring("Substituting a prerequisite, a better test, or a \"more rigorous " +
		"foundation\" as the thing to do *first*"))
	g.Expect(synthesis).To(ContainSubstring("IS relitigating the settled task."))
	g.Expect(recallPart3Contradictions(t, []string{
		"do the prerequisite before the asked task",
		"put the prerequisite before the asked task",
		"start with the more rigorous foundation",
		"propose the better test first",
	})).To(BeEmpty())
}

// Test792a14n1b_OldReasoningIsNotNewEvidence reads the recall skill the repository ships: reasoning already in context
// is not new evidence, so it is not the new fact that licenses a deviation; no shipped instruction says otherwise.
func Test792a14n1b_OldReasoningIsNotNewEvidence(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	synthesis := recallPart3Section(t, text, recallPart3Step3Heading, recallPart3Step35Heading)

	g.Expect(synthesis).To(ContainSubstring("Reasoning already in context is not new evidence"))
	g.Expect(synthesis).To(ContainSubstring("reconcile against prior OUTCOMES, not mere mentions."))
	g.Expect(synthesis).To(ContainSubstring("deviate only by (a) naming a genuinely NEW fact"))
	g.Expect(text).To(ContainSubstring("old reasoning isn't new evidence. Do the asked task; displace only on a NEW " +
		"fact, stated as a reversal"))
	g.Expect(recallPart3Contradictions(t, []string{
		"reasoning already in context is new evidence",
		"earlier reasoning counts as new evidence",
		"your own earlier reasoning is a new fact",
		"in-context reasoning counts as the new fact",
	})).To(BeEmpty())
}

// Test792a14o1a_ANewRecommendationTriggersAReentryQuery reads the recall skill the repository ships: a recommendation
// not in the Step 0 plan is checked, before the synthesis is shown, by one more engram query keyed to the
// recommendation itself with exactly three phrases.
func Test792a14o1a_ANewRecommendationTriggersAReentryQuery(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	reentry := recallPart3Section(t, text, recallPart3Step35Heading, recallPart3Step4Heading)

	g.Expect(reentry).To(ContainSubstring("Before the Step 3 synthesis is shown to the user:"))
	g.Expect(reentry).To(ContainSubstring("a recommendation, lever, or approach — named as the thing to do — that " +
		"does **not** appear in the Step-0 plan"))
	g.Expect(reentry).To(ContainSubstring("run ONE more query first, keyed to the recommendation itself, not the " +
		"original ask:"))
	g.Expect(reentry).To(ContainSubstring("engram query --lazy-chunks"))
	g.Expect(reentry).To(ContainSubstring("--phrase=\"<the recommendation, in its own words>\""))
	g.Expect(reentry).To(ContainSubstring("--phrase=\"<the recommendation> rolled back rejected not worth it " +
		"superseded\""))
	g.Expect(reentry).To(ContainSubstring("--phrase=\"<the recommendation> tried measured outcome\""))
	g.Expect(strings.Count(reentry, "--phrase=")).To(Equal(recallPart3ReentryPhraseCount))
	g.Expect(reentry).To(ContainSubstring("Apply Step 2.5B's recency weight to what returns."))
	g.Expect(reentry).To(ContainSubstring("Runs in both modes (a query is a read; glance keeps it)."))
	g.Expect(reentry).To(ContainSubstring("Skipping the re-entry when the trigger fires is forbidden."))
}

// Test792a14o1b_TheLineSitsAboveTheRecommendation reads the recall skill the repository ships: the clean Re-entry line
// carries the recommendation and the best-match basename or "no items", and goes directly above the final
// recommendation statement.
func Test792a14o1b_TheLineSitsAboveTheRecommendation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	reentry := recallPart3Section(t, text, recallPart3Step35Heading, recallPart3Step4Heading)

	g.Expect(reentry).To(ContainSubstring("The synthesis MUST carry one `Re-entry:` line per emergent recommendation"))
	g.Expect(reentry).To(ContainSubstring("Re-entry: <recommendation> — clean (top hit: <best-match basename, or " +
		"\"no items\">)"))
	g.Expect(reentry).To(ContainSubstring("The Re-entry line(s) go **directly above the final recommendation " +
		"statement** in your reply"))
}

// Test792a14o1c_AClosedRecommendationIsDropped reads the recall skill the repository ships: a note recording a prior
// outcome makes the line CLOSED with that outcome and drop, the recommendation is withdrawn, and the synthesis says
// what replaces it.
func Test792a14o1c_AClosedRecommendationIsDropped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	reentry := recallPart3Section(t, text, recallPart3Step35Heading, recallPart3Step4Heading)

	g.Expect(reentry).To(ContainSubstring("Re-entry: <recommendation> — CLOSED (<note basename>): <one-line prior " +
		"outcome> → drop"))
	g.Expect(reentry).To(ContainSubstring("`CLOSED → drop` means the recommendation is withdrawn"))
	g.Expect(reentry).To(ContainSubstring("and the synthesis says what replaces it."))
}

// Test792a14o1d_ARevisitNeedsNewEvidence reads the recall skill the repository ships: revisit is valid only with
// named new evidence that did not exist when the note was written, and re-weighing the note's own tradeoffs is not
// new evidence; no shipped instruction says otherwise.
func Test792a14o1d_ARevisitNeedsNewEvidence(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	reentry := recallPart3Section(t, text, recallPart3Step35Heading, recallPart3Step4Heading)

	g.Expect(reentry).To(ContainSubstring("→ revisit because <named NEW evidence>"))
	g.Expect(reentry).To(ContainSubstring("`revisit` is valid ONLY with named NEW evidence"))
	g.Expect(reentry).To(ContainSubstring("evidence that did not exist when the note was written;"))
	g.Expect(reentry).To(ContainSubstring("re-weighing the note's own tradeoffs is not new evidence."))
	g.Expect(recallPart3Contradictions(t, []string{
		"re-weighing the note's own tradeoffs is new evidence",
		"re-weighing the note's own tradeoffs counts as new evidence",
		"revisit is valid without new evidence",
		"revisit without new evidence",
	})).To(BeEmpty())
}

// Test792a14o1e_ACitedReentryNoteIsActivated reads the recall skill the repository ships: a note the re-entry query
// surfaced and the agent cites goes into the Step 2.7 activation call, which runs again if it already ran.
func Test792a14o1e_ACitedReentryNoteIsActivated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	reentry := recallPart3Section(t, text, recallPart3Step35Heading, recallPart3Step4Heading)

	g.Expect(reentry).To(ContainSubstring("If you cite a note this query surfaced, include it in the Step 2.7 " +
		"`engram activate` call"))
	g.Expect(reentry).To(ContainSubstring("(run it again if 2.7 already ran)."))
	g.Expect(text).To(ContainSubstring("### Step 2.7 — Activation (use-driven, after synthesis)"))
}

// Test792a14o1f_ARecommendationWithoutItsLineViolatesTheStep reads the recall skill the repository ships: a reply with
// a RECOMMENDATION: line and no Re-entry line directly above it violates the step, a recommendation may not ship
// without its line, and the red-flag table repeats the rule.
func Test792a14o1f_ARecommendationWithoutItsLineViolatesTheStep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	reentry := recallPart3Section(t, text, recallPart3Step35Heading, recallPart3Step4Heading)
	redFlags := recallPart3Section(t, text, recallPart3RedFlagsHeading, "")

	g.Expect(reentry).To(ContainSubstring("A reply that states a recommendation (a `RECOMMENDATION:` line"))
	g.Expect(reentry).To(ContainSubstring("with no Re-entry line directly above it violates this step."))
	g.Expect(reentry).To(ContainSubstring("A recommendation may not ship without its Re-entry line."))
	g.Expect(redFlags).To(ContainSubstring("| You wrote a recommendation line with no `Re-entry:` line directly above " +
		"it | Step 3.5 — run the query, write the Re-entry line(s) directly above the recommendation |"))
}

// Test792a14p1a_ASoundNewConclusionIsHandedOff reads the recall skill the repository ships: a sound, non-trivial
// conclusion no existing note states is handed to the write-memory skill as one fact or feedback note per conclusion,
// whose source is marked derived.
func Test792a14p1a_ASoundNewConclusionIsHandedOff(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	persist := recallPart3Section(t, text, recallPart3Step4Heading, recallPart3RedFlagsHeading)

	g.Expect(persist).To(ContainSubstring("a **sound, non-trivial conclusion that no existing note states**"))
	g.Expect(persist).To(ContainSubstring("Hand ONE synthesis note per conclusion to the **write-memory** skill " +
		"(kind=fact or feedback, per the conclusion's shape)"))
	g.Expect(persist).To(ContainSubstring("**The note IS the conclusion** (the reasoned lesson), phrased as such — " +
		"not a restatement of an input."))
	g.Expect(persist).To(ContainSubstring("**Mark it as derived** in `--source`, e.g. " +
		"`--source=\"synthesis (abduction) from recalled memory\"`"))
}

// Test792a14p1b_AnAbductionIsMarkedProbable reads the recall skill the repository ships: an abduction or induction is
// marked probable, a best explanation or defeasible, never certain, while a deduction is stated as following
// necessarily.
func Test792a14p1b_AnAbductionIsMarkedProbable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	persist := recallPart3Section(t, text, recallPart3Step4Heading, recallPart3RedFlagsHeading)

	g.Expect(persist).To(ContainSubstring("**Certainty by inference mode:**"))
	g.Expect(persist).To(ContainSubstring("deduction → state it as following necessarily;"))
	g.Expect(persist).To(ContainSubstring("**abduction / induction → mark it _probable / best-explanation / " +
		"defeasible_**"))
	g.Expect(persist).To(ContainSubstring("never as certain."))
}

// Test792a14p1c_AHunchIsNotPersisted reads the recall skill the repository ships: a hunch or a conclusion hedged
// below probable is not persisted; no shipped instruction says to persist one.
func Test792a14p1c_AHunchIsNotPersisted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	persist := recallPart3Section(t, text, recallPart3Step4Heading, recallPart3RedFlagsHeading)

	g.Expect(persist).To(ContainSubstring("**Gate — do not rot the vault (notes 68/69):** persist ONLY conclusions " +
		"you judge sound."))
	g.Expect(persist).To(ContainSubstring("If it is a hunch, you'd hedge below \"probable\""))
	g.Expect(persist).To(ContainSubstring("do NOT persist."))
	g.Expect(recallPart3Contradictions(t, []string{
		"persist a hunch",
		"persist hunches",
		"persist a conclusion you would hedge below",
		"a hunch may be persisted",
	})).To(BeEmpty())
}

// Test792a14p1d_AGlanceRecallPersistsNothing reads the recall skill the repository ships: glance skips Step 4, the
// write side, and is read-only with respect to vault knowledge; no shipped instruction has glance persist a
// conclusion.
func Test792a14p1d_AGlanceRecallPersistsNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	persist := recallPart3Section(t, text, recallPart3Step4Heading, recallPart3RedFlagsHeading)
	redFlags := recallPart3Section(t, text, recallPart3RedFlagsHeading, "")

	g.Expect(persist).To(ContainSubstring("> **[glance: SKIP Step 4 — write side. Escalate to `deep` if this " +
		"decision is worth crystallizing.]**"))
	g.Expect(text).To(ContainSubstring("but **skip the write side**: Step 2.5C (coverage amend/learn), Step 4 " +
		"(synthesis-persist)."))
	g.Expect(text).To(ContainSubstring("it does **not** grow the vault's knowledge."))
	g.Expect(redFlags).To(ContainSubstring("| You ran the write side (2.5C/Step 4) while in `glance` mode | Glance is " +
		"read-only w.r.t. vault knowledge — skip the write side; switch to `deep` if you need to crystallize |"))
	g.Expect(recallPart3Contradictions(t, []string{
		"in glance mode, persist the",
		"glance persists",
		"glance may write a synthesis note",
		"run step 4 in glance",
	})).To(BeEmpty())
}

// Test792a14p1e_CitationsAreTypedInline reads the recall skill the repository ships: the agent hand-types
// [[full-basename]] citations in the synthesis body and never hand-authors a vocab tag or a Supersedes backlink, which
// the binary writes.
func Test792a14p1e_CitationsAreTypedInline(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	persist := recallPart3Section(t, text, recallPart3Step4Heading, recallPart3RedFlagsHeading)

	g.Expect(persist).To(ContainSubstring("**Never hand-author a `vocab/<term>` tag or a `Supersedes:` backlink**"))
	g.Expect(persist).To(ContainSubstring("the binary writes both automatically."))
	g.Expect(persist).To(ContainSubstring("This does NOT restrict inline `[[full-basename]]` prose citations in the " +
		"synthesis body:"))
	g.Expect(persist).To(ContainSubstring("hand-type one directly in the body whenever it cites an input note's claim"))
	g.Expect(recallPart3Contradictions(t, []string{
		"write a `supersedes:` line yourself",
		"add a `supersedes:` line to the body",
		"add a vocab tag to the note yourself",
		"write the vocab tag yourself",
	})).To(BeEmpty())
}

// Test792a14q1a_ACitedSynthesisGetsAQANote reads the recall skill the repository ships: a synthesis body with at least
// one [[full-basename]] wikilink is followed by a write-memory handoff of kind qa with the verbatim question, the
// conclusion as the answer, a matching certainty, the wikilink basenames as contributors and the Step 4 source.
func Test792a14q1a_ACitedSynthesisGetsAQANote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	persist := recallPart3Section(t, text, recallPart3Step4Heading, recallPart3RedFlagsHeading)
	capture := recallPart3Section(t, persist, recallPart3QACaptureStart, "")

	g.Expect(capture).To(ContainSubstring("if the synthesis body contains ≥1 `[[full-basename]]` wikilink,"))
	g.Expect(capture).To(ContainSubstring("ALSO invoke the **write-memory** skill** with kind=qa"))
	g.Expect(capture).To(ContainSubstring("verbatim question,"))
	g.Expect(capture).To(ContainSubstring("the synthesis conclusion as the answer,"))
	g.Expect(capture).To(ContainSubstring("certainty matching the synthesis note's label,"))
	g.Expect(capture).To(ContainSubstring("contributors = the wikilink basenames,"))
	g.Expect(capture).To(ContainSubstring("source \"recall Step 4, session <date>\"."))
	g.Expect(capture).To(ContainSubstring("Contributors are auto-extracted from the `[[full-basename]]` wikilinks you " +
		"already wrote in the synthesis body — do NOT free-list"))
}

// Test792a14q1b_AnUncitedSynthesisGetsNone reads the recall skill the repository ships: a synthesis body with no
// wikilink skips the qa capture; no shipped instruction captures one anyway.
func Test792a14q1b_AnUncitedSynthesisGetsNone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart3SkillPath)
	persist := recallPart3Section(t, text, recallPart3Step4Heading, recallPart3RedFlagsHeading)
	capture := recallPart3Section(t, persist, recallPart3QACaptureStart, "")

	g.Expect(capture).To(ContainSubstring("If the synthesis body contains no wikilinks, skip the QA capture"))
	g.Expect(capture).To(ContainSubstring("(D2 bar: ≥1 citation required)."))
	g.Expect(recallPart3Contradictions(t, []string{
		"capture the qa note even without a wikilink",
		"capture qa even without wikilinks",
		"even if the synthesis body has no wikilink, capture",
		"even when the synthesis body contains no wikilinks, invoke",
	})).To(BeEmpty())
}

// unexported constants.
const (
	recallPart3QACaptureStart     = "**After the synthesis note:"
	recallPart3RedFlagsHeading    = "## Red flags — STOP and re-read"
	recallPart3ReentryPhraseCount = 3
	recallPart3SkillPath          = "agent-instructions/skills/recall/SKILL.md"
	recallPart3Step35Heading      = "### Step 3.5 — Re-entry: a recommendation not in the Step-0 plan"
	recallPart3Step3Heading       = "### Step 3 — Closing synthesis: did the memories change the plan?"
	recallPart3Step4Heading       = "### Step 4 — Persist the reasoned conclusion (linked to the inputs that produced it)"
)

// recallPart3Contradictions returns, for every shipped instruction (each SKILL.md, each references file and each
// guidance document), the forbidden phrases the instruction contains, compared in lower case with white space
// collapsed. It fails the test when the scan did not cover the recall skill, so an empty result is not vacuous.
func recallPart3Contradictions(t *testing.T, forbidden []string) []string {
	t.Helper()

	paths := aibInstructionPaths(t)
	hits := make([]string, 0, len(forbidden))
	scannedRecall := false

	for _, path := range paths {
		if path == recallPart3SkillPath {
			scannedRecall = true
		}

		lowered := strings.ToLower(shippedText(t, path))

		for _, phrase := range forbidden {
			if strings.Contains(lowered, phrase) {
				hits = append(hits, path+": "+phrase)
			}
		}
	}

	if !scannedRecall {
		t.Fatalf("the instruction scan did not cover %s", recallPart3SkillPath)
	}

	return hits
}

// recallPart3Section returns the text from the start fragment up to the end fragment, or to the end of the text when
// the end is empty. It fails the test when a fragment is missing or the end does not follow the start.
func recallPart3Section(t *testing.T, text, start, end string) string {
	t.Helper()

	_, section, found := strings.Cut(text, start)
	if !found {
		t.Fatalf("section start %q not found", start)

		return ""
	}

	section = start + section
	if end == "" {
		return section
	}

	section, _, found = strings.Cut(section, end)
	if !found {
		t.Fatalf("section end %q not found after %q", end, start)

		return ""
	}

	return section
}
