package auto_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

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

// 793d24b4: the lookup with the chunk index holds no one, so it has a deadline of its own, above the 95th percentile
// of the query with the index on the notes and the index of the real vault, six at once, 23.0 s (793b425); the lookup
// that holds the prompt is the one the 8 s hold bounds.
func Test793d24b4_TheDeadlineOfTheLookupWithTheChunkIndexIsAboveItsMeasuredPercentile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(auto.DeadlineNearest).To(BeNumerically(">", perfDeepP95))
	g.Expect(auto.DeadlineNearest).To(BeNumerically(">", perfHoldCeiling), "the pass is not bounded by the hold")
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

// 793d36b3: the relevance judgement was cut at 3 s and its 95th percentile is 3.44 s: one lookup in twenty that found
// candidates was cut, delivered nothing, and warned the user that the System timed out. Its deadline is above the
// percentile, and the lookup and the judgement together stay in the 8 s hold.
func Test793d36b3_APromptAtThe95thPercentileStillGetsItsNoteWithinTheHoldCeiling(t *testing.T) {
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

// 793d36b3: the lookup of a moment is cut at its deadline, and the deadline sits above the 95th percentile of the query
// with a margin of one and a half (793b408). The percentile is that of the query as the System issues it, on the
// notes of a real vault of 2,189 notes, six at once; the same query with the vault's chunk index, which the System
// does not leave out, took 12.3 s alone and 18.5 s six at once, and no deadline in the 8 s hold covers that.
func Test793d36b3_TheDeadlineOfAMomentQueryIsAtLeastOneAndAHalfTimesItsMeasuredPercentile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	margin := time.Duration(float64(perfQueryP95) * perfQueryMargin)

	g.Expect(auto.DeadlineMomentQuery).To(BeNumerically(">=", margin))
}

// unexported constants.
const (
	perfClassifyP95      = 2980
	perfDeepP95          = 23100 * time.Millisecond
	perfHoldCeiling      = 8000 * time.Millisecond
	perfQueryMargin      = 1.5
	perfQueryP95         = 1420 * time.Millisecond
	perfRelevanceP95     = 3440
	perfSubagentFile     = "/home/.claude/projects/p/s/subagents/a1.jsonl"
	perfSubagentLast     = "Noted: targ test."
	perfSubagentUserLine = `{"type":"user","message":{"content":"I ran go test. No, run targ test, not go test."}}`
	perfTranscriptLagMs  = 100
)
