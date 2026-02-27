package memory

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

// QuadrantSummary holds per-quadrant memory counts and current threshold values.
type QuadrantSummary struct {
	WorkingCount        int
	LeechCount          int
	GemCount            int
	NoiseCount          int
	AlphaWeight         float64
	ImportanceThreshold float64
	ImpactThreshold     float64
}

// AutoTuneThresholds adjusts the impact_threshold using EMA-style step adjustments
// based on the current leech percentage. Target leech% is 5-15%.
func AutoTuneThresholds(db *sql.DB) error {
	// Count memories in each quadrant
	var total, leechCount int

	err := db.QueryRow(`
		SELECT COUNT(*) FROM embeddings
		WHERE quadrant IN ('working', 'leech', 'gem', 'noise')
		AND (importance_score > 0 OR impact_score > 0 OR quadrant != 'noise')`).Scan(&total)
	if err != nil {
		return fmt.Errorf("AutoTuneThresholds: count total: %w", err)
	}

	if total == 0 {
		return nil
	}

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE quadrant = 'leech'").Scan(&leechCount)
	if err != nil {
		return fmt.Errorf("AutoTuneThresholds: count leeches: %w", err)
	}

	leechPct := float64(leechCount) / float64(total) * 100.0

	// Read current threshold
	threshStr, _ := getMetadata(db, "impact_threshold")
	threshold := 0.3

	if threshStr != "" {
		if v, err := strconv.ParseFloat(threshStr, 64); err == nil {
			threshold = v
		}
	}

	// Adjust: step ±0.05
	const step = 0.05

	var adjustment float64
	if leechPct > 15.0 {
		adjustment = step // Raise threshold → fewer classified as leech
	} else if leechPct < 5.0 {
		adjustment = -step // Lower threshold → more classified as leech
	}
	// In range [5%, 15%]: no adjustment

	if adjustment != 0 {
		oldThreshold := threshold
		threshold += adjustment
		// Clamp to [0.0, 1.0]
		if threshold > 1.0 {
			threshold = 1.0
		}

		if threshold < 0.0 {
			threshold = 0.0
		}

		reason := "above 15% target"
		if leechPct < 5.0 {
			reason = "below 5% target"
		}

		fmt.Fprintf(os.Stderr, "[memory:autotune] threshold adjusted: %.2f → %.2f (leech%%=%.1f%%, reason: %s)\n",
			oldThreshold, threshold, leechPct, reason)

		err := setMetadata(db, "impact_threshold", fmt.Sprintf("%g", threshold))
		if err != nil {
			return fmt.Errorf("AutoTuneThresholds: set threshold: %w", err)
		}

		err = setMetadata(db, "last_autotune_at", time.Now().Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("AutoTuneThresholds: set timestamp: %w", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "[memory:autotune] threshold unchanged at %.2f (leech%%=%.1f%%, within 5-15%% target)\n",
			threshold, leechPct)
	}

	return nil
}

// ClassifyQuadrants reads importance_score and impact_score for all memories
// and assigns each a quadrant based on metadata thresholds.
func ClassifyQuadrants(db *sql.DB) error {
	importThreshStr, _ := getMetadata(db, "importance_threshold")
	impactThreshStr, _ := getMetadata(db, "impact_threshold")

	importThresh := 0.0
	impactThresh := 0.3

	if importThreshStr != "" {
		if v, err := strconv.ParseFloat(importThreshStr, 64); err == nil {
			importThresh = v
		}
	}

	if impactThreshStr != "" {
		if v, err := strconv.ParseFloat(impactThreshStr, 64); err == nil {
			impactThresh = v
		}
	}

	// Classify each memory with any scoring data
	rows, err := db.Query(`
		SELECT id, importance_score, impact_score, COALESCE(quadrant, '') FROM embeddings
		WHERE importance_score > 0 OR impact_score > 0`)
	if err != nil {
		return fmt.Errorf("ClassifyQuadrants: query: %w", err)
	}

	defer func() { _ = rows.Close() }()

	type memScore struct {
		id         int64
		importance float64
		impact     float64
		oldQuad    string
	}

	var mems []memScore

	for rows.Next() {
		var m memScore

		err := rows.Scan(&m.id, &m.importance, &m.impact, &m.oldQuad)
		if err != nil {
			continue
		}

		mems = append(mems, m)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("ClassifyQuadrants: scan: %w", err)
	}

	reclassified := 0

	for _, m := range mems {
		highImportance := m.importance > importThresh
		highImpact := m.impact > impactThresh

		var quadrant string

		switch {
		case highImportance && highImpact:
			quadrant = "working"
		case highImportance && !highImpact:
			quadrant = "leech"
		case !highImportance && highImpact:
			quadrant = "gem"
		default:
			quadrant = "noise"
		}

		if quadrant != m.oldQuad {
			reclassified++

			fmt.Fprintf(os.Stderr, "[memory:quadrant] memory %d: %s → %s\n", m.id, m.oldQuad, quadrant)
		}

		_, err := db.Exec("UPDATE embeddings SET quadrant = ? WHERE id = ?", quadrant, m.id)
		if err != nil {
			return fmt.Errorf("ClassifyQuadrants: update %d: %w", m.id, err)
		}
	}

	fmt.Fprintf(os.Stderr, "[memory:quadrant] classified %d memories, %d reclassified\n", len(mems), reclassified)

	return nil
}

// GetQuadrantSummary returns the count of memories in each quadrant
// along with the current threshold values from the metadata table.
func GetQuadrantSummary(db *sql.DB) (QuadrantSummary, error) {
	var summary QuadrantSummary

	// Count per quadrant
	rows, err := db.Query("SELECT quadrant, COUNT(*) FROM embeddings GROUP BY quadrant")
	if err != nil {
		return summary, fmt.Errorf("GetQuadrantSummary: query: %w", err)
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			quadrant string
			count    int
		)

		err := rows.Scan(&quadrant, &count)
		if err != nil {
			continue
		}

		switch quadrant {
		case "working":
			summary.WorkingCount = count
		case "leech":
			summary.LeechCount = count
		case "gem":
			summary.GemCount = count
		case "noise":
			summary.NoiseCount = count
		}
	}

	if err := rows.Err(); err != nil {
		return summary, fmt.Errorf("GetQuadrantSummary: scan: %w", err)
	}

	// Read thresholds
	alphaStr, _ := getMetadata(db, "alpha_weight")
	importStr, _ := getMetadata(db, "importance_threshold")
	impactStr, _ := getMetadata(db, "impact_threshold")

	if alphaStr != "" {
		summary.AlphaWeight, _ = strconv.ParseFloat(alphaStr, 64)
	}

	if importStr != "" {
		summary.ImportanceThreshold, _ = strconv.ParseFloat(importStr, 64)
	}

	if impactStr != "" {
		summary.ImpactThreshold, _ = strconv.ParseFloat(impactStr, 64)
	}

	return summary, nil
}

// ScoreSession evaluates all haiku_relevant=true surfacing events in the session
// with faithfulness=null, calling the LLM for post-evaluation and updating
// faithfulness, outcome_signal, impact_score, and leech_count per memory.
func ScoreSession(db *sql.DB, sessionID string, llm LLMExtractor) error {
	events, err := GetSessionSurfacingEvents(db, sessionID)
	if err != nil {
		return fmt.Errorf("ScoreSession: %w", err)
	}

	// Collect events needing evaluation: haiku_relevant=true AND faithfulness=null
	var toEval []SurfacingEvent

	for _, e := range events {
		if e.HaikuRelevant != nil && *e.HaikuRelevant && e.Faithfulness == nil {
			toEval = append(toEval, e)
		}
	}

	if len(toEval) == 0 {
		fmt.Fprintf(os.Stderr, "[memory:scoring] session %s: no events to evaluate\n", sessionID)
		return nil
	}

	fmt.Fprintf(os.Stderr, "[memory:scoring] session %s: evaluating %d events\n", sessionID, len(toEval))

	// Read alpha_weight for effectiveness calculation
	alphaStr, _ := getMetadata(db, "alpha_weight")
	alphaWeight := 0.5

	if alphaStr != "" {
		if v, parseErr := strconv.ParseFloat(alphaStr, 64); parseErr == nil {
			alphaWeight = v
		}
	}

	ctx := context.Background()

	// Track which memories were affected for score recalculation
	affectedMemories := make(map[int64]bool)

	for _, e := range toEval {
		// Look up memory content for post-eval prompt
		var memContent string

		err := db.QueryRow("SELECT content FROM embeddings WHERE id = ?", e.MemoryID).Scan(&memContent)
		if err != nil {
			continue // Skip if memory not found
		}

		result, err := llm.PostEval(ctx, memContent, e.QueryText)
		if err != nil {
			continue // Skip on LLM failure per contract
		}

		if err := UpdateSurfacingOutcome(db, e.ID, result.Faithfulness, result.Signal); err != nil {
			continue
		}

		affectedMemories[e.MemoryID] = true
	}

	// Recalculate impact_score, effectiveness, and leech_count for each affected memory
	fmt.Fprintf(os.Stderr, "[memory:scoring] session %s: %d memories affected, recalculating scores\n", sessionID, len(affectedMemories))

	for memID := range affectedMemories {
		err := recalculateMemoryScores(db, memID, alphaWeight)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[memory:scoring] warning: failed to recalculate scores for memory %d: %v\n", memID, err)
		}
	}

	// Update quadrant classifications and tune thresholds
	if err := ClassifyQuadrants(db); err != nil {
		return fmt.Errorf("ScoreSession: classify: %w", err)
	}

	if err := AutoTuneThresholds(db); err != nil {
		return fmt.Errorf("ScoreSession: autotune: %w", err)
	}

	return nil
}

// UpdateSurfacingOutcome updates the faithfulness and outcome_signal columns
// for the surfacing event identified by eventID.
func UpdateSurfacingOutcome(db *sql.DB, eventID int64, faithfulness float64, signal string) error {
	_, err := db.Exec(
		"UPDATE surfacing_events SET faithfulness = ?, outcome_signal = ? WHERE id = ?",
		faithfulness, signal, eventID)
	if err != nil {
		return fmt.Errorf("UpdateSurfacingOutcome: %w", err)
	}

	return nil
}

// recalculateMemoryScores updates impact_score, effectiveness, and leech_count
// for a single memory based on all its surfacing events.
func recalculateMemoryScores(db *sql.DB, memoryID int64, alphaWeight float64) error {
	// Get all surfacing events with faithfulness scores for this memory
	rows, err := db.Query(`
		SELECT faithfulness, timestamp FROM surfacing_events
		WHERE memory_id = ? AND faithfulness IS NOT NULL
		ORDER BY timestamp ASC`, memoryID)
	if err != nil {
		return err
	}

	defer func() { _ = rows.Close() }()

	type scored struct {
		faithfulness float64
	}

	var events []scored

	for rows.Next() {
		var (
			s     scored
			tsStr string
		)

		err := rows.Scan(&s.faithfulness, &tsStr)
		if err != nil {
			continue
		}

		events = append(events, s)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	// Recency-weighted average: more recent events get higher weight
	// Weight = 0.9^(age_in_events), where newest event has weight 1.0
	var weightedSum, totalWeight float64

	for i, e := range events {
		weight := math.Pow(0.9, float64(len(events)-1-i))
		weightedSum += e.faithfulness * weight
		totalWeight += weight
	}

	impactScore := weightedSum / totalWeight

	// Get current importance_score
	var importanceScore float64

	err = db.QueryRow("SELECT importance_score FROM embeddings WHERE id = ?", memoryID).Scan(&importanceScore)
	if err != nil {
		return err
	}

	// effectiveness = importance_score + α × impact_score
	effectiveness := importanceScore + alphaWeight*impactScore

	// Leech count: check the most recent faithfulness
	lastFaithfulness := events[len(events)-1].faithfulness

	var leechCountUpdate string
	if lastFaithfulness < 0.3 {
		leechCountUpdate = "leech_count + 1"
	} else {
		leechCountUpdate = "0"
	}

	_, err = db.Exec(fmt.Sprintf(`
		UPDATE embeddings
		SET impact_score = ?, effectiveness = ?, leech_count = %s
		WHERE id = ?`, leechCountUpdate),
		impactScore, effectiveness, memoryID)

	return err
}
