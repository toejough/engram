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
	result, err := trace.ValidateV2Artifacts(tempDir, "architect-complete")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

// TEST-202: At architect-complete, ARCH IDs can be unlinked
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
	result, err := trace.ValidateV2Artifacts(tempDir, "architect-complete")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeTrue(), "ARCH should be allowed to be unlinked at architect-complete")
	g.Expect(result.UnlinkedIDs).ToNot(ContainElement("ARCH-001"), "ARCH-001 should not be reported as unlinked at architect-complete")
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
	result, err := trace.ValidateV2Artifacts(tempDir, "breakdown-complete")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeTrue(), "TASK should be allowed to be unlinked at breakdown-complete")
	g.Expect(result.UnlinkedIDs).ToNot(ContainElement("TASK-001"), "TASK-001 should not be reported as unlinked at breakdown-complete")
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
	result, err := trace.ValidateV2Artifacts(tempDir, "task-complete")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeFalse(), "Validation should fail when TASK has no TEST at task-complete")
	g.Expect(result.UnlinkedIDs).To(ContainElement("TASK-001"), "TASK-001 should be reported as unlinked at task-complete")
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
	resultArchitect, err := trace.ValidateV2Artifacts(tempDir, "architect-complete")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(resultArchitect.Pass).To(BeTrue(), "Should pass at architect-complete")

	resultTask, err := trace.ValidateV2Artifacts(tempDir, "task-complete")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(resultTask.Pass).To(BeFalse(), "Should fail at task-complete")
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
	result, err := trace.ValidateV2Artifacts(tempDir, "design-complete")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeTrue(), "DES should be allowed to be unlinked at design-complete")
	g.Expect(result.UnlinkedIDs).ToNot(ContainElement("DES-001"), "DES-001 should not be reported as unlinked at design-complete")
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
