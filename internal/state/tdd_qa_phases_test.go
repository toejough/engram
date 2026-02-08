package state_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/state"
)

// TestPerPhaseQAInTDDLoop verifies that each TDD sub-phase has its own QA phase.
// In the new flat state machine, each TDD phase has explicit produce → qa → decide states.

func TestTDDFlatStateMachineTransitions(t *testing.T) {
	t.Run("tdd_red_produce to tdd_red_qa transition", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		navigateToState(t, dir, "tdd_red_produce", "new")

		s, err := state.Transition(dir, "tdd_red_qa", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_red_qa"))
	})

	t.Run("tdd_red_decide can loop back to tdd_red_produce", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		navigateToState(t, dir, "tdd_red_decide", "new")

		s, err := state.Transition(dir, "tdd_red_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_red_produce"))
	})

	t.Run("tdd_red_decide can advance to tdd_green_produce", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		navigateToState(t, dir, "tdd_red_decide", "new")

		s, err := state.Transition(dir, "tdd_green_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_green_produce"))
	})

	t.Run("tdd_green_decide can advance to tdd_refactor_produce", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		navigateToState(t, dir, "tdd_green_decide", "new")

		s, err := state.Transition(dir, "tdd_refactor_produce", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_refactor_produce"))
	})

	t.Run("tdd_refactor_decide can advance to tdd_commit", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		_, err := state.Init(dir, "test-project", nowFunc(), state.InitOpts{Workflow: "new"})
		g.Expect(err).ToNot(HaveOccurred())

		navigateToState(t, dir, "tdd_refactor_decide", "new")

		s, err := state.Transition(dir, "tdd_commit", state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(s.Project.Phase).To(Equal("tdd_commit"))
	})
}

func TestTDDLegalTargetsFlat(t *testing.T) {
	t.Run("tdd_red_produce targets tdd_red_qa", func(t *testing.T) {
		g := NewWithT(t)
		targets := state.LegalTargets("tdd_red_produce", "new")
		g.Expect(targets).To(ContainElement("tdd_red_qa"))
	})

	t.Run("tdd_red_decide targets both loop-back and forward", func(t *testing.T) {
		g := NewWithT(t)
		targets := state.LegalTargets("tdd_red_decide", "new")
		g.Expect(targets).To(ContainElement("tdd_red_produce"))
		g.Expect(targets).To(ContainElement("tdd_green_produce"))
	})

	t.Run("tdd_green_decide targets both loop-back and forward", func(t *testing.T) {
		g := NewWithT(t)
		targets := state.LegalTargets("tdd_green_decide", "new")
		g.Expect(targets).To(ContainElement("tdd_green_produce"))
		g.Expect(targets).To(ContainElement("tdd_refactor_produce"))
	})

	t.Run("tdd_refactor_decide targets both loop-back and forward", func(t *testing.T) {
		g := NewWithT(t)
		targets := state.LegalTargets("tdd_refactor_decide", "new")
		g.Expect(targets).To(ContainElement("tdd_refactor_produce"))
		g.Expect(targets).To(ContainElement("tdd_commit"))
	})
}

// navigateToState transitions through a path to reach the target state.
// It uses BFS on the TOML transition graph to find the path from init_state.
func navigateToState(t *testing.T, dir string, target string, wf string) {
	t.Helper()
	g := NewWithT(t)

	// Get init state for this workflow
	initState, err := state.WorkflowInitState(wf)
	g.Expect(err).ToNot(HaveOccurred())

	// Get current phase
	s, err := state.Get(dir)
	g.Expect(err).ToNot(HaveOccurred())

	// If we're still at "init", transition to init_state first
	if s.Project.Phase == "init" {
		_, err = state.Transition(dir, initState, state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred())
		if initState == target {
			return
		}
	}

	// BFS from current state to target
	path := findPath(t, initState, target, wf)
	g.Expect(path).ToNot(BeEmpty(), "no path found from %s to %s in workflow %s", initState, target, wf)

	// Skip states we've already passed (init_state)
	for _, step := range path[1:] { // skip init_state since we already transitioned there
		_, err = state.Transition(dir, step, state.TransitionOpts{}, nowFunc())
		g.Expect(err).ToNot(HaveOccurred(), "failed to transition to %s", step)
		if step == target {
			return
		}
	}
}

// findPath uses BFS to find a path from start to target in the workflow transition graph.
func findPath(t *testing.T, start, target, wf string) []string {
	t.Helper()

	transitions := state.TransitionsForWorkflow(wf)
	if transitions == nil {
		return nil
	}

	type node struct {
		state string
		path  []string
	}

	visited := map[string]bool{}
	queue := []node{{state: start, path: []string{start}}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.state == target {
			return current.path
		}

		if visited[current.state] {
			continue
		}
		visited[current.state] = true

		for _, next := range transitions[current.state] {
			if !visited[next] {
				newPath := make([]string, len(current.path)+1)
				copy(newPath, current.path)
				newPath[len(current.path)] = next
				queue = append(queue, node{state: next, path: newPath})
			}
		}
	}

	return nil
}
