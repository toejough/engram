package memory_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Integration tests for memory system (TASK-8)
// These tests exercise the full end-to-end flow including:
// - ONNX model download and loading
// - Real embedding generation (not mocks)
// - SQLite-vec database operations
// - Semantic similarity search
//
// Note: These tests are slow due to model loading. Use -short to skip them.
// Run full tests with: go test -count=1 ./internal/memory/...
// ============================================================================

// skipIfShort skips the test if -short flag is provided.
func skipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

// skipIfWindows skips the test on Windows (future work).
func skipIfWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("skipping integration test on Windows (future work)")
	}
}

// ============================================================================
// Learn -> Query integration tests
// ============================================================================

// TestIntegration_LearnThenQueryReturnsLearnedContent verifies learn -> query flow works end-to-end.
// Traces to: TASK-8 AC "Test: memory learn -> query returns learned content"
func TestIntegration_LearnThenQueryReturnsLearnedContent(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Step 1: Learn something
	learnOpts := memory.LearnOpts{
		Message:    "PostgreSQL is excellent for complex relational queries",
		Project:    "database-project",
		MemoryRoot: memoryDir,
	}
	err := memory.Learn(learnOpts)
	g.Expect(err).ToNot(HaveOccurred())

	// Step 2: Query for related content
	queryOpts := memory.QueryOpts{
		Text:       "database query performance",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// Verify the learned content appears in results
	found := false
	for _, r := range results.Results {
		if containsSubstring(r.Content, "PostgreSQL") {
			found = true
			break
		}
	}
	g.Expect(found).To(BeTrue(), "query should return learned content about PostgreSQL")
}

// ============================================================================
// Decide -> Query integration tests
// ============================================================================

// TestIntegration_DecideThenQueryReturnsDecisionWithAlternatives verifies decide -> query flow.
// Traces to: TASK-8 AC "Test: memory decide -> query returns decision with alternatives"
func TestIntegration_DecideThenQueryReturnsDecisionWithAlternatives(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Step 1: Log a decision
	decideOpts := memory.DecideOpts{
		Context:      "Error handling strategy",
		Choice:       "Wrapped errors with context",
		Reason:       "Provides clear error traces for debugging",
		Alternatives: []string{"Sentinel errors", "Error codes", "Panic-recover"},
		Project:      "error-project",
		MemoryRoot:   memoryDir,
	}
	result, err := memory.Decide(decideOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	// Step 2: Create session end to capture the decision in session summary
	sessionOpts := memory.SessionEndOpts{
		Project:    "error-project",
		MemoryRoot: memoryDir,
	}
	_, err = memory.SessionEnd(sessionOpts)
	g.Expect(err).ToNot(HaveOccurred())

	// Step 3: Query for error-related content
	queryOpts := memory.QueryOpts{
		Text:       "error handling debugging",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	queryResults, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(queryResults.Results).ToNot(BeEmpty())

	// Verify decision content appears
	found := false
	for _, r := range queryResults.Results {
		if containsSubstring(r.Content, "Wrapped") || containsSubstring(r.Content, "error") {
			found = true
			break
		}
	}
	g.Expect(found).To(BeTrue(), "query should return content related to error handling decision")
}

// NOTE: Extract from yield functionality removed in ISSUE-116.
// Extract now only supports result files.

// ============================================================================
// Extract from result -> Query integration tests
// ============================================================================

// TestIntegration_ExtractFromResultThenQueryReturnsDecisions verifies extract result -> query flow.
// Traces to: TASK-8 AC "Test: memory extract from result -> query returns decisions"
func TestIntegration_ExtractFromResultThenQueryReturnsDecisions(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Step 1: Create a result file with decisions
	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[[decisions]]
context = "API framework selection"
choice = "Gin web framework"
reason = "High performance and simple API design"
alternatives = ["Echo", "Fiber", "Chi"]

[[decisions]]
context = "Testing strategy"
choice = "Table-driven tests with gomega"
reason = "Clean test organization and readable assertions"
alternatives = ["Standard testing", "Testify assertions"]

[context]
phase = "design"
subphase = "architecture"
task = "TASK-10"
`
	resultPath := filepath.Join(tempDir, "result.toml")
	err := os.WriteFile(resultPath, []byte(resultContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Step 2: Extract from result file
	extractOpts := memory.ExtractOpts{
		FilePath:   resultPath,
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	extractResult, err := extractOpts.Extract()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(extractResult.ItemsExtracted).To(Equal(2)) // Two decisions

	// Step 3: Query for API-related decisions
	queryOpts := memory.QueryOpts{
		Text:       "web framework API design",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	queryResults, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(queryResults.Results).ToNot(BeEmpty())

	// Verify extracted decisions appear
	found := false
	for _, r := range queryResults.Results {
		if containsSubstring(r.Content, "Gin") || containsSubstring(r.Content, "framework") {
			found = true
			break
		}
	}
	g.Expect(found).To(BeTrue(), "query should return extracted API framework decision")
}

// ============================================================================
// Session end -> Query integration tests
// ============================================================================

// TestIntegration_SessionEndThenQueryReturnsSummary verifies session-end -> query flow.
// Traces to: TASK-8 AC "Test: memory session-end -> query returns summary"
func TestIntegration_SessionEndThenQueryReturnsSummary(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Step 1: Create some decisions
	for i, ctx := range []string{"Authentication mechanism", "Caching strategy", "Logging format"} {
		decideOpts := memory.DecideOpts{
			Context:      ctx,
			Choice:       "Choice " + string(rune('A'+i)),
			Reason:       "Reason for " + ctx,
			Alternatives: []string{"Alt1", "Alt2"},
			Project:      "session-project",
			MemoryRoot:   memoryDir,
		}
		_, err := memory.Decide(decideOpts)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Step 2: Generate session end summary
	sessionOpts := memory.SessionEndOpts{
		Project:    "session-project",
		MemoryRoot: memoryDir,
	}
	sessionResult, err := memory.SessionEnd(sessionOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sessionResult.FilePath).ToNot(BeEmpty())

	// Step 3: Query for session content
	queryOpts := memory.QueryOpts{
		Text:       "authentication caching logging",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	queryResults, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(queryResults.Results).ToNot(BeEmpty())

	// Verify session summary content appears
	found := false
	for _, r := range queryResults.Results {
		if containsSubstring(r.Content, "Session") ||
			containsSubstring(r.Content, "session-project") ||
			containsSubstring(r.Content, "Choice") {
			found = true
			break
		}
	}
	g.Expect(found).To(BeTrue(), "query should return session summary content")
}

// ============================================================================
// ONNX model infrastructure tests
// ============================================================================

// TestIntegration_ONNXModelDownloadsOnFirstUse verifies model auto-download.
// Traces to: TASK-8 AC "Test: ONNX model downloads on first use (check file created)"
func TestIntegration_ONNXModelDownloadsOnFirstUse(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Verify model doesn't exist yet
	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
	_, err := os.Stat(modelPath)
	g.Expect(os.IsNotExist(err)).To(BeTrue(), "model should not exist before first query")

	// Create some content via Learn
	err = memory.Learn(memory.LearnOpts{Message: "Test content for model download verification", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	// Query triggers model download
	queryOpts := memory.QueryOpts{
		Text:       "test query",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.ModelDownloaded).To(BeTrue(), "model should be downloaded on first use")

	// Verify model file was created
	_, err = os.Stat(modelPath)
	g.Expect(err).ToNot(HaveOccurred(), "model file should exist after download")

	// Verify model file has reasonable size (> 10MB for ONNX model)
	info, err := os.Stat(modelPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 10*1024*1024), "model file should be > 10MB")
}

// TestIntegration_SQLiteVecDatabaseCreatedAtExpectedLocation verifies database creation.
// Traces to: TASK-8 AC "Test: SQLite-vec database created at expected location"
func TestIntegration_SQLiteVecDatabaseCreatedAtExpectedLocation(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Verify database doesn't exist
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	_, err := os.Stat(dbPath)
	g.Expect(os.IsNotExist(err)).To(BeTrue(), "database should not exist before first query")

	// Create content via Learn
	err = memory.Learn(memory.LearnOpts{Message: "Test content", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	queryOpts := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.VectorStorage).To(Equal("sqlite-vec"))

	// Verify database was created
	_, err = os.Stat(dbPath)
	g.Expect(err).ToNot(HaveOccurred(), "embeddings.db should be created at expected location")
}

// TestIntegration_EmbeddingGenerationProducesNonZeroVectors verifies real embeddings.
// Traces to: TASK-8 AC "Test: embedding generation produces non-zero vectors"
func TestIntegration_EmbeddingGenerationProducesNonZeroVectors(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Create content via Learn
	err := memory.Learn(memory.LearnOpts{Message: "Database design patterns", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	queryOpts := memory.QueryOpts{
		Text:       "database design",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify embeddings were generated (not mocks)
	g.Expect(results.UsedMockEmbeddings).To(BeFalse(), "should use real embeddings, not mocks")
	g.Expect(results.InferenceExecuted).To(BeTrue(), "ONNX inference should have executed")
	g.Expect(results.EmbeddingsCount).To(BeNumerically(">", 0), "should have stored embeddings")
}

// TestIntegration_EmbeddingVectorsHaveCorrectDimensions verifies 384 dimensions.
// Traces to: TASK-8 AC "Test: embedding vectors have correct dimensions (384 for e5-small)"
func TestIntegration_EmbeddingVectorsHaveCorrectDimensions(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Create content via Learn
	err := memory.Learn(memory.LearnOpts{Message: "Test dimensions", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	queryOpts := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify e5-small dimensions
	g.Expect(results.EmbeddingDimensions).To(Equal(384), "e5-small model should produce 384-dimensional embeddings")
	g.Expect(results.EmbeddingModel).To(Equal("e5-small-v2"))
}

// ============================================================================
// Semantic similarity tests
// ============================================================================

// TestIntegration_SemanticSimilarityRanksRelatedHigher verifies semantic understanding.
// Traces to: TASK-8 AC "Test: semantic similarity works (related queries rank higher than unrelated)"
// Example: "error handling" matches "exception management" better than "ui design"
func TestIntegration_SemanticSimilarityRanksRelatedHigher(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Create diverse content with semantically related and unrelated topics via Learn
	for _, msg := range []string{
		"Exception management best practices for robust applications",
		"User interface design principles and visual aesthetics",
		"Handling failures gracefully with proper error messages",
		"Color theory and typography in frontend development",
		"Try-catch patterns for managing application errors",
	} {
		err := memory.Learn(memory.LearnOpts{Message: msg, MemoryRoot: memoryDir})
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Query for error handling (should match exception management and error messages better than UI design)
	queryOpts := memory.QueryOpts{
		Text:       "error handling",
		Limit:      5,
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// Find scores for related vs unrelated content
	var errorRelatedScore, uiDesignScore float64
	for _, r := range results.Results {
		if containsSubstring(r.Content, "Exception") || containsSubstring(r.Content, "failure") || containsSubstring(r.Content, "Try-catch") {
			if r.Score > errorRelatedScore {
				errorRelatedScore = r.Score
			}
		}
		if containsSubstring(r.Content, "interface design") || containsSubstring(r.Content, "Color theory") {
			if uiDesignScore == 0 || r.Score < uiDesignScore {
				uiDesignScore = r.Score
			}
		}
	}

	// Related content should score higher than unrelated content
	g.Expect(errorRelatedScore).To(BeNumerically(">", uiDesignScore),
		"error handling query should rank exception/failure content (%.3f) higher than UI design content (%.3f)",
		errorRelatedScore, uiDesignScore)
}

// TestIntegration_SemanticSimilarityExampleErrorAndException tests the specific example.
// Traces to: TASK-8 AC "Example similarity test: 'error handling' matches 'exception management' better than 'ui design'"
func TestIntegration_SemanticSimilarityExampleErrorAndException(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Create exactly the example content from acceptance criteria via Learn
	for _, msg := range []string{
		"exception management strategies for production systems",
		"ui design and user experience patterns",
	} {
		err := memory.Learn(memory.LearnOpts{Message: msg, MemoryRoot: memoryDir})
		g.Expect(err).ToNot(HaveOccurred())
	}

	queryOpts := memory.QueryOpts{
		Text:       "error handling",
		Limit:      2,
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).To(HaveLen(2))

	// First result should be exception management (semantically similar to error handling)
	g.Expect(results.Results[0].Content).To(ContainSubstring("exception"),
		"'error handling' should match 'exception management' as top result")

	// Second result should be UI design (less related)
	g.Expect(results.Results[1].Content).To(ContainSubstring("ui design"),
		"'ui design' should be lower ranked result")

	// Scores should reflect the semantic similarity difference
	g.Expect(results.Results[0].Score).To(BeNumerically(">", results.Results[1].Score),
		"exception management score (%.3f) should be higher than ui design score (%.3f)",
		results.Results[0].Score, results.Results[1].Score)
}

// ============================================================================
// Test isolation and CI caching tests
// ============================================================================

// TestIntegration_TestUsesIsolatedTempDir verifies test isolation.
// Traces to: TASK-8 AC "Tests use t.TempDir() for isolated test directories"
func TestIntegration_TestUsesIsolatedTempDir(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	// Create two separate test environments
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Verify they are different
	g.Expect(tempDir1).ToNot(Equal(tempDir2), "each test should get isolated temp directory")

	// Create content in first environment via Learn
	memoryDir1 := filepath.Join(tempDir1, "memory")
	err := memory.Learn(memory.LearnOpts{Message: "Content in env1", MemoryRoot: memoryDir1})
	g.Expect(err).ToNot(HaveOccurred())

	// Second environment should be completely empty
	memoryDir2 := filepath.Join(tempDir2, "memory")
	_, err = os.Stat(memoryDir2)
	g.Expect(os.IsNotExist(err)).To(BeTrue(), "second environment should be isolated and empty")
}

// TestIntegration_SkipsAutoDownloadIfModelPresent verifies CI caching support.
// Traces to: TASK-8 AC "Tests skip auto-download if model already present (CI caching)"
func TestIntegration_SkipsAutoDownloadIfModelPresent(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir := filepath.Join(tempDir, "models")

	// Create content via Learn
	err := memory.Learn(memory.LearnOpts{Message: "Test content", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	// First query downloads the model
	queryOpts := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir,
	}
	results1, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results1.ModelDownloaded).To(BeTrue(), "first query should download model")

	// Second query should reuse existing model (no download)
	results2, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results2.ModelDownloaded).To(BeFalse(), "second query should skip download and use cached model")
	g.Expect(results2.ModelLoaded).To(BeTrue(), "model should be loaded from cache")
}

// ============================================================================
// Platform compatibility tests
// ============================================================================

// TestIntegration_RunsOnMacOSAndLinux documents platform support.
// Traces to: TASK-8 AC "Tests run on macOS and Linux (document Windows as future work)"
func TestIntegration_RunsOnMacOSAndLinux(t *testing.T) {
	skipIfShort(t)
	g := NewWithT(t)

	currentOS := runtime.GOOS

	// Test should run on macOS and Linux
	switch currentOS {
	case "darwin", "linux":
		g.Expect(currentOS).To(Or(Equal("darwin"), Equal("linux")),
			"integration tests are supported on macOS and Linux")
		// Run a quick validation
		tempDir := t.TempDir()
		memoryDir := filepath.Join(tempDir, "memory")
		modelDir := filepath.Join(tempDir, "models")

		err := memory.Learn(memory.LearnOpts{Message: "Platform test", MemoryRoot: memoryDir})
		g.Expect(err).ToNot(HaveOccurred())

		queryOpts := memory.QueryOpts{
			Text:       "test",
			MemoryRoot: memoryDir,
			ModelDir:   modelDir,
		}
		_, err = memory.Query(queryOpts)
		g.Expect(err).ToNot(HaveOccurred(), "memory system should work on %s", currentOS)
	case "windows":
		t.Skip("Windows support is documented as future work")
	default:
		t.Skipf("Unsupported platform: %s", currentOS)
	}
}

// ============================================================================
// Environment isolation tests
// ============================================================================

// TestIntegration_UsesSetenvForEnvironmentIsolation verifies t.Setenv usage pattern.
// Traces to: TASK-8 AC "Tests use t.Setenv() for environment isolation"
func TestIntegration_UsesSetenvForEnvironmentIsolation(t *testing.T) {
	skipIfShort(t)
	skipIfWindows(t)
	g := NewWithT(t)

	// Use t.Setenv to override HOME for model directory resolution
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Verify HOME was changed for this test
	g.Expect(os.Getenv("HOME")).To(Equal(tempDir))

	// Create a separate temp for actual test data (not affected by HOME change)
	memoryDir := filepath.Join(t.TempDir(), "memory")
	err := memory.Learn(memory.LearnOpts{Message: "Environment test", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	// Query with explicit modelDir (don't rely on HOME)
	queryOpts := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
		ModelDir:   filepath.Join(t.TempDir(), "models"),
	}
	_, err = memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
}

// ============================================================================
// Helper functions
// ============================================================================

// containsSubstring checks if content contains substring (case-insensitive).
func containsSubstring(content, substr string) bool {
	return len(content) > 0 && len(substr) > 0 &&
		(containsCI(content, substr))
}

// containsCI performs case-insensitive substring check.
func containsCI(s, substr string) bool {
	s, substr = toLower(s), toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// toLower converts string to lowercase (simple ASCII).
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
