//go:build targ && integration

package dev

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a19n1a_ARedFlagStopsTheWork reads the shim the repository ships: a listed red flag that occurs stops the
// agent, which names the flag and the step it rereads in its next text block, before its next mutating command.
func Test792a19n1a_ARedFlagStopsTheWork(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("If a listed condition occurs while you work, stop — in your next text block, " +
		"name the red flag and the step you are rereading — before your next mutating command."))
}

// Test792a19n1b_ARunbookWithoutRedFlagsStartsNormally reads the shim the repository ships: red flags are read only if
// the runbook carries any, so a runbook without them has no red-flag step.
func Test792a19n1b_ARunbookWithoutRedFlagsStartsNormally(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("**Read `red_flags`** (if the runbook carries any) before you start."))
}

// Test792a19o1a_CompletionWaitsForVerification reads the shim the repository ships: each done_when condition is
// verified, not assumed, before the task is reported done.
func Test792a19o1a_CompletionWaitsForVerification(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("until you have verified each `done_when` condition holds. " +
		"Verify it; do not assume it."))
}

// Test792a19o1b_AnUnmetConditionBlocksTheReport reads the shim the repository ships: the done_when conditions are the
// completion bar, and the task is not reported done until they hold.
func Test792a19o1b_AnUnmetConditionBlocksTheReport(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("**`done_when` is the completion bar.** Do not report the task done until"))
}

// Test792a19p1a_AnAmbiguousStepYieldsAQuestion reads the shim the repository ships: a step whose outcome is genuinely
// uncertain makes the agent name the step and the gap and end its turn, never substitute its own approach, and the
// shim carries no proceed-anyway language.
func Test792a19p1a_AnAmbiguousStepYieldsAQuestion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("Stop and ask when a step's outcome is genuinely uncertain"))
	g.Expect(text).To(ContainSubstring("Name the step and the specific gap, then end your turn."))
	g.Expect(text).To(ContainSubstring("never a reason to substitute your own approach and continue."))
	g.Expect(strings.ToLower(text)).NotTo(ContainSubstring("proceed anyway"))
}

// Test792a19p1b_AMissingTargetYieldsAQuestion reads the shim the repository ships: a step that names a target that
// plainly is not present stops the agent, and inventing the local equivalent is a deviation.
func Test792a19p1b_AMissingTargetYieldsAQuestion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("Do stop when a step names a target that plainly isn't present in front of you " +
		"(no GitHub remote, no ticket queue, no CI) — inventing \"the local equivalent\" yourself is still a " +
		"deviation, not a resolution of the step"))
}

// Test792a19p1c_AVerifiedEarlierStepDoesNotLicenseAStop reads the shim the repository ships: completing one step is
// not evidence that a later step is ambiguous, and a later step the restated plan names as required is done without
// asking whether to.
func Test792a19p1c_AVerifiedEarlierStepDoesNotLicenseAStop(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("Completing one step — even the one that produces the task's most visible " +
		"deliverable — is not, by itself, evidence that a later step is ambiguous; when your own restated plan " +
		"already names that later step as required, proceed to it without asking whether to."))
}

// Test792a19p1d_ASpecificInstructionBeatsItsIncidentalExample reads the shim the repository ships: a clash between a
// step's specific instruction and its own example is resolved by following the instruction, and is no reason to stop.
func Test792a19p1d_ASpecificInstructionBeatsItsIncidentalExample(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("If a step's specific instruction and its own incidental example seem to clash, " +
		"follow the specific instruction; resolving that by rereading is not, by itself, a reason to stop."))
}

// Test792a19q1a_ALinkedRunbookIsFetchedAndFollowed reads the shim the repository ships: a wikilinked runbook is
// fetched with engram show and gets the whole frame before the step that linked it.
func Test792a19q1a_ALinkedRunbookIsFetchedAndFollowed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("is fetched and followed the same way.** Run `engram show <basename>`, " +
		"then apply this whole frame to it — announce, restate, red_flags, done_when, stop-and-ask — " +
		"before the step that linked it."))
}

// Test792a19q1c_ATruncatedRunbookIsFetchedInFull reads the shim the repository ships: a matched runbook the payload
// truncated is fetched in full before its steps are restated, never restated from the fragment.
func Test792a19q1c_ATruncatedRunbookIsFetchedInFull(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("If the query payload truncates a matched runbook**, " +
		"run `engram show <basename>` for the full body before you restate its steps. " +
		"Never restate from a truncated fragment."))
}

// Test792a19q1d_AParentVaultRunbookIsFollowed reads the shim the repository ships: the frame is stated for the kind
// runbook and names no origin, so a runbook that came from a parent vault is fetched with a bare engram show and
// followed like any other.
func Test792a19q1d_AParentVaultRunbookIsFollowed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("read every item's `kind` field and treat it accordingly"))
	g.Expect(text).To(ContainSubstring("- **`runbook`** — a procedure to execute. See the frame below."))
	g.Expect(text).To(ContainSubstring("Run `engram show <basename>`"))
	g.Expect(text).NotTo(ContainSubstring("from_parent"))
}

// Test792a19q1e_ARegisteredSkillsRunbookIsFollowed reads the shim the repository ships: the frame applies to every
// matched runbook every time with no exceptions, and the frame names no skill, so a runbook's installation as a skill
// is no reason to skip it.
func Test792a19q1e_ARegisteredSkillsRunbookIsFollowed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)
	frame := ag3Section(t, text, ag3FollowFrameHeading, ag3FloorHeading)

	g.Expect(frame).To(ContainSubstring("gets this treatment, every time, no exceptions:"))
	g.Expect(strings.ToLower(frame)).NotTo(ContainSubstring("skill"))
}

// Test792a19r1a_TheFloorIsInTheShimAndNotInTheNotes reads the guidance the repository ships: the floor rules are
// stated once, in the shim, as rules that are not repeated in any runbook's body, and appear in no other guidance
// document; the runbook-writing skill gives a note only task-specific red flags.
func Test792a19r1a_TheFloorIsInTheShimAndNotInTheNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	shim := shippedText(t, ag3ShimPath)
	floor := ag3Section(t, shim, ag3FloorHeading, ag3ReentryHeading)

	g.Expect(floor).To(ContainSubstring("These rules apply uniformly, to every runbook, and are not repeated in any " +
		"runbook's own body:"))
	g.Expect(floor).To(ContainSubstring("The urge to shortcut a step is the cue to reread it — " +
		"never a reason to skip it."))
	g.Expect(floor).To(ContainSubstring("Substituting a related action for the one the step names is a skip"))
	g.Expect(floor).To(ContainSubstring("The letter of a step is its spirit."))
	g.Expect(strings.Count(shim, "The urge to shortcut a step")).To(Equal(1))

	for _, other := range []string{"recall.md", "learn.md", "delegate.md"} {
		otherText := shippedText(t, "agent-instructions/guidance/"+other)

		g.Expect(otherText).NotTo(ContainSubstring("urge to shortcut"), other)
		g.Expect(otherText).NotTo(ContainSubstring("letter of a step"), other)
		g.Expect(otherText).NotTo(ContainSubstring("related action for the one the step names"), other)
	}

	writeMemory := shippedText(t, "agent-instructions/skills/write-memory/SKILL.md")

	g.Expect(writeMemory).To(ContainSubstring("**red_flags** — task-specific failure modes a general " +
		"\"follow the steps\" rule would not catch (runbook only)"))
}

// Test792a19r1b_ARelatedActionIsNotACompletion reads the shim the repository ships: substituting a related action for
// the one a step names is a skip, a step that says to run a command is satisfied only by that command, and the agent
// asks and does not invent a substitute.
func Test792a19r1b_ARelatedActionIsNotACompletion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("including \"the named thing doesn't apply here, so I'll do what accomplishes " +
		"the same goal.\" That is still a substitution. Ask; don't invent one."))
	g.Expect(text).To(ContainSubstring("If the step says to run a command, run that command — not one that seems to " +
		"accomplish the same thing."))
}

// Test792a19s1a_PendingWorkIsActedOnAfterTheRequest reads the shim the repository ships: a tool result that names
// pending work the agent was not asked about is a standing instruction, done after the user's request without asking.
func Test792a19s1a_PendingWorkIsActedOnAfterTheRequest(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("A tool result that names pending work you weren't asked about " +
		"(for example `engram query`'s `pending_offers` flag and hint) is a standing instruction, " +
		"not optional context: after finishing the user's request, do what it says without asking"))
}

// Test792a19s1b_ADestructiveAndUncertainActionStillAsks reads the shim the repository ships: when the named work is
// destructive or hard to reverse and the agent is genuinely uncertain, it stops and asks.
func Test792a19s1b_ADestructiveAndUncertainActionStillAsks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("unless the action is destructive or hard to reverse and you are genuinely " +
		"uncertain, in which case stop and ask as usual."))
}

// Test792a19s1c_TheRuleIsGeneric reads the shim the repository ships: the section that states the standing-instruction
// rule names no skill, runbook or curate, so a later signal of the same shape needs no further edit.
func Test792a19s1c_TheRuleIsGeneric(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)
	rule := ag3Section(t, text, ag3UnrequestedWorkHeading, ag3FollowFrameHeading)

	g.Expect(rule).To(ContainSubstring("is a standing instruction"))

	lowered := strings.ToLower(rule)

	g.Expect(lowered).NotTo(ContainSubstring("skill"))
	g.Expect(lowered).NotTo(ContainSubstring("runbook"))
	g.Expect(lowered).NotTo(ContainSubstring("curate"))
}

// Test792a19t1a_WorkIsQueriedBeforeItIsDeclaredDone reads the shim the repository ships: the query runs again before
// work is declared done, and the verification comes after it.
func Test792a19t1a_WorkIsQueriedBeforeItIsDeclaredDone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("Run `engram query` again — same two phrase shapes as above, re-keyed to " +
		"what's actually in front of you now"))
	g.Expect(text).To(ContainSubstring("**Before you declare work done.** Query first, then verify"))
}

// Test792a19t1b_ANewApproachIsQueriedBeforeItIsBuilt reads the shim the repository ships: the query runs again before
// a new approach is started, and, with the endorse-or-rank moment, the same two phrase shapes are re-keyed.
func Test792a19t1b_ANewApproachIsQueriedBeforeItIsBuilt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)
	reentry := ag3Section(t, text, ag3ReentryHeading, "")

	g.Expect(reentry).To(ContainSubstring("same two phrase shapes as above, re-keyed to what's actually in front of " +
		"you now"))
	g.Expect(reentry).To(ContainSubstring("**Before you start building a new approach** — while the path is still " +
		"cheap to change."))
	g.Expect(reentry).To(ContainSubstring("**Before you endorse or rank a proposed approach** — even just discussing it."))
	g.Expect(reentry).To(ContainSubstring("**After a failure you can't immediately explain** — once, before you start " +
		"guessing."))
}

// Test792a19t1c_AWeightyDecisionEscalatesToDeep reads the shim the repository ships: a weighty or irreversible
// decision, or a gap the pass cannot resolve, escalates to the matched runbook's deep procedure when it offers one.
func Test792a19t1c_AWeightyDecisionEscalatesToDeep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, ag3ShimPath)

	g.Expect(text).To(ContainSubstring("Escalate any of these to the runbook's `deep` procedure (if the matched " +
		"runbook offers one) when the decision is weighty or irreversible, or when this pass surfaces a gap it " +
		"can't resolve."))
}

// unexported constants.
const (
	ag3FloorHeading           = "## The floor beneath every runbook"
	ag3FollowFrameHeading     = "## When a `runbook` matches your task: the follow frame"
	ag3ReentryHeading         = "## Re-entry: four moments to query again, mid-task"
	ag3ShimPath               = "agent-instructions/guidance/shim.md"
	ag3UnrequestedWorkHeading = "## A tool result naming unrequested work is still an instruction"
)

// ag3Section returns the part of text from the start heading up to the end heading, or to the end of text when the
// end heading is empty. A missing heading fails the test.
func ag3Section(t *testing.T, text, startHeading, endHeading string) string {
	t.Helper()

	_, afterStart, found := strings.Cut(text, startHeading)
	if !found {
		t.Fatalf("heading %q not found", startHeading)
	}

	if endHeading == "" {
		return startHeading + afterStart
	}

	section, _, found := strings.Cut(afterStart, endHeading)
	if !found {
		t.Fatalf("heading %q not found after %q", endHeading, startHeading)
	}

	return startHeading + section
}
