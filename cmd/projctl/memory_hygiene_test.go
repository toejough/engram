package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// ============================================================================
// CLI decay command tests (TASK-43)
// ============================================================================

// TEST-990: memory decay command runs successfully with defaults
// traces: TASK-43
func TestMemoryDecayCommand(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store a learning first
	learnCmd := exec.Command("projctl", "memory", "learn",
		"--message", "Learning for decay CLI test",
		"--memoryroot", memoryDir)
	output, err := learnCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "Learn should succeed: %s", string(output))

	// Run decay
	cmd := exec.Command("projctl", "memory", "decay",
		"--memoryroot", memoryDir)
	output, err = cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Decay command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Decayed"))
}

// TEST-991: memory decay command accepts --factor flag
// traces: TASK-43
func TestMemoryDecayWithFactorFlag(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store a learning first
	learnCmd := exec.Command("projctl", "memory", "learn",
		"--message", "Learning for factor test",
		"--memoryroot", memoryDir)
	_, err = learnCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	// Run decay with custom factor
	cmd := exec.Command("projctl", "memory", "decay",
		"--factor", "0.5",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Decay with factor should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("0.5"))
}

// TEST-992: memory decay command shows affected entry count
// traces: TASK-43
func TestMemoryDecayShowsAffectedCount(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store multiple learnings
	for _, msg := range []string{"first learning", "second learning", "third learning"} {
		learnCmd := exec.Command("projctl", "memory", "learn",
			"--message", msg,
			"--memoryroot", memoryDir)
		_, err = learnCmd.CombinedOutput()
		g.Expect(err).ToNot(HaveOccurred())
	}

	cmd := exec.Command("projctl", "memory", "decay",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Decay should succeed: %s", string(output))
	// Output should mention how many entries were affected
	g.Expect(string(output)).To(MatchRegexp(`\d+ entr`))
}

// ============================================================================
// CLI prune command tests (TASK-43)
// ============================================================================

// TEST-993: memory prune command runs successfully with defaults
// traces: TASK-43
func TestMemoryPruneCommand(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store a learning and decay it below threshold
	learnCmd := exec.Command("projctl", "memory", "learn",
		"--message", "Learning for prune CLI test",
		"--memoryroot", memoryDir)
	_, err = learnCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	// Decay to push below default threshold
	decayCmd := exec.Command("projctl", "memory", "decay",
		"--factor", "0.05",
		"--memoryroot", memoryDir)
	_, err = decayCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	// Run prune
	cmd := exec.Command("projctl", "memory", "prune",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Prune command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Pruned"))
}

// TEST-994: memory prune command accepts --threshold flag
// traces: TASK-43
func TestMemoryPruneWithThresholdFlag(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store a learning and decay it
	learnCmd := exec.Command("projctl", "memory", "learn",
		"--message", "Learning for threshold test",
		"--memoryroot", memoryDir)
	_, err = learnCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	decayCmd := exec.Command("projctl", "memory", "decay",
		"--factor", "0.5",
		"--memoryroot", memoryDir)
	_, err = decayCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	// Prune with high threshold to catch the decayed entry
	cmd := exec.Command("projctl", "memory", "prune",
		"--threshold", "0.6",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Prune with threshold should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("0.6"))
}

// TEST-995: memory prune command shows removed and retained counts
// traces: TASK-43
func TestMemoryPruneShowsCounts(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store learning, decay it below threshold
	learnCmd := exec.Command("projctl", "memory", "learn",
		"--message", "Will be pruned",
		"--memoryroot", memoryDir)
	_, err = learnCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	decayCmd := exec.Command("projctl", "memory", "decay",
		"--factor", "0.05",
		"--memoryroot", memoryDir)
	_, err = decayCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	// Add a fresh learning (stays at 1.0)
	learnCmd2 := exec.Command("projctl", "memory", "learn",
		"--message", "Will be retained",
		"--memoryroot", memoryDir)
	_, err = learnCmd2.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "prune",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Prune should succeed: %s", string(output))
	// Output should include both removed and retained counts
	g.Expect(string(output)).To(MatchRegexp(`removed.*\d+`))
	g.Expect(string(output)).To(MatchRegexp(`retained.*\d+`))
}

// TEST-996: memory prune with nothing to prune reports zero
// traces: TASK-43
func TestMemoryPruneNothingToPrune(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store a fresh learning (confidence 1.0, well above threshold)
	learnCmd := exec.Command("projctl", "memory", "learn",
		"--message", "Fresh high-confidence learning",
		"--memoryroot", memoryDir)
	_, err = learnCmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "prune",
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Prune should succeed: %s", string(output))
	g.Expect(string(output)).To(MatchRegexp(`removed.*0|no entries|nothing`))
}
