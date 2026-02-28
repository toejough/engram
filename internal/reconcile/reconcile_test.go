package reconcile_test

// Tests for ARCH-5: Reconciler (shared component)
// These tests define what reconcile.Reconciler, reconcile.OverlapGate,
// and reconcile.New should do. Nothing compiles yet — RED phase.

//go:generate impgen store.MemoryStore --dependency
//go:generate impgen reconcile.OverlapGate --dependency
//go:generate impgen reconcile.ReconcileRun --target

import (
	"context"
	"testing"

	"engram/internal"
	"engram/internal/reconcile"
	_ "engram/internal/store"
	"github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// --- Generators ---

func genLearning() *rapid.Generator[internal.Learning] {
	return rapid.Custom(func(t *rapid.T) internal.Learning {
		return internal.Learning{
			Title:    rapid.String().Draw(t, "title"),
			Content:  rapid.String().Draw(t, "content"),
			Keywords: rapid.SliceOfN(rapid.String(), 1, 5).Draw(t, "keywords"),
		}
	})
}

func genMemory(id string) *rapid.Generator[internal.Memory] {
	return rapid.Custom(func(t *rapid.T) internal.Memory {
		return internal.Memory{
			ID:       id,
			Title:    rapid.String().Draw(t, "title"),
			Content:  rapid.String().Draw(t, "content"),
			Keywords: rapid.SliceOfN(rapid.String(), 1, 5).Draw(t, "keywords"),
		}
	})
}

// T-27: Empty store → reconciler always creates.
func TestReconciler_NoExistingMemories_Creates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, _ := MockOverlapGate(t)
		learning := genLearning().Draw(t, "learning")
		k := rapid.IntRange(1, 10).Draw(t, "k")

		wrapper := StartReconcileRun(t, reconcile.ReconcileRun,
			mockStore, mockGate, k, context.Background(), learning)

		// FindSimilar returns empty → no candidates
		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.Equal(k),
		).Return(nil, nil)

		// Create should be called for the new memory
		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		wrapper.ReturnsShould(
			gomega.HaveField("Action", "created"),
			gomega.BeNil(),
		)
	})
}

// T-28: Overlap gate says yes → enriches best candidate.
func TestReconciler_OverlapGateSaysYes_Enriches(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, expectGate := MockOverlapGate(t)
		learning := genLearning().Draw(t, "learning")
		existing := genMemory("m_best").Draw(t, "existing")

		wrapper := StartReconcileRun(t, reconcile.ReconcileRun,
			mockStore, mockGate, 3, context.Background(), learning)

		// FindSimilar returns one candidate
		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.Equal(3),
		).Return([]internal.ScoredMemory{{Memory: existing, Score: 0.9}}, nil)

		// OverlapGate says yes for this candidate
		expectGate.Check.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(learning),
			gomega.Equal(existing),
		).Return(true, "overlapping content", nil)

		// Update should be called to enrich the existing memory
		expectStore.Update.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		wrapper.ReturnsShould(
			gomega.HaveField("Action", "enriched"),
			gomega.BeNil(),
		)
	})
}

// T-29: Overlap gate says no for all → creates new.
func TestReconciler_OverlapGateSaysNo_Creates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, expectGate := MockOverlapGate(t)
		learning := genLearning().Draw(t, "learning")
		m1 := genMemory("m_1").Draw(t, "m1")
		m2 := genMemory("m_2").Draw(t, "m2")

		wrapper := StartReconcileRun(t, reconcile.ReconcileRun,
			mockStore, mockGate, 3, context.Background(), learning)

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.Equal(3),
		).Return([]internal.ScoredMemory{
			{Memory: m1, Score: 0.8},
			{Memory: m2, Score: 0.5},
		}, nil)

		// OverlapGate says no for both
		expectGate.Check.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(learning),
			gomega.Equal(m1),
		).Return(false, "different topic", nil)

		expectGate.Check.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(learning),
			gomega.Equal(m2),
		).Return(false, "different topic", nil)

		// Create should be called
		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		wrapper.ReturnsShould(
			gomega.HaveField("Action", "created"),
			gomega.BeNil(),
		)
	})
}

// T-30: Reconciler evaluates at most K candidates via overlap gate.
func TestReconciler_RespectsKBudget(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, expectGate := MockOverlapGate(t)
		learning := genLearning().Draw(t, "learning")
		k := rapid.IntRange(1, 3).Draw(t, "k")

		// Generate exactly k candidates
		candidates := make([]internal.ScoredMemory, k)
		for i := range candidates {
			candidates[i] = internal.ScoredMemory{
				Memory: genMemory(rapid.String().Draw(t, "id")).Draw(t, "candidate"),
				Score:  rapid.Float64Range(0.1, 1.0).Draw(t, "score"),
			}
		}

		wrapper := StartReconcileRun(t, reconcile.ReconcileRun,
			mockStore, mockGate, k, context.Background(), learning)

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.Equal(k),
		).Return(candidates, nil)

		// Gate called for each candidate — all say no
		for _, c := range candidates {
			expectGate.Check.ArgsShould(
				gomega.BeAssignableToTypeOf(context.Background()),
				gomega.Equal(learning),
				gomega.Equal(c.Memory),
			).Return(false, "no overlap", nil)
		}

		// Create called since all rejected
		expectStore.Create.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		).Return(nil)

		wrapper.ReturnsShould(
			gomega.HaveField("Action", "created"),
			gomega.BeNil(),
		)
	})
}

// T-31: Enriched memory has merged keywords and incremented enrichment count.
func TestReconciler_EnrichAddsKeywordsAndContext(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		mockStore, expectStore := MockMemoryStore(t)
		mockGate, expectGate := MockOverlapGate(t)

		existingKW := rapid.SliceOfN(rapid.String(), 1, 3).Draw(t, "existingKW")
		newKW := rapid.SliceOfN(rapid.String(), 1, 3).Draw(t, "newKW")

		existing := internal.Memory{
			ID:              "m_0001",
			Title:           rapid.String().Draw(t, "existingTitle"),
			Content:         rapid.String().Draw(t, "existingContent"),
			Keywords:        existingKW,
			EnrichmentCount: rapid.IntRange(0, 10).Draw(t, "existingEnrichCount"),
		}
		learning := internal.Learning{
			Title:    rapid.String().Draw(t, "learningTitle"),
			Content:  rapid.String().Draw(t, "learningContent"),
			Keywords: newKW,
		}

		wrapper := StartReconcileRun(t, reconcile.ReconcileRun,
			mockStore, mockGate, 3, context.Background(), learning)

		expectStore.FindSimilar.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.BeAssignableToTypeOf(""),
			gomega.Equal(3),
		).Return([]internal.ScoredMemory{{Memory: existing, Score: 0.9}}, nil)

		expectGate.Check.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Equal(learning),
			gomega.Equal(existing),
		).Return(true, "overlapping", nil)

		// Capture the updated memory via GetArgs to verify merged keywords
		updateCall := expectStore.Update.ArgsShould(
			gomega.BeAssignableToTypeOf(context.Background()),
			gomega.Not(gomega.BeNil()),
		)
		updatedMem := updateCall.GetArgs().Mem
		updateCall.Return(nil)

		wrapper.ReturnsShould(
			gomega.HaveField("Action", "enriched"),
			gomega.BeNil(),
		)

		// Verify merged keywords contain all originals
		g.Expect(updatedMem).ToNot(gomega.BeNil())
		kwSet := make(map[string]bool)
		for _, kw := range updatedMem.Keywords {
			kwSet[kw] = true
		}
		for _, kw := range existingKW {
			g.Expect(kwSet).To(gomega.HaveKey(kw))
		}
		for _, kw := range newKW {
			g.Expect(kwSet).To(gomega.HaveKey(kw))
		}
		g.Expect(updatedMem.EnrichmentCount).To(gomega.Equal(existing.EnrichmentCount + 1))
	})
}
