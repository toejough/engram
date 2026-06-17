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
	p := cli.ExportNewRecencyParams(3, 0.2, 0)

	out := cli.ExportApplyChunkRecency(scored, ages, maxTurn, p)

	g.Expect(cli.ExportScoredChunkScore(out[1])).To(BeNumerically(">", cli.ExportScoredChunkScore(out[0])))
}

func TestDefaultRecencyParamsSaneDefaults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	p := cli.ExportDefaultRecencyParams()

	g.Expect(cli.ExportRecencyFloor(p)).To(BeNumerically(">", 0))
}

func TestFillRecencyBandBackfillsDeficit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// capped items: all stale notes/chunks; recentPool has 2 recent chunk items not present.
	items := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("old.jsonl#turn-1", 0.9),
		cli.ExportNewChunkResolvedItem("old.jsonl#turn-2", 0.8),
		cli.ExportNewChunkResolvedItem("old.jsonl#turn-3", 0.7),
	}
	recentPool := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-9", 0.30),
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-8", 0.20),
	}

	out := cli.ExportFillRecencyBand(items, recentPool, len(items))

	g.Expect(out).To(HaveLen(len(items))) // budget preserved

	paths := map[string]bool{}
	for _, it := range out {
		paths[cli.ExportResolvedItemPath(it)] = true
	}

	g.Expect(paths["recent.jsonl#turn-9"]).To(BeTrue())
	g.Expect(paths["recent.jsonl#turn-8"]).To(BeTrue())
	g.Expect(paths["old.jsonl#turn-1"]).To(BeTrue()) // highest-ranked stale retained
}

func TestFillRecencyBandClampsToLimit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("old.jsonl#turn-1", 0.9),
	}
	recentPool := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-9", 0.3),
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-8", 0.2),
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-7", 0.1),
	}

	// mustInclude (3) > limit (2): the band must never grow the payload past limit.
	out := cli.ExportFillRecencyBand(items, recentPool, 2)

	g.Expect(len(out)).To(BeNumerically("<=", 2))
}

func TestFillRecencyBandGuaranteesWeeksOldNewest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Scenario: newest content is 21 days old (3 weeks). Without the window gate
	// it should still be force-included by the band.
	newest := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("session-3wk.jsonl#turn-10", 0.30),
	}
	// items capped before band fires — does not contain the newest chunk.
	items := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("old.jsonl#turn-1", 0.99),
		cli.ExportNewChunkResolvedItem("old.jsonl#turn-2", 0.95),
		cli.ExportNewChunkResolvedItem("old.jsonl#turn-3", 0.90),
	}

	out := cli.ExportFillRecencyBand(items, newest, len(items))

	g.Expect(out).To(HaveLen(len(items)))

	paths := make(map[string]bool, len(out))

	for _, it := range out {
		paths[cli.ExportResolvedItemPath(it)] = true
	}

	g.Expect(paths["session-3wk.jsonl#turn-10"]).To(BeTrue(),
		"band must force-include weeks-old newest chunk even when it did not score into the cap")
}

func TestFillRecencyBandNoDeficitNoChange(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-9", 0.9),
		cli.ExportNewChunkResolvedItem("recent.jsonl#turn-8", 0.8),
	}
	recentPool := items // both already present and recent

	out := cli.ExportFillRecencyBand(items, recentPool, len(items))
	g.Expect(out).To(Equal(items))
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

func TestNewestChunkItemsNZeroReturnsNil(t *testing.T) {
	t.Parallel()

	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunk(chunk.Record{Source: "a.jsonl", Anchor: "turn-1"}, 0.5),
	}
	ages := map[string]float64{"a.jsonl": 1.0}

	out := cli.ExportNewestChunkItems(scored, ages, 0)

	if out != nil {
		panic("expected nil for n=0")
	}
}

func TestNewestChunkItemsOrdersByAgeAscending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Three sources: recent (0.5d), mid (14d), old (60d). newestChunkItems
	// should return the floor-newest by age, ignoring cosine score.
	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunk(chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.90),
		cli.ExportNewScoredChunk(chunk.Record{Source: "recent.jsonl", Anchor: "turn-7"}, 0.20),
		cli.ExportNewScoredChunk(chunk.Record{Source: "mid.jsonl", Anchor: "turn-5"}, 0.50),
	}
	ages := map[string]float64{"old.jsonl": 60.0, "recent.jsonl": 0.5, "mid.jsonl": 14.0}

	out := cli.ExportNewestChunkItems(scored, ages, 2)

	g.Expect(out).To(HaveLen(2))

	if len(out) < 2 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("recent.jsonl#turn-7"), "slot 0 must be newest source")
	g.Expect(cli.ExportResolvedItemPath(out[1])).To(Equal("mid.jsonl#turn-5"), "slot 1 must be second-newest source")
}

func TestNewestChunkItemsTieBreaksByTurnDesc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Same source age → descending turn-N wins.
	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunk(chunk.Record{Source: "a.jsonl", Anchor: "turn-2"}, 0.5),
		cli.ExportNewScoredChunk(chunk.Record{Source: "a.jsonl", Anchor: "turn-9"}, 0.5),
		cli.ExportNewScoredChunk(chunk.Record{Source: "a.jsonl", Anchor: "turn-5"}, 0.5),
	}
	ages := map[string]float64{"a.jsonl": 3.0}

	out := cli.ExportNewestChunkItems(scored, ages, 2)

	g.Expect(out).To(HaveLen(2))

	if len(out) < 2 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("a.jsonl#turn-9"), "highest turn first on tie")
	g.Expect(cli.ExportResolvedItemPath(out[1])).To(Equal("a.jsonl#turn-5"), "second-highest turn second")
}

func TestNoteAgeDaysPrefersLastUsedThenCreated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// noon UTC; date-only stamps parse to midnight, so age = days + fraction.
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	// LastUsed=June 15 midnight → 2.5 days to noon June 17.
	g.Expect(cli.ExportNoteAgeDays("2026-06-15", "2026-01-01", now)).To(BeNumerically("~", 2.5, 0.01))
	// No LastUsed → falls back to created=June 10 midnight → 7.5 days.
	g.Expect(cli.ExportNoteAgeDays("", "2026-06-10", now)).To(BeNumerically("~", 7.5, 0.01))
	// Empty both → 0 (treat as fresh).
	g.Expect(cli.ExportNoteAgeDays("", "", now)).To(BeNumerically("~", 0.0, 0.01))
}

func TestParseCreatedFromNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	note := []byte("---\ntype: fact\ncreated: 2026-06-10\nsituation: x\n---\nbody")

	g.Expect(cli.ExportParseCreatedFromNote(note)).To(Equal("2026-06-10"))
	g.Expect(cli.ExportParseCreatedFromNote([]byte("no frontmatter"))).To(Equal(""))
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

func TestRecencyMultiplier(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	p := cli.ExportNewRecencyParams(3, 0.2, 0) // halfLife=3, tail=0.2

	g.Expect(cli.ExportRecencyMultiplier(0, 0, p)).To(BeNumerically("~", 1.0, 1e-6))
	g.Expect(cli.ExportRecencyMultiplier(3, 0, p)).To(BeNumerically("~", 0.5, 1e-6))
	g.Expect(cli.ExportRecencyMultiplier(0, 1, p)).To(BeNumerically("~", 1.2, 1e-6))
	g.Expect(cli.ExportRecencyMultiplier(6, 0, p)).To(BeNumerically("<", cli.ExportRecencyMultiplier(3, 0, p)))
}

func TestSortScoredDescOrdersDescending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunk(chunk.Record{Source: "a.jsonl", Anchor: "turn-0"}, 0.3),
		cli.ExportNewScoredChunk(chunk.Record{Source: "b.jsonl", Anchor: "turn-1"}, 0.9),
		cli.ExportNewScoredChunk(chunk.Record{Source: "c.jsonl", Anchor: "turn-2"}, 0.6),
	}

	cli.ExportSortScoredDesc(scored)

	g.Expect(cli.ExportScoredChunkScore(scored[0])).To(BeNumerically("~", 0.9, 1e-6))
	g.Expect(cli.ExportScoredChunkScore(scored[1])).To(BeNumerically("~", 0.6, 1e-6))
	g.Expect(cli.ExportScoredChunkScore(scored[2])).To(BeNumerically("~", 0.3, 1e-6))
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
