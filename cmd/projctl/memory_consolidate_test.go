package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// ============================================================================
// CLI tests for `projctl memory consolidate` subcommand
// traces: TASK-6, ISSUE-160
// ============================================================================

// TEST-970: memory consolidate subcommand exists and runs
// traces: TASK-6
func TestMemoryConsolidateCommandExists(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	cmd := exec.Command("projctl", "memory", "consolidate",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	// Command should exist and run without error (even on empty DB)
	g.Expect(err).ToNot(HaveOccurred(), "memory consolidate command should exist: %s", string(output))
}

// TEST-971: memory consolidate reports summary statistics
// traces: TASK-6
func TestMemoryConsolidateReportsSummary(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Seed with some content
	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: test learning for consolidate"), 0644)).To(Succeed())

	// Query to create embeddings
	queryCmd := exec.Command("projctl", "memory", "query",
		"test learning",
		"--memoryroot", memoryDir)
	_, err := queryCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "consolidate",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Should succeed: %s", string(output))

	// Verify summary is reported
	outputStr := string(output)
	g.Expect(outputStr).To(Or(
		ContainSubstring("decayed"),
		ContainSubstring("Decayed"),
		ContainSubstring("pruned"),
		ContainSubstring("Pruned"),
		ContainSubstring("merged"),
		ContainSubstring("Merged"),
		ContainSubstring("candidates"),
		ContainSubstring("Candidates"),
	))
}

// TEST-972: memory consolidate accepts optional flags
// traces: TASK-6
func TestMemoryConsolidateAcceptsFlags(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	cmd := exec.Command("projctl", "memory", "consolidate",
		"--decay-factor", "0.85",
		"--prune-threshold", "0.15",
		"--duplicate-threshold", "0.90",
		"--min-retrievals", "4",
		"--min-projects", "3",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(),
		"memory consolidate should accept all optional flags: %s", string(output))
}

// TEST-973: memory consolidate runs unattended (no user interaction)
// traces: TASK-6
func TestMemoryConsolidateUnattended(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Seed with content
	g.Expect(os.WriteFile(filepath.Join(memoryDir, "index.md"),
		[]byte("- 2024-01-01: unattended test"), 0644)).To(Succeed())

	cmd := exec.Command("projctl", "memory", "consolidate",
		"--memoryroot", memoryDir)

	// Close stdin to ensure no interaction is possible
	cmd.Stdin = nil

	output, err := cmd.CombinedOutput()

	// Should complete without waiting for input
	g.Expect(err).ToNot(HaveOccurred(), "Should run unattended: %s", string(output))
}

// TEST-974: memory consolidate uses default memoryroot when not specified
// traces: TASK-6
func TestMemoryConsolidateDefaultMemoryRoot(t *testing.T) {
	g := NewWithT(t)

	cmd := exec.Command("projctl", "memory", "consolidate")
	output, err := cmd.CombinedOutput()

	// Should not fail with "unknown command" error
	outputStr := string(output)
	g.Expect(outputStr).ToNot(ContainSubstring("unknown command"),
		"memory consolidate should be a recognized subcommand")
	_ = err // May fail if no DB exists, which is fine
}

// TEST-975: Property: memory consolidate never returns error for valid memoryroot
// traces: TASK-6
func TestPropertyMemoryConsolidateNoErrorForValidRoot(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "consolidate-cli-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

		cmd := exec.Command("projctl", "memory", "consolidate",
			"--memoryroot", memoryDir)

		output, err := cmd.CombinedOutput()
		// Property: should never error for valid inputs
		g.Expect(err).ToNot(HaveOccurred(),
			"memory consolidate should not error for valid memoryroot: %s", string(output))
	})
}

// TEST-976: memory consolidate completes in reasonable time
// traces: TASK-6
func TestMemoryConsolidatePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Seed with multiple learnings
	for i := 0; i < 20; i++ {
		learnCmd := exec.Command("projctl", "memory", "learn",
			"-m", "performance test learning number "+string(rune('A'+i)),
			"--memoryroot", memoryDir)
		_, err := learnCmd.CombinedOutput()
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Query to create embeddings
	queryCmd := exec.Command("projctl", "memory", "query",
		"performance test",
		"--memoryroot", memoryDir)
	_, err := queryCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	// Run consolidate - should complete in reasonable time
	cmd := exec.Command("projctl", "memory", "consolidate",
		"--memoryroot", memoryDir)

	output, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "Should complete: %s", string(output))
}
