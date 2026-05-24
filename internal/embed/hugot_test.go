package embed_test

import (
	stdembed "embed"
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// emptyFS is a zero-value stdembed.FS with no entries — used to verify
// the UAT 10 missing-model error path returns a clear, actionable message.
var emptyFS stdembed.FS

func TestT10_MissingBundledModel_ClearError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := embed.NewHugotEmbedderFromFS(context.Background(), emptyFS, "assets/model", "x@1")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ENGRAM_MODEL_PATH"))
}

// TestBundledHugotEmbedder_Smoke exercises the production constructor
// end-to-end. Skipped under -short because it unpacks the ~90MB ONNX.
func TestBundledHugotEmbedder_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bundled-embedder smoke test under -short")
	}

	t.Parallel()

	g := NewWithT(t)

	embedder, err := embed.NewBundledHugotEmbedder(t.Context())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defer func() {
		_ = embedder.Close()
	}()

	const expectedDims = 384

	g.Expect(embedder.ModelID()).To(Equal(embed.BundledModelID))
	g.Expect(embedder.Dims()).To(Equal(expectedDims))

	vec, embErr := embedder.Embed(t.Context(), "hello world")
	g.Expect(embErr).NotTo(HaveOccurred())
	g.Expect(vec).To(HaveLen(expectedDims))
}
