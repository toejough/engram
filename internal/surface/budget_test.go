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
		PreToolUse:       200,
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

// T-192: matchToolMemories respects budget cap
func TestT192_ToolBudgetEnforcement(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 3 matching anti-pattern memories + 7 non-matching (BM25 needs <50% for positive IDF).
	// Each matching: title "commit advisory" (15) + anti_pattern (40) + keyword "commit" (6) ≈ 63 chars → 15 tokens.
	// Budget 35 → fits 2 (30 tokens), not 3 (45 tokens).
	memories := []*memory.Stored{
		{
			Title:       "commit advisory",
			AntiPattern: strings.Repeat("a", 40),
			FilePath:    "mem-a.toml",
			Keywords:    []string{"commit"},
		},
		{
			Title:       "commit advisory",
			AntiPattern: strings.Repeat("b", 40),
			FilePath:    "mem-b.toml",
			Keywords:    []string{"commit"},
		},
		{
			Title:       "commit advisory",
			AntiPattern: strings.Repeat("c", 40),
			FilePath:    "mem-c.toml",
			Keywords:    []string{"commit"},
		},
		{
			Title:       "unrelated thing",
			AntiPattern: "do not use alpha",
			FilePath:    "mem-d.toml",
			Keywords:    []string{"alpha"},
		},
		{
			Title:       "another thing",
			AntiPattern: "do not use beta",
			FilePath:    "mem-e.toml",
			Keywords:    []string{"beta"},
		},
		{
			Title:       "other thing",
			AntiPattern: "do not use gamma",
			FilePath:    "mem-f.toml",
			Keywords:    []string{"gamma"},
		},
		{
			Title:       "more thing",
			AntiPattern: "do not use delta",
			FilePath:    "mem-g.toml",
			Keywords:    []string{"delta"},
		},
		{
			Title:       "extra thing",
			AntiPattern: "do not use epsilon",
			FilePath:    "mem-h.toml",
			Keywords:    []string{"epsilon"},
		},
		{
			Title:       "final thing",
			AntiPattern: "do not use zeta",
			FilePath:    "mem-i.toml",
			Keywords:    []string{"zeta"},
		},
		{
			Title:       "last thing",
			AntiPattern: "do not use eta",
			FilePath:    "mem-j.toml",
			Keywords:    []string{"eta"},
		},
	}

	budgetCfg := surface.BudgetConfig{
		PreToolUse:       35,
		UserPromptSubmit: 300,
		SessionStart:     800,
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithBudgetConfig(budgetCfg))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: "git commit -m something",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	count := strings.Count(output, "  - mem-")
	g.Expect(count).To(Equal(2), "expected 2 memories within 35 token budget, got %d", count)
}

// T-193: Custom config values override defaults
func TestT193_BudgetConfigCustomValues(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	custom := surface.BudgetConfig{
		SessionStart:     1000,
		UserPromptSubmit: 500,
		PreToolUse:       300,
		PostToolUse:      150,
		Stop:             600,
		PreCompact:       400,
	}

	g.Expect(custom.ForMode(surface.ModeSessionStart)).To(Equal(1000))
	g.Expect(custom.ForMode(surface.ModePrompt)).To(Equal(500))
	g.Expect(custom.ForMode(surface.ModeTool)).To(Equal(300))
	g.Expect(custom.ForMode(surface.ModePreCompact)).To(Equal(400))
}

// T-193: Budget cap configuration loads from config with defaults fallback
func TestT193_BudgetConfigDefaults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	cfg := surface.DefaultBudgetConfig()
	g.Expect(cfg.SessionStart).To(Equal(surface.DefaultSessionStartBudget))
	g.Expect(cfg.UserPromptSubmit).To(Equal(surface.DefaultUserPromptSubmitBudget))
	g.Expect(cfg.PreToolUse).To(Equal(surface.DefaultPreToolUseBudget))
	g.Expect(cfg.PostToolUse).To(Equal(surface.DefaultPostToolUseBudget))
	g.Expect(cfg.Stop).To(Equal(surface.DefaultStopBudget))
	g.Expect(cfg.PreCompact).To(Equal(surface.DefaultPreCompactBudget))
}
