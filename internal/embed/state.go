package embed

import (
	"errors"
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
// ComputeState never returns an error: every failure mode classifies as
// a State (Missing for absent sidecar; Broken for unreadable note,
// unreadable sidecar, malformed JSON, or dims mismatch). The State IS
// the report so vault-wide passes can iterate without short-circuiting.
func ComputeState(filesystem FS, notePath, currentModelID string) State {
	noteBytes, noteErr := filesystem.ReadFile(notePath)
	if noteErr != nil {
		return StateBroken
	}

	scBytes, scErr := filesystem.ReadFile(SidecarPath(notePath))
	if scErr != nil {
		if notExist(scErr) {
			return StateMissing
		}

		return StateBroken
	}

	sidecar, parseErr := UnmarshalSidecar(scBytes)
	if parseErr != nil {
		return StateBroken
	}

	if sidecar.EmbeddingModelID != currentModelID {
		return StateIncompatible
	}

	if sidecar.ContentHash != ContentHash(noteBytes) {
		return StateStale
	}

	return StateOK
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
