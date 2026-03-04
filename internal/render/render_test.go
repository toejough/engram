package render_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/render"
)

// T-13: Normal memory produces DES-1 format.
func TestT13_NormalMemoryProducesDES1Format(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	renderer := render.New()
	mem := &memory.Enriched{
		Title:           "Use targ not go test",
		ObservationType: "reminder",
	}
	filePath := "memories/use-targ-not-go-test.toml"

	result := renderer.Render(mem, filePath)

	expected := "<system-reminder source=\"engram\">\n" +
		"[engram] Memory captured.\n" +
		"  Created: \"Use targ not go test\"\n" +
		"  Type: reminder\n" +
		"  File: memories/use-targ-not-go-test.toml\n" +
		"</system-reminder>\n"

	g.Expect(result).To(Equal(expected))
}
