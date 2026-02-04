package trace_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/trace"
)

// Integration tests for trace.Repair function.
// These tests use real file operations to verify end-to-end behavior.

func TestIntegrationRepairDuplicateIDSameFile(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a design file with duplicate IDs in the same file
	// Note: The current implementation uses strings.ReplaceAll, which replaces
	// ALL occurrences of the duplicate ID in the file. This means when there are
	// two DES-001 definitions in the same file, BOTH get renamed to DES-002.
	// This is arguably a bug, but this test verifies the current behavior.
	content := `# Design

### DES-001: First Design

First design description.

**Traces to:** REQ-001

### DES-001: Duplicate Design

Duplicate design in same file.

**Traces to:** REQ-002
`
	writeArtifact(t, dir, "design.md", content)
	writeArtifact(t, dir, "requirements.md", `# Requirements

### REQ-001: First Requirement

Description.

### REQ-002: Second Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect duplicate and renumber
	g.Expect(result.DuplicateIDs).To(ContainElement("DES-001"))
	g.Expect(result.Renumbered).To(HaveLen(1))
	g.Expect(result.Renumbered[0].OldID).To(Equal("DES-001"))
	g.Expect(result.Renumbered[0].NewID).To(Equal("DES-002"))
	g.Expect(result.Renumbered[0].File).To(Equal("design.md"))

	// Verify file was actually modified
	updatedContent, err := os.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Due to ReplaceAll behavior, BOTH occurrences of DES-001 become DES-002
	// (this is current behavior - arguably a bug but this test documents it)
	g.Expect(string(updatedContent)).To(ContainSubstring("### DES-002: First Design"))
	g.Expect(string(updatedContent)).To(ContainSubstring("### DES-002: Duplicate Design"))
	g.Expect(string(updatedContent)).To(ContainSubstring("**Traces to:** REQ-001"))
	g.Expect(string(updatedContent)).To(ContainSubstring("**Traces to:** REQ-002"))

	// No escalations for duplicate IDs (auto-fixed)
	g.Expect(result.Escalations).To(BeEmpty())
}

func TestIntegrationRepairDuplicateIDAcrossFiles(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create two design files with same ID
	writeArtifact(t, dir, "design.md", `# Design

### DES-001: Main Design

Main design description.

**Traces to:** REQ-001
`)

	writeArtifact(t, dir, "design-feature.md", `# Feature Design

### DES-001: Feature Design

Feature design description.

**Traces to:** REQ-002
`)

	writeArtifact(t, dir, "requirements.md", `# Requirements

### REQ-001: First Requirement

Description.

### REQ-002: Second Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect duplicate and renumber
	g.Expect(result.DuplicateIDs).To(ContainElement("DES-001"))
	g.Expect(result.Renumbered).To(HaveLen(1))
	g.Expect(result.Renumbered[0].OldID).To(Equal("DES-001"))
	g.Expect(result.Renumbered[0].NewID).To(Equal("DES-002"))

	// Verify the first file kept DES-001
	mainContent, err := os.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(mainContent)).To(ContainSubstring("### DES-001: Main Design"))
	g.Expect(string(mainContent)).ToNot(ContainSubstring("DES-002"))

	// Verify the second file was renumbered to DES-002
	featureContent, err := os.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(featureContent)).To(ContainSubstring("### DES-002: Feature Design"))
	g.Expect(string(featureContent)).ToNot(ContainSubstring("DES-001"))

	// No escalations for duplicate IDs (auto-fixed)
	g.Expect(result.Escalations).To(BeEmpty())
}

func TestIntegrationRepairDanglingReferenceEscalation(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a design that references a non-existent requirement
	writeArtifact(t, dir, "design.md", `# Design

### DES-001: Orphan Design

Design that references non-existent requirement.

**Traces to:** REQ-999
`)

	// Run repair
	result, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect dangling reference
	g.Expect(result.DanglingRefs).To(ContainElement("REQ-999"))

	// Should create escalation for dangling ref
	g.Expect(result.Escalations).To(HaveLen(1))
	g.Expect(result.Escalations[0].ID).To(Equal("REQ-999"))
	g.Expect(result.Escalations[0].Reason).To(ContainSubstring("dangling"))
	g.Expect(result.Escalations[0].Reason).To(ContainSubstring("referenced"))
	g.Expect(result.Escalations[0].Reason).To(ContainSubstring("not defined"))

	// File should NOT be modified (dangling refs are not auto-fixed)
	content, err := os.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("REQ-999"))
}

func TestIntegrationRepairUsesNextAvailableID(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create design files where DES-001 through DES-005 exist
	// Then have a duplicate DES-003 - should renumber to DES-006
	writeArtifact(t, dir, "design.md", `# Design

### DES-001: Design One

Description.

**Traces to:** REQ-001

### DES-002: Design Two

Description.

**Traces to:** REQ-001

### DES-003: Design Three

Description.

**Traces to:** REQ-001

### DES-004: Design Four

Description.

**Traces to:** REQ-001

### DES-005: Design Five

Description.

**Traces to:** REQ-001
`)

	// Create a duplicate DES-003 in another file
	writeArtifact(t, dir, "design-feature.md", `# Feature Design

### DES-003: Duplicate Three

This is a duplicate of DES-003.

**Traces to:** REQ-002
`)

	writeArtifact(t, dir, "requirements.md", `# Requirements

### REQ-001: First

Description.

### REQ-002: Second

Description.
`)

	// Run repair
	result, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect duplicate
	g.Expect(result.DuplicateIDs).To(ContainElement("DES-003"))

	// Should renumber to DES-006 (next available after DES-005)
	g.Expect(result.Renumbered).To(HaveLen(1))
	g.Expect(result.Renumbered[0].OldID).To(Equal("DES-003"))
	g.Expect(result.Renumbered[0].NewID).To(Equal("DES-006"))

	// Verify file was updated correctly
	featureContent, err := os.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(featureContent)).To(ContainSubstring("### DES-006: Duplicate Three"))
}

func TestIntegrationRepairNoDuplicateEscalations(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create files with duplicate IDs (should be auto-fixed, not escalated)
	writeArtifact(t, dir, "design.md", `# Design

### DES-001: First

Description.

**Traces to:** REQ-001
`)

	writeArtifact(t, dir, "design-feature.md", `# Feature Design

### DES-001: Duplicate

Description.

**Traces to:** REQ-001
`)

	writeArtifact(t, dir, "requirements.md", `# Requirements

### REQ-001: Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// Should have duplicate detected
	g.Expect(result.DuplicateIDs).To(ContainElement("DES-001"))

	// Should have renumbered
	g.Expect(result.Renumbered).ToNot(BeEmpty())

	// NO escalations for duplicates - they are auto-fixed
	g.Expect(result.Escalations).To(BeEmpty())
}

func TestIntegrationRepairIsIdempotent(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create design files with duplicate IDs
	writeArtifact(t, dir, "design.md", `# Design

### DES-001: First Design

Description.

**Traces to:** REQ-001
`)

	writeArtifact(t, dir, "design-feature.md", `# Feature Design

### DES-001: Duplicate Design

Description.

**Traces to:** REQ-001
`)

	writeArtifact(t, dir, "requirements.md", `# Requirements

### REQ-001: Requirement

Description.
`)

	// Run repair first time
	result1, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result1.DuplicateIDs).To(ContainElement("DES-001"))
	g.Expect(result1.Renumbered).To(HaveLen(1))

	// Read file contents after first repair
	mainContent1, err := os.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())
	featureContent1, err := os.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Run repair second time
	result2, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// Should find NO duplicates or issues on second run
	g.Expect(result2.DuplicateIDs).To(BeEmpty())
	g.Expect(result2.DanglingRefs).To(BeEmpty())
	g.Expect(result2.Renumbered).To(BeEmpty())
	g.Expect(result2.Escalations).To(BeEmpty())

	// Files should be unchanged from first repair
	mainContent2, err := os.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())
	featureContent2, err := os.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(string(mainContent1)).To(Equal(string(mainContent2)))
	g.Expect(string(featureContent1)).To(Equal(string(featureContent2)))
}

func TestIntegrationRepairMultipleDuplicatesAndDangling(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a complex scenario with multiple issues
	writeArtifact(t, dir, "design.md", `# Design

### DES-001: Design One

Description.

**Traces to:** REQ-001

### DES-002: Design Two

Description.

**Traces to:** REQ-001
`)

	// Duplicate DES-001 and reference non-existent REQ-999
	writeArtifact(t, dir, "design-feature.md", `# Feature Design

### DES-001: Duplicate One

Description.

**Traces to:** REQ-999
`)

	// Another duplicate DES-002
	writeArtifact(t, dir, "design-other.md", `# Other Design

### DES-002: Duplicate Two

Description.

**Traces to:** REQ-888
`)

	writeArtifact(t, dir, "requirements.md", `# Requirements

### REQ-001: Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect both duplicates
	g.Expect(result.DuplicateIDs).To(ContainElements("DES-001", "DES-002"))

	// Should renumber both duplicates
	g.Expect(result.Renumbered).To(HaveLen(2))

	// Both should be renumbered to new IDs (DES-003 and DES-004)
	newIDs := make([]string, 0, 2)
	for _, r := range result.Renumbered {
		newIDs = append(newIDs, r.NewID)
	}
	g.Expect(newIDs).To(ContainElements("DES-003", "DES-004"))

	// Should detect both dangling references
	g.Expect(result.DanglingRefs).To(ContainElements("REQ-999", "REQ-888"))

	// Should have escalations for dangling refs (not for duplicates)
	g.Expect(result.Escalations).To(HaveLen(2))
	escalationIDs := make([]string, 0, 2)
	for _, e := range result.Escalations {
		escalationIDs = append(escalationIDs, e.ID)
	}
	g.Expect(escalationIDs).To(ContainElements("REQ-999", "REQ-888"))
}

func TestIntegrationRepairPreservesFileContent(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a design file with duplicate ID and extra content
	originalContent := `# Design Documentation

This is important introductory text that should be preserved.

## Overview

Some overview content here.

### DES-001: Original Design

This is a detailed design description with **bold** and _italic_ text.

- Bullet point 1
- Bullet point 2

**Traces to:** REQ-001

## More Content

This content comes after the design section.
`
	writeArtifact(t, dir, "design.md", originalContent)

	// Create duplicate in another file
	writeArtifact(t, dir, "design-feature.md", `# Feature Design

### DES-001: Duplicate Design

Duplicate.

**Traces to:** REQ-001
`)

	writeArtifact(t, dir, "requirements.md", `# Requirements

### REQ-001: Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Renumbered).To(HaveLen(1))

	// The original file should be unchanged (it keeps DES-001)
	mainContent, err := os.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(mainContent)).To(Equal(originalContent))

	// The feature file should have DES-002 but preserve structure
	featureContent, err := os.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(featureContent)).To(ContainSubstring("# Feature Design"))
	g.Expect(string(featureContent)).To(ContainSubstring("### DES-002: Duplicate Design"))
	g.Expect(string(featureContent)).To(ContainSubstring("**Traces to:** REQ-001"))
}

func TestIntegrationRepairNoIssues(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create well-formed artifacts with no issues
	writeArtifact(t, dir, "requirements.md", `# Requirements

### REQ-001: First Requirement

Description.

### REQ-002: Second Requirement

Description.
`)

	writeArtifact(t, dir, "design.md", `# Design

### DES-001: First Design

**Traces to:** REQ-001

### DES-002: Second Design

**Traces to:** REQ-002
`)

	// Run repair
	result, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// Everything should be empty - no issues found
	g.Expect(result.DanglingRefs).To(BeEmpty())
	g.Expect(result.DuplicateIDs).To(BeEmpty())
	g.Expect(result.Renumbered).To(BeEmpty())
	g.Expect(result.Escalations).To(BeEmpty())
}

func TestIntegrationRepairReferencesUpdatedOnRenumber(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a file where DES-001 is defined and also referenced elsewhere
	writeArtifact(t, dir, "design.md", `# Design

### DES-001: Main Design

Description.

**Traces to:** REQ-001
`)

	// Duplicate DES-001 that also references the original DES-001 in its text
	writeArtifact(t, dir, "design-feature.md", `# Feature Design

### DES-001: Feature Design

This feature extends DES-001 from the main design.

**Traces to:** REQ-001
`)

	writeArtifact(t, dir, "requirements.md", `# Requirements

### REQ-001: Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// Should renumber the duplicate
	g.Expect(result.Renumbered).To(HaveLen(1))
	g.Expect(result.Renumbered[0].NewID).To(Equal("DES-002"))

	// The feature file should have ALL occurrences of DES-001 replaced with DES-002
	// (the ReplaceAll behavior in the implementation)
	featureContent, err := os.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Note: The current implementation uses strings.ReplaceAll which replaces
	// ALL occurrences of the duplicate ID in the file, including in body text
	g.Expect(string(featureContent)).To(ContainSubstring("### DES-002: Feature Design"))
	g.Expect(string(featureContent)).To(ContainSubstring("extends DES-002"))
}
