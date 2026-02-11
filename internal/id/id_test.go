package id_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/id"
)

// TEST-161 traces: TASK-003
// Test Next returns TYPE-1 when no files exist
func TestNext_NoFiles_ReturnsFirst(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-1"))
}

// TEST-162 traces: TASK-003
// Test Next returns TYPE-1 when files exist but have no IDs
func TestNext_FilesWithoutIDs_ReturnsFirst(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Create a markdown file without any IDs
	err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# README\nNo IDs here"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-1"))
}

// TEST-163 traces: TASK-003
// Test Next finds ID in root markdown file
func TestNext_FindsIDInRoot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	content := `# Requirements

## REQ-001: First requirement

Some content

## REQ-002: Second requirement

More content
`
	err := os.WriteFile(filepath.Join(dir, "requirements.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-3"))
}

// TEST-164 traces: TASK-003
// Test Next finds ID in docs/ subdirectory
func TestNext_FindsIDInDocsSubdir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	err := os.MkdirAll(docsDir, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	content := `# Architecture

## ARCH-005: Component design

Some content
`
	err = os.WriteFile(filepath.Join(docsDir, "architecture.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "ARCH")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("ARCH-6"))
}

// TEST-165 traces: TASK-003
// Test Next scans both root and docs/ to find max
func TestNext_ScansRootAndDocs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	err := os.MkdirAll(docsDir, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	// Root has REQ-003
	rootContent := `# Requirements

## REQ-003: Root requirement
`
	err = os.WriteFile(filepath.Join(dir, "requirements.md"), []byte(rootContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Docs has REQ-007
	docsContent := `# More Requirements

## REQ-007: Docs requirement
`
	err = os.WriteFile(filepath.Join(docsDir, "more-reqs.md"), []byte(docsContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-8"))
}

// TEST-166 traces: TASK-003
// Test Next handles DES prefix
func TestNext_DESPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	content := `# Design

## DES-010: Design decision
`
	err := os.WriteFile(filepath.Join(dir, "design.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "DES")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("DES-11"))
}

// TEST-167 traces: TASK-003
// Test Next handles TASK prefix
func TestNext_TASKPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	content := `# Tasks

## TASK-042: Do something
`
	err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "TASK")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("TASK-43"))
}

// TEST-168 traces: TASK-003
// Test Next handles ISSUE prefix
func TestNext_ISSUEPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	content := `# Issues

## ISSUE-99: Bug report
`
	err := os.WriteFile(filepath.Join(dir, "issues.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "ISSUE")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("ISSUE-100"))
}

// TEST-169 traces: TASK-003
// Test Next only counts IDs of the requested type
func TestNext_OnlyCountsRequestedType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	content := `# Mixed Artifacts

## REQ-050: A requirement

## TASK-100: A task

## ARCH-025: An architecture
`
	err := os.WriteFile(filepath.Join(dir, "artifacts.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	reqResult, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(reqResult).To(Equal("REQ-51"))

	taskResult, err := id.Next(dir, "TASK")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(taskResult).To(Equal("TASK-101"))

	archResult, err := id.Next(dir, "ARCH")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(archResult).To(Equal("ARCH-26"))
}

// TEST-170 traces: TASK-003
// Test Next finds IDs in various markdown patterns
func TestNext_VariousMarkdownPatterns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// IDs can appear in various contexts
	content := `# Document

## REQ-001: Header format

REQ-002 mentioned inline

**Traces to:** REQ-003

- REQ-004 in list

| ID | Title |
| REQ-005 | Table |

`
	err := os.WriteFile(filepath.Join(dir, "doc.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-6"))
}

// TEST-171 traces: TASK-003
// Test Next ignores non-markdown files
func TestNext_IgnoresNonMarkdown(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Put a high ID in a non-markdown file
	err := os.WriteFile(filepath.Join(dir, "data.txt"), []byte("REQ-999: ignored"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Put a lower ID in markdown
	err = os.WriteFile(filepath.Join(dir, "reqs.md"), []byte("## REQ-001: counted"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-2")) // Should ignore .txt file
}

// TEST-172 traces: TASK-003
// Test Next handles IDs with more than 3 digits
func TestNext_HandlesLargeNumbers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	content := `# Tasks

## TASK-1234: Large number task
`
	err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "TASK")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("TASK-1235"))
}

// TEST-173 traces: TASK-003
// Test Next generates simple incrementing numbers
func TestNext_GeneratesIncrementingNumbers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// When starting fresh, should get 1
	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-1"))

	// Create REQ-9
	content := `## REQ-9: Ninth requirement`
	err = os.WriteFile(filepath.Join(dir, "reqs.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err = id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-10"))

	// Create REQ-99
	content = `## REQ-99: 99th requirement`
	err = os.WriteFile(filepath.Join(dir, "reqs.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err = id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-100"))
}

// TEST-174 traces: TASK-003
// Property: Next always returns valid ID format
func TestNext_PropertyValidFormat(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		dir := t.TempDir()

		idType := rapid.SampledFrom([]string{"REQ", "DES", "ARCH", "TASK", "ISSUE"}).Draw(rt, "idType")

		// Optionally create some existing IDs
		numExisting := rapid.IntRange(0, 10).Draw(rt, "numExisting")
		if numExisting > 0 {
			var content string
			for i := 1; i <= numExisting; i++ {
				content += "## " + idType + "-" + itoa(i) + ": Item\n\n"
			}
			err := os.WriteFile(filepath.Join(dir, "artifacts.md"), []byte(content), 0o644)
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := id.Next(dir, idType)
		g.Expect(err).ToNot(HaveOccurred())

		// Should match PREFIX-N format (any number of digits)
		g.Expect(result).To(MatchRegexp(`^` + idType + `-\d+$`))
	})
}

// TEST-175 traces: TASK-003
// Property: Next returns number one greater than max existing
func TestNext_PropertyIncrements(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		dir := t.TempDir()

		idType := rapid.SampledFrom([]string{"REQ", "DES", "ARCH", "TASK", "ISSUE"}).Draw(rt, "idType")
		maxExisting := rapid.IntRange(1, 500).Draw(rt, "maxExisting")

		// Create files with IDs up to maxExisting (not necessarily all of them)
		content := "## " + idType + "-" + itoa(maxExisting) + ": Max item\n"
		err := os.WriteFile(filepath.Join(dir, "artifacts.md"), []byte(content), 0o644)
		g.Expect(err).ToNot(HaveOccurred())

		result, err := id.Next(dir, idType)
		g.Expect(err).ToNot(HaveOccurred())

		expected := idType + "-" + itoa(maxExisting+1)
		g.Expect(result).To(Equal(expected))
	})
}

// TEST-176 traces: TASK-003
// Test Next returns error for invalid prefix
func TestNext_InvalidPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	_, err := id.Next(dir, "INVALID")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid"))
}

// TEST-177 traces: TASK-003
// Test Next returns error for empty prefix
func TestNext_EmptyPrefix(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	_, err := id.Next(dir, "")
	g.Expect(err).To(HaveOccurred())
}

// TEST-178 traces: TASK-001
// Test Next generates simple numbers (REQ-1, REQ-2, ...) not zero-padded (REQ-001)
func TestNext_GeneratesSimpleNumbers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// First ID should be REQ-1
	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-1"))

	// Create REQ-9
	content := `## REQ-9: Ninth requirement`
	err = os.WriteFile(filepath.Join(dir, "reqs.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err = id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-10"))
}

// TEST-179 traces: TASK-001
// Test Next scans for \d+ pattern (any number of digits)
func TestNext_ScansAnyNumberDigits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Create files with 1, 2, 3, 4, and 5 digit IDs
	content := `# Requirements
## REQ-1: One digit
## REQ-99: Two digits
## REQ-500: Three digits
## REQ-1234: Four digits
## REQ-99999: Five digits
`
	err := os.WriteFile(filepath.Join(dir, "reqs.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-100000"))
}

// TEST-180 traces: TASK-001
// Test Next is backward compatible with existing 3-digit zero-padded IDs
func TestNext_BackwardCompatibleWithPaddedIDs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Create old-style zero-padded IDs
	content := `# Requirements
## REQ-001: First
## REQ-002: Second
## REQ-042: Forty-second
`
	err := os.WriteFile(filepath.Join(dir, "reqs.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Should recognize 042 as 42 and return next as 43 (without padding)
	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-43"))
}

// TEST-181 traces: TASK-001
// Test Next handles mix of padded and unpadded IDs
func TestNext_HandlesMixedFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Mix of old padded and new unpadded format
	content := `# Requirements
## REQ-001: Old style
## REQ-5: New style
## REQ-099: Old style
## REQ-200: Could be either
`
	err := os.WriteFile(filepath.Join(dir, "reqs.md"), []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := id.Next(dir, "REQ")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("REQ-201"))
}

// itoa converts int to string for test helpers
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
