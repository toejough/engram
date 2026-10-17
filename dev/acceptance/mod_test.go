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
		machine.hasStandInEngram(standInBlock)
	})

	var sent *turn

	step(t, "When a prompt is sent", func() {
		sent = machine.startSession("a-project").sendOffline(promptHello)
	})

	step(t, "Then the command received the prompt text, and the session holds the prompt exactly as sent "+
		"and then the block as context before the agent's first reply", func() {
		args, stdin := machine.standInCalled()

		g.Expect(args).To(Equal("auto deliver"))
		g.Expect(stdin).To(MatchJSON(`{"text":"` + promptHello + `"}`))

		text, isPrompt := sent.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(promptHello))

		g.Expect(sent.blocks()).To(Equal([]string{strings.TrimSpace(standInBlock)}))
		g.Expect(sent.blockAt()).To(BeNumerically(">", 0), "the block follows the prompt's own row")
		g.Expect(sent.blockAt()).To(BeNumerically("<", sent.agentActsAt()), "the block comes before the agent acts")
	})
}

// unexported constants.
const (
	// promptHello is a prompt no note bears on; the stand-in engram answers whatever it is asked.
	promptHello = "hello there"
	// standInBlock is what the stand-in engram prints for any prompt.
	standInBlock = "[engram memory] A block from a stand-in engram.\n- a.md (s): b\n"
)
