package surface_test

// Tests for stricter BM25 floor on unproven memories (#307).
//
// Strategy: tune document frequency so the shared query term has an IDF
// that produces a BM25 score between the proven floor (0.05) and the
// unproven floor (0.20 prompt, 0.30 tool).
//
// With N=21 docs and df=10 for "deploy":
//   IDF = ln((21-10+0.5)/(10+0.5)) = ln(1.095) ≈ 0.091
// A single-term match with TF=1 and dl≈avgdl yields score ≈ 0.091.
//   0.091 > 0.05 (proven passes)
//   0.091 < 0.20 (unproven prompt fails)
//   0.091 < 0.30 (unproven tool fails)

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

// TestProvenMemoryPassesAtLowerBM25Score verifies that a proven memory (SurfacedCount >= 1)
// with a weak BM25 score (between 0.05 and 0.20) still surfaces in prompt mode.
func TestProvenMemoryPassesAtLowerBM25Score(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	provenPath := "proven-weak-match.toml"

	// 21 docs total: 1 proven target + 9 fillers with "deploy" + 11 fillers without.
	// "deploy" appears in 10/21 docs → IDF ≈ 0.091 → score ≈ 0.091 > 0.05.
	memories := make([]*memory.Stored, 0, 21)
	memories = append(memories, &memory.Stored{
		Title:     "Proven Rule",
		FilePath:  provenPath,
		Principle: "deploy safely always",
		Keywords:  []string{"deploy"},
	})

	for i := range 9 {
		memories = append(memories, &memory.Stored{
			Title:     fmt.Sprintf("Filler Deploy %d", i),
			FilePath:  fmt.Sprintf("filler-deploy-%d.toml", i),
			Principle: fmt.Sprintf("deploy filler %d", i),
			Keywords:  []string{"deploy"},
		})
	}

	for i := range 11 {
		memories = append(memories, &memory.Stored{
			Title:     fmt.Sprintf("Filler Other %d", i),
			FilePath:  fmt.Sprintf("filler-other-%d.toml", i),
			Principle: fmt.Sprintf("other filler %d", i),
			Keywords:  []string{"other"},
		})
	}

	stats := map[string]surface.EffectivenessStat{
		provenPath: {SurfacedCount: 3, EffectivenessScore: 60.0},
	}

	eff := &fakeEffectivenessComputer{stats: stats}
	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "deploy",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(
		ContainSubstring("proven-weak-match"),
		"proven memory with weak BM25 should still surface in prompt mode",
	)
}

// TestUnprovenPromptMemoryFilteredByHigherBM25Floor verifies that an unproven memory
// with a BM25 score below unprovenBM25FloorPrompt (0.20) is filtered, while a proven
// memory with the same BM25 score passes through.
func TestUnprovenPromptMemoryFilteredByHigherBM25Floor(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	unprovenPath := "unproven-deploy.toml"
	provenPath := "proven-deploy.toml"

	// 21 docs total: 2 targets + 8 fillers with "deploy" + 11 fillers without.
	// "deploy" appears in 10/21 docs → IDF ≈ 0.091.
	// Unproven gets 0.20 floor → 0.091 < 0.20 → filtered.
	// Proven gets 0.05 floor → 0.091 > 0.05 → passes.
	memories := make([]*memory.Stored, 0, 21)
	memories = append(memories,
		&memory.Stored{
			Title:     "Unproven Rule",
			FilePath:  unprovenPath,
			Principle: "deploy safely always",
			Keywords:  []string{"deploy"},
		},
		&memory.Stored{
			Title:     "Proven Rule",
			FilePath:  provenPath,
			Principle: "deploy safely always",
			Keywords:  []string{"deploy"},
		},
	)

	for i := range 8 {
		memories = append(memories, &memory.Stored{
			Title:     fmt.Sprintf("Filler Deploy %d", i),
			FilePath:  fmt.Sprintf("filler-deploy-%d.toml", i),
			Principle: fmt.Sprintf("deploy filler %d", i),
			Keywords:  []string{"deploy"},
		})
	}

	for i := range 11 {
		memories = append(memories, &memory.Stored{
			Title:     fmt.Sprintf("Filler Other %d", i),
			FilePath:  fmt.Sprintf("filler-other-%d.toml", i),
			Principle: fmt.Sprintf("other filler %d", i),
			Keywords:  []string{"other"},
		})
	}

	stats := map[string]surface.EffectivenessStat{
		provenPath: {SurfacedCount: 1, EffectivenessScore: 50.0},
		// unprovenPath absent → isUnproven returns true → floor = 0.20.
	}

	eff := &fakeEffectivenessComputer{stats: stats}
	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever, surface.WithEffectiveness(eff))

	var buf bytes.Buffer

	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/tmp/data",
		Message: "deploy",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).NotTo(
		ContainSubstring("unproven-deploy"),
		"unproven memory should be filtered by higher BM25 floor",
	)
	g.Expect(output).To(
		ContainSubstring("proven-deploy"),
		"proven memory should pass the lower BM25 floor",
	)
}
