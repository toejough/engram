package embed_test

import (
	"encoding/json"
	"io/fs"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// fakeFS is a trivial in-memory file map for ComputeState tests.
type fakeFS map[string][]byte

func (f fakeFS) ReadFile(path string) ([]byte, error) {
	data, ok := f[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}

	return data, nil
}

func mustSidecar(t *testing.T, s embed.Sidecar) []byte {
	t.Helper()

	out, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return out
}

func TestComputeState_Missing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	filesystem := fakeFS{"Permanent/x.md": []byte("---\nx: 1\n---\nbody\n")}

	state, err := embed.ComputeState(filesystem, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(state).To(Equal(embed.StateMissing))
}

func TestComputeState_OK(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	noteBytes := []byte("---\nx: 1\n---\nbody\n")
	sidecar := embed.Sidecar{
		EmbeddingModelID: "model@384",
		Dims:             1,
		Vector:           []float32{0.1},
		ContentHash:      embed.ContentHash(noteBytes),
	}
	filesystem := fakeFS{
		"Permanent/x.md":       noteBytes,
		"Permanent/x.vec.json": mustSidecar(t, sidecar),
	}

	state, err := embed.ComputeState(filesystem, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(state).To(Equal(embed.StateOK))
}

func TestComputeState_Stale(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	sidecar := embed.Sidecar{
		EmbeddingModelID: "model@384",
		Dims:             1,
		Vector:           []float32{0.1},
		ContentHash:      "sha256:stalehash",
	}
	filesystem := fakeFS{
		"Permanent/x.md":       []byte("---\nx: 1\n---\nbody\n"),
		"Permanent/x.vec.json": mustSidecar(t, sidecar),
	}

	state, err := embed.ComputeState(filesystem, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(state).To(Equal(embed.StateStale))
}

func TestComputeState_Incompatible(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	noteBytes := []byte("---\nx: 1\n---\nbody\n")
	sidecar := embed.Sidecar{
		EmbeddingModelID: "OLDmodel@256",
		Dims:             1,
		Vector:           []float32{0.1},
		ContentHash:      embed.ContentHash(noteBytes),
	}
	filesystem := fakeFS{
		"Permanent/x.md":       noteBytes,
		"Permanent/x.vec.json": mustSidecar(t, sidecar),
	}

	state, err := embed.ComputeState(filesystem, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(state).To(Equal(embed.StateIncompatible))
}

func TestComputeState_Broken_BadJSON(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	filesystem := fakeFS{
		"Permanent/x.md":       []byte("body\n"),
		"Permanent/x.vec.json": []byte("{not json"),
	}

	state, err := embed.ComputeState(filesystem, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(state).To(Equal(embed.StateBroken))
}

func TestComputeState_Broken_DimsMismatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	filesystem := fakeFS{
		"Permanent/x.md": []byte("body\n"),
		"Permanent/x.vec.json": []byte(
			`{"embedding_model_id":"model@384","dims":2,"vector":[0.1,0.2,0.3],"content_hash":"sha256:abc"}`,
		),
	}

	state, err := embed.ComputeState(filesystem, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(state).To(Equal(embed.StateBroken))
}
