package cli

import (
	"context"
	"io"

	"github.com/toejough/engram/internal/transcript"
)

// Exported variables.
var (
	ExportBuildAndInstall            = buildAndInstall
	ExportExtractLuhmannFromFilename = extractLuhmannFromFilename
	ExportLearnPath                  = learnPath
	ExportMarshalFrontmatter         = marshalFrontmatter
	ExportNewErrHandler              = newErrHandler
	ExportNextLuhmannID              = nextLuhmannID
	ExportRenderFactBody             = renderFactBody
	ExportRenderFactFrontmatter      = renderFactFrontmatter
	ExportRenderFeedbackBody         = renderFeedbackBody
	ExportRenderFeedbackFrontmatter  = renderFeedbackFrontmatter
	ExportRenderMOCBody              = renderMOCBody
	ExportRenderMOCFrontmatter       = renderMOCFrontmatter
	ExportRenderRelatedSection       = renderRelatedSection
	ExportResolveVault               = resolveVault
	ExportRunBuildSelf               = runBuildSelf
	ExportRunLearn                   = runLearn
	ExportValidateSlug               = validateSlug
)

// Exported types.
type ExportFactFields = factFields

type ExportFeedbackFields = feedbackFields

type ExportMOCFields = mocFields

// Exported functions.

// ExportEmitTranscripts exposes emitTranscripts for whitebox testing.
func ExportEmitTranscripts(
	reader transcript.Reader,
	entries []transcript.FileEntry,
	stdout io.Writer,
) error {
	return emitTranscripts(reader, entries, stdout)
}

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

// RunRecallForTest exposes runRecall for whitebox testing.
func RunRecallForTest(ctx context.Context, args RecallArgs, stdout io.Writer) error {
	return runRecall(ctx, args, stdout)
}

// RunTranscriptForTest exposes runTranscript for whitebox testing.
func RunTranscriptForTest(ctx context.Context, args TranscriptArgs, stdout io.Writer) error {
	return runTranscript(ctx, args, stdout)
}
