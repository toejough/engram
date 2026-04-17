package cli

import (
	"context"
	"io"

	"engram/internal/memory"
	"engram/internal/recall"
)

// Exported variables.
var (
	ExportApplyDataDirDefault            = applyDataDirDefault
	ExportApplyProjectSlugDefault        = applyProjectSlugDefault
	ExportBuildMemoryIndex               = buildMemoryIndex
	ExportComputeMainProjectDir          = computeMainProjectDir
	ExportDescribeNewMemory              = describeNewMemory
	ExportOsDirListMd                    = osDirListMd
	ExportOsMatchAny                     = osMatchAny
	ExportOsStatExists                   = osStatExists
	ExportOsWalkMd                       = osWalkMd
	ExportOsWalkSkills                   = osWalkSkills
	ExportParseConflictResponse          = parseConflictResponse
	ExportReadAutoMemoryDirectorySetting = readAutoMemoryDirectorySetting
	ExportRenderConflictContent          = renderConflictContent
	ExportRenderFactContent              = renderFactContent
	ExportRenderMemoryContent            = renderMemoryContent
	ExportValidateSource                 = validateSource
)

// ExportCallHaikuForConflicts wraps callHaikuForConflicts for testing.
func ExportCallHaikuForConflicts(
	ctx context.Context,
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
	index, description string,
) (string, error) {
	return callHaikuForConflicts(ctx, caller, index, description)
}

// ExportCheckForConflicts wraps checkForConflicts for testing.
func ExportCheckForConflicts(
	ctx context.Context,
	record *memory.MemoryRecord,
	dataDir string,
	stdout io.Writer,
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
	lister memoryLister,
) (bool, error) {
	return checkForConflicts(ctx, record, dataDir, stdout, caller, lister)
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

// ExportNewSummarizer wraps newSummarizer for testing.
func ExportNewSummarizer(token string) recall.SummarizerI {
	return newSummarizer(token)
}

// ExportParseConflictLine wraps parseConflictLine for testing.
func ExportParseConflictLine(line, dataDir string, stdout io.Writer) {
	parseConflictLine(line, dataDir, stdout)
}

// ExportRunRecallSessions wraps runRecallSessions for testing.
func ExportRunRecallSessions(
	ctx context.Context,
	stdout io.Writer,
	projectSlug string,
	summarizer recall.SummarizerI,
	memLister recall.MemoryLister,
	dataDir, query string,
	getwd func() (string, error),
	userHomeDir func() (string, error),
) error {
	slug := projectSlug
	return runRecallSessions(ctx, stdout, &slug, summarizer, memLister, dataDir, query, getwd, userHomeDir)
}

// ExportWriteMemoryForTest wraps writeMemory for testing with a pre-built record.
func ExportWriteMemoryForTest(
	ctx context.Context,
	record *memory.MemoryRecord,
	situation, dataDir string,
	noDupCheck bool,
	stdout io.Writer,
	cmdName string,
) error {
	dd := dataDir
	return writeMemory(ctx, record, situation, &dd, noDupCheck, stdout, cmdName, nil, memory.NewLister())
}

// ExportWriteMemoryWithDeps wraps writeMemory for testing with injected deps.
func ExportWriteMemoryWithDeps(
	ctx context.Context,
	record *memory.MemoryRecord,
	situation, dataDir string,
	noDupCheck bool,
	stdout io.Writer,
	cmdName string,
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
	lister memoryLister,
) error {
	dd := dataDir
	return writeMemory(ctx, record, situation, &dd, noDupCheck, stdout, cmdName, caller, lister)
}
