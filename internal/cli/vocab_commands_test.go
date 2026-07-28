package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// ── Coverage: bump version, noteContainsAnyRemoval ───────────────────────────

// TestBumpVersion_InvalidInput verifies bumpMajorVersion and bumpMinorVersion
// return the input unchanged when the version string has no '.' separator.
func TestBumpVersion_InvalidInput(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportBumpMajorVersion("invalid")).To(Equal("invalid"),
		"bumpMajorVersion must return input unchanged for a non-semver string")
	g.Expect(cli.ExportBumpMinorVersion("invalid")).To(Equal("invalid"),
		"bumpMinorVersion must return input unchanged for a non-semver string")
}

// TestClearRemovedTermsFromMembers_SkipsDefinitionNotes verifies that a
// bare-vocab definition note is never rewritten by term-removal clearing,
// even when its body text contains the removed term's name (which would
// otherwise trigger noteContainsAnyRemoval).
// TestClearRemovedTermsFromMembers_SkipsDefinitionNotes verifies that a
// bare-vocab definition note is never rewritten by term-removal clearing,
// even when its body text contains the removed term's name (which would
// otherwise trigger noteContainsAnyRemoval).
func TestClearRemovedTermsFromMembers_SkipsDefinitionNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const definitionPath = "/vault/2.2026-07-10.vocab-old-term-definition.md"

	const memberPath = "/vault/1.2026-07-10.member.md"

	definitionContent := "---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines old-term.\n"
	memberContent := "---\ntype: fact\ntags:\n    - vocab/old-term\n---\n\nMember mentions old-term.\n"

	files := map[string][]byte{
		definitionPath: []byte(definitionContent),
		memberPath:     []byte(memberContent),
	}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		ListMD: func(string) ([]string, error) {
			return []string{"2.2026-07-10.vocab-old-term-definition.md", "1.2026-07-10.member.md"}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:  func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning: func(string, ...any) {},
	}

	err := cli.ExportClearRemovedTermsFromMembers(deps, "/vault", []string{"old-term"})
	g.Expect(err).NotTo(HaveOccurred())

	_, definitionWasWritten := written[definitionPath]
	g.Expect(definitionWasWritten).To(BeFalse(), "a definition note must never be rewritten by term removal")

	g.Expect(written).To(HaveKey(memberPath), "member note must still have old-term cleared")
}

// TestCollectCurrentTermEntries_DefinitionNotesCarryDescriptions verifies
// that both #678 Task 4 term-definition notes appear in the refit-request
// term list, with descriptions read from the object: field, and the family
// note contributes no entry.
// TestCollectVaultStats_DefinitionNoteTermsWithMemberCounts verifies stats
// term enumeration reads bare-vocab-tagged definition notes (term from
// termFromDefinitionSlug) and tallies members from tags: vocab/<term> — the
// #678 Task 4 fixture: 2 terms, 2 members for retrieval-design, 1 for
// token-budget. The family note (slug vocab-definition) contributes neither
// a term nor a member/untagged tally.
func TestCollectVaultStats_DefinitionNoteTermsWithMemberCounts(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := buildTask4DefinitionFixture(t)

	osFS := cli.ExportNewVaultFS(realFSForTest())

	names, listErr := osFS.ListMD(vault)
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	statsDeps := cli.VocabStatsDeps{ListMD: osFS.ListMD, ReadFile: osFS.ReadFile}
	termNames, memberCounts, totalNotes, untaggedCount := cli.ExportCollectVaultStats(names, statsDeps, vault)

	g.Expect(termNames).To(ConsistOf("retrieval-design", "token-budget"))
	g.Expect(memberCounts["retrieval-design"]).To(Equal(2))
	g.Expect(memberCounts["token-budget"]).To(Equal(1))
	g.Expect(totalNotes).To(Equal(3), "only the 3 member notes are member-scanned")
	g.Expect(untaggedCount).To(Equal(0))
}

// TestCollectVaultStats_DefinitionNotesExcluded verifies that a bare-vocab
// definition note is excluded from collectVaultStats' totals — neither
// counted as a member note nor as untagged (the extractNoteVocabTags site).
// TestCollectVaultStats_DefinitionNotesExcluded verifies that a bare-vocab
// definition note is excluded from collectVaultStats' totals — neither
// counted as a member note nor as untagged (the extractNoteVocabTags site).
func TestCollectVaultStats_DefinitionNotesExcluded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	writeNote(t, vault, "1.2026-07-10.member.md",
		"---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nBody.\n")
	writeNote(t, vault, "2.2026-07-10.vocab-retrieval-design-definition.md",
		"---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines.\n")

	osFS := cli.ExportNewVaultFS(realFSForTest())

	names, listErr := osFS.ListMD(vault)
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	statsDeps := cli.VocabStatsDeps{ListMD: osFS.ListMD, ReadFile: osFS.ReadFile}
	_, _, totalNotes, untaggedCount := cli.ExportCollectVaultStats(names, statsDeps, vault)

	g.Expect(totalNotes).To(Equal(1), "the definition note must not be counted as a member note")
	g.Expect(untaggedCount).To(Equal(0), "the member note IS tagged (tags: vocab/retrieval-design), so it isn't untagged")
}

// TestComputeTermCentroids_ExcludesDefinitionVectors verifies the
// centroid-purity invariant end to end: a scratch vault with 2 members
// assigned to term X (known vectors) plus X's own definition note (a wildly
// different known vector) — after a full retag pass, X's centroid must equal
// the exact mean of the TWO member vectors, with the definition vector absent.
// TestComputeTermCentroids_ExcludesDefinitionVectors verifies the
// centroid-purity invariant end to end: a scratch vault with 2 members
// assigned to term X (known vectors) plus X's own definition note (a wildly
// different known vector) — after a full retag pass, X's centroid must equal
// the exact mean of the TWO member vectors, with the definition vector absent.
func TestComputeTermCentroids_ExcludesDefinitionVectors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	memberAVec := []float32{1, 0, 0}
	memberBVec := []float32{0, 1, 0}
	definitionVec := []float32{0, 0, 100} // wildly different — must never enter the mean

	writeNoteAndSidecar(t, vault, "1.2026-07-10.member-a.md",
		"---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nMember A.\n", memberAVec)
	writeNoteAndSidecar(t, vault, "2.2026-07-10.member-b.md",
		"---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nMember B.\n", memberBVec)
	writeNoteAndSidecar(t, vault, "3.2026-07-10.vocab-retrieval-design-definition.md",
		"---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines retrieval-design.\n", definitionVec)

	deps := cli.ExportNewVocabDeps(cli.ExportNewTestOsDeps())
	terms := []cli.TermWithVector{{Term: "retrieval-design", Vector: []float32{1, 0, 0}}}

	// floor=-1 makes pass 1 assign EVERY member note to the sole term
	// regardless of its cosine similarity to the description vector (cosine
	// never goes below -1) — the point under test is centroid PURITY (which
	// vectors enter the mean), not the assignment-floor threshold itself, so
	// both orthogonal member vectors must land as pass-1 members.
	cli.ExportRetagAllNotesTwoPass(deps, vault, terms, -1, nil)

	centroidsRaw, readErr := os.ReadFile(filepath.Join(vault, "vocab.centroids.json"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var doc cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(centroidsRaw, &doc)).To(Succeed())

	entry, ok := doc.Terms["retrieval-design"]
	g.Expect(ok).To(BeTrue(), "retrieval-design must have a derived centroid entry")

	if !ok {
		return
	}

	wantMean := []float32{0.5, 0.5, 0}
	g.Expect(entry.Vector).To(HaveLen(len(wantMean)))

	if len(entry.Vector) != len(wantMean) {
		return
	}

	for i, want := range wantMean {
		g.Expect(entry.Vector[i]).To(BeNumerically("~", want, 1e-5),
			"centroid component %d must equal the exact mean of the two member vectors", i)
	}

	g.Expect(entry.MemberCount).To(Equal(2), "the definition note's vector must not enter the member count")
}

// TestEnsureVocabFamilyNote_MintFailureReturnsFalse is TEST (b) from #678
// Task 7's fix brief (FIX 2): a forced WriteFile failure during the family
// note's mint must make ensureVocabFamilyNote report false — not the prior
// unconditional true after an attempt — so a caller's counts summary never
// claims "family note: minted" for a note that was never actually written.
// TestEnsureVocabFamilyNote_MintFailureReturnsFalse is TEST (b) from #678
// Task 7's fix brief (FIX 2): a forced WriteFile failure during the family
// note's mint must make ensureVocabFamilyNote report false — not the prior
// unconditional true after an attempt — so a caller's counts summary never
// claims "family note: minted" for a note that was never actually written.
func TestEnsureVocabFamilyNote_MintFailureReturnsFalse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"
	names := []string{}

	deps := cli.VocabDeps{
		ReadFile:   func(path string) ([]byte, error) { return nil, &testNotFoundError{path: path} },
		WriteFile:  func(string, []byte) error { return errors.New("disk full") },
		LogWarning: func(string, ...any) {},
	}

	when := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	minted := cli.ExportEnsureVocabFamilyNote(t.Context(), deps, vault, &names, "1.0", when, "bootstrap")
	g.Expect(minted).To(BeFalse(), "a failed family note mint must report false, never a false 'minted'")
}

// TestEnsureVocabFamilyNote_StaysBareOnly verifies that the family note
// (slug vocab-definition) is minted with tags: [vocab] only, no self-tag.
// TestEnsureVocabFamilyNote_StaysBareOnly verifies that the family note
// (slug vocab-definition) is minted with tags: [vocab] only, no self-tag.
func TestEnsureVocabFamilyNote_StaysBareOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"
	names := []string{}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) {
			return func() {}, nil
		},
		ListMD: func(string) ([]string, error) { return names, nil },
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := written[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, data []byte) error {
			written[path] = data

			return nil
		},
		LogWarning: func(string, ...any) {},
		Now:        func() time.Time { return time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC) },
	}

	minted := cli.ExportEnsureVocabFamilyNote(context.Background(), deps, vault, &names, "1.0", deps.Now(), "test-site")
	g.Expect(minted).To(BeTrue())

	// Verify exactly one file was written
	g.Expect(written).To(HaveLen(1))

	// Extract the written content
	var content []byte
	for _, data := range written {
		content = data
	}

	// Parse the frontmatter and verify tags: [vocab] only
	frontmatter, ok := cli.ExportSplitFrontmatter(content)
	g.Expect(ok).To(BeTrue())

	var doc struct {
		Tags []string `yaml:"tags"`
	}

	err := yaml.Unmarshal(frontmatter, &doc)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(doc.Tags).To(Equal([]string{"vocab"}),
		"family note must carry [vocab] only, no self-tag")
}

// TestExtractNoteVocabTags_TypeFieldIsIrrelevant verifies that a note with
// type: vocab in its frontmatter but ordinary tags: [vocab/foo] IS extracted
// as a member note — extractNoteVocabTags no longer sniffs the legacy type:
// field (#681); membership is decided solely by the tags: vocab/<term> namespace.
// TestExtractNoteVocabTags_TypeFieldIsIrrelevant verifies that a note with
// type: vocab in its frontmatter but ordinary tags: [vocab/foo] IS extracted
// as a member note — extractNoteVocabTags no longer sniffs the legacy type:
// field (#681); membership is decided solely by the tags: vocab/<term> namespace.
func TestExtractNoteVocabTags_TypeFieldIsIrrelevant(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	names := []string{"1aa.2026-01-01.test.md"}
	deps := cli.VocabStatsDeps{
		ListMD: func(string) ([]string, error) {
			return names, nil
		},
		ReadFile: func(_ string) ([]byte, error) {
			return []byte("---\ntype: vocab\ntags:\n    - vocab/foo\n---\nFoo body.\n"), nil
		},
	}

	_, memberCounts, totalNotes, untaggedCount := cli.ExportCollectVaultStats(names, deps, "/vault")

	g.Expect(totalNotes).To(Equal(1),
		"a type: vocab note with ordinary vocab/<term> tags must still count as a member note")
	g.Expect(untaggedCount).To(Equal(0))
	g.Expect(memberCounts["foo"]).To(Equal(1),
		"the note's vocab/foo tag must be extracted despite the legacy type: vocab field")
}

// TestIdAndDateFromNoteFilename table-tests the "<id>.<date>" prefix parser:
// a filename with no valid leading Luhmann id, and one with fewer than three
// dot-separated segments, both return ok=false — direct unit coverage since
// renameDefinitionNote's production call path only reaches this helper after
// termFromDefinitionSlug has already confirmed a well-formed definition-note
// filename, making these guards unreachable through that path.
// TestIdAndDateFromNoteFilename table-tests the "<id>.<date>" prefix parser:
// a filename with no valid leading Luhmann id, and one with fewer than three
// dot-separated segments, both return ok=false — direct unit coverage since
// renameDefinitionNote's production call path only reaches this helper after
// termFromDefinitionSlug has already confirmed a well-formed definition-note
// filename, making these guards unreachable through that path.
func TestIdAndDateFromNoteFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		wantID   string
		wantDate string
		wantOK   bool
	}{
		{
			name:     "valid definition note filename",
			filename: "211.2026-07-10.vocab-retrieval-design-definition.md",
			wantID:   "211", wantDate: "2026-07-10", wantOK: true,
		},
		{name: "no leading luhmann id", filename: "not-an-id.2026-07-10.vocab-x-definition.md", wantOK: false},
		{name: "too few segments", filename: "211.md", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			gotID, gotDate, gotOK := cli.ExportIDAndDateFromNoteFilename(tc.filename)
			g.Expect(gotOK).To(Equal(tc.wantOK))
			g.Expect(gotID).To(Equal(tc.wantID))
			g.Expect(gotDate).To(Equal(tc.wantDate))
		})
	}
}

// TestLoadAssignmentTermVectors_PrefersCentroids verifies write-time assignment
// vectors: terms present in vocab.centroids.json use the stored centroid; absent
// terms fall back to the term sidecar (description) embedding; a model-id
// mismatch discards the whole file (stale embedding space).
// TestLoadAssignmentTermVectors_PrefersCentroids verifies write-time assignment
// vectors: terms present in vocab.centroids.json use the stored centroid; absent
// terms fall back to the term sidecar (description) embedding; a model-id
// mismatch discards the whole file (stale embedding space).
func TestLoadAssignmentTermVectors_PrefersCentroids(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	zeroVec := make([]float32, 2)
	sidecar := func(vec []float32) []byte {
		return embed.MarshalSidecar(embed.Sidecar{
			SchemaVersion:    1,
			EmbeddingModelID: "test",
			Dims:             2,
			BodyVector:       vec,
			SituationVector:  zeroVec,
		})
	}

	definitionNote := func(id, term string) []byte {
		return []byte("---\ntype: fact\ntier: L2\nsituation: s\nsubject: the " + term + " vocab term\n" +
			"predicate: covers\nobject: desc\nluhmann: \"" + id + "\"\ncreated: \"2026-01-01\"\nsource: test\n" +
			"tags:\n    - vocab\n---\n\nInformation learned: desc\n")
	}

	// bad1 has no sidecar (read error) and bad2 a malformed sidecar — both
	// are skipped by the model-id walk before it reaches x's readable sidecar.
	listMD := func(string) ([]string, error) {
		return []string{
			"210.2026-01-01.vocab-bad1-definition.md",
			"211.2026-01-01.vocab-bad2-definition.md",
			"212.2026-01-01.vocab-x-definition.md",
			"213.2026-01-01.vocab-y-definition.md",
		}, nil
	}

	centroids := []byte(`{"schema_version":1,"embedding_model_id":"test","dims":2,` +
		`"terms":{"x":{"vector":[0.5,0.5],"member_count":3}}}`)

	files := map[string][]byte{
		"/vault/210.2026-01-01.vocab-bad1-definition.md":       definitionNote("210", "bad1"),
		"/vault/211.2026-01-01.vocab-bad2-definition.md":       definitionNote("211", "bad2"),
		"/vault/211.2026-01-01.vocab-bad2-definition.vec.json": []byte("{not json"),
		"/vault/212.2026-01-01.vocab-x-definition.md":          definitionNote("212", "x"),
		"/vault/212.2026-01-01.vocab-x-definition.vec.json":    sidecar([]float32{1, 0}),
		"/vault/213.2026-01-01.vocab-y-definition.md":          definitionNote("213", "y"),
		"/vault/213.2026-01-01.vocab-y-definition.vec.json":    sidecar([]float32{0, 1}),
		"/vault/vocab.centroids.json":                          centroids,
	}
	readFile := func(path string) ([]byte, error) {
		if data, ok := files[path]; ok {
			return data, nil
		}

		return nil, &testNotFoundError{path: path}
	}

	terms, err := cli.ExportLoadAssignmentTermVectors("/vault", listMD, readFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	byName := map[string][]float32{}
	for _, term := range terms {
		byName[term.Term] = term.Vector
	}

	g.Expect(byName["x"]).To(Equal([]float32{0.5, 0.5}), "term with a stored centroid must use it")
	g.Expect(byName["y"]).To(Equal([]float32{0, 1}), "term absent from the file falls back to its sidecar")

	// Model-id mismatch: the centroid file is from another embedding space — ignore it.
	files["/vault/vocab.centroids.json"] = []byte(`{"schema_version":1,` +
		`"embedding_model_id":"other-model","dims":2,"terms":{"x":{"vector":[0.5,0.5],"member_count":3}}}`)

	staleTerms, staleErr := cli.ExportLoadAssignmentTermVectors("/vault", listMD, readFile)
	g.Expect(staleErr).NotTo(HaveOccurred())

	if staleErr != nil {
		return
	}

	staleByName := map[string][]float32{}
	for _, term := range staleTerms {
		staleByName[term.Term] = term.Vector
	}

	g.Expect(staleByName["x"]).To(Equal([]float32{1, 0}),
		"a centroids file from a different model must be ignored (stale space)")

	// Malformed centroids file: degrade to description embeddings.
	files["/vault/vocab.centroids.json"] = []byte("{not json")

	malformedTerms, malformedErr := cli.ExportLoadAssignmentTermVectors("/vault", listMD, readFile)
	g.Expect(malformedErr).NotTo(HaveOccurred())

	if malformedErr != nil {
		return
	}

	malformedByName := map[string][]float32{}
	for _, term := range malformedTerms {
		malformedByName[term.Term] = term.Vector
	}

	g.Expect(malformedByName["x"]).To(Equal([]float32{1, 0}),
		"a malformed centroids file must be ignored")
}

// TestLoadCurrentVocabVersion_ListMDError_DefaultsInitial covers the
// listMD-error branch.
// TestLoadCurrentVocabVersion_ListMDError_DefaultsInitial covers the
// listMD-error branch.
func TestLoadCurrentVocabVersion_ListMDError_DefaultsInitial(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := cli.ExportLoadCurrentVocabVersion(
		"/vault",
		func(string) ([]string, error) { return nil, errors.New("list error") },
		func(string) ([]byte, error) { return nil, errors.New("unused") },
	)

	g.Expect(got).To(Equal("1.0"))
}

// TestLoadCurrentVocabVersion_NoFamilyNote_DefaultsInitial verifies the
// migration-safe default (initialVocabVersion, "1.0") when no family note
// exists in the vault.
// TestLoadCurrentVocabVersion_NoFamilyNote_DefaultsInitial verifies the
// migration-safe default (initialVocabVersion, "1.0") when no family note
// exists in the vault.
func TestLoadCurrentVocabVersion_NoFamilyNote_DefaultsInitial(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	writeNote(t, vault, "1.2026-07-10.other.md", "---\ntype: fact\n---\n\nBody.\n")

	osFS := cli.ExportNewVaultFS(realFSForTest())

	got := cli.ExportLoadCurrentVocabVersion(vault, osFS.ListMD, osFS.ReadFile)
	g.Expect(got).To(Equal("1.0"))
}

// TestLoadCurrentVocabVersion_ReadsFamilyNote verifies loadCurrentVocabVersion
// reads vocab_version from the vocab-definition family note rather than
// vocab.index.md.
// TestLoadCurrentVocabVersion_ReadsFamilyNote verifies loadCurrentVocabVersion
// reads vocab_version from the vocab-definition family note rather than
// vocab.index.md.
func TestLoadCurrentVocabVersion_ReadsFamilyNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := buildTask4DefinitionFixture(t)

	osFS := cli.ExportNewVaultFS(realFSForTest())

	got := cli.ExportLoadCurrentVocabVersion(vault, osFS.ListMD, osFS.ReadFile)
	g.Expect(got).To(Equal("6.0"))
}

// ── Task 2: bare-vocab definition exemption ──────────────────────────────────

// TestLoadMemberNoteVectors_ExcludesDefinitionNotes verifies the
// centroid-purity fix: loadMemberNoteVectors must read each note's content
// and skip bare-vocab definition notes before including their vectors — a
// definition's vector must never reach pass-1 assignment or
// computeTermCentroids (AC4).
// TestLoadMemberNoteVectors_ExcludesDefinitionNotes verifies the
// centroid-purity fix: loadMemberNoteVectors must read each note's content
// and skip bare-vocab definition notes before including their vectors — a
// definition's vector must never reach pass-1 assignment or
// computeTermCentroids (AC4).
func TestLoadMemberNoteVectors_ExcludesDefinitionNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	writeNoteAndSidecar(t, vault, "1.2026-07-10.member.md",
		"---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nMember body.\n", []float32{1, 0, 0})
	writeNoteAndSidecar(t, vault, "2.2026-07-10.vocab-retrieval-design-definition.md",
		"---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines the term.\n", []float32{0, 1, 0})

	vectors := cli.ExportLoadMemberNoteVectors(vault)

	g.Expect(vectors).To(HaveLen(1))
	g.Expect(vectors).To(HaveKey("1.2026-07-10.member.md"))
}

// TestLoadTermVectors_DuplicateDefinitionNotes_FirstWins verifies the
// seen-map deduplication in loadTermVectors: when two definition notes
// resolve to the SAME term slug (different files with the same term in
// their filename), only the FIRST one in filename-sorted order contributes
// its vector to the result; the second is skipped by the seen check.
// TestLoadTermVectors_DuplicateDefinitionNotes_FirstWins verifies the
// seen-map deduplication in loadTermVectors: when two definition notes
// resolve to the SAME term slug (different files with the same term in
// their filename), only the FIRST one in filename-sorted order contributes
// its vector to the result; the second is skipped by the seen check.
func TestLoadTermVectors_DuplicateDefinitionNotes_FirstWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	firstVec := []float32{1, 0}
	secondVec := []float32{0, 1}

	// Two definition notes for the same term "dup-term", sorted by filename.
	// 210 sorts before 211, so firstVec should win.
	writeNoteAndSidecar(t, vault, "210.2026-07-10.vocab-dup-term-definition.md",
		"---\ntype: fact\nobject: desc\ntags:\n    - vocab\n---\n\nDefines dup-term (first).\n",
		firstVec)
	writeNoteAndSidecar(t, vault, "211.2026-07-10.vocab-dup-term-definition.md",
		"---\ntype: fact\nobject: desc\ntags:\n    - vocab\n---\n\nDefines dup-term (second).\n",
		secondVec)

	osFS := cli.ExportNewVaultFS(realFSForTest())

	terms, err := cli.ExportLoadTermVectors(vault, osFS.ListMD, osFS.ReadFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(terms).To(HaveLen(1), "exactly one TermWithVector for dup-term")

	if len(terms) != 1 {
		return
	}

	g.Expect(terms[0].Term).To(Equal("dup-term"))
	g.Expect(terms[0].Vector).To(Equal(firstVec),
		"the first definition note's vector (210...) must win over the second (211...)")
}

// TestLoadTermVectors_ReadsDefinitionNoteSidecars verifies the non-centroids
// fallback reads term vectors from bare-vocab-tagged definition note
// sidecars, keyed by term (termFromDefinitionSlug), and skips the family note.
// TestLoadTermVectors_ReadsDefinitionNoteSidecars verifies the non-centroids
// fallback reads term vectors from bare-vocab-tagged definition note
// sidecars, keyed by term (termFromDefinitionSlug), and skips the family note.
func TestLoadTermVectors_ReadsDefinitionNoteSidecars(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	writeNoteAndSidecar(t, vault, "211.2026-07-10.vocab-retrieval-design-definition.md",
		"---\ntype: fact\nobject: desc\ntags:\n    - vocab\n---\n\nDefines retrieval-design.\n",
		[]float32{1, 0})
	writeNoteAndSidecar(t, vault, "212.2026-07-10.vocab-token-budget-definition.md",
		"---\ntype: fact\nobject: desc\ntags:\n    - vocab\n---\n\nDefines token-budget.\n",
		[]float32{0, 1})
	writeNote(t, vault, "210.2026-07-10.vocab-definition.md",
		"---\ntype: fact\nvocab_version: \"6.0\"\ntags:\n    - vocab\n---\n\nFamily root.\n")

	osFS := cli.ExportNewVaultFS(realFSForTest())

	terms, err := cli.ExportLoadTermVectors(vault, osFS.ListMD, osFS.ReadFile)
	g.Expect(err).NotTo(HaveOccurred())

	names := make([]string, 0, len(terms))
	for _, term := range terms {
		names = append(names, term.Term)
	}

	g.Expect(names).To(ConsistOf("retrieval-design", "token-budget"))
}

// ── Task 2 (#681): legacy filename-sniff retirement ──────────────────────────

// TestMemberScan_FilenamePrefixNoLongerSpecial proves the LIVE exclusions
// survive the legacy filename-sniff removal: a bare-vocab definition note
// stays excluded by CONTENT (isVocabDefinitionNote, inside assignVocabToNote)
// even though its sidecar vector matches the term, a qa.*.q.md question note
// stays excluded by isQAQuestionFilename, and an ordinary note whose filename
// happens to start with "vocab." (no such file exists in a migrated vault,
// but the name must no longer be special) IS scanned and assigned. Before the
// #681 edit, the (now-removed) filename-prefix skip excludes the ordinary
// note too, so memberCounts["x"] == 0 and this test FAILS; after the edit it
// is 1.
// TestMemberScan_FilenamePrefixNoLongerSpecial proves the LIVE exclusions
// survive the legacy filename-sniff removal: a bare-vocab definition note
// stays excluded by CONTENT (isVocabDefinitionNote, inside assignVocabToNote)
// even though its sidecar vector matches the term, a qa.*.q.md question note
// stays excluded by isQAQuestionFilename, and an ordinary note whose filename
// happens to start with "vocab." (no such file exists in a migrated vault,
// but the name must no longer be special) IS scanned and assigned. Before the
// #681 edit, the (now-removed) filename-prefix skip excludes the ordinary
// note too, so memberCounts["x"] == 0 and this test FAILS; after the edit it
// is 1.
func TestMemberScan_FilenamePrefixNoLongerSpecial(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	definitionNote := "---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines x.\n"
	qaQuestionNote := "---\ntype: qa-question\ndate: \"2026-07-03\"\nanswered_by: qa.2026-07-03.slug.a\n---\n\n" +
		"Question body.\n"
	ordinaryNote := "---\ntype: fact\n---\n\nAn ordinary note whose filename happens to start with vocab.\n"

	matchingVec := []float32{1, 0}
	sidecar := mustMarshalSidecarWithBodyVector(t, matchingVec)

	files := map[string][]byte{
		"/vault/10.2026-07-03.vocab-x-definition.md":       []byte(definitionNote),
		"/vault/10.2026-07-03.vocab-x-definition.vec.json": sidecar,
		"/vault/qa.2026-07-03.slug.q.md":                   []byte(qaQuestionNote),
		"/vault/vocab.ordinary.md":                         []byte(ordinaryNote),
		"/vault/vocab.ordinary.vec.json":                   sidecar,
	}

	deps := cli.VocabDeps{
		ListMD: func(string) ([]string, error) {
			return []string{
				"10.2026-07-03.vocab-x-definition.md",
				"qa.2026-07-03.slug.q.md",
				"vocab.ordinary.md",
			}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:  func(string, []byte) error { return nil },
		LogWarning: func(string, ...any) {},
	}

	terms := []cli.TermWithVector{{Term: "x", Vector: matchingVec}}

	memberCounts := cli.ExportRetagAllNotesTwoPass(deps, "/vault", terms, 0.5, nil)

	g.Expect(memberCounts["x"]).To(Equal(1),
		"only the ordinary vocab.-prefixed note may count; the definition note stays excluded by "+
			"content (isVocabDefinitionNote) and the qa question note stays excluded by filename "+
			"(isQAQuestionFilename)")
}

// ── Coverage: definition self-tags (Group 2) ────────────────────────────────

// TestMintDefinitionNote_SelfTagOrder verifies that mintDefinitionNote writes
// a definition note with tags: [vocab, vocab/<term>] in that exact order
// (bare marker first, self-tag appended).
// TestMintDefinitionNote_SelfTagOrder verifies that mintDefinitionNote writes
// a definition note with tags: [vocab, vocab/<term>] in that exact order
// (bare marker first, self-tag appended).
func TestMintDefinitionNote_SelfTagOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"
	names := []string{}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) {
			return func() {}, nil
		},
		ListMD: func(string) ([]string, error) { return names, nil },
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := written[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, data []byte) error {
			written[path] = data

			return nil
		},
		LogWarning: func(string, ...any) {},
		Now:        func() time.Time { return time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC) },
	}

	f := cli.ExportDefinitionNoteFactFields("test-term", "A test term definition", "test source")
	err := cli.ExportMintDefinitionNote(context.Background(), deps, vault, &names,
		"vocab-test-term-definition", f, "", nil, deps.Now())
	g.Expect(err).NotTo(HaveOccurred())

	// Verify exactly one file was written
	g.Expect(written).To(HaveLen(1))

	// Extract the written content
	var content []byte
	for _, data := range written {
		content = data
	}

	// Parse the frontmatter and verify tags: [vocab, vocab/test-term]
	frontmatter, ok := cli.ExportSplitFrontmatter(content)
	g.Expect(ok).To(BeTrue())

	var doc struct {
		Tags []string `yaml:"tags"`
	}

	err = yaml.Unmarshal(frontmatter, &doc)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(doc.Tags).To(Equal([]string{"vocab", "vocab/test-term"}),
		"definition note must carry [vocab, vocab/test-term] in that order")
}

// ── Coverage: newVocabDeps closures ──────────────────────────────────────────

// TestNewVocabDeps_ClosuresCalled verifies that the ListMD, WriteFile, and
// DeleteFile closures inside newVocabDeps are wired correctly. Covers the
// function body and its closure blocks.
// TestNewVocabDeps_ClosuresCalled verifies that the ListMD, WriteFile, and
// DeleteFile closures inside newVocabDeps are wired correctly. Covers the
// function body and its closure blocks.
func TestNewVocabDeps_ClosuresCalled(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	deps := cli.ExportNewVocabDeps(cli.ExportNewTestOsDeps())

	// ListMD: empty vault returns no files.
	names, listErr := deps.ListMD(vault)
	g.Expect(listErr).NotTo(HaveOccurred())
	g.Expect(names).To(BeEmpty())

	// WriteFile: creates the file.
	notePath := filepath.Join(vault, "test.md")
	writeErr := deps.WriteFile(notePath, []byte("hello"))
	g.Expect(writeErr).NotTo(HaveOccurred())

	// DeleteFile: removes the file successfully (covers post-if nil return in closure).
	deleteErr := deps.DeleteFile(notePath)
	g.Expect(deleteErr).NotTo(HaveOccurred())
}

// TestNoteContainsAnyRemoval_NoMatch verifies that false is returned when
// no removal term appears in the note content.
// TestNoteContainsAnyRemoval_NoMatch verifies that false is returned when
// no removal term appears in the note content.
func TestNoteContainsAnyRemoval_NoMatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportNoteContainsAnyRemoval("some content about eval", []string{"nonexistent-term"})).
		To(BeFalse(), "must return false when no removal term is in content")
}

// ── QA round-1: qa pairs count + round-2 gate ────────────────────────────────

func TestPrintStatsReport_QAGateReady(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf strings.Builder

	cli.ExportPrintStatsReport(&buf, nil, nil, 50, 0, "1.0", false, "", 20)

	g.Expect(buf.String()).To(ContainSubstring("qa round-2 gate: READY (20>=20)"))
}

func TestPrintStatsReport_QAPairsLine(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf strings.Builder

	cli.ExportPrintStatsReport(&buf, nil, nil, 10, 0, "1.0", false, "", 5)

	g.Expect(buf.String()).To(ContainSubstring("qa pairs: 5"))
	g.Expect(buf.String()).To(ContainSubstring("qa round-2 gate: accumulating (5/20)"))
}

// ── Task 7: verdict line ──────────────────────────────────────────────────────

func TestPrintStatsReport_VerdictOK(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf strings.Builder

	cli.ExportPrintStatsReport(&buf, nil, nil, 10, 0, "1.0", false, "", 0)

	g.Expect(buf.String()).To(ContainSubstring("verdict: OK"))
}

func TestPrintStatsReport_VerdictRefitPending(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf strings.Builder

	cli.ExportPrintStatsReport(&buf, nil, nil, 10, 0, "1.0", true, "growth: 41 notes, 15 days", 0)

	g.Expect(buf.String()).To(ContainSubstring("verdict: REFIT_PENDING (growth: 41 notes, 15 days)"))
}

// TestProcessVocabDefinitionNote_WriteError verifies that processVocabDefinitionNote
// handles write failures gracefully, reporting "write failed" and not panicking
// even when LogWarning is nil.
// TestProcessVocabDefinitionNote_WriteError verifies that processVocabDefinitionNote
// handles write failures gracefully, reporting "write failed" and not panicking
// even when LogWarning is nil.
func TestProcessVocabDefinitionNote_WriteError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"
	testErr := errors.New("disk full")
	files := map[string][]byte{
		"/vault/1.2026-07-10.vocab-test-definition.md": []byte(
			"---\ntype: fact\ntags:\n    - vocab\n---\n\nTest definition.\n",
		),
	}

	output := &bytes.Buffer{}

	deps := cli.VocabDeps{
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(string, []byte) error {
			return testErr
		},
		LogWarning: nil, // Explicitly test the nil case
	}

	cli.ExportProcessVocabDefinitionNote("1.2026-07-10.vocab-test-definition.md", vault, deps, output)

	outStr := output.String()
	g.Expect(outStr).To(ContainSubstring("write failed"))
}

// TestRenderDefinitionNoteContent_NoDoublePeriod is TEST (c) from #678 Task
// 7's fix brief (FIX 3): a description that already ends in a period must
// not double-punctuate the rendered "... covers <description>." body
// sentence (real production example: cost-optimization's stored description
// ends in a period, which used to render "...capability loss..").
// TestRenderDefinitionNoteContent_NoDoublePeriod is TEST (c) from #678 Task
// 7's fix brief (FIX 3): a description that already ends in a period must
// not double-punctuate the rendered "... covers <description>." body
// sentence (real production example: cost-optimization's stored description
// ends in a period, which used to render "...capability loss..").
func TestRenderDefinitionNoteContent_NoDoublePeriod(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	f := cli.ExportFactFields{
		Situation: "recalling what the cost-optimization vocab term covers",
		Subject:   "the cost-optimization vocab term",
		Predicate: "covers",
		Object:    "Reducing LLM and computation costs without capability loss.",
	}

	when := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	got := cli.ExportRenderDefinitionNoteContent(f, "", nil, when)

	g.Expect(got).NotTo(ContainSubstring(".."),
		"a description already ending in a period must not double-punctuate the body sentence")
	g.Expect(got).To(ContainSubstring(
		"covers Reducing LLM and computation costs without capability loss.\n"),
		"exactly one trailing period on the rendered body sentence")
	g.Expect(got).To(ContainSubstring(
		"object: Reducing LLM and computation costs without capability loss."),
		"the frontmatter object: field must still carry the description's own trailing period verbatim")
}

// TestRetagAllNotesTwoPass_ListMDError_LogsWarning covers loadMemberNoteVectors'
// and assignTermsToAllNotes' listMD-error branches: a ListMD that always
// errors makes pass 1 silently see zero members (loadMemberNoteVectors
// returns nil) and pass 2 log a warning (assignTermsToAllNotes returns the
// wrapped error, which retagAllNotesTwoPass logs as "pass-2 assignment").
// TestRetagAllNotesTwoPass_ListMDError_LogsWarning covers loadMemberNoteVectors'
// and assignTermsToAllNotes' listMD-error branches: a ListMD that always
// errors makes pass 1 silently see zero members (loadMemberNoteVectors
// returns nil) and pass 2 log a warning (assignTermsToAllNotes returns the
// wrapped error, which retagAllNotesTwoPass logs as "pass-2 assignment").
func TestRetagAllNotesTwoPass_ListMDError_LogsWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var messages []string

	deps := cli.VocabDeps{
		ListMD:     func(string) ([]string, error) { return nil, errors.New("list error") },
		ReadFile:   func(string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFile:  func(string, []byte) error { return nil },
		LogWarning: func(format string, args ...any) { messages = append(messages, fmt.Sprintf(format, args...)) },
	}

	terms := []cli.TermWithVector{{Term: "x", Vector: []float32{1, 0}}}

	cli.ExportRetagAllNotesTwoPass(deps, "/vault", terms, 0.35, nil)

	found := false

	for _, msg := range messages {
		if strings.Contains(msg, "pass-2 assignment") {
			found = true
		}
	}

	g.Expect(found).To(BeTrue(), "assignTermsToAllNotes' listMD error must be logged")
}

// TestRetagAllNotesTwoPass_LoadMemberNoteVectors_SkipsDefinitionAndUnreadable
// covers loadMemberNoteVectors' remaining branches: a bare-vocab DEFINITION
// note is skipped by CONTENT (isVocabDefinitionNote) even though it carries a
// fully valid, matching sidecar; a note with no sidecar is skipped after a
// readable-content check; a note with a malformed sidecar is skipped after a
// successful sidecar read. Only the one fully-readable member note becomes a
// pass-1 member, so the derived centroid's member_count must be exactly 1.
// TestRetagAllNotesTwoPass_LoadMemberNoteVectors_SkipsDefinitionAndUnreadable
// covers loadMemberNoteVectors' remaining branches: a bare-vocab DEFINITION
// note is skipped by CONTENT (isVocabDefinitionNote) even though it carries a
// fully valid, matching sidecar; a note with no sidecar is skipped after a
// readable-content check; a note with a malformed sidecar is skipped after a
// successful sidecar read. Only the one fully-readable member note becomes a
// pass-1 member, so the derived centroid's member_count must be exactly 1.
func TestRetagAllNotesTwoPass_LoadMemberNoteVectors_SkipsDefinitionAndUnreadable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	memberValid := "---\ntype: fact\ntags:\n    - vocab/x\n---\n\nvalid member.\n"
	memberNoSidecar := "---\ntype: fact\ntags:\n    - vocab/x\n---\n\nno sidecar.\n"
	memberBadSidecar := "---\ntype: fact\ntags:\n    - vocab/x\n---\n\nbad sidecar.\n"
	definitionNote := "---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines x.\n"

	zeroVec := make([]float32, 2)
	validSidecar := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: 1, EmbeddingModelID: "test", Dims: 2,
		BodyVector: []float32{1, 0}, SituationVector: zeroVec,
	})

	files := map[string][]byte{
		"/vault/1.valid.md":             []byte(memberValid),
		"/vault/1.valid.vec.json":       validSidecar,
		"/vault/2.no-sidecar.md":        []byte(memberNoSidecar),
		"/vault/3.bad-sidecar.md":       []byte(memberBadSidecar),
		"/vault/3.bad-sidecar.vec.json": []byte("{not json"),
		"/vault/0.def.md":               []byte(definitionNote),
		"/vault/0.def.vec.json":         validSidecar,
	}

	written := map[string][]byte{}

	deps := cli.VocabDeps{
		ListMD: func(string) ([]string, error) {
			return []string{"0.def.md", "1.valid.md", "2.no-sidecar.md", "3.bad-sidecar.md"}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:  func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning: func(string, ...any) {},
	}

	terms := []cli.TermWithVector{{Term: "x", Vector: []float32{1, 0}}}

	cli.ExportRetagAllNotesTwoPass(deps, "/vault", terms, 0.5, nil)

	centroidsRaw, ok := written["/vault/vocab.centroids.json"]
	g.Expect(ok).To(BeTrue(), "retag must write the derived centroids file")

	if !ok {
		return
	}

	var doc cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(centroidsRaw, &doc)).To(Succeed())

	entry, hasEntry := doc.Terms["x"]
	g.Expect(hasEntry).To(BeTrue())

	if !hasEntry {
		return
	}

	g.Expect(entry.MemberCount).To(Equal(1),
		"only the one fully-readable member note may contribute to the centroid")
}

// TestRetagAllNotesTwoPass_PreservesDefinitionSelfTags verifies that when
// retagAllNotesTwoPass is run over a vault containing self-tagged definition
// notes (tags: [vocab, vocab/<term>]) plus member notes, the definition notes'
// bytes remain byte-for-byte identical after the refit. Definition notes must
// never be modified by the retagging operation.
// TestRetagAllNotesTwoPass_PreservesDefinitionSelfTags verifies that when
// retagAllNotesTwoPass is run over a vault containing self-tagged definition
// notes (tags: [vocab, vocab/<term>]) plus member notes, the definition notes'
// bytes remain byte-for-byte identical after the refit. Definition notes must
// never be modified by the retagging operation.
func TestRetagAllNotesTwoPass_PreservesDefinitionSelfTags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Build closure-deps fixture with self-tagged definition notes and members.
	definitionContent1 := "" +
		"---\ntype: fact\ntags:\n    - vocab\n    - vocab/retrieval-design\n---\n\n" +
		"retrieval-design is when you narrow vault searches to task-relevant regions.\n"
	definitionContent2 := "" +
		"---\ntype: fact\ntags:\n    - vocab\n    - vocab/token-budget\n---\n\n" +
		"token-budget tracks payload size against context limits.\n"
	memberContent1 := "" +
		"---\ntype: feedback\nsituation: ctx\ntags:\n    - vocab/retrieval-design\n---\n\n" +
		"Member of retrieval-design.\n"
	memberContent2 := "" +
		"---\ntype: feedback\nsituation: ctx\ntags:\n    - vocab/token-budget\n---\n\n" +
		"Member of token-budget.\n"

	files := map[string][]byte{
		"/vault/210.2026-07-10.vocab-retrieval-design-definition.md": []byte(definitionContent1),
		"/vault/211.2026-07-10.vocab-token-budget-definition.md":     []byte(definitionContent2),
		"/vault/1aa.2026-07-10.member-retrieval.md":                  []byte(memberContent1),
		"/vault/1ab.2026-07-10.member-token.md":                      []byte(memberContent2),
	}

	written := make(map[string][]byte)

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{
				"210.2026-07-10.vocab-retrieval-design-definition.md",
				"211.2026-07-10.vocab-token-budget-definition.md",
				"1aa.2026-07-10.member-retrieval.md",
				"1ab.2026-07-10.member-token.md",
			}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			// Prefer written map so pass 2 sees pass-1 output.
			if data, ok := written[path]; ok {
				return data, nil
			}

			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, data []byte) error {
			written[path] = data

			return nil
		},
		WriteSidecar: func(string, []byte) error { return nil },
		DeleteFile:   func(string) error { return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC) },
	}

	terms := []cli.TermWithVector{
		{Term: "retrieval-design", Vector: []float32{1, 0, 0}},
		{Term: "token-budget", Vector: []float32{0, 1, 0}},
	}

	cli.ExportRetagAllNotesTwoPass(deps, "/vault", terms, 0.0, nil)

	// Assert definition notes were never written, or if written, bytes are identical.
	for defPath, originalBytes := range map[string][]byte{
		"/vault/210.2026-07-10.vocab-retrieval-design-definition.md": []byte(definitionContent1),
		"/vault/211.2026-07-10.vocab-token-budget-definition.md":     []byte(definitionContent2),
	} {
		if writtenBytes, wasWritten := written[defPath]; wasWritten {
			g.Expect(writtenBytes).To(Equal(originalBytes),
				fmt.Sprintf("definition note %s must not be rewritten; bytes must be identical", filepath.Base(defPath)))
		}
	}
}

// ── Task 4: retagAllNotesTwoPass seeds last_refit ────────────────────────────

// TestRetagAllNotesTwoPass_SeedsLastRefit verifies that a non-nil lastRefit
// supplied to retagAllNotesTwoPass is written into vocab.centroids.json.
// TestRetagAllNotesTwoPass_SeedsLastRefit verifies that a non-nil lastRefit
// supplied to retagAllNotesTwoPass is written into vocab.centroids.json.
func TestRetagAllNotesTwoPass_SeedsLastRefit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	lastRefit := &cli.ExportVocabLastRefitDoc{NoteCount: 50, Date: "2026-07-03"}

	var written []byte

	deps := cli.VocabDeps{
		ListMD:     func(string) ([]string, error) { return nil, nil },
		ReadFile:   func(string) ([]byte, error) { return nil, os.ErrNotExist },
		WriteFile:  func(_ string, data []byte) error { written = data; return nil },
		LogWarning: nil,
	}

	cli.ExportRetagAllNotesTwoPass(deps, "/vault", nil, 0.35, lastRefit)

	g.Expect(written).NotTo(BeNil())

	var doc cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(written, &doc)).NotTo(HaveOccurred())

	if err := json.Unmarshal(written, &doc); err != nil {
		return
	}

	g.Expect(doc.LastRefit).NotTo(BeNil())

	if doc.LastRefit == nil {
		return
	}

	g.Expect(doc.LastRefit.NoteCount).To(Equal(50))
}

// TestRetagAllNotesTwoPass_SkipsDefinitionNotes is the fixture assertion for
// both retagAllNotesTwoPass (pass 1, via loadMemberNoteVectors) and
// assignTermsToAllNotes (pass 2, via assignVocabToNote): over a scratch vault
// containing one bare-vocab definition note plus one member note, a full
// retag must leave the definition file's bytes byte-for-byte unchanged.
// TestRetagAllNotesTwoPass_SkipsDefinitionNotes is the fixture assertion for
// both retagAllNotesTwoPass (pass 1, via loadMemberNoteVectors) and
// assignTermsToAllNotes (pass 2, via assignVocabToNote): over a scratch vault
// containing one bare-vocab definition note plus one member note, a full
// retag must leave the definition file's bytes byte-for-byte unchanged.
func TestRetagAllNotesTwoPass_SkipsDefinitionNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	const definitionBasename = "2.2026-07-10.vocab-retrieval-design-definition.md"

	definitionContent := "---\ntype: fact\ntags:\n    - vocab\n---\n\nDefines the retrieval-design term.\n"
	writeNoteAndSidecar(t, vault, definitionBasename, definitionContent, []float32{0, 1, 0})

	memberContent := "---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nMember body.\n"
	writeNoteAndSidecar(t, vault, "1.2026-07-10.member.md", memberContent, []float32{1, 0, 0})

	deps := cli.ExportNewVocabDeps(cli.ExportNewTestOsDeps())
	terms := []cli.TermWithVector{{Term: "retrieval-design", Vector: []float32{1, 0, 0}}}

	cli.ExportRetagAllNotesTwoPass(deps, vault, terms, 0.35, nil)

	gotBytes, readErr := os.ReadFile(filepath.Join(vault, definitionBasename))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(gotBytes)).To(Equal(definitionContent),
		"the definition file's bytes must be unchanged after a full retag over a scratch vault")
}

// TestRewriteVocabVersionKey_NoFrontmatter_ReturnsUnchanged covers the
// no-parseable-frontmatter guard — unreachable via
// writeVocabVersionToFamilyNote's production call path (findVocabFamilyNote
// already validates frontmatter via isVocabDefinitionNote before a note is
// selected as the family note), so exercised directly here.
// TestRewriteVocabVersionKey_NoFrontmatter_ReturnsUnchanged covers the
// no-parseable-frontmatter guard — unreachable via
// writeVocabVersionToFamilyNote's production call path (findVocabFamilyNote
// already validates frontmatter via isVocabDefinitionNote before a note is
// selected as the family note), so exercised directly here.
func TestRewriteVocabVersionKey_NoFrontmatter_ReturnsUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportRewriteVocabVersionKey("no frontmatter here", "1.1")).To(Equal("no frontmatter here"))
}

// ── Coverage: assigner error paths ───────────────────────────────────────────

// TestRunAmend_VocabAssignment_SkipsWhenSidecarMissing verifies that when
// the note sidecar is unreadable, vocab assignment silently no-ops and the
// bare-amend write still succeeds (backward compat).
// TestRunAmend_VocabAssignment_SkipsWhenSidecarMissing verifies that when
// the note sidecar is unreadable, vocab assignment silently no-ops and the
// bare-amend write still succeeds (backward compat).
func TestRunAmend_VocabAssignment_SkipsWhenSidecarMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	noteContent := []byte(
		"---\ntype: feedback\ntier: L2\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
			"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\n" +
			"Lesson learned: when ctx, a.\n\n",
	)

	termVec := makeUnitVec(0.95)

	var written []byte

	deps := cli.AmendDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".vec.json") {
				return nil, &testNotFoundError{path: path}
			}

			return noteContent, nil
		},
		Write: func(_ string, data []byte) error { written = data; return nil },
		Now:   func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		ListIndexes: func(string) ([]string, error) { return nil, nil },
		LoadTermVectors: func(string) ([]cli.TermWithVector, error) {
			return []cli.TermWithVector{{Term: "eval-methodology", Vector: termVec}}, nil
		},
	}

	args := cli.AmendArgs{
		Vault:  "/vault",
		Target: "1aa",
	}

	var buf strings.Builder

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(written)).NotTo(ContainSubstring("vocab:"),
		"vocab: must not be added when sidecar is missing")
	g.Expect(string(written)).NotTo(ContainSubstring("Vocab:"),
		"Vocab: body line must not be added when sidecar is missing")
}

// ── Assigner wiring: amend ────────────────────────────────────────────────────

// TestRunAmend_VocabAssignment_WritesVocabWhenTermsPresent verifies that after
// an amend, if terms are present, the note gets vocab/<term> tags written.
// TestRunAmend_VocabAssignment_WritesVocabWhenTermsPresent verifies that after
// an amend, if terms are present, the note gets vocab/<term> tags written.
func TestRunAmend_VocabAssignment_WritesVocabWhenTermsPresent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const basename = "1aa.2026-01-01.test.md"

	noteContent := []byte(
		"---\ntype: feedback\ntier: L2\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
			"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\n" +
			"Lesson learned: when ctx, a.\n\n",
	)

	bodyVec := makeUnitVec(1.0)
	termVec := makeUnitVec(0.95)

	sidecar := embed.Sidecar{
		SchemaVersion:    1,
		EmbeddingModelID: "test",
		Dims:             2,
		BodyVector:       bodyVec,
		SituationVector:  bodyVec,
		ContentHash:      "abc",
	}

	var written []byte

	deps := cli.AmendDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".vec.json") {
				return embed.MarshalSidecar(sidecar), nil
			}

			return noteContent, nil
		},
		Write: func(_ string, data []byte) error { written = data; return nil },
		Now:   func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		ListIndexes: func(string) ([]string, error) { return nil, nil },
		LoadTermVectors: func(string) ([]cli.TermWithVector, error) {
			return []cli.TermWithVector{{Term: "eval-methodology", Vector: termVec}}, nil
		},
	}

	args := cli.AmendArgs{
		Vault:  "/vault",
		Target: "1aa",
	}

	var buf strings.Builder

	err := cli.ExportRunAmend(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(written)).To(ContainSubstring("tags:\n    - vocab/eval-methodology"),
		"the vocab/ namespace of tags: must be present after amend when terms are assigned")
	g.Expect(string(written)).NotTo(ContainSubstring("\nvocab:"),
		"legacy vocab: frontmatter key must not be written")
	g.Expect(string(written)).NotTo(ContainSubstring("Vocab:"),
		"legacy Vocab: body line must not be written")
}

// TestRunLearn_VocabAssignment_SkipsWhenDepsNotWired verifies backward compat:
// when LoadTermVectors is nil, no vocab assignment occurs and no panic.
// TestRunLearn_VocabAssignment_SkipsWhenDepsNotWired verifies backward compat:
// when LoadTermVectors is nil, no vocab assignment occurs and no panic.
func TestRunLearn_VocabAssignment_SkipsWhenDepsNotWired(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.LearnDeps{
		Now:           func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		StatDir:       func(string) error { return nil },
		InitVault:     func(string) error { return nil },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(string, []byte) error { return nil },
		// LoadTermVectors, ReadSidecar, WriteNote are all nil
	}

	args := cli.LearnArgs{
		Type:      "fact",
		Slug:      "backwards-compat",
		Vault:     "/vault",
		Position:  "top",
		Source:    "test",
		Situation: "backward compat with no vocab",
		Subject:   "X", Predicate: "has", Object: "Y",
	}

	var buf strings.Builder

	g.Expect(cli.ExportRunLearn(t.Context(), args, deps, &buf)).To(Succeed(),
		"learn must succeed even when vocab deps are not wired")
}

// TestRunLearn_VocabAssignment_SkipsWhenNoTerms verifies that when
// LoadTermVectors returns an empty slice, WriteNote is never called.
// TestRunLearn_VocabAssignment_SkipsWhenNoTerms verifies that when
// LoadTermVectors returns an empty slice, WriteNote is never called.
func TestRunLearn_VocabAssignment_SkipsWhenNoTerms(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	writeNoteCalled := false

	deps := cli.LearnDeps{
		Now:           func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		StatDir:       func(string) error { return nil },
		InitVault:     func(string) error { return nil },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(string, []byte) error { return nil },
		LoadTermVectors: func(string) ([]cli.TermWithVector, error) {
			return nil, nil // no terms
		},
		WriteNote: func(string, []byte) error { writeNoteCalled = true; return nil },
	}

	args := cli.LearnArgs{
		Type:      "feedback",
		Slug:      "test-no-vocab",
		Vault:     "/vault",
		Position:  "top",
		Source:    "test",
		Situation: "no terms in vault",
		Action:    "do nothing",
	}

	var buf strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writeNoteCalled).To(BeFalse(), "WriteNote must not be called when no terms are present")
}

// TestRunLearn_VocabAssignment_SkipsWhenSidecarMissing verifies that when
// ReadSidecar returns an error after learn, vocab assignment silently no-ops.
// TestRunLearn_VocabAssignment_SkipsWhenSidecarMissing verifies that when
// ReadSidecar returns an error after learn, vocab assignment silently no-ops.
func TestRunLearn_VocabAssignment_SkipsWhenSidecarMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	writeNoteCalled := false

	deps := cli.LearnDeps{
		Now:           func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		StatDir:       func(string) error { return nil },
		InitVault:     func(string) error { return nil },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(string, []byte) error { return nil },
		ReadSidecar: func(path string) ([]byte, error) {
			return nil, &testNotFoundError{path: path}
		},
		LoadTermVectors: func(string) ([]cli.TermWithVector, error) {
			return []cli.TermWithVector{{Term: "eval-methodology", Vector: makeUnitVec(0.95)}}, nil
		},
		WriteNote: func(_ string, _ []byte) error { writeNoteCalled = true; return nil },
	}

	args := cli.LearnArgs{
		Type:      "feedback",
		Slug:      "test-no-sidecar",
		Vault:     "/vault",
		Position:  "top",
		Source:    "test",
		Situation: "testing sidecar-missing skip",
		Action:    "do nothing",
	}

	var buf strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(writeNoteCalled).To(BeFalse(), "WriteNote must not be called when sidecar is missing")
}

// ── Assigner wiring: learn ────────────────────────────────────────────────────

// TestRunLearn_VocabAssignment_WritesVocabWhenTermsPresent verifies that when
// LoadTermVectors returns a non-empty set and ReadSidecar returns a valid sidecar,
// WriteNote is called with vocab/<term> tags only — neither the legacy vocab:
// frontmatter key nor a Vocab: body line is written.
// TestRunLearn_VocabAssignment_WritesVocabWhenTermsPresent verifies that when
// LoadTermVectors returns a non-empty set and ReadSidecar returns a valid sidecar,
// WriteNote is called with vocab/<term> tags only — neither the legacy vocab:
// frontmatter key nor a Vocab: body line is written.
func TestRunLearn_VocabAssignment_WritesVocabWhenTermsPresent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	bodyVec := makeUnitVec(1.0)
	termVec := makeUnitVec(0.95) // cosine >> floor 0.30 → assigned

	sidecar := embed.Sidecar{
		SchemaVersion:    1,
		EmbeddingModelID: "test",
		Dims:             2,
		BodyVector:       bodyVec,
		SituationVector:  bodyVec,
		ContentHash:      "abc",
	}

	var updatedContent []byte

	deps := cli.LearnDeps{
		Now:           func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
		Getenv:        func(string) string { return "" },
		StatDir:       func(string) error { return nil },
		InitVault:     func(string) error { return nil },
		ListIDs:       func(string) ([]string, error) { return nil, nil },
		ListBasenames: func(string) ([]string, error) { return nil, nil },
		Lock:          func(string) (func(), error) { return func() {}, nil },
		WriteNew:      func(string, []byte) error { return nil },
		ReadSidecar: func(_ string) ([]byte, error) {
			return embed.MarshalSidecar(sidecar), nil
		},
		LoadTermVectors: func(string) ([]cli.TermWithVector, error) {
			return []cli.TermWithVector{{Term: "eval-methodology", Vector: termVec}}, nil
		},
		WriteNote: func(_ string, data []byte) error { updatedContent = data; return nil },
	}

	args := cli.LearnArgs{
		Type:      "feedback",
		Slug:      "test-feedback",
		Vault:     "/vault",
		Position:  "top",
		Source:    "test",
		Situation: "testing vocab wiring",
		Action:    "do something",
	}

	var buf strings.Builder

	err := cli.ExportRunLearn(t.Context(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(updatedContent).NotTo(BeNil(), "WriteNote must be called when terms are assigned")
	g.Expect(string(updatedContent)).To(ContainSubstring("tags:\n    - vocab/eval-methodology"),
		"the vocab/ namespace of tags: must be present in updated note")
	g.Expect(string(updatedContent)).NotTo(ContainSubstring("\nvocab:"),
		"legacy vocab: frontmatter key must not be written")
	g.Expect(string(updatedContent)).NotTo(ContainSubstring("Vocab:"),
		"legacy Vocab: body line must not be written")
}

// ── Coverage: assignVocabToNote sidecar-error and note-read-error paths ───────

// TestRunVocabBootstrap_AssignErrors_SkipBothNotes verifies that when note A
// has no sidecar (sidecarErr → return nil) and note B has a sidecar but no
// note content (readErr → return assigned), assignVocabToNote skips writing
// for both (covers sidecarErr and readErr branches in assignVocabToNote).
// TestRunVocabBootstrap_AssignErrors_SkipBothNotes verifies that when note A
// has no sidecar (sidecarErr → return nil) and note B has a sidecar but no
// note content (readErr → return assigned), assignVocabToNote skips writing
// for both (covers sidecarErr and readErr branches in assignVocabToNote).
func TestRunVocabBootstrap_AssignErrors_SkipBothNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{Term: "eval-topic", Description: "desc"}}
	seedYAML, yamlErr := yaml.Marshal(seed)
	g.Expect(yamlErr).NotTo(HaveOccurred())

	if yamlErr != nil {
		return
	}

	termVec := makeUnitVec(0.95)
	noteBVec := makeUnitVec(1.0)

	// SituationVector must be set to len==Dims or UnmarshalSidecar returns ErrDimsMismatch.
	zeroVec := make([]float32, 2)

	termSidecar := embed.Sidecar{
		SchemaVersion:    1,
		EmbeddingModelID: "test",
		Dims:             2,
		BodyVector:       termVec,
		SituationVector:  zeroVec,
	}
	noteBSidecar := embed.Sidecar{
		SchemaVersion:    1,
		EmbeddingModelID: "test",
		Dims:             2,
		BodyVector:       noteBVec,
		SituationVector:  zeroVec,
	}

	// eval-topic already has a definition note (idempotent skip — bootstrap
	// mints nothing for it, only the absent family note).
	definitionNote := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: subj\npredicate: covers\n" +
		"object: desc\nluhmann: \"5\"\ncreated: \"2026-01-01\"\nsource: prior\ntags:\n    - vocab\n" +
		"---\n\nInformation learned: ...\n"

	// 1aa.note-a has no sidecar → sidecarErr in assignVocabToNote.
	// 1ab.note-b has sidecar but no note → readErr in assignVocabToNote.
	files := map[string][]byte{
		"/seed.yaml": seedYAML,
		"/vault/5.2026-01-01.vocab-eval-topic-definition.md":       []byte(definitionNote),
		"/vault/5.2026-01-01.vocab-eval-topic-definition.vec.json": embed.MarshalSidecar(termSidecar),
		"/vault/1ab.note-b.vec.json":                               embed.MarshalSidecar(noteBSidecar),
	}

	var memberWriteCount int

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{"1aa.note-a.md", "1ab.note-b.md", "5.2026-01-01.vocab-eval-topic-definition.md"}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, _ []byte) error {
			base := filepath.Base(path)
			if !strings.HasPrefix(base, "vocab.") && !strings.Contains(base, "-definition") {
				memberWriteCount++
			}

			return nil
		},
		WriteSidecar: func(_ string, _ []byte) error { return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabBootstrap(t.Context(), args, deps, &buf)).To(Succeed())
	g.Expect(memberWriteCount).To(Equal(0),
		"neither member note must be written when sidecar or note is unreadable")
}

// TestRunVocabBootstrap_AssignsTermsToExistingNote verifies that an existing
// member note (with a valid sidecar) gets vocab/<term> tags written.
// TestRunVocabBootstrap_AssignsTermsToExistingNote verifies that an existing
// member note (with a valid sidecar) gets vocab/<term> tags written.
func TestRunVocabBootstrap_AssignsTermsToExistingNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{Term: "eval-methodology", Description: "how we evaluate"}}
	seedYAML, err := yaml.Marshal(seed)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	existingNote := "---\ntype: feedback\nsituation: testing\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\nLesson learned: when testing, a.\n\n"

	// Unit body vector closely aligned with term vector → high cosine → assigned.
	noteVec := makeUnitVec(1.0)
	termVec := makeUnitVec(0.95) // cosine ≈ 0.95 >> floor 0.30

	noteSidecar := embed.Sidecar{
		SchemaVersion:    1,
		EmbeddingModelID: "test",
		Dims:             2,
		BodyVector:       noteVec,
		SituationVector:  noteVec,
		ContentHash:      "abc",
	}
	termSidecar := embed.Sidecar{
		SchemaVersion:    1,
		EmbeddingModelID: "test",
		Dims:             2,
		BodyVector:       termVec,
		SituationVector:  termVec,
		ContentHash:      "def",
	}

	// eval-methodology already has a definition note — bootstrap's idempotent
	// skip means it mints nothing for it, but assignment still runs against
	// its pre-existing sidecar.
	definitionNote := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: subj\npredicate: covers\n" +
		"object: how we evaluate\nluhmann: \"5\"\ncreated: \"2026-01-01\"\nsource: prior\ntags:\n    - vocab\n" +
		"---\n\nInformation learned: ...\n"

	files := map[string][]byte{
		"/seed.yaml":                                                     seedYAML,
		"/vault/1aa.2026-01-01.test.md":                                  []byte(existingNote),
		"/vault/1aa.2026-01-01.test.vec.json":                            embed.MarshalSidecar(noteSidecar),
		"/vault/5.2026-01-01.vocab-eval-methodology-definition.md":       []byte(definitionNote),
		"/vault/5.2026-01-01.vocab-eval-methodology-definition.vec.json": embed.MarshalSidecar(termSidecar),
	}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		// ListMD returns both the member note AND the definition note so that
		// loadTermVectors can discover the term's pre-populated sidecar.
		ListMD: func(_ string) ([]string, error) {
			return []string{"1aa.2026-01-01.test.md", "5.2026-01-01.vocab-eval-methodology-definition.md"}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	bootErr := cli.RunVocabBootstrap(t.Context(), args, deps, &buf)
	g.Expect(bootErr).NotTo(HaveOccurred())

	if bootErr != nil {
		return
	}

	notePath := "/vault/1aa.2026-01-01.test.md"
	g.Expect(written).To(HaveKey(notePath), "existing note must be updated with vocab assignment")
	updatedContent := string(written[notePath])
	g.Expect(updatedContent).To(ContainSubstring("tags:\n    - vocab/eval-methodology"),
		"the vocab/ namespace of tags: must be written")
	g.Expect(updatedContent).NotTo(ContainSubstring("\nvocab:"), "legacy vocab: frontmatter key must not be written")
	g.Expect(updatedContent).NotTo(ContainSubstring("Vocab:"), "legacy Vocab: body line must not be written")
}

// TestRunVocabBootstrap_CentroidTwoPass_SecondPassAssignsNote verifies the
// centroid two-pass: note B ([0,1]) misses every term's DESCRIPTION embedding
// at floor 0.30, but after pass 1 makes note A the sole eval-topic member the
// term centroid becomes [0.8,0.6] and pass 2 assigns B (cos 0.6). The derived
// centroids land in vocab.centroids.json; member-less terms are omitted
// (fallback = description embedding).
// TestRunVocabBootstrap_CentroidTwoPass_SecondPassAssignsNote verifies the
// centroid two-pass: note B ([0,1]) misses every term's DESCRIPTION embedding
// at floor 0.30, but after pass 1 makes note A the sole eval-topic member the
// term centroid becomes [0.8,0.6] and pass 2 assigns B (cos 0.6). The derived
// centroids land in vocab.centroids.json; member-less terms are omitted
// (fallback = description embedding).
func TestRunVocabBootstrap_CentroidTwoPass_SecondPassAssignsNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files := centroidTwoPassFiles(g)
	written := map[string][]byte{}
	deps := centroidTwoPassDeps(files, written)

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabBootstrap(t.Context(), args, deps, &buf)).To(Succeed())

	noteB := string(written["/vault/1ab.note-b.md"])
	g.Expect(noteB).To(ContainSubstring("eval-topic"),
		"pass 2 must assign note B against the member centroid (desc embedding alone misses it)")

	centroidsRaw, ok := written["/vault/vocab.centroids.json"]
	g.Expect(ok).To(BeTrue(), "bootstrap must write the derived centroids file")

	if !ok {
		return
	}

	//nolint:tagliatelle // centroids JSON keys follow the sidecar spec contract (snake_case)
	var doc struct {
		SchemaVersion    int    `json:"schema_version"`
		EmbeddingModelID string `json:"embedding_model_id"`
		Dims             int    `json:"dims"`
		Terms            map[string]struct {
			Vector      []float32 `json:"vector"`
			MemberCount int       `json:"member_count"`
		} `json:"terms"`
	}
	g.Expect(json.Unmarshal(centroidsRaw, &doc)).To(Succeed())
	g.Expect(doc.EmbeddingModelID).To(Equal("test"), "centroids file must carry the sidecar model id")
	g.Expect(doc.Dims).To(Equal(2))
	g.Expect(doc.Terms).To(HaveKey("eval-topic"))
	g.Expect(doc.Terms).NotTo(HaveKey("orphan-topic"),
		"member-less terms keep their description embedding and are omitted from the file")
	g.Expect(doc.Terms["eval-topic"].MemberCount).To(Equal(1), "centroid computed from the 1 pass-1 member")
	g.Expect(doc.Terms["eval-topic"].Vector).To(HaveLen(2))
	g.Expect(doc.Terms["eval-topic"].Vector[0]).To(BeNumerically("~", 0.8, 1e-5))
	g.Expect(doc.Terms["eval-topic"].Vector[1]).To(BeNumerically("~", 0.6, 1e-5))
}

// ── Vocab commands: bootstrap ─────────────────────────────────────────────────

// TestRunVocabBootstrap_CreatesTermNote verifies that bootstrap writes a
// bare-vocab-tagged definition note with the correct fact-note frontmatter
// (marshaled through factFrontmatterDoc) and body. Since the vault is empty,
// the family note mints first at id "1"; the term definition mints second at
// id "2" — both deterministic under an empty ListMD.
// TestRunVocabBootstrap_CreatesTermNote verifies that bootstrap writes a
// bare-vocab-tagged definition note with the correct fact-note frontmatter
// (marshaled through factFrontmatterDoc) and body. Since the vault is empty,
// the family note mints first at id "1"; the term definition mints second at
// id "2" — both deterministic under an empty ListMD.
func TestRunVocabBootstrap_CreatesTermNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{Term: "eval-methodology", Description: "how we evaluate"}}
	seedYAML, err := yaml.Marshal(seed)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	files := map[string][]byte{"/seed.yaml": seedYAML}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock:         func(string) (func(), error) { return func() {}, nil },
		ListMD:       func(string) ([]string, error) { return nil, nil },
		ReadFile:     func(path string) ([]byte, error) { return files[path], nil },
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	bootErr := cli.RunVocabBootstrap(t.Context(), args, deps, &buf)
	g.Expect(bootErr).NotTo(HaveOccurred())

	if bootErr != nil {
		return
	}

	termPath := "/vault/2.2026-07-02.vocab-eval-methodology-definition.md"
	g.Expect(written).To(HaveKey(termPath), "definition note must be written")

	content := string(written[termPath])
	g.Expect(content).To(ContainSubstring("type: fact"), "definition note must have type: fact")
	g.Expect(content).To(ContainSubstring("tags:\n    - vocab"), "definition note must carry the bare vocab tag")
	g.Expect(content).To(ContainSubstring("the eval-methodology vocab term"), "definition note must carry term name")
	g.Expect(content).To(ContainSubstring("how we evaluate"), "description must appear in body")
}

// ── Coverage: embedTermNote embed/write error paths ───────────────────────────

// TestRunVocabBootstrap_EmbedError_BootstrapSucceeds verifies that when the
// embedder returns an error for a term note, bootstrap warns-and-skips the sidecar
// (covers embedTermNote.embErr path).
// TestRunVocabBootstrap_EmbedError_BootstrapSucceeds verifies that when the
// embedder returns an error for a term note, bootstrap warns-and-skips the sidecar
// (covers embedTermNote.embErr path).
func TestRunVocabBootstrap_EmbedError_BootstrapSucceeds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{Term: "eval-topic", Description: "desc"}}
	seedYAML, yamlErr := yaml.Marshal(seed)
	g.Expect(yamlErr).NotTo(HaveOccurred())

	if yamlErr != nil {
		return
	}

	var warned bool

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return nil, nil },
		ReadFile: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".yaml") {
				return seedYAML, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:    func(_ string, _ []byte) error { return nil },
		WriteSidecar: func(_ string, _ []byte) error { return nil },
		Embedder:     &errEmbedder{},
		LogWarning:   func(string, ...any) { warned = true },
		Now:          func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabBootstrap(t.Context(), args, deps, &buf)).To(Succeed(),
		"bootstrap must succeed even when embed fails")
	g.Expect(warned).To(BeTrue(), "embed error must trigger a log warning")
}

// TestRunVocabBootstrap_ExemplarsInTermNoteBody verifies that seed exemplars
// are rendered into the term-note body — the body IS the term's embedding text,
// so exemplars must be present for the embedding to reflect member usage.
// TestRunVocabBootstrap_ExemplarsInTermNoteBody verifies that seed exemplars
// are rendered into the term-note body — the body IS the term's embedding text,
// so exemplars must be present for the embedding to reflect member usage.
func TestRunVocabBootstrap_ExemplarsInTermNoteBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{
		Term:        "eval-methodology",
		Description: "how we evaluate",
		Exemplars: []string{
			"designing an eval harness for memory-vs-baseline comparison",
			"choosing the miss population for a retrieval probe",
			"validating a cheap model tier against a gold standard",
		},
	}}
	seedYAML, err := yaml.Marshal(seed)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	files := map[string][]byte{"/seed.yaml": seedYAML}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock:         func(string) (func(), error) { return func() {}, nil },
		ListMD:       func(string) ([]string, error) { return nil, nil },
		ReadFile:     func(path string) ([]byte, error) { return files[path], nil },
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	bootErr := cli.RunVocabBootstrap(t.Context(), args, deps, &buf)
	g.Expect(bootErr).NotTo(HaveOccurred())

	if bootErr != nil {
		return
	}

	content := string(written["/vault/2.2026-07-02.vocab-eval-methodology-definition.md"])
	g.Expect(content).To(ContainSubstring("Exemplars:"), "body must carry an exemplar section")
	g.Expect(content).To(ContainSubstring("- designing an eval harness for memory-vs-baseline comparison"),
		"each exemplar must appear as a body list line")
	g.Expect(content).To(ContainSubstring("- validating a cheap model tier against a gold standard"),
		"all exemplars must be rendered")
}

// TestRunVocabBootstrap_Idempotent verifies that re-running bootstrap over an
// already-bootstrapped vault mints NOTHING for a term (or family) that
// already has a definition note — #678 Task 5's idempotency contract.
// TestRunVocabBootstrap_Idempotent verifies that re-running bootstrap over an
// already-bootstrapped vault mints NOTHING for a term (or family) that
// already has a definition note — #678 Task 5's idempotency contract.
func TestRunVocabBootstrap_Idempotent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{Term: "eval-methodology", Description: "updated description"}}
	seedYAML, err := yaml.Marshal(seed)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	existingDefinition := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: subj\npredicate: covers\n" +
		"object: how we evaluate\nluhmann: \"5\"\ncreated: \"2026-01-01\"\nsource: prior\ntags:\n    - vocab\n" +
		"---\n\nInformation learned: ...\n"
	existingFamily := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: the vocab tag family\n" +
		"predicate: covers\nobject: o\nluhmann: \"1\"\ncreated: \"2026-01-01\"\nsource: prior\n" +
		"vocab_version: \"1.0\"\ntags:\n    - vocab\n---\n\nInformation learned: ...\n"
	files := map[string][]byte{
		"/seed.yaml": seedYAML,
		"/vault/5.2026-01-01.vocab-eval-methodology-definition.md": []byte(existingDefinition),
		"/vault/1.2026-01-01.vocab-definition.md":                  []byte(existingFamily),
	}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{
				"5.2026-01-01.vocab-eval-methodology-definition.md",
				"1.2026-01-01.vocab-definition.md",
			}, nil
		},
		ReadFile:     func(path string) ([]byte, error) { return files[path], nil },
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	bootErr := cli.RunVocabBootstrap(t.Context(), args, deps, &buf)
	g.Expect(bootErr).NotTo(HaveOccurred(), "re-running bootstrap over an already-bootstrapped vault must not error")

	if bootErr != nil {
		return
	}

	for path := range written {
		g.Expect(path).NotTo(ContainSubstring("vocab-eval-methodology-definition"),
			"a second bootstrap run must not re-mint an existing term's definition note")
		g.Expect(path).NotTo(HaveSuffix("vocab-definition.md"),
			"a second bootstrap run must not re-mint an existing family note")
	}
}

// TestRunVocabBootstrap_NoIndexFile verifies that bootstrap NEVER writes
// vocab.index.md — the index is retired (#678 Task 5); term identity lives
// solely in the minted definition notes' tags:.
// TestRunVocabBootstrap_NoIndexFile verifies that bootstrap NEVER writes
// vocab.index.md — the index is retired (#678 Task 5); term identity lives
// solely in the minted definition notes' tags:.
func TestRunVocabBootstrap_NoIndexFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{Term: "eval-methodology", Description: "how we evaluate"}}
	seedYAML, err := yaml.Marshal(seed)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	files := map[string][]byte{"/seed.yaml": seedYAML}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock:         func(string) (func(), error) { return func() {}, nil },
		ListMD:       func(string) ([]string, error) { return nil, nil },
		ReadFile:     func(path string) ([]byte, error) { return files[path], nil },
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	bootErr := cli.RunVocabBootstrap(t.Context(), args, deps, &buf)
	g.Expect(bootErr).NotTo(HaveOccurred())

	if bootErr != nil {
		return
	}

	g.Expect(written).NotTo(HaveKey("/vault/vocab.index.md"), "vocab.index.md must never be written — index retired")

	for path := range written {
		g.Expect(path).NotTo(HaveSuffix("vocab.index.md"), "no write may target the retired index file")
	}
}

// TestRunVocabBootstrap_SidecarWriteError_BootstrapSucceeds verifies that when
// WriteSidecar fails after a successful embed, bootstrap warns and continues
// (covers embedTermNote.writeErr path).
// TestRunVocabBootstrap_SidecarWriteError_BootstrapSucceeds verifies that when
// WriteSidecar fails after a successful embed, bootstrap warns and continues
// (covers embedTermNote.writeErr path).
func TestRunVocabBootstrap_SidecarWriteError_BootstrapSucceeds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{Term: "eval-topic", Description: "desc"}}
	seedYAML, yamlErr := yaml.Marshal(seed)
	g.Expect(yamlErr).NotTo(HaveOccurred())

	if yamlErr != nil {
		return
	}

	var warned bool

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return nil, nil },
		ReadFile: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".yaml") {
				return seedYAML, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:    func(_ string, _ []byte) error { return nil },
		WriteSidecar: func(_ string, _ []byte) error { return errors.New("sidecar write failed") },
		Embedder:     &fakeEmbedder{},
		LogWarning:   func(string, ...any) { warned = true },
		Now:          func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabBootstrap(t.Context(), args, deps, &buf)).To(Succeed(),
		"bootstrap must succeed even when sidecar write fails")
	g.Expect(warned).To(BeTrue(), "sidecar write error must trigger a log warning")
}

// ── Coverage: writeAndEmbedSeedTerms warning path ────────────────────────────

// TestRunVocabBootstrap_TermNoteWriteError_LogsWarning verifies that when
// WriteFile fails for a term note, bootstrap logs a warning but continues
// (covers writeAndEmbedSeedTerms warning path).
// TestRunVocabBootstrap_TermNoteWriteError_LogsWarning verifies that when
// WriteFile fails for a term note, bootstrap logs a warning but continues
// (covers writeAndEmbedSeedTerms warning path).
func TestRunVocabBootstrap_TermNoteWriteError_LogsWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{Term: "eval-topic", Description: "desc"}}
	seedYAML, yamlErr := yaml.Marshal(seed)
	g.Expect(yamlErr).NotTo(HaveOccurred())

	if yamlErr != nil {
		return
	}

	var warned bool

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return nil, nil },
		ReadFile: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".yaml") {
				return seedYAML, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, _ []byte) error {
			if strings.Contains(path, "vocab-eval-topic-definition") {
				return errors.New("write failed")
			}

			return nil
		},
		WriteSidecar: func(_ string, _ []byte) error { return nil },
		LogWarning:   func(string, ...any) { warned = true },
		Now:          func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabBootstrap(t.Context(), args, deps, &buf)).To(Succeed(),
		"bootstrap must succeed even when term note write fails")
	g.Expect(warned).To(BeTrue(), "term note write failure must trigger log warning")
}

// ── Coverage: embedTermNote via mock embedder ─────────────────────────────────

// TestRunVocabBootstrap_WithMockEmbedder verifies that when a mock embedder
// is wired, embedTermNote embeds the term note and writes the sidecar.
// TestRunVocabBootstrap_WithMockEmbedder verifies that when a mock embedder
// is wired, embedTermNote embeds the term note and writes the sidecar.
func TestRunVocabBootstrap_WithMockEmbedder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{Term: "eval-methodology", Description: "how we evaluate"}}
	seedYAML, err := yaml.Marshal(seed)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var sidecarWritten []byte

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return nil, nil },
		ReadFile: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
				return seedYAML, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:    func(_ string, _ []byte) error { return nil },
		WriteSidecar: func(_ string, data []byte) error { sidecarWritten = data; return nil },
		Embedder:     &fakeEmbedder{},
		Now:          func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{
		Vault:    "/vault",
		SeedFile: "/seed.yaml",
		Floor:    0.30,
	}

	var stdout strings.Builder

	g.Expect(cli.RunVocabBootstrap(t.Context(), args, deps, &stdout)).To(Succeed())
	g.Expect(sidecarWritten).NotTo(BeEmpty(), "sidecar must be written when mock embedder is wired")
}

// TestRunVocabPropose_BumpsMinorVersion verifies that propose increments the
// minor component of the current vocab_version — read from the
// vocab-definition family note (#678 Task 4) — and rewrites the bump onto
// that same family note.
// TestRunVocabPropose_BumpsMinorVersion verifies that propose increments the
// minor component of the current vocab_version — read from the
// vocab-definition family note (#678 Task 4) — and rewrites the bump onto
// that same family note.
func TestRunVocabPropose_BumpsMinorVersion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const familyNotePath = "/vault/210.2026-07-02.vocab-definition.md"

	familyNote := "---\ntype: fact\nvocab_version: \"1.3\"\ntags:\n    - vocab\n---\n\nVocab family root.\n"
	files := map[string][]byte{
		familyNotePath: []byte(familyNote),
	}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{"210.2026-07-02.vocab-definition.md"}, nil
		},
		ReadFile:     func(path string) ([]byte, error) { return files[path], nil },
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabProposeArgs{Vault: "/vault", Term: "new-term", Description: "desc"}

	var buf strings.Builder

	g.Expect(cli.RunVocabPropose(t.Context(), args, deps, &buf)).To(Succeed())

	familyContent := string(written[familyNotePath])
	g.Expect(familyContent).To(ContainSubstring(`vocab_version: "1.4"`),
		"the family note must carry the bumped version")

	newTermPath := "/vault/211.2026-07-02.vocab-new-term-definition.md"
	g.Expect(written).To(HaveKey(newTermPath), "propose must mint the new term's definition note")
}

// ── Vocab commands: propose ───────────────────────────────────────────────────

// TestRunVocabPropose_CreatesTermNote verifies that propose mints a new
// bare-vocab-tagged definition note with the supplied term name and
// description. The vault is empty, so the fresh mint allocates id "1".
// TestRunVocabPropose_CreatesTermNote verifies that propose mints a new
// bare-vocab-tagged definition note with the supplied term name and
// description. The vault is empty, so the fresh mint allocates id "1".
func TestRunVocabPropose_CreatesTermNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock:         func(string) (func(), error) { return func() {}, nil },
		ListMD:       func(string) ([]string, error) { return nil, nil },
		ReadFile:     func(string) ([]byte, error) { return nil, &testNotFoundError{path: "missing"} },
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabProposeArgs{
		Vault:       "/vault",
		Term:        "new-insight",
		Description: "tracking novel insights",
	}

	var buf strings.Builder

	propErr := cli.RunVocabPropose(t.Context(), args, deps, &buf)
	g.Expect(propErr).NotTo(HaveOccurred())

	if propErr != nil {
		return
	}

	termPath := "/vault/1.2026-07-02.vocab-new-insight-definition.md"
	g.Expect(written).To(HaveKey(termPath), "propose must create the definition note")
	g.Expect(string(written[termPath])).To(ContainSubstring("the new-insight vocab term"),
		"definition note must carry term name")
	g.Expect(string(written[termPath])).To(ContainSubstring("tracking novel insights"),
		"definition note must carry description")
	g.Expect(string(written[termPath])).To(ContainSubstring("tags:\n    - vocab"),
		"definition note must carry the bare vocab tag")
}

// TestRunVocabPropose_TermNoteWriteError_LogsWarning verifies that when
// WriteFile fails for the new definition note, RunVocabPropose logs a
// warning and still succeeds (covers mintDefinitionNote's write-error path).
// TestRunVocabPropose_TermNoteWriteError_LogsWarning verifies that when
// WriteFile fails for the new definition note, RunVocabPropose logs a
// warning and still succeeds (covers mintDefinitionNote's write-error path).
func TestRunVocabPropose_TermNoteWriteError_LogsWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var loggedMsg string

	deps := cli.VocabDeps{
		Lock:     func(string) (func(), error) { return func() {}, nil },
		ListMD:   func(string) ([]string, error) { return nil, nil },
		ReadFile: func(string) ([]byte, error) { return nil, &testNotFoundError{path: "missing"} },
		WriteFile: func(path string, _ []byte) error {
			if path == "/vault/1.2026-07-02.vocab-new-term-definition.md" {
				return errors.New("disk full")
			}

			return nil
		},
		WriteSidecar: func(string, []byte) error { return nil },
		LogWarning:   func(format string, args ...any) { loggedMsg = fmt.Sprintf(format, args...) },
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabProposeArgs{Vault: "/vault", Term: "new-term", Description: "desc"}

	var buf strings.Builder

	proposeErr := cli.RunVocabPropose(t.Context(), args, deps, &buf)
	g.Expect(proposeErr).NotTo(HaveOccurred(), "propose still succeeds; the failed definition-note write is only logged")

	g.Expect(loggedMsg).To(ContainSubstring("embedding new-term failed"))
}

// TestRunVocabRefit_AppliesRemovals verifies that refit deletes the removed
// term's definition note + sidecar and clears vocab/<term> from member tags.
func TestRunVocabStats_CountsQAPairs(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	names := []string{
		"qa.2026-07-03.slug.q.md",   // complete pair with the next entry: 1
		"qa.2026-07-03.slug.a.md",   // pair member
		"qa.2026-07-03.orphan.q.md", // no matching .a.md: not counted
		"100.note.md",
	}

	deps := cli.VocabStatsDeps{
		ListMD:   func(string) ([]string, error) { return names, nil },
		ReadFile: func(string) ([]byte, error) { return nil, os.ErrNotExist },
	}

	var buf strings.Builder

	g.Expect(cli.RunVocabStats(cli.VocabStatsArgs{Vault: "/vault"}, deps, &buf)).To(Succeed())
	g.Expect(buf.String()).To(ContainSubstring("qa pairs: 1"))
}

// ── Vocab commands: stats ─────────────────────────────────────────────────────

// TestRunVocabStats_NoTerms_PrintsZeroStats verifies that stats on a vault
// with no vocab term notes prints a report without panicking.
// TestRunVocabStats_NoTerms_PrintsZeroStats verifies that stats on a vault
// with no vocab term notes prints a report without panicking.
func TestRunVocabStats_NoTerms_PrintsZeroStats(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.VocabStatsDeps{
		ListMD:   func(string) ([]string, error) { return nil, nil },
		ReadFile: func(string) ([]byte, error) { return nil, nil },
	}

	args := cli.VocabStatsArgs{Vault: "/vault"}

	var buf strings.Builder

	statsErr := cli.RunVocabStats(args, deps, &buf)
	g.Expect(statsErr).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(ContainSubstring("terms: 0"))
}

// ── Coverage: extractNoteVocabTags edge cases ─────────────────────────────────

// TestRunVocabStats_NoteWithNoFrontmatter verifies that a note with no YAML
// frontmatter is silently excluded from member counts (extractNoteVocabTags
// returns nil, false for the !ok path). The note does not appear in totalNotes.
// TestRunVocabStats_NoteWithNoFrontmatter verifies that a note with no YAML
// frontmatter is silently excluded from member counts (extractNoteVocabTags
// returns nil, false for the !ok path). The note does not appear in totalNotes.
func TestRunVocabStats_NoteWithNoFrontmatter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.VocabStatsDeps{
		// Two names: one no-frontmatter note (excluded) + one tagged note (counted).
		ListMD: func(string) ([]string, error) {
			return []string{"1aa.2026-01-01.nofm.md", "1ab.2026-01-02.tagged.md"}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if strings.Contains(path, "nofm") {
				return []byte("no frontmatter here\njust plain text\n"), nil
			}

			return []byte("---\ntype: feedback\nvocab: [eval-methodology]\n---\nbody\n"), nil
		},
	}

	args := cli.VocabStatsArgs{Vault: "/vault"}

	var stdout strings.Builder

	g.Expect(cli.RunVocabStats(args, deps, &stdout)).To(Succeed())
	// Only the tagged note is counted; the no-frontmatter note is excluded.
	g.Expect(stdout.String()).To(ContainSubstring("member-notes: 1"),
		"only the tagged note must appear in member-notes count")
}

func TestRunVocabStats_ReadsRefitPendingFromCentroids(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroidsDoc := cli.ExportVocabCentroidsDoc{
		RefitPending: true,
		RefitReason:  "hub: agentic-recall-triggers (30%)",
	}

	centroidsData, marshalErr := json.Marshal(centroidsDoc)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	deps := cli.VocabStatsDeps{
		ListMD: func(string) ([]string, error) { return nil, nil },
		ReadFile: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, "vocab.centroids.json") {
				return centroidsData, nil
			}

			return nil, os.ErrNotExist
		},
	}

	var buf strings.Builder

	g.Expect(cli.RunVocabStats(cli.VocabStatsArgs{Vault: "/vault"}, deps, &buf)).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(ContainSubstring("REFIT_PENDING"))
}

// TestRunVocabStats_ReportsHubAndOrphan verifies hub (>25% of vault) and
// orphan (<2 members) detection in the stats output.
// TestRunVocabStats_ReportsHubAndOrphan verifies hub (>25% of vault) and
// orphan (<2 members) detection in the stats output.
func TestRunVocabStats_ReportsHubAndOrphan(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// 4 total notes: 3 tagged eval-methodology (hub, 75%), 0 tagged scope-discipline (orphan).
	evalDefinitionNote := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: subj\npredicate: covers\n" +
		"object: eval\nluhmann: \"210\"\ncreated: \"2026-07-02\"\nsource: test\ntags:\n    - vocab\n" +
		"---\n\nInformation learned: ...\n"
	scopeDefinitionNote := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: subj\npredicate: covers\n" +
		"object: scope\nluhmann: \"211\"\ncreated: \"2026-07-02\"\nsource: test\ntags:\n    - vocab\n" +
		"---\n\nInformation learned: ...\n"
	// 3 notes tagged eval-methodology, 1 untagged.
	taggedNote := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\ntags:\n    - vocab/eval-methodology\n---\n\n" +
		"Lesson learned: when ctx, a.\n\n"
	untaggedNote := "---\ntype: feedback\nsituation: ctx2\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"2aa\"\ncreated: 2026-01-02\nsource: test\n---\n\nLesson learned: when ctx2, a.\n\n"

	allFiles := map[string]string{
		"/vault/210.2026-07-02.vocab-eval-methodology-definition.md": evalDefinitionNote,
		"/vault/211.2026-07-02.vocab-scope-discipline-definition.md": scopeDefinitionNote,
		"/vault/1aa.2026-01-01.note.md":                              taggedNote,
		"/vault/1bb.2026-01-01.note.md":                              taggedNote,
		"/vault/1cc.2026-01-01.note.md":                              taggedNote,
		"/vault/2aa.2026-01-02.note.md":                              untaggedNote,
	}
	allNames := []string{
		"210.2026-07-02.vocab-eval-methodology-definition.md",
		"211.2026-07-02.vocab-scope-discipline-definition.md",
		"1aa.2026-01-01.note.md",
		"1bb.2026-01-01.note.md",
		"1cc.2026-01-01.note.md",
		"2aa.2026-01-02.note.md",
	}

	deps := cli.VocabStatsDeps{
		ListMD: func(string) ([]string, error) { return allNames, nil },
		ReadFile: func(path string) ([]byte, error) {
			name := filepath.Base(path)

			for fullPath, content := range allFiles {
				if filepath.Base(fullPath) == name {
					return []byte(content), nil
				}
			}

			return nil, &testNotFoundError{path: path}
		},
	}

	args := cli.VocabStatsArgs{Vault: "/vault"}

	var buf strings.Builder

	statsErr := cli.RunVocabStats(args, deps, &buf)
	g.Expect(statsErr).NotTo(HaveOccurred())

	if statsErr != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("hub"), "hub term must be flagged")
	g.Expect(output).To(ContainSubstring("orphan"), "orphan term must be flagged")
	g.Expect(output).To(ContainSubstring("eval-methodology"), "hub term name must appear")
	g.Expect(output).To(ContainSubstring("verdict: OK"),
		"hub concentration is diagnostic-only — it must not force a REFIT_PENDING verdict")
	g.Expect(output).To(ContainSubstring("scope-discipline"), "orphan term name must appear")
}

// TestRunVocabStats_UntaggedRateHigh_DiagnosticOnly verifies the trigger
// collapse: an untagged rate above 8% is flagged [high] on the untagged-rate
// line as a diagnostic, and the verdict stays OK — the rate carries no
// REFIT_PENDING force.
func TestRunVocabStats_UntaggedRateHigh_DiagnosticOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	untaggedNote := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\nLesson learned: when ctx, a.\n\n"

	deps := cli.VocabStatsDeps{
		ListMD: func(string) ([]string, error) {
			return []string{"1aa.2026-01-01.note.md", "2aa.2026-01-02.note.md"}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".md") {
				return []byte(untaggedNote), nil
			}

			return nil, &testNotFoundError{path: path}
		},
	}

	var buf strings.Builder

	statsErr := cli.RunVocabStats(cli.VocabStatsArgs{Vault: "/vault"}, deps, &buf)
	g.Expect(statsErr).NotTo(HaveOccurred())

	if statsErr != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("untagged-rate: 100.0% [high]"),
		"untagged rate above 8% must be flagged as a [high] diagnostic")
	g.Expect(output).To(ContainSubstring("verdict: OK"),
		"untagged rate is diagnostic-only — it must not force a REFIT_PENDING verdict")
	g.Expect(output).NotTo(ContainSubstring("REFIT_PENDING"))
}

// TestRunVocabTagDefinitions_PublicAPI verifies that the public RunVocabTagDefinitions
// function (with VocabTagDefinitionsArgs) works end-to-end through its closure, calling
// the internal runVocabTagDefinitions implementation.
// TestRunVocabTagDefinitions_PublicAPI verifies that the public RunVocabTagDefinitions
// function (with VocabTagDefinitionsArgs) works end-to-end through its closure, calling
// the internal runVocabTagDefinitions implementation.
func TestRunVocabTagDefinitions_PublicAPI(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	// Write test definition notes
	bareDefinitionContent := "---\ntype: fact\ntags:\n    - vocab\n---\n\nTest definition.\n"
	writeNote(t, vault, "1.2026-07-10.vocab-test-definition.md", bareDefinitionContent)

	deps := cli.ExportNewVocabDeps(cli.ExportNewTestOsDeps())
	args := cli.VocabTagDefinitionsArgs{Vault: vault}

	var buf bytes.Buffer

	err := cli.RunVocabTagDefinitions(context.Background(), args, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	output := buf.String()
	g.Expect(output).To(ContainSubstring("added"))
	g.Expect(output).To(ContainSubstring("vocab-test-definition"))
}

// TestSlugFromNoteFilename table-tests the "<id>.<date>.<slug>.md" parser.
// TestSlugFromNoteFilename table-tests the "<id>.<date>.<slug>.md" parser.
func TestSlugFromNoteFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "family note", filename: "210.2026-07-10.vocab-definition.md", want: "vocab-definition"},
		{
			name:     "term definition with dashes",
			filename: "211.2026-07-10.vocab-retrieval-design-definition.md",
			want:     "vocab-retrieval-design-definition",
		},
		{name: "non-md extension", filename: "210.2026-07-10.vocab-definition.txt", want: ""},
		{name: "too few segments", filename: "210.md", want: ""},
		{
			// Real vault shape: a slug containing dots. Everything after the
			// 2nd dot-segment (id, date) is the slug, dots included.
			name:     "dots in slug (real vault shape)",
			filename: "36.2026-06-17.recency-band-conversion1.2026-06-17.note-recency-decay.md",
			want:     "recency-band-conversion1.2026-06-17.note-recency-decay",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(cli.ExportSlugFromNoteFilename(tc.filename)).To(Equal(tc.want))
		})
	}
}

// ── Coverage: vocabTargets bootstrap closure ──────────────────────────────────

// TestTargets_VocabBootstrapNonExistentSeed exercises the vocab bootstrap
// closure end-to-end (covers newVocabDeps and the bootstrap closure in
// vocabTargets). The seed file does not exist, so the command returns an error.
// TestTargets_VocabBootstrapNonExistentSeed exercises the vocab bootstrap
// closure end-to-end (covers newVocabDeps and the bootstrap closure in
// vocabTargets). The seed file does not exist, so the command returns an error.
func TestTargets_VocabBootstrapNonExistentSeed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(newTestDeps(&stdout, &stderr))

	_, targErr := targ.Execute(
		[]string{"engram", "vocab", "bootstrap", "--vault", vault, "--seed", filepath.Join(vault, "nonexistent.yaml")},
		targets...,
	)
	if targErr != nil {
		stderr.WriteString(targErr.Error())
	}

	g.Expect(stderr.String()).NotTo(BeEmpty(), "error must appear in stderr when seed is missing")
}

// ── Coverage: vocabTargets propose and refit closures ─────────────────────────

// TestTargets_VocabProposeCreatesNote exercises the vocab propose closure via
// Targets() on an empty vault (covers the propose target wiring in vocabTargets).
// A pre-bootstrap vault has no vocab-definition family note, so propose
// succeeds and emits exactly the documented missing-family-note warning
// (bumpAndPersistVocabVersion: logged and skipped, not fatal). Before #700
// T12 that warning leaked to the real process stderr; the composed
// logWarningTo(d.Stderr) routes it to the injected stderr, where this test
// pins it.
// TestTargets_VocabProposeCreatesNote exercises the vocab propose closure via
// Targets() on an empty vault (covers the propose target wiring in vocabTargets).
// A pre-bootstrap vault has no vocab-definition family note, so propose
// succeeds and emits exactly the documented missing-family-note warning
// (bumpAndPersistVocabVersion: logged and skipped, not fatal). Before #700
// T12 that warning leaked to the real process stderr; the composed
// logWarningTo(d.Stderr) routes it to the injected stderr, where this test
// pins it.
func TestTargets_VocabProposeCreatesNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(newTestDeps(&stdout, &stderr))

	_, targErr := targ.Execute(
		[]string{"engram", "vocab", "propose", "--vault", vault, "--term", "test-term", "--description", "a test"},
		targets...,
	)
	if targErr != nil {
		stderr.WriteString(targErr.Error())
	}

	g.Expect(stdout.String()).To(ContainSubstring("vocab propose: created test-term"),
		"vocab propose must succeed on an empty vault")
	g.Expect(stderr.String()).To(Equal(
		"warning: vocab propose: writing family note version: "+
			"vocab: family definition note (vocab-definition) not found\n"),
		"pre-bootstrap propose warns about the absent family note via the injected stderr — nothing else")
}

// TestTargets_VocabRefitDryRun exercises the vocab refit closure via
// Targets(). An empty vault has no clusterable structure, so --dry-run
// reports the unchanged vocabulary without erroring.
func TestTargets_VocabRefitDryRun(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(newTestDeps(&stdout, &stderr))

	_, targErr := targ.Execute(
		[]string{"engram", "vocab", "refit", "--vault", vault, "--dry-run"},
		targets...,
	)
	g.Expect(targErr).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("no structure"),
		"an empty vault derives no clusters and leaves the vocabulary unchanged")
}

// ── Vocab integration: OS wiring ─────────────────────────────────────────────

// TestTargets_VocabStatsEmpty exercises the vocab stats closure end-to-end
// through Targets() with an empty vault so newVocabStatsDeps wiring is covered.
// TestTargets_VocabStatsEmpty exercises the vocab stats closure end-to-end
// through Targets() with an empty vault so newVocabStatsDeps wiring is covered.
func TestTargets_VocabStatsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	vault := t.TempDir()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(newTestDeps(&stdout, &stderr))

	_, targErr := targ.Execute([]string{"engram", "vocab", "stats", "--vault", vault}, targets...)
	if targErr != nil {
		stderr.WriteString(targErr.Error())
	}

	g.Expect(stderr.String()).To(BeEmpty(), "vocab stats on empty vault must not produce errors")
	g.Expect(stdout.String()).To(ContainSubstring("terms: 0"), "empty vault must report zero terms")
}

// TestTermFromDefinitionSlug table-tests the "vocab-<term>-definition" slug
// parser: the family slug and non-matching slugs return false; a term may
// itself contain dashes.
// TestTermFromDefinitionSlug table-tests the "vocab-<term>-definition" slug
// parser: the family slug and non-matching slugs return false; a term may
// itself contain dashes.
func TestTermFromDefinitionSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		slug     string
		wantTerm string
		wantOK   bool
	}{
		{name: "family slug is not a term", slug: "vocab-definition", wantTerm: "", wantOK: false},
		{name: "simple term", slug: "vocab-retrieval-design-definition", wantTerm: "retrieval-design", wantOK: true},
		{name: "unrelated slug", slug: "some-other-note-slug", wantTerm: "", wantOK: false},
		{
			name:     "term containing dashes",
			slug:     "vocab-skill-and-guidance-design-definition",
			wantTerm: "skill-and-guidance-design", wantOK: true,
		},
		{name: "empty term between prefix and suffix", slug: "vocab--definition", wantTerm: "", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			gotTerm, gotOK := cli.ExportTermFromDefinitionSlug(tc.slug)
			g.Expect(gotOK).To(Equal(tc.wantOK))
			g.Expect(gotTerm).To(Equal(tc.wantTerm))
		})
	}
}

// ── Task 1: vocabCentroidsDoc new fields round-trip ──────────────────────────

func TestVocabCentroidsDoc_NewFieldsRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	doc := cli.ExportVocabCentroidsDoc{
		SchemaVersion: 1,
		RefitPending:  true,
		RefitReason:   "growth: 41 notes, 15 days",
		LastRefit:     &cli.ExportVocabLastRefitDoc{NoteCount: 100, Date: "2026-07-03"},
		Terms:         map[string]cli.ExportVocabCentroidEntry{"x": {MemberCount: 3}},
	}
	data, marshalErr := json.Marshal(doc)

	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	var got cli.ExportVocabCentroidsDoc

	g.Expect(json.Unmarshal(data, &got)).NotTo(HaveOccurred())

	if err := json.Unmarshal(data, &got); err != nil {
		return
	}

	g.Expect(got.RefitPending).To(BeTrue())
	g.Expect(got.RefitReason).To(Equal("growth: 41 notes, 15 days"))
	g.Expect(got.LastRefit).NotTo(BeNil())

	if got.LastRefit == nil {
		return
	}

	g.Expect(got.LastRefit.NoteCount).To(Equal(100))
	g.Expect(got.LastRefit.Date).To(Equal("2026-07-03"))
}

func TestVocabCentroidsDoc_ZeroValueOmitted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	doc := cli.ExportVocabCentroidsDoc{SchemaVersion: 1}

	data, marshalErr := json.Marshal(doc)

	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	jsonStr := string(data)
	g.Expect(jsonStr).NotTo(ContainSubstring("refit_pending"))
	g.Expect(jsonStr).NotTo(ContainSubstring("refit_reason"))
	g.Expect(jsonStr).NotTo(ContainSubstring("last_refit"))
}

// ── Task 5: bootstrap/refit invariants (Gate A ask-alignment findings) ───────

// TestVocabFamilyNote_NeverEnumeratesTerms bootstraps a scratch vault with two
// terms and asserts the minted vocab-definition family note's full content
// contains NEITHER term name. A maintained term list in the family note is
// the stale-index problem reborn (issue #678's most explicit warning).
// TestVocabFamilyNote_NeverEnumeratesTerms bootstraps a scratch vault with two
// terms and asserts the minted vocab-definition family note's full content
// contains NEITHER term name. A maintained term list in the family note is
// the stale-index problem reborn (issue #678's most explicit warning).
func TestVocabFamilyNote_NeverEnumeratesTerms(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	seedPath := filepath.Join(vault, "seed.yaml")

	seed := []cli.SeedTerm{
		{Term: "retrieval-design", Description: "keeps queries scoped to task-relevant vault slices"},
		{Term: "token-budget", Description: "tracks payload size against context limits"},
	}
	seedYAML, marshalErr := yaml.Marshal(seed)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	g.Expect(os.WriteFile(seedPath, seedYAML, 0o600)).To(Succeed())

	deps := cli.ExportNewVocabDeps(cli.ExportNewTestOsDeps())
	deps.Embedder = &fakeEmbedder{}
	deps.Now = func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) }

	args := cli.VocabBootstrapArgs{Vault: vault, SeedFile: seedPath}

	var buf strings.Builder

	bootErr := cli.RunVocabBootstrap(t.Context(), args, deps, &buf)
	g.Expect(bootErr).NotTo(HaveOccurred())

	if bootErr != nil {
		return
	}

	entries, readDirErr := os.ReadDir(vault)
	g.Expect(readDirErr).NotTo(HaveOccurred())

	var familyContent string

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".vocab-definition.md") {
			data, readErr := os.ReadFile(filepath.Join(vault, entry.Name()))
			g.Expect(readErr).NotTo(HaveOccurred())

			familyContent = string(data)

			break
		}
	}

	g.Expect(familyContent).NotTo(BeEmpty(), "bootstrap must mint the family note")
	g.Expect(familyContent).NotTo(ContainSubstring("retrieval-design"),
		"the family note must never enumerate a term name")
	g.Expect(familyContent).NotTo(ContainSubstring("token-budget"),
		"the family note must never enumerate a term name")
}

// ── Coverage: backfill subcommand (Group 4) ─────────────────────────────────

// TestVocabTagDefinitionsBackfill_AddsAndIsIdempotent verifies the backfill
// subcommand adds missing self-tags to definitions and is idempotent.
// TestVocabTagDefinitionsBackfill_AddsAndIsIdempotent verifies the backfill
// subcommand adds missing self-tags to definitions and is idempotent.
func TestVocabTagDefinitionsBackfill_AddsAndIsIdempotent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := "/vault"

	// Bare-only definition (current state before backfill)
	bareDefinitionContent := "---\ntype: fact\ntags:\n    - vocab\n---\n\nA test term.\n"
	// Definition that already has the self-tag
	selfTaggedContent := "---\ntype: fact\ntags:\n    - vocab\n    - vocab/already-tagged\n---\n\nAlready tagged.\n"
	// Family note (should be skipped)
	familyContent := "---\ntype: fact\ntags:\n    - vocab\nvocab_version: \"1.0\"\n---\n\nFamily note.\n"
	// Non-definition note (should be skipped)
	memberContent := "---\ntype: fact\ntags:\n    - other\n---\n\nMember note.\n"

	files := map[string][]byte{
		"/vault/1.2026-07-10.vocab-bare-term-definition.md":      []byte(bareDefinitionContent),
		"/vault/2.2026-07-10.vocab-already-tagged-definition.md": []byte(selfTaggedContent),
		"/vault/3.2026-07-10.vocab-definition.md":                []byte(familyContent),
		"/vault/4.2026-07-10.member.md":                          []byte(memberContent),
	}

	written := map[string][]byte{}
	output := &bytes.Buffer{}

	// Call-counting stubs for the no-re-embed guard
	sidecarWriteCount := 0
	embedderCallCount := 0

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) {
			return func() {}, nil
		},
		ListMD: func(string) ([]string, error) {
			return []string{
				"1.2026-07-10.vocab-bare-term-definition.md",
				"2.2026-07-10.vocab-already-tagged-definition.md",
				"3.2026-07-10.vocab-definition.md",
				"4.2026-07-10.member.md",
			}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, data []byte) error {
			written[path] = data
			files[path] = data // Update for second run

			return nil
		},
		WriteSidecar: func(_ string, _ []byte) error {
			sidecarWriteCount++
			return nil
		},
		Embedder:   &testEmbedder{callCount: &embedderCallCount},
		LogWarning: func(string, ...any) {},
		Now:        func() time.Time { return time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC) },
	}

	// First run: should add self-tags
	err := cli.ExportRunVocabTagDefinitions(context.Background(), vault, deps, output)
	g.Expect(err).NotTo(HaveOccurred())

	output1 := output.String()
	g.Expect(output1).To(ContainSubstring("added"), "first run should report added self-tags")
	g.Expect(output1).To(ContainSubstring("bare-term"), "should process bare-term definition")
	g.Expect(output1).To(ContainSubstring("family"), "should report family note as skipped")

	// No-re-embed guard: backfill should not trigger sidecar writes or embedding
	g.Expect(sidecarWriteCount).To(Equal(0), "backfill must not create sidecars — self-tags are non-semantic")
	g.Expect(embedderCallCount).To(Equal(0), "backfill must not invoke the embedder")

	// Verify the bare-term definition was written with self-tag
	g.Expect(written).To(HaveKey("/vault/1.2026-07-10.vocab-bare-term-definition.md"))
	updatedContent := written["/vault/1.2026-07-10.vocab-bare-term-definition.md"]
	frontmatter, ok := cli.ExportSplitFrontmatter(updatedContent)

	g.Expect(ok).To(BeTrue())

	var doc struct {
		Tags []string `yaml:"tags"`
	}

	err = yaml.Unmarshal(frontmatter, &doc)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(doc.Tags).To(Equal([]string{"vocab", "vocab/bare-term"}))

	// Second run: should be idempotent (already-present everywhere)

	written = make(map[string][]byte) // reset write tracking

	output.Reset()
	err = cli.ExportRunVocabTagDefinitions(context.Background(), vault, deps, output)
	g.Expect(err).NotTo(HaveOccurred())

	output2 := output.String()
	g.Expect(output2).To(ContainSubstring("already present"), "second run should report already-present")
	g.Expect(written).To(BeEmpty(), "second run should not write anything")
}

// TestWriteVocabVersionToFamilyNote_ListMDError_ReturnsWrappedError covers the
// listMD-error branch.
// TestWriteVocabVersionToFamilyNote_ListMDError_ReturnsWrappedError covers the
// listMD-error branch.
func TestWriteVocabVersionToFamilyNote_ListMDError_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	writeErr := cli.ExportWriteVocabVersionToFamilyNote(
		"/vault", "1.1",
		func(string) ([]string, error) { return nil, errors.New("list error") },
		func(string) ([]byte, error) { return nil, errors.New("unused") },
		func(string, []byte) error { return nil },
	)

	g.Expect(writeErr).To(HaveOccurred())
	g.Expect(writeErr.Error()).To(ContainSubstring("listing vault"))
}

// TestWriteVocabVersionToFamilyNote_MissingFamilyNote_ReturnsSentinel verifies
// the sentinel error when no family note exists.
// TestWriteVocabVersionToFamilyNote_MissingFamilyNote_ReturnsSentinel verifies
// the sentinel error when no family note exists.
func TestWriteVocabVersionToFamilyNote_MissingFamilyNote_ReturnsSentinel(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	writeNote(t, vault, "1.2026-07-10.other.md", "---\ntype: fact\n---\n\nBody.\n")

	osFS := cli.ExportNewVaultFS(realFSForTest())

	writeErr := cli.ExportWriteVocabVersionToFamilyNote(vault, "1.1", osFS.ListMD, osFS.ReadFile,
		func(string, []byte) error { return nil })

	g.Expect(writeErr).To(MatchError(cli.ErrVocabFamilyNoteMissing))
}

// TestWriteVocabVersionToFamilyNote_RewritesOnlyFamilyNote verifies the
// version-bump write site rewrites vocab_version in place on the family note
// ONLY — the term-definition notes in the same vault are never touched.
// TestWriteVocabVersionToFamilyNote_RewritesOnlyFamilyNote verifies the
// version-bump write site rewrites vocab_version in place on the family note
// ONLY — the term-definition notes in the same vault are never touched.
func TestWriteVocabVersionToFamilyNote_RewritesOnlyFamilyNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := buildTask4DefinitionFixture(t)

	osFS := cli.ExportNewVaultFS(realFSForTest())

	written := make(map[string][]byte)
	writeFile := func(path string, data []byte) error {
		written[path] = data
		return nil
	}

	writeErr := cli.ExportWriteVocabVersionToFamilyNote(vault, "6.1", osFS.ListMD, osFS.ReadFile, writeFile)
	g.Expect(writeErr).NotTo(HaveOccurred())

	g.Expect(written).To(HaveLen(1), "only the family note may be rewritten")

	familyPath := filepath.Join(vault, "210.2026-07-10.vocab-definition.md")
	g.Expect(written).To(HaveKey(familyPath))
	g.Expect(string(written[familyPath])).To(ContainSubstring(`vocab_version: "6.1"`))
	g.Expect(string(written[familyPath])).To(ContainSubstring("Vocab family root."),
		"the body must survive the rewrite untouched")
}

// errEmbedder is a test-only embed.Embedder that always returns an error on Embed.
// errEmbedder is a test-only embed.Embedder that always returns an error on Embed.
type errEmbedder struct{}

func (e *errEmbedder) Dims() int { return 2 }

func (e *errEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, errors.New("embed error")
}

func (e *errEmbedder) ModelID() string { return "err-v1" }

// fakeEmbedder is a test-only embed.Embedder that returns a fixed 2D vector.
// fakeEmbedder is a test-only embed.Embedder that returns a fixed 2D vector.
type fakeEmbedder struct{}

func (f *fakeEmbedder) Dims() int { return 2 }

func (f *fakeEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return []float32{0.8, 0.6}, nil
}

func (f *fakeEmbedder) ModelID() string { return "fake-v1" }

// testEmbedder is a no-op embedder that counts invocations.
// testEmbedder is a no-op embedder that counts invocations.
type testEmbedder struct {
	callCount *int
}

func (te *testEmbedder) Dims() int {
	return 384
}

func (te *testEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	*te.callCount++
	return make([]float32, 384), nil // Return a dummy 384-dim vector
}

func (te *testEmbedder) ModelID() string {
	return "test-model"
}

// testNotFoundError is a stub os.ErrNotExist-compatible error for test fakes.
// testNotFoundError is a stub os.ErrNotExist-compatible error for test fakes.
type testNotFoundError struct {
	path string
}

func (e *testNotFoundError) Error() string { return "not found: " + e.path }

func (e *testNotFoundError) Is(target error) bool {
	return target.Error() == "file does not exist" || strings.Contains(target.Error(), "not exist")
}

// ── Task 4: definition-note read path + vocab_version home ───────────────────

// buildTask4DefinitionFixture builds the shared #678 Task 4 scratch vault: a
// family note (vocab_version "6.0"), two term-definition notes (retrieval-
// design, token-budget) with object: descriptions, and three member notes
// tagged into the vocab/<term> namespace (2 retrieval-design, 1 token-budget).
// buildTask4DefinitionFixture builds the shared #678 Task 4 scratch vault: a
// family note (vocab_version "6.0"), two term-definition notes (retrieval-
// design, token-budget) with object: descriptions, and three member notes
// tagged into the vocab/<term> namespace (2 retrieval-design, 1 token-budget).
func buildTask4DefinitionFixture(t *testing.T) string {
	t.Helper()

	vault := t.TempDir()

	writeNote(t, vault, "210.2026-07-10.vocab-definition.md",
		"---\ntype: fact\nvocab_version: \"6.0\"\ntags:\n    - vocab\n---\n\nVocab family root.\n")
	writeNote(t, vault, "211.2026-07-10.vocab-retrieval-design-definition.md",
		"---\ntype: fact\nobject: keeps queries scoped to task-relevant vault slices\n"+
			"tags:\n    - vocab\n---\n\nDefines retrieval-design.\n")
	writeNote(t, vault, "212.2026-07-10.vocab-token-budget-definition.md",
		"---\ntype: fact\nobject: tracks payload size against context limits\n"+
			"tags:\n    - vocab\n---\n\nDefines token-budget.\n")
	writeNote(t, vault, "220.2026-07-10.member-a.md",
		"---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nMember A.\n")
	writeNote(t, vault, "221.2026-07-10.member-b.md",
		"---\ntype: fact\ntags:\n    - vocab/retrieval-design\n---\n\nMember B.\n")
	writeNote(t, vault, "222.2026-07-10.member-c.md",
		"---\ntype: fact\ntags:\n    - vocab/token-budget\n---\n\nMember C.\n")

	return vault
}

// centroidTwoPassDeps wires VocabDeps over the fixture files, capturing writes.
// Reads prefer `written` so pass 2 sees pass-1 output when both write.
// centroidTwoPassDeps wires VocabDeps over the fixture files, capturing writes.
// Reads prefer `written` so pass 2 sees pass-1 output when both write.
func centroidTwoPassDeps(files, written map[string][]byte) cli.VocabDeps {
	return cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{
				"1aa.note-a.md", "1ab.note-b.md",
				"210.2026-01-01.vocab-eval-topic-definition.md",
				"211.2026-01-01.vocab-orphan-topic-definition.md",
			}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := written[path]; ok {
				return data, nil
			}

			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		DeleteFile:   func(string) error { return nil },
		WriteSidecar: func(string, []byte) error { return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}
}

// ── Centroid two-pass assignment ──────────────────────────────────────────────

// centroidTwoPassFiles builds the shared fixture for the two-pass tests:
// term eval-topic (desc vector [1,0]), term orphan-topic (desc vector [0,-1],
// no members at floor 0.30), note A [0.8,0.6] (pass-1 member of eval-topic),
// note B [0,1] (below floor vs desc; cos 0.6 vs the A-only centroid → pass-2 member).
// Both terms are pre-seeded as bare-vocab-tagged definition notes so
// bootstrap/refit's idempotent skip leaves them untouched — the fixture
// exercises the centroid math, not minting.
// centroidTwoPassFiles builds the shared fixture for the two-pass tests:
// term eval-topic (desc vector [1,0]), term orphan-topic (desc vector [0,-1],
// no members at floor 0.30), note A [0.8,0.6] (pass-1 member of eval-topic),
// note B [0,1] (below floor vs desc; cos 0.6 vs the A-only centroid → pass-2 member).
// Both terms are pre-seeded as bare-vocab-tagged definition notes so
// bootstrap/refit's idempotent skip leaves them untouched — the fixture
// exercises the centroid math, not minting.
func centroidTwoPassFiles(g Gomega) map[string][]byte {
	zeroVec := make([]float32, 2)

	marshalTermSidecar := func(vec []float32) []byte {
		return embed.MarshalSidecar(embed.Sidecar{
			SchemaVersion:    1,
			EmbeddingModelID: "test",
			Dims:             2,
			BodyVector:       vec,
			SituationVector:  zeroVec,
		})
	}

	noteA := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\nLesson learned: when ctx, a.\n"
	noteB := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1ab\"\ncreated: 2026-01-01\nsource: test\n---\n\nLesson learned: when ctx, a.\n"
	definitionNote := func(id, term string) string {
		return "---\ntype: fact\ntier: L2\nsituation: s\nsubject: the " + term + " vocab term\n" +
			"predicate: covers\nobject: desc\nluhmann: \"" + id + "\"\ncreated: \"2026-01-01\"\nsource: test\n" +
			"tags:\n    - vocab\n---\n\nInformation learned: desc\n"
	}

	seed := []cli.SeedTerm{
		{Term: "eval-topic", Description: "desc"},
		{Term: "orphan-topic", Description: "desc"},
	}
	seedYAML, yamlErr := yaml.Marshal(seed)
	g.Expect(yamlErr).NotTo(HaveOccurred())

	return map[string][]byte{
		"/seed.yaml": seedYAML,
		"/plan.yaml": []byte("removals: []\n"),
		"/vault/210.2026-01-01.vocab-eval-topic-definition.md":         []byte(definitionNote("210", "eval-topic")),
		"/vault/211.2026-01-01.vocab-orphan-topic-definition.md":       []byte(definitionNote("211", "orphan-topic")),
		"/vault/210.2026-01-01.vocab-eval-topic-definition.vec.json":   marshalTermSidecar([]float32{1, 0}),
		"/vault/211.2026-01-01.vocab-orphan-topic-definition.vec.json": marshalTermSidecar([]float32{0, -1}),
		"/vault/1aa.note-a.md":       []byte(noteA),
		"/vault/1ab.note-b.md":       []byte(noteB),
		"/vault/1aa.note-a.vec.json": marshalTermSidecar([]float32{0.8, 0.6}),
		"/vault/1ab.note-b.vec.json": marshalTermSidecar([]float32{0, 1}),
	}
}

// makeUnitVec builds a unit-ish 2D vector with component 0 set to value
// and component 1 inferred as sqrt(1 - value²) for a proper unit vector.
// Used in tests to produce cosine similarities equal to value (cosine with [1,0]).
// makeUnitVec builds a unit-ish 2D vector with component 0 set to value
// and component 1 inferred as sqrt(1 - value²) for a proper unit vector.
// Used in tests to produce cosine similarities equal to value (cosine with [1,0]).
func makeUnitVec(value float32) []float32 {
	const (
		dims      = 2
		component = 0
	)

	vec := make([]float32, dims)
	vec[component] = value

	otherSq := float32(1.0) - value*value
	if otherSq < 0 {
		otherSq = 0
	}

	other := float32(1)
	if dims > 1 {
		// sqrt approximation — good enough for test vectors
		other = sqrtFloat32(otherSq)
	}

	for idx := range vec {
		if idx != component {
			vec[idx] = other / float32(dims-1)
		}
	}

	return vec
}

// ── helpers ───────────────────────────────────────────────────────────────────

// mustMarshalSidecarWithBodyVector builds a marshaled sidecar carrying the
// given body vector (SituationVector dims-matched so UnmarshalSidecar accepts
// it), for tests that only care about the body vector.
// mustMarshalSidecarWithBodyVector builds a marshaled sidecar carrying the
// given body vector (SituationVector dims-matched so UnmarshalSidecar accepts
// it), for tests that only care about the body vector.
func mustMarshalSidecarWithBodyVector(t *testing.T, vec []float32) []byte {
	t.Helper()

	return embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "test",
		Dims:             len(vec),
		SituationVector:  vec,
		BodyVector:       vec,
	})
}

// sqrtFloat32 returns an approximate square root of squaredValue (Newton's method, 5 iterations).
// sqrtFloat32 returns an approximate square root of squaredValue (Newton's method, 5 iterations).
func sqrtFloat32(squaredValue float32) float32 {
	if squaredValue <= 0 {
		return 0
	}

	guess := squaredValue / 2

	for range 5 {
		guess = (guess + squaredValue/guess) / 2
	}

	return guess
}

// writeNote writes a note's raw content to vault/basename on the real
// filesystem, for tests exercising real OS-backed VocabDeps against a
// t.TempDir() vault.
// writeNote writes a note's raw content to vault/basename on the real
// filesystem, for tests exercising real OS-backed VocabDeps against a
// t.TempDir() vault.
func writeNote(t *testing.T, vault, basename, content string) {
	t.Helper()

	g := NewWithT(t)
	g.Expect(os.WriteFile(filepath.Join(vault, basename), []byte(content), 0o600)).To(Succeed())
}

// writeNoteAndSidecar writes a note plus its embedding sidecar (body vector
// only) to vault/basename, for real-FS centroid/member-scan tests.
// writeNoteAndSidecar writes a note plus its embedding sidecar (body vector
// only) to vault/basename, for real-FS centroid/member-scan tests.
func writeNoteAndSidecar(t *testing.T, vault, basename, content string, vec []float32) {
	t.Helper()

	writeNote(t, vault, basename, content)

	g := NewWithT(t)
	sidecarPath := embed.SidecarPath(filepath.Join(vault, basename))
	g.Expect(os.WriteFile(sidecarPath, mustMarshalSidecarWithBodyVector(t, vec), 0o600)).To(Succeed())
}
