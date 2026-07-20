package embed_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestEmbedRetriesShorterOnOverLengthFailure(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	handle := &lengthCappedHandle{capChars: 700}
	embedder, err := embed.NewHugotEmbedderFromDir(context.Background(), cappedBackend{handle: handle}, "dir", "m@2")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if embedder == nil {
		return
	}

	// 1400 chars passes the char guard but "tokenizes" over the cap; the
	// embedder must back off to a shorter prefix instead of failing ingest.
	vec, err := embedder.Embed(context.Background(), strings.Repeat("x", 1400))

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(vec).To(gomega.Equal([]float32{1, 2}))
	g.Expect(handle.calls).To(gomega.BeNumerically(">", 1), "must have retried shorter")
}

func TestEmbedStillFailsOnPersistentError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// capChars 10: the builder's short probe passes, but the real input fails
	// at every halving step down to the floor — the error must surface, not
	// loop forever.
	handle := &lengthCappedHandle{capChars: 10}
	embedder, err := embed.NewHugotEmbedderFromDir(context.Background(), cappedBackend{handle: handle}, "dir", "m@2")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if embedder == nil {
		return
	}

	_, err = embedder.Embed(context.Background(), strings.Repeat("x", 1400))

	g.Expect(err).To(gomega.HaveOccurred())
}

// unexported variables.
var (
	errOverLength = errors.New("dimension of axis #1 doesn't match and cannot be broadcast")
)

type cappedBackend struct{ handle *lengthCappedHandle }

func (cappedBackend) Close() error { return nil }

func (b cappedBackend) OpenPipeline(
	_ context.Context, _ string,
) (embed.PipelineHandle, error) {
	return b.handle, nil
}

// lengthCappedHandle fails RunPipeline whenever the input exceeds capChars,
// mimicking MiniLM's 512-token positional limit blowing up on code-dense text
// that fits the char guard but not the token budget.
type lengthCappedHandle struct {
	capChars int
	calls    int
}

func (h *lengthCappedHandle) Destroy() error { return nil }

func (h *lengthCappedHandle) RunPipeline(
	_ context.Context, inputs []string,
) (embed.FeatureOutput, error) {
	h.calls++
	if len(inputs) > 0 && len(inputs[0]) > h.capChars {
		return embed.FeatureOutput{}, errOverLength
	}

	return embed.FeatureOutput{Embeddings: [][]float32{{1, 2}}}, nil
}
