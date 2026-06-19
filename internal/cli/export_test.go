package cli

import (
	"context"
	"io"
	"math"
	"testing"
	"time"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/transcript"
)

// Exported variables.
var (
	ErrCheckFailedForTest            = errCheckFailed
	ErrLearnBadTierForTest           = errLearnBadTier
	ErrQueryModeConflict             = errQueryModeConflict
	ErrResituateNoteNotFoundForTest  = errResituateNoteNotFound
	ExportAnyHarnessFailed           = anyHarnessFailed
	ExportApplyProjectFilter         = applyProjectFilter
	ExportApplyTierFilter            = applyTierFilter
	ExportAutoEmbedNote              = autoEmbedNote
	ExportBumpLastUsed               = bumpLastUsed
	ExportComputePrecedingLinks      = computePrecedingLinks
	ExportDefaultRecencyParams       = defaultRecencyParams
	ExportExtractLuhmannFromFilename = extractLuhmannFromFilename
	ExportFillRecencyBand            = fillRecencyBand
	ExportFinishUpdate               = finishUpdate
	ExportInitializeVault            = initializeVault
	ExportKindFromContent            = kindFromContent
	ExportLearnPath                  = learnPath
	ExportLogWarningToStderr         = logWarningToStderrf
	ExportMarshalFrontmatter         = marshalFrontmatter
	ExportMaxTurnBySource            = maxTurnBySource
	ExportMigrateRelationLinks       = migrateRelationLinks
	ExportMostRecentlyUsedNoteItems  = mostRecentlyUsedNoteItems
	ExportNewErrHandler              = newErrHandler
	ExportNewOsActivateDeps          = newOsActivateDeps
	ExportNewOsCheckDeps             = newOsCheckDeps
	ExportNewOsMigrateDeps           = newOsMigrateDeps
	ExportNewOsShowDeps              = newOsShowDeps
	ExportNextLuhmannID              = nextLuhmannID
	ExportNoteAgeDays                = noteAgeDays
	ExportParseCreatedFromNote       = parseCreatedFromNote
	ExportParseTurnN                 = parseTurnN
	ExportPluralFile                 = pluralFile
	ExportPrintLinkExamples          = printLinkExamples
	ExportPrintNoteExamples          = printNoteExamples
	ExportRecencyMultiplier          = recencyMultiplier
	ExportRenderEpisodeBody          = renderEpisodeBody
	ExportRenderEpisodeFrontmatter   = renderEpisodeFrontmatter
	ExportRenderFactBody             = renderFactBody
	ExportRenderFactFrontmatter      = renderFactFrontmatter
	ExportRenderFeedbackBody         = renderFeedbackBody
	ExportRenderFeedbackFrontmatter  = renderFeedbackFrontmatter
	ExportRenderRelatedSection       = renderRelatedSection
	ExportResolveRelationTargets     = resolveRelationTargets
	ExportResolveVault               = resolveVault
	ExportRunActivate                = RunActivate
	ExportRunLearn                   = runLearn
	ExportRunUpdate                  = runUpdate
	ExportSelectStates               = selectStates
	ExportShouldEmbed                = func(args EmbedApplyArgs, state embed.State) bool {
		return selectStates(args).shouldEmbed(state)
	}
	ExportSortScoredDesc      = sortScoredDesc
	ExportSourceAgeDays       = sourceAgeDays
	ExportTildify             = tildify
	ExportValidateIssueID     = validateIssueID
	ExportValidateProjectSlug = validateProjectSlug
	ExportValidateSlug        = validateSlug
	ExportWriteUpdateReport   = writeUpdateReport
)

type ExportEpisodeFields = episodeFields

// ExportEpisodeLink aliases the unexported episodeLink so cli_test can assert
// on computed preceding-episode links.
type ExportEpisodeLink = episodeLink

// ExportEpisodeRange aliases the exported EpisodeRange for symmetry with the
// other Export* test handles.
type ExportEpisodeRange = EpisodeRange

type ExportFactFields = factFields

type ExportFeedbackFields = feedbackFields

type ExportRecencyParams = recencyParams

// ExportResolvedItem aliases the unexported resolvedItem so cli_test can
// construct test fixtures via ExportNewResolvedItem.
type ExportResolvedItem = resolvedItem

type ExportScoredChunk = scoredChunk

// Exported types.
type ExportVaultInitFS = VaultInitFS

// AdvanceAndReportMarkerForTest exposes advanceAndReportMarker for unit testing.
func AdvanceAndReportMarkerForTest(
	markerPath string,
	fromTime, lastIncluded time.Time,
	hadEntries, pending bool,
	now time.Time,
	stdout io.Writer,
) error {
	return advanceAndReportMarker(markerPath, fromTime, lastIncluded, hadEntries, pending, now, stdout)
}

// DefaultSessionPathResolverForTest exposes defaultSessionPathResolver
// for coverage. The resolver maps a Claude Code session ID to its
// per-project JSONL path.
func DefaultSessionPathResolverForTest(sessionID string) (string, error) {
	return defaultSessionPathResolver(sessionID)
}

// EmitSegmentsForTest exposes emitSegments for unit testing.
func EmitSegmentsForTest(
	reader transcript.SegmentsReader,
	entries []transcript.FileEntry,
	maxBytes int,
	seed map[string]time.Time,
	stdout io.Writer,
) (map[string]time.Time, map[string]bool, map[string]time.Time, error) {
	result, err := emitSegments(reader, entries, maxBytes, seed, stdout)

	return result.lastIncluded, result.hadEntries, result.firstUnincluded, err
}

// EmitTranscriptsForTest is an exported entry point so the cli_test package
// can exercise emitTranscripts directly without going through the full
// runTranscript flow. Returns the lastIncluded, hadEntries, and
// firstUnincluded per-source maps. Production code does not call this.
func EmitTranscriptsForTest(
	reader transcript.Reader,
	entries []transcript.FileEntry,
	maxBytes int,
	seed map[string]time.Time,
	stdout io.Writer,
) (map[string]time.Time, map[string]bool, map[string]time.Time, error) {
	result, err := emitTranscripts(reader, entries, maxBytes, seed, stdout)

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

// ExportApplyChunkRecencyByTime exposes the new per-IngestedAt applyChunkRecency for recency tests.
func ExportApplyChunkRecencyByTime(
	scored []scoredChunk, now time.Time, maxTurnBySrc map[string]int, p recencyParams,
) []scoredChunk {
	return applyChunkRecency(scored, now, maxTurnBySrc, p)
}

// ExportApplyCombinedRecencyBand exposes applyCombinedRecencyBand for band interleave tests.
func ExportApplyCombinedRecencyBand(
	items []resolvedItem,
	chunkMust []resolvedItem,
	nowFn func() time.Time,
	limit int,
	chunksActive bool,
) []resolvedItem {
	return applyCombinedRecencyBand(items, chunkMust, nowFn, limit, chunksActive)
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

// ExportBuildChunkIDSet exposes buildChunkIDSet for validation tests.
// Component 5 reuses buildChunkIDSet (not a second implementation) via AmendDeps injection.
func ExportBuildChunkIDSet(
	chunksDir string,
	listIndexes func(dir string) ([]string, error),
	readFile func(path string) ([]byte, error),
) (map[string]bool, error) {
	return buildChunkIDSet(chunksDir, listIndexes, readFile)
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
	_, err := emitTranscripts(reader, entries, math.MaxInt32, nil, stdout)

	return err
}

// ExportIndexFileName exposes sourceSlug-based index naming so tests can
// locate a source's chunk index file.
func ExportIndexFileName(source string) string { return sourceSlug(source) + ".jsonl" }

// ExportLoadPriorRecords exposes loadPriorRecords for ingest unit tests.
func ExportLoadPriorRecords(indexPath string, deps IngestDeps) map[string]chunk.Record {
	return loadPriorRecords(indexPath, deps)
}

// ExportMergeIntoExisting exposes mergeIntoExisting for whitebox testing.
func ExportMergeIntoExisting(existing, src *resolvedItem) {
	mergeIntoExisting(existing, src)
}

// ExportNewChunkResolvedItem builds a chunk-kind resolvedItem for band tests.
// notePath mirrors mergeChunkSpace's "source#anchor" form.
func ExportNewChunkResolvedItem(notePath string, score float32) resolvedItem {
	return resolvedItem{notePath: notePath, score: score, kind: chunkItemKind}
}

// ExportNewNoteResolvedItem builds a note-kind resolvedItem for recency band
// tests. lastUsed and created are YYYY-MM-DD strings (empty = not set).
// kind is intentionally left blank — the zero value means "note" in the
// resolvedItem model (only chunkItemKind overrides content-derived detection).
func ExportNewNoteResolvedItem(notePath, lastUsed, created string) resolvedItem {
	return resolvedItem{notePath: notePath, lastUsed: lastUsed, created: created}
}

// ExportNewNoteResolvedItemWithBaseScore builds a note-kind resolvedItem with
// an explicit baseScore, for testing mergeIntoExisting activation logic.
func ExportNewNoteResolvedItemWithBaseScore(notePath string, baseScore float32, lastUsed, created string) resolvedItem {
	return resolvedItem{notePath: notePath, baseScore: baseScore, lastUsed: lastUsed, created: created}
}

// ExportNewOsChunkQueryDeps returns production ChunkQueryDeps with an
// injected embedder, mirroring ExportNewOsIngestDeps.
func ExportNewOsChunkQueryDeps(emb embed.Embedder) ChunkQueryDeps {
	deps := newOsChunkQueryDeps()
	deps.Embedder = emb

	return deps
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

// ExportNewOsIngestDeps returns production IngestDeps with an injected
// embedder so coverage tests can drive the wiring without unpacking the
// lazy bundled embedder.
func ExportNewOsIngestDeps(emb embed.Embedder) IngestDeps {
	deps := newOsIngestDeps()
	deps.Embedder = emb

	return deps
}

// ExportNewOsLearnFS returns the production osLearnFS adapter for testing.
func ExportNewOsLearnFS() *osLearnFS { return &osLearnFS{} }

// ExportNewOsMigrateEpisodesDeps returns production MigrateEpisodesDeps with an
// injected embedder so integration tests can drive Scan/Read/Write against a
// temp vault without unpacking the lazy bundled embedder.
func ExportNewOsMigrateEpisodesDeps(emb embed.Embedder) MigrateEpisodesDeps {
	deps := newOsMigrateEpisodesDeps()
	deps.Embedder = emb

	return deps
}

// ExportNewOsResituateDeps returns production ResituateDeps with an injected
// embedder so coverage tests can drive Scan/Read/Write without unpacking the
// lazy bundled embedder.
func ExportNewOsResituateDeps(emb embed.Embedder) ResituateDeps {
	deps := newOsResituateDeps()
	deps.Embedder = emb

	return deps
}

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

// ExportNewRecencyParams builds a recencyParams for tests.
func ExportNewRecencyParams(halfLifeDays, tailWeight float64, floor int) recencyParams {
	return recencyParams{halfLifeDays: halfLifeDays, tailWeight: tailWeight, floor: floor}
}

// ExportNewResolvedItem builds a resolvedItem for tests that need to
// drive applyProjectFilter without going through the full pipeline.
func ExportNewResolvedItem(notePath, content string) ExportResolvedItem {
	return ExportResolvedItem{notePath: notePath, content: content}
}

// ExportNewScoredChunk builds a scoredChunk for tests.
func ExportNewScoredChunk(rec chunk.Record, score float32) scoredChunk {
	return scoredChunk{record: rec, score: score}
}

// ExportNewScoredChunkWithIngestedAt builds a scoredChunk with IngestedAt set for recency tests.
func ExportNewScoredChunkWithIngestedAt(rec chunk.Record, score float32, ingestedAt time.Time) scoredChunk {
	rec.IngestedAt = ingestedAt

	return scoredChunk{record: rec, score: score}
}

// ExportNewestChunkItems exposes newestChunkItems (new 2-arg signature: no ages map).
func ExportNewestChunkItems(scored []scoredChunk, n int) []resolvedItem {
	return newestChunkItems(scored, n)
}

// ExportNewestChunkItemsByTime is an alias for tests that use the IngestedAt-keyed sort.
// Both names wrap the same 2-arg newestChunkItems; ExportNewestChunkItems keeps the
// existing test-helper name stable, ExportNewestChunkItemsByTime is the semantic form.
func ExportNewestChunkItemsByTime(scored []scoredChunk, n int) []resolvedItem {
	return newestChunkItems(scored, n)
}

// ExportParseEpisodeBody exposes parseEpisodeBody for round-trip testing,
// returning the summary, transcript, and "basename|rationale" relation
// entries decomposed from an episode body.
func ExportParseEpisodeBody(body string) (summary, transcript string, relations []string) {
	parsed := parseEpisodeBody(body)

	return parsed.summary, parsed.transcript, parsed.relations
}

// ExportRecencyFloor exposes the floor field of recencyParams for tests.
func ExportRecencyFloor(p recencyParams) int { return p.floor }

// ExportResolvedItemBaseScore exposes the pre-decay baseScore field for
// activation-cutoff and band assertions (populated by Task 2.3).
func ExportResolvedItemBaseScore(item ExportResolvedItem) float32 { return item.baseScore }

// ExportResolvedItemCreated exposes the created frontmatter date field.
func ExportResolvedItemCreated(item ExportResolvedItem) string { return item.created }

// ExportResolvedItemLastUsed exposes the LastUsed sidecar date field.
func ExportResolvedItemLastUsed(item ExportResolvedItem) string { return item.lastUsed }

// ExportResolvedItemPath exposes the unexported notePath field for assertions.
func ExportResolvedItemPath(item ExportResolvedItem) string { return item.notePath }

// ExportResolvedItemScore exposes the unexported score field for assertions.
func ExportResolvedItemScore(item ExportResolvedItem) float32 { return item.score }

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

func ExportScoredChunkRecord(s scoredChunk) chunk.Record { return s.record }

// ExportScoredChunkScore / Record expose the unexported fields for assertions.
func ExportScoredChunkScore(s scoredChunk) float32 { return s.score }

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
	opencodeDBPath string,
	deps LearnDeps,
	stdout io.Writer,
) error {
	return runLearnFromEpisodeArgsWithReader(ctx, a, reader, sessionPath, opencodeDBPath, deps, stdout)
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

// TestMergeIntoExisting_SetsInDegreeFromSrc covers the branch where existing.inDegree is
// nil (note not a hub in an earlier phrase) and src.inDegree is set (hub in a later phrase).
// This branch cannot be exercised via RunQuery black-box tests because undirected BFS always
// expands to a direct-hit note's linkers at hop 1, making the note a hub in every phrase
// that contains it as a direct hit.
func TestMergeIntoExisting_SetsInDegreeFromSrc(t *testing.T) {
	t.Parallel()

	const expectedDegree = 7

	deg := expectedDegree
	existing := &resolvedItem{
		notePath:    "X.md",
		score:       0.8,
		provenances: []string{provenanceDirect},
	}
	src := &resolvedItem{
		notePath:    "X.md",
		score:       0.6,
		provenances: []string{provenanceHub},
		inDegree:    &deg,
	}

	mergeIntoExisting(existing, src)

	if existing.inDegree == nil {
		t.Fatal("expected inDegree to be set, got nil")
	}

	if *existing.inDegree != expectedDegree {
		t.Fatalf("expected inDegree=%d, got %d", expectedDegree, *existing.inDegree)
	}
}
