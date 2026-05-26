package cli

import (
	"context"
	"io"
	"math"
	"time"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/transcript"
)

// Exported variables.
var (
	ExportAnyHarnessFailed           = anyHarnessFailed
	ExportAutoEmbedNote              = autoEmbedNote
	ExportExtractLuhmannFromFilename = extractLuhmannFromFilename
	ExportFinishUpdate               = finishUpdate
	ExportInitializeVault            = initializeVault
	ExportKindFromContent            = kindFromContent
	ExportLearnPath                  = learnPath
	ExportLogWarningToStderr         = logWarningToStderrf
	ExportMarshalFrontmatter         = marshalFrontmatter
	ExportNewErrHandler              = newErrHandler
	ExportNextLuhmannID              = nextLuhmannID
	ExportPluralFile                 = pluralFile
	ExportRenderFactBody             = renderFactBody
	ExportRenderFactFrontmatter      = renderFactFrontmatter
	ExportRenderFeedbackBody         = renderFeedbackBody
	ExportRenderFeedbackFrontmatter  = renderFeedbackFrontmatter
	ExportRenderRelatedSection       = renderRelatedSection
	ExportResolveVault               = resolveVault
	ExportRunLearn                   = runLearn
	ExportRunUpdate                  = runUpdate
	ExportSelectStates               = selectStates
	ExportShouldEmbed                = func(args EmbedApplyArgs, state embed.State) bool {
		return selectStates(args).shouldEmbed(state)
	}
	ExportTildify           = tildify
	ExportValidateSlug      = validateSlug
	ExportWriteUpdateReport = writeUpdateReport
)

type ExportFactFields = factFields

type ExportFeedbackFields = feedbackFields

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

// DefaultSessionPathResolverForTest exposes defaultSessionPathResolver
// for coverage. The resolver maps a Claude Code session ID to its
// per-project JSONL path.
func DefaultSessionPathResolverForTest(sessionID string) (string, error) {
	return defaultSessionPathResolver(sessionID)
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

// ExportAppendUniqueProvenance returns the provenances slice after adding
// role twice via the helper; verifies idempotency in tests.
func ExportAppendUniqueProvenance(initial []string, roles ...string) []string {
	item := &resolvedItem{provenances: initial}
	for _, role := range roles {
		appendUniqueProvenance(item, role)
	}

	return item.provenances
}

// ExportBreakRepresentativeTie is a whitebox handle on the tiebreak helper
// used by cluster representative selection.
func ExportBreakRepresentativeTie(
	scoreA float32,
	pathA string,
	scoreB float32,
	pathB string,
) string {
	subgraph := expandedSubgraph{
		members: []subgraphMember{
			{notePath: pathA, score: scoreA, vector: []float32{1, 0}},
			{notePath: pathB, score: scoreB, vector: []float32{1, 0}},
		},
	}

	winnerIdx := breakRepresentativeTie(subgraph, 0, 1)

	return subgraph.members[winnerIdx].notePath
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

// ExportNewOsEmbedDeps returns production EmbedDeps with an injected
// embedder so coverage tests can drive Read/Write/Scan without going
// through the lazy bundled embedder.
func ExportNewOsEmbedDeps(emb embed.Embedder) EmbedDeps {
	deps := newOsEmbedDeps()
	deps.Embedder = emb

	return deps
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

// NewTranscriptDepsForTest exposes newTranscriptDeps for whitebox testing.
func NewTranscriptDepsForTest(cwd string) (transcript.Finder, transcript.Reader) {
	return newTranscriptDeps(cwd)
}

// ParseFromTranscriptRangeForTest exposes parseFromTranscriptRange so
// tests can drive every error branch (malformed input, unparseable
// timestamps, out-of-order range) without going through the full
// runLearnFromEpisodeArgsWithReader path.
func ParseFromTranscriptRangeForTest(raw string) (string, time.Time, time.Time, error) {
	return parseFromTranscriptRange(raw)
}

// ResolveMaxBytesForTest exposes resolveMaxBytes for unit testing.
func ResolveMaxBytesForTest(maxBytes int) int { return resolveMaxBytes(maxBytes) }

// ResolveProjectSlugForTest exposes resolveProjectSlug for unit testing.
func ResolveProjectSlugForTest(args TranscriptArgs) (string, error) {
	return resolveProjectSlug(args)
}

// ResolveSessionPathForTest exposes resolveSessionPath so the cwd/home
// error branches are unit-testable via injected fakes.
func ResolveSessionPathForTest(
	sessionID string,
	getwd func() (string, error),
	homeDir func() (string, error),
) (string, error) {
	return resolveSessionPath(sessionID, getwd, homeDir)
}

// ResolveStateDirForTest exposes resolveStateDir for unit testing.
func ResolveStateDirForTest(args TranscriptArgs) (string, error) {
	return resolveStateDir(args)
}

// RunLearnFromEpisodeArgsWithReaderForTest exposes
// runLearnFromEpisodeArgsWithReader so tests can drive the
// --from-transcript-range / --transcript-text body-source XOR with an
// injected RangeReader and session-path resolver.
func RunLearnFromEpisodeArgsWithReaderForTest(
	ctx context.Context,
	a LearnEpisodeArgs,
	reader transcript.RangeReader,
	sessionPath func(sessionID string) (string, error),
	deps LearnDeps,
	stdout io.Writer,
) error {
	return runLearnFromEpisodeArgsWithReader(ctx, a, reader, sessionPath, deps, stdout)
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
