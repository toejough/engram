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
	// retiredScenarioIDs are the scenarios of 793 that 855 removed and the 170 scenarios of 792a23, 792b38 and 792b39
	// that 868 removed with the development-tooling capability and components. A removed ID is never reused, and their
	// tests are deleted.
	retiredScenarioIDs = []string{
		"793d4a1c", "793d4a1f", "793d4a1h", "793d4a1i", "793d4p1k", "792a23a1a", "792a23a1b",
		"792a23a1c", "792a23b1a", "792a23b1b", "792a23b1c", "792a23c1a", "792a23c1b", "792a23d1a",
		"792a23d1b", "792a23d1c", "792a23e1a", "792a23e1b", "792a23e1c", "792a23f1a", "792a23f1b",
		"792a23f1c", "792a23f1d", "792a23g1a", "792a23g1b", "792a23g1c", "792a23g1d", "792a23h1a",
		"792a23h1b", "792a23i1a", "792a23j1a", "792a23j1b", "792a23k1a", "792a23k1b", "792a23k1c",
		"792a23l1a", "792a23l1b", "792a23m1a", "792a23m1b", "792a23m1c", "792a23m1d", "792a23n1a",
		"792a23n1b", "792a23o1a", "792a23o1b", "792a23o1c", "792a23p1a", "792a23q1a", "792b38c10a1",
		"792b38c10a2", "792b38c10a3", "792b38c10a4", "792b38c11a1", "792b38c11a2", "792b38c11a3", "792b38c11a4",
		"792b38c11a5", "792b38c11a6", "792b38c11a7", "792b38c1a1", "792b38c1a2", "792b38c1a3", "792b38c2a1",
		"792b38c2a10", "792b38c2a11", "792b38c2a12", "792b38c2a2", "792b38c2a3", "792b38c2a4", "792b38c2a5",
		"792b38c2a6", "792b38c2a7", "792b38c2a8", "792b38c2a9", "792b38c3a1", "792b38c3a10", "792b38c3a11",
		"792b38c3a12", "792b38c3a13", "792b38c3a14", "792b38c3a2", "792b38c3a3", "792b38c3a4", "792b38c3a5",
		"792b38c3a6", "792b38c3a7", "792b38c3a8", "792b38c3a9", "792b38c4a1", "792b38c4a2", "792b38c5a1",
		"792b38c5a2", "792b38c5a3", "792b38c5a4", "792b38c5a5", "792b38c5a6", "792b38c5a7", "792b38c5a8",
		"792b38c6a1", "792b38c6a2", "792b38c6a3", "792b38c7a1", "792b38c7a2", "792b38c7a3", "792b38c7a4",
		"792b38c8a1", "792b38c8a10", "792b38c8a2", "792b38c8a3", "792b38c8a4", "792b38c8a5", "792b38c8a6",
		"792b38c8a7", "792b38c8a8", "792b38c8a9", "792b38c9a1", "792b38c9a2", "792b38c9a3", "792b38c9a4",
		"792b38c9a5", "792b38c9a6", "792b39c1a1", "792b39c1a2", "792b39c1a3", "792b39c1a4", "792b39c1a5",
		"792b39c1a6", "792b39c2a1", "792b39c2a2", "792b39c2a3", "792b39c2a4", "792b39c2a5", "792b39c2a6",
		"792b39c2a7", "792b39c2a8", "792b39c2a9", "792b39c3a1", "792b39c3a10", "792b39c3a11", "792b39c3a12",
		"792b39c3a2", "792b39c3a3", "792b39c3a4", "792b39c3a5", "792b39c3a6", "792b39c3a7", "792b39c3a8",
		"792b39c3a9", "792b39c4a1", "792b39c4a2", "792b39c4a3", "792b39c4a4", "792b39c4a5", "792b39c4a6",
		"792b39c4a7", "792b39c4a8", "792b39c5a1", "792b39c5a2", "792b39c5a3", "792b39c5a4", "792b39c5a5",
		"792b39c6a1", "792b39c6a2", "792b39c6a3", "792b39c6a4", "792b39c6a5", "792b39c6a6", "792b39c6a7",
		"792b39c6a8", "792b39c7a1", "792b39c8a1", "792b39c8a2", "792b39c8a3", "792b39c8a4", "792b39c8a5",
	}
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
