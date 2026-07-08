package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cluster"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// Exported constants.
const (
	ExportDefaultContentBudget = defaultContentBudget
	ExportDefaultRecentFill    = defaultRecentFill
)

// Exported variables.
var (
	ErrCheckFailedForTest                  = errCheckFailed
	ErrCountBadFilterForTest               = errCountBadFilter
	ErrCountBothModesForTest               = errCountBothModes
	ErrCountNoModeForTest                  = errCountNoMode
	ErrLearnBadTierForTest                 = errLearnBadTier
	ErrQAAnswerSourceRequired              = errQAAnswerSourceRequired
	ErrQACertaintyInvalid                  = errQACertaintyInvalid
	ErrQAContributorNotFound               = errQAContributorNotFound
	ErrQAQuestionRequired                  = errQAQuestionRequired
	ErrQASourceRequired                    = errQASourceRequired
	ErrResituateNoteNotFoundForTest        = errResituateNoteNotFound
	ExportAnyHarnessFailed                 = anyHarnessFailed
	ExportApplyProjectFilter               = applyProjectFilter
	ExportApplySupersedesRideAlong         = applySupersedesRideAlong
	ExportApplyVocabAssignmentAfterAmend   = applyVocabAssignmentAfterAmend
	ExportApplyVocabAssignmentAfterLearn   = applyVocabAssignmentAfterLearn
	ExportAutoEmbedNote                    = autoEmbedNote
	ExportBuildSupersedesInverse           = BuildSupersedesInverse
	ExportBumpLastUsed                     = bumpLastUsed
	ExportBumpMajorVersion                 = bumpMajorVersion
	ExportBumpMinorVersion                 = bumpMinorVersion
	ExportCheckAndPersistVocabRefitTrigger = checkAndPersistVocabRefitTrigger
	ExportCollectTriggerVaultStats         = collectTriggerVaultStats
	ExportCountNonVocabNoteFiles           = countNonVocabNoteFiles
	ExportCountQAPairs                     = countQAPairs
	ExportDefaultRecencyParams             = defaultRecencyParams
	ExportEvaluateVocabTriggers            = evaluateVocabTriggers
	ExportExtractLuhmannFromFilename       = extractLuhmannFromFilename
	ExportFillRecencyBand                  = fillRecencyBand
	ExportFinishUpdate                     = finishUpdate
	ExportInitializeVault                  = initializeVault
	ExportIsQAQuestionFilename             = isQAQuestionFilename
	ExportIsQueryExcludedKind              = isQueryExcludedKind
	ExportIsVocabKind                      = isVocabKind
	ExportKindFromContent                  = kindFromContent
	ExportLearnPath                        = learnPath
	ExportLoadAllVaultNotesMeta            = loadAllVaultNotesMeta
	ExportLoadAssignmentTermVectors        = loadAssignmentTermVectors
	ExportLogWarningToStderr               = logWarningToStderrf
	ExportMarshalFrontmatter               = marshalFrontmatter
	ExportMaxTurnBySource                  = maxTurnBySource
	ExportMostRecentlyUsedNoteItems        = mostRecentlyUsedNoteItems
	ExportNewErrHandler                    = newErrHandler
	ExportNewOsActivateDeps                = newOsActivateDeps
	ExportNewOsAmendDeps                   = newOsAmendDeps
	ExportNewOsCheckDeps                   = newOsCheckDeps
	ExportNewOsCountDeps                   = newOsCountDeps
	ExportNewOsPruneDeps                   = newOsPruneDeps
	ExportNewOsShowDeps                    = newOsShowDeps
	ExportNewOsVocabDeps                   = newOsVocabDeps
	ExportNextLuhmannID                    = nextLuhmannID
	ExportNoteAgeDays                      = noteAgeDays
	ExportNoteContainsAnyRemoval           = noteContainsAnyRemoval
	ExportParseCreatedFromNote             = parseCreatedFromNote
	ExportParseNoteQueryFrontmatter        = parseNoteQueryFrontmatter
	ExportParseSupersedesFlag              = parseSupersedesFlag
	ExportParseTurnN                       = parseTurnN
	ExportPluralFile                       = pluralFile
	ExportPrintLinkExamples                = printLinkExamples
	ExportPrintNoteExamples                = printNoteExamples
	ExportPrintStatsReport                 = printStatsReport
	ExportRecencyMultiplier                = recencyMultiplier
	ExportRenderFactBody                   = renderFactBody
	ExportRenderFactFrontmatter            = renderFactFrontmatter
	ExportRenderFeedbackBody               = renderFeedbackBody
	ExportRenderFeedbackFrontmatter        = renderFeedbackFrontmatter
	ExportRenderQAAnswerNote               = renderQAAnswerNote
	ExportRenderQAQuestionNote             = renderQAQuestionNote
	ExportResolveVault                     = resolveVault
	ExportRetagAllNotesTwoPass             = retagAllNotesTwoPass
	ExportRunActivate                      = RunActivate
	ExportRunAmend                         = RunAmend
	ExportRunLearn                         = runLearn
	ExportRunUpdate                        = runUpdate
	ExportScanNonVocabNotes                = scanNonVocabNotes
	ExportSelectStates                     = selectStates
	ExportShouldEmbed                      = func(args EmbedApplyArgs, state embed.State) bool {
		return selectStates(args).shouldEmbed(state)
	}
	ExportShouldSkipDir        = shouldSkipDir
	ExportTildify              = tildify
	ExportTopDeliveredNotes    = topDeliveredNotes
	ExportValidateContributors = validateContributors
	ExportValidateIssueID      = validateIssueID
	ExportValidateLearnQAArgs  = validateLearnQAArgs
	ExportValidateProjectSlug  = validateProjectSlug
	ExportValidateSlug         = validateSlug
	ExportWriteCentroidsDocRaw = writeCentroidsDocRaw
	ExportWriteUpdateReport    = writeUpdateReport
	ExportWriteVocabAssignment = WriteVocabAssignment
)

// ExportAllVaultNotesMeta aliases AllVaultNotesMeta for cli_test fixtures.
type ExportAllVaultNotesMeta = AllVaultNotesMeta

// ExportCompatibleSidecar aliases the unexported compatibleSidecar so cli_test
// can construct hits slices via ExportNewCompatibleSidecars.
type ExportCompatibleSidecar = compatibleSidecar

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

// ExportNominationEntry aliases NominationEntry for cli_test struct literals.
type ExportNominationEntry = NominationEntry

// ExportQueriedCandidateNote aliases the unexported queryCandidateNote so
// cli_test can construct nomination fixtures for ExportRenderClustersTagNominations.
type ExportQueriedCandidateNote = queryCandidateNote

type ExportRecencyParams = recencyParams

// ExportResolvedItem aliases the unexported resolvedItem so cli_test can
// construct test fixtures via ExportNewResolvedItem.
type ExportResolvedItem = resolvedItem

type ExportScoredCandidate = scoredCandidate

type ExportScoredChunk = scoredChunk

type ExportSupersedesEntry = supersedesEntry

// Exported types.
type ExportVaultInitFS = VaultInitFS

type ExportVocabCentroidEntry = vocabCentroidEntry

// Exported vocab schema types (Task 1).
type ExportVocabCentroidsDoc = vocabCentroidsDoc

type ExportVocabLastRefitDoc = vocabLastRefitDoc

// ExportAppendUniqueProvenance returns the provenances slice after adding
// role twice via the helper; verifies idempotency in tests.
func ExportAppendUniqueProvenance(initial []string, roles ...string) []string {
	item := &resolvedItem{provenances: initial}
	for _, role := range roles {
		appendUniqueProvenance(item, role)
	}

	return item.provenances
}

// ExportApplyVocabAssignmentAfterResituate exposes applyVocabAssignmentAfterResituate
// so resituate_test can assert the vocab-assignment + trigger wiring.
func ExportApplyVocabAssignmentAfterResituate(deps ResituateDeps, vault, notePath, content string) {
	applyVocabAssignmentAfterResituate(deps, vault, notePath, content)
}

// ExportAtomicWriteFile exposes atomicWriteFile for writesafe tests.
func ExportAtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	return atomicWriteFile(path, data, perm)
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

// ExportBuildTagNominationsUnit drives buildTagNominations with empty matchedSet
// and clusterReport (cluster-0 fallback), so unit tests can assert nomination
// results without wiring a full clustering pipeline. The tally is discarded;
// tests asserting the tally use ExportBuildTagNominationsWithTally.
func ExportBuildTagNominationsUnit(
	resolved []resolvedItem,
	meta AllVaultNotesMeta,
) map[int][]queryCandidateNote {
	nominations, _ := buildTagNominations(resolved, meta, matchedSet{}, clusterReport{})

	return nominations
}

// ExportBuildTagNominationsWithClusters drives buildTagNominations with a REAL
// matchedSet + clusterReport built from plain member paths and per-cluster
// member-index lists, so tests can exercise the real clusterID assignment path
// (noteClusterIDForPath) rather than the cluster-0 fallback.
func ExportBuildTagNominationsWithClusters(
	resolved []resolvedItem,
	meta AllVaultNotesMeta,
	memberPaths []string,
	memberIDs [][]int,
) map[int][]queryCandidateNote {
	members := make([]matchedMember, len(memberPaths))
	for i, path := range memberPaths {
		members[i] = matchedMember{notePath: path}
	}

	nominations, _ := buildTagNominations(
		resolved,
		meta,
		matchedSet{members: members},
		clusterReport{memberIDs: memberIDs},
	)

	return nominations
}

// ExportBuildTagNominationsWithTally drives buildTagNominations (cluster-0
// fallback path) and returns the nomination map plus the tally's added and
// dropped counts, for no-silent-caps assertions.
func ExportBuildTagNominationsWithTally(
	resolved []resolvedItem,
	meta AllVaultNotesMeta,
) (map[int][]queryCandidateNote, int, int) {
	nominations, tally := buildTagNominations(resolved, meta, matchedSet{}, clusterReport{})

	return nominations, tally.added, tally.dropped
}

// ExportCapChunkContent builds queryItems from parallel kind/content slices,
// applies capChunkContent, and returns the resulting contents + snipped count.
func ExportCapChunkContent(kinds, contents []string, budget int) ([]string, int) {
	items := make([]queryItem, len(kinds))
	for i := range kinds {
		items[i] = queryItem{Kind: kinds[i], Content: contents[i]}
	}

	capped, snipped := capChunkContent(items, budget)

	out := make([]string, len(capped))
	for i := range capped {
		out[i] = capped[i].Content
	}

	return out, snipped
}

// ExportClearChunkContent builds queryItems from parallel kind/content slices,
// applies clearChunkContent (lazy-chunk mode), and returns the resulting
// contents — chunk contents zeroed, note contents preserved.
func ExportClearChunkContent(kinds, contents []string) []string {
	items := make([]queryItem, len(kinds))
	for i := range kinds {
		items[i] = queryItem{Kind: kinds[i], Content: contents[i]}
	}

	cleared := clearChunkContent(items)

	out := make([]string, len(cleared))
	for i := range cleared {
		out[i] = cleared[i].Content
	}

	return out
}

// ExportCollectVaultStats exposes collectVaultStats for cli_test fixtures.
func ExportCollectVaultStats(names []string, deps VocabStatsDeps, vault string) ([]string, map[string]int, int, int) {
	return collectVaultStats(names, deps, vault)
}

// ExportDoAtomicWrite exposes doAtomicWrite for writesafe tests that need to
// inject a failing rename to cover the rename-error and defer-cleanup paths.
func ExportDoAtomicWrite(
	path string,
	data []byte,
	perm os.FileMode,
	rename func(oldpath, newpath string) error,
) error {
	return doAtomicWrite(path, data, perm, rename)
}

// ExportFlockPath exposes flockPath for the concurrent regression test in
// ingest_test.go. The test goroutines use the real flock, not an injected one,
// so they can race on the SAME lock file and prove the locking prevents corruption.
func ExportFlockPath(lockPath string) (func(), error) {
	return flockPath(lockPath)
}

// Exported functions.

// ExportIndexFileName exposes sourceSlug-based index naming so tests can
// locate a source's chunk index file.
func ExportIndexFileName(source string) string { return sourceSlug(source) + ".jsonl" }

// ExportLoadPriorRecords exposes loadPriorRecords for ingest unit tests.
func ExportLoadPriorRecords(indexPath string, deps IngestDeps) map[string]chunk.Record {
	return loadPriorRecords(indexPath, deps)
}

// ExportManifestLockFile returns the manifestLockFile constant so the concurrent
// regression test can compute the lock path without duplicating the constant.
func ExportManifestLockFile() string {
	return manifestLockFile
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

// ExportMergePhraseIntoUnion runs mergePhraseIntoUnion into a fresh union and
// returns the resulting item keys, so cli_test can assert which notes/chunks
// survived the per-phrase cap without naming the unexported matchedSetItem.
func ExportMergePhraseIntoUnion(noteHits []scoredCandidate, chunkHits []scoredChunk) []string {
	byKey := make(map[string]matchedSetItem)
	mergePhraseIntoUnion(noteHits, chunkHits, byKey)

	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}

	return keys
}

// ExportNewChunkResolvedItem builds a chunk-kind resolvedItem for band tests.
// notePath mirrors chunkNotePath's "source#anchor" form.
func ExportNewChunkResolvedItem(notePath string, score float32) resolvedItem {
	return resolvedItem{notePath: notePath, score: score, kind: chunkItemKind}
}

// ExportNewCompatibleSidecars zips parallel slices of notes and sidecars into
// the internal compatibleSidecar type, for loadAllVaultNotesMeta unit tests.
// notes[i] is paired with sidecars[i].
func ExportNewCompatibleSidecars(
	notes []vaultgraph.Note,
	sidecars []embed.Sidecar,
) []compatibleSidecar {
	out := make([]compatibleSidecar, len(notes))

	for i, note := range notes {
		out[i] = compatibleSidecar{note: note, sidecar: sidecars[i]}
	}

	return out
}

// ExportNewEmptyVaultNotesMeta constructs an AllVaultNotesMeta with
// empty (but non-nil) maps — the backward-compat no-op fixture.
func ExportNewEmptyVaultNotesMeta() AllVaultNotesMeta {
	return AllVaultNotesMeta{
		TermIndex:         make(map[string][]NominationEntry),
		SupersedesInverse: make(SupersedesInverse),
		ContentByBasename: make(map[string]string),
	}
}

// ExportNewNoteResolvedItem builds a note-kind resolvedItem for recency band
// tests. lastUsed and created are YYYY-MM-DD strings (empty = not set).
// kind is intentionally left blank — the zero value means "note" in the
// resolvedItem model (only chunkItemKind overrides content-derived detection).
func ExportNewNoteResolvedItem(notePath, lastUsed, created string) resolvedItem {
	return resolvedItem{notePath: notePath, lastUsed: lastUsed, created: created}
}

// ExportNewNoteResolvedItemWithContentAndProvenances builds a note-kind
// resolvedItem with explicit content, score, and provenances for nomination
// and ride-along unit tests.
func ExportNewNoteResolvedItemWithContentAndProvenances(
	notePath, content string,
	score float32,
	provenances []string,
) resolvedItem {
	return resolvedItem{
		notePath:    notePath,
		content:     content,
		score:       score,
		provenances: provenances,
	}
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

// ExportNewScoredCandidate builds a scoredCandidate (note hit) for tests.
func ExportNewScoredCandidate(notePath string, score, baseScore float32) scoredCandidate {
	return scoredCandidate{
		notePath:  notePath,
		basename:  basenameFromNotePath(notePath),
		score:     score,
		baseScore: baseScore,
	}
}

// ExportNewScoredCandidateWithContent builds a scoredCandidate with the content
// field set so vocab-exclusion tests can drive isVocabKind inside the pipeline.
func ExportNewScoredCandidateWithContent(
	notePath string,
	score, baseScore float32,
	content string,
) scoredCandidate {
	return scoredCandidate{
		notePath:  notePath,
		basename:  basenameFromNotePath(notePath),
		score:     score,
		baseScore: baseScore,
		content:   content,
	}
}

// ExportNewScoredChunk builds a scoredChunk for tests.
func ExportNewScoredChunk(rec chunk.Record, score float32) scoredChunk {
	return scoredChunk{record: rec, score: score}
}

// ExportNewScoredChunkWithIngestedAt builds a scoredChunk with IngestedAt set for recency tests.
func ExportNewScoredChunkWithIngestedAt(
	rec chunk.Record,
	score float32,
	ingestedAt time.Time,
) scoredChunk {
	rec.IngestedAt = ingestedAt

	return scoredChunk{record: rec, score: score}
}

// ExportNewVaultNotesMetaWithSupersedes builds an AllVaultNotesMeta that has
// only SupersedesInverse + ContentByBasename populated (no TermIndex).
// supersedersByNote maps each SUPERSEDER's basename to the list of entries it
// supersedes (same shape as BuildSupersedesInverse's input).
func ExportNewVaultNotesMetaWithSupersedes(
	supersedersByNote map[string][]supersedesEntry,
	contentByBasename map[string]string,
) AllVaultNotesMeta {
	return AllVaultNotesMeta{
		TermIndex:         make(map[string][]NominationEntry),
		SupersedesInverse: BuildSupersedesInverse(supersedersByNote),
		ContentByBasename: contentByBasename,
	}
}

// ExportNewVaultNotesMetaWithTerms builds an AllVaultNotesMeta that has only
// TermIndex populated (no supersedes data), for tag-nomination unit tests.
func ExportNewVaultNotesMetaWithTerms(terms map[string][]NominationEntry) AllVaultNotesMeta {
	return AllVaultNotesMeta{
		TermIndex:         terms,
		SupersedesInverse: make(SupersedesInverse),
		ContentByBasename: make(map[string]string),
	}
}

// ExportNewestChunkItems exposes newestChunkItems with the direct provenance.
func ExportNewestChunkItems(scored []scoredChunk, n int) []resolvedItem {
	return newestChunkItems(scored, n, provenanceDirect)
}

// ExportNoteClusterIDForPathFromPlain exercises noteClusterIDForPath with
// plain-data inputs, for coverage of the note-finding and cluster-lookup paths.
// memberPaths[i] is matched by notePath; memberIDs[c] is the set of member
// indices belonging to cluster c.
func ExportNoteClusterIDForPathFromPlain(
	notePath string,
	memberPaths []string,
	memberIDs [][]int,
) int {
	members := make([]matchedMember, len(memberPaths))

	for i, path := range memberPaths {
		members[i] = matchedMember{notePath: path}
	}

	return noteClusterIDForPath(
		notePath,
		matchedSet{members: members},
		clusterReport{memberIDs: memberIDs},
	)
}

// ExportOsManifestLock exposes osManifestLock for coverage of its MkdirAll-error branch.
func ExportOsManifestLock(dir string) (func(), error) {
	return osManifestLock(dir)
}

// ExportProvenanceRankFor exposes provenanceRankFor for whitebox testing.
func ExportProvenanceRankFor(role string) int { return provenanceRankFor(role) }

// ExportRecencyFloor exposes the floor field of recencyParams for tests.
func ExportRecencyFloor(p recencyParams) int { return p.floor }

// ExportRenderClustersTagNominations exercises the tag-nomination merging
// path in renderClusters. It builds a minimal single-member, single-cluster
// phrasedCluster with the given nominations and returns the CandidateL2s
// of the first emitted cluster. Returns nil when no clusters are emitted.
func ExportRenderClustersTagNominations(
	nominations map[int][]queryCandidateNote,
) []queryCandidateNote {
	vec := []float32{1, 0}

	member := matchedMember{
		notePath: "test.md",
		vector:   vec,
		sitVec:   vec,
		bodyVec:  vec,
		score:    0.9,
		content:  "# test member",
	}

	matched := matchedSet{members: []matchedMember{member}}

	report := clusterReport{
		autoK:           cluster.AutoKResult{K: 1, Centroids: [][]float32{vec}},
		memberIDs:       [][]int{{0}},
		representatives: []int{0},
		silhouettesByID: []float64{0},
	}

	pc := phrasedCluster{
		phrase:         "test phrase",
		report:         report,
		matched:        matched,
		tagNominations: nominations,
	}

	clusters := renderClusters([]phrasedCluster{pc})

	if len(clusters) == 0 {
		return nil
	}

	return clusters[0].CandidateL2s
}

// ExportRenderQueryPayloadBudget builds an aggregatedSummary from parallel
// kind/content slices, renders the YAML payload, and returns the encoded text.
// It exists so tests can assert the budget block (lazy_chunks, chunks_snippeted).
func ExportRenderQueryPayloadBudget(
	kinds, contents []string,
	lazyChunks bool,
	contentBudget int,
) (string, error) {
	resolved := make([]resolvedItem, len(kinds))
	for i := range kinds {
		resolved[i] = resolvedItem{kind: kinds[i], content: contents[i]}
	}

	var buf bytes.Buffer

	err := renderQueryPayload(&buf, aggregatedSummary{
		resolvedItems: resolved,
		lazyChunks:    lazyChunks,
		contentBudget: contentBudget,
	})

	return buf.String(), err
}

// ExportRenderQueryPayloadRefitPending renders a minimal payload with refitPending set,
// so tests can assert the refit_pending field's presence/omission.
func ExportRenderQueryPayloadRefitPending(pending bool) (string, error) {
	var buf bytes.Buffer

	err := renderQueryPayload(&buf, aggregatedSummary{refitPending: pending})

	return buf.String(), err
}

// ExportRenderQueryPayloadTagNominationBudget renders a minimal payload whose
// aggregatedSummary carries the given tag-nomination tally, so tests can assert
// the budget's tag_nominations_added / tag_nominations_dropped fields (and
// their omitempty behavior at zero).
func ExportRenderQueryPayloadTagNominationBudget(added, dropped int) (string, error) {
	var buf bytes.Buffer

	err := renderQueryPayload(&buf, aggregatedSummary{
		tagNomsAdded:   added,
		tagNomsDropped: dropped,
	})

	return buf.String(), err
}

// ExportResolveContentBudget exposes resolveContentBudget for tests.
func ExportResolveContentBudget(raw int) int {
	return resolveContentBudget(raw)
}

// ExportResolveRecentFill exposes resolveRecentFill for tests.
func ExportResolveRecentFill(raw int) int {
	return resolveRecentFill(raw)
}

// ExportResolvedItemBaseScore exposes the pre-decay baseScore field for
// activation-cutoff and band assertions (populated by Task 2.3).
func ExportResolvedItemBaseScore(item ExportResolvedItem) float32 { return item.baseScore }

// ExportResolvedItemContent exposes the unexported content field for
// ride-along insertion assertions.
func ExportResolvedItemContent(item resolvedItem) string { return item.content }

// ExportResolvedItemCreated exposes the created frontmatter date field.
func ExportResolvedItemCreated(item ExportResolvedItem) string { return item.created }

// ExportResolvedItemLastUsed exposes the LastUsed sidecar date field.
func ExportResolvedItemLastUsed(item ExportResolvedItem) string { return item.lastUsed }

// ExportResolvedItemLess exposes resolvedItemLess for whitebox testing.
func ExportResolvedItemLess(a, b resolvedItem) bool { return resolvedItemLess(a, b) }

// ExportResolvedItemPath exposes the unexported notePath field for assertions.
func ExportResolvedItemPath(item ExportResolvedItem) string { return item.notePath }

// ExportResolvedItemProvenances exposes the unexported provenances field for
// ride-along insertion assertions.
func ExportResolvedItemProvenances(item resolvedItem) []string { return item.provenances }

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

// ExportSnippet exposes the snippet helper for the content-cap tests.
func ExportSnippet(content string) string {
	return snippet(content)
}
