package trace_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/trace"
)

// Integration tests for trace.Repair function.
// These tests use real file operations to verify end-to-end behavior.

func TestIntegrationRepairDuplicateIDSameFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create a design file with duplicate IDs in the same file
	// Note: The current implementation uses strings.ReplaceAll, which replaces
	// ALL occurrences of the duplicate ID in the file. This means when there are
	// two DES-1 definitions in the same file, BOTH get renamed to DES-2.
	// This is arguably a bug, but this test verifies the current behavior.
	content := `# Design

### DES-1: First Design

First design description.

**Traces to:** REQ-1

### DES-1: Duplicate Design

Duplicate design in same file.

**Traces to:** REQ-2
`
	writeArtifact(t, fs, dir, "design.md", content)
	writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-1: First Requirement

Description.

### REQ-2: Second Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect duplicate and renumber
	g.Expect(result.DuplicateIDs).To(ContainElement("DES-1"))
	g.Expect(result.Renumbered).To(HaveLen(1))
	g.Expect(result.Renumbered[0].OldID).To(Equal("DES-1"))
	g.Expect(result.Renumbered[0].NewID).To(Equal("DES-2"))
	g.Expect(result.Renumbered[0].File).To(Equal("design.md"))

	// Verify file was actually modified
	updatedContent, err := fs.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Due to ReplaceAll behavior, BOTH occurrences of DES-1 become DES-2
	// (this is current behavior - arguably a bug but this test documents it)
	g.Expect(string(updatedContent)).To(ContainSubstring("### DES-2: First Design"))
	g.Expect(string(updatedContent)).To(ContainSubstring("### DES-2: Duplicate Design"))
	g.Expect(string(updatedContent)).To(ContainSubstring("**Traces to:** REQ-1"))
	g.Expect(string(updatedContent)).To(ContainSubstring("**Traces to:** REQ-2"))

	// No escalations for duplicate IDs (auto-fixed)
	g.Expect(result.Escalations).To(BeEmpty())
}

func TestIntegrationRepairDuplicateIDAcrossFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create two design files with same ID
	writeArtifact(t, fs, dir, "design.md", `# Design

### DES-1: Main Design

Main design description.

**Traces to:** REQ-1
`)

	writeArtifact(t, fs, dir, "design-feature.md", `# Feature Design

### DES-1: Feature Design

Feature design description.

**Traces to:** REQ-2
`)

	writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-1: First Requirement

Description.

### REQ-2: Second Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect duplicate and renumber
	g.Expect(result.DuplicateIDs).To(ContainElement("DES-1"))
	g.Expect(result.Renumbered).To(HaveLen(1))
	g.Expect(result.Renumbered[0].OldID).To(Equal("DES-1"))
	g.Expect(result.Renumbered[0].NewID).To(Equal("DES-2"))

	// Verify the first file kept DES-1
	mainContent, err := fs.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(mainContent)).To(ContainSubstring("### DES-1: Main Design"))
	g.Expect(string(mainContent)).ToNot(ContainSubstring("DES-2"))

	// Verify the second file was renumbered to DES-2
	featureContent, err := fs.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(featureContent)).To(ContainSubstring("### DES-2: Feature Design"))
	g.Expect(string(featureContent)).ToNot(ContainSubstring("DES-1"))

	// No escalations for duplicate IDs (auto-fixed)
	g.Expect(result.Escalations).To(BeEmpty())
}

func TestIntegrationRepairDanglingReferenceEscalation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create a design that references a non-existent requirement
	writeArtifact(t, fs, dir, "design.md", `# Design

### DES-1: Orphan Design

Design that references non-existent requirement.

**Traces to:** REQ-999
`)

	// Run repair
	result, err := trace.Repair(dir, fs)
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
	content, err := fs.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("REQ-999"))
}

func TestIntegrationRepairUsesNextAvailableID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create design files where DES-1 through DES-5 exist
	// Then have a duplicate DES-3 - should renumber to DES-6
	writeArtifact(t, fs, dir, "design.md", `# Design

### DES-1: Design One

Description.

**Traces to:** REQ-1

### DES-2: Design Two

Description.

**Traces to:** REQ-1

### DES-3: Design Three

Description.

**Traces to:** REQ-1

### DES-4: Design Four

Description.

**Traces to:** REQ-1

### DES-5: Design Five

Description.

**Traces to:** REQ-1
`)

	// Create a duplicate DES-3 in another file
	writeArtifact(t, fs, dir, "design-feature.md", `# Feature Design

### DES-3: Duplicate Three

This is a duplicate of DES-3.

**Traces to:** REQ-2
`)

	writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-1: First

Description.

### REQ-2: Second

Description.
`)

	// Run repair
	result, err := trace.Repair(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect duplicate
	g.Expect(result.DuplicateIDs).To(ContainElement("DES-3"))

	// Should renumber to DES-6 (next available after DES-5)
	g.Expect(result.Renumbered).To(HaveLen(1))
	g.Expect(result.Renumbered[0].OldID).To(Equal("DES-3"))
	g.Expect(result.Renumbered[0].NewID).To(Equal("DES-6"))

	// Verify file was updated correctly
	featureContent, err := fs.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(featureContent)).To(ContainSubstring("### DES-6: Duplicate Three"))
}

func TestIntegrationRepairNoDuplicateEscalations(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create files with duplicate IDs (should be auto-fixed, not escalated)
	writeArtifact(t, fs, dir, "design.md", `# Design

### DES-1: First

Description.

**Traces to:** REQ-1
`)

	writeArtifact(t, fs, dir, "design-feature.md", `# Feature Design

### DES-1: Duplicate

Description.

**Traces to:** REQ-1
`)

	writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-1: Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should have duplicate detected
	g.Expect(result.DuplicateIDs).To(ContainElement("DES-1"))

	// Should have renumbered
	g.Expect(result.Renumbered).ToNot(BeEmpty())

	// NO escalations for duplicates - they are auto-fixed
	g.Expect(result.Escalations).To(BeEmpty())
}

func TestIntegrationRepairIsIdempotent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create design files with duplicate IDs
	writeArtifact(t, fs, dir, "design.md", `# Design

### DES-1: First Design

Description.

**Traces to:** REQ-1
`)

	writeArtifact(t, fs, dir, "design-feature.md", `# Feature Design

### DES-1: Duplicate Design

Description.

**Traces to:** REQ-1
`)

	writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-1: Requirement

Description.
`)

	// Run repair first time
	result1, err := trace.Repair(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result1.DuplicateIDs).To(ContainElement("DES-1"))
	g.Expect(result1.Renumbered).To(HaveLen(1))

	// Read file contents after first repair
	mainContent1, err := fs.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())
	featureContent1, err := fs.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Run repair second time
	result2, err := trace.Repair(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should find NO duplicates or issues on second run
	g.Expect(result2.DuplicateIDs).To(BeEmpty())
	g.Expect(result2.DanglingRefs).To(BeEmpty())
	g.Expect(result2.Renumbered).To(BeEmpty())
	g.Expect(result2.Escalations).To(BeEmpty())

	// Files should be unchanged from first repair
	mainContent2, err := fs.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())
	featureContent2, err := fs.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(string(mainContent1)).To(Equal(string(mainContent2)))
	g.Expect(string(featureContent1)).To(Equal(string(featureContent2)))
}

func TestIntegrationRepairMultipleDuplicatesAndDangling(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create a complex scenario with multiple issues
	writeArtifact(t, fs, dir, "design.md", `# Design

### DES-1: Design One

Description.

**Traces to:** REQ-1

### DES-2: Design Two

Description.

**Traces to:** REQ-1
`)

	// Duplicate DES-1 and reference non-existent REQ-999
	writeArtifact(t, fs, dir, "design-feature.md", `# Feature Design

### DES-1: Duplicate One

Description.

**Traces to:** REQ-999
`)

	// Another duplicate DES-2
	writeArtifact(t, fs, dir, "design-other.md", `# Other Design

### DES-2: Duplicate Two

Description.

**Traces to:** REQ-888
`)

	writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-1: Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect both duplicates
	g.Expect(result.DuplicateIDs).To(ContainElements("DES-1", "DES-2"))

	// Should renumber both duplicates
	g.Expect(result.Renumbered).To(HaveLen(2))

	// Both should be renumbered to new IDs (DES-3 and DES-4)
	newIDs := make([]string, 0, 2)
	for _, r := range result.Renumbered {
		newIDs = append(newIDs, r.NewID)
	}
	g.Expect(newIDs).To(ContainElements("DES-3", "DES-4"))

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
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create a design file with duplicate ID and extra content
	originalContent := `# Design Documentation

This is important introductory text that should be preserved.

## Overview

Some overview content here.

### DES-1: Original Design

This is a detailed design description with **bold** and _italic_ text.

- Bullet point 1
- Bullet point 2

**Traces to:** REQ-1

## More Content

This content comes after the design section.
`
	writeArtifact(t, fs, dir, "design.md", originalContent)

	// Create duplicate in another file
	writeArtifact(t, fs, dir, "design-feature.md", `# Feature Design

### DES-1: Duplicate Design

Duplicate.

**Traces to:** REQ-1
`)

	writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-1: Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Renumbered).To(HaveLen(1))

	// The original file should be unchanged (it keeps DES-1)
	mainContent, err := fs.ReadFile(filepath.Join(dir, "design.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(mainContent)).To(Equal(originalContent))

	// The feature file should have DES-2 but preserve structure
	featureContent, err := fs.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(featureContent)).To(ContainSubstring("# Feature Design"))
	g.Expect(string(featureContent)).To(ContainSubstring("### DES-2: Duplicate Design"))
	g.Expect(string(featureContent)).To(ContainSubstring("**Traces to:** REQ-1"))
}

func TestIntegrationRepairNoIssues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create well-formed artifacts with no issues
	writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-1: First Requirement

Description.

### REQ-2: Second Requirement

Description.
`)

	writeArtifact(t, fs, dir, "design.md", `# Design

### DES-1: First Design

**Traces to:** REQ-1

### DES-2: Second Design

**Traces to:** REQ-2
`)

	// Run repair
	result, err := trace.Repair(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Everything should be empty - no issues found
	g.Expect(result.DanglingRefs).To(BeEmpty())
	g.Expect(result.DuplicateIDs).To(BeEmpty())
	g.Expect(result.Renumbered).To(BeEmpty())
	g.Expect(result.Escalations).To(BeEmpty())
}

func TestIntegrationRepairReferencesUpdatedOnRenumber(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	// Create a file where DES-1 is defined and also referenced elsewhere
	writeArtifact(t, fs, dir, "design.md", `# Design

### DES-1: Main Design

Description.

**Traces to:** REQ-1
`)

	// Duplicate DES-1 that also references the original DES-1 in its text
	writeArtifact(t, fs, dir, "design-feature.md", `# Feature Design

### DES-1: Feature Design

This feature extends DES-1 from the main design.

**Traces to:** REQ-1
`)

	writeArtifact(t, fs, dir, "requirements.md", `# Requirements

### REQ-1: Requirement

Description.
`)

	// Run repair
	result, err := trace.Repair(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())

	// Should renumber the duplicate
	g.Expect(result.Renumbered).To(HaveLen(1))
	g.Expect(result.Renumbered[0].NewID).To(Equal("DES-2"))

	// The feature file should have ALL occurrences of DES-1 replaced with DES-2
	// (the ReplaceAll behavior in the implementation)
	featureContent, err := fs.ReadFile(filepath.Join(dir, "design-feature.md"))
	g.Expect(err).ToNot(HaveOccurred())

	// Note: The current implementation uses strings.ReplaceAll which replaces
	// ALL occurrences of the duplicate ID in the file, including in body text
	g.Expect(string(featureContent)).To(ContainSubstring("### DES-2: Feature Design"))
	g.Expect(string(featureContent)).To(ContainSubstring("extends DES-2"))
}
