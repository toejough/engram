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
	deps.ListSources = func(root string, _ []string) ([]string, error) { return listed[root], nil }
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

func TestDefaultSessionDirHonorsTranscriptDirEnv(t *testing.T) {
	// Not parallel: mutates process env.
	g := gomega.NewWithT(t)

	t.Setenv("ENGRAM_TRANSCRIPT_DIR", "/custom/sessions")

	deps := cli.ExportNewOsIngestDeps(fakeIngestEmbedder{})

	g.Expect(deps.SessionDir("/anywhere")).To(gomega.Equal("/custom/sessions"))
}

func TestDefaultSessionDirIsAllProjects(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	deps := cli.ExportNewOsIngestDeps(fakeIngestEmbedder{})
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

	deps := cli.ExportNewOsIngestDeps(fakeIngestEmbedder{})
	paths, err := deps.ListSources(dir, []string{"node_modules"})

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(paths).To(gomega.ContainElement(filepath.Join(dir, "docs", "real.md")))

	for _, p := range paths {
		g.Expect(p).NotTo(gomega.ContainSubstring("node_modules"), "dependency markdown excluded")
	}
}
