package embed_test

import (
	"context"
	"crypto/sha256"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestBuildSidecar_EmbedsBothAndStamps(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	// Distinct situation and body so a swapped-input bug produces distinct,
	// detectably-wrong vectors.
	raw := []byte("---\ntype: fact\nsituation: when X\n---\n\nbody Y\n")

	sidecar, err := embed.BuildSidecar(context.Background(), hashEmbedder{}, raw)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(sidecar.SchemaVersion).To(Equal(embed.SidecarSchemaVersion))
	// Each vector is tied to its specific input: situation slot embeds the
	// situation text, body slot embeds the body text. A bug that swaps the
	// two inputs fails here because vecFor(situation) != vecFor(body).
	g.Expect(sidecar.SituationVector).To(Equal(vecFor(string(embed.SituationText(raw)))))
	g.Expect(sidecar.BodyVector).To(Equal(vecFor(string(embed.BodyText(raw)))))
	g.Expect(sidecar.ContentHash).To(Equal(embed.ContentHash(raw)))
}

func TestBuildSidecar_NoSituation_FallsBackToBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	raw := []byte("no frontmatter, body only\n") // SituationText(raw) == ""

	sidecar, err := embed.BuildSidecar(context.Background(), hashEmbedder{}, raw)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// With no situation field, the fallback embeds BodyText into the
	// situation slot. An input-deterministic embedder maps identical input
	// to identical output, so the two vectors MUST be equal. Removing the
	// fallback would embed "" into the situation slot -> a different vector.
	g.Expect(sidecar.SituationVector).To(Equal(sidecar.BodyVector))
}

// hashEmbedder is an input-deterministic fake: its output is a pure function
// of the input text, so different text -> different vector and identical text
// -> identical vector. This lets the tests verify which input fed which slot
// (the dual-vector routing contract) rather than mere call order.
type hashEmbedder struct{}

func (hashEmbedder) Dims() int { return 3 }

func (hashEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return vecFor(text), nil
}

func (hashEmbedder) ModelID() string { return "m@3" }

// vecFor maps text to a deterministic 3-dim vector via its sha256 digest.
// Distinct inputs yield distinct vectors; identical inputs yield identical
// vectors.
func vecFor(text string) []float32 {
	sum := sha256.Sum256([]byte(text))

	return []float32{float32(sum[0]), float32(sum[1]), float32(sum[2])}
}
