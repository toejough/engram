package signal_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

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
