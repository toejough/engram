package cli

import (
	"math"
	"sort"

	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	// exploreMatchEvidenceBonus is the flat additive similarity bonus
	// applied to a term's cosine before softmax when >=1 of its members
	// appears in the exploit half (design.md Decision 5). Calibrated
	// alongside exploreTemperatureDefault via task 3.1.
	exploreMatchEvidenceBonus = float32(0.05)
	// exploreTemperatureDefault is the softmax temperature for explore-half
	// centroid allocation. Placeholder pending empirical calibration
	// (design.md Decision 1, task 3.1): run the representative query set
	// against the production vault and solve
	// exp(sim_top/tau)/exp(sim_med/tau) = 10 for tau.
	exploreTemperatureDefault = float32(0.1)
)

// exploreBackfillState is dedupeAndBackfill's working state: which note
// paths are already selected, each term's read cursor into its ranked
// candidate list, and the picks accumulated so far.
type exploreBackfillState struct {
	ranked   map[string][]explorePick
	selected map[string]struct{}
	exploit  map[string]struct{}
	cursor   map[string]int
	picks    []explorePick
}

// fillTerm appends up to want additional picks from term's ranked candidate
// list, skipping candidates already selected or present in the exploit half.
func (s *exploreBackfillState) fillTerm(term string, want int) {
	for range want {
		pick, ok := s.takeNext(term)
		if !ok {
			return
		}

		s.picks = append(s.picks, pick)
		s.selected[pick.Path] = struct{}{}
	}
}

// takeNext returns the next not-yet-selected, non-exploit-duplicate
// candidate from term's ranked list, advancing the cursor past any
// duplicates it skips.
func (s *exploreBackfillState) takeNext(term string) (explorePick, bool) {
	ranked := s.ranked[term]
	for s.cursor[term] < len(ranked) {
		candidate := ranked[s.cursor[term]]
		s.cursor[term]++

		if _, dupExploit := s.exploit[candidate.Path]; dupExploit {
			continue
		}

		if _, dupSelected := s.selected[candidate.Path]; dupSelected {
			continue
		}

		return candidate, true
	}

	return explorePick{}, false
}

// exploreCandidate is a note eligible for explore-half selection under some
// vocab term: its path and embedding vector. Callers pass membership already
// filtered to exclude definition notes (isVocabDefinitionNote).
type exploreCandidate struct {
	Path   string
	Vector []float32
}

// explorePick is a single explore-half selection: the note path, the vocab
// term it was sampled under, and its cosine similarity to that term's
// centroid.
type explorePick struct {
	Path       string
	SourceTerm string
	Cosine     float32
}

// applyMatchEvidenceBonus returns similarities with bonus added once (flat,
// non-stacking) to every term present in evidenced.
func applyMatchEvidenceBonus(
	similarities map[string]float32,
	evidenced map[string]bool,
	bonus float32,
) map[string]float32 {
	boosted := make(map[string]float32, len(similarities))

	for term, sim := range similarities {
		if evidenced[term] {
			boosted[term] = sim + bonus
			continue
		}

		boosted[term] = sim
	}

	return boosted
}

// dedupeAndBackfill drops explore picks that duplicate an exploit-half note
// or another cluster's selection, then refills freed slots following order
// (the softmax allocation priority order) until each term's allocation is
// met or its ranked candidate list (rankedByTerm, expected to hold every
// member ranked, not just the initially allocated count) is exhausted.
//
// Phase 1 fills each term's own allocation from its own ranked list,
// skipping duplicates. Phase 2 sweeps order again for any remaining deficit,
// pulling from whichever term still has unused ranked candidates -
// "next-highest remaining cluster weight" per design.md Decision 4.
func dedupeAndBackfill(
	allocated map[string]int,
	order []string,
	rankedByTerm map[string][]explorePick,
	exploitPaths map[string]struct{},
) []explorePick {
	budget := 0
	for _, count := range allocated {
		budget += count
	}

	state := &exploreBackfillState{
		selected: make(map[string]struct{}, budget),
		cursor:   make(map[string]int, len(order)),
		ranked:   rankedByTerm,
		exploit:  exploitPaths,
		picks:    make([]explorePick, 0, budget),
	}

	// Phase 1: each term fills its own allocation from its own ranked list.
	for _, term := range order {
		state.fillTerm(term, allocated[term])
	}

	// Phase 2: sweep order for any remaining deficit, pulling from whichever
	// term still has unused ranked candidates ("next-highest remaining
	// cluster weight" per design.md Decision 4).
	for len(state.picks) < budget {
		before := len(state.picks)

		for _, term := range order {
			if len(state.picks) >= budget {
				break
			}

			state.fillTerm(term, 1)
		}

		if len(state.picks) == before {
			break
		}
	}

	return state.picks
}

// exploreAllocationOrder returns terms sorted by descending similarity (ties
// broken by term name), the priority order softmaxAllocate uses for leftover
// distribution and dedupeAndBackfill reuses for refill priority (design.md
// Decision 4: "next-highest remaining cluster weight").
func exploreAllocationOrder(similarities map[string]float32) []string {
	terms := make([]string, 0, len(similarities))
	for term := range similarities {
		terms = append(terms, term)
	}

	sort.Slice(terms, func(i, j int) bool {
		if similarities[terms[i]] != similarities[terms[j]] {
			return similarities[terms[i]] > similarities[terms[j]]
		}

		return terms[i] < terms[j]
	})

	return terms
}

// selectWithinCluster ranks a term's member candidates by descending cosine
// similarity to the term centroid (ties broken by path for determinism) and
// returns the top sampleCount. A non-positive sampleCount or an empty member
// list yields an empty (non-nil) slice.
func selectWithinCluster(
	term string,
	centroid []float32,
	members []exploreCandidate,
	sampleCount int,
) []explorePick {
	if sampleCount <= 0 || len(members) == 0 {
		return []explorePick{}
	}

	ranked := make([]explorePick, len(members))
	for i, member := range members {
		ranked[i] = explorePick{
			Path:       member.Path,
			SourceTerm: term,
			Cosine:     embed.Cosine(centroid, member.Vector),
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Cosine != ranked[j].Cosine {
			return ranked[i].Cosine > ranked[j].Cosine
		}

		return ranked[i].Path < ranked[j].Path
	})

	if sampleCount > len(ranked) {
		sampleCount = len(ranked)
	}

	return ranked[:sampleCount]
}

// softmaxAllocate distributes budget integer samples across terms
// proportional to softmax(similarities[term]/temperature). Totals sum
// exactly to budget via largest-remainder apportionment: each term's exact
// fractional share is floored, then the leftover units are handed to the
// terms with the largest fractional remainders (ties broken by descending
// similarity, then term name, for determinism). A non-positive budget or an
// empty term set yields an empty allocation (the B=0 skip case).
func softmaxAllocate(similarities map[string]float32, temperature float32, budget int) map[string]int {
	allocated := make(map[string]int, len(similarities))
	if budget <= 0 || len(similarities) == 0 {
		return allocated
	}

	terms := exploreAllocationOrder(similarities)

	weights := make(map[string]float64, len(terms))

	var totalWeight float64

	for _, term := range terms {
		weight := math.Exp(float64(similarities[term]) / float64(temperature))
		weights[term] = weight
		totalWeight += weight
	}

	type share struct {
		term      string
		remainder float64
	}

	shares := make([]share, 0, len(terms))
	assigned := 0

	for _, term := range terms {
		exact := weights[term] / totalWeight * float64(budget)
		floorPart := int(math.Floor(exact))
		allocated[term] = floorPart
		assigned += floorPart

		shares = append(shares, share{term: term, remainder: exact - float64(floorPart)})
	}

	sort.SliceStable(shares, func(i, j int) bool {
		if shares[i].remainder != shares[j].remainder {
			return shares[i].remainder > shares[j].remainder
		}

		if similarities[shares[i].term] != similarities[shares[j].term] {
			return similarities[shares[i].term] > similarities[shares[j].term]
		}

		return shares[i].term < shares[j].term
	})

	remaining := budget - assigned
	for i := 0; i < remaining && i < len(shares); i++ {
		allocated[shares[i].term]++
	}

	return allocated
}

// termsWithExploitEvidence reports, for each term in members, whether at
// least one of its member notes' paths appears in exploitPaths (the
// match-evidence condition for applyMatchEvidenceBonus). Terms with no
// exploit-half evidence are omitted rather than mapped to false.
func termsWithExploitEvidence(
	members map[string][]exploreCandidate,
	exploitPaths map[string]struct{},
) map[string]bool {
	evidenced := make(map[string]bool, len(members))

	for term, candidates := range members {
		for _, candidate := range candidates {
			if _, ok := exploitPaths[candidate.Path]; ok {
				evidenced[term] = true
				break
			}
		}
	}

	return evidenced
}
