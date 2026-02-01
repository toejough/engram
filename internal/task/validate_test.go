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

// TEST-540 traces: TASK-031
// Test Parallel returns independent pending tasks.
func TestParallel_IndependentTasks(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: First

**Dependencies:** None
**Status:** pending

### TASK-002: Second

**Dependencies:** None
**Status:** pending

### TASK-003: Third

**Dependencies:** TASK-001
**Status:** pending
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	parallel, err := task.Parallel(dir)
	g.Expect(err).ToNot(HaveOccurred())
	// TASK-001 and TASK-002 are independent (no deps)
	// TASK-003 depends on TASK-001 which is pending
	g.Expect(parallel).To(ConsistOf("TASK-001", "TASK-002"))
}

// TEST-541 traces: TASK-031
// Test Parallel excludes tasks with pending dependencies.
func TestParallel_BlockedTasks(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: First

**Dependencies:** None
**Status:** complete

### TASK-002: Second

**Dependencies:** TASK-001
**Status:** pending
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	parallel, err := task.Parallel(dir)
	g.Expect(err).ToNot(HaveOccurred())
	// TASK-001 is complete, so TASK-002 is unblocked
	g.Expect(parallel).To(Equal([]string{"TASK-002"}))
}

// TEST-542 traces: TASK-031
// Test Parallel returns empty when all tasks complete.
func TestParallel_AllComplete(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: First

**Dependencies:** None
**Status:** complete
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	parallel, err := task.Parallel(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(parallel).To(BeEmpty())
}

// TEST-590 traces: TASK-059
// Test ValidateAcceptanceCriteria parses complete checkboxes.
func TestValidateAcceptanceCriteria_CompleteCheckboxes(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Test task

**Acceptance Criteria:**
- [x] First criterion done
- [x] Second criterion done
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := task.ValidateAcceptanceCriteria(dir, "TASK-001")
	g.Expect(result.Complete).To(Equal(2))
	g.Expect(result.Incomplete).To(Equal(0))
	g.Expect(result.AllComplete).To(BeTrue())
}

// TEST-591 traces: TASK-059
// Test ValidateAcceptanceCriteria parses incomplete checkboxes.
func TestValidateAcceptanceCriteria_IncompleteCheckboxes(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Test task

**Acceptance Criteria:**
- [x] First criterion done
- [ ] Second criterion pending
- [ ] Third criterion pending
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := task.ValidateAcceptanceCriteria(dir, "TASK-001")
	g.Expect(result.Complete).To(Equal(1))
	g.Expect(result.Incomplete).To(Equal(2))
	g.Expect(result.AllComplete).To(BeFalse())
	g.Expect(result.Items).To(HaveLen(3))
	g.Expect(result.Items[0].Text).To(ContainSubstring("First criterion"))
	g.Expect(result.Items[0].Complete).To(BeTrue())
	g.Expect(result.Items[1].Text).To(ContainSubstring("Second criterion"))
	g.Expect(result.Items[1].Complete).To(BeFalse())
}

// TEST-592 traces: TASK-059
// Test ValidateAcceptanceCriteria returns error when task not found.
func TestValidateAcceptanceCriteria_TaskNotFound(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Test task

**Acceptance Criteria:**
- [x] Done
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := task.ValidateAcceptanceCriteria(dir, "TASK-999")
	g.Expect(result.Error).To(ContainSubstring("not found"))
}

// TEST-593 traces: TASK-059
// Test ValidateAcceptanceCriteria handles tasks with no AC section.
func TestValidateAcceptanceCriteria_NoAcceptanceCriteria(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Test task

**Description:** Some description
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := task.ValidateAcceptanceCriteria(dir, "TASK-001")
	g.Expect(result.Complete).To(Equal(0))
	g.Expect(result.Incomplete).To(Equal(0))
	g.Expect(result.AllComplete).To(BeTrue()) // Vacuous truth: no criteria = all complete
}

// TEST-594 traces: TASK-059
// Test task-complete transition is blocked when AC incomplete.
func TestStateTransition_TaskCompleteBlockedByIncompleteAC(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Test task

**Acceptance Criteria:**
- [x] Done
- [ ] Not done
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Create state file in task-audit phase
	stateContent := `[project]
  name = "test"
  created = 2026-01-31T12:00:00Z
  phase = "task-audit"

[progress]
  current_task = "TASK-001"
  current_subphase = ""
  tasks_complete = 0
  tasks_total = 1
  tasks_escalated = 0

[[history]]
  timestamp = 2026-01-31T12:00:00Z
  phase = "task-audit"
`
	err = os.WriteFile(filepath.Join(dir, "state.toml"), []byte(stateContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Attempt task-complete transition - should fail
	checker := &task.DefaultPreconditionChecker{}
	err = task.ValidateTaskComplete(dir, "TASK-001", checker)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("acceptance criteria"))
	g.Expect(err.Error()).To(ContainSubstring("unmet"))
}

// TEST-595 traces: TASK-059
// Test task-complete transition succeeds when all AC complete.
func TestStateTransition_TaskCompleteSucceedsWithCompleteAC(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Test task

**Acceptance Criteria:**
- [x] First done
- [x] Second done
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	checker := &task.DefaultPreconditionChecker{}
	err = task.ValidateTaskComplete(dir, "TASK-001", checker)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-596 traces: TASK-059
// Test force flag bypasses AC validation.
func TestValidateAcceptanceCriteria_ForceBypass(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Test task

**Acceptance Criteria:**
- [ ] Not done
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Without force, should fail
	checker := &task.DefaultPreconditionChecker{}
	err = task.ValidateTaskCompleteWithOpts(dir, "TASK-001", checker, task.TaskCompleteOpts{Force: false})
	g.Expect(err).To(HaveOccurred())

	// With force, should succeed
	err = task.ValidateTaskCompleteWithOpts(dir, "TASK-001", checker, task.TaskCompleteOpts{Force: true})
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-597 traces: TASK-059
// Test error message lists specific incomplete AC items.
func TestValidateAcceptanceCriteria_ErrorListsItems(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	tasksContent := `# Tasks

### TASK-001: Test task

**Acceptance Criteria:**
- [x] Done item
- [ ] Missing item A
- [ ] Missing item B
`
	err := os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte(tasksContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	checker := &task.DefaultPreconditionChecker{}
	err = task.ValidateTaskComplete(dir, "TASK-001", checker)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Missing item A"))
	g.Expect(err.Error()).To(ContainSubstring("Missing item B"))
}
