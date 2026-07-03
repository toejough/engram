package cli_test

import (
	"fmt"
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

// TestBuildTagNominations_AlreadyInResults_NotNominated verifies that a note
// already in the ranked items is never re-nominated.
func TestBuildTagNominations_AlreadyInResults_NotNominated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	top3Content := "---\ntype: fact\nsituation: ctx\nvocab: [eval-methodology]\n---\n\nbody A\n"
	alreadyContent := "---\ntype: fact\nsituation: x\nvocab: [eval-methodology]\n---\n\nbody B\n"

	// Both notes in resolved (already-present is a direct hit).
	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithContentAndProvenances(
			"1aa.top-note.md", top3Content, 0.9, []string{"direct"},
		),
		cli.ExportNewNoteResolvedItemWithContentAndProvenances(
			"already-present.md", alreadyContent, 0.7, []string{"direct"},
		),
	}

	meta := cli.ExportNewVaultNotesMetaWithTerms(map[string][]cli.ExportNominationEntry{
		"eval-methodology": {
			{NotePath: "already-present.md", Content: alreadyContent},
		},
	})

	noms := cli.ExportBuildTagNominationsUnit(resolved, meta)

	var foundPaths []string

	for _, candidates := range noms {
		for _, c := range candidates {
			foundPaths = append(foundPaths, c.Path)
		}
	}

	g.Expect(foundPaths).NotTo(ContainElement("already-present.md"),
		"a note already in ranked items must not be nominated again")
}

// ── Unit: buildTagNominations ─────────────────────────────────────────────────

// TestBuildTagNominations_CapExceeded_ReportsAddedAndDropped is the RED test for
// the no-silent-caps rule: when a term's entries exceed nominationCapPerCluster,
// the tally must report how many nominations were kept (added) and how many the
// cap truncated (dropped) — so the payload budget can surface the truncation.
func TestBuildTagNominations_CapExceeded_ReportsAddedAndDropped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// nominationCapPerCluster is 40; overflow by 5 to force truncation.
	const expectedAdded = 40

	const expectedDropped = 5

	const entryCount = expectedAdded + expectedDropped

	top3Content := "---\ntype: fact\nsituation: ctx\nvocab: [eval-methodology]\n---\n\nbody A\n"
	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithContentAndProvenances(
			"1aa.top-note.md", top3Content, 0.9, []string{"direct"},
		),
	}

	entries := make([]cli.ExportNominationEntry, 0, entryCount)
	for i := range entryCount {
		entries = append(entries, cli.ExportNominationEntry{
			NotePath: fmt.Sprintf("nominee-%02d.md", i),
			Content:  "---\ntype: fact\nsituation: x\n---\n\nbody\n",
		})
	}

	meta := cli.ExportNewVaultNotesMetaWithTerms(map[string][]cli.ExportNominationEntry{
		"eval-methodology": entries,
	})

	noms, added, dropped := cli.ExportBuildTagNominationsWithTally(resolved, meta)

	g.Expect(added).To(Equal(expectedAdded), "tally must report nominations kept after the cap")
	g.Expect(dropped).To(Equal(expectedDropped), "tally must report nominations truncated by the cap")

	total := 0
	for _, candidates := range noms {
		total += len(candidates)
	}

	g.Expect(total).To(Equal(expectedAdded), "the map must carry exactly the capped nominations")
}

// TestBuildTagNominations_EmptyMeta_Nil verifies that buildTagNominations returns
// nil when there is no vocab data — backward compat no-op.
func TestBuildTagNominations_EmptyMeta_Nil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithProvenances("note-a.md", 0.9, []string{"direct"}),
	}

	noms := cli.ExportBuildTagNominationsUnit(resolved, cli.ExportNewEmptyVaultNotesMeta())

	g.Expect(noms).To(BeNil())
}

// TestBuildTagNominations_MultiCluster_NomineesKeyedByTriggeringCluster exercises
// the REAL clusterID assignment path (non-empty matchedSet + clusterReport):
// each nominee must be keyed under the cluster of the top-3 note that triggered
// it — not the cluster-0 fallback.
func TestBuildTagNominations_MultiCluster_NomineesKeyedByTriggeringCluster(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	noteAContent := "---\ntype: fact\nsituation: ctx\nvocab: [term-a]\n---\n\nbody A\n"
	noteBContent := "---\ntype: fact\nsituation: ctx\nvocab: [term-b]\n---\n\nbody B\n"

	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithContentAndProvenances(
			"note-a.md", noteAContent, 0.9, []string{"direct"},
		),
		cli.ExportNewNoteResolvedItemWithContentAndProvenances(
			"note-b.md", noteBContent, 0.8, []string{"direct"},
		),
	}

	meta := cli.ExportNewVaultNotesMetaWithTerms(map[string][]cli.ExportNominationEntry{
		"term-a": {{NotePath: "nominee-a.md", Content: "---\ntype: fact\nsituation: p\n---\n\nna\n"}},
		"term-b": {{NotePath: "nominee-b.md", Content: "---\ntype: fact\nsituation: q\n---\n\nnb\n"}},
	})

	// Member index 0 = note-a.md lives in cluster 1; member index 1 = note-b.md
	// lives in cluster 0.
	noms := cli.ExportBuildTagNominationsWithClusters(
		resolved,
		meta,
		[]string{"note-a.md", "note-b.md"},
		[][]int{{1}, {0}},
	)

	g.Expect(noms).To(HaveKey(1), "nominee-a must be keyed under note-a's cluster (1)")
	g.Expect(noms).To(HaveKey(0), "nominee-b must be keyed under note-b's cluster (0)")

	clusterOnePaths := make([]string, 0, len(noms[1]))
	for _, candidate := range noms[1] {
		clusterOnePaths = append(clusterOnePaths, candidate.Path)
	}

	clusterZeroPaths := make([]string, 0, len(noms[0]))
	for _, candidate := range noms[0] {
		clusterZeroPaths = append(clusterZeroPaths, candidate.Path)
	}

	g.Expect(clusterOnePaths).To(ConsistOf("nominee-a.md"))
	g.Expect(clusterZeroPaths).To(ConsistOf("nominee-b.md"))
}

// TestBuildTagNominations_NoTop3Notes_Nil verifies that buildTagNominations returns
// nil when the resolved list has no delivered note items (only chunks).
func TestBuildTagNominations_NoTop3Notes_Nil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Only chunk items in resolved — no note items to trigger nomination.
	resolved := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("src#turn-1", 0.9),
		cli.ExportNewChunkResolvedItem("src#turn-2", 0.8),
	}

	meta := cli.ExportNewVaultNotesMetaWithTerms(map[string][]cli.ExportNominationEntry{
		"eval-methodology": {{NotePath: "nominee.md", Content: "some content"}},
	})

	noms := cli.ExportBuildTagNominationsUnit(resolved, meta)

	g.Expect(noms).To(BeNil())
}

// TestBuildTagNominations_TagSharedNote_AppearsInNominations is the RED test for
// tag-match nomination: a note absent from the ranked items but sharing a vocab
// term with a top-3 note must appear in the nomination result.
func TestBuildTagNominations_TagSharedNote_AppearsInNominations(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Top-3 delivered note with vocab: [eval-methodology].
	top3Content := "---\ntype: fact\nsituation: ctx\nvocab: [eval-methodology]\n---\n\nbody A\n"
	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithContentAndProvenances(
			"1aa.top-note.md", top3Content, 0.9, []string{"direct"},
		),
	}

	// nominee.md also has vocab: [eval-methodology] but is NOT in resolved.
	nomineeContent := "---\ntype: feedback\nsituation: x\nvocab: [eval-methodology]\n---\n\nbody nominee\n"
	meta := cli.ExportNewVaultNotesMetaWithTerms(map[string][]cli.ExportNominationEntry{
		"eval-methodology": {{NotePath: "nominee.md", Content: nomineeContent}},
	})

	noms := cli.ExportBuildTagNominationsUnit(resolved, meta)

	g.Expect(noms).NotTo(BeNil(), "should return nominations when a vocab term is shared")

	// The nomination should appear in some cluster's candidate list.
	var foundPaths []string

	for _, candidates := range noms {
		for _, c := range candidates {
			foundPaths = append(foundPaths, c.Path)
		}
	}

	g.Expect(foundPaths).To(ContainElement("nominee.md"),
		"a note sharing a vocab term with a top-3 note must appear in nominations")
}

// TestBuildTagNominations_VocabNoteExcluded verifies that vocab/vocab-index notes
// are never nominated even if they share a term with a top-3 note.
// This is the RED test for "vocab term notes must NOT" appear in candidate_l2s.
func TestBuildTagNominations_VocabNoteExcluded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	top3Content := "---\ntype: fact\nsituation: ctx\nvocab: [eval-methodology]\n---\n\nbody A\n"
	resolved := []cli.ExportResolvedItem{
		cli.ExportNewNoteResolvedItemWithContentAndProvenances(
			"1aa.top-note.md", top3Content, 0.9, []string{"direct"},
		),
	}

	// vocab note with the same term — must NOT be nominated.
	vocabNoteContent := "---\ntype: vocab\nterm: eval-methodology\n---\n\nhow we evaluate.\n"
	meta := cli.ExportNewVaultNotesMetaWithTerms(map[string][]cli.ExportNominationEntry{
		"eval-methodology": {{NotePath: "vocab.eval-methodology.md", Content: vocabNoteContent}},
	})

	noms := cli.ExportBuildTagNominationsUnit(resolved, meta)

	var foundPaths []string

	for _, candidates := range noms {
		for _, c := range candidates {
			foundPaths = append(foundPaths, c.Path)
		}
	}

	g.Expect(foundPaths).NotTo(ContainElement("vocab.eval-methodology.md"),
		"vocab notes must never be nominated")
}

// ── Integration: loadAllVaultNotesMeta ───────────────────────────────────────

// TestLoadAllVaultNotesMeta_BuildsTermIndexAndInverse verifies that a single scan
// over all vault notes builds both the term index and the supersedes inverse map.
func TestLoadAllVaultNotesMeta_BuildsTermIndexAndInverse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Note-A: has vocab [eval-methodology] and supersedes 9a.old.
	noteAContent := "---\ntype: fact\nsituation: ctx\nvocab: [eval-methodology]\n" +
		"supersedes:\n  - note: 9a.old\n    type: updates\n    claim: old was wrong\n---\n\nbody A\n"
	// Note-B: plain note, no vocab or supersedes.
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

	g.Expect(meta.TermIndex).To(BeEmpty(), "TermIndex must be empty when no notes have vocab: frontmatter")
	g.Expect(meta.SupersedesInverse).To(BeEmpty(), "SupersedesInverse must be empty when no notes have supersedes:")

	// Content by basename must still be populated.
	g.Expect(meta.ContentByBasename).To(HaveKey(basename))
}

// ── Unit: noteClusterIDForPath ────────────────────────────────────────────────

// TestNoteClusterIDForPath_NoteInCluster verifies that the correct cluster ID
// is returned when the note IS a member of the matched set.
func TestNoteClusterIDForPath_NoteInCluster(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Member paths: index 0 = "other.md", index 1 = "target.md"
	// Cluster 0 contains index 0; cluster 1 contains index 1.
	got := cli.ExportNoteClusterIDForPathFromPlain(
		"target.md",
		[]string{"other.md", "target.md"},
		[][]int{{0}, {1}},
	)

	g.Expect(got).To(Equal(1), "target.md is in cluster 1")
}

// TestNoteClusterIDForPath_NoteInMembersNotInCluster_ReturnsZero verifies that
// the final fallback 0 is returned when a note is in members but its index
// does not appear in any cluster's memberIDs.
func TestNoteClusterIDForPath_NoteInMembersNotInCluster_ReturnsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Member index 1 ("target.md") exists in members but is not in any cluster.
	got := cli.ExportNoteClusterIDForPathFromPlain(
		"target.md",
		[]string{"other.md", "target.md"},
		[][]int{{0}}, // only index 0 is in cluster 0
	)

	g.Expect(got).To(Equal(0), "fallback to cluster 0 when note index not in any cluster")
}

// TestNoteClusterIDForPath_NoteNotInMembers_ReturnsZero verifies that the
// fallback cluster 0 is returned when the note is not in the matched set.
func TestNoteClusterIDForPath_NoteNotInMembers_ReturnsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := cli.ExportNoteClusterIDForPathFromPlain(
		"absent.md",
		[]string{"other.md"},
		[][]int{{0}},
	)

	g.Expect(got).To(Equal(0), "fallback to cluster 0 when note is absent from matched set")
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

	g.Expect(doc.Vocab).To(BeNil())
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

// TestParseNoteQueryFrontmatter_VocabList_Parsed verifies that a note's vocab:
// frontmatter list is correctly extracted.
func TestParseNoteQueryFrontmatter_VocabList_Parsed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	content := "---\ntype: fact\nsituation: ctx\nvocab: [eval-methodology, scope-discipline]\n---\n\nbody\n"
	doc := cli.ExportParseNoteQueryFrontmatter(content)

	g.Expect(doc.Vocab).To(Equal([]string{"eval-methodology", "scope-discipline"}))
}

// ── Unit: renderClusters (tag-nomination merge path) ─────────────────────────

// TestRenderClusters_TagNominationsAdded verifies that tag-nominated notes are
// appended to a cluster's candidate_l2s and deduped against existing candidates.
func TestRenderClusters_TagNominationsAdded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	nominations := map[int][]cli.ExportQueriedCandidateNote{
		0: {
			{Path: "nominee-a.md", Content: "body a"},
			{Path: "nominee-b.md", Content: "body b"},
		},
	}

	// ExportRenderClustersTagNominations builds a single-member cluster 0
	// and merges nominations into its candidate_l2s.
	got := cli.ExportRenderClustersTagNominations(nominations)

	paths := make([]string, 0, len(got))

	for _, c := range got {
		paths = append(paths, c.Path)
	}

	g.Expect(paths).To(ContainElement("nominee-a.md"), "tag nomination must appear in candidate_l2s")
	g.Expect(paths).To(ContainElement("nominee-b.md"), "tag nomination must appear in candidate_l2s")
}

// ── Unit: renderQueryPayload (tag-nomination budget) ─────────────────────────

// TestRenderQueryPayloadBudget_TagNominationCounts is the RED test for the
// budget half of the no-silent-caps rule: the payload budget must emit
// tag_nominations_added and tag_nominations_dropped when nominations occurred.
func TestRenderQueryPayloadBudget_TagNominationCounts(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const added = 40

	const dropped = 5

	out, err := cli.ExportRenderQueryPayloadTagNominationBudget(added, dropped)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(out).To(ContainSubstring("tag_nominations_added: 40"),
		"budget must report the nomination count")
	g.Expect(out).To(ContainSubstring("tag_nominations_dropped: 5"),
		"budget must report the cap-truncated count — truncation is never silent")
}

// TestRenderQueryPayloadBudget_TagNominationCountsOmittedWhenZero verifies the
// omitempty contract: a query with no nominations emits neither budget field.
func TestRenderQueryPayloadBudget_TagNominationCountsOmittedWhenZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	out, err := cli.ExportRenderQueryPayloadTagNominationBudget(0, 0)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(out).NotTo(ContainSubstring("tag_nominations_added"),
		"zero nominations must omit the added field")
	g.Expect(out).NotTo(ContainSubstring("tag_nominations_dropped"),
		"zero drops must omit the dropped field")
}

// ── Unit: topDeliveredNotes ───────────────────────────────────────────────────

// TestTopDeliveredNotes_SkipsChunks verifies that chunk items are not included
// in the top-N delivered notes pool.
func TestTopDeliveredNotes_SkipsChunks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Three chunk items followed by one note — only the note qualifies.
	resolved := []cli.ExportResolvedItem{
		cli.ExportNewChunkResolvedItem("src#turn-1", 0.90),
		cli.ExportNewChunkResolvedItem("src#turn-2", 0.85),
		cli.ExportNewChunkResolvedItem("src#turn-3", 0.80),
		cli.ExportNewNoteResolvedItem("1aa.note.md", "", ""),
	}

	top := cli.ExportTopDeliveredNotes(resolved, 3)

	g.Expect(top).To(HaveLen(1))

	if len(top) == 0 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(top[0])).To(Equal("1aa.note.md"))
}

// TestTopDeliveredNotes_SkipsRecentItems verifies that recent-channel items are
// excluded from the top-N delivered notes pool.
func TestTopDeliveredNotes_SkipsRecentItems(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	recentItem := cli.ExportNewNoteResolvedItemWithProvenances("recent.md", 0.90, []string{"recent"})
	normalNote := cli.ExportNewNoteResolvedItemWithProvenances("normal.md", 0.85, []string{"direct"})

	top := cli.ExportTopDeliveredNotes([]cli.ExportResolvedItem{recentItem, normalNote}, 3)

	g.Expect(top).To(HaveLen(1))

	if len(top) == 0 {
		return
	}

	g.Expect(cli.ExportResolvedItemPath(top[0])).To(Equal("normal.md"))
}
