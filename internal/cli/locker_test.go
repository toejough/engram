package cli_test

import (
	"errors"
	"io"
	"io/fs"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestPrimLocker_FlockFailureClosesDescriptor(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const fakeFD = uintptr(7)

	closed := make([]uintptr, 0, 1)
	boom := errors.New("flock boom")

	locker := lockerFromPrims(cli.Primitives{
		OpenLockFile:   func(string, fs.FileMode) (uintptr, error) { return fakeFD, nil },
		FlockExclusive: func(uintptr) error { return boom },
		CloseFD: func(fd uintptr) error {
			closed = append(closed, fd)

			return nil
		},
	})

	_, err := locker.Lock("/vault/.lock")
	g.Expect(err).To(gomega.MatchError(boom))
	g.Expect(err.Error()).To(gomega.ContainSubstring("/vault/.lock"))
	g.Expect(closed).To(gomega.Equal([]uintptr{fakeFD}), "flock failure must not leak the fd")
}

func TestPrimLocker_OpenFailureWrapsWithPath(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	boom := errors.New("open boom")

	locker := lockerFromPrims(cli.Primitives{
		OpenLockFile: func(string, fs.FileMode) (uintptr, error) { return 0, boom },
		FlockExclusive: func(uintptr) error {
			t.Error("flock must not run after a failed open")

			return nil
		},
	})

	_, err := locker.Lock("/vault/.lock")
	g.Expect(err).To(gomega.MatchError(boom))
	g.Expect(err.Error()).To(gomega.ContainSubstring("open lock /vault/.lock"))
}

func TestPrimLocker_UnlockLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("unlock flocks LOCK_UN then closes", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		calls := &callRecorder{}
		locker := lockerFromPrims(cli.Primitives{
			OpenLockFile: func(string, fs.FileMode) (uintptr, error) { return 4, nil },
			FlockExclusive: func(uintptr) error {
				calls.add("flock-ex")

				return nil
			},
			FlockUnlock: func(uintptr) error {
				calls.add("flock-un")

				return nil
			},
			CloseFD: func(uintptr) error {
				calls.add("close")

				return nil
			},
		})

		unlock, err := locker.Lock("/vault/.lock")
		g.Expect(err).NotTo(gomega.HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(unlock()).To(gomega.Succeed())
		g.Expect(calls.list()).To(gomega.Equal([]string{"flock-ex", "flock-un", "close"}))
	})

	t.Run("funlock error reported and fd still closed", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		boom := errors.New("funlock boom")
		closed := make([]uintptr, 0, 1)
		locker := lockerFromPrims(cli.Primitives{
			OpenLockFile:   func(string, fs.FileMode) (uintptr, error) { return 4, nil },
			FlockExclusive: func(uintptr) error { return nil },
			FlockUnlock:    func(uintptr) error { return boom },
			CloseFD: func(fd uintptr) error {
				closed = append(closed, fd)

				return nil
			},
		})

		unlock, err := locker.Lock("/vault/.lock")
		g.Expect(err).NotTo(gomega.HaveOccurred())

		if err != nil {
			return
		}

		unlockErr := unlock()
		g.Expect(unlockErr).To(gomega.MatchError(boom))
		g.Expect(unlockErr.Error()).To(gomega.ContainSubstring("funlock"))
		g.Expect(closed).To(gomega.HaveLen(1), "unlock error must not leak the fd")
	})

	t.Run("close error reported when funlock succeeds", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		boom := errors.New("close boom")
		locker := lockerFromPrims(cli.Primitives{
			OpenLockFile:   func(string, fs.FileMode) (uintptr, error) { return 4, nil },
			FlockExclusive: func(uintptr) error { return nil },
			FlockUnlock:    func(uintptr) error { return nil },
			CloseFD:        func(uintptr) error { return boom },
		})

		unlock, err := locker.Lock("/vault/.lock")
		g.Expect(err).NotTo(gomega.HaveOccurred())

		if err != nil {
			return
		}

		unlockErr := unlock()
		g.Expect(unlockErr).To(gomega.MatchError(boom))
		g.Expect(unlockErr.Error()).To(gomega.ContainSubstring("close lock"))
	})
}

// lockerFromPrims composes the production FileLocker from fake primitives
// via the public composition root.
func lockerFromPrims(prims cli.Primitives) cli.FileLocker {
	return cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).Lock
}
