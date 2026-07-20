package embed_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestBuildEmbedder_EmptyProbeDestroysAndReportsSentinel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	handle := &fakePipelineHandle{runResult: embed.FeatureOutput{Embeddings: [][]float32{}}}
	_, err := embed.NewHugotEmbedderFromDir(
		context.Background(),
		&fakeBackend{handle: handle},
		"/tmp/x", "fake@1",
	)
	g.Expect(err).To(MatchError(embed.ErrHugotProbeEmpty))
	g.Expect(handle.destroyHits).To(Equal(1))
}

func TestBuildEmbedder_HappyPathBindsRunnerAndCloser(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	handle := &fakePipelineHandle{
		runResult: embed.FeatureOutput{Embeddings: [][]float32{{0.1, 0.2}}},
	}
	emb, err := embed.NewHugotEmbedderFromDir(
		context.Background(),
		&fakeBackend{handle: handle},
		"/tmp/x", "fake@2",
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(emb.Dims()).To(Equal(2))
	g.Expect(emb.ModelID()).To(Equal("fake@2"))

	g.Expect(emb.Close()).NotTo(HaveOccurred())
	g.Expect(handle.destroyHits).To(Equal(1))
}

func TestBuildEmbedder_OpenFailurePropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bootErr := errors.New("session blocked")
	_, err := embed.NewHugotEmbedderFromDir(
		context.Background(),
		&fakeBackend{openErr: bootErr},
		"/tmp/x", "fake@1",
	)
	g.Expect(err).To(MatchError(bootErr))
}

func TestBuildEmbedder_ProbeFailureDestroysAndPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	probeErr := errors.New("probe blocked")
	handle := &fakePipelineHandle{runErr: probeErr}
	_, err := embed.NewHugotEmbedderFromDir(
		context.Background(),
		&fakeBackend{handle: handle},
		"/tmp/x", "fake@1",
	)
	g.Expect(err).To(MatchError(probeErr))
	g.Expect(handle.destroyHits).To(Equal(1), "destroy must fire on probe failure")
}

func TestBuildEmbedder_ZeroLengthVectorIsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	handle := &fakePipelineHandle{runResult: embed.FeatureOutput{Embeddings: [][]float32{{}}}}
	_, err := embed.NewHugotEmbedderFromDir(
		context.Background(),
		&fakeBackend{handle: handle},
		"/tmp/x", "fake@1",
	)
	g.Expect(err).To(MatchError(embed.ErrHugotProbeEmpty))
	g.Expect(handle.destroyHits).To(Equal(1))
}

// fakeBackend implements embed.Backend for unit tests.
type fakeBackend struct {
	openErr error
	handle  *fakePipelineHandle
}

func (f *fakeBackend) OpenPipeline(
	_ context.Context,
	_ string,
) (embed.PipelineHandle, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}

	return f.handle, nil
}

// fakePipelineHandle implements embed.PipelineHandle.
type fakePipelineHandle struct {
	runErr      error
	runResult   embed.FeatureOutput
	destroyErr  error
	destroyHits int
}

func (h *fakePipelineHandle) Destroy() error {
	h.destroyHits++

	return h.destroyErr
}

func (h *fakePipelineHandle) RunPipeline(
	_ context.Context, _ []string,
) (embed.FeatureOutput, error) {
	if h.runErr != nil {
		return embed.FeatureOutput{}, h.runErr
	}

	return h.runResult, nil
}
