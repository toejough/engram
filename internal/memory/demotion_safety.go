package memory

import (
	"strings"
)

// Exported constants.
const (
	// DestinationEmbedding indicates content should become an embedding.
	DestinationEmbedding DemotionDestination = "embedding"
	// DestinationHook indicates content should become a hook.
	DestinationHook DemotionDestination = "hook"
	// DestinationSkill indicates content should become a skill.
	DestinationSkill DemotionDestination = "skill"
)

// DemotionDestination represents the target tier for demoted content.
type DemotionDestination string

// DemotionPlan describes how content should be safely demoted from CLAUDE.md.
type DemotionPlan struct {
	Content         string              `json:"content"`
	CurrentTier     string              `json:"current_tier"`
	DestinationTier DemotionDestination `json:"destination_tier"`
	Reasoning       string              `json:"reasoning"`
	Safe            bool                `json:"safe"`
	CreateAction    string              `json:"create_action"`
	RemovalAction   string              `json:"removal_action"`
}

// PlanCLAUDEMDDemotion analyzes content and creates a safe demotion plan.
// It classifies content into appropriate destinations and validates safety.
func PlanCLAUDEMDDemotion(content string, metadata map[string]any) DemotionPlan {
	plan := DemotionPlan{
		Content:     content,
		CurrentTier: "claude-md",
		Safe:        false,
	}

	lower := strings.ToLower(content)

	// Check for deterministic rules (highest priority)
	if isDeterministicRule(lower) {
		plan.DestinationTier = DestinationHook
		plan.Safe = true
		plan.Reasoning = "Contains deterministic directive (always/never/must/use X not Y) - should be enforced, not suggested"
		plan.CreateAction = "Create hook configuration file in .claude/hooks/"
		plan.RemovalAction = "Remove from CLAUDE.md Promoted Learnings section"

		return plan
	}

	// Check for procedural workflows
	if isProceduralWorkflow(lower) {
		plan.DestinationTier = DestinationSkill
		plan.Safe = true
		plan.Reasoning = "Contains procedural workflow (first/then/phase/step/cycle) - reusable pattern"
		plan.CreateAction = "Create skill file in .claude/skills/"
		plan.RemovalAction = "Remove from CLAUDE.md Promoted Learnings section"

		return plan
	}

	// Check for situational/narrow context
	if isSituationalContent(lower) {
		plan.DestinationTier = DestinationEmbedding
		plan.Safe = true
		plan.Reasoning = "Contains situational or narrow context (this project/only when/specific to) - retrieve when relevant"
		plan.CreateAction = "Store as embedding in memory database"
		plan.RemovalAction = "Remove from CLAUDE.md Promoted Learnings section"

		return plan
	}

	// Unclassifiable content
	plan.Reasoning = "Cannot classify content into deterministic rule, procedural workflow, or situational pattern - manual review required"
	plan.Safe = false
	plan.CreateAction = "Manual review needed"
	plan.RemovalAction = "Do not remove until classification is clear"

	return plan
}

// isDeterministicRule checks if content contains deterministic directives.
func isDeterministicRule(lower string) bool {
	// Check for "use X not Y" or "is X (not Y)" pattern first (most specific)
	if strings.Contains(lower, " not ") || strings.Contains(lower, "(not") {
		return true
	}

	// Check for "never" directive
	if strings.Contains(lower, "never") {
		return true
	}

	// Check for "must" directive (uppercase in original is common)
	// But skip if it's part of a workflow description (contains cycle/phase/step)
	if strings.Contains(lower, "must") {
		if !strings.Contains(lower, "cycle") && !strings.Contains(lower, "phase") && !strings.Contains(lower, "step") {
			return true
		}
	}

	// Check for "always" directive
	// But skip if it's part of a workflow description (contains cycle/phase/step/red/green/refactor)
	if strings.Contains(lower, "always") {
		if !strings.Contains(lower, "cycle") && !strings.Contains(lower, "phase") && !strings.Contains(lower, "step") && !strings.Contains(lower, "red/green/refactor") {
			return true
		}
	}

	return false
}

// isProceduralWorkflow checks if content contains procedural workflow keywords.
func isProceduralWorkflow(lower string) bool {
	// Check for whole-word matches or specific patterns
	// to avoid false positives (e.g., "then" in "authentication")

	// Check for "red/green/refactor" first (most specific)
	if strings.Contains(lower, "red/green/refactor") {
		return true
	}

	// Check for TDD as whole word or at word boundary
	if strings.Contains(lower, "tdd") {
		return true
	}

	// Check for procedural keywords with word boundaries
	proceduralKeywords := []string{
		" first ",
		" then ",
		" cycle",
		" phase",
		" step",
		"first ",
		"then ",
	}

	for _, kw := range proceduralKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}

// isSituationalContent checks if content is situational or narrow in scope.
func isSituationalContent(lower string) bool {
	situationalKeywords := []string{
		"this project",
		"this codebase",
		"only when",
		"specific to",
		"when size",
		"in this",
	}

	for _, kw := range situationalKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}
