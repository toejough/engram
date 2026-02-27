package memory

import (
	"strings"
	"time"
)

// RetrievalRelevance represents the relevance score for a single retrieved result.
type RetrievalRelevance struct {
	RetrievalID string    `json:"retrieval_id"`
	RetrievedAt time.Time `json:"retrieved_at"`
	Query       string    `json:"query"`
	ResultID    int64     `json:"result_id"`
	ResultScore float64   `json:"result_score"`
	Relevant    bool      `json:"relevant"`
	Precision   float64   `json:"precision"`
}

// ComputeAverageRetrievalPrecision computes the average precision across all scored retrievals.
func ComputeAverageRetrievalPrecision(scores []RetrievalRelevance) float64 {
	if len(scores) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, score := range scores {
		sum += score.Precision
	}

	return sum / float64(len(scores))
}

// ScoreRetrievalRelevance scores a retrieval by checking if corrections followed.
// If no correction on same topic within timeWindow after retrieval → relevant (precision=1.0).
// If correction followed → not relevant (precision=0.0).
func ScoreRetrievalRelevance(retrieval RetrievalLogEntry, subsequentCorrections []ChangelogEntry, timeWindow time.Duration) []RetrievalRelevance {
	// Parse retrieval timestamp
	retrievedAt, err := time.Parse(time.RFC3339, retrieval.Timestamp)
	if err != nil {
		return nil
	}

	var scores []RetrievalRelevance

	// For each result in the retrieval, check if a correction followed
	for _, result := range retrieval.Results {
		relevant := true
		precision := 1.0

		// Check if any subsequent corrections match this retrieval topic
		for _, correction := range subsequentCorrections {
			// Only consider corrections after retrieval
			if !correction.Timestamp.After(retrievedAt) {
				continue
			}

			// Check if correction is within time window
			if correction.Timestamp.Sub(retrievedAt) > timeWindow {
				continue
			}

			// Simple topic matching: check if correction content relates to retrieval query or result
			if topicsMatch(retrieval.Query, result.Content, correction.ContentSummary) {
				relevant = false
				precision = 0.0

				break
			}
		}

		scores = append(scores, RetrievalRelevance{
			RetrievalID: retrieval.SessionID, // Using session ID as retrieval identifier
			RetrievedAt: retrievedAt,
			Query:       retrieval.Query,
			ResultID:    result.ID,
			ResultScore: result.Score,
			Relevant:    relevant,
			Precision:   precision,
		})
	}

	return scores
}

// extractSignificantWords extracts words longer than 3 characters.
func extractSignificantWords(text string) []string {
	words := strings.Fields(text)

	var significant []string

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:\"'")
		if len(word) > 3 {
			significant = append(significant, word)
		}
	}

	return significant
}

// topicsMatch determines if a correction relates to a retrieval query/result.
// Simple heuristic: check for common significant words.
func topicsMatch(query, resultContent, correctionSummary string) bool {
	// Normalize to lowercase for comparison
	query = strings.ToLower(query)
	resultContent = strings.ToLower(resultContent)
	correctionSummary = strings.ToLower(correctionSummary)

	// Extract significant words from query (length > 3)
	queryWords := extractSignificantWords(query)

	// Check if any significant query words appear in the correction
	for _, word := range queryWords {
		if strings.Contains(correctionSummary, word) {
			return true
		}
	}

	// Also check result content for common words with correction
	resultWords := extractSignificantWords(resultContent)
	for _, word := range resultWords {
		if strings.Contains(correctionSummary, word) {
			return true
		}
	}

	return false
}
