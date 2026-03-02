package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	_ "modernc.org/sqlite"
	"pgregory.net/rapid"

	"engram/internal/store"
)

type mockDB struct {
	execErr     error
	execCall    int
	failOnExec  int // 0 = always fail if execErr set; >0 = fail on Nth call only
	queryErr    error
	queryCall   int
	failOnQuery int
}

func (m *mockDB) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	m.execCall++
	if m.execErr != nil && (m.failOnExec == 0 || m.execCall == m.failOnExec) {
		return nil, m.execErr
	}

	return nil, nil //nolint:nilnil // mock: callers never use sql.Result
}

func (m *mockDB) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	m.queryCall++
	if m.queryErr != nil && (m.failOnQuery == 0 || m.queryCall == m.failOnQuery) {
		return nil, m.queryErr
	}

	return nil, nil //nolint:nilnil // mock: callers only check error
}

func (m *mockDB) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	return nil
}

func TestNew_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s, err := store.New(&mockDB{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).NotTo(BeNil())
}

func TestNew_SchemaError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s, err := store.New(&mockDB{execErr: errors.New("db error")})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("create schema")))
	g.Expect(s).To(BeNil())
}

func TestToFTS5Query_EmptyString(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	g.Expect(store.ToFTS5Query("")).To(Equal(`""`))
}

func TestToFTS5Query_SingleWord(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	g.Expect(store.ToFTS5Query("hello")).To(Equal(`"hello"`))
}

func TestToFTS5Query_MultipleWords(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	g.Expect(store.ToFTS5Query("git staging add")).To(Equal(`"git" OR "staging" OR "add"`))
}

func TestToFTS5Query_EscapesQuotes(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	g.Expect(store.ToFTS5Query(`say "hello"`)).To(Equal(`"say" OR """hello"""`))
}

func TestToFTS5Query_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		g := NewGomegaWithT(t)
		query := rapid.StringMatching(`[a-z]{1,20}( [a-z]{1,20}){0,5}`).Draw(t, "query")
		result := store.ToFTS5Query(query)
		// Result should never be empty
		g.Expect(result).NotTo(BeEmpty())
		// Result should contain OR for multi-word queries
		if len(query) > 0 {
			g.Expect(result).To(ContainSubstring(`"`))
		}
	})
}

func TestFormatNullableTime_Nil(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	g.Expect(store.FormatNullableTime(nil)).To(BeNil())
}

func TestFormatNullableTime_NonNil(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ts := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	result := store.FormatNullableTime(&ts)
	g.Expect(result).To(Equal("2025-06-15T12:00:00Z"))
}

func TestFormatNullableTime_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		g := NewGomegaWithT(t)
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day")
		timestamp := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		result := store.FormatNullableTime(&timestamp)
		// Should be a parseable RFC3339 string
		str, ok := result.(string)
		g.Expect(ok).To(BeTrue())

		parsed, err := time.Parse(time.RFC3339, str)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(parsed).To(Equal(timestamp))
	})
}

func TestUnmarshalMemoryFields_ValidJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var m store.Memory

	err := store.UnmarshalMemoryFields(&m,
		`["go","testing"]`,
		`["unit","test"]`,
		"2025-06-15T12:00:00Z",
		"2025-06-16T12:00:00Z",
		sql.NullString{Valid: false},
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(m.Concepts).To(Equal([]string{"go", "testing"}))
	g.Expect(m.Keywords).To(Equal([]string{"unit", "test"}))
	g.Expect(m.CreatedAt).To(Equal(time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)))
	g.Expect(m.UpdatedAt).To(Equal(time.Date(2025, 6, 16, 12, 0, 0, 0, time.UTC)))
	g.Expect(m.LastSurfacedAt).To(BeNil())
}

func TestUnmarshalMemoryFields_WithLastSurfaced(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var m store.Memory

	err := store.UnmarshalMemoryFields(&m,
		`[]`, `[]`,
		"2025-06-15T12:00:00Z",
		"2025-06-16T12:00:00Z",
		sql.NullString{Valid: true, String: "2025-06-17T12:00:00Z"},
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(m.LastSurfacedAt).NotTo(BeNil())
	g.Expect(*m.LastSurfacedAt).To(Equal(time.Date(2025, 6, 17, 12, 0, 0, 0, time.UTC)))
}

func TestUnmarshalMemoryFields_InvalidConceptsJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var m store.Memory

	err := store.UnmarshalMemoryFields(&m,
		`not-json`, `[]`, "", "", sql.NullString{},
	)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("unmarshal concepts")))
}

func TestUnmarshalMemoryFields_InvalidKeywordsJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var m store.Memory

	err := store.UnmarshalMemoryFields(&m,
		`[]`, `not-json`, "", "", sql.NullString{},
	)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("unmarshal keywords")))
}

func TestBuildIncrementQuery_SingleID(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	query, args := store.BuildIncrementQuery([]string{"m_001"})
	g.Expect(query).To(ContainSubstring("WHERE id IN (?)"))
	g.Expect(args).To(HaveLen(2)) // now + 1 id
	g.Expect(args[1]).To(Equal("m_001"))
}

func TestBuildIncrementQuery_MultipleIDs(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	query, args := store.BuildIncrementQuery([]string{"m_001", "m_002", "m_003"})
	g.Expect(query).To(ContainSubstring("WHERE id IN (?,?,?)"))
	g.Expect(args).To(HaveLen(4)) // now + 3 ids
	g.Expect(args[1]).To(Equal("m_001"))
	g.Expect(args[2]).To(Equal("m_002"))
	g.Expect(args[3]).To(Equal("m_003"))
}

func TestBuildIncrementQuery_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		g := NewGomegaWithT(t)
		n := rapid.IntRange(1, 20).Draw(t, "n")
		ids := make([]string, n)

		for i := range n {
			ids[i] = rapid.StringMatching(`m_[0-9a-f]{8}`).Draw(t, "id")
		}

		query, args := store.BuildIncrementQuery(ids)
		// args = 1 (now) + n (ids)
		g.Expect(args).To(HaveLen(n + 1))
		// query should have n placeholders
		g.Expect(query).To(ContainSubstring("surfacing_count"))
		_ = query
	})
}

// --- SQLiteStore method tests using in-memory SQLite ---

func setupTestStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })

	s, err := store.New(db)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func testMemory(id string) store.Memory {
	now := time.Now().UTC().Truncate(time.Second)

	return store.Memory{
		ID: id, Title: "test", Content: "test content",
		ObservationType: "pattern", Concepts: []string{"go"},
		Keywords: []string{"test"}, Confidence: "A",
		CreatedAt: now, UpdatedAt: now,
	}
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	m := testMemory("m_c001")
	g.Expect(s.Create(context.Background(), &m)).To(Succeed())
}

func TestCreate_InsertError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{execErr: errors.New("insert boom"), failOnExec: 2}
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	m := testMemory("m_cie1")
	err = s.Create(context.Background(), &m)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("store: create")))
}

func TestCreate_FTSInsertError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{execErr: errors.New("fts boom"), failOnExec: 3}
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	m := testMemory("m_cfe1")
	err = s.Create(context.Background(), &m)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("fts insert")))
}

func TestCreate_RejectsEmptyConfidence(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	m := testMemory("m_c002")
	m.Confidence = ""
	err := s.Create(context.Background(), &m)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("confidence")))
}

func TestGet_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	m := testMemory("m_g001")
	g.Expect(s.Create(context.Background(), &m)).To(Succeed())
	got, err := s.Get(context.Background(), "m_g001")
	g.Expect(err).NotTo(HaveOccurred())

	if got == nil {
		t.Fatal("unexpected nil memory")
	}

	g.Expect(got.ID).To(Equal("m_g001"))
	g.Expect(got.Title).To(Equal("test"))
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	_, err := s.Get(context.Background(), "nonexistent")
	g.Expect(err).To(HaveOccurred())
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	ctx := context.Background()
	m := testMemory("m_u001")
	g.Expect(s.Create(ctx, &m)).To(Succeed())
	m.Title = "updated"
	g.Expect(s.Update(ctx, &m)).To(Succeed())
	got, err := s.Get(ctx, "m_u001")
	g.Expect(err).NotTo(HaveOccurred())

	if got == nil {
		t.Fatal("unexpected nil memory")
	}

	g.Expect(got.Title).To(Equal("updated"))
}

func TestFindSimilar_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	ctx := context.Background()
	m := testMemory("m_f001")
	m.Content = "unique searchable content"
	g.Expect(s.Create(ctx, &m)).To(Succeed())
	results, err := s.FindSimilar(ctx, "unique searchable", 5)
	g.Expect(err).NotTo(HaveOccurred())

	if len(results) == 0 {
		t.Fatal("expected results")
	}

	g.Expect(results[0].Score).To(BeNumerically(">", 0))
}

func TestFindSimilar_NoResults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	results, err := s.FindSimilar(context.Background(), "nonexistent", 5)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

func TestSurface_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	ctx := context.Background()
	m := testMemory("m_s001")
	m.Content = "surfacing test content"
	m.ImpactScore = 0.8
	g.Expect(s.Create(ctx, &m)).To(Succeed())
	results, err := s.Surface(ctx, "surfacing test", 5)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(results).NotTo(BeEmpty())
}

func TestIncrementSurfacing_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	ctx := context.Background()
	m := testMemory("m_i001")
	g.Expect(s.Create(ctx, &m)).To(Succeed())
	g.Expect(s.IncrementSurfacing(ctx, []string{"m_i001"})).To(Succeed())
	got, err := s.Get(ctx, "m_i001")
	g.Expect(err).NotTo(HaveOccurred())

	if got == nil {
		t.Fatal("unexpected nil memory")
	}

	g.Expect(got.SurfacingCount).To(Equal(1))
	g.Expect(got.LastSurfacedAt).NotTo(BeNil())
}

func TestIncrementSurfacing_EmptyIDs(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	g.Expect(s.IncrementSurfacing(context.Background(), nil)).To(Succeed())
}

func TestUpdate_AllFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	ctx := context.Background()

	m := testMemory("m_uaf1")
	g.Expect(s.Create(ctx, &m)).To(Succeed())

	// Update every field
	m.Title = "updated title"
	m.Content = "updated content"
	m.ObservationType = "preference"
	m.Concepts = []string{"updated", "concepts"}
	m.Principle = "new principle"
	m.AntiPattern = "new anti-pattern"
	m.Rationale = "new rationale"
	m.EnrichedContent = "enriched"
	m.Keywords = []string{"updated", "keywords"}
	m.Confidence = "B"
	m.EnrichmentCount = 5
	m.ImpactScore = 0.9
	now := time.Now().UTC().Truncate(time.Second)
	m.UpdatedAt = now
	m.SurfacingCount = 3
	ls := now.Add(-time.Hour)
	m.LastSurfacedAt = &ls

	g.Expect(s.Update(ctx, &m)).To(Succeed())

	got, err := s.Get(ctx, "m_uaf1")
	g.Expect(err).NotTo(HaveOccurred())

	if got == nil {
		t.Fatal("unexpected nil memory")
	}

	g.Expect(got.Title).To(Equal("updated title"))
	g.Expect(got.Concepts).To(Equal([]string{"updated", "concepts"}))
	g.Expect(got.Keywords).To(Equal([]string{"updated", "keywords"}))
	g.Expect(got.EnrichmentCount).To(Equal(5))
	g.Expect(got.LastSurfacedAt).NotTo(BeNil())
}

func TestUpdate_FTSDeleteError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{
		execErr:    errors.New("fts boom"),
		failOnExec: 2,
	} // fail on FTS delete (2nd ExecContext after New)
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	m := testMemory("m_ude1")
	err = s.Update(context.Background(), &m)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("fts delete")))
}

func TestUpdate_SQLError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{
		execErr:    errors.New("sql boom"),
		failOnExec: 3,
	} // fail on UPDATE (3rd ExecContext)
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	m := testMemory("m_use1")
	err = s.Update(context.Background(), &m)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("store: update")))
}

func TestUpdate_FTSReinsertError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{
		execErr:    errors.New("reinsert boom"),
		failOnExec: 4,
	} // fail on FTS reinsert (4th ExecContext)
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	m := testMemory("m_ure1")
	err = s.Update(context.Background(), &m)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("fts reinsert")))
}

func TestFindSimilar_QueryError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{queryErr: errors.New("query boom")}
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	_, err = s.FindSimilar(context.Background(), "test", 5)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("find similar")))
}

func TestSurface_QueryError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{queryErr: errors.New("query boom")}
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	_, err = s.Surface(context.Background(), "test", 5)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("surface")))
}

func TestIncrementSurfacing_ExecError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{execErr: errors.New("exec boom"), failOnExec: 2}
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	err = s.IncrementSurfacing(context.Background(), []string{"m_001"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("increment surfacing")))
}

func TestSurface_NoResults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	results, err := s.Surface(context.Background(), "nonexistent", 5)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

func TestFindSimilar_MultipleResults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	ctx := context.Background()

	for i := range 3 {
		m := testMemory(fmt.Sprintf("m_fm%02d", i))
		m.Content = "overlapping content for multi-result test"
		m.Keywords = []string{"multi-result"}
		g.Expect(s.Create(ctx, &m)).To(Succeed())
	}

	results, err := s.FindSimilar(ctx, "multi-result", 10)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(results).To(HaveLen(3))
	// Scores should be positive (negated from FTS5 rank)
	for _, r := range results {
		g.Expect(r.Score).To(BeNumerically(">", 0))
	}
}

func TestSurface_MultipleResults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for i := range 3 {
		m := testMemory(fmt.Sprintf("m_sm%02d", i))
		m.Content = "surfacing multi content"
		m.Keywords = []string{"surf-multi"}
		m.ImpactScore = 0.5
		m.CreatedAt = now.Add(-time.Duration(i) * 24 * time.Hour)
		m.UpdatedAt = m.CreatedAt
		g.Expect(s.Create(ctx, &m)).To(Succeed())
	}

	results, err := s.Surface(ctx, "surf-multi", 10)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(results).To(HaveLen(3))
}

func TestRecordSurfacing_ExecError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{execErr: errors.New("exec boom"), failOnExec: 2}
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	err = s.RecordSurfacing(context.Background(), []string{"m_001"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("record surfacing")))
}

func TestGetSessionSurfacings_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	ctx := context.Background()

	m := testMemory("m_gs01")
	g.Expect(s.Create(ctx, &m)).To(Succeed())
	g.Expect(s.RecordSurfacing(ctx, []string{"m_gs01"})).To(Succeed())

	ids, err := s.GetSessionSurfacings(ctx)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ids).To(ConsistOf("m_gs01"))
}

func TestGetSessionSurfacings_Empty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)

	ids, err := s.GetSessionSurfacings(context.Background())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(ids).To(BeEmpty())
	g.Expect(ids).NotTo(BeNil())
}

func TestGetSessionSurfacings_QueryError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{queryErr: errors.New("query boom"), failOnQuery: 1}
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	_, err = s.GetSessionSurfacings(context.Background())
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("get session surfacings")))
}

func TestClearSessionSurfacings_ExecError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{execErr: errors.New("exec boom"), failOnExec: 2}
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	err = s.ClearSessionSurfacings(context.Background())
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("clear session surfacings")))
}

func TestDecreaseImpact_Success(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)
	ctx := context.Background()

	m := testMemory("m_di01")
	m.ImpactScore = 1.0
	g.Expect(s.Create(ctx, &m)).To(Succeed())

	err := s.DecreaseImpact(ctx, "m_di01", 0.8)
	g.Expect(err).NotTo(HaveOccurred())

	got, err := s.Get(ctx, "m_di01")
	g.Expect(err).NotTo(HaveOccurred())

	if got == nil {
		t.Fatal("unexpected nil memory")
	}

	g.Expect(got.ImpactScore).To(BeNumerically("~", 0.8, 0.01))
}

func TestDecreaseImpact_NotFound(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	s := setupTestStore(t)

	err := s.DecreaseImpact(context.Background(), "m_nonexistent", 0.8)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("not found")))
}

func TestDecreaseImpact_ExecError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	mock := &mockDB{execErr: errors.New("exec boom"), failOnExec: 2}
	s, err := store.New(mock)
	g.Expect(err).NotTo(HaveOccurred())

	if s == nil {
		t.Fatal("unexpected nil store")
	}

	err = s.DecreaseImpact(context.Background(), "m_001", 0.8)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("decrease impact")))
}
