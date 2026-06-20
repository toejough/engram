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

	// Appending a turn re-chunks the file. Append-only (D5): the prior chunk is
	// retained and the new chunking is added, so the index reflects the new
	// content while keeping history.
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

	g.Expect(joined.String()).To(gomega.ContainSubstring("coverage thresholds"), "new content ingested")
	g.Expect(joined.String()).To(gomega.ContainSubstring("wire the linter"), "prior chunk retained (append-only)")
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

func TestSweepMergeAppendKeepsStaleAndReusesEmbeddings(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := newSweepFS()
	fs.put("/docs/conv.md", sweepDoc, 100)

	emb := &countingEmbedder{}
	deps := sweepDeps(fs, emb, "/docs/conv.md")
	args := cli.IngestArgs{Sweep: []string{"/docs"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	// Section B is rewritten; Section A is untouched. Append-only (D5): the old
	// Section B chunk is RETAINED, Section A is reused WITHOUT re-embedding, and
	// only the new Section B text embeds.
	edited := "## Section A\nAlways name constants instead of magic numbers in this codebase.\n\n" +
		"## Section B\nUse sentinel errors and errors.Is for all not-found conditions.\n"
	fs.put("/docs/conv.md", edited, 200)

	before := emb.calls

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	records, err := chunk.DecodeRecords(fs.files["/chunks/"+cli.ExportIndexFileName("/docs/conv.md")])
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).To(gomega.HaveLen(3), "merge-append: prior 2 chunks retained + 1 new chunk added")

	var all strings.Builder

	for _, r := range records {
		all.WriteString(r.Text)
		all.WriteString("\n")
	}

	g.Expect(all.String()).To(gomega.ContainSubstring("sentinel errors"))
	g.Expect(all.String()).To(gomega.ContainSubstring("Wrap errors with context"), "prior chunk retained (append-only)")
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

func TestSweepSkipsVanishedSourceButExplicitErrors(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// /docs/gone.md is listed by the sweep but unreadable (vanished between
	// walk and read). The sweep must skip it and still ingest the live file
	// AND write the manifest. An explicitly-named missing source still errors.
	fs := newSweepFS()
	fs.put("/docs/conv.md", sweepDoc, 100)

	emb := &countingEmbedder{}
	deps := sweepDeps(fs, emb, "/docs/conv.md", "/docs/gone.md")
	args := cli.IngestArgs{Sweep: []string{"/docs"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	_, manifestWritten := fs.files["/chunks/manifest.json"]
	g.Expect(manifestWritten).To(gomega.BeTrue(), "manifest lands despite the vanished source")

	records, err := chunk.DecodeRecords(fs.files["/chunks/"+cli.ExportIndexFileName("/docs/conv.md")])
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).NotTo(gomega.BeEmpty(), "live file still ingested")

	explicit := cli.IngestArgs{Markdowns: []string{"/docs/gone.md"}, ChunksDir: "/chunks"}
	g.Expect(cli.RunIngest(context.Background(), explicit, deps, io.Discard)).NotTo(gomega.Succeed(),
		"explicitly-named missing sources still error loudly")
}

func TestShouldPruneDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	excludeNames := map[string]struct{}{"node_modules": {}}
	prefixes := []string{"-private-tmp-", "-tmp-"}

	g.Expect(cli.ExportShouldPruneDir("node_modules", excludeNames, prefixes)).To(gomega.BeTrue(), "name match")
	g.Expect(cli.ExportShouldPruneDir("-private-tmp-cummatrix-x", excludeNames, prefixes)).To(gomega.BeTrue(), "prefix match")
	g.Expect(cli.ExportShouldPruneDir("-Users-joe-repos-engram", excludeNames, prefixes)).To(gomega.BeFalse(), "persistent dir kept")
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
