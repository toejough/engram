//go:build network

package acceptance_test

import (
	"testing"

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

// unexported constants.
const (
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
