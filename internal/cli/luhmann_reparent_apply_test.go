package cli_test

import (
	"bytes"
	"encoding/json"
	"maps"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"

	"github.com/toejough/engram/internal/cli"
)

func TestRunReparentLuhmann_ApplyContinuationRenamesAndRewritesBacklinks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()

	// Note 20 links to note 12 — the rename must chase this reference.
	referencer := "20.2026-01-03.unrelated-note.md"
	files["/vault/"+referencer] = []byte(
		"---\ntype: fact\nluhmann: \"20\"\ncreated: 2026-01-03\n---\n\nsee [[12.2026-01-02.second-note]] also.\n",
	)

	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)

	answers := `{"reparenting":[{"note":"12","position":"continuation","target":"7"}],"fingerprint":"` +
		fingerprint + `"}`
	files["/answers.json"] = []byte(answers)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "/answers.json", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(renames).To(HaveKey("/vault/12.2026-01-02.second-note.md"))
	newPath := renames["/vault/12.2026-01-02.second-note.md"]
	g.Expect(newPath).To(Equal("/vault/7a.2026-01-02.second-note.md"))

	updatedReferencer, ok := files["/vault/"+referencer]
	g.Expect(ok).To(BeTrue())
	g.Expect(string(updatedReferencer)).To(ContainSubstring("[[7a.2026-01-02.second-note]]"))
	g.Expect(string(updatedReferencer)).NotTo(ContainSubstring("[[12.2026-01-02.second-note]]"))

	renamedContent, ok := files[newPath]
	g.Expect(ok).To(BeTrue())
	g.Expect(string(renamedContent)).To(ContainSubstring(`luhmann: "7a"`))
}

func TestRunReparentLuhmann_ApplyStaleAnswersRejected(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, renames := newReparentDeps(files, names)

	before := make(map[string][]byte, len(files))
	maps.Copy(before, files)

	answers := `{"reparenting":[{"note":"12","position":"continuation","target":"7"}],"fingerprint":"stale-value"}`
	files["/answers.json"] = []byte(answers)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "/answers.json", false, deps, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("stale")))
	g.Expect(renames).To(BeEmpty())

	for k, v := range before {
		g.Expect(files[k]).To(Equal(v), "no file may be modified on a stale-answers rejection: %s", k)
	}
}

func TestRunReparentLuhmann_ApplyTopAnswerIsNoop(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)

	answers := `{"reparenting":[{"note":"12","position":"top","target":"7"}],"fingerprint":"` + fingerprint + `"}`
	files["/answers.json"] = []byte(answers)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "/answers.json", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(renames).To(BeEmpty())
	g.Expect(files).To(HaveKey("/vault/12.2026-01-02.second-note.md"), "the top-answered note keeps its filename")
}

func TestRunReparentLuhmann_DeriveEmitsCandidates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, _ := newReparentDeps(files, names)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	var payload map[string]any

	decodeErr := json.Unmarshal(stdout.Bytes(), &payload)
	g.Expect(decodeErr).NotTo(HaveOccurred(), "derive must print a JSON payload: %s", stdout.String())

	g.Expect(payload).To(HaveKey("candidates"))
	g.Expect(payload).To(HaveKey("fingerprint"))

	candidates, ok := payload["candidates"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(candidates).NotTo(BeEmpty(), "notes 7 and 12 are near-identical vectors — must surface as a candidate")

	found := false

	for _, raw := range candidates {
		entry, ok := raw.(map[string]any)
		g.Expect(ok).To(BeTrue())

		if entry["note"] == "7" && entry["target"] == "12" {
			found = true

			g.Expect(entry["similarity"]).To(BeNumerically(">", 0.9))
			g.Expect(entry["note_excerpt"]).NotTo(BeEmpty())
			g.Expect(entry["target_excerpt"]).NotTo(BeEmpty())
		}
	}

	g.Expect(found).To(BeTrue(), "expected a 7→12 candidate pair, got %v", candidates)
}

func TestRunReparentLuhmann_DeriveNeverWrites(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	before := make(map[string][]byte, len(files))
	maps.Copy(before, files)

	deps, renames := newReparentDeps(files, names)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(renames).To(BeEmpty())
	g.Expect(files).To(HaveLen(len(before)), "derive must never write any file")
}

func TestRunReparentLuhmann_DeriveNoCandidatesAboveFloor(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	noteA := "1.2026-01-01.alpha.md"
	noteB := "2.2026-01-02.beta.md"

	files := map[string][]byte{
		"/vault/" + noteA: []byte("---\ntype: fact\nluhmann: \"1\"\ncreated: 2026-01-01\n---\n\nalpha body.\n"),
		"/vault/" + noteB: []byte("---\ntype: fact\nluhmann: \"2\"\ncreated: 2026-01-02\n---\n\nbeta body.\n"),
	}
	files["/vault/1.2026-01-01.alpha.vec.json"] = reparentSidecar([]float32{1, 0, 0})
	files["/vault/2.2026-01-02.beta.vec.json"] = reparentSidecar([]float32{0, 1, 0})

	deps, _ := newReparentDeps(files, []string{noteA, noteB})

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("no candidates found"))
}

func TestRunReparentLuhmann_DryRunPreviewsWithoutWriting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()

	referencer := "20.2026-01-03.unrelated-note.md"
	files["/vault/"+referencer] = []byte(
		"---\ntype: fact\nluhmann: \"20\"\ncreated: 2026-01-03\n---\n\nsee [[12.2026-01-02.second-note]] also.\n",
	)

	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)

	answers := `{"reparenting":[{"note":"12","position":"continuation","target":"7"}],"fingerprint":"` +
		fingerprint + `"}`
	files["/answers.json"] = []byte(answers)

	before := make(map[string][]byte, len(files))
	maps.Copy(before, files)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "/answers.json", true, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(renames).To(BeEmpty())

	for k, v := range before {
		g.Expect(files[k]).To(Equal(v), "dry-run must not modify any file: %s", k)
	}

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("12.2026-01-02.second-note"))
	g.Expect(out).To(ContainSubstring("7a.2026-01-02.second-note"))
	g.Expect(out).To(ContainSubstring(referencer))
}

func TestRunReparentLuhmann_DryRunWithoutAnswersRejected(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, renames := newReparentDeps(files, names)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "", true, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(renames).To(BeEmpty())
	g.Expect(stdout.String()).To(BeEmpty())
}

type reparentNotFoundError struct{ path string }

func (e *reparentNotFoundError) Error() string { return "not found: " + e.path }

// newReparentDeps builds RenameRewriteDeps over an in-memory file map, with
// renames tracked separately so tests can assert on them.
func newReparentDeps(files map[string][]byte, names []string) (cli.RenameRewriteDeps, map[string]string) {
	renames := map[string]string{}

	return cli.RenameRewriteDeps{
		ListMD: func(string) ([]string, error) { return names, nil },
		ReadFile: func(path string) ([]byte, error) {
			data, ok := files[path]
			if !ok {
				return nil, &reparentNotFoundError{path: path}
			}

			return data, nil
		},
		WriteFile: func(path string, data []byte) error {
			files[path] = data

			return nil
		},
		Rename: func(oldPath, newPath string) error {
			renames[oldPath] = newPath
			if data, ok := files[oldPath]; ok {
				files[newPath] = data
				delete(files, oldPath)
			}

			return nil
		},
	}, renames
}

func reparentDeriveFingerprint(t *testing.T, deps cli.RenameRewriteDeps) string {
	t.Helper()

	g := NewWithT(t)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann("/vault", "", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	var payload struct {
		Fingerprint string `json:"fingerprint"`
	}

	decodeErr := json.Unmarshal(stdout.Bytes(), &payload)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	g.Expect(payload.Fingerprint).NotTo(BeEmpty())

	return payload.Fingerprint
}

// reparentSidecar builds a minimal sidecar JSON payload carrying vec as the
// body vector — the only field the derive phase reads.
func reparentSidecar(vec []float32) []byte {
	return embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: 1, EmbeddingModelID: "test", Dims: len(vec),
		BodyVector: vec, SituationVector: make([]float32, len(vec)),
	})
}

// twoRelatedTopLevelNotesFixture builds two top-level notes whose body
// vectors are near-identical (cosine ~1.0, well above any reasonable floor)
// and a third, unrelated note far away in vector space.
func twoRelatedTopLevelNotesFixture() (map[string][]byte, []string) {
	noteA := "7.2026-01-01.first-note.md"
	noteB := "12.2026-01-02.second-note.md"
	noteC := "20.2026-01-03.unrelated-note.md"

	files := map[string][]byte{
		"/vault/" + noteA: []byte("---\ntype: fact\nluhmann: \"7\"\ncreated: 2026-01-01\n---\n\nfirst note body text.\n"),
		"/vault/" + noteB: []byte(
			"---\ntype: fact\nluhmann: \"12\"\ncreated: 2026-01-02\n---\n\nsecond note body text.\n",
		),
		"/vault/" + noteC: []byte(
			"---\ntype: fact\nluhmann: \"20\"\ncreated: 2026-01-03\n---\n\nunrelated note body text.\n",
		),
		"/vault/" + noteA + ".vec.json.tmp": nil, // placeholder to keep gofmt happy about map ordering
	}
	delete(files, "/vault/"+noteA+".vec.json.tmp")

	files["/vault/"+strings.TrimSuffix(noteA, ".md")+".vec.json"] = reparentSidecar([]float32{1, 0, 0})
	files["/vault/"+strings.TrimSuffix(noteB, ".md")+".vec.json"] = reparentSidecar([]float32{0.99, 0.01, 0})
	files["/vault/"+strings.TrimSuffix(noteC, ".md")+".vec.json"] = reparentSidecar([]float32{0, 0, 1})

	return files, []string{noteA, noteB, noteC}
}
