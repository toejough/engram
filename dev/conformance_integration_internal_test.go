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

	ids := scenarioIDs(canonicalSpecs(t))
	g.Expect(ids).NotTo(BeEmpty(), "the scenarios of the specs")

	g.Expect(scenariosWithoutTest(ids, testSources(t))).To(BeEmpty(), "the scenarios no test is named for")
}

func TestConformance_TheCanonicalSpecsHoldTheScenariosOfBothChanges(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ids := scenarioIDs(canonicalSpecs(t))

	g.Expect(countWithPrefix(ids, "793")).To(Equal(expected793Scenarios),
		"the scenarios of 793 in the canonical specs: a promoted spec lost or gained one")
	g.Expect(countWithPrefix(ids, "855")).To(Equal(expected855Scenarios),
		"the scenarios of 855 in the canonical specs: a promoted spec lost or gained one")

	for _, retired := range retiredScenarioIDs {
		g.Expect(ids).NotTo(ContainElement(retired), "the retired scenario "+retired+" is in a spec again")
	}
}

// unexported constants.
const (
	// expected793Scenarios and expected855Scenarios are how many distinct scenario IDs rooted at 793 and at 855 the
	// canonical specs hold; a spec that gains or loses one changes them.
	expected793Scenarios   = 178
	expected855Scenarios   = 133
	productSpecDirectory   = "docs/specs/product"
	repositoryRoot         = ".."
	technicalSpecDirectory = "docs/specs/technical"
)

// unexported variables.
var (
	// retiredScenarioIDs are the scenarios of 793 that 855 removed. A removed ID is never reused, and their tests are
	// deleted.
	retiredScenarioIDs = []string{"793d4a1c", "793d4a1f", "793d4a1h", "793d4a1i", "793d4p1k"}
)

// canonicalSpecs returns the text of every Markdown file of the canonical product and technical specs, joined by a
// newline.
func canonicalSpecs(t *testing.T) string {
	t.Helper()

	var texts []string

	for _, directory := range []string{productSpecDirectory, technicalSpecDirectory} {
		entries, err := os.ReadDir(filepath.Join(repositoryRoot, directory))
		if err != nil {
			t.Fatalf("reading the directory %s: %v", directory, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			text, readErr := os.ReadFile(filepath.Join(repositoryRoot, directory, entry.Name()))
			if readErr != nil {
				t.Fatalf("reading the spec %s: %v", entry.Name(), readErr)
			}

			texts = append(texts, string(text))
		}
	}

	return strings.Join(texts, "\n")
}

// countWithPrefix returns how many of the IDs begin with the prefix.
func countWithPrefix(ids []string, prefix string) int {
	count := 0

	for _, id := range ids {
		if strings.HasPrefix(id, prefix) {
			count++
		}
	}

	return count
}

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
