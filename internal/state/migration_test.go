package state_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

// TestBackwardCompatibilityMigration verifies auto-migration from legacy
// phase="tdd" to the flat state machine's phase="tdd_red_produce".

func TestLegacyPhaseMigration(t *testing.T) {
	t.Run("detects legacy phase tdd and migrates to tdd_red_produce", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

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

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_red_produce"), "legacy tdd phase should migrate to tdd_red_produce")
	})

	t.Run("migration persists to disk", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

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

		_, err = state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		// Second load should read migrated state from disk
		s2, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s2.Project.Phase).To(Equal("tdd_red_produce"), "migrated state should be persisted to disk")
	})

	t.Run("migration is idempotent", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

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

		s1, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		s2, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		s3, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(s1.Project.Phase).To(Equal("tdd_red_produce"))
		g.Expect(s2.Project.Phase).To(Equal("tdd_red_produce"))
		g.Expect(s3.Project.Phase).To(Equal("tdd_red_produce"))

		g.Expect(len(s3.History)).To(Equal(len(s1.History)), "history should not grow on repeated migrations")
	})

	t.Run("non-legacy phases are not affected by migration", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		modernState := `[project]
name = "test-project"
phase = "tdd_green_produce"
workflow = "new"
issue = ""

[progress]
current_task = ""
current_subphase = ""
completed_tasks = []

[[history]]
phase = "tdd_green_produce"
timestamp = "2026-02-06T10:00:00Z"
`
		statePath := filepath.Join(dir, "state.toml")
		err := os.WriteFile(statePath, []byte(modernState), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_green_produce"), "modern phases should not be affected by migration logic")
	})
}

func TestMigrationWorkflowContinuation(t *testing.T) {
	t.Run("workflow continues correctly after migration from legacy tdd phase", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

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
phase = "tdd"
timestamp = "2026-02-06T10:15:00Z"
`
		statePath := filepath.Join(dir, "state.toml")
		err := os.WriteFile(statePath, []byte(legacyState), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Load state to trigger migration
		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_red_produce"))

		// Verify workflow can continue from migrated tdd_red_produce
		_, err = state.Transition(dir, "tdd_red_qa", state.TransitionOpts{
			Task: "TASK-001",
		}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "should be able to transition from migrated tdd_red_produce phase")
	})

	t.Run("migration preserves task context", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		legacyState := `[project]
name = "test-project"
phase = "tdd"
workflow = "scoped"
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

		s, err := state.Get(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_red_produce"))
		g.Expect(s.Progress.CurrentTask).To(Equal("TASK-003"), "current task should be preserved")
		g.Expect(s.Progress.CompletedTasks).To(HaveLen(2), "completed tasks should be preserved")
		g.Expect(s.Progress.CompletedTasks).To(ContainElements("TASK-001", "TASK-002"))
	})
}
