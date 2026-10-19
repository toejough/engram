//go:build network

package acceptance_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/luhmann"
)

func Test855a2b1a_ANoteStatesTheRuleAndNotTheStory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands "+
		"and has force-pushed a branch with git push --force", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
		chat.tell(askToForcePush)
	})

	step(t, "When the user tells the agent in a prompt to push with --force-with-lease and never with --force, and "+
		"the turn ends", func() {
		chat.tell(correctTheForcePush)
	})

	step(t, "Then within 2 minutes the vault holds a note that states the rule, holds the flag --force-with-lease, "+
		"names pushing a branch as the situation in which it applies, and does not tell what happened in the turn",
		func() {
			name := machine.awaitAutomaticNote(chat.turnEnds, "that states the rule", leaseFlag)
			written := machine.noteOf(name)

			g.Expect(strings.ToLower(written.field("situation"))).To(ContainSubstring("push"), "the situation")
			g.Expect(written.text).NotTo(ContainSubstring(forcePushedBranch), "the story of the turn")

			machine.awaitSettled(chat.session.id, chat.turnEnds, 2)
		})
}

func Test855a2b1b_TheNoteStatesTheLessonTheLearnSkillWrites(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var automatic, byTheSkill vaultNote

	step(t, "Given the user has run engram update, and two sessions with the same history, in which the agent has "+
		"force-pushed a branch with git push --force, the first where the agent may not run engram commands and the "+
		"second where it may and where automatic memory is not on", func() {
		machine.hasRunUpdate()
	})

	var first, second *conversation

	step(t, "When in each the user tells the agent in a prompt to push with --force-with-lease and never with "+
		"--force, and in the second asks it to run the learn skill, and the turn ends", func() {
		first = machine.startSession("zorblax").converse()
		first.tell(askToForcePush)
		first.tell(correctTheForcePush)

		name := machine.awaitAutomaticNote(first.turnEnds, "that states the rule", leaseFlag)
		automatic = machine.noteOf(name)

		machine.automaticMemoryIsOff()
		machine.agentMayRunEngramAndSkills()

		second = machine.startSession("another-project").converse()
		second.tell(askToForcePush)
		second.tell(correctTheForcePush)
		second.tell(askToRunTheLearnSkill)

		byTheSkill, _ = machine.awaitNote(second.turnEnds, "that the learn skill wrote", func(note vaultNote) bool {
			return note.name != name && note.states(leaseFlag) && !strings.Contains(note.source(), "written automatically")
		})
	})

	step(t, "Then within 2 minutes the note that automatic memory wrote in the first and the note that the learn "+
		"skill wrote in the second state the same rule, hold the same flag and name the same situation", func() {
		for _, written := range []vaultNote{automatic, byTheSkill} {
			g.Expect(written.states(leaseFlag)).To(BeTrue(), "the flag, in "+written.name)
			g.Expect(strings.ToLower(written.field("situation"))).To(ContainSubstring("push"),
				"the situation of "+written.name)
		}
	})
}

func Test855a2b1c_AConfirmedProcedureIsWrittenAsARunbook(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user tells the agent in a prompt that releasing this project is always: run make ci, then tag "+
		"with git tag vX.Y.Z, then push the tag with git push origin vX.Y.Z, and the agent does these in that order "+
		"and the user confirms it is the way, and the turn ends", func() {
		chat.tell(askToReleaseInOrder)
		chat.tell(confirmTheRelease)
	})

	step(t, "Then within 2 minutes the vault holds a runbook note that names releasing the project as its "+
		"situation, lists the three commands in that order, and says what is true once they are done", func() {
		note, _ := machine.awaitNote(chat.turnEnds, "that is a runbook for the release", func(note vaultNote) bool {
			return note.field("type") == "runbook" && note.states("make ci", "git tag", "git push origin")
		})

		g.Expect(strings.ToLower(note.field("situation"))).To(ContainSubstring("releas"), "the situation")
		g.Expect(note.field("done_when")).NotTo(BeEmpty(), "what is true once the steps are done")
		g.Expect(note.source()).To(ContainSubstring("written automatically"), "the source")

		text := strings.ToLower(note.text)
		ci, tag, push := strings.Index(text, "make ci"), strings.Index(text, "git tag"),
			strings.Index(text, "git push origin")

		g.Expect([]int{ci, tag, push}).To(HaveEach(BeNumerically(">=", 0)), "the three commands")
		g.Expect(ci).To(BeNumerically("<", tag), "make ci comes before the tag")
		g.Expect(tag).To(BeNumerically("<", push), "the tag comes before the push")

		machine.awaitSettled(chat.session.id, chat.turnEnds, 2)
	})
}

func Test855a2c1a_ASubPointOfANoteTheSessionHoldsContinues(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		makeCI string
		chat   *conversation
	)

	step(t, "Given the user has run engram update, a vault holding a note that says the project is built with make "+
		"ci, which runs the lint, test and package steps, and a session in which that note arrived with a prompt",
		func() {
			machine.hasRunUpdate()
			machine.stepsAreRecorded()

			makeCI = machine.learnsFact(buildWithMakeCI)
			chat = machine.startSession("zorblax").converse()

			g.Expect(chat.tell(askPlanTheBuild).blocks()).To(HaveLen(1), "the note arrived with the prompt")
		})

	step(t, "When the user tells the agent in a prompt that the lint step of make ci needs LINT_STRICT=1, and the "+
		"turn ends", func() {
		chat.tell(sayTheLintNeedsStrict)
	})

	step(t, "Then within 2 minutes the vault holds a note for it whose ID continues the ID of the make ci note",
		func() {
			added := machine.awaitAutomaticNote(chat.turnEnds, "that states the lint step", "LINT_STRICT")
			machine.logPlacement()

			g.Expect(idOf(added)).To(HavePrefix(idOf(makeCI)), "the ID of "+added)
			g.Expect(idOf(added)).NotTo(Equal(idOf(makeCI)), "the ID of "+added)

			machine.awaitSettled(chat.session.id, chat.turnEnds, 2)
		})
}

func Test855a2c1b_TheSameThoughtAtTheSameLevelIsASibling(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		black string
		chat  *conversation
	)

	step(t, "Given the user has run engram update, a vault holding a note that says to format Python files with "+
		"black before committing, and a session in which that note arrived with a prompt", func() {
		machine.hasRunUpdate()
		machine.stepsAreRecorded()

		parent := machine.learnsFact(formattingIsChecked)
		black = machine.learnsFactUnder(parent, formatPythonWithBlack)
		chat = machine.startSession("zorblax").converse()

		g.Expect(chat.tell(askToFormatPython).blocks()).To(HaveLen(1), "the note arrived with the prompt")
	})

	step(t, "When the user tells the agent in a prompt to format Go files with gofmt before committing, and the turn "+
		"ends", func() {
		chat.tell(sayToFormatGo)
	})

	step(t, "Then within 2 minutes the vault holds a note for it whose ID is a sibling of the ID of the black note",
		func() {
			added := machine.awaitAutomaticNote(chat.turnEnds, "that states the Go formatting", "gofmt")
			machine.logPlacement()

			g.Expect(parentOf(idOf(added))).To(Equal(parentOf(idOf(black))), "the parent of the ID of "+added)
			g.Expect(idOf(added)).NotTo(Equal(idOf(black)), "the ID of "+added)

			machine.awaitSettled(chat.session.id, chat.turnEnds, 2)
		})
}

func Test855a2c1c_ANoteTheSessionHasNotSeenIsNotATarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		black string
		chat  *conversation
	)

	step(t, "Given the user has run engram update, a vault holding a note that says to format Python files with "+
		"black before committing, and a session in which no note arrived or was looked up, the prompt below included",
		func() {
			machine.hasRunUpdate()
			machine.stepsAreRecorded()

			black = machine.learnsFact(formatPythonWithBlack)
			chat = machine.startSession("zorblax").converse()
		})

	step(t, "When the user tells the agent in a prompt to format Go files with gofmt before committing, and the turn "+
		"ends", func() {
		g.Expect(chat.tell(sayToFormatGo).blocks()).To(BeEmpty(), "no note arrived with the prompt")
	})

	step(t, "Then within 2 minutes the vault holds a note for it placed at the top, and its ID continues no other "+
		"note's and is no other note's sibling", func() {
		added := machine.awaitAutomaticNote(chat.turnEnds, "that states the Go formatting", "gofmt")
		machine.logPlacement()

		g.Expect(parentOf(idOf(added))).To(BeEmpty(), "the ID of "+added+" has no parent: it is at the top")
		g.Expect(idOf(added)).NotTo(Equal(idOf(black)), "the ID of "+added)

		machine.awaitSettled(chat.session.id, chat.turnEnds, 1)
	})
}

func Test855a2d1a_ACorrectingNoteNamesTheNoteItCorrects(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		sunday string
		chat   *conversation
	)

	step(t, "Given the user has run engram update, a vault holding a note that says the staging database is reset "+
		"every Sunday night, and a session in which that note arrived with a prompt and the agent may not run engram "+
		"commands", func() {
		machine.hasRunUpdate()
		machine.stepsAreRecorded()

		sunday = machine.learnsFact(stagingReset)
		chat = machine.startSession("zorblax").converse()

		g.Expect(chat.tell(askPrepareRun).blocks()).To(HaveLen(1), "the note arrived with the prompt")
	})

	step(t, "When the user tells the agent in a prompt that since last week the staging database is reset every "+
		"Monday night, and the turn ends", func() {
		chat.tell(sayTheResetMoved)
	})

	step(t, "Then within 2 minutes the vault holds a note that says it is reset every Monday night and that it "+
		"updates the Sunday note, and the Sunday note shows that the Monday note supersedes it", func() {
		monday := machine.awaitAutomaticNote(chat.turnEnds, "that states the new reset", "Monday")
		machine.logPlacement()

		text := machine.noteText(monday)

		g.Expect(text).To(ContainSubstring("Supersedes: [["+strings.TrimSuffix(sunday, ".md")+"]] — updates:"),
			"the note names the note it updates")

		// the note is written before its sidecar is, which the lookup needs
		g.Eventually(func() string {
			out := machine.engram("query", "--text", "when is the Zorblax staging database reset", "--phrase",
				"Zorblax staging database reset")
			machine.mustSucceed(out, "engram query")

			return out.stdout
		}).WithTimeout(lookupWait).WithPolling(capturePoll).Should(And(ContainSubstring(sunday), ContainSubstring(monday)),
			"a lookup of the Sunday note shows the Monday note beside it")

		machine.awaitSettled(chat.session.id, chat.turnEnds, 2)
	})
}

// unexported constants.
const (
	// askPlanTheBuild is a recall moment that the buildWithMakeCI note applies to (855a2c1a).
	askPlanTheBuild = "Plan how to build the Zorblax project for release, and say which command you would run. " +
		"Reply with one short sentence."
	// askToForcePush makes the agent say that it force-pushed a branch: the history of 855a2b1a and 855a2b1b.
	askToForcePush = "Reply with exactly this sentence and nothing else: I pushed the branch " + forcePushedBranch +
		" with git push --force origin " + forcePushedBranch + "."
	// askToFormatPython is a recall moment that the formatPythonWithBlack note applies to (855a2c1b): a task of several
	// steps.
	askToFormatPython = "Prepare the Python files of the Zorblax project for a commit: format them, run the lint " +
		"and run the tests. Reply with one short sentence."
	// askToReleaseInOrder states a procedure and has the agent say its three commands in order (855a2b1c).
	askToReleaseInOrder = "Releasing the Zorblax project is always: run make ci, then tag with git tag vX.Y.Z, then " +
		"push the tag with git push origin vX.Y.Z. Say those three commands in that order, one to a line, and run none " +
		"of them."
	// askToRunTheLearnSkill asks the agent for the learn skill in the second session of 855a2b1b.
	askToRunTheLearnSkill = "Run the learn skill now to save the lesson of this conversation. " +
		"Then reply with one short sentence."
	// confirmTheRelease confirms the procedure of askToReleaseInOrder as the way to release.
	confirmTheRelease = "Yes, that is the way: release the Zorblax project always like that, in that order. " +
		"Reply with one short sentence."
	// correctTheForcePush corrects the push of askToForcePush.
	correctTheForcePush = "No - in the Zorblax project push a branch with git push " + leaseFlag +
		" and never with git push --force. Reply with one short sentence."
	// forcePushedBranch is the branch of the story of 855a2b1a: a note that states the rule does not name it.
	forcePushedBranch = "zorblax-fix-1138"
	// leaseFlag is the flag the rule of 855a2b1a names.
	leaseFlag = "--force-with-lease"
	// lookupWait is how long the sidecar of a note that was just written has to be there for the lookup.
	lookupWait = 20 * time.Second
	// sayTheLintNeedsStrict states a sub-point of the buildWithMakeCI note.
	sayTheLintNeedsStrict = "No - make ci alone fails: the lint step of make ci needs LINT_STRICT=1 set. " +
		"Reply with one short sentence."
	// sayTheResetMoved states what moved the reset of stagingReset.
	sayTheResetMoved = "Since last week the Zorblax staging database is reset every Monday night and no longer on " +
		"Sunday night. Reply with one short sentence."
	// sayToFormatGo states a rule at the level of formatPythonWithBlack, as a request to remember it.
	sayToFormatGo = "Remember this: in the Zorblax project, format Go files with gofmt before committing. " +
		"Reply with one short sentence."
)

// unexported variables.
var (
	buildWithMakeCI = fact{
		slug:      "zorblax-build-make-ci",
		situation: "building the Zorblax project for release",
		subject:   "the Zorblax project",
		predicate: "is built with",
		object:    "make ci, which runs the lint, test and package steps",
	}
	formatPythonWithBlack = fact{
		slug:      "zorblax-format-python-black",
		situation: "committing a Python file in the Zorblax project",
		subject:   "Python files of the Zorblax project",
		predicate: "are formatted before committing with",
		object:    "black",
	}
	// formattingIsChecked is the note formatPythonWithBlack continues, so that it is not at the top of the order.
	formattingIsChecked = fact{
		slug:      "zorblax-formatting",
		situation: "committing a file in the Zorblax project",
		subject:   "the files of the Zorblax project",
		predicate: "are formatted before",
		object:    "they are committed",
	}
)

// logPlacement logs what the placement printed for the last turn that was placed, and what the writing was given: the
// reply of the model to it is the writing's placement.
func (m *machine) logPlacement() {
	m.t.Helper()

	printed, ran := m.stepPrinted("place")
	m.t.Logf("the judgement was given %s\nthe placement ran: %v, and printed %s\nthe writing was given %s",
		m.stepsGiven("judge"), ran, printed, m.stepsGiven("write"))
}

// noteOf returns the note of the machine's vault with a file name.
func (m *machine) noteOf(name string) vaultNote {
	m.t.Helper()

	for _, note := range m.notes() {
		if note.name == name {
			return note
		}
	}

	m.t.Fatalf("no note %s in the vault: %v", name, m.noteNames())

	return vaultNote{}
}

// idOf is the Luhmann ID a note's file name begins with.
func idOf(name string) string {
	id, _, _ := strings.Cut(name, ".")

	return id
}

// parentOf is the Luhmann ID of the note an ID continues, or nothing for an ID at the top.
func parentOf(id string) string {
	segments, err := luhmann.ParseID(id)
	if err != nil || len(segments) < 2 {
		return ""
	}

	return strings.Join(slices.Clone(segments[:len(segments)-1]), "")
}
