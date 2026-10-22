//go:build targ && integration

package dev

import (
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a18g1a_AUserSuppliedPlanIsStillCaptured reads the please skill the repository ships: a plan the user supplied
// is not a reason to skip step 3; it is captured as the planning artifact, reviewed, completed and written down.
func Test792a18g1a_AUserSuppliedPlanIsStillCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	sentence := pleasePart2Between(t, text, "If the user already supplied a plan", pleasePart2GrepAnchor)

	g.Expect(sentence).To(ContainSubstring("do not skip this step — capture their plan as the planning artifact"))
	g.Expect(sentence).To(ContainSubstring("review it"))
	g.Expect(sentence).To(ContainSubstring("fill gaps"))
	g.Expect(sentence).To(ContainSubstring("write it down"))
}

// Test792a18g1b_ThePlanIsCommittedBeforeCode reads the please skill the repository ships: under version control the
// plan is committed, it is not approved until gate A closes, and no code is written before then.
func Test792a18g1b_ThePlanIsCommittedBeforeCode(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	stepThree := pleasePart2Between(t, text, pleasePart2StepThree, pleasePart2StepFour)
	gateA := pleasePart2Between(t, text, "| A | end of step 3", "| B | step 4")

	g.Expect(stepThree).To(ContainSubstring("If the repo is under VCS, commit the plan"))
	g.Expect(stepThree).To(ContainSubstring("The plan is not approved until **gate A** closes"))
	g.Expect(gateA).To(ContainSubstring("end of step 3, before any execution"))
	g.Expect(text).To(ContainSubstring("| You're writing code before the plan is committed (step 3) | " +
		"Stop. Write the plan first, commit it, then execute. |"))
	g.Expect(text).To(ContainSubstring("Do not begin coding (step 4) while the plan (step 3) is still being drafted."))
	g.Expect(pleasePart2Before(stepThree, "If the repo is under VCS, commit the plan", "**gate A** closes")).To(BeTrue())
}

// Test792a18h1a_ARepeatedInvariantChanges reads the please skill the repository ships: a planned change to a repeated
// invariant runs a grep for the term, its synonyms, its hyphenated forms and the old text's echoes, and the plan gets
// the per-file disposition list.
func Test792a18h1a_ARepeatedInvariantChanges(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	grep := pleasePart2GrepSection(t, text)

	g.Expect(grep).To(ContainSubstring("When a planned change alters a *repeated invariant*"))
	g.Expect(grep).To(ContainSubstring("a payload shape, a sweep cadence, a command set, a count, or a naming " +
		"convention echoed across docs, diagrams, or skills"))
	g.Expect(grep).To(ContainSubstring("RUN this grep over the repo before writing the doc-scrub task"))
	g.Expect(grep).To(ContainSubstring("search the term itself"))
	g.Expect(grep).To(ContainSubstring("its synonyms"))
	g.Expect(grep).To(ContainSubstring("its hyphenated or compound forms"))
	g.Expect(grep).To(ContainSubstring("the OLD text's echoes in diagram labels and comments"))
	g.Expect(grep).To(ContainSubstring("Paste the resulting per-file disposition list"))
	g.Expect(grep).To(ContainSubstring("INTO the plan itself"))
}

// Test792a18h1b_ASmallSurfaceStillGreps reads the please skill the repository ships: the enumeration grep is
// non-waivable and a small surface is a cheap grep, never an exemption.
func Test792a18h1b_ASmallSurfaceStillGreps(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	grep := pleasePart2GrepSection(t, text)

	g.Expect(grep).To(ContainSubstring("(non-waivable).**"))
	g.Expect(grep).To(ContainSubstring("a small surface is a cheap grep, never an exemption."))
}

// Test792a18h1c_TheListNamesFileDispositionAndReason reads the please skill the repository ships: each entry of the
// pasted list is a file, a disposition of keep, update, rewrite or N/A, and a one-line reason.
func Test792a18h1c_TheListNamesFileDispositionAndReason(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	grep := pleasePart2GrepSection(t, text)
	entry := pleasePart2Between(t, grep, "(file", "INTO the plan itself")

	g.Expect(grep).To(ContainSubstring("per-file disposition list"))
	g.Expect(entry).To(ContainSubstring("(file → "))

	for _, disposition := range []string{"keep", "update", "rewrite", "N/A"} {
		g.Expect(entry).To(ContainSubstring(disposition))
	}

	g.Expect(entry).To(ContainSubstring("with the one-line reason)"))
}

// Test792a18h1d_AMinimalChangeToAWidelyReferencedConcept reads the please skill the repository ships: the full grep
// runs however large the surface looks, and no shipped instruction lets the cost exempt the change from it.
func Test792a18h1d_AMinimalChangeToAWidelyReferencedConcept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	grep := pleasePart2GrepSection(t, text)

	g.Expect(grep).To(ContainSubstring("RUN this grep over the repo"))
	g.Expect(grep).To(ContainSubstring("The grep runs regardless of how large the surface looks"))
	g.Expect(grep).To(ContainSubstring("its cost scales with the surface"))
	g.Expect(pleasePart2PermissiveSkipHits(t, `\bgrep\b`)).To(BeEmpty())
}

// Test792a18i1a_REDIsNotSkippedAsTooSimple reads the please skill the repository ships: every unit of work starts
// with a repeatable test, and the thought that the unit is too simple to test is a red flag, not a licence.
func Test792a18i1a_REDIsNotSkippedAsTooSimple(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	stepFour := pleasePart2Between(t, text, pleasePart2StepFour, pleasePart2StepFive)

	g.Expect(stepFour).To(ContainSubstring("For each unit of work, follow test-driven development"))
	g.Expect(stepFour).To(ContainSubstring("**RED:** validate the challenge by writing a repeatable test " +
		"(or the closest analogue when the unit is non-code)."))
	g.Expect(pleasePart2Before(stepFour, "**RED:**", "**GREEN:**")).To(BeTrue())
	g.Expect(text).To(ContainSubstring("| You're skipping RED because \"this is too simple to test\" | " +
		"Apply the TDD discipline regardless — with or without a TDD skill installed. |"))
}

// Test792a18i1b_AUnitIsVerifiedByRunningTheCommands reads the please skill the repository ships: a unit is verified
// before it is declared done, by running the actual commands and reading their output.
func Test792a18i1b_AUnitIsVerifiedByRunningTheCommands(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	stepFour := pleasePart2Between(t, text, pleasePart2StepFour, pleasePart2StepFive)

	g.Expect(stepFour).To(ContainSubstring("verify before declaring any unit done"))
	g.Expect(stepFour).To(ContainSubstring("running the actual commands"))
	g.Expect(stepFour).To(ContainSubstring("reading their output before claiming success."))
	g.Expect(text).To(ContainSubstring("| You declared a unit done without running the verifier | " +
		"Verify before claiming done: run the real commands and read the output"))
}

// Test792a18i1c_EachRefactorIsGated reads the please skill the repository ships: each refactor passes gate B, the
// design-fit gate over the refactored diff, before the unit is declared done.
func Test792a18i1c_EachRefactorIsGated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	stepFour := pleasePart2Between(t, text, pleasePart2StepFour, pleasePart2StepFive)
	gateB := pleasePart2Between(t, text, "| B | step 4", "| C | end of step 5")

	g.Expect(stepFour).To(ContainSubstring("Each refactor then passes **gate B** before the unit is declared done."))
	g.Expect(gateB).To(ContainSubstring("the refactored unit's diff"))
}

// Test792a18j1a_TouchedDocsAreUpdatedAndGated reads the please skill the repository ships: step 5 updates every
// documentation piece the change touches and completes only when gate C closes over every touched doc.
func Test792a18j1a_TouchedDocsAreUpdatedAndGated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	stepFive := pleasePart2Between(t, text, pleasePart2StepFive, pleasePart2StepSix)
	gateC := pleasePart2Between(t, text, "| C | end of step 5", "| D | step 6")

	g.Expect(stepFive).To(ContainSubstring("Update every piece of documentation the changes touch"))
	g.Expect(stepFive).To(ContainSubstring("`README.md`, `CLAUDE.md`, `docs/`, glossaries, skill references"))
	g.Expect(stepFive).To(ContainSubstring("so the docs match the new reality."))
	g.Expect(stepFive).To(ContainSubstring("The step completes only when **gate C** closes over every touched doc."))
	g.Expect(gateC).To(ContainSubstring("every doc file touched"))
}

// Test792a18k1a_CompletionCleansUpAndCommits reads the please skill the repository ships: step 6 closes the
// originating issue, deletes temporary artifacts, commits under version control, and passes commit messages and
// outward prose through gate D first.
func Test792a18k1a_CompletionCleansUpAndCommits(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	stepSix := pleasePart2Between(t, text, pleasePart2StepSix, pleasePart2StepSeven)

	g.Expect(stepSix).To(ContainSubstring("If the work originated from an issue, close it."))
	g.Expect(stepSix).To(ContainSubstring("Delete any planning or temporary build/test artifacts created along the way."))
	g.Expect(stepSix).To(ContainSubstring("If the repo is under VCS, stage and commit the changes"))
	g.Expect(stepSix).To(ContainSubstring("Commit messages and any outward prose pass **gate D** before the " +
		"commit/close."))
}

// Test792a18l1a_GateAFansOutFourReviewers reads the please skill the repository ships: gate A fires at the end of
// step 3 over the committed plan with one fresh reviewer per angle for the four angles, sharing no author context.
func Test792a18l1a_GateAFansOutFourReviewers(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	gateA := pleasePart2Between(t, text, "| A | end of step 3", "| B | step 4")

	g.Expect(text).To(ContainSubstring("Each gate fans out ONE fresh-context reviewer subagent PER ANGLE " +
		"(Task tool; no author context shared)"))
	g.Expect(gateA).To(ContainSubstring("the committed plan/spec"))
	g.Expect(gateA).To(ContainSubstring("ask-alignment"))
	g.Expect(gateA).To(ContainSubstring("code-alignment"))
	g.Expect(gateA).To(ContainSubstring("docs/diagrams-alignment"))
	g.Expect(gateA).To(ContainSubstring("clarity/standards"))
}

// Test792a18l1b_GateBFollowsEveryRefactor reads the please skill the repository ships: gate B fires after every
// refactor phase in step 4, over the refactored diff, with a design-fit reviewer.
func Test792a18l1b_GateBFollowsEveryRefactor(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	gateB := pleasePart2Between(t, text, "| B | step 4", "| C | end of step 5")

	g.Expect(gateB).To(ContainSubstring("step 4, after EVERY refactor phase"))
	g.Expect(gateB).To(ContainSubstring("the refactored unit's diff"))
	g.Expect(gateB).To(ContainSubstring("design-fit"))
}

// Test792a18l1c_GateCReviewsEveryTouchedDoc reads the please skill the repository ships: gate C fires at the end of
// step 5 over every documentation file touched, with relevance and clarity/cohesion reviewers.
func Test792a18l1c_GateCReviewsEveryTouchedDoc(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	gateC := pleasePart2Between(t, text, "| C | end of step 5", "| D | step 6")

	g.Expect(gateC).To(ContainSubstring("every doc file touched"))
	g.Expect(gateC).To(ContainSubstring("relevance"))
	g.Expect(gateC).To(ContainSubstring("clarity/cohesion"))
}

// Test792a18l1d_GateDReviewsOutwardProse reads the please skill the repository ships: gate D fires in step 6 before
// the commit or close over commit messages, issue text and any outward prose, with a clarity/standards reviewer.
func Test792a18l1d_GateDReviewsOutwardProse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	gateD := pleasePart2Between(t, text, "| D | step 6", "Angle charges")

	g.Expect(gateD).To(ContainSubstring("step 6, before commit/close"))
	g.Expect(gateD).To(ContainSubstring("commit messages"))
	g.Expect(gateD).To(ContainSubstring("issue text"))
	g.Expect(gateD).To(ContainSubstring("any outward prose"))
	g.Expect(gateD).To(ContainSubstring("clarity/standards"))
}

// Test792a18l1e_ASmallArtifactIsStillGated reads the please skill the repository ships: running a gate is
// non-waivable, and a small or obviously fine artifact is a cheap review, not a skip.
func Test792a18l1e_ASmallArtifactIsStillGated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)

	g.Expect(text).To(ContainSubstring("**Running the gate is non-waivable;"))
	g.Expect(text).To(ContainSubstring("\"The artifact is small\" is never a skip; a small artifact is a cheap review."))
	g.Expect(text).To(ContainSubstring("| You skipped a review gate because the artifact is \"obviously fine\", " +
		"\"tiny\", or \"the reviewer would just agree\" | Run the gate."))
}

// Test792a18l1f_AnglesAreNotMerged reads the please skill the repository ships: the gate runs per angle with a fresh
// reviewer, never a shared reviewer or a skipped angle, and batching the angles into one reviewer to save cost is a
// red flag.
func Test792a18l1f_AnglesAreNotMerged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)

	g.Expect(text).To(ContainSubstring("the gate runs per-angle with a fresh-context reviewer — never a shared " +
		"reviewer or a skipped angle."))
	g.Expect(text).To(ContainSubstring("| You batched all angles into one reviewer to save cost | " +
		"One fresh reviewer per angle"))
	g.Expect(text).To(ContainSubstring("merging angles merges blind spots."))
}

// Test792a18l1g_StaffingIsRouted reads the please skill the repository ships: the reviewer model is routed, not
// pinned; the table models are defaults and the route rubric confirms or adjusts each per angle.
func Test792a18l1g_StaffingIsRouted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)

	g.Expect(text).To(ContainSubstring("the model for each reviewer is not pinned — it is routed."))
	g.Expect(text).To(ContainSubstring("The models in the table below are the rubric's *defaults*, not fixed pins;"))
	g.Expect(text).To(ContainSubstring("apply the `route` rubric per angle to confirm or adjust the default."))
}

// Test792a18m1a_AskAlignmentFindsAGapAndScopeCreep reads the please skill the repository ships: the ask-alignment
// reviewer traces the ask to the plan and the plan back to the ask, and gaps and scope creep are findings.
func Test792a18m1a_AskAlignmentFindsAGapAndScopeCreep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	charge := pleasePart2Between(t, text, "- **ask-alignment:**", "- **code-alignment:**")

	g.Expect(charge).To(ContainSubstring("trace every element of the user's verbatim ask to a plan item AND every " +
		"plan item back to the ask."))
	g.Expect(charge).To(ContainSubstring("Gaps (ask items without coverage)"))
	g.Expect(charge).To(ContainSubstring("scope creep (plan items beyond the ask)"))
	g.Expect(charge).To(ContainSubstring("are findings."))
}

// Test792a18m1b_TheEnumerationListDoesNotNarrowTheScan reads the please skill the repository ships: the
// docs/diagrams-alignment reviewer verifies every disposition of an author-pasted list and still runs its own
// independent discovery pass, so the list never narrows its scan.
func Test792a18m1b_TheEnumerationListDoesNotNarrowTheScan(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)
	charge := pleasePart2Between(t, text, "- **docs/diagrams-alignment:**", "- **clarity/standards:**")

	g.Expect(charge).To(ContainSubstring("carries an author-pasted enumeration-grep disposition list"))
	g.Expect(charge).To(ContainSubstring("VERIFY every listed disposition against the files"))
	g.Expect(charge).To(ContainSubstring("AND still run your own independent discovery pass"))
	g.Expect(charge).To(ContainSubstring("you are never the list's source, and its presence never narrows your scan."))
}

// Test792a18m1c_ACleanPassSaysWhatWasChecked reads the please skill the repository ships: a reviewer is charged to
// refute and not to bless, a clean pass must state what was checked, and no shipped instruction lets a reviewer
// bless an artifact.
func Test792a18m1c_ACleanPassSaysWhatWasChecked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart2SkillPath)

	g.Expect(text).To(ContainSubstring("each reviewer is prompted to REFUTE the artifact, not to bless it"))
	g.Expect(text).To(ContainSubstring("A clean pass must state what was checked."))
	g.Expect(pleasePart2PermissiveBlessHits(t)).To(BeEmpty())
}

// TestPleasePart2PermissiveHits proves that the absence scans of the part 2 tests find an instruction that permits
// what the shipped skill forbids, and do not find the sentences the skill ships.
func TestPleasePart2PermissiveHits(t *testing.T) {
	t.Parallel()

	t.Run("a permission to skip the grep is found", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		pattern := pleasePart2SkipPattern(`\bgrep\b`)

		g.Expect(pattern.MatchString("A small surface may skip the grep.")).To(BeTrue())
		g.Expect(pattern.MatchString("The enumeration grep can be skipped when the change is small.")).To(BeTrue())
	})

	t.Run("an imperative or optional form that skips the grep is found", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		pattern := pleasePart2SkipPattern(`\bgrep\b`)

		g.Expect(pattern.MatchString("It is cheap. Skip the grep when the surface is large.")).To(BeTrue())
		g.Expect(pattern.MatchString("- Omit the grep for a rename.")).To(BeTrue())
		g.Expect(pattern.MatchString("For a widely referenced concept the grep is optional.")).To(BeTrue())
		g.Expect(pattern.MatchString("The grep need not run for a small change.")).To(BeTrue())
	})

	t.Run("a permission to bless is found", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		g.Expect(pleasePart2BlessPattern.MatchString("A reviewer that finds nothing may just bless it.")).To(BeTrue())
		g.Expect(pleasePart2BlessPattern.MatchString("A reviewer can approve the artifact as is.")).To(BeTrue())
	})

	t.Run("an imperative or LGTM form that blesses is found", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		g.Expect(pleasePart2BlessPattern.MatchString("A clean pass is rare. Approve the artifact and stop.")).To(BeTrue())
		g.Expect(pleasePart2BlessPattern.MatchString("- Bless a small diff.")).To(BeTrue())
		g.Expect(pleasePart2BlessPattern.MatchString("A reviewer that finds nothing replies LGTM and stops.")).To(BeTrue())
		g.Expect(pleasePart2BlessPattern.MatchString("A reviewer that finds nothing says \"looks good\".")).To(BeTrue())
	})

	t.Run("the shipped sentences are not found", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		pattern := pleasePart2SkipPattern(`\bgrep\b`)

		g.Expect(pattern.MatchString("a small surface is a cheap grep, never an exemption.")).To(BeFalse())
		g.Expect(pattern.MatchString("Not \"I'll just grep\".")).To(BeFalse())
		g.Expect(pleasePart2BlessPattern.MatchString("not to bless it")).To(BeFalse())
	})
}

// unexported constants.
const (
	pleasePart2ClauseStart = `(?:^|[.!?|:]\s+)(?:[-*]\s+)?(?:\*\*)?`
	pleasePart2GrepAnchor  = "**Doc-surface enumeration grep"
	pleasePart2SkillPath   = "agent-instructions/skills/please/SKILL.md"
	pleasePart2StepFive    = "5. **Document.**"
	pleasePart2StepFour    = "4. **Execute (TDD).**"
	pleasePart2StepSeven   = "7. **Capture (close)"
	pleasePart2StepSix     = "6. **Complete.**"
	pleasePart2StepThree   = "3. **Plan.**"
)

// unexported variables.
var (
	pleasePart2BlessPattern = regexp.MustCompile(`(?im)\b(?:may|can|could|might|(?:is|are) allowed to|(?:ok|okay|fine) ` +
		`to|free to)\s+(?:simply\s+|just\s+)?(?:bless|approve|rubber-stamp|wave through)\b|` +
		pleasePart2ClauseStart + `(?:approve|bless|rubber-stamp|wave through)\b|` +
		`\b(?:reply|replies|respond|responds|say|says|return|returns|answer|answers|write|writes)\s+` +
		`(?:with\s+|just\s+|simply\s+)?["'` + "`" + `*]*(?:lgtm|looks good|approved?)\b|` +
		`(?:looks good|lgtm|no findings)\b[^.]{0,40}\b(?:is|are) (?:enough|sufficient|acceptable|fine)\b`)
)

// pleasePart2Before reports whether both fragments occur in the text and the first occurs before the second.
func pleasePart2Before(text, first, second string) bool {
	firstAt := strings.Index(text, first)
	secondAt := strings.Index(text, second)

	return firstAt >= 0 && secondAt >= 0 && firstAt < secondAt
}

// pleasePart2Between returns the text from the start fragment, included, up to the end fragment, excluded.
func pleasePart2Between(t *testing.T, text, start, end string) string {
	t.Helper()

	startAt := strings.Index(text, start)
	if startAt < 0 {
		t.Fatalf("the shipped text has no %q", start)

		return ""
	}

	endAt := strings.Index(text[startAt:], end)
	if endAt < 0 {
		t.Fatalf("the shipped text has no %q after %q", end, start)

		return ""
	}

	return text[startAt : startAt+endAt]
}

// pleasePart2GrepSection returns the doc-surface enumeration grep paragraph of step 3.
func pleasePart2GrepSection(t *testing.T, text string) string {
	t.Helper()

	return pleasePart2Between(t, text, pleasePart2GrepAnchor, "If the repo is under VCS, commit the plan")
}

// pleasePart2PermissiveBlessHits returns every place in a shipped SKILL.md, references file or guidance document that
// lets a reviewer bless or approve an artifact without refuting it.
func pleasePart2PermissiveBlessHits(t *testing.T) []string {
	t.Helper()

	return pleasePart2ScanShipped(t, pleasePart2BlessPattern)
}

// pleasePart2PermissiveSkipHits returns every place in a shipped SKILL.md, references file or guidance document that
// permits skipping, omitting or waiving the subject.
func pleasePart2PermissiveSkipHits(t *testing.T, subject string) []string {
	t.Helper()

	return pleasePart2ScanShipped(t, pleasePart2SkipPattern(subject))
}

// pleasePart2ScanShipped returns path and match for every match of the pattern in a shipped instruction text.
func pleasePart2ScanShipped(t *testing.T, pattern *regexp.Regexp) []string {
	t.Helper()

	hits := make([]string, 0)

	for _, path := range aibInstructionPaths(t) {
		for _, match := range pattern.FindAllString(shippedText(t, path), -1) {
			hits = append(hits, path+": "+match)
		}
	}

	return hits
}

// pleasePart2SkipPattern returns the pattern of a sentence that permits skipping, omitting or waiving the subject: a
// permission verb before the subject, the subject before a "may be skipped" form, a sentence that opens with the
// imperative, or the subject before "is optional".
func pleasePart2SkipPattern(subject string) *regexp.Regexp {
	const (
		permission = `(?:may|can|could|might|(?:is|are) allowed to|(?:ok|okay|fine) to|free to|permitted to)`
		omission   = `(?:skip|skipped|omit|omitted|waive|waived|drop|dropped|ignore|ignored|bypass|bypassed)`
		imperative = `(?:skip|omit|waive|drop|ignore|bypass)`
		optional   = `(?:(?:is|are) (?:optional|not required|not needed|unnecessary)|need not (?:run|be run))`
	)

	return regexp.MustCompile(`(?im)\b` + permission + `\s+(?:simply\s+|safely\s+|just\s+)?(?:be\s+)?` + omission +
		`\b[^.]{0,80}?` + subject + `|` + subject + `[^.]{0,80}?\b(?:may|can|could|might)\s+(?:simply\s+|safely\s+)?` +
		`be\s+` + omission + `\b|` + pleasePart2ClauseStart + imperative + `\b[^.]{0,80}?` + subject + `|` +
		subject + `[^.]{0,80}?\b` + optional)
}
