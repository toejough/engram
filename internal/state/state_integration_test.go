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

	// Navigate to tdd_red_produce using the BFS helper
	navigateToState(t, projectDir, "tdd_red_produce", "new")

	// Verify we're at tdd_red_produce
	s, err = state.Get(projectDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.Project.Phase).To(Equal("tdd_red_produce"))

	// tdd_red_produce → tdd_red_qa
	s, err = state.Transition(projectDir, "tdd_red_qa", state.TransitionOpts{Task: "TASK-001"}, nowFunc)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.Project.Phase).To(Equal("tdd_red_qa"))

	// tdd_red_qa → tdd_red_decide
	s, err = state.Transition(projectDir, "tdd_red_decide", state.TransitionOpts{}, nowFunc)
	g.Expect(err).ToNot(HaveOccurred())

	// tdd_red_decide → tdd_green_produce
	s, err = state.Transition(projectDir, "tdd_green_produce", state.TransitionOpts{}, nowFunc)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.Project.Phase).To(Equal("tdd_green_produce"))
}
