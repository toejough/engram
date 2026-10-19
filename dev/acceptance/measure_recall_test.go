//go:build network

package acceptance_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test855d7a1a_Hyp_ARecallChainIsTimed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given a real Claude Code session and a vault, and 20 prompts", func() {
		machine.hasMeasuredVault("echo")

		chat = machine.startSession("a-project").converse()
	})

	step(t, "When each is sent", func() {
		chat.tell(recallPrompts[0])
	})

	step(t, "Then the depth the moment judgement chose, the time of each step and of each model call, and of the "+
		"chain, is recorded, and no wait exceeds the 2 minutes", func() {
		g.Expect(machine.meterRecords()).NotTo(BeEmpty(), "the model calls the meter recorded")
		g.Expect(machine.chains()).NotTo(BeEmpty(), "the chains recorded")
	})
}

// unexported variables.
var (
	// recallPrompts are the 20 prompts of the measurements of the chain of a prompt: tasks the vault's notes apply to,
	// decisions that cannot be undone, and prompts no note applies to (855d7a1a).
	recallPrompts = []string{
		askPrepareRun,
		askPlanMigration,
		askStagingReset,
		askPlanFreeze,
		askPlanCutover,
		"Plan how to restore the Zorblax staging database from last night's backup. Reply with one short sentence.",
		"I am about to commit trailing.py after editing it. What should I do first? Reply with one short sentence.",
		"Cut release v2.0.0 for the Zorblax project and tell me when it is done. Reply with one short sentence.",
		"Add an index on orders.created_at to the schema. Reply with one short sentence.",
		"Why would make ci fail here with a LINT_STRICT message? Reply with one short sentence.",
		"Should the Zorblax staging test run go ahead on a public holiday? Reply with one short sentence.",
		"What is 17 times 3? Reply with just the number.",
		askThanks,
		"Rename the variable total to sum in a small example. Reply with one short sentence.",
		"Plan the irreversible deletion of all production data older than two years, due at the end of the month. " +
			"Reply with one short sentence.",
		"Plan a test run against the Zorblax staging database for tomorrow night. Reply with one short sentence.",
		"Summarize in one sentence what a database migration needs.",
		"Plan the staging deploys for the week of the production cutover freeze. Reply with one short sentence.",
		"Draft the steps to tag and publish a release of the Zorblax project. Reply with one short sentence.",
		"Ok, that is all for now. Reply with one short word.",
	}
)

// hasMeasuredVault is a machine on which the user has run engram update, with a vault of eight facts and a feedback, a
// searchable earlier conversation, the steps recorded, and the agent allowed to run the commands named.
func (m *machine) hasMeasuredVault(agentMayRun ...string) {
	m.t.Helper()

	m.hasEngram()
	m.hasClaudeCode()
	m.agentMayRun(agentMayRun...)

	for _, note := range []fact{
		stagingReset, stagingBackup, stagingRestore, stagingHoliday, migrationBackup, releaseChangelog, schemaMigrations,
		makeCIStrict,
	} {
		m.learnsFact(note)
	}

	m.learnsFeedback(formatPython)
	m.hasConversation(freezeConversation)
	m.stepsAreRecorded()
	m.mustSucceed(m.update(), "engram update")
}
