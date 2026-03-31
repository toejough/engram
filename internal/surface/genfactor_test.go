package surface_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/surface"
)

func TestGenFactor_NotProjectScoped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Non-scoped memory always gets factor 1.0 regardless of project.
	g.Expect(surface.GenFactor(false, "proj-a", "proj-b")).To(Equal(1.0))
	g.Expect(surface.GenFactor(false, "", "proj-b")).To(Equal(1.0))
}

func TestGenFactor_ProjectScoped_CrossProject(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Cross-project project-scoped memory gets full penalty.
	g.Expect(surface.GenFactor(true, "proj-a", "proj-b")).To(Equal(0.0))
}

func TestGenFactor_ProjectScoped_EmptySlug(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Empty slug = no penalty (can't determine cross-project).
	g.Expect(surface.GenFactor(true, "", "proj-b")).To(Equal(1.0))
	g.Expect(surface.GenFactor(true, "proj-a", "")).To(Equal(1.0))
}

func TestGenFactor_ProjectScoped_SameProject(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Same project = no penalty.
	g.Expect(surface.GenFactor(true, "proj-a", "proj-a")).To(Equal(1.0))
}
