package memory

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OptimizeOpts holds options for the unified optimization pipeline.
type OptimizeOpts struct {
	MemoryRoot     string
	ClaudeMDPath   string
	DecayBase      float64 // Non-promoted decay base (default 0.9)
	PruneThreshold float64 // default 0.1
	DupThreshold   float64 // default 0.95
	SynthThreshold float64 // default 0.8
	MinClusterSize int     // default 3
	MinRetrievals  int     // default 5
	MinProjects    int     // default 3
	AutoApprove    bool    // --yes flag
	ReviewFunc     func(action string, description string) (bool, error)
}

// OptimizeResult contains the results of the optimization pipeline.
type OptimizeResult struct {
	DecayApplied          bool
	DecayFactor           float64
	PromotedDecayFactor   float64
	DaysSinceLastOptimize float64
	EntriesDecayed        int
	ContradictionsFound   int
	AutoDemoted           int
	EntriesPruned         int
	DuplicatesMerged      int
	PatternsFound         int
	PatternsApproved      int
	PromotionCandidates   int
	PromotionsApproved    int
	ClaudeMDDeduped       int
}

// Optimize runs the unified memory optimization pipeline.
// Steps: time-decay → contradiction detection → auto-demote → prune → dedup → synthesize → promote → dedup CLAUDE.md.
func Optimize(opts OptimizeOpts) (*OptimizeResult, error) {
	if opts.MemoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.MemoryRoot = filepath.Join(home, ".claude", "memory")
	}

	if opts.ClaudeMDPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.ClaudeMDPath = filepath.Join(home, ".claude", "CLAUDE.md")
	}

	// Set defaults
	if opts.DecayBase == 0 {
		opts.DecayBase = 0.9
	}
	if opts.PruneThreshold == 0 {
		opts.PruneThreshold = 0.1
	}
	if opts.DupThreshold == 0 {
		opts.DupThreshold = 0.95
	}
	if opts.SynthThreshold == 0 {
		opts.SynthThreshold = 0.8
	}
	if opts.MinClusterSize == 0 {
		opts.MinClusterSize = 3
	}
	if opts.MinRetrievals == 0 {
		opts.MinRetrievals = 5
	}
	if opts.MinProjects == 0 {
		opts.MinProjects = 3
	}

	// Ensure memory dir exists
	if err := os.MkdirAll(opts.MemoryRoot, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}

	// Open DB
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	result := &OptimizeResult{}

	// Step 1: Time-based decay
	if err := optimizeDecay(db, opts, result); err != nil {
		return nil, fmt.Errorf("decay failed: %w", err)
	}

	// Step 2: Contradiction detection
	if err := optimizeContradictions(db, opts, result); err != nil {
		return nil, fmt.Errorf("contradiction detection failed: %w", err)
	}

	// Step 3: Auto-demote
	if err := optimizeAutoDemote(db, opts, result); err != nil {
		return nil, fmt.Errorf("auto-demote failed: %w", err)
	}

	// Step 4: Prune DB
	if err := optimizePrune(db, opts, result); err != nil {
		return nil, fmt.Errorf("prune failed: %w", err)
	}

	// Step 5: Deduplicate DB
	if err := optimizeDedup(db, opts, result); err != nil {
		return nil, fmt.Errorf("dedup failed: %w", err)
	}

	// Step 6: Synthesize patterns (interactive)
	if err := optimizeSynthesize(db, opts, result); err != nil {
		return nil, fmt.Errorf("synthesis failed: %w", err)
	}

	// Step 7: Promote (interactive)
	if err := optimizePromote(db, opts, result); err != nil {
		return nil, fmt.Errorf("promote failed: %w", err)
	}

	// Step 8: Deduplicate CLAUDE.md
	if err := optimizeClaudeMDDedup(db, opts, result); err != nil {
		return nil, fmt.Errorf("CLAUDE.md dedup failed: %w", err)
	}

	return result, nil
}

// optimizeDecay applies time-based decay to all entries.
func optimizeDecay(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Read last_optimized_at from metadata
	lastOptStr, err := getMetadata(db, "last_optimized_at")
	if err != nil {
		return err
	}

	now := time.Now()
	var daysSince float64

	if lastOptStr != "" {
		lastOpt, err := time.Parse(time.RFC3339, lastOptStr)
		if err == nil {
			hoursSince := now.Sub(lastOpt).Hours()
			daysSince = hoursSince / 24.0

			// If less than 1 hour elapsed, skip decay
			if hoursSince < 1.0 {
				result.DecayApplied = false
				result.DaysSinceLastOptimize = daysSince
				// Still update timestamp
				return setMetadata(db, "last_optimized_at", now.Format(time.RFC3339))
			}
		}
	} else {
		// First run — apply decay for 1 day as bootstrap
		daysSince = 1.0
	}

	result.DaysSinceLastOptimize = daysSince

	// Non-promoted: factor = DecayBase ^ days (default 0.9^days, half-life ~6.6 days)
	factor := math.Pow(opts.DecayBase, daysSince)
	// Promoted: factor = 0.995 ^ days (half-life ~139 days)
	promotedFactor := math.Pow(0.995, daysSince)

	result.DecayFactor = factor
	result.PromotedDecayFactor = promotedFactor
	result.DecayApplied = true

	// Apply decay to non-promoted entries
	res1, err := db.Exec("UPDATE embeddings SET confidence = confidence * ? WHERE promoted = 0", factor)
	if err != nil {
		return fmt.Errorf("failed to decay non-promoted: %w", err)
	}
	affected1, _ := res1.RowsAffected()

	// Apply decay to promoted entries
	res2, err := db.Exec("UPDATE embeddings SET confidence = confidence * ? WHERE promoted = 1", promotedFactor)
	if err != nil {
		return fmt.Errorf("failed to decay promoted: %w", err)
	}
	affected2, _ := res2.RowsAffected()

	result.EntriesDecayed = int(affected1 + affected2)

	return setMetadata(db, "last_optimized_at", now.Format(time.RFC3339))
}

// optimizeContradictions finds recent memories that contradict promoted entries.
func optimizeContradictions(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Get all promoted entries
	rows, err := db.Query("SELECT id, content, embedding_id FROM embeddings WHERE promoted = 1 AND embedding_id IS NOT NULL")
	if err != nil {
		return err
	}

	type promotedEntry struct {
		id          int64
		content     string
		embeddingID int64
	}
	var promoted []promotedEntry
	for rows.Next() {
		var e promotedEntry
		if err := rows.Scan(&e.id, &e.content, &e.embeddingID); err != nil {
			_ = rows.Close()
			return err
		}
		promoted = append(promoted, e)
	}
	_ = rows.Close()

	if len(promoted) == 0 {
		return nil
	}

	// Collect IDs to update (avoid write-during-read deadlock)
	var contradictionIDs []int64

	for _, p := range promoted {
		// Find correction memories with high similarity using SQL
		query := `
			SELECT e.id, e.content, e.memory_type,
			       (1 - vec_distance_cosine(v.embedding,
			           (SELECT embedding FROM vec_embeddings WHERE rowid = ?)
			       )) as similarity
			FROM vec_embeddings v
			JOIN embeddings e ON e.embedding_id = v.rowid
			WHERE e.id != ?
			  AND e.memory_type = 'correction'
			ORDER BY similarity DESC
			LIMIT 10
		`
		cRows, err := db.Query(query, p.embeddingID, p.id)
		if err != nil {
			continue
		}

		for cRows.Next() {
			var cID int64
			var cContent, cMemType string
			var similarity float64
			if err := cRows.Scan(&cID, &cContent, &cMemType, &similarity); err != nil {
				continue
			}

			if similarity > 0.8 {
				// Check if this is actually a contradiction (not just similar topic)
				conflictType := detectConflictType(extractMessageContent(cContent), extractMessageContent(p.content))
				if conflictType == "contradiction" {
					// Collect ID for update after closing iterator
					contradictionIDs = append(contradictionIDs, p.id)
					result.ContradictionsFound++
				}
			}
		}
		_ = cRows.Close()
	}

	// Apply confidence penalties after all reads complete
	for _, id := range contradictionIDs {
		_, _ = db.Exec("UPDATE embeddings SET confidence = confidence * 0.5 WHERE id = ?", id)
	}

	return nil
}

// optimizeAutoDemote removes promoted entries with confidence below threshold from CLAUDE.md.
func optimizeAutoDemote(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	rows, err := db.Query("SELECT id, content FROM embeddings WHERE promoted = 1 AND confidence < 0.3")
	if err != nil {
		return err
	}

	type demoteEntry struct {
		id      int64
		content string
	}
	var toDemote []demoteEntry
	for rows.Next() {
		var e demoteEntry
		if err := rows.Scan(&e.id, &e.content); err != nil {
			_ = rows.Close()
			return err
		}
		toDemote = append(toDemote, e)
	}
	_ = rows.Close()

	if len(toDemote) == 0 {
		return nil
	}

	// Collect content strings for removal from CLAUDE.md
	var toRemove []string
	for _, e := range toDemote {
		// Extract the message content to match against CLAUDE.md lines
		msg := extractMessageContent(e.content)
		if msg != "" {
			toRemove = append(toRemove, msg)
		}
		// Also try matching full content
		toRemove = append(toRemove, e.content)
	}

	// Remove from CLAUDE.md
	if err := RemoveFromClaudeMD(opts.ClaudeMDPath, toRemove); err != nil {
		return fmt.Errorf("failed to remove demoted entries from CLAUDE.md: %w", err)
	}

	// Clear promoted flag in DB
	for _, e := range toDemote {
		_, _ = db.Exec("UPDATE embeddings SET promoted = 0, promoted_at = '' WHERE id = ?", e.id)
	}

	result.AutoDemoted = len(toDemote)
	return nil
}

// optimizePrune removes non-promoted entries below confidence threshold.
func optimizePrune(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Get entries to delete (need embedding_id for cleanup)
	rows, err := db.Query("SELECT id, embedding_id FROM embeddings WHERE confidence < ? AND promoted = 0", opts.PruneThreshold)
	if err != nil {
		return err
	}

	type pruneEntry struct {
		id          int64
		embeddingID int64
	}
	var toDelete []pruneEntry
	for rows.Next() {
		var e pruneEntry
		if err := rows.Scan(&e.id, &e.embeddingID); err != nil {
			_ = rows.Close()
			return err
		}
		toDelete = append(toDelete, e)
	}
	_ = rows.Close()

	// Delete entries and clean up vec/FTS5
	for _, e := range toDelete {
		_, _ = db.Exec("DELETE FROM embeddings WHERE id = ?", e.id)
		if e.embeddingID > 0 {
			_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", e.embeddingID)
		}
		deleteFTS5(db, e.id)
	}

	result.EntriesPruned = len(toDelete)
	return nil
}

// optimizeDedup merges entries with similarity > threshold.
func optimizeDedup(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Get all entries with embeddings
	rows, err := db.Query(`
		SELECT e.id, e.content, e.embedding_id, e.confidence, e.retrieval_count
		FROM embeddings e
		WHERE e.embedding_id IS NOT NULL
		ORDER BY e.id
	`)
	if err != nil {
		return err
	}

	type entry struct {
		id             int64
		content        string
		embeddingID    int64
		confidence     float64
		retrievalCount int
	}

	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.content, &e.embeddingID, &e.confidence, &e.retrievalCount); err != nil {
			_ = rows.Close()
			return err
		}
		entries = append(entries, e)
	}
	_ = rows.Close()

	toDelete := make(map[int64]bool)

	for i := 0; i < len(entries); i++ {
		if toDelete[entries[i].id] {
			continue
		}
		for j := i + 1; j < len(entries); j++ {
			if toDelete[entries[j].id] {
				continue
			}

			sim, err := calculateSimilarity(db, entries[i].embeddingID, entries[j].embeddingID)
			if err != nil {
				continue
			}

			if sim >= opts.DupThreshold {
				// Keep the one with higher confidence/retrieval count
				if entries[j].confidence > entries[i].confidence ||
					(entries[j].confidence == entries[i].confidence && entries[j].retrievalCount > entries[i].retrievalCount) {
					toDelete[entries[i].id] = true
					break // i is deleted, move to next i
				}
				toDelete[entries[j].id] = true
			}
		}
	}

	// Delete duplicates
	for id := range toDelete {
		var embeddingID int64
		_ = db.QueryRow("SELECT embedding_id FROM embeddings WHERE id = ?", id).Scan(&embeddingID)
		_, _ = db.Exec("DELETE FROM embeddings WHERE id = ?", id)
		if embeddingID > 0 {
			_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", embeddingID)
		}
		deleteFTS5(db, id)
	}

	result.DuplicatesMerged = len(toDelete)
	return nil
}

// optimizeSynthesize clusters similar entries and generates patterns.
func optimizeSynthesize(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Get all entries with embeddings
	rows, err := db.Query(`
		SELECT e.id, e.content, e.embedding_id
		FROM embeddings e
		WHERE e.embedding_id IS NOT NULL
		ORDER BY e.id
	`)
	if err != nil {
		return err
	}

	var entries []clusterEntry
	for rows.Next() {
		var e clusterEntry
		if err := rows.Scan(&e.id, &e.content, &e.embeddingID); err != nil {
			_ = rows.Close()
			return err
		}
		entries = append(entries, e)
	}
	_ = rows.Close()

	if len(entries) < opts.MinClusterSize {
		return nil
	}

	// Cluster by similarity
	clusters := clusterBySimilarity(db, entries, opts.SynthThreshold)

	// Filter clusters by min size
	for _, cluster := range clusters {
		if len(cluster) < opts.MinClusterSize {
			continue
		}

		pattern := generatePattern(cluster)
		result.PatternsFound++

		// Ask for approval
		approved := opts.AutoApprove
		if !approved && opts.ReviewFunc != nil {
			var err error
			approved, err = opts.ReviewFunc("synthesize", fmt.Sprintf("Pattern: %s\n%s", pattern.Theme, pattern.Synthesis))
			if err != nil {
				return err
			}
		}

		if approved {
			result.PatternsApproved++
			// Insert synthesized pattern as new memory
			_, _ = db.Exec(
				"INSERT INTO embeddings (content, source, source_type, confidence, memory_type) VALUES (?, 'synthesized', 'internal', 1.0, 'reflection')",
				pattern.Synthesis,
			)
			// Delete cluster members
			for _, e := range cluster {
				var embeddingID int64
				_ = db.QueryRow("SELECT embedding_id FROM embeddings WHERE id = ?", e.id).Scan(&embeddingID)
				_, _ = db.Exec("DELETE FROM embeddings WHERE id = ?", e.id)
				if embeddingID > 0 {
					_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", embeddingID)
				}
				deleteFTS5(db, e.id)
			}
		}
	}

	return nil
}

// optimizePromote finds high-value memories for promotion to CLAUDE.md.
func optimizePromote(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Find candidates: high retrieval count, multiple projects, not yet promoted
	query := `
		SELECT id, content, retrieval_count, projects_retrieved
		FROM embeddings
		WHERE retrieval_count >= ?
		  AND promoted = 0
	`
	rows, err := db.Query(query, opts.MinRetrievals)
	if err != nil {
		return err
	}

	type candidate struct {
		id             int64
		content        string
		retrievalCount int
		uniqueProjects int
	}

	var candidates []candidate
	for rows.Next() {
		var c candidate
		var projectsStr string
		if err := rows.Scan(&c.id, &c.content, &c.retrievalCount, &projectsStr); err != nil {
			_ = rows.Close()
			return err
		}

		// Count unique projects
		if projectsStr != "" {
			projectMap := make(map[string]bool)
			for _, p := range strings.Split(projectsStr, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					projectMap[p] = true
				}
			}
			c.uniqueProjects = len(projectMap)
		}

		if c.uniqueProjects >= opts.MinProjects {
			candidates = append(candidates, c)
		}
	}
	_ = rows.Close()

	result.PromotionCandidates = len(candidates)

	for _, c := range candidates {
		approved := opts.AutoApprove
		if !approved && opts.ReviewFunc != nil {
			var err error
			desc := fmt.Sprintf("[%d retrievals, %d projects] %s", c.retrievalCount, c.uniqueProjects, c.content)
			approved, err = opts.ReviewFunc("promote", desc)
			if err != nil {
				return err
			}
		}

		if approved {
			result.PromotionsApproved++

			// Set promoted flag in DB
			now := time.Now().Format(time.RFC3339)
			_, _ = db.Exec("UPDATE embeddings SET promoted = 1, promoted_at = ? WHERE id = ?", now, c.id)

			// Extract message content for CLAUDE.md
			msg := extractMessageContent(c.content)
			if msg == "" {
				msg = c.content
			}

			// Append to CLAUDE.md
			if err := appendToClaudeMD(opts.ClaudeMDPath, []string{msg}); err != nil {
				return fmt.Errorf("failed to append to CLAUDE.md: %w", err)
			}
		}
	}

	return nil
}

// optimizeClaudeMDDedup finds and removes redundant entries within CLAUDE.md's Promoted Learnings.
func optimizeClaudeMDDedup(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	content, err := os.ReadFile(opts.ClaudeMDPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	sections := ParseCLAUDEMD(string(content))
	promoted, ok := sections["Promoted Learnings"]
	if !ok || len(promoted) < 2 {
		return nil
	}

	// Initialize ONNX for embedding comparison
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil // Skip dedup if we can't get home dir
	}
	modelDir := filepath.Join(homeDir, ".claude", "models")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return nil
	}
	if err := initializeONNXRuntime(modelDir); err != nil {
		return nil
	}
	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil // Can't dedup without model
	}

	// Parse learning entries
	type learningEntry struct {
		line    string
		content string
	}
	var entries []learningEntry
	for _, line := range promoted {
		trimmed := strings.TrimSpace(line)
		entry := strings.TrimPrefix(trimmed, "- ")
		if entry != "" {
			entries = append(entries, learningEntry{line: line, content: entry})
		}
	}

	if len(entries) < 2 {
		return nil
	}

	// Generate embeddings for each entry
	type embEntry struct {
		learningEntry
		embedding []float32
	}
	var embEntries []embEntry
	for _, e := range entries {
		emb, _, _, err := generateEmbeddingONNX(e.content, modelPath)
		if err != nil {
			continue
		}
		embEntries = append(embEntries, embEntry{learningEntry: e, embedding: emb})
	}

	// Find duplicates (similarity > 0.9)
	toRemove := make(map[int]bool)
	for i := 0; i < len(embEntries); i++ {
		if toRemove[i] {
			continue
		}
		for j := i + 1; j < len(embEntries); j++ {
			if toRemove[j] {
				continue
			}
			sim := cosineSimilarity(embEntries[i].embedding, embEntries[j].embedding)
			if sim > 0.9 {
				// Remove the later one (keep earlier)
				toRemove[j] = true
				result.ClaudeMDDeduped++
			}
		}
	}

	if len(toRemove) == 0 {
		return nil
	}

	// Collect content strings to remove
	var removeStrings []string
	for idx := range toRemove {
		removeStrings = append(removeStrings, embEntries[idx].content)
	}

	return RemoveFromClaudeMD(opts.ClaudeMDPath, removeStrings)
}

// cosineSimilarity calculates cosine similarity between two float32 vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
