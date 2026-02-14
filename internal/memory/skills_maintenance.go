package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SkillsScanner scans the skills tier for maintenance opportunities.
type SkillsScanner struct {
	db   *sql.DB
	opts SkillsScannerOpts
}

// SkillsScannerOpts configures skills tier scanning.
type SkillsScannerOpts struct {
	// Unused detection
	UnusedDaysThreshold int     // Days without retrieval (default 30)
	LowUtilityThreshold float64 // Utility below this is considered low (default 0.4)

	// Redundancy detection
	SimilarityThreshold float64                                             // Similarity above this triggers consolidation (default 0.85)
	SimilarityFunc      func(db *sql.DB, emb1, emb2 int64) (float64, error) // Optional similarity function

	// Promotion criteria
	PromoteUtilityThreshold float64 // Minimum utility for promotion (default 0.7)
	PromoteMinProjects      int     // Minimum project count for promotion (default 3)

	// Demotion criteria
	DemoteMaxProjects int // Maximum projects for demotion (default 1)
}

// NewSkillsScanner creates a new skills tier scanner.
func NewSkillsScanner(db *sql.DB, opts SkillsScannerOpts) *SkillsScanner {
	// Set defaults
	if opts.UnusedDaysThreshold == 0 {
		opts.UnusedDaysThreshold = 30
	}
	if opts.LowUtilityThreshold == 0 {
		opts.LowUtilityThreshold = 0.4
	}
	if opts.SimilarityThreshold == 0 {
		opts.SimilarityThreshold = 0.85
	}
	if opts.SimilarityFunc == nil {
		opts.SimilarityFunc = calculateSimilarity
	}
	if opts.PromoteUtilityThreshold == 0 {
		opts.PromoteUtilityThreshold = 0.7
	}
	if opts.PromoteMinProjects == 0 {
		opts.PromoteMinProjects = 3
	}
	if opts.DemoteMaxProjects == 0 {
		opts.DemoteMaxProjects = 1
	}

	return &SkillsScanner{db: db, opts: opts}
}

// Scan returns maintenance proposals for the skills tier.
func (s *SkillsScanner) Scan() ([]MaintenanceProposal, error) {
	var proposals []MaintenanceProposal

	// Detect unused/stale skills
	unusedProposals, err := s.detectUnusedSkills()
	if err != nil {
		return nil, fmt.Errorf("failed to detect unused skills: %w", err)
	}
	proposals = append(proposals, unusedProposals...)

	// Detect low-utility skills (for decay)
	decayProposals, err := s.detectLowUtilitySkills()
	if err != nil {
		return nil, fmt.Errorf("failed to detect low-utility skills: %w", err)
	}
	proposals = append(proposals, decayProposals...)

	// Detect redundant skills (for consolidation)
	redundantProposals, err := s.detectRedundantSkills()
	if err != nil {
		return nil, fmt.Errorf("failed to detect redundant skills: %w", err)
	}
	proposals = append(proposals, redundantProposals...)

	// Detect high-utility multi-project skills (for promotion)
	promoteProposals, err := s.detectPromotionCandidates()
	if err != nil {
		return nil, fmt.Errorf("failed to detect promotion candidates: %w", err)
	}
	proposals = append(proposals, promoteProposals...)

	// Detect single-project skills (for demotion)
	demoteProposals, err := s.detectDemotionCandidates()
	if err != nil {
		return nil, fmt.Errorf("failed to detect demotion candidates: %w", err)
	}
	proposals = append(proposals, demoteProposals...)

	return proposals, nil
}

// detectUnusedSkills finds skills with no retrievals in N+ days and low utility.
func (s *SkillsScanner) detectUnusedSkills() ([]MaintenanceProposal, error) {
	var proposals []MaintenanceProposal

	// Query skills with no retrievals and old last_retrieved (or NULL)
	query := `
		SELECT id, slug, theme, utility, retrieval_count, created_at
		FROM generated_skills
		WHERE pruned = 0
		  AND retrieval_count = 0
		  AND utility < ?
	`
	rows, err := s.db.Query(query, s.opts.LowUtilityThreshold)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	now := time.Now()
	threshold := time.Duration(s.opts.UnusedDaysThreshold) * 24 * time.Hour

	for rows.Next() {
		var id int64
		var slug, theme, createdAt string
		var utility float64
		var retrievalCount int

		if err := rows.Scan(&id, &slug, &theme, &utility, &retrievalCount, &createdAt); err != nil {
			continue
		}

		// Parse created_at to check age
		created, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			continue
		}

		daysSinceCreation := int(now.Sub(created).Hours() / 24)
		if now.Sub(created) > threshold {
			proposals = append(proposals, MaintenanceProposal{
				Tier:    "skills",
				Action:  "prune",
				Target:  slug,
				Reason:  fmt.Sprintf("Unused: no retrievals in %d days, utility %.2f", daysSinceCreation, utility),
				Preview: fmt.Sprintf("Remove skill %q (%s)", slug, theme),
			})
		}
	}

	return proposals, rows.Err()
}

// detectLowUtilitySkills finds skills with low utility (but some usage).
func (s *SkillsScanner) detectLowUtilitySkills() ([]MaintenanceProposal, error) {
	var proposals []MaintenanceProposal

	// Query skills with retrieval_count > 0 but low utility
	query := `
		SELECT slug, theme, utility, retrieval_count
		FROM generated_skills
		WHERE pruned = 0
		  AND retrieval_count > 0
		  AND utility < ?
	`
	rows, err := s.db.Query(query, s.opts.LowUtilityThreshold)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var slug, theme string
		var utility float64
		var retrievalCount int

		if err := rows.Scan(&slug, &theme, &utility, &retrievalCount); err != nil {
			continue
		}

		proposals = append(proposals, MaintenanceProposal{
			Tier:    "skills",
			Action:  "decay",
			Target:  slug,
			Reason:  fmt.Sprintf("Low utility %.2f despite %d retrievals", utility, retrievalCount),
			Preview: fmt.Sprintf("Decrease confidence for %q (%s)", slug, theme),
		})
	}

	return proposals, rows.Err()
}

// detectRedundantSkills finds skills with high semantic similarity.
func (s *SkillsScanner) detectRedundantSkills() ([]MaintenanceProposal, error) {
	var proposals []MaintenanceProposal

	// Query skills with embeddings
	query := `
		SELECT id, slug, theme, embedding_id, utility
		FROM generated_skills
		WHERE pruned = 0
		  AND embedding_id IS NOT NULL
		ORDER BY utility DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	type skillEntry struct {
		id          int64
		slug        string
		theme       string
		embeddingID int64
		utility     float64
	}

	var skills []skillEntry
	for rows.Next() {
		var s skillEntry
		if err := rows.Scan(&s.id, &s.slug, &s.theme, &s.embeddingID, &s.utility); err != nil {
			_ = rows.Close()
			return nil, err
		}
		skills = append(skills, s)
	}
	_ = rows.Close()

	if len(skills) < 2 {
		return proposals, nil
	}

	// Check pairwise similarity
	checked := make(map[int64]bool)
	for i := 0; i < len(skills); i++ {
		if checked[skills[i].id] {
			continue
		}
		for j := i + 1; j < len(skills); j++ {
			if checked[skills[j].id] {
				continue
			}

			sim, err := s.opts.SimilarityFunc(s.db, skills[i].embeddingID, skills[j].embeddingID)
			if err != nil {
				continue
			}

			if sim >= s.opts.SimilarityThreshold {
				// Keep higher utility skill, propose consolidating the other
				keepSkill, mergeSkill := skills[i], skills[j]
				if skills[j].utility > skills[i].utility {
					keepSkill, mergeSkill = skills[j], skills[i]
				}

				proposals = append(proposals, MaintenanceProposal{
					Tier:    "skills",
					Action:  "consolidate",
					Target:  mergeSkill.slug,
					Reason:  fmt.Sprintf("Redundant with %q (similarity %.2f)", keepSkill.slug, sim),
					Preview: fmt.Sprintf("Merge %q into %q", mergeSkill.slug, keepSkill.slug),
				})

				checked[mergeSkill.id] = true
			}
		}
	}

	return proposals, nil
}

// detectPromotionCandidates finds high-utility skills used across multiple projects.
func (s *SkillsScanner) detectPromotionCandidates() ([]MaintenanceProposal, error) {
	var proposals []MaintenanceProposal

	// Query high-utility skills not already promoted
	query := `
		SELECT id, slug, theme, utility, retrieval_count
		FROM generated_skills
		WHERE pruned = 0
		  AND claude_md_promoted = 0
		  AND utility >= ?
	`
	rows, err := s.db.Query(query, s.opts.PromoteUtilityThreshold)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id int64
		var slug, theme string
		var utility float64
		var retrievalCount int

		if err := rows.Scan(&id, &slug, &theme, &utility, &retrievalCount); err != nil {
			continue
		}

		// Count unique projects from skill_usage table
		var projectCount int
		err := s.db.QueryRow(`
			SELECT COUNT(DISTINCT project)
			FROM skill_usage
			WHERE skill_id = ?
		`, id).Scan(&projectCount)

		if err != nil {
			// Table might not exist or no usage recorded
			continue
		}

		if projectCount >= s.opts.PromoteMinProjects {
			proposals = append(proposals, MaintenanceProposal{
				Tier:    "skills",
				Action:  "promote",
				Target:  slug,
				Reason:  fmt.Sprintf("High utility %.2f across %d projects", utility, projectCount),
				Preview: fmt.Sprintf("Add %q to CLAUDE.md Promoted Learnings", theme),
			})
		}
	}

	return proposals, rows.Err()
}

// detectDemotionCandidates finds single-project skills that should be demoted to embeddings.
func (s *SkillsScanner) detectDemotionCandidates() ([]MaintenanceProposal, error) {
	var proposals []MaintenanceProposal

	// Query skills with usage data
	query := `
		SELECT id, slug, theme, utility
		FROM generated_skills
		WHERE pruned = 0
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id int64
		var slug, theme string
		var utility float64

		if err := rows.Scan(&id, &slug, &theme, &utility); err != nil {
			continue
		}

		// Count unique projects
		var projectCount int
		err := s.db.QueryRow(`
			SELECT COUNT(DISTINCT project)
			FROM skill_usage
			WHERE skill_id = ?
		`, id).Scan(&projectCount)

		if err != nil {
			// No usage data, skip
			continue
		}

		if projectCount <= s.opts.DemoteMaxProjects {
			proposals = append(proposals, MaintenanceProposal{
				Tier:    "skills",
				Action:  "demote",
				Target:  slug,
				Reason:  fmt.Sprintf("Narrow scope: only used in %d project(s)", projectCount),
				Preview: fmt.Sprintf("Convert %q to embeddings", slug),
			})
		}
	}

	return proposals, rows.Err()
}

// SkillsApplier applies maintenance proposals to the skills tier.
type SkillsApplier struct {
	db   *sql.DB
	opts SkillsApplierOpts
}

// SkillsApplierOpts configures skills tier proposal application.
type SkillsApplierOpts struct {
	SkillsDir    string // Directory containing skill files
	ClaudeMDPath string // Path to CLAUDE.md for promotions
}

// NewSkillsApplier creates a new skills tier proposal applier.
func NewSkillsApplier(db *sql.DB, opts SkillsApplierOpts) *SkillsApplier {
	return &SkillsApplier{db: db, opts: opts}
}

// Apply executes a maintenance proposal.
func (a *SkillsApplier) Apply(proposal MaintenanceProposal) error {
	if proposal.Tier != "skills" {
		return fmt.Errorf("invalid tier %q for skills applier", proposal.Tier)
	}

	switch proposal.Action {
	case "prune":
		return a.applyPrune(proposal)
	case "decay":
		return a.applyDecay(proposal)
	case "consolidate":
		return a.applyConsolidate(proposal)
	case "split":
		return a.applySplit(proposal)
	case "promote":
		return a.applyPromote(proposal)
	case "demote":
		return a.applyDemote(proposal)
	default:
		return fmt.Errorf("unknown action %q", proposal.Action)
	}
}

// applyPrune soft-deletes a skill in DB and removes its file.
func (a *SkillsApplier) applyPrune(proposal MaintenanceProposal) error {
	// Soft delete in DB
	_, err := a.db.Exec("UPDATE generated_skills SET pruned = 1 WHERE slug = ?", proposal.Target)
	if err != nil {
		return fmt.Errorf("failed to soft-delete skill: %w", err)
	}

	// Remove skill directory if SkillsDir is configured
	if a.opts.SkillsDir != "" {
		skillDir := filepath.Join(a.opts.SkillsDir, "mem-"+proposal.Target)
		if err := os.RemoveAll(skillDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove skill directory: %w", err)
		}
	}

	return nil
}

// applyDecay decreases skill confidence by increasing beta.
func (a *SkillsApplier) applyDecay(proposal MaintenanceProposal) error {
	// Get current skill
	skill, err := getSkillBySlug(a.db, proposal.Target)
	if err != nil {
		return err
	}
	if skill == nil {
		return fmt.Errorf("skill %q not found", proposal.Target)
	}

	// Increase beta (negative feedback) to decrease confidence/utility
	skill.Beta += 2.0

	// Recompute utility
	skill.Utility = computeUtility(skill.Alpha, skill.Beta, skill.RetrievalCount, skill.LastRetrieved)
	skill.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return updateSkill(a.db, skill)
}

// applyConsolidate merges one skill into another.
func (a *SkillsApplier) applyConsolidate(proposal MaintenanceProposal) error {
	// Extract keep-skill slug from Reason field using helper function
	keepSlug := extractKeepSkillSlug(proposal.Reason)

	// Get target skill (the one being merged away)
	targetSkill, err := getSkillBySlug(a.db, proposal.Target)
	if err != nil {
		return fmt.Errorf("failed to get target skill: %w", err)
	}
	if targetSkill == nil {
		return fmt.Errorf("target skill %q not found", proposal.Target)
	}

	// If we found a keep-skill slug, try to merge
	if keepSlug != "" {
		keepSkill, err := getSkillBySlug(a.db, keepSlug)
		if err != nil {
			return fmt.Errorf("failed to get keep skill: %w", err)
		}

		// If keep-skill exists, perform the merge
		if keepSkill != nil {
			// Parse source_memory_ids from both skills
			var targetMemIDs, keepMemIDs []int64
			if err := json.Unmarshal([]byte(targetSkill.SourceMemoryIDs), &targetMemIDs); err != nil {
				return fmt.Errorf("failed to parse target source_memory_ids: %w", err)
			}
			if err := json.Unmarshal([]byte(keepSkill.SourceMemoryIDs), &keepMemIDs); err != nil {
				return fmt.Errorf("failed to parse keep source_memory_ids: %w", err)
			}

			// Combine and deduplicate source_memory_ids using helper function
			mergedMemIDs := deduplicateInt64(append(keepMemIDs, targetMemIDs...))
			mergedMemJSON, err := json.Marshal(mergedMemIDs)
			if err != nil {
				return fmt.Errorf("failed to marshal merged source_memory_ids: %w", err)
			}

			// Parse existing merge_source_ids from DB
			var mergeSourceIDs []int64
			var mergeSourceIDsStr string
			err = a.db.QueryRow("SELECT COALESCE(merge_source_ids, '') FROM generated_skills WHERE id = ?",
				keepSkill.ID).Scan(&mergeSourceIDsStr)
			if err == nil && mergeSourceIDsStr != "" && mergeSourceIDsStr != "[]" {
				_ = json.Unmarshal([]byte(mergeSourceIDsStr), &mergeSourceIDs)
			}

			// Append target skill ID to merge_source_ids
			mergeSourceIDs = append(mergeSourceIDs, targetSkill.ID)
			mergeSourceJSON, err := json.Marshal(mergeSourceIDs)
			if err != nil {
				return fmt.Errorf("failed to marshal merge_source_ids: %w", err)
			}

			// Update keep-skill
			now := time.Now().UTC().Format(time.RFC3339)
			_, err = a.db.Exec(`
				UPDATE generated_skills
				SET source_memory_ids = ?,
				    merge_source_ids = ?,
				    updated_at = ?
				WHERE id = ?
			`, string(mergedMemJSON), string(mergeSourceJSON), now, keepSkill.ID)
			if err != nil {
				return fmt.Errorf("failed to update keep skill: %w", err)
			}
		}
	}

	// Prune the target skill (whether or not merge succeeded)
	_, err = a.db.Exec("UPDATE generated_skills SET pruned = 1 WHERE slug = ?", proposal.Target)
	if err != nil {
		return fmt.Errorf("failed to consolidate skill: %w", err)
	}

	// Remove skill directory
	if a.opts.SkillsDir != "" {
		skillDir := filepath.Join(a.opts.SkillsDir, "mem-"+proposal.Target)
		if err := os.RemoveAll(skillDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove consolidated skill directory: %w", err)
		}
	}

	return nil
}

// applySplit splits an overly broad skill into sub-skills.
func (a *SkillsApplier) applySplit(proposal MaintenanceProposal) error {
	// Get skill by slug
	skill, err := getSkillBySlug(a.db, proposal.Target)
	if err != nil {
		return fmt.Errorf("failed to get skill: %w", err)
	}
	if skill == nil {
		return fmt.Errorf("skill %q not found", proposal.Target)
	}

	// Parse source_memory_ids JSON array
	var sourceMemoryIDs []int64
	if err := json.Unmarshal([]byte(skill.SourceMemoryIDs), &sourceMemoryIDs); err != nil {
		return fmt.Errorf("failed to parse source_memory_ids: %w", err)
	}

	// Check minimum source memories
	if len(sourceMemoryIDs) < 2 {
		return fmt.Errorf("cannot split skill with fewer than 2 source memories")
	}

	// Simple split: divide memories in half
	midpoint := len(sourceMemoryIDs) / 2
	groupA := sourceMemoryIDs[:midpoint]
	groupB := sourceMemoryIDs[midpoint:]

	now := time.Now().UTC().Format(time.RFC3339)

	// Create skill A
	if len(groupA) > 0 {
		groupAJSON, err := json.Marshal(groupA)
		if err != nil {
			return fmt.Errorf("failed to marshal group A IDs: %w", err)
		}

		skillA := &GeneratedSkill{
			Slug:            skill.Slug + "-a",
			Theme:           skill.Theme + " (part A)",
			Description:     skill.Description,
			Content:         skill.Content,
			SourceMemoryIDs: string(groupAJSON),
			SplitFromID:     skill.ID,
			Alpha:           1.0,
			Beta:            1.0,
			Utility:         computeUtility(1.0, 1.0, 0, ""),
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if _, err := insertSkill(a.db, skillA); err != nil {
			return fmt.Errorf("failed to insert skill A: %w", err)
		}
	}

	// Create skill B
	if len(groupB) > 0 {
		groupBJSON, err := json.Marshal(groupB)
		if err != nil {
			return fmt.Errorf("failed to marshal group B IDs: %w", err)
		}

		skillB := &GeneratedSkill{
			Slug:            skill.Slug + "-b",
			Theme:           skill.Theme + " (part B)",
			Description:     skill.Description,
			Content:         skill.Content,
			SourceMemoryIDs: string(groupBJSON),
			SplitFromID:     skill.ID,
			Alpha:           1.0,
			Beta:            1.0,
			Utility:         computeUtility(1.0, 1.0, 0, ""),
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if _, err := insertSkill(a.db, skillB); err != nil {
			return fmt.Errorf("failed to insert skill B: %w", err)
		}
	}

	// Prune original skill
	pruneProposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "prune",
		Target: proposal.Target,
		Reason: "split into sub-skills",
	}

	return a.applyPrune(pruneProposal)
}

// applyPromote adds skill to CLAUDE.md.
func (a *SkillsApplier) applyPromote(proposal MaintenanceProposal) error {
	if a.opts.ClaudeMDPath == "" {
		return fmt.Errorf("ClaudeMDPath not configured")
	}

	// Get skill
	skill, err := getSkillBySlug(a.db, proposal.Target)
	if err != nil {
		return err
	}
	if skill == nil {
		return fmt.Errorf("skill %q not found", proposal.Target)
	}

	// Use preview as the principle text, or fallback to theme
	principle := proposal.Preview
	if principle == "" {
		principle = skill.Theme
	}

	// Append to CLAUDE.md
	if err := appendToClaudeMD(a.opts.ClaudeMDPath, []string{principle}); err != nil {
		return fmt.Errorf("failed to append to CLAUDE.md: %w", err)
	}

	// Mark as promoted in DB
	now := time.Now().Format(time.RFC3339)
	_, err = a.db.Exec("UPDATE generated_skills SET claude_md_promoted = 1, promoted_at = ? WHERE slug = ?",
		now, proposal.Target)
	if err != nil {
		return fmt.Errorf("failed to mark skill as promoted: %w", err)
	}

	return nil
}

// applyDemote converts skill to embedding entries.
func (a *SkillsApplier) applyDemote(proposal MaintenanceProposal) error {
	// Get skill by slug
	skill, err := getSkillBySlug(a.db, proposal.Target)
	if err != nil {
		return fmt.Errorf("failed to get skill: %w", err)
	}
	if skill == nil {
		return fmt.Errorf("skill %q not found", proposal.Target)
	}

	// Parse source_memory_ids JSON array
	var sourceMemoryIDs []int64
	if err := json.Unmarshal([]byte(skill.SourceMemoryIDs), &sourceMemoryIDs); err != nil {
		return fmt.Errorf("failed to parse source_memory_ids: %w", err)
	}

	// Check if source memories still exist
	missingCount := 0
	for _, memID := range sourceMemoryIDs {
		var exists int
		err := a.db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE id = ?", memID).Scan(&exists)
		if err != nil || exists == 0 {
			missingCount++
		}
	}

	// Log warning if some sources are missing
	if missingCount > 0 && missingCount < len(sourceMemoryIDs) {
		fmt.Fprintf(os.Stderr, "Warning: skill %q has %d/%d source memories missing\n",
			proposal.Target, missingCount, len(sourceMemoryIDs))
	}

	// Prune the skill (sources already exist as embeddings, or are gone and can't be recreated)
	return a.applyPrune(proposal)
}

// extractKeepSkillSlug extracts the keep-skill slug from a consolidation reason string.
// Format: 'Redundant with "keep-skill-slug" (similarity 0.XX)'
func extractKeepSkillSlug(reason string) string {
	// Find text between quotes
	startIdx := strings.Index(reason, `"`)
	if startIdx == -1 {
		return ""
	}
	endIdx := strings.Index(reason[startIdx+1:], `"`)
	if endIdx == -1 {
		return ""
	}
	return reason[startIdx+1 : startIdx+1+endIdx]
}

// deduplicateInt64 removes duplicate values from a slice of int64.
func deduplicateInt64(slice []int64) []int64 {
	seen := make(map[int64]bool)
	result := make([]int64, 0, len(slice))
	for _, val := range slice {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}
