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

// sliceIndex returns the first index of target in sl, -1 if absent.
func sliceIndex(sl []string, target string) int {
	for i, v := range sl {
		if v == target {
			return i
		}
	}

	return -1
}
