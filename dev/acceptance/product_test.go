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
		machine.hasEngram()
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

func Test793a2b1a_TheNotesThatBearOnAPromptArriveWithIt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var stagingNote, formatNote string

	step(t, "Given the user has run engram update, and a vault holding a note that says when the staging "+
		"database is reset and a note about formatting Python files", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		stagingNote = machine.learnsFact(stagingReset)
		formatNote = machine.learnsFeedback(formatPython)
		machine.mustSucceed(machine.update(), "engram update")
	})

	var sent *turn

	step(t, "When the user sends a prompt asking when the staging database is reset", func() {
		sent = machine.startSession("a-project").send(askStagingReset)
		t.Logf("the agent replied %q (cost $%.4f)", sent.reply, sent.cost)
	})

	step(t, "Then before the agent acts its context holds the staging note and not the formatting note, "+
		"and the prompt in the conversation is exactly as the user typed it", func() {
		blocks := sent.blocks()

		g.Expect(blocks).To(HaveLen(1), "blocks automatic memory delivered with the prompt")
		g.Expect(blocks[0]).To(ContainSubstring(stagingNote))
		g.Expect(blocks[0]).NotTo(ContainSubstring(formatNote))
		g.Expect(sent.blockAt()).To(BeNumerically("<", sent.agentActsAt()), "the block comes before the agent acts")

		text, isPrompt := sent.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(askStagingReset))
	})
}
