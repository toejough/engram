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
// Schema tests: retrieval_count, last_retrieved, projects_retrieved columns
// traces: ARCH-060, REQ-013
// ============================================================================

// TEST-950: Embeddings schema includes retrieval_count column
// traces: ARCH-060, REQ-013
func TestEmbeddingsSchemaHasRetrievalCount(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Write content so Query creates the DB
	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: schema test entry"), 0644)).To(Succeed())

	opts := memory.QueryOpts{
		Text:       "schema test",
		MemoryRoot: memoryDir,
	}
	_, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Open DB and verify column exists
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('embeddings') WHERE name = 'retrieval_count'`).Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(1), "embeddings table should have retrieval_count column")
}

// TEST-951: Embeddings schema includes last_retrieved column
// traces: ARCH-060, REQ-013
func TestEmbeddingsSchemaHasLastRetrieved(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: last retrieved test"), 0644)).To(Succeed())

	opts := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
	}
	_, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('embeddings') WHERE name = 'last_retrieved'`).Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(1), "embeddings table should have last_retrieved column")
}

// TEST-952: Embeddings schema includes projects_retrieved column
// traces: ARCH-060, REQ-013
func TestEmbeddingsSchemaHasProjectsRetrieved(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: projects retrieved test"), 0644)).To(Succeed())

	opts := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
	}
	_, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('embeddings') WHERE name = 'projects_retrieved'`).Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(1), "embeddings table should have projects_retrieved column")
}

// ============================================================================
// searchSimilar counter increment tests
// traces: ARCH-060, REQ-013
// ============================================================================

// TEST-953: searchSimilar increments retrieval_count after query
// traces: ARCH-060, REQ-013
func TestSearchSimilarIncrementsRetrievalCount(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: retrieval counting test"), 0644)).To(Succeed())

	opts := memory.QueryOpts{
		Text:       "retrieval counting",
		MemoryRoot: memoryDir,
	}

	// First query
	_, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Second query (should increment retrieval_count for matching results)
	_, err = memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Check the retrieval_count in the database
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var retrievalCount int
	err = db.QueryRow(`SELECT retrieval_count FROM embeddings WHERE content LIKE '%retrieval counting%' LIMIT 1`).Scan(&retrievalCount)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(retrievalCount).To(BeNumerically(">=", 2), "retrieval_count should be incremented after each search")
}

// TEST-954: searchSimilar updates last_retrieved timestamp
// traces: ARCH-060, REQ-013
func TestSearchSimilarUpdatesLastRetrieved(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: timestamp update test"), 0644)).To(Succeed())

	opts := memory.QueryOpts{
		Text:       "timestamp update",
		MemoryRoot: memoryDir,
	}

	_, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var lastRetrieved sql.NullString
	err = db.QueryRow(`SELECT last_retrieved FROM embeddings WHERE content LIKE '%timestamp update%' LIMIT 1`).Scan(&lastRetrieved)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(lastRetrieved.Valid).To(BeTrue(), "last_retrieved should be set after search")
	g.Expect(lastRetrieved.String).ToNot(BeEmpty(), "last_retrieved should have a timestamp value")
}

// TEST-955: searchSimilar tracks project in projects_retrieved
// traces: ARCH-060, REQ-013
func TestSearchSimilarTracksProject(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: project tracking test"), 0644)).To(Succeed())

	// Query with project context
	opts := memory.QueryOpts{
		Text:       "project tracking",
		MemoryRoot: memoryDir,
		Project:    "project-alpha",
	}

	_, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var projectsRetrieved string
	err = db.QueryRow(`SELECT projects_retrieved FROM embeddings WHERE content LIKE '%project tracking%' LIMIT 1`).Scan(&projectsRetrieved)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(projectsRetrieved).To(ContainSubstring("project-alpha"))
}

// TEST-956: searchSimilar deduplicates projects in projects_retrieved
// traces: ARCH-060, REQ-013
func TestSearchSimilarDeduplicatesProjects(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: dedup projects test"), 0644)).To(Succeed())

	// Query twice with same project
	opts := memory.QueryOpts{
		Text:       "dedup projects",
		MemoryRoot: memoryDir,
		Project:    "same-project",
	}

	_, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var projectsRetrieved string
	err = db.QueryRow(`SELECT projects_retrieved FROM embeddings WHERE content LIKE '%dedup projects%' LIMIT 1`).Scan(&projectsRetrieved)
	g.Expect(err).ToNot(HaveOccurred())

	// "same-project" should appear exactly once even after two queries
	g.Expect(projectsRetrieved).To(Equal("same-project"),
		"project should not be duplicated in projects_retrieved")
}

// TEST-957: searchSimilar accumulates distinct projects
// traces: ARCH-060, REQ-013
func TestSearchSimilarAccumulatesDistinctProjects(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: multi project test"), 0644)).To(Succeed())

	// Query from two different projects
	opts1 := memory.QueryOpts{
		Text:       "multi project",
		MemoryRoot: memoryDir,
		Project:    "project-one",
	}
	_, err := memory.Query(opts1)
	g.Expect(err).ToNot(HaveOccurred())

	opts2 := memory.QueryOpts{
		Text:       "multi project",
		MemoryRoot: memoryDir,
		Project:    "project-two",
	}
	_, err = memory.Query(opts2)
	g.Expect(err).ToNot(HaveOccurred())

	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var projectsRetrieved string
	err = db.QueryRow(`SELECT projects_retrieved FROM embeddings WHERE content LIKE '%multi project%' LIMIT 1`).Scan(&projectsRetrieved)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(projectsRetrieved).To(ContainSubstring("project-one"))
	g.Expect(projectsRetrieved).To(ContainSubstring("project-two"))
}

// ============================================================================
// QueryOpts.Project field tests
// traces: ARCH-060, REQ-013
// ============================================================================

// TEST-958: QueryOpts accepts Project field
// traces: ARCH-060, REQ-013
func TestQueryOptsHasProjectField(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: project field test"), 0644)).To(Succeed())

	// This should compile and work with the Project field
	opts := memory.QueryOpts{
		Text:       "project field",
		MemoryRoot: memoryDir,
		Project:    "test-project",
	}

	_, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
}

// ============================================================================
// Promote function tests
// traces: ARCH-060, REQ-013
// ============================================================================

// TEST-960: Promote returns candidates meeting default thresholds
// traces: ARCH-060, REQ-013
func TestPromoteReturnsQualifyingCandidates(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: high value learning"), 0644)).To(Succeed())

	// Seed the database: query from multiple projects multiple times
	// to build up retrieval_count and projects_retrieved
	projects := []string{"proj-a", "proj-b", "proj-c"}
	for _, proj := range projects {
		opts := memory.QueryOpts{
			Text:       "high value",
			MemoryRoot: memoryDir,
			Project:    proj,
		}
		_, err := memory.Query(opts)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Now promote should find this candidate (3 retrievals, 3 projects)
	promoteOpts := memory.PromoteOpts{
		MemoryRoot: memoryDir,
	}

	result, err := memory.Promote(promoteOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Candidates).ToNot(BeEmpty(),
		"should find candidates with >=3 retrievals and >=2 unique projects")
}

// TEST-961: Promote uses default MinRetrievals of 3
// traces: ARCH-060, REQ-013
func TestPromoteDefaultMinRetrievals(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: below threshold entry"), 0644)).To(Succeed())

	// Only query twice (below default threshold of 3), from 2 projects
	for _, proj := range []string{"proj-a", "proj-b"} {
		opts := memory.QueryOpts{
			Text:       "below threshold",
			MemoryRoot: memoryDir,
			Project:    proj,
		}
		_, err := memory.Query(opts)
		g.Expect(err).ToNot(HaveOccurred())
	}

	promoteOpts := memory.PromoteOpts{
		MemoryRoot: memoryDir,
	}

	result, err := memory.Promote(promoteOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Candidates).To(BeEmpty(),
		"should NOT find candidates with only 2 retrievals (below default 3)")
}

// TEST-962: Promote uses default MinProjects of 2
// traces: ARCH-060, REQ-013
func TestPromoteDefaultMinProjects(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: single project entry"), 0644)).To(Succeed())

	// Query 5 times but always from the same project
	for i := 0; i < 5; i++ {
		opts := memory.QueryOpts{
			Text:       "single project",
			MemoryRoot: memoryDir,
			Project:    "only-project",
		}
		_, err := memory.Query(opts)
		g.Expect(err).ToNot(HaveOccurred())
	}

	promoteOpts := memory.PromoteOpts{
		MemoryRoot: memoryDir,
	}

	result, err := memory.Promote(promoteOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Candidates).To(BeEmpty(),
		"should NOT find candidates with only 1 unique project (below default 2)")
}

// TEST-963: Promote respects custom MinRetrievals
// traces: ARCH-060, REQ-013
func TestPromoteCustomMinRetrievals(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: custom threshold entry"), 0644)).To(Succeed())

	// Query twice from 2 different projects
	for _, proj := range []string{"proj-x", "proj-y"} {
		opts := memory.QueryOpts{
			Text:       "custom threshold",
			MemoryRoot: memoryDir,
			Project:    proj,
		}
		_, err := memory.Query(opts)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Set MinRetrievals to 1 (lower than default 3)
	promoteOpts := memory.PromoteOpts{
		MemoryRoot:    memoryDir,
		MinRetrievals: 1,
	}

	result, err := memory.Promote(promoteOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Candidates).ToNot(BeEmpty(),
		"should find candidates with custom MinRetrievals=1")
}

// TEST-964: Promote respects custom MinProjects
// traces: ARCH-060, REQ-013
func TestPromoteCustomMinProjects(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: custom projects threshold"), 0644)).To(Succeed())

	// Query 3 times from 1 project
	for i := 0; i < 3; i++ {
		opts := memory.QueryOpts{
			Text:       "custom projects",
			MemoryRoot: memoryDir,
			Project:    "solo-project",
		}
		_, err := memory.Query(opts)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Set MinProjects to 1 (lower than default 2)
	promoteOpts := memory.PromoteOpts{
		MemoryRoot:  memoryDir,
		MinProjects: 1,
	}

	result, err := memory.Promote(promoteOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Candidates).ToNot(BeEmpty(),
		"should find candidates with custom MinProjects=1")
}

// TEST-965: PromoteResult includes content and metadata
// traces: ARCH-060, REQ-013
func TestPromoteResultIncludesMetadata(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: metadata candidate entry"), 0644)).To(Succeed())

	// Build up sufficient retrievals from multiple projects
	for _, proj := range []string{"proj-m", "proj-n", "proj-o"} {
		opts := memory.QueryOpts{
			Text:       "metadata candidate",
			MemoryRoot: memoryDir,
			Project:    proj,
		}
		_, err := memory.Query(opts)
		g.Expect(err).ToNot(HaveOccurred())
	}

	promoteOpts := memory.PromoteOpts{
		MemoryRoot: memoryDir,
	}

	result, err := memory.Promote(promoteOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Candidates).ToNot(BeEmpty())

	candidate := result.Candidates[0]
	g.Expect(candidate.Content).ToNot(BeEmpty(), "candidate should have content")
	g.Expect(candidate.RetrievalCount).To(BeNumerically(">=", 3))
	g.Expect(candidate.UniqueProjects).To(BeNumerically(">=", 2))
}

// TEST-966: Promote returns empty when no candidates qualify
// traces: ARCH-060, REQ-013
func TestPromoteEmptyWhenNoCandidates(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Create DB but don't query anything, so no retrieval counts
	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: never queried entry"), 0644)).To(Succeed())

	// Query once from one project (below both thresholds)
	opts := memory.QueryOpts{
		Text:       "never queried",
		MemoryRoot: memoryDir,
		Project:    "single-proj",
	}
	_, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())

	promoteOpts := memory.PromoteOpts{
		MemoryRoot: memoryDir,
	}

	result, err := memory.Promote(promoteOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Candidates).To(BeEmpty())
}

// TEST-967: Promote requires MemoryRoot
// traces: ARCH-060, REQ-013
func TestPromoteRequiresMemoryRoot(t *testing.T) {
	g := NewWithT(t)

	promoteOpts := memory.PromoteOpts{
		MemoryRoot: "",
	}

	_, err := memory.Promote(promoteOpts)
	g.Expect(err).To(HaveOccurred())
}

// ============================================================================
// Property-based tests
// traces: ARCH-060, REQ-013
// ============================================================================

// TEST-970: Property: retrieval_count is non-negative and monotonically increasing
// traces: ARCH-060, REQ-013
func TestPropertyRetrievalCountMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "promote-prop-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

		content := rapid.StringMatching(`[a-zA-Z ]{10,30}`).Draw(t, "content")
		g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
			[]byte("- 2024-01-01: "+content), 0644)).To(Succeed())

		numQueries := rapid.IntRange(1, 5).Draw(t, "numQueries")
		project := rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "project")

		var prevCount int
		for i := 0; i < numQueries; i++ {
			opts := memory.QueryOpts{
				Text:       content,
				MemoryRoot: memoryDir,
				Project:    project,
			}
			_, err := memory.Query(opts)
			g.Expect(err).ToNot(HaveOccurred())

			// Check retrieval_count in DB
			dbPath := filepath.Join(memoryDir, "embeddings.db")
			db, err := sql.Open("sqlite3", dbPath)
			g.Expect(err).ToNot(HaveOccurred())

			var retrievalCount int
			err = db.QueryRow(`SELECT COALESCE(MAX(retrieval_count), 0) FROM embeddings`).Scan(&retrievalCount)
			_ = db.Close() // Close immediately, not deferred
			g.Expect(err).ToNot(HaveOccurred())

			// Property: retrieval_count >= previous
			g.Expect(retrievalCount).To(BeNumerically(">=", prevCount),
				"retrieval_count should be monotonically non-decreasing")
			prevCount = retrievalCount
		}
	})
}

// TEST-971: Property: Promote candidates always meet both thresholds
// traces: ARCH-060, REQ-013
func TestPropertyPromoteCandidatesMeetThresholds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "promote-thresh-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

		content := rapid.StringMatching(`[a-zA-Z ]{10,30}`).Draw(t, "content")
		g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
			[]byte("- 2024-01-01: "+content), 0644)).To(Succeed())

		minRetrievals := rapid.IntRange(1, 5).Draw(t, "minRetrievals")
		minProjects := rapid.IntRange(1, 3).Draw(t, "minProjects")

		// Do a bunch of queries from various projects
		numQueries := rapid.IntRange(1, 8).Draw(t, "numQueries")
		for i := 0; i < numQueries; i++ {
			proj := rapid.StringMatching(`[a-z]{3,6}`).Draw(t, "proj")
			opts := memory.QueryOpts{
				Text:       content,
				MemoryRoot: memoryDir,
				Project:    proj,
			}
			_, err := memory.Query(opts)
			g.Expect(err).ToNot(HaveOccurred())
		}

		promoteOpts := memory.PromoteOpts{
			MemoryRoot:    memoryDir,
			MinRetrievals: minRetrievals,
			MinProjects:   minProjects,
		}

		result, err := memory.Promote(promoteOpts)
		g.Expect(err).ToNot(HaveOccurred())

		// Property: every returned candidate meets both thresholds
		for _, c := range result.Candidates {
			g.Expect(c.RetrievalCount).To(BeNumerically(">=", minRetrievals),
				"candidate retrieval_count should meet MinRetrievals threshold")
			g.Expect(c.UniqueProjects).To(BeNumerically(">=", minProjects),
				"candidate unique projects should meet MinProjects threshold")
		}
	})
}
