package checker_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/checker"
)

func TestAcceptanceCriteriaComplete_AllComplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	tasksContent := `# Tasks

### TASK-1: Some task

**Acceptance Criteria:**
- [x] First criterion
- [x] Second criterion
`
	writeFile(t, dir, "tasks.md", tasksContent)

	c := &checker.DefaultChecker{}
	result := c.AcceptanceCriteriaComplete(dir, "TASK-1")
	g.Expect(result).To(BeTrue())
}

func TestAcceptanceCriteriaComplete_EmptyTaskID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := &checker.DefaultChecker{}
	result := c.AcceptanceCriteriaComplete("", "")
	g.Expect(result).To(BeTrue())
}

func TestAcceptanceCriteriaComplete_NoTasksFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	result := c.AcceptanceCriteriaComplete(dir, "TASK-1")
	g.Expect(result).To(BeFalse())
}

func TestAcceptanceCriteriaComplete_NotAllComplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	tasksContent := `# Tasks

### TASK-1: Some task

**Acceptance Criteria:**
- [x] First criterion
- [ ] Second criterion
`
	writeFile(t, dir, "tasks.md", tasksContent)

	c := &checker.DefaultChecker{}
	result := c.AcceptanceCriteriaComplete(dir, "TASK-1")
	g.Expect(result).To(BeFalse())
}

func TestAcceptanceCriteriaComplete_TaskNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "tasks.md", "# Tasks\n\n### TASK-1: Some task\n\n**Acceptance Criteria:**\n- [x] Done\n")

	c := &checker.DefaultChecker{}
	result := c.AcceptanceCriteriaComplete(dir, "TASK-999")
	g.Expect(result).To(BeTrue())
}

func TestDesignExists_Missing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	g.Expect(c.DesignExists(dir)).To(BeFalse())
}

func TestDesignExists_Present(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "design.md", "# Design")

	c := &checker.DefaultChecker{}
	g.Expect(c.DesignExists(dir)).To(BeTrue())
}

func TestDesignHasIDs_Missing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	g.Expect(c.DesignHasIDs(dir)).To(BeFalse())
}

func TestDesignHasIDs_NoIDs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "design.md", "# Design\n\nNo identifiers here.")

	c := &checker.DefaultChecker{}
	g.Expect(c.DesignHasIDs(dir)).To(BeFalse())
}

func TestDesignHasIDs_WithIDs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "design.md", "# Design\n\n### DES-001: Something")

	c := &checker.DefaultChecker{}
	g.Expect(c.DesignHasIDs(dir)).To(BeTrue())
}

func TestIncompleteAcceptanceCriteria_AllComplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "tasks.md", "# Tasks\n\n### TASK-1: Task\n\n**Acceptance Criteria:**\n- [x] Done\n")

	c := &checker.DefaultChecker{}
	result := c.IncompleteAcceptanceCriteria(dir, "TASK-1")
	g.Expect(result).To(BeEmpty())
}

func TestIncompleteAcceptanceCriteria_EmptyTaskID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := &checker.DefaultChecker{}
	result := c.IncompleteAcceptanceCriteria("", "")
	g.Expect(result).To(BeNil())
}

func TestIncompleteAcceptanceCriteria_NoFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	result := c.IncompleteAcceptanceCriteria(dir, "TASK-1")
	g.Expect(result).To(BeNil())
}

func TestIncompleteAcceptanceCriteria_SomeIncomplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "tasks.md", "# Tasks\n\n### TASK-1: Task\n\n**Acceptance Criteria:**\n- [x] Done\n- [ ] Not done\n")

	c := &checker.DefaultChecker{}
	result := c.IncompleteAcceptanceCriteria(dir, "TASK-1")

	if len(result) < 1 {
		t.Fatal("expected at least 1 incomplete item")
	}

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(ContainSubstring("Not done"))
}

func TestIncompleteIssueAC_AllComplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")

	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("failed to create docs dir: %v", err)
	}

	issuesContent := "### ISSUE-1: Test Issue\n\n**Priority:** Medium\n**Status:** Open\n**Created:** 2026-01-01\n\n**Acceptance Criteria:**\n- [x] Done item\n- [x] Also done\n"
	writeFile(t, docsDir, "issues.md", issuesContent)

	c := &checker.DefaultChecker{}
	result := c.IncompleteIssueAC(dir, "ISSUE-1")
	g.Expect(result).To(BeEmpty())
}

func TestIncompleteIssueAC_EmptyIssueID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := &checker.DefaultChecker{}
	result := c.IncompleteIssueAC("", "")
	g.Expect(result).To(BeNil())
}

func TestIncompleteIssueAC_NoIssuesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	result := c.IncompleteIssueAC(dir, "ISSUE-1")
	g.Expect(result).To(BeNil())
}

func TestIncompleteIssueAC_WithIncomplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")

	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("failed to create docs dir: %v", err)
	}

	issuesContent := "### ISSUE-1: Test Issue\n\n**Priority:** Medium\n**Status:** Open\n**Created:** 2026-01-01\n\n**Acceptance Criteria:**\n- [x] Done item\n- [ ] Not done item\n"
	writeFile(t, docsDir, "issues.md", issuesContent)

	c := &checker.DefaultChecker{}
	result := c.IncompleteIssueAC(dir, "ISSUE-1")

	if len(result) < 1 {
		t.Fatal("expected at least 1 incomplete item")
	}

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(ContainSubstring("Not done item"))
}

func TestIssueACComplete_EmptyIssueID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := &checker.DefaultChecker{}
	result := c.IssueACComplete("", "")
	g.Expect(result).To(BeTrue())
}

func TestIssueACComplete_NoIssuesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	// Error contains "not found" → returns true (treats missing issue as complete)
	result := c.IssueACComplete(dir, "ISSUE-1")
	g.Expect(result).To(BeTrue())
}

func TestRequirementsExist_Missing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	g.Expect(c.RequirementsExist(dir)).To(BeFalse())
}

func TestRequirementsExist_Present(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "requirements.md", "# Requirements")

	c := &checker.DefaultChecker{}
	g.Expect(c.RequirementsExist(dir)).To(BeTrue())
}

func TestRequirementsHaveIDs_Missing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	g.Expect(c.RequirementsHaveIDs(dir)).To(BeFalse())
}

func TestRequirementsHaveIDs_NoIDs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "requirements.md", "# Requirements\n\nNo identifiers.")

	c := &checker.DefaultChecker{}
	g.Expect(c.RequirementsHaveIDs(dir)).To(BeFalse())
}

func TestRequirementsHaveIDs_WithIDs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "requirements.md", "# Requirements\n\n### REQ-001: Something")

	c := &checker.DefaultChecker{}
	g.Expect(c.RequirementsHaveIDs(dir)).To(BeTrue())
}

func TestRetroExists_Missing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	g.Expect(c.RetroExists(dir)).To(BeFalse())
}

func TestRetroExists_Present(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "retro.md", "# Retro")

	c := &checker.DefaultChecker{}
	g.Expect(c.RetroExists(dir)).To(BeTrue())
}

func TestSummaryExists_Missing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	g.Expect(c.SummaryExists(dir)).To(BeFalse())
}

func TestSummaryExists_Present(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	writeFile(t, dir, "summary.md", "# Summary")

	c := &checker.DefaultChecker{}
	g.Expect(c.SummaryExists(dir)).To(BeTrue())
}

func TestTestsExist_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	g.Expect(c.TestsExist(dir)).To(BeFalse())
}

func TestTestsFail(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := &checker.DefaultChecker{}
	g.Expect(c.TestsFail("")).To(BeTrue())
}

func TestTestsPass(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := &checker.DefaultChecker{}
	g.Expect(c.TestsPass("")).To(BeTrue())
}

func TestTraceValidationPasses_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	// Empty dir with no phase is vacuously true (no artifacts to fail)
	g.Expect(c.TraceValidationPasses(dir, "")).To(BeTrue())
}

func TestTraceValidationPasses_InvalidPhase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	// Invalid phase causes error → returns false
	g.Expect(c.TraceValidationPasses(dir, "not-a-phase")).To(BeFalse())
}

func TestTraceValidationPasses_WithPhase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	g.Expect(c.TraceValidationPasses(dir, "design")).To(BeFalse())
}

func TestUnblockedTasks_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	c := &checker.DefaultChecker{}
	result := c.UnblockedTasks(dir, "TASK-1")
	g.Expect(result).To(BeNil())
}

func TestUnblockedTasks_FailedTaskFiltered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	tasksContent := "### TASK-1: Task 1\n\n**Status:** pending\n**Dependencies:** None\n\n### TASK-2: Task 2\n\n**Status:** pending\n**Dependencies:** None\n"
	writeFile(t, dir, "tasks.md", tasksContent)

	c := &checker.DefaultChecker{}
	result := c.UnblockedTasks(dir, "TASK-1")
	g.Expect(result).To(ContainElement("TASK-2"))
	g.Expect(result).NotTo(ContainElement("TASK-1"))
}

func TestUnblockedTasks_WithParallelTasks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	tasksContent := "### TASK-1: Task 1\n\n**Status:** pending\n**Dependencies:** None\n\n### TASK-2: Task 2\n\n**Status:** pending\n**Dependencies:** None\n"
	writeFile(t, dir, "tasks.md", tasksContent)

	c := &checker.DefaultChecker{}
	result := c.UnblockedTasks(dir, "TASK-99")
	g.Expect(result).To(ContainElement("TASK-1"))
	g.Expect(result).To(ContainElement("TASK-2"))
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
}
