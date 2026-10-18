//go:build network

package acceptance_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test793a2e1a_WhatWasDeliveredAndWhatWasWrittenForThePreviousTurnIsInTheSessionByTheNextPrompt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		stagingNote string
		chat        *conversation
		earlier     *turn
		written     string
	)

	step(t, "Given the user has run engram update, a vault holding a note that says when the staging database is "+
		"reset, and a session in which the user asked in one prompt when it is reset and to remember that it is "+
		"backed up every Monday at 05:00 UTC", func() {
		machine.hasRunUpdate()

		stagingNote = machine.learnsFact(stagingReset)
		chat = machine.startSession("zorblax").converse()
		earlier = chat.tell(askResetAndRememberBackup)

		written = machine.awaitAutomaticNote(chat.turnEnds, "that states the backup", "Monday", "05:00")
		machine.awaitRecords(chat.session.id, chat.turnEnds, 1)
	})

	var next *turn

	step(t, "When the user sends the next prompt", func() {
		next = chat.tell(askSomethingElse)
	})

	step(t, "Then the conversation shows the staging note delivered with the earlier prompt, and the next prompt "+
		"arrives with a line that names the note written from it", func() {
		g.Expect(earlier.blocks()).To(ConsistOf(ContainSubstring(stagingNote)), "the blocks of the earlier prompt")

		g.Expect(next.blocks()).To(HaveLen(1), "the blocks of the next prompt")

		if len(next.blocks()) == 1 {
			g.Expect(linesToldBy(next.blocks()[0])).To(ConsistOf(And(ContainSubstring("wrote the note"),
				ContainSubstring(written))), "the lines that tell what automatic memory did")
		}
	})
}

func Test793a2e1b_AWriteThatFinishesAfterTheNextPromptIsShownWhenItFinishes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a write that takes 12 seconds", func() {
		machine.hasRunUpdate()
		machine.writeTakes(slowWrite)

		chat = machine.startSession("zorblax").converse()
	})

	var (
		second  *turn
		third   *turn
		written string
	)

	step(t, "When the user corrects the agent in a prompt, sends another before the write finishes, and a third "+
		"after it finished", func() {
		chat.tell(askHowToBuild)
		chat.tell(correctTheBuild)

		g.Expect(machine.recordKinds(chat.session.id)).NotTo(ContainElement("written"),
			"the write had not finished when the second prompt was sent")

		second = chat.tell(askSomethingElse)

		written = machine.awaitAutomaticNote(chat.turnEnds, "that states the correction", buildTarget)
		machine.awaitRecordOfKind(chat.session.id, chat.turnEnds, "written")

		third = chat.tell(askAnotherThing)
	})

	step(t, "Then the second prompt arrives with no line about the write, and the third arrives with a line that "+
		"names the note written", func() {
		for _, block := range second.blocks() {
			g.Expect(linesToldBy(block)).To(BeEmpty(), "the lines that tell what automatic memory did, with the second prompt")
		}

		g.Expect(third.blocks()).To(HaveLen(1), "the blocks of the third prompt")

		if len(third.blocks()) == 1 {
			g.Expect(linesToldBy(third.blocks()[0])).To(ConsistOf(And(ContainSubstring("wrote the note"),
				ContainSubstring(written))), "the lines that tell what automatic memory did, with the third prompt")
		}
	})
}

func Test793a2e1c_ADeliveryThatCouldNotBeDoneIsShownWithItsReason(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, a vault holding a note that says when the staging database is "+
		"reset, and a lookup that takes 20 seconds", func() {
		machine.hasRunUpdate()
		machine.learnsFact(stagingReset)
		machine.deliveryTakes(slowLookup)

		chat = machine.startSession("zorblax").converse()
	})

	var first, second *turn

	step(t, "When the user sends a prompt asking when the staging database is reset, the lookup becomes quick again, "+
		"and the user sends another prompt", func() {
		first = chat.tell(askStagingReset)

		g.Expect(machine.deliveryStarted()).To(BeTrue(), "the slow delivery was run")
		machine.awaitRecordOfKind(chat.session.id, chat.turnEnds, "failed")
		machine.deliveryIsQuick()

		second = chat.tell(askSomethingElse)
	})

	step(t, "Then the second prompt arrives with a line that says the delivery for the first could not be done, with "+
		"the reason", func() {
		g.Expect(first.blocks()).To(BeEmpty(), "the blocks of the first prompt")
		g.Expect(second.blocks()).To(HaveLen(1), "the blocks of the second prompt")

		if len(second.blocks()) == 1 {
			told := linesToldBy(second.blocks()[0])

			t.Logf("the lines that tell what automatic memory did, with the second prompt: %q", told)

			g.Expect(told).To(ConsistOf(And(ContainSubstring("failed:"),
				ContainSubstring("the delivery for a prompt could not be done"),
				Or(ContainSubstring("within 7000 ms"), ContainSubstring("ended")))),
				"the lines that tell what automatic memory did, with the second prompt")
		}
	})
}

// unexported constants.
const (
	// askAnotherThing is a prompt that no note bears on, other than askSomethingElse.
	askAnotherThing = "What is 9 times 6? Reply with the number only."
	// askResetAndRememberBackup asks what the stagingReset note answers, and asks to remember a second lesson about the
	// same database.
	askResetAndRememberBackup = "When is the Zorblax staging database reset? Also remember that it is backed up " +
		"every Monday at 05:00 UTC. Reply with one short sentence."
	// slowWrite is how long the write takes in the scenario of a write that finishes after the next prompt.
	slowWrite = 12 * time.Second
)

// linesToldBy returns the lines of a block that tell what automatic memory did: those that begin with an asterisk.
func linesToldBy(block string) []string {
	var told []string

	for line := range strings.SplitSeq(block, "\n") {
		if strings.HasPrefix(line, "* ") {
			told = append(told, line)
		}
	}

	return told
}
