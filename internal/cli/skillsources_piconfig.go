package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Exported constants.
const (
	// SkillScopePiPkgPrefix prefixes a Pi package's scope ID and key segment:
	// `pi-pkg:<pkg-id>` (design D3).
	SkillScopePiPkgPrefix = "pi-pkg:"
	// SkillScopePiSettings is the scope ID (and key segment) of Pi's global
	// settings.json `skills`/`prompts` entries (design D3 `pi-settings`).
	SkillScopePiSettings = "pi-settings"
)

// PiConfiguredScan is the input to ScanPiConfiguredSources.
type PiConfiguredScan struct {
	// Home expands a leading `~` in settings paths.
	Home string
	// AgentDir is ~/.pi/agent: the global settings dir, the base for global
	// relative entries, and the global npm/git install root.
	AgentDir string
	// Cwd is the working directory; project settings are <Cwd>/.pi/settings.json
	// and project entries resolve against <Cwd>/.pi.
	Cwd string
	// ScopeID is the project scope (`project:<r>`) stamped on project
	// candidates.
	ScopeID string
	// Trusted is Pi's project-trust decision (ResolvePiProjectTrust). When
	// false the project settings are never read.
	Trusted bool
}

// PiSettings is the part of one Pi settings.json that registration uses.
type PiSettings struct {
	// Skills and Prompts are the raw `skills`/`prompts` entries: paths plus
	// Pi's glob, `!pattern`, `+path` and `-path` patterns. They are also the
	// overrides ApplyPiSettingsOverrides applies to the default folders.
	Skills  []string
	Prompts []string
	// packages are the `packages` entries, in order.
	packages []piPackageEntry
}

// ApplyPiSettingsOverrides applies one settings file's `skills` or `prompts`
// entries to candidates found in Pi's default folders (ruling R8; Pi's
// isEnabledByOverrides, dist/core/package-manager.js:515-539, applied in
// addAutoDiscoveredResources :1904-1985). Only override entries act: a
// `!pattern` excludes, then `+path` re-enables and `-path` force-excludes by
// exact path. Plain entries and include globs never filter default folders.
//
// Patterns match a candidate's WalkedPath (the path as discovered, not
// symlink-resolved) relative to baseDir, its file name, or its absolute path,
// and — for a SKILL.md — the same three forms of its directory. baseDir is
// the dir Pi matches against: ~/.pi/agent for ~/.pi/agent/skills and
// ~/.pi/agent/prompts (global settings); ~/.agents for ~/.agents/skills
// (global settings); <cwd>/.pi for .pi/skills and .pi/prompts, and each
// <dir>/.agents for the project `.agents/skills` chain (project settings).
//
// It returns a copy with Disabled set on every candidate Pi switches off. A
// Disabled candidate must never be offered, but it still counts as present
// for removal (a disabled skill keeps its note). An override containing `**`
// cannot be evaluated: every candidate is then marked Disabled, with a
// warning, so nothing is offered and nothing is removed.
func ApplyPiSettingsOverrides(
	candidates []SkillCandidate, patterns []string, baseDir string,
) ([]SkillCandidate, []string) {
	out := slices.Clone(candidates)

	overrides := slices.DeleteFunc(slices.Clone(patterns), func(pattern string) bool {
		return !isPiOverridePattern(pattern)
	})

	if hasPiGlobstar(overrides) {
		for index := range out {
			out[index].Disabled = true
		}

		return out, []string{fmt.Sprintf(piGlobstarOverrideWarningFormat, baseDir)}
	}

	split := splitPiPatterns(overrides)

	for index := range out {
		target := newPiPatternTarget(candidateWalkedPath(out[index]), baseDir)

		enabled := !target.matchesAnyGlob(split.excludes)
		if target.matchesAnyExact(split.forceIncludes) {
			enabled = true
		}

		if target.matchesAnyExact(split.forceExcludes) {
			enabled = false
		}

		out[index].Disabled = out[index].Disabled || !enabled
	}

	return out, nil
}

// ReadPiSettings reads one Pi settings.json. A missing file is empty
// settings. A file that cannot be read or parsed is empty settings plus a
// warning (Pi itself then runs with empty settings); a `skills`, `prompts`
// or `packages` value of the wrong shape is ignored with a warning. The
// legacy `skills: {customDirectories: [...]}` form is migrated as Pi's
// migrateSettings does (dist/core/settings-manager.js:207-221).
func ReadPiSettings(fsys SkillSourceFS, settingsPath string) (PiSettings, []string) {
	object, err := readPiJSONObject(fsys, settingsPath)
	if err != nil {
		return PiSettings{}, []string{fmt.Sprintf(piSettingsUnreadableWarningFormat, settingsPath, err)}
	}

	var (
		settings PiSettings
		warnings []string
	)

	warn := func(key string, decodeErr error) {
		warnings = append(warnings, fmt.Sprintf(piSettingsKeyWarningFormat, settingsPath, key, decodeErr))
	}

	settings.Skills, err = decodePiSkillsSetting(object[piSkillsDirName])
	if err != nil {
		warn(piSkillsDirName, err)
	}

	settings.Prompts, err = decodePiStringList(object[piPromptsDirName])
	if err != nil {
		warn(piPromptsDirName, err)
	}

	settings.packages, err = decodePiPackages(object[piPackagesKey])
	if err != nil {
		warn(piPackagesKey, err)
	}

	return settings, warnings
}

// ScanPiConfiguredSources scans Pi's configured sources (design D2 sources
// 6, 6b, 7 and the settings/package part of 10): the global
// <AgentDir>/settings.json `skills`/`prompts` entries and `packages`, plus —
// only when scan.Trusted — the same keys of <Cwd>/.pi/settings.json.
//
// Settings entries (Pi's resolveLocalEntries, package-manager.js:1890-1903):
// plain entries are paths (relative to the settings dir, `~` expanded); a
// directory is searched recursively (skills: ScanPiSkillDir with root `.md`
// files, :241; prompts: recursive `*.md`, :149-192), a `.md` file is one
// resource. Entries that are patterns (Pi's isPattern, :127-129: a `!`, `+`
// or `-` prefix, or a `*`/`?`) filter the collected files (applyPatterns,
// :540-584, with Go path.Match per segment); a filtered-out file is kept as
// a Disabled candidate. A pattern containing `**` is unsupported: warning,
// and every entry root of that list is recorded not scanned. Each plain
// entry is its own root (its resolved path), and every candidate's ReadRoot
// is that root.
//
// Packages (package-manager.js:977-1033, collectPackageResources
// :1747-1785): see scanPiPackage. A project entry replaces a global entry
// with the same identity (dedupePackages, :1386-1407).
//
// Global candidates carry ScopeID SkillScopePiSettings or
// `pi-pkg:<pkg-id>`; project candidates carry scan.ScopeID. Every candidate
// carries SourceSegment (`pi-settings` or `pi-pkg:<pkg-id>`) for key
// construction (task 2.1). Keys are left empty.
func ScanPiConfiguredSources(fsys SkillSourceFS, scan PiConfiguredScan) SkillScanResult {
	var result SkillScanResult

	global, warnings := ReadPiSettings(fsys, filepath.Join(scan.AgentDir, piSettingsFilename))
	result.Warnings = append(result.Warnings, warnings...)

	var project PiSettings

	projectBase := filepath.Join(scan.Cwd, piConfigDirName)

	if scan.Trusted {
		project, warnings = ReadPiSettings(fsys, filepath.Join(projectBase, piSettingsFilename))
		result.Warnings = append(result.Warnings, warnings...)
	}

	scopes := []piSettingsScope{
		{settings: global, baseDir: scan.AgentDir, scopeID: SkillScopePiSettings, home: scan.Home},
		{settings: project, baseDir: projectBase, scopeID: scan.ScopeID, home: scan.Home, project: true},
	}

	for _, scope := range scopes {
		mergeSkillScanResult(&result, scope.scanEntries(fsys, scope.settings.Skills, SkillSourceKindSkill))
		mergeSkillScanResult(&result, scope.scanEntries(fsys, scope.settings.Prompts, SkillSourceKindPrompt))
	}

	for _, pkg := range dedupePiPackages(scopes) {
		mergeSkillScanResult(&result, pkg.scan(fsys))
	}

	sortSkillCandidates(result.Candidates)

	return result
}

// unexported constants.
const (
	piAutoloadWarningFormat = "engram: Pi package %s has `autoload: false` (a delta engram does not " +
		"evaluate); it is not scanned"
	piFileURLPrefix                 = "file://"
	piGitDirName                    = "git"
	piGlobstar                      = "**"
	piGlobstarOverrideWarningFormat = "engram: Pi settings override for %s uses unsupported `**`; " +
		"its default-folder entries are treated as disabled"
	piGlobstarWarningFormat = "engram: Pi %s uses an unsupported `**` pattern; it is not scanned"
	piManifestFilename      = "package.json"
	piManifestKey           = "pi"
	piMinGitPathSegments    = 2
	piNodeModulesPath       = "node_modules"
	piNpmDirName            = "npm"
	piNpmPrefix             = "npm:"
	piPackagesKey           = "packages"
	// piSettingsKeyWarningFormat reports a settings key of an unusable shape.
	piSettingsKeyWarningFormat        = "engram: Pi settings %s: ignoring `%s`: %v"
	piSettingsUnreadableWarningFormat = "engram: cannot read Pi settings %s: %v; its sources are not scanned"
	piTildePrefix                     = "~/"
	piURLSchemeSeparator              = "://"
)

// unexported variables.
var (
	errPiNotStringList   = errors.New("expected an array of strings")
	errPiPackageShape    = errors.New("expected a string or an object with a string `source`")
	errPiPromptTooDeep   = errors.New("pi prompt directory nesting exceeds the depth limit")
	piGitProtocolPattern = regexp.MustCompile(`(?i)^(https?|ssh|git)://`)
	// piNpmSpecPattern is Pi's parseNpmSpec (package-manager.js:1408-1416).
	piNpmSpecPattern = regexp.MustCompile(`^(@?[^@]+(?:/[^@]+)?)(?:@(.+))?$`)
	piScpGitPattern  = regexp.MustCompile(`^git@([^:]+):(.+)$`)
)

// piCollected is one plain path's collection: its candidates, the resolved
// path, whether it exists, and whether every read under it succeeded.
type piCollected struct {
	candidates []SkillCandidate
	resolved   string
	found      bool
	ok         bool
	warnings   []string
}

// piManifest is package.json's `pi` value. present is Pi's truthiness of
// `pkg.pi ?? null`; entries holds each resource type's list (nil = absent).
type piManifest struct {
	present bool
	entries map[SkillSourceKind][]string
}

// piPackage is one configured package after source parsing and dedupe.
type piPackage struct {
	entry     piPackageEntry
	root      string
	pkgID     string
	identity  string
	scopeID   string
	skipRoots []string
}

// collect gathers the package's skills and prompts into result, returning
// false when any read failed or a pattern was unsupported.
func (p piPackage) collect(fsys SkillSourceFS, result *SkillScanResult) bool {
	manifest, err := readPiManifest(fsys, filepath.Join(p.root, piManifestFilename))
	if err != nil {
		warnUnlessNotExist(result, filepath.Join(p.root, piManifestFilename), err)

		return false
	}

	ok := true

	for _, kind := range []SkillSourceKind{SkillSourceKindSkill, SkillSourceKindPrompt} {
		candidates, kindOK, warnings := p.collectKind(fsys, manifest, kind)
		result.Candidates = append(result.Candidates, candidates...)
		result.Warnings = append(result.Warnings, warnings...)
		ok = ok && kindOK
	}

	return ok
}

// collectKind gathers one resource type per the entry form (see scan).
func (p piPackage) collectKind(
	fsys SkillSourceFS, manifest piManifest, kind SkillSourceKind,
) ([]SkillCandidate, bool, []string) {
	if p.entry.filter != nil {
		if patterns, set := p.entry.filter.patterns[kind]; set {
			return p.filtered(fsys, manifest, kind, patterns)
		}

		if entries, listed := manifest.entries[kind]; listed {
			return p.manifestEntries(fsys, entries, kind)
		}

		return p.convention(fsys, kind)
	}

	if manifest.present {
		return p.manifestEntries(fsys, manifest.entries[kind], kind)
	}

	return p.convention(fsys, kind)
}

// convention collects the type's convention dir (`skills/` or `prompts/`).
func (p piPackage) convention(fsys SkillSourceFS, kind SkillSourceKind) ([]SkillCandidate, bool, []string) {
	collected := collectPiPath(fsys, filepath.Join(p.root, piKindDirName(kind)), kind)

	return collected.candidates, collected.ok, collected.warnings
}

// filtered applies an object-form entry's explicit pattern list to the
// package's files for kind (Pi's applyPackageFilter + collectManifestFiles,
// :1802-1848): every file stays, filtered-out ones marked Disabled.
func (p piPackage) filtered(
	fsys SkillSourceFS, manifest piManifest, kind SkillSourceKind, patterns []string,
) ([]SkillCandidate, bool, []string) {
	if hasPiGlobstar(patterns) {
		return nil, false, []string{fmt.Sprintf(piGlobstarWarningFormat, "package "+p.entry.source)}
	}

	var (
		candidates []SkillCandidate
		ok         bool
		warnings   []string
	)

	if entries := manifest.entries[kind]; len(entries) > 0 {
		candidates, ok, warnings = p.manifestEntries(fsys, entries, kind)
	} else {
		candidates, ok, warnings = p.convention(fsys, kind)
	}

	if len(patterns) == 0 {
		for index := range candidates {
			candidates[index].Disabled = true
		}

		return candidates, ok, warnings
	}

	enabled := applyPiPatterns(candidates, patterns, p.root)
	for index := range candidates {
		candidates[index].Disabled = !enabled[index]
	}

	return candidates, ok, warnings
}

// manifestEntries collects a manifest entry list (Pi's
// collectFilesFromManifestEntries + addManifestEntries, :1863-1889): plain
// entries resolve against the package root, globs expand per segment, and
// the list's own `!`/`+`/`-` patterns drop files outright.
func (p piPackage) manifestEntries(
	fsys SkillSourceFS, entries []string, kind SkillSourceKind,
) ([]SkillCandidate, bool, []string) {
	if hasPiGlobstar(entries) {
		return nil, false, []string{fmt.Sprintf(piGlobstarWarningFormat, "package "+p.entry.source)}
	}

	var (
		candidates []SkillCandidate
		warnings   []string
	)

	ok := true
	overrides := []string{}

	for _, entry := range entries {
		if isPiOverridePattern(entry) {
			overrides = append(overrides, entry)

			continue
		}

		paths, globOK, globWarnings := expandPiManifestEntry(fsys, p.root, entry)
		warnings = append(warnings, globWarnings...)
		ok = ok && globOK

		for _, match := range paths {
			collected := collectPiPath(fsys, match, kind)
			candidates = append(candidates, collected.candidates...)
			warnings = append(warnings, collected.warnings...)
			ok = ok && collected.ok
		}
	}

	candidates = dedupePiWalked(candidates)
	enabled := applyPiPatterns(candidates, overrides, p.root)

	kept := make([]SkillCandidate, 0, len(candidates))

	for index, candidate := range candidates {
		if enabled[index] {
			kept = append(kept, candidate)
		}
	}

	return kept, ok, warnings
}

// scan collects one package's skills and prompts per Pi's
// collectPackageResources (package-manager.js:1747-1785):
//
//   - `autoload: false` → warning; the root (and any global entry it is a
//     delta over) is recorded not scanned.
//   - A missing root is recorded not scanned, silently. A root that is a file
//     (an extension) has no skills or prompts.
//   - An unreadable package.json → warning, not scanned. An unparseable one
//     is no manifest, as in Pi's readPiManifest (:1849-1862).
//   - Object form: per type, an explicit list filters the package's files
//     (applyPackageFilter :1802-1817; `[]` disables all) and filtered-out
//     files are Disabled candidates; an omitted type falls back to the
//     manifest entry, else the convention dir (collectDefaultResources
//     :1786-1801).
//   - String form: any truthy `pi` value → only the manifest's entries load
//     (addManifestEntries :1863-1874, whose own `!`/`+`/`-` patterns drop
//     files outright); no `pi` value → the convention dirs `skills/` and
//     `prompts/`.
//
// The package root is the one ScannedRoot; every candidate's ReadRoot is the
// resolved package root.
func (p piPackage) scan(fsys SkillSourceFS) SkillScanResult {
	var result SkillScanResult

	form := SkillRootFormPiPkg
	if strings.HasPrefix(p.scopeID, SkillScopeProjectPrefix) {
		form = SkillRootFormProject
	}

	segment := SkillScopePiPkgPrefix + p.pkgID
	prefixes := []string{
		skillRootKeyPrefix(p.scopeID, segment, SkillSourceKindSkill),
		skillRootKeyPrefix(p.scopeID, segment, SkillSourceKindPrompt),
	}

	for _, skipped := range p.skipRoots {
		result.Roots = append(result.Roots, ScannedRoot{Path: skipped, Form: form, KeyPrefixes: prefixes})
	}

	record := ScannedRoot{Path: p.root, Form: form, KeyPrefixes: prefixes}

	if p.entry.filter != nil && p.entry.filter.autoloadOff {
		result.Warnings = append(result.Warnings, fmt.Sprintf(piAutoloadWarningFormat, p.entry.source))
		result.Roots = append(result.Roots, record)

		return result
	}

	resolved, info, err := resolveSkillPathInfo(fsys, p.root)
	if err != nil {
		warnUnlessNotExist(&result, p.root, err)
		result.Roots = append(result.Roots, record)

		return result
	}

	record.Resolved = resolved
	record.Scanned = true

	if info.IsDir() {
		record.Scanned = p.collect(fsys, &result)
		stampPiCandidates(result.Candidates, p.scopeID, SkillScopePiPkgPrefix+p.pkgID, resolved)
	}

	result.Roots = append(result.Roots, record)

	return result
}

// piPackageEntry is one `packages` entry: a string source, or an object
// (filter non-nil).
type piPackageEntry struct {
	source string
	filter *piPackageFilter
}

// piPackageFilter is an object-form entry's per-type pattern lists (a type
// absent from patterns was omitted) and its `autoload: false` flag.
type piPackageFilter struct {
	patterns    map[SkillSourceKind][]string
	autoloadOff bool
}

// piPatternSplit is a pattern list split by kind (Pi's applyPatterns).
type piPatternSplit struct {
	includes, excludes, forceIncludes, forceExcludes []string
}

// piPatternTarget holds the forms of one file path Pi matches patterns
// against (matchesAnyPattern / matchesAnyExactPattern,
// package-manager.js:465-511).
type piPatternTarget struct {
	rel, name, abs              string
	skill                       bool
	parentRel, parentName, pAbs string
}

// matchesAnyExact reports whether any `+`/`-` path equals the target's
// relative or absolute path (or, for a SKILL.md, its directory's).
func (t piPatternTarget) matchesAnyExact(patterns []string) bool {
	for _, pattern := range patterns {
		normalized := filepath.ToSlash(strings.TrimPrefix(pattern, "./"))
		if normalized == t.rel || normalized == t.abs {
			return true
		}

		if t.skill && (normalized == t.parentRel || normalized == t.pAbs) {
			return true
		}
	}

	return false
}

// matchesAnyGlob reports whether any pattern path.Matches one of the
// target's forms. A malformed pattern matches nothing.
func (t piPatternTarget) matchesAnyGlob(patterns []string) bool {
	forms := []string{t.rel, t.name, t.abs}
	if t.skill {
		forms = append(forms, t.parentRel, t.parentName, t.pAbs)
	}

	for _, pattern := range patterns {
		normalized := filepath.ToSlash(pattern)

		for _, form := range forms {
			if matched, _ := path.Match(normalized, form); matched {
				return true
			}
		}
	}

	return false
}

// piPromptWalker carries one recursive prompt-dir walk (Pi's collectFiles,
// package-manager.js:149-192): `.md` files at any depth, skipping
// `.`-prefixed entries and node_modules, following symlinks.
type piPromptWalker struct {
	fsys      SkillSourceFS
	readRoot  string
	result    *SkillScanResult
	ancestors map[string]bool
}

// descend visits one resolved subdirectory unless it is on the descent path
// (a cycle, skipped) or too deep (reported).
func (w piPromptWalker) descend(path, resolved string, depth int) bool {
	if w.ancestors[resolved] {
		return true
	}

	if depth > maxPiSkillDepth {
		return w.fail(path, errPiPromptTooDeep)
	}

	entries, readErr := w.fsys.ReadDir(resolved)
	if readErr != nil {
		return w.fail(path, readErr)
	}

	w.ancestors[resolved] = true
	defer delete(w.ancestors, resolved)

	return w.visit(path, resolved, entries, depth)
}

// entry handles one directory entry (walked as path, located at entryPath
// under the resolved parent).
func (w piPromptWalker) entry(path, entryPath string, depth int) bool {
	resolved, info, resolveErr := resolveSkillPathInfo(w.fsys, entryPath)
	if resolveErr != nil {
		if errors.Is(resolveErr, fs.ErrNotExist) {
			return true
		}

		return w.fail(path, resolveErr)
	}

	if info.IsDir() {
		return w.descend(path, resolved, depth+1)
	}

	if !strings.HasSuffix(path, markdownFileExt) {
		return true
	}

	content, readErr := w.fsys.ReadFile(resolved)
	if readErr != nil {
		if errors.Is(readErr, fs.ErrNotExist) {
			return true
		}

		return w.fail(path, readErr)
	}

	w.result.Candidates = append(w.result.Candidates, SkillCandidate{
		Name:       strings.TrimSuffix(filepath.Base(path), markdownFileExt),
		ReadRoot:   w.readRoot,
		SourcePath: resolved,
		WalkedPath: path,
		Kind:       SkillSourceKindPrompt,
		Content:    content,
	})

	return true
}

// fail records a warning for path and returns false (not scanned).
func (w piPromptWalker) fail(path string, err error) bool {
	w.result.Warnings = append(w.result.Warnings, fmt.Sprintf(skillSourceReadWarningFormat, path, err))

	return false
}

// visit handles one resolved directory's entries.
func (w piPromptWalker) visit(path, resolved string, entries []fs.DirEntry, depth int) bool {
	ok := true

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == nodeModulesDirName {
			continue
		}

		ok = w.entry(filepath.Join(path, name), filepath.Join(resolved, name), depth) && ok
	}

	return ok
}

// piSettingsScope is one settings file (global or project) with the dir its
// relative entries resolve against and the scope stamped on its candidates.
type piSettingsScope struct {
	settings PiSettings
	baseDir  string
	scopeID  string
	home     string
	project  bool
}

// packageRef resolves one package entry to its install root, pkg-id and
// identity (Pi's parseSource / getPackageIdentity / install paths,
// package-manager.js:1144-1166, 1366-1380, 1669-1704).
func (s piSettingsScope) packageRef(entry piPackageEntry) piPackage {
	pkg := piPackage{entry: entry, scopeID: s.scopeID}
	source := strings.TrimSpace(entry.source)

	if spec, isNpm := strings.CutPrefix(source, piNpmPrefix); isNpm {
		name := strings.TrimSpace(spec)
		if match := piNpmSpecPattern.FindStringSubmatch(name); match != nil {
			name = match[1]
		}

		pkg.root = filepath.Join(s.baseDir, piNpmDirName, piNodeModulesPath, name)
		pkg.pkgID, pkg.identity = name, "npm:"+name
	} else if host, repoPath, isGit := parsePiGitSource(source); isGit && !isPiLocalSource(source) {
		pkg.root = filepath.Join(s.baseDir, piGitDirName, host, repoPath)
		pkg.pkgID = host + "/" + repoPath
		pkg.identity = "git:" + pkg.pkgID
	} else {
		pkg.root = resolvePiPath(source, s.baseDir, s.home)
		pkg.pkgID, pkg.identity = tildeRelative(pkg.root, s.home), "local:"+pkg.root
	}

	if !s.project {
		pkg.scopeID = SkillScopePiPkgPrefix + pkg.pkgID
	}

	return pkg
}

// rootForm is the removal-eligibility form of this settings file's entry
// roots (design D5): each entry is its own pi-settings root, or a project
// root for the project settings.
func (s piSettingsScope) rootForm() SkillRootForm {
	if s.project {
		return SkillRootFormProject
	}

	return SkillRootFormPiSettings
}

// rootKeyPrefixes are the key prefixes one of this settings file's
// entries of kind vouches for: `pi-settings:` (or `pi-settings:pi-prompt:`),
// under `project:<r>:` for the project settings (design D5).
func (s piSettingsScope) rootKeyPrefixes(kind SkillSourceKind) []string {
	return []string{skillRootKeyPrefix(s.scopeID, SkillScopePiSettings, kind)}
}

// scanEntries scans one settings `skills` or `prompts` list (see
// ScanPiConfiguredSources).
func (s piSettingsScope) scanEntries(fsys SkillSourceFS, entries []string, kind SkillSourceKind) SkillScanResult {
	var (
		result SkillScanResult
		plain  []string
	)

	patterns := []string{}

	for _, entry := range entries {
		if isPiPattern(entry) {
			patterns = append(patterns, entry)
		} else {
			plain = append(plain, resolvePiPath(entry, s.baseDir, s.home))
		}
	}

	if hasPiGlobstar(patterns) {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf(piGlobstarWarningFormat, "settings "+filepath.Join(s.baseDir, piSettingsFilename)))

		for _, root := range plain {
			result.Roots = append(result.Roots, ScannedRoot{
				Path: root, Form: s.rootForm(), KeyPrefixes: s.rootKeyPrefixes(kind),
			})
		}

		return result
	}

	for _, root := range plain {
		collected := collectPiPath(fsys, root, kind)
		stampPiCandidates(collected.candidates, s.scopeID, SkillScopePiSettings, collected.resolved)

		result.Candidates = append(result.Candidates, collected.candidates...)
		result.Warnings = append(result.Warnings, collected.warnings...)
		result.Roots = append(result.Roots, ScannedRoot{
			Path: root, Resolved: collected.resolved, Scanned: collected.found && collected.ok,
			Form: s.rootForm(), KeyPrefixes: s.rootKeyPrefixes(kind),
		})
	}

	result.Candidates = dedupePiWalked(result.Candidates)
	enabled := applyPiPatterns(result.Candidates, patterns, s.baseDir)

	for index := range result.Candidates {
		result.Candidates[index].Disabled = !enabled[index]
	}

	return result
}

// applyPiPatterns returns, per candidate, whether Pi's applyPatterns
// (package-manager.js:540-584) enables it: includes (all when none), minus
// `!` excludes, plus exact `+` paths, minus exact `-` paths.
func applyPiPatterns(candidates []SkillCandidate, patterns []string, baseDir string) []bool {
	split := splitPiPatterns(patterns)
	enabled := make([]bool, len(candidates))

	for index, candidate := range candidates {
		target := newPiPatternTarget(candidateWalkedPath(candidate), baseDir)

		isEnabled := len(split.includes) == 0 || target.matchesAnyGlob(split.includes)
		if isEnabled && target.matchesAnyGlob(split.excludes) {
			isEnabled = false
		}

		if !isEnabled && target.matchesAnyExact(split.forceIncludes) {
			isEnabled = true
		}

		if isEnabled && target.matchesAnyExact(split.forceExcludes) {
			isEnabled = false
		}

		enabled[index] = isEnabled
	}

	return enabled
}

// candidateWalkedPath is the candidate's discovered path, falling back to
// its resolved source.
func candidateWalkedPath(candidate SkillCandidate) string {
	if candidate.WalkedPath != "" {
		return candidate.WalkedPath
	}

	return candidate.SourcePath
}

// collectPiFile collects one resolved `.md` file entry.
func collectPiFile(fsys SkillSourceFS, root, resolved string, kind SkillSourceKind) piCollected {
	collected := piCollected{resolved: resolved, found: true, ok: true}

	if !strings.HasSuffix(root, markdownFileExt) {
		return collected
	}

	content, err := fsys.ReadFile(resolved)
	if err != nil {
		var result SkillScanResult

		warnUnlessNotExist(&result, root, err)

		collected.ok, collected.warnings = errors.Is(err, fs.ErrNotExist), result.Warnings

		return collected
	}

	name := strings.TrimSuffix(filepath.Base(root), markdownFileExt)
	if kind == SkillSourceKindSkill && filepath.Base(root) == skillMDFilename {
		name = filepath.Base(filepath.Dir(root))
	}

	collected.candidates = []SkillCandidate{{
		Name: name, SourcePath: resolved, WalkedPath: root, Kind: kind, Content: content,
	}}

	return collected
}

// collectPiPath collects one path for kind (Pi's collectFilesFromPaths,
// package-manager.js:1986-2005): a directory is searched (skills:
// ScanPiSkillDir with root `.md` files; prompts: recursive `*.md`); a `.md`
// file is one resource (a skill named by its stem, or by its directory when
// it is SKILL.md); a missing path is not found.
func collectPiPath(fsys SkillSourceFS, root string, kind SkillSourceKind) piCollected {
	resolved, info, err := resolveSkillPathInfo(fsys, root)
	if err != nil {
		var result SkillScanResult

		warnUnlessNotExist(&result, root, err)

		return piCollected{ok: errors.Is(err, fs.ErrNotExist), warnings: result.Warnings}
	}

	if info.IsDir() {
		var scanned SkillScanResult
		if kind == SkillSourceKindSkill {
			scanned = ScanPiSkillDir(fsys, root, "", true)
		} else {
			scanned = scanPiPromptTree(fsys, root)
		}

		ok := len(scanned.Roots) > 0 && scanned.Roots[0].Scanned

		return piCollected{
			candidates: scanned.Candidates, resolved: resolved, found: true, ok: ok, warnings: scanned.Warnings,
		}
	}

	return collectPiFile(fsys, root, resolved, kind)
}

// decodeJSONObjectOrNil decodes raw as a JSON object, or returns nil.
func decodeJSONObjectOrNil(raw json.RawMessage) map[string]json.RawMessage {
	var object map[string]json.RawMessage

	_ = json.Unmarshal(raw, &object)

	return object
}

// decodePiPackageEntry decodes one `packages` element.
func decodePiPackageEntry(raw json.RawMessage) (piPackageEntry, error) {
	var source string
	if json.Unmarshal(raw, &source) == nil {
		return piPackageEntry{source: source}, nil
	}

	var object struct {
		Source   *string          `json:"source"`
		Skills   *json.RawMessage `json:"skills"`
		Prompts  *json.RawMessage `json:"prompts"`
		Autoload *bool            `json:"autoload"`
	}

	err := json.Unmarshal(raw, &object)
	if err != nil || object.Source == nil {
		return piPackageEntry{}, errPiPackageShape
	}

	filter := &piPackageFilter{
		patterns:    map[SkillSourceKind][]string{},
		autoloadOff: object.Autoload != nil && !*object.Autoload,
	}

	for kind, field := range map[SkillSourceKind]*json.RawMessage{
		SkillSourceKindSkill: object.Skills, SkillSourceKindPrompt: object.Prompts,
	} {
		if field == nil || strings.TrimSpace(string(*field)) == jsonNullLiteral {
			continue
		}

		list, listErr := decodePiStringList(*field)
		if listErr != nil {
			return piPackageEntry{}, listErr
		}

		filter.patterns[kind] = list
	}

	return piPackageEntry{source: *object.Source, filter: filter}, nil
}

// decodePiPackages decodes the `packages` list; absent is empty.
func decodePiPackages(raw json.RawMessage) ([]piPackageEntry, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var elements []json.RawMessage

	err := json.Unmarshal(raw, &elements)
	if err != nil {
		return nil, fmt.Errorf("decoding: %w", err)
	}

	entries := make([]piPackageEntry, 0, len(elements))

	for _, element := range elements {
		entry, entryErr := decodePiPackageEntry(element)
		if entryErr != nil {
			return nil, entryErr
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// decodePiSkillsSetting decodes `skills`: an array of strings, or the legacy
// object whose `customDirectories` array replaces it.
func decodePiSkillsSetting(raw json.RawMessage) ([]string, error) {
	var legacy struct {
		CustomDirectories []string `json:"customDirectories"`
	}

	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "{") && json.Unmarshal(raw, &legacy) == nil {
		return legacy.CustomDirectories, nil
	}

	return decodePiStringList(raw)
}

// decodePiStringList decodes an array of strings; absent or null is empty.
func decodePiStringList(raw json.RawMessage) ([]string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == jsonNullLiteral {
		return nil, nil
	}

	var list []string

	err := json.Unmarshal(raw, &list)
	if err != nil {
		return nil, errPiNotStringList
	}

	return list, nil
}

// dedupePiPackages orders project packages before global ones and keeps one
// entry per identity, the project's winning (Pi's dedupePackages,
// package-manager.js:1386-1407). A project `autoload: false` entry is a
// delta over the global one; neither is evaluated, so the global root is
// recorded not scanned alongside it.
func dedupePiPackages(scopes []piSettingsScope) []piPackage {
	var ordered []piPackage

	for _, scope := range slices.Backward(scopes) {
		for _, entry := range scope.settings.packages {
			ordered = append(ordered, scope.packageRef(entry))
		}
	}

	seen := map[string]int{}
	result := make([]piPackage, 0, len(ordered))

	for _, pkg := range ordered {
		winnerIndex, dup := seen[pkg.identity]
		if !dup {
			seen[pkg.identity] = len(result)
			result = append(result, pkg)

			continue
		}

		winner := &result[winnerIndex]
		if winner.entry.filter != nil && winner.entry.filter.autoloadOff && winner.root != pkg.root {
			winner.skipRoots = append(winner.skipRoots, pkg.root)
		}
	}

	return result
}

// dedupePiWalked keeps the first candidate per walked path (Pi's
// addResource keeps the first entry per path).
func dedupePiWalked(candidates []SkillCandidate) []SkillCandidate {
	seen := map[string]bool{}

	return slices.DeleteFunc(candidates, func(candidate SkillCandidate) bool {
		walked := candidateWalkedPath(candidate)
		if seen[walked] {
			return true
		}

		seen[walked] = true

		return false
	})
}

// expandPiGlobSegment extends each match by one path segment.
func expandPiGlobSegment(fsys SkillSourceFS, matches []string, segment string) ([]string, bool, []string) {
	var next []string

	for _, match := range matches {
		if !strings.ContainsAny(segment, "*?[") {
			next = append(next, filepath.Join(match, segment))

			continue
		}

		children, err := matchPiGlobChildren(fsys, match, segment)
		if err != nil {
			return nil, false, []string{fmt.Sprintf(skillSourceReadWarningFormat, match, err)}
		}

		next = append(next, children...)
	}

	return next, true, nil
}

// expandPiManifestEntry resolves one manifest entry against root: a plain
// entry is one path; an entry with `*`/`?` is expanded segment by segment
// with path.Match, skipping `.`-prefixed names (Pi's globSync with
// dot:false, package-manager.js:1875-1889).
func expandPiManifestEntry(fsys SkillSourceFS, root, entry string) ([]string, bool, []string) {
	joined := entry
	if !filepath.IsAbs(entry) {
		joined = filepath.Join(root, entry)
	}

	if !strings.ContainsAny(entry, "*?") {
		return []string{joined}, true, nil
	}

	matches := []string{string(filepath.Separator)}

	for _, segment := range splitSkillPath(joined) {
		if segment == "" {
			continue
		}

		next, ok, warnings := expandPiGlobSegment(fsys, matches, segment)
		if !ok {
			return nil, false, warnings
		}

		matches = next
	}

	return matches, true, nil
}

// hasPiGlobstar reports whether any pattern uses the unsupported `**`.
func hasPiGlobstar(patterns []string) bool {
	return slices.ContainsFunc(patterns, func(pattern string) bool {
		return strings.Contains(pattern, piGlobstar)
	})
}

// isJSONTruthy is JavaScript truthiness for a raw JSON value (absent,
// null, false, 0 and "" are falsy).
func isJSONTruthy(raw json.RawMessage) bool {
	switch strings.TrimSpace(string(raw)) {
	case "", jsonNullLiteral, "false", "0", `""`:
		return false
	default:
		return true
	}
}

// isPiLocalSource is Pi's isLocalPath (dist/utils/paths.js:25-37).
func isPiLocalSource(source string) bool {
	nonLocal := []string{"npm:", "git:", "github:", "http:", "https:", "ssh:"}

	return !slices.ContainsFunc(nonLocal, func(prefix string) bool {
		return strings.HasPrefix(source, prefix)
	})
}

// isPiOverridePattern is Pi's isOverridePattern (package-manager.js:130-132).
func isPiOverridePattern(entry string) bool {
	return strings.HasPrefix(entry, "!") || strings.HasPrefix(entry, "+") || strings.HasPrefix(entry, "-")
}

// isPiPattern is Pi's isPattern (package-manager.js:127-129).
func isPiPattern(entry string) bool {
	return isPiOverridePattern(entry) || strings.ContainsAny(entry, "*?")
}

// matchPiGlobChildren lists dir's non-hidden children whose names match
// segment. A missing dir or a non-directory has none.
func matchPiGlobChildren(fsys SkillSourceFS, dir, segment string) ([]string, error) {
	resolved, info, err := resolveSkillPathInfo(fsys, dir)
	if err != nil {
		return nil, ignoreNotExist(err)
	}

	if !info.IsDir() {
		return nil, nil
	}

	entries, err := fsys.ReadDir(resolved)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	var children []string

	for _, entry := range entries {
		if matched, _ := path.Match(segment, entry.Name()); matched && !strings.HasPrefix(entry.Name(), ".") {
			children = append(children, filepath.Join(dir, entry.Name()))
		}
	}

	return children, nil
}

// newPiPatternTarget builds the match forms of walked relative to baseDir.
func newPiPatternTarget(walked, baseDir string) piPatternTarget {
	rel, err := filepath.Rel(baseDir, walked)
	if err != nil {
		rel = walked
	}

	target := piPatternTarget{
		rel:  filepath.ToSlash(rel),
		name: filepath.Base(walked),
		abs:  filepath.ToSlash(walked),
	}

	if target.name == skillMDFilename {
		parent := filepath.Dir(walked)

		parentRel, relErr := filepath.Rel(baseDir, parent)
		if relErr != nil {
			parentRel = parent
		}

		target.skill = true
		target.parentRel = filepath.ToSlash(parentRel)
		target.parentName = filepath.Base(parent)
		target.pAbs = filepath.ToSlash(parent)
	}

	return target
}

// parsePiGitSource is the subset of Pi's parseGitUrl (dist/utils/git.js)
// engram needs for install paths: a `git:` source in scp
// (`git@host:owner/repo`), URL or `host/owner/repo` shorthand form, or a
// bare http(s)/ssh/git URL. `@ref`/`#ref`, a trailing `.git` and leading
// `/` are stripped; at least two path segments and no `..` are required.
func parsePiGitSource(source string) (host, repoPath string, ok bool) {
	trimmed := strings.TrimSpace(source)
	body, hasPrefix := strings.CutPrefix(trimmed, "git:")
	body = strings.TrimSpace(body)

	if !hasPrefix && !piGitProtocolPattern.MatchString(body) {
		return "", "", false
	}

	body, _, _ = strings.Cut(body, "#")

	host, repoPath, ok = splitPiGitHostPath(body)
	if !ok {
		return "", "", false
	}

	repoPath, _, _ = strings.Cut(repoPath, "@")
	repoPath = strings.TrimLeft(strings.TrimSuffix(repoPath, ".git"), "/")

	segments := strings.Split(repoPath, "/")
	if host == "" || strings.Contains(host, "/") || len(segments) < piMinGitPathSegments ||
		slices.Contains(segments, "..") || slices.Contains(segments, "") {
		return "", "", false
	}

	return host, repoPath, true
}

// piKindDirName is the convention dir of a resource kind.
func piKindDirName(kind SkillSourceKind) string {
	if kind == SkillSourceKindPrompt {
		return piPromptsDirName
	}

	return piSkillsDirName
}

// piManifestValue returns package.json's raw `pi` value; parsed is false
// when the file is absent or does not decode as a JSON object.
func piManifestValue(content []byte, found bool) (json.RawMessage, bool) {
	if !found {
		return nil, false
	}

	var pkg map[string]json.RawMessage
	if json.Unmarshal(content, &pkg) != nil {
		return nil, false
	}

	return pkg[piManifestKey], true
}

// readPiManifest reads package.json's `pi` value. A missing file, or one
// that does not parse, is no manifest (Pi's readPiManifest swallows parse
// errors); any other read failure is returned. A `pi` resource list that is
// not an array of strings is an error.
func readPiManifest(fsys SkillSourceFS, manifestPath string) (piManifest, error) {
	_, content, found, err := readMarkdownFile(fsys, manifestPath)
	if err != nil {
		return piManifest{}, err
	}

	raw, parsed := piManifestValue(content, found)

	manifest := piManifest{present: parsed && isJSONTruthy(raw), entries: map[SkillSourceKind][]string{}}
	if !manifest.present {
		return manifest, nil
	}

	// A non-object `pi` value is truthy but lists nothing.
	object := decodeJSONObjectOrNil(raw)

	for kind, key := range map[SkillSourceKind]string{
		SkillSourceKindSkill: piSkillsDirName, SkillSourceKindPrompt: piPromptsDirName,
	} {
		field, listed := object[key]
		if !listed || strings.TrimSpace(string(field)) == jsonNullLiteral {
			continue
		}

		list, listErr := decodePiStringList(field)
		if listErr != nil {
			return piManifest{}, fmt.Errorf("%s `pi.%s`: %w", manifestPath, key, listErr)
		}

		manifest.entries[kind] = list
	}

	return manifest, nil
}

// resolvePiPath is Pi's resolvePath (dist/utils/paths.js:39-62) with `~`
// expansion: trimmed, `~`/`~/…` expanded against home, `file://` stripped,
// absolute paths cleaned, relative ones joined to baseDir.
func resolvePiPath(input, baseDir, home string) string {
	normalized := strings.TrimSpace(input)

	switch {
	case normalized == "~":
		return filepath.Clean(home)
	case strings.HasPrefix(normalized, piTildePrefix):
		return filepath.Join(home, normalized[len(piTildePrefix):])
	}

	if rest, isFileURL := strings.CutPrefix(normalized, piFileURLPrefix); isFileURL {
		normalized = rest
	}

	if filepath.IsAbs(normalized) {
		return filepath.Clean(normalized)
	}

	return filepath.Join(baseDir, normalized)
}

// scanPiPromptTree scans one prompt directory recursively (piPromptWalker),
// named by stem, with R5 root semantics.
func scanPiPromptTree(fsys SkillSourceFS, promptsRoot string) SkillScanResult {
	var result SkillScanResult

	resolvedRoot, entries, record := readSkillSourceRoot(fsys, promptsRoot, &result)
	if record.Scanned {
		walker := piPromptWalker{
			fsys:      fsys,
			readRoot:  resolvedRoot,
			result:    &result,
			ancestors: map[string]bool{resolvedRoot: true},
		}

		record.Scanned = walker.visit(promptsRoot, resolvedRoot, entries, 0)
	}

	result.Roots = append(result.Roots, record)

	return result
}

// splitPiGitHostPath splits a git source body (ref already cut) into host
// and path: scp form, protocol URL, or `host/path` shorthand whose host has
// a dot (or is localhost).
func splitPiGitHostPath(body string) (host, repoPath string, ok bool) {
	if match := piScpGitPattern.FindStringSubmatch(body); match != nil {
		return match[1], match[2], true
	}

	if _, afterScheme, isURL := strings.Cut(body, piURLSchemeSeparator); isURL {
		authority, rest, _ := strings.Cut(afterScheme, "/")
		if userinfoEnd := strings.LastIndex(authority, "@"); userinfoEnd >= 0 {
			authority = authority[userinfoEnd+1:]
		}

		hostname, _, _ := strings.Cut(authority, ":")

		return strings.ToLower(hostname), rest, true
	}

	host, repoPath, found := strings.Cut(body, "/")
	if !found || (!strings.Contains(host, ".") && host != "localhost") {
		return "", "", false
	}

	return host, repoPath, true
}

// splitPiPatterns splits a pattern list as Pi's applyPatterns does.
func splitPiPatterns(patterns []string) piPatternSplit {
	var split piPatternSplit

	for _, pattern := range patterns {
		switch {
		case strings.HasPrefix(pattern, "+"):
			split.forceIncludes = append(split.forceIncludes, pattern[1:])
		case strings.HasPrefix(pattern, "-"):
			split.forceExcludes = append(split.forceExcludes, pattern[1:])
		case strings.HasPrefix(pattern, "!"):
			split.excludes = append(split.excludes, pattern[1:])
		default:
			split.includes = append(split.includes, pattern)
		}
	}

	return split
}

// stampPiCandidates sets scope, key segment and read root on candidates.
func stampPiCandidates(candidates []SkillCandidate, scopeID, segment, readRoot string) {
	for index := range candidates {
		candidates[index].ScopeID = scopeID
		candidates[index].SourceSegment = segment
		candidates[index].ReadRoot = readRoot
	}
}

// tildeRelative renders path relative to home as `~/…` when it lies under
// home, else unchanged.
func tildeRelative(path, home string) string {
	rel, err := filepath.Rel(home, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return path
	}

	if rel == "." {
		return "~"
	}

	return piTildePrefix + rel
}
