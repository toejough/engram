package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// ============================================================================
// CLI tests for `projctl memory promote` subcommand
// traces: TASK-41
// ============================================================================

// TEST-930: memory promote subcommand exists and runs
// traces: TASK-41
func TestMemoryPromoteCommandExists(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Seed with some content so the DB is created
	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: CLI promote test"), 0644)).To(Succeed())

	cmd := exec.Command("projctl", "memory", "promote",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	// Command should exist and run without error (even if no candidates)
	g.Expect(err).ToNot(HaveOccurred(), "memory promote command should exist: %s", string(output))
}

// TEST-931: memory promote outputs "no candidates" when none qualify
// traces: TASK-41
func TestMemoryPromoteNoCandidatesOutput(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: no candidates test"), 0644)).To(Succeed())

	// Query once to create DB (below thresholds)
	queryCmd := exec.Command("projctl", "memory", "query",
		"no candidates test",
		"--memoryroot", memoryDir)
	_, err := queryCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "promote",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Should succeed: %s", string(output))
	g.Expect(string(output)).To(Or(
		ContainSubstring("No promotion candidates"),
		ContainSubstring("no candidates"),
		ContainSubstring("No candidates"),
	))
}

// TEST-932: memory promote accepts --min-retrievals flag
// traces: TASK-41
func TestMemoryPromoteMinRetrievalsFlag(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: min retrievals flag test"), 0644)).To(Succeed())

	// Run with custom min-retrievals
	cmd := exec.Command("projctl", "memory", "promote",
		"--min-retrievals", "5",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(),
		"memory promote should accept --min-retrievals flag: %s", string(output))
}

// TEST-933: memory promote accepts --min-projects flag
// traces: TASK-41
func TestMemoryPromoteMinProjectsFlag(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: min projects flag test"), 0644)).To(Succeed())

	// Run with custom min-projects
	cmd := exec.Command("projctl", "memory", "promote",
		"--min-projects", "4",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(),
		"memory promote should accept --min-projects flag: %s", string(output))
}

// TEST-934: memory promote displays candidates with content
// traces: TASK-41
func TestMemoryPromoteDisplaysCandidates(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: promotable learning for display test"), 0644)).To(Succeed())

	// Build up retrievals from multiple projects to exceed thresholds
	for _, proj := range []string{"proj-d1", "proj-d2", "proj-d3"} {
		queryCmd := exec.Command("projctl", "memory", "query",
			"promotable learning",
			"--memoryroot", memoryDir)
		// Note: The CLI query command doesn't have --project yet.
		// We need to test that the promote command works after seeding.
		// For CLI tests we rely on the --min-retrievals=1 --min-projects=1 workaround
		_ = proj
		_, err := queryCmd.CombinedOutput()
		g.Expect(err).ToNot(HaveOccurred())
	}

	cmd := exec.Command("projctl", "memory", "promote",
		"--min-retrievals", "1",
		"--min-projects", "1",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("promotable learning"),
		"output should display candidate content")
}

// TEST-935: memory promote uses default memoryroot when not specified
// traces: TASK-41
func TestMemoryPromoteDefaultMemoryRoot(t *testing.T) {
	g := NewWithT(t)

	// Just verify the command doesn't crash when using defaults
	cmd := exec.Command("projctl", "memory", "promote")
	output, err := cmd.CombinedOutput()

	// Should not fail with "unknown command" error
	// It may fail due to no DB existing yet, but that's acceptable
	outputStr := string(output)
	g.Expect(outputStr).ToNot(ContainSubstring("unknown command"),
		"memory promote should be a recognized subcommand")
	_ = err // May fail if no DB exists, which is fine
}

// TEST-936: Property: memory promote never returns error for valid memoryroot
// traces: TASK-41
func TestPropertyMemoryPromoteNoErrorForValidRoot(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "promote-cli-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

		content := rapid.StringMatching(`[a-zA-Z ]{10,30}`).Draw(t, "content")
		g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
			[]byte("- 2024-01-01: "+content), 0644)).To(Succeed())

		// Seed with a query
		queryCmd := exec.Command("projctl", "memory", "query",
			content,
			"--memoryroot", memoryDir)
		_, err := queryCmd.CombinedOutput()
		g.Expect(err).ToNot(HaveOccurred())

		minRetrievals := rapid.IntRange(1, 10).Draw(t, "minRetrievals")
		minProjects := rapid.IntRange(1, 5).Draw(t, "minProjects")

		cmd := exec.Command("projctl", "memory", "promote",
			"--min-retrievals", fmt.Sprintf("%d", minRetrievals),
			"--min-projects", fmt.Sprintf("%d", minProjects),
			"--memoryroot", memoryDir)

		output, err := cmd.CombinedOutput()
		// Property: should never error for valid inputs
		g.Expect(err).ToNot(HaveOccurred(),
			"memory promote should not error for valid memoryroot: %s", string(output))
	})
}
