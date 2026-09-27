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
	// SkillScopeClaudeCmd is the scope ID of Claude Code user commands
	// (~/.claude/commands; design D3 `claude-cmd`).
	SkillScopeClaudeCmd = "claude-cmd"
	// SkillScopeClaudeUser is the scope ID of Claude Code user skills
	// (~/.claude/skills; design D3 `claude-user`).
	SkillScopeClaudeUser = "claude-user"
	// SkillScopeSynced is the scope ID of claude.ai-synced skills
	// (~/.claude/skills/synced/<bucket>; design D3 `synced`).
	SkillScopeSynced = "synced"
	// SkillSourceKindCommand, SkillSourceKindPrompt and SkillSourceKindSkill
	// are the three SkillSourceKind values (design D1).
	SkillSourceKindCommand SkillSourceKind = "command"
	SkillSourceKindPrompt  SkillSourceKind = "prompt"
	SkillSourceKindSkill   SkillSourceKind = "skill"
)

// ScannedRoot records one source root a scanner tried to read, and whether
// the read succeeded (design D5). Any read error, including not-exist, means
// not scanned — never "scanned and empty" — so a removal is never offered
// for a note whose source could not be read.
type ScannedRoot struct {
	// Path is the root as the scanner was given it (unresolved).
	Path string
	// Resolved is Path with every symlink resolved; empty when resolution
	// failed (e.g. the root does not exist).
	Resolved string
	// Scanned is true only when the root was read successfully.
	Scanned bool
	// Form is the removal-eligibility form of the notes this root can prove
	// absent (design D5). Scanners leave it empty; ResolveSkillSources and
	// the Pi configured-source scanner stamp it. An empty Form (e.g. the
	// synced directory itself, or a plugin root) never grants eligibility.
	Form SkillRootForm
	// KeyPrefixes are, for a source-rooted root (synced bucket, Pi settings
	// entry, Pi package, project directory), the key prefixes of the notes
	// it can vouch for: `anthropic-skills:`, `pi-settings:` or
	// `pi-settings:pi-prompt:`, `pi-pkg:<id>:` and `pi-pkg:<id>:pi-prompt:`,
	// `project:<r>:` plus its segment (``, `cmd:`, `pi:`, `agents:`, …).
	// A read root vouches for a note only when the note's key is one of its
	// prefixes followed by a name (design D5).
	KeyPrefixes []string
}

// SkillCandidate is one skill, command or prompt file found by a source
// scanner — the shared record every scanner emits and the resolver
// (ResolveSkillSources, task 1.10) composes (design D1).
type SkillCandidate struct {
	// Key is the source-qualified skill key (design D3). Scanners leave it
	// empty; AssignSkillKeys builds it from ScopeID, SourceSegment, Kind and
	// Name (plus the engram-owned-root rule over SourcePath), and
	// ResolveSkillSources stamps it on every candidate it returns.
	Key string
	// Name is the scope-local raw name: a skill's directory name, or a
	// command's relative path with `/` replaced by `:` and `.md` stripped
	// (`opsx/apply.md` → `opsx:apply`). Never a frontmatter field (D2).
	Name string
	// ScopeID is the source scope the scanner was run for (e.g.
	// SkillScopeClaudeUser, SkillScopeSynced, `plugin:<plugin>`).
	ScopeID string
	// ReadRoot is the fully resolved root whose successful read produced
	// this candidate (the skills dir, a synced bucket, a commands dir or a
	// declared command path) — the root D5 removal eligibility checks.
	ReadRoot string
	// SourcePath is the fully symlink-resolved path of the file whose bytes
	// are Content.
	SourcePath string
	// Kind says whether the file is a skill, command or prompt template.
	Kind SkillSourceKind
	// Content is the file's bytes (the note body and hash input).
	Content []byte
	// WalkedPath is the file's path as discovered, before symlink
	// resolution (e.g. `~/.pi/agent/skills/route/SKILL.md` for a symlinked
	// skill). The Pi scanners set it, because Pi matches settings patterns
	// against it (ApplyPiSettingsOverrides); other scanners leave it empty.
	WalkedPath string
	// SourceSegment is the key segment between the scope and the name for
	// sources whose ScopeID does not determine it:
	//   - `pi-settings` or `pi-pkg:<pkg-id>` for Pi configured sources, global
	//     or project (ScanPiConfiguredSources);
	//   - within a `project:<r>` scope, the segment of the matching user
	//     scope (ruling R11): SkillSegmentPi for `.pi/skills`,
	//     SkillSegmentAgents for `.agents/skills`, SkillScopePiPrompt for
	//     `.pi/prompts` — so key construction (task 2.1) builds
	//     `project:<r>:pi:<n>`, `project:<r>:agents:<n>`,
	//     `project:<r>:pi-prompt:<n>`, `project:<r>:pi-settings:<n>` and
	//     `project:<r>:pi-pkg:<id>:<n>` (design D3).
	// Empty for every other source, including Claude project skills and
	// commands (`project:<r>:<n>` / `project:<r>:cmd:<n>`, told apart by
	// Kind).
	SourceSegment string
	// Disabled marks a file Pi finds but has switched off: a settings
	// `!pattern`/`-path` over a default folder (ApplyPiSettingsOverrides,
	// ruling R8), a settings include glob or `!`/`-` filter over a settings
	// entry, or an object-form package filter (including `[]`) (ruling R10).
	// A Disabled candidate MUST NOT be offered for registration or refresh,
	// but it counts as PRESENT for removal eligibility, exactly like a
	// disabled plugin keeping its notes.
	Disabled bool
}

// SkillScanResult is what one source scanner returns: the candidates it
// found, every root it tried to read (with its scanned flag), and warning
// lines for entries it skipped for an unsupported shape or a read error.
type SkillScanResult struct {
	Candidates []SkillCandidate
	Roots      []ScannedRoot
	Warnings   []string
}

// SkillSourceFS is the read-only filesystem capability the source scanners
// need. EdgeFS satisfies it, so production composition passes Deps.FS
// directly. ReadDir and ReadFile are only ever called on fully resolved
// paths; symlinks are resolved through Lstat and Readlink
// (ResolveSkillPath).
type SkillSourceFS interface {
	ReadDir(path string) ([]fs.DirEntry, error)
	ReadFile(path string) ([]byte, error)
	Lstat(path string) (fs.FileInfo, error)
	Readlink(path string) (string, error)
}

// SkillSourceKind says what kind of file a SkillCandidate mirrors.
type SkillSourceKind string

// ResolveSkillPath returns the absolute path with every symlink in it
// resolved (like filepath.EvalSymlinks), using only fsys.Lstat and
// fsys.Readlink so it stays pure over the injected FS. A dangling link yields
// an error satisfying errors.Is(err, fs.ErrNotExist); a link loop yields
// errSkillPathLinkLoop.
func ResolveSkillPath(fsys SkillSourceFS, path string) (string, error) {
	resolved, _, err := resolveSkillPathInfo(fsys, path)

	return resolved, err
}

// ScanClaudeUserSkills scans ~/.claude/skills (design D2 source 1): its
// immediate children that are directories, or symlinks resolving to
// directories, containing SKILL.md. `synced` (the claude.ai buckets, see
// ScanSyncedSkills), root files, dangling symlinks and directories without
// SKILL.md are skipped. Candidates carry scope SkillScopeClaudeUser.
func ScanClaudeUserSkills(fsys SkillSourceFS, skillsRoot string) SkillScanResult {
	return ScanSkillChildren(fsys, skillsRoot, SkillScopeClaudeUser, syncedDirName)
}

// ScanCommandDir scans one commands directory for `**/*.md`, following
// symlinked files and directories (cycle-guarded on resolved paths, depth
// bounded). A command's Name is its path relative to commandsRoot with `/`
// replaced by `:` and `.md` stripped (`opsx/apply.md` → `opsx:apply`, design
// D2 source 2). It serves ~/.claude/commands, project `.claude/commands` and
// plugin `commands/` alike; scopeID is stamped on every candidate. A file or
// namespace directory whose own name contains `:` is skipped with a warning
// (the joined name would be ambiguous); that does not unscan the root.
//
// The root is scanned only when it and every entry beneath it were read:
// any read error other than a dangling link marks it not scanned, with a
// warning, so an unreadable file never looks like a deleted command.
func ScanCommandDir(fsys SkillSourceFS, commandsRoot, scopeID string) SkillScanResult {
	var result SkillScanResult

	resolvedRoot, entries, record := readSkillSourceRoot(fsys, commandsRoot, &result)
	if record.Scanned {
		walker := commandWalker{
			fsys:      fsys,
			scopeID:   scopeID,
			readRoot:  resolvedRoot,
			result:    &result,
			ancestors: map[string]bool{resolvedRoot: true},
		}

		record.Scanned = walker.walk(resolvedRoot, entries, nil)
	}

	result.Roots = append(result.Roots, record)
	sortSkillCandidates(result.Candidates)

	return result
}

// ScanPluginCommands scans a plugin's commands given its `plugin.json`
// `commands` value (raw JSON; plugin.json parsing itself is the plugin
// scanner's job). Absent or null → `<pluginRoot>/commands`. A path string or
// an array of path strings replaces that default: each path (relative to
// pluginRoot) is a commands directory (ScanCommandDir rules) or a single
// `.md` command file named by its stem. Any other form — notably the object
// map — is unsupported: it is skipped with a warning and
// `<pluginRoot>/commands` is recorded as not scanned (design D2 source 8,
// Non-Goals).
//
// A declared path must stay inside the plugin (declaredPluginPath): an
// absolute path, one escaping pluginRoot lexically or through a symlink, or
// one that does not exist is refused with a warning and recorded as a root
// that was not scanned, so the plugin counts as not scanned. A single
// command file that is not `.md` is refused the same way.
func ScanPluginCommands(
	fsys SkillSourceFS, pluginRoot string, commandsField json.RawMessage, scopeID string,
) SkillScanResult {
	defaultDir := filepath.Join(pluginRoot, pluginCommandsDirName)

	paths, useDefault, decodeErr := decodePluginPathsField(commandsField)

	switch {
	case decodeErr != nil:
		return SkillScanResult{
			Roots:    []ScannedRoot{{Path: defaultDir}},
			Warnings: []string{fmt.Sprintf(pluginCommandsUnsupportedWarningFormat, pluginRoot)},
		}
	case useDefault:
		return ScanCommandDir(fsys, defaultDir, scopeID)
	}

	var result SkillScanResult

	for _, declared := range paths {
		path, problem := declaredPluginPath(fsys, pluginRoot, declared)
		if problem != "" {
			result.Roots = append(result.Roots, ScannedRoot{Path: path})
			result.Warnings = append(result.Warnings, fmt.Sprintf(pluginDeclaredPathWarningFormat,
				pluginRoot, pluginCommandsDirName, declared, problem))

			continue
		}

		mergeSkillScanResult(&result, scanCommandPath(fsys, path, scopeID))
	}

	sortSkillCandidates(result.Candidates)

	return result
}

// ScanSkillChildren scans skillsRoot's immediate children for skills: a
// child counts when it is a directory, or a symlink resolving to a
// directory, that contains SKILL.md (itself possibly a symlink). Children
// named in excluded, root files, dangling symlinks and directories without
// SKILL.md are skipped silently. Each candidate records its resolved
// SKILL.md path, the resolved root as ReadRoot, and scopeID.
//
// The root is scanned only when its ReadDir succeeded and no child hit a
// read error other than not-exist; such a child is reported as a warning and
// marks the root not scanned, so it can never look deleted.
func ScanSkillChildren(fsys SkillSourceFS, skillsRoot, scopeID string, excluded ...string) SkillScanResult {
	var result SkillScanResult

	resolvedRoot, entries, record := readSkillSourceRoot(fsys, skillsRoot, &result)

	for _, entry := range entries {
		if slices.Contains(excluded, entry.Name()) {
			continue
		}

		childPath := filepath.Join(skillsRoot, entry.Name())

		sourcePath, content, found, readErr := readSkillChild(fsys, filepath.Join(resolvedRoot, entry.Name()))
		if readErr != nil {
			record.Scanned = false
			result.Warnings = append(result.Warnings, fmt.Sprintf(skillSourceReadWarningFormat, childPath, readErr))

			continue
		}

		if !found {
			continue
		}

		result.Candidates = append(result.Candidates, SkillCandidate{
			Name:       entry.Name(),
			ScopeID:    scopeID,
			ReadRoot:   resolvedRoot,
			SourcePath: sourcePath,
			Kind:       SkillSourceKindSkill,
			Content:    content,
		})
	}

	result.Roots = append(result.Roots, record)
	sortSkillCandidates(result.Candidates)

	return result
}

// ScanSyncedSkills scans ~/.claude/skills/synced (design D2 source 3): each
// directory child ("bucket") whose manifest.json parses as JSON contributes
// its skill children (ScanSkillChildren rules, scope SkillScopeSynced, the
// resolved bucket as ReadRoot). Non-directory children such as the stray
// `.bucket-*` file are ignored. Every bucket directory is recorded as its own
// root — scanned only when its manifest parsed and it was read — so one
// failing bucket never exposes its notes to removal (D5). The synced root
// itself is scanned iff at least one manifest parsed. The manifest's content
// is not otherwise used; skills with one name in two buckets are both
// emitted (collapse or conflict is decided by dedupe, task 2.3).
func ScanSyncedSkills(fsys SkillSourceFS, syncedRoot string) SkillScanResult {
	var (
		result  SkillScanResult
		buckets SkillScanResult
	)

	resolvedRoot, entries, record := readSkillSourceRoot(fsys, syncedRoot, &result)
	anyParsed := false

	for _, entry := range entries {
		bucketPath := filepath.Join(syncedRoot, entry.Name())

		resolvedBucket, info, resolveErr := resolveSkillPathInfo(fsys, filepath.Join(resolvedRoot, entry.Name()))
		if resolveErr != nil {
			if !errors.Is(resolveErr, fs.ErrNotExist) {
				buckets.Roots = append(buckets.Roots, ScannedRoot{Path: bucketPath})
				buckets.Warnings = append(buckets.Warnings,
					fmt.Sprintf(skillSourceReadWarningFormat, bucketPath, resolveErr))
			}

			continue
		}

		if !info.IsDir() {
			continue
		}

		if !syncedManifestParses(fsys, resolvedBucket) {
			buckets.Roots = append(buckets.Roots, ScannedRoot{Path: bucketPath, Resolved: resolvedBucket})

			continue
		}

		anyParsed = true

		mergeSkillScanResult(&buckets, ScanSkillChildren(fsys, bucketPath, SkillScopeSynced))
	}

	if record.Scanned {
		record.Scanned = anyParsed
	}

	result.Roots = append(result.Roots, record)
	mergeSkillScanResult(&result, buckets)
	sortSkillCandidates(result.Candidates)

	return result
}

// unexported constants.
const (
	// commandColonWarningFormat reports a command entry (file or namespace
	// directory) skipped because its name contains `:`, which would make its
	// `:`-joined command name ambiguous (design D2).
	commandColonWarningFormat = "engram: skipping command %s: its name contains `:`"
	commandFileExt            = ".md"
	// jsonNullLiteral is the JSON null token.
	jsonNullLiteral = "null"
	// maxCommandDepth bounds command-directory recursion below the root; a
	// deeper directory is reported and marks the root not scanned.
	maxCommandDepth = 16
	// maxSkillPathLinks bounds symlink hops while resolving one path (the
	// same bound filepath.EvalSymlinks uses).
	maxSkillPathLinks = 255
	// pluginCommandNotMarkdownWarningFormat reports a declared single
	// command file that is not `.md`.
	pluginCommandNotMarkdownWarningFormat  = "engram: plugin command %s is not a .md file; it is not scanned"
	pluginCommandsDirName                  = "commands"
	pluginCommandsUnsupportedWarningFormat = "engram: plugin %s: unsupported plugin.json `commands` form " +
		"(object map or non-path value); its commands are not scanned"
	// pluginDeclaredPathWarningFormat reports a plugin.json `skills` or
	// `commands` path that is refused (declaredPluginPath): the plugin, the
	// field, the declared path and the problem.
	pluginDeclaredPathWarningFormat = "engram: plugin %s: plugin.json `%s` path %s %s; the plugin is not scanned"
	pluginPathAbsoluteProblem       = "is absolute"
	pluginPathEscapesProblem        = "leads outside the plugin directory"
	pluginPathMissingProblem        = "does not exist"
	skillSourceReadWarningFormat    = "engram: cannot read skill source %s: %v; its root is not scanned"
	syncedDirName                   = "synced"
	syncedManifestFilename          = "manifest.json"
)

// unexported variables.
var (
	_                    SkillSourceFS = EdgeFS(nil)
	errCommandDirTooDeep               = errors.New("command directory nesting exceeds the depth limit")
	// errPluginPathsFieldUnsupported reports a plugin.json `skills` or
	// `commands` value that is neither a path string nor an array of path
	// strings.
	errPluginPathsFieldUnsupported = errors.New("unsupported plugin.json path form")
	errSkillPathLinkLoop           = errors.New("too many symlinks")
	errSkillPathNotAbsolute        = errors.New("skill source path is not absolute")
)

// commandWalker carries one ScanCommandDir walk's fixed inputs and output.
type commandWalker struct {
	fsys     SkillSourceFS
	scopeID  string
	readRoot string
	result   *SkillScanResult
	// ancestors holds the resolved directories on the current descent path
	// (the cycle guard).
	ancestors map[string]bool
}

// fail records a warning for path and returns false (not scanned).
func (w commandWalker) fail(path string, err error) bool {
	w.result.Warnings = append(w.result.Warnings, fmt.Sprintf(skillSourceReadWarningFormat, path, err))

	return false
}

// walk visits one resolved directory's entries, appending command candidates
// named by prefix plus the entry path. It returns false when any read
// failed (warned), so the root is not scanned.
func (w commandWalker) walk(dir string, entries []fs.DirEntry, prefix []string) bool {
	ok := true

	for _, entry := range entries {
		childPath := filepath.Join(dir, entry.Name())

		resolved, info, resolveErr := resolveSkillPathInfo(w.fsys, childPath)
		if resolveErr != nil {
			if !errors.Is(resolveErr, fs.ErrNotExist) {
				ok = w.fail(childPath, resolveErr)
			}

			continue
		}

		if !info.IsDir() && !strings.HasSuffix(entry.Name(), commandFileExt) {
			continue
		}

		if strings.Contains(entry.Name(), skillKeySeparator) {
			w.result.Warnings = append(w.result.Warnings, fmt.Sprintf(commandColonWarningFormat, childPath))

			continue
		}

		if info.IsDir() {
			ok = w.walkSubdir(childPath, resolved, append(slices.Clone(prefix), entry.Name())) && ok

			continue
		}

		content, readErr := w.fsys.ReadFile(resolved)
		if readErr != nil {
			ok = w.fail(childPath, readErr)

			continue
		}

		name := strings.Join(append(slices.Clone(prefix), strings.TrimSuffix(entry.Name(), commandFileExt)), ":")

		w.result.Candidates = append(w.result.Candidates, SkillCandidate{
			Name:       name,
			ScopeID:    w.scopeID,
			ReadRoot:   w.readRoot,
			SourcePath: resolved,
			Kind:       SkillSourceKindCommand,
			Content:    content,
		})
	}

	return ok
}

// walkSubdir descends into one resolved subdirectory unless it is already on
// the descent path (a cycle, skipped) or too deep (reported).
func (w commandWalker) walkSubdir(path, resolved string, prefix []string) bool {
	if w.ancestors[resolved] {
		return true
	}

	if len(prefix) > maxCommandDepth {
		return w.fail(path, errCommandDirTooDeep)
	}

	entries, readErr := w.fsys.ReadDir(resolved)
	if readErr != nil {
		return w.fail(path, readErr)
	}

	w.ancestors[resolved] = true
	defer delete(w.ancestors, resolved)

	return w.walk(resolved, entries, prefix)
}

// declaredPluginPath joins one plugin.json `skills` or `commands` path onto
// pluginRoot and checks that it stays inside the plugin, returning the
// joined path and a non-empty problem when it is refused: an absolute path,
// a path leaving pluginRoot lexically (`..`) or, once every symlink is
// resolved, leaving the resolved pluginRoot, or a path that does not exist.
// A resolution failure other than not-exist is left for the scanner to
// report as a read error.
func declaredPluginPath(fsys SkillSourceFS, pluginRoot, declared string) (path, problem string) {
	if filepath.IsAbs(declared) {
		return filepath.Clean(declared), pluginPathAbsoluteProblem
	}

	path = filepath.Join(pluginRoot, declared)
	if !pathWithinRoot(path, pluginRoot) {
		return path, pluginPathEscapesProblem
	}

	resolved, resolveErr := ResolveSkillPath(fsys, path)

	switch {
	case errors.Is(resolveErr, fs.ErrNotExist):
		return path, pluginPathMissingProblem
	case resolveErr != nil:
		return path, ""
	case !pathWithinRoot(resolved, resolveSkillPathOrClean(fsys, pluginRoot)):
		return path, pluginPathEscapesProblem
	default:
		return path, ""
	}
}

// decodePluginPathsField decodes a plugin.json path field (`skills` or
// `commands`): absent or null → absent; a string → one path; an array of
// strings → those paths; anything else → errPluginPathsFieldUnsupported.
func decodePluginPathsField(field json.RawMessage) (paths []string, absent bool, err error) {
	trimmed := strings.TrimSpace(string(field))
	if trimmed == "" || trimmed == jsonNullLiteral {
		return nil, true, nil
	}

	var single string
	if json.Unmarshal(field, &single) == nil {
		return []string{single}, false, nil
	}

	var many []string
	if json.Unmarshal(field, &many) == nil {
		return many, false, nil
	}

	return nil, false, errPluginPathsFieldUnsupported
}

// finishResolvedSkillPath returns resolved with its FileInfo, looking the
// info up when the walk ended without one (the root, or after a `..`).
func finishResolvedSkillPath(
	fsys SkillSourceFS, path, resolved string, info fs.FileInfo,
) (string, fs.FileInfo, error) {
	if info != nil {
		return resolved, info, nil
	}

	rootInfo, _, _, lstatErr := lstatSkillPathComponent(fsys, resolved)
	if lstatErr != nil {
		return "", nil, fmt.Errorf("resolving %s: %w", path, lstatErr)
	}

	return resolved, rootInfo, nil
}

// ignoreNotExist returns nil for a not-exist error and err otherwise.
func ignoreNotExist(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}

// lstatSkillPathComponent Lstats one path component and, when it is a
// symlink, reads its target. A nil FileInfo without an error is treated as
// not-exist.
func lstatSkillPathComponent(
	fsys SkillSourceFS, path string,
) (info fs.FileInfo, target string, isLink bool, err error) {
	info, lstatErr := fsys.Lstat(path)
	if lstatErr != nil {
		return nil, "", false, lstatErr
	}

	if info == nil {
		return nil, "", false, fmt.Errorf("lstat %s: %w", path, fs.ErrNotExist)
	}

	if info.Mode()&fs.ModeSymlink == 0 {
		return info, "", false, nil
	}

	target, readlinkErr := fsys.Readlink(path)
	if readlinkErr != nil {
		return nil, "", false, fmt.Errorf("readlink: %w", readlinkErr)
	}

	return info, target, true, nil
}

// mergeSkillScanResult appends from's candidates, roots and warnings to into.
func mergeSkillScanResult(into *SkillScanResult, from SkillScanResult) {
	into.Candidates = append(into.Candidates, from.Candidates...)
	into.Roots = append(into.Roots, from.Roots...)
	into.Warnings = append(into.Warnings, from.Warnings...)
}

// readSkillChild resolves a skills-dir child and reads its SKILL.md. found is
// false (with a nil error) for a dangling link, a non-directory, or a
// directory without SKILL.md; any other read failure is returned.
func readSkillChild(
	fsys SkillSourceFS, childPath string,
) (sourcePath string, content []byte, found bool, err error) {
	resolvedDir, info, resolveErr := resolveSkillPathInfo(fsys, childPath)
	if resolveErr != nil {
		return "", nil, false, ignoreNotExist(resolveErr)
	}

	if !info.IsDir() {
		return "", nil, false, nil
	}

	return readMarkdownFile(fsys, filepath.Join(resolvedDir, skillMDFilename))
}

// readSkillSourceRoot resolves and lists one source root. The returned record
// is scanned only when both succeeded; a failure other than not-exist is
// also reported as a warning on result.
func readSkillSourceRoot(
	fsys SkillSourceFS, root string, result *SkillScanResult,
) (resolved string, entries []fs.DirEntry, record ScannedRoot) {
	record = ScannedRoot{Path: root}

	resolved, _, resolveErr := resolveSkillPathInfo(fsys, root)
	if resolveErr != nil {
		warnUnlessNotExist(result, root, resolveErr)

		return "", nil, record
	}

	record.Resolved = resolved

	entries, readErr := fsys.ReadDir(resolved)
	if readErr != nil {
		warnUnlessNotExist(result, root, readErr)

		return resolved, nil, record
	}

	record.Scanned = true

	return resolved, entries, record
}

// resolveSkillPathBestEffort resolves path as far as it can: every symlink
// up to the first component that fails is followed, and the unresolved rest
// is joined on. It gives a root that could not be read (a dangling link, a
// denied directory) the real location it points into, so D5 eligibility can
// match a resolved skill_source against it (design D5).
func resolveSkillPathBestEffort(fsys SkillSourceFS, path string) string {
	resolved, rest, _, err := walkSkillPath(fsys, path)
	if err != nil && resolved == "" {
		return path
	}

	return filepath.Join(append([]string{resolved}, rest...)...)
}

// resolveSkillPathInfo is ResolveSkillPath that also returns the resolved
// path's (non-symlink) FileInfo.
func resolveSkillPathInfo(fsys SkillSourceFS, path string) (string, fs.FileInfo, error) {
	resolved, _, info, err := walkSkillPath(fsys, path)
	if err != nil {
		return "", nil, err
	}

	return finishResolvedSkillPath(fsys, path, resolved, info)
}

// scanCommandPath scans one declared plugin commands path: a directory
// (ScanCommandDir rules) or a single `.md` command file named by its stem.
// Any other file is refused with a warning and recorded as not scanned.
func scanCommandPath(fsys SkillSourceFS, path, scopeID string) SkillScanResult {
	var result SkillScanResult

	resolved, info, resolveErr := resolveSkillPathInfo(fsys, path)
	if resolveErr != nil {
		warnUnlessNotExist(&result, path, resolveErr)
		result.Roots = []ScannedRoot{{Path: path}}

		return result
	}

	if info.IsDir() {
		return ScanCommandDir(fsys, path, scopeID)
	}

	if !strings.HasSuffix(filepath.Base(path), commandFileExt) {
		result.Roots = []ScannedRoot{{Path: path, Resolved: resolved}}
		result.Warnings = []string{fmt.Sprintf(pluginCommandNotMarkdownWarningFormat, path)}

		return result
	}

	name := strings.TrimSuffix(filepath.Base(path), commandFileExt)
	if strings.Contains(name, skillKeySeparator) {
		result.Roots = []ScannedRoot{{Path: path, Resolved: resolved, Scanned: true}}
		result.Warnings = []string{fmt.Sprintf(commandColonWarningFormat, path)}

		return result
	}

	content, readErr := fsys.ReadFile(resolved)
	if readErr != nil {
		warnUnlessNotExist(&result, path, readErr)
		result.Roots = []ScannedRoot{{Path: path, Resolved: resolved}}

		return result
	}

	result.Roots = []ScannedRoot{{Path: path, Resolved: resolved, Scanned: true}}
	result.Candidates = []SkillCandidate{{
		Name:       name,
		ScopeID:    scopeID,
		ReadRoot:   resolved,
		SourcePath: resolved,
		Kind:       SkillSourceKindCommand,
		Content:    content,
	}}

	return result
}

// sortSkillCandidates orders candidates by name, then source path, for
// deterministic output.
func sortSkillCandidates(candidates []SkillCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Name != candidates[j].Name {
			return candidates[i].Name < candidates[j].Name
		}

		return candidates[i].SourcePath < candidates[j].SourcePath
	})
}

// splitSkillPath splits a path into its separator-delimited components.
func splitSkillPath(path string) []string {
	return strings.Split(path, string(filepath.Separator))
}

// syncedManifestParses reports whether bucket's manifest.json exists, reads,
// and decodes as JSON.
func syncedManifestParses(fsys SkillSourceFS, bucket string) bool {
	manifestPath, _, resolveErr := resolveSkillPathInfo(fsys, filepath.Join(bucket, syncedManifestFilename))
	if resolveErr != nil {
		return false
	}

	content, readErr := fsys.ReadFile(manifestPath)
	if readErr != nil {
		return false
	}

	var manifest map[string]json.RawMessage

	return json.Unmarshal(content, &manifest) == nil
}

// walkSkillPath resolves path component by component, following symlinks.
// On a failing component it returns the error together with the prefix
// resolved so far and the components still pending (the failing one first);
// resolved is empty for a path that is not absolute or loops.
func walkSkillPath(fsys SkillSourceFS, path string) (string, []string, fs.FileInfo, error) {
	if !filepath.IsAbs(path) {
		return "", nil, nil, fmt.Errorf("%w: %s", errSkillPathNotAbsolute, path)
	}

	resolved := string(filepath.Separator)
	pending := splitSkillPath(path)
	links := 0

	var info fs.FileInfo

	for len(pending) > 0 {
		component := pending[0]

		if component == "" || component == "." {
			pending = pending[1:]

			continue
		}

		if component == ".." {
			resolved, info, pending = filepath.Dir(resolved), nil, pending[1:]

			continue
		}

		next := filepath.Join(resolved, component)

		nextInfo, target, isLink, componentErr := lstatSkillPathComponent(fsys, next)
		if componentErr != nil {
			return resolved, pending, nil, fmt.Errorf("resolving %s: %w", path, componentErr)
		}

		pending = pending[1:]

		if !isLink {
			resolved, info = next, nextInfo

			continue
		}

		links++
		if links > maxSkillPathLinks {
			return "", nil, nil, fmt.Errorf("resolving %s: %w", path, errSkillPathLinkLoop)
		}

		if filepath.IsAbs(target) {
			resolved = string(filepath.Separator)
		}

		pending = append(splitSkillPath(target), pending...)
	}

	return resolved, nil, info, nil
}

// warnUnlessNotExist appends a read warning for path unless err is a plain
// not-exist (an absent source is normal and silent).
func warnUnlessNotExist(result *SkillScanResult, path string, err error) {
	if errors.Is(err, fs.ErrNotExist) {
		return
	}

	result.Warnings = append(result.Warnings, fmt.Sprintf(skillSourceReadWarningFormat, path, err))
}
