//go:build integration

package store_test

// Tests for ARCH-1: Memory Storage (SQLite + FTS5)
// Integration tests — real SQLite, no mocks.
// Won't compile yet — RED phase.

import (
	"context"
	"sync"
	"testing"
	"time"

	"engram/internal"
	"engram/internal/store"
	"github.com/onsi/gomega"
)

var testTime = time.Date(2026, 2, 27, 16, 30, 0, 0, time.UTC)

func newTestStore(t *testing.T) store.MemoryStore {
	t.Helper()
	s, err := store.NewSQLite(t.TempDir() + "/test.db")
	gomega.NewWithT(t).Expect(err).ToNot(gomega.HaveOccurred())
	t.Cleanup(func() { s.Close() })
	return s
}

func testMemory(id, title string) *internal.Memory {
	return &internal.Memory{
		ID:              id,
		Title:           title,
		Content:         "Test content for " + title,
		ObservationType: "correction",
		Concepts:        []string{"testing", "go"},
		Principle:       "test principle",
		AntiPattern:     "test anti-pattern",
		Rationale:       "test rationale",
		EnrichedContent: "enriched test content",
		Keywords:        []string{"test", "keyword"},
		Confidence:      "A",
		ImpactScore:     0.5,
		CreatedAt:       testTime,
		UpdatedAt:       testTime,
	}
}

// T-1: Every created memory has all 6 metadata fields populated and retrievable.
func TestMemoryStore_CreatePopulatesAllMetadataFields(t *testing.T) {
	g := gomega.NewWithT(t)
	s := newTestStore(t)
	ctx := context.Background()

	m := testMemory("m_0001", "Test memory")
	err := s.Create(ctx, m)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	got, err := s.Get(ctx, "m_0001")
	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Expect(got.ObservationType).To(gomega.Equal("correction"))
	g.Expect(got.Concepts).To(gomega.Equal([]string{"testing", "go"}))
	g.Expect(got.Principle).To(gomega.Equal("test principle"))
	g.Expect(got.AntiPattern).To(gomega.Equal("test anti-pattern"))
	g.Expect(got.Rationale).To(gomega.Equal("test rationale"))
	g.Expect(got.EnrichedContent).To(gomega.Equal("enriched test content"))
	g.Expect(got.Keywords).To(gomega.Equal([]string{"test", "keyword"}))
}

// T-2: Creating a memory without a confidence tier (A/B/C) fails.
func TestMemoryStore_ConfidenceTierIsRequired(t *testing.T) {
	g := gomega.NewWithT(t)
	s := newTestStore(t)
	ctx := context.Background()

	m := testMemory("m_0001", "No confidence")
	m.Confidence = ""

	err := s.Create(ctx, m)
	g.Expect(err).To(gomega.HaveOccurred())
}

// T-3: FindSimilar returns results with BM25 scores, ordered highest first.
func TestMemoryStore_FindSimilarReturnsScoredCandidates(t *testing.T) {
	g := gomega.NewWithT(t)
	s := newTestStore(t)
	ctx := context.Background()

	m1 := testMemory("m_0001", "Use git add specific files")
	m1.Content = "Always use git add with specific file paths, never git add -A"
	m1.Keywords = []string{"git", "staging", "add", "specific-files"}

	m2 := testMemory("m_0002", "DI pattern in internal")
	m2.Content = "All file access through injected interfaces"
	m2.Keywords = []string{"dependency-injection", "interfaces", "internal"}

	g.Expect(s.Create(ctx, m1)).To(gomega.Succeed())
	g.Expect(s.Create(ctx, m2)).To(gomega.Succeed())

	results, err := s.FindSimilar(ctx, "git staging add files", 5)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(results).ToNot(gomega.BeEmpty())

	// First result should be the git-related memory
	g.Expect(results[0].Memory.ID).To(gomega.Equal("m_0001"))

	// Scores should be descending
	for i := 1; i < len(results); i++ {
		g.Expect(results[i].Score).To(gomega.BeNumerically("<=", results[i-1].Score))
	}
}

// T-4: FindSimilar never returns more than K results.
func TestMemoryStore_FindSimilarRespectsK(t *testing.T) {
	g := gomega.NewWithT(t)
	s := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		m := testMemory("m_"+string(rune('a'+i)), "Testing memory")
		m.Content = "This is about testing and go test commands"
		m.Keywords = []string{"testing", "go", "test"}
		g.Expect(s.Create(ctx, m)).To(gomega.Succeed())
	}

	results, err := s.FindSimilar(ctx, "testing go", 2)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(len(results)).To(gomega.BeNumerically("<=", 2))
}

// T-5: Updating a memory changes updated_at but preserves created_at.
func TestMemoryStore_UpdatePreservesCreatedAt(t *testing.T) {
	g := gomega.NewWithT(t)
	s := newTestStore(t)
	ctx := context.Background()

	m := testMemory("m_0001", "Original")
	g.Expect(s.Create(ctx, m)).To(gomega.Succeed())

	m.Title = "Updated"
	m.UpdatedAt = testTime.Add(time.Hour)
	g.Expect(s.Update(ctx, m)).To(gomega.Succeed())

	got, err := s.Get(ctx, "m_0001")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(got.Title).To(gomega.Equal("Updated"))
	g.Expect(got.CreatedAt).To(gomega.BeTemporally("==", testTime))
	g.Expect(got.UpdatedAt).To(gomega.BeTemporally("==", testTime.Add(time.Hour)))
}

// T-6: Enriching a memory increments its enrichment_count by 1.
func TestMemoryStore_EnrichmentIncrementsCount(t *testing.T) {
	g := gomega.NewWithT(t)
	s := newTestStore(t)
	ctx := context.Background()

	m := testMemory("m_0001", "Original")
	m.EnrichmentCount = 0
	g.Expect(s.Create(ctx, m)).To(gomega.Succeed())

	m.EnrichmentCount = 1
	m.UpdatedAt = testTime.Add(time.Hour)
	g.Expect(s.Update(ctx, m)).To(gomega.Succeed())

	got, err := s.Get(ctx, "m_0001")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(got.EnrichmentCount).To(gomega.Equal(1))
}

// T-7: Memories whose keywords contain query terms rank above content-only matches.
func TestMemoryStore_FindSimilarRanksKeywordMatchesHigher(t *testing.T) {
	g := gomega.NewWithT(t)
	s := newTestStore(t)
	ctx := context.Background()

	m1 := testMemory("m_kw", "Use targ build system")
	m1.Content = "This project uses a build tool"
	m1.Keywords = []string{"targ", "build", "test"}

	m2 := testMemory("m_ct", "Build tool note")
	m2.Content = "Remember to use targ for building and testing"
	m2.Keywords = []string{"build", "tool"}

	g.Expect(s.Create(ctx, m1)).To(gomega.Succeed())
	g.Expect(s.Create(ctx, m2)).To(gomega.Succeed())

	results, err := s.FindSimilar(ctx, "targ", 5)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(len(results)).To(gomega.BeNumerically(">=", 2))
	g.Expect(results[0].Memory.ID).To(gomega.Equal("m_kw"))
}

// T-8: FTS5 index syncs with table on insert, update, and delete.
func TestMemoryStore_FTS5IndexSyncsWithTable(t *testing.T) {
	g := gomega.NewWithT(t)
	s := newTestStore(t)
	ctx := context.Background()

	m := testMemory("m_0001", "Findable memory")
	m.Keywords = []string{"unique-keyword-xyz"}
	g.Expect(s.Create(ctx, m)).To(gomega.Succeed())

	// Should be findable after insert
	results, err := s.FindSimilar(ctx, "unique-keyword-xyz", 5)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(results).ToNot(gomega.BeEmpty())

	// Update keywords
	m.Keywords = []string{"different-keyword-abc"}
	m.UpdatedAt = testTime.Add(time.Hour)
	g.Expect(s.Update(ctx, m)).To(gomega.Succeed())

	// Old keyword should not match
	results, err = s.FindSimilar(ctx, "unique-keyword-xyz", 5)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(results).To(gomega.BeEmpty())

	// New keyword should match
	results, err = s.FindSimilar(ctx, "different-keyword-abc", 5)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(results).ToNot(gomega.BeEmpty())
}

// T-9: Multiple concurrent FindSimilar calls don't conflict.
func TestMemoryStore_ConcurrentReads(t *testing.T) {
	g := gomega.NewWithT(t)
	s := newTestStore(t)
	ctx := context.Background()

	m := testMemory("m_0001", "Concurrent test")
	m.Keywords = []string{"concurrent", "test"}
	g.Expect(s.Create(ctx, m)).To(gomega.Succeed())

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.FindSimilar(ctx, "concurrent test", 5)
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		g.Expect(err).ToNot(gomega.HaveOccurred())
	}
}
