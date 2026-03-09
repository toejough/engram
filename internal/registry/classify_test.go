package registry_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/registry"
)

// traces: T-281
func TestT281_RulesClassifiedAsAlwaysLoaded(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "rule",
		SurfacedCount: 0, // irrelevant for always-loaded
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 0, Ignored: 2,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	// 3/5 = 60% effectiveness → Working (binary: no Hidden Gem/Noise)
	g.Expect(quadrant).To(Equal(registry.Working))
}

// traces: T-282
func TestT282_SkillsClassifiedAsAlwaysLoaded(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "skill",
		SurfacedCount: 0, // irrelevant for always-loaded
		Evaluations: registry.EvaluationCounters{
			Followed: 1, Contradicted: 4, Ignored: 0,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	// 1/5 = 20% effectiveness → Leech (binary: low effectiveness, always-loaded)
	g.Expect(quadrant).To(Equal(registry.Leech))
}
