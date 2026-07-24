package embed

import (
	"context"
	stdembed "embed"
	"errors"
	"fmt"
	"sync"
)

// Exported constants.
const (
	// BundledModelDir is the directory inside BundledModelFS holding the
	// bundled model files.
	BundledModelDir = "assets/model"
	BundledModelID  = "minilm-l6-v2@384"
)

// Exported variables.
var (
	ErrBundledModelUnavailable = errors.New(
		"bundled model missing or empty — rebuild the binary with the model in place, " +
			"or set ENGRAM_MODEL_PATH to a directory containing model.onnx",
	)
	ErrHugotEmbedEmpty = errors.New("hugot embed: empty result")
	ErrHugotProbeEmpty = errors.New("hugot probe returned no embedding")
)

// Backend opens an embedding pipeline for an on-disk model directory. The
// production implementation is composed internally by NewRuntimeBackend
// (runtime.go) over the raw Runtime that cmd wires into cli.Primitives —
// no hugot import anywhere in internal (#700); tests inject fakes to
// exercise every constructor branch.
type Backend interface {
	OpenPipeline(ctx context.Context, modelDir string) (PipelineHandle, error)
}

// FeatureOutput is the embedding shape returned by
// PipelineHandle.RunPipeline, mirrored here so implementations don't leak
// their runtime's own output types.
type FeatureOutput struct {
	Embeddings [][]float32
}

// HugotEmbedder wraps an embedding pipeline. Safe for concurrent use — the
// production pipeline runs the model under its own lock.
type HugotEmbedder struct {
	pipeline interface {
		RunPipeline(ctx context.Context, inputs []string) (out FeatureOutput, err error)
	}
	modelID string
	dims    int

	// Capture the close logic at construction time so the destroy chain
	// stays encapsulated even if the backend's session type changes.
	close func() error
}

// NewBundledHugotEmbedder is the production constructor: bundled assets FS,
// fixed model directory, fixed model ID, and caller-supplied backend, cache
// FS, and cache dir. The cache dir is the XDG-keyed path where the model is
// extracted once and reused across all subsequent invocations.
func NewBundledHugotEmbedder(
	ctx context.Context, backend Backend, cfs CacheFS, cacheDir string,
) (*HugotEmbedder, error) {
	return NewHugotEmbedderFromFS(
		ctx, backend, cfs, bundledModel, BundledModelDir, BundledModelID, cacheDir)
}

// NewHugotEmbedderFromDir constructs an embedder reading the model from a
// directory on disk via the injected backend, probing once to learn the
// embedding dimensionality. Every error branch (pipeline open, probe run,
// empty probe) is unit-testable with a fake Backend.
func NewHugotEmbedderFromDir(
	ctx context.Context, backend Backend, modelDir, modelID string,
) (*HugotEmbedder, error) {
	handle, openErr := backend.OpenPipeline(ctx, modelDir)
	if openErr != nil {
		return nil, openErr
	}

	probe, probeErr := handle.RunPipeline(ctx, []string{"probe"})
	if probeErr != nil {
		_ = handle.Destroy()

		return nil, probeErr
	}

	if len(probe.Embeddings) == 0 || len(probe.Embeddings[0]) == 0 {
		_ = handle.Destroy()

		return nil, ErrHugotProbeEmpty
	}

	runner := &pipelineRunner{run: handle.RunPipeline}

	return &HugotEmbedder{
		pipeline: runner,
		modelID:  modelID,
		dims:     len(probe.Embeddings[0]),
		close:    handle.Destroy,
	}, nil
}

// NewHugotEmbedderFromFS constructs an embedder from any stdembed.FS rooted
// at modelDir. cacheDir is the stable directory where the model is
// extracted once via cfs and reused across invocations (XDG-keyed). Tests
// pass an empty FS to verify UAT 10's clear-error path.
func NewHugotEmbedderFromFS(
	ctx context.Context, backend Backend, cfs CacheFS,
	modelFS stdembed.FS, modelDir, modelID, cacheDir string,
) (*HugotEmbedder, error) {
	dir, extractErr := extractToCache(cfs, modelFS, modelDir, cacheDir)
	if extractErr != nil {
		return nil, extractErr
	}

	return NewHugotEmbedderFromDir(ctx, backend, dir, modelID)
}

// Close releases the underlying session. Safe to call multiple times. The
// model cache dir is NOT removed — it is a shared, persistent cache reused
// across all engram invocations.
func (h *HugotEmbedder) Close() error {
	if h.close != nil {
		err := h.close()
		h.close = nil

		return err
	}

	return nil
}

// Dims reports the embedding dimensionality.
func (h *HugotEmbedder) Dims() int { return h.dims }

// Embed runs the pipeline on text (truncated to fit the model's context
// window) and returns the resulting vector.
//
// The char guard assumes prose density; code-dense text can still exceed the
// model's 512-token positional limit within the char limit (observed: 1500
// chars of transcript tokenizing to 538 tokens, panicking graph compilation).
// On failure the input is halved and retried until it succeeds or bottoms out,
// so a single dense chunk degrades to a shorter prefix instead of failing the
// whole ingest.
func (h *HugotEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if len(text) > hugotInputCharLimit {
		text = text[:hugotInputCharLimit]
	}

	for {
		out, err := h.pipeline.RunPipeline(ctx, []string{text})
		if err != nil {
			if len(text) >= hugotRetryFloorChars {
				text = text[:len(text)/2]

				continue
			}

			return nil, err
		}

		if len(out.Embeddings) == 0 {
			return nil, ErrHugotEmbedEmpty
		}

		return out.Embeddings[0], nil
	}
}

// ModelID reports the configured model identifier.
func (h *HugotEmbedder) ModelID() string { return h.modelID }

// LazyEmbedder defers construction of an embedder until first use so
// commands that don't need it (help, update, transcript) don't pay the
// model-unpack cost or die if model loading fails. The construction is
// factory-injected so tests can drive both the success and failure
// init paths without a real backend.
type LazyEmbedder struct {
	once    sync.Once
	factory func() (*HugotEmbedder, error)
	emb     *HugotEmbedder
	initErr error
}

// NewLazyEmbedder returns a wrapper around NewBundledHugotEmbedder that
// extracts the bundled model to cacheDir at most once (on first Embed or
// Dims call) using the injected backend and cache FS. ModelID returns the
// bundled model ID without triggering initialization. cacheDir should be
// the XDG-keyed stable cache path for the model, e.g.
// $XDG_CACHE_HOME/engram/models/<model_id>/.
func NewLazyEmbedder(backend Backend, cfs CacheFS, cacheDir string) *LazyEmbedder {
	return &LazyEmbedder{
		// Background context: the lazy init runs at most once per process;
		// a request-scoped context could cancel construction partway through
		// model extraction and leave a partial temp dir.
		factory: func() (*HugotEmbedder, error) {
			return NewBundledHugotEmbedder(context.Background(), backend, cfs, cacheDir)
		},
	}
}

// Dims lazily constructs the embedder, then delegates. Returns 0 when
// construction failed; callers should detect via an Embed error.
func (l *LazyEmbedder) Dims() int {
	l.init()

	if l.initErr != nil {
		return 0
	}

	return l.emb.Dims()
}

// Embed lazily constructs the embedder, then delegates.
func (l *LazyEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	l.init()

	if l.initErr != nil {
		return nil, fmt.Errorf("embedder unavailable: %w", l.initErr)
	}

	return l.emb.Embed(ctx, text)
}

// ModelID lazily constructs the embedder, then delegates. Returns the
// bundled model id when construction has not been attempted yet so
// status-style callers can avoid paying the unpack cost.
func (l *LazyEmbedder) ModelID() string {
	if l.emb == nil && l.initErr == nil {
		return BundledModelID
	}

	if l.initErr != nil {
		return BundledModelID
	}

	return l.emb.ModelID()
}

// init runs at most once per LazyEmbedder via sync.Once. The factory
// is provided at construction time so tests can drive both success and
// failure init paths without a real backend.
func (l *LazyEmbedder) init() {
	l.once.Do(func() {
		l.emb, l.initErr = l.factory()
	})
}

// PipelineHandle is the runtime surface of an opened pipeline plus its
// owning session; Destroy releases both together.
type PipelineHandle interface {
	RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error)
	Destroy() error
}

// BundledModelFS returns the go:embed-ed bundled model assets, rooted at
// BundledModelDir. Exposed so cmd/engram (and its integration tests) can
// hand the bundled assets to the injectable constructors.
func BundledModelFS() stdembed.FS { return bundledModel }

// unexported constants.
const (
	hugotInputCharLimit = 1500
	// hugotRetryFloorChars stops the over-length halving retry: below this
	// the failure is not a token-budget problem and must surface.
	hugotRetryFloorChars = 100
)

//go:embed assets/model/*
var bundledModel stdembed.FS

// pipelineRunner adapts a PipelineHandle's run function to the small
// interface HugotEmbedder depends on; isolating the dependency makes
// backend version bumps a surgical edit instead of a sweep.
type pipelineRunner struct {
	run func(ctx context.Context, inputs []string) (FeatureOutput, error)
}

func (p *pipelineRunner) RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error) {
	return p.run(ctx, inputs)
}
