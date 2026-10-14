//go:build refactorred

package auto_test

import (
	"encoding/json"
	"github.com/toejough/engram/internal/auto"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

// 793d22b3: the relevance judgement is cut at 3 s today and its 95th percentile is 3.44 s: one lookup in twenty that
// found candidates is cut, delivers nothing, and warns the user that the System timed out.
func Test793d22b3_RefactorredAPromptAtThe95thPercentileStillGetsItsNoteWithinTheHoldCeiling(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{Stdout: NoteQuery}, nil }
	world.Model = Replies(map[string]string{"relevance": "[1]"})

	world.Do(t, map[string]any{"kind": "lookup", "moment": "prompt", "text": "Is the rename done?", "step": "the prompt lookup"})

	g.Expect(world.Commands).NotTo(BeEmpty())
	g.Expect(world.Requests).To(HaveLen(1))

	if len(world.Commands) == 0 || len(world.Requests) != 1 {
		return
	}

	timeout := time.Duration(world.Requests[0].TimeoutMs) * time.Millisecond

	g.Expect(world.Requests[0].TimeoutMs).To(BeNumerically(">", rrRelevanceP95), "the relevance judgement is cut before its p95")
	g.Expect(world.Commands[0].Deadline + timeout).To(BeNumerically("<=", rrHoldCeiling))
}

// 793d23b2: the classification of the in-turn cue must not be cut at its 95th percentile of 2.98 s. In Go the deadline
// is already 4 s, so this gate is green today and guards against a tuning that lowers it.
func Test793d23b2_RefactorredTheInTurnCueAtItsClassifyP95IsNotCut(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Model = Replies(map[string]string{"classify": "[]"})

	world.Do(t, map[string]any{
		"kind": "step", "step": "the step cue", "answer": "Looking at util.",
		"messages": []map[string]any{
			{"role": "user", "text": "Follow the instructions in task.md.", "toolUses": []any{}},
			{"role": "assistant", "text": "Looking at util.", "toolUses": []map[string]any{{"tool": "Grep"}}},
		},
		"toolUses": []map[string]any{{"name": "Grep", "input": map[string]any{"pattern": "util"}}},
	})

	classify := []int{}

	for _, request := range world.Requests {
		if kindOf(request.Prompt) == "classify" {
			classify = append(classify, request.TimeoutMs)
		}
	}

	g.Expect(classify).NotTo(BeEmpty(), "no classify request was made")

	for _, timeout := range classify {
		g.Expect(timeout).To(BeNumerically(">", rrClassifyP95))
	}
}

func Test793d24b2_RefactorredCompactingASubagentLeavesTheLedgerOfTheMainAgentAndEmptiesItsOwn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := rrHoldNotes(t)

	world.Do(t, map[string]any{"kind": "compact", "agent": "a1"})

	ledgers := rrLedgers(t, world)
	g.Expect(ledgers["main"]).NotTo(BeEmpty(), "the main agent's ledger was emptied by the subagent's compaction")
	g.Expect(ledgers["a1"]).To(BeEmpty(), "the subagent's ledger kept the notes it lost")
}

func Test793d24b2_RefactorredCompactingTheMainAgentLeavesTheLedgerOfASubagent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := rrHoldNotes(t)

	world.Do(t, map[string]any{"kind": "compact"})

	ledgers := rrLedgers(t, world)
	g.Expect(ledgers["main"]).To(BeEmpty())
	g.Expect(ledgers["a1"]).NotTo(BeEmpty(), "the subagent's ledger was emptied by the main agent's compaction")
}

// 793d27b1: the transcript of a subagent lags its stop event; the job polls it every 500 ms, so a file that completes
// 100 ms after the stop is read 500 ms after it. The goal: the wait it asks for is at most the lag.
func Test793d27b1_RefactorredATranscriptCompleteAfter100msIsReadWithinThatWait(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Files[rrSubagentFile] = rrSubagentUserLine

	world.Do(t, map[string]any{
		"kind": "subagent", "path": rrSubagentFile, "last": rrSubagentLast, "sessionId": "sess-1", "turn": 3,
		"agent": "a1", "step": "subagent capture",
	})

	g.Expect(world.Waits).NotTo(BeEmpty())

	if len(world.Waits) > 0 {
		g.Expect(world.Waits[0]).To(BeNumerically("<=", rrTranscriptLagMs))
	}
}

// unexported constants.
const (
	rrClassifyP95      = 2980
	rrHoldCeiling      = 8000 * time.Millisecond
	rrRelevanceP95     = 3440
	rrSubagentFile     = "/home/.claude/projects/p/s/subagents/a1.jsonl"
	rrSubagentLast     = "Noted: targ test."
	rrSubagentUserLine = `{"type":"user","message":{"content":"I ran go test. No, run targ test, not go test. Acknowledge."}}`
	rrTranscriptLagMs  = 100
)

func rrHoldNotes(t *testing.T) *World {
	t.Helper()

	world := NewWorld()
	world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{Stdout: NoteQuery}, nil }
	world.Model = Replies(map[string]string{"relevance": "[1]"})

	for _, agent := range []string{"", "a1"} {
		job := map[string]any{"kind": "lookup", "moment": "prompt", "text": "Is the rename done?", "step": "the prompt lookup"}
		if agent != "" {
			job["agent"] = agent
		}

		done := world.Do(t, job)
		if !strings.Contains(done.Block, "note-n") {
			t.Fatalf("the note was not delivered to %q", agent)
		}
	}

	return world
}

// 793d24b2: the job `compact` empties the ledger of the main agent whoever was compacted: after a subagent's
// compaction a note the main agent still holds is delivered to it a second time, and the subagent's own ledger keeps
// notes it lost. The goal: a job `compact` has an `agent` field (empty means main) and empties that agent's ledger.
func rrLedgers(t *testing.T, world *World) map[string][]string {
	t.Helper()

	ledgers := map[string][]string{}
	if err := json.Unmarshal(world.Ledger, &ledgers); err != nil {
		t.Fatalf("the ledger cannot be read: %v", err)
	}

	return ledgers
}
