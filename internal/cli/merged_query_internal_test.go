package cli

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// TestMergeQueryPayloads_ClustersStayLocalOnly verifies the merged payload
// does not mix local and parent clusters — no cross-node cluster grouping
// is computed (spec: "Merged results are not re-clustered across nodes").
func TestMergeQueryPayloads_ClustersStayLocalOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	local := queryPayload{Clusters: []queryCluster{{ID: 0, Phrase: "local phrase"}}}
	parent := queryPayload{Clusters: []queryCluster{{ID: 0, Phrase: "parent phrase"}}}

	merged := mergeQueryPayloads(local, parent, QueryArgs{})

	g.Expect(merged.Clusters).To(Equal(local.Clusters))
}

// TestMergeQueryPayloads_CombinesAndSortsByScore verifies items from both
// sources appear in one list, ordered by descending score.
func TestMergeQueryPayloads_CombinesAndSortsByScore(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	local := queryPayload{
		ModelID: "m@4",
		Items: []queryItem{
			{Path: "local-high", Score: 0.9},
			{Path: "local-low", Score: 0.1},
		},
	}
	parent := queryPayload{
		ModelID: "m@4",
		Items: []queryItem{
			{Path: "parent-mid", Score: 0.5},
		},
	}

	merged := mergeQueryPayloads(local, parent, QueryArgs{})

	paths := make([]string, len(merged.Items))
	for i, item := range merged.Items {
		paths[i] = item.Path
	}

	g.Expect(paths).To(Equal([]string{"local-high", "parent-mid", "local-low"}))
}

// TestMergeQueryPayloads_ContentBudgetCapsMergedChunks verifies
// --content-budget caps full-content chunk items across the merged set,
// not per-source (each source's own content-budget pass is bypassed by
// fetching unbounded — this test exercises the merge's own re-application).
func TestMergeQueryPayloads_ContentBudgetCapsMergedChunks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	long := "line one\nline two with many words " + strings.Repeat("x", 200)

	local := queryPayload{Items: []queryItem{
		{Path: "l1", Kind: chunkItemKind, Score: 0.9, Content: long},
		{Path: "l2", Kind: chunkItemKind, Score: 0.7, Content: long},
	}}
	parent := queryPayload{Items: []queryItem{
		{Path: "p1", Kind: chunkItemKind, Score: 0.8, Content: long},
	}}

	merged := mergeQueryPayloads(local, parent, QueryArgs{ContentBudget: 2})

	// Rank order: l1 (0.9), p1 (0.8), l2 (0.7) — content-budget=2 keeps the
	// first two chunks full, snippets the third.
	g.Expect(merged.Items[0].Content).To(Equal(long))
	g.Expect(merged.Items[1].Content).To(Equal(long))
	g.Expect(merged.Items[2].Content).NotTo(Equal(long))
}

// TestMergeQueryPayloads_LimitCapsTotalMergedSet verifies --limit caps the
// combined item count, not each source independently.
func TestMergeQueryPayloads_LimitCapsTotalMergedSet(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	local := queryPayload{Items: []queryItem{
		{Path: "l1", Score: 0.9}, {Path: "l2", Score: 0.8}, {Path: "l3", Score: 0.7},
	}}
	parent := queryPayload{Items: []queryItem{
		{Path: "p1", Score: 0.85}, {Path: "p2", Score: 0.75},
	}}

	merged := mergeQueryPayloads(local, parent, QueryArgs{Limit: 3})

	g.Expect(merged.Items).To(HaveLen(3))
	g.Expect(merged.Items[0].Path).To(Equal("l1"))
	g.Expect(merged.Items[1].Path).To(Equal("p1"))
	g.Expect(merged.Items[2].Path).To(Equal("l2"))
}

// TestMergeQueryPayloads_MismatchedModelIDStillMerges verifies the merge
// proceeds unconditionally regardless of model_id mismatch (design.md
// Decision 3 — no refuse, no fallback on mismatch).
func TestMergeQueryPayloads_MismatchedModelIDStillMerges(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	local := queryPayload{ModelID: "m@4", Items: []queryItem{{Path: "a", Score: 0.5}}}
	parent := queryPayload{ModelID: "m@5-different", Items: []queryItem{{Path: "b", Score: 0.3}}}

	merged := mergeQueryPayloads(local, parent, QueryArgs{})

	g.Expect(merged.Items).To(HaveLen(2))
}

// TestMergeQueryPayloads_ORsAdvisoryFlags verifies RefitPending and
// PendingOffers are true in the merged payload if either source has them.
func TestMergeQueryPayloads_ORsAdvisoryFlags(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	local := queryPayload{RefitPending: false, PendingOffers: true}
	parent := queryPayload{RefitPending: true, PendingOffers: false}

	merged := mergeQueryPayloads(local, parent, QueryArgs{})

	g.Expect(merged.RefitPending).To(BeTrue())
	g.Expect(merged.PendingOffers).To(BeTrue())
}

// TestMergeQueryPayloads_PayloadModelIDIsQueryingNodesOwn verifies the
// payload-level ModelID stays the querying (local) node's own model — the
// per-item field is the new per-source signal, not this one.
func TestMergeQueryPayloads_PayloadModelIDIsQueryingNodesOwn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	local := queryPayload{ModelID: "local-model"}
	parent := queryPayload{ModelID: "parent-model"}

	merged := mergeQueryPayloads(local, parent, QueryArgs{})

	g.Expect(merged.ModelID).To(Equal("local-model"))
}

// TestMergeQueryPayloads_RecencyItemsDoNotCountAgainstMainRanking verifies
// recency-channel items (score 0, provenance recent) are appended after
// the score-ranked main items, mirroring single-source behavior
// (assembleResolvedItems appends Channel 2 after Channel 1).
func TestMergeQueryPayloads_RecencyItemsDoNotCountAgainstMainRanking(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	local := queryPayload{Items: []queryItem{
		{Path: "recent", Score: 0, Provenances: []string{provenanceRecent}},
		{Path: "matched-low", Score: 0.1},
	}}
	parent := queryPayload{}

	merged := mergeQueryPayloads(local, parent, QueryArgs{})

	g.Expect(merged.Items[0].Path).To(Equal("matched-low"), "matched item ranks before recency channel")
	g.Expect(merged.Items[1].Path).To(Equal("recent"))
}

// TestMergeQueryPayloads_RecentFillCapsMergedRecencyChannel verifies
// --recent-fill caps the combined recency-channel item count across both
// sources, not each source independently.
func TestMergeQueryPayloads_RecentFillCapsMergedRecencyChannel(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	local := queryPayload{Items: []queryItem{
		{Path: "l-recent-1", Score: 0, Provenances: []string{provenanceRecent}},
		{Path: "l-recent-2", Score: 0, Provenances: []string{provenanceRecent}},
	}}
	parent := queryPayload{Items: []queryItem{
		{Path: "p-recent-1", Score: 0, Provenances: []string{provenanceRecent}},
		{Path: "p-recent-2", Score: 0, Provenances: []string{provenanceRecent}},
	}}

	merged := mergeQueryPayloads(local, parent, QueryArgs{RecentFill: 2})

	recentCount := 0

	for _, item := range merged.Items {
		for _, p := range item.Provenances {
			if p == provenanceRecent {
				recentCount++
			}
		}
	}

	g.Expect(recentCount).To(Equal(2))
}

// TestMergeQueryPayloads_TagsItemsWithOriginAndModelID verifies each item
// carries its source node's model_id and a from_parent flag.
func TestMergeQueryPayloads_TagsItemsWithOriginAndModelID(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	local := queryPayload{
		ModelID: "local-model",
		Items:   []queryItem{{Path: "local-item", Score: 0.9}},
	}
	parent := queryPayload{
		ModelID: "parent-model",
		Items:   []queryItem{{Path: "parent-item", Score: 0.5}},
	}

	merged := mergeQueryPayloads(local, parent, QueryArgs{})

	g.Expect(merged.Items).To(ConsistOf(
		queryItem{Path: "local-item", Score: 0.9, ModelID: "local-model", FromParent: false},
		queryItem{Path: "parent-item", Score: 0.5, ModelID: "parent-model", FromParent: true},
	))
}
