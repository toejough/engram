package lock

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Name is the lock file every stock-writing command takes first.
const Name = ".tarnbrook.lock"

// ErrHeld is returned when another process already holds the lock.
var ErrHeld = errors.New("stock table locked")

// Lock is a held stock-table lock.
type Lock struct{ path string }

// Acquire creates the lock file exclusively, recording this process's pid. If the file exists it
// returns ErrHeld naming the pid that holds it.
func Acquire(dir string) (*Lock, error) {
	path := dir + string(os.PathSeparator) + Name

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, os.ErrExist) {
		held, _ := os.ReadFile(path)
		pid, _ := strconv.Atoi(strings.TrimSpace(string(held)))
		return nil, fmt.Errorf("%w by pid %d", ErrHeld, pid)
	}
	if err != nil {
		return nil, fmt.Errorf("taking %s: %w", Name, err)
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
		return nil, fmt.Errorf("writing %s: %w", Name, err)
	}

	return &Lock{path: path}, nil
}

// Release removes the lock file.
func (l *Lock) Release() error {
	if err := os.Remove(l.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("releasing %s: %w", Name, err)
	}
	return nil
}
