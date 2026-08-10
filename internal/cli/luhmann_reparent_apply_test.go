package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
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

// TestRunReparentLuhmann_ApplyDryRunSkipsPipeline covers design.md's
// Non-Goal that --dry-run never touches the chunk index, even with a
// non-empty rename map.
func TestRunReparentLuhmann_ApplyDryRunSkipsPipeline(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)

	answers := `{"reparenting":[{"note":"12","position":"continuation","target":"7"}],"fingerprint":"` +
		fingerprint + `"}`
	files["/answers.json"] = []byte(answers)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", true, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(renames).To(BeEmpty())

	for path := range files {
		g.Expect(path).NotTo(HavePrefix("/chunks/"), "dry-run must never create a chunk-index file")
	}
}

// TestRunReparentLuhmann_ApplyPipelineFailureDoesNotRollbackRename covers
// spec "Chunk-index pipeline failure does not roll back the rename": when
// the rename+rewrite step succeeds but RunIngest subsequently fails (here,
// the embedder errors), apply reports which stage failed, that the vault
// rename is intact, and the manual-fallback commands — WITHOUT undoing the
// already-applied rename.
func TestRunReparentLuhmann_ApplyPipelineFailureDoesNotRollbackRename(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)

	// Force the ingest stage to fail.
	deps.Ingest.Embedder = reparentFailingEmbedder{}

	answers := `{"reparenting":[{"note":"12","position":"continuation","target":"7"}],"fingerprint":"` +
		fingerprint + `"}`
	files["/answers.json"] = []byte(answers)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
	g.Expect(err).To(HaveOccurred())

	g.Expect(renames).To(HaveKey("/vault/12.2026-01-02.second-note.md"), "the rename must still have happened")
	g.Expect(files).NotTo(HaveKey("/vault/12.2026-01-02.second-note.md"), "old path must be gone (renamed, not reverted)")

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("re-indexing renamed notes failed"))
	g.Expect(out).To(ContainSubstring("vault rename above is intact and authoritative"))
	g.Expect(out).To(ContainSubstring("engram ingest --auto"))
	g.Expect(out).To(ContainSubstring("engram prune"))
}

// TestRunReparentLuhmann_ApplyPipelineReindexesAndDetachesStaleEntry covers
// spec "Apply phase consumes agent-authored disposition answers" scenario
// "Continuation answer renames, rewrites, and reconciles the chunk index":
// a successful apply, on its own, re-indexes the renamed note's new path
// AND detaches the old path's now-stale chunk-manifest entry — no separate
// `engram ingest --auto` / `engram prune` call from the test.
func TestRunReparentLuhmann_ApplyPipelineReindexesAndDetachesStaleEntry(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	oldPath := "/vault/12.2026-01-02.second-note.md"

	// Simulate the vault having already been ingested at the old path
	// before this reparent run.
	files["/chunks/manifest.json"] = []byte(
		`{"` + oldPath + `":{"mtime_unix_nano":1,"size":10,"file_hash":"sha256:old"}}`,
	)

	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)

	answers := `{"reparenting":[{"note":"12","position":"continuation","target":"7"}],"fingerprint":"` +
		fingerprint + `"}`
	files["/answers.json"] = []byte(answers)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	newPath, renamed := renames[oldPath]
	g.Expect(renamed).To(BeTrue())

	var manifest map[string]map[string]any

	decodeErr := json.Unmarshal(files["/chunks/manifest.json"], &manifest)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	g.Expect(manifest).NotTo(HaveKey(oldPath), "old path's stale manifest entry must be detached by RunPrune")
	g.Expect(manifest).To(HaveKey(newPath), "new path must be indexed by RunIngest")
}

// TestRunReparentLuhmann_ApplyReportsFurtherCandidatesRemain covers spec
// "Apply reports whether further candidates remain": after a successful
// pipeline-complete apply, when the vault's now-current state still has an
// above-floor candidate pair, apply's output reports the count and the
// literal next command to loop — WITHOUT printing the full candidate
// payload (that only happens on an actual derive run).
func TestRunReparentLuhmann_ApplyReportsFurtherCandidatesRemain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Three top-level notes: 7 and 12 are near-identical (an above-floor
	// pair), 20 is unrelated. Answering only 20 as top leaves 7/12 as a
	// still-unresolved candidate pair after this apply.
	files, names := twoRelatedTopLevelNotesFixture()
	deps, _ := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)

	answers := `{"reparenting":[{"note":"20","position":"top","target":"7"}],"fingerprint":"` + fingerprint + `"}`
	files["/answers.json"] = []byte(answers)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("further candidate"))
	g.Expect(out).To(ContainSubstring("run `engram update --reparent-luhmann` again"))

	var payload map[string]any
	g.Expect(json.Unmarshal([]byte(out), &payload)).To(HaveOccurred(),
		"the further-candidates report must not print the full candidate JSON payload")
}

// TestRunReparentLuhmann_ApplyReportsNoFurtherCandidates covers spec
// scenario "No further candidates": once every above-floor pair has been
// answered, apply reports the vault as fully evaluated.
func TestRunReparentLuhmann_ApplyReportsNoFurtherCandidates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, _ := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)

	answers := `{"reparenting":[` +
		`{"note":"12","position":"continuation","target":"7"},` +
		`{"note":"20","position":"top","target":"7"}` +
		`],"fingerprint":"` + fingerprint + `"}`
	files["/answers.json"] = []byte(answers)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("no further candidates — vault fully evaluated"))
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

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
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

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(renames).To(BeEmpty())
	g.Expect(files).To(HaveKey("/vault/12.2026-01-02.second-note.md"), "the top-answered note keeps its filename")
}

// TestRunReparentLuhmann_ApplyZeroRenamesSkipsPipeline covers spec's
// "Top answer is a no-op" scenario extended to the pipeline: an all-`top`
// answers file renames nothing, so apply must not touch the chunk index at
// all (no ingest, no prune) — nothing changed, nothing to reconcile.
func TestRunReparentLuhmann_ApplyZeroRenamesSkipsPipeline(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)

	answers := `{"reparenting":[{"note":"12","position":"top","target":"7"}],"fingerprint":"` + fingerprint + `"}`
	files["/answers.json"] = []byte(answers)

	before := make(map[string][]byte, len(files))
	maps.Copy(before, files)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(renames).To(BeEmpty())

	for k := range before {
		if strings.HasPrefix(k, "/chunks/") {
			_, stillPresent := files[k]
			g.Expect(stillPresent).To(BeFalse(), "no chunk-index file may be created for a zero-rename apply: %s", k)
		}
	}
}

func TestRunReparentLuhmann_DeriveEmitsCandidates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, _ := newReparentDeps(files, names)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "", false, deps, &stdout)
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

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "", false, deps, &stdout)
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

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("no candidates found"))
}

// TestRunReparentLuhmann_DeriveOutputNamesNextCommand covers spec
// "Derive phase proposes candidates from existing embeddings": the derive
// payload includes a literal, fillable `engram update --reparent-luhmann
// --answers <path>` next-command line alongside the machine-parseable
// candidate payload.
func TestRunReparentLuhmann_DeriveOutputNamesNextCommand(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()
	deps, _ := newReparentDeps(files, names)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	var payload struct {
		NextCommand string `json:"next_command"` //nolint:tagliatelle // payload keys follow the vault's snake_case contract
	}

	decodeErr := json.Unmarshal(stdout.Bytes(), &payload)
	g.Expect(decodeErr).NotTo(HaveOccurred())
	g.Expect(payload.NextCommand).To(ContainSubstring("engram update --reparent-luhmann --answers"))
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

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", true, deps, &stdout)
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

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "", true, deps, &stdout)
	g.Expect(err).To(HaveOccurred())
	g.Expect(renames).To(BeEmpty())
	g.Expect(stdout.String()).To(BeEmpty())
}

// unexported variables.
var (
	errReparentEmbedFailed = errors.New("reparent test: embedder failed")
)

// reparentFailingEmbedder always errors, forcing RunIngest to fail mid-pipeline so
// the pipeline-failure/no-rollback path can be exercised.
type reparentFailingEmbedder struct{}

func (reparentFailingEmbedder) Dims() int { return 1 }

func (reparentFailingEmbedder) Embed(context.Context, string) ([]float32, error) {
	return nil, errReparentEmbedFailed
}

func (reparentFailingEmbedder) ModelID() string { return "failing@1" }

// reparentFakeEmbedder is a deterministic stand-in Embedder for the
// in-process RunIngest call inside apply's mechanical pipeline — tests never
// exercise real embedding math, only that ingest ran and indexed something.
type reparentFakeEmbedder struct{}

func (reparentFakeEmbedder) Dims() int { return 1 }

func (reparentFakeEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{1}, nil
}

func (reparentFakeEmbedder) ModelID() string { return "reparent-fake@1" }

type reparentNotFoundError struct{ path string }

func (e *reparentNotFoundError) Error() string { return "not found: " + e.path }

// newReparentDeps builds a full cli.ReparentDeps (rename/rewrite +
// in-process ingest + in-process prune) over a single shared in-memory file
// map, so a pipeline test can observe the chunk index (under "/chunks")
// react to the rename actually applied to the vault (under "/vault"), with
// renames tracked separately so tests can assert on them directly.
func newReparentDeps(files map[string][]byte, names []string) (cli.ReparentDeps, map[string]string) {
	renames := map[string]string{}

	readFile := func(path string) ([]byte, error) {
		data, ok := files[path]
		if !ok {
			return nil, &reparentNotFoundError{path: path}
		}

		return data, nil
	}
	writeFile := func(path string, data []byte) error {
		files[path] = data

		return nil
	}
	remove := func(path string) error {
		delete(files, path)

		return nil
	}

	renameDeps := cli.RenameRewriteDeps{
		ListMD:    func(string) ([]string, error) { return names, nil },
		ReadFile:  readFile,
		WriteFile: writeFile,
		Rename: func(oldPath, newPath string) error {
			renames[oldPath] = newPath
			if data, ok := files[oldPath]; ok {
				files[newPath] = data
				delete(files, oldPath)
			}

			return nil
		},
	}

	ingestDeps := cli.IngestDeps{
		ReadFile:  readFile,
		WriteFile: writeFile,
		Stat:      func(string) (cli.SourceStat, error) { return cli.SourceStat{}, nil },
		ListSources: func(root cli.SweepRoot) ([]string, error) {
			var paths []string

			prefix := strings.TrimSuffix(root.Path, "/") + "/"
			for path := range files {
				if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, ".md") {
					paths = append(paths, path)
				}
			}

			return paths, nil
		},
		Embedder: reparentFakeEmbedder{},
		Remove:   remove,
	}

	pruneDeps := cli.PruneDeps{
		ReadFile:  readFile,
		WriteFile: writeFile,
		Exists:    func(path string) bool { _, ok := files[path]; return ok },
		ListIndexes: func(dir string) ([]string, error) {
			var paths []string

			prefix := strings.TrimSuffix(dir, "/") + "/"
			for path := range files {
				if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, ".jsonl") {
					paths = append(paths, path)
				}
			}

			return paths, nil
		},
		Remove: remove,
	}

	return cli.ReparentDeps{Rename: renameDeps, Ingest: ingestDeps, Prune: pruneDeps}, renames
}

func reparentDeriveFingerprint(t *testing.T, deps cli.ReparentDeps) string {
	t.Helper()

	g := NewWithT(t)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "", false, deps, &stdout)
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
