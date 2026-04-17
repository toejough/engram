package externalsources_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscover_AgainstFixtureProject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fixtureRoot, err := filepath.Abs("testdata/fixture-project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	cache := externalsources.NewFileCache(os.ReadFile)

	autoMemDir := filepath.Join(fixtureRoot, "auto-memory")

	deps := externalsources.DiscoverDeps{
		CWD:            fixtureRoot,
		Home:           "/nonexistent",
		GOOS:           runtime.GOOS,
		CWDProjectDir:  autoMemDir,
		MainProjectDir: "",
		StatFn: func(path string) (bool, error) {
			_, statErr := os.Stat(path)
			return statErr == nil, nil
		},
		Reader:      cache.Read,
		MdWalker:    walkMdInTest,
		MatchAny:    func(_ []string) bool { return false },
		Settings:    func() (string, bool) { return autoMemDir, true },
		DirLister:   listDirEntriesInTest,
		SkillFinder: walkSkillsInTest,
	}

	files := externalsources.Discover(deps)

	kinds := make(map[externalsources.Kind]int)
	for _, file := range files {
		kinds[file.Kind]++
	}

	g.Expect(kinds[externalsources.KindClaudeMd]).To(BeNumerically(">=", 1),
		"expected at least one CLAUDE.md")
	g.Expect(kinds[externalsources.KindRules]).To(BeNumerically(">=", 1),
		"expected at least one rules file")
	g.Expect(kinds[externalsources.KindAutoMemory]).To(BeNumerically(">=", 2),
		"expected MEMORY.md + topic.md")
	g.Expect(kinds[externalsources.KindSkill]).To(BeNumerically(">=", 1),
		"expected at least one SKILL.md")
}

// listDirEntriesInTest returns absolute paths to entries in dir (non-recursive).
func listDirEntriesInTest(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil //nolint:nilerr // missing dir is normal
	}

	out := make([]string, 0, len(entries))

	for _, entry := range entries {
		out = append(out, filepath.Join(dir, entry.Name()))
	}

	return out, nil
}

// walkMdInTest walks root recursively returning all *.md file paths.
func walkMdInTest(root string) []string {
	out := make([]string, 0)

	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return nil //nolint:nilerr // skip unreadable subtrees
		}

		if filepath.Ext(entry.Name()) == ".md" {
			out = append(out, path)
		}

		return nil
	})

	return out
}

// walkSkillsInTest walks root returning all SKILL.md file paths.
func walkSkillsInTest(root string) []string {
	out := make([]string, 0)

	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return nil //nolint:nilerr // skip unreadable subtrees
		}

		if entry.Name() == "SKILL.md" {
			out = append(out, path)
		}

		return nil
	})

	return out
}
