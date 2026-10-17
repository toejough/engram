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

func Test793a2b1b_ANoteDeliveredEarlierInTheSessionIsNotDeliveredAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		stagingNote string
		inSession   *session
	)

	step(t, "Given a session in which a prompt asking when the staging database is reset "+
		"arrived with the staging note", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		stagingNote = machine.learnsFact(stagingReset)
		machine.mustSucceed(machine.update(), "engram update")

		inSession = machine.startSession("a-project")
		first := inSession.send(askStagingReset)

		g.Expect(first.blocks()).To(HaveLen(1), "blocks automatic memory delivered with the first prompt")
		g.Expect(first.blocks()[0]).To(ContainSubstring(stagingNote))
	})

	var second *turn

	step(t, "When the user sends another prompt about the staging database reset", func() {
		second = inSession.send(askStagingResetDay)
		t.Logf("the agent replied %q (cost $%.4f)", second.reply, second.cost)
	})

	step(t, "Then the second prompt arrives with nothing from the vault, "+
		"and the staging note is in the agent's context once", func() {
		g.Expect(second.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the second prompt")
		g.Expect(inSession.blocksNaming(stagingNote)).To(Equal(1), "blocks in the session that name the note")
	})
}

func Test793a2b1c_ANoteTheAgentLookedUpItselfIsNotDelivered(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		stagingNote string
		inSession   *session
	)

	step(t, "Given a session in which, before the user ran engram update, "+
		"the agent looked up the staging note itself with engram query", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		stagingNote = machine.learnsFact(stagingReset)

		inSession = machine.startSession("a-project")
		first := inSession.send(askStagingQuery)

		g.Expect(first.toolResultsNaming(stagingNote)).To(BeTrue(), "the agent's own lookup printed the note")
		g.Expect(first.blocks()).To(BeEmpty(), "the mod was not installed yet")
	})

	var second *turn

	step(t, "When the user runs engram update, resumes that session, "+
		"and sends a prompt asking when the staging database is reset", func() {
		machine.mustSucceed(machine.update(), "engram update")

		second = inSession.send(askStagingReset)
		t.Logf("the agent replied %q (cost $%.4f)", second.reply, second.cost)
	})

	step(t, "Then the prompt arrives with nothing from the vault", func() {
		g.Expect(second.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
	})
}
