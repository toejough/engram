//go:build targ && integration

package dev

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// TestMods_EveryJavaScriptModuleParses: the real mods parse under the real node, and the real node rejects a broken
// module, which the stand-in node of the unit tests cannot show (793b55).
func TestMods_EveryJavaScriptModuleParses(t *testing.T) {
	t.Parallel()

	t.Run("the mods of the source", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		modules := modulesOf(t, filepath.Join(repositoryRoot, "agent-instructions", "mods"))

		g.Expect(modules).NotTo(BeEmpty(), "the JavaScript of the mods")
		g.Expect(checkModuleSyntax(t.Context(), modules, findNode, parseModule)).To(Succeed())
	})

	t.Run("a broken module", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		err := checkModuleSyntax(t.Context(),
			map[string]string{"hooks/broken.js": "export function register( {\n"}, findNode, parseModule)

		g.Expect(err).To(MatchError(ContainSubstring("hooks/broken.js")))
		g.Expect(err).To(MatchError(ContainSubstring("SyntaxError")))
	})
}

// modulesOf reads every JavaScript file below a directory, by its path relative to the directory.
func modulesOf(t *testing.T, dir string) map[string]string {
	t.Helper()

	modules := map[string]string{}

	walkErr := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".js") {
			return err
		}

		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		relative, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}

		modules[relative] = string(source)

		return nil
	})
	if walkErr != nil {
		t.Fatalf("reading the JavaScript of %s: %v", dir, walkErr)
	}

	return modules
}
