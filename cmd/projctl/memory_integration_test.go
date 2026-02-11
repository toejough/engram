//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

// TestMemoryLearnCommand tests the CLI interface for projctl memory learn
func TestMemoryLearnCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	// Run the memory learn command
	cmd := exec.Command("projctl", "memory", "learn",
		"--message", "Test learning from CLI",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Learned"))

	// Verify content was stored in embeddings DB via grep
	grepCmd := exec.Command("projctl", "memory", "grep",
		"Test learning from CLI",
		"--memoryroot", memoryDir)
	grepOutput, err := grepCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "Grep should succeed: %s", string(grepOutput))
	g.Expect(string(grepOutput)).To(ContainSubstring("Test learning from CLI"))
}

// TestMemoryLearnWithProject tests the --project flag
func TestMemoryLearnWithProject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	cmd := exec.Command("projctl", "memory", "learn",
		"--message", "Project-specific learning",
		"--project", "my-project",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))

	// Verify content was stored in embeddings DB
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	_, err = os.Stat(dbPath)
	g.Expect(err).ToNot(HaveOccurred(), "embeddings.db should be created")
}

// TestMemoryLearnRequiresMessage tests that --message is required
func TestMemoryLearnRequiresMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	cmd := exec.Command("projctl", "memory", "learn",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred(), "Should fail without message")
	g.Expect(string(output)).To(ContainSubstring("message"))
}

// TestMemoryLearnMultipleEntries tests appending multiple learnings
func TestMemoryLearnMultipleEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	// Use Go API directly instead of spawning CLI processes
	// This avoids cross-process SQLite issues and makes debugging easier
	err := memory.Learn(memory.LearnOpts{
		Message:    "First CLI learning",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Second CLI learning",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify both are in the DB via grep
	for _, msg := range []string{"First CLI learning", "Second CLI learning"} {
		result, grepErr := memory.Grep(memory.GrepOpts{
			Pattern:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(grepErr).ToNot(HaveOccurred())
		g.Expect(len(result.Matches)).To(BeNumerically(">", 0), "Message %q not found", msg)
	}
}

// TestMemoryDecideCommand tests the CLI interface for projctl memory decide
func TestMemoryDecideCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	cmd := exec.Command("projctl", "memory", "decide",
		"--context", "Test decision context",
		"--choice", "Option A",
		"--reason", "Best option for testing",
		"--alternatives", "Option B, Option C",
		"--project", "test-project",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Decision logged"))
}

// TestMemoryDecideRequiresFields tests that required fields are enforced
func TestMemoryDecideRequiresFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	// Missing context
	cmd := exec.Command("projctl", "memory", "decide",
		"--choice", "Option A",
		"--reason", "Test reason",
		"--project", "test",
		"--memoryroot", memoryDir)
	_, err := cmd.CombinedOutput()
	g.Expect(err).To(HaveOccurred(), "Should fail without context")
}

// ============================================================================
// TASK-42: External knowledge capture - CLI --source flag
// ============================================================================

// TestMemoryLearnWithSourceInternal tests --source internal flag
func TestMemoryLearnWithSourceInternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	cmd := exec.Command("projctl", "memory", "learn",
		"--message", "Internal source learning",
		"--source", "internal",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Learned"))
}

// TestMemoryLearnWithSourceExternal tests --source external flag
func TestMemoryLearnWithSourceExternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	cmd := exec.Command("projctl", "memory", "learn",
		"--message", "External source learning from web",
		"--source", "external",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Learned"))
}

// TestMemoryLearnSourceDefaultsToInternal tests that omitting --source defaults to internal
func TestMemoryLearnSourceDefaultsToInternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	// Run without --source flag
	cmd := exec.Command("projctl", "memory", "learn",
		"--message", "No source specified",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Learned"))
}

// TestMemoryLearnSourceWithProject tests --source combined with --project
func TestMemoryLearnSourceWithProject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	cmd := exec.Command("projctl", "memory", "learn",
		"--message", "External learning with project",
		"--source", "external",
		"--project", "my-project",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Learned"))
}
