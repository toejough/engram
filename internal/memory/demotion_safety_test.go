package memory_test

import (
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestPlanCLAUDEMDDemotion_AllPlansHaveReasoning(t *testing.T) {
	testCases := []string{
		"Trailer is AI-Used: [claude]",
		"First do X, then do Y, finally do Z",
		"Use make([]T, 0, capacity) when size known",
		"Random unclassifiable content",
	}

	for _, content := range testCases {
		plan := memory.PlanCLAUDEMDDemotion(content, map[string]any{})
		if plan.Reasoning == "" {
			t.Errorf("Expected non-empty reasoning for content: %s", content)
		}
	}
}

func TestPlanCLAUDEMDDemotion_DeterministicRuleToHook(t *testing.T) {
	content := "Trailer is AI-Used: [claude] (NOT Co-Authored-By)"
	metadata := map[string]any{}

	plan := memory.PlanCLAUDEMDDemotion(content, metadata)

	if !plan.Safe {
		t.Errorf("Expected safe=true for deterministic rule, got false")
	}

	if plan.DestinationTier != "hook" {
		t.Errorf("Expected destination=hook for deterministic rule, got %s", plan.DestinationTier)
	}

	if plan.Reasoning == "" {
		t.Error("Expected non-empty reasoning")
	}

	if plan.CreateAction == "" {
		t.Error("Expected non-empty create_action")
	}

	if plan.RemovalAction == "" {
		t.Error("Expected non-empty removal_action")
	}
}

func TestPlanCLAUDEMDDemotion_MustKeywordToHook(t *testing.T) {
	content := "You MUST follow these instructions exactly as written"
	metadata := map[string]any{}

	plan := memory.PlanCLAUDEMDDemotion(content, metadata)

	if !plan.Safe {
		t.Error("Expected safe=true for 'must' rule")
	}

	if plan.DestinationTier != "hook" {
		t.Errorf("Expected destination=hook for 'must' rule, got %s", plan.DestinationTier)
	}
}

func TestPlanCLAUDEMDDemotion_NeverKeywordToHook(t *testing.T) {
	content := "Never use git checkout -- . or git restore . — destroys work"
	metadata := map[string]any{}

	plan := memory.PlanCLAUDEMDDemotion(content, metadata)

	if !plan.Safe {
		t.Error("Expected safe=true for 'never' rule")
	}

	if plan.DestinationTier != "hook" {
		t.Errorf("Expected destination=hook for 'never' rule, got %s", plan.DestinationTier)
	}
}

func TestPlanCLAUDEMDDemotion_OnlyWhenSpecificToEmbedding(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{"only when", "Only when building Go projects use this pattern"},
		{"specific to", "This pattern is specific to our authentication system"},
		{"this project", "In this project, use the custom logger"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plan := memory.PlanCLAUDEMDDemotion(tc.content, map[string]any{})
			if !plan.Safe {
				t.Errorf("Expected safe=true for %s pattern", tc.name)
			}

			if plan.DestinationTier != "embedding" {
				t.Errorf("Expected destination=embedding for %s pattern, got %s", tc.name, plan.DestinationTier)
			}
		})
	}
}

func TestPlanCLAUDEMDDemotion_PhaseStepCycleToSkill(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{"phase", "Complete Phase 1 foundation tasks first"},
		{"step", "Follow these steps in order: 1, 2, 3"},
		{"cycle", "Run the cycle repeatedly until complete"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plan := memory.PlanCLAUDEMDDemotion(tc.content, map[string]any{})
			if !plan.Safe {
				t.Errorf("Expected safe=true for %s keyword", tc.name)
			}

			if plan.DestinationTier != "skill" {
				t.Errorf("Expected destination=skill for %s keyword, got %s", tc.name, plan.DestinationTier)
			}
		})
	}
}

func TestPlanCLAUDEMDDemotion_ProceduralWorkflowToSkill(t *testing.T) {
	content := "TDD for ALL artifact changes - Always full red/green/refactor cycle"
	metadata := map[string]any{}

	plan := memory.PlanCLAUDEMDDemotion(content, metadata)

	if !plan.Safe {
		t.Errorf("Expected safe=true for procedural workflow, got false")
	}

	if plan.DestinationTier != "skill" {
		t.Errorf("Expected destination=skill for procedural workflow, got %s", plan.DestinationTier)
	}

	if plan.Reasoning == "" {
		t.Error("Expected non-empty reasoning")
	}
}

func TestPlanCLAUDEMDDemotion_SituationalToEmbedding(t *testing.T) {
	content := "Use make([]T, 0, capacity) when size is known in this codebase"
	metadata := map[string]any{}

	plan := memory.PlanCLAUDEMDDemotion(content, metadata)

	if !plan.Safe {
		t.Errorf("Expected safe=true for situational content, got false")
	}

	if plan.DestinationTier != "embedding" {
		t.Errorf("Expected destination=embedding for situational content, got %s", plan.DestinationTier)
	}

	if plan.Reasoning == "" {
		t.Error("Expected non-empty reasoning")
	}
}

func TestPlanCLAUDEMDDemotion_UnclassifiableNotSafe(t *testing.T) {
	content := "Some random unstructured text that doesn't fit any pattern"
	metadata := map[string]any{}

	plan := memory.PlanCLAUDEMDDemotion(content, metadata)

	if plan.Safe {
		t.Error("Expected safe=false for unclassifiable content")
	}

	if plan.Reasoning == "" {
		t.Error("Expected reasoning explaining why it's not safe")
	}
}

func TestPlanCLAUDEMDDemotion_UseXNotYToHook(t *testing.T) {
	content := "Use targ build system (NOT mage)"
	metadata := map[string]any{}

	plan := memory.PlanCLAUDEMDDemotion(content, metadata)

	if !plan.Safe {
		t.Error("Expected safe=true for 'use X not Y' pattern")
	}

	if plan.DestinationTier != "hook" {
		t.Errorf("Expected destination=hook for 'use X not Y' pattern, got %s", plan.DestinationTier)
	}
}
