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
		machine.awaitAutomaticNote(chat.turnEnds, "that states the correction", buildTarget)

		g.Expect(inSession.agentRan("engram")).To(BeFalse(), "the agent ran an engram command")
	})
}

func Test793a2c1b_ARequestToRememberBecomesANote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user asks the agent in a prompt to remember something, and the turn ends", func() {
		asked := chat.say(askToRemember)
		t.Logf("the agent replied %q", asked.reply)
	})

	step(t, "Then within 2 minutes the vault holds a note that states it and says it was written automatically, "+
		"and the agent wrote nothing to the vault", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states what was asked to be remembered", "Sunday", "03:40")

		g.Expect(chat.session.agentRan("engram")).To(BeFalse(), "the agent ran an engram command")
	})
}

func Test793a2c1c_AnApproachTheUserConfirmedBecomesANote(t *testing.T) {
	t.Parallel()

	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands "+
		"and has done a task one way", func() {
		machine.hasRunUpdate("sort")

		inSession := machine.startSession("zorblax")
		inSession.hasFile("zorblax.txt", "pear\nfig\napple\n")
		chat = inSession.converse()

		done := chat.say(askToSort)
		t.Logf("the agent replied %q", done.reply)
	})

	step(t, "When the user tells the agent in a prompt that the way worked and to keep doing it, and the turn ends",
		func() {
			confirmed := chat.say(confirmTheSort)
			t.Logf("the agent replied %q", confirmed.reply)
		})

	step(t, "Then within 2 minutes the vault holds a note that states the approach and says it was written "+
		"automatically", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states the approach", "sort", "zorblax")
	})
}

func Test793a2c1d_TheAgentsOwnCorrectionBecomesANote(t *testing.T) {
	t.Parallel()

	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may read files "+
		"but may not run engram commands", func() {
		machine.hasRunUpdate()

		inSession := machine.startSession("zorblax")
		inSession.hasFile("zorblax.conf", "listen_port = 7731\n")
		chat = inSession.converse()
	})

	step(t, "When the user asks the agent in a prompt to guess what a file says and then to read it, "+
		"and the agent says its guess was wrong and states what the file says, and the turn ends", func() {
		corrected := chat.say(askToGuessAndRead)
		t.Logf("the agent replied %q", corrected.reply)
	})

	step(t, "Then within 2 minutes the vault holds a note that states what the file says and says it was written "+
		"automatically", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states what the file says", "7731")
	})
}

func Test793a2c1e_AnApproachThatSucceededWithEvidenceBecomesANote(t *testing.T) {
	t.Parallel()

	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may run a check script "+
		"but may not run engram commands", func() {
		machine.hasRunUpdate("sh check.sh")

		inSession := machine.startSession("zorblax")
		inSession.hasFile("check.sh", checkScript)
		chat = inSession.converse()
	})

	step(t, "When the agent, unsure whether an approach will make the check pass, tries it in a turn "+
		"and the turn shows that the check passed", func() {
		tried := chat.say(askToTryTheFlag)
		t.Logf("the agent replied %q", tried.reply)
	})

	step(t, "Then within 2 minutes the vault holds a note that states the approach and says it was written "+
		"automatically", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states the approach", "zorblax-safe")
	})
}

// unexported constants.
const (
	// askHowToBuild is a question the agent cannot answer from anything but a guess: the project is made up.
	askHowToBuild = "How do I build the Zorblax project? Answer in one line."
	// askForAGuessAndAPlan makes the agent answer with a guess, a plan and chatter, none of which is a lesson.
	askForAGuessAndAPlan = "What do you think is making the Zorblax service start slowly? " +
		"Answer with a guess, then a plan, then a friendly remark. Do not read or run anything."
	// askToGuessAndRead makes the agent state a guess, read a file whose content it cannot guess, and say that the guess
	// was wrong.
	askToGuessAndRead = "Before you read anything, say in one sentence which port the Zorblax service listens on, " +
		"guessing from its name. Then read zorblax.conf with the Read tool. Then say in one sentence which port it " +
		"really listens on, and say that your first guess was wrong."
	// askToRemember asks for a fact to be remembered.
	askToRemember = "Remember that the Zorblax staging database is reset every Sunday at 03:40 UTC. " +
		"Reply with one short sentence."
	// askToSort is a task the agent does one way.
	askToSort = "Print the lines of zorblax.txt in sorted order with the sort command. Reply with one short sentence."
	// askToTryTheFlag makes the agent say what it does not know, try it, and say what it saw.
	askToTryTheFlag = "You do not know whether passing --zorblax-safe makes check.sh pass: say so in one sentence. " +
		"Then run exactly this one command, with nothing added to it: sh check.sh --zorblax-safe. " +
		"Then say in one sentence whether the check passed."
	// buildTarget is the word of the correction that no other text of the conversation holds.
	buildTarget = "zorblax-build"
	// checkScript passes only when it is given the flag.
	checkScript = "#!/bin/sh\nif [ \"$1\" = \"--zorblax-safe\" ]; then echo \"CHECK PASSED\"; exit 0; fi\n" +
		"echo \"CHECK FAILED: the flag is missing\"\nexit 1\n"
	// confirmTheSort confirms the way the agent did askToSort.
	confirmTheSort = "That worked - sorting zorblax.txt with the sort command was exactly the right way. " +
		"Keep sorting Zorblax lists with the sort command. Reply with one short sentence."
	// correctTheBuild corrects whatever the agent answered to askHowToBuild.
	correctTheBuild = "No - in the Zorblax project you build with make " + buildTarget + ", never with make all. " +
		"Reply with one short sentence."
)

func Test793a2c1f_GuessesPlansAndChatterProduceNoNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user sends a prompt that the agent answers with a guess, a plan and chatter, and the turn ends",
		func() {
			answered := chat.say(askForAGuessAndAPlan)
			t.Logf("the agent replied %q", answered.reply)
		})

	step(t, "Then 2 minutes after the turn ends the vault holds no note", func() {
		g.Expect(machine.awaitRecord(chat.session.id, chat.turnEnds)).To(Equal("nothing"), "what became of the turn")
		g.Expect(machine.noteNames()).To(BeEmpty(), "the notes of the vault")
	})
}
