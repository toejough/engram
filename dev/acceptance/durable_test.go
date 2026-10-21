//go:build network

package acceptance_test

import (
	"strings"
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

	step(t, "Given the user has run engram update, a vault crowded with notes about a user repeating an instruction, "+
		"and a session whose first turn became a note", func() {
		machine.hasEngram()

		for _, filler := range planFillers {
			machine.learnsFact(filler)
		}

		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
		chat.tell(askHowToBuild)
		chat.tell(correctTheBuild)

		machine.awaitAutomaticNote(chat.turnEnds, "that states the correction", buildTarget)
	})

	step(t, "When the user corrects the agent again in a prompt with none of the lesson's words, the agent's answer "+
		"states the lesson again, and the turn ends", func() {
		chat.tell(repeatTheCorrection)
	})

	step(t, "Then 2 minutes after the turn ends the vault holds the one note that states the correction", func() {
		machine.awaitSettled(chat.session.id, chat.turnEnds, 3)

		var stating []string

		for name, text := range machine.noteTexts() {
			if strings.Contains(strings.ToLower(text), buildTarget) {
				stating = append(stating, name)
			}
		}

		g.Expect(stating).To(HaveLen(1), "the notes of the vault that state the correction")
	})
}

func Test855a2g1c_ACorrectionMadeInWorkOnATicketIsWrittenWithNoTicketInItsSituation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent has answered a question about work "+
		"on a ticket", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
		chat.tell(askAboutTheTicketRows)
	})

	step(t, "When the user corrects the agent in a prompt with a rule that holds beyond the ticket, "+
		"and the turn ends", func() {
		chat.tell(correctTheTicketRows)
	})

	step(t, "Then within 2 minutes the vault holds a note that states the rule and whose situation names no "+
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
	// correctTheTicketRows corrects whatever the agent answered to askAboutTheTicketRows, with a rule that holds beyond
	// the ticket.
	correctTheTicketRows = "No - a measurement row passes only when the meter matched a model call: with no call " +
		"matched the meter reads 0 and 0, which falls under every upper bound, so in the work on #4417 and in any other " +
		"the row must fail when no call was matched. Reply with one short sentence."
	// repeatTheCorrection corrects the agent again with none of the words of correctTheBuild, which the notes about a user
	// repeating an instruction crowd: the lookup of the prompt does not find the note of the session, and the agent's
	// answer states its lesson again.
	repeatTheCorrection = "No, that is not how I told you to do it earlier. Do it my way, and say in one sentence " +
		"what you do now."
)

// unexported variables.
var (
	// planFillers are notes about a user who repeats an instruction, that crowd the lookup of a prompt that repeats one,
	// as the many notes of a real vault do, so that the note of the session is not among the nearest.
	planFillers = []fact{
		planFact("repeat-earlier", "the user says the agent did not follow an earlier instruction",
			"an earlier instruction", "is followed as it was given"),
		planFact("repeat-my-way", "the user asks the agent to do it their way", "the user's way", "is the way to follow"),
		planFact("repeat-not-as-told", "the user says it is not how they told the agent to do it",
			"how the user told the agent", "is what the agent does"),
		planFact("repeat-answer-short", "the user asks for one sentence about what the agent does now",
			"a one sentence answer", "says what is done now"),
		planFact("repeat-correct-again", "the user corrects the agent a second time", "a second correction",
			"is taken as the rule"),
		planFact("repeat-redo", "the user asks the agent to redo a step their way", "a redone step",
			"follows the user's instruction"),
		planFact("repeat-instruction", "the user repeats an instruction", "a repeated instruction",
			"is not to be asked about again"),
		planFact("repeat-earlier-told", "the agent is reminded of what the user told it earlier", "a reminder",
			"is acted on at once"),
		planFact("repeat-do-now", "the user says what the agent should do now", "what to do now",
			"is the next action"),
		planFact("repeat-sentence", "replying to a correction in one sentence", "a reply to a correction",
			"is one sentence"),
	}
)

// planFact is a note about a user who repeats an instruction.
func planFact(slug, situation, subject, object string) fact {
	return fact{slug: slug, situation: situation, subject: subject, predicate: "is", object: object}
}
