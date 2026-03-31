package surface_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// TestApplyColdStartBudget_LimitsUnproven verifies that unproven memories are capped at budget.
func TestApplyColdStartBudget_LimitsUnproven(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "proven-a.toml", SurfacedCount: 3},
		{FilePath: "proven-b.toml", SurfacedCount: 1},
		{FilePath: "unproven-a.toml", SurfacedCount: 0},
		{FilePath: "unproven-b.toml", SurfacedCount: 0},
		{FilePath: "unproven-c.toml", SurfacedCount: 0},
	}

	result := surface.ApplyColdStartBudget(candidates, 2)

	g.Expect(result).To(HaveLen(4))

	paths := make([]string, 0, len(result))
	for _, m := range result {
		paths = append(paths, m.FilePath)
	}

	g.Expect(paths).To(ContainElements("proven-a.toml", "proven-b.toml", "unproven-a.toml", "unproven-b.toml"))
	g.Expect(paths).NotTo(ContainElement("unproven-c.toml"))
}

// TestApplyColdStartBudget_ZeroBudget_AllowsAll verifies that budget=0 passes all through.
func TestApplyColdStartBudget_ZeroBudget_AllowsAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "proven.toml", SurfacedCount: 5},
		{FilePath: "unproven-a.toml", SurfacedCount: 0},
		{FilePath: "unproven-b.toml", SurfacedCount: 0},
		{FilePath: "unproven-c.toml", SurfacedCount: 0},
	}

	result := surface.ApplyColdStartBudget(candidates, 0)

	g.Expect(result).To(HaveLen(4))
}
