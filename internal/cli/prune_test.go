package cli_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestPruneDetachesDeadSources asserts that prune DETACHES rather than
// deletes: the manifest entry for a dead source is dropped, but its
// per-source .jsonl index file (the embedded chunk vectors) is left on disk
// so chunk search — which discovers .jsonl files by directory scan, not the
// manifest — can still find and search it. A live source is untouched.
func TestPruneDetachesDeadSources(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	live := "/sessions/live.jsonl"
	dead := "/sessions/-private-tmp-eval/dead.jsonl"
	manifest := map[string]map[string]any{
		live: {"mtime_unix_nano": 1, "size": 10, "file_hash": "sha256:a"},
		dead: {"mtime_unix_nano": 2, "size": 20, "file_hash": "sha256:b"},
	}
	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs := newPruneFS()
	fs.files["/chunks/manifest.json"] = manBytes
	fs.files["/chunks/"+cli.ExportIndexFileName(live)] = []byte("[]")
	fs.files["/chunks/"+cli.ExportIndexFileName(dead)] = []byte("[]")
	fs.exists[live] = true // dead source file is absent

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks"}, fs.pruneDeps(), io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	_, deadIndexPresent := fs.files["/chunks/"+cli.ExportIndexFileName(dead)]
	g.Expect(deadIndexPresent).To(gomega.BeTrue(), "dead source index preserved on disk (still searchable)")

	_, liveIndexPresent := fs.files["/chunks/"+cli.ExportIndexFileName(live)]
	g.Expect(liveIndexPresent).To(gomega.BeTrue(), "live source index untouched")

	var rewritten map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).To(gomega.HaveKey(live))
	g.Expect(rewritten).NotTo(gomega.HaveKey(dead), "dead source dropped from manifest")
}

func TestPruneNoDeadSources(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	live := "/sessions/live.jsonl"
	manifest := map[string]map[string]any{
		live: {"mtime_unix_nano": 1, "size": 10, "file_hash": "sha256:a"},
	}

	manBytes, err := json.Marshal(manifest)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	fs := newPruneFS()
	fs.files["/chunks/manifest.json"] = manBytes
	fs.exists[live] = true // live source exists

	err = cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks"}, fs.pruneDeps(), io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Manifest should be unchanged (not rewritten since nothing pruned)
	var rewritten map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).To(gomega.HaveKey(live))
}

func TestPruneNoManifestIsNoOp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := newPruneFS() // empty — no manifest file

	err := cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks"}, fs.pruneDeps(), io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())
}

// TestRunPrune_LocksManifestAroundReadModifyWrite asserts that RunPrune
// acquires the manifest lock BEFORE reading the manifest and releases it
// AFTER writing it, preventing concurrent lost updates alongside ingest (#660).
func TestRunPrune_LocksManifestAroundReadModifyWrite(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var order []string

	const (
		chunksDir    = "/chunks"
		manifestPath = "/chunks/manifest.json"
		deadSource   = "/sessions/dead.jsonl"
	)

	manifest := map[string]map[string]any{
		deadSource: {"mtime_unix_nano": 1, "size": 5, "file_hash": "sha256:x"},
	}

	manBytes, marshalErr := json.Marshal(manifest)
	g.Expect(marshalErr).NotTo(gomega.HaveOccurred())

	if marshalErr != nil {
		return
	}

	files := map[string][]byte{
		manifestPath: manBytes,
	}

	deps := cli.PruneDeps{
		Lock: func(string) (func(), error) {
			order = append(order, "lock")

			return func() { order = append(order, "unlock") }, nil
		},
		ReadFile: func(path string) ([]byte, error) {
			if path == manifestPath {
				order = append(order, "read:"+path)
			}

			data, ok := files[path]
			if !ok {
				return nil, io.ErrUnexpectedEOF
			}

			return data, nil
		},
		WriteFile: func(path string, data []byte) error {
			if path == manifestPath {
				order = append(order, "write:"+path)
			}

			files[path] = data

			return nil
		},
		Exists: func(string) bool { return false }, // dead source: file does not exist
	}

	err := cli.RunPrune(context.Background(), cli.PruneArgs{ChunksDir: chunksDir}, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Must have all four events.
	g.Expect(order).
		To(gomega.ContainElements("lock", "read:"+manifestPath, "write:"+manifestPath, "unlock"),
			"all lock events must be recorded")

	lockIdx := sliceIndex(order, "lock")
	readIdx := sliceIndex(order, "read:"+manifestPath)
	writeIdx := sliceIndex(order, "write:"+manifestPath)
	unlockIdx := sliceIndex(order, "unlock")

	g.Expect(lockIdx).To(gomega.BeNumerically("<", readIdx),
		"lock must precede manifest read")
	g.Expect(readIdx).To(gomega.BeNumerically("<", writeIdx),
		"manifest read must precede manifest write")
	g.Expect(writeIdx).To(gomega.BeNumerically("<", unlockIdx),
		"manifest write must precede unlock")
}

type pruneFS struct {
	files  map[string][]byte
	exists map[string]bool
}

func (p *pruneFS) pruneDeps() cli.PruneDeps {
	return cli.PruneDeps{
		ReadFile:  func(path string) ([]byte, error) { return p.read(path) },
		WriteFile: func(path string, data []byte) error { p.files[path] = data; return nil },
		Exists:    func(path string) bool { return p.exists[path] },
	}
}

func (p *pruneFS) read(path string) ([]byte, error) {
	data, ok := p.files[path]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}

	return data, nil
}

func newPruneFS() *pruneFS {
	return &pruneFS{files: map[string][]byte{}, exists: map[string]bool{}}
}
