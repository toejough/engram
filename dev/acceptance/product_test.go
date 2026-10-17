//go:build network

package acceptance_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test793a2a1a_NextSessionHasMemoryAfterOneEngramUpdate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var noteName string

	step(t, "Given a machine with Claude Code on which the user has never run engram update, "+
		"and a vault holding a note that says when the staging database is reset", func() {
		machine.hasClaudeCode()
		noteName = machine.learnsFact(stagingReset)
	})

	step(t, "When the user runs engram update", func() {
		machine.mustSucceed(machine.update(), "engram update")
	})

	var sent *turn

	step(t, "And starts a Claude Code session in a project other than the one engram update was run in, "+
		"and sends a prompt asking when the staging database is reset", func() {
		sent = machine.startSession("another-project").send(askStagingReset)
		t.Logf("the agent replied %q (cost $%.4f)", sent.reply, sent.cost)
	})

	step(t, "Then the note is in the agent's context before the agent answers", func() {
		blocks := sent.blocks()

		g.Expect(blocks).To(HaveLen(1), "blocks automatic memory delivered with the prompt")
		g.Expect(blocks[0]).To(ContainSubstring(noteName))
		g.Expect(blocks[0]).To(ContainSubstring(stagingReset.object))
		g.Expect(sent.blockAt()).To(BeNumerically(">=", 0))
		g.Expect(sent.blockAt()).To(BeNumerically("<", sent.agentActsAt()), "the block comes before the agent acts")
	})
}
