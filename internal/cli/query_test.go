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
		EmbeddingModelID: "OLD@384",
		Dims:             4,
		Vector:           []float32{0, 0, 0, 0},
		ContentHash:      embed.ContentHash(body),
	}
	memFS.files[filepath.Join(vault, embed.SidecarPath(relPath))] = embed.MarshalSidecar(incompat)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"anything"}, VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).To(MatchError(ContainSubstring("engram embed apply --all")))
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

type errorEmbedder struct{}

func (errorEmbedder) Dims() int { return 4 }

func (errorEmbedder) Embed(context.Context, string) ([]float32, error) {
	return nil, errors.New("embedder down")
}

func (errorEmbedder) ModelID() string { return "m@4" }

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
	vec, _ := emb.Embed(context.Background(), string(embed.ExtractBody([]byte(body))))
	sidecar := embed.Sidecar{
		EmbeddingModelID: emb.ModelID(),
		Dims:             emb.Dims(),
		Vector:           vec,
		ContentHash:      embed.ContentHash([]byte(body)),
	}

	memFS.files[filepath.Join(vault, embed.SidecarPath(relPath))] = embed.MarshalSidecar(sidecar)
}
