package cli_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// TestClearDedupedNoteContent_ClearsCandidatesKeepsOthersAndChunks verifies the
// pure post-process: a note item whose path matches a candidate_l2s path is
// cleared; a non-candidate note item keeps its content; a chunk item is never
// cleared even when its path happens to match a candidate path (Kind guard,
// not incidental non-collision); the returned count reflects only the notes
// actually cleared.
func TestClearDedupedNoteContent_ClearsCandidatesKeepsOthersAndChunks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	paths := []string{"a.md", "b.md", "chunk-x"}
	kinds := []string{"fact", "feedback", "chunk"}
	contents := []string{"content A", "content B", "chunk content"}
	candidatePaths := []string{"a.md", "chunk-x"}

	out, count := cli.ExportClearDedupedNoteContent(paths, kinds, contents, candidatePaths)

	g.Expect(out[0]).To(Equal(""), "a.md is a candidate — content must be cleared")
	g.Expect(out[1]).To(Equal("content B"), "b.md is not a candidate — content must survive")
	g.Expect(out[2]).To(Equal("chunk content"), "chunk items must never be cleared, even on a path match")
	g.Expect(count).To(Equal(1), "only the one note actually cleared is counted")
}

// TestClearDedupedNoteContent_NoCandidatesIsNoOp verifies an empty candidate
// set leaves every item's content untouched and reports zero cleared.
func TestClearDedupedNoteContent_NoCandidatesIsNoOp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	paths := []string{"a.md", "b.md"}
	kinds := []string{"fact", "feedback"}
	contents := []string{"content A", "content B"}

	out, count := cli.ExportClearDedupedNoteContent(paths, kinds, contents, nil)

	g.Expect(out).To(Equal(contents))
	g.Expect(count).To(Equal(0))
}

// TestRenderQueryPayload_BudgetDedupedCountMatchesClearedItems is a
// table-driven check that items_content_deduped always equals the number of
// items[] notes whose path collided with a candidate_l2s path.
func TestRenderQueryPayload_BudgetDedupedCountMatchesClearedItems(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		candidatePaths []string
		wantDeduped    int
	}{
		"no candidates":             {candidatePaths: nil, wantDeduped: 0},
		"one of two matches":        {candidatePaths: []string{"note-1.md"}, wantDeduped: 1},
		"both match":                {candidatePaths: []string{"note-1.md", "note-2.md"}, wantDeduped: 2},
		"candidate not among items": {candidatePaths: []string{"note-9.md"}, wantDeduped: 0},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			out, err := cli.ExportRenderQueryPayloadWithCandidates(
				[]string{"note-1.md", "note-2.md"},
				[]string{"fact", "fact"},
				[]string{"---\ntype: fact\n---\nbody one\n", "---\ntype: fact\n---\nbody two\n"},
				tc.candidatePaths,
			)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			var parsed queryParsed

			unmarshalErr := yaml.Unmarshal([]byte(out), &parsed)
			g.Expect(unmarshalErr).NotTo(HaveOccurred())

			if unmarshalErr != nil {
				return
			}

			g.Expect(parsed.Budget.ItemsContentDeduped).To(Equal(tc.wantDeduped))
		})
	}
}

// TestRenderQueryPayload_ClustersRenderBeforeItems verifies the Variant-A
// struct-order swap: "clusters:" precedes "items:" in the marshaled YAML.
func TestRenderQueryPayload_ClustersRenderBeforeItems(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	out, err := cli.ExportRenderQueryPayloadWithCandidates(
		[]string{"a.md"}, []string{"fact"}, []string{"---\ntype: fact\n---\nbody\n"}, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	clustersIdx := strings.Index(out, "clusters:")
	itemsIdx := strings.Index(out, "items:")

	g.Expect(clustersIdx).To(BeNumerically(">=", 0), "clusters: must be present")
	g.Expect(itemsIdx).To(BeNumerically(">=", 0), "items: must be present")
	g.Expect(clustersIdx).To(BeNumerically("<", itemsIdx), "clusters must render before items")
}

// TestRenderQueryPayload_DedupesCandidateNoteContentPreservesKind is the
// end-to-end RED test for the Variant-A dedupe: a note item whose path is in
// a cluster's candidate_l2s renders with empty content while Kind stays
// derived from the ORIGINAL content (the kindFromContent trap — Kind is
// computed once in renderItems, before this post-process runs, so clearing
// content afterward must never corrupt it to "unknown"); a non-candidate note
// item keeps its content and kind; the budget reports the correct
// deduped count.
func TestRenderQueryPayload_DedupesCandidateNoteContentPreservesKind(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	contentA := "---\ntype: fact\n---\nCluster candidate note body.\n"
	contentB := "---\ntype: feedback\n---\nNon-candidate note body.\n"
	expectedKindA := cli.ExportKindFromContent(contentA)
	expectedKindB := cli.ExportKindFromContent(contentB)

	g.Expect(expectedKindA).To(Equal("fact"), "sanity: fixture content derives to fact")
	g.Expect(expectedKindB).To(Equal("feedback"), "sanity: fixture content derives to feedback")

	out, err := cli.ExportRenderQueryPayloadWithCandidates(
		[]string{"note-a.md", "note-b.md"},
		[]string{"", ""},
		[]string{contentA, contentB},
		[]string{"note-a.md"},
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	unmarshalErr := yaml.Unmarshal([]byte(out), &parsed)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())

	if unmarshalErr != nil {
		return
	}

	itemA, foundA := findItemByPath(parsed, "note-a.md")
	itemB, foundB := findItemByPath(parsed, "note-b.md")

	g.Expect(foundA).To(BeTrue(), "note-a.md must be present in items[]")
	g.Expect(foundB).To(BeTrue(), "note-b.md must be present in items[]")

	if !foundA || !foundB {
		return
	}

	g.Expect(itemA.Content).To(Equal(""), "candidate note must render content-free")
	g.Expect(itemA.Kind).To(Equal(expectedKindA), "Kind must survive content-clearing (the kindFromContent trap)")

	g.Expect(itemB.Content).To(Equal(contentB), "non-candidate note must keep its content")
	g.Expect(itemB.Kind).To(Equal(expectedKindB), "non-candidate Kind unaffected")

	g.Expect(parsed.Budget.ItemsContentDeduped).To(Equal(1), "budget must count exactly the one cleared item")
}
