package trace_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/trace"
)

// TEST-201: ValidateV2Artifacts accepts phase parameter
// traces: TASK-1
func TestValidateV2Artifacts_AcceptsPhaseParameter(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Create minimal artifact
	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Feature

Description.
`), 0o644)).To(Succeed())

	// Should accept phase parameter (even if validation rules don't change yet)
	result, err := trace.ValidateV2Artifacts(tempDir, "arch_commit")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

// TEST-202: At arch_commit, ARCH IDs can be unlinked
// traces: TASK-1
func TestValidateV2Artifacts_ArchitectComplete_AllowsUnlinkedARCH(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// ARCH exists but no TASK traces to it yet (this is expected during architect phase)
	g.Expect(os.WriteFile(filepath.Join(tempDir, "architecture.md"), []byte(`# Architecture

### ARCH-001: Component Design

Description.

**Traces to:** REQ-001
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Feature

Description.
`), 0o644)).To(Succeed())

	// At architect-complete, ARCH-001 should be allowed to be unlinked (no TASK traces to it yet)
	result, err := trace.ValidateV2Artifacts(tempDir, "arch_commit")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeTrue(), "ARCH should be allowed to be unlinked at arch_commit")
	g.Expect(result.UnlinkedIDs).ToNot(ContainElement("ARCH-001"), "ARCH-001 should not be reported as unlinked at arch_commit")
}

// TEST-203: At breakdown-complete, TASK IDs can be unlinked
// traces: TASK-1
func TestValidateV2Artifacts_BreakdownComplete_AllowsUnlinkedTASK(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// TASK exists but no TEST traces to it yet (this is expected during breakdown phase)
	g.Expect(os.WriteFile(filepath.Join(tempDir, "tasks.md"), []byte(`# Tasks

### TASK-001: Implement feature

Description.

**Traces to:** ARCH-001
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "architecture.md"), []byte(`# Architecture

### ARCH-001: Component Design

**Traces to:** REQ-001
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Feature

Description.
`), 0o644)).To(Succeed())

	// At breakdown-complete, TASK-001 should be allowed to be unlinked (no TEST traces to it yet)
	result, err := trace.ValidateV2Artifacts(tempDir, "breakdown_commit")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeTrue(), "TASK should be allowed to be unlinked at breakdown_commit")
	g.Expect(result.UnlinkedIDs).ToNot(ContainElement("TASK-001"), "TASK-001 should not be reported as unlinked at breakdown_commit")
}

// TEST-204: At task-complete and later, full chain is required
// traces: TASK-1
func TestValidateV2Artifacts_TaskComplete_RequiresFullChain(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// TASK exists but no TEST traces to it
	g.Expect(os.WriteFile(filepath.Join(tempDir, "tasks.md"), []byte(`# Tasks

### TASK-001: Implement feature

Description.

**Traces to:** ARCH-001
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "architecture.md"), []byte(`# Architecture

### ARCH-001: Component Design

**Traces to:** REQ-001
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Feature

Description.
`), 0o644)).To(Succeed())

	// At task-complete, TASK-001 MUST have TEST tracing to it
	result, err := trace.ValidateV2Artifacts(tempDir, "tdd_commit")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeFalse(), "Validation should fail when TASK has no TEST at tdd_commit")
	g.Expect(result.UnlinkedIDs).To(ContainElement("TASK-001"), "TASK-001 should be reported as unlinked at tdd_commit")
}

// TEST-205: Without phase parameter, strictest validation applies
// traces: TASK-1
func TestValidateV2Artifacts_NoPhase_StrictestValidation(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// ARCH exists but no TASK traces to it
	g.Expect(os.WriteFile(filepath.Join(tempDir, "architecture.md"), []byte(`# Architecture

### ARCH-001: Component Design

**Traces to:** REQ-001
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Feature

Description.
`), 0o644)).To(Succeed())

	// Without phase, should use strictest validation (current behavior)
	result, err := trace.ValidateV2Artifacts(tempDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeFalse(), "Validation should fail for unlinked ARCH without phase parameter")
	g.Expect(result.UnlinkedIDs).To(ContainElement("ARCH-001"), "ARCH-001 should be reported as unlinked without phase parameter")
}

// TEST-206: Phase-aware validation propagates through CLI
// traces: TASK-1
func TestValidateV2Artifacts_PhaseParameterPropagation(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(tempDir, "architecture.md"), []byte(`# Architecture

### ARCH-001: Component Design

**Traces to:** REQ-001
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Feature

Description.
`), 0o644)).To(Succeed())

	// Test with different phases to ensure parameter is actually used
	resultArchitect, err := trace.ValidateV2Artifacts(tempDir, "arch_commit")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(resultArchitect.Pass).To(BeTrue(), "Should pass at arch_commit")

	resultTask, err := trace.ValidateV2Artifacts(tempDir, "tdd_commit")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(resultTask.Pass).To(BeFalse(), "Should fail at tdd_commit")
}

// TEST-207: Early phases allow upstream unlinked IDs
// traces: TASK-1
func TestValidateV2Artifacts_EarlyPhases_AllowUpstreamUnlinked(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// DES exists but no ARCH traces to it yet (expected during design phase)
	g.Expect(os.WriteFile(filepath.Join(tempDir, "design.md"), []byte(`# Design

### DES-001: Feature Design

**Traces to:** REQ-001
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Feature

Description.
`), 0o644)).To(Succeed())

	// At design-complete, DES-001 should be allowed to be unlinked
	result, err := trace.ValidateV2Artifacts(tempDir, "design_commit")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeTrue(), "DES should be allowed to be unlinked at design_commit")
	g.Expect(result.UnlinkedIDs).ToNot(ContainElement("DES-001"), "DES-001 should not be reported as unlinked at design_commit")
}

// TEST-209: ISSUE IDs referenced in Traces to are not reported as orphan when defined in issues.md
// traces: TASK-1
// Reproduces ISSUE-57: projctl trace validate reported ISSUE-54 as orphan
func TestValidateV2Artifacts_IssueIDNotOrphan(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// ISSUE-54 is defined in issues.md
	g.Expect(os.WriteFile(filepath.Join(tempDir, "issues.md"), []byte(`# Issues

### ISSUE-54: PM phase must interview user before producing artifacts

**Priority:** High
**Status:** done
`), 0o644)).To(Succeed())

	// Requirements trace to the issue
	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Interview User

The PM skill must interview the user.

**Traces to:** ISSUE-54
`), 0o644)).To(Succeed())

	result, err := trace.ValidateV2Artifacts(tempDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.OrphanIDs).ToNot(ContainElement("ISSUE-54"), "ISSUE-54 is defined in issues.md, should not be orphan")
}

// TEST-210: ISSUE IDs not orphan when docs_dir is configured
// traces: TASK-1
// Reproduces ISSUE-57 with docs_dir = "docs" (real project config)
func TestValidateV2Artifacts_IssueIDNotOrphan_WithDocsDir(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Create docs directory and config
	g.Expect(os.MkdirAll(filepath.Join(tempDir, "docs"), 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(tempDir, ".claude"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(tempDir, ".claude", "project-config.toml"), []byte(`[paths]
docs_dir = "docs"
`), 0o644)).To(Succeed())

	// ISSUE-54 is defined in docs/issues.md
	g.Expect(os.WriteFile(filepath.Join(tempDir, "docs", "issues.md"), []byte(`# Issues

### ISSUE-54: PM phase must interview user before producing artifacts

**Priority:** High
**Status:** done
`), 0o644)).To(Succeed())

	// Requirements in docs/ trace to the issue
	g.Expect(os.WriteFile(filepath.Join(tempDir, "docs", "requirements.md"), []byte(`# Requirements

### REQ-001: Interview User

The PM skill must interview the user.

**Traces to:** ISSUE-54
`), 0o644)).To(Succeed())

	result, err := trace.ValidateV2Artifacts(tempDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.OrphanIDs).ToNot(ContainElement("ISSUE-54"), "ISSUE-54 is defined in docs/issues.md, should not be orphan")
}

// TEST-211: ISSUE IDs not orphan when validated from project subdirectory
// traces: TASK-1
// Core ISSUE-57 scenario: project artifacts in subdirectory reference ISSUE-NNN
// defined in repo-level docs/issues.md. Validation from subdirectory must not
// report ISSUE IDs as orphans since issues are always defined at repo root.
func TestValidateV2Artifacts_IssueIDNotOrphan_ProjectSubdir(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Simulate repo structure with project subdirectory
	projectDir := filepath.Join(tempDir, ".claude", "projects", "my-project")
	g.Expect(os.MkdirAll(projectDir, 0o755)).To(Succeed())

	// ISSUE-45 defined at repo root in docs/issues.md
	g.Expect(os.MkdirAll(filepath.Join(tempDir, "docs"), 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(tempDir, ".claude"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(tempDir, ".claude", "project-config.toml"), []byte(`[paths]
docs_dir = "docs"
`), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(tempDir, "docs", "issues.md"), []byte(`# Issues

### ISSUE-45: Layer 0 Foundation

Description.
`), 0o644)).To(Succeed())

	// Project artifacts in subdirectory reference the issue
	g.Expect(os.WriteFile(filepath.Join(projectDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Foundation Feature

Description.

**Traces to:** ISSUE-45
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(projectDir, "architecture.md"), []byte(`# Architecture

### ARCH-001: Component

Description.

**Traces to:** REQ-001
`), 0o644)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(projectDir, "tasks.md"), []byte(`# Tasks

### TASK-001: Implement feature

Description.

**Traces to:** ARCH-001
`), 0o644)).To(Succeed())

	// Validate from project subdirectory (the actual scenario that fails)
	result, err := trace.ValidateV2Artifacts(projectDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.OrphanIDs).ToNot(ContainElement("ISSUE-45"),
		"ISSUE-45 should not be orphan - issues are defined at repo root, not in project directories")
}

// TEST-208: Invalid phase parameter returns error
// traces: TASK-1
func TestValidateV2Artifacts_InvalidPhase_ReturnsError(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(tempDir, "requirements.md"), []byte(`# Requirements

### REQ-001: Feature

Description.
`), 0o644)).To(Succeed())

	// Should reject invalid phase names
	_, err := trace.ValidateV2Artifacts(tempDir, "invalid-phase-name")
	g.Expect(err).To(HaveOccurred(), "Should return error for invalid phase")
	g.Expect(err.Error()).To(ContainSubstring("invalid phase"), "Error should mention invalid phase")
}
