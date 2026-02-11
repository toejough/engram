package memory_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// ISSUE-188 Task 7: LLM-curated hook injection
// ============================================================================

// --- TierCurated format tests ---

// TEST-7001: TierCurated with mock extractor returns annotated results with relevance
func TestTierCuratedWithExtractorReturnsAnnotatedResults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	curated := []memory.CuratedResult{
		{Content: "Always use TDD", Relevance: "Directly relevant to testing query", MemoryType: "correction"},
		{Content: "Prefer gomega matchers", Relevance: "Related testing tool", MemoryType: "pattern"},
	}
	jsonBytes, _ := json.Marshal(curated)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return jsonBytes, nil
		},
	}

	results := []memory.QueryResult{
		{Content: "Always use TDD", Score: 0.9, Confidence: 0.9},
		{Content: "Prefer gomega matchers", Score: 0.85, Confidence: 0.85},
		{Content: "Unrelated memory", Score: 0.3, Confidence: 0.5},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 10,
		MaxTokens:  2000,
		Tier:       memory.TierCurated,
		Query:      "how should I test?",
		Extractor:  extractor,
	})

	g.Expect(output).To(ContainSubstring("Always use TDD"))
	g.Expect(output).To(ContainSubstring("relevant:"))
	g.Expect(output).To(ContainSubstring("Directly relevant to testing query"))
}

// TEST-7002: TierCurated with failing extractor falls back to compact
func TestTierCuratedWithFailingExtractorFallsBackToCompact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("LLM unavailable")
		},
	}

	results := []memory.QueryResult{
		{Content: "Always use TDD", Score: 0.9, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 10,
		MaxTokens:  2000,
		Tier:       memory.TierCurated,
		Query:      "how should I test?",
		Extractor:  extractor,
	})

	// Should fall back to compact format (no "relevant:" annotation)
	g.Expect(output).To(ContainSubstring("Always use TDD"))
	g.Expect(output).ToNot(ContainSubstring("relevant:"))
}

// TEST-7003: TierCurated without extractor falls back to compact
func TestTierCuratedWithoutExtractorFallsBackToCompact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "Always use TDD", Score: 0.9, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 10,
		MaxTokens:  2000,
		Tier:       memory.TierCurated,
		Query:      "how should I test?",
		// No Extractor set
	})

	// Should fall back to compact format
	g.Expect(output).To(ContainSubstring("Always use TDD"))
	g.Expect(output).ToNot(ContainSubstring("relevant:"))
}

// --- Hook type detection tests ---

// TEST-7004: IsPreToolUse returns true for PreToolUse hook event
func TestHookInputIsPreToolUse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{HookEventName: "PreToolUse"}
	g.Expect(hi.IsPreToolUse()).To(BeTrue())
}

// TEST-7005: IsPreToolUse returns false for UserPromptSubmit hook event
func TestHookInputIsPreToolUseFalseForPromptSubmit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{HookEventName: "UserPromptSubmit"}
	g.Expect(hi.IsPreToolUse()).To(BeFalse())
}

// TEST-7006: SupportsCuration returns true for UserPromptSubmit
func TestHookInputSupportsCurationForUserPromptSubmit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{HookEventName: "UserPromptSubmit"}
	g.Expect(hi.SupportsCuration()).To(BeTrue())
}

// TEST-7007: SupportsCuration returns true for SessionStart
func TestHookInputSupportsCurationForSessionStart(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{HookEventName: "SessionStart"}
	g.Expect(hi.SupportsCuration()).To(BeTrue())
}

// TEST-7008: SupportsCuration returns false for PreToolUse
func TestHookInputSupportsCurationFalseForPreToolUse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{HookEventName: "PreToolUse"}
	g.Expect(hi.SupportsCuration()).To(BeFalse())
}

// TEST-7009: SupportsCuration returns false for nil HookInput
func TestHookInputSupportsCurationFalseForNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var hi *memory.HookInput
	g.Expect(hi.SupportsCuration()).To(BeFalse())
}

// --- ResolveTier tests ---

// TEST-7010: ResolveTier downgrades TierCurated to TierCompact for PreToolUse
func TestResolveTierDowngradesCuratedForPreToolUse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{HookEventName: "PreToolUse"}
	tier := memory.ResolveTier(memory.TierCurated, hi)
	g.Expect(tier).To(Equal(memory.TierCompact))
}

// TEST-7011: ResolveTier keeps TierCurated for UserPromptSubmit
func TestResolveTierKeepsCuratedForUserPromptSubmit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{HookEventName: "UserPromptSubmit"}
	tier := memory.ResolveTier(memory.TierCurated, hi)
	g.Expect(tier).To(Equal(memory.TierCurated))
}

// TEST-7012: ResolveTier keeps TierCurated for SessionStart
func TestResolveTierKeepsCuratedForSessionStart(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{HookEventName: "SessionStart"}
	tier := memory.ResolveTier(memory.TierCurated, hi)
	g.Expect(tier).To(Equal(memory.TierCurated))
}

// TEST-7013: ResolveTier keeps TierCompact regardless of hook type
func TestResolveTierKeepsCompactForAnyHook(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{HookEventName: "UserPromptSubmit"}
	tier := memory.ResolveTier(memory.TierCompact, hi)
	g.Expect(tier).To(Equal(memory.TierCompact))
}

// TEST-7014: ResolveTier keeps TierFull regardless of hook type
func TestResolveTierKeepsFullForAnyHook(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	hi := &memory.HookInput{HookEventName: "PreToolUse"}
	tier := memory.ResolveTier(memory.TierFull, hi)
	g.Expect(tier).To(Equal(memory.TierFull))
}

// TEST-7015: ResolveTier keeps TierCurated when hookInput is nil (CLI mode, not hook)
func TestResolveTierKeepsCuratedWhenNoHookInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tier := memory.ResolveTier(memory.TierCurated, nil)
	g.Expect(tier).To(Equal(memory.TierCurated))
}

// --- Property tests ---

// TEST-7016: Property: curated output entry count <= input candidate count
func TestPropertyCuratedOutputCountLEInputCount(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		numCandidates := rapid.IntRange(1, 20).Draw(rt, "numCandidates")
		numCurated := rapid.IntRange(0, numCandidates).Draw(rt, "numCurated")

		// Build candidates
		candidates := make([]memory.QueryResult, numCandidates)
		for i := range candidates {
			candidates[i] = memory.QueryResult{
				Content:    strings.Repeat("memory ", i+1),
				Score:      0.9 - float64(i)*0.03,
				Confidence: 0.9,
			}
		}

		// Build curated response with numCurated entries
		curated := make([]memory.CuratedResult, numCurated)
		for i := range curated {
			curated[i] = memory.CuratedResult{
				Content:    candidates[i].Content,
				Relevance:  "relevant",
				MemoryType: "pattern",
			}
		}
		jsonBytes, _ := json.Marshal(curated)

		extractor := &memory.ClaudeCLIExtractor{
			Model:   "haiku",
			Timeout: 30,
			CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return jsonBytes, nil
			},
		}

		output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
			Results:    candidates,
			MaxEntries: 100,
			MaxTokens:  100000,
			Tier:       memory.TierCurated,
			Query:      "test query",
			Extractor:  extractor,
		})

		// Count bullet entries in output
		bulletCount := strings.Count(output, "\n- ")

		g.Expect(bulletCount).To(BeNumerically("<=", numCandidates),
			"Curated output entries must not exceed input candidates")
	})
}

// TEST-7017: Property: ResolveTier never returns TierCurated for PreToolUse
func TestPropertyResolveTierNeverCuratedForPreToolUse(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		tier := rapid.SampledFrom([]memory.OutputTier{
			memory.TierCompact, memory.TierFull, memory.TierCurated,
		}).Draw(rt, "tier")

		hi := &memory.HookInput{HookEventName: "PreToolUse"}
		resolved := memory.ResolveTier(tier, hi)
		g.Expect(resolved).ToNot(Equal(memory.TierCurated),
			"PreToolUse should never resolve to TierCurated")
	})
}
