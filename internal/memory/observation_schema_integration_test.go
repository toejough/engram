//go:build integration

package memory_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for observation column schema migrations (ISSUE-188)
// ============================================================================

// TEST-1400: Observation columns exist after fresh DB init
func TestObservationColumnsExistOnFreshDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Init DB via exported helper (triggers initEmbeddingsDB)
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Verify each observation column exists by querying PRAGMA
	columns := getColumnNames(g, db)
	g.Expect(columns).To(ContainElement("observation_type"))
	g.Expect(columns).To(ContainElement("concepts"))
	g.Expect(columns).To(ContainElement("principle"))
	g.Expect(columns).To(ContainElement("anti_pattern"))
	g.Expect(columns).To(ContainElement("rationale"))
	g.Expect(columns).To(ContainElement("enriched_content"))
}

// TEST-1401: Observation columns exist after double-init (migration on existing DB)
func TestObservationColumnsMigrationIdempotent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// First init
	db1, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	_ = db1.Close()

	// Second init (simulates migration on existing DB)
	db2, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db2.Close() }()

	columns := getColumnNames(g, db2)
	g.Expect(columns).To(ContainElement("observation_type"))
	g.Expect(columns).To(ContainElement("concepts"))
	g.Expect(columns).To(ContainElement("principle"))
	g.Expect(columns).To(ContainElement("anti_pattern"))
	g.Expect(columns).To(ContainElement("rationale"))
	g.Expect(columns).To(ContainElement("enriched_content"))
}

// TEST-1402: Observation columns have empty string defaults
func TestObservationColumnDefaults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Insert a row without specifying observation columns
	_, err = db.Exec("INSERT INTO embeddings (content, source) VALUES ('test content', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	// Read back the defaults
	var obsType, concepts, principle, antiPattern, rationale, enrichedContent string
	err = db.QueryRow(
		"SELECT observation_type, concepts, principle, anti_pattern, rationale, enriched_content FROM embeddings WHERE content = 'test content'",
	).Scan(&obsType, &concepts, &principle, &antiPattern, &rationale, &enrichedContent)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(obsType).To(Equal(""))
	g.Expect(concepts).To(Equal(""))
	g.Expect(principle).To(Equal(""))
	g.Expect(antiPattern).To(Equal(""))
	g.Expect(rationale).To(Equal(""))
	g.Expect(enrichedContent).To(Equal(""))
}

// TEST-1403: Insert and read back observation fields
func TestObservationFieldsRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Insert a row with observation fields populated
	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, observation_type, concepts, principle, anti_pattern, rationale, enriched_content)
		 VALUES ('test content', 'test', 'pattern', 'go,testing', 'always use TDD', 'skip tests', 'TDD catches bugs early', 'enriched: always use TDD for Go testing')`,
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Read back
	var obsType, concepts, principle, antiPattern, rationale, enrichedContent string
	err = db.QueryRow(
		"SELECT observation_type, concepts, principle, anti_pattern, rationale, enriched_content FROM embeddings WHERE content = 'test content'",
	).Scan(&obsType, &concepts, &principle, &antiPattern, &rationale, &enrichedContent)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(obsType).To(Equal("pattern"))
	g.Expect(concepts).To(Equal("go,testing"))
	g.Expect(principle).To(Equal("always use TDD"))
	g.Expect(antiPattern).To(Equal("skip tests"))
	g.Expect(rationale).To(Equal("TDD catches bugs early"))
	g.Expect(enrichedContent).To(Equal("enriched: always use TDD for Go testing"))
}

// getColumnNames returns all column names for the embeddings table.
func getColumnNames(g Gomega, db *sql.DB) []string {
	rows, err := db.Query("PRAGMA table_info(embeddings)")
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = rows.Close() }()

	var names []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue *string
		var pk int
		err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		g.Expect(err).ToNot(HaveOccurred())
		names = append(names, name)
	}
	return names
}
