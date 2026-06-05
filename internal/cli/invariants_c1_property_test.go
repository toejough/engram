package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestInvariant_C1_ClusteringDeterminism locks invariant C1: AutoK
// (k-means + silhouette) is seeded by FNV-1a(query) via seedFromQuery, so
// clustering the SAME vault + SAME phrase twice yields byte-identical
// output — same K, same member partition, same representatives. We drive
// this through the public RunQuery pipeline (which invokes clusterSubgraph)
// twice over one generated vault and assert the two payloads are
// byte-for-byte identical. A non-deterministic seed (or unstable k-means)
// would surface as differing cluster IDs / members / representatives.
func TestInvariant_C1_ClusteringDeterminism(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		gExpect := NewWithT(rt)

		// At least minSubgraphForClustering (6) direct hits so the cluster
		// path actually runs rather than short-circuiting to "no clusters".
		const (
			minNotes = 6
			maxNotes = 24
		)

		noteCount := rapid.IntRange(minNotes, maxNotes).Draw(rt, "noteCount")

		vault := t.TempDir()
		memFS := newInMemoryFS()

		for i := range noteCount {
			relPath := "Permanent/" + propertyNodeName(i) + ".md"
			// Distinct per-note body content (beyond the shared "body" hit
			// token) so the vectors spread out and k-means has real structure
			// to partition — making any seed instability observable.
			body := fmt.Sprintf("---\ntype: fact\ntier: L3\n---\nbody cluster note %d variant %d\n",
				i, i*7%5)

			plantNoteWithSidecar(t, memFS, filepath.Clean(vault), relPath, body)
		}

		phrase := rapid.SampledFrom([]string{
			"body", "cluster note", "variant body", "note cluster body",
		}).Draw(rt, "phrase")

		first := runC1Query(t, gExpect, memFS, vault, phrase)
		second := runC1Query(t, gExpect, memFS, vault, phrase)

		// Guard against vacuity: C1 is about cluster STRUCTURE, so the run
		// must actually produce clusters — otherwise byte-identity is trivially
		// true and never compares K / members / representatives.
		var parsed queryParsed

		gExpect.Expect(yaml.Unmarshal(first, &parsed)).NotTo(HaveOccurred())
		gExpect.Expect(parsed.Budget.ClustersFound).To(BeNumerically(">", 0),
			"C1 fixture produced no clusters (phrase %q, notes %d) — test would be vacuous",
			phrase, noteCount)

		gExpect.Expect(second).To(Equal(first),
			"C1: same vault + phrase %q must cluster identically across runs", phrase)
	})
}

// runC1Query runs a single-phrase un-tiered query and returns the raw
// payload bytes so two runs can be compared byte-for-byte.
func runC1Query(t *testing.T, gExpect *WithT, memFS *inMemoryFS, vault, phrase string) []byte {
	t.Helper()

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{phrase}, VaultPath: vault, Limit: 20},
		newQueryDeps(memFS), &out)
	gExpect.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return nil
	}

	// Return a copy: bytes.Buffer reuses its backing array across calls.
	return bytes.Clone(out.Bytes())
}
