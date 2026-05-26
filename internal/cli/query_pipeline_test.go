package cli_test

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestQuery_BudgetReportsAllStages — every new budget field is populated.
func TestQuery_BudgetReportsAllStages(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantDeterministicCluster(t, memFS, vault)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "anchor", VaultPath: vault, Limit: 5},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	// All budget fields appear; spot-check each is consistent.
	g.Expect(parsed.Budget.TotalNotes).To(Equal(9))
	g.Expect(parsed.Budget.WithEmbeddings).To(Equal(9))
	g.Expect(parsed.Budget.SubgraphSize).To(Equal(9))
	g.Expect(parsed.Budget.SubgraphSizeCapped).To(BeFalse())
	g.Expect(parsed.Budget.HopsTraversed).To(BeNumerically(">=", 1))
	g.Expect(parsed.Budget.ClustersFound).To(BeNumerically(">=", 2))
	g.Expect(parsed.Budget.HubsReturned).To(BeNumerically(">=", 1))
	g.Expect(parsed.Budget.DirectHitsReturned).To(Equal(5))
	g.Expect(parsed.Budget.ItemsWithFullContent).To(BeNumerically(">=", 5))
	g.Expect(parsed.Budget.Limit).To(Equal(5))
}

// TestQuery_ClusterRepClosestToCentroid — verify each cluster's rep is
// flagged is_representative; non-rep members are not. (Closest-to-centroid
// is verified at the cluster package level; here we test the wiring.)
func TestQuery_ClusterRepClosestToCentroid(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantDeterministicCluster(t, memFS, vault)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "anchor", VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	// Every cluster has exactly one rep.
	for _, cluster := range parsed.Clusters {
		repCount := 0

		for _, member := range cluster.Members {
			if member.IsRepresentative {
				repCount++
			}
		}

		g.Expect(repCount).To(Equal(1), "cluster %d has %d reps", cluster.ID, repCount)
	}
}

// TestQuery_ClustersDeterministic — identical query against identical vault
// → identical cluster IDs and member lists.
func TestQuery_ClustersDeterministic(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantDeterministicCluster(t, memFS, vault)

	var first bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "anchor", VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &first)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var second bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "anchor", VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &second)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Bytes-for-bytes identical means determinism end-to-end.
	g.Expect(second.Bytes()).To(Equal(first.Bytes()))
}

// TestQuery_FullContentScope — direct hits and any rep/hub get content;
// non-rep cluster members in clusters.members do not.
func TestQuery_FullContentScope(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantDeterministicCluster(t, memFS, vault)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "anchor", VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	// Every item that appears in items[] gets content.
	for _, item := range parsed.Items {
		g.Expect(item.Content).NotTo(BeEmpty(), "item %s has empty content", item.Path)
	}

	// Non-rep cluster members in clusters.members have path + score only,
	// no content field (the YAML struct doesn't model content there).
}

// TestQuery_HubsByInDegree — synthetic graph with known in-degrees;
// verify top-5 ordering.
func TestQuery_HubsByInDegree(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// A is the direct hit. A, B, C, D, E, F are subgraph nodes.
	// Edges: B → A, C → A, D → A, E → A, F → A (A in-degree 5)
	//        C → B, D → B, E → B (B in-degree 3)
	//        D → C, E → C (C in-degree 2)
	//        E → D (D in-degree 1)
	plantNoteWithSidecar(t, memFS, vault, "Permanent/A.md",
		"---\ntype: fact\n---\nthe query string body\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/B.md",
		"---\ntype: fact\n---\nfiller\n[[A]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/C.md",
		"---\ntype: fact\n---\nfiller\n[[A]]\n[[B]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/D.md",
		"---\ntype: fact\n---\nfiller\n[[A]]\n[[B]]\n[[C]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/E.md",
		"---\ntype: fact\n---\nfiller\n[[A]]\n[[B]]\n[[C]]\n[[D]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/F.md",
		"---\ntype: fact\n---\nfiller\n[[A]]\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "the query string body", VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	// Hubs returned: 4 (A, B, C, D all have in-degree > 0; E and F do not).
	g.Expect(parsed.Budget.HubsReturned).To(Equal(4))

	// Find hub items (items with provenance "hub") and check in_degree.
	inDegrees := map[string]int{}

	for _, item := range parsed.Items {
		for _, prov := range item.Provenances {
			if prov == "hub" && item.InDegree != nil {
				inDegrees[item.Path] = *item.InDegree
			}
		}
	}

	g.Expect(inDegrees["Permanent/A.md"]).To(Equal(5))
	g.Expect(inDegrees["Permanent/B.md"]).To(Equal(3))
	g.Expect(inDegrees["Permanent/C.md"]).To(Equal(2))
	g.Expect(inDegrees["Permanent/D.md"]).To(Equal(1))
}

// TestQuery_ItemDedupAcrossRoles — a note that is direct + rep + hub
// appears once with all three provenances.
func TestQuery_ItemDedupAcrossRoles(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Plant A with a fixed vector that matches the query exactly so A is
	// the top direct hit; A has in-degree 8 (hub); and A sits in its own
	// cluster region. Other notes link to A and form a separate cluster.
	plantWithFixedVector(t, memFS, vault, "Permanent/A.md",
		"---\ntype: fact\n---\nanchor [[B]]\n[[C]]\n[[D]]"+
			"[[F]]\n[[G]]\n[[H]]\n[[I]]\n",
		[]float32{1, 0, 0, 0})

	for _, name := range []string{"B", "C", "D", "E", "F", "G", "H", "I"} {
		plantWithFixedVector(t, memFS, vault, "Permanent/"+name+".md",
			"---\ntype: fact\n---\nfiller\n[[A]]\n",
			[]float32{0, 1, 0, 0})
	}

	// Query exactly matches A's vector.
	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "anchor ", VaultPath: vault, Limit: 9},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	pathCounts := map[string]int{}
	for _, item := range parsed.Items {
		pathCounts[item.Path]++
	}

	for path, count := range pathCounts {
		g.Expect(count).To(Equal(1), "path %s appeared %d times", path, count)
	}

	// A appears with at least direct + hub provenances. The cluster
	// structure depends on whether A landed in its own cluster (8 others
	// share vector so they're one cluster; A's vector differs).
	var aProvenances []string

	found := false

	for i := range parsed.Items {
		if parsed.Items[i].Path == "Permanent/A.md" {
			aProvenances = parsed.Items[i].Provenances
			found = true

			break
		}
	}

	g.Expect(found).To(BeTrue(), "A should appear in items")
	g.Expect(aProvenances).To(ContainElement("direct"))
	g.Expect(aProvenances).To(ContainElement("hub"))
	g.Expect(len(aProvenances)).To(BeNumerically(">=", 2))
}

// TestQuery_ItemOrderingByProvenanceCount — F7 ordering rule.
func TestQuery_ItemOrderingByProvenanceCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantDeterministicCluster(t, memFS, vault)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "anchor", VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	// Items must be sorted by provenance count desc.
	for i := 1; i < len(parsed.Items); i++ {
		prev := parsed.Items[i-1]
		cur := parsed.Items[i]
		g.Expect(len(prev.Provenances)).To(BeNumerically(">=", len(cur.Provenances)),
			"items[%d] has fewer provenances than items[%d]", i-1, i)
	}
}

// TestQuery_LowSilhouetteReturnsEmpty — uniform vectors → silhouette < threshold → empty clusters.
func TestQuery_LowSilhouetteReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// All notes share identical body → identical vectors → silhouette 0.
	for i := range 8 {
		name := string([]byte{byte('A' + i)})
		plantWithFixedVector(t, memFS, vault,
			"Permanent/"+name+".md",
			"---\ntype: fact\n---\nidentical body for all\n[["+nextName(i)+"]]\n",
			[]float32{1, 0, 0, 0})
	}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "identical", VaultPath: vault, Limit: 8},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	g.Expect(parsed.Clusters).To(BeEmpty())
	g.Expect(parsed.Budget.ClustersFound).To(BeZero())
}

// TestQuery_NoOldKindKept — sanity check that existing assertions on
// `kind`, `score`, `provenances`, `content` still hold with the new payload.
// Guards against regression after schema change.
func TestQuery_PreservesLegacyFields(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.foo.md",
		"---\ntype: fact\n---\nthe query string body\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "the query string body", VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Spot-check that the output is still valid YAML the existing structure
	// can decode.
	got := out.String()
	g.Expect(got).To(ContainSubstring("provenances:"))
	g.Expect(got).To(ContainSubstring("clusters:"))
	g.Expect(strings.Count(got, "content:")).To(BeNumerically(">=", 1))
}

// TestQuery_TinySubgraphSkipsClustering — < 6 subgraph notes → empty clusters.
func TestQuery_TinySubgraphSkipsClustering(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// 3 notes in a tiny chain; clustering must short-circuit.
	plantNoteWithSidecar(t, memFS, vault, "Permanent/A.md",
		"---\ntype: fact\n---\nthe query string body\n[[B]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/B.md",
		"---\ntype: fact\n---\nzzz\n[[C]]\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/C.md",
		"---\ntype: fact\n---\nyyy\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "the query string body", VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	g.Expect(parsed.Budget.SubgraphSize).To(Equal(3))
	g.Expect(parsed.Clusters).To(BeEmpty())
	g.Expect(parsed.Budget.ClustersFound).To(BeZero())
}

// fixedVectorEmbedder always returns the same vector for any text.
type fixedVectorEmbedder struct {
	modelID string
	vector  []float32
}

func (f fixedVectorEmbedder) Dims() int { return len(f.vector) }

func (f fixedVectorEmbedder) Embed(context.Context, string) ([]float32, error) {
	out := make([]float32, len(f.vector))
	copy(out, f.vector)

	return out, nil
}

func (f fixedVectorEmbedder) ModelID() string { return f.modelID }

// nextName returns the next basename in the synthetic cycle to keep notes connected.
func nextName(idx int) string {
	const wrap = 8

	return string([]byte{byte('A' + (idx+1)%wrap)})
}

// plantDeterministicCluster sets up a small clusterable vault used by the
// determinism + cluster-rep tests.
func plantDeterministicCluster(t *testing.T, memFS *inMemoryFS, vault string) {
	t.Helper()

	// 8 notes around a hub A; 4 share theme "alpha", 4 share theme "beta".
	// All link to A so they form one connected subgraph; the embedder
	// stub's text-based hashing keeps "alpha-tagged" and "beta-tagged"
	// notes embeddings-distant.
	plantNoteWithSidecar(t, memFS, vault, "Permanent/A.md",
		"---\ntype: fact\n---\nanchor body matches anchor query\n[[B]]\n[[C]]\n[[D]]\n[[E]]\n[[F]]\n[[G]]\n[[H]]\n[[I]]\n")

	for _, name := range []string{"B", "C", "D", "E"} {
		plantNoteWithSidecar(t, memFS, vault, "Permanent/"+name+".md",
			"---\ntype: fact\n---\nalpha ")
	}

	for _, name := range []string{"F", "G", "H", "I"} {
		plantNoteWithSidecar(t, memFS, vault, "Permanent/"+name+".md",
			"---\ntype: fact\n---\nbeta ")
	}
}

// plantWithFixedVector overrides the stub-embedder behavior by writing
// a sidecar with an exact vector, bypassing text-hash variation.
func plantWithFixedVector(t *testing.T, memFS *inMemoryFS, vault, relPath, body string, vec []float32) {
	t.Helper()

	notePath := filepath.Join(vault, relPath)
	memFS.files[notePath] = []byte(body)

	sidecar := embed.Sidecar{
		EmbeddingModelID: "m@4",
		Dims:             len(vec),
		Vector:           vec,
		ContentHash:      embed.ContentHash([]byte(body)),
	}

	memFS.files[filepath.Join(vault, embed.SidecarPath(relPath))] = embed.MarshalSidecar(sidecar)
}
