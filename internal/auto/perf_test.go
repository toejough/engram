package auto_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

// 793d22b3: the relevance judgement was cut at 3 s and its 95th percentile is 3.44 s: one lookup in twenty that found
// candidates was cut, delivered nothing, and warned the user that the System timed out. Its deadline is above the
// percentile, and the lookup and the judgement together stay in the 8 s hold.
func Test793d22b3_APromptAtThe95thPercentileStillGetsItsNoteWithinTheHoldCeiling(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["query"] = func([]string) (auto.Output, error) { return auto.Output{Stdout: NoteQuery}, nil }
	world.Model = Replies(map[string]string{"relevance": "[1]"})

	world.Do(t, map[string]any{
		"kind": "lookup", "moment": "prompt", "text": "Is the rename done?", "step": "the prompt lookup",
	})

	g.Expect(world.Commands).NotTo(BeEmpty())
	g.Expect(world.Requests).To(HaveLen(1))

	if len(world.Commands) == 0 || len(world.Requests) != 1 {
		return
	}

	timeout := time.Duration(world.Requests[0].TimeoutMs) * time.Millisecond

	g.Expect(world.Requests[0].TimeoutMs).To(BeNumerically(">", perfRelevanceP95), "cut before its p95")
	g.Expect(world.Commands[0].Deadline + timeout).To(BeNumerically("<=", perfHoldCeiling))
}

// 793d23b2: the classification of the in-turn cue must not be cut at its 95th percentile of 2.98 s. The deadline
// is 4 s; the test guards against a tuning that lowers it.
func Test793d23b2_TheInTurnCueAtItsClassifyP95IsNotCut(t *testing.T) {
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
		g.Expect(timeout).To(BeNumerically(">", perfClassifyP95))
	}
}

// 793d27b1: the transcript of a subagent lags its stop event by about 100 ms; the job polled it every 500 ms, so a
// file that completes 100 ms after the stop was read 500 ms after it. The wait the job asks for is at most the lag.
func Test793d27b1_ATranscriptCompleteAfter100msIsReadWithinThatWait(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Files[perfSubagentFile] = perfSubagentUserLine

	world.Do(t, map[string]any{
		"kind": "subagent", "path": perfSubagentFile, "last": perfSubagentLast, "sessionId": "sess-1", "turn": 3,
		"agent": "a1", "step": "subagent capture",
	})

	g.Expect(world.Waits).NotTo(BeEmpty())

	if len(world.Waits) > 0 {
		g.Expect(world.Waits[0]).To(BeNumerically("<=", perfTranscriptLagMs))
	}
}

// unexported constants.
const (
	perfClassifyP95      = 2980
	perfHoldCeiling      = 8000 * time.Millisecond
	perfRelevanceP95     = 3440
	perfSubagentFile     = "/home/.claude/projects/p/s/subagents/a1.jsonl"
	perfSubagentLast     = "Noted: targ test."
	perfSubagentUserLine = `{"type":"user","message":{"content":"I ran go test. No, run targ test, not go test."}}`
	perfTranscriptLagMs  = 100
)
