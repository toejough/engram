//go:build network

package acceptance_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test793a2d1c_TheWorkAfterATurnDoesNotHoldTheNextPrompt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a judgement, a write and a sweep that each take 60 seconds",
		func() {
			machine.hasRunUpdate()
			machine.jobsTake(slowJob)

			chat = machine.startSession("zorblax").converse()
		})

	var second *turn

	step(t, "When the user sends a prompt, and then another while that work goes on", func() {
		chat.tell(askHowToBuild)

		g.Eventually(func() bool { return machine.jobStarted("judge") && machine.jobStarted("sweep") }).
			WithTimeout(jobStartWait).WithPolling(capturePoll).Should(BeTrue(), "the work after the turn started")

		second = chat.tell(askSomethingElse)
		t.Logf("the second prompt reached the model %s after it was sent, and was answered after %s", second.reached(),
			second.repliedAfter())
	})

	step(t, "Then the second prompt reaches the agent within 8 seconds, as the user typed it, and the agent answers "+
		"it within 20 seconds", func() {
		g.Expect(second.reached()).To(BeNumerically("<", holdLimit), "how long the second prompt waited")
		g.Expect(second.repliedAfter()).To(BeNumerically("<", replyLimit), "how long the second prompt waited for its reply")

		text, isPrompt := second.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(askSomethingElse))

		g.Expect(second.reply).NotTo(BeEmpty(), "the agent answered")
	})
}

func Test793a2d1d_ThePromptAndTheReplyAreAsTheyWere(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		chat    *conversation
		written string
	)

	step(t, "Given the user has run engram update, and a session in which the user's correction became a note",
		func() {
			machine.hasRunUpdate()

			chat = machine.startSession("zorblax").converse()
			chat.tell(askHowToBuild)
			chat.tell(correctTheBuild)

			written = machine.awaitAutomaticNote(chat.turnEnds, "that states the correction", buildTarget)

			kinds := machine.awaitRecords(chat.session.id, chat.turnEnds, 2)
			g.Expect(kinds[len(kinds)-1]).To(Equal("written"), "what became of the correction")
		})

	var asked *turn

	step(t, "When the user sends a prompt that the note bears on, typed with extra spaces and a line break", func() {
		asked = chat.tell(askHowToBuildSpaced)
	})

	step(t, "Then the prompt in the conversation is exactly as the user typed it, and the reply is exactly the "+
		"agent's text", func() {
		g.Expect(asked.blocks()).To(ConsistOf(ContainSubstring(written)), "the note arrived with the prompt")

		text, isPrompt := asked.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(askHowToBuildSpaced), "the prompt in the conversation")

		g.Expect(asked.reply).NotTo(BeEmpty(), "the agent answered")
		g.Expect(asked.finalText()).To(Equal(asked.reply), "the agent's text in the conversation, and the reply")
		g.Expect(chat.shownByTheMod()).To(BeEmpty(), "what the mod showed beneath a reply")
	})
}

func Test793a2d1e_AWriteThatFailsDoesNotBreakTheSession(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a vault that cannot be written", func() {
		machine.hasRunUpdate()
		machine.vaultCannotBeWritten()

		chat = machine.startSession("zorblax").converse()
	})

	var second *turn

	step(t, "When the user corrects the agent in a prompt, the turn ends, and the user sends another prompt", func() {
		chat.tell(askHowToBuild)
		chat.tell(correctTheBuild)

		kinds := machine.awaitRecords(chat.session.id, chat.turnEnds, 2)
		g.Expect(kinds[len(kinds)-1]).To(Equal("failed"), "what became of the correction: the write failed")

		second = chat.tell(askSomethingElse)
	})

	step(t, "Then the agent answers the second prompt as typed, with nothing about the failed write in its context, "+
		"and the vault holds no new note", func() {
		text, isPrompt := second.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(askSomethingElse))

		g.Expect(second.reply).NotTo(BeEmpty(), "the agent answered")
		g.Expect(second.contexts()).To(BeEmpty(), "what the agent's context holds beside the second prompt")
		g.Expect(machine.noteNames()).To(BeEmpty(), "the notes of the vault")
		g.Expect(machine.hookFailures()).To(BeEmpty(), "the mod's hook was skipped")
	})
}

// unexported constants.
const (
	// askHowToBuildSpaced is askHowToBuild typed with extra spaces and a line break, which nothing has a reason to
	// change.
	askHowToBuildSpaced = "How  do I build   the Zorblax project?\nAnswer in one line."
	// askSomethingElse is a prompt that no note bears on.
	askSomethingElse = "What is 17 plus 25? Reply with the number only."
	// jobStartWait is how long the work after a turn has to start.
	jobStartWait = 10 * time.Second
	// replyLimit is the longest the reply to a prompt may take after the prompt is sent, with the hold of 8 seconds in it
	// and well under the 60 seconds of the jobs that go on (793a2d1c).
	replyLimit = 20 * time.Second
	// slowJob is a job after a turn that takes longer than its bound.
	slowJob = 60 * time.Second
)
