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
	ExportBuildAndInstall                = buildAndInstall
	ExportComputeMainProjectDir          = computeMainProjectDir
	ExportDescribeNewMemory              = describeNewMemory
	ExportFleetingPath                   = fleetingPath
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
	ExportRequireLLMCmd                  = requireLLMCmd
	ExportResolveContent                 = resolveContent
	ExportResolveLLMCmd                  = resolveLLMCmd
	ExportResolveVault                   = resolveVault
	ExportRunBuildSelf                   = runBuildSelf
	ExportValidateSlug                   = validateSlug
	ExportValidateSource                 = validateSource
	ExportWriteMemory                    = writeMemory
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

	return runRecallSessions(ctx, stdout, &slug, summarizer, memLister, dataDir, query, getwd,
		userHomeDir, "")
}

// ExportRunRecallSessionsWithOpts wraps runRecallSessions for testing with transcript-dir.
func ExportRunRecallSessionsWithOpts(
	ctx context.Context,
	stdout io.Writer,
	projectSlug string,
	summarizer recall.SummarizerI,
	memLister recall.MemoryLister,
	dataDir, query string,
	getwd func() (string, error),
	userHomeDir func() (string, error),
	transcriptDir string,
) error {
	slug := projectSlug

	return runRecallSessions(ctx, stdout, &slug, summarizer, memLister, dataDir, query, getwd,
		userHomeDir, transcriptDir)
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
	_, _, err := writeMemory(
		ctx, record, situation, &dd, noDupCheck, stdout, cmdName, nil, memory.NewLister(),
	)

	return err
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
	_, _, err := writeMemory(ctx, record, situation, &dd, noDupCheck, stdout, cmdName, caller, lister)

	return err
}
