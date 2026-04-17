package externalsources

import (
	"path/filepath"
)

// GlobMatcher reports whether at least one path under cwd matches any of the
// given globs. Wired at the edge to a small filepath.Glob loop; replaced by
// a fake in tests.
type GlobMatcher func(globs []string) (anyMatch bool)

// MdWalker returns absolute paths to all *.md files found recursively under
// root. Wired at the edge to filepath.WalkDir; replaced by a fake in tests.
type MdWalker func(root string) []string

// DiscoverRules walks <project>/.claude/rules and ~/.claude/rules collecting
// markdown rule files. Files with a `paths:` frontmatter are included only
// when at least one file under cwd matches the glob (per the engram-specific
// adaptation noted in the spec).
func DiscoverRules(
	cwd, home string,
	walker MdWalker,
	reader ReaderFunc,
	matchAny GlobMatcher,
) []ExternalFile {
	files := make([]ExternalFile, 0, defaultRulesCapacity)

	roots := []string{filepath.Join(cwd, ".claude", "rules")}
	if home != "" {
		roots = append(roots, filepath.Join(home, ".claude", "rules"))
	}

	for _, root := range roots {
		for _, mdPath := range walker(root) {
			if shouldIncludeRule(mdPath, reader, matchAny) {
				files = append(files, ExternalFile{Kind: KindRules, Path: mdPath})
			}
		}
	}

	return files
}

// unexported constants.
const (
	defaultRulesCapacity = 8
)

func shouldIncludeRule(path string, reader ReaderFunc, matchAny GlobMatcher) bool {
	body, err := reader(path)
	if err != nil || body == nil {
		return false
	}

	fm, _ := ParseFrontmatter(body)
	if len(fm.Paths) == 0 {
		return true
	}

	return matchAny(fm.Paths)
}
