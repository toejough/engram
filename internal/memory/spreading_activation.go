package memory

import (
	"database/sql"
	"sort"
)

// unexported constants.
const (
	spreadingActivationBoostFactor = 0.8
	spreadingActivationThreshold   = 0.7
)

// applySpreadingActivation performs a secondary vector search for memories
// that are semantically similar to the top-K primary results (FR-017).
//
// For each primary result, its stored embedding is used to search for additional
// memories with raw cosine similarity > threshold. Matching memories receive a
// temporary score boost:
//
//	boostedScore = primary.Score × similarity × spreadingActivationBoostFactor
//
// The boost is interaction-scoped — nothing is written to the database.
// Primary results are never duplicated in the output.
// The returned slice is sorted by score descending.
func applySpreadingActivation(db *sql.DB, topK []QueryResult, threshold float64) ([]QueryResult, error) {
	if len(topK) == 0 {
		return topK, nil
	}

	// Index primary IDs to prevent duplicates.
	inTopK := make(map[int64]bool, len(topK))
	for _, r := range topK {
		inTopK[r.ID] = true
	}

	secondary := make(map[int64]QueryResult)

	for _, primary := range topK {
		// Retrieve the stored embedding for this primary result.
		var embeddingID int64
		if err := db.QueryRow("SELECT embedding_id FROM embeddings WHERE id = ?", primary.ID).Scan(&embeddingID); err != nil {
			continue // Embedding not found — skip gracefully.
		}

		var embeddingBlob []byte
		if err := db.QueryRow("SELECT embedding FROM vec_embeddings WHERE rowid = ?", embeddingID).Scan(&embeddingBlob); err != nil {
			continue
		}

		// Secondary search: find memories similar to this primary result.
		// Uses raw cosine similarity (not confidence-weighted) to identify
		// semantic neighbours regardless of their retrieval history.
		rows, err := db.Query(`
			SELECT e.id, e.content, e.source, e.source_type, e.confidence, e.memory_type,
			       e.retrieval_count, e.projects_retrieved,
			       (1 - vec_distance_cosine(v.embedding, ?)) as similarity
			FROM vec_embeddings v
			JOIN embeddings e ON e.embedding_id = v.rowid
			WHERE e.id != ?
			ORDER BY similarity DESC
			LIMIT 20
		`, embeddingBlob, primary.ID)
		if err != nil {
			continue
		}

		for rows.Next() {
			var (
				r           QueryResult
				projectsRaw string
				similarity  float64
			)

			err := rows.Scan(&r.ID, &r.Content, &r.Source, &r.SourceType, &r.Confidence,
				&r.MemoryType, &r.RetrievalCount, &projectsRaw, &similarity)
			if err != nil {
				continue
			}

			if similarity < threshold {
				break // Results are DESC-ordered; remaining will also be below threshold.
			}

			if inTopK[r.ID] {
				continue // Already a primary result.
			}

			r.ProjectsRetrieved = parseProjectsList(projectsRaw)
			r.MatchType = "vector"

			boostedScore := primary.Score * similarity * spreadingActivationBoostFactor
			if boostedScore > 1.0 {
				boostedScore = 1.0
			}

			r.Score = boostedScore

			if existing, ok := secondary[r.ID]; !ok || boostedScore > existing.Score {
				secondary[r.ID] = r
			}
		}

		_ = rows.Close()
	}

	// Merge primary and secondary results.
	result := make([]QueryResult, len(topK), len(topK)+len(secondary))
	copy(result, topK)

	for _, r := range secondary {
		result = append(result, r)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	return result, nil
}
