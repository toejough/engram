package embed

import (
	"context"
	stdembed "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/knights-analytics/hugot"
)

// Exported constants.
const (
	BundledModelID = "minilm-l6-v2@384"
)

// Sentinel errors for the Hugot embedder.
var (
	ErrHugotProbeEmpty         = errors.New("hugot probe returned no embedding")
	ErrHugotEmbedEmpty         = errors.New("hugot embed: empty result")
	ErrBundledModelUnavailable = errors.New(
		"bundled model missing or empty — rebuild the binary with the model in place, " +
			"or set ENGRAM_MODEL_PATH to a directory containing model.onnx",
	)
)

// HugotEmbedder wraps a Hugot pipeline + the temp-dir lifecycle for
// unpacked model files. Safe for concurrent use — Hugot's pipeline runs
// the model under its own lock.
type HugotEmbedder struct {
	pipeline interface {
		RunPipeline(ctx context.Context, inputs []string) (out featureOutput, err error)
	}
	tmpDir  string
	modelID string
	dims    int

	// Capture the close logic at construction time so the destroy +
	// temp-dir cleanup chain stays encapsulated even if Hugot's Session
	// type changes across versions.
	close func() error
}

// NewBundledHugotEmbedder is the production constructor: bundled assets
// FS, fixed model directory, fixed model ID.
func NewBundledHugotEmbedder(ctx context.Context) (*HugotEmbedder, error) {
	return NewHugotEmbedderFromFS(ctx, bundledModel, "assets/model", BundledModelID)
}

// NewHugotEmbedderFromDir constructs an embedder reading the model from
// a directory on disk. Used by the parity gate and the FS-based test
// path; production uses NewBundledHugotEmbedder.
func NewHugotEmbedderFromDir(
	ctx context.Context,
	modelDir, modelID string,
) (*HugotEmbedder, error) {
	session, err := hugot.NewGoSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("hugot session: %w", err)
	}

	config := hugot.FeatureExtractionConfig{
		ModelPath:    modelDir,
		Name:         "engram-embed",
		OnnxFilename: "model.onnx",
	}

	pipeline, pipeErr := hugot.NewPipeline(session, config)
	if pipeErr != nil {
		_ = session.Destroy()

		return nil, fmt.Errorf("hugot pipeline: %w", pipeErr)
	}

	probe, probeErr := pipeline.RunPipeline(ctx, []string{"probe"})
	if probeErr != nil {
		_ = session.Destroy()

		return nil, fmt.Errorf("hugot probe: %w", probeErr)
	}

	if len(probe.Embeddings) == 0 || len(probe.Embeddings[0]) == 0 {
		_ = session.Destroy()

		return nil, ErrHugotProbeEmpty
	}

	runner := &pipelineRunner{
		run: func(ctx context.Context, inputs []string) (featureOutput, error) {
			out, runErr := pipeline.RunPipeline(ctx, inputs)
			if runErr != nil {
				return featureOutput{}, fmt.Errorf("hugot run: %w", runErr)
			}

			return featureOutput{Embeddings: out.Embeddings}, nil
		},
	}

	return &HugotEmbedder{
		pipeline: runner,
		modelID:  modelID,
		dims:     len(probe.Embeddings[0]),
		close: func() error {
			destroyErr := session.Destroy()
			if destroyErr != nil {
				return fmt.Errorf("hugot session destroy: %w", destroyErr)
			}

			return nil
		},
	}, nil
}

// NewHugotEmbedderFromFS constructs an embedder from any stdembed.FS
// rooted at modelDir. Tests pass an empty FS to verify UAT 10's
// clear-error path; production wraps the bundled assets.
func NewHugotEmbedderFromFS(
	ctx context.Context, modelFS stdembed.FS, modelDir, modelID string,
) (*HugotEmbedder, error) {
	tmp, unpackErr := unpackModelToTemp(modelFS, modelDir)
	if unpackErr != nil {
		return nil, unpackErr
	}

	embedder, embErr := NewHugotEmbedderFromDir(ctx, tmp, modelID)
	if embErr != nil {
		_ = os.RemoveAll(tmp)

		return nil, embErr
	}

	embedder.tmpDir = tmp

	return embedder, nil
}

// Close releases the Hugot session and removes the unpacked model temp
// directory (if any). Safe to call multiple times.
func (h *HugotEmbedder) Close() error {
	var firstErr error

	if h.close != nil {
		firstErr = h.close()
		h.close = nil
	}

	if h.tmpDir != "" {
		_ = os.RemoveAll(h.tmpDir)
		h.tmpDir = ""
	}

	return firstErr
}

// Dims reports the embedding dimensionality.
func (h *HugotEmbedder) Dims() int { return h.dims }

// Embed runs the pipeline on text (truncated to fit the model's context
// window) and returns the resulting vector.
func (h *HugotEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if len(text) > hugotInputCharLimit {
		text = text[:hugotInputCharLimit]
	}

	out, err := h.pipeline.RunPipeline(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(out.Embeddings) == 0 {
		return nil, ErrHugotEmbedEmpty
	}

	return out.Embeddings[0], nil
}

// ModelID reports the configured model identifier.
func (h *HugotEmbedder) ModelID() string { return h.modelID }

// LazyEmbedder defers construction of a bundled embedder until first use
// so commands that don't need it (help, update, transcript) don't pay
// the model-unpack cost or die if model loading fails.
type LazyEmbedder struct {
	once    sync.Once
	emb     *HugotEmbedder
	initErr error
}

// NewLazyEmbedder returns a wrapper around NewBundledHugotEmbedder. Each
// LazyEmbedder unpacks the model at most once, on first call to Embed /
// ModelID / Dims.
func NewLazyEmbedder() *LazyEmbedder { return &LazyEmbedder{} }

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

// init runs at most once per LazyEmbedder. The closure intentionally
// uses context.Background() instead of plumbing a request-scoped ctx
// through sync.Once: the lazy init unpacks the bundled model to a temp
// directory, and a canceled request-scoped context could strand temp
// files mid-unpack. contextcheck flags this — disable for the closure.
//

func (l *LazyEmbedder) init() {
	l.once.Do(func() {
		ctx := context.Background()
		l.emb, l.initErr = NewBundledHugotEmbedder(ctx)
	})
}

// unexported constants.
const (
	hugotInputCharLimit = 1500
)

//go:embed assets/model/*
var bundledModel stdembed.FS

// featureOutput mirrors the shape we care about from
// hugot/pipelines.FeatureExtractionOutput so the test surface doesn't
// have to import Hugot directly.
type featureOutput struct {
	Embeddings [][]float32
}

// pipelineRunner adapts hugot's concrete pipeline to the small interface
// we depend on; isolating the dependency makes Hugot version bumps a
// surgical edit instead of a sweep.
type pipelineRunner struct {
	run func(ctx context.Context, inputs []string) (featureOutput, error)
}

func (p *pipelineRunner) RunPipeline(ctx context.Context, inputs []string) (featureOutput, error) {
	return p.run(ctx, inputs)
}

// unpackModelToTemp copies every file from modelFS rooted at modelDir
// into a fresh temp directory and returns its path. Extracted so UAT 10
// (missing model file) can be exercised by passing an empty embed.FS.
func unpackModelToTemp(modelFS stdembed.FS, modelDir string) (string, error) {
	entries, dirErr := modelFS.ReadDir(modelDir)
	if dirErr != nil || len(entries) == 0 {
		return "", fmt.Errorf("%w: dir %s (underlying: %w)",
			ErrBundledModelUnavailable, modelDir, dirErr,
		)
	}

	tmp, mkErr := os.MkdirTemp("", "engram-embed-model-*")
	if mkErr != nil {
		return "", fmt.Errorf("temp dir: %w", mkErr)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, readErr := modelFS.ReadFile(filepath.Join(modelDir, entry.Name()))
		if readErr != nil {
			_ = os.RemoveAll(tmp)

			return "", fmt.Errorf("read embedded %s: %w", entry.Name(), readErr)
		}

		const perm = 0o600

		writeErr := os.WriteFile(filepath.Join(tmp, entry.Name()), data, perm)
		if writeErr != nil {
			_ = os.RemoveAll(tmp)

			return "", fmt.Errorf("unpack %s: %w", entry.Name(), writeErr)
		}
	}

	return tmp, nil
}
