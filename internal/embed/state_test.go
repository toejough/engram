package embed_test

import (
	"encoding/json"
	"io/fs"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestComputeState_Broken_BadJSON(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	filesystem := fakeFS{
		"x.md":       []byte("body\n"),
		"x.vec.json": []byte("{not json"),
	}

	state := embed.ComputeState(filesystem, "x.md", "model@384")
	g.Expect(state).To(Equal(embed.StateBroken))
}

func TestComputeState_Broken_DimsMismatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	// A current-schema sidecar with a genuine dims mismatch (body vector
	// shorter than Dims) must classify Broken, not Incompatible — the schema
	// check passes, the vector-length check fails.
	filesystem := fakeFS{
		"x.md": []byte("body\n"),
		"x.vec.json": []byte(
			`{"schema_version":1,"embedding_model_id":"model@384","dims":3,` +
				`"situation_vector":[0.1,0.2,0.3],"body_vector":[0.1,0.2],"content_hash":"sha256:abc"}`,
		),
	}

	state := embed.ComputeState(filesystem, "x.md", "model@384")
	g.Expect(state).To(Equal(embed.StateBroken))
}

func TestComputeState_Incompatible(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	noteBytes := []byte("---\nx: 1\n---\nbody\n")
	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "OLDmodel@256",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.1},
		ContentHash:      embed.ContentHash(noteBytes),
	}
	filesystem := fakeFS{
		"x.md":       noteBytes,
		"x.vec.json": mustSidecar(t, sidecar),
	}

	state := embed.ComputeState(filesystem, "x.md", "model@384")
	g.Expect(state).To(Equal(embed.StateIncompatible))
}

func TestComputeState_Missing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	filesystem := fakeFS{"x.md": []byte("---\nx: 1\n---\nbody\n")}

	state := embed.ComputeState(filesystem, "x.md", "model@384")
	g.Expect(state).To(Equal(embed.StateMissing))
}

func TestComputeState_OK(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	noteBytes := []byte("---\nx: 1\n---\nbody\n")
	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "model@384",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.1},
		ContentHash:      embed.ContentHash(noteBytes),
	}
	filesystem := fakeFS{
		"x.md":       noteBytes,
		"x.vec.json": mustSidecar(t, sidecar),
	}

	state := embed.ComputeState(filesystem, "x.md", "model@384")
	g.Expect(state).To(Equal(embed.StateOK))
}

func TestComputeState_OK_AfterLinkOnlyEdit(t *testing.T) {
	t.Parallel()

	// A note whose body-only change was adding a "Related to:" section must
	// remain StateOK — the ContentHash excludes that section (D3), so no
	// re-embed is triggered.
	g := NewWithT(t)

	baseNote := []byte("---\ntype: fact\nluhmann: \"1\"\n---\nbody content here.\n")
	noteWithLinks := []byte(
		"---\ntype: fact\nluhmann: \"1\"\n---\nbody content here.\n" +
			"Related to:\n- [[105.2024-01-01.some-note]] — context.\n",
	)

	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "model@384",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.1},
		ContentHash:      embed.ContentHash(baseNote),
	}

	filesystem := fakeFS{
		"n.md":       noteWithLinks,
		"n.vec.json": mustSidecar(t, sidecar),
	}

	state := embed.ComputeState(filesystem, "n.md", "model@384")
	g.Expect(state).To(Equal(embed.StateOK))
}

func TestComputeState_OldSchemaSidecar_IsIncompatible(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	note := []byte("---\ntype: fact\nsituation: x\n---\n\nbody\n")
	oldSidecar := []byte(
		`{"embedding_model_id":"minilm-l6-v2@384","dims":3,"vector":[0.1,0.2,0.3],"content_hash":"sha256:x"}`,
	)
	filesystem := fakeFS{
		"n.md":       note,
		"n.vec.json": oldSidecar,
	}

	state := embed.ComputeState(filesystem, "n.md", "minilm-l6-v2@384")
	g.Expect(state).To(Equal(embed.StateIncompatible))
}

func TestComputeState_Stale(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "model@384",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.1},
		ContentHash:      "sha256:stalehash",
	}
	filesystem := fakeFS{
		"x.md":       []byte("---\nx: 1\n---\nbody\n"),
		"x.vec.json": mustSidecar(t, sidecar),
	}

	state := embed.ComputeState(filesystem, "x.md", "model@384")
	g.Expect(state).To(Equal(embed.StateStale))
}

func TestComputeState_Stale_AfterBodyChange_BeyondLinks(t *testing.T) {
	t.Parallel()

	// A note whose actual body content changed (not just links) must be Stale.
	// This confirms D3 doesn't accidentally suppress real staleness detection.
	g := NewWithT(t)

	original := []byte("---\ntype: fact\nluhmann: \"1\"\n---\noriginal body.\n")
	edited := []byte(
		"---\ntype: fact\nluhmann: \"1\"\n---\nedited body.\n" +
			"Related to:\n- [[105]] — context.\n",
	)

	sidecar := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "model@384",
		Dims:             1,
		SituationVector:  []float32{0.1},
		BodyVector:       []float32{0.1},
		ContentHash:      embed.ContentHash(original),
	}

	filesystem := fakeFS{
		"n.md":       edited,
		"n.vec.json": mustSidecar(t, sidecar),
	}

	state := embed.ComputeState(filesystem, "n.md", "model@384")
	g.Expect(state).To(Equal(embed.StateStale))
}

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
