package state_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

// TestBackwardCompatibilityMigration verifies TASK-8 and TASK-15 acceptance criteria:
// Auto-migration from legacy phase="tdd" to phase="tdd-red"
//
// Traces to: REQ-105-004, DES-105-008, ARCH-105-012, TASK-8, TASK-15, ISSUE-105

func TestLegacyPhaseMigration(t *testing.T) {
	t.Run("detects legacy phase tdd and migrates to tdd-red", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create a legacy state file with phase="tdd"
		legacyState := `[project]
name = "test-project"
phase = "tdd"
workflow = "new"
issue = ""

[progress]
current_task = ""
current_subphase = ""
completed_tasks = []

[[history]]
phase = "init"
timestamp = "2026-02-06T10:00:00Z"

[[history]]
phase = "tdd"
timestamp = "2026-02-06T10:05:00Z"
`
		statePath := filepath.Join(dir, "state.toml")
		err := os.WriteFile(statePath, []byte(legacyState), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Load state - should trigger migration
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify phase migrated to tdd-red
		g.Expect(s.Project.Phase).To(Equal("tdd-red"), "legacy tdd phase should migrate to tdd-red")
	})

	t.Run("migration sets iteration to 0 for tdd-red phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create legacy state with phase="tdd"
		legacyState := `[project]
name = "test-project"
phase = "tdd"
workflow = "new"
issue = ""

[progress]
current_task = "TASK-001"
current_subphase = ""
completed_tasks = []

[[history]]
phase = "tdd"
timestamp = "2026-02-06T10:00:00Z"
`
		statePath := filepath.Join(dir, "state.toml")
		err := os.WriteFile(statePath, []byte(legacyState), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Load state
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify iteration is 0 after migration
		// Pair state for tdd-red should be initialized with iteration 0
		g.Expect(s.Pairs).ToNot(BeNil(), "Pairs map should exist")
		pairState, exists := s.Pairs["tdd-red"]
		g.Expect(exists).To(BeTrue(), "Pair state for tdd-red should be initialized")
		g.Expect(pairState.Iteration).To(Equal(0), "iteration should be 0 after migration")
	})

	t.Run("migration persists to disk", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create legacy state
		legacyState := `[project]
name = "test-project"
phase = "tdd"
workflow = "new"
issue = ""

[progress]
current_task = ""
current_subphase = ""
completed_tasks = []

[[history]]
phase = "tdd"
timestamp = "2026-02-06T10:00:00Z"
`
		statePath := filepath.Join(dir, "state.toml")
		err := os.WriteFile(statePath, []byte(legacyState), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// First load triggers migration
		_, err = state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Second load should read migrated state from disk
		s2, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s2.Project.Phase).To(Equal("tdd-red"), "migrated state should be persisted to disk")
	})

	t.Run("migration is idempotent", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create legacy state
		legacyState := `[project]
name = "test-project"
phase = "tdd"
workflow = "new"
issue = ""

[progress]
current_task = ""
current_subphase = ""
completed_tasks = []

[[history]]
phase = "tdd"
timestamp = "2026-02-06T10:00:00Z"
`
		statePath := filepath.Join(dir, "state.toml")
		err := os.WriteFile(statePath, []byte(legacyState), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Load state multiple times
		s1, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		s2, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		s3, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// All loads should produce same result
		g.Expect(s1.Project.Phase).To(Equal("tdd-red"))
		g.Expect(s2.Project.Phase).To(Equal("tdd-red"))
		g.Expect(s3.Project.Phase).To(Equal("tdd-red"))

		// History should not accumulate duplicate migration entries
		g.Expect(len(s3.History)).To(Equal(len(s1.History)), "history should not grow on repeated migrations")
	})

	t.Run("non-legacy phases are not affected by migration", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create state with modern TDD sub-phase
		modernState := `[project]
name = "test-project"
phase = "tdd-green"
workflow = "new"
issue = ""

[progress]
current_task = ""
current_subphase = ""
completed_tasks = []

[[history]]
phase = "tdd-green"
timestamp = "2026-02-06T10:00:00Z"
`
		statePath := filepath.Join(dir, "state.toml")
		err := os.WriteFile(statePath, []byte(modernState), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Load state
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify phase unchanged
		g.Expect(s.Project.Phase).To(Equal("tdd-green"), "modern phases should not be affected by migration logic")
	})

	t.Run("other legacy phases remain unchanged", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create state with a non-TDD phase
		otherState := `[project]
name = "test-project"
phase = "pm"
workflow = "new"
issue = ""

[progress]
current_task = ""
current_subphase = ""
completed_tasks = []

[[history]]
phase = "pm"
timestamp = "2026-02-06T10:00:00Z"
`
		statePath := filepath.Join(dir, "state.toml")
		err := os.WriteFile(statePath, []byte(otherState), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Load state
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify phase unchanged
		g.Expect(s.Project.Phase).To(Equal("pm"), "non-TDD legacy phases should remain unchanged")
	})
}

func TestMigrationWorkflowContinuation(t *testing.T) {
	t.Run("workflow continues correctly after migration from legacy tdd phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create legacy state at tdd phase
		legacyState := `[project]
name = "test-project"
phase = "tdd"
workflow = "new"
issue = "ISSUE-105"

[progress]
current_task = "TASK-001"
current_subphase = ""
completed_tasks = []

[[history]]
phase = "init"
timestamp = "2026-02-06T10:00:00Z"

[[history]]
phase = "pm"
timestamp = "2026-02-06T10:05:00Z"

[[history]]
phase = "pm-complete"
timestamp = "2026-02-06T10:10:00Z"

[[history]]
phase = "tdd"
timestamp = "2026-02-06T10:15:00Z"
`
		statePath := filepath.Join(dir, "state.toml")
		err := os.WriteFile(statePath, []byte(legacyState), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Load state to trigger migration
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-red"))

		// Verify workflow can continue from tdd-red
		_, err = state.Transition(dir, "tdd-red-qa", state.TransitionOpts{
			Task: "TASK-001",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "should be able to transition from migrated tdd-red phase")

		// Verify full TDD cycle can continue (tdd-red-qa goes directly to tdd-green)
		s, err = state.Transition(dir, "tdd-green", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-green"))
	})

	t.Run("migration preserves task context", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Legacy state with task in progress
		legacyState := `[project]
name = "test-project"
phase = "tdd"
workflow = "task"
issue = "ISSUE-105"

[progress]
current_task = "TASK-003"
current_subphase = "green"
completed_tasks = ["TASK-001", "TASK-002"]

[[history]]
phase = "tdd"
timestamp = "2026-02-06T10:00:00Z"
`
		statePath := filepath.Join(dir, "state.toml")
		err := os.WriteFile(statePath, []byte(legacyState), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Load and verify task context preserved
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd-red"))
		g.Expect(s.Progress.CurrentTask).To(Equal("TASK-003"), "current task should be preserved")
		g.Expect(s.Progress.CompletedTasks).To(HaveLen(2), "completed tasks should be preserved")
		g.Expect(s.Progress.CompletedTasks).To(ContainElements("TASK-001", "TASK-002"))
	})
}
