package cli

import (
	"context"
	"io"
	"time"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/embed"
)

// Exported variables.
var (
	ErrCheckFailedForTest              = errCheckFailed
	ErrLearnBadTierForTest             = errLearnBadTier
	ErrResituateNoteNotFoundForTest    = errResituateNoteNotFound
	ExportAnyHarnessFailed             = anyHarnessFailed
	ExportApplyProjectFilter           = applyProjectFilter
	ExportAutoEmbedNote                = autoEmbedNote
	ExportBumpLastUsed                 = bumpLastUsed
	ExportDefaultRecencyParams         = defaultRecencyParams
	ExportExtractLuhmannFromFilename   = extractLuhmannFromFilename
	ExportFillRecencyBand              = fillRecencyBand
	ExportFinishUpdate                 = finishUpdate
	ExportInitializeVault              = initializeVault
	ExportKindFromContent              = kindFromContent
	ExportLearnPath                    = learnPath
	ExportLogWarningToStderr           = logWarningToStderrf
	ExportMarshalFrontmatter           = marshalFrontmatter
	ExportMaxTurnBySource              = maxTurnBySource
	ExportMigrateRelationLinks         = migrateRelationLinks
	ExportMostRecentlyUsedNoteItems    = mostRecentlyUsedNoteItems
	ExportNewErrHandler                = newErrHandler
	ExportNewOsActivateDeps            = newOsActivateDeps
	ExportNewOsAmendDeps               = newOsAmendDeps
	ExportNewOsCheckDeps               = newOsCheckDeps
	ExportNewOsMigrateDeps             = newOsMigrateDeps
	ExportNewOsPruneDeps               = newOsPruneDeps
	ExportNewOsShowDeps                = newOsShowDeps
	ExportNextLuhmannID                = nextLuhmannID
	ExportNoteAgeDays                  = noteAgeDays
	ExportParseCreatedFromNote         = parseCreatedFromNote
	ExportParseTurnN                   = parseTurnN
	ExportPluralFile                   = pluralFile
	ExportPrintLinkExamples            = printLinkExamples
	ExportPrintNoteExamples            = printNoteExamples
	ExportRecencyMultiplier            = recencyMultiplier
	ExportRenderFactBody               = renderFactBody
	ExportRenderFactFrontmatter        = renderFactFrontmatter
	ExportRenderFeedbackBody           = renderFeedbackBody
	ExportRenderFeedbackFrontmatter    = renderFeedbackFrontmatter
	ExportRenderRelatedSection         = renderRelatedSection
	ExportResolveRelationTargets       = resolveRelationTargets
	ExportResolveRelationTargetsStrict = resolveRelationTargetsStrict
	ExportResolveVault                 = resolveVault
	ExportRunActivate                  = RunActivate
	ExportRunAmend                     = RunAmend
	ExportRunLearn                     = runLearn
	ExportRunUpdate                    = runUpdate
	ExportSelectStates                 = selectStates
	ExportShouldEmbed                  = func(args EmbedApplyArgs, state embed.State) bool {
		return selectStates(args).shouldEmbed(state)
	}
	ExportShouldSkipDir       = shouldSkipDir
	ExportSortScoredDesc      = sortScoredDesc
	ExportTildify             = tildify
	ExportValidateIssueID     = validateIssueID
	ExportValidateProjectSlug = validateProjectSlug
	ExportValidateSlug        = validateSlug
	ExportWriteUpdateReport   = writeUpdateReport
)

type ExportFactFields = factFields

type ExportFeedbackFields = feedbackFields

// ExportMergeClusterRepsCall is a simplified wrapper around mergeClusterReps
// that takes plain slices instead of unexported types, for whitebox testing.
// members is a list of (notePath, score, content) tuples.
// representatives maps clusterID → memberIndex.
// Returns the updated byBasename map as a slice of (path, provenances, clusterID) tuples.
type ExportMergeClusterRepsEntry struct {
	NotePath    string
	Provenances []string
	ClusterID   *int
}

type ExportRecencyParams = recencyParams

// ExportResolvedItem aliases the unexported resolvedItem so cli_test can
// construct test fixtures via ExportNewResolvedItem.
type ExportResolvedItem = resolvedItem

type ExportScoredChunk = scoredChunk

// Exported types.
type ExportVaultInitFS = VaultInitFS

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
	matchSet := matchedSet{
		members: []matchedMember{
			{notePath: pathA, score: scoreA, vector: []float32{1, 0}},
			{notePath: pathB, score: scoreB, vector: []float32{1, 0}},
		},
	}

	winnerIdx := breakRepresentativeTie(matchSet, 0, 1)

	return matchSet.members[winnerIdx].notePath
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

// ExportIndexFileName exposes sourceSlug-based index naming so tests can
// locate a source's chunk index file.
func ExportIndexFileName(source string) string { return sourceSlug(source) + ".jsonl" }

// ExportLoadPriorRecords exposes loadPriorRecords for ingest unit tests.
func ExportLoadPriorRecords(indexPath string, deps IngestDeps) map[string]chunk.Record {
	return loadPriorRecords(indexPath, deps)
}

// ExportMergeClusterReps drives mergeClusterReps with plain-data inputs.
func ExportMergeClusterReps(
	memberPaths []string,
	memberScores []float32,
	memberContents []string,
	representatives map[int]int,
) []ExportMergeClusterRepsEntry {
	members := make([]matchedMember, len(memberPaths))
	for i, path := range memberPaths {
		basename := path
		members[i] = matchedMember{
			basename: basename,
			notePath: path,
			score:    memberScores[i],
			content:  memberContents[i],
		}
	}

	matchSet := matchedSet{members: members}

	clusters := clusterReport{
		representatives: make([]int, 0),
	}

	for clusterID := range representatives {
		for clusterID >= len(clusters.representatives) {
			clusters.representatives = append(clusters.representatives, -1)
		}
	}

	for clusterID, memberIdx := range representatives {
		clusters.representatives[clusterID] = memberIdx
	}

	byBasename := make(map[string]*resolvedItem)
	mergeClusterReps(matchSet, clusters, byBasename)

	result := make([]ExportMergeClusterRepsEntry, 0, len(byBasename))
	for _, item := range byBasename {
		var clusterID *int
		if item.clusterID != nil {
			v := *item.clusterID
			clusterID = &v
		}

		result = append(result, ExportMergeClusterRepsEntry{
			NotePath:    item.notePath,
			Provenances: item.provenances,
			ClusterID:   clusterID,
		})
	}

	return result
}

// ExportMergeIntoExisting exposes mergeIntoExisting for whitebox testing.
func ExportMergeIntoExisting(existing, src *resolvedItem) {
	mergeIntoExisting(existing, src)
}

// ExportNewChunkResolvedItem builds a chunk-kind resolvedItem for band tests.
// notePath mirrors chunkNotePath's "source#anchor" form.
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

// ExportNewNoteResolvedItemWithInDegree builds a note-kind resolvedItem with
// inDegree set for testing mergeIntoExisting inDegree propagation.
func ExportNewNoteResolvedItemWithInDegree(notePath string, inDegree int) resolvedItem {
	v := inDegree

	return resolvedItem{notePath: notePath, inDegree: &v}
}

// ExportNewNoteResolvedItemWithProvenances builds a note-kind resolvedItem
// with explicit provenances and score, for testing resolvedItemLess ordering.
func ExportNewNoteResolvedItemWithProvenances(
	notePath string, score float32, provenances []string,
) resolvedItem {
	return resolvedItem{notePath: notePath, score: score, provenances: provenances}
}

// ExportNewNoteResolvedItemWithScore builds a note-kind resolvedItem with
// both score and baseScore set, for testing mergeIntoExisting score logic.
func ExportNewNoteResolvedItemWithScore(notePath string, score, baseScore float32) resolvedItem {
	return resolvedItem{notePath: notePath, score: score, baseScore: baseScore}
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

// ExportNewestChunkItems exposes newestChunkItems with the direct provenance.
func ExportNewestChunkItems(scored []scoredChunk, n int) []resolvedItem {
	return newestChunkItems(scored, n, provenanceDirect)
}

// ExportProvenanceRankFor exposes provenanceRankFor for whitebox testing.
func ExportProvenanceRankFor(role string) int { return provenanceRankFor(role) }

// ExportRecencyFloor exposes the floor field of recencyParams for tests.
func ExportRecencyFloor(p recencyParams) int { return p.floor }

// ExportResolvedItemBaseScore exposes the pre-decay baseScore field for
// activation-cutoff and band assertions (populated by Task 2.3).
func ExportResolvedItemBaseScore(item ExportResolvedItem) float32 { return item.baseScore }

// ExportResolvedItemCreated exposes the created frontmatter date field.
func ExportResolvedItemCreated(item ExportResolvedItem) string { return item.created }

// ExportResolvedItemInDegree exposes the inDegree field for assertions.
func ExportResolvedItemInDegree(item resolvedItem) *int { return item.inDegree }

// ExportResolvedItemLastUsed exposes the LastUsed sidecar date field.
func ExportResolvedItemLastUsed(item ExportResolvedItem) string { return item.lastUsed }

// ExportResolvedItemLess exposes resolvedItemLess for whitebox testing.
func ExportResolvedItemLess(a, b resolvedItem) bool { return resolvedItemLess(a, b) }

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
