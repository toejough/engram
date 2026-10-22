package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// Test792b27c1a2_VaultHasOnlyTopLevelNotes_OneBranchedNoteIsEnough pins 792b27c1a2: a vault with a top-level note
// and a branched note (ID 1a) does not hold only top-level notes.
func Test792b27c1a2_VaultHasOnlyTopLevelNotes_OneBranchedNoteIsEnough(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	fileSystem := nrp1VaultFS(map[string][]byte{
		"/vault/1.2026-07-01.first-note.md":   []byte("note one"),
		"/vault/1a.2026-07-02.branch-note.md": []byte("branched note"),
	})

	g.Expect(cli.ExportVaultHasOnlyTopLevelNotes("/vault", fileSystem)).To(BeFalse())
}

// Test792b27c1a3_VaultHasOnlyTopLevelNotes_AnEmptyVaultIsNotFlat pins 792b27c1a3: a readable vault directory with
// no note file does not hold only top-level notes (a sidecar alone is not a note).
func Test792b27c1a3_VaultHasOnlyTopLevelNotes_AnEmptyVaultIsNotFlat(t *testing.T) {
	t.Parallel()

	t.Run("a directory with no file", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		g.Expect(cli.ExportVaultHasOnlyTopLevelNotes("/vault", nrp1VaultFS(nil))).To(BeFalse())
	})

	t.Run("a directory holding only a sidecar", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		fileSystem := nrp1VaultFS(map[string][]byte{"/vault/1.2026-07-01.first-note.vec.json": []byte("{}")})

		g.Expect(cli.ExportVaultHasOnlyTopLevelNotes("/vault", fileSystem)).To(BeFalse())
	})
}

// Test792b27c1a4_VaultHasOnlyTopLevelNotes_AnUnreadableVaultIsNotFlat pins 792b27c1a4: a vault directory that
// cannot be read does not hold only top-level notes, although the same notes in a readable vault would.
func Test792b27c1a4_VaultHasOnlyTopLevelNotes_AnUnreadableVaultIsNotFlat(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := map[string][]byte{
		"/vault/1.2026-07-01.first-note.md":  []byte("note one"),
		"/vault/2.2026-07-02.second-note.md": []byte("note two"),
	}

	g.Expect(cli.ExportVaultHasOnlyTopLevelNotes("/vault", nrp1VaultFS(notes))).To(BeTrue(),
		"control: the same notes in a readable vault are flat")
	g.Expect(cli.ExportVaultHasOnlyTopLevelNotes("/vault", nrp1UnreadableFS{u1FS: nrp1VaultFS(notes)})).To(BeFalse())
}

// Test792b27c1a5_VaultHasOnlyTopLevelNotes_AnIDThatDoesNotParseIsNotTopLevel pins 792b27c1a5 for the Given as
// written: a vault whose only note-like file has a leading ID that does not parse is not reported as flat.
// A malformed name beside a top-level note is open question 861d35.
func Test792b27c1a5_VaultHasOnlyTopLevelNotes_AnIDThatDoesNotParseIsNotTopLevel(t *testing.T) {
	t.Parallel()

	unparseable := map[string]string{
		"leading letter":   "a1.2026-07-01.bad-id.md",
		"symbol in the ID": "1-2.2026-07-01.bad-id.md",
	}

	for name, filename := range unparseable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			fileSystem := nrp1VaultFS(map[string][]byte{"/vault/" + filename: []byte("note")})

			g.Expect(cli.ExportVaultHasOnlyTopLevelNotes("/vault", fileSystem)).To(BeFalse())
		})
	}
}

// Test792b27c2a4_RunReparentLuhmann_BranchedNotesAndNotesWithoutASidecarAreLeftOut pins 792b27c2a4: a branched note
// (ID 7a), a note without a sidecar and notes with an unusable sidecar are neither a candidate nor a target, and
// do not enter the fingerprint of the considered notes.
func Test792b27c2a4_RunReparentLuhmann_BranchedNotesAndNotesWithoutASidecarAreLeftOut(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	baselineDeps, _ := newReparentDeps(maps.Clone(files), slices.Clone(names))
	baselineFingerprint := reparentDeriveFingerprint(t, baselineDeps)

	similar := []float32{1, 0, 0}
	branched := "7a.2026-01-04.branch-note.md"
	noSidecar := "30.2026-01-05.no-sidecar.md"
	emptyVector := "31.2026-01-06.empty-vector.md"
	corrupt := "32.2026-01-07.corrupt-sidecar.md"

	for _, name := range []string{branched, noSidecar, emptyVector, corrupt} {
		files["/vault/"+name] = []byte("---\ntype: fact\n---\n\nbody of " + name + "\n")
	}

	files["/vault/"+strings.TrimSuffix(branched, ".md")+".vec.json"] = reparentSidecar(similar)
	files["/vault/"+strings.TrimSuffix(emptyVector, ".md")+".vec.json"] = reparentSidecar([]float32{})
	files["/vault/"+strings.TrimSuffix(corrupt, ".md")+".vec.json"] = []byte("{not json")

	names = append(names, branched, noSidecar, emptyVector, corrupt)
	deps, _ := newReparentDeps(files, names)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(t.Context(), "/vault", "/chunks", "", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	payload := nrp1DerivePayload(t, &stdout)

	g.Expect(nrp1CandidatePairs(payload)).To(ConsistOf("12>7", "7>12"),
		"only the two top-level notes with a usable sidecar are considered")
	g.Expect(payload["fingerprint"]).To(Equal(baselineFingerprint),
		"the notes left out must not enter the fingerprint of the considered notes")
}

// Test792b27c2a5_RunReparentLuhmann_AnExcerptIsCapped pins 792b27c2a5: a body longer than 200 bytes gives an
// excerpt of the first 200 bytes of the trimmed body, counted in bytes (a 300-byte body of two-byte runes).
func Test792b27c2a5_RunReparentLuhmann_AnExcerptIsCapped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const excerptBytes = 200

	asciiBody := strings.Repeat("abcdefghij", 30)
	twoByteBody := strings.Repeat("é", 150)

	files, names := twoRelatedTopLevelNotesFixture()
	files["/vault/7.2026-01-01.first-note.md"] = []byte(
		"---\ntype: fact\nluhmann: \"7\"\ncreated: 2026-01-01\n---\n\n\n  " + asciiBody + "\n\n")
	files["/vault/12.2026-01-02.second-note.md"] = []byte(
		"---\ntype: fact\nluhmann: \"12\"\ncreated: 2026-01-02\n---\n\n" + twoByteBody + "\n")

	deps, _ := newReparentDeps(files, names)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(t.Context(), "/vault", "/chunks", "", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	payload := nrp1DerivePayload(t, &stdout)
	excerpts := map[string]string{}

	for _, candidate := range nrp1Candidates(t, payload) {
		excerpts[nrp1String(candidate["note"])] = nrp1String(candidate["note_excerpt"])
		excerpts[nrp1String(candidate["target"])] = nrp1String(candidate["target_excerpt"])
	}

	g.Expect(excerpts["7"]).To(Equal(asciiBody[:excerptBytes]))
	g.Expect(excerpts["12"]).To(HaveLen(excerptBytes))
	g.Expect(excerpts["12"]).To(Equal(strings.Repeat("é", excerptBytes/len("é"))))
}

// Test792b27c3a1_RunReparentLuhmann_ThePayloadCarriesCandidatesFingerprintAndNextCommand pins 792b27c3a1: the
// derive output is one JSON document holding exactly the candidates, a fingerprint over the considered notes,
// a next_command and an instruction.
func Test792b27c3a1_RunReparentLuhmann_ThePayloadCarriesCandidatesFingerprintAndNextCommand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, _ := newReparentDeps(files, names)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(t.Context(), "/vault", "/chunks", "", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	payload := nrp1DerivePayload(t, &stdout)

	g.Expect(slices.Sorted(maps.Keys(payload))).To(Equal([]string{
		"candidates", "fingerprint", "instruction", "next_command",
	}))
	g.Expect(nrp1CandidatePairs(payload)).To(ConsistOf("12>7", "7>12"))

	for _, candidate := range nrp1Candidates(t, payload) {
		g.Expect(candidate).To(HaveKey("note_excerpt"))
		g.Expect(candidate).To(HaveKey("target_excerpt"))
		g.Expect(candidate).To(HaveKey("similarity"))
	}

	g.Expect(payload["fingerprint"]).To(MatchRegexp(`^n3-[0-9a-f]{64}$`),
		"the vocabulary derivation fingerprint over the three considered notes")
	g.Expect(payload["next_command"]).To(Equal("engram update --reparent-luhmann --answers <path-to-answers-file>"))
	g.Expect(payload["instruction"]).NotTo(BeEmpty())
}

// Test792b27c3a2_RunReparentLuhmann_NoCandidatesIsAMessage pins 792b27c3a2: with no candidate the derive output is
// exactly the one-line message, not a JSON document.
func Test792b27c3a2_RunReparentLuhmann_NoCandidatesIsAMessage(t *testing.T) {
	t.Parallel()

	const message = "update --reparent-luhmann: no candidates found\n"

	t.Run("no pair is similar enough", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		noteA := "1.2026-01-01.alpha.md"
		noteB := "2.2026-01-02.beta.md"
		files := map[string][]byte{
			"/vault/" + noteA: []byte("---\ntype: fact\n---\n\nalpha body.\n"),
			"/vault/" + noteB: []byte("---\ntype: fact\n---\n\nbeta body.\n"),
		}
		files["/vault/1.2026-01-01.alpha.vec.json"] = reparentSidecar([]float32{1, 0, 0})
		files["/vault/2.2026-01-02.beta.vec.json"] = reparentSidecar([]float32{0, 1, 0})

		deps, renames := newReparentDeps(files, []string{noteA, noteB})

		var stdout bytes.Buffer

		err := cli.RunReparentLuhmann(t.Context(), "/vault", "/chunks", "", false, deps, &stdout)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stdout.String()).To(Equal(message))
		g.Expect(renames).To(BeEmpty())
	})

	t.Run("a vault with no note", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		deps, _ := newReparentDeps(map[string][]byte{}, []string{})

		var stdout bytes.Buffer

		err := cli.RunReparentLuhmann(t.Context(), "/vault", "/chunks", "", false, deps, &stdout)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stdout.String()).To(Equal(message))
	})
}

// Test792b27c5a3_RunReparentLuhmann_TwoChildrenAreAllocatedInOrder pins 792b27c5a3: notes 2 and 3 both answered
// continuation of note 1 (listed 3 first in the answers) map to 1a and 1b in ascending order of the note IDs,
// each keeping its date and slug; note 1 is not renamed.
func Test792b27c5a3_RunReparentLuhmann_TwoChildrenAreAllocatedInOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	first := "1.2026-01-01.first-note.md"
	second := "2.2026-01-02.second-note.md"
	third := "3.2026-01-03.third-note.md"
	files := map[string][]byte{}
	vectors := map[string][]float32{first: {1, 0, 0}, second: {0.99, 0.01, 0}, third: {0.98, 0.02, 0}}

	for name, vector := range vectors {
		files["/vault/"+name] = []byte("---\ntype: fact\n---\n\nbody of " + name + "\n")
		files["/vault/"+strings.TrimSuffix(name, ".md")+".vec.json"] = reparentSidecar(vector)
	}

	deps, renames := newReparentDeps(files, []string{first, second, third})
	fingerprint := reparentDeriveFingerprint(t, deps)

	files["/answers.json"] = []byte(`{"reparenting":[` +
		`{"note":"3","position":"continuation","target":"1"},` +
		`{"note":"2","position":"continuation","target":"1"}],"fingerprint":"` + fingerprint + `"}`)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(t.Context(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	renamedNotes := map[string]string{}

	for oldPath, newPath := range renames {
		if strings.HasSuffix(oldPath, ".md") {
			renamedNotes[oldPath] = newPath
		}
	}

	g.Expect(renamedNotes).To(Equal(map[string]string{
		"/vault/" + second: "/vault/1a.2026-01-02.second-note.md",
		"/vault/" + third:  "/vault/1b.2026-01-03.third-note.md",
	}))
}

// Test792b27c9a3_RunReparentLuhmann_AFailedPruneKeepsTheRename pins 792b27c9a3: the rename and the re-index
// succeeded and the prune fails; the error and the output name the prune stage and the manual fallback, the
// renamed note stays renamed and the re-index is kept.
func Test792b27c9a3_RunReparentLuhmann_AFailedPruneKeepsTheRename(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	errPruneWrite := errors.New("note reparenting test: manifest write refused")

	files, names := twoRelatedTopLevelNotesFixture()
	oldPath := "/vault/12.2026-01-02.second-note.md"
	newPath := "/vault/7a.2026-01-02.second-note.md"
	files["/chunks/manifest.json"] = []byte(
		`{"` + oldPath + `":{"mtime_unix_nano":1,"size":10,"file_hash":"sha256:old"}}`)

	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)
	deps.Prune.WriteFile = func(string, []byte) error { return errPruneWrite }

	files["/answers.json"] = []byte(
		`{"reparenting":[{"note":"12","position":"continuation","target":"7"}],"fingerprint":"` + fingerprint + `"}`)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(t.Context(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
	g.Expect(err).To(MatchError(errPruneWrite))
	g.Expect(err).To(MatchError(ContainSubstring("pruning stale chunk-manifest entries")))

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("pruning stale chunk-manifest entries failed"))
	g.Expect(out).To(ContainSubstring("run `engram prune` manually"))
	g.Expect(out).To(ContainSubstring("vault rename above is intact and authoritative"))
	g.Expect(out).NotTo(ContainSubstring("re-indexing renamed notes failed"))

	g.Expect(renames).To(HaveKeyWithValue(oldPath, newPath), "the rename must be kept")
	g.Expect(files).To(HaveKey(newPath))
	g.Expect(files).NotTo(HaveKey(oldPath), "the rename must not be undone")

	var manifest map[string]map[string]any

	decodeErr := json.Unmarshal(files["/chunks/manifest.json"], &manifest)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	g.Expect(manifest).To(HaveKey(newPath), "the re-index before the failed prune must be kept")
}

// nrp1UnreadableFS is a vault filesystem whose directory listing is refused.
type nrp1UnreadableFS struct {
	*u1FS
}

// ReadDir refuses with a permission error, whatever the path.
func (nrp1UnreadableFS) ReadDir(string) ([]update.DirEntry, error) {
	return nil, fs.ErrPermission
}

// nrp1CandidatePairs returns "note>target" for every candidate of a derive payload.
func nrp1CandidatePairs(payload map[string]any) []string {
	candidates, _ := payload["candidates"].([]any)
	pairs := make([]string, 0, len(candidates))

	for _, raw := range candidates {
		candidate, _ := raw.(map[string]any)
		pairs = append(pairs, nrp1String(candidate["note"])+">"+nrp1String(candidate["target"]))
	}

	return pairs
}

// nrp1Candidates returns the candidates of a derive payload as maps.
func nrp1Candidates(t *testing.T, payload map[string]any) []map[string]any {
	t.Helper()

	g := NewWithT(t)

	raw, ok := payload["candidates"].([]any)
	g.Expect(ok).To(BeTrue(), "the payload must hold a candidates list")

	candidates := make([]map[string]any, 0, len(raw))

	for _, item := range raw {
		candidate, isMap := item.(map[string]any)
		g.Expect(isMap).To(BeTrue(), "every candidate must be a JSON object")

		candidates = append(candidates, candidate)
	}

	return candidates
}

// nrp1DerivePayload decodes the derive output as one JSON document.
func nrp1DerivePayload(t *testing.T, stdout *bytes.Buffer) map[string]any {
	t.Helper()

	g := NewWithT(t)

	var payload map[string]any

	decodeErr := json.Unmarshal(stdout.Bytes(), &payload)
	g.Expect(decodeErr).NotTo(HaveOccurred(), "the derive output must be one JSON document: %s", stdout.String())

	return payload
}

// nrp1String returns value when it is a string, else "".
func nrp1String(value any) string {
	text, _ := value.(string)

	return text
}

// nrp1VaultFS returns a readable /vault directory holding files.
func nrp1VaultFS(files map[string][]byte) *u1FS {
	fileSystem := newU1FS()
	fileSystem.dirs["/vault"] = true

	maps.Copy(fileSystem.files, files)

	return fileSystem
}
