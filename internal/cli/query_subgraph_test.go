package cli_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// TestQuery_BacklinksTraversed: note A links to B; query surfaces B → subgraph includes A via backlink.
func TestQuery_BacklinksTraversed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// B is the direct hit; A links to B but B does not link to A. Backlink expansion should pull A in.
	plantNoteWithSidecar(t, memFS, vault, "Permanent/A.md",
		"---\ntype: fact\n---\nunrelated\n[[B]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/B.md",
		"---\ntype: fact\n---\nthe query string body\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"the query string body"}, VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	// Subgraph includes B (direct hit) + A (via backlink).
	g.Expect(parsed.Budget.SubgraphSize).To(Equal(2))
}

// TestQuery_ExpandsSubgraph_BFS3Hops verifies the BFS expansion stage.
// Synthetic graph: query matches direct hit "A"; A→B→C→D→E (chain).
// Expansion stops at 3 hops, so {A, B, C, D} are in the subgraph and E is not.
func TestQuery_ExpandsSubgraph_BFS3Hops(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Direct hit will be A; it's the only note whose body matches the query.
	plantNoteWithSidecar(t, memFS, vault, "Permanent/A.md",
		"---\ntype: fact\n---\nthe query string body\n[[B]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/B.md",
		"---\ntype: fact\n---\nzzz\n[[C]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/C.md",
		"---\ntype: fact\n---\nyyy\n[[D]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/D.md",
		"---\ntype: fact\n---\nxxx\n[[E]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/E.md",
		"---\ntype: fact\n---\nwww\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"the query string body"}, VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	// Subgraph: A (depth 0) + B (1) + C (2) + D (3) = 4 notes. E is at depth 4.
	g.Expect(parsed.Budget.SubgraphSize).To(Equal(4))
	g.Expect(parsed.Budget.HopsTraversed).To(BeNumerically("<=", 3))
	g.Expect(parsed.Budget.SubgraphSizeCapped).To(BeFalse())
}

// TestQuery_SubgraphCap_StopsAtThreshold: a synthetic dense graph that
// would exceed the 200-note cap; budget signals the cap.
func TestQuery_SubgraphCap_StopsAtThreshold(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// 250 notes: hub H links to all others; all others link back to H.
	// The query matches H. From H: depth-1 neighbors = 249 nodes; capped to 200.
	const total = 250

	const hubMatch = "the query string body"

	plantNoteWithSidecar(t, memFS, vault, "Permanent/H.md",
		"---\ntype: fact\n---\n"+hubMatch+"\n"+buildLinkBlock(total))

	for i := range total {
		plantNoteWithSidecar(t, memFS, vault,
			"Permanent/"+nodeBase(i)+".md",
			"---\ntype: fact\n---\nleaf body\n[[H]]\n")
	}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{hubMatch}, VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	g.Expect(parsed.Budget.SubgraphSize).To(Equal(200))
	g.Expect(parsed.Budget.SubgraphSizeCapped).To(BeTrue())
}

// unexported variables.
var (
	_ = filepath.Join
)

// queryParsed is the YAML parse target for new-payload tests in this file.
type queryParsed struct {
	Version int      `yaml:"version"`
	Phrases []string `yaml:"phrases"`
	Items   []struct {
		Path        string   `yaml:"path"`
		Kind        string   `yaml:"kind"`
		Score       float32  `yaml:"score"`
		Provenances []string `yaml:"provenances"`
		ClusterID   *int     `yaml:"cluster_id,omitempty"`
		InDegree    *int     `yaml:"in_degree,omitempty"`
		Content     string   `yaml:"content"`
	} `yaml:"items"`
	Clusters []struct {
		ID         int     `yaml:"id"`
		Phrase     string  `yaml:"phrase"`
		Size       int     `yaml:"size"`
		Silhouette float64 `yaml:"silhouette"`
		Members    []struct {
			Path             string  `yaml:"path"`
			Score            float32 `yaml:"score"`
			IsRepresentative bool    `yaml:"is_representative"`
		} `yaml:"members"`
		NearestL3 *struct {
			Path   string  `yaml:"path"`
			Cosine float32 `yaml:"cosine"`
		} `yaml:"nearest_l3,omitempty"`
	} `yaml:"clusters"`
	Budget struct {
		PhrasesQueried       int  `yaml:"phrases_queried"`
		TotalNotes           int  `yaml:"total_notes"`
		WithEmbeddings       int  `yaml:"with_embeddings"`
		SubgraphSize         int  `yaml:"subgraph_size"`
		SubgraphSizeCapped   bool `yaml:"subgraph_size_capped"`
		HopsTraversed        int  `yaml:"hops_traversed"`
		ClustersFound        int  `yaml:"clusters_found"`
		HubsReturned         int  `yaml:"hubs_returned"`
		DirectHitsReturned   int  `yaml:"direct_hits_returned"`
		ItemsWithFullContent int  `yaml:"items_with_full_content"`
		Limit                int  `yaml:"limit"`
	} `yaml:"budget"`
}

// buildLinkBlock returns "[[N0]]\n[[N1]]\n..." for n nodes.
func buildLinkBlock(n int) string {
	const linkOverhead = 6 // "[[", "]]", "\n"

	sb := make([]byte, 0, n*linkOverhead)

	for i := range n {
		sb = append(sb, '[', '[')
		sb = append(sb, nodeBase(i)...)
		sb = append(sb, ']', ']', '\n')
	}

	return string(sb)
}

// nodeBase returns a unique short basename for synthetic notes.
func nodeBase(idx int) string {
	const offsetA = 'A'

	const wrap = 26

	first := byte(offsetA + (idx/wrap)%wrap)
	second := byte(offsetA + idx%wrap)
	third := byte(offsetA + (idx/(wrap*wrap))%wrap)

	return string([]byte{third, first, second})
}
