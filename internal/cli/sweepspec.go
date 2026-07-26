package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// SweepEnv is the deterministic input to root resolution: where we are, where
// this project's session logs live, and how to check a directory exists.
type SweepEnv struct {
	Cwd        string
	SessionDir string
	IsDir      func(path string) bool
}

// SweepRoot pairs a resolved sweep root with the exclude rules that apply to
// walks under it (general excludes everywhere; .claude roots add their own).
type SweepRoot struct {
	Path        string
	ExcludeDirs []string
	// ExcludePrefixes prunes any directory whose NAME starts with one of these
	// slugified-cwd prefixes — used to keep non-persistent workspaces (e.g.
	// session logs under `-private-tmp-…`) out of the main index. Empty for
	// manual --sweep roots, so deliberate test ingestion still works.
	ExcludePrefixes []string
	// SkipHidden prunes every dot-directory during the walk — one
	// deterministic rule covering .git, .claude, .layer-run, .obsidian, and
	// whatever appears next, instead of an ever-growing name list.
	SkipHidden bool
	// Origin tags which category of root this is — repo-markdown, an
	// ancestor .claude/.pi dir, a session-log root, a manual --sweep root,
	// or (via SweepSpec.ExtraRoots) an explicitly-configured extra root.
	// Propagated onto every sourceRef discovered under this root, feeding
	// selectCanonical's precedence ranking (Unit 3 dedup). ResolveSweepRoots
	// sets it per category below; assembleSweepRoots sets it for manual
	// --sweep roots.
	Origin sourceOrigin
	// AncestorDepth is the number of directory levels climbed from cwd to
	// reach this ancestor root (0 = cwd itself, 1 = parent, ...) — real
	// physical proximity, not an ordinal position among found ancestor
	// dirs. That distinction matters because selectCanonical's rule 3
	// ("closest ancestor first") compares this value ACROSS origin types
	// once both a claude-ancestor and a pi-ancestor candidate tie at rank
	// 3: a .pi dir found immediately at cwd (level 0) must outrank a
	// .claude dir found one level up (level 1), even though each is the
	// "0th" hit within its own walk. Meaningless (left 0) for every
	// non-ancestor origin.
	AncestorDepth int
}

// SweepSpec declares what `engram ingest --auto` sweeps. It is deliberately
// declarative and inspectable: defaults are compiled in (DefaultSweepSpec),
// and a repo can override them with .engram/sweep.json at its root. Every
// field is data, not behavior — developers tweak the JSON, not the code.
type SweepSpec struct {
	// RepoMarkdown sweeps every .md under the repo root (the nearest ancestor
	// of cwd containing a VCS marker; cwd itself when none is found).
	RepoMarkdown bool `json:"repo_markdown"` //nolint:tagliatelle // developer-facing config uses snake_case
	// AncestorClaudeDirs sweeps every .claude directory on the ancestor chain
	// from cwd up to the filesystem root (project + user-level config/skills).
	AncestorClaudeDirs bool `json:"ancestor_claude_dirs"` //nolint:tagliatelle // developer-facing config uses snake_case
	// AncestorPiDirs sweeps every .pi directory on the ancestor chain for
	// PI coding agent session transcripts.
	AncestorPiDirs bool `json:"ancestor_pi_dirs"` //nolint:tagliatelle // developer-facing config uses snake_case
	// SessionLogs sweeps ALL recorded session transcripts (every project,
	// every conversation) — memory learns from the full conversation history.
	SessionLogs bool `json:"session_logs"` //nolint:tagliatelle // developer-facing config uses snake_case
	// ExtraRoots are swept verbatim, in addition to everything above.
	ExtraRoots []string `json:"extra_roots"` //nolint:tagliatelle // developer-facing config uses snake_case
	// ExcludeDirs are directory NAMES skipped during any sweep walk —
	// build/dependency trees whose markdown is not project memory.
	ExcludeDirs []string `json:"exclude_dirs"` //nolint:tagliatelle // developer-facing config uses snake_case
	// ClaudeExcludeDirs are ADDITIONALLY skipped inside ancestor .claude dirs:
	// harness state and third-party plugin content, not user memory. projects/
	// holds EVERY project's transcripts — this project's sessions come in via
	// session_logs instead.
	ClaudeExcludeDirs []string `json:"claude_exclude_dirs"` //nolint:tagliatelle // developer-facing config uses snake_case
	// IncludeHiddenDirs disables the default pruning of dot-directories
	// (.git, .layer-run, .obsidian, ...) during sweep walks.
	IncludeHiddenDirs bool `json:"include_hidden_dirs"` //nolint:tagliatelle // developer-facing config uses snake_case
	// NonPersistentPrefixes name project-dir prefixes that `--auto` skips:
	// session logs whose slugified cwd lives under a throwaway root
	// (`/private/tmp`, `/tmp`, macOS `$TMPDIR` at `/var/folders`). Eval/test
	// runs never bloat the main index; explicit --sweep/--transcript bypass it.
	//
	// This matches slugified directory NAMES (`-private-tmp-…`) against
	// entry.Name() during a walk, and only ever gets attached to the
	// session-logs root (see ResolveSweepRoots below) — it prunes
	// throwaway *session* subdirectories found underneath a persistent
	// root, not the root itself.
	NonPersistentPrefixes []string `json:"non_persistent_prefixes"` //nolint:tagliatelle,lll // developer-facing config uses snake_case
	// NonPersistentPathPrefixes name real filesystem PATH prefixes under
	// which an entire sweep root is dropped, root and all — a sibling of
	// NonPersistentPrefixes above, not an overload of it: that field
	// matches slugified NAMES and only reaches the session-logs root's
	// children, so it cannot catch a root whose own resolved Path sits
	// under a throwaway tree (e.g. cwd is `/tmp/some-eval-pool/warm0` with
	// no repo marker, so the repo-markdown root IS that path). sweepListerFrom
	// exempts a root's own path from every in-walk exclude check
	// (`if path == root.Path { return nil }`), so without this field such a
	// root is swept whole and permanently indexed. Applies only to --auto's
	// resolved roots; explicit --sweep/--transcript/--markdown bypass it, as
	// documented for NonPersistentPrefixes.
	NonPersistentPathPrefixes []string `json:"non_persistent_path_prefixes"` //nolint:tagliatelle,lll // developer-facing config uses snake_case
}

// DefaultSweepSpec is the compiled-in declaration: repo markdown + ancestor
// .claude dirs + session logs, minus common build/dependency directories.
func DefaultSweepSpec() SweepSpec {
	return SweepSpec{
		RepoMarkdown:       true,
		AncestorClaudeDirs: true,
		AncestorPiDirs:     true,
		SessionLogs:        true,
		ExtraRoots:         nil,
		ExcludeDirs: []string{
			"node_modules", "vendor", ".git", ".hg", ".jj", "dist", "build",
			"target", ".venv", "venv", "__pycache__", ".next", ".cache", ".idea",
		},
		ClaudeExcludeDirs: []string{
			excludeDirProjects, "plugins", "cache", "todos", "shell-snapshots",
			"file-history", "history", "ide", "statsig", "session-env", "debug",
			"worktrees",
		},
		NonPersistentPrefixes: []string{"-private-tmp-", "-tmp-", "-var-folders-", "-private-var-folders-"},
		NonPersistentPathPrefixes: []string{
			nonPersistentPathTmp, nonPersistentPathPrivateTmp,
			nonPersistentPathVarFolders, nonPersistentPathPrivateVarFolders,
		},
	}
}

// LoadSweepSpec overlays a repo's sweep.json onto the defaults: fields the
// file sets win; fields it omits keep their default values.
func LoadSweepSpec(raw []byte) (SweepSpec, error) {
	spec := DefaultSweepSpec()

	err := json.Unmarshal(raw, &spec)
	if err != nil {
		return SweepSpec{}, fmt.Errorf("sweep spec: %w", err)
	}

	return spec, nil
}

// ResolveSweepRoots computes the sweep root list from a spec and environment.
// Pure given env.IsDir — same inputs, same roots, in a stable order. Each root
// carries the exclude list for walks under it.
func ResolveSweepRoots(spec SweepSpec, env SweepEnv) []SweepRoot {
	var roots []SweepRoot

	skipHidden := !spec.IncludeHiddenDirs

	if spec.RepoMarkdown {
		// Hidden dirs (.claude, .layer-run, ...) are pruned by SkipHidden;
		// .claude content comes in via ancestor_claude_dirs with its own rules.
		roots = append(roots, SweepRoot{
			Path: repoRootFor(env), ExcludeDirs: spec.ExcludeDirs, SkipHidden: skipHidden,
			Origin: originRepo,
		})
	}

	if spec.AncestorClaudeDirs {
		claudeExcludes := append(append([]string{}, spec.ExcludeDirs...), spec.ClaudeExcludeDirs...)
		for _, ancestor := range ancestorClaudeDirs(env) {
			roots = append(roots, SweepRoot{
				Path: ancestor.path, ExcludeDirs: claudeExcludes, SkipHidden: skipHidden,
				Origin: originClaudeAncestor, AncestorDepth: ancestor.level,
			})
		}
	}

	if spec.AncestorPiDirs {
		piExcludes := []string{"jobs", excludeDirProjects} // PI sessions have similar subdirs to Claude
		for _, ancestor := range ancestorPiDirs(env) {
			roots = append(roots, SweepRoot{
				Path: ancestor.path, ExcludeDirs: piExcludes, SkipHidden: skipHidden,
				Origin: originPiAncestor, AncestorDepth: ancestor.level,
			})
		}
	}

	if spec.SessionLogs && env.SessionDir != "" && env.IsDir(env.SessionDir) {
		roots = append(roots, SweepRoot{
			Path:            env.SessionDir,
			ExcludeDirs:     spec.ExcludeDirs,
			ExcludePrefixes: spec.NonPersistentPrefixes,
			SkipHidden:      skipHidden,
			Origin:          originSessionLog,
		})
	}

	// Drop non-persistent-path roots BEFORE ExtraRoots is appended: ExtraRoots
	// is an explicit, deliberate configuration choice (like --sweep), not an
	// accidentally-swept throwaway tree, so it must bypass this filter and
	// stay "swept verbatim" per the SweepSpec.ExtraRoots doc comment above —
	// the same bypass NonPersistentPrefixes already gives it.
	roots = dropNonPersistentRoots(roots, spec.NonPersistentPathPrefixes)

	for _, extra := range spec.ExtraRoots {
		roots = append(roots, SweepRoot{
			Path: extra, ExcludeDirs: spec.ExcludeDirs, SkipHidden: skipHidden,
			Origin: originManualSweep,
		})
	}

	return roots
}

// unexported constants.
const (
	excludeDirProjects                 = "projects"
	nonPersistentPathPrivateTmp        = "/private/tmp"
	nonPersistentPathPrivateVarFolders = "/private/var/folders"
	nonPersistentPathTmp               = "/tmp"
	nonPersistentPathVarFolders        = "/var/folders"
)

// ancestorDir pairs a found ancestor directory with its physical distance
// from cwd — the number of directory levels climbed to reach it (0 = cwd
// itself), NOT its ordinal position among found ancestor dirs. Levels are
// climbed one at a time regardless of whether a match is found there, so
// two callers walking different subdirectory names (.claude vs .pi) still
// produce directly-comparable level numbers for the same physical distance.
type ancestorDir struct {
	path  string
	level int
}

// ancestorClaudeDirs collects every existing .claude directory from cwd up
// to the filesystem root (closest first), tagged with its physical level.
func ancestorClaudeDirs(env SweepEnv) []ancestorDir {
	var dirs []ancestorDir

	for dir, level := env.Cwd, 0; ; dir, level = filepath.Dir(dir), level+1 {
		candidate := filepath.Join(dir, ".claude")
		if env.IsDir(candidate) {
			dirs = append(dirs, ancestorDir{path: candidate, level: level})
		}

		if dir == filepath.Dir(dir) {
			return dirs
		}
	}
}

// ancestorPiDirs collects every existing .pi directory from cwd up to
// the filesystem root (closest first), tagged with its physical level.
func ancestorPiDirs(env SweepEnv) []ancestorDir {
	var dirs []ancestorDir

	for dir, level := env.Cwd, 0; ; dir, level = filepath.Dir(dir), level+1 {
		candidate := filepath.Join(dir, ".pi")
		if env.IsDir(candidate) {
			dirs = append(dirs, ancestorDir{path: candidate, level: level})
		}

		if dir == filepath.Dir(dir) {
			return dirs
		}
	}
}

// dropNonPersistentRoots removes any root whose OWN resolved Path sits under
// one of the given throwaway-filesystem prefixes. sweepListerFrom exempts a
// root's own path from every in-walk exclude check, so a root that IS (or is
// nested inside) a non-persistent path — not merely one that contains a
// non-persistent subdirectory — would otherwise be swept in full and
// permanently indexed. Only reached from --auto's resolved roots
// (ResolveSweepRoots); manual --sweep roots are built separately in
// assembleSweepRoots and never pass through here, preserving that escape hatch.
func dropNonPersistentRoots(roots []SweepRoot, prefixes []string) []SweepRoot {
	if len(prefixes) == 0 {
		return roots
	}

	kept := make([]SweepRoot, 0, len(roots))

	for _, root := range roots {
		if isUnderPathPrefix(root.Path, prefixes) {
			continue
		}

		kept = append(kept, root)
	}

	return kept
}

// isUnderPathPrefix reports whether path equals one of prefixes or is nested
// under one, respecting path-segment boundaries (so "/tmpfoo" is not treated
// as under "/tmp").
func isUnderPathPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if path == prefix || strings.HasPrefix(path, prefix+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

// repoRootFor walks from cwd upward looking for a VCS marker; the nearest
// marked ancestor is the repo root. No marker anywhere -> cwd itself.
func repoRootFor(env SweepEnv) string {
	for dir := env.Cwd; ; dir = filepath.Dir(dir) {
		for _, marker := range []string{".git", ".hg", ".jj"} {
			if env.IsDir(filepath.Join(dir, marker)) {
				return dir
			}
		}

		if dir == filepath.Dir(dir) {
			return env.Cwd
		}
	}
}
