package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestEngramServerSet_HardErrorsEveryCommand covers vault-local-first's
// "ENGRAM_SERVER SHALL be a hard error" (design D1): with ENGRAM_SERVER set,
// every subcommand — serve included — exits non-zero with a message naming
// ENGRAM_PARENT and the same URL, before any filesystem, lock, fetch or
// bind capability is touched.
func TestEngramServerSet_HardErrorsEveryCommand(t *testing.T) {
	t.Parallel()

	const serverURL = "http://host:8093"

	for _, args := range everySubcommandArgs() {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			rec := &capabilityRecorder{}

			stderr, exitCodes := runWithRemovedServer(rec, serverURL, args)

			g.Expect(exitCodes).To(ContainElement(1), "must exit non-zero")
			g.Expect(stderr).To(ContainSubstring("ENGRAM_SERVER is no longer supported"))
			g.Expect(stderr).To(ContainSubstring("ENGRAM_PARENT=" + serverURL))
			g.Expect(rec.fsCalls.Load()).To(BeZero(), "no filesystem access")
			g.Expect(rec.lockCalls.Load()).To(BeZero(), "no lock access")
			g.Expect(rec.fetchCalls.Load()).To(BeZero(), "no network fetch")
			g.Expect(rec.bindCalls.Load()).To(BeZero(), "no mux, route or bind")
			g.Expect(rec.execCalls.Load()).To(BeZero(), "no external command")
		})
	}
}

// TestEngramServerSet_NeverFetchesAsAlias covers "ENGRAM_SERVER is not
// honored as an alias": for any non-empty ENGRAM_SERVER value (and no
// ENGRAM_PARENT), no subcommand ever issues a fetch, and the refusal always
// names ENGRAM_PARENT with that exact value.
func TestEngramServerSet_NeverFetchesAsAlias(t *testing.T) {
	t.Parallel()

	commands := everySubcommandArgs()

	rapid.Check(t, func(rt *rapid.T) {
		serverURL := rapid.StringMatching(`https?://[a-z0-9.-]{1,20}(:[0-9]{1,5})?/?`).Draw(rt, "serverURL")
		args := rapid.SampledFrom(commands).Draw(rt, "args")

		rec := &capabilityRecorder{}

		stderr, exitCodes := runWithRemovedServer(rec, serverURL, args)

		if rec.fetchCalls.Load() != 0 {
			rt.Fatalf("%v with ENGRAM_SERVER=%q fetched %d time(s)", args, serverURL, rec.fetchCalls.Load())
		}

		if !strings.Contains(stderr, "ENGRAM_PARENT="+serverURL) {
			rt.Fatalf("%v: stderr %q does not name ENGRAM_PARENT=%s", args, stderr, serverURL)
		}

		if len(exitCodes) == 0 || exitCodes[0] != 1 {
			rt.Fatalf("%v: exit codes %v, want a first exit(1)", args, exitCodes)
		}
	})
}

// unexported variables.
var (
	errRecorderCapability = errors.New("recorder: capability must not be called")
)

// capabilityRecorder counts every call into a fake capability group, so a
// test can assert the ENGRAM_SERVER guard touched none of them.
type capabilityRecorder struct {
	fsCalls    atomic.Int64
	lockCalls  atomic.Int64
	fetchCalls atomic.Int64
	bindCalls  atomic.Int64
	execCalls  atomic.Int64
}

// primitives returns a Primitives whose every FS/Lock/HTTP/Exec/Spawn
// capability records its call and fails, and whose Getenv reports only
// ENGRAM_SERVER=serverURL.
func (r *capabilityRecorder) primitives(serverURL string) cli.Primitives {
	fsHit := func() error {
		r.fsCalls.Add(1)

		return errRecorderCapability
	}

	return cli.Primitives{
		FS: cli.FSPrims{
			ReadFile:     func(string) ([]byte, error) { return nil, fsHit() },
			WriteFile:    func(string, []byte, fs.FileMode) error { return fsHit() },
			MkdirAll:     func(string, fs.FileMode) error { return fsHit() },
			MkdirTemp:    func(string, string) (string, error) { return "", fsHit() },
			Stat:         func(string) (fs.FileInfo, error) { return nil, fsHit() },
			ReadDir:      func(string) ([]fs.DirEntry, error) { return nil, fsHit() },
			Remove:       func(string) error { return fsHit() },
			RemoveAll:    func(string) error { return fsHit() },
			Rename:       func(string, string) error { return fsHit() },
			WalkDir:      func(string, fs.WalkDirFunc) error { return fsHit() },
			Chmod:        func(string, fs.FileMode) error { return fsHit() },
			OpenFileExcl: func(string, fs.FileMode) (io.WriteCloser, error) { return nil, fsHit() },
			Symlink:      func(string, string) error { return fsHit() },
			Readlink:     func(string) (string, error) { return "", fsHit() },
			Lstat:        func(string) (fs.FileInfo, error) { return nil, fsHit() },
		},
		Lock: cli.LockPrims{
			OpenLockFile: func(string, fs.FileMode) (uintptr, error) {
				r.lockCalls.Add(1)

				return 0, errRecorderCapability
			},
			FlockExclusive: func(uintptr) error { r.lockCalls.Add(1); return errRecorderCapability },
			FlockUnlock:    func(uintptr) error { r.lockCalls.Add(1); return errRecorderCapability },
			CloseFD:        func(uintptr) error { r.lockCalls.Add(1); return errRecorderCapability },
		},
		Exec: cli.ExecPrims{
			RunCommand: func(context.Context, string, string, []string, io.Writer, io.Writer) error {
				r.execCalls.Add(1)

				return errRecorderCapability
			},
		},
		Spawn: cli.SpawnPrims{
			RunInherited: func(string, []string, []string) (int, error) {
				r.execCalls.Add(1)

				return 1, errRecorderCapability
			},
		},
		Proc: cli.ProcPrims{
			Getenv: func(key string) string {
				if key == "ENGRAM_SERVER" {
					return serverURL
				}

				return ""
			},
			Now:         time.Now,
			Getwd:       func() (string, error) { return "/nonexistent/cwd", nil },
			UserHomeDir: func() (string, error) { return "/nonexistent/home", nil },
			Username:    func() (string, error) { return "tester", nil },
			IsTerminal:  func() bool { return false },
		},
		HTTP: cli.HTTPPrims{
			NewServeMux: func() cli.RawServeMux {
				r.bindCalls.Add(1)

				return nil
			},
			RegisterRoute: func(cli.RawServeMux, string, string, cli.ServeHandler) { r.bindCalls.Add(1) },
			ListenAndServe: func(context.Context, cli.RawServeMux, string) error {
				r.bindCalls.Add(1)

				return errRecorderCapability
			},
			Fetch: func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				r.fetchCalls.Add(1)

				return cli.FetchResponse{}, errRecorderCapability
			},
		},
	}
}

// everySubcommandArgs returns one plausible invocation of every engram
// subcommand (serve included), each with its required flags.
func everySubcommandArgs() [][]string {
	return [][]string{
		{"engram", "query", "--phrase", "x"},
		{"engram", "ingest"},
		{"engram", "prune"},
		{"engram", "query-chunks", "--phrase", "x"},
		{"engram", "activate", "--note", "1.a.md"},
		{"engram", "count", "--group-by", "type"},
		{"engram", "check"},
		{"engram", "show", "1"},
		{"engram", "show-chunk", "a.md#b"},
		{"engram", "learn", "fact", "--source", "s", "--situation", "x"},
		{"engram", "learn", "feedback", "--source", "s", "--situation", "x"},
		{"engram", "learn", "runbook", "--source", "s", "--situation", "x", "--done-when", "y"},
		{"engram", "learn", "qa", "--source", "s"},
		{"engram", "update"},
		{"engram", "embed", "apply"},
		{"engram", "embed", "status"},
		{"engram", "resituate", "--target", "1", "--situation", "x"},
		{"engram", "amend", "--target", "1", "--discard"},
		{"engram", "vocab", "bootstrap"},
		{"engram", "vocab", "stats"},
		{"engram", "vocab", "propose"},
		{"engram", "vocab", "refit"},
		{"engram", "vocab", "tag-definitions"},
		{"engram", "register-skills"},
		{"engram", "serve", "--addr", "127.0.0.1:8093"},
	}
}

// runWithRemovedServer builds Deps over rec's recording primitives with
// ENGRAM_SERVER=serverURL, runs args through targ, and returns stderr plus
// every exit code the command requested (Exit is a recorder, so execution
// continues past it — the guard must still touch no capability).
func runWithRemovedServer(rec *capabilityRecorder, serverURL string, args []string) (string, []int) {
	var stdout, stderr bytes.Buffer

	var exitCodes []int

	deps := cli.NewDeps(rec.primitives(serverURL), nil, &stdout, &stderr, func(code int) {
		exitCodes = append(exitCodes, code)
	})

	_, _ = targ.Execute(args, cli.Targets(deps)...)

	return stderr.String(), exitCodes
}
