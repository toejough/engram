package cli_test

// Scenario tests for the Memory Query product capability, part 2 (861c2zm), first half: the clusters (792a7g),
// the candidate notes (792a7h) and the superseding notes that ride along (792a7n). The explore half (792a7i to
// 792a7m) is in memory_query_part2_explore_scenarios_test.go, which also holds nothing the first half needs. Spec:
// docs/specs/product/memory-query.md.
//
// The scenarios run cli.RunQuery over the in-memory world of part 1 (mq1World: an in-memory vault and chunk index
// and a phrase-keyed embedder, phrase i embeds to the unit vector of axis i), so every cosine in a scenario is a
// vector the test chose. The helpers of this part carry the mq2 prefix. The expected cosines and clusters are
// computed in the test (mq2Cosine, mq2Mean), never taken from the code under test.

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// Test792a7g1b_RunQuery_MatchesWithNoStructureFormOneCluster verifies 792a7g1b (Matches with no structure form
// one cluster): three notes and a chunk whose vectors are one and the same vector cannot be split, so clusters
// holds one cluster that holds every match, with silhouette 0.
func Test792a7g1b_RunQuery_MatchesWithNoStructureFormOneCluster(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given matches too alike to split.
	alike := mq2Vec(mq2Main, 0, mq2Side, 0)
	world := newMq1World(mq2Dims, mq2GoPhrase)

	noteNames := []string{"1.alike-a", "2.alike-b", "3.alike-c"}
	for _, name := range noteNames {
		mq2PlantNote(world, name, alike, "")
	}

	world.chunks = append(world.chunks, mq1ChunkRecord(mq2AnchorOne, alike, world.clock()))

	// When the user runs engram query with a phrase that matches them.
	result := mq2MustRun(t, world)

	// Then clusters holds one cluster holding every match, with silhouette 0.
	g.Expect(result.payload.Clusters).To(HaveLen(1))

	if len(result.payload.Clusters) != 1 {
		return
	}

	only := result.payload.Clusters[0]
	g.Expect(only.Silhouette).To(BeZero())
	g.Expect(only.Size).To(Equal(len(noteNames) + 1))
	g.Expect(only.memberPaths()).To(ConsistOf(
		"1.alike-a.md", "2.alike-b.md", "3.alike-c.md", mq1ChunkPath(mq2AnchorOne)))
}

// Test792a7g1c_RunQuery_NothingMatchedMeansNoClusters verifies 792a7g1c (Nothing matched means no clusters): a
// vault whose only note has a cosine of 0.1 to the phrase, below the floor, gives no item and an empty clusters.
func Test792a7g1c_RunQuery_NothingMatchedMeansNoClusters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const belowFloor = 0.1

	// Given a vault with no note above the floor.
	world := newMq1World(mq2Dims, mq2GoPhrase)
	mq2PlantNote(world, "1.far-off", world.cosines(map[int]float64{0: belowFloor}), "")

	// When the user runs engram query with a phrase.
	result := mq2MustRun(t, world)

	// Then clusters is empty.
	g.Expect(result.payload.Clusters).To(BeEmpty())
	g.Expect(result.payload.Budget.ClustersFound).To(BeZero())
	g.Expect(result.stdout).To(ContainSubstring("clusters: []"))
	g.Expect(result.payload.Items).To(BeEmpty(), "the vault had nothing above the floor")
}

// Test792a7g1d_RunQuery_TheSameQueryGivesTheSameClusters verifies 792a7g1d (The same query gives the same
// clusters): a vault with two topics, queried again and again with the same two phrases, gives the same clusters
// every time; each run is over a freshly built vault, so the order the notes are scanned in differs between them.
func Test792a7g1d_RunQuery_TheSameQueryGivesTheSameClusters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault and a set of phrases.
	build := func() *mq1World {
		world := newMq1World(mq2Dims, mq2GoPhrase, mq2BakingPhrase)

		for index, deviation := range []float32{0, 0.02, -0.04, 0.06} {
			mq2PlantNote(world, fmt.Sprintf("%d.go-%d", index+1, index), mq2Vec(mq2Main, 0, deviation, 0), "")
			mq2PlantNote(world, fmt.Sprintf("%d.baking-%d", index+5, index), mq2Vec(0, mq2Main, 0, deviation), "")
		}

		world.chunks = append(world.chunks,
			mq1ChunkRecord(mq2AnchorOne, mq2Vec(0, mq2Main, 0, mq2Side), world.clock()))

		return world
	}

	// When the user runs engram query with them, and again (many times, so a seed that changed would show).
	const repeats = 12

	first := mq2MustRun(t, build())

	// Then every payload holds the same clusters.
	g.Expect(first.payload.Clusters).To(HaveLen(2), "the two topics are two clusters, so the comparison is not empty")

	for run := range repeats {
		again := mq2MustRun(t, build())
		g.Expect(again.payload.Clusters).To(Equal(first.payload.Clusters), "run %d", run)
	}
}

// Test792a7h1a_RunQuery_CandidatesAreTheClustersNearestNotes verifies 792a7h1a (Candidates are the cluster's
// nearest notes): a cluster of seven notes and three chunks, three of which chunks are nearer the centroid than
// the fifth note, lists five of the notes as candidate_l2s, ordered by cosine to the centroid and in the order
// the test computes them from the vectors, and lists no chunk.
func Test792a7h1a_RunQuery_CandidatesAreTheClustersNearestNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const nearestFive = 5

	// Given a cluster of seven notes and three chunks, and another cluster.
	noteDeviations := []float32{0, 0.02, -0.04, 0.06, -0.08, 0.10, -0.12}
	chunkDeviations := []float32{0.001, -0.01, 0.03}

	world := newMq1World(mq2Dims, mq2GoPhrase)
	clusterVectors := make([][]float32, 0, len(noteDeviations)+len(chunkDeviations))
	notePaths := make([]string, 0, len(noteDeviations))
	chunkPaths := make([]string, 0, len(chunkDeviations))

	for index, deviation := range noteDeviations {
		name := fmt.Sprintf("%d.near-%d", index+1, index)
		vec := mq2Vec(mq2Main, 0, deviation, 0)

		mq2PlantNote(world, name, vec, "")

		notePaths = append(notePaths, name+".md")
		clusterVectors = append(clusterVectors, vec)
	}

	for index, deviation := range chunkDeviations {
		anchor := mq1RampAnchor(index)
		vec := mq2Vec(mq2Main, 0, deviation, 0)

		world.chunks = append(world.chunks, mq1ChunkRecord(anchor, vec, world.clock()))

		chunkPaths = append(chunkPaths, mq1ChunkPath(anchor))
		clusterVectors = append(clusterVectors, vec)
	}

	mq2PlantOtherTopic(world, 3)

	// When the user runs engram query with a phrase that matches them.
	result := mq2MustRun(t, world)

	// Then candidate_l2s lists five of the notes, ordered by cosine to the centroid, and no chunk.
	cluster, found := result.payload.clusterOf(notePaths[0])
	g.Expect(found).To(BeTrue())

	if !found {
		return
	}

	g.Expect(cluster.Size).To(Equal(len(noteDeviations)+len(chunkDeviations)), "the cluster holds all ten")

	centroid := mq2Mean(clusterVectors)
	wantOrder := mq2ByCosineTo(centroid, notePaths, clusterVectors[:len(notePaths)])

	g.Expect(cluster.candidatePaths()).To(Equal(wantOrder[:nearestFive]))

	for _, candidate := range cluster.CandidateL2s {
		g.Expect(chunkPaths).NotTo(ContainElement(candidate.Path))
	}

	// The check is not vacuous: the nearest chunk is nearer the centroid than the fifth note.
	fifthNote := clusterVectors[slices.Index(notePaths, wantOrder[nearestFive-1])]
	nearestChunk := clusterVectors[len(notePaths)]
	g.Expect(mq2Cosine(centroid, nearestChunk)).To(BeNumerically(">", mq2Cosine(centroid, fifthNote)))

	for index, candidate := range cluster.CandidateL2s {
		want := mq2Cosine(centroid, clusterVectors[slices.Index(notePaths, wantOrder[index])])
		g.Expect(float64(candidate.Cosine)).To(BeNumerically("~", want, mq2Tolerance), candidate.Path)
	}
}

// Test792a7h1b_RunQuery_AClusterOfChunksHasNoCandidates verifies 792a7h1b (A cluster of chunks has no
// candidates): a cluster whose members are all chunks has no candidate_l2s, while the cluster of notes beside it
// has some.
func Test792a7h1b_RunQuery_AClusterOfChunksHasNoCandidates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a cluster whose members are all chunks, and a cluster of notes.
	world := newMq1World(mq2Dims, mq2GoPhrase)

	for index, deviation := range []float32{0, 0.02, -0.04, 0.06} {
		mq2PlantNote(world, fmt.Sprintf("%d.note-%d", index+1, index), mq2Vec(mq2Main, 0, deviation, 0), "")
	}

	chunkPaths := make([]string, 0, mq2OtherTopicSize)

	for index := range mq2OtherTopicSize {
		anchor := mq1RampAnchor(index)
		vec := mq2OtherTopicVec(float32(index) * mq2OtherTopicStep)

		world.chunks = append(world.chunks, mq1ChunkRecord(anchor, vec, world.clock()))
		chunkPaths = append(chunkPaths, mq1ChunkPath(anchor))
	}

	// When the user runs engram query with a phrase that matches them.
	result := mq2MustRun(t, world)

	// Then that cluster has no candidate_l2s.
	chunkCluster, chunksFound := result.payload.clusterOf(chunkPaths[0])
	g.Expect(chunksFound).To(BeTrue())

	noteCluster, notesFound := result.payload.clusterOf("1.note-0.md")
	g.Expect(notesFound).To(BeTrue())

	if !chunksFound || !notesFound {
		return
	}

	g.Expect(chunkCluster.memberPaths()).To(ConsistOf(chunkPaths))
	g.Expect(chunkCluster.CandidateL2s).To(BeEmpty())
	g.Expect(noteCluster.CandidateL2s).NotTo(BeEmpty())
	g.Expect(strings.Count(result.stdout, "candidate_l2s:")).To(Equal(1), "only the cluster of notes has the key")
}

// Test792a7h1c_RunQuery_ASmallClusterListsAllItsNotes verifies 792a7h1c (A small cluster lists all its notes): a
// cluster of three notes lists all three as candidate_l2s, nearest the centroid first.
func Test792a7h1c_RunQuery_ASmallClusterListsAllItsNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a cluster of three notes, and another cluster.
	deviations := []float32{0.05, -0.01, 0.12}
	world := newMq1World(mq2Dims, mq2GoPhrase)
	paths := make([]string, 0, len(deviations))
	vectors := make([][]float32, 0, len(deviations))

	for index, deviation := range deviations {
		name := fmt.Sprintf("%d.small-%d", index+1, index)
		vec := mq2Vec(mq2Main, 0, deviation, 0)

		mq2PlantNote(world, name, vec, "")

		paths = append(paths, name+".md")
		vectors = append(vectors, vec)
	}

	mq2PlantOtherTopic(world, mq2OtherTopicSize)

	// When the user runs engram query with a phrase that matches them.
	result := mq2MustRun(t, world)

	// Then candidate_l2s lists all three.
	cluster, found := result.payload.clusterOf(paths[0])
	g.Expect(found).To(BeTrue())

	if !found {
		return
	}

	g.Expect(cluster.Size).To(Equal(len(deviations)))
	g.Expect(cluster.candidatePaths()).To(Equal(mq2ByCosineTo(mq2Mean(vectors), paths, vectors)))
	g.Expect(cluster.CandidateL2s).To(HaveLen(len(deviations)))
}

// Test792a7n1a_RunQuery_TheNewerNoteFollowsTheOlderOne verifies 792a7n1a (The newer note follows the older one):
// a note N that supersedes the matched note P but does not match the phrase is the item directly after P, with
// the provenance ride_along and a score of 0; the matched notes around P keep their order.
func Test792a7n1a_RunQuery_TheNewerNoteFollowsTheOlderOne(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a note P matched by the phrase, and a note N that supersedes P but does not match.
	world := mq2RideAlongWorld()
	mq2PlantNote(world, "4.newer", mq2Unmatched, mq2SupersedesOlder())

	// When the user runs engram query with that phrase.
	result := mq2MustRun(t, world)

	// Then N is the item directly after P, with provenance ride_along and score 0.
	g.Expect(result.payload.paths()).To(Equal([]string{"1.top.md", "2.older.md", "4.newer.md", "3.low.md"}))

	item, found := result.payload.find("4.newer.md")
	g.Expect(found).To(BeTrue())
	g.Expect(item.Provenances).To(Equal([]string{"ride_along"}))
	g.Expect(item.Score).To(BeZero())
	g.Expect(item.Content).To(ContainSubstring("the newer note"))
}

// Test792a7n1b_RunQuery_ANoteAlreadyPresentIsNotInsertedAgain verifies 792a7n1b (A note already present is not
// inserted again): when both P and N match, N appears once, in its own place, as a direct match.
func Test792a7n1b_RunQuery_ANoteAlreadyPresentIsNotInsertedAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given both P and N matched.
	world := mq2RideAlongWorld()
	mq2PlantNote(world, "4.newer", mq2Vec(mq2Main, 0, 0, mq2Side), mq2SupersedesOlder())

	// When the user runs engram query with a phrase that matches them.
	result := mq2MustRun(t, world)

	// Then N appears once.
	g.Expect(result.payload.paths()).To(ConsistOf("1.top.md", "2.older.md", "3.low.md", "4.newer.md"))

	item, found := result.payload.find("4.newer.md")
	g.Expect(found).To(BeTrue())
	g.Expect(item.Provenances).To(Equal([]string{"direct"}), "N stays the match it was")
}

// Test792a7n1c_RunQuery_SeveralSupersedersAreAllInserted verifies 792a7n1c (Several superseders are all
// inserted): two notes N1 and N2 that both supersede the matched note P follow P in order. The query is run over
// a freshly built vault many times, so an order that depended on the scan order or on a map would show.
func Test792a7n1c_RunQuery_SeveralSupersedersAreAllInserted(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const repeats = 24

	for run := range repeats {
		// Given two notes N1 and N2 that both supersede P.
		world := mq2RideAlongWorld()
		mq2PlantNote(world, "4.newer", mq2Unmatched, mq2SupersedesOlder())
		mq2PlantNote(world, "5.newest", mq2Unmatched, mq2SupersedesOlder())

		// When the user runs engram query with a phrase that matches P.
		result := mq2MustRun(t, world)

		// Then N1 and N2 follow P in order.
		g.Expect(result.payload.paths()).To(Equal(
			[]string{"1.top.md", "2.older.md", "4.newer.md", "5.newest.md", "3.low.md"}), "run %d", run)

		for _, path := range []string{"4.newer.md", "5.newest.md"} {
			item, found := result.payload.find(path)
			g.Expect(found).To(BeTrue())
			g.Expect(item.Provenances).To(Equal([]string{"ride_along"}), path)
		}
	}
}

// Test792a7n1d_RunQuery_AMissingSupersederIsSkipped verifies 792a7n1d (A missing superseder is skipped): when
// the note that supersedes the matched note P has no usable sidecar, whether none at all or one made by another
// model, no item is inserted and the query succeeds.
func Test792a7n1d_RunQuery_AMissingSupersederIsSkipped(t *testing.T) {
	t.Parallel()

	cases := map[string]mq1Note{
		"no sidecar":              {noSidecar: true},
		"a sidecar of old schema": {oldSchema: true},
		"a sidecar of another model": {
			modelID: mq1OtherModelID,
		},
	}

	for name, unusable := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a note P whose superseder has no usable sidecar.
			world := mq2RideAlongWorld()
			unusable.name = "4.newer"
			unusable.content = mq2NoteContent(mq2SupersedesOlder(), "the newer note")
			unusable.situation = mq2Unmatched
			unusable.body = mq2Unmatched
			world.plant(unusable)

			// When the user runs engram query with a phrase that matches P.
			result := world.query(t, cli.QueryArgs{Phrases: world.phrases})

			// Then no item is inserted and the query succeeds.
			g.Expect(result.err).NotTo(HaveOccurred())

			var payload mq2Payload

			g.Expect(yaml.Unmarshal([]byte(result.stdout), &payload)).To(Succeed())
			g.Expect(payload.paths()).To(Equal([]string{"1.top.md", "2.older.md", "3.low.md"}))
		})
	}
}

// unexported constants.
const (
	mq2AnchorOne      = "section-000"
	mq2BakingPhrase   = "how to bake sourdough bread"
	mq2Dims           = 4
	mq2GoPhrase       = "writing Go tests"
	mq2Main           = 0.9
	mq2OtherTopicSize = 3
	mq2OtherTopicStep = 0.04
	mq2Side           = 0.1
	mq2Tolerance      = 1e-4
)

// unexported variables.
var (
	// mq2Unmatched is a vector whose cosine to the first phrase is 0, so a note holding it is not matched.
	mq2Unmatched = mq2Vec(0, 1, 0, 0)
)

// mq2Candidate is one candidate_l2s entry of a cluster.
type mq2Candidate struct {
	Path   string  `yaml:"path"`
	Cosine float32 `yaml:"cosine"`
}

// mq2Cluster is one cluster of the query payload.
type mq2Cluster struct {
	ID           int                `yaml:"id"`
	Phrase       string             `yaml:"phrase"`
	Size         int                `yaml:"size"`
	Silhouette   float64            `yaml:"silhouette"`
	Members      []mq2ClusterMember `yaml:"members"`
	CandidateL2s []mq2Candidate     `yaml:"candidate_l2s"`
}

// candidatePaths returns the paths of the cluster's candidates, in order.
func (c mq2Cluster) candidatePaths() []string {
	paths := make([]string, 0, len(c.CandidateL2s))

	for _, candidate := range c.CandidateL2s {
		paths = append(paths, candidate.Path)
	}

	return paths
}

// memberPaths returns the paths of the cluster's members.
func (c mq2Cluster) memberPaths() []string {
	paths := make([]string, 0, len(c.Members))

	for _, member := range c.Members {
		paths = append(paths, member.Path)
	}

	return paths
}

// mq2ClusterMember is one member of a cluster.
type mq2ClusterMember struct {
	Path string `yaml:"path"`
}

// mq2Item is one item of the query payload.
type mq2Item struct {
	Path        string   `yaml:"path"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	SourceTerm  string   `yaml:"source_term"`
	Content     string   `yaml:"content"`
}

// mq2Payload is the part of the query payload that the scenarios of part 2 read.
type mq2Payload struct {
	Items    []mq2Item    `yaml:"items"`
	Clusters []mq2Cluster `yaml:"clusters"`
	Budget   struct {
		ClustersFound    int            `yaml:"clusters_found"`
		ExploreAllocated map[string]int `yaml:"explore_allocated"`
	} `yaml:"budget"`
}

// clusterOf returns the cluster that has the path among its members.
func (p mq2Payload) clusterOf(path string) (mq2Cluster, bool) {
	for _, group := range p.Clusters {
		if slices.Contains(group.memberPaths(), path) {
			return group, true
		}
	}

	return mq2Cluster{}, false
}

// directPaths returns the paths of the items carrying the provenance direct, in item order.
func (p mq2Payload) directPaths() []string {
	return p.pathsWith("direct")
}

// exploreItems returns the items carrying the provenance explore, in item order.
func (p mq2Payload) exploreItems() []mq2Item {
	items := make([]mq2Item, 0, len(p.Items))

	for _, item := range p.Items {
		if slices.Contains(item.Provenances, "explore") {
			items = append(items, item)
		}
	}

	return items
}

// explorePaths returns the paths of the items carrying the provenance explore, in item order.
func (p mq2Payload) explorePaths() []string {
	return p.pathsWith("explore")
}

// find returns the item with the given path.
func (p mq2Payload) find(path string) (mq2Item, bool) {
	for _, item := range p.Items {
		if item.Path == path {
			return item, true
		}
	}

	return mq2Item{}, false
}

// paths returns the paths of every item, in order.
func (p mq2Payload) paths() []string {
	paths := make([]string, 0, len(p.Items))

	for _, item := range p.Items {
		paths = append(paths, item.Path)
	}

	return paths
}

// pathsWith returns the paths of the items carrying the provenance, in item order.
func (p mq2Payload) pathsWith(provenance string) []string {
	paths := make([]string, 0, len(p.Items))

	for _, item := range p.Items {
		if slices.Contains(item.Provenances, provenance) {
			paths = append(paths, item.Path)
		}
	}

	return paths
}

// mq2Result is what one run of the query produced.
type mq2Result struct {
	payload mq2Payload
	stdout  string
}

// mq2ByCosineTo returns the paths ordered by the cosine of their vector to the centroid, nearest first.
func mq2ByCosineTo(centroid []float32, paths []string, vectors [][]float32) []string {
	order := slices.Clone(paths)

	slices.SortStableFunc(order, func(left, right string) int {
		leftCosine := mq2Cosine(centroid, vectors[slices.Index(paths, left)])
		rightCosine := mq2Cosine(centroid, vectors[slices.Index(paths, right)])

		switch {
		case leftCosine > rightCosine:
			return -1
		case leftCosine < rightCosine:
			return 1
		default:
			return 0
		}
	})

	return order
}

// mq2Cosine is the cosine of two vectors, computed here and not by the code under test.
func mq2Cosine(left, right []float32) float64 {
	var dot, leftNorm, rightNorm float64

	for index := range left {
		dot += float64(left[index]) * float64(right[index])
		leftNorm += float64(left[index]) * float64(left[index])
		rightNorm += float64(right[index]) * float64(right[index])
	}

	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

// mq2Mean is the unweighted mean of the vectors.
func mq2Mean(vectors [][]float32) []float32 {
	mean := make([]float32, len(vectors[0]))

	for _, vec := range vectors {
		for dim := range mean {
			mean[dim] += vec[dim] / float32(len(vectors))
		}
	}

	return mean
}

// mq2MustRun queries the world's own phrases and returns the decoded payload and the raw output, failing the test
// on any error.
func mq2MustRun(t *testing.T, world *mq1World) mq2Result {
	t.Helper()

	result := world.query(t, cli.QueryArgs{Phrases: world.phrases})
	if result.err != nil {
		t.Fatalf("query failed: %v", result.err)
	}

	var payload mq2Payload

	err := yaml.Unmarshal([]byte(result.stdout), &payload)
	if err != nil {
		t.Fatalf("decoding the payload: %v", err)
	}

	return mq2Result{payload: payload, stdout: result.stdout}
}

// mq2NoteContent returns the file text of a fact note with the extra frontmatter lines and the body.
func mq2NoteContent(frontmatter, body string) string {
	return "---\ntype: fact\nsituation: a situation\n" + frontmatter + "---\n\n" + body + "\n"
}

// mq2OtherTopicVec returns a vector of the second topic: it matches the first phrase at about 0.5, far from the
// first topic, which sits near axis 0 with the deviation on axis 2.
func mq2OtherTopicVec(deviation float32) []float32 {
	return mq2Vec(0.5, 0, deviation, 0.85)
}

// mq2PlantNote plants a fact note of the given name with the vector as both its situation and body vector. The
// frontmatter lines are added to its frontmatter, and its body is "the <name after its ID> note".
func mq2PlantNote(world *mq1World, name string, vec []float32, frontmatter string) {
	content := mq2NoteContent(frontmatter, "the "+name[2:]+" note")
	world.plant(mq1Note{name: name, content: content, situation: vec, body: vec})
}

// mq2PlantOtherTopic plants count notes of the second topic.
func mq2PlantOtherTopic(world *mq1World, count int) {
	for index := range count {
		mq2PlantNote(world, fmt.Sprintf("%d.other-%d", index+20, index),
			mq2OtherTopicVec(float32(index)*mq2OtherTopicStep), "")
	}
}

// mq2RideAlongWorld returns a world whose phrase matches the notes 1.top (cosine 0.95), 2.older (0.9) and 3.low
// (0.8), which are in that order of score.
func mq2RideAlongWorld() *mq1World {
	const (
		topCosine   = 0.95
		olderCosine = 0.9
		lowCosine   = 0.8
	)

	world := newMq1World(mq2Dims, mq2GoPhrase)
	mq2PlantNote(world, "1.top", world.cosines(map[int]float64{0: topCosine}), "")
	mq2PlantNote(world, "2.older", world.cosines(map[int]float64{0: olderCosine}), "")
	mq2PlantNote(world, "3.low", world.cosines(map[int]float64{0: lowCosine}), "")

	return world
}

// mq2SupersedesOlder returns the frontmatter lines of a note that updates the note 2.older.
func mq2SupersedesOlder() string {
	return "supersedes:\n  - note: 2.older\n    type: updates\n    claim: the older note was wrong\n"
}

// mq2Vec returns a four-dimensional vector.
func mq2Vec(axis0, axis1, axis2, axis3 float32) []float32 {
	return []float32{axis0, axis1, axis2, axis3}
}
