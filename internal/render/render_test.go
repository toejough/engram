package render_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/render"
)

// T-13: Memory produces DES-1 format with tier.
func TestT13_MemoryProducesDES1FormatWithTier(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	renderer := render.New()
	classified := &memory.ClassifiedMemory{
		Tier:            "A",
		Title:           "Use targ not go test",
		ObservationType: "reminder",
	}
	filePath := "memories/use-targ-not-go-test.toml"

	result := renderer.Render(classified, filePath)

	expected := "<system-reminder source=\"engram\">\n" +
		"[engram] Memory captured (tier A).\n" +
		"  Created: \"Use targ not go test\"\n" +
		"  Type: reminder\n" +
		"  File: memories/use-targ-not-go-test.toml\n" +
		"</system-reminder>\n"

	g.Expect(result).To(Equal(expected))
}

// TestTierBOutput verifies tier B format.
func TestTierBOutput(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	renderer := render.New()
	classified := &memory.ClassifiedMemory{
		Tier:            "B",
		Title:           "Check tests before commit",
		ObservationType: "correction",
	}

	result := renderer.Render(classified, "memories/check-tests.toml")

	g.Expect(result).To(ContainSubstring("(tier B)"))
	g.Expect(result).To(ContainSubstring("Check tests before commit"))
}
