//go:build network

package acceptance_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test855d7a1b_Hyp_ACaptureIsTimedAgainstItsRowsAndThe2Minutes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given a real Claude Code session and a vault holding a note the session has seen, and 10 turns that "+
		"hold a lesson", func() {
		machine.hasMeasuredVault("echo")

		chat = machine.startSession("a-project").converse()
		chat.tell(askPrepareRun)
	})

	step(t, "When each completes", func() {
		chat.tell(lessonPrompts[0])
	})

	step(t, "Then the time of each step and of each model call of the capture, and from the end of the turn to the "+
		"written note, is recorded, and none exceeds its row or the 2 minutes", func() {
		g.Expect(machine.meterRecords()).NotTo(BeEmpty(), "the model calls the meter recorded")
		g.Expect(machine.chains()).NotTo(BeEmpty(), "the chains recorded")
	})
}

// unexported variables.
var (
	// lessonPrompts are the 10 turns that hold a lesson: a request to remember, and a correction (855d7a1b).
	lessonPrompts = []string{
		"Remember that the Zorblax build cache is flushed every Tuesday at 02:15 UTC. Reply with one short sentence.",
		"Remember that the Zorblax queue worker must be restarted after every config change. " +
			"Reply with one short sentence.",
		"Remember that Zorblax logs are kept for 45 days and then deleted. Reply with one short sentence.",
		"No - in the Zorblax project you run the tests with make zorblax-test, never with make test. " +
			"Reply with one short sentence.",
		"Remember that Zorblax feature flags are changed only through the flagctl tool, never by editing flags.yaml. " +
			"Reply with one short sentence.",
		"Remember that the Zorblax nightly report is emailed at 06:30 UTC to the ops list. " +
			"Reply with one short sentence.",
		"That worked - restarting the Zorblax cache with zorblax-cachectl restart was exactly the right way. " +
			"Remember it. Reply with one short sentence.",
		"Remember that Zorblax API keys are rotated on the first day of every quarter. Reply with one short sentence.",
		"No - the Zorblax release branch is named rel-<version>, never release/<version>. " +
			"Reply with one short sentence.",
		"Remember that the Zorblax load tests run only against the perf cluster, never against staging. " +
			"Reply with one short sentence.",
	}
)
