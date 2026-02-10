//go:build sqlite_fts5

package memory

import "database/sql"

// Test wrappers for unexported skill_gen functions (TASK-1)

// InsertSkillForTest wraps insertSkill for blackbox testing.
func InsertSkillForTest(db *sql.DB, skill *GeneratedSkill) (int64, error) {
	return insertSkill(db, skill)
}

// GetSkillBySlugForTest wraps getSkillBySlug for blackbox testing.
func GetSkillBySlugForTest(db *sql.DB, slug string) (*GeneratedSkill, error) {
	return getSkillBySlug(db, slug)
}

// ListSkillsForTest wraps listSkills for blackbox testing.
func ListSkillsForTest(db *sql.DB) ([]GeneratedSkill, error) {
	return listSkills(db)
}

// UpdateSkillForTest wraps updateSkill for blackbox testing.
func UpdateSkillForTest(db *sql.DB, skill *GeneratedSkill) error {
	return updateSkill(db, skill)
}

// SoftDeleteSkillForTest wraps softDeleteSkill for blackbox testing.
func SoftDeleteSkillForTest(db *sql.DB, id int64) error {
	return softDeleteSkill(db, id)
}

// ComputeUtilityForTest wraps computeUtility for blackbox testing.
func ComputeUtilityForTest(alpha, beta float64, retrievals int, lastRetrieved string) float64 {
	return computeUtility(alpha, beta, retrievals, lastRetrieved)
}

// Test wrappers for unexported skill_gen functions (TASK-2)

// ScoreClusterForTest wraps scoreCluster for blackbox testing.
func ScoreClusterForTest(db *sql.DB, cluster []ClusterEntry) (float64, error) {
	return scoreCluster(db, cluster)
}

// GenerateSkillContentForTest wraps generateSkillContent for blackbox testing.
func GenerateSkillContentForTest(theme string, cluster []ClusterEntry, compiler SkillCompiler) (string, error) {
	return generateSkillContent(theme, cluster, compiler)
}

// SlugifyForTest wraps slugify for blackbox testing.
func SlugifyForTest(theme string) string {
	return slugify(theme)
}

// WriteSkillFileForTest wraps writeSkillFile for blackbox testing.
func WriteSkillFileForTest(skillsDir string, skill *GeneratedSkill) error {
	return writeSkillFile(skillsDir, skill)
}

// RecordSkillFeedbackForTest wraps RecordSkillFeedback for blackbox testing.
func RecordSkillFeedbackForTest(db *sql.DB, slug string, success bool) error {
	return RecordSkillFeedback(db, slug, success)
}

// RecordSkillUsageForTest wraps RecordSkillUsage for blackbox testing (TASK-9).
func RecordSkillUsageForTest(db *sql.DB, slug string, success bool) error {
	return RecordSkillUsage(db, slug, success)
}

// MigrateMemoryGenSkillsForTest wraps migrateMemoryGenSkills for blackbox testing.
func MigrateMemoryGenSkillsForTest(skillsDir string) error {
	return migrateMemoryGenSkills(skillsDir)
}
