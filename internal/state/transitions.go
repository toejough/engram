package state

// LegalTransitions maps each phase to its valid next phases.
// Aligned with docs/orchestration-system.md Section 7 workflows.
var LegalTransitions = map[string][]string{
	// === INIT ===
	// Can start any workflow
	"init": {"pm", "adopt-explore", "align-explore", "task-implementation"},

	// === NEW PROJECT WORKFLOW (Section 7.2) ===
	// PM → Design → Architecture → Breakdown → Implementation → Documentation → (main flow ending)

	// PM phase
	"pm":          {"pm-complete"},
	"pm-complete": {"design"},

	// Design phase
	"design":          {"design-complete"},
	"design-complete": {"architect"},

	// Architecture phase
	"architect":          {"architect-complete"},
	"architect-complete": {"breakdown"},

	// Breakdown phase
	"breakdown":          {"breakdown-complete"},
	"breakdown-complete": {"implementation"},

	// Implementation phase (TDD loop with per-phase QA)
	// Traces: ARCH-034, ARCH-035, ARCH-036
	// QA iteration loops: improvement-request returns to same producer phase
	"implementation":          {"task-start"},
	"task-start":              {"tdd-red"},
	"tdd-red":                 {"tdd-red-qa"},
	"tdd-red-qa":              {"commit-red"},
	"commit-red":              {"commit-red-qa"},
	"commit-red-qa":           {"tdd-green", "tdd-red"}, // Forward or loop back for improvement
	"tdd-green":               {"tdd-green-qa"},
	"tdd-green-qa":            {"commit-green"},
	"commit-green":            {"commit-green-qa"},
	"commit-green-qa":         {"tdd-refactor", "tdd-green"}, // Forward or loop back for improvement
	"tdd-refactor":            {"tdd-refactor-qa"},
	"tdd-refactor-qa":         {"commit-refactor"},
	"commit-refactor":         {"commit-refactor-qa"},
	"commit-refactor-qa":      {"task-complete", "task-retry", "task-escalated", "tdd-refactor"}, // Forward or loop back for improvement
	"task-complete":           {"task-start", "implementation-complete", "task-documentation"}, // task-documentation for single-task workflow
	"task-retry":              {"tdd-red"},
	"task-escalated":          {"task-start", "implementation-complete"},
	"implementation-complete": {"documentation"},

	// Documentation phase
	"documentation":          {"documentation-complete"},
	"documentation-complete": {"alignment"},

	// === MAIN FLOW ENDING (runs after every workflow) ===
	// Alignment → Retro → Summary → Issue Update → Next Steps → Complete

	"alignment":          {"alignment-complete"},
	"alignment-complete": {"retro"},
	"retro":              {"retro-complete"},
	"retro-complete":     {"summary"},
	"summary":            {"summary-complete"},
	"summary-complete":   {"issue-update"},
	"issue-update":       {"next-steps"},
	"next-steps":         {"complete"},
	"complete":           {}, // Terminal state

	// === ADOPT WORKFLOW (Section 7.3 - bottom-up) ===
	// Explore → Infer-Tests → Infer-Arch → Infer-Design → Infer-Reqs → Escalations → Documentation → (main flow ending)

	"adopt-explore":       {"adopt-infer-tests"},
	"adopt-infer-tests":   {"adopt-infer-arch"},
	"adopt-infer-arch":    {"adopt-infer-design"},
	"adopt-infer-design":  {"adopt-infer-reqs"},
	"adopt-infer-reqs":    {"adopt-escalations"},
	"adopt-escalations":   {"adopt-documentation"},
	"adopt-documentation": {"alignment"}, // Joins main flow ending

	// === ALIGN WORKFLOW (Section 7.4 - same as adopt) ===

	"align-explore":       {"align-infer-tests"},
	"align-infer-tests":   {"align-infer-arch"},
	"align-infer-arch":    {"align-infer-design"},
	"align-infer-design":  {"align-infer-reqs"},
	"align-infer-reqs":    {"align-escalations"},
	"align-escalations":   {"align-documentation"},
	"align-documentation": {"alignment"}, // Joins main flow ending

	// === TASK WORKFLOW (Section 7.5 - single task) ===
	// Implementation → Documentation (optional) → (main flow ending)

	"task-implementation": {"task-start"}, // Enters TDD loop
	"task-documentation":  {"alignment"},  // After single task, joins main flow ending
}

// IsLegalTransition checks whether transitioning from one phase to another is allowed.
func IsLegalTransition(from, to string) bool {
	targets, ok := LegalTransitions[from]
	if !ok {
		return false
	}

	for _, t := range targets {
		if t == to {
			return true
		}
	}

	return false
}

// LegalTargets returns the valid next phases for a given phase.
func LegalTargets(from string) []string {
	return LegalTransitions[from]
}
