//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// TestTraceShowASCII tests the CLI interface for projctl trace show (default ASCII)
func TestTraceShowASCII(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Create artifacts with trace links (at root, since DocsDir defaults to "")
	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Feature

Description.
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "design.md"), []byte(`# Design

### DES-001: Design

**Traces to:** REQ-001
`), 0o644)).To(Succeed())

	cmd := exec.Command("projctl", "trace", "show", "--dir", tempDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("REQ-001"))
	g.Expect(string(output)).To(ContainSubstring("DES-001"))
}

// TestTraceShowJSON tests the CLI interface for projctl trace show --format json
func TestTraceShowJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Create artifacts with trace links (at root, since DocsDir defaults to "")
	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Feature

Description.
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "design.md"), []byte(`# Design

### DES-001: Design

**Traces to:** REQ-001
`), 0o644)).To(Succeed())

	cmd := exec.Command("projctl", "trace", "show", "--dir", tempDir, "--format", "json")
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring(`"nodes"`))
	g.Expect(string(output)).To(ContainSubstring(`"edges"`))
	g.Expect(string(output)).To(ContainSubstring(`"REQ-001"`))
	g.Expect(string(output)).To(ContainSubstring(`"DES-001"`))
}

// TestTraceShowInvalidFormat tests error handling for invalid format
func TestTraceShowInvalidFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	cmd := exec.Command("projctl", "trace", "show", "--dir", tempDir, "--format", "invalid")
	output, err := cmd.CombinedOutput()

	g.Expect(err).To(HaveOccurred(), "Command should fail for invalid format")
	g.Expect(string(output)).To(ContainSubstring("invalid format"))
}

// TEST-800: CLI trace promote promotes TASK traces to permanent IDs
// traces: TASK-008
func TestTracePromote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	// tasks.md goes at root (DocsDir is empty by default)
	g.Expect(os.WriteFile(filepath.Join(tempDir, "tasks.md"), []byte(`# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`), 0o644)).To(Succeed())

	// Create a test file with TASK trace
	internalDir := filepath.Join(tempDir, "internal", "feature")
	g.Expect(os.MkdirAll(internalDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(internalDir, "feature_test.go"), []byte(`package feature_test

import "testing"

// TEST-100: Feature test
// traces: TASK-001
func TestFeature(t *testing.T) {
}
`), 0o644)).To(Succeed())

	cmd := exec.Command("projctl", "trace", "promote", "--dir", tempDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("1 file"))
	g.Expect(string(output)).To(ContainSubstring("TASK-001"))
	g.Expect(string(output)).To(ContainSubstring("ARCH-001"))

	// Verify file was actually modified
	content, err := os.ReadFile(filepath.Join(internalDir, "feature_test.go"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("// traces: ARCH-001"))
	g.Expect(string(content)).ToNot(ContainSubstring("// traces: TASK-001"))
}

// TEST-801: CLI trace promote --dry-run shows changes without modifying
// traces: TASK-008
func TestTracePromoteDryRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	// tasks.md goes at root (DocsDir is empty by default)
	g.Expect(os.WriteFile(filepath.Join(tempDir, "tasks.md"), []byte(`# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`), 0o644)).To(Succeed())

	internalDir := filepath.Join(tempDir, "internal", "feature")
	g.Expect(os.MkdirAll(internalDir, 0o755)).To(Succeed())
	testFile := filepath.Join(internalDir, "feature_test.go")
	originalContent := `package feature_test

import "testing"

// TEST-100: Feature test
// traces: TASK-001
func TestFeature(t *testing.T) {
}
`
	g.Expect(os.WriteFile(testFile, []byte(originalContent), 0o644)).To(Succeed())

	cmd := exec.Command("projctl", "trace", "promote", "--dir", tempDir, "--dryrun")
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("dry run"))
	g.Expect(string(output)).To(ContainSubstring("TASK-001"))
	g.Expect(string(output)).To(ContainSubstring("ARCH-001"))

	// Verify file was NOT modified
	content, err := os.ReadFile(testFile)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal(originalContent))
}

// TEST-802: CLI trace promote reports number of files modified
// traces: TASK-008
func TestTracePromoteReportsFileCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	docsDir := filepath.Join(tempDir, "docs")
	g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "tasks.md"), []byte(`# Tasks

### TASK-001: Feature one

**Traces to:** ARCH-001

### TASK-002: Feature two

**Traces to:** ARCH-002
`), 0o644)).To(Succeed())

	// Create two test files
	dir1 := filepath.Join(tempDir, "internal", "feature1")
	dir2 := filepath.Join(tempDir, "internal", "feature2")
	g.Expect(os.MkdirAll(dir1, 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(dir2, 0o755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(dir1, "feature1_test.go"), []byte(`package feature1_test

// TEST-100: Feature test
// traces: TASK-001
func TestFeature1() {}
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(dir2, "feature2_test.go"), []byte(`package feature2_test

// TEST-101: Feature test
// traces: TASK-002
func TestFeature2() {}
`), 0o644)).To(Succeed())

	cmd := exec.Command("projctl", "trace", "promote", "--dir", tempDir)
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring("2 file"))
}

// TEST-803: CLI trace promote with JSON output
// traces: TASK-008
func TestTracePromoteJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	// tasks.md goes at root (DocsDir is empty by default)
	g.Expect(os.WriteFile(filepath.Join(tempDir, "tasks.md"), []byte(`# Tasks

### TASK-001: Implement feature

**Traces to:** ARCH-001
`), 0o644)).To(Succeed())

	internalDir := filepath.Join(tempDir, "internal", "feature")
	g.Expect(os.MkdirAll(internalDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(internalDir, "feature_test.go"), []byte(`package feature_test

// TEST-100: Feature test
// traces: TASK-001
func TestFeature() {}
`), 0o644)).To(Succeed())

	cmd := exec.Command("projctl", "trace", "promote", "--dir", tempDir, "--json")
	output, err := cmd.CombinedOutput()

	g.Expect(err).ToNot(HaveOccurred(), "Command should succeed: %s", string(output))
	g.Expect(string(output)).To(ContainSubstring(`"promotions"`))
	g.Expect(string(output)).To(ContainSubstring(`"old_trace"`))
	g.Expect(string(output)).To(ContainSubstring(`"new_trace"`))
}
