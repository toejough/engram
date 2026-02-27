//go:build sqlite_fts5

package memory

import (
	"database/sql"
	"fmt"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

// CheckContextForTest wraps checkContext for blackbox testing.
func CheckContextForTest(opts OptimizeOpts) error {
	return checkContext(opts)
}

// ClusterEntriesToEmbeddingsForTest wraps clusterEntriesToEmbeddings for blackbox testing.
func ClusterEntriesToEmbeddingsForTest(cluster []ClusterEntry) []Embedding {
	return clusterEntriesToEmbeddings(cluster)
}

// ClusterHasExistingSkillForTest wraps clusterHasExistingSkill for blackbox testing.
func ClusterHasExistingSkillForTest(cluster []ClusterEntry, existingSourceIDs map[int64]bool) bool {
	return clusterHasExistingSkill(cluster, existingSourceIDs)
}

// CosineSimilarityForTest wraps cosineSimilarity for blackbox testing.
func CosineSimilarityForTest(a, b []float32) float64 {
	return cosineSimilarity(a, b)
}

// DefaultSkillTesterForTest creates a defaultSkillTester for blackbox testing.
func DefaultSkillTesterForTest(apiKey string) SkillTester {
	return &defaultSkillTester{apiKey: apiKey}
}

// FormatClusterSourceIDsForTest wraps formatClusterSourceIDs for blackbox testing.
func FormatClusterSourceIDsForTest(cluster []ClusterEntry) string {
	return formatClusterSourceIDs(cluster)
}

// GenerateSkillFromLearningForTest wraps generateSkillFromLearning for blackbox testing.
func GenerateSkillFromLearningForTest(db *sql.DB, opts OptimizeOpts, learning string) error {
	return generateSkillFromLearning(db, opts, learning)
}

// GenerateThemeFromClusterForTest wraps generateThemeFromCluster for blackbox testing.
func GenerateThemeFromClusterForTest(cluster []ClusterEntry) string {
	return generateThemeFromCluster(cluster)
}

// GetExistingPatternsForTest wraps getExistingPatterns for blackbox testing.
func GetExistingPatternsForTest(db *sql.DB) []string {
	return getExistingPatterns(db)
}

// GetExistingSkillSourceIDsForTest wraps getExistingSkillSourceIDs for blackbox testing.
func GetExistingSkillSourceIDsForTest(db *sql.DB) (map[int64]bool, error) {
	return getExistingSkillSourceIDs(db)
}

// GetONNXRuntimeInitializedForTest returns the current value of the global onnxRuntimeInitialized flag.
func GetONNXRuntimeInitializedForTest() bool {
	return onnxRuntimeInitialized
}

// HasLearnTimestampPrefixForTest wraps hasLearnTimestampPrefix for blackbox testing.
func HasLearnTimestampPrefixForTest(content string) bool {
	return hasLearnTimestampPrefix(content)
}

// InsertVecEmbeddingForTest inserts a float32 embedding into vec_embeddings for test setup.
// Returns the rowid of the inserted row.
func InsertVecEmbeddingForTest(db *sql.DB, embedding []float32) (int64, error) {
	blob, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return 0, fmt.Errorf("serialize embedding: %w", err)
	}

	res, err := db.Exec(`INSERT INTO vec_embeddings(embedding) VALUES (?)`, blob)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

// IsNarrowByKeywordsForTest wraps isNarrowByKeywords for blackbox testing.
func IsNarrowByKeywordsForTest(learning string) (bool, string) {
	return isNarrowByKeywords(learning)
}

// OptimizeAutoDemoteForTest wraps optimizeAutoDemote for blackbox testing.
func OptimizeAutoDemoteForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizeAutoDemote(db, opts, result)
}

// OptimizeClaudeMDDedupForTest wraps optimizeClaudeMDDedup for blackbox testing.
func OptimizeClaudeMDDedupForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizeClaudeMDDedup(db, opts, result)
}

// OptimizeCompileSkillsForTest wraps optimizeCompileSkills for blackbox testing.
func OptimizeCompileSkillsForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizeCompileSkills(db, opts, result)
}

// OptimizeContradictionsForTest wraps optimizeContradictions for blackbox testing.
func OptimizeContradictionsForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizeContradictions(db, opts, result)
}

// OptimizeDecayForTest wraps optimizeDecay for blackbox testing.
func OptimizeDecayForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizeDecay(db, opts, result)
}

// OptimizeDedupForTest wraps optimizeDedup for blackbox testing.
func OptimizeDedupForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizeDedup(db, opts, result)
}

// OptimizeDemoteClaudeMDForTest wraps optimizeDemoteClaudeMD for blackbox testing.
func OptimizeDemoteClaudeMDForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizeDemoteClaudeMD(db, opts, result)
}

// OptimizeMergeSkillsForTest wraps optimizeMergeSkills for blackbox testing.
func OptimizeMergeSkillsForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizeMergeSkills(db, opts, result)
}

// OptimizePromoteForTest wraps optimizePromote for blackbox testing.
func OptimizePromoteForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizePromote(db, opts, result)
}

// OptimizePromoteSkillsForTest wraps optimizePromoteSkills for blackbox testing.
func OptimizePromoteSkillsForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizePromoteSkills(db, opts, result)
}

// OptimizePruneForTest wraps optimizePrune for blackbox testing.
func OptimizePruneForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizePrune(db, opts, result)
}

// OptimizePurgeBoilerplateForTest wraps optimizePurgeBoilerplate for blackbox testing.
func OptimizePurgeBoilerplateForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizePurgeBoilerplate(db, opts, result)
}

// OptimizePurgeLegacySessionEmbeddingsForTest wraps optimizePurgeLegacySessionEmbeddings for blackbox testing.
func OptimizePurgeLegacySessionEmbeddingsForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizePurgeLegacySessionEmbeddings(db, opts, result)
}

// OptimizeSplitSkillsForTest wraps optimizeSplitSkills for blackbox testing.
func OptimizeSplitSkillsForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizeSplitSkills(db, opts, result)
}

// OptimizeSynthesizeForTest wraps optimizeSynthesize for blackbox testing.
func OptimizeSynthesizeForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return optimizeSynthesize(db, opts, result)
}

// PerformSkillReorganizationForTest wraps performSkillReorganization for blackbox testing.
func PerformSkillReorganizationForTest(db *sql.DB, opts OptimizeOpts, result *OptimizeResult) error {
	return performSkillReorganization(db, opts, result)
}

// PruneOrphanedSkillsForTest wraps pruneOrphanedSkills for blackbox testing.
func PruneOrphanedSkillsForTest(db *sql.DB, skillsDir string, activeThemes map[string]bool) (int, error) {
	return pruneOrphanedSkills(db, skillsDir, activeThemes)
}

// PruneStaleSkillsForTest wraps pruneStaleSkills for blackbox testing.
func PruneStaleSkillsForTest(db *sql.DB, skillsDir string, autoDemoteUtility float64) int {
	return pruneStaleSkills(db, skillsDir, autoDemoteUtility)
}

// SetONNXRuntimeInitializedForTest sets the global onnxRuntimeInitialized flag for testing.
// This bypasses the ONNX runtime download in functions that call initializeONNXRuntime.
// NOTE: Do not use in parallel tests - this mutates global state.
func SetONNXRuntimeInitializedForTest(v bool) {
	onnxRuntimeInitialized = v
}
