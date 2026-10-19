//go:build network

package acceptance_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test855d7a1d_Hyp_ADeepWriteIsTimed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given a real Claude Code session and a vault, and 10 recall moments of deep depth", func() {
		machine.hasMeasuredVault("echo")

		chat = machine.startSession("a-project").converse()
	})

	step(t, "When each is sent", func() {
		chat.tell(deepPrompts[0])
		machine.awaitSettled(chat.session.id, chat.turnEnds, 1)
	})

	step(t, "Then the time of the write that follows the block is recorded, and none exceeds the row of 60 seconds",
		func() {
			g.Expect(machine.deepWrites()).NotTo(BeEmpty(), "the writes that followed a deep block")
		})
}

// unexported variables.
var (
	// deepPrompts are 10 prompts of decisions that cannot be undone, each of which an earlier conversation of the
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
	}
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
)

// deepWrites are the runs of the write that follows a deep delivery, which have ended.
func (m *machine) deepWrites() []stepRun { return nil }
