//go:build integration

package memory_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// TEST: Concepts are stored as comma-joined string
func TestLearnConceptsCommaJoined(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	obs := &memory.Observation{
		Type:        "pattern",
		Concepts:    []string{"golang", "concurrency", "channels"},
		Principle:   "Use channels for synchronization",
		AntiPattern: "Sharing memory without sync primitives",
		Rationale:   "Go idiom: communicate by sharing",
	}

	err := memory.Learn(memory.LearnOpts{
		Message:    "concepts comma join verification unique2345",
		MemoryRoot: memoryDir,
		Extractor:  mockExtractor(obs),
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, concepts, _, _, _, _ := readObservationColumns(g, memoryDir, "unique2345")
	g.Expect(concepts).To(Equal("golang,concurrency,channels"))
}

// TEST: Enriched content format is "[type] principle - Context: rationale"
func TestLearnEnrichedContentFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	obs := &memory.Observation{
		Type:        "correction",
		Concepts:    []string{"git"},
		Principle:   "Never force push to main",
		AntiPattern: "git push --force origin main",
		Rationale:   "Destroys shared history",
	}

	err := memory.Learn(memory.LearnOpts{
		Message:    "enriched format verification unique7890",
		MemoryRoot: memoryDir,
		Extractor:  mockExtractor(obs),
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, _, _, _, _, enrichedContent := readObservationColumns(g, memoryDir, "unique7890")
	g.Expect(enrichedContent).To(Equal("[correction] Never force push to main - Context: Destroys shared history"))
}

// TEST: Enriched content is used for embedding when extractor succeeds
func TestLearnEnrichedContentUsedForEmbedding(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	obs := &memory.Observation{
		Type:        "correction",
		Concepts:    []string{"testing"},
		Principle:   "Always write tests first",
		AntiPattern: "Skipping tests",
		Rationale:   "TDD prevents regressions",
	}

	err := memory.Learn(memory.LearnOpts{
		Message:    "embedding enrichment verification unique6789",
		MemoryRoot: memoryDir,
		Extractor:  mockExtractor(obs),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// The enriched_content should be stored and non-empty
	_, _, _, _, _, enrichedContent := readObservationColumns(g, memoryDir, "unique6789")
	g.Expect(enrichedContent).ToNot(BeEmpty(),
		"enriched_content should be populated for embedding generation")
	g.Expect(enrichedContent).To(ContainSubstring("Always write tests first"),
		"enriched_content should contain the principle")
}

// TEST: Observation with empty concepts produces empty concepts column
func TestLearnExtractorEmptyConcepts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	obs := &memory.Observation{
		Type:      "discovery",
		Concepts:  []string{},
		Principle: "Some principle",
		Rationale: "Some rationale",
	}

	err := memory.Learn(memory.LearnOpts{
		Message:    "empty concepts verification unique0123",
		MemoryRoot: memoryDir,
		Extractor:  mockExtractor(obs),
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, concepts, _, _, _, _ := readObservationColumns(g, memoryDir, "unique0123")
	g.Expect(concepts).To(Equal(""), "empty concepts slice should produce empty string")
}

// TEST: Learn with ErrLLMUnavailable falls back gracefully
func TestLearnWithErrLLMUnavailableFallsBack(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	err := memory.Learn(memory.LearnOpts{
		Message:    "llm unavailable fallback unique3456",
		MemoryRoot: memoryDir,
		Extractor:  failingExtractor(memory.ErrLLMUnavailable),
	})
	g.Expect(err).ToNot(HaveOccurred(), "Learn should not fail when LLM is unavailable")

	// Observation columns should be empty defaults
	obsType, _, _, _, _, _ := readObservationColumns(g, memoryDir, "unique3456")
	g.Expect(obsType).To(Equal(""))
}

// TEST: Learn with mock extractor populates observation columns
func TestLearnWithExtractorPopulatesObservationColumns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	obs := &memory.Observation{
		Type:        "pattern",
		Concepts:    []string{"go", "testing"},
		Principle:   "Always use TDD for Go code",
		AntiPattern: "Writing code without tests",
		Rationale:   "TDD catches regressions early and improves design",
	}

	err := memory.Learn(memory.LearnOpts{
		Message:    "llm extraction test unique1234",
		MemoryRoot: memoryDir,
		Extractor:  mockExtractor(obs),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify observation columns in DB
	obsType, concepts, principle, antiPattern, rationale, enrichedContent := readObservationColumns(g, memoryDir, "unique1234")
	g.Expect(obsType).To(Equal("pattern"))
	g.Expect(concepts).To(Equal("go,testing"))
	g.Expect(principle).To(Equal("Always use TDD for Go code"))
	g.Expect(antiPattern).To(Equal("Writing code without tests"))
	g.Expect(rationale).To(Equal("TDD catches regressions early and improves design"))
	g.Expect(enrichedContent).ToNot(BeEmpty(), "enriched_content should be built from observation")
}

// TEST: Learn with extractor stores to embeddings DB
func TestLearnWithExtractorStoresToDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	obs := &memory.Observation{
		Type:      "pattern",
		Concepts:  []string{"test"},
		Principle: "test",
		Rationale: "test",
	}

	err := memory.Learn(memory.LearnOpts{
		Message:    "index write verification unique4567",
		MemoryRoot: memoryDir,
		Extractor:  mockExtractor(obs),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify entry is in embeddings DB via Grep
	grepResult, err := memory.Grep(memory.GrepOpts{Pattern: "unique4567", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(grepResult.Matches).ToNot(BeEmpty(), "entry should be stored in DB")
}

// TEST: Learn with failing extractor falls back gracefully (no error returned)
func TestLearnWithFailingExtractorFallsBack(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	err := memory.Learn(memory.LearnOpts{
		Message:    "failing extractor fallback unique9012",
		MemoryRoot: memoryDir,
		Extractor:  failingExtractor(errors.New("connection refused")),
	})
	g.Expect(err).ToNot(HaveOccurred(), "Learn should not fail due to extractor errors")

	// Observation columns should be empty defaults (graceful fallback)
	obsType, concepts, principle, antiPattern, rationale, enrichedContent := readObservationColumns(g, memoryDir, "unique9012")
	g.Expect(obsType).To(Equal(""))
	g.Expect(concepts).To(Equal(""))
	g.Expect(principle).To(Equal(""))
	g.Expect(antiPattern).To(Equal(""))
	g.Expect(rationale).To(Equal(""))
	g.Expect(enrichedContent).To(Equal(""))
}

// TEST: Learn with nil extractor behaves as before (backward compat)
func TestLearnWithNilExtractorBackwardCompat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	err := memory.Learn(memory.LearnOpts{
		Message:    "nil extractor backward compat unique5678",
		MemoryRoot: memoryDir,
		// Extractor is nil — default behavior
	})
	g.Expect(err).ToNot(HaveOccurred())

	// All observation columns should be empty defaults
	obsType, concepts, principle, antiPattern, rationale, enrichedContent := readObservationColumns(g, memoryDir, "unique5678")
	g.Expect(obsType).To(Equal(""))
	g.Expect(concepts).To(Equal(""))
	g.Expect(principle).To(Equal(""))
	g.Expect(antiPattern).To(Equal(""))
	g.Expect(rationale).To(Equal(""))
	g.Expect(enrichedContent).To(Equal(""))
}

// TEST: Property - Learn never fails due to extractor errors
func TestPropertyLearnNeverFailsDueToExtractorErrors(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		tempDir := os.TempDir()
		memoryDir, err := os.MkdirTemp(tempDir, "learn-extract-prop-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = os.RemoveAll(memoryDir) }()

		errMsg := rapid.StringMatching(`[a-zA-Z ]{1,50}`).Draw(rt, "errorMessage")

		err = memory.Learn(memory.LearnOpts{
			Message:    "property test entry " + errMsg,
			MemoryRoot: memoryDir,
			Extractor:  failingExtractor(errors.New(errMsg)),
		})
		g.Expect(err).ToNot(HaveOccurred(),
			"Learn must never fail due to extractor errors, got: %v", err)
	})
}

// TEST: Property - Learn with extractor always populates observation_type non-empty
func TestPropertyLearnWithExtractorPopulatesObservationType(t *testing.T) {
	t.Parallel()
	validTypes := []string{"correction", "pattern", "decision", "discovery"}

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		tempDir := os.TempDir()
		memoryDir, err := os.MkdirTemp(tempDir, "learn-obstype-prop-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = os.RemoveAll(memoryDir) }()

		obsType := rapid.SampledFrom(validTypes).Draw(rt, "type")
		suffix := rapid.StringMatching(`[a-z]{8}`).Draw(rt, "suffix")

		obs := &memory.Observation{
			Type:      obsType,
			Concepts:  []string{"test"},
			Principle: "test principle",
			Rationale: "test rationale",
		}

		err = memory.Learn(memory.LearnOpts{
			Message:    "prop obs type " + suffix,
			MemoryRoot: memoryDir,
			Extractor:  mockExtractor(obs),
		})
		g.Expect(err).ToNot(HaveOccurred())

		storedType, _, _, _, _, _ := readObservationColumns(g, memoryDir, suffix)
		g.Expect(storedType).To(Equal(obsType),
			"observation_type should match extracted type")
	})
}

// failingExtractor returns an error for any input.
func failingExtractor(err error) memory.LLMExtractor {
	return &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, err
		},
	}
}

// ============================================================================
// ISSUE-188 Task 5: Learn() with LLM extraction
// ============================================================================

// mockExtractor returns a fixed Observation for any input.
func mockExtractor(obs *memory.Observation) memory.LLMExtractor {
	return &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			jsonBytes, _ := json.Marshal(obs)
			return jsonBytes, nil
		},
	}
}

// readObservationColumns reads observation-related columns for a row matching contentSubstr.
func readObservationColumns(g Gomega, memoryRoot, contentSubstr string) (obsType, concepts, principle, antiPattern, rationale, enrichedContent string) {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	err = db.QueryRow(
		`SELECT observation_type, concepts, principle, anti_pattern, rationale, enriched_content
		 FROM embeddings WHERE content LIKE ?`,
		"%"+contentSubstr+"%",
	).Scan(&obsType, &concepts, &principle, &antiPattern, &rationale, &enrichedContent)
	g.Expect(err).ToNot(HaveOccurred())
	return
}
