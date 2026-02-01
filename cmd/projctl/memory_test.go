package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// TestMemoryLearnCommand tests the CLI interface for projctl memory learn
func TestMemoryLearnCommand(t *testing.T) {
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

	// Verify file was created
	indexPath := filepath.Join(memoryDir, "index.md")
	content, err := os.ReadFile(indexPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("Test learning from CLI"))
}

// TestMemoryLearnWithProject tests the --project flag
func TestMemoryLearnWithProject(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	cmd := exec.Command("projctl", "memory", "learn",
		"--message", "Project-specific learning",
		"--project", "my-project",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))

	indexPath := filepath.Join(memoryDir, "index.md")
	content, err := os.ReadFile(indexPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("[my-project]"))
}

// TestMemoryLearnRequiresMessage tests that --message is required
func TestMemoryLearnRequiresMessage(t *testing.T) {
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
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	// Add first learning
	cmd := exec.Command("projctl", "memory", "learn",
		"--message", "First CLI learning",
		"--memoryroot", memoryDir)
	_, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	// Add second learning
	cmd = exec.Command("projctl", "memory", "learn",
		"--message", "Second CLI learning",
		"--memoryroot", memoryDir)
	_, err = cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	// Verify both are in the file
	indexPath := filepath.Join(memoryDir, "index.md")
	content, err := os.ReadFile(indexPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("First CLI learning"))
	g.Expect(string(content)).To(ContainSubstring("Second CLI learning"))
}
