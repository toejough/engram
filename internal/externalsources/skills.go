package externalsources

import (
	"path/filepath"
)

// SkillFinder returns absolute paths to every SKILL.md found anywhere under
// root (recursive). Wired at the edge to filepath.WalkDir; replaced by a
// fake in tests.
type SkillFinder func(root string) []string

// DiscoverSkills returns SKILL.md files from the three documented locations:
// project, user, and plugin cache. Each root is walked independently; an
// empty root simply contributes nothing.
func DiscoverSkills(cwd, home string, finder SkillFinder) []ExternalFile {
	roots := []string{filepath.Join(cwd, ".claude", "skills")}

	if home != "" {
		roots = append(roots,
			filepath.Join(home, ".claude", "skills"),
			filepath.Join(home, ".claude", "plugins", "cache"),
		)
	}

	files := make([]ExternalFile, 0, defaultSkillsCapacity)

	for _, root := range roots {
		for _, path := range finder(root) {
			files = append(files, ExternalFile{Kind: KindSkill, Path: path})
		}
	}

	return files
}

// unexported constants.
const (
	defaultSkillsCapacity = 32
)
