package cli_test

// Scenario tests for the Query Engine technical spec (792b24, docs/specs/technical/query-engine.md):
// 792b24c12a7 (project filter on explore and ride-along notes), 792b24c24a1 (stage readings) and
// 792b24c4a3 (the floor uses the unscaled cosine). Helpers carry the qe1 prefix.

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// Test792b24c12a7_QueryEngine_ExploreAndRideAlongNotesAreProjectFiltered verifies the project filter
// reaches the explore half and the superseding (ride-along) notes: with --project alpha, a beta explore
// note and a beta note superseding an alpha match are dropped from the items and the clusters (as is a
// matched beta note, which reaches the clusters), an alpha explore note stays, and the chunk items are
// unchanged by the filter.
func Test792b24c12a7_QueryEngine_ExploreAndRideAlongNotesAreProjectFiltered(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		alphaMatch   = "1.alpha-one.md"
		alphaSecond  = "2.alpha-two.md"
		betaRide     = "3.beta-successor.md"
		betaExplore  = "4.beta-explore.md"
		alphaExplore = "5.alpha-explore.md"
		betaMatched  = "6.beta-matched.md"
	)

	fixture := qe1NewProjectFixture(t)

	// Fixture validity: without a project the beta explore and ride-along notes are delivered, so a
	// filtered run that lacks them is the filter's doing, not an unreachable fixture.
	unfiltered, ok := qe1RunProjectQuery(t, fixture, "")
	if !ok {
		return
	}

	g.Expect(qe1ItemPaths(unfiltered)).To(ContainElements(betaRide, betaExplore, alphaExplore))
	g.Expect(qe1ClusterPaths(unfiltered)).To(ContainElement(betaMatched))

	filtered, ok := qe1RunProjectQuery(t, fixture, "alpha")
	if !ok {
		return
	}

	g.Expect(qe1ItemPaths(filtered)).To(ContainElements(alphaMatch, alphaSecond, alphaExplore))

	for _, betaNote := range []string{betaRide, betaExplore, betaMatched} {
		g.Expect(qe1ItemPaths(filtered)).NotTo(ContainElement(betaNote))
		g.Expect(qe1ClusterPaths(filtered)).NotTo(ContainElement(betaNote))
	}

	g.Expect(qe1ChunkItems(filtered)).NotTo(BeEmpty())
	g.Expect(qe1ChunkItems(filtered)).To(Equal(qe1ChunkItems(unfiltered)))
}

// Test792b24c24a1_QueryEngine_EachStageEndsWithAReading verifies that, with timing on and a clock that
// advances a different amount between consecutive readings, each of scan_ms, embed_ms, cluster_ms and
// nominate_ms is the gap between its stage's readings, and that the query reads the clock once at its
// start and once at the end of each stage and nowhere else (recency reuses the scan-stage reading).
func Test792b24c24a1_QueryEngine_EachStageEndsWithAReading(t *testing.T) {
	t.Parallel()

	const (
		scanGap      = 300 * time.Millisecond
		embedGap     = 700 * time.Millisecond
		clusterGap   = 40 * time.Millisecond
		nominateGap  = 90 * time.Millisecond
		renderGap    = 15 * time.Millisecond
		expectedRead = 6 // the start, scan, embed, cluster, nominate, and the render stage's end
	)

	g := NewWithT(t)

	gaps := []time.Duration{scanGap, embedGap, clusterGap, nominateGap, renderGap}
	clock := &qe1AdvancingClock{next: time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC), gaps: gaps}

	vault := t.TempDir()
	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, vault, "1.2026-07-13.alpha.md", "---\ntype: fact\n---\nalpha body")

	deps := newQueryDeps(memFS)
	deps.Now = clock.now

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

	var payload struct {
		Timings struct {
			ScanMS     int64 `yaml:"scan_ms"`
			EmbedMS    int64 `yaml:"embed_ms"`
			ClusterMS  int64 `yaml:"cluster_ms"`
			NominateMS int64 `yaml:"nominate_ms"`
		} `yaml:"timings"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &payload)).To(Succeed())
	g.Expect(payload.Timings.ScanMS).To(Equal(scanGap.Milliseconds()))
	g.Expect(payload.Timings.EmbedMS).To(Equal(embedGap.Milliseconds()))
	g.Expect(payload.Timings.ClusterMS).To(Equal(clusterGap.Milliseconds()))
	g.Expect(payload.Timings.NominateMS).To(Equal(nominateGap.Milliseconds()))
	g.Expect(clock.reads).To(Equal(expectedRead))
}

// Test792b24c4a3_QueryEngine_FloorUsesTheUnscaledCosine verifies that a chunk with cosine 0.53 ingested
// 120 days before the clock (two half-lives, factor 0.25) is a direct match scored about 0.13: the
// 0.25 floor tests the unscaled cosine, not the recency-scaled score that falls below it.
func Test792b24c4a3_QueryEngine_FloorUsesTheUnscaledCosine(t *testing.T) {
	t.Parallel()

	const (
		chunkAnchor    = "para-1"
		chunkSource    = "/s/a.jsonl"
		ageDays        = 120
		unscaledCosine = 0.53
		expectedScore  = 0.13
		scoreTolerance = 0.005
		hoursPerDay    = 24
	)

	g := NewWithT(t)

	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	queryVec := []float32{1, 0, 0, 0}
	// A unit vector whose cosine against queryVec is unscaledCosine.
	chunkVec := []float32{unscaledCosine, 0.8479976, 0, 0}

	data, err := chunk.EncodeRecords([]chunk.Record{{
		Source:      chunkSource,
		Anchor:      chunkAnchor,
		ContentHash: chunk.HashText("an old chunk"),
		Text:        "an old chunk",
		Vector:      chunkVec,
		IngestedAt:  now.Add(-ageDays * hoursPerDay * time.Hour),
	}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	memFS := newInMemoryFS()
	memFS.files["/chunks/a.jsonl"] = data

	deps := newQueryDeps(memFS)
	deps.Scan = func(string) ([]vaultgraph.Note, error) { return nil, nil }
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
	deps.ListChunkIndexes = func(string) ([]string, error) { return []string{"/chunks/a.jsonl"}, nil }
	deps.Now = func() time.Time { return now }

	var out bytes.Buffer

	err = cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: t.TempDir(), ChunksDir: "/chunks"},
		deps, &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).To(Succeed())

	var found *queryParsedItem

	for i := range parsed.Items {
		if parsed.Items[i].Path == chunkSource+"#"+chunkAnchor {
			found = &parsed.Items[i]
		}
	}

	g.Expect(found).NotTo(BeNil(), "the old chunk must be an item")

	if found == nil {
		return
	}

	g.Expect(found.Provenances).To(Equal([]string{"direct"}))
	g.Expect(found.Score).To(BeNumerically("~", expectedScore, scoreTolerance))
}

// qe1AdvancingClock is a clock that advances by the next gap on every reading after the first and counts
// its readings; once the gaps run out it stands still.
type qe1AdvancingClock struct {
	next  time.Time
	gaps  []time.Duration
	reads int
}

func (c *qe1AdvancingClock) now() time.Time {
	current := c.next
	if c.reads < len(c.gaps) {
		c.next = c.next.Add(c.gaps[c.reads])
	}

	c.reads++

	return current
}

// qe1ProjectFixture is the in-memory vault of the project-filter scenario: alpha matches, a beta note
// superseding an alpha match, beta and alpha explore-half notes, a matched beta note (the clusters' check)
// and one matching chunk.
type qe1ProjectFixture struct {
	deps  cli.QueryDeps
	vault string
}

func qe1ChunkItems(parsed queryParsed) []queryParsedItem {
	var chunks []queryParsedItem

	for _, item := range parsed.Items {
		if item.Kind == "chunk" {
			chunks = append(chunks, item)
		}
	}

	return chunks
}

func qe1ClusterPaths(parsed queryParsed) []string {
	var paths []string

	for _, clusterEntry := range parsed.Clusters {
		for _, member := range clusterEntry.Members {
			paths = append(paths, member.Path)
		}

		for _, candidate := range clusterEntry.CandidateL2s {
			paths = append(paths, candidate.Path)
		}
	}

	return paths
}

func qe1ItemPaths(parsed queryParsed) []string {
	paths := make([]string, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		paths = append(paths, item.Path)
	}

	return paths
}

func qe1NewProjectFixture(t *testing.T) qe1ProjectFixture {
	t.Helper()

	const dims = 4

	matchVec := []float32{1, 0, 0, 0}
	exploreVec := []float32{0, 1, 0, 0}

	emb := tokenEmbedder{
		modelID: "qe1@4",
		dims:    dims,
		vectors: map[string][]float32{"MATCHTOKEN": matchVec, "EXPLORETOKEN": exploreVec},
	}

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithEmbedder(t, memFS, vault, "1.alpha-one.md",
		"---\ntype: fact\nproject: alpha\nsituation: s\n---\nMATCHTOKEN one\n", emb)
	plantNoteWithEmbedder(t, memFS, vault, "2.alpha-two.md",
		"---\ntype: fact\nproject: alpha\nsituation: s\n---\nMATCHTOKEN two\n", emb)
	plantNoteWithEmbedder(t, memFS, vault, "3.beta-successor.md",
		"---\ntype: fact\nproject: beta\nsituation: s\nsupersedes:\n  - note: 1.alpha-one\n"+
			"    type: updates\n    claim: one was wrong\n---\nsuccessor body\n", emb)
	plantNoteWithEmbedder(t, memFS, vault, "4.beta-explore.md",
		"---\ntype: fact\nproject: beta\nsituation: s\ntags:\n    - vocab/topic-a\n---\nEXPLORETOKEN beta\n", emb)
	plantNoteWithEmbedder(t, memFS, vault, "5.alpha-explore.md",
		"---\ntype: fact\nproject: alpha\nsituation: s\ntags:\n    - vocab/topic-a\n---\nEXPLORETOKEN alpha\n", emb)
	plantNoteWithEmbedder(t, memFS, vault, "6.beta-matched.md",
		"---\ntype: fact\nproject: beta\nsituation: s\n---\nMATCHTOKEN beta\n", emb)

	centroids, err := json.Marshal(cli.ExportVocabCentroidsDoc{
		SchemaVersion: 1,
		Terms:         map[string]cli.ExportVocabCentroidEntry{"topic-a": {Vector: exploreVec, MemberCount: 2}},
	})
	if err != nil {
		t.Fatalf("marshal centroids: %v", err)
	}

	memFS.files[filepath.Join(vault, "vocab.centroids.json")] = centroids

	chunkData, err := chunk.EncodeRecords([]chunk.Record{{
		Source: "/s/a.jsonl", Anchor: "para-1", ContentHash: chunk.HashText("MATCHTOKEN chunk"),
		Text: "MATCHTOKEN chunk", Vector: matchVec,
	}})
	if err != nil {
		t.Fatalf("encode chunk: %v", err)
	}

	memFS.files["/chunks/a.jsonl"] = chunkData

	return qe1ProjectFixture{
		vault: vault,
		deps: cli.QueryDeps{
			Scan:             memFS.Scan,
			Read:             memFS.Read,
			Embedder:         emb,
			ListChunkIndexes: func(string) ([]string, error) { return []string{"/chunks/a.jsonl"}, nil },
		},
	}
}

func qe1RunProjectQuery(t *testing.T, fixture qe1ProjectFixture, project string) (queryParsed, bool) {
	t.Helper()

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(), cli.QueryArgs{
		Phrases:   []string{"MATCHTOKEN"},
		VaultPath: fixture.vault,
		ChunksDir: "/chunks",
		Project:   project,
	}, fixture.deps, &out)
	if err != nil {
		t.Errorf("RunQuery(project=%q): %v", project, err)

		return queryParsed{}, false
	}

	var parsed queryParsed

	err = yaml.Unmarshal(out.Bytes(), &parsed)
	if err != nil {
		t.Errorf("decode payload: %v", err)

		return queryParsed{}, false
	}

	return parsed, true
}
