package surface_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// T-190: Token estimation formula computes len(text) / 4
func TestT190_EstimateTokens(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	g.Expect(surface.EstimateTokens(strings.Repeat("a", 100))).To(Equal(25))
	g.Expect(surface.EstimateTokens(strings.Repeat("a", 99))).To(Equal(24))
	g.Expect(surface.EstimateTokens("")).To(Equal(0))
	g.Expect(surface.EstimateTokens("abc")).To(Equal(0))
	g.Expect(surface.EstimateTokens("abcd")).To(Equal(1))
}

// T-191: matchPromptMemories respects budget cap
func TestT191_PromptBudgetEnforcement(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 3 matching memories + 7 non-matching (BM25 needs <50% term frequency for positive IDF).
	// Each matching memory: title "budget keyword" (14) + content (50) + keyword (7) ≈ 73 chars → 18 tokens.
	// Budget 40 → fits 2 (36 tokens), not 3 (54 tokens).
	memories := []*memory.Stored{
		{
			Title:    "budget keyword",
			Content:  strings.Repeat("a", 50),
			FilePath: "mem-a.toml",
			Keywords: []string{"keyword"},
		},
		{
			Title:    "budget keyword",
			Content:  strings.Repeat("b", 50),
			FilePath: "mem-b.toml",
			Keywords: []string{"keyword"},
		},
		{
			Title:    "budget keyword",
			Content:  strings.Repeat("c", 50),
			FilePath: "mem-c.toml",
			Keywords: []string{"keyword"},
		},
		{
			Title:    "unrelated alpha",
			Content:  "nothing here",
			FilePath: "mem-d.toml",
			Keywords: []string{"alpha"},
		},
		{
			Title:    "unrelated beta",
			Content:  "nothing here",
			FilePath: "mem-e.toml",
			Keywords: []string{"beta"},
		},
		{
			Title:    "unrelated gamma",
			Content:  "other stuff",
			FilePath: "mem-f.toml",
			Keywords: []string{"gamma"},
		},
		{
			Title:    "unrelated delta",
			Content:  "more stuff",
			FilePath: "mem-g.toml",
			Keywords: []string{"delta"},
		},
		{
			Title:    "unrelated epsilon",
			Content:  "yet more",
			FilePath: "mem-h.toml",
			Keywords: []string{"epsilon"},
		},
		{
			Title:    "unrelated zeta",
			Content:  "and more",
			FilePath: "mem-i.toml",
			Keywords: []string{"zeta"},
		},
		{
			Title:    "unrelated eta",
			Content:  "final one",
			FilePath: "mem-j.toml",
			Keywords: []string{"eta"},
		},
	}

	budgetCfg := surface.BudgetConfig{
		UserPromptSubmit: 40,
		SessionStart:     800,
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithBudgetConfig(budgetCfg))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "keyword",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Count how many memories surfaced. Each line with "  - mem-" is a surfaced memory.
	output := buf.String()
	count := strings.Count(output, "  - mem-")
	g.Expect(count).To(Equal(2), "expected 2 memories within 40 token budget, got %d", count)
}

// T-193: Custom config values override defaults
func TestT193_BudgetConfigCustomValues(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	custom := surface.BudgetConfig{
		SessionStart:     1000,
		UserPromptSubmit: 500,
		Stop:             600,
	}

	g.Expect(custom.ForMode(surface.ModePrompt)).To(Equal(500))
}

// T-193: Budget cap configuration loads from config with defaults fallback (REQ-P4e-2/3/4: updated targets).
func TestT193_BudgetConfigDefaults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	cfg := surface.DefaultBudgetConfig()
	g.Expect(cfg.SessionStart).To(Equal(surface.DefaultSessionStartBudget))         // 600
	g.Expect(cfg.UserPromptSubmit).To(Equal(surface.DefaultUserPromptSubmitBudget)) // 250
	g.Expect(cfg.Stop).To(Equal(surface.DefaultStopBudget))
}
