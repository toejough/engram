package cli_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/transcript"
)

func TestChunkQueryClustersManyChunks(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Two well-separated vector groups, enough members to clear the
	// clustering floor, so the clusters section is exercised.
	records := make([]chunk.Record, 0, 8)
	for i := range 4 {
		records = append(records, chunk.Record{
			Source: "/s/a.jsonl", Anchor: "linter-" + string(rune('a'+i)),
			ContentHash: chunk.HashText("linter " + string(rune('a'+i))),
			Text:        "linter conventions discussion variant with enough text",
			Vector:      []float32{1, 0.01 * float32(i), 0},
		})
		records = append(records, chunk.Record{
			Source: "/s/a.jsonl", Anchor: "feeds-" + string(rune('a'+i)),
			ContentHash: chunk.HashText("feeds " + string(rune('a'+i))),
			Text:        "feed parsing discussion variant with enough text",
			Vector:      []float32{0, 1, 0.01 * float32(i)},
		})
	}

	fs := chunkIndexFS(t, records)
	deps := cli.ChunkQueryDeps{
		ListIndexes: func(string) ([]string, error) { return []string{"/chunks/s1.jsonl"}, nil },
		ReadFile:    fs.read,
		Embedder:    axisEmbedder{axes: map[string][]float32{"linter": {1, 0, 0}}},
	}

	var out strings.Builder

	err := cli.RunChunkQuery(context.Background(), cli.ChunkQueryArgs{
		Phrases: []string{"linter rules"}, ChunksDir: "/chunks", Limit: 10,
	}, deps, &out)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(out.String()).To(gomega.ContainSubstring("clusters:"))
}

func TestChunkQueryIndexErrorsPropagate(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	listErr := cli.ChunkQueryDeps{
		ListIndexes: func(string) ([]string, error) { return nil, errBoom },
		ReadFile:    (&memFS{files: map[string][]byte{}}).read,
		Embedder:    fakeIngestEmbedder{},
	}
	err := cli.RunChunkQuery(context.Background(), cli.ChunkQueryArgs{
		Phrases: []string{"x"}, ChunksDir: "/chunks",
	}, listErr, io.Discard)
	g.Expect(err).To(gomega.MatchError(errBoom))

	readErr := cli.ChunkQueryDeps{
		ListIndexes: func(string) ([]string, error) { return []string{"/chunks/missing.jsonl"}, nil },
		ReadFile:    (&memFS{files: map[string][]byte{}}).read,
		Embedder:    fakeIngestEmbedder{},
	}
	err = cli.RunChunkQuery(context.Background(), cli.ChunkQueryArgs{
		Phrases: []string{"x"}, ChunksDir: "/chunks",
	}, readErr, io.Discard)
	g.Expect(err).To(gomega.HaveOccurred())

	malformed := cli.ChunkQueryDeps{
		ListIndexes: func(string) ([]string, error) { return []string{"/chunks/bad.jsonl"}, nil },
		ReadFile:    (&memFS{files: map[string][]byte{"/chunks/bad.jsonl": []byte("{not json")}}).read,
		Embedder:    fakeIngestEmbedder{},
	}
	err = cli.RunChunkQuery(context.Background(), cli.ChunkQueryArgs{
		Phrases: []string{"x"}, ChunksDir: "/chunks",
	}, malformed, io.Discard)
	g.Expect(err).To(gomega.HaveOccurred())
}

func TestChunkQueryMissingChunksDirIsEmpty(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var out strings.Builder

	deps := cli.ExportNewOsChunkQueryDeps(fakeIngestEmbedder{})
	err := cli.RunChunkQuery(context.Background(), cli.ChunkQueryArgs{
		Phrases: []string{"anything"}, ChunksDir: filepath.Join(t.TempDir(), "absent"), Limit: 5,
	}, deps, &out)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(out.String()).To(gomega.ContainSubstring("total_chunks: 0"))
}

func TestChunkQueryRequiresPhrase(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	err := cli.RunChunkQuery(context.Background(), cli.ChunkQueryArgs{ChunksDir: "/x"}, cli.ChunkQueryDeps{}, io.Discard)

	g.Expect(err).To(gomega.HaveOccurred())
}

func TestIngestMarkdownReadErrorPropagates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	deps := cli.IngestDeps{
		ReadFile:       (&memFS{files: map[string][]byte{}}).read,
		WriteFile:      func(string, []byte) error { return nil },
		ReadTranscript: transcriptReader(""),
		Embedder:       fakeIngestEmbedder{},
	}

	err := cli.RunIngest(context.Background(), cli.IngestArgs{
		Markdowns: []string{"/missing.md"}, ChunksDir: "/chunks",
	}, deps, io.Discard)

	g.Expect(err).To(gomega.HaveOccurred())
}

func TestIngestTranscriptReadErrorPropagates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	deps := cli.IngestDeps{
		ReadFile:  (&memFS{files: map[string][]byte{}}).read,
		WriteFile: func(string, []byte) error { return nil },
		ReadTranscript: func(string, time.Time, int) (transcript.ReadResult, error) {
			return transcript.ReadResult{}, errBoom
		},
		Embedder: fakeIngestEmbedder{},
	}

	err := cli.RunIngest(context.Background(), cli.IngestArgs{
		Transcripts: []string{"/sessions/x.jsonl"}, ChunksDir: "/chunks",
	}, deps, io.Discard)

	g.Expect(err).To(gomega.MatchError(errBoom))
}

// TestOsIngestThenChunkQuery drives the production wiring (with a fake
// embedder) end-to-end through a temp dir: markdown ingest -> index on disk
// -> chunk query payload. Integration test for the thin os adapters.
func TestOsIngestThenChunkQuery(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	g.Expect(os.WriteFile(mdPath, []byte("## Conventions\nName constants instead of magic numbers everywhere.\n"), 0o600)).
		To(gomega.Succeed())

	chunksDir := filepath.Join(dir, "chunks")
	ingestDeps := cli.ExportNewOsIngestDeps(fakeIngestEmbedder{})

	err := cli.RunIngest(context.Background(), cli.IngestArgs{
		Markdowns: []string{mdPath}, ChunksDir: chunksDir,
	}, ingestDeps, io.Discard)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	var out strings.Builder

	queryDeps := cli.ExportNewOsChunkQueryDeps(fakeIngestEmbedder{})
	err = cli.RunChunkQuery(context.Background(), cli.ChunkQueryArgs{
		Phrases: []string{"magic numbers"}, ChunksDir: chunksDir, Limit: 5,
	}, queryDeps, &out)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(out.String()).To(gomega.ContainSubstring("Name constants"))
}

// unexported variables.
var (
	errBoom = errors.New("boom")
)
