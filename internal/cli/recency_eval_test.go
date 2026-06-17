package cli_test

import (
	"strconv"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

func TestRecencyEvalSweepAndValidateDefaults(t *testing.T) {
	t.Parallel()

	pool, ages, maxTurn := buildSyntheticPool(40)

	const limit = 20

	for _, hl := range []float64{1, 3, 7, 14} {
		for _, fl := range []int{0, 1, 3} {
			p := cli.ExportNewRecencyParams(hl, 0.2, fl, 1)
			t.Logf("halfLife=%4.0f floor=%d -> plantedRank=%d", hl, fl, plantedRank(pool, ages, maxTurn, p, limit))
		}
	}

	g := NewWithT(t)
	rank := plantedRank(pool, ages, maxTurn, cli.ExportDefaultRecencyParams(), limit)
	g.Expect(rank).To(BeNumerically(">=", 0), "planted chunk must surface")
	g.Expect(rank).To(BeNumerically("<=", 5), "tuned defaults must put planted narration in the top 6 (got rank %d)", rank)
}

// unexported constants.
const (
	evalPlantedHash = "sha256:planted"
)

func buildSyntheticPool(n int) ([]cli.ExportScoredChunk, map[string]float64, map[string]int) {
	planted := chunk.Record{
		Source: "recent.jsonl", Anchor: "turn-40",
		Text:        "ASSISTANT: I'll file issue #644 for the recall flakiness",
		ContentHash: evalPlantedHash,
	}
	pool := make([]cli.ExportScoredChunk, 0, 1+n)
	pool = append(pool, cli.ExportNewScoredChunk(planted, 0.42))
	ages := map[string]float64{"recent.jsonl": 0.01}
	maxTurn := map[string]int{"recent.jsonl": 40}

	for i := range n {
		rec := chunk.Record{
			Source:      "old.jsonl",
			Anchor:      "turn-" + strconv.Itoa(i),
			ContentHash: "sha256:old" + strconv.Itoa(i),
		}
		pool = append(pool, cli.ExportNewScoredChunk(rec, 0.55+0.003*float32(i))) // all beat planted cosine
		ages["old.jsonl"] = 90
		maxTurn["old.jsonl"] = 200
	}

	return pool, ages, maxTurn
}

// plantedRank returns the 0-based rank of the planted chunk after recency re-rank
// + cap + band, or -1. Mirrors the mergeChunkSpace ordering at the chunk level.
func plantedRank(
	pool []cli.ExportScoredChunk,
	ages map[string]float64,
	maxTurn map[string]int,
	p cli.ExportRecencyParams,
	limit int,
) int {
	scored := cli.ExportApplyChunkRecency(pool, ages, maxTurn, p)
	cli.ExportSortScoredDesc(scored)

	// to resolvedItems (chunk kind), cap, then band over the recent pool.
	items := make([]cli.ExportResolvedItem, 0, len(scored))

	var recentPool []cli.ExportResolvedItem

	for _, s := range scored {
		rec := cli.ExportScoredChunkRecord(s)
		it := cli.ExportNewChunkResolvedItem(rec.Source+"#"+rec.Anchor, cli.ExportScoredChunkScore(s))
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
		if cli.ExportResolvedItemPath(it) == "recent.jsonl#turn-40" {
			return i
		}
	}

	return -1
}
