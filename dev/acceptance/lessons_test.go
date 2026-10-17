//go:build network

package acceptance_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test793a2c1a_ACorrectionBecomesANote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		inSession *session
		chat      *conversation
	)

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands "+
		"and has answered a question about how to build a project", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayNotRunEngram()
		machine.mustSucceed(machine.update(), "engram update")

		inSession = machine.startSession("zorblax")
		chat = inSession.converse()
		answered := chat.say(askHowToBuild)
		t.Logf("the agent replied %q", answered.reply)
	})

	step(t, "When the user corrects the agent's answer in a prompt, and the turn ends", func() {
		corrected := chat.say(correctTheBuild)
		t.Logf("the agent replied %q", corrected.reply)
	})

	step(t, "Then within 2 minutes the vault holds a note that states the correction and says it was written "+
		"automatically, and the agent wrote nothing to the vault", func() {
		note, took := machine.awaitNote(chat.turnEnds, "that states the correction", func(note vaultNote) bool {
			return note.states(buildTarget)
		})
		t.Logf("the note %s was written %s after the turn ended:\n%s", note.name, took, note.text)

		g.Expect(note.field("source")).To(ContainSubstring("written automatically"))
		g.Expect(inSession.agentRan("engram")).To(BeFalse(), "the agent ran an engram command")
	})
}

// unexported constants.
const (
	// askHowToBuild is a question the agent cannot answer from anything but a guess: the project is made up.
	askHowToBuild = "How do I build the Zorblax project? Answer in one line."
	// buildTarget is the word of the correction that no other text of the conversation holds.
	buildTarget = "zorblax-build"
	// correctTheBuild corrects whatever the agent answered to askHowToBuild.
	correctTheBuild = "No - in the Zorblax project you build with make " + buildTarget + ", never with make all. " +
		"Reply with one short sentence."
)
