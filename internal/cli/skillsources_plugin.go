package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Exported constants.
const (
	// SkillScopePluginPrefix prefixes a Claude Code plugin's scope ID:
	// `plugin:<plugin>` (design D3). Its skills and commands share the
	// scope; a candidate's Kind tells them apart (a command's key gains the
	// `cmd:` segment, task 2.1), so SourceSegment stays empty.
	SkillScopePluginPrefix = "plugin:"
)

// ClaudePluginConflict is one plugin name installed from more than one
// marketplace (design D4, ruling R3). It is returned as data: the caller
// prints the loud line and sets the failure exit (task 2.3), and treats the
// plugin's scope as not scanned (task 2.4). The scanner emits no candidates
// for it.
type ClaudePluginConflict struct {
	// Plugin is the plugin name (the part of the manifest key before `@`).
	Plugin string
	// Marketplaces are the marketplaces it is installed from, sorted.
	Marketplaces []string
}

// ClaudePluginScan is the input to ScanClaudePlugins.
type ClaudePluginScan struct {
	// ClaudeDir is ~/.claude: plugins/installed_plugins.json and
	// settings.json (enabledPlugins) are read from it.
	ClaudeDir string
	// RepoTopLevel is the current repository's top-level directory (`git
	// rev-parse --show-toplevel`, task 1.8), or empty outside a repository.
	// A `project`/`local` install applies only when its projectPath equals it.
	RepoTopLevel string
}

// ClaudePluginScanResult is ScanClaudePlugins' output: the shared scan
// result plus the plugin-level facts removal eligibility needs (design D5
// "Plugin keys").
type ClaudePluginScanResult struct {
	SkillScanResult

	// ManifestsRead is true only when installed_plugins.json and
	// settings.json were both read and parsed.
	ManifestsRead bool
	// Plugins has one entry per plugin name installed in the manifest,
	// sorted by name. A plugin absent from it (with ManifestsRead) is
	// uninstalled.
	Plugins []ClaudePluginStatus
	// PluginConflicts lists plugin names installed from several
	// marketplaces, sorted by name.
	PluginConflicts []ClaudePluginConflict
}

// ClaudePluginStatus says whether one installed plugin was scanned.
type ClaudePluginStatus struct {
	Name string
	// Scanned is true only when the plugin is enabled, applies to this
	// repository, has no conflict or reserved name, and its installPath and
	// every source under it were read. A disabled, out-of-scope, conflicted
	// or unreadable plugin is not scanned, so its notes are never removed.
	Scanned bool
}

// ScanClaudePlugins scans the skills and commands of Claude Code's installed
// plugins (design D2 source 8). Each installed_plugins.json (v2) entry is
// scanned when it is enabled (its settings.json `enabledPlugins` value, or —
// when absent — its plugin.json `defaultEnabled`, default true), its scope
// is `user` or `project`/`local` with projectPath equal to
// scan.RepoTopLevel (the most specific applicable install wins), and its
// installPath is readable. Only installPath is read: `plugins/cache` is
// never globbed, so stale cached versions contribute nothing.
//
// Skills come from `<installPath>/skills/<n>/SKILL.md` plus plugin.json
// `skills` paths (each a directory of skill folders, or one folder holding
// SKILL.md directly, named by its folder). Commands follow
// ScanPluginCommands. Candidates carry ScopeID `plugin:<plugin>` and leave
// Key empty.
//
// A plugin name installed from several marketplaces is a conflict (returned
// in PluginConflicts, never re-keyed); a reserved name, or one containing
// `:`, is skipped with a warning. Neither emits candidates.
func ScanClaudePlugins(fsys SkillSourceFS, scan ClaudePluginScan) ClaudePluginScanResult {
	var result ClaudePluginScanResult

	installs, installsRead, installWarnings := readClaudeInstalledPlugins(
		fsys, filepath.Join(scan.ClaudeDir, claudePluginsDirName, claudeInstalledPluginsFilename),
	)
	result.Warnings = append(result.Warnings, installWarnings...)

	enabled, settingsParsed, settingsUsable, settingsWarnings := readClaudeEnabledPlugins(
		fsys, filepath.Join(scan.ClaudeDir, claudeSettingsFilename),
	)
	result.Warnings = append(result.Warnings, settingsWarnings...)

	result.ManifestsRead = installsRead && settingsParsed

	byName := groupClaudePluginInstalls(installs)
	names := make([]string, 0, len(byName))

	for name := range byName {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		keys, found := byName[name]
		if !found || len(keys) == 0 {
			continue
		}

		status := ClaudePluginStatus{Name: name}

		switch {
		case len(keys) > 1:
			result.PluginConflicts = append(
				result.PluginConflicts,
				claudePluginConflict(name, keys),
			)
		case !settingsUsable:
			// Enablement is unknown: nothing is scanned (warned above).
		case isReservedPluginName(name):
			result.Warnings = append(
				result.Warnings,
				fmt.Sprintf(claudePluginReservedWarningFormat, name),
			)
		default:
			var scanned SkillScanResult

			scanned, status.Scanned = scanClaudePlugin(fsys, scan, name, keys[0], enabled)
			mergeSkillScanResult(&result.SkillScanResult, scanned)
		}

		result.Plugins = append(result.Plugins, status)
	}

	sortSkillCandidates(result.Candidates)

	return result
}

// unexported constants.
const (
	claudeInstalledPluginsFilename           = "installed_plugins.json"
	claudeInstalledPluginsVersion            = 2
	claudeInstalledPluginsVersionErrorFormat = "unsupported version %d (want %d)"
	claudeInstalledPluginsWarningFormat      = "engram: cannot read %s: %v; plugins are not scanned"
	claudePluginEnabledValueWarningFormat    = "engram: plugin %s: `enabledPlugins` value is not a boolean; " +
		"it is not scanned"
	claudePluginKeyMarketplaceSeparator = "@"
	claudePluginManifestDirName         = ".claude-plugin"
	claudePluginManifestFilename        = "plugin.json"
	claudePluginManifestWarningFormat   = "engram: plugin %s: cannot read plugin.json: %v; it is not scanned"
	claudePluginReservedWarningFormat   = "engram: plugin %q uses a reserved name; it is not scanned"
	claudePluginRootSkillWarningFormat  = "engram: plugin %s: plugin.json `skills` names the plugin root " +
		"as one skill; it is not scanned"
	claudePluginSkillsDirName            = "skills"
	claudePluginSkillsFieldWarningFormat = "engram: plugin %s: unsupported plugin.json `skills` value; " +
		"it is not scanned"
	claudePluginsDirName        = "plugins"
	claudeScopeLocal            = "local"
	claudeScopeProject          = "project"
	claudeScopeUser             = "user"
	claudeSettingsFilename      = "settings.json"
	claudeSettingsWarningFormat = "engram: cannot read %s: %v; plugins are not scanned"
)

// unexported variables.
var (
	errClaudeInstalledPluginsVersion = errors.New("unsupported installed_plugins.json version")
	errClaudePluginSkillsField       = errors.New("unsupported plugin.json skills form")
)

// claudePluginInstall is one installed_plugins.json entry for one plugin key.
type claudePluginInstall struct {
	Scope       string `json:"scope"`
	InstallPath string `json:"installPath"`
	ProjectPath string `json:"projectPath"`
}

// claudePluginKey is one `<plugin>@<marketplace>` manifest key with its
// install entries.
type claudePluginKey struct {
	key         string
	marketplace string
	installs    []claudePluginInstall
}

// claudePluginManifest is the part of plugin.json the scanner uses.
type claudePluginManifest struct {
	Skills         json.RawMessage `json:"skills"`
	Commands       json.RawMessage `json:"commands"`
	DefaultEnabled *bool           `json:"defaultEnabled"`
}

// allSkillRootsScanned reports whether every recorded root was scanned.
func allSkillRootsScanned(roots []ScannedRoot) bool {
	for _, root := range roots {
		if !root.Scanned {
			return false
		}
	}

	return true
}

// applicableClaudeInstall returns the most specific install that applies to
// this repository: `user`, or `project`/`local` with projectPath equal to
// repoTopLevel (both symlink-resolved).
func applicableClaudeInstall(
	fsys SkillSourceFS, installs []claudePluginInstall, repoTopLevel string,
) (claudePluginInstall, bool) {
	// Most specific first: local overrides project overrides user.
	for _, scope := range []string{claudeScopeLocal, claudeScopeProject, claudeScopeUser} {
		for _, install := range installs {
			if install.Scope != scope {
				continue
			}

			if scope == claudeScopeUser {
				return install, true
			}

			if repoTopLevel != "" && install.ProjectPath != "" &&
				resolveSkillPathOrClean(
					fsys,
					install.ProjectPath,
				) == resolveSkillPathOrClean(
					fsys,
					repoTopLevel,
				) {
				return install, true
			}
		}
	}

	return claudePluginInstall{}, false
}

// claudePluginConflict builds the conflict record for name's keys.
func claudePluginConflict(name string, keys []claudePluginKey) ClaudePluginConflict {
	markets := make([]string, 0, len(keys))
	for _, key := range keys {
		markets = append(markets, key.marketplace)
	}

	sort.Strings(markets)

	return ClaudePluginConflict{Plugin: name, Marketplaces: markets}
}

// claudePluginEnabledSetting reads key's `enabledPlugins` value: has is
// false when absent; valid is false (with a warning) for a non-boolean.
func claudePluginEnabledSetting(
	enabled map[string]json.RawMessage, key string, result *SkillScanResult,
) (value, has, valid bool) {
	setting, has := enabled[key]
	if !has {
		return false, false, true
	}

	if json.Unmarshal(setting, &value) != nil {
		result.Warnings = append(
			result.Warnings,
			fmt.Sprintf(claudePluginEnabledValueWarningFormat, key),
		)

		return false, true, false
	}

	return value, true, true
}

// claudePluginInstallToScan applies the enablement setting and the scope
// rule: it returns the install to scan, whether an `enabledPlugins` value
// was set, and false when the plugin is disabled or does not apply here.
func claudePluginInstallToScan(
	fsys SkillSourceFS, scan ClaudePluginScan, key claudePluginKey,
	enabled map[string]json.RawMessage, result *SkillScanResult,
) (install claudePluginInstall, hasSetting, ok bool) {
	enabledValue, hasSetting, valid := claudePluginEnabledSetting(enabled, key.key, result)
	if !valid || (hasSetting && !enabledValue) {
		return claudePluginInstall{}, hasSetting, false
	}

	install, applies := applicableClaudeInstall(fsys, key.installs, scan.RepoTopLevel)

	return install, hasSetting, applies && install.InstallPath != ""
}

// decodeClaudePluginSkillsField decodes plugin.json's `skills` value: absent
// or null → none; a string → one path; an array of strings → those paths.
func decodeClaudePluginSkillsField(field json.RawMessage) ([]string, error) {
	trimmed := strings.TrimSpace(string(field))
	if trimmed == "" || trimmed == jsonNullLiteral {
		return nil, nil
	}

	var single string
	if json.Unmarshal(field, &single) == nil {
		return []string{single}, nil
	}

	var many []string
	if json.Unmarshal(field, &many) == nil {
		return many, nil
	}

	return nil, errClaudePluginSkillsField
}

// dedupeSkillCandidatesBySource keeps the first candidate per kind, name
// and resolved source (a plugin.json `skills` path may repeat `./skills`).
func dedupeSkillCandidatesBySource(candidates []SkillCandidate) []SkillCandidate {
	seen := map[string]bool{}

	return slices.DeleteFunc(candidates, func(candidate SkillCandidate) bool {
		identity := string(candidate.Kind) + "\x00" + candidate.Name + "\x00" + candidate.SourcePath
		if seen[identity] {
			return true
		}

		seen[identity] = true

		return false
	})
}

// groupClaudePluginInstalls groups manifest keys by plugin name.
func groupClaudePluginInstalls(
	installs map[string][]claudePluginInstall,
) map[string][]claudePluginKey {
	byName := map[string][]claudePluginKey{}

	for key, entries := range installs {
		name, marketplace, _ := strings.Cut(key, claudePluginKeyMarketplaceSeparator)
		byName[name] = append(
			byName[name],
			claudePluginKey{key: key, marketplace: marketplace, installs: entries},
		)
	}

	return byName
}

// isReservedPluginName reports whether a plugin name would collide with a
// fixed key prefix (design D3) or cannot be a key segment (contains `:`).
func isReservedPluginName(name string) bool {
	switch name {
	case SkillSegmentPi, SkillSegmentAgents, strings.TrimSuffix(SkillScopeProjectPrefix, skillKeySeparator),
		skillKeySegmentAnthropic, skillKeySegmentCommand, SkillScopePiSettings,
		strings.TrimSuffix(SkillScopePiPkgPrefix, skillKeySeparator), SkillScopePiPrompt:
		return true
	default:
		return strings.Contains(name, skillKeySeparator)
	}
}

// readClaudeEnabledPlugins reads settings.json's `enabledPlugins`. parsed is
// true only when the file was read and parsed (design D5); usable is false
// when the file exists but cannot be read or parsed, since enablement is
// then unknown. A missing file is usable and empty (every plugin falls back
// to defaultEnabled, as Claude Code does).
func readClaudeEnabledPlugins(
	fsys SkillSourceFS, settingsPath string,
) (enabled map[string]json.RawMessage, parsed, usable bool, warnings []string) {
	_, content, found, err := readMarkdownFile(fsys, settingsPath)
	if err != nil {
		return nil, false, false, []string{
			fmt.Sprintf(claudeSettingsWarningFormat, settingsPath, err),
		}
	}

	if !found {
		return map[string]json.RawMessage{}, false, true, nil
	}

	var settings struct {
		EnabledPlugins map[string]json.RawMessage `json:"enabledPlugins"`
	}

	decodeErr := json.Unmarshal(content, &settings)
	if decodeErr != nil {
		return nil, false, false, []string{
			fmt.Sprintf(claudeSettingsWarningFormat, settingsPath, decodeErr),
		}
	}

	if settings.EnabledPlugins == nil {
		settings.EnabledPlugins = map[string]json.RawMessage{}
	}

	return settings.EnabledPlugins, true, true, nil
}

// readClaudeInstalledPlugins reads installed_plugins.json (v2:
// `{"version":2,"plugins":{"<plugin>@<marketplace>":[{scope,installPath,
// projectPath?},…]}}`). A missing file is silently not read; any other
// failure is a warning.
func readClaudeInstalledPlugins(
	fsys SkillSourceFS, path string,
) (installs map[string][]claudePluginInstall, read bool, warnings []string) {
	_, content, found, err := readMarkdownFile(fsys, path)
	if err == nil && !found {
		return nil, false, nil
	}

	if err == nil {
		var manifest struct {
			Version int                              `json:"version"`
			Plugins map[string][]claudePluginInstall `json:"plugins"`
		}

		err = json.Unmarshal(content, &manifest)
		if err == nil && manifest.Version != claudeInstalledPluginsVersion {
			err = fmt.Errorf("%w: "+claudeInstalledPluginsVersionErrorFormat,
				errClaudeInstalledPluginsVersion, manifest.Version, claudeInstalledPluginsVersion)
		}

		if err == nil {
			return manifest.Plugins, true, nil
		}
	}

	return nil, false, []string{fmt.Sprintf(claudeInstalledPluginsWarningFormat, path, err)}
}

// readClaudePluginManifest reads `<root>/.claude-plugin/plugin.json`. The
// manifest is optional: a missing file is an empty manifest.
func readClaudePluginManifest(fsys SkillSourceFS, root string) (claudePluginManifest, error) {
	var manifest claudePluginManifest

	_, content, found, err := readMarkdownFile(
		fsys, filepath.Join(root, claudePluginManifestDirName, claudePluginManifestFilename),
	)
	if err != nil || !found {
		return manifest, err
	}

	decodeErr := json.Unmarshal(content, &manifest)
	if decodeErr != nil {
		return manifest, fmt.Errorf("decoding: %w", decodeErr)
	}

	return manifest, nil
}

// scanClaudePlugin scans one uncontested plugin, returning its candidates
// and roots and whether it counts as scanned.
func scanClaudePlugin(
	fsys SkillSourceFS, scan ClaudePluginScan, name string, key claudePluginKey,
	enabled map[string]json.RawMessage,
) (SkillScanResult, bool) {
	var result SkillScanResult

	install, hasSetting, ok := claudePluginInstallToScan(fsys, scan, key, enabled, &result)
	if !ok {
		return result, false
	}

	resolvedRoot, entries, record := readSkillSourceRoot(fsys, install.InstallPath, &result)
	result.Roots = append(result.Roots, record)

	if !record.Scanned {
		return result, false
	}

	manifest, manifestErr := readClaudePluginManifest(fsys, resolvedRoot)
	if manifestErr != nil {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf(claudePluginManifestWarningFormat, key.key, manifestErr))

		return result, false
	}

	if !hasSetting && manifest.DefaultEnabled != nil && !*manifest.DefaultEnabled {
		return result, false
	}

	ok = scanClaudePluginContents(fsys, &result, resolvedRoot, entries, manifest,
		SkillScopePluginPrefix+name, key.key)

	return result, ok && allSkillRootsScanned(result.Roots)
}

// scanClaudePluginContents scans an enabled plugin's skills and commands
// into result. entries is the installPath listing, which says whether the
// default skills/ and commands/ directories exist.
func scanClaudePluginContents(
	fsys SkillSourceFS, result *SkillScanResult, root string, entries []fs.DirEntry,
	manifest claudePluginManifest, scopeID, pluginKey string,
) bool {
	hasEntry := func(entryName string) bool {
		return slices.ContainsFunc(entries, func(entry fs.DirEntry) bool { return entry.Name() == entryName })
	}

	ok := scanClaudePluginSkills(fsys, result, root, manifest.Skills, scopeID, pluginKey,
		hasEntry(claudePluginSkillsDirName))

	if len(manifest.Commands) > 0 || hasEntry(pluginCommandsDirName) {
		mergeSkillScanResult(result, ScanPluginCommands(fsys, root, manifest.Commands, scopeID))
	}

	result.Candidates = dedupeSkillCandidatesBySource(result.Candidates)

	return ok
}

// scanClaudePluginSkills scans the default skills/ directory (when the
// installPath listing holds one) plus each plugin.json `skills` path into
// result. It returns false when a declared path cannot be used.
func scanClaudePluginSkills(
	fsys SkillSourceFS, result *SkillScanResult, root string, field json.RawMessage,
	scopeID, pluginKey string, hasDefault bool,
) bool {
	if hasDefault {
		mergeSkillScanResult(result, ScanSkillChildren(fsys, filepath.Join(root, claudePluginSkillsDirName), scopeID))
	}

	paths, err := decodeClaudePluginSkillsField(field)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf(claudePluginSkillsFieldWarningFormat, pluginKey))

		return false
	}

	ok := true

	for _, declared := range paths {
		ok = scanDeclaredPluginSkillPath(fsys, result, root, filepath.Join(root, declared), scopeID, pluginKey) && ok
	}

	return ok
}

// scanDeclaredPluginSkillPath scans one plugin.json `skills` path: a folder
// holding SKILL.md directly is one skill named by the folder; any other
// directory is scanned for skill folders (ScanSkillChildren rules). The
// plugin root itself as a single skill has no folder name to key on: it is
// reported and marks the plugin not scanned.
func scanDeclaredPluginSkillPath(
	fsys SkillSourceFS, result *SkillScanResult, root, path, scopeID, pluginKey string,
) bool {
	sourcePath, content, holdsSkill, readErr := readSkillChild(fsys, path)

	switch {
	case readErr != nil:
		result.Roots = append(result.Roots, ScannedRoot{Path: path})
		result.Warnings = append(result.Warnings, fmt.Sprintf(skillSourceReadWarningFormat, path, readErr))

		return false
	case !holdsSkill:
		mergeSkillScanResult(result, ScanSkillChildren(fsys, path, scopeID))

		return true
	case filepath.Clean(path) == filepath.Clean(root):
		result.Warnings = append(result.Warnings, fmt.Sprintf(claudePluginRootSkillWarningFormat, pluginKey))

		return false
	}

	resolvedDir := filepath.Dir(sourcePath)
	result.Roots = append(result.Roots, ScannedRoot{Path: path, Resolved: resolvedDir, Scanned: true})
	result.Candidates = append(result.Candidates, SkillCandidate{
		Name:       filepath.Base(filepath.Clean(path)),
		ScopeID:    scopeID,
		ReadRoot:   resolvedDir,
		SourcePath: sourcePath,
		Kind:       SkillSourceKindSkill,
		Content:    content,
	})

	return true
}
