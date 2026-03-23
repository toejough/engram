package signal_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

func TestConsolidateCluster_ArchivesAllMembers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := &memory.MemoryRecord{Title: "mem-a", SourcePath: "/a.toml"}
	memB := &memory.MemoryRecord{Title: "mem-b", SourcePath: "/b.toml"}
	memC := &memory.MemoryRecord{Title: "mem-c", SourcePath: "/c.toml"}

	extracted := &memory.MemoryRecord{
		Title: "principle", SourcePath: "/consolidated.toml",
	}

	archiver := &mockArchiver{}

	consolidator := signal.NewConsolidator(
		signal.WithExtractor(&mockExtractor{result: extracted}),
		signal.WithArchiver(archiver),
	)

	cluster := &signal.ConfirmedCluster{
		Members:   []*memory.MemoryRecord{memA, memB, memC},
		Principle: "shared",
	}

	_, err := consolidator.ConsolidateClusterForTest(context.Background(), cluster)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(archiver.archived).To(ConsistOf("/a.toml", "/b.toml", "/c.toml"))
}

func TestConsolidateCluster_ExtractorError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	errExtract := errors.New("LLM down")

	consolidator := signal.NewConsolidator(
		signal.WithExtractor(&mockExtractor{err: errExtract}),
	)

	cluster := &signal.ConfirmedCluster{
		Members: []*memory.MemoryRecord{
			{Title: "a"}, {Title: "b"}, {Title: "c"},
		},
		Principle: "shared",
	}

	_, err := consolidator.ConsolidateClusterForTest(context.Background(), cluster)
	g.Expect(err).To(MatchError(ContainSubstring("LLM down")))
}

func TestConsolidateCluster_ExtractsAndTransfers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := &memory.MemoryRecord{
		Title: "mem-a", SourcePath: "/a.toml",
		FollowedCount: 2, ContradictedCount: 1, SurfacedCount: 5,
	}
	memB := &memory.MemoryRecord{
		Title: "mem-b", SourcePath: "/b.toml",
		FollowedCount: 3, ContradictedCount: 0, SurfacedCount: 2,
	}
	memC := &memory.MemoryRecord{
		Title: "mem-c", SourcePath: "/c.toml",
		FollowedCount: 1, ContradictedCount: 2, SurfacedCount: 1,
	}

	extracted := &memory.MemoryRecord{
		Title: "consolidated-principle", SourcePath: "/consolidated.toml",
	}

	cluster := &signal.ConfirmedCluster{
		Members:   []*memory.MemoryRecord{memA, memB, memC},
		Principle: "Shared principle",
	}

	archiver := &mockArchiver{}

	consolidator := signal.NewConsolidator(
		signal.WithExtractor(&mockExtractor{result: extracted}),
		signal.WithArchiver(archiver),
	)

	action, err := consolidator.ConsolidateClusterForTest(context.Background(), cluster)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(action.Type).To(Equal(signal.Consolidated))
	g.Expect(action.ConsolidatedMem).NotTo(BeNil())

	if action.ConsolidatedMem == nil {
		return
	}

	// TransferFields sums followed/contradicted from all 3 originals.
	const expectedFollowed = 6 // 2+3+1

	const expectedContradicted = 3 // 1+0+2

	g.Expect(action.ConsolidatedMem.FollowedCount).To(Equal(expectedFollowed))
	g.Expect(action.ConsolidatedMem.ContradictedCount).To(Equal(expectedContradicted))
	g.Expect(action.ConsolidatedMem.Absorbed).To(HaveLen(3))
	g.Expect(action.Archived).To(ConsistOf("mem-a", "mem-b", "mem-c"))
}

func TestConsolidateCluster_NilArchiver_StillWorks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extracted := &memory.MemoryRecord{
		Title: "principle", SourcePath: "/consolidated.toml",
	}

	consolidator := signal.NewConsolidator(
		signal.WithExtractor(&mockExtractor{result: extracted}),
		// No archiver set — should not panic.
	)

	cluster := &signal.ConfirmedCluster{
		Members: []*memory.MemoryRecord{
			{Title: "a", SourcePath: "/a.toml"},
			{Title: "b", SourcePath: "/b.toml"},
			{Title: "c", SourcePath: "/c.toml"},
		},
		Principle: "shared",
	}

	action, err := consolidator.ConsolidateClusterForTest(context.Background(), cluster)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(action.Type).To(Equal(signal.Consolidated))
	g.Expect(action.Archived).To(HaveLen(3))
}

func TestConsolidateCluster_NilExtractor_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	consolidator := signal.NewConsolidator()

	cluster := &signal.ConfirmedCluster{
		Members:   []*memory.MemoryRecord{{Title: "a"}},
		Principle: "shared",
	}

	_, err := consolidator.ConsolidateClusterForTest(context.Background(), cluster)
	g.Expect(err).To(MatchError(signal.ErrNilExtractor))
}

func TestConsolidateCluster_UpdatesExistingConsolidated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// existingConsolidated has Absorbed records — it's already a consolidated memory.
	existingConsolidated := &memory.MemoryRecord{
		Title:      "existing-consolidated",
		SourcePath: "/existing.toml",
		Absorbed: []memory.AbsorbedRecord{
			{From: "/old.toml", SurfacedCount: 10},
		},
		FollowedCount: 5,
	}

	memNew1 := &memory.MemoryRecord{
		Title: "new-1", SourcePath: "/new1.toml",
		FollowedCount: 2,
	}

	memNew2 := &memory.MemoryRecord{
		Title: "new-2", SourcePath: "/new2.toml",
		FollowedCount: 3,
	}

	extracted := &memory.MemoryRecord{
		Title: "updated-principle", SourcePath: "/updated.toml",
	}

	archiver := &mockArchiver{}

	consolidator := signal.NewConsolidator(
		signal.WithExtractor(&mockExtractor{result: extracted}),
		signal.WithArchiver(archiver),
	)

	cluster := &signal.ConfirmedCluster{
		Members:   []*memory.MemoryRecord{existingConsolidated, memNew1, memNew2},
		Principle: "evolved principle",
	}

	action, err := consolidator.ConsolidateClusterForTest(context.Background(), cluster)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(action.ConsolidatedMem).NotTo(BeNil())

	if action.ConsolidatedMem == nil {
		return
	}

	// Only new members are transferred (not the existing consolidated).
	// 2 new originals + 1 pre-existing absorbed = 3 total absorbed records.
	const expectedAbsorbedCount = 3

	g.Expect(action.ConsolidatedMem.Absorbed).To(HaveLen(expectedAbsorbedCount))

	// Only new members are archived; existing consolidated is NOT archived.
	g.Expect(archiver.archived).To(ConsistOf("/new1.toml", "/new2.toml"))
	g.Expect(action.Archived).To(ConsistOf("new-1", "new-2"))
}

func TestConsolidateCluster_WithLinkRecomputer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := &memory.MemoryRecord{Title: "a", SourcePath: "/a.toml"}
	memB := &memory.MemoryRecord{Title: "b", SourcePath: "/b.toml"}
	memC := &memory.MemoryRecord{Title: "c", SourcePath: "/c.toml"}

	extracted := &memory.MemoryRecord{
		Title: "principle", SourcePath: "/consolidated.toml",
	}

	linkRecomputer := &mockLinkRecomputer{}

	consolidator := signal.NewConsolidator(
		signal.WithExtractor(&mockExtractor{result: extracted}),
		signal.WithLinkRecomputer(linkRecomputer),
	)

	cluster := &signal.ConfirmedCluster{
		Members:   []*memory.MemoryRecord{memA, memB, memC},
		Principle: "shared",
	}

	action, err := consolidator.ConsolidateClusterForTest(context.Background(), cluster)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(action.Type).To(Equal(signal.Consolidated))

	// Link recomputer should be called once per original member.
	const expectedRecomputeCalls = 3

	g.Expect(linkRecomputer.callCount).To(Equal(expectedRecomputeCalls))
}

func TestFindCluster_BelowMinSize_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Only 1 candidate: query + 1 = 2, below min 3.
	onlyOneCandidate := []signal.ScoredCandidate{
		{Memory: &memory.MemoryRecord{Title: "similar"}, Score: 0.8},
	}

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&mockScorer{candidates: onlyOneCandidate}),
		signal.WithConfirmer(&mockConfirmer{}),
	)

	query := &memory.MemoryRecord{Title: "test"}
	result := consolidator.FindClusterForTest(context.Background(), query, nil)

	g.Expect(result).To(BeNil())
}

func TestFindCluster_ConfirmedCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := &memory.MemoryRecord{Title: "mem-a", Principle: "DI everywhere"}
	memB := &memory.MemoryRecord{Title: "mem-b", Principle: "Inject interfaces"}
	memC := &memory.MemoryRecord{Title: "mem-c", Principle: "Mock via DI"}

	candidates := []signal.ScoredCandidate{
		{Memory: memB, Score: 0.9},
		{Memory: memC, Score: 0.85},
	}

	confirmedClusters := []signal.ConfirmedCluster{
		{
			Members:   []*memory.MemoryRecord{memA, memB, memC},
			Principle: "Use dependency injection",
		},
	}

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&mockScorer{candidates: candidates}),
		signal.WithConfirmer(&mockConfirmer{clusters: confirmedClusters}),
	)

	result := consolidator.FindClusterForTest(context.Background(), memA, nil)

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Members).To(HaveLen(3))
	g.Expect(result.Principle).To(Equal("Use dependency injection"))
}

func TestFindCluster_ConfirmerError_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	candidates := []signal.ScoredCandidate{
		{Memory: &memory.MemoryRecord{Title: "a"}, Score: 0.9},
		{Memory: &memory.MemoryRecord{Title: "b"}, Score: 0.85},
	}

	errConfirmer := errors.New("LLM unavailable")

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&mockScorer{candidates: candidates}),
		signal.WithConfirmer(&mockConfirmer{err: errConfirmer}),
	)

	query := &memory.MemoryRecord{Title: "test"}
	result := consolidator.FindClusterForTest(context.Background(), query, nil)

	g.Expect(result).To(BeNil())
}

func TestFindCluster_ConfirmerRejectsAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	candidates := []signal.ScoredCandidate{
		{Memory: &memory.MemoryRecord{Title: "a"}, Score: 0.9},
		{Memory: &memory.MemoryRecord{Title: "b"}, Score: 0.85},
	}

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&mockScorer{candidates: candidates}),
		signal.WithConfirmer(&mockConfirmer{clusters: nil}),
	)

	query := &memory.MemoryRecord{Title: "test"}
	result := consolidator.FindClusterForTest(context.Background(), query, nil)

	g.Expect(result).To(BeNil())
}

func TestFindCluster_MultipleClustersSortedSmallestFirst(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memA := &memory.MemoryRecord{Title: "a"}
	memB := &memory.MemoryRecord{Title: "b"}
	memC := &memory.MemoryRecord{Title: "c"}
	memD := &memory.MemoryRecord{Title: "d"}
	memE := &memory.MemoryRecord{Title: "e"}

	candidates := []signal.ScoredCandidate{
		{Memory: memB, Score: 0.9},
		{Memory: memC, Score: 0.85},
		{Memory: memD, Score: 0.80},
		{Memory: memE, Score: 0.75},
	}

	// Confirmer returns two clusters: one with 4 members, one with 3.
	// Smallest (3) should be returned first.
	confirmedClusters := []signal.ConfirmedCluster{
		{
			Members:   []*memory.MemoryRecord{memA, memB, memC, memD},
			Principle: "Larger cluster",
		},
		{
			Members:   []*memory.MemoryRecord{memA, memD, memE},
			Principle: "Smaller cluster",
		},
	}

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&mockScorer{candidates: candidates}),
		signal.WithConfirmer(&mockConfirmer{clusters: confirmedClusters}),
	)

	result := consolidator.FindClusterForTest(context.Background(), memA, nil)

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Members).To(HaveLen(3))
	g.Expect(result.Principle).To(Equal("Smaller cluster"))
}

func TestFindCluster_NoCandidates_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	consolidator := signal.NewConsolidator(
		signal.WithScorer(&mockScorer{candidates: nil}),
		signal.WithConfirmer(&mockConfirmer{}),
	)

	query := &memory.MemoryRecord{Title: "test"}
	result := consolidator.FindClusterForTest(context.Background(), query, nil)

	g.Expect(result).To(BeNil())
}

func TestFindCluster_NoScorer_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	consolidator := signal.NewConsolidator(
		signal.WithConfirmer(&mockConfirmer{}),
	)

	query := &memory.MemoryRecord{Title: "test"}
	result := consolidator.FindClusterForTest(context.Background(), query, nil)

	g.Expect(result).To(BeNil())
}

// mockArchiver implements signal.Archiver for tests.
type mockArchiver struct {
	archived []string
	err      error
}

func (m *mockArchiver) Archive(sourcePath string) error {
	m.archived = append(m.archived, sourcePath)
	return m.err
}

// mockConfirmer implements signal.Confirmer for tests.
type mockConfirmer struct {
	clusters []signal.ConfirmedCluster
	err      error
}

func (m *mockConfirmer) ConfirmClusters(
	_ context.Context,
	_ *memory.MemoryRecord,
	_ []signal.ScoredCandidate,
) ([]signal.ConfirmedCluster, error) {
	return m.clusters, m.err
}

// mockExtractor implements signal.Extractor for tests.
type mockExtractor struct {
	result *memory.MemoryRecord
	err    error
}

func (m *mockExtractor) ExtractPrinciple(
	_ context.Context,
	_ signal.ConfirmedCluster,
) (*memory.MemoryRecord, error) {
	return m.result, m.err
}

// mockLinkRecomputer implements signal.LinkRecomputer for tests.
type mockLinkRecomputer struct {
	callCount int
}

func (m *mockLinkRecomputer) RecomputeAfterMerge(_, _ string) error {
	m.callCount++

	return nil
}

// mockScorer implements signal.Scorer for tests.
type mockScorer struct {
	candidates []signal.ScoredCandidate
	err        error
}

func (m *mockScorer) FindSimilar(
	_ context.Context,
	_ *memory.MemoryRecord,
	_ []string,
) ([]signal.ScoredCandidate, error) {
	return m.candidates, m.err
}
