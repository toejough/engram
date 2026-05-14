// Package learnmarker tracks the per-project timestamp of the most recent
// successful transcript scope advance ("last /learn"). The marker is a
// single RFC3339Nano timestamp written to a file under the XDG state dir.
package learnmarker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FS is the minimal filesystem surface learnmarker needs.
// OSFS in osfs.go wraps os.* for production; tests inject fakes.
type FS interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

// MarkerPath returns the full path to the last-learn-at file for a given project slug.
func MarkerPath(stateDir, projectSlug string) string {
	return filepath.Join(stateDir, "projects", projectSlug, "last-learn-at")
}

// MarkerPathWithSuffix returns the full path to a per-harness marker file,
// e.g. "last-learn-at-claude" or "last-learn-at-opencode".
func MarkerPathWithSuffix(stateDir, projectSlug, suffix string) string {
	return filepath.Join(stateDir, "projects", projectSlug, "last-learn-at-"+suffix)
}

// Read returns the marker timestamp at path. The bool return is true when the
// marker file existed; false (with nil error) when it did not — callers handle
// the absent case (first-run) without treating it as an error.
func Read(fs FS, path string) (time.Time, bool, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, false, nil
		}

		return time.Time{}, false, fmt.Errorf("learnmarker: reading %s: %w", path, err)
	}

	t, parseErr := time.Parse(time.RFC3339Nano, string(data))
	if parseErr != nil {
		return time.Time{}, false, fmt.Errorf("learnmarker: parsing %s: %w", path, parseErr)
	}

	return t, true, nil
}

// StateDirFromHome returns the engram state directory.
// Respects $XDG_STATE_HOME if set, otherwise defaults to $HOME/.local/state/engram.
// getenv is injected so callers control environment access (pass os.Getenv in production).
func StateDirFromHome(home string, getenv func(string) string) string {
	if xdg := getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "engram")
	}

	return filepath.Join(home, ".local", "state", "engram")
}

// Write replaces the marker file at path with the RFC3339Nano-formatted timestamp.
// Creates parent directories as needed (0o755 perms).
func Write(fs FS, path string, when time.Time) error {
	err := fs.MkdirAll(filepath.Dir(path), dirPerm)
	if err != nil {
		return fmt.Errorf("learnmarker: writing %s: %w", path, err)
	}

	err = fs.WriteFile(path, []byte(when.Format(time.RFC3339Nano)), filePerm)
	if err != nil {
		return fmt.Errorf("learnmarker: writing %s: %w", path, err)
	}

	return nil
}

// unexported constants.
const (
	dirPerm  = 0o755
	filePerm = 0o644
)
