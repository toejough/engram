package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

// ConsolidateOpts holds options for memory consolidation.
type ConsolidateOpts struct {
	MemoryRoot           string
	DecayFactor          float64 // Decay factor (default: 0.9)
	PruneThreshold       float64 // Confidence threshold for pruning (default: 0.1)
	DuplicateThreshold   float64 // Similarity threshold for duplicates (default: 0.95)
	MinRetrievals        int     // Minimum retrieval count for promotion (default: 3)
	MinProjects          int     // Minimum unique projects for promotion (default: 2)
	EnableSynthesis      bool    // Enable pattern synthesis (ISSUE-179)
	SynthesisThreshold   float64 // Similarity threshold for clustering (default: 0.8)
	MinClusterSize       int     // Minimum cluster size for patterns (default: 3)
}

// ConsolidateResult contains the result of memory consolidation.
type ConsolidateResult struct {
	EntriesDecayed       int
	EntriesPruned        int
	DuplicatesMerged     int
	PromotionCandidates  int
	PatternsIdentified   int // ISSUE-179: Number of synthesized patterns
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

	// Step 5: Synthesize patterns (ISSUE-179)
	if opts.EnableSynthesis {
		synthThreshold := opts.SynthesisThreshold
		if synthThreshold == 0 {
			synthThreshold = 0.8
		}
		minCluster := opts.MinClusterSize
		if minCluster == 0 {
			minCluster = 3
		}
		synthResult, err := SynthesizePatterns(opts.MemoryRoot, synthThreshold, minCluster)
		if err != nil {
			return nil, fmt.Errorf("synthesis failed: %w", err)
		}
		result.PatternsIdentified = len(synthResult.Patterns)
	}

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

		// Delete from FTS5 table if available (rowid matches embeddings.id)
		deleteFTS5(db, id)
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

// ConsolidateClaudeMDOpts holds options for CLAUDE.md consolidation analysis.
type ConsolidateClaudeMDOpts struct {
	MemoryRoot   string
	ClaudeMDPath string                                           // Path to CLAUDE.md (default: ~/.claude/CLAUDE.md)
	ReviewFunc   func(proposal ConsolidateProposal) (bool, error) // For interactive mode
}

// ConsolidateProposal represents a proposed maintenance action for CLAUDE.md.
type ConsolidateProposal struct {
	Type       string  // "redundant", "promote", "stale"
	Content    string  // The learning/section content
	Reason     string  // Why this is proposed
	Similarity float64 // Similarity score if redundancy
	Action     string  // Proposed action: "remove", "add", "update"
}

// ConsolidateClaudeMDResult contains the result of CLAUDE.md consolidation analysis.
type ConsolidateClaudeMDResult struct {
	Proposals      []ConsolidateProposal
	RedundantCount int
	PromoteCount   int
	Applied        int // How many proposals were approved and applied
}

// ParseCLAUDEMD splits CLAUDE.md content on `## ` headers, returning a map
// of section name to the lines in that section (excluding the header itself).
func ParseCLAUDEMD(content string) map[string][]string {
	sections := make(map[string][]string)
	if content == "" {
		return sections
	}

	lines := strings.Split(content, "\n")
	var currentSection string

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			currentSection = strings.TrimPrefix(line, "## ")
			currentSection = strings.TrimSpace(currentSection)
			if _, exists := sections[currentSection]; !exists {
				sections[currentSection] = nil
			}
		} else if currentSection != "" {
			// Only include non-empty lines
			if strings.TrimSpace(line) != "" {
				sections[currentSection] = append(sections[currentSection], line)
			}
		}
	}

	return sections
}

// ConsolidateClaudeMD analyzes CLAUDE.md for redundancy with the memory DB
// and proposes maintenance actions.
func ConsolidateClaudeMD(opts ConsolidateClaudeMDOpts) (*ConsolidateClaudeMDResult, error) {
	if opts.MemoryRoot == "" {
		return nil, fmt.Errorf("memory root is required")
	}

	claudeMDPath := opts.ClaudeMDPath
	if claudeMDPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		claudeMDPath = filepath.Join(homeDir, ".claude", "CLAUDE.md")
	}

	result := &ConsolidateClaudeMDResult{}

	// Read CLAUDE.md content
	content, err := os.ReadFile(claudeMDPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil // No file, no proposals
		}
		return nil, fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	if len(content) == 0 {
		return result, nil
	}

	// Parse into sections
	sections := ParseCLAUDEMD(string(content))

	// Get promoted learnings
	promotedLearnings, hasPromoted := sections["Promoted Learnings"]
	if !hasPromoted || len(promotedLearnings) == 0 {
		return result, nil
	}

	// Open embeddings DB
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Determine model directory and initialize ONNX
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	modelDir := filepath.Join(homeDir, ".claude", "models")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create model directory: %w", err)
	}
	if err := initializeONNXRuntime(modelDir); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}
	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		if err := downloadModel(modelPath); err != nil {
			return nil, fmt.Errorf("failed to download model: %w", err)
		}
	}

	// Check each promoted learning against the embeddings DB
	for _, line := range promotedLearnings {
		// Strip bullet prefix
		learning := strings.TrimSpace(line)
		learning = strings.TrimPrefix(learning, "- ")
		learning = strings.TrimSpace(learning)
		if learning == "" {
			continue
		}

		// Generate embedding for the learning text
		embedding, _, _, err := generateEmbeddingONNX(learning, modelPath)
		if err != nil {
			continue // Skip on error
		}

		// Serialize for sqlite-vec query
		embeddingBlob, err := sqlite_vec.SerializeFloat32(embedding)
		if err != nil {
			continue
		}

		// Search for similar entries in the DB
		query := `
			SELECT e.content,
			       (1 - vec_distance_cosine(v.embedding, ?)) as score
			FROM vec_embeddings v
			JOIN embeddings e ON e.embedding_id = v.rowid
			ORDER BY score DESC
			LIMIT 1
		`
		var dbContent string
		var similarity float64
		err = db.QueryRow(query, embeddingBlob).Scan(&dbContent, &similarity)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			continue
		}

		if similarity > 0.9 {
			proposal := ConsolidateProposal{
				Type:       "redundant",
				Content:    learning,
				Reason:     fmt.Sprintf("already in memory DB (%.0f%% similar to: %s)", similarity*100, truncateContent(dbContent, 60)),
				Similarity: similarity,
				Action:     "remove",
			}
			result.Proposals = append(result.Proposals, proposal)
			result.RedundantCount++
		}
	}

	// Find promotion candidates not already in CLAUDE.md
	promoteResult, err := Promote(PromoteOpts{
		MemoryRoot:    opts.MemoryRoot,
		MinRetrievals: 5,
		MinProjects:   3,
	})
	if err == nil && len(promoteResult.Candidates) > 0 {
		contentStr := string(content)
		for _, candidate := range promoteResult.Candidates {
			msg := extractMessageContent(strings.ToLower(candidate.Content))
			if !strings.Contains(strings.ToLower(contentStr), msg) && msg != "" {
				proposal := ConsolidateProposal{
					Type:    "promote",
					Content: candidate.Content,
					Reason:  fmt.Sprintf("retrieved %d times across %d projects", candidate.RetrievalCount, candidate.UniqueProjects),
					Action:  "add",
				}
				result.Proposals = append(result.Proposals, proposal)
				result.PromoteCount++
			}
		}
	}

	// If ReviewFunc is provided, apply interactive review
	if opts.ReviewFunc != nil && len(result.Proposals) > 0 {
		for _, proposal := range result.Proposals {
			approved, err := opts.ReviewFunc(proposal)
			if err != nil {
				return nil, fmt.Errorf("review failed: %w", err)
			}
			if approved {
				result.Applied++
			}
		}
	}

	return result, nil
}

// truncateContent truncates a string to maxLen, adding ellipsis if truncated.
func truncateContent(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ============================================================================
// ISSUE-179: Pattern synthesis from repeated episodes
// ============================================================================

// SynthesisPattern represents a single identified pattern from clustered memories.
type SynthesisPattern struct {
	Theme       string   // Extracted theme/topic
	Examples    []string // Content of clustered memories
	Synthesis   string   // Generated pattern description
	Occurrences int      // Number of memories in cluster
}

// SynthesisResult contains the result of pattern synthesis.
type SynthesisResult struct {
	Patterns []SynthesisPattern
}

// clusterEntry holds data for a single embedding entry during clustering.
type clusterEntry struct {
	id          int64
	content     string
	embeddingID int64
}

// SynthesizePatterns clusters similar memories and generates patterns.
func SynthesizePatterns(memoryRoot string, threshold float64, minClusterSize int) (*SynthesisResult, error) {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Get all entries with embeddings
	rows, err := db.Query(`
		SELECT e.id, e.content, e.embedding_id
		FROM embeddings e
		WHERE e.embedding_id IS NOT NULL
		ORDER BY e.id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []clusterEntry
	for rows.Next() {
		var e clusterEntry
		if err := rows.Scan(&e.id, &e.content, &e.embeddingID); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	if len(entries) < minClusterSize {
		return &SynthesisResult{}, nil
	}

	// Cluster by similarity
	clusters := clusterBySimilarity(db, entries, threshold)

	// Filter clusters by min size and generate patterns
	var patterns []SynthesisPattern
	for _, cluster := range clusters {
		if len(cluster) >= minClusterSize {
			patterns = append(patterns, generatePattern(cluster))
		}
	}

	return &SynthesisResult{Patterns: patterns}, nil
}

// clusterBySimilarity performs single-linkage clustering using union-find.
func clusterBySimilarity(db *sql.DB, entries []clusterEntry, threshold float64) [][]clusterEntry {
	n := len(entries)
	// Union-find parent array
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	// Compare all pairs
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			sim, err := calculateSimilarity(db, entries[i].embeddingID, entries[j].embeddingID)
			if err != nil {
				continue
			}
			if sim >= threshold {
				union(i, j)
			}
		}
	}

	// Collect clusters
	clusterMap := make(map[int][]clusterEntry)
	for i := 0; i < n; i++ {
		root := find(i)
		clusterMap[root] = append(clusterMap[root], entries[i])
	}

	clusters := make([][]clusterEntry, 0, len(clusterMap))
	for _, c := range clusterMap {
		clusters = append(clusters, c)
	}
	return clusters
}

// synthesisStopWords is a minimal set of stop words for keyword extraction.
var synthesisStopWords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
	"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
	"could": true, "should": true, "may": true, "might": true, "shall": true, "can": true,
	"to": true, "of": true, "in": true, "for": true, "on": true, "with": true,
	"at": true, "by": true, "from": true, "as": true, "into": true, "through": true,
	"during": true, "before": true, "after": true, "above": true, "below": true,
	"between": true, "under": true, "and": true, "but": true, "or": true, "nor": true,
	"not": true, "so": true, "yet": true, "both": true, "either": true, "neither": true,
	"each": true, "every": true, "all": true, "any": true, "few": true, "more": true,
	"most": true, "other": true, "some": true, "such": true, "no": true, "only": true,
	"own": true, "same": true, "than": true, "too": true, "very": true, "just": true,
	"because": true, "if": true, "when": true, "where": true, "how": true, "what": true,
	"which": true, "who": true, "whom": true, "this": true, "that": true, "these": true,
	"those": true, "it": true, "its": true, "i": true, "me": true, "my": true,
	"we": true, "our": true, "you": true, "your": true, "he": true, "him": true,
	"his": true, "she": true, "her": true, "they": true, "them": true, "their": true,
}

// generatePattern extracts common keywords and builds a pattern from a cluster.
func generatePattern(cluster []clusterEntry) SynthesisPattern {
	// Count word frequencies across all entries
	wordCount := make(map[string]int)
	for _, e := range cluster {
		// Extract message content (strip timestamp/project prefix)
		msg := extractMessageContent(strings.ToLower(e.content))
		seen := make(map[string]bool)
		for _, word := range strings.Fields(msg) {
			// Strip punctuation
			word = strings.Trim(word, ".,;:!?\"'()[]{}")
			if word == "" || synthesisStopWords[word] || seen[word] {
				continue
			}
			seen[word] = true
			wordCount[word]++
		}
	}

	// Find words appearing in >50% of entries
	threshold := len(cluster) / 2
	type wordFreq struct {
		word  string
		count int
	}
	var common []wordFreq
	for word, count := range wordCount {
		if count > threshold {
			common = append(common, wordFreq{word, count})
		}
	}

	// Sort by frequency descending
	for i := 0; i < len(common); i++ {
		for j := i + 1; j < len(common); j++ {
			if common[j].count > common[i].count {
				common[i], common[j] = common[j], common[i]
			}
		}
	}

	// Build theme from top 3 keywords
	var themeWords []string
	for i := 0; i < len(common) && i < 3; i++ {
		themeWords = append(themeWords, common[i].word)
	}
	theme := strings.Join(themeWords, ", ")

	// Build examples (first 2 entries, truncated)
	var examples []string
	for _, e := range cluster {
		examples = append(examples, e.content)
	}
	var exampleSnippets []string
	for i := 0; i < len(examples) && i < 2; i++ {
		exampleSnippets = append(exampleSnippets, truncateContent(examples[i], 80))
	}

	synthesis := fmt.Sprintf(
		"Pattern observed across %d memories: [%s]. Examples: %s",
		len(cluster),
		strings.Join(themeWords, ", "),
		strings.Join(exampleSnippets, "; "),
	)

	return SynthesisPattern{
		Theme:       theme,
		Examples:    examples,
		Synthesis:   synthesis,
		Occurrences: len(cluster),
	}
}
