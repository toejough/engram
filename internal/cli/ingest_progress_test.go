package cli_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// 793d37b1: a run that is cut by its context, or killed, keeps the sources it completed, newest first.

func Test793d37b1a1_ACutIngestKeepsTheSourcesItCompleted(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs := progressFS()
	embedder := progressEmbedder{onEmbed: func(text string) error {
		if strings.Contains(text, "gamma") {
			cancel()

			return ctx.Err()
		}

		return nil
	}}
	deps := cli.IngestDeps{ReadFile: fs.read, WriteFile: fs.write, Embedder: embedder}

	err := cli.RunIngest(ctx, progressArgs(), deps, io.Discard)

	g.Expect(err).To(gomega.MatchError(context.Canceled))
	g.Expect(manifestSources(fs.files["/chunks/manifest.json"])).To(gomega.ConsistOf(alphaDoc, betaDoc))
}

func Test793d37b1a2_AKilledIngestKeepsTheSourcesItCheckpointed(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := progressFS()

	var onDisk []byte

	embedder := progressEmbedder{onEmbed: func(text string) error {
		if strings.Contains(text, "gamma") {
			onDisk = fs.files["/chunks/manifest.json"] // what a kill at this moment leaves

			return errProgressKilled
		}

		return nil
	}}
	deps := cli.IngestDeps{ReadFile: fs.read, WriteFile: fs.write, Embedder: embedder, Now: steppingClock(11 * time.Second)}

	err := cli.RunIngest(context.Background(), progressArgs(), deps, io.Discard)

	g.Expect(err).To(gomega.MatchError(errProgressKilled))
	g.Expect(manifestSources(onDisk)).To(gomega.ConsistOf(alphaDoc, betaDoc))
}

func Test793d37b1a3_TheManifestIsNotRewrittenWithinTheCheckpointInterval(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := progressFS()
	writes := 0
	deps := cli.IngestDeps{
		ReadFile: fs.read,
		WriteFile: func(path string, data []byte) error {
			if path == "/chunks/manifest.json" {
				writes++
			}

			return fs.write(path, data)
		},
		Embedder: progressEmbedder{},
		Now:      func() time.Time { return time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC) },
	}

	g.Expect(cli.RunIngest(context.Background(), progressArgs(), deps, io.Discard)).To(gomega.Succeed())

	g.Expect(writes).To(gomega.Equal(1), "a large manifest is written once per interval, and at the end")
	g.Expect(manifestSources(fs.files["/chunks/manifest.json"])).To(gomega.ConsistOf(alphaDoc, betaDoc, gammaDoc))
}

func Test793d37b1a4_TheNewestSourceIsIndexedFirst(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs := progressFS()
	mtimes := map[string]int64{alphaDoc: 100, betaDoc: 300, gammaDoc: 200}
	deps := cli.IngestDeps{
		ReadFile:  fs.read,
		WriteFile: fs.write,
		Stat: func(path string) (cli.SourceStat, error) {
			return cli.SourceStat{MtimeUnixNano: mtimes[path], Size: 1}, nil
		},
		Embedder: progressEmbedder{onEmbed: func(string) error {
			cancel() // the run is cut as soon as its first source is embedded

			return nil
		}},
	}

	err := cli.RunIngest(ctx, progressArgs(), deps, io.Discard)

	g.Expect(err).To(gomega.MatchError(context.Canceled))
	g.Expect(manifestSources(fs.files["/chunks/manifest.json"])).To(gomega.ConsistOf(betaDoc))
}

func Test793d37b1a5_ARunThatFindsEverythingIndexedWritesNothing(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := progressFS()
	stat := func(string) (cli.SourceStat, error) { return cli.SourceStat{MtimeUnixNano: 1, Size: 1}, nil }
	deps := cli.IngestDeps{ReadFile: fs.read, WriteFile: fs.write, Stat: stat, Embedder: progressEmbedder{}}

	g.Expect(cli.RunIngest(context.Background(), progressArgs(), deps, io.Discard)).To(gomega.Succeed())

	writes := 0
	deps.WriteFile = func(path string, data []byte) error {
		writes++

		return fs.write(path, data)
	}

	g.Expect(cli.RunIngest(context.Background(), progressArgs(), deps, io.Discard)).To(gomega.Succeed())
	g.Expect(writes).To(gomega.BeZero())
}

// 793d37c1 (Modified: 792b18c2): an automatic ingest that finds the manifest lock held skips; a named ingest waits.

func Test793d37c1a1_AnAutomaticIngestSkipsWhenTheManifestLockIsHeld(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := progressFS()
	lock := newHeldLock()
	reads := &atomic.Int64{}
	deps := cli.IngestDeps{
		Lock: lock.lock,
		ReadFile: func(path string) ([]byte, error) {
			reads.Add(1)

			return fs.read(path)
		},
		WriteFile: fs.write,
		Embedder:  progressEmbedder{},
		LockWait:  20 * time.Millisecond,
	}

	started := time.Now()
	err := cli.RunIngest(context.Background(), cli.IngestArgs{Auto: true, ChunksDir: "/chunks"}, deps, io.Discard)

	g.Expect(err).To(gomega.MatchError(cli.ErrIngestBusy))
	g.Expect(time.Since(started)).To(gomega.BeNumerically("<", time.Second), "it does not wait out a deadline")
	g.Expect(reads.Load()).To(gomega.BeZero(), "it reads no source and no manifest: the work is not duplicated")

	g.Expect(fs.files).NotTo(gomega.HaveKey("/chunks/manifest.json"))
}

func Test793d37c1a2_ALockGrantedAfterTheSkipIsGivenBack(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	lock := newHeldLock()
	deps := cli.IngestDeps{Lock: lock.lock, ReadFile: progressFS().read, LockWait: 5 * time.Millisecond}

	err := cli.RunIngest(context.Background(), cli.IngestArgs{Auto: true, ChunksDir: "/chunks"}, deps, io.Discard)
	g.Expect(err).To(gomega.MatchError(cli.ErrIngestBusy))

	lock.free()

	g.Eventually(lock.releases.Load).Should(gomega.BeEquivalentTo(1), "the other run's hold ends, and ours is given back")
}

func Test793d37c1a3_ANamedIngestWaitsForTheManifestLock(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := progressFS()
	lock := newHeldLock()
	deps := cli.IngestDeps{
		Lock: lock.lock, ReadFile: fs.read, WriteFile: fs.write, Embedder: progressEmbedder{},
		LockWait: 5 * time.Millisecond,
	}

	time.AfterFunc(100*time.Millisecond, lock.free)

	g.Expect(cli.RunIngest(context.Background(), progressArgs(), deps, io.Discard)).To(gomega.Succeed())
	g.Expect(manifestSources(fs.files["/chunks/manifest.json"])).To(gomega.ConsistOf(alphaDoc, betaDoc, gammaDoc))
}

const (
	alphaDoc = "/docs/alpha.md"
	betaDoc  = "/docs/beta.md"
	gammaDoc = "/docs/gamma.md"
)

var errProgressKilled = errors.New("the process was killed")

// heldLock is a manifest lock another process holds until free is called: lock blocks until then.
type heldLock struct {
	gate     chan struct{}
	once     sync.Once
	releases atomic.Int64
}

func newHeldLock() *heldLock { return &heldLock{gate: make(chan struct{})} }

func (h *heldLock) free() { h.once.Do(func() { close(h.gate) }) }

func (h *heldLock) lock(string) (func(), error) {
	<-h.gate

	return func() { h.releases.Add(1) }, nil
}

// progressEmbedder runs onEmbed before it answers, so that a test can cut a run at a chosen source.
type progressEmbedder struct{ onEmbed func(text string) error }

func (progressEmbedder) Dims() int { return 2 }

func (p progressEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if p.onEmbed != nil {
		err := p.onEmbed(text)
		if err != nil {
			return nil, err
		}
	}

	return []float32{float32(len(text)), 1}, nil
}

func (progressEmbedder) ModelID() string { return "fake@2" }

func progressArgs() cli.IngestArgs {
	return cli.IngestArgs{Markdowns: []string{alphaDoc, betaDoc, gammaDoc}, ChunksDir: "/chunks"}
}

func progressFS() *memFS {
	body := func(name string) []byte {
		return []byte("## " + name + "\nThe " + name + " document holds enough text to form one chunk in the index.\n")
	}

	return &memFS{files: map[string][]byte{alphaDoc: body("alpha"), betaDoc: body("beta"), gammaDoc: body("gamma")}}
}

// manifestSources lists the sources a manifest holds; an absent manifest holds none.
func manifestSources(data []byte) []string {
	entries := map[string]json.RawMessage{}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &entries)
	}

	sources := make([]string, 0, len(entries))
	for source := range entries {
		sources = append(sources, filepath.ToSlash(source))
	}

	return sources
}

// steppingClock is a clock that moves forward by step at every reading.
func steppingClock(step time.Duration) func() time.Time {
	var mu sync.Mutex

	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()

		now = now.Add(step)

		return now
	}
}
