package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

func TestEmbedApply_DryRunDoesNotWrite(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	memFS.files[filepath.Join(vault, "1.foo.md")] = []byte("body")

	var out bytes.Buffer

	err := cli.RunEmbedApply(context.Background(),
		cli.EmbedApplyArgs{VaultPath: vault, DryRun: true},
		newEmbedDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("would-embed 1.foo.md (missing)"))
	g.Expect(memFS.files).NotTo(HaveKey(filepath.Join(vault, "1.foo.vec.json")))
}

func TestEmbedApply_MissingOnly_WritesSidecar(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	memFS.files[filepath.Join(vault, "1.foo.md")] = []byte("---\ntype: fact\n---\nbody\n")

	var out bytes.Buffer

	err := cli.RunEmbedApply(context.Background(),
		cli.EmbedApplyArgs{VaultPath: vault},
		newEmbedDeps(memFS), &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("embedded  1.foo.md (missing)"))

	sidecarPath := filepath.Join(vault, "1.foo.vec.json")
	g.Expect(memFS.files).To(HaveKey(sidecarPath))

	var sidecar embed.Sidecar
	g.Expect(json.Unmarshal(memFS.files[sidecarPath], &sidecar)).NotTo(HaveOccurred())
	g.Expect(sidecar.SchemaVersion).To(Equal(embed.SidecarSchemaVersion))
	g.Expect(sidecar.EmbeddingModelID).To(Equal("m@4"))
	g.Expect(sidecar.Dims).To(Equal(4))
	g.Expect(sidecar.SituationVector).To(HaveLen(4))
	g.Expect(sidecar.BodyVector).To(HaveLen(4))
	g.Expect(sidecar.ContentHash).To(HavePrefix("sha256:"))
}

func TestEmbedApply_StaleVsIncompatibleDistinction(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	notePath := filepath.Join(vault, "1.foo.md")
	scPath := filepath.Join(vault, "1.foo.vec.json")
	noteBody := []byte("---\ntype: fact\n---\nbody\n")
	memFS := newInMemoryFS()
	memFS.files[notePath] = noteBody

	// Plant an incompatible sidecar (different model_id).
	incompatSidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "OLD@384",
		Dims:             4,
		SituationVector:  []float32{1, 1, 1, 1},
		BodyVector:       []float32{1, 1, 1, 1},
		ContentHash:      embed.ContentHash(noteBody),
	}
	incompatBytes := embed.MarshalSidecar(incompatSidecar)
	memFS.files[scPath] = incompatBytes

	// --stale should not touch incompatible sidecars.
	var out bytes.Buffer

	err := cli.RunEmbedApply(context.Background(),
		cli.EmbedApplyArgs{VaultPath: vault, Stale: true},
		newEmbedDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(memFS.files[scPath]).To(Equal(incompatBytes), "stale must not rewrite incompatible")

	// --force should re-embed it.
	out.Reset()
	err = cli.RunEmbedApply(context.Background(),
		cli.EmbedApplyArgs{VaultPath: vault, Force: true, All: true},
		newEmbedDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(memFS.files[scPath]).NotTo(Equal(incompatBytes), "force must rewrite incompatible")

	var fresh embed.Sidecar
	g.Expect(json.Unmarshal(memFS.files[scPath], &fresh)).NotTo(HaveOccurred())
	g.Expect(fresh.EmbeddingModelID).To(Equal("m@4"))
}

func TestEmbedApply_WriteFailureReported(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	memFS.files[filepath.Join(vault, "1.foo.md")] = []byte("body")

	deps := newEmbedDeps(memFS)
	deps.Write = func(string, []byte) error { return errors.New("disk full") }

	var out bytes.Buffer

	err := cli.RunEmbedApply(context.Background(), cli.EmbedApplyArgs{VaultPath: vault}, deps, &out)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("fail      1.foo.md"))
	g.Expect(out.String()).To(ContainSubstring("disk full"))
}

func TestEmbedStatus_AllMissing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	memFS.files[filepath.Join(vault, "1.foo.md")] = []byte("body")
	memFS.files[filepath.Join(vault, "2.bar.md")] = []byte("body")

	var out bytes.Buffer

	err := cli.RunEmbedStatus(
		context.Background(),
		cli.EmbedStatusArgs{VaultPath: vault},
		newEmbedDeps(memFS),
		&out,
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("total:           2"))
	g.Expect(out.String()).To(ContainSubstring("with-embeddings: 0"))
	g.Expect(out.String()).To(ContainSubstring("without:         2"))
}

// inMemoryFS is a tiny map-backed filesystem used by embed CLI tests so
// the EmbedDeps don't need a real disk.
type inMemoryFS struct {
	files map[string][]byte
}

func (m *inMemoryFS) Read(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}

	return data, nil
}

func (m *inMemoryFS) Scan(_ string) ([]vaultgraph.Note, error) {
	notes := make([]vaultgraph.Note, 0, len(m.files))
	seen := map[string]bool{}

	for path, body := range m.files {
		if !strings.HasSuffix(path, ".md") {
			continue
		}

		base := filepath.Base(strings.TrimSuffix(path, ".md"))

		if seen[base] {
			continue
		}

		seen[base] = true
		notes = append(notes, vaultgraph.Note{
			Basename: base,
			Outgoing: vaultgraph.ParseWikilinks(body),
		})
	}

	sort.Slice(notes, func(i, j int) bool { return notes[i].Basename < notes[j].Basename })

	return notes, nil
}

func (m *inMemoryFS) Write(path string, data []byte) error {
	m.files[path] = data

	return nil
}

// stubEmbedder produces a deterministic vector by hashing text bytes.
type stubEmbedder struct {
	modelID string
	dims    int
}

func (s stubEmbedder) Dims() int { return s.dims }

func (s stubEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	out := make([]float32, s.dims)

	for i, char := range []byte(text) {
		out[i%s.dims] += float32(char) / 100
	}

	return out, nil
}

func (s stubEmbedder) ModelID() string { return s.modelID }

func newEmbedDeps(memFS *inMemoryFS) cli.EmbedDeps {
	return cli.EmbedDeps{
		Scan:     memFS.Scan,
		Read:     memFS.Read,
		Write:    memFS.Write,
		Embedder: stubEmbedder{modelID: "m@4", dims: 4},
	}
}

func newInMemoryFS() *inMemoryFS { return &inMemoryFS{files: map[string][]byte{}} }
