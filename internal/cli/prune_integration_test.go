package cli_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestOsPruneDetachesDeadSource drives the production os wiring (Stat/Read/
// Write) end-to-end through a temp dir: a manifest references one live
// source (file present) and one dead source (file absent); prune drops the
// dead source's manifest entry but leaves its index file on disk, since
// chunk search discovers .jsonl files by directory scan and never consults
// the manifest — detached chunks stay fully searchable.
func TestOsPruneDetachesDeadSource(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	chunksDir := filepath.Join(dir, "chunks")
	g.Expect(os.MkdirAll(chunksDir, 0o700)).To(gomega.Succeed())

	live := filepath.Join(dir, "live.jsonl")
	g.Expect(os.WriteFile(live, []byte("{}"), 0o600)).To(gomega.Succeed())

	dead := filepath.Join(dir, "gone", "dead.jsonl") // never created on disk

	manifest := `{` +
		`"` + live + `":{"mtime_unix_nano":1,"size":2,"file_hash":"sha256:a"},` +
		`"` + dead + `":{"mtime_unix_nano":3,"size":4,"file_hash":"sha256:b"}}`
	g.Expect(os.WriteFile(filepath.Join(chunksDir, "manifest.json"), []byte(manifest), 0o600)).To(gomega.Succeed())

	liveIndex := filepath.Join(chunksDir, cli.ExportIndexFileName(live))
	deadIndex := filepath.Join(chunksDir, cli.ExportIndexFileName(dead))

	g.Expect(os.WriteFile(liveIndex, []byte("[]"), 0o600)).To(gomega.Succeed())
	g.Expect(os.WriteFile(deadIndex, []byte("[]"), 0o600)).To(gomega.Succeed())

	var out strings.Builder

	deps := cli.ExportNewPruneDeps(newTestDeps(io.Discard, io.Discard))

	err := cli.RunPrune(context.Background(), cli.PruneArgs{ChunksDir: chunksDir}, deps, &out)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(gomega.ContainSubstring("detached 1 source"))
	g.Expect(out.String()).To(gomega.ContainSubstring("preserved"))

	_, statErr := os.Stat(deadIndex)
	g.Expect(statErr).NotTo(gomega.HaveOccurred(), "dead source index file preserved on disk (still searchable)")

	_, statErr = os.Stat(liveIndex)
	g.Expect(statErr).NotTo(gomega.HaveOccurred(), "live source index file kept")

	manifestData, readErr := os.ReadFile(filepath.Join(chunksDir, "manifest.json"))
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	var rewritten map[string]any
	g.Expect(json.Unmarshal(manifestData, &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).NotTo(gomega.HaveKey(dead), "dead source dropped from manifest")
	g.Expect(rewritten).To(gomega.HaveKey(live), "live source kept in manifest")
}
