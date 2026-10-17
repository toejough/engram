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

func TestConformance_EveryScenarioOfTheSpecsHasATest(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	specs := make([]string, 0, len(specFiles))

	for _, name := range specFiles {
		text, err := os.ReadFile(filepath.Join(repositoryRoot, changeDirectory, name))
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		specs = append(specs, string(text))
	}

	ids := scenarioIDs(strings.Join(specs, "\n"))
	g.Expect(ids).NotTo(BeEmpty(), "the scenarios of the specs")

	g.Expect(scenariosWithoutTest(ids, testSources(t))).To(BeEmpty(), "the scenarios no test is named for")
}

// unexported constants.
const (
	changeDirectory = "docs/changes/793_automatic-memory"
	repositoryRoot  = ".."
)

// unexported variables.
var (
	specFiles = []string{"793a_product-spec.md", "793d_technical-spec.md"}
)

// testSources returns the text of every Go test file of the repository, whatever its build tags.
func testSources(t *testing.T) []string {
	t.Helper()

	var sources []string

	walkErr := filepath.WalkDir(repositoryRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}

		if entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		text, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		sources = append(sources, string(text))

		return nil
	})
	if walkErr != nil {
		t.Fatalf("reading the test files of the repository: %v", walkErr)
	}

	return sources
}
