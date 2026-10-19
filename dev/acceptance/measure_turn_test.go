//go:build network

package acceptance_test

import (
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"
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
			bare.hasMeter()
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
		bareSpans, modSpans := toolSpans(without.rows), toolSpans(with.rows)
		chains := slices.DeleteFunc(mod.chains(), func(each chain) bool {
			return each.steps[0].cue() == cueTurnEnd || each.steps[0].cue() == "prompt"
		})
		rows := make([][]string, 0, len(modSpans))
		saved := make([]time.Duration, 0, len(modSpans))

		// a call is let run while its chain does (855b102): the call and its chain end together, so the chain is
		// hidden behind the call by as much as the shorter of the two
		for _, call := range modSpans {
			name := callName(call.command)
			alone, _ := meterCallNamed(bare.meterRecords(), name)
			chained := chainNamed(chains, name)
			saving := min(alone, chained.took())

			saved = append(saved, saving)

			rows = append(rows, []string{name, chained.steps[0].cue(), secs(alone), secs(chained.took()), secs(saving)})
		}

		bareGaps, modGaps := requestGaps(without.rows), requestGaps(with.rows)
		gapRows := make([][]string, 0, len(modGaps))

		var delayed time.Duration

		for index, gap := range modGaps {
			var inBatch []time.Duration

			for _, each := range chains {
				if each.start().After(gap.since) && !each.start().After(gap.at) {
					inBatch = append(inBatch, each.took())
				}
			}

			delay := gap.delay
			if index < len(bareGaps) {
				delay -= bareGaps[index].delay
			}

			delayed += delay

			gapRows = append(gapRows, []string{
				strconv.Itoa(index + 1), strconv.Itoa(gap.calls), secs(gap.delay), secs(longest(inBatch)),
				strconv.Itoa(len(inBatch)),
			})
		}

		var sum time.Duration

		for _, each := range chains {
			sum += each.took()
		}

		phaseWithout, phaseWith := toolPhase(without.rows), toolPhase(with.rows)
		added := phaseWith - phaseWithout

		t.Logf("the tool calls, in seconds (the call alone as the meter saw it without the mod, the chain of the call "+
			"with the mod, and the time the chain is hidden behind the call, the shorter of the two):\n%s",
			table([]string{"call", "chain of", "call alone", "chain", "hidden"}, rows))
		t.Logf("the time from the last result of each batch of calls to the next request to the model, in seconds, "+
			"with the mod (without it the same gaps are %v), the longest chain that began in it and how many chains "+
			"began:\n%s", delays(bareGaps), table([]string{"batch", "calls", "gap", "longest chain", "chains"}, gapRows))
		t.Logf("the turn of the tool calls (the first call to the agent's last message): without the mod %s s, with it "+
			"%s s, added %s s; the gaps with the mod add %s s to those without; the whole turn from the prompt: without "+
			"%s s, with %s s; the %d chains of the tool calls and batches took %s s in all; blocks delivered within "+
			"the turn: %d; hidden per call: median %s s",
			secs(phaseWithout), secs(phaseWith), secs(added), secs(delayed), secs(without.repliedAfter()),
			secs(with.repliedAfter()), len(chains), secs(sum), len(with.withinTurn()), secs(median(saved)))

		g.Expect(mod.meterRecords()).NotTo(BeEmpty(), "the model calls the meter recorded")
		g.Expect(unmatchedRequests(mod.chains())).To(BeEmpty(),
			"the steps that printed a request and have no model call")
		g.Expect(modSpans).To(HaveLen(len(bareSpans)), "the tool calls with and without the mod")
		g.Expect(len(modSpans)).To(BeNumerically(">=", tenToolCalls), "the tool calls of the turn")
		g.Expect(added).To(BeNumerically("<", sum), "the time added to the turn against the sum of the chains")
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
	// tenToolCalls is how many tool calls the turn of tenCalls holds.
	tenToolCalls = 10
)

// unexported variables.
var (
	callPattern = regexp.MustCompile(`c[0-9]+`)
)

// requestGap is the wait between the last result of the tool calls the agent made and the request to the model that
// followed them.
type requestGap struct {
	calls int           // the results the gap follows
	delay time.Duration // from the last of them to the request
	since time.Time     // the request before
	at    time.Time     // the request after
}

// toolSpan is a tool call of the agent in a turn's rows, from the agent's message that holds it to the row that holds
// its result.
type toolSpan struct {
	id      string
	command string
	start   time.Time
	end     time.Time
}

// callName is the name cN that a command of tenCalls carries, or "".
func callName(command string) string { return callPattern.FindString(command) }

// chainNamed is the chain of the tool event that holds the call named, or an empty chain.
func chainNamed(chains []chain, name string) chain {
	for _, each := range chains {
		if callName(each.steps[0].field("event")) == name {
			return each
		}
	}

	return chain{steps: []stepRun{{}}}
}

// delays are the delays of the gaps in seconds.
func delays(gaps []requestGap) []string {
	out := make([]string, 0, len(gaps))

	for _, gap := range gaps {
		out = append(out, secs(gap.delay))
	}

	return out
}

// meterCallNamed is how long the meter saw the tool call that holds the call named take: from the call to its answer
// at the meter's place in the hooks.
func meterCallNamed(records []meterRecord, name string) (time.Duration, bool) {
	for _, record := range records {
		if record.Kind == meterTool && callName(record.Command) == name {
			return time.Duration(record.Took) * time.Millisecond, true
		}
	}

	return 0, false
}

// requestGaps are the gaps of the turn's rows, one for each request to the model that follows tool results.
func requestGaps(rows []transcriptRow) []requestGap {
	var (
		gaps    []requestGap
		current requestGap
		last    time.Time
		since   time.Time
	)

	for _, row := range rows {
		when, err := time.Parse(time.RFC3339Nano, row.Timestamp)
		if err != nil {
			continue
		}

		switch {
		case row.Type == "user" && strings.Contains(string(row.Message.Content), "tool_result"):
			current.calls++
			last = when
		case row.Type == "api-request":
			if current.calls > 0 {
				current.delay, current.since, current.at = when.Sub(last), since, when
				gaps = append(gaps, current)
				current = requestGap{}
			}

			since = when
		}
	}

	return gaps
}

// toolPhase is how long the turn's tool calls took: from the first one to the agent's last message.
func toolPhase(rows []transcriptRow) time.Duration {
	spans := toolSpans(rows)
	if len(spans) == 0 {
		return 0
	}

	var last time.Time

	for _, row := range rows {
		if at, err := time.Parse(time.RFC3339Nano, row.Timestamp); err == nil && row.Type == "assistant" {
			last = at
		}
	}

	return last.Sub(spans[0].start)
}

// toolSpans are the tool calls of the turn's rows that have a result, in the order the agent made them.
func toolSpans(rows []transcriptRow) []toolSpan {
	var (
		spans   []toolSpan
		results = map[string]time.Time{}
	)

	for _, row := range rows {
		when, err := time.Parse(time.RFC3339Nano, row.Timestamp)
		if err != nil {
			continue
		}

		var blocks []struct {
			Type      string `json:"type"`
			ID        string `json:"id"`
			ToolUseID string `json:"tool_use_id"` //nolint:tagliatelle // the host spells a tool result in snake_case
			Input     struct {
				Command string `json:"command"`
			} `json:"input"`
		}

		if json.Unmarshal(row.Message.Content, &blocks) != nil {
			continue
		}

		for _, block := range blocks {
			switch {
			case row.Type == "assistant" && block.Type == "tool_use":
				spans = append(spans, toolSpan{id: block.ID, command: block.Input.Command, start: when})
			case row.Type == "user" && block.Type == "tool_result":
				results[block.ToolUseID] = when
			}
		}
	}

	for index := range spans {
		spans[index].end = results[spans[index].id]
	}

	return slices.DeleteFunc(spans, func(span toolSpan) bool { return span.end.IsZero() })
}
