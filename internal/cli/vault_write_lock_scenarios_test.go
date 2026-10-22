package cli_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func Test792b8c2a2_VaultWriteLock_UnlockFailureReportedAndFileStillClosed(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const lockPath = "/vault/.luhmann.lock"

	boom := errors.New("funlock boom")
	calls := &callRecorder{}
	locker := lockerFromPrims(cli.Primitives{Lock: cli.LockPrims{
		OpenLockFile:   func(string, fs.FileMode) (uintptr, error) { return 4, nil },
		FlockExclusive: func(uintptr) error { return nil },
		FlockUnlock: func(uintptr) error {
			calls.add("flock-un")

			return boom
		},
		CloseFD: func(uintptr) error {
			calls.add("close")

			return nil
		},
	}})

	unlock, err := locker.Lock(lockPath)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil || unlock == nil {
		return
	}

	releaseErr := unlock()

	g.Expect(releaseErr).To(gomega.MatchError(boom))
	g.Expect(releaseErr).To(gomega.MatchError(gomega.ContainSubstring("funlock " + lockPath)))
	g.Expect(calls.list()).To(gomega.Equal([]string{"flock-un", "close"}), "the file is closed after the failed unlock")
}

func Test792b8c2a3_VaultWriteLock_CloseFailureReported(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const lockPath = "/vault/.luhmann.lock"

	boom := errors.New("close boom")
	calls := &callRecorder{}
	locker := lockerFromPrims(cli.Primitives{Lock: cli.LockPrims{
		OpenLockFile:   func(string, fs.FileMode) (uintptr, error) { return 4, nil },
		FlockExclusive: func(uintptr) error { return nil },
		FlockUnlock: func(uintptr) error {
			calls.add("flock-un")

			return nil
		},
		CloseFD: func(uintptr) error {
			calls.add("close")

			return boom
		},
	}})

	unlock, err := locker.Lock(lockPath)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil || unlock == nil {
		return
	}

	releaseErr := unlock()

	g.Expect(releaseErr).To(gomega.MatchError(boom))
	g.Expect(releaseErr).To(gomega.MatchError(gomega.ContainSubstring("close lock " + lockPath)))
	g.Expect(calls.list()).To(gomega.Equal([]string{"flock-un", "close"}))
}
