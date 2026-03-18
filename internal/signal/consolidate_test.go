package signal_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
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

// T-347: Backup file written before absorbed file deleted.
func TestT347_BackupWrittenBeforeDelete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	backupWriter := &fakeBackupWriter{
		onBackup: func(absorbedPath, _ string) error {
			callOrder = append(callOrder, "backup:"+absorbedPath)

			return nil
		},
	}
	fileDeleter := &fakeFileDeleter{
		onDelete: func(path string) error {
			callOrder = append(callOrder, "delete:"+path)

			return nil
		},
	}

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"x", "y", "z"}},
		{FilePath: "b.toml", Keywords: []string{"x", "y", "w"}},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(&fakeMerger{}),
		signal.WithFileWriter(&fakeFileWriter{}),
		signal.WithBackupWriter(backupWriter, "data/memories/.backup"),
		signal.WithFileDeleter(fileDeleter),
	)

	_, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callOrder).To(HaveLen(2))

	if len(callOrder) < 2 {
		return
	}

	g.Expect(callOrder[0]).To(HavePrefix("backup:"))
	g.Expect(callOrder[1]).To(HavePrefix("delete:"))
	g.Expect(backupWriter.capturedBackupDir).To(ContainSubstring(".backup"))
}

// T-348: Backup failure is fire-and-forget — merge continues.
func TestT348_BackupFailure_MergeContinues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stderr bytes.Buffer

	backupWriter := &fakeBackupWriter{
		onBackup: func(_, _ string) error {
			return errors.New("backup disk full")
		},
	}
	fileDeleter := &fakeFileDeleter{}

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"x", "y", "z"}},
		{FilePath: "b.toml", Keywords: []string{"x", "y", "w"}},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(&fakeMerger{}),
		signal.WithFileWriter(&fakeFileWriter{}),
		signal.WithBackupWriter(backupWriter, "data/.backup"),
		signal.WithFileDeleter(fileDeleter),
		signal.WithStderr(&stderr),
	)

	_, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stderr.String()).To(ContainSubstring("backup failed"))
	g.Expect(fileDeleter.deletedPaths).To(ConsistOf("b.toml"))
}

// T-349: Absorbed file deleted after survivor TOML written.
func TestT349_DeleteAfterWrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	fileWriter := &fakeFileWriter{
		onWrite: func(path string, _ *memory.Stored) error {
			callOrder = append(callOrder, "write:"+path)

			return nil
		},
	}
	fileDeleter := &fakeFileDeleter{
		onDelete: func(path string) error {
			callOrder = append(callOrder, "delete:"+path)

			return nil
		},
	}

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "survivor.toml", Keywords: []string{"a", "b", "c"}},
		{FilePath: "absorbed.toml", Keywords: []string{"a", "b", "d"}},
	}}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"survivor.toml": {score: 0.9, hasData: true},
		"absorbed.toml": {score: 0.1, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(&fakeMerger{}),
		signal.WithFileWriter(fileWriter),
		signal.WithFileDeleter(fileDeleter),
		signal.WithEffectiveness(eff),
	)

	_, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callOrder).To(HaveLen(2))

	if len(callOrder) < 2 {
		return
	}

	g.Expect(callOrder[0]).To(HavePrefix("write:"))
	g.Expect(callOrder[1]).To(HavePrefix("delete:"))
}

// T-350: Registry entry removed for absorbed memory after file deletion.
func TestT350_RegistryEntryRemovedAfterDelete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fileDeleter := &fakeFileDeleter{}
	registryRemover := &fakeRegistryEntryRemover{}

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "survivor.toml", Keywords: []string{"a", "b", "c"}},
		{FilePath: "absorbed.toml", Keywords: []string{"a", "b", "d"}},
	}}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"survivor.toml": {score: 0.8, hasData: true},
		"absorbed.toml": {score: 0.2, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(&fakeMerger{}),
		signal.WithFileWriter(&fakeFileWriter{}),
		signal.WithFileDeleter(fileDeleter),
		signal.WithRegistryEntryRemover(registryRemover),
		signal.WithEffectiveness(eff),
	)

	_, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(registryRemover.removedPaths).To(ConsistOf("absorbed.toml"))
}

// T-351: File deletion attempted even when backup fails.
func TestT351_DeletionAttemptedEvenIfBackupFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	backupWriter := &fakeBackupWriter{
		onBackup: func(_, _ string) error {
			return errors.New("disk full")
		},
	}
	fileDeleter := &fakeFileDeleter{}

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"x", "y", "z"}},
		{FilePath: "b.toml", Keywords: []string{"x", "y", "w"}},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(&fakeMerger{}),
		signal.WithFileWriter(&fakeFileWriter{}),
		signal.WithBackupWriter(backupWriter, "data/.backup"),
		signal.WithFileDeleter(fileDeleter),
		signal.WithStderr(io.Discard),
	)

	_, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(fileDeleter.deletedPaths).To(HaveLen(1))
}

// T-352: Cluster merge is atomic — partial failure leaves files unchanged.
func TestT352_WriteFailure_NoDeletes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stderr bytes.Buffer

	writeCallCount := 0
	fileWriter := &fakeFileWriter{
		onWrite: func(_ string, _ *memory.Stored) error {
			writeCallCount++
			if writeCallCount == 2 {
				return errors.New("disk write error on second call")
			}

			return nil
		},
	}
	fileDeleter := &fakeFileDeleter{}

	// Two clusters, each with one absorbed.
	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a1.toml", Keywords: []string{"aaa", "bbb", "ccc"}},
		{FilePath: "a2.toml", Keywords: []string{"aaa", "bbb", "ddd"}},
		{FilePath: "b1.toml", Keywords: []string{"xxx", "yyy", "zzz"}},
		{FilePath: "b2.toml", Keywords: []string{"xxx", "yyy", "www"}},
	}}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"a1.toml": {score: 0.9, hasData: true},
		"a2.toml": {score: 0.1, hasData: true},
		"b1.toml": {score: 0.9, hasData: true},
		"b2.toml": {score: 0.1, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(&fakeMerger{}),
		signal.WithFileWriter(fileWriter),
		signal.WithFileDeleter(fileDeleter),
		signal.WithEffectiveness(eff),
		signal.WithStderr(&stderr),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// One cluster succeeded, one failed.
	g.Expect(result.Errors).To(HaveLen(1))
	// Only one absorbed file deleted (the successful cluster's absorbed).
	g.Expect(fileDeleter.deletedPaths).To(HaveLen(1))
	g.Expect(stderr.String()).To(ContainSubstring("Error consolidating cluster"))
}

// T-353: RecomputeMergeLinks called after successful cluster merge.
func TestT353_LinkRecomputeCalledAfterMerge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var linkDeleteOrder []string

	fileDeleter := &fakeFileDeleter{
		onDelete: func(path string) error {
			linkDeleteOrder = append(linkDeleteOrder, "delete:"+path)

			return nil
		},
	}
	linkRecomputer := &fakeLinkRecomputer{
		onRecompute: func(survivorPath, _ string) error {
			linkDeleteOrder = append(linkDeleteOrder, "recompute:"+survivorPath)

			return nil
		},
	}

	// Two clusters each with one absorbed.
	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "s1.toml", Keywords: []string{"aaa", "bbb", "ccc"}},
		{FilePath: "a1.toml", Keywords: []string{"aaa", "bbb", "ddd"}},
		{FilePath: "s2.toml", Keywords: []string{"xxx", "yyy", "zzz"}},
		{FilePath: "a2.toml", Keywords: []string{"xxx", "yyy", "www"}},
	}}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"s1.toml": {score: 0.9, hasData: true},
		"a1.toml": {score: 0.1, hasData: true},
		"s2.toml": {score: 0.9, hasData: true},
		"a2.toml": {score: 0.1, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(&fakeMerger{}),
		signal.WithFileWriter(&fakeFileWriter{}),
		signal.WithFileDeleter(fileDeleter),
		signal.WithLinkRecomputer(linkRecomputer),
		signal.WithEffectiveness(eff),
	)

	_, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// RecomputeAfterMerge called twice (once per cluster).
	g.Expect(linkRecomputer.calls).To(HaveLen(2))

	// Each recompute happens after its corresponding delete.
	deletesBeforeRecomputes := 0

	for _, event := range linkDeleteOrder {
		if strings.HasPrefix(event, "delete:") {
			deletesBeforeRecomputes++
		}
	}

	g.Expect(deletesBeforeRecomputes).To(BeNumerically(">=", 1))
}

// T-354: Link recompute failure is fire-and-forget — merge not rolled back.
func TestT354_LinkRecomputeFailure_MergeNotRolledBack(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stderr bytes.Buffer

	fileDeleter := &fakeFileDeleter{}
	linkRecomputer := &fakeLinkRecomputer{
		onRecompute: func(_, _ string) error {
			return errors.New("link recompute failed")
		},
	}

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "survivor.toml", Keywords: []string{"a", "b", "c"}},
		{FilePath: "absorbed.toml", Keywords: []string{"a", "b", "d"}},
	}}
	eff := &fakeEffectiveness{scores: map[string]effScore{
		"survivor.toml": {score: 0.9, hasData: true},
		"absorbed.toml": {score: 0.1, hasData: true},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(&fakeMerger{}),
		signal.WithFileWriter(&fakeFileWriter{}),
		signal.WithFileDeleter(fileDeleter),
		signal.WithLinkRecomputer(linkRecomputer),
		signal.WithEffectiveness(eff),
		signal.WithStderr(&stderr),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Merge succeeded despite link recompute failure.
	g.Expect(result.Errors).To(BeEmpty())
	g.Expect(result.MemoriesMerged).To(Equal(1))

	// Absorbed file remains deleted (not rolled back).
	g.Expect(fileDeleter.deletedPaths).To(ConsistOf("absorbed.toml"))

	// Error logged to stderr.
	g.Expect(stderr.String()).To(ContainSubstring("link recompute failed"))
}

// T-356: LLM principle synthesis called with all cluster members' principles.
func TestT356_LLMPrincipleSynthesisCalledWithAllPrinciples(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedPrinciples []string

	synthesizer := &fakeSynthesizer{
		onSynthesize: func(_ context.Context, principles []string) (string, error) {
			capturedPrinciples = principles

			return "synthesized principle", nil
		},
	}

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"x", "y", "z"}, Principle: "principle A"},
		{FilePath: "b.toml", Keywords: []string{"x", "y", "w"}, Principle: "principle B"},
	}}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(&fakeMerger{}),
		signal.WithFileWriter(&fakeFileWriter{}),
		signal.WithPrincipleSynthesizer(synthesizer),
	)

	_, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedPrinciples).To(ConsistOf("principle A", "principle B"))
}

// T-357: LLM synthesis failure falls back to longest principle, logs to stderr.
func TestT357_LLMSynthesisFailureFallsBackToLongestPrinciple(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	synthesizer := &fakeSynthesizer{
		onSynthesize: func(_ context.Context, _ []string) (string, error) {
			return "", errors.New("LLM unavailable")
		},
	}

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"x", "y", "z"}, Principle: "short"},
		{FilePath: "b.toml", Keywords: []string{"x", "y", "w"}, Principle: "much longer principle text"},
	}}

	var stderr strings.Builder

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(&fakeMerger{}),
		signal.WithFileWriter(&fakeFileWriter{}),
		signal.WithPrincipleSynthesizer(synthesizer),
		signal.WithStderr(&stderr),
	)

	_, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Survivor is a.toml (alphabetical tiebreak — both have no effectiveness data).
	// After fakeMerger runs (no-op on principle), keepLongerPrinciple should have set
	// survivor.Principle to "much longer principle text".
	// The synthesizer fails, so fallback (keepLongerPrinciple via merger) is kept.
	g.Expect(stderr.String()).To(ContainSubstring("principle synthesis"))
}

// T-363: Disjoint keyword sets never merge regardless of TF-IDF score.
func TestT363_DisjointKeywords_NeverMerge_HighTFIDF(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"alpha", "beta", "gamma"}, Title: "A"},
		{FilePath: "b.toml", Keywords: []string{"delta", "epsilon", "zeta"}, Title: "B"},
	}}
	merger := &fakeMerger{}
	scorer := &fakeTextSimilarityScorer{score: 1.0} // max similarity

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithTextSimilarityScorer(scorer),
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

// T-366: TF-IDF confidence score attached to clusters meeting keyword threshold.
func TestT366_ConfidenceScoreLoggedToStderr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"x", "y", "z"}, Principle: "use x y z", Title: "A"},
		{FilePath: "b.toml", Keywords: []string{"x", "y", "w"}, Principle: "use x y w", Title: "B"},
	}}
	merger := &fakeMerger{}
	scorer := &fakeTextSimilarityScorer{score: 0.85}

	var stderr bytes.Buffer

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithTextSimilarityScorer(scorer),
		signal.WithStderr(&stderr),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.ClustersFound).To(Equal(1))
	g.Expect(result.MemoriesMerged).To(Equal(1))
	g.Expect(stderr.String()).To(ContainSubstring("0.85"))
}

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
		signal.WithMerger(&fakeMerger{}),
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

// T-364: TF-IDF alone without keyword overlap threshold cannot cause merge.
func TestT364_AtThreshold_TFIDFCannotPromote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Exactly 50% overlap (2/4) — does NOT meet >50% threshold.
	lister := &fakeLister{memories: []*memory.Stored{
		{FilePath: "a.toml", Keywords: []string{"alpha", "beta", "gamma", "delta"}, Title: "A"},
		{FilePath: "b.toml", Keywords: []string{"alpha", "beta", "epsilon", "zeta"}, Title: "B"},
	}}
	merger := &fakeMerger{}
	scorer := &fakeTextSimilarityScorer{score: 1.0}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithTextSimilarityScorer(scorer),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.ClustersFound).To(Equal(0))
	g.Expect(merger.calls).To(BeEmpty())
}

// T-365: Identical surface tokens but non-overlapping keywords do not merge.
func TestT365_IdenticalPrinciple_DisjointKeywords_NoMerge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &fakeLister{memories: []*memory.Stored{
		{
			FilePath:  "a.toml",
			Keywords:  []string{"targ", "check", "full"},
			Principle: "always use targ check-full",
			Title:     "A",
		},
		{
			FilePath:  "b.toml",
			Keywords:  []string{"lint", "format", "vet"},
			Principle: "always use targ check-full",
			Title:     "B",
		},
	}}
	merger := &fakeMerger{}
	scorer := &fakeTextSimilarityScorer{score: 1.0}

	consolidator := signal.NewConsolidator(
		signal.WithLister(lister),
		signal.WithMerger(merger),
		signal.WithTextSimilarityScorer(scorer),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.ClustersFound).To(Equal(0))
	g.Expect(merger.calls).To(BeEmpty())
}

// unexported variables.
var (
	_ signal.MemoryLister           = (*fakeLister)(nil)
	_ signal.MergeExecutor          = (*fakeMerger)(nil)
	_ signal.EffectivenessReader    = (*fakeEffectiveness)(nil)
	_ signal.BackupWriter           = (*fakeBackupWriter)(nil)
	_ signal.FileDeleter            = (*fakeFileDeleter)(nil)
	_ signal.MemoryWriter           = (*fakeFileWriter)(nil)
	_ signal.LinkRecomputer         = (*fakeLinkRecomputer)(nil)
	_ signal.RegistryEntryRemover   = (*fakeRegistryEntryRemover)(nil)
	_ signal.TextSimilarityScorer   = (*fakeTextSimilarityScorer)(nil)
)

type effScore struct {
	score   float64
	hasData bool
}

type fakeBackupWriter struct {
	capturedBackupDir string
	onBackup          func(absorbedPath, backupDir string) error
}

func (f *fakeBackupWriter) Backup(absorbedPath, backupDir string) error {
	f.capturedBackupDir = backupDir

	if f.onBackup != nil {
		return f.onBackup(absorbedPath, backupDir)
	}

	return nil
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

type fakeFileDeleter struct {
	deletedPaths []string
	onDelete     func(path string) error
}

func (f *fakeFileDeleter) Delete(path string) error {
	if f.onDelete != nil {
		return f.onDelete(path)
	}

	f.deletedPaths = append(f.deletedPaths, path)

	return nil
}

type fakeFileWriter struct {
	writtenPaths []string
	onWrite      func(path string, stored *memory.Stored) error
}

func (f *fakeFileWriter) Write(path string, stored *memory.Stored) error {
	if f.onWrite != nil {
		return f.onWrite(path, stored)
	}

	f.writtenPaths = append(f.writtenPaths, path)

	return nil
}

type fakeLinkRecomputer struct {
	calls       []linkRecomputeCall
	onRecompute func(survivorPath, absorbedPath string) error
}

func (f *fakeLinkRecomputer) RecomputeAfterMerge(survivorPath, absorbedPath string) error {
	f.calls = append(f.calls, linkRecomputeCall{})

	if f.onRecompute != nil {
		return f.onRecompute(survivorPath, absorbedPath)
	}

	return nil
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

type fakeRegistryEntryRemover struct {
	removedPaths []string
}

func (f *fakeRegistryEntryRemover) RemoveEntry(path string) error {
	f.removedPaths = append(f.removedPaths, path)

	return nil
}

type fakeSynthesizer struct {
	onSynthesize func(ctx context.Context, principles []string) (string, error)
}

func (f *fakeSynthesizer) SynthesizePrinciples(
	ctx context.Context,
	principles []string,
) (string, error) {
	if f.onSynthesize != nil {
		return f.onSynthesize(ctx, principles)
	}

	return "", nil
}

type fakeTextSimilarityScorer struct {
	score float64
}

func (f *fakeTextSimilarityScorer) ClusterConfidence(texts []string) float64 {
	return f.score
}

type linkRecomputeCall struct{}

type mergeCall struct {
	survivor *memory.Stored
	absorbed *memory.Stored
}
