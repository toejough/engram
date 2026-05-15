package cli

import (
	"context"
	"io"
	"math"
	"time"

	"github.com/toejough/engram/internal/transcript"
)

// Exported variables.
var (
	ExportAnyHarnessFailed           = anyHarnessFailed
	ExportExtractLuhmannFromFilename = extractLuhmannFromFilename
	ExportFinishUpdate               = finishUpdate
	ExportInitializeVault            = initializeVault
	ExportLearnPath                  = learnPath
	ExportMarshalFrontmatter         = marshalFrontmatter
	ExportNewErrHandler              = newErrHandler
	ExportNextLuhmannID              = nextLuhmannID
	ExportPluralFile                 = pluralFile
	ExportRenderFactBody             = renderFactBody
	ExportRenderFactFrontmatter      = renderFactFrontmatter
	ExportRenderFeedbackBody         = renderFeedbackBody
	ExportRenderFeedbackFrontmatter  = renderFeedbackFrontmatter
	ExportRenderMOCBody              = renderMOCBody
	ExportRenderMOCFrontmatter       = renderMOCFrontmatter
	ExportRenderRelatedSection       = renderRelatedSection
	ExportResolveVault               = resolveVault
	ExportRunLearn                   = runLearn
	ExportRunUpdate                  = runUpdate
	ExportTildify                    = tildify
	ExportValidateSlug               = validateSlug
	ExportWriteUpdateReport          = writeUpdateReport
)

type ExportFactFields = factFields

type ExportFeedbackFields = feedbackFields

type ExportMOCFields = mocFields

// Exported types.
type ExportVaultInitFS = VaultInitFS

// AdvanceAndReportMarkerForTest exposes advanceAndReportMarker for unit testing.
func AdvanceAndReportMarkerForTest(
	markerPath string,
	fromTime, lastIncluded time.Time,
	hadEntries bool,
	now time.Time,
	stdout io.Writer,
) error {
	return advanceAndReportMarker(markerPath, fromTime, lastIncluded, hadEntries, now, stdout)
}

// EmitTranscriptsForTest is an exported entry point so the cli_test package
// can exercise emitTranscripts directly without going through the full
// runTranscript flow. Returns the lastIncluded, hadEntries, and
// firstUnincluded per-source maps. Production code does not call this.
func EmitTranscriptsForTest(
	reader transcript.Reader,
	entries []transcript.FileEntry,
	maxBytes int,
	stdout io.Writer,
) (map[string]time.Time, map[string]bool, map[string]time.Time, error) {
	result, err := emitTranscripts(reader, entries, maxBytes, stdout)

	return result.lastIncluded, result.hadEntries, result.firstUnincluded, err
}

// Exported functions.

// ExportEmitTranscripts exposes emitTranscripts for whitebox testing with an
// unlimited byte budget. Discards per-source bookkeeping because the legacy
// tests using this wrapper only care about error paths.
func ExportEmitTranscripts(
	reader transcript.Reader,
	entries []transcript.FileEntry,
	stdout io.Writer,
) error {
	_, err := emitTranscripts(reader, entries, math.MaxInt32, stdout)

	return err
}

// ExportNewOsCommander returns the production Commander adapter for testing.
func ExportNewOsCommander() *osCommander { return &osCommander{} }

// ExportNewOsDirLister creates an osDirLister for testing.
func ExportNewOsDirLister() transcript.DirLister {
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

// ExportNewOsUpdateEnv returns the production Env adapter for testing.
func ExportNewOsUpdateEnv() *osUpdateEnv { return &osUpdateEnv{} }

// ExportNewOsUpdateFS returns the production Filesystem adapter for testing.
func ExportNewOsUpdateFS() *osUpdateFS { return &osUpdateFS{} }

// ExportNewOsVaultFS returns the production osVaultFS adapter for testing.
func ExportNewOsVaultFS() interface {
	ListMD(dir string) ([]string, error)
	ReadFile(path string) ([]byte, error)
} {
	return &osVaultFS{}
}

// ExportRunLearnFromFactArgs invokes the unexported runLearnFromFactArgs for testing.
func ExportRunLearnFromFactArgs(ctx context.Context, a LearnFactArgs, stdout io.Writer) error {
	return runLearnFromFactArgs(ctx, a, stdout)
}

// ExportRunLearnFromFeedbackArgs invokes the unexported runLearnFromFeedbackArgs for testing.
func ExportRunLearnFromFeedbackArgs(
	ctx context.Context,
	a LearnFeedbackArgs,
	stdout io.Writer,
) error {
	return runLearnFromFeedbackArgs(ctx, a, stdout)
}

// ExportRunLearnFromMOCArgs invokes the unexported runLearnFromMOCArgs for testing.
func ExportRunLearnFromMOCArgs(ctx context.Context, a LearnMOCArgs, stdout io.Writer) error {
	return runLearnFromMOCArgs(ctx, a, stdout)
}

// NewTranscriptDepsForTest exposes newTranscriptDeps for whitebox testing.
func NewTranscriptDepsForTest(cwd string) (transcript.Finder, transcript.Reader) {
	return newTranscriptDeps(cwd)
}

// ResolveMaxBytesForTest exposes resolveMaxBytes for unit testing.
func ResolveMaxBytesForTest(maxBytes int) int { return resolveMaxBytes(maxBytes) }

// ResolveProjectSlugForTest exposes resolveProjectSlug for unit testing.
func ResolveProjectSlugForTest(args TranscriptArgs) (string, error) {
	return resolveProjectSlug(args)
}

// ResolveStateDirForTest exposes resolveStateDir for unit testing.
func ResolveStateDirForTest(args TranscriptArgs) (string, error) {
	return resolveStateDir(args)
}

// RunRecallForTest exposes runRecall for whitebox testing.
func RunRecallForTest(ctx context.Context, args RecallArgs, stdout io.Writer) error {
	return runRecall(ctx, args, stdout)
}

// RunTranscriptForTest exposes runTranscript for whitebox testing.
func RunTranscriptForTest(
	ctx context.Context,
	args TranscriptArgs,
	finder transcript.Finder,
	reader transcript.Reader,
	stdout io.Writer,
) error {
	return runTranscript(ctx, args, finder, reader, stdout)
}
