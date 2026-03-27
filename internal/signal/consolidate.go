package signal

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"engram/internal/keyword"
	"engram/internal/memory"
)

// Consolidator detects duplicate memory clusters and plans merges (UC-34).
type Consolidator struct {
	lister        MemoryLister
	effectiveness EffectivenessReader
	similarity    TextSimilarityScorer
	stderr        io.Writer
	scorer        Scorer
	confirmer     Confirmer
	extractor     Extractor
	archiver      Archiver
	minConfidence float64
}

// NewConsolidator creates a Consolidator with the given options.
func NewConsolidator(opts ...ConsolidatorOption) *Consolidator {
	consolidator := &Consolidator{}
	for _, opt := range opts {
		opt(consolidator)
	}

	return consolidator
}

// Plan detects duplicate clusters and returns the merge plan without executing it.
// It is the dry-run equivalent of the former Consolidate (#335).
func (c *Consolidator) Plan(ctx context.Context) ([]MergePlan, error) {
	if c.lister == nil {
		return nil, nil
	}

	memories, err := c.lister.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("plan: listing memories: %w", err)
	}

	clusters := buildClusters(memories)
	plans := make([]MergePlan, 0, len(clusters))

	for _, cluster := range clusters {
		if len(cluster) < minClusterSize {
			continue
		}

		survivor := c.selectSurvivor(cluster)
		absorbed := make([]string, 0, len(cluster)-1)

		for _, mem := range cluster {
			if mem.FilePath != survivor.FilePath {
				absorbed = append(absorbed, mem.FilePath)
			}
		}

		confidence := c.clusterConfidence(cluster)

		if confidence >= 0 && confidence < c.minConfidence {
			continue
		}

		plans = append(plans, MergePlan{
			Survivor:   survivor.FilePath,
			Absorbed:   absorbed,
			Confidence: confidence,
		})
	}

	return plans, nil
}

// clusterConfidence returns the TF-IDF similarity score for a cluster, or -1 if no scorer.
func (c *Consolidator) clusterConfidence(cluster []*memory.Stored) float64 {
	if c.similarity == nil {
		return -1
	}

	texts := make([]string, 0, len(cluster))

	for _, mem := range cluster {
		text := strings.Join(mem.Keywords, " ") + " " + mem.Principle
		texts = append(texts, text)
	}

	return c.similarity.ClusterConfidence(texts)
}

func (c *Consolidator) logStderrf(format string, args ...any) {
	if c.stderr != nil {
		//nolint:errcheck // fire-and-forget stderr logging (ARCH-6)
		fmt.Fprintf(c.stderr, format, args...)
	}
}

func (c *Consolidator) selectSurvivor(cluster []*memory.Stored) *memory.Stored {
	items := make([]scoredMemory, 0, len(cluster))

	for _, mem := range cluster {
		item := scoredMemory{mem: mem, surfacedCount: mem.SurfacedCount}

		if c.effectiveness != nil {
			score, hasData, effErr := c.effectiveness.EffectivenessScore(mem.FilePath)
			if effErr == nil {
				item.effectiveness = score
				item.hasData = hasData
			}
		}

		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		// Scored beats unscored
		if items[i].hasData != items[j].hasData {
			return items[i].hasData
		}
		// Higher effectiveness wins
		if items[i].effectiveness != items[j].effectiveness {
			return items[i].effectiveness > items[j].effectiveness
		}
		// Higher surfaced count wins
		if items[i].surfacedCount != items[j].surfacedCount {
			return items[i].surfacedCount > items[j].surfacedCount
		}
		// Alphabetical tiebreak
		return items[i].mem.FilePath < items[j].mem.FilePath
	})

	return items[0].mem
}

// ConsolidatorOption configures a Consolidator.
type ConsolidatorOption func(*Consolidator)

// EffectivenessReader reads effectiveness score for a memory.
type EffectivenessReader interface {
	EffectivenessScore(path string) (score float64, hasData bool, err error)
}

// MemoryLister loads all existing memories from the data directory.
type MemoryLister interface {
	ListAll(ctx context.Context) ([]*memory.Stored, error)
}

// MergePlan describes what would happen for one cluster in a dry run.
type MergePlan struct {
	Survivor   string   `json:"survivor"`
	Absorbed   []string `json:"absorbed"`
	Confidence float64  `json:"confidence"`
}

// TextSimilarityScorer computes pairwise text similarity within a cluster (ARCH-82).
// Returns a confidence score in [0,1] where 1 = identical content.
type TextSimilarityScorer interface {
	ClusterConfidence(texts []string) float64
}

// WithArchiver sets the archiver for consolidated memory originals.
func WithArchiver(a Archiver) ConsolidatorOption {
	return func(c *Consolidator) { c.archiver = a }
}

// WithConfirmer sets the LLM cluster confirmer for semantic clustering.
func WithConfirmer(cf Confirmer) ConsolidatorOption {
	return func(c *Consolidator) { c.confirmer = cf }
}

// WithEffectiveness sets the effectiveness reader.
func WithEffectiveness(e EffectivenessReader) ConsolidatorOption {
	return func(c *Consolidator) {
		c.effectiveness = e
	}
}

// WithExtractor sets the LLM principle extractor for semantic clustering.
func WithExtractor(e Extractor) ConsolidatorOption {
	return func(c *Consolidator) { c.extractor = e }
}

// WithLister sets the memory lister.
func WithLister(l MemoryLister) ConsolidatorOption {
	return func(c *Consolidator) {
		c.lister = l
	}
}

// WithMinConfidence sets the minimum TF-IDF confidence for cluster inclusion.
func WithMinConfidence(minConfidence float64) ConsolidatorOption {
	return func(c *Consolidator) {
		c.minConfidence = minConfidence
	}
}

// WithScorer sets the BM25 candidate scorer for semantic clustering.
func WithScorer(s Scorer) ConsolidatorOption {
	return func(c *Consolidator) { c.scorer = s }
}

// WithStderr sets the stderr writer for consolidation feedback (DES-48).
func WithStderr(w io.Writer) ConsolidatorOption {
	return func(c *Consolidator) {
		c.stderr = w
	}
}

// WithTextSimilarityScorer sets the TF-IDF similarity scorer (ARCH-82, REQ-140).
func WithTextSimilarityScorer(s TextSimilarityScorer) ConsolidatorOption {
	return func(c *Consolidator) {
		c.similarity = s
	}
}

// unexported constants.
const (
	minClusterSize   = 2
	overlapThreshold = 0.5
)

type scoredMemory struct {
	mem           *memory.Stored
	effectiveness float64
	hasData       bool
	surfacedCount int
}

// buildClusters groups memories by >50% keyword overlap with transitive closure.
func buildClusters(memories []*memory.Stored) [][]*memory.Stored {
	memoryCount := len(memories)
	if memoryCount == 0 {
		return nil
	}

	// Union-Find for transitive closure
	parent := make([]int, memoryCount)
	for i := range parent {
		parent[i] = i
	}

	for i := range memoryCount {
		for j := i + 1; j < memoryCount; j++ {
			if overlaps(memories[i], memories[j]) {
				union(parent, i, j)
			}
		}
	}

	// Group by root
	groups := make(map[int][]*memory.Stored)

	for i, mem := range memories {
		root := find(parent, i)
		groups[root] = append(groups[root], mem)
	}

	clusters := make([][]*memory.Stored, 0, len(groups))

	for _, group := range groups {
		clusters = append(clusters, group)
	}

	return clusters
}

// Union-Find helpers.
func find(parent []int, idx int) int {
	for parent[idx] != idx {
		parent[idx] = parent[parent[idx]]
		idx = parent[idx]
	}

	return idx
}

func intersectionSize(first, second map[string]struct{}) int {
	count := 0

	for key := range first {
		if _, ok := second[key]; ok {
			count++
		}
	}

	return count
}

func keywordSet(keywords []string) map[string]struct{} {
	set := make(map[string]struct{}, len(keywords))

	for _, kw := range keywords {
		set[keyword.Normalize(kw)] = struct{}{}
	}

	return set
}

// overlaps returns true if two memories have >50% keyword overlap in either direction.
func overlaps(first, second *memory.Stored) bool {
	if first == nil || second == nil {
		return false
	}

	firstKeys := keywordSet(first.Keywords)
	secondKeys := keywordSet(second.Keywords)

	if len(firstKeys) == 0 || len(secondKeys) == 0 {
		return false
	}

	count := intersectionSize(firstKeys, secondKeys)

	firstOverlap := float64(count) / float64(len(firstKeys))
	secondOverlap := float64(count) / float64(len(secondKeys))

	return firstOverlap > overlapThreshold || secondOverlap > overlapThreshold
}

func union(parent []int, first, second int) {
	rootFirst, rootSecond := find(parent, first), find(parent, second)
	if rootFirst != rootSecond {
		parent[rootFirst] = rootSecond
	}
}
