package cli

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/toejough/engram/internal/chunk"
)

const turnAnchorPrefix = "turn-"

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

// recencyParams are the tunable knobs (defaults chosen by the eval in recency_eval_test.go).
type recencyParams struct {
	halfLifeDays float64 // age at which the decay factor is 0.5
	tailWeight   float64 // extra lift for the last turn of a session (turnFrac=1)
	floor        int     // min recent chunk items the band guarantees
	windowDays   float64 // age below which a chunk counts "recent"
}

// recencyMultiplier returns exp2(-ageDays/halfLife) * (1 + tailWeight*turnFrac).
// ageDays>=0; turnFrac in [0,1]. At age 0, turnFrac 0 it is exactly 1.0.
func recencyMultiplier(ageDays, turnFrac float64, p recencyParams) float64 {
	decay := math.Exp2(-ageDays / p.halfLifeDays)

	return decay * (1 + p.tailWeight*turnFrac)
}

const hoursPerDay = 24

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
