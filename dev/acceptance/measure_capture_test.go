//go:build network

package acceptance_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test855d7a1b_Hyp_ACaptureIsTimedAgainstItsRowsAndThe2Minutes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		chat     *conversation
		answered []time.Time
		written  []time.Time
	)

	step(t, "Given a real Claude Code session and a vault holding a note the session has seen, and 10 turns that "+
		"hold a lesson", func() {
		machine.hasMeasuredVault("echo")

		chat = machine.startSession("a-project").converse()
		chat.tell(askPrepareRun)
	})

	step(t, "When each completes", func() {
		for index, prompt := range lessonPrompts {
			before := machine.noteTimes()

			chat.tell(prompt)

			// the capture of the turn is over before the next turn is sent
			machine.awaitSettled(chat.session.id, chat.turnEnds, index+2)

			answered = append(answered, chat.turnEnds)
			written = append(written, machine.newestNoteSince(before))
		}
	})

	step(t, "Then the time of each step and of each model call of the capture, and from the end of the turn to the "+
		"written note, is recorded, and none exceeds its row or the 2 minutes", func() {
		captures := chainsOfCue(machine.chains(), cueTurnEnd)
		ends := meterTurnEnds(machine.meterRecords())

		g.Expect(machine.meterRecords()).NotTo(BeEmpty(), "the model calls the meter recorded")
		g.Expect(unmatchedRequests(machine.chains())).To(BeEmpty(),
			"the steps that printed a request and have no model call")
		g.Expect(captures).To(HaveLen(len(lessonPrompts)+1), "the captures of the first turn and of the 10 turns")
		g.Expect(ends).To(HaveLen(len(lessonPrompts)+1), "the turns the meter saw end")

		rows := make([][]string, 0, len(lessonPrompts))
		limits := captureRows()

		var (
			over    = make([]string, 0, len(lessonPrompts))
			notes   int
			toNotes = make([]time.Duration, 0, len(lessonPrompts))
		)

		for index := range lessonPrompts {
			each, ended := captures[index+1], ends[index+1]
			toNote := time.Duration(0)

			if !written[index].IsZero() {
				notes++
				toNote = written[index].Sub(ended)

				toNotes = append(toNotes, toNote)
			}

			rows = append(rows, []string{
				strconv.Itoa(index + 1), secs(each.start().Sub(ended)), secs(each.stepTook(stepMoment)),
				secs(each.callTook(stepMoment)), secs(each.stepTook(stepJudge)), secs(each.callTook(stepJudge)),
				secs(each.stepTook(stepPlace)), secs(each.callTook(stepPlace)), secs(each.stepTook(stepWrite)),
				secs(toNote), secs(written[index].Sub(answered[index])),
			})

			over = append(over, limits.exceeded(index+1, each, each.start().Sub(ended), toNote)...)
		}

		t.Logf("the captures of the 10 turns, in seconds (the rows: the call for the ID 5, moment 5, its call 10, judge 10, "+
			"its call 30, place 5, its call 15, write 20, all 115; from the end of the turn to the note 120):\n%s",
			table([]string{"turn", "end to moment", "moment", "moment call", "judge", "judge call", "place", "place call",
				"write", "end to note", "result to note"}, rows))
		t.Logf("%d of the 10 turns wrote a note; from the end of the turn to the note: median %s s, longest %s s; "+
			"waits over their rows: %v", notes, secs(median(toNotes)), secs(longest(toNotes)), over)

		g.Expect(notes).To(BeNumerically(">", 0), "the turns that wrote a note")
		g.Expect(over).To(BeEmpty(), "the waits over their rows")
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

// captureLimits are the rows of the waits of a capture (855d2b), in seconds.
type captureLimits struct {
	idCall, moment, momentCall, judge, judgeCall, place, placeCall, write, all, toNote time.Duration
}

// exceeded names each wait of a capture that passes its row, and the time from the end of the turn to the note when it
// passes the 2 minutes.
func (l captureLimits) exceeded(turn int, each chain, spawn, toNote time.Duration) []string {
	var over []string

	for _, wait := range []struct {
		name  string
		took  time.Duration
		limit time.Duration
	}{
		{"call for the ID", spawn, l.idCall}, {"moment", each.stepTook(stepMoment), l.moment},
		{"moment call", each.callTook(stepMoment), l.momentCall},
		{"judge", each.stepTook(stepJudge), l.judge}, {"judge call", each.callTook(stepJudge), l.judgeCall},
		{"place", each.stepTook(stepPlace), l.place}, {"place call", each.callTook(stepPlace), l.placeCall},
		{"write", each.stepTook(stepWrite), l.write}, {"capture", each.took(), l.all}, {"end to note", toNote, l.toNote},
	} {
		if wait.took > wait.limit {
			over = append(over, fmt.Sprintf("turn %d %s %s s over %s s", turn, wait.name, secs(wait.took), secs(wait.limit)))
		}
	}

	return over
}

// newestNoteSince is when the newest note that is new, or written again, since the times given was written, or zero.
func (m *machine) newestNoteSince(before map[string]time.Time) time.Time {
	var newest time.Time

	for name, written := range m.noteTimes() {
		if written.After(before[name]) && written.After(newest) {
			newest = written
		}
	}

	return newest
}

// noteTimes is when each note of the machine's vault was last written, by its file name.
func (m *machine) noteTimes() map[string]time.Time {
	times := map[string]time.Time{}

	for _, note := range m.notes() {
		times[note.name] = note.written
	}

	return times
}

// captureRows are the rows of 855d2b.
func captureRows() captureLimits {
	return captureLimits{
		idCall: 5 * time.Second, moment: 5 * time.Second, momentCall: 10 * time.Second, judge: 10 * time.Second,
		judgeCall: 30 * time.Second, place: 5 * time.Second, placeCall: 15 * time.Second, write: 20 * time.Second,
		all: 115 * time.Second, toNote: measureBound,
	}
}

// meterTurnEnds are the times the meter saw a turn of the main loop end, oldest first.
func meterTurnEnds(records []meterRecord) []time.Time {
	var ends []time.Time

	for _, record := range records {
		if record.Kind == meterTurn {
			ends = append(ends, time.UnixMilli(record.At))
		}
	}

	return ends
}
