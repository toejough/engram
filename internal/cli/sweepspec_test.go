package cli_test

import (
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestDefaultSweepSpecExcludesDependencyDirs(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	spec := cli.DefaultSweepSpec()

	g.Expect(spec.ExcludeDirs).To(gomega.ContainElements("node_modules", "vendor", ".git"))
	g.Expect(spec.RepoMarkdown).To(gomega.BeTrue())
	g.Expect(spec.AncestorClaudeDirs).To(gomega.BeTrue())
	g.Expect(spec.SessionLogs).To(gomega.BeTrue())
}

func TestLoadSweepSpecOverridesDefaults(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	raw := []byte(`{"repo_markdown": false, "exclude_dirs": ["node_modules", "gen"]}`)

	spec, err := cli.LoadSweepSpec(raw)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(spec.RepoMarkdown).To(gomega.BeFalse())
	g.Expect(spec.ExcludeDirs).To(gomega.ConsistOf("node_modules", "gen"))
	g.Expect(spec.SessionLogs).To(gomega.BeTrue(), "unset fields keep defaults")
}

func TestResolveSweepRootsCoversRepoAncestorsAndSessions(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := specFS{dirs: map[string]bool{
		"/home/dev/proj/.git":          true, // repo root marker
		"/home/dev/proj/.claude":       true,
		"/home/dev/.claude":            true,
		"/home/dev/proj/sub":           true,
		"/sessions/-home-dev-proj-sub": true,
	}}

	roots := cli.ResolveSweepRoots(cli.DefaultSweepSpec(), cli.SweepEnv{
		Cwd: "/home/dev/proj/sub", SessionDir: "/sessions/-home-dev-proj-sub", IsDir: fs.isDir,
	})

	g.Expect(roots).To(gomega.ContainElement("/home/dev/proj"), "repo root (dir holding .git)")
	g.Expect(roots).To(gomega.ContainElement("/home/dev/proj/.claude"), "project .claude")
	g.Expect(roots).To(gomega.ContainElement("/home/dev/.claude"), "ancestor .claude")
	g.Expect(roots).To(gomega.ContainElement("/sessions/-home-dev-proj-sub"), "session log dir")
}

func TestResolveSweepRootsHonorsSpecToggles(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := specFS{dirs: map[string]bool{
		"/home/dev/proj/.git":    true,
		"/home/dev/proj/.claude": true,
	}}
	spec := cli.SweepSpec{
		RepoMarkdown: false, AncestorClaudeDirs: false, SessionLogs: false,
		ExtraRoots: []string{"/opt/notes"},
	}

	roots := cli.ResolveSweepRoots(spec, cli.SweepEnv{
		Cwd: "/home/dev/proj", SessionDir: "/sessions/x", IsDir: fs.isDir,
	})

	g.Expect(roots).To(gomega.ConsistOf("/opt/notes"), "only extra_roots when toggles are off")
}

func TestResolveSweepRootsNoRepoFallsBackToCwd(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := specFS{dirs: map[string]bool{}}

	roots := cli.ResolveSweepRoots(cli.DefaultSweepSpec(), cli.SweepEnv{
		Cwd: "/tmp/scratch", SessionDir: "", IsDir: fs.isDir,
	})

	g.Expect(roots).To(gomega.ContainElement("/tmp/scratch"), "no VCS marker: sweep cwd itself")
}

// specFS fakes the directory-existence checks spec resolution makes.
type specFS struct {
	dirs map[string]bool
}

func (s specFS) isDir(path string) bool { return s.dirs[path] }
