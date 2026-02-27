package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestClaudeCLIExtractor_AddRationale_CommandError verifies AddRationale returns error when command fails.
func TestClaudeCLIExtractor_AddRationale_CommandError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("command not found")
		},
	}

	result, err := ext.AddRationale(context.Background(), "Always test code")

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

// TestClaudeCLIExtractor_AddRationale_Success verifies AddRationale trims and returns output.
func TestClaudeCLIExtractor_AddRationale_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("  Always test code - ensures correctness and prevents regressions  "), nil
		},
	}

	result, err := ext.AddRationale(context.Background(), "Always test code")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("Always test code - ensures correctness and prevents regressions"))
}

// TestClaudeCLIExtractor_CompileSkill_CommandError verifies CompileSkill returns error when command fails.
func TestClaudeCLIExtractor_CompileSkill_CommandError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("exec failed")
		},
	}

	result, err := ext.CompileSkill(context.Background(), "TDD", []string{"Always write tests first"})

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

// TestClaudeCLIExtractor_CompileSkill_Success verifies CompileSkill returns raw output.
func TestClaudeCLIExtractor_CompileSkill_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	want := `{"description":"Use when doing TDD","body":"## Overview\nTest content"}`

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(want), nil
		},
	}

	result, err := ext.CompileSkill(context.Background(), "TDD", []string{"mem1", "mem2"})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal(want))
}

// TestClaudeCLIExtractor_Decide_CommandError verifies Decide returns error when command fails.
func TestClaudeCLIExtractor_Decide_CommandError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("exec failed")
		},
	}

	decision, err := ext.Decide(context.Background(), "new content", nil)

	g.Expect(err).To(HaveOccurred())
	g.Expect(decision).To(BeNil())
}

// TestClaudeCLIExtractor_Decide_InvalidJSON verifies Decide returns error on invalid JSON.
func TestClaudeCLIExtractor_Decide_InvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("not json"), nil
		},
	}

	decision, err := ext.Decide(context.Background(), "new content", nil)

	g.Expect(err).To(HaveOccurred())
	g.Expect(decision).To(BeNil())
}

// TestClaudeCLIExtractor_Decide_Success verifies Decide parses valid JSON decision.
func TestClaudeCLIExtractor_Decide_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"action":"ADD","target_id":0,"reason":"genuinely new knowledge"}`), nil
		},
	}

	existing := []memory.ExistingMemory{
		{ID: 1, Content: "existing content", Similarity: 0.7},
	}

	decision, err := ext.Decide(context.Background(), "new content", existing)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(decision).ToNot(BeNil())

	if decision == nil {
		t.Fatal("decision is nil")
	}

	g.Expect(string(decision.Action)).To(Equal("ADD"))
	g.Expect(decision.Reason).To(ContainSubstring("new knowledge"))
}

// TestClaudeCLIExtractor_Filter_Empty verifies Filter returns empty slice for nil candidates.
func TestClaudeCLIExtractor_Filter_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{}

	results, err := ext.Filter(context.Background(), "test query", nil)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

// TestClaudeCLIExtractor_Filter_WithCandidates verifies Filter returns all as relevant.
func TestClaudeCLIExtractor_Filter_WithCandidates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{}
	candidates := []memory.QueryResult{
		{ID: 1, Content: "content one", Score: 0.9, MemoryType: "user"},
		{ID: 2, Content: "content two", Score: 0.7, MemoryType: "correction"},
	}

	results, err := ext.Filter(context.Background(), "test query", candidates)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).ToNot(BeNil())

	g.Expect(results).To(HaveLen(2))

	if len(results) == 0 {
		t.Fatal("results is empty")
	}

	g.Expect(results[0].Relevant).To(BeTrue())
	g.Expect(results[0].RelevanceScore).To(Equal(-1.0))
	g.Expect(results[0].MemoryType).To(Equal("user"))
}

// TestClaudeCLIExtractor_PostEval_CommandError verifies PostEval returns error when command fails.
func TestClaudeCLIExtractor_PostEval_CommandError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("exec failed")
		},
	}

	result, err := ext.PostEval(context.Background(), "memory content", "query text")

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestClaudeCLIExtractor_PostEval_InvalidJSON verifies PostEval returns error on invalid JSON.
func TestClaudeCLIExtractor_PostEval_InvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("not json"), nil
		},
	}

	result, err := ext.PostEval(context.Background(), "memory content", "query text")

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestClaudeCLIExtractor_PostEval_Success verifies PostEval parses valid JSON result.
func TestClaudeCLIExtractor_PostEval_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"faithfulness":0.85,"signal":"positive"}`), nil
		},
	}

	result, err := ext.PostEval(context.Background(), "memory content", "query text")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.Faithfulness).To(BeNumerically("~", 0.85, 0.001))
	g.Expect(result.Signal).To(Equal("positive"))
}

// TestClaudeCLIExtractor_Rewrite_CommandError verifies Rewrite returns error when command fails.
func TestClaudeCLIExtractor_Rewrite_CommandError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("exec failed")
		},
	}

	result, err := ext.Rewrite(context.Background(), "original content")

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeEmpty())
}

// TestClaudeCLIExtractor_Rewrite_Success verifies Rewrite trims and returns output.
func TestClaudeCLIExtractor_Rewrite_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 5 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("\n  Always verify inputs before processing  \n"), nil
		},
	}

	result, err := ext.Rewrite(context.Background(), "verify inputs")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("Always verify inputs before processing"))
}
