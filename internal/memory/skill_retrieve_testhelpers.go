//go:build sqlite_fts5

package memory

import "database/sql"

// Test wrappers for unexported skill_retrieve functions (TASK-4)

// SearchSkillsForTest wraps searchSkills for blackbox testing.
func SearchSkillsForTest(db *sql.DB, queryEmbedding []float32, limit int) ([]SkillSearchResult, error) {
	return searchSkills(db, queryEmbedding, limit)
}

// FormatSkillContextForTest wraps FormatSkillContext for blackbox testing.
func FormatSkillContextForTest(skills []SkillSearchResult) string {
	return FormatSkillContext(skills)
}
