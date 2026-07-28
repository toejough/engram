package cli_test

import (
	"math"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestApplyVocabKHysteresis_DegeneratePinnedFallsBackToFree verifies a
// pinned derivation with no structure (K=0) never wins.
func TestApplyVocabKHysteresis_DegeneratePinnedFallsBackToFree(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	free := cli.ExportVocabDerivation{K: 3, Silhouette: 0.50}
	pinned := cli.ExportVocabDerivation{}

	g.Expect(cli.ExportApplyVocabKHysteresis(free, pinned)).To(Equal(free))
}

// TestApplyVocabKHysteresis_FreeWinsOutsideEpsilon verifies a clearly better
// free optimum overrides the previous K.
func TestApplyVocabKHysteresis_FreeWinsOutsideEpsilon(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	free := cli.ExportVocabDerivation{K: 3, Silhouette: 0.50}
	pinned := cli.ExportVocabDerivation{K: 2, Silhouette: 0.50 - 2*float64(cli.ExportVocabSilhouetteEpsilon)}

	g.Expect(cli.ExportApplyVocabKHysteresis(free, pinned)).To(Equal(free))
}

// ── Task 1.3: silhouette hysteresis + min-K floor ────────────────────────────

// TestApplyVocabKHysteresis_PrefersPreviousKWithinEpsilon verifies that when
// the previous-K derivation scores within epsilon of the free optimum, the
// previous K is kept (name/structure stability beats a marginal gain).
func TestApplyVocabKHysteresis_PrefersPreviousKWithinEpsilon(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	free := cli.ExportVocabDerivation{K: 3, Silhouette: 0.50}
	pinned := cli.ExportVocabDerivation{K: 2, Silhouette: 0.50 - float64(cli.ExportVocabSilhouetteEpsilon)/2}

	g.Expect(cli.ExportApplyVocabKHysteresis(free, pinned)).To(Equal(pinned))
}

// TestDeriveVocabClusters_CentroidIsMeanOfMembers verifies the derived
// centroid of each cluster equals the arithmetic mean of its members'
// vectors.
func TestDeriveVocabClusters_CentroidIsMeanOfMembers(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := twoBlobNotes()

	derivation, err := cli.ExportDeriveVocabClusters(notes, noPreviousK, testDeriveSeed)
	g.Expect(err).NotTo(HaveOccurred())

	vectorsByName := make(map[string][]float32, len(notes))
	for _, note := range notes {
		vectorsByName[note.Name] = note.Vector
	}

	for _, derived := range derivation.Clusters {
		g.Expect(derived.Members).NotTo(BeEmpty())

		mean := meanOfNamed(vectorsByName, derived.Members)
		g.Expect(derived.Centroid).To(HaveLen(len(mean)))

		for dim := range mean {
			g.Expect(derived.Centroid[dim]).To(BeNumerically("~", mean[dim], centroidTolerance))
		}
	}
}

// TestDeriveVocabClusters_ConservesAllNotes is a property test: every input
// note appears in exactly one cluster (allocation conservation), regardless
// of blob sizes.
func TestDeriveVocabClusters_ConservesAllNotes(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		eastCount := rapid.IntRange(minBlobSize, maxBlobSize).Draw(rt, "eastCount")
		northCount := rapid.IntRange(minBlobSize, maxBlobSize).Draw(rt, "northCount")

		notes := blobNotes("east", eastCount, []float32{1, 0})
		notes = append(notes, blobNotes("north", northCount, []float32{0, 1})...)

		derivation, err := cli.ExportDeriveVocabClusters(notes, noPreviousK, testDeriveSeed)
		g.Expect(err).NotTo(HaveOccurred())

		if derivation.K == 0 {
			return // no structure found — nothing to conserve
		}

		seen := make(map[string]int, len(notes))

		for _, derived := range derivation.Clusters {
			for _, member := range derived.Members {
				seen[member]++
			}
		}

		g.Expect(seen).To(HaveLen(len(notes)))

		for name, count := range seen {
			g.Expect(count).To(Equal(1), "note %s assigned %d times", name, count)
		}
	})
}

// TestDeriveVocabClusters_DegenerateCorpusYieldsZero verifies the
// noise-rejection floor still fires on truly structureless input: identical
// vectors have silhouette 0, below even the lowered floor, so derivation
// reports no structure.
func TestDeriveVocabClusters_DegenerateCorpusYieldsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := make([]cli.ExportNoteVector, 0, degenerateCorpusSize)
	for i := range degenerateCorpusSize {
		name := "same-" + string(rune('a'+i)) + ".md"
		notes = append(notes, cli.ExportNoteVector{Name: name, Vector: []float32{1, 0}})
	}

	derivation, err := cli.ExportDeriveVocabClusters(notes, noPreviousK, testDeriveSeed)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(derivation.K).To(BeZero())
	g.Expect(derivation.Clusters).To(BeEmpty())
}

// TestDeriveVocabClusters_DiffuseCorpusYieldsNonZeroK pins the noise-floor
// semantics of the silhouette floor: a diffuse-but-structured corpus whose
// peak silhouette sits in the ~0.05-0.09 band (where the real 597-note vault
// measured its 0.0987 peak) must yield a non-zero K — never a no-op. The floor
// exists only to reject degenerate no-structure input, not to arbitrate
// among plausible clusterings.
func TestDeriveVocabClusters_DiffuseCorpusYieldsNonZeroK(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	derivation, err := cli.ExportDeriveVocabClusters(diffuseCorpusNotes(), noPreviousK, testDeriveSeed)
	g.Expect(err).NotTo(HaveOccurred())

	// The fixture's measured peak silhouette is ~0.074 — inside the real-vault
	// operating band and below the refuted 0.10 floor that zeroed derivation.
	g.Expect(derivation.Silhouette).To(BeNumerically("<", 0.10))
	g.Expect(derivation.Silhouette).To(BeNumerically(">=", 0.02))
	g.Expect(derivation.K).To(BeNumerically(">", 0), "diffuse corpus must derive a non-zero K, not no-op")
	g.Expect(derivation.Clusters).To(HaveLen(derivation.K))
}

// ── Task 1.1: derivation core — cluster non-definition note vectors ──────────

// TestDeriveVocabClusters_GroupsMembersByCluster verifies that derivation
// clusters two well-separated note groups into K=2 clusters and that the
// note↔vector correlation is owned correctly: each cluster's member list
// contains exactly the note names of one group.
func TestDeriveVocabClusters_GroupsMembersByCluster(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := twoBlobNotes()

	derivation, err := cli.ExportDeriveVocabClusters(notes, noPreviousK, testDeriveSeed)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(derivation.K).To(Equal(2))
	g.Expect(derivation.Clusters).To(HaveLen(2))

	groups := make([][]string, 0, len(derivation.Clusters))
	for _, derived := range derivation.Clusters {
		groups = append(groups, derived.Members)
	}

	g.Expect(groups).To(ConsistOf(
		ConsistOf("east-a.md", "east-b.md", "east-c.md", "east-d.md"),
		ConsistOf("north-a.md", "north-b.md", "north-c.md", "north-d.md"),
	))
}

// TestDeriveVocabClusters_PreviousKMatchingOptimumIsStable verifies the
// end-to-end path: when the previous K equals the free optimum the
// derivation is unchanged by hysteresis.
func TestDeriveVocabClusters_PreviousKMatchingOptimumIsStable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := twoBlobNotes()

	fresh, err := cli.ExportDeriveVocabClusters(notes, noPreviousK, testDeriveSeed)
	g.Expect(err).NotTo(HaveOccurred())

	withPrev, err := cli.ExportDeriveVocabClusters(notes, fresh.K, testDeriveSeed)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(withPrev).To(Equal(fresh))
}

// TestDeriveVocabClusters_TinyVaultYieldsNoDerivation verifies the tiny-vault
// floor: below the minimum note count for meaningful clustering the
// derivation reports K=0 and no clusters (caller keeps the existing vocab).
func TestDeriveVocabClusters_TinyVaultYieldsNoDerivation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []cli.ExportNoteVector{{Name: "only.md", Vector: []float32{1, 0}}}

	derivation, err := cli.ExportDeriveVocabClusters(notes, noPreviousK, testDeriveSeed)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(derivation.K).To(BeZero())
	g.Expect(derivation.Clusters).To(BeEmpty())
}

// TestDeriveVocabClusters_WeakPreviousKDoesNotOverride verifies the
// end-to-end path with a previous K that clusters the data much worse than
// the free optimum: the free K wins.
func TestDeriveVocabClusters_WeakPreviousKDoesNotOverride(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := twoBlobNotes()
	weakPreviousK := 4 // splitting two tight blobs four ways scores far worse

	derivation, err := cli.ExportDeriveVocabClusters(notes, weakPreviousK, testDeriveSeed)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(derivation.K).To(Equal(2))
}

// TestMatchClustersToTerms_BelowThresholdSplitsBothWays verifies a cluster
// below the threshold against every term becomes a new cluster, and the
// unclaimed derived term is reported for retirement.
func TestMatchClustersToTerms_BelowThresholdSplitsBothWays(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroids := [][]float32{{0, 1}} // orthogonal to "east"
	existing := []cli.ExportExistingVocabTerm{
		{Name: "east", Vector: []float32{1, 0}, Origin: cli.ExportVocabOriginDerived},
	}

	result := cli.ExportMatchClustersToTerms(centroids, existing, cli.ExportVocabNameMatchThreshold)

	g.Expect(result.Matched).To(BeEmpty())
	g.Expect(result.NewClusters).To(Equal([]int{0}))
	g.Expect(result.RetiredTerms).To(Equal([]string{"east"}))
}

// TestMatchClustersToTerms_GreedyHighestSimilarityWins verifies that when two
// clusters both exceed the threshold against one term, the higher-similarity
// cluster claims the name and the other is reported as a new cluster.
func TestMatchClustersToTerms_GreedyHighestSimilarityWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroids := [][]float32{
		{1, 0.30}, // similar to "east", but less than the next cluster
		{1, 0.01}, // nearly identical to "east"
	}
	existing := []cli.ExportExistingVocabTerm{
		{Name: "east", Vector: []float32{1, 0}, Origin: cli.ExportVocabOriginDerived},
	}

	result := cli.ExportMatchClustersToTerms(centroids, existing, cli.ExportVocabNameMatchThreshold)

	g.Expect(result.Matched).To(HaveLen(1))
	g.Expect(result.Matched[0].ClusterIndex).To(Equal(1))
	g.Expect(result.Matched[0].Term).To(Equal("east"))
	g.Expect(result.NewClusters).To(Equal([]int{0}))
	g.Expect(result.RetiredTerms).To(BeEmpty())
}

// ── Task 1.2: greedy centroid-cosine name matching ───────────────────────────

// TestMatchClustersToTerms_KeepsNameAboveThreshold verifies a cluster whose
// centroid is nearly identical to an existing term's vector keeps that
// term's name, with no new clusters and no retirements.
func TestMatchClustersToTerms_KeepsNameAboveThreshold(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroids := [][]float32{{1, 0}}
	existing := []cli.ExportExistingVocabTerm{
		{Name: "east", Vector: []float32{1, 0.01}, Origin: cli.ExportVocabOriginDerived},
	}

	result := cli.ExportMatchClustersToTerms(centroids, existing, cli.ExportVocabNameMatchThreshold)

	g.Expect(result.Matched).To(HaveLen(1))
	g.Expect(result.Matched[0].ClusterIndex).To(Equal(0))
	g.Expect(result.Matched[0].Term).To(Equal("east"))
	g.Expect(result.NewClusters).To(BeEmpty())
	g.Expect(result.RetiredTerms).To(BeEmpty())
}

// TestMatchClustersToTerms_PartitionInvariants is a property test: every
// cluster lands in exactly one of Matched/NewClusters, each term is claimed
// at most once, matched similarities meet the threshold, and no proposed
// term is ever retired.
func TestMatchClustersToTerms_PartitionInvariants(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		centroids := rapid.SliceOfN(unitVec2Gen(), 0, maxMatchEntities).Draw(rt, "centroids")
		termVecs := rapid.SliceOfN(unitVec2Gen(), 0, maxMatchEntities).Draw(rt, "termVecs")

		existing := make([]cli.ExportExistingVocabTerm, 0, len(termVecs))

		for i, vec := range termVecs {
			origin := cli.ExportVocabOriginDerived
			if rapid.Bool().Draw(rt, "proposed") {
				origin = cli.ExportVocabOriginProposed
			}

			name := "term-" + string(rune('a'+i))
			existing = append(existing, cli.ExportExistingVocabTerm{Name: name, Vector: vec, Origin: origin})
		}

		result := cli.ExportMatchClustersToTerms(centroids, existing, cli.ExportVocabNameMatchThreshold)

		clusterSeen := make(map[int]int, len(centroids))
		termClaimed := make(map[string]int, len(existing))

		for _, match := range result.Matched {
			clusterSeen[match.ClusterIndex]++
			termClaimed[match.Term]++
			g.Expect(match.Similarity).To(BeNumerically(">=", cli.ExportVocabNameMatchThreshold))
		}

		for _, clusterIdx := range result.NewClusters {
			clusterSeen[clusterIdx]++
		}

		g.Expect(clusterSeen).To(HaveLen(len(centroids)), "every cluster in exactly one bucket")

		for clusterIdx, count := range clusterSeen {
			g.Expect(count).To(Equal(1), "cluster %d appears %d times", clusterIdx, count)
		}

		for term, count := range termClaimed {
			g.Expect(count).To(Equal(1), "term %s claimed %d times", term, count)
		}

		proposed := make(map[string]bool, len(existing))
		for _, term := range existing {
			proposed[term.Name] = term.Origin == cli.ExportVocabOriginProposed
		}

		for _, retired := range result.RetiredTerms {
			g.Expect(proposed[retired]).To(BeFalse(), "proposed term %s retired", retired)
			g.Expect(termClaimed).NotTo(HaveKey(retired), "matched term %s retired", retired)
		}
	})
}

// TestMatchClustersToTerms_ProposedTermsNeverRetired verifies that an
// unclaimed term with origin "proposed" is excluded from the retirement
// output — proposed terms exist precisely because clustering cannot see them.
func TestMatchClustersToTerms_ProposedTermsNeverRetired(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroids := [][]float32{{0, 1}}
	existing := []cli.ExportExistingVocabTerm{
		{Name: "scattered", Vector: []float32{1, 0}, Origin: cli.ExportVocabOriginProposed},
		{Name: "stale", Vector: []float32{0.7, -0.7}, Origin: cli.ExportVocabOriginDerived},
	}

	result := cli.ExportMatchClustersToTerms(centroids, existing, cli.ExportVocabNameMatchThreshold)

	g.Expect(result.RetiredTerms).To(Equal([]string{"stale"}))
}

// unexported constants.
const (
	// blobJitter is the per-note perturbation applied within a blob — small
	// enough that blobs stay well separated.
	blobJitter = float32(0.05)
	// centroidTolerance bounds float32 accumulation error in mean checks.
	centroidTolerance = 1e-4
	// degenerateCorpusSize is enough identical vectors to attempt clustering
	// (well above minK) while guaranteeing silhouette 0.
	degenerateCorpusSize = 10
	// diffuseClusterBias is the weak per-cluster signal added on top of the
	// pseudo-noise — tuned so the fixture's peak silhouette lands ~0.074,
	// inside the real vault's measured operating band.
	diffuseClusterBias = float32(0.25)
	// diffuseCorpusDims spreads the noise over enough dimensions that cosine
	// similarities stay diffuse, mimicking a whole-vault embedding cloud.
	diffuseCorpusDims = 32
	diffuseCorpusSize = 120
	maxBlobSize       = 8
	// maxMatchEntities bounds generated cluster/term counts in the matching
	// property test; names stay within a single alphabetic suffix.
	maxMatchEntities = 6
	minBlobSize      = 3
	// noPreviousK signals a first derivation with no prior K to prefer.
	noPreviousK = 0
	// noiseHashScaleA/B/C parameterize the deterministic sin-hash noise
	// (classic frac(sin(n)*large) generator) used instead of math/rand.
	noiseHashScaleA = 12.9898
	noiseHashScaleB = 78.233
	noiseHashScaleC = 43758.5453
	// testDeriveSeed keeps k-means deterministic across test runs.
	testDeriveSeed = uint64(42)
)

// blobNotes builds count notes clustered around center, each offset by a
// deterministic multiple of jitter in the first dimension.
func blobNotes(prefix string, count int, center []float32) []cli.ExportNoteVector {
	notes := make([]cli.ExportNoteVector, 0, count)

	for i := range count {
		vector := make([]float32, len(center))
		copy(vector, center)
		vector[0] += blobJitter * float32(i%3)

		name := prefix + "-" + string(rune('a'+i)) + ".md"
		notes = append(notes, cli.ExportNoteVector{Name: name, Vector: vector})
	}

	return notes
}

// diffuseCorpusNotes builds a deterministic diffuse-but-structured corpus:
// unit pseudo-noise vectors with a weak two-cluster bias, whose peak
// silhouette sits in the sub-0.10 band real vault embedding clouds occupy.
func diffuseCorpusNotes() []cli.ExportNoteVector {
	notes := make([]cli.ExportNoteVector, 0, diffuseCorpusSize)

	for i := range diffuseCorpusSize {
		vector := make([]float32, diffuseCorpusDims)

		var norm float64

		for dim := range diffuseCorpusDims {
			noise := pseudoNoise(float64(i)*noiseHashScaleA + float64(dim)*noiseHashScaleB)
			vector[dim] = float32(noise)
			norm += noise * noise
		}

		norm = math.Sqrt(norm)
		for dim := range vector {
			vector[dim] /= float32(norm)
		}

		vector[i%2] += diffuseClusterBias

		name := "diffuse-" + strconv.Itoa(i) + ".md"
		notes = append(notes, cli.ExportNoteVector{Name: name, Vector: vector})
	}

	return notes
}

// meanOfNamed returns the arithmetic mean of the named vectors.
func meanOfNamed(vectorsByName map[string][]float32, names []string) []float32 {
	dims := len(vectorsByName[names[0]])
	sums := make([]float64, dims)

	for _, name := range names {
		for dim, value := range vectorsByName[name] {
			sums[dim] += float64(value)
		}
	}

	mean := make([]float32, dims)
	for dim := range dims {
		mean[dim] = float32(sums[dim] / float64(len(names)))
	}

	return mean
}

// pseudoNoise maps n to a deterministic pseudo-random value in [-1, 1) via
// the classic frac(sin(n)*large) hash — reproducible without math/rand.
func pseudoNoise(n float64) float64 {
	scaled := math.Sin(n) * noiseHashScaleC

	return 2*(scaled-math.Floor(scaled)) - 1
}

// twoBlobNotes returns eight notes in two tight, well-separated groups.
func twoBlobNotes() []cli.ExportNoteVector {
	notes := blobNotes("east", 4, []float32{1, 0})

	return append(notes, blobNotes("north", 4, []float32{0, 1})...)
}

// unitVec2Gen generates 2D vectors on (roughly) the unit circle so cosine
// similarities span the full [-1, 1] range.
func unitVec2Gen() *rapid.Generator[[]float32] {
	return rapid.Custom(func(rt *rapid.T) []float32 {
		angle := rapid.Float64Range(0, 2*math.Pi).Draw(rt, "angle")

		return []float32{float32(math.Cos(angle)), float32(math.Sin(angle))}
	})
}
