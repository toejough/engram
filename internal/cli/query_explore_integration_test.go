package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestComputeExploreHalf_DeliversMemberNotInExploit verifies the query-path
// glue (task 2.1): a vocab term read from vocab.centroids.json, with a
// vault-tagged member note not present in the exploit half, is delivered as
// an explore pick carrying its source term, and the returned delivered-count
// map reports it under that term.
func TestComputeExploreHalf_DeliversMemberNotInExploit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const vault = "/vault"

	centroid := []float32{1, 0, 0, 0}
	memberVec := []float32{1, 0, 0, 0}

	centroidsDoc := cli.ExportVocabCentroidsDoc{
		SchemaVersion: 1,
		Terms: map[string]cli.ExportVocabCentroidEntry{
			"topic-a": {Vector: centroid, MemberCount: 1},
		},
	}

	centroidsData, marshalErr := json.Marshal(centroidsDoc)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	readFile := func(path string) ([]byte, error) {
		if path == filepath.Join(vault, "vocab.centroids.json") {
			return centroidsData, nil
		}

		return nil, &testNotFoundError{path: path}
	}

	meta := cli.ExportNewVaultNotesMetaWithTerms(map[string][]cli.ExportVaultTermMember{
		"topic-a": {
			{NotePath: "2.fact.md", Content: "---\ntype: fact\nsituation: s\n---\nbody\n", Vector: memberVec},
		},
	})

	exploitPaths := map[string]struct{}{"1.fact.md": {}}

	picks, delivered := cli.ExportComputeExploreHalf(vault, readFile, centroid, exploitPaths, 1, meta)

	g.Expect(picks).To(HaveLen(1))
	g.Expect(delivered).To(Equal(map[string]int{"topic-a": 1}))

	if len(picks) == 0 {
		return
	}

	g.Expect(picks[0].Path).To(Equal("2.fact.md"))
	g.Expect(picks[0].SourceTerm).To(Equal("topic-a"))
}

// TestComputeExploreHalf_ExcludesDefinitionAndVectorlessMembers verifies that
// exploreMembersByTerm drops definition notes (bare "vocab" tag) and members
// with no sidecar vector, leaving only the one eligible member selectable.
func TestComputeExploreHalf_ExcludesDefinitionAndVectorlessMembers(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const vault = "/vault"

	centroid := []float32{1, 0, 0, 0}

	centroidsDoc := cli.ExportVocabCentroidsDoc{
		SchemaVersion: 1,
		Terms: map[string]cli.ExportVocabCentroidEntry{
			"topic-a": {Vector: centroid, MemberCount: 1},
		},
	}

	centroidsData, marshalErr := json.Marshal(centroidsDoc)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	readFile := func(path string) ([]byte, error) {
		if path == filepath.Join(vault, "vocab.centroids.json") {
			return centroidsData, nil
		}

		return nil, &testNotFoundError{path: path}
	}

	definitionContent := "---\ntype: vocab\ntags:\n    - vocab\n---\n\ndefinition body\n"

	meta := cli.ExportNewVaultNotesMetaWithTerms(map[string][]cli.ExportVaultTermMember{
		"topic-a": {
			{NotePath: "def.md", Content: definitionContent, Vector: centroid},
			{NotePath: "novec.md", Content: "---\ntype: fact\nsituation: s\n---\nbody\n", Vector: nil},
			{NotePath: "eligible.md", Content: "---\ntype: fact\nsituation: s\n---\nbody\n", Vector: centroid},
		},
	})

	picks, delivered := cli.ExportComputeExploreHalf(vault, readFile, centroid, map[string]struct{}{}, 1, meta)

	g.Expect(picks).To(HaveLen(1))
	g.Expect(delivered).To(Equal(map[string]int{"topic-a": 1}))

	if len(picks) == 0 {
		return
	}

	g.Expect(picks[0].Path).To(Equal("eligible.md"),
		"the definition note and the vectorless note must both be excluded")
}

// TestComputeExploreHalf_ZeroBudget_ReturnsEmptyAllocation verifies the B=0
// skip case (spec: "a query whose exploit half delivers zero notes skips
// explore sampling entirely").
func TestComputeExploreHalf_ZeroBudget_ReturnsEmptyAllocation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	readFile := func(string) ([]byte, error) {
		t.Fatal("computeExploreHalf must not read centroids when budget is zero")

		return nil, nil
	}

	picks, delivered := cli.ExportComputeExploreHalf(
		"/vault", readFile, []float32{1, 0}, map[string]struct{}{}, 0, cli.ExportNewEmptyVaultNotesMeta(),
	)

	g.Expect(picks).To(BeEmpty())
	g.Expect(delivered).To(Equal(map[string]int{}))
}

// TestRenderQueryPayload_ExploreItem_ProvenanceAndSourceTerm verifies the
// spec's "Explore provenance is reported" requirement: an explore-half item
// renders with provenance=explore and its source_term in the YAML payload.
func TestRenderQueryPayload_ExploreItem_ProvenanceAndSourceTerm(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	item := cli.ExportNewExploreResolvedItem("2.fact.md", "explore content", "topic-a", 0.8)

	out, err := cli.ExportRenderQueryPayloadFromResolvedItems([]cli.ExportResolvedItem{item})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(out).To(ContainSubstring("- explore"))
	g.Expect(out).To(ContainSubstring("source_term: topic-a"))
}

// TestRunQuery_ExploreHalf_FullPipeline_DeliversExploreItem drives the full
// RunQuery pipeline (task 2.1 integration) with a controlled, non-hash
// embedder so the explore-half note is guaranteed to miss the exploit floor
// (orthogonal to the query) yet be selectable by its own vocab term centroid
// — exercising contentByPath, explorePicksToResolvedItems, and
// exploreMembersByTerm's non-excluded path end-to-end.
func TestRunQuery_ExploreHalf_FullPipeline_DeliversExploreItem(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	matchVec := []float32{1, 0, 0, 0}
	exploreVec := []float32{0, 1, 0, 0}

	emb := tokenEmbedder{
		modelID: "cov@4",
		dims:    4,
		vectors: map[string][]float32{
			"MATCHTOKEN":   matchVec,
			"EXPLORETOKEN": exploreVec,
		},
	}

	const matchedRelPath = "1.fact.md"

	const exploreRelPath = "2.fact.md"

	plantNoteWithEmbedder(t, memFS, vault, matchedRelPath,
		"---\ntype: fact\nsituation: s\n---\nMATCHTOKEN body\n", emb)
	plantNoteWithEmbedder(t, memFS, vault, exploreRelPath,
		"---\ntype: fact\nsituation: s\ntags:\n    - vocab/topic-a\n---\nEXPLORETOKEN body\n", emb)

	centroidsDoc := cli.ExportVocabCentroidsDoc{
		SchemaVersion: 1,
		Terms: map[string]cli.ExportVocabCentroidEntry{
			"topic-a": {Vector: exploreVec, MemberCount: 1},
		},
	}

	centroidsData, marshalErr := json.Marshal(centroidsDoc)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	memFS.files[filepath.Join(vault, "vocab.centroids.json")] = centroidsData

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"MATCHTOKEN"}, VaultPath: vault},
		cli.QueryDeps{Scan: memFS.Scan, Read: memFS.Read, Embedder: emb}, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(ContainSubstring("- explore"))
	g.Expect(out.String()).To(ContainSubstring("source_term: topic-a"))
	g.Expect(out.String()).To(ContainSubstring("EXPLORETOKEN body"),
		"explore item must carry its note content via contentByPath")
	g.Expect(out.String()).To(ContainSubstring("explore_allocated:\n    topic-a: 1"))
}

// TestRunQuery_ExploreHalf_MissingCentroids_ExploitOnly verifies the
// degrade-loudly path (design.md, spec "Missing centroids degrade loudly"):
// when vocab.centroids.json is absent, the payload is exploit-only and the
// budget reports an explicit empty explore_allocated map, not an omitted one.
func TestRunQuery_ExploreHalf_MissingCentroids_ExploitOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "1.fact.md", "---\ntype: fact\nsituation: s\n---\nmatched body\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"matched body"}, VaultPath: vault},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(ContainSubstring("explore_allocated: {}"),
		"missing centroids must degrade to an explicit empty explore allocation")
	g.Expect(out.String()).NotTo(ContainSubstring("- explore"),
		"missing centroids must deliver no explore-half items")
}

// tokenEmbedder returns a fixed vector for any text CONTAINING a known
// token, and the zero vector otherwise — a controllable alternative to the
// hash-based stubEmbedder for tests that need guaranteed orthogonality.
type tokenEmbedder struct {
	modelID string
	dims    int
	vectors map[string][]float32
}

func (e tokenEmbedder) Dims() int { return e.dims }

func (e tokenEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	for token, vec := range e.vectors {
		if strings.Contains(text, token) {
			return vec, nil
		}
	}

	return make([]float32, e.dims), nil
}

func (e tokenEmbedder) ModelID() string { return e.modelID }

// plantNoteWithEmbedder is plantNoteWithSidecar with an injected embedder,
// for tests that need controlled (non-hash) vectors.
func plantNoteWithEmbedder(
	t *testing.T, memFS *inMemoryFS, vault, relPath, body string, emb tokenEmbedder,
) {
	t.Helper()

	notePath := filepath.Join(vault, relPath)
	memFS.files[notePath] = []byte(body)

	sidecar, err := embed.BuildSidecar(context.Background(), emb, []byte(body))
	if err != nil {
		t.Fatalf("build sidecar: %v", err)
	}

	memFS.files[filepath.Join(vault, embed.SidecarPath(relPath))] = embed.MarshalSidecar(sidecar)
}
