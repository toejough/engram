package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"engram/internal/externalsources"
)

// computeMainProjectDir returns the slug-based memory dir for the main repo
// when cwd is inside a worktree distinct from the main checkout. Returns
// empty string when not in a worktree (or git is unavailable).
func computeMainProjectDir(ctx context.Context, cwd, home string) string {
	mainRepoRoot := detectMainRepoRoot(ctx, cwd)
	if mainRepoRoot == "" || mainRepoRoot == cwd {
		return ""
	}

	return filepath.Join(home, ".claude", "projects", externalsources.ProjectSlug(mainRepoRoot), "memory")
}

// detectMainRepoRoot returns the main repo root if cwd is inside a git
// worktree distinct from the main checkout. Returns "" on any error or
// non-worktree case.
func detectMainRepoRoot(ctx context.Context, cwd string) string {
	cmd := exec.CommandContext(ctx, //nolint:gosec // fixed argv
		"git", "-C", cwd, "rev-parse", "--git-common-dir")

	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	commonDir := strings.TrimSpace(string(out))
	if commonDir == "" {
		return ""
	}

	return filepath.Dir(commonDir)
}

// discoverExternalSources builds the externalsources.Discover input for the
// CLI's recall flow and returns the discovered file list along with the
// shared FileCache the recall pipeline will read through.
func discoverExternalSources(
	ctx context.Context,
	home string,
) ([]externalsources.ExternalFile, *externalsources.FileCache) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}

	cache := externalsources.NewFileCache(os.ReadFile)

	cwdProjectDir := filepath.Join(home, ".claude", "projects", externalsources.ProjectSlug(cwd), "memory")
	mainProjectDir := computeMainProjectDir(ctx, cwd, home)

	deps := externalsources.DiscoverDeps{
		CWD:            cwd,
		Home:           home,
		GOOS:           runtime.GOOS,
		CWDProjectDir:  cwdProjectDir,
		MainProjectDir: mainProjectDir,
		StatFn:         osStatExists,
		Reader:         cache.Read,
		MdWalker:       osWalkMd,
		MatchAny:       osMatchAny(cwd),
		Settings:       readAutoMemoryDirectorySetting(home),
		DirLister:      osDirListMd,
		SkillFinder:    osWalkSkills,
	}

	return externalsources.Discover(deps), cache
}

// osDirListMd returns absolute paths to *.md files in dir (non-recursive).
// Returns nil for missing dirs or read errors so DiscoverAutoMemory treats it
// as "no files".
func osDirListMd(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil //nolint:nilerr // missing dir is normal for opt-in auto memory
	}

	out := make([]string, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			out = append(out, filepath.Join(dir, entry.Name()))
		}
	}

	return out, nil
}

// osMatchAny returns a GlobMatcher that reports whether any file under cwd
// matches any of the globs.
//
// Limitation: filepath.Glob does not support `**` (double-star) patterns.
// A rule with `paths: ["src/api/**/*.ts"]` will never match — the `**` is
// treated as a literal directory name. To support double-star globs, swap
// in github.com/bmatcuk/doublestar.Glob. For first ship this is a known
// safe-failure mode (rules with unsupported globs are excluded, never
// spuriously included).
func osMatchAny(cwd string) externalsources.GlobMatcher {
	return func(globs []string) bool {
		for _, glob := range globs {
			matches, err := filepath.Glob(filepath.Join(cwd, glob))
			if err == nil && len(matches) > 0 {
				return true
			}
		}

		return false
	}
}

// osStatExists reports whether a file exists at path. Adapter for
// externalsources.StatFunc.
func osStatExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("stat %s: %w", path, err)
}

// osWalkMd returns absolute paths to all *.md files under root (recursive).
// Errors and missing directories are silently treated as empty.
func osWalkMd(root string) []string {
	return walkMatching(root, func(entry fs.DirEntry) bool {
		return filepath.Ext(entry.Name()) == ".md"
	})
}

// osWalkSkills returns absolute paths to all SKILL.md files under root.
func osWalkSkills(root string) []string {
	return walkMatching(root, func(entry fs.DirEntry) bool {
		return entry.Name() == "SKILL.md"
	})
}

// readAutoMemoryDirectorySetting returns an AutoMemorySettingsFunc that reads
// the user/local settings.json for the autoMemoryDirectory field. Returns
// ("", false) on any error or when the field is absent.
func readAutoMemoryDirectorySetting(home string) externalsources.AutoMemorySettingsFunc {
	paths := []string{
		filepath.Join(".", ".claude", "settings.local.json"),
		filepath.Join(home, ".claude", "settings.json"),
	}

	return func() (string, bool) {
		for _, path := range paths {
			body, err := os.ReadFile(path) //nolint:gosec // user-controlled path inside .claude
			if err != nil {
				continue
			}

			var settings struct {
				AutoMemoryDirectory string `json:"autoMemoryDirectory"`
			}

			if json.Unmarshal(body, &settings) == nil && settings.AutoMemoryDirectory != "" {
				return settings.AutoMemoryDirectory, true
			}
		}

		return "", false
	}
}

// walkMatching returns absolute paths to every non-directory entry under root
// for which match returns true. Per-subtree errors are silently skipped.
func walkMatching(root string, match func(fs.DirEntry) bool) []string {
	found := make([]string, 0)

	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil //nolint:nilerr // skip unreadable subtrees, continue walk
		}

		if match(entry) {
			found = append(found, path)
		}

		return nil
	})

	return found
}
