//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
)

// TEST: memory extract integration with result file end-to-end
// Traces to: TASK-7 AC-12
func TestMemoryExtractIntegrationResultFile(t *testing.T) {
	t.Parallel()
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

// TEST: memory extract fails when --result flag not provided
// Traces to: TASK-7 AC-5, AC-15
func TestMemoryExtractMissingResultFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cmd := exec.Command("projctl", "memory", "extract")
	output, err := cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred(), "Should fail with no flags: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("--result"))
}

// TEST: memory extract outputs TOML to stdout
// Traces to: TASK-7 AC-11
func TestMemoryExtractOutputsTOML(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:30:00Z"

[[decisions]]
context = "Test TOML output"
choice = "Test choice"
reason = "Test reason"
alternatives = []

[context]
phase = "design"
task = "TASK-7"
`
	resultPath := filepath.Join(tempDir, "result.toml")
	err := os.WriteFile(resultPath, []byte(resultContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--result", resultPath,
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

// ============================================================================
// Memory Extract CLI Tests (TASK-7)
// ============================================================================

// TEST: memory extract --result flag works
// Traces to: TASK-7 AC-3, AC-11, AC-12
func TestMemoryExtractResultFlag(t *testing.T) {
	t.Parallel()
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

// TEST: memory extract shows error in proper format
// Traces to: TASK-7 AC-10
func TestMemoryExtractShowsErrorFormat(t *testing.T) {
	t.Parallel()
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

// TEST: memory extract shows item breakdown
// Traces to: TASK-7 AC-8
func TestMemoryExtractShowsItemBreakdown(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:30:00Z"

[[decisions]]
context = "Test decision"
choice = "Test choice"
reason = "Test reason"
alternatives = []

[context]
phase = "design"
task = "TASK-7"
`
	resultPath := filepath.Join(tempDir, "result.toml")
	err := os.WriteFile(resultPath, []byte(resultContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--result", resultPath,
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

// TEST: memory extract shows success message with item count
// Traces to: TASK-7 AC-7
func TestMemoryExtractShowsSuccessMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:30:00Z"

[[decisions]]
context = "Test decision"
choice = "Test choice"
reason = "Test reason"
alternatives = []

[context]
phase = "design"
task = "TASK-7"
`
	resultPath := filepath.Join(tempDir, "result.toml")
	err := os.WriteFile(resultPath, []byte(resultContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--result", resultPath,
		"--memoryroot", memoryDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	// Should show success with checkmark and item count
	g.Expect(string(output)).To(MatchRegexp(`(?i)(Extracted|✓).*\d+.*items?`))
}

// TEST: memory extract TOML output is machine-readable
// Traces to: TASK-7 AC-11
func TestMemoryExtractTOMLIsMachineReadable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")

	resultContent := `
[status]
result = "success"
timestamp = "2026-02-04T10:30:00Z"

[[decisions]]
context = "Test TOML parsing"
choice = "Test choice"
reason = "Test reason"
alternatives = []

[context]
phase = "design"
task = "TASK-7"
`
	resultPath := filepath.Join(tempDir, "result.toml")
	err := os.WriteFile(resultPath, []byte(resultContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command("projctl", "memory", "extract",
		"--result", resultPath,
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
