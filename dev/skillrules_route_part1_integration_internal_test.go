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

// Test792a17a1a_ObjectLevelWorkIsDispatched reads the route skill the repository ships: object-level work, writing code
// and running tests, is delegated to a subagent and the orchestrator does not do it itself, and no shipped instruction
// has the orchestrator do object-level work itself.
func Test792a17a1a_ObjectLevelWorkIsDispatched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)
	division := learnPart1Between(t, text, routePart1OrchestrationHeading, routePart1TierHeading)

	g.Expect(text).To(ContainSubstring("You are an orchestrator. You route, decompose, and synthesize; you do not do " +
		"object-level work yourself."))
	g.Expect(division).To(ContainSubstring("**You delegate (object-level):**"))
	g.Expect(division).To(ContainSubstring("writing code or prose"))
	g.Expect(division).To(ContainSubstring("running tests/builds"))
	g.Expect(division).To(ContainSubstring("anything that produces or evaluates the deliverable."))
	g.Expect(routePart1Contradictions(t, routePart1ObjectLevelScan)).To(BeEmpty())
}

// Test792a17a1b_OrchestrationStaysWithTheOrchestrator reads the route skill the repository ships: the orchestrator
// itself decides the next step and synthesizes the returned results, and the act of delegating is not delegated.
func Test792a17a1b_OrchestrationStaysWithTheOrchestrator(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)
	division := learnPart1Between(t, text, routePart1OrchestrationHeading, routePart1TierHeading)

	g.Expect(division).To(ContainSubstring("The line that keeps \"delegate everything\" from collapsing into either " +
		"\"delegate nothing\" or \"delegate the act of delegating\":"))
	g.Expect(division).To(ContainSubstring("- **You do (orchestration):** routing/decomposition decisions, dispatching " +
		"subagents"))
	g.Expect(division).To(ContainSubstring("and synthesizing subagents' returned results into the next decision or the " +
		"user-facing report."))
	g.Expect(routePart1Contradictions(t, routePart1DelegatingScan)).To(BeEmpty())
}

// Test792a17a1c_EasyWorkIsDelegatedCheaply reads the route skill the repository ships: there is no inline escape for
// easy work, which is delegated to a cheap model and not skipped, and every unit starts at the cheapest tier.
func Test792a17a1c_EasyWorkIsDelegatedCheaply(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("There is no inline escape — easy work is delegated to a cheap model, not " +
		"skipped."))
	g.Expect(text).To(ContainSubstring("Every unit starts at the cheapest / fastest available tier."))
	g.Expect(routePart1Contradictions(t, routePart1EasyInlineScan)).To(BeEmpty())
}

// Test792a17b1a_NoEvidenceCheapestTier reads the route skill the repository ships: a unit with no recalled evidence
// starts at the cheapest tier, and the dispatch record names "cold-start default" as the reason for the choice.
func Test792a17b1a_NoEvidenceCheapestTier(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("2. **Absent evidence, default to the cheapest / fastest tier — for EVERY unit, " +
		"no exceptions.**"))
	g.Expect(text).To(ContainSubstring("Cold-start is cheapest-for-everything"))
	why := learnPart1Between(t, text, "| **why** |", "| **outcome** |")

	g.Expect(why).To(ContainSubstring("\"cold-start default\""))
}

// Test792a17b1b_HardLookingWorkStartsCheap reads the route skill the repository ships: work that feels hard, a
// cross-cutting refactor, a new API design and a race condition, starts at the cheap tier on a cold start, and the red
// flag names choosing mid or deep because a unit looks hard.
func Test792a17b1b_HardLookingWorkStartsCheap(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)
	feels := learnPart1Between(t, text, "This includes work that *feels* genuinely hard:", "Your sense that")
	coldStart := learnPart1Between(t, text, "On a cold start with no recalled evidence,", "same as a variable rename.")

	g.Expect(feels).To(ContainSubstring("cross-cutting refactors"))
	g.Expect(coldStart).To(ContainSubstring("a race condition"))
	g.Expect(coldStart).To(ContainSubstring("an 8-file refactor"))
	g.Expect(coldStart).To(ContainSubstring("a new API design"))
	g.Expect(coldStart).To(ContainSubstring("all start **cheap**"))
	g.Expect(text).To(ContainSubstring("| You picked mid/deep because the unit \"looks hard\" | Default cheapest; only " +
		"recalled evidence raises the start. A surface hunch is the guess the loop replaces. |"))
}

// Test792a17b1c_RecalledFailureRaisesTheStart reads the route skill the repository ships: only recalled evidence that
// the cheaper tier failed raises the starting tier, the evidence sets the starting tier, and its example names the
// tier the failure needs; no shipped instruction starts above the tier the evidence supports.
func Test792a17b1c_RecalledFailureRaisesTheStart(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("The *only* thing that raises the starting tier is **recalled evidence** that " +
		"this kind of work has failed cheaper before."))
	g.Expect(text).To(ContainSubstring("(\"cheap tier sufficed for X\" / \"cheap failed on Y — needs mid\")"))
	g.Expect(text).To(ContainSubstring("Recalled evidence sets the starting tier."))
	g.Expect(routePart1Contradictions(t, routePart1AboveEvidenceScan)).To(BeEmpty())
}

// Test792a17b1d_AHunchIsNotEvidence reads the route skill the repository ships: a sense that a unit needs a strong
// model is a prediction of difficulty and not evidence, the red flag says to start cheap, and no shipped instruction
// counts predicted difficulty as evidence.
func Test792a17b1d_AHunchIsNotEvidence(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("Your sense that a unit \"needs a strong model\" is a **prediction of " +
		"difficulty, which is not evidence**"))
	g.Expect(text).To(ContainSubstring("\"genuinely complex\" and \"looks hard\" are the same hunch in different " +
		"words, and the hunch is exactly what this loop replaces."))
	g.Expect(text).To(ContainSubstring("| You reasoned \"correctness-critical / broad impact / needs careful reasoning " +
		"→ deep\" | That is forecasting difficulty. Start cheap; let a *failed review* prove it needs more — don't " +
		"pre-provision. |"))
	g.Expect(routePart1Contradictions(t, routePart1HunchScan)).To(BeEmpty())
}

// Test792a17b1e_EvidenceIsReadByPlainRecall reads the route skill the repository ships: the route-evidence aggregates
// surface through plain recall with no special query and no counting on the read path, the tallies sit in the
// aggregate's text, and engram count is never used to route.
func Test792a17b1e_EvidenceIsReadByPlainRecall(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("Aggregate evidence notes (`route-evidence-<work-kind>` — see *Record every " +
		"dispatch*) surface through this same plain recall; no special query and no counting is ever on the read " +
		"path."))
	g.Expect(text).To(ContainSubstring("<tier tallies, e.g. cheap 14/16, mid 2/2> as of <date>"))
	g.Expect(text).To(ContainSubstring("use it to verify/repair an aggregate, never to route (routing reads are plain " +
		"recall)."))
	g.Expect(routePart1Contradictions(t, routePart1CountToRouteScan)).To(BeEmpty())
}

// Test792a17c1a_TheFirstFailureRewritesTheSpec reads the route skill the repository ships: a first failure is a spec
// failure, so the handoff is rewritten and the same tier is retried, and the red flag says not to escalate on it.
func Test792a17c1a_TheFirstFailureRewritesTheSpec(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("**First fail** → the failure is usually a *spec* failure, not a model failure."))
	g.Expect(text).To(ContainSubstring("Rewrite the handoff (sharper files, acceptance checks, tighter do-NOT-touch), " +
		"retry the **same** tier."))
	g.Expect(text).To(ContainSubstring("The builder never gets to guess twice off the same spec."))
	g.Expect(text).To(ContainSubstring("| You escalated a tier on the first failure | First fail → rewrite the spec, " +
		"retry same tier. Escalate only on the second fail. |"))
}

// Test792a17c1b_TheSecondFailureEscalates reads the route skill the repository ships: a second failure of the same
// unit escalates one tier and retries; the red flag row against escalating on the first failure is the clause of
// 792a17c1a.
func Test792a17c1b_TheSecondFailureEscalates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("**Second fail on the same unit** → escalate **one** tier and retry."))
}

// Test792a17c1c_EscalationStopsAtDeep reads the route skill the repository ships: the repeated escalation ends when
// the unit passes or reaches the deep tier, and no shipped instruction escalates beyond the deep tier.
func Test792a17c1c_EscalationStopsAtDeep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("Repeat until it passes or you reach the deep tier."))
	g.Expect(routePart1Contradictions(t, routePart1BeyondDeepScan)).To(BeEmpty())
}

// Test792a17d1a_TheDiscountApplies reads the route skill the repository ships: a unit whose needed knowledge is
// recallable drops one tier, a convention is such knowledge, and the dispatch record names "memory-discount applied"
// as the reason.
func Test792a17d1a_TheDiscountApplies(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("4. **Memory discounts the tier (a special case of evidence lowering it).** A " +
		"unit whose needed knowledge is recallable — a known convention, prior decision, crystallized diagnostic — " +
		"drops one tier"))
	g.Expect(text).To(ContainSubstring("because the model *applies* recalled knowledge instead of *deriving* it."))
	g.Expect(text).To(ContainSubstring("\"memory-discount applied\""))
}

// Test792a17d1b_TheDiscountStopsAtCheap reads the route skill the repository ships: the one-tier drop of a
// memory-backed unit is floored at the cheap tier, in the rule and in the cold-start table.
func Test792a17d1b_TheDiscountStopsAtCheap(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("(floored at cheap)"))
	g.Expect(text).To(ContainSubstring("| Memory-backed unit (answer is recallable) | one tier down, floored at cheap |"))
}

// Test792a17e1a_AFreeLocalModelIsPreferred reads the route skill the repository ships: a free local model counts as
// cheaper than any paid model and is tried first, paid models enter the roster only above the free-local options, and
// no shipped instruction tries a paid model ahead of an available free one.
func Test792a17e1a_AFreeLocalModelIsPreferred(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("A free local model (e.g. an LMStudio provider) counts as cheaper than any paid " +
		"API model and is tried first when one is available;"))
	g.Expect(text).To(ContainSubstring("paid models enter the roster only above free-local options."))
	g.Expect(routePart1Contradictions(t, routePart1PaidFirstScan)).To(BeEmpty())
}

// Test792a17e1b_NoFreeLocalModelTheCheapestPaidOne reads the route skill the repository ships: the cheap tier is the
// cheapest agent available in the current environment, and without a free-local option it resolves to the cheapest
// paid model.
func Test792a17e1b_NoFreeLocalModelTheCheapestPaidOne(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("the cheap tier means the cheapest agent genuinely available right now."))
	g.Expect(text).To(ContainSubstring("Absent a free-local option, cheap resolves to the cheapest paid model, as " +
		"before."))
}

// Test792a17e1c_APaidDispatchWithAFreeOptionAvailableIsARedFlag reads the route skill the repository ships: the red
// flags table lists a cheap dispatch to a paid model while a free local one was available, and says to resolve the
// roster against the environment first.
func Test792a17e1c_APaidDispatchWithAFreeOptionAvailableIsARedFlag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)
	_, redFlags, found := strings.Cut(text, routePart1RedFlagsHeading)

	g.Expect(found).To(BeTrue())
	g.Expect(redFlags).To(ContainSubstring("| You dispatched cheap work to a paid API model while a free local model " +
		"was available | Resolve the roster against the environment first — free-local beats any paid model at the cheap " +
		"tier. |"))
}

// Test792a17f1a_AWorkKindClearsTheThresholds reads the route skill the repository ships: the cold-start priors table
// has a row for work-kind/go-package-implementation starting at mid that cites the tally of 8 of 12 cheap passes and 10
// of 10 mid passes.
func Test792a17f1a_AWorkKindClearsTheThresholds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)
	row := learnPart1Between(t, text, "| work-kind/go-package-implementation |", "2026-08-09) |")

	g.Expect(row).To(ContainSubstring("| work-kind/go-package-implementation | mid |"))
	g.Expect(row).To(ContainSubstring("**evidence-backed**"))
	g.Expect(row).To(ContainSubstring("8/12 cheap vs 10/10 mid"))
}

// Test792a17f1b_AHighCheapPassRateStaysAtTheDefault reads the route skill the repository ships: the cold-start rule
// names the thresholds that start a work-kind above cheap (a cheap pass rate below 70% and a mid pass rate above 85%,
// both past the five-dispatch evidence floor), so a work-kind that passes at the cheap tier keeps the default, and no
// clause says that none of the entries has earned a start above cheap, which the evidence-backed mid row contradicts.
func Test792a17f1b_AHighCheapPassRateStaysAtTheDefault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)
	rule := learnPart1Between(t, text, routePart1ColdStartHeading, "| Work character |")

	g.Expect(rule).To(ContainSubstring("An entry starts above cheap only when its recorded cheap-tier pass rate is " +
		"below 70% and its mid-tier pass rate is above 85%, both past the five-dispatch evidence floor"))
	g.Expect(text).NotTo(ContainSubstring("none has earned it yet"))
}

// Test792a17f1c_ARecalledOutcomeOverridesAPrior reads the route skill the repository ships: the cold-start priors are
// hypotheses that recalled evidence overwrites, and the red flag for treating the table as a prescription says
// recalled evidence wins.
func Test792a17f1c_ARecalledOutcomeOverridesAPrior(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("## Cold-start priors (unproven — evidence overwrites these)"))
	g.Expect(text).To(ContainSubstring("these are the *starting* tiers — hypotheses the dispatch record confirms or " +
		"overturns, not prescriptions."))
	g.Expect(text).To(ContainSubstring("| You treated the cold-start table as a prescription | It's a set of " +
		"overwritable priors; recalled evidence wins. |"))
}

// Test792a17g1a_ACompleteHandoff reads the route skill the repository ships: every dispatch must hand the subagent the
// exact files and paths, the acceptance checks, the do-not-touch bounds, the recall-first instruction and the
// completion-report contract that ends in a LESSONS line.
func Test792a17g1a_ACompleteHandoff(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)
	handoff := learnPart1Between(t, text, "Every dispatch MUST hand the subagent:", "Vague handoffs are why")

	g.Expect(handoff).To(ContainSubstring("- **Exact files and paths** to create/modify (not \"the auth code\" — " +
		"`internal/foo/bar.go:20-45`)."))
	g.Expect(handoff).To(ContainSubstring("- **Acceptance checks** — the concrete, verifiable conditions that mean " +
		"\"done\""))
	g.Expect(handoff).To(ContainSubstring("- **Explicit do-NOT-touch bounds** — files, interfaces, and behaviors the " +
		"unit must leave alone."))
	g.Expect(handoff).To(ContainSubstring("- **The subagent's recall-first instruction**"))
	g.Expect(handoff).To(ContainSubstring("- **The completion-report contract:** the subagent's final report MUST end " +
		"with a `LESSONS:` line"))
}

// Test792a17g1b_AVagueHandoffIsFixed reads the route skill the repository ships: vague handoffs are why cheap tiers
// fail, the spec is suspected before the model, and the red flag for a dispatch without exact files, acceptance checks
// and do-not-touch bounds says to fix the handoff.
func Test792a17g1b_AVagueHandoffIsFixed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("A cheap-tier model matches an expensive one when the spec is exact; when it " +
		"fails, suspect the spec before the model."))
	g.Expect(text).To(ContainSubstring("Vague handoffs are why cheap tiers \"fail.\" Fix the handoff first."))
	g.Expect(text).To(ContainSubstring("| You dispatched without exact files, acceptance checks, and do-NOT-touch " +
		"bounds | Fix the handoff — a vague spec is why cheap tiers \"fail.\" |"))
}

// Test792a17h1a_RecallFirstIsInEveryHandoff reads the route skill the repository ships: every dispatched subagent is
// instructed that its first action is recall, and the red flag for skipping the recall of a subagent that already has
// a prompt says it is non-waivable; the handoff bullet that lists the instruction is the clause of 792a17g1a.
func Test792a17h1a_RecallFirstIsInEveryHandoff(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("1. **The subagent recalls first.** Instruct every dispatched subagent that its " +
		"FIRST action is `/recall`, with phrases drawn from its unit, before doing the work."))
	g.Expect(text).To(ContainSubstring("| \"The subagent has the prompt, skip its recall\" | Recall-first is " +
		"non-waivable. |"))
}

// Test792a17h1b_ALargeUnitIsDecomposed reads the route skill the repository ships: a unit that spans multiple files or
// concerns or has more than one deliverable is broken into smaller units that are each routed, the decomposition is
// the orchestrator's, and no shipped instruction dispatches such a unit whole.
func Test792a17h1b_ALargeUnitIsDecomposed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart1SkillPath)

	g.Expect(text).To(ContainSubstring("2. **Decompose before dispatch.** A unit too large for one focused subagent — " +
		"it spans multiple files or concerns, or needs more than one clear deliverable — is not dispatched as-is."))
	g.Expect(text).To(ContainSubstring("Break it into smaller units and route each."))
	g.Expect(text).To(ContainSubstring("Decomposition is orchestration; you do it yourself."))
	g.Expect(routePart1Contradictions(t, routePart1WholeUnitScan)).To(BeEmpty())
}

// TestRoutePart1Hits supports the absence scans of this file: a match is an instruction unless a negation governs it
// in its own clause, a clause ends at a sentence end, a colon, a comma, a table cell border and a line break, and
// every scan finds a contradiction worded independently of its pattern, a paraphrase, an imperative and the
// contradiction under a negated condition while it passes the shipped wording. The scans are phrase lists: a
// contradiction in words none of them lists passes (decision 861d392).
func TestRoutePart1Hits(t *testing.T) {
	t.Parallel()

	for _, scan := range routePart1Scans() {
		t.Run(scan.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(routePart1Hits(scan.independent, scan)).NotTo(BeEmpty())
			g.Expect(routePart1Hits(scan.paraphrase, scan)).NotTo(BeEmpty())
			g.Expect(routePart1Hits(scan.imperative, scan)).NotTo(BeEmpty())
			g.Expect(routePart1Hits(scan.conditional, scan)).NotTo(BeEmpty())
			g.Expect(routePart1Hits(scan.stated, scan)).To(BeEmpty())
		})
	}

	inline := routePart1ObjectLevelScan

	for name, scenario := range map[string]struct {
		content string
		want    int
	}{
		"a negated sentence is a prohibition": {"Do not do object-level work yourself.", 0},
		"an instruction is a hit":             {"Do the object-level work yourself.", 1},
		"a table row after a negated row is a hit": {
			"| Rule | Never delegate |\n| Next | Do the object-level work yourself |\n", 1,
		},
		"a cell after a negated cell is a hit":           {"| No tiers | Do the object-level work yourself |", 1},
		"a list item after a negated item is a hit":      {"- Never summarize\n- Do the object-level work yourself\n", 1},
		"a clause after a colon is a hit":                {"Never skip the check: do the object-level work yourself.", 1},
		"a clause after a comma is a hit":                {"If it isn't large, do the object-level work yourself.", 1},
		"a negation in a wrapped line stays in the unit": {"You do not\ndo object-level work yourself.", 0},
		"a quoted instruction in a blockquote is a hit":  {"> Do the object-level work yourself.", 1},
		"a leading without phrase does not excuse it": {
			"Without waiting for a dispatch do the object-level work yourself.", 1,
		},
		"a leading no matter phrase does not excuse it": {
			"No matter how small do the object-level work yourself.", 1,
		},
		"a negation just before the verb is a prohibition": {"You must never do the object-level work yourself.", 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(routePart1Hits(scenario.content, inline)).To(HaveLen(scenario.want))
		})
	}
}

// unexported constants.
const (
	routePart1ColdStartHeading     = "## Cold-start priors"
	routePart1NegationReach        = 4
	routePart1OrchestrationHeading = "## Orchestration work vs object-level work"
	routePart1RedFlagsHeading      = "## Red flags — STOP and re-read"
	routePart1SkillPath            = "agent-instructions/skills/route/SKILL.md"
	routePart1TierHeading          = "## How to pick a tier"
)

// unexported variables.
var (
	// routePart1AboveEvidenceScan finds an instruction to start above the tier the evidence supports.
	routePart1AboveEvidenceScan = routePart1Scan{
		name: "starting above the evidence",
		pattern: regexp.MustCompile(`(?i)\b(?:start|begin|dispatch|route)\b[^.\n]{0,40}` +
			`\b(?:above|higher than|beyond|over)\b[^.\n]{0,30}\bevidence\b|` +
			`\b(?:add|bump|pad)\b[^.\n]{0,25}\b(?:an extra|a|one)\b[^.\n]{0,10}\btier\b` +
			`[^.\n]{0,40}\b(?:safe|margin|buffer|just in case)\b|\b(?:start|begin)\b[^.\n]{0,20}` +
			`\b(?:mid|deep|higher|one tier up|a tier up)\b[^.\n]{0,40}\b(?:to be safe|for safety|as a precaution|` +
			`just in case|as insurance|for margin)\b`),
		independent: "Begin at the deep tier as insurance when the evidence is thin.",
		paraphrase:  "Start one tier above what the evidence supports, to be safe.",
		imperative:  "Bump the start by an extra tier as a margin.",
		conditional: "If the evidence isn't recent, start above the tier the evidence supports.",
		stated:      "Recalled evidence sets the starting tier, and the unit never starts above what it supports.",
	}
	// routePart1BeyondDeepScan finds an instruction to escalate past the deep tier.
	routePart1BeyondDeepScan = routePart1Scan{
		name: "escalating beyond deep",
		pattern: regexp.MustCompile(`(?i)\b(?:escalate|escalates|escalating|go|move|step|retry|retries)\b[^.\n]{0,30}` +
			`\b(?:beyond|past|above)\b[^.\n]{0,20}\bdeep\b|\b(?:tier|model)\b[^.\n]{0,20}\b(?:above|beyond|past)\b` +
			`[^.\n]{0,10}\bdeep\b|\bdeep\b[^.\n]{0,40}\b(?:escalate|escalating|go|move|step)\b[^.\n]{0,20}` +
			`\bto (?:a|an|the)\b[^.\n]{0,15}\b(?:larger|bigger|stronger|more capable|next)\b`),
		independent: "If deep fails too, escalate to a larger model.",
		paraphrase:  "Keep escalating past the deep tier until it passes.",
		imperative:  "Escalate beyond deep to a larger model.",
		conditional: "If deep also isn't enough, move above the deep tier.",
		stated:      "Repeat until it passes or you reach the deep tier; do not escalate beyond deep.",
	}
	// routePart1ClauseBorders are the strings that end a clause of an instruction, in a block of a shipped file.
	routePart1ClauseBorders = []string{". ", "? ", "! ", ": ", "| ", ", ", "; ", " — ", " (", "\n"}
	// routePart1ClauseEnds are the strings that end the clause that follows a match.
	routePart1ClauseEnds = []string{". ", "? ", "! ", ": ", " |", ", ", "; ", " — ", "\n"}
	// routePart1CountToRouteScan finds an instruction to run engram count in order to route.
	routePart1CountToRouteScan = routePart1Scan{
		name: "counting to route",
		pattern: regexp.MustCompile(`(?i)\b(?:run|use|call|invoke|execute|issue)\b[^.\n]{0,30}\bengram count\b[^.\n]{0,50}` +
			`\b(?:to route|for routing|when routing|to pick|to choose|before (?:you )?dispatch\w*|on the read path)\b` +
			`|\bengram count\b[^.\n]{0,25}\b(?:to route|for routing|to pick the tier|to choose the tier)\b|` +
			`\b(?:check|query|read|run|use|call)\b[^.\n]{0,15}\bengram count\b[^.\n]{0,30}\b(?:tier tallies|tallies)\b|` +
			`\bengram count\b[^.\n]{0,30}\b(?:before|when|while|prior to) (?:routing|you route|dispatch\w*)\b`),
		independent: "Before routing, check engram count for the tier tallies.",
		paraphrase:  "Query engram count for routing.",
		imperative:  "Run engram count before you dispatch to read the tallies.",
		conditional: "If the aggregate isn't surfaced, run engram count to route the unit.",
		stated:      "Use engram count to verify an aggregate, never to route.",
	}
	// routePart1DelegatingScan finds an instruction to delegate the orchestration itself.
	routePart1DelegatingScan = routePart1Scan{
		name: "delegating the orchestration",
		pattern: regexp.MustCompile(`(?i)\bdelegate\b[^.,;:\n]{0,40}\b(?:the act of delegating|routing|dispatching|` +
			`decomposition|synthesi[sz]\w*|the next decision|sequencing)\b|\b(?:ask|have|let|tell)\b[^.,;:\n]{0,20}` +
			`\ba (?:subagent|helper)\b[^.,;:\n]{0,30}\b(?:synthesi[sz]e|decide|route|dispatch|sequence|plan)\b|` +
			`\b(?:hand|pass|give|send|push)\b[^.,;:\n]{0,30}\b(?:synthesi[sz]\w*|the next decision|routing|sequencing|` +
			`the decision)\b[^.,;:\n]{0,40}\bto\b[^.,;:\n]{0,15}\b(?:subagent|helper|agent|model)\b`),
		exempt:      regexp.MustCompile(`(?i)\bfrom collapsing\b`),
		independent: "Hand the synthesis of the returned results to a subagent.",
		paraphrase:  "Let a subagent decide the next step from the three returned results.",
		imperative:  "Delegate the routing and sequencing decisions to a cheap model.",
		conditional: "If the three results aren't conclusive, delegate the next decision to a subagent.",
		stated:      "You do not delegate the routing decision or the synthesis of the returned results.",
	}
	// routePart1EasyInlineScan finds an instruction to do easy work inline or to skip the dispatch of it.
	routePart1EasyInlineScan = routePart1Scan{
		name: "easy work inline",
		pattern: regexp.MustCompile(`(?i)\b(?:do|does|handle|make|perform|take care of)\b[^.\n]{0,40}` +
			`\b(?:easy|trivial|one-line|simple|small|quick)\b[^.\n]{0,40}\b(?:inline|yourself|directly|` +
			`without dispatch\w*|without delegat\w*)\b|\bthere is an? inline (?:escape|path|exception)\b|` +
			`\b(?:skip|bypass)\b[^.\n]{0,25}\b(?:dispatch\w*|delegat\w*|routing)\b[^.\n]{0,40}` +
			`\b(?:easy|trivial|one-line|simple|small|quick)\b|` +
			`\bjust (?:edit|rename|change|fix|patch|do|make)\b[^.,;:\n]{0,30}\b(?:yourself|inline|directly)\b`),
		independent: "For a one-line rename, just edit the file yourself.",
		paraphrase:  "Skip the dispatch for a one-line rename.",
		imperative:  "Do the easy rename inline.",
		conditional: "If the rename isn't risky, do the easy rename inline.",
		stated:      "There is no inline escape — easy work is delegated to a cheap model, not skipped.",
	}
	// routePart1HunchScan finds an instruction that counts predicted difficulty as evidence.
	routePart1HunchScan = routePart1Scan{
		name: "difficulty as evidence",
		pattern: regexp.MustCompile(`(?i)\b(?:predicted difficulty|difficulty|your sense|a hunch|the hunch|looks hard|` +
			`complex\w*|correctness-critical|broad impact)\b[^.\n]{0,40}\b(?:counts? as|serves as|qualifies as|is|are)\b` +
			`[^.\n]{0,20}\bevidence\b|\b(?:treat|count|take|accept|use)\b[^.\n]{0,40}` +
			`\b(?:hunch|sense|difficulty|prediction|impression)\b[^.\n]{0,40}\bas (?:\w+ ){0,2}evidence\b`),
		independent: "Treat the predicted difficulty as sufficient evidence for a higher start.",
		paraphrase:  "Predicted difficulty counts as evidence for a higher start.",
		imperative:  "Treat a hunch that the unit is hard as evidence.",
		conditional: "If the work isn't a rename, your sense that it is hard is evidence.",
		stated:      "Predicted difficulty is not recorded evidence.",
	}
	// routePart1NegationPattern marks a clause that forbids or excludes what it names.
	routePart1NegationPattern = regexp.MustCompile(`(?i)\b(?:not|never|no|neither|nor)\b|n't\b`)
	// routePart1NoMatterPattern finds "no matter", which concedes a condition and negates nothing.
	routePart1NoMatterPattern = regexp.MustCompile(`(?i)\bno matter\b`)
	// routePart1ObjectLevelScan finds an instruction for the orchestrator to do object-level work itself.
	routePart1ObjectLevelScan = routePart1Scan{
		name: "object-level work by the orchestrator",
		pattern: regexp.MustCompile(`(?i)\b(?:do|does|perform|handle|write|run|review)\b[^.\n]{0,30}\bobject-level work\b` +
			`[^.\n]{0,25}\b(?:yourself|inline|directly|itself)\b|\bobject-level work\b[^.\n]{0,25}` +
			`\b(?:inline|yourself|directly)\b|\b(?:write|run|review|judge)\b[^.\n]{0,25}` +
			`\b(?:the code|the tests|the build|the artifact|prose)\b[^.\n]{0,15}\b(?:yourself|inline)\b|` +
			`\b(?:implement|edit|fix|make|code|build)\b[^.,;:\n]{0,30}\b(?:yourself|inline|directly)\b`),
		independent: "When the fix is small, implement it yourself rather than dispatching.",
		paraphrase:  "Handle the object-level work yourself when it is small.",
		imperative:  "Run the tests inline.",
		conditional: "If the unit isn't large, write the code yourself.",
		stated:      "You do not do object-level work yourself.",
	}
	// routePart1PaidFirstScan finds an instruction to try a paid model ahead of a free local one.
	routePart1PaidFirstScan = routePart1Scan{
		name: "a paid model ahead of a free one",
		pattern: regexp.MustCompile(`(?i)\b(?:dispatch|send|route|use|prefer|try)\b[^.\n]{0,40}\b(?:paid|API)\b[^.\n]{0,30}` +
			`\b(?:before|over|instead of|ahead of|first)\b[^.\n]{0,25}\b(?:free|local)\b|\bpaid (?:API )?models? ` +
			`(?:is|are|gets?) (?:tried|preferred|used) first\b|\b(?:use|prefer|choose|pick|dispatch to)\b[^.,;:\n]{0,30}` +
			`\b(?:paid|API)\b[^.,;:\n]{0,20}\beven (?:when|if)\b[^.,;:\n]{0,30}\b(?:local|free)\b`),
		independent: "Use the paid model even when a local model is available.",
		paraphrase:  "A paid model is tried first, and the local model only as a fallback.",
		imperative:  "Prefer the paid API model over a local one.",
		conditional: "If the local model isn't registered yet, dispatch to the paid API model before the free one.",
		stated:      "A free local model is tried first; paid models enter the roster only above free-local options.",
	}
	// routePart1WholeUnitScan finds an instruction to dispatch a large unit whole.
	routePart1WholeUnitScan = routePart1Scan{
		name: "a large unit dispatched whole",
		pattern: regexp.MustCompile(`(?i)\b(?:dispatch|send|hand|give|route)\b[^.\n]{0,30}` +
			`\b(?:as is|as-is|as one|as a single|whole|unsplit|in one piece)\b|\bunits?\b[^.,;:\n]{0,20}` +
			`\b(?:may|can|should|could)\b[^.,;:\n]{0,10}\bgo (?:to|as)\b[^.,;:\n]{0,10}\b(?:a|one) (?:single )?` +
			`(?:agent|subagent)\b`),
		independent: "Large units may go to a single agent.",
		paraphrase:  "Send the whole unit to one agent.",
		imperative:  "Dispatch it as one agent.",
		conditional: "If the six packages aren't coupled, hand it over as a single agent.",
		stated:      "A unit that spans six packages is not dispatched as-is; break it into smaller units.",
	}
)

// routePart1Scan is one absence scan: a pattern for an instruction that contradicts a rule, an optional pattern for a
// clause about another rule that the scan skips, a contradiction worded independently of the pattern's vocabulary, a
// paraphrase of the contradiction, an imperative and the contradiction under a negated condition to insert, and the
// shipped wording that must pass.
type routePart1Scan struct {
	name        string
	pattern     *regexp.Regexp
	exempt      *regexp.Regexp
	independent string
	paraphrase  string
	imperative  string
	conditional string
	stated      string
}

// routePart1Contradictions returns, by path, the clauses of every shipped SKILL.md, references file and guidance
// document that the scan finds. It fails the test when the scan did not cover the route skill, so an empty result is
// not vacuous.
func routePart1Contradictions(t *testing.T, scan routePart1Scan) map[string][]string {
	t.Helper()

	found := make(map[string][]string)
	scannedRoute := false

	for _, path := range aibInstructionPaths(t) {
		if path == routePart1SkillPath {
			scannedRoute = true
		}

		content, err := os.ReadFile(filepath.Join(repositoryRoot, path))
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)

			return nil
		}

		hits := routePart1Hits(string(content), scan)
		if len(hits) > 0 {
			found[path] = hits
		}
	}

	if !scannedRoute {
		t.Fatalf("the instruction scan did not cover %s", routePart1SkillPath)
	}

	return found
}

// routePart1Hits returns the clause of every match of the scan's pattern in the content of a shipped file, except a
// match that a negation governs (a prohibition is not an instruction): a negation word inside the match or among the
// last words of its own clause before the match, and a match whose whole clause the exempt pattern matches. A clause
// runs from the last clause border before the match; a negation elsewhere in the sentence, such as a negated
// condition or a leading "without ..." phrase before the instruction, does not excuse it.
func routePart1Hits(content string, scan routePart1Scan) []string {
	text := learnPart1Blocks(content)
	hits := make([]string, 0)

	for _, location := range scan.pattern.FindAllStringIndex(text, -1) {
		start := 0

		for _, border := range routePart1ClauseBorders {
			if at := strings.LastIndex(text[:location[0]], border); at >= 0 {
				start = max(start, at+len(border))
			}
		}

		if routePart1Negated(text[start:location[0]], text[location[0]:location[1]]) {
			continue
		}

		end := len(text)

		for _, border := range routePart1ClauseEnds {
			if at := strings.Index(text[location[1]:], border); at >= 0 {
				end = min(end, location[1]+at)
			}
		}

		if scan.exempt != nil && scan.exempt.MatchString(text[start:end]) {
			continue
		}

		hits = append(hits, text[start:end])
	}

	return hits
}

// routePart1Negated reports whether a negation governs a match: one inside the matched text, or one among the last
// routePart1NegationReach words of the clause before it.
func routePart1Negated(before, matched string) bool {
	if routePart1NegationPattern.MatchString(routePart1NoMatterPattern.ReplaceAllString(matched, "")) {
		return true
	}

	words := strings.Fields(routePart1NoMatterPattern.ReplaceAllString(before, ""))

	return routePart1NegationPattern.MatchString(strings.Join(words[max(0, len(words)-routePart1NegationReach):], " "))
}

// routePart1Scans returns every absence scan of this file.
func routePart1Scans() []routePart1Scan {
	return []routePart1Scan{
		routePart1AboveEvidenceScan, routePart1BeyondDeepScan, routePart1CountToRouteScan, routePart1DelegatingScan,
		routePart1EasyInlineScan, routePart1HunchScan, routePart1ObjectLevelScan, routePart1PaidFirstScan,
		routePart1WholeUnitScan,
	}
}
