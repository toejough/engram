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
		t.Logf("the second prompt reached the model %s after it was sent", second.reached())
	})

	step(t, "Then the second prompt reaches the agent within 8 seconds, as the user typed it, and the agent answers it",
		func() {
			g.Expect(second.reached()).To(BeNumerically("<", holdLimit), "how long the second prompt waited")

			text, isPrompt := second.rows[0].promptText()
			g.Expect(isPrompt).To(BeTrue())
			g.Expect(text).To(Equal(askSomethingElse))

			g.Expect(second.reply).NotTo(BeEmpty(), "the agent answered")
		})
}

// unexported constants.
const (
	// askSomethingElse is a prompt that no note bears on.
	askSomethingElse = "What is 17 plus 25? Reply with the number only."
	// jobStartWait is how long the work after a turn has to start.
	jobStartWait = 10 * time.Second
	// slowJob is a job after a turn that takes longer than its bound.
	slowJob = 60 * time.Second
)
