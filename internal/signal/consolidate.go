package signal

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"engram/internal/memory"
)

// ConsolidateResult holds the outcome of a consolidation run.
type ConsolidateResult struct {
	ClustersFound  int
	MemoriesMerged int
	Errors         []error
}

// Consolidator merges duplicate memory clusters before classification (UC-34).
type Consolidator struct {
	lister        MemoryLister
	merger        MergeExecutor
	effectiveness EffectivenessReader
	stderr        io.Writer
}

// NewConsolidator creates a Consolidator with the given options.
func NewConsolidator(opts ...ConsolidatorOption) *Consolidator {
	consolidator := &Consolidator{}
	for _, opt := range opts {
		opt(consolidator)
	}

	return consolidator
}

// Consolidate detects duplicate clusters and merges them (UC-34).
func (c *Consolidator) Consolidate(ctx context.Context) (ConsolidateResult, error) {
	if c.lister == nil || c.merger == nil {
		return ConsolidateResult{}, nil
	}

	memories, err := c.lister.ListAll(ctx)
	if err != nil {
		return ConsolidateResult{}, fmt.Errorf("consolidate: listing memories: %w", err)
	}

	clusters := buildClusters(memories)

	var result ConsolidateResult

	for _, cluster := range clusters {
		if len(cluster) < minClusterSize {
			continue
		}

		result.ClustersFound++

		mergeErr := c.mergeCluster(ctx, cluster, &result)
		if mergeErr != nil {
			result.Errors = append(result.Errors, mergeErr)
			c.logStderrf("[engram] Error consolidating cluster: %v\n", mergeErr)
		}
	}

	if result.MemoriesMerged > 0 {
		c.logStderrf(
			"[engram] Consolidated %d duplicate clusters (%d memories merged)\n",
			result.ClustersFound, result.MemoriesMerged,
		)
	}

	return result, nil
}

func (c *Consolidator) logStderrf(format string, args ...any) {
	if c.stderr != nil {
		//nolint:errcheck // fire-and-forget stderr logging (ARCH-6)
		fmt.Fprintf(c.stderr, format, args...)
	}
}

func (c *Consolidator) mergeCluster(
	ctx context.Context,
	cluster []*memory.Stored,
	result *ConsolidateResult,
) error {
	survivor := c.selectSurvivor(cluster)

	for _, mem := range cluster {
		if mem.FilePath == survivor.FilePath {
			continue
		}

		mergeErr := c.merger.Merge(ctx, survivor, mem)
		if mergeErr != nil {
			return fmt.Errorf(
				"merging %q into %q: %w", mem.Title, survivor.Title, mergeErr,
			)
		}

		result.MemoriesMerged++

		keywordsAdded := countNewKeywords(survivor.Keywords, mem.Keywords)
		c.logStderrf(
			"[engram] Merged %q into %q (%d keywords added)\n",
			mem.Title, survivor.Title, keywordsAdded,
		)
	}

	return nil
}

func (c *Consolidator) selectSurvivor(cluster []*memory.Stored) *memory.Stored {
	items := make([]scoredMemory, 0, len(cluster))

	for _, mem := range cluster {
		item := scoredMemory{mem: mem}

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

// MergeExecutor performs a merge of one memory into another.
type MergeExecutor interface {
	Merge(ctx context.Context, survivor, absorbed *memory.Stored) error
}

// WithEffectiveness sets the effectiveness reader.
func WithEffectiveness(e EffectivenessReader) ConsolidatorOption {
	return func(c *Consolidator) {
		c.effectiveness = e
	}
}

// WithLister sets the memory lister.
func WithLister(l MemoryLister) ConsolidatorOption {
	return func(c *Consolidator) {
		c.lister = l
	}
}

// WithMerger sets the merge executor.
func WithMerger(m MergeExecutor) ConsolidatorOption {
	return func(c *Consolidator) {
		c.merger = m
	}
}

// WithStderr sets the stderr writer for consolidation feedback (DES-48).
func WithStderr(w io.Writer) ConsolidatorOption {
	return func(c *Consolidator) {
		c.stderr = w
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

func countNewKeywords(survivorKW, absorbedKW []string) int {
	existing := keywordSet(survivorKW)
	count := 0

	for _, keyword := range absorbedKW {
		if _, ok := existing[strings.ToLower(keyword)]; !ok {
			count++
		}
	}

	return count
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

	for _, keyword := range keywords {
		set[strings.ToLower(keyword)] = struct{}{}
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
