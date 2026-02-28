package surface_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/store"
	"engram/internal/surface"
)

func TestT49_FullFormatWithNumberedList(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given two ScoredMemory items: high-impact surfaced + medium-impact new
	memories := []store.ScoredMemory{
		{
			Memory: store.Memory{
				Title:          "Use targ build system",
				Content:        "Build commands: targ test, targ lint",
				Confidence:     "A",
				ImpactScore:    0.9,
				SurfacingCount: 5,
			},
			Score: 0.8,
		},
		{
			Memory: store.Memory{
				Title:          "DI pattern in internal/",
				Content:        "All I/O through injected interfaces",
				Confidence:     "B",
				ImpactScore:    0.5,
				SurfacingCount: 0,
			},
			Score: 0.3,
		},
	}

	// When FormatSurfacing called with "session-start"
	result := surface.FormatSurfacing(memories, "session-start")
	// Then result contains system-reminder tags, numbered list, titles with confidence/impact
	g.Expect(result).To(ContainSubstring(`<system-reminder source="engram">`))
	g.Expect(result).To(ContainSubstring("[engram] 2 memories for this context:"))
	g.Expect(result).To(ContainSubstring("1. Use targ build system (A, high)"))
	g.Expect(result).To(ContainSubstring("Build commands:"))
	g.Expect(result).To(ContainSubstring("2. DI pattern in internal/ (B, new)"))
	g.Expect(result).To(ContainSubstring("</system-reminder>"))
}

func TestT50_CompactFormatForPreToolUse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given one ScoredMemory with high impact
	memories := []store.ScoredMemory{
		{
			Memory: store.Memory{
				Title:          "Use targ test not go test",
				Confidence:     "A",
				ImpactScore:    0.9,
				SurfacingCount: 3,
			},
			Score: 0.9,
		},
	}

	// When FormatSurfacing called with "pre-tool-use"
	result := surface.FormatSurfacing(memories, "pre-tool-use")
	// Then result uses compact single-line format, no numbered list, no "memories for this context"
	g.Expect(result).To(ContainSubstring(`<system-reminder source="engram">`))
	g.Expect(result).To(ContainSubstring("[engram] Use targ test not go test (A, high)"))
	g.Expect(result).NotTo(ContainSubstring("1."))
	g.Expect(result).NotTo(ContainSubstring("memories for this context"))
	g.Expect(result).To(ContainSubstring("</system-reminder>"))
}

func TestT51_EmptyMemoriesReturnsEmptyString(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given empty memories (nil)
	// When FormatSurfacing called with any hookType
	result := surface.FormatSurfacing(nil, "session-start")
	// Then result equals ""
	g.Expect(result).To(Equal(""))

	// Given empty memories (empty slice)
	// When FormatSurfacing called with any hookType
	result = surface.FormatSurfacing([]store.ScoredMemory{}, "user-prompt")
	// Then result equals ""
	g.Expect(result).To(Equal(""))
}

func TestT52_SingularWordingForOneMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given one ScoredMemory
	memories := []store.ScoredMemory{
		{
			Memory: store.Memory{
				Title:          "Single memory",
				Content:        "Some content",
				Confidence:     "B",
				ImpactScore:    0.6,
				SurfacingCount: 2,
			},
			Score: 0.5,
		},
	}

	// When FormatSurfacing called with "user-prompt"
	result := surface.FormatSurfacing(memories, "user-prompt")
	// Then result uses singular "1 memory", not "1 memories"
	g.Expect(result).To(ContainSubstring("[engram] 1 memory for this context:"))
	g.Expect(result).NotTo(ContainSubstring("memories"))
}
