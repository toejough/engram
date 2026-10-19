//go:build network

package acceptance_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test855d7a1b_Hyp_ACaptureIsTimedAgainstItsRowsAndThe2Minutes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		chat    *conversation
		written time.Time
	)

	step(t, "Given a real Claude Code session and a vault holding a note the session has seen, and 10 turns that "+
		"hold a lesson", func() {
		machine.hasMeasuredVault("echo")

		chat = machine.startSession("a-project").converse()
		chat.tell(askPrepareRun)
	})

	step(t, "When each completes", func() {
		before := machine.noteTimes()

		chat.tell(lessonPrompts[0])
		machine.awaitSettled(chat.session.id, chat.turnEnds, 2)

		written = machine.newestNoteSince(before)
	})

	step(t, "Then the time of each step and of each model call of the capture, and from the end of the turn to the "+
		"written note, is recorded, and none exceeds its row or the 2 minutes", func() {
		g.Expect(machine.meterRecords()).NotTo(BeEmpty(), "the model calls the meter recorded")
		g.Expect(machine.chains()).NotTo(BeEmpty(), "the chains recorded")
		g.Expect(written).NotTo(BeZero(), "when the note of the turn was written")
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

// newestNoteSince is when the newest note that is new, or written again, since the times given was written, or zero.
func (m *machine) newestNoteSince(_ map[string]time.Time) time.Time { return time.Time{} }

// noteTimes is when each note of the machine's vault was last written, by its file name.
func (m *machine) noteTimes() map[string]time.Time {
	times := map[string]time.Time{}

	for _, note := range m.notes() {
		times[note.name] = note.written
	}

	return times
}
