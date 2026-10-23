//go:build targ && integration

package dev

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a17i1a_APassingDispatchIsRecorded reads the route skill the repository ships: a dispatch whose review passes
// is recorded as one evidence row, collected into the mini-report table, with the work-kind, tier, model at dispatch,
// reason, outcome, duration and cost.
func Test792a17i1a_APassingDispatchIsRecorded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"After each dispatch resolves, record one evidence row for the mini-report AND make the structured vault " +
			"write"))
	g.Expect(text).To(ContainSubstring("| **work-kind** | your classification of the unit (see above) |"))
	g.Expect(text).To(ContainSubstring("| **tier used** | your routing decision (cheap / mid / deep) |"))
	g.Expect(text).To(ContainSubstring(
		"| **model (roster @ dispatch)** | the concrete model the tier resolved to *at dispatch time*"))
	g.Expect(text).To(ContainSubstring("| **why** |"))
	g.Expect(text).To(ContainSubstring("| **outcome** |"))
	g.Expect(text).To(ContainSubstring("| **duration** | the harness's reported wall-clock if it exposes one"))
	g.Expect(text).To(ContainSubstring("| **cost** | the harness's usage signal if it exposes one"))
	g.Expect(text).To(ContainSubstring(
		"collect the rows into a compact table in your user-facing report (the mini-report)"))
}

// Test792a17i1b_TheOutcomeIsTheVerdict reads the route skill the repository ships: the outcome of a dispatch is the
// review or gate verdict and never the subagent's own report of success, and no shipped instruction takes the outcome
// from the subagent's report.
func Test792a17i1b_TheOutcomeIsTheVerdict(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"| **outcome** | **the review/gate verdict** — PASS/FAIL. Never the subagent's self-report (it " +
			"confabulates — vault notes 148, 162). |"))
	g.Expect(text).To(ContainSubstring(
		"| You wrote the outcome from the subagent's \"done\" claim | Outcome = the review verdict, a ground-truth " +
			"artifact. Subagent reports confabulate (notes 148, 162). |"))
	g.Expect(routePart2Contradictions(t, routePart2OutcomeScan)).To(BeEmpty())
}

// Test792a17i1c_AnEscalatedDispatchRecordsWhereItPassed reads the route skill the repository ships: the escalation
// field of the row names the tier a unit that failed and escalated finally passed at.
func Test792a17i1c_AnEscalatedDispatchRecordsWhereItPassed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"| **escalation** | if it failed and escalated, the tier it finally passed at |"))
}

// Test792a17j1a_ARateIsKnown reads the route skill and its price table: a model with a row in the price table gets a
// dollar estimate computed from subagent_tokens and that row.
func Test792a17j1a_ARateIsKnown(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)
	table := shippedText(t, routePart2PriceTablePath)

	g.Expect(text).To(ContainSubstring(
		"**Look up the model in `price-table.md`:** a match → compute the $ estimate from `subagent_tokens` and " +
			"that row's rate, e.g. `45,231 tok (~$0.68 @ opus)`"))
	g.Expect(table).To(ContainSubstring("Look up the dispatch's model"))
	g.Expect(table).To(ContainSubstring("as an exact-match row key."))
	g.Expect(table).To(ContainSubstring("**Match found:** compute a blended $ estimate from `subagent_tokens`."))
	g.Expect(table).To(ContainSubstring("price each split against its own column and sum"))
}

// Test792a17j1b_NoRateIsKnown reads the route skill and its price table: a model with no row is shown as the
// unit-labeled token count and no dollar figure, and no shipped instruction computes a dollar figure for it.
func Test792a17j1b_NoRateIsKnown(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)
	table := shippedText(t, routePart2PriceTablePath)

	g.Expect(text).To(ContainSubstring(
		"fall back to the raw unit-labeled token count, e.g. `45,231 tok (no rate on hand)` — never fabricate " +
			"a $ figure."))
	g.Expect(table).To(ContainSubstring(
		"the mini-report MUST fall back to unit-labeled raw tokens — never compute or guess a $ figure."))
	g.Expect(table).To(ContainSubstring(
		"**No match found** (model not in this table, or the harness doesn't expose a model string at all"))
	g.Expect(table).To(ContainSubstring(
		"show the raw unit-labeled token count. Never fabricate a $ figure for an unmapped model."))
	g.Expect(routePart2Contradictions(t, routePart2DollarScan)).To(BeEmpty())
}

// Test792a17j1c_AHarnessWithNoUsageSignal reads the route skill the repository ships: a dispatch on Pi, which exposes
// no per-subagent usage signal, shows n/a for both cost and duration with the stated reason.
func Test792a17j1c_AHarnessWithNoUsageSignal(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring("**Pi** (`~/.pi/agent/` — the other harness this skill deploys to, per"))
	g.Expect(text).To(ContainSubstring(
		"has **no documented per-subagent usage or duration signal** as of this writing."))
	g.Expect(text).To(ContainSubstring(
		"Dispatches routed on Pi show `n/a` for both cost and duration, with that reason stated"))
	g.Expect(text).To(ContainSubstring("(e.g. `n/a — Pi exposes no per-subagent usage signal`)"))
}

// Test792a17j1d_ANonAnthropicPlatform reads the route skill and its price table: a model priced by Bedrock, Vertex or
// Foundry is treated as having no rate and the platform is noted in the mini-report.
func Test792a17j1d_ANonAnthropicPlatform(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)
	table := shippedText(t, routePart2PriceTablePath)

	g.Expect(table).To(ContainSubstring(
		"4. A model on a non-Anthropic platform (Bedrock, Vertex, Foundry) is priced differently by that platform"))
	g.Expect(table).To(ContainSubstring("treat it as \"no match\" here even if the bare model ID matches a row,"))
	g.Expect(table).To(ContainSubstring(
		"and note the platform in the mini-report (`n/a — Bedrock pricing not in this table`)."))
	g.Expect(text).To(ContainSubstring(
		"no match (model not in the table, or priced by a different platform — see the table's platform note)"))
}

// Test792a17j1e_ABareNumberIsNeverRecorded reads the route skill the repository ships: a cost in tokens is written
// with its unit, never as a bare number under a column named cost, and no shipped instruction records a bare number.
func Test792a17j1e_ABareNumberIsNeverRecorded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"Record the **unit explicitly** either way — never a bare number under a column named cost."))
	g.Expect(text).To(ContainSubstring(
		"duration: <duration_ms> ms; cost: <subagent_tokens> tok (<rate note, or 'no rate on hand'>)\""))
	g.Expect(routePart2Contradictions(t, routePart2BareNumberScan)).To(BeEmpty())
}

// Test792a17k1a_APassingDispatchGetsATaggedNote reads the route skill the repository ships: a dispatch with a passing
// verdict is handed to write-memory as a fact tagged work-kind, tier and outcome with the dispatch details in the
// object.
func Test792a17k1a_APassingDispatchGetsATaggedNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"This is a field handoff to the write-memory skill, not a CLI invocation you compose yourself"))
	g.Expect(text).To(ContainSubstring("- kind: fact"))
	g.Expect(text).To(ContainSubstring("- slug: `route-dispatch-<work-kind>`"))
	g.Expect(text).To(ContainSubstring("- tags: `work-kind/<k>`, `tier/<cheap|mid|deep>`,"))
	g.Expect(text).To(ContainSubstring("- object: \"<pass|fail> per review verdict;"))
	g.Expect(text).To(ContainSubstring("why: <recalled evidence|memory-discount|cold-start default>;"))
}

// Test792a17k1b_AFailingDispatchIsTaggedFailWithItsEscalation reads the route skill the repository ships: a dispatch
// that fails and escalates is handed to write-memory with the outcome tag family and an object that names the
// escalation.
func Test792a17k1b_AFailingDispatchIsTaggedFailWithItsEscalation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring("`outcome/<pass|fail>`"))
	g.Expect(text).To(ContainSubstring("escalation: <none|passed at TIER>;"))
}

// Test792a17k1c_TheTagFormatIsValidated reads the route skill the repository ships: the work-kind tag value is
// kebab-case from an open set, and tier and outcome are closed sets. The tags line itself is asserted by
// Test792a17k1a and Test792a17k1b, and the refusal of a malformed tag before the write is the binary's
// (Test792a3g1a_Learn_AnInvalidArgumentIsRefusedBeforeAnyWrite), so neither is asserted here (decision 871d1).
func Test792a17k1c_TheTagFormatIsValidated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring("Work-kind values are kebab-case, an open set"))
	g.Expect(text).To(ContainSubstring("tier and outcome are closed sets"))
}

// Test792a17k1d_FieldsNotFlags reads the route skill the repository ships: the evidence note is handed off as fields
// stated as written, with no invented flag such as --kind, and no shipped instruction passes one.
func Test792a17k1d_FieldsNotFlags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"do not invent flags for these fields (there is no `--kind` flag; write-memory maps the fields below to " +
			"the real `engram learn fact` flags)."))
	g.Expect(text).To(ContainSubstring(
		"State the fields exactly as written below, verbatim, not retabled or turned into flags — the first is " +
			"`kind: fact`:"))
	g.Expect(routePart2Contradictions(t, routePart2FlagScan)).To(BeEmpty())
}

// Test792a17l1a_TheFirstDispatchCreatesTheAggregate reads the route skill the repository ships: with no aggregate and
// a uniqueness check that finds nothing, the aggregate route-evidence-<work-kind> is created with a 1/1 tally, a link
// to the evidence note and no tags, and no shipped instruction tags it.
func Test792a17l1a_TheFirstDispatchCreatesTheAggregate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"**Uniqueness check finds nothing** → create the aggregate (NO tags — aggregates are prose summaries, not " +
			"evidence rows):"))
	g.Expect(text).To(ContainSubstring("engram learn fact --slug route-evidence-<work-kind> --position top"))
	g.Expect(text).To(ContainSubstring(
		"--object=\"<tier> 1/1 as of <date> — evidence: [[<evidence-note basename>]]\""))
	g.Expect(routePart2Contradictions(t, routePart2AggregateTagsScan)).To(BeEmpty())
}

// Test792a17l1b_ALaterDispatchAmendsIt reads the route skill the repository ships: a later dispatch recomputes the
// tier tally from the aggregate's text plus the dispatch, amends the aggregate and appends the new evidence wikilink
// while keeping the old ones.
func Test792a17l1b_ALaterDispatchAmendsIt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring("- **Match** → recompute the tally from the aggregate's current object text"))
	g.Expect(text).To(ContainSubstring("engram amend --target <matched basename, no .md>"))
	g.Expect(text).To(ContainSubstring(
		"— evidence: [[<existing evidence wikilinks, kept>]], [[<new evidence-note basename>]]\""))
	g.Expect(text).To(ContainSubstring("every evidence note it summarizes, append-only."))
}

// Test792a17l1c_AFuzzyMatchIsNotAMatch reads the route skill the repository ships: a returned note counts as the
// aggregate only when its slug equals route-evidence-<work-kind> exactly, a prefix or fuzzy match does not count, and
// no shipped instruction counts one.
func Test792a17l1c_AFuzzyMatchIsNotAMatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"take the basename (text after the last `/`), strip `.md`, and split on `.` — the final segment is the " +
			"slug."))
	g.Expect(text).To(ContainSubstring(
		"A match iff that slug EQUALS `route-evidence-<work-kind>` exactly. Prefix/fuzzy matches do not count."))
	g.Expect(routePart2Contradictions(t, routePart2FuzzyMatchScan)).To(BeEmpty())
}

// Test792a17l1d_ANoMatchResultIsCheckedBeforeCreating reads the route skill the repository ships: on no match the
// orchestrator first lists vault basenames matching route-evidence-<work-kind>* under the same exact rule, to rule out
// a missed lookup, before it creates an aggregate.
func Test792a17l1d_ANoMatchResultIsCheckedBeforeCreating(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"- **No match** → before creating a new aggregate, run a uniqueness check to rule out a missed lookup"))
	g.Expect(text).To(ContainSubstring("list vault basenames matching `route-evidence-<work-kind>*`"))
	g.Expect(text).To(ContainSubstring("EQUAL `route-evidence-<work-kind>` exactly)."))
	g.Expect(text).To(ContainSubstring("plus every no-match result here."))
}

// Test792a17l1e_DuplicatesAreMerged reads the route skill the repository ships: when the uniqueness check finds more
// than one aggregate for a work-kind, the evidence wikilinks are united, the tally is recomputed by count, one
// aggregate is amended into the merged one and the incident is recorded.
func Test792a17l1e_DuplicatesAreMerged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring("**Uniqueness check finds more than one match** → merge instead of creating:"))
	g.Expect(text).To(ContainSubstring("union the evidence wikilinks from every matched aggregate's object text,"))
	g.Expect(text).To(ContainSubstring(
		"recompute the tally via the `engram count --group-by tags` commands documented below,"))
	g.Expect(text).To(ContainSubstring("`engram amend` one of them into the merged aggregate,"))
	g.Expect(text).To(ContainSubstring("and record the incident as evidence for the 792a6c drowning-remedy decision"))
}

// Test792a17l1f_TheWritePathSplitIsStated reads the route skill the repository ships: the evidence note goes through
// write-memory while the aggregate is composed and run directly, because write-memory has no amend form, and no
// shipped instruction hands the aggregate to write-memory.
func Test792a17l1f_TheWritePathSplitIsStated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring("**(a) Evidence note — hand off to write-memory.**"))
	g.Expect(text).To(ContainSubstring(
		"**(b) Aggregate update — amend or create `route-evidence-<work-kind>`.** Unlike (a), this write is " +
			"composed and executed directly by you, the route-executing agent, not handed to write-memory"))
	g.Expect(text).To(ContainSubstring(
		"write-memory has no amend form, so this is a deliberate, structurally-forced exception to the write-site " +
			"doctrine"))
	g.Expect(routePart2Contradictions(t, routePart2WriteMemoryScan)).To(BeEmpty())
}

// Test792a17m1a_AWorkKindIsAudited reads the route skill the repository ships: a doubted tally is audited with engram
// count grouped by tags with the work-kind and tier filters, whose numerators come from the evidence tags.
func Test792a17m1a_AWorkKindIsAudited(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring("`engram count` recomputes ground truth from the evidence notes' tags"))
	g.Expect(text).To(ContainSubstring(
		"engram count --group-by tags --filter tags=work-kind/<k> --filter tags=tier/<t>"))
	g.Expect(text).To(ContainSubstring("# single-kind spot check: read the \"total:\" line"))
	g.Expect(text).To(ContainSubstring(
		"engram count --group-by tags --filter tags=tier/<t> --filter tags=outcome/pass"))
	g.Expect(text).To(ContainSubstring("Run this when a tally is doubted and at periodic consolidation"))
}

// Test792a17m1b_ADriftedAggregateIsRepaired reads the route skill the repository ships: when the count output
// disagrees with an aggregate's tally, the aggregate is amended to the recomputed numbers.
func Test792a17m1b_ADriftedAggregateIsRepaired(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring("Per kind: numerator P over denominator D → \"<t> P/D\"."))
	g.Expect(text).To(ContainSubstring(
		"if count disagrees with an aggregate, amend the aggregate to the recomputed numbers."))
}

// Test792a17m1c_TheFootgunIsAvoided reads the route skill the repository ships: the count by evidence tier is composed
// with --group-by tags --filter tags=tier/<t> and never with --group-by tier, which groups the note-tier frontmatter
// field, and no shipped instruction uses --group-by tier.
func Test792a17m1c_TheFootgunIsAvoided(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"Also note `--group-by tier` runs without error but groups the wrong attribute"))
	g.Expect(text).To(ContainSubstring(
		"the pre-existing L1/L2/L3 note-tier frontmatter field, not the `tier/<cheap|mid|deep>` evidence tag " +
			"family route uses."))
	g.Expect(text).To(ContainSubstring("Use `--group-by tags --filter tags=tier/<t>` instead."))
	g.Expect(routePart2Contradictions(t, routePart2GroupByTierScan)).To(BeEmpty())
}

// Test792a17n1b_AMissingAggregateIsReported reads the route skill the repository ships: a known aggregate that does
// not surface in the lookup is the pre-registered drowning case, which is reported and not patched ad hoc, and no
// shipped instruction patches the skill for it.
func Test792a17n1b_AMissingAggregateIsReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"**Drowning audit (same trigger moments):** run the (b) lookup query for a work-kind with many evidence " +
			"notes"))
	g.Expect(text).To(ContainSubstring(
		"confirm the aggregate still surfaces (its `path:` in items or any cluster's candidate_l2s)."))
	g.Expect(text).To(ContainSubstring(
		"If it does not, that is the pre-registered drowning case — report it rather than patching ad hoc;"))
	g.Expect(routePart2Contradictions(t, routePart2PatchScan)).To(BeEmpty())
}

// Test792a17o1a_AConfirmingTierOutcomeIsAConfirmedApproach reads the route skill the repository ships: a strong and
// general finding that a tier's outcome confirms the routing is crystallized through learn as a confirmed approach.
func Test792a17o1a_AConfirmingTierOutcomeIsAConfirmedApproach(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring(
		"When a finding is strong and general (\"cheap failed N× on work-kind K — needs mid\"), crystallize it via " +
			"`/learn`:"))
	g.Expect(text).To(ContainSubstring(
		"a **confirmed approach** (kind 4) when a tier's outcome confirms the routing (e.g. \"cheap sufficed for " +
			"work-kind K\")"))
}

// Test792a17o1b_AnOverturnedTierAssumptionIsAReversal reads the route skill the repository ships: a strong and general
// finding that overturns a prior tier assumption is crystallized through learn as a reversal.
func Test792a17o1b_AnOverturnedTierAssumptionIsAReversal(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, routePart2SkillPath)

	g.Expect(text).To(ContainSubstring("or a **reversal** if it overturns a prior tier assumption."))
}

// TestRoutePart2Scans shows the scanner of the absence checks of this file finds an instruction that contradicts each
// rule, in a paraphrase, as an imperative, as an optional step, with a modal verb, under a negated condition with and
// without a comma and with a negation that sits in an aside, and passes the shipped wording that states the rule and
// the legitimate wording that sits near it.
func TestRoutePart2Scans(t *testing.T) {
	t.Parallel()

	for _, scan := range routePart2Scans() {
		t.Run(scan.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(routePart2Hits(scan.paraphrase, scan)).NotTo(BeEmpty())
			g.Expect(routePart2Hits(scan.imperative, scan)).NotTo(BeEmpty())
			g.Expect(routePart2Hits(scan.optional, scan)).NotTo(BeEmpty())
			g.Expect(routePart2Hits(scan.modal, scan)).NotTo(BeEmpty())
			g.Expect(routePart2Hits(scan.conditional, scan)).NotTo(BeEmpty())
			g.Expect(routePart2Hits(scan.uncommaed, scan)).NotTo(BeEmpty())
			g.Expect(routePart2Hits(scan.stated, scan)).To(BeEmpty())

			if scan.aside != "" {
				g.Expect(routePart2Hits(scan.aside, scan)).NotTo(BeEmpty())
			}

			if scan.legitimate != "" {
				g.Expect(routePart2Hits(scan.legitimate, scan)).To(BeEmpty())
			}
		})
	}
}

// unexported constants.
const (
	routePart2LeadWindow     = 200
	routePart2PriceTablePath = "agent-instructions/skills/route/price-table.md"
	routePart2SkillPath      = "agent-instructions/skills/route/SKILL.md"
)

// unexported variables.
var (
	routePart2AggregateTagsScan = routePart2Scan{
		name: "tags on the aggregate",
		pattern: regexp.MustCompile(
			`(?i)route-evidence-<work-kind>[^\n]{0,400}--tag\b` +
				`|` + routePart2Verbs("add", "attach", "give", "apply", "put", "carry", "tag") +
				`[^.\n]{0,40}\baggregates?\b[^.\n]{0,40}\btags?\b` +
				`|` + routePart2Verbs("add", "attach", "give", "apply", "put") +
				`[^.\n]{0,30}\btags?\b[^.\n]{0,30}\baggregates?\b` +
				`|\baggregates?\b[^.\n]{0,30}\b(?:carries|carry|gets|has|takes)\b[^.\n]{0,50}\btags?\b`),
		paraphrase:  "The aggregate carries the work-kind, tier and outcome tags.",
		imperative:  "Tag the aggregate with the outcome tags.",
		optional:    "Tagging the aggregate with the outcome tags is optional.",
		modal:       "The orchestrator may tag the aggregate with the outcome tags.",
		conditional: "If the lookup isn't clean, tag the aggregate with the outcome tags.",
		uncommaed:   "If the lookup is not clean tag the aggregate with the outcome tags.",
		stated:      "Create the aggregate (NO tags — aggregates are prose summaries, not evidence rows).",
	}
	routePart2BareNumberScan = routePart2Scan{
		name: "a bare number as the cost",
		pattern: regexp.MustCompile(
			`(?i)` + routePart2Verbs("record", "write", "show", "render", "report", "put", "use") + `[^.\n]{0,30}` +
				`\b(?:bare|unlabeled|unlabelled|unitless)\s+(?:number|count|figure|token count)\b` +
				`|\b(?:bare|unlabeled|unlabelled|unitless)\s+(?:number|count|figure|token count)\b[^.\n]{0,40}` +
				`\b(?:is|are|may be|can be|could be|should be|might be)\s+` +
				`(?:fine|ok|okay|acceptable|enough|allowed|recorded|written|shown|used|reported)\b`),
		paraphrase:  "A unitless count is acceptable in the cost column.",
		imperative:  "Record a bare number under the cost column.",
		optional:    "Recording a bare number under the cost column is fine.",
		modal:       "A bare number may be recorded under the cost column.",
		conditional: "If the harness is quiet, record a bare number under the cost column.",
		uncommaed:   "If the harness is quiet record a bare number under the cost column.",
		stated:      "Record the unit explicitly either way — never a bare number under a column named cost.",
	}
	// routePart2ClauseBoundary marks where a clause of a sentence ends, so that a negation in an earlier clause does
	// not reach a later one.
	routePart2ClauseBoundary = regexp.MustCompile(`[,;(—]`)
	// routePart2ConditionLead marks a clause that is a condition, whose own negation does not negate the instruction
	// that follows it.
	routePart2ConditionLead = regexp.MustCompile(`(?i)^\s*(?:if|when|whenever|unless|once|while|since|because|in case)\b`)
	routePart2DollarScan    = routePart2Scan{
		name: "a dollar figure without a rate",
		pattern: regexp.MustCompile(
			`(?i)` + routePart2Verbs("compute", "estimate", "guess", "fabricate", "invent", "show", "record", "render",
				"derive", "infer", "approximate") +
				`[^.\n]{0,30}(?:\$|dollar)\s?(?:figure|amount|number|value|cost)\b`),
		context: regexp.MustCompile(
			`(?i)\bunmapped\b|\bno\s+(?:matching\s+)?(?:row|rate|match)\b|\bnot in (?:the|this) (?:price )?table\b` +
				`|\bwithout (?:a )?(?:row|rate)\b|\bdifferent platform\b|\bnon-anthropic\b|\bunknown model\b`),
		paraphrase:  "For an unmapped model, show a dollar amount from the nearest rate.",
		imperative:  "Guess a $ figure for a model with no row.",
		optional:    "Guessing a $ figure for a model with no row is optional.",
		modal:       "The orchestrator may estimate a dollar cost for an unmapped model.",
		conditional: "If the harness isn't Pi, guess a $ figure for a model with no row.",
		uncommaed:   "If the harness is not Pi guess a $ figure for a model with no row.",
		stated:      "Never fabricate a $ figure for an unmapped model.",
		legitimate:  "For a model with a matching row, compute the dollar cost from that row's rate.",
	}
	routePart2FlagScan = routePart2Scan{
		name: "a --kind flag",
		pattern: regexp.MustCompile(
			`(?i)` + routePart2Verbs("use", "pass", "add", "include", "supply", "hand", "give", "set") +
				`[^.\n]{0,40}--kind\b` +
				`|--kind\W{0,3}\s+(?:flag|option)\s+(?:sets|selects|takes|accepts|is)\b`),
		paraphrase:  "The `--kind` flag selects the note kind.",
		imperative:  "Pass `--kind fact` on the handoff.",
		optional:    "Passing a `--kind` flag on the handoff is optional.",
		modal:       "The orchestrator may pass `--kind fact` on the handoff.",
		conditional: "If write-memory isn't loaded, pass `--kind fact`.",
		uncommaed:   "If write-memory is not loaded pass `--kind fact`.",
		stated:      "Do not invent flags for these fields (there is no `--kind` flag).",
	}
	routePart2FuzzyMatchScan = routePart2Scan{
		name: "a prefix or fuzzy match",
		pattern: regexp.MustCompile(
			`(?i)` + routePart2Verbs("treat", "count", "accept", "take") +
				`[^.\n]{0,40}\b(?:prefix|fuzzy|partial|near)\b[^.\n]{0,20}\bmatch\w*` +
				`|\b(?:prefix|fuzzy|partial)\s+(?:match|matches)\b[^.\n]{0,20}` +
				`\b(?:count|counts|suffices|(?:is|are|may be|can be|could be|should be) ` +
				`(?:enough|fine|accepted|treated|counted|taken))\b` +
				`|\bslug\b[^.\n]{0,40}\b(?:starts?|begins?)\s+with\b[^.\n]{0,60}` +
				`\b(?:is|counts as|treat\w*\s+as)\b[^.\n]{0,15}\b(?:a match|the aggregate|a hit)`),
		paraphrase:  "A note whose slug begins with route-evidence-<work-kind> is the aggregate.",
		imperative:  "Treat a prefix match as the aggregate.",
		optional:    "Counting a prefix match as the aggregate is optional.",
		modal:       "The orchestrator may accept a prefix match as the aggregate.",
		conditional: "If the exact slug isn't found, treat a prefix match as the aggregate.",
		uncommaed:   "If the exact slug is not found treat a prefix match as the aggregate.",
		stated:      "Prefix/fuzzy matches do not count.",
	}
	routePart2GroupByTierScan = routePart2Scan{
		name: "grouping the count by tier",
		pattern: regexp.MustCompile(
			`(?i)` + routePart2Verbs("use", "run", "pass", "group", "prefer", "try") +
				`[^.\n]{0,30}--group-by\s+tier(?:[^/\w-]|$)` +
				`|--group-by\s+tier(?:[^/\w-]|$)\W{0,3}\s*(?:works|is correct|is fine|is right|gives)` +
				`|\bengram count\b[^\n]{0,30}--group-by\s+tier(?:[^/\w-]|$)`),
		paraphrase:  "`--group-by tier` gives the evidence tier counts.",
		imperative:  "Run `engram count --group-by tier`.",
		optional:    "Running `engram count --group-by tier` is fine.",
		modal:       "The orchestrator may use `--group-by tier`.",
		conditional: "If the tag family is missing, use `--group-by tier`.",
		uncommaed:   "If the tag family is missing use `--group-by tier`.",
		stated:      "Use `--group-by tags --filter tags=tier/<t>` instead.",
	}
	// routePart2NegationBefore marks a negation that ends the text before an instruction, as in "do not" or "never".
	routePart2NegationBefore = regexp.MustCompile(
		`(?i)(?:\bnot|\bnever|\bno|\bnor|\bneither|\bwithout|n't)(?:\s+(?:ever|also|simply|just|again|even|only))?\s*$`)
	// routePart2NegationPattern marks a clause that forbids or excludes what it names.
	routePart2NegationPattern = regexp.MustCompile(`(?i)\b(?:not|never|neither|nor|no|without)\b|n't\b`)
	routePart2OutcomeScan     = routePart2Scan{
		name: "the outcome from the subagent's report",
		pattern: regexp.MustCompile(
			`(?i)` + routePart2Verbs("take", "use", "record", "write", "set", "derive", "accept") +
				`[^.\n]{0,30}\boutcome\b[^.\n]{0,40}` +
				`\b(?:subagent|sub-agent)(?:'s|s)?\s+(?:self-?report|report|claim|word)` +
				`|` + routePart2Verbs("take", "use", "accept", "record", "treat", "derive", "read", "trust") +
				`[^.\n]{0,30}` +
				`\b(?:subagent|sub-agent)(?:'s|s)?\s+(?:self-?report|report|claim|word)\b[^.\n]{0,30}` +
				`\b(?:as|for)\b[^.\n]{0,15}\boutcome\b` +
				`|\boutcome\b[^.\n]{0,30}\b(?:is|comes from|taken from|follows)\b[^.\n]{0,20}` +
				`\b(?:subagent|sub-agent)(?:'s|s)?\s+(?:self-?report|report|claim|word)` +
				`|` + routePart2Verbs("trust", "believe", "accept") + `[^.\n]{0,20}\b(?:subagent|sub-agent)(?:'s|s)?\s+` +
				`(?:self-?report|report|claim|word)`),
		paraphrase:  "The dispatch outcome is whatever the subagent's report says.",
		imperative:  "Record the outcome from the subagent's self-report.",
		optional:    "Taking the outcome from the subagent's report is optional.",
		modal:       "The orchestrator may take the subagent's report as the outcome.",
		conditional: "If the reviewer isn't available, record the outcome from the subagent's report.",
		uncommaed:   "If the reviewer is not available record the outcome from the subagent's report.",
		stated:      "Never the subagent's self-report (it confabulates).",
	}
	routePart2PatchScan = routePart2Scan{
		name: "patching the skill ad hoc",
		pattern: regexp.MustCompile(
			`(?i)` + routePart2Verbs("patch", "fix", "edit", "rewrite", "amend", "change", "modify", "update") +
				`[^.\n]{0,25}\b(?:this|the|route)\b` +
				`[^.\n]{0,12}\b(?:skill|SKILL\.md|lookup query)\b` +
				`|\b(?:fixed|patched|remedied|handled)\s+by\s+(?:editing|patching|changing|rewriting)\b`),
		context:     regexp.MustCompile(`(?i)drown\w*|aggregate|lookup|surfac\w*|missing|not found|route-evidence`),
		paraphrase:  "A missing aggregate is fixed by editing the skill.",
		imperative:  "Patch the skill so the lookup surfaces the aggregate.",
		optional:    "Patching the skill ad hoc is optional when the aggregate drowns.",
		modal:       "The orchestrator may patch the skill when the aggregate drowns.",
		conditional: "If the aggregate is not found, patch the skill ad hoc.",
		uncommaed:   "If the aggregate is not found patch the skill ad hoc.",
		stated:      "Report it rather than patching ad hoc.",
		legitimate:  "When a cold-start prior here is outdated, update the skill through writing-skills TDD.",
	}
	routePart2WriteMemoryScan = routePart2Scan{
		name: "handing the aggregate to write-memory",
		pattern: regexp.MustCompile(
			`(?i)` + routePart2Verbs("hand", "pass", "send", "route", "give", "delegate") +
				`[^.\n]{0,40}\baggregate\b[^.\n]{0,40}\bwrite-memory\b` +
				`|\baggregate\b[^.\n]{0,40}\b(?:goes|go|is handed|handed)\b[^.\n]{0,20}\b(?:through|to)\b` +
				`[^.\n]{0,10}write-memory`),
		paraphrase:  "The aggregate write goes through write-memory like the evidence note.",
		imperative:  "Hand the aggregate to write-memory.",
		optional:    "Handing the aggregate to write-memory is optional.",
		modal:       "The orchestrator may hand the aggregate to write-memory.",
		conditional: "If amend is unavailable, hand the aggregate to write-memory.",
		uncommaed:   "If amend is unavailable hand the aggregate to write-memory.",
		aside:       "Hand the aggregate, not only the evidence note, to write-memory.",
		stated:      "The aggregate is composed directly, not handed to write-memory.",
	}
)

// routePart2Scan is one absence scan: a pattern for an instruction that contradicts a rule, an optional context that
// the sentence of a match must also match, a contradiction worded as a paraphrase, as an imperative, as an optional
// step, with a modal verb, under a negated condition with and without a comma and, for some scans, with a negation in
// an aside, the shipped wording that must pass and, for some scans, a legitimate wording that must pass.
type routePart2Scan struct {
	name        string
	pattern     *regexp.Regexp
	context     *regexp.Regexp
	paraphrase  string
	imperative  string
	optional    string
	modal       string
	conditional string
	uncommaed   string
	aside       string
	stated      string
	legitimate  string
}

// routePart2ClauseStart returns the offset in the lead where the clause that precedes a match begins: after the last
// sentence end, colon, table cell border, line break, comma, semicolon, dash or parenthesis.
func routePart2ClauseStart(lead string) int {
	start := learnPart1Boundary(lead)

	for _, boundary := range routePart2ClauseBoundary.FindAllStringIndex(lead, -1) {
		start = max(start, boundary[1])
	}

	return start
}

// routePart2Contradictions returns, by path, the matches of the scan in every shipped SKILL.md, references file and
// guidance document, and in the price table of the route skill. It fails the test when the scan did not cover the
// route skill, so an empty result is not vacuous.
func routePart2Contradictions(t *testing.T, scan routePart2Scan) []string {
	t.Helper()

	paths := aibInstructionPaths(t)
	if !slices.Contains(paths, routePart2PriceTablePath) {
		paths = append(paths, routePart2PriceTablePath)
	}

	if !slices.Contains(paths, routePart2SkillPath) {
		t.Fatalf("the instruction scan did not cover %s", routePart2SkillPath)

		return nil
	}

	found := make([]string, 0)

	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(repositoryRoot, path))
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)

			return nil
		}

		for _, hit := range routePart2Hits(string(content), scan) {
			found = append(found, path+": "+hit)
		}
	}

	return found
}

// routePart2Excused reports whether the match is a prohibition and not an instruction: a negation in the clause before
// the match (for a clause that opens with a condition, only a negation right before the match, since the negation of
// a condition does not negate the instruction), or a negation in the first or last clause of the match itself. A
// negation in a middle clause of the match, as in an aside, does not excuse it.
func routePart2Excused(text string, location []int) bool {
	lead := text[max(0, location[0]-routePart2LeadWindow):location[0]]
	clause := lead[routePart2ClauseStart(lead):]

	if routePart2ConditionLead.MatchString(clause) {
		if routePart2NegationBefore.MatchString(clause) {
			return true
		}
	} else if routePart2NegationPattern.MatchString(clause) {
		return true
	}

	segments := routePart2ClauseBoundary.Split(text[location[0]:location[1]], -1)

	return routePart2NegationPattern.MatchString(segments[0]) ||
		routePart2NegationPattern.MatchString(segments[len(segments)-1])
}

// routePart2Hits returns every match of the scan's pattern in the content that is not excused by a negation of its own
// clause and whose sentence matches the scan's context, when it has one. A negation elsewhere in the sentence, such as
// a negated condition before the instruction, does not excuse it.
func routePart2Hits(content string, scan routePart2Scan) []string {
	text := learnPart1Blocks(content)
	hits := make([]string, 0)

	for _, location := range scan.pattern.FindAllStringIndex(text, -1) {
		if routePart2Excused(text, location) {
			continue
		}

		if scan.context != nil && !scan.context.MatchString(routePart2Sentence(text, location)) {
			continue
		}

		hits = append(hits, text[location[0]:location[1]])
	}

	return hits
}

// routePart2Scans returns every absence scan of this file.
func routePart2Scans() []routePart2Scan {
	return []routePart2Scan{
		routePart2AggregateTagsScan, routePart2BareNumberScan, routePart2DollarScan, routePart2FlagScan,
		routePart2FuzzyMatchScan, routePart2GroupByTierScan, routePart2OutcomeScan, routePart2PatchScan,
		routePart2WriteMemoryScan,
	}
}

// routePart2Sentence returns the sentence of the text that holds the match: from the last sentence end or line break
// before it to the first one after it.
func routePart2Sentence(text string, location []int) string {
	start := 0

	for _, border := range []string{". ", "? ", "! ", "\n"} {
		if at := strings.LastIndex(text[:location[0]], border); at >= 0 {
			start = max(start, at+len(border))
		}
	}

	end := len(text)

	for _, border := range []string{". ", "? ", "! ", "\n"} {
		if at := strings.Index(text[location[1]:], border); at >= 0 {
			end = min(end, location[1]+at)
		}
	}

	return text[start:end]
}

// routePart2Verbs returns a pattern for any inflection of the verbs: the verb itself, its third person, past
// tense and gerund, so that a gerund subject such as "Tagging" is found as well as the imperative "Tag".
func routePart2Verbs(verbs ...string) string {
	forms := make([]string, 0, len(verbs))

	for _, verb := range verbs {
		switch {
		case strings.HasSuffix(verb, "e"):
			forms = append(forms, strings.TrimSuffix(verb, "e")+`(?:e|es|ed|ing)`)
		case strings.HasSuffix(verb, "y"):
			forms = append(forms, strings.TrimSuffix(verb, "y")+`(?:y|ies|ied|ying)`)
		default:
			forms = append(forms, verb+`\w*`)
		}
	}

	return `\b(?:` + strings.Join(forms, "|") + `)\b`
}
