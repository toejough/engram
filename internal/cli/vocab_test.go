package cli_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// ── Unit 4: dual-channel writer (WriteVocabAssignment) ───────────────────────

// TestAmendRoundTrip_VocabKey_PreservedAfterField is the RED→GREEN test for
// adding the `vocab` field to the typed frontmatter structs. Before the field
// is added, `amend` silently drops `vocab:` from the frontmatter. After it is
// added, the key round-trips.
func TestAmendRoundTrip_VocabKey_PreservedAfterField(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	noteContent := []byte(
		"---\ntype: fact\ntier: L2\nsituation: ctx\nsubject: A\npredicate: has\nobject: B\n" +
			"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\nvocab: [eval-methodology, scope-discipline]\n---\n\n" +
			"Information learned: when in ctx, A has B.\n\n" +
			"Vocab: [[vocab.eval-methodology]], [[vocab.scope-discipline]]\n",
	)

	var written []byte

	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:  func(string) ([]byte, error) { return noteContent, nil },
		Write: func(_ string, data []byte) error { written = data; return nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{"fake-source#turn-1": true}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.AmendArgs{
		Vault:        "/vault",
		Target:       "1aa",
		ChunkSources: []string{"fake-source#turn-1"},
	}

	var buf strings.Builder

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(written)).To(ContainSubstring("vocab:"),
		"vocab: frontmatter key must survive a round-trip through amend")
}

func TestAssignVocabTerms_BelowFloor_NilResult(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	bodyVec := []float32{1.0, 0.0}
	terms := []cli.TermWithVector{
		// cosine([1,0],[0.20,0.9798]) ≈ 0.20 < floor 0.30 → filtered out.
		{Term: "foo", Vector: []float32{0.20, 0.9798}},
	}

	got := cli.AssignVocabTerms(bodyVec, terms, 0.30)
	g.Expect(got).To(BeNil(), "no terms above floor must return nil")
}

func TestAssignVocabTerms_EmptyBodyVec_NilResult(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	terms := []cli.TermWithVector{{Term: "foo", Vector: []float32{1.0}}}
	got := cli.AssignVocabTerms(nil, terms, 0.30)
	g.Expect(got).To(BeNil())
}

func TestAssignVocabTerms_EmptyTerms_NilResult(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := cli.AssignVocabTerms([]float32{1.0}, nil, 0.30)
	g.Expect(got).To(BeNil())
}

// ── Unit 3: write-time assigner (AssignVocabTerms) ───────────────────────────

func TestAssignVocabTerms_FourQualifying_CapsAtTopThree(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Four terms qualify; plain top-3 (the sweep-chosen K) caps the selection
	// at the three highest cosines, in descending order.
	bodyVec := []float32{1.0, 0.0}
	terms := []cli.TermWithVector{
		{Term: "alpha", Vector: []float32{0.90, 0.4359}}, // cosine ≈ 0.90
		{Term: "beta", Vector: []float32{0.80, 0.60}},    // cosine = 0.80
		{Term: "gamma", Vector: []float32{0.79, 0.6131}}, // cosine ≈ 0.79
		{Term: "delta", Vector: []float32{0.50, 0.8660}}, // cosine = 0.50
	}

	got := cli.AssignVocabTerms(bodyVec, terms, 0.30)

	g.Expect(got).To(HaveLen(3), "selection caps at top-3")

	if got == nil {
		return
	}

	g.Expect(got).To(Equal([]string{"alpha", "beta", "gamma"}))
}

func TestAssignVocabTerms_OnlyOneQualifies_ReturnsSingle(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Unit vectors at different angles from bodyVec=[1,0] to produce meaningful
	// cosine similarities. [a, b] where a²+b²≈1 gives cosine≈a with bodyVec.
	bodyVec := []float32{1.0, 0.0}
	terms := []cli.TermWithVector{
		// cosine([1,0],[0.90,0.4359]) ≈ 0.90 ≥ floor 0.30 → qualifies.
		{Term: "alpha", Vector: []float32{0.90, 0.4359}},
		// cosine([1,0],[0.10,0.9950]) ≈ 0.10 < floor 0.30 → filtered out.
		{Term: "below", Vector: []float32{0.10, 0.9950}},
	}

	got := cli.AssignVocabTerms(bodyVec, terms, 0.30)

	g.Expect(got).To(HaveLen(1))

	if got == nil {
		return
	}

	g.Expect(got[0]).To(Equal("alpha"))
}

func TestAssignVocabTerms_ThreeQualifying_TakesAllThree(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// All three qualify; plain top-3 takes all of them regardless of the
	// cosine gap between 2nd and 3rd (no close-3rd rider in the K3 config).
	bodyVec := []float32{1.0, 0.0, 0.0}
	terms := []cli.TermWithVector{
		{Term: "eval-methodology", Vector: []float32{0.95, 0.1, 0.0}},
		{Term: "scope-discipline", Vector: []float32{0.85, 0.2, 0.0}},
		{Term: "verification", Vector: []float32{0.75, 0.3, 0.0}},
	}

	got := cli.AssignVocabTerms(bodyVec, terms, 0.30)

	g.Expect(got).To(HaveLen(3), "all floor-passing terms up to 3 are selected")

	if got == nil {
		return
	}

	g.Expect(got[0]).To(Equal("eval-methodology"), "highest cosine term should be first")
	g.Expect(got[1]).To(Equal("scope-discipline"), "second cosine term should be second")
	g.Expect(got[2]).To(Equal("verification"), "third cosine term should be third")
}

func TestAssignVocabTerms_TwoQualifyingOneBelowFloor_TakesTwo(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Third term does not meet the floor, so only top-2 are returned.
	bodyVec := []float32{1.0, 0.0}
	terms := []cli.TermWithVector{
		{Term: "alpha", Vector: []float32{0.90, 0.4359}}, // cosine ≈ 0.90 ≥ floor
		{Term: "beta", Vector: []float32{0.80, 0.60}},    // cosine = 0.80 ≥ floor
		{Term: "gamma", Vector: []float32{0.20, 0.9798}}, // cosine = 0.20 < floor 0.30
	}

	got := cli.AssignVocabTerms(bodyVec, terms, 0.30)

	g.Expect(got).To(HaveLen(2), "only floor-passing terms are selected")
}

func TestIsVocabKind_TypeFact_False(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\nsituation: testing\n---\n\nbody\n"
	g.Expect(cli.ExportIsVocabKind(content)).To(BeFalse())
}

func TestIsVocabKind_TypeFeedback_False(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: feedback\nsituation: testing\n---\n\nbody\n"
	g.Expect(cli.ExportIsVocabKind(content)).To(BeFalse())
}

func TestIsVocabKind_TypeVocabIndex_True(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: vocab-index\nvocab_version: 1.0\n---\n\n[[vocab.foo]] — foo — 3 members\n"
	g.Expect(cli.ExportIsVocabKind(content)).To(BeTrue())
}

// ── Unit 1: term-note model (isVocabKind) ─────────────────────────────────────

func TestIsVocabKind_TypeVocab_True(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: vocab\nterm: eval-methodology\ndescription: how we evaluate.\n---\n\nhow we evaluate.\n"
	g.Expect(cli.ExportIsVocabKind(content)).To(BeTrue())
}

func TestParseVocabFrontmatter_InvalidYAML_Errors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Tabs in YAML frontmatter are illegal per the YAML spec.
	input := []byte("type: vocab\n\tterm: broken-indent\n")

	_, err := cli.ParseVocabFrontmatter(input)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("parsing vocab frontmatter"))
}

func TestParseVocabFrontmatter_Valid(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	input := []byte("type: vocab\nterm: eval-methodology\ndescription: how we evaluate.\n")

	doc, err := cli.ParseVocabFrontmatter(input)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(doc.Type).To(Equal("vocab"))
	g.Expect(doc.Term).To(Equal("eval-methodology"))
	g.Expect(doc.Description).To(Equal("how we evaluate."))
}

// TestVocabAssignment_KeepsSidecarStateOK is the write→embed→assign→state
// round-trip: a note embedded BEFORE vocab assignment must still classify
// StateOK afterwards — the machine-written Vocab: line and vocab: frontmatter
// key are channel content, not body text, so the assignment write must not
// stale the sidecar (otherwise `embed apply --stale` re-embeds every assigned
// note with [[vocab.…]] wikilink noise baked into its body vector).
func TestVocabAssignment_KeepsSidecarStateOK(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const modelID = "test-model"

	preAssign := "---\ntype: feedback\nsituation: wiring a Go CLI\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\n" +
		"Lesson learned: when wiring a Go CLI, a.\n"

	// "Embed" the pre-assignment note: sidecar carries its ContentHash.
	sidecar := embed.Sidecar{
		SchemaVersion:    1,
		EmbeddingModelID: modelID,
		Dims:             2,
		SituationVector:  []float32{1, 0},
		BodyVector:       []float32{1, 0},
		ContentHash:      embed.ContentHash([]byte(preAssign)),
	}

	// Assign vocab AFTER embedding — the write-time assignment path.
	postAssign := cli.WriteVocabAssignment(preAssign, []string{"eval-methodology", "retrieval-design"})
	g.Expect(postAssign).NotTo(Equal(preAssign), "assignment must have written both channels")

	filesystem := &fakeStateFS{files: map[string][]byte{
		"/vault/1aa.note.md":       []byte(postAssign),
		"/vault/1aa.note.vec.json": embed.MarshalSidecar(sidecar),
	}}

	g.Expect(embed.ComputeState(filesystem, "/vault/1aa.note.md", modelID)).
		To(Equal(embed.StateOK),
			"vocab assignment must not stale the sidecar (Vocab: line excluded from body hash)")
}

// ── Unit 2: exclusion seam ────────────────────────────────────────────────────

// TestVocabNote_ExcludedFromFloorPromotion proves that a vocab note is NOT
// promoted by the note-floor reservation even when it scores above the floor.
// This is the RED test for the isFloorQualifyingNote exclusion site.
func TestVocabNote_ExcludedFromFloorPromotion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const vocabPath = "vault/vocab.eval-methodology.md"

	const normalPath = "vault/1aa.2026-01-01.note.md"

	vocabContent := "---\ntype: vocab\nterm: eval-methodology\n---\n\nevaluation methodology content\n"
	normalContent := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n---\n\n" +
		"Lesson learned: when ctx, a.\n\n"

	vocabNote := cli.ExportNewScoredCandidateWithContent(vocabPath, 0.45, 0.45, vocabContent)
	normalNote := cli.ExportNewScoredCandidateWithContent(normalPath, 0.30, 0.30, normalContent)

	// 30 chunks scoring above both notes — enough to fill matchPhraseLimit entirely,
	// so both notes are below the cap and need floor promotion to survive.
	chunks := make([]cli.ExportScoredChunk, 30)
	for i := range chunks {
		rec := chunk.Record{Source: "/s/x.jsonl", Anchor: "turn-" + strconv.Itoa(i+1)}
		chunks[i] = cli.ExportNewScoredChunk(rec, 0.50)
	}

	keys := cli.ExportMergePhraseIntoUnion(
		[]cli.ExportScoredCandidate{vocabNote, normalNote},
		chunks,
	)

	// The vocab note must NOT survive — it must not occupy a floor slot.
	// The normal note (with the floor active) should survive.
	g.Expect(keys).NotTo(ContainElement(vocabPath), "vocab note must not be promoted by the note floor")
}

// TestVocabNote_ExcludedWhenOnlyItem proves that a vocab note is excluded even
// when it would be the only item in the matched set (applyFloorAndCap site).
func TestVocabNote_ExcludedWhenOnlyItem(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const vocabPath = "vault/vocab.eval-methodology.md"

	vocabContent := "---\ntype: vocab\nterm: eval-methodology\n---\n\nevaluation methodology content\n"

	vocabNote := cli.ExportNewScoredCandidateWithContent(vocabPath, 0.80, 0.80, vocabContent)

	keys := cli.ExportMergePhraseIntoUnion(
		[]cli.ExportScoredCandidate{vocabNote},
		nil,
	)

	g.Expect(keys).NotTo(ContainElement(vocabPath), "vocab note must not enter the matched set even as sole item")
}

// TestWriteVocabAssignment_BlockStyleVocabKey_Removed proves that
// WriteVocabAssignment can strip a YAML block-sequence vocab: key
// (multi-line "  - term" form) from the frontmatter, exercising the
// continuation-line loop in removeYAMLKey.
func TestWriteVocabAssignment_BlockStyleVocabKey_Removed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: feedback\nsituation: ctx\naction: do it\nvocab:\n  - old-term-a\n  - old-term-b\n" +
		"---\n\nLesson learned: when ctx, do it.\n\n"

	got := cli.WriteVocabAssignment(content, nil)

	g.Expect(got).NotTo(ContainSubstring("vocab:"),
		"block-style vocab: key must be removed when no terms")
	g.Expect(got).NotTo(ContainSubstring("old-term"),
		"old terms must be removed")
}

func TestWriteVocabAssignment_EmptyTerms_RemovesBothChannels(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: feedback\nsituation: ctx\naction: do it\nluhmann: \"1aa\"\n" +
		"created: 2026-01-01\nsource: test\nvocab: [eval-methodology]\n---\n\n" +
		"Lesson learned: when ctx, do it.\n\nVocab: [[vocab.eval-methodology]]\n"

	got := cli.WriteVocabAssignment(content, nil)

	g.Expect(got).NotTo(ContainSubstring("vocab:"), "vocab: frontmatter key must be removed when no terms")
	g.Expect(got).NotTo(ContainSubstring("Vocab:"), "Vocab: body line must be removed when no terms")
}

func TestWriteVocabAssignment_Idempotent_ReplaceWholeList(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: feedback\nsituation: ctx\naction: do it\nluhmann: \"1aa\"\n" +
		"created: 2026-01-01\nsource: test\nvocab: [old-term]\n---\n\n" +
		"Lesson learned: when ctx, do it.\n\nVocab: [[vocab.old-term]]\n"
	newTerms := []string{"new-term-a", "new-term-b"}

	got := cli.WriteVocabAssignment(content, newTerms)

	g.Expect(got).NotTo(ContainSubstring("old-term"), "old term must be replaced")
	g.Expect(got).To(ContainSubstring("new-term-a"), "new terms must be present")
	g.Expect(got).To(ContainSubstring("new-term-b"), "new terms must be present")
	g.Expect(strings.Count(got, "vocab:")).To(Equal(1), "exactly one vocab: key in frontmatter")
	g.Expect(strings.Count(got, "Vocab:")).To(Equal(1), "exactly one Vocab: line in body")
}

func TestWriteVocabAssignment_WritesBothChannels(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: feedback\nsituation: ctx\naction: do it\nluhmann: \"1aa\"\n" +
		"created: 2026-01-01\nsource: test\n---\n\nLesson learned: when ctx, do it.\n\n"
	terms := []string{"eval-methodology", "scope-discipline"}

	got := cli.WriteVocabAssignment(content, terms)

	g.Expect(got).To(ContainSubstring("vocab: [eval-methodology, scope-discipline]"),
		"frontmatter vocab list must be written")
	g.Expect(got).To(ContainSubstring("Vocab: [[vocab.eval-methodology]], [[vocab.scope-discipline]]"),
		"body Vocab wikilink line must be written")
}

// fakeStateFS serves ComputeState reads from a map (note + sidecar paths).
type fakeStateFS struct {
	files map[string][]byte
}

func (f *fakeStateFS) ReadFile(path string) ([]byte, error) {
	if data, ok := f.files[path]; ok {
		return data, nil
	}

	return nil, &testNotFoundError{path: path}
}
