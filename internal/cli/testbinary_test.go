package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// unexported variables.
var (
	errSharedBinary  error
	sharedBinaryOnce sync.Once
	sharedBinaryOut  []byte
	sharedBinaryPath string
)

// sharedEngramBinary builds the engram binary once and returns its path.
// The binary and its directory are written once and never mutated, shared
// immutable state across all parallel e2e tests. Each test keeps its own
// vault, cache, and XDG tempdirs. On build error, the helper calls t.Fatalf
// and does not return.
func sharedEngramBinary(t *testing.T) string {
	t.Helper()

	sharedBinaryOnce.Do(func() {
		//nolint:usetesting // dir lifecycle is intentionally outside test cleanup
		tmpDir, err := os.MkdirTemp("", "engram-e2e-bin")
		if err != nil {
			errSharedBinary = err
			return
		}

		binPath := filepath.Join(tmpDir, "engram")
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/engram")
		cmd.Dir = projectRoot(t)

		sharedBinaryOut, errSharedBinary = cmd.CombinedOutput()
		if errSharedBinary == nil {
			sharedBinaryPath = binPath
		}
	})

	if errSharedBinary != nil {
		t.Fatalf("failed to build shared engram binary: %v\nbuild output:\n%s", errSharedBinary, sharedBinaryOut)
	}

	return sharedBinaryPath
}
