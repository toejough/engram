package cli_test

// Scenario tests of the query-payload-renderer component (docs/specs/technical/query-payload-renderer.md): the
// order of the document's keys and the figures of the budget block. Helpers carry the prefix qpr1. The renderer
// is a pure function of its result, so each scenario drives it through the exported render seam (or RunQuery over
// the in-memory vault) and reads the encoded YAML; no collaborator is mocked.

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

// Test792b23c1a4_RenderQueryPayload_TheKeysComeInOrder verifies 792b23c1a4: a result with items and clusters
// renders version, phrases, items, clusters, budget and model_id as the document's top-level keys, in that order.
func Test792b23c1a4_RenderQueryPayload_TheKeysComeInOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	out := qpr1QueryWithItemsAndClusters(t)

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out, &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).NotTo(BeEmpty(), "the result must carry items")
	g.Expect(parsed.Clusters).NotTo(BeEmpty(), "the result must carry clusters")

	var document yaml.Node

	g.Expect(yaml.Unmarshal(out, &document)).NotTo(HaveOccurred())
	g.Expect(document.Content).NotTo(BeEmpty())

	if len(document.Content) == 0 {
		return
	}

	mapping := document.Content[0]
	keys := make([]string, 0, len(mapping.Content)/2)

	for index := 0; index < len(mapping.Content); index += 2 {
		keys = append(keys, mapping.Content[index].Value)
	}

	g.Expect(keys).To(Equal([]string{"version", "phrases", "items", "clusters", "budget", "model_id"}))
}

// Test792b23c6a2_RenderQueryPayload_CountsFollowWhatIsPrinted verifies 792b23c6a2: two direct notes, a direct
// chunk within the budget and a direct chunk reduced to a snippet (plus an explore note without content) render
// direct_hits_returned as the four direct items, items_with_full_content as the three that carry content less
// the snippet's, and chunks_snippeted as 1.
func Test792b23c6a2_RenderQueryPayload_CountsFollowWhatIsPrinted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		contentBudget      = 1
		wantDirectHits     = 4
		wantFullContent    = 3
		wantChunksSnipped  = 1
		snippetRepeatCount = 300
	)

	direct := []string{"direct"}
	long := "first line\nsecond line " + strings.Repeat("x", snippetRepeatCount)
	items := []cli.ExportResolvedItem{
		cli.ExportQpr1NewResolvedItem("1.first.md", "", "---\ntype: fact\n---\nfirst note\n", direct),
		cli.ExportQpr1NewResolvedItem("2.second.md", "", "---\ntype: fact\n---\nsecond note\n", direct),
		cli.ExportQpr1NewResolvedItem("a.jsonl#turn-1", "chunk", "kept chunk", direct),
		cli.ExportQpr1NewResolvedItem("a.jsonl#turn-2", "chunk", long, direct),
		cli.ExportQpr1NewResolvedItem("3.explore.md", "", "", []string{"explore"}),
	}

	out, err := cli.ExportQpr1RenderQueryPayloadItems(items, contentBudget)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed struct {
		Items  []queryParsedItem `yaml:"items"`
		Budget struct {
			DirectHitsReturned   int `yaml:"direct_hits_returned"`
			ItemsWithFullContent int `yaml:"items_with_full_content"`
			ChunksSnippeted      int `yaml:"chunks_snippeted"`
		} `yaml:"budget"`
	}

	g.Expect(yaml.Unmarshal([]byte(out), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).To(HaveLen(len(items)))
	g.Expect(parsed.Budget.DirectHitsReturned).To(Equal(wantDirectHits), "only the direct items count")
	g.Expect(parsed.Budget.ItemsWithFullContent).To(Equal(wantFullContent), "the snippet is left out")
	g.Expect(parsed.Budget.ChunksSnippeted).To(Equal(wantChunksSnipped))
}

// Test792b23c6a3_RenderQueryPayload_TheAllocationIsAlwaysPresent verifies 792b23c6a3: a result whose explore
// allocation is empty, as an empty map or as none at all, still renders explore_allocated: {} in the budget.
func Test792b23c6a3_RenderQueryPayload_TheAllocationIsAlwaysPresent(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]int{
		"an empty map": {},
		"no map":       nil,
	}

	for name, allocated := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			out, err := cli.ExportRenderQueryPayloadExploreAllocated(allocated)
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			var parsed struct {
				Budget map[string]any `yaml:"budget"`
			}

			g.Expect(yaml.Unmarshal([]byte(out), &parsed)).NotTo(HaveOccurred())
			g.Expect(parsed.Budget).To(HaveKeyWithValue("explore_allocated", BeEmpty()))
			g.Expect(out).To(ContainSubstring("explore_allocated: {}"))
		})
	}
}

// qpr1QueryWithItemsAndClusters runs a query over an in-memory vault holding one note and two neighborhoods of
// indexed chunks, and returns the encoded payload, which carries both items and clusters.
func qpr1QueryWithItemsAndClusters(t *testing.T) []byte {
	t.Helper()

	const (
		chunkNeighborhoods = 4
		vectorSpread       = 0.01
		queryLimit         = 20
		chunksDir          = "/chunks"
		indexPath          = "/chunks/s1.jsonl"
		chunkSource        = "/s/a.jsonl"
	)

	g := NewWithT(t)
	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "1.linting.md",
		"---\ntype: fact\ntier: L2\n---\nAlways run the linter before committing changes.\n")

	records := make([]chunk.Record, 0, 2*chunkNeighborhoods)

	for index := range chunkNeighborhoods {
		suffix := string(rune('a' + index))
		spread := vectorSpread * float32(index)

		records = append(records, chunk.Record{
			Source: chunkSource, Anchor: "lint-" + suffix,
			ContentHash: chunk.HashText("lint" + suffix),
			Text:        "reviewer flagged linter convention variant " + suffix,
			Vector:      []float32{1, spread, 0, 0},
		}, chunk.Record{
			Source: chunkSource, Anchor: "feed-" + suffix,
			ContentHash: chunk.HashText("feed" + suffix),
			Text:        "feed parsing discussion variant " + suffix,
			Vector:      []float32{0, 1, spread, 0},
		})
	}

	data, err := chunk.EncodeRecords(records)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		t.FailNow()
	}

	memFS.files[indexPath] = data

	var out bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"linter conventions"}, VaultPath: vault, Limit: queryLimit, ChunksDir: chunksDir},
		unifiedQueryDeps(memFS, indexPath), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		t.FailNow()
	}

	return out.Bytes()
}
