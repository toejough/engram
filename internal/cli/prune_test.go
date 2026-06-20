package cli_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestPruneRemovesDeadSources(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	live := "/sessions/live.jsonl"
	dead := "/sessions/-private-tmp-eval/dead.jsonl"
	manifest := map[string]map[string]any{
		live: {"mtime_unix_nano": 1, "size": 10, "file_hash": "sha256:a"},
		dead: {"mtime_unix_nano": 2, "size": 20, "file_hash": "sha256:b"},
	}
	manBytes, _ := json.Marshal(manifest)

	fs := newPruneFS()
	fs.files["/chunks/manifest.json"] = manBytes
	fs.files["/chunks/"+cli.ExportIndexFileName(live)] = []byte("[]")
	fs.files["/chunks/"+cli.ExportIndexFileName(dead)] = []byte("[]")
	fs.exists[live] = true // dead source file is absent

	err := cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks"}, fs.pruneDeps(), io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	_, deadIndexPresent := fs.files["/chunks/"+cli.ExportIndexFileName(dead)]
	g.Expect(deadIndexPresent).To(gomega.BeFalse(), "dead source index removed")

	_, liveIndexPresent := fs.files["/chunks/"+cli.ExportIndexFileName(live)]
	g.Expect(liveIndexPresent).To(gomega.BeTrue(), "live source index kept")

	var rewritten map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).To(gomega.HaveKey(live))
	g.Expect(rewritten).NotTo(gomega.HaveKey(dead), "dead source dropped from manifest")
}

type pruneFS struct {
	files  map[string][]byte
	exists map[string]bool
}

func newPruneFS() *pruneFS {
	return &pruneFS{files: map[string][]byte{}, exists: map[string]bool{}}
}

func (p *pruneFS) pruneDeps() cli.PruneDeps {
	return cli.PruneDeps{
		ReadFile:  func(path string) ([]byte, error) { return p.read(path) },
		WriteFile: func(path string, data []byte) error { p.files[path] = data; return nil },
		Exists:    func(path string) bool { return p.exists[path] },
		Remove:    func(path string) error { delete(p.files, path); return nil },
	}
}

func (p *pruneFS) read(path string) ([]byte, error) {
	data, ok := p.files[path]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}

	return data, nil
}
