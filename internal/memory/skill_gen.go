package memory

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// GeneratedSkill represents a dynamically generated skill from memory clusters.
type GeneratedSkill struct {
	ID              int64
	Slug            string
	Theme           string
	Description     string
	Content         string
	SourceMemoryIDs string // JSON array of embedding IDs
	Alpha           float64
	Beta            float64
	Utility         float64
	RetrievalCount  int
	LastRetrieved   string
	CreatedAt       string
	UpdatedAt       string
	Pruned          bool
	EmbeddingID     int64
}

// SkillConfidence represents Beta distribution parameters for skill confidence.
type SkillConfidence struct {
	Alpha float64
	Beta  float64
}

// Mean returns the mean of the Beta distribution (alpha/(alpha+beta)).
func (sc SkillConfidence) Mean() float64 {
	sum := sc.Alpha + sc.Beta
	if sum == 0 {
		return 0
	}
	return sc.Alpha / sum
}

// SkillGenOpts holds options for skill generation.
type SkillGenOpts struct {
	SkillsDir      string
	MinClusterSize int
	MinUtility     float64
}

// SkillGenResult holds the result of skill generation.
type SkillGenResult struct {
	SkillsCompiled int
	SkillsMerged   int
	SkillsPruned   int
}

// computeUtility calculates skill utility using the MACLA formula:
// utility = 0.5*(alpha/(alpha+beta)) + 0.3*min(1, ln(1+retrievals)/5) + 0.2*exp(-days_since_last/30)
func computeUtility(alpha, beta float64, retrievals int, lastRetrieved string) float64 {
	// Confidence score
	confidence := alpha / (alpha + beta)

	// Retrieval score
	retrievalScore := math.Min(1.0, math.Log(1+float64(retrievals))/5.0)

	// Recency score
	var recencyScore float64
	if lastRetrieved != "" {
		t, err := time.Parse(time.RFC3339, lastRetrieved)
		if err == nil {
			daysSince := time.Since(t).Hours() / 24.0
			recencyScore = math.Exp(-daysSince / 30.0)
		}
	}

	utility := 0.5*confidence + 0.3*retrievalScore + 0.2*recencyScore

	// Clamp to [0, 1]
	if utility < 0 {
		utility = 0
	}
	if utility > 1 {
		utility = 1
	}

	return utility
}

// insertSkill inserts a GeneratedSkill and returns its auto-generated ID.
func insertSkill(db *sql.DB, skill *GeneratedSkill) (int64, error) {
	prunedInt := 0
	if skill.Pruned {
		prunedInt = 1
	}

	stmt := `INSERT INTO generated_skills (
		slug, theme, description, content, source_memory_ids,
		alpha, beta, utility, retrieval_count, last_retrieved,
		created_at, updated_at, pruned, embedding_id
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(stmt,
		skill.Slug, skill.Theme, skill.Description, skill.Content, skill.SourceMemoryIDs,
		skill.Alpha, skill.Beta, skill.Utility, skill.RetrievalCount, nullString(skill.LastRetrieved),
		skill.CreatedAt, skill.UpdatedAt, prunedInt, nullInt64(skill.EmbeddingID))

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// getSkillBySlug retrieves a GeneratedSkill by its slug.
// Returns (nil, nil) if not found.
func getSkillBySlug(db *sql.DB, slug string) (*GeneratedSkill, error) {
	stmt := `SELECT
		id, slug, theme, description, content, source_memory_ids,
		alpha, beta, utility, retrieval_count, last_retrieved,
		created_at, updated_at, pruned, embedding_id
	FROM generated_skills WHERE slug = ?`

	skill := &GeneratedSkill{}
	var prunedInt int
	var lastRetrieved sql.NullString
	var embeddingID sql.NullInt64

	err := db.QueryRow(stmt, slug).Scan(
		&skill.ID, &skill.Slug, &skill.Theme, &skill.Description, &skill.Content, &skill.SourceMemoryIDs,
		&skill.Alpha, &skill.Beta, &skill.Utility, &skill.RetrievalCount, &lastRetrieved,
		&skill.CreatedAt, &skill.UpdatedAt, &prunedInt, &embeddingID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	skill.Pruned = prunedInt != 0
	if lastRetrieved.Valid {
		skill.LastRetrieved = lastRetrieved.String
	}
	if embeddingID.Valid {
		skill.EmbeddingID = embeddingID.Int64
	}

	return skill, nil
}

// listSkills returns all non-pruned GeneratedSkills.
func listSkills(db *sql.DB) ([]GeneratedSkill, error) {
	stmt := `SELECT
		id, slug, theme, description, content, source_memory_ids,
		alpha, beta, utility, retrieval_count, last_retrieved,
		created_at, updated_at, pruned, embedding_id
	FROM generated_skills WHERE pruned = 0`

	rows, err := db.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	skills := make([]GeneratedSkill, 0)

	for rows.Next() {
		var skill GeneratedSkill
		var prunedInt int
		var lastRetrieved sql.NullString
		var embeddingID sql.NullInt64

		err := rows.Scan(
			&skill.ID, &skill.Slug, &skill.Theme, &skill.Description, &skill.Content, &skill.SourceMemoryIDs,
			&skill.Alpha, &skill.Beta, &skill.Utility, &skill.RetrievalCount, &lastRetrieved,
			&skill.CreatedAt, &skill.UpdatedAt, &prunedInt, &embeddingID)

		if err != nil {
			return nil, err
		}

		skill.Pruned = prunedInt != 0
		if lastRetrieved.Valid {
			skill.LastRetrieved = lastRetrieved.String
		}
		if embeddingID.Valid {
			skill.EmbeddingID = embeddingID.Int64
		}

		skills = append(skills, skill)
	}

	return skills, rows.Err()
}

// updateSkill updates an existing GeneratedSkill by ID.
func updateSkill(db *sql.DB, skill *GeneratedSkill) error {
	prunedInt := 0
	if skill.Pruned {
		prunedInt = 1
	}

	stmt := `UPDATE generated_skills SET
		slug = ?, theme = ?, description = ?, content = ?, source_memory_ids = ?,
		alpha = ?, beta = ?, utility = ?, retrieval_count = ?, last_retrieved = ?,
		created_at = ?, updated_at = ?, pruned = ?, embedding_id = ?
	WHERE id = ?`

	_, err := db.Exec(stmt,
		skill.Slug, skill.Theme, skill.Description, skill.Content, skill.SourceMemoryIDs,
		skill.Alpha, skill.Beta, skill.Utility, skill.RetrievalCount, nullString(skill.LastRetrieved),
		skill.CreatedAt, skill.UpdatedAt, prunedInt, nullInt64(skill.EmbeddingID),
		skill.ID)

	return err
}

// softDeleteSkill marks a skill as pruned (soft delete).
func softDeleteSkill(db *sql.DB, id int64) error {
	_, err := db.Exec("UPDATE generated_skills SET pruned = 1 WHERE id = ?", id)
	return err
}

// nullString converts an empty string to NULL for SQL.
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// nullInt64 converts a zero int64 to NULL for SQL.
func nullInt64(i int64) interface{} {
	if i == 0 {
		return nil
	}
	return i
}

// scoreCluster computes the MACLA utility score for a memory cluster
// by averaging the utility of all member memories.
func scoreCluster(db *sql.DB, cluster []ClusterEntry) (float64, error) {
	if len(cluster) == 0 {
		return 0, nil
	}

	var totalUtility float64

	for _, entry := range cluster {
		// Fetch metadata for this embedding
		var confidence float64
		var retrievalCount int
		var lastRetrieved sql.NullString

		err := db.QueryRow(`
			SELECT confidence, retrieval_count, last_retrieved
			FROM embeddings
			WHERE id = ?
		`, entry.ID).Scan(&confidence, &retrievalCount, &lastRetrieved)

		if err != nil {
			return 0, fmt.Errorf("failed to fetch metadata for embedding %d: %w", entry.ID, err)
		}

		// Convert confidence to alpha/beta parameters (simplified: confidence -> alpha, 1-confidence -> beta)
		// For MACLA formula, we treat confidence as alpha/(alpha+beta)
		// If confidence = c, then alpha = c and beta = 1-c satisfies alpha/(alpha+beta) = c
		alpha := confidence
		beta := 1.0 - confidence
		if beta < 0.001 {
			beta = 0.001 // Avoid division by zero
		}

		lastRetrievedStr := ""
		if lastRetrieved.Valid {
			lastRetrievedStr = lastRetrieved.String
		}

		utility := computeUtility(alpha, beta, retrievalCount, lastRetrievedStr)
		totalUtility += utility
	}

	return totalUtility / float64(len(cluster)), nil
}

// generateSkillContent generates skill markdown content from a theme and cluster.
// Uses the SkillCompiler if available, otherwise falls back to a template.
func generateSkillContent(theme string, cluster []ClusterEntry, compiler SkillCompiler) (string, error) {
	// Try to use compiler if available
	if compiler != nil {
		memories := make([]string, len(cluster))
		for i, entry := range cluster {
			memories[i] = entry.Content
		}

		content, err := compiler.CompileSkill(theme, memories)
		if err == nil {
			return content, nil
		}
		// On error (including ErrLLMUnavailable), fall through to template
	}

	// Fallback: use simple template
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", theme))
	sb.WriteString("This skill was automatically generated from memory patterns.\n\n")
	sb.WriteString("## Related Memories\n\n")
	for i, entry := range cluster {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, entry.Content))
	}
	sb.WriteString("\n## Application\n\n")
	sb.WriteString("Apply these patterns when working on similar tasks.\n")

	return sb.String(), nil
}

// slugify converts a theme string to a URL-safe slug.
// Converts to lowercase, replaces non-alphanumeric with hyphens,
// removes consecutive hyphens, and trims leading/trailing hyphens.
func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace non-alphanumeric characters with hyphens
	reg := regexp.MustCompile("[^a-z0-9]+")
	s = reg.ReplaceAllString(s, "-")

	// Remove leading/trailing hyphens
	s = strings.Trim(s, "-")

	return s
}

// RecordSkillFeedback updates a skill's alpha/beta parameters and recomputes utility.
// If success is true, alpha is incremented; otherwise beta is incremented.
func RecordSkillFeedback(db *sql.DB, slug string, success bool) error {
	skill, err := getSkillBySlug(db, slug)
	if err != nil {
		return fmt.Errorf("failed to find skill %q: %w", slug, err)
	}
	if skill == nil {
		return fmt.Errorf("skill %q not found", slug)
	}

	if success {
		skill.Alpha += 1.0
	} else {
		skill.Beta += 1.0
	}

	skill.Utility = computeUtility(skill.Alpha, skill.Beta, skill.RetrievalCount, skill.LastRetrieved)
	skill.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return updateSkill(db, skill)
}

// ListSkillsPublic returns all non-pruned skills. Exported for CLI usage.
func ListSkillsPublic(db *sql.DB) ([]GeneratedSkill, error) {
	return listSkills(db)
}

// OpenSkillDB opens the embeddings database for skill operations.
func OpenSkillDB(memoryRoot string) (*sql.DB, error) {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	return initEmbeddingsDB(dbPath)
}

// writeSkillFile creates a SKILL.md file with YAML frontmatter.
func writeSkillFile(skillsDir string, skill *GeneratedSkill) error {
	// Create skill directory
	skillDir := filepath.Join(skillsDir, skill.Slug)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Compute confidence from alpha/beta
	confidence := skill.Alpha / (skill.Alpha + skill.Beta)

	// Build frontmatter
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: memory-gen-%s\n", skill.Slug))
	sb.WriteString(fmt.Sprintf("description: %s\n", skill.Description))
	sb.WriteString("context: inherit\n")
	sb.WriteString("model: haiku\n")
	sb.WriteString("user-invocable: false\n")
	sb.WriteString("generated: true\n")
	sb.WriteString(fmt.Sprintf("confidence: %.2f\n", confidence))
	sb.WriteString("source: memory-compilation\n")
	sb.WriteString("---\n\n")

	// Append skill content
	sb.WriteString(skill.Content)

	// Write to file
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
}
