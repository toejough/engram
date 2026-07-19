package cli

import "fmt"

// unexported constants.
const (
	lockFilePerm = 0o600
)

// unexported variables.
var (
	_ FileLocker = primLocker{}
)

// primLocker is the production FileLocker: an exclusive advisory flock via
// the injected syscall primitives. Open-then-flock, unlock-then-close, and
// the unlock-error semantics all live here (ADR-0013; #700). The fd-shaped
// primitives deliberately avoid *os.File (design flag P-3: a dropped
// *os.File's finalizer would close the fd and silently release the lock
// mid-hold).
type primLocker struct {
	prims Primitives
}

// Lock acquires an exclusive flock on path, creating the file if absent.
func (l primLocker) Lock(path string) (func() error, error) {
	lockFD, err := l.prims.OpenLockFile(path, lockFilePerm)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", path, err)
	}

	flockErr := l.prims.FlockExclusive(lockFD)
	if flockErr != nil {
		_ = l.prims.CloseFD(lockFD)

		return nil, fmt.Errorf("flock %s: %w", path, flockErr)
	}

	unlock := func() error {
		unlockErr := l.prims.FlockUnlock(lockFD)
		closeErr := l.prims.CloseFD(lockFD)

		if unlockErr != nil {
			return fmt.Errorf("funlock %s: %w", path, unlockErr)
		}

		if closeErr != nil {
			return fmt.Errorf("close lock %s: %w", path, closeErr)
		}

		return nil
	}

	return unlock, nil
}
