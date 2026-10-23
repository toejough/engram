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

func TestConformance_EveryAllowlistEntryIsCurrent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ids := scenarioIDs(canonicalSpecs(t))

	g.Expect(staleAllowlistEntries(ids, testSources(t), unverifiedScenarios())).To(BeEmpty(),
		"the allowlisted scenarios that a spec no longer holds or a test now covers: remove their entries")
}

func TestConformance_EveryAllowlistEntryNamesItsReason(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(malformedAllowlistEntries(unverifiedScenarios())).To(BeEmpty(),
		"the allowlist entries with no decision row or issue in their reason, out of order or listed twice")
}

func TestConformance_EveryScenarioOfTheSpecsHasATest(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ids := scenarioIDs(canonicalSpecs(t))
	g.Expect(ids).NotTo(BeEmpty(), "the scenarios of the specs")

	g.Expect(scenariosNeedingATest(ids, testSources(t), unverifiedScenarios())).To(BeEmpty(),
		"the scenarios no test is named for and no allowlist entry rules out")
}

func TestConformance_TheRetiredScenariosStayOutOfTheSpecs(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ids := scenarioIDs(canonicalSpecs(t))

	for _, retired := range retiredScenarioIDs {
		g.Expect(ids).NotTo(ContainElement(retired), "the retired scenario "+retired+" is in a spec again")
	}
}

// unexported constants.
const (
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

// isOtherCheckout tells whether a directory is the repository's own metadata, a place where worktrees are kept, or the
// root of another checkout (a directory with a .git entry): the test files in those are not the repository's own.
func isOtherCheckout(path, name string) bool {
	if name == ".git" || name == ".claude" || name == ".worktrees" {
		return true
	}

	if path == repositoryRoot {
		return false
	}

	_, statErr := os.Stat(filepath.Join(path, ".git"))

	return statErr == nil
}

// testSources returns the text of every Go test file of the repository, whatever its build tags.
func testSources(t *testing.T) []string {
	t.Helper()

	var sources []string

	walkErr := filepath.WalkDir(repositoryRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() && isOtherCheckout(path, entry.Name()) {
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
