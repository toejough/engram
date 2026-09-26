package cli

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/toejough/engram/internal/update"
)

// Exported constants.
const (
	// SkillScopeProjectPrefix prefixes a project's scope ID: `project:<r>`
	// (design D3), where `<r>` is ProjectIdentity.ID.
	SkillScopeProjectPrefix = "project:"
	// SkillSegmentAgents and SkillSegmentPi are the SourceSegment values of
	// a project's `.agents/skills` and `.pi/skills` candidates (ruling R11):
	// the key segments the user scopes agents-user and pi-user contribute
	// (`project:<r>:agents:<n>`, `project:<r>:pi:<n>`, design D3). A
	// project's `.pi/prompts` candidates carry SkillScopePiPrompt.
	SkillSegmentAgents = "agents"
	SkillSegmentPi     = "pi"
)

// ClaudeProjectScan is the input to ScanClaudeProjectSources.
type ClaudeProjectScan struct {
	// Cwd is the working directory registration runs from.
	Cwd string
	// TopLevel is the repository top-level (`git rev-parse
	// --show-toplevel`), the last level of the chain.
	TopLevel string
	// UserSkillsRoot and UserCommandsRoot are ~/.claude/skills and
	// ~/.claude/commands: user sources, never project levels, even when the
	// chain reaches $HOME.
	UserSkillsRoot   string
	UserCommandsRoot string
	// ScopeID is stamped on every candidate (`project:<r>`).
	ScopeID string
}

// ProjectIdentity is the project identity probe's result (design D3).
type ProjectIdentity struct {
	// ID is `<r>`: `<host>/<owner>/<repo>` from the normalized origin URL,
	// or `local/<name>` for a repository without a usable origin.
	ID string
	// TopLevel is `git rev-parse --show-toplevel`: where the project's files
	// physically live, and the end of every project directory chain.
	TopLevel string
}

// ScopeID returns the project's scope ID, `project:<r>`.
func (p ProjectIdentity) ScopeID() string {
	return SkillScopeProjectPrefix + p.ID
}

// ScanClaudeProjectSources scans Claude Code's project sources (design D2
// source 9): `.claude/skills` (ScanSkillChildren rules, so root `.md` files
// are ignored) and `.claude/commands` (ScanCommandDir rules) in scan.Cwd and
// each ancestor up to scan.TopLevel. The chain walks the symlink-resolved
// cwd, so it meets the (resolved) top-level; a cwd outside the top-level
// scans nothing.
//
// Each existing directory is recorded as its own root, so removal
// eligibility is tied to the exact directory a note came from (D5); a level
// without the directory records no root (absent, not failed). A level that
// resolves to the user's own ~/.claude/skills or ~/.claude/commands is
// skipped. The same name at two levels is emitted twice: that is a key
// conflict (task 2.3), never nearest-wins. Skills precede commands, as in
// the harness ("skills take precedence over commands").
func ScanClaudeProjectSources(fsys SkillSourceFS, scan ClaudeProjectScan) SkillScanResult {
	var skills, commands SkillScanResult

	topLevel := resolveSkillPathOrClean(fsys, scan.TopLevel)
	cwd := resolveSkillPathOrClean(fsys, scan.Cwd)

	if !pathWithinRoot(cwd, topLevel) {
		return SkillScanResult{}
	}

	userRoots := []string{
		resolveSkillPathOrClean(fsys, scan.UserSkillsRoot),
		resolveSkillPathOrClean(fsys, scan.UserCommandsRoot),
	}

	for dir := cwd; ; dir = filepath.Dir(dir) {
		claudeDir := filepath.Join(dir, claudeProjectDirName)

		if level, present := projectLevelDir(fsys, filepath.Join(claudeDir, claudeSkillsDirName), userRoots); present {
			mergeSkillScanResult(&skills, ScanSkillChildren(fsys, level, scan.ScopeID))
		}

		if level, present := projectLevelDir(fsys, filepath.Join(claudeDir, claudeCommandsDirName), userRoots); present {
			mergeSkillScanResult(&commands, ScanCommandDir(fsys, level, scan.ScopeID))
		}

		if dir == topLevel || filepath.Dir(dir) == dir {
			break
		}
	}

	sortSkillCandidates(skills.Candidates)
	sortSkillCandidates(commands.Candidates)

	return joinSkillScanResults(skills, commands)
}

// unexported constants.
const (
	claudeCommandsDirName = "commands"
	claudeProjectDirName  = ".claude"
	claudeSkillsDirName   = "skills"
	localProjectPrefix    = "local/"
	remoteGitSuffix       = ".git"
	remoteSchemeSep       = "://"
)

// localProjectID returns `local/<basename of the parent of commonDir>`, the
// identity of a repository without a usable origin: stable across its
// worktrees, which share one common dir.
func localProjectID(commonDir string) string {
	return localProjectPrefix + filepath.Base(filepath.Dir(filepath.Clean(commonDir)))
}

// normalizeProjectRemote turns a git origin URL into `<host>/<path>` (design
// D3 `<r>`): the whole string lowercased, userinfo and any `:port` stripped,
// and trailing `.git` and `/` removed. It parses the scheme form
// (`ssh://`, `https://`, …), the scp form (`git@host:owner/repo`), and an
// already-normalized `host/path`, so normalization is idempotent. ok is
// false for a URL with no host or path — a local path or `file://` origin —
// or one whose result would keep a `:` (e.g. an IPv6 host), which would
// make keys unparseable.
func normalizeProjectRemote(remote string) (string, bool) {
	lowered := strings.ToLower(strings.TrimSpace(remote))

	var host, path string

	if _, rest, isURL := strings.Cut(lowered, remoteSchemeSep); isURL {
		authority, urlPath, _ := strings.Cut(rest, "/")
		host, path = stripRemotePort(stripRemoteUserinfo(authority)), urlPath
	} else {
		host, path = splitSchemelessRemote(lowered)
	}

	path = strings.TrimLeft(path, "/")

	for {
		trimmed := strings.TrimSuffix(strings.TrimRight(path, "/"), remoteGitSuffix)
		if trimmed == path {
			break
		}

		path = trimmed
	}

	if !validRemoteHost(host) || path == "" || strings.ContainsAny(path, ": \t\n") {
		return "", false
	}

	return host + "/" + path, true
}

// pathWithinRoot reports whether path is root or lies beneath it.
func pathWithinRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)

	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// probeProjectIdentity runs the project identity probe (design D3) in cwd
// through the injected commander. Outside a git repository (show-toplevel
// fails or is empty), found is false and there are no project sources.
// Otherwise the ID is the normalized origin URL, or — with no usable origin —
// `local/<basename of the parent of git rev-parse --path-format=absolute
// --git-common-dir>`; found is false when that also fails.
func probeProjectIdentity(ctx context.Context, cwd string, commander update.Commander) (ProjectIdentity, bool) {
	topLevel, inRepo := gitTopLevel(ctx, commander, cwd)
	if !inRepo || topLevel == "" {
		return ProjectIdentity{}, false
	}

	if origin, found := gitOriginURL(ctx, commander, cwd); found {
		if id, parsed := normalizeProjectRemote(origin); parsed {
			return ProjectIdentity{ID: id, TopLevel: topLevel}, true
		}
	}

	commonDir, ok := runGit(ctx, commander, cwd, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if !ok || commonDir == "" {
		return ProjectIdentity{}, false
	}

	return ProjectIdentity{ID: localProjectID(commonDir), TopLevel: topLevel}, true
}

// projectLevelDir reports whether one project chain level's directory is a
// source to scan: it is when it resolves to something other than a user
// root, or when resolving it failed for a reason other than not-exist (the
// scanner then records it not scanned, with a warning). A missing directory
// is absent.
func projectLevelDir(fsys SkillSourceFS, dir string, userRoots []string) (string, bool) {
	resolved, _, err := resolveSkillPathInfo(fsys, dir)

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", false
	case err == nil && slices.Contains(userRoots, resolved):
		return "", false
	default:
		return dir, true
	}
}

// splitSchemelessRemote splits an scp-form remote (`[user@]host:path`,
// recognized by a `:` before the first `/`) or a plain `host/path` into host
// and path, with any userinfo stripped.
func splitSchemelessRemote(remote string) (host, path string) {
	head := remote
	if slash := strings.IndexByte(remote, '/'); slash >= 0 {
		head = remote[:slash]
	}

	if strings.Contains(head, ":") {
		if at := strings.LastIndexByte(head, '@'); at >= 0 {
			remote = remote[at+1:]
		}

		host, path, _ = strings.Cut(remote, ":")

		return host, path
	}

	host, path, _ = strings.Cut(remote, "/")

	return stripRemoteUserinfo(host), path
}

// stripRemotePort removes a trailing `:<digits>` port from host.
func stripRemotePort(host string) string {
	colon := strings.LastIndexByte(host, ':')
	if colon < 0 {
		return host
	}

	for _, char := range host[colon+1:] {
		if char < '0' || char > '9' {
			return host
		}
	}

	return host[:colon]
}

// stripRemoteUserinfo removes a `user[:token]@` prefix from an authority.
func stripRemoteUserinfo(authority string) string {
	if at := strings.LastIndexByte(authority, '@'); at >= 0 {
		return authority[at+1:]
	}

	return authority
}

// validRemoteHost reports whether host can name a forge: non-empty, not a
// relative-path component, and free of `:` and whitespace.
func validRemoteHost(host string) bool {
	return host != "" && host != "." && host != ".." && !strings.ContainsAny(host, ": \t\n")
}
