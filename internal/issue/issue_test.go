package issue_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/issue"
)

func TestCreate_AutoIncrementsID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	iss1, err := issue.Create(dir, issue.CreateOpts{Title: "First"}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(iss1.ID).To(Equal("ISSUE-1"))

	iss2, err := issue.Create(dir, issue.CreateOpts{Title: "Second"}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(iss2.ID).To(Equal("ISSUE-2"))
}

func TestCreate_NewFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	iss, err := issue.Create(dir, issue.CreateOpts{Title: "My new issue"}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(iss.ID).To(Equal("ISSUE-1"))
	g.Expect(iss.Title).To(Equal("My new issue"))
	g.Expect(iss.Status).To(Equal("Open"))
	g.Expect(iss.Priority).To(Equal("Medium")) // default
	g.Expect(iss.Created).To(Equal("2026-02-01"))
}

func TestCreate_WithBody(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	iss, err := issue.Create(dir, issue.CreateOpts{
		Title:    "Issue with body",
		Priority: "High",
		Body:     "Some detailed description",
	}, nowFunc())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(iss.Priority).To(Equal("High"))
	g.Expect(iss.Body).To(Equal("Some detailed description"))
}

func TestGet_Found(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	iss, err := issue.Get(dir, "ISSUE-1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(iss).ToNot(BeNil())

	if iss == nil {
		t.Fatal("expected non-nil issue")
	}

	g.Expect(iss.ID).To(Equal("ISSUE-1"))
	g.Expect(iss.Title).To(Equal("First issue"))
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	_, err := issue.Get(dir, "ISSUE-999")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("not found"))
	}
}

func TestList_All(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	issues, err := issue.List(dir, "")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(issues).To(HaveLen(2))
}

func TestList_FilterByStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	issues, err := issue.List(dir, "open")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(issues).To(HaveLen(1))

	if len(issues) < 1 {
		t.Fatal("expected at least 1 issue")
	}

	g.Expect(issues[0].ID).To(Equal("ISSUE-1"))
}

func TestList_NoFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	issues, err := issue.List(dir, "")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(issues).To(BeEmpty())
}

func TestNextID_EmptyFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	id, err := issue.NextID(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(id).To(Equal("ISSUE-1"))
}

func TestNextID_ExistingIssues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	id, err := issue.NextID(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(id).To(Equal("ISSUE-3"))
}

func TestParseAcceptanceCriteria_NoAC(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	result := issue.ParseAcceptanceCriteria(dir, "ISSUE-1")
	g.Expect(result.Error).To(BeEmpty())
	g.Expect(result.Items).To(BeEmpty())
	g.Expect(result.AllComplete).To(BeTrue())
}

func TestParseAcceptanceCriteria_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	result := issue.ParseAcceptanceCriteria(dir, "ISSUE-999")
	g.Expect(result.Error).ToNot(BeEmpty())
}

func TestParseAcceptanceCriteria_WithItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, issueWithAC)

	result := issue.ParseAcceptanceCriteria(dir, "ISSUE-1")
	g.Expect(result.Error).To(BeEmpty())
	g.Expect(result.Items).To(HaveLen(2))
	g.Expect(result.Complete).To(Equal(1))
	g.Expect(result.Incomplete).To(Equal(1))
	g.Expect(result.AllComplete).To(BeFalse())
}

func TestParseContent_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	issues := issue.ParseContent("")
	g.Expect(issues).To(BeEmpty())
}

func TestParseContent_MultipleIssues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	issues := issue.ParseContent(simpleIssueContent)
	g.Expect(issues).To(HaveLen(2))
}

func TestParseContent_SingleIssue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `### ISSUE-42: Test issue

**Priority:** High
**Status:** Open
**Created:** 2026-02-01
`

	issues := issue.ParseContent(content)
	g.Expect(issues).To(HaveLen(1))

	if len(issues) < 1 {
		t.Fatal("expected at least 1 issue")
	}

	g.Expect(issues[0].ID).To(Equal("ISSUE-42"))
	g.Expect(issues[0].Title).To(Equal("Test issue"))
	g.Expect(issues[0].Priority).To(Equal("High"))
	g.Expect(issues[0].Status).To(Equal("Open"))
}

func TestParse_NoFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	issues, err := issue.Parse(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(issues).To(BeNil())
}

func TestParse_WithFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	issues, err := issue.Parse(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(issues).To(HaveLen(2))
}

func TestUpdate_ClosesWithAllAC(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	allCompleteContent := `# projctl Issues

---

### ISSUE-1: Issue with all AC done

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-01

### Acceptance Criteria

- [x] All done
`
	writeIssuesFile(t, dir, allCompleteContent)

	err := issue.Update(dir, "ISSUE-1", issue.UpdateOpts{Status: "Closed"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestUpdate_ClosingWithIncompleteACFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, issueWithAC)

	err := issue.Update(dir, "ISSUE-1", issue.UpdateOpts{Status: "closed"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("incomplete"))
	}
}

func TestUpdate_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	err := issue.Update(dir, "ISSUE-999", issue.UpdateOpts{Status: "Closed"})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("not found"))
	}
}

func TestUpdate_StatusChange(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	err := issue.Update(dir, "ISSUE-1", issue.UpdateOpts{
		Status: "Closed",
		Force:  true,
	})
	g.Expect(err).ToNot(HaveOccurred())

	iss, err := issue.Get(dir, "ISSUE-1")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(iss).ToNot(BeNil())

	if iss == nil {
		t.Fatal("expected non-nil issue")
	}

	g.Expect(iss.Status).To(Equal("Closed"))
}

func TestUpdate_WithComment(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	err := issue.Update(dir, "ISSUE-1", issue.UpdateOpts{
		Comment: "This is a comment",
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidateClose_AllComplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Issues

---

### ISSUE-1: Complete issue

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-01

### Acceptance Criteria

- [x] First done
- [x] Second done
`
	writeIssuesFile(t, dir, content)

	err := issue.ValidateClose(dir, "ISSUE-1")
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidateClose_Incomplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, issueWithAC)
	err := issue.ValidateClose(dir, "ISSUE-1")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("incomplete"))
	}
}

func TestValidateClose_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	writeIssuesFile(t, dir, simpleIssueContent)

	err := issue.ValidateClose(dir, "ISSUE-999")
	g.Expect(err).To(HaveOccurred())
}

// unexported constants.
const (
	issueWithAC = `# projctl Issues

---

### ISSUE-1: Issue with AC

**Priority:** Medium
**Status:** Open
**Created:** 2026-02-01

### Acceptance Criteria

- [x] First criterion done
- [ ] Second criterion pending
`
	simpleIssueContent = `# projctl Issues

Tracked issues for future work beyond the current task list.

---

### ISSUE-1: First issue

**Priority:** High
**Status:** Open
**Created:** 2026-02-01

---

### ISSUE-2: Second issue

**Priority:** Low
**Status:** Closed
**Created:** 2026-02-01
`
)

// unexported variables.
var (
	fixedTime = time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
)

func nowFunc() func() time.Time {
	return func() time.Time { return fixedTime }
}

func writeIssuesFile(t *testing.T, dir, content string) {
	t.Helper()

	docsDir := filepath.Join(dir, "docs")

	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(docsDir, "issues.md")

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
