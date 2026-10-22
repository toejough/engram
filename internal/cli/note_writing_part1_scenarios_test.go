package cli_test

// Scenario-ID tests of the Note Writing product capability, part 1 (docs/specs/product/note-writing.md): 792a3a Fact
// notes, 792a3b Feedback notes, 792a3c Runbook notes, 792a3d Runbook red flags and triggers, 792a3e No task-type or
// input fields, 792a3f Required content, 792a3g Argument validation, 792a3h Optional metadata, 792a3i Tier field,
// 792a3j Luhmann placement and 792a3k Placement refusals.
//
// Every scenario drives the real command path (cli.Targets over the production compositions) over the in-memory vault
// of the controlled-vocabulary scenarios (cv1Vault: a cli.EdgeFS, a fixed clock and a deterministic embedder), so no
// scenario touches the real filesystem, home, git or embedding model. A vault "whose notes have IDs ..." is built by
// engram learn itself, so every note is in the shape the system produces. Helpers carry the prefix nw1.

import (
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
)

// Test792a3a1b_LearnFact_ALeadingWhenIsNotDoubled verifies 792a3a1b (note-writing.md:43): in an empty vault, engram
// learn fact with --situation "When testing" writes a note whose body reads "Information learned: when in testing,
// s p o."
func Test792a3a1b_LearnFact_ALeadingWhenIsNotDoubled(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an empty vault
	vault := newCV1Vault()

	// When the user runs engram learn fact with --situation "When testing" --subject s --predicate p --object o
	flags := nw1FactFlags().set("--situation", "When testing").set("--subject", "s").set("--predicate", "p").
		set("--object", "o")
	result := nw1Learn(vault, "fact", flags)
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note's body reads "Information learned: when in testing, s p o."
	note := nw1ReadPrinted(g, vault, result)
	g.Expect(strings.TrimSpace(note.body)).To(Equal("Information learned: when in testing, s p o."))
}

// Test792a3b1b_LearnFeedback_ALeadingWhenIsNotDoubled verifies 792a3b1b (note-writing.md:64): in an empty vault,
// engram learn feedback with --situation "When x" --action a writes a note whose body reads "Lesson learned: when x,
// a."
func Test792a3b1b_LearnFeedback_ALeadingWhenIsNotDoubled(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an empty vault
	vault := newCV1Vault()

	// When the user runs engram learn feedback with --situation "When x" --action a
	result := nw1Learn(vault, "feedback", nw1FeedbackFlags().set("--situation", "When x").set("--action", "a"))
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note's body reads "Lesson learned: when x, a."
	note := nw1ReadPrinted(g, vault, result)
	g.Expect(strings.TrimSpace(note.body)).To(Equal("Lesson learned: when x, a."))
}

// Test792a3c1a_LearnRunbook_ARunbookNoteIsWritten verifies 792a3c1a (note-writing.md:88): in an empty vault, engram
// learn runbook prints the path of 1.<today>.rb.md, whose frontmatter has type: runbook, situation and done_when and
// whose body is "1. Run the checks".
func Test792a3c1a_LearnRunbook_ARunbookNoteIsWritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an empty vault
	vault := newCV1Vault()

	// When the user runs engram learn runbook --slug rb --position top --source src --situation "releasing a build"
	// --done-when "the tag is pushed" --body "1. Run the checks"
	flags := nw1Flags{
		"--slug", "rb", "--position", "top", "--source", "src", "--situation", "releasing a build",
		"--done-when", "the tag is pushed", "--body", "1. Run the checks",
	}
	result := nw1Learn(vault, "runbook", flags)
	g.Expect(result.stderr).To(BeEmpty())

	// Then stdout prints the path of 1.<today>.rb.md, whose frontmatter has type: runbook, situation and done_when,
	// and whose body is "1. Run the checks"
	g.Expect(strings.TrimSpace(result.stdout)).To(Equal(cv1VaultPath + "/1." + cv1Today + ".rb.md"))

	note := nw1ReadPrinted(g, vault, result)
	g.Expect(note.front.Type).To(Equal("runbook"))
	g.Expect(note.front.Situation).To(Equal("releasing a build"))
	g.Expect(note.front.DoneWhen).To(Equal("the tag is pushed"))
	g.Expect(strings.TrimSpace(note.body)).To(Equal("1. Run the checks"))
}

// Test792a3d1a_LearnRunbook_RedFlagsAreKeptInOrder verifies 792a3d1a (note-writing.md:108): engram learn runbook with
// --red-flag "A" --red-flag "B" writes a red_flags: list of A then B.
func Test792a3d1a_LearnRunbook_RedFlagsAreKeptInOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an empty vault
	vault := newCV1Vault()

	// When the user runs engram learn runbook with --red-flag "A" --red-flag "B"
	result := nw1Learn(vault, "runbook", nw1RunbookFlags().add("--red-flag", "A").add("--red-flag", "B"))
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note's red_flags: lists A then B
	note := nw1ReadPrinted(g, vault, result)
	g.Expect(note.front.RedFlags).To(Equal([]string{"A", "B"}))
}

// Test792a3d1b_LearnRunbook_TriggersFollowTheRedFlagsInOrder verifies 792a3d1b (note-writing.md:113): engram learn
// runbook with --red-flag "A" --trigger "cue one" --trigger "cue two" writes a triggers: list of "cue one" then "cue
// two", after red_flags:.
func Test792a3d1b_LearnRunbook_TriggersFollowTheRedFlagsInOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an empty vault
	vault := newCV1Vault()

	// When the user runs engram learn runbook with --red-flag "A" --trigger "cue one" --trigger "cue two"
	flags := nw1RunbookFlags().add("--red-flag", "A").add("--trigger", "cue one").add("--trigger", "cue two")
	result := nw1Learn(vault, "runbook", flags)
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note's triggers: lists "cue one" then "cue two", after red_flags:
	note := nw1ReadPrinted(g, vault, result)
	g.Expect(note.front.Triggers).To(Equal([]string{"cue one", "cue two"}))

	redFlagsAt := strings.Index(note.raw, "\nred_flags:")
	triggersAt := strings.Index(note.raw, "\ntriggers:")
	g.Expect(redFlagsAt).To(BeNumerically(">=", 0))
	g.Expect(triggersAt).To(BeNumerically(">", redFlagsAt))
}

// Test792a3d1c_LearnRunbook_NeitherFlagMeansNeitherField verifies 792a3d1c (note-writing.md:118): engram learn runbook
// with no --red-flag and no --trigger writes a note with no red_flags: and no triggers: field.
func Test792a3d1c_LearnRunbook_NeitherFlagMeansNeitherField(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an empty vault
	vault := newCV1Vault()

	// When the user runs engram learn runbook with no --red-flag and no --trigger
	result := nw1Learn(vault, "runbook", nw1RunbookFlags())
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note is written with no red_flags: and no triggers: field
	note := nw1ReadPrinted(g, vault, result)
	g.Expect(note.frontKeys()).NotTo(HaveKey("red_flags"))
	g.Expect(note.frontKeys()).NotTo(HaveKey("triggers"))
}

// Test792a3e1a_LearnRunbook_ATaskTypeIsRefused verifies 792a3e1a (note-writing.md:131): in an existing vault, engram
// learn runbook with --task-type foo fails naming --task-type as an undefined flag, and no note is written.
func Test792a3e1a_LearnRunbook_ATaskTypeIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an existing vault
	vault := newCV1Vault()
	before := vault.fsys.snapshot()

	// When the user runs engram learn runbook with --task-type foo
	result := nw1Learn(vault, "runbook", nw1RunbookFlags().add("--task-type", "foo"))

	// Then it fails naming --task-type as an undefined flag, and no note is written
	g.Expect(result.stderr).To(ContainSubstring("not defined: --task-type"))
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// Test792a3f1a_LearnFact_AMissingSituationIsRefused verifies 792a3f1a (note-writing.md:149): in an existing vault,
// engram learn fact without --situation fails naming --situation, and no note is written.
func Test792a3f1a_LearnFact_AMissingSituationIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an existing vault
	vault := newCV1Vault()
	before := vault.fsys.snapshot()

	// When the user runs engram learn fact without --situation
	result := nw1Learn(vault, "fact", nw1FactFlags().without("--situation"))

	// Then it fails naming --situation, and no note is written
	g.Expect(result.stderr).To(ContainSubstring("--situation"))
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// Test792a3f1b_LearnFeedback_ABlankSituationIsRefused verifies 792a3f1b (note-writing.md:154): in an existing vault,
// engram learn feedback with --situation "  " fails saying --situation is required, and no note is written.
func Test792a3f1b_LearnFeedback_ABlankSituationIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an existing vault
	vault := newCV1Vault()
	before := vault.fsys.snapshot()

	// When the user runs engram learn feedback with --situation "  "
	result := nw1Learn(vault, "feedback", nw1FeedbackFlags().set("--situation", "  "))

	// Then it fails saying --situation is required, and no note is written
	g.Expect(result.stderr).To(ContainSubstring("--situation is required"))
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// Test792a3f1c_LearnRunbook_ARunbookWithoutDoneWhenIsRefused verifies 792a3f1c (note-writing.md:159): in an existing
// vault, engram learn runbook without --done-when fails naming --done-when, and no note is written.
func Test792a3f1c_LearnRunbook_ARunbookWithoutDoneWhenIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an existing vault
	vault := newCV1Vault()
	before := vault.fsys.snapshot()

	// When the user runs engram learn runbook without --done-when
	result := nw1Learn(vault, "runbook", nw1RunbookFlags().without("--done-when"))

	// Then it fails naming --done-when, and no note is written
	g.Expect(result.stderr).To(ContainSubstring("--done-when"))
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// Test792a3g1a_Learn_AnInvalidArgumentIsRefusedBeforeAnyWrite verifies the nine examples of 792a3g1a
// (note-writing.md:177): each invalid argument of engram learn fails with an error saying the problem of its row, and
// no note is written. The error text of a row is matched on the words of its problem column.
func Test792a3g1a_Learn_AnInvalidArgumentIsRefusedBeforeAnyWrite(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		kind    string
		flags   nw1Flags
		problem []string
	}{
		"fact with no --source": {
			kind: "fact", flags: nw1FactFlags().without("--source"),
			problem: []string{"--source", "required"},
		},
		"fact with no --slug": {
			kind: "fact", flags: nw1FactFlags().without("--slug"),
			problem: []string{"slug is required"},
		},
		"fact with --slug X_y": {
			kind: "fact", flags: nw1FactFlags().set("--slug", "X_y"),
			problem: []string{"slug must match [a-z0-9-]+"},
		},
		"fact with --project A": {
			kind: "fact", flags: nw1FactFlags().add("--project", "A"),
			problem: []string{"project slug must match [a-z0-9-]+"},
		},
		"fact with --issue \"a b\"": {
			kind: "fact", flags: nw1FactFlags().add("--issue", "a b"),
			problem: []string{"issue must be non-empty with no whitespace"},
		},
		"fact with --tag \"A B\"": {
			kind: "fact", flags: nw1FactFlags().add("--tag", "A B"),
			problem: []string{"tag must be <family> or <family>/<value>"},
		},
		"runbook with --trigger ab": {
			kind: "runbook", flags: nw1RunbookFlags().add("--trigger", "ab"),
			problem: []string{"trigger must be at least 3 characters"},
		},
		"fact with --tier L9": {
			kind: "fact", flags: nw1FactFlags().add("--tier", "L9"),
			problem: []string{"tier must be L2", "L1", "L3"},
		},
		"fact with --supersedes \"a|bogus|c\"": {
			kind: "fact", flags: nw1FactFlags().add("--supersedes", "a|bogus|c"),
			problem: []string{"type must be", "updates", "narrows", "refutes"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given an existing vault
			vault := newCV1Vault()
			before := vault.fsys.snapshot()

			// When the user runs engram learn <kind> with <argument>
			result := nw1Learn(vault, tc.kind, tc.flags)

			// Then it fails with an error saying <problem>, and no note is written
			for _, word := range tc.problem {
				g.Expect(result.stderr).To(ContainSubstring(word))
			}

			g.Expect(vault.fsys.snapshot()).To(Equal(before))
		})
	}
}

// Test792a3h1a_LearnRunbook_OptionalMetadataIsWritten verifies 792a3h1a (note-writing.md:210): engram learn runbook
// with --project proj --issue 12 --tag fam/val --chunk-source "a#b" --supersedes "1.x|updates|claim" writes project:
// proj, issue: "12", tags: [fam/val], sources: [a#b], a supersedes: entry for 1.x of type updates, and the body line
// "Supersedes: [[1.x]] — updates: claim".
func Test792a3h1a_LearnRunbook_OptionalMetadataIsWritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an empty vault
	vault := newCV1Vault()

	// When the user runs engram learn runbook with --project proj --issue 12 --tag fam/val --chunk-source "a#b"
	// --supersedes "1.x|updates|claim"
	flags := nw1RunbookFlags().add("--project", "proj").add("--issue", "12").add("--tag", "fam/val").
		add("--chunk-source", "a#b").add("--supersedes", "1.x|updates|claim")
	result := nw1Learn(vault, "runbook", flags)
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note carries project: proj, issue: "12", tags: [fam/val], sources: [a#b], a supersedes: entry for 1.x of
	// type updates, and the body line "Supersedes: [[1.x]] — updates: claim"
	note := nw1ReadPrinted(g, vault, result)
	g.Expect(note.front.Project).To(Equal("proj"))
	g.Expect(note.raw).To(MatchRegexp(`(?m)^issue: "12"$`))
	g.Expect(note.front.Tags).To(Equal([]string{"fam/val"}))
	g.Expect(note.front.Sources).To(Equal([]string{"a#b"}))
	g.Expect(note.front.Supersedes).To(Equal([]nw1Supersede{{Note: "1.x", Type: "updates", Claim: "claim"}}))
	g.Expect(note.body).To(ContainSubstring("\nSupersedes: [[1.x]] — updates: claim\n"))
}

// Test792a3h1b_LearnFact_AbsentMetadataWritesNoField verifies 792a3h1b (note-writing.md:215): engram learn fact with
// none of those flags writes a note with no project:, issue:, tags:, sources: or supersedes: field.
func Test792a3h1b_LearnFact_AbsentMetadataWritesNoField(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an empty vault
	vault := newCV1Vault()

	// When the user runs engram learn fact with none of those flags
	result := nw1Learn(vault, "fact", nw1FactFlags())
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note has no project:, issue:, tags:, sources: or supersedes: field
	note := nw1ReadPrinted(g, vault, result)

	for _, field := range []string{"project", "issue", "tags", "sources", "supersedes"} {
		g.Expect(note.frontKeys()).NotTo(HaveKey(field))
	}

	g.Expect(note.body).NotTo(ContainSubstring("Supersedes:"))
}

// Test792a3i1a_Learn_TheTierDefaultsToL2 verifies 792a3i1a (note-writing.md:231): engram learn fact or feedback without
// --tier writes tier: L2.
func Test792a3i1a_Learn_TheTierDefaultsToL2(t *testing.T) {
	t.Parallel()

	cases := map[string]nw1Flags{
		"fact":     nw1FactFlags(),
		"feedback": nw1FeedbackFlags(),
	}

	for kind, flags := range cases {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given an empty vault
			vault := newCV1Vault()

			// When the user runs engram learn fact or feedback without --tier
			result := nw1Learn(vault, kind, flags)
			g.Expect(result.stderr).To(BeEmpty())

			// Then the note carries tier: L2
			note := nw1ReadPrinted(g, vault, result)
			g.Expect(note.front.Tier).To(Equal("L2"))
		})
	}
}

// Test792a3i1b_LearnFact_ALegacyTierIsWrittenAsGiven verifies 792a3i1b (note-writing.md:236): engram learn fact with
// --tier L3 writes tier: L3.
func Test792a3i1b_LearnFact_ALegacyTierIsWrittenAsGiven(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an empty vault
	vault := newCV1Vault()

	// When the user runs engram learn fact with --tier L3
	result := nw1Learn(vault, "fact", nw1FactFlags().add("--tier", "L3"))
	g.Expect(result.stderr).To(BeEmpty())

	// Then the note carries tier: L3
	note := nw1ReadPrinted(g, vault, result)
	g.Expect(note.front.Tier).To(Equal("L3"))
}

// Test792a3j1a_Learn_TopTakesTheNextTopLevelNumber verifies 792a3j1a (note-writing.md:258): in a vault whose notes have
// IDs 1, 2 and 2a, engram learn fact --position top gives the new note the ID 3.
func Test792a3j1a_Learn_TopTakesTheNextTopLevelNumber(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose notes have IDs 1, 2 and 2a
	vault := newCV1Vault()
	nw1Seed(g, vault, nw1Step{"top", ""}, nw1Step{"top", ""}, nw1Step{"continuation", "2"})
	g.Expect(nw1IDs(vault)).To(Equal([]string{"1", "2", "2a"}))

	// When the user runs engram learn fact --position top
	result := nw1Learn(vault, "fact", nw1FactFlags().set("--position", "top"))
	g.Expect(result.stderr).To(BeEmpty())

	// Then the new note's ID is 3
	g.Expect(nw1PrintedID(result)).To(Equal("3"))
}

// Test792a3j1b_Learn_AContinuationOfANumberTakesTheNextLetter verifies 792a3j1b (note-writing.md:263): in a vault whose
// notes have IDs 1 and 1a, engram learn fact --position continuation --target 1 gives the new note the ID 1b.
func Test792a3j1b_Learn_AContinuationOfANumberTakesTheNextLetter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose notes have IDs 1 and 1a
	vault := newCV1Vault()
	nw1Seed(g, vault, nw1Step{"top", ""}, nw1Step{"continuation", "1"})
	g.Expect(nw1IDs(vault)).To(Equal([]string{"1", "1a"}))

	// When the user runs engram learn fact --position continuation --target 1
	result := nw1Learn(vault, "fact", nw1FactFlags().set("--position", "continuation").add("--target", "1"))
	g.Expect(result.stderr).To(BeEmpty())

	// Then the new note's ID is 1b
	g.Expect(nw1PrintedID(result)).To(Equal("1b"))
}

// Test792a3j1c_LearnRunbook_AContinuationOfALetterTakesTheNextNumber verifies 792a3j1c (note-writing.md:268): in a
// vault whose note has ID 1a, engram learn runbook --position continuation --target 1a gives the new note the ID 1a1.
func Test792a3j1c_LearnRunbook_AContinuationOfALetterTakesTheNextNumber(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose note has ID 1a
	vault := newCV1Vault()
	nw1Seed(g, vault, nw1Step{"top", ""}, nw1Step{"continuation", "1"})
	g.Expect(nw1IDs(vault)).To(ContainElement("1a"))

	// When the user runs engram learn runbook --position continuation --target 1a
	result := nw1Learn(vault, "runbook", nw1RunbookFlags().set("--position", "continuation").add("--target", "1a"))
	g.Expect(result.stderr).To(BeEmpty())

	// Then the new note's ID is 1a1
	g.Expect(nw1PrintedID(result)).To(Equal("1a1"))
}

// Test792a3j1d_LearnFeedback_ASiblingTakesTheNextChildOfTheTargetsParent verifies 792a3j1d (note-writing.md:273): in a
// vault whose notes have IDs 1, 1a and 1a1, engram learn feedback --position sibling --target 1a1 gives the new note
// the ID 1a2.
func Test792a3j1d_LearnFeedback_ASiblingTakesTheNextChildOfTheTargetsParent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose notes have IDs 1, 1a and 1a1
	vault := newCV1Vault()
	nw1Seed(g, vault, nw1Step{"top", ""}, nw1Step{"continuation", "1"}, nw1Step{"continuation", "1a"})
	g.Expect(nw1IDs(vault)).To(Equal([]string{"1", "1a", "1a1"}))

	// When the user runs engram learn feedback --position sibling --target 1a1
	result := nw1Learn(vault, "feedback", nw1FeedbackFlags().set("--position", "sibling").add("--target", "1a1"))
	g.Expect(result.stderr).To(BeEmpty())

	// Then the new note's ID is 1a2
	g.Expect(nw1PrintedID(result)).To(Equal("1a2"))
}

// Test792a3j1e_Learn_LettersRollOverFromZToAA verifies 792a3j1e (note-writing.md:278): in a vault whose notes have IDs
// 1 and 1a through 1z, engram learn fact --position continuation --target 1 gives the new note the ID 1aa.
func Test792a3j1e_Learn_LettersRollOverFromZToAA(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose notes have IDs 1 and 1a through 1z
	vault := newCV1Vault()
	steps := make([]nw1Step, 0, nw1LetterCount+1)
	steps = append(steps, nw1Step{"top", ""})

	for range nw1LetterCount {
		steps = append(steps, nw1Step{"continuation", "1"})
	}

	nw1Seed(g, vault, steps...)
	g.Expect(nw1IDs(vault)).To(HaveLen(nw1LetterCount + 1))
	g.Expect(nw1IDs(vault)).To(ContainElement("1z"))

	// When the user runs engram learn fact --position continuation --target 1
	result := nw1Learn(vault, "fact", nw1FactFlags().set("--position", "continuation").add("--target", "1"))
	g.Expect(result.stderr).To(BeEmpty())

	// Then the new note's ID is 1aa
	g.Expect(nw1PrintedID(result)).To(Equal("1aa"))
}

// Test792a3k1a_LearnFact_AMissingPositionIsRefused verifies 792a3k1a (note-writing.md:294): in an existing vault,
// engram learn fact without --position fails saying the position must be top, continuation or sibling, and no note is
// written.
func Test792a3k1a_LearnFact_AMissingPositionIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an existing vault
	vault := newCV1Vault()
	before := vault.fsys.snapshot()

	// When the user runs engram learn fact without --position
	result := nw1Learn(vault, "fact", nw1FactFlags().without("--position"))

	// Then it fails saying the position must be top, continuation or sibling, and no note is written
	g.Expect(result.stderr).To(ContainSubstring("position must be top, continuation"))
	g.Expect(result.stderr).To(ContainSubstring("sibling"))
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// Test792a3k1b_LearnFact_AContinuationWithoutATargetIsRefused verifies 792a3k1b (note-writing.md:299): in an existing
// vault, engram learn fact --position continuation without --target fails saying a target is required, and no note is
// written.
func Test792a3k1b_LearnFact_AContinuationWithoutATargetIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an existing vault
	vault := newCV1Vault()
	before := vault.fsys.snapshot()

	// When the user runs engram learn fact --position continuation without --target
	result := nw1Learn(vault, "fact", nw1FactFlags().set("--position", "continuation"))

	// Then it fails saying a target is required, and no note is written
	g.Expect(result.stderr).To(ContainSubstring("target required"))
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// Test792a3k1c_LearnFact_ASiblingOfATopLevelNoteIsRefused verifies 792a3k1c (note-writing.md:304): in a vault whose
// note has ID 1, engram learn fact --position sibling --target 1 fails saying a sibling of a top-level note requires
// --position top, and no note is written.
func Test792a3k1c_LearnFact_ASiblingOfATopLevelNoteIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose note has ID 1
	vault := newCV1Vault()
	nw1Seed(g, vault, nw1Step{"top", ""})
	g.Expect(nw1IDs(vault)).To(Equal([]string{"1"}))

	before := vault.fsys.snapshot()

	// When the user runs engram learn fact --position sibling --target 1
	result := nw1Learn(vault, "fact", nw1FactFlags().set("--position", "sibling").add("--target", "1"))

	// Then it fails saying a sibling of a top-level note requires --position top, and no note is written
	g.Expect(result.stderr).To(ContainSubstring("sibling of top-level requires position=top"))
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// Test792a3k1d_LearnFact_AMalformedTargetIsRefused verifies 792a3k1d (note-writing.md:309): in an existing vault,
// engram learn fact --position continuation --target abc fails saying the ID must start with a digit, and no note is
// written.
func Test792a3k1d_LearnFact_AMalformedTargetIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given an existing vault
	vault := newCV1Vault()
	before := vault.fsys.snapshot()

	// When the user runs engram learn fact --position continuation --target abc
	result := nw1Learn(vault, "fact", nw1FactFlags().set("--position", "continuation").add("--target", "abc"))

	// Then it fails saying the ID must start with a digit, and no note is written
	g.Expect(result.stderr).To(ContainSubstring("ID must start with a digit"))
	g.Expect(vault.fsys.snapshot()).To(Equal(before))
}

// unexported constants.
const (
	// nw1LetterCount is the number of letters a to z.
	nw1LetterCount = 26
)

// nw1Flags is the flags of one engram learn command, in command-line order.
type nw1Flags []string

// add appends a flag with its value, keeping any existing flag of the same name (a repeatable flag).
func (f nw1Flags) add(name, value string) nw1Flags {
	return append(append(nw1Flags{}, f...), name, value)
}

// set replaces the value of the flag, or appends the flag when it is absent.
func (f nw1Flags) set(name, value string) nw1Flags {
	out := append(nw1Flags{}, f...)

	for index := 0; index+1 < len(out); index += 2 {
		if out[index] == name {
			out[index+1] = value

			return out
		}
	}

	return append(out, name, value)
}

// without drops the flag and its value.
func (f nw1Flags) without(name string) nw1Flags {
	out := make(nw1Flags, 0, len(f))

	for index := 0; index+1 < len(f); index += 2 {
		if f[index] != name {
			out = append(out, f[index], f[index+1])
		}
	}

	return out
}

// nw1Front is the part of a note's frontmatter the scenarios read.
type nw1Front struct {
	Type       string         `yaml:"type"`
	Tier       string         `yaml:"tier"`
	Situation  string         `yaml:"situation"`
	DoneWhen   string         `yaml:"done_when"`
	Luhmann    string         `yaml:"luhmann"`
	Project    string         `yaml:"project"`
	Tags       []string       `yaml:"tags"`
	Sources    []string       `yaml:"sources"`
	Supersedes []nw1Supersede `yaml:"supersedes"`
	RedFlags   []string       `yaml:"red_flags"`
	Triggers   []string       `yaml:"triggers"`
}

// nw1Note is one written note: its raw text, parsed frontmatter and body.
type nw1Note struct {
	raw   string
	front nw1Front
	body  string
}

// frontKeys is the set of top-level frontmatter keys, so a scenario can assert a field is absent.
func (n nw1Note) frontKeys() map[string]any {
	keys := map[string]any{}

	block, _, found := embed.SplitFrontmatter([]byte(n.raw))
	if !found {
		return keys
	}

	if yaml.Unmarshal(block, &keys) != nil {
		return map[string]any{}
	}

	return keys
}

// nw1Step is one note of a seeded vault: the position and the target of the engram learn fact that writes it.
type nw1Step struct {
	position string
	target   string
}

// nw1Supersede is one supersedes: entry of a note's frontmatter.
type nw1Supersede struct {
	Note  string `yaml:"note"`
	Type  string `yaml:"type"`
	Claim string `yaml:"claim"`
}

// nw1FactFlags is a valid engram learn fact command line; a scenario sets, adds or drops flags of it.
func nw1FactFlags() nw1Flags {
	return nw1Flags{
		"--slug", "nw1-note", "--position", "top", "--source", "nw1", "--situation", "concurrent Go code",
		"--subject", "s", "--predicate", "p", "--object", "o",
	}
}

// nw1FeedbackFlags is a valid engram learn feedback command line.
func nw1FeedbackFlags() nw1Flags {
	return nw1Flags{
		"--slug", "nw1-note", "--position", "top", "--source", "nw1", "--situation", "writing Go code",
		"--behavior", "b", "--impact", "i", "--action", "a",
	}
}

// nw1IDs lists the Luhmann IDs of the vault's notes, in the order the vault lists them.
func nw1IDs(vault *cv1Vault) []string {
	idPattern := regexp.MustCompile(`^([0-9a-z]+)\.`)
	names := vault.namesMatching(`\.md$`)
	ids := make([]string, 0, len(names))

	for _, name := range names {
		if match := idPattern.FindStringSubmatch(name); match != nil {
			ids = append(ids, match[1])
		}
	}

	return ids
}

// nw1Learn runs engram learn <kind> with the flags over the vault.
func nw1Learn(vault *cv1Vault, kind string, flags nw1Flags) cv1Result {
	return vault.run(append([]string{"learn", kind}, flags...)...)
}

// nw1PrintedID is the Luhmann ID in the note path a command printed on stdout.
func nw1PrintedID(result cv1Result) string {
	name := result.stdout[strings.LastIndex(result.stdout, "/")+1:]

	id, _, _ := strings.Cut(strings.TrimSpace(name), ".")

	return id
}

// nw1ReadPrinted reads the note whose path the command printed on stdout; it fails the scenario when the path is not
// a note of the vault.
func nw1ReadPrinted(g *WithT, vault *cv1Vault, result cv1Result) nw1Note {
	path := strings.TrimSpace(result.stdout)
	g.Expect(path).To(HavePrefix(cv1VaultPath+"/"), "the printed path")

	raw := vault.read(strings.TrimPrefix(path, cv1VaultPath+"/"))
	g.Expect(raw).NotTo(BeEmpty(), "the note at %s", path)

	block, rest, found := embed.SplitFrontmatter([]byte(raw))
	g.Expect(found).To(BeTrue(), "the note has frontmatter")

	var front nw1Front

	g.Expect(yaml.Unmarshal(block, &front)).To(Succeed())

	return nw1Note{raw: raw, front: front, body: string(rest)}
}

// nw1RunbookFlags is a valid engram learn runbook command line.
func nw1RunbookFlags() nw1Flags {
	return nw1Flags{
		"--slug", "nw1-note", "--position", "top", "--source", "nw1", "--situation", "releasing a build",
		"--done-when", "the tag is pushed", "--body", "1. Run the checks",
	}
}

// nw1Seed learns one fact per step, in order, and fails the scenario when a command prints an error.
func nw1Seed(g *WithT, vault *cv1Vault, steps ...nw1Step) {
	for _, step := range steps {
		flags := nw1FactFlags().set("--position", step.position)

		if step.target != "" {
			flags = flags.add("--target", step.target)
		}

		result := nw1Learn(vault, "fact", flags)
		g.Expect(result.stderr).To(BeEmpty())
	}
}
