//go:build network

package acceptance_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func Test793d2a1a_Mod_AddsTheBlockBesideThePrompt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given a session with the mod installed, and an engram whose auto deliver prints a block", func() {
		machine.hasClaudeCode()
		machine.hasMod()
		machine.hasStandInEngram(standInBlock, 0, 0)
	})

	var sent *turn

	step(t, "When a prompt is sent", func() {
		sent = machine.startSession("a-project").sendOffline()
	})

	step(t, "Then the command received the prompt text, and the session holds the prompt exactly as sent "+
		"and then the block as context before the agent's first reply", func() {
		args, request := machine.standInCalled()

		g.Expect(args).To(Equal("auto deliver"))
		g.Expect(request).To(HaveKeyWithValue("text", promptHello))

		text, isPrompt := sent.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(promptHello))

		g.Expect(sent.blocks()).To(Equal([]string{strings.TrimSpace(standInBlock)}))
		g.Expect(sent.blockAt()).To(BeNumerically(">", 0), "the block follows the prompt's own row")
		g.Expect(sent.blockAt()).To(BeNumerically("<", sent.agentActsAt()), "the block comes before the agent acts")
	})
}

func Test793d2a1b_Mod_GivesTheCommandTheSessionsID(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var inSession *session

	step(t, "Given a session with the mod installed, and an engram whose auto deliver prints a block", func() {
		machine.hasClaudeCode()
		machine.hasMod()
		machine.hasStandInEngram(standInBlock, 0, 0)
		inSession = machine.startSession("a-project")
	})

	step(t, "When a prompt is sent", func() {
		inSession.sendOffline()
	})

	step(t, "Then the command received the ID of the session the prompt was sent in", func() {
		_, request := machine.standInCalled()

		g.Expect(request).To(HaveKeyWithValue("session", inSession.id))
	})
}

func Test793d2a1c_Mod_AnEmptyOutputAddsNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given a session with the mod installed, and an engram whose auto deliver prints nothing", func() {
		machine.hasClaudeCode()
		machine.hasMod()
		machine.hasStandInEngram("", 0, 0)
	})

	var sent *turn

	step(t, "When a prompt is sent", func() {
		sent = machine.startSession("a-project").sendOffline()
	})

	step(t, "Then the session holds the prompt exactly as sent and no context beside it", func() {
		text, isPrompt := sent.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(promptHello))

		g.Expect(sent.contexts()).To(BeEmpty(), "context beside the prompt")
		g.Expect(machine.hookFailures()).To(BeEmpty(), "the mod's hook was skipped")
		g.Expect(sent.agentActsAt()).To(BeNumerically(">", 0), "the model was asked")
	})
}

func Test793d2a1d_Mod_ASlowCommandIsEndedAndThePromptContinues(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given a session with the mod installed, and an engram whose auto deliver takes 20 seconds", func() {
		machine.hasClaudeCode()
		machine.hasMod()
		machine.hasStandInEngram(standInBlock, slowLookup, 0)
	})

	var sent *turn

	step(t, "When a prompt is sent", func() {
		sent = machine.startSession("a-project").sendOffline()
		t.Logf("the prompt waited %s", sent.hold())
	})

	step(t, "Then the model is asked within 8 seconds of the prompt, with the prompt exactly as sent and no context "+
		"beside it, and the mod's hook did not fail", func() {
		g.Expect(sent.hold()).To(BeNumerically("<", holdLimit), "how long the prompt waited")

		text, isPrompt := sent.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(promptHello))

		g.Expect(sent.contexts()).To(BeEmpty(), "context beside the prompt")
		g.Expect(machine.hookFailures()).To(BeEmpty(), "the mod's hook was skipped")
	})
}

func Test793d2a1e_Mod_AFailedCommandAddsNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given a session with the mod installed, and an engram whose auto deliver prints a block and exits with 1",
		func() {
			machine.hasClaudeCode()
			machine.hasMod()
			machine.hasStandInEngram(standInBlock, 0, 1)
		})

	var sent *turn

	step(t, "When a prompt is sent", func() {
		sent = machine.startSession("a-project").sendOffline()
	})

	step(t, "Then the session holds the prompt exactly as sent and no context beside it, "+
		"and the mod's hook did not fail", func() {
		args, _ := machine.standInCalled()
		g.Expect(args).To(Equal("auto deliver"), "the command was run")

		text, isPrompt := sent.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(promptHello))

		g.Expect(sent.contexts()).To(BeEmpty(), "context beside the prompt")
		g.Expect(machine.hookFailures()).To(BeEmpty(), "the mod's hook was skipped")
		g.Expect(sent.agentActsAt()).To(BeNumerically(">", 0), "the model was asked")
	})
}

func Test793d2a1f_Mod_OutputThatIsNotABlockAddsNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given a session with the mod installed, and an engram whose auto deliver prints text that does not begin "+
		"with [engram memory] and exits with 0", func() {
		machine.hasClaudeCode()
		machine.hasMod()
		machine.hasStandInEngram(standInNotABlock, 0, 0)
	})

	var sent *turn

	step(t, "When a prompt is sent", func() {
		sent = machine.startSession("a-project").sendOffline()
	})

	step(t, "Then the session holds the prompt exactly as sent and no context beside it, "+
		"and the mod's hook did not fail", func() {
		args, _ := machine.standInCalled()
		g.Expect(args).To(Equal("auto deliver"), "the command was run")

		text, isPrompt := sent.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(promptHello))

		g.Expect(sent.contexts()).To(BeEmpty(), "context beside the prompt")
		g.Expect(machine.hookFailures()).To(BeEmpty(), "the mod's hook was skipped")
		g.Expect(sent.agentActsAt()).To(BeNumerically(">", 0), "the model was asked")
	})
}

// unexported constants.
const (
	// promptHello is a prompt no note bears on; the stand-in engram answers whatever it is asked.
	promptHello = "hello there"
	// standInBlock is what the stand-in engram prints for any prompt.
	standInBlock = "[engram memory] A block from a stand-in engram.\n- a.md (s): b\n"
	// standInNotABlock is what an engram that does not know the command prints: not a block.
	standInNotABlock = "Unknown command: auto\nusage: engram <command> [flags]\n"
)
