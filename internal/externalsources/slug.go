package externalsources

import "strings"

// ProjectSlug returns the slugified form of an absolute path that Claude Code
// uses when constructing the auto-memory directory path
// (~/.claude/projects/<slug>/memory/).
//
// Claude Code's algorithm: replace every "/" in the absolute path with "-".
// "/Users/joe/repos/engram" → "-Users-joe-repos-engram".
//
// Verified against the actual ~/.claude/projects/ subdirectories on the dev
// machine. If Claude Code changes this algorithm, this function (and the
// auto memory resolver) must be updated together.
func ProjectSlug(absPath string) string {
	return strings.ReplaceAll(absPath, "/", "-")
}
