// Package learnmarker tracks the per-project timestamp of the most recent
// successful transcript scope advance ("last /learn"). The marker is a
// single RFC3339Nano timestamp written to a file under the XDG state dir.
package learnmarker

import "path/filepath"

// StateDirFromHome returns the engram state directory.
// Respects $XDG_STATE_HOME if set, otherwise defaults to $HOME/.local/state/engram.
// getenv is injected so callers control environment access (pass os.Getenv in production).
func StateDirFromHome(home string, getenv func(string) string) string {
	if xdg := getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "engram")
	}

	return filepath.Join(home, ".local", "state", "engram")
}

// MarkerPath returns the full path to the last-learn-at file for a given project slug.
func MarkerPath(stateDir, projectSlug string) string {
	return filepath.Join(stateDir, "projects", projectSlug, "last-learn-at")
}
