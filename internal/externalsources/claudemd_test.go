package externalsources_test

import (
	"runtime"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscoverClaudeMd_AncestorWalk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	statFn := func(path string) (bool, error) {
		// Pretend CLAUDE.md exists at /a/b and /a, plus a CLAUDE.local.md at /a/b.
		switch path {
		case "/a/b/CLAUDE.md", "/a/CLAUDE.md", "/a/b/CLAUDE.local.md":
			return true, nil
		default:
			return false, nil
		}
	}

	files := externalsources.DiscoverClaudeMd("/a/b", "/home/user", runtime.GOOS, statFn)

	paths := make([]string, 0, len(files))
	for _, f := range files {
		g.Expect(f.Kind).To(Equal(externalsources.KindClaudeMd))
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ContainElement("/a/b/CLAUDE.md"))
	g.Expect(paths).To(ContainElement("/a/b/CLAUDE.local.md"))
	g.Expect(paths).To(ContainElement("/a/CLAUDE.md"))
}

func TestDiscoverClaudeMd_EmptyHomeSkipsUserScope(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	called := make([]string, 0)
	statFn := func(path string) (bool, error) {
		called = append(called, path)
		return false, nil
	}

	files := externalsources.DiscoverClaudeMd("/x/y", "", runtime.GOOS, statFn)
	g.Expect(files).To(BeEmpty())

	for _, p := range called {
		g.Expect(p).NotTo(ContainSubstring("/.claude/CLAUDE.md"),
			"empty home should not trigger user-scope check")
	}
}

func TestDiscoverClaudeMd_IncludesManagedPolicy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	wantPath := externalsources.ManagedPolicyPath(runtime.GOOS)
	g.Expect(wantPath).NotTo(BeEmpty())

	statFn := func(path string) (bool, error) {
		return path == wantPath, nil
	}

	files := externalsources.DiscoverClaudeMd("/somewhere", "/home/user", runtime.GOOS, statFn)

	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ContainElement(wantPath))
}

func TestDiscoverClaudeMd_IncludesUserScope(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	statFn := func(path string) (bool, error) {
		return path == "/home/user/.claude/CLAUDE.md", nil
	}

	files := externalsources.DiscoverClaudeMd("/somewhere", "/home/user", runtime.GOOS, statFn)

	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ContainElement("/home/user/.claude/CLAUDE.md"))
}

func TestDiscoverClaudeMd_NoFilesPresent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	statFn := func(_ string) (bool, error) { return false, nil }

	files := externalsources.DiscoverClaudeMd("/x/y", "/home/user", runtime.GOOS, statFn)
	g.Expect(files).To(BeEmpty())
}

func TestDiscoverClaudeMd_UnrecognizedGOOSSkipsManagedPolicy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	statFn := func(_ string) (bool, error) { return true, nil }

	files := externalsources.DiscoverClaudeMd("/x", "/home/user", "plan9", statFn)

	for _, f := range files {
		g.Expect(f.Path).NotTo(ContainSubstring("ClaudeCode"),
			"unrecognized GOOS should not include managed-policy path")
	}
}

func TestManagedPolicyPath_KnownPlatforms(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(externalsources.ManagedPolicyPath("darwin")).
		To(Equal("/Library/Application Support/ClaudeCode/CLAUDE.md"))
	g.Expect(externalsources.ManagedPolicyPath("linux")).
		To(Equal("/etc/claude-code/CLAUDE.md"))
	g.Expect(externalsources.ManagedPolicyPath("windows")).
		To(Equal(`C:\Program Files\ClaudeCode\CLAUDE.md`))
}

func TestManagedPolicyPath_UnknownPlatformReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(externalsources.ManagedPolicyPath("plan9")).To(BeEmpty())
	g.Expect(externalsources.ManagedPolicyPath("")).To(BeEmpty())
}
