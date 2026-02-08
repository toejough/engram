package step

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/toejough/projctl/internal/workflow"
)

// PhaseInfo holds per-phase metadata for deterministic orchestration.
type PhaseInfo struct {
	Producer      string // Producer skill name
	ProducerPath  string // Path to producer SKILL.md
	QA            string // QA skill name (always "qa" for qa-type states)
	QAPath        string // Path to QA SKILL.md
	Artifact      string // Artifact filename produced by this phase
	IDFormat      string // ID prefix format (REQ, DES, ARCH, TASK)
	ProducerModel string // Model for producer (from frontmatter or fallback)
	QAModel       string // Model for QA (from frontmatter or fallback)
	StateType     workflow.StateType // Type of state (produce, qa, decide, etc.)
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
// falling back to the provided default if the file can't be read or has no model.
// Precedence: SKILL.md frontmatter model > TOML default_model
func resolveModel(readFunc ReadFunc, skillPath string, defaultModel string) string {
	content, err := readFunc(skillPath)
	if err != nil {
		return defaultModel
	}
	model := ParseSkillModel(content)
	if model == "" {
		return defaultModel
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

// NewRegistry creates a PhaseRegistry derived from the TOML workflow config.
// It resolves model assignments from SKILL.md frontmatter using the provided
// ReadFunc, falling back to the TOML-defined default_model if the file can't
// be read or the model field is missing.
func NewRegistry(readFunc ReadFunc) *PhaseRegistry {
	cfg := workflow.DefaultConfig
	phases := make(map[string]PhaseInfo, len(cfg.States))

	for name, def := range cfg.States {
		info := PhaseInfo{
			Artifact:  def.Artifact,
			IDFormat:  def.IDFormat,
			StateType: def.Type,
		}

		switch def.Type {
		case workflow.StateTypeProduce:
			info.Producer = def.Skill
			info.ProducerPath = def.SkillPath
			info.ProducerModel = resolveModel(readFunc, def.SkillPath, def.DefaultModel)
		case workflow.StateTypeQA:
			info.QA = def.Skill
			info.QAPath = def.SkillPath
			info.QAModel = resolveModel(readFunc, def.SkillPath, def.DefaultModel)
		}

		phases[name] = info
	}

	return &PhaseRegistry{phases: phases}
}

// Registry is the global phase registry with all known phases.
// Models are resolved from SKILL.md frontmatter at init time.
var Registry = NewRegistry(newDefaultReadFunc())
