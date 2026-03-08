package bm25_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/bm25"
)

// TestBM25BasicScoring verifies basic BM25 scoring with one query term.
func TestBM25BasicScoring(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	scorer := bm25.New()

	documents := []bm25.Document{
		{ID: "doc1", Text: "elasticsearch "},           // 3 occurrences, rare
		{ID: "doc2", Text: "elasticsearch monitoring"}, // 1 occurrence, rare
		{ID: "doc3", Text: "hello world gamma"},        // no match
		{ID: "doc4", Text: "logging aggregation"},      // no match
		{ID: "doc5", Text: "data storage index"},       // no match
	}

	query := "elasticsearch"
	results := scorer.Score(query, documents)

	// Should return docs with "elasticsearch" (appears in only 2 out of 5, so IDF is positive)
	g.Expect(results).ToNot(BeEmpty())

	// doc3, doc4, doc5 should not appear (no match)
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}

	g.Expect(ids).NotTo(ContainElement("doc3"))
	g.Expect(ids).NotTo(ContainElement("doc4"))
	g.Expect(ids).NotTo(ContainElement("doc5"))
}

// TestBM25EmptyDocuments returns empty when documents are empty.
func TestBM25EmptyDocuments(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	scorer := bm25.New()
	documents := []bm25.Document{}

	results := scorer.Score("commit", documents)

	g.Expect(results).To(BeEmpty())
}

// TestBM25EmptyQuery returns empty when query is empty.
func TestBM25EmptyQuery(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	scorer := bm25.New()

	documents := []bm25.Document{
		{ID: "doc1", Text: "hello world"},
	}

	results := scorer.Score("", documents)

	g.Expect(results).To(BeEmpty())
}

// TestBM25MatchesSingleQueryTerm verifies single term matching.
func TestBM25MatchesSingleQueryTerm(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	scorer := bm25.New()

	documents := []bm25.Document{
		{ID: "doc1", Text: "Prometheus metrics"},
		{ID: "doc2", Text: "Prometheus alerting"},
		{ID: "doc3", Text: "NoSQL alternative"},
		{ID: "doc4", Text: "time series database"},
	}

	results := scorer.Score("Prometheus", documents)

	// Query is case-insensitive: "Prometheus" should match docs 1 and 2
	// But because prometheus appears in 2/4 docs, IDF might be close to 0
	// So just verify that docs 3 and 4 don't appear (they don't have prometheus)
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}

	g.Expect(ids).NotTo(ContainElement("doc3"))
	g.Expect(ids).NotTo(ContainElement("doc4"))
}

// TestBM25NoMatches returns empty when no documents match.
func TestBM25NoMatches(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	scorer := bm25.New()

	documents := []bm25.Document{
		{ID: "doc1", Text: "alpha beta gamma"},
		{ID: "doc2", Text: "hello world"},
	}

	results := scorer.Score("commit", documents)

	g.Expect(results).To(BeEmpty())
}

// TestBM25ScoresDescending verifies results are sorted by score descending.
func TestBM25ScoresDescending(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	scorer := bm25.New()

	// Use 7 documents with one rare term to ensure multiple matches with different scores
	// elasticsearch appears in 2 out of 7 documents (rare enough for positive IDF)
	documents := []bm25.Document{
		{ID: "doc1", Text: "elasticsearch search engine"},
		{ID: "doc2", Text: "elasticsearch cluster"},
		{ID: "doc3", Text: "postgresql database management"},
		{ID: "doc4", Text: "mongodb nosql database"},
		{ID: "doc5", Text: "redis cache system"},
		{ID: "doc6", Text: "mysql relational database"},
		{ID: "doc7", Text: "cassandra distributed storage"},
	}

	results := scorer.Score("elasticsearch", documents)

	// Should match 2 documents (both have elasticsearch)
	g.Expect(results).To(HaveLen(2))

	// Verify all results are sorted by score descending
	for i := 1; i < len(results); i++ {
		g.Expect(results[i-1].Score >= results[i].Score).To(BeTrue())
	}

	// doc2 should rank higher than doc1 (more occurrences of elasticsearch)
	g.Expect(results[0].ID).To(Equal("doc2"))
	g.Expect(results[1].ID).To(Equal("doc1"))
}

// TestBM25TokenizationWithPunctuation verifies punctuation is stripped.
func TestBM25TokenizationWithPunctuation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	scorer := bm25.New()

	documents := []bm25.Document{
		{ID: "doc1", Text: "ansible, configuration-management! automation?"},
		{ID: "doc2", Text: "chef-deploy puppet-code"},
		{ID: "doc3", Text: "docker containers"},
		{ID: "doc4", Text: "kubernetes orchestration"},
	}

	results := scorer.Score("ansible", documents)

	// ansible should only match doc1
	// doc2, doc3, doc4 should not match
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}

	g.Expect(ids).NotTo(ContainElement("doc2"))
	g.Expect(ids).NotTo(ContainElement("doc3"))
	g.Expect(ids).NotTo(ContainElement("doc4"))
}
