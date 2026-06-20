package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestOsPruneRemovesDeadSource drives the production os wiring (Stat/Remove/
// Read/Write) end-to-end through a temp dir: a manifest references one live
// source (file present) and one dead source (file absent); prune deletes the
// dead source's index file and manifest entry while keeping the live one.
func TestOsPruneRemovesDeadSource(t *testing.T) {
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

	err := cli.RunPrune(context.Background(), cli.PruneArgs{ChunksDir: chunksDir}, cli.ExportNewOsPruneDeps(), &out)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(gomega.ContainSubstring("removed 1 dead source"))

	_, statErr := os.Stat(deadIndex)
	g.Expect(os.IsNotExist(statErr)).To(gomega.BeTrue(), "dead source index file removed")

	_, statErr = os.Stat(liveIndex)
	g.Expect(statErr).NotTo(gomega.HaveOccurred(), "live source index file kept")
}
