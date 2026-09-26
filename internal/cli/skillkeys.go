package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/toejough/engram/internal/update"
)

// AssignSkillKeys stamps each candidate's source-qualified key (design D3)
// and returns the keyed candidates in their input order, plus a warning for
// every candidate it skips. It is pure: engramSkillRoots are the fully
// symlink-resolved engram-owned skills roots (ResolveEngramSkillRoots), and
// candidates' SourcePaths are already fully resolved, so the engram-owned
// rule compares resolved paths on both sides.
//
// The key is built from ScopeID, SourceSegment, Kind and Name:
//   - a candidate whose SourcePath lies under an engram-owned root, and every
//     Claude user skill, gets the bare key `<n>`;
//   - otherwise the scope's prefix (`anthropic-skills`, `pi`, `agents`,
//     `<plugin>`, `project:<r>`; none for claude-cmd, pi-prompt and the Pi
//     configured scopes), then SourceSegment when set, then `cmd` for a
//     command or `pi-prompt` for a prompt (unless SourceSegment already is
//     `pi-prompt`), then the name — joined by `:`.
//
// Disabled candidates are keyed like any other (they count as present for
// removal). Skipped with a warning: a name containing `:` (commands are
// already filtered per path segment by their scanner, since their names
// join segments with `:`), an unknown or reserved scope, and a bare key with
// no slug character (its note slug would be empty).
func AssignSkillKeys(candidates []SkillCandidate, engramSkillRoots []string) ([]SkillCandidate, []string) {
	keyed := make([]SkillCandidate, 0, len(candidates))

	var warnings []string

	for _, candidate := range candidates {
		key, problem := skillKeyFor(candidate, engramSkillRoots)
		if problem != "" {
			warnings = append(warnings, fmt.Sprintf(skillKeySkipWarningFormat,
				candidate.Kind, candidate.Name, candidate.SourcePath, problem))

			continue
		}

		candidate.Key = key
		keyed = append(keyed, candidate)
	}

	return keyed, warnings
}

// ResolveEngramSkillRoots returns the fully symlink-resolved engram-owned
// skills roots, `<home>/<EngramRootRel>/skills`, of every supported harness
// (update.EngramOwnedSkillsRels), whether or not the harness is detected. A
// root that cannot be resolved (typically absent) is left out: no candidate
// can have been read from under it.
func ResolveEngramSkillRoots(fsys SkillSourceFS, home string) []string {
	rels := update.EngramOwnedSkillsRels()
	roots := make([]string, 0, len(rels))

	for _, rel := range rels {
		resolved, err := ResolveSkillPath(fsys, filepath.Join(home, rel))
		if err != nil {
			continue
		}

		roots = append(roots, resolved)
	}

	return roots
}

// SkillKeySlug derives a skill note's slug from its key (design D3): the key
// lowercased, every run of characters outside `[a-z0-9]` replaced by `-`,
// leading and trailing `-` trimmed, prefixed with `skill-`. A key holding at
// least one ASCII letter or digit yields a slug matching
// `^skill-[a-z0-9]+(-[a-z0-9]+)*$`; a bare key `x` gives `skill-x`.
func SkillKeySlug(key string) string {
	var slug strings.Builder

	pendingDash := false

	for _, char := range strings.ToLower(key) {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			if pendingDash && slug.Len() > 0 {
				slug.WriteByte('-')
			}

			pendingDash = false

			slug.WriteRune(char)

			continue
		}

		pendingDash = true
	}

	return skillSlugPrefix + slug.String()
}

// unexported constants.
const (
	skillKeyNameColonProblem  = "its name contains `:`"
	skillKeyNoSlugProblem     = "its key has no letter or digit for a note slug"
	skillKeySegmentAnthropic  = "anthropic-skills"
	skillKeySegmentCommand    = "cmd"
	skillKeySeparator         = ":"
	skillKeySkipWarningFormat = "engram: skipping %s %q at %s: %s"
	skillKeyUnknownScope      = "unknown or reserved scope %q"
)

// skillKeyFor returns candidate's key, or a non-empty problem explaining why
// it has none.
func skillKeyFor(candidate SkillCandidate, engramSkillRoots []string) (key, problem string) {
	if strings.Contains(candidate.Name, skillKeySeparator) && candidate.Kind != SkillSourceKindCommand {
		return "", skillKeyNameColonProblem
	}

	parts, known := skillKeyPrefix(candidate)
	if !known {
		return "", fmt.Sprintf(skillKeyUnknownScope, candidate.ScopeID)
	}

	if underEngramSkillRoot(candidate.SourcePath, engramSkillRoots) {
		parts = nil
	}

	if len(parts) == 0 && SkillKeySlug(candidate.Name) == skillSlugPrefix {
		return "", skillKeyNoSlugProblem
	}

	return strings.Join(append(parts, candidate.Name), skillKeySeparator), ""
}

// skillKeyPrefix returns the key segments preceding candidate's name, and
// false for a scope D3 does not define (including a reserved plugin name).
func skillKeyPrefix(candidate SkillCandidate) ([]string, bool) {
	if candidate.ScopeID == SkillScopeClaudeUser {
		return nil, true
	}

	parts, known := skillScopeKeyParts(candidate.ScopeID)
	if !known {
		return nil, false
	}

	if candidate.SourceSegment != "" {
		parts = append(parts, candidate.SourceSegment)
	}

	switch {
	case candidate.Kind == SkillSourceKindCommand:
		parts = append(parts, skillKeySegmentCommand)
	case candidate.Kind == SkillSourceKindPrompt && candidate.SourceSegment != SkillScopePiPrompt:
		parts = append(parts, SkillScopePiPrompt)
	}

	return parts, true
}

// skillScopeKeyParts returns the key segments a scope ID contributes: none
// for claude-cmd, pi-prompt and the Pi configured scopes (whose segment is
// SourceSegment), a fixed segment for synced, pi-user and agents-user, the
// plugin name, or the whole `project:<r>` scope. known is false for any
// other scope, and for a plugin scope with an empty or reserved name.
func skillScopeKeyParts(scope string) (parts []string, known bool) {
	switch scope {
	case SkillScopeClaudeCmd, SkillScopePiPrompt, SkillScopePiSettings:
		return nil, true
	case SkillScopeSynced:
		return []string{skillKeySegmentAnthropic}, true
	case SkillScopePiUser:
		return []string{SkillSegmentPi}, true
	case SkillScopeAgentsUser:
		return []string{SkillSegmentAgents}, true
	}

	switch {
	case strings.HasPrefix(scope, SkillScopePiPkgPrefix):
		return nil, true
	case strings.HasPrefix(scope, SkillScopePluginPrefix):
		plugin := strings.TrimPrefix(scope, SkillScopePluginPrefix)

		return []string{plugin}, plugin != "" && !isReservedPluginName(plugin)
	case strings.HasPrefix(scope, SkillScopeProjectPrefix):
		return []string{scope}, true
	default:
		return nil, false
	}
}

// underEngramSkillRoot reports whether the resolved path lies strictly
// under one of the resolved engram-owned skills roots.
func underEngramSkillRoot(path string, roots []string) bool {
	for _, root := range roots {
		if path != root && pathWithinRoot(path, root) {
			return true
		}
	}

	return false
}
