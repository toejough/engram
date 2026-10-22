//go:build targ && integration

package dev

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a14a1a_TheThreeBlocksComeFirst reads the recall skill the repository ships: Step 0 prints the Ask, the
// Situation and the Plan before any engram call, and skipping it is forbidden.
func Test792a14a1a_TheThreeBlocksComeFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("**Before any engram call**, print three short blocks."))
	g.Expect(text).To(ContainSubstring("- **Ask:** what is being asked, in your own words. One or two sentences."))
	g.Expect(text).To(ContainSubstring("- **Situation:** the concrete context — repo, branch, the operation underway, " +
		"what's loaded into your context."))
	g.Expect(text).To(ContainSubstring("- **Plan:** the action(s) you would take next absent any guidance from memory."))
	g.Expect(text).To(ContainSubstring("Skipping Step 0 is forbidden."))
	g.Expect(recallPart1Before(text, "### Step 0 — Print your upfront judgement", "### Step 0.5 — Sweep")).To(BeTrue())
	g.Expect(recallPart1Before(text, "### Step 0 — Print your upfront judgement", "### Step 2 — Run ONE unified")).
		To(BeTrue())
}

// Test792a14a1b_ThePlanIsTheUnguidedPlan reads the recall skill the repository ships: the Plan lists the actions the
// agent would take absent any guidance from memory, as a numbered list.
func Test792a14a1b_ThePlanIsTheUnguidedPlan(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("- **Plan:** the action(s) you would take next absent any guidance from memory. " +
		"Numbered list."))
	g.Expect(text).To(ContainSubstring("The whole purpose of recall is to test a stated plan against memory."))
}

// Test792a14b1a_TheFirstRecallOfASessionSweeps reads the recall skill the repository ships: the sweep, engram ingest
// --auto, is a step that comes before the query.
func Test792a14b1a_TheFirstRecallOfASessionSweeps(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("### Step 0.5 — Sweep ```bash engram ingest --auto ```"))
	g.Expect(text).To(ContainSubstring("guarantees the index includes the latest sessions and doc edits."))
	g.Expect(text).To(ContainSubstring("**Sweep, then run ONE unified `engram query`.**"))
	g.Expect(recallPart1Before(text, "### Step 0.5 — Sweep", "### Step 2 — Run ONE unified")).To(BeTrue())
}

// Test792a14b1b_ALaterRecallDoesNotSweepAgain reads the recall skill the repository ships: a sweep that already ran in
// the session is not repeated, even on a suspicion of a change from another terminal.
func Test792a14b1b_ALaterRecallDoesNotSweepAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("If a sweep (`engram ingest --auto`) already ran earlier in THIS session, skip it"))
	g.Expect(text).To(ContainSubstring("This holds even under a \"something might have changed outside this session " +
		"(another terminal, another agent)\" worry"))
	g.Expect(text).To(ContainSubstring("not by re-sweeping now on suspicion."))
}

// Test792a14b1c_AFailedSweepDoesNotStopRecall reads the recall skill the repository ships: a sweep that errors is
// reported and the query still runs.
func Test792a14b1c_AFailedSweepDoesNotStopRecall(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("If it errors, say so and continue — retrieval over a slightly-stale index " +
		"beats no retrieval."))
}

// Test792a14c1a_NoModeMeansDeep reads the recall skill the repository ships: an absent mode word means deep, the whole
// procedure with ten phrases and the write side.
func Test792a14c1a_NoModeMeansDeep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("(the mode word is the skill argument; absent → `deep`)"))
	g.Expect(text).To(ContainSubstring("- **`deep` (default).** The full procedure below — all 10 phrases and the " +
		"write side (Steps 2.5C, Step 4)."))
	g.Expect(text).To(ContainSubstring("It both *applies* memory to this decision **and** *grows the vault*"))
}

// Test792a14c1b_GlanceAsksForAboutThreePhrases reads the recall skill the repository ships: glance generates about
// three phrases, not ten.
func Test792a14c1b_GlanceAsksForAboutThreePhrases(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("Steps 0–3.5 with **~3 phrases** (not 10)"))
	g.Expect(text).To(ContainSubstring("> **[glance: generate ~3 phrases, not 10"))
}

// Test792a14c1c_GlanceWritesNoVaultKnowledge reads the recall skill the repository ships: glance is read-only with
// respect to vault knowledge and skips the coverage writes and the persisting of a conclusion.
func Test792a14c1c_GlanceWritesNoVaultKnowledge(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("A pass that is **read-only with respect to vault knowledge**"))
	g.Expect(text).To(ContainSubstring("but **skip the write side**: Step 2.5C (coverage amend/learn), Step 4 " +
		"(synthesis-persist)."))
	g.Expect(text).To(ContainSubstring("> **[glance: SKIP Step 2.5C — it is the write side."))
	g.Expect(text).To(ContainSubstring("do not amend/learn — continue to Step 2.7 (activate).]**"))
	g.Expect(text).To(ContainSubstring("> **[glance: SKIP Step 4 — write side."))
	g.Expect(text).To(ContainSubstring("Glance *applies* memory to this decision; it does **not** grow the vault's " +
		"knowledge."))
}

// Test792a14c1d_GlanceStillReadsWeightsActivatesAndSynthesizes reads the recall skill the repository ships: glance
// keeps the reading of candidates, the recency weight, the activation, the synthesis and the re-entry query.
func Test792a14c1d_GlanceStillReadsWeightsActivatesAndSynthesizes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("**keep the read side** — Step 2.5A (read candidates), **Step 2.5B " +
		"(apply the recency weight)**, Step 2.7 (activate used notes), the Step 3 synthesis, and Step 3.5 (the " +
		"re-entry query, when triggered)"))
	g.Expect(text).To(ContainSubstring("Read 2.5A + apply 2.5B, do not amend/learn"))
	g.Expect(text).To(ContainSubstring("Glance does this too."))
	g.Expect(text).To(ContainSubstring("Runs in both modes (a query is a read; glance keeps it)."))
}

// Test792a14d1a_ARecentConventionThatTheDecisionTurnsOn reads the recall skill the repository ships: glance switches
// to deep when the decision turns on a standard that surfaced in the recent-activity channel, before the synthesis.
func Test792a14d1a_ARecentConventionThatTheDecisionTurnsOn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("So if your decision turns on **honoring a standard that surfaced in the " +
		"recent-activity channel** (a \"use X going forward\" / \"the new convention is Y\" item in Channel 2), " +
		"**switch to `deep`**."))
	g.Expect(text).To(ContainSubstring("> **[glance: before synthesizing, check for a load-bearing Channel 2 " +
		"(recent-activity) standard. If your decision turns on honoring a recent convention, escalate to `deep` now"))
	g.Expect(recallPart1Before(text, "**switch to `deep`**", "### Step 3 — Closing synthesis")).To(BeTrue())
}

// Test792a14d1b_OrdinaryConventionsDoNotEscalate reads the recall skill the repository ships: glance is validated as
// deep-equivalent for applying conventions, recency supersession within the matched set, and abduction and synthesis,
// so the one escalation trigger is the recent-activity standard.
func Test792a14d1b_OrdinaryConventionsDoNotEscalate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("Glance is validated as deep-equivalent only for applying conventions (C3), " +
		"recency *supersession within the matched set* (C4i, via 2.5B), and abduction/synthesis (C6)."))
	g.Expect(strings.Count(text, "**switch to `deep`**")).To(Equal(1))
}

// Test792a14e1a_TenPhrasesOnePerAngle reads the recall skill the repository ships: deep generates exactly ten
// phrases, one from each of the ten angles, in order.
func Test792a14e1a_TenPhrasesOnePerAngle(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("Always generate exactly **10** short queryable phrases, one from each of " +
		"these angles:"))

	angles := []string{
		"1. **Situation/setting**",
		"2. **User's intent/goal**",
		"3. **Current concrete action**",
		"4. **Problem/blocker**",
		"5. **Candidate solution/approach**",
		"6. **Tooling/tech in play**",
		"7. **Prior related work**",
		"8. **Adjacent technique**",
		"9. **Failure mode to avoid**",
		"10. **Domain/concept**",
	}

	previous := ""

	for _, angle := range angles {
		g.Expect(text).To(ContainSubstring(angle))

		if previous != "" {
			g.Expect(recallPart1Before(text, previous, angle)).To(BeTrue())
		}

		previous = angle
	}
}

// Test792a14e1b_PhrasesNameTheTaskNotTheFear reads the recall skill the repository ships: phrases are queried by task,
// not by fear.
func Test792a14e1b_PhrasesNameTheTaskNotTheFear(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("Each phrase is short and specific."))
	g.Expect(text).To(ContainSubstring("**Query by task, not by fear** — \"implementing Claude Code hooks\", not " +
		"\"common mistakes when writing hooks\"."))
}

// Test792a14f1a_AllPhrasesInOneCall reads the recall skill the repository ships: one engram query carries one
// --phrase per phrase, with no per-phrase calls.
func Test792a14f1a_AllPhrasesInOneCall(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("### Step 2 — Run ONE unified `engram query` with all phrases"))
	g.Expect(text).To(ContainSubstring("engram query --lazy-chunks \\ --phrase=\"<phrase 1>\" \\ " +
		"--phrase=\"<phrase 2>\" # ... one --phrase per Step 1 phrase (deep: 10; glance: ~3)"))
	g.Expect(text).To(ContainSubstring("One call;"))
	g.Expect(text).To(ContainSubstring("Do NOT collapse phrases, do NOT run per-phrase calls."))
	g.Expect(text).To(ContainSubstring("| Separate query calls per phrase | One call, repeatable `--phrase` flags |"))
}

// Test792a14f1b_NoVaultFlags reads the recall skill the repository ships: the query passes neither --vault nor
// --chunks-dir, because the binary resolves them.
func Test792a14f1b_NoVaultFlags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("The binary resolves the vault and chunk index automatically"))
	g.Expect(text).To(ContainSubstring("**Do not pass `--vault` or `--chunks-dir`.**"))
	g.Expect(text).To(ContainSubstring("| `--vault` or `--chunks-dir` on the query |"))
}

// Test792a14f1c_ValuesAreAttachedToTheirFlags reads the recall skill the repository ships: a free-text flag value is
// written --flag="value", the quote touching the equals sign, because a value beginning with a dash is otherwise
// misread as a flag.
func Test792a14f1c_ValuesAreAttachedToTheirFlags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("must be written as `--flag=\"value\"` — the quote touching the `=`, no space — " +
		"because a value beginning with `-` is otherwise misread as a new flag and the command fails."))
	g.Expect(text).To(ContainSubstring("--phrase=\"<phrase 1>\""))
}

// Test792a14g1a_ChunksAreEvidenceNotQuotations reads the recall skill the repository ships: a chunk is evidence from
// which the agent extracts the principle, and is never quoted wholesale.
func Test792a14g1a_ChunksAreEvidenceNotQuotations(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("These are EVIDENCE: extract the convention, decision, or correction they show " +
		"(a reviewer correcting code, a stated standard); never quote them wholesale."))
	g.Expect(text).To(ContainSubstring("| You quoted chunks wholesale into the reply | Extract the principle a chunk " +
		"evidences; paraphrase |"))
}

// Test792a14g1b_NotesAreAppliedAsTheyStand reads the recall skill the repository ships: a note carries its content
// inline and is applied directly, with no engram show.
func Test792a14g1b_NotesAreAppliedAsTheyStand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("crystallized lessons; apply directly (notes always carry full content inline)."))
	g.Expect(text).To(ContainSubstring("**no `engram show` calls for candidates**"))
	g.Expect(text).To(ContainSubstring("| You called `engram show` on a note already in `items[]` | NOTE members in " +
		"`items[]` carry `content` — use it directly."))
}

// Test792a14g1c_RecentItemsAreReadFirstAndNotJudged reads the recall skill the repository ships: the recent-activity
// block is read first to re-immerse in recent work and is not used for coverage or synthesis judgment.
func Test792a14g1c_RecentItemsAreReadFirstAndNotJudged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("Read this block first for situational continuity — re-immerse in recent work " +
		"before diving into the clustered results. These items are NOT used for coverage or synthesis judgment."))
	g.Expect(text).To(ContainSubstring("do not re-derive them, do not express surprise at them"))
}

// Test792a14g1d_AnEmptyMatchedSetIsReported reads the recall skill the repository ships: an empty matched channel is
// reported in one sentence, the coverage step is skipped and the plan goes on, and a recent block alone does not count
// as something surfacing.
func Test792a14g1d_AnEmptyMatchedSetIsReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("If the matched items (Channel 1) are empty, say so in one sentence, skip Step " +
		"2.5, and proceed with your plan."))
	g.Expect(text).To(ContainSubstring("(A non-empty recent-activity block alone does not count as \"something " +
		"surfaced\" for coverage purposes.)"))
}

// Test792a14h1a_AZeroNoteClusterStillGetsItsChunksRead reads the recall skill the repository ships: a cluster with an
// empty candidate list still has every chunk member read with engram show-chunk before any judgment.
func Test792a14h1a_AZeroNoteClusterStillGetsItsChunksRead(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("**For zero-note clusters (empty `candidate_l2s` list):** The absence of note " +
		"candidates does NOT skip the chunk-reading requirement. You MUST still call `engram show-chunk` on every " +
		"chunk member before stating any relevance or coverage judgment about the cluster."))
}

// Test792a14h1b_ATitleIsNotEvidence reads the recall skill the repository ships: a judgment from a chunk's path or
// title alone is a violation, because under lazy chunks the content is not in the payload.
func Test792a14h1b_ATitleIsNotEvidence(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("A metadata-only judgment (path/title alone, no fetched content) is a violation " +
		"even when `candidate_l2s` is empty."))
	g.Expect(text).To(ContainSubstring("Under `--lazy-chunks` chunks carry path/source only — `engram show-chunk " +
		"<source#anchor>` on-demand before judging coverage"))
}

// Test792a14h1c_ALoadBearingChunkBesideADistractor reads the recall skill the repository ships: every chunk member of
// a cluster is read before any relevance judgment, so a topically distant chunk is not skipped by its path.
func Test792a14h1c_ALoadBearingChunkBesideADistractor(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("You MUST still call `engram show-chunk` on every chunk member before stating " +
		"any relevance or coverage judgment about the cluster."))
	g.Expect(text).To(ContainSubstring("the content is NOT in the payload (chunks carry path/source only under " +
		"`--lazy-chunks`; the cluster's `members` list never carries content)"))
}

// Test792a14h1d_CandidateNotesAreReadFromThePayload reads the recall skill the repository ships: a candidate note's
// content is read inline from the payload, with no engram show, and before coverage is judged.
func Test792a14h1d_CandidateNotesAreReadFromThePayload(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, recallPart1SkillPath)

	g.Expect(text).To(ContainSubstring("`candidate_l2s` entries carry their `content` inline — read it directly; " +
		"**no `engram show` calls for candidates**"))
	g.Expect(text).To(ContainSubstring("Do not judge coverage before you have read the candidate content."))
}

// unexported constants.
const (
	recallPart1SkillPath = "agent-instructions/skills/recall/SKILL.md"
)

// recallPart1Before reports whether both fragments occur in the text and the first occurs before the second.
func recallPart1Before(text, first, second string) bool {
	firstAt := strings.Index(text, first)
	secondAt := strings.Index(text, second)

	return firstAt >= 0 && secondAt >= 0 && firstAt < secondAt
}
