package embed

import (
	"context"
	stdembed "embed"
)

// Exported variables.
var (
	ExportNotExist = notExist
)

// ExportFeatureOutput is the test alias for the unexported
// featureOutput shape used by hugotPipelineHandle.RunPipeline.
type ExportFeatureOutput = featureOutput

// ExportHugotBackend exposes the unexported hugotBackend interface so
// tests can inject fakes into BuildEmbedderForTest.
type ExportHugotBackend = hugotBackend

// ExportHugotPipelineHandle exposes the unexported pipeline handle
// interface so fakes can implement it.
type ExportHugotPipelineHandle = hugotPipelineHandle

// ExportHugotRawPipeline exposes the unexported hugotRawPipeline interface
// so tests can inject fakes into BuildProductionHugotPipelineForTest.
type ExportHugotRawPipeline = hugotRawPipeline

// ExportHugotSessionDestroyer exposes the unexported hugotSessionDestroyer
// interface so tests can inject fakes into BuildProductionHugotBackendForTest.
type ExportHugotSessionDestroyer = hugotSessionDestroyer

// ExportTempFS aliases the unexported tempFS so tests can implement it.
type ExportTempFS = tempFS

// BuildEmbedderForTest drives the unexported buildEmbedder so each of
// its branches can be unit-tested via a fake hugotBackend.
func BuildEmbedderForTest(
	ctx context.Context, backend ExportHugotBackend, modelDir, modelID string,
) (*HugotEmbedder, error) {
	return buildEmbedder(ctx, backend, modelDir, modelID)
}

// BuildProductionHugotBackendForTest constructs a productionHugotBackend with
// injected openSession and openPipeline so each branch of OpenPipeline can be
// exercised without a real Hugot runtime.
func BuildProductionHugotBackendForTest(
	openSession func(context.Context) (hugotSessionDestroyer, error),
	openPipeline func(hugotSessionDestroyer, string) (hugotRawPipeline, error),
) hugotBackend {
	return productionHugotBackend{
		openSession:  openSession,
		openPipeline: openPipeline,
	}
}

// BuildProductionHugotPipelineForTest constructs a productionHugotPipeline with
// injected session and pipeline so the Destroy and RunPipeline error branches can
// be exercised without a real Hugot session.
func BuildProductionHugotPipelineForTest(
	session hugotSessionDestroyer,
	pipeline hugotRawPipeline,
) hugotPipelineHandle {
	return &productionHugotPipeline{session: session, pipeline: pipeline}
}

// ExportProductionTempFS returns the production temp-FS adapter so its
// individual methods can be exercised under coverage without going
// through the unpack wrapper.
func ExportProductionTempFS() ExportTempFS { return productionTempFS{} }

// ExportUnpackModelToTemp exposes the unexported unpack helper with an
// injectable tempFS so tests can exercise mkdir/write/remove error
// branches without touching the real disk.
func ExportUnpackModelToTemp(tfs ExportTempFS, modelFS stdembed.FS, modelDir string) (string, error) {
	return unpackModelToTemp(tfs, modelFS, modelDir)
}

// ExportUnpackModelToTempProduction exposes the production wiring so an
// integration test exercises productionTempFS's MkdirTemp / WriteFile /
// RemoveAll on a real disk.
func ExportUnpackModelToTempProduction(modelFS stdembed.FS, modelDir string) (string, error) {
	return unpackModelToTemp(productionTempFS{}, modelFS, modelDir)
}

// NewHugotEmbedderWithPipelineForTest constructs a HugotEmbedder around
// a caller-supplied pipeline implementation. Tests use this to exercise
// the Embed/Close error branches without depending on a real Hugot
// session.
func NewHugotEmbedderWithPipelineForTest(
	modelID string, dims int,
	runFn func(text string) ([][]float32, error),
	closeFn func() error,
) *HugotEmbedder {
	runner := &pipelineRunner{
		run: func(_ context.Context, inputs []string) (featureOutput, error) {
			out, err := runFn(inputs[0])
			if err != nil {
				return featureOutput{}, err
			}

			return featureOutput{Embeddings: out}, nil
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
// failure paths without running real Hugot.
func NewLazyEmbedderWithFactoryForTest(factory func() (*HugotEmbedder, error)) *LazyEmbedder {
	return &LazyEmbedder{factory: factory}
}
