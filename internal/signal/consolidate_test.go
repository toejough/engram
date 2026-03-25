package signal_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

// T-369: Cluster confidence included in MergePlan for dry-run visibility.
func TestT369_ConfidenceInMergePlan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"x", "y", "z"}, Principle: "use x y z", Title: "A"},
		{FilePath: "b.toml", Keywords: []string{"x", "y", "w"}, Principle: "use x y w", Title: "B"},
	}}
	scorer := &fakeTextSimilarityScorer{score: 0.72}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithTextSimilarityScorer(scorer),
	)

	plans, err := consolidator.Plan(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(plans).To(HaveLen(1))

	if len(plans) < 1 {
		return
	}

	g.Expect(plans[0].Confidence).To(BeNumerically("~", 0.72, 0.001))
}

// unexported variables.
var (
	_ signal.MemoryLister         = (*fakeLister)(nil)
	_ signal.EffectivenessReader  = (*fakeEffectiveness)(nil)
	_ signal.TextSimilarityScorer = (*fakeTextSimilarityScorer)(nil)
)

type effScore struct {
	score   float64
	hasData bool
}

type fakeEffectiveness struct {
	scores map[string]effScore
}

func (f *fakeEffectiveness) EffectivenessScore(
	path string,
) (float64, bool, error) {
	score, ok := f.scores[path]
	if !ok {
		return 0, false, nil
	}

	return score.score, score.hasData, nil
}

// --- Test fakes ---

type fakeLister struct {
	memories []*memory.Stored
	err      error
}

func (f *fakeLister) ListAll(_ context.Context) ([]*memory.Stored, error) {
	return f.memories, f.err
}

type fakeTextSimilarityScorer struct {
	score float64
}

func (f *fakeTextSimilarityScorer) ClusterConfidence(_ []string) float64 {
	return f.score
}
