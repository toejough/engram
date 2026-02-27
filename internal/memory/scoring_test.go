package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestAutoTuneThresholds_ClampsThresholdToMax(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// High leech% → would increase threshold beyond 1.0
	insertMemoriesWithQuadrant(t, db, "leech", 5)
	insertMemoriesWithQuadrant(t, db, "noise", 5)

	// Set threshold very close to max
	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('impact_threshold', '0.99')")
	g.Expect(err).ToNot(HaveOccurred())

	err = AutoTuneThresholds(db)
	g.Expect(err).ToNot(HaveOccurred())

	threshold := getImpactThreshold(t, db)
	g.Expect(threshold).To(BeNumerically("<=", 1.0), "threshold should be clamped to 1.0")
}

func TestAutoTuneThresholds_ClampsThresholdToMin(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Low leech% → would decrease threshold below 0.0
	insertMemoriesWithQuadrant(t, db, "working", 10)

	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('impact_threshold', '0.01')")
	g.Expect(err).ToNot(HaveOccurred())

	err = AutoTuneThresholds(db)
	g.Expect(err).ToNot(HaveOccurred())

	threshold := getImpactThreshold(t, db)
	g.Expect(threshold).To(BeNumerically(">=", 0.0), "threshold should be clamped to 0.0")
}

func TestAutoTuneThresholds_DecreasesThresholdWhenLeechPercentLow(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// 0 leeches out of 10 total = 0% < 5% → decrease threshold
	insertMemoriesWithQuadrant(t, db, "working", 5)
	insertMemoriesWithQuadrant(t, db, "noise", 5)

	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('impact_threshold', '0.5')")
	g.Expect(err).ToNot(HaveOccurred())

	err = AutoTuneThresholds(db)
	g.Expect(err).ToNot(HaveOccurred())

	threshold := getImpactThreshold(t, db)
	g.Expect(threshold).To(BeNumerically("<", 0.5), "threshold should decrease when leech% < 5%%")
	g.Expect(threshold).To(BeNumerically("~", 0.45, 0.001), "threshold should decrease by 0.05 step")
}

func TestAutoTuneThresholds_IncreasesThresholdWhenLeechPercentHigh(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// 2 leeches out of 10 total = 20% > 15% → increase threshold
	insertMemoriesWithQuadrant(t, db, "leech", 2)
	insertMemoriesWithQuadrant(t, db, "working", 4)
	insertMemoriesWithQuadrant(t, db, "noise", 4)

	// Set initial threshold
	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('impact_threshold', '0.3')")
	g.Expect(err).ToNot(HaveOccurred())

	err = AutoTuneThresholds(db)
	g.Expect(err).ToNot(HaveOccurred())

	threshold := getImpactThreshold(t, db)
	g.Expect(threshold).To(BeNumerically(">", 0.3), "threshold should increase when leech% > 15%%")
	g.Expect(threshold).To(BeNumerically("~", 0.35, 0.001), "threshold should increase by 0.05 step")
}

func TestAutoTuneThresholds_NoChangeWhenLeechPercentInRange(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// 1 leech out of 10 = 10% — within 5-15% range
	insertMemoriesWithQuadrant(t, db, "leech", 1)
	insertMemoriesWithQuadrant(t, db, "working", 9)

	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('impact_threshold', '0.3')")
	g.Expect(err).ToNot(HaveOccurred())

	err = AutoTuneThresholds(db)
	g.Expect(err).ToNot(HaveOccurred())

	threshold := getImpactThreshold(t, db)
	g.Expect(threshold).To(BeNumerically("~", 0.3, 0.001), "threshold should not change when leech% is in 5-15%% range")
}

func TestClassifyQuadrants_GemQuadrant(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	setQuadrantThresholds(t, db, 0.5, 0.5)
	memID := insertMemoryWithScores(t, db, 0.2, 0.8) // low importance + high impact

	err = ClassifyQuadrants(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(getQuadrant(t, db, memID)).To(Equal("gem"))
}

func TestClassifyQuadrants_LeechQuadrant(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	setQuadrantThresholds(t, db, 0.5, 0.5)
	memID := insertMemoryWithScores(t, db, 0.8, 0.2) // high importance + low impact

	err = ClassifyQuadrants(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(getQuadrant(t, db, memID)).To(Equal("leech"))
}

func TestClassifyQuadrants_NoiseQuadrant(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	setQuadrantThresholds(t, db, 0.5, 0.5)
	memID := insertMemoryWithScores(t, db, 0.2, 0.2) // low importance + low impact

	err = ClassifyQuadrants(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(getQuadrant(t, db, memID)).To(Equal("noise"))
}

func TestClassifyQuadrants_UsesMetadataThresholds(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// importance=0.8, impact=0.5 — just on the boundary
	memID := insertMemoryWithScores(t, db, 0.8, 0.5)

	// Threshold=0.6 → impact 0.5 < 0.6 → leech
	setQuadrantThresholds(t, db, 0.5, 0.6)
	err = ClassifyQuadrants(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(getQuadrant(t, db, memID)).To(Equal("leech"), "impact 0.5 should be below threshold 0.6")

	// Update threshold to 0.4 → impact 0.5 > 0.4 → working
	setQuadrantThresholds(t, db, 0.5, 0.4)
	err = ClassifyQuadrants(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(getQuadrant(t, db, memID)).To(Equal("working"), "impact 0.5 should be above threshold 0.4")
}

func TestClassifyQuadrants_WorkingQuadrant(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	setQuadrantThresholds(t, db, 0.5, 0.5)
	memID := insertMemoryWithScores(t, db, 0.8, 0.8) // high importance + high impact

	err = ClassifyQuadrants(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(getQuadrant(t, db, memID)).To(Equal("working"))
}

// ─── GetQuadrantSummary tests ──────────────────────────────────────────────────

func TestGetQuadrantSummary_ReturnsCorrectCounts(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	insertMemoriesWithQuadrant(t, db, "working", 3)
	insertMemoriesWithQuadrant(t, db, "leech", 2)
	insertMemoriesWithQuadrant(t, db, "gem", 4)
	insertMemoriesWithQuadrant(t, db, "noise", 1)

	summary, err := GetQuadrantSummary(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(summary.WorkingCount).To(Equal(3))
	g.Expect(summary.LeechCount).To(Equal(2))
	g.Expect(summary.GemCount).To(Equal(4))
	g.Expect(summary.NoiseCount).To(Equal(1))
}

func TestGetQuadrantSummary_ReturnsCurrentThresholds(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Override defaults
	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('alpha_weight', '0.7')")
	g.Expect(err).ToNot(HaveOccurred())
	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('importance_threshold', '0.4')")
	g.Expect(err).ToNot(HaveOccurred())
	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('impact_threshold', '0.6')")
	g.Expect(err).ToNot(HaveOccurred())

	summary, err := GetQuadrantSummary(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(summary.AlphaWeight).To(BeNumerically("~", 0.7, 0.001))
	g.Expect(summary.ImportanceThreshold).To(BeNumerically("~", 0.4, 0.001))
	g.Expect(summary.ImpactThreshold).To(BeNumerically("~", 0.6, 0.001))
}

// ─── ScoreSession tests ───────────────────────────────────────────────────────

func TestScoreSession_EvaluatesHaikuRelevantEvents(t *testing.T) {
	g := NewWithT(t)
	db, memID := scoringTestDB(t, "test memory")

	eventID := logRelevantEvent(t, db, memID, "sess-eval", time.Now().UTC().Truncate(time.Second))

	server := makePostEvalServer(0.85, "positive")
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	err := ScoreSession(db, "sess-eval", llm)
	g.Expect(err).ToNot(HaveOccurred())

	var (
		faithfulness  sql.NullFloat64
		outcomeSignal sql.NullString
	)

	err = db.QueryRow(
		"SELECT faithfulness, outcome_signal FROM surfacing_events WHERE id = ?", eventID,
	).Scan(&faithfulness, &outcomeSignal)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(faithfulness.Valid).To(BeTrue(), "faithfulness should be set after post-eval")
	g.Expect(faithfulness.Float64).To(BeNumerically("~", 0.85, 0.001))
	g.Expect(outcomeSignal.String).To(Equal("positive"))
}

func TestScoreSession_IncrementsLeechCountWhenFaithfulnessLow(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source, leech_count) VALUES ('test memory', 'test', 0)")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID, _ := res.LastInsertId()

	logRelevantEvent(t, db, memID, "sess-leech", time.Now().UTC().Truncate(time.Second))

	// faithfulness < 0.3 → leech
	server := makePostEvalServer(0.1, "negative")
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	err = ScoreSession(db, "sess-leech", llm)
	g.Expect(err).ToNot(HaveOccurred())

	var leechCount int

	err = db.QueryRow("SELECT leech_count FROM embeddings WHERE id = ?", memID).Scan(&leechCount)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(leechCount).To(Equal(1), "leech_count should be incremented when faithfulness < 0.3")
}

func TestScoreSession_NoOpWhenNoHaikuRelevantEvents(t *testing.T) {
	g := NewWithT(t)
	db, memID := scoringTestDB(t, "test memory")

	// haiku_relevant = false
	notRelevant := false
	event := SurfacingEvent{
		MemoryID:      memID,
		QueryText:     "test query",
		HookEvent:     "PreToolUse",
		Timestamp:     time.Now().UTC().Truncate(time.Second),
		SessionID:     "sess-nohaiku",
		HaikuRelevant: &notRelevant,
	}
	_, err := LogSurfacingEvent(db, event)
	g.Expect(err).ToNot(HaveOccurred())

	var serverCalled int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&serverCalled, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	err = ScoreSession(db, "sess-nohaiku", llm)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(atomic.LoadInt32(&serverCalled)).To(Equal(int32(0)), "server should not be called for non-haiku-relevant events")
}

func TestScoreSession_NoOpWhenNoSurfacingEvents(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	var serverCalled int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&serverCalled, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	err = ScoreSession(db, "sess-empty", llm)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(atomic.LoadInt32(&serverCalled)).To(Equal(int32(0)), "server should not be called when session has no surfacing events")
}

func TestScoreSession_ResetsLeechCountWhenFaithfulnessHigh(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Start with leech_count = 5
	res, err := db.Exec("INSERT INTO embeddings (content, source, leech_count) VALUES ('test memory', 'test', 5)")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID, _ := res.LastInsertId()

	logRelevantEvent(t, db, memID, "sess-reset", time.Now().UTC().Truncate(time.Second))

	// faithfulness >= 0.3 → reset
	server := makePostEvalServer(0.9, "positive")
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	err = ScoreSession(db, "sess-reset", llm)
	g.Expect(err).ToNot(HaveOccurred())

	var leechCount int

	err = db.QueryRow("SELECT leech_count FROM embeddings WHERE id = ?", memID).Scan(&leechCount)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(leechCount).To(Equal(0), "leech_count should be reset to 0 when faithfulness >= 0.3")
}

func TestScoreSession_SkipsAlreadyEvaluatedEvents(t *testing.T) {
	g := NewWithT(t)
	db, memID := scoringTestDB(t, "test memory")

	// Insert event with faithfulness already set (pre-evaluated)
	existing := 0.7
	relevant := true
	event := SurfacingEvent{
		MemoryID:      memID,
		QueryText:     "test query",
		HookEvent:     "PreToolUse",
		Timestamp:     time.Now().UTC().Truncate(time.Second),
		SessionID:     "sess-skip",
		HaikuRelevant: &relevant,
		Faithfulness:  &existing,
	}
	eventID, err := LogSurfacingEvent(db, event)
	g.Expect(err).ToNot(HaveOccurred())

	var serverCalled int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&serverCalled, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	err = ScoreSession(db, "sess-skip", llm)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(atomic.LoadInt32(&serverCalled)).To(Equal(int32(0)), "server should not be called for already-evaluated events")

	// Original faithfulness preserved
	var faithfulness sql.NullFloat64

	err = db.QueryRow("SELECT faithfulness FROM surfacing_events WHERE id = ?", eventID).Scan(&faithfulness)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(faithfulness.Float64).To(BeNumerically("~", 0.7, 0.001))
}

func TestScoreSession_UpdatesImpactScore(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert memory with impact_score=0.0
	res, err := db.Exec("INSERT INTO embeddings (content, source, impact_score) VALUES ('test memory', 'test', 0.0)")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID, _ := res.LastInsertId()

	base := time.Now().UTC().Truncate(time.Second)
	logRelevantEvent(t, db, memID, "sess-impact", base)
	logRelevantEvent(t, db, memID, "sess-impact", base.Add(time.Minute))

	// Server returns 0.6 for first event, 0.9 for second
	server, _ := makeCountingPostEvalServer([]float64{0.6, 0.9})
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	err = ScoreSession(db, "sess-impact", llm)
	g.Expect(err).ToNot(HaveOccurred())

	var impactScore float64

	err = db.QueryRow("SELECT impact_score FROM embeddings WHERE id = ?", memID).Scan(&impactScore)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(impactScore).To(BeNumerically(">", 0.0), "impact_score should be updated after ScoreSession")
	g.Expect(impactScore).To(BeNumerically("<=", 1.0), "impact_score should be in [0,1]")
}

// ─── UpdateSurfacingOutcome tests ─────────────────────────────────────────────

func TestUpdateSurfacingOutcome(t *testing.T) {
	g := NewWithT(t)
	db, memID := scoringTestDB(t, "test memory")

	event := SurfacingEvent{
		MemoryID:  memID,
		QueryText: "test query",
		HookEvent: "PreToolUse",
		Timestamp: time.Now().UTC().Truncate(time.Second),
		SessionID: "sess-outcome",
	}
	eventID, err := LogSurfacingEvent(db, event)
	g.Expect(err).ToNot(HaveOccurred())

	err = UpdateSurfacingOutcome(db, eventID, 0.75, "positive")
	g.Expect(err).ToNot(HaveOccurred())

	var (
		faithfulness  sql.NullFloat64
		outcomeSignal sql.NullString
	)

	err = db.QueryRow(
		"SELECT faithfulness, outcome_signal FROM surfacing_events WHERE id = ?", eventID,
	).Scan(&faithfulness, &outcomeSignal)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(faithfulness.Valid).To(BeTrue(), "faithfulness should be set after UpdateSurfacingOutcome")
	g.Expect(faithfulness.Float64).To(BeNumerically("~", 0.75, 0.001))
	g.Expect(outcomeSignal.String).To(Equal("positive"))
}

func getImpactThreshold(t *testing.T, db *sql.DB) float64 {
	t.Helper()

	v, err := getMetadata(db, "impact_threshold")
	if err != nil {
		t.Fatalf("getMetadata impact_threshold: %v", err)
	}

	var f float64

	_, err = fmt.Sscanf(v, "%f", &f)
	if err != nil {
		t.Fatalf("parse impact_threshold %q: %v", v, err)
	}

	return f
}

func getQuadrant(t *testing.T, db *sql.DB, memID int64) string {
	t.Helper()

	var q string

	err := db.QueryRow("SELECT quadrant FROM embeddings WHERE id = ?", memID).Scan(&q)
	if err != nil {
		t.Fatalf("SELECT quadrant: %v", err)
	}

	return q
}

// ─── AutoTuneThresholds tests ─────────────────────────────────────────────────

func insertMemoriesWithQuadrant(t *testing.T, db *sql.DB, quadrant string, count int) {
	t.Helper()

	for i := range count {
		_, err := db.Exec(
			"INSERT INTO embeddings (content, source, quadrant) VALUES (?, 'test', ?)",
			fmt.Sprintf("mem-%s-%d", quadrant, i), quadrant,
		)
		if err != nil {
			t.Fatalf("INSERT embeddings with quadrant: %v", err)
		}
	}
}

func insertMemoryWithScores(t *testing.T, db *sql.DB, importance, impact float64) int64 {
	t.Helper()

	res, err := db.Exec(
		"INSERT INTO embeddings (content, source, importance_score, impact_score) VALUES (?, 'test', ?, ?)",
		fmt.Sprintf("mem-imp%.1f-impact%.1f", importance, impact), importance, impact,
	)
	if err != nil {
		t.Fatalf("INSERT embeddings with scores: %v", err)
	}

	id, _ := res.LastInsertId()

	return id
}

// logRelevantEvent inserts a haiku_relevant=true surfacing event with no faithfulness.
func logRelevantEvent(t *testing.T, db *sql.DB, memID int64, sessionID string, ts time.Time) int64 {
	t.Helper()

	relevant := true
	event := SurfacingEvent{
		MemoryID:      memID,
		QueryText:     "test query",
		HookEvent:     "PreToolUse",
		Timestamp:     ts,
		SessionID:     sessionID,
		HaikuRelevant: &relevant,
	}

	id, err := LogSurfacingEvent(db, event)
	if err != nil {
		t.Fatalf("LogSurfacingEvent: %v", err)
	}

	return id
}

// makeCountingPostEvalServer creates an httptest server that counts calls and returns
// different faithfulness values per call (in order of the provided values slice).
func makeCountingPostEvalServer(values []float64) (*httptest.Server, *int32) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(atomic.AddInt32(&callCount, 1)) - 1

		faith := 0.5
		if n < len(values) {
			faith = values[n]
		}

		text, _ := json.Marshal(map[string]any{"faithfulness": faith, "signal": "positive"})
		response := map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"model": "claude-haiku-4-5-20251001",
			"content": []map[string]any{
				{"type": "text", "text": string(text)},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))

	return server, &callCount
}

// makePostEvalServer creates an httptest server that returns a fixed faithfulness/signal
// response in the Anthropic API format, used for ScoreSession post-evaluation mocks.
func makePostEvalServer(faithfulness float64, signal string) *httptest.Server {
	text, _ := json.Marshal(map[string]any{"faithfulness": faithfulness, "signal": signal})

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"model": "claude-haiku-4-5-20251001",
			"content": []map[string]any{
				{"type": "text", "text": string(text)},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
}

// scoringTestDB creates a test DB and inserts a single embedding, returning the DB and memory ID.
func scoringTestDB(t *testing.T, content string) (*sql.DB, int64) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		t.Fatalf("initEmbeddingsDB: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES (?, 'test')", content)
	if err != nil {
		t.Fatalf("INSERT embeddings: %v", err)
	}

	id, _ := res.LastInsertId()

	return db, id
}

// ─── ClassifyQuadrants tests ──────────────────────────────────────────────────

func setQuadrantThresholds(t *testing.T, db *sql.DB, importanceThreshold, impactThreshold float64) {
	t.Helper()

	_, err := db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('importance_threshold', ?)",
		fmt.Sprintf("%g", importanceThreshold))
	if err != nil {
		t.Fatalf("set importance_threshold: %v", err)
	}

	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('impact_threshold', ?)",
		fmt.Sprintf("%g", impactThreshold))
	if err != nil {
		t.Fatalf("set impact_threshold: %v", err)
	}
}
