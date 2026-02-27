package task_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/task"
)

func TestDependencyGraph_Roots(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Root task

**Status:** pending
**Dependencies:** None

### TASK-2: Child task

**Status:** pending
**Dependencies:** TASK-1
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	graph, err := task.ParseDependencies(dir)
	g.Expect(err).ToNot(HaveOccurred())

	roots := graph.Roots()
	g.Expect(roots).To(ConsistOf("TASK-1"))
}

func TestDetectOverlap_ReturnsNotImplementedError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := task.DetectOverlap("/any/dir")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("not implemented"))
	}
}

func TestNotImplementedError_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := task.DetectOverlap("/any/dir")

	var niErr *task.NotImplementedError

	g.Expect(err).To(BeAssignableToTypeOf(niErr))

	if err != nil {
		g.Expect(err.Error()).ToNot(BeEmpty())
	}
}

func TestParallel_AllComplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Task

**Status:** complete
**Dependencies:** None
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	parallel, err := task.Parallel(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(parallel).To(BeEmpty())
}

func TestParallel_NoTasksFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := task.Parallel(dir)
	g.Expect(err).To(HaveOccurred())
}

func TestParallel_ReturnsPendingUnblockedTasks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Root task

**Status:** pending
**Dependencies:** None

### TASK-2: Child task

**Status:** pending
**Dependencies:** TASK-1
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	parallel, err := task.Parallel(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(parallel).To(ConsistOf("TASK-1"))
}

func TestParseDependencies_DetectsCycle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: First task

**Status:** pending
**Dependencies:** TASK-2

### TASK-2: Second task

**Status:** pending
**Dependencies:** TASK-1
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	graph, err := task.ParseDependencies(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(graph.HasCycle()).To(BeTrue())
	g.Expect(graph.CyclePath()).ToNot(BeEmpty())
}

func TestParseDependencies_EmptyTasksFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("# Tasks\n"), 0o644)).To(Succeed())

	graph, err := task.ParseDependencies(dir)
	g.Expect(err).ToNot(HaveOccurred())

	if graph != nil {
		g.Expect(graph.Tasks).To(BeEmpty())
	}

	g.Expect(graph.HasCycle()).To(BeFalse())
}

func TestParseDependencies_TasksNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	_, err := task.ParseDependencies(dir)
	g.Expect(err).To(HaveOccurred())
}

func TestParseDependencies_WithTasks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: First task

**Status:** pending
**Dependencies:** None

### TASK-2: Second task

**Status:** pending
**Dependencies:** TASK-1
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	graph, err := task.ParseDependencies(dir)

	g.Expect(err).ToNot(HaveOccurred())

	if graph != nil {
		g.Expect(graph.Tasks).To(ConsistOf("TASK-1", "TASK-2"))
	}

	g.Expect(graph.HasCycle()).To(BeFalse())
}

func TestRunDeps_CycleDetected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: First task

**Status:** pending
**Dependencies:** TASK-2

### TASK-2: Second task

**Status:** pending
**Dependencies:** TASK-1
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())
	err := task.RunDeps(task.DepsArgs{Dir: dir})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("cycle detected"))
	}
}

func TestRunDeps_DotOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Root task

**Status:** pending
**Dependencies:** None

### TASK-2: Child task

**Status:** pending
**Dependencies:** TASK-1
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	err := task.RunDeps(task.DepsArgs{Dir: dir, Format: "dot"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunDeps_JSONOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Root task

**Status:** pending
**Dependencies:** None

### TASK-2: Child task

**Status:** pending
**Dependencies:** TASK-1
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	err := task.RunDeps(task.DepsArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunDeps_NoTasksFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := task.RunDeps(task.DepsArgs{Dir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunParallel_EmptyResult(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Task

**Status:** complete
**Dependencies:** None
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	err := task.RunParallel(task.ParallelArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunParallel_JSONOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Root task

**Status:** pending
**Dependencies:** None
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	err := task.RunParallel(task.ParallelArgs{Dir: dir, Format: "json"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunParallel_NoTasksFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := task.RunParallel(task.ParallelArgs{Dir: dir})
	g.Expect(err).To(HaveOccurred())
}

func TestRunParallel_TextOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Root task

**Status:** pending
**Dependencies:** None
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	err := task.RunParallel(task.ParallelArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunValidate_InvalidTask(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: UI task

**Status:** pending
**UI:** true
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())
	err := task.RunValidate(task.ValidateArgs{Dir: dir, Task: "TASK-1"})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("validation failed"))
	}
}

func TestRunValidate_ManualVisualVerified(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: UI task

**Status:** pending
**UI:** true
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	err := task.RunValidate(task.ValidateArgs{Dir: dir, Task: "TASK-1", ManualVisualVerified: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunValidate_ValidTask(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Backend task

**Status:** pending
**UI:** false
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	err := task.RunValidate(task.ValidateArgs{Dir: dir, Task: "TASK-1"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidateAcceptanceCriteria_AllComplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Task

**Status:** pending
**Acceptance Criteria:**
- [x] First criterion
- [x] Second criterion
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	result := task.ValidateAcceptanceCriteria(dir, "TASK-1")
	g.Expect(result.Error).To(BeEmpty())
	g.Expect(result.AllComplete).To(BeTrue())
	g.Expect(result.Complete).To(Equal(2))
	g.Expect(result.Incomplete).To(Equal(0))
}

func TestValidateAcceptanceCriteria_SomeIncomplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Task

**Status:** pending
**Acceptance Criteria:**
- [x] Done item
- [ ] Not done item
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	result := task.ValidateAcceptanceCriteria(dir, "TASK-1")
	g.Expect(result.Error).To(BeEmpty())
	g.Expect(result.AllComplete).To(BeFalse())
	g.Expect(result.Complete).To(Equal(1))
	g.Expect(result.Incomplete).To(Equal(1))
}

func TestValidateAcceptanceCriteria_TasksNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	result := task.ValidateAcceptanceCriteria(dir, "TASK-1")
	g.Expect(result.Error).To(ContainSubstring("could not read"))
}

func TestValidateTaskComplete_AllCriteriaComplete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Task

**Status:** pending
**Acceptance Criteria:**
- [x] Done
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	err := task.ValidateTaskComplete(dir, "TASK-1", nil)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidateTaskComplete_IncompleteCriteria(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Task

**Status:** pending
**Acceptance Criteria:**
- [ ] Not done

`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	err := task.ValidateTaskComplete(dir, "TASK-1", nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("acceptance criteria"))
	}
}

func TestValidateTaskComplete_WithForce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := task.ValidateTaskCompleteWithOpts(dir, "TASK-1", nil, task.TaskCompleteOpts{Force: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidateWithOpts_ManualVisualVerified(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: UI task

**Status:** pending
**UI:** true
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	result := task.ValidateWithOpts(dir, "TASK-1", task.ValidateOpts{ManualVisualVerified: true})
	g.Expect(result.Valid).To(BeTrue())
	g.Expect(result.Warning).To(ContainSubstring("manual verification"))
}

func TestValidate_NonUITask(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: Backend task

**Status:** pending
**UI:** false
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	result := task.Validate(dir, "TASK-1")
	g.Expect(result.Valid).To(BeTrue())
}

func TestValidate_TaskNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("# Tasks\n"), 0o644)).To(Succeed())

	result := task.Validate(dir, "TASK-999")
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Error).To(ContainSubstring("task not found"))
}

func TestValidate_TasksNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	result := task.Validate(dir, "TASK-1")
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Error).To(ContainSubstring("could not read"))
}

func TestValidate_UITaskWithEvidence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: UI task

**Status:** pending
**UI:** true
**Visual evidence:** screenshot.png
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	result := task.Validate(dir, "TASK-1")
	g.Expect(result.Valid).To(BeTrue())
}

func TestValidate_UITaskWithoutEvidence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Tasks

### TASK-1: UI task

**Status:** pending
**UI:** true
`
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644)).To(Succeed())

	result := task.Validate(dir, "TASK-1")
	g.Expect(result.Valid).To(BeFalse())
	g.Expect(result.Error).To(ContainSubstring("visual evidence"))
}
