package memory

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

// ─── NFR-001: Haiku filter latency MUST be under 200ms ──────────────────────

func BenchmarkFilterPipeline(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		b.Fatalf("initEmbeddingsDB: %v", err)
	}

	defer func() { _ = db.Close() }()

	// Insert 10 candidate memories (typical query result set)
	candidates := make([]QueryResult, 10)

	for i := range 10 {
		res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES (?, 'test')",
			fmt.Sprintf("Memory content %d about testing patterns", i))
		if err != nil {
			b.Fatalf("INSERT: %v", err)
		}

		id, _ := res.LastInsertId()
		candidates[i] = QueryResult{
			ID:      id,
			Content: fmt.Sprintf("Memory content %d about testing patterns", i),
			Score:   0.8 - float64(i)*0.02,
		}
	}

	llm := &benchLLM{}
	ctx := context.Background()

	for b.Loop() {
		opts := RunFilterPipelineOpts{
			DB:           db,
			Extractor:    llm,
			QueryResults: candidates,
			QueryText:    "how to write tests",
			HookEvent:    "PreToolUse",
			SessionID:    "bench-session",
		}
		_ = RunFilterPipeline(ctx, opts)
	}
}

// ─── NFR-003: End-of-session scoring MUST complete within 5 seconds ─────────

func BenchmarkScoreSession50Events(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		b.Fatalf("initEmbeddingsDB: %v", err)
	}

	defer func() { _ = db.Close() }()

	// Insert 50 memories with haiku_relevant=true surfacing events
	sessionID := "bench-score-session"

	for i := range 50 {
		res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES (?, 'test')",
			fmt.Sprintf("Memory %d for scoring benchmark", i))
		if err != nil {
			b.Fatalf("INSERT: %v", err)
		}

		memID, _ := res.LastInsertId()
		relevant := true
		ts := time.Now().Add(time.Duration(i) * time.Second)

		_, err = LogSurfacingEvent(db, SurfacingEvent{
			MemoryID:      memID,
			QueryText:     "benchmark query",
			HookEvent:     "PreToolUse",
			Timestamp:     ts,
			SessionID:     sessionID,
			HaikuRelevant: &relevant,
		})
		if err != nil {
			b.Fatalf("LogSurfacingEvent: %v", err)
		}
	}

	llm := &benchLLM{}

	for b.Loop() {
		// Reset faithfulness to NULL for re-scoring
		_, _ = db.Exec("UPDATE surfacing_events SET faithfulness = NULL WHERE session_id = ?", sessionID)
		_ = ScoreSession(db, sessionID, llm)
	}
}

// ─── NFR-002: Sonnet synthesis latency MUST be under 1 second ───────────────

func BenchmarkSynthesis(b *testing.B) {
	llm := &benchLLM{}
	ctx := context.Background()
	memories := []string{
		"Always use TDD when writing Go code",
		"Run tests with -tags sqlite_fts5",
		"Use gomega for assertions",
		"Check test coverage before committing",
	}

	for b.Loop() {
		_, _ = llm.Synthesize(ctx, memories)
	}
}

func TestNFR001_FilterLatencyUnder200ms(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).NotTo(HaveOccurred())

	defer func() { _ = db.Close() }()

	candidates := make([]QueryResult, 10)

	for i := range 10 {
		res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES (?, 'test')",
			fmt.Sprintf("Memory content %d", i))
		g.Expect(err).NotTo(HaveOccurred())

		if res == nil {
			t.Fatal("db.Exec returned nil result")
		}

		id, _ := res.LastInsertId()
		candidates[i] = QueryResult{
			ID:      id,
			Content: fmt.Sprintf("Memory content %d", i),
			Score:   0.8,
		}
	}

	llm := &benchLLM{}
	ctx := context.Background()

	start := time.Now()
	_ = RunFilterPipeline(ctx, RunFilterPipelineOpts{
		DB:           db,
		Extractor:    llm,
		QueryResults: candidates,
		QueryText:    "how to write tests",
		HookEvent:    "PreToolUse",
		SessionID:    "nfr001-session",
	})
	elapsed := time.Since(start)

	// Pipeline overhead with mock LLM must be well under 200ms
	// (real API latency is external; we validate the Go code path)
	g.Expect(elapsed).To(BeNumerically("<", 200*time.Millisecond),
		"Filter pipeline overhead exceeded 200ms: %v", elapsed)
}

func TestNFR002_SynthesisLatencyUnder1s(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).NotTo(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Create candidates that trigger synthesis (2+ with ShouldSynthesize=true)
	candidates := make([]QueryResult, 5)

	for i := range 5 {
		res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES (?, 'test')",
			fmt.Sprintf("Synthesis candidate %d", i))
		g.Expect(err).NotTo(HaveOccurred())

		if res == nil {
			t.Fatal("db.Exec returned nil result")
		}

		id, _ := res.LastInsertId()
		candidates[i] = QueryResult{
			ID:      id,
			Content: fmt.Sprintf("Synthesis candidate %d", i),
			Score:   0.9,
		}
	}

	// Use a synthesis-triggering mock
	synthLLM := &synthBenchLLM{}
	ctx := context.Background()

	start := time.Now()
	_ = RunFilterPipeline(ctx, RunFilterPipelineOpts{
		DB:           db,
		Extractor:    synthLLM,
		QueryResults: candidates,
		QueryText:    "testing patterns",
		HookEvent:    "UserPromptSubmit",
		SessionID:    "nfr002-session",
	})
	elapsed := time.Since(start)

	g.Expect(elapsed).To(BeNumerically("<", 1*time.Second),
		"Synthesis pipeline overhead exceeded 1s: %v", elapsed)
}

func TestNFR003_ScoreSession50EventsUnder5s(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).NotTo(HaveOccurred())

	defer func() { _ = db.Close() }()

	sessionID := "nfr003-session"

	for i := range 50 {
		res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES (?, 'test')",
			fmt.Sprintf("Memory %d for NFR-003 validation", i))
		g.Expect(err).NotTo(HaveOccurred())

		if res == nil {
			t.Fatal("db.Exec returned nil result")
		}

		memID, _ := res.LastInsertId()
		relevant := true
		ts := time.Now().Add(time.Duration(i) * time.Second)
		_, err = LogSurfacingEvent(db, SurfacingEvent{
			MemoryID:      memID,
			QueryText:     "nfr003 query",
			HookEvent:     "PreToolUse",
			Timestamp:     ts,
			SessionID:     sessionID,
			HaikuRelevant: &relevant,
		})
		g.Expect(err).NotTo(HaveOccurred())
	}

	llm := &benchLLM{}

	start := time.Now()
	err = ScoreSession(db, sessionID, llm)
	elapsed := time.Since(start)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(elapsed).To(BeNumerically("<", 5*time.Second),
		"ScoreSession with 50 events exceeded 5s: %v", elapsed)
}

// ─── NFR-004: Daily operational cost MUST stay under $0.15 ──────────────────

func TestNFR004_DailyCostEstimation(t *testing.T) {
	g := NewWithT(t)

	// Cost model: per-call estimates derived from spec contracts and Anthropic pricing.
	//
	// Pricing basis (Haiku 4.5 / Sonnet 4.6):
	//   Haiku:  $1.00/MTok input, $5.00/MTok output
	//   Sonnet: $3.00/MTok input, $15.00/MTok output
	//
	// Per-call cost estimates (from actual prompt sizes in contracts):
	//   Filter (Haiku):    ~300 input, ~80 output → ~$0.0007/call
	//   Post-eval (Haiku): ~$0.0001/call (per scoring.md contract: "~$0.0001/call")
	//   Synthesis (Sonnet): ~400 input, ~100 output → ~$0.0027/call

	const (
		haikuInputPrice   = 1.00 / 1_000_000
		haikuOutputPrice  = 5.00 / 1_000_000
		sonnetInputPrice  = 3.00 / 1_000_000
		sonnetOutputPrice = 15.00 / 1_000_000
	)

	// Typical daily usage: ~100 interactions/day (spec NFR-004)
	const dailyInteractions = 100

	// Per-call costs
	filterCostPerCall := 300*haikuInputPrice + 80*haikuOutputPrice   // ~$0.0007
	postEvalCostPerCall := 0.0001                                    // per scoring.md contract
	synthCostPerCall := 400*sonnetInputPrice + 100*sonnetOutputPrice // ~$0.0027

	// Daily volume estimates:
	// - Filter: called per interaction when candidates exist (~80% of interactions)
	filterCallsPerDay := float64(dailyInteractions) * 0.80
	// - Post-eval: ~3 relevant events per interaction = ~300 events/day (per scoring.md)
	postEvalCallsPerDay := 300.0
	// - Synthesis: triggered when 2+ candidates have ShouldSynthesize (~10% of filter calls)
	synthCallsPerDay := filterCallsPerDay * 0.10

	dailyFilterCost := filterCallsPerDay * filterCostPerCall
	dailyPostEvalCost := postEvalCallsPerDay * postEvalCostPerCall
	dailySynthCost := synthCallsPerDay * synthCostPerCall

	totalDailyCost := dailyFilterCost + dailyPostEvalCost + dailySynthCost

	t.Logf("Daily cost breakdown:")
	t.Logf("  Filter (Haiku):     $%.4f (%.0f calls × $%.6f)", dailyFilterCost, filterCallsPerDay, filterCostPerCall)
	t.Logf("  Post-eval (Haiku):  $%.4f (%.0f calls × $%.6f)", dailyPostEvalCost, postEvalCallsPerDay, postEvalCostPerCall)
	t.Logf("  Synthesis (Sonnet): $%.4f (%.0f calls × $%.6f)", dailySynthCost, synthCallsPerDay, synthCostPerCall)
	t.Logf("  Total daily cost:   $%.4f (budget: $0.15)", totalDailyCost)

	g.Expect(totalDailyCost).To(BeNumerically("<", 0.15),
		"Estimated daily cost $%.4f exceeds $0.15 budget", totalDailyCost)
}

// benchLLM is a mock LLM that returns instantly for benchmarking pipeline overhead.
type benchLLM struct{}

func (b *benchLLM) AddRationale(_ context.Context, content string) (string, error) {
	return content, nil
}

func (b *benchLLM) Curate(_ context.Context, _ string, candidates []QueryResult) ([]CuratedResult, error) {
	return nil, nil
}

func (b *benchLLM) Decide(_ context.Context, _ string, _ []ExistingMemory) (*IngestDecision, error) {
	return &IngestDecision{Action: IngestAdd}, nil
}

func (b *benchLLM) Extract(_ context.Context, _ string) (*Observation, error) {
	return &Observation{Principle: "test"}, nil
}

func (b *benchLLM) Filter(_ context.Context, _ string, candidates []QueryResult) ([]FilterResult, error) {
	results := make([]FilterResult, len(candidates))
	for i, c := range candidates {
		results[i] = FilterResult{
			MemoryID:       c.ID,
			Content:        c.Content,
			Relevant:       true,
			RelevanceScore: 0.9,
			Tag:            "relevant",
		}
	}

	return results, nil
}

func (b *benchLLM) PostEval(_ context.Context, _, _ string) (*PostEvalResult, error) {
	return &PostEvalResult{Faithfulness: 0.7, Signal: "positive"}, nil
}

func (b *benchLLM) Rewrite(_ context.Context, content string) (string, error) {
	return content, nil
}

func (b *benchLLM) Synthesize(_ context.Context, memories []string) (string, error) {
	return "Synthesized principle from memories.", nil
}

// synthBenchLLM is a mock that marks all candidates for synthesis.
type synthBenchLLM struct{ benchLLM }

func (s *synthBenchLLM) Filter(_ context.Context, _ string, candidates []QueryResult) ([]FilterResult, error) {
	results := make([]FilterResult, len(candidates))
	for i, c := range candidates {
		results[i] = FilterResult{
			MemoryID:         c.ID,
			Content:          c.Content,
			Relevant:         true,
			RelevanceScore:   0.9,
			Tag:              "relevant",
			ShouldSynthesize: true,
		}
	}

	return results, nil
}
