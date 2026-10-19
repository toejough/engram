//go:build network

package acceptance_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestCandidateNotes_ListsTheNotesAmongTheCandidatesOfARecallStep(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	output := `{"request":{},"carry":{"candidates":[{"id":"1.a.md","kind":"note"},{"id":"chunk 1","kind":"chunk"},` +
		`{"id":"2.b.md","kind":"note"}]}}`

	g.Expect(candidateNotes(output)).To(Equal([]string{"1.a.md", "2.b.md"}))
	g.Expect(candidateNotes("")).To(BeEmpty())
	g.Expect(namedNotes("1.a.md\n3.c.md", []string{"1.a.md", "2.b.md", "3.c.md"})).To(Equal([]string{"1.a.md", "3.c.md"}))
	g.Expect(overlap([]string{"x", "y"}, []string{"y"})).To(Equal([]string{"y"}))
	g.Expect(ratio(1, 0)).To(BeZero())
	g.Expect(ratio(1, 4)).To(Equal(0.25))
}

func TestChainsOf_ACaptureEndsAtTheWriteOfItsTurn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	origin := time.Unix(1000, 0)
	offset := func(seconds float64) time.Time { return origin.Add(time.Duration(seconds * float64(time.Second))) }
	runs := []stepRun{
		{step: stepMoment, input: map[string]any{"cue": cueTurnEnd, "turn": "t1"}, start: offset(0), end: offset(0.1),
			output: `{"request":{"model":"haiku","system":"a","prompt":"b"},"carry":{"k":1}}`},
		{step: stepJudge, input: map[string]any{"turn": "t1", "carry": map[string]any{"k": float64(1)}}, start: offset(1),
			end: offset(1.1), output: `{"request":{"model":"sonnet","system":"a","prompt":"b"},"carry":{"k":2}}`},
		{step: stepPlace, input: map[string]any{"turn": "t1", "carry": map[string]any{"k": float64(2)}}, start: offset(2),
			end: offset(2.2), output: `{"request":{"model":"sonnet","system":"a","prompt":"b"},"carry":{"k":3}}`},
		{step: stepWrite, input: map[string]any{"turn": "t1", "reply": "x"}, start: offset(4), end: offset(4.5)},
		{step: stepWrite, input: map[string]any{"turn": "t2", "reply": "x"}, start: offset(5), end: offset(5.5)},
	}

	chains := chainsOf(runs, nil)

	g.Expect(chains).To(HaveLen(1))
	g.Expect(chains[0].steps).To(HaveLen(4))
	g.Expect(chains[0].end()).To(Equal(offset(4.5)))
}

func TestChainsOf_ADeliveryAfterAStepThatPrintedNothingIsTheEarliestOfItsCue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	origin := time.Unix(1000, 0)
	offset := func(seconds float64) time.Time { return origin.Add(time.Duration(seconds * float64(time.Second))) }
	same := map[string]any{"session": "s", "cue": "before-tool", "text": "ask", "event": "echo"}
	runs := []stepRun{
		{step: stepMoment, input: same, start: offset(0), end: offset(0.2),
			output: `{"request":{"model":"haiku","system":"a","prompt":"b"},"carry":{"cue":"before-tool"}}`},
		{step: stepPhrases, input: map[string]any{"session": "s", "cue": "before-tool",
			"carry": map[string]any{"cue": "before-tool"}}, start: offset(1), end: offset(1.1)},
		{step: stepDeliver, input: same, start: offset(0.5), end: offset(0.6)},
		{step: stepDeliver, input: same, start: offset(1.2), end: offset(1.3)},
		{step: stepDeliver, input: same, start: offset(2), end: offset(2.1)},
	}

	chains := chainsOf(runs, nil)

	g.Expect(chains).To(HaveLen(1))
	g.Expect(chains[0].steps).To(HaveLen(3), "moment, phrases and the delivery that began after the phrases ended")
	g.Expect(chains[0].end()).To(Equal(offset(1.3)))
	g.Expect(chains[0].depth()).To(BeEmpty(), "the phrases were given no reply here")
}

func TestChainsOf_LinksADeliveryByTheCarryAndMatchesEachModelCall(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	origin := time.Unix(1000, 0)
	offset := func(seconds float64) time.Time { return origin.Add(time.Duration(seconds * float64(time.Second))) }
	runs := []stepRun{
		{step: stepMoment, input: map[string]any{"cue": "prompt", "text": "ask"}, start: offset(0), end: offset(0.2),
			output: `{"request":{"model":"haiku","system":"abc","prompt":"de"},"carry":{"cue":"prompt"}}`},
		{step: stepPhrases, input: map[string]any{"cue": "prompt", "reply": `{"depth":"glance"}`,
			"carry": map[string]any{"cue": "prompt"}}, start: offset(1.5), end: offset(1.6),
			output: `{"request":{"model":"sonnet","system":"abcd","prompt":"e"},"carry":{"cue":"prompt","depth":"glance"}}`},
		{step: stepRecall, input: map[string]any{"cue": "prompt",
			"carry": map[string]any{"cue": "prompt", "depth": "glance"}},
			start: offset(3), end: offset(3.4), output: `{"request":{"model":"sonnet","system":"a","prompt":"b"},"carry":{"x":1}}`},
		{step: stepDeliver, input: map[string]any{"cue": "prompt", "carry": map[string]any{"x": float64(1)}},
			start: offset(5), end: offset(5.1)},
		{step: stepMoment, input: map[string]any{"cue": cueTurnEnd, "turn": "t1"}, start: offset(6), end: offset(6.1)},
	}
	meter := []meterRecord{
		{Kind: meterModel, At: origin.Add(300 * time.Millisecond).UnixMilli(), Took: 1000, System: 3, Prompt: 2},
		{Kind: meterModel, At: origin.Add(1700 * time.Millisecond).UnixMilli(), Took: 1200, System: 4, Prompt: 1},
		{Kind: meterModel, At: origin.Add(3500 * time.Millisecond).UnixMilli(), Took: 1500, System: 1, Prompt: 1},
	}

	chains := chainsOf(runs, meter)

	g.Expect(chains).To(HaveLen(2))
	g.Expect(chains[0].steps).To(HaveLen(4), "moment, phrases, recall and deliver")
	g.Expect(chains[0].took()).To(Equal(5100 * time.Millisecond))
	g.Expect(chains[0].depth()).To(Equal("glance"))
	g.Expect(chains[0].callTook(stepMoment)).To(Equal(time.Second))
	g.Expect(chains[0].callTook(stepPhrases)).To(Equal(1200 * time.Millisecond))
	g.Expect(chains[0].callTook(stepRecall)).To(Equal(1500 * time.Millisecond))
	g.Expect(chains[0].stepTook(stepRecall)).To(Equal(400 * time.Millisecond))
	g.Expect(chains[1].steps).To(HaveLen(1), "a capture whose moment printed nothing ends there")
}

func TestFalseNone_CountsThePromptsWithANoteDrawnOnThatWereJudgedNone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	chains := []promptRecall{{depth: "none"}, {depth: "glance"}, {depth: "none"}, {depth: "deep"}}
	drawn := [][]string{{"a.md"}, {"b.md", "c.md"}, nil, {"d.md"}}

	none, withNotes := falseNone(chains, drawn)

	g.Expect(none).To(Equal(1), "the prompt judged none that the skill drew a note for")
	g.Expect(withNotes).To(Equal(3), "the prompts the skill drew a note for")
}

func TestMedianAndLongest_ReadAListOfDurations(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	durations := []time.Duration{3 * time.Second, time.Second, 2 * time.Second}

	g.Expect(median(durations)).To(Equal(2 * time.Second))
	g.Expect(longest(durations)).To(Equal(3 * time.Second))
	g.Expect(median(nil)).To(BeZero())
	g.Expect(longest(nil)).To(BeZero())
	g.Expect(secs(1500 * time.Millisecond)).To(Equal("1.50"))
	g.Expect(secs(0)).To(Equal("-"))
}

func TestQueryPhrases_ReadsThePhrasesOfAnEngramQueryWhateverTheirQuoting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	command := `engram query --lazy-chunks --text="a b" --phrase="staging reset schedule" --phrase 'database backup'`

	g.Expect(queryPhrases(command)).To(Equal([]string{"staging reset schedule", "database backup"}))
	g.Expect(queryPhrases(`engram query --phrase=plain --phrase=\"x`)).To(Equal([]string{"plain", `"x`}))
	g.Expect(queryPhrases(`ls -la`)).To(BeEmpty())
	g.Expect(shellWords("a  'b c'\td \"e \\\" f\"")).To(Equal([]string{"a", "b c", "d", `e " f`}))
}

func TestTokenUsage_CostsEachKindOfTokenAtItsPrice(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	usage := tokenUsage{Input: 2, Output: 74, CacheRead: 11229, CacheCreate: 6690}

	g.Expect(usage.cost(sonnetPrice)).To(BeNumerically("~", 0.0286269, 1e-9), "the cost the host printed for the turn")
	g.Expect(usage.plus(usage)).To(Equal(tokenUsage{Input: 4, Output: 148, CacheRead: 22458, CacheCreate: 13380}))
}
