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

	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	oldTime := now.Add(-90 * 24 * time.Hour)
	recentTime := now.Add(-6 * time.Minute)

	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.80, oldTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "recent.jsonl", Anchor: "turn-9"}, 0.45, recentTime),
	}
	maxTurn := map[string]int{"old.jsonl": 3, "recent.jsonl": 9}
	p := cli.ExportNewRecencyParams(3, 0.2, 0)

	out := cli.ExportApplyChunkRecencyByTime(scored, now, maxTurn, p)

	g.Expect(cli.ExportScoredChunkScore(out[1])).To(BeNumerically(">", cli.ExportScoredChunkScore(out[0])))
}

func TestApplyChunkRecencyUsesIngestedAt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	oldTime := now.Add(-90 * 24 * time.Hour) // 90 days ago
	recentTime := now.Add(-1 * time.Hour)    // 1 hour ago

	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.80, oldTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "recent.jsonl", Anchor: "turn-9"}, 0.45, recentTime),
	}

	maxTurn := map[string]int{"old.jsonl": 3, "recent.jsonl": 9}
	p := cli.ExportNewRecencyParams(60, 0.2, 0)

	out := cli.ExportApplyChunkRecencyByTime(scored, now, maxTurn, p)

	// Recent chunk (0.45 base) should outscore old chunk (0.80 base) after recency.
	g.Expect(cli.ExportScoredChunkScore(out[1])).To(
		BeNumerically(">", cli.ExportScoredChunkScore(out[0])),
		"recent chunk must outscore old chunk after per-IngestedAt recency")
}

// TestApplyCombinedRecencyBandInterleavesFairMix verifies that when both
// chunkMust (3 items) and the derived noteMust (3 items from items) exceed the
// limit of 4, the result contains at least 1 chunk AND at least 1 note
// must-item. With the old chunks-first combined slice, fillRecencyBand fills
// the 4-item deficit with the first 4 entries — all chunks — silently dropping
// every note. With interleaving (chunk0, note0, chunk1, note1, ...) the first
// 4 are 2 chunks + 2 notes.
func TestApplyCombinedRecencyBandInterleavesFairMix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// limit=3; chunkMust has 3 items, noteMust will have 3 items (derived from
	// the 3 notes in items). All 6 are absent from the capped set (the 3 stale
	// chunks fill the cap). With chunks-first combined=[c0,c1,c2,n0,n1,n2],
	// fillRecencyBand clamps deficit to limit=3 and injects [c0,c1,c2] only —
	// 0 notes survive. With interleaving [c0,n0,c1,n1,c2,n2], the first 3 are
	// [c0,n0,c1] — at least 1 chunk and 1 note.
	const limit = 3

	// chunkMust — 3 chunk items that are NOT present in items.
	chunkMust := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("chunks.jsonl#turn-3", 0.20),
		cli.ExportNewChunkResolvedItem("chunks.jsonl#turn-2", 0.19),
		cli.ExportNewChunkResolvedItem("chunks.jsonl#turn-1", 0.18),
	}

	// Stale high-score chunks that fill the cap, evicting the notes below.
	stale1 := cli.ExportNewChunkResolvedItem("stale.jsonl#turn-10", 0.99)
	stale2 := cli.ExportNewChunkResolvedItem("stale.jsonl#turn-9", 0.98)
	stale3 := cli.ExportNewChunkResolvedItem("stale.jsonl#turn-8", 0.97)

	// Recently-used notes (will become noteMust via mostRecentlyUsedNoteItems).
	noteA := cli.ExportNewNoteResolvedItem("note-a.md", "2026-06-16", "")
	noteB := cli.ExportNewNoteResolvedItem("note-b.md", "2026-06-15", "")
	noteC := cli.ExportNewNoteResolvedItem("note-c.md", "2026-06-14", "")

	// items sorted descending by score: 3 stale chunks first, then 3 notes.
	// The internal cap (items[:limit=3]) keeps only the 3 stale chunks,
	// evicting all 3 notes. Both chunkMust and noteMust are absent from capped set.
	items := []cli.ExportResolvedItem{stale1, stale2, stale3, noteA, noteB, noteC}

	nowFn := func() time.Time { return time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC) }

	out := cli.ExportApplyCombinedRecencyBand(items, chunkMust, nowFn, limit, true)

	g.Expect(len(out)).To(BeNumerically("<=", limit), "must not exceed limit")

	hasChunk := false
	hasNote := false

	for _, it := range out {
		path := cli.ExportResolvedItemPath(it)
		if len(path) >= 6 && path[:6] == "chunks" {
			hasChunk = true
		} else if len(path) >= 4 && path[:4] == "note" {
			hasNote = true
		}
	}

	g.Expect(hasChunk).To(BeTrue(), "result must contain at least one chunk must-item")
	g.Expect(hasNote).To(BeTrue(), "result must contain at least one note must-item")
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

// TestMergeIntoExistingCopiesLastUsedWhenExistingEmpty verifies that when
// existing.lastUsed is empty and src has a value, the value is copied.
func TestMergeIntoExistingCopiesLastUsedWhenExistingEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	existing := cli.ExportNewNoteResolvedItemWithBaseScore("note.md", 0.6, "", "")
	src := cli.ExportNewNoteResolvedItemWithBaseScore("note.md", 0.5, "2026-06-10", "2026-01-01")

	cli.ExportMergeIntoExisting(&existing, &src)

	g.Expect(cli.ExportResolvedItemLastUsed(existing)).To(Equal("2026-06-10"),
		"lastUsed should be copied from src when existing is empty")
	g.Expect(cli.ExportResolvedItemCreated(existing)).To(Equal("2026-01-01"),
		"created should be copied from src when existing is empty")
}

// TestMergeIntoExistingTakesMaxBaseScore verifies that when the same note is
// matched by two phrases with different baseScores, mergeIntoExisting stores
// the higher one so the activated flag is not phrase-order-dependent.
func TestMergeIntoExistingTakesMaxBaseScore(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// First phrase gave a below-cutoff score; second phrase gave an above-cutoff score.
	existing := cli.ExportNewNoteResolvedItemWithBaseScore("note.md", 0.48, "", "")
	src := cli.ExportNewNoteResolvedItemWithBaseScore("note.md", 0.55, "", "")

	cli.ExportMergeIntoExisting(&existing, &src)

	g.Expect(cli.ExportResolvedItemBaseScore(existing)).To(BeNumerically("~", float32(0.55), 1e-6),
		"baseScore must be max of both phrases, not first-phrase value")
}

func TestMostRecentlyUsedNoteItemsFallsBackToCreated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	// lastUsed empty → falls back to created field.
	noteA := cli.ExportNewNoteResolvedItem("a.md", "", "2026-06-15") // 2.5d via created
	noteB := cli.ExportNewNoteResolvedItem("b.md", "", "2026-06-10") // 7.5d via created

	out := cli.ExportMostRecentlyUsedNoteItems([]cli.ExportResolvedItem{noteB, noteA}, now, 2)

	g.Expect(out).To(HaveLen(2))

	if len(out) < 2 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("a.md"), "fresher (created fallback) must be first")
}

func TestMostRecentlyUsedNoteItemsNZeroReturnsNil(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	items := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItem("a.md", "2026-06-16", ""),
	}

	out := cli.ExportMostRecentlyUsedNoteItems(items, now, 0)
	if out != nil {
		panic("expected nil for n=0")
	}
}

func TestMostRecentlyUsedNoteItemsSelectsNoteKindSortedByAge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	// Three note items with different lastUsed ages plus one chunk item.
	// Ages (from lastUsed→created): fresh=1d, mid=10d, stale=60d.
	notesFresh := cli.ExportNewNoteResolvedItem("fresh.md", "2026-06-16", "")
	notesMid := cli.ExportNewNoteResolvedItem("mid.md", "2026-06-07", "")
	notesStale := cli.ExportNewNoteResolvedItem("stale.md", "2026-04-18", "")
	chunk1 := cli.ExportNewChunkResolvedItem("recent.jsonl#turn-1", 0.9)

	items := []cli.ExportResolvedItem{notesStale, notesMid, chunk1, notesFresh}

	out := cli.ExportMostRecentlyUsedNoteItems(items, now, 2)

	// Must return exactly 2 freshest notes, chunk excluded.
	g.Expect(out).To(HaveLen(2))

	if len(out) < 2 {
		return
	}

	paths := []string{
		cli.ExportResolvedItemPath(out[0]),
		cli.ExportResolvedItemPath(out[1]),
	}

	g.Expect(paths[0]).To(Equal("fresh.md"), "freshest note must be first")
	g.Expect(paths[1]).To(Equal("mid.md"), "second-freshest note must be second")
}

func TestMostRecentlyUsedNoteItemsZeroTimeReturnsNil(t *testing.T) {
	t.Parallel()

	items := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItem("a.md", "2026-06-16", ""),
	}

	out := cli.ExportMostRecentlyUsedNoteItems(items, time.Time{}, 3)
	if out != nil {
		panic("expected nil for zero time")
	}
}

func TestNewestChunkItemsNZeroReturnsNil(t *testing.T) {
	t.Parallel()

	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "a.jsonl", Anchor: "turn-1"}, 0.5,
			time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	out := cli.ExportNewestChunkItems(scored, 0)

	if out != nil {
		panic("expected nil for n=0")
	}
}

func TestNewestChunkItemsOrdersByAgeAscending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	// Three sources: recent (0.5d), mid (14d), old (60d). newestChunkItems
	// should return the floor-newest by IngestedAt, ignoring cosine score.
	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.90,
			now.Add(-60*24*time.Hour)),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "recent.jsonl", Anchor: "turn-7"}, 0.20,
			now.Add(-12*time.Hour)),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "mid.jsonl", Anchor: "turn-5"}, 0.50,
			now.Add(-14*24*time.Hour)),
	}

	out := cli.ExportNewestChunkItems(scored, 2)

	g.Expect(out).To(HaveLen(2))

	if len(out) < 2 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("recent.jsonl#turn-7"), "slot 0 must be newest source")
	g.Expect(cli.ExportResolvedItemPath(out[1])).To(Equal("mid.jsonl#turn-5"), "slot 1 must be second-newest source")
}

func TestNewestChunkItemsSortsByIngestedAt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	oldTime := now.Add(-60 * 24 * time.Hour)
	midTime := now.Add(-14 * 24 * time.Hour)
	recentTime := now.Add(-12 * time.Hour)

	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "old.jsonl", Anchor: "turn-3"}, 0.90, oldTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "recent.jsonl", Anchor: "turn-7"}, 0.20, recentTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "mid.jsonl", Anchor: "turn-5"}, 0.50, midTime),
	}

	out := cli.ExportNewestChunkItems(scored, 2)

	g.Expect(out).To(HaveLen(2))

	if len(out) < 2 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("recent.jsonl#turn-7"), "newest IngestedAt first")
	g.Expect(cli.ExportResolvedItemPath(out[1])).To(Equal("mid.jsonl#turn-5"), "second-newest IngestedAt second")
}

func TestNewestChunkItemsTieBreaksByTurnDesc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Same IngestedAt → descending turn-N wins.
	sameTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "a.jsonl", Anchor: "turn-2"}, 0.5, sameTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "a.jsonl", Anchor: "turn-9"}, 0.5, sameTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "a.jsonl", Anchor: "turn-5"}, 0.5, sameTime),
	}

	out := cli.ExportNewestChunkItems(scored, 2)

	g.Expect(out).To(HaveLen(2))

	if len(out) < 2 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("a.jsonl#turn-9"), "highest turn first on tie")
	g.Expect(cli.ExportResolvedItemPath(out[1])).To(Equal("a.jsonl#turn-5"), "second-highest turn second")
}

func TestNewestChunkItemsTieBreaksByTurnDescIngestedAt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sameTime := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	scored := []cli.ExportScoredChunk{
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "a.jsonl", Anchor: "turn-2"}, 0.5, sameTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "a.jsonl", Anchor: "turn-9"}, 0.5, sameTime),
		cli.ExportNewScoredChunkWithIngestedAt(
			chunk.Record{Source: "a.jsonl", Anchor: "turn-5"}, 0.5, sameTime),
	}

	out := cli.ExportNewestChunkItems(scored, 2)

	g.Expect(out).To(HaveLen(2))

	if len(out) < 2 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(out[0])).To(Equal("a.jsonl#turn-9"), "highest turn on tie")
	g.Expect(cli.ExportResolvedItemPath(out[1])).To(Equal("a.jsonl#turn-5"), "second-highest turn on tie")
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

// TestParseCreatedFromNoteFrontmatterBoundary verifies that a body line
// matching `created:` after the closing `---` fence is NOT misread as the
// frontmatter date.
func TestParseCreatedFromNoteFrontmatterBoundary(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Frontmatter has the real date; body contains a misleading created: line.
	noteWithBothFrontmatterAndBody := []byte(
		"---\ntype: fact\ncreated: 2026-06-10\n---\nSome body text\ncreated: 2020-01-01\n",
	)

	g.Expect(cli.ExportParseCreatedFromNote(noteWithBothFrontmatterAndBody)).
		To(Equal("2026-06-10"),
			"must use frontmatter created:, not body-line created:")

	// No frontmatter at all: body-only created: should return "".
	bodyOnlyCreated := []byte("Some note text\ncreated: 2020-01-01\n")

	g.Expect(cli.ExportParseCreatedFromNote(bodyOnlyCreated)).
		To(Equal(""),
			"body-only created: with no frontmatter must return empty string")
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
