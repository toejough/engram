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

// TestColdStartBudgetDoesNotLimitProvenMemories verifies that proven memories (SurfacedCount >= 1)
// are not limited by the cold-start budget in session-start mode.
func TestColdStartBudgetDoesNotLimitProvenMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 5 memories, all proven (SurfacedCount >= 1 in effectiveness stats).
	memories := make([]*memory.Stored, 0, 5)
	stats := make(map[string]surface.EffectivenessStat, 5)

	for i := range 5 {
		path := fmt.Sprintf("proven-%d.toml", i)
		memories = append(memories, &memory.Stored{
			Title:     fmt.Sprintf("Proven Memory %d", i),
			FilePath:  path,
			Principle: fmt.Sprintf("principle %d", i),
			Keywords:  []string{fmt.Sprintf("keyword%d", i)},
		})
		stats[path] = surface.EffectivenessStat{SurfacedCount: 1, EffectivenessScore: 60.0}
	}

	eff := &fakeEffectivenessComputer{stats: stats}
	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()

	surfaced := 0

	for i := range 5 {
		if strings.Contains(output, fmt.Sprintf("proven-%d", i)) {
			surfaced++
		}
	}

	g.Expect(surfaced).To(Equal(5), "cold-start budget should not limit proven memories")
}

// TestColdStartBudgetLimitsUnprovenPromptMemories verifies that when all candidates are unproven,
// only 1 surfaces in prompt mode (cold-start budget = 1), not promptLimit (10).
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

	g.Expect(surfaced).To(Equal(1), "cold-start budget should allow only 1 unproven memory in prompt mode")
}

// TestColdStartBudgetLimitsUnprovenSessionStart verifies that when all memories are unproven,
// only 1 surfaces in session-start mode (cold-start budget = 1), not sessionStartLimit (7).
func TestColdStartBudgetLimitsUnprovenSessionStart(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 5 memories, all unproven (absent from effectiveness map).
	memories := make([]*memory.Stored, 0, 5)

	for i := range 5 {
		memories = append(memories, &memory.Stored{
			Title:     fmt.Sprintf("Unproven Memory %d", i),
			FilePath:  fmt.Sprintf("unproven-%d.toml", i),
			Principle: fmt.Sprintf("principle %d", i),
			Keywords:  []string{fmt.Sprintf("keyword%d", i)},
		})
	}

	// No effectiveness data → all memories are unproven.
	eff := &fakeEffectivenessComputer{stats: map[string]surface.EffectivenessStat{}}
	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModeSessionStart,
		DataDir: "/tmp/data",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()

	surfaced := 0

	for i := range 5 {
		if strings.Contains(output, fmt.Sprintf("unproven-%d", i)) {
			surfaced++
		}
	}

	g.Expect(surfaced).To(Equal(1), "cold-start budget should allow only 1 unproven memory in session-start mode")
}

// TestColdStartBudgetLimitsUnprovenToolMemories verifies that when all candidates are unproven,
// only 1 surfaces in tool mode (cold-start budget = 1), not toolLimit (2).
func TestColdStartBudgetLimitsUnprovenToolMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 3 target memories: strong match to "git commit fix" — all unproven (absent from effectiveness).
	// 8 fillers with different keywords for IDF contrast.
	// "commit"/"git"/"fix" each appear in only 3/11 docs → high IDF → BM25 well above 0.30 floor.
	memories := make([]*memory.Stored, 0, 11)

	for i := range 3 {
		memories = append(memories, &memory.Stored{
			Title:       fmt.Sprintf("Commit Rule %d", i),
			FilePath:    fmt.Sprintf("commit-rule-%d.toml", i),
			AntiPattern: "manual git commit fix",
			Keywords:    []string{"commit", "git", "fix"},
			Principle:   "use commit skill for git fix",
		})
	}

	fillerKeywords := []string{"logging", "testing", "deploy", "config", "monitoring", "caching", "auth", "docs"}

	for _, keyword := range fillerKeywords {
		memories = append(memories, &memory.Stored{
			Title:       keyword + " rule",
			FilePath:    keyword + "-rule.toml",
			AntiPattern: keyword + " violation",
			Keywords:    []string{keyword},
			Principle:   keyword + " standards",
		})
	}

	// No effectiveness data → all 3 target memories are unproven.
	eff := &fakeEffectivenessComputer{stats: map[string]surface.EffectivenessStat{}}
	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m fix",
		Format:    surface.FormatJSON,
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()

	// Count how many of the 3 target memories surfaced.
	surfaced := 0

	for i := range 3 {
		if strings.Contains(output, fmt.Sprintf("commit-rule-%d", i)) {
			surfaced++
		}
	}

	g.Expect(surfaced).To(Equal(1), "cold-start budget should allow only 1 unproven memory in tool mode")
}
