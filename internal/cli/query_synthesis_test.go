package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// TestQuery_NonSynthesis_StillPerPhraseClusters is the regression guard: with
// Synthesis:false (the default), multi-phrase query still produces PER-PHRASE
// clusters, each tagged with its originating phrase. Synthesis mode must not
// change the non-synthesis pipeline.
func TestQuery_NonSynthesis_StillPerPhraseClusters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	for i := range 12 {
		plantNoteWithSidecar(t, memFS, vault,
			fmt.Sprintf("Permanent/%d.note.md", i+1),
			fmt.Sprintf("---\ntype: fact\n---\nbody %d\n", i))
	}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body", "fact"}, VaultPath: vault, Synthesis: false},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Clusters).NotTo(BeEmpty(), "non-synthesis multi-phrase must still cluster per-phrase")

	for _, cluster := range parsed.Clusters {
		g.Expect(cluster.Phrase).NotTo(BeEmpty(),
			"non-synthesis cluster id=%d must be tagged with its phrase", cluster.ID)
	}
}

// TestQuery_Synthesis_BelowFloorStillClusters proves the >=6 minimum-subgraph
// floor is dropped in synthesis mode: a union of 4 direct hits (below the old
// floor of minSubgraphForClustering=6) still yields >=1 cluster, and every
// union member is accounted for across the emitted clusters. The fixture is two
// tight vector pairs so AutoK has a clean k=2 split available — the point is
// that clustering RUNS at all below the floor, which the non-synthesis path
// would short-circuit to "no clusters".
func TestQuery_Synthesis_BelowFloorStillClusters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Two tight groups of two notes each; well-separated across groups.
	groupA := []float32{1, 0, 0, 0}
	groupB := []float32{0, 1, 0, 0}

	plantWithFixedVector(t, memFS, vault, "Permanent/a1.md",
		"---\ntype: fact\n---\nunion seed a1\n", groupA)
	plantWithFixedVector(t, memFS, vault, "Permanent/a2.md",
		"---\ntype: fact\n---\nunion seed a2\n", groupA)
	plantWithFixedVector(t, memFS, vault, "Permanent/b1.md",
		"---\ntype: fact\n---\nunion seed b1\n", groupB)
	plantWithFixedVector(t, memFS, vault, "Permanent/b2.md",
		"---\ntype: fact\n---\nunion seed b2\n", groupB)

	deps := newQueryDeps(memFS)
	// Any query vector works: rankCandidates scores every compatible note,
	// so all four become direct hits and seed the union.
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: groupA}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"seed a", "seed b"}, VaultPath: vault, Limit: 20, Synthesis: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	// Floor dropped: clustering ran on a 4-member union (the non-synthesis
	// path returns no clusters below 6).
	g.Expect(parsed.Clusters).NotTo(BeEmpty(), "synthesis must cluster below the old >=6 floor")
	g.Expect(parsed.Budget.ClustersFound).To(BeNumerically(">=", 1))

	totalMembers := 0
	for _, cluster := range parsed.Clusters {
		totalMembers += len(cluster.Members)
	}

	g.Expect(totalMembers).To(Equal(4), "every union member must appear in exactly one cluster")
}

// TestQuery_Synthesis_EmptyVaultYieldsNoClusters proves the empty-union guard:
// a synthesis query over an empty vault produces no direct hits, so the union
// subgraph is empty and clustering is skipped (the >=1-cluster invariant only
// holds for a NON-empty match set). Items and clusters are both empty; exit 0.
func TestQuery_Synthesis_EmptyVaultYieldsNoClusters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"anything"}, VaultPath: vault, Limit: 20, Synthesis: true},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	g.Expect(parsed.Items).To(BeEmpty(), "empty vault has no union direct hits")
	g.Expect(parsed.Clusters).To(BeEmpty(), "empty union must not cluster")
	g.Expect(parsed.Budget.ClustersFound).To(BeZero())
}

// TestQuery_Synthesis_ItemsAreUnionDirectHits proves items[] in synthesis mode
// is the deduped union of direct hits (with content), independent of phrase
// ordering or repetition.
func TestQuery_Synthesis_ItemsAreUnionDirectHits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantWithFixedVector(t, memFS, vault, "Permanent/x.md",
		"---\ntype: fact\n---\nunion content x\n", []float32{1, 0, 0, 0})
	plantWithFixedVector(t, memFS, vault, "Permanent/y.md",
		"---\ntype: fact\n---\nunion content y\n", []float32{0, 1, 0, 0})

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer

	// Two phrases, both surfacing both notes — the union must dedup by path.
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:   []string{"union content", "union content"},
			VaultPath: vault,
			Limit:     20,
			Synthesis: true,
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	seen := map[string]int{}
	for _, item := range parsed.Items {
		seen[item.Path]++
		g.Expect(item.Content).NotTo(BeEmpty(), "union direct hit %s must carry content", item.Path)
		g.Expect(item.Provenances).To(ContainElement("direct"))
	}

	g.Expect(seen).To(HaveLen(2), "items must be the two deduped union direct hits")
	g.Expect(seen["Permanent/x.md"]).To(Equal(1))
	g.Expect(seen["Permanent/y.md"]).To(Equal(1))
}

// TestQuery_Synthesis_NoGoodSplitReturnsOneCluster proves the K=0 invariant:
// when AutoK finds no split that beats the silhouette floor (here, identical
// vectors → silhouette 0), synthesis returns a SINGLE cluster containing ALL
// union members with one representative — never "no clusters". An L3 note in
// the vault also exercises the nearest_l3 annotation on the single cluster.
func TestQuery_Synthesis_NoGoodSplitReturnsOneCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	const memberCount = 4

	identical := []float32{1, 0, 0, 0}

	for i := range memberCount {
		plantWithFixedVector(t, memFS, vault,
			fmt.Sprintf("Permanent/u%d.md", i),
			fmt.Sprintf("---\ntype: fact\n---\nidentical union seed %d\n", i),
			identical)
	}

	// A standalone L3 ADR so nearest_l3 has a target for the single cluster.
	plantNoteWithSidecar(t, memFS, vault, "Permanent/ADR.md",
		"---\ntype: fact\ntier: L3\n---\narchitectural decision record body\n")

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: identical}

	var out bytes.Buffer

	// Limit == memberCount keeps the union to just the four identical-vector
	// notes (the ADR scores lower and falls outside the top-N direct hits), so
	// the ADR stays out of the cluster but remains in the L3 index for
	// nearest_l3. This also exercises the advisor's pitfall: direct hits are
	// the top-`limit` by cosine, not a thresholded set.
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"identical union"}, VaultPath: vault, Limit: memberCount, Synthesis: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	// K=0 → exactly one cluster of all members.
	g.Expect(parsed.Clusters).To(HaveLen(1), "K=0 must collapse to one cluster, not zero")
	g.Expect(parsed.Budget.ClustersFound).To(Equal(1))

	single := parsed.Clusters[0]
	// The four identical-vector union members all land in the one cluster
	// (the ADR is L3-distinct vectorwise and is not a synthesis seed here —
	// it shares no body token, so it is not a direct hit).
	g.Expect(single.Members).To(HaveLen(memberCount), "single cluster must hold every union member")

	reps := 0

	for _, member := range single.Members {
		if member.IsRepresentative {
			reps++
		}
	}

	g.Expect(reps).To(Equal(1), "single cluster must have exactly one representative")

	g.Expect(single.NearestL3).NotTo(BeNil(), "single cluster must carry nearest_l3 when an L3 note exists")

	if single.NearestL3 == nil {
		return
	}

	g.Expect(single.NearestL3.Path).To(Equal("Permanent/ADR.md"))
}

// TestQuery_Synthesis_SingleMemberUnionYieldsOneCluster proves the smallest
// case of the K=0 invariant: a union of one direct hit (AutoK returns K=0 via
// len(vectors) < minK) still yields exactly one cluster of one member.
func TestQuery_Synthesis_SingleMemberUnionYieldsOneCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantWithFixedVector(t, memFS, vault, "Permanent/only.md",
		"---\ntype: fact\n---\nthe one and only seed\n", []float32{1, 0, 0, 0})

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"the one and only seed"}, VaultPath: vault, Limit: 1, Synthesis: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	g.Expect(parsed.Clusters).To(HaveLen(1), "a one-member union must still yield one cluster")
	g.Expect(parsed.Clusters[0].Members).To(HaveLen(1))
	g.Expect(parsed.Clusters[0].Members[0].Path).To(Equal("Permanent/only.md"))
	g.Expect(parsed.Clusters[0].Members[0].IsRepresentative).To(BeTrue())
}
