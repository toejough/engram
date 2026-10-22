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

// Test792a18n1a_Please_AReviewerRecallsFirst reads the please skill the repository ships: a dispatched reviewer
// recalls first, with phrases drawn from the artifact and its angle.
func Test792a18n1a_Please_AReviewerRecallsFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("The reviewer recalls first (the `route` skill's recall-first rule), " +
		"with phrases drawn from the artifact and its angle"))
	g.Expect(text).To(ContainSubstring("You dispatched a reviewer without `/recall` as its first action | " +
		"Reviewers recall first"))
	g.Expect(pleasePart3Before(text, "Reviewer protocol:", "The reviewer recalls first")).To(BeTrue())
}

// Test792a18n1b_Please_EveryFindingIsFixedOrRebutted reads the please skill the repository ships: the author fixes or
// rebuts every finding, the reviewer ACKs or counters, and no instruction lets a finding be dropped.
func Test792a18n1b_Please_EveryFindingIsFixedOrRebutted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("The author addresses every finding: fix it, or rebut with reasons."))
	g.Expect(text).To(ContainSubstring("The reviewer ACKs or counters."))
	g.Expect(text).To(ContainSubstring("Keep the reviewer alive until its gate closes."))
	g.Expect(text).To(ContainSubstring("the gated step is not `completed` until every finding is resolved."))
	g.Expect(text).To(ContainSubstring("You resolved a finding by silently dropping it | " +
		"Every finding is fixed or rebutted to reviewer ACK"))
	g.Expect(pleasePart3Offenders(t, pleasePart3DropFindingPattern, nil)).To(BeEmpty())
}

// Test792a18n1c_Please_ADeadlockIsEscalated reads the please skill the repository ships: after about two unresolved
// rounds please stops, summarizes both positions and escalates through AskUserQuestion.
func Test792a18n1c_Please_ADeadlockIsEscalated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("After ~2 unresolved rounds, stop:"))
	g.Expect(text).To(ContainSubstring("summarize both positions"))
	g.Expect(text).To(ContainSubstring("escalate via `AskUserQuestion`."))
	g.Expect(text).To(ContainSubstring("You argued past ~2 rounds without escalating | " +
		"Stop, summarize both positions, ask the user."))
}

// Test792a18n1d_Please_AnAbsentSubjectIsNamed reads the please skill the repository ships: an angle is not
// applicable only when its subject is absent, and the missing subject is named out loud.
func Test792a18n1d_Please_AnAbsentSubjectIsNamed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("| A | end of step 3, before any execution | the committed plan/spec | " +
		"ask-alignment (sonnet); code-alignment (sonnet); docs/diagrams-alignment (haiku);"))
	g.Expect(text).To(ContainSubstring("An angle is N/A only when its subject is absent from the environment " +
		"(no diagrams in the repo, no docs touched)"))
	g.Expect(text).To(ContainSubstring("name the missing subject out loud."))
}

// Test792a18o1a_Please_AMeasuredClaimHasAPointer reads the please skill the repository ships: a measured claim in an
// escalation carries the file or command that produced it and a one-line statement of how it was verified.
func Test792a18o1a_Please_AMeasuredClaimHasAPointer(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("Any MEASURED claim (a count, rate, cost, duration) in a mid-cycle " +
		"escalation — an `AskUserQuestion` or a STOP report —"))
	g.Expect(text).To(ContainSubstring("carries its evidence pointer (the file or command that produced it)"))
	g.Expect(text).To(ContainSubstring("a one-line validity statement (\"verified how?\")"))
	g.Expect(text).To(ContainSubstring("You're about to ship a measured claim without an evidence pointer + " +
		"\"verified how?\" line"))
}

// Test792a18o1b_Please_AnUnverifiedFigureIsLabeled reads the please skill the repository ships: a claim whose honest
// validity line is "not verified" is verified first or presented as an unverified hypothesis, and does not ship as a
// finding.
func Test792a18o1b_Please_AnUnverifiedFigureIsLabeled(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("A claim whose honest validity line is \"not verified\" does not ship as a " +
		"finding"))
	g.Expect(text).To(ContainSubstring("verify it first, or present it explicitly as an unverified hypothesis."))
	g.Expect(text).To(ContainSubstring("Escalation provenance — verify it, or label it an unverified hypothesis"))
	g.Expect(pleasePart3Offenders(t, pleasePart3ShipUnverifiedPattern, nil)).To(BeEmpty())
}

// Test792a18p1a_Please_TheCorpusIsEnumerated reads the please skill the repository ships: the lessons audit before the
// closing learn lists every fired stop, every gate FAIL, every CORRECTION-class commit and every escalation.
func Test792a18p1a_Please_TheCorpusIsEnumerated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("Before invoking the closing `/learn`, run the **lessons audit** over the " +
		"cycle's mechanical corpus:"))
	g.Expect(text).To(ContainSubstring("every pre-registered STOP that fired"))
	g.Expect(text).To(ContainSubstring("every gate FAIL verdict"))
	g.Expect(text).To(ContainSubstring("every commit whose message contains CORRECTION, supersede/superseded, " +
		"instrument-invalid, or redraw/redrawn"))
	g.Expect(text).To(ContainSubstring("every mid-cycle escalation to the user"))
	g.Expect(text).To(ContainSubstring("You're closing the cycle without the step-7 lessons audit | " +
		"Enumerate STOPs, gate FAILs, CORRECTION-class commits, escalations"))
}

// Test792a18p1b_Please_EachItemMapsOrSaysWhyNot reads the please skill the repository ships: the audit maps each item
// to the vault note that captures its lesson or writes the one-line "no lesson: <why>".
func Test792a18p1b_Please_EachItemMapsOrSaysWhyNot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("Map each item to the vault note that captures its lesson"))
	g.Expect(text).To(ContainSubstring("or write the one-line \"no lesson: <why>\"."))
	g.Expect(text).To(ContainSubstring("map each to a note or a \"no lesson: why\" line"))
}

// Test792a18p1c_Please_AnUnmappedItemBecomesAReversalHandoff reads the please skill the repository ships: an unmapped
// item is a reversal handoff for the closing learn.
func Test792a18p1c_Please_AnUnmappedItemBecomesAReversalHandoff(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("Unmapped items are reversal handoffs for the closing learn (its Step-2 kind 3)."))
	g.Expect(pleasePart3Before(text, "Unmapped items are reversal handoffs", "Run the `learn` skill again")).To(BeTrue())
}

// Test792a18p1d_Please_TheListIsInTheClosingReport reads the please skill the repository ships: the audit list, item
// to note-or-no-lesson, appears in the cycle's closing report to the user.
func Test792a18p1d_Please_TheListIsInTheClosingReport(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("The audit list (item → note-or-no-lesson) appears in the cycle's closing " +
		"report to the user."))
}

// Test792a18q1a_Please_AnExistingNoteIsSearchedForFirst reads the please skill the repository ships: for each lesson
// about to be captured please asks which existing artifact should have surfaced it and searches the vault for a note
// that already covers the situation.
func Test792a18q1a_Please_AnExistingNoteIsSearchedForFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("For each lesson this cycle is about to capture, ask which artifact that " +
		"already existed should have surfaced it first."))
	g.Expect(text).To(ContainSubstring("Before writing a new note, search the vault for one that already covers " +
		"the situation."))
}

// Test792a18q1b_Please_AStaleSituationLineIsReworded reads the please skill the repository ships: a note that existed
// but did not surface has its situation line checked and reworded, which beats writing a second note on the same
// lesson, and no instruction writes the second note.
func Test792a18q1b_Please_AStaleSituationLineIsReworded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("If a note existed but did not surface at the moment it was needed, check " +
		"whether its `situation:` line matches how the moment actually presented;"))
	g.Expect(text).To(ContainSubstring("if it doesn't, reword the note"))
	g.Expect(text).To(ContainSubstring("that beats writing a second note on the same lesson."))
	g.Expect(pleasePart3Offenders(t, pleasePart3SecondNotePattern, nil)).To(BeEmpty())
}

// Test792a18q1c_Please_ARepositoryChangeIsALastResort reads the please skill the repository ships: a change that
// ships from the repository is proposed only as a last resort and never filed automatically.
func Test792a18q1c_Please_ARepositoryChangeIsALastResort(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("anything that ships from this repo instead — code, skills, guidance — " +
		"makes the user wait on a release, so propose that only as a last resort,"))
	g.Expect(text).To(ContainSubstring("and never file it automatically."))
	g.Expect(pleasePart3Offenders(t, pleasePart3AutoFilePattern, nil)).To(BeEmpty())
}

// Test792a18r1a_Please_TheClosingLearnReceivesBothInputs reads the please skill the repository ships: the closing
// learn is run again after the audit and is handed the full list of LESSONS lines, none entries included, and the
// audit's map.
func Test792a18r1a_Please_TheClosingLearnReceivesBothInputs(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("Run the `learn` skill again to preserve the lessons from this session,"))
	g.Expect(text).To(ContainSubstring("handing it the full list of `LESSONS:` lines collected across step 4's " +
		"dispatches"))
	g.Expect(text).To(ContainSubstring("(including any `none` entries)"))
	g.Expect(text).To(ContainSubstring("alongside the lessons-audit map above"))
	g.Expect(text).To(ContainSubstring("step 7 drops no line."))
	g.Expect(pleasePart3Before(text, "run the **lessons audit**", "Run the `learn` skill again")).To(BeTrue())
}

// Test792a18r1b_Please_TheQuestionAndAnswerCaptureIsLearns reads the please skill the repository ships: learn's Step
// 2.5 handles the capture of a substantive answered question, please does not duplicate it, and no instruction of
// please writes the pair.
func Test792a18r1b_Please_TheQuestionAndAnswerCaptureIsLearns(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("The learn skill's Step 2.5 handles ad-hoc QA pair capture for substantive " +
		"answered questions from this session"))
	g.Expect(text).To(ContainSubstring("**do not duplicate that logic here**."))
	g.Expect(pleasePart3Offenders(t, pleasePart3WriteQAPairPattern, pleasePart3QACaptureOwners)).To(BeEmpty())
}

// Test792a18s1a_Please_ReviewsAndBuildingAreDispatched reads the please skill the repository ships: object-level work
// and gate reviews are delegated to subagents per route, one fresh reviewer per angle routed, and the orchestrator
// routes, decomposes and synthesizes.
func Test792a18s1a_Please_ReviewsAndBuildingAreDispatched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart3SkillPath)

	g.Expect(text).To(ContainSubstring("Object-level work and gate reviews are delegated to subagents per the " +
		"`route` skill"))
	g.Expect(text).To(ContainSubstring("the orchestrator routes, decomposes, and synthesizes; it does not do the " +
		"object-level work itself."))
	g.Expect(text).To(ContainSubstring("Each gate fans out ONE fresh-context reviewer subagent PER ANGLE"))
	g.Expect(text).To(ContainSubstring("apply the `route` rubric per angle to confirm or adjust the default."))
}

// TestPleasePart3Offenders shows the scanner of Test792a18n1b, Test792a18o1b, Test792a18q1b, Test792a18q1c and
// Test792a18r1b finds an instruction that contradicts a rule, passes the shipped wording that states the rule, and
// skips a file that is exempt.
func TestPleasePart3Offenders(t *testing.T) {
	t.Parallel()

	t.Run("an instruction that contradicts a rule is found", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		cases := map[*regexp.Regexp][]string{
			pleasePart3DropFindingPattern: {
				"A minor finding may be dropped silently.",
				"Drop any finding the author disagrees with.",
			},
			pleasePart3ShipUnverifiedPattern: {
				"An unverified figure may still ship as a finding.",
				"Report an unverified number as a finding.",
			},
			pleasePart3SecondNotePattern: {
				"Then write a second note on the same lesson.",
				"Create another note about the same lesson.",
			},
			pleasePart3AutoFilePattern: {
				"File it automatically as an issue.",
				"Automatically file an issue for the lesson.",
			},
			pleasePart3WriteQAPairPattern: {
				"At close, write the QA pair for each substantive answered question yourself.",
				"Capture the Q&A pair at step 7.",
			},
		}

		for pattern, sentences := range cases {
			for _, sentence := range sentences {
				g.Expect(pleasePart3Hits(pattern, sentence)).NotTo(BeEmpty(), sentence)
			}
		}
	})

	t.Run("the shipped wording that states a rule is clean", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		text := "You resolved a finding by silently dropping it | Every finding is fixed or rebutted. " +
			"A claim whose honest validity line is \"not verified\" does not ship as a finding. " +
			"that beats writing a second note on the same lesson. so propose that only as a last resort, " +
			"and never file it automatically. The learn skill's Step 2.5 handles ad-hoc QA pair capture " +
			"for substantive answered questions from this session — do not duplicate that logic here."

		for _, pattern := range []*regexp.Regexp{
			pleasePart3DropFindingPattern, pleasePart3ShipUnverifiedPattern, pleasePart3SecondNotePattern,
			pleasePart3AutoFilePattern, pleasePart3WriteQAPairPattern,
		} {
			g.Expect(pleasePart3Hits(pattern, text)).To(BeEmpty(), pattern.String())
		}
	})

	t.Run("an exempt file is not scanned", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		owners := pleasePart3QACaptureOwners
		pattern := pleasePart3Pattern(`--question=`)

		g.Expect(slices.Contains(owners, pleasePart3SkillPath)).To(BeFalse())
		g.Expect(pleasePart3Offenders(t, pattern, nil)).NotTo(BeEmpty())
		g.Expect(pleasePart3Offenders(t, pattern, owners)).To(BeEmpty())
	})
}

// unexported constants.
const (
	pleasePart3NegationPrefix = `(\b(?:never|not|do not|must not|cannot|don't)\s+)?`
	pleasePart3SkillPath      = "agent-instructions/skills/please/SKILL.md"
)

// unexported variables.
var (
	// pleasePart3AutoFilePattern finds an instruction to file or open an issue or ticket without asking.
	pleasePart3AutoFilePattern = pleasePart3Pattern(`(?:file|open|create)\s+(?:it|an?\s+(?:issue|ticket)|the\s+` +
		`(?:issue|ticket))\s+automatically|automatically\s+(?:file|open|create)\s+(?:it|an?\s+(?:issue|ticket)|` +
		`the\s+(?:issue|ticket))`)
	// pleasePart3DropFindingPattern finds an instruction that lets a finding be dropped, ignored or skipped.
	pleasePart3DropFindingPattern = pleasePart3Pattern(`(?:drop|ignore|discard|skip)\s+(?:a|an|any|the|each|every)` +
		`\s+(?:\w+\s+)?findings?\b|findings?\s+(?:may|can|could)\s+(?:be\s+)?(?:silently\s+)?` +
		`(?:dropped|ignored|skipped|discarded)\b`)
	// pleasePart3QACaptureOwners is the instruction files that own the capture of a question-and-answer pair: learn
	// performs it, write-memory composes it and recall's write side hands it off.
	pleasePart3QACaptureOwners = []string{
		"agent-instructions/skills/learn/SKILL.md",
		"agent-instructions/skills/recall/SKILL.md",
		"agent-instructions/skills/write-memory/SKILL.md",
	}
	// pleasePart3SecondNotePattern finds an instruction to write a second note on a lesson a note already covers.
	pleasePart3SecondNotePattern = pleasePart3Pattern(`(?:write|create|add)\s+(?:a\s+(?:second|duplicate|` +
		`parallel|new)|another)\s+note\s+(?:on|for|about)\s+the\s+same`)
	// pleasePart3ShipUnverifiedPattern finds an instruction that lets an unverified claim ship as a finding.
	pleasePart3ShipUnverifiedPattern = pleasePart3Pattern(`unverified\s+(?:claims?|figures?|numbers?)\s+` +
		`(?:may|can)\s+(?:still\s+)?ship\b|(?:ship|report|present)\s+(?:an?\s+)?(?:unverified|unmeasured)\s+` +
		`(?:claims?|figures?|numbers?)\s+as\s+(?:a\s+)?findings?`)
	// pleasePart3WriteQAPairPattern finds an instruction to write or capture a question-and-answer pair.
	pleasePart3WriteQAPairPattern = pleasePart3Pattern(`(?:write|capture|record|save|store)\s+(?:the|each|every|a|an)` +
		`\s+(?:(?:substantive|answered|ad-hoc)\s+)*(?:question[- ]and[- ]answer|q&a|qa)\b`)
)

// pleasePart3Before reports whether both fragments occur in the text and the first occurs before the second.
func pleasePart3Before(text, first, second string) bool {
	firstAt := strings.Index(text, first)
	secondAt := strings.Index(text, second)

	return firstAt >= 0 && secondAt >= 0 && firstAt < secondAt
}

// pleasePart3Hits returns every match of the pattern in the text that is not negated by the word before it.
func pleasePart3Hits(pattern *regexp.Regexp, text string) []string {
	hits := make([]string, 0)

	for _, match := range pattern.FindAllStringSubmatch(strings.Join(strings.Fields(text), " "), -1) {
		if match[1] == "" {
			hits = append(hits, match[0])
		}
	}

	return hits
}

// pleasePart3Offenders returns "path: match" for every instruction in every shipped SKILL.md, references markdown file
// and guidance document that the pattern finds, except in the exempt files.
func pleasePart3Offenders(t *testing.T, pattern *regexp.Regexp, exempt []string) []string {
	t.Helper()

	offenders := make([]string, 0)

	for _, relative := range aibInstructionPaths(t) {
		if slices.Contains(exempt, relative) {
			continue
		}

		content, err := os.ReadFile(filepath.Join(repositoryRoot, relative))
		if err != nil {
			t.Fatalf("reading %s: %v", relative, err)
		}

		for _, hit := range pleasePart3Hits(pattern, string(content)) {
			offenders = append(offenders, relative+": "+hit)
		}
	}

	return offenders
}

// pleasePart3Pattern compiles an instruction pattern, case-insensitive, with an optional negation word captured before
// it as group 1.
func pleasePart3Pattern(body string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)` + pleasePart3NegationPrefix + `(?:` + body + `)`)
}
