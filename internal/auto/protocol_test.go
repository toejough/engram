package auto_test

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

// The wire format the mod reads (793b197, 793b370): `requests`, `waitMs` and `state` when a job needs the mod, `done`
// when it is over, and the effects of the round beside them. The mod's own tests script an engine with these names.
func Test793d36b1_TheResponseHasTheNamesTheModReads(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["query"] = noteQuery()

	job := map[string]any{"kind": "lookup", "moment": "prompt", "step": "the prompt lookup", "text": "is it done?"}

	var first map[string]any

	g.Expect(json.Unmarshal(runOnce(t, world, job), &first)).To(Succeed())
	g.Expect(first).To(HaveKey("requests"))
	g.Expect(first).To(HaveKey("state"))
	g.Expect(first).NotTo(HaveKey("done"))
	g.Expect(first).To(HaveKey("counts"))

	requests, _ := first["requests"].([]any)
	g.Expect(requests).To(HaveLen(1))

	request, _ := requests[0].(map[string]any)
	g.Expect(request).To(HaveLen(5))

	for _, key := range []string{"model", "effort", "prompt", "maxTokens", "timeoutMs"} {
		g.Expect(request).To(HaveKey(key))
	}

	job["state"], job["answers"] = first["state"], []any{Answer("[1]")}

	var second map[string]any

	g.Expect(json.Unmarshal(runOnce(t, world, job), &second)).To(Succeed())

	done, _ := second["done"].(map[string]any)
	g.Expect(done).To(HaveKey("block"))
	g.Expect(done).To(HaveKey("ledger"))
	g.Expect(second).To(HaveKey("log"))
	g.Expect(second["counts"]).To(Equal(map[string]any{"delivered": 1.0, "added": 0.0, "amended": 0.0, "curated": 0.0}))
}

func Test793d36b2_AJobWhoseRequestChangedBetweenItsRoundsFailsAndDeliversNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["query"] = noteQuery()
	job := map[string]any{"kind": "lookup", "moment": "prompt", "step": "the prompt lookup", "text": "is it done?"}

	first := world.round(t, job)
	job["state"], job["answers"], job["text"] = first.State, world.relay(first), "is it ready?"
	again := world.round(t, job)

	g.Expect(again.Done).NotTo(BeNil())

	if again.Done == nil {
		return
	}

	g.Expect(again.Done.Block).To(BeEmpty())
	g.Expect(again.Warnings).To(Equal([]string{"Automatic memory was skipped because of an error (the prompt lookup)"}))
}

func Test793d36b2_ARoundAnswersFromTheRecordWhatAnEarlierRoundDid(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Process["query"] = noteQuery()
	job := map[string]any{"kind": "lookup", "moment": "prompt", "step": "the prompt lookup", "text": "is it done?"}

	first := world.round(t, job)
	g.Expect(first.Requests).To(HaveLen(1))
	g.Expect(world.Names()).To(Equal([]string{"query"}))

	job["state"], job["answers"] = first.State, world.relay(first)
	again := world.round(t, job)

	g.Expect(again.Done).NotTo(BeNil())
	g.Expect(world.Names()).To(Equal([]string{"query"}), "the second round ran the query again")
	g.Expect(world.Requests).To(HaveLen(1))
}

// Test793d36b2: a job takes several rounds, and each is a replay from the start of the job; what the job did in an
// earlier round is answered from the record, never done again, and reported once.
func Test793d36b2_AnEffectIsPerformedOnceWhateverTheNumberOfRounds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	world := NewWorld()
	world.Model = repliesInOrder(lessonsText(lessonWith(nil)))

	world.Do(t, captureOf(turnMessages()))

	g.Expect(world.Rounds).To(BeNumerically(">=", 2), "the job did not need the mod, so no replay was made")
	g.Expect(world.Names()).To(Equal([]string{"query", "learn"}))
	g.Expect(world.Requests).To(HaveLen(1))
	g.Expect(world.Counts.Added).To(Equal(1))
	g.Expect(world.Logs).To(HaveLen(1), "the log line of an effect was reported again by a replay")
}

func runOnce(t *testing.T, world *World, job map[string]any) []byte {
	t.Helper()

	output, err := auto.Run(world.Deps(), asJSON(job))
	if err != nil {
		t.Fatalf("the job failed: %v", err)
	}

	return output
}
