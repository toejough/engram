//go:build network

package acceptance_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test793a2f1a_TheConversationSoFarIsFindableAfterATurn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands and "+
		"the user has told the agent a first thing that only this conversation holds", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
		chat.tell(tellTheRelayLabel)
	})

	step(t, "When the user tells it a second such thing in a prompt, and the turn ends", func() {
		chat.tell(tellTheFuseDrawer)
	})

	step(t, "Then within 2 minutes engram query returns a chunk of the conversation for each of the two things, "+
		"and neither the user nor the agent ran engram ingest", func() {
		took := machine.awaitChunk(chat.turnEnds, askTheRelayLabel, relayLabel)
		t.Logf("a chunk of the first thing was found %s after the turn ended", took)

		took = machine.awaitChunk(chat.turnEnds, askTheFuseDrawer, fuseDrawer)
		t.Logf("a chunk of the second thing was found %s after the turn ended", took)

		g.Expect(chat.session.agentRan("engram")).To(BeFalse(), "the agent ran an engram command")
	})
}

func Test793a2f1b_AConversationTooLongToIndexInTimeIsLeftAloneAndTheUserIsToldOnceWhatToRun(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a conversation that is too long to be indexed within 2 minutes",
		func() {
			machine.hasRunUpdate()
			machine.indexIsBusy()

			chat = machine.startSession("zorblax").converse()
		})

	var first, second, third *turn

	step(t, "When the user sends a prompt, and 2 minutes after its turn ends sends a second, and 2 minutes after the "+
		"second turn ends sends a third", func() {
		first = chat.tell(askSomethingElse)

		time.Sleep(time.Until(chat.turnEnds.Add(captureBound)))

		second = chat.tell(askAnotherThing)

		time.Sleep(time.Until(chat.turnEnds.Add(captureBound)))

		third = chat.tell(askOneMoreThing)
	})

	step(t, "Then the session shows, with the second prompt, one line that says the conversation could not be indexed "+
		"in time and that engram ingest --auto indexes it, and shows no line about it with the third", func() {
		t.Logf("the lines shown with the second prompt: %q", second.shown)

		g.Expect(first.shown).To(BeEmpty(), "the lines shown with the first prompt")
		g.Expect(second.shown).To(ConsistOf(And(ContainSubstring("failed:"), ContainSubstring("sweep"),
			ContainSubstring("engram ingest --auto"))), "the lines shown with the second prompt")
		g.Expect(third.shown).To(BeEmpty(), "the lines shown with the third prompt")

		holdNone(g, second.shown, first, second, third)
	})
}

// unexported constants.
const (
	// askOneMoreThing is a prompt that no note bears on, other than askSomethingElse and askAnotherThing.
	askOneMoreThing = "What is 12 times 12? Reply with the number only."
	// askTheFuseDrawer and askTheRelayLabel are the lookups of the user, which name no word of what was said.
	askTheFuseDrawer = "which drawer holds the spare fuses of the Zorblax cabinet"
	askTheRelayLabel = "what label does the Zorblax relay in the east wing carry"
	// fuseDrawer and relayLabel are the words of the two things, which no vault note and no earlier session holds.
	fuseDrawer = "drawer 31"
	relayLabel = "Q7-amber"
	// tellTheFuseDrawer and tellTheRelayLabel are two things that only this conversation holds: the Zorblax cabinet is
	// made up.
	tellTheFuseDrawer = "The spare fuses of the Zorblax cabinet are kept in drawer 31 of the north rack. " +
		"Just reply OK, and do not read or run anything."
	tellTheRelayLabel = "The Zorblax relay in the east wing carries the label Q7-amber. " +
		"Just reply OK, and do not read or run anything."
)
