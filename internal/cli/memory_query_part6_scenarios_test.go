package cli_test

// Scenario tests for the Memory Query product capability, part 6 (861c2zq): searching chunks alone (792a7zf), CRLF
// notes (792a7zg) and a query that changes nothing (792a7zh). Spec: docs/specs/product/memory-query.md.
//
// 792a7zf is about what the user sees at the command line, so it runs `engram query-chunks` through cli.Targets over
// a cli.Deps whose EdgeFS is the in-memory fake that refuses every write (mq4Machine, memory_query_part4); 792a7zh
// runs `engram query` the same way, so a write by a read-only command would show. 792a7zg runs cli.RunQuery over the
// in-memory world of memory_query_part1_scenarios_test.go (mq1World, a phrase-keyed embedder, so every cosine is a
// number the test chose), except 792a7zg1c, which is about the file and so runs the command over the machine. A CRLF
// note is a note planted as an LF note and then rewritten with \r\n line endings (mq6ConvertAll), its sidecar hash
// with it. The helpers carry the mq6 prefix.

import (
	"bytes"
	"context"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a7zf1a_Targets_QueryChunks_ChunksAreRankedByCosine verifies 792a7zf1a (Chunks are ranked by cosine): over a
// chunk index holding a baking chunk and a networking chunk (the networking chunk first in the file), engram
// query-chunks --phrase "sourdough" lists the baking chunk first, each item with source, anchor, score and content,
// and the budget says total_chunks: 2 and returned: 2.
func Test792a7zf1a_Targets_QueryChunks_ChunksAreRankedByCosine(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "sourdough"
		bakingText  = "Sourdough needs a long, cold proof before it goes into a hot oven."
		routingText = "A router forwards packets between two networks."
		bakingCos   = 0.9
		routingCos  = 0.1
	)

	// Given a chunk index holding a baking chunk and a networking chunk.
	world := newMq1World(mq6Dims, phrase)
	machine := mq6ChunkMachine(t,
		mq6Chunk("routing", routingText, world.cosines(map[int]float64{0: routingCos})),
		mq6Chunk("baking", bakingText, world.cosines(map[int]float64{0: bakingCos})))

	// When the user runs engram query-chunks --phrase "sourdough".
	run := mq6RunCommand(t, machine, mq6NewEmbedder(world), "query-chunks", "--chunks-dir", mq4ChunksDir,
		"--phrase", phrase)
	payload := mq6DecodeChunks(t, run.stdout)

	// Then items lists the baking chunk first, each with source, anchor, score and content,
	g.Expect(run.stderr).To(BeEmpty())
	g.Expect(run.exits).To(BeEmpty())
	g.Expect(payload.Version).To(Equal(1))
	g.Expect(payload.Space).To(Equal("chunks"))
	g.Expect(payload.Items).To(HaveLen(2))

	if len(payload.Items) != 2 {
		return
	}

	first, second := payload.Items[0], payload.Items[1]
	g.Expect(first.Anchor).To(Equal("baking"))
	g.Expect(first.Source).To(Equal(mq6Source))
	g.Expect(first.Content).To(Equal(bakingText + "\n"))
	g.Expect(first.Score).NotTo(BeNil())
	g.Expect(second.Anchor).To(Equal("routing"))
	g.Expect(second.Source).To(Equal(mq6Source))
	g.Expect(second.Content).To(Equal(routingText + "\n"))
	g.Expect(second.Score).NotTo(BeNil())

	if first.Score == nil || second.Score == nil {
		return
	}

	g.Expect(*first.Score).To(BeNumerically("~", bakingCos, mq1Tolerance))
	g.Expect(*second.Score).To(BeNumerically("~", routingCos, mq1Tolerance))

	// and the budget says total_chunks: 2 and returned: 2.
	g.Expect(payload.Budget.TotalChunks).To(HaveValue(Equal(2)))
	g.Expect(payload.Budget.Returned).To(HaveValue(Equal(2)))
}

// Test792a7zf1b_Targets_QueryChunks_TheLimitIsHonoured verifies 792a7zf1b (The limit is honoured): over a chunk index
// of five chunks (the best last in the file), engram query-chunks --phrase p --limit 2 returns the two best chunks
// and the budget says returned: 2 (total_chunks stays 5); without --limit the same index returns all five.
func Test792a7zf1b_Targets_QueryChunks_TheLimitIsHonoured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase   = "p"
		chunkSum = 5
		limit    = 2
		baseCos  = 0.5
		stepCos  = 0.1
	)

	// Given a chunk index of five chunks, in ascending order of cosine.
	world := newMq1World(mq6Dims, phrase)
	records := make([]chunk.Record, 0, chunkSum)

	for index := range chunkSum {
		cosine := baseCos + stepCos*float64(index)
		records = append(records, mq6Chunk(mq1RampAnchor(index), "chunk text "+mq1RampAnchor(index),
			world.cosines(map[int]float64{0: cosine})))
	}

	// When the user runs engram query-chunks --phrase p --limit 2.
	limited := mq6RunCommand(t, mq6ChunkMachine(t, records...), mq6NewEmbedder(world), "query-chunks",
		"--chunks-dir", mq4ChunksDir, "--phrase", phrase, "--limit", "2")
	payload := mq6DecodeChunks(t, limited.stdout)

	// Then two items are returned, the best two, and the budget says returned: 2.
	g.Expect(limited.stderr).To(BeEmpty())
	g.Expect(payload.Items).To(HaveLen(limit))
	g.Expect(mq6Anchors(payload.Items)).To(Equal([]string{mq1RampAnchor(chunkSum - 1), mq1RampAnchor(chunkSum - limit)}))
	g.Expect(payload.Budget.Returned).To(HaveValue(Equal(limit)))
	g.Expect(payload.Budget.TotalChunks).To(HaveValue(Equal(chunkSum)))

	// The control: without --limit the same index returns all five.
	unlimited := mq6RunCommand(t, mq6ChunkMachine(t, records...), mq6NewEmbedder(world), "query-chunks",
		"--chunks-dir", mq4ChunksDir, "--phrase", phrase)
	g.Expect(mq6DecodeChunks(t, unlimited.stdout).Items).To(HaveLen(chunkSum))
}

// Test792a7zf1c_Targets_QueryChunks_ManyChunksAreClustered verifies 792a7zf1c (Many chunks are clustered): over a
// chunk index of twelve chunks about two topics, engram query-chunks with a phrase prints clusters, each with an id,
// a size and members (source, anchor and score), that together hold the twelve returned chunks, and each cluster holds
// the chunks of one topic.
func Test792a7zf1c_Targets_QueryChunks_ManyChunksAreClustered(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase       = "sourdough"
		topicSize    = 6
		wantClusters = 2
		spread       = 0.02
	)

	// Given a chunk index of twelve chunks about two topics, interleaved in the file.
	records := make([]chunk.Record, 0, 2*topicSize)

	for index := range topicSize {
		deviation := spread * float32(index)

		records = append(records,
			mq6Chunk("baking-"+mq1RampAnchor(index), "baking chunk "+mq1RampAnchor(index), []float32{1, deviation, 0}),
			mq6Chunk("routing-"+mq1RampAnchor(index), "routing chunk "+mq1RampAnchor(index), []float32{0, 1, deviation}))
	}

	world := newMq1World(mq6Dims, phrase)

	// When the user runs engram query-chunks with a phrase.
	run := mq6RunCommand(t, mq6ChunkMachine(t, records...), mq6NewEmbedder(world), "query-chunks",
		"--chunks-dir", mq4ChunksDir, "--phrase", phrase)
	payload := mq6DecodeChunks(t, run.stdout)

	// Then the payload has clusters, each with an id, a size and members.
	g.Expect(run.stderr).To(BeEmpty())
	g.Expect(payload.Items).To(HaveLen(2 * topicSize))
	g.Expect(payload.Clusters).To(HaveLen(wantClusters))

	ids := make([]int, 0, len(payload.Clusters))
	delivered := 0

	for _, group := range payload.Clusters {
		g.Expect(group.ID).NotTo(BeNil(), "a cluster has an id")
		g.Expect(group.Size).To(Equal(len(group.Members)), "its size counts its members")
		g.Expect(group.Members).NotTo(BeEmpty())

		if group.ID != nil {
			ids = append(ids, *group.ID)
		}

		topics := map[string]bool{}

		for _, member := range group.Members {
			g.Expect(member.Source).To(Equal(mq6Source))
			g.Expect(member.Anchor).NotTo(BeEmpty())
			g.Expect(member.Score).NotTo(BeNil(), "a member has a score")

			topic, _, _ := strings.Cut(member.Anchor, "-")
			topics[topic] = true
		}

		g.Expect(topics).To(HaveLen(1), "a cluster holds one topic")

		delivered += group.Size
	}

	g.Expect(ids).To(ConsistOf(0, 1), "the ids tell the clusters apart")
	g.Expect(delivered).To(Equal(2*topicSize), "every returned chunk is in a cluster")
}

// Test792a7zf1d_Targets_QueryChunks_APhraseIsRequired verifies 792a7zf1d (A phrase is required): over any chunk
// index, engram query-chunks with no phrase fails with "query chunks: at least one --phrase is required", exits
// non-zero, prints no payload and embeds nothing.
func Test792a7zf1d_Targets_QueryChunks_APhraseIsRequired(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a chunk index.
	world := newMq1World(mq6Dims)
	embedder := mq6NewEmbedder(world)
	machine := mq6ChunkMachine(t, mq6Chunk("baking", "baking text", []float32{1, 0, 0}))

	// When the user runs engram query-chunks with no phrase.
	run := mq6RunCommand(t, machine, embedder, "query-chunks", "--chunks-dir", mq4ChunksDir)

	// Then it fails with "query chunks: at least one --phrase is required".
	g.Expect(run.stderr).To(Equal("query chunks: at least one --phrase is required\n"))
	g.Expect(run.exits).To(Equal([]int{1}))
	g.Expect(run.stdout).To(BeEmpty())
	g.Expect(embedder.calls.Load()).To(BeZero())
}

// Test792a7zf1e_Targets_QueryChunks_AnEmptyIndexIsAnEmptyPayload verifies 792a7zf1e (An empty index is an empty
// payload): with no chunk index directory, engram query-chunks --phrase p succeeds, prints a payload with no items
// and total_chunks: 0, and the phrase is never embedded.
func Test792a7zf1e_Targets_QueryChunks_AnEmptyIndexIsAnEmptyPayload(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a chunk index directory that does not exist; the world's embedder refuses the phrase, as it declares none.
	world := newMq1World(mq6Dims)
	embedder := mq6NewEmbedder(world)
	machine := mq4NewMachine()
	delete(machine.dirs, mq4ChunksDir)

	// When the user runs engram query-chunks --phrase p.
	run := mq6RunCommand(t, machine, embedder, "query-chunks", "--chunks-dir", mq4ChunksDir, "--phrase", "p")
	payload := mq6DecodeChunks(t, run.stdout)

	// Then the payload has no items, and the budget says total_chunks: 0.
	g.Expect(run.stderr).To(BeEmpty())
	g.Expect(run.exits).To(BeEmpty())
	g.Expect(payload.Items).To(BeEmpty())
	g.Expect(payload.Clusters).To(BeEmpty())
	g.Expect(payload.Budget.TotalChunks).To(HaveValue(BeZero()))
	g.Expect(payload.Budget.Returned).To(HaveValue(BeZero()))
	g.Expect(run.stdout).To(ContainSubstring("total_chunks: 0\n"))
	g.Expect(embedder.calls.Load()).To(BeZero(), "the phrase is not embedded")
}

// Test792a7zg1a_RunQuery_TheProjectFilterReadsACRLFNote verifies 792a7zg1a (The project filter reads a CRLF note):
// of CRLF notes with project: alpha and project: beta, engram query --project alpha returns the alpha notes and not
// the beta notes, whether the note is a matched note or an explore-half note; without the filter both are returned.
func Test792a7zg1a_RunQuery_TheProjectFilterReadsACRLFNote(t *testing.T) {
	t.Parallel()

	t.Run("a matched note", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		const alphaPath = mq1Vault + "/1.alpha-note.md"

		// Given a note with CRLF line endings and project: alpha, and one with project: beta.
		world := newMq1World(mq2Dims, mq2GoPhrase)
		mq2PlantNote(world, "1.alpha-note", world.cosines(map[int]float64{0: mq4Closer}), "project: alpha\n")
		mq2PlantNote(world, "2.beta-note", world.cosines(map[int]float64{0: mq4Close}), "project: beta\n")
		mq6ConvertAll(t, world)
		g.Expect(string(world.files.files[alphaPath])).To(ContainSubstring("\r\nproject: alpha\r\n"))

		// When the user runs engram query --project alpha with a phrase that matches it.
		result := mq4MustQuery(t, world, cli.QueryArgs{Project: "alpha"})

		// Then the note is returned, and the note of the other project is not.
		g.Expect(result.payload.paths()).To(Equal([]string{"1.alpha-note.md"}))
		g.Expect(result.stdout).NotTo(ContainSubstring("beta-note"))

		// The control: without the filter both CRLF notes are matched.
		unfiltered := mq4MustQuery(t, world, cli.QueryArgs{})
		g.Expect(unfiltered.payload.paths()).To(Equal([]string{"1.alpha-note.md", "2.beta-note.md"}))
	})

	t.Run("an explore note", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		// Given two matched CRLF notes with project: alpha, and CRLF explore-half notes with project: beta and alpha.
		world := newMq1World(mq2Dims, mq2GoPhrase)
		mq2PlantNote(world, "1.alpha-top", world.cosines(map[int]float64{0: mq4Closer}), "project: alpha\n")
		mq2PlantNote(world, "2.alpha-next", world.cosines(map[int]float64{0: mq4Close}), "project: alpha\n")
		mq2PlantNote(world, "4.beta-explore", mq2Spec{cosine: mq4ExploreNear}.vec(), mq2TagLine()+"project: beta\n")
		mq2PlantNote(world, "5.alpha-explore", mq2Spec{cosine: mq4ExploreFar}.vec(), mq2TagLine()+"project: alpha\n")
		mq2WriteCentroids(t, world)
		mq6ConvertAll(t, world)

		control := mq4MustQuery(t, world, cli.QueryArgs{})
		g.Expect(control.payload.pathsWith("explore")).To(Equal([]string{"4.beta-explore.md", "5.alpha-explore.md"}))

		// When the user runs engram query --project alpha.
		result := mq4MustQuery(t, world, cli.QueryArgs{Project: "alpha"})

		// Then the alpha explore note is returned and the beta one is not.
		g.Expect(result.payload.pathsWith("explore")).To(Equal([]string{"5.alpha-explore.md"}))
		g.Expect(result.stdout).NotTo(ContainSubstring("beta"))
	})
}

// Test792a7zg1b_RunQuery_ACRLFRunbooksTriggerMatches verifies 792a7zg1b (A CRLF runbook's trigger matches): a runbook
// whose lines end in \r\n and whose triggers include "curate" is returned by engram query --text "curate" as a
// trigger hit (provenance trigger, score 0) with kind runbook; a message that holds no trigger returns nothing.
func Test792a7zg1b_RunQuery_ACRLFRunbooksTriggerMatches(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		trigger     = "curate"
		runbookName = "1.curate-runbook"
	)

	// Given a runbook whose lines end in \r\n and whose triggers include "curate".
	world := newMq1World(mq1SmallDims)
	mq5Plant(world, runbookName, mq5Content(mq5Note{kind: "runbook", triggers: []string{trigger, "/curate"}}),
		map[int]float64{})
	mq6ConvertAll(t, world)
	g.Expect(string(world.files.files[mq1Vault+"/"+runbookName+".md"])).To(ContainSubstring("type: runbook\r\n"))

	// When the user runs engram query --text "curate".
	outcome := mq5Run(t, world, cli.QueryArgs{Text: trigger}, mq5Hooks{})

	// Then it is returned as a trigger hit with kind runbook.
	g.Expect(outcome.payload.Items).To(HaveLen(1))

	if len(outcome.payload.Items) != 1 {
		return
	}

	item := outcome.payload.Items[0]
	g.Expect(item.Path).To(Equal(runbookName + ".md"))
	g.Expect(item.Kind).To(Equal("runbook"))
	g.Expect(item.Provenances).To(Equal([]string{"trigger"}))
	g.Expect(item.Score).To(BeZero())

	// The control: a message with no trigger in it hits nothing.
	missed := mq5Run(t, world, cli.QueryArgs{Text: "nothing relevant here"}, mq5Hooks{})
	g.Expect(missed.payload.Items).To(BeEmpty())
}

// Test792a7zg1c_Targets_Query_ACRLFNoteIsNotRewritten verifies 792a7zg1c (A CRLF note is not rewritten): after
// engram query with a phrase that matches a CRLF note, the note's file is byte-identical, the query asked its file
// system for no write, and the note is among the items.
func Test792a7zg1c_Targets_Query_ACRLFNoteIsNotRewritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const notePath = mq1Vault + "/1.crlf-note.md"

	// Given a note with CRLF line endings.
	world := newMq1World(mq2Dims, mq2GoPhrase)
	mq2PlantNote(world, "1.crlf-note", world.cosines(map[int]float64{0: mq4Closer}), "project: alpha\n")
	mq6ConvertAll(t, world)

	machine := mq6VaultMachine(t, world)
	before := slices.Clone(machine.files[notePath])
	g.Expect(string(before)).To(ContainSubstring("\r\n"))

	// When the user runs engram query with a phrase that matches it.
	run := mq6RunQuery(t, machine, world)

	// Then the note's file is byte-identical afterwards, and no write was asked for.
	g.Expect(run.stderr).To(BeEmpty())
	g.Expect(run.payload.paths()).To(Equal([]string{"1.crlf-note.md"}))
	// mq4Machine refuses every write, so the byte comparison cannot fail alone; the empty writes record guards it.
	g.Expect(machine.files[notePath]).To(Equal(before))
	g.Expect(machine.writes).To(BeEmpty())
}

// Test792a7zg1d_RunQuery_TheCreationDateOfACRLFNoteIsRead verifies 792a7zg1d (The creation date of a CRLF note is
// read): a CRLF note whose created line is the bare date 120 days ago, with no last_used in its sidecar, scores its
// cosine scaled by that age, the same as the same note with LF line endings.
func Test792a7zg1d_RunQuery_TheCreationDateOfACRLFNoteIsRead(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase    = "the phrase"
		noteCos   = 0.8
		ageDays   = 120
		fourTimes = 4
		crlfName  = "1.crlf-note"
		lfName    = "2.lf-note"
	)

	// Given a CRLF note whose created line is the bare date 120 days ago and whose sidecar has no last_used, and
	// the same note with LF line endings.
	created := mq1Now.AddDate(0, 0, -ageDays).Format(time.DateOnly)
	content := "---\ntype: fact\nsituation: a situation\ncreated: " + created + "\n---\n\nthe same text\n"
	world := newMq1World(mq1SmallDims, phrase)
	vec := world.cosines(map[int]float64{0: noteCos})
	world.plant(mq1Note{name: crlfName, content: content, situation: vec, body: vec})
	world.plant(mq1Note{name: lfName, content: content, situation: vec, body: vec})
	mq6ConvertNote(t, world, mq1Vault+"/"+crlfName+".md")
	g.Expect(string(world.files.files[mq1Vault+"/"+crlfName+".md"])).To(ContainSubstring("created: " + created + "\r\n"))

	// When the user runs engram query with a phrase that matches it.
	payload := world.mustRun(t)

	// Then its score is scaled by that age, as for the same note with LF line endings.
	crlfItem, crlfFound := payload.find(crlfName + ".md")
	lfItem, lfFound := payload.find(lfName + ".md")
	g.Expect(crlfFound).To(BeTrue())
	g.Expect(lfFound).To(BeTrue())
	g.Expect(float64(lfItem.Score)).To(BeNumerically("~", noteCos/fourTimes, mq1Tolerance), "the LF note is scaled")
	g.Expect(float64(crlfItem.Score)).To(BeNumerically("~", float64(lfItem.Score), mq1Tolerance))
}

// Test792a7zg1e_RunQuery_ACRLFRunbooksRedFlagsArePreviewed verifies 792a7zg1e (A CRLF runbook's red flags are
// previewed): a CRLF runbook whose red flags take more than 1200 bytes, matched by a phrase, has in its content the
// omission marker and the last ten entries, and its whole content is the one of the same runbook with LF line
// endings.
func Test792a7zg1e_RunQuery_ACRLFRunbooksRedFlagsArePreviewed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a runbook whose lines end in \r\n and whose red flags take more than 1200 bytes, and the same runbook
	// with LF line endings.
	entries := mq4RedFlagEntries(mq4OversizedCount)
	lfWorld := newMq1World(mq2Dims, mq2GoPhrase)
	mq4PlantRunbook(lfWorld, mq4RedFlagsLines(entries))

	crlfWorld := newMq1World(mq2Dims, mq2GoPhrase)
	mq4PlantRunbook(crlfWorld, mq4RedFlagsLines(entries))
	mq6ConvertAll(t, crlfWorld)
	g.Expect(string(crlfWorld.files.files[mq1Vault+"/"+mq4RunbookName+".md"])).To(ContainSubstring("red_flags:\r\n"))
	g.Expect(mq4RedFlagsBytes(entries)).To(BeNumerically(">", mq4PreviewBudget))

	// When the user runs engram query with a phrase that matches it.
	crlf := mq4MustQuery(t, crlfWorld, cli.QueryArgs{})
	lf := mq4MustQuery(t, lfWorld, cli.QueryArgs{})

	// Then its content holds the omission marker and the newest entries, as for the same note with LF line endings.
	crlfItem, crlfFound := crlf.payload.find(mq4RunbookName + ".md")
	lfItem, lfFound := lf.payload.find(mq4RunbookName + ".md")
	g.Expect(crlfFound).To(BeTrue())
	g.Expect(lfFound).To(BeTrue())
	g.Expect(crlfItem.Content).To(ContainSubstring("20 EARLIER RED_FLAGS OMITTED"))
	g.Expect(mq4RedFlagsBlock(crlfItem.Content)).To(Equal(mq4WantTruncatedBlock(entries)))
	g.Expect(crlfItem.Content).To(Equal(lfItem.Content))
}

// Test792a7zg1f_RunQuery_ACRLFNoteKeepsItsKind verifies 792a7zg1f (A CRLF note keeps its kind): the item of a fact
// note whose lines end in \r\n, matched by a phrase, has kind fact exactly (no \r in the label), and so has the
// item of a CRLF fact note that rides along or is explored, whose kind is read from the file text.
func Test792a7zg1f_RunQuery_ACRLFNoteKeepsItsKind(t *testing.T) {
	t.Parallel()

	t.Run("a matched note", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		const noteName = "1.crlf-fact"

		// Given a fact note whose lines end in \r\n.
		world := newMq1World(mq2Dims, mq2GoPhrase)
		mq2PlantNote(world, noteName, world.cosines(map[int]float64{0: mq4Closer}), "")
		mq6ConvertAll(t, world)
		g.Expect(string(world.files.files[mq1Vault+"/"+noteName+".md"])).To(ContainSubstring("type: fact\r\n"))

		// When the user runs engram query with a phrase that matches it.
		payload := world.mustRun(t)

		// Then its item has kind fact.
		item, found := payload.find(noteName + ".md")
		g.Expect(found).To(BeTrue())
		g.Expect(item.Kind).To(Equal("fact"))
	})

	t.Run("a note that rides along", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		// Given a CRLF fact note that supersedes a matched note without matching itself.
		world := mq2RideAlongWorld()
		mq2PlantNote(world, "4.newer", mq2Unmatched, mq2SupersedesOlder())
		mq6ConvertAll(t, world)

		// When the user runs engram query with a phrase that matches the superseded note.
		payload := world.mustRun(t)

		// Then the item that rides along has kind fact.
		item, found := payload.find("4.newer.md")
		g.Expect(found).To(BeTrue())
		g.Expect(item.Provenances).To(Equal([]string{"ride_along"}))
		g.Expect(item.Kind).To(Equal("fact"))
	})

	t.Run("an explore note", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		// Given CRLF fact notes of a term, none of them matched.
		world := mq2NewExploreWorld(2, false, []mq2Spec{{name: "3.go-core-a", cosine: 0.80}})
		mq2WriteCentroids(t, world)
		mq6ConvertAll(t, world)

		// When the user runs engram query with a phrase about cooking.
		payload := world.mustRun(t)

		// Then the explore item has kind fact.
		item, found := payload.find("3.go-core-a.md")
		g.Expect(found).To(BeTrue())
		g.Expect(item.Provenances).To(Equal([]string{"explore"}))
		g.Expect(item.Kind).To(Equal("fact"))
	})
}

// Test792a7zg1g_RunQuery_TheListsInACRLFNotesFrontmatterAreRead verifies 792a7zg1g (The lists in a CRLF note's
// frontmatter are read): with every note of the vault in CRLF, the notes tagged vocab/go-tooling are the explore
// notes of that term, as for the same vault with LF line endings, and a note that supersedes a matched note rides
// along after it.
func Test792a7zg1g_RunQuery_TheListsInACRLFNotesFrontmatterAreRead(t *testing.T) {
	t.Parallel()

	t.Run("a tagged note can be an explore note", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		build := func() *mq1World {
			members := []mq2Spec{
				{name: "3.go-core-a", cosine: 0.80},
				{name: "4.go-core-b", cosine: 0.60},
				{name: "5.go-fringe", cosine: 0.50},
			}
			world := mq2NewExploreWorld(2, false, members)
			mq2WriteCentroids(t, world)

			return world
		}

		// Given the notes of a term tagged vocab/go-tooling, in CRLF, and the same vault in LF.
		crlfWorld := build()
		mq6ConvertAll(t, crlfWorld)
		g.Expect(string(crlfWorld.files.files[mq1Vault+"/3.go-core-a.md"])).To(ContainSubstring("- vocab/go-tooling\r\n"))

		// When the user runs engram query with a phrase that matches the vault.
		crlf := mq2MustRun(t, crlfWorld)
		lf := mq2MustRun(t, build())

		// Then the tagged notes are explore notes of the term, the notes the LF vault gives.
		g.Expect(lf.payload.explorePaths()).To(Equal([]string{"3.go-core-a.md", "4.go-core-b.md"}))
		g.Expect(crlf.payload.explorePaths()).To(Equal(lf.payload.explorePaths()))

		for _, item := range crlf.payload.exploreItems() {
			g.Expect(item.SourceTerm).To(Equal(mq2Term))
		}
	})

	t.Run("the superseding note rides along", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		// Given a CRLF note that supersedes a matched note.
		world := mq2RideAlongWorld()
		mq2PlantNote(world, "4.newer", mq2Unmatched, mq2SupersedesOlder())
		mq6ConvertAll(t, world)
		g.Expect(string(world.files.files[mq1Vault+"/4.newer.md"])).To(ContainSubstring("supersedes:\r\n"))

		// When the user runs engram query with a phrase that matches the superseded note.
		result := mq2MustRun(t, world)

		// Then the superseding note rides along, directly after the superseded one.
		g.Expect(result.payload.paths()).To(Equal([]string{"1.top.md", "2.older.md", "4.newer.md", "3.low.md"}))

		item, found := result.payload.find("4.newer.md")
		g.Expect(found).To(BeTrue())
		g.Expect(item.Provenances).To(Equal([]string{"ride_along"}))
	})
}

// Test792a7zh1a_Targets_Query_AQueryLeavesTheVaultAsItWas verifies 792a7zh1a (A query leaves the vault as it was):
// over a vault with notes (one used before, with a last_used in its sidecar) and sidecars and a chunk index,
// engram query with a phrase that matches them leaves every note, sidecar and chunk-index file byte-identical, asks
// the file system for no write, and prints a payload with no activated field.
func Test792a7zh1a_Targets_Query_AQueryLeavesTheVaultAsItWas(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		usedCos   = 0.95
		unusedCos = 0.9
		chunkCos  = 0.85
	)

	// Given a vault with notes and sidecars, and a chunk index.
	world := newMq1World(mq2Dims, mq2GoPhrase)
	usedVec := world.cosines(map[int]float64{0: usedCos})
	world.plant(mq1Note{
		name: "1.used-note", content: mq2NoteContent("", "the used note"), situation: usedVec, body: usedVec,
		lastUsed: mq1Now.AddDate(0, 0, -7).Format(time.DateOnly),
	})
	mq2PlantNote(world, "2.unused-note", world.cosines(map[int]float64{0: unusedCos}), "")
	world.chunks = append(world.chunks, world.chunk("section-000", map[int]float64{0: chunkCos}))

	machine := mq6VaultMachine(t, world)
	before := maps.Clone(machine.files)

	for path, data := range before {
		before[path] = slices.Clone(data)
	}

	g.Expect(before).To(HaveKey(mq1Vault + "/1.used-note.vec.json"))
	g.Expect(before).To(HaveKey(mq1ChunkIndex))

	// When the user runs engram query with a phrase that matches them.
	run := mq6RunQuery(t, machine, world)

	// Then every note, sidecar and chunk-index file is byte-identical afterwards, and no write was asked for,
	g.Expect(run.stderr).To(BeEmpty())
	g.Expect(run.payload.paths()).To(ConsistOf("1.used-note.md", "2.unused-note.md", mq1ChunkPath("section-000")))
	// mq4Machine refuses every write, so the byte comparison cannot fail alone; the empty writes record guards it.
	g.Expect(machine.files).To(Equal(before))
	g.Expect(machine.writes).To(BeEmpty())

	// and the payload has no activated field.
	g.Expect(run.raw).NotTo(HaveKey("activated"))
	g.Expect(run.stdout).NotTo(ContainSubstring("activated"))
}

// unexported constants.
const (
	mq6Dims = 3
	// mq6FixedArgs is the number of words of `engram query --vault <v> --chunks-dir <d>` before the phrases.
	mq6FixedArgs = 5
	mq6Source    = "/docs/notes.md"
)

// mq6ChunkCluster is one cluster of the payload of engram query-chunks.
type mq6ChunkCluster struct {
	ID      *int             `yaml:"id"`
	Size    int              `yaml:"size"`
	Members []mq6ChunkMember `yaml:"members"`
}

// mq6ChunkItem is one item of the payload of engram query-chunks; a pointer marks a key the test checks is there.
type mq6ChunkItem struct {
	Source  string   `yaml:"source"`
	Anchor  string   `yaml:"anchor"`
	Score   *float64 `yaml:"score"`
	Content string   `yaml:"content"`
}

// mq6ChunkMember is one member of a cluster of the payload of engram query-chunks.
type mq6ChunkMember struct {
	Source string   `yaml:"source"`
	Anchor string   `yaml:"anchor"`
	Score  *float64 `yaml:"score"`
}

// mq6ChunkPayload is the payload of engram query-chunks.
type mq6ChunkPayload struct {
	Version  int               `yaml:"version"`
	Space    string            `yaml:"space"`
	Items    []mq6ChunkItem    `yaml:"items"`
	Clusters []mq6ChunkCluster `yaml:"clusters"`
	Budget   struct {
		TotalChunks *int `yaml:"total_chunks"`
		Returned    *int `yaml:"returned"`
	} `yaml:"budget"`
}

// mq6CountingEmbedder is the phrase-keyed embedder of an mq1World that counts how often it is asked to embed.
type mq6CountingEmbedder struct {
	mq1PhraseEmbedder

	calls *atomic.Int32
}

func (e mq6CountingEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	e.calls.Add(1)

	return e.mq1PhraseEmbedder.Embed(ctx, text)
}

// mq6Outcome is what one run of a command produced.
type mq6Outcome struct {
	stdout string
	stderr string
	exits  []int
}

// mq6QueryOutcome is what one run of engram query over a machine produced, with the payload decoded.
type mq6QueryOutcome struct {
	mq6Outcome

	payload mq4Payload
	raw     map[string]any
}

// mq6Anchors returns the anchors of the items, in item order.
func mq6Anchors(items []mq6ChunkItem) []string {
	anchors := make([]string, 0, len(items))

	for _, item := range items {
		anchors = append(anchors, item.Anchor)
	}

	return anchors
}

// mq6Chunk returns a chunk record of mq6Source.
func mq6Chunk(anchor, text string, vec []float32) chunk.Record {
	return chunk.Record{
		Source:      mq6Source,
		Anchor:      anchor,
		ContentHash: chunk.HashText(text),
		Text:        text,
		Vector:      vec,
		IngestedAt:  mq1Now,
	}
}

// mq6ChunkMachine returns a machine that holds the chunk directory with an index of the records.
func mq6ChunkMachine(t *testing.T, records ...chunk.Record) *mq4Machine {
	t.Helper()

	encoded, err := chunk.EncodeRecords(records)
	if err != nil {
		t.Fatalf("encoding the chunk index: %v", err)
	}

	machine := mq4NewMachine()
	machine.files[filepath.Join(mq4ChunksDir, mq4IndexFile)] = encoded

	return machine
}

// mq6ConvertAll rewrites every note of the world with \r\n line endings.
func mq6ConvertAll(t *testing.T, world *mq1World) {
	t.Helper()

	for path := range world.files.files {
		if strings.HasSuffix(path, ".md") {
			mq6ConvertNote(t, world, path)
		}
	}
}

// mq6ConvertNote rewrites the note with \r\n line endings and sets the content hash of its sidecar (if it has one) to
// the hash of the new text, as the embedding of the CRLF file would have.
func mq6ConvertNote(t *testing.T, world *mq1World, path string) {
	t.Helper()

	crlf := bytes.ReplaceAll(world.files.files[path], []byte("\n"), []byte("\r\n"))
	world.files.files[path] = crlf

	sidecarPath := embed.SidecarPath(path)

	raw, found := world.files.files[sidecarPath]
	if !found {
		return
	}

	sidecar, err := embed.UnmarshalSidecar(raw)
	if err != nil {
		t.Fatalf("decoding the sidecar of %s: %v", path, err)
	}

	sidecar.ContentHash = embed.ContentHash(crlf)
	world.files.files[sidecarPath] = embed.MarshalSidecar(sidecar)
}

// mq6DecodeChunks decodes the payload of engram query-chunks.
func mq6DecodeChunks(t *testing.T, stdout string) mq6ChunkPayload {
	t.Helper()

	var payload mq6ChunkPayload

	err := yaml.Unmarshal([]byte(stdout), &payload)
	if err != nil {
		t.Fatalf("decoding the payload %q: %v", stdout, err)
	}

	return payload
}

// mq6NewEmbedder returns the embedder of the world, counting its calls.
func mq6NewEmbedder(world *mq1World) mq6CountingEmbedder {
	return mq6CountingEmbedder{
		mq1PhraseEmbedder: mq1PhraseEmbedder{dims: world.dims, vectors: world.vectors},
		calls:             &atomic.Int32{},
	}
}

// mq6RunCommand runs `engram <args>` through cli.Targets over the machine, with the embedder, and collects what it
// printed and which exit statuses it asked for.
func mq6RunCommand(t *testing.T, machine *mq4Machine, embedder embed.Embedder, args ...string) mq6Outcome {
	t.Helper()

	var (
		stdout, stderr bytes.Buffer
		exits          []int
	)

	deps := cli.Deps{
		Stdout:      &stdout,
		Stderr:      &stderr,
		Exit:        func(code int) { exits = append(exits, code) },
		Getenv:      func(string) string { return "" },
		Now:         func() time.Time { return mq1Now },
		UserHomeDir: func() (string, error) { return "/home/mq6", nil },
		FS:          machine.edgeFS(),
		Embed:       embedder,
	}

	_, err := targ.Execute(append([]string{"engram"}, args...), cli.Targets(deps)...)
	if err != nil {
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return mq6Outcome{stdout: stdout.String(), stderr: stderr.String(), exits: exits}
}

// mq6RunQuery runs engram query with the world's own phrases over the machine and decodes the payload.
func mq6RunQuery(t *testing.T, machine *mq4Machine, world *mq1World) mq6QueryOutcome {
	t.Helper()

	args := make([]string, 0, mq6FixedArgs+2*len(world.phrases))
	args = append(args, "query", "--vault", mq1Vault, "--chunks-dir", mq1ChunksDir)

	for _, phrase := range world.phrases {
		args = append(args, "--phrase", phrase)
	}

	run := mq6RunCommand(t, machine, mq6NewEmbedder(world), args...)
	outcome := mq6QueryOutcome{mq6Outcome: run}

	err := yaml.Unmarshal([]byte(run.stdout), &outcome.payload)
	if err != nil {
		t.Fatalf("decoding the payload %q (stderr %q): %v", run.stdout, run.stderr, err)
	}

	err = yaml.Unmarshal([]byte(run.stdout), &outcome.raw)
	if err != nil {
		t.Fatalf("decoding the payload keys: %v", err)
	}

	return outcome
}

// mq6VaultMachine returns a machine that holds the files of the world, and its chunk index if it has chunks.
func mq6VaultMachine(t *testing.T, world *mq1World) *mq4Machine {
	t.Helper()

	machine := mq4NewMachine()

	for path, data := range world.files.files {
		machine.files[path] = slices.Clone(data)
	}

	if len(world.chunks) > 0 {
		encoded, err := chunk.EncodeRecords(world.chunks)
		if err != nil {
			t.Fatalf("encoding the chunk index: %v", err)
		}

		machine.files[mq1ChunkIndex] = encoded
	}

	return machine
}
