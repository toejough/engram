package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OptimizeOpts holds options for the unified optimization pipeline.
type OptimizeOpts struct {
	MemoryRoot       string
	ClaudeMDPath     string
	DecayBase        float64 // Non-promoted decay base (default 0.9)
	PruneThreshold   float64 // default 0.1
	DupThreshold     float64 // default 0.95
	SynthThreshold   float64 // default 0.8
	MinClusterSize     int     // default 3
	MinRetrievals      int     // default 5
	MinProjects        int     // default 3
	MinSkillUtility    float64 // default 0.8 (TASK-8)
	MinSkillConfidence float64 // default 0.8 (TASK-8)
	MinSkillProjects   int     // default 3 (TASK-8)
	AutoDemoteUtility  float64 // default 0.4 (TASK-8) - skills with utility below this and retrieval_count >= 5 are pruned
	ForceReorg         bool    // TASK-11: Force full skill reorganization regardless of elapsed time
	ReorgThreshold     float64 // TASK-11: Reorganization clustering threshold (default 0.8)
	AutoApprove        bool    // --yes flag
	Extractor        LLMExtractor // Optional LLM extractor for synthesis (ISSUE-188)
	ReviewFunc       func(action string, description string) (bool, error)
	SkillsDir        string              // Directory for generated skill files
	SkillCompiler    SkillCompiler       // Optional compiler for skill content
	SpecificDetector SpecificityDetector // Optional detector for narrow/universal learning classification
	SimilarityFunc   func(db *sql.DB, id1, id2 int64) (float64, error) // TASK-10: Optional similarity function for testing
	Context          context.Context     // Optional context for cancellation support
	TestSkills       bool                // Task 8: Enable skill testing before deployment (default true)
	TestRuns         int                 // Task 8: Number of test runs for RED/GREEN protocol (default 3)
	SkillTester      SkillTester         // Task 8: Optional skill tester for testing (defaults to API-based implementation)
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
	BoilerplatePurged     int
	LegacySessionPurged   int
	DuplicatesMerged      int
	PatternsFound         int
	PatternsApproved      int
	PromotionCandidates   int
	PromotionsApproved    int
	ClaudeMDDeduped       int
	ClaudeMDDemoted       int // TASK-2: Narrow learnings demoted to skills
	SkillsCompiled        int
	SkillsMerged          int
	SkillsPruned          int
	SkillsPromoted        int // TASK-3: Skills promoted from memory to CLAUDE.md
	SkillsSplit           int // TASK-10: Skills split due to incoherence
	SkillsReorganized     int // TASK-11: Number of skills affected by periodic reorganization
	FeedbackPropagated    int // ISSUE-224: Number of skills with propagated feedback
}

// MaintenanceProposal represents a proposed maintenance action for ISSUE-212.
// It describes what tier, what action, what target, why it's proposed, and what the result would look like.
type MaintenanceProposal struct {
	Tier    string // "embeddings", "skills", "claude-md"
	Action  string // "prune", "decay", "consolidate", "split", "promote", "demote"
	Target  string // ID, file path, or entry text
	Reason  string // Why this is proposed
	Preview string // What the result would look like
	LLMEval *LLMEvalResult
}

// LLMEvalResult holds the results of LLM evaluation stages.
type LLMEvalResult struct {
	HaikuValid       bool             // Did Haiku consider this a valid concern?
	HaikuRationale   string           // Haiku's one-line explanation
	SonnetRecommend  string           // "apply" or "skip"
	SonnetConfidence string           // "high", "medium", "low"
	SonnetSummary    string           // Human-readable change analysis
	ScenarioResults  []ScenarioResult // Per-scenario preservation checks
}

// ScenarioResult holds one behavioral test scenario result.
type ScenarioResult struct {
	Prompt    string // Simulated user prompt
	Preserved bool   // Did expected guidance surface?
	Lost      string // What was lost (if not preserved)
}

// MaintenanceReviewFunc is called to review a maintenance proposal.
// It returns true if the proposal is approved, false if rejected.
type MaintenanceReviewFunc func(MaintenanceProposal) bool

// TierScanner scans a memory tier (embeddings, skills, or CLAUDE.md) and returns maintenance proposals.
type TierScanner interface {
	Scan() ([]MaintenanceProposal, error)
}

// ProposalApplier applies a maintenance proposal to the memory system.
type ProposalApplier interface {
	Apply(MaintenanceProposal) error
}

// SkillTester tests skill candidates using RED/GREEN protocol.
type SkillTester interface {
	TestAndEvaluate(scenario TestScenario, runs int) (pass bool, reasoning string, err error)
}

// SkillCandidate represents a skill candidate for testing before deployment.
type SkillCandidate struct {
	Theme           string
	Content         string
	SourceEmbeddings []Embedding
}

// checkContext checks if the context has been cancelled and returns an error if so.
func checkContext(opts OptimizeOpts) error {
	if opts.Context != nil {
		return opts.Context.Err()
	}
	return nil
}

// defaultSkillTester implements SkillTester using the skill test harness.
type defaultSkillTester struct {
	apiKey string
}

func (d *defaultSkillTester) TestAndEvaluate(scenario TestScenario, runs int) (bool, string, error) {
	// Call the harness to run RED/GREEN tests
	ctx := context.Background()
	redResults, greenResults, err := TestSkillCandidate(ctx, scenario, runs, d.apiKey)
	if err != nil {
		return false, "", err
	}

	// Evaluate the results
	pass, reasoning := EvaluateTestResults(redResults, greenResults)
	return pass, reasoning, nil
}

// clusterEntriesToEmbeddings converts ClusterEntry slice to Embedding slice for skill testing.
func clusterEntriesToEmbeddings(cluster []ClusterEntry) []Embedding {
	embeddings := make([]Embedding, 0, len(cluster))
	for _, entry := range cluster {
		embeddings = append(embeddings, Embedding{
			ID:      entry.ID,
			Content: entry.Content,
		})
	}
	return embeddings
}

// TestAndCompileSkill tests a skill candidate and returns an error if tests fail.
// This is a simplified version for testing the integration.
func TestAndCompileSkill(opts OptimizeOpts, candidate SkillCandidate) error {
	// Skip testing if TestSkills is false
	if !opts.TestSkills {
		return nil
	}

	// Derive scenario from embeddings
	scenario := DeriveScenarioFromEmbeddings(candidate.SourceEmbeddings)
	scenario.SkillContent = candidate.Content
	scenario.SkillName = candidate.Theme

	// Test the skill
	runs := opts.TestRuns
	if runs == 0 {
		runs = 3
	}

	pass, reasoning, err := opts.SkillTester.TestAndEvaluate(scenario, runs)
	if err != nil {
		// Log test failure to changelog
		_ = WriteChangelogEntry(opts.MemoryRoot, ChangelogEntry{
			Action:         "skill_test_fail",
			SourceTier:     "skills",
			ContentSummary: candidate.Theme,
			Reason:         fmt.Sprintf("test error: %v", err),
		})
		return fmt.Errorf("skill test failed: %w", err)
	}

	if !pass {
		// Log test failure to changelog
		_ = WriteChangelogEntry(opts.MemoryRoot, ChangelogEntry{
			Action:         "skill_test_fail",
			SourceTier:     "skills",
			ContentSummary: candidate.Theme,
			Reason:         reasoning,
		})
		return fmt.Errorf("skill test failed: %s", reasoning)
	}

	// Log test success to changelog
	_ = WriteChangelogEntry(opts.MemoryRoot, ChangelogEntry{
		Action:         "skill_test_pass",
		SourceTier:     "skills",
		ContentSummary: candidate.Theme,
		Reason:         reasoning,
	})

	return nil
}

// Optimize runs the unified memory optimization pipeline.
// Steps: time-decay → contradiction detection → auto-demote → prune → dedup → synthesize → compile skills → merge skills → split skills → promote → dedup CLAUDE.md.
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
	if opts.MinSkillUtility == 0 {
		opts.MinSkillUtility = 0.8
	}
	if opts.MinSkillConfidence == 0 {
		opts.MinSkillConfidence = 0.8
	}
	if opts.MinSkillProjects == 0 {
		opts.MinSkillProjects = 3
	}
	if opts.AutoDemoteUtility == 0 {
		opts.AutoDemoteUtility = 0.4
	}
	if opts.SimilarityFunc == nil {
		opts.SimilarityFunc = calculateSimilarity
	}
	if opts.ReorgThreshold == 0 {
		opts.ReorgThreshold = 0.8
	}

	// Task 8: Set defaults for skill testing
	// TestSkills defaults to true (test skills before deployment)
	// TestRuns defaults to 3 (run 3 RED and 3 GREEN tests)
	if opts.TestRuns == 0 {
		opts.TestRuns = 3
	}
	// Set default SkillTester if not provided
	if opts.SkillTester == nil {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		opts.SkillTester = &defaultSkillTester{apiKey: apiKey}
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
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizeDecay(db, opts, result); err != nil {
		return nil, fmt.Errorf("decay failed: %w", err)
	}
	if result.EntriesDecayed > 0 {
		logChangelogMutation(opts.MemoryRoot, "decay", "embeddings", "embeddings",
			fmt.Sprintf("decayed %d entries (factor=%.4f)", result.EntriesDecayed, result.DecayFactor))
	}

	// Step 2: Contradiction detection
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizeContradictions(db, opts, result); err != nil {
		return nil, fmt.Errorf("contradiction detection failed: %w", err)
	}
	if result.ContradictionsFound > 0 {
		logChangelogMutation(opts.MemoryRoot, "demote", "embeddings", "embeddings",
			fmt.Sprintf("penalized %d contradicted entries", result.ContradictionsFound))
	}

	// Step 3: Auto-demote
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizeAutoDemote(db, opts, result); err != nil {
		return nil, fmt.Errorf("auto-demote failed: %w", err)
	}
	if result.AutoDemoted > 0 {
		logChangelogMutation(opts.MemoryRoot, "demote", "claude-md", "embeddings",
			fmt.Sprintf("auto-demoted %d low-confidence entries", result.AutoDemoted))
	}

	// Step 4: Prune DB
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizePrune(db, opts, result); err != nil {
		return nil, fmt.Errorf("prune failed: %w", err)
	}
	if result.EntriesPruned > 0 {
		logChangelogMutation(opts.MemoryRoot, "prune", "embeddings", "",
			fmt.Sprintf("pruned %d entries below threshold %.2f", result.EntriesPruned, opts.PruneThreshold))
	}

	// Step 4.5: Purge boilerplate session entries
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizePurgeBoilerplate(db, opts, result); err != nil {
		return nil, fmt.Errorf("boilerplate purge failed: %w", err)
	}
	if result.BoilerplatePurged > 0 {
		logChangelogMutation(opts.MemoryRoot, "prune", "embeddings", "",
			fmt.Sprintf("purged %d boilerplate entries", result.BoilerplatePurged))
	}

	// Step 4.6: Purge legacy session embeddings created by old Query() behavior
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizePurgeLegacySessionEmbeddings(db, opts, result); err != nil {
		return nil, fmt.Errorf("legacy session purge failed: %w", err)
	}
	if result.LegacySessionPurged > 0 {
		logChangelogMutation(opts.MemoryRoot, "prune", "embeddings", "",
			fmt.Sprintf("purged %d legacy session embeddings", result.LegacySessionPurged))
	}

	// Step 5: Deduplicate DB
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizeDedup(db, opts, result); err != nil {
		return nil, fmt.Errorf("dedup failed: %w", err)
	}
	if result.DuplicatesMerged > 0 {
		logChangelogMutation(opts.MemoryRoot, "dedup", "embeddings", "embeddings",
			fmt.Sprintf("merged %d duplicate entries", result.DuplicatesMerged))
	}

	// Step 6: Synthesize patterns (interactive)
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizeSynthesize(db, opts, result); err != nil {
		return nil, fmt.Errorf("synthesis failed: %w", err)
	}
	if result.PatternsApproved > 0 {
		logChangelogMutation(opts.MemoryRoot, "merge", "embeddings", "embeddings",
			fmt.Sprintf("synthesized %d patterns from clusters", result.PatternsApproved))
	}

	// Step 7: Compile Skills (memories → skills)
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizeCompileSkills(db, opts, result); err != nil {
		return nil, fmt.Errorf("skill compilation failed: %w", err)
	}
	if result.SkillsCompiled > 0 {
		logChangelogMutation(opts.MemoryRoot, "promote", "embeddings", "skills",
			fmt.Sprintf("compiled %d new skills", result.SkillsCompiled))
	}

	// Step 8: Merge similar skills (TASK-10)
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizeMergeSkills(db, opts, result); err != nil {
		return nil, fmt.Errorf("skill merge failed: %w", err)
	}
	if result.SkillsMerged > 0 {
		logChangelogMutation(opts.MemoryRoot, "merge", "skills", "skills",
			fmt.Sprintf("merged %d similar skills", result.SkillsMerged))
	}

	// Step 9: Split incoherent skills (TASK-10)
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizeSplitSkills(db, opts, result); err != nil {
		return nil, fmt.Errorf("skill split failed: %w", err)
	}
	if result.SkillsSplit > 0 {
		logChangelogMutation(opts.MemoryRoot, "split", "skills", "skills",
			fmt.Sprintf("split %d incoherent skills", result.SkillsSplit))
	}

	// Step 10: Promote (interactive)
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizePromote(db, opts, result); err != nil {
		return nil, fmt.Errorf("promote failed: %w", err)
	}
	if result.PromotionsApproved > 0 {
		logChangelogMutation(opts.MemoryRoot, "promote", "embeddings", "claude-md",
			fmt.Sprintf("promoted %d entries to CLAUDE.md", result.PromotionsApproved))
	}

	// Step 11: Deduplicate CLAUDE.md
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizeClaudeMDDedup(db, opts, result); err != nil {
		return nil, fmt.Errorf("CLAUDE.md dedup failed: %w", err)
	}
	if result.ClaudeMDDeduped > 0 {
		logChangelogMutation(opts.MemoryRoot, "dedup", "claude-md", "claude-md",
			fmt.Sprintf("deduped %d CLAUDE.md entries", result.ClaudeMDDeduped))
	}

	// Step 12: Demote narrow learnings from CLAUDE.md to skills
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizeDemoteClaudeMD(db, opts, result); err != nil {
		return nil, fmt.Errorf("CLAUDE.md demotion failed: %w", err)
	}
	if result.ClaudeMDDemoted > 0 {
		logChangelogMutation(opts.MemoryRoot, "demote", "claude-md", "skills",
			fmt.Sprintf("demoted %d narrow learnings to skills", result.ClaudeMDDemoted))
	}

	// Step 13: Promote high-utility skills to CLAUDE.md (TASK-3)
	if err := checkContext(opts); err != nil {
		return nil, err
	}
	if err := optimizePromoteSkills(db, opts, result); err != nil {
		return nil, fmt.Errorf("skill promotion failed: %w", err)
	}
	if result.SkillsPromoted > 0 {
		logChangelogMutation(opts.MemoryRoot, "promote", "skills", "claude-md",
			fmt.Sprintf("promoted %d skills to CLAUDE.md", result.SkillsPromoted))
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
	if err := RemoveFromClaudeMD(RealFS{}, opts.ClaudeMDPath, toRemove); err != nil {
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

// optimizePurgeBoilerplate removes boilerplate session summary entries.
func optimizePurgeBoilerplate(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Get non-promoted entries (need embedding_id for cleanup)
	rows, err := db.Query("SELECT id, embedding_id, content FROM embeddings WHERE promoted = 0")
	if err != nil {
		return err
	}

	type purgeEntry struct {
		id          int64
		embeddingID int64
		content     string
	}
	var toPurge []purgeEntry
	for rows.Next() {
		var e purgeEntry
		var embID sql.NullInt64
		if err := rows.Scan(&e.id, &embID, &e.content); err != nil {
			_ = rows.Close()
			return err
		}
		if embID.Valid {
			e.embeddingID = embID.Int64
		}
		// Check both raw content and extracted message content
		// (Learn() wraps messages with timestamps like "- 2026-02-10 18:15: message")
		msg := extractMessageContent(e.content)
		if IsSessionBoilerplate(e.content) || (msg != "" && IsSessionBoilerplate(msg)) {
			toPurge = append(toPurge, e)
		}
	}
	_ = rows.Close()

	// Delete entries and clean up vec/FTS5
	for _, e := range toPurge {
		_, _ = db.Exec("DELETE FROM embeddings WHERE id = ?", e.id)
		if e.embeddingID > 0 {
			_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", e.embeddingID)
		}
		deleteFTS5(db, e.id)
	}

	result.BoilerplatePurged = len(toPurge)
	return nil
}

// optimizePurgeLegacySessionEmbeddings removes legacy session embeddings created by old Query() behavior.
// Learn() stores content with timestamp prefix: "- 2026-02-10 15:04: message" or "- 2026-02-10 15:04: [project] message"
// Old createEmbeddings() stored raw session lines WITHOUT timestamp prefix.
// Heuristic: non-promoted entries where content does NOT match Learn() timestamp format AND has empty observation_type and memory_type.
// ISSUE-210: Preserve entries with retrieval_count > 0 (they're being actively used, not legacy).
func optimizePurgeLegacySessionEmbeddings(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Query non-promoted entries with empty observation_type and memory_type
	rows, err := db.Query(`
		SELECT id, embedding_id, content, retrieval_count
		FROM embeddings
		WHERE promoted = 0
		  AND observation_type = ''
		  AND memory_type = ''
		  AND source = 'memory'
	`)
	if err != nil {
		return err
	}

	type purgeEntry struct {
		id             int64
		embeddingID    int64
		content        string
		retrievalCount int
	}
	var candidates []purgeEntry
	for rows.Next() {
		var e purgeEntry
		var embID sql.NullInt64
		if err := rows.Scan(&e.id, &embID, &e.content, &e.retrievalCount); err != nil {
			_ = rows.Close()
			return err
		}
		if embID.Valid {
			e.embeddingID = embID.Int64
		}
		candidates = append(candidates, e)
	}
	_ = rows.Close()

	// Filter: keep entries whose content matches Learn() timestamp format OR have retrieval_count > 0
	// Remaining entries are legacy session embeddings
	var toPurge []purgeEntry
	for _, e := range candidates {
		// Skip entries with retrieval_count > 0 (they're being used, not legacy)
		if e.retrievalCount > 0 {
			continue
		}
		// Check if content starts with Learn() timestamp format: "- YYYY-MM-DD HH:MM:"
		// Use simple string matching instead of regex for performance
		if !hasLearnTimestampPrefix(e.content) {
			toPurge = append(toPurge, e)
		}
	}

	// Delete legacy session embeddings
	for _, e := range toPurge {
		_, _ = db.Exec("DELETE FROM embeddings WHERE id = ?", e.id)
		if e.embeddingID > 0 {
			_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", e.embeddingID)
		}
		deleteFTS5(db, e.id)
	}

	result.LegacySessionPurged = len(toPurge)
	return nil
}

// hasLearnTimestampPrefix checks if content starts with Learn() timestamp format: "- YYYY-MM-DD HH:MM:"
func hasLearnTimestampPrefix(content string) bool {
	// Learn() format: "- 2026-02-10 15:04: message" or "- 2026-02-10 15:04: [project] message"
	if len(content) < 19 { // Minimum: "- YYYY-MM-DD HH:MM:"
		return false
	}
	if !strings.HasPrefix(content, "- ") {
		return false
	}
	// Check format: "YYYY-MM-DD HH:MM:"
	// Position 2-6: YYYY-
	// Position 7-9: MM-
	// Position 10-12: DD
	// Position 13: space
	// Position 14-16: HH:
	// Position 17-19: MM:
	if len(content) < 20 {
		return false
	}
	part := content[2:20]
	// Check pattern: DDDD-DD-DD DD:DD:
	if len(part) != 18 {
		return false
	}
	// Quick check for dashes and colons in expected positions
	if part[4] != '-' || part[7] != '-' || part[10] != ' ' || part[13] != ':' || part[16] != ':' {
		return false
	}
	// Check digits in expected positions
	for _, i := range []int{0, 1, 2, 3, 5, 6, 8, 9, 11, 12, 14, 15} {
		if part[i] < '0' || part[i] > '9' {
			return false
		}
	}
	return true
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

// getExistingPatterns retrieves existing synthesized patterns from the database.
func getExistingPatterns(db *sql.DB) []string {
	rows, err := db.Query(`
		SELECT content FROM embeddings
		WHERE source = 'synthesized'
		  AND memory_type = 'reflection'
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return []string{}
	}
	defer func() { _ = rows.Close() }()

	var patterns []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			continue
		}
		patterns = append(patterns, content)
	}

	return patterns
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

	var entries []ClusterEntry
	for rows.Next() {
		var e ClusterEntry
		if err := rows.Scan(&e.ID, &e.Content, &e.EmbeddingID); err != nil {
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
		// Check context inside loop
		if err := checkContext(opts); err != nil {
			return err
		}

		if len(cluster) < opts.MinClusterSize {
			continue
		}

		var pattern SynthesisPattern
		if opts.Extractor != nil {
			ctx := opts.Context
			if ctx == nil {
				ctx = context.Background()
			}
			pattern = GeneratePatternLLM(ctx, cluster, opts.Extractor)
		} else {
			pattern = generatePattern(cluster)
		}
		result.PatternsFound++

		// Validate synthesis quality (Task 9)
		existingPatterns := getExistingPatterns(db)
		validation := ValidateSynthesis(pattern.Synthesis, existingPatterns)

		// Reject low-quality synthesis (quality floor 0.8)
		if validation.Quality < 0.8 {
			logChangelogMutation(opts.MemoryRoot, "reject", "synthesis", "",
				fmt.Sprintf("rejected pattern (quality=%.2f): %s. Issues: %v", validation.Quality, pattern.Synthesis, validation.Issues))
			continue
		}

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
				_ = db.QueryRow("SELECT embedding_id FROM embeddings WHERE id = ?", e.ID).Scan(&embeddingID)
				_, _ = db.Exec("DELETE FROM embeddings WHERE id = ?", e.ID)
				if embeddingID > 0 {
					_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", embeddingID)
				}
				deleteFTS5(db, e.ID)
			}
		}
	}

	return nil
}

// optimizePromote finds high-value memories for promotion to CLAUDE.md.
// ISSUE-210: Only promote enriched entries (principle != '').
func optimizePromote(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Find candidates: high retrieval count, multiple projects, not yet promoted
	query := `
		SELECT id, content, principle, retrieval_count, projects_retrieved
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
		principle      string
		retrievalCount int
		uniqueProjects int
	}

	var candidates []candidate
	for rows.Next() {
		var c candidate
		var projectsStr string
		if err := rows.Scan(&c.id, &c.content, &c.principle, &c.retrievalCount, &projectsStr); err != nil {
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

	// Load existing Promoted Learnings from CLAUDE.md for non-redundancy check
	var existingEmbeddings []struct {
		content   string
		embedding []float32
	}

	claudeContent, err := os.ReadFile(opts.ClaudeMDPath)
	if err == nil {
		sections := ParseCLAUDEMD(string(claudeContent))
		if promoted, ok := sections["Promoted Learnings"]; ok {
			// Initialize ONNX for embedding generation
			homeDir, _ := os.UserHomeDir()
			if homeDir != "" {
				modelDir := filepath.Join(homeDir, ".claude", "models")
				_ = os.MkdirAll(modelDir, 0755)
				if initializeONNXRuntime(modelDir) == nil {
					modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
					if _, err := os.Stat(modelPath); err == nil {
						// Generate embeddings for existing learnings
						for _, line := range promoted {
							trimmed := strings.TrimSpace(line)
							learning := strings.TrimPrefix(trimmed, "- ")
							if learning != "" {
								emb, _, _, err := generateEmbeddingONNX("passage: "+learning, modelPath)
								if err == nil {
									existingEmbeddings = append(existingEmbeddings, struct {
										content   string
										embedding []float32
									}{content: learning, embedding: emb})
								}
							}
						}
					}
				}
			}
		}
	}

	// Collect approved principles to batch append to CLAUDE.md
	var approvedPrinciples []string
	var approvedIDs []int64

	for _, c := range candidates {
		// ISSUE-210: Skip unenriched entries (principle = '')
		if c.principle == "" {
			continue
		}

		// ISSUE-215: Non-redundancy check - skip if similar to existing Promoted Learning
		if len(existingEmbeddings) > 0 {
			homeDir, _ := os.UserHomeDir()
			if homeDir != "" {
				modelDir := filepath.Join(homeDir, ".claude", "models")
				modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
				if _, err := os.Stat(modelPath); err == nil {
					candidateEmb, _, _, err := generateEmbeddingONNX("passage: "+c.principle, modelPath)
					if err == nil {
						// Check similarity against existing learnings
						isDuplicate := false
						for _, existing := range existingEmbeddings {
							sim := cosineSimilarity(candidateEmb, existing.embedding)
							if sim > 0.9 {
								isDuplicate = true
								break
							}
						}
						if isDuplicate {
							continue // Skip this candidate
						}
					}
				}
			}
		}

		approved := opts.AutoApprove
		if !approved && opts.ReviewFunc != nil {
			var err error
			desc := fmt.Sprintf("[%d retrievals, %d projects] %s", c.retrievalCount, c.uniqueProjects, c.principle)
			approved, err = opts.ReviewFunc("promote", desc)
			if err != nil {
				return err
			}
		}

		if approved {
			approvedPrinciples = append(approvedPrinciples, c.principle)
			approvedIDs = append(approvedIDs, c.id)
		}
	}

	// Batch update DB and append to CLAUDE.md
	if len(approvedPrinciples) > 0 {
		now := time.Now().Format(time.RFC3339)
		for _, id := range approvedIDs {
			_, _ = db.Exec("UPDATE embeddings SET promoted = 1, promoted_at = ? WHERE id = ?", now, id)
		}

		// Append all approved principles to CLAUDE.md at once
		if err := appendToClaudeMD(opts.ClaudeMDPath, approvedPrinciples); err != nil {
			return fmt.Errorf("failed to append to CLAUDE.md: %w", err)
		}

		result.PromotionsApproved = len(approvedPrinciples)
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
		emb, _, _, err := generateEmbeddingONNX("passage: "+e.content, modelPath)
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

	return RemoveFromClaudeMD(RealFS{}, opts.ClaudeMDPath, removeStrings)
}

// migrateMemoryGenSkills migrates skills from legacy memory-gen/ to mem-{slug}/ structure.
func migrateMemoryGenSkills(fs FileSystem, skillsDir string) error {
	memGenDir := filepath.Join(skillsDir, "memory-gen")
	entries, err := fs.ReadDir(memGenDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No memory-gen dir, nothing to migrate
		}
		return fmt.Errorf("failed to read memory-gen directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		oldPath := filepath.Join(memGenDir, entry.Name())
		newPath := filepath.Join(skillsDir, "mem-"+entry.Name())

		// Skip if destination already exists
		if _, err := fs.Stat(newPath); err == nil {
			continue
		}

		if err := fs.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to migrate skill %s: %w", entry.Name(), err)
		}
	}

	// Remove empty memory-gen directory
	_ = fs.Remove(memGenDir)
	return nil
}

// optimizeCompileSkills clusters similar memories and compiles them into skills.
func optimizeCompileSkills(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Skip if no skills directory configured
	if opts.SkillsDir == "" {
		return nil
	}

	// Ensure skills directory exists
	if err := os.MkdirAll(opts.SkillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Migrate legacy memory-gen/ skills to mem-{slug}/ structure
	if err := migrateMemoryGenSkills(RealFS{}, opts.SkillsDir); err != nil {
		return fmt.Errorf("failed to migrate memory-gen skills: %w", err)
	}

	// TASK-11: Check if periodic reorganization should trigger
	shouldReorg := opts.ForceReorg
	if !shouldReorg {
		lastReorgStr, err := getMetadata(db, "last_skill_reorg_at")
		if err != nil {
			return fmt.Errorf("failed to get last_skill_reorg_at: %w", err)
		}

		if lastReorgStr != "" {
			lastReorg, err := time.Parse(time.RFC3339, lastReorgStr)
			if err == nil {
				daysSince := time.Since(lastReorg).Hours() / 24.0
				shouldReorg = daysSince > 30
			}
		} else {
			// Never reorganized before - should reorganize
			shouldReorg = true
		}
	}

	// TASK-11: Perform full reorganization if triggered
	if shouldReorg {
		if err := performSkillReorganization(db, opts, result); err != nil {
			return fmt.Errorf("failed to perform skill reorganization: %w", err)
		}
		return nil
	}

	// Get existing skill source memory IDs to skip already-compiled clusters
	existingSourceIDs, err := getExistingSkillSourceIDs(db)
	if err != nil {
		return err
	}

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

	var entries []ClusterEntry
	for rows.Next() {
		var e ClusterEntry
		if err := rows.Scan(&e.ID, &e.Content, &e.EmbeddingID); err != nil {
			_ = rows.Close()
			return err
		}
		entries = append(entries, e)
	}
	_ = rows.Close()

	minCluster := opts.MinClusterSize
	if minCluster == 0 {
		minCluster = 3
	}

	// TASK-10: Merge/split existing skills (runs regardless of new memory compilation)
	if err := optimizeMergeSkills(db, opts, result); err != nil {
		return fmt.Errorf("skill merge failed: %w", err)
	}
	if err := optimizeSplitSkills(db, opts, result); err != nil {
		return fmt.Errorf("skill split failed: %w", err)
	}

	if len(entries) < minCluster {
		// Propagate embedding feedback to skills before pruning
		propagated, err := PropagateEmbeddingFeedbackToSkills(db)
		if err == nil && propagated > 0 {
			result.FeedbackPropagated = propagated
		}
		// Still prune stale skills even if nothing to compile
		result.SkillsPruned = pruneStaleSkills(db, opts.SkillsDir, opts.AutoDemoteUtility)
		return nil
	}

	threshold := opts.SynthThreshold
	if threshold == 0 {
		threshold = 0.8
	}

	clusters := clusterBySimilarity(db, entries, threshold)

	for _, cluster := range clusters {
		// Check context inside loop
		if err := checkContext(opts); err != nil {
			return err
		}

		if len(cluster) < minCluster {
			continue
		}

		// Skip clusters whose members already belong to an existing skill
		if clusterHasExistingSkill(cluster, existingSourceIDs) {
			continue
		}

		// Score cluster
		score, err := scoreCluster(db, cluster)
		if err != nil {
			continue
		}

		// Generate theme from cluster content
		theme := generateThemeFromCluster(cluster)
		slug := slugify(theme)

		// Generate skill content
		ctx := opts.Context
		if ctx == nil {
			ctx = context.Background()
		}
		content, err := generateSkillContent(ctx, theme, cluster, opts.SkillCompiler)
		if err != nil {
			continue
		}

		// Build description (first 1500 chars of content, potentially multi-line)
		description := ExtractSkillDescription(content, 1500)

		// Build source memory IDs
		sourceIDs := formatClusterSourceIDs(cluster)

		now := time.Now().UTC().Format(time.RFC3339)
		skill := &GeneratedSkill{
			Slug:            slug,
			Theme:           theme,
			Description:     description,
			Content:         content,
			SourceMemoryIDs: sourceIDs,
			Alpha:           1.0,
			Beta:            1.0,
			Utility:         score,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		// Check for merge candidate (existing skill with similar theme)
		merged := false
		existing, err := getSkillBySlug(db, slug)
		if err == nil && existing != nil && !existing.Pruned {
			// Test the merged skill before updating
			candidate := SkillCandidate{
				Theme:            theme,
				Content:          content,
				SourceEmbeddings: clusterEntriesToEmbeddings(cluster),
			}
			if err := TestAndCompileSkill(opts, candidate); err != nil {
				// Test failed, skip merge
				continue
			}

			// Merge: update existing skill
			existing.Content = content
			existing.SourceMemoryIDs = sourceIDs
			existing.UpdatedAt = now
			existing.Alpha += 1.0 // Positive reinforcement
			existing.Utility = computeUtility(existing.Alpha, existing.Beta, existing.RetrievalCount, existing.LastRetrieved)
			if err := updateSkill(db, existing); err == nil {
				_ = writeSkillFile(opts.SkillsDir, existing)
				result.SkillsMerged++
				merged = true
			}
		}

		if !merged {
			// Test the new skill before inserting
			candidate := SkillCandidate{
				Theme:            theme,
				Content:          content,
				SourceEmbeddings: clusterEntriesToEmbeddings(cluster),
			}
			if err := TestAndCompileSkill(opts, candidate); err != nil {
				// Test failed, skip insertion
				continue
			}

			_, err = insertSkill(db, skill)
			if err != nil {
				continue
			}

			_ = writeSkillFile(opts.SkillsDir, skill)
			result.SkillsCompiled++
		}
	}

	// Propagate embedding feedback to skills
	propagated, err := PropagateEmbeddingFeedbackToSkills(db)
	if err == nil && propagated > 0 {
		result.FeedbackPropagated = propagated
	}

	// Prune stale skills with configurable threshold
	result.SkillsPruned = pruneStaleSkills(db, opts.SkillsDir, opts.AutoDemoteUtility)

	return nil
}

// performSkillReorganization performs a full reorganization of skills by re-clustering
// all memories and updating/creating/pruning skills accordingly.
func performSkillReorganization(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Fetch ALL memories with embeddings
	rows, err := db.Query(`
		SELECT e.id, e.content, e.embedding_id
		FROM embeddings e
		WHERE e.embedding_id IS NOT NULL
		ORDER BY e.id
	`)
	if err != nil {
		return err
	}

	var allMemories []ClusterEntry
	for rows.Next() {
		var e ClusterEntry
		if err := rows.Scan(&e.ID, &e.Content, &e.EmbeddingID); err != nil {
			_ = rows.Close()
			return err
		}
		allMemories = append(allMemories, e)
	}
	_ = rows.Close()

	minCluster := opts.MinClusterSize
	if minCluster == 0 {
		minCluster = 3
	}

	if len(allMemories) < minCluster {
		// Not enough memories to reorganize
		now := time.Now().UTC().Format(time.RFC3339)
		return setMetadata(db, "last_skill_reorg_at", now)
	}

	// Re-cluster at configured threshold (default 0.8, stricter than incremental compilation)
	threshold := opts.ReorgThreshold
	if threshold == 0 {
		threshold = 0.8
	}
	clusters := clusterBySimilarity(db, allMemories, threshold)

	// Track which themes are still present
	activeThemes := make(map[string]bool)

	// Process each cluster
	for _, cluster := range clusters {
		// Check context inside loop
		if err := checkContext(opts); err != nil {
			return err
		}

		if len(cluster) < minCluster {
			continue
		}

		// Generate theme from cluster
		theme := generateThemeFromCluster(cluster)
		slug := slugify(theme)
		activeThemes[slug] = true

		// Score cluster
		score, err := scoreCluster(db, cluster)
		if err != nil {
			continue
		}

		// Generate skill content
		ctx := opts.Context
		if ctx == nil {
			ctx = context.Background()
		}
		content, err := generateSkillContent(ctx, theme, cluster, opts.SkillCompiler)
		if err != nil {
			continue
		}

		// Build description
		description := ExtractSkillDescription(content, 1500)

		// Build source memory IDs
		sourceIDs := formatClusterSourceIDs(cluster)

		// Check if skill already exists
		existing, err := getSkillBySlug(db, slug)
		if err == nil && existing != nil && !existing.Pruned {
			// Update existing skill: regenerate content, preserve alpha/beta
			existing.Content = content
			existing.SourceMemoryIDs = sourceIDs
			existing.Description = description
			existing.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			// Preserve alpha/beta (usage history)
			// Recompute utility with updated retrieval stats
			existing.Utility = computeUtility(existing.Alpha, existing.Beta, existing.RetrievalCount, existing.LastRetrieved)

			if err := updateSkill(db, existing); err == nil {
				_ = writeSkillFile(opts.SkillsDir, existing)
				result.SkillsReorganized++
			}
		} else {
			// Create new skill with fresh prior (Alpha=1, Beta=1)
			now := time.Now().UTC().Format(time.RFC3339)
			skill := &GeneratedSkill{
				Slug:            slug,
				Theme:           theme,
				Description:     description,
				Content:         content,
				SourceMemoryIDs: sourceIDs,
				Alpha:           1.0,
				Beta:            1.0,
				Utility:         score,
				CreatedAt:       now,
				UpdatedAt:       now,
			}

			_, err = insertSkill(db, skill)
			if err != nil {
				continue
			}

			_ = writeSkillFile(opts.SkillsDir, skill)
			result.SkillsReorganized++
		}
	}

	// Prune skills whose themes no longer appear in clusters
	pruned, err := pruneOrphanedSkills(db, opts.SkillsDir, activeThemes)
	if err != nil {
		return err
	}
	result.SkillsReorganized += pruned

	// Update metadata timestamp
	now := time.Now().UTC().Format(time.RFC3339)
	return setMetadata(db, "last_skill_reorg_at", now)
}

// pruneOrphanedSkills soft-deletes skills whose slugs are not in the activeThemes set.
func pruneOrphanedSkills(db *sql.DB, skillsDir string, activeThemes map[string]bool) (int, error) {
	// Get all non-pruned skills
	rows, err := db.Query("SELECT id, slug FROM generated_skills WHERE pruned = 0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()

	type skillRecord struct {
		id   int64
		slug string
	}
	var skills []skillRecord

	for rows.Next() {
		var s skillRecord
		if err := rows.Scan(&s.id, &s.slug); err != nil {
			continue
		}
		skills = append(skills, s)
	}

	// Prune skills not in activeThemes
	pruned := 0
	for _, s := range skills {
		if !activeThemes[s.slug] {
			if err := softDeleteSkill(db, s.id); err != nil {
				continue
			}
			// Remove skill directory from disk
			skillDir := filepath.Join(skillsDir, "mem-"+s.slug)
			_ = os.RemoveAll(skillDir)
			pruned++
		}
	}

	return pruned, nil
}

// getExistingSkillSourceIDs returns a set of embedding IDs that are already
// referenced by non-pruned skills.
func getExistingSkillSourceIDs(db *sql.DB) (map[int64]bool, error) {
	ids := make(map[int64]bool)

	rows, err := db.Query("SELECT source_memory_ids FROM generated_skills WHERE pruned = 0")
	if err != nil {
		return ids, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var sourceIDsStr string
		if err := rows.Scan(&sourceIDsStr); err != nil {
			continue
		}
		var memIDs []int64
		if err := json.Unmarshal([]byte(sourceIDsStr), &memIDs); err != nil {
			continue
		}
		for _, id := range memIDs {
			ids[id] = true
		}
	}

	return ids, rows.Err()
}

// clusterHasExistingSkill checks if any cluster member's ID is already
// referenced by an existing non-pruned skill.
func clusterHasExistingSkill(cluster []ClusterEntry, existingSourceIDs map[int64]bool) bool {
	for _, entry := range cluster {
		if existingSourceIDs[entry.ID] {
			return true
		}
	}
	return false
}

// generateThemeFromCluster produces a theme string from cluster content.
func generateThemeFromCluster(cluster []ClusterEntry) string {
	if len(cluster) == 0 {
		return "Unknown"
	}
	// Use the first entry's content as the basis for the theme
	// Extract the key words (first ~50 chars)
	content := cluster[0].Content
	content = extractMessageContent(content)
	if content == "" {
		content = cluster[0].Content
	}
	if len(content) > 50 {
		content = content[:50]
	}
	// Clean up trailing partial words
	if idx := strings.LastIndex(content, " "); idx > 20 {
		content = content[:idx]
	}
	return strings.TrimSpace(content)
}

// ExtractSkillDescription creates a short description from skill content.
// If content contains structured markers (Core:, Triggers:, Domains:, etc.),
// extracts lines from first marker through last marker (up to maxLen).
// Otherwise, collects multiple non-empty, non-header lines up to maxLen.
func ExtractSkillDescription(content string, maxLen int) string {
	lines := strings.Split(content, "\n")

	// Check for structured markers
	structuredMarkers := []string{"Core:", "Triggers:", "Domains:", "Anti-patterns:", "Related:"}
	firstMarkerIdx := -1
	lastMarkerIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, marker := range structuredMarkers {
			if strings.HasPrefix(trimmed, marker) {
				if firstMarkerIdx == -1 {
					firstMarkerIdx = i
				}
				lastMarkerIdx = i
				break
			}
		}
	}

	// If structured markers found, extract that section
	if firstMarkerIdx != -1 {
		var result strings.Builder
		for i := firstMarkerIdx; i <= lastMarkerIdx && i < len(lines); i++ {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(strings.TrimSpace(lines[i]))
			if result.Len() >= maxLen {
				break
			}
		}
		extracted := result.String()
		if len(extracted) > maxLen {
			return extracted[:maxLen]
		}
		return extracted
	}

	// Otherwise, collect multiple non-empty, non-header lines up to maxLen
	var result strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") {
			continue
		}
		if result.Len() > 0 {
			result.WriteString(" ")
		}
		result.WriteString(trimmed)
		if result.Len() >= maxLen {
			break
		}
	}

	extracted := result.String()
	if len(extracted) > maxLen {
		return extracted[:maxLen]
	}
	if extracted == "" && len(content) > 0 {
		// Fallback: return first maxLen chars of content
		if len(content) > maxLen {
			return strings.TrimSpace(content[:maxLen])
		}
		return strings.TrimSpace(content)
	}
	return extracted
}

// formatClusterSourceIDs returns a JSON array of cluster member IDs.
func formatClusterSourceIDs(cluster []ClusterEntry) string {
	ids := make([]int64, len(cluster))
	for i, entry := range cluster {
		ids[i] = entry.ID
	}
	data, _ := json.Marshal(ids)
	return string(data)
}

// pruneStaleSkills soft-deletes skills with utility < threshold and retrieval_count >= 5,
// and removes their files from disk.
func pruneStaleSkills(db *sql.DB, skillsDir string, autoDemoteUtility float64) int {
	rows, err := db.Query(`
		SELECT id, slug FROM generated_skills
		WHERE pruned = 0
		  AND utility < ?
		  AND retrieval_count >= 5
	`, autoDemoteUtility)
	if err != nil {
		return 0
	}
	defer func() { _ = rows.Close() }()

	type staleSkill struct {
		id   int64
		slug string
	}
	var stale []staleSkill
	for rows.Next() {
		var s staleSkill
		if err := rows.Scan(&s.id, &s.slug); err != nil {
			continue
		}
		stale = append(stale, s)
	}

	pruned := 0
	for _, s := range stale {
		if err := softDeleteSkill(db, s.id); err != nil {
			continue
		}
		// Remove skill directory from disk
		skillDir := filepath.Join(skillsDir, "mem-"+s.slug)
		_ = os.RemoveAll(skillDir)
		pruned++
	}

	return pruned
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

// optimizeDemoteClaudeMD demotes narrow/context-specific learnings from CLAUDE.md to skills.
func optimizeDemoteClaudeMD(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Skip if no skills directory configured
	if opts.SkillsDir == "" {
		return nil
	}

	// Read CLAUDE.md
	content, err := os.ReadFile(opts.ClaudeMDPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	sections := ParseCLAUDEMD(string(content))
	promoted, ok := sections["Promoted Learnings"]
	if !ok || len(promoted) == 0 {
		return nil
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

	if len(entries) == 0 {
		return nil
	}

	// Collect candidates for demotion
	type demoteCandidate struct {
		entry  learningEntry
		reason string
	}
	var candidates []demoteCandidate

	for _, e := range entries {
		isNarrow, reason := false, ""
		var err error

		// Try LLM detection first
		if opts.SpecificDetector != nil {
			ctx := opts.Context
			if ctx == nil {
				ctx = context.Background()
			}
			isNarrow, reason, err = opts.SpecificDetector.IsNarrowLearning(ctx, e.content)
			if err != nil {
				if errors.Is(err, ErrLLMUnavailable) {
					fmt.Fprintf(os.Stderr, "LLM unavailable for specificity detection, using keyword fallback\n")
				}
				// Fall back to keyword heuristics
				isNarrow, reason = isNarrowByKeywords(e.content)
			}
		} else {
			// Use keyword heuristics as fallback
			isNarrow, reason = isNarrowByKeywords(e.content)
		}

		if isNarrow {
			candidates = append(candidates, demoteCandidate{entry: e, reason: reason})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Present candidates for review
	var toRemove []string
	for _, candidate := range candidates {
		approved := opts.AutoApprove
		if !approved && opts.ReviewFunc != nil {
			var err error
			desc := fmt.Sprintf("Demote to skill: [%s] %s", candidate.reason, candidate.entry.content)
			approved, err = opts.ReviewFunc("demote-claude-md", desc)
			if err != nil {
				return err
			}
		}

		// Dry-run mode: if no AutoApprove and no ReviewFunc, just print proposals
		if !opts.AutoApprove && opts.ReviewFunc == nil {
			continue
		}

		if approved {
			// TASK-10: Plan demotion with safety check
			plan := PlanCLAUDEMDDemotion(candidate.entry.content, map[string]interface{}{
				"reason": candidate.reason,
			})

			// Safety check: only proceed if destination is clear
			if !plan.Safe {
				// Log unsafe demotion attempt to changelog
				logChangelogMutation(opts.MemoryRoot, "demote_blocked", "claude-md", "",
					fmt.Sprintf("blocked unsafe demotion: %s - %s", candidate.entry.content, plan.Reasoning))
				continue // Skip unsafe demotion
			}

			// Create destination based on plan
			switch plan.DestinationTier {
			case DestinationSkill:
				// Generate skill from learning
				if err := generateSkillFromLearning(db, opts, candidate.entry.content); err != nil {
					continue // Skip on error, don't fail the entire pipeline
				}
			case DestinationEmbedding:
				// Store as embedding in memory database
				_, err := db.Exec(
					"INSERT INTO embeddings (content, source, source_type, confidence, memory_type) VALUES (?, 'claude-md-demotion', 'internal', 0.8, 'pattern')",
					candidate.entry.content,
				)
				if err != nil {
					continue // Skip on error
				}
			case DestinationHook:
				// For now, log that hook creation is needed (hook creation requires user action)
				logChangelogMutation(opts.MemoryRoot, "demote_hook_needed", "claude-md", "hook",
					fmt.Sprintf("hook needed: %s - %s", candidate.entry.content, plan.Reasoning))
				// Don't remove from CLAUDE.md until hook is created
				continue
			}

			// Log successful demotion to changelog
			logChangelogMutation(opts.MemoryRoot, "demote", "claude-md", string(plan.DestinationTier),
				fmt.Sprintf("demoted: %s - reason: %s", candidate.entry.content, plan.Reasoning))

			// Mark for removal from CLAUDE.md
			toRemove = append(toRemove, candidate.entry.content)
			result.ClaudeMDDemoted++
		}
	}

	// Remove approved entries from CLAUDE.md
	if len(toRemove) > 0 {
		if err := RemoveFromClaudeMD(RealFS{}, opts.ClaudeMDPath, toRemove); err != nil {
			return fmt.Errorf("failed to remove demoted entries from CLAUDE.md: %w", err)
		}
	}

	return nil
}

// isNarrowByKeywords detects narrow/context-specific learnings using keyword heuristics.
// Returns (isNarrow, reason).
func isNarrowByKeywords(learning string) (bool, string) {
	lower := strings.ToLower(learning)

	// Check for project names
	projectKeywords := []string{"projctl", "project ", "repository ", "repo ", "codebase "}
	for _, kw := range projectKeywords {
		if strings.Contains(lower, kw) {
			return true, "Contains project/repository name"
		}
	}

	// Check for file paths
	if strings.Contains(lower, "/users/") || strings.Contains(lower, "/home/") ||
		strings.Contains(lower, "~/") || strings.Contains(learning, "internal/") {
		return true, "Contains file system path"
	}

	// Check for tool names
	toolKeywords := []string{"mage ", "targ ", "claude ", "npm ", "git ", "docker ", "pytest "}
	for _, kw := range toolKeywords {
		if strings.Contains(lower, kw) {
			return true, "Contains specific tool name"
		}
	}

	// Check for technology-specific terms
	techKeywords := []string{"golang", "go test", "python", "typescript", "javascript", ".go", ".py", ".ts", ".js"}
	for _, kw := range techKeywords {
		if strings.Contains(lower, kw) {
			return true, "Contains technology-specific reference"
		}
	}

	return false, ""
}

// generateSkillFromLearning creates a skill file from a demoted learning entry.
func generateSkillFromLearning(db *sql.DB, opts OptimizeOpts, learning string) error {
	// Extract theme (first 50 chars or until first punctuation)
	theme := learning
	if len(theme) > 50 {
		theme = theme[:50]
	}
	// Trim at last space to avoid partial words
	if idx := strings.LastIndex(theme, " "); idx > 20 {
		theme = theme[:idx]
	}
	theme = strings.TrimSpace(theme)

	// Generate slug
	slug := slugify(theme)

	// Generate skill content
	var content string
	if opts.SkillCompiler != nil {
		ctx := opts.Context
		if ctx == nil {
			ctx = context.Background()
		}
		var err error
		content, err = opts.SkillCompiler.CompileSkill(ctx, theme, []string{learning})
		if err != nil {
			if errors.Is(err, ErrLLMUnavailable) {
				fmt.Fprintf(os.Stderr, "LLM unavailable for skill compilation, using template fallback\n")
			}
			// Fall back to template on error
			content = generateSkillTemplate(theme, learning)
		}
	} else {
		content = generateSkillTemplate(theme, learning)
	}

	// Build description
	description := theme
	if len(description) > 200 {
		description = description[:200]
	}

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:        slug,
		Theme:       theme,
		Description: description,
		Content:     content,
		// Empty SourceMemoryIDs since this came from CLAUDE.md, not memory cluster
		SourceMemoryIDs:      "[]",
		Alpha:                1.0,
		Beta:                 1.0,
		Utility:              0.5,
		CreatedAt:            now,
		UpdatedAt:            now,
		DemotedFromClaudeMD:  now, // Mark as demoted from CLAUDE.md
	}

	// Check if skill already exists
	existing, err := getSkillBySlug(db, slug)
	if err == nil && existing != nil && !existing.Pruned {
		// Update existing skill
		existing.Content = content
		existing.UpdatedAt = now
		if err := updateSkill(db, existing); err != nil {
			return err
		}
		return writeSkillFile(opts.SkillsDir, existing)
	}

	// Insert new skill
	_, err = insertSkill(db, skill)
	if err != nil {
		return err
	}

	return writeSkillFile(opts.SkillsDir, skill)
}

// generateSkillTemplate creates a simple template-based skill content.
func generateSkillTemplate(theme, learning string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", theme))
	sb.WriteString("This skill was automatically generated from a CLAUDE.md learning.\n\n")
	sb.WriteString("## Context\n\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", learning))
	sb.WriteString("## Application\n\n")
	sb.WriteString("Apply this pattern when working in similar contexts.\n")
	return sb.String()
}

// optimizePromoteSkills promotes high-utility skills to CLAUDE.md (TASK-3).
func optimizePromoteSkills(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Skip if no skills directory configured (dry-run mode)
	if opts.SkillsDir == "" {
		return nil
	}

	// Skip if no SkillCompiler available
	if opts.SkillCompiler == nil {
		return nil
	}

	// Query high-utility, high-confidence skills
	query := `
		SELECT id, slug, theme, content, source_memory_ids, alpha, beta, utility, retrieval_count
		FROM generated_skills
		WHERE pruned = 0
		  AND claude_md_promoted = 0
		  AND utility >= ?
		  AND (alpha / (alpha + beta)) >= ?
	`
	rows, err := db.Query(query, opts.MinSkillUtility, opts.MinSkillConfidence)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	type candidate struct {
		id              int64
		slug            string
		theme           string
		content         string
		sourceMemoryIDs string
		alpha           float64
		beta            float64
		utility         float64
		retrievalCount  int
	}

	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.slug, &c.theme, &c.content, &c.sourceMemoryIDs, &c.alpha, &c.beta, &c.utility, &c.retrievalCount); err != nil {
			continue
		}
		candidates = append(candidates, c)
	}

	// Filter by project count
	var filteredCandidates []candidate
	for _, c := range candidates {
		// Count unique projects in usage history
		var projectCount int
		err := db.QueryRow(`
			SELECT COUNT(DISTINCT project)
			FROM skill_usage
			WHERE skill_id = ?
		`, c.id).Scan(&projectCount)
		if err != nil || projectCount < opts.MinSkillProjects {
			continue
		}
		filteredCandidates = append(filteredCandidates, c)
	}

	// Load existing CLAUDE.md Promoted Learnings for semantic dedup
	claudeContent, err := os.ReadFile(opts.ClaudeMDPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	// Parse CLAUDE.md to get existing promoted learnings
	sections := ParseCLAUDEMD(string(claudeContent))
	existingLearnings := sections["Promoted Learnings"]

	// Initialize ONNX for semantic deduplication
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	modelDir := filepath.Join(homeDir, ".claude", "models")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}
	if err := initializeONNXRuntime(modelDir); err != nil {
		return fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}
	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		if err := downloadModel(modelPath); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
	}

	// Generate embeddings for existing learnings
	type learningEntry struct {
		content   string
		embedding []float32
	}
	var existingEmbeddings []learningEntry
	for _, line := range existingLearnings {
		trimmed := strings.TrimSpace(line)
		learning := strings.TrimPrefix(trimmed, "- ")
		if learning == "" {
			continue
		}
		emb, _, _, err := generateEmbeddingONNX("passage: "+learning, modelPath)
		if err != nil {
			continue
		}
		existingEmbeddings = append(existingEmbeddings, learningEntry{content: learning, embedding: emb})
	}

	// Process each candidate
	for _, c := range filteredCandidates {
		// Parse source_memory_ids to extract memories
		var memoryIDs []int64
		if err := json.Unmarshal([]byte(c.sourceMemoryIDs), &memoryIDs); err != nil {
			continue
		}

		// Fetch memory content from DB
		var memories []string
		for _, memID := range memoryIDs {
			var content string
			err := db.QueryRow("SELECT content FROM embeddings WHERE id = ?", memID).Scan(&content)
			if err == nil {
				memories = append(memories, content)
			}
		}

		// Synthesize principle via LLM
		ctx := opts.Context
		if ctx == nil {
			ctx = context.Background()
		}
		principle, err := opts.SkillCompiler.Synthesize(ctx, memories)
		if err != nil {
			if errors.Is(err, ErrLLMUnavailable) {
				fmt.Fprintf(os.Stderr, "LLM unavailable for skill synthesis, skipping promotion candidate\n")
			}
			continue
		}

		// Semantic deduplication: check similarity with existing CLAUDE.md entries
		principleEmb, _, _, err := generateEmbeddingONNX("passage: "+principle, modelPath)
		if err != nil {
			continue
		}

		isDuplicate := false
		for _, existing := range existingEmbeddings {
			sim := cosineSimilarity(principleEmb, existing.embedding)
			if sim > 0.9 {
				isDuplicate = true
				break
			}
		}

		if isDuplicate {
			continue
		}

		// Ask for approval
		confidence := c.alpha / (c.alpha + c.beta)
		approved := opts.AutoApprove
		if !approved && opts.ReviewFunc != nil {
			var err error
			description := fmt.Sprintf("[utility %.2f, confidence %.2f, %d projects] %s", c.utility, confidence, opts.MinSkillProjects, principle)
			approved, err = opts.ReviewFunc("promote_skill", description)
			if err != nil {
				return err
			}
		}

		if approved {
			// Test the skill before promotion
			var sourceEmbeddings []Embedding
			for _, memID := range memoryIDs {
				var content string
				err := db.QueryRow("SELECT content FROM embeddings WHERE id = ?", memID).Scan(&content)
				if err == nil {
					sourceEmbeddings = append(sourceEmbeddings, Embedding{
						ID:      memID,
						Content: content,
					})
				}
			}

			candidate := SkillCandidate{
				Theme:            c.theme,
				Content:          principle,
				SourceEmbeddings: sourceEmbeddings,
			}
			if err := TestAndCompileSkill(opts, candidate); err != nil {
				// Test failed, skip promotion
				continue
			}

			result.SkillsPromoted++

			// Set promoted flag in DB
			now := time.Now().Format(time.RFC3339)
			_, _ = db.Exec("UPDATE generated_skills SET claude_md_promoted = 1, promoted_at = ? WHERE id = ?", now, c.id)

			// Append to CLAUDE.md
			if err := appendToClaudeMD(opts.ClaudeMDPath, []string{principle}); err != nil {
				return fmt.Errorf("failed to append to CLAUDE.md: %w", err)
			}

			// Add to existing embeddings to prevent subsequent duplicates in this run
			existingEmbeddings = append(existingEmbeddings, learningEntry{content: principle, embedding: principleEmb})
		}
	}

	return nil
}

// optimizeMergeSkills detects and merges skills with similar centroids (>0.85 similarity).
func optimizeMergeSkills(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Get all active skills with embeddings
	rows, err := db.Query(`
		SELECT id, slug, source_memory_ids, alpha, beta, utility, embedding_id
		FROM generated_skills
		WHERE pruned = 0 AND embedding_id IS NOT NULL
		ORDER BY utility DESC
	`)
	if err != nil {
		return err
	}

	type skillEntry struct {
		id              int64
		slug            string
		sourceMemoryIDs string
		alpha           float64
		beta            float64
		utility         float64
		embeddingID     int64
	}

	var skills []skillEntry
	for rows.Next() {
		var s skillEntry
		if err := rows.Scan(&s.id, &s.slug, &s.sourceMemoryIDs, &s.alpha, &s.beta, &s.utility, &s.embeddingID); err != nil {
			_ = rows.Close()
			return err
		}
		skills = append(skills, s)
	}
	_ = rows.Close()

	if len(skills) < 2 {
		return nil
	}

	// Check pairwise similarity
	merged := make(map[int64]bool)
	for i := 0; i < len(skills); i++ {
		if merged[skills[i].id] {
			continue
		}
		for j := i + 1; j < len(skills); j++ {
			if merged[skills[j].id] {
				continue
			}

			sim, err := opts.SimilarityFunc(db, skills[i].embeddingID, skills[j].embeddingID)
			if err != nil {
				continue
			}

			if sim > 0.85 {
				// Merge: keep higher utility skill, soft-delete the other
				var keepSkill, deleteSkill skillEntry
				if skills[i].utility >= skills[j].utility {
					keepSkill, deleteSkill = skills[i], skills[j]
				} else {
					keepSkill, deleteSkill = skills[j], skills[i]
				}

				// Combine source_memory_ids
				var keepIDs, deleteIDs []int64
				_ = json.Unmarshal([]byte(keepSkill.sourceMemoryIDs), &keepIDs)
				_ = json.Unmarshal([]byte(deleteSkill.sourceMemoryIDs), &deleteIDs)
				combinedIDs := append(keepIDs, deleteIDs...)
				combinedJSON, _ := json.Marshal(combinedIDs)

				// Sum alpha/beta
				newAlpha := keepSkill.alpha + deleteSkill.alpha
				newBeta := keepSkill.beta + deleteSkill.beta

				// Record merge
				mergeSourceIDs, _ := json.Marshal([]int64{deleteSkill.id})

				// Update kept skill
				now := time.Now().UTC().Format(time.RFC3339)
				_, _ = db.Exec(`
					UPDATE generated_skills
					SET source_memory_ids = ?, alpha = ?, beta = ?, merge_source_ids = ?, updated_at = ?
					WHERE id = ?
				`, string(combinedJSON), newAlpha, newBeta, string(mergeSourceIDs), now, keepSkill.id)

				// Soft-delete merged skill
				_, _ = db.Exec("UPDATE generated_skills SET pruned = 1 WHERE id = ?", deleteSkill.id)

				// Remove deleted skill file
				skillDir := filepath.Join(opts.SkillsDir, "mem-"+deleteSkill.slug)
				_ = os.RemoveAll(skillDir)

				merged[deleteSkill.id] = true
				result.SkillsMerged++
			}
		}
	}

	return nil
}

// optimizeSplitSkills detects and splits incoherent skills by re-clustering source memories.
func optimizeSplitSkills(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	// Get all active skills
	rows, err := db.Query(`
		SELECT id, slug, source_memory_ids
		FROM generated_skills
		WHERE pruned = 0
	`)
	if err != nil {
		return err
	}

	type skillEntry struct {
		id              int64
		slug            string
		sourceMemoryIDs string
	}

	var skills []skillEntry
	for rows.Next() {
		var s skillEntry
		if err := rows.Scan(&s.id, &s.slug, &s.sourceMemoryIDs); err != nil {
			_ = rows.Close()
			return err
		}
		skills = append(skills, s)
	}
	_ = rows.Close()

	minCluster := opts.MinClusterSize
	if minCluster == 0 {
		minCluster = 3
	}

	for _, skill := range skills {
		// Check context inside loop
		if err := checkContext(opts); err != nil {
			return err
		}

		var memoryIDs []int64
		if err := json.Unmarshal([]byte(skill.sourceMemoryIDs), &memoryIDs); err != nil {
			continue
		}

		if len(memoryIDs) < minCluster*2 {
			continue // Too small to split
		}

		// Fetch memories
		var entries []ClusterEntry
		for _, memID := range memoryIDs {
			var content string
			var embeddingID int64
			err := db.QueryRow("SELECT content, embedding_id FROM embeddings WHERE id = ?", memID).Scan(&content, &embeddingID)
			if err != nil {
				continue
			}
			entries = append(entries, ClusterEntry{ID: memID, Content: content, EmbeddingID: embeddingID})
		}

		if len(entries) < minCluster*2 {
			continue
		}

		// Re-cluster at lower threshold (0.6)
		subclusters := clusterBySimilarityWithFunc(db, entries, 0.6, opts.SimilarityFunc)

		// Filter subclusters by min size
		var validSubclusters [][]ClusterEntry
		for _, cluster := range subclusters {
			if len(cluster) >= minCluster {
				validSubclusters = append(validSubclusters, cluster)
			}
		}

		if len(validSubclusters) < 2 {
			continue // Not incoherent enough to split
		}

		// Split: create new skills from subclusters
		for _, subcluster := range validSubclusters {
			theme := generateThemeFromCluster(subcluster)
			slug := slugify(theme)

			ctx := opts.Context
			if ctx == nil {
				ctx = context.Background()
			}
			content, err := generateSkillContent(ctx, theme, subcluster, opts.SkillCompiler)
			if err != nil {
				continue
			}

			description := ExtractSkillDescription(content, 1500)
			sourceIDs := formatClusterSourceIDs(subcluster)

			now := time.Now().UTC().Format(time.RFC3339)
			newSkill := &GeneratedSkill{
				Slug:            slug,
				Theme:           theme,
				Description:     description,
				Content:         content,
				SourceMemoryIDs: sourceIDs,
				Alpha:           1.0,
				Beta:            1.0,
				Utility:         0.5,
				CreatedAt:       now,
				UpdatedAt:       now,
				SplitFromID:     skill.id,
			}

			_, err = insertSkill(db, newSkill)
			if err != nil {
				continue
			}

			_ = writeSkillFile(opts.SkillsDir, newSkill)
		}

		// Soft-delete original skill
		_, _ = db.Exec("UPDATE generated_skills SET pruned = 1 WHERE id = ?", skill.id)

		// Remove original skill file
		skillDir := filepath.Join(opts.SkillsDir, "mem-"+skill.slug)
		_ = os.RemoveAll(skillDir)

		result.SkillsSplit++
	}

	return nil
}
