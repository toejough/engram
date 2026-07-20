package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

func TestChunkQueryDeps_ListIndexes_ListsOnlyJSONLFiles(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(dir, "sub.jsonl"), 0o750)).To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0o600)).To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o600)).To(gomega.Succeed())

	deps := cli.ExportNewChunkQueryDeps(realFSForTest(), nil)
	paths, err := deps.ListIndexes(dir)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(paths).To(gomega.ConsistOf(filepath.Join(dir, "a.jsonl")))
}

func TestChunkQueryDeps_ListIndexes_WrappedNotExistIsEmptyIndex(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	deps := cli.ExportNewChunkQueryDeps(wrappedNotExistEdgeFS{}, nil)
	paths, err := deps.ListIndexes("/any/chunks/dir")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(paths).To(gomega.BeEmpty())
}

func TestChunkQueryEmptyIndexSaysSo(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	deps := cli.ChunkQueryDeps{
		ListIndexes: func(string) ([]string, error) { return nil, nil },
		ReadFile:    (&memFS{files: map[string][]byte{}}).read,
		Embedder:    axisEmbedder{},
	}

	var out strings.Builder

	err := cli.RunChunkQuery(context.Background(), cli.ChunkQueryArgs{
		Phrases: []string{"anything"}, ChunksDir: "/chunks", Limit: 5,
	}, deps, &out)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(out.String()).To(gomega.ContainSubstring("total_chunks: 0"))
}

func TestChunkQueryHonorsLimit(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	records := make([]chunk.Record, 0, 10)
	for i := range 10 {
		records = append(records, chunk.Record{
			Source: "/s/a.jsonl", Anchor: "turn-" + string(rune('0'+i)),
			ContentHash: chunk.HashText(string(rune('a' + i))),
			Text:        "linter topic variant " + string(rune('a'+i)),
			Vector:      []float32{1, float32(i) * 0.01, 0},
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
		Phrases: []string{"linter rules"}, ChunksDir: "/chunks", Limit: 3,
	}, deps, &out)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(strings.Count(out.String(), "- source:")).To(gomega.Equal(3))
}

func TestChunkQueryRanksByCosine(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := chunkIndexFS(t, []chunk.Record{
		{
			Source: "/s/a.jsonl", Anchor: "turn-1", ContentHash: "sha256:aa",
			Text: "linter conventions discussion", Vector: []float32{1, 0, 0},
		},
		{
			Source: "/s/a.jsonl", Anchor: "turn-2", ContentHash: "sha256:bb",
			Text: "unrelated feed parsing chat", Vector: []float32{0, 1, 0},
		},
	})
	deps := cli.ChunkQueryDeps{
		ListIndexes: func(string) ([]string, error) { return []string{"/chunks/s1.jsonl"}, nil },
		ReadFile:    fs.read,
		Embedder:    axisEmbedder{axes: map[string][]float32{"linter": {1, 0, 0}}},
	}

	var out strings.Builder

	err := cli.RunChunkQuery(context.Background(), cli.ChunkQueryArgs{
		Phrases: []string{"linter rules"}, ChunksDir: "/chunks", Limit: 5,
	}, deps, &out)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	payload := out.String()
	g.Expect(payload).To(gomega.ContainSubstring("linter conventions discussion"))

	first := strings.Index(payload, "turn-1")
	second := strings.Index(payload, "turn-2")

	if second != -1 {
		g.Expect(first).To(gomega.BeNumerically("<", second), "best match listed first")
	}
}

// axisEmbedder maps known texts onto fixed unit vectors so cosine ranking is
// fully deterministic in tests.
type axisEmbedder struct{ axes map[string][]float32 }

func (axisEmbedder) Dims() int { return 3 }

func (e axisEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	for key, vec := range e.axes {
		if strings.Contains(text, key) {
			return vec, nil
		}
	}

	return []float32{0, 0, 1}, nil
}

func (axisEmbedder) ModelID() string { return "fake@3" }

func chunkIndexFS(t *testing.T, records []chunk.Record) *memFS {
	t.Helper()

	data, err := chunk.EncodeRecords(records)
	if err != nil {
		t.Fatal(err)
	}

	return &memFS{files: map[string][]byte{"/chunks/s1.jsonl": data}}
}
