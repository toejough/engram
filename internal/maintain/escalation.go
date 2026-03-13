package maintain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Exported constants.
const (
	LevelAdvisory           EscalationLevel = "advisory"
	LevelEmphasizedAdvisory EscalationLevel = "emphasized_advisory"
	LevelGraduated          EscalationLevel = "graduated"
	LevelReminder           EscalationLevel = "reminder"
)

// EffData maps escalation levels to observed effectiveness deltas for
// memories that were previously at that level.
type EffData map[EscalationLevel][]float64

// EnforcementApplier updates the enforcement_level in the instruction registry.
type EnforcementApplier interface {
	SetEnforcementLevel(id string, level string, reason string) error
}

// EscalationEngine analyzes leech memories and proposes escalation actions.
type EscalationEngine struct {
	effData EffData
}

// NewEscalationEngine creates an EscalationEngine.
func NewEscalationEngine(effData EffData, _ func() time.Time) *EscalationEngine {
	return &EscalationEngine{
		effData: effData,
	}
}

// Analyze processes leech memories and returns escalation proposals.
func (e *EscalationEngine) Analyze(
	leeches []EscalationMemory,
) ([]EscalationProposal, error) {
	proposals := make([]EscalationProposal, 0, len(leeches))

	for idx := range leeches {
		proposal, ok := e.analyzeOne(&leeches[idx])
		if ok {
			proposals = append(proposals, proposal)
		}
	}

	return proposals, nil
}

func (e *EscalationEngine) analyzeOne(
	mem *EscalationMemory,
) (EscalationProposal, bool) {
	currentLevel := mem.EscalationLevel
	if currentLevel == "" {
		currentLevel = LevelAdvisory
	}

	// De-escalation check.
	if deEsc, proposal := e.tryDeEscalation(mem, currentLevel); deEsc {
		return proposal, true
	}

	// Standard escalation: propose next level up.
	nextLevel, ok := nextEscalationLevel(currentLevel)
	if !ok {
		return EscalationProposal{}, false
	}

	predictedImpact := e.predictImpact(nextLevel)

	return EscalationProposal{
		MemoryPath:      mem.Path,
		ProposalType:    "escalate",
		CurrentLevel:    string(currentLevel),
		ProposedLevel:   string(nextLevel),
		Rationale:       fmt.Sprintf("Memory ineffective at %s level", currentLevel),
		PredictedImpact: predictedImpact,
	}, true
}

func (e *EscalationEngine) predictImpact(level EscalationLevel) string {
	deltas, ok := e.effData[level]
	if !ok || len(deltas) == 0 {
		return "unknown"
	}

	var sum float64

	for _, delta := range deltas {
		sum += delta
	}

	avg := sum / float64(len(deltas))

	return fmt.Sprintf("%+.0f%% follow rate", avg)
}

func (e *EscalationEngine) tryDeEscalation(
	mem *EscalationMemory,
	currentLevel EscalationLevel,
) (bool, EscalationProposal) {
	history := mem.EscalationHistory
	if len(history) < deEscalationCycles {
		return false, EscalationProposal{}
	}

	// Find pre-escalation effectiveness (the entry before current level).
	var preEff float64

	var preFound bool

	for idx := range history {
		if history[idx].Level == currentLevel && idx > 0 {
			preEff = history[idx-1].Effectiveness
			preFound = true

			break
		}
	}

	if !preFound {
		return false, EscalationProposal{}
	}

	// Check last deEscalationCycles entries: all must show post >= pre (REQ-P6f-9).
	tail := history[len(history)-deEscalationCycles:]
	allImproved := true

	for idx := range tail {
		if tail[idx].Effectiveness < preEff {
			allImproved = false

			break
		}
	}

	if !allImproved {
		return false, EscalationProposal{}
	}

	prevLevel, ok := prevEscalationLevel(currentLevel)
	if !ok {
		return false, EscalationProposal{}
	}

	return true, EscalationProposal{
		MemoryPath:      mem.Path,
		ProposalType:    "de_escalate",
		CurrentLevel:    string(currentLevel),
		ProposedLevel:   string(prevLevel),
		Rationale:       "Post-escalation effectiveness improved vs pre-escalation for 3+ cycles",
		PredictedImpact: e.predictImpact(prevLevel),
	}
}

// EscalationHistoryEntry records a level change and its observed effectiveness.
type EscalationHistoryEntry struct {
	Level         EscalationLevel `json:"level"         toml:"level"`
	Since         time.Time       `json:"since"         toml:"since"`
	Effectiveness float64         `json:"effectiveness" toml:"effectiveness"`
}

// EscalationLevel represents the enforcement intensity for a memory.
type EscalationLevel string

// EscalationMemory is the view of a memory needed by the escalation engine.
type EscalationMemory struct {
	Path              string
	Content           string
	EscalationLevel   EscalationLevel
	EscalationHistory []EscalationHistoryEntry
	Effectiveness     float64
}

// EscalationProposal recommends an escalation action.
//
//nolint:tagliatelle // DES-31 specifies snake_case JSON field names.
type EscalationProposal struct {
	MemoryPath      string `json:"memory_path"`
	ProposalType    string `json:"proposal_type"`
	CurrentLevel    string `json:"current_level"`
	ProposedLevel   string `json:"proposed_level"`
	Rationale       string `json:"rationale"`
	PredictedImpact string `json:"predicted_impact"`
}

// GraduationEmitter emits a graduation signal when a memory reaches graduated level.
type GraduationEmitter interface {
	EmitGraduation(memoryPath, recommendation string, detectedAt time.Time) error
}

// ApplyEscalationProposal persists a proposal's level change to the registry and
// emits a graduation signal when the proposed level is "graduated".
// Returns nil (no-op) if applier is nil.
func ApplyEscalationProposal(
	proposal EscalationProposal,
	content string,
	applier EnforcementApplier,
	emitter GraduationEmitter,
	now func() time.Time,
) error {
	if applier == nil {
		return nil
	}

	err := applier.SetEnforcementLevel(
		proposal.MemoryPath,
		proposal.ProposedLevel,
		proposal.Rationale,
	)
	if err != nil {
		return fmt.Errorf("setting enforcement level: %w", err)
	}

	if EscalationLevel(proposal.ProposedLevel) == LevelGraduated && emitter != nil {
		recommendation := ClassifyContent(content)
		detectedAt := time.Now()

		if now != nil {
			detectedAt = now()
		}

		_ = emitter.EmitGraduation(proposal.MemoryPath, recommendation, detectedAt)
	}

	return nil
}

// ClassifyContent determines the recommended destination for a graduated memory.
// Returns one of: "settings.json", ".claude/rules/", "skill", "CLAUDE.md".
func ClassifyContent(content string) string {
	lower := strings.ToLower(content)

	if containsAny(lower, "linter", "lint", "deny", "eslint", "golangci", "settings", "config") {
		return "settings.json"
	}

	if containsAny(lower, "glob pattern", "file glob", "rule file", ".claude/rules", "file-scoped") {
		return ".claude/rules/"
	}

	if containsAny(lower, "workflow", "procedure", "step-by-step", "how to", "process", "skill") {
		return "skill"
	}

	return "CLAUDE.md"
}

// MarshalProposal serializes an EscalationProposal to JSON.
// EscalationProposal contains only string fields, so json.Marshal cannot fail.
func MarshalProposal(proposal EscalationProposal) json.RawMessage {
	//nolint:errchkjson // string-only struct cannot fail
	data, _ := json.Marshal(proposal)

	return data
}

// unexported constants.
const (
	deEscalationCycles = 3
)

// unexported variables.
var (
	escalationLadder = []EscalationLevel{ //nolint:gochecknoglobals // package-level constant table
		LevelAdvisory,
		LevelEmphasizedAdvisory,
		LevelReminder,
		LevelGraduated,
	}
)

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}

	return false
}

func nextEscalationLevel(current EscalationLevel) (EscalationLevel, bool) {
	for idx, level := range escalationLadder {
		if level == current && idx+1 < len(escalationLadder) {
			return escalationLadder[idx+1], true
		}
	}

	return "", false
}

func prevEscalationLevel(current EscalationLevel) (EscalationLevel, bool) {
	for idx, level := range escalationLadder {
		if level == current && idx > 0 {
			return escalationLadder[idx-1], true
		}
	}

	return "", false
}
