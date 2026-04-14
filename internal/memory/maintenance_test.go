package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestMemoryRecord_ToStored_PreservesFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rec := memory.MemoryRecord{
		Type:      "feedback",
		Source:    "observation",
		Situation: "when running tests",
		Content: memory.ContentFields{
			Behavior: "use go test directly",
			Impact:   "misses coverage",
			Action:   "use targ test",
		},
		CreatedAt: "2026-03-27T10:00:00Z",
		UpdatedAt: "2026-03-27T10:00:00Z",
	}

	stored := rec.ToStored("/path/to/test.toml")
	g.Expect(stored).NotTo(BeNil())

	if stored == nil {
		return
	}

	g.Expect(stored.Type).To(Equal("feedback"))
	g.Expect(stored.Source).To(Equal("observation"))
	g.Expect(stored.Situation).To(Equal("when running tests"))
	g.Expect(stored.Content.Behavior).To(Equal("use go test directly"))
	g.Expect(stored.Content.Impact).To(Equal("misses coverage"))
	g.Expect(stored.Content.Action).To(Equal("use targ test"))
	g.Expect(stored.FilePath).To(Equal("/path/to/test.toml"))
}
