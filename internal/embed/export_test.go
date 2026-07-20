package embed

import (
	"context"
	stdembed "embed"
)

// Exported variables.
var (
	ExportNotExist = notExist
)

// ExportExtractToCache exposes the unexported extractToCache helper so
// tests can exercise the sentinel / race / error branches with a fake
// CacheFS without touching the real disk.
func ExportExtractToCache(
	cfs CacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	return extractToCache(cfs, modelFS, modelDir, cacheDir)
}

// NewHugotEmbedderWithPipelineForTest constructs a HugotEmbedder around
// a caller-supplied pipeline implementation. Tests use this to exercise
// the Embed/Close error branches without depending on a real backend.
func NewHugotEmbedderWithPipelineForTest(
	modelID string, dims int,
	runFn func(text string) ([][]float32, error),
	closeFn func() error,
) *HugotEmbedder {
	runner := &pipelineRunner{
		run: func(_ context.Context, inputs []string) (FeatureOutput, error) {
			out, err := runFn(inputs[0])
			if err != nil {
				return FeatureOutput{}, err
			}

			return FeatureOutput{Embeddings: out}, nil
		},
	}

	return &HugotEmbedder{
		pipeline: runner,
		modelID:  modelID,
		dims:     dims,
		close:    closeFn,
	}
}

// NewLazyEmbedderWithFactoryForTest constructs a LazyEmbedder with a
// caller-supplied factory so tests can drive both init success and
// failure paths without a real backend.
func NewLazyEmbedderWithFactoryForTest(factory func() (*HugotEmbedder, error)) *LazyEmbedder {
	return &LazyEmbedder{factory: factory}
}

// SetCacheDirForTest is a no-op test helper for the Close-does-not-delete
// test. HugotEmbedder no longer holds a tmpDir field — Close only closes the
// backend session and never removes any directory. The function is kept for
// test readability; the test creates its own dir and verifies it survives.
func SetCacheDirForTest(_ *HugotEmbedder, _ string) {}
