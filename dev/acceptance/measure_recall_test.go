//go:build network

package acceptance_test

import (
	"slices"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test855d7a1a_Hyp_ARecallChainIsTimed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given a real Claude Code session and a vault, and 20 prompts", func() {
		machine.hasMeasuredVault("echo")

		chat = machine.startMeasuredSession("a-project").converse()
	})

	step(t, "When each is sent", func() {
		for _, prompt := range recallPrompts {
			chat.tell(prompt)
		}

		// the captures of the turns are over before the row returns
		machine.awaitSettled(chat.session.id, chat.turnEnds, len(recallPrompts))
	})

	step(t, "Then the depth the moment judgement chose, the time of each step and of each model call, and of the "+
		"chain, is recorded, and no wait exceeds the 2 minutes", func() {
		chains := machine.chains()
		prompted := chainsOfCue(chains, "prompt")

		t.Logf("the chains of the 20 prompts, in seconds (a step is the run of the engram step, a call is the model call "+
			"that followed it):\n%s", table([]string{"prompt", "depth", "moment", "moment call", "phrases", "phrases call",
			"recall", "candidates call", "deliver", "chain"}, chainRows(prompted)))

		var waits []time.Duration

		for _, each := range chains {
			for _, run := range each.steps {
				waits = append(waits, run.end.Sub(run.start))
			}

			for step := range each.calls {
				waits = append(waits, each.callTook(step))
			}
		}

		depths := map[string]int{}
		totals := make([]time.Duration, 0, len(prompted))

		for _, each := range prompted {
			depths[each.depth()]++

			totals = append(totals, each.took())
		}

		t.Logf("the depths chosen for the 20 prompts: %v; the chains of the prompts: median %s s, longest %s s; "+
			"the longest wait of any step or model call of the session (%d chains of every cue): %s s",
			depths, secs(median(totals)), secs(longest(totals)), len(chains), secs(longest(waits)))

		g.Expect(machine.meterRecords()).NotTo(BeEmpty(), "the model calls the meter recorded")
		g.Expect(unmatchedRequests(chains)).To(BeEmpty(), "the steps that printed a request and have no model call")
		g.Expect(prompted).To(HaveLen(len(recallPrompts)), "the chains of the prompts the meter and the steps recorded")
		g.Expect(longest(waits)).To(BeNumerically("<", measureBound), "the longest wait of the session")
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
// searchable earlier conversation, the steps recorded and the meter installed, and the agent allowed to run the
// commands named.
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
	m.hasMeter()
}

// chainRows are the rows of the table of the chains of the prompts, in seconds.
func chainRows(chains []chain) [][]string {
	rows := make([][]string, 0, len(chains))

	for index, each := range chains {
		rows = append(rows, []string{
			strconv.Itoa(index + 1), each.depth(), secs(each.stepTook(stepMoment)), secs(each.callTook(stepMoment)),
			secs(each.stepTook(stepPhrases)), secs(each.callTook(stepPhrases)), secs(each.stepTook(stepRecall)),
			secs(each.callTook(stepRecall)), secs(each.stepTook(stepDeliver)), secs(each.took()),
		})
	}

	return rows
}

// chainsOfCue are the chains whose first step was given the cue, in the order they began.
func chainsOfCue(chains []chain, cue string) []chain {
	return slices.DeleteFunc(slices.Clone(chains), func(each chain) bool { return each.steps[0].cue() != cue })
}
