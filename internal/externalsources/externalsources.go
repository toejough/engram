// Package externalsources discovers and reads files outside engram's own
// memory store that recall/prepare may want to cross-search:
// CLAUDE.md hierarchy, .claude/rules/, Claude Code auto memory, and
// installed skills.
package externalsources

// Kind identifies which kind of external source an ExternalFile came from.
type Kind int

// Kind values.
const (
	KindUnknown Kind = iota
	KindClaudeMd
	KindRules
	KindAutoMemory
	KindSkill
)

// String returns the canonical lowercase identifier for a Kind, used in
// status output and DUPLICATE response shapes.
func (k Kind) String() string {
	switch k {
	case KindClaudeMd:
		return "claude_md"
	case KindRules:
		return "rules"
	case KindAutoMemory:
		return "auto_memory"
	case KindSkill:
		return "skill"
	case KindUnknown:
		return "unknown"
	default:
		return "invalid"
	}
}

// ExternalFile names one discovered file along with its source kind.
// Discovery produces a slice of these; phase extractors consume them.
type ExternalFile struct {
	Kind Kind
	Path string // absolute path
}
