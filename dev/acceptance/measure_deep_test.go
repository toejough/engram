//go:build network

package acceptance_test

import (
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test855d7a1d_Hyp_ADeepWriteIsTimed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		chat *conversation
		sent int
	)

	step(t, "Given a real Claude Code session and a vault, and 10 recall moments of deep depth", func() {
		machine.hasMeasuredVault("echo")
		machine.hasConversations(deepConversations...)

		chat = machine.startMeasuredSession("a-project").converse()
	})

	step(t, "When each is sent", func() {
		// the prompts are sent until 10 of them are deep moments: the depth is the moment judgement's to choose
		for _, prompt := range deepPrompts {
			if len(deepChains(machine.chains())) >= deepMoments {
				break
			}

			sent++

			chat.tell(prompt)

			// the write of a deep delivery has ended before the next prompt is sent
			machine.awaitDeepWrites(deepChains(machine.chains()))
		}

		machine.awaitSettled(chat.session.id, chat.turnEnds, sent)
	})

	step(t, "Then the time of the write that follows the block is recorded, and none exceeds the row of 60 seconds",
		func() {
			chains := deepChains(machine.chains())
			writes := machine.deepWrites()
			rows := make([][]string, 0, len(chains))

			var (
				took  []time.Duration
				after []time.Duration
			)

			for index, each := range chains {
				row := []string{strconv.Itoa(index + 1), each.depth(), secs(each.took())}

				if index < len(writes) {
					took = append(took, writes[index].end.Sub(writes[index].start))
					after = append(after, writes[index].end.Sub(each.end()))
					row = append(row, secs(writes[index].start.Sub(each.end())), secs(took[len(took)-1]),
						secs(after[len(after)-1]))
				}

				rows = append(rows, row)
			}

			t.Logf("the deep deliveries, in seconds (the row of the write: 60):\n%s", table([]string{"deep delivery", "depth",
				"chain", "block to write", "write", "block to written"}, rows))
			t.Logf("%d deep deliveries of the %d prompts sent; the depths chosen: %v; the writes: median %s s, longest %s s",
				len(chains), sent, depthsOf(machine.chains()), secs(median(took)), secs(longest(took)))

			g.Expect(machine.meterRecords()).NotTo(BeEmpty(), "the model calls the meter recorded")
			g.Expect(unmatchedRequests(machine.chains())).To(BeEmpty(),
				"the steps that printed a request and have no model call")
			g.Expect(writes).To(HaveLen(len(chains)), "the writes that followed the deep blocks")
			g.Expect(chains).To(HaveLen(deepMoments), "the deep moments")
			g.Expect(chains).NotTo(BeEmpty(), "the deep deliveries")
			g.Expect(longest(took)).To(BeNumerically("<=", deepWriteRow), "the longest write")
		})
}

// unexported constants.
const (
	// deepMoments is how many deep moments the row times (855d7a1d).
	deepMoments = 10
	// deepWriteRow is the row of the write that follows a deep delivery (855d2a).
	deepWriteRow = 60 * time.Second
)

// unexported variables.
var (
	// deepConversations are earlier conversations that nine of deepPrompts apply to: the first prompt applies to
	// freezeConversation (855d7a1d).
	deepConversations = []string{
		"In an earlier session the user said that the ledger database must be dumped with ledger-dump and verified with " +
			"ledger-verify before any schema drop, and that a drop without a verified dump has lost data twice.",
		"In an earlier session the user said that the payments gateway key is rotated on the first of each quarter, and " +
			"that rotating it invalidates every pending refund, so it is never rotated within two days of month-end close.",
		"In an earlier session the user said that customer data deletion requests must be approved by legal before the " +
			"purge job runs, and that the purge cannot be undone.",
		"In an earlier session the user said that the DNS cutover for zorblax.example.com has a TTL of 48 hours, so it " +
			"cannot be rolled back faster than that.",
		"In an earlier session the user said that the warehouse inventory sync deletes remote records that are not in " +
			"the local set, so it must never run from a partial export.",
		"In an earlier session the user said that the orders-v2 Kafka topic keeps 7 days of data, and that reducing " +
			"the retention deletes the older data at once.",
		"In an earlier session the user said that force-pushes to the main branch of the Zorblax repo are forbidden, " +
			"because the release tags hang off it.",
		"In an earlier session the user said that the key of the backup encryption is held by two people, and that a " +
			"re-key must be witnessed by both.",
		"In an earlier session the user said that the Zorblax vendor contract renews on its own unless it is cancelled " +
			"90 days before the end of March.",
	}
	// deepPrompts are 20 prompts of decisions that cannot be undone, the first 10 of which an earlier conversation of the
	// machine applies to (855d7a1d).
	deepPrompts = []string{
		askPlanCutover,
		"Drop the legacy columns from the production ledger schema tomorrow. Reply with one short sentence.",
		"Rotate the payments gateway key on the 30th of this month. Reply with one short sentence.",
		"Run the customer data purge job against production on Friday. Reply with one short sentence.",
		"Switch the zorblax.example.com DNS to the new hosting provider this afternoon. Reply with one short sentence.",
		"Run the warehouse inventory sync from the partial export we have. Reply with one short sentence.",
		"Cut the orders-v2 Kafka topic retention to one day. Reply with one short sentence.",
		"Force-push the cleaned history to the main branch of the Zorblax repo. Reply with one short sentence.",
		"Re-key the backup encryption tonight on my own. Reply with one short sentence.",
		"Let the vendor contract run past March without a decision. Reply with one short sentence.",
		"Delete the production assets bucket today. Reply with one short sentence.",
		"Truncate the production sessions table tonight. Reply with one short sentence.",
		"Revoke every employee's production access at once and rebuild the roles from scratch. " +
			"Reply with one short sentence.",
		"Wipe the old Zorblax staging cluster and its backups. Reply with one short sentence.",
		"Migrate the production database to a new engine over this weekend. Reply with one short sentence.",
		"Delete all git tags older than a year from the Zorblax repo. Reply with one short sentence.",
		"Turn off the production database replication to save cost. Reply with one short sentence.",
		"Reissue every customer API token tomorrow morning. Reply with one short sentence.",
		"Archive and remove the production audit logs from the last two years. Reply with one short sentence.",
		"Cancel the production certificate renewal and replace it by hand. Reply with one short sentence.",
	}
)

// awaitDeepWrites waits until the write of each deep delivery of the chains has ended, for the 2 minutes at most.
func (m *machine) awaitDeepWrites(deep []chain) {
	m.t.Helper()

	deadline := time.Now().Add(measureBound)

	for len(m.deepWrites()) < len(deep) {
		if time.Now().After(deadline) {
			m.t.Fatalf("%d deep deliveries and %d ended writes after %s", len(deep), len(m.deepWrites()), measureBound)
		}

		time.Sleep(capturePoll)
	}
}

// deepWrites are the runs of the write that follows a deep delivery, which have ended, in the order they began: the
// write of a capture is given no cue.
func (m *machine) deepWrites() []stepRun {
	var writes []stepRun

	for _, run := range m.stepRuns() {
		if run.step == stepWrite && run.cue() != "" {
			writes = append(writes, run)
		}
	}

	slices.SortFunc(writes, func(a, b stepRun) int { return a.start.Compare(b.start) })

	return writes
}

// hasConversations makes each text a searchable earlier conversation of its own, as hasConversation does for one.
func (m *machine) hasConversations(texts ...string) {
	m.t.Helper()

	for index, text := range texts {
		source := filepath.Join(m.root, "conversations", "earlier-session-"+strconv.Itoa(index)+".md")
		mkdirAll(m.t, filepath.Dir(source))
		writeFile(m.t, source, text)

		m.mustSucceed(m.engram("ingest", "--markdown", source), "ingest")
	}
}

// deepChains are the chains of the delivery flow that chose the depth deep, in the order they began.
func deepChains(chains []chain) []chain {
	var deep []chain

	for _, each := range chains {
		if each.steps[0].cue() != cueTurnEnd && each.depth() == "deep" {
			deep = append(deep, each)
		}
	}

	return deep
}

// depthsOf are the depths the moment judgement chose for the chains of the prompts, in the order they began.
func depthsOf(chains []chain) []string {
	depths := make([]string, 0, len(chains))

	for _, each := range chains {
		if each.steps[0].cue() == "prompt" {
			depths = append(depths, each.depth())
		}
	}

	return depths
}
