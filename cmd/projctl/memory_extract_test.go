package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
)

// ============================================================================
// Memory Extract CLI Tests (TASK-7)
// ============================================================================

// TEST: memory extract --result flag works
// Traces to: TASK-7 AC-3, AC-11, AC-12
func TestMemoryExtractResultFlag(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	// Create a valid result file
	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[[decisions]]
context = "Error handling strategy"
choice = "Use wrapped errors"
reason = "Clear error traces"
alternatives = ["Sentinel errors"]

[context]
phase = "design"
task = "TASK-10"
`
	resultPath := filepath.Join(tempDir, "result.toml")
	err := os.WriteFile(resultPath, []byte(resultContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--result", resultPath,
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Extracted"))
	g.Expect(string(output)).To(ContainSubstring("1 decision"))
}

// TEST: memory extract --yield flag works
// Traces to: TASK-7 AC-4, AC-11, AC-13
func TestMemoryExtractYieldFlag(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	// Create a valid yield file
	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Implemented the Extract function"
learnings = ["SQLite-vec works well", "Mean pooling recommended"]

[context]
phase = "tdd-green"
task = "TASK-4"
`
	yieldPath := filepath.Join(tempDir, "yield.toml")
	err := os.WriteFile(yieldPath, []byte(yieldContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--yield", yieldPath,
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Extracted"))
	g.Expect(string(output)).To(ContainSubstring("2 learning"))
}

// TEST: memory extract fails when both --result and --yield provided
// Traces to: TASK-7 AC-5, AC-15
func TestMemoryExtractMutualExclusionBothFlags(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	resultPath := filepath.Join(tempDir, "result.toml")
	yieldPath := filepath.Join(tempDir, "yield.toml")
	err := os.WriteFile(resultPath, []byte("[status]\nresult = \"success\""), 0644)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(yieldPath, []byte("[yield]\ntype = \"complete\""), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--result", resultPath,
		"--yield", yieldPath)
	output, err := cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred(), "Should fail with both flags: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("mutually exclusive"))
}

// TEST: memory extract fails when neither --result nor --yield provided
// Traces to: TASK-7 AC-5, AC-15
func TestMemoryExtractMutualExclusionNoFlags(t *testing.T) {
	g := NewWithT(t)

	cmd := exec.Command("projctl", "memory", "extract")
	output, err := cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred(), "Should fail with no flags: %s", string(output))
	g.Expect(string(output)).To(SatisfyAny(
		ContainSubstring("--result"),
		ContainSubstring("--yield"),
	))
}

// TEST: memory extract shows success message with item count
// Traces to: TASK-7 AC-7
func TestMemoryExtractShowsSuccessMessage(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Test summary"
findings = ["Finding 1", "Finding 2"]
learnings = ["Learning 1"]

[context]
phase = "tdd-green"
task = "TASK-7"
`
	yieldPath := filepath.Join(tempDir, "yield.toml")
	err := os.WriteFile(yieldPath, []byte(yieldContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--yield", yieldPath,
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	// Should show success with checkmark and item count
	g.Expect(string(output)).To(MatchRegexp(`(?i)(Extracted|✓).*\d+.*items?`))
}

// TEST: memory extract shows item breakdown
// Traces to: TASK-7 AC-8
func TestMemoryExtractShowsItemBreakdown(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[[decisions]]
context = "Decision 1"
choice = "Choice A"
reason = "Reason A"
alternatives = []

[[decisions]]
context = "Decision 2"
choice = "Choice B"
reason = "Reason B"
alternatives = []

[context]
phase = "design"
task = "TASK-10"
`
	resultPath := filepath.Join(tempDir, "result.toml")
	err := os.WriteFile(resultPath, []byte(resultContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--result", resultPath,
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	// Should show breakdown with decision count
	g.Expect(string(output)).To(ContainSubstring("2 decision"))
}

// TEST: memory extract shows storage location
// Traces to: TASK-7 AC-9
func TestMemoryExtractShowsStorageLocation(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Test summary"

[context]
phase = "tdd-green"
task = "TASK-7"
`
	yieldPath := filepath.Join(tempDir, "yield.toml")
	err := os.WriteFile(yieldPath, []byte(yieldContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--yield", yieldPath,
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	// Should show storage location
	g.Expect(string(output)).To(SatisfyAny(
		ContainSubstring("embeddings.db"),
		ContainSubstring("semantic memory"),
		ContainSubstring("Stored"),
	))
}

// TEST: memory extract shows error in proper format
// Traces to: TASK-7 AC-10
func TestMemoryExtractShowsErrorFormat(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Non-existent file
	cmd := exec.Command("projctl", "memory", "extract",
		"--result", filepath.Join(tempDir, "nonexistent.toml"))
	output, err := cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred(), "Should fail: %s", string(output))
	// Error should be displayed (stderr or stdout)
	outputStr := string(output)
	g.Expect(outputStr).To(SatisfyAny(
		ContainSubstring("Error"),
		ContainSubstring("error"),
		ContainSubstring("failed"),
	))
}

// TEST: memory extract outputs TOML to stdout
// Traces to: TASK-7 AC-11
func TestMemoryExtractOutputsTOML(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Test TOML output"
learnings = ["Learning 1"]

[context]
phase = "tdd-green"
task = "TASK-7"
`
	yieldPath := filepath.Join(tempDir, "yield.toml")
	err := os.WriteFile(yieldPath, []byte(yieldContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--yield", yieldPath,
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))

	// Output should contain TOML that can be parsed
	// Look for TOML structure markers
	outputStr := string(output)
	g.Expect(outputStr).To(SatisfyAny(
		ContainSubstring("[result]"),
		ContainSubstring("status ="),
		ContainSubstring("items_extracted ="),
	))
}

// TEST: memory extract TOML output is machine-readable
// Traces to: TASK-7 AC-11
func TestMemoryExtractTOMLIsMachineReadable(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Test TOML parsing"

[context]
phase = "tdd-green"
task = "TASK-7"
`
	yieldPath := filepath.Join(tempDir, "yield.toml")
	err := os.WriteFile(yieldPath, []byte(yieldContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--yield", yieldPath,
		"--memoryroot", memoryDir)
	// Capture stdout only (TOML goes to stdout, human-readable goes to stderr)
	stdout, err := cmd.Output()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed")

	// stdout should be pure TOML
	outputStr := string(stdout)
	g.Expect(outputStr).To(ContainSubstring("[result]"))

	// Try to parse as TOML
	var result struct {
		Result struct {
			Status         string `toml:"status"`
			ItemsExtracted int    `toml:"items_extracted"`
		} `toml:"result"`
	}
	_, err = toml.Decode(outputStr, &result)
	g.Expect(err).ToNot(HaveOccurred(), "TOML should be parseable: %s", outputStr)
	g.Expect(result.Result.Status).To(Equal("success"))
	g.Expect(result.Result.ItemsExtracted).To(BeNumerically(">=", 1))
}

// TEST: memory extract integration with result file end-to-end
// Traces to: TASK-7 AC-12
func TestMemoryExtractIntegrationResultFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:45:00Z"

[[decisions]]
context = "Integration test decision"
choice = "Use subprocess testing"
reason = "Validates CLI end-to-end"
alternatives = ["Unit tests only"]

[context]
phase = "design"
task = "TASK-7"
`
	resultPath := filepath.Join(tempDir, "integration-result.toml")
	err := os.WriteFile(resultPath, []byte(resultContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--result", resultPath,
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Extracted"))
	g.Expect(string(output)).To(ContainSubstring("decision"))
}

// TEST: memory extract integration with yield file end-to-end
// Traces to: TASK-7 AC-13
func TestMemoryExtractIntegrationYieldFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	yieldContent := `
[yield]
type = "complete"
timestamp = "2026-02-04T10:30:00Z"

[payload]
summary = "Integration test for yield extraction"
findings = ["Finding A", "Finding B"]
learnings = ["Learning X", "Learning Y", "Learning Z"]

[context]
phase = "tdd-green"
task = "TASK-7"
`
	yieldPath := filepath.Join(tempDir, "integration-yield.toml")
	err := os.WriteFile(yieldPath, []byte(yieldContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--yield", yieldPath,
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("Extracted"))
	// Should show summary, findings, and learnings in some form
	g.Expect(string(output)).To(SatisfyAny(
		ContainSubstring("6 items"), // 1 summary + 2 findings + 3 learnings
		ContainSubstring("3 learning"),
		ContainSubstring("2 finding"),
	))
}
