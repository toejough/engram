package main_test

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

// traces: TASK-6
// Test that --reported-model flag parses and populates CompleteResult.ReportedModel.
func TestStepComplete_ReportedModelFlag(t *testing.T) {
	g := NewWithT(t)

	dir := t.TempDir()
	setupStepTestProject(t, dir)

	binary := buildProjctl(t)

	// Complete spawn-producer as failed with --reported-model
	cmd := exec.Command(binary, "step", "complete",
		"-d", dir,
		"-a", "spawn-producer",
		"-s", "failed",
		"--reportedmodel", "haiku",
	)
	out, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "step complete failed: %s", string(out))

	// Verify the reported model was persisted via FailedModels
	s, err := state.Get(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.Pairs["pm"].SpawnAttempts).To(Equal(1))
	g.Expect(s.Pairs["pm"].FailedModels).To(ContainElement("haiku"))
}

// traces: TASK-6
// Test that --reported-model flag is optional (empty when not provided).
func TestStepComplete_ReportedModelOptional(t *testing.T) {
	g := NewWithT(t)

	dir := t.TempDir()
	setupStepTestProject(t, dir)

	binary := buildProjctl(t)

	// Complete spawn-producer as done without --reported-model
	cmd := exec.Command(binary, "step", "complete",
		"-d", dir,
		"-a", "spawn-producer",
		"-s", "done",
	)
	out, err := cmd.CombinedOutput()
	g.Expect(err).ToNot(HaveOccurred(), "step complete failed: %s", string(out))

	// Verify it worked (producer marked complete)
	s, err := state.Get(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(s.Pairs["pm"].ProducerComplete).To(BeTrue())
}

// buildProjctl builds the projctl binary and returns the path.
func buildProjctl(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "projctl")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = filepath.Join(projectRoot())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build projctl: %s\n%s", err, string(out))
	}
	return binary
}

func projectRoot() string {
	// This file is at cmd/projctl/step_test.go, project root is 2 levels up
	// But we use an absolute path since tests may run from different dirs
	return "/Users/joe/repos/personal/projctl/cmd/projctl"
}

// setupStepTestProject creates a minimal project directory for step tests.
func setupStepTestProject(t *testing.T, dir string) {
	t.Helper()
	g := NewWithT(t)

	// Init state and transition to pm
	_, err := state.Init(dir, "test-project", func() time.Time { return time.Now() }, state.InitOpts{
		Issue: "ISSUE-98",
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = state.Transition(dir, "pm", state.TransitionOpts{}, func() time.Time { return time.Now() })
	g.Expect(err).ToNot(HaveOccurred())
}
