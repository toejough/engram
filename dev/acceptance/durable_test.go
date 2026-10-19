//go:build network

package acceptance_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test855a2g1a_AStatusOfTheWorkIsNotWritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user asks the agent in a prompt to remember the status of the work, a branch and a worktree, "+
		"and the turn ends", func() {
		chat.tell(askToRememberTheStatus)
	})

	step(t, "Then 2 minutes after the turn ends the vault holds no note", func() {
		machine.awaitSettled(chat.session.id, chat.turnEnds, 1)

		g.Expect(machine.noteNames()).To(BeEmpty(), "the notes of the vault")
	})
}

func Test855a2g1b_ALessonTheSessionWroteIsNotWrittenTwice(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands and "+
		"whose first turn became a note", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
		chat.tell(askHowToBuild)
		chat.tell(correctTheBuild)

		machine.awaitAutomaticNote(chat.turnEnds, "that states the correction", buildTarget)
	})

	step(t, "When the user states the same lesson again in other words in a prompt of the session, and the turn ends",
		func() {
			chat.tell(correctTheBuildAgain)
		})

	step(t, "Then 2 minutes after the turn ends the vault holds the one note", func() {
		machine.awaitSettled(chat.session.id, chat.turnEnds, 3)

		g.Expect(machine.noteNames()).To(HaveLen(1), "the notes of the vault")
	})
}

func Test855a2g1c_ACorrectionMadeInWorkOnATicketIsWrittenWithNoTicketInItsSituation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands and "+
		"has answered a question about work on a ticket", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
		chat.tell(askAboutTheTicketRows)
	})

	step(t, "When the user corrects the agent in a prompt, and the turn ends", func() {
		chat.tell(correctTheTicketRows)
	})

	step(t, "Then within 2 minutes the vault holds a note that states the correction and whose situation names no "+
		"ticket", func() {
		note, _ := machine.awaitNote(chat.turnEnds, "that states the correction", func(note vaultNote) bool {
			return note.states("meter")
		})

		g.Expect(note.field("situation")).NotTo(ContainSubstring("4417"), "the situation of "+note.name)
	})
}

// unexported constants.
const (
	// askAboutTheTicketRows is a question about work on a ticket whose rows the agent cannot have run.
	askAboutTheTicketRows = "Did the measurement rows of ticket #4417 of the Zorblax project pass? Answer in one line."
	// askToRememberTheStatus asks for the status of the work to be remembered: a branch, a worktree and a next step.
	askToRememberTheStatus = "Remember that the Zorblax release work is on the branch zorblax/release-9 in the " +
		"worktree zorblax-wt at commit 4f1c0a9, and that the changelog is the next step. Reply with one short sentence."
	// correctTheBuildAgain states the lesson of correctTheBuild again, in other words.
	correctTheBuildAgain = "Once more, for your notes: the build command of the Zorblax project is make " + buildTarget +
		", and make all is wrong there. Reply with one short sentence."
	// correctTheTicketRows corrects whatever the agent answered to askAboutTheTicketRows, with a rule that holds beyond
	// the ticket.
	correctTheTicketRows = "No - a measurement row passes only when the meter matched a model call: with no call " +
		"matched the meter reads 0 and 0, which falls under every upper bound, so in the work on #4417 and in any other " +
		"the row must fail when no call was matched. Reply with one short sentence."
)
