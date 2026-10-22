package cli_test

// Scenario tests for the Memory Query product capability, part 1 (861c2zl): the query input (792a7a), the
// payload (792a7b), the embeddings of the current model (792a7c) and the matched notes and chunks (792a7d). The
// matched-note floor (792a7e) and the recency scaling (792a7f) are in
// memory_query_part1_floor_recency_scenarios_test.go. Spec: docs/specs/product/memory-query.md.
//
// The scenarios run cli.RunQuery over an in-memory vault and chunk index with a phrase-keyed embedder, so every
// cosine in a scenario is a number the test chose: phrase i embeds to the unit vector of axis i, and a note or
// chunk is a unit vector whose cosine to that axis is the number the scenario names (mq1Vector). The shared
// helpers carry the mq1 prefix. Only the 792a7a1a command-line journey touches the disk, and only a t.TempDir()
// vault (the recorded deviation, #860).

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a7a1a_RunQuery_ANullQueryFails verifies 792a7a1a (A query with nothing to search for fails): with no
// --phrase and no --text, or with --text "   ", the query fails with "query: empty query string" and exits
// non-zero; the same invocations are driven through the command line for the exit status.
func Test792a7a1a_RunQuery_ANullQueryFails(t *testing.T) {
	t.Parallel()

	const wantMessage = "query: empty query string"

	cases := map[string]struct {
		args  cli.QueryArgs
		flags []string
	}{
		"no phrase and no text": {args: cli.QueryArgs{}, flags: nil},
		"blank text":            {args: cli.QueryArgs{Text: "   "}, flags: []string{"--text", "   "}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			result := newMq1World(mq1SmallDims).query(t, tc.args)
			g.Expect(result.err).To(MatchError(wantMessage))
			g.Expect(result.stdout).To(BeEmpty(), "nothing is printed before the failure")

			var exits []int

			argv := slices.Concat([]string{"engram", "query", "--vault", t.TempDir()}, tc.flags)
			stdout, stderr := executeCapturingBoth(t, argv, func(d *cli.Deps) {
				d.Getenv = func(string) string { return "" }
				d.Exit = func(code int) { exits = append(exits, code) }
			})

			g.Expect(stderr).To(ContainSubstring(wantMessage))
			g.Expect(stdout).To(BeEmpty())
			g.Expect(exits).To(Equal([]int{1}), "the command exits non-zero")
		})
	}
}

// Test792a7a1b_RunQuery_SeveralPhrasesAreSearchedTogether verifies 792a7a1b (Several phrases are searched
// together): a vault holding a note about Go tests and a chunk about baking, queried with both phrases, prints the
// phrases in order and holds the note and the chunk among its items.
func Test792a7a1b_RunQuery_SeveralPhrasesAreSearchedTogether(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		goPhrase    = "writing Go tests"
		bakePhrase  = "how to bake sourdough bread"
		noteName    = "1.go-tests"
		chunkAnchor = "section-baking"
	)

	world := newMq1World(mq1SmallDims, goPhrase, bakePhrase)
	world.plant(mq1Note{name: noteName, content: mq1FactContent("Table tests keep Go tests small."),
		situation: world.cosines(map[int]float64{0: mq1Strong}), body: world.cosines(map[int]float64{0: mq1Strong})})
	world.chunks = append(world.chunks, world.chunk(chunkAnchor, map[int]float64{1: mq1Strong}))

	payload := world.mustRun(t)

	g.Expect(payload.Phrases).To(Equal([]string{goPhrase, bakePhrase}))
	g.Expect(payload.Budget.PhrasesQueried).To(Equal(2))
	g.Expect(payload.directPaths()).To(ConsistOf(noteName+".md", mq1ChunkPath(chunkAnchor)))
}

// Test792a7b1a_RunQuery_ThePayloadHasTheDocumentedShape verifies 792a7b1a (The payload has the documented
// shape): a vault of six notes with current sidecars and no chunk index, queried with one phrase, prints one YAML
// document with version 1, the phrase, items, clusters, a budget whose total_notes and with_embeddings are both 6,
// and a model_id.
func Test792a7b1a_RunQuery_ThePayloadHasTheDocumentedShape(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase    = "writing Go tests"
		noteCount = 6
	)

	world := newMq1World(mq1SmallDims, phrase)

	for index := range noteCount {
		vec := world.cosines(map[int]float64{0: mq1Strong - 0.05*float64(index)})
		world.plant(mq1Note{name: fmt.Sprintf("%d.note", index+1), content: mq1FactContent(fmt.Sprintf("body %d", index)),
			situation: vec, body: vec})
	}

	world.noChunkIndex = true

	result := world.query(t, cli.QueryArgs{Phrases: []string{phrase}})
	g.Expect(result.err).NotTo(HaveOccurred())

	if result.err != nil {
		return
	}

	var document map[string]any

	g.Expect(yaml.Unmarshal([]byte(result.stdout), &document)).To(Succeed(), "stdout is one YAML document")
	g.Expect(document).To(HaveKeyWithValue("version", 1))
	g.Expect(document).To(HaveKeyWithValue("phrases", []any{phrase}))
	g.Expect(document).To(HaveKey("items"))
	g.Expect(document).To(HaveKey("clusters"))
	g.Expect(document).To(HaveKey("budget"))
	g.Expect(document).To(HaveKeyWithValue("model_id", mq1ModelID))

	var payload mq1Payload

	g.Expect(yaml.Unmarshal([]byte(result.stdout), &payload)).To(Succeed())
	g.Expect(payload.Budget.TotalNotes).To(Equal(noteCount))
	g.Expect(payload.Budget.WithEmbeddings).To(Equal(noteCount))
	g.Expect(payload.Items).NotTo(BeEmpty())
}

// Test792a7b1b_RunQuery_ANoteItemIsLabelledByItsType verifies 792a7b1b (A note item is labelled by its type): a
// fact note matched by the phrase is an item of kind fact, with provenances [direct] and its full file text as
// content.
func Test792a7b1b_RunQuery_ANoteItemIsLabelledByItsType(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase   = "prefer table tests"
		noteName = "1.table-tests"
	)

	content := mq1FactContent("Prefer table tests for Go.")
	world := newMq1World(mq1SmallDims, phrase)
	world.plant(mq1Note{name: noteName, content: content,
		situation: world.cosines(map[int]float64{0: mq1Strong}), body: world.cosines(map[int]float64{0: mq1Strong})})

	payload := world.mustRun(t)

	item, found := payload.find(noteName + ".md")
	g.Expect(found).To(BeTrue(), "the matching note is an item")
	g.Expect(item.Kind).To(Equal("fact"))
	g.Expect(item.Provenances).To(Equal([]string{"direct"}))
	g.Expect(item.Content).To(Equal(content), "its full file text")
}

// Test792a7b1c_RunQuery_WikilinksAreFlattenedInTheContent verifies 792a7b1c (Wikilinks are flattened in the
// content): a matched note whose body holds [[1.target]] and [[2.other|the other note]] has content that
// contains 1.target and the other note and no [[.
func Test792a7b1c_RunQuery_WikilinksAreFlattenedInTheContent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase   = "following links"
		noteName = "3.links"
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.plant(mq1Note{name: noteName,
		content:   mq1FactContent("See [[1.target]] and [[2.other|the other note]] for more."),
		situation: world.cosines(map[int]float64{0: mq1Strong}), body: world.cosines(map[int]float64{0: mq1Strong})})

	payload := world.mustRun(t)

	item, found := payload.find(noteName + ".md")
	g.Expect(found).To(BeTrue(), "the matching note is an item")
	g.Expect(item.Content).To(ContainSubstring("1.target"))
	g.Expect(item.Content).To(ContainSubstring("the other note"))
	g.Expect(item.Content).NotTo(ContainSubstring("[["))
	g.Expect(item.Content).To(ContainSubstring("See 1.target and the other note for more."))
}

// Test792a7c1a_RunQuery_NotesOfAnotherModelAreDroppedWithAWarning verifies 792a7c1a (Notes of another model are
// dropped with a warning): in a vault of ten notes of which nine have sidecars made by another model, one warning
// says "dropped 9 sidecar(s)" and names the other model and the active one, and the one note with a current
// sidecar is returned.
func Test792a7c1a_RunQuery_NotesOfAnotherModelAreDroppedWithAWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "pasta"
		otherCount  = 9
		currentNote = "10.current"
	)

	world := newMq1World(mq1SmallDims, phrase)
	vec := world.cosines(map[int]float64{0: mq1Strong})

	for index := range otherCount {
		world.plant(mq1Note{name: fmt.Sprintf("%d.other", index+1), content: mq1FactContent("pasta"),
			situation: vec, body: vec, modelID: mq1OtherModelID})
	}

	world.plant(mq1Note{name: currentNote, content: mq1FactContent("pasta"), situation: vec, body: vec})

	result := world.query(t, cli.QueryArgs{Phrases: []string{phrase}})
	g.Expect(result.err).NotTo(HaveOccurred())

	if result.err != nil {
		return
	}

	g.Expect(result.warnings).To(HaveLen(1), "one warning, not one per note")
	g.Expect(result.warnings[0]).To(ContainSubstring("dropped 9 sidecar(s)"))
	g.Expect(result.warnings[0]).To(ContainSubstring(mq1OtherModelID))
	g.Expect(result.warnings[0]).To(ContainSubstring(mq1ModelID))
	g.Expect(result.warnings[0]).To(ContainSubstring("engram embed apply --all"))

	var payload mq1Payload

	g.Expect(yaml.Unmarshal([]byte(result.stdout), &payload)).To(Succeed())
	g.Expect(payload.directPaths()).To(Equal([]string{currentNote + ".md"}))
	g.Expect(payload.Budget.TotalNotes).To(Equal(otherCount + 1))
	g.Expect(payload.Budget.WithEmbeddings).To(Equal(1))
}

// Test792a7c1b_RunQuery_AVaultWithNoUsableEmbeddingsIsAnError verifies 792a7c1b (A vault with no usable
// embeddings is an error): a vault of notes whose sidecars are all from another model, or missing, fails with
// "query: vault has notes but no current-model embeddings; run `engram embed apply --all`".
func Test792a7c1b_RunQuery_AVaultWithNoUsableEmbeddingsIsAnError(t *testing.T) {
	t.Parallel()

	const (
		phrase      = "pasta"
		noteCount   = 3
		wantMessage = "query: vault has notes but no current-model embeddings; run `engram embed apply --all`"
	)

	cases := map[string]mq1Note{
		"every sidecar is from another model": {modelID: mq1OtherModelID},
		"every sidecar is missing":            {noSidecar: true},
	}

	for name, template := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			world := newMq1World(mq1SmallDims, phrase)
			vec := world.cosines(map[int]float64{0: mq1Strong})

			for index := range noteCount {
				note := template
				note.name = fmt.Sprintf("%d.note", index+1)
				note.content = mq1FactContent("pasta")
				note.situation = vec
				note.body = vec
				world.plant(note)
			}

			result := world.query(t, cli.QueryArgs{Phrases: []string{phrase}})
			g.Expect(result.err).To(MatchError(wantMessage))
			g.Expect(result.stdout).To(BeEmpty(), "no payload is printed")
		})
	}
}

// Test792a7c1c_RunQuery_OldSchemaSidecarsAreSkippedWithTheirOwnWarning verifies 792a7c1c (Old-schema sidecars
// are skipped with their own warning): a vault in which one sidecar is on an old schema version prints
// "1 sidecar(s) on an old schema were skipped" and names engram embed apply --force, and no other warning.
func Test792a7c1c_RunQuery_OldSchemaSidecarsAreSkippedWithTheirOwnWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "pasta"
		oldNote     = "1.old"
		currentNote = "2.current"
	)

	world := newMq1World(mq1SmallDims, phrase)
	vec := world.cosines(map[int]float64{0: mq1Strong})
	world.plant(mq1Note{name: oldNote, content: mq1FactContent("pasta"), situation: vec, body: vec, oldSchema: true})
	world.plant(mq1Note{name: currentNote, content: mq1FactContent("pasta"), situation: vec, body: vec})

	result := world.query(t, cli.QueryArgs{Phrases: []string{phrase}})
	g.Expect(result.err).NotTo(HaveOccurred())

	if result.err != nil {
		return
	}

	g.Expect(result.warnings).To(HaveLen(1), "the old schema has its own warning and no model-mismatch warning")
	g.Expect(result.warnings[0]).To(ContainSubstring("1 sidecar(s) on an old schema were skipped"))
	g.Expect(result.warnings[0]).To(ContainSubstring("engram embed apply --force"))

	var payload mq1Payload

	g.Expect(yaml.Unmarshal([]byte(result.stdout), &payload)).To(Succeed())
	g.Expect(payload.directPaths()).To(Equal([]string{currentNote + ".md"}), "the old-schema note is skipped")
}

// Test792a7d1a_RunQuery_NotesAreRankedByTheirBetterAxis verifies 792a7d1a (Notes are ranked by their better
// axis): three notes whose situation and body cosines differ are scored by the higher of the two and ordered by
// it, highest first; the order differs from a ranking by the situation alone, by the body alone and by their mean.
func Test792a7d1a_RunQuery_NotesAreRankedByTheirBetterAxis(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const phrase = "the phrase"

	type axisCosines struct {
		name      string
		situation float64
		body      float64
		wantScore float64
	}

	notes := []axisCosines{
		{name: "1.situation-close", situation: 0.9, body: 0.1, wantScore: 0.9},
		{name: "2.body-close", situation: 0.1, body: 0.7, wantScore: 0.7},
		{name: "3.middling", situation: 0.3, body: 0.5, wantScore: 0.5},
	}

	world := newMq1World(mq1SmallDims, phrase)

	for _, note := range notes {
		world.plant(mq1Note{name: note.name, content: mq1FactContent(note.name),
			situation: world.cosines(map[int]float64{0: note.situation}),
			body:      world.cosines(map[int]float64{0: note.body})})
	}

	payload := world.mustRun(t)

	g.Expect(payload.directPaths()).To(Equal([]string{"1.situation-close.md", "2.body-close.md", "3.middling.md"}))

	for _, note := range notes {
		item, found := payload.find(note.name + ".md")
		g.Expect(found).To(BeTrue())
		g.Expect(float64(item.Score)).To(BeNumerically("~", note.wantScore, mq1Tolerance), note.name)
	}
}

// Test792a7d1b_RunQuery_AnItemBelowTheFloorIsDropped verifies 792a7d1b (An item below the floor is dropped): a
// note and a chunk whose cosine to the phrase is below 0.25 are not among the direct matches, while a note and a
// chunk just above it are.
func Test792a7d1b_RunQuery_AnItemBelowTheFloorIsDropped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "the phrase"
		justBelow   = 0.24
		justAbove   = 0.26
		lowNoteName = "1.low"
		keptNote    = "2.kept"
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.plant(mq1Note{name: lowNoteName, content: mq1FactContent("low"),
		situation: world.cosines(map[int]float64{0: justBelow}), body: world.cosines(map[int]float64{0: justBelow})})
	world.plant(mq1Note{name: keptNote, content: mq1FactContent("kept"),
		situation: world.cosines(map[int]float64{0: justAbove}), body: world.cosines(map[int]float64{0: justAbove})})
	world.chunks = append(world.chunks,
		world.chunk("section-low", map[int]float64{0: justBelow}),
		world.chunk("section-kept", map[int]float64{0: justAbove}))

	payload := world.mustRun(t)

	g.Expect(payload.directPaths()).To(ConsistOf(keptNote+".md", mq1ChunkPath("section-kept")))
}

// Test792a7d1c_RunQuery_OnlyTheBest30OfAPhraseAreKept verifies 792a7d1c (Only the best 30 of a phrase are
// kept): a phrase that 40 chunks match above the floor keeps 30 direct matches, the 30 with the highest scores.
func Test792a7d1c_RunQuery_OnlyTheBest30OfAPhraseAreKept(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "the phrase"
		chunkCount  = 40
		phraseLimit = 30
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.limit = mq1BigLimit
	world.chunks = world.cosineRamp(chunkCount, 0.50, 0.01)

	payload := world.mustRun(t)

	wantPaths := make([]string, 0, phraseLimit)

	for index := chunkCount - phraseLimit; index < chunkCount; index++ {
		wantPaths = append(wantPaths, mq1ChunkPath(mq1RampAnchor(index)))
	}

	g.Expect(payload.directPaths()).To(ConsistOf(wantPaths), "the 30 highest-scored of the 40")
	g.Expect(payload.Budget.DirectHitsReturned).To(Equal(phraseLimit))
}

// Test792a7d1d_RunQuery_AnItemKeepsItsBestScoreAcrossPhrases verifies 792a7d1d (An item keeps its best score
// across phrases): a note that scores 0.4 for one phrase and 0.7 for another appears once, scored 0.7.
func Test792a7d1d_RunQuery_AnItemKeepsItsBestScoreAcrossPhrases(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		weakCosine   = 0.4
		strongCosine = 0.7
		noteName     = "1.both"
	)

	world := newMq1World(mq1SmallDims, "first phrase", "second phrase")
	both := world.cosines(map[int]float64{0: weakCosine, 1: strongCosine})
	world.plant(mq1Note{name: noteName, content: mq1FactContent("both"), situation: both, body: both})

	payload := world.mustRun(t)

	g.Expect(payload.directPaths()).To(Equal([]string{noteName + ".md"}), "the note appears once")

	item, found := payload.find(noteName + ".md")
	g.Expect(found).To(BeTrue())
	g.Expect(float64(item.Score)).To(BeNumerically("~", strongCosine, mq1Tolerance))
}

// Test792a7d1e_RunQuery_TheMatchedSetIsCappedAt300 verifies 792a7d1e (The matched set is capped at 300): twelve
// phrases that each match 30 different chunks above the floor leave 300 matched items in the clusters, the 300
// with the highest scores.
func Test792a7d1e_RunQuery_TheMatchedSetIsCappedAt300(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phraseCount   = 12
		perPhrase     = 30
		matchSetCap   = 300
		baseCosine    = 0.30
		cosineStep    = 0.0015
		droppedChunks = phraseCount*perPhrase - matchSetCap
	)

	phrases := make([]string, 0, phraseCount)
	for index := range phraseCount {
		phrases = append(phrases, fmt.Sprintf("phrase %02d", index))
	}

	world := newMq1World(phraseCount+2, phrases...)
	world.limit = mq1BigLimit

	// Chunk (phrase, rank) scores baseCosine + cosineStep*(rank*phraseCount+phrase) for its own phrase only, so the
	// 360 chunks have distinct scores and the 60 lowest (ranks 0 to 4 of every phrase) fall below the cap.
	wantMembers := make([]string, 0, matchSetCap)

	for rank := range perPhrase {
		for phrase := range phraseCount {
			ordinal := rank*phraseCount + phrase
			anchor := mq1RampAnchor(ordinal)
			cosine := baseCosine + cosineStep*float64(ordinal)

			world.chunks = append(world.chunks, world.chunk(anchor, map[int]float64{phrase: cosine}))

			if ordinal >= droppedChunks {
				wantMembers = append(wantMembers, mq1ChunkPath(anchor))
			}
		}
	}

	payload := world.mustRun(t)

	g.Expect(payload.memberPaths()).To(ConsistOf(wantMembers), "the 300 highest-scored chunks are clustered")
	g.Expect(payload.directPaths()).To(ConsistOf(wantMembers))
	g.Expect(payload.Budget.DirectHitsReturned).To(Equal(matchSetCap))
}

// Test792a7d1f_RunQuery_NotesAndChunksCompeteInOneList verifies 792a7d1f (Notes and chunks compete in one list):
// a vault note and an indexed chunk that both match the phrase both appear, the note first even though the
// chunk's score is higher.
func Test792a7d1f_RunQuery_NotesAndChunksCompeteInOneList(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase      = "the phrase"
		noteName    = "1.note"
		noteCosine  = 0.6
		chunkCosine = 0.9
		chunkAnchor = "section-chunk"
	)

	world := newMq1World(mq1SmallDims, phrase)
	world.plant(mq1Note{name: noteName, content: mq1FactContent("note"),
		situation: world.cosines(map[int]float64{0: noteCosine}), body: world.cosines(map[int]float64{0: noteCosine})})
	world.chunks = append(world.chunks, world.chunk(chunkAnchor, map[int]float64{0: chunkCosine}))

	payload := world.mustRun(t)

	chunkPath := mq1ChunkPath(chunkAnchor)
	g.Expect(payload.directPaths()).To(Equal([]string{noteName + ".md", chunkPath}), "the note first, then the chunk")

	noteItem, noteFound := payload.find(noteName + ".md")
	chunkItem, chunkFound := payload.find(chunkPath)
	g.Expect(noteFound).To(BeTrue())
	g.Expect(chunkFound).To(BeTrue())
	g.Expect(chunkItem.Score).To(BeNumerically(">", noteItem.Score), "the chunk scores higher and still follows the note")
}

// Test792a7d1g_RunQuery_ADefinitionNoteIsAnOrdinaryMatch verifies 792a7d1g (A definition note is an ordinary
// match): in a vault whose only note close to the phrase is a vocabulary definition note (a fact tagged vocab),
// the query returns the definition note.
func Test792a7d1g_RunQuery_ADefinitionNoteIsAnOrdinaryMatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		phrase         = "what the eval-methodology term covers"
		definitionName = "1.vocab-eval-methodology"
		distantName    = "2.unrelated"
	)

	definition := "---\ntype: fact\nsituation: recalling what the eval-methodology vocab term covers\n" +
		"tags:\n    - vocab\n---\n\nDefines the eval-methodology term.\n"

	world := newMq1World(mq1SmallDims, phrase)
	world.plant(mq1Note{name: definitionName, content: definition,
		situation: world.cosines(map[int]float64{0: mq1Strong}), body: world.cosines(map[int]float64{0: mq1Strong})})
	world.plant(mq1Note{name: distantName, content: mq1FactContent("unrelated"),
		situation: world.cosines(map[int]float64{1: mq1Strong}), body: world.cosines(map[int]float64{1: mq1Strong})})

	payload := world.mustRun(t)

	item, found := payload.find(definitionName + ".md")
	g.Expect(found).To(BeTrue(), "the definition note is returned")
	g.Expect(item.Provenances).To(ContainElement("direct"))
	g.Expect(payload.directPaths()).To(Equal([]string{definitionName + ".md"}))
}

// Test792a7d1h_RunQuery_ANoteIsFoundByItsOwnSituation verifies 792a7d1h (A note is found by its own situation):
// among several notes whose body text is their situation, a query phrased as one note's situation returns that
// note with cosine 1.
func Test792a7d1h_RunQuery_ANoteIsFoundByItsOwnSituation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const vault = "/vault"

	situations := []string{
		"orchestrating multi-step work under context pressure",
		"reasoning about agent coordination and dispatch",
		"writing concurrent go code with cancellation",
		"sharpening a spec before dispatching the work",
	}

	files := newInMemoryFS()

	for index, situation := range situations {
		content := "---\ntype: fact\nsituation: " + situation + "\n---\n\n" + situation + "\n"
		plantNoteWithSidecar(t, files, vault, fmt.Sprintf("%d.note.md", index+1), content)
	}

	for index, situation := range situations {
		var out bytes.Buffer

		err := cli.RunQuery(context.Background(),
			cli.QueryArgs{Phrases: []string{situation}, VaultPath: vault, Limit: mq1BigLimit},
			newQueryDeps(files), &out)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		var payload mq1Payload

		g.Expect(yaml.Unmarshal(out.Bytes(), &payload)).To(Succeed())

		item, found := payload.find(fmt.Sprintf("%d.note.md", index+1))
		g.Expect(found).To(BeTrue(), "the note of %q is among the items", situation)
		g.Expect(float64(item.Score)).To(BeNumerically("~", 1, mq1Tolerance), "cosine 1 for %q", situation)
	}
}

// unexported constants.
const (
	mq1BigLimit     = 1000
	mq1ChunkIndex   = "/chunks/index.jsonl"
	mq1ChunksDir    = "/chunks"
	mq1ModelID      = "m@4"
	mq1OtherModelID = "old-model@4"
	mq1SmallDims    = 4
	mq1Source       = "/s/a.jsonl"
	mq1Strong       = 0.8
	mq1Tolerance    = 1e-4
	mq1Vault        = "/vault"
)

// unexported variables.
var (
	errMq1UnknownPhrase = errors.New("mq1: unknown phrase")
	mq1Now              = time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
)

// mq1Item is one item of the query payload.
type mq1Item struct {
	Path        string   `yaml:"path"`
	Kind        string   `yaml:"kind"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	Content     string   `yaml:"content"`
}

// mq1Note describes a note and its sidecar for mq1World.plant.
type mq1Note struct {
	name      string // file name without .md
	content   string
	situation []float32
	body      []float32
	modelID   string // empty means the active model
	lastUsed  string // YYYY-MM-DD, empty means never used
	oldSchema bool
	noSidecar bool
}

// mq1Payload is the part of the query payload the scenarios read.
type mq1Payload struct {
	Version  int       `yaml:"version"`
	Phrases  []string  `yaml:"phrases"`
	Items    []mq1Item `yaml:"items"`
	Clusters []struct {
		ID      int    `yaml:"id"`
		Phrase  string `yaml:"phrase"`
		Size    int    `yaml:"size"`
		Members []struct {
			Path string `yaml:"path"`
		} `yaml:"members"`
	} `yaml:"clusters"`
	Budget struct {
		PhrasesQueried     int `yaml:"phrases_queried"`
		TotalNotes         int `yaml:"total_notes"`
		WithEmbeddings     int `yaml:"with_embeddings"`
		DirectHitsReturned int `yaml:"direct_hits_returned"`
	} `yaml:"budget"`
	ModelID string `yaml:"model_id"`
}

// directPaths returns the paths of the items carrying the provenance direct, in item order.
func (p mq1Payload) directPaths() []string {
	paths := make([]string, 0, len(p.Items))

	for _, item := range p.Items {
		if slices.Contains(item.Provenances, "direct") {
			paths = append(paths, item.Path)
		}
	}

	return paths
}

// find returns the item with the given path.
func (p mq1Payload) find(path string) (mq1Item, bool) {
	for _, item := range p.Items {
		if item.Path == path {
			return item, true
		}
	}

	return mq1Item{}, false
}

// memberPaths returns the paths of every cluster member.
func (p mq1Payload) memberPaths() []string {
	paths := make([]string, 0, len(p.Items))

	for _, group := range p.Clusters {
		for _, member := range group.Members {
			paths = append(paths, member.Path)
		}
	}

	return paths
}

// mq1PhraseEmbedder embeds a declared phrase to its unit vector and refuses any other text.
type mq1PhraseEmbedder struct {
	dims    int
	vectors map[string][]float32
}

func (e mq1PhraseEmbedder) Dims() int { return e.dims }

func (e mq1PhraseEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec, found := e.vectors[text]
	if !found {
		return nil, fmt.Errorf("embedding %q: %w", text, errMq1UnknownPhrase)
	}

	return slices.Clone(vec), nil
}

func (e mq1PhraseEmbedder) ModelID() string { return mq1ModelID }

// mq1Result is what one run of the query produced.
type mq1Result struct {
	err      error
	stdout   string
	warnings []string
}

// mq1World is an in-memory vault plus chunk index and the embedder that serves its phrases: phrase i embeds to
// the unit vector of axis i, and the last axis is never a phrase (it takes the remainder of every other vector).
type mq1World struct {
	files        *inMemoryFS
	dims         int
	phrases      []string
	vectors      map[string][]float32
	chunks       []chunk.Record
	limit        int
	now          time.Time
	noChunkIndex bool
}

// chunk returns a chunk record of the world's clock whose vector has the given cosines to the phrase axes.
func (w *mq1World) chunk(anchor string, cosines map[int]float64) chunk.Record {
	return mq1ChunkRecord(anchor, w.cosines(cosines), w.clock())
}

// clock returns the query-time clock of the world.
func (w *mq1World) clock() time.Time {
	if w.now.IsZero() {
		return mq1Now
	}

	return w.now
}

// cosineRamp returns count chunks whose cosine to the first phrase rises from base by step, named by
// mq1RampAnchor, all first indexed at the world's clock.
func (w *mq1World) cosineRamp(count int, base, step float64) []chunk.Record {
	records := make([]chunk.Record, 0, count)

	for index := range count {
		records = append(records,
			w.chunk(mq1RampAnchor(index), map[int]float64{0: base + step*float64(index)}))
	}

	return records
}

// cosines returns a unit vector of the world's dimension whose cosine to the unit vector of each listed axis is the
// given number; the remainder of the unit length sits on the last axis.
func (w *mq1World) cosines(cosines map[int]float64) []float32 {
	vec := make([]float32, w.dims)

	var squares float64

	for axis, cosine := range cosines {
		vec[axis] = float32(cosine)
		squares += cosine * cosine
	}

	vec[w.dims-1] = float32(math.Sqrt(1 - squares))

	return vec
}

// mustRun queries the world's own phrases and returns the decoded payload, failing the test on any error.
func (w *mq1World) mustRun(t *testing.T) mq1Payload {
	t.Helper()

	result := w.query(t, cli.QueryArgs{Phrases: w.phrases, Limit: w.limit})
	if result.err != nil {
		t.Fatalf("query failed: %v", result.err)
	}

	var payload mq1Payload

	err := yaml.Unmarshal([]byte(result.stdout), &payload)
	if err != nil {
		t.Fatalf("decoding the payload: %v", err)
	}

	return payload
}

// plant writes a note and, unless noSidecar, its sidecar.
func (w *mq1World) plant(note mq1Note) {
	notePath := mq1Vault + "/" + note.name + ".md"
	w.files.files[notePath] = []byte(note.content)

	if note.noSidecar {
		return
	}

	modelID := note.modelID
	if modelID == "" {
		modelID = mq1ModelID
	}

	schema := embed.SidecarSchemaVersion
	if note.oldSchema {
		schema = 0
	}

	sidecar := embed.Sidecar{
		SchemaVersion:    schema,
		EmbeddingModelID: modelID,
		Dims:             len(note.situation),
		SituationVector:  note.situation,
		BodyVector:       note.body,
		ContentHash:      embed.ContentHash([]byte(note.content)),
		LastUsed:         note.lastUsed,
	}

	w.files.files[embed.SidecarPath(notePath)] = embed.MarshalSidecar(sidecar)
}

// query runs cli.RunQuery over the world with the given arguments (the vault and chunk directory are the
// world's) and collects the output and the warnings.
func (w *mq1World) query(t *testing.T, args cli.QueryArgs) mq1Result {
	t.Helper()

	encoded, err := chunk.EncodeRecords(w.chunks)
	if err != nil {
		t.Fatalf("encoding the chunk index: %v", err)
	}

	w.files.files[mq1ChunkIndex] = encoded

	var (
		warnings []string
		out      bytes.Buffer
	)

	deps := cli.QueryDeps{
		Scan:     w.files.Scan,
		Read:     w.files.Read,
		Embedder: mq1PhraseEmbedder{dims: w.dims, vectors: w.vectors},
		LogWarning: func(format string, values ...any) {
			warnings = append(warnings, fmt.Sprintf(format, values...))
		},
		ListChunkIndexes: func(string) ([]string, error) { return []string{mq1ChunkIndex}, nil },
		Now:              w.clock,
	}

	if w.noChunkIndex {
		deps.ListChunkIndexes = nil
	}

	args.VaultPath = mq1Vault
	args.ChunksDir = mq1ChunksDir

	runErr := cli.RunQuery(t.Context(), args, deps, &out)

	return mq1Result{err: runErr, stdout: out.String(), warnings: warnings}
}

// mq1ChunkPath is a chunk item's path: <source>#<anchor>.
func mq1ChunkPath(anchor string) string { return mq1Source + "#" + anchor }

// mq1ChunkRecord returns a chunk record with a heading-style anchor (no turn ordinal), so only age and cosine
// decide its score.
func mq1ChunkRecord(anchor string, vec []float32, ingestedAt time.Time) chunk.Record {
	text := "chunk text of " + mq1Source + " " + anchor

	return chunk.Record{
		Source:      mq1Source,
		Anchor:      anchor,
		ContentHash: chunk.HashText(text),
		Text:        text,
		Vector:      vec,
		IngestedAt:  ingestedAt,
	}
}

// mq1FactContent returns a fact note's file text with the given body.
func mq1FactContent(body string) string {
	return "---\ntype: fact\nsituation: a situation\n---\n\n" + body + "\n"
}

// mq1RampAnchor names the index-th chunk of a ramp; the zero-padded number keeps the anchors in score order.
func mq1RampAnchor(index int) string { return fmt.Sprintf("section-%03d", index) }

// newMq1World returns an empty world whose phrase i embeds to the unit vector of axis i.
func newMq1World(dims int, phrases ...string) *mq1World {
	vectors := make(map[string][]float32, len(phrases))

	for axis, phrase := range phrases {
		vec := make([]float32, dims)
		vec[axis] = 1
		vectors[phrase] = vec
	}

	return &mq1World{files: newInMemoryFS(), dims: dims, phrases: phrases, vectors: vectors}
}
