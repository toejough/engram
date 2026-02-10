package memory

import (
	"database/sql"
	"fmt"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

// SkillSearchResult represents a generated skill found via similarity search.
type SkillSearchResult struct {
	Slug        string
	Theme       string
	Description string
	Confidence  float64
	Utility     float64
	Similarity  float64
}

// searchSkills performs vector similarity search on generated_skills
// via their embedding_id. Only returns non-pruned skills with confidence > 0.3.
func searchSkills(db *sql.DB, queryEmbedding []float32, limit int) ([]SkillSearchResult, error) {
	query := `
		SELECT gs.slug, gs.theme, gs.description,
		       gs.alpha, gs.beta, gs.utility,
		       (1 - vec_distance_cosine(v.embedding, ?)) as similarity
		FROM generated_skills gs
		JOIN vec_embeddings v ON v.rowid = gs.embedding_id
		WHERE gs.pruned = 0
		  AND gs.embedding_id IS NOT NULL
		  AND (gs.alpha / (gs.alpha + gs.beta)) > 0.3
		ORDER BY similarity DESC
		LIMIT ?
	`

	queryBlob, err := sqlite_vec.SerializeFloat32(queryEmbedding)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, queryBlob, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	results := make([]SkillSearchResult, 0)
	for rows.Next() {
		var r SkillSearchResult
		var alpha, beta float64
		if err := rows.Scan(&r.Slug, &r.Theme, &r.Description,
			&alpha, &beta, &r.Utility, &r.Similarity); err != nil {
			return nil, err
		}
		if alpha+beta > 0 {
			r.Confidence = alpha / (alpha + beta)
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// FormatSkillContext renders a markdown section for relevant skills.
// Returns empty string if no skills are provided.
func FormatSkillContext(skills []SkillSearchResult) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Relevant Skills\n\n")

	for _, skill := range skills {
		confPct := int(skill.Confidence * 100)
		sb.WriteString(fmt.Sprintf("### %s (Confidence: %d%%)\n\n", skill.Theme, confPct))
		sb.WriteString(fmt.Sprintf("%s\n\n", skill.Description))
	}

	return sb.String()
}
