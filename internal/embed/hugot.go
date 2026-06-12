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
	"github.com/knights-analytics/hugot/pipelines"
)

// Exported constants.
const (
	BundledModelID = "minilm-l6-v2@384"
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
// a directory on disk. Thin wrapper over buildEmbedder with the
// production Hugot backend; tests use buildEmbedder directly with a
// fake backend to exercise every error branch.
func NewHugotEmbedderFromDir(
	ctx context.Context,
	modelDir, modelID string,
) (*HugotEmbedder, error) {
	return buildEmbedder(ctx, newProductionHugotBackend(), modelDir, modelID)
}

// NewHugotEmbedderFromFS constructs an embedder from any stdembed.FS
// rooted at modelDir. Tests pass an empty FS to verify UAT 10's
// clear-error path; production wraps the bundled assets.
func NewHugotEmbedderFromFS(
	ctx context.Context, modelFS stdembed.FS, modelDir, modelID string,
) (*HugotEmbedder, error) {
	tmp, unpackErr := unpackModelToTemp(productionTempFS{}, modelFS, modelDir)
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
// init paths without running real Hugot.
type LazyEmbedder struct {
	once    sync.Once
	factory func() (*HugotEmbedder, error)
	emb     *HugotEmbedder
	initErr error
}

// NewLazyEmbedder returns a wrapper around NewBundledHugotEmbedder. Each
// LazyEmbedder unpacks the model at most once, on first call to Embed /
// ModelID / Dims.
func NewLazyEmbedder() *LazyEmbedder {
	return &LazyEmbedder{
		// Background context: the lazy init runs at most once per
		// process and unpacks the bundled model to a temp directory; a
		// request-scoped context could cancel construction partway and
		// strand temp files.
		factory: func() (*HugotEmbedder, error) {
			return NewBundledHugotEmbedder(context.Background())
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
// failure init paths without running real Hugot.
func (l *LazyEmbedder) init() {
	l.once.Do(func() {
		l.emb, l.initErr = l.factory()
	})
}

// unexported constants.
const (
	hugotInputCharLimit = 1500
	// hugotRetryFloorChars stops the over-length halving retry: below this
	// the failure is not a token-budget problem and must surface.
	hugotRetryFloorChars = 100
)

//go:embed assets/model/*
var bundledModel stdembed.FS

// featureOutput mirrors the shape we care about from
// hugot/pipelines.FeatureExtractionOutput so the test surface doesn't
// have to import Hugot directly.
type featureOutput struct {
	Embeddings [][]float32
}

// hugotBackend is the Hugot surface we depend on, extracted so tests
// can inject failures for every constructor branch.
type hugotBackend interface {
	OpenPipeline(ctx context.Context, modelDir string) (hugotPipelineHandle, error)
}

// hugotPipelineHandle is the runtime surface we use from a Hugot
// pipeline + its owning session.
type hugotPipelineHandle interface {
	RunPipeline(ctx context.Context, inputs []string) (featureOutput, error)
	Destroy() error
}

// hugotRawPipeline is the minimal Hugot pipeline surface we depend on, extracted
// so tests can inject a failing implementation without a real Hugot session.
type hugotRawPipeline interface {
	RunPipeline(ctx context.Context, inputs []string) (*pipelines.FeatureExtractionOutput, error)
}

// hugotSessionDestroyer is the minimal Hugot session surface we need —
// just cleanup on pipeline-creation failure and on normal close.
type hugotSessionDestroyer interface {
	Destroy() error
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

// productionHugotBackend wires the real hugot.NewGoSession + NewPipeline.
// The openSession and openPipeline fields are injectable so each error branch
// of OpenPipeline can be covered by a unit test without a real Hugot runtime.
type productionHugotBackend struct {
	openSession  func(ctx context.Context) (hugotSessionDestroyer, error)
	openPipeline func(session hugotSessionDestroyer, modelDir string) (hugotRawPipeline, error)
}

func (b productionHugotBackend) OpenPipeline(
	ctx context.Context, modelDir string,
) (hugotPipelineHandle, error) {
	session, err := b.openSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("hugot session: %w", err)
	}

	pipeline, pipeErr := b.openPipeline(session, modelDir)
	if pipeErr != nil {
		_ = session.Destroy()

		return nil, fmt.Errorf("hugot pipeline: %w", pipeErr)
	}

	return &productionHugotPipeline{session: session, pipeline: pipeline}, nil
}

// productionHugotPipeline pairs a Hugot pipeline with the session that
// owns it so Destroy releases both together.
type productionHugotPipeline struct {
	session  hugotSessionDestroyer
	pipeline hugotRawPipeline
}

func (p *productionHugotPipeline) Destroy() error {
	err := p.session.Destroy()
	if err != nil {
		return fmt.Errorf("hugot session destroy: %w", err)
	}

	return nil
}

func (p *productionHugotPipeline) RunPipeline(
	ctx context.Context, inputs []string,
) (featureOutput, error) {
	out, err := p.pipeline.RunPipeline(ctx, inputs)
	if err != nil {
		return featureOutput{}, fmt.Errorf("hugot run: %w", err)
	}

	return featureOutput{Embeddings: out.Embeddings}, nil
}

// productionTempFS is the canonical os.*-backed tempFS. Every method is
// a one-line passthrough — coverage is asserted via integration through
// NewBundledHugotEmbedder, not by unit tests on this adapter.
type productionTempFS struct{}

func (productionTempFS) MkdirTemp(dir, pattern string) (string, error) {
	tmp, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp: %w", err)
	}

	return tmp, nil
}

// RemoveAll deletes path. os.RemoveAll's error contract is already
// caller-friendly (returns nil on missing paths); wrapping adds noise
// without information, so the underlying error propagates as-is.
func (productionTempFS) RemoveAll(path string) error {
	return os.RemoveAll(path) //nolint:wrapcheck // thin adapter; see comment above
}

func (productionTempFS) WriteFile(name string, data []byte) error {
	const perm = 0o600

	err := os.WriteFile(name, data, perm)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// tempFS is the I/O surface unpackModelToTemp uses. Production wires
// productionTempFS (thin wrapper around os.*); tests inject fakes to
// exercise every error branch without touching the real disk.
type tempFS interface {
	MkdirTemp(dir, pattern string) (string, error)
	WriteFile(name string, data []byte) error
	RemoveAll(path string) error
}

// buildEmbedder is the orchestration shared between production and the
// parity-gate constructor. Takes a hugotBackend so every error branch
// (pipeline open, probe run, empty probe) is unit-testable with fakes.
func buildEmbedder(
	ctx context.Context, backend hugotBackend, modelDir, modelID string,
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

func newProductionHugotBackend() productionHugotBackend {
	return productionHugotBackend{
		openSession: func(ctx context.Context) (hugotSessionDestroyer, error) {
			return hugot.NewGoSession(ctx)
		},
		// openPipeline type-asserts session to *hugot.Session because openSession
		// always returns *hugot.Session in production; test code injects its own
		// openPipeline that never performs this assertion.
		openPipeline: func(session hugotSessionDestroyer, modelDir string) (hugotRawPipeline, error) {
			config := hugot.FeatureExtractionConfig{
				ModelPath:    modelDir,
				Name:         "engram-embed",
				OnnxFilename: "model.onnx",
			}

			//nolint:forcetypeassert // production invariant
			return hugot.NewPipeline(
				session.(*hugot.Session),
				config,
			)
		},
	}
}

// unpackModelToTemp copies every file from modelFS rooted at modelDir
// into a fresh temp directory and returns its path. Extracted so UAT 10
// (missing model file) can be exercised by passing an empty embed.FS,
// and so the mkdir/write/remove error branches can be unit-tested with
// a fake tempFS.
func unpackModelToTemp(tfs tempFS, modelFS stdembed.FS, modelDir string) (string, error) {
	entries, dirErr := modelFS.ReadDir(modelDir)
	if dirErr != nil || len(entries) == 0 {
		return "", fmt.Errorf("%w: dir %s (underlying: %w)",
			ErrBundledModelUnavailable, modelDir, dirErr,
		)
	}

	tmp, mkErr := tfs.MkdirTemp("", "engram-embed-model-*")
	if mkErr != nil {
		return "", fmt.Errorf("temp dir: %w", mkErr)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, readErr := modelFS.ReadFile(filepath.Join(modelDir, entry.Name()))
		if readErr != nil {
			_ = tfs.RemoveAll(tmp)

			return "", fmt.Errorf("read embedded %s: %w", entry.Name(), readErr)
		}

		writeErr := tfs.WriteFile(filepath.Join(tmp, entry.Name()), data)
		if writeErr != nil {
			_ = tfs.RemoveAll(tmp)

			return "", fmt.Errorf("unpack %s: %w", entry.Name(), writeErr)
		}
	}

	return tmp, nil
}
