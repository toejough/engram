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

func TestIngestIsIdempotentByHash(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stripped := "USER: same content as before, long enough to clear the noise floor easily"
	fs := &memFS{files: map[string][]byte{"/sessions/s1.jsonl": []byte("{raw jsonl}")}}
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(stripped),
		Embedder:       fakeIngestEmbedder{},
	}
	args := cli.IngestArgs{Transcripts: []string{"/sessions/s1.jsonl"}, ChunksDir: "/chunks"}

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	first := fs.files["/chunks/s1.jsonl"]

	g.Expect(cli.RunIngest(context.Background(), args, deps, io.Discard)).To(gomega.Succeed())

	second := fs.files["/chunks/s1.jsonl"]

	g.Expect(second).To(gomega.Equal(first), "second ingest adds nothing")

	records, err := chunk.DecodeRecords(second)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).To(gomega.HaveLen(1))
}

func TestIngestMarkdownFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	md := "## Conventions\nAlways name constants instead of magic numbers in this codebase.\n"
	fs := &memFS{files: map[string][]byte{"/docs/conventions.md": []byte(md)}}
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(""),
		Embedder:       fakeIngestEmbedder{},
	}

	err := cli.RunIngest(context.Background(), cli.IngestArgs{
		Markdowns: []string{"/docs/conventions.md"},
		ChunksDir: "/chunks",
	}, deps, io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	records, err := chunk.DecodeRecords(fs.files["/chunks/conventions.jsonl"])
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).To(gomega.HaveLen(1))

	if len(records) != 1 {
		return
	}

	g.Expect(records[0].Anchor).To(gomega.Equal("Conventions"))
}

func TestIngestTranscriptWritesChunkIndex(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stripped := strings.Join([]string{
		"USER: please add the linter config and wire it into the build system",
		"ASSISTANT: added golangci config and wired into targ check, all green",
	}, "\n")
	fs := &memFS{files: map[string][]byte{"/sessions/abc.jsonl": []byte("{raw jsonl}")}}
	deps := cli.IngestDeps{
		ReadFile:       fs.read,
		WriteFile:      fs.write,
		ReadTranscript: transcriptReader(stripped),
		Embedder:       fakeIngestEmbedder{},
	}

	err := cli.RunIngest(context.Background(), cli.IngestArgs{
		Transcripts: []string{"/sessions/abc.jsonl"},
		ChunksDir:   "/chunks",
	}, deps, io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	data, ok := fs.files["/chunks/abc.jsonl"]
	g.Expect(ok).To(gomega.BeTrue(), "index file written")

	records, err := chunk.DecodeRecords(data)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(records).To(gomega.HaveLen(1))

	if len(records) != 1 {
		return
	}

	g.Expect(records[0].Source).To(gomega.Equal("/sessions/abc.jsonl"))
	g.Expect(records[0].Vector).NotTo(gomega.BeEmpty())
	g.Expect(records[0].ContentHash).To(gomega.HavePrefix("sha256:"))
}

// fakeIngestEmbedder returns a fixed-dim vector derived from text length so
// tests can assert vectors landed without a real model.
type fakeIngestEmbedder struct{}

func (fakeIngestEmbedder) Dims() int { return 2 }

func (fakeIngestEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return []float32{float32(len(text)), 1}, nil
}

func (fakeIngestEmbedder) ModelID() string { return "fake@2" }

// memFS is an in-memory filesystem for ingest deps.
type memFS struct {
	files map[string][]byte
}

func (m *memFS) read(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}

	return data, nil
}

func (m *memFS) write(path string, data []byte) error {
	m.files[path] = data

	return nil
}

func transcriptReader(stripped string) func(string, time.Time, int) (transcript.ReadResult, error) {
	return func(string, time.Time, int) (transcript.ReadResult, error) {
		return transcript.ReadResult{Content: stripped}, nil
	}
}
