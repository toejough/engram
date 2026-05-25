package cli_test

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

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
	scBytes, marshalErr := embed.MarshalSidecar(sidecar)
	if marshalErr != nil {
		t.Fatalf("marshal sidecar: %v", marshalErr)
	}

	memFS.files[filepath.Join(vault, embed.SidecarPath(relPath))] = scBytes
}

func TestQuery_EmptyVault_ItemsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	var out bytes.Buffer
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "anything", VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []any `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).To(BeEmpty())
}

func TestQuery_NotesButNoSidecars_ErrorWithRecoveryHint(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	memFS.files[filepath.Join(vault, "Permanent/1.foo.md")] = []byte("body")

	var out bytes.Buffer
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Query: "anything", VaultPath: vault},
		newQueryDeps(memFS), &out)

	g.Expect(err).To(MatchError(ContainSubstring("engram embed apply --all")))
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
		cli.QueryArgs{Query: "the query string body", VaultPath: vault, Limit: 2},
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
		cli.QueryArgs{Query: "body", VaultPath: vault, Limit: 2},
		newQueryDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())

	var parsed struct {
		Items []any `yaml:"items"`
	}

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).To(HaveLen(2))
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
		cli.QueryArgs{Query: "x", VaultPath: vault}, deps, &out)

	g.Expect(err).To(MatchError(ContainSubstring("embed")))
}

type errorEmbedder struct{}

func (errorEmbedder) Embed(context.Context, string) ([]float32, error) {
	return nil, errors.New("embedder down")
}

func (errorEmbedder) ModelID() string { return "m@4" }
func (errorEmbedder) Dims() int       { return 4 }
