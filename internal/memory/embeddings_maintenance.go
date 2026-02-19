package memory

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AutoMaintenance runs automatic prune and decay operations.
// Prune: Delete embeddings with confidence < 0.3
// Decay: Multiply confidence by 0.5 for entries >90 days old with <5 retrievals
// Returns counts of pruned and decayed entries.
func AutoMaintenance(db *sql.DB) (pruned int, decayed int, err error) {
	// Prune: confidence < 0.3
	res, err := db.Exec(`DELETE FROM embeddings WHERE confidence < 0.3`)
	if err != nil {
		return 0, 0, fmt.Errorf("auto-prune: %w", err)
	}
	rows, _ := res.RowsAffected()
	pruned = int(rows)

	// Decay: >90 days old, <5 retrievals → confidence × 0.5
	res, err = db.Exec(`UPDATE embeddings SET confidence = confidence * 0.5
		WHERE julianday('now') - julianday(created_at) > 90
		AND retrieval_count < 5
		AND confidence >= 0.3`)
	if err != nil {
		return pruned, 0, fmt.Errorf("auto-decay: %w", err)
	}
	rows, _ = res.RowsAffected()
	decayed = int(rows)

	return pruned, decayed, nil
}

// scanEmbeddings scans the embeddings database tier and returns maintenance proposals.
// It detects:
// - Redundant embeddings (similarity > 0.92) → consolidate
// - Multi-topic embeddings (token count > threshold) → split
// - High-value embeddings (retrieval 10+, confidence 0.8+, multi-project) → promote to skill
// Note: Low-confidence and stale embeddings are now handled automatically by AutoMaintenance.
func scanEmbeddings(db *sql.DB, memoryRoot, skillsDir string) ([]MaintenanceProposal, error) {
	var proposals []MaintenanceProposal

	// Scan for redundant entries (similarity > 0.92)
	redundantProposals, err := scanRedundantEmbeddings(db)
	if err != nil {
		return nil, fmt.Errorf("failed to scan redundant embeddings: %w", err)
	}
	proposals = append(proposals, redundantProposals...)

	// Scan for high-value entries (retrieval 10+, confidence 0.8+, multi-project) → promote to skill
	if skillsDir != "" {
		promoteProposals, err := scanPromotableEmbeddings(db)
		if err != nil {
			return nil, fmt.Errorf("failed to scan promotable embeddings: %w", err)
		}
		proposals = append(proposals, promoteProposals...)
	}

	return proposals, nil
}

// scanLowConfidenceEmbeddings detects embeddings with confidence < 0.3 that should be pruned.
func scanLowConfidenceEmbeddings(db *sql.DB) ([]MaintenanceProposal, error) {
	rows, err := db.Query(`
		SELECT id, content, confidence
		FROM embeddings
		WHERE confidence < 0.3 AND promoted = 0
		ORDER BY confidence ASC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var proposals []MaintenanceProposal
	for rows.Next() {
		var id int64
		var content string
		var confidence float64
		if err := rows.Scan(&id, &content, &confidence); err != nil {
			continue
		}

		proposals = append(proposals, MaintenanceProposal{
			Tier:    "embeddings",
			Action:  "prune",
			Target:  fmt.Sprintf("%d", id),
			Reason:  fmt.Sprintf("Low confidence (%.2f)", confidence),
			Preview: content,
		})
	}

	return proposals, nil
}

// scanStaleEmbeddings detects embeddings that are >90 days old with low retrieval counts.
func scanStaleEmbeddings(db *sql.DB) ([]MaintenanceProposal, error) {
	rows, err := db.Query(`
		SELECT id, content, confidence, last_retrieved, retrieval_count
		FROM embeddings
		WHERE promoted = 0
		  AND retrieval_count < 5
		  AND last_retrieved IS NOT NULL
		  AND last_retrieved != ''
		ORDER BY last_retrieved ASC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var proposals []MaintenanceProposal
	now := time.Now()
	threshold := 90 * 24 * time.Hour

	for rows.Next() {
		var id int64
		var content string
		var confidence float64
		var lastRetrievedStr string
		var retrievalCount int
		if err := rows.Scan(&id, &content, &confidence, &lastRetrievedStr, &retrievalCount); err != nil {
			continue
		}

		// Parse last_retrieved timestamp
		lastRetrieved, err := time.Parse(time.RFC3339, lastRetrievedStr)
		if err != nil {
			continue
		}

		// Check if stale (>90 days old)
		age := now.Sub(lastRetrieved)
		if age > threshold {
			// Calculate decay factor (reduce confidence by 50%)
			newConfidence := confidence * 0.5

			proposals = append(proposals, MaintenanceProposal{
				Tier:    "embeddings",
				Action:  "decay",
				Target:  fmt.Sprintf("%d", id),
				Reason:  fmt.Sprintf("Stale (%d days, %d retrievals)", int(age.Hours()/24), retrievalCount),
				Preview: fmt.Sprintf("Confidence: %.2f → %.2f\n%s", confidence, newConfidence, content),
			})
		}
	}

	return proposals, nil
}

// scanRedundantEmbeddings detects embeddings with high semantic similarity (>0.8) for consolidation.
func scanRedundantEmbeddings(db *sql.DB) ([]MaintenanceProposal, error) {
	// Get all entries with embeddings
	rows, err := db.Query(`
		SELECT e.id, e.content, e.embedding_id, e.confidence, e.retrieval_count
		FROM embeddings e
		WHERE e.embedding_id IS NOT NULL
		  AND e.promoted = 0
		ORDER BY e.id
	`)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		entries = append(entries, e)
	}
	_ = rows.Close()

	var proposals []MaintenanceProposal
	checked := make(map[int64]bool)

	// Compare pairs for similarity
	for i := 0; i < len(entries); i++ {
		if checked[entries[i].id] {
			continue
		}
		for j := i + 1; j < len(entries); j++ {
			if checked[entries[j].id] {
				continue
			}

			// Calculate similarity
			sim, err := calculateSimilarity(db, entries[i].embeddingID, entries[j].embeddingID)
			if err != nil {
				continue
			}

			if sim > 0.92 {
				// Identify which to keep (higher confidence/retrieval count)
				var keepEntry, deleteEntry entry
				if entries[i].confidence > entries[j].confidence ||
					(entries[i].confidence == entries[j].confidence && entries[i].retrievalCount >= entries[j].retrievalCount) {
					keepEntry, deleteEntry = entries[i], entries[j]
				} else {
					keepEntry, deleteEntry = entries[j], entries[i]
				}

				proposals = append(proposals, MaintenanceProposal{
					Tier:    "embeddings",
					Action:  "consolidate",
					Target:  fmt.Sprintf("%d,%d", keepEntry.id, deleteEntry.id),
					Reason:  fmt.Sprintf("Redundant (similarity %.2f)", sim),
					Preview: fmt.Sprintf("Keep:   %s\nDelete: %s", keepEntry.content, deleteEntry.content),
				})

				checked[deleteEntry.id] = true
			}
		}
		checked[entries[i].id] = true
	}

	return proposals, nil
}

// scanPromotableEmbeddings detects high-value embeddings that should be promoted to skills.
func scanPromotableEmbeddings(db *sql.DB) ([]MaintenanceProposal, error) {
	rows, err := db.Query(`
		SELECT id, content, confidence, retrieval_count, projects_retrieved, principle
		FROM embeddings
		WHERE retrieval_count >= 10
		  AND confidence >= 0.8
		  AND promoted = 0
		  AND principle != ''
		ORDER BY retrieval_count DESC, confidence DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var proposals []MaintenanceProposal
	for rows.Next() {
		var id int64
		var content, projectsStr, principle string
		var confidence float64
		var retrievalCount int
		if err := rows.Scan(&id, &content, &confidence, &retrievalCount, &projectsStr, &principle); err != nil {
			continue
		}

		// Count unique projects
		projectCount := 0
		if projectsStr != "" {
			projectMap := make(map[string]bool)
			for _, p := range strings.Split(projectsStr, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					projectMap[p] = true
				}
			}
			projectCount = len(projectMap)
		}

		// Require at least 3 projects for promotion
		if projectCount >= 3 {
			// Generate slug from principle
			slug := slugify(principle)

			proposals = append(proposals, MaintenanceProposal{
				Tier:    "embeddings",
				Action:  "promote",
				Target:  fmt.Sprintf("%d", id),
				Reason:  fmt.Sprintf("High retrieval (%dx), confidence %.2f, %d projects", retrievalCount, confidence, projectCount),
				Preview: fmt.Sprintf("Generate skill: %s\nPrinciple: %s", slug, principle),
			})
		}
	}

	return proposals, nil
}

// applyEmbeddingsProposal applies a maintenance proposal to the embeddings tier.
func applyEmbeddingsProposal(db *sql.DB, memoryRoot, skillsDir string, proposal MaintenanceProposal) error {
	switch proposal.Action {
	case "prune":
		return applyPrune(db, proposal)
	case "decay":
		return applyDecay(db, proposal)
	case "consolidate":
		return applyConsolidate(db, proposal)
	case "split":
		return applySplit(db, proposal)
	case "promote":
		return applyPromote(db, skillsDir, proposal)
	case "rewrite":
		// ISSUE-218: Rewrite embedding content
		return applyEmbeddingsRewrite(db, proposal)
	case "add-rationale":
		// ISSUE-218: Add rationale to embedding
		return applyEmbeddingsAddRationale(db, proposal)
	default:
		return fmt.Errorf("unknown action: %s", proposal.Action)
	}
}

// applyPrune deletes an embedding from the database.
func applyPrune(db *sql.DB, proposal MaintenanceProposal) error {
	id, err := strconv.ParseInt(proposal.Target, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid target ID: %w", err)
	}

	// ISSUE-230: Archive before deletion
	_ = ArchiveEmbedding(db, id, "prune", proposal.Reason)

	// Get embedding_id for cleanup
	var embeddingID sql.NullInt64
	err = db.QueryRow("SELECT embedding_id FROM embeddings WHERE id = ?", id).Scan(&embeddingID)
	if err != nil {
		return fmt.Errorf("failed to get embedding_id: %w", err)
	}

	// Delete from embeddings table
	_, err = db.Exec("DELETE FROM embeddings WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete embedding: %w", err)
	}

	// Delete from vec_embeddings table
	if embeddingID.Valid {
		_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", embeddingID.Int64)
	}

	// Delete from FTS5 table
	deleteFTS5(db, id)

	return nil
}

// applyDecay reduces the confidence of a stale embedding.
func applyDecay(db *sql.DB, proposal MaintenanceProposal) error {
	id, err := strconv.ParseInt(proposal.Target, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid target ID: %w", err)
	}

	// Reduce confidence by 50%
	_, err = db.Exec("UPDATE embeddings SET confidence = confidence * 0.5 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to decay confidence: %w", err)
	}

	return nil
}

// applyConsolidate merges redundant embeddings.
func applyConsolidate(db *sql.DB, proposal MaintenanceProposal) error {
	// Parse target IDs (format: "keepID,deleteID")
	parts := strings.Split(proposal.Target, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid consolidate target format: %s", proposal.Target)
	}

	keepID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid keep ID: %w", err)
	}

	deleteID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid delete ID: %w", err)
	}

	// Update keep entry: increment retrieval_count by merging stats
	_, err = db.Exec(`
		UPDATE embeddings
		SET retrieval_count = retrieval_count + (SELECT retrieval_count FROM embeddings WHERE id = ?)
		WHERE id = ?
	`, deleteID, keepID)
	if err != nil {
		return fmt.Errorf("failed to merge retrieval counts: %w", err)
	}

	// ISSUE-230: Archive before deletion
	_ = ArchiveEmbedding(db, deleteID, "consolidate", proposal.Reason)

	// Delete redundant entry
	var embeddingID sql.NullInt64
	_ = db.QueryRow("SELECT embedding_id FROM embeddings WHERE id = ?", deleteID).Scan(&embeddingID)
	_, err = db.Exec("DELETE FROM embeddings WHERE id = ?", deleteID)
	if err != nil {
		return fmt.Errorf("failed to delete redundant embedding: %w", err)
	}

	// Cleanup vec_embeddings and FTS5
	if embeddingID.Valid {
		_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", embeddingID.Int64)
	}
	deleteFTS5(db, deleteID)

	return nil
}

// applySplit splits a multi-topic embedding into multiple entries.
func applySplit(db *sql.DB, proposal MaintenanceProposal) error {
	// TODO: Implement splitting logic (requires topic detection)
	// For now, return not implemented
	return fmt.Errorf("split action not yet implemented")
}

// applyPromote promotes a high-value embedding to a skill.
func applyPromote(db *sql.DB, skillsDir string, proposal MaintenanceProposal) error {
	if skillsDir == "" {
		return fmt.Errorf("skills directory not configured")
	}

	id, err := strconv.ParseInt(proposal.Target, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid target ID: %w", err)
	}

	// Fetch embedding data
	var content, principle string
	var confidence float64
	var retrievalCount int
	err = db.QueryRow(`
		SELECT content, principle, confidence, retrieval_count
		FROM embeddings
		WHERE id = ?
	`, id).Scan(&content, &principle, &confidence, &retrievalCount)
	if err != nil {
		return fmt.Errorf("failed to fetch embedding: %w", err)
	}

	// Generate skill from embedding
	slug := slugify(principle)
	theme := principle
	skillContent := generateSkillTemplate(theme, content)
	description := generateTriggerDescription(theme, skillContent)

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            slug,
		Theme:           theme,
		Description:     description,
		Content:         skillContent,
		SourceMemoryIDs: fmt.Sprintf("[%d]", id),
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         confidence,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Validate compliance before writing
	compliance := ValidateSkillCompliance(skill)
	if len(compliance.Issues) > 0 {
		return fmt.Errorf("skill %q failed compliance validation: %v", slug, compliance.Issues)
	}

	// Check if skill already exists
	existing, err := getSkillBySlug(db, slug)
	if err == nil && existing != nil && !existing.Pruned {
		// Update existing skill
		existing.Content = skillContent
		existing.Description = description
		existing.SourceMemoryIDs = fmt.Sprintf("[%d]", id)
		existing.UpdatedAt = now
		if err := updateSkill(db, existing); err != nil {
			return fmt.Errorf("failed to update skill: %w", err)
		}
		return writeSkillFile(skillsDir, existing)
	}

	// Insert new skill
	_, err = insertSkill(db, skill)
	if err != nil {
		return fmt.Errorf("failed to insert skill: %w", err)
	}

	return writeSkillFile(skillsDir, skill)
}

// applyEmbeddingsRewrite rewrites an embedding's content.
func applyEmbeddingsRewrite(db *sql.DB, proposal MaintenanceProposal) error {
	id, err := strconv.ParseInt(proposal.Target, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid target ID: %w", err)
	}

	// Update content with refined version
	_, err = db.Exec("UPDATE embeddings SET content = ? WHERE id = ?", proposal.Preview, id)
	if err != nil {
		return fmt.Errorf("failed to update content: %w", err)
	}

	// Clear flagged_for_rewrite flag
	_, _ = db.Exec("UPDATE embeddings SET flagged_for_rewrite = 0 WHERE id = ?", id)

	return nil
}

// applyEmbeddingsAddRationale adds rationale to an embedding.
func applyEmbeddingsAddRationale(db *sql.DB, proposal MaintenanceProposal) error {
	id, err := strconv.ParseInt(proposal.Target, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid target ID: %w", err)
	}

	// Extract rationale from preview (format: "principle - rationale")
	// The preview contains the enriched version with rationale
	// We need to extract just the rationale part
	rationale := extractRationaleFromEnriched(proposal.Preview)

	// Update rationale field
	_, err = db.Exec("UPDATE embeddings SET rationale = ? WHERE id = ?", rationale, id)
	if err != nil {
		return fmt.Errorf("failed to update rationale: %w", err)
	}

	return nil
}

// extractRationaleFromEnriched extracts the rationale from an enriched entry.
// Format: "principle - rationale" or "principle because rationale"
func extractRationaleFromEnriched(enriched string) string {
	// Try to split on " - "
	if idx := strings.Index(enriched, " - "); idx != -1 {
		return strings.TrimSpace(enriched[idx+3:])
	}

	// Try to split on " because "
	if idx := strings.Index(enriched, " because "); idx != -1 {
		return strings.TrimSpace(enriched[idx+9:])
	}

	// Otherwise return the whole enriched string as rationale
	return enriched
}
