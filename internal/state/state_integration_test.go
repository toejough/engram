package state_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

// Integration test for TDD cycle with repo dir detection.
// This validates that the full workflow works end-to-end.
func TestIntegration_TDDCycleWithRepoDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	g := NewWithT(t)

	// Create temp directory for the repo
	repoDir := t.TempDir()
	// Resolve symlinks for macOS /var -> /private/var
	repoDir, _ = filepath.EvalSymlinks(repoDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	g.Expect(cmd.Run()).To(Succeed())

	// Configure git for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = repoDir
	g.Expect(cmd.Run()).To(Succeed())
	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = repoDir
	g.Expect(cmd.Run()).To(Succeed())

	// Create a test file in the repo
	testFile := filepath.Join(repoDir, "example_test.go")
	testContent := `package example

import "testing"

func TestExample(t *testing.T) {
	t.Fatal("not implemented")
}
`
	g.Expect(os.WriteFile(testFile, []byte(testContent), 0o644)).To(Succeed())

	// Create project directory inside repo
	projectDir := filepath.Join(repoDir, ".claude", "projects", "test-project")
	g.Expect(os.MkdirAll(projectDir, 0o755)).To(Succeed())

	// Initialize state with repo dir
	nowFunc := func() time.Time { return time.Now() }
	s, err := state.Init(projectDir, "test-project", nowFunc, state.InitOpts{
		Workflow: "new",
		RepoDir:  repoDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.Project.RepoDir).To(Equal(repoDir))

	// Create a mock checker that uses the actual file system for test detection
	checker := &integrationChecker{repoDir: repoDir}

	// Transition through init -> pm -> pm-complete
	s, err = state.TransitionWithChecker(projectDir, "pm", state.TransitionOpts{}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())

	// For pm-complete, we need requirements.md
	reqContent := "# Requirements\n\n### REQ-001: Test requirement\n"
	g.Expect(os.WriteFile(filepath.Join(projectDir, "requirements.md"), []byte(reqContent), 0o644)).To(Succeed())

	s, err = state.TransitionWithChecker(projectDir, "pm-complete", state.TransitionOpts{}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())

	// Continue through design -> design-complete
	s, err = state.TransitionWithChecker(projectDir, "design", state.TransitionOpts{}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())

	designContent := "# Design\n\n### DES-001: Test design\n\n**Traces to:** REQ-001\n"
	g.Expect(os.WriteFile(filepath.Join(projectDir, "design.md"), []byte(designContent), 0o644)).To(Succeed())

	s, err = state.TransitionWithChecker(projectDir, "design-complete", state.TransitionOpts{}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())

	// Continue through architect -> architect-complete -> breakdown -> breakdown-complete -> implementation
	s, err = state.TransitionWithChecker(projectDir, "architect", state.TransitionOpts{}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())
	s, err = state.TransitionWithChecker(projectDir, "architect-complete", state.TransitionOpts{}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())
	s, err = state.TransitionWithChecker(projectDir, "breakdown", state.TransitionOpts{}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())
	s, err = state.TransitionWithChecker(projectDir, "breakdown-complete", state.TransitionOpts{}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())
	s, err = state.TransitionWithChecker(projectDir, "implementation", state.TransitionOpts{}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())

	// Now the key test: task-start -> tdd-red -> tdd-green
	// tdd-green should find tests in repoDir, not projectDir
	s, err = state.TransitionWithChecker(projectDir, "task-start", state.TransitionOpts{Task: "TASK-001"}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())

	s, err = state.TransitionWithChecker(projectDir, "tdd-red", state.TransitionOpts{Task: "TASK-001"}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())

	// Per-phase QA: tdd-red -> tdd-red-qa -> commit-red -> commit-red-qa -> tdd-green
	s, err = state.TransitionWithChecker(projectDir, "tdd-red-qa", state.TransitionOpts{Task: "TASK-001"}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())

	s, err = state.TransitionWithChecker(projectDir, "commit-red", state.TransitionOpts{Task: "TASK-001"}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())

	s, err = state.TransitionWithChecker(projectDir, "commit-red-qa", state.TransitionOpts{Task: "TASK-001"}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred())

	// tdd-green should also succeed
	s, err = state.TransitionWithChecker(projectDir, "tdd-green", state.TransitionOpts{Task: "TASK-001"}, nowFunc, checker)
	g.Expect(err).ToNot(HaveOccurred(), "tdd-green should succeed when tests exist in repo dir")

	// Verify state is correct
	g.Expect(s.Project.Phase).To(Equal("tdd-green"))
	g.Expect(s.Progress.CurrentTask).To(Equal("TASK-001"))
}

// integrationChecker implements PreconditionChecker for integration tests.
type integrationChecker struct {
	repoDir string
}

func (c *integrationChecker) RequirementsExist(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "requirements.md"))
	return err == nil
}

func (c *integrationChecker) RequirementsHaveIDs(dir string) bool {
	return true // Simplified for integration test
}

func (c *integrationChecker) DesignExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "design.md"))
	return err == nil
}

func (c *integrationChecker) DesignHasIDs(dir string) bool {
	return true // Simplified
}

func (c *integrationChecker) TraceValidationPasses(dir string, phase string) bool {
	return true // Simplified
}

func (c *integrationChecker) TestsExist(dir string) bool {
	// Check if *_test.go files exist in the given directory
	matches, _ := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if len(matches) > 0 {
		return true
	}
	// Also check subdirectories
	matches, _ = filepath.Glob(filepath.Join(dir, "**", "*_test.go"))
	return len(matches) > 0
}

func (c *integrationChecker) TestsFail(dir string) bool {
	return true // For tdd-red phase
}

func (c *integrationChecker) TestsPass(dir string) bool {
	return true // For tdd-green phase
}

func (c *integrationChecker) AcceptanceCriteriaComplete(dir, taskID string) bool {
	return true // Simplified
}

func (c *integrationChecker) IncompleteAcceptanceCriteria(dir, taskID string) []string {
	return nil
}

func (c *integrationChecker) UnblockedTasks(dir, failedTask string) []string {
	return nil
}

func (c *integrationChecker) RetroExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "retro.md"))
	return err == nil
}

func (c *integrationChecker) SummaryExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "summary.md"))
	return err == nil
}

func (c *integrationChecker) IssueACComplete(repoDir, issueID string) bool {
	return true // Simplified for integration test
}

func (c *integrationChecker) IncompleteIssueAC(repoDir, issueID string) []string {
	return nil
}
