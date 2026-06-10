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

// TestQuery_SynthesizeL2_EmitsRawCosineNoBand verifies that a cluster whose
// centroid is FAR from the only L2 still emits nearest_l2 with that raw low
// cosine — the binary applies no <0.80 cutoff (the skill bands later).
func TestQuery_SynthesizeL2_EmitsRawCosineNoBand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Many L1 notes at synthVec keep the cluster centroid near synthVec even
	// after the single far L2 joins the clustered set, so the centroid stays
	// far from that L2 (cos well below the skill's 0.80 create-band).
	const l1Count = 10
	for i := range l1Count {
		plantDualVector(t, memFS, vault, fmt.Sprintf("Permanent/%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	}

	// The only L2 is far from the centroid: cos(synthVec, far) ~ 0.30.
	far := []float32{0.3, 0.954, 0, 0}
	plantDualVector(t, memFS, vault, "Permanent/dup.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", far, far)

	parsed := runSynthesizeL2(t, memFS, vault)

	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	// Every cluster surfaces the L2 (it exists in the vault), and at least one
	// reports a raw cosine below 0.80 — proof the binary applies no band cutoff.
	sawSubBandRawCosine := false

	for _, cluster := range parsed.Clusters {
		g.Expect(cluster.NearestL2).NotTo(BeNil(), "a far L2 is still emitted; the binary applies no band cutoff")
		g.Expect(cluster.NearestL2.Cosine).To(BeNumerically(">", float32(0.0)))

		if cluster.NearestL2.Cosine < 0.8 {
			sawSubBandRawCosine = true
		}
	}

	g.Expect(sawSubBandRawCosine).To(BeTrue(),
		"a raw cosine below the skill's 0.80 create-band must still be emitted (no cutoff in the binary)")
}

// TestQuery_SynthesizeL2_ExcludesL3FromClusters verifies the pre-clustering
// L1+L2 constraint: an L3 note that matches the query must never appear as a
// cluster member in synthesize-l2 mode.
func TestQuery_SynthesizeL2_ExcludesL3FromClusters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantDualVector(t, memFS, vault, "Permanent/1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	plantDualVector(t, memFS, vault, "Permanent/2.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	plantDualVector(t, memFS, vault, "Permanent/3.adr.md",
		"---\ntype: fact\ntier: L3\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())

	parsed := runSynthesizeL2(t, memFS, vault)

	for _, c := range parsed.Clusters {
		for _, m := range c.Members {
			g.Expect(m.Path).NotTo(ContainSubstring("3.adr"), "L3 must not be clustered in synthesize-l2")
		}
	}
}

// TestQuery_SynthesizeL2_FlagAndSynthesisAreMutuallyExclusive verifies that
// requesting both modes in one invocation is rejected with the sentinel.
func TestQuery_SynthesizeL2_FlagAndSynthesisAreMutuallyExclusive(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := newQueryDeps(newInMemoryFS())
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: synthVec()}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"x"}, VaultPath: t.TempDir(), Synthesis: true, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).To(MatchError(cli.ErrQueryModeConflict))
}

// TestQuery_SynthesizeL2_NearDuplicateL2_CosineAtLeast095 verifies the raw
// nearest_l2.cosine is >= 0.95 when an existing L2 is a near-duplicate of the
// cluster centroid. The binary applies no band; the high cosine is reported raw.
func TestQuery_SynthesizeL2_NearDuplicateL2_CosineAtLeast095(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Clustered L1 notes sit exactly at synthVec, so the centroid is synthVec.
	plantDualVector(t, memFS, vault, "Permanent/1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	// An L2 near-duplicate of the centroid: cos(synthVec, nearDup) ~ 0.995.
	nearDup := []float32{1, 0.1, 0, 0}
	plantDualVector(t, memFS, vault, "Permanent/2.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", nearDup, nearDup)

	parsed := runSynthesizeL2(t, memFS, vault)

	g.Expect(parsed.Clusters).NotTo(BeEmpty())
	g.Expect(parsed.Clusters[0].NearestL2).NotTo(BeNil())
	g.Expect(parsed.Clusters[0].NearestL2.Cosine).To(BeNumerically(">=", float32(0.95)))
}

// TestQuery_SynthesizeL2_NearestL2FromFullVaultNotJustClustered locks the
// design property that the L2 nearest-index is gathered from the FULL hit set,
// not just the clustered (union) members. With Limit:2 the two top-scored L1
// notes fill the union and the lower-scored L2 is truncated out — so the L2 is
// NOT a cluster member. It must still surface as nearest_l2 because the index
// comes from every vault L2, not the clustered set. If the gather source were
// ever narrowed to the union, this test would fail.
func TestQuery_SynthesizeL2_NearestL2FromFullVaultNotJustClustered(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	// Two L1 notes at the query vector (score 1.0) fill the Limit:2 union.
	plantDualVector(t, memFS, vault, "Permanent/1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)
	plantDualVector(t, memFS, vault, "Permanent/2.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)

	// A lower-scored L2 (cos ~0.30 to the query) ranks third, so Limit:2
	// truncates it out of the union — it is never clustered, but stays in hits.
	lowMatch := []float32{0.3, 0.954, 0, 0}
	plantDualVector(t, memFS, vault, "Permanent/3.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", lowMatch, lowMatch)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, Limit: 2, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	// The L2 is not a cluster member (truncated out of the Limit:2 union)...
	for _, cluster := range parsed.Clusters {
		for _, member := range cluster.Members {
			g.Expect(member.Path).NotTo(ContainSubstring("3.fact"),
				"the lower-scored L2 must be truncated out of the clustered union")
		}
	}

	// ...yet it still surfaces as nearest_l2 (index gathered from FULL hits).
	for _, cluster := range parsed.Clusters {
		g.Expect(cluster.NearestL2).NotTo(BeNil(),
			"nearest_l2 is gathered from every vault L2, not just clustered members")
		g.Expect(cluster.NearestL2.Path).To(Equal("Permanent/3.fact.md"))
	}
}

// TestQuery_SynthesizeL2_NearestL2PresentWhenL2Exists verifies that when a
// matching L2 exists in the vault, each cluster carries a non-nil nearest_l2.
func TestQuery_SynthesizeL2_NearestL2PresentWhenL2Exists(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantDualVector(t, memFS, vault, "Permanent/1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	plantDualVector(t, memFS, vault, "Permanent/2.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())

	parsed := runSynthesizeL2(t, memFS, vault)

	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	for _, c := range parsed.Clusters {
		g.Expect(c.NearestL2).NotTo(BeNil(), "an existing matching L2 must surface as nearest_l2")
		g.Expect(c.NearestL2.Path).To(Equal("Permanent/2.fact.md"))
	}
}

// TestQuery_SynthesizeL2_NoL2_NearestL2Nil verifies that with only L1 notes in
// the vault, no cluster carries a nearest_l2 (nothing to crystallize against).
func TestQuery_SynthesizeL2_NoL2_NearestL2Nil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantDualVector(t, memFS, vault, "Permanent/1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	plantDualVector(t, memFS, vault, "Permanent/2.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())

	parsed := runSynthesizeL2(t, memFS, vault)

	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	for _, c := range parsed.Clusters {
		g.Expect(c.NearestL2).To(BeNil(), "no L2 in vault means no nearest_l2")
	}
}

// runSynthesizeL2 plants nothing; it embeds synthVec() and runs the mode
// against the given memFS/vault, returning the parsed payload.
func runSynthesizeL2(t *testing.T, memFS *inMemoryFS, vault string) queryParsed {
	t.Helper()

	g := NewWithT(t)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: synthVec()}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return queryParsed{}
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	return parsed
}

// --- Phase 2: --synthesize-l2 mode ---

// synthVec returns a fresh query/plant vector for the synthesize-l2 tests so
// the fixed-vector embedder, the planted notes, and the cluster centroid all
// align. A function (not a shared global) keeps parallel tests free of shared
// mutable state.
func synthVec() []float32 { return []float32{1, 0, 0, 0} }
