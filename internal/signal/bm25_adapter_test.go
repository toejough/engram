package signal_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

func TestBM25Adapter_FindSimilar_CapsAtMax(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create many similar records.
	titles := []string{
		"error handling alpha", "error handling beta", "error handling gamma",
		"error handling delta", "error handling epsilon", "error handling zeta",
		"error handling eta", "error handling theta", "error handling iota",
		"error handling kappa", "error handling lambda", "error handling mu",
		"error handling nu", "error handling xi", "error handling omicron",
		"error handling pi", "error handling rho", "error handling sigma",
		"error handling tau", "error handling upsilon",
	}

	corpus := make([]*memory.MemoryRecord, 0, len(titles))

	for _, title := range titles {
		corpus = append(corpus, makeRecord(title, "wrap errors with context",
			[]string{"error", "handling", "wrap"}))
	}

	lister := &mockRecordLister{records: corpus}
	query := makeRecord("error handling", "wrap errors", []string{"error", "handling", "wrap"})

	const maxResults = 3

	adapter := signal.NewBM25ScorerAdapter(
		lister,
		signal.WithMaxCandidates(maxResults),
		signal.WithScoreThreshold(0), // Allow all scores.
	)

	candidates, err := adapter.FindSimilar(context.Background(), query, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(len(candidates)).To(BeNumerically("<=", maxResults))
}

func TestBM25Adapter_FindSimilar_EmptyCorpus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := &mockRecordLister{records: nil}
	query := makeRecord("error handling", "wrap errors", []string{"error"})

	adapter := signal.NewBM25ScorerAdapter(lister)
	candidates, err := adapter.FindSimilar(context.Background(), query, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(candidates).To(BeEmpty())
}

func TestBM25Adapter_FindSimilar_ExcludesAbsorbed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	corpus := []*memory.MemoryRecord{
		makeRecord("error handling patterns", "wrap errors", []string{"error", "handling"}),
		makeAbsorbedRecord("error retry strategy", "use backoff", []string{"error", "retry"}),
		makeRecord("error recovery", "graceful degradation", []string{"error", "recovery"}),
	}

	lister := &mockRecordLister{records: corpus}
	query := makeRecord("error management", "handle errors well", []string{"error", "management"})

	adapter := signal.NewBM25ScorerAdapter(lister)
	candidates, err := adapter.FindSimilar(context.Background(), query, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	for _, candidate := range candidates {
		g.Expect(candidate.Memory.Title).NotTo(Equal("error retry strategy"),
			"absorbed memories should not appear in results")
	}
}

func TestBM25Adapter_FindSimilar_ExcludesSlugs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	corpus := []*memory.MemoryRecord{
		makeRecord("error handling patterns", "wrap errors", []string{"error", "handling"}),
		makeRecord("error retry strategy", "use backoff", []string{"error", "retry"}),
		makeRecord("error recovery", "graceful degradation", []string{"error", "recovery"}),
	}

	lister := &mockRecordLister{records: corpus}
	query := makeRecord("error management", "handle errors well", []string{"error", "management"})

	adapter := signal.NewBM25ScorerAdapter(lister)
	candidates, err := adapter.FindSimilar(
		context.Background(), query, []string{"error handling patterns"},
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	for _, candidate := range candidates {
		g.Expect(candidate.Memory.Title).NotTo(Equal("error handling patterns"),
			"excluded slug should not appear in results")
	}
}

func TestBM25Adapter_FindSimilar_ListerError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	listErr := errors.New("disk read failed")
	lister := &mockRecordLister{err: listErr}
	query := makeRecord("error handling", "wrap errors", []string{"error"})

	adapter := signal.NewBM25ScorerAdapter(lister)
	_, err := adapter.FindSimilar(context.Background(), query, nil)
	g.Expect(err).To(MatchError(ContainSubstring("disk read failed")))
}

func TestBM25Adapter_FindSimilar_RespectsThreshold(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	corpus := []*memory.MemoryRecord{
		makeRecord("error handling patterns", "wrap errors", []string{"error", "handling"}),
		makeRecord("database connection pooling", "reuse connections", []string{"database", "connection", "pool"}),
	}

	lister := &mockRecordLister{records: corpus}
	query := makeRecord("error management", "handle errors", []string{"error", "management"})

	// Use a high threshold so only strong matches pass.
	const highThreshold = 5.0

	adapter := signal.NewBM25ScorerAdapter(
		lister,
		signal.WithScoreThreshold(highThreshold),
	)

	candidates, err := adapter.FindSimilar(context.Background(), query, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	for _, candidate := range candidates {
		g.Expect(candidate.Score).To(
			BeNumerically(">=", highThreshold),
			"all candidates should be above the threshold",
		)
	}
}

func TestBM25Adapter_FindSimilar_ReturnsTopCandidates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	corpus := []*memory.MemoryRecord{
		makeRecord("error handling patterns", "always wrap errors with context", []string{"error", "handling", "wrap"}),
		makeRecord("logging best practices", "use structured logging", []string{"logging", "structured", "debug"}),
		makeRecord("error retry strategy", "use exponential backoff for retries", []string{"error", "retry", "backoff"}),
		makeRecord("database connection pooling", "reuse database connections", []string{"database", "connection", "pool"}),
		makeRecord("error recovery mechanisms", "graceful degradation on failure", []string{"error", "recovery", "graceful"}),
	}

	lister := &mockRecordLister{records: corpus}
	query := makeRecord(
		"error handling in distributed systems",
		"handle errors at boundaries",
		[]string{"error", "handling", "distributed"},
	)

	adapter := signal.NewBM25ScorerAdapter(lister)
	candidates, err := adapter.FindSimilar(context.Background(), query, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(candidates).NotTo(BeEmpty())

	// Results should be sorted by score descending.
	for idx := 1; idx < len(candidates); idx++ {
		g.Expect(candidates[idx-1].Score).To(
			BeNumerically(">=", candidates[idx].Score),
			"candidates should be sorted by score descending",
		)
	}

	// The error-related memories should score higher than unrelated ones.
	g.Expect(candidates[0].Memory.Title).To(ContainSubstring("error"))
}

// mockRecordLister implements signal.MemoryRecordLister for tests.
type mockRecordLister struct {
	records []*memory.MemoryRecord
	err     error
}

func (m *mockRecordLister) ListAllRecords(_ context.Context) ([]*memory.MemoryRecord, error) {
	return m.records, m.err
}

func makeAbsorbedRecord(title, principle string, keywords []string) *memory.MemoryRecord {
	record := makeRecord(title, principle, keywords)
	record.Absorbed = []memory.AbsorbedRecord{
		{From: "some-other-memory", ContentHash: "abc123"},
	}

	return record
}

func makeRecord(title, principle string, keywords []string) *memory.MemoryRecord {
	return &memory.MemoryRecord{
		Title:     title,
		Principle: principle,
		Keywords:  keywords,
	}
}
