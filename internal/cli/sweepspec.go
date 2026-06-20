package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
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
	NonPersistentPrefixes []string `json:"non_persistent_prefixes"` //nolint:tagliatelle // developer-facing config uses snake_case
}

// DefaultSweepSpec is the compiled-in declaration: repo markdown + ancestor
// .claude dirs + session logs, minus common build/dependency directories.
func DefaultSweepSpec() SweepSpec {
	return SweepSpec{
		RepoMarkdown:       true,
		AncestorClaudeDirs: true,
		SessionLogs:        true,
		ExtraRoots:         nil,
		ExcludeDirs: []string{
			"node_modules", "vendor", ".git", ".hg", ".jj", "dist", "build",
			"target", ".venv", "venv", "__pycache__", ".next", ".cache", ".idea",
		},
		ClaudeExcludeDirs: []string{
			"projects", "plugins", "cache", "todos", "shell-snapshots",
			"file-history", "history", "ide", "statsig", "session-env", "debug",
			"worktrees",
		},
		NonPersistentPrefixes: []string{"-private-tmp-", "-tmp-", "-var-folders-"},
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
		roots = append(roots, SweepRoot{Path: repoRootFor(env), ExcludeDirs: spec.ExcludeDirs, SkipHidden: skipHidden})
	}

	if spec.AncestorClaudeDirs {
		claudeExcludes := append(append([]string{}, spec.ExcludeDirs...), spec.ClaudeExcludeDirs...)
		for _, dir := range ancestorClaudeDirs(env) {
			roots = append(roots, SweepRoot{Path: dir, ExcludeDirs: claudeExcludes, SkipHidden: skipHidden})
		}
	}

	if spec.SessionLogs && env.SessionDir != "" && env.IsDir(env.SessionDir) {
		roots = append(roots, SweepRoot{
			Path:            env.SessionDir,
			ExcludeDirs:     spec.ExcludeDirs,
			ExcludePrefixes: spec.NonPersistentPrefixes,
			SkipHidden:      skipHidden,
		})
	}

	for _, extra := range spec.ExtraRoots {
		roots = append(roots, SweepRoot{Path: extra, ExcludeDirs: spec.ExcludeDirs, SkipHidden: skipHidden})
	}

	return roots
}

// ancestorClaudeDirs collects every existing .claude directory from cwd up to
// the filesystem root (closest first).
func ancestorClaudeDirs(env SweepEnv) []string {
	var dirs []string

	for dir := env.Cwd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, ".claude")
		if env.IsDir(candidate) {
			dirs = append(dirs, candidate)
		}

		if dir == filepath.Dir(dir) {
			return dirs
		}
	}
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
