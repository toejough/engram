package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Exported constants.
const (
	// SkillRootFormAgentsUser .. SkillRootFormSynced name the removal-
	// eligibility form of a ScannedRoot (design D5): which key form's notes
	// the root can prove absent. The five single-root user forms
	// (claude-user, claude-cmd, pi-user, agents-user, pi-prompt) are fixed
	// roots; synced (one per bucket), pi-settings (one per settings entry),
	// pi-pkg (one per package root) and project (one per project directory)
	// are source-rooted: a note is eligible only when the root holding its
	// recorded skill_source was read.
	SkillRootFormAgentsUser SkillRootForm = SkillScopeAgentsUser
	SkillRootFormClaudeCmd  SkillRootForm = SkillScopeClaudeCmd
	SkillRootFormClaudeUser SkillRootForm = SkillScopeClaudeUser
	SkillRootFormPiPkg      SkillRootForm = "pi-pkg"
	SkillRootFormPiPrompt   SkillRootForm = SkillScopePiPrompt
	SkillRootFormPiSettings SkillRootForm = SkillScopePiSettings
	SkillRootFormPiUser     SkillRootForm = SkillScopePiUser
	SkillRootFormProject    SkillRootForm = "project"
	SkillRootFormSynced     SkillRootForm = SkillScopeSynced
)

// SkillOfferComparison is CompareSkillOffers' result.
type SkillOfferComparison struct {
	// Offers are sorted by (ScopeID, Key).
	Offers []SkillOffer
	// Conflicts are the loud lines for every key conflict and plugin name
	// conflict (design D4). Any conflict makes the run fail after every other
	// offer is handled (ReportSkillOfferProblems).
	Conflicts []string
	// Warnings are non-failing lines: one per engram-owned bare key whose
	// copies differ in bytes (the first in precedence wins).
	Warnings []string
}

// SkillOfferInput is CompareSkillOffers' input.
type SkillOfferInput struct {
	// Vault is the vault root; Names is its ListMD-shaped listing of full .md
	// filenames, and ReadFile reads a vault-joined path.
	Vault    string
	Names    []string
	ReadFile func(string) ([]byte, error)
	// Declined is the decline state: skill key → declined hash.
	Declined map[string]string
	// Home expands a note's `~`-relative skill_source.
	Home string
	// Sources is ResolveSkillSources' output: keyed candidates, the roots
	// read (with their removal-eligibility forms), the engram-owned roots and
	// the plugin facts.
	Sources ResolvedSkillSources
	// NoRemovals suppresses every removal offer (`--skills-dir`, design D9).
	NoRemovals bool
}

// SkillRootForm is a ScannedRoot's removal-eligibility form (design D5).
type SkillRootForm string

// CompareSkillOffers computes the registration offers over keyed candidates
// (design D4, D5; skill-runbook-registration: "Registration SHALL offer,
// not act, and remember declines by hash").
//
// Candidates are first put in precedence order — design D2's source order
// (skillPrecedenceTier), then key, then source path — so the result never
// depends on the order the scanners listed them. Then, over the enabled
// candidates:
//   - entries resolving to one file collapse to the first;
//   - entries sharing a key collapse when byte-identical; when they differ,
//     the key is a conflict (a Conflicts line naming every path, no offer),
//     unless every copy lies under an engram-owned root, where the first
//     wins with one warning;
//   - a key's winner whose hash equals that of an earlier candidate or any
//     skill note's skill_hash is an alias and makes no offer;
//   - otherwise it offers Register (no note) or Refresh (a stale note),
//     unless that hash is declined for the key.
//
// A skill note is offered for removal only when no candidate at all —
// disabled, collapsed, conflicted or alias — holds its key, its key form's
// root was read (skillRemovalEligibility), and its skill_hash is not
// declined; NoRemovals suppresses every removal. A plugin name conflict adds
// a Conflicts line; the plugin scanner already emitted none of its
// candidates. Offers are sorted by (ScopeID, Key). A duplicate skill note
// aborts the comparison with errDuplicateSkillNote.
func CompareSkillOffers(input SkillOfferInput) (SkillOfferComparison, error) {
	notes, notesErr := resolveEverySkillNote(groupSkillNoteCandidates(input.Vault, input.Names, input.ReadFile))
	if notesErr != nil {
		return SkillOfferComparison{}, notesErr
	}

	groups, comparison := dedupeSkillCandidates(input.Sources.Candidates, input.Sources.EngramSkillRoots)
	comparison.Conflicts = append(pluginConflictLines(input.Sources.PluginConflicts), comparison.Conflicts...)
	comparison.Offers = candidateOffers(groups, notes, input.Declined)

	if !input.NoRemovals {
		comparison.Offers = append(comparison.Offers, removalOffers(input, notes)...)
	}

	for index := range comparison.Offers {
		comparison.Offers[index].EngramOwned = offerEngramOwned(comparison.Offers[index], input)
	}

	sort.Slice(comparison.Offers, func(i, j int) bool {
		left, right := comparison.Offers[i], comparison.Offers[j]
		if left.ScopeID != right.ScopeID {
			return left.ScopeID < right.ScopeID
		}

		return left.Key < right.Key
	})

	return comparison, nil
}

// ReportSkillOfferProblems prints comparison's warnings, then its conflict
// lines, and returns errSkillOfferConflict when there is any conflict
// (design D4: a failure status once every other offer is handled). Callers
// run it last, so a conflict never blocks another offer.
func ReportSkillOfferProblems(stdout io.Writer, comparison SkillOfferComparison) error {
	for _, line := range comparison.Warnings {
		_, _ = fmt.Fprintln(stdout, line)
	}

	for _, line := range comparison.Conflicts {
		_, _ = fmt.Fprintln(stdout, line)
	}

	if len(comparison.Conflicts) > 0 {
		return fmt.Errorf("%w: %d", errSkillOfferConflict, len(comparison.Conflicts))
	}

	return nil
}

// unexported constants.
const (
	// engramCopiesDifferWarningFormat is the one warning for an engram-owned
	// bare key whose copies differ (design D4's exception).
	engramCopiesDifferWarningFormat = "engram: engram skill %s differs between %s; using %s — " +
		"run `engram update` to resync the copies"
	// homeRelPrefix prefixes a `~`-relative skill_source.
	homeRelPrefix = "~" + string(filepath.Separator)
	// pluginConflictLineFormat reports a plugin name installed from several
	// marketplaces (design D4).
	pluginConflictLineFormat = "engram: plugin name conflict: %s in %s"
	// skillKeyConflictLineFormat reports a key whose copies differ (design D4).
	skillKeyConflictLineFormat = "engram: skill key conflict: %s at %s"
)

// skillPrecedence ranks a candidate's source (skillPrecedenceTier).
type skillPrecedence int

// skillPrecedence values.
const (
	precedenceClaudeUser skillPrecedence = iota
	precedenceClaudeCmd
	precedenceSynced
	precedencePiUser
	precedenceAgentsUser
	precedencePiSettingsSkill
	precedencePiPrompt
	precedencePiSettingsPrompt
	precedencePiPkg
	precedencePlugin
	precedenceProjectClaude
	precedenceProjectPi
	precedenceProjectAgents
	precedenceProjectPiSettingsSkill
	precedenceProjectPiPrompt
	precedenceProjectPiSettingsPrompt
	precedenceProjectPiPkg
	precedenceUnknown
)

// unexported variables.
var (
	// errSkillOfferConflict is the failure status a key or plugin name
	// conflict leaves after every other offer is handled (design D4).
	errSkillOfferConflict = errors.New("register-skills: skill conflicts")
)

// deepestRootMatch folds the roots containing a note's source into the
// deepest one's facts (sourceRootRead).
type deepestRootMatch struct {
	depth   int
	read    bool
	vouches bool
}

// consider folds root in when it contains path: a deeper root replaces the
// match; at equal depth every root must be read and one must vouch.
func (m *deepestRootMatch) consider(root ScannedRoot, path, key string) {
	depth, contains := rootDepthContaining(root, path)

	switch {
	case !contains || depth < m.depth:
	case depth == m.depth:
		m.read = m.read && root.Scanned
		m.vouches = m.vouches || rootVouchesFor(root, key)
	default:
		m.depth, m.read, m.vouches = depth, root.Scanned, rootVouchesFor(root, key)
	}
}

// skillKeyGroup is one key's enabled candidates, in precedence order, after
// same-path collapse. conflicted marks a key whose copies differ outside the
// engram-owned exception.
type skillKeyGroup struct {
	key        string
	members    []SkillCandidate
	conflicted bool
}

// skillRemovalEligibility decides design D5 removal eligibility for one run.
type skillRemovalEligibility struct {
	home          string
	roots         []ScannedRoot
	manifestsRead bool
	pluginScanned map[string]bool
	conflicted    map[string]bool
	// engramRoots are the resolved engram-owned skills roots, and
	// engramUnresolved is set when one of them could not be resolved: a
	// bare key also comes from copies under them (engramCopiesRead).
	engramRoots      []string
	engramUnresolved bool
}

// eligible reports whether the note with key and recorded skill_source may
// be offered for removal: its key form's root was read (design D5). An
// unrecognized key (parseSkillKey) is never eligible.
func (e skillRemovalEligibility) eligible(key, source string) bool {
	parsed := parseSkillKey(key)

	switch {
	case !parsed.recognized:
		return false
	case parsed.plugin != "":
		scanned, installed := e.pluginScanned[parsed.plugin]

		return e.manifestsRead && !e.conflicted[parsed.plugin] && (!installed || scanned)
	case parsed.form == SkillRootFormClaudeUser:
		return e.fixedRootRead(parsed.form) && e.engramCopiesRead()
	case !parsed.sourced:
		return e.fixedRootRead(parsed.form)
	default:
		return e.sourceRootRead(parsed.form, key, source)
	}
}

// engramCopiesRead reports whether every root that emitted, or could have
// emitted, an engram-owned copy (whose key is bare, like a Claude user
// skill's) was read (design D5): the Pi and agents user roots, every root
// overlapping an engram-owned skills root, and the engram-owned roots
// themselves (none failed to resolve).
func (e skillRemovalEligibility) engramCopiesRead() bool {
	if e.engramUnresolved {
		return false
	}

	for _, root := range e.roots {
		if root.Scanned {
			continue
		}

		if root.Form == SkillRootFormPiUser || root.Form == SkillRootFormAgentsUser ||
			rootOverlapsAny(root, e.engramRoots) {
			return false
		}
	}

	return true
}

// fixedRootRead reports whether form's fixed user root was recorded and
// every root recorded with that form was read.
func (e skillRemovalEligibility) fixedRootRead(form SkillRootForm) bool {
	found := false

	for _, root := range e.roots {
		if root.Form != form {
			continue
		}

		if !root.Scanned {
			return false
		}

		found = true
	}

	return found
}

// sourceRootRead reports whether the deepest root of form containing the
// note's skill_source was read and vouches for the note's key (design D5:
// the settings entry, the package root, the exact project directory). A
// source under no root of that form (or no source at all) is never
// eligible; a deeper root of another scope or segment hides a shallower
// one; at equal depth every root must be read and one must vouch.
func (e skillRemovalEligibility) sourceRootRead(form SkillRootForm, key, source string) bool {
	if source == "" {
		return false
	}

	path := expandHomeRel(source, e.home)
	best := deepestRootMatch{depth: -1}

	for _, root := range e.roots {
		if root.Form == form {
			best.consider(root, path, key)
		}
	}

	return best.depth >= 0 && best.read && best.vouches
}

// candidateOffers turns each non-conflicted, non-alias key group's winner
// into a Register or Refresh offer, in group (precedence) order.
func candidateOffers(
	groups []skillKeyGroup, notes map[string]skillNoteCandidate, declined map[string]string,
) []SkillOffer {
	noteHashes := make(map[string]bool, len(notes))
	for _, note := range notes {
		noteHashes[note.Hash] = true
	}

	seen := map[string]bool{}
	offers := make([]SkillOffer, 0, len(groups))

	for _, group := range groups {
		winner := group.members[0]
		hash := SkillContentHash(winner.Content)
		alias := seen[hash] || noteHashes[hash]

		for _, member := range group.members {
			seen[SkillContentHash(member.Content)] = true
		}

		if group.conflicted || alias || declined[group.key] == hash {
			continue
		}

		offer := SkillOffer{
			Kind: SkillOfferRegister, Key: group.key, ScopeID: winner.ScopeID, SourcePath: winner.SourcePath, Hash: hash,
		}

		if note, found := notes[group.key]; found {
			if note.Hash == hash {
				continue
			}

			offer.Kind = SkillOfferRefresh
			offer.Basename = note.Basename
		}

		offers = append(offers, offer)
	}

	return offers
}

// dedupeSkillCandidates orders the enabled candidates by precedence,
// collapses entries resolving to one file, and groups the rest by key,
// marking conflicts. It returns the groups in precedence order and a
// comparison holding the key-conflict lines and engram-copy warnings.
func dedupeSkillCandidates(
	candidates []SkillCandidate, engramRoots []string,
) ([]skillKeyGroup, SkillOfferComparison) {
	active := make([]SkillCandidate, 0, len(candidates))

	for _, candidate := range candidates {
		if !candidate.Disabled {
			active = append(active, candidate)
		}
	}

	sort.SliceStable(active, func(i, j int) bool { return skillCandidateLess(active[i], active[j]) })

	seenPaths := make(map[string]bool, len(active))
	groupIndex := map[string]int{}
	groups := make([]skillKeyGroup, 0, len(active))

	for _, candidate := range active {
		if seenPaths[candidate.SourcePath] {
			continue
		}

		seenPaths[candidate.SourcePath] = true

		index, known := groupIndex[candidate.Key]
		if !known {
			index = len(groups)
			groupIndex[candidate.Key] = index
			groups = append(groups, skillKeyGroup{key: candidate.Key})
		}

		groups[index].members = append(groups[index].members, candidate)
	}

	var comparison SkillOfferComparison

	for index := range groups {
		judgeSkillKeyGroup(&groups[index], engramRoots, &comparison)
	}

	return groups, comparison
}

// expandHomeRel expands a `~/`-relative path against home; without a home
// it is left as is, so it matches no absolute root.
func expandHomeRel(path, home string) string {
	rest, homeRelative := strings.CutPrefix(path, homeRelPrefix)
	if !homeRelative || home == "" {
		return path
	}

	return filepath.Join(home, rest)
}

// joinPathList renders two or more paths as "a and b" or "a, b and c".
func joinPathList(paths []string) string {
	return strings.Join(paths[:len(paths)-1], ", ") + " and " + paths[len(paths)-1]
}

// judgeSkillKeyGroup marks group conflicted when its members' bytes differ,
// unless every member lies under an engram-owned root (design D4's only
// exception), where the first wins and one warning is recorded.
func judgeSkillKeyGroup(group *skillKeyGroup, engramRoots []string, comparison *SkillOfferComparison) {
	first := SkillContentHash(group.members[0].Content)
	differs := false
	allEngram := true
	paths := make([]string, 0, len(group.members))

	for _, member := range group.members {
		differs = differs || SkillContentHash(member.Content) != first
		allEngram = allEngram && underEngramSkillRoot(member.SourcePath, engramRoots)
		paths = append(paths, member.SourcePath)
	}

	if !differs {
		return
	}

	slices.Sort(paths)

	if allEngram {
		comparison.Warnings = append(comparison.Warnings, fmt.Sprintf(engramCopiesDifferWarningFormat,
			group.key, joinPathList(paths), group.members[0].SourcePath))

		return
	}

	group.conflicted = true
	comparison.Conflicts = append(comparison.Conflicts,
		fmt.Sprintf(skillKeyConflictLineFormat, group.key, joinPathList(paths)))
}

// newSkillRemovalEligibility indexes input's read roots and plugin facts.
func newSkillRemovalEligibility(input SkillOfferInput) skillRemovalEligibility {
	eligibility := skillRemovalEligibility{
		home:          input.Home,
		roots:         input.Sources.Roots,
		manifestsRead: input.Sources.PluginManifestsRead,
		pluginScanned: make(map[string]bool, len(input.Sources.Plugins)),
		conflicted:    make(map[string]bool, len(input.Sources.PluginConflicts)),

		engramRoots:      input.Sources.EngramSkillRoots,
		engramUnresolved: input.Sources.EngramSkillRootsUnresolved,
	}

	for _, plugin := range input.Sources.Plugins {
		eligibility.pluginScanned[plugin.Name] = plugin.Scanned
	}

	for _, conflict := range input.Sources.PluginConflicts {
		eligibility.conflicted[conflict.Plugin] = true
	}

	return eligibility
}

// offerEngramOwned reports whether offer's source lies under an engram-owned
// skills root: a candidate's resolved path, or a removal's recorded
// skill_source expanded against the resolved home or the home as given.
func offerEngramOwned(offer SkillOffer, input SkillOfferInput) bool {
	roots := input.Sources.EngramSkillRoots

	if offer.Kind != SkillOfferRemove {
		return underEngramSkillRoot(offer.SourcePath, roots)
	}

	if offer.SourcePath == "" {
		return false
	}

	for _, home := range []string{input.Sources.ResolvedHome, input.Home} {
		if underEngramSkillRoot(expandHomeRel(offer.SourcePath, home), roots) {
			return true
		}
	}

	return false
}

// pluginConflictLines renders one line per plugin name conflict, sorted by
// plugin name.
func pluginConflictLines(conflicts []ClaudePluginConflict) []string {
	lines := make([]string, 0, len(conflicts))

	for _, conflict := range conflicts {
		marketplaces := slices.Clone(conflict.Marketplaces)
		slices.Sort(marketplaces)
		lines = append(lines, fmt.Sprintf(pluginConflictLineFormat, conflict.Plugin, strings.Join(marketplaces, ", ")))
	}

	slices.Sort(lines)

	return lines
}

// removalOffers offers to remove each skill note whose key no candidate
// holds (disabled ones included) and whose key form's root was read.
func removalOffers(input SkillOfferInput, notes map[string]skillNoteCandidate) []SkillOffer {
	present := make(map[string]bool, len(input.Sources.Candidates))
	for _, candidate := range input.Sources.Candidates {
		present[candidate.Key] = true
	}

	eligibility := newSkillRemovalEligibility(input)
	offers := make([]SkillOffer, 0, len(notes))

	for key, note := range notes {
		if present[key] || !eligibility.eligible(key, note.Source) || input.Declined[key] == note.Hash {
			continue
		}

		offers = append(offers, SkillOffer{
			Kind:       SkillOfferRemove,
			Key:        key,
			ScopeID:    parseSkillKey(key).scopeID,
			SourcePath: note.Source,
			Basename:   note.Basename,
			Hash:       note.Hash,
		})
	}

	return offers
}

// resolveEverySkillNote reduces every key's note matches to its one note,
// failing (in key order, for a deterministic message) on the first key with
// several notes.
func resolveEverySkillNote(byKey map[string][]skillNoteCandidate) (map[string]skillNoteCandidate, error) {
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	notes := make(map[string]skillNoteCandidate, len(keys))

	for _, key := range keys {
		matches := byKey[key]

		switch len(matches) {
		case 0:
		case 1:
			notes[key] = matches[0]
		default:
			_, _, _, err := resolveSkillNoteMatches(matches)

			return nil, err
		}
	}

	return notes, nil
}

// rootDepthContaining reports whether path lies at or under root (by its
// resolved path or its path as given) and the length of the matching root
// path, used to pick the deepest containing root.
func rootDepthContaining(root ScannedRoot, path string) (int, bool) {
	depth, contains := -1, false

	for _, rootPath := range []string{root.Resolved, root.Path} {
		if rootPath != "" && pathWithinRoot(path, rootPath) && len(rootPath) > depth {
			depth, contains = len(rootPath), true
		}
	}

	return depth, contains
}

// rootOverlapsAny reports whether root (by its resolved path or its path as
// given) lies at or under one of paths, or holds one of them.
func rootOverlapsAny(root ScannedRoot, paths []string) bool {
	for _, rootPath := range []string{root.Resolved, root.Path} {
		if rootPath == "" {
			continue
		}

		for _, path := range paths {
			if pathWithinRoot(rootPath, path) || pathWithinRoot(path, rootPath) {
				return true
			}
		}
	}

	return false
}

// rootVouchesFor reports whether key is one of root's key prefixes followed
// by a name: one segment, or — after a `cmd:` segment — a command's
// `:`-joined namespace path.
func rootVouchesFor(root ScannedRoot, key string) bool {
	commandSuffix := skillKeySegmentCommand + skillKeySeparator

	for _, prefix := range root.KeyPrefixes {
		name, found := strings.CutPrefix(key, prefix)
		if !found || name == "" {
			continue
		}

		isCommandPrefix := prefix == commandSuffix || strings.HasSuffix(prefix, skillKeySeparator+commandSuffix)
		if isCommandPrefix || !strings.Contains(name, skillKeySeparator) {
			return true
		}
	}

	return false
}

// skillCandidateLess orders candidates by design D2 precedence, then key,
// then source path, enabled before disabled — a total order over distinct
// candidates, so the comparison never depends on scan order.
func skillCandidateLess(left, right SkillCandidate) bool {
	leftTier, rightTier := skillPrecedenceTier(left), skillPrecedenceTier(right)

	switch {
	case leftTier != rightTier:
		return leftTier < rightTier
	case left.Key != right.Key:
		return left.Key < right.Key
	case left.SourcePath != right.SourcePath:
		return left.SourcePath < right.SourcePath
	default:
		return !left.Disabled && right.Disabled
	}
}

// skillPrecedenceTier ranks a candidate's source by design D2 precedence,
// mirroring ResolveSkillSources' order: Claude user skills, commands,
// synced, Pi user, agents user, Pi settings skills, Pi prompt templates,
// Pi settings prompts, Pi packages, plugins, then the project sources
// (Claude, then Pi in the same order as the global Pi sources).
func skillPrecedenceTier(candidate SkillCandidate) skillPrecedence {
	isPrompt := candidate.Kind == SkillSourceKindPrompt

	if candidate.ScopeID == SkillScopePiSettings && isPrompt {
		return precedencePiSettingsPrompt
	}

	if tier, fixed := skillScopePrecedences()[candidate.ScopeID]; fixed {
		return tier
	}

	switch {
	case strings.HasPrefix(candidate.ScopeID, SkillScopePiPkgPrefix):
		return precedencePiPkg
	case strings.HasPrefix(candidate.ScopeID, SkillScopePluginPrefix):
		return precedencePlugin
	case strings.HasPrefix(candidate.ScopeID, SkillScopeProjectPrefix):
		return skillProjectPrecedence(candidate.SourceSegment, isPrompt)
	default:
		return precedenceUnknown
	}
}

// skillProjectPrecedence ranks a project candidate's sub-source: Claude
// project skills and commands, `.pi/skills`, `.agents/skills`, settings
// skills, `.pi/prompts`, settings prompts, packages (design D2 sources 9
// and 10).
func skillProjectPrecedence(segment string, isPrompt bool) skillPrecedence {
	switch {
	case segment == "":
		return precedenceProjectClaude
	case segment == SkillSegmentPi:
		return precedenceProjectPi
	case segment == SkillSegmentAgents:
		return precedenceProjectAgents
	case segment == SkillScopePiSettings && !isPrompt:
		return precedenceProjectPiSettingsSkill
	case segment == SkillScopePiPrompt:
		return precedenceProjectPiPrompt
	case segment == SkillScopePiSettings:
		return precedenceProjectPiSettingsPrompt
	default:
		return precedenceProjectPiPkg
	}
}

// skillScopePrecedences ranks the fixed scope IDs (design D2); the Pi
// settings scope ranks its skills here and its prompts separately.
func skillScopePrecedences() map[string]skillPrecedence {
	return map[string]skillPrecedence{
		SkillScopeClaudeUser: precedenceClaudeUser,
		SkillScopeClaudeCmd:  precedenceClaudeCmd,
		SkillScopeSynced:     precedenceSynced,
		SkillScopePiUser:     precedencePiUser,
		SkillScopeAgentsUser: precedenceAgentsUser,
		SkillScopePiSettings: precedencePiSettingsSkill,
		SkillScopePiPrompt:   precedencePiPrompt,
	}
}
