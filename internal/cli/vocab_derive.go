package cli

import (
	"fmt"
	"sort"

	"github.com/toejough/engram/internal/cluster"
	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	// vocabDeriveMaxK caps the derived term count. Vocab-scale, deliberately
	// larger than recall's clusterMaxK: a vault's vocabulary can meaningfully
	// hold tens of terms, but bounding K keeps derivation from re-approaching
	// the observed ~150-term additive drift.
	vocabDeriveMaxK = 40
	// vocabDeriveMinK is the minimum-K floor (design open question, decided):
	// 2 — the smallest K the silhouette machinery can score. Tiny vaults are
	// handled by AutoK itself: with fewer points than vocabDeriveMinK, or no
	// structure above vocabDeriveSilhouetteFloor, derivation returns K=0 and
	// the caller keeps the existing vocabulary rather than forcing a floor of
	// artificial clusters onto too little data.
	vocabDeriveMinK = cluster.MinClustersK
	// vocabDeriveSilhouetteFloor is the minimum silhouette for a derivation
	// to count as structure, matching recall's clusterSilhouetteFloor.
	vocabDeriveSilhouetteFloor = 0.10
	// vocabNameMatchThreshold is the minimum centroid cosine similarity for a
	// derived cluster to claim an existing term's name (design decision 2,
	// starting value 0.80 — tune empirically on the real vaults).
	vocabNameMatchThreshold = float32(0.80)
	// vocabOriginDerived marks a term produced (or absorbed) by derivation.
	vocabOriginDerived = "derived"
	// vocabOriginProposed marks a term minted via `vocab propose`; proposed
	// terms are never retired by derivation.
	vocabOriginProposed = "proposed"
	// vocabSilhouetteEpsilon is the hysteresis band: when re-clustering at
	// the previous K scores within this much of the free silhouette optimum,
	// the previous K is kept — term-name stability beats a marginal
	// silhouette gain (design risk: silhouette-K instability on near-tie K).
	vocabSilhouetteEpsilon = float32(0.02)
)

// derivedCluster is one derivation-produced cluster: the note names assigned
// to it and its centroid (mean of the members' vectors).
type derivedCluster struct {
	Members  []string
	Centroid []float32
}

// existingVocabTerm is an existing term entering name matching: its name,
// current centroid (or description) vector, and provenance origin.
type existingVocabTerm struct {
	Name   string
	Vector []float32
	Origin string
}

// noteVector pairs a non-definition note's identity with its body vector.
// Inputs are DI'd data — callers load sidecars; derivation is pure logic.
type noteVector struct {
	Name   string
	Vector []float32
}

// vocabDerivation is the result of clustering the vault's non-definition
// note vectors. K == 0 means no meaningful structure was found (tiny vault
// or silhouette below floor); Clusters is nil in that case and the caller
// keeps the existing vocabulary.
type vocabDerivation struct {
	K          int
	Silhouette float64
	Clusters   []derivedCluster
}

// vocabMatchResult partitions a derivation against the existing vocabulary:
// Matched clusters keep their term's name; NewClusters (cluster indices,
// ascending) need LLM naming; RetiredTerms are unclaimed derived-origin
// terms (proposed terms are never listed).
type vocabMatchResult struct {
	Matched      []vocabNameMatch
	NewClusters  []int
	RetiredTerms []string
}

// vocabNameMatch pairs a derived cluster with the existing term whose name
// it keeps.
type vocabNameMatch struct {
	ClusterIndex int
	Term         string
	Similarity   float32
}

// applyVocabKHysteresis chooses between the free silhouette optimum and the
// previous-K derivation: the pinned (previous-K) result wins when it has
// structure (K > 0) and its silhouette is within vocabSilhouetteEpsilon of
// the free optimum; otherwise the free result stands.
func applyVocabKHysteresis(free, pinned vocabDerivation) vocabDerivation {
	if pinned.K == 0 {
		return free
	}

	if free.Silhouette-pinned.Silhouette <= float64(vocabSilhouetteEpsilon) {
		return pinned
	}

	return free
}

// deriveAtKRange runs silhouette auto-K over [minK, maxK] and regroups the
// result into a vocabDerivation.
func deriveAtKRange(notes []noteVector, minK, maxK int, seed uint64) (vocabDerivation, error) {
	vectors := make([][]float32, len(notes))
	for i, note := range notes {
		vectors[i] = note.Vector
	}

	autoK, err := cluster.AutoK(vectors, minK, maxK, vocabDeriveSilhouetteFloor, seed)
	if err != nil {
		return vocabDerivation{}, fmt.Errorf("clustering note vectors: %w", err)
	}

	return groupDerivation(notes, autoK), nil
}

// deriveVocabClusters clusters the given notes' vectors with silhouette
// auto-K k-means (reusing internal/cluster) and regroups the flat
// assignments into per-cluster member lists with mean centroids. previousK
// is reserved for silhouette hysteresis (0 = no previous derivation).
func deriveVocabClusters(notes []noteVector, previousK int, seed uint64) (vocabDerivation, error) {
	free, err := deriveAtKRange(notes, vocabDeriveMinK, vocabDeriveMaxK, seed)
	if err != nil || previousK < vocabDeriveMinK || previousK == free.K {
		return free, err
	}

	pinned, err := deriveAtKRange(notes, previousK, previousK, seed)
	if err != nil {
		return vocabDerivation{}, err
	}

	return applyVocabKHysteresis(free, pinned), nil
}

// groupDerivation owns the note↔vector correlation: AutoK returns flat
// Assignments over bare vectors in input order, so notes[i] belongs to
// cluster Assignments[i].
func groupDerivation(notes []noteVector, autoK cluster.AutoKResult) vocabDerivation {
	if autoK.K == 0 {
		return vocabDerivation{}
	}

	clusters := make([]derivedCluster, autoK.K)
	for clusterIdx := range clusters {
		clusters[clusterIdx].Centroid = autoK.Centroids[clusterIdx]
	}

	for i, clusterIdx := range autoK.Assignments {
		clusters[clusterIdx].Members = append(clusters[clusterIdx].Members, notes[i].Name)
	}

	return vocabDerivation{K: autoK.K, Silhouette: autoK.Silhouette, Clusters: clusters}
}

// matchCandidates returns every (cluster, term) pair at or above threshold,
// sorted best-first: similarity descending, then term name ascending, then
// cluster index ascending for determinism.
func matchCandidates(
	centroids [][]float32,
	existing []existingVocabTerm,
	threshold float32,
) []vocabNameMatch {
	candidates := make([]vocabNameMatch, 0, len(centroids))

	for clusterIdx, centroid := range centroids {
		for _, term := range existing {
			similarity := embed.Cosine(centroid, term.Vector)
			if similarity < threshold {
				continue
			}

			candidates = append(candidates, vocabNameMatch{
				ClusterIndex: clusterIdx,
				Term:         term.Name,
				Similarity:   similarity,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.Similarity != right.Similarity {
			return left.Similarity > right.Similarity
		}

		if left.Term != right.Term {
			return left.Term < right.Term
		}

		return left.ClusterIndex < right.ClusterIndex
	})

	return candidates
}

// matchClustersToTerms greedily matches derived cluster centroids to
// existing terms by cosine similarity: candidate pairs at or above threshold
// are taken best-first, each cluster and each term claimed at most once.
// Greedy (not optimal assignment) is deliberate: auditable and stable at
// tens-of-terms scale (design decision 2).
func matchClustersToTerms(
	centroids [][]float32,
	existing []existingVocabTerm,
	threshold float32,
) vocabMatchResult {
	candidates := matchCandidates(centroids, existing, threshold)

	clusterClaimed := make(map[int]bool, len(centroids))
	termClaimed := make(map[string]bool, len(existing))

	var result vocabMatchResult

	for _, candidate := range candidates {
		if clusterClaimed[candidate.ClusterIndex] || termClaimed[candidate.Term] {
			continue
		}

		clusterClaimed[candidate.ClusterIndex] = true
		termClaimed[candidate.Term] = true
		result.Matched = append(result.Matched, candidate)
	}

	for clusterIdx := range centroids {
		if !clusterClaimed[clusterIdx] {
			result.NewClusters = append(result.NewClusters, clusterIdx)
		}
	}

	for _, term := range existing {
		if !termClaimed[term.Name] && term.Origin != vocabOriginProposed {
			result.RetiredTerms = append(result.RetiredTerms, term.Name)
		}
	}

	return result
}
