//go:build integration

package cli_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
)

// Test792b8c1a2_RealFlockLocker_AbsentLockFileIsCreatedWithMode0600 takes a lock over the real syscall primitives
// on a path with no file and checks, through the real filesystem, that the lock file exists afterwards with mode 0600.
// The fake-based twin lives in locker_test.go; this is the real-boundary check.
func Test792b8c1a2_RealFlockLocker_AbsentLockFileIsCreatedWithMode0600(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	lockPath := filepath.Join(t.TempDir(), "absent.lock")

	_, statErr := os.Lstat(lockPath)
	g.Expect(statErr).To(gomega.MatchError(fs.ErrNotExist), "Given: a lock path with no file")

	unlock, err := realDepsForTest().Lock.Lock(lockPath)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil || unlock == nil {
		return
	}

	info, infoErr := os.Lstat(lockPath)
	g.Expect(infoErr).NotTo(gomega.HaveOccurred())

	if infoErr == nil && info != nil {
		g.Expect(info.Mode().IsRegular()).To(gomega.BeTrue(), "the lock file is a regular file")
		g.Expect(info.Mode().Perm()).To(gomega.Equal(vwl1LockFileMode))
	}

	g.Expect(unlock()).To(gomega.Succeed())
}

// unexported constants.
const (
	vwl1LockFileMode = fs.FileMode(0o600)
)
