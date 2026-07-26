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

func TestDefaultSweepSpecSkipsNonPersistentWorkspaces(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	spec := cli.DefaultSweepSpec()

	g.Expect(spec.NonPersistentPrefixes).To(gomega.ContainElements("-private-tmp-", "-tmp-", "-var-folders-"))
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

func TestResolveSweepRootsClaudeAncestorExcludesJobsScratch(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// ~/.claude/jobs is agent-harness scratch (whole snapshot copies of the
	// user's engram vault included) — the .pi sweep already excludes "jobs"
	// (piExcludes in ResolveSweepRoots, and again in ingest.go's
	// piSessionSources) because PI has the same subdir shape as Claude. This
	// asserts the Claude ancestor list carries the matching exclude.
	fs := specFS{dirs: map[string]bool{
		"/home/dev/proj/.git":    true,
		"/home/dev/proj/.claude": true,
	}}

	roots := cli.ResolveSweepRoots(cli.DefaultSweepSpec(), cli.SweepEnv{
		Cwd: "/home/dev/proj", SessionDir: "", IsDir: fs.isDir,
	})

	var claudeExcludes []string

	for _, root := range roots {
		if root.Path == "/home/dev/proj/.claude" {
			claudeExcludes = root.ExcludeDirs
		}
	}

	g.Expect(claudeExcludes).To(gomega.ContainElement("jobs"),
		"jobs/ under .claude is agent scratch (whole vault snapshot copies), not user memory")
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

func TestResolveSweepRootsDropsRootsUnderNonPersistentPaths(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// No VCS marker anywhere, so repoRootFor falls back to cwd itself — the
	// exact shape of the confirmed-live bug: a sweep root (repo-markdown
	// root here, but a session-log slug is symmetric) that IS a throwaway
	// path, not merely a descendant of one. sweepListerFrom exempts a
	// root's own path from every in-walk exclude check, so such a root is
	// swept whole and permanently indexed unless it is dropped up front.
	fs := specFS{dirs: map[string]bool{
		"/private/tmp/work/proj/.claude": true,
	}}

	roots := cli.ResolveSweepRoots(cli.DefaultSweepSpec(), cli.SweepEnv{
		Cwd: "/private/tmp/work/proj", SessionDir: "", IsDir: fs.isDir,
	})

	for _, root := range roots {
		g.Expect(root.Path).NotTo(gomega.HavePrefix("/private/tmp"),
			"a sweep root under a non-persistent path must be dropped outright")
	}
}

func TestResolveSweepRootsExtraRootsBypassNonPersistentPathDrop(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Deliberately DefaultSweepSpec(), not a bare cli.SweepSpec{} literal:
	// a literal leaves NonPersistentPathPrefixes nil, so the filter never
	// engages and the gap this test targets stays masked.
	spec := cli.DefaultSweepSpec()
	spec.RepoMarkdown = false
	spec.AncestorClaudeDirs = false
	spec.AncestorPiDirs = false
	spec.SessionLogs = false
	spec.ExtraRoots = []string{"/tmp/mydata"}

	roots := cli.ResolveSweepRoots(spec, cli.SweepEnv{
		Cwd: "/home/dev/proj", SessionDir: "", IsDir: func(string) bool { return false },
	})

	g.Expect(roots).To(gomega.HaveLen(1),
		"extra_roots is explicit, deliberate config — the same bypass category as --sweep")

	if len(roots) != 1 {
		return
	}

	g.Expect(roots[0].Path).To(gomega.Equal("/tmp/mydata"),
		"ExtraRoots must be swept verbatim even under a non-persistent path prefix")
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

	// Cwd deliberately outside any non-persistent prefix: this test targets
	// repoRootFor's cwd-fallback behavior, not the non-persistent-path drop
	// (covered by TestResolveSweepRootsDropsRootsUnderNonPersistentPaths).
	roots := cli.ResolveSweepRoots(cli.DefaultSweepSpec(), cli.SweepEnv{
		Cwd: "/home/dev/scratch", SessionDir: "", IsDir: fs.isDir,
	})

	paths := make([]string, 0, len(roots))
	for _, root := range roots {
		paths = append(paths, root.Path)
	}

	g.Expect(paths).To(gomega.ContainElement("/home/dev/scratch"), "no VCS marker: sweep cwd itself")
}

// specFS fakes the directory-existence checks spec resolution makes.
type specFS struct {
	dirs map[string]bool
}

func (s specFS) isDir(path string) bool { return s.dirs[path] }
