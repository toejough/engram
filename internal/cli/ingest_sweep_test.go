package cli_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/transcript"
)

func TestSweepHandlesTranscriptsViaSameMechanism(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := newSweepFS()
	fs.put("/sessions/s1.jsonl",
		"USER: please wire the linter into the build system for this project\nASSISTANT: wired into targ check",
		100)

	emb := &countingEmbedder{}
	deps := sweepDeps(fs, emb, "/sessions/s1.jsonl")
	args := cli.IngestArgs{Sweep: []string{"/sessions"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	records, err := chunk.DecodeRecords(fs.files["/chunks/"+cli.ExportIndexFileName("/sessions/s1.jsonl")])
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).To(gomega.HaveLen(1))

	if len(records) != 1 {
		return
	}

	g.Expect(records[0].Anchor).To(gomega.HavePrefix("turn-"))

	// Appending a turn re-chunks the file; the unchanged-boundary case here
	// merges into one bigger chunk, so the old vector is NOT reusable — but
	// the index must reflect ONLY the new chunking (no stale duplicates).
	fs.put("/sessions/s1.jsonl",
		"USER: please wire the linter into the build system for this project\nASSISTANT: wired into targ check\n"+
			"USER: also add coverage thresholds to the check pipeline\nASSISTANT: added with 80 percent floor",
		200)

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	records, err = chunk.DecodeRecords(fs.files["/chunks/"+cli.ExportIndexFileName("/sessions/s1.jsonl")])
	g.Expect(err).NotTo(gomega.HaveOccurred())

	var joined strings.Builder

	for _, r := range records {
		joined.WriteString(r.Text)
		joined.WriteString("\n")
	}

	g.Expect(joined.String()).To(gomega.ContainSubstring("coverage thresholds"))
	g.Expect(strings.Count(joined.String(), "wire the linter")).To(gomega.Equal(1), "no stale duplicate chunks")
}

func TestSweepIngestsNewMarkdown(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := newSweepFS()
	fs.put("/docs/conv.md", sweepDoc, 100)

	emb := &countingEmbedder{}

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Sweep: []string{"/docs"}, ChunksDir: "/chunks"},
		sweepDeps(fs, emb, "/docs/conv.md"), io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	records, err := chunk.DecodeRecords(fs.files["/chunks/"+cli.ExportIndexFileName("/docs/conv.md")])
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).To(gomega.HaveLen(2))
	g.Expect(emb.calls).To(gomega.Equal(2))
}

func TestSweepManifestWriteErrorPropagates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := newSweepFS()
	fs.put("/docs/conv.md", sweepDoc, 100)

	deps := sweepDeps(fs, &countingEmbedder{}, "/docs/conv.md")
	deps.WriteFile = func(path string, data []byte) error {
		if strings.HasSuffix(path, "manifest.json") {
			return errBoom
		}

		return fs.write(path, data)
	}

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Sweep: []string{"/docs"}, ChunksDir: "/chunks"}, deps, io.Discard)

	g.Expect(err).To(gomega.MatchError(errBoom))
}

func TestSweepRebuildRemovesStaleChunksAndReusesEmbeddings(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := newSweepFS()
	fs.put("/docs/conv.md", sweepDoc, 100)

	emb := &countingEmbedder{}
	deps := sweepDeps(fs, emb, "/docs/conv.md")
	args := cli.IngestArgs{Sweep: []string{"/docs"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	// Section B is rewritten; Section A is untouched. The rebuild must drop the
	// old Section B chunk, keep Section A WITHOUT re-embedding it, and embed
	// only the new text.
	edited := "## Section A\nAlways name constants instead of magic numbers in this codebase.\n\n" +
		"## Section B\nUse sentinel errors and errors.Is for all not-found conditions.\n"
	fs.put("/docs/conv.md", edited, 200)

	before := emb.calls

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	records, err := chunk.DecodeRecords(fs.files["/chunks/"+cli.ExportIndexFileName("/docs/conv.md")])
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).To(gomega.HaveLen(2), "rebuild replaces, never appends")

	var all strings.Builder

	for _, r := range records {
		all.WriteString(r.Text)
		all.WriteString("\n")
	}

	g.Expect(all.String()).To(gomega.ContainSubstring("sentinel errors"))
	g.Expect(all.String()).NotTo(gomega.ContainSubstring("Wrap errors with context"), "stale chunk must be gone")
	g.Expect(emb.calls-before).To(gomega.Equal(1), "only the edited section re-embeds")
}

func TestSweepSkipsUnchangedWithoutEmbedding(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := newSweepFS()
	fs.put("/docs/conv.md", sweepDoc, 100)

	emb := &countingEmbedder{}
	deps := sweepDeps(fs, emb, "/docs/conv.md")
	args := cli.IngestArgs{Sweep: []string{"/docs"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	before := emb.calls

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())
	g.Expect(emb.calls).To(gomega.Equal(before), "unchanged file must not re-embed")
}

// unexported constants.
const (
	sweepDoc = "## Section A\nAlways name constants instead of magic numbers in this codebase.\n\n" +
		"## Section B\nWrap errors with context strings before returning them upward.\n"
)

// countingEmbedder counts Embed calls so tests can assert embedding reuse.
type countingEmbedder struct{ calls int }

func (e *countingEmbedder) Dims() int { return 2 }

func (e *countingEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	e.calls++

	return []float32{float32(len(text)), 1}, nil
}

func (e *countingEmbedder) ModelID() string { return "fake@2" }

// sweepFS extends memFS with stat metadata per path.
type sweepFS struct {
	memFS

	stats map[string]cli.SourceStat
}

func (s *sweepFS) put(path, content string, mtime int64) {
	s.files[path] = []byte(content)
	s.stats[path] = cli.SourceStat{MtimeUnixNano: mtime, Size: int64(len(content))}
}

func (s *sweepFS) stat(path string) (cli.SourceStat, error) {
	st, ok := s.stats[path]
	if !ok {
		return cli.SourceStat{}, io.ErrUnexpectedEOF
	}

	return st, nil
}

func newSweepFS() *sweepFS {
	return &sweepFS{memFS: memFS{files: map[string][]byte{}}, stats: map[string]cli.SourceStat{}}
}

func sweepDeps(fs *sweepFS, emb *countingEmbedder, sources ...string) cli.IngestDeps {
	return cli.IngestDeps{
		ReadFile:  fs.read,
		WriteFile: fs.write,
		Stat:      fs.stat,
		ListSources: func(cli.SweepRoot) ([]string, error) {
			return sources, nil
		},
		ReadTranscript: func(path string, _ time.Time, _ int) (transcript.ReadResult, error) {
			data, err := fs.read(path)
			if err != nil {
				return transcript.ReadResult{}, err
			}

			return transcript.ReadResult{Content: string(data)}, nil
		},
		Embedder: emb,
	}
}
