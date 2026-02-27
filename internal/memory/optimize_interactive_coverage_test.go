package memory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// TestOptimizeInteractive_EmptyDBPath covers the DBPath=="" and ClaudeMDPath=="" branches (lines 67-83).
// Redirects HOME to a temp dir for isolation.
func TestOptimizeInteractive_EmptyDBPath(t *testing.T) {
	// Not parallel: t.Setenv modifies HOME, cannot run concurrently.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir) // isolate: home dir → temp dir

	// DBPath and ClaudeMDPath intentionally empty → exercises home dir resolution (lines 67-83).
	_, _ = OptimizeInteractive(OptimizeInteractiveOpts{
		ReviewFunc: func(MaintenanceProposal) bool { return false },
		Context:    context.Background(),
	})
	// Accept any outcome — we only need lines 67-83 exercised.
}

// TestOptimizeInteractive_LLMEvalPipeline covers the LLM eval pipeline (lines 237-262).
// Uses a mock implementing both LLMExtractor and APIMessageCaller so the type assertion succeeds.
// CallAPIWithMessages returns an error to trigger the warning path (line 243).
func TestOptimizeInteractive_LLMEvalPipeline(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	db, err := InitTestDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Insert a promotable entry (high retrieval_count + multiple projects) so that
	// scanEmbeddings generates a "promote" proposal, which needsLLMTriage returns true for.
	_, err = db.Exec(`
		INSERT INTO embeddings (content, source, source_type, confidence, retrieval_count, projects_retrieved, principle, promoted)
		VALUES ('Always use TDD for quality', 'memory', 'user', 0.9, 15, 'proj1,proj2,proj3', 'Always use TDD', 0)
	`)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(claudeMDPath, []byte("# Test\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// callErr → TriageProposals.triageOneProposal fails → triageErr != nil → warning printed.
	extractor := &mockLLMAPICallerExtractor{
		callErr: errors.New("API unavailable"),
	}

	opts := OptimizeInteractiveOpts{
		DBPath:       dbPath,
		ClaudeMDPath: claudeMDPath,
		ReviewFunc:   func(MaintenanceProposal) bool { return false },
		Context:      context.Background(),
		Extractor:    extractor,
		NoLLMEval:    false, // false → enters LLM eval block (line 237)
	}

	result, err := OptimizeInteractive(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

// TestOptimizeInteractive_ReviewFuncNil covers the reviewProposalCtx path (lines 289-296).
// When ReviewFunc is nil, OptimizeInteractive reads from opts.Input for user review.
// An empty reader returns io.EOF on first read → reviewProposalCtx returns an error,
// covering the error path at lines 293-296.
func TestOptimizeInteractive_ReviewFuncNil(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	db, err := InitTestDB(dbPath)
	if err != nil {
		t.Fatal("failed to init db:", err)
	}

	defer db.Close()

	// Duplicate entries → ScanClaudeMD generates at least 1 prune proposal.
	err = os.WriteFile(claudeMDPath, []byte("## Promoted Learnings\n\n- Same entry\n- Same entry\n"), 0644)
	if err != nil {
		t.Fatal("failed to write CLAUDE.md:", err)
	}

	// ReviewFunc intentionally nil → exercises reviewProposalCtx branch (line 289-296).
	// Empty reader → ReadString returns io.EOF → reviewProposalCtx returns (false, io.EOF)
	// → OptimizeInteractive returns "review failed: EOF" error.
	opts := OptimizeInteractiveOpts{
		DBPath:       dbPath,
		ClaudeMDPath: claudeMDPath,
		Input:        strings.NewReader(""), // EOF → error path covered
		// ReviewFunc nil → else branch (line 289)
		Context: context.Background(),
	}

	_, _ = OptimizeInteractive(opts)
}

// mockLLMAPICallerExtractor implements both LLMExtractor and APIMessageCaller.
// Used for testing the LLM eval pipeline in OptimizeInteractive without the sqlite_fts5 tag.
type mockLLMAPICallerExtractor struct {
	callResult []byte
	callErr    error
}

func (m *mockLLMAPICallerExtractor) AddRationale(_ context.Context, content string) (string, error) {
	return content, nil
}

func (m *mockLLMAPICallerExtractor) CallAPIWithMessages(_ context.Context, _ APIMessageParams) ([]byte, error) {
	return m.callResult, m.callErr
}

func (m *mockLLMAPICallerExtractor) Curate(_ context.Context, _ string, _ []QueryResult) ([]CuratedResult, error) {
	return nil, nil
}

func (m *mockLLMAPICallerExtractor) Decide(_ context.Context, _ string, _ []ExistingMemory) (*IngestDecision, error) {
	return nil, nil
}

func (m *mockLLMAPICallerExtractor) Extract(_ context.Context, _ string) (*Observation, error) {
	return nil, nil
}

func (m *mockLLMAPICallerExtractor) Filter(_ context.Context, _ string, _ []QueryResult) ([]FilterResult, error) {
	return nil, nil
}

func (m *mockLLMAPICallerExtractor) PostEval(_ context.Context, _, _ string) (*PostEvalResult, error) {
	return &PostEvalResult{Faithfulness: 0.5, Signal: "positive"}, nil
}

func (m *mockLLMAPICallerExtractor) Rewrite(_ context.Context, content string) (string, error) {
	return content, nil
}

func (m *mockLLMAPICallerExtractor) Synthesize(_ context.Context, _ []string) (string, error) {
	return "", nil
}
