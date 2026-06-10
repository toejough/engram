package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestProperty_SynthesizeL2_NearDuplicateL2CosineAtLeast095 locks the lazy-L2
// no-op band's precondition: for any vault whose only L2 is a near-duplicate of
// the cluster centroid, --synthesize-l2 reports nearest_l2.cosine >= 0.95.
//
// The clustered L1 notes all sit exactly at the query vector, so the cluster
// centroid is that vector. The single L2 perturbs one orthogonal axis by a
// small epsilon kept under the 0.95-cosine boundary (cos = 1/sqrt(1+e^2) >=
// 0.95 holds for e <= ~0.329; we draw e in [0, 0.3]). The binary applies no
// band, so the raw cosine is reported and must clear the threshold.
func TestProperty_SynthesizeL2_NearDuplicateL2CosineAtLeast095(t *testing.T) {
	t.Parallel()

	const (
		minL1Notes = 1
		maxL1Notes = 6
		maxEpsilon = 0.3
		noOpFloor  = float32(0.95)
	)

	queryVec := []float32{1, 0, 0, 0}

	rapid.Check(t, func(rt *rapid.T) {
		l1Count := rapid.IntRange(minL1Notes, maxL1Notes).Draw(rt, "l1Count")
		epsilon := float32(rapid.Float64Range(0, maxEpsilon).Draw(rt, "epsilon"))

		vault := t.TempDir()
		memFS := newInMemoryFS()

		for i := range l1Count {
			plantDualVector(t, memFS, vault, fmt.Sprintf("Permanent/%d.ep.md", i+1),
				"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)
		}

		// The sole L2: a near-duplicate of the centroid on one orthogonal axis.
		nearDup := []float32{1, epsilon, 0, 0}
		plantDualVector(t, memFS, vault, "Permanent/dup.fact.md",
			"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", nearDup, nearDup)

		deps := newQueryDeps(memFS)
		deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}

		var out bytes.Buffer

		err := cli.RunQuery(context.Background(),
			cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true}, deps, &out)
		if err != nil {
			rt.Fatalf("RunQuery: %v", err)
		}

		var parsed queryParsed
		if unmarshalErr := yaml.Unmarshal(out.Bytes(), &parsed); unmarshalErr != nil {
			rt.Fatalf("unmarshal: %v", unmarshalErr)
		}

		g := NewWithT(rt)
		g.Expect(parsed.Clusters).NotTo(BeEmpty())

		for _, cluster := range parsed.Clusters {
			g.Expect(cluster.NearestL2).NotTo(BeNil(),
				"a near-duplicate L2 must always surface as nearest_l2")
			g.Expect(cluster.NearestL2.Cosine).To(BeNumerically(">=", noOpFloor),
				"a near-duplicate L2 must report raw cosine >= 0.95 (the no-op band precondition)")
		}
	})
}
