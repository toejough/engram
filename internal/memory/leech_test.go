package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestApplyLeechAction_ConvertToHook_MarksActionRecommended(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "test memory", "leech", 5, 0.8, 0.1)

	rec := &Recommendation{
		Category:    "hook-conversion",
		Description: "Convert to hook: enforce test memory rule",
		Evidence:    "Agent references but user corrects",
	}
	diagnosis := LeechDiagnosis{
		MemoryID:       id,
		Content:        "test memory",
		DiagnosisType:  "enforcement_gap",
		ProposedAction: "convert_to_hook",
		Recommendation: rec,
	}

	err := ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify action_recommended is marked
	var leechAction string

	err = db.QueryRow("SELECT leech_action FROM embeddings WHERE id = ?", id).Scan(&leechAction)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(leechAction).To(Equal("action_recommended"))
}

func TestApplyLeechAction_NarrowScope_FailsWhenNoEmbeddingRow(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "broad content", "leech", 5, 0.8, 0.1)

	diagnosis := LeechDiagnosis{
		MemoryID:         id,
		Content:          "broad content",
		DiagnosisType:    "retrieval_mismatch",
		ProposedAction:   "narrow_scope",
		SuggestedContent: "narrow and specific content",
	}

	err := ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).To(HaveOccurred(), "Should fail when memory has no embedding row")

	// Content should be preserved
	var content string

	dbErr := db.QueryRow("SELECT content FROM embeddings WHERE id = ?", id).Scan(&content)
	g.Expect(dbErr).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("broad content"), "Content should be preserved after rollback")
}

// ─── ApplyLeechAction tests ───────────────────────────────────────────────────

func TestApplyLeechAction_PromoteToClaudeMD_MarksActionRecommended(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "test memory", "leech", 5, 0.8, 0.1)

	rec := &Recommendation{
		Category:    "claude-md-promotion",
		Description: "Add to CLAUDE.md: test memory",
		Evidence:    "Consistently surfaced too late to prevent mistakes",
	}
	diagnosis := LeechDiagnosis{
		MemoryID:       id,
		Content:        "test memory",
		DiagnosisType:  "wrong_tier",
		ProposedAction: "promote_to_claude_md",
		Recommendation: rec,
	}

	err := ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify action_recommended is marked
	var leechAction string

	err = db.QueryRow("SELECT leech_action FROM embeddings WHERE id = ?", id).Scan(&leechAction)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(leechAction).To(Equal("action_recommended"))
}

func TestApplyLeechAction_Rewrite_FailsWhenNoEmbeddingRow(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	// Memory inserted without a vec_embeddings row (no ONNX model needed for this test)
	id := insertLeechMemory(t, db, "original content", "leech", 5, 0.8, 0.1)

	diagnosis := LeechDiagnosis{
		MemoryID:         id,
		Content:          "original content",
		DiagnosisType:    "content_quality",
		ProposedAction:   "rewrite",
		SuggestedContent: "improved content",
	}

	// Rewrite requires re-embedding; fails because memory has no embedding_id
	err := ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).To(HaveOccurred(), "Should fail when memory has no embedding row")

	// Content should be preserved (transaction rolled back)
	var content string

	dbErr := db.QueryRow("SELECT content FROM embeddings WHERE id = ?", id).Scan(&content)
	g.Expect(dbErr).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("original content"), "Content should be preserved after rollback")
}

func TestDiagnoseLeech_ContentQualityDiagnosis(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "test memory about coding", "leech", 5, 0.8, 0.1)

	ts := time.Now().UTC().Truncate(time.Second)
	notRelevant := false
	f := 0.1 // low faithfulness

	// 3 events with low faithfulness, negative outcome, no user correction feedback
	for i := range 3 {
		insertLeechSurfacingEvent(t, db, id, fmt.Sprintf("sess-%d", i), ts.Add(time.Duration(i)*time.Hour), &notRelevant, &f, "negative", "")
	}

	diag, err := DiagnoseLeech(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	if diag == nil {
		t.Fatal("DiagnoseLeech returned nil diag")
	}

	g.Expect(diag.DiagnosisType).To(Equal("content_quality"))
	g.Expect(diag.ProposedAction).To(Equal("rewrite"))
	g.Expect(diag.SuggestedContent).To(BeEmpty(), "SuggestedContent should be empty from DiagnoseLeech")
	g.Expect(diag.Recommendation).To(BeNil(), "Recommendation should be nil for content_quality")
	g.Expect(diag.Signal).ToNot(BeEmpty())
}

func TestDiagnoseLeech_ContentQualityOverRetrievalMismatch(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "test memory", "leech", 5, 0.8, 0.1)

	ts := time.Now().UTC().Truncate(time.Second)
	notRelevant := false
	f := 0.1

	// Mix: >50% haiku_relevant=false AND low faithfulness on those that are relevant
	// → both retrieval_mismatch and content_quality signal, but content_quality wins
	insertLeechSurfacingEvent(t, db, id, "sess-1", ts, &notRelevant, nil, "", "")
	insertLeechSurfacingEvent(t, db, id, "sess-2", ts.Add(time.Hour), &notRelevant, nil, "", "")
	insertLeechSurfacingEvent(t, db, id, "sess-3", ts.Add(2*time.Hour), &notRelevant, &f, "negative", "")

	diag, err := DiagnoseLeech(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	if diag == nil {
		t.Fatal("DiagnoseLeech returned nil diag")
	}
	// content_quality (priority 3) should win over retrieval_mismatch (priority 4)
	g.Expect(diag.DiagnosisType).To(Equal("content_quality"))
}

func TestDiagnoseLeech_EnforcementGapDiagnosis(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "test memory", "leech", 5, 0.8, 0.1)

	ts := time.Now().UTC().Truncate(time.Second)
	// High faithfulness (agent referenced content) but user corrected anyway
	highF := 0.7
	for i := range 3 {
		insertLeechSurfacingEvent(t, db, id, fmt.Sprintf("sess-gap-%d", i), ts.Add(time.Duration(i)*time.Hour), nil, &highF, "positive", "wrong")
	}

	diag, err := DiagnoseLeech(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	if diag == nil {
		t.Fatal("DiagnoseLeech returned nil diag")
	}

	g.Expect(diag.DiagnosisType).To(Equal("enforcement_gap"))
	g.Expect(diag.ProposedAction).To(Equal("convert_to_hook"))
	g.Expect(diag.Recommendation).ToNot(BeNil())
	g.Expect(diag.Recommendation.Category).To(Equal("hook-conversion"))
	g.Expect(diag.Recommendation.Description).ToNot(BeEmpty())
	g.Expect(diag.Recommendation.Evidence).ToNot(BeEmpty())
}

// ─── DiagnoseLeech tests ──────────────────────────────────────────────────────

func TestDiagnoseLeech_ErrorWhenMemoryNotFound(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	_, err := DiagnoseLeech(db, 99999)
	g.Expect(err).To(HaveOccurred())
}

func TestDiagnoseLeech_InsufficientDataWhenNoEvents(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "test memory", "leech", 5, 0.8, 0.1)

	diag, err := DiagnoseLeech(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	if diag == nil {
		t.Fatal("DiagnoseLeech returned nil diag")
	}

	g.Expect(diag.MemoryID).To(Equal(id))
	g.Expect(diag.Content).To(Equal("test memory"))
	g.Expect(diag.DiagnosisType).To(Equal("insufficient_data"))
}

func TestDiagnoseLeech_PriorityEnforcementGapOverContent(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "test memory", "leech", 5, 0.8, 0.1)

	ts := time.Now().UTC().Truncate(time.Second)
	// High faithfulness + user correction → enforcement_gap
	// Also has enough events to match content_quality, but enforcement_gap wins
	highF := 0.7
	for i := range 3 {
		insertLeechSurfacingEvent(t, db, id, fmt.Sprintf("sess-gap-%d", i), ts.Add(time.Duration(i)*time.Hour), nil, &highF, "positive", "wrong")
	}

	diag, err := DiagnoseLeech(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	if diag == nil {
		t.Fatal("DiagnoseLeech returned nil diag")
	}
	// enforcement_gap (priority 2) should win over content_quality (priority 3)
	g.Expect(diag.DiagnosisType).To(Equal("enforcement_gap"))
}

func TestDiagnoseLeech_PriorityWrongTierOverContent(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "test memory", "leech", 5, 0.8, 0.1)

	ts := time.Now().UTC().Truncate(time.Second)
	// Events matching both wrong_tier (user_feedback=wrong) AND content_quality (low faithfulness)
	f := 0.1
	for i := range 3 {
		insertLeechSurfacingEvent(t, db, id, fmt.Sprintf("sess-%d", i), ts.Add(time.Duration(i)*time.Hour), nil, &f, "negative", "wrong")
	}

	diag, err := DiagnoseLeech(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	if diag == nil {
		t.Fatal("DiagnoseLeech returned nil diag")
	}
	// wrong_tier (priority 1) should win over content_quality (priority 3)
	g.Expect(diag.DiagnosisType).To(Equal("wrong_tier"))
}

func TestDiagnoseLeech_RetrievalMismatchDiagnosis(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "test memory", "leech", 5, 0.8, 0.1)

	ts := time.Now().UTC().Truncate(time.Second)
	notRelevant := false
	relevant := true

	// 3 events haiku_relevant=false, 1 event haiku_relevant=true → >50% false
	insertLeechSurfacingEvent(t, db, id, "sess-1", ts, &notRelevant, nil, "", "")
	insertLeechSurfacingEvent(t, db, id, "sess-2", ts.Add(time.Hour), &notRelevant, nil, "", "")
	insertLeechSurfacingEvent(t, db, id, "sess-3", ts.Add(2*time.Hour), &notRelevant, nil, "", "")
	insertLeechSurfacingEvent(t, db, id, "sess-4", ts.Add(3*time.Hour), &relevant, nil, "", "")

	diag, err := DiagnoseLeech(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	if diag == nil {
		t.Fatal("DiagnoseLeech returned nil diag")
	}

	g.Expect(diag.DiagnosisType).To(Equal("retrieval_mismatch"))
	g.Expect(diag.ProposedAction).To(Equal("narrow_scope"))
	g.Expect(diag.Recommendation).To(BeNil())
}

func TestDiagnoseLeech_WrongTierDiagnosis(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "test memory", "leech", 5, 0.8, 0.1)

	ts := time.Now().UTC().Truncate(time.Second)
	f := 0.1

	// 3 events with user_feedback='wrong' (surfaced but user corrected)
	for i := range 3 {
		insertLeechSurfacingEvent(t, db, id, fmt.Sprintf("sess-wrong-%d", i), ts.Add(time.Duration(i)*time.Hour), nil, &f, "negative", "wrong")
	}

	diag, err := DiagnoseLeech(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	if diag == nil {
		t.Fatal("DiagnoseLeech returned nil diag")
	}

	g.Expect(diag.DiagnosisType).To(Equal("wrong_tier"))
	g.Expect(diag.ProposedAction).To(Equal("promote_to_claude_md"))
	g.Expect(diag.Recommendation).ToNot(BeNil())
	g.Expect(diag.Recommendation.Category).To(Equal("claude-md-promotion"))
	g.Expect(diag.Recommendation.Description).ToNot(BeEmpty())
	g.Expect(diag.Recommendation.Evidence).ToNot(BeEmpty())
}

func TestGetLeeches_IncludesRelevantFields(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	_, err := db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('leech_threshold', '1')")
	g.Expect(err).ToNot(HaveOccurred())

	id := insertLeechMemory(t, db, "test leech", "leech", 3, 0.75, 0.15)

	ts := time.Now().UTC().Truncate(time.Second)
	f := 0.2
	insertLeechSurfacingEvent(t, db, id, "sess-1", ts, nil, &f, "negative", "wrong")
	insertLeechSurfacingEvent(t, db, id, "sess-2", ts.Add(time.Hour), nil, &f, "negative", "wrong")

	leeches, err := GetLeeches(db)
	g.Expect(err).ToNot(HaveOccurred())

	if len(leeches) < 1 {
		t.Fatal("expected at least 1 leech from GetLeeches")
	}

	g.Expect(leeches).To(HaveLen(1))
	g.Expect(leeches[0].MemoryID).To(Equal(id))
	g.Expect(leeches[0].LeechCount).To(Equal(3))
	g.Expect(leeches[0].ImportanceScore).To(BeNumerically("~", 0.75, 0.01))
	g.Expect(leeches[0].ImpactScore).To(BeNumerically("~", 0.15, 0.01))
	g.Expect(leeches[0].SurfacingCount).To(Equal(2))
}

func TestGetLeeches_ReturnsEmptyWhenNoLeeches(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	insertLeechMemory(t, db, "working memory", "working", 5, 0.8, 0.9)

	leeches, err := GetLeeches(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(leeches).To(BeEmpty())
}

// ─── GetLeeches tests ─────────────────────────────────────────────────────────

func TestGetLeeches_ReturnsLeechesAboveThreshold(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	// Set threshold to 3
	_, err := db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('leech_threshold', '3')")
	g.Expect(err).ToNot(HaveOccurred())

	// leech with count=5 (above threshold → should be returned)
	id1 := insertLeechMemory(t, db, "leech memory above", "leech", 5, 0.8, 0.1)

	// leech with count=2 (below threshold → should NOT be returned)
	insertLeechMemory(t, db, "leech memory below", "leech", 2, 0.8, 0.1)

	// working with count=5 (not a leech quadrant → should NOT be returned)
	insertLeechMemory(t, db, "working memory", "working", 5, 0.8, 0.9)

	leeches, err := GetLeeches(db)
	g.Expect(err).ToNot(HaveOccurred())

	if len(leeches) < 1 {
		t.Fatal("expected at least 1 leech from GetLeeches")
	}

	g.Expect(leeches).To(HaveLen(1))
	g.Expect(leeches[0].MemoryID).To(Equal(id1))
	g.Expect(leeches[0].Content).To(Equal("leech memory above"))
	g.Expect(leeches[0].LeechCount).To(Equal(5))
}

func TestGetLeeches_UsesMetadataThreshold(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	// Default threshold is 5; insert a leech with count=4 (below default threshold)
	insertLeechMemory(t, db, "leech below default", "leech", 4, 0.8, 0.1)

	leeches, err := GetLeeches(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(leeches).To(BeEmpty(), "leech_count=4 should be below default threshold of 5")

	// Insert one with count=5 (at default threshold)
	id := insertLeechMemory(t, db, "leech at threshold", "leech", 5, 0.8, 0.1)

	leeches, err = GetLeeches(db)
	g.Expect(err).ToNot(HaveOccurred())

	if len(leeches) < 1 {
		t.Fatal("expected at least 1 leech from GetLeeches")
	}

	g.Expect(leeches).To(HaveLen(1))
	g.Expect(leeches[0].MemoryID).To(Equal(id))
}

func TestPreviewLeechRewrite_ErrorOnNonContentQuality(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	diagnosis := LeechDiagnosis{
		MemoryID:       1,
		Content:        "test memory",
		DiagnosisType:  "wrong_tier",
		ProposedAction: "promote_to_claude_md",
	}

	llm := NewDirectAPIExtractor("test-token", WithBaseURL("http://localhost:1"))

	_, err := PreviewLeechRewrite(db, diagnosis, llm)
	g.Expect(err).To(HaveOccurred(), "Should return error for non-content_quality diagnosis")
}

func TestPreviewLeechRewrite_ErrorOnRetrievalMismatch(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	diagnosis := LeechDiagnosis{
		MemoryID:       1,
		Content:        "test memory",
		DiagnosisType:  "retrieval_mismatch",
		ProposedAction: "narrow_scope",
	}

	llm := NewDirectAPIExtractor("test-token", WithBaseURL("http://localhost:1"))

	_, err := PreviewLeechRewrite(db, diagnosis, llm)
	g.Expect(err).To(HaveOccurred(), "Should return error for non-content_quality diagnosis")
}

func TestPreviewLeechRewrite_ReturnsEmptyOnLLMFailure(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "original content", "leech", 5, 0.8, 0.1)

	diagnosis := LeechDiagnosis{
		MemoryID:       id,
		Content:        "original content",
		DiagnosisType:  "content_quality",
		Signal:         "low faithfulness",
		ProposedAction: "rewrite",
	}

	// Use unreachable URL to simulate LLM failure
	llm := NewDirectAPIExtractor("test-token", WithBaseURL("http://localhost:1"), WithTimeout(time.Millisecond*100))

	result, err := PreviewLeechRewrite(db, diagnosis, llm)
	g.Expect(err).ToNot(HaveOccurred(), "LLM failure should not return error")
	g.Expect(result).To(BeEmpty(), "Should return empty string on LLM failure")
}

// ─── PreviewLeechRewrite tests ────────────────────────────────────────────────

func TestPreviewLeechRewrite_ReturnsRewriteForContentQuality(t *testing.T) {
	g := NewWithT(t)
	db := leechTestDB(t)

	id := insertLeechMemory(t, db, "original content", "leech", 5, 0.8, 0.1)

	diagnosis := LeechDiagnosis{
		MemoryID:       id,
		Content:        "original content",
		DiagnosisType:  "content_quality",
		Signal:         "low faithfulness",
		ProposedAction: "rewrite",
	}

	server := makeLeechRewriteServer("improved and clearer content")
	defer server.Close()

	llm := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	result, err := PreviewLeechRewrite(db, diagnosis, llm)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeEmpty())
}

// insertLeechMemory inserts a memory with specified leech status and returns ID.
func insertLeechMemory(t *testing.T, db *sql.DB, content, quadrant string, leechCount int, importance, impact float64) int64 {
	t.Helper()

	res, err := db.Exec(
		"INSERT INTO embeddings (content, source, quadrant, leech_count, importance_score, impact_score) VALUES (?, 'test', ?, ?, ?, ?)",
		content, quadrant, leechCount, importance, impact,
	)
	if err != nil {
		t.Fatalf("INSERT embeddings: %v", err)
	}

	id, _ := res.LastInsertId()

	return id
}

// insertLeechSurfacingEvent inserts a surfacing event for leech tests.
func insertLeechSurfacingEvent(t *testing.T, db *sql.DB, memID int64, sessionID string, ts time.Time, haikuRelevant *bool, faithfulness *float64, outcomeSignal, userFeedback string) int64 {
	t.Helper()

	event := SurfacingEvent{
		MemoryID:      memID,
		QueryText:     "test query",
		HookEvent:     "PreToolUse",
		Timestamp:     ts,
		SessionID:     sessionID,
		HaikuRelevant: haikuRelevant,
		Faithfulness:  faithfulness,
		OutcomeSignal: outcomeSignal,
		UserFeedback:  userFeedback,
	}

	id, err := LogSurfacingEvent(db, event)
	if err != nil {
		t.Fatalf("LogSurfacingEvent: %v", err)
	}

	return id
}

// leechTestDB creates a test DB for leech tests.
func leechTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		t.Fatalf("initEmbeddingsDB: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return db
}

// makeLeechRewriteServer creates a mock LLM server that returns a fixed rewrite response.
func makeLeechRewriteServer(rewrittenContent string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"model": "claude-haiku-4-5-20251001",
			"content": []map[string]any{
				{"type": "text", "text": rewrittenContent},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
}
