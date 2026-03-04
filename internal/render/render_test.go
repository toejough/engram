package render_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/render"
)

// T-13: Normal (non-degraded) memory produces DES-1 format.
func TestT13_NormalMemoryProducesDES1Format(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	renderer := render.New()
	mem := &memory.Enriched{
		Title:           "Use targ not go test",
		ObservationType: "reminder",
	}
	filePath := "memories/use-targ-not-go-test.toml"

	result := renderer.Render(mem, filePath, false)

	expected := "<system-reminder source=\"engram\">\n" +
		"[engram] Memory captured.\n" +
		"  Created: \"Use targ not go test\"\n" +
		"  Type: reminder\n" +
		"  File: memories/use-targ-not-go-test.toml\n" +
		"</system-reminder>\n"

	g.Expect(result).To(Equal(expected))
}

// T-14: Degraded memory produces DES-2 format.
func TestT14_DegradedMemoryProducesDES2Format(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	renderer := render.New()
	mem := &memory.Enriched{
		Title: "remember to use targ",
	}
	filePath := "memories/remember-to-use-targ.toml"

	result := renderer.Render(mem, filePath, true)

	expected := "<system-reminder source=\"engram\">\n" +
		"[engram] Memory captured (degraded \u2014 no API key).\n" +
		"  Created: \"remember to use targ\"\n" +
		"  File: memories/remember-to-use-targ.toml\n" +
		"  Note: Set ANTHROPIC_API_KEY for enriched memories.\n" +
		"</system-reminder>\n"

	g.Expect(result).To(Equal(expected))
}
