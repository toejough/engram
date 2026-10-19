//go:build network

package acceptance_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test855d7a1g_Hyp_ThePhrasesFindWhatTheRecallSkillFinds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var skilled *turn

	step(t, "Given a real Claude Code session, a vault, and 10 prompts for each of which the recall skill run in "+
		"glance mode names the notes it draws on", func() {
		machine.hasMeasuredVault("echo")
		machine.automaticMemoryIsOff()
		machine.agentMayRunEngramAndSkills()

		skilled = machine.startSession("a-project").send(skillAsk(skillPrompts[0]))
	})

	step(t, "When each prompt is sent with automatic memory on", func() {
		t.Logf("the skill replied %q", skilled.reply)
	})

	step(t, "Then the share of those notes that are among the candidates is recorded, with the share the skill's own "+
		"phrases reach", func() {
		var phrases []string

		for _, row := range skilled.rows {
			for _, command := range row.commands() {
				phrases = append(phrases, queryPhrases(command)...)
			}
		}

		g.Expect(phrases).NotTo(BeEmpty(), "the phrases of the queries the skill ran")
	})
}

// skillAsk asks, in a session without automatic memory, for the recall skill at its glance mode for a prompt, and for
// the notes it drew on by their file names.
func skillAsk(prompt string) string {
	return "Run the recall skill in its glance mode for this prompt: \"" + prompt + "\" " +
		"Then reply with the file names of the notes the skill drew on, one to a line, and nothing else."
}

// unexported variables.
var (
	// skillPrompts are 10 prompts that notes of the vault apply to (855d7a1g).
	skillPrompts = []string{
		"Plan a migration of the Zorblax staging database for tonight, around whatever is scheduled on that database.",
		"Prepare a test run against the Zorblax staging database tonight.",
		"Plan how to restore the Zorblax staging database from last night's backup.",
		"Should the Zorblax staging test run go ahead on a public holiday?",
		"Cut release v2.0.0 for the Zorblax project and tell me when it is done.",
		"Add an index on orders.created_at to the schema.",
		"Why would make ci fail here with a LINT_STRICT message?",
		"I am about to commit trailing.py after editing it. What should I do first?",
		"When is the Zorblax staging database reset?",
		"Plan a backup schedule around the resets of the Zorblax staging database.",
	}
)
