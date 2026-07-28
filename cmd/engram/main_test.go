package main

import (
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/gomega"
)

// TestSpawnPrimitives_ExitCodeOnNonZeroExit guards that a started child's
// non-zero exit comes back via exitCode with a nil error (not via err) —
// the spawn-failure/exit-code distinction the D1 design requires.
func TestSpawnPrimitives_ExitCodeOnNonZeroExit(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("shell-based fixture assumes a POSIX shell")
	}

	g := NewWithT(t)

	code, err := spawnPrimitives().RunInherited("sh", []string{"-c", "exit 7"}, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(code).To(Equal(7))
}

// TestSpawnPrimitives_ExitCodeOnSuccess guards the production spawn
// closure in main.go's spawnPrimitives: a clean exit returns exit code 0
// and a nil error.
func TestSpawnPrimitives_ExitCodeOnSuccess(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("shell-based fixture assumes a POSIX shell")
	}

	g := NewWithT(t)

	code, err := spawnPrimitives().RunInherited("true", nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(code).To(Equal(0))
}

// TestSpawnPrimitives_PassesArgsAndEnv guards that args and extra
// environment variables reach the child process.
func TestSpawnPrimitives_PassesArgsAndEnv(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("shell-based fixture assumes a POSIX shell")
	}

	g := NewWithT(t)

	marker := filepath.Join(t.TempDir(), "marker")
	code, err := spawnPrimitives().RunInherited(
		"sh",
		[]string{"-c", `[ "$1" = "arg-one" ] && [ "$ENGRAM_TEST_SENTINEL" = "1" ] && touch "$2"`, "sh", "arg-one", marker},
		[]string{"ENGRAM_TEST_SENTINEL=1"},
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(code).To(Equal(0))
	g.Expect(marker).To(BeAnExistingFile())
}

// TestSpawnPrimitives_SpawnFailureReturnsErr guards that a binary which
// cannot be started (missing/not executable) reports a non-nil err —
// never confused with a started child's exit code.
func TestSpawnPrimitives_SpawnFailureReturnsErr(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	missing := filepath.Join(t.TempDir(), "no-such-engram-binary")

	_, err := spawnPrimitives().RunInherited(missing, nil, nil)
	g.Expect(err).To(HaveOccurred())
}
