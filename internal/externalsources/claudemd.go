package externalsources

import (
	"path/filepath"
)

// StatFunc reports whether a file exists at the given absolute path.
// Wired at the edge to a thin os.Stat wrapper.
type StatFunc func(path string) (exists bool, err error)

// DiscoverClaudeMd walks ancestors from cwd to "/", collecting any CLAUDE.md
// or CLAUDE.local.md it finds. It also adds the user-scope CLAUDE.md
// (~/.claude/CLAUDE.md) and the managed-policy CLAUDE.md if present.
//
// goos is the runtime.GOOS value (passed in for testability).
//
// Files that do not exist are silently skipped. Stat errors are also skipped.
func DiscoverClaudeMd(cwd, home, goos string, statFn StatFunc) []ExternalFile {
	files := make([]ExternalFile, 0, defaultClaudeMdCapacity)
	files = walkAncestors(files, cwd, statFn)
	files = addUserScope(files, home, statFn)
	files = addManagedPolicy(files, goos, statFn)

	return files
}

// ManagedPolicyPath returns the documented system-wide CLAUDE.md location for
// the given GOOS, or empty string for an unrecognized OS.
func ManagedPolicyPath(goos string) string {
	switch goos {
	case "darwin":
		return "/Library/Application Support/ClaudeCode/CLAUDE.md"
	case "linux":
		return "/etc/claude-code/CLAUDE.md"
	case "windows":
		return `C:\Program Files\ClaudeCode\CLAUDE.md`
	default:
		return ""
	}
}

// unexported constants.
const (
	defaultClaudeMdCapacity = 8
)

// addManagedPolicy appends the OS-specific managed-policy CLAUDE.md when it
// exists. Unrecognized GOOS values produce no entry.
func addManagedPolicy(files []ExternalFile, goos string, statFn StatFunc) []ExternalFile {
	managed := ManagedPolicyPath(goos)
	if managed == "" {
		return files
	}

	if exists, _ := statFn(managed); exists {
		files = append(files, ExternalFile{Kind: KindClaudeMd, Path: managed})
	}

	return files
}

// addUserScope appends the user-scope CLAUDE.md (~/.claude/CLAUDE.md) when it
// exists. A blank home is treated as "no user scope".
func addUserScope(files []ExternalFile, home string, statFn StatFunc) []ExternalFile {
	if home == "" {
		return files
	}

	userMd := filepath.Join(home, ".claude", "CLAUDE.md")
	if exists, _ := statFn(userMd); exists {
		files = append(files, ExternalFile{Kind: KindClaudeMd, Path: userMd})
	}

	return files
}

// walkAncestors visits each directory from start up to the filesystem root,
// appending any CLAUDE.md / CLAUDE.local.md that statFn reports as existing.
func walkAncestors(files []ExternalFile, start string, statFn StatFunc) []ExternalFile {
	filenames := []string{"CLAUDE.md", "CLAUDE.local.md"}

	dir := start

	for {
		for _, name := range filenames {
			candidate := filepath.Join(dir, name)
			if exists, _ := statFn(candidate); exists {
				files = append(files, ExternalFile{Kind: KindClaudeMd, Path: candidate})
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return files
}
