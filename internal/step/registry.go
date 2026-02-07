package step

import (
	"os"
	"path/filepath"
	"strings"
)

// PhaseInfo holds per-phase metadata for deterministic orchestration.
type PhaseInfo struct {
	Producer      string // Producer skill name
	ProducerPath  string // Path to producer SKILL.md
	QA            string // QA skill name
	QAPath        string // Path to QA SKILL.md
	Artifact      string // Artifact filename produced by this phase
	IDFormat      string // ID prefix format (REQ, DES, ARCH, TASK)
	ProducerModel string // Model for producer (from frontmatter)
	QAModel       string // Model for QA (from frontmatter)
	// CompletionPhase is the phase to transition to when sub-phase is done.
	// E.g., for "pm" this is "pm-complete".
	CompletionPhase string
}

// PhaseRegistry is a lookup table for phase metadata.
type PhaseRegistry struct {
	phases map[string]PhaseInfo
}

// Lookup returns the phase info for the given phase, and whether it was found.
func (r *PhaseRegistry) Lookup(phase string) (PhaseInfo, bool) {
	info, ok := r.phases[phase]
	return info, ok
}

// Phases returns all registered phase names.
func (r *PhaseRegistry) Phases() []string {
	result := make([]string, 0, len(r.phases))
	for k := range r.phases {
		result = append(result, k)
	}
	return result
}

// ReadFunc reads a file and returns its contents. Used for dependency injection
// so tests can provide SKILL.md content without real filesystem access.
type ReadFunc func(path string) ([]byte, error)

// ParseSkillModel extracts the model field from SKILL.md YAML frontmatter.
// Returns the model string if found, or empty string if not found or unparseable.
func ParseSkillModel(content []byte) string {
	s := string(content)

	// Find opening ---
	idx := strings.Index(s, "---")
	if idx == -1 {
		return ""
	}
	rest := s[idx+3:]

	// Find closing ---
	end := strings.Index(rest, "---")
	if end == -1 {
		return ""
	}

	frontmatter := rest[:end]

	// Parse model field line by line
	for _, line := range strings.Split(frontmatter, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "model:") {
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "model:"))
			if value != "" {
				return value
			}
		}
	}
	return ""
}

// resolveModel reads a SKILL.md file and returns its model field,
// falling back to the provided fallback if the file can't be read or has no model.
func resolveModel(readFunc ReadFunc, skillPath string, fallback string) string {
	content, err := readFunc(skillPath)
	if err != nil {
		return fallback
	}
	model := ParseSkillModel(content)
	if model == "" {
		return fallback
	}
	return model
}

// skillsBaseDir returns the base directory for skills resolution.
// Skills paths like "skills/foo/SKILL.md" are resolved relative to ~/.claude/.
func skillsBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// newDefaultReadFunc returns a ReadFunc that reads from ~/.claude/<skillPath>.
func newDefaultReadFunc() ReadFunc {
	base := skillsBaseDir()
	return func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(base, path))
	}
}

// phaseDefinition holds the static configuration for a phase, without model fields.
// Models are resolved at registry construction time from SKILL.md frontmatter.
type phaseDefinition struct {
	Producer        string
	ProducerPath    string
	QA              string
	QAPath          string
	Artifact        string
	IDFormat        string
	FallbackPModel  string // Fallback producer model if SKILL.md can't be read
	FallbackQAModel string // Fallback QA model if SKILL.md can't be read
	CompletionPhase string
}

// phaseDefinitions is the static configuration for all known phases.
// Model values here serve as fallbacks when SKILL.md frontmatter is unavailable.
var phaseDefinitions = map[string]phaseDefinition{
	"pm": {
		Producer:        "pm-interview-producer",
		ProducerPath:    "skills/pm-interview-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "requirements.md",
		IDFormat:        "REQ",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "pm-complete",
	},
	"design": {
		Producer:        "design-interview-producer",
		ProducerPath:    "skills/design-interview-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "design.md",
		IDFormat:        "DES",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "design-complete",
	},
	"architect": {
		Producer:        "arch-interview-producer",
		ProducerPath:    "skills/arch-interview-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "architecture.md",
		IDFormat:        "ARCH",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "architect-complete",
	},
	"breakdown": {
		Producer:        "breakdown-producer",
		ProducerPath:    "skills/breakdown-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "tasks.md",
		IDFormat:        "TASK",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "breakdown-complete",
	},
	"tdd-red": {
		Producer:        "tdd-red-producer",
		ProducerPath:    "skills/tdd-red-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "tdd-red-qa",
	},
	"tdd-green": {
		Producer:        "tdd-green-producer",
		ProducerPath:    "skills/tdd-green-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "sonnet",
		FallbackQAModel: "haiku",
		CompletionPhase: "tdd-green-qa",
	},
	"tdd-refactor": {
		Producer:        "tdd-refactor-producer",
		ProducerPath:    "skills/tdd-refactor-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "sonnet",
		FallbackQAModel: "haiku",
		CompletionPhase: "tdd-refactor-qa",
	},
	"alignment": {
		Producer:        "alignment-producer",
		ProducerPath:    "skills/alignment-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "sonnet",
		FallbackQAModel: "haiku",
		CompletionPhase: "alignment-complete",
	},
	"retro": {
		Producer:        "retro-producer",
		ProducerPath:    "skills/retro-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "retro.md",
		FallbackPModel:  "sonnet",
		FallbackQAModel: "haiku",
		CompletionPhase: "retro-complete",
	},
	"summary": {
		Producer:        "summary-producer",
		ProducerPath:    "skills/summary-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "summary.md",
		FallbackPModel:  "sonnet",
		FallbackQAModel: "haiku",
		CompletionPhase: "summary-complete",
	},
	"documentation": {
		Producer:        "doc-producer",
		ProducerPath:    "skills/doc-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "sonnet",
		FallbackQAModel: "haiku",
		CompletionPhase: "documentation-complete",
	},

	// === ADOPT WORKFLOW ===
	// adopt-explore and adopt-escalations are transition-only phases
	// (no producer/QA pair) and are handled by the non-registered path in Next().

	"adopt-infer-tests": {
		Producer:        "tdd-red-infer-producer",
		ProducerPath:    "skills/tdd-red-infer-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "adopt-infer-arch",
	},
	"adopt-infer-arch": {
		Producer:        "arch-infer-producer",
		ProducerPath:    "skills/arch-infer-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "architecture.md",
		IDFormat:        "ARCH",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "adopt-infer-design",
	},
	"adopt-infer-design": {
		Producer:        "design-infer-producer",
		ProducerPath:    "skills/design-infer-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "design.md",
		IDFormat:        "DES",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "adopt-infer-reqs",
	},
	"adopt-infer-reqs": {
		Producer:        "pm-infer-producer",
		ProducerPath:    "skills/pm-infer-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "requirements.md",
		IDFormat:        "REQ",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "adopt-escalations",
	},
	"adopt-documentation": {
		Producer:        "doc-producer",
		ProducerPath:    "skills/doc-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "sonnet",
		FallbackQAModel: "haiku",
		CompletionPhase: "alignment",
	},

	// === ALIGN WORKFLOW ===
	// align-explore and align-escalations are transition-only phases
	// (no producer/QA pair) and are handled by the non-registered path in Next().

	"align-infer-tests": {
		Producer:        "tdd-red-infer-producer",
		ProducerPath:    "skills/tdd-red-infer-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "align-infer-arch",
	},
	"align-infer-arch": {
		Producer:        "arch-infer-producer",
		ProducerPath:    "skills/arch-infer-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "architecture.md",
		IDFormat:        "ARCH",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "align-infer-design",
	},
	"align-infer-design": {
		Producer:        "design-infer-producer",
		ProducerPath:    "skills/design-infer-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "design.md",
		IDFormat:        "DES",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "align-infer-reqs",
	},
	"align-infer-reqs": {
		Producer:        "pm-infer-producer",
		ProducerPath:    "skills/pm-infer-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		Artifact:        "requirements.md",
		IDFormat:        "REQ",
		FallbackPModel:  "opus",
		FallbackQAModel: "haiku",
		CompletionPhase: "align-escalations",
	},
	"align-documentation": {
		Producer:        "doc-producer",
		ProducerPath:    "skills/doc-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "sonnet",
		FallbackQAModel: "haiku",
		CompletionPhase: "alignment",
	},

	// === TASK WORKFLOW ===

	"task-documentation": {
		Producer:        "doc-producer",
		ProducerPath:    "skills/doc-producer/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "sonnet",
		FallbackQAModel: "haiku",
		CompletionPhase: "alignment",
	},

	// === TDD QA PHASES ===
	// Per-phase QA for TDD red/green/refactor sub-phases
	// Traces: ARCH-034, ARCH-037

	"tdd-red-qa": {
		Producer:        "qa",
		ProducerPath:    "skills/qa/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "haiku",
		FallbackQAModel: "haiku",
	},
	"tdd-green-qa": {
		Producer:        "qa",
		ProducerPath:    "skills/qa/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "haiku",
		FallbackQAModel: "haiku",
	},
	"tdd-refactor-qa": {
		Producer:        "qa",
		ProducerPath:    "skills/qa/SKILL.md",
		QA:              "qa",
		QAPath:          "skills/qa/SKILL.md",
		FallbackPModel:  "haiku",
		FallbackQAModel: "haiku",
	},
}

// NewRegistry creates a PhaseRegistry that resolves model assignments from SKILL.md
// frontmatter using the provided ReadFunc. Falls back to hardcoded defaults if
// the file can't be read or the model field is missing.
func NewRegistry(readFunc ReadFunc) *PhaseRegistry {
	phases := make(map[string]PhaseInfo, len(phaseDefinitions))
	for name, def := range phaseDefinitions {
		phases[name] = PhaseInfo{
			Producer:        def.Producer,
			ProducerPath:    def.ProducerPath,
			QA:              def.QA,
			QAPath:          def.QAPath,
			Artifact:        def.Artifact,
			IDFormat:        def.IDFormat,
			ProducerModel:   resolveModel(readFunc, def.ProducerPath, def.FallbackPModel),
			QAModel:         resolveModel(readFunc, def.QAPath, def.FallbackQAModel),
			CompletionPhase: def.CompletionPhase,
		}
	}
	return &PhaseRegistry{phases: phases}
}

// Registry is the global phase registry with all known phases.
// Models are resolved from SKILL.md frontmatter at init time.
var Registry = NewRegistry(newDefaultReadFunc())
