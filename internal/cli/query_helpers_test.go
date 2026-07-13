package cli_test

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestAppendUniqueProvenance_AddsNewRole exercises the append branch.
func TestAppendUniqueProvenance_AddsNewRole(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := cli.ExportAppendUniqueProvenance(nil, "direct")
	g.Expect(got).To(Equal([]string{"direct"}))
}

// TestAppendUniqueProvenance_AppendsDistinctRoles exercises both branches.
func TestAppendUniqueProvenance_AppendsDistinctRoles(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := cli.ExportAppendUniqueProvenance(nil, "direct", "cluster_rep", "hub", "direct")
	g.Expect(got).To(Equal([]string{"direct", "cluster_rep", "hub"}))
}

// TestAppendUniqueProvenance_DedupsExisting exercises the dedup branch.
func TestAppendUniqueProvenance_DedupsExisting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := cli.ExportAppendUniqueProvenance([]string{"direct"}, "direct", "direct")
	g.Expect(got).To(Equal([]string{"direct"}))
}

// TestBreakRepresentativeTie_HigherScoreWins exercises the score-tiebreak branch.
func TestBreakRepresentativeTie_HigherScoreWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	winner := cli.ExportBreakRepresentativeTie(0.9, "A.md", 0.5, "B.md")
	g.Expect(winner).To(Equal("A.md"))
}

// TestBreakRepresentativeTie_LexicographicPath exercises the final path tiebreak.
func TestBreakRepresentativeTie_LexicographicPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Same score → "A.md" < "B.md" → A wins.
	winner := cli.ExportBreakRepresentativeTie(0.5, "A.md", 0.5, "B.md")
	g.Expect(winner).To(Equal("A.md"))
}

// TestBreakRepresentativeTie_LexicographicPathReverse exercises the default branch.
func TestBreakRepresentativeTie_LexicographicPathReverse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Same score, "Z" > "A" → B wins.
	winner := cli.ExportBreakRepresentativeTie(0.5, "Z.md", 0.5, "A.md")
	g.Expect(winner).To(Equal("A.md"))
}

// TestBreakRepresentativeTie_SecondHigherWins exercises the reverse score-tiebreak branch.
func TestBreakRepresentativeTie_SecondHigherWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	winner := cli.ExportBreakRepresentativeTie(0.5, "A.md", 0.9, "B.md")
	g.Expect(winner).To(Equal("B.md"))
}

// fixedVectorEmbedder always returns the same vector for every input.
type fixedVectorEmbedder struct {
	modelID string
	vector  []float32
}

func (f fixedVectorEmbedder) Dims() int { return len(f.vector) }

func (f fixedVectorEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	out := make([]float32, len(f.vector))
	copy(out, f.vector)

	return out, nil
}

func (f fixedVectorEmbedder) ModelID() string { return f.modelID }

// queryParsed is the shared YAML parse target for query payload tests.
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
		CandidateL2s []struct {
			Path    string  `yaml:"path"`
			Cosine  float32 `yaml:"cosine"`
			Content string  `yaml:"content"`
		} `yaml:"candidate_l2s"`
	} `yaml:"clusters"`
	Budget struct {
		PhrasesQueried       int `yaml:"phrases_queried"`
		TotalNotes           int `yaml:"total_notes"`
		WithEmbeddings       int `yaml:"with_embeddings"`
		ClustersFound        int `yaml:"clusters_found"`
		DirectHitsReturned   int `yaml:"direct_hits_returned"`
		ItemsWithFullContent int `yaml:"items_with_full_content"`
		Limit                int `yaml:"limit"`
		ItemsContentDeduped  int `yaml:"items_content_deduped"`
	} `yaml:"budget"`
}

// findCandidateByPath returns the candidate_l2s entry across all clusters
// whose Path matches, and whether one was found — for Variant-A dedupe
// assertions (a deduped items[] note's content lives here instead).
func findCandidateByPath(parsed queryParsed, path string) (content string, found bool) {
	for _, cluster := range parsed.Clusters {
		for _, candidate := range cluster.CandidateL2s {
			if candidate.Path == path {
				return candidate.Content, true
			}
		}
	}

	return "", false
}

// findItemByPath returns the items[] entry whose Path matches, and whether
// one was found — for Variant-A dedupe assertions that need both Kind and
// Content off the same item.
func findItemByPath(parsed queryParsed, path string) (item struct {
	Path        string   `yaml:"path"`
	Kind        string   `yaml:"kind"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	ClusterID   *int     `yaml:"cluster_id,omitempty"`
	InDegree    *int     `yaml:"in_degree,omitempty"`
	Content     string   `yaml:"content"`
}, found bool,
) {
	for _, candidate := range parsed.Items {
		if candidate.Path == path {
			return candidate, true
		}
	}

	return item, false
}

// plantDualVector writes a note with distinct situation and body vectors so
// tests can exercise the max(situation,body) scoring axis selection.
func plantDualVector(
	t *testing.T,
	memFS *inMemoryFS,
	vault, relPath, content string,
	sit, body []float32,
) {
	t.Helper()

	memFS.files[filepath.Join(vault, relPath)] = []byte(content)

	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@4",
		Dims:             len(sit),
		SituationVector:  sit,
		BodyVector:       body,
		ContentHash:      embed.ContentHash([]byte(content)),
	}

	memFS.files[filepath.Join(vault, embed.SidecarPath(relPath))] = embed.MarshalSidecar(sidecar)
}

// plantWithFixedVector overrides the stub-embedder behavior by writing
// a sidecar with an exact vector, bypassing text-hash variation.
func plantWithFixedVector(
	t *testing.T,
	memFS *inMemoryFS,
	vault, relPath, body string,
	vec []float32,
) {
	t.Helper()

	notePath := filepath.Join(vault, relPath)
	memFS.files[notePath] = []byte(body)

	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@4",
		Dims:             len(vec),
		SituationVector:  vec,
		BodyVector:       vec,
		ContentHash:      embed.ContentHash([]byte(body)),
	}

	memFS.files[filepath.Join(vault, embed.SidecarPath(relPath))] = embed.MarshalSidecar(sidecar)
}
