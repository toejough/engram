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
	// SessionLogs sweeps the current project's session transcript directory.
	SessionLogs bool `json:"session_logs"` //nolint:tagliatelle // developer-facing config uses snake_case
	// ExtraRoots are swept verbatim, in addition to everything above.
	ExtraRoots []string `json:"extra_roots"` //nolint:tagliatelle // developer-facing config uses snake_case
	// ExcludeDirs are directory NAMES skipped during any sweep walk —
	// build/dependency trees whose markdown is not project memory.
	ExcludeDirs []string `json:"exclude_dirs"` //nolint:tagliatelle // developer-facing config uses snake_case
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
// Pure given env.IsDir — same inputs, same roots, in a stable order.
func ResolveSweepRoots(spec SweepSpec, env SweepEnv) []string {
	var roots []string

	if spec.RepoMarkdown {
		roots = append(roots, repoRootFor(env))
	}

	if spec.AncestorClaudeDirs {
		roots = append(roots, ancestorClaudeDirs(env)...)
	}

	if spec.SessionLogs && env.SessionDir != "" && env.IsDir(env.SessionDir) {
		roots = append(roots, env.SessionDir)
	}

	roots = append(roots, spec.ExtraRoots...)

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
