package memory

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestEmbeddingsNewColumns(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Verify new columns on embeddings table per data-model.md
	newColumns := map[string]string{
		"importance_score": "REAL",
		"impact_score":     "REAL",
		"effectiveness":    "REAL",
		"quadrant":         "TEXT",
		"leech_count":      "INTEGER",
	}
	for col, expectedType := range newColumns {
		var colType string

		err := db.QueryRow(
			"SELECT type FROM pragma_table_info('embeddings') WHERE name = ?", col,
		).Scan(&colType)
		g.Expect(err).ToNot(HaveOccurred(), "column %s should exist on embeddings", col)
		g.Expect(colType).To(Equal(expectedType), "column %s type mismatch", col)
	}

	// Verify defaults by inserting a row and reading back
	_, err = db.Exec("INSERT INTO embeddings (content, source) VALUES ('test', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	var (
		importanceScore, impactScore, effectiveness float64
		quadrant                                    string
		leechCount                                  int
	)

	err = db.QueryRow(`
		SELECT importance_score, impact_score, effectiveness, quadrant, leech_count
		FROM embeddings WHERE content = 'test'
	`).Scan(&importanceScore, &impactScore, &effectiveness, &quadrant, &leechCount)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(importanceScore).To(Equal(0.0))
	g.Expect(impactScore).To(Equal(0.0))
	g.Expect(effectiveness).To(Equal(0.0))
	g.Expect(quadrant).To(Equal("noise"))
	g.Expect(leechCount).To(Equal(0))
}

func TestGetMemorySurfacingHistory(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES ('mem', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	for i := range 3 {
		e := SurfacingEvent{
			MemoryID:  memoryID,
			QueryText: fmt.Sprintf("q%d", i),
			HookEvent: "evt",
			Timestamp: base.Add(time.Duration(i) * time.Minute),
			SessionID: "s1",
		}
		_, err = LogSurfacingEvent(db, e)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Returns most recent first (DESC)
	events, err := GetMemorySurfacingHistory(db, memoryID, 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(events).To(HaveLen(3))

	if len(events) < 3 {
		t.Fatalf("expected 3 events from GetMemorySurfacingHistory, got %d", len(events))
	}

	g.Expect(events[0].QueryText).To(Equal("q2"))
	g.Expect(events[1].QueryText).To(Equal("q1"))
	g.Expect(events[2].QueryText).To(Equal("q0"))

	// Limit is respected
	events, err = GetMemorySurfacingHistory(db, memoryID, 2)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(events).To(HaveLen(2))

	if len(events) < 1 {
		t.Fatal("expected at least 1 event from GetMemorySurfacingHistory")
	}

	g.Expect(events[0].QueryText).To(Equal("q2"))

	// non-existent memory returns empty slice, not error
	events, err = GetMemorySurfacingHistory(db, 99999, 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(events).To(BeEmpty())
}

func TestGetSessionSurfacingEvents(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES ('mem', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Insert two events for "s1" (inserted out of timestamp order)
	e1 := SurfacingEvent{MemoryID: memoryID, QueryText: "q1", HookEvent: "evt", Timestamp: base.Add(2 * time.Minute), SessionID: "s1"}
	e2 := SurfacingEvent{MemoryID: memoryID, QueryText: "q2", HookEvent: "evt", Timestamp: base.Add(1 * time.Minute), SessionID: "s1"}
	// One event for "s2"
	e3 := SurfacingEvent{MemoryID: memoryID, QueryText: "q3", HookEvent: "evt", Timestamp: base, SessionID: "s2"}

	_, err = LogSurfacingEvent(db, e1)
	g.Expect(err).ToNot(HaveOccurred())
	_, err = LogSurfacingEvent(db, e2)
	g.Expect(err).ToNot(HaveOccurred())
	_, err = LogSurfacingEvent(db, e3)
	g.Expect(err).ToNot(HaveOccurred())

	// s1: should return e2 then e1 (ASC by timestamp)
	events, err := GetSessionSurfacingEvents(db, "s1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(events).To(HaveLen(2))

	if len(events) < 2 {
		t.Fatalf("expected 2 events from GetSessionSurfacingEvents, got %d", len(events))
	}

	g.Expect(events[0].QueryText).To(Equal("q2"))
	g.Expect(events[1].QueryText).To(Equal("q1"))

	// non-existent session returns empty slice, not error
	events, err = GetSessionSurfacingEvents(db, "nonexistent")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(events).To(BeEmpty())
}

func TestLogSurfacingEvent(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES ('mem', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	now := time.Now().UTC().Truncate(time.Second)
	relevant := true
	score := 0.85
	synthesize := false
	faith := 0.9

	event := SurfacingEvent{
		MemoryID:            memoryID,
		QueryText:           "what is X?",
		HookEvent:           "PreToolUse",
		Timestamp:           now,
		SessionID:           "sess-1",
		HaikuRelevant:       &relevant,
		HaikuTag:            "relevant",
		HaikuRelevanceScore: &score,
		ShouldSynthesize:    &synthesize,
		Faithfulness:        &faith,
		OutcomeSignal:       "positive",
		UserFeedback:        "helpful",
		E5Similarity:        0.75,
		ContextPrecision:    0.8,
	}

	id, err := LogSurfacingEvent(db, event)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(id).To(BeNumerically(">", int64(0)))

	// Verify stored data by reading raw columns
	var (
		gotMemID                             int64
		gotQuery, gotHook, gotTS, gotSession string
		gotHaikuRelevant                     sql.NullBool
		gotHaikuTag                          sql.NullString
		gotScore                             sql.NullFloat64
		gotSynthesize                        sql.NullBool
		gotFaith                             sql.NullFloat64
		gotOutcome, gotFeedback              sql.NullString
		gotE5, gotCP                         float64
	)

	err = db.QueryRow(`
		SELECT memory_id, query_text, hook_event, timestamp, session_id,
		       haiku_relevant, haiku_tag, haiku_relevance_score, should_synthesize,
		       faithfulness, outcome_signal, user_feedback, e5_similarity, context_precision
		FROM surfacing_events WHERE id = ?`, id).Scan(
		&gotMemID, &gotQuery, &gotHook, &gotTS, &gotSession,
		&gotHaikuRelevant, &gotHaikuTag, &gotScore, &gotSynthesize,
		&gotFaith, &gotOutcome, &gotFeedback, &gotE5, &gotCP,
	)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(gotMemID).To(Equal(memoryID))
	g.Expect(gotQuery).To(Equal("what is X?"))
	g.Expect(gotHook).To(Equal("PreToolUse"))
	g.Expect(gotTS).To(Equal(now.Format(time.RFC3339)))
	g.Expect(gotSession).To(Equal("sess-1"))
	g.Expect(gotHaikuRelevant.Valid).To(BeTrue())
	g.Expect(gotHaikuRelevant.Bool).To(BeTrue())
	g.Expect(gotHaikuTag.String).To(Equal("relevant"))
	g.Expect(gotScore.Valid).To(BeTrue())
	g.Expect(gotScore.Float64).To(BeNumerically("~", 0.85, 0.001))
	g.Expect(gotSynthesize.Valid).To(BeTrue())
	g.Expect(gotSynthesize.Bool).To(BeFalse())
	g.Expect(gotFaith.Valid).To(BeTrue())
	g.Expect(gotFaith.Float64).To(BeNumerically("~", 0.9, 0.001))
	g.Expect(gotOutcome.String).To(Equal("positive"))
	g.Expect(gotFeedback.String).To(Equal("helpful"))
	g.Expect(gotE5).To(BeNumerically("~", 0.75, 0.001))
	g.Expect(gotCP).To(BeNumerically("~", 0.8, 0.001))
}

func TestMetadataDefaultKeys(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Verify default metadata keys per data-model.md
	expectedDefaults := map[string]string{
		"alpha_weight":         "0.5",
		"leech_threshold":      "5",
		"importance_threshold": "0.0",
		"impact_threshold":     "0.3",
		"last_autotune_at":     "",
	}
	for key, expectedValue := range expectedDefaults {
		value, err := getMetadata(db, key)
		g.Expect(err).ToNot(HaveOccurred(), "metadata key %s", key)
		g.Expect(value).To(Equal(expectedValue), "metadata key %s should have default %s", key, expectedValue)
	}
}

func TestRecordMemoryFeedback_Helpful(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source, impact_score) VALUES ('mem', 'test', 0.5)")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	relevant := true
	event := SurfacingEvent{MemoryID: memoryID, QueryText: "q1", HookEvent: "evt", Timestamp: time.Now().UTC(), SessionID: "s1", HaikuRelevant: &relevant}
	_, err = LogSurfacingEvent(db, event)
	g.Expect(err).ToNot(HaveOccurred())

	err = RecordMemoryFeedback(db, "s1", "helpful")
	g.Expect(err).ToNot(HaveOccurred())

	var impactScore float64

	err = db.QueryRow("SELECT impact_score FROM embeddings WHERE id = ?", memoryID).Scan(&impactScore)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(impactScore).To(BeNumerically("~", 0.6, 0.001))
}

func TestRecordMemoryFeedback_HelpfulCapped(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source, impact_score) VALUES ('mem', 'test', 0.95)")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	relevant := true
	event := SurfacingEvent{MemoryID: memoryID, QueryText: "q1", HookEvent: "evt", Timestamp: time.Now().UTC(), SessionID: "s1", HaikuRelevant: &relevant}
	_, err = LogSurfacingEvent(db, event)
	g.Expect(err).ToNot(HaveOccurred())

	err = RecordMemoryFeedback(db, "s1", "helpful")
	g.Expect(err).ToNot(HaveOccurred())

	var impactScore float64

	err = db.QueryRow("SELECT impact_score FROM embeddings WHERE id = ?", memoryID).Scan(&impactScore)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(impactScore).To(BeNumerically("<=", 1.0))
	g.Expect(impactScore).To(BeNumerically("~", 1.0, 0.001))
}

func TestRecordMemoryFeedback_InvalidType(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	err = RecordMemoryFeedback(db, "s1", "bogus")
	g.Expect(err).To(HaveOccurred())
}

func TestRecordMemoryFeedback_Unclear(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source, impact_score) VALUES ('mem', 'test', 0.0)")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	relevant := true
	event := SurfacingEvent{MemoryID: memoryID, QueryText: "q1", HookEvent: "evt", Timestamp: time.Now().UTC(), SessionID: "s1", HaikuRelevant: &relevant}
	_, err = LogSurfacingEvent(db, event)
	g.Expect(err).ToNot(HaveOccurred())

	err = RecordMemoryFeedback(db, "s1", "unclear")
	g.Expect(err).ToNot(HaveOccurred())

	var impactScore float64

	err = db.QueryRow("SELECT impact_score FROM embeddings WHERE id = ?", memoryID).Scan(&impactScore)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(impactScore).To(BeNumerically("~", -0.05, 0.001))
}

func TestRecordMemoryFeedback_Wrong(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source, impact_score) VALUES ('mem', 'test', 0.0)")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	relevant := true
	event := SurfacingEvent{MemoryID: memoryID, QueryText: "q1", HookEvent: "evt", Timestamp: time.Now().UTC(), SessionID: "s1", HaikuRelevant: &relevant}
	_, err = LogSurfacingEvent(db, event)
	g.Expect(err).ToNot(HaveOccurred())

	err = RecordMemoryFeedback(db, "s1", "wrong")
	g.Expect(err).ToNot(HaveOccurred())

	var impactScore float64

	err = db.QueryRow("SELECT impact_score FROM embeddings WHERE id = ?", memoryID).Scan(&impactScore)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(impactScore).To(BeNumerically("~", -0.2, 0.001))
}

func TestRecordMemoryFeedback_WrongFloored(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source, impact_score) VALUES ('mem', 'test', -0.9)")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	relevant := true
	event := SurfacingEvent{MemoryID: memoryID, QueryText: "q1", HookEvent: "evt", Timestamp: time.Now().UTC(), SessionID: "s1", HaikuRelevant: &relevant}
	_, err = LogSurfacingEvent(db, event)
	g.Expect(err).ToNot(HaveOccurred())

	err = RecordMemoryFeedback(db, "s1", "wrong")
	g.Expect(err).ToNot(HaveOccurred())

	var impactScore float64

	err = db.QueryRow("SELECT impact_score FROM embeddings WHERE id = ?", memoryID).Scan(&impactScore)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(impactScore).To(BeNumerically(">=", -1.0))
	g.Expect(impactScore).To(BeNumerically("~", -1.0, 0.001))
}

func TestSurfacingEventsIndexes(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Verify indexes exist per data-model.md
	indexes := []string{
		"idx_surfacing_memory",
		"idx_surfacing_timestamp",
		"idx_surfacing_session",
	}
	for _, idx := range indexes {
		var count int

		err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx,
		).Scan(&count)
		g.Expect(err).ToNot(HaveOccurred(), "checking index %s", idx)
		g.Expect(count).To(Equal(1), "surfacing_events should have index %s", idx)
	}
}

func TestSurfacingEventsTableExists(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Verify surfacing_events table exists with all columns per data-model.md
	columns := []string{
		"id", "memory_id", "query_text", "hook_event", "timestamp",
		"session_id", "haiku_relevant", "haiku_tag", "haiku_relevance_score",
		"should_synthesize", "faithfulness", "outcome_signal", "user_feedback",
		"e5_similarity", "context_precision",
	}
	for _, col := range columns {
		var count int

		err := db.QueryRow(
			"SELECT COUNT(*) FROM pragma_table_info('surfacing_events') WHERE name = ?", col,
		).Scan(&count)
		g.Expect(err).ToNot(HaveOccurred(), "checking column %s", col)
		g.Expect(count).To(Equal(1), "surfacing_events should have column %s", col)
	}
}

func TestUpdateSurfacingFeedback_FindsMostRecent(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES ('mem', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	relevant := true
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Insert two events with haiku_relevant=true; second is more recent
	older := SurfacingEvent{MemoryID: memoryID, QueryText: "q1", HookEvent: "evt", Timestamp: base, SessionID: "s1", HaikuRelevant: &relevant}
	newer := SurfacingEvent{MemoryID: memoryID, QueryText: "q2", HookEvent: "evt", Timestamp: base.Add(time.Minute), SessionID: "s1", HaikuRelevant: &relevant}

	olderID, err := LogSurfacingEvent(db, older)
	g.Expect(err).ToNot(HaveOccurred())
	newerID, err := LogSurfacingEvent(db, newer)
	g.Expect(err).ToNot(HaveOccurred())

	err = UpdateSurfacingFeedback(db, "s1", "helpful")
	g.Expect(err).ToNot(HaveOccurred())

	// Only the most recent event should be updated
	var newerFeedback, olderFeedback sql.NullString

	err = db.QueryRow("SELECT user_feedback FROM surfacing_events WHERE id = ?", newerID).Scan(&newerFeedback)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(newerFeedback.String).To(Equal("helpful"))

	err = db.QueryRow("SELECT user_feedback FROM surfacing_events WHERE id = ?", olderID).Scan(&olderFeedback)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(olderFeedback.String).ToNot(Equal("helpful"))
}

func TestUpdateSurfacingFeedback_LastWins(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES ('mem', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	relevant := true
	event := SurfacingEvent{MemoryID: memoryID, QueryText: "q1", HookEvent: "evt", Timestamp: time.Now().UTC(), SessionID: "s1", HaikuRelevant: &relevant}
	id, err := LogSurfacingEvent(db, event)
	g.Expect(err).ToNot(HaveOccurred())

	err = UpdateSurfacingFeedback(db, "s1", "helpful")
	g.Expect(err).ToNot(HaveOccurred())
	err = UpdateSurfacingFeedback(db, "s1", "wrong")
	g.Expect(err).ToNot(HaveOccurred())

	var feedback sql.NullString

	err = db.QueryRow("SELECT user_feedback FROM surfacing_events WHERE id = ?", id).Scan(&feedback)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(feedback.String).To(Equal("wrong"))
}

func TestUpdateSurfacingFeedback_NoEventError(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	err = UpdateSurfacingFeedback(db, "nonexistent-session", "helpful")
	g.Expect(err).To(HaveOccurred())
}

func TestUpdateSurfacingFeedback_SkipsNonRelevant(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES ('mem', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memoryID, _ := res.LastInsertId()

	// Insert events with haiku_relevant=false or nil
	notRelevant := false
	e1 := SurfacingEvent{MemoryID: memoryID, QueryText: "q1", HookEvent: "evt", Timestamp: time.Now().UTC(), SessionID: "s1", HaikuRelevant: &notRelevant}
	e2 := SurfacingEvent{MemoryID: memoryID, QueryText: "q2", HookEvent: "evt", Timestamp: time.Now().UTC(), SessionID: "s1"} // nil HaikuRelevant

	_, err = LogSurfacingEvent(db, e1)
	g.Expect(err).ToNot(HaveOccurred())
	_, err = LogSurfacingEvent(db, e2)
	g.Expect(err).ToNot(HaveOccurred())

	err = UpdateSurfacingFeedback(db, "s1", "helpful")
	g.Expect(err).To(HaveOccurred())
}
