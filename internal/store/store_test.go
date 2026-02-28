//go:build integration

package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"engram/internal/store"

	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"
	"pgregory.net/rapid"
)

func TestT1_CreatePopulatesAllMetadataFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Given any Memory m with all metadata fields populated
		m := genMemory().Draw(t, "memory")

		// When test calls store.Create with (any ctx, m)
		err := s.Create(ctx, &m)
		// Then store.Create returns nil error
		g.Expect(err).NotTo(HaveOccurred())

		// When test calls store.Get with (any ctx, m.ID)
		got, err := s.Get(ctx, m.ID)
		// Then store.Get returns (Memory got, nil error)
		g.Expect(err).NotTo(HaveOccurred())
		// And got matches m on all metadata fields
		g.Expect(got.ObservationType).To(Equal(m.ObservationType))
		g.Expect(got.Concepts).To(Equal(m.Concepts))
		g.Expect(got.Principle).To(Equal(m.Principle))
		g.Expect(got.AntiPattern).To(Equal(m.AntiPattern))
		g.Expect(got.Rationale).To(Equal(m.Rationale))
		g.Expect(got.EnrichedContent).To(Equal(m.EnrichedContent))
		g.Expect(got.Keywords).To(Equal(m.Keywords))
	})
}

func TestT2_CreateRejectsEmptyConfidence(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()

	// Given a Memory m with Confidence = ""
	m := store.Memory{
		ID:         "m_00000001",
		Title:      "test",
		Content:    "test content",
		Confidence: "",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	// When test calls store.Create with (any ctx, m)
	err := s.Create(ctx, &m)
	// Then store.Create returns non-nil error
	g.Expect(err).To(HaveOccurred())
}

func TestT3_FindSimilarReturnsScoredCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Given two memories in store: m1 with git content, m2 with DI content
	m1 := store.Memory{
		ID: "m_00000001", Title: "Git staging", Content: "Always use git add with specific file paths",
		Keywords: []string{"git", "staging", "add"}, Confidence: "A",
		Concepts: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	m2 := store.Memory{
		ID: "m_00000002", Title: "DI", Content: "All file access through injected interfaces",
		Keywords: []string{"dependency-injection", "interfaces"}, Confidence: "A",
		Concepts: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	g.Expect(s.Create(ctx, &m1)).To(Succeed())
	g.Expect(s.Create(ctx, &m2)).To(Succeed())

	// When test calls store.FindSimilar with (any ctx, "git staging add files", 5)
	results, err := s.FindSimilar(ctx, "git staging add files", 5)
	// Then store.FindSimilar returns (results, nil error)
	g.Expect(err).NotTo(HaveOccurred())
	// And results is non-empty
	g.Expect(results).NotTo(BeEmpty())
	// And results[0].Memory.ID equals m1.ID
	g.Expect(results[0].Memory.ID).To(Equal(m1.ID))
	// And scores are in descending order
	for i := 1; i < len(results); i++ {
		g.Expect(results[i-1].Score).To(BeNumerically(">=", results[i].Score))
	}
}

func TestT44_SurfaceReturnsFrecencyRankedResults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Given three memories with overlapping content matching query:
	// m1: created 1 day ago, impact_score = 0.9 (high frecency)
	m1 := store.Memory{
		ID: "m_00000101", Title: "Recent high", Content: "memory about testing patterns",
		Keywords: []string{"testing", "patterns"}, Confidence: "A",
		Concepts: []string{}, ImpactScore: 0.9,
		CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour),
	}
	// m2: created 30 days ago, impact_score = 0.9 (harmonic mean penalizes old)
	m2 := store.Memory{
		ID: "m_00000102", Title: "Old high", Content: "memory about testing patterns too",
		Keywords: []string{"testing", "patterns"}, Confidence: "A",
		Concepts: []string{}, ImpactScore: 0.9,
		CreatedAt: now.Add(-30 * 24 * time.Hour), UpdatedAt: now.Add(-30 * 24 * time.Hour),
	}
	// m3: created 1 day ago, impact_score = 0.1 (harmonic mean penalizes low impact)
	m3 := store.Memory{
		ID: "m_00000103", Title: "Recent low", Content: "memory about testing patterns also",
		Keywords: []string{"testing", "patterns"}, Confidence: "A",
		Concepts: []string{}, ImpactScore: 0.1,
		CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour),
	}

	g.Expect(s.Create(ctx, &m1)).To(Succeed())
	g.Expect(s.Create(ctx, &m2)).To(Succeed())
	g.Expect(s.Create(ctx, &m3)).To(Succeed())

	// When test calls store.Surface with (any ctx, matching query, 5)
	results, err := s.Surface(ctx, "testing patterns", 5)
	// Then store.Surface returns (results, nil error)
	g.Expect(err).NotTo(HaveOccurred())
	// And results are non-empty
	g.Expect(results).NotTo(BeEmpty())
	// And results[0].Memory.ID equals m1.ID
	g.Expect(results[0].Memory.ID).To(Equal(m1.ID))
	// And results are in descending frecency order
	for i := 1; i < len(results); i++ {
		g.Expect(results[i-1].Score).To(BeNumerically(">=", results[i].Score))
	}
}

func TestT45_ColdStartRankingEqualsRecency(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Given three memories all with default impact_score = 0.5, different creation times
	m1 := store.Memory{
		ID: "m_00000201", Title: "Most recent", Content: "memory about build tools",
		Keywords: []string{"build", "tools"}, Confidence: "A",
		Concepts: []string{}, ImpactScore: 0.5,
		CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour),
	}
	m2 := store.Memory{
		ID: "m_00000202", Title: "Week old", Content: "memory about build tools usage",
		Keywords: []string{"build", "tools"}, Confidence: "A",
		Concepts: []string{}, ImpactScore: 0.5,
		CreatedAt: now.Add(-7 * 24 * time.Hour), UpdatedAt: now.Add(-7 * 24 * time.Hour),
	}
	m3 := store.Memory{
		ID: "m_00000203", Title: "Month old", Content: "memory about build tools guide",
		Keywords: []string{"build", "tools"}, Confidence: "A",
		Concepts: []string{}, ImpactScore: 0.5,
		CreatedAt: now.Add(-30 * 24 * time.Hour), UpdatedAt: now.Add(-30 * 24 * time.Hour),
	}

	g.Expect(s.Create(ctx, &m1)).To(Succeed())
	g.Expect(s.Create(ctx, &m2)).To(Succeed())
	g.Expect(s.Create(ctx, &m3)).To(Succeed())

	// When test calls store.Surface with (any ctx, matching query, 5)
	results, err := s.Surface(ctx, "build tools", 5)
	// Then store.Surface returns (results, nil error)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(len(results)).To(BeNumerically(">=", 3))
	// And results ordered: m1, m2, m3 (most recent first)
	g.Expect(results[0].Memory.ID).To(Equal(m1.ID))
	g.Expect(results[1].Memory.ID).To(Equal(m2.ID))
	g.Expect(results[2].Memory.ID).To(Equal(m3.ID))
}

func TestT46_ConfidenceTiebreakerWhenFrecencyTied(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Given two memories with identical created_at, updated_at, impact_score
	// m1: confidence = "A"
	m1 := store.Memory{
		ID: "m_00000301", Title: "High confidence", Content: "memory about DI patterns",
		Keywords: []string{"dependency", "injection"}, Confidence: "A",
		Concepts: []string{}, ImpactScore: 0.7,
		CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour),
	}
	// m2: confidence = "B"
	m2 := store.Memory{
		ID: "m_00000302", Title: "Low confidence", Content: "memory about DI patterns too",
		Keywords: []string{"dependency", "injection"}, Confidence: "B",
		Concepts: []string{}, ImpactScore: 0.7,
		CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour),
	}

	g.Expect(s.Create(ctx, &m1)).To(Succeed())
	g.Expect(s.Create(ctx, &m2)).To(Succeed())

	// When test calls store.Surface with (any ctx, matching query, 5)
	results, err := s.Surface(ctx, "dependency injection", 5)
	// Then store.Surface returns (results, nil error)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(len(results)).To(BeNumerically(">=", 2))
	// And results[0].Memory.ID equals m1.ID (A > B tiebreaker)
	g.Expect(results[0].Memory.ID).To(Equal(m1.ID))
}

func TestT47_SurfaceRespectsKLimit(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Given more than K memories in store with overlapping content
	for i := range 10 {
		m := store.Memory{
			ID: fmt.Sprintf("m_%08x", 0x400+i), Title: "surface limit test",
			Content:  "common surfacing content for limit testing",
			Keywords: []string{"surface", "limit"}, Confidence: "B",
			Concepts: []string{}, CreatedAt: now, UpdatedAt: now,
		}
		g.Expect(s.Create(ctx, &m)).To(Succeed())
	}

	rapid.Check(t, func(t *rapid.T) {
		// Given any K in [1,5]
		k := rapid.IntRange(1, 5).Draw(t, "k")
		// When test calls store.Surface with (any ctx, any matching query, K)
		results, err := s.Surface(ctx, "common surfacing content", k)
		// Then store.Surface returns (results, nil error)
		g.Expect(err).NotTo(HaveOccurred())
		// And len(results) <= K
		g.Expect(len(results)).To(BeNumerically("<=", k))
	})
}

func TestT48_IncrementSurfacingUpdatesMetadata(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Given a Memory m in store with SurfacingCount = 0 and LastSurfacedAt = ""
	m := store.Memory{
		ID: "m_00000501", Title: "Surfacing test", Content: "test content",
		Keywords: []string{"test"}, Confidence: "A",
		Concepts: []string{}, CreatedAt: now, UpdatedAt: now,
		SurfacingCount: 0,
	}
	g.Expect(s.Create(ctx, &m)).To(Succeed())

	// Verify initial state
	got, err := s.Get(ctx, m.ID)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got.SurfacingCount).To(Equal(0))
	g.Expect(got.LastSurfacedAt).To(BeNil())

	// When test calls store.IncrementSurfacing with (any ctx, [m.ID])
	err = s.IncrementSurfacing(ctx, []string{m.ID})
	// Then store.IncrementSurfacing returns nil error
	g.Expect(err).NotTo(HaveOccurred())

	// When test calls store.Get with (any ctx, m.ID)
	got, err = s.Get(ctx, m.ID)
	// Then store.Get returns (Memory got, nil error)
	g.Expect(err).NotTo(HaveOccurred())
	// And got.SurfacingCount equals 1
	g.Expect(got.SurfacingCount).To(Equal(1))
	// And got.LastSurfacedAt is non-empty
	g.Expect(got.LastSurfacedAt).NotTo(BeNil())
}

func TestT4_FindSimilarRespectsKLimit(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Given more than K memories in store with overlapping content
	for i := range 15 {
		m := store.Memory{
			ID: fmt.Sprintf("m_%08x", i), Title: "testing memory",
			Content:  "common overlapping content for testing purposes",
			Keywords: []string{"test", "overlap"}, Confidence: "B",
			Concepts: []string{}, CreatedAt: now, UpdatedAt: now,
		}
		g.Expect(s.Create(ctx, &m)).To(Succeed())
	}

	rapid.Check(t, func(t *rapid.T) {
		// Given any K in [1,10]
		k := rapid.IntRange(1, 10).Draw(t, "k")
		// When test calls store.FindSimilar with (any ctx, any matching query, K)
		results, err := s.FindSimilar(ctx, "common overlapping content", k)
		// Then store.FindSimilar returns (results, nil error)
		g.Expect(err).NotTo(HaveOccurred())
		// And len(results) <= K
		g.Expect(len(results)).To(BeNumerically("<=", k))
	})
}

func TestT5_UpdatePreservesCreatedAt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Given any Memory m created in store at time T1
		m := genMemory().Draw(t, "memory")
		g.Expect(s.Create(ctx, &m)).To(Succeed())

		// When test calls store.Update with (any ctx, m with UpdatedAt = T2 where T2 > T1)
		t2 := m.CreatedAt.Add(time.Hour)
		m.UpdatedAt = t2
		// Then store.Update returns nil error
		g.Expect(s.Update(ctx, &m)).To(Succeed())

		// When test calls store.Get with (any ctx, m.ID)
		got, err := s.Get(ctx, m.ID)
		// Then store.Get returns (Memory got, nil error)
		g.Expect(err).NotTo(HaveOccurred())
		// And got.CreatedAt equals T1
		g.Expect(got.CreatedAt).To(Equal(m.CreatedAt))
		// And got.UpdatedAt equals T2
		g.Expect(got.UpdatedAt).To(Equal(t2))
	})
}

func TestT6_EnrichmentIncrementsCount(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		// Given any Memory m in store with EnrichmentCount = N
		m := genMemory().Draw(t, "memory")
		n := rapid.IntRange(0, 10).Draw(t, "initial_count")
		m.EnrichmentCount = n
		g.Expect(s.Create(ctx, &m)).To(Succeed())

		// When test calls store.Update with (any ctx, m with EnrichmentCount = N+1)
		m.EnrichmentCount = n + 1
		// Then store.Update returns nil error
		g.Expect(s.Update(ctx, &m)).To(Succeed())

		// When test calls store.Get with (any ctx, m.ID)
		got, err := s.Get(ctx, m.ID)
		// Then store.Get returns (Memory got, nil error)
		g.Expect(err).NotTo(HaveOccurred())
		// And got.EnrichmentCount equals N+1
		g.Expect(got.EnrichmentCount).To(Equal(n + 1))
	})
}

func TestT7_FindSimilarRanksKeywordMatchesHigher(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Given two memories: m1 with keywords ["targ","build","test"], m2 with keywords ["build","tool"]
	m1 := store.Memory{
		ID: "m_00000001", Title: "Build tool", Content: "This project uses a build tool",
		Keywords: []string{"targ", "build", "test"}, Confidence: "A",
		Concepts: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	m2 := store.Memory{
		ID: "m_00000002", Title: "Building", Content: "Remember to use targ for building and testing",
		Keywords: []string{"build", "tool"}, Confidence: "A",
		Concepts: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	g.Expect(s.Create(ctx, &m1)).To(Succeed())
	g.Expect(s.Create(ctx, &m2)).To(Succeed())

	// When test calls store.FindSimilar with (any ctx, "targ", 5)
	results, err := s.FindSimilar(ctx, "targ", 5)
	// Then store.FindSimilar returns (results, nil error)
	g.Expect(err).NotTo(HaveOccurred())
	// And len(results) >= 2
	g.Expect(len(results)).To(BeNumerically(">=", 2))
	// And results[0].Memory.ID equals m1.ID (keyword match ranked higher)
	g.Expect(results[0].Memory.ID).To(Equal(m1.ID))
}

func TestT8_FTS5IndexSyncsWithTable(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Given a Memory m in store with keywords ["unique-keyword-xyz"]
	m := store.Memory{
		ID: "m_00000001", Title: "Unique test", Content: "something",
		Keywords: []string{"unique-keyword-xyz"}, Confidence: "A",
		Concepts: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	g.Expect(s.Create(ctx, &m)).To(Succeed())

	// When test calls store.FindSimilar with (any ctx, "unique-keyword-xyz", 5)
	results, err := s.FindSimilar(ctx, "unique-keyword-xyz", 5)
	// Then store.FindSimilar returns (non-empty results, nil error)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(results).NotTo(BeEmpty())

	// When test calls store.Update with (any ctx, m with keywords = ["different-keyword-abc"])
	m.Keywords = []string{"different-keyword-abc"}
	// Then store.Update returns nil error
	g.Expect(s.Update(ctx, &m)).To(Succeed())

	// When test calls store.FindSimilar with (any ctx, "unique-keyword-xyz", 5)
	results, err = s.FindSimilar(ctx, "unique-keyword-xyz", 5)
	// Then store.FindSimilar returns (empty results, nil error)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(results).To(BeEmpty())

	// When test calls store.FindSimilar with (any ctx, "different-keyword-abc", 5)
	results, err = s.FindSimilar(ctx, "different-keyword-abc", 5)
	// Then store.FindSimilar returns (non-empty results, nil error)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(results).NotTo(BeEmpty())
}

func TestT9_ConcurrentReadsDontConflict(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Given a Memory m in store
	m := store.Memory{
		ID: "m_00000001", Title: "Concurrent test", Content: "test content for concurrency",
		Keywords: []string{"concurrent"}, Confidence: "A",
		Concepts: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	g.Expect(s.Create(ctx, &m)).To(Succeed())

	// When 10 goroutines concurrently call store.FindSimilar with (any ctx, any query, 5)
	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = s.FindSimilar(ctx, "test content", 5)
		}(i)
	}
	wg.Wait()
	// Then all calls return nil error
	for _, err := range errs {
		g.Expect(err).NotTo(HaveOccurred())
	}
}

func genMemory() *rapid.Generator[store.Memory] {
	return rapid.Custom(func(t *rapid.T) store.Memory {
		now := time.Now().UTC().Truncate(time.Second)
		id := rapid.StringMatching(`m_[0-9a-f]{8}`).Draw(t, "id")
		return store.Memory{
			ID:              id,
			Title:           rapid.StringMatching(`[A-Za-z ]{5,30}`).Draw(t, "title"),
			Content:         rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(t, "content"),
			ObservationType: rapid.SampledFrom([]string{"pattern", "preference", "anti-pattern"}).Draw(t, "obs_type"),
			Concepts:        []string{rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "concept")},
			Principle:       rapid.StringMatching(`[A-Za-z ]{5,50}`).Draw(t, "principle"),
			AntiPattern:     rapid.StringMatching(`[A-Za-z ]{5,50}`).Draw(t, "anti_pattern"),
			Rationale:       rapid.StringMatching(`[A-Za-z ]{5,50}`).Draw(t, "rationale"),
			EnrichedContent: rapid.StringMatching(`[A-Za-z ]{10,100}`).Draw(t, "enriched"),
			Keywords:        []string{rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "keyword")},
			Confidence:      rapid.SampledFrom([]string{"A", "B", "C"}).Draw(t, "confidence"),
			EnrichmentCount: 0,
			ImpactScore:     0.5,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	})
}

func setupDB(t *testing.T) *store.SQLiteStore {
	t.Helper()
	// Use a unique file URI with shared cache for concurrent access support.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, err := store.New(db)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
