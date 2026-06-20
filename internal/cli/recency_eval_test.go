package cli_test

import (
	"sort"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// TestRecencyEvalDiscriminatingHalfLife sweeps half-life and floor values over a
// synthetic pool that covers a realistic monthly-cadence age distribution. The pool
// contains:
//
//   - Three "weeks-old" chunks from a single source (age=21d) — the load-bearing
//     scenario: the absolute-newest available content is 3 weeks old; the band must
//     still guarantee these appear in the result.
//   - Distractors spread across 1mo / 2mo / 3mo / 4mo / 6mo (8 per tier) with higher
//     cosines than the weeks-old chunks, so they dominate the cap without the band.
//
// Chosen cell after running the sweep: halfLife=60, floor=3.
// Rationale:
//   - With a monthly cadence, short half-lives (1–7d) suppress content within weeks,
//     making recency undesirably aggressive. halfLife=60 gives a gentle tilt:
//     2wk→0.85, 1mo→0.71, 2mo→0.50, 4mo→0.25. The re-rank is a soft bias, not a gate.
//   - The band carries the freshness guarantee: newestChunkItems selects the
//     floor-newest chunks by age regardless of absolute date. When the newest
//     available source is 3 weeks old, the band still force-includes its chunks.
//   - floor=3 ensures the 3 absolute-newest chunks always survive the cap.
//
// Discrimination assertion (honest at hl=60):
//   - (a) Band guarantee: with floor=3 and all distractors older than the weeks-old
//     source, newestChunkItems picks the weeks-old chunks as the floor-newest;
//     fillRecencyBand forces them into the result even when their score ranks them
//     below the distractor cap.
//   - (b) Re-rank monotonicity at hl=60: a 2mo chunk outranks a 4mo chunk of equal
//     cosine (decay ratio ≈2x). This is intentionally soft — the band, not the
//     re-rank, carries the freshness guarantee.
//
// Scope: this eval exercises the chunk-only space (re-rank + band over scored
// chunks). It does NOT cover the mixed chunk+note merged cap in mergeChunkSpace,
// where the band's displacement competes against vault-note items — that
// cross-space behavior is a known boundary, exercised by the query_test.go
// integration test, not here.
func TestRecencyEvalDiscriminatingHalfLife(t *testing.T) {
	t.Parallel()

	pool, maxTurn := buildSyntheticPool()

	const limit = 20

	t.Log("--- half-life sweep (weeks-old probe ranks) ---")

	for _, hl := range []float64{1, 7, 30, 60} {
		for _, fl := range []int{0, 1, 3} {
			params := cli.ExportNewRecencyParams(hl, 0.2, fl)
			r0 := weeksOldRankOf(pool, maxTurn, params, limit, "weeksold.jsonl#turn-10")
			r1 := weeksOldRankOf(pool, maxTurn, params, limit, "weeksold.jsonl#turn-5")
			t.Logf("halfLife=%4.0f floor=%d -> turn-10 rank=%d turn-5 rank=%d", hl, fl, r0, r1)
		}
	}

	g := NewWithT(t)

	// (a) Band guarantee: the 3 weeks-old chunks must appear even though they are
	// outscored by 8 tiers of higher-cosine distractors. At floor=3 and with
	// weeksold.jsonl as the absolute-newest source, newestChunkItems returns all 3
	// weeksold chunks and fillRecencyBand forces them into the cap=20 result.
	defaultParams := cli.ExportDefaultRecencyParams()
	g.Expect(cli.ExportRecencyFloor(defaultParams)).To(BeNumerically(">=", 3),
		"default floor must guarantee at least 3 of the newest chunks")

	r10 := weeksOldRankOf(pool, maxTurn, defaultParams, limit, "weeksold.jsonl#turn-10")
	g.Expect(r10).To(BeNumerically(">=", 0),
		"band must force-include weeks-old newest chunk turn-10 (3-week-old source, floor guarantee)")

	r5 := weeksOldRankOf(pool, maxTurn, defaultParams, limit, "weeksold.jsonl#turn-5")
	g.Expect(r5).To(BeNumerically(">=", 0),
		"band must force-include weeks-old newest chunk turn-5 (3-week-old source, floor guarantee)")

	r1 := weeksOldRankOf(pool, maxTurn, defaultParams, limit, "weeksold.jsonl#turn-1")
	g.Expect(r1).To(BeNumerically(">=", 0),
		"band must force-include weeks-old newest chunk turn-1 (3-week-old source, floor guarantee)")

	// Without the band (floor=0) and a tight cap, the weeks-old chunks are
	// displaced by higher-scoring distractors. This confirms the band is doing
	// real work, not a trivial pass-through.
	const tightLimit = 5 // only the top 5 — all 5 slots go to 1mo distractors at hl=60

	noFloorParams := cli.ExportNewRecencyParams(60.0, 0.2, 0)
	rNoFloor := weeksOldRankOf(pool, maxTurn, noFloorParams, tightLimit, "weeksold.jsonl#turn-10")
	g.Expect(rNoFloor).To(Equal(-1),
		"without the band (floor=0) the weeks-old chunk must not appear in top-%d (non-trivial)", tightLimit)

	// With the band (floor=3) and the same tight cap, the weeks-old chunks are
	// force-included — this is the load-bearing guarantee for Change 2.
	floorParams := cli.ExportNewRecencyParams(60.0, 0.2, 3)
	r10tight := weeksOldRankOf(pool, maxTurn, floorParams, tightLimit, "weeksold.jsonl#turn-10")
	g.Expect(r10tight).To(BeNumerically(">=", 0),
		"with floor=3 the weeks-old newest chunk must be force-included in cap=%d", tightLimit)

	// (b) Re-rank monotonicity at hl=60: a 2mo chunk outranks a 4mo chunk of
	// equal cosine. The gap is modest but real (decay 0.50 vs 0.25).
	const (
		twoMonthAge  = 60.0
		fourMonthAge = 120.0
		equalCosine  = float32(0.5)
		halfLife60   = 60.0
		tailWeight   = 0.2
		zeroFloor    = 0
		zeroTurn     = 0
	)

	tuned := cli.ExportNewRecencyParams(halfLife60, tailWeight, zeroFloor)
	score2mo := float64(equalCosine) * cli.ExportRecencyMultiplier(twoMonthAge, zeroTurn, tuned)
	score4mo := float64(equalCosine) * cli.ExportRecencyMultiplier(fourMonthAge, zeroTurn, tuned)

	g.Expect(score2mo).To(BeNumerically(">", score4mo),
		"2mo chunk must outscore 4mo chunk at hl=60 (monotonic re-rank)")

	// Honest: at hl=60 the discrimination between 2mo and 4mo is intentionally
	// gentle (ratio ≈2x). The band, not the re-rank, carries the freshness
	// guarantee for the absolute-newest content.
}

// unexported constants.
const (
	cosineFourMonth = float32(0.64)
	// cosine* are the raw cosine scores assigned to each distractor tier.
	// All higher than weeksOldCosine so that, without the band, distractors
	// fill the entire cap and the weeks-old chunks are absent.
	cosineOneMonth     = float32(0.70)
	cosineSixMonth     = float32(0.62)
	cosineThreeMonth   = float32(0.66)
	cosineTwoMonth     = float32(0.68)
	distractorsPerTier = 8
)

// biasedScore applies recency multiplier to a scored chunk.
func biasedScore(s cli.ExportScoredChunk, now time.Time, maxTurn map[string]int, p cli.ExportRecencyParams) float32 {
	rec := cli.ExportScoredChunkRecord(s)
	ageDays := 0.0

	if !rec.IngestedAt.IsZero() && !now.IsZero() {
		age := now.Sub(rec.IngestedAt).Hours() / 24
		if age > 0 {
			ageDays = age
		}
	}

	turnFrac := 0.0

	if n, ok := cli.ExportParseTurnN(rec.Anchor); ok {
		if maxN := maxTurn[rec.Source]; maxN > 0 {
			turnFrac = float64(n) / float64(maxN)
		}
	}

	return cli.ExportScoredChunkScore(s) * float32(cli.ExportRecencyMultiplier(ageDays, turnFrac, p))
}

// buildSyntheticPool returns a pool with:
//   - 3 weeks-old chunks from weeksold.jsonl (age=21d, cosine=0.45) — the
//     absolute-newest source; all distractors are older.
//   - 8 distractors each at 1mo, 2mo, 3mo, 4mo, 6mo — all with higher cosines.
//
// Key invariant: without the band, all 20 cap slots go to distractors. With
// floor=3 the band force-includes the 3 weeks-old chunks.
func buildSyntheticPool() ([]cli.ExportScoredChunk, map[string]int) {
	const weeksOldCosine = float32(0.45)

	type tier struct {
		source  string
		ageDays float64
		cosine  float32
	}

	tiers := []tier{
		{source: "src-1mo.jsonl", ageDays: 30.0, cosine: cosineOneMonth},
		{source: "src-2mo.jsonl", ageDays: 60.0, cosine: cosineTwoMonth},
		{source: "src-3mo.jsonl", ageDays: 90.0, cosine: cosineThreeMonth},
		{source: "src-4mo.jsonl", ageDays: 120.0, cosine: cosineFourMonth},
		{source: "src-6mo.jsonl", ageDays: 180.0, cosine: cosineSixMonth},
	}

	totalSize := 3 + len(tiers)*distractorsPerTier
	pool := make([]cli.ExportScoredChunk, 0, totalSize)
	maxTurn := make(map[string]int, 1+len(tiers))
	evalNow := recencyEvalNow()

	// The 3 weeks-old chunks: turn-10 (latest), turn-5 (mid), turn-1 (earliest).
	weeksOldTime := evalNow.Add(-21 * 24 * time.Hour)

	for _, turn := range []string{"turn-10", "turn-5", "turn-1"} {
		rec := chunk.Record{
			Source:      "weeksold.jsonl",
			Anchor:      turn,
			Text:        "ASSISTANT: session narration at " + turn,
			ContentHash: "sha256:weeksold-" + turn,
			IngestedAt:  weeksOldTime,
		}
		pool = append(pool, cli.ExportNewScoredChunk(rec, weeksOldCosine))
	}

	maxTurn["weeksold.jsonl"] = 10

	for _, tier := range tiers {
		maxTurn[tier.source] = distractorsPerTier
		tierTime := evalNow.Add(-time.Duration(tier.ageDays) * 24 * time.Hour)

		for i := range distractorsPerTier {
			rec := chunk.Record{
				Source:      tier.source,
				Anchor:      "turn-" + itoa(i),
				ContentHash: "sha256:" + tier.source + itoa(i),
				IngestedAt:  tierTime,
			}
			pool = append(pool, cli.ExportNewScoredChunk(rec, tier.cosine))
		}
	}

	return pool, maxTurn
}

// itoa converts an int to its decimal string representation.
func itoa(n int) string {
	const digits = "0123456789"
	if n < 10 {
		return string(digits[n])
	}

	return string([]byte{digits[n/10], digits[n%10]})
}

// rankOf returns the 0-based rank of targetPath after recency re-rank + cap +
// band, or -1 if absent. Mirrors the mergeChunkSpace ordering at chunk level.
func rankOf(
	targetPath string,
	pool []cli.ExportScoredChunk,
	maxTurn map[string]int,
	p cli.ExportRecencyParams,
	limit int,
) int {
	now := recencyEvalNow()
	scored := make([]cli.ExportScoredChunk, len(pool))

	for i, s := range pool {
		rec := cli.ExportScoredChunkRecord(s)
		scored[i] = cli.ExportNewScoredChunkWithIngestedAt(rec, biasedScore(s, now, maxTurn, p), rec.IngestedAt)
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return cli.ExportScoredChunkScore(scored[i]) > cli.ExportScoredChunkScore(scored[j])
	})

	items := make([]cli.ExportResolvedItem, 0, len(scored))

	for _, s := range scored {
		rec := cli.ExportScoredChunkRecord(s)
		path := rec.Source + "#" + rec.Anchor
		it := cli.ExportNewChunkResolvedItem(path, cli.ExportScoredChunkScore(s))
		items = append(items, it)
	}

	if len(items) > limit {
		items = items[:limit]
	}

	floor := cli.ExportRecencyFloor(p)
	mustInclude := cli.ExportNewestChunkItems(scored, floor)
	items = cli.ExportFillRecencyBand(items, mustInclude, limit)

	for i, it := range items {
		if cli.ExportResolvedItemPath(it) == targetPath {
			return i
		}
	}

	return -1
}

// recencyEvalNow is the fixed reference time the synthetic pool ages chunks
// against (IngestedAt = recencyEvalNow - ageDays). Keeping it a single helper
// keeps buildSyntheticPool and rankOf consistent.
func recencyEvalNow() time.Time {
	return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
}

// weeksOldRankOf returns the 0-based rank of targetPath after recency re-rank
// + cap + band, or -1 if absent.
func weeksOldRankOf(
	pool []cli.ExportScoredChunk,
	maxTurn map[string]int,
	p cli.ExportRecencyParams,
	limit int,
	targetPath string,
) int {
	return rankOf(targetPath, pool, maxTurn, p, limit)
}
