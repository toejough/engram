package embed_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestMarshalUnmarshal_DualVector_RoundTrip(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	original := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "minilm-l6-v2@384",
		Dims:             3,
		SituationVector:  []float32{0.1, 0.2, 0.3},
		BodyVector:       []float32{0.4, 0.5, 0.6},
		ContentHash:      "sha256:deadbeef",
	}

	out, parseErr := embed.UnmarshalSidecar(embed.MarshalSidecar(original))
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(out).To(Equal(original))
}

func TestSidecarPath_FromNotePath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	g.Expect(embed.SidecarPath("132.2026-05-23.foo.md")).
		To(Equal("132.2026-05-23.foo.vec.json"))
}

func TestSidecarPath_NonMdReturnsAppended(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	g.Expect(embed.SidecarPath("README")).To(Equal("README.vec.json"))
}

func TestUnmarshalSidecar_DimsMismatch_OnEitherVector(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	bad := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@3",
		Dims:             3,
		SituationVector:  []float32{0.1, 0.2, 0.3},
		BodyVector:       []float32{0.4, 0.5}, // body short
		ContentHash:      "sha256:x",
	}

	_, parseErr := embed.UnmarshalSidecar(embed.MarshalSidecar(bad))
	g.Expect(parseErr).To(MatchError(embed.ErrDimsMismatch))
}

func TestUnmarshalSidecar_Malformed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	_, err := embed.UnmarshalSidecar([]byte("{not json"))
	g.Expect(err).To(MatchError(embed.ErrSidecarMalformed))
}

func TestUnmarshalSidecar_OldSingleVector_IsSchemaError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	// An old-format sidecar: no schema_version, single "vector" key.
	old := []byte(
		`{"embedding_model_id":"minilm-l6-v2@384","dims":3,"vector":[0.1,0.2,0.3],"content_hash":"sha256:x"}`,
	)

	_, parseErr := embed.UnmarshalSidecar(old)
	g.Expect(parseErr).To(MatchError(embed.ErrSchemaVersion))
}
