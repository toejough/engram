package cli

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/toejough/engram/internal/chunk"
)

// unexported constants.
const (
	defaultHalfLifeDays     = 3.0
	defaultRecencyFloor     = 3
	defaultRecentWindowDays = 1.0
	defaultTailWeight       = 0.2
	hoursPerDay             = 24
	turnAnchorPrefix        = "turn-"
)

// recencyParams are the tunable knobs (defaults chosen by the eval in recency_eval_test.go).
type recencyParams struct {
	halfLifeDays float64 // age at which the decay factor is 0.5
	tailWeight   float64 // extra lift for the last turn of a session (turnFrac=1)
	floor        int     // min recent chunk items the band guarantees
	windowDays   float64 // age below which a chunk counts "recent"
}

// applyChunkRecency returns a copy of scored with each score multiplied by its
// recency factor. turnFrac = turnN / maxTurn(source); 0 when the source has no
// turn anchors. Sources absent from ages (e.g. never-swept) are treated as age 0
// (maximally recent) so a freshly written but not-yet-manifested source is not
// penalised.
func applyChunkRecency(
	scored []scoredChunk,
	ageDaysBySource map[string]float64,
	maxTurnBySrc map[string]int,
	p recencyParams,
) []scoredChunk {
	out := make([]scoredChunk, len(scored))

	for i, s := range scored {
		age := ageDaysBySource[s.record.Source] // missing → 0.0

		turnFrac := 0.0

		if n, ok := parseTurnN(s.record.Anchor); ok {
			if maxN := maxTurnBySrc[s.record.Source]; maxN > 0 {
				turnFrac = float64(n) / float64(maxN)
			}
		}

		out[i] = scoredChunk{
			record: s.record,
			score:  s.score * float32(recencyMultiplier(age, turnFrac, p)),
		}
	}

	return out
}

// defaultRecencyParams returns the eval-tuned recency knobs.
func defaultRecencyParams() recencyParams {
	return recencyParams{
		halfLifeDays: defaultHalfLifeDays,
		tailWeight:   defaultTailWeight,
		floor:        defaultRecencyFloor,
		windowDays:   defaultRecentWindowDays,
	}
}

// fillRecencyBand guarantees at least floor of recentPool's items appear in the
// returned slice of length <= limit. recentPool is the recency-ordered (newest
// first) chunk items the caller deemed "recent". Items already present count
// toward the floor; the deficit is filled from recentPool (skipping those
// already present), displacing the lowest-ranked items NOT in recentPool. No-op
// when the floor is already met or recentPool is empty.
func fillRecencyBand(items, recentPool []resolvedItem, floor, limit int) []resolvedItem {
	recentKey := make(map[string]bool, len(recentPool))
	for _, r := range recentPool {
		recentKey[r.notePath] = true
	}

	present := make(map[string]bool, len(items))
	have := 0

	for _, it := range items {
		present[it.notePath] = true
		if recentKey[it.notePath] {
			have++
		}
	}

	deficit := floor - have
	if deficit <= 0 {
		return items
	}

	// Never inject more than the whole budget — guards floor > limit, where the
	// band would otherwise prepend more recent items than limit allows.
	if deficit > limit {
		deficit = limit
	}

	missing := make([]resolvedItem, 0, deficit)

	for _, r := range recentPool {
		if len(missing) >= deficit {
			break
		}

		if !present[r.notePath] {
			missing = append(missing, r)
		}
	}

	if len(missing) == 0 {
		return items
	}

	return spliceRecent(items, missing, recentKey, limit)
}

// maxTurnBySource returns the highest turn ordinal seen per source.
// Sources with no turn anchors are absent from the map.
func maxTurnBySource(records []chunk.Record) map[string]int {
	maxBySource := make(map[string]int, len(records))

	for _, r := range records {
		n, ok := parseTurnN(r.Anchor)
		if !ok {
			continue
		}

		if cur, seen := maxBySource[r.Source]; !seen || n > cur {
			maxBySource[r.Source] = n
		}
	}

	return maxBySource
}

// parseTurnN extracts the turn ordinal from a "turn-N" anchor.
// Returns (0, false) for preamble/heading anchors that carry no ordinal.
func parseTurnN(anchor string) (int, bool) {
	rest, ok := strings.CutPrefix(anchor, turnAnchorPrefix)
	if !ok {
		return 0, false
	}

	n, err := strconv.Atoi(rest)
	if err != nil || n < 0 {
		return 0, false
	}

	return n, true
}

// recencyMultiplier returns exp2(-ageDays/halfLife) * (1 + tailWeight*turnFrac).
// ageDays>=0; turnFrac in [0,1]. At age 0, turnFrac 0 it is exactly 1.0.
func recencyMultiplier(ageDays, turnFrac float64, p recencyParams) float64 {
	decay := math.Exp2(-ageDays / p.halfLifeDays)

	return decay * (1 + p.tailWeight*turnFrac)
}

// sortScoredDesc sorts in place by descending score (stable).
func sortScoredDesc(scored []scoredChunk) {
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
}

// sourceAgeDays converts per-source mtime (unix nanos) into age in days relative
// to now. Negative ages (clock skew / future mtime) clamp to 0.
func sourceAgeDays(mtimeBySource map[string]int64, now time.Time) map[string]float64 {
	ages := make(map[string]float64, len(mtimeBySource))

	for source, mtime := range mtimeBySource {
		age := now.Sub(time.Unix(0, mtime)).Hours() / hoursPerDay
		if age < 0 {
			age = 0
		}

		ages[source] = age
	}

	return ages
}

// spliceRecent prepends the missing recent items, then refills from the original
// items dropping the lowest-ranked NON-recent ones first, capped at limit.
func spliceRecent(items, missing []resolvedItem, recentKey map[string]bool, limit int) []resolvedItem {
	out := make([]resolvedItem, 0, limit)
	out = append(out, missing...)

	// keep recent items from the original first, then non-recent, in original order.
	for _, item := range items {
		if len(out) >= limit {
			break
		}

		if recentKey[item.notePath] {
			out = append(out, item)
		}
	}

	for _, item := range items {
		if len(out) >= limit {
			break
		}

		if !recentKey[item.notePath] {
			out = append(out, item)
		}
	}

	return out
}
