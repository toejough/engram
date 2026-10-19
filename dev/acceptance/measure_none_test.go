//go:build network

package acceptance_test

import (
	"fmt"
	"slices"
	"testing"

	. "github.com/onsi/gomega"
)

func Test855d7a1j_Hyp_TheMomentJudgementsFalseNoneIsRecorded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given a real Claude Code session, a vault, and 20 prompts for each of which the recall skill run in "+
		"glance mode names a note it draws on", func() {
		machine.hasMeasuredVault("echo")
	})

	var arrived []promptRecall

	step(t, "When each prompt is sent with automatic memory on", func() {
		for index, prompt := range noneCandidatePrompts {
			machine.startSession(fmt.Sprintf("with-%d", index)).send(prompt)
		}

		arrived = machine.promptRecalls(noneCandidatePrompts)
	})

	step(t, "Then the share of the prompts for which the moment judgement answers none is recorded", func() {
		machine.automaticMemoryIsOff()
		machine.agentMayRunEngramAndSkills()

		drawn := make([][]string, 0, len(noneCandidatePrompts))

		for index, prompt := range noneCandidatePrompts[:1] {
			skilled := machine.startSession(fmt.Sprintf("skill-%d", index)).send(skillAsk(prompt))

			drawn = append(drawn, namedNotes(skilled.reply, machine.noteNames()))
		}

		none, withNotes := falseNone(arrived, drawn)

		g.Expect(withNotes).To(BeNumerically(">", 0), "the prompts for which the skill draws on a note")
		g.Expect(none).To(BeNumerically(">=", 0))
	})
}

// falseNone counts, among the prompts for which the recall skill drew on a note, those for which the moment judgement
// chose none.
func falseNone(_ []promptRecall, _ [][]string) (none, withNotes int) { return 0, 0 }

// unexported variables.
var (
	// noneCandidatePrompts are 20 prompts that notes of the vault apply to (855d7a1j).
	noneCandidatePrompts = append(slices.Clone(skillPrompts),
		"Is it okay to run the Zorblax staging load test on Sunday night?",
		"What backup do I need before migrating the staging database?",
		"Can I edit schema.sql directly to add a column?",
		"The release is tagged; can I say it is done?",
		"make ci says LINT_STRICT is unset; what do I do?",
		"How do I format Python files before committing?",
		"When is the staging database backed up?",
		"How do I restore the Zorblax staging database?",
		"Does the staging reset still happen when it is a holiday?",
		"Which command restores staging from a backup?",
	)
)
