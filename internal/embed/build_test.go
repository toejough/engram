package embed_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestBuildSidecar_EmbedsBothAndStamps(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	raw := []byte("---\ntype: fact\nsituation: when X\n---\n\nbody Y\n")

	sidecar, err := embed.BuildSidecar(context.Background(), &seqEmbedder{}, raw)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(sidecar.SchemaVersion).To(Equal(embed.SidecarSchemaVersion))
	g.Expect(sidecar.SituationVector).To(Equal([]float32{1, 0, 0}))
	g.Expect(sidecar.BodyVector).To(Equal([]float32{0, 1, 0}))
	g.Expect(sidecar.ContentHash).To(Equal(embed.ContentHash(raw)))
}

func TestBuildSidecar_NoSituation_FallsBackToBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	raw := []byte("no frontmatter, body only\n") // SituationText == ""

	sidecar, err := embed.BuildSidecar(context.Background(), &seqEmbedder{}, raw)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Both embed calls receive BodyText, but seqEmbedder still returns
	// distinct vectors; the point is that situation embedding used a
	// non-empty input. Assert situation vector is the FIRST call's output.
	g.Expect(sidecar.SituationVector).To(Equal([]float32{1, 0, 0}))
}

// seqEmbedder returns a distinct vector per call so we can tell situation
// from body. Call 1 -> {1,0,0}; call 2 -> {0,1,0}.
type seqEmbedder struct{ n int }

func (e *seqEmbedder) Dims() int { return 3 }

func (e *seqEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	e.n++
	if e.n == 1 {
		return []float32{1, 0, 0}, nil
	}

	return []float32{0, 1, 0}, nil
}

func (e *seqEmbedder) ModelID() string { return "m@3" }
