package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// TestClearNoteContent_TableDriven verifies the Variant B (#684) rule: every
// note item (Kind != "chunk") loses its Content, chunk items are untouched,
// and the returned count matches exactly how many note items actually had
// content withheld (no-silent-caps rule — mirrors TagNominationsAdded).
func TestClearNoteContent_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		kinds     []string
		contents  []string
		wantOut   []string
		wantCount int
	}{
		{
			name:      "note content cleared, chunk content kept",
			kinds:     []string{"fact", "feedback", "chunk"},
			contents:  []string{"note one body", "note two body", "chunk stays full"},
			wantOut:   []string{"", "", "chunk stays full"},
			wantCount: 2,
		},
		{
			name:      "unknown kind is still a note, not a chunk",
			kinds:     []string{"unknown"},
			contents:  []string{"body"},
			wantOut:   []string{""},
			wantCount: 1,
		},
		{
			name:      "already-empty note content is not double counted",
			kinds:     []string{"fact"},
			contents:  []string{""},
			wantOut:   []string{""},
			wantCount: 0,
		},
		{
			name:      "all chunks: zero notes deduped",
			kinds:     []string{"chunk", "chunk"},
			contents:  []string{"c1", "c2"},
			wantOut:   []string{"c1", "c2"},
			wantCount: 0,
		},
		{
			name:      "empty input",
			kinds:     nil,
			contents:  nil,
			wantOut:   []string{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			out, count := cli.ExportClearNoteContent(tt.kinds, tt.contents)

			g.Expect(out).To(Equal(tt.wantOut))
			g.Expect(count).To(Equal(tt.wantCount))
		})
	}
}

// TestRenderQueryPayloadBudget_ItemsContentWithheldCounts is the RED test for
// the budget half of the no-silent-caps rule (TagNominationsAdded precedent):
// the payload budget must report how many note items' content was withheld.
func TestRenderQueryPayloadBudget_ItemsContentWithheldCounts(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	kinds := []string{"fact", "feedback", "chunk"}
	contents := []string{"note one body", "note two body", "chunk body"}

	out, err := cli.ExportRenderQueryPayloadBudget(kinds, contents, false, 0)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("items_content_withheld: 2"),
		"budget must report how many note items' content was withheld")
}

// TestRenderQueryPayloadBudget_ItemsContentWithheldOmittedWhenZero verifies the
// omitempty contract: no matched notes → the field is omitted entirely.
func TestRenderQueryPayloadBudget_ItemsContentWithheldOmittedWhenZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	out, err := cli.ExportRenderQueryPayloadBudget([]string{"chunk"}, []string{"chunk body"}, false, 0)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).NotTo(ContainSubstring("items_content_withheld"),
		"omitempty: zero notes matched, so the field must not render")
}

// TestRenderQueryPayload_ClustersRenderBeforeItems verifies the Variant B
// struct-order swap: clusters: must appear before items: in the rendered
// top-level YAML (yaml.v3 renders struct fields in declared order).
func TestRenderQueryPayload_ClustersRenderBeforeItems(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	out, err := cli.ExportRenderQueryPayloadBudget([]string{"fact"}, []string{"body"}, false, 0)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	clustersIdx := strings.Index(out, "\nclusters:")
	itemsIdx := strings.Index(out, "\nitems:")

	g.Expect(clustersIdx).To(BeNumerically(">", -1), "clusters: must render")
	g.Expect(itemsIdx).To(BeNumerically(">", -1), "items: must render")
	g.Expect(clustersIdx).To(BeNumerically("<", itemsIdx),
		"Variant B (#684): clusters render before items — struct order swap")
}

// TestRenderQueryPayload_NoteItemsContentFree_BothModes verifies that note
// items are content-free REGARDLESS of --lazy-chunks — the restructure is the
// payload's shape, not a lazy-chunks opt-in — while chunk lazy/cap semantics
// are unchanged (chunk cleared under lazy, capped-then-snippeted otherwise).
func TestRenderQueryPayload_NoteItemsContentFree_BothModes(t *testing.T) {
	t.Parallel()

	kinds := []string{"fact", "chunk"}
	contents := []string{"note body must never render", "chunk body"}

	tests := []struct {
		name       string
		lazyChunks bool
	}{
		{name: "lazy chunks on", lazyChunks: true},
		{name: "lazy chunks off (non-lazy)", lazyChunks: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			out, err := cli.ExportRenderQueryPayloadBudget(kinds, contents, tt.lazyChunks, 0)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(out).NotTo(ContainSubstring("note body must never render"),
				"note content must never render in items[], lazy or not (#684 Variant B)")
		})
	}
}

// TestRunQuery_NoteItemContentFreeCandidateL2sIntact is the end-to-end proof:
// a real RunQuery for a single note asserts items[0].content is empty while
// the SAME note's content rides inline, wikilink-stripped, in the cluster's
// candidate_l2s — Step 2.5's per-cluster nomination keeps content local to
// the payload; items[] never carries it (#684 Variant B).
func TestRunQuery_NoteItemContentFreeCandidateL2sIntact(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	body := "---\ntype: fact\n---\n" +
		"See [[1a.foo]] and [[2b.bar|the bar note]] for context.\n"

	plantNoteWithSidecar(t, memFS, vault, "1.foo.md", body)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"context"}, VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed struct {
		Items []struct {
			Path    string `yaml:"path"`
			Content string `yaml:"content"`
		} `yaml:"items"`
		Clusters []struct {
			CandidateL2s []struct {
				Path    string `yaml:"path"`
				Content string `yaml:"content"`
			} `yaml:"candidate_l2s"`
		} `yaml:"clusters"`
		Budget struct {
			ItemsContentWithheld int `yaml:"items_content_withheld"`
		} `yaml:"budget"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).To(HaveLen(1))
	g.Expect(parsed.Items[0].Content).To(BeEmpty(),
		"note items render path-only under Variant B (#684); content rides in candidate_l2s")

	// The same note's content rides inline in the cluster's candidate_l2s,
	// wikilinks stripped exactly as items[] content used to be.
	var candidateContent string

	for _, c := range parsed.Clusters {
		for _, cand := range c.CandidateL2s {
			if cand.Path == "1.foo.md" {
				candidateContent = cand.Content
			}
		}
	}

	g.Expect(candidateContent).NotTo(BeEmpty(), "the note must appear as a candidate_l2 with content")
	g.Expect(candidateContent).NotTo(ContainSubstring("[["))
	g.Expect(candidateContent).NotTo(ContainSubstring("]]"))
	g.Expect(candidateContent).To(ContainSubstring("See 1a.foo and the bar note for context."))

	g.Expect(parsed.Budget.ItemsContentWithheld).To(Equal(1),
		"budget must count the one note item whose content was withheld")
}
