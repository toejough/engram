package surface_test

// Tests for cold-start budget enforcement (#307).
// After ranking, unproven memories (0 surfacings) are limited to at most 1 per invocation.

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// TestColdStartBudgetLimitsUnprovenPromptMemories verifies that when all candidates are unproven,
// only 2 surface in prompt mode (cold-start budget = 2), not promptLimit (10).
func TestColdStartBudgetLimitsUnprovenPromptMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 5 target memories: strong match to "refactor extract method" — all unproven.
	// 8 fillers with different keywords for IDF contrast.
	// "refactor"/"extract"/"method" each appear in only 5/13 docs → high IDF → BM25 above 0.20 floor.
	memories := make([]*memory.Stored, 0, 13)

	for i := range 5 {
		memories = append(memories, &memory.Stored{
			Title:     fmt.Sprintf("Refactor Rule %d", i),
			FilePath:  fmt.Sprintf("refactor-rule-%d.toml", i),
			Principle: "refactor extract method properly",
			Keywords:  []string{"refactor", "extract", "method"},
		})
	}

	fillerKeywords := []string{"logging", "testing", "deploy", "config", "monitoring", "caching", "auth", "docs"}

	for _, keyword := range fillerKeywords {
		memories = append(memories, &memory.Stored{
			Title:     keyword + " rule",
			FilePath:  keyword + "-rule.toml",
			Principle: keyword + " standards",
			Keywords:  []string{keyword},
		})
	}

	// No effectiveness data → all 5 target memories are unproven.
	eff := &fakeEffectivenessComputer{stats: map[string]surface.EffectivenessStat{}}
	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "refactor extract method",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()

	surfaced := 0

	for i := range 5 {
		if strings.Contains(output, fmt.Sprintf("refactor-rule-%d", i)) {
			surfaced++
		}
	}

	g.Expect(surfaced).To(Equal(2), "cold-start budget should allow only 2 unproven memories in prompt mode")
}
