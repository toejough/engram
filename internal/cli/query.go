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

	"github.com/toejough/engram/internal/cluster"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// QueryArgs holds parsed flags for `engram query`.
type QueryArgs struct {
	Phrases      []string `targ:"flag,name=phrase,desc=query phrase (repeatable)"`
	VaultPath    string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root"`
	ChunksDir    string   `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks); chunks compete in the same ranking as notes"` //nolint:lll // single unbreakable struct-tag string
	Limit        int      `targ:"flag,name=limit,desc=max number of items to return (default 20)"`
	Project      string   `targ:"flag,name=project,desc=restrict items to notes with matching project: field (optional)"`
	Tiers        []string `targ:"flag,name=tier,desc=restrict items to notes matching these tier: values (repeatable)"`
	Synthesis    bool     `targ:"flag,name=synthesis,desc=union all phrase matches and cluster once for L3 synthesis (K=0 means one cluster; no min-size floor)"`    //nolint:lll // single unbreakable struct-tag string
	SynthesizeL2 bool     `targ:"flag,name=synthesize-l2,desc=union matched L1+L2 notes then cluster once and emit candidate_l2s per cluster for lazy L2 synthesis"` //nolint:lll // single unbreakable struct-tag string
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

// RunQuery embeds each query phrase, scores it against every note that
// has a current-model sidecar, ranks by descending cosine, expands a
// 3-hop subgraph over authored wikilinks, clusters that subgraph, and
// identifies hubs by in-degree before emitting the resolved YAML
// payload per the F6+F9.1 spec.
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

	if handled, err := dispatchSynthesisMode(ctx, args, notes, hits, limit, deps, stdout); handled {
		return err
	}

	summaries := make([]queryPipelineSummary, 0, len(args.Phrases))

	for _, phrase := range args.Phrases {
		summary, err := runSinglePhraseQuery(ctx, phrase, notes, hits, args.VaultPath, limit, deps)
		if err != nil {
			return err
		}

		summaries = append(summaries, summary)
	}

	merged := aggregatePhraseSummaries(args.Phrases, summaries, limit)
	merged.l3 = gatherL3Index(hits, args.VaultPath, deps.Read)
	merged.l2 = gatherTierIndex(hits, args.VaultPath, deps.Read, tierL2)
	merged.outgoing = outgoingByBasename(notes)
	merged.tiers = args.Tiers
	merged.resolvedItems = applyProjectFilter(merged.resolvedItems, args.Project)
	merged.resolvedItems = applyTierFilter(merged.resolvedItems, args.Tiers)

	chunkMust, chunkErr := mergeChunkSpace(ctx, args, deps, &merged, limit)
	if chunkErr != nil {
		return chunkErr
	}

	merged.resolvedItems = applyCombinedRecencyBand(
		merged.resolvedItems, chunkMust, deps.Now, limit,
		chunksConfigured(args, deps))

	return renderQueryPayload(stdout, merged)
}

// unexported constants.
const (
	// activationCosineCutoff is the minimum pre-decay baseScore a note must
	// have to be flagged activated: true in the payload. Provisional default
	// (a sanity floor for "genuine hit", NOT an empirically-tuned optimum;
	// true calibration requires end-to-end recall evals — see GitHub #646).
	activationCosineCutoff = 0.5
	// candidateL2K is the minimum number of candidate L2s to nominate per
	// cluster. The recall skill reads all K candidates to judge coverage;
	// generous nomination costs nothing (recall is the binary's job,
	// precision is the agent's).
	candidateL2K = 3
	// chunkClusterPhrase tags deterministic chunk-space clusters in the
	// unified payload (they span all phrases, like synthesis clusters).
	chunkClusterPhrase = "chunks"
	// chunkItemKind tags unified-ranking items sourced from the chunk index
	// (vs notes, whose kind derives from frontmatter).
	chunkItemKind            = "chunk"
	clusterMaxK              = 7
	clusterMinK              = 2
	clusterSilhouetteFloor   = 0.10
	defaultQueryLimit        = 20
	maxHubs                  = 5
	minSubgraphForClustering = 6
	provenanceClusterRep     = "cluster_rep"
	provenanceDirect         = "direct"
	provenanceHub            = "hub"
	provenanceRankClusterRep = 2
	provenanceRankDirect     = 3
	provenanceRankHub        = 1
	// singletonClusterSilhouette is the silhouette reported for the K=0->one
	// synthesis fallback cluster. Silhouette is undefined for a single
	// cluster, so zero stands in.
	singletonClusterSilhouette = 0.0
	subgraphCap                = 200
	subgraphMaxHops            = 3
	// synthesisClusterPhrase tags synthesis union clusters. They span every
	// seed phrase rather than one, so the per-phrase phrase tag is left empty.
	synthesisClusterPhrase = ""
	unknownKind            = "unknown"
)

// unexported variables.
var (
	errQueryEmptyString  = errors.New("query: empty query string")
	errQueryModeConflict = errors.New("query: --synthesis and --synthesize-l2 are mutually exclusive")
	errQueryNoEmbeddings = errors.New(
		"query: vault has notes but no current-model embeddings; run `engram embed apply --all`",
	)
	// projectLineRE matches a `project: <slug>` line in YAML frontmatter,
	// anchored to start-of-line so body text can't false-match. Slug shape
	// mirrors the write-side validation: [a-z0-9-]+.
	projectLineRE = regexp.MustCompile(`(?m)^project:\s*([a-z0-9-]+)\s*$`)
	// tierLineRE matches a `tier: L<n>` line in YAML frontmatter,
	// anchored to start-of-line so body text can't false-match.
	tierLineRE = regexp.MustCompile(`(?m)^tier:\s*(L[0-9]+)\s*$`)
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
	chunkClusters  []queryCluster
	l3             tierIndex
	l2             tierIndex
	outgoing       map[string][]string
	tiers          []string
	totalNotes     int
	withEmbeddings int
	limit          int
	subgraphSize   int
	subgraphCapped bool
	hopsTraversed  int
}

// clusterReport collects the AutoK output for the payload-rendering
// stage. Empty Members means clustering was skipped or yielded nothing.
type clusterReport struct {
	autoK           cluster.AutoKResult
	memberIDs       [][]int // memberIDs[c] = subgraphMember indices in cluster c
	representatives []int   // representatives[c] = subgraphMember index for cluster c
	silhouettesByID []float64
}

// compatibleSidecar bundles a note with its already-loaded current-model
// sidecar — the result of loadCompatibleSidecars's filtering pass. Carrying
// the parsed sidecar forward avoids re-reading it during scoring.
type compatibleSidecar struct {
	note    vaultgraph.Note
	sidecar embed.Sidecar
}

// expandedSubgraph is the post-BFS, post-sidecar-filtering subgraph the
// later stages operate on.
type expandedSubgraph struct {
	members       []subgraphMember
	graph         vaultgraph.Graph
	hopsTraversed int
	capped        bool
}

// hubReport identifies the top-N hubs by subgraph in-degree.
type hubReport struct {
	memberIDs []int // member index per hub, sorted by spec rules
	inDegrees []int // inDegrees[i] = in-degree of members[memberIDs[i]]
}

// phrasedCluster pairs a cluster report with the phrase that produced it,
// so the payload can tag each cluster with its originating query phrase.
type phrasedCluster struct {
	phrase   string
	report   clusterReport
	subgraph expandedSubgraph
}

// queryBudget reports the totals visible to the caller per the YAML schema.
type queryBudget struct {
	PhrasesQueried       int  `yaml:"phrases_queried"`
	TotalNotes           int  `yaml:"total_notes"`
	WithEmbeddings       int  `yaml:"with_embeddings"`
	SubgraphSize         int  `yaml:"subgraph_size"`
	SubgraphSizeCapped   bool `yaml:"subgraph_size_capped"`
	HopsTraversed        int  `yaml:"hops_traversed"`
	ClustersFound        int  `yaml:"clusters_found"`
	HubsReturned         int  `yaml:"hubs_returned"`
	DirectHitsReturned   int  `yaml:"direct_hits_returned"`
	ItemsWithFullContent int  `yaml:"items_with_full_content"`
	Limit                int  `yaml:"limit"`
}

// queryCandidateL2 is one candidate L2 note for a cluster. The binary emits
// top-K by centroid cosine (K >= candidateL2K); the recall skill judges
// coverage — no cosine-band decision happens in the binary.
type queryCandidateL2 struct {
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
	NearestL3    *queryNearestL3      `yaml:"nearest_l3,omitempty"`
	CandidateL2s []queryCandidateL2   `yaml:"candidate_l2s,omitempty"`
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
// fence-aware graph parser's output) so a tier-limited recall still shows what
// is one hop away to fetch with `engram show <basename>`; it is never
// tier-filtered, which is what keeps the tier-read axis a test of
// direct-provision-vs-follow-on-demand rather than a blinding.
type queryItem struct {
	Path          string   `yaml:"path"`
	Kind          string   `yaml:"kind"`
	Score         float32  `yaml:"score"`
	Provenances   []string `yaml:"provenances"`
	ClusterID     *int     `yaml:"cluster_id,omitempty"`
	InDegree      *int     `yaml:"in_degree,omitempty"`
	OutboundLinks []string `yaml:"outbound_links,omitempty"`
	Content       string   `yaml:"content,omitempty"`
	Activated     bool     `yaml:"activated,omitempty"`
}

// queryNearestL3 is the nearest existing L3 note for a cluster centroid.
type queryNearestL3 struct {
	Path   string  `yaml:"path"`
	Cosine float32 `yaml:"cosine"`
}

// queryPayload is the top-level YAML document.
type queryPayload struct {
	Version  int            `yaml:"version"`
	Phrases  []string       `yaml:"phrases"`
	Items    []queryItem    `yaml:"items"`
	Clusters []queryCluster `yaml:"clusters"`
	Budget   queryBudget    `yaml:"budget"`
}

// queryPipelineSummary bundles every stage's output for rendering.
type queryPipelineSummary struct {
	subgraph       expandedSubgraph
	clusters       clusterReport
	resolvedItems  []resolvedItem
	totalNotes     int
	withEmbeddings int
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
// the vector that matched it).
type scoredCandidate struct {
	notePath  string
	basename  string
	score     float32
	baseScore float32 // pre-decay cosine; preserved through the pipeline
	lastUsed  string  // YYYY-MM-DD from Sidecar.LastUsed
	created   string  // YYYY-MM-DD from note frontmatter
	coord     []float32
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

// subgraphMember bundles a node's basename, vault-relative path,
// sidecar vector, query-similarity score, and (optionally) cached body.
// kind overrides content-derived kind detection for chunk members.
type subgraphMember struct {
	basename string
	notePath string
	vector   []float32
	score    float32
	content  string
	kind     string // empty = note; chunkItemKind for chunks
}

// tierIndex holds the vault-wide set of one tier's note paths and BOTH
// sidecar vectors for per-cluster nearest-tier lookup by max(situation,body).
// Built once per RunQuery call.
type tierIndex struct {
	paths []string
	sit   [][]float32
	body  [][]float32
}

// aggregatePhraseSummaries merges per-phrase pipeline results into a single
// aggregatedSummary per the issue-639 spec:
//   - items: dedup by path, max score across phrases, union provenances,
//     max in_degree; cluster_id cleared (clusters are per-phrase).
//   - clusters: retained per-phrase, tagged with their originating phrase.
//   - budget: subgraphSize is sum, hopsTraversed is max, capped is OR.
func aggregatePhraseSummaries(phrases []string, summaries []queryPipelineSummary, limit int) aggregatedSummary {
	items := mergeItemsByPath(summaries, limit)

	phraseClusters := make([]phrasedCluster, 0, len(summaries))
	for i, s := range summaries {
		phraseClusters = append(phraseClusters, phrasedCluster{
			phrase:   phrases[i],
			report:   s.clusters,
			subgraph: s.subgraph,
		})
	}

	totalSubgraph, capped, maxHops := aggregateSubgraphBudget(summaries)
	first := summaries[0]

	return aggregatedSummary{
		phrases:        phrases,
		resolvedItems:  items,
		phraseClusters: phraseClusters,
		totalNotes:     first.totalNotes,
		withEmbeddings: first.withEmbeddings,
		limit:          limit,
		subgraphSize:   totalSubgraph,
		subgraphCapped: capped,
		hopsTraversed:  maxHops,
	}
}

// aggregateSubgraphBudget computes the cross-phrase budget fields: total
// subgraph size (sum), capped flag (OR), and max hops traversed.
func aggregateSubgraphBudget(summaries []queryPipelineSummary) (totalSize int, capped bool, maxHops int) {
	for _, s := range summaries {
		totalSize += len(s.subgraph.members)

		if s.subgraph.capped {
			capped = true
		}

		if s.subgraph.hopsTraversed > maxHops {
			maxHops = s.subgraph.hopsTraversed
		}
	}

	return totalSize, capped, maxHops
}

// appendSynthesisChunks loads the matched chunks, appends them as subgraph
// members (so they cluster together with the notes — D1), and returns the
// parallel resolvedItem slice for items[]. Each chunk member's basename is set
// to its source#anchor notePath to avoid byBasename[""] collisions if the
// subgraph is ever passed to mergeProvenances (CA-09).
func appendSynthesisChunks(
	ctx context.Context,
	args QueryArgs,
	deps QueryDeps,
	subgraph *expandedSubgraph,
	limit int,
) ([]resolvedItem, error) {
	records, loadErr := loadChunkRecords(args.ChunksDir, ChunkQueryDeps{
		ListIndexes: deps.ListChunkIndexes, ReadFile: deps.Read, Embedder: deps.Embedder,
	})
	if loadErr != nil {
		return nil, loadErr
	}

	scored, scoreErr := scoreChunks(ctx, args.Phrases, records, deps.Embedder)
	if scoreErr != nil {
		return nil, scoreErr
	}

	// Bound the clustered + returned chunk set to the top-limit by score.
	// cluster.Silhouette is O(n^2) per K, so clustering the whole corpus is
	// prohibitively slow on large indices — the same bound mergeChunkSpace
	// already applies for the non-synthesis path.
	sortScoredDesc(scored)

	if len(scored) > limit {
		scored = scored[:limit]
	}

	chunkItems := make([]resolvedItem, 0, len(scored))

	for _, s := range scored {
		path := chunkNotePath(s.record)
		subgraph.members = append(subgraph.members, subgraphMember{
			basename: path, // set to avoid byBasename[""] collisions if passed to mergeProvenances
			notePath: path,
			content:  s.record.Text,
			vector:   s.record.Vector,
			score:    s.score,
			kind:     chunkItemKind,
		})
		chunkItems = append(chunkItems, resolvedItem{
			notePath:    path,
			content:     s.record.Text,
			score:       s.score,
			provenances: []string{provenanceDirect},
			kind:        chunkItemKind,
		})
	}

	return chunkItems, nil
}

// appendUniqueProvenance adds role to item.provenances iff not already present.
func appendUniqueProvenance(item *resolvedItem, role string) {
	if slices.Contains(item.provenances, role) {
		return
	}

	item.provenances = append(item.provenances, role)
}

// applyCombinedRecencyBand applies the single combined floor band that
// guarantees both the floor-newest chunks (chunkMust) and the most-recently-
// used notes survive the limit cap without mutual eviction.
//
// chunksActive must be true when a chunks dir is configured; without chunks
// the cap is handled by renderQueryPayload (pre-Phase-4 note-only path).
// When nowFn is nil, only a plain cap is applied (no recency band).
// items must already be sorted descending by score.
func applyCombinedRecencyBand(
	items []resolvedItem,
	chunkMust []resolvedItem,
	nowFn func() time.Time,
	limit int,
	chunksActive bool,
) []resolvedItem {
	if !chunksActive {
		return items
	}

	if nowFn == nil {
		if len(items) > limit {
			return items[:limit]
		}

		return items
	}

	// Collect note-must PRE-CAP so low-cosine but recently-used notes that
	// rank below the cap boundary are still eligible for the floor band.
	noteMust := mostRecentlyUsedNoteItems(items, nowFn(), defaultRecencyFloor)

	// Interleave chunkMust and noteMust (chunk0, note0, chunk1, note1, …)
	// so that when the total exceeds the limit, fillRecencyBand injects a
	// fair mix of both rather than exhausting the budget with chunks-first.
	combined := make([]resolvedItem, 0, len(chunkMust)+len(noteMust))

	for i := 0; i < len(chunkMust) || i < len(noteMust); i++ {
		if i < len(chunkMust) {
			combined = append(combined, chunkMust[i])
		}

		if i < len(noteMust) {
			combined = append(combined, noteMust[i])
		}
	}

	// Cap first, then let fillRecencyBand re-insert any evicted must-includes.
	if len(items) > limit {
		items = items[:limit]
	}

	return fillRecencyBand(items, combined, limit)
}

// applyProjectFilter drops items whose frontmatter project: field doesn't
// match the requested slug. Empty project is a no-op (returns items
// unchanged). Items with no loaded content cannot be verified and are
// dropped when a non-empty project is specified — the wikilink graph
// stayed intact during BFS, so a project-A note still reaches its
// project-A neighbors through a project-B bridge; the filter only
// affects which items are emitted, not which ones were considered.
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

// applyTierFilter drops items whose frontmatter tier: field matches none of
// the requested tier labels (the union read of §1.4). An empty tiers slice is
// a no-op (returns items unchanged). Items with no loaded content cannot be
// verified and are dropped when any tier is specified.
func applyTierFilter(items []resolvedItem, tiers []string) []resolvedItem {
	if len(tiers) == 0 {
		return items
	}

	out := make([]resolvedItem, 0, len(items))

	for _, item := range items {
		if itemMatchesTier(item, tiers) {
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
func breakRepresentativeTie(subgraph expandedSubgraph, a, b int) int {
	memberA := subgraph.members[a]
	memberB := subgraph.members[b]

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

// buildSubgraphMembers assembles the final subgraphMember list, reading
// non-direct bodies on demand and scoring each member against queryVec.
func buildSubgraphMembers(
	memberNames []string,
	hitByName map[string]compatibleSidecar,
	directContentByBasename map[string]string,
	vault string,
	read func(string) ([]byte, error),
	queryVec []float32,
) []subgraphMember {
	members := make([]subgraphMember, 0, len(memberNames))

	for _, name := range memberNames {
		hit := hitByName[name]
		notePath := pathOf(name)

		score, coord := bestVector(queryVec, hit.sidecar)
		member := subgraphMember{
			basename: name,
			notePath: notePath,
			vector:   coord,
			score:    score,
		}

		if content, ok := directContentByBasename[name]; ok {
			member.content = content
		} else {
			body, err := read(filepath.Join(vault, notePath))
			if err == nil {
				member.content = stripWikilinks(string(body))
			}
		}

		members = append(members, member)
	}

	return members
}

// buildUnionSubgraph turns the deduped union direct hits into an
// expandedSubgraph whose members ARE those hits (no BFS expansion), carrying
// each hit's winning-vector coordinate forward. The graph/hops/capped fields
// stay zero-valued because synthesis clusters the union itself and computes
// no hubs.
func buildUnionSubgraph(union []scoredCandidate) expandedSubgraph {
	members := make([]subgraphMember, 0, len(union))

	for _, hit := range union {
		members = append(members, subgraphMember{
			basename: hit.basename,
			notePath: hit.notePath,
			vector:   hit.coord,
			score:    hit.score,
			content:  hit.content,
		})
	}

	return expandedSubgraph{members: members}
}

// chunksConfigured reports whether a chunk index is wired into this run
// (non-empty chunks dir and a list function available).
func chunksConfigured(args QueryArgs, deps QueryDeps) bool {
	return args.ChunksDir != "" && deps.ListChunkIndexes != nil
}

// clusterChunkItems runs the SAME deterministic clustering machinery the note
// pipeline uses (auto-k k-means + silhouette floor) over the scored chunks,
// annotating each cluster with the nearest existing L2 note so the recall
// skill can apply its 95/80/<80 crystallization bands mechanically instead of
// judging novelty by eye.
func clusterChunkItems(scored []scoredChunk, l2Notes tierIndex, tiers, phrases []string) []queryCluster {
	if len(scored) < minSubgraphForClustering {
		return nil
	}

	vectors := make([][]float32, 0, len(scored))
	for _, s := range scored {
		vectors = append(vectors, s.record.Vector)
	}

	seed := seedFromQuery(strings.Join(phrases, "|"))

	autoK, err := cluster.AutoK(vectors, clusterMinK, clusterMaxK, clusterSilhouetteFloor, seed)
	if err != nil || autoK.K == 0 {
		return nil
	}

	silhouettes := perClusterMeanSilhouette(vectors, autoK.Assignments, autoK.K)
	clusters := make([]queryCluster, 0, autoK.K)

	for clusterID := range autoK.K {
		var members []queryClusterMember

		for memberIdx, assigned := range autoK.Assignments {
			if assigned == clusterID {
				members = append(members, queryClusterMember{
					Path:  chunkNotePath(scored[memberIdx].record),
					Score: scored[memberIdx].score,
				})
			}
		}

		if len(members) == 0 {
			continue
		}

		clusters = append(clusters, queryCluster{
			ID:           clusterID,
			Phrase:       chunkClusterPhrase,
			Size:         len(members),
			Silhouette:   silhouettes[clusterID],
			Members:      members,
			CandidateL2s: topKCandidateL2sForTier(autoK.Centroids[clusterID], l2Notes, tiers),
		})
	}

	return clusters
}

// clusterSubgraph runs auto-k k-means + silhouette over the subgraph's
// vectors with a query-derived deterministic seed. Subgraphs smaller
// than minSubgraphForClustering short-circuit to "no clusters".
func clusterSubgraph(subgraph expandedSubgraph, query string) clusterReport {
	if len(subgraph.members) < minSubgraphForClustering {
		return clusterReport{}
	}

	vectors := make([][]float32, len(subgraph.members))
	for i, member := range subgraph.members {
		vectors[i] = member.vector
	}

	seed := seedFromQuery(query)

	autoK, err := cluster.AutoK(vectors, clusterMinK, clusterMaxK, clusterSilhouetteFloor, seed)
	if err != nil || autoK.K == 0 {
		return clusterReport{}
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
		representatives[c] = pickRepresentative(subgraph, memberIDs[c], autoK.Centroids[c])
	}

	perClusterSilhouettes := perClusterMeanSilhouette(vectors, autoK.Assignments, autoK.K)

	return clusterReport{
		autoK:           autoK,
		memberIDs:       memberIDs,
		representatives: representatives,
		silhouettesByID: perClusterSilhouettes,
	}
}

// clusterUnionForSynthesis clusters the union subgraph exactly once. It mirrors
// clusterSubgraph but (a) skips the minSubgraphForClustering floor, and (b) on
// AutoK returning K==0 (no split beats the silhouette floor) or an error, falls
// back to a SINGLE cluster of all members so a non-empty union always yields
// >=1 cluster. An empty union yields an empty (K==0) report.
func clusterUnionForSynthesis(subgraph expandedSubgraph, query string) clusterReport {
	if len(subgraph.members) == 0 {
		return clusterReport{}
	}

	vectors := make([][]float32, len(subgraph.members))
	for i, member := range subgraph.members {
		vectors[i] = member.vector
	}

	seed := seedFromQuery(query)

	autoK, err := cluster.AutoK(vectors, clusterMinK, clusterMaxK, clusterSilhouetteFloor, seed)
	if err != nil || autoK.K == 0 {
		return singleClusterReport(subgraph, vectors)
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
		representatives[c] = pickRepresentative(subgraph, memberIDs[c], autoK.Centroids[c])
	}

	return clusterReport{
		autoK:           autoK,
		memberIDs:       memberIDs,
		representatives: representatives,
		silhouettesByID: perClusterMeanSilhouette(vectors, autoK.Assignments, autoK.K),
	}
}

// collectClusterMembers gathers per-cluster member rows in score-desc
// order, marking the representative. When tiers is non-empty, members whose
// frontmatter tier matches none of them are dropped (T1a tier isolation); if
// the elected representative is among those dropped, the highest-scoring
// surviving member is promoted so every non-empty cluster still reports
// exactly one representative.
func collectClusterMembers(
	subgraph expandedSubgraph,
	report clusterReport,
	clusterID int,
	tiers []string,
) []queryClusterMember {
	memberIndices := make([]int, len(report.memberIDs[clusterID]))
	copy(memberIndices, report.memberIDs[clusterID])

	sort.SliceStable(memberIndices, func(i, j int) bool {
		return subgraph.members[memberIndices[i]].score > subgraph.members[memberIndices[j]].score
	})

	repIdx := report.representatives[clusterID]
	members := make([]queryClusterMember, 0, len(memberIndices))
	repRetained := false

	for _, idx := range memberIndices {
		member := subgraph.members[idx]
		if !memberMatchesTier(member, tiers) {
			continue
		}

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

	// Members are score-desc, so the first survivor is the best-scoring
	// one; promote it when the original representative was tier-filtered out.
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

// dispatchSynthesisMode routes to the single-cluster synthesis modes. It
// returns handled=true (with the mode's error) when --synthesis or
// --synthesize-l2 is set, and handled=false to fall through to the default
// per-phrase pipeline. The two modes are mutually exclusive; that conflict is
// validated up front in RunQuery, so this router never sees both flags set.
func dispatchSynthesisMode(
	ctx context.Context,
	args QueryArgs,
	notes []vaultgraph.Note,
	hits []compatibleSidecar,
	limit int,
	deps QueryDeps,
	stdout io.Writer,
) (bool, error) {
	switch {
	case args.SynthesizeL2:
		return true, runSynthesizeL2Query(ctx, args, notes, hits, limit, deps, stdout)
	case args.Synthesis:
		return true, runSynthesisQuery(ctx, args, notes, hits, limit, deps, stdout)
	default:
		return false, nil
	}
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

// expandSubgraph runs a 3-hop BFS over the authored wikilink graph,
// starting from direct hits, undirected for expansion, capped at 200
// notes. Subgraph membership requires a compatible sidecar — notes
// without one are filtered out silently after BFS completes.
//
// All notes (regardless of sidecar status) participate in the graph
// itself, since a non-embedded intermediate node can still bridge two
// embedded notes. After BFS, drop non-compatible-sidecar notes from the
// visited set: their presence as bridges is preserved by the graph
// edges, not by their inclusion in the subgraph member list.
//
// Each member's body is read once and its similarity to queryVec is
// computed once. Direct-hit candidates carry their pre-loaded content
// and score forward instead of being re-read.
func expandSubgraph(
	notes []vaultgraph.Note,
	hits []compatibleSidecar,
	directHits []scoredCandidate,
	vault string,
	read func(string) ([]byte, error),
	queryVec []float32,
) expandedSubgraph {
	graph := vaultgraph.BuildGraph(notes)
	seeds := seedBasenames(directHits)
	bfs := vaultgraph.BFSWithCap(graph, seeds, subgraphMaxHops, subgraphCap)

	hitByName := indexHitsByBasename(hits)
	memberNames := filterToCompatibleMembers(bfs.Visited, hitByName)
	directContentByBasename := indexDirectContent(directHits)

	members := buildSubgraphMembers(
		memberNames,
		hitByName,
		directContentByBasename,
		vault,
		read,
		queryVec,
	)

	return expandedSubgraph{
		members:       members,
		graph:         graph,
		hopsTraversed: bfs.HopsReached,
		capped:        bfs.Capped,
	}
}

// filterHitsToTiers keeps only the hits whose note frontmatter tier is in the
// given tier set, by reading each note's body. Used by --synthesize-l2 to
// constrain the CLUSTERED set to L1+L2 (distinct from --tier, which filters
// emitted items post-clustering).
func filterHitsToTiers(
	hits []compatibleSidecar,
	vault string,
	read func(string) ([]byte, error),
	tiers []string,
) []compatibleSidecar {
	kept := make([]compatibleSidecar, 0, len(hits))

	for _, hit := range hits {
		body, readErr := read(filepath.Join(vault, pathOf(hit.note.Basename)))
		if readErr != nil {
			continue
		}

		if itemMatchesTier(resolvedItem{content: stripWikilinks(string(body))}, tiers) {
			kept = append(kept, hit)
		}
	}

	return kept
}

// filterToCompatibleMembers returns the sorted basenames from visited
// that also appear in hitByName (compatible-sidecar set). Sorting fixes
// the visit-order non-determinism inherent to map iteration.
func filterToCompatibleMembers(
	visited map[string]struct{}, hitByName map[string]compatibleSidecar,
) []string {
	memberNames := make([]string, 0, len(visited))

	for name := range visited {
		if _, ok := hitByName[name]; !ok {
			continue
		}

		memberNames = append(memberNames, name)
	}

	sort.Strings(memberNames)

	return memberNames
}

// gatherL3Index collects the L3 notes into a tierIndex for nearest-L3 lookup.
func gatherL3Index(
	hits []compatibleSidecar,
	vault string,
	read func(string) ([]byte, error),
) tierIndex {
	return gatherTierIndex(hits, vault, read, tierL3)
}

// gatherTierIndex reads each compatible-sidecar note's content and collects
// those whose frontmatter carries the requested tier into a small index used
// for nearest-tier lookup during cluster rendering. Both sidecar vectors are
// carried so the per-cluster lookup can gate by max(situation,body). Called
// once per query so the per-cluster lookups are O(1) in I/O.
func gatherTierIndex(
	hits []compatibleSidecar,
	vault string,
	read func(string) ([]byte, error),
	tier string,
) tierIndex {
	idx := tierIndex{}

	for _, hit := range hits {
		notePath := pathOf(hit.note.Basename)

		body, readErr := read(filepath.Join(vault, notePath))
		if readErr != nil {
			continue
		}

		item := resolvedItem{content: stripWikilinks(string(body))}
		if !itemMatchesTier(item, []string{tier}) {
			continue
		}

		idx.paths = append(idx.paths, notePath)
		idx.sit = append(idx.sit, hit.sidecar.SituationVector)
		idx.body = append(idx.body, hit.sidecar.BodyVector)
	}

	return idx
}

// identifyHubs returns the top-N (≤ maxHubs) subgraph notes by
// subgraph-internal in-degree, ties broken by direct-hit score desc
// then by lexicographic notePath asc. Notes with zero in-degree are
// excluded.
func identifyHubs(subgraph expandedSubgraph) hubReport {
	if len(subgraph.members) == 0 {
		return hubReport{}
	}

	subset := make(map[string]struct{}, len(subgraph.members))
	for _, member := range subgraph.members {
		subset[member.basename] = struct{}{}
	}

	type indexedDegree struct {
		memberIdx int
		inDegree  int
	}

	candidates := make([]indexedDegree, 0, len(subgraph.members))

	for idx, member := range subgraph.members {
		degree := subgraph.graph.InDegreeIn(member.basename, subset)
		if degree == 0 {
			continue
		}

		candidates = append(candidates, indexedDegree{memberIdx: idx, inDegree: degree})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].inDegree != candidates[j].inDegree {
			return candidates[i].inDegree > candidates[j].inDegree
		}

		memberI := subgraph.members[candidates[i].memberIdx]
		memberJ := subgraph.members[candidates[j].memberIdx]

		if memberI.score != memberJ.score {
			return memberI.score > memberJ.score
		}

		return memberI.notePath < memberJ.notePath
	})

	if len(candidates) > maxHubs {
		candidates = candidates[:maxHubs]
	}

	report := hubReport{
		memberIDs: make([]int, len(candidates)),
		inDegrees: make([]int, len(candidates)),
	}

	for i, c := range candidates {
		report.memberIDs[i] = c.memberIdx
		report.inDegrees[i] = c.inDegree
	}

	return report
}

// indexDirectContent maps each direct hit's basename to its already-loaded
// (wikilink-stripped) content so we don't re-read those files.
func indexDirectContent(directHits []scoredCandidate) map[string]string {
	out := make(map[string]string, len(directHits))
	for _, candidate := range directHits {
		out[candidate.basename] = candidate.content
	}

	return out
}

// indexHitsByBasename keys the compatible-sidecar set by note basename
// for O(1) lookup during member assembly.
func indexHitsByBasename(hits []compatibleSidecar) map[string]compatibleSidecar {
	out := make(map[string]compatibleSidecar, len(hits))
	for _, hit := range hits {
		out[hit.note.Basename] = hit
	}

	return out
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

// itemMatchesTier scans the item's loaded content's frontmatter for a
// tier: L<n> line whose value is one of the requested tier labels. Returns
// false when content is missing or when the frontmatter block is malformed.
// Callers guarantee a non-empty tiers slice (the empty-slice no-op is handled
// upstream in applyTierFilter/memberMatchesTier).
func itemMatchesTier(item resolvedItem, tiers []string) bool {
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

	match := tierLineRE.FindStringSubmatch(front)

	return len(match) == 2 && slices.Contains(tiers, match[1])
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

// memberMatchesTier reports whether a subgraph member's loaded content
// carries the requested tier. An empty tier matches everything (the
// blended recall path). Chunk members carry no frontmatter tier (kind ==
// chunkItemKind); they are excluded only when an explicit tier set is
// requested, and always pass the blended (empty-tier) path so D1's unified
// synthesize-l2 clustering retains them. Note members without loaded content
// cannot be verified and are treated as non-matching when a tier is requested
// — mirroring applyTierFilter's items[] behavior so all channels stay
// consistent.
func memberMatchesTier(member subgraphMember, tiers []string) bool {
	if len(tiers) == 0 {
		return true
	}

	if member.kind == chunkItemKind {
		return false
	}

	return itemMatchesTier(resolvedItem{content: member.content}, tiers)
}

// mergeChunkSpace scores every indexed chunk against the query phrases and
// merges the results into the resolved items, re-ranking by score. The cap
// is deferred to the caller (RunQuery) so the caller can collect
// mostRecentlyUsedNoteItems from the full sorted list before eviction.
// Clustering is done on the top-limit slice to keep O(n²) silhouette bounded.
// Returns the newest-chunk must-include set so RunQuery can build the combined
// band. No-op (nil, nil) when no chunks dir is configured.
func mergeChunkSpace(
	ctx context.Context,
	args QueryArgs,
	deps QueryDeps,
	merged *aggregatedSummary,
	limit int,
) ([]resolvedItem, error) {
	if args.ChunksDir == "" || deps.ListChunkIndexes == nil {
		return nil, nil
	}

	records, err := loadChunkRecords(args.ChunksDir, ChunkQueryDeps{
		ListIndexes: deps.ListChunkIndexes, ReadFile: deps.Read, Embedder: deps.Embedder,
	})
	if err != nil {
		return nil, err
	}

	scored, err := scoreChunks(ctx, args.Phrases, records, deps.Embedder)
	if err != nil {
		return nil, err
	}

	// Recency re-rank (chunk-only): lift recent chunks before they compete with notes.
	var chunkMust []resolvedItem

	if deps.Now != nil {
		params := defaultRecencyParams()
		scored = applyChunkRecency(scored, deps.Now(), maxTurnBySource(records), params)
		sortScoredDesc(scored)
		chunkMust = newestChunkItems(scored, params.floor)
	}

	for _, s := range scored {
		merged.resolvedItems = append(merged.resolvedItems, resolvedItem{
			notePath:    chunkNotePath(s.record),
			content:     s.record.Text,
			score:       s.score,
			provenances: []string{provenanceDirect},
			kind:        chunkItemKind,
		})
	}

	sort.SliceStable(merged.resolvedItems, func(i, j int) bool {
		return merged.resolvedItems[i].score > merged.resolvedItems[j].score
	})

	// Cluster the top-limit slice so the O(n²) silhouette stays bounded.
	// The cap to limit is applied after return (in RunQuery), but clustering
	// over all chunks would be prohibitively slow on large indices.
	clusterView := merged.resolvedItems
	if len(clusterView) > limit {
		clusterView = clusterView[:limit]
	}

	merged.chunkClusters = clusterChunkItems(
		survivingChunks(scored, clusterView), merged.l2, merged.tiers, merged.phrases)

	return chunkMust, nil
}

// mergeClusterReps annotates representatives with provenance + cluster_id,
// adding new entries when a rep is not already a direct hit.
func mergeClusterReps(
	subgraph expandedSubgraph,
	clusters clusterReport,
	byBasename map[string]*resolvedItem,
) {
	for clusterID, memberIdx := range clusters.representatives {
		if memberIdx < 0 || memberIdx >= len(subgraph.members) {
			continue
		}

		member := subgraph.members[memberIdx]

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

// mergeHubItems annotates hubs with provenance + in_degree, adding new
// entries when a hub is not already a direct hit or rep.
func mergeHubItems(
	subgraph expandedSubgraph,
	hubs hubReport,
	byBasename map[string]*resolvedItem,
) {
	for i, memberIdx := range hubs.memberIDs {
		if memberIdx < 0 || memberIdx >= len(subgraph.members) {
			continue
		}

		member := subgraph.members[memberIdx]

		resolved := byBasename[member.basename]
		if resolved == nil {
			resolved = &resolvedItem{
				notePath: member.notePath,
				content:  member.content,
				score:    member.score,
			}
			byBasename[member.basename] = resolved
		}

		appendUniqueProvenance(resolved, provenanceHub)

		inDegreeCopy := hubs.inDegrees[i]
		resolved.inDegree = &inDegreeCopy
	}
}

// mergeIntoExisting updates existing with the best score and baseScore,
// unioned provenances, in_degree from src (if existing has none), and
// lastUsed/created from src when existing fields are empty.
// baseScore is maximised so the activated flag (baseScore >=
// activationCosineCutoff) is not phrase-order-dependent.
// in_degree is not maximised across phrases because undirected BFS
// always reaches the same linkers for a note regardless of starting
// point, so both phrases produce identical in_degrees.
func mergeIntoExisting(existing, src *resolvedItem) {
	if src.score > existing.score {
		existing.score = src.score
		existing.content = src.content
	}

	if src.baseScore > existing.baseScore {
		existing.baseScore = src.baseScore
	}

	if existing.lastUsed == "" && src.lastUsed != "" {
		existing.lastUsed = src.lastUsed
	}

	if existing.created == "" && src.created != "" {
		existing.created = src.created
	}

	for _, p := range src.provenances {
		appendUniqueProvenance(existing, p)
	}

	if src.inDegree != nil && existing.inDegree == nil {
		v := *src.inDegree
		existing.inDegree = &v
	}
}

// mergeItemsByPath deduplicates resolved items across all phrase summaries:
// max score wins, provenances are unioned, in_degree takes the max, and
// cluster_id is cleared (clusters are per-phrase in the multi-phrase payload).
func mergeItemsByPath(summaries []queryPipelineSummary, limit int) []resolvedItem {
	byPath := make(map[string]*resolvedItem, len(summaries)*limit)

	for _, s := range summaries {
		for i := range s.resolvedItems {
			src := &s.resolvedItems[i]
			existing, ok := byPath[src.notePath]

			if !ok {
				c := *src
				c.clusterID = nil
				byPath[src.notePath] = &c

				continue
			}

			mergeIntoExisting(existing, src)
		}
	}

	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}

	sort.Strings(paths)

	items := make([]resolvedItem, 0, len(byPath))
	for _, path := range paths {
		item := byPath[path]
		if item == nil {
			continue
		}

		items = append(items, *item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		return resolvedItemLess(items[i], items[j])
	})

	return items
}

// mergeProvenances builds the resolved item list per F7's rules:
// items = direct hits ∪ cluster reps ∪ hubs, deduped by basename,
// each item carrying every applicable provenance role + metadata.
//
// Ordering: provenance count desc → highest-priority provenance desc →
// score desc.
//
// Bodies for non-direct entries are looked up from the subgraph
// member's `content` field (loaded if it was a direct hit) — non-direct
// reps/hubs need a separate fill pass via a deps.Read callback; that
// happens in the renderer to keep this stage pure.
func mergeProvenances(
	directHits []scoredCandidate,
	subgraph expandedSubgraph,
	clusters clusterReport,
	hubs hubReport,
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

	mergeClusterReps(subgraph, clusters, byBasename)

	mergeHubItems(subgraph, hubs, byBasename)

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

// nearestInTierIndex returns the index note nearest the centroid by the
// stronger of its two axes (the "either axis" gate). found is false for an
// empty index.
func nearestInTierIndex(centroid []float32, idx tierIndex) (string, float32, bool) {
	best, bestSim := -1, float32(-1)

	for i := range idx.paths {
		sim := eitherAxisCosine(centroid, idx.sit[i], idx.body[i])

		if sim > bestSim {
			bestSim = sim
			best = i
		}
	}

	if best < 0 {
		return "", 0, false
	}

	return idx.paths[best], bestSim, true
}

// nearestL3For returns the nearest L3 note to centroid from l3Notes by
// max(situation,body), or nil if the index is empty. No threshold is applied;
// the skill applies its own 0.9 cut.
func nearestL3For(centroid []float32, l3Notes tierIndex) *queryNearestL3 {
	path, cosine, found := nearestInTierIndex(centroid, l3Notes)
	if !found {
		return nil
	}

	return &queryNearestL3{Path: path, Cosine: cosine}
}

// nearestL3ForTier gates nearestL3For on the requested tiers for T1a
// isolation. The tierIndex is L3-only by construction, so its sole result is
// suppressed whenever a non-empty tier set omits L3; an empty set or one that
// includes L3 passes through unchanged.
func nearestL3ForTier(centroid []float32, l3Notes tierIndex, tiers []string) *queryNearestL3 {
	if len(tiers) > 0 && !slices.Contains(tiers, tierL3) {
		return nil
	}

	return nearestL3For(centroid, l3Notes)
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

// pickRepresentative returns the subgraph-member index closest to the
// centroid by cosine distance. Ties broken by direct-hit score desc,
// then by lexicographic path.
func pickRepresentative(subgraph expandedSubgraph, memberIndices []int, centroid []float32) int {
	best := memberIndices[0]
	bestDist := cluster.CosineDistance(subgraph.members[best].vector, centroid)

	for _, idx := range memberIndices[1:] {
		dist := cluster.CosineDistance(subgraph.members[idx].vector, centroid)
		switch {
		case dist < bestDist:
			best = idx
			bestDist = dist
		case dist == bestDist:
			best = breakRepresentativeTie(subgraph, best, idx)
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
	case provenanceHub:
		return provenanceRankHub
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
// cluster is tagged with the phrase that produced it. l3Notes provides the
// vault-wide L3 note index for nearest-L3 annotation.
//
// tiers enforces T1a isolation across the cluster channels: members are
// constrained to the requested tier set, a cluster that empties after
// filtering is dropped entirely, and nearest_l3 (always an L3 note by
// construction) is suppressed for any tier set that omits L3. An empty tier
// set leaves all channels blended.
func renderClusters(phraseClusters []phrasedCluster, l3Notes, l2Notes tierIndex, tiers []string) []queryCluster {
	var out []queryCluster

	for _, pc := range phraseClusters {
		if pc.report.autoK.K == 0 {
			continue
		}

		for clusterID := range pc.report.autoK.K {
			members := collectClusterMembers(pc.subgraph, pc.report, clusterID, tiers)
			if len(members) == 0 {
				continue
			}

			centroid := pc.report.autoK.Centroids[clusterID]

			out = append(out, queryCluster{
				ID:           clusterID,
				Phrase:       pc.phrase,
				Size:         len(members),
				Silhouette:   pc.report.silhouettesByID[clusterID],
				Members:      members,
				NearestL3:    nearestL3ForTier(centroid, l3Notes, tiers),
				CandidateL2s: topKCandidateL2sForTier(centroid, l2Notes, tiers),
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
// item carries the basenames one hop away (for follow-on `engram show`).
func renderItems(resolved []resolvedItem, outgoing map[string][]string) []queryItem {
	items := make([]queryItem, len(resolved))

	for i, item := range resolved {
		kind := item.kind
		if kind == "" {
			kind = kindFromContent(item.content)
		}

		// activated: true iff this is a note (not a chunk) with a pre-decay
		// baseScore at or above the activation cutoff. Chunks are crystallized
		// by the recall skill (not by the binary), so they are never marked.
		// Query stays read-only — no sidecar writes here.
		activated := kind != chunkItemKind && item.baseScore >= activationCosineCutoff

		items[i] = queryItem{
			Path:          item.notePath,
			Kind:          kind,
			Score:         item.score,
			Provenances:   item.provenances,
			ClusterID:     item.clusterID,
			InDegree:      item.inDegree,
			OutboundLinks: outgoing[basenameFromNotePath(item.notePath)],
			Content:       item.content,
			Activated:     activated,
		}
	}

	return items
}

// renderQueryPayload encodes the resolved YAML payload for the multi-phrase
// pipeline output.
func renderQueryPayload(stdout io.Writer, merged aggregatedSummary) error {
	items := renderItems(merged.resolvedItems, merged.outgoing)
	clusters := renderClusters(merged.phraseClusters, merged.l3, merged.l2, merged.tiers)
	clusters = append(clusters, merged.chunkClusters...)
	contentful := countItemsWithContent(items)

	directCount := 0
	hubCount := 0

	for _, item := range items {
		if slices.Contains(item.Provenances, provenanceDirect) {
			directCount++
		}

		if item.InDegree != nil {
			hubCount++
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
			SubgraphSize:         merged.subgraphSize,
			SubgraphSizeCapped:   merged.subgraphCapped,
			HopsTraversed:        merged.hopsTraversed,
			ClustersFound:        len(clusters),
			HubsReturned:         hubCount,
			DirectHitsReturned:   directCount,
			ItemsWithFullContent: contentful,
			Limit:                merged.limit,
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

// runSinglePhraseQuery runs the full per-phrase pipeline for one phrase
// and returns a queryPipelineSummary. notes and hits are already loaded
// (shared across all phrases in a multi-phrase run).
func runSinglePhraseQuery(
	ctx context.Context,
	phrase string,
	notes []vaultgraph.Note,
	hits []compatibleSidecar,
	vault string,
	limit int,
	deps QueryDeps,
) (queryPipelineSummary, error) {
	queryVec, qErr := deps.Embedder.Embed(ctx, phrase)
	if qErr != nil {
		return queryPipelineSummary{}, fmt.Errorf("query: embed: %w", qErr)
	}

	var now time.Time
	if deps.Now != nil {
		now = deps.Now()
	}

	directHits := rankCandidates(hits, vault, deps.Read, queryVec, now)
	if len(directHits) > limit {
		directHits = directHits[:limit]
	}

	subgraph := expandSubgraph(notes, hits, directHits, vault, deps.Read, queryVec)
	clusters := clusterSubgraph(subgraph, phrase)
	hubs := identifyHubs(subgraph)
	resolved := mergeProvenances(directHits, subgraph, clusters, hubs)

	return queryPipelineSummary{
		subgraph:       subgraph,
		clusters:       clusters,
		resolvedItems:  resolved,
		totalNotes:     len(notes),
		withEmbeddings: len(hits),
	}, nil
}

// runSynthesisQuery implements `engram query --synthesis`: it unions every
// phrase's DIRECT HITS (semantic matches, truncated to limit — not the
// graph-expansion neighbors), deduplicates by note path keeping the max score,
// and clusters that union ONE time. Unlike the per-phrase pipeline it skips the
// minSubgraphForClustering floor and never returns "no clusters" for a
// non-empty union: when AutoK finds no split that beats the silhouette floor it
// emits a single cluster of all union members (the §6b L3-synthesis invariant).
// items[] is the deduped union direct hits; tier/project filters still apply.
func runSynthesisQuery(
	ctx context.Context,
	args QueryArgs,
	notes []vaultgraph.Note,
	hits []compatibleSidecar,
	limit int,
	deps QueryDeps,
	stdout io.Writer,
) error {
	var nowSynthesis time.Time
	if deps.Now != nil {
		nowSynthesis = deps.Now()
	}

	union, err := unionDirectHits(ctx, args.Phrases, hits, args.VaultPath, limit, nowSynthesis, deps)
	if err != nil {
		return err
	}

	subgraph := buildUnionSubgraph(union)
	report := clusterUnionForSynthesis(subgraph, strings.Join(args.Phrases, "\n"))

	resolved := mergeProvenances(union, expandedSubgraph{}, clusterReport{}, hubReport{})
	resolved = applyProjectFilter(resolved, args.Project)
	resolved = applyTierFilter(resolved, args.Tiers)

	merged := aggregatedSummary{
		phrases:        args.Phrases,
		resolvedItems:  resolved,
		phraseClusters: []phrasedCluster{{phrase: synthesisClusterPhrase, report: report, subgraph: subgraph}},
		l3:             gatherL3Index(hits, args.VaultPath, deps.Read),
		outgoing:       outgoingByBasename(notes),
		tiers:          args.Tiers,
		totalNotes:     len(notes),
		withEmbeddings: len(hits),
		limit:          limit,
		subgraphSize:   len(subgraph.members),
	}

	return renderQueryPayload(stdout, merged)
}

// runSynthesizeL2Query mirrors runSynthesisQuery for the lazy-L2 path. It
// constrains the CLUSTERED set to matched L1+L2 notes (L3 excluded from
// clusters), then emits raw candidate_l2s [{path, cosine}] per cluster — the
// max(situation,body) cosine from the cluster centroid to the nearest existing
// L2 in the vault. No band decision happens here; the recall skill bands it.
// The L2 index is gathered from the FULL hits (every L2 in the vault is a
// candidate nearest), while the clustered set is only the matched L1+L2.
func runSynthesizeL2Query(
	ctx context.Context,
	args QueryArgs,
	notes []vaultgraph.Note,
	hits []compatibleSidecar,
	limit int,
	deps QueryDeps,
	stdout io.Writer,
) error {
	l1l2Hits := filterHitsToTiers(hits, args.VaultPath, deps.Read, []string{tierL1, tierL2})

	var nowL2 time.Time
	if deps.Now != nil {
		nowL2 = deps.Now()
	}

	union, err := unionDirectHits(ctx, args.Phrases, l1l2Hits, args.VaultPath, limit, nowL2, deps)
	if err != nil {
		return err
	}

	// D1: build the subgraph from the note union, then extend with matched chunks
	// so one AutoK pass clusters notes and chunks together.
	subgraph := buildUnionSubgraph(union)

	var chunkItems []resolvedItem // collected for items[] only

	if chunksConfigured(args, deps) {
		chunkItems, err = appendSynthesisChunks(ctx, args, deps, &subgraph, limit)
		if err != nil {
			return err
		}
	}

	report := clusterUnionForSynthesis(subgraph, strings.Join(args.Phrases, "\n"))

	// mergeProvenances receives an empty expandedSubgraph{} deliberately:
	// mergeClusterReps/mergeHubItems must not promote cluster reps into items[]
	// because the L2 representative is agent-decided, not binary-computed (spec §2
	// step 4). Direct-hit items come from the union; chunk items are appended below.
	resolved := mergeProvenances(union, expandedSubgraph{}, clusterReport{}, hubReport{})
	resolved = applyProjectFilter(resolved, args.Project)
	resolved = append(resolved, chunkItems...)

	merged := aggregatedSummary{
		phrases:        args.Phrases,
		resolvedItems:  resolved,
		phraseClusters: []phrasedCluster{{phrase: synthesisClusterPhrase, report: report, subgraph: subgraph}},
		l3:             tierIndex{}, // L3 not emitted in this mode
		l2:             gatherTierIndex(hits, args.VaultPath, deps.Read, tierL2),
		outgoing:       outgoingByBasename(notes),
		tiers:          nil,
		totalNotes:     len(notes),
		withEmbeddings: len(hits),
		limit:          limit,
		subgraphSize:   len(subgraph.members),
	}

	return renderQueryPayload(stdout, merged)
}

// seedBasenames extracts seed basenames from direct hits in the order they appear.
func seedBasenames(directHits []scoredCandidate) []string {
	out := make([]string, 0, len(directHits))
	for _, hit := range directHits {
		out = append(out, hit.basename)
	}

	return out
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
func singleClusterReport(subgraph expandedSubgraph, vectors [][]float32) clusterReport {
	allIndices := make([]int, len(subgraph.members))
	for i := range allIndices {
		allIndices[i] = i
	}

	centroid := meanVector(vectors)
	rep := pickRepresentative(subgraph, allIndices, centroid)

	return clusterReport{
		autoK:           cluster.AutoKResult{K: 1, Centroids: [][]float32{centroid}},
		memberIDs:       [][]int{allIndices},
		representatives: []int{rep},
		silhouettesByID: []float64{singletonClusterSilhouette},
	}
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

// survivingChunks filters the scored chunks down to those that made the final
// ranking. Clustering ONLY the returned set matters: k-means over the whole
// index produces meaninglessly huge clusters (and a payload to match) — the
// returned set is what recall reasons over.
func survivingChunks(scored []scoredChunk, items []resolvedItem) []scoredChunk {
	returned := make(map[string]struct{}, len(items))

	for _, item := range items {
		if item.kind == chunkItemKind {
			returned[item.notePath] = struct{}{}
		}
	}

	top := make([]scoredChunk, 0, len(returned))

	for _, s := range scored {
		if _, ok := returned[chunkNotePath(s.record)]; ok {
			top = append(top, s)
		}
	}

	return top
}

// topKCandidateL2s returns the top-K L2 notes nearest the centroid by
// max(situation,body) cosine, sorted descending by centroid cosine (ties broken
// by lexicographic path for stability). K is at least candidateL2K; when fewer
// than candidateL2K L2 notes exist, all are returned. An empty index returns
// nil. No cosine threshold is applied — nomination is generous (D7). The sort
// key is CENTROID cosine (per spec §3.3: "top-K by centroid cosine");
// max-member cosine was rejected because it overfits to a cluster fragment.
func topKCandidateL2s(centroid []float32, idx tierIndex) []queryCandidateL2 {
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

	count := min(candidateL2K, len(all))

	out := make([]queryCandidateL2, count)
	for i := range count {
		out[i] = queryCandidateL2{Path: all[i].path, Cosine: all[i].cosine}
	}

	return out
}

// topKCandidateL2sForTier gates topKCandidateL2s on the requested tiers for
// T1a isolation. Suppressed when a non-empty tier set omits L2; nil/empty
// tiers always passes through (--synthesize-l2 passes nil).
func topKCandidateL2sForTier(centroid []float32, l2Notes tierIndex, tiers []string) []queryCandidateL2 {
	if len(tiers) > 0 && !slices.Contains(tiers, tierL2) {
		return nil
	}

	return topKCandidateL2s(centroid, l2Notes)
}

// unionDirectHits embeds each phrase, ranks every compatible note against it,
// truncates to the top `limit` direct hits, then merges all phrases' hits into
// one deduped set keyed by note path (max score wins). The result is sorted by
// note path so downstream clustering and tie-breaks are deterministic.
//
// now is forwarded to rankCandidates for note recency decay; a zero value
// disables decay (pure cosine).
func unionDirectHits(
	ctx context.Context,
	phrases []string,
	hits []compatibleSidecar,
	vault string,
	limit int,
	now time.Time,
	deps QueryDeps,
) ([]scoredCandidate, error) {
	byPath := make(map[string]scoredCandidate)

	for _, phrase := range phrases {
		queryVec, embedErr := deps.Embedder.Embed(ctx, phrase)
		if embedErr != nil {
			return nil, fmt.Errorf("query: embed: %w", embedErr)
		}

		directHits := rankCandidates(hits, vault, deps.Read, queryVec, now)
		if len(directHits) > limit {
			directHits = directHits[:limit]
		}

		for _, hit := range directHits {
			existing, ok := byPath[hit.notePath]
			if !ok || hit.score > existing.score {
				byPath[hit.notePath] = hit
			}
		}
	}

	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}

	sort.Strings(paths)

	union := make([]scoredCandidate, 0, len(byPath))
	for _, path := range paths {
		union = append(union, byPath[path])
	}

	return union, nil
}

// validateQueryArgs rejects invalid invocations before any vault I/O runs, so
// argument errors take precedence over data-state guards (e.g. notes-but-no-
// embeddings). It enforces a non-empty phrase set and the --synthesis /
// --synthesize-l2 mutual exclusion.
func validateQueryArgs(args QueryArgs) error {
	if len(args.Phrases) == 0 {
		return errQueryEmptyString
	}

	if args.Synthesis && args.SynthesizeL2 {
		return errQueryModeConflict
	}

	return nil
}

// warnModelMismatch emits a single aggregated advisory (M4) when sidecars
// were dropped because their embedding model differs from the active one.
// A no-op when nothing mismatched or no warning hook is wired. The message
// names the dropped count and the distinct stale model id(s) so a silent
// model swap that empties recall is surfaced instead of hidden.
func warnModelMismatch(logWarning func(string, ...any), loaded sidecarLoadResult, activeModelID string) {
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
