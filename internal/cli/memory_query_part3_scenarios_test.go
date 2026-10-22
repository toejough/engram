package cli_test

// Scenario tests for the Memory Query product capability, part 3 (861c2zn): the newest chunks (792a7o), the limit
// (792a7q), the chunk content budget (792a7r) and lazy chunks (792a7s). The first-indexed time of a chunk (792a7p)
// is in memory_query_part3_ingest_scenarios_test.go. Spec: docs/specs/product/memory-query.md.
//
// The scenarios run cli.RunQuery over the in-memory world of memory_query_part1_scenarios_test.go (mq1World:
// a phrase-keyed embedder, so every cosine is a number the test chose). The scenarios that name a flag value the
// command line must parse (--recent-fill=-1, --recent-fill 0) or an environment variable (ENGRAM_RECENT_FILL,
// ENGRAM_LAZY_CHUNKS) run `engram query` through cli.Targets over a t.TempDir() vault and chunk directory (the
// recorded deviation, #860): the flag parser reads the environment with os.Getenv, so the two environment scenarios
// cannot be parallel (t.Setenv). The helpers carry the mq3 prefix.

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// Test792a7o1a_RunQuery_RecentChunksFollowTheMatches verifies 792a7o1a (Recent chunks follow the matches): of
// three chunks, one of which matches the phrase, items ends with the two other chunks, newest first, each with
// provenance recent and score 0, and they are in no cluster.
func Test792a7o1a_RunQuery_RecentChunksFollowTheMatches(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase        = "the phrase"
		matchedAnchor = "section-matched"
		olderAnchor   = "section-older"
		newerAnchor   = "section-newer"
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = []chunk.Record{
		mq3ChunkRecord(world, olderAnchor, 0, mq3DaysAgo(2), "the older chunk, not about the phrase"),
		mq3ChunkRecord(world, matchedAnchor, mq1Strong, mq3DaysAgo(3), "the matched chunk, about the phrase"),
		mq3ChunkRecord(world, newerAnchor, 0, mq3DaysAgo(1), "the newer chunk, not about the phrase"),
	}

	payload := mq3Run(t, world, cli.QueryArgs{}).payload

	g.Expect(payload.paths()).To(Equal([]string{
		mq1ChunkPath(matchedAnchor), mq1ChunkPath(newerAnchor), mq1ChunkPath(olderAnchor),
	}), "the match, then the other two chunks, newest first")

	if len(payload.Items) == 0 {
		return
	}

	for _, item := range payload.Items[1:] {
		g.Expect(item.Provenances).To(Equal([]string{"recent"}), "%s is a recent chunk", item.Path)
		g.Expect(item.Score).To(BeZero(), "%s has score 0", item.Path)
		g.Expect(item.ClusterID).To(BeNil(), "%s is in no cluster", item.Path)
	}

	g.Expect(payload.memberPaths()).NotTo(ContainElement(mq1ChunkPath(newerAnchor)), "the newer is no cluster member")
	g.Expect(payload.memberPaths()).NotTo(ContainElement(mq1ChunkPath(olderAnchor)), "the older is no cluster member")
}

// Test792a7o1b_RunQuery_AMatchedChunkIsNotRepeated verifies 792a7o1b (A matched chunk is not repeated): a chunk
// that is both matched and among the newest appears once, with provenance direct.
func Test792a7o1b_RunQuery_AMatchedChunkIsNotRepeated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase        = "the phrase"
		matchedAnchor = "section-matched"
		otherAnchor   = "section-other"
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = []chunk.Record{
		mq3ChunkRecord(world, matchedAnchor, mq1Strong, mq3DaysAgo(0), "the matched chunk, the newest of all"),
		mq3ChunkRecord(world, otherAnchor, 0, mq3DaysAgo(1), "another chunk, not about the phrase"),
	}

	payload := mq3Run(t, world, cli.QueryArgs{}).payload

	g.Expect(payload.paths()).To(Equal([]string{mq1ChunkPath(matchedAnchor), mq1ChunkPath(otherAnchor)}),
		"the matched chunk appears once")

	matched, found := payload.find(mq1ChunkPath(matchedAnchor))
	g.Expect(found).To(BeTrue())
	g.Expect(matched.Provenances).To(Equal([]string{"direct"}))
}

// Test792a7o1c_Targets_Query_TheCountCanBeSet verifies 792a7o1c (The count can be set): with three chunks and
// none matched, engram query --recent-fill 1 returns exactly one recent chunk, the newest, and so does
// ENGRAM_RECENT_FILL=1. Not parallel: the environment half needs t.Setenv.
func Test792a7o1c_Targets_Query_TheCountCanBeSet(t *testing.T) {
	g := NewWithT(t)

	world, newest := mq3UnmatchedWorld()

	t.Setenv(mq3RecentFillEnv, "")

	byFlag := mq3TargetsQuery(t, world, "--recent-fill", "1")
	g.Expect(byFlag.payload.paths()).To(Equal([]string{newest}), "--recent-fill 1 returns the newest chunk only")
	g.Expect(byFlag.payload.withProvenance("recent")).To(HaveLen(1))

	t.Setenv(mq3RecentFillEnv, "1")

	byEnv := mq3TargetsQuery(t, world)
	g.Expect(byEnv.payload.paths()).To(Equal([]string{newest}), "ENGRAM_RECENT_FILL=1 returns the newest chunk only")
	g.Expect(byEnv.payload.withProvenance("recent")).To(HaveLen(1))
}

// Test792a7o1d_Targets_Query_ANegativeCountTurnsTheChannelOff verifies 792a7o1d (A negative count turns the
// channel off): with three chunks, engram query --recent-fill=-1 returns no item with provenance recent.
func Test792a7o1d_Targets_Query_ANegativeCountTurnsTheChannelOff(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	world, _ := mq3UnmatchedWorld()

	off := mq3TargetsQuery(t, world, "--recent-fill=-1")

	g.Expect(off.payload.withProvenance("recent")).To(BeEmpty(), "no item has provenance recent")
	g.Expect(off.payload.Items).To(BeEmpty(), "and no chunk is returned in any other role: none matched")
}

// Test792a7o1e_Targets_Query_ZeroMeansTheDefault verifies 792a7o1e (Zero means the default): with three chunks,
// engram query --recent-fill 0 returns all three as recent chunks, as with no flag.
func Test792a7o1e_Targets_Query_ZeroMeansTheDefault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	world, _ := mq3UnmatchedWorld()

	zero := mq3TargetsQuery(t, world, "--recent-fill", "0")
	bare := mq3Run(t, world, cli.QueryArgs{}).payload

	const chunkCount = 3

	g.Expect(zero.payload.withProvenance("recent")).To(HaveLen(chunkCount), "all three chunks are recent")
	g.Expect(zero.payload.paths()).To(Equal(bare.paths()), "the same items, in the same order, as with no flag")
}

// Test792a7o1f_RunQuery_AChunkWithoutATimeSortsLast verifies 792a7o1f (A chunk without a time sorts last): of a
// chunk with no first-indexed time and a chunk with one, a count of 1 returns the chunk with a time (the chunk
// without a time has the later turn, so only the missing time puts it last).
func Test792a7o1f_RunQuery_AChunkWithoutATimeSortsLast(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase         = "the phrase"
		untimedAnchor  = "turn-9"
		timedAnchor    = "turn-1"
		recentFillOnly = 1
	)

	untimed := func(world *mq1World) chunk.Record {
		return mq3ChunkRecord(world, untimedAnchor, 0, time.Time{}, "the later turn, with no first-indexed time")
	}
	timed := func(world *mq1World) chunk.Record {
		return mq3ChunkRecord(world, timedAnchor, 0, mq3DaysAgo(30), "the earlier turn, first indexed a month ago")
	}

	// The comparison is made on pairs in the order the index lists them, so both orders are run.
	orders := map[string]func(world *mq1World) []chunk.Record{
		"the chunk without a time listed first": func(world *mq1World) []chunk.Record {
			return []chunk.Record{untimed(world), timed(world)}
		},
		"the chunk with a time listed first": func(world *mq1World) []chunk.Record {
			return []chunk.Record{timed(world), untimed(world)}
		},
	}

	for name, listing := range orders {
		world := newMq1World(mq1SmallDims, phrase)
		world.chunks = listing(world)

		payload := mq3Run(t, world, cli.QueryArgs{RecentFill: recentFillOnly}).payload

		g.Expect(payload.paths()).To(Equal([]string{mq1ChunkPath(timedAnchor)}),
			"%s: the chunk with a time is the one returned", name)
	}
}

// Test792a7q1a_RunQuery_TheDefaultCapsALargeResult verifies 792a7q1a (The default caps a large result): with 25
// matched items and no --limit, items holds exactly 20 of them, the highest ranked, and the budget's limit is 20.
func Test792a7q1a_RunQuery_TheDefaultCapsALargeResult(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase       = "the phrase"
		matchedCount = 25
		defaultLimit = 20
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = world.cosineRamp(matchedCount, 0.50, 0.01)

	payload := mq3Run(t, world, cli.QueryArgs{}).payload

	g.Expect(payload.Items).To(HaveLen(defaultLimit))
	g.Expect(payload.Budget.Limit).To(Equal(defaultLimit))

	// The ramp rises with the index, so the 20 highest ranked are the last 20, best first.
	wantPaths := make([]string, 0, defaultLimit)

	for index := matchedCount - 1; index >= matchedCount-defaultLimit; index-- {
		wantPaths = append(wantPaths, mq1ChunkPath(mq1RampAnchor(index)))
	}

	g.Expect(payload.paths()).To(Equal(wantPaths), "the highest ranked 20, in rank order")
}

// Test792a7q1b_RunQuery_AnExplicitLimitCapsTheResult verifies 792a7q1b (An explicit limit caps the result): with
// 12 matched items, --limit 5 holds at most 5 of them, the highest ranked.
func Test792a7q1b_RunQuery_AnExplicitLimitCapsTheResult(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase       = "the phrase"
		matchedCount = 12
		explicitCap  = 5
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = world.cosineRamp(matchedCount, 0.50, 0.01)

	payload := mq3Run(t, world, cli.QueryArgs{Limit: explicitCap}).payload

	g.Expect(payload.Items).To(HaveLen(explicitCap))
	g.Expect(payload.Budget.Limit).To(Equal(explicitCap))

	for rank, item := range payload.Items {
		g.Expect(item.Path).To(Equal(mq1ChunkPath(mq1RampAnchor(matchedCount-1-rank))), "rank %d", rank)
	}
}

// Test792a7q1c_RunQuery_ASmallResultIsReturnedWhole verifies 792a7q1c (A small result is returned whole): with
// matches that total fewer than the limit, every match is returned.
func Test792a7q1c_RunQuery_ASmallResultIsReturnedWhole(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase       = "the phrase"
		matchedCount = 7
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = world.cosineRamp(matchedCount, 0.50, 0.01)

	payload := mq3Run(t, world, cli.QueryArgs{}).payload

	g.Expect(payload.Items).To(HaveLen(matchedCount), "every match is returned")

	got := payload.paths()
	slices.Sort(got)

	want := make([]string, 0, matchedCount)

	for index := range matchedCount {
		want = append(want, mq1ChunkPath(mq1RampAnchor(index)))
	}

	g.Expect(got).To(Equal(want))
}

// Test792a7q1d_RunQuery_TriggerHitsDoNotUseTheLimit verifies 792a7q1d (Trigger hits do not use the limit): of a
// message that hits one runbook and several matched notes, --text with --limit 1 lists the hit first, then one
// matched item, and the hit is not counted against the limit.
func Test792a7q1d_RunQuery_TriggerHitsDoNotUseTheLimit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "the phrase"
		runbookName = "1.drive-ask"
		message     = "/please fix the flaky login test"
		oneItem     = 1
	)

	world := newMq1World(mq1SmallDims, phrase)

	// The runbook's text is not about the phrase: only the message brings it in.
	world.plant(mq1Note{name: runbookName, content: mq3RunbookContent(),
		situation: world.cosines(map[int]float64{0: 0}), body: world.cosines(map[int]float64{0: 0})})

	for index, cosine := range []float64{0.9, 0.8, 0.7} {
		vec := world.cosines(map[int]float64{0: cosine})
		world.plant(mq1Note{name: mq3NoteName(index + 1), content: mq1FactContent(mq3NoteName(index + 1)),
			situation: vec, body: vec})
	}

	payload := mq3Run(t, world, cli.QueryArgs{Text: message, Limit: oneItem}).payload

	g.Expect(payload.paths()).To(Equal([]string{runbookName + ".md", mq3NoteName(1) + ".md"}),
		"the hit first, then the one best matched note")
	g.Expect(payload.withProvenance("trigger")).To(HaveLen(1), "one hit")
	g.Expect(payload.withProvenance("direct")).To(HaveLen(1), "one matched item")
	g.Expect(payload.Budget.Limit).To(Equal(oneItem), "the limit stays 1")
}

// Test792a7q1e_RunQuery_TheRecentChunksSurviveAFullSetOfMatches verifies 792a7q1e (The recent chunks survive a
// full set of matches): with matches that alone reach the limit and a chunk index with recent chunks, --limit 2
// holds two matched items followed by the recent chunks.
func Test792a7q1e_RunQuery_TheRecentChunksSurviveAFullSetOfMatches(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase     = "the phrase"
		limit      = 2
		recentName = "section-recent"
		olderName  = "section-recent-older"
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = append(world.cosineRamp(4, 0.50, 0.10),
		mq3ChunkRecord(world, olderName, 0, mq3DaysAgo(2), "an older chunk, not about the phrase"),
		mq3ChunkRecord(world, recentName, 0, mq3DaysAgo(1), "a newer chunk, not about the phrase"))

	payload := mq3Run(t, world, cli.QueryArgs{Limit: limit}).payload

	g.Expect(payload.paths()).To(Equal([]string{
		mq1ChunkPath(mq1RampAnchor(3)), mq1ChunkPath(mq1RampAnchor(2)),
		mq1ChunkPath(recentName), mq1ChunkPath(olderName),
	}), "two matched items, then the recent chunks")
	g.Expect(payload.withProvenance("direct")).To(HaveLen(limit))
	g.Expect(payload.withProvenance("recent")).To(HaveLen(limit))
}

// Test792a7r1a_RunQuery_LaterChunksBecomeSnippets verifies 792a7r1a (Later chunks become snippets): of three
// chunk items, a content budget of 1 leaves the first chunk its content, the other two carry snippets, and
// chunks_snippeted is 2.
func Test792a7r1a_RunQuery_LaterChunksBecomeSnippets(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase  = "the phrase"
		budget  = 1
		snipped = 2
	)

	// Each text has runs of blank space, so a snippet (whitespace collapsed) differs from the full content.
	texts := []string{
		"third  chunk\n\n   with   spaces",
		"second  chunk\n\n   with   spaces",
		"first  chunk\n\n   with   spaces",
	}

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = mq3RankedChunks(world, texts)

	payload := mq3Run(t, world, cli.QueryArgs{ContentBudget: budget}).payload

	g.Expect(payload.Items).To(HaveLen(len(texts)))

	if len(payload.Items) != len(texts) {
		return
	}

	g.Expect(payload.Items[0].Content).To(Equal(texts[2]), "the first chunk keeps its content")
	g.Expect(payload.Items[1].Content).To(Equal("second chunk with spaces"), "the second chunk is a snippet")
	g.Expect(payload.Items[2].Content).To(Equal("third chunk with spaces"), "the third chunk is a snippet")
	g.Expect(payload.Budget.ChunksSnippeted).To(Equal(snipped))
	g.Expect(payload.Budget.ItemsWithFullContent).To(Equal(budget))
	g.Expect(payload.Budget.ContentBudget).To(Equal(budget))
}

// Test792a7r1b_RunQuery_ALongChunkIsCutWithAnEllipsis verifies 792a7r1b (A long chunk is cut with an ellipsis):
// of a payload whose second chunk item is 300 characters long, a content budget of 1 gives the second chunk a
// content of 160 characters ending in …
func Test792a7r1b_RunQuery_ALongChunkIsCutWithAnEllipsis(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "the phrase"
		budget      = 1
		longLength  = 300
		snippetSize = 160
	)

	long := mq3WordsOfLength(longLength)
	g.Expect(long).To(HaveLen(longLength), "the second chunk is 300 characters long")

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = mq3RankedChunks(world, []string{"the lowest ranked chunk", long, "the highest ranked chunk"})

	payload := mq3Run(t, world, cli.QueryArgs{ContentBudget: budget}).payload

	g.Expect(payload.Items).To(HaveLen(3))

	if len(payload.Items) != 3 {
		return
	}

	second := payload.Items[1].Content
	g.Expect([]rune(second)).To(HaveLen(snippetSize), "160 characters")
	g.Expect(second).To(HaveSuffix("…"))
	g.Expect(second).To(Equal(long[:snippetSize-1]+"…"), "the first 159 characters, then …")
	g.Expect(payload.Items[0].Content).To(Equal("the highest ranked chunk"), "the first chunk keeps its content")
}

// Test792a7r1c_RunQuery_NotesAreNeverReduced verifies 792a7r1c (Notes are never reduced): of a payload with notes
// and chunks beyond the budget, a content budget of 1 leaves every note its full content. The payload lists the
// notes before the chunks, so a budget of 1 reaches the second note only if notes were counted as chunks.
func Test792a7r1c_RunQuery_NotesAreNeverReduced(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase     = "the phrase"
		firstNote  = "1.long-note"
		secondNote = "2.long-note"
		budget     = 1
		secondStep = 0.05
	)

	// A body of two paragraphs, each far over 160 characters: a snippet would be one line of 160.
	body := mq3WordsOfLength(250) + "\n\n" + mq3WordsOfLength(250)

	world := newMq1World(mq1SmallDims, phrase)

	for index, name := range []string{firstNote, secondNote} {
		vec := world.cosines(map[int]float64{0: mq1Strong - secondStep*float64(index)})
		world.plant(mq1Note{name: name, content: mq1FactContent(body), situation: vec, body: vec})
	}

	world.chunks = mq3RankedChunks(world, []string{"chunk three", "chunk two", "chunk one"})

	payload := mq3Run(t, world, cli.QueryArgs{ContentBudget: budget}).payload

	for _, name := range []string{firstNote, secondNote} {
		note, found := payload.find(name + ".md")
		g.Expect(found).To(BeTrue(), name)
		g.Expect(note.Content).To(ContainSubstring(body), name+" keeps its full content")
		g.Expect(note.Content).NotTo(ContainSubstring("…"), name)
	}

	g.Expect(payload.Budget.ChunksSnippeted).To(Equal(2), "the chunks beyond the budget are the ones reduced")
}

// Test792a7r1d_RunQuery_ANegativeBudgetMeansNone verifies 792a7r1d (A negative budget means none): of a payload
// with 20 chunk items, --content-budget=-1 leaves every chunk its full content.
func Test792a7r1d_RunQuery_ANegativeBudgetMeansNone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase        = "the phrase"
		chunkCount    = 20
		negativeLimit = -1
	)

	// Every text is over 160 characters with doubled spaces, so a snippet differs from the full content, and the
	// default budget of 15 would reduce the last five.
	texts := make([]string, 0, chunkCount)

	for index := range chunkCount {
		texts = append(texts, strings.ReplaceAll(mq3WordsOfLength(200), " ", "  ")+" "+mq3NoteName(index))
	}

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = mq3RankedChunks(world, texts)

	payload := mq3Run(t, world, cli.QueryArgs{ContentBudget: negativeLimit}).payload

	g.Expect(payload.Items).To(HaveLen(chunkCount))

	got := make([]string, 0, chunkCount)

	for _, item := range payload.Items {
		got = append(got, item.Content)
	}

	slices.Sort(got)

	want := slices.Clone(texts)
	slices.Sort(want)

	g.Expect(got).To(Equal(want), "every chunk keeps its full content")
	g.Expect(payload.Budget.ChunksSnippeted).To(BeZero())
	g.Expect(payload.Budget.ContentBudget).To(Equal(negativeLimit))
}

// Test792a7s1a_RunQuery_ChunksCarryNoContent verifies 792a7s1a (Chunks carry no content): of a vault with a
// matching chunk, --lazy-chunks gives the chunk item a path and a score and no content, and the budget says
// lazy_chunks: true.
func Test792a7s1a_RunQuery_ChunksCarryNoContent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase = "the phrase"
		anchor = "section-a"
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = []chunk.Record{mq3ChunkRecord(world, anchor, mq1Strong, mq3DaysAgo(0), "the evidence of the chunk")}

	result := mq3Run(t, world, cli.QueryArgs{LazyChunks: true})

	g.Expect(result.payload.Items).To(HaveLen(1))

	if len(result.payload.Items) != 1 {
		return
	}

	item := result.payload.Items[0]
	g.Expect(item.Path).To(Equal(mq1ChunkPath(anchor)))
	g.Expect(item.Score).To(BeNumerically("~", mq1Strong, mq1Tolerance))
	g.Expect(item.Content).To(BeEmpty(), "no content")
	g.Expect(result.stdout).NotTo(ContainSubstring("the evidence of the chunk"))
	g.Expect(result.payload.Budget.LazyChunks).To(HaveValue(BeTrue()))
}

// Test792a7s1b_RunQuery_NotesKeepTheirContent verifies 792a7s1b (Notes keep their content): of a vault with a
// matching note, --lazy-chunks leaves the note item its full content (a matching chunk beside it is cleared).
func Test792a7s1b_RunQuery_NotesKeepTheirContent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase   = "the phrase"
		noteName = "1.matching-note"
		anchor   = "section-a"
	)

	world := newMq1World(mq1SmallDims, phrase)
	vec := world.cosines(map[int]float64{0: mq1Strong})
	content := mq1FactContent("the full content of the note")
	world.plant(mq1Note{name: noteName, content: content, situation: vec, body: vec})
	world.chunks = []chunk.Record{mq3ChunkRecord(world, anchor, mq1Strong, mq3DaysAgo(0), "the evidence of the chunk")}

	payload := mq3Run(t, world, cli.QueryArgs{LazyChunks: true}).payload

	note, noteFound := payload.find(noteName + ".md")
	g.Expect(noteFound).To(BeTrue())
	g.Expect(note.Content).To(ContainSubstring("the full content of the note"), "the note keeps its content")

	chunkItem, chunkFound := payload.find(mq1ChunkPath(anchor))
	g.Expect(chunkFound).To(BeTrue())
	g.Expect(chunkItem.Content).To(BeEmpty(), "the chunk beside it carries none")
}

// Test792a7s1c_Targets_Query_TheEnvironmentCanTurnItOn verifies 792a7s1c (The environment can turn it on): with
// ENGRAM_LAZY_CHUNKS=true, engram query with a phrase that matches a chunk gives the chunk item no content. Not
// parallel: it needs t.Setenv.
func Test792a7s1c_Targets_Query_TheEnvironmentCanTurnItOn(t *testing.T) {
	g := NewWithT(t)

	const (
		phrase = "the phrase"
		anchor = "section-a"
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = []chunk.Record{mq3ChunkRecord(world, anchor, mq1Strong, mq3DaysAgo(0), "the evidence of the chunk")}

	t.Setenv(mq3LazyChunksEnv, "true")

	result := mq3TargetsQuery(t, world)

	g.Expect(result.payload.Items).To(HaveLen(1))

	if len(result.payload.Items) != 1 {
		return
	}

	g.Expect(result.payload.Items[0].Path).To(Equal(mq1ChunkPath(anchor)))
	g.Expect(result.payload.Items[0].Content).To(BeEmpty(), "the chunk item has no content")
	g.Expect(result.payload.Budget.LazyChunks).To(HaveValue(BeTrue()))
}

// Test792a7s1d_RunQuery_WithoutTheFlagTheFieldIsAbsent verifies 792a7s1d (Without the flag the field is absent):
// of a vault with a matching chunk, engram query with no --lazy-chunks gives the chunk item its content and the
// budget has no lazy_chunks.
func Test792a7s1d_RunQuery_WithoutTheFlagTheFieldIsAbsent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase = "the phrase"
		anchor = "section-a"
		text   = "the evidence of the chunk"
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = []chunk.Record{mq3ChunkRecord(world, anchor, mq1Strong, mq3DaysAgo(0), text)}

	result := mq3Run(t, world, cli.QueryArgs{})

	g.Expect(result.payload.Items).To(HaveLen(1))

	if len(result.payload.Items) != 1 {
		return
	}

	g.Expect(result.payload.Items[0].Content).To(Equal(text), "the chunk item has its content")
	g.Expect(result.payload.Budget.LazyChunks).To(BeNil())
	g.Expect(result.stdout).NotTo(ContainSubstring("lazy_chunks"), "the payload has no lazy_chunks field")
}

// unexported constants.
const (
	mq3FixedArgs     = 6
	mq3LazyChunksEnv = "ENGRAM_LAZY_CHUNKS"
	mq3PhraseText    = "the phrase"
	mq3RecentFillEnv = "ENGRAM_RECENT_FILL"
)

// mq3Budget is the budget block of the query payload.
type mq3Budget struct {
	Limit                int   `yaml:"limit"`
	ContentBudget        int   `yaml:"content_budget"`
	ChunksSnippeted      int   `yaml:"chunks_snippeted"`
	ItemsWithFullContent int   `yaml:"items_with_full_content"`
	LazyChunks           *bool `yaml:"lazy_chunks"`
}

// mq3Item is one item of the query payload.
type mq3Item struct {
	Path        string   `yaml:"path"`
	Kind        string   `yaml:"kind"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	ClusterID   *int     `yaml:"cluster_id"`
	Content     string   `yaml:"content"`
}

// mq3Payload is the part of the query payload that part 3 reads.
type mq3Payload struct {
	Items    []mq3Item `yaml:"items"`
	Clusters []struct {
		Members []struct {
			Path string `yaml:"path"`
		} `yaml:"members"`
	} `yaml:"clusters"`
	Budget mq3Budget `yaml:"budget"`
}

// find returns the item with the given path.
func (p mq3Payload) find(path string) (mq3Item, bool) {
	for _, item := range p.Items {
		if item.Path == path {
			return item, true
		}
	}

	return mq3Item{}, false
}

// memberPaths returns the paths of every cluster member.
func (p mq3Payload) memberPaths() []string {
	paths := make([]string, 0, len(p.Items))

	for _, group := range p.Clusters {
		for _, member := range group.Members {
			paths = append(paths, member.Path)
		}
	}

	return paths
}

// paths returns the paths of the items, in item order.
func (p mq3Payload) paths() []string {
	paths := make([]string, 0, len(p.Items))

	for _, item := range p.Items {
		paths = append(paths, item.Path)
	}

	return paths
}

// withProvenance returns the items that carry the given provenance.
func (p mq3Payload) withProvenance(role string) []mq3Item {
	found := make([]mq3Item, 0, len(p.Items))

	for _, item := range p.Items {
		if slices.Contains(item.Provenances, role) {
			found = append(found, item)
		}
	}

	return found
}

// mq3Result is what one run of the query printed, with the decoded payload.
type mq3Result struct {
	payload mq3Payload
	stdout  string
}

// mq3ChunkRecord returns a chunk record of the world's source whose vector has the given cosine to the first
// phrase, first indexed at the given time (the zero time for a chunk with none).
func mq3ChunkRecord(
	world *mq1World, anchor string, cosine float64, indexedAt time.Time, text string,
) chunk.Record {
	return chunk.Record{
		Source:      mq1Source,
		Anchor:      anchor,
		ContentHash: chunk.HashText(text),
		Text:        text,
		Vector:      world.cosines(map[int]float64{0: cosine}),
		IngestedAt:  indexedAt,
	}
}

// mq3DaysAgo returns the time the given number of whole days before the fixed clock mq1Now.
func mq3DaysAgo(days int) time.Time { return mq1Now.AddDate(0, 0, -days) }

// mq3Decode decodes a query payload, failing the test when it is not YAML.
func mq3Decode(t *testing.T, stdout string) mq3Result {
	t.Helper()

	var payload mq3Payload

	err := yaml.Unmarshal([]byte(stdout), &payload)
	if err != nil {
		t.Fatalf("decoding the payload: %v", err)
	}

	return mq3Result{payload: payload, stdout: stdout}
}

// mq3NoteName names the index-th note of a scenario.
func mq3NoteName(index int) string { return fmt.Sprintf("%d.note", index+1) }

// mq3RankedChunks returns one chunk per text, all matched, whose rank is the reverse of the order of the texts:
// the last text is the highest ranked chunk. All are first indexed at the fixed clock.
func mq3RankedChunks(world *mq1World, texts []string) []chunk.Record {
	const (
		baseCosine = 0.50
		step       = 0.01
	)

	records := make([]chunk.Record, 0, len(texts))

	for index, text := range texts {
		records = append(records,
			mq3ChunkRecord(world, mq1RampAnchor(index), baseCosine+step*float64(index), mq3DaysAgo(0), text))
	}

	return records
}

// mq3Run queries the world's phrases with the given arguments over the in-memory world.
func mq3Run(t *testing.T, world *mq1World, args cli.QueryArgs) mq3Result {
	t.Helper()

	if args.Phrases == nil {
		args.Phrases = world.phrases
	}

	result := world.query(t, args)
	if result.err != nil {
		t.Fatalf("query failed: %v", result.err)
	}

	return mq3Decode(t, result.stdout)
}

// mq3RunbookContent is a runbook note that a message starting with /please hits.
func mq3RunbookContent() string {
	return "---\ntype: runbook\nsituation: zebra migration planning at the savanna\ndone_when: done\n" +
		"triggers:\n    - /please\n---\n\n1. Do it\n"
}

// mq3TargetsQuery runs `engram query --phrase <the world's phrase>` through cli.Targets over a temporary vault
// and a temporary chunk directory that holds the world's chunks, with the given extra arguments, and decodes the
// payload. Parent and home lookups are stubbed, so only the extra arguments and the process environment (for the
// flags that read an environment variable) reach the flag parser.
func mq3TargetsQuery(t *testing.T, world *mq1World, extra ...string) mq3Result {
	t.Helper()

	encoded, err := chunk.EncodeRecords(world.chunks)
	if err != nil {
		t.Fatalf("encoding the chunk index: %v", err)
	}

	vault := t.TempDir()
	chunksDir := t.TempDir()

	err = os.WriteFile(filepath.Join(chunksDir, "index.jsonl"), encoded, 0o600)
	if err != nil {
		t.Fatalf("writing the chunk index: %v", err)
	}

	args := make([]string, 0, mq3FixedArgs+2*len(world.phrases)+len(extra))
	args = append(args, "engram", "query", "--vault", vault, "--chunks-dir", chunksDir)

	for _, phrase := range world.phrases {
		args = append(args, "--phrase", phrase)
	}

	args = append(args, extra...)

	stdout, stderr := executeCapturingBoth(t, args, func(deps *cli.Deps) {
		deps.Embed = mq1PhraseEmbedder{dims: world.dims, vectors: world.vectors}
		deps.Getenv = func(string) string { return "" }
		deps.Now = world.clock
	})

	if stderr != "" {
		t.Fatalf("engram query wrote to stderr: %s", stderr)
	}

	return mq3Decode(t, stdout)
}

// mq3UnmatchedWorld returns a world of three chunks none of which matches the phrase, first indexed one, two and
// three days ago, with the path of the newest.
func mq3UnmatchedWorld() (*mq1World, string) {
	world := newMq1World(mq1SmallDims, mq3PhraseText)
	world.chunks = []chunk.Record{
		mq3ChunkRecord(world, "section-oldest", 0, mq3DaysAgo(3), "the oldest chunk, not about the phrase"),
		mq3ChunkRecord(world, "section-newest", 0, mq3DaysAgo(1), "the newest chunk, not about the phrase"),
		mq3ChunkRecord(world, "section-middle", 0, mq3DaysAgo(2), "the middle chunk, not about the phrase"),
	}

	return world, mq1ChunkPath("section-newest")
}

// mq3WordsOfLength returns a text of single-spaced words that is exactly the given number of characters long and
// ends in a letter, so collapsing its whitespace leaves its length unchanged.
func mq3WordsOfLength(length int) string {
	var builder strings.Builder

	for index := 0; builder.Len() < length; index++ {
		fmt.Fprintf(&builder, "w%05d ", index)
	}

	runes := []rune(builder.String())[:length]
	runes[length-1] = 'z'

	return string(runes)
}
