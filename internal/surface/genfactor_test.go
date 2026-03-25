package surface_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/surface"
)

func TestGenFactor_CrossProjectPenalty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(surface.GenFactor(5, "proj-a", "proj-b")).To(Equal(1.0))
	g.Expect(surface.GenFactor(4, "proj-a", "proj-b")).To(Equal(0.8))
	g.Expect(surface.GenFactor(3, "proj-a", "proj-b")).To(Equal(0.5))
	g.Expect(surface.GenFactor(2, "proj-a", "proj-b")).To(Equal(0.2))
	g.Expect(surface.GenFactor(1, "proj-a", "proj-b")).To(Equal(0.05))
}

func TestGenFactor_EmptySlug(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(surface.GenFactor(1, "", "proj-a")).To(Equal(1.0))
	g.Expect(surface.GenFactor(1, "proj-a", "")).To(Equal(1.0))
}

func TestGenFactor_SameProject(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(surface.GenFactor(1, "proj-a", "proj-a")).To(Equal(1.0))
	g.Expect(surface.GenFactor(5, "proj-a", "proj-a")).To(Equal(1.0))
}

func TestGenFactor_UnsetGeneralizability(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(surface.GenFactor(0, "proj-a", "proj-b")).To(Equal(0.5))
}
