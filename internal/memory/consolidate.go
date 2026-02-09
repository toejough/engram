package memory

import (
	"database/sql"
	"fmt"
	"path/filepath"
)

// ConsolidateOpts holds options for memory consolidation.
type ConsolidateOpts struct {
	MemoryRoot           string
	DecayFactor          float64 // Decay factor (default: 0.9)
	PruneThreshold       float64 // Confidence threshold for pruning (default: 0.1)
	DuplicateThreshold   float64 // Similarity threshold for duplicates (default: 0.95)
	MinRetrievals        int     // Minimum retrieval count for promotion (default: 3)
	MinProjects          int     // Minimum unique projects for promotion (default: 2)
}

// ConsolidateResult contains the result of memory consolidation.
type ConsolidateResult struct {
	EntriesDecayed       int
	EntriesPruned        int
	DuplicatesMerged     int
	PromotionCandidates  int
}

// Consolidate performs periodic memory maintenance by running decay, pruning,
// deduplication, and surfacing promotion candidates.
func Consolidate(opts ConsolidateOpts) (*ConsolidateResult, error) {
	if opts.MemoryRoot == "" {
		return nil, fmt.Errorf("memory root is required")
	}

	// Set defaults
	decayFactor := opts.DecayFactor
	if decayFactor == 0 {
		decayFactor = 0.9
	}

	pruneThreshold := opts.PruneThreshold
	if pruneThreshold == 0 {
		pruneThreshold = 0.1
	}

	duplicateThreshold := opts.DuplicateThreshold
	if duplicateThreshold == 0 {
		duplicateThreshold = 0.95
	}

	minRetrievals := opts.MinRetrievals
	if minRetrievals == 0 {
		minRetrievals = 3
	}

	minProjects := opts.MinProjects
	if minProjects == 0 {
		minProjects = 2
	}

	result := &ConsolidateResult{}

	// Step 1: Decay all memories
	decayResult, err := Decay(DecayOpts{
		MemoryRoot: opts.MemoryRoot,
		Factor:     decayFactor,
	})
	if err != nil {
		return nil, fmt.Errorf("decay failed: %w", err)
	}
	result.EntriesDecayed = decayResult.EntriesAffected

	// Step 2: Prune low-confidence entries
	pruneResult, err := Prune(PruneOpts{
		MemoryRoot: opts.MemoryRoot,
		Threshold:  pruneThreshold,
	})
	if err != nil {
		return nil, fmt.Errorf("prune failed: %w", err)
	}
	result.EntriesPruned = pruneResult.EntriesRemoved

	// Step 3: Identify and merge duplicates
	duplicatesMerged, err := mergeDuplicates(opts.MemoryRoot, duplicateThreshold)
	if err != nil {
		return nil, fmt.Errorf("merge duplicates failed: %w", err)
	}
	result.DuplicatesMerged = duplicatesMerged

	// Step 4: Identify promotion candidates
	promoteResult, err := Promote(PromoteOpts{
		MemoryRoot:    opts.MemoryRoot,
		MinRetrievals: minRetrievals,
		MinProjects:   minProjects,
	})
	if err != nil {
		return nil, fmt.Errorf("promote failed: %w", err)
	}
	result.PromotionCandidates = len(promoteResult.Candidates)

	return result, nil
}

// mergeDuplicates identifies and merges duplicate memories based on semantic similarity.
func mergeDuplicates(memoryRoot string, threshold float64) (int, error) {
	// Open DB
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Get all embeddings with their content
	rows, err := db.Query(`
		SELECT e.id, e.content, e.embedding_id
		FROM embeddings e
		WHERE e.embedding_id IS NOT NULL
		ORDER BY e.id
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type entry struct {
		id          int64
		content     string
		embeddingID int64
	}

	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.content, &e.embeddingID); err != nil {
			return 0, fmt.Errorf("failed to scan row: %w", err)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating rows: %w", err)
	}

	// Find duplicates by comparing all pairs
	duplicateCount := 0
	toDelete := make(map[int64]bool)

	for i := 0; i < len(entries); i++ {
		if toDelete[entries[i].id] {
			continue
		}

		for j := i + 1; j < len(entries); j++ {
			if toDelete[entries[j].id] {
				continue
			}

			// Calculate similarity using vector distance
			similarity, err := calculateSimilarity(db, entries[i].embeddingID, entries[j].embeddingID)
			if err != nil {
				continue // Skip on error
			}

			if similarity >= threshold {
				// Mark the second entry for deletion (keep the first)
				toDelete[entries[j].id] = true
				duplicateCount++
			}
		}
	}

	// Delete duplicates
	for id := range toDelete {
		// Get embedding_id before deleting from embeddings table
		var embeddingID int64
		err := db.QueryRow("SELECT embedding_id FROM embeddings WHERE id = ?", id).Scan(&embeddingID)
		if err != nil && err != sql.ErrNoRows {
			continue
		}

		// Delete from embeddings table
		_, _ = db.Exec("DELETE FROM embeddings WHERE id = ?", id)

		// Delete from vec_embeddings table
		if embeddingID > 0 {
			_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", embeddingID)
		}
	}

	return duplicateCount, nil
}

// calculateSimilarity calculates cosine similarity between two embeddings.
func calculateSimilarity(db *sql.DB, embeddingID1, embeddingID2 int64) (float64, error) {
	// Use sqlite-vec's distance function (returns distance, not similarity)
	// Cosine distance = 1 - cosine similarity
	// So similarity = 1 - distance
	var distance float64
	query := `
		SELECT vec_distance_cosine(
			(SELECT embedding FROM vec_embeddings WHERE rowid = ?),
			(SELECT embedding FROM vec_embeddings WHERE rowid = ?)
		)
	`
	err := db.QueryRow(query, embeddingID1, embeddingID2).Scan(&distance)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate distance: %w", err)
	}

	// Convert distance to similarity
	similarity := 1.0 - distance
	return similarity, nil
}
