package cli

import (
	"context"
	"io"

	"engram/internal/memory"
	"engram/internal/recall"
)

// Exported variables.
var (
	ExportApplyDataDirDefault     = applyDataDirDefault
	ExportApplyProjectSlugDefault = applyProjectSlugDefault
	ExportBuildMemoryIndex        = buildMemoryIndex
	ExportDescribeNewMemory       = describeNewMemory
	ExportParseConflictResponse   = parseConflictResponse
	ExportRenderConflictContent   = renderConflictContent
	ExportRenderFactContent       = renderFactContent
	ExportRenderMemoryContent     = renderMemoryContent
	ExportValidateSource          = validateSource
)

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

// ExportParseConflictLine wraps parseConflictLine for testing.
func ExportParseConflictLine(line, dataDir string, stdout io.Writer) {
	parseConflictLine(line, dataDir, stdout)
}

// ExportWriteMemoryForTest wraps writeMemory for testing with a pre-built record.
func ExportWriteMemoryForTest(
	record *memory.MemoryRecord,
	situation, dataDir string,
	noDupCheck bool,
	stdout io.Writer,
	cmdName string,
) error {
	dd := dataDir
	return writeMemory(context.Background(), record, situation, &dd, noDupCheck, stdout, cmdName)
}
