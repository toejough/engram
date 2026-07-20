package cli_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

func TestAutoSweepUsesSpecRootsAndRepoOverride(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := newSweepFS()
	fs.put("/repo/README.md", "## Conventions\nAlways wrap errors with context before returning them.\n", 100)
	// repo override disables session logs; the session dir must NOT be swept.
	fs.put("/repo/.engram/sweep.json", `{"session_logs": false}`, 100)

	listed := map[string][]string{
		"/repo": {"/repo/README.md", "/repo/.engram/sweep.json"},
	}
	deps := sweepDeps(fs, &countingEmbedder{})
	deps.ListSources = func(root cli.SweepRoot) ([]string, error) { return listed[root.Path], nil }
	deps.IsDir = func(path string) bool {
		return map[string]bool{"/repo/.git": true, "/sessions/proj": true}[path]
	}
	deps.Getwd = func() (string, error) { return "/repo", nil }
	deps.SessionDir = func(string) string { return "/sessions/proj" }

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Auto: true, ChunksDir: "/chunks"}, deps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	records, err := chunk.DecodeRecords(fs.files["/chunks/"+cli.ExportIndexFileName("/repo/README.md")])
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).NotTo(gomega.BeEmpty(), "repo markdown swept")

	_, sessionsSwept := fs.files["/chunks/s9.jsonl"]
	g.Expect(sessionsSwept).To(gomega.BeFalse(), "session_logs disabled by repo override")
}

func TestDefaultSessionDirIsAllProjects(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	deps := cli.ExportNewIngestDeps(newTestDeps(io.Discard, io.Discard), fakeIngestEmbedder{})
	dir := deps.SessionDir("/anywhere/at/all")

	g.Expect(dir).To(gomega.HaveSuffix(filepath.Join(".claude", "projects")),
		"ALL recorded sessions, not just the current project's")
	g.Expect(deps.IsDir(t.TempDir())).To(gomega.BeTrue())
	g.Expect(deps.IsDir("/definitely/not/a/dir")).To(gomega.BeFalse())

	cwd, err := deps.Getwd()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(cwd).NotTo(gomega.BeEmpty())
}

func TestOsListSourcesSkipsExcludedDirs(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o700)).To(gomega.Succeed())
	g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o700)).To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "dep.md"), []byte("# dep"), 0o600)).
		To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "docs", "real.md"), []byte("# real"), 0o600)).
		To(gomega.Succeed())

	// Hidden dirs (.layer-run-style eval state, .obsidian, ...) are pruned by
	// the SkipHidden rule — caught live: a stale eval cfg dir held a full
	// plugin-marketplace copy that a name list would never have anticipated.
	g.Expect(os.MkdirAll(filepath.Join(dir, ".layer-run", "cfg"), 0o700)).To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, ".layer-run", "cfg", "junk.md"), []byte("# junk"), 0o600)).
		To(gomega.Succeed())

	deps := cli.ExportNewIngestDeps(newTestDeps(io.Discard, io.Discard), fakeIngestEmbedder{})
	paths, err := deps.ListSources(cli.SweepRoot{Path: dir, ExcludeDirs: []string{"node_modules"}, SkipHidden: true})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(paths).To(gomega.ContainElement(filepath.Join(dir, "docs", "real.md")))

	for _, p := range paths {
		g.Expect(p).NotTo(gomega.ContainSubstring("node_modules"), "dependency markdown excluded")
		g.Expect(p).NotTo(gomega.ContainSubstring(".layer-run"), "hidden dirs pruned")
	}
}
