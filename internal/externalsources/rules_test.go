package externalsources_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscoverRules_FilesWithoutPathsAlwaysIncluded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mdContent := []byte("# Always-on rule\nNo frontmatter.\n")

	contents := map[string][]byte{
		"/proj/.claude/rules/always.md": mdContent,
	}

	walker := func(root string) []string {
		if root == "/proj/.claude/rules" {
			return []string{"/proj/.claude/rules/always.md"}
		}

		return nil
	}

	matchAny := func(_ []string) bool { return false }

	files := externalsources.DiscoverRules("/proj", "/home/user", walker, fakeReader(contents), matchAny)

	paths := make([]string, 0, len(files))
	for _, file := range files {
		g.Expect(file.Kind).To(Equal(externalsources.KindRules))
		paths = append(paths, file.Path)
	}

	g.Expect(paths).To(ConsistOf("/proj/.claude/rules/always.md"))
}

func TestDiscoverRules_PathsGlobExcludedWhenNoFilesMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/proj/.claude/rules/api.md": []byte(`---
paths:
  - "src/api/**/*.ts"
---
API rule
`),
	}

	walker := func(_ string) []string { return []string{"/proj/.claude/rules/api.md"} }
	matchAny := func(_ []string) bool { return false }

	files := externalsources.DiscoverRules("/proj", "/home/user", walker, fakeReader(contents), matchAny)
	g.Expect(files).To(BeEmpty())
}

func TestDiscoverRules_PathsGlobIncludedWhenAtLeastOneFileMatches(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	contents := map[string][]byte{
		"/proj/.claude/rules/api.md": []byte(`---
paths:
  - "src/api/**/*.ts"
---
API rule
`),
	}

	walker := func(root string) []string {
		if root == "/proj/.claude/rules" {
			return []string{"/proj/.claude/rules/api.md"}
		}

		return nil
	}

	matchAny := func(globs []string) bool {
		return len(globs) > 0
	}

	files := externalsources.DiscoverRules("/proj", "/home/user", walker, fakeReader(contents), matchAny)
	g.Expect(files).To(HaveLen(1))
	g.Expect(files[0].Path).To(Equal(filepath.FromSlash("/proj/.claude/rules/api.md")))
}

func TestDiscoverRules_ReadErrorSkipsFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	walker := func(_ string) []string { return []string{"/missing.md"} }
	reader := func(_ string) ([]byte, error) { return nil, nil }
	matchAny := func(_ []string) bool { return false }

	files := externalsources.DiscoverRules("/proj", "", walker, reader, matchAny)
	g.Expect(files).To(BeEmpty())
}
