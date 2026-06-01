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

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cluster"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// QueryArgs holds parsed flags for `engram query`.
type QueryArgs struct {
	Phrases   []string `targ:"flag,name=phrase,desc=query phrase (repeatable)"`
	VaultPath string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root"`
	Limit     int      `targ:"flag,name=limit,desc=max number of items to return (default 20)"`
	Project   string   `targ:"flag,name=project,desc=restrict items to notes with matching project: field (optional)"`
	Tier      string   `targ:"flag,name=tier,desc=restrict items to notes with matching tier: field (optional)"`
}

// QueryDeps holds injected dependencies for the query command.
type QueryDeps struct {
	Scan     func(vault string) ([]vaultgraph.Note, error)
	Read     func(path string) ([]byte, error)
	Embedder embed.Embedder
}

// RunQuery embeds each query phrase, scores it against every note that
// has a current-model sidecar, ranks by descending cosine, expands a
// 3-hop subgraph over authored wikilinks, clusters that subgraph, and
// identifies hubs by in-degree before emitting the resolved YAML
// payload per the F6+F9.1 spec.
func RunQuery(ctx context.Context, args QueryArgs, deps QueryDeps, stdout io.Writer) error {
	phrases := args.Phrases
	if len(phrases) == 0 {
		return errQueryEmptyString
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
	hits := loadCompatibleSidecars(notes, args.VaultPath, deps.Read, modelID)

	if len(notes) > 0 && len(hits) == 0 {
		return errQueryNoEmbeddings
	}

	summaries := make([]queryPipelineSummary, 0, len(phrases))

	for _, phrase := range phrases {
		summary, err := runSinglePhraseQuery(ctx, phrase, notes, hits, args.VaultPath, limit, deps)
		if err != nil {
			return err
		}

		summaries = append(summaries, summary)
	}

	l3 := gatherL3Index(hits, args.VaultPath, deps.Read)

	merged := aggregatePhraseSummaries(phrases, summaries, limit)
	merged.l3 = l3
	merged.resolvedItems = applyProjectFilter(merged.resolvedItems, args.Project)
	merged.resolvedItems = applyTierFilter(merged.resolvedItems, args.Tier)

	return renderQueryPayload(stdout, merged)
}

// unexported constants.
const (
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
	subgraphCap              = 200
	subgraphMaxHops          = 3
	unknownKind              = "unknown"
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
	l3             l3Index
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

// l3Index holds the vault-wide set of L3 note paths and their sidecar
// vectors for per-cluster nearest-L3 lookup. Built once per RunQuery call.
type l3Index struct {
	paths   []string
	vectors [][]float32
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

// queryCluster is the cluster shape in the payload.
type queryCluster struct {
	ID         int                  `yaml:"id"`
	Phrase     string               `yaml:"phrase"`
	Size       int                  `yaml:"size"`
	Silhouette float64              `yaml:"silhouette"`
	Members    []queryClusterMember `yaml:"members"`
	NearestL3  *queryNearestL3      `yaml:"nearest_l3,omitempty"`
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
type queryItem struct {
	Path        string   `yaml:"path"`
	Kind        string   `yaml:"kind"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	ClusterID   *int     `yaml:"cluster_id,omitempty"`
	InDegree    *int     `yaml:"in_degree,omitempty"`
	Content     string   `yaml:"content,omitempty"`
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
	provenances []string
	clusterID   *int
	inDegree    *int
}

// scoredCandidate aggregates one note's match against the query vector.
type scoredCandidate struct {
	notePath string
	basename string
	score    float32
	content  string
}

// subgraphMember bundles a node's basename, vault-relative path,
// sidecar vector, query-similarity score, and (optionally) cached body.
type subgraphMember struct {
	basename string
	notePath string
	vector   []float32
	score    float32
	content  string
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

// appendUniqueProvenance adds role to item.provenances iff not already present.
func appendUniqueProvenance(item *resolvedItem, role string) {
	if slices.Contains(item.provenances, role) {
		return
	}

	item.provenances = append(item.provenances, role)
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

// applyTierFilter drops items whose frontmatter tier: field doesn't match
// the requested tier label. Empty tier is a no-op (returns items unchanged).
// Items with no loaded content cannot be verified and are dropped when a
// non-empty tier is specified.
func applyTierFilter(items []resolvedItem, tier string) []resolvedItem {
	if tier == "" {
		return items
	}

	out := make([]resolvedItem, 0, len(items))

	for _, item := range items {
		if itemMatchesTier(item, tier) {
			out = append(out, item)
		}
	}

	return out
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
		notePath := pathOf(name, hit.note.IsMOC)

		member := subgraphMember{
			basename: name,
			notePath: notePath,
			vector:   hit.sidecar.Vector,
			score:    embed.Cosine(queryVec, hit.sidecar.Vector),
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

// collectClusterMembers gathers per-cluster member rows in score-desc
// order, marking the representative.
func collectClusterMembers(
	subgraph expandedSubgraph,
	report clusterReport,
	clusterID int,
) []queryClusterMember {
	memberIndices := make([]int, len(report.memberIDs[clusterID]))
	copy(memberIndices, report.memberIDs[clusterID])

	sort.SliceStable(memberIndices, func(i, j int) bool {
		return subgraph.members[memberIndices[i]].score > subgraph.members[memberIndices[j]].score
	})

	members := make([]queryClusterMember, 0, len(memberIndices))
	repIdx := report.representatives[clusterID]

	for _, idx := range memberIndices {
		member := subgraph.members[idx]
		members = append(members, queryClusterMember{
			Path:             member.notePath,
			Score:            member.score,
			IsRepresentative: idx == repIdx,
		})
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

// gatherL3Index reads each compatible-sidecar note's content and collects
// those whose frontmatter carries "tier: L3" into a small index used for
// nearest-L3 lookup during cluster rendering. Called once per query so the
// per-cluster BestMatch calls are O(1) in I/O.
func gatherL3Index(
	hits []compatibleSidecar,
	vault string,
	read func(string) ([]byte, error),
) l3Index {
	paths := make([]string, 0)
	vectors := make([][]float32, 0)

	for _, hit := range hits {
		notePath := pathOf(hit.note.Basename, hit.note.IsMOC)

		body, readErr := read(filepath.Join(vault, notePath))
		if readErr != nil {
			continue
		}

		item := resolvedItem{content: stripWikilinks(string(body))}
		if !itemMatchesTier(item, "L3") {
			continue
		}

		paths = append(paths, notePath)
		vectors = append(vectors, hit.sidecar.Vector)
	}

	return l3Index{paths: paths, vectors: vectors}
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
// tier: L<n> line matching the requested tier label. Returns false when
// content is missing or when the frontmatter block is malformed.
func itemMatchesTier(item resolvedItem, tier string) bool {
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

	return len(match) == 2 && match[1] == tier
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
// Missing, malformed, or incompatible sidecars are silently skipped — the
// missing-coverage case (none compatible) is surfaced by RunQuery's guard.
func loadCompatibleSidecars(
	notes []vaultgraph.Note,
	vault string,
	read func(string) ([]byte, error),
	modelID string,
) []compatibleSidecar {
	hits := make([]compatibleSidecar, 0, len(notes))

	for _, note := range notes {
		notePath := pathOf(note.Basename, note.IsMOC)
		scFull := filepath.Join(vault, embed.SidecarPath(notePath))

		scBytes, readErr := read(scFull)
		if readErr != nil {
			continue
		}

		sidecar, parseErr := embed.UnmarshalSidecar(scBytes)
		if parseErr != nil {
			continue
		}

		if sidecar.EmbeddingModelID != modelID {
			continue
		}

		hits = append(hits, compatibleSidecar{note: note, sidecar: sidecar})
	}

	return hits
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

// mergeIntoExisting updates existing with the best score, unioned
// provenances, and in_degree from src (if existing has none).
// in_degree is not maximised across phrases because undirected BFS
// always reaches the same linkers for a note regardless of starting
// point, so both phrases produce identical in_degrees.
func mergeIntoExisting(existing, src *resolvedItem) {
	if src.score > existing.score {
		existing.score = src.score
		existing.content = src.content
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
				notePath: hit.notePath,
				content:  hit.content,
				score:    hit.score,
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

// nearestL3For returns the nearest L3 note to centroid from l3Notes, or nil if
// the index is empty. threshold 0 ensures the best is always reported; the
// skill applies its own 0.9 cut.
func nearestL3For(centroid []float32, l3Notes l3Index) *queryNearestL3 {
	if len(l3Notes.vectors) == 0 {
		return nil
	}

	bestIdx, sim := cluster.BestMatch(centroid, l3Notes.vectors, 0)

	result := &queryNearestL3{Cosine: sim}
	if bestIdx >= 0 {
		result.Path = l3Notes.paths[bestIdx]
	}

	return result
}

// newOsQueryDeps wires the production scan + read for the query command.
func newOsQueryDeps() QueryDeps {
	embedDeps := newOsEmbedDeps()

	return QueryDeps{
		Scan:     embedDeps.Scan,
		Read:     embedDeps.Read,
		Embedder: embedDeps.Embedder,
	}
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
func rankCandidates(
	hits []compatibleSidecar,
	vault string,
	read func(string) ([]byte, error),
	queryVec []float32,
) []scoredCandidate {
	candidates := make([]scoredCandidate, 0, len(hits))

	for _, hit := range hits {
		notePath := pathOf(hit.note.Basename, hit.note.IsMOC)
		full := filepath.Join(vault, notePath)

		noteBytes, noteErr := read(full)
		if noteErr != nil {
			continue
		}

		candidates = append(candidates, scoredCandidate{
			notePath: notePath,
			basename: hit.note.Basename,
			score:    embed.Cosine(queryVec, hit.sidecar.Vector),
			content:  stripWikilinks(string(noteBytes)),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	return candidates
}

// renderClusters converts a clusterReport into the YAML clusters[] section.
// Members are sorted by score desc; representative is_representative is set.
// renderClusters converts per-phrase cluster reports into the YAML wire shape.
// Each cluster is tagged with the phrase that produced it. l3Notes provides the
// vault-wide L3 note index for nearest-L3 annotation.
func renderClusters(phraseClusters []phrasedCluster, l3Notes l3Index) []queryCluster {
	var out []queryCluster

	for _, pc := range phraseClusters {
		if pc.report.autoK.K == 0 {
			continue
		}

		for clusterID := range pc.report.autoK.K {
			members := collectClusterMembers(pc.subgraph, pc.report, clusterID)
			centroid := pc.report.autoK.Centroids[clusterID]

			out = append(out, queryCluster{
				ID:         clusterID,
				Phrase:     pc.phrase,
				Size:       len(members),
				Silhouette: pc.report.silhouettesByID[clusterID],
				Members:    members,
				NearestL3:  nearestL3For(centroid, l3Notes),
			})
		}
	}

	if out == nil {
		return []queryCluster{}
	}

	return out
}

// renderItems converts resolved items into the YAML wire-shape items.
func renderItems(resolved []resolvedItem) []queryItem {
	items := make([]queryItem, len(resolved))

	for i, item := range resolved {
		items[i] = queryItem{
			Path:        item.notePath,
			Kind:        kindFromContent(item.content),
			Score:       item.score,
			Provenances: item.provenances,
			ClusterID:   item.clusterID,
			InDegree:    item.inDegree,
			Content:     item.content,
		}
	}

	return items
}

// renderQueryPayload encodes the resolved YAML payload for the multi-phrase
// pipeline output.
func renderQueryPayload(stdout io.Writer, merged aggregatedSummary) error {
	items := renderItems(merged.resolvedItems)
	clusters := renderClusters(merged.phraseClusters, merged.l3)
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

	directHits := rankCandidates(hits, vault, deps.Read, queryVec)
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
