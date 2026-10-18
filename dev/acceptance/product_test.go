//go:build network

package acceptance_test

import (
	"path/filepath"
	"strings"
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

func Test793a2a1b_ASecondUpdateStillDeliversEachNoteOnce(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var noteName string

	step(t, "Given a machine on which the user has run engram update once, "+
		"and a vault holding a note that says when the staging database is reset", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayNotRunEngram()
		noteName = machine.learnsFact(stagingReset)
		machine.mustSucceed(machine.update(), "engram update")
	})

	step(t, "When the user runs engram update again", func() {
		machine.mustSucceed(machine.update(), "engram update, again")
	})

	var sent *turn

	step(t, "And starts a session and sends a prompt asking when the staging database is reset", func() {
		sent = machine.startSession("a-project").send(askStagingReset)
		t.Logf("the agent replied %q (cost $%.4f)", sent.reply, sent.cost)
	})

	step(t, "Then the note is in the agent's context once", func() {
		g.Expect(sent.blocks()).To(HaveLen(1), "blocks automatic memory delivered with the prompt")
		g.Expect(strings.Count(strings.Join(sent.contexts(), "\n"), noteName)).To(Equal(1),
			"how many times any context beside the prompt names the note")
		g.Expect(sent.toolResultsNaming(noteName)).To(BeFalse(), "the agent looked the note up")
	})
}

func Test793a2a1c_ADryRunTurnsNothingOn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var stagingNote string

	step(t, "Given a machine with Claude Code on which the user has never run engram update, "+
		"and a vault holding a note that says when the staging database is reset", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayNotRunEngram()
		stagingNote = machine.learnsFact(stagingReset)
	})

	var report result

	step(t, "When the user runs engram update --dry-run", func() {
		report = machine.update("--dry-run")
		machine.mustSucceed(report, "engram update --dry-run")
	})

	var sent *turn

	step(t, "And starts a session and sends a prompt asking when the staging database is reset", func() {
		sent = machine.startSession("a-project").send(askStagingReset)
		t.Logf("the agent replied %q (cost $%.4f)", sent.reply, sent.cost)
	})

	step(t, "Then the report lists the automatic memory as planned, and the agent's context holds no note", func() {
		g.Expect(report.stdout).To(HavePrefix("[dry-run] engram update"), "the report says it is a dry run")
		g.Expect(reportSection(report.stdout, "Claude Code")).To(
			ContainElement(HavePrefix("agent-instructions/mods/"+modName+"/")), "the report lists the mod as planned")
		g.Expect(machine.installed(machine.claudeDir, modName)).To(BeEmpty(), "nothing of the mod is installed")

		g.Expect(sent.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
		g.Expect(sent.contexts()).NotTo(ContainElement(ContainSubstring(stagingNote)), "any context beside the prompt")
		g.Expect(sent.toolResultsNaming(stagingNote)).To(BeFalse(), "the agent looked the note up")
		g.Expect(sent.reply).NotTo(BeEmpty(), "the agent answered")
	})
}

func Test793a2a1d_PiGetsNoAutomaticMemory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given a machine with both Claude Code and Pi", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.hasPi()
	})

	var report result

	step(t, "When the user runs engram update", func() {
		report = machine.update()
		machine.mustSucceed(report, "engram update")
	})

	step(t, "Then the report lists the automatic memory under Claude Code and not under Pi, "+
		"and nothing of it is installed under ~/.pi", func() {
		g.Expect(reportSection(report.stdout, "Claude Code")).To(ContainElement(ContainSubstring(modName)))
		g.Expect(reportSection(report.stdout, "Pi")).NotTo(BeEmpty(), "Pi has a section of its own")
		g.Expect(reportSection(report.stdout, "Pi")).NotTo(ContainElement(ContainSubstring(modName)))

		g.Expect(machine.installed(machine.claudeDir, modName)).NotTo(BeEmpty(), "Claude Code has the mod")
		g.Expect(machine.installed(filepath.Join(machine.home, piDirectory), modName)).To(BeEmpty())
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
		machine.lookupTakes(0) // no wait: the machine's engram only notes that it was run

		g.Expect(machine.hasVault()).To(BeFalse(), "the vault is gone")
	})

	var sent *turn

	step(t, "When the user starts a session and sends a prompt", func() {
		sent = machine.startSession("a-project").send(askStagingReset)
		t.Logf("the agent replied %q (cost $%.4f)", sent.reply, sent.cost)
	})

	step(t, "Then the agent answers the prompt, and the vault still does not exist", func() {
		g.Expect(machine.lookupStarted()).To(BeTrue(), "the mod ran its command for the prompt")
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
		chat        *conversation
	)

	step(t, "Given a session in which a prompt asking when the staging database is reset "+
		"arrived with the staging note", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		stagingNote = machine.learnsFact(stagingReset)
		machine.mustSucceed(machine.update(), "engram update")

		// both prompts go through one running process, so that the second prompt's delivery reads a record that the
		// session wrote while it was open, as a user's next prompt does
		inSession = machine.startSession("a-project")
		chat = inSession.converse()
		first := chat.say(askStagingReset)

		g.Expect(first.blocks()).To(HaveLen(1), "blocks automatic memory delivered with the first prompt")
		g.Expect(first.blocks()[0]).To(ContainSubstring(stagingNote))
	})

	var second *turn

	step(t, "When the user sends another prompt about the staging database reset", func() {
		second = chat.say(askStagingResetDay)
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
		stagingNote, backupNote string
		inSession               *session
	)

	step(t, "Given a session in which, before the user ran engram update, "+
		"the agent looked up the staging note itself with engram query, "+
		"and a vault that has gained another note about the staging database since", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		stagingNote = machine.learnsFact(stagingReset)

		inSession = machine.startSession("a-project")
		first := inSession.send(askStagingQuery)

		g.Expect(first.toolResultsNaming(stagingNote)).To(BeTrue(), "the agent's own lookup printed the note")
		g.Expect(first.blocks()).To(BeEmpty(), "the mod was not installed yet")

		backupNote = machine.learnsFact(stagingBackup)
	})

	var second *turn

	step(t, "When the user runs engram update, resumes that session, "+
		"and sends a prompt asking when the staging database is reset", func() {
		machine.mustSucceed(machine.update(), "engram update")

		second = inSession.send(askStagingReset)
		t.Logf("the agent replied %q (cost $%.4f)", second.reply, second.cost)
	})

	step(t, "Then the prompt arrives with the other note and not the staging note", func() {
		blocks := second.blocks()

		g.Expect(blocks).To(HaveLen(1), "blocks automatic memory delivered with the prompt")
		g.Expect(blocks[0]).To(ContainSubstring(backupNote))
		g.Expect(blocks[0]).NotTo(ContainSubstring(stagingNote))
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

	var chat *conversation

	step(t, "Given the user has run engram update, and a lookup that takes 20 seconds", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.learnsFact(stagingReset)
		machine.mustSucceed(machine.update(), "engram update")
		machine.deliveryTakes(slowLookup)

		// the first prompt of a session takes up the start of the process: the prompt that is measured is the second
		chat = machine.startSession("a-project").converse()
		chat.tell(askSomethingElse)
	})

	var sent *turn

	step(t, "When the user sends a prompt", func() {
		sent = chat.tell(askStagingReset)
		t.Logf("the prompt reached the model %s after it was sent; the agent replied %q", sent.reached(), sent.reply)
	})

	step(t, "Then the prompt reaches the agent within 8 seconds, as the user typed it and with nothing from the vault, "+
		"and the agent answers it", func() {
		g.Expect(machine.deliveryStarted()).To(BeTrue(), "the slow lookup was run")
		g.Expect(sent.reached()).To(BeNumerically("<", holdLimit), "how long the prompt waited")
		g.Expect(sent.reached()).To(BeNumerically(">", holdLimit/2), "the prompt did wait for the lookup")

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
