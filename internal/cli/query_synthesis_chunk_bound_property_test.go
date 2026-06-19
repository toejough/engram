package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// TestProperty_SynthesizeL2_ChunkItemsBoundedByLimit locks the perf-critical
// bound: matched chunks fed to the unified clustering (and emitted in items[])
// are capped at limit, so silhouette stays O(limit^2) regardless of corpus
// size. Without the cap, appendSynthesisChunks clusters and emits the entire
// corpus, making recall O(corpus^2).
func TestProperty_SynthesizeL2_ChunkItemsBoundedByLimit(t *testing.T) {
	t.Parallel()

	queryVec := []float32{1, 0, 0, 0}

	rapid.Check(t, func(rt *rapid.T) {
		limit := rapid.IntRange(1, 8).Draw(rt, "limit")
		// Strictly more matched chunks than the limit.
		chunkCount := limit + rapid.IntRange(1, 25).Draw(rt, "extraChunks")

		vault := t.TempDir()
		memFS := newInMemoryFS()

		// One L1 note so the union is non-empty and clustering runs.
		plantDualVector(t, memFS, vault, "1.ep.md",
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)

		records := make([]chunk.Record, 0, chunkCount)
		for i := range chunkCount {
			text := fmt.Sprintf("chunk alpha %d", i)
			records = append(records, chunk.Record{
				Source:      "/sessions/s.jsonl",
				Anchor:      fmt.Sprintf("turn-%d", i),
				ContentHash: chunk.HashText(text),
				Text:        text,
				Vector:      queryVec,
			})
		}

		data, encErr := chunk.EncodeRecords(records)
		if encErr != nil {
			rt.Fatalf("encode records: %v", encErr)
		}

		memFS.files["/chunks/s.jsonl"] = data

		deps := newQueryDeps(memFS)
		deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
		deps.ListChunkIndexes = func(string) ([]string, error) {
			return []string{"/chunks/s.jsonl"}, nil
		}

		var out bytes.Buffer

		err := cli.RunQuery(context.Background(),
			cli.QueryArgs{
				Phrases:      []string{"alpha"},
				VaultPath:    vault,
				SynthesizeL2: true,
				ChunksDir:    "/chunks",
				Limit:        limit,
			},
			deps, &out)
		if err != nil {
			rt.Fatalf("RunQuery: %v", err)
		}

		var raw map[string]any
		if err := yaml.Unmarshal(out.Bytes(), &raw); err != nil {
			rt.Fatalf("unmarshal: %v", err)
		}

		items, ok := raw["items"].([]any)
		if !ok {
			rt.Fatalf("payload has no items[] list (got %T)", raw["items"])
		}

		chunkItems := 0

		for _, item := range items {
			mapped, _ := item.(map[string]any)
			if kind, _ := mapped["kind"].(string); kind == "chunk" {
				chunkItems++
			}
		}

		if chunkItems > limit {
			rt.Fatalf("chunk items = %d, must be <= limit %d (chunkCount=%d)",
				chunkItems, limit, chunkCount)
		}
	})
}
