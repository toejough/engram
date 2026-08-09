// Package recencycentroid652 is a throwaway eval harness for issue #652:
// does recency-weighting the within-cluster centroid used to rank
// candidate_l2s reduce superseded-note over-surfacing vs. the current
// unweighted vector-mean centroid, without materially degrading relevance?
//
// Scope: real vault data (`~/.local/share/engram/vault`), real `supersedes:`
// frontmatter pairs as ground truth, real cluster.AutoK clustering (same
// package the production query pipeline uses), only clusters with >5 note
// members (the only case where nomination can change per candidateNoteK=5).
//
// Not part of the production build; kept as a citable reference for the
// #652 eval result recorded in dev/eval/LEDGER.md. Skipped by default since
// it reads a machine-local vault path — run explicitly via
// `go test ./dev/eval/recency-centroid-652/ -run TestRecencyCentroidEval -v`.
package recencycentroid652_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/toejough/engram/internal/cluster"
	"github.com/toejough/engram/internal/embed"
)

func TestRecencyCentroidEval(t *testing.T) { //nolint:gocognit,cyclop // throwaway eval harness, kept as citable record
	t.Parallel()
	t.Skip("throwaway #652 eval harness; kept as citable record — run explicitly to reproduce")

	notes, err := loadVault(vaultDir)
	if err != nil {
		t.Fatalf("load vault: %v", err)
	}

	byPath := make(map[string]note, len(notes))
	for _, n := range notes {
		byPath[n.path] = n
	}

	now := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)

	type pairResult struct {
		newPath, oldPath                       string
		clusterSize                            int
		unweightedOldRank, unweightedNewRank   int
		weightedOldRank, weightedNewRank       int
		unweightedOldInTop, weightedOldInTop   bool
		unweightedNewInTop, weightedNewInTop   bool
		unweightedRelevance, weightedRelevance float32 // mean query-cosine of top5 picks
	}

	var results []pairResult

	tested, skippedNoOld, skippedSmallCluster := 0, 0, 0

	for _, n := range notes {
		for _, target := range n.supersedes {
			old, ok := byPath[target]
			if !ok {
				skippedNoOld++
				continue
			}

			tested++

			// Proxy query vector: the newer note's own situation vector represents
			// "what a user would search for on this topic" (the topic that caused
			// the supersession).
			queryVec := n.sit

			pool := topMatchPool(queryVec, notes, matchPoolSize)

			// Ensure both notes are present in the pool (guarantee the test case is
			// evaluable even if one fell just outside the raw top-N).
			pool = ensurePresent(pool, old)
			pool = ensurePresent(pool, n)

			vectors := make([][]float32, len(pool))
			for i, p := range pool {
				vectors[i] = bestAxis(queryVec, p)
			}

			seed := seedFromString(n.path)

			autoK, akErr := cluster.AutoK(vectors, clusterMinK, clusterMaxK, silhouetteFloor, seed)
			if akErr != nil || autoK.K == 0 {
				continue
			}

			// Locate the cluster containing the NEW note (supersession target's cluster).
			newIdx, oldIdx := -1, -1

			for i, p := range pool {
				if p.path == n.path {
					newIdx = i
				}

				if p.path == old.path {
					oldIdx = i
				}
			}

			if newIdx == -1 || oldIdx == -1 {
				continue
			}

			targetCluster := autoK.Assignments[newIdx]
			if autoK.Assignments[oldIdx] != targetCluster {
				// old and new landed in different clusters — nomination within the
				// new note's cluster can't surface the old one; not an over-surfacing
				// test case for THIS mechanism.
				continue
			}

			var (
				members    []note
				memberVecs [][]float32
			)

			for i, p := range pool {
				if autoK.Assignments[i] == targetCluster {
					members = append(members, p)
					memberVecs = append(memberVecs, vectors[i])
				}
			}

			if len(members) <= candidateNoteK {
				skippedSmallCluster++
				continue
			}

			unweightedCentroid := meanVector(memberVecs)
			weightedCentroid := recencyWeightedCentroid(members, memberVecs, now)

			uRanked := rankByCentroid(unweightedCentroid, members)
			wRanked := rankByCentroid(weightedCentroid, members)

			pairRes := pairResult{
				newPath:     n.path,
				oldPath:     old.path,
				clusterSize: len(members),
			}

			pairRes.unweightedOldRank, pairRes.unweightedOldInTop = rankOf(uRanked, old.path)
			pairRes.unweightedNewRank, pairRes.unweightedNewInTop = rankOf(uRanked, n.path)
			pairRes.weightedOldRank, pairRes.weightedOldInTop = rankOf(wRanked, old.path)
			pairRes.weightedNewRank, pairRes.weightedNewInTop = rankOf(wRanked, n.path)

			pairRes.unweightedRelevance = meanQueryCosine(uRanked[:min(candidateNoteK, len(uRanked))], queryVec)
			pairRes.weightedRelevance = meanQueryCosine(wRanked[:min(candidateNoteK, len(wRanked))], queryVec)

			results = append(results, pairRes)
		}
	}

	fmt.Printf("supersedes: pairs found=%d, tested=%d, no-old-target=%d, cluster<=5=%d, eligible(same-cluster,>5)=%d\n",
		tested+skippedNoOld, tested, skippedNoOld, skippedSmallCluster,
		len(results))
	fmt.Println()

	if len(results) == 0 {
		fmt.Println("NO eligible test cases (same-cluster supersession pairs in a >5-member cluster).")
		fmt.Println("Cannot run the eval on real data.")

		return
	}

	oldSurfacedUnweighted, oldSurfacedWeighted := 0, 0

	var sumURel, sumWRel float32

	fmt.Println("path pairs (new supersedes old):")

	for _, r := range results {
		fmt.Printf("  cluster_size=%d  new=%s  old=%s\n", r.clusterSize, r.newPath, r.oldPath)
		fmt.Printf("    unweighted: old_rank=%d old_in_top5=%v  new_rank=%d new_in_top5=%v  top5_relevance=%.4f\n",
			r.unweightedOldRank, r.unweightedOldInTop, r.unweightedNewRank, r.unweightedNewInTop, r.unweightedRelevance)
		fmt.Printf("    weighted:   old_rank=%d old_in_top5=%v  new_rank=%d new_in_top5=%v  top5_relevance=%.4f\n",
			r.weightedOldRank, r.weightedOldInTop, r.weightedNewRank, r.weightedNewInTop, r.weightedRelevance)

		if r.unweightedOldInTop {
			oldSurfacedUnweighted++
		}

		if r.weightedOldInTop {
			oldSurfacedWeighted++
		}

		sumURel += r.unweightedRelevance
		sumWRel += r.weightedRelevance
	}

	n := float32(len(results))

	fmt.Println()
	fmt.Println("=== SUMMARY ===")
	fmt.Printf("n eligible cases: %d\n", len(results))
	fmt.Printf("unweighted: superseded-note over-surfacing rate = %d/%d = %.1f%%  |  mean top5 relevance = %.4f\n",
		oldSurfacedUnweighted, len(results), 100*float32(oldSurfacedUnweighted)/n, sumURel/n)
	fmt.Printf("weighted:   superseded-note over-surfacing rate = %d/%d = %.1f%%  |  mean top5 relevance = %.4f\n",
		oldSurfacedWeighted, len(results), 100*float32(oldSurfacedWeighted)/n, sumWRel/n)
}

// unexported constants.
const (
	candidateNoteK  = 5
	clusterMaxK     = 7
	clusterMinK     = 2
	halfLifeDays    = 60.0
	matchPoolSize   = 30
	noteDateFormat  = "2006-01-02"
	silhouetteFloor = 0.10
	tailWeight      = 0.2
	vaultDir        = "/Users/joe/.local/share/engram/vault"
)

// unexported variables.
var (
	supersedesLinkRE = regexp.MustCompile(`\[\[([^\]|]+)`)
)

type note struct {
	path       string // basename without extension
	created    string
	supersedes []string // basenames (no ext) of older notes this one supersedes
	sit        []float32
	body       []float32
}

type sidecarDoc struct {
	SituationVector []float32 `json:"situation_vector"` //nolint:tagliatelle // mirrors production sidecar field naming
	BodyVector      []float32 `json:"body_vector"`      //nolint:tagliatelle // mirrors production sidecar field naming
}

func ageDays(created string, now time.Time) float64 {
	if created == "" {
		return 0
	}

	t, err := time.Parse(noteDateFormat, created)
	if err != nil {
		return 0
	}

	d := now.Sub(t).Hours() / 24
	if d < 0 {
		d = 0
	}

	return d
}

func bestAxis(queryVec []float32, n note) []float32 {
	sitSim := embed.Cosine(queryVec, n.sit)

	bodySim := embed.Cosine(queryVec, n.body)
	if bodySim > sitSim {
		return n.body
	}

	return n.sit
}

func ensurePresent(pool []note, n note) []note {
	for _, p := range pool {
		if p.path == n.path {
			return pool
		}
	}

	return append(pool, n)
}

// extractBasename normalizes a wikilink target to the vault's basename
// convention (some links carry a trailing ".md", most don't).
func extractBasename(link string) string {
	return strings.TrimSuffix(strings.TrimSpace(link), ".md")
}

func loadVault(dir string) ([]note, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var notes []note

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		base := strings.TrimSuffix(name, ".md")
		vecPath := filepath.Join(dir, base+".vec.json")

		raw, err := os.ReadFile(vecPath)
		if err != nil {
			continue // no sidecar, skip
		}

		var doc sidecarDoc
		if err := json.Unmarshal(raw, &doc); err != nil {
			continue
		}

		if len(doc.SituationVector) == 0 || len(doc.BodyVector) == 0 {
			continue
		}

		mdRaw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}

		created, supersedes := parseFrontmatter(string(mdRaw))

		notes = append(notes, note{
			path:       base,
			created:    created,
			supersedes: supersedes,
			sit:        doc.SituationVector,
			body:       doc.BodyVector,
		})
	}

	return notes, nil
}

func meanQueryCosine(top []note, queryVec []float32) float32 {
	if len(top) == 0 {
		return 0
	}

	var sum float32
	for _, t := range top {
		sum += embed.Cosine(queryVec, bestAxis(queryVec, t))
	}

	return sum / float32(len(top))
}

func meanVector(vectors [][]float32) []float32 {
	dims := len(vectors[0])

	sums := make([]float64, dims)
	for _, vec := range vectors {
		for d := range dims {
			sums[d] += float64(vec[d])
		}
	}

	mean := make([]float32, dims)
	for d := range dims {
		mean[d] = float32(sums[d] / float64(len(vectors)))
	}

	return mean
}

// parseFrontmatter extracts `created:` and every target of a body-level
// `Supersedes: [[path]] ...` line (a note may supersede more than one older
// note; each line is collected).
func parseFrontmatter(raw string) (created string, supersedes []string) {
	lines := strings.Split(raw, "\n")
	fences := 0
	inBody := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			fences++
			if fences == 2 {
				inBody = true
			}

			continue
		}

		if fences == 1 {
			if rest, ok := strings.CutPrefix(trimmed, "created:"); ok {
				created = strings.Trim(strings.TrimSpace(rest), `"`)
			}
		}

		if inBody && strings.HasPrefix(trimmed, "Supersedes:") {
			if m := supersedesLinkRE.FindStringSubmatch(trimmed); m != nil {
				supersedes = append(supersedes, extractBasename(m[1]))
			}
		}
	}

	return created, supersedes
}

func rankByCentroid(centroid []float32, members []note) []note {
	type scored struct {
		n   note
		sim float32
	}

	scoredList := make([]scored, len(members))
	for i, m := range members {
		sim := embed.Cosine(centroid, m.sit)
		if bodySim := embed.Cosine(centroid, m.body); bodySim > sim {
			sim = bodySim
		}

		scoredList[i] = scored{n: m, sim: sim}
	}

	sort.SliceStable(scoredList, func(i, j int) bool {
		if scoredList[i].sim != scoredList[j].sim {
			return scoredList[i].sim > scoredList[j].sim
		}

		return scoredList[i].n.path < scoredList[j].n.path
	})

	out := make([]note, len(scoredList))
	for i, s := range scoredList {
		out[i] = s.n
	}

	return out
}

func rankOf(ranked []note, path string) (int, bool) {
	for i, r := range ranked {
		if r.path == path {
			return i + 1, i < candidateNoteK
		}
	}

	return -1, false
}

func recencyMultiplier(ageDays float64) float64 {
	decay := math.Exp2(-ageDays / halfLifeDays)
	return decay * (1 + tailWeight*0)
}

// recencyWeightedCentroid computes Σ(recency_i · vec_i) / Σ recency_i using
// the SAME decay formula as production's recencyMultiplier (half-life 60d,
// tailWeight 0.2, turnFrac=0 since these are notes not chunk turns):
// recency_i = exp2(-ageDays_i/60).
func recencyWeightedCentroid(members []note, vectors [][]float32, now time.Time) []float32 {
	dims := len(vectors[0])
	sums := make([]float64, dims)

	var weightSum float64

	for i, m := range members {
		age := ageDays(m.created, now)
		w := recencyMultiplier(age)

		weightSum += w
		for d := range dims {
			sums[d] += w * float64(vectors[i][d])
		}
	}

	if weightSum == 0 {
		return meanVector(vectors)
	}

	out := make([]float32, dims)
	for d := range dims {
		out[d] = float32(sums[d] / weightSum)
	}

	return out
}

func seedFromString(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := range len(s) {
		h ^= uint64(s[i])
		h *= 1099511628211
	}

	return h
}

func topMatchPool(queryVec []float32, all []note, n int) []note {
	type scored struct {
		n   note
		sim float32
	}

	scoredList := make([]scored, len(all))
	for i, nt := range all {
		sim := embed.Cosine(queryVec, bestAxis(queryVec, nt))
		scoredList[i] = scored{n: nt, sim: sim}
	}

	sort.SliceStable(scoredList, func(i, j int) bool {
		return scoredList[i].sim > scoredList[j].sim
	})

	if n > len(scoredList) {
		n = len(scoredList)
	}

	out := make([]note, n)
	for i := range n {
		out[i] = scoredList[i].n
	}

	return out
}
