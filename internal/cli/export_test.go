package cli

import (
	"context"
	"os"

	"engram/internal/chat"
	"engram/internal/recall"
	"engram/internal/surface"
)

// Exported variables.
var (
	ExportApplyDataDirDefault     = applyDataDirDefault
	ExportApplyProjectSlugDefault = applyProjectSlugDefault
	ExportBuildRecallSurfacer     = buildRecallSurfacer
	ExportDeriveChatFilePath      = deriveChatFilePath
	ExportLoadChatMessages        = func(path string) ([]chat.Message, error) {
		return loadChatMessages(path, os.ReadFile)
	}
	ExportOsAppendFile             = osAppendFile
	ExportOutputAckResult          = outputAckResult
	ExportReadModifyWriteStateFile = readModifyWriteStateFile
	ExportRecordSurfacing          = recordSurfacing
	ExportRenderFactContent        = renderFactContent
	ExportRenderMemoryContent      = renderMemoryContent
	ExportRenderMemoryMeta         = renderMemoryMeta
	ExportResolveChatFile          = resolveChatFile
	ExportResolveStateFile         = resolveStateFile
)

// --- Factory functions for structs with unexported fields ---

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
