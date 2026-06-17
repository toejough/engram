package cli_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

func TestApplyChunkRecencyLiftsRecentOverStaleHighCosine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunk(chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.80),
		cli.ExportNewScoredChunk(chunk.Record{Source: "recent.jsonl", Anchor: "turn-9"}, 0.45),
	}
	ages := map[string]float64{"old.jsonl": 90, "recent.jsonl": 0.01}
	maxTurn := map[string]int{"old.jsonl": 3, "recent.jsonl": 9}
	p := cli.ExportNewRecencyParams(3, 0.2, 0, 1)

	out := cli.ExportApplyChunkRecency(scored, ages, maxTurn, p)

	g.Expect(cli.ExportScoredChunkScore(out[1])).To(BeNumerically(">", cli.ExportScoredChunkScore(out[0])))
}

func TestSourceAgeDays(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	mtimes := map[string]int64{
		"recent.jsonl": now.Add(-12 * time.Hour).UnixNano(),
		"old.jsonl":    now.Add(-72 * time.Hour).UnixNano(),
		"future.jsonl": now.Add(24 * time.Hour).UnixNano(), // clamp to 0
	}

	got := cli.ExportSourceAgeDays(mtimes, now)

	g.Expect(got["recent.jsonl"]).To(BeNumerically("~", 0.5, 1e-6))
	g.Expect(got["old.jsonl"]).To(BeNumerically("~", 3.0, 1e-6))
	g.Expect(got["future.jsonl"]).To(BeNumerically("~", 0.0, 1e-6))
}

func TestRecencyMultiplier(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	p := cli.ExportNewRecencyParams(3, 0.2, 0, 0) // halfLife=3, tail=0.2

	g.Expect(cli.ExportRecencyMultiplier(0, 0, p)).To(BeNumerically("~", 1.0, 1e-6))
	g.Expect(cli.ExportRecencyMultiplier(3, 0, p)).To(BeNumerically("~", 0.5, 1e-6))
	g.Expect(cli.ExportRecencyMultiplier(0, 1, p)).To(BeNumerically("~", 1.2, 1e-6))
	g.Expect(cli.ExportRecencyMultiplier(6, 0, p)).To(BeNumerically("<", cli.ExportRecencyMultiplier(3, 0, p)))
}

func TestMaxTurnBySource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	recs := []chunk.Record{
		{Source: "a.jsonl", Anchor: "turn-0"},
		{Source: "a.jsonl", Anchor: "turn-5"},
		{Source: "a.jsonl", Anchor: "preamble"},
		{Source: "b.jsonl", Anchor: "turn-2"},
		{Source: "c.md", Anchor: "Heading"},
	}

	got := cli.ExportMaxTurnBySource(recs)

	g.Expect(got["a.jsonl"]).To(Equal(5))
	g.Expect(got["b.jsonl"]).To(Equal(2))
	_, hasC := got["c.md"]
	g.Expect(hasC).To(BeFalse())
}

func TestParseTurnN(t *testing.T) {
	t.Parallel()

	cases := []struct {
		anchor string
		wantN  int
		wantOK bool
	}{
		{"turn-0", 0, true},
		{"turn-42", 42, true},
		{"preamble", 0, false},
		{"Some Heading", 0, false},
		{"turn-", 0, false},
		{"turn-x", 0, false},
	}

	for _, tc := range cases {
		g := NewWithT(t)
		gotN, gotOK := cli.ExportParseTurnN(tc.anchor)
		g.Expect(gotOK).To(Equal(tc.wantOK), "ok for %q", tc.anchor)
		g.Expect(gotN).To(Equal(tc.wantN), "n for %q", tc.anchor)
	}
}
