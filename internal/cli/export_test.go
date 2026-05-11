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
	ExportExtractLuhmannFromFilename     = extractLuhmannFromFilename
	ExportFleetingPath                   = fleetingPath
	ExportLearnPath                      = learnPath
	ExportNewErrHandler                  = newErrHandler
	ExportNextLuhmannID                  = nextLuhmannID
	ExportOsDirListMd                    = osDirListMd
	ExportOsMatchAny                     = osMatchAny
	ExportOsStatExists                   = osStatExists
	ExportOsWalkMd                       = osWalkMd
	ExportOsWalkSkills                   = osWalkSkills
	ExportParseConflictResponse          = parseConflictResponse
	ExportReadAutoMemoryDirectorySetting = readAutoMemoryDirectorySetting
	ExportRenderConflictContent          = renderConflictContent
	ExportRenderFactBody                 = renderFactBody
	ExportRenderFactContent              = renderFactContent
	ExportRenderFactFrontmatter          = renderFactFrontmatter
	ExportRenderFeedbackBody             = renderFeedbackBody
	ExportRenderFeedbackFrontmatter      = renderFeedbackFrontmatter
	ExportRenderMOCBody                  = renderMOCBody
	ExportRenderMOCFrontmatter           = renderMOCFrontmatter
	ExportRenderMemoryContent            = renderMemoryContent
	ExportRenderRelatedSection           = renderRelatedSection
	ExportRequireLLMCmd                  = requireLLMCmd
	ExportRequireVaultDirs               = requireVaultDirs
	ExportResolveContent                 = resolveContent
	ExportResolveLLMCmd                  = resolveLLMCmd
	ExportResolveVault                   = resolveVault
	ExportRunBuildSelf                   = runBuildSelf
	ExportRunLearn                       = runLearn
	ExportRunQuick                       = runQuick
	ExportValidateSlug                   = validateSlug
	ExportValidateSource                 = validateSource
	ExportWriteMemory                    = writeMemory
)

type ExportFactFields = factFields

type ExportFeedbackFields = feedbackFields

type ExportMOCFields = mocFields

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

// ExportNewCyclePersisterAdapter returns a cycle.Persister built from injected deps.
func ExportNewCyclePersisterAdapter(
	dataDir string,
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error),
	lister memoryLister,
	stdout io.Writer,
) interface {
	WriteFact(ctx context.Context, situation, subject, predicate, object string) (string, bool, error)
	WriteFeedback(ctx context.Context, situation, behavior, impact, action string) (string, bool, error)
} {
	return &cyclePersisterAdapter{
		dataDir: dataDir,
		caller:  caller,
		lister:  lister,
		stdout:  stdout,
	}
}

// ExportNewCycleRecallerAdapter returns a cycle.Recaller built with no LLM summarizer (nil-safe error path).
func ExportNewCycleRecallerAdapter(dataDir string, summarizer recall.SummarizerI) interface {
	Recall(ctx context.Context, projectDir, query string) (string, error)
} {
	return &cycleRecallerAdapter{dataDir: dataDir, summarizer: summarizer}
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

// ExportNewOsLearnFS returns the production osLearnFS adapter for testing.
func ExportNewOsLearnFS() *osLearnFS { return &osLearnFS{} }

// ExportNewOsQuickFS returns the production osQuickFS adapter for testing.
func ExportNewOsQuickFS() interface {
	StatDir(path string) error
	WriteNew(path string, data []byte) error
} {
	return &osQuickFS{}
}

// ExportNewOsVaultFS returns the production osVaultFS adapter for testing.
func ExportNewOsVaultFS() interface {
	ListMD(dir string) ([]string, error)
	ReadFile(path string) ([]byte, error)
} {
	return &osVaultFS{}
}

// ExportParseConflictLine wraps parseConflictLine for testing.
func ExportParseConflictLine(line, dataDir string, stdout io.Writer) {
	parseConflictLine(line, dataDir, stdout)
}

// ExportRunLearnFromFactArgs invokes the unexported runLearnFromFactArgs for testing.
func ExportRunLearnFromFactArgs(ctx context.Context, a LearnFactArgs, stdout io.Writer) error {
	return runLearnFromFactArgs(ctx, a, stdout)
}

// ExportRunLearnFromFeedbackArgs invokes the unexported runLearnFromFeedbackArgs for testing.
func ExportRunLearnFromFeedbackArgs(ctx context.Context, a LearnFeedbackArgs, stdout io.Writer) error {
	return runLearnFromFeedbackArgs(ctx, a, stdout)
}

// ExportRunLearnFromMOCArgs invokes the unexported runLearnFromMOCArgs for testing.
func ExportRunLearnFromMOCArgs(ctx context.Context, a LearnMOCArgs, stdout io.Writer) error {
	return runLearnFromMOCArgs(ctx, a, stdout)
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
