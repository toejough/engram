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

	paths := make([]string, 0, len(roots))
	excludesByPath := map[string][]string{}

	for _, root := range roots {
		paths = append(paths, root.Path)
		excludesByPath[root.Path] = root.ExcludeDirs
	}

	g.Expect(paths).To(gomega.ContainElement("/home/dev/proj"), "repo root (dir holding .git)")
	g.Expect(paths).To(gomega.ContainElement("/home/dev/proj/.claude"), "project .claude")
	g.Expect(paths).To(gomega.ContainElement("/home/dev/.claude"), "ancestor .claude")
	g.Expect(paths).To(gomega.ContainElement("/sessions/-home-dev-proj-sub"), "session log dir")
	g.Expect(excludesByPath["/home/dev/.claude"]).To(gomega.ContainElements("projects", "plugins", "cache"),
		".claude roots add harness-state excludes")
	g.Expect(excludesByPath["/home/dev/proj"]).NotTo(gomega.ContainElement("projects"),
		"repo sweeps keep only the general excludes")
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

	g.Expect(roots).To(gomega.HaveLen(1), "only extra_roots when toggles are off")

	if len(roots) != 1 {
		return
	}

	g.Expect(roots[0].Path).To(gomega.Equal("/opt/notes"))
}

func TestResolveSweepRootsNoRepoFallsBackToCwd(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := specFS{dirs: map[string]bool{}}

	roots := cli.ResolveSweepRoots(cli.DefaultSweepSpec(), cli.SweepEnv{
		Cwd: "/tmp/scratch", SessionDir: "", IsDir: fs.isDir,
	})

	paths := make([]string, 0, len(roots))
	for _, root := range roots {
		paths = append(paths, root.Path)
	}

	g.Expect(paths).To(gomega.ContainElement("/tmp/scratch"), "no VCS marker: sweep cwd itself")
}

func TestDefaultSweepSpecSkipsNonPersistentWorkspaces(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	spec := cli.DefaultSweepSpec()

	g.Expect(spec.NonPersistentPrefixes).To(gomega.ContainElements("-private-tmp-", "-tmp-", "-var-folders-"))
}

func TestResolveSweepRootsAttachesPrefixesToSessionRootOnly(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := specFS{dirs: map[string]bool{
		"/home/dev/proj/.git":      true,
		"/sessions/-home-dev-proj": true,
	}}

	roots := cli.ResolveSweepRoots(cli.DefaultSweepSpec(), cli.SweepEnv{
		Cwd: "/home/dev/proj", SessionDir: "/sessions/-home-dev-proj", IsDir: fs.isDir,
	})

	prefixesByPath := map[string][]string{}
	for _, root := range roots {
		prefixesByPath[root.Path] = root.ExcludePrefixes
	}

	g.Expect(prefixesByPath["/sessions/-home-dev-proj"]).To(gomega.ContainElement("-private-tmp-"),
		"session-logs root prunes non-persistent project dirs")
	g.Expect(prefixesByPath["/home/dev/proj"]).To(gomega.BeEmpty(),
		"repo-markdown root carries no non-persistent prefixes")
}

// specFS fakes the directory-existence checks spec resolution makes.
type specFS struct {
	dirs map[string]bool
}

func (s specFS) isDir(path string) bool { return s.dirs[path] }
