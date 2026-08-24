package cli_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// TestMain removes sharedEngramBinary's temp directory after this test
// binary's whole run completes. The directory is deliberately built once and
// shared across every parallel e2e test in this run (see sharedEngramBinary),
// so no single test's t.Cleanup() is the right hook — only the binary-wide
// exit is. Without this, a long-lived, non-ephemeral sandbox (unlike a CI
// runner that resets between runs) accumulates one ~113MB leaked directory
// per `targ test-integration` invocation, on tmpfs — RAM-backed, not disk —
// until host memory pressure starts killing unrelated memory-hungry
// processes (observed: nilaway/check-nils-for-fail OOM-killed by 6GB of
// leaked engram-e2e-bin* directories filling /tmp, 2026-08-24).
func TestMain(m *testing.M) {
	code := m.Run()

	if sharedBinaryDir != "" {
		if err := os.RemoveAll(sharedBinaryDir); err != nil {
			fmt.Fprintf(os.Stderr, "TestMain: removing shared e2e binary dir %s: %v\n", sharedBinaryDir, err)
		}
	}

	os.Exit(code)
}

// unexported variables.
var (
	errSharedBinary  error
	sharedBinaryDir  string
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
		//nolint:usetesting // dir lifecycle is intentionally outside any single test's cleanup; TestMain removes it
		tmpDir, err := os.MkdirTemp("", "engram-e2e-bin")
		if err != nil {
			errSharedBinary = err
			return
		}

		sharedBinaryDir = tmpDir

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
