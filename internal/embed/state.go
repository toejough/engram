package embed

import (
	"errors"
	"fmt"
	"io/fs"
)

// FS is the read-only filesystem surface used by ComputeState. The
// production reader returns *os.PathError which satisfies errors.Is for
// fs.ErrNotExist; test fakes can hand any error implementing
// IsNotExist() bool — the interface fallback covers them.
type FS interface {
	ReadFile(path string) ([]byte, error)
}

// ComputeState reads notePath and the sibling .vec.json and returns the
// note's State relative to currentModelID. Stale-vs-incompatible
// precedence: model_id mismatch first (a re-embed under the new model
// also picks up any body change), then content_hash mismatch.
//
// The function only returns a non-nil error when the note itself is
// unreadable — sidecar problems are reported as classification states
// (Missing / Broken) so callers can iterate over a whole vault without
// short-circuiting on the first unhappy note.
func ComputeState(filesystem FS, notePath, currentModelID string) (State, error) {
	noteBytes, noteErr := filesystem.ReadFile(notePath)
	if noteErr != nil {
		return StateBroken, fmt.Errorf("read note %s: %w", notePath, noteErr)
	}

	scBytes, scErr := filesystem.ReadFile(SidecarPath(notePath))
	if scErr != nil {
		if notExist(scErr) {
			return StateMissing, nil
		}

		// Sidecar present but unreadable for some other reason — report
		// as Broken rather than propagating, so vault-wide passes
		// continue past the bad file. The state itself is the report.
		return StateBroken, nil
	}

	sidecar, parseErr := UnmarshalSidecar(scBytes)
	if parseErr != nil {
		// Sidecar parseable as JSON but mis-shaped (bad dims, bad
		// embedding_model_id) — same Broken-state treatment.
		return StateBroken, nil //nolint:nilerr // intentional: broken-state classification is the report
	}

	if sidecar.EmbeddingModelID != currentModelID {
		return StateIncompatible, nil
	}

	if sidecar.ContentHash != ContentHash(noteBytes) {
		return StateStale, nil
	}

	return StateOK, nil
}

// notExist reports whether err is a "file does not exist" error.
func notExist(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, fs.ErrNotExist) {
		return true
	}

	var typed interface{ IsNotExist() bool }
	if errors.As(err, &typed) && typed.IsNotExist() {
		return true
	}

	return false
}
