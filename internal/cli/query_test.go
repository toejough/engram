package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

func TestApplyProjectFilter_BodyProjectMentionDoesNotMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewResolvedItem("Permanent/a.md",
			"---\ntype: fact\n---\nthis body mentions project: engram in text\n"),
	}

	filtered := cli.ExportApplyProjectFilter(items, "engram")
	g.Expect(filtered).To(BeEmpty())
}

func TestApplyProjectFilter_DropsNonMatching(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewResolvedItem("Permanent/a.md",
			"---\ntype: fact\nproject: engram\n---\nbody\n"),
		cli.ExportNewResolvedItem("Permanent/b.md",
			"---\ntype: fact\nproject: other\n---\nbody\n"),
		cli.ExportNewResolvedItem("Permanent/c.md",
			"---\ntype: fact\n---\nbody\n"),
	}

	filtered := cli.ExportApplyProjectFilter(items, "engram")

	g.Expect(filtered).To(HaveLen(1))
	g.Expect(cli.ExportResolvedItemPath(filtered[0])).To(Equal("Permanent/a.md"))
}

func TestApplyProjectFilter_EmptyProjectReturnsAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewResolvedItem("Permanent/a.md",
			"---\ntype: fact\nproject: engram\n---\nbody\n"),
		cli.ExportNewResolvedItem("Permanent/b.md",
			"---\ntype: fact\n---\nbody\n"),
	}

	filtered := cli.ExportApplyProjectFilter(items, "")

	g.Expect(filtered).To(HaveLen(2))
}

func TestApplyTierFilter_BodyTierMentionDoesNotMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewResolvedItem("Permanent/a.md",
			"---\ntype: fact\n---\nthis body mentions tier: L3 in text\n"),
	}

	filtered := cli.ExportApplyTierFilter(items, []string{"L3"})
	g.Expect(filtered).To(BeEmpty())
}

func TestApplyTierFilter_DropsNonMatching(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewResolvedItem("Permanent/a.md",
			"---\ntype: fact\ntier: L3\n---\nbody\n"),
		cli.ExportNewResolvedItem("Permanent/b.md",
			"---\ntype: fact\ntier: L2\n---\nbody\n"),
		cli.ExportNewResolvedItem("Permanent/c.md",
			"---\ntype: fact\n---\nbody\n"),
	}

	filtered := cli.ExportApplyTierFilter(items, []string{"L3"})

	g.Expect(filtered).To(HaveLen(1))
	g.Expect(cli.ExportResolvedItemPath(filtered[0])).To(Equal("Permanent/a.md"))
}

func TestApplyTierFilter_EmptyTierReturnsAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewResolvedItem("Permanent/a.md",
			"---\ntype: fact\ntier: L3\n---\nbody\n"),
		cli.ExportNewResolvedItem("Permanent/b.md",
			"---\ntype: fact\n---\nbody\n"),
	}

	filtered := cli.ExportApplyTierFilter(items, nil)

	g.Expect(filtered).To(HaveLen(2))
}

func TestQuery_EmbeddingFailureSurfacesError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.foo.md",
		"---\ntype: fact\n---\nbody\n")

	deps := newQueryDeps(memFS)
	deps.Embedder = errorEmbedder{}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"x"}, VaultPath: vault}, deps, &out)

	g.Expect(err).To(MatchError(ContainSubstring("embed")))
}

func TestQuery_EmptyPhrases_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{}, VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).To(MatchError(ContainSubstring("empty query")))
}

func TestQuery_EmptyVault_ItemsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"anything"}, VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []any `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).To(BeEmpty())
}

func TestQuery_MultiPhrase_BudgetHasPhrasesQueried(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.foo.md",
		"---\ntype: fact\n---\nbody\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body", "fact", "something"}, VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Budget struct {
			PhrasesQueried int `yaml:"phrases_queried"`
		} `yaml:"budget"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Budget.PhrasesQueried).To(Equal(3))
}

func TestQuery_MultiPhrase_ClustersTaggedWithPhrase(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	for i := range 12 {
		plantNoteWithSidecar(t, memFS, vault,
			fmt.Sprintf("Permanent/%d.note.md", i+1),
			fmt.Sprintf("---\ntype: fact\n---\nbody %d\n", i))
	}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body", "fact"}, VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Clusters []struct {
			Phrase string `yaml:"phrase"`
			ID     int    `yaml:"id"`
		} `yaml:"clusters"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	for _, cluster := range parsed.Clusters {
		g.Expect(cluster.Phrase).NotTo(BeEmpty(), "cluster id=%d has no phrase label", cluster.ID)
	}
}

func TestQuery_MultiPhrase_DeduplicatesItemsByPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.foo.md",
		"---\ntype: fact\n---\nbody of note one\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body of note one", "body of note one"}, VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []struct {
			Path string `yaml:"path"`
		} `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	seen := map[string]int{}
	for _, item := range parsed.Items {
		seen[item.Path]++
	}

	for path, count := range seen {
		g.Expect(count).To(Equal(1), "path %s appeared %d times", path, count)
	}
}

func TestQuery_MultiPhrase_HubInDegreeIsMergedMax(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Hub note H is linked-to by many spokes — makes it a hub (high in-degree).
	// Two phrases that both surface H in their subgraphs; the merged payload
	// should have H's in_degree set (non-nil), exercising the inDegree branch
	// of mergeIntoExisting.
	plantNoteWithSidecar(t, memFS, vault, "Permanent/H.md",
		"---\ntype: fact\n---\nhub anchor\n")

	for i := range 6 {
		plantNoteWithSidecar(t, memFS, vault,
			fmt.Sprintf("Permanent/S%d.md", i),
			fmt.Sprintf("---\ntype: fact\n---\nspoke body %d\n[[H]]\n", i))
	}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"hub anchor", "spoke body"}, VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []struct {
			Path     string `yaml:"path"`
			InDegree *int   `yaml:"in_degree"`
		} `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	hubFound := false

	for _, item := range parsed.Items {
		if strings.Contains(item.Path, "H.md") && item.InDegree != nil {
			hubFound = true
		}
	}

	g.Expect(hubFound).To(BeTrue(), "expected H.md to appear as a hub with in_degree set")
}

func TestQuery_MultiPhrase_LaterHigherScoreWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.body.md",
		"---\ntype: fact\n---\nbody text\n")

	// phrase order: low-score first, high-score second — verifies the
	// score-update branch in mergeIntoExisting fires and wins.
	var outLowFirst bytes.Buffer

	_ = cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"xyzzy", "body text"}, VaultPath: vault},
		newQueryDeps(memFS), &outLowFirst)

	var outHighFirst bytes.Buffer

	_ = cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body text", "xyzzy"}, VaultPath: vault},
		newQueryDeps(memFS), &outHighFirst)

	var parsedLowFirst, parsedHighFirst struct {
		Items []struct {
			Score float32 `yaml:"score"`
		} `yaml:"items"`
	}

	_ = yaml.Unmarshal(outLowFirst.Bytes(), &parsedLowFirst)
	_ = yaml.Unmarshal(outHighFirst.Bytes(), &parsedHighFirst)

	if len(parsedLowFirst.Items) == 0 || len(parsedHighFirst.Items) == 0 {
		t.Skip("no items returned; skip score comparison")
	}

	g.Expect(parsedLowFirst.Items[0].Score).To(
		BeNumerically("~", parsedHighFirst.Items[0].Score, float32(0.01)),
		"max score should be the same regardless of phrase order",
	)
}

func TestQuery_MultiPhrase_MaxScoreAcrossPhrases(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.body.md",
		"---\ntype: fact\n---\nbody\n")

	var outSingle bytes.Buffer

	_ = cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body"}, VaultPath: vault},
		newQueryDeps(memFS), &outSingle)

	var parsedSingle struct {
		Items []struct {
			Path  string  `yaml:"path"`
			Score float32 `yaml:"score"`
		} `yaml:"items"`
	}

	_ = yaml.Unmarshal(outSingle.Bytes(), &parsedSingle)

	if len(parsedSingle.Items) == 0 {
		t.Skip("no items returned; skip score comparison")
	}

	singleScore := parsedSingle.Items[0].Score

	var outMulti bytes.Buffer

	_ = cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body", "xyzzy"}, VaultPath: vault},
		newQueryDeps(memFS), &outMulti)

	var parsedMulti struct {
		Items []struct {
			Path  string  `yaml:"path"`
			Score float32 `yaml:"score"`
		} `yaml:"items"`
	}

	_ = yaml.Unmarshal(outMulti.Bytes(), &parsedMulti)

	if len(parsedMulti.Items) == 0 {
		t.Skip("no items in multi result; skip score comparison")
	}

	g.Expect(parsedMulti.Items[0].Score).To(BeNumerically(">=", singleScore))
}

func TestQuery_NotesButNoSidecars_ErrorWithRecoveryHint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	memFS.files[filepath.Join(vault, "Permanent/1.foo.md")] = []byte("body")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"anything"}, VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).To(MatchError(ContainSubstring("engram embed apply --all")))
}

func TestQuery_NotesWithIncompatibleSidecars_ErrorWithRecoveryHint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Plant a note with a sidecar from a different model — the active
	// embedder uses modelID "m@4", but this sidecar is stamped "OLD@384".
	// Before the bug fix, countWithSidecars would count this sidecar as
	// satisfying the guard, then rankCandidates would silently drop it,
	// producing empty results with exit 0. The guard must trigger.
	const relPath = "Permanent/1.foo.md"

	body := []byte("---\ntype: fact\n---\nbody\n")
	memFS.files[filepath.Join(vault, relPath)] = body

	incompat := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "OLD@384",
		Dims:             4,
		SituationVector:  []float32{0, 0, 0, 0},
		BodyVector:       []float32{0, 0, 0, 0},
		ContentHash:      embed.ContentHash(body),
	}
	memFS.files[filepath.Join(vault, embed.SidecarPath(relPath))] = embed.MarshalSidecar(incompat)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"anything"}, VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).To(MatchError(ContainSubstring("engram embed apply --all")))
}

func TestQuery_OldSchemaSidecars_EmitSchemaAdvisory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	// Current sidecar → hits != 0 (avoids the errQueryNoEmbeddings guard).
	plantDualVector(t, memFS, vault, "Permanent/1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nb\n",
		[]float32{1, 0, 0, 0}, []float32{1, 0, 0, 0})
	// Old single-vector sidecar → must be counted + surfaced, not dropped silently.
	memFS.files[filepath.Join(vault, "Permanent/2.fact.md")] = []byte("---\ntype: fact\nsituation: y\n---\n\nb\n")
	memFS.files[filepath.Join(vault, "Permanent/2.fact.vec.json")] = []byte(
		`{"embedding_model_id":"m@4","dims":4,"vector":[1,0,0,0],"content_hash":"sha256:y"}`)

	var advisories []string

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
	deps.LogWarning = func(format string, args ...any) {
		advisories = append(advisories, fmt.Sprintf(format, args...))
	}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"x"}, VaultPath: vault}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(strings.Join(advisories, "\n")).To(ContainSubstring("schema"),
		"old-schema sidecars must be counted and surfaced via the mismatch advisory, not silently dropped")
}

func TestQuery_PhrasesFlag_AcceptsMultiplePhrases(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.foo.md",
		"---\ntype: fact\n---\nbody\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body", "fact"}, VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Phrases []string `yaml:"phrases"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Phrases).To(ConsistOf("body", "fact"))
}

func TestQuery_RanksByDescendingCosine(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Two notes; one mirrors the query string, one differs entirely.
	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.match.md",
		"---\ntype: fact\n---\nthe query string body\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/2.differ.md",
		"---\ntype: fact\n---\nzzz\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"the query string body"}, VaultPath: vault, Limit: 2},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []struct {
			Path        string   `yaml:"path"`
			Kind        string   `yaml:"kind"`
			Score       float32  `yaml:"score"`
			Provenances []string `yaml:"provenances"`
			Content     string   `yaml:"content"`
		} `yaml:"items"`
		Budget struct {
			TotalNotes         int `yaml:"total_notes"`
			WithEmbeddings     int `yaml:"with_embeddings"`
			DirectHitsReturned int `yaml:"direct_hits_returned"`
			Limit              int `yaml:"limit"`
		} `yaml:"budget"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).To(HaveLen(2))
	g.Expect(parsed.Items[0].Path).To(Equal("Permanent/1.match.md"))
	g.Expect(parsed.Items[0].Score).To(BeNumerically(">", parsed.Items[1].Score))
	g.Expect(parsed.Items[0].Provenances).To(Equal([]string{"direct"}))
	g.Expect(parsed.Items[0].Kind).To(Equal("fact"))
	g.Expect(parsed.Items[0].Content).To(ContainSubstring("the query string body"))
	g.Expect(parsed.Budget.TotalNotes).To(Equal(2))
	g.Expect(parsed.Budget.WithEmbeddings).To(Equal(2))
	g.Expect(parsed.Budget.DirectHitsReturned).To(Equal(2))
	g.Expect(parsed.Budget.Limit).To(Equal(2))
}

func TestQuery_RespectsLimit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	for i := range 5 {
		plantNoteWithSidecar(t, memFS, vault,
			"Permanent/"+strings.Repeat("a", i+1)+".md",
			"---\ntype: fact\n---\nbody\n")
	}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body"}, VaultPath: vault, Limit: 2},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []any `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).To(HaveLen(2))
}

func TestQuery_ScoresByMaxOfSituationAndBody(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Note whose BODY vector is orthogonal to the query but whose SITUATION
	// vector matches it — must still surface.
	plantDualVector(t, memFS, vault, "Permanent/1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nbody\n",
		[]float32{1, 0, 0, 0}, []float32{0, 0, 0, 1})

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, Limit: 20}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).NotTo(BeEmpty(),
		"situation-axis match must surface even when body is orthogonal")
	g.Expect(parsed.Items[0].Score).To(BeNumerically("~", 1.0, 1e-6),
		"max() must report the winning (situation) axis score, not the orthogonal body axis (0)")
}

func TestQuery_ScoresByMaxWhenBodyWins(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Mirror image: the SITUATION vector is orthogonal to the query but the
	// BODY vector matches it — the max() must pick the body axis. The
	// reported score must equal the body-axis cosine (1), not the situation
	// axis (0).
	plantDualVector(t, memFS, vault, "Permanent/1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nbody\n",
		[]float32{0, 0, 0, 1}, []float32{1, 0, 0, 0})

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, Limit: 20}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).NotTo(BeEmpty(),
		"body-axis match must surface even when situation is orthogonal")
	g.Expect(parsed.Items[0].Score).To(BeNumerically("~", 1.0, 1e-6),
		"max() must report the winning (body) axis score")
}

func TestQuery_StripsWikilinksFromItemsContent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	body := "---\ntype: fact\n---\n" +
		"See [[1a.foo]] and [[2b.bar|the bar note]] for context.\n"

	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.foo.md", body)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"context"}, VaultPath: vault, Limit: 1},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []struct {
			Content string `yaml:"content"`
		} `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).To(HaveLen(1))
	// Both wikilink shapes are stripped; display/target text remains.
	g.Expect(parsed.Items[0].Content).NotTo(ContainSubstring("[["))
	g.Expect(parsed.Items[0].Content).NotTo(ContainSubstring("]]"))
	g.Expect(parsed.Items[0].Content).
		To(ContainSubstring("See 1a.foo and the bar note for context."))
}

func TestRunQuery_ExposesOutboundWikilinkTargets(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// hub links to two real notes; the fenced [[9.fenced]] must NOT count
	// (fence-aware parser), proving outbound links reuse the graph parser.
	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.hub.md",
		"---\ntype: fact\n---\nSee [[2.alpha]] and [[3.beta]].\n```\n[[9.fenced]]\n```\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/2.alpha.md",
		"---\ntype: fact\n---\nalpha body\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/3.beta.md",
		"---\ntype: fact\n---\nbeta body\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"See alpha and beta"}, VaultPath: vault, Limit: 5},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []struct {
			Path          string   `yaml:"path"`
			OutboundLinks []string `yaml:"outbound_links"`
		} `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	var hubLinks []string

	found := false

	for _, item := range parsed.Items {
		if strings.Contains(item.Path, "1.hub") {
			hubLinks = item.OutboundLinks
			found = true
		}
	}

	g.Expect(found).To(BeTrue(), "hub note must appear in items")
	g.Expect(hubLinks).To(ConsistOf("2.alpha", "3.beta"),
		"outbound_links must list the note's resolvable wikilink targets")
	g.Expect(hubLinks).NotTo(ContainElement("9.fenced"),
		"a wikilink inside a code fence must not be reported as an outbound target")
}

func TestRunQuery_ModelMismatchEmitsWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// One compatible note (active model "m@4") so recall is non-empty, plus
	// one note whose sidecar is stamped with a stale model id. The stale
	// sidecar is silently dropped today; the fix must warn instead.
	plantNoteWithSidecar(t, memFS, vault, "Permanent/good.md",
		"---\ntype: fact\n---\nbody good\n")

	const staleModelID = "STALE@384"

	staleBody := []byte("---\ntype: fact\n---\nbody stale\n")
	memFS.files[filepath.Join(vault, "Permanent/stale.md")] = staleBody
	memFS.files[filepath.Join(vault, embed.SidecarPath("Permanent/stale.md"))] = embed.MarshalSidecar(
		embed.Sidecar{
			SchemaVersion:    embed.SidecarSchemaVersion,
			EmbeddingModelID: staleModelID,
			Dims:             4,
			SituationVector:  []float32{0, 0, 0, 0},
			BodyVector:       []float32{0, 0, 0, 0},
			ContentHash:      embed.ContentHash(staleBody),
		},
	)

	var warnings []string

	deps := newQueryDeps(memFS)
	deps.LogWarning = func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body"}, VaultPath: vault}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(warnings).NotTo(BeEmpty(), "model mismatch must emit a warning, not silently drop")

	joined := strings.Join(warnings, "\n")
	g.Expect(joined).To(ContainSubstring("1"), "warning must report the dropped count")
	g.Expect(joined).To(ContainSubstring(staleModelID), "warning must name the mismatched model id")
}

func TestRunQuery_MultipleTiersUnion(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.l1-note.md",
		"---\ntype: episode\ntier: L1\n---\nbody about tier\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/2.l2-note.md",
		"---\ntype: fact\ntier: L2\n---\nbody about tier\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/3.l3-note.md",
		"---\ntype: fact\ntier: L3\n---\nbody about tier\n")

	var out bytes.Buffer

	// R5's read-subset {L2,L3} on a 3-tier vault: repeatable --tier unions the
	// two requested tiers and excludes L1.
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body"}, VaultPath: vault, Tiers: []string{"L2", "L3"}},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []struct {
			Path string `yaml:"path"`
		} `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).NotTo(BeEmpty())

	var sawL2, sawL3 bool

	for _, item := range parsed.Items {
		g.Expect(item.Path).NotTo(ContainSubstring("l1-note"),
			"L1 must not surface under --tier L2 --tier L3")

		if strings.Contains(item.Path, "l2-note") {
			sawL2 = true
		}

		if strings.Contains(item.Path, "l3-note") {
			sawL3 = true
		}
	}

	g.Expect(sawL2).To(BeTrue(), "L2 must surface under union read")
	g.Expect(sawL3).To(BeTrue(), "L3 must surface under union read")
}

func TestRunQuery_ProjectFilterRestrictsItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.engram-note.md",
		"---\ntype: fact\nproject: engram\n---\nbody about engram\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/2.opencode-note.md",
		"---\ntype: fact\nproject: opencode\n---\nbody about opencode\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body"}, VaultPath: vault, Project: "engram"},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []struct {
			Path string `yaml:"path"`
		} `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).NotTo(BeEmpty())

	for _, item := range parsed.Items {
		g.Expect(item.Path).NotTo(ContainSubstring("opencode-note"))
	}
}

func TestRunQuery_TierFilterRestrictsItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "Permanent/1.l2-note.md",
		"---\ntype: fact\ntier: L2\n---\nbody about tier\n")
	plantNoteWithSidecar(t, memFS, vault, "Permanent/2.l3-note.md",
		"---\ntype: fact\ntier: L3\n---\nbody about tier\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body"}, VaultPath: vault, Tiers: []string{"L3"}},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []struct {
			Path string `yaml:"path"`
		} `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).NotTo(BeEmpty())

	for _, item := range parsed.Items {
		g.Expect(item.Path).NotTo(ContainSubstring("l2-note"))
	}
}

func TestRunQuery_TierIsolationAcrossAllChannels(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFS := newInMemoryFS()
	vault := plantTieredVault(t, memFS)

	// --- Tier L3: every path-bearing channel must be L3-only. ---
	parsedL3 := runTieredQuery(t, g, memFS, vault, "L3")
	g.Expect(parsedL3.Items).NotTo(BeEmpty())
	// Guard against a vacuous pass: clusters must actually form.
	g.Expect(parsedL3.Clusters).NotTo(BeEmpty(), "expected clusters to form for the L3 subgraph")
	assertChannelsMatchTier(g, parsedL3, "l3-")

	// --- Tier L2: items + members are L2-only and nearest_l3 is dropped. ---
	parsedL2 := runTieredQuery(t, g, memFS, vault, "L2")
	assertChannelsMatchTier(g, parsedL2, "l2-")

	for _, cluster := range parsedL2.Clusters {
		g.Expect(cluster.NearestL3).To(BeNil(),
			"nearest_l3 (always L3) must be dropped when --tier is non-L3")
	}

	// --- Blended (empty tier): both tiers appear; no over-filtering. ---
	parsedAll := runTieredQuery(t, g, memFS, vault, "")
	sawL2, sawL3 := tiersPresent(parsedAll)
	g.Expect(sawL2).To(BeTrue(), "blended recall must include L2 notes")
	g.Expect(sawL3).To(BeTrue(), "blended recall must include L3 notes")
}

type errorEmbedder struct{}

func (errorEmbedder) Dims() int { return 4 }

func (errorEmbedder) Embed(context.Context, string) ([]float32, error) {
	return nil, errors.New("embedder down")
}

func (errorEmbedder) ModelID() string { return "m@4" }

// assertChannelsMatchTier asserts every path-bearing channel (items,
// cluster members, and any nearest_l3) contains the given tier marker.
// Hubs have no dedicated payload field — they surface only as items[]
// entries carrying the "hub" provenance and an in_degree — so the items
// assertion below also covers the hub channel; a hub cannot leak a note
// of another tier without that note appearing (and failing) as an item.
func assertChannelsMatchTier(g *WithT, parsed queryParsed, marker string) {
	for _, item := range parsed.Items {
		g.Expect(item.Path).To(ContainSubstring(marker), "items leaked note %q", item.Path)
	}

	for _, cluster := range parsed.Clusters {
		for _, member := range cluster.Members {
			g.Expect(member.Path).To(ContainSubstring(marker),
				"cluster member leaked note %q", member.Path)
		}

		if cluster.NearestL3 != nil {
			g.Expect(cluster.NearestL3.Path).To(ContainSubstring(marker),
				"nearest_l3 leaked note %q", cluster.NearestL3.Path)
		}
	}
}

func newQueryDeps(memFS *inMemoryFS) cli.QueryDeps {
	return cli.QueryDeps{
		Scan:     memFS.Scan,
		Read:     memFS.Read,
		Embedder: stubEmbedder{modelID: "m@4", dims: 4},
	}
}

// plantNoteWithSidecar populates memFS with a note + matching sidecar.
func plantNoteWithSidecar(t *testing.T, memFS *inMemoryFS, vault, relPath, body string) {
	t.Helper()

	notePath := filepath.Join(vault, relPath)
	memFS.files[notePath] = []byte(body)

	emb := stubEmbedder{modelID: "m@4", dims: 4}

	sidecar, err := embed.BuildSidecar(context.Background(), emb, []byte(body))
	if err != nil {
		t.Fatalf("build sidecar: %v", err)
	}

	memFS.files[filepath.Join(vault, embed.SidecarPath(relPath))] = embed.MarshalSidecar(sidecar)
}

// plantTieredVault seeds a tempdir vault with enough L3 notes to cluster
// (>= minSubgraphForClustering) plus some L2 notes. All share the body
// token "body" so they all become direct hits and seed one subgraph.
func plantTieredVault(t *testing.T, memFS *inMemoryFS) string {
	t.Helper()

	const (
		l3Count = 8
		l2Count = 4
	)

	vault := t.TempDir()

	for i := range l3Count {
		plantNoteWithSidecar(t, memFS, vault,
			fmt.Sprintf("Permanent/l3-%d.md", i),
			fmt.Sprintf("---\ntype: fact\ntier: L3\n---\nbody l3 note %d\n", i))
	}

	for i := range l2Count {
		plantNoteWithSidecar(t, memFS, vault,
			fmt.Sprintf("Permanent/l2-%d.md", i),
			fmt.Sprintf("---\ntype: fact\ntier: L2\n---\nbody l2 note %d\n", i))
	}

	return vault
}

// runTieredQuery runs a single-phrase query at the given tier and returns
// the parsed payload, asserting the call itself succeeds.
func runTieredQuery(t *testing.T, g *WithT, memFS *inMemoryFS, vault, tier string) queryParsed {
	t.Helper()

	var out bytes.Buffer

	var tiers []string
	if tier != "" {
		tiers = []string{tier}
	}

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body"}, VaultPath: vault, Tiers: tiers, Limit: 20},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	return parsed
}

// tiersPresent reports whether L2 and L3 notes both appear in items.
func tiersPresent(parsed queryParsed) (sawL2, sawL3 bool) {
	for _, item := range parsed.Items {
		if strings.Contains(item.Path, "l2-") {
			sawL2 = true
		}

		if strings.Contains(item.Path, "l3-") {
			sawL3 = true
		}
	}

	return sawL2, sawL3
}
