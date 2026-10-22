package cli_test

// Scenario tests for the Memory Query product capability, part 1 (861c2zl), second half: the matched-note floor
// (792a7e) and the recency scaling (792a7f). Spec: docs/specs/product/memory-query.md. The helpers (mq1World and
// the rest) are in memory_query_part1_scenarios_test.go. Every age is a whole number of days back from the fixed
// clock mq1Now, so a scaled score is an exact power of two.

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

// Test792a7e1a_RunQuery_ARelevantNoteSurvivesAFloodOfChunks verifies 792a7e1a (A relevant note survives a flood
// of chunks): a phrase that matches 30 chunks above a note that scores 0.4 for it still returns the note, in
// place of the lowest-scoring chunk.
func Test792a7e1a_RunQuery_ARelevantNoteSurvivesAFloodOfChunks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase     = "the phrase"
		noteName   = "1.lesson"
		noteCosine = 0.4
		chunkCount = 30
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.limit = mq1BigLimit
	world.plant(mq1Note{name: noteName, content: mq1FactContent("lesson"),
		situation: world.cosines(map[int]float64{0: noteCosine}), body: world.cosines(map[int]float64{0: noteCosine})})
	world.chunks = world.cosineRamp(chunkCount, 0.50, 0.01)

	payload := world.mustRun(t)

	direct := payload.directPaths()
	g.Expect(direct).To(ContainElement(noteName+".md"), "the note is among the results")
	g.Expect(direct).To(HaveLen(chunkCount), "the note takes a chunk's slot")
	g.Expect(direct).NotTo(ContainElement(mq1ChunkPath(mq1RampAnchor(0))),
		"the lowest-scoring chunk is the one evicted")
	g.Expect(direct).To(ContainElement(mq1ChunkPath(mq1RampAnchor(1))), "the next chunk stays")
}

// Test792a7e1b_RunQuery_ANoteBelowTheFloorIsNotPromoted verifies 792a7e1b (A note below the floor is not
// promoted): with a phrase that matches more than 30 chunks and a note that scores 0.2 for it, the note is not
// among the results and no chunk is evicted for it.
func Test792a7e1b_RunQuery_ANoteBelowTheFloorIsNotPromoted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase     = "the phrase"
		noteName   = "1.weak"
		noteCosine = 0.2
		chunkCount = 32
		keptChunks = 30
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.limit = mq1BigLimit
	world.plant(mq1Note{name: noteName, content: mq1FactContent("weak"),
		situation: world.cosines(map[int]float64{0: noteCosine}), body: world.cosines(map[int]float64{0: noteCosine})})
	world.chunks = world.cosineRamp(chunkCount, 0.50, 0.01)

	payload := world.mustRun(t)

	_, found := payload.find(noteName + ".md")
	g.Expect(found).To(BeFalse(), "the note is not among the results")
	g.Expect(payload.directPaths()).To(HaveLen(keptChunks), "all 30 chunk slots stay with chunks")
}

// Test792a7e1c_RunQuery_SeveralNotesShareTheReservedSlots verifies 792a7e1c (Several notes share the reserved
// slots): a phrase that 40 chunks match above seven notes that each score above 0.25 returns five of the notes,
// the five with the highest scores.
func Test792a7e1c_RunQuery_SeveralNotesShareTheReservedSlots(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase       = "the phrase"
		chunkCount   = 40
		keptSlots    = 30
		reserved     = 5
		noteBaseline = 0.30
		noteStep     = 0.05
		noteCount    = 7
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.limit = mq1BigLimit

	for index := range noteCount {
		vec := world.cosines(map[int]float64{0: noteBaseline + noteStep*float64(index)})
		world.plant(mq1Note{name: fmt.Sprintf("%d.note", index+1), content: mq1FactContent(fmt.Sprintf("note %d", index)),
			situation: vec, body: vec})
	}

	world.chunks = world.cosineRamp(chunkCount, 0.65, 0.008)

	payload := world.mustRun(t)

	wantNotes := []string{"3.note.md", "4.note.md", "5.note.md", "6.note.md", "7.note.md"}
	direct := payload.directPaths()

	g.Expect(direct).To(ContainElements(wantNotes), "the five best notes")
	g.Expect(direct).NotTo(ContainElement("1.note.md"))
	g.Expect(direct).NotTo(ContainElement("2.note.md"))
	g.Expect(direct).To(HaveLen(keptSlots), "the notes take chunk slots")

	// The five lowest of the 30 best chunks gave way: chunks 10 to 14 are evicted, chunks 15 to 39 stay.
	for index := chunkCount - keptSlots; index < chunkCount-keptSlots+reserved; index++ {
		g.Expect(direct).NotTo(ContainElement(mq1ChunkPath(mq1RampAnchor(index))))
	}

	g.Expect(direct).To(ContainElement(mq1ChunkPath(mq1RampAnchor(chunkCount - keptSlots + reserved))))
}

// Test792a7e1d_RunQuery_ADefinitionNoteIsPromotedLikeAnyOther verifies 792a7e1d (A definition note is promoted
// like any other): a phrase that 30 chunks match above a vocabulary definition note that scores 0.45 returns the
// definition note.
func Test792a7e1d_RunQuery_ADefinitionNoteIsPromotedLikeAnyOther(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase         = "the phrase"
		definitionName = "1.vocab-eval-methodology-definition"
		definitionCos  = 0.45
		chunkCount     = 30
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.limit = mq1BigLimit
	world.plant(mq1Note{name: definitionName, content: mq1DefinitionContent("eval-methodology"),
		situation: world.cosines(map[int]float64{0: definitionCos}), body: world.cosines(map[int]float64{0: definitionCos})})
	world.chunks = world.cosineRamp(chunkCount, 0.50, 0.01)

	payload := world.mustRun(t)

	g.Expect(payload.directPaths()).To(ContainElement(definitionName+".md"), "the definition note is promoted")
	g.Expect(payload.directPaths()).NotTo(ContainElement(mq1ChunkPath(mq1RampAnchor(0))),
		"in place of the lowest-scoring chunk")
}

// Test792a7f1a_RunQuery_AnOldChunkIsScaledDown verifies 792a7f1a (An old chunk is scaled down): a chunk first
// indexed 120 days ago scores a quarter of the same chunk indexed today.
func Test792a7f1a_RunQuery_AnOldChunkIsScaledDown(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "the phrase"
		chunkCos    = 0.8
		ageDays     = 120
		quarter     = 0.25
		oldAnchor   = "section-old"
		freshAnchor = "section-fresh"
	)

	world := newMq1World(mq1SmallDims, phrase)
	vec := world.cosines(map[int]float64{0: chunkCos})
	world.chunks = append(world.chunks,
		mq1ChunkRecord(oldAnchor, vec, mq1Now.AddDate(0, 0, -ageDays)),
		mq1ChunkRecord(freshAnchor, vec, mq1Now))

	payload := world.mustRun(t)

	oldItem, oldFound := payload.find(mq1ChunkPath(oldAnchor))
	freshItem, freshFound := payload.find(mq1ChunkPath(freshAnchor))
	g.Expect(oldFound).To(BeTrue())
	g.Expect(freshFound).To(BeTrue())
	g.Expect(float64(freshItem.Score)).To(BeNumerically("~", chunkCos, mq1Tolerance))
	g.Expect(float64(oldItem.Score)).To(BeNumerically("~", quarter*float64(freshItem.Score), mq1Tolerance))
}

// Test792a7f1b_RunQuery_ALongUnusedNoteIsScaledDown verifies 792a7f1b (A long-unused note is scaled down): of two
// notes with the same text, the one whose sidecar says last_used today scores four times the one last used 120
// days ago and is listed first.
func Test792a7f1b_RunQuery_ALongUnusedNoteIsScaledDown(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase    = "the phrase"
		noteCos   = 0.8
		ageDays   = 120
		fourTimes = 4
		usedToday = "1.used-today"
		usedOld   = "2.used-long-ago"
	)

	world := newMq1World(mq1SmallDims, phrase)
	vec := world.cosines(map[int]float64{0: noteCos})
	content := mq1FactContent("the same text")
	world.plant(mq1Note{name: usedOld, content: content, situation: vec, body: vec,
		lastUsed: mq1Now.AddDate(0, 0, -ageDays).Format(time.DateOnly)})
	world.plant(mq1Note{name: usedToday, content: content, situation: vec, body: vec,
		lastUsed: mq1Now.Format(time.DateOnly)})

	payload := world.mustRun(t)

	g.Expect(payload.directPaths()).To(Equal([]string{usedToday + ".md", usedOld + ".md"}), "the fresher note first")

	todayItem, todayFound := payload.find(usedToday + ".md")
	oldItem, oldFound := payload.find(usedOld + ".md")
	g.Expect(todayFound).To(BeTrue())
	g.Expect(oldFound).To(BeTrue())
	g.Expect(float64(todayItem.Score)).To(BeNumerically("~", noteCos, mq1Tolerance))
	g.Expect(float64(todayItem.Score)).To(BeNumerically("~", fourTimes*float64(oldItem.Score), mq1Tolerance))
}

// Test792a7f1c_RunQuery_TheFloorIgnoresTheScaling verifies 792a7f1c (The floor ignores the scaling): a chunk
// first indexed 120 days ago whose cosine to the phrase is 0.53 is a direct match, scored about 0.13.
func Test792a7f1c_RunQuery_TheFloorIgnoresTheScaling(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase     = "the phrase"
		chunkCos   = 0.53
		ageDays    = 120
		wantScore  = 0.1325
		chunkTitle = "section-old"
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.chunks = append(world.chunks,
		mq1ChunkRecord(chunkTitle, world.cosines(map[int]float64{0: chunkCos}), mq1Now.AddDate(0, 0, -ageDays)))

	payload := world.mustRun(t)

	chunkPath := mq1ChunkPath(chunkTitle)
	g.Expect(payload.directPaths()).To(Equal([]string{chunkPath}), "the scaled score is below 0.25, the cosine is not")

	item, found := payload.find(chunkPath)
	g.Expect(found).To(BeTrue())
	g.Expect(float64(item.Score)).To(BeNumerically("~", wantScore, mq1Tolerance))
}

// Test792a7f1d_RunQuery_ANoteThatWasNeverUsedIsNotScaled verifies 792a7f1d (A note that was never used is not
// scaled): a note engram learn wrote 120 days ago (its created date quoted) with no last_used in its sidecar
// scores its cosine.
func Test792a7f1d_RunQuery_ANoteThatWasNeverUsedIsNotScaled(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase   = "the phrase"
		noteCos  = 0.6
		ageDays  = 120
		noteName = "1.learned"
	)

	created := mq1Now.AddDate(0, 0, -ageDays).Format(time.DateOnly)
	content := "---\ntype: fact\nsituation: a situation\ncreated: \"" + created + "\"\n---\n\nlearned\n"

	world := newMq1World(mq1SmallDims, phrase)
	vec := world.cosines(map[int]float64{0: noteCos})
	world.plant(mq1Note{name: noteName, content: content, situation: vec, body: vec})

	payload := world.mustRun(t)

	item, found := payload.find(noteName + ".md")
	g.Expect(found).To(BeTrue())
	g.Expect(float64(item.Score)).To(BeNumerically("~", noteCos, mq1Tolerance), "its cosine, unscaled")
}

// Test792a7f1e_RunQuery_ALateTurnIsLifted verifies 792a7f1e (A late turn is lifted): of two chunks of one
// transcript with the same cosine to the phrase, one at its first turn (the chunker's turn 0, the preamble) and
// one at its last, both first indexed today, the last turn scores 1.2 times the first turn.
func Test792a7f1e_RunQuery_ALateTurnIsLifted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase    = "the phrase"
		chunkCos  = 0.7
		lastTurn  = "turn-4"
		firstTurn = "preamble"
		lift      = 1.2
	)

	world := newMq1World(mq1SmallDims, phrase)
	vec := world.cosines(map[int]float64{0: chunkCos})
	world.chunks = append(world.chunks,
		mq1ChunkRecord(firstTurn, vec, mq1Now),
		mq1ChunkRecord(lastTurn, vec, mq1Now))

	payload := world.mustRun(t)

	firstItem, firstFound := payload.find(mq1ChunkPath(firstTurn))
	lastItem, lastFound := payload.find(mq1ChunkPath(lastTurn))
	g.Expect(firstFound).To(BeTrue())
	g.Expect(lastFound).To(BeTrue())
	g.Expect(float64(firstItem.Score)).To(BeNumerically("~", chunkCos, mq1Tolerance))
	g.Expect(float64(lastItem.Score)).To(BeNumerically("~", lift*float64(firstItem.Score), mq1Tolerance))
}
