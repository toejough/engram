//go:build targ && integration

package dev

import (
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a14i1a_AReversalFavorsTheNewerNote reads the recall skill the repository ships: a newer item that says
// "use X not Y" reverses the older claim, and when it conflicts the newer evidence wins.
func Test792a14i1a_AReversalFavorsTheNewerNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Evidence **conflicts** when a newer member explicitly negates or reverses " +
		"an older claim."))
	g.Expect(text).To(ContainSubstring("Reversal cues: \"no longer\", \"replaced by\", \"use X not Y\""))
	g.Expect(text).To(ContainSubstring("When conflict is present: **recent wins**."))
}

// Test792a14i1b_AStableConventionIsNotDemoted reads the recall skill the repository ships: with no conflict, older and
// newer evidence are independently valid and a stable convention is not demoted for lacking a recent instance.
func Test792a14i1b_AStableConventionIsNotDemoted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("When no conflict: treat older and newer evidence as independently valid"))
	g.Expect(text).To(ContainSubstring("do not demote a stable convention merely because it lacks a recent instance."))
}

// Test792a14i1c_ACandidateThatOnlyMatchesTheSupersededClaimIsNear reads the recall skill the repository ships: a
// candidate that only matches the superseded content is near, not covered.
func Test792a14i1c_ACandidateThatOnlyMatchesTheSupersededClaimIsNear(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Judge coverage against the recency-weighted view — in this order"))
	g.Expect(text).To(ContainSubstring("a candidate that only matches the superseded content is **near**, " +
		"not covered)"))
}

// Test792a14j1a_CoveredRefreshesProvenanceAndRecencyOnly reads the recall skill the repository ships: a covered
// candidate gets engram amend with --activate and the chunk sources, and its content is not rewritten.
func Test792a14j1a_CoveredRefreshesProvenanceAndRecencyOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("states the cluster's principle with **no material omission** vs the " +
		"recency-weighted members"))
	g.Expect(text).To(ContainSubstring("`engram amend --target <candidate-path> --activate " +
		"--chunk-source <new-chunk-ids>` — provenance-enrich only; **do not rewrite content**."))
}

// Test792a14j1b_NearReSynthesizes reads the recall skill the repository ships: a near candidate is amended with the
// chunk sources and content flags that sharpen its own claim from all members, and no different claim is added.
func Test792a14j1b_NearReSynthesizes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("A candidate addresses the same situation and omits ≥ 1 substantive part of " +
		"**its own claim** (same subject, same principle) that the members evidence"))
	g.Expect(text).To(ContainSubstring("`engram amend --target <candidate-path> --chunk-source <chunk-ids> " +
		"--subject ... --predicate ... --object ...` (or `--behavior/--impact/--action`) — sharpen the note's own claim " +
		"from all members, recency-weighted; never add a different claim."))
}

// Test792a14j1c_AbsentCreatesThroughTheWorker reads the recall skill the repository ships: an absent cluster whose
// principle is durable and atomic is handed to the write-memory skill as one fact or feedback note with the chunk
// sources and a source string.
func Test792a14j1c_AbsentCreatesThroughTheWorker(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("No candidate states the cluster's principle (a candidate on the same " +
		"situation with a different claim does not)"))
	g.Expect(text).To(ContainSubstring("Only when the principle is **durable**"))
	g.Expect(text).To(ContainSubstring("and **atomic** (one claim); otherwise write nothing."))
	g.Expect(text).To(ContainSubstring("Invoke the **write-memory** skill with this handoff — kind=fact or feedback " +
		"(pick per the cluster's principle), situation + content fields, `--source=\"<descriptive>\"`, " +
		"the cluster's chunk-source IDs"))
}

// Test792a14j1d_ACorrectionCarriesASupersession reads the recall skill the repository ships: a write that corrects,
// narrows or refutes a surfaced note carries --supersedes with the note, the type and the claim.
func Test792a14j1d_ACorrectionCarriesASupersession(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("also pass `--supersedes=\"<basename>\\|<type>\\|<claim>\"` " +
		"(types: `updates\\|narrows\\|refutes`)"))
	g.Expect(text).To(ContainSubstring("plus supersedes details if the new note corrects a surfaced note."))
}

// Test792a14j1e_OneNotePerCluster reads the recall skill the repository ships: an absent cluster gets exactly one
// representative note of one kind, never both a fact and a feedback.
func Test792a14j1e_OneNotePerCluster(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**One write per cluster; one representative note per cluster.**"))
	g.Expect(text).To(ContainSubstring("write exactly one note (fact *or* feedback) covering the cluster's principle."))
	g.Expect(text).To(ContainSubstring("Do not write one fact and one feedback note for the same cluster."))
}

// Test792a14j1f_HighCosineDoesNotDecide reads the recall skill the repository ships: coverage is judged from the
// candidate content, and a high cosine only nominates a candidate.
func Test792a14j1f_HighCosineDoesNotDecide(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Do not judge coverage before you have read the candidate content."))
	g.Expect(text).To(ContainSubstring("| A `≥0.95` cluster → you activated without reading the candidates | " +
		"Read first; high cosine nominates, it does not decide |"))
}

// Test792a14j1g_WritesAreSequential reads the recall skill the repository ships: each write finishes before the next
// cluster is judged.
func Test792a14j1g_WritesAreSequential(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**WAIT for each write before moving to the next cluster.** " +
		"Writes are blocking and inline"))
	g.Expect(text).To(ContainSubstring("the note created or updated by one cluster may be a candidate for another."))
}

// Test792a14j1h_UpdatesNeverUseLearn reads the recall skill the repository ships: an existing note is updated with
// engram amend, and engram learn is create-only.
func Test792a14j1h_UpdatesNeverUseLearn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("| You called `engram learn --target` to update a note in place | " +
		"Updates use `engram amend`; `engram learn` is create-only |"))
}

// Test792a14k1a_AParentNoteCoversACluster reads the recall skill the repository ships: a from_parent note that covers
// a cluster is activated, not amended, and no new note restates it.
func Test792a14k1a_AParentNoteCoversACluster(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**A `from_parent` item is never an amend target, and neither is its " +
		"pending pulled copy.**"))
	g.Expect(text).To(ContainSubstring("When one covers the cluster, the action is Step 2.7's `engram activate` " +
		"(which pulls it down) — no `amend --activate`, no `--chunk-source`, and no new note restating it."))
}

// Test792a14k1b_APendingCopyIsNotAmended reads the recall skill the repository ships: even when the user asks to
// enrich the note, the pulled copy is an amend target only after curation accepts it, so the agent curates first.
func Test792a14k1b_APendingCopyIsNotAmended(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Even when the user asks you to enrich or record against the note that " +
		"states the rule, the copy becomes an amend target only after curation has accepted it " +
		"(it is then a live local note): curate first, then amend."))
}

// Test792a14l1a_UsedNotesAreActivated reads the recall skill the repository ships: one engram activate call carries
// a --note for each note judged covered or near and each note cited in the synthesis, and no other.
func Test792a14l1a_UsedNotesAreActivated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("After processing all clusters, call `engram activate` on the notes you " +
		"actually drew on — the `candidate_l2s` you judged Covered or Near at the coverage table, and any notes " +
		"you cited in the Step 3 synthesis"))
	g.Expect(text).To(ContainSubstring("# ... one --note per used note only"))
}

// Test792a14l1b_UnusedReturnsAreNotActivated reads the recall skill the repository ships: returned notes the agent did
// not use are not activated.
func Test792a14l1b_UnusedReturnsAreNotActivated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Do NOT activate every returned note."))
	g.Expect(text).To(ContainSubstring("Activating only what you used lets superseded-but-surfaced notes fade via " +
		"recency rank"))
}

// Test792a14l1c_ChunksAndRecentItemsAreNeverActivated reads the recall skill the repository ships: no chunk and no
// recent-channel item is activated.
func Test792a14l1c_ChunksAndRecentItemsAreNeverActivated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Do NOT activate recent-channel items (chunks are never activated)."))
}

// Test792a14l1d_AUsedParentNoteIsPulledDownAndLeftPending reads the recall skill the repository ships: a used
// from_parent note is activated by its path, the pulled copy is left pending and not amended, and the reply says the
// copy awaits curation or that curate ran as a separate step.
func Test792a14l1d_AUsedParentNoteIsPulledDownAndLeftPending(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Activate the `from_parent` notes you used the same way, by their `path`: " +
		"this pulls them down as local pending copies."))
	g.Expect(text).To(ContainSubstring("leave the copies pending, finish the user's request, then run the curate " +
		"skill as a separate step and say in your reply that you did (or that they await curation). " +
		"Never amend the pulled copy."))
}

// Test792a14l1e_NoNotesDrawnNoActivation reads the recall skill the repository ships: the activation call is skipped
// when no notes were drawn on, as when the payload was empty or the coverage step was skipped.
func Test792a14l1e_NoNotesDrawnNoActivation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Skip this call when you drew on no notes (e.g., payload was empty or " +
		"Step 2.5 was skipped)."))
}

// Test792a14m1a_TheCountOpensTheSynthesis reads the recall skill the repository ships: the synthesis opens with one
// sentence counting the items, chunks and notes surfaced and the lessons crystallized.
func Test792a14m1a_TheCountOpensTheSynthesis(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**Open with the count.** One sentence: \"Query surfaced N items " +
		"(K chunks, M notes); crystallized J lessons.\""))
}

// Test792a14m1b_ThePlanIsWalkedInOrder reads the recall skill the repository ships: the synthesis walks the Step 0
// plan in order, marking each numbered action confirmed, adjusted, contradicted or silent.
func Test792a14m1b_ThePlanIsWalkedInOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**Walk the plan from Step 0 in order.** Per numbered action: **confirmed**, " +
		"**adjusted** (and how), **contradicted**, or **silent**."))
}

// Test792a14m1c_ConventionsAreFramedAsRequirements reads the recall skill the repository ships: load-bearing
// conventions from lessons and chunk evidence alike are listed under "Apply these as requirements:".
func Test792a14m1c_ConventionsAreFramedAsRequirements(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**Frame load-bearing conventions as requirements.** " +
		"Lead with \"Apply these as requirements:\" and list them — drawn from lessons and chunk evidence alike."))
}

// Test792a14m1d_SilenceIsBrief reads the recall skill the repository ships: when memory was silent on everything the
// synthesis is one sentence.
func Test792a14m1d_SilenceIsBrief(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("if memory was silent on everything, one sentence."))
}

// Test792a14m1e_NoPayloadDump reads the recall skill the repository ships: the synthesis never re-emits the payload,
// pastes whole chunks or lists raw scores or paths.
func Test792a14m1e_NoPayloadDump(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart2SkillPath)

	g.Expect(text).To(ContainSubstring("**No payload dumps** — never re-emit YAML, paste whole chunks, or list raw " +
		"scores/paths."))
}

// unexported constants.
const (
	recallPart2SkillPath = "agent-instructions/skills/recall/SKILL.md"
)
