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

// Test792a22d1a_AServedAndAPulledDownOfferAreJudgedAlike reads the curate skill the repository ships: both kinds of
// pending offer land as the same kind of file and go through the one judgment of Step 2, and no shipped instruction
// judges one kind differently from the other.
func Test792a22d1a_AServedAndAPulledDownOfferAreJudgedAlike(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)

	g.Expect(text).To(ContainSubstring("An offer (a served `engram learn` from a child, or a parent note pulled down " +
		"by `engram activate`) lands as a **pending offer**"))
	g.Expect(text).To(ContainSubstring("Curation is that review — the same agent-judged covered/near/absent reasoning " +
		"`recall`'s Step 2.5 performs against query candidates, applied instead to a pending offer against the host " +
		"vault's existing notes."))
	g.Expect(text).To(ContainSubstring("## Step 2 — Judge each offer against the host vault's existing notes"))
	g.Expect(curateRulesContradictions(t, curateRulesUnevenScan)).To(BeEmpty())
}

// Test792a22d1b_JudgmentReadsTheContentNotAScore reads the curate skill the repository ships: an offer is judged by its
// content and never by a similarity score alone, so a high score against a note that states a different claim is a
// false positive, and covered means the existing note states the offer's claim; no shipped instruction decides
// covered, near or absent by a score.
func Test792a22d1b_JudgmentReadsTheContentNotAScore(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)

	g.Expect(text).To(ContainSubstring("Judge content, never a cosine/similarity score alone — recall's own documented " +
		"mistake, and the same trap here: a high score can still be an unrelated false positive, a low one can still " +
		"be the same claim in different words."))
	g.Expect(text).To(ContainSubstring("| **Covered** | an existing note already states the offer's claim, no material " +
		"omission |"))
	g.Expect(text).To(ContainSubstring("| You applied a cosine threshold to decide covered/near/absent | Judge content " +
		"— " +
		"recall's own documented mistake, same trap here |"))
	g.Expect(curateRulesContradictions(t, curateRulesScoreScan)).To(BeEmpty())
}

// Test792a22d1c_TheOffersAreFoundByScanningTheVault reads the curate skill the repository ships: engram query excludes
// pending offers by design, so the skill has the agent scan the vault's notes for pending: true; no shipped
// instruction finds the offers with engram query.
func Test792a22d1c_TheOffersAreFoundByScanningTheVault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)

	g.Expect(text).To(ContainSubstring("`engram query` excludes pending offers from its results by design — it can only " +
		"tell you THAT some exist (`pending_offers: true` in its payload), not which ones."))
	g.Expect(text).To(ContainSubstring("Scan the vault directly instead:"))
	g.Expect(text).To(ContainSubstring("grep -l '^pending: true$' <vault>/*.md"))
	g.Expect(text).To(ContainSubstring("| You used `engram query` to find pending offers | Query excludes them by " +
		"design — " +
		"scan the vault's `.md` files directly for `pending: true` |"))
	g.Expect(curateRulesBefore(text, "`engram query` excludes pending offers", "grep -l '^pending: true$'")).To(BeTrue())
	g.Expect(curateRulesContradictions(t, curateRulesQueryFindScan)).To(BeEmpty())
}

// Test792a22d1e_TheVaultIsCheckedAfterwards reads the curate skill the repository ships: after a pass the agent runs
// engram check, which confirms the vault invariants, and engram query no longer shows the curated offers as pending.
func Test792a22d1e_TheVaultIsCheckedAfterwards(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)

	g.Expect(text).To(ContainSubstring("## Step 4 — Verify"))
	g.Expect(text).To(ContainSubstring("Run `engram check` after curating — confirms vault invariants (Luhmann " +
		"structure, " +
		"wikilink graph) still hold."))
	g.Expect(text).To(ContainSubstring("`engram query` should no longer show the curated offers as pending;"))
	g.Expect(curateRulesBefore(text, "## Step 3 — ", "## Step 4 — Verify")).To(BeTrue())
}

// Test792a22e1a_AnAmendOfferIsJudgedAgainstItsTarget reads the curate skill the repository ships: an offer that
// carries offer.for is judged against the note it names before any query, it is usually near or covered, a near
// judgment folds it into the judged note by --discard --into, and no shipped instruction has the agent query first for
// such an offer.
func Test792a22e1a_AnAmendOfferIsJudgedAgainstItsTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)

	g.Expect(text).To(ContainSubstring("**An offer carrying `offer.for: <note>` is an amend-offer for that note: judge " +
		"it " +
		"against that note first** (`engram show <note>`), before any query. It is usually near (it adds a claim) or " +
		"covered."))
	g.Expect(text).To(ContainSubstring("| **Near** | overlaps an existing note's topic but adds ≥1 substantive claim " +
		"the " +
		"existing note omits | first fold the offer away: `engram amend --target <offer> --discard --into <existing> " +
		"--expect-hash <hash>`"))
	g.Expect(text).To(ContainSubstring("| You judged an `offer.for` offer by query alone | Judge it against the " +
		"`offer.for` " +
		"note first |"))
	g.Expect(curateRulesBefore(text, "judge it against that note first**",
		"Otherwise, `engram query --phrase=")).To(BeTrue())
	g.Expect(curateRulesContradictions(t, curateRulesQueryFirstScan)).To(BeEmpty())
}

// Test792a22i1b_ARejectedOfferIsNeverFoldedIn reads the curate skill the repository ships: a rejected offer is settled
// by a bare --discard and never by --into, because folding it into the note that contradicts it would record the bad
// note as covered by it; no shipped instruction folds a rejected offer in.
func Test792a22i1b_ARejectedOfferIsNeverFoldedIn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)

	g.Expect(text).To(ContainSubstring("| **Rejected** | the offer is wrong: an existing note (or a user correction) " +
		"contradicts it, or it is advice you would never apply | `engram amend --target <offer> --discard " +
		"--expect-hash <hash>` — a bare `--discard`, **never `--into`**."))
	g.Expect(text).To(ContainSubstring("Folding it `--into` the note that contradicts it would record the bad note as " +
		"covered by it"))
	g.Expect(text).To(ContainSubstring("| You folded a wrong or contradicted offer `--into` the note that contradicts " +
		"it | " +
		"Rejected is a bare `--discard`; `--into` means \"covered by\" |"))
	g.Expect(curateRulesContradictions(t, curateRulesFoldRejectedScan)).To(BeEmpty())
}

// Test792a22j1a_CurationNeverComposesANewLearn reads the curate skill the repository ships: every curation action is
// an engram amend call composed and executed directly, with no engram learn and no handoff to write-memory, and no
// shipped instruction has curation run engram learn or hand an offer off to write-memory.
func Test792a22j1a_CurationNeverComposesANewLearn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)

	g.Expect(text).To(ContainSubstring("**Never hand off to `write-memory`.**"))
	g.Expect(text).To(ContainSubstring("Every curation action above is an `engram amend` call, composed and executed " +
		"directly."))
	g.Expect(text).To(ContainSubstring("`write-memory`'s contract only composes brand-new `engram learn` calls from " +
		"scratch content fields — an offer's content already exists as a file, so there's nothing to compose from " +
		"scratch, not even in the absent case."))
	g.Expect(text).To(ContainSubstring("| You ran `engram learn` for the absent case | The note already exists as the " +
		"offer " +
		"— `--clear-pending` is the entire action, no new write |"))
	g.Expect(text).To(ContainSubstring("| You handed a judgment off to `write-memory` | write-memory composes brand-new " +
		"`engram learn` calls only; curation is self-contained `engram amend` |"))
	g.Expect(curateRulesContradictions(t, curateRulesLearnScan)).To(BeEmpty())
}

// Test792a22j1b_AFreeTextValueIsAttachedToItsFlag reads the curate skill the repository ships: a free-text flag value
// is written as --flag="value" with the quote touching the equals sign, because a value that begins with a dash is
// otherwise misread as a new flag and the command fails, and the skill's own amend command is written that way; no
// shipped instruction writes a free-text value after a space.
func Test792a22j1b_AFreeTextValueIsAttachedToItsFlag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)

	g.Expect(text).To(ContainSubstring("A free-text flag value (for example `--situation`, `--source`, `--subject`, " +
		"`--behavior`) must be written as `--flag=\"value\"` — the quote touching the `=`, no space — because a value " +
		"beginning with `-` is otherwise misread as a new flag and the command fails."))
	g.Expect(text).To(ContainSubstring("`engram amend --target <basename> --situation=\"<text>\" --done-when=\"<text>\" " +
		"--red-flag=\"<text>\" --trigger=\"<phrase>\"`"))
	g.Expect(curateRulesContradictions(t, curateRulesDetachedValueScan)).To(BeEmpty())
}

// Test792a22k1a_ANewSkillNoteGetsItsRunbookFields reads the curate skill the repository ships: a pending runbook note
// that carries skill_hash and has no runbook fields gets its situation, done_when, triggers and red_flags authored from
// the skill's body by an amend, then its marker is cleared, and it is never judged and never discarded; no shipped
// instruction discards such a note.
func Test792a22k1a_ANewSkillNoteGetsItsRunbookFields(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)
	special := curateRulesSection(t, text, curateRulesSpecialHeading, curateRulesStep1Heading)
	fresh := curateRulesSection(t, special, curateRulesNewNotesStart, curateRulesRefreshedStart)

	g.Expect(special).To(ContainSubstring("A pending runbook note that carries a `skill_hash` field is a skill's " +
		"registration offer, not a served write."))
	g.Expect(special).To(ContainSubstring("never judge it covered/near/absent, never discard it, and always keep it in " +
		"the vault."))
	g.Expect(fresh).To(ContainSubstring("Author its `situation`, `done_when`, `triggers`, and `red_flags` from the " +
		"skill's body:"))
	g.Expect(special).To(ContainSubstring("Amend the note: `engram amend --target <basename> --situation=\"<text>\""))
	g.Expect(special).To(ContainSubstring("Then clear the marker: `engram amend --target <basename> --clear-pending`"))
	g.Expect(curateRulesBefore(special, "Amend the note: `engram amend", "Then clear the marker:")).To(BeTrue())
	g.Expect(curateRulesContradictions(t, curateRulesDiscardSkillNoteScan)).To(BeEmpty())
}

// Test792a22k1b_ARefreshedSkillNoteIsReCheckedNotDiscarded reads the curate skill the repository ships: a pending
// runbook note that already has runbook fields is re-checked field by field against the skill's current body, the
// fields that no longer fit are amended, the marker is cleared, and the note is never judged covered, near or absent
// and never discarded, whatever other notes cover; no shipped instruction judges or discards such a note.
func Test792a22k1b_ARefreshedSkillNoteIsReCheckedNotDiscarded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)
	special := curateRulesSection(t, text, curateRulesSpecialHeading, curateRulesStep1Heading)
	refreshed := curateRulesSection(t, special, curateRulesRefreshedStart, "")

	g.Expect(special).To(ContainSubstring("never judge it covered/near/absent, never discard it, and always keep it in " +
		"the vault."))
	g.Expect(refreshed).To(ContainSubstring("**For refreshed notes (fields already present):** Re-check each field " +
		"against the skill's current body."))
	g.Expect(refreshed).To(ContainSubstring("Amend any that no longer fit"))
	g.Expect(refreshed).To(ContainSubstring("Then clear the marker: `engram amend --target <basename> --clear-pending`."))
	g.Expect(curateRulesContradictions(t, curateRulesJudgeSkillNoteScan)).To(BeEmpty())
	g.Expect(curateRulesContradictions(t, curateRulesDiscardSkillNoteScan)).To(BeEmpty())
}

// Test792a22k1c_CurationDoesNotTrimValidRedFlagsToFitThePreviewBudget reads the curate skill the repository ships: for
// a new and for a refreshed skill note the red flags are aimed at the 1200-byte preview budget, which the skill
// states in rendered YAML bytes, and a valid red flag is never dropped to fit; no shipped instruction trims a red flag
// to the budget or measures the budget as the sum of plain strings.
func Test792a22k1c_CurationDoesNotTrimValidRedFlagsToFitThePreviewBudget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, curateRulesSkillPath)
	special := curateRulesSection(t, text, curateRulesSpecialHeading, curateRulesStep1Heading)
	refreshed := curateRulesSection(t, special, curateRulesRefreshedStart, "")

	g.Expect(special).To(ContainSubstring("`red_flags`: the skill's key failure modes — "))
	g.Expect(refreshed).To(ContainSubstring("Amend any that no longer fit — "))
	g.Expect(strings.Count(special, curateRulesBudgetSentence)).To(Equal(curateRulesBudgetSentenceCount))
	g.Expect(strings.Count(refreshed, curateRulesBudgetSentence)).To(Equal(1))
	g.Expect(curateRulesContradictions(t, curateRulesTrimRedFlagsScan)).To(BeEmpty())
	g.Expect(curateRulesContradictions(t, curateRulesPlainSumScan)).To(BeEmpty())
}

// TestCurateRulesHits supports the absence scans of this file: a scan finds an instruction in each form (a paraphrase
// of the contradiction, an imperative, an imperative that follows a negated condition, an "is optional" form, a modal
// form and the wordings a review found missed) and finds nothing in the wording that states the rule or forbids the
// opposite.
func TestCurateRulesHits(t *testing.T) {
	t.Parallel()

	for _, scan := range curateRulesScans() {
		t.Run(scan.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			forms := append([]string{
				scan.paraphrase, scan.imperative, scan.negatedCondition, scan.optional, scan.modal,
			}, scan.variants...)
			for _, form := range forms {
				g.Expect(curateRulesHits(form, scan.pattern)).NotTo(BeEmpty(), form)
			}

			g.Expect(curateRulesHits(scan.stated, scan.pattern)).To(BeEmpty())
		})
	}
}

// TestCurateRulesHitsClauses supports the absence scans of this file: a negation excuses a match only when it stands
// in the clause before the match, or right before the last word of the match; a clause ends at a sentence end, a
// colon, a table cell border, a list item, a line break, a comma, a semicolon or a dash.
func TestCurateRulesHitsClauses(t *testing.T) {
	t.Parallel()

	pattern := curateRulesDiscardSkillNoteScan.pattern

	for name, scenario := range map[string]struct {
		content string
		want    int
	}{
		"a negated sentence is a prohibition":                  {"Never discard a skill note.", 0},
		"an instruction is a hit":                              {"Discard a skill note.", 1},
		"an imperative after a negated condition is hit":       {"If it isn't stale, discard a skill note.", 1},
		"an imperative after a negated sentence is hit":        {"Do not stall. Discard a skill note.", 1},
		"a clause after a colon is a hit":                      {"Never wait: discard a skill note.", 1},
		"a clause after a dash is a hit":                       {"Never wait — discard a skill note.", 1},
		"a table cell after a negated cell is a hit":           {"| Never wait | Discard a skill note |", 1},
		"a list item after a negated item is a hit":            {"- Never wait\n- Discard a skill note\n", 1},
		"a negation in a wrapped line is a prohibition":        {"Do NOT\ndiscard a skill note.", 0},
		"a negation inside the match does not excuse it":       {"A skill note with no runbook fields is discarded.", 1},
		"a negated condition inside the match does not excuse": {"A skill note that is not refreshed is discarded.", 1},
		"a negation right before the last word is a ban":       {"A skill note is never discarded.", 0},
		"a negated requirement at the end of a match is a hit": {"Keeping a skill note in the vault is not required.", 1},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(curateRulesHits(scenario.content, pattern)).To(HaveLen(scenario.want))
		})
	}
}

// unexported constants.
const (
	curateRulesAmendOffer     = `(?:offer\.for|amend-offer)`
	curateRulesBudgetSentence = "aim for the 1200-byte `engram query` preview budget (rendered YAML bytes); never drop " +
		"a valid red flag to fit"
	curateRulesBudgetSentenceCount = 2
	curateRulesClauseWindow        = 240
	curateRulesDifference          = `(?:differently|separately|more strictly|more leniently|stricter|looser|lighter|` +
		`heavier|weaker|easier|harder|less strictly)`
	curateRulesDropVerb       = `(?:discard|delete|drop|remove|purge|prune)`
	curateRulesFoldVerb       = `(?:fold|merge|discard|settle|use|pass|add|apply|run|combine|attach)`
	curateRulesFoldedForm     = `(?:folded|merged|discarded|settled|attached)`
	curateRulesGap10          = `(?:[^.;:|,\n]|\.\d){0,10}`
	curateRulesGap20          = `(?:[^.;:|,\n]|\.\d){0,20}`
	curateRulesGap40          = `(?:[^.;:|,\n]|\.\d){0,40}`
	curateRulesGap60          = `(?:[^.;:|,\n]|\.\d){0,60}`
	curateRulesGap80          = `(?:[^.;:|,\n]|\.\d){0,80}`
	curateRulesGapComma80     = `(?:[^.;:|\n]|\.\d){0,80}`
	curateRulesJudgment       = `(?:covered|near|absent)`
	curateRulesLearnTarget    = "`?\\b(?:engram learn|write-memory)\\b"
	curateRulesLearnVerb      = `(?:runs?|uses?|calls?|invokes?|executes?|issues?|hand\s+(?:it\s+|them\s+)?off\s+to)`
	curateRulesModal          = `(?:may|can|could|should)`
	curateRulesNewNotesStart  = "**For new notes (no runbook fields yet):**"
	curateRulesOfferWord      = `\b(?:offer|curat\w*|pending)\b`
	curateRulesOptional       = `\b(?:is|are)\s+(?:optional|not required|not needed|unnecessary)\b`
	curateRulesPlainStrings   = `(?:plain|raw|unrendered)\s+(?:strings?|text|characters)`
	curateRulesQueryCommand   = "`?\\bengram query\\b"
	curateRulesRedFlag        = `red[- ]flags?`
	curateRulesRefreshedStart = "**For refreshed notes (fields already present):**"
	curateRulesRejected       = `(?:rejected|contradicted|contradicting|wrong)`
	curateRulesScoreWord      = `(?:score|similarity|cosine)`
	curateRulesServed         = `(?:pulled[- ]down|served)`
	curateRulesSkillNoteWords = `(?:skill (?:runbook )?note|runbook note)`
	curateRulesSkillPath      = "agent-instructions/skills/curate/SKILL.md"
	curateRulesSpecialHeading = "## Special Case — Pending Skill Runbook Notes"
	curateRulesStep1Heading   = "## Step 1 — Find pending offers"
	curateRulesTrimVerb       = `(?:drop|trim|shorten|cut|remove|omit|truncate)`
	curateRulesTrimmedForm    = `(?:dropped|trimmed|shortened|cut|removed|omitted|truncated)`
)

// unexported variables.
var (
	// curateRulesClauseBreak marks a break inside a sentence that starts a new clause for the scope of a negation.
	curateRulesClauseBreak       = regexp.MustCompile(`[,;(—]`)
	curateRulesDetachedValueScan = curateRulesScan{
		name: "a free-text value after a space",
		pattern: curateRulesPattern(
			`--(?:situation|done-when|red-flag|trigger|source|subject|predicate|object|behavior|impact|action|phrase)\s+["'<]`,
			`\bspace\b`+curateRulesGap60+`\b(?:flag|value)\b`+curateRulesGap60+
				curateRulesOptional,
			`\bquote\b`+curateRulesGap60+`\btouching\b`+curateRulesGap60+curateRulesOptional,
			`\bvalue\b`+curateRulesGap60+`\b`+curateRulesModal+
				`\s+(?:be\s+(?:passed|written|given)|follow)\b`+curateRulesGap60+
				`\bafter (?:a|the) space\b`,
		),
		paraphrase:       "A free-text value may follow its flag after a space, for example --source \"a note\".",
		imperative:       "Pass the situation as --situation \"<text>\" to the amend.",
		negatedCondition: "If the value doesn't begin with a dash, pass it as --situation \"<text>\".",
		optional:         "The quote touching the equals sign is optional.",
		modal:            "A value can be passed after a space.",
		variants: []string{
			"Pass the situation as --situation <text> to the amend.",
		},
		stated: "Write a free-text value as --flag=\"value\", not --flag \"value\".",
	}
	curateRulesDiscardSkillNoteScan = curateRulesScan{
		name: "discarding a skill note",
		pattern: curateRulesPattern(
			`\b`+curateRulesSkillNoteWords+`\b`+curateRulesGap80+`\b`+curateRulesDropVerb+
				`\w*\b`,
			`\b`+curateRulesDropVerb+`\b`+curateRulesGap60+`\b`+curateRulesSkillNoteWords+
				`\b`,
			`\b(?:keep\w*|retain\w*|preserv\w*|stor\w*)\b`+curateRulesGap60+`\b`+
				curateRulesSkillNoteWords+`\b`+curateRulesGap60+curateRulesOptional,
		),
		paraphrase:       "A pending skill note that another note covers is discarded.",
		imperative:       "Discard a pending skill runbook note once its fields are authored.",
		negatedCondition: "If the note has no xid, discard a pending skill note.",
		optional:         "Keeping a skill note in the vault is optional.",
		modal:            "A skill note may be discarded once another note covers it.",
		variants: []string{
			"A skill note with no runbook fields is discarded.",
			"A skill note that is not refreshed is discarded.",
			"Keeping a skill note in the vault is not required.",
		},
		stated: "Handle a skill note separately and never discard it.",
	}
	curateRulesFoldRejectedScan = curateRulesScan{
		name: "folding a rejected offer in",
		pattern: curateRulesPattern(
			`\b`+curateRulesRejected+`\b`+curateRulesGap80+`\b`+curateRulesFoldVerb+`\b`+
				curateRulesGap60+`--into\b`,
			`\b`+curateRulesFoldVerb+`\b`+curateRulesGap60+`\b`+curateRulesRejected+`\b`+
				curateRulesGap60+`--into\b`,
			`\b`+curateRulesFoldVerb+`\b`+curateRulesGap60+`--into\b`+curateRulesGap60+`\b`+
				curateRulesRejected+`\b`,
			`\b`+curateRulesRejected+`\b`+curateRulesGap80+`\b(?:`+curateRulesModal+
				` be|is|are|gets?)\s+`+curateRulesFoldedForm+`\b`+curateRulesGap60+`--into\b`,
			`(?:\bfold\w*|--into)\b`+curateRulesGap80+curateRulesOptional,
		),
		paraphrase:       "A rejected offer is settled with --discard --into the note that contradicts it.",
		imperative:       "Fold a contradicted offer --into the note that contradicts it.",
		negatedCondition: "If the offer isn't covered, discard a rejected offer with --into the contradicting note.",
		optional:         "Folding a rejected offer --into the note that contradicts it is optional.",
		modal:            "A rejected offer may be folded --into the note that contradicts it.",
		variants: []string{
			"Use --discard --into for a rejected offer.",
		},
		stated: "Never fold a rejected offer with --discard --into.",
	}
	curateRulesJudgeSkillNoteScan = curateRulesScan{
		name: "judging a skill note against other notes",
		pattern: curateRulesPattern(
			`\b(?:judge|classify|treat|mark|rate|decide)\b`+curateRulesGap40+`\b`+
				curateRulesSkillNoteWords+`\b`+curateRulesGap40+`\b`+curateRulesJudgment+`\b`,
			`\b`+curateRulesSkillNoteWords+`\b`+curateRulesGap60+`\b(?:is|are|`+
				curateRulesModal+` be|gets?)\s+(?:judged|treated as|classified as|marked|rated)\s+`+
				curateRulesJudgment+`\b`,
			`\b(?:fold|discard)\b`+curateRulesGap60+`\b`+curateRulesSkillNoteWords+`\b`+
				curateRulesGap80+`\bcovers?\b`,
			`\b(?:judg\w*|re-?check\w*|review\w*)\b`+curateRulesGap60+`\b`+
				curateRulesSkillNoteWords+`\b`+curateRulesGap60+curateRulesOptional,
		),
		paraphrase:       "A refreshed skill note is judged covered when another note covers similar ground.",
		imperative:       "Judge a refreshed skill note covered when another note states the same ground.",
		negatedCondition: "If the fields aren't stale, fold a refreshed skill note away when another note covers it.",
		optional:         "Re-checking a refreshed skill note is optional.",
		modal:            "A skill note may be judged covered.",
		variants: []string{
			"A skill note is classified as near when another note overlaps.",
		},
		stated: "Never judge a skill note covered, near or absent.",
	}
	curateRulesLearnScan = curateRulesScan{
		name: "curation composing a new learn",
		pattern: curateRulesPattern(
			`\b`+curateRulesLearnVerb+`\b`+curateRulesGap40+curateRulesLearnTarget+
				curateRulesGap80+curateRulesOfferWord,
			curateRulesOfferWord+curateRulesGap80+`\b`+curateRulesLearnVerb+`\b`+
				curateRulesGap40+curateRulesLearnTarget,
			`\bhand\b`+curateRulesGap60+`\b(?:offer|curat\w*|pending|absent)\b`+curateRulesGap60+
				`\bto\b`+curateRulesGap10+curateRulesLearnTarget,
			`\b(?:us\w*|run\w*|call\w*|invok\w*|hand\b)`+curateRulesGap60+curateRulesLearnTarget+
				curateRulesGap80+curateRulesOptional,
			curateRulesLearnTarget+curateRulesGap60+`\b`+curateRulesModal+
				` be\s+(?:used|run|called|invoked|handed)\b`+curateRulesGap80+
				`\b(?:offer|curat\w*|pending|absent)\b`,
		),
		paraphrase:       "Curation of an absent offer uses `engram learn` to write the note.",
		imperative:       "Run `engram learn` to write the absent offer as a new note.",
		negatedCondition: "If the offer isn't covered, hand it off to `write-memory` for the pending offer.",
		optional:         "Using `engram learn` for an absent offer is optional.",
		modal:            "`write-memory` can be used for an absent offer.",
		variants: []string{
			"Hand the absent offer to `write-memory`.",
		},
		stated: "Never hand off to `write-memory`: an offer is already a file.",
	}
	// curateRulesNegation marks a clause that forbids or excludes what it names.
	curateRulesNegation     = regexp.MustCompile(`(?i)\b(?:not|never|neither|nor|without|skip|no)\b|n't\b`)
	curateRulesPlainSumScan = curateRulesScan{
		name: "measuring the budget as the sum of plain strings",
		pattern: curateRulesPattern(
			`\b(?:measure|count|sum|add up|total|report|compute|calculate|tally)\b`+curateRulesGap60+
				`\b`+curateRulesPlainStrings+`\b`,
			`\b(?:measure|count|report|compute|calculate|tally|express)\b`+curateRulesGap60+
				`\b(?:length|size|total|sum)\b`+curateRulesGap60+`\b`+curateRulesRedFlag+
				`\s+(?:strings?|text)\b`,
			`\b(?:measur\w*|count\w*|render\w*)\b`+curateRulesGap60+`\bbytes\b`+curateRulesGap60+
				curateRulesOptional,
			`\bbudget\b`+curateRulesGap60+`\b`+curateRulesModal+
				`\s+be\s+(?:measured|counted|summed|totalled|totaled|computed|calculated)\b`+
				curateRulesGap80+`\b`+curateRulesPlainStrings+`\b`,
		),
		paraphrase:       "The budget figure is the sum of the plain strings of the red flags.",
		imperative:       "Measure the budget as the sum of the plain strings.",
		negatedCondition: "If the note isn't rendered yet, count the plain strings against the budget.",
		optional:         "Measuring the budget in rendered YAML bytes is optional.",
		modal:            "The budget can be measured as the sum of the plain strings.",
		variants: []string{
			"Report the budget as the total length of the red flag strings.",
		},
		stated: "Measure the budget in rendered YAML bytes, not by summing the plain strings.",
	}
	curateRulesQueryFindScan = curateRulesScan{
		name: "finding pending offers with engram query",
		pattern: curateRulesPattern(
			`\b(?:use|run|call|try)\b`+curateRulesGap40+`\bengram query\b`+curateRulesGap60+
				`\b(?:find|list|locate|discover|see|get)\b`+curateRulesGap40+`\bpending\b`,
			`\bengram query\b`+curateRulesGap40+`\b(?:lists|returns|shows|finds|surfaces)\b`+
				curateRulesGap40+`\bpending (?:offers|notes)\b`,
			`\bpending (?:offers|notes)\b`+curateRulesGap40+`\b(?:(?:are|is)|`+curateRulesModal+
				` be)\s+(?:listed|found|returned|shown|surfaced)\s+(?:by|in|with|using|via)\b`+
				curateRulesGap40+`\bengram query\b`,
			`\b(?:find|list|locate|discover|get|see)\b`+curateRulesGap40+`\bpending\b`+
				curateRulesGap60+`\b(?:with|using|via|through|by)\b`+curateRulesGap10+
				curateRulesQueryCommand,
			`\b(?:scan\w*|read\w*|search\w*|look\w*)\b`+curateRulesGap60+`\bvault\b`+
				curateRulesGap60+curateRulesOptional,
		),
		paraphrase: "Pending offers are listed by `engram query`.",
		imperative: "Use `engram query` to find the pending offers.",
		negatedCondition: "If the vault is large and the scan isn't quick, use `engram query` to find the pending " +
			"offers.",
		optional: "Scanning the vault for pending offers is optional.",
		modal:    "Pending offers can be found with `engram query`.",
		variants: []string{
			"Find pending offers with `engram query`.",
		},
		stated: "Do not use `engram query` to find pending offers.",
	}
	curateRulesQueryFirstScan = curateRulesScan{
		name: "querying first for an amend-offer",
		pattern: curateRulesPattern(
			`\b`+curateRulesAmendOffer+`\b`+curateRulesGap80+`\bquery\b`+curateRulesGap40+
				`\bfirst\b`,
			`\b`+curateRulesAmendOffer+`\b`+curateRulesGapComma80+`\bquery\b`+curateRulesGap40+
				`\bfirst\b`,
			`\b(?:run|use)\b`+curateRulesGap40+`\bquery\b`+curateRulesGap40+`\bfirst\b`+
				curateRulesGap80+`\b`+curateRulesAmendOffer+`\b`,
			`\bquery\s+first\b`+curateRulesGap80+`\b`+curateRulesAmendOffer+`\b`,
			`\b`+curateRulesAmendOffer+`\b`+curateRulesGap80+`\b`+curateRulesModal+
				` be (?:judged|reviewed|assessed)\b`+curateRulesGap60+`\b(?:by|with|using|via)\b`+
				curateRulesGap20+`\bquery\b`,
			`\bjudg\w*\b`+curateRulesGap60+`\b`+curateRulesAmendOffer+`\b`+curateRulesGap60+
				curateRulesOptional,
		),
		paraphrase:       "An amend-offer is judged by running engram query first.",
		imperative:       "Run `engram query` first for the offer.for offer against the note.",
		negatedCondition: "If the target note isn't found, run `engram query` first for an offer.for offer.",
		optional:         "Judging the offer.for note first is optional.",
		modal:            "An offer.for offer may be judged with a query first.",
		variants: []string{
			"For an offer.for offer, query first.",
		},
		stated: "Do not run `engram query` first for an amend-offer.",
	}
	// curateRulesRequirementTail marks a match that ends in a negated requirement, which is an optional form.
	curateRulesRequirementTail = regexp.MustCompile(`(?i)\bnot\s+(?:required|needed|necessary)$`)
	curateRulesScoreScan       = curateRulesScan{
		name: "deciding covered, near or absent by a score",
		pattern: curateRulesPattern(
			`\b(?:judge|decide|classify|call|mark|treat|count|determine|settle)\w*\b`+curateRulesGap80+
				`\b(?:covered|near|absent|duplicate)\b`+curateRulesGap80+`\b`+curateRulesScoreWord+
				`\b`,
			`\b`+curateRulesScoreWord+`\b`+curateRulesGap60+
				`\b(?:above|over|exceeds?|at least|high)\b`+curateRulesGap60+`\bcovered\b`,
			`\b(?:high|strong|large)\s+(?:(?:cosine|similarity)\s+)*`+curateRulesScoreWord+`\b`+
				curateRulesGap40+`\b(?:means?|shows?|proves?|is|makes?)\b`+curateRulesGap40+
				`\bcovered\b`,
			`\b`+curateRulesScoreWord+`\b`+curateRulesGap60+`\b`+curateRulesModal+
				`\s+(?:be used to\s+)?(?:decide|determine|settle|judge|classify)\b`+curateRulesGap60+
				`\b(?:covered|near|absent|duplicate)\b`,
			`\b(?:covered|near|absent)\b`+curateRulesGap80+`\b`+curateRulesModal+
				` be (?:decided|judged|determined|settled|classified)\b`+curateRulesGap40+`\bby\b`+
				curateRulesGap40+`\b`+curateRulesScoreWord+`\b`,
			`\b(?:read\w*|judg\w*)\b`+curateRulesGap40+`\bcontent\b`+curateRulesGap60+
				curateRulesOptional,
		),
		paraphrase:       "A high similarity score means the offer is covered.",
		imperative:       "Judge the offer covered when the similarity score is above the threshold.",
		negatedCondition: "If the content isn't clear, judge the offer covered when the cosine score is high.",
		optional:         "Reading the content is optional when the cosine score is high.",
		modal:            "Covered may be decided by a similarity score.",
		variants: []string{
			"If the similarity is above 0.85 the offer is covered.",
		},
		stated: "Judge content, never a cosine/similarity score alone.",
	}
	// curateRulesTailNegation marks a match that ends in a negation that stands right before its last word.
	curateRulesTailNegation = regexp.MustCompile(`(?i)(?:\b(?:not|never|neither|nor|without|skip|no)\b|n't)` +
		`\s+(?:\w+\s+)?\w+$`)
	curateRulesTrimRedFlagsScan = curateRulesScan{
		name: "trimming red flags to fit the budget",
		pattern: curateRulesPattern(
			`\b`+curateRulesTrimVerb+`\b`+curateRulesGap60+`\b`+curateRulesRedFlag+`\b`+
				curateRulesGap60+`\b(?:fit|budget|1200)\b`,
			`\b(?:to fit|within|under)\b`+curateRulesGap40+`\b(?:budget|1200)\b`+curateRulesGap60+
				`\b`+curateRulesTrimVerb+`\b`+curateRulesGap40+`\b`+curateRulesRedFlag+`\b`,
			`\b`+curateRulesRedFlag+`\b`+curateRulesGap60+`\b(?:is|are|`+curateRulesModal+
				` be)\s+`+curateRulesTrimmedForm+`\b`,
			`\b(?:keep\w*|retain\w*|preserv\w*|inclu\w*)\b`+curateRulesGap60+`\b`+
				curateRulesRedFlag+`\b`+curateRulesGap80+curateRulesOptional,
		),
		paraphrase:       "Red flags beyond the 1200-byte budget are dropped.",
		imperative:       "Drop the least important red flags to fit the 1200-byte budget.",
		negatedCondition: "If the red flags aren't all valid, drop a valid red flag to fit the budget.",
		optional:         "Keeping every valid red flag is optional when the preview budget is tight.",
		modal:            "A valid red flag may be dropped to fit the budget.",
		stated:           "Never drop a valid red flag to fit the 1200-byte budget.",
	}
	curateRulesUnevenScan = curateRulesScan{
		name: "judging a served and a pulled-down offer differently",
		pattern: curateRulesPattern(
			`\b`+curateRulesServed+`\b`+curateRulesGap80+
				`\b(?:judge|judged|review|reviewed|treat|treated|assess|assessed)\b`+curateRulesGap80+
				`\b`+curateRulesDifference+`\b`,
			`\b(?:judge|review|treat|assess)\b`+curateRulesGap80+`\b`+curateRulesServed+`\b`+
				curateRulesGap80+`\b`+curateRulesDifference+`\b`,
			`\b`+curateRulesServed+`\b`+curateRulesGap80+`\b`+curateRulesDifference+`\b`,
			`\b`+curateRulesDifference+`\b`+curateRulesGap60+`\b(?:than|from)\b`+
				curateRulesGap40+`\b`+curateRulesServed+`\b`,
			`\b(?:judg\w*|review\w*|treat\w*)\b`+curateRulesGap80+`\b`+curateRulesServed+`\b`+
				curateRulesGap80+curateRulesOptional,
		),
		paraphrase:       "A pulled-down offer is judged differently from a served offer.",
		imperative:       "Judge a pulled-down offer more strictly than a served one.",
		negatedCondition: "If the offer isn't served, judge a pulled-down offer more leniently.",
		optional:         "Judging a pulled-down offer like a served one is optional.",
		modal:            "A pulled-down offer may be judged more leniently than a served one.",
		variants: []string{
			"A served offer gets a lighter review than a pulled-down one.",
		},
		stated: "Judge a pulled-down offer and a served offer alike, not differently.",
	}
)

// curateRulesScan is one absence scan: a pattern for an instruction that contradicts a rule, a contradicting
// instruction in each form to insert (a paraphrase, an imperative, an imperative that follows a negated condition, an
// "is optional" form and a modal form), the wordings a review found missed, and the shipped wording that must pass.
type curateRulesScan struct {
	name             string
	pattern          *regexp.Regexp
	paraphrase       string
	imperative       string
	negatedCondition string
	optional         string
	modal            string
	variants         []string
	stated           string
}

// curateRulesBefore reports whether both fragments occur in the text and the first occurs before the second.
func curateRulesBefore(text, first, second string) bool {
	firstAt := strings.Index(text, first)
	secondAt := strings.Index(text, second)

	return firstAt >= 0 && secondAt >= 0 && firstAt < secondAt
}

// curateRulesContradictions returns, by path, the instructions of every shipped SKILL.md, references file and guidance
// document that the scan finds. It fails the test when the scan did not cover the curate skill, so an empty result is
// not vacuous.
func curateRulesContradictions(t *testing.T, scan curateRulesScan) map[string][]string {
	t.Helper()

	found := make(map[string][]string)
	scannedCurate := false

	for _, path := range aibInstructionPaths(t) {
		content, err := os.ReadFile(filepath.Join(repositoryRoot, path))
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)

			return nil
		}

		if path == curateRulesSkillPath {
			scannedCurate = true
		}

		hits := curateRulesHits(string(content), scan.pattern)
		if len(hits) > 0 {
			found[path] = hits
		}
	}

	if !scannedCurate {
		t.Fatalf("the instruction scan did not cover %s", curateRulesSkillPath)
	}

	return found
}

// curateRulesHits returns every match of the pattern in the content of a shipped file, except a match that a negation
// excuses (a prohibition is not an instruction): a negation in the clause before the match, as in learnPart1Hits, or a
// negation right before the last word of the match (see curateRulesNegatesTail). The clause runs from the last clause
// break before the start of the match to the start of the match: a sentence end, a colon, a table cell border, a list
// item, a line break end it as in learnPart1Blocks (blockquote markers stripped), and so do a comma, a semicolon and a
// dash, so that a negated condition before the instruction does not excuse it.
func curateRulesHits(content string, pattern *regexp.Regexp) []string {
	text := learnPart1Blocks(content)
	hits := make([]string, 0)

	for _, location := range pattern.FindAllStringIndex(text, -1) {
		lead := text[max(0, location[0]-curateRulesClauseWindow):location[0]]
		lead = lead[learnPart1Boundary(lead):]

		breaks := curateRulesClauseBreak.FindAllStringIndex(lead, -1)
		if len(breaks) > 0 {
			lead = lead[breaks[len(breaks)-1][1]:]
		}

		matched := text[location[0]:location[1]]
		if curateRulesNegation.MatchString(lead) || curateRulesNegatesTail(matched) {
			continue
		}

		hits = append(hits, matched)
	}

	return hits
}

// curateRulesNegatesTail reports whether the match ends in a negation that stands right before its last word ("is
// never discarded"), which is a prohibition. A negation further back ("with no runbook fields is discarded") is not,
// and neither is a negated requirement ("is not required"), which is an optional form.
func curateRulesNegatesTail(matched string) bool {
	return curateRulesTailNegation.MatchString(matched) && !curateRulesRequirementTail.MatchString(matched)
}

// curateRulesPattern compiles the alternatives into one case-insensitive pattern.
func curateRulesPattern(alternatives ...string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)` + strings.Join(alternatives, "|"))
}

// curateRulesScans returns every absence scan of this file.
func curateRulesScans() []curateRulesScan {
	return []curateRulesScan{
		curateRulesDetachedValueScan, curateRulesDiscardSkillNoteScan, curateRulesFoldRejectedScan,
		curateRulesJudgeSkillNoteScan, curateRulesLearnScan, curateRulesPlainSumScan, curateRulesQueryFindScan,
		curateRulesQueryFirstScan, curateRulesScoreScan, curateRulesTrimRedFlagsScan, curateRulesUnevenScan,
	}
}

// curateRulesSection returns the text from the start fragment up to the end fragment, or to the end of the text when
// the end is empty. It fails the test when a fragment is missing or the end does not follow the start.
func curateRulesSection(t *testing.T, text, start, end string) string {
	t.Helper()

	startAt := strings.Index(text, start)
	if startAt < 0 {
		t.Fatalf("section start %q not found", start)

		return ""
	}

	section := text[startAt:]
	if end == "" {
		return section
	}

	before, _, found := strings.Cut(section, end)
	if !found {
		t.Fatalf("section end %q not found after %q", end, start)

		return ""
	}

	return before
}
