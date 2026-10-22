package cli

// Scenario tests of the parent-sync product capability, part 1 (861c2zx): the order, budgets, tags, clusters,
// budget block and dedupe of a merged query (792a21d, 792a21e, 792a21h, 792a21i, 792a21j). Spec:
// docs/specs/product/parent-sync.md. White-box: mergeQueryPayloads is the one place the merged answer is made, and its
// inputs (queryPayload, queryItem, exchangeNote) are unexported. The CLI journeys of the same capability are in
// parent_sync_part1_scenarios_test.go.

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
)

// Test792a21d1a_MergeQueryPayloads_TriggerHitsLeadLocalBeforeParent verifies 792a21d1a (Trigger hits lead, local
// before parent): with --limit 1, the items run local trigger hit, parent trigger hit, local 0.9 item; both trigger
// hits survive the limit and the parent's 0.8 item is dropped.
func Test792a21d1a_MergeQueryPayloads_TriggerHitsLeadLocalBeforeParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	local := queryPayload{ModelID: ps1LocalModel, Items: []queryItem{
		ps1Direct("local-high.md", 0.9),
		ps1Trigger("local-trigger.md"),
	}}
	parent := queryPayload{ModelID: ps1ParentModel, Items: []queryItem{
		ps1Direct("parent-high.md", 0.8),
		ps1Trigger("parent-trigger.md"),
	}}

	merged := mergeQueryPayloads(local, parent, nil, QueryArgs{Limit: 1})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"local-trigger.md", "parent-trigger.md", "local-high.md"}))
	g.Expect(ps1Tag(merged.Items, "local-trigger.md")).To(Equal("local"))
	g.Expect(ps1Tag(merged.Items, "parent-trigger.md")).To(Equal("parent"))
}

// Test792a21d1b_MergeQueryPayloads_ExplorePicksDoNotCrowdOutDirectMatches verifies 792a21d1b (Explore picks do not
// crowd out direct matches): with --limit 10, 12 explore picks scoring 0.8 to 0.9 and 15 direct items scoring 0.5 to
// 0.7 across both sources, the 10 relevance items are the 10 highest-scoring direct items, in score order, from both
// sources, and no explore pick is included.
func Test792a21d1b_MergeQueryPayloads_ExplorePicksDoNotCrowdOutDirectMatches(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		limit        = 10
		directCount  = 15
		exploreCount = 12
	)

	local := queryPayload{ModelID: ps1LocalModel}
	parent := queryPayload{ModelID: ps1ParentModel}

	// Even indexes come from the local vault and odd ones from the parent; a higher index scores higher.
	for index := range directCount {
		item := ps1Direct(fmt.Sprintf("direct-%02d.md", index), 0.5+0.2*float32(index)/(directCount-1))
		ps1Spread(&local, &parent, index, item)
	}

	for index := range exploreCount {
		item := ps1Explore(fmt.Sprintf("explore-%02d.md", index), 0.8+0.1*float32(index)/(exploreCount-1))
		ps1Spread(&local, &parent, index, item)
	}

	merged := mergeQueryPayloads(local, parent, nil, QueryArgs{Limit: limit})

	want := make([]string, 0, limit)
	for index := directCount - 1; index >= directCount-limit; index-- {
		want = append(want, fmt.Sprintf("direct-%02d.md", index))
	}

	g.Expect(itemPaths(merged.Items)).To(Equal(want), "the 10 highest-scoring direct items, highest first")

	sources := make(map[string]bool, 2)

	for _, item := range merged.Items {
		g.Expect(item.Provenances).NotTo(ContainElement(provenanceExplore), "no explore pick: %s", item.Path)
		sources[ps1Tag(merged.Items, item.Path)] = true
	}

	g.Expect(sources).To(HaveLen(2), "the ten are drawn from both sources")
}

// Test792a21d1c_MergeQueryPayloads_TheNoteFloorSurvivesTheMergedCap verifies 792a21d1c (The note floor survives the
// merged cap): with --limit 5, 5 direct chunk items scoring 0.9 and 3 direct note items scoring 0.5, the 5 relevance
// items hold all 3 notes and the 2 highest-scoring chunks.
func Test792a21d1c_MergeQueryPayloads_TheNoteFloorSurvivesTheMergedCap(t *testing.T) {
	t.Parallel()

	t.Run("chunks of equal score keep the first two", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		local := queryPayload{ModelID: ps1LocalModel, Items: []queryItem{
			ps1Chunk("chunk-1", 0.9, ""), ps1Chunk("chunk-2", 0.9, ""), ps1Chunk("chunk-3", 0.9, ""),
			ps1Chunk("chunk-4", 0.9, ""), ps1Chunk("chunk-5", 0.9, ""),
			ps1Direct("note-1.md", 0.5),
		}}
		parent := queryPayload{ModelID: ps1ParentModel, Items: []queryItem{
			ps1Direct("note-2.md", 0.5), ps1Direct("note-3.md", 0.5),
		}}

		merged := mergeQueryPayloads(local, parent, nil, QueryArgs{Limit: 5})

		g.Expect(ps1Head(itemPaths(merged.Items), 2)).To(Equal([]string{"chunk-1", "chunk-2"}))
		g.Expect(itemPaths(merged.Items)).To(ConsistOf(
			"chunk-1", "chunk-2", "note-1.md", "note-2.md", "note-3.md"), "all three notes are kept")
	})

	t.Run("the two highest-scoring chunks are the ones kept", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		local := queryPayload{ModelID: ps1LocalModel, Items: []queryItem{
			ps1Chunk("chunk-low-1", 0.91, ""), ps1Chunk("chunk-high-2", 0.95, ""),
			ps1Chunk("chunk-low-3", 0.92, ""), ps1Chunk("chunk-high-1", 0.97, ""),
			ps1Chunk("chunk-low-2", 0.93, ""),
			ps1Direct("note-1.md", 0.5), ps1Direct("note-2.md", 0.5), ps1Direct("note-3.md", 0.5),
		}}

		merged := mergeQueryPayloads(local, queryPayload{ModelID: ps1ParentModel}, nil, QueryArgs{Limit: 5})

		g.Expect(ps1Head(itemPaths(merged.Items), 2)).To(Equal([]string{"chunk-high-1", "chunk-high-2"}))
		g.Expect(itemPaths(merged.Items)).To(ConsistOf(
			"chunk-high-1", "chunk-high-2", "note-1.md", "note-2.md", "note-3.md"), "all three notes are kept")
	})
}

// Test792a21e1a_MergeQueryPayloads_ContentBudgetCapsFullContentChunksAcrossTheMergedSet verifies 792a21e1a
// (content-budget caps full-content chunk items across the merged set): with --content-budget 2, at most two chunk
// items render full content by rank, the lower-ranked chunk items render as a snippet, every chunk item is local,
// and the parent's top-scoring chunk neither appears nor uses up a full-content slot.
func Test792a21e1a_MergeQueryPayloads_ContentBudgetCapsFullContentChunksAcrossTheMergedSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	long := strings.Repeat("a long chunk line\n", 40)
	local := queryPayload{ModelID: ps1LocalModel, Items: []queryItem{
		ps1Chunk("local-1", 0.9, long), ps1Chunk("local-2", 0.8, long),
		ps1Chunk("local-3", 0.7, long), ps1Chunk("local-4", 0.6, long),
	}}
	parent := queryPayload{ModelID: ps1ParentModel, Items: []queryItem{
		ps1Chunk("parent-chunk", 0.99, long), ps1Direct("parent-note.md", 0.1),
	}}

	merged := mergeQueryPayloads(local, parent, nil, QueryArgs{ContentBudget: 2})

	g.Expect(itemPaths(merged.Items)).To(Equal(
		[]string{"local-1", "local-2", "local-3", "local-4", "parent-note.md"}), "no parent chunk")

	g.Expect(merged.Items[0].Content).To(Equal(long))
	g.Expect(merged.Items[1].Content).To(Equal(long))
	g.Expect(merged.Items[2].Content).To(Equal(snippet(long)))
	g.Expect(merged.Items[3].Content).To(Equal(snippet(long)))
	g.Expect(merged.Items[2].Content).NotTo(Equal(long))

	for _, item := range merged.Items {
		if item.Kind == chunkItemKind {
			g.Expect(item.FromParent).To(HaveValue(BeFalse()), "chunk %s is local", item.Path)
		}
	}
}

// Test792a21e1b_MergeQueryPayloads_RecentFillCapsTheRecencyChannelAcrossTheMergedSet verifies 792a21e1b
// (recent-fill caps the recency channel across the merged set): with --recent-fill 2, the recency channel holds the
// two newest local chunk items, none from the parent, and not 2 from each source.
func Test792a21e1b_MergeQueryPayloads_RecentFillCapsTheRecencyChannelAcrossTheMergedSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// each source lists its recency chunks newest first
	local := queryPayload{ModelID: ps1LocalModel, Items: []queryItem{
		ps1Direct("local-note.md", 0.4),
		ps1Recent("local-newest"), ps1Recent("local-newer"), ps1Recent("local-older"), ps1Recent("local-oldest"),
	}}
	parent := queryPayload{ModelID: ps1ParentModel, Items: []queryItem{
		ps1Recent("parent-newest"), ps1Recent("parent-newer"), ps1Recent("parent-older"),
	}}

	merged := mergeQueryPayloads(local, parent, nil, QueryArgs{RecentFill: 2})

	recency := make([]queryItem, 0, 2)

	for _, item := range merged.Items {
		if hasProvenance(item, provenanceRecent) {
			recency = append(recency, item)
		}
	}

	g.Expect(itemPaths(recency)).To(Equal([]string{"local-newest", "local-newer"}))

	for _, item := range recency {
		g.Expect(item.FromParent).To(HaveValue(BeFalse()), "recency item %s is local", item.Path)
	}
}

// Test792a21e1c_MergeQueryPayloads_LimitCapsTheMergedRelevanceChannel verifies 792a21e1c (limit caps the merged
// relevance channel): with --limit 3 the relevance items total 3, the three highest scores of the combined set and
// not 3 from each source, and the two recency items are not counted against the 3.
func Test792a21e1c_MergeQueryPayloads_LimitCapsTheMergedRelevanceChannel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const limit = 3

	local := queryPayload{ModelID: ps1LocalModel, Items: []queryItem{
		ps1Direct("local-1.md", 0.9), ps1Direct("local-2.md", 0.6),
		ps1Direct("local-3.md", 0.3), ps1Direct("local-4.md", 0.2),
		ps1Recent("recent-1"), ps1Recent("recent-2"),
	}}
	parent := queryPayload{ModelID: ps1ParentModel, Items: []queryItem{
		ps1Direct("parent-1.md", 0.8), ps1Direct("parent-2.md", 0.7), ps1Direct("parent-3.md", 0.5),
	}}

	merged := mergeQueryPayloads(local, parent, nil, QueryArgs{Limit: limit})

	g.Expect(itemPaths(merged.Items)).To(Equal(
		[]string{"local-1.md", "parent-1.md", "parent-2.md", "recent-1", "recent-2"}),
		"the top 3 of the combined set, then the recency channel")

	relevance := 0

	for _, item := range merged.Items {
		if !hasProvenance(item, provenanceRecent) {
			relevance++
		}
	}

	g.Expect(relevance).To(Equal(limit))
}

// Test792a21h1a_MergeQueryPayloads_NoCrossNodeClusterGrouping verifies 792a21h1a (No cross-node cluster
// grouping): the clusters of a merged payload are the local query's own, and no cluster has a member that is a
// parent-origin item.
func Test792a21h1a_MergeQueryPayloads_NoCrossNodeClusterGrouping(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	local := queryPayload{
		ModelID: ps1LocalModel,
		Items:   []queryItem{ps1Direct("local-a.md", 0.9), ps1Direct("local-b.md", 0.8)},
		Clusters: []queryCluster{{
			ID: 0, Phrase: "local phrase", Size: 2,
			Members: []queryClusterMember{{Path: "local-a.md"}, {Path: "local-b.md"}},
		}},
	}
	parent := queryPayload{
		ModelID: ps1ParentModel,
		Items:   []queryItem{ps1Direct("parent-a.md", 0.85), ps1Direct("parent-b.md", 0.7)},
		Clusters: []queryCluster{{
			ID: 0, Phrase: "parent phrase", Size: 2,
			Members: []queryClusterMember{{Path: "parent-a.md"}, {Path: "parent-b.md"}},
		}},
	}

	merged := mergeQueryPayloads(local, parent, nil, QueryArgs{})

	g.Expect(merged.Clusters).To(Equal(local.Clusters), "the clusters are the local query's")
	g.Expect(itemPaths(merged.Items)).To(ContainElements("parent-a.md", "parent-b.md"), "parent items are present")

	for _, cluster := range merged.Clusters {
		for _, member := range cluster.Members {
			g.Expect(ps1Tag(merged.Items, member.Path)).To(Equal("local"),
				"cluster %q holds no parent-origin item: %s", cluster.Phrase, member.Path)
		}
	}
}

// Test792a21i1a_MergeQueryPayloads_TheBudgetBlockIsPopulated verifies 792a21i1a (The budget block is populated):
// for --limit 5 --content-budget 2 --lazy-chunks the merged payload's budget shows limit 5, content_budget 2,
// lazy_chunks true and the snippeted-chunk count the merge produced, with explore_allocated summed per term over
// both sources.
func Test792a21i1a_MergeQueryPayloads_TheBudgetBlockIsPopulated(t *testing.T) {
	t.Parallel()

	t.Run("with lazy chunks", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		local, parent := ps1BudgetPayloads()

		merged := mergeQueryPayloads(local, parent, nil, QueryArgs{Limit: 5, ContentBudget: 2, LazyChunks: true})

		g.Expect(merged.Budget.Limit).To(Equal(5))
		g.Expect(merged.Budget.ContentBudget).To(Equal(2))
		g.Expect(merged.Budget.LazyChunks).To(BeTrue())
		g.Expect(merged.Budget.ChunksSnippeted).To(BeZero(), "lazy mode clears chunk content and snippets none")
		g.Expect(merged.Budget.ExploreAllocated).To(Equal(map[string]int{"alpha": 1, "beta": 5, "gamma": 4}))

		printed, marshalErr := yaml.Marshal(merged.Budget)
		g.Expect(marshalErr).NotTo(HaveOccurred())
		g.Expect(string(printed)).To(ContainSubstring("limit: 5\n"))
		g.Expect(string(printed)).To(ContainSubstring("content_budget: 2\n"))
		g.Expect(string(printed)).To(ContainSubstring("lazy_chunks: true\n"))
	})

	t.Run("without lazy chunks the content pass counts its snippets", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		local, parent := ps1BudgetPayloads()

		merged := mergeQueryPayloads(local, parent, nil, QueryArgs{Limit: 5, ContentBudget: 2})

		g.Expect(merged.Budget.Limit).To(Equal(5))
		g.Expect(merged.Budget.ContentBudget).To(Equal(2))
		g.Expect(merged.Budget.LazyChunks).To(BeFalse())
		g.Expect(merged.Budget.ChunksSnippeted).To(Equal(2), "four local chunks, two full, two snippeted")
	})
}

// Test792a21j1b_MergeQueryPayloads_ACoveredLinkDedupes verifies 792a21j1b (A covered link dedupes): local note L
// carries a covered link to parent note P2 in addition to its primary link; P2 does not appear and L appears once.
func Test792a21j1b_MergeQueryPayloads_ACoveredLinkDedupes(t *testing.T) {
	t.Parallel()

	frontmatter := "parent:\n    vault: " + ps1ParentVault + "\n    links:\n" +
		"        - note: 1100.primary\n          via: offered\n" +
		"        - note: 812.covered\n          via: covered\n"

	t.Run("L ranked locally", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linked := dedupeNote(t, "5.local", frontmatter, "body")

		merged := mergeQueryPayloads(
			queryPayload{ModelID: ps1LocalModel, Items: []queryItem{ps1Direct("5.local.md", 0.5)}},
			queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{
				ps1Direct("1100.primary.md", 0.7), ps1Direct("812.covered.md", 0.6),
			}},
			[]exchangeNote{linked}, QueryArgs{})

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{"5.local.md"}), "L once, neither P nor P2")
	})

	t.Run("only P2 returned and L did not rank", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linked := dedupeNote(t, "5.local", frontmatter, "body")

		merged := mergeQueryPayloads(
			queryPayload{ModelID: ps1LocalModel, Items: []queryItem{ps1Direct("8.other.md", 0.9)}},
			queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{
				ps1Direct("812.covered.md", 0.6),
			}},
			[]exchangeNote{linked}, QueryArgs{})

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{"8.other.md", "5.local.md"}),
			"L stands in for P2, once, and P2 is gone")
	})
}

// Test792a21j1c_MergeQueryPayloads_ALinkResolvedThroughAnAlias verifies 792a21j1c (A link resolved through an
// alias): L's link names basename O and the parent's note E lists O in its aliases, so the merged payload keeps L
// and drops E; without the alias E stays.
func Test792a21j1c_MergeQueryPayloads_ALinkResolvedThroughAnAlias(t *testing.T) {
	t.Parallel()

	const (
		oldName = "12.old-name"
		newPath = "1101.new-name.md"
	)

	t.Run("E lists O among its aliases", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linked := dedupeNote(t, "5.local", linkFrontmatter(ps1ParentVault, oldName, "offered"), "body")
		renamed := ps1Direct(newPath, 0.6)
		renamed.Aliases = []string{oldName}

		merged := mergeQueryPayloads(
			queryPayload{ModelID: ps1LocalModel, Items: []queryItem{ps1Direct("5.local.md", 0.5)}},
			queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{renamed}},
			[]exchangeNote{linked}, QueryArgs{})

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{"5.local.md"}), "L kept, E dropped")
	})

	t.Run("E does not list O among its aliases", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linked := dedupeNote(t, "5.local", linkFrontmatter(ps1ParentVault, oldName, "offered"), "body")

		merged := mergeQueryPayloads(
			queryPayload{ModelID: ps1LocalModel, Items: []queryItem{ps1Direct("5.local.md", 0.5)}},
			queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{
				ps1Direct(newPath, 0.6),
			}},
			[]exchangeNote{linked}, QueryArgs{})

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{newPath, "5.local.md"}), "no alias, no link, no merge")
	})
}

// Test792a21j1d_MergeQueryPayloads_IdenticalContentWithoutALink verifies 792a21j1d (Identical content without a
// link): an unlinked live local note and a parent note with equal exchange hashes appear once, as the local note;
// a parent note with no exchange hash is never matched by content.
func Test792a21j1d_MergeQueryPayloads_IdenticalContentWithoutALink(t *testing.T) {
	t.Parallel()

	t.Run("equal exchange hashes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		unlinked := dedupeNote(t, "5.local", "", "shared body")
		twin := ps1Direct("900.parent.md", 0.6)
		twin.ExchangeHash = mustExchangeHash(t, unlinked.raw)

		merged := mergeQueryPayloads(
			queryPayload{ModelID: ps1LocalModel, Items: []queryItem{ps1Direct("5.local.md", 0.5)}},
			queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{twin}},
			[]exchangeNote{unlinked}, QueryArgs{})

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{"5.local.md"}))
		g.Expect(ps1Tag(merged.Items, "5.local.md")).To(Equal("local"))
	})

	t.Run("a parent note without an exchange hash", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		unlinked := dedupeNote(t, "5.local", "", "shared body")

		merged := mergeQueryPayloads(
			queryPayload{ModelID: ps1LocalModel, Items: []queryItem{ps1Direct("5.local.md", 0.5)}},
			queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{
				ps1Direct("900.parent.md", 0.6),
			}},
			[]exchangeNote{unlinked}, QueryArgs{})

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{"900.parent.md", "5.local.md"}))
	})
}

// Test792a21j1e_MergeQueryPayloads_APendingPulledCopyDoesNotHideTheParentNote verifies 792a21j1e (A pending pulled
// copy does not hide the parent note): the only local copy of parent note P is a pending pull-down, so P still
// appears, with from_parent true, and the pending copy is not substituted for it.
func Test792a21j1e_MergeQueryPayloads_APendingPulledCopyDoesNotHideTheParentNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pending := dedupeNote(t, "5.pulled",
		linkFrontmatter(ps1ParentVault, "900.parent", "pulled")+"pending: true\n", "body")
	original := ps1Direct("900.parent.md", 0.6)
	original.ExchangeHash = mustExchangeHash(t, pending.raw)

	merged := mergeQueryPayloads(
		queryPayload{ModelID: ps1LocalModel},
		queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{original}},
		[]exchangeNote{pending}, QueryArgs{})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"900.parent.md"}))
	g.Expect(ps1Tag(merged.Items, "900.parent.md")).To(Equal("parent"))
}

// Test792a21j1f_MergeQueryPayloads_TheLocalCopyIsSubstitutedWhenItDidNotRankLocally verifies 792a21j1f (The local
// copy is substituted when it did not rank locally): P matches linked local note L and L is not among the local
// items, so an item for L stands at P's position with P's score and provenances, L's content, the local model_id
// and from_parent false; P's source_term is kept only for an explore pick.
func Test792a21j1f_MergeQueryPayloads_TheLocalCopyIsSubstitutedWhenItDidNotRankLocally(t *testing.T) {
	t.Parallel()

	t.Run("a direct item", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linked := dedupeNote(t, "5.local", linkFrontmatter(ps1ParentVault, "900.parent", "offered"),
			"see [[7.other]] for more")
		original := queryItem{
			Path: "900.parent.md", Kind: typeFeedback, Score: 0.6, Content: "parent content",
			Provenances: []string{provenanceDirect, provenanceClusterRep}, SourceTerm: "vocab/ignored",
		}

		merged := mergeQueryPayloads(
			queryPayload{ModelID: ps1LocalModel, Items: []queryItem{
				ps1Direct("8.ranked.md", 0.7), ps1Direct("9.ranked.md", 0.3),
			}},
			queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{original}},
			[]exchangeNote{linked}, QueryArgs{})

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{"8.ranked.md", "5.local.md", "9.ranked.md"}),
			"L stands where P would have")

		substitute := merged.Items[1]
		g.Expect(substitute.Score).To(Equal(float32(0.6)), "P's score")
		g.Expect(substitute.Provenances).To(Equal([]string{provenanceDirect, provenanceClusterRep}), "P's provenances")
		g.Expect(substitute.Content).To(ContainSubstring("see 7.other for more"), "L's content, not P's")
		g.Expect(substitute.Content).NotTo(ContainSubstring("parent content"))
		g.Expect(substitute.ModelID).To(Equal(ps1LocalModel))
		g.Expect(substitute.FromParent).To(HaveValue(BeFalse()))
		g.Expect(substitute.SourceTerm).To(BeEmpty(), "a direct item has no source_term")
	})

	t.Run("an explore pick keeps its source_term", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linked := dedupeNote(t, "5.local", linkFrontmatter(ps1ParentVault, "900.parent", "offered"), "body")
		pick := ps1Explore("900.parent.md", 0.8)
		pick.SourceTerm = "vocab/term"

		merged := mergeQueryPayloads(
			queryPayload{ModelID: ps1LocalModel},
			queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{pick}},
			[]exchangeNote{linked}, QueryArgs{})

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{"5.local.md"}))
		g.Expect(merged.Items[0].SourceTerm).To(Equal("vocab/term"))
		g.Expect(merged.Items[0].Provenances).To(Equal([]string{provenanceExplore}))
	})
}

// Test792a21j1g_MergeQueryPayloads_NearDuplicatesAreLeftAlone verifies 792a21j1g (Near-duplicates are left alone):
// a local note and a parent note with different content and no link both appear, each under its own origin.
func Test792a21j1g_MergeQueryPayloads_NearDuplicatesAreLeftAlone(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	localNote := dedupeNote(t, "5.local", "", "almost the same body")
	nearTwin := dedupeNote(t, "6.twin", "", "almost the same body!")
	parentItem := ps1Direct("900.parent.md", 0.6)
	parentItem.ExchangeHash = mustExchangeHash(t, nearTwin.raw)

	merged := mergeQueryPayloads(
		queryPayload{ModelID: ps1LocalModel, Items: []queryItem{ps1Direct("5.local.md", 0.5)}},
		queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{parentItem}},
		[]exchangeNote{localNote}, QueryArgs{})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"900.parent.md", "5.local.md"}))
	g.Expect(ps1Tag(merged.Items, "900.parent.md")).To(Equal("parent"))
	g.Expect(ps1Tag(merged.Items, "5.local.md")).To(Equal("local"))
}

// Test792a21j1h_MergeQueryPayloads_ALinkUnderAnotherParentVaultIsIgnored verifies 792a21j1h (A link under another
// parent vault is ignored): a local note linked to a parent vault other than the one that answered is treated as
// unlinked, so the parent's note still appears; the same link under the answering vault dedupes.
func Test792a21j1h_MergeQueryPayloads_ALinkUnderAnotherParentVaultIsIgnored(t *testing.T) {
	t.Parallel()

	merge := func(t *testing.T, linkVault, answeringVault string) queryPayload {
		t.Helper()

		linked := dedupeNote(t, "5.local", linkFrontmatter(linkVault, "900.parent", "offered"), "body")

		return mergeQueryPayloads(
			queryPayload{ModelID: ps1LocalModel, Items: []queryItem{ps1Direct("5.local.md", 0.5)}},
			queryPayload{ModelID: ps1ParentModel, VaultID: answeringVault, Items: []queryItem{
				ps1Direct("900.parent.md", 0.6),
			}},
			[]exchangeNote{linked}, QueryArgs{})
	}

	t.Run("the link names another vault", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		merged := merge(t, ps1OtherVault, ps1ParentVault)

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{"900.parent.md", "5.local.md"}))
		g.Expect(ps1Tag(merged.Items, "900.parent.md")).To(Equal("parent"))
	})

	t.Run("the parent reported no vault ID", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		merged := merge(t, ps1ParentVault, "")

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{"900.parent.md", "5.local.md"}))
	})

	t.Run("the link names the answering vault", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		merged := merge(t, ps1ParentVault, ps1ParentVault)

		g.Expect(itemPaths(merged.Items)).To(Equal([]string{"5.local.md"}))
	})
}

// Test792a21j1i_MergeQueryPayloads_TheDedupeKeysAreNotInThePayload verifies 792a21j1i (The dedupe keys are not in
// the payload): the printed merged payload carries no exchange_hash, aliases or parent vault_id, whether a parent
// item was kept, dropped or replaced by a local copy.
func Test792a21j1i_MergeQueryPayloads_TheDedupeKeysAreNotInThePayload(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	linked := dedupeNote(t, "5.local", linkFrontmatter(ps1ParentVault, "901.replaced", "offered"), "body")

	kept := ps1Direct("900.kept.md", 0.7)
	kept.ExchangeHash = "xh1:kept"
	kept.Aliases = []string{"12.kept-old"}

	replaced := ps1Direct("901.replaced.md", 0.6)
	replaced.ExchangeHash = "xh1:replaced"
	replaced.Aliases = []string{"13.replaced-old"}

	merged := mergeQueryPayloads(
		queryPayload{ModelID: ps1LocalModel},
		queryPayload{ModelID: ps1ParentModel, VaultID: ps1ParentVault, Items: []queryItem{kept, replaced}},
		[]exchangeNote{linked}, QueryArgs{})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"900.kept.md", "5.local.md"}), "a kept and a replaced item")

	printed, marshalErr := yaml.Marshal(merged)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	for _, key := range []string{"exchange_hash", "aliases", "vault_id", "xh1:", "12.kept-old"} {
		g.Expect(string(printed)).NotTo(ContainSubstring(key))
	}
}

// unexported constants.
const (
	ps1LocalModel  = "ps1-local-model"
	ps1OtherVault  = "0123456789abcdef0123456789abcdef"
	ps1ParentModel = "ps1-parent-model"
	ps1ParentVault = "9a1e9a1e9a1e9a1e9a1e9a1e9a1e9a1e"
)

// ps1BudgetPayloads is a local answer of four chunks with per-term explore counts and a parent answer with more.
func ps1BudgetPayloads() (queryPayload, queryPayload) {
	long := strings.Repeat("a long chunk line\n", 40)

	local := queryPayload{
		ModelID: ps1LocalModel,
		Items: []queryItem{
			ps1Chunk("chunk-1", 0.9, long), ps1Chunk("chunk-2", 0.8, long),
			ps1Chunk("chunk-3", 0.7, long), ps1Chunk("chunk-4", 0.6, long),
		},
		Budget: queryBudget{ExploreAllocated: map[string]int{"alpha": 1, "beta": 2}},
	}
	parent := queryPayload{
		ModelID: ps1ParentModel,
		Items:   []queryItem{ps1Direct("parent-note.md", 0.5)},
		Budget:  queryBudget{ExploreAllocated: map[string]int{"beta": 3, "gamma": 4}},
	}

	return local, parent
}

// ps1Chunk is a direct chunk item with the given content.
func ps1Chunk(path string, score float32, content string) queryItem {
	return queryItem{
		Path: path, Kind: chunkItemKind, Score: score, Content: content, Provenances: []string{provenanceDirect},
	}
}

// ps1Direct is a direct note item.
func ps1Direct(path string, score float32) queryItem {
	return queryItem{Path: path, Kind: typeFact, Score: score, Provenances: []string{provenanceDirect}}
}

// ps1Explore is an explore pick sampled under a vocabulary term.
func ps1Explore(path string, score float32) queryItem {
	return queryItem{
		Path: path, Kind: typeFact, Score: score, Provenances: []string{provenanceExplore}, SourceTerm: "vocab/ps1",
	}
}

// ps1Head is the first count entries of paths.
func ps1Head(paths []string, count int) []string {
	return paths[:min(count, len(paths))]
}

// ps1Recent is a recency-channel chunk item.
func ps1Recent(path string) queryItem {
	return queryItem{Path: path, Kind: chunkItemKind, Provenances: []string{provenanceRecent}}
}

// ps1Spread appends item to the local payload for an even index and to the parent payload for an odd one.
func ps1Spread(local, parent *queryPayload, index int, item queryItem) {
	if index%2 == 0 {
		local.Items = append(local.Items, item)

		return
	}

	parent.Items = append(parent.Items, item)
}

// ps1Tag reads which node produced the merged item with the given path: "local", "parent", or "" when the item is
// absent or untagged.
func ps1Tag(items []queryItem, path string) string {
	for _, item := range items {
		if item.Path != path || item.FromParent == nil {
			continue
		}

		if *item.FromParent {
			return "parent"
		}

		return "local"
	}

	return ""
}

// ps1Trigger is a runbook that a trigger phrase of the message hit.
func ps1Trigger(path string) queryItem {
	return queryItem{Path: path, Kind: typeRunbook, Provenances: []string{provenanceTrigger}}
}
