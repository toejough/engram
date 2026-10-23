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

// Text-level: each test below reads the write-memory skill the repository ships and asserts the sentences that state
// the rule of its scenario. It verifies that the shipped rule is stated, not that an agent follows it. The scenarios
// whose Then has a prohibition half also scan every shipped SKILL.md, references file and guidance document for an
// instruction that contradicts it.

// Test792a13a1a_ACompleteHandoffIsExecuted reads the write-memory skill the repository ships: the worker's job is to
// compose, execute, verify and report, and the feedback command is composed from the handed-off fields. That the
// command is run and the written note path is reported is asserted by Test792a13n1a_SuccessIsReportedInOneLine.
func Test792a13a1a_ACompleteHandoffIsExecuted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	feedback := writeMemoryPart1Between(t, text, writeMemoryPart1FeedbackStart, writeMemoryPart1FactStart)

	g.Expect(text).To(ContainSubstring("Your job is the write itself: compose, execute, verify, report."))
	g.Expect(text).To(ContainSubstring("**kind** — `fact`, `feedback`, `qa`, or `runbook`"))
	g.Expect(text).To(ContainSubstring("**content fields** — by kind, per the blocks below"))
	g.Expect(feedback).To(ContainSubstring("engram learn feedback"))
}

// Test792a13a1b_TheParentsContentIsNotReworded reads the write-memory skill the repository ships: the handed-off
// content is passed as handed off, neither reworded nor merged, and no shipped instruction tells a skill to reword,
// merge, decline or skip handed-off content. That the write is not declined is the rule "do not decide WHETHER to
// write" beside it, asserted here too (decision 861d402).
func Test792a13a1b_TheParentsContentIsNotReworded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)

	g.Expect(text).To(ContainSubstring("do not decide WHETHER to write."))
	g.Expect(text).To(ContainSubstring("Pass the handed-off content as handed off, neither reworded nor merged."))
	g.Expect(writeMemoryPart1Instructions(t, writeMemoryPart1AlterScan)).To(BeEmpty())
}

// Test792a13a1c_NoHandoffNoFire reads the write-memory skill the repository ships: the skill requires a handoff and
// does not fire on its own judgment that something is worth remembering, and no shipped instruction tells a skill to
// fire without a handoff.
func Test792a13a1c_NoHandoffNoFire(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)

	g.Expect(text).To(ContainSubstring("Executes a vault write handed off by another skill (recall, learn)"))
	g.Expect(text).To(ContainSubstring("Requires a handoff — do not fire on your own judgment that something is worth " +
		"remembering."))
	g.Expect(text).To(ContainSubstring("You were invoked by a parent skill that already made the judgment"))
	g.Expect(writeMemoryPart1Instructions(t, writeMemoryPart1FireScan)).To(BeEmpty())
}

// Test792a13b1a_AWriteSiteNamesTheWorker reads the learn and recall skills the repository ships: the learn skill
// invokes the write-memory skill at six write sites, one per kind of lesson, and the recall skill at three. The
// sentences of each site are asserted by the learn and recall units and the instruction bundle test, the report of the
// note path by Test792a13n1a_SuccessIsReportedInOneLine, and recall's red flag by Test792b32c4a1; only the counts and
// the kind of each learn handoff are asserted here.
func Test792a13b1a_AWriteSiteNamesTheWorker(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	learn := shippedText(t, writeMemoryPart1LearnPath)
	recall := shippedText(t, writeMemoryPart1RecallPath)

	g.Expect(strings.Count(learn, "invoke the **write-memory** skill")).To(Equal(writeMemoryPart1LearnWriteSites))
	g.Expect(learn).To(ContainSubstring("**REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff — " +
		"kind=feedback,"))
	g.Expect(learn).To(ContainSubstring("**REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff — " +
		"kind=fact,"))
	g.Expect(learn).To(ContainSubstring("For a **runbook**, **REQUIRED SUB-SKILL:** invoke the **write-memory** skill " +
		"with this handoff — kind=runbook,"))
	g.Expect(learn).To(ContainSubstring("**invoke the **write-memory** skill** with this handoff — kind=qa,"))
	g.Expect(strings.Count(recall, "the **write-memory** skill")).To(Equal(writeMemoryPart1RecallWriteSites))
}

// Test792a13b1b_ASessionWithoutTheSkillFollowsTheRunbookNote reads the shim and the write-memory skill the repository
// ships: a runbook reached over a wikilink is followed through the shim's follow-frame, and the contract the note
// carries is the skill's handoff contract, whose heading the registered note mirrors in its body. The frame's own
// sentences are asserted by Test792a19q1a_ALinkedRunbookIsFetchedAndFollowed; only the rule's opening clause is
// asserted here (decision 871d2).
func Test792a13b1b_ASessionWithoutTheSkillFollowsTheRunbookNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	shim := shippedText(t, ag3ShimPath)
	skill := writeMemoryPart1Text(t)

	g.Expect(shim).To(ContainSubstring("6. **A wikilinked runbook (`[[basename]]`)"))
	g.Expect(skill).To(ContainSubstring("## The handoff contract The parent provides:"))
}

// Test792a13c1a_AMissingFieldIsRequested reads the write-memory skill the repository ships: a missing required field
// is asked for from the in-session parent context and no command is run until it is supplied, and no shipped
// instruction tells a skill to run the command with the field missing (decision 861d404).
func Test792a13c1a_AMissingFieldIsRequested(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)

	g.Expect(text).To(ContainSubstring("If a required field is missing, ask for it from the in-session parent context " +
		"and run no command until it is supplied"))
	g.Expect(writeMemoryPart1Instructions(t, writeMemoryPart1RunEarlyScan)).To(BeEmpty())
}

// Test792a13c1b_ContentIsNeverInvented reads the write-memory skill the repository ships: the worker does not invent
// content on the parent's behalf, and no shipped instruction tells a skill to make up a missing field. The sentence
// also says a missing field is asked for, which is scenario 792a13c1a, asserted by
// Test792a13c1a_AMissingFieldIsRequested.
func Test792a13c1b_ContentIsNeverInvented(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)

	g.Expect(text).To(ContainSubstring("do not invent content on the parent's behalf."))
	g.Expect(writeMemoryPart1Instructions(t, writeMemoryPart1InventScan)).To(BeEmpty())
}

// Test792a13d1a_FeedbackComposition reads the write-memory skill the repository ships: the feedback command is
// engram learn feedback with the slug, the position, the source, the situation as the retrieval-shaped phrase, the
// behavior, the impact and the action, in that order. The default position is scenario 792a13g1a.
func Test792a13d1a_FeedbackComposition(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	feedback := writeMemoryPart1Between(t, text, writeMemoryPart1FeedbackStart, writeMemoryPart1FactStart)

	g.Expect(feedback).To(ContainSubstring("engram learn feedback --slug <kebab-slug>"))
	g.Expect(feedback).To(ContainSubstring("--position <top|continuation|sibling>"))
	g.Expect(feedback).To(ContainSubstring("--source=\"<source>\""))
	g.Expect(feedback).To(ContainSubstring("--situation=\"<retrieval-shaped phrase: when does this apply>\""))
	g.Expect(feedback).To(ContainSubstring("--behavior=\"<what was done>\""))
	g.Expect(feedback).To(ContainSubstring("--impact=\"<why it was wrong/costly>\""))
	g.Expect(feedback).To(ContainSubstring("--action=\"<what to do instead>\""))
	g.Expect(writeMemoryPart1InOrder(feedback, "engram learn feedback", "--slug", "--position", "--source=",
		"--situation=", "--behavior=", "--impact=", "--action=")).To(BeTrue())
}

// Test792a13d1b_FactComposition reads the write-memory skill the repository ships: the fact command is engram learn
// fact with the slug, the position, the source, the situation as the retrieval-shaped phrase, the subject, the
// predicate and the object, in that order. The default position is scenario 792a13g1a.
func Test792a13d1b_FactComposition(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	fact := writeMemoryPart1Between(t, text, writeMemoryPart1FactStart, writeMemoryPart1RunbookStart)

	g.Expect(fact).To(ContainSubstring("engram learn fact --slug <kebab-slug>"))
	g.Expect(fact).To(ContainSubstring("--position <top|continuation|sibling>"))
	g.Expect(fact).To(ContainSubstring("--source=\"<source>\""))
	g.Expect(fact).To(ContainSubstring("--situation=\"<retrieval-shaped phrase: when does this apply>\""))
	g.Expect(fact).To(ContainSubstring("--subject=\"<the thing>\""))
	g.Expect(fact).To(ContainSubstring("--predicate=\"<requires / must use / is>\""))
	g.Expect(fact).To(ContainSubstring("--object=\"<the standard or value>\""))
	g.Expect(writeMemoryPart1InOrder(fact, "engram learn fact", "--slug", "--position", "--source=", "--situation=",
		"--subject=", "--predicate=", "--object=")).To(BeTrue())
}

// Test792a13e1a_ARunbookHandoffIsExecuted reads the write-memory skill the repository ships: the runbook command is
// engram learn runbook with the slug, the position, the source, the situation, the done-when and the body, and one
// --red-flag per handed-off red flag in order. The report of the written note path is scenario 792a13n1a.
func Test792a13e1a_ARunbookHandoffIsExecuted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	runbook := writeMemoryPart1Between(t, text, writeMemoryPart1RunbookStart, writeMemoryPart1QAStart)

	g.Expect(runbook).To(ContainSubstring("engram learn runbook --slug <kebab-slug>"))
	g.Expect(runbook).To(ContainSubstring("--position <top|continuation|sibling>"))
	g.Expect(runbook).To(ContainSubstring("--source=\"<source>\""))
	g.Expect(runbook).To(ContainSubstring("--situation=\"<retrieval-shaped phrase: when should this runbook be used>\""))
	g.Expect(runbook).To(ContainSubstring("--done-when=\"<what should be true when the procedure is complete>\""))
	g.Expect(runbook).To(ContainSubstring("--body=\"<numbered steps, may [[wikilink]] fact/feedback notes to consider>\""))
	g.Expect(runbook).To(ContainSubstring("[--red-flag=\"<task-specific failure mode>\" ...]"))
	g.Expect(text).To(ContainSubstring("One `--red-flag` per handed-off `red_flags` entry, in order;"))
	g.Expect(writeMemoryPart1InOrder(runbook, "engram learn runbook", "--slug", "--position", "--source=",
		"--situation=", "--done-when=", "--body=", "--red-flag=")).To(BeTrue())
}

// Test792a13e1b_NoRedFlagsNoFlag reads the write-memory skill the repository ships: the --red-flag is omitted entirely
// when the handoff carries no red flags, with no error and no empty flag, and no shipped instruction tells a skill to
// pass an empty red flag.
func Test792a13e1b_NoRedFlagsNoFlag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)

	g.Expect(text).To(ContainSubstring("omit entirely when the parent's handoff carries no red_flags " +
		"(no error, no empty flag)."))
	g.Expect(writeMemoryPart1Instructions(t, writeMemoryPart1EmptyFlagScan)).To(BeEmpty())
}

// Test792a13f1a_QAComposition reads the write-memory skill the repository ships: the qa command is engram learn qa
// with the slug, the question, the answer copied rather than re-derived, the contributors, the certainty and the
// source, one --contributors per handed-off basename, and no shipped instruction tells a skill to rewrite the answer.
func Test792a13f1a_QAComposition(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	qaBlock := writeMemoryPart1Between(t, text, writeMemoryPart1QAStart, writeMemoryPart1AppendStart)

	g.Expect(qaBlock).To(ContainSubstring("engram learn qa"))
	g.Expect(qaBlock).To(ContainSubstring("--slug=\"<kebab summary of the question>\""))
	g.Expect(qaBlock).To(ContainSubstring("--question=\"<verbatim question>\""))
	g.Expect(qaBlock).To(ContainSubstring("--answer=\"<the answer body, copied — no re-derive>\""))
	g.Expect(qaBlock).To(ContainSubstring("--contributors=\"<full-basename>\""))
	g.Expect(qaBlock).To(ContainSubstring("--certainty=\"<high|medium|low>\""))
	g.Expect(qaBlock).To(ContainSubstring("--source=\"<source>\""))
	g.Expect(text).To(ContainSubstring("for qa: one `--contributors <full-basename>` per basename the parent provided"))
	g.Expect(writeMemoryPart1InOrder(qaBlock, "engram learn qa", "--slug=", "--question=", "--answer=", "--contributors=",
		"--certainty=", "--source=")).To(BeTrue())
	g.Expect(writeMemoryPart1Instructions(t, writeMemoryPart1RewriteAnswerScan)).To(BeEmpty())
}

// Test792a13g1a_NoPlacementMeansTop reads the write-memory skill the repository ships: a fact handoff with no position
// or target gets --position top and no --target. That no shipped instruction passes a target with top is asserted by
// Test792a13g1c_TopNeverCarriesATarget.
func Test792a13g1a_NoPlacementMeansTop(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	fact := writeMemoryPart1Between(t, text, writeMemoryPart1FactStart, writeMemoryPart1RunbookStart)

	g.Expect(fact).To(ContainSubstring("[--target <id>]"))
	g.Expect(text).To(ContainSubstring("Omitted → `--position top` (today's default), fact/feedback/runbook only."))
	g.Expect(text).To(ContainSubstring("`--position` defaults to `top` and `--target` is omitted whenever the parent's " +
		"handoff doesn't include a position/target"))
}

// Test792a13g1b_AContinuationCarriesItsTarget reads the write-memory skill the repository ships: a handed-off position
// of continuation is passed as --position with an optional --target <id> in the command, the target note ID is required
// with it, and --target <id> is included when the position is continuation or sibling.
func Test792a13g1b_AContinuationCarriesItsTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	feedback := writeMemoryPart1Between(t, text, writeMemoryPart1FeedbackStart, writeMemoryPart1FactStart)

	g.Expect(feedback).To(ContainSubstring("[--target <id>]"))
	g.Expect(text).To(ContainSubstring("optional **position** / **target** — Luhmann placement (`top` (default), " +
		"`continuation`, or `sibling`) plus the target note ID (required when position is not `top`)"))
	g.Expect(text).To(ContainSubstring("Include `--target <id>` only when position is `continuation` or `sibling`;"))
}

// Test792a13g1c_TopNeverCarriesATarget reads the write-memory skill the repository ships: --target is never emitted
// alongside --position top, and no shipped instruction tells a skill to pass a target that is not tied to a
// continuation or a sibling.
func Test792a13g1c_TopNeverCarriesATarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)

	g.Expect(text).To(ContainSubstring("never emit `--target` alongside `--position top`."))
	g.Expect(writeMemoryPart1Instructions(t, writeMemoryPart1TargetScan)).To(BeEmpty())
}

// TestWriteMemoryPart1Hits shows the scanner of the absence checks of this file finds an instruction that contradicts
// each rule (as a paraphrase, as an imperative, under a negated condition, as an imperative after a negated aside, as
// an "is optional" statement, in a table row after a negated row and, for the target rule, with a bare "only" that is
// not the shipped phrase) and passes the shipped wording that states the rule or forbids the opposite.
func TestWriteMemoryPart1Hits(t *testing.T) {
	t.Parallel()

	for _, scan := range writeMemoryPart1Scans() {
		t.Run(scan.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(writeMemoryPart1Hits(scan.paraphrase, scan)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart1Hits(scan.imperative, scan)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart1Hits(scan.conditional, scan)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart1Hits(scan.aside, scan)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart1Hits(scan.optional, scan)).NotTo(BeEmpty())

			for _, abuse := range scan.exemptAbuse {
				g.Expect(writeMemoryPart1Hits(abuse, scan)).NotTo(BeEmpty())
			}

			for _, variant := range scan.variants {
				g.Expect(writeMemoryPart1Hits(variant, scan)).NotTo(BeEmpty())
			}

			g.Expect(writeMemoryPart1Hits("| Rule | Never improvise |\n| Next | "+scan.imperative+" |\n", scan)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart1Hits(scan.stated, scan)).To(BeEmpty())
		})
	}
}

// unexported constants.
const (
	writeMemoryPart1Allowed = `[^.\n|]{0,30}\b(?:is|are)\s+` +
		`(?:optional|fine|allowed|acceptable|ok(?:ay)?|permitted)\b`
	writeMemoryPart1AppendStart      = "Append to any kind:"
	writeMemoryPart1ContextWindow    = 160
	writeMemoryPart1FactStart        = "kind=fact:"
	writeMemoryPart1FeedbackStart    = "kind=feedback:"
	writeMemoryPart1LearnPath        = "agent-instructions/skills/learn/SKILL.md"
	writeMemoryPart1LearnWriteSites  = 6
	writeMemoryPart1QAStart          = "kind=qa:"
	writeMemoryPart1RecallPath       = "agent-instructions/skills/recall/SKILL.md"
	writeMemoryPart1RecallWriteSites = 3
	writeMemoryPart1RunbookStart     = "kind=runbook:"
	writeMemoryPart1SkillPath        = "agent-instructions/skills/write-memory/SKILL.md"
	writeMemoryPart1TrailWindow      = 60
)

// unexported variables.
var (
	writeMemoryPart1AlterScan = writeMemoryPart1Scan{
		name: "rewording, merging or declining the handoff",
		pattern: regexp.MustCompile(
			`(?i)\b(?:reword\w*|rephras\w*|paraphras\w*|rewr(?:ite|ites|iting|ote)|polish\w*|tidy\w*|condens\w*` +
				`|shorten\w*|merg(?:e|es|ed|ing)|combin(?:e|es|ed|ing)|consolidat\w*|declin\w*|refus\w*|skip\w*` +
				`|discard\w*)\b[^.\n]{0,40}\bhand(?:offs?|ed[- ]off)\b`),
		// The route skill tells its caller to rewrite the handoff of a failed dispatch, a brief and not handed-off content.
		exempt:      regexp.MustCompile(`(?i)\brewrite\s+the\s+handoff\s*\(sharper\b`),
		paraphrase:  "Polish a clumsy handed-off note before you compose the command.",
		imperative:  "Reword the handed-off content to read better.",
		conditional: "If two handoffs overlap, merge the handed-off notes into one.",
		aside:       "No change is needed, reword the handed-off content anyway.",
		optional:    "Rewording the handed-off content is optional.",
		exemptAbuse: []string{
			"Rewrite the handoff content to read better.",
			"Rewrite the handoff (sharper files), then rewrite the handoff to read better.",
		},
		variants: []string{
			"Reword the handoff to read better before you compose.",
			"Merge two overlapping handoffs into one note.",
		},
		stated: "Pass the handed-off content as handed off, neither reworded nor merged.",
	}
	writeMemoryPart1ClauseSplit = regexp.MustCompile(`[,;]\s+`)
	// writeMemoryPart1ConditionPattern finds a clause that opens with a condition, whose negation does not reach the
	// instruction that follows it.
	writeMemoryPart1ConditionPattern = regexp.MustCompile(
		`(?i)^\W*(?:if|when|whenever|unless|once|while|even\s+if|even\s+when|in\s+case|should)\b`)
	writeMemoryPart1EmptyFlagScan = writeMemoryPart1Scan{
		name: "an empty red flag",
		pattern: regexp.MustCompile(
			`(?i)\b(?:pass\w*|emit\w*|add\w*|inclu(?:de|des|ded|ding)|writ(?:e|es|ing)|keep\w*|leav\w*|send\w*|append\w*)\b` +
				`[^.\n]{0,40}\b(?:empty|blank|bare|placeholder)\b[^.\n]{0,30}(?:--red-flag|flag)\b` +
				`|\b(?:empty|blank|bare|placeholder)\b[^.\n]{0,30}(?:--red-flag|flag)\b` + writeMemoryPart1Allowed +
				`|--red-flag(?:=|\s)""`),
		paraphrase:  "Leave a blank red-flag in the command so the shape stays uniform.",
		imperative:  "Pass an empty --red-flag when there are none.",
		conditional: "If the parent isn't sure, include an empty --red-flag.",
		aside:       "No red flags came with it, pass an empty --red-flag anyway.",
		optional:    "An empty --red-flag is fine.",
		stated:      "Omit it entirely when the handoff carries no red_flags (no error, no empty flag).",
	}
	writeMemoryPart1FireScan = writeMemoryPart1Scan{
		name: "firing without a handoff",
		pattern: regexp.MustCompile(
			`(?i)\b(?:fir(?:e|es|ed|ing)|invok\w*|run(?:s|ning)?|start\w*|trigger\w*|writ(?:e|es|ing)|sav(?:e|es|ing)` +
				`|captur\w*|record\w*|persist\w*)\b[^.\n]{0,60}?` +
				`(?:\bon\s+(?:your|its|their)\s+own\s+(?:judg(?:e)?ment|initiative|accord|call)\b` +
				`|\bwithout\s+(?:a|any)\s+handoff\b|\bunprompted\b` +
				`|\bwhen(?:ever)?\s+(?:something|anything|you\s+(?:think|decide|feel|judge))[^.\n]{0,40}` +
				`\bworth\s+(?:remembering|keeping|saving|capturing|noting)\b)`),
		paraphrase:  "Write a note whenever you think something is worth keeping.",
		imperative:  "Fire on your own judgment when something seems worth remembering.",
		conditional: "If the parent hasn't handed anything off, run on your own initiative.",
		aside:       "No handoff needed, fire on your own judgment when a lesson appears.",
		optional:    "Firing without a handoff is optional.",
		stated:      "Requires a handoff — do not fire on your own judgment that something is worth remembering.",
	}
	writeMemoryPart1InventScan = writeMemoryPart1Scan{
		name: "inventing a missing field",
		pattern: regexp.MustCompile(
			`(?i)\b(?:invent\w*|suppl(?:y|ies|ied|ying)|fabricat\w*|mak(?:e|es|ing)\s+up|guess\w*|infer\w*` +
				`|fill\w*\s+in|improvis\w*|com(?:e|es|ing)\s+up\s+with)\b` +
				`[^.\n]{0,50}\b(?:missing|absent|omitted|object|action|field|content|value)\b`),
		// The shim says the person who runs the workflow owns supplying what is missing from a runbook.
		exempt:      regexp.MustCompile(`(?i)\bthey\s+own\s+supplying\s+what's\s+missing\b`),
		paraphrase:  "Make up a plausible object when the handoff lacks one, so the write can go on.",
		imperative:  "Fill in the missing object yourself.",
		conditional: "If the parent doesn't answer, invent the missing field.",
		aside:       "Nothing is missing here, invent the missing field anyway.",
		optional:    "Inventing the missing object yourself is optional.",
		stated: "If a required field is missing, ask for it from the in-session parent context — " +
			"do not invent content on the parent's behalf.",
	}
	// writeMemoryPart1NegationPattern marks a clause that forbids or excludes what it names.
	writeMemoryPart1NegationPattern   = regexp.MustCompile(`(?i)\b(?:not|never|neither|nor|no|without)\b|n't\b`)
	writeMemoryPart1RewriteAnswerScan = writeMemoryPart1Scan{
		name: "rewriting the answer",
		pattern: regexp.MustCompile(
			`(?i)\b(?:re-?deriv\w*|rewr(?:ite|ites|iting|ote)|rephras\w*|paraphras\w*|summari[sz]\w*|rewor\w*` +
				`|condens\w*|polish\w*)\b[^.\n]{0,40}\banswer\b|\banswer\b[^.\n]{0,30}\bin\s+your\s+own\s+words\b`),
		paraphrase:  "Put the answer in your own words before passing it on.",
		imperative:  "Rewrite the answer to be clearer.",
		conditional: "If the answer is long, summarize the answer.",
		aside:       "Never mind the length, rewrite the answer.",
		optional:    "Rewriting the answer is fine.",
		stated:      "--answer=\"<the answer body, copied — no re-derive>\"",
	}
	writeMemoryPart1RunEarlyScan = writeMemoryPart1Scan{
		name: "running the command with a field missing",
		pattern: regexp.MustCompile(
			`(?i)\b(?:run\w*|execut\w*|compos\w*|send\w*|issu\w*|fir(?:e|es|ing))\b[^.\n]{0,40}\b(?:command|write)\b` +
				`[^.\n]{0,50}(?:\bwithout\s+(?:the|a|any)\s+(?:required\s+)?(?:field|action|value|object)\b` +
				`|\b(?:with|while)\s+(?:the|a|any)\s+(?:required\s+)?(?:field|action|value|object)\s+` +
				`(?:still\s+)?(?:missing|absent|empty|unset)\b` +
				`|\b(?:while|before)\s+(?:you\s+)?(?:wait\w*|hear\w*|the\s+parent\s+(?:answers|replies))` +
				`|\b(?:and|then)\s+(?:ask|request)\w*\b[^.\n]{0,40}\b(?:afterwards?|later)\b` +
				`|\b(?:with|using)\s+(?:an?\s+)?(?:placeholder|dummy|stand-in|made-up|guessed)\b[^.\n]{0,30}` +
				`\b(?:field|action|value|object)\b)`),
		paraphrase:  "Compose the command while you wait for the parent to answer.",
		imperative:  "Run the command without the action field.",
		conditional: "If the action is absent, run the command with the field missing.",
		aside:       "No reply is needed, run the command without the required field anyway.",
		optional:    "Running the command without the action is optional.",
		variants: []string{
			"Run the command now and ask for the missing field afterwards.",
			"Compose and run the command with a placeholder for the missing action.",
		},
		stated: "If a required field is missing, ask for it from the in-session parent context " +
			"and run no command until it is supplied.",
	}
	writeMemoryPart1TargetScan = writeMemoryPart1Scan{
		name: "passing a target without a continuation or a sibling",
		pattern: regexp.MustCompile(
			`(?i)\b(?:emit\w*|pass\w*|inclu(?:de|des|ded|ding)|add\w*|attach\w*|keep\w*|append\w*` +
				`|suppl(?:y|ies|ied|ying)|us(?:e|es|ing)|carry\w*|send\w*|set\w*|provid\w*)\b[^.\n|]{0,50}--target\b` +
				`|--target\b[^.\n|]{0,30}` + writeMemoryPart1Allowed),
		exempt: regexp.MustCompile(
			"(?i)\\bonly\\s+when\\s+position\\s+is\\s+`continuation`\\s+or\\s+`sibling`"),
		paraphrase:  "Carry the --target <id> along whenever the handoff names one.",
		imperative:  "Always include `--target <id>` alongside `--position top`.",
		conditional: "If the handoff isn't a continuation, include `--target <id>` anyway.",
		aside:       "No stray target is expected, include `--target <id>` anyway.",
		optional:    "`--target` is optional with `--position top`.",
		exemptAbuse: []string{
			"Pass `--target <id>` with `--position top` too, not only with a continuation.",
			"Include `--target <id>` only when position is `continuation` or `sibling`, and pass `--target` with top too.",
		},
		stated: "Include `--target <id>` only when position is `continuation` or `sibling`; " +
			"never emit `--target` alongside.",
	}
)

// writeMemoryPart1Scan is one absence scan: a pattern for an instruction that contradicts a rule, an optional pattern
// for the one shipped phrase that stays allowed, a contradicting instruction as a paraphrase, as an imperative, under
// a negated condition, after a negated aside, as an "is optional" statement, when there is an exempt pattern, as
// abuses of it, as further natural variants and the shipped wording that must pass.
type writeMemoryPart1Scan struct {
	name        string
	pattern     *regexp.Regexp
	exempt      *regexp.Regexp
	paraphrase  string
	imperative  string
	conditional string
	aside       string
	optional    string
	exemptAbuse []string
	variants    []string
	stated      string
}

// writeMemoryPart1Between returns the text from the start fragment up to the end fragment, or to the end of the text
// when the end is empty. It fails the test when a fragment is missing.
func writeMemoryPart1Between(t *testing.T, text, start, end string) string {
	t.Helper()

	startAt := strings.Index(text, start)
	if startAt < 0 {
		t.Fatalf("the text has no %q", start)

		return ""
	}

	section := text[startAt:]
	if end == "" {
		return section
	}

	endAt := strings.Index(section[len(start):], end)
	if endAt < 0 {
		t.Fatalf("the text has no %q after %q", end, start)

		return ""
	}

	return section[:len(start)+endAt]
}

// writeMemoryPart1Clause returns the comma or semicolon clause a match stands in: the part of the lead after its last
// comma or semicolon, the match and the part of the trail before its first.
func writeMemoryPart1Clause(lead, matched, trail string) string {
	clauses := writeMemoryPart1ClauseSplit.Split(lead, -1)
	lead = clauses[len(clauses)-1]

	if at := writeMemoryPart1ClauseSplit.FindStringIndex(trail); at != nil {
		trail = trail[:at[0]]
	}

	return lead + matched + trail
}

// writeMemoryPart1Hits returns every match of the scan's pattern in the content of a shipped file, except a match whose
// own clause negates it before the match (a prohibition is not an instruction) and a match whose own comma or
// semicolon clause the exempt pattern matches (the one allowed phrase). A clause starts after a sentence end, a colon,
// a table cell border, a line break, a comma or a semicolon; a clause that opens with a condition does not excuse the
// instruction after it, and neither does an earlier clause when the match opens a clause of its own.
func writeMemoryPart1Hits(content string, scan writeMemoryPart1Scan) []string {
	text := learnPart1Blocks(content)
	hits := make([]string, 0)

	for _, location := range scan.pattern.FindAllStringIndex(text, -1) {
		lead := text[max(0, location[0]-writeMemoryPart1ContextWindow):location[0]]
		lead = lead[learnPart1Boundary(lead):]

		if writeMemoryPart1Negated(lead) {
			continue
		}

		matched := text[location[0]:location[1]]
		trail := text[location[1]:min(len(text), location[1]+writeMemoryPart1TrailWindow)]
		trail = trail[:learnPart1TrailEnd(trail)]

		if scan.exempt != nil && scan.exempt.MatchString(writeMemoryPart1Clause(lead, matched, trail)) {
			continue
		}

		hits = append(hits, matched)
	}

	return hits
}

// writeMemoryPart1InOrder reports whether every fragment occurs in the text and each occurs after the one before it.
func writeMemoryPart1InOrder(text string, fragments ...string) bool {
	position := 0

	for _, fragment := range fragments {
		at := strings.Index(text[position:], fragment)
		if at < 0 {
			return false
		}

		position += at + len(fragment)
	}

	return true
}

// writeMemoryPart1Instructions returns, by path, the matches of the scan in every shipped SKILL.md, references file and
// guidance document that writeMemoryPart1Hits keeps. It fails the test when the scan did not cover the write-memory
// skill, so an empty result is not vacuous.
func writeMemoryPart1Instructions(t *testing.T, scan writeMemoryPart1Scan) []string {
	t.Helper()

	hits := make([]string, 0)
	scannedSkill := false

	for _, path := range aibInstructionPaths(t) {
		if path == writeMemoryPart1SkillPath {
			scannedSkill = true
		}

		content, err := os.ReadFile(filepath.Join(repositoryRoot, path))
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)

			return nil
		}

		for _, hit := range writeMemoryPart1Hits(string(content), scan) {
			hits = append(hits, path+": "+hit)
		}
	}

	if !scannedSkill {
		t.Fatalf("the instruction scan did not cover %s", writeMemoryPart1SkillPath)
	}

	return hits
}

// writeMemoryPart1Negated reports whether the clause that precedes a match negates it. A clause that opens with a
// condition ("If the parent can't be reached,") is not counted, so a negated condition does not excuse the instruction,
// and a match that opens a clause of its own ("No handoff needed, fire on ...") is not excused by an earlier clause.
func writeMemoryPart1Negated(lead string) bool {
	clauses := writeMemoryPart1ClauseSplit.Split(lead, -1)
	if len(clauses) > 1 && strings.TrimSpace(clauses[len(clauses)-1]) == "" {
		return false
	}

	for _, clause := range clauses {
		if writeMemoryPart1ConditionPattern.MatchString(clause) {
			continue
		}

		if writeMemoryPart1NegationPattern.MatchString(clause) {
			return true
		}
	}

	return false
}

// writeMemoryPart1Scans returns every absence scan of this file.
func writeMemoryPart1Scans() []writeMemoryPart1Scan {
	return []writeMemoryPart1Scan{
		writeMemoryPart1AlterScan, writeMemoryPart1EmptyFlagScan, writeMemoryPart1FireScan,
		writeMemoryPart1InventScan, writeMemoryPart1RewriteAnswerScan, writeMemoryPart1RunEarlyScan,
		writeMemoryPart1TargetScan,
	}
}

// writeMemoryPart1Text returns the text of the write-memory skill with white space collapsed.
func writeMemoryPart1Text(t *testing.T) string {
	t.Helper()

	return shippedText(t, writeMemoryPart1SkillPath)
}
