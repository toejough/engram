package embed_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/knights-analytics/hugot/pipelines"
	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// TestNewHugotEmbedderFromFS_InvalidModelFails exercises the embErr != nil branch
// of NewHugotEmbedderFromFS: unpack succeeds (files exist) but Hugot rejects the
// directory because it contains no valid model.onnx.
func TestNewHugotEmbedderFromFS_InvalidModelFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// nonEmptyTestFS has real files but they are not Hugot model files, so
	// unpackModelToTemp will succeed and NewHugotEmbedderFromDir will fail.
	cacheDir := filepath.Join(t.TempDir(), "model-cache")
	_, err := embed.NewHugotEmbedderFromFS(t.Context(), nonEmptyTestFS, "testdata", "fake@1", cacheDir)
	g.Expect(err).To(HaveOccurred())
}

// TestOpenPipeline_PipelineFailDestroysSession exercises the second error
// branch of productionHugotBackend.OpenPipeline: openPipeline returns an error
// and the session's Destroy is called.
func TestOpenPipeline_PipelineFailDestroysSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pipeErr := errors.New("pipeline blocked")
	session := &fakeSession{}

	backend := embed.BuildProductionHugotBackendForTest(
		func(context.Context) (embed.ExportHugotSessionDestroyer, error) {
			return session, nil
		},
		func(embed.ExportHugotSessionDestroyer, string) (embed.ExportHugotRawPipeline, error) {
			return nil, pipeErr
		},
	)

	_, err := embed.BuildEmbedderForTest(t.Context(), backend, "/tmp/x", "fake@1")
	g.Expect(err).To(MatchError(ContainSubstring("hugot pipeline")))
	g.Expect(err).To(MatchError(ContainSubstring("pipeline blocked")))
	g.Expect(session.destroyCalls).
		To(Equal(1), "session.Destroy must be called on pipeline failure")
}

// TestOpenPipeline_SessionFailPropagates exercises the first error branch
// of productionHugotBackend.OpenPipeline: openSession returns an error.
func TestOpenPipeline_SessionFailPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessionErr := errors.New("session blocked")

	backend := embed.BuildProductionHugotBackendForTest(
		func(context.Context) (embed.ExportHugotSessionDestroyer, error) {
			return nil, sessionErr
		},
		func(embed.ExportHugotSessionDestroyer, string) (embed.ExportHugotRawPipeline, error) {
			return nil, errors.New("openPipeline must not be called when openSession fails")
		},
	)

	_, err := embed.BuildEmbedderForTest(t.Context(), backend, "/tmp/x", "fake@1")
	g.Expect(err).To(MatchError(ContainSubstring("hugot session")))
	g.Expect(err).To(MatchError(ContainSubstring("session blocked")))
}

// TestProductionPipeline_DestroyErrorPropagates exercises the error branch of
// productionHugotPipeline.Destroy.
func TestProductionPipeline_DestroyErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	destroyErr := errors.New("destroy blocked")
	handle := embed.BuildProductionHugotPipelineForTest(
		&fakeSession{destroyErr: destroyErr},
		nil,
	)

	err := handle.Destroy()
	g.Expect(err).To(MatchError(ContainSubstring("hugot session destroy")))
	g.Expect(err).To(MatchError(ContainSubstring("destroy blocked")))
}

// TestProductionPipeline_RunPipelineErrorPropagates exercises the error branch of
// productionHugotPipeline.RunPipeline.
func TestProductionPipeline_RunPipelineErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runErr := errors.New("run blocked")
	handle := embed.BuildProductionHugotPipelineForTest(
		&fakeSession{},
		&fakeRawPipeline{runErr: runErr},
	)

	_, err := handle.RunPipeline(t.Context(), []string{"hello"})
	g.Expect(err).To(MatchError(ContainSubstring("hugot run")))
	g.Expect(err).To(MatchError(ContainSubstring("run blocked")))
}

type fakeRawPipeline struct {
	runErr error
}

func (f *fakeRawPipeline) RunPipeline(
	_ context.Context,
	_ []string,
) (*pipelines.FeatureExtractionOutput, error) {
	if f.runErr != nil {
		return nil, f.runErr
	}

	return &pipelines.FeatureExtractionOutput{Embeddings: [][]float32{{1, 2}}}, nil
}

// unexported types.

type fakeSession struct {
	destroyCalls int
	destroyErr   error
}

func (s *fakeSession) Destroy() error {
	s.destroyCalls++

	return s.destroyErr
}
