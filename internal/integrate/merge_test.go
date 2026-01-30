package integrate_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/integrate"
)

// Mock file system for merge tests
type mockFS struct {
	files map[string]string
}

func (m *mockFS) ReadFile(path string) (string, error) {
	content, exists := m.files[path]
	if !exists {
		return "", &fileNotFoundError{path: path}
	}
	return content, nil
}

func (m *mockFS) WriteFile(path string, content string) error {
	m.files[path] = content
	return nil
}

func (m *mockFS) FileExists(path string) bool {
	_, exists := m.files[path]
	return exists
}

func (m *mockFS) RemoveAll(path string) error {
	// Remove all files under path
	for k := range m.files {
		if len(k) >= len(path) && k[:len(path)] == path {
			delete(m.files, k)
		}
	}
	return nil
}

type fileNotFoundError struct {
	path string
}

func (e *fileNotFoundError) Error() string {
	return "file not found: " + e.path
}

// TEST-220 traces: TASK-016
// Test merge with no ID conflicts appends items with same IDs
func TestMerge_NoConflicts(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/requirements.md": `# Requirements

## REQ-001: Existing Requirement

Existing description.
`,
		"/project/docs/projects/feature/requirements.md": `# Requirements

## REQ-002: New Requirement

New description.
`,
	}}

	result, err := integrate.Merge("/project", "feature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.RequirementsAdded).To(Equal(1))
	g.Expect(result.IDsRenumbered).To(Equal(0))

	// Check merged content
	merged := fs.files["/project/docs/requirements.md"]
	g.Expect(merged).To(ContainSubstring("REQ-001"))
	g.Expect(merged).To(ContainSubstring("REQ-002"))
	g.Expect(merged).To(ContainSubstring("New Requirement"))
}

// TEST-221 traces: TASK-016
// Test merge with ID conflicts renumbers items
func TestMerge_WithConflicts(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/requirements.md": `# Requirements

## REQ-001: First

First description.

## REQ-002: Second

Second description.
`,
		"/project/docs/projects/feature/requirements.md": `# Requirements

## REQ-001: Conflicting ID

This has same ID as existing.
`,
	}}

	result, err := integrate.Merge("/project", "feature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.RequirementsAdded).To(Equal(1))
	g.Expect(result.IDsRenumbered).To(BeNumerically(">", 0))

	// Check merged content - conflicting ID should be renumbered to REQ-003
	merged := fs.files["/project/docs/requirements.md"]
	g.Expect(merged).To(ContainSubstring("REQ-001"))
	g.Expect(merged).To(ContainSubstring("REQ-002"))
	g.Expect(merged).To(ContainSubstring("REQ-003"))
	g.Expect(merged).To(ContainSubstring("Conflicting ID"))
}

// TEST-222 traces: TASK-016
// Test merge updates traceability links after renumbering
func TestMerge_UpdatesTraceability(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/requirements.md": `# Requirements

## REQ-001: Existing

Existing.
`,
		"/project/docs/traceability.toml": `[[links]]
from = "REQ-001"
to = ["DES-001"]
`,
		"/project/docs/projects/feature/requirements.md": `# Requirements

## REQ-001: New Feature

New feature requirement.
`,
		"/project/docs/projects/feature/traceability.toml": `[[links]]
from = "REQ-001"
to = ["DES-002"]
`,
	}}

	result, err := integrate.Merge("/project", "feature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LinksUpdated).To(BeNumerically(">", 0))

	// Check traceability - REQ-001 from feature should be renumbered to REQ-002
	traceability := fs.files["/project/docs/traceability.toml"]
	g.Expect(traceability).To(ContainSubstring("REQ-001"))
	g.Expect(traceability).To(ContainSubstring("REQ-002"))
	g.Expect(traceability).To(ContainSubstring("DES-001"))
	g.Expect(traceability).To(ContainSubstring("DES-002"))
}

// TEST-223 traces: TASK-016
// Test merge returns summary with counts
func TestMerge_ReturnsSummary(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/requirements.md": `# Requirements
`,
		"/project/docs/design.md": `# Design
`,
		"/project/docs/architecture.md": `# Architecture
`,
		"/project/docs/projects/feature/requirements.md": `# Requirements

## REQ-001: New Req

Desc.
`,
		"/project/docs/projects/feature/design.md": `# Design

## DES-001: New Design

Desc.

## DES-002: Another Design

Desc.
`,
		"/project/docs/projects/feature/architecture.md": `# Architecture

## ARCH-001: New Arch

Desc.
`,
	}}

	result, err := integrate.Merge("/project", "feature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.RequirementsAdded).To(Equal(1))
	g.Expect(result.DesignAdded).To(Equal(2))
	g.Expect(result.ArchitectureAdded).To(Equal(1))
	g.Expect(result.Summary).To(ContainSubstring("requirement"))
}

// TEST-224 traces: TASK-016
// Test merge handles missing per-project files gracefully
func TestMerge_MissingPerProjectFiles(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/requirements.md": `# Requirements
`,
		// No per-project files - only requirements exists
		"/project/docs/projects/feature/requirements.md": `# Requirements

## REQ-001: Only Req

Desc.
`,
	}}

	result, err := integrate.Merge("/project", "feature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.RequirementsAdded).To(Equal(1))
	g.Expect(result.DesignAdded).To(Equal(0))
	g.Expect(result.ArchitectureAdded).To(Equal(0))
}

// TEST-225 traces: TASK-016
// Test merge handles empty top-level files
func TestMerge_EmptyTopLevel(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/requirements.md": `# Requirements
`,
		"/project/docs/projects/feature/requirements.md": `# Requirements

## REQ-001: First

First.

## REQ-002: Second

Second.
`,
	}}

	result, err := integrate.Merge("/project", "feature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.RequirementsAdded).To(Equal(2))
	g.Expect(result.IDsRenumbered).To(Equal(0)) // No conflicts with empty

	merged := fs.files["/project/docs/requirements.md"]
	g.Expect(merged).To(ContainSubstring("REQ-001"))
	g.Expect(merged).To(ContainSubstring("REQ-002"))
}
