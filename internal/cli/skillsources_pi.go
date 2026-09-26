package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// Exported constants.
const (
	// SkillScopeAgentsUser is the scope ID of ~/.agents/skills (design D3
	// `agents-user`).
	SkillScopeAgentsUser = "agents-user"
	// SkillScopePiPrompt is the scope ID of Pi's global prompt templates,
	// ~/.pi/agent/prompts (design D3 `pi-prompt`).
	SkillScopePiPrompt = "pi-prompt"
	// SkillScopePiUser is the scope ID of Pi's user skills,
	// ~/.pi/agent/skills (design D3 `pi-user`).
	SkillScopePiUser = "pi-user"
)

// PiProjectScan is the input shared by the Pi project scanners (design D2
// source 10). The project identity probe (task 1.8) supplies TopLevel and
// ScopeID; ResolvePiProjectTrust (task 1.6) supplies Trusted. When Trusted
// is false the scanners read nothing and record no root.
type PiProjectScan struct {
	// Cwd is the working directory registration runs from.
	Cwd string
	// TopLevel bounds the `.agents/skills` ancestor chain (inclusive): the
	// repository top-level. Empty means no bound: the chain runs to the
	// filesystem root, as Pi does outside a git repository.
	TopLevel string
	// UserAgentsRoot is ~/.agents/skills. It is the user source, never a
	// project level, even when the chain reaches $HOME (Pi's
	// collectAncestorAgentsSkillDirs filter).
	UserAgentsRoot string
	// ScopeID is stamped on every candidate (`project:<r>`).
	ScopeID string
	// Trusted is Pi's project-trust decision for Cwd.
	Trusted bool
}

// ResolvePiProjectTrust decides whether Pi trusts the project at cwd
// (design D3 trust test; Pi's resolveProjectTrusted in
// dist/core/project-trust.js). cwd is canonicalized (symlinks resolved), then
// the nearest saved decision in <agentDir>/trust.json for cwd or any parent
// wins; a null entry is no decision. With no decision, the project is
// trusted only when <agentDir>/settings.json has `"defaultProjectTrust":
// "always"`; Pi's "ask" cannot prompt here and "never" declines.
//
// A missing trust.json or settings.json is simply empty. Any other read or
// parse failure (Pi refuses such a trust store) yields untrusted plus a
// warning, so an unreadable trust state never exposes project sources.
func ResolvePiProjectTrust(fsys SkillSourceFS, agentDir, cwd string) (trusted bool, warnings []string) {
	canonicalCwd := resolveSkillPathOrClean(fsys, cwd)

	trustPath := filepath.Join(agentDir, piTrustFilename)

	decisions, trustErr := readPiTrustStore(fsys, trustPath)
	if trustErr != nil {
		return false, []string{fmt.Sprintf(piTrustUnreadableWarningFormat, trustPath, trustErr)}
	}

	if decision, found := nearestPiTrustDecision(decisions, canonicalCwd); found {
		return decision, nil
	}

	settingsPath := filepath.Join(agentDir, piSettingsFilename)

	defaultTrust, settingsErr := readPiDefaultProjectTrust(fsys, settingsPath)
	if settingsErr != nil {
		return false, []string{fmt.Sprintf(piTrustUnreadableWarningFormat, settingsPath, settingsErr)}
	}

	return defaultTrust == piDefaultTrustAlways, nil
}

// ScanAgentsProjectSkills scans `.agents/skills` in scan.Cwd and each of its
// ancestors up to scan.TopLevel (design D2 source 10; Pi's
// collectAncestorAgentsSkillDirs), with ScanPiSkillDir rules and no root
// `.md` files. The chain walks the symlink-resolved cwd, so it meets the
// (resolved) top-level. Each existing level is recorded as its own root; a
// missing level is absent, not failed. The same name at two levels is
// emitted twice (a key conflict, task 2.3). Untrusted → nothing is read.
func ScanAgentsProjectSkills(fsys SkillSourceFS, scan PiProjectScan) SkillScanResult {
	var result SkillScanResult

	if !scan.Trusted {
		return result
	}

	userRoot := resolveSkillPathOrClean(fsys, scan.UserAgentsRoot)
	topLevel := resolveSkillPathOrClean(fsys, scan.TopLevel)

	for dir := resolveSkillPathOrClean(fsys, scan.Cwd); ; dir = filepath.Dir(dir) {
		skillsDir := filepath.Join(dir, agentsDirName, piSkillsDirName)

		resolved, _, resolveErr := resolveSkillPathInfo(fsys, skillsDir)

		absent := errors.Is(resolveErr, fs.ErrNotExist)
		if !absent && resolved != userRoot && skillsDir != userRoot {
			mergeSkillScanResult(&result, ScanPiSkillDir(fsys, skillsDir, scan.ScopeID, false))
		}

		if dir == topLevel || filepath.Dir(dir) == dir {
			break
		}
	}

	sortSkillCandidates(result.Candidates)

	return result
}

// ScanAgentsUserSkills scans ~/.agents/skills (design D2 source 5):
// ScanPiSkillDir rules with root `.md` files ignored, scope
// SkillScopeAgentsUser.
func ScanAgentsUserSkills(fsys SkillSourceFS, agentsSkillsRoot string) SkillScanResult {
	return ScanPiSkillDir(fsys, agentsSkillsRoot, SkillScopeAgentsUser, false)
}

// ScanPiProjectPrompts scans `.pi/prompts/*.md` in scan.Cwd (design D2
// source 10; ScanPiPromptDir rules). Untrusted → nothing is read and no root
// is recorded.
func ScanPiProjectPrompts(fsys SkillSourceFS, scan PiProjectScan) SkillScanResult {
	if !scan.Trusted {
		return SkillScanResult{}
	}

	return ScanPiPromptDir(fsys, filepath.Join(scan.Cwd, piConfigDirName, piPromptsDirName), scan.ScopeID)
}

// ScanPiProjectSkills scans `.pi/skills` in scan.Cwd only, never its
// ancestors (design D2 source 10; Pi's project baseDir is `<cwd>/.pi`):
// ScanPiSkillDir rules with root `.md` files. Untrusted → nothing is read
// and no root is recorded.
func ScanPiProjectSkills(fsys SkillSourceFS, scan PiProjectScan) SkillScanResult {
	if !scan.Trusted {
		return SkillScanResult{}
	}

	return ScanPiSkillDir(fsys, filepath.Join(scan.Cwd, piConfigDirName, piSkillsDirName), scan.ScopeID, true)
}

// ScanPiPromptDir scans one Pi prompt-template directory (Pi's
// collectAutoPromptEntries, docs/prompt-templates.md): its immediate `*.md`
// files, following symlinks, each named by its file stem, Kind
// SkillSourceKindPrompt. It is flat: subdirectories are never descended.
// Hidden (`.`-prefixed) entries and dangling links are skipped. Like the
// other scanners, an entry that exists but cannot be read is a warning and
// marks the root not scanned (ruling R5).
func ScanPiPromptDir(fsys SkillSourceFS, promptsRoot, scopeID string) SkillScanResult {
	var result SkillScanResult

	resolvedRoot, entries, record := readSkillSourceRoot(fsys, promptsRoot, &result)

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, markdownFileExt) {
			continue
		}

		entryPath := filepath.Join(promptsRoot, name)

		sourcePath, content, found, readErr := readMarkdownFile(fsys, filepath.Join(resolvedRoot, name))
		if readErr != nil {
			record.Scanned = false
			result.Warnings = append(result.Warnings, fmt.Sprintf(skillSourceReadWarningFormat, entryPath, readErr))

			continue
		}

		if !found {
			continue
		}

		result.Candidates = append(result.Candidates, SkillCandidate{
			Name:       strings.TrimSuffix(name, markdownFileExt),
			ScopeID:    scopeID,
			ReadRoot:   resolvedRoot,
			SourcePath: sourcePath,
			WalkedPath: entryPath,
			Kind:       SkillSourceKindPrompt,
			Content:    content,
		})
	}

	result.Roots = append(result.Roots, record)
	sortSkillCandidates(result.Candidates)

	return result
}

// ScanPiSkillDir scans one Pi skill location with Pi's recursive discovery
// (collectSkillEntries in dist/core/package-manager.js; docs/skills.md):
//
//   - A directory holding a SKILL.md file is one skill, named by its
//     directory name (the name as walked, so a symlinked skill keeps its link
//     name), and descent stops there. That includes skillsRoot itself.
//   - Otherwise every subdirectory is searched, following symlinks, skipping
//     `.`-prefixed entries, `node_modules` and dangling links. Descent is
//     depth-bounded and cycle-guarded on resolved paths.
//   - When includeRootFiles is set (Pi's `~/.pi/agent/skills` and
//     `.pi/skills`), skillsRoot's own `*.md` files are skills named by stem.
//
// Names never come from frontmatter (design D2). The root is scanned only
// when it and everything read beneath it were read: any read error other
// than a dangling link, or a too-deep directory, is a warning and marks the
// root not scanned (ruling R5).
func ScanPiSkillDir(fsys SkillSourceFS, skillsRoot, scopeID string, includeRootFiles bool) SkillScanResult {
	var result SkillScanResult

	resolvedRoot, entries, record := readSkillSourceRoot(fsys, skillsRoot, &result)
	if record.Scanned {
		walker := piSkillWalker{
			fsys:      fsys,
			scopeID:   scopeID,
			readRoot:  resolvedRoot,
			result:    &result,
			ancestors: map[string]bool{resolvedRoot: true},
		}

		record.Scanned = walker.visit(skillsRoot, resolvedRoot, entries, 0, includeRootFiles)
	}

	result.Roots = append(result.Roots, record)
	sortSkillCandidates(result.Candidates)

	return result
}

// ScanPiUserPrompts scans ~/.pi/agent/prompts (design D2 source 6b):
// ScanPiPromptDir rules, scope SkillScopePiPrompt.
func ScanPiUserPrompts(fsys SkillSourceFS, promptsRoot string) SkillScanResult {
	return ScanPiPromptDir(fsys, promptsRoot, SkillScopePiPrompt)
}

// ScanPiUserSkills scans ~/.pi/agent/skills (design D2 source 4):
// ScanPiSkillDir rules with root `.md` files, scope SkillScopePiUser.
func ScanPiUserSkills(fsys SkillSourceFS, piSkillsRoot string) SkillScanResult {
	return ScanPiSkillDir(fsys, piSkillsRoot, SkillScopePiUser, true)
}

// unexported constants.
const (
	agentsDirName   = ".agents"
	markdownFileExt = commandFileExt
	// maxPiSkillDepth bounds Pi skill-directory recursion below the root; a
	// deeper directory is reported and marks the root not scanned.
	maxPiSkillDepth                = 16
	nodeModulesDirName             = "node_modules"
	piConfigDirName                = ".pi"
	piDefaultProjectTrustKey       = "defaultProjectTrust"
	piDefaultTrustAlways           = "always"
	piPromptsDirName               = "prompts"
	piSettingsFilename             = "settings.json"
	piSkillsDirName                = "skills"
	piTrustFilename                = "trust.json"
	piTrustUnreadableWarningFormat = "engram: cannot use Pi trust state %s: %v; Pi project sources are not scanned"
)

// unexported variables.
var (
	errPiJSONNotObject   = errors.New("expected a JSON object")
	errPiSkillDirTooDeep = errors.New("pi skill directory nesting exceeds the depth limit")
	errPiTrustValue      = errors.New("trust decision must be true, false, or null")
)

// piSkillWalker carries one ScanPiSkillDir walk's fixed inputs and output.
type piSkillWalker struct {
	fsys     SkillSourceFS
	scopeID  string
	readRoot string
	result   *SkillScanResult
	// ancestors holds the resolved directories on the current descent path
	// (the cycle guard).
	ancestors map[string]bool
}

// add appends one skill candidate; walked is its file's discovered path.
func (w piSkillWalker) add(name, walked, sourcePath string, content []byte) {
	w.result.Candidates = append(w.result.Candidates, SkillCandidate{
		Name:       name,
		ScopeID:    w.scopeID,
		ReadRoot:   w.readRoot,
		SourcePath: sourcePath,
		WalkedPath: walked,
		Kind:       SkillSourceKindSkill,
		Content:    content,
	})
}

// descend visits one resolved subdirectory unless it is already on the
// descent path (a cycle, skipped) or too deep (reported).
func (w piSkillWalker) descend(path, resolved string, depth int) bool {
	if w.ancestors[resolved] {
		return true
	}

	if depth > maxPiSkillDepth {
		return w.fail(path, errPiSkillDirTooDeep)
	}

	entries, readErr := w.fsys.ReadDir(resolved)
	if readErr != nil {
		return w.fail(path, readErr)
	}

	w.ancestors[resolved] = true
	defer delete(w.ancestors, resolved)

	return w.visit(path, resolved, entries, depth, false)
}

// fail records a warning for path and returns false (not scanned).
func (w piSkillWalker) fail(path string, err error) bool {
	w.result.Warnings = append(w.result.Warnings, fmt.Sprintf(skillSourceReadWarningFormat, path, err))

	return false
}

// rootFile adds one root `.md` file as a skill named by its stem. It
// returns false when the file exists but could not be read (warned).
func (w piSkillWalker) rootFile(path, resolved string) bool {
	content, readErr := w.fsys.ReadFile(resolved)
	if readErr != nil {
		if errors.Is(readErr, fs.ErrNotExist) {
			return true
		}

		return w.fail(path, readErr)
	}

	w.add(strings.TrimSuffix(filepath.Base(path), markdownFileExt), path, resolved, content)

	return true
}

// skillFile looks for a SKILL.md file among one directory's entries. held is
// true when the directory is a skill; ok is false when SKILL.md exists but
// could not be read (warned).
func (w piSkillWalker) skillFile(path, resolved string, entries []fs.DirEntry) (held, ok bool) {
	for _, entry := range entries {
		if entry.Name() != skillMDFilename {
			continue
		}

		sourcePath, content, found, readErr := readMarkdownFile(w.fsys, filepath.Join(resolved, skillMDFilename))
		if readErr != nil {
			return false, w.fail(filepath.Join(path, skillMDFilename), readErr)
		}

		if found {
			w.add(filepath.Base(path), filepath.Join(path, skillMDFilename), sourcePath, content)

			return true, true
		}
	}

	return false, true
}

// visit handles one resolved directory (walked as path): a skill when it
// holds SKILL.md, otherwise its subdirectories and — when includeRootFiles —
// its `.md` files. It returns false when any read failed (warned).
func (w piSkillWalker) visit(
	path, resolved string, entries []fs.DirEntry, depth int, includeRootFiles bool,
) bool {
	held, ok := w.skillFile(path, resolved, entries)
	if held || !ok {
		return ok
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == nodeModulesDirName {
			continue
		}

		ok = w.visitChild(filepath.Join(path, name), filepath.Join(resolved, name), depth, includeRootFiles) && ok
	}

	return ok
}

// visitChild handles one directory entry (walked as path, located at
// entryPath under the resolved parent): a subdirectory is descended; a `.md`
// file is a skill when includeRootFiles; a dangling link is skipped. It
// returns false when a read failed (warned).
func (w piSkillWalker) visitChild(path, entryPath string, depth int, includeRootFiles bool) bool {
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

	if includeRootFiles && strings.HasSuffix(path, markdownFileExt) {
		return w.rootFile(path, resolved)
	}

	return true
}

// decodePiJSONObject decodes content as a JSON object (Pi rejects a trust
// store that is not one; `null` is not an object).
func decodePiJSONObject(content []byte) (map[string]json.RawMessage, error) {
	if bytes.Equal(bytes.TrimSpace(content), []byte(jsonNullLiteral)) {
		return nil, errPiJSONNotObject
	}

	var object map[string]json.RawMessage

	err := json.Unmarshal(content, &object)
	if err != nil {
		return nil, fmt.Errorf("decoding: %w", err)
	}

	return object, nil
}

// nearestPiTrustDecision walks from cwd up to the filesystem root and
// returns the first saved decision (Pi's findNearestTrustEntry: exact
// canonical-path keys; a null entry is no decision).
func nearestPiTrustDecision(decisions map[string]*bool, cwd string) (decision, found bool) {
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if value := decisions[dir]; value != nil {
			return *value, true
		}

		if filepath.Dir(dir) == dir {
			return false, false
		}
	}
}

// readMarkdownFile resolves one candidate file and reads it. found is false
// (with a nil error) for a dangling link, a non-file or a vanished file; any
// other failure is returned.
func readMarkdownFile(fsys SkillSourceFS, path string) (sourcePath string, content []byte, found bool, err error) {
	resolved, info, resolveErr := resolveSkillPathInfo(fsys, path)
	if resolveErr != nil {
		return "", nil, false, ignoreNotExist(resolveErr)
	}

	if info.IsDir() {
		return "", nil, false, nil
	}

	content, readErr := fsys.ReadFile(resolved)
	if readErr != nil {
		return "", nil, false, ignoreNotExist(readErr)
	}

	return resolved, content, true, nil
}

// readPiDefaultProjectTrust reads settings.json's `defaultProjectTrust`
// string ("" when the file or key is absent, or the value is not a string —
// Pi then falls back to "ask").
func readPiDefaultProjectTrust(fsys SkillSourceFS, settingsPath string) (string, error) {
	object, err := readPiJSONObject(fsys, settingsPath)
	if err != nil {
		return "", err
	}

	raw, present := object[piDefaultProjectTrustKey]
	if !present {
		return "", nil
	}

	var value any

	decodeErr := json.Unmarshal(raw, &value)
	if decodeErr != nil {
		return "", fmt.Errorf("decoding %s: %w", piDefaultProjectTrustKey, decodeErr)
	}

	text, _ := value.(string)

	return text, nil
}

// readPiJSONObject reads and decodes one Pi JSON object file. A missing file
// is an empty object; any other failure is an error.
func readPiJSONObject(fsys SkillSourceFS, path string) (map[string]json.RawMessage, error) {
	_, content, found, err := readMarkdownFile(fsys, path)
	if err != nil {
		return nil, err
	}

	if !found {
		return map[string]json.RawMessage{}, nil
	}

	return decodePiJSONObject(content)
}

// readPiTrustStore reads trust.json as canonical path → decision (nil for a
// null entry). A missing file is empty; a value other than true, false or
// null is an error, as in Pi's readTrustFile.
func readPiTrustStore(fsys SkillSourceFS, trustPath string) (map[string]*bool, error) {
	object, err := readPiJSONObject(fsys, trustPath)
	if err != nil {
		return nil, err
	}

	decisions := make(map[string]*bool, len(object))

	for path, raw := range object {
		var decision *bool

		decodeErr := json.Unmarshal(raw, &decision)
		if decodeErr != nil {
			return nil, fmt.Errorf("%w: %q", errPiTrustValue, path)
		}

		decisions[path] = decision
	}

	return decisions, nil
}

// resolveSkillPathOrClean resolves path's symlinks, falling back to the
// cleaned path when resolution fails (Pi's canonicalizePath).
func resolveSkillPathOrClean(fsys SkillSourceFS, path string) string {
	resolved, err := ResolveSkillPath(fsys, path)
	if err != nil {
		return filepath.Clean(path)
	}

	return resolved
}
