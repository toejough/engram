package externalsources_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscoverAutoMemory_DefaultProjectDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := func() (string, bool) { return "", false }

	wantDir := "/home/user/.claude/projects/-proj-cwd/memory"
	dirLister := func(dir string) ([]string, error) {
		if dir == wantDir {
			return []string{filepath.Join(wantDir, "MEMORY.md")}, nil
		}

		return nil, nil
	}

	files := externalsources.DiscoverAutoMemory(wantDir, "", settings, dirLister)

	g.Expect(files).To(HaveLen(1))
	g.Expect(files[0].Path).To(Equal(filepath.Join(wantDir, "MEMORY.md")))
}

func TestDiscoverAutoMemory_HonorsAutoMemoryDirectorySetting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := func() (string, bool) { return "/custom/memdir", true }

	dirLister := func(dir string) ([]string, error) {
		if dir == "/custom/memdir" {
			return []string{
				"/custom/memdir/MEMORY.md",
				"/custom/memdir/debugging.md",
			}, nil
		}

		return nil, nil
	}

	files := externalsources.DiscoverAutoMemory(
		"/home/user/.claude/projects/-some-cwd/memory",
		"",
		settings, dirLister,
	)

	paths := make([]string, 0, len(files))
	for _, file := range files {
		g.Expect(file.Kind).To(Equal(externalsources.KindAutoMemory))
		paths = append(paths, file.Path)
	}

	g.Expect(paths).To(ConsistOf(
		"/custom/memdir/MEMORY.md",
		"/custom/memdir/debugging.md",
	))
}

func TestDiscoverAutoMemory_MainProjectFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := func() (string, bool) { return "", false }

	cwdSlugDir := "/home/user/.claude/projects/-proj-wt/memory"
	mainSlugDir := "/home/user/.claude/projects/-proj/memory"

	dirLister := func(dir string) ([]string, error) {
		switch dir {
		case cwdSlugDir:
			return nil, nil // worktree slug has no dir
		case mainSlugDir:
			return []string{filepath.Join(mainSlugDir, "MEMORY.md")}, nil
		}

		return nil, nil
	}

	files := externalsources.DiscoverAutoMemory(cwdSlugDir, mainSlugDir, settings, dirLister)

	g.Expect(files).To(HaveLen(1))
	g.Expect(files[0].Path).To(Equal(filepath.Join(mainSlugDir, "MEMORY.md")))
}

func TestDiscoverAutoMemory_MainProjectSkippedWhenSameAsCwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := func() (string, bool) { return "", false }

	sameDir := "/home/user/.claude/projects/-proj/memory"
	callCount := 0
	dirLister := func(dir string) ([]string, error) {
		if dir == sameDir {
			callCount++
		}

		return nil, nil
	}

	externalsources.DiscoverAutoMemory(sameDir, sameDir, settings, dirLister)

	// Should only call dirLister once for cwdProjectDir, not again for mainProjectDir
	g.Expect(callCount).To(Equal(1))
}

func TestDiscoverAutoMemory_NoDirReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := func() (string, bool) { return "", false }
	dirLister := func(_ string) ([]string, error) { return nil, nil }

	files := externalsources.DiscoverAutoMemory(
		"/home/user/.claude/projects/-no-such/memory",
		"",
		settings, dirLister,
	)

	g.Expect(files).To(BeEmpty())
}

func TestDiscoverAutoMemory_SkipsNonMarkdownFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	settings := func() (string, bool) { return "", false }

	wantDir := "/home/user/.claude/projects/-proj/memory"
	dirLister := func(dir string) ([]string, error) {
		if dir == wantDir {
			return []string{
				filepath.Join(wantDir, "MEMORY.md"),
				filepath.Join(wantDir, "notes.txt"),
				filepath.Join(wantDir, "config.json"),
			}, nil
		}

		return nil, nil
	}

	files := externalsources.DiscoverAutoMemory(wantDir, "", settings, dirLister)

	g.Expect(files).To(HaveLen(1))
	g.Expect(files[0].Path).To(Equal(filepath.Join(wantDir, "MEMORY.md")))
}
