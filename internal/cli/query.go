package cli

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cluster"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// QueryArgs holds parsed flags for `engram query`.
type QueryArgs struct {
	Phrases   []string `targ:"flag,name=phrase,desc=query phrase (repeatable)"`
	VaultPath string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root"`
	ChunksDir string   `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks); chunks compete in the same ranking as notes"` //nolint:lll // single unbreakable struct-tag string
	Limit     int      `targ:"flag,name=limit,desc=max number of items to return (default 20)"`
	Project   string   `targ:"flag,name=project,desc=restrict items to notes with matching project: field (optional)"`
	// ContentBudget caps how many chunk items (in rank order) render with full
	// content; later chunks get a one-line snippet. 0 = unlimited. Notes are
	// never capped. env= lets the recall sweep inject the cap without a skill edit.
	ContentBudget int `targ:"flag,name=content-budget,env=ENGRAM_CONTENT_BUDGET,desc=max chunk items with full content (0=unlimited); later chunks get a snippet"` //nolint:lll // single unbreakable struct-tag string
}

// QueryDeps holds injected dependencies for the query command.
//
// LogWarning, when non-nil, receives non-fatal advisories (e.g. sidecars
// dropped for a stale embedding model). It mirrors LearnDeps.LogWarning so
// both commands share the production logWarningToStderrf hook.
type QueryDeps struct {
	Scan       func(vault string) ([]vaultgraph.Note, error)
	Read       func(path string) ([]byte, error)
	Embedder   embed.Embedder
	LogWarning func(format string, args ...any)
	// ListChunkIndexes returns the .jsonl chunk index files under a chunks
	// dir. Nil (or an empty ChunksDir) disables chunk-space merging.
	ListChunkIndexes func(dir string) ([]string, error)
	// Now supplies the query-time clock for recency (DI; production = time.Now).
	// When nil, recency re-rank is skipped and pure cosine ranking applies.
	Now func() time.Time
}

// RunQuery embeds each query phrase, unions the matched notes and chunks, clusters
// once, and emits per-cluster candidate_l2s.
func RunQuery(ctx context.Context, args QueryArgs, deps QueryDeps, stdout io.Writer) error {
	validationErr := validateQueryArgs(args)
	if validationErr != nil {
		return validationErr
	}

	limit := args.Limit
	if limit == 0 {
		limit = defaultQueryLimit
	}

	notes, scanErr := deps.Scan(args.VaultPath)
	if scanErr != nil {
		return fmt.Errorf("query: scan: %w", scanErr)
	}

	modelID := deps.Embedder.ModelID()
	loaded := loadCompatibleSidecars(notes, args.VaultPath, deps.Read, modelID)
	hits := loaded.hits

	warnModelMismatch(deps.LogWarning, loaded, modelID)
	warnOldSchema(deps.LogWarning, loaded)

	if len(notes) > 0 && len(hits) == 0 {
		return errQueryNoEmbeddings
	}

	return runQuery(ctx, args, notes, hits, limit, deps, stdout)
}

// unexported constants.
const (
	// candidateNoteK is the minimum number of candidate notes to nominate per
	// cluster. The recall skill reads all K candidates to judge coverage;
	// generous nomination costs nothing (recall is the binary's job,
	// precision is the agent's). Raised from 3→5 in recall-v2 Phase 0.
	candidateNoteK = 5
	// chunkItemKind tags unified-ranking items sourced from the chunk index
	// (vs notes, whose kind derives from frontmatter).
	chunkItemKind          = "chunk"
	clusterMaxK            = 7
	clusterMinK            = 2
	clusterSilhouetteFloor = 0.10
	defaultQueryLimit      = 20
	// matchPhraseLimit is the maximum number of candidates (notes + chunks
	// combined) taken per phrase before union across phrases. Bounds
	// clustering at O(matchSetCap^2) regardless of corpus size.
	matchPhraseLimit = 30
	// matchRelevanceFloor is the minimum raw cosine (baseScore, pre-decay)
	// an item must have to enter the matched set. Applied BEFORE recency
	// bias so topically-perfect-but-old items still activate (vault note 53).
	matchRelevanceFloor = float32(0.25)
	// matchSetCap is the hard cap on the union of all per-phrase matched
	// sets fed to clustering. 10 phrases × matchPhraseLimit = 300 worst-case.
	matchSetCap              = 300
	provenanceClusterRep     = "cluster_rep"
	provenanceDirect         = "direct"
	provenanceRankClusterRep = 2
	provenanceRankDirect     = 3
	// provenanceRecent tags un-clustered recency-channel chunks (Channel 2,
	// Phase 2). Items carrying this role appear in items[] but in NO cluster's
	// members[], so the skill can render a separate "recent activity" block.
	provenanceRecent = "recent"
	// recentFillChunks is the number of newest-by-IngestedAt chunks appended
	// to the recency channel (Channel 2, Phase 2). Defined here in Phase 0
	// so it is available as a named constant before Phase 2 lands.
	recentFillChunks = 200
	// singleClusterPhrase is the empty phrase tag on the single synthesis
	// cluster, which spans all seed phrases rather than any one of them.
	singleClusterPhrase = ""
	// singletonClusterSilhouette is the silhouette reported for the K=0->one
	// synthesis fallback cluster. Silhouette is undefined for a single
	// cluster, so zero stands in.
	singletonClusterSilhouette = 0.0
	snippetMaxRunes            = 160
	unknownKind                = "unknown"
)

// unexported variables.
var (
	errQueryEmptyString  = errors.New("query: empty query string")
	errQueryNoEmbeddings = errors.New(
		"query: vault has notes but no current-model embeddings; run `engram embed apply --all`",
	)
	// projectLineRE matches a `project: <slug>` line in YAML frontmatter,
	// anchored to start-of-line so body text can't false-match. Slug shape
	// mirrors the write-side validation: [a-z0-9-]+.
	projectLineRE = regexp.MustCompile(`(?m)^project:\s*([a-z0-9-]+)\s*$`)
	// wikilinkRE matches `[[target]]` and `[[target|display]]`.
	// Used by stripWikilinks to remove pointer syntax from the
	// rendered items.content per the spike spec — engram returns
	// the relevant set in `items`; inline pointers are noise.
	wikilinkRE = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)
)

// aggregatedSummary holds the merged result of running RunQuery across
// multiple phrases.
type aggregatedSummary struct {
	phrases        []string
	resolvedItems  []resolvedItem
	phraseClusters []phrasedCluster
	outgoing       map[string][]string
	totalNotes     int
	withEmbeddings int
	limit          int
	contentBudget  int
}

// candidateNoteIndex holds the note paths and BOTH sidecar vectors for the
// note members of a single cluster — used for per-cluster nearest-note
// nomination by max(situation,body) cosine against the cluster centroid.
type candidateNoteIndex struct {
	paths []string
	sit   [][]float32
	body  [][]float32
}

// clusterReport collects the AutoK output for the payload-rendering
// stage. Empty Members means clustering was skipped or yielded nothing.
type clusterReport struct {
	autoK           cluster.AutoKResult
	memberIDs       [][]int // memberIDs[c] = matchedMember indices in cluster c
	representatives []int   // representatives[c] = matchedMember index for cluster c
	silhouettesByID []float64
}

// compatibleSidecar bundles a note with its already-loaded current-model
// sidecar — the result of loadCompatibleSidecars's filtering pass. Carrying
// the parsed sidecar forward avoids re-reading it during scoring.
type compatibleSidecar struct {
	note    vaultgraph.Note
	sidecar embed.Sidecar
}

// matchedMember bundles a node's basename, vault-relative path,
// sidecar vector, query-similarity score, and (optionally) cached body.
// kind overrides content-derived kind detection for chunk members.
// sitVec and bodyVec carry both sidecar axes for within-cluster note
// nomination (clusterNoteIndexFromMembers), so eitherAxisCosine can pick
// the stronger axis against the centroid rather than the query.
type matchedMember struct {
	basename string
	notePath string
	vector   []float32 // winning coord (best-of sit/body vs queryVec)
	sitVec   []float32 // situation-axis vector from sidecar
	bodyVec  []float32 // body-axis vector from sidecar
	score    float32
	content  string
	kind     string // empty = note; chunkItemKind for chunks
}

// matchedSet holds the unified set of notes and chunks that matched the query
// phrases. Members are the inputs to the single AutoK clustering pass.
type matchedSet struct {
	members []matchedMember
}

// matchedSetItem is the unified per-phrase ranking element used by
// buildMatchedSetFromPhrases. It holds the common fields needed for
// dedup/floor/cap and carries either a note or a chunk.
type matchedSetItem struct {
	key       string  // notePath for notes; source#anchor for chunks
	score     float32 // recency-biased ranking score
	baseScore float32 // pre-decay raw cosine; used for the relevance floor
	isChunk   bool
	note      scoredCandidate
	chunk     scoredChunk
}

// phrasedCluster pairs a cluster report with the phrase that produced it,
// so the payload can tag each cluster with its originating query phrase.
type phrasedCluster struct {
	phrase  string
	report  clusterReport
	matched matchedSet
}

// queryBudget reports the totals visible to the caller per the YAML schema.
type queryBudget struct {
	PhrasesQueried       int `yaml:"phrases_queried"`
	TotalNotes           int `yaml:"total_notes"`
	WithEmbeddings       int `yaml:"with_embeddings"`
	ClustersFound        int `yaml:"clusters_found"`
	DirectHitsReturned   int `yaml:"direct_hits_returned"`
	ItemsWithFullContent int `yaml:"items_with_full_content"`
	Limit                int `yaml:"limit"`
	ContentBudget        int `yaml:"content_budget"`
	ChunksSnippeted      int `yaml:"chunks_snippeted"`
}

// queryCandidateNote is one candidate note for a cluster. The binary emits
// top-K by centroid cosine (K >= candidateNoteK); the recall skill judges
// coverage — no cosine-band decision happens in the binary.
type queryCandidateNote struct {
	Path   string  `yaml:"path"`
	Cosine float32 `yaml:"cosine"`
}

// queryCluster is the cluster shape in the payload.
type queryCluster struct {
	ID           int                  `yaml:"id"`
	Phrase       string               `yaml:"phrase"`
	Size         int                  `yaml:"size"`
	Silhouette   float64              `yaml:"silhouette"`
	Members      []queryClusterMember `yaml:"members"`
	CandidateL2s []queryCandidateNote `yaml:"candidate_l2s,omitempty"`
}

// queryClusterMember is the per-member shape in clusters.members.
type queryClusterMember struct {
	Path             string  `yaml:"path"`
	Score            float32 `yaml:"score"`
	IsRepresentative bool    `yaml:"is_representative"`
}

// queryItem is the rendered item shape per the resolved-payload spec.
// ClusterID and InDegree use *int so YAML omits them when nil per
// the spec contract (set only when the provenance role is present).
// OutboundLinks lists the note's authored wikilink target basenames (the
// fence-aware graph parser's output) so the recall skill can follow links
// to adjacent notes via `engram show <basename>` without a separate query.
type queryItem struct {
	Path          string   `yaml:"path"`
	Kind          string   `yaml:"kind"`
	Score         float32  `yaml:"score"`
	Provenances   []string `yaml:"provenances"`
	ClusterID     *int     `yaml:"cluster_id,omitempty"`
	InDegree      *int     `yaml:"in_degree,omitempty"`
	OutboundLinks []string `yaml:"outbound_links,omitempty"`
	Content       string   `yaml:"content,omitempty"`
}

// queryPayload is the top-level YAML document.
type queryPayload struct {
	Version  int            `yaml:"version"`
	Phrases  []string       `yaml:"phrases"`
	Items    []queryItem    `yaml:"items"`
	Clusters []queryCluster `yaml:"clusters"`
	Budget   queryBudget    `yaml:"budget"`
}

// resolvedItem is the working shape for the items[] section before
// rendering — gathers provenance roles, scores, and optional metadata.
type resolvedItem struct {
	notePath    string
	content     string
	score       float32
	baseScore   float32 // pre-decay cosine; for activation cutoff + Phase 4 band
	lastUsed    string  // YYYY-MM-DD from Sidecar.LastUsed (notes only)
	created     string  // YYYY-MM-DD from note frontmatter (notes only)
	provenances []string
	clusterID   *int
	inDegree    *int
	// kind overrides content-derived kind detection when set (chunk items).
	kind string
}

// scoredCandidate aggregates one note's match against the query vector.
// coord is the WINNING vector — the one of the note's two axes that scored
// highest — used as the note's clustering coordinate (a note is clustered by
// the vector that matched it). sitVec and bodyVec carry both sidecar axes so
// within-cluster L2 nomination can apply eitherAxisCosine against the
// cluster centroid (the winning coord vs queryVec is not the same as the
// better axis vs the centroid — Phase 3 nominal fix).
type scoredCandidate struct {
	notePath  string
	basename  string
	score     float32
	baseScore float32 // pre-decay cosine; preserved through the pipeline
	lastUsed  string  // YYYY-MM-DD from Sidecar.LastUsed
	created   string  // YYYY-MM-DD from note frontmatter
	coord     []float32
	sitVec    []float32 // SituationVector from sidecar
	bodyVec   []float32 // BodyVector from sidecar
	content   string
}

// sidecarLoadResult bundles the compatible-sidecar hits with summaries of
// sidecars dropped for a stale embedding model (mismatch fields) and for an
// old sidecar schema (oldSchemaCount). Tracking the two reasons separately
// lets RunQuery emit a distinct, non-contradictory advisory for each at the
// command edge instead of doing I/O inside the loader.
type sidecarLoadResult struct {
	hits               []compatibleSidecar
	mismatchedCount    int
	mismatchedModelIDs []string
	oldSchemaCount     int
}

// addMatchedChunksToMatchedSet adds matched chunks to the matched set as members
// (so they cluster with notes — D1) and returns the parallel resolvedItem
// slice for items[]. Chunks are sorted by score desc so items[] presents them
// newest-most-relevant first within the chunk block.
func addMatchedChunksToMatchedSet(chunkUnion []scoredChunk, matchSet *matchedSet) []resolvedItem {
	sort.SliceStable(chunkUnion, func(i, j int) bool {
		return chunkUnion[i].score > chunkUnion[j].score
	})

	chunkItems := make([]resolvedItem, 0, len(chunkUnion))

	for _, scored := range chunkUnion {
		path := chunkNotePath(scored.record)
		matchSet.members = append(matchSet.members, matchedMember{
			basename: path,
			notePath: path,
			content:  scored.record.Text,
			vector:   scored.record.Vector,
			score:    scored.score,
			kind:     chunkItemKind,
		})
		chunkItems = append(chunkItems, resolvedItem{
			notePath:    path,
			content:     scored.record.Text,
			score:       scored.score,
			baseScore:   scored.baseScore,
			provenances: []string{provenanceDirect},
			kind:        chunkItemKind,
		})
	}

	return chunkItems
}

// appendUniqueProvenance adds role to item.provenances iff not already present.
func appendUniqueProvenance(item *resolvedItem, role string) {
	if slices.Contains(item.provenances, role) {
		return
	}

	item.provenances = append(item.provenances, role)
}

// applyFloorAndCap filters matched set items by the relevance floor on
// baseScore, sorts by score desc (highest-scoring survive the cap), then caps
// at matchSetCap. Returns items sorted by key for deterministic clustering.
func applyFloorAndCap(byKey map[string]matchedSetItem) []matchedSetItem {
	matched := make([]matchedSetItem, 0, len(byKey))

	for _, item := range byKey {
		if item.baseScore >= matchRelevanceFloor {
			matched = append(matched, item)
		}
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].score != matched[j].score {
			return matched[i].score > matched[j].score
		}

		return matched[i].key < matched[j].key
	})

	if len(matched) > matchSetCap {
		matched = matched[:matchSetCap]
	}

	// Final sort by key for deterministic clustering order.
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].key < matched[j].key
	})

	return matched
}

// applyProjectFilter drops items whose frontmatter project: field doesn't
// match the requested slug. Empty project is a no-op (returns items
// unchanged). Items with no loaded content cannot be verified and are
// dropped when a non-empty project is specified. The filter only
// affects which matched items are emitted, not which ones were scored.
func applyProjectFilter(items []resolvedItem, project string) []resolvedItem {
	if project == "" {
		return items
	}

	out := make([]resolvedItem, 0, len(items))

	for _, item := range items {
		if itemMatchesProject(item, project) {
			out = append(out, item)
		}
	}

	return out
}

// basenameFromNotePath strips the directory and ".md" extension from a
// vault-relative note path, yielding the graph-node basename key.
func basenameFromNotePath(notePath string) string {
	return strings.TrimSuffix(filepath.Base(notePath), ".md")
}

// bestVector scores a sidecar against the query by the stronger of its two
// axes and returns that score with the WINNING vector — the coordinate the
// note is positioned by for clustering (per the lazy-L2 design: a note is
// clustered by the vector that matched it).
func bestVector(queryVec []float32, sidecar embed.Sidecar) (float32, []float32) {
	situationScore := embed.Cosine(queryVec, sidecar.SituationVector)
	bodyScore := embed.Cosine(queryVec, sidecar.BodyVector)

	if situationScore >= bodyScore {
		return situationScore, sidecar.SituationVector
	}

	return bodyScore, sidecar.BodyVector
}

// breakRepresentativeTie returns whichever of two member indices wins
// the secondary tiebreakers: higher direct-hit score, then lexicographic
// notePath ascending.
func breakRepresentativeTie(matchSet matchedSet, a, b int) int {
	memberA := matchSet.members[a]
	memberB := matchSet.members[b]

	switch {
	case memberA.score > memberB.score:
		return a
	case memberB.score > memberA.score:
		return b
	case memberA.notePath < memberB.notePath:
		return a
	default:
		return b
	}
}

// buildMatchedSet converts the matched note candidates into a matchedSet,
// carrying each hit's winning-vector coordinate forward for clustering.
func buildMatchedSet(union []scoredCandidate) matchedSet {
	members := make([]matchedMember, 0, len(union))

	for _, hit := range union {
		members = append(members, matchedMember{
			basename: hit.basename,
			notePath: hit.notePath,
			vector:   hit.coord,
			sitVec:   hit.sitVec,
			bodyVec:  hit.bodyVec,
			score:    hit.score,
			content:  hit.content,
		})
	}

	return matchedSet{members: members}
}

// buildMatchedSetFromPhrases performs per-phrase unified note+chunk matching for
// the query path. For each phrase it embeds once, scores notes and
// chunks with recency bias, merges into one list (top-matchPhraseLimit=30 per
// phrase), then unions across phrases with dedup, relevance floor, and cap.
// Returns matched notes and chunks as separate slices sorted by key.
func buildMatchedSetFromPhrases(
	ctx context.Context,
	phrases []string,
	hits []compatibleSidecar,
	records []chunk.Record,
	vault string,
	now time.Time,
	maxTurnBySrc map[string]int,
	deps QueryDeps,
) ([]scoredCandidate, []scoredChunk, error) {
	recency := defaultRecencyParams()
	byKey := make(map[string]matchedSetItem)

	for _, phrase := range phrases {
		queryVec, embedErr := deps.Embedder.Embed(ctx, phrase)
		if embedErr != nil {
			return nil, nil, fmt.Errorf("query: embed: %w", embedErr)
		}

		noteHits := rankCandidates(hits, vault, deps.Read, queryVec, now)
		chunkHits := scoreChunkForPhrase(queryVec, records, now, maxTurnBySrc, recency)

		mergePhraseIntoUnion(noteHits, chunkHits, byKey)
	}

	matched := applyFloorAndCap(byKey)

	notes, chunks := splitMatchedSet(matched)

	return notes, chunks, nil
}

// buildRecentFillItems returns the un-clustered recency-channel items for
// the query path (Channel 2). It selects the N newest
// chunks by IngestedAt (using newestChunkItems for ordering), deduplicates
// them against the already-matched chunk paths, and tags the survivors with
// provenanceRecent. These items appear in items[] only — NOT in matched.members
// and therefore NOT in any cluster's members[].
func buildRecentFillItems(
	allRecords []chunk.Record,
	matchedChunks []scoredChunk,
	n int,
) []resolvedItem {
	if n <= 0 {
		return nil
	}

	// Build a set of paths already in the matched set so we can dedup.
	matchedPaths := make(map[string]bool, len(matchedChunks))
	for _, scored := range matchedChunks {
		matchedPaths[chunkNotePath(scored.record)] = true
	}

	// Convert all records to scoredChunk (score=0; ordering is by IngestedAt,
	// not by cosine, so the score field is unused here).
	all := make([]scoredChunk, 0, len(allRecords))
	for _, rec := range allRecords {
		all = append(all, scoredChunk{record: rec, score: 0})
	}

	// Use newestChunkItems to get the N newest in IngestedAt order, tagged
	// with the recent provenance. We may need more than n to account for
	// dedup, so fetch all and slice.
	newest := newestChunkItems(all, len(all), provenanceRecent)

	// Filter out chunks already in the matched set and take the top n.
	out := make([]resolvedItem, 0, n)
	for _, item := range newest {
		if len(out) >= n {
			break
		}

		if matchedPaths[item.notePath] {
			continue
		}

		out = append(out, item)
	}

	return out
}

// capChunkContent keeps the first `budget` chunk items (in rank order) at full
// content and replaces later chunks' content with a snippet. Note items are
// never capped. budget <= 0 disables capping. Returns the (mutated) items and
// the number of chunks snippeted.
func capChunkContent(items []queryItem, budget int) ([]queryItem, int) {
	if budget <= 0 {
		return items, 0
	}

	chunksSeen := 0
	snipped := 0

	for i := range items {
		if items[i].Kind != chunkItemKind {
			continue
		}

		chunksSeen++

		if chunksSeen > budget && items[i].Content != "" {
			items[i].Content = snippet(items[i].Content)
			snipped++
		}
	}

	return items, snipped
}

// chunksConfigured reports whether a chunk index is wired into this run
// (non-empty chunks dir and a list function available).
func chunksConfigured(args QueryArgs, deps QueryDeps) bool {
	return args.ChunksDir != "" && deps.ListChunkIndexes != nil
}

// clusterMatchedSet clusters the matched set exactly once. On
// AutoK returning K==0 (no split beats the silhouette floor) or an error, falls
// back to a SINGLE cluster of all members so a non-empty matched set always yields
// >=1 cluster. An empty matched set yields an empty (K==0) report.
func clusterMatchedSet(matchSet matchedSet, query string) clusterReport {
	if len(matchSet.members) == 0 {
		return clusterReport{}
	}

	vectors := make([][]float32, len(matchSet.members))
	for i, member := range matchSet.members {
		vectors[i] = member.vector
	}

	seed := seedFromQuery(query)

	autoK, err := cluster.AutoK(vectors, clusterMinK, clusterMaxK, clusterSilhouetteFloor, seed)
	if err != nil || autoK.K == 0 {
		return singleClusterReport(matchSet, vectors)
	}

	memberIDs := make([][]int, autoK.K)
	for i := range memberIDs {
		memberIDs[i] = make([]int, 0)
	}

	for i, c := range autoK.Assignments {
		memberIDs[c] = append(memberIDs[c], i)
	}

	representatives := make([]int, autoK.K)

	for c := range autoK.K {
		representatives[c] = pickRepresentative(matchSet, memberIDs[c], autoK.Centroids[c])
	}

	return clusterReport{
		autoK:           autoK,
		memberIDs:       memberIDs,
		representatives: representatives,
		silhouettesByID: perClusterMeanSilhouette(vectors, autoK.Assignments, autoK.K),
	}
}

// clusterNoteIndexFromMembers builds a candidateNoteIndex from the note members of a
// single cluster (Phase 3, within-cluster nomination). Only matched members
// whose kind is NOT chunkItemKind are included; chunk members are skipped.
// An empty index (no note members) yields an empty candidateNoteIndex{}.
func clusterNoteIndexFromMembers(matchSet matchedSet, memberIndices []int) candidateNoteIndex {
	idx := candidateNoteIndex{}

	for _, i := range memberIndices {
		member := matchSet.members[i]
		if member.kind == chunkItemKind {
			continue
		}

		idx.paths = append(idx.paths, member.notePath)
		idx.sit = append(idx.sit, member.sitVec)
		idx.body = append(idx.body, member.bodyVec)
	}

	return idx
}

// collectClusterMembers gathers per-cluster member rows in score-desc
// order, marking the representative. If the elected representative is somehow
// absent from the member list, the highest-scoring member is promoted so
// every non-empty cluster still reports exactly one representative.
func collectClusterMembers(
	matchSet matchedSet,
	report clusterReport,
	clusterID int,
) []queryClusterMember {
	memberIndices := make([]int, len(report.memberIDs[clusterID]))
	copy(memberIndices, report.memberIDs[clusterID])

	sort.SliceStable(memberIndices, func(i, j int) bool {
		return matchSet.members[memberIndices[i]].score > matchSet.members[memberIndices[j]].score
	})

	repIdx := report.representatives[clusterID]
	members := make([]queryClusterMember, 0, len(memberIndices))
	repRetained := false

	for _, idx := range memberIndices {
		member := matchSet.members[idx]
		isRep := idx == repIdx

		if isRep {
			repRetained = true
		}

		members = append(members, queryClusterMember{
			Path:             member.notePath,
			Score:            member.score,
			IsRepresentative: isRep,
		})
	}

	if !repRetained && len(members) > 0 {
		members[0].IsRepresentative = true
	}

	return members
}

// countItemsWithContent reports how many rendered items carry a
// non-empty Content field. Used to populate `items_with_full_content`.
func countItemsWithContent(items []queryItem) int {
	count := 0

	for _, item := range items {
		if item.Content != "" {
			count++
		}
	}

	return count
}

// eitherAxisCosine returns the stronger of the situation- and body-axis cosines
// between centroid and a note's two vectors (the "either axis" gate).
func eitherAxisCosine(centroid, sit, body []float32) float32 {
	sim := embed.Cosine(centroid, sit)
	if bodySim := embed.Cosine(centroid, body); bodySim > sim {
		sim = bodySim
	}

	return sim
}

// itemMatchesProject scans the item's loaded content's frontmatter for a
// project: <slug> line matching the requested project. Returns false when
// content is missing or when the frontmatter block is malformed.
func itemMatchesProject(item resolvedItem, project string) bool {
	if item.content == "" {
		return false
	}

	const delim = "---\n"

	body := strings.TrimPrefix(item.content, delim)

	end := strings.Index(body, "\n"+delim)
	if end < 0 {
		return false
	}

	front := body[:end+1]

	match := projectLineRE.FindStringSubmatch(front)

	return len(match) == 2 && match[1] == project
}

// kindFromContent reads the frontmatter type field to label the item.
// Falls back to "unknown" — engram's other readers (notes, recall)
// already tolerate this case.
func kindFromContent(content string) string {
	const (
		maxScan        = 256
		typeLineMarker = "\ntype: "
		minViableLen   = len("---\ntype: x\n")
	)

	if len(content) < minViableLen {
		return unknownKind
	}

	scan := content
	if len(scan) > maxScan {
		scan = scan[:maxScan]
	}

	_, after, ok := strings.Cut(scan, typeLineMarker)
	if !ok {
		return unknownKind
	}

	kind, _, ok := strings.Cut(after, "\n")
	if !ok {
		return unknownKind
	}

	return kind
}

// loadClusterChunkRecords loads chunk records for the query path.
// Returns an empty slice (not an error) when chunks are not configured.
func loadClusterChunkRecords(args QueryArgs, deps QueryDeps) ([]chunk.Record, error) {
	if !chunksConfigured(args, deps) {
		return nil, nil
	}

	records, err := loadChunkRecords(args.ChunksDir, ChunkQueryDeps{
		ListIndexes: deps.ListChunkIndexes, ReadFile: deps.Read, Embedder: deps.Embedder,
	})
	if err != nil {
		return nil, err
	}

	return records, nil
}

// loadCompatibleSidecars reads every note's sidecar once, parses it, and
// returns only those whose EmbeddingModelID matches the active model.
// Missing or malformed sidecars are silently skipped. Sidecars dropped for
// a stale embedding model are also skipped but recorded in the result's
// mismatch fields so RunQuery can warn (M4) instead of emptying recall in
// silence — the all-mismatch case is still surfaced by RunQuery's guard.
func loadCompatibleSidecars(
	notes []vaultgraph.Note,
	vault string,
	read func(string) ([]byte, error),
	modelID string,
) sidecarLoadResult {
	hits := make([]compatibleSidecar, 0, len(notes))
	mismatchedCount := 0
	mismatchedIDs := map[string]struct{}{}
	oldSchemaCount := 0

	for _, note := range notes {
		notePath := pathOf(note.Basename)
		scFull := filepath.Join(vault, embed.SidecarPath(notePath))

		scBytes, readErr := read(scFull)
		if readErr != nil {
			continue
		}

		sidecar, parseErr := embed.UnmarshalSidecar(scBytes)
		if parseErr != nil {
			if errors.Is(parseErr, embed.ErrSchemaVersion) {
				oldSchemaCount++
			}

			continue
		}

		if sidecar.EmbeddingModelID != modelID {
			mismatchedCount++
			mismatchedIDs[sidecar.EmbeddingModelID] = struct{}{}

			continue
		}

		hits = append(hits, compatibleSidecar{note: note, sidecar: sidecar})
	}

	ids := make([]string, 0, len(mismatchedIDs))
	for id := range mismatchedIDs {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	return sidecarLoadResult{
		hits:               hits,
		mismatchedCount:    mismatchedCount,
		mismatchedModelIDs: ids,
		oldSchemaCount:     oldSchemaCount,
	}
}

// maxProvenanceRank returns the highest priority value among the roles
// listed in provenances.
func maxProvenanceRank(provenances []string) int {
	best := 0

	for _, role := range provenances {
		if rank := provenanceRankFor(role); rank > best {
			best = rank
		}
	}

	return best
}

// meanVector returns the element-wise mean of the given vectors. Callers
// guarantee a non-empty, uniformly-dimensioned input (the union members all
// carry same-model sidecar vectors).
func meanVector(vectors [][]float32) []float32 {
	dims := len(vectors[0])
	sums := make([]float64, dims)

	for _, vec := range vectors {
		for dim := range dims {
			sums[dim] += float64(vec[dim])
		}
	}

	mean := make([]float32, dims)
	for dim := range dims {
		mean[dim] = float32(sums[dim] / float64(len(vectors)))
	}

	return mean
}

// mergeClusterReps annotates representatives with provenance + cluster_id,
// adding new entries when a rep is not already a direct hit.
func mergeClusterReps(
	matchSet matchedSet,
	clusters clusterReport,
	byBasename map[string]*resolvedItem,
) {
	for clusterID, memberIdx := range clusters.representatives {
		if memberIdx < 0 || memberIdx >= len(matchSet.members) {
			continue
		}

		member := matchSet.members[memberIdx]

		resolved := byBasename[member.basename]
		if resolved == nil {
			resolved = &resolvedItem{
				notePath: member.notePath,
				content:  member.content,
				score:    member.score,
			}
			byBasename[member.basename] = resolved
		}

		appendUniqueProvenance(resolved, provenanceClusterRep)

		clusterIDCopy := clusterID
		resolved.clusterID = &clusterIDCopy
	}
}

// mergePhraseIntoUnion builds a unified per-phrase ranked list from noteHits
// and chunkHits, takes the top-matchPhraseLimit (30), then merges into byKey
// (dedup by key, keeping the max-score item).
func mergePhraseIntoUnion(
	noteHits []scoredCandidate,
	chunkHits []scoredChunk,
	byKey map[string]matchedSetItem,
) {
	perPhrase := make([]matchedSetItem, 0, len(noteHits)+len(chunkHits))

	for _, note := range noteHits {
		perPhrase = append(perPhrase, matchedSetItem{
			key:       note.notePath,
			score:     note.score,
			baseScore: note.baseScore,
			isChunk:   false,
			note:      note,
		})
	}

	for _, chunkHit := range chunkHits {
		perPhrase = append(perPhrase, matchedSetItem{
			key:       chunkNotePath(chunkHit.record),
			score:     chunkHit.score,
			baseScore: chunkHit.baseScore,
			isChunk:   true,
			chunk:     chunkHit,
		})
	}

	sort.SliceStable(perPhrase, func(i, j int) bool {
		return perPhrase[i].score > perPhrase[j].score
	})

	if len(perPhrase) > matchPhraseLimit {
		perPhrase = perPhrase[:matchPhraseLimit]
	}

	for _, item := range perPhrase {
		existing, ok := byKey[item.key]
		if !ok || item.score > existing.score {
			byKey[item.key] = item
		}
	}
}

// mergeProvenances builds the resolved item list per F7's rules:
// items = direct hits ∪ cluster reps, deduped by basename,
// each item carrying every applicable provenance role + metadata.
//
// Ordering: provenance count desc → highest-priority provenance desc →
// score desc.
//
// Bodies for non-direct entries are looked up from the matched member's
// `content` field (loaded if it was a direct hit); non-direct cluster reps
// need a separate fill pass via a deps.Read callback that happens in the
// renderer to keep this stage pure.
func mergeProvenances(
	directHits []scoredCandidate,
	matchSet matchedSet,
	clusters clusterReport,
) []resolvedItem {
	byBasename := make(map[string]*resolvedItem)

	for _, hit := range directHits {
		resolved := byBasename[hit.basename]
		if resolved == nil {
			resolved = &resolvedItem{
				notePath:  hit.notePath,
				content:   hit.content,
				score:     hit.score,
				baseScore: hit.baseScore,
				lastUsed:  hit.lastUsed,
				created:   hit.created,
			}
			byBasename[hit.basename] = resolved
		}

		appendUniqueProvenance(resolved, provenanceDirect)
	}

	mergeClusterReps(matchSet, clusters, byBasename)

	// Drain the map in basename-sorted order so the pre-sort slice
	// shape is deterministic. The final sort below is stable, so any
	// items with identical comparator values preserve this lexicographic
	// secondary order.
	basenames := make([]string, 0, len(byBasename))
	for name := range byBasename {
		basenames = append(basenames, name)
	}

	sort.Strings(basenames)

	items := make([]resolvedItem, 0, len(byBasename))
	for _, name := range basenames {
		item, ok := byBasename[name]
		if !ok || item == nil {
			continue
		}

		items = append(items, *item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		return resolvedItemLess(items[i], items[j])
	})

	return items
}

// newOsQueryDeps wires the production scan + read for the query command.
func newOsQueryDeps() QueryDeps {
	embedDeps := newOsEmbedDeps()

	return QueryDeps{
		Scan:             embedDeps.Scan,
		Read:             embedDeps.Read,
		Embedder:         embedDeps.Embedder,
		LogWarning:       logWarningToStderrf,
		ListChunkIndexes: listJSONLIndexes,
		Now:              time.Now,
	}
}

// outgoingByBasename indexes each scanned note's authored wikilink targets by
// its basename. Used to attach outbound-link basenames to rendered items.
func outgoingByBasename(notes []vaultgraph.Note) map[string][]string {
	out := make(map[string][]string, len(notes))
	for _, note := range notes {
		out[note.Basename] = note.Outgoing
	}

	return out
}

// perClusterMeanSilhouette returns one mean silhouette score per cluster
// by recomputing the per-point silhouettes and averaging within cluster.
// Mirrors standard silhouette analysis tooling.
func perClusterMeanSilhouette(vectors [][]float32, assignments []int, clusterCount int) []float64 {
	scoresByCluster := make([][]float64, clusterCount)
	for clusterIdx := range scoresByCluster {
		scoresByCluster[clusterIdx] = make([]float64, 0)
	}

	// Reuse cluster.Silhouette's logic by computing per-point. We
	// rebuild member lists once here for efficiency.
	members := make([][]int, clusterCount)

	for i, clusterIdx := range assignments {
		members[clusterIdx] = append(members[clusterIdx], i)
	}

	for i, vec := range vectors {
		own := assignments[i]
		score := cluster.PointSilhouette(vec, vectors, members, own, i)
		scoresByCluster[own] = append(scoresByCluster[own], score)
	}

	means := make([]float64, clusterCount)

	for clusterIdx, scores := range scoresByCluster {
		if len(scores) == 0 {
			continue
		}

		var total float64

		for _, score := range scores {
			total += score
		}

		means[clusterIdx] = total / float64(len(scores))
	}

	return means
}

// pickRepresentative returns the matched-member index closest to the
// centroid by cosine distance. Ties broken by direct-hit score desc,
// then by lexicographic path.
func pickRepresentative(matchSet matchedSet, memberIndices []int, centroid []float32) int {
	best := memberIndices[0]
	bestDist := cluster.CosineDistance(matchSet.members[best].vector, centroid)

	for _, idx := range memberIndices[1:] {
		dist := cluster.CosineDistance(matchSet.members[idx].vector, centroid)
		switch {
		case dist < bestDist:
			best = idx
			bestDist = dist
		case dist == bestDist:
			best = breakRepresentativeTie(matchSet, best, idx)
		}
	}

	return best
}

// provenanceRankFor maps a provenance role string to its F7 priority.
// Unknown roles get rank 0.
func provenanceRankFor(role string) int {
	switch role {
	case provenanceDirect:
		return provenanceRankDirect
	case provenanceClusterRep:
		return provenanceRankClusterRep
	default:
		return 0
	}
}

// rankCandidates scores each pre-filtered hit against queryVec, reads the
// note body for inclusion in the payload, and returns candidates sorted by
// descending cosine. Sidecars have already been validated and parsed by
// loadCompatibleSidecars, so no sidecar I/O happens here.
//
// When now is non-zero, each note's score is multiplied by a recency factor
// keyed on the sidecar's LastUsed date (falling back to the frontmatter
// created date). baseScore retains the pre-decay cosine for the activation
// cutoff and band logic. When now is zero, score == baseScore (pure cosine).
func rankCandidates(
	hits []compatibleSidecar,
	vault string,
	read func(string) ([]byte, error),
	queryVec []float32,
	now time.Time,
) []scoredCandidate {
	candidates := make([]scoredCandidate, 0, len(hits))

	for _, hit := range hits {
		notePath := pathOf(hit.note.Basename)
		full := filepath.Join(vault, notePath)

		noteBytes, noteErr := read(full)
		if noteErr != nil {
			continue
		}

		base, coord := bestVector(queryVec, hit.sidecar)
		created := parseCreatedFromNote(noteBytes)
		recencyScore := base

		if !now.IsZero() {
			ageDays := noteAgeDays(hit.sidecar.LastUsed, created, now)
			recencyScore = base * float32(recencyMultiplier(ageDays, 0, defaultRecencyParams()))
		}

		candidates = append(candidates, scoredCandidate{
			notePath:  notePath,
			basename:  hit.note.Basename,
			score:     recencyScore,
			baseScore: base,
			lastUsed:  hit.sidecar.LastUsed,
			created:   created,
			coord:     coord,
			sitVec:    hit.sidecar.SituationVector,
			bodyVec:   hit.sidecar.BodyVector,
			content:   stripWikilinks(string(noteBytes)),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	return candidates
}

// renderClusters converts per-phrase cluster reports into the YAML wire shape.
// Members are sorted by score desc and the representative is flagged. Each
// cluster is tagged with the phrase that produced it.
//
// candidate_l2s is nominated from each cluster's OWN note members (Phase 3,
// within-cluster nomination — reversal of D7). A cluster with no note members
// yields an empty candidate_l2s (explicitly allowed; chunk-only clusters exist).
func renderClusters(phraseClusters []phrasedCluster) []queryCluster {
	var out []queryCluster

	for _, pc := range phraseClusters {
		if pc.report.autoK.K == 0 {
			continue
		}

		for clusterID := range pc.report.autoK.K {
			members := collectClusterMembers(pc.matched, pc.report, clusterID)
			if len(members) == 0 {
				continue
			}

			centroid := pc.report.autoK.Centroids[clusterID]
			clusterNotes := clusterNoteIndexFromMembers(pc.matched, pc.report.memberIDs[clusterID])

			out = append(out, queryCluster{
				ID:           clusterID,
				Phrase:       pc.phrase,
				Size:         len(members),
				Silhouette:   pc.report.silhouettesByID[clusterID],
				Members:      members,
				CandidateL2s: topKCandidateNotes(centroid, clusterNotes),
			})
		}
	}

	if out == nil {
		return []queryCluster{}
	}

	return out
}

// renderItems converts resolved items into the YAML wire-shape items.
// outgoing maps a note basename to its authored wikilink targets so each
// item carries the basenames of its authored wikilink targets (for follow-on `engram show`).
func renderItems(resolved []resolvedItem, outgoing map[string][]string) []queryItem {
	items := make([]queryItem, len(resolved))

	for i, item := range resolved {
		kind := item.kind
		if kind == "" {
			kind = kindFromContent(item.content)
		}

		items[i] = queryItem{
			Path:          item.notePath,
			Kind:          kind,
			Score:         item.score,
			Provenances:   item.provenances,
			ClusterID:     item.clusterID,
			InDegree:      item.inDegree,
			OutboundLinks: outgoing[basenameFromNotePath(item.notePath)],
			Content:       item.content,
		}
	}

	return items
}

// renderQueryPayload encodes the resolved YAML payload for the multi-phrase
// pipeline output.
func renderQueryPayload(stdout io.Writer, merged aggregatedSummary) error {
	items := renderItems(merged.resolvedItems, merged.outgoing)
	clusters := renderClusters(merged.phraseClusters)

	items, snipped := capChunkContent(items, merged.contentBudget)
	// Full content = items still carrying their complete text — snippeted
	// chunks retain (truncated) content, so exclude them from the count.
	contentful := countItemsWithContent(items) - snipped

	directCount := 0

	for _, item := range items {
		if slices.Contains(item.Provenances, provenanceDirect) {
			directCount++
		}
	}

	payload := queryPayload{
		Version:  1,
		Phrases:  merged.phrases,
		Items:    items,
		Clusters: clusters,
		Budget: queryBudget{
			PhrasesQueried:       len(merged.phrases),
			TotalNotes:           merged.totalNotes,
			WithEmbeddings:       merged.withEmbeddings,
			ClustersFound:        len(clusters),
			DirectHitsReturned:   directCount,
			ItemsWithFullContent: contentful,
			Limit:                merged.limit,
			ContentBudget:        merged.contentBudget,
			ChunksSnippeted:      snipped,
		},
	}

	const yamlIndent = 2

	encoder := yaml.NewEncoder(stdout)
	encoder.SetIndent(yamlIndent)

	err := encoder.Encode(payload)
	if err != nil {
		return fmt.Errorf("query: encode: %w", err)
	}

	closeErr := encoder.Close()
	if closeErr != nil {
		return fmt.Errorf("query: close encoder: %w", closeErr)
	}

	return nil
}

// resolvedItemLess compares two items by F7 rules: provenance count
// desc → highest-rank provenance desc → score desc.
func resolvedItemLess(a, b resolvedItem) bool {
	if len(a.provenances) != len(b.provenances) {
		return len(a.provenances) > len(b.provenances)
	}

	rankA := maxProvenanceRank(a.provenances)
	rankB := maxProvenanceRank(b.provenances)

	if rankA != rankB {
		return rankA > rankB
	}

	return a.score > b.score
}

// runQuery is the sole query path. For each phrase it embeds once,
// scores notes and chunks with recency bias, merges into one ranked list
// (top-matchPhraseLimit=30 per phrase), then unions across phrases with dedup,
// relevance floor on baseScore, and a hard cap at matchSetCap=300. The
// resulting matched set is clustered exactly once (D1) and emits per-cluster
// candidate_l2s [{path, cosine}] so the recall skill can judge coverage.
func runQuery(
	ctx context.Context,
	args QueryArgs,
	notes []vaultgraph.Note,
	hits []compatibleSidecar,
	limit int,
	deps QueryDeps,
	stdout io.Writer,
) error {
	var nowL2 time.Time
	if deps.Now != nil {
		nowL2 = deps.Now()
	}

	chunkRecords, loadErr := loadClusterChunkRecords(args, deps)
	if loadErr != nil {
		return loadErr
	}

	noteUnion, chunkUnion, matchErr := buildMatchedSetFromPhrases(
		ctx, args.Phrases, hits, chunkRecords,
		args.VaultPath, nowL2, maxTurnBySource(chunkRecords), deps,
	)
	if matchErr != nil {
		return matchErr
	}

	// D1: build the matched set from the note union, then extend with matched chunks
	// so one AutoK pass clusters notes and chunks together.
	matchSet := buildMatchedSet(noteUnion)
	chunkItems := addMatchedChunksToMatchedSet(chunkUnion, &matchSet)

	report := clusterMatchedSet(matchSet, strings.Join(args.Phrases, "\n"))

	// mergeProvenances receives an empty matchedSet{} deliberately:
	// mergeClusterReps must not promote cluster reps into items[] because
	// the representative is agent-decided, not binary-computed (spec §2 step 4).
	// Direct-hit items come from the note union; chunk items are appended below.
	resolved := mergeProvenances(noteUnion, matchedSet{}, clusterReport{})
	resolved = applyProjectFilter(resolved, args.Project)
	resolved = append(resolved, chunkItems...)

	// Channel 2 — Recency (Phase 2): append the recentFillChunks newest chunks
	// by IngestedAt, deduped against the matched set, tagged provenanceRecent.
	// These are NOT added to the matched set and therefore do NOT appear in any
	// cluster's members[].
	recentItems := buildRecentFillItems(chunkRecords, chunkUnion, recentFillChunks)
	resolved = append(resolved, recentItems...)

	merged := aggregatedSummary{
		phrases:       args.Phrases,
		resolvedItems: resolved,
		phraseClusters: []phrasedCluster{
			{phrase: singleClusterPhrase, report: report, matched: matchSet},
		},
		// candidate_l2s are nominated from each cluster's own note members
		// (Phase 3, within-cluster nomination), not the full-vault index.
		outgoing:       outgoingByBasename(notes),
		totalNotes:     len(notes),
		withEmbeddings: len(hits),
		limit:          limit,
		contentBudget:  args.ContentBudget,
	}

	return renderQueryPayload(stdout, merged)
}

// seedFromQuery returns a deterministic uint64 seed for k-means
// initialization derived from the query string via FNV-1a.
func seedFromQuery(query string) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(query))

	return hasher.Sum64()
}

// singleClusterReport builds the K==0 fallback: one cluster holding every
// member, with a centroid computed as the mean of all member vectors (AutoK
// returns nil centroids when K==0) and a representative picked the normal way.
// Silhouette is undefined for a single cluster, so it is reported as zero.
func singleClusterReport(matchSet matchedSet, vectors [][]float32) clusterReport {
	allIndices := make([]int, len(matchSet.members))
	for i := range allIndices {
		allIndices[i] = i
	}

	centroid := meanVector(vectors)
	rep := pickRepresentative(matchSet, allIndices, centroid)

	return clusterReport{
		autoK:           cluster.AutoKResult{K: 1, Centroids: [][]float32{centroid}},
		memberIDs:       [][]int{allIndices},
		representatives: []int{rep},
		silhouettesByID: []float64{singletonClusterSilhouette},
	}
}

// snippet collapses all whitespace to single spaces, trims, and truncates to
// snippetMaxRunes runes — appending an ellipsis only when truncation occurs.
func snippet(content string) string {
	collapsed := strings.Join(strings.Fields(content), " ")

	runes := []rune(collapsed)
	if len(runes) <= snippetMaxRunes {
		return collapsed
	}

	return string(runes[:snippetMaxRunes-1]) + "…"
}

// splitMatchedSet splits the matched set into separate note and chunk slices
// (order preserved from the input, which is key-sorted).
func splitMatchedSet(matched []matchedSetItem) ([]scoredCandidate, []scoredChunk) {
	notes := make([]scoredCandidate, 0, len(matched))
	chunks := make([]scoredChunk, 0, len(matched))

	for _, item := range matched {
		if item.isChunk {
			chunks = append(chunks, item.chunk)
		} else {
			notes = append(notes, item.note)
		}
	}

	return notes, chunks
}

// stripWikilinks removes `[[target]]` and `[[target|display]]` syntax
// from markdown text.
func stripWikilinks(content string) string {
	return wikilinkRE.ReplaceAllStringFunc(content, func(match string) string {
		groups := wikilinkRE.FindStringSubmatch(match)
		if groups[2] != "" {
			return groups[2]
		}

		return groups[1]
	})
}

// topKCandidateNotes returns the top-K notes nearest the centroid by
// max(situation,body) cosine, sorted descending by centroid cosine (ties broken
// by lexicographic path for stability). K is at least candidateNoteK; when fewer
// than candidateNoteK notes exist, all are returned. An empty index returns
// nil. No cosine threshold is applied — all within-cluster note members are
// eligible (D7's full-vault nomination was reversed; see DESIGN-HISTORY §9). The sort
// key is CENTROID cosine (per spec §3.3: "top-K by centroid cosine");
// max-member cosine was rejected because it overfits to a cluster fragment.
func topKCandidateNotes(centroid []float32, idx candidateNoteIndex) []queryCandidateNote {
	if len(idx.paths) == 0 {
		return nil
	}

	type ranked struct {
		path   string
		cosine float32
	}

	all := make([]ranked, 0, len(idx.paths))

	for i := range idx.paths {
		sim := eitherAxisCosine(centroid, idx.sit[i], idx.body[i])
		all = append(all, ranked{path: idx.paths[i], cosine: sim})
	}

	sort.SliceStable(all, func(i, j int) bool {
		if all[i].cosine != all[j].cosine {
			return all[i].cosine > all[j].cosine
		}

		return all[i].path < all[j].path
	})

	count := min(candidateNoteK, len(all))

	out := make([]queryCandidateNote, count)
	for i := range count {
		out[i] = queryCandidateNote{Path: all[i].path, Cosine: all[i].cosine}
	}

	return out
}

// validateQueryArgs rejects invalid invocations before any vault I/O runs.
func validateQueryArgs(args QueryArgs) error {
	if len(args.Phrases) == 0 {
		return errQueryEmptyString
	}

	return nil
}

// warnModelMismatch emits a single aggregated advisory (M4) when sidecars
// were dropped because their embedding model differs from the active one.
// A no-op when nothing mismatched or no warning hook is wired. The message
// names the dropped count and the distinct stale model id(s) so a silent
// model swap that empties recall is surfaced instead of hidden.
func warnModelMismatch(
	logWarning func(string, ...any),
	loaded sidecarLoadResult,
	activeModelID string,
) {
	if logWarning == nil || loaded.mismatchedCount == 0 {
		return
	}

	logWarning(
		"query: dropped %d sidecar(s) with embedding model(s) %s != active %s; "+
			"run `engram embed apply --all` to re-embed",
		loaded.mismatchedCount,
		strings.Join(loaded.mismatchedModelIDs, ", "),
		activeModelID,
	)
}

// warnOldSchema emits a single advisory when sidecars were skipped for an
// old (unsupported) schema version. It is kept separate from the model
// mismatch advisory so the two reasons never contradict each other — this
// one names `--force` (re-embed in place) and the model advisory names
// `--all`. A no-op when nothing was old-schema or no warning hook is wired.
func warnOldSchema(logWarning func(string, ...any), loaded sidecarLoadResult) {
	if logWarning == nil || loaded.oldSchemaCount == 0 {
		return
	}

	logWarning(
		"query: %d sidecar(s) on an old schema were skipped; "+
			"run `engram embed apply --force` to re-embed",
		loaded.oldSchemaCount,
	)
}
