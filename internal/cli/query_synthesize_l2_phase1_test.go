package cli_test

// Phase 1 RED tests for recall-v2: per-phrase unified note+chunk matching,
// recency-biased. These verify the four invariants from the plan:
//
//	(a) newer IngestedAt chunk ranks above an identical older one in the matched set
//	(b) only top-matchPhraseLimit (30) candidates per phrase enter the union
//	(c) items with baseScore < matchRelevanceFloor (0.25) are dropped
//	(d) the matched set never exceeds matchSetCap (300)
//
// All four tests exercise the --synthesize-l2 path (runSynthesizeL2Query).

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// TestSynthesizeL2_ItemsBelowRelevanceFloorDropped verifies invariant (c):
// items whose baseScore (raw cosine, pre-decay) is below matchRelevanceFloor
// (0.25) are dropped from the matched set. We plant two chunks: one
// near-orthogonal (low cosine) and one on-axis (high cosine). The
// near-orthogonal chunk must not appear in items[].
func TestSynthesizeL2_ItemsBelowRelevanceFloorDropped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	// One L1 note so the union is non-empty.
	plantWithFixedVector(t, memFS, vault, "1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\ncontent\n", queryVec)

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// highChunk: cosine(queryVec, {1,0,0,0}) = 1.0 — well above floor 0.25.
	highChunk := chunk.Record{
		Source:      "/s/a.jsonl",
		Anchor:      "turn-1",
		ContentHash: chunk.HashText("high cosine alpha content"),
		Text:        "high cosine alpha content",
		Vector:      []float32{1, 0, 0, 0},
		IngestedAt:  baseTime,
	}
	// lowChunk: cosine(queryVec, {0,1,0,0}) = 0.0 — well below floor 0.25.
	lowChunk := chunk.Record{
		Source:      "/s/a.jsonl",
		Anchor:      "turn-2",
		ContentHash: chunk.HashText("low cosine unrelated content"),
		Text:        "low cosine unrelated content",
		Vector:      []float32{0, 1, 0, 0}, // orthogonal → cosine 0
		IngestedAt:  baseTime,
	}

	records := []chunk.Record{highChunk, lowChunk}
	data, err := chunk.EncodeRecords(records)
	g.Expect(err).NotTo(HaveOccurred())

	memFS.files["/chunks/a.jsonl"] = data

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	deps.ListChunkIndexes = func(string) ([]string, error) {
		return []string{"/chunks/a.jsonl"}, nil
	}
	deps.Now = func() time.Time { return baseTime }

	var out bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:      []string{"alpha"},
			VaultPath:    vault,
			SynthesizeL2: true,
			ChunksDir:    "/chunks",
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var raw map[string]any

	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	items, _ := raw["items"].([]any)

	seenHigh := false
	seenLow := false

	for _, item := range items {
		mapped, _ := item.(map[string]any)
		if mapped["kind"] != "chunk" {
			continue
		}

		path, _ := mapped["path"].(string)
		switch path {
		case "/s/a.jsonl#turn-1":
			seenHigh = true
		case "/s/a.jsonl#turn-2":
			seenLow = true
		}
	}

	g.Expect(seenHigh).To(BeTrue(), "high-cosine chunk must appear in items[]")
	g.Expect(seenLow).To(BeFalse(),
		"low-cosine chunk (cosine=0 < matchRelevanceFloor=0.25) must be dropped from items[]")
}

// TestSynthesizeL2_MatchedSetCappedAtMatchSetCap verifies invariant (d):
// the total matched set (notes + chunks combined) never exceeds matchSetCap
// (300), even when more candidates exist across phrases. We use multiple
// phrases and enough candidates to exceed 300 if uncapped.
func TestSynthesizeL2_MatchedSetCappedAtMatchSetCap(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	const (
		matchSetCap      = 300
		matchPhraseLimit = 30
		// Use enough phrases that naive union would exceed cap:
		// 15 phrases × 20 candidates each = 300, add extra to push over.
		phraseCount    = 15
		candidateCount = 25 // per phrase, across notes+chunks
	)

	// Plant candidateCount notes (all identical vector so all score highly).
	for i := range candidateCount {
		plantWithFixedVector(t, memFS, vault,
			fmt.Sprintf("%03d.note.md", i),
			fmt.Sprintf("---\ntype: fact\ntier: L1\n---\ncontent alpha %d\n", i),
			queryVec)
	}

	// Add additional chunk candidates to push total above matchSetCap.
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	records := make([]chunk.Record, 0, 300)

	for i := range 300 {
		records = append(records, chunk.Record{
			Source:      "/s/a.jsonl",
			Anchor:      fmt.Sprintf("turn-%d", i),
			ContentHash: chunk.HashText(fmt.Sprintf("alpha chunk %d", i)),
			Text:        fmt.Sprintf("alpha chunk %d", i),
			Vector:      queryVec,
			IngestedAt:  baseTime.Add(time.Duration(i) * time.Second),
		})
	}

	data, err := chunk.EncodeRecords(records)
	g.Expect(err).NotTo(HaveOccurred())

	memFS.files["/chunks/a.jsonl"] = data

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	deps.ListChunkIndexes = func(string) ([]string, error) {
		return []string{"/chunks/a.jsonl"}, nil
	}
	deps.Now = func() time.Time { return baseTime }

	phrases := make([]string, phraseCount)
	for i := range phraseCount {
		phrases[i] = fmt.Sprintf("alpha phrase %d", i)
	}

	var out bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:      phrases,
			VaultPath:    vault,
			SynthesizeL2: true,
			ChunksDir:    "/chunks",
			Limit:        1000, // large so limit flag doesn't cap before matchSetCap
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var raw map[string]any

	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	items, _ := raw["items"].([]any)

	g.Expect(len(items)).To(BeNumerically("<=", matchSetCap),
		"matched set must be capped at matchSetCap=%d, got %d", matchSetCap, len(items))
}

// TestSynthesizeL2_NewerChunkRanksAboveOlderWithSameVector verifies invariant
// (a): two chunks with identical vectors against the phrase — the one with a
// more recent IngestedAt must appear earlier in items[] than the older one.
// Today the code uses global scoreChunks (max-across-phrases) with no per-phrase
// recency, so the two chunks tie and the test fails.
func TestSynthesizeL2_NewerChunkRanksAboveOlderWithSameVector(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	// One L1 note so the union is non-empty and clustering runs.
	plantWithFixedVector(t, memFS, vault, "1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\ncontent\n", queryVec)

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	olderTime := baseTime.Add(-30 * 24 * time.Hour) // 30 days ago
	newerTime := baseTime.Add(-1 * 24 * time.Hour)  // 1 day ago

	olderChunk := chunk.Record{
		Source:      "/s/a.jsonl",
		Anchor:      "turn-1",
		ContentHash: chunk.HashText("older chunk content alpha"),
		Text:        "older chunk content alpha",
		Vector:      queryVec,
		IngestedAt:  olderTime,
	}
	newerChunk := chunk.Record{
		Source:      "/s/a.jsonl",
		Anchor:      "turn-2",
		ContentHash: chunk.HashText("newer chunk content alpha"),
		Text:        "newer chunk content alpha",
		Vector:      queryVec, // identical vector — same raw cosine
		IngestedAt:  newerTime,
	}

	records := []chunk.Record{olderChunk, newerChunk}
	data, err := chunk.EncodeRecords(records)
	g.Expect(err).NotTo(HaveOccurred())

	memFS.files["/chunks/a.jsonl"] = data

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	deps.ListChunkIndexes = func(string) ([]string, error) {
		return []string{"/chunks/a.jsonl"}, nil
	}
	now := baseTime
	deps.Now = func() time.Time { return now }

	var out bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:      []string{"alpha"},
			VaultPath:    vault,
			SynthesizeL2: true,
			ChunksDir:    "/chunks",
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var raw map[string]any

	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	items, ok := raw["items"].([]any)
	g.Expect(ok).To(BeTrue(), "payload must have items[]")

	// Find the two chunk items and verify newer appears before older.
	newerIdx := -1
	olderIdx := -1

	for i, item := range items {
		mapped, _ := item.(map[string]any)
		if mapped["kind"] != "chunk" {
			continue
		}

		path, _ := mapped["path"].(string)
		switch path {
		case "/s/a.jsonl#turn-2":
			newerIdx = i
		case "/s/a.jsonl#turn-1":
			olderIdx = i
		}
	}

	g.Expect(newerIdx).NotTo(Equal(-1), "newer chunk must appear in items[]")
	g.Expect(olderIdx).NotTo(Equal(-1), "older chunk must appear in items[]")
	g.Expect(newerIdx).To(BeNumerically("<", olderIdx),
		"newer chunk (idx %d) must rank above older chunk (idx %d) — recency bias not applied per-phrase",
		newerIdx, olderIdx)
}

// TestSynthesizeL2_OnlyTopMatchPhraseLimitPerPhraseEnterUnion verifies
// invariant (b): when a phrase has more than matchPhraseLimit (30) candidate
// matches, only the top 30 enter the union from that phrase. Today
// unionDirectHits uses the query's `limit` flag (not matchPhraseLimit=30) so
// passing Limit:100 allows >30 candidates — the test must fail today.
func TestSynthesizeL2_OnlyTopMatchPhraseLimitPerPhraseEnterUnion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	// Plant matchPhraseLimit (30) + 10 extra notes — 40 total.
	// All share the same vector so all are candidates for the phrase.
	const (
		matchPhraseLimit = 30
		extraNotes       = 10
		totalNotes       = matchPhraseLimit + extraNotes
	)

	for i := range totalNotes {
		plantWithFixedVector(t, memFS, vault,
			fmt.Sprintf("%02d.note.md", i),
			fmt.Sprintf("---\ntype: fact\ntier: L1\n---\ncontent alpha %d\n", i),
			queryVec)
	}

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	// No chunks needed — testing note-only per-phrase cap.
	deps.ListChunkIndexes = func(string) ([]string, error) {
		return nil, nil
	}

	var out bytes.Buffer

	// Limit is set large enough so the L-flag alone wouldn't cap the union.
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:      []string{"alpha"},
			VaultPath:    vault,
			SynthesizeL2: true,
			ChunksDir:    "/chunks",
			Limit:        100, // large — matchPhraseLimit (30) must still cap
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var raw map[string]any

	g.Expect(yaml.Unmarshal(out.Bytes(), &raw)).NotTo(HaveOccurred())

	items, _ := raw["items"].([]any)

	noteCount := 0

	for _, item := range items {
		mapped, _ := item.(map[string]any)
		if kind, _ := mapped["kind"].(string); kind != "chunk" {
			noteCount++
		}
	}

	g.Expect(noteCount).To(BeNumerically("<=", matchPhraseLimit),
		"matched notes must be capped at matchPhraseLimit=%d, got %d", matchPhraseLimit, noteCount)
}
