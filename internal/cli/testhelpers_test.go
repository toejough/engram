package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestOsManifestLock_MkdirError covers the MkdirAll-error branch of
// osManifestLock: passing a path whose parent is a regular file makes
// MkdirAll fail with ENOTDIR.
func TestOsManifestLock_MkdirError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Create a regular file, then ask MkdirAll to treat it as a directory
	// parent — this forces an ENOTDIR error.
	tmp := t.TempDir()
	blockedPath := filepath.Join(tmp, "isfile")
	g.Expect(os.WriteFile(blockedPath, []byte("x"), 0o600)).To(gomega.Succeed())

	_, err := cli.ExportOsManifestLock(filepath.Join(blockedPath, "subdir"))
	g.Expect(err).To(gomega.HaveOccurred())
}

// realFSDepsForTest is the learn-family test Deps: production Deps composed
// by cli.NewDeps over real OS primitives (T1-rework's realDepsForTest), with
// Embed forced nil so auto-embed skips — unit tests must not load the
// bundled model (the embed-on-write path stays covered by cli_test.go's
// real-binary end-to-end test). No signal registration occurs:
// realPrimitives() omits StartSignalPulses, so startForceExit nil-skips
// (doctrine flag SIG-1).
func realFSDepsForTest() cli.Deps {
	deps := realDepsForTest()
	deps.Embed = nil

	return deps
}

// sliceIndex returns the first index of target in sl, -1 if absent.
func sliceIndex(sl []string, target string) int {
	for i, v := range sl {
		if v == target {
			return i
		}
	}

	return -1
}
