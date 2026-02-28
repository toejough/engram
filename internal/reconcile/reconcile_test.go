package reconcile_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"
	"pgregory.net/rapid"

	"engram/internal/reconcile"
	"engram/internal/store"
)

func TestRun_CreateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	learning := reconcile.Learning{Content: "test", Keywords: []string{"k"}, Title: "t"}
	mockStore, storeExp := MockStore(t)
	mockGate, _ := MockOverlapGate(t)
	call := StartRun(t, reconcile.Run, ctx, mockStore, mockGate, 3, learning)

	storeExp.FindSimilar.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return(nil, nil)
	storeExp.Create.ArgsShould(match.BeAny, match.BeAny).
		Return(errors.New("create error"))
	call.ReturnsShould(match.BeAny, HaveOccurred())
}

func TestRun_FindSimilarError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	learning := reconcile.Learning{Content: "test", Keywords: []string{"k"}, Title: "t"}
	mockStore, storeExp := MockStore(t)
	mockGate, _ := MockOverlapGate(t)
	call := StartRun(t, reconcile.Run, ctx, mockStore, mockGate, 3, learning)

	storeExp.FindSimilar.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return(nil, errors.New("db error"))
	call.ReturnsShould(match.BeAny, HaveOccurred())
}

func TestRun_OverlapGateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	learning := reconcile.Learning{Content: "test", Keywords: []string{"k"}, Title: "t"}
	existing := store.Memory{ID: "m_1", Confidence: "A", Concepts: []string{}, Keywords: []string{}}
	mockStore, storeExp := MockStore(t)
	mockGate, gateExp := MockOverlapGate(t)
	call := StartRun(t, reconcile.Run, ctx, mockStore, mockGate, 3, learning)

	storeExp.FindSimilar.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return([]store.ScoredMemory{{Memory: existing, Score: 0.9}}, nil)
	gateExp.Check.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return(false, "", errors.New("gate error"))
	call.ReturnsShould(match.BeAny, HaveOccurred())
}

func TestRun_UpdateError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	learning := reconcile.Learning{Content: "test", Keywords: []string{"k"}, Title: "t"}
	existing := store.Memory{ID: "m_1", Confidence: "A", Concepts: []string{}, Keywords: []string{}}
	mockStore, storeExp := MockStore(t)
	mockGate, gateExp := MockOverlapGate(t)
	call := StartRun(t, reconcile.Run, ctx, mockStore, mockGate, 3, learning)

	storeExp.FindSimilar.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return([]store.ScoredMemory{{Memory: existing, Score: 0.9}}, nil)
	gateExp.Check.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return(true, "overlap", nil)
	storeExp.Update.ArgsShould(match.BeAny, match.BeAny).
		Return(errors.New("update error"))
	call.ReturnsShould(match.BeAny, HaveOccurred())
}

func TestT27_EmptyStoreCreatesNewMemory(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any Learning l, any K in [1,10]
		learning := genLearning().Draw(rt, "learning")
		limit := rapid.IntRange(1, 10).Draw(rt, "k")
		ctx := context.Background()

		// When test calls ReconcileRun with (store, gate, K, any ctx, learning)
		mockStore, storeExp := MockStore(t)
		mockGate, _ := MockOverlapGate(t)
		call := StartRun(t, reconcile.Run, ctx, mockStore, mockGate, limit, learning)

		// Then ReconcileRun calls store.FindSimilar with (any ctx, any string, equal to K)
		// Given nil, nil error — When store.FindSimilar responds with (nil, nil)
		storeExp.FindSimilar.ArgsShould(match.BeAny, match.BeAny, Equal(limit)).
			Return(nil, nil)

		// Then ReconcileRun calls store.Create with (any ctx, non-nil Memory)
		// Given nil error — When store.Create responds with nil
		storeExp.Create.ArgsShould(match.BeAny, Not(BeNil())).
			Return(nil)

		// Then ReconcileRun returns (ReconcileResult{Action: "created"}, nil error)
		call.ReturnsShould(
			HaveField("Action", Equal("created")),
			Not(HaveOccurred()),
		)
	})
}

func TestT28_OverlapGateSaysYesEnrichesBestCandidate(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any Learning l, any existing Memory
		learning := genLearning().Draw(rt, "learning")
		existing := genMemory().Draw(rt, "existing")
		ctx := context.Background()

		// When test calls ReconcileRun with (store, gate, 3, any ctx, learning)
		mockStore, storeExp := MockStore(t)
		mockGate, gateExp := MockOverlapGate(t)
		call := StartRun(t, reconcile.Run, ctx, mockStore, mockGate, 3, learning)

		// Then ReconcileRun calls store.FindSimilar with (any ctx, any string, 3)
		// Given [ScoredMemory{existing, 0.9}], nil error
		storeExp.FindSimilar.ArgsShould(match.BeAny, match.BeAny, Equal(3)).
			Return([]store.ScoredMemory{{Memory: existing, Score: 0.9}}, nil)

		// Then ReconcileRun calls gate.Check with (any ctx, l, existing)
		// Given true, "overlapping content", nil error
		gateExp.Check.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
			Return(true, "overlapping content", nil)

		// Then ReconcileRun calls store.Update with (any ctx, non-nil Memory)
		// Given nil error — When store.Update responds with nil
		storeExp.Update.ArgsShould(match.BeAny, Not(BeNil())).
			Return(nil)

		// Then ReconcileRun returns (ReconcileResult{Action: "enriched"}, nil error)
		call.ReturnsShould(
			HaveField("Action", Equal("enriched")),
			Not(HaveOccurred()),
		)
	})
}

func TestT29_OverlapGateSaysNoForAllCreatesNew(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any Learning, any two existing Memories
		learning := genLearning().Draw(rt, "learning")
		memory1 := genMemory().Draw(rt, "m1")
		memory2 := genMemory().Draw(rt, "m2")
		ctx := context.Background()

		// When test calls ReconcileRun with (store, gate, 3, any ctx, learning)
		mockStore, storeExp := MockStore(t)
		mockGate, gateExp := MockOverlapGate(t)
		call := StartRun(t, reconcile.Run, ctx, mockStore, mockGate, 3, learning)

		// Then ReconcileRun calls store.FindSimilar with (any ctx, any string, 3)
		// Given [ScoredMemory{memory1, 0.8}, ScoredMemory{memory2, 0.5}], nil error
		storeExp.FindSimilar.ArgsShould(match.BeAny, match.BeAny, Equal(3)).
			Return([]store.ScoredMemory{
				{Memory: memory1, Score: 0.8},
				{Memory: memory2, Score: 0.5},
			}, nil)

		// Then ReconcileRun calls gate.Check with (any ctx, learning, memory1)
		// Given false, "different topic", nil error
		gateExp.Check.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
			Return(false, "different topic", nil)
		// Then ReconcileRun calls gate.Check with (any ctx, learning, memory2)
		// Given false, "different topic", nil error
		gateExp.Check.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
			Return(false, "different topic", nil)

		// Then ReconcileRun calls store.Create with (any ctx, non-nil Memory)
		// Given nil error — When store.Create responds with nil
		storeExp.Create.ArgsShould(match.BeAny, Not(BeNil())).
			Return(nil)

		// Then ReconcileRun returns (ReconcileResult{Action: "created"}, nil error)
		call.ReturnsShould(
			HaveField("Action", Equal("created")),
			Not(HaveOccurred()),
		)
	})
}

func TestT30_RespectsKBudget(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		// Given any Learning, any K in [1,3], exactly K candidates
		learning := genLearning().Draw(rt, "learning")
		budget := rapid.IntRange(1, 3).Draw(rt, "k")
		ctx := context.Background()

		candidates := make([]store.ScoredMemory, budget)

		for i := range budget {
			m := genMemory().Draw(rt, "candidate")
			candidates[i] = store.ScoredMemory{Memory: m, Score: float64(budget-i) * 0.3}
		}

		// When test calls ReconcileRun with (store, gate, K, any ctx, learning)
		mockStore, storeExp := MockStore(t)
		mockGate, gateExp := MockOverlapGate(t)
		call := StartRun(t, reconcile.Run, ctx, mockStore, mockGate, budget, learning)

		// Then ReconcileRun calls store.FindSimilar with (any ctx, any string, K)
		// Given K candidates, nil error
		storeExp.FindSimilar.ArgsShould(match.BeAny, match.BeAny, Equal(budget)).
			Return(candidates, nil)

		// Then ReconcileRun calls gate.Check exactly K times, once per candidate
		// Given false for each — When gate.Check responds with (false, "no overlap", nil)
		for range budget {
			gateExp.Check.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
				Return(false, "no overlap", nil)
		}

		// Then ReconcileRun calls store.Create with (any ctx, non-nil Memory)
		// Given nil error — When store.Create responds with nil
		storeExp.Create.ArgsShould(match.BeAny, Not(BeNil())).
			Return(nil)

		// Then ReconcileRun returns (ReconcileResult{Action: "created"}, nil error)
		call.ReturnsShould(
			HaveField("Action", Equal("created")),
			Not(HaveOccurred()),
		)
	})
}

func TestT31_EnrichAddsKeywordsAndIncrementsCount(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	rapid.Check(t, func(rt *rapid.T) {
		// Given any Learning l with keywords, any existing Memory with keywords and EnrichmentCount N
		learning := genLearning().Draw(rt, "learning")
		existing := genMemory().Draw(rt, "existing")
		ctx := context.Background()

		// When test calls ReconcileRun with (store, gate, 3, any ctx, learning)
		mockStore, storeExp := MockStore(t)
		mockGate, gateExp := MockOverlapGate(t)
		call := StartRun(t, reconcile.Run, ctx, mockStore, mockGate, 3, learning)

		// Then ReconcileRun calls store.FindSimilar with (any ctx, any string, 3)
		// Given [ScoredMemory{existing, 0.9}], nil error
		storeExp.FindSimilar.ArgsShould(match.BeAny, match.BeAny, Equal(3)).
			Return([]store.ScoredMemory{{Memory: existing, Score: 0.9}}, nil)

		// Then ReconcileRun calls gate.Check with (any ctx, l, existing)
		// Given true, "overlapping", nil error
		gateExp.Check.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
			Return(true, "overlapping", nil)

		// Then ReconcileRun calls store.Update with (any ctx, updated Memory)
		updateCall := storeExp.Update.ArgsShould(match.BeAny, Not(BeNil()))
		args := updateCall.GetArgs()
		updated := args.M

		// Verify: Keywords contains all of existingKW AND all of newKW
		for _, kw := range existing.Keywords {
			g.Expect(updated.Keywords).To(ContainElement(kw))
		}

		for _, kw := range learning.Keywords {
			g.Expect(updated.Keywords).To(ContainElement(kw))
		}
		// Verify: EnrichmentCount equals N+1
		g.Expect(updated.EnrichmentCount).To(Equal(existing.EnrichmentCount + 1))

		// Given nil error — When store.Update responds with nil
		updateCall.Return(nil)

		// Then ReconcileRun returns (ReconcileResult{Action: "enriched"}, nil error)
		call.ReturnsShould(
			HaveField("Action", Equal("enriched")),
			Not(HaveOccurred()),
		)
	})
}

func genLearning() *rapid.Generator[reconcile.Learning] {
	return rapid.Custom(func(t *rapid.T) reconcile.Learning {
		return reconcile.Learning{
			Content:  rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(t, "content"),
			Keywords: []string{rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "keyword")},
			Title:    rapid.StringMatching(`[A-Za-z ]{5,30}`).Draw(t, "title"),
		}
	})
}

func genMemory() *rapid.Generator[store.Memory] {
	return rapid.Custom(func(t *rapid.T) store.Memory {
		return store.Memory{
			ID:              rapid.StringMatching(`m_[0-9a-f]{8}`).Draw(t, "id"),
			Title:           rapid.StringMatching(`[A-Za-z ]{5,30}`).Draw(t, "title"),
			Content:         rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(t, "content"),
			Keywords:        []string{rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "keyword")},
			Confidence:      rapid.SampledFrom([]string{"A", "B", "C"}).Draw(t, "confidence"),
			Concepts:        []string{},
			EnrichmentCount: rapid.IntRange(0, 10).Draw(t, "enrichment_count"),
		}
	})
}
