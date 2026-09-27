package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/toejough/engram/internal/update"
)

// AssignSkillKeys stamps each candidate's source-qualified key (design D3)
// and returns the keyed candidates in their input order, plus a warning for
// every candidate it skips. It is pure.
//
// The key is built from ScopeID, SourceSegment, Kind and Name: the scope's
// prefix (`claude` for the Claude user skills and commands,
// `anthropic-skills`, `pi`, `agents`, `<plugin>`, `project:<r>`; none for
// pi-prompt and the Pi configured scopes), then SourceSegment when set, then
// `cmd` for a command or `pi-prompt` for a prompt (unless SourceSegment
// already is `pi-prompt`), then the name — joined by `:`. Every key is
// qualified: there is no bare key, and no path-based exception, so engram's
// installed skills are keyed by the folder they are found in (`claude:route`
// through ~/.claude/skills, `pi:route` through ~/.pi/agent/skills).
//
// Disabled candidates are keyed like any other (they count as present for
// removal). Skipped with a warning: a name containing `:` (commands are
// already filtered per path segment by their scanner, since their names
// join segments with `:`), an unknown or reserved scope, and a key the key
// parser does not recognize (e.g. an empty command segment), so every
// returned key is recognized (parseSkillKey).
func AssignSkillKeys(candidates []SkillCandidate) ([]SkillCandidate, []string) {
	keyed := make([]SkillCandidate, 0, len(candidates))

	var warnings []string

	for _, candidate := range candidates {
		key, problem := skillKeyFor(candidate)
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
// root that does not exist is left out silently: no candidate can have been
// read from under it. A root that fails to resolve for any other reason (a
// permission error, a link loop) is left out with a warning (ruling R26),
// since a copy read through it would then lose its bare key.
func ResolveEngramSkillRoots(fsys SkillSourceFS, home string) ([]string, []string) {
	rels := update.EngramOwnedSkillsRels()
	roots := make([]string, 0, len(rels))

	var warnings []string

	for _, rel := range rels {
		root := filepath.Join(home, rel)

		resolved, err := ResolveSkillPath(fsys, root)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				warnings = append(warnings, fmt.Sprintf(engramRootUnresolvedWarningFormat, root, err))
			}

			continue
		}

		roots = append(roots, resolved)
	}

	return roots, warnings
}

// SkillKeySlug derives a skill note's slug from its key (design D3): the key
// lowercased, every run of characters outside `[a-z0-9]` replaced by `-`,
// leading and trailing `-` trimmed, prefixed with `skill-`. A key holding at
// least one ASCII letter or digit yields a slug matching
// `^skill-[a-z0-9]+(-[a-z0-9]+)*$`: `claude:route` gives
// `skill-claude-route`, and `claude:cmd:audit` gives `skill-claude-cmd-audit`.
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
	// engramRootUnresolvedWarningFormat reports an engram-owned skills root
	// that exists but cannot be resolved (ruling R26).
	engramRootUnresolvedWarningFormat = "engram: cannot resolve the engram skills root %s: %v; " +
		"copies read through it lose their bare key"
	// skillKeyMinSegments is the fewest segments a recognized key has: a
	// source qualifier and a name (design D3).
	skillKeyMinSegments      = 2
	skillKeyNameColonProblem = "its name contains `:`"
	// skillKeySegmentAndName is the length of a key tail holding one
	// sub-source segment and a name (e.g. `pi-prompt:<n>`).
	skillKeySegmentAndName    = 2
	skillKeySegmentAnthropic  = "anthropic-skills"
	skillKeySegmentClaude     = "claude"
	skillKeySegmentCommand    = "cmd"
	skillKeySeparator         = ":"
	skillKeySkipWarningFormat = "engram: skipping %s %q at %s: %s"
	skillKeyUnknownScope      = "unknown or reserved scope %q"
	skillKeyUnrecognized      = "its key %q is not a recognized key"
)

// parsedSkillKey is what a skill key's namespace says (design D3, D5).
type parsedSkillKey struct {
	// recognized is false for design D3's three unrecognized cases: no
	// `:`, an empty segment, or a reserved form's prefix whose tail lacks
	// that form's shape. An unrecognized key is never matched, never an
	// alias source and never removal-eligible.
	recognized bool
	// form is the removal-eligibility form of a recognized non-plugin key,
	// and sourced marks the source-rooted forms (checked against the note's
	// skill_source).
	form    SkillRootForm
	sourced bool
	// plugin is the plugin name of a plugin key: any recognized key whose
	// first segment is not a reserved form.
	plugin string
	// scopeID is the key's answer scope (design D3's scope ID column).
	scopeID string
}

// fixedSkillKey is a recognized (when shaped) single-root user form key,
// whose scope ID is its form.
func fixedSkillKey(form SkillRootForm, shaped bool) parsedSkillKey {
	if !shaped {
		return parsedSkillKey{}
	}

	return parsedSkillKey{recognized: true, form: form, scopeID: string(form)}
}

// isReservedPluginName reports whether a plugin name is one of the
// top-level key forms (design D3: `claude`, `pi`, `agents`, `project`,
// `anthropic-skills`, `pi-settings`, `pi-pkg`, `pi-prompt`), or cannot be a
// key segment (it contains `:`).
func isReservedPluginName(name string) bool {
	_, reserved := skillKeyFormParsers()[name]

	return reserved || strings.Contains(name, skillKeySeparator)
}

// parseClaudeSkillKey classifies the tail of a `claude:` key: one name is a
// user skill (even one named `cmd`), and `cmd:` followed by a command path
// a user command.
func parseClaudeSkillKey(tail []string) parsedSkillKey {
	if len(tail) == 1 {
		return fixedSkillKey(SkillRootFormClaudeUser, true)
	}

	return fixedSkillKey(SkillRootFormClaudeCmd, tail[0] == skillKeySegmentCommand)
}

// parseSkillKey classifies key by its namespace (design D3). The first
// segment names the source family (skillKeyFormParsers); any other first
// segment is a plugin's. A key with no `:`, an empty segment, or a reserved
// prefix with a malformed tail is unrecognized.
func parseSkillKey(key string) parsedSkillKey {
	segments := strings.Split(key, skillKeySeparator)
	if len(segments) < skillKeyMinSegments || slices.Contains(segments, "") {
		return parsedSkillKey{}
	}

	head, tail := segments[0], segments[1:]

	if parse, reserved := skillKeyFormParsers()[head]; reserved {
		return parse(tail)
	}

	return parsedSkillKey{recognized: true, plugin: head, scopeID: SkillScopePluginPrefix + head}
}

// piConfiguredKeyTail reports whether tail is a Pi configured source's
// entry: a skill name, or `pi-prompt:<n>`.
func piConfiguredKeyTail(tail []string) bool {
	return len(tail) == 1 || (len(tail) == skillKeySegmentAndName && tail[0] == SkillScopePiPrompt)
}

// piPackageKeyTail reports whether tail is a Pi package's `<pkg-id>:` and
// entry. A pkg-id may itself hold `:` (a local path), so any tail of two
// or more segments has the shape; the package root that vouches for the
// note checks the exact prefix (design D5).
func piPackageKeyTail(tail []string) bool {
	return len(tail) > 1
}

// projectKeyTail reports whether tail, the key after `project:<r>:`, is one
// of the project sub-source shapes: a skill name, `cmd:` and a command
// path, `pi:`/`agents:`/`pi-prompt:` and a name, or a Pi settings or
// package tail.
func projectKeyTail(tail []string) bool {
	switch {
	case len(tail) == 1:
		return true
	case tail[0] == skillKeySegmentCommand:
		return true
	case tail[0] == SkillSegmentPi || tail[0] == SkillSegmentAgents || tail[0] == SkillScopePiPrompt:
		return len(tail) == skillKeySegmentAndName
	case tail[0] == SkillScopePiSettings:
		return piConfiguredKeyTail(tail[1:])
	case tail[0] == string(SkillRootFormPiPkg):
		return piPackageKeyTail(tail[1:])
	default:
		return false
	}
}

// skillFixedScopeKeyParts are the key segments of the fixed scope IDs.
func skillFixedScopeKeyParts() map[string][]string {
	return map[string][]string{
		SkillScopeClaudeUser: {skillKeySegmentClaude},
		SkillScopeClaudeCmd:  {skillKeySegmentClaude},
		SkillScopePiPrompt:   nil,
		SkillScopePiSettings: nil,
		SkillScopeSynced:     {skillKeySegmentAnthropic},
		SkillScopePiUser:     {SkillSegmentPi},
		SkillScopeAgentsUser: {SkillSegmentAgents},
	}
}

// skillKeyFor returns candidate's key, or a non-empty problem explaining why
// it has none.
func skillKeyFor(candidate SkillCandidate) (key, problem string) {
	if strings.Contains(candidate.Name, skillKeySeparator) && candidate.Kind != SkillSourceKindCommand {
		return "", skillKeyNameColonProblem
	}

	parts, known := skillKeyPrefix(candidate)
	if !known {
		return "", fmt.Sprintf(skillKeyUnknownScope, candidate.ScopeID)
	}

	key = strings.Join(append(parts, candidate.Name), skillKeySeparator)
	if !parseSkillKey(key).recognized {
		return "", fmt.Sprintf(skillKeyUnrecognized, key)
	}

	return key, ""
}

// skillKeyFormParsers maps each top-level key form (design D3) — the first
// segments no plugin may use — to the parser of its tail: `claude:<n>` or
// `claude:cmd:<ns:…:n>`; one name after `pi:`, `agents:`, `pi-prompt:` and
// `anthropic-skills:`; a name or `pi-prompt:<n>` after `pi-settings:`; the
// same after `pi-pkg:<pkg-id>:`; and a project tail after `project:<r>:`.
func skillKeyFormParsers() map[string]func(tail []string) parsedSkillKey {
	return map[string]func(tail []string) parsedSkillKey{
		skillKeySegmentClaude: parseClaudeSkillKey,
		SkillSegmentPi: func(tail []string) parsedSkillKey {
			return fixedSkillKey(SkillRootFormPiUser, len(tail) == 1)
		},
		SkillSegmentAgents: func(tail []string) parsedSkillKey {
			return fixedSkillKey(SkillRootFormAgentsUser, len(tail) == 1)
		},
		SkillScopePiPrompt: func(tail []string) parsedSkillKey {
			return fixedSkillKey(SkillRootFormPiPrompt, len(tail) == 1)
		},
		skillKeySegmentAnthropic: func(tail []string) parsedSkillKey {
			return sourcedSkillKey(SkillRootFormSynced, SkillScopeSynced, len(tail) == 1)
		},
		SkillScopePiSettings: func(tail []string) parsedSkillKey {
			return sourcedSkillKey(SkillRootFormPiSettings, SkillScopePiSettings, piConfiguredKeyTail(tail))
		},
		string(SkillRootFormPiPkg): func(tail []string) parsedSkillKey {
			return sourcedSkillKey(SkillRootFormPiPkg, SkillScopePiPkgPrefix+tail[0], piPackageKeyTail(tail))
		},
		string(SkillRootFormProject): func(tail []string) parsedSkillKey {
			return sourcedSkillKey(SkillRootFormProject, SkillScopeProjectPrefix+tail[0],
				len(tail) > 1 && projectKeyTail(tail[1:]))
		},
	}
}

// skillKeyPrefix returns the key segments preceding candidate's name, and
// false for a scope D3 does not define (including a reserved plugin name).
func skillKeyPrefix(candidate SkillCandidate) ([]string, bool) {
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

// skillRootKeyPrefix is the key prefix of every candidate a root of scope,
// source segment and kind yields (design D3 without the name): e.g.
// `project:<r>:cmd:` or `pi-pkg:<id>:pi-prompt:`. A root vouches for the
// notes whose keys carry it (design D5).
func skillRootKeyPrefix(scope, segment string, kind SkillSourceKind) string {
	parts, _ := skillKeyPrefix(SkillCandidate{ScopeID: scope, SourceSegment: segment, Kind: kind})

	return strings.Join(append(parts, ""), skillKeySeparator)
}

// skillScopeKeyParts returns the key segments a scope ID contributes:
// `claude` for the Claude user skills and commands, none for pi-prompt and
// the Pi configured scopes (whose segment is SourceSegment), a fixed segment
// for synced, pi-user and agents-user, the plugin name, or the whole
// `project:<r>` scope. known is false for any other scope, and for a plugin
// scope with an empty or reserved name.
func skillScopeKeyParts(scope string) (parts []string, known bool) {
	if fixedParts, fixed := skillFixedScopeKeyParts()[scope]; fixed {
		return fixedParts, true
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

// sourcedSkillKey is a recognized (when shaped) source-rooted form key.
func sourcedSkillKey(form SkillRootForm, scopeID string, shaped bool) parsedSkillKey {
	if !shaped {
		return parsedSkillKey{}
	}

	return parsedSkillKey{recognized: true, form: form, sourced: true, scopeID: scopeID}
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
