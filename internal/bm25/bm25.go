// Package bm25 implements Best Matching 25 ranking for memory retrieval.
package bm25

import (
	"math"

	"engram/internal/tokenize"
)

// Document represents a searchable document (memory).
type Document struct {
	ID   string // memory filepath
	Text string // searchable content
}

// ScoredDocument is a document ranked by BM25 relevance score.
type ScoredDocument struct {
	ID    string
	Score float64
}

// Scorer computes BM25 relevance scores for documents.
type Scorer struct {
	// Unexported fields for per-query indexing
}

// New creates a new BM25 scorer.
func New() *Scorer {
	return &Scorer{}
}

// Score returns documents ranked by BM25 relevance to the query.
// Query and document text are tokenized by whitespace and punctuation.
// Documents are scored by summing BM25 component for each query term.
// Results are sorted by score descending (highest first).
//
//nolint:cyclop,funlen // BM25 algorithm has necessary complexity
func (s *Scorer) Score(query string, documents []Document) []ScoredDocument {
	if query == "" || len(documents) == 0 {
		return []ScoredDocument{}
	}

	// Tokenize query
	queryTerms := tokenize.Tokenize(query)
	if len(queryTerms) == 0 {
		return []ScoredDocument{}
	}

	// Build inverted index: term -> list of (docIndex, termFrequency)
	index := make(map[string]map[int]int)
	docTokenCounts := make([]int, len(documents))

	for docIdx, doc := range documents {
		docTokens := tokenize.Tokenize(doc.Text)
		docTokenCounts[docIdx] = len(docTokens)

		// Count term frequencies in this document
		termFreq := make(map[string]int)
		for _, token := range docTokens {
			termFreq[token]++
		}

		// Add to inverted index
		for term, freq := range termFreq {
			if index[term] == nil {
				index[term] = make(map[int]int)
			}

			index[term][docIdx] = freq
		}
	}

	// Compute average document length
	totalTokens := 0
	for _, count := range docTokenCounts {
		totalTokens += count
	}

	avgDocLen := float64(totalTokens) / float64(len(documents))

	// Score each document
	scores := make([]ScoredDocument, 0, len(documents))
	for docIdx, doc := range documents {
		score := 0.0

		for _, queryTerm := range queryTerms {
			// Document frequency for this term
			documentFreq := 0
			if termDocs, ok := index[queryTerm]; ok {
				documentFreq = len(termDocs)
			}

			// If term not in any document, skip
			if documentFreq == 0 {
				continue
			}

			// Inverse document frequency (IDF)
			idf := math.Log(
				(float64(len(documents)-documentFreq) + idfSmoothing) / (float64(documentFreq) + idfSmoothing),
			)

			// Term frequency in this document
			termFreq := 0.0

			if docTerms, ok := index[queryTerm]; ok {
				if freq, hasDoc := docTerms[docIdx]; hasDoc {
					termFreq = float64(freq)
				}
			}

			// BM25 component for this term
			docLen := float64(docTokenCounts[docIdx])
			bm25Component := idf * (termFreq * (k1 + 1)) / (termFreq + k1*(1-b+b*(docLen/avgDocLen)))

			score += bm25Component
		}

		if score > 0 {
			scores = append(scores, ScoredDocument{ID: doc.ID, Score: score})
		}
	}

	// Sort by score descending
	sortDescending(scores)

	return scores
}

// unexported constants.
const (
	// b controls length normalization (standard value).
	b = 0.75
	// idfSmoothing smooths IDF to avoid zero values.
	idfSmoothing = 0.5
	// k1 controls term frequency saturation (standard value).
	k1 = 1.5
)

// sortDescending sorts scored documents by score in descending order (highest first).
func sortDescending(scores []ScoredDocument) {
	// Simple insertion sort for small arrays (typical use: <100 documents)
	for i := 1; i < len(scores); i++ {
		key := scores[i]
		j := i - 1

		for j >= 0 && scores[j].Score < key.Score {
			scores[j+1] = scores[j]
			j--
		}

		scores[j+1] = key
	}
}

