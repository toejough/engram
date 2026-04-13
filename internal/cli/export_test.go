package cli

import (
	"context"
	"io"

	"engram/internal/apiclient"
	"engram/internal/recall"
	"engram/internal/surface"
)

// Exported variables.
var (
	ExportApplyDataDirDefault     = applyDataDirDefault
	ExportApplyProjectSlugDefault = applyProjectSlugDefault
	ExportBuildRecallSurfacer     = buildRecallSurfacer
	ExportBuildRunClaude          = buildRunClaude
	ExportDeriveChatFilePath      = deriveChatFilePath
	ExportNewFilePoster           = newFilePoster
	ExportNewFileWatcher          = newFileWatcher
	ExportOsAppendFile            = osAppendFile
	ExportOsLineCount             = osLineCount
	ExportOsLockFile              = osLockFile
	ExportRecordSurfacing         = recordSurfacing
	ExportRenderFactContent       = renderFactContent
	ExportRenderMemoryContent     = renderMemoryContent
	ExportRenderMemoryMeta        = renderMemoryMeta
	ExportRunAPIDispatch          = runAPIDispatch
)

// ExportDoIntent exposes doIntent for testing as a function value.
// The explicit return type ensures the generated StartExportDoIntent wrapper has the correct signature.
func ExportDoIntent(
	ctx context.Context,
	api apiclient.API,
	from, toAgent, situation, plannedAction string,
	stdout io.Writer,
) error {
	return doIntent(ctx, api, from, toAgent, situation, plannedAction, stdout)
}

// ExportDoLearn exposes doLearn for testing as a function value.
// The explicit return type ensures the generated StartExportDoLearn wrapper has the correct signature.
func ExportDoLearn(
	ctx context.Context,
	api apiclient.API,
	from, learnType, situation, behavior, impact, action, subject, predicate, object string,
	stdout io.Writer,
) error {
	return doLearn(
		ctx, api, from, learnType, situation, behavior, impact,
		action, subject, predicate, object, stdout,
	)
}

// ExportDoPost exposes doPost for testing as a function value.
// The explicit return type ensures the generated StartExportDoPost wrapper has the correct signature.
func ExportDoPost(
	ctx context.Context,
	api apiclient.API,
	from, to, text string,
	stdout io.Writer,
) error {
	return doPost(ctx, api, from, to, text, stdout)
}

// ExportDoStatus exposes doStatus for testing as a function value.
// The explicit return type ensures the generated StartExportDoStatus wrapper has the correct signature.
func ExportDoStatus(
	ctx context.Context,
	api apiclient.API,
	stdout io.Writer,
) error {
	return doStatus(ctx, api, stdout)
}

// ExportDoSubscribe exposes doSubscribe for testing as a function value.
// The explicit return type ensures the generated StartExportDoSubscribe wrapper has the correct signature.
func ExportDoSubscribe(
	ctx context.Context,
	api apiclient.API,
	agent string,
	afterCursor int,
	stdout io.Writer,
) error {
	return doSubscribe(ctx, api, agent, afterCursor, stdout)
}

// ExportNewHaikuCallerAdapter creates a haikuCallerAdapter for testing.
func ExportNewHaikuCallerAdapter(
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
) recall.HaikuCaller {
	return &haikuCallerAdapter{caller: caller}
}

// ExportNewOsDirLister creates an osDirLister for testing.
func ExportNewOsDirLister() recall.DirLister {
	return &osDirLister{}
}

// ExportNewOsFileReader creates an osFileReader for testing.
func ExportNewOsFileReader() interface {
	Read(path string) ([]byte, error)
} {
	return &osFileReader{}
}

// ExportNewSurfaceRunnerAdapter creates a surfaceRunnerAdapter for testing.
func ExportNewSurfaceRunnerAdapter(surfacer *surface.Surfacer) SurfaceRunner {
	return &surfaceRunnerAdapter{surfacer: surfacer}
}
