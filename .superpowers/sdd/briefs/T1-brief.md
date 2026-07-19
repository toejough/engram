### Task T1-rework: compose EdgeFS/flock/sink internally from Primitives; thin cmd/engram (REWORK of landed commit de484526)

> **Context:** the original T1 landed at de484526 with the production adapters (`osFS`,
> `flockLocker`, `syncWriter`/`openDebugSink`, `registerForceExit`/`forwardAsPulses`) implemented
> in `cmd/engram` — and `targ check-thin-api` FAILED with 9 non-thin declarations across the 3 new
> cmd files. That layout is REJECTED (user correction 2026-07-19, vault note 303; see "Revised
> composition doctrine" under Design flags). This task REWORKS the landed state: the logic moves
> into `internal/cli` composed from a new `cli.Primitives` carrier of raw capability funcs, the six
> cmd/engram adapter files are deleted, and the relocated integration tests run against the
> composed implementations with REAL os/syscall primitives in internal `_test` files (sanctioned —
> the T-final-1 purity lint excludes `!$test`). What SURVIVES from de484526 unchanged:
> `internal/cli/deps.go` (Deps/EdgeFS/FileLocker, incl. the `//nolint:interfacebloat`), the pure
> `ForceExitOnRepeatedSignal` + its signal_test.go tests, the `SetupSignalHandling` shim +
> `internal/cli/main.go` (both die in T2), and `cmd/engram/main.go`'s single-statement form.

**Files:**
- Create: `internal/cli/primitives.go` (Primitives, WriteSyncer, NewDeps, envOrEmpty)
- Create: `internal/cli/edgefs.go` (primFS — EdgeFS composed from primitives, %w wraps + ADR-0013 atomic dance)
- Create: `internal/cli/locker.go` (primLocker — flock lifecycle over fd primitives)
- Create: `internal/cli/debugsink.go` (openDebugSink, syncWriter, debugLogEnvVar/debugLogPerm)
- Create: `internal/cli/primitives_test.go`, `internal/cli/edgefs_test.go`, `internal/cli/locker_test.go` (unit, fake primitives)
- Create: `internal/cli/primitives_integration_test.go`, `internal/cli/signal_integration_test.go` (real os/syscall — relocated cmd suites)
- Modify: `internal/cli/signal.go` (add generic `ForwardAsPulses` + `startForceExit`; shim goroutine now calls ForwardAsPulses)
- Modify: `internal/cli/signal_test.go` (add ForwardAsPulses test, chan int)
- Delete: `cmd/engram/os_fs.go`, `cmd/engram/os_fs_test.go`, `cmd/engram/os_signal.go`, `cmd/engram/os_signal_test.go`, `cmd/engram/debuglog_sink.go`, `cmd/engram/debuglog_sink_test.go`
- Unchanged: `internal/cli/deps.go`, `cmd/engram/main.go` (still the 1-statement `cli.Main(...)` — T2 rewrites it), `internal/cli/main.go`

**Interfaces:**
- Consumes: `cli.Deps`/`cli.EdgeFS`/`cli.FileLocker` (landed, unchanged); `cli.ForceExitOnRepeatedSignal`, `cli.ExitCodeSigInt`.
- Produces: `cli.Primitives` (struct — exact fields in the doctrine subsection, consume verbatim); `type WriteSyncer interface { io.Writer; Sync() error }`; `func NewDeps(prims Primitives, stdout, stderr io.Writer, exit func(int)) Deps`; `func ForwardAsPulses[T any](in <-chan T, out chan<- struct{})`; unexported `primFS`, `primLocker`, `openDebugSink`, `syncWriter`, `startForceExit`, `envOrEmpty`, consts `lockFilePerm`/`debugLogPerm`/`debugLogEnvVar`/`maxTempAttempts`, sentinel `errTempNameExhausted`.

**Steps:**

1. [ ] RED — create the unit-test files against the not-yet-existing composition seams. All three compile-fail (`undefined: cli.Primitives`, `undefined: cli.NewDeps`) — that is the RED; the behaviors they pin (wrap-with-%w, temp-cleanup-on-failure, unique-temp-name collision retry, fd-close-on-flock-failure, per-write Sync, force-exit watcher registration) were UNTESTABLE against the real-os cmd adapters, which is exactly the seam this rework buys.

   `internal/cli/primitives_test.go`:

```go
package cli_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestNewDeps_ComposesCarrierFromPrimitives(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var stdout, stderr bytes.Buffer

	exitCodes := make([]int, 0, 1)
	fixed := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)

	prims := cli.Primitives{
		Getenv:      func(string) string { return "" },
		Now:         func() time.Time { return fixed },
		Getwd:       func() (string, error) { return "/work", nil },
		UserHomeDir: func() (string, error) { return "/home/x", nil },
	}

	deps := cli.NewDeps(prims, &stdout, &stderr, func(code int) { exitCodes = append(exitCodes, code) })

	g.Expect(deps.Stdout).To(gomega.BeIdenticalTo(&stdout))
	g.Expect(deps.Stderr).To(gomega.BeIdenticalTo(&stderr))
	g.Expect(deps.Now()).To(gomega.Equal(fixed))

	wd, err := deps.Getwd()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(wd).To(gomega.Equal("/work"))

	home, homeErr := deps.UserHomeDir()
	g.Expect(homeErr).NotTo(gomega.HaveOccurred())
	g.Expect(home).To(gomega.Equal("/home/x"))

	g.Expect(deps.FS).NotTo(gomega.BeNil())
	g.Expect(deps.Lock).NotTo(gomega.BeNil())

	deps.Exit(3)
	g.Expect(exitCodes).To(gomega.Equal([]int{3}))
}

func TestNewDeps_DebugSinkEmptyEnvOrFailedOpenIsNil(t *testing.T) {
	t.Parallel()

	t.Run("empty env yields nil sink without opening", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		prims := cli.Primitives{
			Getenv: func(string) string { return "" },
			OpenDebugFile: func(string, fs.FileMode) (cli.WriteSyncer, error) {
				t.Error("open must not be called for an empty path")

				return nil, nil
			},
		}

		g.Expect(cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).DebugLog).To(gomega.BeNil())
	})

	t.Run("failed open yields nil sink", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		prims := cli.Primitives{
			Getenv: func(string) string { return "/nope/debug.log" },
			OpenDebugFile: func(string, fs.FileMode) (cli.WriteSyncer, error) {
				return nil, errors.New("open failed")
			},
		}

		g.Expect(cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).DebugLog).To(gomega.BeNil())
	})
}

func TestNewDeps_DebugSinkSyncsEveryWrite(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	sink := &recordingSyncer{}
	prims := cli.Primitives{
		Getenv: func(key string) string {
			if key == "ENGRAM_DEBUG_LOG" {
				return "/dev/fake/debug.log"
			}

			return ""
		},
		OpenDebugFile: func(path string, _ fs.FileMode) (cli.WriteSyncer, error) {
			g.Expect(path).To(gomega.Equal("/dev/fake/debug.log"))

			return sink, nil
		},
	}

	deps := cli.NewDeps(prims, io.Discard, io.Discard, func(int) {})
	g.Expect(deps.DebugLog).NotTo(gomega.BeNil())

	if deps.DebugLog == nil {
		return
	}

	_, err := deps.DebugLog.Write([]byte("line one\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	_, err = deps.DebugLog.Write([]byte("line two\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(sink.contents()).To(gomega.Equal("line one\nline two\n"))
	g.Expect(sink.syncCount()).To(gomega.Equal(2), "per-line Sync is the tail -F liveness contract")
}

func TestNewDeps_StartsForceExitWatcherFromPrimitive(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	pulsesCh := make(chan chan<- struct{}, 1)
	exitCodes := make(chan int, 1)

	prims := cli.Primitives{
		StartSignalPulses: func(pulses chan<- struct{}, buffer int) {
			g.Expect(buffer).To(gomega.BeNumerically(">", 0))
			pulsesCh <- pulses
		},
	}

	cli.NewDeps(prims, io.Discard, io.Discard, func(code int) { exitCodes <- code })

	var pulses chan<- struct{}

	select {
	case pulses = <-pulsesCh:
	case <-time.After(time.Second):
		t.Fatal("StartSignalPulses was not invoked by NewDeps")
	}

	pulses <- struct{}{}

	pulses <- struct{}{}

	select {
	case code := <-exitCodes:
		g.Expect(code).To(gomega.Equal(cli.ExitCodeSigInt))
	case <-time.After(time.Second):
		t.Fatal("exit not called after two pulses")
	}
}

func TestNewDeps_ZeroPrimitivesDisablesOptionalEdges(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	deps := cli.NewDeps(cli.Primitives{}, io.Discard, io.Discard, func(int) {})

	g.Expect(deps.DebugLog).To(gomega.BeNil())
}

// recordingSyncer is a fake WriteSyncer that records writes and counts Sync
// calls (safe for concurrent use).
type recordingSyncer struct {
	mu    sync.Mutex
	data  strings.Builder
	syncs int
}

func (r *recordingSyncer) Sync() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.syncs++

	return nil
}

func (r *recordingSyncer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.data.Write(p)
}

func (r *recordingSyncer) contents() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.data.String()
}

func (r *recordingSyncer) syncCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.syncs
}
```

   `internal/cli/edgefs_test.go`:

```go
package cli_test

import (
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestEdgeFS_PreservesSentinelChainsThroughWrapping(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := fsFromPrims(cli.Primitives{
		ReadFile: func(string) ([]byte, error) {
			return nil, &fs.PathError{Op: "open", Path: "x", Err: fs.ErrNotExist}
		},
	})

	_, err := fsys.ReadFile("x")
	g.Expect(err).To(gomega.MatchError(fs.ErrNotExist), "%w wrapping must preserve errors.Is chains")
	g.Expect(err.Error()).To(gomega.ContainSubstring("x"), "wrap must add path context")
}

func TestEdgeFS_WriteFileAtomicFailuresRemoveTemp(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")

	t.Run("rename failure removes the created temp", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var created string

		removed := make([]string, 0, 1)
		prims := cli.Primitives{
			Now: func() time.Time { return time.Unix(0, fakeDanceNanos) },
			WriteFileExcl: func(path string, _ []byte, _ fs.FileMode) error {
				created = path

				return nil
			},
			Rename: func(string, string) error { return boom },
			Remove: func(path string) error {
				removed = append(removed, path)

				return nil
			},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(err.Error()).To(gomega.ContainSubstring("rename"))
		g.Expect(removed).To(gomega.Equal([]string{created}),
			"a failed dance must remove the temp file it created")
	})

	t.Run("chmod failure removes the created temp", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var created string

		removed := make([]string, 0, 1)
		prims := cli.Primitives{
			Now: func() time.Time { return time.Unix(0, fakeDanceNanos) },
			WriteFileExcl: func(path string, _ []byte, _ fs.FileMode) error {
				created = path

				return nil
			},
			Chmod: func(string, fs.FileMode) error { return boom },
			Remove: func(path string) error {
				removed = append(removed, path)

				return nil
			},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(err.Error()).To(gomega.ContainSubstring("chmod"))
		g.Expect(removed).To(gomega.Equal([]string{created}),
			"a failed dance must remove the temp file it created")
	})

	t.Run("exclusive-create failure aborts with nothing to clean", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		prims := cli.Primitives{
			Now:           func() time.Time { return time.Unix(0, fakeDanceNanos) },
			WriteFileExcl: func(string, []byte, fs.FileMode) error { return boom },
			Remove: func(string) error {
				t.Error("nothing was created, so nothing may be removed")

				return nil
			},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(err.Error()).To(gomega.ContainSubstring("create temp"))
	})
}

func TestEdgeFS_WriteFileAtomicHappyPathDance(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	calls := &callRecorder{}
	target := filepath.Join("some", "dir", "note.md")

	fsys := fsFromPrims(cli.Primitives{
		Now: func() time.Time { return time.Unix(0, fakeDanceNanos) },
		WriteFileExcl: func(path string, data []byte, perm fs.FileMode) error {
			g.Expect(filepath.Dir(path)).To(gomega.Equal(filepath.Join("some", "dir")),
				"temp must be created in the target's dir — same-directory rename is the ADR-0013 primitive")
			g.Expect(filepath.Base(path)).To(gomega.Equal(".note.md.tmp-12345-0"),
				"candidate names derive from target base + clock nanos + attempt counter (P-4)")
			g.Expect(string(data)).To(gomega.Equal("v2"), "the data lands in the exclusive create itself")
			g.Expect(perm).To(gomega.Equal(atomicPerm), "the target perm reaches the exclusive create")
			calls.add("writeexcl " + filepath.Base(path))

			return nil
		},
		Chmod: func(path string, perm fs.FileMode) error {
			g.Expect(perm).To(gomega.Equal(atomicPerm),
				"chmod must force the EXACT target perm regardless of umask")
			calls.add("chmod " + filepath.Base(path))

			return nil
		},
		Rename: func(oldPath, newPath string) error {
			calls.add("rename " + filepath.Base(oldPath) + "->" + filepath.Base(newPath))

			return nil
		},
		Remove: func(path string) error {
			calls.add("remove " + filepath.Base(path))

			return nil
		},
	})

	g.Expect(fsys.WriteFileAtomic(target, []byte("v2"), atomicPerm)).To(gomega.Succeed())
	g.Expect(calls.list()).To(gomega.Equal([]string{
		"writeexcl .note.md.tmp-12345-0",
		"chmod .note.md.tmp-12345-0",
		"rename .note.md.tmp-12345-0->note.md",
	}), "success path must not remove the renamed file")
}

func TestEdgeFS_WriteFileAtomicUniqueNameRetry(t *testing.T) {
	t.Parallel()

	t.Run("collision retries a fresh candidate then succeeds", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		target := filepath.Join("some", "dir", "note.md")
		tried := make([]string, 0, 2)

		var renamed string

		prims := cli.Primitives{
			Now: func() time.Time { return time.Unix(0, fakeDanceNanos) },
			WriteFileExcl: func(path string, _ []byte, _ fs.FileMode) error {
				tried = append(tried, path)
				if len(tried) == 1 {
					return &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
				}

				return nil
			},
			Rename: func(oldPath, _ string) error {
				renamed = oldPath

				return nil
			},
			Remove: func(string) error {
				t.Error("a colliding candidate was not created by the dance and must not be removed")

				return nil
			},
		}

		g.Expect(fsFromPrims(prims).WriteFileAtomic(target, []byte("v2"), atomicPerm)).To(gomega.Succeed())
		g.Expect(tried).To(gomega.HaveLen(2))
		g.Expect(tried[0]).NotTo(gomega.Equal(tried[1]), "each retry must try a FRESH candidate name")
		g.Expect(renamed).To(gomega.Equal(tried[1]), "the created candidate is the one renamed into place")
	})

	t.Run("exhausted candidates yield a bounded wrapped error", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		attempts := 0
		prims := cli.Primitives{
			Now: func() time.Time { return time.Unix(0, fakeDanceNanos) },
			WriteFileExcl: func(path string, _ []byte, _ fs.FileMode) error {
				attempts++

				return &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
			},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(fs.ErrExist), "the last collision stays in the error chain")
		g.Expect(err.Error()).To(gomega.ContainSubstring("create temp"))
		g.Expect(err.Error()).To(gomega.ContainSubstring("attempts"))
		g.Expect(attempts).To(gomega.Equal(danceMaxAttempts), "the retry loop must be BOUNDED")
	})
}

// fsFromPrims composes the production EdgeFS from fake primitives via the
// public composition root.
func fsFromPrims(prims cli.Primitives) cli.EdgeFS {
	return cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).FS
}

// callRecorder records call labels in order (single-goroutine use).
type callRecorder struct{ calls []string }

func (c *callRecorder) add(call string) { c.calls = append(c.calls, call) }

func (c *callRecorder) list() []string { return c.calls }

// unexported constants.
const (
	atomicPerm fs.FileMode = 0o600

	// danceMaxAttempts mirrors edgefs.go's maxTempAttempts — the spec'd
	// bound on unique-temp-name candidates (doctrine flag P-4).
	danceMaxAttempts = 10

	// fakeDanceNanos is the fixed clock reading the dance fakes inject;
	// candidate temp names embed it.
	fakeDanceNanos = 12345
)
```

   `internal/cli/locker_test.go`:

```go
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
```

   Also modify `internal/cli/signal_test.go` — add (same imports; `cli` already imported):

```go
func TestForwardAsPulses_ForwardsEachValue(t *testing.T) {
	t.Parallel()

	const valueCount = 2

	in := make(chan int, valueCount)
	pulses := make(chan struct{}, valueCount)

	go cli.ForwardAsPulses(in, pulses)

	in <- 1

	in <- 2

	const pulseTimeout = time.Second

	for range valueCount {
		select {
		case <-pulses:
		case <-time.After(pulseTimeout):
			t.Fatal("pulse not forwarded within timeout")
		}
	}
}
```

   Run `targ test` — expect compile failure in internal/cli tests (`undefined: cli.Primitives`, `undefined: cli.NewDeps`, `undefined: cli.ForwardAsPulses`). That is the RED.

2. [ ] GREEN — create the internal composition. `internal/cli/primitives.go` (the Primitives struct is the doctrine subsection's canonical inventory — byte-identical):

```go
package cli

import (
	"io"
	"io/fs"
	"time"
)

// Primitives carries raw impure capabilities as func values. cmd/engram
// populates it with direct references to os/syscall/filepath/time functions,
// single-call closures where a signature must be erased (fd instead of
// *os.File, WriteSyncer instead of *os.File, pulses instead of os.Signal),
// or an enumerated stdlib-equivalent survivor closure (doctrine survivors:
// S-1 WriteFileExcl here; C-1 RunCommand lands in T17).
// ALL composition, error wrapping, and lifecycle logic lives in internal/cli;
// targ check-thin-api enforces that the cmd side stays declaration-free (#700).
type Primitives struct {
	// Filesystem (direct os/filepath references).
	ReadFile  func(path string) ([]byte, error)                      // os.ReadFile
	WriteFile func(path string, data []byte, perm fs.FileMode) error // os.WriteFile
	MkdirAll  func(path string, perm fs.FileMode) error              // os.MkdirAll
	MkdirTemp func(dir, pattern string) (string, error)              // os.MkdirTemp
	Stat      func(path string) (fs.FileInfo, error)                 // os.Stat
	ReadDir   func(path string) ([]fs.DirEntry, error)               // os.ReadDir
	Remove    func(path string) error                                // os.Remove
	RemoveAll func(path string) error                                // os.RemoveAll
	Rename    func(oldPath, newPath string) error                    // os.Rename
	WalkDir   func(root string, fn fs.WalkDirFunc) error             // filepath.WalkDir
	Chmod     func(path string, mode fs.FileMode) error              // os.Chmod

	// Exclusive create (doctrine survivor S-1 — a stdlib-equivalent
	// primitive closure: os.WriteFile's own body with O_CREATE|O_EXCL;
	// behavior changes extend this SIGNATURE, never the cmd body).
	WriteFileExcl func(path string, data []byte, perm fs.FileMode) error

	// Process, env, clock (direct references).
	Getenv      func(key string) string // os.Getenv
	Now         func() time.Time        // time.Now
	Getwd       func() (string, error)  // os.Getwd
	UserHomeDir func() (string, error)  // os.UserHomeDir

	// Advisory file locking (single-syscall closures; lifecycle internal —
	// design flags P-2/P-3: semantic per-op funcs over a raw uintptr fd,
	// via syscall.Open, never os.OpenFile().Fd()).
	OpenLockFile   func(path string, perm fs.FileMode) (uintptr, error) // syscall.Open O_CREAT|O_RDWR
	FlockExclusive func(fd uintptr) error                               // syscall.Flock LOCK_EX
	FlockUnlock    func(fd uintptr) error                               // syscall.Flock LOCK_UN
	CloseFD        func(fd uintptr) error                               // syscall.Close

	// Debug sink (single-call closure; empty-path branch + sync policy internal).
	OpenDebugFile func(path string, perm fs.FileMode) (WriteSyncer, error) // os.OpenFile O_APPEND|O_CREATE|O_WRONLY

	// Signal (single-purpose starter closure; pulse forwarding is internal
	// via ForwardAsPulses; buffer/pulse-channel/force-exit policy internal).
	StartSignalPulses func(pulses chan<- struct{}, buffer int)
}

// WriteSyncer is the debug-sink capability surface (*os.File satisfies it).
type WriteSyncer interface {
	io.Writer
	Sync() error
}

// NewDeps composes the production Deps carrier from raw primitives: the
// EdgeFS implementation (contextual %w wrapping + the ADR-0013 atomic-write
// dance), the flock lifecycle, the debug sink (ENGRAM_DEBUG_LOG; empty path
// or failed open → nil → no-op logger), and the repeated-signal force-exit
// watcher. cmd/engram calls this exactly once from main(); tests call it
// with fake primitives to unit-test the composition (#700).
func NewDeps(prims Primitives, stdout, stderr io.Writer, exit func(int)) Deps {
	startForceExit(prims, exit)

	return Deps{
		Stdout:      stdout,
		Stderr:      stderr,
		Exit:        exit,
		Getenv:      prims.Getenv,
		Now:         prims.Now,
		Getwd:       prims.Getwd,
		UserHomeDir: prims.UserHomeDir,
		FS:          primFS{prims: prims},
		Lock:        primLocker{prims: prims},
		DebugLog:    openDebugSink(envOrEmpty(prims.Getenv, debugLogEnvVar), prims.OpenDebugFile),
	}
}

// envOrEmpty reads key via getenv, tolerating a nil (unwired) capability.
func envOrEmpty(getenv func(string) string, key string) string {
	if getenv == nil {
		return ""
	}

	return getenv(key)
}
```

   `internal/cli/edgefs.go` — the landed cmd/engram/os_fs.go `osFS` logic verbatim with `os.X`/`filepath.WalkDir` swapped for `f.prims.X`, PLUS the atomic-write dance re-sequenced for internal unique-temp-name generation over the exclusive-create `WriteFileExcl` primitive (design flags P-4/S-1 — same-directory rename atomicity unchanged):

```go
package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// Compile-time interface conformance (internal — the thin-api checker does
// not walk internal/).
var _ EdgeFS = primFS{}

// primFS is the production EdgeFS: it composes the injected raw primitives
// with contextual error wrapping (%w preserves errors.Is chains such as
// fs.ErrNotExist) and the ADR-0013 atomic-write dance. All orchestration
// lives here in internal/; cmd/engram contributes only raw os/filepath
// references (#700).
type primFS struct {
	prims Primitives
}

// MkdirAll creates path with any missing parents; no-op when path exists.
func (f primFS) MkdirAll(path string, perm fs.FileMode) error {
	err := f.prims.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}

	return nil
}

// MkdirTemp creates a fresh unique directory in dir matching pattern.
func (f primFS) MkdirTemp(dir, pattern string) (string, error) {
	made, err := f.prims.MkdirTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp in %s: %w", dir, err)
	}

	return made, nil
}

// ReadDir returns the directory entries of path.
func (f primFS) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := f.prims.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", path, err)
	}

	return entries, nil
}

// ReadFile reads the file at path.
func (f primFS) ReadFile(path string) ([]byte, error) {
	data, err := f.prims.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}

// Remove deletes the file or empty directory at path.
func (f primFS) Remove(path string) error {
	err := f.prims.Remove(path)
	if err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}

	return nil
}

// RemoveAll deletes path and any children; no-op when path is absent.
func (f primFS) RemoveAll(path string) error {
	err := f.prims.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("remove all %s: %w", path, err)
	}

	return nil
}

// Rename atomically renames oldPath to newPath (same-directory renames are
// atomic on POSIX — the ADR-0013 primitive).
func (f primFS) Rename(oldPath, newPath string) error {
	err := f.prims.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("rename %s -> %s: %w", oldPath, newPath, err)
	}

	return nil
}

// Stat returns the fs.FileInfo for path.
func (f primFS) Stat(path string) (fs.FileInfo, error) {
	info, err := f.prims.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	return info, nil
}

// WalkDir walks the file tree rooted at root, calling fn for each entry.
func (f primFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	err := f.prims.WalkDir(root, fn)
	if err != nil {
		return fmt.Errorf("walk %s: %w", root, err)
	}

	return nil
}

// WriteFile writes data to path with perm.
func (f primFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	err := f.prims.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// WriteFileAtomic writes data to path atomically: it derives a unique temp
// name in filepath.Dir(path) from the target base + the injected clock's
// nanos + an attempt counter, creates it exclusively (data written at perm)
// via the WriteFileExcl primitive — retrying fresh candidates on
// fs.ErrExist, bounded by maxTempAttempts — then chmods the temp to the
// exact target perm (umask-independent) and renames into place. A
// same-directory rename is atomic on POSIX — a concurrent reader sees
// either the old or the new file, never a partial one. On any failure
// after creation the temp file is removed and the original (if any) is
// left untouched (ADR-0013; design flag P-4: the unique-temp-name policy
// is INTERNAL — cmd contributes only the stdlib-equivalent WriteFileExcl
// primitive, doctrine survivor S-1, plus the restored direct Chmod
// primitive for umask-independent perms).
func (f primFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	tmpName, err := f.createUniqueTemp(path, data, perm)
	if err != nil {
		return fmt.Errorf("atomic write %s: create temp: %w", path, err)
	}

	// chmod after write (temp is never wider than final); explicit chmod
	// keeps atomic-write perms umask-independent — parity with the
	// pre-#700 dance. Do NOT reorder chmod before the data write.
	chmodErr := f.prims.Chmod(tmpName, perm)
	if chmodErr != nil {
		_ = f.prims.Remove(tmpName)

		return fmt.Errorf("atomic write %s: chmod temp: %w", path, chmodErr)
	}

	renameErr := f.prims.Rename(tmpName, path)
	if renameErr != nil {
		// Cleanup on any failure after creation (P-4).
		_ = f.prims.Remove(tmpName)

		return fmt.Errorf("atomic write %s: rename: %w", path, renameErr)
	}

	return nil
}

// createUniqueTemp writes data exclusively to a fresh candidate temp name
// beside path (".<base>.tmp-<nanos>-<attempt>"). A candidate that already
// exists (fs.ErrExist) is retried with the next attempt counter, bounded
// by maxTempAttempts; any other error aborts immediately — nothing was
// created, so there is nothing to clean.
func (f primFS) createUniqueTemp(path string, data []byte, perm fs.FileMode) (string, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	nanos := f.prims.Now().UnixNano()

	var lastErr error

	for attempt := range maxTempAttempts {
		candidate := filepath.Join(dir, fmt.Sprintf(".%s.tmp-%d-%d", base, nanos, attempt))

		lastErr = f.prims.WriteFileExcl(candidate, data, perm)
		if lastErr == nil {
			return candidate, nil
		}

		if !errors.Is(lastErr, fs.ErrExist) {
			return "", lastErr
		}
	}

	return "", fmt.Errorf("%w after %d attempts: %w", errTempNameExhausted, maxTempAttempts, lastErr)
}

// unexported variables.
var errTempNameExhausted = errors.New("no unique temp name available")

// unexported constants.
const (
	// maxTempAttempts bounds the fs.ErrExist retry when deriving a unique
	// temp name for the atomic-write dance (doctrine flag P-4).
	maxTempAttempts = 10
)
```

   `internal/cli/locker.go` — the landed `flockLocker.Lock` lifecycle verbatim over the fd primitives:

```go
package cli

import "fmt"

// Compile-time interface conformance.
var _ FileLocker = primLocker{}

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
	fd, err := l.prims.OpenLockFile(path, lockFilePerm)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", path, err)
	}

	flockErr := l.prims.FlockExclusive(fd)
	if flockErr != nil {
		_ = l.prims.CloseFD(fd)

		return nil, fmt.Errorf("flock %s: %w", path, flockErr)
	}

	unlock := func() error {
		unlockErr := l.prims.FlockUnlock(fd)
		closeErr := l.prims.CloseFD(fd)

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

// unexported constants.
const (
	lockFilePerm = 0o600
)
```

   `internal/cli/debugsink.go` — the landed cmd sink logic, parameterized over the open primitive (perm policy internal, design flag P-1):

```go
package cli

import (
	"fmt"
	"io"
	"io/fs"
)

// openDebugSink builds the debug-log sink: nil for an empty path, an
// unwired open capability, or a failed open — debuglog treats a nil writer
// as "logging disabled", so the CLI still runs (pre-#700 behavior
// preserved). Otherwise every write is followed by Sync so `tail -F` shows
// progress live.
func openDebugSink(path string, open func(string, fs.FileMode) (WriteSyncer, error)) io.Writer {
	if path == "" || open == nil {
		return nil
	}

	file, err := open(path, debugLogPerm)
	if err != nil {
		return nil
	}

	return &syncWriter{file: file}
}

// syncWriter flushes after every write. debuglog is documented tail -F
// friendly; the Logger sees only an io.Writer, so the per-line sync lives
// here in the composed sink.
type syncWriter struct {
	file WriteSyncer
}

// Write appends p and syncs the underlying sink.
func (w *syncWriter) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	if err != nil {
		return n, fmt.Errorf("debug log write: %w", err)
	}

	_ = w.file.Sync()

	return n, nil
}

// unexported constants.
const (
	debugLogEnvVar             = "ENGRAM_DEBUG_LOG"
	debugLogPerm   fs.FileMode = 0o644
)
```

   Modify `internal/cli/signal.go` — add `ForwardAsPulses` + `startForceExit`, and point the shim's goroutine at the generic (full replacement file):

```go
package cli

import (
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/toejough/engram/internal/debuglog"
)

// Exported constants.
const (
	ExitCodeSigInt = 130
)

// ForceExitOnRepeatedSignal starts a goroutine that waits for two pulses
// on the given channel, then calls exitFn. The first pulse allows graceful
// shutdown; the second forces immediate exit. Pulses are pure struct{}
// units — cmd/engram adapts real os.Signal deliveries into them, so no
// os.Signal type crosses into internal/ (#700).
func ForceExitOnRepeatedSignal(signals <-chan struct{}, exitFn func(int)) {
	var signalCount atomic.Int32

	go func() {
		for range signals {
			count := signalCount.Add(1)
			if count >= secondSignal {
				// Second signal or later: force exit immediately
				exitFn(ExitCodeSigInt)

				return
			}
			// First signal: will be handled by targ's context cancellation
		}
	}()
}

// ForwardAsPulses forwards each value received on in as a unit pulse on
// out. It is generic so cmd/engram can feed a chan os.Signal without
// os.Signal entering any internal signature (#700); tests drive it with a
// chan int.
func ForwardAsPulses[T any](in <-chan T, out chan<- struct{}) {
	for range in {
		out <- struct{}{}
	}
}

// SetupSignalHandling registers signal handlers and starts the force-exit goroutine.
// Returns the configured targets for targ.Main.
//
// Deprecated: interim shim only — deleted by the #700 wiring task; cmd/engram
// wires signal pulses through cli.Primitives.StartSignalPulses instead.
func SetupSignalHandling(
	stdout, stderr io.Writer,
	exitFn func(int),
	logger *debuglog.Logger,
) []any {
	sigCh := make(chan os.Signal, signalChannelBuffer)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	pulses := make(chan struct{}, signalChannelBuffer)

	go ForwardAsPulses(sigCh, pulses)

	ForceExitOnRepeatedSignal(pulses, exitFn)

	return Targets(stdout, stderr, exitFn, logger)
}

// startForceExit starts the repeated-signal force-exit watcher from the
// injected starter primitive. A nil primitive or exit func (minimal test
// Deps) skips registration. The pulse channel, buffer size, and force-exit
// policy live here — cmd only subscribes and forwards (#700).
func startForceExit(prims Primitives, exit func(int)) {
	if prims.StartSignalPulses == nil || exit == nil {
		return
	}

	pulses := make(chan struct{}, signalChannelBuffer)
	prims.StartSignalPulses(pulses, signalChannelBuffer)
	ForceExitOnRepeatedSignal(pulses, exit)
}

// unexported constants.
const (
	secondSignal        = 2  // Force exit on second signal
	signalChannelBuffer = 10 // Buffer size for signal + pulse channels
)
```

   Run `targ test` — expect green (unit tests pass; existing suite untouched; shim behavior identical through ForwardAsPulses).

3. [ ] Integration tests — relocate the cmd/engram adapter suites into internal `_test` files running the COMPOSED implementations over REAL os/syscall primitives (this preserves the ADR-0013 regression coverage the cmd tests carried; test files are exempt from the T-final-1 purity lint by design). Create `internal/cli/primitives_integration_test.go`:

```go
package cli_test

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// This file exercises the internally-composed EdgeFS/FileLocker/debug-sink
// implementations over REAL os/syscall primitives — the relocated
// cmd/engram adapter integration suite (#700 rework). realPrimitives()
// mirrors cmd/engram/main.go's Primitives literal (doctrine flag DRIFT:
// cli_test.go's end-to-end binary tests guard the production literal).

func TestRealDebugSink_AppendsAcrossOpens(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := filepath.Join(t.TempDir(), "debug.log")

	first := debugSinkAt(path)
	g.Expect(first).NotTo(gomega.BeNil())

	if first == nil {
		return
	}

	_, err := first.Write([]byte("line one\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Re-open the same path: append mode must preserve the first line —
	// the tail -F contract debuglog documents.
	second := debugSinkAt(path)
	g.Expect(second).NotTo(gomega.BeNil())

	if second == nil {
		return
	}

	_, err = second.Write([]byte("line two\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	contents, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(contents)).To(gomega.Equal("line one\nline two\n"))
}

func TestRealDebugSink_UnopenablePathYieldsNilSink(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Parent is a regular file, so opening a child path fails -> nil sink
	// (the CLI must run without debug logging rather than fail).
	dir := t.TempDir()
	blocked := filepath.Join(dir, "isfile")
	g.Expect(os.WriteFile(blocked, []byte("x"), realFSFilePerm)).To(gomega.Succeed())

	g.Expect(debugSinkAt(filepath.Join(blocked, "debug.log"))).To(gomega.BeNil())
}

func TestRealEdgeFS_MkdirAllStatReadDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b")
	fsys := realFSForTest()

	g.Expect(fsys.MkdirAll(nested, realFSDirPerm)).To(gomega.Succeed())

	info, err := fsys.Stat(nested)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(info.IsDir()).To(gomega.BeTrue())

	entries, readErr := fsys.ReadDir(filepath.Join(dir, "a"))
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Name()).To(gomega.Equal("b"))
}

func TestRealEdgeFS_MkdirTempAndWalkDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := realFSForTest()

	tmpDir, err := fsys.MkdirTemp(dir, "pat-*")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(tmpDir)).To(gomega.HavePrefix("pat-"))
	g.Expect(fsys.WriteFile(filepath.Join(tmpDir, "leaf.txt"), []byte("x"), realFSFilePerm)).To(gomega.Succeed())

	visited := make([]string, 0, 3)
	walkErr := fsys.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		visited = append(visited, path)

		return nil
	})
	g.Expect(walkErr).NotTo(gomega.HaveOccurred())
	g.Expect(visited).To(gomega.ContainElement(filepath.Join(tmpDir, "leaf.txt")))
}

func TestRealEdgeFS_ReadFileMissingSatisfiesErrNotExist(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := realFSForTest()

	_, err := fsys.ReadFile(filepath.Join(t.TempDir(), "missing.txt"))
	g.Expect(err).To(gomega.MatchError(fs.ErrNotExist))
}

func TestRealEdgeFS_ReadWriteRoundTrip(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFile(path, []byte("hello"), realFSFilePerm)).To(gomega.Succeed())

	data, err := fsys.ReadFile(path)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("hello"))
}

func TestRealEdgeFS_RenameRemoveRemoveAll(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := realFSForTest()

	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")

	g.Expect(fsys.WriteFile(oldPath, []byte("x"), realFSFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.Rename(oldPath, newPath)).To(gomega.Succeed())
	g.Expect(newPath).To(gomega.BeAnExistingFile())
	g.Expect(oldPath).NotTo(gomega.BeAnExistingFile())

	g.Expect(fsys.Remove(newPath)).To(gomega.Succeed())
	g.Expect(newPath).NotTo(gomega.BeAnExistingFile())

	sub := filepath.Join(dir, "sub")
	g.Expect(fsys.MkdirAll(sub, realFSDirPerm)).To(gomega.Succeed())
	g.Expect(fsys.WriteFile(filepath.Join(sub, "f"), []byte("x"), realFSFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.RemoveAll(sub)).To(gomega.Succeed())
	g.Expect(sub).NotTo(gomega.BeADirectory())
}

func TestRealEdgeFS_WriteFileAtomicReplacesContentAndCleansTemp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFile(path, []byte("v1"), realFSFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.WriteFileAtomic(path, []byte("v2"), realFSFilePerm)).To(gomega.Succeed())

	data, err := fsys.ReadFile(path)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("v2"))

	entries, readErr := fsys.ReadDir(dir)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1), "temp files must be renamed or removed")
}

// TestRealEdgeFS_WriteFileAtomicPermsAreUmaskIndependent proves the restored
// Chmod step (P-4) makes WriteFileAtomic's final perm exact regardless of
// the process umask — parity with the pre-#700 dance.
func TestRealEdgeFS_WriteFileAtomicPermsAreUmaskIndependent(t *testing.T) {
	// serial: syscall.Umask is process-global; parallel file-creating tests would flake
	g := gomega.NewWithT(t)

	old := syscall.Umask(umaskParityRestrictiveMask)
	defer syscall.Umask(old)

	dir := t.TempDir()
	target := filepath.Join(dir, "note.md")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFileAtomic(target, []byte("v1"), umaskParityPerm)).To(gomega.Succeed())

	info, err := os.Stat(target)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(info.Mode().Perm()).To(gomega.Equal(umaskParityPerm))
}

// TestRealFlockLocker_SecondLockWaitsForUnlock is the relocated ADR-0013
// lock regression guard: a second locker on the same path must block until
// the first unlocks — never proceed concurrently, never fail.
func TestRealFlockLocker_SecondLockWaitsForUnlock(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := realDepsForTest().Lock

	unlock, err := locker.Lock(lockPath)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	acquired := make(chan struct{})

	go func() {
		secondUnlock, secondErr := locker.Lock(lockPath)
		if secondErr == nil {
			_ = secondUnlock()
		}

		close(acquired)
	}()

	const holdWindow = 100 * time.Millisecond

	select {
	case <-acquired:
		t.Fatal("second locker acquired while first still held the lock")
	case <-time.After(holdWindow):
		// good — second locker is blocked while the lock is held
	}

	g.Expect(unlock()).To(gomega.Succeed())

	const releaseTimeout = 2 * time.Second

	select {
	case <-acquired:
		// good — released lock was acquired
	case <-time.After(releaseTimeout):
		t.Fatal("second locker did not acquire after unlock")
	}
}

// debugSinkAt composes a real debug sink for path via NewDeps (Getenv fake
// points ENGRAM_DEBUG_LOG at path; the open primitive is real).
func debugSinkAt(path string) io.Writer {
	prims := realPrimitives()
	prims.Getenv = func(key string) string {
		if key == "ENGRAM_DEBUG_LOG" {
			return path
		}

		return ""
	}

	return cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).DebugLog
}

// realDepsForTest composes production Deps over real OS primitives.
func realDepsForTest() cli.Deps {
	return cli.NewDeps(realPrimitives(), io.Discard, io.Discard, func(int) {})
}

// realFSForTest composes the production EdgeFS over real OS primitives.
func realFSForTest() cli.EdgeFS {
	return realDepsForTest().FS
}

// realPrimitives mirrors cmd/engram/main.go's production Primitives literal
// (minus the signal starter — tests must not subscribe process signals).
func realPrimitives() cli.Primitives {
	return cli.Primitives{
		ReadFile:    os.ReadFile,
		WriteFile:   os.WriteFile,
		MkdirAll:    os.MkdirAll,
		MkdirTemp:   os.MkdirTemp,
		Stat:        os.Stat,
		ReadDir:     os.ReadDir,
		Remove:      os.Remove,
		RemoveAll:   os.RemoveAll,
		Rename:      os.Rename,
		WalkDir:     filepath.WalkDir,
		Chmod:       os.Chmod,
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
		WriteFileExcl: func(path string, data []byte, perm fs.FileMode) error {
			//nolint:gosec // test helper, path from test
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
			if err != nil {
				return err
			}

			_, err = file.Write(data)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}

			return err
		},
		OpenLockFile: func(path string, perm fs.FileMode) (uintptr, error) {
			fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR, uint32(perm))

			return uintptr(fd), err
		},
		FlockExclusive: func(fd uintptr) error {
			return syscall.Flock(int(fd), syscall.LOCK_EX)
		},
		FlockUnlock: func(fd uintptr) error {
			return syscall.Flock(int(fd), syscall.LOCK_UN)
		},
		CloseFD: func(fd uintptr) error {
			return syscall.Close(int(fd))
		},
		OpenDebugFile: func(path string, perm fs.FileMode) (cli.WriteSyncer, error) {
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
		},
	}
}

// unexported constants.
const (
	realFSDirPerm  fs.FileMode = 0o750
	realFSFilePerm fs.FileMode = 0o600

	// umaskParityPerm is the target perm for the umask-independence parity
	// test (P-4 restored chmod step).
	umaskParityPerm fs.FileMode = 0o644

	// umaskParityRestrictiveMask is a deliberately restrictive umask (would
	// mask 0o644 down to 0o600 without the explicit chmod step).
	umaskParityRestrictiveMask = 0o077
)
```

   And `internal/cli/signal_integration_test.go` (relocated real-signal test):

```go
package cli_test

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestForceExit_RealSignalDeliveryThroughForwardAsPulses replicates
// cmd/engram/main.go's StartSignalPulses closure with SIGUSR2 (registered
// only by this test; Notify overrides the default terminate disposition, so
// the test run is safe) and proves the full chain: real signal -> Notify ->
// ForwardAsPulses -> ForceExitOnRepeatedSignal. Pending same-signal
// coalescing can swallow a rapid second delivery, so the test keeps
// signalling (paced) until exit fires — over-delivery still means "force
// exit", so this can never false-pass.
func TestForceExit_RealSignalDeliveryThroughForwardAsPulses(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const buffer = 10

	exitCodes := make(chan int, 1)

	sigCh := make(chan os.Signal, buffer)
	signal.Notify(sigCh, syscall.SIGUSR2)

	pulses := make(chan struct{}, buffer)

	go cli.ForwardAsPulses(sigCh, pulses)
	cli.ForceExitOnRepeatedSignal(pulses, func(code int) { exitCodes <- code })

	proc, err := os.FindProcess(os.Getpid())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	const (
		pace    = 20 * time.Millisecond
		maxWait = 5 * time.Second
	)

	deadline := time.After(maxWait)

	for {
		g.Expect(proc.Signal(syscall.SIGUSR2)).To(gomega.Succeed())

		select {
		case code := <-exitCodes:
			g.Expect(code).To(gomega.Equal(cli.ExitCodeSigInt))

			return
		case <-deadline:
			t.Fatal("force-exit not fired within timeout")
		case <-time.After(pace):
		}
	}
}
```

   Run `targ test` — expect green (the relocated suites pass against the composed implementations).

4. [ ] Thin cmd — delete all six cmd/engram adapter files (their logic now lives in internal/cli; their tests were relocated in steps 1 and 3):

```
git rm cmd/engram/os_fs.go cmd/engram/os_fs_test.go \
       cmd/engram/os_signal.go cmd/engram/os_signal_test.go \
       cmd/engram/debuglog_sink.go cmd/engram/debuglog_sink_test.go
```

   `cmd/engram/main.go` is untouched — still the single-statement `cli.Main(os.Stdout, os.Stderr, os.Exit)` (T2 rewrites it over NewDeps). Run `targ test` — expect green (cli_test.go's end-to-end binary tests still build and run the real binary).

5. [ ] Gate — run `targ check-thin-api`. Expect PASS: `All N public API files are thin wrappers.` — cmd/engram now contains only main.go (one single-external-call statement, zero other declarations). If ANY finding remains, escalate the exact finding to the orchestrator (do not suppress, do not restructure ad hoc — doctrine flag SIG-1 documents the checker's exact rules). Then run `targ check-full` — the ONLY tolerated new-vs-baseline residual is the KNOWN `SetupSignalHandling` 0% function-coverage gap (T1-report concern #1; resolved when T2 deletes the shim). Run `targ reorder-decls` if `reorder-decls-check` flags the new files (revert any out-of-scope `dev/eval/**/testdata` touches before committing, as the landed T1 did). Any OTHER new failure: fix before commit.

6. [ ] Commit:

```
refactor(cli): compose I/O adapters internally from Primitives (#700)

Rework of the landed T1 (de484526): targ check-thin-api rejected real
adapter logic in cmd/engram (9 non-thin declarations). EdgeFS wrapping +
the ADR-0013 atomic-write dance, the flock lifecycle, and the syncing
debug sink move behind internal/cli composition (cli.Primitives +
cli.NewDeps), unit-tested with fake primitives and integration-tested
with real os/syscall funcs in internal _test files. Adds generic
ForwardAsPulses; deletes all six cmd/engram adapter files. cmd keeps its
single-statement main until T2 wires NewDeps.

AI-Used: [claude]
```

---

