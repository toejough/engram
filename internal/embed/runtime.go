package embed

import (
	"context"
	"errors"
	"fmt"
)

// Exported variables.
var (
	// ErrRuntimeMissing reports an embed attempt through a backend composed
	// from a nil Runtime (a Deps carrier whose Primitives had no
	// EmbedRuntime wired). Production main.go always wires one; minimal
	// test Primitives may not — this surfaces that as a clean lazy-init
	// error instead of a panic.
	ErrRuntimeMissing = errors.New(
		"embed runtime not wired: cli.Primitives.EmbedRuntime is required for embedding")
)

// RawSession is the minimal runtime-session surface the composed backend
// needs: cleanup on pipeline-creation failure and on normal close.
// *hugot.Session satisfies it structurally.
type RawSession interface {
	Destroy() error
}

// RunPipelineFunc runs an opened embedding pipeline on inputs and returns
// one vector per input. cmd's Runtime.NewPipeline returns one as a closure
// over the concrete pipeline, erasing the runtime's types without any
// re-assertion at call time (doctrine flag E-1).
type RunPipelineFunc func(ctx context.Context, inputs []string) ([][]float32, error)

// Runtime is the raw model-runtime capability surface. The production
// implementation is cmd/engram's EMPTY hugotRuntime struct whose two
// methods are single-call bodies (targ check-thin-api); ALL lifecycle
// orchestration and config policy live here, behind NewRuntimeBackend
// (#700, doctrine flag D-1).
type Runtime interface {
	// NewSession opens a runtime session.
	NewSession(ctx context.Context) (RawSession, error)
	// NewPipeline opens a feature-extraction pipeline for the model at
	// modelPath on session and returns its run function.
	NewPipeline(session RawSession, modelPath, name, onnxFilename string) (RunPipelineFunc, error)
}

// NewRuntimeBackend composes the production Backend from a raw Runtime:
// the open-session → open-pipeline → destroy-on-failure lifecycle, the
// pipeline config policy, and all error wrapping happen here, internally.
func NewRuntimeBackend(runtime Runtime) Backend {
	return runtimeBackend{runtime: runtime}
}

// unexported constants.
const (
	// pipelineName and pipelineOnnxFilename are the feature-extraction
	// pipeline config policy — kept internal so cmd passes values through
	// without declaring any constants (thin-api).
	pipelineName         = "engram-embed"
	pipelineOnnxFilename = "model.onnx"
)

// runtimeBackend implements Backend over a raw Runtime.
type runtimeBackend struct {
	runtime Runtime
}

// OpenPipeline opens a session, then a feature-extraction pipeline on it,
// destroying the session if pipeline creation fails.
func (b runtimeBackend) OpenPipeline(
	ctx context.Context, modelDir string,
) (PipelineHandle, error) {
	if b.runtime == nil {
		return nil, ErrRuntimeMissing
	}

	session, err := b.runtime.NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("hugot session: %w", err)
	}

	run, pipeErr := b.runtime.NewPipeline(session, modelDir, pipelineName, pipelineOnnxFilename)
	if pipeErr != nil {
		_ = session.Destroy()

		return nil, fmt.Errorf("hugot pipeline: %w", pipeErr)
	}

	return &runtimePipeline{session: session, run: run}, nil
}

// runtimePipeline pairs a pipeline run function with the session that owns
// it so Destroy releases both together.
type runtimePipeline struct {
	session RawSession
	run     RunPipelineFunc
}

// Destroy releases the owning session (which owns the pipeline).
func (p *runtimePipeline) Destroy() error {
	err := p.session.Destroy()
	if err != nil {
		return fmt.Errorf("hugot session destroy: %w", err)
	}

	return nil
}

// RunPipeline runs the model and maps the raw vectors into the
// runtime-neutral FeatureOutput shape.
func (p *runtimePipeline) RunPipeline(
	ctx context.Context, inputs []string,
) (FeatureOutput, error) {
	out, err := p.run(ctx, inputs)
	if err != nil {
		return FeatureOutput{}, fmt.Errorf("hugot run: %w", err)
	}

	return FeatureOutput{Embeddings: out}, nil
}
