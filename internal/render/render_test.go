package render_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/render"
)

func TestRenderStored_FormatsSBIAFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	renderer := render.New()
	mem := &memory.Stored{
		Situation: "when running tests",
		Behavior:  "use go test directly",
		Impact:    "misses coverage",
		Action:    "use targ test instead",
	}

	result := renderer.RenderStored(mem, "memories/use-targ-test.toml")

	g.Expect(result).To(ContainSubstring("<system-reminder"))
	g.Expect(result).To(ContainSubstring("Memory captured"))
	g.Expect(result).To(ContainSubstring("Situation: when running tests"))
	g.Expect(result).To(ContainSubstring("Action: use targ test instead"))
	g.Expect(result).To(ContainSubstring("File: memories/use-targ-test.toml"))
}
