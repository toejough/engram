package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// TestDedupeProperty_NeverBothLiveLocalAndMatchedParent: whatever the local
// notes and parent items, the merged output never holds a live local note
// together with a parent item that note matches (vault-merged-recall).
func TestDedupeProperty_NeverBothLiveLocalAndMatchedParent(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		notes, localItems := drawLocalNotes(rt)
		parentItems := drawParentItems(rt)
		// The vault ID the parent reports in this query: the one local
		// links were recorded under, another vault's, or none (a parent
		// too old to report it).
		reportedID := rapid.SampledFrom([]string{testParentVaultID, testOtherVaultID, ""}).Draw(rt, "reportedID")

		merged := mergeQueryPayloads(
			queryPayload{ModelID: testLocalModel, Items: localItems},
			queryPayload{ModelID: testParentModel, VaultID: reportedID, Items: parentItems},
			notes, QueryArgs{Limit: -1})

		for _, outParent := range merged.Items {
			if !outParent.FromParent {
				continue
			}

			// Parent paths are unique, so the path recovers the item as
			// sent, with the dedupe keys the output strips.
			sent := parentItems[slices.IndexFunc(parentItems, func(item queryItem) bool {
				return item.Path == outParent.Path
			})]

			for _, note := range liveNotesInOutput(merged.Items, notes) {
				if sameNoteByD4(sent, note, reportedID) {
					rt.Fatalf("output holds live local %s and the parent item %s it matches",
						note.basename, outParent.Path)
				}
			}
		}
	})
}

// TestMergeQueryPayloads_AliasLinkDedupes: L links basename O, and the
// parent's note E lists O among its aliases — E is dropped, L kept.
func TestMergeQueryPayloads_AliasLinkDedupes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	local := dedupeNote(t, "5.local", linkFrontmatter(testParentVaultID, "12.old-name", "offered"), "body")

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: []queryItem{
			{Path: "5.local.md", Kind: typeFact, Score: 0.5, Provenances: []string{provenanceDirect}},
		}},
		queryPayload{ModelID: testParentModel, VaultID: testParentVaultID, Items: []queryItem{
			{
				Path: "1101.new-name.md", Kind: typeFact, Score: 0.6,
				Provenances: []string{provenanceDirect}, Aliases: []string{"12.old-name"},
			},
		}},
		[]exchangeNote{local}, QueryArgs{})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"5.local.md"}))
}

// TestMergeQueryPayloads_BudgetBlockReportsMergeValues: the budget block
// carries the values the merge applied (#743).
func TestMergeQueryPayloads_BudgetBlockReportsMergeValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	long := "line one\n" + strings.Repeat("x", 300)

	merged := mergeQueryPayloads(
		queryPayload{
			ModelID: testLocalModel,
			Budget:  queryBudget{ExploreAllocated: map[string]int{"term-a": 2, "term-b": 1}},
			Items: []queryItem{
				{Path: "c1", Kind: chunkItemKind, Score: 0.9, Provenances: []string{provenanceDirect}, Content: long},
				{Path: "c2", Kind: chunkItemKind, Score: 0.8, Provenances: []string{provenanceDirect}, Content: long},
				{Path: "c3", Kind: chunkItemKind, Score: 0.7, Provenances: []string{provenanceDirect}, Content: long},
			},
		},
		queryPayload{
			ModelID: testParentModel,
			Budget:  queryBudget{ExploreAllocated: map[string]int{"term-a": 3, "term-c": 4}},
		},
		nil, QueryArgs{Limit: 5, ContentBudget: 2})

	g.Expect(merged.Budget.Limit).To(Equal(5))
	g.Expect(merged.Budget.ContentBudget).To(Equal(2))
	g.Expect(merged.Budget.LazyChunks).To(BeFalse())
	g.Expect(merged.Budget.ChunksSnippeted).To(Equal(1))
	g.Expect(merged.Budget.ExploreAllocated).To(Equal(map[string]int{"term-a": 5, "term-b": 1, "term-c": 4}))
}

// TestMergeQueryPayloads_BudgetBlockUnderLazyChunks mirrors the spec
// scenario `--limit 5 --content-budget 2 --lazy-chunks`: lazy mode clears
// chunk content, so nothing is snippeted, and the block says so.
func TestMergeQueryPayloads_BudgetBlockUnderLazyChunks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: []queryItem{
			{Path: "c1", Kind: chunkItemKind, Score: 0.9, Provenances: []string{provenanceDirect}, Content: "text"},
		}},
		queryPayload{ModelID: testParentModel},
		nil, QueryArgs{Limit: 5, ContentBudget: 2, LazyChunks: true})

	g.Expect(merged.Budget.Limit).To(Equal(5))
	g.Expect(merged.Budget.ContentBudget).To(Equal(2))
	g.Expect(merged.Budget.LazyChunks).To(BeTrue())
	g.Expect(merged.Budget.ChunksSnippeted).To(BeZero())
	g.Expect(merged.Budget.ExploreAllocated).NotTo(BeNil())
	g.Expect(merged.Items[0].Content).To(BeEmpty())
}

// TestMergeQueryPayloads_CoveredLinkDedupes: a covered link dedupes a
// second parent note, and L appears once.
func TestMergeQueryPayloads_CoveredLinkDedupes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	frontmatter := "parent:\n    vault: " + testParentVaultID + "\n    links:\n" +
		"        - note: 1100.primary\n          via: offered\n" +
		"        - note: 812.covered\n          via: covered\n"
	local := dedupeNote(t, "5.local", frontmatter, "body")

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel},
		queryPayload{ModelID: testParentModel, VaultID: testParentVaultID, Items: []queryItem{
			{Path: "1100.primary.md", Kind: typeFact, Score: 0.7, Provenances: []string{provenanceDirect}},
			{Path: "812.covered.md", Kind: typeFact, Score: 0.6, Provenances: []string{provenanceDirect}},
		}},
		[]exchangeNote{local}, QueryArgs{})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"5.local.md"}))
}

// TestMergeQueryPayloads_DirectBeforeExplore: explore picks never crowd
// out direct matches under --limit (#744, spec scenario: 12 explore picks
// at 0.8–0.9, 15 direct items at 0.5–0.7, --limit 10).
func TestMergeQueryPayloads_DirectBeforeExplore(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		exploreCount = 12
		directCount  = 15
		limit        = 10
	)

	localItems := make([]queryItem, 0, exploreCount+directCount)
	parentItems := make([]queryItem, 0, directCount)

	for index := range exploreCount {
		localItems = append(localItems, queryItem{
			Path: fmt.Sprintf("explore-%d.md", index), Kind: typeFact,
			Score: 0.8 + float32(index)/100, Provenances: []string{provenanceExplore}, SourceTerm: "t",
		})
	}

	for index := range directCount {
		item := queryItem{
			Path: fmt.Sprintf("direct-%02d.md", index), Kind: typeFact,
			Score: 0.5 + float32(index)/100, Provenances: []string{provenanceDirect},
		}

		if index%2 == 0 {
			parentItems = append(parentItems, item)
		} else {
			localItems = append(localItems, item)
		}
	}

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: localItems},
		queryPayload{ModelID: testParentModel, Items: parentItems},
		nil, QueryArgs{Limit: limit})

	g.Expect(merged.Items).To(HaveLen(limit))

	for index, item := range merged.Items {
		g.Expect(item.Provenances).NotTo(ContainElement(provenanceExplore))
		g.Expect(item.Path).To(Equal(fmt.Sprintf("direct-%02d.md", directCount-1-index)))
	}
}

// TestMergeQueryPayloads_DropsParentChunks: every parent chunk item is
// dropped, including the parent's recency channel (Q1); parent notes and
// local chunks stay.
func TestMergeQueryPayloads_DropsParentChunks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: []queryItem{
			{Path: "local-chunk", Kind: chunkItemKind, Score: 0.4, Provenances: []string{provenanceDirect}},
			{Path: "local-recent", Kind: chunkItemKind, Provenances: []string{provenanceRecent}},
		}},
		queryPayload{ModelID: testParentModel, Items: []queryItem{
			{Path: "parent-chunk", Kind: chunkItemKind, Score: 0.9, Provenances: []string{provenanceDirect}},
			{Path: "parent-recent", Kind: chunkItemKind, Provenances: []string{provenanceRecent}},
			{Path: "parent-note.md", Kind: typeFact, Score: 0.5, Provenances: []string{provenanceDirect}},
		}},
		nil, QueryArgs{})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"parent-note.md", "local-chunk", "local-recent"}))
}

// TestMergeQueryPayloads_EqualHashDedupes: an unlinked live local note
// whose exchange hash equals the parent item's is the same note.
func TestMergeQueryPayloads_EqualHashDedupes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	local := dedupeNote(t, "5.local", "", "shared body")

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: []queryItem{
			{Path: "5.local.md", Kind: typeFact, Score: 0.5, Provenances: []string{provenanceDirect}},
		}},
		queryPayload{ModelID: testParentModel, VaultID: testParentVaultID, Items: []queryItem{
			{
				Path: "900.parent.md", Kind: typeFact, Score: 0.6, Provenances: []string{provenanceDirect},
				ExchangeHash: mustExchangeHash(t, local.raw),
			},
		}},
		[]exchangeNote{local}, QueryArgs{})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"5.local.md"}))
}

// TestMergeQueryPayloads_FloorOnIssueShape: the #744 reproduction shape —
// 10 phrases' worth of direct notes and chunks, explore picks at 0.75–0.89
// — delivers direct items first and keeps the note floor under the default
// --limit.
func TestMergeQueryPayloads_FloorOnIssueShape(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		exploreCount = 44
		chunkCount   = 40
		noteCount    = 8
	)

	localItems := make([]queryItem, 0, exploreCount+chunkCount)
	parentItems := make([]queryItem, 0, exploreCount+noteCount)

	// Explore picks from both sources, on the centroid-cosine scale
	// (0.75–0.89) that outranks every direct match.
	for index := range exploreCount {
		pick := queryItem{
			Path: fmt.Sprintf("explore-%02d.md", index), Kind: typeFeedback,
			Score:       0.75 + float32(index%15)/100,
			Provenances: []string{provenanceExplore}, SourceTerm: "term",
		}

		if index%2 == 0 {
			parentItems = append(parentItems, pick)
		} else {
			localItems = append(localItems, pick)
		}
	}

	for index := range chunkCount {
		localItems = append(localItems, queryItem{
			Path: fmt.Sprintf("chunk-%02d", index), Kind: chunkItemKind,
			Score: 0.72 - float32(index)/1000, Provenances: []string{provenanceDirect},
		})
	}

	for index := range noteCount {
		parentItems = append(parentItems, queryItem{
			Path: fmt.Sprintf("note-%d.md", index), Kind: typeFeedback,
			Score: 0.66 - float32(index)/100, Provenances: []string{provenanceDirect},
		})
	}

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: localItems},
		queryPayload{ModelID: testParentModel, Items: parentItems},
		nil, QueryArgs{})

	g.Expect(merged.Items).To(HaveLen(defaultQueryLimit))

	notes := 0

	for _, item := range merged.Items {
		g.Expect(item.Provenances).To(ContainElement(provenanceDirect))

		if item.Kind != chunkItemKind {
			notes++
		}
	}

	g.Expect(notes).To(Equal(noteFloorK))
	g.Expect(scoresDescending(merged.Items)).To(BeTrue())
}

// TestMergeQueryPayloads_LinkUnderOtherVaultIgnored: a link recorded under
// a vault ID other than the one the parent reported in this query never
// dedupes (S16).
func TestMergeQueryPayloads_LinkUnderOtherVaultIgnored(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	local := dedupeNote(t, "5.local", linkFrontmatter(testOtherVaultID, "900.parent", "offered"), "body")

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: []queryItem{
			{Path: "5.local.md", Kind: typeFact, Score: 0.5, Provenances: []string{provenanceDirect}},
		}},
		queryPayload{ModelID: testParentModel, VaultID: testParentVaultID, Items: []queryItem{
			{Path: "900.parent.md", Kind: typeFact, Score: 0.6, Provenances: []string{provenanceDirect}},
		}},
		[]exchangeNote{local}, QueryArgs{})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"900.parent.md", "5.local.md"}))
}

// TestMergeQueryPayloads_NearDuplicatesLeftAlone: different content and no
// link — both appear; the query never judges similarity.
func TestMergeQueryPayloads_NearDuplicatesLeftAlone(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	local := dedupeNote(t, "5.local", "", "almost the same body")
	other := dedupeNote(t, "6.other", "", "almost the same body!")

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: []queryItem{
			{Path: "5.local.md", Kind: typeFact, Score: 0.5, Provenances: []string{provenanceDirect}},
		}},
		queryPayload{ModelID: testParentModel, VaultID: testParentVaultID, Items: []queryItem{
			{
				Path: "900.parent.md", Kind: typeFact, Score: 0.6, Provenances: []string{provenanceDirect},
				ExchangeHash: mustExchangeHash(t, other.raw),
			},
		}},
		[]exchangeNote{local}, QueryArgs{})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"900.parent.md", "5.local.md"}))
}

// TestMergeQueryPayloads_NoteFloorSkipsLowScoringNotes: a direct note
// below the 0.25 relevance floor is not promoted.
func TestMergeQueryPayloads_NoteFloorSkipsLowScoringNotes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: []queryItem{
			{Path: "c1", Kind: chunkItemKind, Score: 0.9, Provenances: []string{provenanceDirect}},
			{Path: "c2", Kind: chunkItemKind, Score: 0.8, Provenances: []string{provenanceDirect}},
			{Path: "low.md", Kind: typeFact, Score: 0.2, Provenances: []string{provenanceDirect}},
		}},
		queryPayload{ModelID: testParentModel},
		nil, QueryArgs{Limit: 2})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"c1", "c2"}))
}

// TestMergeQueryPayloads_NoteFloorSurvivesCap is the spec scenario: 5
// direct chunks at 0.9 and 3 direct notes at 0.5 under --limit 5 keep all
// 3 notes and the 2 highest-scoring chunks, in score order (M3).
func TestMergeQueryPayloads_NoteFloorSurvivesCap(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	localItems := []queryItem{
		{Path: "c1", Kind: chunkItemKind, Score: 0.95, Provenances: []string{provenanceDirect}},
		{Path: "c2", Kind: chunkItemKind, Score: 0.94, Provenances: []string{provenanceDirect}},
		{Path: "c3", Kind: chunkItemKind, Score: 0.93, Provenances: []string{provenanceDirect}},
		{Path: "c4", Kind: chunkItemKind, Score: 0.92, Provenances: []string{provenanceDirect}},
		{Path: "c5", Kind: chunkItemKind, Score: 0.91, Provenances: []string{provenanceDirect}},
		{Path: "n1.md", Kind: typeFact, Score: 0.5, Provenances: []string{provenanceDirect}},
	}
	parentItems := []queryItem{
		{Path: "n2.md", Kind: typeFeedback, Score: 0.49, Provenances: []string{provenanceDirect}},
		{Path: "n3.md", Kind: typeRunbook, Score: 0.48, Provenances: []string{provenanceDirect}},
	}

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: localItems},
		queryPayload{ModelID: testParentModel, Items: parentItems},
		nil, QueryArgs{Limit: 5})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"c1", "c2", "n1.md", "n2.md", "n3.md"}))
}

// TestMergeQueryPayloads_PendingHintIsLocalOnly: the parent's pending flag
// is ignored; only local pending offers raise the hint (H4).
func TestMergeQueryPayloads_PendingHintIsLocalOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fromParent := mergeQueryPayloads(queryPayload{},
		queryPayload{PendingOffers: true, PendingOffersHint: pendingOfferCurateInstruction}, nil, QueryArgs{})
	g.Expect(fromParent.PendingOffers).To(BeFalse())
	g.Expect(fromParent.PendingOffersHint).To(BeEmpty())

	fromLocal := mergeQueryPayloads(queryPayload{PendingOffers: true}, queryPayload{}, nil, QueryArgs{})
	g.Expect(fromLocal.PendingOffers).To(BeTrue())
	g.Expect(fromLocal.PendingOffersHint).To(Equal(pendingOfferCurateInstruction))
}

// TestMergeQueryPayloads_PendingLocalCopyDoesNotSuppress: a pending
// pulled copy never hides the parent note it copies.
func TestMergeQueryPayloads_PendingLocalCopyDoesNotSuppress(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	frontmatter := linkFrontmatter(testParentVaultID, "900.parent", "pulled") + "pending: true\n"
	pendingCopy := dedupeNote(t, "5.pulled", frontmatter, "body")

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel},
		queryPayload{ModelID: testParentModel, VaultID: testParentVaultID, Items: []queryItem{
			{
				Path: "900.parent.md", Kind: typeFact, Score: 0.6, Provenances: []string{provenanceDirect},
				ExchangeHash: mustExchangeHash(t, pendingCopy.raw),
			},
		}},
		[]exchangeNote{pendingCopy}, QueryArgs{})

	g.Expect(merged.Items).To(HaveLen(1))
	g.Expect(merged.Items[0].Path).To(Equal("900.parent.md"))
	g.Expect(merged.Items[0].FromParent).To(BeTrue())
}

// TestMergeQueryPayloads_PrimaryLinkDedupes: a linked note that ranked
// locally appears once, as the local copy.
func TestMergeQueryPayloads_PrimaryLinkDedupes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	local := dedupeNote(t, "5.local", linkFrontmatter(testParentVaultID, "900.parent", "offered"), "body")

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: []queryItem{
			{Path: "5.local.md", Kind: typeFact, Score: 0.5, Provenances: []string{provenanceDirect}},
		}},
		queryPayload{ModelID: testParentModel, VaultID: testParentVaultID, Items: []queryItem{
			{Path: "900.parent.md", Kind: typeFact, Score: 0.6, Provenances: []string{provenanceDirect}},
		}},
		[]exchangeNote{local}, QueryArgs{})

	g.Expect(merged.Items).To(HaveLen(1))
	g.Expect(merged.Items[0].Path).To(Equal("5.local.md"))
	g.Expect(merged.Items[0].FromParent).To(BeFalse())
}

// TestMergeQueryPayloads_StripsDedupeKeys: exchange_hash and aliases never
// reach the merged payload, and neither does the parent's vault_id.
func TestMergeQueryPayloads_StripsDedupeKeys(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel},
		queryPayload{ModelID: testParentModel, VaultID: testParentVaultID, Items: []queryItem{
			{
				Path: "900.parent.md", Kind: typeFact, Score: 0.6, Provenances: []string{provenanceDirect},
				ExchangeHash: "xh1:abc", Aliases: []string{"12.old"},
			},
		}},
		nil, QueryArgs{})

	g.Expect(merged.VaultID).To(BeEmpty())
	g.Expect(merged.Items).To(HaveLen(1))
	g.Expect(merged.Items[0].ExchangeHash).To(BeEmpty())
	g.Expect(merged.Items[0].Aliases).To(BeNil())
}

// TestMergeQueryPayloads_SubstitutesExploreKeepsSourceTerm: a substituted
// explore pick keeps P's source_term and renders L as an explore pick does.
func TestMergeQueryPayloads_SubstitutesExploreKeepsSourceTerm(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	local := dedupeNote(t, "5.local", linkFrontmatter(testParentVaultID, "900.parent", "offered"),
		"see [[7.other]] for more")

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel},
		queryPayload{ModelID: testParentModel, VaultID: testParentVaultID, Items: []queryItem{
			{
				Path: "900.parent.md", Kind: typeFact, Score: 0.8,
				Provenances: []string{provenanceExplore}, SourceTerm: "vocab/term",
			},
		}},
		[]exchangeNote{local}, QueryArgs{})

	g.Expect(merged.Items).To(HaveLen(1))
	g.Expect(merged.Items[0].SourceTerm).To(Equal("vocab/term"))
	g.Expect(merged.Items[0].Content).To(Equal(stripWikilinks(string(local.raw))))
}

// TestMergeQueryPayloads_SubstitutesUnrankedLocalCopy: P matches L, and L
// did not rank locally — L takes P's place with exactly the M4 fields.
func TestMergeQueryPayloads_SubstitutesUnrankedLocalCopy(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	local := dedupeNote(t, "5.local", linkFrontmatter(testParentVaultID, "900.parent", "offered"),
		"see [[7.other]] for more")
	clusterID := 3

	merged := mergeQueryPayloads(
		queryPayload{ModelID: testLocalModel, Items: []queryItem{
			{Path: "8.ranked.md", Kind: typeFact, Score: 0.7, Provenances: []string{provenanceDirect}},
			{Path: "9.ranked.md", Kind: typeFact, Score: 0.3, Provenances: []string{provenanceDirect}},
		}},
		queryPayload{ModelID: testParentModel, VaultID: testParentVaultID, Items: []queryItem{
			{
				Path: "900.parent.md", Kind: typeFeedback, Score: 0.6, ClusterID: &clusterID,
				Provenances: []string{provenanceDirect, provenanceClusterRep}, SourceTerm: "ignored",
				Content: "parent content", ExchangeHash: "xh1:other", Aliases: []string{"1.a"},
			},
		}},
		[]exchangeNote{local}, QueryArgs{})

	g.Expect(itemPaths(merged.Items)).To(Equal([]string{"8.ranked.md", "5.local.md", "9.ranked.md"}))
	g.Expect(merged.Items[1]).To(Equal(queryItem{
		Path:        "5.local.md",
		Kind:        typeFact,
		Score:       0.6,
		Provenances: []string{provenanceDirect, provenanceClusterRep},
		Content:     capRedFlagsForPreview(stripWikilinks(string(local.raw))),
		ModelID:     testLocalModel,
		FromParent:  false,
	}))
}

// TestScanDedupeNotes_ScanFailureWarnsAndMergesWithoutDedupe: a failed
// local-notes scan warns and yields no dedupe candidates rather than
// failing the read-only query.
func TestScanDedupeNotes_ScanFailureWarnsAndMergesWithoutDedupe(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stderr bytes.Buffer

	notes := scanDedupeNotes(Deps{FS: failingListFS{}, Stderr: &stderr}, "/vault")

	g.Expect(notes).To(BeNil())
	g.Expect(stderr.String()).To(ContainSubstring("could not read local notes for dedupe"))
}

// unexported constants.
const (
	testLocalModel    = "local-model"
	testOtherVaultID  = "0123456789abcdef0123456789abcdef"
	testParentModel   = "parent-model"
	testParentVaultID = "9a1e9a1e9a1e9a1e9a1e9a1e9a1e9a1e"
)

// unexported variables.
var (
	errFailingList = errors.New("listing failed")
)

// failingListFS is an EdgeFS whose directory listing always fails.
type failingListFS struct{ EdgeFS }

func (failingListFS) ReadDir(string) ([]fs.DirEntry, error) { return nil, errFailingList }

// dedupeBodies is the pool of note bodies the property draws from, so
// equal hashes occur.
func dedupeBodies() []string { return []string{"alpha body", "beta body", "gamma body"} }

// dedupeNames is the pool of parent basenames the property draws from.
func dedupeNames() []string { return []string{"1.a", "2.b", "3.c", "4.d"} }

// dedupeNote builds a parsed local fact note for the dedupe tests.
func dedupeNote(t *testing.T, basename, frontmatter, body string) exchangeNote {
	t.Helper()

	note, ok := parseExchangeNote("/vault", basename+mdExt, dedupeRaw(frontmatter, body))
	if !ok {
		t.Fatalf("fixture %s did not parse", basename)
	}

	return note
}

// dedupeRapidNote is dedupeNote for a rapid run.
func dedupeRapidNote(rt *rapid.T, basename, frontmatter, body string) exchangeNote {
	note, ok := parseExchangeNote("/vault", basename+mdExt, dedupeRaw(frontmatter, body))
	if !ok {
		rt.Fatalf("fixture %s did not parse", basename)
	}

	return note
}

// dedupeRaw is a fact note's bytes; the situation is fixed so equal bodies
// give equal exchange hashes across basenames.
func dedupeRaw(frontmatter, body string) []byte {
	return []byte("---\ntype: fact\ntier: L2\nsituation: shared situation\n" + frontmatter + "---\n\n" + body + "\n")
}

// drawLocalNotes draws local notes (linked or not, under this or another
// parent vault ID, live or pending) and the live ones that ranked locally.
func drawLocalNotes(rt *rapid.T) ([]exchangeNote, []queryItem) {
	noteCount := rapid.IntRange(0, 4).Draw(rt, "noteCount")
	notes := make([]exchangeNote, 0, noteCount)
	localItems := make([]queryItem, 0, noteCount)

	for index := range noteCount {
		basename := fmt.Sprintf("10%d.local", index)
		frontmatter := ""

		if rapid.Bool().Draw(rt, "linked") {
			frontmatter += linkFrontmatter(
				rapid.SampledFrom([]string{testParentVaultID, testOtherVaultID}).Draw(rt, "linkVault"),
				rapid.SampledFrom(dedupeNames()).Draw(rt, "linkTo"),
				rapid.SampledFrom([]string{"offered", "pulled", "covered"}).Draw(rt, "via"))
		}

		if rapid.Bool().Draw(rt, "pending") {
			frontmatter += "pending: true\n"
		}

		note := dedupeRapidNote(rt, basename, frontmatter, rapid.SampledFrom(dedupeBodies()).Draw(rt, "body"))
		notes = append(notes, note)

		if !note.pending && rapid.Bool().Draw(rt, "ranked") {
			localItems = append(localItems, queryItem{Path: basename + mdExt, Kind: typeFact, Score: 0.4})
		}
	}

	return notes, localItems
}

// drawParentItems draws parent note items with unique paths and optional
// exchange hashes and aliases.
func drawParentItems(rt *rapid.T) []queryItem {
	names := dedupeNames()
	parentCount := rapid.IntRange(0, len(names)).Draw(rt, "parentCount")
	parentItems := make([]queryItem, 0, parentCount)

	for index := range parentCount {
		item := queryItem{
			Path:        names[index] + mdExt,
			Kind:        typeFact,
			Score:       float32(index+1) / 10,
			Provenances: []string{provenanceDirect},
		}

		if rapid.Bool().Draw(rt, "hashed") {
			body := rapid.SampledFrom(dedupeBodies()).Draw(rt, "parentBody")
			item.ExchangeHash = mustExchangeHash(rt, dedupeRaw("", body))
		}

		if rapid.Bool().Draw(rt, "aliased") {
			item.Aliases = []string{rapid.SampledFrom(names).Draw(rt, "alias")}
		}

		parentItems = append(parentItems, item)
	}

	return parentItems
}

// itemPaths lists the items' paths in order.
func itemPaths(items []queryItem) []string {
	paths := make([]string, len(items))
	for index, item := range items {
		paths[index] = item.Path
	}

	return paths
}

// linkFrontmatter is a parent block with one link.
func linkFrontmatter(vaultID, note, via string) string {
	return "parent:\n    vault: " + vaultID + "\n    links:\n        - note: " + note + "\n          via: " + via + "\n"
}

// liveNotesInOutput returns the live local notes that appear in items.
func liveNotesInOutput(items []queryItem, notes []exchangeNote) []exchangeNote {
	out := make([]exchangeNote, 0, len(notes))

	for _, note := range notes {
		if note.pending {
			continue
		}

		if slices.ContainsFunc(items, func(item queryItem) bool {
			return !item.FromParent && item.Path == note.basename+mdExt
		}) {
			out = append(out, note)
		}
	}

	return out
}

// mustExchangeHash hashes raw or fails the test.
func mustExchangeHash(t interface {
	Fatalf(format string, args ...any)
}, raw []byte) string {
	hash, err := exchangeHash(raw)
	if err != nil {
		t.Fatalf("hashing: %v", err)
	}

	return hash
}

// sameNoteByD4 is the property's oracle, stated as design D4 words it:
// parent item P and live local note L are the same note when (1) the parent
// reported a vault ID, L's parent.vault is that ID, and one of L's links
// names P's basename or one of P's aliases; or (2) both exchange hashes are
// known and equal.
func sameNoteByD4(parentItem queryItem, note exchangeNote, reportedID string) bool {
	parentBasename := strings.TrimSuffix(parentItem.Path, mdExt)

	if reportedID != "" && note.exchange.Parent.Vault == reportedID {
		for _, link := range note.exchange.Parent.Links {
			if link.Note == parentBasename || slices.Contains(parentItem.Aliases, link.Note) {
				return true
			}
		}
	}

	localHash, hashErr := exchangeHash(note.raw)

	return hashErr == nil && localHash != "" && parentItem.ExchangeHash != "" && localHash == parentItem.ExchangeHash
}

// scoresDescending reports whether items are in non-increasing score order.
func scoresDescending(items []queryItem) bool {
	return slices.IsSortedFunc(items, func(first, second queryItem) int {
		switch {
		case first.Score > second.Score:
			return -1
		case first.Score < second.Score:
			return 1
		default:
			return 0
		}
	})
}
