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

// The tests of this file read the write-memory skill the repository ships and verify, per the Why column of node
// 861c2zzzz, that the shipped rule is stated, not that an agent follows it. The helper shippedText is in
// skillrules_integration_internal_test.go, and the helper aibInstructionPaths in
// skillrules_agent_instruction_bundle_integration_internal_test.go. The scenarios of 792a13a to 792a13g belong to
// part 1 and are not repeated here.

// Test792a13h1a_AValueBeginningWithADashSurvives reads the write-memory skill the repository ships: a free-text flag
// value is written --flag="value" with the quote touching the equals sign and no space, because a value that begins
// with a dash is otherwise misread as a new flag, and no shipped instruction writes a free-text flag with a space.
func Test792a13h1a_AValueBeginningWithADashSurvives(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("A free-text flag value (for example `--situation`, `--source`, `--subject`, " +
		"`--behavior`) must be written as `--flag=\"value\"` — the quote touching the `=`, no space —"))
	g.Expect(text).To(ContainSubstring("because a value beginning with `-` is otherwise misread as a new flag and the " +
		"command fails."))
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2SpacedFlagScan)).To(BeEmpty())
}

// Test792a13i1a_ChunkSourcesBecomeFlags reads the write-memory skill the repository ships: the handoff carries
// chunk-source IDs as source#anchor, and the worker appends one --chunk-source flag per provided chunk ID.
func Test792a13i1a_ChunkSourcesBecomeFlags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("optional **chunk-sources** — `<source#anchor>` chunk IDs (provenance)"))
	g.Expect(text).To(ContainSubstring("one `--chunk-source <source#anchor>` per provided chunk ID"))
}

// Test792a13i1b_TagsArePassedThroughExactly reads the write-memory skill the repository ships: the worker appends one
// --tag per handed-off tag, exactly as provided, never invents a tag and never writes the vocab namespace, and no
// shipped instruction has the worker add a tag of its own.
func Test792a13i1b_TagsArePassedThroughExactly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("optional **tags** — categorical `<family>` or `<family>/<value>` strings " +
		"(kebab-case; fact/feedback/runbook only), e.g. `work-kind/rename`, `tier/cheap`, `outcome/pass`"))
	g.Expect(text).To(ContainSubstring("one `--tag <t>` per provided tag (fact/feedback/runbook only"))
	g.Expect(text).To(ContainSubstring("pass them through exactly as provided;"))
	g.Expect(text).To(ContainSubstring("never invent tags and never write the `vocab/` namespace yourself."))
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2OwnTagScan)).To(BeEmpty())
}

// Test792a13i1c_ASupersessionIsPassedAsGiven reads the write-memory skill the repository ships: the handoff carries a
// supersession as basename, type and claim, the worker appends --supersedes="<basename>|<type>|<claim>" for each, and
// the binary writes the Supersedes line itself.
func Test792a13i1c_ASupersessionIsPassedAsGiven(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("optional **supersedes** — `<basename>|<type>|<claim>` " +
		"(types: `updates|narrows|refutes`), when the parent determined this write corrects a surfaced note"))
	g.Expect(text).To(ContainSubstring("`--supersedes=\"<basename>|<type>|<claim>\"` if provided (repeatable)"))
	g.Expect(text).To(ContainSubstring("writes the `Supersedes:` body line itself when `--supersedes` is passed."))
}

// Test792a13j1a_QATagsAreDroppedAndReported reads the write-memory skill the repository ships: a qa handoff that
// carries tags has them dropped, the qa command has no --tag, the exact line is appended to the output even when the
// output is only the command, and no shipped instruction keeps tags on a qa note.
func Test792a13j1a_QATagsAreDroppedAndReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)
	qaCommand := writeMemoryPart2Between(t, text, "kind=qa:", "Append to any kind:")

	g.Expect(qaCommand).To(ContainSubstring("engram learn qa"))
	g.Expect(qaCommand).NotTo(ContainSubstring("--tag"))
	g.Expect(text).To(ContainSubstring("`engram learn qa` and `engram amend` take no `--tag`;"))
	g.Expect(text).To(ContainSubstring("a qa handoff carrying tags → drop them and say so:"))
	g.Expect(text).To(ContainSubstring("append the exact line `tags dropped: qa takes no tag flags` to whatever you " +
		"output, even command-only output)"))
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2QATagScan)).To(BeEmpty())
}

// Test792a13k1a_OneKindsFlagsPerCommand reads the write-memory skill the repository ships: the fact command carries
// the fact flags and none of the feedback or runbook flags, and the worker never mixes the flags of two kinds in one
// command.
func Test792a13k1a_OneKindsFlagsPerCommand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)
	factCommand := writeMemoryPart2Between(t, text, "kind=fact:", "`--position` defaults to `top`")

	g.Expect(factCommand).To(HavePrefix("kind=fact:"))
	g.Expect(factCommand).To(ContainSubstring("engram learn fact"))

	for _, flag := range []string{
		"--behavior", "--impact", "--action", "--done-when", "--body", "--red-flag", "--trigger",
	} {
		g.Expect(factCommand).NotTo(ContainSubstring(flag))
	}

	g.Expect(text).To(ContainSubstring("Never mix fact flags (`--subject/--predicate/--object`), feedback flags " +
		"(`--behavior/--impact/--action`), or runbook flags (`--done-when/--body/--red-flag/--trigger`) in one command."))
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2MixedFlagsScan)).To(BeEmpty())
}

// Test792a13k1b_ACitedNoteIsWrittenInline reads the write-memory skill the repository ships: an inline [[basename]]
// citation is typed directly into the --body or the --object when the handoff content calls for citing a note, the
// rule against hand-authored notation does not restrict it, and no shipped instruction strips it.
func Test792a13k1b_ACitedNoteIsWrittenInline(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("This does NOT restrict inline `[[basename]]` prose citations:"))
	g.Expect(text).To(ContainSubstring("hand-type one directly in `--body`"))
	g.Expect(text).To(ContainSubstring("or `--object` whenever the parent's handoff content calls for citing a " +
		"fact/feedback note"))
	g.Expect(text).To(ContainSubstring("--body=\"<numbered steps, may [[wikilink]] fact/feedback notes to consider>\""))
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2StrippedCitationScan)).To(BeEmpty())
}

// Test792a13l1a_TriggersBecomeFlags reads the write-memory skill the repository ships: the handoff carries triggers
// as literal cues such as /please and "take this end-to-end", and the worker appends one --trigger per handed-off
// trigger, in order, after any --red-flag.
func Test792a13l1a_TriggersBecomeFlags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)
	runbookCommand := writeMemoryPart2Between(t, text, "kind=runbook:", "Same `--position`/`--target` disposition rules")

	g.Expect(text).To(ContainSubstring("optional **triggers** — literal cues in a user's message that should surface " +
		"this runbook first (runbook only), e.g. `/please`, \"take this end-to-end\""))
	g.Expect(text).To(ContainSubstring("One `--trigger` per handed-off `triggers` entry, in order, placed after any " +
		"`--red-flag`;"))
	g.Expect(writeMemoryPart2Before(runbookCommand, "[--red-flag=", "[--trigger=")).To(BeTrue())
}

// Test792a13l1b_NoTriggersNoFlag reads the write-memory skill the repository ships: the worker omits --trigger
// entirely when the handoff carries no triggers, and no shipped instruction emits an empty, default or mandatory
// trigger.
func Test792a13l1b_NoTriggersNoFlag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("omit entirely when the handoff carries none."))
	g.Expect(text).To(ContainSubstring("[--trigger=\"<literal cue>\" ...]"))
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2TriggerEmissionScan)).To(BeEmpty())
}

// Test792a13m1a_ADistinctiveBareWordIsComposedWithoutAsking reads the write-memory skill the repository ships: a
// trigger is a slash form, a multi-word phrase or a single distinctive word, curate is the distinctive example that
// hits curate and /curate, and no shipped instruction has the worker ask about every trigger.
func Test792a13m1a_ADistinctiveBareWordIsComposedWithoutAsking(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("A trigger is a cue word or phrase the user (or an engram notice) would " +
		"literally write: a slash form (`/please`),"))
	g.Expect(text).To(ContainSubstring("a multi-word phrase (\"take this end-to-end\"),"))
	g.Expect(text).To(ContainSubstring("or a single distinctive word (`curate`), at least 3 characters."))
	g.Expect(text).To(ContainSubstring("so `curate` hits \"curate\" and \"/curate\" but not \"accurate\"."))
	g.Expect(text).To(ContainSubstring("(`curate` qualifies;"))
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2AskEveryTriggerScan)).To(BeEmpty())
}

// Test792a13m1b_AVeryCommonBareWordIsQuestioned reads the write-memory skill the repository ships: a handed-off single
// word that is common in ordinary prose, such as fix, is not passed through and the worker asks the parent for a
// distinctive cue, and no shipped instruction passes a common word through.
func Test792a13m1b_AVeryCommonBareWordIsQuestioned(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("A single word must be distinctive enough that whole-word matching will not " +
		"fire on ordinary prose"))
	g.Expect(text).To(ContainSubstring("(`curate` qualifies; \"fix\", \"run\", \"the\" do not)."))
	g.Expect(text).To(ContainSubstring("A handed-off single word that is common in ordinary prose → ask the parent " +
		"for a distinctive cue instead of passing it through,"))
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2CommonWordScan)).To(BeEmpty())
}

// Test792a13m1c_ADeliberateOverFireIsPassedThrough reads the write-memory skill the repository ships: a common word is
// passed through when the handoff states that over-firing is a deliberate, accepted choice, and the please runbook is
// the example.
func Test792a13m1c_ADeliberateOverFireIsPassedThrough(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("unless the handoff states that over-firing is a deliberate, accepted choice"))
	g.Expect(text).To(ContainSubstring("(e.g. \"please\" on the please runbook, 2026-09-20);"))
	g.Expect(text).To(ContainSubstring("then pass it through."))
}

// Test792a13n1a_SuccessIsReportedInOneLine reads the write-memory skill the repository ships: the worker runs the
// command, the CLI prints the written note path on success, and the worker reports the printed path to the parent in
// one line.
func Test792a13n1a_SuccessIsReportedInOneLine(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Run the command. On success the CLI prints the written note path(s)."))
	g.Expect(text).To(ContainSubstring("Success → report the printed note path(s) to the parent flow in one line."))
}

// Test792a13n1b_ANamedErrorIsCorrectedAndRetried reads the write-memory skill the repository ships: on a CLI error the
// worker reads it, fixes exactly the named problem such as a typo'd flag, retries, and retries at most twice, and no
// shipped instruction retries without that bound or fixes more than the named problem.
func Test792a13n1b_ANamedErrorIsCorrectedAndRetried(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("CLI error → read it, fix exactly the named problem (missing/typo'd flag, " +
		"bad value), retry."))
	g.Expect(text).To(ContainSubstring("Max 2 retries."))
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2UnboundedRetryScan)).To(BeEmpty())
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2BeyondNamedScan)).To(BeEmpty())
}

// Test792a13n1c_PersistentFailureIsReportedVerbatim reads the write-memory skill the repository ships: after the
// retries the worker reports the exact command and the CLI error verbatim, never silently skips a handed-off write,
// and no shipped instruction skips the write or summarizes the error.
func Test792a13n1c_PersistentFailureIsReportedVerbatim(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, writeMemoryPart2SkillPath)

	g.Expect(text).To(ContainSubstring("Still failing after retries → report the exact command"))
	g.Expect(text).To(ContainSubstring("and the CLI error verbatim."))
	g.Expect(text).To(ContainSubstring("Never silently skip a handed-off write."))
	g.Expect(writeMemoryPart2Contradictions(t, writeMemoryPart2SkippedFailureScan)).To(BeEmpty())
}

// TestWriteMemoryPart2Scans supports the absence scans of this file: each scan finds a paraphrase, an imperative and
// an imperative that stands after a negated condition (in the comma form and in the arrow form), each of which
// contradicts the rule, and finds nothing in the shipped wording that states the rule or forbids the opposite.
func TestWriteMemoryPart2Scans(t *testing.T) {
	t.Parallel()

	for _, scan := range writeMemoryPart2Scans() {
		t.Run(scan.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(writeMemoryPart2Hits(scan.paraphrase, scan.pattern)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart2Hits(scan.imperative, scan.pattern)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart2Hits(scan.conditional, scan.pattern)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart2Hits(scan.arrow, scan.pattern)).NotTo(BeEmpty())
			g.Expect(writeMemoryPart2Hits(scan.stated, scan.pattern)).To(BeEmpty())
		})
	}
}

// unexported constants.
const (
	// writeMemoryPart2Gap matches one character of the stretch between the words of an instruction: it stops at a clause
	// border (the arrow of the skill's own conditional idiom included), so a pattern never spans two clauses.
	writeMemoryPart2Gap       = `[^.?!:|\n,;(—→]`
	writeMemoryPart2SkillPath = "agent-instructions/skills/write-memory/SKILL.md"
)

// unexported variables.
var (
	// writeMemoryPart2AskEveryTriggerScan finds an instruction to ask about every trigger.
	writeMemoryPart2AskEveryTriggerScan = writeMemoryPart2Scan{
		name: "asking about every trigger",
		pattern: regexp.MustCompile(`(?i)\b(?:ask|asks|confirm|confirms|check|checks|verify|verifies)\b` +
			writeMemoryPart2Gap + `{0,60}\b(?:every|each|all|any)\s+(?:handed-off\s+)?(?:triggers?|cues?)\b`),
		paraphrase:  "The worker confirms each trigger with the parent before composing.",
		imperative:  "Ask the parent to confirm every trigger first.",
		conditional: "If the trigger isn't a slash form, ask the parent about each cue.",
		arrow:       "A trigger that isn't a slash form → ask the parent about each cue.",
		stated: "A handed-off single word that is common in ordinary prose → ask the parent for a distinctive cue " +
			"instead of passing it through, unless the handoff states that over-firing is a deliberate, accepted choice",
	}
	// writeMemoryPart2BeyondNamedScan finds an instruction to fix more than the problem the CLI error names.
	writeMemoryPart2BeyondNamedScan = writeMemoryPart2Scan{
		name: "fixing more than the named problem",
		pattern: regexp.MustCompile(`(?i)\b(?:fix|correct|change|adjust|rewrite|repair)\b` + writeMemoryPart2Gap +
			`{0,30}\b(?:everything|anything\s+else|all\s+other|other\s+flags?|the\s+whole\s+command|unrelated)\b`),
		paraphrase:  "Repair anything else that looks off in the command as well.",
		imperative:  "Rewrite the whole command from the handoff before retrying.",
		conditional: "If the error isn't specific, fix everything that looks wrong.",
		arrow:       "A CLI error that isn't specific → fix everything that looks wrong.",
		stated:      "CLI error → read it, fix exactly the named problem (missing/typo'd flag, bad value), retry.",
	}
	// writeMemoryPart2BlockStart finds a line that starts a new block: a table row, a list item, a heading or a fence.
	writeMemoryPart2BlockStart = regexp.MustCompile("^(?:\\||[-*+] |#|\\d+\\. |```)")
	// writeMemoryPart2ClauseBorders are the strings that end a clause: a sentence end, a colon, a table cell border, a
	// line break, a comma, a semicolon, a dash, an arrow and an opening parenthesis.
	writeMemoryPart2ClauseBorders = []string{". ", "? ", "! ", ": ", "| ", "\n", ", ", "; ", " — ", " → ", "("}
	// writeMemoryPart2CommonWordScan finds an instruction to pass a common word through as a trigger.
	writeMemoryPart2CommonWordScan = writeMemoryPart2Scan{
		name: "passing a common word through",
		pattern: regexp.MustCompile(`(?i)\b(?:pass|emit|append|compose|use|accept|keep)\b` + writeMemoryPart2Gap +
			`{0,40}\b(?:common|ordinary|everyday|generic)\b` + writeMemoryPart2Gap + `{0,30}\b(?:word|trigger|cue)s?\b` +
			`|\b(?:common|ordinary|everyday)\b` + writeMemoryPart2Gap + `{0,30}\b(?:word|trigger|cue)s?\b` +
			writeMemoryPart2Gap + `{0,40}\b(?:is|are|may\s+be|can\s+be|gets?)\s+` +
			`(?:passed|emitted|used|accepted|composed|kept)\b|--trigger="(?:fix|run|the)"`),
		paraphrase:  "A common word such as fix is passed through as a trigger as-is.",
		imperative:  "Pass a common word like fix through as a trigger.",
		conditional: "If the parent isn't responding, emit a common word like fix as the trigger.",
		arrow:       "A parent that isn't responding → emit a common word like fix as the trigger.",
		stated: "A handed-off single word that is common in ordinary prose → ask the parent for a distinctive cue " +
			"instead of passing it through, unless the handoff states that over-firing is a deliberate, accepted " +
			"choice (e.g. \"please\" on the please runbook, 2026-09-20); then pass it through.",
	}
	// writeMemoryPart2MixedFlagsScan finds an instruction to put the flags of two kinds in one command.
	writeMemoryPart2MixedFlagsScan = writeMemoryPart2Scan{
		name: "mixing the flags of two kinds",
		pattern: regexp.MustCompile(`(?i)\b(?:mix|combine|blend|merge)\b` + writeMemoryPart2Gap +
			`{0,40}\b(?:fact|feedback|runbook)\b` + writeMemoryPart2Gap + `{0,30}\bflags?\b` +
			`|\b(?:also|additionally)\b` + writeMemoryPart2Gap + `{0,30}--(?:behavior|impact|action|done-when|body|` +
			`red-flag|trigger|subject|predicate|object)\b` +
			`|\b(?:fact|feedback|runbook)\s+(?:command|note)s?\s+(?:may|can|should)\s+(?:also\s+)?` +
			`(?:carry|take|include|have)\b` + writeMemoryPart2Gap + `{0,30}\bflags?\b`),
		paraphrase:  "A fact command may carry feedback flags as well.",
		imperative:  "Combine the fact and feedback flags in one command.",
		conditional: "If the handoff isn't a feedback note, also add --behavior to the command.",
		arrow:       "A handoff that isn't a feedback note → also add --behavior to the command.",
		stated: "Never mix fact flags (`--subject/--predicate/--object`), feedback flags " +
			"(`--behavior/--impact/--action`), or runbook flags (`--done-when/--body/--red-flag/--trigger`) in one command.",
	}
	// writeMemoryPart2Negation finds a negation in the clause that leads up to a match and holds it.
	writeMemoryPart2Negation = regexp.MustCompile(`(?i)\b(?:not|never|no|neither|nor|without|cannot)\b|n't\b`)
	// writeMemoryPart2OwnTagScan finds an instruction to add a tag the handoff did not carry.
	writeMemoryPart2OwnTagScan = writeMemoryPart2Scan{
		name: "adding a tag of the worker's own",
		pattern: regexp.MustCompile(`(?i)\b(?:invent|make\s+up|derive|generate|supply|come\s+up\s+with)\b` +
			writeMemoryPart2Gap + `{0,40}\btags?\b|\btags?\s+of\s+(?:its|your|the\s+worker's)\s+own\b` +
			`|\b(?:add|create|choose|pick|attach)\b` + writeMemoryPart2Gap + `{0,20}\b(?:own|extra|additional|new|` +
			`custom|your)\b` + writeMemoryPart2Gap + `{0,15}\btags?\b|\b(?:write|use|add)\b` + writeMemoryPart2Gap +
			`{0,20}\bvocab/`),
		paraphrase:  "The worker may also supply tags of its own describing the work.",
		imperative:  "Add an extra tag of your own that names the work kind.",
		conditional: "If the handoff doesn't carry tags, invent tags that describe the work.",
		arrow:       "A handoff that doesn't carry tags → invent tags that describe the work.",
		stated: "Handed-off --tag categoricals ride the same `tags:` list as vocab but are NOT vocab: pass them " +
			"through exactly as provided; never invent tags and never write the `vocab/` namespace yourself.",
	}
	// writeMemoryPart2QATagScan finds an instruction to keep tags on a qa note.
	writeMemoryPart2QATagScan = writeMemoryPart2Scan{
		name: "tags on a qa note",
		pattern: regexp.MustCompile(`(?i)\b(?:keep|retain|pass|append|add|include|attach|carry|write|put|apply)\b` +
			writeMemoryPart2Gap + `{0,40}(?:\btags?|--tag)\b` + writeMemoryPart2Gap + `{0,40}\b(?:to|for|on|with|in)\b` +
			writeMemoryPart2Gap + `{0,20}\bqa\b|\bqa\b` + writeMemoryPart2Gap + `{0,40}\b(?:takes?|accepts?|gets?|` +
			`carries|keeps?|allows?)\b` + writeMemoryPart2Gap + `{0,15}(?:\btags?|--tag)\b|\bqa\b` +
			writeMemoryPart2Gap + `{0,30}\btags?\b` + writeMemoryPart2Gap + `{0,15}\b(?:is|are)\s+(?:allowed|optional|` +
			`fine|supported)\b`),
		paraphrase:  "A qa note accepts tags like any other kind.",
		imperative:  "Pass the tags to qa as --tag flags.",
		conditional: "If the qa handoff isn't empty, append the tags to the qa command.",
		arrow:       "A qa handoff that isn't empty → append the tags to the qa command.",
		stated: "(fact/feedback/runbook only — `engram learn qa` and `engram amend` take no `--tag`; a qa handoff " +
			"carrying tags → drop them and say so: append the exact line `tags dropped: qa takes no tag flags` to " +
			"whatever you output, even command-only output)",
	}
	// writeMemoryPart2SkippedFailureScan finds an instruction to skip a failed write or to soften its report.
	writeMemoryPart2SkippedFailureScan = writeMemoryPart2Scan{
		name: "skipping or softening a failed write",
		pattern: regexp.MustCompile(`(?i)\b(?:skip|drop|ignore|abandon|swallow)\b` + writeMemoryPart2Gap +
			`{0,30}\b(?:handoff|handed-off|failed\s+write|failure|error|the\s+write\s+(?:when|if|after|that|and))\b` +
			`|\b(?:paraphrase|summari[sz]e|abbreviate|shorten|reword|condense)\b` + writeMemoryPart2Gap +
			`{0,30}\b(?:error|command|failure)\b|\b(?:write|handoff)\b` + writeMemoryPart2Gap +
			`{0,40}\bis\s+(?:dropped|skipped|ignored|abandoned)\b`),
		paraphrase:  "A write that keeps failing is dropped and the flow moves on.",
		imperative:  "Skip the write when it still fails.",
		conditional: "If the parent isn't waiting, summarize the error instead of quoting it.",
		arrow:       "A parent that isn't waiting → summarize the error instead of quoting it.",
		stated: "Still failing after retries → report the exact command and the CLI error verbatim. Never silently " +
			"skip a handed-off write.",
	}
	// writeMemoryPart2SpacedFlagScan finds an instruction to write a free-text flag with a space before its value.
	writeMemoryPart2SpacedFlagScan = writeMemoryPart2Scan{
		name: "a free-text flag written with a space",
		pattern: regexp.MustCompile(`(?i)--(?:situation|source|subject|predicate|object|behavior|impact|action|` +
			`done-when|body|red-flag|trigger|question|answer|certainty)[ ]+["<]|\b(?:put|add|leave|insert|type)\b` +
			writeMemoryPart2Gap + `{0,30}\b(?:a\s+)?space\b` + writeMemoryPart2Gap + `{0,40}(?:=|equals|flag\b)`),
		paraphrase:  "Leave a space between the flag and its quoted value.",
		imperative:  "Write --behavior \"<what was done>\" with a space before the quote.",
		conditional: "If the value doesn't begin with a dash, write --behavior \"<what was done>\" with a space.",
		arrow:       "A value that doesn't begin with a dash → write --behavior \"<what was done>\" with a space.",
		stated: "A free-text flag value (for example `--situation`, `--source`, `--subject`, `--behavior`) must be " +
			"written as `--flag=\"value\"` — the quote touching the `=`, no space — because a value beginning with " +
			"`-` is otherwise misread as a new flag and the command fails.",
	}
	// writeMemoryPart2StrippedCitationScan finds an instruction to strip or rewrite an inline citation.
	writeMemoryPart2StrippedCitationScan = writeMemoryPart2Scan{
		name: "stripping an inline citation",
		pattern: regexp.MustCompile(`(?i)\b(?:strip|remove|rewrite|replace|delete|drop|escape|convert|unlink)\b` +
			writeMemoryPart2Gap + `{0,40}(?:\[\[|\bwikilinks?\b|\bcitations?\b)|(?:\[\[|\bwikilinks?\b|\bcitations?\b)` +
			writeMemoryPart2Gap + `{0,40}\b(?:is|are|gets?|must\s+be|should\s+be)\s+(?:stripped|removed|rewritten|` +
			`replaced|escaped|dropped|converted)\b`),
		paraphrase:  "A citation in the body is escaped so it renders as plain text.",
		imperative:  "Strip the [[basename]] citation from the body before running.",
		conditional: "If the note isn't in the vault, remove the wikilink from the body.",
		arrow:       "A note that isn't in the vault → remove the wikilink from the body.",
		stated: "This does NOT restrict inline `[[basename]]` prose citations: hand-type one directly in `--body` or " +
			"`--object` whenever the parent's handoff content calls for citing a fact/feedback note",
	}
	// writeMemoryPart2TriggerEmissionScan finds an instruction to emit a trigger the handoff did not carry.
	writeMemoryPart2TriggerEmissionScan = writeMemoryPart2Scan{
		name: "emitting a trigger the handoff lacks",
		pattern: regexp.MustCompile(`(?i)--trigger=(?:""|'')|\b(?:emit|add|include|append|pass|write|supply)\b` +
			writeMemoryPart2Gap + `{0,40}\b(?:empty|default|placeholder|blank|generic|made-up)\b` + writeMemoryPart2Gap +
			`{0,30}\btriggers?\b|\btriggers?\b` + writeMemoryPart2Gap + `{0,30}\b(?:is|are)\s+(?:required|mandatory)\b` +
			`|\b(?:always|must)\s+(?:carry|have|include|emit|add)\b` + writeMemoryPart2Gap + `{0,20}\btriggers?\b` +
			`|\b(?:emit|add|include|append)\b` + writeMemoryPart2Gap + `{0,30}--trigger\b` + writeMemoryPart2Gap +
			`{0,30}\b(?:even|anyway|regardless|always)\b`),
		paraphrase:  "A trigger is required on every runbook.",
		imperative:  "Emit a default trigger such as the slug when the handoff has none.",
		conditional: "If the handoff carries no triggers, emit --trigger=\"\" anyway.",
		arrow:       "A handoff with no triggers → emit --trigger=\"\" anyway.",
		stated: "One `--trigger` per handed-off `triggers` entry, in order, placed after any `--red-flag`; omit " +
			"entirely when the handoff carries none.",
	}
	// writeMemoryPart2UnboundedRetryScan finds an instruction to retry more than twice.
	writeMemoryPart2UnboundedRetryScan = writeMemoryPart2Scan{
		name: "retrying more than twice",
		pattern: regexp.MustCompile(`(?i)\b(?:retry|retries|retrying|re-?run|try\s+again)\b` +
			writeMemoryPart2Gap + `{0,40}\b(?:until|indefinitely|forever|as\s+(?:often|long|many)|as\s+needed)\b` +
			`|\b(?:retry|re-?run|try\s+again)\b` + writeMemoryPart2Gap + `{0,30}\b(?:up\s+to|at\s+most|max(?:imum)?` +
			`(?:\s+of)?)\s+(?:[3-9]|\d\d|three|four|five|six|ten)\b` +
			`|\bmax(?:imum)?\s+(?:of\s+)?(?:[3-9]|\d\d|three|four|five|six|ten)\s+retr`),
		paraphrase:  "Keep retrying as often as needed until the command succeeds.",
		imperative:  "Retry until the command succeeds.",
		conditional: "If the error isn't clear, retry up to five times.",
		arrow:       "A CLI error that isn't clear → retry up to five times.",
		stated: "CLI error → read it, fix exactly the named problem (missing/typo'd flag, bad value), retry. " +
			"Max 2 retries.",
	}
)

// writeMemoryPart2Scan is one absence scan: a pattern for an instruction that contradicts a rule, a paraphrase and an
// imperative that contradict it, an imperative that stands after a negated condition in the comma form and one in the
// arrow form the skill itself uses, and the shipped wording that must pass.
type writeMemoryPart2Scan struct {
	name        string
	pattern     *regexp.Regexp
	paraphrase  string
	imperative  string
	conditional string
	arrow       string
	stated      string
}

// writeMemoryPart2Before reports whether both fragments occur in the text and the first occurs before the second.
func writeMemoryPart2Before(text, first, second string) bool {
	firstAt := strings.Index(text, first)
	secondAt := strings.Index(text, second)

	return firstAt >= 0 && secondAt >= 0 && firstAt < secondAt
}

// writeMemoryPart2Between returns the text from the start fragment up to the end fragment, and fails the test when
// either is missing or out of order.
func writeMemoryPart2Between(t *testing.T, text, start, end string) string {
	t.Helper()

	startAt := strings.Index(text, start)
	if startAt < 0 {
		t.Fatalf("the text has no %q", start)

		return ""
	}

	endAt := strings.Index(text[startAt:], end)
	if endAt < 0 {
		t.Fatalf("the text has no %q after %q", end, start)

		return ""
	}

	return text[startAt : startAt+endAt]
}

// writeMemoryPart2Blocks returns the text of a shipped file with the quote markers of its blockquotes removed, the
// lines of one block (a paragraph, a list item, a table row or a heading) joined by single spaces and the blocks
// separated by a line break.
func writeMemoryPart2Blocks(content string) string {
	var builder strings.Builder

	separator := "\n"

	for raw := range strings.SplitSeq(content, "\n") {
		line := strings.Join(strings.Fields(strings.TrimLeft(strings.TrimSpace(raw), "> ")), " ")
		if line == "" {
			separator = "\n"

			continue
		}

		if writeMemoryPart2BlockStart.MatchString(line) {
			separator = "\n"
		}

		if builder.Len() > 0 {
			builder.WriteString(separator)
		}

		builder.WriteString(line)

		separator = " "
		if strings.HasPrefix(line, "#") {
			separator = "\n"
		}
	}

	return builder.String()
}

// writeMemoryPart2ClauseStart returns the offset in the lead where the clause that holds the end of the lead begins:
// after the last clause border.
func writeMemoryPart2ClauseStart(lead string) int {
	start := 0

	for _, border := range writeMemoryPart2ClauseBorders {
		if at := strings.LastIndex(lead, border); at >= 0 {
			start = max(start, at+len(border))
		}
	}

	return start
}

// writeMemoryPart2Contradictions returns, by path, the matches of the scan in every shipped SKILL.md, references file
// and guidance document that no negation in the matched clause excuses. It fails the test when the scan did not cover
// the write-memory skill, so an empty result is not vacuous.
func writeMemoryPart2Contradictions(t *testing.T, scan writeMemoryPart2Scan) map[string][]string {
	t.Helper()

	found := make(map[string][]string)
	scannedSkill := false

	for _, path := range aibInstructionPaths(t) {
		if path == writeMemoryPart2SkillPath {
			scannedSkill = true
		}

		content, err := os.ReadFile(filepath.Join(repositoryRoot, path))
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)

			return nil
		}

		if hits := writeMemoryPart2Hits(string(content), scan.pattern); len(hits) > 0 {
			found[path] = hits
		}
	}

	if !scannedSkill {
		t.Fatalf("the instruction scan did not cover %s", writeMemoryPart2SkillPath)
	}

	return found
}

// writeMemoryPart2Hits returns every match of the pattern in the content of a shipped file, except a match whose
// clause, from the last clause border before the match to the end of the match, carries a negation: a prohibition is
// not an instruction, and a negation in another clause of the sentence, such as a negated condition, excuses nothing.
func writeMemoryPart2Hits(content string, pattern *regexp.Regexp) []string {
	text := writeMemoryPart2Blocks(content)
	hits := make([]string, 0)

	for _, location := range pattern.FindAllStringIndex(text, -1) {
		clause := text[writeMemoryPart2ClauseStart(text[:location[0]]):location[1]]
		if writeMemoryPart2Negation.MatchString(clause) {
			continue
		}

		hits = append(hits, text[location[0]:location[1]])
	}

	return hits
}

// writeMemoryPart2Scans returns every absence scan of this file.
func writeMemoryPart2Scans() []writeMemoryPart2Scan {
	return []writeMemoryPart2Scan{
		writeMemoryPart2AskEveryTriggerScan, writeMemoryPart2BeyondNamedScan, writeMemoryPart2CommonWordScan,
		writeMemoryPart2MixedFlagsScan, writeMemoryPart2OwnTagScan, writeMemoryPart2QATagScan,
		writeMemoryPart2SkippedFailureScan, writeMemoryPart2SpacedFlagScan, writeMemoryPart2StrippedCitationScan,
		writeMemoryPart2TriggerEmissionScan, writeMemoryPart2UnboundedRetryScan,
	}
}
