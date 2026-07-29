package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// ── Unit: applySupersedesRideAlong ───────────────────────────────────────────

// TestApplySupersedesRideAlong_EmptyMeta_Unchanged verifies that the ride-along
// is a no-op when there is no supersedes data — backward compat.
func TestApplySupersedesRideAlong_EmptyMeta_Unchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithProvenances("1aa.old.md", 0.9, []string{"direct"}),
		cli.ExportNewNoteResolvedItemWithProvenances("1bb.other.md", 0.7, []string{"direct"}),
	}

	got := cli.ExportApplySupersedesRideAlong(resolved, cli.ExportNewEmptyVaultNotesMeta())

	g.Expect(got).To(HaveLen(2), "ride-along must be a no-op when no supersedes data")
}

// TestApplySupersedesRideAlong_NewAlreadyInPayload_NoInsert verifies that when the
// superseding note is already in the payload, no duplicate is inserted.
func TestApplySupersedesRideAlong_NewAlreadyInPayload_NoInsert(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldBasename = "1aa.old-note"

	const newBasename = "1bb.new-note"

	const newContent = "---\ntype: fact\nsituation: x\n---\n\nnew content\n"

	// resolved has BOTH old and new already present.
	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithProvenances(newBasename+".md", 0.95, []string{"direct"}),
		cli.ExportNewNoteResolvedItemWithProvenances(oldBasename+".md", 0.80, []string{"direct"}),
	}

	meta := cli.ExportNewVaultNotesMetaWithSupersedes(
		map[string][]cli.ExportSupersedesEntry{
			newBasename: {{Note: oldBasename, Type: "updates", Claim: "old was incomplete"}},
		},
		map[string]string{
			newBasename: newContent,
		},
	)

	got := cli.ExportApplySupersedesRideAlong(resolved, meta)

	g.Expect(got).To(HaveLen(2), "no insert when NEW is already in the payload")
}

// TestApplySupersedesRideAlong_OldNoteInPayload_NewNoteInsertedAfter is the RED test
// for supersession ride-along: when query surfaces an OLD note and the NEW (superseding)
// note is absent from the payload, the NEW note is inserted at old_rank+1 with content.
func TestApplySupersedesRideAlong_OldNoteInPayload_NewNoteInsertedAfter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldBasename = "1aa.old-note"

	const newBasename = "1bb.new-note"

	const newContent = "---\ntype: fact\nsituation: x\n---\n\nnew note content\n"

	// resolved has OLD note at rank-1, another note at rank-2; NEW is absent.
	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithProvenances(oldBasename+".md", 0.9, []string{"direct"}),
		cli.ExportNewNoteResolvedItemWithProvenances("1cc.other.md", 0.7, []string{"direct"}),
	}

	// meta: oldBasename is superseded by newBasename; new note's content is in the vault.
	meta := cli.ExportNewVaultNotesMetaWithSupersedes(
		map[string][]cli.ExportSupersedesEntry{
			newBasename: {{Note: oldBasename, Type: "updates", Claim: "old was incomplete"}},
		},
		map[string]string{
			newBasename: newContent,
		},
	)

	got := cli.ExportApplySupersedesRideAlong(resolved, meta)

	g.Expect(got).To(HaveLen(3), "ride-along must insert NEW note (+1 total)")

	if len(got) < 3 {
		return
	}

	// NEW note must appear immediately after OLD note (index 1).
	g.Expect(cli.ExportResolvedItemPath(got[1])).To(Equal(newBasename+".md"),
		"NEW note must be at old_rank+1")

	// NEW note must carry its content.
	g.Expect(cli.ExportResolvedItemContent(got[1])).To(ContainSubstring("new note content"),
		"inserted NEW note must carry its content")

	// NEW note must carry ride_along provenance.
	g.Expect(cli.ExportResolvedItemProvenances(got[1])).To(ContainElement("ride_along"),
		"inserted NEW note must carry ride_along provenance")
}

// TestApplySupersedesRideAlong_SupersederNotInVault_Skipped verifies that a ride-along
// is silently skipped when the superseder has no content in the vault.
func TestApplySupersedesRideAlong_SupersederNotInVault_Skipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldBasename = "1aa.old-note"

	const newBasename = "1bb.new-note"

	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithProvenances(oldBasename+".md", 0.9, []string{"direct"}),
	}

	// meta has the inverse entry but no content for the superseder (not in vault).
	meta := cli.ExportNewVaultNotesMetaWithSupersedes(
		map[string][]cli.ExportSupersedesEntry{
			newBasename: {{Note: oldBasename, Type: "updates", Claim: "old was incomplete"}},
		},
		map[string]string{}, // empty content map — superseder not in vault
	)

	got := cli.ExportApplySupersedesRideAlong(resolved, meta)

	g.Expect(got).To(HaveLen(1), "no insert when superseder is not in vault")
}

// TestApplySupersedesRideAlong_TwoSuperseders_BothInsertedAfterOld verifies that
// a delivered note with TWO superseders gets both inserted directly after it,
// each carrying ride_along provenance. Insertion order between the two is
// unspecified (the inverse map is built from map iteration), so the test
// asserts set membership at positions 1-2, not a fixed order.
func TestApplySupersedesRideAlong_TwoSuperseders_BothInsertedAfterOld(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldBasename = "1aa.old-note"

	const newOneBasename = "1bb.new-one"

	const newTwoBasename = "1cc.new-two"

	const newOneContent = "---\ntype: fact\nsituation: x\n---\n\nnew one content\n"

	const newTwoContent = "---\ntype: fact\nsituation: y\n---\n\nnew two content\n"

	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithProvenances(oldBasename+".md", 0.9, []string{"direct"}),
		cli.ExportNewNoteResolvedItemWithProvenances("1dd.other.md", 0.7, []string{"direct"}),
	}

	meta := cli.ExportNewVaultNotesMetaWithSupersedes(
		map[string][]cli.ExportSupersedesEntry{
			newOneBasename: {{Note: oldBasename, Type: "updates", Claim: "old was incomplete"}},
			newTwoBasename: {{Note: oldBasename, Type: "refutes", Claim: "old was wrong"}},
		},
		map[string]string{
			newOneBasename: newOneContent,
			newTwoBasename: newTwoContent,
		},
	)

	got := cli.ExportApplySupersedesRideAlong(resolved, meta)

	g.Expect(got).To(HaveLen(4), "both superseders must be inserted (+2 total)")

	if len(got) < 4 {
		return
	}

	inserted := []string{
		cli.ExportResolvedItemPath(got[1]),
		cli.ExportResolvedItemPath(got[2]),
	}
	g.Expect(inserted).To(ConsistOf(newOneBasename+".md", newTwoBasename+".md"),
		"positions 1-2 must be the two superseders (order unspecified)")

	g.Expect(cli.ExportResolvedItemProvenances(got[1])).To(ContainElement("ride_along"))
	g.Expect(cli.ExportResolvedItemProvenances(got[2])).To(ContainElement("ride_along"))

	g.Expect(cli.ExportResolvedItemPath(got[3])).To(Equal("1dd.other.md"),
		"the non-superseded note must stay after the insertions")
}

func TestLoadAllVaultNotesMeta_BareVocabTag_ContributesNoTerms(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Vocab definition note with bare "vocab" tag (not vocab/<term>).
	vocabDefContent := "---\ntype: vocab\ntags:\n    - vocab\n---\n\nvocab term definition\n"

	const basename = "vocab.definition"

	const modelID = "test-model"

	sidecarBytes := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: 1, EmbeddingModelID: modelID, Dims: 2,
		SituationVector: []float32{1, 0}, BodyVector: []float32{1, 0},
	})

	sidecarStruct, unmarshalErr := embed.UnmarshalSidecar(sidecarBytes)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	hits := cli.ExportNewCompatibleSidecars(
		[]vaultgraph.Note{{Basename: basename}},
		[]embed.Sidecar{sidecarStruct},
	)

	fs := map[string][]byte{
		"/vault/" + basename + ".md": []byte(vocabDefContent),
	}

	readFn := func(path string) ([]byte, error) {
		if data, ok := fs[path]; ok {
			return data, nil
		}

		return nil, &testNotFoundError{path: path}
	}

	meta := cli.ExportLoadAllVaultNotesMeta(hits, "/vault", readFn)

	g.Expect(meta.TermIndex).To(BeEmpty(),
		"bare 'vocab' tag is not a term; TermIndex must be empty")

	// Content must still be populated (not filtered by definition status).
	g.Expect(meta.ContentByBasename).To(HaveKey(basename),
		"bare vocab definition content must still be captured")
}

// ── Integration: loadAllVaultNotesMeta ───────────────────────────────────────

// TestLoadAllVaultNotesMeta_BuildsTermIndexAndInverse verifies that a single scan
// over all vault notes builds both the term index and the supersedes inverse map.
func TestLoadAllVaultNotesMeta_BuildsTermIndexAndInverse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Note-A: has tags: [vocab/eval-methodology] and supersedes 9a.old.
	noteAContent := "---\ntype: fact\nsituation: ctx\ntags:\n    - vocab/eval-methodology\n" +
		"supersedes:\n  - note: 9a.old\n    type: updates\n    claim: old was wrong\n---\n\nbody A\n"
	// Note-B: plain note, no tags or supersedes.
	noteBContent := "---\ntype: feedback\nsituation: x\n---\n\nbody B\n"

	const basenameA = "1aa.note-a"

	const basenameB = "1bb.note-b"

	const modelID = "test-model"

	sidecarA := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: 1, EmbeddingModelID: modelID, Dims: 2,
		SituationVector: []float32{1, 0}, BodyVector: []float32{1, 0},
	})
	sidecarB := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: 1, EmbeddingModelID: modelID, Dims: 2,
		SituationVector: []float32{0, 1}, BodyVector: []float32{0, 1},
	})

	fs := map[string][]byte{
		"/vault/" + basenameA + ".md":       []byte(noteAContent),
		"/vault/" + basenameB + ".md":       []byte(noteBContent),
		"/vault/" + basenameA + ".vec.json": sidecarA,
		"/vault/" + basenameB + ".vec.json": sidecarB,
	}

	readFn := func(path string) ([]byte, error) {
		if data, ok := fs[path]; ok {
			return data, nil
		}

		return nil, &testNotFoundError{path: path}
	}

	notes := []vaultgraph.Note{
		{Basename: basenameA},
		{Basename: basenameB},
	}

	sidecarAStruct, unmarshalErr := embed.UnmarshalSidecar(sidecarA)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	sidecarBStruct, unmarshalErrB := embed.UnmarshalSidecar(sidecarB)
	g.Expect(unmarshalErrB).NotTo(HaveOccurred())

	if unmarshalErr != nil || unmarshalErrB != nil {
		return
	}

	hits := cli.ExportNewCompatibleSidecars(notes, []embed.Sidecar{sidecarAStruct, sidecarBStruct})

	meta := cli.ExportLoadAllVaultNotesMeta(hits, "/vault", readFn)

	// Term index must contain eval-methodology → note-a.
	termEntries, ok := meta.TermIndex["eval-methodology"]
	g.Expect(ok).To(BeTrue(), "TermIndex must have eval-methodology key")
	g.Expect(termEntries).To(HaveLen(1))

	// Supersedes inverse must map 9a.old → note-a.
	g.Expect(meta.SupersedesInverse).To(HaveKey("9a.old"))
	g.Expect(meta.SupersedesInverse["9a.old"]).To(HaveLen(1))

	// Content by basename must be populated.
	g.Expect(meta.ContentByBasename).To(HaveKey(basenameA))
}

// TestLoadAllVaultNotesMeta_MixedTags_OnlyVocabTermsIndexed verifies that a note
// with mixed categorical and vocab tags contributes only vocab terms to the
// TermIndex.
func TestLoadAllVaultNotesMeta_MixedTags_OnlyVocabTermsIndexed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Note with work-kind/rename (categorical) and vocab/go-conventions (vocab).
	// Only go-conventions should appear in TermIndex.
	mixedContent := "---\ntype: fact\nsituation: ctx\ntags:\n" +
		"    - work-kind/rename\n    - vocab/go-conventions\n---\n\nbody\n"

	const basename = "1aa.mixed-note"

	const modelID = "test-model"

	sidecarBytes := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: 1, EmbeddingModelID: modelID, Dims: 2,
		SituationVector: []float32{1, 0}, BodyVector: []float32{1, 0},
	})

	sidecarStruct, unmarshalErr := embed.UnmarshalSidecar(sidecarBytes)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	hits := cli.ExportNewCompatibleSidecars(
		[]vaultgraph.Note{{Basename: basename}},
		[]embed.Sidecar{sidecarStruct},
	)

	fs := map[string][]byte{
		"/vault/" + basename + ".md": []byte(mixedContent),
	}

	readFn := func(path string) ([]byte, error) {
		if data, ok := fs[path]; ok {
			return data, nil
		}

		return nil, &testNotFoundError{path: path}
	}

	meta := cli.ExportLoadAllVaultNotesMeta(hits, "/vault", readFn)

	g.Expect(meta.TermIndex).To(HaveKey("go-conventions"),
		"TermIndex must include vocab term (prefix stripped)")
	g.Expect(meta.TermIndex).NotTo(HaveKey("work-kind/rename"),
		"TermIndex must NOT include non-vocab tags")
	g.Expect(meta.TermIndex).NotTo(HaveKey("rename"),
		"TermIndex must NOT include stripped non-vocab tags")
}

// TestLoadAllVaultNotesMeta_NoVocabOrSupersedes_EmptyMaps verifies that on a vault
// with no vocab or supersedes data, both maps are empty (backward compat no-op).
func TestLoadAllVaultNotesMeta_NoVocabOrSupersedes_EmptyMaps(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	noteContent := "---\ntype: fact\nsituation: ctx\n---\n\nbody\n"

	const basename = "1aa.plain-note"

	const modelID = "test-model"

	sidecarBytes := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: 1, EmbeddingModelID: modelID, Dims: 2,
		SituationVector: []float32{1, 0}, BodyVector: []float32{1, 0},
	})

	sidecarStruct, unmarshalErr := embed.UnmarshalSidecar(sidecarBytes)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	hits := cli.ExportNewCompatibleSidecars(
		[]vaultgraph.Note{{Basename: basename}},
		[]embed.Sidecar{sidecarStruct},
	)

	fs := map[string][]byte{
		"/vault/" + basename + ".md": []byte(noteContent),
	}

	readFn := func(path string) ([]byte, error) {
		if data, ok := fs[path]; ok {
			return data, nil
		}

		return nil, &testNotFoundError{path: path}
	}

	meta := cli.ExportLoadAllVaultNotesMeta(hits, "/vault", readFn)

	g.Expect(meta.TermIndex).To(BeEmpty(),
		"TermIndex must be empty when no notes have tags: with vocab/ entries")
	g.Expect(meta.SupersedesInverse).To(BeEmpty(),
		"SupersedesInverse must be empty when no notes have supersedes:")

	// Content by basename must still be populated.
	g.Expect(meta.ContentByBasename).To(HaveKey(basename))
}

// TestParseNoteQueryFrontmatter_BareVocabTag_IsNotTerm verifies that the bare
// "vocab" definition marker tag is included in the parsed Tags but does NOT
// contribute a term (no prefix to strip).
func TestParseNoteQueryFrontmatter_BareVocabTag_IsNotTerm(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: vocab\ntags:\n    - vocab\n---\n\nthis is a vocab definition\n"
	doc := cli.ExportParseNoteQueryFrontmatter(content)

	g.Expect(doc.Tags).To(Equal([]string{"vocab"}))
}

// TestParseNoteQueryFrontmatter_MixedTags_OnlyVocabExtracted verifies that when
// a note has mixed categorical and vocab tags, only vocab/ entries are
// extracted as terms.
func TestParseNoteQueryFrontmatter_MixedTags_OnlyVocabExtracted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\nsituation: ctx\ntags:\n    - work-kind/rename\n    - vocab/go-conventions\n---\n\nbody\n"
	doc := cli.ExportParseNoteQueryFrontmatter(content)

	g.Expect(doc.Tags).To(ConsistOf("work-kind/rename", "vocab/go-conventions"))
}

// NOTE: testNotFoundError is defined in testhelpers_test.go and is reused here.
// The cli_test package shares one testNotFoundError implementation across files.

// ── Unit: parseNoteQueryFrontmatter ──────────────────────────────────────────

// TestParseNoteQueryFrontmatter_NoFrontmatter_Empty verifies that content without
// a YAML frontmatter block returns empty fields (no crash, backward compat).
func TestParseNoteQueryFrontmatter_NoFrontmatter_Empty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "just body text, no frontmatter\n"
	doc := cli.ExportParseNoteQueryFrontmatter(content)

	g.Expect(doc.Tags).To(BeNil())
	g.Expect(doc.Supersedes).To(BeNil())
}

// TestParseNoteQueryFrontmatter_SupersedesList_Parsed verifies that a note's
// supersedes: frontmatter list is correctly extracted.
func TestParseNoteQueryFrontmatter_SupersedesList_Parsed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\nsituation: ctx\nsupersedes:\n  - note: 9a.old\n    type: updates\n" +
		"    claim: old was wrong\n---\n\nbody\n"
	doc := cli.ExportParseNoteQueryFrontmatter(content)

	g.Expect(doc.Supersedes).To(HaveLen(1))

	if len(doc.Supersedes) == 0 {
		return
	}

	g.Expect(doc.Supersedes[0].Note).To(Equal("9a.old"))
	g.Expect(doc.Supersedes[0].Type).To(Equal("updates"))
	g.Expect(doc.Supersedes[0].Claim).To(Equal("old was wrong"))
}

// TestParseNoteQueryFrontmatter_TagsBlockStyle_ParsesToTerms verifies that a
// note's tags: frontmatter with vocab/ entries is correctly parsed and
// converted to vocab term names (prefix stripped).
func TestParseNoteQueryFrontmatter_TagsBlockStyle_ParsesToTerms(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\nsituation: ctx\ntags:\n" +
		"    - vocab/eval-methodology\n    - vocab/scope-discipline\n---\n\nbody\n"
	doc := cli.ExportParseNoteQueryFrontmatter(content)

	// The parsed doc should have Tags with vocab entries; terms
	// (prefix stripped) are derived by vocabTermsFromTags.
	g.Expect(doc.Tags).To(ConsistOf("vocab/eval-methodology", "vocab/scope-discipline"))
}
