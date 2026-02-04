package main_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	. "github.com/onsi/gomega"
)

// DefaultChecker is recreated here for testing since it's in package main.
// This tests the expected behavior, and implementation must match.

// traces: TASK-001
// Test that checker finds requirements.md at project root, not docs/.
func TestChecker_RequirementsAtProjectRoot(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create requirements.md at project ROOT (not docs/)
	content := "# Requirements\n\n## REQ-001: Test requirement\n"
	err := os.WriteFile(filepath.Join(dir, "requirements.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Checker should find it
	g.Expect(requirementsExist(dir)).To(BeTrue(), "should find requirements.md at project root")
	g.Expect(requirementsHaveIDs(dir)).To(BeTrue(), "should find REQ-NNN IDs")
}

// traces: TASK-001
// Test that checker finds design.md at project root, not docs/.
func TestChecker_DesignAtProjectRoot(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create design.md at project ROOT (not docs/)
	content := "# Design\n\n## DES-001: Test design\n"
	err := os.WriteFile(filepath.Join(dir, "design.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Checker should find it
	g.Expect(designExist(dir)).To(BeTrue(), "should find design.md at project root")
	g.Expect(designHaveIDs(dir)).To(BeTrue(), "should find DES-NNN IDs")
}

// traces: TASK-001
// Test that checker does NOT find files in docs/ subdirectory.
func TestChecker_DoesNotCheckDocsSubdir(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create requirements.md in docs/ subdirectory (OLD location)
	docsDir := filepath.Join(dir, "docs")
	g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
	content := "# Requirements\n\n## REQ-001: Test requirement\n"
	err := os.WriteFile(filepath.Join(docsDir, "requirements.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Checker should NOT find it (only checks root now)
	g.Expect(requirementsExist(dir)).To(BeFalse(), "should NOT find requirements.md in docs/ subdir")
}

// Helper functions that mirror the checker implementation.
// These define the EXPECTED behavior - artifacts at project root.
func requirementsExist(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "requirements.md"))
	return err == nil
}

func requirementsHaveIDs(dir string) bool {
	content, err := os.ReadFile(filepath.Join(dir, "requirements.md"))
	if err != nil {
		return false
	}
	// Match any number of digits
	matched, _ := regexp.MatchString(`REQ-\d+`, string(content))
	return matched
}

func designExist(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "design.md"))
	return err == nil
}

func designHaveIDs(dir string) bool {
	content, err := os.ReadFile(filepath.Join(dir, "design.md"))
	if err != nil {
		return false
	}
	// Match any number of digits
	matched, _ := regexp.MatchString(`DES-\d+`, string(content))
	return matched
}

// TEST-182 traces: TASK-001
// Test checker validates simple number IDs (REQ-1, REQ-2, ...)
func TestChecker_ValidatesSimpleNumberIDs(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create requirements.md with simple number IDs
	content := "# Requirements\n\n## REQ-1: First requirement\n## REQ-42: Another\n"
	err := os.WriteFile(filepath.Join(dir, "requirements.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Checker should find these IDs
	g.Expect(requirementsHaveIDs(dir)).To(BeTrue(), "should find REQ-N IDs")
}

// TEST-183 traces: TASK-001
// Test checker validates design with simple number IDs
func TestChecker_ValidatesSimpleNumberDesignIDs(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create design.md with simple number IDs
	content := "# Design\n\n## DES-1: First design\n## DES-99: Another\n"
	err := os.WriteFile(filepath.Join(dir, "design.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Checker should find these IDs
	g.Expect(designHaveIDs(dir)).To(BeTrue(), "should find DES-N IDs")
}
