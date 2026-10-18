//go:build network

package acceptance_test

import (
	"slices"
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

	step(t, "Then the session shows a line that names the staging note delivered with the earlier prompt, and, with "+
		"the next prompt, a line that names the note written from it", func() {
		g.Expect(earlier.blocks()).To(ConsistOf(ContainSubstring(stagingNote)), "the blocks of the earlier prompt")

		g.Expect(linesNaming(earlier.shown, stagingNote)).To(ConsistOf("delivered the note "+stagingNote),
			"the lines shown with the earlier prompt that name the staging note")
		g.Expect(linesNaming(next.shown, written)).To(ConsistOf(ContainSubstring("wrote the note")),
			"the lines shown with the next prompt that name the note written")

		t.Logf("a log line as the host streams it to the user: %s", chat.rawLine(written))

		holdNone(g, append(slices.Clone(earlier.shown), next.shown...), earlier, next)
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

	step(t, "Then the session shows no line about the write with the second prompt, and with the third a line that "+
		"names the note written", func() {
		g.Expect(second.shown).To(BeEmpty(), "the lines shown with the second prompt")
		g.Expect(linesNaming(third.shown, written)).To(ConsistOf(ContainSubstring("wrote the note")),
			"the lines shown with the third prompt that name the note written")

		holdNone(g, third.shown, second, third)
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

	step(t, "Then the session shows, with the second prompt, a line that says the delivery for the first could not be "+
		"done, with the reason", func() {
		g.Expect(first.shown).To(BeEmpty(), "the lines shown with the first prompt")

		t.Logf("the lines shown with the second prompt: %q", second.shown)

		g.Expect(second.shown).To(ConsistOf(And(ContainSubstring("failed:"),
			ContainSubstring("the delivery for a prompt could not be done"),
			Or(ContainSubstring("within 7000 ms"), ContainSubstring("ended")))),
			"the lines shown with the second prompt")

		holdNone(g, second.shown, first, second)
	})
}

func Test793a2e1d_AWriteThatFailedIsShownWithItsReason(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a vault that cannot be written", func() {
		machine.hasRunUpdate()
		machine.vaultCannotBeWritten()

		chat = machine.startSession("zorblax").converse()
	})

	var second *turn

	step(t, "When the user corrects the agent in a prompt, the turn ends, and the user sends another prompt", func() {
		chat.tell(askHowToBuild)
		chat.tell(correctTheBuild)

		kinds := machine.awaitRecords(chat.session.id, chat.turnEnds, 2)
		g.Expect(kinds[len(kinds)-1]).To(Equal("failed"), "what became of the correction: the write failed")

		second = chat.tell(askSomethingElse)
	})

	step(t, "Then the session shows, with the second prompt, a line that says the write failed, with the reason", func() {
		t.Logf("the lines shown with the second prompt: %q", second.shown)

		g.Expect(second.shown).To(ContainElement(And(ContainSubstring("failed: writing "), HaveSuffix("permission denied"))),
			"the lines shown with the second prompt")

		holdNone(g, second.shown, second)
	})
}

func Test793a2e1e_ASweepThatFailedIsShownWithItsReason(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a chunk index that cannot be written", func() {
		machine.hasRunUpdate()
		machine.chunksCannotBeWritten()

		chat = machine.startSession("zorblax").converse()
	})

	var second *turn

	step(t, "When the user says something in a prompt, the turn ends, and the user sends another prompt", func() {
		chat.tell(askSomethingElse)

		machine.awaitRecordOfKind(chat.session.id, chat.turnEnds, "failed")

		second = chat.tell(askAnotherThing)
	})

	step(t, "Then the session shows, with the second prompt, a line that says the sweep failed, with the reason", func() {
		t.Logf("the lines shown with the second prompt: %q", second.shown)

		g.Expect(second.shown).To(ConsistOf(And(ContainSubstring("failed: "), ContainSubstring("sweep"),
			HaveSuffix("permission denied"))), "the lines shown with the second prompt")

		holdNone(g, second.shown, second)
	})
}

func Test793a2e1f_ATurnThatIsNotCapturedIsShownWithTheReason(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and headless prompts are captured only when the user asks", func() {
		machine.hasRunUpdate()
		machine.headlessIsNotCaptured()

		chat = machine.startSession("zorblax").converse()
	})

	var second *turn

	step(t, "When the user corrects the agent in a headless session, and sends another prompt", func() {
		chat.tell(correctTheBuild)

		machine.awaitRecordOfKind(chat.session.id, chat.turnEnds, "skipped")

		second = chat.tell(askSomethingElse)
	})

	step(t, "Then the session shows, with the second prompt, a line that says the turn was not captured, with the "+
		"reason", func() {
		t.Logf("the lines shown with the second prompt: %q", second.shown)

		g.Expect(second.shown).To(ConsistOf(And(ContainSubstring("not captured"),
			ContainSubstring("ENGRAM_AUTO_CAPTURE_SDK"))), "the lines shown with the second prompt")

		holdNone(g, second.shown, second)
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

// linesNaming returns the lines that tell what automatic memory did that name a note: the judgement of a turn may
// write more than one note, and a line is told for each.
func linesNaming(told []string, note string) []string {
	var named []string

	for _, line := range told {
		if strings.Contains(line, note) {
			named = append(named, line)
		}
	}

	return named
}

// holdNone fails the test when an entry of the context beside the prompts of the turns holds one of the lines shown to
// the user, or a line that begins with an asterisk: the lines go to the user, and the model is told none (793a2d).
func holdNone(g *WithT, shown []string, turns ...*turn) {
	for _, round := range turns {
		for _, entry := range round.contexts() {
			g.Expect(entry).NotTo(MatchRegexp(`(?m)^\* `), "a line of the context that begins with an asterisk")

			for _, line := range shown {
				g.Expect(entry).NotTo(ContainSubstring(line), "an entry of the context holds a line shown to the user")
			}
		}
	}
}
