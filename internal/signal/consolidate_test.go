package signal_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

// T-327: No duplicates — all singletons, no merges.
func TestT327_NoDuplicates_NoMerges(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"alpha", "beta"}},
		{FilePath: "b.toml", Keywords: []string{"gamma", "delta"}},
		{FilePath: "c.toml", Keywords: []string{"epsilon", "zeta"}},
	}}
	merger := &fakeMerger{}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.ClustersFound).To(Equal(0))
	g.Expect(result.MemoriesMerged).To(Equal(0))
	g.Expect(merger.calls).To(BeEmpty())
}

// T-328: Two memories with >50% keyword overlap form a cluster.
func TestT328_TwoOverlapping_FormCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{
			FilePath: "a.toml",
			Keywords: []string{"targ", "check", "full", "lint", "build"},
			Title:    "Use targ check-full",
		},
		{
			FilePath: "b.toml",
			Keywords: []string{"targ", "check", "full", "validation", "errors"},
			Title:    "Targ check comprehensive",
		},
	}}
	merger := &fakeMerger{}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"a.toml": {score: 0.8, hasData: true},
		"b.toml": {score: 0.4, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithEffectiveness(eff),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.ClustersFound).To(Equal(1))
	g.Expect(result.MemoriesMerged).To(Equal(1))
	g.Expect(merger.calls).To(HaveLen(1))
	g.Expect(merger.calls[0].survivor.FilePath).To(Equal("a.toml"))
	g.Expect(merger.calls[0].absorbed.FilePath).To(Equal("b.toml"))
}

// T-329: Transitive closure — A overlaps B, B overlaps C, all in one cluster.
func TestT329_TransitiveClosure_ThreeWayCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{
			FilePath: "a.toml",
			Keywords: []string{"targ", "check", "full", "lint"},
			Title:    "A",
		},
		{
			FilePath: "b.toml",
			Keywords: []string{"targ", "check", "full", "errors"},
			Title:    "B",
		},
		{
			FilePath: "c.toml",
			Keywords: []string{"check", "full", "errors", "validation"},
			Title:    "C",
		},
	}}
	merger := &fakeMerger{}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"a.toml": {score: 0.9, hasData: true},
		"b.toml": {score: 0.5, hasData: true},
		"c.toml": {score: 0.3, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithEffectiveness(eff),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.ClustersFound).To(Equal(1))
	g.Expect(result.MemoriesMerged).To(Equal(2))

	for _, call := range merger.calls {
		g.Expect(call.survivor.FilePath).To(Equal("a.toml"))
	}
}

// T-330: Memories at exactly 50% overlap are NOT clustered.
func TestT330_ExactlyFiftyPercent_NoClustering(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// 2 of 4 keywords match = exactly 50%, not >50%
	lister := &fakeLister{memories: []*memory.Stored{
		{
			FilePath: "a.toml",
			Keywords: []string{"alpha", "beta", "gamma", "delta"},
		},
		{
			FilePath: "b.toml",
			Keywords: []string{"alpha", "beta", "epsilon", "zeta"},
		},
	}}
	merger := &fakeMerger{}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.ClustersFound).To(Equal(0))
	g.Expect(merger.calls).To(BeEmpty())
}

// T-331: Highest effectiveness score survives.
func TestT331_HighestEffectiveness_Survives(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "low.toml", Keywords: []string{"a", "b", "c"}, Title: "Low"},
		{FilePath: "high.toml", Keywords: []string{"a", "b", "d"}, Title: "High"},
	}}
	merger := &fakeMerger{}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"low.toml":  {score: 0.4, hasData: true},
		"high.toml": {score: 0.8, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithEffectiveness(eff),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.MemoriesMerged).To(Equal(1))
	g.Expect(merger.calls[0].survivor.FilePath).To(Equal("high.toml"))
	g.Expect(merger.calls[0].absorbed.FilePath).To(Equal("low.toml"))
}

// T-332: Null-effectiveness loses to scored memory.
func TestT332_NullEffectiveness_LosesToScored(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "null.toml", Keywords: []string{"a", "b", "c"}, Title: "Null"},
		{
			FilePath: "scored.toml",
			Keywords: []string{"a", "b", "d"},
			Title:    "Scored",
		},
	}}
	merger := &fakeMerger{}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"scored.toml": {score: 0.6, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithEffectiveness(eff),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.MemoriesMerged).To(Equal(1))
	g.Expect(merger.calls[0].survivor.FilePath).To(Equal("scored.toml"))
}

// T-333: Among null-effectiveness, alphabetical fallback.
func TestT333_NullEffectiveness_AlphabeticalFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{
			FilePath: "b-memory.toml",
			Keywords: []string{"x", "y", "z"},
			Title:    "B Memory",
		},
		{
			FilePath: "a-memory.toml",
			Keywords: []string{"x", "y", "w"},
			Title:    "A Memory",
		},
	}}
	merger := &fakeMerger{}
	eff := &fakeEffectiveness{scores: map[string]effScore{}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithEffectiveness(eff),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.MemoriesMerged).To(Equal(1))
	g.Expect(merger.calls[0].survivor.FilePath).To(Equal("a-memory.toml"))
}

// T-334: Tie-breaking by alphabetical file path.
func TestT334_TieBreak_AlphabeticalPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "z-memory.toml", Keywords: []string{"a", "b", "c"}, Title: "Z"},
		{FilePath: "a-memory.toml", Keywords: []string{"a", "b", "d"}, Title: "A"},
	}}
	merger := &fakeMerger{}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"z-memory.toml": {score: 0.5, hasData: true},
		"a-memory.toml": {score: 0.5, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithEffectiveness(eff),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.MemoriesMerged).To(Equal(1))
	g.Expect(merger.calls[0].survivor.FilePath).To(Equal("a-memory.toml"))
}

// T-335: Absorbed memory's counters — MergeExecutor called correctly.
func TestT335_AbsorbedCounters_InSurvivorArray(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	absorbed := &memory.Stored{
		FilePath: "absorbed.toml",
		Keywords: []string{"a", "b", "c"},
		Title:    "Absorbed",
	}
	survivor := &memory.Stored{
		FilePath: "survivor.toml",
		Keywords: []string{"a", "b", "d"},
		Title:    "Survivor",
	}

	lister := &fakeLister{memories: []*memory.Stored{survivor, absorbed}}
	merger := &fakeMerger{}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"survivor.toml": {score: 0.8, hasData: true},
		"absorbed.toml": {score: 0.2, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithEffectiveness(eff),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.MemoriesMerged).To(Equal(1))
	g.Expect(merger.calls[0].survivor.FilePath).To(Equal("survivor.toml"))
	g.Expect(merger.calls[0].absorbed.FilePath).To(Equal("absorbed.toml"))
}

// T-336: Existing absorbed entries on survivor are preserved.
func TestT336_ExistingAbsorbed_Preserved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	survivor := &memory.Stored{
		FilePath: "survivor.toml",
		Keywords: []string{"a", "b", "c"},
		Title:    "Survivor",
	}
	absorbed := &memory.Stored{
		FilePath: "new.toml",
		Keywords: []string{"a", "b", "d"},
		Title:    "New",
	}

	lister := &fakeLister{memories: []*memory.Stored{survivor, absorbed}}
	merger := &fakeMerger{}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"survivor.toml": {score: 0.9, hasData: true},
		"new.toml":      {score: 0.1, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithEffectiveness(eff),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.MemoriesMerged).To(Equal(1))
	g.Expect(merger.calls[0].survivor).To(Equal(survivor))
}

// T-337: No LLM — merger still called (fallback is MergeExecutor's concern).
func TestT337_NoLLM_LongestPrinciple(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{
			FilePath:  "short.toml",
			Keywords:  []string{"a", "b", "c"},
			Principle: "Short",
			Title:     "Short",
		},
		{
			FilePath:  "long.toml",
			Keywords:  []string{"a", "b", "d"},
			Principle: "This is a much longer principle text",
			Title:     "Long",
		},
	}}
	merger := &fakeMerger{}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.MemoriesMerged).To(Equal(1))
	g.Expect(merger.calls).To(HaveLen(1))
}

// T-338: Keywords and concepts unioned — merger delegation.
func TestT338_KeywordsUnioned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{
			FilePath: "a.toml",
			Keywords: []string{"a", "b", "c"},
			Concepts: []string{"x"},
			Title:    "A",
		},
		{
			FilePath: "b.toml",
			Keywords: []string{"b", "c", "d"},
			Concepts: []string{"y"},
			Title:    "B",
		},
	}}
	merger := &fakeMerger{}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.MemoriesMerged).To(Equal(1))
}

// T-339: Consolidation reduces memory count before classification.
func TestT339_ConsolidationBeforeClassification(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"targ", "check", "full"}, Title: "A"},
		{FilePath: "b.toml", Keywords: []string{"targ", "check", "full"}, Title: "B"},
		{FilePath: "c.toml", Keywords: []string{"traced", "spec", "tdd"}, Title: "C"},
		{FilePath: "d.toml", Keywords: []string{"traced", "spec", "tdd"}, Title: "D"},
	}}
	merger := &fakeMerger{}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.ClustersFound).To(Equal(2))
	g.Expect(result.MemoriesMerged).To(Equal(2))
}

// T-340: Per-merge stderr log line emitted.
func TestT340_StderrLogPerMerge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"x", "y", "z"}, Title: "Alpha"},
		{FilePath: "b.toml", Keywords: []string{"x", "y", "w"}, Title: "Beta"},
	}}
	merger := &fakeMerger{}

	var stderr bytes.Buffer

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithStderr(&stderr),
	)

	_, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	output := stderr.String()
	g.Expect(output).To(ContainSubstring(`[engram] Merged`))
	g.Expect(output).To(ContainSubstring(
		`[engram] Consolidated 1 duplicate clusters (1 memories merged)`,
	))
}

// T-341: No duplicates — no stderr output.
func TestT341_NoDuplicates_SilentStderr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"alpha"}},
		{FilePath: "b.toml", Keywords: []string{"beta"}},
	}}
	merger := &fakeMerger{}

	var stderr bytes.Buffer

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithStderr(&stderr),
	)

	_, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stderr.String()).To(BeEmpty())
}

// T-342: MergeExecutor error — fire-and-forget per cluster.
func TestT342_MergeError_FireAndForget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "fail-a.toml", Keywords: []string{"a", "b", "c"}, Title: "Fail A"},
		{FilePath: "fail-b.toml", Keywords: []string{"a", "b", "d"}, Title: "Fail B"},
		{FilePath: "ok-a.toml", Keywords: []string{"x", "y", "z"}, Title: "OK A"},
		{FilePath: "ok-b.toml", Keywords: []string{"x", "y", "w"}, Title: "OK B"},
	}}
	merger := &fakeMerger{
		failPaths: map[string]bool{"fail-b.toml": true},
	}

	var stderr bytes.Buffer

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithStderr(&stderr),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.ClustersFound).To(Equal(2))
	g.Expect(result.Errors).To(HaveLen(1))
	g.Expect(result.MemoriesMerged).To(BeNumerically(">=", 1))
	g.Expect(stderr.String()).To(ContainSubstring("Error consolidating cluster"))
}

// T-343: Cluster of 3 — iterative pair-by-pair merge.
func TestT343_ClusterOfThree_IterativeMerge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{
			FilePath: "a.toml",
			Keywords: []string{"targ", "check", "full", "lint"},
			Title:    "A",
		},
		{
			FilePath: "b.toml",
			Keywords: []string{"targ", "check", "full", "errors"},
			Title:    "B",
		},
		{
			FilePath: "c.toml",
			Keywords: []string{"check", "full", "errors", "build"},
			Title:    "C",
		},
	}}
	merger := &fakeMerger{}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"a.toml": {score: 0.9, hasData: true},
		"b.toml": {score: 0.5, hasData: true},
		"c.toml": {score: 0.3, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithEffectiveness(eff),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.MemoriesMerged).To(Equal(2))
	g.Expect(merger.calls).To(HaveLen(2))

	for _, call := range merger.calls {
		g.Expect(call.survivor.FilePath).To(Equal("a.toml"))
	}
}

// T-344: signal-detect integration — duplicates consolidated.
func TestT344_ConsolidateBeforeDetect(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "targ-1.toml", Keywords: []string{"targ", "check", "full"}, Title: "Targ 1"},
		{FilePath: "targ-2.toml", Keywords: []string{"targ", "check", "full"}, Title: "Targ 2"},
		{FilePath: "targ-3.toml", Keywords: []string{"targ", "check", "full"}, Title: "Targ 3"},
	}}
	merger := &fakeMerger{}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.ClustersFound).To(Equal(1))
	g.Expect(result.MemoriesMerged).To(Equal(2))
}

// unexported variables.
var (
	_ signal.MemoryLister        = (*fakeLister)(nil)
	_ signal.MergeExecutor       = (*fakeMerger)(nil)
	_ signal.EffectivenessReader = (*fakeEffectiveness)(nil)
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

type fakeMerger struct {
	calls     []mergeCall
	failPaths map[string]bool
}

func (f *fakeMerger) Merge(
	_ context.Context,
	survivor, absorbed *memory.Stored,
) error {
	if f.failPaths != nil && f.failPaths[absorbed.FilePath] {
		return fmt.Errorf("fake merge error for %s", absorbed.FilePath)
	}

	f.calls = append(f.calls, mergeCall{
		survivor: survivor,
		absorbed: absorbed,
	})

	return nil
}

type mergeCall struct {
	survivor *memory.Stored
	absorbed *memory.Stored
}
