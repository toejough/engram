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
// instruction that contradicts it. The scenarios 792a13a1b, 792a13b1b and 792a13c1a are not tested here: the skill does
// not state the clause that is left over (decisions 861d402 to 861d404).

// Test792a13a1a_ACompleteHandoffIsExecuted reads the write-memory skill the repository ships: the worker was invoked by
// a parent skill, its job is to compose, execute, verify and report, the feedback command is composed from the
// handed-off fields, the command is run, and the written note path is reported to the parent.
func Test792a13a1a_ACompleteHandoffIsExecuted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	feedback := writeMemoryPart1Between(t, text, writeMemoryPart1FeedbackStart, writeMemoryPart1FactStart)
	execute := writeMemoryPart1Between(t, text, writeMemoryPart1ExecuteHeading, "")

	g.Expect(text).To(ContainSubstring("You were invoked by a parent skill that already made the judgment"))
	g.Expect(text).To(ContainSubstring("Your job is the write itself: compose, execute, verify, report."))
	g.Expect(text).To(ContainSubstring("**kind** — `fact`, `feedback`, `qa`, or `runbook`"))
	g.Expect(text).To(ContainSubstring("**content fields** — by kind, per the blocks below"))
	g.Expect(feedback).To(ContainSubstring("engram learn feedback"))
	g.Expect(execute).To(ContainSubstring("Run the command. On success the CLI prints the written note path(s)."))
	g.Expect(execute).To(ContainSubstring("Success → report the printed note path(s) to the parent flow in one line."))
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

// Test792a13b1a_AWriteSiteNamesTheWorker reads the learn and recall skills the repository ships: at each write site
// the parent names the write-memory skill as the next action and invokes it with the handoff, and recall's red flag
// says a write site hands off instead of composing the command itself.
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
	g.Expect(learn).To(ContainSubstring("write-memory composes, executes, and reports the note path."))
	g.Expect(recall).To(ContainSubstring("Invoke the **write-memory** skill with this handoff — kind=fact or feedback"))
	g.Expect(recall).To(ContainSubstring("Hand ONE synthesis note per conclusion to the **write-memory** skill"))
	g.Expect(recall).To(ContainSubstring("ALSO invoke the **write-memory** skill** with kind=qa"))
	g.Expect(recall).To(ContainSubstring("| You composed an engram learn command yourself at a write site | " +
		"Write sites hand off to write-memory — parents judge, the worker writes |"))
}

// Test792a13c1b_ContentIsNeverInvented reads the write-memory skill the repository ships: a fact handoff carries an
// object among its content fields, a missing required field is asked for and the worker does not invent content on
// the parent's behalf, and no shipped instruction tells a skill to make up a missing field.
func Test792a13c1b_ContentIsNeverInvented(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	fact := writeMemoryPart1Between(t, text, writeMemoryPart1FactStart, writeMemoryPart1RunbookStart)

	g.Expect(fact).To(ContainSubstring("--object=\"<the standard or value>\""))
	g.Expect(text).To(ContainSubstring("If a required field is missing, ask for it from the in-session parent context — " +
		"do not invent content on the parent's behalf."))
	g.Expect(writeMemoryPart1Instructions(t, writeMemoryPart1InventScan)).To(BeEmpty())
}

// Test792a13d1a_FeedbackComposition reads the write-memory skill the repository ships: the feedback command is
// engram learn feedback with the slug, the position (top when the handoff carries none), the source, the situation as
// the retrieval-shaped phrase, the behavior, the impact and the action, in that order.
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
	g.Expect(text).To(ContainSubstring("`--position` defaults to `top`"))
	g.Expect(writeMemoryPart1InOrder(feedback, "engram learn feedback", "--slug", "--position", "--source=",
		"--situation=", "--behavior=", "--impact=", "--action=")).To(BeTrue())
}

// Test792a13d1b_FactComposition reads the write-memory skill the repository ships: the fact command is engram learn
// fact with the slug, the position (top when the handoff carries none), the source, the situation as the
// retrieval-shaped phrase, the subject, the predicate and the object, in that order.
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
	g.Expect(text).To(ContainSubstring("`--position` defaults to `top`"))
	g.Expect(writeMemoryPart1InOrder(fact, "engram learn fact", "--slug", "--position", "--source=", "--situation=",
		"--subject=", "--predicate=", "--object=")).To(BeTrue())
}

// Test792a13e1a_ARunbookHandoffIsExecuted reads the write-memory skill the repository ships: the runbook command is
// engram learn runbook with the slug, the position, the source, the situation, the done-when and the body, one
// --red-flag per handed-off red flag in order, and the written note path is reported.
func Test792a13e1a_ARunbookHandoffIsExecuted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	runbook := writeMemoryPart1Between(t, text, writeMemoryPart1RunbookStart, writeMemoryPart1QAStart)
	execute := writeMemoryPart1Between(t, text, writeMemoryPart1ExecuteHeading, "")

	g.Expect(runbook).To(ContainSubstring("engram learn runbook --slug <kebab-slug>"))
	g.Expect(runbook).To(ContainSubstring("--position <top|continuation|sibling>"))
	g.Expect(runbook).To(ContainSubstring("--source=\"<source>\""))
	g.Expect(runbook).To(ContainSubstring("--situation=\"<retrieval-shaped phrase: when should this runbook be used>\""))
	g.Expect(runbook).To(ContainSubstring("--done-when=\"<what should be true when the procedure is complete>\""))
	g.Expect(runbook).To(ContainSubstring("--body=\"<numbered steps, may [[wikilink]] fact/feedback notes to consider>\""))
	g.Expect(runbook).To(ContainSubstring("[--red-flag=\"<task-specific failure mode>\" ...]"))
	g.Expect(text).To(ContainSubstring("One `--red-flag` per handed-off `red_flags` entry, in order;"))
	g.Expect(execute).To(ContainSubstring("Success → report the printed note path(s) to the parent flow in one line."))
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
// or target gets --position top and no --target, and no shipped instruction tells a skill to pass a target that is
// not tied to a continuation or a sibling.
func Test792a13g1a_NoPlacementMeansTop(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	fact := writeMemoryPart1Between(t, text, writeMemoryPart1FactStart, writeMemoryPart1RunbookStart)

	g.Expect(fact).To(ContainSubstring("--position <top|continuation|sibling> [--target <id>]"))
	g.Expect(text).To(ContainSubstring("Omitted → `--position top` (today's default), fact/feedback/runbook only."))
	g.Expect(text).To(ContainSubstring("`--position` defaults to `top` and `--target` is omitted whenever the parent's " +
		"handoff doesn't include a position/target"))
	g.Expect(writeMemoryPart1Instructions(t, writeMemoryPart1TargetScan)).To(BeEmpty())
}

// Test792a13g1b_AContinuationCarriesItsTarget reads the write-memory skill the repository ships: a handed-off position
// of continuation is passed as --position, the target note ID is required with it, and --target <id> is included when
// the position is continuation.
func Test792a13g1b_AContinuationCarriesItsTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)
	feedback := writeMemoryPart1Between(t, text, writeMemoryPart1FeedbackStart, writeMemoryPart1FactStart)

	g.Expect(feedback).To(ContainSubstring("--position <top|continuation|sibling> [--target <id>]"))
	g.Expect(text).To(ContainSubstring("optional **position** / **target** — Luhmann placement (`top` (default), " +
		"`continuation`, or `sibling`) plus the target note ID (required when position is not `top`)"))
	g.Expect(text).To(ContainSubstring("Include `--target <id>` only when position is `continuation` or `sibling`;"))
}

// Test792a13g1c_TopNeverCarriesATarget reads the write-memory skill the repository ships: --target is never emitted
// alongside --position top, and no shipped instruction tells a skill to pass a target with a top position.
func Test792a13g1c_TopNeverCarriesATarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := writeMemoryPart1Text(t)

	g.Expect(text).To(ContainSubstring("never emit `--target` alongside `--position top`."))
	g.Expect(text).To(ContainSubstring("Include `--target <id>` only when position is `continuation` or `sibling`;"))
	g.Expect(writeMemoryPart1Instructions(t, writeMemoryPart1TargetScan)).To(BeEmpty())
}

// TestWriteMemoryPart1Hits shows the scanner of the absence checks of this file finds an instruction that contradicts
// each rule (as a paraphrase, as an imperative, under a negated condition and in a table row after a negated row) and
// passes the shipped wording that states the rule or forbids the opposite.
func TestWriteMemoryPart1Hits(t *testing.T) {
	t.Parallel()

	for _, scan := range writeMemoryPart1Scans() {
		t.Run(scan.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(writeMemoryPart1Hits(scan.paraphrase, scan)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart1Hits(scan.imperative, scan)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart1Hits(scan.conditional, scan)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart1Hits("| Rule | Never improvise |\n| Next | "+scan.imperative+" |\n", scan)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart1Hits(scan.stated, scan)).To(BeEmpty())
		})
	}
}

// unexported constants.
const (
	writeMemoryPart1AppendStart     = "Append to any kind:"
	writeMemoryPart1ContextWindow   = 160
	writeMemoryPart1ExecuteHeading  = "## Execute, verify, report"
	writeMemoryPart1FactStart       = "kind=fact:"
	writeMemoryPart1FeedbackStart   = "kind=feedback:"
	writeMemoryPart1LearnPath       = "agent-instructions/skills/learn/SKILL.md"
	writeMemoryPart1LearnWriteSites = 6
	writeMemoryPart1QAStart         = "kind=qa:"
	writeMemoryPart1RecallPath      = "agent-instructions/skills/recall/SKILL.md"
	writeMemoryPart1RunbookStart    = "kind=runbook:"
	writeMemoryPart1SkillPath       = "agent-instructions/skills/write-memory/SKILL.md"
	writeMemoryPart1TrailWindow     = 60
)

// unexported variables.
var (
	writeMemoryPart1ClauseSplit = regexp.MustCompile(`[,;]\s+`)
	// writeMemoryPart1ConditionPattern finds a clause that opens with a condition, whose negation does not reach the
	// instruction that follows it.
	writeMemoryPart1ConditionPattern = regexp.MustCompile(
		`(?i)^\W*(?:if|when|whenever|unless|once|while|even\s+if|even\s+when|in\s+case|should)\b`)
	writeMemoryPart1EmptyFlagScan = writeMemoryPart1Scan{
		name: "an empty red flag",
		pattern: regexp.MustCompile(
			`(?i)\b(?:pass|emit|add|include|write|keep|leave|send|append)\b[^.\n]{0,40}\b(?:empty|blank|bare|placeholder)\b` +
				`[^.\n]{0,30}(?:--red-flag|flag)\b|--red-flag(?:=|\s)""`),
		paraphrase:  "Leave a blank red-flag in the command so the shape stays uniform.",
		imperative:  "Pass an empty --red-flag when there are none.",
		conditional: "If the parent isn't sure, include an empty --red-flag.",
		stated:      "Omit it entirely when the handoff carries no red_flags (no error, no empty flag).",
	}
	writeMemoryPart1FireScan = writeMemoryPart1Scan{
		name: "firing without a handoff",
		pattern: regexp.MustCompile(
			`(?i)\b(?:fire|invoke|run|start|trigger|write|save|capture|record|persist)\b[^.\n]{0,60}?` +
				`(?:\bon\s+(?:your|its|their)\s+own\s+(?:judg(?:e)?ment|initiative|accord|call)\b` +
				`|\bwithout\s+(?:a|any)\s+handoff\b|\bunprompted\b` +
				`|\bwhen(?:ever)?\s+(?:something|anything|you\s+(?:think|decide|feel|judge))[^.\n]{0,40}` +
				`\bworth\s+(?:remembering|keeping|saving|capturing|noting)\b)`),
		paraphrase:  "Write a note whenever you think something is worth keeping.",
		imperative:  "Fire on your own judgment when something seems worth remembering.",
		conditional: "If the parent hasn't handed anything off, run on your own initiative.",
		stated:      "Requires a handoff — do not fire on your own judgment that something is worth remembering.",
	}
	writeMemoryPart1InventScan = writeMemoryPart1Scan{
		name: "inventing a missing field",
		pattern: regexp.MustCompile(
			`(?i)\b(?:invent|supply|fabricate|make\s+up|guess|infer|fill\s+in|improvise|come\s+up\s+with)\b` +
				`[^.\n]{0,50}\b(?:missing|absent|omitted|object|action|field|content|value)\b`),
		paraphrase:  "Make up a plausible object when the handoff lacks one, so the write can go on.",
		imperative:  "Fill in the missing object yourself.",
		conditional: "If the parent doesn't answer, invent the missing field.",
		stated: "If a required field is missing, ask for it from the in-session parent context — " +
			"do not invent content on the parent's behalf.",
	}
	// writeMemoryPart1NegationPattern marks a clause that forbids or excludes what it names.
	writeMemoryPart1NegationPattern   = regexp.MustCompile(`(?i)\b(?:not|never|neither|nor|no|without)\b|n't\b`)
	writeMemoryPart1RewriteAnswerScan = writeMemoryPart1Scan{
		name: "rewriting the answer",
		pattern: regexp.MustCompile(
			`(?i)\b(?:re-?derive|rewrite|rephrase|paraphrase|summari[sz]e|reword|condense|polish)\b` +
				`[^.\n]{0,40}\banswer\b|\banswer\b[^.\n]{0,30}\bin\s+your\s+own\s+words\b`),
		paraphrase:  "Put the answer in your own words before passing it on.",
		imperative:  "Rewrite the answer to be clearer.",
		conditional: "If the answer is long, summarize the answer.",
		stated:      "--answer=\"<the answer body, copied — no re-derive>\"",
	}
	writeMemoryPart1TargetScan = writeMemoryPart1Scan{
		name: "passing a target without a continuation or a sibling",
		pattern: regexp.MustCompile(
			`(?i)\b(?:emit|pass|include|add|attach|keep|append|supply|use|carry|send|set|provide)\b[^.\n|]{0,50}--target\b`),
		exempt:      regexp.MustCompile(`(?i)\bonly\b`),
		paraphrase:  "Carry the --target <id> along whenever the handoff names one.",
		imperative:  "Always include `--target <id>` alongside `--position top`.",
		conditional: "If the handoff isn't a continuation, include `--target <id>` anyway.",
		stated: "Include `--target <id>` only when position is `continuation` or `sibling`; " +
			"never emit `--target` alongside.",
	}
)

// writeMemoryPart1Scan is one absence scan: a pattern for an instruction that contradicts a rule, an optional pattern
// for a clause that stays allowed, a contradicting instruction as a paraphrase, as an imperative and under a negated
// condition, and the shipped wording that must pass.
type writeMemoryPart1Scan struct {
	name        string
	pattern     *regexp.Regexp
	exempt      *regexp.Regexp
	paraphrase  string
	imperative  string
	conditional string
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

// writeMemoryPart1Hits returns every match of the scan's pattern in the content of a shipped file, except a match whose
// own clause negates it before the match (a prohibition is not an instruction) and a match whose clause the exempt
// pattern matches (an allowed use). A clause starts after a sentence end, a colon, a table cell border, a line break, a
// comma or a semicolon, and a clause that opens with a condition does not excuse the instruction after it.
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

		if scan.exempt != nil && scan.exempt.MatchString(lead+matched+trail) {
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
// condition ("If the parent can't be reached,") is not counted, so a negated condition does not excuse the instruction.
func writeMemoryPart1Negated(lead string) bool {
	for _, clause := range writeMemoryPart1ClauseSplit.Split(lead, -1) {
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
		writeMemoryPart1EmptyFlagScan, writeMemoryPart1FireScan, writeMemoryPart1InventScan,
		writeMemoryPart1RewriteAnswerScan, writeMemoryPart1TargetScan,
	}
}

// writeMemoryPart1Text returns the text of the write-memory skill with white space collapsed.
func writeMemoryPart1Text(t *testing.T) string {
	t.Helper()

	return shippedText(t, writeMemoryPart1SkillPath)
}
