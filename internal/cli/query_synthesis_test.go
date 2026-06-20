package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
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
			fmt.Sprintf("%d.note.md", i+1),
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

	plantWithFixedVector(t, memFS, vault, "a1.md",
		"---\ntype: fact\n---\nunion seed a1\n", groupA)
	plantWithFixedVector(t, memFS, vault, "a2.md",
		"---\ntype: fact\n---\nunion seed a2\n", groupA)
	plantWithFixedVector(t, memFS, vault, "b1.md",
		"---\ntype: fact\n---\nunion seed b1\n", groupB)
	plantWithFixedVector(t, memFS, vault, "b2.md",
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

	plantWithFixedVector(t, memFS, vault, "x.md",
		"---\ntype: fact\n---\nunion content x\n", []float32{1, 0, 0, 0})
	plantWithFixedVector(t, memFS, vault, "y.md",
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
	g.Expect(seen["x.md"]).To(Equal(1))
	g.Expect(seen["y.md"]).To(Equal(1))
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
			fmt.Sprintf("u%d.md", i),
			fmt.Sprintf("---\ntype: fact\n---\nidentical union seed %d\n", i),
			identical)
	}

	// A standalone L3 ADR so nearest_l3 has a target for the single cluster.
	plantNoteWithSidecar(t, memFS, vault, "ADR.md",
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

	g.Expect(single.NearestL3.Path).To(Equal("ADR.md"))
}

// TestQuery_Synthesis_SingleMemberUnionYieldsOneCluster proves the smallest
// case of the K=0 invariant: a union of one direct hit (AutoK returns K=0 via
// len(vectors) < minK) still yields exactly one cluster of one member.
func TestQuery_Synthesis_SingleMemberUnionYieldsOneCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantWithFixedVector(t, memFS, vault, "only.md",
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
	g.Expect(parsed.Clusters[0].Members[0].Path).To(Equal("only.md"))
	g.Expect(parsed.Clusters[0].Members[0].IsRepresentative).To(BeTrue())
}

// TestQuery_SynthesizeL2_CandidateL2sFromWithinClusterMembersOnly locks the
// within-cluster nomination property (Phase 3, reversal of D7): a cluster's
// candidate_l2s must contain ONLY notes that are members of that cluster.
//
// Three sub-properties verified:
//  1. A below-floor L2 (excluded from the matched set, never a cluster member)
//     is NEVER nominated — even though it lives in the vault.
//  2. A note-less cluster (one whose members are all L1, no L2) yields an EMPTY
//     candidate_l2s (explicitly allowed per Phase-3 spec).
//  3. When an L2 IS a cluster member, it IS nominated.
func TestQuery_SynthesizeL2_CandidateL2sFromWithinClusterMembersOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	// Four L1 notes at the query vector — these enter the matched set and cluster.
	// They are NOT L2 notes, so a cluster containing only them has no L2 members.
	for i := range 4 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)
	}

	// An L2 with cosine ~0.20 to the query — below matchRelevanceFloor (0.25),
	// so it is dropped from the matched set by the floor filter and is never a
	// cluster member. Under within-cluster nomination it must NOT appear in
	// candidate_l2s for any cluster.
	belowFloor := []float32{0.2, 0.98, 0, 0} // cosine to {1,0,0,0} ≈ 0.20
	plantDualVector(t, memFS, vault, "below-floor.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", belowFloor, belowFloor)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	// The below-floor L2 must not appear as a cluster member.
	for _, parsedCluster := range parsed.Clusters {
		for _, member := range parsedCluster.Members {
			g.Expect(member.Path).NotTo(ContainSubstring("below-floor"),
				"the below-floor L2 must be excluded from the matched set and cluster membership")
		}
	}

	// Every cluster has only L1 members (no L2 members) — candidate_l2s must be
	// empty (nil/absent in YAML). The below-floor L2 must NOT be nominated.
	for _, parsedCluster := range parsed.Clusters {
		for _, cand := range parsedCluster.CandidateL2s {
			g.Expect(cand.Path).NotTo(ContainSubstring("below-floor"),
				"a note absent from the cluster's membership must never appear in candidate_l2s")
		}

		// No L2 note is a member of any cluster, so candidate_l2s must be empty.
		g.Expect(parsedCluster.CandidateL2s).To(BeEmpty(),
			"a cluster with no L2 note members must yield an empty candidate_l2s")
	}
}

// TestQuery_SynthesizeL2_CandidateL2sIncludesL2ClusterMembers verifies the
// positive case: when an L2 note IS a cluster member (above the relevance floor
// and clustered with the other matches), it IS nominated in candidate_l2s.
func TestQuery_SynthesizeL2_CandidateL2sIncludesL2ClusterMembers(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	// L1 notes at the query vector — seed the matched set.
	for i := range 4 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)
	}

	// An L2 well above the relevance floor — it enters the matched set and clusters
	// with the L1 notes. Under within-cluster nomination it MUST appear in candidate_l2s.
	aboveFloor := []float32{1, 0.1, 0, 0} // cosine to {1,0,0,0} ≈ 0.995
	plantDualVector(t, memFS, vault, "in-cluster.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", aboveFloor, aboveFloor)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	// Find the cluster that contains in-cluster.fact.md as a member.
	foundNominated := false

	for _, parsedCluster := range parsed.Clusters {
		memberPaths := make(map[string]bool)

		for _, member := range parsedCluster.Members {
			memberPaths[member.Path] = true
		}

		if !memberPaths["in-cluster.fact.md"] {
			continue
		}

		// This cluster has the L2 as a member; it must appear in candidate_l2s.
		candPaths := make([]string, 0, len(parsedCluster.CandidateL2s))

		for _, cand := range parsedCluster.CandidateL2s {
			candPaths = append(candPaths, cand.Path)
		}

		g.Expect(candPaths).To(ContainElement("in-cluster.fact.md"),
			"an L2 that is a cluster member must be nominated in candidate_l2s")

		foundNominated = true
	}

	g.Expect(foundNominated).To(BeTrue(),
		"in-cluster.fact.md must be a cluster member and appear in candidate_l2s")
}

// TestQuery_SynthesizeL2_CandidateL2sTopK verifies the within-cluster top-K
// invariant (Phase 3):
//   - candidate_l2s for each cluster is bounded by candidateL2K (5) and by the
//     number of L2 note members in that cluster;
//   - candidates are drawn from WITHIN the cluster (no L2 from another cluster
//     or from the full vault appears in a cluster it didn't join);
//   - candidates are sorted cosine-desc.
//
// To isolate the "fewer members → fewer candidates" property, the test plants
// exactly 3 L2 notes so that all 3 land in the matched set and one cluster,
// and expects exactly 3 candidates (fewer than candidateL2K=5). Notes are tight
// around {1,0,0,0} so AutoK collapses to a single cluster via singleClusterReport.
func TestQuery_SynthesizeL2_CandidateL2sTopK(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	// Four L1 notes at the query vector — seed the cluster.
	for i := range 4 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)
	}

	// Three L2 notes, all well above the relevance floor and tight enough to
	// the query that AutoK won't split them away from the L1 cluster.
	l2Vecs := [][]float32{
		{1, 0, 0, 0},    // cosine 1.0
		{1, 0.05, 0, 0}, // cosine ~0.999
		{1, 0.10, 0, 0}, // cosine ~0.995
	}
	l2Paths := make([]string, len(l2Vecs))

	for i, vec := range l2Vecs {
		l2Paths[i] = fmt.Sprintf("l2-%d.fact.md", i)
		plantDualVector(t, memFS, vault, l2Paths[i],
			"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", vec, vec)
	}

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	// Per-cluster invariants: candidate count ≤ min(5, L2-member-count); all
	// candidates are L2 members of that cluster; sorted cosine desc.
	var rawMap map[string]any

	g.Expect(yaml.Unmarshal(out.Bytes(), &rawMap)).NotTo(HaveOccurred())

	rawClusters, _ := rawMap["clusters"].([]any)

	assertWithinClusterL2Bounds(g, parsed, l2Paths, rawClusters)

	// Total L2 candidates across all clusters must cover every L2 in the matched
	// set (3 in this fixture — fewer than candidateL2K=5).
	totalL2CandidatesByPath := make(map[string]bool)

	for _, parsedCluster := range parsed.Clusters {
		for _, cand := range parsedCluster.CandidateL2s {
			totalL2CandidatesByPath[cand.Path] = true
		}
	}

	for _, l2Path := range l2Paths {
		g.Expect(totalL2CandidatesByPath).To(HaveKey(l2Path),
			"L2 note %q is in the matched set and must appear in candidate_l2s of its cluster", l2Path)
	}
}

// TestQuery_SynthesizeL2_CandidateUsesStrongerAxis verifies the candidate_l2s
// cosine is the max of the situation- and body-axis cosines to the centroid (the
// "either axis" gate). The L2's situation vector is orthogonal to the centroid
// (cosine 0) while its body vector is on-axis (cosine 1); the emitted cosine must
// follow the stronger body axis, not the weaker situation axis.
func TestQuery_SynthesizeL2_CandidateUsesStrongerAxis(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Four L1 notes anchor the cluster centroid at {1,0,0,0}.
	centroidVec := []float32{1, 0, 0, 0}
	for i := range 4 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", centroidVec, centroidVec)
	}

	// One L2 whose situation axis is orthogonal to the centroid (cosine 0) but
	// whose body axis is on-axis (cosine 1). bestVector clusters it by the
	// query-winning body axis, so it joins the centroid cluster; eitherAxisCosine
	// must then report the stronger body-axis cosine (~1.0).
	orthogonalSit := []float32{0, 1, 0, 0}
	onAxisBody := []float32{1, 0, 0, 0}
	plantDualVector(t, memFS, vault, "l2-bodywins.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", orthogonalSit, onAxisBody)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: centroidVec}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var raw map[string]any

	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	clusters, ok := raw["clusters"].([]any)
	g.Expect(ok).To(BeTrue(), "clusters must be a sequence")
	g.Expect(clusters).NotTo(BeEmpty())

	first, ok := clusters[0].(map[string]any)
	g.Expect(ok).To(BeTrue(), "a cluster must be a mapping")

	candidates, ok := first["candidate_l2s"].([]any)
	g.Expect(ok).To(BeTrue(), "candidate_l2s must be a sequence")
	g.Expect(candidates).NotTo(BeEmpty())

	bodywins, ok := candidates[0].(map[string]any)
	g.Expect(ok).To(BeTrue(), "a candidate must be a mapping")
	g.Expect(bodywins["path"]).To(Equal("l2-bodywins.fact.md"))
	g.Expect(yamlCosine(g, bodywins["cosine"])).To(BeNumerically(">", 0.99),
		"either-axis gate must report the stronger body-axis cosine, not the orthogonal situation axis")
}

// TestQuery_SynthesizeL2_CoverL2NotCentroidFirst_AppearsInTopK verifies that
// within a cluster, an L2 that is NOT the centroid-nearest still appears in
// candidate_l2s when it is a cluster member — the ranking is by centroid cosine
// desc, so any L2 member with a lower cosine than another still surfaces so long
// as there are fewer than candidateL2K L2 members ahead of it. The fixture
// plants an L2 DISTRACTOR at the cluster centroid (cosine 1.0) and a COVER L2
// at cosine ~0.9: both are cluster members, distractor ranks first, cover
// still appears. The FAR L2 (cosine 0 to the query) is excluded by
// matchRelevanceFloor and is never a cluster member, so it is never nominated.
func TestQuery_SynthesizeL2_CoverL2NotCentroidFirst_AppearsInTopK(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Four L1 notes anchor the note cluster centroid near {1,0,0,0}.
	noteVec := []float32{1, 0, 0, 0}
	for i := range 4 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", noteVec, noteVec)
	}

	// DISTRACTOR: sits on the centroid axis (cosine 1.0 → ranks #1 in candidates).
	distractor := []float32{1, 0, 0, 0}
	plantDualVector(t, memFS, vault, "l2-distractor.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", distractor, distractor)

	// COVER: cosine ~0.9 to centroid — lower than distractor but still a cluster
	// member; must appear in candidate_l2s because there are <candidateL2K L2
	// members ahead of it.
	cover := []float32{0.9, 0.436, 0, 0}
	plantDualVector(t, memFS, vault, "l2-cover.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", cover, cover)

	// FAR: cosine 0 to the query — below matchRelevanceFloor (0.25), so it is
	// excluded from the matched set and never becomes a cluster member.
	far := []float32{0, 0, 1, 0}
	plantDualVector(t, memFS, vault, "l2-far.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", far, far)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: noteVec}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	// Find the cluster containing both distractor and cover as members, then verify
	// the within-cluster nomination: distractor ranks first, cover also appears.
	foundCluster := false

	for _, parsedCluster := range parsed.Clusters {
		memberPaths := clusterMemberPaths(parsedCluster)

		if !memberPaths["l2-distractor.fact.md"] || !memberPaths["l2-cover.fact.md"] {
			continue
		}

		foundCluster = true
		candPaths := candidatePaths(parsedCluster)

		g.Expect(candPaths).To(ContainElement("l2-cover.fact.md"),
			"the covering L2 is a cluster member and must appear in candidate_l2s")
		g.Expect(candPaths).NotTo(ContainElement("l2-far.fact.md"),
			"far L2 is below the relevance floor and is never a cluster member")
		g.Expect(candPaths[0]).To(Equal("l2-distractor.fact.md"),
			"the centroid-nearest distractor ranks first in the within-cluster list")

		rawCandidates := rawCandidatesForClusterContaining(g, out.Bytes(), "l2-distractor.fact.md")
		expectCandidatesSortedDesc(g, rawCandidates)

		break
	}

	g.Expect(foundCluster).To(BeTrue(),
		"a cluster containing both l2-distractor and l2-cover must exist")
}

// TestQuery_SynthesizeL2_EmitsCandidateL2sSlice verifies the payload emits a
// candidate_l2s sequence per cluster (the plural top-K form) and no longer
// carries the singular nearest_l2 key.
func TestQuery_SynthesizeL2_EmitsCandidateL2sSlice(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	for i := range 4 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)
	}

	plantDualVector(t, memFS, vault, "near.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var raw map[string]any

	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	clusters, ok := raw["clusters"].([]any)
	g.Expect(ok).To(BeTrue(), "clusters must be a sequence")
	g.Expect(clusters).NotTo(BeEmpty())

	first, ok := clusters[0].(map[string]any)
	g.Expect(ok).To(BeTrue(), "a cluster must be a mapping")

	_, hasOld := first["nearest_l2"]
	g.Expect(hasOld).To(BeFalse(), "nearest_l2 must be removed; use candidate_l2s")

	candidates, hasCandidates := first["candidate_l2s"]
	g.Expect(hasCandidates).To(BeTrue(), "candidate_l2s must appear in synthesize-l2 cluster output")

	_, isSlice := candidates.([]any)
	g.Expect(isSlice).To(BeTrue(), "candidate_l2s must be a sequence, not a scalar")
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
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	}

	// The only L2 is far from the centroid: cos(synthVec, far) ~ 0.30.
	far := []float32{0.3, 0.954, 0, 0}
	plantDualVector(t, memFS, vault, "dup.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", far, far)

	parsed := runSynthesizeL2(t, memFS, vault)

	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	// Every cluster surfaces the L2 (it exists in the vault), and at least one
	// reports a raw cosine below 0.80 — proof the binary applies no band cutoff.
	sawSubBandRawCosine := false

	for _, cluster := range parsed.Clusters {
		g.Expect(cluster.CandidateL2s).NotTo(BeEmpty(), "a far L2 is still emitted; the binary applies no band cutoff")
		g.Expect(cluster.CandidateL2s[0].Cosine).To(BeNumerically(">", float32(0.0)))

		if cluster.CandidateL2s[0].Cosine < 0.8 {
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

	plantDualVector(t, memFS, vault, "1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	plantDualVector(t, memFS, vault, "2.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	plantDualVector(t, memFS, vault, "3.adr.md",
		"---\ntype: fact\ntier: L3\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())

	parsed := runSynthesizeL2(t, memFS, vault)

	for _, c := range parsed.Clusters {
		for _, m := range c.Members {
			g.Expect(m.Path).NotTo(ContainSubstring("3.adr"), "L3 must not be clustered in synthesize-l2")
		}
	}

	// The L3 note must also be absent from the top-level items[] channel: items
	// derive from the same L1+L2-only union, so L3 is excluded there too.
	for _, item := range parsed.Items {
		g.Expect(item.Path).NotTo(ContainSubstring("3.adr"), "L3 must not surface in items[] in synthesize-l2")
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

// TestQuery_SynthesizeL2_IncludesMatchedChunksInClustering verifies D1: when
// ChunksDir is set, matched chunks appear as cluster members in the unified
// --synthesize-l2 clustering (not in a separate chunk-clusters channel).
func TestQuery_SynthesizeL2_IncludesMatchedChunksInClustering(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	// Three L1 notes.
	for i := range 3 {
		plantDualVector(t, memFS, vault, fmt.Sprintf("%d.ep.md", i+1),
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)
	}

	// Two chunk records at the same vector.
	records := []chunk.Record{
		{
			Source:      "/sessions/s.jsonl",
			Anchor:      "turn-1",
			ContentHash: chunk.HashText("chunk alpha one"),
			Text:        "chunk alpha one",
			Vector:      queryVec,
		},
		{
			Source:      "/sessions/s.jsonl",
			Anchor:      "turn-2",
			ContentHash: chunk.HashText("chunk alpha two"),
			Text:        "chunk alpha two",
			Vector:      queryVec,
		},
	}

	data, encErr := chunk.EncodeRecords(records)
	g.Expect(encErr).NotTo(HaveOccurred())

	if encErr != nil {
		return
	}

	memFS.files["/chunks/s.jsonl"] = data

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	deps.ListChunkIndexes = func(string) ([]string, error) {
		return []string{"/chunks/s.jsonl"}, nil
	}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:      []string{"alpha"},
			VaultPath:    vault,
			SynthesizeL2: true,
			ChunksDir:    "/chunks",
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var raw map[string]any

	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	// items[] must include chunk-kind items.
	items, _ := raw["items"].([]any)
	kinds := map[string]bool{}

	for _, item := range items {
		mapped, isMap := item.(map[string]any)
		g.Expect(isMap).To(BeTrue(), "each item must be a mapping")

		kind, _ := mapped["kind"].(string)
		kinds[kind] = true
	}

	g.Expect(kinds["chunk"]).To(BeTrue(), "chunk items must appear in synthesize-l2 items[]")

	// clusters must have NO phrase="chunks" channel (D1: one unified pass only).
	clusters, _ := raw["clusters"].([]any)
	g.Expect(clusters).NotTo(BeEmpty())

	sawChunkMember := false

	for _, cl := range clusters {
		clusterMap, isMap := cl.(map[string]any)
		g.Expect(isMap).To(BeTrue(), "each cluster must be a mapping")

		phrase, _ := clusterMap["phrase"].(string)
		g.Expect(phrase).NotTo(Equal("chunks"),
			"D1: synthesize-l2 must not emit a separate chunks cluster channel")

		members, _ := clusterMap["members"].([]any)
		for _, member := range members {
			memberMap, isMemberMap := member.(map[string]any)
			g.Expect(isMemberMap).To(BeTrue(), "each member must be a mapping")

			path, _ := memberMap["path"].(string)
			if strings.Contains(path, "#") {
				sawChunkMember = true
			}
		}
	}

	g.Expect(sawChunkMember).To(BeTrue(),
		"at least one cluster member must be a chunk (source#anchor) in synthesize-l2")
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
	plantDualVector(t, memFS, vault, "1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	// An L2 near-duplicate of the centroid: cos(synthVec, nearDup) ~ 0.995.
	nearDup := []float32{1, 0.1, 0, 0}
	plantDualVector(t, memFS, vault, "2.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", nearDup, nearDup)

	parsed := runSynthesizeL2(t, memFS, vault)

	g.Expect(parsed.Clusters).NotTo(BeEmpty())
	g.Expect(parsed.Clusters[0].CandidateL2s).NotTo(BeEmpty())
	g.Expect(parsed.Clusters[0].CandidateL2s[0].Cosine).To(BeNumerically(">=", float32(0.95)))
}

// TestQuery_SynthesizeL2_NearestL2PresentWhenL2Exists verifies that when a
// matching L2 exists in the vault, each cluster carries a non-nil nearest_l2.
func TestQuery_SynthesizeL2_NearestL2PresentWhenL2Exists(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantDualVector(t, memFS, vault, "1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	plantDualVector(t, memFS, vault, "2.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())

	parsed := runSynthesizeL2(t, memFS, vault)

	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	for _, c := range parsed.Clusters {
		g.Expect(c.CandidateL2s).NotTo(BeEmpty(), "an existing matching L2 must surface in candidate_l2s")
		g.Expect(c.CandidateL2s[0].Path).To(Equal("2.fact.md"))
	}
}

// TestQuery_SynthesizeL2_NoL2_NearestL2Nil verifies that with only L1 notes in
// the vault, no cluster carries a nearest_l2 (nothing to crystallize against).
func TestQuery_SynthesizeL2_NoL2_NearestL2Nil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantDualVector(t, memFS, vault, "1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())
	plantDualVector(t, memFS, vault, "2.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", synthVec(), synthVec())

	parsed := runSynthesizeL2(t, memFS, vault)

	g.Expect(parsed.Clusters).NotTo(BeEmpty())

	for _, c := range parsed.Clusters {
		g.Expect(c.CandidateL2s).To(BeEmpty(), "no L2 in vault means no candidate_l2s")
	}
}

// assertWithinClusterL2Bounds verifies per-cluster within-cluster nomination
// invariants: candidate count ≤ min(5, L2-member-count); all candidates are
// L2 note members of that cluster; candidates sorted cosine desc.
func assertWithinClusterL2Bounds(g *WithT, parsed queryParsed, l2Paths []string, rawClusters []any) {
	for idx, parsedCluster := range parsed.Clusters {
		l2MemberPaths := make(map[string]bool)

		for _, member := range parsedCluster.Members {
			if slices.Contains(l2Paths, member.Path) {
				l2MemberPaths[member.Path] = true
			}
		}

		g.Expect(len(parsedCluster.CandidateL2s)).To(BeNumerically("<=", 5),
			"cluster %d: candidate_l2s must not exceed candidateL2K=5", idx)
		g.Expect(len(parsedCluster.CandidateL2s)).To(BeNumerically("<=", len(l2MemberPaths)),
			"cluster %d: candidate count cannot exceed the number of L2 note members", idx)

		for _, cand := range parsedCluster.CandidateL2s {
			g.Expect(l2MemberPaths).To(HaveKey(cand.Path),
				"cluster %d: candidate %q must be an L2 member of this cluster", idx, cand.Path)
		}

		if idx < len(rawClusters) {
			if rawCluster, ok := rawClusters[idx].(map[string]any); ok {
				rawCands, _ := rawCluster["candidate_l2s"].([]any)
				expectCandidatesSortedDesc(g, rawCands)
			}
		}
	}
}

// candidatePaths collects candidate_l2s paths from a parsed cluster.
func candidatePaths(parsedCluster struct {
	ID         int     `yaml:"id"`
	Phrase     string  `yaml:"phrase"`
	Size       int     `yaml:"size"`
	Silhouette float64 `yaml:"silhouette"`
	Members    []struct {
		Path             string  `yaml:"path"`
		Score            float32 `yaml:"score"`
		IsRepresentative bool    `yaml:"is_representative"`
	} `yaml:"members"`
	NearestL3 *struct {
		Path   string  `yaml:"path"`
		Cosine float32 `yaml:"cosine"`
	} `yaml:"nearest_l3,omitempty"`
	CandidateL2s []struct {
		Path   string  `yaml:"path"`
		Cosine float32 `yaml:"cosine"`
	} `yaml:"candidate_l2s"`
}) []string {
	paths := make([]string, 0, len(parsedCluster.CandidateL2s))

	for _, cand := range parsedCluster.CandidateL2s {
		paths = append(paths, cand.Path)
	}

	return paths
}

// clusterMemberPaths returns a set of member paths for a parsed cluster.
func clusterMemberPaths(parsedCluster struct {
	ID         int     `yaml:"id"`
	Phrase     string  `yaml:"phrase"`
	Size       int     `yaml:"size"`
	Silhouette float64 `yaml:"silhouette"`
	Members    []struct {
		Path             string  `yaml:"path"`
		Score            float32 `yaml:"score"`
		IsRepresentative bool    `yaml:"is_representative"`
	} `yaml:"members"`
	NearestL3 *struct {
		Path   string  `yaml:"path"`
		Cosine float32 `yaml:"cosine"`
	} `yaml:"nearest_l3,omitempty"`
	CandidateL2s []struct {
		Path   string  `yaml:"path"`
		Cosine float32 `yaml:"cosine"`
	} `yaml:"candidate_l2s"`
}) map[string]bool {
	memberPaths := make(map[string]bool, len(parsedCluster.Members))

	for _, member := range parsedCluster.Members {
		memberPaths[member.Path] = true
	}

	return memberPaths
}

// expectCandidatesSortedDesc asserts candidate_l2s cosines are non-increasing
// (the centroid-cosine sort order).
func expectCandidatesSortedDesc(g *WithT, candidates []any) {
	for i := 1; i < len(candidates); i++ {
		prev, prevOK := candidates[i-1].(map[string]any)
		g.Expect(prevOK).To(BeTrue())

		curr, currOK := candidates[i].(map[string]any)
		g.Expect(currOK).To(BeTrue())

		prevCos := yamlCosine(g, prev["cosine"])
		currCos := yamlCosine(g, curr["cosine"])
		g.Expect(prevCos).To(BeNumerically(">=", currCos),
			"candidate_l2s must be sorted centroid-cosine desc (index %d >= %d)", i-1, i)
	}
}

// rawCandidatesForClusterContaining finds the raw YAML cluster containing the
// given member path and returns its candidate_l2s list.
func rawCandidatesForClusterContaining(g *WithT, payload []byte, memberPath string) []any {
	var rawMap map[string]any

	g.Expect(yaml.Unmarshal(payload, &rawMap)).NotTo(HaveOccurred())

	rawClusters, ok := rawMap["clusters"].([]any)
	g.Expect(ok).To(BeTrue(), "clusters must be a sequence")

	for _, rawCluster := range rawClusters {
		rawClusterMap, clOK := rawCluster.(map[string]any)
		if !clOK {
			continue
		}

		rawMembers, membOK := rawClusterMap["members"].([]any)
		if !membOK {
			continue
		}

		for _, rawMember := range rawMembers {
			rawMemberMap, mOK := rawMember.(map[string]any)
			if !mOK {
				continue
			}

			if rawMemberMap["path"] == memberPath {
				rawCands, _ := rawClusterMap["candidate_l2s"].([]any)

				return rawCands
			}
		}
	}

	return nil
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

// yamlCosine coerces a YAML-decoded cosine to float64. go.yaml.in/yaml/v3
// decodes an integral value (e.g. a perfect cosine of 1) as int, not float64,
// so a bare `.(float64)` assertion would silently read it as zero.
func yamlCosine(g *WithT, v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		g.Expect(v).To(BeNumerically(">=", 0), "cosine must decode to a number, got %T", v)

		return 0
	}
}
