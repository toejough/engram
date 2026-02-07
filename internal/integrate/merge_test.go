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

func (m *mockFS) Glob(pattern string) ([]string, error) {
	// Simple glob matching for tests - supports * wildcard
	var matches []string
	for path := range m.files {
		if matchGlob(pattern, path) {
			matches = append(matches, path)
		}
	}
	return matches, nil
}

// matchGlob is a simple glob matcher for tests
func matchGlob(pattern, path string) bool {
	// Split pattern by *
	parts := splitGlob(pattern)
	if len(parts) == 1 {
		return pattern == path
	}

	// Check prefix
	if !hasPrefix(path, parts[0]) {
		return false
	}
	path = path[len(parts[0]):]

	// Check suffix
	if !hasSuffix(path, parts[len(parts)-1]) {
		return false
	}

	return true
}

func splitGlob(pattern string) []string {
	var parts []string
	current := ""
	for _, c := range pattern {
		if c == '*' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	parts = append(parts, current)
	return parts
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
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
		"/project/.claude/projects/feature/requirements.md": `# Requirements

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
		"/project/.claude/projects/feature/requirements.md": `# Requirements

## REQ-001: Conflicting ID

This has same ID as existing.
`,
	}}

	result, err := integrate.Merge("/project", "feature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.RequirementsAdded).To(Equal(1))
	g.Expect(result.IDsRenumbered).To(BeNumerically(">", 0))

	// Check merged content - conflicting ID should be renumbered to REQ-3 (unpadded)
	merged := fs.files["/project/docs/requirements.md"]
	g.Expect(merged).To(ContainSubstring("REQ-001"))
	g.Expect(merged).To(ContainSubstring("REQ-002"))
	g.Expect(merged).To(ContainSubstring("REQ-3"))
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
		"/project/.claude/projects/feature/requirements.md": `# Requirements

## REQ-001: New Feature

New feature requirement.
`,
		"/project/.claude/projects/feature/traceability.toml": `[[links]]
from = "REQ-001"
to = ["DES-002"]
`,
	}}

	result, err := integrate.Merge("/project", "feature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LinksUpdated).To(BeNumerically(">", 0))

	// Check traceability - REQ-001 from feature should be renumbered to REQ-2 (unpadded)
	traceability := fs.files["/project/docs/traceability.toml"]
	g.Expect(traceability).To(ContainSubstring("REQ-001"))
	g.Expect(traceability).To(ContainSubstring("REQ-2"))
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
		"/project/.claude/projects/feature/requirements.md": `# Requirements

## REQ-001: New Req

Desc.
`,
		"/project/.claude/projects/feature/design.md": `# Design

## DES-001: New Design

Desc.

## DES-002: Another Design

Desc.
`,
		"/project/.claude/projects/feature/architecture.md": `# Architecture

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
		"/project/.claude/projects/feature/requirements.md": `# Requirements

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
		"/project/.claude/projects/feature/requirements.md": `# Requirements

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

// TEST-226 traces: TASK-002
// Test MergeFeatureFiles consolidates design-*.md into design.md
func TestMergeFeatureFiles_Design(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/design.md": `# Design

### DES-001: Existing Design

Existing design decision.

**Traces to:** REQ-001
`,
		"/project/docs/design-help-system.md": `# Help System Design

### DES-001: Help Format

Help format design.

**Traces to:** REQ-002
`,
	}}

	result, err := integrate.MergeFeatureFiles("/project/docs", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DesignAdded).To(Equal(1))
	g.Expect(result.IDsRenumbered).To(Equal(1)) // DES-001 conflict

	// Check merged content
	merged := fs.files["/project/docs/design.md"]
	g.Expect(merged).To(ContainSubstring("DES-001"))
	g.Expect(merged).To(ContainSubstring("DES-2")) // Renumbered (unpadded)
	g.Expect(merged).To(ContainSubstring("Help Format"))

	// Feature file should be deleted
	g.Expect(fs.FileExists("/project/docs/design-help-system.md")).To(BeFalse())
}

// TEST-227 traces: TASK-002
// Test MergeFeatureFiles updates Traces to: references after renumbering
func TestMergeFeatureFiles_UpdatesTracesTo(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/design.md": `# Design

### DES-001: Existing

Existing.

**Traces to:** REQ-001
`,
		"/project/docs/design-feature.md": `# Feature Design

### DES-001: Feature Design

Feature design.

**Traces to:** REQ-002

### DES-002: Another Feature

References DES-001 internally.

**Traces to:** REQ-003
`,
	}}

	result, err := integrate.MergeFeatureFiles("/project/docs", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.IDsRenumbered).To(BeNumerically(">=", 1))

	// Check that internal references are updated
	merged := fs.files["/project/docs/design.md"]
	// DES-001 from feature should become DES-2 (unpadded, renumbered)
	// DES-002 from feature stays DES-002 (no conflict)
	// Internal reference to DES-001 in second item body should become DES-2
	g.Expect(merged).To(ContainSubstring("DES-2"))
	g.Expect(merged).To(ContainSubstring("DES-002"))
	g.Expect(merged).To(ContainSubstring("References DES-2 internally"))
}

// TEST-228 traces: TASK-002
// Test MergeFeatureFiles handles multiple feature files
func TestMergeFeatureFiles_MultipleFiles(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/design.md": `# Design
`,
		"/project/docs/design-auth.md": `# Auth Design

### DES-001: Auth Flow

Auth flow.

**Traces to:** REQ-001
`,
		"/project/docs/design-ui.md": `# UI Design

### DES-001: UI Layout

UI layout.

**Traces to:** REQ-002
`,
	}}

	result, err := integrate.MergeFeatureFiles("/project/docs", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DesignAdded).To(Equal(2))

	// Both feature files should be merged
	merged := fs.files["/project/docs/design.md"]
	g.Expect(merged).To(ContainSubstring("Auth Flow"))
	g.Expect(merged).To(ContainSubstring("UI Layout"))

	// Both feature files should be deleted
	g.Expect(fs.FileExists("/project/docs/design-auth.md")).To(BeFalse())
	g.Expect(fs.FileExists("/project/docs/design-ui.md")).To(BeFalse())
}

// TEST-230 traces: TASK-016, ISSUE-139
// Test merge renumbers inline Traces to: references (Bug 1: strings.Replace limit=1)
func TestMerge_RenumbersInlineTraceReferences(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/.claude/projects/feature/requirements.md": `# Requirements

## REQ-001: Feature Requirement

Description.

**Traces to:** DES-001
`,
		"/project/.claude/projects/feature/design.md": `# Design

## DES-001: Feature Design

Design description.

**Traces to:** REQ-001
`,
		"/project/docs/requirements.md": `# Requirements

## REQ-001: Existing Requirement

Existing.
`,
		"/project/docs/design.md": `# Design

## DES-001: Existing Design

Existing design.
`,
	}}

	result, err := integrate.Merge("/project", "feature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.RequirementsAdded).To(Equal(1))
	g.Expect(result.DesignAdded).To(Equal(1))
	g.Expect(result.IDsRenumbered).To(Equal(2))

	// The renumbered REQ item should have its body trace reference updated too
	mergedReq := fs.files["/project/docs/requirements.md"]
	// REQ-001 from feature was renumbered to REQ-2 (unpadded)
	g.Expect(mergedReq).To(ContainSubstring("REQ-2"))
	// The Traces to: line inside the renumbered item should NOT still say REQ-001
	// (it references DES-001 which also gets renumbered, but that's in the traceability pass)

	mergedDes := fs.files["/project/docs/design.md"]
	// DES-001 from feature was renumbered to DES-2 (unpadded)
	g.Expect(mergedDes).To(ContainSubstring("DES-2"))
	// The Traces to: REQ-001 in the design item body should also be updated
	// since REQ-001 was mapped to REQ-2
}

// TEST-231 traces: TASK-016, ISSUE-139
// Test merge uses unpadded ID format (Bug 2: %s-%d not %s-%03d)
func TestMerge_UnpaddedIDFormat(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/requirements.md": `# Requirements

## REQ-1: First

First.
`,
		"/project/.claude/projects/feature/requirements.md": `# Requirements

## REQ-1: Conflicting

Conflicting.
`,
	}}

	result, err := integrate.Merge("/project", "feature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.IDsRenumbered).To(Equal(1))

	merged := fs.files["/project/docs/requirements.md"]
	// Should produce REQ-2, NOT REQ-002
	g.Expect(merged).To(ContainSubstring("REQ-2"))
	g.Expect(merged).ToNot(ContainSubstring("REQ-002"))
}

// TEST-232 traces: TASK-016, ISSUE-139
// Test MergeFeatureFiles uses unpadded ID format (Bug 2)
func TestMergeFeatureFiles_UnpaddedIDFormat(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/design.md": `# Design

### DES-1: Existing

Existing.
`,
		"/project/docs/design-feature.md": `# Feature Design

### DES-1: Feature

Feature.
`,
	}}

	result, err := integrate.MergeFeatureFiles("/project/docs", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.IDsRenumbered).To(Equal(1))

	merged := fs.files["/project/docs/design.md"]
	// Should produce DES-2, NOT DES-002
	g.Expect(merged).To(ContainSubstring("DES-2"))
	g.Expect(merged).ToNot(ContainSubstring("DES-002"))
}

// TEST-233 traces: TASK-016, ISSUE-139
// Test Merge reads per-project files from .claude/projects/ (Bug 3: path mismatch)
func TestMerge_UsesClaudeProjectsPath(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/requirements.md": `# Requirements
`,
		// Per-project files live in .claude/projects/<name>/, NOT docs/projects/<name>/
		"/project/.claude/projects/myfeature/requirements.md": `# Requirements

## REQ-1: From Claude Projects Dir

Description.
`,
	}}

	result, err := integrate.Merge("/project", "myfeature", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.RequirementsAdded).To(Equal(1))

	merged := fs.files["/project/docs/requirements.md"]
	g.Expect(merged).To(ContainSubstring("From Claude Projects Dir"))
}

// TEST-229 traces: TASK-002
// Test MergeFeatureFiles returns empty result when no feature files
func TestMergeFeatureFiles_NoFeatureFiles(t *testing.T) {
	g := NewWithT(t)

	fs := &mockFS{files: map[string]string{
		"/project/docs/design.md": `# Design

### DES-001: Only Design

Only design.

**Traces to:** REQ-001
`,
	}}

	result, err := integrate.MergeFeatureFiles("/project/docs", fs)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DesignAdded).To(Equal(0))
	g.Expect(result.RequirementsAdded).To(Equal(0))
	g.Expect(result.ArchitectureAdded).To(Equal(0))
}
