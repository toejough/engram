package embed_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// TestRuntimeBackend_ConfigPolicyIsInternal proves the pipeline config
// policy (name, onnx filename) lives internal: the raw runtime receives
// the values without cmd declaring them.
func TestRuntimeBackend_ConfigPolicyIsInternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runtime := &fakeRuntime{
		runFn: func(context.Context, []string) ([][]float32, error) {
			return [][]float32{{1}}, nil
		},
	}

	_, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/models/m1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(runtime.gotModelPath).To(Equal("/models/m1"))
	g.Expect(runtime.gotName).To(Equal("engram-embed"))
	g.Expect(runtime.gotOnnxFile).To(Equal("model.onnx"))
}

// TestRuntimeBackend_DestroyErrorPropagates exercises the error branch of
// the composed handle's Destroy.
func TestRuntimeBackend_DestroyErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	destroyErr := errors.New("destroy blocked")
	runtime := &fakeRuntime{
		destroyErr: destroyErr,
		runFn: func(context.Context, []string) ([][]float32, error) {
			return [][]float32{{1}}, nil
		},
	}

	handle, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = handle.Destroy()
	g.Expect(err).To(MatchError(ContainSubstring("hugot session destroy")))
	g.Expect(err).To(MatchError(ContainSubstring("destroy blocked")))
}

// TestRuntimeBackend_NilRuntimeFailsLoud asserts a Deps carrier built from
// Primitives without EmbedRuntime surfaces a clear error, never a panic.
func TestRuntimeBackend_NilRuntimeFailsLoud(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := embed.NewRuntimeBackend(nil).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(embed.ErrRuntimeMissing))
}

// TestRuntimeBackend_PipelineFailDestroysSession exercises the second error
// branch: NewPipeline fails and the session's Destroy is called.
func TestRuntimeBackend_PipelineFailDestroysSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pipeErr := errors.New("pipeline blocked")
	runtime := &fakeRuntime{pipelineErr: pipeErr}

	_, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(ContainSubstring("hugot pipeline")))
	g.Expect(err).To(MatchError(ContainSubstring("pipeline blocked")))
	g.Expect(runtime.destroyCalls).
		To(Equal(1), "session.Destroy must be called on pipeline failure")
}

// TestRuntimeBackend_RunErrorPropagates exercises the run error branch of
// the composed handle.
func TestRuntimeBackend_RunErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runErr := errors.New("run blocked")
	runtime := &fakeRuntime{
		runFn: func(context.Context, []string) ([][]float32, error) {
			return nil, runErr
		},
	}

	handle, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	_, err = handle.RunPipeline(t.Context(), []string{"hello"})
	g.Expect(err).To(MatchError(ContainSubstring("hugot run")))
	g.Expect(err).To(MatchError(ContainSubstring("run blocked")))
}

// TestRuntimeBackend_RunMapsOutput drives the happy path through the
// returned handle: raw [][]float32 maps into embed.FeatureOutput.
func TestRuntimeBackend_RunMapsOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runtime := &fakeRuntime{
		runFn: func(context.Context, []string) ([][]float32, error) {
			return [][]float32{{1, 2}}, nil
		},
	}

	handle, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	out, runErr := handle.RunPipeline(t.Context(), []string{"hello"})
	g.Expect(runErr).NotTo(HaveOccurred())
	g.Expect(out.Embeddings).To(Equal([][]float32{{1, 2}}))
}

// TestRuntimeBackend_SessionFailPropagates exercises the first error branch
// of the composed OpenPipeline: NewSession returns an error.
func TestRuntimeBackend_SessionFailPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessionErr := errors.New("session blocked")
	runtime := &fakeRuntime{sessionErr: sessionErr}

	_, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(ContainSubstring("hugot session")))
	g.Expect(err).To(MatchError(ContainSubstring("session blocked")))
	g.Expect(runtime.pipelineCalled).To(BeFalse(),
		"NewPipeline must not be called when NewSession fails")
}

// fakeRuntime scripts the raw runtime seam so every branch of the
// internally-composed backend is unit-testable without hugot.
type fakeRuntime struct {
	sessionErr     error
	pipelineErr    error
	destroyErr     error
	destroyCalls   int
	pipelineCalled bool
	runFn          embed.RunPipelineFunc
	gotModelPath   string
	gotName        string
	gotOnnxFile    string
}

func (f *fakeRuntime) NewPipeline(
	_ embed.RawSession, modelPath, name, onnxFilename string,
) (embed.RunPipelineFunc, error) {
	f.pipelineCalled = true
	f.gotModelPath = modelPath
	f.gotName = name
	f.gotOnnxFile = onnxFilename

	if f.pipelineErr != nil {
		return nil, f.pipelineErr
	}

	return f.runFn, nil
}

func (f *fakeRuntime) NewSession(context.Context) (embed.RawSession, error) {
	if f.sessionErr != nil {
		return nil, f.sessionErr
	}

	return fakeRuntimeSession{runtime: f}, nil
}

type fakeRuntimeSession struct{ runtime *fakeRuntime }

func (s fakeRuntimeSession) Destroy() error {
	s.runtime.destroyCalls++

	return s.runtime.destroyErr
}
