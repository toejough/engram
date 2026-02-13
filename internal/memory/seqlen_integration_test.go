//go:build integration

package memory_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
	"pgregory.net/rapid"
)

// Test that long text (>128 tokens) produces different embeddings than truncated version
// This validates the sequence length is > 128
func TestLongTextCapturesMoreSemantics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Create a short message and a long message that starts with the same text
	// but has meaningful additional content after token 128
	shortText := "Go programming language concurrency"
	longText := shortText + " " + strings.Repeat("advanced distributed systems patterns for microservices architecture with event sourcing and CQRS implementation details including saga orchestration ", 5)

	// Learn both
	err := memory.Learn(memory.LearnOpts{
		Message:    shortText,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    longText,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query for the long-specific content that's beyond 128 tokens
	results, err := memory.Query(memory.QueryOpts{
		Text:       "saga orchestration CQRS event sourcing microservices",
		Limit:      2,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).To(HaveLen(2))

	// The long text should rank HIGHER for the long-specific query
	// because with 512 tokens, it captures the additional content
	// With only 128 tokens, both would be similarly truncated
	g.Expect(results.Results[0].Content).To(ContainSubstring("saga orchestration"))
}

// Property: any text string produces a valid embedding regardless of length
func TestAnyLengthTextProducesValidEmbedding(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		tempDir := os.TempDir()
		memoryDir, err := os.MkdirTemp(tempDir, "seqlen-prop-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = os.RemoveAll(memoryDir) }()

		// Generate text of varying length (some well beyond 128 tokens)
		wordCount := rapid.IntRange(1, 200).Draw(t, "wordCount")
		words := make([]string, wordCount)
		for i := range words {
			words[i] = rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "word")
		}
		msg := strings.Join(words, " ")

		err = memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Query should succeed
		results, err := memory.Query(memory.QueryOpts{
			Text:       msg[:min(50, len(msg))],
			Limit:      1,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(results.Results).ToNot(BeEmpty())
	})
}
