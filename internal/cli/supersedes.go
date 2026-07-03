package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/toejough/engram/internal/embed"
)

// SupersedesInverse maps a superseded-note basename to the slice of notes that
// supersede it, populated by BuildSupersedesInverse from scanned frontmatter.
// Consumed by slice-3 ride-along insertion.
type SupersedesInverse map[string][]supersedesEntry

// BuildSupersedesInverse scans a slice of note frontmatter supersedes entries
// (keyed by superseder basename) and builds the inverse map: superseded
// basename → []supersedesEntry for all superseder notes. Consumed by the
// slice-3 ride-along insertion at query time.
//
// supersedersByNote maps each superseding note's basename to its parsed
// supersedes list. (This input is produced by the query scanner reading
// frontmatter from the vault at scan time.)
func BuildSupersedesInverse(supersedersByNote map[string][]supersedesEntry) SupersedesInverse {
	inverse := make(SupersedesInverse)

	for supersederBasename, entries := range supersedersByNote {
		for _, entry := range entries {
			inverse[entry.Note] = append(inverse[entry.Note], supersedesEntry{
				Note:  supersederBasename,
				Type:  entry.Type,
				Claim: entry.Claim,
			})
		}
	}

	return inverse
}

// unexported constants.
const (
	// supersedesBodyMarker is aliased to the embed marker so the writer's line
	// matching and the BodyText/ContentHash exclusion can never drift apart.
	supersedesBodyMarker = embed.SupersedesBodyMarker
	// supersedesPartCount is the number of pipe-separated parts in a
	// --supersedes flag value: <note>|<type>|<claim>.
	supersedesPartCount = 3
	// Valid supersedes type values (validate on write; reject others).
	supersedesTypeNarrows = "narrows"
	supersedesTypeRefutes = "refutes"
	supersedesTypeUpdates = "updates"
)

// unexported variables.
var (
	errSupersedesEmptyClaim    = errors.New("--supersedes: claim must be non-empty")
	errSupersedesEmptyNote     = errors.New("--supersedes: basename must be non-empty")
	errSupersedesInvalidFormat = errors.New("--supersedes: format must be \"<basename>|<type>|<claim>\"")
	errSupersedesInvalidType   = errors.New("--supersedes: type must be updates|narrows|refutes")
)

// supersedesEntry is the per-entry shape stored in the `supersedes:` frontmatter
// list and scanned for in-memory inverse construction. Also used to render the
// `Supersedes: [[...]] — type: claim` body lines.
type supersedesEntry struct {
	Note  string `yaml:"note"`
	Type  string `yaml:"type"`
	Claim string `yaml:"claim"`
}

// parseAllSupersedes parses every raw `--supersedes` flag value and returns the
// resulting entries. Returns the first parse error encountered.
func parseAllSupersedes(rawFlags []string) ([]supersedesEntry, error) {
	if len(rawFlags) == 0 {
		return nil, nil
	}

	entries := make([]supersedesEntry, 0, len(rawFlags))

	for _, raw := range rawFlags {
		entry, err := parseSupersedesFlag(raw)
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// parseSupersedesFlag parses one raw `--supersedes` flag value of the form
// `"<basename>|<type>|<claim>"`. Rejects any type not in the valid set.
func parseSupersedesFlag(raw string) (supersedesEntry, error) {
	parts := strings.SplitN(raw, "|", supersedesPartCount)
	if len(parts) != supersedesPartCount {
		return supersedesEntry{}, fmt.Errorf("%w: got %q", errSupersedesInvalidFormat, raw)
	}

	note := strings.TrimSpace(parts[0])
	typ := strings.TrimSpace(parts[1])
	claim := strings.TrimSpace(parts[2])

	if note == "" {
		return supersedesEntry{}, errSupersedesEmptyNote
	}

	if claim == "" {
		return supersedesEntry{}, errSupersedesEmptyClaim
	}

	if !validSupersedesType(typ) {
		return supersedesEntry{}, fmt.Errorf("%w: got %q", errSupersedesInvalidType, typ)
	}

	return supersedesEntry{Note: note, Type: typ, Claim: claim}, nil
}

// renderSupersedes produces the `Supersedes:` body lines for the supplied
// entries. Returns "" when entries is empty.
func renderSupersedes(entries []supersedesEntry) string {
	if len(entries) == 0 {
		return ""
	}

	lines := make([]string, len(entries))

	for i, entry := range entries {
		lines[i] = fmt.Sprintf("Supersedes: [[%s]] — %s: %s", entry.Note, entry.Type, entry.Claim)
	}

	return strings.Join(lines, "\n") + "\n"
}

// replaceSupersedes removes any existing `Supersedes:` lines from body and
// appends new ones for entries. When entries is empty, the lines are removed.
// Idempotency rule: the whole block is replaced, never appended.
func replaceSupersedes(body string, entries []supersedesEntry) string {
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.HasPrefix(line, supersedesBodyMarker) {
			continue
		}

		out = append(out, line)
	}

	// Trim trailing blank lines.
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}

	if len(entries) > 0 {
		out = append(out, "")
		superLines := strings.Split(strings.TrimRight(renderSupersedes(entries), "\n"), "\n")
		out = append(out, superLines...)
	}

	result := strings.Join(out, "\n")
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result
}

// validSupersedesType reports whether typ is one of the permitted supersession
// relation types.
func validSupersedesType(typ string) bool {
	switch typ {
	case supersedesTypeUpdates, supersedesTypeNarrows, supersedesTypeRefutes:
		return true
	default:
		return false
	}
}
