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

// TestBudgetConfigCustomValues verifies custom config values.
func TestBudgetConfigCustomValues(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	custom := surface.BudgetConfig{
		UserPromptSubmit: 500,
	}

	g.Expect(custom.ForMode(surface.ModePrompt)).To(Equal(500))
}

// TestBudgetConfigDefaults verifies default budget values.
func TestBudgetConfigDefaults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	cfg := surface.DefaultBudgetConfig()
	g.Expect(cfg.UserPromptSubmit).To(Equal(surface.DefaultUserPromptSubmitBudget))
}

// TestEstimateTokens verifies token estimation formula.
func TestEstimateTokens(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	g.Expect(surface.EstimateTokens(strings.Repeat("a", 100))).To(Equal(25))
	g.Expect(surface.EstimateTokens(strings.Repeat("a", 99))).To(Equal(24))
	g.Expect(surface.EstimateTokens("")).To(Equal(0))
	g.Expect(surface.EstimateTokens("abc")).To(Equal(0))
	g.Expect(surface.EstimateTokens("abcd")).To(Equal(1))
}

// TestForMode_UnknownModeReturnsZero verifies that ForMode returns 0 for unknown modes.
func TestForMode_UnknownModeReturnsZero(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	cfg := surface.DefaultBudgetConfig()
	g.Expect(cfg.ForMode("unknown-mode")).To(Equal(0))
}

// TestPromptBudgetEnforcement verifies budget cap on prompt mode.
func TestPromptBudgetEnforcement(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// 3 matching memories + 7 non-matching for BM25 IDF contrast.
	memories := []*memory.Stored{
		{
			Situation: "budget keyword situation",
			Behavior:  strings.Repeat("a", 50),
			Action:    "use budget keyword action",
			FilePath:  "mem-a.toml",
		},
		{
			Situation: "budget keyword situation",
			Behavior:  strings.Repeat("b", 50),
			Action:    "use budget keyword action",
			FilePath:  "mem-b.toml",
		},
		{
			Situation: "budget keyword situation",
			Behavior:  strings.Repeat("c", 50),
			Action:    "use budget keyword action",
			FilePath:  "mem-c.toml",
		},
		{Situation: "unrelated alpha", FilePath: "mem-d.toml"},
		{Situation: "unrelated beta", FilePath: "mem-e.toml"},
		{Situation: "unrelated gamma", FilePath: "mem-f.toml"},
		{Situation: "unrelated delta", FilePath: "mem-g.toml"},
		{Situation: "unrelated epsilon", FilePath: "mem-h.toml"},
		{Situation: "unrelated zeta", FilePath: "mem-i.toml"},
		{Situation: "unrelated eta", FilePath: "mem-j.toml"},
	}

	budgetCfg := surface.BudgetConfig{
		UserPromptSubmit: 40,
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(retriever, surface.WithBudgetConfig(budgetCfg))

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
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
	g.Expect(count).To(BeNumerically("<=", 2), "expected at most 2 memories within 40 token budget")
}
