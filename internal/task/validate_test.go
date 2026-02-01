package task_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/task"
)

// TEST-460 traces: TASK-025
// Test ValidateTask checks for visual evidence when UI flag is true.
func TestValidateTask_UIRequiresVisualEvidence(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	// Create tasks.md with UI task without visual evidence
	tasksContent := `# Tasks

### TASK-001: Add button

**UI:** true
**Acceptance Criteria:**
- Button renders
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	if err != nil {
		os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
		err = os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	}
	g.Expect(err).ToNot(HaveOccurred())

	result := task.Validate(dir, "TASK-001")
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Error).To(ContainSubstring("visual"))
}

// TEST-461 traces: TASK-025
// Test ValidateTask passes when UI task has visual evidence.
func TestValidateTask_UIWithVisualEvidence(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Add button

**UI:** true
**Visual evidence:** screenshots/task-001.png
**Acceptance Criteria:**
- Button renders
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := task.Validate(dir, "TASK-001")
	g.Expect(result.Valid).To(BeTrue())
}

// TEST-462 traces: TASK-025
// Test ValidateTask passes for non-UI tasks without visual evidence.
func TestValidateTask_NonUIWithoutVisualEvidence(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Add function

**UI:** false
**Acceptance Criteria:**
- Function works
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := task.Validate(dir, "TASK-001")
	g.Expect(result.Valid).To(BeTrue())
}
