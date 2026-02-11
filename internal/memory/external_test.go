package memory_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// TASK-42: External knowledge capture - LearnOpts.Source field
// ============================================================================

// TEST-900: LearnOpts accepts Source field "internal"
// traces: ARCH-061, REQ-014
func TestLearnOptsAcceptsSourceInternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "Internal learning",
		Source:     "internal",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-901: LearnOpts accepts Source field "external"
// traces: ARCH-061, REQ-014
func TestLearnOptsAcceptsSourceExternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "External learning from docs",
		Source:     "external",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-902: LearnOpts defaults Source to "internal" when empty
// traces: ARCH-061, REQ-014
func TestLearnOptsDefaultsSourceToInternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn without Source set (should default to "internal")
	opts := memory.LearnOpts{
		Message:    "Default source learning",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify the stored entry has source_type "internal" in the DB
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var sourceType string
	err = db.QueryRow("SELECT source_type FROM embeddings ORDER BY id DESC LIMIT 1").Scan(&sourceType)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sourceType).To(Equal("internal"))
}

// ============================================================================
// TASK-42: External knowledge capture - confidence values
// ============================================================================

// TEST-903: Internal memories get default confidence 1.0
// traces: ARCH-061, REQ-014
func TestLearnInternalConfidenceIsOne(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "High confidence internal learning",
		Source:     "internal",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Check confidence in database
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var confidence float64
	err = db.QueryRow("SELECT confidence FROM embeddings ORDER BY id DESC LIMIT 1").Scan(&confidence)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(confidence).To(BeNumerically("==", 1.0))
}

// TEST-904: External memories get initial confidence 0.7
// traces: ARCH-061, REQ-014
func TestLearnExternalConfidenceIsPointSeven(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "External learning from web search",
		Source:     "external",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Check confidence in database
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var confidence float64
	err = db.QueryRow("SELECT confidence FROM embeddings ORDER BY id DESC LIMIT 1").Scan(&confidence)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(confidence).To(BeNumerically("~", 0.7, 0.001))
}

// TEST-905: Default (empty source) memories get confidence 1.0
// traces: ARCH-061, REQ-014
func TestLearnDefaultSourceConfidenceIsOne(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "No source specified learning",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Check confidence in database
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var confidence float64
	err = db.QueryRow("SELECT confidence FROM embeddings ORDER BY id DESC LIMIT 1").Scan(&confidence)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(confidence).To(BeNumerically("==", 1.0))
}

// ============================================================================
// TASK-42: External knowledge capture - schema columns
// ============================================================================

// TEST-906: Embeddings table has source_type column
// traces: ARCH-061, REQ-014
func TestEmbeddingsSchemaHasSourceTypeColumn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn something to create the database
	opts := memory.LearnOpts{
		Message:    "Schema test learning",
		Source:     "internal",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Open the database and check for source_type column
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Query source_type column - this will fail if column doesn't exist
	var sourceType string
	err = db.QueryRow("SELECT source_type FROM embeddings LIMIT 1").Scan(&sourceType)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sourceType).To(Equal("internal"))
}

// TEST-907: Embeddings table has confidence column
// traces: ARCH-061, REQ-014
func TestEmbeddingsSchemaHasConfidenceColumn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn something to create the database
	opts := memory.LearnOpts{
		Message:    "Confidence column test",
		Source:     "internal",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Open the database and check for confidence column
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Query confidence column - this will fail if column doesn't exist
	var confidence float64
	err = db.QueryRow("SELECT confidence FROM embeddings LIMIT 1").Scan(&confidence)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-908: source_type defaults to "internal" in schema
// traces: ARCH-061, REQ-014
func TestEmbeddingsSchemaSourceTypeDefaultsToInternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn without specifying source to test schema default
	opts := memory.LearnOpts{
		Message:    "Default test",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var sourceType string
	err = db.QueryRow("SELECT source_type FROM embeddings LIMIT 1").Scan(&sourceType)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(sourceType).To(Equal("internal"))
}

// TEST-909: confidence defaults to 1.0 in schema
// traces: ARCH-061, REQ-014
func TestEmbeddingsSchemaConfidenceDefaultsToOne(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn without specifying source to test schema default
	opts := memory.LearnOpts{
		Message:    "Default confidence test",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var confidence float64
	err = db.QueryRow("SELECT confidence FROM embeddings LIMIT 1").Scan(&confidence)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(confidence).To(BeNumerically("==", 1.0))
}

// ============================================================================
// TASK-42: External knowledge capture - source_type storage and retrieval
// ============================================================================

// TEST-910: Source type is stored and retrievable for internal memories
// traces: ARCH-061, REQ-014
func TestSourceTypeStoredAndRetrievableInternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "Internal knowledge about Go patterns",
		Source:     "internal",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Query and verify source_type comes back in results
	queryOpts := memory.QueryOpts{
		Text:       "Go patterns",
		MemoryRoot: memoryDir,
		Limit:      1,
	}

	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// The result should have source_type information available
	// This verifies the full round-trip: store with source -> query -> source in results
	topResult := results.Results[0]
	g.Expect(topResult.SourceType).To(Equal("internal"))
}

// TEST-911: Source type is stored and retrievable for external memories
// traces: ARCH-061, REQ-014
func TestSourceTypeStoredAndRetrievableExternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	opts := memory.LearnOpts{
		Message:    "External knowledge from web search about Go patterns",
		Source:     "external",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Query and verify source_type comes back in results
	queryOpts := memory.QueryOpts{
		Text:       "Go patterns",
		MemoryRoot: memoryDir,
		Limit:      1,
	}

	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	topResult := results.Results[0]
	g.Expect(topResult.SourceType).To(Equal("external"))
}

// TEST-912: Confidence is stored and retrievable
// traces: ARCH-061, REQ-014
func TestConfidenceStoredAndRetrievable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Store an external memory (confidence 0.7)
	opts := memory.LearnOpts{
		Message:    "External learning about database indexing strategies",
		Source:     "external",
		MemoryRoot: memoryDir,
	}

	err := memory.Learn(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Query and verify confidence comes back in results
	queryOpts := memory.QueryOpts{
		Text:       "database indexing",
		MemoryRoot: memoryDir,
		Limit:      1,
	}

	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	topResult := results.Results[0]
	g.Expect(topResult.Confidence).To(BeNumerically("~", 0.7, 0.001))
}

// TEST-913: Internal and external memories can coexist
// traces: ARCH-061, REQ-014
func TestInternalAndExternalMemoriesCoexist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Store internal memory
	err := memory.Learn(memory.LearnOpts{
		Message:    "Internal: database design best practices from team",
		Source:     "internal",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store external memory
	err = memory.Learn(memory.LearnOpts{
		Message:    "External: database design patterns from PostgreSQL docs",
		Source:     "external",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query both
	queryOpts := memory.QueryOpts{
		Text:       "database design",
		MemoryRoot: memoryDir,
		Limit:      10,
	}

	results, err := memory.Query(queryOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).To(HaveLen(2))

	// Verify we have both source types
	sourceTypes := make(map[string]bool)
	for _, r := range results.Results {
		sourceTypes[r.SourceType] = true
	}
	g.Expect(sourceTypes).To(HaveKey("internal"))
	g.Expect(sourceTypes).To(HaveKey("external"))
}

// ============================================================================
// TASK-42: Property-based tests
// ============================================================================

// TEST-914: Property: source type is always "internal" or "external" in stored data
// traces: ARCH-061, REQ-014
func TestPropertySourceTypeAlwaysValid(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "external-test-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		// Draw a random source value from the valid set
		source := rapid.SampledFrom([]string{"internal", "external", ""}).Draw(t, "source")

		message := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "message")
		if message == "" {
			message = "default message"
		}

		opts := memory.LearnOpts{
			Message:    message,
			Source:     source,
			MemoryRoot: memoryDir,
		}

		err := memory.Learn(opts)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify source_type is stored as either "internal" or "external"
		dbPath := filepath.Join(memoryDir, "embeddings.db")
		db, err := sql.Open("sqlite3", dbPath)
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = db.Close() }()

		var sourceType string
		err = db.QueryRow("SELECT source_type FROM embeddings ORDER BY id DESC LIMIT 1").Scan(&sourceType)
		g.Expect(err).ToNot(HaveOccurred())

		// Property: source_type is always one of the valid values
		g.Expect(sourceType).To(BeElementOf("internal", "external"))
	})
}

// TEST-915: Property: confidence is always between 0 and 1
// traces: ARCH-061, REQ-014
func TestPropertyConfidenceAlwaysInRange(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "confidence-test-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		source := rapid.SampledFrom([]string{"internal", "external", ""}).Draw(t, "source")

		message := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "message")
		if message == "" {
			message = "default message"
		}

		opts := memory.LearnOpts{
			Message:    message,
			Source:     source,
			MemoryRoot: memoryDir,
		}

		err := memory.Learn(opts)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify confidence is in valid range
		dbPath := filepath.Join(memoryDir, "embeddings.db")
		db, err := sql.Open("sqlite3", dbPath)
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = db.Close() }()

		var confidence float64
		err = db.QueryRow("SELECT confidence FROM embeddings ORDER BY id DESC LIMIT 1").Scan(&confidence)
		g.Expect(err).ToNot(HaveOccurred())

		// Property: confidence is always in [0, 1]
		g.Expect(confidence).To(BeNumerically(">=", 0.0))
		g.Expect(confidence).To(BeNumerically("<=", 1.0))
	})
}

// TEST-916: Property: external source always gets confidence 0.7, internal always gets 1.0
// traces: ARCH-061, REQ-014
func TestPropertyConfidenceMatchesSource(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "conf-match-test-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		source := rapid.SampledFrom([]string{"internal", "external"}).Draw(t, "source")

		message := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "message")
		if message == "" {
			message = "default message"
		}

		opts := memory.LearnOpts{
			Message:    message,
			Source:     source,
			MemoryRoot: memoryDir,
		}

		err := memory.Learn(opts)
		g.Expect(err).ToNot(HaveOccurred())

		dbPath := filepath.Join(memoryDir, "embeddings.db")
		db, err := sql.Open("sqlite3", dbPath)
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = db.Close() }()

		var confidence float64
		err = db.QueryRow("SELECT confidence FROM embeddings ORDER BY id DESC LIMIT 1").Scan(&confidence)
		g.Expect(err).ToNot(HaveOccurred())

		// Property: confidence matches source type
		switch source {
		case "internal":
			g.Expect(confidence).To(BeNumerically("==", 1.0))
		case "external":
			g.Expect(confidence).To(BeNumerically("~", 0.7, 0.001))
		}
	})
}
