package state

// LegalTransitions maps each phase to its valid next phases.
var LegalTransitions = map[string][]string{
	"init":                    {"pm-interview"},
	"pm-interview":            {"pm-complete"},
	"pm-complete":             {"design-interview"},
	"design-interview":        {"design-complete"},
	"design-complete":         {"alignment-check"},
	"architect-interview":     {"architect-complete"},
	"architect-complete":      {"alignment-check"},
	"alignment-check":         {"architect-interview", "design-interview", "task-breakdown", "implementation", "audit", "completion"},
	"task-breakdown":          {"planning-complete"},
	"planning-complete":       {"alignment-check"},
	"implementation":          {"task-start"},
	"task-start":              {"tdd-red"},
	"tdd-red":                 {"commit-red"},
	"commit-red":              {"tdd-green"},
	"tdd-green":               {"commit-green"},
	"commit-green":            {"tdd-refactor"},
	"tdd-refactor":            {"commit-refactor"},
	"commit-refactor":         {"task-audit"},
	"task-audit":              {"task-complete", "task-retry", "task-escalated"},
	"task-complete":           {"task-start", "implementation-complete"},
	"task-retry":              {"tdd-red"},
	"task-escalated":          {"task-start", "implementation-complete"},
	"implementation-complete": {"audit"},
	"audit":                   {"audit-complete", "audit-fix"},
	"audit-fix":               {"audit"},
	"audit-complete":          {"completion"},
	"completion":              {"integrate-commit"},
	"integrate-commit":        {"integrate-merge"},
	"integrate-merge":         {"integrate-cleanup"},
	"integrate-cleanup":       {"integrate-complete"},
	"integrate-complete":      {},
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
