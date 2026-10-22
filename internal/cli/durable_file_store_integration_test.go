//go:build integration

package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/onsi/gomega"
)

// Test792b3c1a5_RealEdgeFS_WriteFileAtomicChmodFailureRemovesTemp is the
// real-boundary twin of the fake-primitive chmod subtest of
// Test792b3c1a4_EdgeFS_WriteFileAtomicFailureRemovesTemp: the temp is created,
// listed and removed on the real filesystem; only the chmod step fails.
func Test792b3c1a5_RealEdgeFS_WriteFileAtomicChmodFailureRemovesTemp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	original := []byte("original content")

	g.Expect(os.WriteFile(target, original, realFSFilePerm)).To(gomega.Succeed())

	// Real primitives except Chmod: it first proves the dance has created a
	// real temp beside the target, then fails the way the OS refuses a
	// permission change.
	var tmpSeen string

	var tmpExistedAtChmod bool

	prims := realPrimitives()
	prims.FS.Chmod = func(name string, _ os.FileMode) error {
		tmpSeen = name
		_, statErr := os.Stat(name)
		tmpExistedAtChmod = statErr == nil

		return &os.PathError{Op: "chmod", Path: name, Err: syscall.EPERM}
	}

	err := fsFromPrims(prims).WriteFileAtomic(target, []byte("new content"), realFSFilePerm)
	g.Expect(err).To(gomega.MatchError(syscall.EPERM))
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring(target)), "error must name the target path")
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("chmod")), "error must name the failing dance step")

	g.Expect(errors.Is(err, syscall.EPERM)).To(gomega.BeTrue())
	g.Expect(tmpSeen).NotTo(gomega.BeEmpty(), "the dance must reach the chmod step")
	g.Expect(tmpExistedAtChmod).To(gomega.BeTrue(), "a real temp must exist when chmod is attempted")
	g.Expect(tmpSeen).NotTo(gomega.BeAnExistingFile(), "temp file must be removed by the failure cleanup")

	entries, readErr := os.ReadDir(dir)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1), "only the original may remain in the directory")

	got, readFileErr := os.ReadFile(target)
	g.Expect(readFileErr).NotTo(gomega.HaveOccurred())

	if readFileErr != nil {
		return
	}

	g.Expect(got).To(gomega.Equal(original), "original must be untouched after chmod failure")
}
