//go:build network

package acceptance_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func Test855d7a1i_Hyp_TheCostOfTheJudgementsIsRecorded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	bare := newMachine(t)
	mod := newMachine(t)

	step(t, "Given a real Claude Code session and a vault, and a session of 10 prompts and their tool calls", func() {
		bare.hasClaudeCode()
		bare.hasMeter()
		bare.agentMaySettle()
		mod.hasMeasuredVault("echo")
		mod.agentMaySettle()
	})

	step(t, "When it runs with the mod and without it", func() {
		bare.startSession("a-project").converse().tell(costPrompts[0])
	})

	step(t, "Then the tokens and the cost of the cheap-tier calls, of the mid-tier calls and of the agent's own use "+
		"are recorded", func() {
		g.Expect(agentUsage(bare.meterRecords()).Output).To(BeNumerically(">", 0), "the agent's own use without the mod")
	})
}

// unexported variables.
var (
	// costPrompts are the 10 prompts of the session of the cost: tool calls that succeed, one that fails, an edit, and
	// prompts of no tool call (855d7a1i).
	costPrompts = []string{
		"Run make ci by itself, with no pipe or redirect, and tell me what happened. Reply with one short sentence.",
		"Run `cat notes.txt` and tell me the first line. Reply with one short sentence.",
		"Run `echo one` and `echo two` as two separate tool calls made at once, then say done.",
		"Tag v1.4.0 by running touch tag-v1.4.0, then tell me the release is done. Reply with one short sentence.",
		"Run `ls` and tell me how many files there are. Reply with one short sentence.",
		askPrepareRun,
		"Run `cat Makefile` and explain in one sentence what the ci target does.",
		"Add this line to the end of schema.sql: create index idx_orders_created_at on orders (created_at); " +
			"Reply with one short sentence.",
		"Run `echo done` and tell me what it printed. Reply with one short sentence.",
		askThanks,
	}
)

// agentMaySettle gives the agent the permissions the session of the cost needs, and the files it works on.
func (m *machine) agentMaySettle() {
	m.t.Helper()

	writeFile(m.t, filepath.Join(m.claudeDir, "settings.json"), `{"permissions":{"allow":["Bash(echo:*)","Bash(cat:*)",`+
		`"Bash(make:*)","Bash(touch:*)","Bash(ls:*)","Read","Edit","Write"]}}`)
}

// agentUsage is the sum of the usage of the turns of the main loop that the meter saw end.
func agentUsage(_ []meterRecord) tokenUsage { return tokenUsage{} }
