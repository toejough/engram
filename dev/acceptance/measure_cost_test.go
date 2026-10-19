//go:build network

package acceptance_test

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
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

	var without, with []*turn

	step(t, "When it runs with the mod and without it", func() {
		without = bare.runCostSession()

		var chat *conversation

		with, chat = mod.runCostSessionOnChat()

		mod.awaitSettled(chat.session.id, chat.turnEnds, len(costPrompts))
	})

	step(t, "Then the tokens and the cost of the cheap-tier calls, of the mid-tier calls and of the agent's own use "+
		"are recorded", func() {
		own, ownBare := agentUsage(mod.meterRecords()), agentUsage(bare.meterRecords())
		chains := mod.chains()
		uses := callUses(chains)
		cheap, mid, cheapTools := tierUse(uses, "haiku", nil), tierUse(uses, "sonnet", nil), tierUse(uses, "haiku", toolCues)

		t.Logf("the model calls of the judgements by cue and step (a call is one request of the mod; tokens are the counts "+
			"the host reported, the cost is at the published price of the alias):\n%s", table([]string{"cue", "step", "model",
			"calls", "input", "output", "cost USD"}, useRows(uses)))
		t.Logf("%s", table([]string{"use", "calls", "input", "output", "cache read", "cache write", "all tokens", "cost USD"},
			[][]string{
				usageRow("agent, without the mod", len(without), ownBare, sonnetPrice),
				usageRow("agent, with the mod", len(with), own, sonnetPrice),
				usageRow("cheap tier (haiku), all calls", cheap.calls, cheap.usage, haikuPrice),
				usageRow("cheap tier, calls of tool calls", cheapTools.calls, cheapTools.usage, haikuPrice),
				usageRow("mid tier (sonnet), all calls", mid.calls, mid.usage, sonnetPrice),
			}))
		t.Logf("the host's own cost of the agent's turns (total_cost_usd): without the mod $%.4f, with it $%.4f; "+
			"prices in USD for a million tokens: sonnet %.2f in, %.2f out, %.2f cache read, %.2f cache write; haiku %.2f "+
			"in, %.2f out, %.2f cache read, %.2f cache write", costOf(without), costOf(with), sonnetPrice.input*perMillion,
			sonnetPrice.output*perMillion, sonnetPrice.cacheRead*perMillion, sonnetPrice.cacheWrite*perMillion,
			haikuPrice.input*perMillion, haikuPrice.output*perMillion, haikuPrice.cacheRead*perMillion,
			haikuPrice.cacheWrite*perMillion)
		t.Logf("the cheap calls of each turn by cue, and the blocks that landed at the same request (855c8):\n%s",
			table([]string{"turn", "prompt", "before-tool", "failure", "batch-done", "turn-end", "blocks", "requests with a block",
				"requests with 2 or more"}, overlapRows(costPrompts, chains, with)))

		g.Expect(own.Output).To(BeNumerically(">", 0), "the agent's own use with the mod")
		g.Expect(cheapTools.usage.Input+cheapTools.usage.Output).To(BeNumerically("<",
			own.Input+own.Output+own.CacheRead+own.CacheCreate), "the tokens of the cheap calls of tool calls against the agent's")
		g.Expect(cheapTools.usage.cost(haikuPrice)).To(BeNumerically("<", own.cost(sonnetPrice)),
			"the cost of the cheap calls of tool calls against the agent's")
	})
}

// unexported constants.
const (
	// perMillion turns the price of a token into the price of a million.
	perMillion = 1e6
)

// unexported variables.
var (
	// toolCues are the cues of the chains of tool calls (855d2c).
	toolCues = []string{"before-tool", "failure", "batch-done"}
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
func agentUsage(records []meterRecord) tokenUsage {
	var sum tokenUsage

	for _, record := range records {
		if record.Kind == meterTurn && record.Usage != nil {
			sum = sum.plus(*record.Usage)
		}
	}

	return sum
}

// callUse is the model calls of a cue, a step and a model, and what they cost in tokens.
type callUse struct {
	cue, step, model string
	calls            int
	usage            tokenUsage
}

// callUses are the model calls of the chains grouped by cue, step and model, in the order they first appear.
func callUses(chains []chain) []callUse {
	var uses []callUse

	for _, each := range chains {
		for _, run := range each.steps {
			call, found := each.calls[run.step]
			if !found || call.Usage == nil {
				continue
			}

			cue := each.steps[0].cue()
			index := slices.IndexFunc(uses, func(use callUse) bool {
				return use.cue == cue && use.step == run.step && use.model == call.Model
			})

			if index < 0 {
				uses = append(uses, callUse{cue: cue, step: run.step, model: call.Model})
				index = len(uses) - 1
			}

			uses[index].calls++
			uses[index].usage = uses[index].usage.plus(*call.Usage)
		}
	}

	return uses
}

// costOf is the cost the host printed for the session after its last turn, in USD: the figure of a result is the whole
// process's so far.
func costOf(turns []*turn) float64 {
	if len(turns) == 0 {
		return 0
	}

	return turns[len(turns)-1].cost
}

// overlapRows are, for each prompt, the cheap calls its turn made by cue, and where the blocks of the turn landed: the
// number of blocks delivered within the turn, the requests to the model that received a block, and those that received
// two or more.
func overlapRows(prompts []string, chains []chain, turns []*turn) [][]string {
	rows := make([][]string, 0, len(prompts))

	for index, prompt := range prompts {
		counts := map[string]int{}

		for _, each := range chains {
			if _, called := each.calls[stepMoment]; called && each.steps[0].field("text") == prompt {
				counts[each.steps[0].cue()]++
			}
		}

		blocks, received, shared := blocksPerRequest(turns[index].rows)

		rows = append(rows, []string{
			strconv.Itoa(index + 1), strconv.Itoa(counts["prompt"]), strconv.Itoa(counts["before-tool"]),
			strconv.Itoa(counts["failure"]), strconv.Itoa(counts["batch-done"]), strconv.Itoa(counts[cueTurnEnd]),
			strconv.Itoa(blocks), strconv.Itoa(received), strconv.Itoa(shared),
		})
	}

	return rows
}

// blocksPerRequest counts the blocks delivered within a turn, the requests to the model that received at least one of
// them and the requests that received two or more.
func blocksPerRequest(rows []transcriptRow) (blocks, received, shared int) {
	pending := 0

	for _, row := range rows {
		switch {
		case row.Type == "attachment" && !row.besideThePrompt():
			pending += len(engramEntries(row.delivered()))
		case row.Type == "api-request" && pending > 0:
			blocks += pending
			received++

			if pending > 1 {
				shared++
			}

			pending = 0
		}
	}

	return blocks + pending, received, shared
}

// runCostSession runs the session of the cost without the mod's chains and returns its turns.
func (m *machine) runCostSession() []*turn {
	turns, _ := m.runCostSessionOnChat()

	return turns
}

// runCostSessionOnChat runs the prompts of the cost in one session and returns the turns and the session.
func (m *machine) runCostSessionOnChat() ([]*turn, *conversation) {
	m.t.Helper()

	project := m.startSession("a-project")
	project.hasFile("Makefile", "ci:\n\t@test \"$$LINT_STRICT\" = 1 || { echo \"make ci: LINT_STRICT is unset\" >&2; exit 1; }\n"+
		"\t@echo ci passed\n")
	project.hasFile("notes.txt", "the first line of the notes\nthe second line\n")
	project.hasFile("schema.sql", "create table orders (id integer primary key, created_at text);\n")

	chat := project.converse()
	turns := make([]*turn, 0, len(costPrompts))

	for _, prompt := range costPrompts {
		turns = append(turns, chat.tell(prompt))
	}

	return turns, chat
}

// tierUse is the model calls of one model alias, and of the cues given when there are any.
func tierUse(uses []callUse, model string, cues []string) callUse {
	total := callUse{model: model}

	for _, use := range uses {
		if use.model == model && (cues == nil || slices.Contains(cues, use.cue)) {
			total.calls += use.calls
			total.usage = total.usage.plus(use.usage)
		}
	}

	return total
}

// usageRow is a row of the table of the uses.
func usageRow(name string, calls int, usage tokenUsage, at price) []string {
	return []string{
		name, strconv.Itoa(calls), strconv.Itoa(usage.Input), strconv.Itoa(usage.Output), strconv.Itoa(usage.CacheRead),
		strconv.Itoa(usage.CacheCreate), strconv.Itoa(usage.Input + usage.Output + usage.CacheRead + usage.CacheCreate),
		fmt.Sprintf("%.4f", usage.cost(at)),
	}
}

// useRows are the rows of the table of the calls by cue and step, with their cost at the price of the alias.
func useRows(uses []callUse) [][]string {
	rows := make([][]string, 0, len(uses))

	for _, use := range uses {
		at := sonnetPrice
		if use.model == "haiku" {
			at = haikuPrice
		}

		rows = append(rows, []string{
			use.cue, use.step, use.model, strconv.Itoa(use.calls), strconv.Itoa(use.usage.Input),
			strconv.Itoa(use.usage.Output), fmt.Sprintf("%.4f", use.usage.cost(at)),
		})
	}

	return rows
}
