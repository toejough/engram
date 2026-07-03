package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	// vocab.index.md exercises the skip branch; bad1 has no sidecar (read
	// error) and bad2 a malformed sidecar — both are skipped by the model-id
	// walk before it reaches x's readable sidecar.
	listMD := func(string) ([]string, error) {
		return []string{"vocab.index.md", "vocab.bad1.md", "vocab.bad2.md", "vocab.x.md", "vocab.y.md"}, nil
	}

	centroids := []byte(`{"schema_version":1,"embedding_model_id":"test","dims":2,` +
		`"terms":{"x":{"vector":[0.5,0.5],"member_count":3}}}`)

	files := map[string][]byte{
		"/vault/vocab.bad2.vec.json":  []byte("{not json"),
		"/vault/vocab.x.vec.json":     sidecar([]float32{1, 0}),
		"/vault/vocab.y.vec.json":     sidecar([]float32{0, 1}),
		"/vault/vocab.centroids.json": centroids,
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

// ── Coverage: newOsVocabDeps closures ────────────────────────────────────────

// TestNewOsVocabDeps_ClosuresCalled verifies that the ListMD, WriteFile, and
// DeleteFile closures inside newOsVocabDeps are wired correctly. Covers the
// function body and its closure blocks.
func TestNewOsVocabDeps_ClosuresCalled(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	deps := cli.ExportNewOsVocabDeps()

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
func TestNoteContainsAnyRemoval_NoMatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ExportNoteContainsAnyRemoval("some content about eval", []string{"nonexistent-term"})).
		To(BeFalse(), "must return false when no removal term is in content")
}

// ── Coverage: assigner error paths ───────────────────────────────────────────

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
// an amend, if terms are present, the note gets vocab channels written.
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

	g.Expect(string(written)).To(ContainSubstring("vocab:"),
		"vocab: frontmatter must be present after amend when terms are assigned")
	g.Expect(string(written)).To(ContainSubstring("Vocab:"),
		"Vocab: body line must be present after amend when terms are assigned")
}

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
// WriteNote is called with both the vocab: frontmatter key and Vocab: body line.
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
	g.Expect(string(updatedContent)).To(ContainSubstring("vocab:"),
		"vocab: frontmatter key must be present in updated note")
	g.Expect(string(updatedContent)).To(ContainSubstring("Vocab:"),
		"Vocab: body line must be present in updated note")
}

// ── Coverage: assignVocabToNote sidecar-error and note-read-error paths ───────

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

	// 1aa.note-a has no sidecar → sidecarErr in assignVocabToNote.
	// 1ab.note-b has sidecar but no note → readErr in assignVocabToNote.
	files := map[string][]byte{
		"/seed.yaml":                       seedYAML,
		"/vault/vocab.eval-topic.vec.json": embed.MarshalSidecar(termSidecar),
		"/vault/1ab.note-b.vec.json":       embed.MarshalSidecar(noteBSidecar),
	}

	var memberWriteCount int

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{"1aa.note-a.md", "1ab.note-b.md", "vocab.eval-topic.md"}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, _ []byte) error {
			if !strings.HasPrefix(filepath.Base(path), "vocab.") {
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
// member note (with a valid sidecar) gets both vocab channels written.
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

	files := map[string][]byte{
		"/seed.yaml":                             seedYAML,
		"/vault/1aa.2026-01-01.test.md":          []byte(existingNote),
		"/vault/1aa.2026-01-01.test.vec.json":    embed.MarshalSidecar(noteSidecar),
		"/vault/vocab.eval-methodology.vec.json": embed.MarshalSidecar(termSidecar),
	}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		// ListMD returns both the member note AND the vocab term note so that
		// loadTermVectors can discover the term's pre-populated sidecar.
		ListMD: func(_ string) ([]string, error) {
			return []string{"1aa.2026-01-01.test.md", "vocab.eval-methodology.md"}, nil
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
	g.Expect(updatedContent).To(ContainSubstring("vocab:"), "vocab: frontmatter key must be written")
	g.Expect(updatedContent).To(ContainSubstring("Vocab:"), "Vocab: body line must be written")
	g.Expect(updatedContent).To(ContainSubstring("eval-methodology"), "term must be assigned")
}

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

// TestRunVocabBootstrap_CreatesTermNote verifies that bootstrap writes a term
// note file with the correct type/term/description frontmatter.
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

	termPath := "/vault/vocab.eval-methodology.md"
	g.Expect(written).To(HaveKey(termPath), "term note must be written")

	content := string(written[termPath])
	g.Expect(content).To(ContainSubstring("type: vocab"), "term note must have type: vocab")
	g.Expect(content).To(ContainSubstring("term: eval-methodology"), "term note must carry term name")
	g.Expect(content).To(ContainSubstring("how we evaluate"), "description must appear in body")
}

// ── Coverage: embedTermNote embed/write error paths ───────────────────────────

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

	content := string(written["/vault/vocab.eval-methodology.md"])
	g.Expect(content).To(ContainSubstring("Exemplars:"), "body must carry an exemplar section")
	g.Expect(content).To(ContainSubstring("- designing an eval harness for memory-vs-baseline comparison"),
		"each exemplar must appear as a body list line")
	g.Expect(content).To(ContainSubstring("- validating a cheap model tier against a gold standard"),
		"all exemplars must be rendered")
}

// TestRunVocabBootstrap_GeneratesIndex verifies that bootstrap writes
// vocab.index.md with type: vocab-index.
func TestRunVocabBootstrap_GeneratesIndex(t *testing.T) {
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

	indexPath := "/vault/vocab.index.md"
	g.Expect(written).To(HaveKey(indexPath), "vocab.index.md must be generated")
	indexContent := string(written[indexPath])
	g.Expect(indexContent).To(ContainSubstring("type: vocab-index"), "index must have type: vocab-index")
	g.Expect(indexContent).To(ContainSubstring("[[vocab.eval-methodology]]"), "index must link to term note")
}

// TestRunVocabBootstrap_Idempotent verifies that re-running bootstrap with the
// same seed overwrites the existing term note without erroring on ErrExist.
func TestRunVocabBootstrap_Idempotent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	seed := []cli.SeedTerm{{Term: "eval-methodology", Description: "updated description"}}
	seedYAML, err := yaml.Marshal(seed)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	existingTermNote := "---\ntype: vocab\nterm: eval-methodology\n" +
		"description: old description\nvocab_version: 1.0\n---\n\nold description\n"
	files := map[string][]byte{
		"/seed.yaml":                       seedYAML,
		"/vault/vocab.eval-methodology.md": []byte(existingTermNote),
	}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock:         func(string) (func(), error) { return func() {}, nil },
		ListMD:       func(string) ([]string, error) { return []string{"vocab.eval-methodology.md"}, nil },
		ReadFile:     func(path string) ([]byte, error) { return files[path], nil },
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabBootstrapArgs{Vault: "/vault", SeedFile: "/seed.yaml"}

	var buf strings.Builder

	bootErr := cli.RunVocabBootstrap(t.Context(), args, deps, &buf)
	g.Expect(bootErr).NotTo(HaveOccurred(), "re-running bootstrap must not error")

	if bootErr != nil {
		return
	}

	termPath := "/vault/vocab.eval-methodology.md"
	g.Expect(written).To(HaveKey(termPath), "term note must be rewritten on idempotent run")
	g.Expect(string(written[termPath])).To(ContainSubstring("updated description"),
		"term note must carry new description")
}

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
			if strings.Contains(path, "vocab.eval-topic") {
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
// minor component of the current vocab_version in vocab.index.md.
func TestRunVocabPropose_BumpsMinorVersion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existingIndex := "---\ntype: vocab-index\nvocab_version: 1.3\ncreated: 2026-07-02\n---\n\n"
	files := map[string][]byte{
		"/vault/vocab.index.md": []byte(existingIndex),
	}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock:         func(string) (func(), error) { return func() {}, nil },
		ListMD:       func(string) ([]string, error) { return []string{"vocab.index.md"}, nil },
		ReadFile:     func(path string) ([]byte, error) { return files[path], nil },
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabProposeArgs{Vault: "/vault", Term: "new-term", Description: "desc"}

	var buf strings.Builder

	g.Expect(cli.RunVocabPropose(t.Context(), args, deps, &buf)).To(Succeed())

	indexContent := string(written["/vault/vocab.index.md"])
	g.Expect(indexContent).To(ContainSubstring("1.4"), "minor version must be bumped from 1.3 → 1.4")
}

// ── Vocab commands: propose ───────────────────────────────────────────────────

// TestRunVocabPropose_CreatesTermNote verifies that propose creates a new term
// note with the supplied name and description.
func TestRunVocabPropose_CreatesTermNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existingIndex := "---\ntype: vocab-index\nvocab_version: 1.0\ncreated: 2026-07-02\n---\n\n"
	files := map[string][]byte{
		"/vault/vocab.index.md": []byte(existingIndex),
	}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock:         func(string) (func(), error) { return func() {}, nil },
		ListMD:       func(string) ([]string, error) { return []string{"vocab.index.md"}, nil },
		ReadFile:     func(path string) ([]byte, error) { return files[path], nil },
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

	termPath := "/vault/vocab.new-insight.md"
	g.Expect(written).To(HaveKey(termPath), "propose must create term note")
	g.Expect(string(written[termPath])).To(ContainSubstring("new-insight"), "term note must carry term name")
	g.Expect(string(written[termPath])).To(ContainSubstring("tracking novel insights"),
		"term note must carry description")
}

// TestRunVocabRefit_AppliesRemovals verifies that refit deletes the removed
// term note and clears the vocab: key from member notes.
func TestRunVocabRefit_AppliesRemovals(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	termNote := "---\ntype: vocab\nterm: orphan-term\ndescription: desc\n" +
		"vocab_version: 1.0\ncreated: 2026-07-02\n---\n\ndesc\n"
	memberNote := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\nvocab: [orphan-term]\n---\n\n" +
		"Lesson learned: when ctx, a.\n\nVocab: [[vocab.orphan-term]]\n"
	indexNote := "---\ntype: vocab-index\nvocab_version: 1.0\ncreated: 2026-07-02\n---\n\n" +
		"[[vocab.orphan-term]] — desc — 1 members\n"

	planContent := "removals:\n  - orphan-term\n"

	files := map[string][]byte{
		"/vault/vocab.orphan-term.md": []byte(termNote),
		"/vault/1aa.2026-01-01.md":    []byte(memberNote),
		"/vault/vocab.index.md":       []byte(indexNote),
		"/plan.yaml":                  []byte(planContent),
	}

	written := map[string][]byte{}
	deleted := map[string]bool{}

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{"vocab.orphan-term.md", "1aa.2026-01-01.md", "vocab.index.md"}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		DeleteFile:   func(path string) error { deleted[path] = true; return nil },
		WriteSidecar: func(string, []byte) error { return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{Vault: "/vault", PlanFile: "/plan.yaml"}

	var buf strings.Builder

	refitErr := cli.RunVocabRefit(t.Context(), args, deps, &buf)
	g.Expect(refitErr).NotTo(HaveOccurred())

	if refitErr != nil {
		return
	}

	// Term note must be deleted.
	g.Expect(deleted).To(HaveKey("/vault/vocab.orphan-term.md"), "removed term note must be deleted")

	// Member note must have vocab: key cleared.
	updatedMember := string(written["/vault/1aa.2026-01-01.md"])
	g.Expect(updatedMember).NotTo(ContainSubstring("orphan-term"), "removed term must be cleared from member")
}

// TestRunVocabRefit_AppliesRenames verifies that refit renames a term note
// and rewrites member notes' vocab: frontmatter and Vocab: body line.
func TestRunVocabRefit_AppliesRenames(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	oldTermNote := "---\ntype: vocab\nterm: old-term\ndescription: old\n" +
		"vocab_version: 1.0\ncreated: 2026-07-02\n---\n\nold\n"
	memberNote := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\nvocab: [old-term]\n---\n\n" +
		"Lesson learned: when ctx, a.\n\nVocab: [[vocab.old-term]]\n"
	indexNote := "---\ntype: vocab-index\nvocab_version: 1.0\ncreated: 2026-07-02\n---\n\n" +
		"[[vocab.old-term]] — old — 1 members\n"

	planContent := "renames:\n  - from: old-term\n    to: new-term\n"

	files := map[string][]byte{
		"/vault/vocab.old-term.md": []byte(oldTermNote),
		"/vault/1aa.2026-01-01.md": []byte(memberNote),
		"/vault/vocab.index.md":    []byte(indexNote),
		"/plan.yaml":               []byte(planContent),
	}

	written := map[string][]byte{}
	deleted := map[string]bool{}

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{"vocab.old-term.md", "1aa.2026-01-01.md", "vocab.index.md"}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		DeleteFile:   func(path string) error { deleted[path] = true; return nil },
		WriteSidecar: func(path string, data []byte) error { written[path] = data; return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{Vault: "/vault", PlanFile: "/plan.yaml"}

	var buf strings.Builder

	refitErr := cli.RunVocabRefit(t.Context(), args, deps, &buf)
	g.Expect(refitErr).NotTo(HaveOccurred())

	if refitErr != nil {
		return
	}

	// Old term note must be deleted; new term note must be created.
	g.Expect(deleted).To(HaveKey("/vault/vocab.old-term.md"), "old term note must be deleted")
	g.Expect(written).To(HaveKey("/vault/vocab.new-term.md"), "new term note must be created")

	// Member note must have old-term replaced with new-term.
	updatedMember := string(written["/vault/1aa.2026-01-01.md"])
	g.Expect(updatedMember).NotTo(ContainSubstring("old-term"), "old term must be removed from member")
	g.Expect(updatedMember).To(ContainSubstring("new-term"), "new term must be written to member")
}

// TestRunVocabRefit_CentroidTwoPass_RetagsAgainstCentroids verifies the refit
// re-tag pass runs the same centroid two-pass and refreshes vocab.centroids.json.
func TestRunVocabRefit_CentroidTwoPass_RetagsAgainstCentroids(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files := centroidTwoPassFiles(g)
	written := map[string][]byte{}
	deps := centroidTwoPassDeps(files, written)

	args := cli.VocabRefitArgs{Vault: "/vault", PlanFile: "/plan.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), args, deps, &buf)).To(Succeed())

	noteB := string(written["/vault/1ab.note-b.md"])
	g.Expect(noteB).To(ContainSubstring("eval-topic"),
		"refit re-tag must assign note B against the member centroid")
	g.Expect(written).To(HaveKey("/vault/vocab.centroids.json"),
		"refit must refresh the derived centroids file")
}

// ── Coverage: clearRemovalsFromNoteContent unmarshal-error path ───────────────

// TestRunVocabRefit_ClearRemovals_BadYAML_ReturnsRaw verifies that when a member
// note has syntactically invalid YAML frontmatter, clearRemovalsFromNoteContent
// returns the raw content unchanged (covers unmarshalErr branch).
func TestRunVocabRefit_ClearRemovals_BadYAML_ReturnsRaw(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Note has valid "---" delimiters but unparseable YAML body.
	// The raw content also contains the removal term name so noteContainsAnyRemoval
	// fires and clearRemovalsFromNoteContent is called.
	badYAMLNote := "---\n{unclosed brace: orphan-term\n---\n" +
		"body text containing orphan-term here\n"
	planYAML := "removals:\n  - orphan-term\n"

	files := map[string][]byte{
		"/plan.yaml":    []byte(planYAML),
		"/vault/1aa.md": []byte(badYAMLNote),
	}

	var memberWriteCount int

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return []string{"1aa.md"}, nil },
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, _ []byte) error {
			if !strings.HasPrefix(filepath.Base(path), "vocab.") {
				memberWriteCount++
			}

			return nil
		},
		DeleteFile:   func(string) error { return nil },
		WriteSidecar: func(_ string, _ []byte) error { return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{Vault: "/vault", PlanFile: "/plan.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), args, deps, &buf)).To(Succeed())
	g.Expect(memberWriteCount).To(Equal(0),
		"note with bad YAML frontmatter must not be rewritten (content unchanged)")
}

// ── Coverage: clearRemovedTermsFromMembers write-error path ───────────────────

// TestRunVocabRefit_ClearRemovals_WriteError_LogsWarning verifies that when
// WriteFile fails for a member note during term removal, a warning is logged
// but refit still succeeds (covers clearRemovedTermsFromMembers write-error path).
func TestRunVocabRefit_ClearRemovals_WriteError_LogsWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	memberNote := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\nvocab: [orphan-term]\n---\n\n" +
		"Lesson learned: when ctx, a.\n\nVocab: [[vocab.orphan-term]]\n"
	planYAML := "removals:\n  - orphan-term\n"

	files := map[string][]byte{
		"/plan.yaml":    []byte(planYAML),
		"/vault/1aa.md": []byte(memberNote),
	}

	var warned bool

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return []string{"1aa.md"}, nil },
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, _ []byte) error {
			if !strings.HasPrefix(filepath.Base(path), "vocab.") {
				return errors.New("write failed")
			}

			return nil
		},
		DeleteFile:   func(string) error { return nil },
		WriteSidecar: func(_ string, _ []byte) error { return nil },
		LogWarning:   func(string, ...any) { warned = true },
		Now:          func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{Vault: "/vault", PlanFile: "/plan.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), args, deps, &buf)).To(Succeed(),
		"refit must succeed even when member note write fails")
	g.Expect(warned).To(BeTrue(), "write error must trigger log warning")
}

// ── Coverage: collectNoteStats read-error path ────────────────────────────────

// TestRunVocabRefit_EmitRequest_NoteReadError_CountsNote verifies that when
// ReadFile fails for a non-vocab note in collectNoteStats, totalNotes is still
// incremented but untaggedCount is not (covers the readErr → continue branch).
func TestRunVocabRefit_EmitRequest_NoteReadError_CountsNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.VocabDeps{
		ListMD: func(string) ([]string, error) {
			return []string{"1aa.note.md"}, nil // non-vocab note that cannot be read
		},
		ReadFile: func(path string) ([]byte, error) {
			return nil, &testNotFoundError{path: path}
		},
	}

	args := cli.VocabRefitArgs{Vault: "/vault", EmitRequest: true}

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), args, deps, &buf)).To(Succeed())

	var payload map[string]any

	g.Expect(json.Unmarshal([]byte(buf.String()), &payload)).To(Succeed(),
		"emit-request output must be valid JSON")

	stats, ok := payload["stats"].(map[string]any)
	g.Expect(ok).To(BeTrue(), "stats block must be present in payload")
	g.Expect(stats["totalNotes"]).To(BeEquivalentTo(1),
		"unreadable note must still be counted in totalNotes")
	g.Expect(stats["untaggedCount"]).To(BeEquivalentTo(0),
		"unreadable note must not increment untaggedCount")
}

// ── Vocab commands: refit ─────────────────────────────────────────────────────

// TestRunVocabRefit_EmitRequest_PrintsPayload verifies that --emit-request
// prints a JSON payload containing current_terms.
func TestRunVocabRefit_EmitRequest_PrintsPayload(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	evalTermNote := "---\ntype: vocab\nterm: eval-methodology\ndescription: how we evaluate\n" +
		"vocab_version: 1.0\ncreated: 2026-07-02\n---\n\nhow we evaluate\n"
	indexNote := "---\ntype: vocab-index\nvocab_version: 1.0\ncreated: 2026-07-02\n---\n\n" +
		"[[vocab.eval-methodology]] — how we evaluate — 3 members\n"
	regularNote := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\nvocab: [eval-methodology]\n---\n\n" +
		"Lesson learned: when ctx, a.\n\n"

	files := map[string]string{
		"/vault/vocab.eval-methodology.md": evalTermNote,
		"/vault/vocab.index.md":            indexNote,
		"/vault/1aa.2026-01-01.note.md":    regularNote,
	}

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{
				"vocab.eval-methodology.md",
				"vocab.index.md",
				"1aa.2026-01-01.note.md",
			}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			for fullPath, content := range files {
				if filepath.Base(fullPath) == filepath.Base(path) {
					return []byte(content), nil
				}
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:    func(string, []byte) error { return nil },
		WriteSidecar: func(string, []byte) error { return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{Vault: "/vault", EmitRequest: true}

	var buf strings.Builder

	refitErr := cli.RunVocabRefit(t.Context(), args, deps, &buf)
	g.Expect(refitErr).NotTo(HaveOccurred())

	if refitErr != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("current_terms"), "emit-request must include current_terms")

	// Verify it's valid JSON.
	var payload map[string]any

	g.Expect(json.Unmarshal([]byte(output), &payload)).To(Succeed(),
		"emit-request output must be valid JSON")
}

// TestRunVocabRefit_MajorVersionBump verifies that refit increments the major
// version in vocab.index.md.
func TestRunVocabRefit_MajorVersionBump(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	indexNote := "---\ntype: vocab-index\nvocab_version: 1.4\ncreated: 2026-07-02\n---\n\n"
	planContent := "new_terms:\n  - term: extra-term\n    description: extra\n"

	files := map[string][]byte{
		"/vault/vocab.index.md": []byte(indexNote),
		"/plan.yaml":            []byte(planContent),
	}
	written := map[string][]byte{}

	deps := cli.VocabDeps{
		Lock:         func(string) (func(), error) { return func() {}, nil },
		ListMD:       func(string) ([]string, error) { return []string{"vocab.index.md"}, nil },
		ReadFile:     func(path string) ([]byte, error) { return files[path], nil },
		WriteFile:    func(path string, data []byte) error { written[path] = data; return nil },
		DeleteFile:   func(string) error { return nil },
		WriteSidecar: func(string, []byte) error { return nil },
		LogWarning:   func(string, ...any) {},
		Now:          func() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{Vault: "/vault", PlanFile: "/plan.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), args, deps, &buf)).To(Succeed())

	indexContent := string(written["/vault/vocab.index.md"])
	g.Expect(indexContent).To(ContainSubstring("2.0"), "major version must be bumped from 1.4 → 2.0")
}

// ── Coverage: applyRefitNewTerms warning path ─────────────────────────────────

// TestRunVocabRefit_NewTermWriteError_LogsWarning verifies that when WriteFile
// fails for a new term note, applyRefitNewTerms logs a warning and continues
// (covers applyRefitNewTerms warning path).
func TestRunVocabRefit_NewTermWriteError_LogsWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	planYAML := "new_terms:\n  - term: new-insight\n    description: a new insight\n"

	files := map[string][]byte{
		"/plan.yaml": []byte(planYAML),
	}

	var warned bool

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return nil, nil },
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, _ []byte) error {
			if strings.Contains(path, "vocab.new-insight") {
				return errors.New("write failed")
			}

			return nil
		},
		WriteSidecar: func(_ string, _ []byte) error { return nil },
		LogWarning:   func(string, ...any) { warned = true },
		Now:          func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{Vault: "/vault", PlanFile: "/plan.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), args, deps, &buf)).To(Succeed(),
		"refit must succeed even when new term write fails")
	g.Expect(warned).To(BeTrue(), "term write failure must trigger log warning")
}

// TestRunVocabRefit_NilDeleteFile verifies that when VocabDeps.DeleteFile is nil,
// applyRefitRemovals silently skips deletion (backward compat path).
func TestRunVocabRefit_NilDeleteFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	planYAML := `removals: [eval-methodology]`

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return nil, nil },
		ReadFile: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
				return []byte(planYAML), nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:  func(string, []byte) error { return nil },
		DeleteFile: nil, // intentionally nil — triggers the nil-delete skip path
		Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{
		Vault:    "/vault",
		PlanFile: "/plan.yaml",
	}

	var stdout strings.Builder

	err := cli.RunVocabRefit(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred(),
		"RunVocabRefit with nil DeleteFile must succeed (skip deletions)")
}

// ── Coverage: loadRefitPlan, applyRefitRemovals ───────────────────────────────

// TestRunVocabRefit_ReadPlanError verifies that RunVocabRefit returns an error
// when the plan file cannot be read (covers loadRefitPlan read error path).
func TestRunVocabRefit_ReadPlanError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return nil, nil },
		ReadFile: func(path string) ([]byte, error) {
			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(string, []byte) error { return nil },
		Now:       func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{
		Vault:    "/vault",
		PlanFile: "/nonexistent/plan.yaml",
	}

	var stdout strings.Builder

	err := cli.RunVocabRefit(t.Context(), args, deps, &stdout)
	g.Expect(err).To(HaveOccurred(), "RunVocabRefit must fail when plan file is missing")
}

// TestRunVocabRefit_Rename_DeleteError_LogsWarning verifies that when DeleteFile
// fails for the old term note during a rename, applyRefitRenames logs a warning
// and continues. Also covers loadTermDescription's unmarshal-error path (the
// old term note has invalid YAML frontmatter so loadTermDescription returns "").
func TestRunVocabRefit_Rename_DeleteError_LogsWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Old term note has valid frontmatter delimiters but invalid YAML body, so
	// loadTermDescription returns "" (covers its unmarshalErr path).
	badOldTermNote := "---\n{invalid yaml\n---\n\nold-term description\n"
	planYAML := "renames:\n  - from: old-term\n    to: new-term\n"

	files := map[string][]byte{
		"/plan.yaml":               []byte(planYAML),
		"/vault/vocab.old-term.md": []byte(badOldTermNote),
	}

	var warned bool

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return nil, nil },
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile:    func(_ string, _ []byte) error { return nil },
		DeleteFile:   func(_ string) error { return errors.New("delete failed") },
		WriteSidecar: func(_ string, _ []byte) error { return nil },
		LogWarning:   func(string, ...any) { warned = true },
		Now:          func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{Vault: "/vault", PlanFile: "/plan.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), args, deps, &buf)).To(Succeed(),
		"refit must succeed even when delete fails")
	g.Expect(warned).To(BeTrue(), "delete failure must trigger log warning")
}

// TestRunVocabRefit_Rename_MemberWriteError_LogsWarning verifies that when
// WriteFile fails for a member note during rewriteMemberTermRename, a warning
// is logged but refit succeeds (covers rewriteMemberTermRename write-error path).
func TestRunVocabRefit_Rename_MemberWriteError_LogsWarning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	memberNote := "---\ntype: feedback\nvocab: [old-term]\n---\n\nVocab: [[vocab.old-term]]\n"
	planYAML := "renames:\n  - from: old-term\n    to: new-term\n"

	files := map[string][]byte{
		"/plan.yaml":    []byte(planYAML),
		"/vault/1aa.md": []byte(memberNote),
	}

	var warned bool

	deps := cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{"1aa.md"}, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return data, nil
			}

			return nil, &testNotFoundError{path: path}
		},
		WriteFile: func(path string, _ []byte) error {
			if filepath.Base(path) == "1aa.md" {
				return errors.New("write failed")
			}

			return nil
		},
		DeleteFile:   func(string) error { return nil },
		WriteSidecar: func(_ string, _ []byte) error { return nil },
		LogWarning:   func(string, ...any) { warned = true },
		Now:          func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := cli.VocabRefitArgs{Vault: "/vault", PlanFile: "/plan.yaml"}

	var buf strings.Builder

	g.Expect(cli.RunVocabRefit(t.Context(), args, deps, &buf)).To(Succeed(),
		"refit must succeed even when member write fails during rename")
	g.Expect(warned).To(BeTrue(), "member write failure must trigger log warning")
}

// ── Vocab commands: stats ─────────────────────────────────────────────────────

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

// TestRunVocabStats_ReportsHubAndOrphan verifies hub (>25% of vault) and
// orphan (<2 members) detection in the stats output.
func TestRunVocabStats_ReportsHubAndOrphan(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// 4 total notes: 3 tagged eval-methodology (hub, 75%), 0 tagged scope-discipline (orphan).
	termIndexContent := "---\ntype: vocab-index\nvocab_version: 1.0\ncreated: 2026-07-02\n---\n\n" +
		"[[vocab.eval-methodology]] — eval — 3 members\n" +
		"[[vocab.scope-discipline]] — scope — 0 members\n"

	evalTermNote := "---\ntype: vocab\nterm: eval-methodology\ndescription: eval\n" +
		"vocab_version: 1.0\ncreated: 2026-07-02\n---\n\neval\n"
	scopeTermNote := "---\ntype: vocab\nterm: scope-discipline\ndescription: scope\n" +
		"vocab_version: 1.0\ncreated: 2026-07-02\n---\n\nscope\n"
	// 3 notes tagged eval-methodology, 1 untagged.
	taggedNote := "---\ntype: feedback\nsituation: ctx\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\nvocab: [eval-methodology]\n---\n\n" +
		"Lesson learned: when ctx, a.\n\n"
	untaggedNote := "---\ntype: feedback\nsituation: ctx2\nbehavior: b\nimpact: i\naction: a\n" +
		"luhmann: \"2aa\"\ncreated: 2026-01-02\nsource: test\n---\n\nLesson learned: when ctx2, a.\n\n"

	allFiles := map[string]string{
		"/vault/vocab.index.md":            termIndexContent,
		"/vault/vocab.eval-methodology.md": evalTermNote,
		"/vault/vocab.scope-discipline.md": scopeTermNote,
		"/vault/1aa.2026-01-01.note.md":    taggedNote,
		"/vault/1bb.2026-01-01.note.md":    taggedNote,
		"/vault/1cc.2026-01-01.note.md":    taggedNote,
		"/vault/2aa.2026-01-02.note.md":    untaggedNote,
	}
	allNames := []string{
		"vocab.index.md",
		"vocab.eval-methodology.md",
		"vocab.scope-discipline.md",
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
	g.Expect(output).To(ContainSubstring("scope-discipline"), "orphan term name must appear")
}

// TestRunVocabStats_VocabTypeNoteExcluded verifies that a note with
// type: vocab is excluded from the member count (extractNoteVocabTags → false).
func TestRunVocabStats_VocabTypeNoteExcluded(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.VocabStatsDeps{
		ListMD: func(string) ([]string, error) {
			// Not a vocab.* filename (so isVocabTermFilename returns false), but
			// the frontmatter says type: vocab — extractNoteVocabTags must filter it.
			return []string{"1aa.2026-01-01.test.md"}, nil
		},
		ReadFile: func(_ string) ([]byte, error) {
			return []byte("---\ntype: vocab\nterm: foo\n---\nFoo is a term.\n"), nil
		},
	}

	args := cli.VocabStatsArgs{Vault: "/vault"}

	var stdout strings.Builder

	g.Expect(cli.RunVocabStats(args, deps, &stdout)).To(Succeed())
	g.Expect(stdout.String()).To(ContainSubstring("terms: 0"),
		"vocab-type note must not count as a term")
}

// ── Coverage: vocabTargets bootstrap closure ──────────────────────────────────

// TestTargets_VocabBootstrapNonExistentSeed exercises the vocab bootstrap
// closure end-to-end (covers newOsVocabDeps and the bootstrap closure in
// vocabTargets). The seed file does not exist, so the command returns an error.
func TestTargets_VocabBootstrapNonExistentSeed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(&stdout, &stderr, func(int) {}, nil)

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
func TestTargets_VocabProposeCreatesNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(&stdout, &stderr, func(int) {}, nil)

	_, targErr := targ.Execute(
		[]string{"engram", "vocab", "propose", "--vault", vault, "--term", "test-term", "--description", "a test"},
		targets...,
	)
	if targErr != nil {
		stderr.WriteString(targErr.Error())
	}

	g.Expect(stderr.String()).To(BeEmpty(), "vocab propose must succeed on an empty vault")
}

// TestTargets_VocabRefitMissingPlan exercises the vocab refit closure via
// Targets(). A missing plan file causes an error written to stderr.
func TestTargets_VocabRefitMissingPlan(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(&stdout, &stderr, func(int) {}, nil)

	_, targErr := targ.Execute(
		[]string{"engram", "vocab", "refit", "--vault", vault, "--plan", filepath.Join(vault, "missing.yaml")},
		targets...,
	)
	if targErr != nil {
		stderr.WriteString(targErr.Error())
	}

	g.Expect(stderr.String()).NotTo(BeEmpty(), "missing plan must produce an error on stderr")
}

// ── Vocab integration: OS wiring ─────────────────────────────────────────────

// TestTargets_VocabStatsEmpty exercises the vocab stats closure end-to-end
// through Targets() with an empty vault so newOsVocabStatsDeps wiring is covered.
func TestTargets_VocabStatsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	vault := t.TempDir()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(&stdout, &stderr, func(int) {}, nil)

	_, targErr := targ.Execute([]string{"engram", "vocab", "stats", "--vault", vault}, targets...)
	if targErr != nil {
		stderr.WriteString(targErr.Error())
	}

	g.Expect(stderr.String()).To(BeEmpty(), "vocab stats on empty vault must not produce errors")
	g.Expect(stdout.String()).To(ContainSubstring("terms: 0"), "empty vault must report zero terms")
}

// errEmbedder is a test-only embed.Embedder that always returns an error on Embed.
type errEmbedder struct{}

func (e *errEmbedder) Dims() int { return 2 }

func (e *errEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, errors.New("embed error")
}

func (e *errEmbedder) ModelID() string { return "err-v1" }

// fakeEmbedder is a test-only embed.Embedder that returns a fixed 2D vector.
type fakeEmbedder struct{}

func (f *fakeEmbedder) Dims() int { return 2 }

func (f *fakeEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return []float32{0.8, 0.6}, nil
}

func (f *fakeEmbedder) ModelID() string { return "fake-v1" }

// testNotFoundError is a stub os.ErrNotExist-compatible error for test fakes.
type testNotFoundError struct {
	path string
}

func (e *testNotFoundError) Error() string { return "not found: " + e.path }

func (e *testNotFoundError) Is(target error) bool {
	return target.Error() == "file does not exist" || strings.Contains(target.Error(), "not exist")
}

// centroidTwoPassDeps wires VocabDeps over the fixture files, capturing writes.
// Reads prefer `written` so pass 2 sees pass-1 output when both write.
func centroidTwoPassDeps(files, written map[string][]byte) cli.VocabDeps {
	return cli.VocabDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			return []string{
				"1aa.note-a.md", "1ab.note-b.md",
				"vocab.eval-topic.md", "vocab.orphan-topic.md",
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
	termNote := func(term string) string {
		return "---\ntype: vocab\nterm: " + term + "\ndescription: desc\n" +
			"vocab_version: \"1.0\"\ncreated: \"2026-01-01\"\n---\n\ndesc\n"
	}

	seed := []cli.SeedTerm{
		{Term: "eval-topic", Description: "desc"},
		{Term: "orphan-topic", Description: "desc"},
	}
	seedYAML, yamlErr := yaml.Marshal(seed)
	g.Expect(yamlErr).NotTo(HaveOccurred())

	return map[string][]byte{
		"/seed.yaml":                         seedYAML,
		"/plan.yaml":                         []byte("removals: []\n"),
		"/vault/vocab.eval-topic.md":         []byte(termNote("eval-topic")),
		"/vault/vocab.orphan-topic.md":       []byte(termNote("orphan-topic")),
		"/vault/vocab.eval-topic.vec.json":   marshalTermSidecar([]float32{1, 0}),
		"/vault/vocab.orphan-topic.vec.json": marshalTermSidecar([]float32{0, -1}),
		"/vault/1aa.note-a.md":               []byte(noteA),
		"/vault/1ab.note-b.md":               []byte(noteB),
		"/vault/1aa.note-a.vec.json":         marshalTermSidecar([]float32{0.8, 0.6}),
		"/vault/1ab.note-b.vec.json":         marshalTermSidecar([]float32{0, 1}),
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

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
