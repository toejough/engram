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

	prims := cli.Primitives{Proc: cli.ProcPrims{
		Getenv:      func(string) string { return "" },
		Now:         func() time.Time { return fixed },
		Getwd:       func() (string, error) { return "/work", nil },
		UserHomeDir: func() (string, error) { return "/home/x", nil },
	}}

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

		prims := cli.Primitives{Proc: cli.ProcPrims{
			Getenv: func(string) string { return "" },
			OpenDebugFile: func(string, fs.FileMode) (cli.WriteSyncer, error) {
				t.Error("open must not be called for an empty path")

				return nil, errors.New("unexpected open")
			},
		}}

		g.Expect(cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).DebugLog).To(gomega.BeNil())
	})

	t.Run("failed open yields nil sink", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		prims := cli.Primitives{Proc: cli.ProcPrims{
			Getenv: func(string) string { return "/nope/debug.log" },
			OpenDebugFile: func(string, fs.FileMode) (cli.WriteSyncer, error) {
				return nil, errors.New("open failed")
			},
		}}

		g.Expect(cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).DebugLog).To(gomega.BeNil())
	})
}

func TestNewDeps_DebugSinkSyncsEveryWrite(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	sink := &recordingSyncer{}
	prims := cli.Primitives{Proc: cli.ProcPrims{
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
	}}

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

	prims := cli.Primitives{Proc: cli.ProcPrims{
		StartSignalPulses: func(pulses chan<- struct{}, buffer int) {
			g.Expect(buffer).To(gomega.BeNumerically(">", 0))

			pulsesCh <- pulses
		},
	}}

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
