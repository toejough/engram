package step

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

// Registry is the global phase registry with all known phases.
var Registry = &PhaseRegistry{
	phases: map[string]PhaseInfo{
		"pm": {
			Producer:        "pm-interview-producer",
			ProducerPath:    "skills/pm-interview-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "requirements.md",
			IDFormat:        "REQ",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "pm-complete",
		},
		"design": {
			Producer:        "design-interview-producer",
			ProducerPath:    "skills/design-interview-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "design.md",
			IDFormat:        "DES",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "design-complete",
		},
		"architect": {
			Producer:        "arch-interview-producer",
			ProducerPath:    "skills/arch-interview-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "architecture.md",
			IDFormat:        "ARCH",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "architect-complete",
		},
		"breakdown": {
			Producer:        "breakdown-producer",
			ProducerPath:    "skills/breakdown-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "tasks.md",
			IDFormat:        "TASK",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "breakdown-complete",
		},
		"tdd-red": {
			Producer:        "tdd-red-producer",
			ProducerPath:    "skills/tdd-red-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "commit-red",
		},
		"tdd-green": {
			Producer:        "tdd-green-producer",
			ProducerPath:    "skills/tdd-green-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "commit-green",
		},
		"tdd-refactor": {
			Producer:        "tdd-refactor-producer",
			ProducerPath:    "skills/tdd-refactor-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "commit-refactor",
		},
		"alignment": {
			Producer:        "alignment-producer",
			ProducerPath:    "skills/alignment-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "alignment-complete",
		},
		"retro": {
			Producer:        "retro-producer",
			ProducerPath:    "skills/retro-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "retro.md",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "retro-complete",
		},
		"summary": {
			Producer:        "summary-producer",
			ProducerPath:    "skills/summary-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "summary.md",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "summary-complete",
		},
		"documentation": {
			Producer:        "doc-producer",
			ProducerPath:    "skills/doc-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
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
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "adopt-infer-arch",
		},
		"adopt-infer-arch": {
			Producer:        "arch-infer-producer",
			ProducerPath:    "skills/arch-infer-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "architecture.md",
			IDFormat:        "ARCH",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "adopt-infer-design",
		},
		"adopt-infer-design": {
			Producer:        "design-infer-producer",
			ProducerPath:    "skills/design-infer-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "design.md",
			IDFormat:        "DES",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "adopt-infer-reqs",
		},
		"adopt-infer-reqs": {
			Producer:        "pm-infer-producer",
			ProducerPath:    "skills/pm-infer-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "requirements.md",
			IDFormat:        "REQ",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "adopt-escalations",
		},
		"adopt-documentation": {
			Producer:        "doc-producer",
			ProducerPath:    "skills/doc-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
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
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "align-infer-arch",
		},
		"align-infer-arch": {
			Producer:        "arch-infer-producer",
			ProducerPath:    "skills/arch-infer-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "architecture.md",
			IDFormat:        "ARCH",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "align-infer-design",
		},
		"align-infer-design": {
			Producer:        "design-infer-producer",
			ProducerPath:    "skills/design-infer-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "design.md",
			IDFormat:        "DES",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "align-infer-reqs",
		},
		"align-infer-reqs": {
			Producer:        "pm-infer-producer",
			ProducerPath:    "skills/pm-infer-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "requirements.md",
			IDFormat:        "REQ",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "align-escalations",
		},
		"align-documentation": {
			Producer:        "doc-producer",
			ProducerPath:    "skills/doc-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "alignment",
		},

		// === TASK WORKFLOW ===

		"task-documentation": {
			Producer:        "doc-producer",
			ProducerPath:    "skills/doc-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "sonnet",
			QAModel:         "haiku",
			CompletionPhase: "alignment",
		},

		// === COMMIT PHASES ===
		// Per-phase commit for TDD red/green/refactor sub-phases
		// Traces: ARCH-034, ARCH-035

		"commit-red": {
			Producer:        "commit-producer",
			ProducerPath:    "skills/commit-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "haiku",
			QAModel:         "haiku",
			CompletionPhase: "commit-red-qa",
		},
		"commit-green": {
			Producer:        "commit-producer",
			ProducerPath:    "skills/commit-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "haiku",
			QAModel:         "haiku",
			CompletionPhase: "commit-green-qa",
		},
		"commit-refactor": {
			Producer:        "commit-producer",
			ProducerPath:    "skills/commit-producer/SKILL.md",
			QA:              "qa",
			QAPath:          "skills/qa/SKILL.md",
			Artifact:        "",
			IDFormat:        "",
			ProducerModel:   "haiku",
			QAModel:         "haiku",
			CompletionPhase: "commit-refactor-qa",
		},

		// === TDD QA PHASES ===
		// Per-phase QA for TDD red/green/refactor sub-phases
		// Traces: ARCH-034, ARCH-037

		"tdd-red-qa": {
			Producer:      "qa",
			ProducerPath:  "skills/qa/SKILL.md",
			QA:            "qa",
			QAPath:        "skills/qa/SKILL.md",
			Artifact:      "",
			IDFormat:      "",
			ProducerModel: "haiku",
			QAModel:       "haiku",
		},
		"tdd-green-qa": {
			Producer:      "qa",
			ProducerPath:  "skills/qa/SKILL.md",
			QA:            "qa",
			QAPath:        "skills/qa/SKILL.md",
			Artifact:      "",
			IDFormat:      "",
			ProducerModel: "haiku",
			QAModel:       "haiku",
		},
		"tdd-refactor-qa": {
			Producer:      "qa",
			ProducerPath:  "skills/qa/SKILL.md",
			QA:            "qa",
			QAPath:        "skills/qa/SKILL.md",
			Artifact:      "",
			IDFormat:      "",
			ProducerModel: "haiku",
			QAModel:       "haiku",
		},

		// === COMMIT QA PHASES ===
		// Per-phase QA for commit red/green/refactor sub-phases
		// Traces: ARCH-035, ARCH-037

		"commit-red-qa": {
			Producer:      "qa",
			ProducerPath:  "skills/qa/SKILL.md",
			QA:            "qa",
			QAPath:        "skills/qa/SKILL.md",
			Artifact:      "",
			IDFormat:      "",
			ProducerModel: "haiku",
			QAModel:       "haiku",
		},
		"commit-green-qa": {
			Producer:      "qa",
			ProducerPath:  "skills/qa/SKILL.md",
			QA:            "qa",
			QAPath:        "skills/qa/SKILL.md",
			Artifact:      "",
			IDFormat:      "",
			ProducerModel: "haiku",
			QAModel:       "haiku",
		},
		"commit-refactor-qa": {
			Producer:      "qa",
			ProducerPath:  "skills/qa/SKILL.md",
			QA:            "qa",
			QAPath:        "skills/qa/SKILL.md",
			Artifact:      "",
			IDFormat:      "",
			ProducerModel: "haiku",
			QAModel:       "haiku",
		},
	},
}
