package cli_test

// Scenario tests for the Memory Query product capability, part 4 (861c2zo): reading one chunk (792a7t), the
// project filter (792a7u), runbooks that rank like other notes (792a7v), red flags in the query preview (792a7w)
// and reading one note (792a7x). Spec: docs/specs/product/memory-query.md.
//
// 792a7u, 792a7v and 792a7w run cli.RunQuery over the in-memory world of memory_query_part1_scenarios_test.go
// (mq1World, a phrase-keyed embedder, so every cosine is a number the test chose). 792a7t and 792a7x are about
// what the user sees at the command line (stdout, stderr, the exit status), so they run `engram show-chunk` and
// `engram show` through cli.Targets over a cli.Deps whose EdgeFS is an in-memory fake that refuses every write
// (mq4Machine): nothing touches the disk, and a write by a read-only command would show. The helpers carry the
// mq4 prefix.

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// Test792a7t1a_Targets_ShowChunk_AChunksTextIsPrinted verifies 792a7t1a (A chunk's text is printed): of a
// document ingested into three sections, engram show-chunk "<document>#Baking" prints the text of the Baking
// section, and only that, followed by a newline, and writes nothing.
func Test792a7t1a_Targets_ShowChunk_AChunksTextIsPrinted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		bakingText  = "# Baking\n\nSourdough needs a long, cold proof before it goes into a hot oven."
		roastedText = "# Roasting\n\nA whole chicken roasts best on a rack, skin dry, in a hot oven."
		sodaText    = "# Baking soda\n\nBaking soda lifts a batter only when it meets an acid, like buttermilk."
	)

	// Given an ingested document with a "Baking" section, among others, one of which is named "Baking soda".
	machine := mq4NewMachine()
	machine.ingest(t, mq4Document, roastedText+"\n\n"+sodaText+"\n\n"+bakingText+"\n")

	// When the user runs engram show-chunk "<document>#Baking".
	outcome := machine.run(t, "show-chunk", mq4Document+"#Baking", "--chunks-dir", mq4ChunksDir)

	// Then stdout is that section's text.
	g.Expect(outcome.stdout).To(Equal(bakingText + "\n"))
	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(outcome.exits).To(BeEmpty())
	g.Expect(machine.writes).To(BeEmpty(), "show-chunk writes nothing")
}

// Test792a7t1b_Targets_ShowChunk_AnUnknownChunkIsAnError verifies 792a7t1b (An unknown chunk is an error): over
// an index that holds no chunk "nosuch#x", engram show-chunk "nosuch#x" fails with "chunk not found: nosuch#x",
// prints nothing and exits non-zero.
func Test792a7t1b_Targets_ShowChunk_AnUnknownChunkIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a chunk index that holds no chunk "nosuch#x".
	machine := mq4NewMachine()
	machine.ingest(t, mq4Document, mq4BakingDocument)

	// When the user runs engram show-chunk "nosuch#x".
	outcome := machine.run(t, "show-chunk", "nosuch#x", "--chunks-dir", mq4ChunksDir)

	// Then it fails with "chunk not found: nosuch#x" and exits non-zero.
	g.Expect(outcome.stderr).To(Equal("chunk not found: nosuch#x\n"))
	g.Expect(outcome.exits).To(Equal([]int{1}))
	g.Expect(outcome.stdout).To(BeEmpty())
	g.Expect(machine.writes).To(BeEmpty())
}

// Test792a7t1c_Targets_ShowChunk_AnEmptyIDIsAnError verifies 792a7t1c (An empty ID is an error): over any chunk
// index, engram show-chunk with a blank ID fails with "show-chunk: empty chunk ref", prints nothing and exits
// non-zero.
func Test792a7t1c_Targets_ShowChunk_AnEmptyIDIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given any chunk index.
	machine := mq4NewMachine()
	machine.ingest(t, mq4Document, mq4BakingDocument)

	// When the user runs engram show-chunk with a blank ID.
	outcome := machine.run(t, "show-chunk", "   ", "--chunks-dir", mq4ChunksDir)

	// Then it fails with "show-chunk: empty chunk ref".
	g.Expect(outcome.stderr).To(Equal("show-chunk: empty chunk ref\n"))
	g.Expect(outcome.exits).To(Equal([]int{1}))
	g.Expect(outcome.stdout).To(BeEmpty())
}

// Test792a7u1a_RunQuery_NotesOfOtherProjectsAreDropped verifies 792a7u1a (Notes of other projects are dropped):
// of two matched notes, one with project: alpha and one with project: beta, engram query --project alpha leaves
// the alpha note only, and the beta note is nowhere in the output, clusters included.
func Test792a7u1a_RunQuery_NotesOfOtherProjectsAreDropped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given matched notes with project: alpha and project: beta.
	world := newMq1World(mq2Dims, mq2GoPhrase)
	mq2PlantNote(world, "1.alpha-note", world.cosines(map[int]float64{0: mq4Closer}), "project: alpha\n")
	mq2PlantNote(world, "2.beta-note", world.cosines(map[int]float64{0: mq4Close}), "project: beta\n")

	// When the user runs engram query --project alpha.
	result := mq4MustQuery(t, world, cli.QueryArgs{Project: "alpha"})

	// Then only the alpha note remains among the matched notes.
	g.Expect(result.payload.paths()).To(Equal([]string{"1.alpha-note.md"}))
	g.Expect(result.payload.clusterPaths()).NotTo(ContainElement("2.beta-note.md"))
	g.Expect(result.stdout).NotTo(ContainSubstring("beta-note"), "a dropped note is nowhere in the output")
}

// Test792a7u1b_RunQuery_ABodyMentionDoesNotMatch verifies 792a7u1b (A body mention does not match): a matched
// note whose body says "project: alpha" and whose frontmatter has no project is dropped by engram query
// --project alpha, while a note whose frontmatter says project: alpha stays.
func Test792a7u1b_RunQuery_ABodyMentionDoesNotMatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a matched note whose body says "project: alpha" and whose frontmatter has no project.
	world := newMq1World(mq2Dims, mq2GoPhrase)
	mq2PlantNote(world, "1.alpha-note", world.cosines(map[int]float64{0: mq4Closer}), "project: alpha\n")
	vec := world.cosines(map[int]float64{0: mq4Close})
	world.plant(mq1Note{
		name:      "2.body-mention",
		content:   mq2NoteContent("", "project: alpha\nthe body-mention note"),
		situation: vec,
		body:      vec,
	})

	// When the user runs engram query --project alpha.
	result := mq4MustQuery(t, world, cli.QueryArgs{Project: "alpha"})

	// Then the note is dropped.
	g.Expect(result.payload.paths()).To(Equal([]string{"1.alpha-note.md"}))
	g.Expect(result.stdout).NotTo(ContainSubstring("body-mention"))
}

// Test792a7u1c_RunQuery_NoFilterDropsNothing verifies 792a7u1c (No filter drops nothing): matched notes of two
// projects and one with no project are all returned by engram query with no --project.
func Test792a7u1c_RunQuery_NoFilterDropsNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given matched notes of several projects.
	world := newMq1World(mq2Dims, mq2GoPhrase)
	mq2PlantNote(world, "1.alpha-note", world.cosines(map[int]float64{0: mq4Closer}), "project: alpha\n")
	mq2PlantNote(world, "2.beta-note", world.cosines(map[int]float64{0: mq4Close}), "project: beta\n")
	mq2PlantNote(world, "3.loose-note", world.cosines(map[int]float64{0: mq4Near}), "")

	// When the user runs engram query with no --project.
	result := mq4MustQuery(t, world, cli.QueryArgs{})

	// Then all of them are returned.
	g.Expect(result.payload.paths()).To(Equal([]string{"1.alpha-note.md", "2.beta-note.md", "3.loose-note.md"}))
}

// Test792a7u1d_RunQuery_ExploreAndRideAlongNotesOfOtherProjectsAreDropped verifies 792a7u1d (Explore and
// ride-along notes of other projects are dropped): over matched notes with project: alpha, an explore-half note
// and a ride-along note with project: beta, engram query --project alpha leaves no note whose project is not
// alpha, in the items or in the clusters, while the explore half still holds the explore note of project alpha.
func Test792a7u1d_RunQuery_ExploreAndRideAlongNotesOfOtherProjectsAreDropped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given matched notes with project: alpha, ...
	world := newMq1World(mq2Dims, mq2GoPhrase)
	mq2PlantNote(world, "1.alpha-top", world.cosines(map[int]float64{0: mq4Closer}), "project: alpha\n")
	mq2PlantNote(world, "2.alpha-older", world.cosines(map[int]float64{0: mq4Close}), "project: alpha\n")

	// ... a ride-along note with project: beta (it supersedes the matched alpha note 2.alpha-older) ...
	mq2PlantNote(world, "3.beta-newer", mq2Unmatched,
		"supersedes:\n  - note: 2.alpha-older\n    type: updates\n    claim: the older note was wrong\n"+
			"project: beta\n")

	// ... and an explore-half note with project: beta, beside one with project: alpha.
	mq2PlantNote(world, "4.beta-explore", mq2Spec{cosine: mq4ExploreNear}.vec(), mq2TagLine()+"project: beta\n")
	mq2PlantNote(world, "5.alpha-explore", mq2Spec{cosine: mq4ExploreFar}.vec(), mq2TagLine()+"project: alpha\n")
	mq2WriteCentroids(t, world)

	// The control: without the filter, the output holds the ride-along note and the explore notes of both projects.
	control := mq4MustQuery(t, world, cli.QueryArgs{})
	g.Expect(control.payload.pathsWith("ride_along")).To(Equal([]string{"3.beta-newer.md"}))
	g.Expect(control.payload.pathsWith("explore")).To(Equal([]string{"4.beta-explore.md", "5.alpha-explore.md"}))

	// When the user runs engram query --project alpha.
	result := mq4MustQuery(t, world, cli.QueryArgs{Project: "alpha"})

	// Then the output holds no note whose project is not alpha, in the items or in the clusters.
	g.Expect(result.payload.paths()).To(Equal([]string{"1.alpha-top.md", "2.alpha-older.md", "5.alpha-explore.md"}))
	g.Expect(result.payload.pathsWith("explore")).To(Equal([]string{"5.alpha-explore.md"}))
	g.Expect(result.payload.clusterPaths()).NotTo(ContainElement(ContainSubstring("beta")))
	g.Expect(result.stdout).NotTo(ContainSubstring("beta"), "a dropped note is nowhere in the output")
}

// Test792a7v1a_RunQuery_ARunbookIsReturnedLikeAFact verifies 792a7v1a (A runbook is returned like a fact): a
// runbook whose situation is close to the phrase appears in items with kind runbook, beside a fact that is as
// close.
func Test792a7v1a_RunQuery_ARunbookIsReturnedLikeAFact(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a runbook whose situation is close to the phrase.
	world := newMq1World(mq2Dims, mq2GoPhrase)
	vec := world.cosines(map[int]float64{0: mq4Closer})
	world.plant(mq1Note{
		name: "1.2026-10-09.release-runbook", content: mq4RunbookContent(""), situation: vec, body: vec,
	})
	mq2PlantNote(world, "2.2026-10-09.release-fact", vec, "")

	// When the user runs engram query with that phrase.
	result := mq4MustQuery(t, world, cli.QueryArgs{})

	// Then it appears in items with kind runbook.
	runbook, found := result.payload.find("1.2026-10-09.release-runbook.md")
	g.Expect(found).To(BeTrue(), "the runbook is among the items")
	g.Expect(runbook.Kind).To(Equal("runbook"))
	g.Expect(runbook.Provenances).To(Equal([]string{"direct"}))

	fact, factFound := result.payload.find("2.2026-10-09.release-fact.md")
	g.Expect(factFound).To(BeTrue())
	g.Expect(fact.Kind).To(Equal("fact"))
	g.Expect(float64(runbook.Score)).To(BeNumerically("~", float64(fact.Score), mq1Tolerance), "same score as the fact")
}

// Test792a7v1b_RunQuery_SurfacingDoesNotNeedACandidateSlot verifies 792a7v1b (Surfacing does not need a
// candidate slot): in a cluster of seven notes, whose candidate_l2s lists five, the runbook that is farthest
// from the cluster's centroid is an item and is not in candidate_l2s, so the presence in items is what shows it
// surfaced.
func Test792a7v1b_RunQuery_SurfacingDoesNotNeedACandidateSlot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		candidateCap = 5
		runbookName  = "7.2026-10-09.release-runbook"
		runbookPath  = runbookName + ".md"
	)

	// Given a runbook in items whose cluster's candidate_l2s does not list it: seven notes of one topic, the
	// runbook the farthest from their centroid, and another topic beside them.
	world := newMq1World(mq2Dims, mq2GoPhrase)

	for index, deviation := range []float32{0, 0.02, -0.04, 0.06, -0.08, 0.10} {
		mq2PlantNote(world, fmt.Sprintf("%d.near-%d", index+1, index), mq2Vec(mq2Main, 0, deviation, 0), "")
	}

	farVec := mq2Vec(mq2Main, 0, mq4RunbookDeviation, 0)
	world.plant(mq1Note{name: runbookName, content: mq4RunbookContent(""), situation: farVec, body: farVec})
	mq2PlantOtherTopic(world, mq4OtherTopicSize)

	// When the payload is read.
	result := mq4MustQuery(t, world, cli.QueryArgs{})

	// Then the runbook has surfaced: it is an item of kind runbook, though no candidate_l2s lists it.
	runbook, found := result.payload.find(runbookPath)
	g.Expect(found).To(BeTrue(), "the runbook is among the items")
	g.Expect(runbook.Kind).To(Equal("runbook"))

	cluster, clusterFound := mq4ClusterOf(result.payload, runbookPath)
	g.Expect(clusterFound).To(BeTrue(), "the runbook is a member of a cluster")

	if !clusterFound {
		return
	}

	g.Expect(cluster.candidatePaths()).To(HaveLen(candidateCap), "the candidate list is capped")
	g.Expect(cluster.candidatePaths()).NotTo(ContainElement(runbookPath), "and the runbook has no slot in it")
}

// Test792a7v1c_RunQuery_ARunbookHasNoKindBoost verifies 792a7v1c (A runbook has no kind boost): a runbook and a
// feedback note whose situations are equally close to the phrase get the same score, and when either is a little
// closer, it comes first, whichever kind it is and whichever file name sorts first.
func Test792a7v1c_RunQuery_ARunbookHasNoKindBoost(t *testing.T) {
	t.Parallel()

	const (
		runbookName  = "1.2026-10-09.release-runbook"
		feedbackName = "2.2026-10-09.release-feedback"
	)

	cases := map[string]struct {
		runbookCosine  float64
		feedbackCosine float64
		wantOrder      []string
	}{
		"equally close": {
			runbookCosine: mq4Close, feedbackCosine: mq4Close,
			wantOrder: nil,
		},
		"the feedback note a little closer": {
			runbookCosine: mq4Close, feedbackCosine: mq4Slightly,
			wantOrder: []string{feedbackName + ".md", runbookName + ".md"},
		},
		"the runbook a little closer": {
			runbookCosine: mq4Slightly, feedbackCosine: mq4Close,
			wantOrder: []string{runbookName + ".md", feedbackName + ".md"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a runbook and a feedback note whose situations are (nearly) equally close to the phrase,
			// and no trigger hit.
			world := newMq1World(mq2Dims, mq2GoPhrase)
			runbookVec := world.cosines(map[int]float64{0: tc.runbookCosine})
			feedbackVec := world.cosines(map[int]float64{0: tc.feedbackCosine})
			world.plant(mq1Note{
				name: runbookName, content: mq4RunbookContent(""), situation: runbookVec, body: runbookVec,
			})
			world.plant(mq1Note{
				name: feedbackName, content: mq4FeedbackContent(), situation: feedbackVec, body: feedbackVec,
			})

			// When the user runs engram query with that phrase.
			result := mq4MustQuery(t, world, cli.QueryArgs{})

			// Then their order follows their scores alone.
			runbook, runbookFound := result.payload.find(runbookName + ".md")
			feedback, feedbackFound := result.payload.find(feedbackName + ".md")
			g.Expect(runbookFound).To(BeTrue())
			g.Expect(feedbackFound).To(BeTrue())

			if tc.wantOrder == nil {
				g.Expect(float64(runbook.Score)).To(BeNumerically("~", float64(feedback.Score), mq1Tolerance))

				return
			}

			g.Expect(result.payload.paths()).To(Equal(tc.wantOrder))
			g.Expect(runbook.Provenances).To(Equal([]string{"direct"}))
			g.Expect(feedback.Provenances).To(Equal([]string{"direct"}))
		})
	}
}

// Test792a7w1a_RunQuery_RedFlagsAreInThePayload verifies 792a7w1a (Red flags are in the payload): the item of a
// runbook with two short red flags, matched by the phrase, has both entries in its content.
func Test792a7w1a_RunQuery_RedFlagsAreInThePayload(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		first  = "tag from a dirty working tree"
		second = "push the tag before the build is green"
	)

	// Given a runbook with two short red flags.
	world := newMq1World(mq2Dims, mq2GoPhrase)
	mq4PlantRunbook(world, mq4RedFlagsLines([]string{first, second}))

	// When the user runs engram query with a phrase that matches it.
	result := mq4MustQuery(t, world, cli.QueryArgs{})

	// Then its item's content includes both entries.
	item, found := result.payload.find(mq4RunbookName + ".md")
	g.Expect(found).To(BeTrue())
	g.Expect(item.Content).To(ContainSubstring("    - " + first + "\n"))
	g.Expect(item.Content).To(ContainSubstring("    - " + second + "\n"))
	g.Expect(item.Content).NotTo(ContainSubstring("OMITTED"))
}

// Test792a7w1b_RunQuery_AnOversizedListKeepsItsNewestEntries verifies 792a7w1b (An oversized list keeps its
// newest entries): of a runbook with 30 red flags of 100 characters, the content holds a marker saying
// "20 EARLIER RED_FLAGS OMITTED" with the note's basename and "all 30", followed by the last ten entries.
func Test792a7w1b_RunQuery_AnOversizedListKeepsItsNewestEntries(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a runbook with 30 red flags of about 100 characters each.
	entries := mq4RedFlagEntries(mq4OversizedCount)
	world := newMq1World(mq2Dims, mq2GoPhrase)
	mq4PlantRunbook(world, mq4RedFlagsLines(entries))

	// When the user runs engram query with a phrase that matches it.
	result := mq4MustQuery(t, world, cli.QueryArgs{})

	// Then its content holds the marker, followed by the last ten entries.
	item, found := result.payload.find(mq4RunbookName + ".md")
	g.Expect(found).To(BeTrue())
	g.Expect(mq4RedFlagsBlock(item.Content)).To(Equal(mq4WantTruncatedBlock(entries)))
	g.Expect(item.Content).To(ContainSubstring("20 EARLIER RED_FLAGS OMITTED"))
	g.Expect(item.Content).To(ContainSubstring("run engram show " + mq4RunbookName + " for all 30"))
}

// Test792a7w1c_RunQuery_AShortListIsUnchanged verifies 792a7w1c (A short list is unchanged): the red_flags block
// of a runbook whose red flags take under 1200 bytes, and of one that takes exactly 1200, is identical in the
// item's content to the note file's.
func Test792a7w1c_RunQuery_AShortListIsUnchanged(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"five entries, well under the limit":   mq4RedFlagEntries(mq4ShortCount),
		"entries that take exactly 1200 bytes": mq4LimitEntries(),
	}

	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a runbook whose red flags take under 1200 bytes.
			world := newMq1World(mq2Dims, mq2GoPhrase)
			mq4PlantRunbook(world, mq4RedFlagsLines(entries))

			noteFile := string(world.files.files[mq1Vault+"/"+mq4RunbookName+".md"])
			g.Expect(mq4RedFlagsBytes(entries)).To(BeNumerically("<=", mq4PreviewBudget))

			// When the user runs engram query with a phrase that matches it.
			result := mq4MustQuery(t, world, cli.QueryArgs{})

			// Then its red_flags block is identical to the note file's.
			item, found := result.payload.find(mq4RunbookName + ".md")
			g.Expect(found).To(BeTrue())
			g.Expect(mq4RedFlagsBlock(item.Content)).To(Equal(mq4RedFlagsBlock(noteFile)))
			g.Expect(mq4RedFlagsBlock(item.Content)).To(HaveLen(len(entries)))
			g.Expect(item.Content).NotTo(ContainSubstring("OMITTED"))
		})
	}
}

// Test792a7w1d_RunQuery_ATriggerHitIsTruncatedTheSameWay verifies 792a7w1d (A trigger hit is truncated the same
// way): the same oversized runbook, hit by --text through one of its triggers, holds the same marker and the
// same last ten entries as when the phrase matches it.
func Test792a7w1d_RunQuery_ATriggerHitIsTruncatedTheSameWay(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given the same oversized runbook, with a trigger.
	entries := mq4RedFlagEntries(mq4OversizedCount)
	world := newMq1World(mq2Dims, mq2GoPhrase)
	mq4PlantRunbook(world, "triggers:\n    - /release\n"+mq4RedFlagsLines(entries))

	byPhrase := mq4MustQuery(t, world, cli.QueryArgs{})

	// When the user runs engram query --text "<message>", with a message that hits one of its triggers.
	byTrigger := mq4MustQuery(t, world, cli.QueryArgs{Text: "please /release it now", Phrases: []string{}})

	// Then its content holds the same marker.
	item, found := byTrigger.payload.find(mq4RunbookName + ".md")
	g.Expect(found).To(BeTrue())
	g.Expect(item.Provenances).To(Equal([]string{"trigger"}), "the item is a trigger hit, not a similarity match")
	g.Expect(mq4RedFlagsBlock(item.Content)).To(Equal(mq4WantTruncatedBlock(entries)))
	g.Expect(item.Content).To(ContainSubstring("20 EARLIER RED_FLAGS OMITTED"))

	matched, matchedFound := byPhrase.payload.find(mq4RunbookName + ".md")
	g.Expect(matchedFound).To(BeTrue())
	g.Expect(mq4RedFlagsBlock(item.Content)).To(Equal(mq4RedFlagsBlock(matched.Content)))
}

// Test792a7x1a_Targets_Show_ANoteIsPrintedWithItsLinks verifies 792a7x1a (A note is printed with its links):
// engram show 1.hub prints the note as the file holds it, then the outbound links line and 2.alpha and 3.beta,
// and writes nothing.
func Test792a7x1a_Targets_Show_ANoteIsPrintedWithItsLinks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const hub = "---\ntype: fact\ntier: L3\nsituation: finding the notes about baking\n---\n\n" +
		"The hub links [[2.alpha]] and [[3.beta]].\n"

	// Given a note 1.hub that links [[2.alpha]] and [[3.beta]].
	machine := mq4NewMachine()
	machine.note("1.hub", hub)
	machine.note("2.alpha", mq4PlainNote("alpha"))
	machine.note("3.beta", mq4PlainNote("beta"))

	// When the user runs engram show 1.hub.
	outcome := machine.run(t, "show", "1.hub", "--vault", mq4Vault)

	// Then stdout is the note, then the outbound links line and 2.alpha and 3.beta.
	g.Expect(outcome.stdout).To(Equal(hub + "\n" + mq4LinksHeader + "\n2.alpha\n3.beta\n"))
	g.Expect(outcome.stderr).To(BeEmpty())
	g.Expect(outcome.exits).To(BeEmpty())
	g.Expect(machine.writes).To(BeEmpty(), "show changes no note")
}

// Test792a7x1b_Targets_Show_ANoteWithNoLinksSaysSo verifies 792a7x1b (A note with no links says so): the stdout
// of engram show for a note that links to nothing ends with the outbound links line and (none).
func Test792a7x1b_Targets_Show_ANoteWithNoLinksSaysSo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note that links to nothing.
	machine := mq4NewMachine()
	machine.note("1.2026-10-09.lonely", mq4PlainNote("lonely"))

	// When the user runs engram show <basename>.
	outcome := machine.run(t, "show", "1.2026-10-09.lonely", "--vault", mq4Vault)

	// Then stdout ends with the outbound links line and (none).
	g.Expect(outcome.stdout).To(Equal(mq4PlainNote("lonely") + "\n" + mq4LinksHeader + "\n(none)\n"))
	g.Expect(outcome.stdout).To(HaveSuffix(mq4LinksHeader + "\n(none)\n"))
}

// Test792a7x1c_Targets_Show_EveryReferenceFormResolves verifies 792a7x1c (Every reference form resolves): over
// the note 1.2026-10-09.go-testing and another note, engram show with the basename, the basename in [[ ]], the
// basename with .md and the bare Luhmann ID 1 each prints the same note.
func Test792a7x1c_Targets_Show_EveryReferenceFormResolves(t *testing.T) {
	t.Parallel()

	const basename = "1.2026-10-09.go-testing"

	forms := map[string]string{
		"the full basename":               basename,
		"the basename in double brackets": "[[" + basename + "]]",
		"the basename with .md":           basename + ".md",
		"the bare Luhmann ID":             "1",
	}

	for name, ref := range forms {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given the note 1.2026-10-09.go-testing, and another note whose ID starts with the same digit.
			machine := mq4NewMachine()
			machine.note(basename, mq4PlainNote("go-testing"))
			machine.note("10.2026-10-09.other", mq4PlainNote("other"))

			// When the user runs engram show with the reference form.
			outcome := machine.run(t, "show", ref, "--vault", mq4Vault)

			// Then it prints that note.
			g.Expect(outcome.stdout).To(Equal(mq4PlainNote("go-testing") + "\n" + mq4LinksHeader + "\n(none)\n"))
			g.Expect(outcome.stderr).To(BeEmpty())
		})
	}
}

// Test792a7x1d_Targets_Show_ShowNeverTruncatesRedFlags verifies 792a7x1d (Show never truncates red flags): for a
// runbook whose red flags take more than 1200 bytes, engram show prints the file with every entry in file order
// and no omission marker.
func Test792a7x1d_Targets_Show_ShowNeverTruncatesRedFlags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a runbook whose red flags take more than 1200 bytes.
	entries := mq4RedFlagEntries(mq4OversizedCount)
	runbook := mq4RunbookContent(mq4RedFlagsLines(entries))
	g.Expect(mq4RedFlagsBytes(entries)).To(BeNumerically(">", mq4PreviewBudget))

	machine := mq4NewMachine()
	machine.note(mq4RunbookName, runbook)

	// When the user runs engram show for it.
	outcome := machine.run(t, "show", mq4RunbookName, "--vault", mq4Vault)

	// Then stdout holds every entry in file order and no omission marker.
	g.Expect(outcome.stdout).To(Equal(runbook + "\n" + mq4LinksHeader + "\n(none)\n"))
	g.Expect(outcome.stdout).NotTo(ContainSubstring("OMITTED"))
	g.Expect(mq4RedFlagsBlock(outcome.stdout)).To(Equal(mq4RedFlagLines(entries)))
}

// Test792a7x1e_Targets_Show_AFencedLinkIsNotListed verifies 792a7x1e (A fenced link is not listed): of a note
// whose body has [[9.fenced]] inside a code fence and [[2.alpha]] outside it, engram show lists 2.alpha only,
// while the note itself is printed whole, fence included.
func Test792a7x1e_Targets_Show_AFencedLinkIsNotListed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const note = "---\ntype: fact\nsituation: showing a fence\n---\n\n" +
		"See [[2.alpha]] for the real thing.\n\n```text\n[[9.fenced]]\n```\n"

	// Given a note whose body has [[9.fenced]] inside a code fence and [[2.alpha]] outside it.
	machine := mq4NewMachine()
	machine.note("1.fence", note)

	// When the user runs engram show for it.
	outcome := machine.run(t, "show", "1.fence", "--vault", mq4Vault)

	// Then only 2.alpha is listed.
	g.Expect(outcome.stdout).To(Equal(note + "\n" + mq4LinksHeader + "\n2.alpha\n"))
}

// Test792a7x1f_Targets_Show_AnUnknownNoteIsAnError verifies 792a7x1f (An unknown note is an error): in a vault
// with no note nosuch, engram show nosuch fails with show: note not found: "nosuch", prints nothing and exits
// non-zero.
func Test792a7x1f_Targets_Show_AnUnknownNoteIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with no note nosuch.
	machine := mq4NewMachine()
	machine.note("1.hub", mq4PlainNote("hub"))

	// When the user runs engram show nosuch.
	outcome := machine.run(t, "show", "nosuch", "--vault", mq4Vault)

	// Then it fails with show: note not found: "nosuch" and exits non-zero.
	g.Expect(outcome.stderr).To(Equal("show: note not found: \"nosuch\"\n"))
	g.Expect(outcome.exits).To(Equal([]int{1}))
	g.Expect(outcome.stdout).To(BeEmpty())
}

// Test792a7x1g_Targets_Show_ABlankReferenceIsAnError verifies 792a7x1g (A blank reference is an error): over any
// vault, engram show with a blank reference fails with show: empty note reference, prints nothing and exits
// non-zero.
func Test792a7x1g_Targets_Show_ABlankReferenceIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given any vault.
	machine := mq4NewMachine()
	machine.note("1.hub", mq4PlainNote("hub"))

	// When the user runs engram show with a blank reference.
	outcome := machine.run(t, "show", "   ", "--vault", mq4Vault)

	// Then it fails with show: empty note reference.
	g.Expect(outcome.stderr).To(Equal("show: empty note reference\n"))
	g.Expect(outcome.exits).To(Equal([]int{1}))
	g.Expect(outcome.stdout).To(BeEmpty())
}

// Test792a7x1h_Targets_Show_EachLinkIsListedOnce verifies 792a7x1h (Each link is listed once): the outbound
// links of a note that links [[2.alpha]], [[3.beta.md]], [[2.alpha]] and [[4.gamma|the gamma note]] are 2.alpha,
// 3.beta and 4.gamma|the gamma note, in that order. The order is the order of first appearance and not the
// alphabetical one: a second note links the same targets in the opposite order. The scenario is classified an
// interaction by its "in that order", but the order is the one of the output lines and no call at a
// collaborator is named, so it is verified on the output, over inputs where the two orders differ.
func Test792a7x1h_Targets_Show_EachLinkIsListedOnce(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		body string
		want string
	}{
		"the scenario's note": {
			body: "Links [[2.alpha]], [[3.beta.md]], [[2.alpha]] and [[4.gamma|the gamma note]].\n",
			want: "2.alpha\n3.beta\n4.gamma|the gamma note\n",
		},
		"first appearance and alphabetical order differ": {
			body: "Links [[4.gamma|the gamma note]], [[3.beta.md]], [[2.alpha]], [[3.beta]] and [[4.gamma|the gamma note]].\n",
			want: "4.gamma|the gamma note\n3.beta\n2.alpha\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a note whose body links the targets.
			note := "---\ntype: fact\nsituation: listing links\n---\n\n" + tc.body

			machine := mq4NewMachine()
			machine.note("1.links", note)

			// When the user runs engram show for it.
			outcome := machine.run(t, "show", "1.links", "--vault", mq4Vault)

			// Then the outbound links are listed once each, in the order of their first appearance.
			g.Expect(outcome.stdout).To(Equal(note + "\n" + mq4LinksHeader + "\n" + tc.want))
		})
	}
}

// unexported constants.
const (
	// mq4BakingDocument is a document of one section.
	mq4BakingDocument = "# Baking\n\nSourdough needs a long, cold proof before it goes into a hot oven.\n"
	mq4ChunksDir      = "/chunks"
	// mq4Close, mq4Closer, mq4Near and mq4Slightly are cosines of a note's situation to the phrase.
	mq4Close    = 0.9
	mq4Closer   = 0.95
	mq4Document = "/docs/cooking.md"
	// mq4ExploreFar and mq4ExploreNear are the cosines of two explore-half notes to their term's centroid.
	mq4ExploreFar  = 0.7
	mq4ExploreNear = 0.8
	mq4IndexFile   = "index.jsonl"
	// mq4LineOverhead is what a rendered entry takes beside its text: the indent of four spaces, the dash, the
	// space and the newline.
	mq4LineOverhead = 7
	// mq4LinksHeader is the line that precedes the outbound links of engram show.
	mq4LinksHeader = "# outbound links (fetch with: engram show <basename>)"
	// mq4MaxChunk is the section length above which the chunker splits a section further.
	mq4MaxChunk = 1500
	mq4Near     = 0.85
	// mq4OtherTopicSize is the number of notes of the other topic of 792a7v1b.
	mq4OtherTopicSize = 4
	// mq4OversizedCount is the number of red flags of the oversized runbook: 30 entries of 100 characters.
	mq4OversizedCount = 30
	// mq4PreviewBudget is the budget of the red_flags preview, in bytes of rendered YAML.
	mq4PreviewBudget = 1200
	// mq4RedFlagLength is the length of one red flag entry, in characters.
	mq4RedFlagLength = 100
	// mq4RunbookDeviation is the deviation of the farthest note of 792a7v1b from the topic's axis.
	mq4RunbookDeviation = 0.20
	mq4RunbookName      = "1.2026-10-09.release-checklist"
	// mq4ShortCount is the number of entries of a short red flags list.
	mq4ShortCount = 5
	// mq4Slightly is a hair above mq4Close: close enough for a boost of a kind to outweigh the gap.
	mq4Slightly = 0.905
	mq4Vault    = "/vault"
)

// unexported variables.
var (
	errMq4Unused = errors.New("mq4: the fake file system does not serve this call")
	errMq4Write  = errors.New("mq4: the fake file system refuses every write")
)

// mq4Item is one item of the query payload.
type mq4Item struct {
	Path        string   `yaml:"path"`
	Kind        string   `yaml:"kind"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	Content     string   `yaml:"content"`
}

// mq4Machine is an in-memory file system that serves `engram show` and `engram show-chunk` through cli.Targets
// and refuses (and records) every write.
type mq4Machine struct {
	files  map[string][]byte
	dirs   map[string]struct{}
	writes []string
}

// edgeFS returns the machine as the file system edge of cli.Deps.
func (m *mq4Machine) edgeFS() cli.EdgeFS {
	refuse := func(operation string) error {
		m.writes = append(m.writes, operation)

		return errMq4Write
	}

	return fakeEdgeFS{
		readFile: func(path string) ([]byte, error) {
			data, found := m.files[path]
			if !found {
				return nil, fmt.Errorf("read %s: %w", path, fs.ErrNotExist)
			}

			return data, nil
		},
		writeFile: func(path string, _ []byte, _ fs.FileMode) error { return refuse("write " + path) },
		writeFileAtomic: func(path string, _ []byte, _ fs.FileMode) error {
			return refuse("write atomic " + path)
		},
		writeFileExcl: func(path string, _ []byte, _ fs.FileMode) error { return refuse("write excl " + path) },
		mkdirAll:      func(path string, _ fs.FileMode) error { return refuse("mkdir " + path) },
		mkdirTemp: func(dir, _ string) (string, error) {
			return "", refuse("mkdir temp " + dir)
		},
		stat:      m.stat,
		lstat:     m.stat,
		readDir:   m.readDir,
		remove:    func(path string) error { return refuse("remove " + path) },
		removeAll: func(path string) error { return refuse("remove all " + path) },
		rename:    func(oldPath, _ string) error { return refuse("rename " + oldPath) },
		symlink:   func(_, link string) error { return refuse("symlink " + link) },
		readlink:  func(string) (string, error) { return "", errMq4Unused },
		walkDir:   func(string, fs.WalkDirFunc) error { return errMq4Unused },
	}
}

// ingest puts the document into the chunk index the way ingestion does: split into sections by chunk.Markdown,
// one record per section, whose source is the document and whose anchor is the section's heading.
func (m *mq4Machine) ingest(t *testing.T, document, raw string) {
	t.Helper()

	sections := chunk.Markdown(raw, mq4MaxChunk)
	records := make([]chunk.Record, 0, len(sections))

	for _, section := range sections {
		records = append(records, chunk.Record{
			Source:      document,
			Anchor:      section.Anchor,
			ContentHash: chunk.HashText(section.Text),
			Text:        section.Text,
			Vector:      []float32{1},
		})
	}

	encoded, err := chunk.EncodeRecords(records)
	if err != nil {
		t.Fatalf("encoding the chunk index: %v", err)
	}

	m.files[filepath.Join(mq4ChunksDir, mq4IndexFile)] = encoded
}

// note puts a note file of the given basename and content into the vault.
func (m *mq4Machine) note(basename, content string) {
	m.files[filepath.Join(mq4Vault, basename+".md")] = []byte(content)
}

// readDir lists the files of the directory, in name order.
func (m *mq4Machine) readDir(path string) ([]fs.DirEntry, error) {
	if _, found := m.dirs[path]; !found {
		return nil, fmt.Errorf("read dir %s: %w", path, fs.ErrNotExist)
	}

	names := make([]string, 0, len(m.files))

	for file := range m.files {
		if filepath.Dir(file) == path {
			names = append(names, filepath.Base(file))
		}
	}

	slices.Sort(names)

	entries := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, fakeDirEntry{name: name})
	}

	return entries, nil
}

// run runs `engram <args>` through cli.Targets over the machine and collects what it printed and which exit
// statuses it asked for.
func (m *mq4Machine) run(t *testing.T, args ...string) mq4Outcome {
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
		UserHomeDir: func() (string, error) { return "/home/mq4", nil },
		FS:          m.edgeFS(),
	}

	_, err := targ.Execute(append([]string{"engram"}, args...), cli.Targets(deps)...)
	if err != nil {
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return mq4Outcome{stdout: stdout.String(), stderr: stderr.String(), exits: exits}
}

// stat answers for the directories and the files of the machine.
func (m *mq4Machine) stat(path string) (fs.FileInfo, error) {
	if _, found := m.dirs[path]; found {
		return fakeFileInfo{dir: true}, nil
	}

	if data, found := m.files[path]; found {
		return fakeFileInfo{size: int64(len(data))}, nil
	}

	return nil, fmt.Errorf("stat %s: %w", path, fs.ErrNotExist)
}

// mq4Outcome is what one run of a command produced.
type mq4Outcome struct {
	stdout string
	stderr string
	exits  []int
}

// mq4Payload is the part of the query payload that the scenarios of part 4 read.
type mq4Payload struct {
	Items    []mq4Item    `yaml:"items"`
	Clusters []mq2Cluster `yaml:"clusters"`
}

// clusterPaths returns the paths of every cluster member and every candidate.
func (p mq4Payload) clusterPaths() []string {
	paths := make([]string, 0, len(p.Items))

	for _, group := range p.Clusters {
		paths = append(paths, group.memberPaths()...)
		paths = append(paths, group.candidatePaths()...)
	}

	return paths
}

// find returns the item with the given path.
func (p mq4Payload) find(path string) (mq4Item, bool) {
	for _, item := range p.Items {
		if item.Path == path {
			return item, true
		}
	}

	return mq4Item{}, false
}

// paths returns the paths of every item, in order.
func (p mq4Payload) paths() []string {
	paths := make([]string, 0, len(p.Items))

	for _, item := range p.Items {
		paths = append(paths, item.Path)
	}

	return paths
}

// pathsWith returns the paths of the items carrying the provenance, in item order.
func (p mq4Payload) pathsWith(provenance string) []string {
	paths := make([]string, 0, len(p.Items))

	for _, item := range p.Items {
		if slices.Contains(item.Provenances, provenance) {
			paths = append(paths, item.Path)
		}
	}

	return paths
}

// mq4Result is what one run of the query produced.
type mq4Result struct {
	payload mq4Payload
	stdout  string
}

// mq4ClusterOf returns the cluster that has the path among its members.
func mq4ClusterOf(payload mq4Payload, path string) (mq2Cluster, bool) {
	for _, group := range payload.Clusters {
		if slices.Contains(group.memberPaths(), path) {
			return group, true
		}
	}

	return mq2Cluster{}, false
}

// mq4FeedbackContent returns the file text of a feedback note.
func mq4FeedbackContent() string {
	return "---\ntype: feedback\nsituation: a situation\n---\n\nBehavior: x. Impact: y. Action: z.\n"
}

// mq4LimitEntries returns red flag entries that take exactly mq4PreviewBudget bytes of rendered YAML: full
// entries, and a last one that makes up the rest.
func mq4LimitEntries() []string {
	const fullCount = 11

	entries := mq4RedFlagEntries(fullCount)
	lastLength := mq4PreviewBudget - mq4RedFlagsBytes(entries) - mq4LineOverhead
	last := ("red flag 12 " + strings.Repeat("klmnopqrst", mq4RedFlagLength))[:lastLength]

	return append(entries, last)
}

// mq4MustQuery queries the world with the arguments (the world's own phrases unless the arguments name phrases
// or a text) and returns the decoded payload and the raw output, failing the test on any error.
func mq4MustQuery(t *testing.T, world *mq1World, args cli.QueryArgs) mq4Result {
	t.Helper()

	if args.Phrases == nil && args.Text == "" {
		args.Phrases = world.phrases
	}

	result := world.query(t, args)
	if result.err != nil {
		t.Fatalf("query failed: %v", result.err)
	}

	var payload mq4Payload

	err := yaml.Unmarshal([]byte(result.stdout), &payload)
	if err != nil {
		t.Fatalf("decoding the payload: %v", err)
	}

	return mq4Result{payload: payload, stdout: result.stdout}
}

// mq4NewMachine returns a machine that holds the vault and the chunk directory, both empty.
func mq4NewMachine() *mq4Machine {
	return &mq4Machine{
		files: make(map[string][]byte),
		dirs:  map[string]struct{}{mq4Vault: {}, mq4ChunksDir: {}},
	}
}

// mq4PlainNote returns the file text of a fact note that links to nothing.
func mq4PlainNote(name string) string {
	return "---\ntype: fact\nsituation: a situation\n---\n\nthe " + name + " note\n"
}

// mq4PlantRunbook plants a runbook, whose situation is as close to the world's phrase as mq4Closer says, with
// the extra frontmatter lines.
func mq4PlantRunbook(world *mq1World, extra string) {
	vec := world.cosines(map[int]float64{0: mq4Closer})
	world.plant(mq1Note{name: mq4RunbookName, content: mq4RunbookContent(extra), situation: vec, body: vec})
}

// mq4RedFlagEntries returns count red flag entries of mq4RedFlagLength characters, numbered from 1.
func mq4RedFlagEntries(count int) []string {
	entries := make([]string, 0, count)

	for number := 1; number <= count; number++ {
		prefix := fmt.Sprintf("red flag %02d ", number)
		padding := strings.Repeat("abcdefghij", mq4RedFlagLength)[:mq4RedFlagLength-len(prefix)]
		entries = append(entries, prefix+padding)
	}

	return entries
}

// mq4RedFlagLines returns the rendered lines of the entries: four spaces, a dash, a space and the entry.
func mq4RedFlagLines(entries []string) []string {
	lines := make([]string, 0, len(entries))

	for _, entry := range entries {
		lines = append(lines, "    - "+entry)
	}

	return lines
}

// mq4RedFlagsBlock returns the list lines that follow the red_flags key of the text.
func mq4RedFlagsBlock(text string) []string {
	_, rest, found := strings.Cut(text, "\nred_flags:\n")
	if !found {
		return nil
	}

	block := []string{}

	for line := range strings.SplitSeq(rest, "\n") {
		if !strings.HasPrefix(line, "    - ") {
			break
		}

		block = append(block, line)
	}

	return block
}

// mq4RedFlagsBytes returns the rendered YAML bytes of the entries' list.
func mq4RedFlagsBytes(entries []string) int {
	size := 0

	for _, line := range mq4RedFlagLines(entries) {
		size += len(line) + 1
	}

	return size
}

// mq4RedFlagsLines returns the red_flags frontmatter lines of the entries.
func mq4RedFlagsLines(entries []string) string {
	return "red_flags:\n" + strings.Join(mq4RedFlagLines(entries), "\n") + "\n"
}

// mq4RunbookContent returns the file text of a runbook note, with the extra frontmatter lines after its
// done_when line.
func mq4RunbookContent(extra string) string {
	return "---\ntype: runbook\nsituation: releasing a module version\ndone_when: the tag is pushed\n" +
		extra + "---\n\n1. Tag the release\n2. Push the tag\n"
}

// mq4WantTruncatedBlock is the red_flags block of the oversized runbook as the spec says the query renders it:
// the marker for the 20 earliest entries, then the last ten.
func mq4WantTruncatedBlock(entries []string) []string {
	const keptCount = 10

	marker := "    - \"[20 EARLIER RED_FLAGS OMITTED — run engram show " + mq4RunbookName + " for all 30]\""

	return append([]string{marker}, mq4RedFlagLines(entries[len(entries)-keptCount:])...)
}
