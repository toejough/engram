package signal

import (
	"context"
	"fmt"
	"strings"

	"engram/internal/bm25"
	"engram/internal/memory"
)

// BM25Option configures a BM25ScorerAdapter.
type BM25Option func(*BM25ScorerAdapter)

// BM25ScorerAdapter wraps bm25.Scorer to satisfy the Scorer interface.
type BM25ScorerAdapter struct {
	lister        MemoryRecordLister
	threshold     float64
	maxCandidates int
}

// NewBM25ScorerAdapter creates a BM25ScorerAdapter with the given options.
func NewBM25ScorerAdapter(lister MemoryRecordLister, opts ...BM25Option) *BM25ScorerAdapter {
	adapter := &BM25ScorerAdapter{
		lister:        lister,
		threshold:     defaultScoreThreshold,
		maxCandidates: defaultMaxCandidates,
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

// FindSimilar scores all non-excluded, non-absorbed memories against the query
// using BM25, returning candidates above threshold sorted by score descending.
func (a *BM25ScorerAdapter) FindSimilar(
	ctx context.Context,
	query *memory.MemoryRecord,
	exclude []string,
) ([]ScoredCandidate, error) {
	records, err := a.lister.ListAllRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding similar: listing records: %w", err)
	}

	excludeSet := buildExcludeSet(exclude)

	// Build corpus of eligible documents, tracking title→record mapping.
	documents := make([]bm25.Document, 0, len(records))
	recordsByTitle := make(map[string]*memory.MemoryRecord, len(records))

	for _, record := range records {
		if shouldExclude(record, excludeSet) {
			continue
		}

		docText := buildDocumentText(record)
		documents = append(documents, bm25.Document{
			ID:   record.Title,
			Text: docText,
		})

		recordsByTitle[record.Title] = record
	}

	if len(documents) == 0 {
		return []ScoredCandidate{}, nil
	}

	queryText := buildDocumentText(query)
	scorer := bm25.New()
	scored := scorer.Score(queryText, documents)

	candidates := make([]ScoredCandidate, 0, min(len(scored), a.maxCandidates))

	for _, doc := range scored {
		if doc.Score < a.threshold {
			break // Results are sorted descending; remaining scores are lower.
		}

		if len(candidates) >= a.maxCandidates {
			break
		}

		record, ok := recordsByTitle[doc.ID]
		if !ok {
			continue
		}

		candidates = append(candidates, ScoredCandidate{
			Memory: record,
			Score:  doc.Score,
		})
	}

	return candidates, nil
}

// MemoryRecordLister loads all MemoryRecords from the data directory.
type MemoryRecordLister interface {
	ListAllRecords(ctx context.Context) ([]*memory.MemoryRecord, error)
}

// WithMaxCandidates sets the maximum number of candidates to return.
func WithMaxCandidates(limit int) BM25Option {
	return func(adapter *BM25ScorerAdapter) {
		adapter.maxCandidates = limit
	}
}

// WithScoreThreshold sets the minimum BM25 score for a candidate to be included.
func WithScoreThreshold(threshold float64) BM25Option {
	return func(adapter *BM25ScorerAdapter) {
		adapter.threshold = threshold
	}
}

// unexported constants.
const (
	defaultMaxCandidates  = 10
	defaultScoreThreshold = 0.3
)

// buildDocumentText constructs BM25 query/document text from title, principle, and keywords.
func buildDocumentText(record *memory.MemoryRecord) string {
	parts := make([]string, 0, 3) //nolint:mnd // title + principle + keywords

	if record.Title != "" {
		parts = append(parts, record.Title)
	}

	if record.Principle != "" {
		parts = append(parts, record.Principle)
	}

	if len(record.Keywords) > 0 {
		parts = append(parts, strings.Join(record.Keywords, " "))
	}

	return strings.Join(parts, " ")
}

// buildExcludeSet creates a set of slugs (titles) to exclude from results.
func buildExcludeSet(exclude []string) map[string]struct{} {
	set := make(map[string]struct{}, len(exclude))

	for _, slug := range exclude {
		set[slug] = struct{}{}
	}

	return set
}

// shouldExclude returns true if a record should be excluded from scoring.
func shouldExclude(record *memory.MemoryRecord, excludeSet map[string]struct{}) bool {
	if _, excluded := excludeSet[record.Title]; excluded {
		return true
	}

	return len(record.Absorbed) > 0
}
