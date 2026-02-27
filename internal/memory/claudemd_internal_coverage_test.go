//go:build sqlite_fts5

package memory

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

// TestCheckContextErr_NilContext verifies checkContextErr returns nil for a nil context.
func TestCheckContextErr_NilContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := checkContextErr(nil)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestFindE5Score_EmptyResults verifies findE5Score returns 0.0 for empty slice.
func TestFindE5Score_EmptyResults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	score := findE5Score(1, nil)

	g.Expect(score).To(Equal(0.0))
}

// TestFindE5Score_NoMatch verifies findE5Score returns 0.0 when memoryID is not in results.
func TestFindE5Score_NoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []QueryResult{
		{ID: 10, Score: 0.9},
		{ID: 20, Score: 0.7},
	}

	score := findE5Score(99, results)

	g.Expect(score).To(Equal(0.0))
}

// TestMergeEntries_WithFailingExtractor verifies mergeEntries falls back to heuristic on LLM error.
func TestMergeEntries_WithFailingExtractor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &mockSynthesizer{result: "", err: errors.New("api failed")}

	// entry1 is longer, so it should be returned by fallback heuristic
	result := mergeEntries("longer entry one here", "short", ext)

	g.Expect(result).To(Equal("longer entry one here"))
}

// TestMergeEntries_WithSuccessfulExtractor verifies mergeEntries uses LLM result when available.
func TestMergeEntries_WithSuccessfulExtractor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := &mockSynthesizer{result: "merged principle", err: nil}

	result := mergeEntries("entry one", "entry two", ext)

	g.Expect(result).To(Equal("merged principle"))
}

// TestSplitLongEntry_MultipleSentences verifies splitLongEntry splits on ". " sentence boundary.
func TestSplitLongEntry_MultipleSentences(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := "First sentence. Second sentence. Third sentence"

	parts := splitLongEntry(entry)

	g.Expect(len(parts)).To(BeNumerically(">=", 2))
	g.Expect(parts[0]).To(ContainSubstring("First"))
	g.Expect(parts[1]).To(ContainSubstring("Second"))
}

// mockSynthesizer implements LLMExtractor with a configurable Synthesize result.
type mockSynthesizer struct {
	result string
	err    error
}

func (m *mockSynthesizer) AddRationale(_ context.Context, _ string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *mockSynthesizer) Curate(_ context.Context, _ string, _ []QueryResult) ([]CuratedResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSynthesizer) Decide(_ context.Context, _ string, _ []ExistingMemory) (*IngestDecision, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSynthesizer) Extract(_ context.Context, _ string) (*Observation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSynthesizer) ExtractBatch(_ context.Context, _ []string) ([]*Observation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSynthesizer) Filter(_ context.Context, _ string, _ []QueryResult) ([]FilterResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSynthesizer) IsNarrowLearning(_ context.Context, _ string, _ string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *mockSynthesizer) PostEval(_ context.Context, _, _ string) (*PostEvalResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSynthesizer) Rewrite(_ context.Context, _ string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *mockSynthesizer) Synthesize(_ context.Context, _ []string) (string, error) {
	return m.result, m.err
}
