package workflow_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/workflow"
	"pgregory.net/rapid"
)

func TestLoad(t *testing.T) {
	t.Run("TOML parses without error", func(t *testing.T) {
		g := NewWithT(t)
		cfg, err := workflow.Load()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(cfg).ToNot(BeNil())
	})

	t.Run("has expected workflows", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		names := cfg.WorkflowNames()
		g.Expect(names).To(ContainElements("new", "scoped", "align"))
	})

	t.Run("no adopt workflow exists", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		names := cfg.WorkflowNames()
		g.Expect(names).ToNot(ContainElement("adopt"))
	})

	t.Run("no task workflow exists", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		names := cfg.WorkflowNames()
		g.Expect(names).ToNot(ContainElement("task"))
	})
}

func TestStatesExist(t *testing.T) {
	t.Run("every state referenced in transitions exists in states", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		for _, wfName := range cfg.WorkflowNames() {
			transitions, err := cfg.TransitionsFor(wfName)
			g.Expect(err).ToNot(HaveOccurred())

			for from, targets := range transitions {
				_, ok := cfg.LookupState(from)
				g.Expect(ok).To(BeTrue(),
					"workflow %q: state %q in transitions but not defined in [states]", wfName, from)

				for _, to := range targets {
					_, ok := cfg.LookupState(to)
					g.Expect(ok).To(BeTrue(),
						"workflow %q: transition target %q (from %q) not defined in [states]",
						wfName, to, from)
				}
			}
		}
	})
}

func TestTransitionsFor(t *testing.T) {
	t.Run("new workflow includes TDD loop transitions", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		transitions, err := cfg.TransitionsFor("new")
		g.Expect(err).ToNot(HaveOccurred())

		// TDD loop states should be present
		g.Expect(transitions).To(HaveKey("tdd_red_produce"))
		g.Expect(transitions).To(HaveKey("tdd_green_produce"))
		g.Expect(transitions).To(HaveKey("tdd_refactor_produce"))

		// Plan and parallel artifact states should be present
		g.Expect(transitions).To(HaveKey("plan_produce"))
		g.Expect(transitions).To(HaveKey("plan_approve"))
		g.Expect(transitions).To(HaveKey("artifact_fork"))
	})

	t.Run("scoped workflow includes TDD loop transitions", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		transitions, err := cfg.TransitionsFor("scoped")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(transitions).To(HaveKey("tdd_red_produce"))
		g.Expect(transitions).To(HaveKey("item_select"))
	})

	t.Run("scoped workflow does NOT have PM states", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		transitions, err := cfg.TransitionsFor("scoped")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(transitions).ToNot(HaveKey("pm_produce"))
		g.Expect(transitions).ToNot(HaveKey("design_produce"))
	})

	t.Run("align workflow includes main ending", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		transitions, err := cfg.TransitionsFor("align")
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(transitions).To(HaveKey("alignment_produce"))
		g.Expect(transitions).To(HaveKey("evaluation_commit"))
	})

	t.Run("unknown workflow returns error", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		_, err := cfg.TransitionsFor("nonexistent")
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("unknown workflow"))
	})
}

func TestInitState(t *testing.T) {
	t.Run("all workflows start at tasklist_create", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		for _, wfName := range cfg.WorkflowNames() {
			init, err := cfg.InitState(wfName)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(init).To(Equal("tasklist_create"),
				"workflow %q should start at tasklist_create", wfName)
		}
	})

	t.Run("new workflow transitions from tasklist_create to plan_produce", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		g.Expect(cfg.IsLegalTransition("tasklist_create", "plan_produce", "new")).To(BeTrue())
	})

	t.Run("scoped workflow transitions from tasklist_create to item_select", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		g.Expect(cfg.IsLegalTransition("tasklist_create", "item_select", "scoped")).To(BeTrue())
	})

	t.Run("align workflow transitions from tasklist_create to align_plan_produce", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		g.Expect(cfg.IsLegalTransition("tasklist_create", "align_plan_produce", "align")).To(BeTrue())
	})

	t.Run("align workflow has parallel inference via fork/join", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		g.Expect(cfg.IsLegalTransition("align_plan_approve", "align_infer_fork", "align")).To(BeTrue())
		g.Expect(cfg.IsLegalTransition("align_infer_fork", "align_infer_reqs_produce", "align")).To(BeTrue())
		g.Expect(cfg.IsLegalTransition("align_infer_fork", "align_infer_tests_produce", "align")).To(BeTrue())
		g.Expect(cfg.IsLegalTransition("align_infer_reqs_produce", "align_infer_join", "align")).To(BeTrue())
		g.Expect(cfg.IsLegalTransition("align_infer_join", "align_crosscut_qa", "align")).To(BeTrue())
	})

	t.Run("unknown workflow returns error", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		_, err := cfg.InitState("nonexistent")
		g.Expect(err).To(HaveOccurred())
	})
}

func TestLookupState(t *testing.T) {
	t.Run("finds produce state with correct fields", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		s, ok := cfg.LookupState("pm_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(s.Type).To(Equal(workflow.StateTypeProduce))
		g.Expect(s.Skill).To(Equal("pm-interview-producer"))
		g.Expect(s.SkillPath).To(Equal("skills/pm-interview-producer/SKILL.md"))
		g.Expect(s.DefaultModel).To(Equal("opus"))
		g.Expect(s.Artifact).To(Equal("requirements.md"))
		g.Expect(s.IDFormat).To(Equal("REQ"))
	})

	t.Run("finds QA state", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		s, ok := cfg.LookupState("pm_qa")
		g.Expect(ok).To(BeTrue())
		g.Expect(s.Type).To(Equal(workflow.StateTypeQA))
		g.Expect(s.Skill).To(Equal("qa"))
		g.Expect(s.DefaultModel).To(Equal("haiku"))
	})

	t.Run("finds decide state", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		s, ok := cfg.LookupState("pm_decide")
		g.Expect(ok).To(BeTrue())
		g.Expect(s.Type).To(Equal(workflow.StateTypeDecide))
	})

	t.Run("finds commit state", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		s, ok := cfg.LookupState("pm_commit")
		g.Expect(ok).To(BeTrue())
		g.Expect(s.Type).To(Equal(workflow.StateTypeCommit))
	})

	t.Run("returns false for unknown state", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		_, ok := cfg.LookupState("nonexistent")
		g.Expect(ok).To(BeFalse())
	})
}

func TestIsLegalTransition(t *testing.T) {
	t.Run("plan_produce to plan_approve is legal in new workflow", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		g.Expect(cfg.IsLegalTransition("plan_produce", "plan_approve", "new")).To(BeTrue())
	})

	t.Run("plan_approve routes to artifact_fork or plan_produce", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		targets := cfg.LegalTargets("plan_approve", "new")
		g.Expect(targets).To(ContainElement("artifact_fork"))
		g.Expect(targets).To(ContainElement("plan_produce"))
	})

	t.Run("tdd_red_produce to tdd_red_qa is legal in new workflow", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		g.Expect(cfg.IsLegalTransition("tdd_red_produce", "tdd_red_qa", "new")).To(BeTrue())
	})

	t.Run("tdd_red_produce to tdd_red_qa is legal in scoped workflow", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		g.Expect(cfg.IsLegalTransition("tdd_red_produce", "tdd_red_qa", "scoped")).To(BeTrue())
	})

	t.Run("decide routes to both improvement and approved", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		targets := cfg.LegalTargets("tdd_red_decide", "new")
		g.Expect(targets).To(ContainElement("tdd_red_produce"))   // improvement
		g.Expect(targets).To(ContainElement("tdd_green_produce")) // approved
	})
}

// Property: every workflow reaches terminal state from init via BFS
func TestPropertyReachesTerminal(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		wfName := rapid.SampledFrom(cfg.WorkflowNames()).Draw(rt, "workflow")

		transitions, err := cfg.TransitionsFor(wfName)
		g.Expect(err).ToNot(HaveOccurred())

		initState, err := cfg.InitState(wfName)
		g.Expect(err).ToNot(HaveOccurred())

		// BFS from init state to find all reachable states
		visited := map[string]bool{}
		queue := []string{initState}

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if visited[current] {
				continue
			}
			visited[current] = true

			for _, target := range transitions[current] {
				if !visited[target] {
					queue = append(queue, target)
				}
			}
		}

		// Must reach "complete" terminal state
		g.Expect(visited).To(HaveKey("complete"),
			"workflow %q must reach terminal state 'complete' from init state %q", wfName, initState)
	})
}

// Property: no orphan states (every state in a workflow is reachable from init)
func TestPropertyNoOrphanStates(t *testing.T) {
	t.Run("all states in transition graph are reachable from init", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		for _, wfName := range cfg.WorkflowNames() {
			transitions, err := cfg.TransitionsFor(wfName)
			g.Expect(err).ToNot(HaveOccurred())

			// BFS from "init" — the bootstrap phase injected by TransitionsFor
			visited := map[string]bool{}
			queue := []string{"init"}
			for len(queue) > 0 {
				current := queue[0]
				queue = queue[1:]
				if visited[current] {
					continue
				}
				visited[current] = true
				for _, target := range transitions[current] {
					if !visited[target] {
						queue = append(queue, target)
					}
				}
			}

			// Every state that appears as a source in transitions must be reachable
			for state := range transitions {
				g.Expect(visited).To(HaveKey(state),
					"workflow %q: state %q appears in transitions but is not reachable from init",
					wfName, state)
			}
		}
	})
}

// Property: every produce state has a non-empty skill and skill_path
func TestPropertyProduceStatesHaveSkills(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		// Collect all produce states
		var produceStates []string
		for name, state := range cfg.States {
			if state.Type == workflow.StateTypeProduce {
				produceStates = append(produceStates, name)
			}
		}
		g.Expect(produceStates).ToNot(BeEmpty())

		state := rapid.SampledFrom(produceStates).Draw(rt, "state")
		def, ok := cfg.LookupState(state)
		g.Expect(ok).To(BeTrue())
		g.Expect(def.Skill).ToNot(BeEmpty(),
			"produce state %q must have a skill", state)
		g.Expect(def.SkillPath).ToNot(BeEmpty(),
			"produce state %q must have a skill_path", state)
		g.Expect(def.DefaultModel).ToNot(BeEmpty(),
			"produce state %q must have a default_model", state)
	})
}

// Property: every QA state uses the "qa" skill
func TestPropertyQAStatesUseQASkill(t *testing.T) {
	t.Run("all QA states use qa skill", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		for name, state := range cfg.States {
			if state.Type == workflow.StateTypeQA {
				g.Expect(state.Skill).To(Equal("qa"),
					"QA state %q must use 'qa' skill", name)
				g.Expect(state.SkillPath).To(Equal("skills/qa/SKILL.md"),
					"QA state %q must use 'skills/qa/SKILL.md' skill_path", name)
			}
		}
	})
}

// Property: every decide state is followed by valid transitions (at least 2: loop-back and forward)
func TestPropertyDecideStatesHaveMultipleTargets(t *testing.T) {
	t.Run("decide states in workflows have at least 2 transition targets", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		for _, wfName := range cfg.WorkflowNames() {
			transitions, err := cfg.TransitionsFor(wfName)
			g.Expect(err).ToNot(HaveOccurred())

			for state, targets := range transitions {
				def, ok := cfg.LookupState(state)
				if !ok {
					continue
				}
				if def.Type == workflow.StateTypeDecide {
					g.Expect(len(targets)).To(BeNumerically(">=", 2),
						"workflow %q: decide state %q should have at least 2 targets (improvement + approved), got %v",
						wfName, state, targets)
				}
			}
		}
	})
}

// Property: main-ending states are in every workflow
func TestPropertyMainEndingInAllWorkflows(t *testing.T) {
	t.Run("every workflow includes the main-ending group or reaches complete", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		mainEndingStates := []string{
			"alignment_produce", "evaluation_produce",
			"issue_update", "next_steps", "complete",
		}

		for _, wfName := range cfg.WorkflowNames() {
			transitions, err := cfg.TransitionsFor(wfName)
			g.Expect(err).ToNot(HaveOccurred())

			// Check that main ending states are present
			for _, state := range mainEndingStates {
				_, inTransitions := transitions[state]
				// State might be a target but not a source (e.g., complete has no targets)
				isTarget := false
				for _, targets := range transitions {
					for _, t := range targets {
						if t == state {
							isTarget = true
							break
						}
					}
					if isTarget {
						break
					}
				}
				g.Expect(inTransitions || isTarget).To(BeTrue(),
					"workflow %q: main-ending state %q should be accessible", wfName, state)
			}
		}
	})
}

func TestNewWorkflowPlanMode(t *testing.T) {
	t.Run("new workflow reaches complete through plan mode", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		transitions, err := cfg.TransitionsFor("new")
		g.Expect(err).ToNot(HaveOccurred())

		// Verify plan mode path exists
		g.Expect(transitions).To(HaveKey("tasklist_create"))
		g.Expect(transitions).To(HaveKey("plan_produce"))
		g.Expect(transitions).To(HaveKey("plan_approve"))
		g.Expect(transitions).To(HaveKey("artifact_fork"))
		g.Expect(transitions).To(HaveKey("artifact_join"))
		g.Expect(transitions).To(HaveKey("crosscut_qa"))
		g.Expect(transitions).To(HaveKey("crosscut_decide"))
		g.Expect(transitions).To(HaveKey("artifact_commit"))
	})

	t.Run("new workflow uses evaluation instead of retro+summary", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		transitions, err := cfg.TransitionsFor("new")
		g.Expect(err).ToNot(HaveOccurred())

		// Evaluation states should be present
		g.Expect(transitions).To(HaveKey("evaluation_produce"))
		g.Expect(transitions).To(HaveKey("evaluation_interview"))
		g.Expect(transitions).To(HaveKey("evaluation_commit"))

		// Old retro/summary states should NOT be in the workflow transitions
		g.Expect(transitions).ToNot(HaveKey("retro_produce"))
		g.Expect(transitions).ToNot(HaveKey("summary_produce"))
	})
}

func TestEvaluationStateTypes(t *testing.T) {
	t.Run("evaluation_produce is a produce state", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		s, ok := cfg.LookupState("evaluation_produce")
		g.Expect(ok).To(BeTrue())
		g.Expect(s.Type).To(Equal(workflow.StateTypeProduce))
		g.Expect(s.Skill).To(Equal("evaluation-producer"))
	})

	t.Run("evaluation_interview is an interview state", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		s, ok := cfg.LookupState("evaluation_interview")
		g.Expect(ok).To(BeTrue())
		g.Expect(s.Type).To(Equal(workflow.StateTypeInterview))
	})

	t.Run("plan_approve is an approve state", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		s, ok := cfg.LookupState("plan_approve")
		g.Expect(ok).To(BeTrue())
		g.Expect(s.Type).To(Equal(workflow.StateTypeApprove))
	})

	t.Run("tasklist_create is a tasklist state", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		s, ok := cfg.LookupState("tasklist_create")
		g.Expect(ok).To(BeTrue())
		g.Expect(s.Type).To(Equal(workflow.StateTypeTaskList))
	})
}

func TestAllStatesForWorkflow(t *testing.T) {
	t.Run("new workflow has many states", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		states, err := cfg.AllStatesForWorkflow("new")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(len(states)).To(BeNumerically(">", 20))
	})

	t.Run("scoped workflow has fewer states than new", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		newStates, _ := cfg.AllStatesForWorkflow("new")
		scopedStates, _ := cfg.AllStatesForWorkflow("scoped")
		g.Expect(len(scopedStates)).To(BeNumerically("<", len(newStates)))
	})
}
