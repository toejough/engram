// Package tfidf computes TF-IDF cosine similarity for duplicate detection.
package tfidf

import (
	"math"
	"strings"
	"unicode"
)

// Scorer computes TF-IDF cosine similarity between texts.
type Scorer struct{}

// NewScorer creates a new TF-IDF scorer.
func NewScorer() *Scorer {
	return &Scorer{}
}

// ClusterConfidence returns the average pairwise cosine similarity
// of TF-IDF vectors for the given texts. Returns a score in [0,1]
// where 1 means identical content.
func (s *Scorer) ClusterConfidence(texts []string) float64 {
	if len(texts) < minPairSize {
		return 0
	}

	// Build corpus-wide document frequency.
	docFreq := make(map[string]int)
	tokenized := make([]map[string]int, len(texts))

	for idx, text := range texts {
		termFreq := termFrequencies(text)
		tokenized[idx] = termFreq

		for term := range termFreq {
			docFreq[term]++
		}
	}

	// Compute pairwise cosine similarity.
	corpusSize := len(texts)
	pairCount := 0
	totalSimilarity := 0.0

	for i := range corpusSize {
		for j := i + 1; j < corpusSize; j++ {
			sim := cosineSimilarity(tokenized[i], tokenized[j], docFreq, corpusSize)
			totalSimilarity += sim
			pairCount++
		}
	}

	if pairCount == 0 {
		return 0
	}

	return totalSimilarity / float64(pairCount)
}

// unexported constants.
const (
	minPairSize = 2
)

func cosineSimilarity(
	first, second map[string]int,
	docFreq map[string]int,
	corpusSize int,
) float64 {
	// Collect all terms from both documents.
	allTerms := make(map[string]struct{}, len(first)+len(second))
	for term := range first {
		allTerms[term] = struct{}{}
	}

	for term := range second {
		allTerms[term] = struct{}{}
	}

	// Compute TF-IDF vectors and dot product simultaneously.
	var dotProduct, normFirst, normSecond float64

	for term := range allTerms {
		idf := math.Log(1 + float64(corpusSize)/float64(docFreq[term]))

		tfidfFirst := float64(first[term]) * idf
		tfidfSecond := float64(second[term]) * idf

		dotProduct += tfidfFirst * tfidfSecond
		normFirst += tfidfFirst * tfidfFirst
		normSecond += tfidfSecond * tfidfSecond
	}

	denominator := math.Sqrt(normFirst) * math.Sqrt(normSecond)
	if denominator == 0 {
		return 0
	}

	return dotProduct / denominator
}

func termFrequencies(text string) map[string]int {
	freqs := make(map[string]int)

	var current strings.Builder

	for _, ch := range strings.ToLower(text) {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			current.WriteRune(ch)
		} else if current.Len() > 0 {
			freqs[current.String()]++
			current.Reset()
		}
	}

	if current.Len() > 0 {
		freqs[current.String()]++
	}

	return freqs
}
