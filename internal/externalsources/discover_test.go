package externalsources_test

import (
	"runtime"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscover_AggregatesEachSourceKind(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := externalsources.DiscoverDeps{
		CWD:            "/proj",
		Home:           "/home/user",
		GOOS:           "darwin",
		CWDProjectDir:  "/mem",
		MainProjectDir: "",
		StatFn: func(p string) (bool, error) {
			return p == "/proj/CLAUDE.md", nil
		},
		Reader: func(p string) ([]byte, error) {
			if p == "/proj/CLAUDE.md" {
				return []byte("# project rules\n"), nil
			}

			if p == "/proj/.claude/rules/code.md" {
				return []byte("# code rules\nNo frontmatter, always included.\n"), nil
			}

			return nil, nil
		},
		MdWalker: func(root string) []string {
			if root == "/proj/.claude/rules" {
				return []string{"/proj/.claude/rules/code.md"}
			}

			return nil
		},
		MatchAny: func(_ []string) bool { return false },
		Settings: func() (string, bool) { return "", false },
		DirLister: func(dir string) ([]string, error) {
			if dir == "/mem" {
				return []string{"/mem/MEMORY.md"}, nil
			}

			return nil, nil
		},
		SkillFinder: func(root string) []string {
			if root == "/proj/.claude/skills" {
				return []string{"/proj/.claude/skills/test/SKILL.md"}
			}

			return nil
		},
	}

	files := externalsources.Discover(deps)

	kinds := make(map[externalsources.Kind]int)
	for _, file := range files {
		kinds[file.Kind]++
	}

	g.Expect(kinds[externalsources.KindClaudeMd]).To(BeNumerically(">=", 1))
	g.Expect(kinds[externalsources.KindRules]).To(BeNumerically(">=", 1))
	g.Expect(kinds[externalsources.KindAutoMemory]).To(BeNumerically(">=", 1))
	g.Expect(kinds[externalsources.KindSkill]).To(BeNumerically(">=", 1))
}

func TestDiscover_CombinesAllSourceKinds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := externalsources.DiscoverDeps{
		CWD:            "/proj",
		Home:           "/home/user",
		GOOS:           runtime.GOOS,
		CWDProjectDir:  "/home/user/.claude/projects/-proj/memory",
		MainProjectDir: "",
		StatFn:         func(_ string) (bool, error) { return true, nil },
		Reader:         func(_ string) ([]byte, error) { return nil, nil },
		MdWalker:       func(_ string) []string { return nil },
		MatchAny:       func(_ []string) bool { return false },
		Settings:       func() (string, bool) { return "", false },
		DirLister:      func(_ string) ([]string, error) { return nil, nil },
		SkillFinder:    func(_ string) []string { return nil },
	}

	files := externalsources.Discover(deps)
	g.Expect(files).NotTo(BeNil())
}
