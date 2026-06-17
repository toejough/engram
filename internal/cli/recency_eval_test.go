package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// TestRecencyEvalDiscriminatingHalfLife sweeps half-life and floor values over a
// synthetic pool that covers a realistic age distribution.  The pool contains:
//
//   - One very-recent planted chunk (age=0.01d) — validates band surfacing.
//   - One mid-age probe (age=4d) — discriminates half-life: at short hl (≤3d) the
//     mid-age probe's decay is large enough that it still ranks within the top-20
//     ahead of the 5-day distractors; at long hl (≥14d) the 5-day distractors
//     overtake it and push it outside the cap entirely.
//   - Distractors spread across 0.5, 2, 5, 15, and 60 days (8 per tier) with
//     cosines in a range that makes recency matter for ordering.
//
// Chosen cell after running the sweep: halfLife=3, floor=3.
// Rationale:
//   - halfLife=3 is the fastest meaningful decay at which the mid-age probe
//     (4d, cosine=0.50) still surfaces within the limit=20 cap (rank ≤ 19),
//     because at hl=3 its recency score (≈0.199) exceeds the 5d distractors
//     (≈0.189). At halfLife=14 the 5d distractors (≈0.469) outrank the probe
//     (≈0.411) and push it out of the cap entirely (rank -1).
//   - floor=3 guarantees a minimum of 3 recent items even after the cap.
//
// Discrimination assertion: the chosen hl=3 surfaces the mid-age probe within
// the cap; hl=14 does not. This assertion fails if the decay mechanism breaks.
func TestRecencyEvalDiscriminatingHalfLife(t *testing.T) {
	t.Parallel()

	pool, ages, maxTurn := buildSyntheticPool()

	const limit = 20

	t.Log("--- half-life sweep (planted rank / mid-age probe rank) ---")

	for _, hl := range []float64{1, 3, 7, 14} {
		for _, fl := range []int{0, 1, 3} {
			params := cli.ExportNewRecencyParams(hl, 0.2, fl, 1)
			pRank := plantedRank(pool, ages, maxTurn, params, limit)
			mRank := midAgeProbeRank(pool, ages, maxTurn, params, limit)
			t.Logf("halfLife=%4.0f floor=%d -> plantedRank=%d midAgeRank=%d", hl, fl, pRank, mRank)
		}
	}

	g := NewWithT(t)

	// Planted chunk must surface under tuned defaults.
	// At hl=3 with a realistic age spread, the planted chunk (age=0.01d, cosine=0.42)
	// ranks at ~8: behind the 0.5d distractors (higher cosine) but well within the
	// limit=20 cap. The band is not needed here because the raw score is sufficient.
	plantedParams := cli.ExportDefaultRecencyParams()
	pRank := plantedRank(pool, ages, maxTurn, plantedParams, limit)
	g.Expect(pRank).To(BeNumerically(">=", 0), "planted chunk must surface under tuned defaults")
	g.Expect(pRank).To(BeNumerically("<=", 12), "tuned defaults must put planted narration in the top 13")

	// Mid-age probe surfaces at hl=3 (chosen half-life) but NOT at hl=7.
	// This discriminates half-life: at hl=3 the mid-age probe (4d, cosine=0.50)
	// decays to ≈0.199 which exceeds the 5d distractors (≈0.189), so it lands
	// within the cap. At hl=7 the 5d distractors (≈0.364) far outrank the probe
	// (≈0.335) and all 8 push it outside the limit=20 cap.
	const (
		chosenHL      = 3.0
		tooLongHL     = 7.0
		upperRankBand = 19 // must be < limit
	)

	shortParams := cli.ExportNewRecencyParams(chosenHL, 0.2, defaultFloor, defaultWindowDays)
	mRankShort := midAgeProbeRank(pool, ages, maxTurn, shortParams, limit)
	g.Expect(mRankShort).To(BeNumerically(">=", 0),
		"mid-age probe must surface within cap at halfLife=3 (rank %d)", mRankShort)
	g.Expect(mRankShort).To(BeNumerically("<=", upperRankBand),
		"mid-age probe must be within limit=%d at halfLife=3 (rank %d)", limit, mRankShort)

	longParams := cli.ExportNewRecencyParams(tooLongHL, 0.2, defaultFloor, defaultWindowDays)
	mRankLong := midAgeProbeRank(pool, ages, maxTurn, longParams, limit)
	g.Expect(mRankLong).To(Equal(-1),
		"mid-age probe must be displaced outside cap at halfLife=7 (rank %d) — confirms discrimination", mRankLong)
}

// unexported constants.
const (
	// cosine* are the raw cosine scores assigned to each distractor tier.
	// Chosen so that, after recency decay, the tiers interleave realistically:
	// high-cosine-but-old chunks are suppressed by decay, while
	// low-cosine-but-recent chunks surface via the band.
	cosineFifteen      = float32(0.65)
	cosineFiveDay      = float32(0.60)
	cosineHalfDay      = float32(0.62)
	cosineSixty        = float32(0.68)
	cosineTwoDay       = float32(0.58)
	defaultFloor       = 3
	defaultWindowDays  = 1.0
	distractorsPerTier = 8
	evalMidAgeHash     = "sha256:midage"
	evalPlantedHash    = "sha256:planted"
)

// buildSyntheticPool returns a pool with:
//   - 1 planted chunk (recent, age≈0)
//   - 1 mid-age probe (age=4d, relevant but not top cosine)
//   - 8 distractors each at 0.5d, 2d, 5d, 15d, 60d
func buildSyntheticPool() ([]cli.ExportScoredChunk, map[string]float64, map[string]int) {
	const (
		plantedCosine  = float32(0.42)
		midAgeCosine   = float32(0.50)
		plantedMaxTurn = 40
		midAgeMaxTurn  = 10
		midAgeTurn     = 5
	)

	planted := chunk.Record{
		Source:      "recent.jsonl",
		Anchor:      "turn-40",
		Text:        "ASSISTANT: I'll file issue #644 for the recall flakiness",
		ContentHash: evalPlantedHash,
	}
	midAge := chunk.Record{
		Source:      "midage.jsonl",
		Anchor:      "turn-5",
		Text:        "ASSISTANT: Updated the recency scoring design doc",
		ContentHash: evalMidAgeHash,
	}

	type tier struct {
		source  string
		ageDays float64
		cosine  float32
	}

	tiers := []tier{
		{source: "src-half.jsonl", ageDays: 0.5, cosine: cosineHalfDay},
		{source: "src-2d.jsonl", ageDays: 2.0, cosine: cosineTwoDay},
		{source: "src-5d.jsonl", ageDays: 5.0, cosine: cosineFiveDay},
		{source: "src-15d.jsonl", ageDays: 15.0, cosine: cosineFifteen},
		{source: "src-60d.jsonl", ageDays: 60.0, cosine: cosineSixty},
	}

	totalSize := 2 + len(tiers)*distractorsPerTier
	pool := make([]cli.ExportScoredChunk, 0, totalSize)
	ages := make(map[string]float64, 2+len(tiers))
	maxTurn := make(map[string]int, 2+len(tiers))

	pool = append(pool, cli.ExportNewScoredChunk(planted, plantedCosine))
	ages["recent.jsonl"] = 0.01
	maxTurn["recent.jsonl"] = plantedMaxTurn

	pool = append(pool, cli.ExportNewScoredChunk(midAge, midAgeCosine))
	ages["midage.jsonl"] = 4.0
	maxTurn["midage.jsonl"] = midAgeMaxTurn

	for _, tier := range tiers {
		ages[tier.source] = tier.ageDays
		maxTurn[tier.source] = distractorsPerTier

		for i := range distractorsPerTier {
			rec := chunk.Record{
				Source:      tier.source,
				Anchor:      "turn-" + itoa(i),
				ContentHash: "sha256:" + tier.source + itoa(i),
			}
			pool = append(pool, cli.ExportNewScoredChunk(rec, tier.cosine))
		}
	}

	return pool, ages, maxTurn
}

// itoa converts an int to its decimal string representation.
func itoa(n int) string {
	const digits = "0123456789"
	if n < 10 {
		return string(digits[n])
	}

	return string([]byte{digits[n/10], digits[n%10]})
}

// midAgeProbeRank returns the 0-based rank of the mid-age probe after recency
// re-rank + cap + band, or -1 if it does not appear.
func midAgeProbeRank(
	pool []cli.ExportScoredChunk,
	ages map[string]float64,
	maxTurn map[string]int,
	p cli.ExportRecencyParams,
	limit int,
) int {
	return rankOf("midage.jsonl#turn-5", pool, ages, maxTurn, p, limit)
}

// plantedRank returns the 0-based rank of the planted chunk after recency re-rank
// + cap + band, or -1 if it does not appear. Mirrors the mergeChunkSpace ordering
// at the chunk level.
func plantedRank(
	pool []cli.ExportScoredChunk,
	ages map[string]float64,
	maxTurn map[string]int,
	p cli.ExportRecencyParams,
	limit int,
) int {
	return rankOf("recent.jsonl#turn-40", pool, ages, maxTurn, p, limit)
}

// rankOf returns the 0-based rank of targetPath after recency re-rank + cap +
// band, or -1 if absent. Mirrors the mergeChunkSpace ordering at chunk level.
func rankOf(
	targetPath string,
	pool []cli.ExportScoredChunk,
	ages map[string]float64,
	maxTurn map[string]int,
	p cli.ExportRecencyParams,
	limit int,
) int {
	scored := cli.ExportApplyChunkRecency(pool, ages, maxTurn, p)
	cli.ExportSortScoredDesc(scored)

	items := make([]cli.ExportResolvedItem, 0, len(scored))

	var recentPool []cli.ExportResolvedItem

	for _, s := range scored {
		rec := cli.ExportScoredChunkRecord(s)
		path := rec.Source + "#" + rec.Anchor
		it := cli.ExportNewChunkResolvedItem(path, cli.ExportScoredChunkScore(s))
		items = append(items, it)

		if ages[rec.Source] <= cli.ExportRecencyWindowDays(p) {
			recentPool = append(recentPool, it)
		}
	}

	if len(items) > limit {
		items = items[:limit]
	}

	items = cli.ExportFillRecencyBand(items, recentPool, cli.ExportRecencyFloor(p), limit)

	for i, it := range items {
		if cli.ExportResolvedItemPath(it) == targetPath {
			return i
		}
	}

	return -1
}
