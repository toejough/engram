package externalsources_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/externalsources"
)

func TestDiscoverSkills_AllThreeRoots(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cwdSkill := filepath.Join("/proj", ".claude", "skills", "projlocal", "SKILL.md")
	userSkill := filepath.Join("/home/user", ".claude", "skills", "personal", "SKILL.md")
	pluginSkill := filepath.Join(
		"/home/user", ".claude", "plugins", "cache", "core", "1.0.0",
		"skills", "core-skill", "SKILL.md",
	)

	skillFinder := func(root string) []string {
		switch root {
		case filepath.Join("/proj", ".claude", "skills"):
			return []string{cwdSkill}
		case filepath.Join("/home/user", ".claude", "skills"):
			return []string{userSkill}
		case filepath.Join("/home/user", ".claude", "plugins", "cache"):
			return []string{pluginSkill}
		}

		return nil
	}

	files := externalsources.DiscoverSkills("/proj", "/home/user", skillFinder)

	paths := make([]string, 0, len(files))
	for _, f := range files {
		g.Expect(f.Kind).To(Equal(externalsources.KindSkill))
		paths = append(paths, f.Path)
	}

	g.Expect(paths).To(ConsistOf(cwdSkill, userSkill, pluginSkill))
}

func TestDiscoverSkills_EmptyHomeSkipsUserAndPluginRoots(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	roots := make([]string, 0)
	skillFinder := func(root string) []string {
		roots = append(roots, root)
		return nil
	}

	externalsources.DiscoverSkills("/proj", "", skillFinder)

	g.Expect(roots).To(ConsistOf(filepath.Join("/proj", ".claude", "skills")))
}

func TestDiscoverSkills_NoneFoundReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := externalsources.DiscoverSkills("/proj", "/home/user", func(_ string) []string { return nil })
	g.Expect(files).To(BeEmpty())
}
