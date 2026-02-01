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

// TEST-470 traces: TASK-026
// Test ValidateTask with manual visual verified flag bypasses visual evidence requirement.
func TestValidateTask_ManualVisualVerified(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Add button

**UI:** true
**Acceptance Criteria:**
- Button renders
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Without manual flag, should fail
	result := task.ValidateWithOpts(dir, "TASK-001", task.ValidateOpts{})
	g.Expect(result.Valid).To(BeFalse())

	// With manual flag, should pass with warning
	result = task.ValidateWithOpts(dir, "TASK-001", task.ValidateOpts{ManualVisualVerified: true})
	g.Expect(result.Valid).To(BeTrue())
	g.Expect(result.Warning).To(ContainSubstring("manual"))
}

// TEST-530 traces: TASK-030
// Test ParseDependencies extracts task dependencies from tasks.md.
func TestParseDependencies(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: First task

**Dependencies:** None

### TASK-002: Second task

**Dependencies:** TASK-001

### TASK-003: Third task

**Dependencies:** TASK-001, TASK-002
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	graph, err := task.ParseDependencies(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(graph.Tasks).To(HaveLen(3))
	g.Expect(graph.Deps["TASK-001"]).To(BeEmpty())
	g.Expect(graph.Deps["TASK-002"]).To(Equal([]string{"TASK-001"}))
	g.Expect(graph.Deps["TASK-003"]).To(ConsistOf("TASK-001", "TASK-002"))
}

// TEST-531 traces: TASK-030
// Test ParseDependencies detects cycles.
func TestParseDependencies_DetectsCycle(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: First

**Dependencies:** TASK-002

### TASK-002: Second

**Dependencies:** TASK-001
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	graph, err := task.ParseDependencies(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(graph.HasCycle()).To(BeTrue())
	g.Expect(graph.CyclePath()).ToNot(BeEmpty())
}

// TEST-532 traces: TASK-030
// Test ParseDependencies finds root tasks (no dependencies).
func TestParseDependencies_RootTasks(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Root A

**Dependencies:** None

### TASK-002: Root B

**Dependencies:** None

### TASK-003: Child

**Dependencies:** TASK-001, TASK-002
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	graph, err := task.ParseDependencies(dir)
	g.Expect(err).ToNot(HaveOccurred())
	roots := graph.Roots()
	g.Expect(roots).To(ConsistOf("TASK-001", "TASK-002"))
}
