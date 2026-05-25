package embed_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestMarshalUnmarshal_RoundTrip(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	original := embed.Sidecar{
		EmbeddingModelID: "minilm-l6-v2@384",
		Dims:             3,
		Vector:           []float32{0.1, 0.2, 0.3},
		ContentHash:      "sha256:deadbeef",
	}
	encoded := embed.MarshalSidecar(original)

	out, parseErr := embed.UnmarshalSidecar(encoded)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(out).To(Equal(original))
}

func TestSidecarPath_FromNotePath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	g.Expect(embed.SidecarPath("Permanent/132.2026-05-23.foo.md")).
		To(Equal("Permanent/132.2026-05-23.foo.vec.json"))
}

func TestSidecarPath_NonMdReturnsAppended(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	g.Expect(embed.SidecarPath("README")).To(Equal("README.vec.json"))
}

func TestUnmarshalSidecar_DimsMismatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	_, err := embed.UnmarshalSidecar([]byte(
		`{"embedding_model_id":"x@2","dims":2,"vector":[0.1,0.2,0.3],"content_hash":"sha256:abc"}`,
	))
	g.Expect(err).To(MatchError(embed.ErrDimsMismatch))
}

func TestUnmarshalSidecar_Malformed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	_, err := embed.UnmarshalSidecar([]byte("{not json"))
	g.Expect(err).To(MatchError(embed.ErrSidecarMalformed))
}
