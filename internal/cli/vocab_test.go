package cli_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// ── Task 2: bare-vocab definition exemption ──────────────────────────────────

// TestApplyVocabAssignmentCore_SkipsDefinitionNotes verifies that
// applyVocabAssignmentCore never writes a term assignment to a note whose
// content is itself a bare-vocab DEFINITION note — a definition must never
// acquire its own term tag, even when its body vector would otherwise
// qualify under the floor.
func TestApplyVocabAssignmentCore_SkipsDefinitionNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	definition := "---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines the retrieval-design term.\n"
	sidecar := mustMarshalSidecarWithBodyVector(t, []float32{1, 0, 0})
	loadTerms := func(string) ([]cli.TermWithVector, error) {
		return []cli.TermWithVector{{Term: "retrieval-design", Vector: []float32{1, 0, 0}}}, nil
	}
	read := func(string) ([]byte, error) { return sidecar, nil }

	wrote := false
	write := func(string, []byte) error { wrote = true; return nil }

	cli.ExportApplyVocabAssignmentCore(loadTerms, read, write, nil, "/v", "/v/n.md", definition, "test")

	g.Expect(wrote).To(BeFalse(), "a definition must never acquire its own term tag")
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

// TestIsVocabDefinitionNote verifies the bare-vocab definition marker:
// a note tagged bare "vocab" is a definition; a note tagged vocab/<term>
// (a member) or an unrelated bare family tag (work-kind) is not.
func TestIsVocabDefinitionNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	definition := "---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines a term.\n"
	member := "---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nA member.\n"
	otherFamily := "---\ntype: fact\ntags:\n    - work-kind\n---\n\nRoute family definition.\n"

	g.Expect(cli.ExportIsVocabDefinitionNote(definition)).To(BeTrue())
	g.Expect(cli.ExportIsVocabDefinitionNote(member)).To(BeFalse())
	g.Expect(cli.ExportIsVocabDefinitionNote(otherFamily)).To(BeFalse())
}

// TestParseTagsFromFrontmatter_EdgeCases pins nil-without-panic semantics for
// an absent key, a key with no value, an empty inline list, and malformed YAML.
func TestParseTagsFromFrontmatter_EdgeCases(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportParseTagsFromFrontmatter("type: fact")).To(BeNil())
	g.Expect(cli.ExportParseTagsFromFrontmatter("type: fact\ntags:")).To(BeNil())
	g.Expect(cli.ExportParseTagsFromFrontmatter("type: fact\ntags: []")).To(BeNil())
	g.Expect(cli.ExportParseTagsFromFrontmatter("type: fact\ntags: [")).To(BeNil())
}

// TestVocabAssignment_KeepsSidecarStateOK is the write→embed→assign→state
// round-trip: a note embedded BEFORE vocab assignment must still classify
// StateOK afterwards — the assigned terms land in the tags: frontmatter
// field, which is neither the situation: field nor the body, so the
// assignment write must not stale the sidecar (otherwise `embed apply
// --stale` re-embeds every assigned note for no reason).
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
	g.Expect(postAssign).NotTo(Equal(preAssign), "assignment must have written the tags")

	filesystem := &fakeStateFS{files: map[string][]byte{
		"/vault/1aa.note.md":       []byte(postAssign),
		"/vault/1aa.note.vec.json": embed.MarshalSidecar(sidecar),
	}}

	g.Expect(embed.ComputeState(filesystem, "/vault/1aa.note.md", modelID)).
		To(Equal(embed.StateOK),
			"vocab assignment must not stale the sidecar (Vocab: line excluded from body hash)")
}

// ── Unit 2: recallability seam (definitions are ordinary notes, #678 Task 6) ─

// TestVocabDefinitionNote_FloorPromoted proves that a bare-vocab-tagged
// definition note (an ordinary fact-shaped note carrying the bare "vocab" tag)
// IS promoted by the note-floor reservation when it scores above the floor —
// the accepted #678 Task 6 behavior change: definitions are ordinary
// recallable notes, no longer excluded via isQueryExcludedKind. Inverse of
// the retired TestVocabNote_ExcludedFromFloorPromotion (isFloorQualifyingNote
// exclusion site).
func TestVocabDefinitionNote_FloorPromoted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const definitionPath = "vault/211.2026-07-10.vocab-eval-methodology-definition.md"

	const normalPath = "vault/1aa.2026-01-01.note.md"

	definitionContent := "---\ntype: fact\nsituation: recalling what the eval-methodology vocab term covers\n" +
		"subject: the eval-methodology vocab term\npredicate: covers\nobject: how we evaluate.\n" +
		"tags:\n    - vocab\n---\n\n" +
		"Information learned: when in recalling what the eval-methodology vocab term covers, " +
		"the eval-methodology vocab term covers how we evaluate.\n"
	normalContent := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n---\n\n" +
		"Lesson learned: when ctx, a.\n\n"

	definitionNote := cli.ExportNewScoredCandidateWithContent(definitionPath, 0.45, 0.45, definitionContent)
	normalNote := cli.ExportNewScoredCandidateWithContent(normalPath, 0.30, 0.30, normalContent)

	// 30 chunks scoring above both notes — enough to fill matchPhraseLimit entirely,
	// so both notes are below the cap and need floor promotion to survive.
	chunks := make([]cli.ExportScoredChunk, 30)
	for i := range chunks {
		rec := chunk.Record{Source: "/s/x.jsonl", Anchor: "turn-" + strconv.Itoa(i+1)}
		chunks[i] = cli.ExportNewScoredChunk(rec, 0.50)
	}

	keys := cli.ExportMergePhraseIntoUnion(
		[]cli.ExportScoredCandidate{definitionNote, normalNote},
		chunks,
	)

	// The definition note MUST survive — it is entitled to a floor slot
	// exactly like the normal note.
	g.Expect(keys).To(ContainElement(definitionPath),
		"a bare-vocab-tagged definition note is promoted by the note floor like any other note")
}

// TestVocabDefinitionNote_ReturnedWhenOnlyItem proves that a bare-vocab-tagged
// definition note is returned even when it is the only item in the matched
// set (mergePhraseIntoUnion's own exclusion site). Inverse of the retired
// TestVocabNote_ExcludedWhenOnlyItem.
func TestVocabDefinitionNote_ReturnedWhenOnlyItem(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const definitionPath = "vault/211.2026-07-10.vocab-eval-methodology-definition.md"

	definitionContent := "---\ntype: fact\nsituation: recalling what the eval-methodology vocab term covers\n" +
		"subject: the eval-methodology vocab term\npredicate: covers\nobject: how we evaluate.\n" +
		"tags:\n    - vocab\n---\n\n" +
		"Information learned: when in recalling what the eval-methodology vocab term covers, " +
		"the eval-methodology vocab term covers how we evaluate.\n"

	definitionNote := cli.ExportNewScoredCandidateWithContent(definitionPath, 0.80, 0.80, definitionContent)

	keys := cli.ExportMergePhraseIntoUnion(
		[]cli.ExportScoredCandidate{definitionNote},
		nil,
	)

	g.Expect(keys).To(ContainElement(definitionPath),
		"a bare-vocab-tagged definition note enters the matched set even as the sole item")
}

// TestVocabTermsFromTags_FiltersAndStripsPrefix locks vocabTermsFromTags'
// contract ahead of its Task 2+ consumers: only vocab/<term> entries are
// returned, prefix-stripped, in order; non-vocab tags and the bare "vocab"
// marker are excluded.
func TestVocabTermsFromTags_FiltersAndStripsPrefix(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	tags := []string{"work-kind/rename", "vocab/retrieval-design", "vocab", "vocab/token-budget"}
	got := cli.ExportVocabTermsFromTags(tags)

	g.Expect(got).To(Equal([]string{"retrieval-design", "token-budget"}))
}

// TestWriteVocabAssignment_BodyIsOpaque proves that a body line starting
// `Vocab: [[vocab.old-term]]` is user prose now — WriteVocabAssignment must
// not parse or strip it; it survives byte-identical through the rewrite.
func TestWriteVocabAssignment_BodyIsOpaque(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := "Body.\n\nVocab: [[vocab.old-term]]\n"
	content := "---\ntype: fact\n---\n\n" + body
	got := cli.WriteVocabAssignment(content, []string{"old-term"})

	g.Expect(strings.HasSuffix(got, body)).To(BeTrue(),
		"the Vocab: body line is user prose now — WriteVocabAssignment must not strip it")
}

// TestWriteVocabAssignment_EmptyTermsNoOtherTagsRemovesTagsKey proves that
// when the vocab/ namespace was the ONLY content of tags:, an empty terms
// list removes the tags: key entirely rather than leaving an empty list.
func TestWriteVocabAssignment_EmptyTermsNoOtherTagsRemovesTagsKey(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\ntags:\n    - vocab/stale-term\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, nil)

	g.Expect(got).NotTo(ContainSubstring("tags:"))
}

// TestWriteVocabAssignment_EmptyTermsRemovesVocabNamespaceOnly proves that an
// empty terms list clears only the vocab/ namespace, leaving other tags intact.
func TestWriteVocabAssignment_EmptyTermsRemovesVocabNamespaceOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\ntags:\n    - work-kind/rename\n    - vocab/stale-term\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, nil)

	g.Expect(got).To(ContainSubstring("tags:\n    - work-kind/rename\n"))
	g.Expect(got).NotTo(ContainSubstring("vocab/"))
}

// TestWriteVocabAssignment_Idempotent proves that applying the same
// assignment twice yields byte-identical output — the vocab/ namespace is
// replaced, never appended, on every call.
func TestWriteVocabAssignment_Idempotent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\ntags:\n    - work-kind/rename\nvocab: [a]\n---\n\nBody.\n\nVocab: [[vocab.a]]\n"
	once := cli.WriteVocabAssignment(content, []string{"a", "b"})
	twice := cli.WriteVocabAssignment(once, []string{"a", "b"})

	g.Expect(twice).To(Equal(once))
}

// TestWriteVocabAssignment_InlineTagsListParsed proves that a pre-existing
// inline-style tags: list ("tags: [a, b]") is parsed correctly, not just the
// block style.
func TestWriteVocabAssignment_InlineTagsListParsed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\ntags: [work-kind/rename, vocab/stale]\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, []string{"fresh"})

	g.Expect(got).To(ContainSubstring("tags:\n    - work-kind/rename\n    - vocab/fresh\n"))
}

// TestWriteVocabAssignment_LearnRendererRoundtripFidelity decodes the
// writer's output — applied on top of a note actually produced by the #674
// learn renderer (ExportRenderFactFrontmatter) — with the same tags-only
// struct shape TestRenderFactFrontmatter_TagsRoundtripFidelity decodes with,
// pinning byte-compatibility between the learn renderer's tags: block style
// and the one WriteVocabAssignment merges into it.
func TestWriteVocabAssignment_LearnRendererRoundtripFidelity(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	when := time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC)
	fields := cli.ExportFactFields{
		Situation: "s", Subject: "a", Predicate: "has", Object: "b",
		Luhmann: "1a", Source: "test", Tags: []string{"work-kind/rename"},
	}

	rendered := cli.ExportRenderFactFrontmatter(fields, when) + cli.ExportRenderFactBody(fields)

	got := cli.WriteVocabAssignment(rendered, []string{"retrieval-design", "token-budget"})

	const delim = "---\n"

	body := strings.TrimPrefix(got, delim)
	end := strings.Index(body, "\n"+delim)
	g.Expect(end).To(BeNumerically(">", -1))

	if end < 0 {
		return
	}

	var doc struct {
		Tags []string `yaml:"tags"`
	}

	unmarshalErr := yaml.Unmarshal([]byte(body[:end+1]), &doc)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	g.Expect(doc.Tags).To(Equal([]string{
		"work-kind/rename", "vocab/retrieval-design", "vocab/token-budget",
	}))
}

// TestWriteVocabAssignment_NoFrontmatterUnchanged proves that content with no
// leading frontmatter block is returned unchanged.
func TestWriteVocabAssignment_NoFrontmatterUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "Just a body, no frontmatter.\n"
	g.Expect(cli.WriteVocabAssignment(content, []string{"a"})).To(Equal(content))
}

// TestWriteVocabAssignment_PreservesCategoricalTags proves that non-vocab
// tags already present survive, in order, ahead of the vocab/ namespace, and
// that a stale vocab/ entry is discarded (namespace is REPLACED, not merged).
func TestWriteVocabAssignment_PreservesCategoricalTags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\ntags:\n    - work-kind/rename\n    - tier/cheap\n    - vocab/stale-term\n" +
		"---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, []string{"retrieval-design"})

	g.Expect(got).To(ContainSubstring(
		"tags:\n    - work-kind/rename\n    - tier/cheap\n    - vocab/retrieval-design\n"))
	g.Expect(got).NotTo(ContainSubstring("vocab/stale-term"))
}

// TestWriteVocabAssignment_PreservesTagsKeyPosition proves the rewritten
// tags: block lands at the ORIGINAL tags: key's line position when other
// frontmatter keys follow it, rather than always being appended at the end
// of the frontmatter block (behavior spec item 4: "at the position of the
// existing tags: key").
func TestWriteVocabAssignment_PreservesTagsKeyPosition(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\ntags:\n    - work-kind/rename\nsource: test\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, []string{"retrieval-design"})

	g.Expect(got).To(Equal(
		"---\ntype: fact\ntags:\n    - work-kind/rename\n    - vocab/retrieval-design\nsource: test\n---\n\nBody.\n"))
}

// TestWriteVocabAssignment_StripsBareVocabMarkerTag proves that a bare
// "vocab" tag entry (the definition marker, not a vocab/<term> entry) is
// filtered by nonVocabTags exactly like a namespaced entry.
func TestWriteVocabAssignment_StripsBareVocabMarkerTag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\ntags:\n    - work-kind/rename\n    - vocab\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, []string{"retrieval-design"})

	g.Expect(got).To(ContainSubstring("tags:\n    - work-kind/rename\n    - vocab/retrieval-design\n"))
	g.Expect(got).NotTo(ContainSubstring("    - vocab\n"))
}

// TestWriteVocabAssignment_TagsRoundtripFidelity is a property test: for
// random non-vocab tag lists and term lists, parseTagsFromFrontmatter of the
// writer's output equals tags ++ map("vocab/"+_, terms), and a second
// application with the same terms is byte-identical (idempotency).
func TestWriteVocabAssignment_TagsRoundtripFidelity(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		tagGen := rapid.StringMatching(`[a-z][a-z0-9-]{0,10}/[a-z][a-z0-9-]{0,10}`).
			Filter(func(s string) bool { return !strings.HasPrefix(s, "vocab/") })
		termGen := rapid.StringMatching(`[a-z][a-z0-9-]{0,10}`)

		const maxGenLen = 4

		tags := rapid.SliceOfN(tagGen, 0, maxGenLen).Draw(rt, "tags")
		terms := rapid.SliceOfN(termGen, 0, maxGenLen).Draw(rt, "terms")

		frontmatterStr := buildTestFrontmatter(tags)
		content := "---\n" + frontmatterStr + "---\n\nBody.\n"
		got := cli.WriteVocabAssignment(content, terms)

		frontmatter := parseAndVerifyFrontmatterDelimiters(rt, got)
		want := buildExpectedTagsForVocabAssignment(tags, terms)
		gotTags := cli.ExportParseTagsFromFrontmatter(frontmatter)

		verifyTagsMatch(rt, gotTags, want)
		verifyIdempotency(rt, got, terms)
	})
}

// TestWriteVocabAssignment_WritesVocabNamespaceTags proves that assigned
// terms land in the vocab/ namespace of the shared tags: list and that
// neither legacy channel (vocab: frontmatter key, Vocab: body line) is written.
func TestWriteVocabAssignment_WritesVocabNamespaceTags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\nsituation: s\n---\n\nBody.\n"
	got := cli.WriteVocabAssignment(content, []string{"retrieval-design", "token-budget"})

	g.Expect(got).To(ContainSubstring("tags:\n    - vocab/retrieval-design\n    - vocab/token-budget\n"))
	g.Expect(got).NotTo(ContainSubstring("\nvocab:"))
	g.Expect(got).NotTo(ContainSubstring("Vocab:"))
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

// buildExpectedTagsForVocabAssignment constructs the expected tag list after assignment.
func buildExpectedTagsForVocabAssignment(tags []string, terms []string) []string {
	want := make([]string, 0, len(tags)+len(terms))
	want = append(want, tags...)

	for _, term := range terms {
		want = append(want, "vocab/"+term)
	}

	return want
}

// buildTestFrontmatter constructs test frontmatter carrying the given tags:
// list. Returns the frontmatter string (without YAML delimiters).
func buildTestFrontmatter(tags []string) string {
	var fmBuilder strings.Builder
	fmBuilder.WriteString("type: fact\n")

	if len(tags) > 0 {
		fmBuilder.WriteString("tags:\n")

		for _, tag := range tags {
			fmBuilder.WriteString("    - " + tag + "\n")
		}
	}

	return fmBuilder.String()
}

// parseAndVerifyFrontmatterDelimiters extracts frontmatter from the output and verifies delimiters.
func parseAndVerifyFrontmatterDelimiters(rt *rapid.T, got string) string {
	const delim = "---\n"

	afterDelim := strings.TrimPrefix(got, delim)

	frontmatter, _, found := strings.Cut(afterDelim, "\n"+delim)
	if !found {
		rt.Fatalf("no closing delimiter in %q", got)
	}

	return frontmatter
}

// verifyIdempotency checks that a second application with the same terms is byte-identical.
func verifyIdempotency(rt *rapid.T, got string, terms []string) {
	twice := cli.WriteVocabAssignment(got, terms)
	if twice != got {
		rt.Fatalf("second application not idempotent:\nfirst:\n%q\nsecond:\n%q", got, twice)
	}
}

// verifyTagsMatch asserts that the parsed tags match the expected tags.
func verifyTagsMatch(rt *rapid.T, gotTags []string, want []string) {
	if len(want) == 0 && len(gotTags) != 0 {
		rt.Fatalf("want no tags, got %v", gotTags)
	}

	if len(want) > 0 && !slices.Equal(gotTags, want) {
		rt.Fatalf("tags: got %v want %v", gotTags, want)
	}
}
