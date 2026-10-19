//go:build network

package acceptance_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test855d7a1e_Hyp_TheChainsOfATurnsToolCallsAreTimed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	bare := newMachine(t)
	mod := newMachine(t)

	var without, with *turn

	step(t, "Given a real Claude Code session and a vault, and a turn of 10 tool calls, some of them in parallel",
		func() {
			bare.hasClaudeCode()
			bare.agentMayRun("echo", "sleep")
			mod.hasMeasuredVault("echo", "sleep")
		})

	step(t, "When the turn runs with the mod and without it", func() {
		without = bare.startSession("a-project").converse().tell(tenCalls)

		chat := mod.startSession("a-project").converse()
		with = chat.tell(tenCalls)

		mod.awaitSettled(chat.session.id, chat.turnEnds, 1)
	})

	step(t, "Then the time added to each tool call and to the turn is recorded, with the sum of the times of the "+
		"chains", func() {
		g.Expect(toolSpans(without.rows)).NotTo(BeEmpty(), "the tool calls of the turn without the mod")
		g.Expect(toolSpans(with.rows)).NotTo(BeEmpty(), "the tool calls of the turn with the mod")
		g.Expect(with.reply).NotTo(BeEmpty())
	})
}

// unexported constants.
const (
	// tenCalls is a turn of 10 tool calls in five messages of the agent: four at once, one, three at once, one and one.
	tenCalls = "Use the Bash tool for these ten commands, and reply with the word done at the end. In your first " +
		"message make four calls at once: `sleep 1 && echo c1`, `sleep 1 && echo c2`, `sleep 1 && echo c3` and " +
		"`sleep 1 && echo c4`. Then make this call alone: `sleep 2 && echo c5`. Then make three calls at once: " +
		"`sleep 1 && echo c6`, `echo c7` and `sleep 1 && echo c8`. Then make this call alone: `echo c9`. Then make " +
		"this call alone: `sleep 2 && echo c10`."
)

// toolSpan is a tool call of the agent in a turn's rows, from the agent's message that holds it to the row that holds
// its result.
type toolSpan struct {
	id      string
	tool    string
	command string
	start   time.Time
	end     time.Time
}

// toolSpans are the tool calls of the turn's rows that have a result, in the order the agent made them.
func toolSpans(_ []transcriptRow) []toolSpan { return nil }
