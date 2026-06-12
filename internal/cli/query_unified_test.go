package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

func TestRunQuery_ChunkClustersCarryNearestL2(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	// An existing L2 fact so nearest_l2 has something to measure against.
	plantNoteWithSidecar(t, memFS, vault, "1.linting.md",
		"---\ntype: fact\ntier: L2\n---\nAlways run the linter before committing changes.\n")

	// Two clear vector neighborhoods of chunks (hand-set 4-dim vectors), big
	// enough to clear the clustering floor.
	records := make([]chunk.Record, 0, 8)
	for i := range 4 {
		records = append(records, chunk.Record{
			Source: "/s/a.jsonl", Anchor: "lint-" + string(rune('a'+i)),
			ContentHash: chunk.HashText("lint" + string(rune('a'+i))),
			Text:        "reviewer flagged linter convention variant " + string(rune('a'+i)),
			Vector:      []float32{1, 0.01 * float32(i), 0, 0},
		})
		records = append(records, chunk.Record{
			Source: "/s/a.jsonl", Anchor: "feed-" + string(rune('a'+i)),
			ContentHash: chunk.HashText("feed" + string(rune('a'+i))),
			Text:        "feed parsing discussion variant " + string(rune('a'+i)),
			Vector:      []float32{0, 1, 0.01 * float32(i), 0},
		})
	}

	data, err := chunk.EncodeRecords(records)
	g.Expect(err).NotTo(HaveOccurred())

	memFS.files["/chunks/s1.jsonl"] = data

	var out bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"linter conventions"}, VaultPath: vault, Limit: 20, ChunksDir: "/chunks"},
		unifiedQueryDeps(memFS, "/chunks/s1.jsonl"), &out)
	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Clusters []struct {
			Phrase    string `yaml:"phrase"`
			Size      int    `yaml:"size"`
			NearestL2 *struct {
				Path   string  `yaml:"path"`
				Cosine float32 `yaml:"cosine"`
			} `yaml:"nearest_l2"`
		} `yaml:"clusters"`
	}
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	chunkClusters := 0
	withNearest := 0

	for _, c := range parsed.Clusters {
		if c.Phrase == "chunks" {
			chunkClusters++

			if c.NearestL2 != nil && c.NearestL2.Path != "" {
				withNearest++
			}
		}
	}

	g.Expect(chunkClusters).To(BeNumerically(">=", 1), "chunk items must be clustered deterministically")
	g.Expect(withNearest).To(Equal(chunkClusters), "every chunk cluster carries nearest_l2 for the bands")
}

func TestRunQuery_MergesChunkAndVaultSpace(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.linting.md",
		"---\ntype: fact\n---\nAlways run the linter before committing changes.\n")
	plantChunkIndex(t, memFS, "/chunks/s1.jsonl",
		"USER: please wire the linter into the build\nASSISTANT: wired into targ check")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"linter conventions"}, VaultPath: vault, Limit: 10, ChunksDir: "/chunks"},
		unifiedQueryDeps(memFS, "/chunks/s1.jsonl"), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed unifiedParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	kinds := map[string]bool{}
	for _, item := range parsed.Items {
		kinds[item.Kind] = true
	}

	g.Expect(kinds["chunk"]).To(BeTrue(), "a chunk item must appear in the unified ranking")
	g.Expect(kinds["fact"]).To(BeTrue(), "the vault note must appear too")
}

func TestRunQuery_NoChunksDirKeepsVaultOnlyBehavior(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.linting.md",
		"---\ntype: fact\n---\nAlways run the linter before committing changes.\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"linter conventions"}, VaultPath: vault, Limit: 10},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).NotTo(ContainSubstring("kind: chunk"))
}

func TestRunQuery_UnifiedRankingHonorsLimit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.linting.md",
		"---\ntype: fact\n---\nAlways run the linter before committing changes.\n")

	texts := make([]string, 0, 12)
	for i := range 12 {
		texts = append(texts, "USER: linter chatter variant "+strings.Repeat(string(rune('a'+i)), 3))
	}

	plantChunkIndex(t, memFS, "/chunks/s1.jsonl", texts...)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"linter conventions"}, VaultPath: vault, Limit: 10, ChunksDir: "/chunks"},
		unifiedQueryDeps(memFS, "/chunks/s1.jsonl"), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed unifiedParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(len(parsed.Items)).To(BeNumerically("<=", 10))
}

// unifiedParsed is the payload subset the unified-query tests assert on.
type unifiedParsed struct {
	Items []struct {
		Path    string  `yaml:"path"`
		Kind    string  `yaml:"kind"`
		Score   float32 `yaml:"score"`
		Content string  `yaml:"content"`
	} `yaml:"items"`
}

// plantChunkIndex writes a chunk index file with one record per text, vectors
// computed by the same stubEmbedder the query will use.
func plantChunkIndex(t *testing.T, memFS *inMemoryFS, path string, texts ...string) {
	t.Helper()

	emb := stubEmbedder{modelID: "m@4", dims: 4}
	records := make([]chunk.Record, 0, len(texts))

	for i, text := range texts {
		vec, err := emb.Embed(context.Background(), text)
		if err != nil {
			t.Fatal(err)
		}

		records = append(records, chunk.Record{
			Source: "/sessions/s1.jsonl", Anchor: "turn-" + string(rune('1'+i)),
			ContentHash: chunk.HashText(text), Text: text, Vector: vec,
		})
	}

	data, err := chunk.EncodeRecords(records)
	if err != nil {
		t.Fatal(err)
	}

	memFS.files[path] = data
}

func unifiedQueryDeps(memFS *inMemoryFS, indexPaths ...string) cli.QueryDeps {
	deps := newQueryDeps(memFS)
	deps.ListChunkIndexes = func(string) ([]string, error) { return indexPaths, nil }

	return deps
}
