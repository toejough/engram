package step_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
	"github.com/toejough/projctl/internal/step"
)

// TestISSUE103_TaskParamsTeamName verifies ISSUE-103 AC-1:
// TaskParams includes TeamName field populated from state.Project.Name
func TestISSUE103_TaskParamsTeamName(t *testing.T) {
	t.Run("spawn-producer includes team_name from project name", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		projectName := "test-project"
		_, err := state.Init(dir, projectName, nowFunc(), state.InitOpts{
			Issue: "ISSUE-103",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.TeamName).To(Equal(projectName),
			"TaskParams.TeamName should be populated from state.Project.Name")
	})

	t.Run("spawn-qa includes team_name from project name", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		projectName := "my-feature-project"
		_, err := state.Init(dir, projectName, nowFunc(), state.InitOpts{
			Issue: "ISSUE-103",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.TeamName).To(Equal(projectName),
			"TaskParams.TeamName should be populated from state.Project.Name")
	})

	t.Run("improvement-request re-spawn includes team_name", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		projectName := "rework-project"
		_, err := state.Init(dir, projectName, nowFunc(), state.InitOpts{
			Issue: "ISSUE-103",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.SetPair(dir, "plan", state.PairState{
			Iteration:          1,
			MaxIterations:      3,
			ProducerComplete:   false,
			QAVerdict:          "improvement-request",
			ImprovementRequest: "needs work",
		})
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-producer"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.TeamName).To(Equal(projectName),
			"TaskParams.TeamName should be populated from state.Project.Name on re-spawn")
	})

	t.Run("team_name matches state.Project.Name exactly", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Use a distinctive project name to verify exact match
		projectName := "special-chars-project_v2-final"
		_, err := state.Init(dir, projectName, nowFunc(), state.InitOpts{
			Issue: "ISSUE-103",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.TeamName).To(Equal(projectName),
			"TeamName must match Project.Name exactly, preserving special characters")
	})
}

// TestISSUE103_SubagentType verifies ISSUE-103 AC-2:
// SubagentType uses a valid Task tool type (general-purpose) instead of "code"
func TestISSUE103_SubagentType(t *testing.T) {
	t.Run("spawn-producer uses general-purpose subagent type", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-103",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("general-purpose"),
			"SubagentType should be 'general-purpose', not 'code'")
	})

	t.Run("spawn-qa uses general-purpose subagent type", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-103",
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, phase := range []string{
			"tasklist_create", "plan_produce", "plan_approve",
			"artifact_fork", "artifact_pm_produce", "artifact_join",
			"crosscut_qa",
		} {
			_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
			g.Expect(err).ToNot(HaveOccurred())
		}

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Action).To(Equal("spawn-qa"))
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).To(Equal("general-purpose"),
			"SubagentType should be 'general-purpose' for QA agents too")
	})

	t.Run("all spawn actions use general-purpose not code", func(t *testing.T) {
		// Test across multiple phases that produce spawn actions
		testCases := []struct {
			name           string
			transitions    []string
			expectedAction string
		}{
			{
				name:           "plan_produce",
				transitions:    []string{"tasklist_create", "plan_produce"},
				expectedAction: "spawn-producer",
			},
			{
				name:           "crosscut_qa (QA spawn)",
				transitions:    []string{"tasklist_create", "plan_produce", "plan_approve", "artifact_fork", "artifact_pm_produce", "artifact_join", "crosscut_qa"},
				expectedAction: "spawn-qa",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				g := NewWithT(t)
				dir := t.TempDir()

				_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
					Issue: "ISSUE-103",
				})
				g.Expect(err).ToNot(HaveOccurred())

				for _, phase := range tc.transitions {
					_, err = state.Transition(dir, phase, state.TransitionOpts{}, nowFunc())
					g.Expect(err).ToNot(HaveOccurred())
				}

				result, err := step.Next(dir)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(result.Action).To(Equal(tc.expectedAction))
				g.Expect(result.TaskParams).ToNot(BeNil())
				g.Expect(result.TaskParams.SubagentType).To(Equal("general-purpose"),
					"SubagentType must be 'general-purpose' in phase %s", tc.name)
			})
		}
	})

	t.Run("subagent_type is never 'code'", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{
			Issue: "ISSUE-103",
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = state.Transition(dir, "tasklist_create", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		_, err = state.Transition(dir, "plan_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())

		result, err := step.Next(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.TaskParams).ToNot(BeNil())
		g.Expect(result.TaskParams.SubagentType).ToNot(Equal("code"),
			"'code' is not a valid Task tool subagent_type")
	})
}
