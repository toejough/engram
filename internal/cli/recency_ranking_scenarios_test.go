package cli_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// Test792b21c1a2_RecencyRanking_ProductionHalfLifeAndTailWeight reads the weights through the decay they produce:
// one half-life of age halves the factor, and the last turn of a session is lifted by exactly the tail weight.
func Test792b21c1a2_RecencyRanking_ProductionHalfLifeAndTailWeight(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	params := cli.ExportDefaultRecencyParams()

	atHalfLife := cli.ExportRecencyMultiplier(rr1HalfLifeDays, 0, params)
	atTwoHalfLives := cli.ExportRecencyMultiplier(2*rr1HalfLifeDays, 0, params)
	atLastTurn := cli.ExportRecencyMultiplier(0, 1, params)

	g.Expect(atHalfLife).To(BeNumerically("~", 0.5, rr1MultiplierEpsilon), "60 days of age halves the factor")
	g.Expect(atTwoHalfLives).To(BeNumerically("~", 0.25, rr1MultiplierEpsilon), "the half-life is not a one-off")
	g.Expect(atLastTurn).To(BeNumerically("~", 1+rr1TailWeight, rr1MultiplierEpsilon), "the tail weight is 0.2")
}

// Test792b21c2a2_RecencyRanking_UnreadableOrFutureDateIsNoAge covers a quoted created value (which does not read
// as a date), an unreadable last_used, and dates after the time, for both stamps.
func Test792b21c2a2_RecencyRanking_UnreadableOrFutureDateIsNoAge(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		lastUsed string
		created  string
	}{
		{name: "quoted created", lastUsed: "", created: `"2026-07-01"`},
		{name: "quoted created before the time", lastUsed: "", created: `"2026-06-10"`},
		{name: "unreadable last_used", lastUsed: "not-a-date", created: "2026-06-10"},
		{name: "future created", lastUsed: "", created: "2026-07-01"},
		{name: "future last_used", lastUsed: "2026-07-01", created: "2026-01-01"},
		{name: "future by a day", lastUsed: "", created: "2026-06-18"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			age := cli.ExportNoteAgeDays(tc.lastUsed, tc.created, now)

			g.Expect(age).To(BeNumerically("==", 0))
		})
	}
}

// Test792b21c5a4_RecencyRanking_ChunkWithoutATimeComesLast puts the chunk without a time first in the input, with
// the higher score and the later turn, so only its missing time can put it behind the chunk that has one.
func Test792b21c5a4_RecencyRanking_ChunkWithoutATimeComesLast(t *testing.T) {
	t.Parallel()

	ingestedAt := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC).Add(-rr1IngestAge)

	tests := []struct {
		name          string
		timeFirst     bool
		timedAnchor   string
		untimedAnchor string
	}{
		{name: "untimed chunk listed first", timeFirst: false, timedAnchor: "turn-1", untimedAnchor: "turn-9"},
		{name: "timed chunk listed first", timeFirst: true, timedAnchor: "turn-1", untimedAnchor: "turn-9"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			timed := cli.ExportNewScoredChunkWithIngestedAt(
				chunk.Record{Source: "timed.jsonl", Anchor: tc.timedAnchor}, 0.1, ingestedAt)
			untimed := cli.ExportNewScoredChunk(
				chunk.Record{Source: "untimed.jsonl", Anchor: tc.untimedAnchor}, 0.9)

			scored := []cli.ExportScoredChunk{untimed, timed}
			if tc.timeFirst {
				scored = []cli.ExportScoredChunk{timed, untimed}
			}

			out := cli.ExportNewestChunkItems(scored, 1)

			g.Expect(out).To(HaveLen(1))

			if len(out) < 1 {
				return
			}

			g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("timed.jsonl#"+tc.timedAnchor),
				"the chunk with a time wins over the untimed one despite score and turn")
		})
	}
}

// unexported constants.
const (
	rr1HalfLifeDays      = 60.0
	rr1IngestAge         = 7 * 24 * time.Hour
	rr1MultiplierEpsilon = 1e-9
	rr1TailWeight        = 0.2
)
