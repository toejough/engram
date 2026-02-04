package issue_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/issue"
)

func fixedTime() time.Time {
	return time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
}

func nowFunc() func() time.Time {
	return func() time.Time { return fixedTime() }
}

func TestParseContent(t *testing.T) {
	t.Run("parses issue from markdown", func(t *testing.T) {
		g := NewWithT(t)
		content := `# Issues

---

## ISSUE-001: Test Issue

**Priority:** High
**Status:** Open
**Created:** 2026-02-03

### Summary

This is a test issue.
`
		issues := issue.ParseContent(content)
		g.Expect(issues).To(HaveLen(1))
		g.Expect(issues[0].ID).To(Equal("ISSUE-001"))
		g.Expect(issues[0].Title).To(Equal("Test Issue"))
		g.Expect(issues[0].Priority).To(Equal("High"))
		g.Expect(issues[0].Status).To(Equal("Open"))
		g.Expect(issues[0].Created).To(Equal("2026-02-03"))
		g.Expect(issues[0].Body).To(ContainSubstring("This is a test issue"))
	})

	t.Run("parses multiple issues", func(t *testing.T) {
		g := NewWithT(t)
		content := `# Issues

## ISSUE-001: First

**Priority:** High
**Status:** Open
**Created:** 2026-02-01

---

## ISSUE-002: Second

**Priority:** Low
**Status:** Closed
**Created:** 2026-02-02
`
		issues := issue.ParseContent(content)
		g.Expect(issues).To(HaveLen(2))
		g.Expect(issues[0].ID).To(Equal("ISSUE-001"))
		g.Expect(issues[1].ID).To(Equal("ISSUE-002"))
		g.Expect(issues[1].Status).To(Equal("Closed"))
	})
}

func TestCreate(t *testing.T) {
	t.Run("creates issue in new file", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create docs directory
		g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())

		i, err := issue.Create(dir, issue.CreateOpts{
			Title:    "Test Issue",
			Priority: "High",
		}, nowFunc())

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(i.ID).To(Equal("ISSUE-001"))
		g.Expect(i.Title).To(Equal("Test Issue"))
		g.Expect(i.Priority).To(Equal("High"))
		g.Expect(i.Status).To(Equal("Open"))

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(dir, issue.IssuesFile))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).To(ContainSubstring("ISSUE-001"))
		g.Expect(string(content)).To(ContainSubstring("Test Issue"))
	})

	t.Run("increments ID from existing issues", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create initial issues file
		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-005: Existing

**Priority:** Medium
**Status:** Open
**Created:** 2026-01-01
`), 0o644)).To(Succeed())

		i, err := issue.Create(dir, issue.CreateOpts{
			Title: "New Issue",
		}, nowFunc())

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(i.ID).To(Equal("ISSUE-006"))
	})

	t.Run("defaults priority to Medium", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())

		i, err := issue.Create(dir, issue.CreateOpts{
			Title: "Test",
		}, nowFunc())

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(i.Priority).To(Equal("Medium"))
	})
}

func TestUpdate(t *testing.T) {
	t.Run("updates status", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create initial issues file
		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: Test

**Priority:** High
**Status:** Open
**Created:** 2026-01-01
`), 0o644)).To(Succeed())

		err := issue.Update(dir, "ISSUE-001", issue.UpdateOpts{
			Status: "Closed",
		})

		g.Expect(err).ToNot(HaveOccurred())

		content, err := os.ReadFile(filepath.Join(docsDir, "issues.md"))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(content)).To(ContainSubstring("**Status:** Closed"))
	})

	t.Run("returns error for unknown issue", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues
`), 0o644)).To(Succeed())

		err := issue.Update(dir, "ISSUE-999", issue.UpdateOpts{
			Status: "Closed",
		})

		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("not found"))
	})
}

func TestList(t *testing.T) {
	t.Run("filters by status", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: Open One

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

## ISSUE-002: Closed One

**Priority:** High
**Status:** Closed
**Created:** 2026-01-01

## ISSUE-003: Open Two

**Priority:** High
**Status:** Open
**Created:** 2026-01-01
`), 0o644)).To(Succeed())

		issues, err := issue.List(dir, "Open")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(issues).To(HaveLen(2))
		g.Expect(issues[0].ID).To(Equal("ISSUE-001"))
		g.Expect(issues[1].ID).To(Equal("ISSUE-003"))
	})
}

func TestParseAcceptanceCriteria(t *testing.T) {
	t.Run("parses AC from issue body", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: Test Issue

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

### Summary

This is a test issue.

### Acceptance Criteria

- [x] First AC item is complete
- [ ] Second AC item is incomplete
- [x] Third AC item is complete

**Traces to:** something
`), 0o644)).To(Succeed())

		result := issue.ParseAcceptanceCriteria(dir, "ISSUE-001")
		g.Expect(result.Error).To(BeEmpty())
		g.Expect(result.Items).To(HaveLen(3))
		g.Expect(result.Complete).To(Equal(2))
		g.Expect(result.Incomplete).To(Equal(1))
		g.Expect(result.AllComplete).To(BeFalse())
		g.Expect(result.Items[0].Text).To(Equal("First AC item is complete"))
		g.Expect(result.Items[0].Complete).To(BeTrue())
		g.Expect(result.Items[1].Text).To(Equal("Second AC item is incomplete"))
		g.Expect(result.Items[1].Complete).To(BeFalse())
	})

	t.Run("returns AllComplete true when all AC are checked", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: All Done

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

### Acceptance Criteria

- [x] One complete
- [x] Two complete
`), 0o644)).To(Succeed())

		result := issue.ParseAcceptanceCriteria(dir, "ISSUE-001")
		g.Expect(result.Error).To(BeEmpty())
		g.Expect(result.AllComplete).To(BeTrue())
		g.Expect(result.Complete).To(Equal(2))
		g.Expect(result.Incomplete).To(Equal(0))
	})

	t.Run("handles issue with no AC section gracefully", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: No AC Issue

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

### Summary

No acceptance criteria defined.
`), 0o644)).To(Succeed())

		result := issue.ParseAcceptanceCriteria(dir, "ISSUE-001")
		g.Expect(result.Error).To(BeEmpty())
		g.Expect(result.Items).To(BeEmpty())
		g.Expect(result.AllComplete).To(BeTrue()) // No AC = vacuously complete
	})

	t.Run("returns error for unknown issue", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues
`), 0o644)).To(Succeed())

		result := issue.ParseAcceptanceCriteria(dir, "ISSUE-999")
		g.Expect(result.Error).To(ContainSubstring("not found"))
	})

	t.Run("parses AC with bold field format", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: Test Issue

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

**Problem:** Some problem description.

**Acceptance Criteria:**
- [x] First AC item is complete
- [ ] Second AC item is incomplete

**Traces to:** something
`), 0o644)).To(Succeed())

		result := issue.ParseAcceptanceCriteria(dir, "ISSUE-001")
		g.Expect(result.Error).To(BeEmpty())
		g.Expect(result.Items).To(HaveLen(2))
		g.Expect(result.Complete).To(Equal(1))
		g.Expect(result.Incomplete).To(Equal(1))
		g.Expect(result.AllComplete).To(BeFalse())
	})
}

func TestValidateClose(t *testing.T) {
	t.Run("rejects closure when AC incomplete", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: Test Issue

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

### Acceptance Criteria

- [x] First AC complete
- [ ] Second AC incomplete
`), 0o644)).To(Succeed())

		err := issue.ValidateClose(dir, "ISSUE-001")
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("incomplete"))
		g.Expect(err.Error()).To(ContainSubstring("Second AC incomplete"))
	})

	t.Run("allows closure when all AC complete", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: Test Issue

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

### Acceptance Criteria

- [x] First AC complete
- [x] Second AC complete
`), 0o644)).To(Succeed())

		err := issue.ValidateClose(dir, "ISSUE-001")
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("allows closure when no AC defined", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: No AC Issue

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

### Summary

No acceptance criteria.
`), 0o644)).To(Succeed())

		err := issue.ValidateClose(dir, "ISSUE-001")
		g.Expect(err).ToNot(HaveOccurred())
	})
}

func TestUpdateWithValidation(t *testing.T) {
	t.Run("rejects status Closed when AC incomplete", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: Test Issue

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

### Acceptance Criteria

- [ ] Incomplete AC
`), 0o644)).To(Succeed())

		err := issue.Update(dir, "ISSUE-001", issue.UpdateOpts{
			Status: "Closed",
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("incomplete"))

		// Verify status was NOT changed
		content, _ := os.ReadFile(filepath.Join(docsDir, "issues.md"))
		g.Expect(string(content)).To(ContainSubstring("**Status:** Open"))
	})

	t.Run("allows status Closed with --force when AC incomplete", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: Test Issue

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

### Acceptance Criteria

- [ ] Incomplete AC
`), 0o644)).To(Succeed())

		err := issue.Update(dir, "ISSUE-001", issue.UpdateOpts{
			Status: "Closed",
			Force:  true,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Verify status was changed
		content, _ := os.ReadFile(filepath.Join(docsDir, "issues.md"))
		g.Expect(string(content)).To(ContainSubstring("**Status:** Closed"))
	})

	t.Run("allows non-Closed status changes without AC check", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: Test Issue

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

### Acceptance Criteria

- [ ] Incomplete AC
`), 0o644)).To(Succeed())

		err := issue.Update(dir, "ISSUE-001", issue.UpdateOpts{
			Status: "In Progress",
		})
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("allows comment-only updates without AC check", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		docsDir := filepath.Join(dir, "docs")
		g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(docsDir, "issues.md"), []byte(`# Issues

## ISSUE-001: Test Issue

**Priority:** High
**Status:** Open
**Created:** 2026-01-01

### Acceptance Criteria

- [ ] Incomplete AC
`), 0o644)).To(Succeed())

		err := issue.Update(dir, "ISSUE-001", issue.UpdateOpts{
			Comment: "Adding a comment",
		})
		g.Expect(err).ToNot(HaveOccurred())
	})
}
