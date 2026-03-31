package surface_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/surface"
)

func TestDefaultSurfaceConfig_Values(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := surface.DefaultSurfaceConfig()

	g.Expect(cfg.CandidateCountMin).To(Equal(3))
	g.Expect(cfg.CandidateCountMax).To(Equal(8))
	g.Expect(cfg.BM25Threshold).To(BeNumerically("==", 0.3))
	g.Expect(cfg.ColdStartBudget).To(Equal(2))
	g.Expect(cfg.IrrelevanceHalfLife).To(Equal(5))
	g.Expect(cfg.InjectionPreamble).NotTo(BeEmpty())
	g.Expect(cfg.InjectionPreamble).To(ContainSubstring("engram"))
}

func TestWithSurfaceConfig_OverridesDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := surface.SurfaceConfig{
		CandidateCountMin:   1,
		CandidateCountMax:   5,
		BM25Threshold:       0.5,
		ColdStartBudget:     1,
		IrrelevanceHalfLife: 3,
		InjectionPreamble:   "custom preamble",
	}

	surfacer := surface.New(&fakeRetriever{}, surface.WithSurfaceConfig(cfg))
	g.Expect(surfacer).NotTo(BeNil())
}
