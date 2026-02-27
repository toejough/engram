package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	. "github.com/onsi/gomega"
)

// TestInstallHooks_ExistingFile verifies InstallHooks merges into existing settings.
func TestInstallHooks_ExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settingsPath := filepath.Join(t.TempDir(), "settings.json")

	existing := map[string]any{"model": "claude-sonnet-4-5-20250514"}

	data, err := json.Marshal(existing)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(settingsPath, data, 0600)
	g.Expect(err).ToNot(HaveOccurred())

	err = InstallHooks(InstallHooksOpts{SettingsPath: settingsPath})
	g.Expect(err).ToNot(HaveOccurred())

	data, err = os.ReadFile(settingsPath)
	g.Expect(err).ToNot(HaveOccurred())

	var settings map[string]any

	err = json.Unmarshal(data, &settings)
	g.Expect(err).ToNot(HaveOccurred())

	// Existing key preserved
	g.Expect(settings).To(HaveKey("model"))
	// Hooks added
	g.Expect(settings).To(HaveKey("hooks"))
}

// TestInstallHooks_NewFile verifies InstallHooks creates a settings file with hooks.
func TestInstallHooks_NewFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settingsPath := filepath.Join(t.TempDir(), "settings.json")

	err := InstallHooks(InstallHooksOpts{SettingsPath: settingsPath})
	g.Expect(err).ToNot(HaveOccurred())

	data, err := os.ReadFile(settingsPath)
	g.Expect(err).ToNot(HaveOccurred())

	var settings map[string]any

	err = json.Unmarshal(data, &settings)
	g.Expect(err).ToNot(HaveOccurred())

	hooks, ok := settings["hooks"].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(hooks).To(HaveKey("Stop"))
	g.Expect(hooks).To(HaveKey("SessionStart"))
}

// TestRunFilterPipelineBasic verifies filter results, surfacing events, and formatted output.
func TestRunFilterPipelineBasic(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES ('commit trailer format', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID1, _ := res.LastInsertId()

	res, err = db.Exec("INSERT INTO embeddings (content, source) VALUES ('noise memory', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID2, _ := res.LastInsertId()

	filterJSON := fmt.Sprintf(`[
		{"memory_id": %d, "relevant": true, "tag": "relevant", "relevance_score": 0.9, "should_synthesize": false},
		{"memory_id": %d, "relevant": false, "tag": "noise", "relevance_score": 0.1, "should_synthesize": false}
	]`, memID1, memID2)

	server := makeFilterServer(filterJSON)
	defer server.Close()

	extractor := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	candidates := []QueryResult{
		{ID: memID1, Content: "commit trailer format", Score: 0.85},
		{ID: memID2, Content: "noise memory", Score: 0.5},
	}

	output := RunFilterPipeline(context.Background(), RunFilterPipelineOpts{
		DB:           db,
		Extractor:    extractor,
		QueryResults: candidates,
		QueryText:    "create a commit",
		HookEvent:    "UserPromptSubmit",
		SessionID:    "sess-basic",
	})

	g.Expect(output).To(ContainSubstring("commit trailer format"))
	g.Expect(output).ToNot(ContainSubstring("noise memory"))

	// Surfacing events logged for ALL candidates
	events, err := GetSessionSurfacingEvents(db, "sess-basic")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(events).To(HaveLen(2))

	// Relevant event has haiku fields populated
	relevantEvent := findEventByMemoryID(events, memID1)
	g.Expect(relevantEvent).ToNot(BeNil())

	if relevantEvent == nil {
		t.Fatal("relevantEvent is nil")
	}

	g.Expect(relevantEvent.HaikuRelevant).ToNot(BeNil())
	g.Expect(*relevantEvent.HaikuRelevant).To(BeTrue())
	g.Expect(relevantEvent.E5Similarity).To(BeNumerically("~", 0.85, 0.01))
	g.Expect(relevantEvent.ContextPrecision).To(BeNumerically("~", 0.5, 0.01))

	// Noise event has HaikuRelevant=false
	noiseEvent := findEventByMemoryID(events, memID2)
	g.Expect(noiseEvent).ToNot(BeNil())

	if noiseEvent == nil {
		t.Fatal("noiseEvent is nil")
	}

	g.Expect(noiseEvent.HaikuRelevant).ToNot(BeNil())
	g.Expect(*noiseEvent.HaikuRelevant).To(BeFalse())
	g.Expect(noiseEvent.E5Similarity).To(BeNumerically("~", 0.5, 0.01))
}

// TestRunFilterPipelineDegradedFilter verifies surfacing events have NULL haiku fields on degradation.
func TestRunFilterPipelineDegradedFilter(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES ('mem a', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID1, _ := res.LastInsertId()

	res, err = db.Exec("INSERT INTO embeddings (content, source) VALUES ('mem b', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID2, _ := res.LastInsertId()

	// Server returns malformed JSON → graceful degradation (RelevanceScore=-1.0)
	server := makeFilterServer("not a valid json array")
	defer server.Close()

	extractor := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	candidates := []QueryResult{
		{ID: memID1, Content: "mem a", Score: 0.8},
		{ID: memID2, Content: "mem b", Score: 0.75},
	}

	_ = RunFilterPipeline(context.Background(), RunFilterPipelineOpts{
		DB:           db,
		Extractor:    extractor,
		QueryResults: candidates,
		QueryText:    "test query",
		HookEvent:    "UserPromptSubmit",
		SessionID:    "sess-degraded",
	})

	// Surfacing events logged with NULL haiku fields
	events, err := GetSessionSurfacingEvents(db, "sess-degraded")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(events).To(HaveLen(2))

	for _, e := range events {
		g.Expect(e.HaikuRelevant).To(BeNil(), "haiku_relevant should be NULL for degraded results")
		g.Expect(e.HaikuRelevanceScore).To(BeNil(), "haiku_relevance_score should be NULL for degraded results")
		g.Expect(e.ShouldSynthesize).To(BeNil(), "should_synthesize should be NULL for degraded results")
	}
}

// TestRunFilterPipelineEmptyInput verifies no API call and empty output for no candidates.
func TestRunFilterPipelineEmptyInput(t *testing.T) {
	g := NewWithT(t)

	serverCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
	}))
	defer server.Close()

	extractor := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	output := RunFilterPipeline(context.Background(), RunFilterPipelineOpts{
		DB:           nil,
		Extractor:    extractor,
		QueryResults: nil,
		QueryText:    "test query",
		HookEvent:    "UserPromptSubmit",
		SessionID:    "sess-empty",
	})

	g.Expect(output).To(BeEmpty())
	g.Expect(serverCalled).To(BeFalse())
}

// TestRunFilterPipelineNoRelevant verifies empty output + events when all filtered out.
func TestRunFilterPipelineNoRelevant(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES ('noise 1', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID1, _ := res.LastInsertId()

	res, err = db.Exec("INSERT INTO embeddings (content, source) VALUES ('noise 2', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID2, _ := res.LastInsertId()

	filterJSON := fmt.Sprintf(`[
		{"memory_id": %d, "relevant": false, "tag": "noise", "relevance_score": 0.2, "should_synthesize": false},
		{"memory_id": %d, "relevant": false, "tag": "noise", "relevance_score": 0.1, "should_synthesize": false}
	]`, memID1, memID2)

	server := makeFilterServer(filterJSON)
	defer server.Close()

	extractor := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	candidates := []QueryResult{
		{ID: memID1, Content: "noise 1", Score: 0.7},
		{ID: memID2, Content: "noise 2", Score: 0.65},
	}

	output := RunFilterPipeline(context.Background(), RunFilterPipelineOpts{
		DB:           db,
		Extractor:    extractor,
		QueryResults: candidates,
		QueryText:    "completely unrelated query",
		HookEvent:    "SessionStart",
		SessionID:    "sess-norelevant",
	})

	g.Expect(output).To(BeEmpty())

	// Surfacing events still logged despite no relevant results
	events, err := GetSessionSurfacingEvents(db, "sess-norelevant")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(events).To(HaveLen(2))
}

// TestRunFilterPipelineSynthesis verifies that 2+ ShouldSynthesize=true triggers Synthesize().
func TestRunFilterPipelineSynthesis(t *testing.T) {
	g := NewWithT(t)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES ('commit trailer', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID1, _ := res.LastInsertId()

	res, err = db.Exec("INSERT INTO embeddings (content, source) VALUES ('commit message style', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID2, _ := res.LastInsertId()

	res, err = db.Exec("INSERT INTO embeddings (content, source) VALUES ('unrelated memory', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	memID3, _ := res.LastInsertId()

	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)

		w.Header().Set("Content-Type", "application/json")

		var text string
		if n == 1 {
			text = fmt.Sprintf(`[
				{"memory_id": %d, "relevant": true, "tag": "relevant", "relevance_score": 0.9, "should_synthesize": true},
				{"memory_id": %d, "relevant": true, "tag": "relevant", "relevance_score": 0.85, "should_synthesize": true},
				{"memory_id": %d, "relevant": false, "tag": "noise", "relevance_score": 0.1, "should_synthesize": false}
			]`, memID1, memID2, memID3)
		} else {
			text = "Use conventional commit format with proper attribution"
		}

		response := map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"model": "claude-haiku-4-5-20251001",
			"content": []map[string]any{
				{"type": "text", "text": text},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	extractor := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	candidates := []QueryResult{
		{ID: memID1, Content: "commit trailer", Score: 0.9},
		{ID: memID2, Content: "commit message style", Score: 0.85},
		{ID: memID3, Content: "unrelated memory", Score: 0.4},
	}

	output := RunFilterPipeline(context.Background(), RunFilterPipelineOpts{
		DB:           db,
		Extractor:    extractor,
		QueryResults: candidates,
		QueryText:    "create a commit",
		HookEvent:    "UserPromptSubmit",
		SessionID:    "sess-synth",
	})

	// Output contains synthesized text (not individual memories)
	g.Expect(output).To(ContainSubstring("Use conventional commit format"))
	g.Expect(output).ToNot(ContainSubstring("1. commit trailer"))

	// Synthesize was called (total calls = 2)
	g.Expect(atomic.LoadInt32(&callCount)).To(Equal(int32(2)))

	// Surfacing events for all 3 candidates
	events, err := GetSessionSurfacingEvents(db, "sess-synth")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(events).To(HaveLen(3))
}

// TestShowHooks_NoFile verifies ShowHooks returns empty JSON when no settings file exists.
func TestShowHooks_NoFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	output, err := ShowHooks(ShowHooksOpts{SettingsPath: filepath.Join(t.TempDir(), "missing.json")})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(output).To(Equal("{}"))
}

// TestShowHooks_WithHooks verifies ShowHooks returns installed hooks as JSON.
func TestShowHooks_WithHooks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settingsPath := filepath.Join(t.TempDir(), "settings.json")

	// Install hooks first, then show them
	err := InstallHooks(InstallHooksOpts{SettingsPath: settingsPath})
	g.Expect(err).ToNot(HaveOccurred())

	output, err := ShowHooks(ShowHooksOpts{SettingsPath: settingsPath})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(output).To(ContainSubstring("Stop"))
	g.Expect(output).To(ContainSubstring("SessionStart"))
}

// findEventByMemoryID returns the surfacing event with the given memory ID, or nil.
func findEventByMemoryID(events []SurfacingEvent, memoryID int64) *SurfacingEvent {
	for i, e := range events {
		if e.MemoryID == memoryID {
			return &events[i]
		}
	}

	return nil
}

func makeFilterServer(filterJSON string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"model": "claude-haiku-4-5-20251001",
			"content": []map[string]any{
				{"type": "text", "text": filterJSON},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
}
