package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

func TestApplyProjectFilter_BodyProjectMentionDoesNotMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewResolvedItem("a.md",
			"---\ntype: fact\n---\nthis body mentions project: engram in text\n"),
	}

	filtered := cli.ExportApplyProjectFilter(items, "engram")
	g.Expect(filtered).To(BeEmpty())
}

func TestApplyProjectFilter_DropsNonMatching(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewResolvedItem("a.md",
			"---\ntype: fact\nproject: engram\n---\nbody\n"),
		cli.ExportNewResolvedItem("b.md",
			"---\ntype: fact\nproject: other\n---\nbody\n"),
		cli.ExportNewResolvedItem("c.md",
			"---\ntype: fact\n---\nbody\n"),
	}

	filtered := cli.ExportApplyProjectFilter(items, "engram")

	g.Expect(filtered).To(HaveLen(1))
	g.Expect(cli.ExportResolvedItemPath(filtered[0])).To(Equal("a.md"))
}

func TestApplyProjectFilter_EmptyProjectReturnsAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []cli.ExportResolvedItem{
		cli.ExportNewResolvedItem("a.md",
			"---\ntype: fact\nproject: engram\n---\nbody\n"),
		cli.ExportNewResolvedItem("b.md",
			"---\ntype: fact\n---\nbody\n"),
	}

	filtered := cli.ExportApplyProjectFilter(items, "")

	g.Expect(filtered).To(HaveLen(2))
}

func TestQueryPayload_RefitPendingOmittedWhenFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	out, err := cli.ExportRenderQueryPayloadRefitPending(false)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).NotTo(ContainSubstring("refit_pending"))
}

func TestQueryPayload_RefitPendingPresentWhenTrue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	out, err := cli.ExportRenderQueryPayloadRefitPending(true)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("refit_pending: true"))
}

func TestQuery_EmbeddingFailureSurfacesError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.foo.md",
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
	plantNoteWithSidecar(t, memFS, vault, "1.foo.md",
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

func TestQuery_MultiPhrase_DeduplicatesItemsByPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.foo.md",
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

func TestQuery_MultiPhrase_LaterHigherScoreWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.body.md",
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
	plantNoteWithSidecar(t, memFS, vault, "1.body.md",
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
	memFS.files[filepath.Join(vault, "1.foo.md")] = []byte("body")

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
	const relPath = "1.foo.md"

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
	plantDualVector(t, memFS, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nb\n",
		[]float32{1, 0, 0, 0}, []float32{1, 0, 0, 0})
	// Old single-vector sidecar → must be counted + surfaced, not dropped silently.
	memFS.files[filepath.Join(vault, "2.fact.md")] = []byte("---\ntype: fact\nsituation: y\n---\n\nb\n")
	memFS.files[filepath.Join(vault, "2.fact.vec.json")] = []byte(
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
	plantNoteWithSidecar(t, memFS, vault, "1.foo.md",
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
	plantNoteWithSidecar(t, memFS, vault, "1.match.md",
		"---\ntype: fact\n---\nthe query string body\n")
	plantNoteWithSidecar(t, memFS, vault, "2.differ.md",
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
	g.Expect(parsed.Items[0].Path).To(Equal("1.match.md"))
	g.Expect(parsed.Items[0].Score).To(BeNumerically(">", parsed.Items[1].Score))
	g.Expect(parsed.Items[0].Provenances).To(Equal([]string{"direct"}))
	g.Expect(parsed.Items[0].Kind).To(Equal("fact"))
	g.Expect(parsed.Items[0].Content).To(ContainSubstring("the query string body"))
	g.Expect(parsed.Budget.TotalNotes).To(Equal(2))
	g.Expect(parsed.Budget.WithEmbeddings).To(Equal(2))
	g.Expect(parsed.Budget.DirectHitsReturned).To(Equal(2))
	g.Expect(parsed.Budget.Limit).To(Equal(2))
}

func TestQuery_ScoresByMaxOfSituationAndBody(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Note whose BODY vector is orthogonal to the query but whose SITUATION
	// vector matches it — must still surface.
	plantDualVector(t, memFS, vault, "1.fact.md",
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
	plantDualVector(t, memFS, vault, "1.fact.md",
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

	plantNoteWithSidecar(t, memFS, vault, "1.foo.md", body)

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

func TestRunQuery_ModelMismatchEmitsWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// One compatible note (active model "m@4") so recall is non-empty, plus
	// one note whose sidecar is stamped with a stale model id. The stale
	// sidecar is silently dropped today; the fix must warn instead.
	plantNoteWithSidecar(t, memFS, vault, "good.md",
		"---\ntype: fact\n---\nbody good\n")

	const staleModelID = "STALE@384"

	staleBody := []byte("---\ntype: fact\n---\nbody stale\n")
	memFS.files[filepath.Join(vault, "stale.md")] = staleBody
	memFS.files[filepath.Join(vault, embed.SidecarPath("stale.md"))] = embed.MarshalSidecar(
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

// TestRunQuery_NoActivatedFlagInPayload verifies Phase 4 of recall-v2:
// the binary never emits an activated field on any item (notes or chunks),
// regardless of baseScore. Activation is now agent-driven — the skill calls
// engram activate on notes it actually used, not the binary pre-judging use.
// Plain query writes nothing (sidecar files are unmodified).
func TestRunQuery_NoActivatedFlagInPayload(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Plant a high-cosine note (baseScore=1.0, well above the old 0.5 cutoff)
	// and a low-cosine note (baseScore=0.0). Neither should carry activated:true
	// in the Phase 4 world — the binary emits no activated field at all.
	queryVec := []float32{1, 0, 0, 0}
	zeroVec := []float32{0, 0, 0, 0}
	fixedNow := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	highBody := "---\ntype: fact\ncreated: 2026-06-01\n---\nhigh cosine note\n"
	lowBody := "---\ntype: fact\ncreated: 2026-06-01\n---\nlow cosine note\n"

	highPath := "high.md"
	lowPath := "low.md"

	memFS.files[filepath.Join(vault, highPath)] = []byte(highBody)
	memFS.files[filepath.Join(vault, lowPath)] = []byte(lowBody)

	// high: vectors aligned with query → baseScore=1.0 (previously above cutoff)
	memFS.files[filepath.Join(vault, embed.SidecarPath(highPath))] = embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@4",
		Dims:             4,
		SituationVector:  queryVec,
		BodyVector:       queryVec,
		ContentHash:      embed.ContentHash([]byte(highBody)),
	})
	// low: zero vectors → cosine=0.0 with query (previously below cutoff)
	memFS.files[filepath.Join(vault, embed.SidecarPath(lowPath))] = embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@4",
		Dims:             4,
		SituationVector:  zeroVec,
		BodyVector:       zeroVec,
		ContentHash:      embed.ContentHash([]byte(lowBody)),
	})

	deps := cli.QueryDeps{
		Scan:     memFS.Scan,
		Read:     memFS.Read,
		Embedder: fixedVectorEmbedder{modelID: "m@4", vector: queryVec},
		Now:      func() time.Time { return fixedNow },
	}

	var out bytes.Buffer

	sidecarBefore := make([]byte, len(memFS.files[filepath.Join(vault, embed.SidecarPath(highPath))]))
	copy(sidecarBefore, memFS.files[filepath.Join(vault, embed.SidecarPath(highPath))])

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"high"}, VaultPath: vault, Limit: 10},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Query must NOT write to the sidecar (read-only invariant).
	g.Expect(memFS.files[filepath.Join(vault, embed.SidecarPath(highPath))]).
		To(Equal(sidecarBefore), "query must not modify sidecar files")

	// Parse the raw YAML to detect any activated key, regardless of value.
	// Using a map so that an omitted key is distinguishable from false.
	var rawPayload struct {
		Items []map[string]any `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &rawPayload)).NotTo(HaveOccurred())
	g.Expect(rawPayload.Items).NotTo(BeEmpty(), "at least one item must surface")

	sawHigh := false

	for _, item := range rawPayload.Items {
		path, _ := item["path"].(string)
		if path == highPath {
			sawHigh = true
		}

		_, hasActivated := item["activated"]
		g.Expect(hasActivated).To(BeFalse(),
			"no item must carry an activated field; activation is agent-driven (path=%s)", path)
	}

	g.Expect(sawHigh).To(BeTrue(), "high-cosine note must appear in payload")
}

func TestRunQuery_NoTimingsByDefault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-07-13.alpha.md", "---\ntype: fact\n---\nalpha body")

	deps := newQueryDeps(memFS)
	deps.Now = func() time.Time { return time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC) }

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(), cli.QueryArgs{
		Phrases:   []string{"alpha"},
		VaultPath: vault,
		Timings:   false,
	}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Structural byte-stability guarantee is omitempty-on-nil; here we assert
	// the block is simply absent. The rest of the payload is guarded by the
	// existing query regression suite (Step 9).
	g.Expect(out.String()).NotTo(ContainSubstring("timings:"))
}

// TestRunQuery_NoteRecencyDecayRanksFreshLastUsedFirst verifies that rankCandidates
// applies a recency multiplier when deps.Now is set: two notes with identical base
// cosine scores are re-ranked so the one with a recent LastUsed appears first (higher
// decayed score). With deps.Now==nil both should appear in pure-cosine order (equal
// scores → stable sort → path-lexicographic).
func TestRunQuery_NoteRecencyDecayRanksFreshLastUsedFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Both notes share the same situation and body vectors → equal base cosine.
	sharedVec := []float32{1, 0, 0, 0}

	// "fresh" was last-used 2 days ago; "stale" was last-used 120 days ago.
	fixedNow := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	freshBody := "---\ntype: fact\ncreated: 2026-01-01\n---\nfresh note body\n"
	staleBody := "---\ntype: fact\ncreated: 2026-01-01\n---\nstale note body\n"

	freshPath := "1.fresh.md"
	stalePath := "2.stale.md"

	// Plant note files.
	memFS.files[filepath.Join(vault, freshPath)] = []byte(freshBody)
	memFS.files[filepath.Join(vault, stalePath)] = []byte(staleBody)

	// Plant sidecars with identical vectors but different LastUsed.
	freshSidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@4",
		Dims:             4,
		SituationVector:  sharedVec,
		BodyVector:       sharedVec,
		ContentHash:      embed.ContentHash([]byte(freshBody)),
		LastUsed:         "2026-06-15", // 2 days before fixedNow
	}
	staleSidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@4",
		Dims:             4,
		SituationVector:  sharedVec,
		BodyVector:       sharedVec,
		ContentHash:      embed.ContentHash([]byte(staleBody)),
		LastUsed:         "2026-01-18", // ~150 days before fixedNow
	}

	memFS.files[filepath.Join(vault, embed.SidecarPath(freshPath))] = embed.MarshalSidecar(freshSidecar)
	memFS.files[filepath.Join(vault, embed.SidecarPath(stalePath))] = embed.MarshalSidecar(staleSidecar)

	// Query with fixed now → recency decay applied.
	deps := cli.QueryDeps{
		Scan:     memFS.Scan,
		Read:     memFS.Read,
		Embedder: fixedVectorEmbedder{modelID: "m@4", vector: sharedVec},
		Now:      func() time.Time { return fixedNow },
	}

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"fact"}, VaultPath: vault, Limit: 2},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed struct {
		Items []struct {
			Path  string  `yaml:"path"`
			Score float32 `yaml:"score"`
		} `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).To(HaveLen(2))

	if len(parsed.Items) < 2 {
		return
	}

	// Fresh note (lower age → higher recency multiplier → higher score) must rank first.
	g.Expect(parsed.Items[0].Path).To(Equal(freshPath),
		"fresh LastUsed note must rank before stale (decay penalises the stale one)")
	g.Expect(parsed.Items[0].Score).To(BeNumerically(">", parsed.Items[1].Score),
		"decayed score of fresh note must exceed stale note's decayed score")
}

func TestRunQuery_ProjectFilterRestrictsItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "1.engram-note.md",
		"---\ntype: fact\nproject: engram\n---\nbody about engram\n")
	plantNoteWithSidecar(t, memFS, vault, "2.opencode-note.md",
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

// TestRunQuery_RecencyLiftsRecentChunkOverStaleHighCosine verifies that a
// recent low-cosine chunk surfaces in RunQuery output even when several stale
// higher-cosine chunks would normally bury it under pure cosine ranking.
//
// Setup:
//   - No vault notes (Scan returns empty) so the query is purely chunk-driven.
//   - One planted recent chunk ("recent.jsonl#turn-40", cosine≈0.316) with
//     mtime=now in the manifest.
//   - Several stale chunks ("old.jsonl#turn-N", cosine≈0.707) with mtime 120
//     days ago — each beats the planted chunk on raw cosine.
//   - Now is a fixed time so the test is deterministic.
//
// With pure cosine (deps.Now==nil), the planted chunk would be buried.
// With recency re-rank + band, its score is lifted and it must appear.
func TestRunQuery_RecencyLiftsRecentChunkOverStaleHighCosine(t *testing.T) {
	t.Parallel()

	// Setup constants: sources, anchor, paths.
	const (
		chunksDir       = "/chunks"
		recentSource    = "recent.jsonl"
		oldSource       = "old.jsonl"
		plantedAnchor   = "turn-40"
		recentIndexPath = "/chunks/recent-idx.jsonl"
		oldIndexPath    = "/chunks/old-idx.jsonl"
		manifestPath    = "/chunks/manifest.json"
		staleCount      = 5
	)

	g := NewWithT(t)

	fixedNow := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	recentMtime := fixedNow.Add(-1 * time.Hour).UnixNano()     // ~0.04 days old
	oldMtime := fixedNow.Add(-120 * 24 * time.Hour).UnixNano() // 120 days old

	// Build chunk records: one recent low-cosine, several stale high-cosine.
	//
	// axisEmbedder maps "my-recent-activity" → {1,0,0}. Cosine of stored vector
	// v against {1,0,0} is v[0]/|v|.
	//   recent: {1, 3, 0} → cosine = 1/√10 ≈ 0.316  (low — buried by pure cosine)
	//   stale:  {1, 1, 0} → cosine = 1/√2  ≈ 0.707  (high — beats recent on cosine)
	recentRecord := chunk.Record{
		Source:      recentSource,
		Anchor:      plantedAnchor,
		Text:        "ASSISTANT: I'll file issue #644 for the recall flakiness",
		ContentHash: "sha256:planted-recent",
		Vector:      []float32{1, 3, 0},
	}

	staleRecords := make([]chunk.Record, staleCount)

	for i := range staleCount {
		staleRecords[i] = chunk.Record{
			Source:      oldSource,
			Anchor:      "turn-" + string(rune('0'+i)),
			Text:        "stale chunk " + string(rune('a'+i)),
			ContentHash: "sha256:stale" + string(rune('0'+i)),
			Vector:      []float32{1, 1, 0},
		}
	}

	// Encode chunk index files.
	recentData, err := chunk.EncodeRecords([]chunk.Record{recentRecord})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	staleData, err := chunk.EncodeRecords(staleRecords)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Build manifest.json using map[string]any so keys use the production
	// snake_case format (mtime_unix_nano, file_hash) without local struct tags.
	manifestData, err := json.Marshal(map[string]any{
		recentSource: map[string]any{
			"mtime_unix_nano": recentMtime,
			"size":            int64(len(recentData)),
			"file_hash":       "hash-recent",
		},
		oldSource: map[string]any{
			"mtime_unix_nano": oldMtime,
			"size":            int64(len(staleData)),
			"file_hash":       "hash-old",
		},
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	files := map[string][]byte{
		recentIndexPath: recentData,
		oldIndexPath:    staleData,
		manifestPath:    manifestData,
	}

	readFn := func(path string) ([]byte, error) {
		data, ok := files[path]
		if !ok {
			return nil, fmt.Errorf("recency test: file not found: %s", path)
		}

		return data, nil
	}

	deps := cli.QueryDeps{
		Scan:     func(string) ([]vaultgraph.Note, error) { return nil, nil },
		Read:     readFn,
		Embedder: axisEmbedder{axes: map[string][]float32{"my-recent-activity": {1, 0, 0}}},
		ListChunkIndexes: func(string) ([]string, error) {
			return []string{recentIndexPath, oldIndexPath}, nil
		},
		Now: func() time.Time { return fixedNow },
	}

	var out bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"my-recent-activity"}, ChunksDir: chunksDir, Limit: 10},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring(recentSource+"#"+plantedAnchor),
		"planted recent chunk must surface in output (pure cosine would bury it)")
}

func TestRunQuery_RefitPendingFromCentroids(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "1.fact.md", "---\ntype: fact\nsituation: s\n---\nbody\n")

	centroidsDoc := cli.ExportVocabCentroidsDoc{
		SchemaVersion: 1,
		RefitPending:  true,
	}

	centroidsData, marshalErr := json.Marshal(centroidsDoc)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	memFS.files[filepath.Join(vault, "vocab.centroids.json")] = centroidsData

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"body"}, VaultPath: vault},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.String()).To(ContainSubstring("refit_pending: true"))
}

func TestRunQuery_TimingsFlag(t *testing.T) {
	t.Parallel()

	// scriptedClock returns times[0], times[1], ... one per call. The final
	// value repeats so a stray trailing call cannot panic. Exactly 6 reads
	// occur under --timings (newPhaseTimer + 5 marks); recency reuses the scan
	// boundary (Step 6) and adds no read.
	newScriptedClock := func(times []time.Time) func() time.Time {
		idx := 0

		return func() time.Time {
			now := times[idx]
			if idx < len(times)-1 {
				idx++
			}

			return now
		}
	}

	base := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	times := []time.Time{
		base,                              // [0] newPhaseTimer: last = now()
		base.Add(500 * time.Millisecond),  // [1] mark(scan)     -> scan = 500ms
		base.Add(1500 * time.Millisecond), // [2] mark(embed)    -> embed = 1000ms
		base.Add(1600 * time.Millisecond), // [3] mark(cluster)  -> cluster = 100ms
		base.Add(1700 * time.Millisecond), // [4] mark(nominate) -> nominate = 100ms
		base.Add(1720 * time.Millisecond), // [5] mark(render)   -> render = 20ms
	}

	g := NewWithT(t)
	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-07-13.alpha.md", "---\ntype: fact\n---\nalpha body")

	deps := newQueryDeps(memFS)
	deps.Now = newScriptedClock(times)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(), cli.QueryArgs{
		Phrases:   []string{"alpha"},
		VaultPath: vault,
		Timings:   true,
	}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got := out.String()
	g.Expect(got).To(ContainSubstring("timings:"))
	g.Expect(got).To(ContainSubstring("scan_ms: 500"))
	g.Expect(got).To(ContainSubstring("embed_ms: 1000"))
	g.Expect(got).To(ContainSubstring("cluster_ms: 100"))
	g.Expect(got).To(ContainSubstring("nominate_ms: 100"))
	g.Expect(got).To(ContainSubstring("render_ms: 20"))
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

	sidecar, err := embed.BuildSidecar(context.Background(), emb, []byte(body))
	if err != nil {
		t.Fatalf("build sidecar: %v", err)
	}

	memFS.files[filepath.Join(vault, embed.SidecarPath(relPath))] = embed.MarshalSidecar(sidecar)
}
