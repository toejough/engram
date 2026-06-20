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

// TestProperty_SynthesizeL2_ChunkItemsBoundedByMatchSetCap locks the
// perf-critical bound for the --synthesize-l2 path: matched chunks (and notes)
// fed to clustering and emitted in items[] with provenance "direct" are capped
// at matchSetCap (300), so silhouette stays O(matchSetCap^2) regardless of
// corpus size. The cap comes from the per-phrase matchPhraseLimit (top-30 per
// phrase) × phrase count. With a single phrase and more than matchPhraseLimit
// chunks, the matched set is capped at matchPhraseLimit (30).
// Note: Phase 2's recency channel may add additional "recent"-provenanced chunk
// items beyond this cap — those do NOT participate in clustering and are NOT
// subject to matchSetCap. This test counts only matched ("direct") chunks.
func TestProperty_SynthesizeL2_ChunkItemsBoundedByMatchSetCap(t *testing.T) {
	t.Parallel()

	queryVec := []float32{1, 0, 0, 0}

	const (
		matchPhraseLimit = 30
		matchSetCap      = 300
	)

	rapid.Check(t, func(rt *rapid.T) {
		// Strictly more matched chunks than matchPhraseLimit.
		chunkCount := matchPhraseLimit + rapid.IntRange(1, 25).Draw(rt, "extraChunks")

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
				Phrases:   []string{"alpha"},
				VaultPath: vault,
				ChunksDir: "/chunks",
				Limit:     matchSetCap * 2, // large so limit flag doesn't confound the matchSetCap test
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

		// Count only matched ("direct") chunk items; Phase 2 recency items are additive.
		matchedChunkItems := countDirectChunkItems(items)

		// Single phrase: matched chunks ≤ matchPhraseLimit (30).
		if matchedChunkItems > matchPhraseLimit {
			rt.Fatalf("matched chunk items = %d, must be <= matchPhraseLimit %d (chunkCount=%d)",
				matchedChunkItems, matchPhraseLimit, chunkCount)
		}
	})
}

// countDirectChunkItems counts items in the raw YAML items list that are of
// kind "chunk" AND carry provenance "direct" (i.e., matched-set chunks, not
// Phase-2 recency-channel chunks which carry "recent").
func countDirectChunkItems(items []any) int {
	count := 0

	for _, item := range items {
		mapped, _ := item.(map[string]any)
		if kind, _ := mapped["kind"].(string); kind != "chunk" {
			continue
		}

		provenances, _ := mapped["provenances"].([]any)
		for _, prov := range provenances {
			if prov == "direct" {
				count++

				break
			}
		}
	}

	return count
}
