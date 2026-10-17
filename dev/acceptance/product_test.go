//go:build network

package acceptance_test

import (
	"testing"
	"time"

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

func Test793a2a1e_AVaultThatDoesNotExistIsNotCreated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given a machine with Claude Code on which the user has run engram update, "+
		"and whose configured vault no longer exists", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayNotRunEngram()
		machine.mustSucceed(machine.update(), "engram update")
		machine.vaultIsRemoved()

		g.Expect(machine.hasVault()).To(BeFalse(), "the vault is gone")
	})

	var sent *turn

	step(t, "When the user starts a session and sends a prompt", func() {
		sent = machine.startSession("a-project").send(askStagingReset)
		t.Logf("the agent replied %q (cost $%.4f)", sent.reply, sent.cost)
	})

	step(t, "Then the agent answers the prompt, and the vault still does not exist", func() {
		g.Expect(sent.reply).NotTo(BeEmpty(), "the agent answered")
		g.Expect(sent.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
		g.Expect(machine.hasVault()).To(BeFalse(), "a vault exists at "+machine.vault)
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

func Test793a2b1d_APromptThatMatchesNoNoteArrivesWithNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given the user has run engram update, a vault whose notes have nothing to do with the prompt, "+
		"and a searchable conversation that does", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.learnsFeedback(formatPython)
		machine.hasConversation(earlierConversation)
		machine.mustSucceed(machine.update(), "engram update")

		lookup := machine.engram("query", "--text", askStagingReset, "--phrase", askStagingReset)
		g.Expect(lookup.stdout).To(ContainSubstring("kind: chunk"), "the agent's own lookup finds the conversation")
	})

	var sent *turn

	step(t, "When the user sends the prompt", func() {
		sent = machine.startSession("a-project").send(askStagingReset)
		t.Logf("the agent replied %q (cost $%.4f)", sent.reply, sent.cost)
	})

	step(t, "Then the prompt arrives with nothing from the vault, and the agent answers it", func() {
		g.Expect(sent.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
		g.Expect(sent.reply).NotTo(BeEmpty(), "the agent answered")
	})
}

func Test793a2d1a_ASlowLookupDoesNotHoldAPromptForMoreThan8Seconds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given the user has run engram update, and a lookup that takes 20 seconds", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.learnsFact(stagingReset)
		machine.mustSucceed(machine.update(), "engram update")
		machine.lookupTakes(slowLookup)
	})

	var sent *turn

	step(t, "When the user sends a prompt", func() {
		sent = machine.startSession("a-project").send(askStagingReset)
		t.Logf("the prompt waited %s; the agent replied %q (cost $%.4f)", sent.hold(), sent.reply, sent.cost)
	})

	step(t, "Then the prompt reaches the agent within 8 seconds, as the user typed it and with nothing from the vault, "+
		"and the agent answers it", func() {
		g.Expect(machine.lookupStarted()).To(BeTrue(), "the slow lookup was run")
		g.Expect(sent.hold()).To(BeNumerically("<", holdLimit), "how long the prompt waited")
		g.Expect(sent.hold()).To(BeNumerically(">", holdLimit/2), "the prompt did wait for the lookup")

		text, isPrompt := sent.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(askStagingReset))

		g.Expect(sent.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
		g.Expect(sent.reply).NotTo(BeEmpty(), "the agent answered")
	})
}

// unexported constants.
const (
	// holdLimit is the longest a prompt may be held (793a2d).
	holdLimit = 8 * time.Second
	// slowLookup is a lookup that takes longer than the hold.
	slowLookup = 20 * time.Second
)
