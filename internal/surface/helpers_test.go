package surface_test

// Whitebox tests for unexported helper functions via export_test.go aliases.

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/surface"
)

// TestEffectivenessScoreFor verifies the three-tier default logic.
func TestEffectivenessScoreFor(t *testing.T) {
	t.Parallel()

	t.Run("nil map returns unprovenDefaultEffectiveness", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		g.Expect(surface.ExportEffectivenessScoreFor("any.toml", nil)).
			To(Equal(surface.ExportUnprovenDefaultEffectiveness))
	})

	t.Run("absent path returns unprovenDefaultEffectiveness", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		g.Expect(surface.ExportEffectivenessScoreFor("missing.toml", map[string]surface.EffectivenessStat{})).
			To(Equal(surface.ExportUnprovenDefaultEffectiveness))
	})

	t.Run("zero surfacings returns unprovenDefaultEffectiveness", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		eff := map[string]surface.EffectivenessStat{
			"mem.toml": {SurfacedCount: 0, EffectivenessScore: 80},
		}
		g.Expect(surface.ExportEffectivenessScoreFor("mem.toml", eff)).
			To(Equal(surface.ExportUnprovenDefaultEffectiveness))
	})

	t.Run("insufficient surfacings returns sessionStartDefaultEffectiveness", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		eff := map[string]surface.EffectivenessStat{
			"mem.toml": {SurfacedCount: 3, EffectivenessScore: 80},
		}
		g.Expect(surface.ExportEffectivenessScoreFor("mem.toml", eff)).
			To(Equal(surface.ExportSessionStartDefaultEffectiveness))
	})

	t.Run("sufficient surfacings returns recorded score", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		eff := map[string]surface.EffectivenessStat{
			"mem.toml": {SurfacedCount: surface.ExportInsufficientDataThreshold, EffectivenessScore: 72.5},
		}
		g.Expect(surface.ExportEffectivenessScoreFor("mem.toml", eff)).To(Equal(72.5))
	})
}

// TestIsUnproven verifies cold-start detection logic.
func TestIsUnproven(t *testing.T) {
	t.Parallel()

	t.Run("nil effectiveness map returns true", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		g.Expect(surface.ExportIsUnproven("any.toml", nil)).To(BeTrue())
	})

	t.Run("path absent from map returns true", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		g.Expect(surface.ExportIsUnproven("missing.toml", map[string]surface.EffectivenessStat{})).To(BeTrue())
	})

	t.Run("SurfacedCount zero returns true", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		eff := map[string]surface.EffectivenessStat{
			"mem.toml": {SurfacedCount: 0, EffectivenessScore: 0},
		}
		g.Expect(surface.ExportIsUnproven("mem.toml", eff)).To(BeTrue())
	})

	t.Run("SurfacedCount one returns false", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		eff := map[string]surface.EffectivenessStat{
			"mem.toml": {SurfacedCount: 1, EffectivenessScore: 50},
		}
		g.Expect(surface.ExportIsUnproven("mem.toml", eff)).To(BeFalse())
	})
}
