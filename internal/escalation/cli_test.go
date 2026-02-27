package escalation_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/escalation"
)

func TestRealCommandExecutor_Run_Failure(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	exec := &escalation.RealCommandExecutor{}

	err := exec.Run("false")
	g.Expect(err).To(HaveOccurred())
}

func TestRealCommandExecutor_Run_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	exec := &escalation.RealCommandExecutor{}

	err := exec.Run("true")
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRealEscalationFS_ReadFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "escalations.md")
	g.Expect(os.WriteFile(path, []byte("content"), 0o644)).To(Succeed())

	fs := &escalation.RealEscalationFS{}

	content, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("content"))
}

func TestRealEscalationFS_ReadFile_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &escalation.RealEscalationFS{}

	_, err := fs.ReadFile("/non-existent-escalation-xyz")
	g.Expect(err).To(HaveOccurred())
}

func TestRealEscalationFS_WriteFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "escalations.md")

	fs := &escalation.RealEscalationFS{}

	g.Expect(fs.WriteFile(path, "written content")).To(Succeed())

	data, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(data)).To(Equal("written content"))
}

// TestRunList_EmptyDirUsesGetwd tests RunList with Dir="" which falls back to os.Getwd().
func TestRunList_EmptyDirUsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Dir="" triggers Getwd(); no escalations.md in the package test dir
	err := escalation.RunList(escalation.ListArgs{})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunList_FilterNoMatch tests RunList when filter produces empty results (non-JSON).
func TestRunList_FilterNoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "escalations.md")
	g.Expect(os.WriteFile(path, []byte(sampleEscalationContent), 0o644)).To(Succeed())

	// Filter by "resolved" but sample only has "pending" escalations → empty result
	err := escalation.RunList(escalation.ListArgs{Dir: dir, Status: "resolved"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunList_NoFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := escalation.RunList(escalation.ListArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunList_NoFile_JSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := escalation.RunList(escalation.ListArgs{Dir: dir, JSON: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunList_WithFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "escalations.md")
	g.Expect(os.WriteFile(path, []byte(sampleEscalationContent), 0o644)).To(Succeed())

	err := escalation.RunList(escalation.ListArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunList_WithFilter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "escalations.md")
	g.Expect(os.WriteFile(path, []byte(sampleEscalationContent), 0o644)).To(Succeed())

	err := escalation.RunList(escalation.ListArgs{Dir: dir, Status: "pending"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunList_WithFilter_JSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "escalations.md")
	g.Expect(os.WriteFile(path, []byte(sampleEscalationContent), 0o644)).To(Succeed())

	err := escalation.RunList(escalation.ListArgs{Dir: dir, JSON: true, Status: "resolved"})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunResolve_EmptyDirUsesGetwd tests RunResolve with Dir="" (uses Getwd, file not present).
func TestRunResolve_EmptyDirUsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Dir="" triggers Getwd(); no escalations.md in the package test dir
	err := escalation.RunResolve(escalation.ResolveArgs{
		ID:     "ESC-001",
		Status: "resolved",
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	}
}

// TestRunResolve_IDNotFound tests RunResolve when the given ID is not in the file.
func TestRunResolve_IDNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "escalations.md")
	g.Expect(os.WriteFile(path, []byte(sampleEscalationContent), 0o644)).To(Succeed())

	err := escalation.RunResolve(escalation.ResolveArgs{
		Dir:    dir,
		ID:     "ESC-999",
		Status: "resolved",
		Notes:  "Not found",
	})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to resolve"))
	}
}

func TestRunResolve_NoFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := escalation.RunResolve(escalation.ResolveArgs{
		Dir:    dir,
		ID:     "ESC-001",
		Status: "resolved",
		Notes:  "done",
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	}
}

func TestRunResolve_WithFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "escalations.md")
	g.Expect(os.WriteFile(path, []byte(sampleEscalationContent), 0o644)).To(Succeed())

	err := escalation.RunResolve(escalation.ResolveArgs{
		Dir:    dir,
		ID:     "ESC-001",
		Status: "resolved",
		Notes:  "Yes, use OAuth",
	})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunReview_EditorFails tests RunReview when the editor command fails.
func TestRunReview_EditorFails(t *testing.T) {
	// No t.Parallel(): uses t.Setenv which is incompatible with t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "escalations.md")
	g.Expect(os.WriteFile(path, []byte(sampleEscalationContent), 0o644)).To(Succeed())

	// Use "false" as editor: always exits with non-zero, causing review to fail
	t.Setenv("EDITOR", "false")

	err := escalation.RunReview(escalation.ReviewArgs{Dir: dir, File: path})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("review failed"))
	}
}

// TestRunReview_EmptyDirUsesGetwd tests RunReview with Dir="" (uses Getwd, file not present).
func TestRunReview_EmptyDirUsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	// Dir="" triggers Getwd(); no escalations.md in the package test dir
	err := escalation.RunReview(escalation.ReviewArgs{})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	}
}

func TestRunReview_NoFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	dir := t.TempDir()

	err := escalation.RunReview(escalation.ReviewArgs{Dir: dir})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	}
}

func TestRunReview_WithFile(t *testing.T) {
	// No t.Parallel(): uses t.Setenv which is incompatible with t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "escalations.md")
	g.Expect(os.WriteFile(path, []byte(sampleEscalationContent), 0o644)).To(Succeed())

	// Use "true" as editor: accepts any args and succeeds without modifying the file
	t.Setenv("EDITOR", "true")

	err := escalation.RunReview(escalation.ReviewArgs{Dir: dir, File: path})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunWrite_AppendToExisting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create first escalation
	err := escalation.RunWrite(escalation.WriteArgs{
		Dir:      dir,
		ID:       "ESC-001",
		Category: "requirement",
		Context:  "First context",
		Question: "First question?",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Add second escalation
	err = escalation.RunWrite(escalation.WriteArgs{
		Dir:      dir,
		ID:       "ESC-002",
		Category: "design",
		Context:  "Second context",
		Question: "Second question?",
	})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunWrite_EmptyDirUsesGetwd tests RunWrite with Dir="" (uses Getwd) and explicit File.
func TestRunWrite_EmptyDirUsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	// Use an explicit File path so we don't write to CWD
	tmpFile := filepath.Join(t.TempDir(), "escalations.md")

	err := escalation.RunWrite(escalation.WriteArgs{
		File:     tmpFile,
		ID:       "ESC-001",
		Category: "requirement",
		Context:  "Testing Getwd fallback",
		Question: "Does Dir fallback to Getwd?",
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunWrite_NewFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := escalation.RunWrite(escalation.WriteArgs{
		Dir:      dir,
		ID:       "ESC-001",
		Category: "requirement",
		Context:  "Analyzing auth flow",
		Question: "Should we support OAuth?",
	})
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(filepath.Join(dir, "escalations.md")).To(BeAnExistingFile())
}

// unexported constants.
const (
	sampleEscalationContent = `# Escalations

Review each escalation and update the **Status** field:
- ` + "`pending`" + ` - Not yet reviewed
- ` + "`resolved`" + ` - Add your answer in **Notes**
- ` + "`deferred`" + ` - Create an issue for later
- ` + "`issue`" + ` - Create an issue with your description in **Notes**

---

## ESC-001

**Category:** requirement
**Context:** Analyzing auth
**Question:** Use OAuth?

**Status:** pending
**Notes:**

---

`
)
