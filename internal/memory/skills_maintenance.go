package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	SimilarityThreshold float64                                                 // Similarity above this triggers consolidation (default 0.85)
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
				Tier:   "skills",
				Action: "prune",
				Target: slug,
				Reason: fmt.Sprintf("Unused: no retrievals in %d days, utility %.2f", daysSinceCreation, utility),
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
			Tier:   "skills",
			Action: "decay",
			Target: slug,
			Reason: fmt.Sprintf("Low utility %.2f despite %d retrievals", utility, retrievalCount),
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
					Tier:   "skills",
					Action: "consolidate",
					Target: mergeSkill.slug,
					Reason: fmt.Sprintf("Redundant with %q (similarity %.2f)", keepSkill.slug, sim),
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
				Tier:   "skills",
				Action: "promote",
				Target: slug,
				Reason: fmt.Sprintf("High utility %.2f across %d projects", utility, projectCount),
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
				Tier:   "skills",
				Action: "demote",
				Target: slug,
				Reason: fmt.Sprintf("Narrow scope: only used in %d project(s)", projectCount),
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
	// Extract target and destination from proposal (format: "target -> destination")
	// For now, just soft-delete the target skill
	// A more sophisticated implementation would merge content and source_memory_ids

	_, err := a.db.Exec("UPDATE generated_skills SET pruned = 1 WHERE slug = ?", proposal.Target)
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
// (Placeholder implementation - full split requires re-clustering logic)
func (a *SkillsApplier) applySplit(proposal MaintenanceProposal) error {
	// This would require re-clustering source memories and creating new skills
	// For now, return not implemented
	return fmt.Errorf("split action not yet implemented")
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
// (Placeholder implementation - full demotion requires creating embedding entries)
func (a *SkillsApplier) applyDemote(proposal MaintenanceProposal) error {
	// This would require creating embeddings from skill content
	// For now, just soft-delete the skill
	return a.applyPrune(proposal)
}
