package cli_test

// Phase 2 RED tests for recall-v2: recency channel (un-clustered recent fill).
// These verify the three invariants from the plan:
//
//	(a) --synthesize-l2 items[] include up to recentFillChunks (200) newest
//	    chunks NOT already in the matched set, tagged with provenance "recent".
//	(b) those recent-tagged items appear in NO cluster's members[].
//	(c) when fewer than recentFillChunks chunks exist, all are included without
//	    error (no panic, no missing items).
//
// All three tests exercise the --synthesize-l2 path (runSynthesizeL2Query).

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// TestSynthesizeL2_FewerThanRecentFillChunks verifies invariant (c):
// when the chunk store has fewer than recentFillChunks (200) chunks, ALL are
// included in items[] without error (no panic, no "fewer than N" error).
// This exercises the boundary: n > len(candidates) path in newestChunkItems.
func TestSynthesizeL2_FewerThanRecentFillChunks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// One L1 note so the union is non-empty.
	queryVec := []float32{1, 0, 0, 0}

	plantWithFixedVector(t, memFS, vault, "note1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\ncontent\n", queryVec)

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Plant only 3 chunks (far fewer than recentFillChunks=200).
	// Use orthogonal vector so they are NOT in the matched set.
	const smallCount = 3

	records := make([]chunk.Record, 0, smallCount)

	for i := range smallCount {
		records = append(records, chunk.Record{
			Source:      "/s/small.jsonl",
			Anchor:      fmt.Sprintf("turn-%d", i+1),
			ContentHash: chunk.HashText(fmt.Sprintf("small chunk %d", i)),
			Text:        fmt.Sprintf("small chunk %d", i),
			Vector:      []float32{0, 1, 0, 0}, // orthogonal → below floor
			IngestedAt:  baseTime.Add(time.Duration(i) * time.Hour),
		})
	}

	data, err := chunk.EncodeRecords(records)
	g.Expect(err).NotTo(HaveOccurred())

	memFS.files["/chunks/small.jsonl"] = data

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	deps.ListChunkIndexes = func(string) ([]string, error) {
		return []string{"/chunks/small.jsonl"}, nil
	}
	deps.Now = func() time.Time { return baseTime }

	var out bytes.Buffer

	// Must not panic or return an error when chunk count < recentFillChunks.
	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:   []string{"alpha"},
			VaultPath: vault,
			ChunksDir: "/chunks",
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	recentCount := countRecentItems(parsed.Items)

	g.Expect(recentCount).To(Equal(smallCount),
		"with only %d chunks (< recentFillChunks=200), all %d must appear as recent items, got %d",
		smallCount, smallCount, recentCount)
}

// TestSynthesizeL2_RecentChunksAppendedWithRecentProvenance verifies invariant
// (a): chunks NOT in the matched set (orthogonal vector — zero cosine against
// all query phrases) still appear in items[] with provenance "recent" when they
// are among the newest by IngestedAt. Up to recentFillChunks (200) newest
// un-matched chunks must be included.
func TestSynthesizeL2_RecentChunksAppendedWithRecentProvenance(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Query vector along axis-0; recent chunks are orthogonal (axis-1) so
	// their cosine against the query phrase is 0 — below matchRelevanceFloor
	// (0.25) so they will NOT enter the matched set.
	queryVec := []float32{1, 0, 0, 0}
	recentVec := []float32{0, 1, 0, 0}

	// One L1 note so the union is non-empty and clustering runs.
	plantWithFixedVector(t, memFS, vault, "note1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\ncontent\n", queryVec)

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Plant 5 recent chunks (orthogonal → will NOT be in the matched set).
	const recentCount = 5

	records := make([]chunk.Record, 0, recentCount)

	for i := range recentCount {
		records = append(records, chunk.Record{
			Source:      "/s/recent.jsonl",
			Anchor:      fmt.Sprintf("turn-%d", i+1),
			ContentHash: chunk.HashText(fmt.Sprintf("recent orthogonal chunk %d", i)),
			Text:        fmt.Sprintf("recent orthogonal chunk %d", i),
			Vector:      recentVec,
			IngestedAt:  baseTime.Add(time.Duration(i) * time.Hour),
		})
	}

	data, err := chunk.EncodeRecords(records)
	g.Expect(err).NotTo(HaveOccurred())

	memFS.files["/chunks/recent.jsonl"] = data

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	deps.ListChunkIndexes = func(string) ([]string, error) {
		return []string{"/chunks/recent.jsonl"}, nil
	}
	deps.Now = func() time.Time { return baseTime }

	var out bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:   []string{"alpha"},
			VaultPath: vault,
			ChunksDir: "/chunks",
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	recentPaths := recentPathSet(parsed.Items)

	g.Expect(recentPaths).To(HaveLen(recentCount),
		"all %d orthogonal (non-matched) chunks must appear in items[] with provenance 'recent'",
		recentCount)

	// Verify they are the expected chunk paths.
	for i := range recentCount {
		expectedPath := fmt.Sprintf("/s/recent.jsonl#turn-%d", i+1)
		g.Expect(recentPaths).To(HaveKey(expectedPath),
			"recent chunk %s must appear in items[]", expectedPath)
	}
}

// TestSynthesizeL2_RecentChunksNotInClusterMembers verifies invariant (b):
// items tagged with provenance "recent" must NOT appear in any cluster's
// members[]. The recent channel is additive and un-clustered by design.
func TestSynthesizeL2_RecentChunksNotInClusterMembers(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Query vector along axis-0; matched notes share this vector.
	// Recent chunks are orthogonal so they miss the relevance floor.
	queryVec := []float32{1, 0, 0, 0}
	recentVec := []float32{0, 1, 0, 0}

	// Plant enough notes to trigger clustering (need >= minSubgraphForClustering=6).
	const noteCount = 8

	for i := range noteCount {
		plantWithFixedVector(t, memFS, vault,
			fmt.Sprintf("note%02d.md", i),
			fmt.Sprintf("---\ntype: fact\ntier: L1\n---\ncontent alpha %d\n", i),
			queryVec)
	}

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Plant 3 recent (orthogonal) chunks that will NOT enter the matched set.
	const numRecent = 3

	records := make([]chunk.Record, 0, numRecent)

	for i := range numRecent {
		records = append(records, chunk.Record{
			Source:      "/s/recent.jsonl",
			Anchor:      fmt.Sprintf("turn-%d", i+1),
			ContentHash: chunk.HashText(fmt.Sprintf("unrelated recent chunk %d", i)),
			Text:        fmt.Sprintf("unrelated recent chunk %d", i),
			Vector:      recentVec,
			IngestedAt:  baseTime.Add(time.Duration(i+1) * time.Hour),
		})
	}

	data, err := chunk.EncodeRecords(records)
	g.Expect(err).NotTo(HaveOccurred())

	memFS.files["/chunks/recent.jsonl"] = data

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	deps.ListChunkIndexes = func(string) ([]string, error) {
		return []string{"/chunks/recent.jsonl"}, nil
	}
	deps.Now = func() time.Time { return baseTime }

	var out bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:   []string{"alpha"},
			VaultPath: vault,
			ChunksDir: "/chunks",
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	recentPaths := recentPathSet(parsed.Items)

	g.Expect(recentPaths).NotTo(BeEmpty(),
		"expected recent chunks to appear in items[] — none found (channel 2 not implemented)")

	// No recent-tagged path must appear in any cluster's members[].
	for _, cl := range parsed.Clusters {
		for _, member := range cl.Members {
			g.Expect(recentPaths).NotTo(HaveKey(member.Path),
				"recent chunk %s must NOT appear in cluster %d members[]", member.Path, cl.ID)
		}
	}
}

// TestSynthesizeL2_RecentDeduplicatesAgainstMatchedSet verifies that chunks
// already in the matched set (high cosine → above matchRelevanceFloor) are NOT
// duplicated in the recent block. A chunk that appears in the matched items[]
// must NOT also appear as a "recent"-provenanced item.
func TestSynthesizeL2_RecentDeduplicatesAgainstMatchedSet(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	queryVec := []float32{1, 0, 0, 0}

	// One L1 note so the union is non-empty.
	plantWithFixedVector(t, memFS, vault, "note1.ep.md",
		"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\ncontent\n", queryVec)

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// matchedChunk: same vector as query → cosine=1.0 → enters the matched set.
	matchedChunk := chunk.Record{
		Source:      "/s/chunks.jsonl",
		Anchor:      "turn-1",
		ContentHash: chunk.HashText("matched chunk alpha content"),
		Text:        "matched chunk alpha content",
		Vector:      queryVec,
		IngestedAt:  baseTime.Add(2 * time.Hour), // newest
	}
	// recentOnlyChunk: orthogonal → below floor → NOT in matched set → should be in recent.
	recentOnlyChunk := chunk.Record{
		Source:      "/s/chunks.jsonl",
		Anchor:      "turn-2",
		ContentHash: chunk.HashText("unrelated recent only chunk"),
		Text:        "unrelated recent only chunk",
		Vector:      []float32{0, 1, 0, 0},
		IngestedAt:  baseTime.Add(1 * time.Hour),
	}

	records := []chunk.Record{matchedChunk, recentOnlyChunk}
	data, err := chunk.EncodeRecords(records)
	g.Expect(err).NotTo(HaveOccurred())

	memFS.files["/chunks/chunks.jsonl"] = data

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	deps.ListChunkIndexes = func(string) ([]string, error) {
		return []string{"/chunks/chunks.jsonl"}, nil
	}
	deps.Now = func() time.Time { return baseTime }

	var out bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{
			Phrases:   []string{"alpha"},
			VaultPath: vault,
			ChunksDir: "/chunks",
		},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	assertMatchedNotDuplicated(g, parsed.Items, "/s/chunks.jsonl#turn-1")
	assertRecentPresent(g, parsed.Items, "/s/chunks.jsonl#turn-2")
}

// assertMatchedNotDuplicated asserts the given path appears exactly once in
// items[] and does NOT carry provenance "recent" (it is in the matched set).
func assertMatchedNotDuplicated(g Gomega, items []struct {
	Path        string   `yaml:"path"`
	Kind        string   `yaml:"kind"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	ClusterID   *int     `yaml:"cluster_id,omitempty"`
	InDegree    *int     `yaml:"in_degree,omitempty"`
	Content     string   `yaml:"content"`
}, matchedPath string,
) {
	matchedAppearances := 0

	for _, item := range items {
		if item.Path == matchedPath {
			matchedAppearances++
		}
	}

	g.Expect(matchedAppearances).To(Equal(1),
		"chunk already in the matched set must appear exactly once in items[], not duplicated as recent; got %d appearances",
		matchedAppearances)

	for _, item := range items {
		if item.Path == matchedPath {
			g.Expect(item.Provenances).NotTo(ContainElement("recent"),
				"matched chunk %s must not carry 'recent' provenance", matchedPath)
		}
	}
}

// assertRecentPresent asserts the given path appears in items[] with
// provenance "recent".
func assertRecentPresent(g Gomega, items []struct {
	Path        string   `yaml:"path"`
	Kind        string   `yaml:"kind"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	ClusterID   *int     `yaml:"cluster_id,omitempty"`
	InDegree    *int     `yaml:"in_degree,omitempty"`
	Content     string   `yaml:"content"`
}, recentPath string,
) {
	foundRecent := false

	for _, item := range items {
		if item.Path == recentPath && slices.Contains(item.Provenances, "recent") {
			foundRecent = true

			break
		}
	}

	g.Expect(foundRecent).To(BeTrue(),
		"orthogonal chunk %s must appear with 'recent' provenance", recentPath)
}

// countRecentItems counts how many items in parsed.Items carry provenance "recent".
func countRecentItems(items []struct {
	Path        string   `yaml:"path"`
	Kind        string   `yaml:"kind"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	ClusterID   *int     `yaml:"cluster_id,omitempty"`
	InDegree    *int     `yaml:"in_degree,omitempty"`
	Content     string   `yaml:"content"`
},
) int {
	count := 0

	for _, item := range items {
		if slices.Contains(item.Provenances, "recent") {
			count++
		}
	}

	return count
}

// recentPathSet returns the set of item paths that carry provenance "recent".
func recentPathSet(items []struct {
	Path        string   `yaml:"path"`
	Kind        string   `yaml:"kind"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	ClusterID   *int     `yaml:"cluster_id,omitempty"`
	InDegree    *int     `yaml:"in_degree,omitempty"`
	Content     string   `yaml:"content"`
},
) map[string]bool {
	out := make(map[string]bool)

	for _, item := range items {
		if slices.Contains(item.Provenances, "recent") {
			out[item.Path] = true
		}
	}

	return out
}
