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

		// PM states should be present (workflow-specific)
		g.Expect(transitions).To(HaveKey("pm_produce"))
		g.Expect(transitions).To(HaveKey("pm_qa"))
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
		g.Expect(transitions).To(HaveKey("summary_commit"))
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
	t.Run("new workflow starts at pm_produce", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		init, err := cfg.InitState("new")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(init).To(Equal("pm_produce"))
	})

	t.Run("scoped workflow starts at item_select", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		init, err := cfg.InitState("scoped")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(init).To(Equal("item_select"))
	})

	t.Run("align workflow starts at align_infer_tests_produce", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()

		init, err := cfg.InitState("align")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(init).To(Equal("align_infer_tests_produce"))
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
		g.Expect(s.FallbackModel).To(Equal("opus"))
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
		g.Expect(s.FallbackModel).To(Equal("haiku"))
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
	t.Run("pm_produce to pm_qa is legal in new workflow", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		g.Expect(cfg.IsLegalTransition("pm_produce", "pm_qa", "new")).To(BeTrue())
	})

	t.Run("pm_produce to pm_qa is NOT legal in scoped workflow", func(t *testing.T) {
		g := NewWithT(t)
		cfg := workflow.MustLoad()
		g.Expect(cfg.IsLegalTransition("pm_produce", "pm_qa", "scoped")).To(BeFalse())
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

			initState, err := cfg.InitState(wfName)
			g.Expect(err).ToNot(HaveOccurred())

			// BFS from init
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

			// Every state that appears as a source in transitions must be reachable
			for state := range transitions {
				g.Expect(visited).To(HaveKey(state),
					"workflow %q: state %q appears in transitions but is not reachable from %q",
					wfName, state, initState)
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
		g.Expect(def.FallbackModel).ToNot(BeEmpty(),
			"produce state %q must have a fallback_model", state)
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
			"alignment_produce", "retro_produce", "summary_produce",
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
