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
		Situation:        "when running tests",
		Behavior:         "use go test directly",
		Impact:           "misses coverage",
		Action:           "use targ test",
		ProjectScoped:    true,
		ProjectSlug:      "engram",
		CreatedAt:        "2026-03-27T10:00:00Z",
		UpdatedAt:        "2026-03-27T10:00:00Z",
		SurfacedCount:    5,
		FollowedCount:    3,
		NotFollowedCount: 1,
		IrrelevantCount:  2,
	}

	stored := rec.ToStored("/path/to/test.toml")
	g.Expect(stored).NotTo(BeNil())

	if stored == nil {
		return
	}

	g.Expect(stored.Situation).To(Equal("when running tests"))
	g.Expect(stored.Behavior).To(Equal("use go test directly"))
	g.Expect(stored.Impact).To(Equal("misses coverage"))
	g.Expect(stored.Action).To(Equal("use targ test"))
	g.Expect(stored.ProjectScoped).To(BeTrue())
	g.Expect(stored.ProjectSlug).To(Equal("engram"))
	g.Expect(stored.SurfacedCount).To(Equal(5))
	g.Expect(stored.FollowedCount).To(Equal(3))
	g.Expect(stored.NotFollowedCount).To(Equal(1))
	g.Expect(stored.IrrelevantCount).To(Equal(2))
	g.Expect(stored.FilePath).To(Equal("/path/to/test.toml"))
}
