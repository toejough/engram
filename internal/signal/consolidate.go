package signal

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"engram/internal/memory"
)

// BackupWriter writes a backup copy of an absorbed file before deletion (REQ-135).
type BackupWriter interface {
	Backup(absorbedPath, backupDir string) error
}

// ConsolidateResult holds the outcome of a consolidation run.
type ConsolidateResult struct {
	ClustersFound  int
	MemoriesMerged int
	Errors         []error
}

// Consolidator merges duplicate memory clusters before classification (UC-34).
type Consolidator struct {
	lister         MemoryLister
	merger         MergeExecutor
	synthesizer    PrincipleSynthesizer
	fileWriter     MemoryWriter
	backupWriter   BackupWriter
	backupDir      string
	fileDeleter    FileDeleter
	entryRemover   func(path string) error
	linkRecomputer LinkRecomputer
	effectiveness  EffectivenessReader
	similarity     TextSimilarityScorer
	stderr         io.Writer
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

		confidence := c.clusterConfidence(cluster)

		mergeErr := c.mergeCluster(ctx, cluster, &result)
		if mergeErr != nil {
			result.Errors = append(result.Errors, mergeErr)
			c.logStderrf("[engram] Error consolidating cluster: %v\n", mergeErr)
		} else if confidence >= 0 {
			c.logStderrf("[engram] Cluster confidence: %.2f\n", confidence)
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

// Plan detects duplicate clusters and returns the merge plan without executing it.
// It is the dry-run equivalent of Consolidate (#335).
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

		plans = append(plans, MergePlan{
			Survivor:   survivor.FilePath,
			Absorbed:   absorbed,
			Confidence: confidence,
		})
	}

	return plans, nil
}

// applySynthesizedPrinciple calls the synthesizer and updates survivor.Principle (REQ-139, Phase 1b).
// Falls back silently to the Phase 1 result on error (fire-and-forget per ARCH-6).
func (c *Consolidator) applySynthesizedPrinciple(
	ctx context.Context,
	survivor *memory.Stored,
	allPrinciples []string,
) {
	if c.synthesizer == nil {
		return
	}

	synthesized, synthErr := c.synthesizer.SynthesizePrinciples(ctx, allPrinciples)
	if synthErr != nil {
		c.logStderrf("[engram] principle synthesis failed: %v; using fallback\n", synthErr)

		return
	}

	survivor.Principle = synthesized
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

func (c *Consolidator) mergeCluster(
	ctx context.Context,
	cluster []*memory.Stored,
	result *ConsolidateResult,
) error {
	survivor := c.selectSurvivor(cluster)

	absorbed := make([]*memory.Stored, 0, len(cluster)-1)

	for _, mem := range cluster {
		if mem.FilePath != survivor.FilePath {
			absorbed = append(absorbed, mem)
		}
	}

	// Collect all principles before Phase 1 for LLM synthesis (REQ-139).
	allPrinciples := collectPrinciples(cluster)

	// Phase 1: compute all merges in memory (no I/O).
	// Capture keyword counts before each merge for accurate logging.
	newKWCounts := make([]int, len(absorbed))

	for idx, mem := range absorbed {
		newKWCounts[idx] = countNewKeywords(survivor.Keywords, mem.Keywords)

		mergeErr := c.merger.Merge(ctx, survivor, mem)
		if mergeErr != nil {
			return fmt.Errorf("merging %q into %q: %w", mem.Title, survivor.Title, mergeErr)
		}
	}

	// Phase 1b: LLM principle synthesis (REQ-139). Falls back to Phase 1 result on failure.
	c.applySynthesizedPrinciple(ctx, survivor, allPrinciples)

	// Phase 2: write merged survivor once (atomic — no deletes if write fails, REQ-137 AC3).
	if c.fileWriter != nil {
		writeErr := c.fileWriter.Write(survivor.FilePath, survivor)
		if writeErr != nil {
			return fmt.Errorf("writing merged survivor %q: %w", survivor.Title, writeErr)
		}
	}

	// Phase 3: backup, delete, and remove registry entry for each absorbed memory.
	for idx, mem := range absorbed {
		absorbErr := c.processAbsorbed(mem, survivor, newKWCounts[idx], result)
		if absorbErr != nil {
			return absorbErr
		}
	}

	// Phase 4: recompute links after absorbed files deleted (REQ-138, fire-and-forget).
	c.recomputeLinks(survivor.FilePath, absorbed)

	return nil
}

// processAbsorbed handles backup, deletion, and registry removal for one absorbed memory (Phase 3).
func (c *Consolidator) processAbsorbed(
	mem *memory.Stored,
	survivor *memory.Stored,
	newKWCount int,
	result *ConsolidateResult,
) error {
	// Backup is fire-and-forget (REQ-135 AC4).
	if c.backupWriter != nil {
		backupErr := c.backupWriter.Backup(mem.FilePath, c.backupDir)
		if backupErr != nil {
			c.logStderrf("[engram] backup failed for %q: %v\n", mem.FilePath, backupErr)
		}
	}

	// Delete absorbed file after survivor written (REQ-136 AC1).
	if c.fileDeleter != nil {
		deleteErr := c.fileDeleter.Delete(mem.FilePath)
		if deleteErr != nil {
			return fmt.Errorf("deleting absorbed %q: %w", mem.Title, deleteErr)
		}
	}

	// Remove registry entry after file deletion (REQ-136 AC3).
	if c.entryRemover != nil {
		regErr := c.entryRemover(mem.FilePath)
		if regErr != nil {
			c.logStderrf("[engram] registry remove failed for %q: %v\n", mem.FilePath, regErr)
		}
	}

	result.MemoriesMerged++

	c.logStderrf(
		"[engram] Merged %q into %q (%d keywords added)\n",
		mem.Title, survivor.Title, newKWCount,
	)

	return nil
}

// recomputeLinks fires link recomputation for each absorbed file (REQ-138, fire-and-forget).
func (c *Consolidator) recomputeLinks(survivorPath string, absorbed []*memory.Stored) {
	if c.linkRecomputer == nil {
		return
	}

	for _, mem := range absorbed {
		linkErr := c.linkRecomputer.RecomputeAfterMerge(survivorPath, mem.FilePath)
		if linkErr != nil {
			c.logStderrf("[engram] link recompute failed for %q: %v\n", mem.FilePath, linkErr)
		}
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

// FileDeleter deletes a file at the given path (REQ-136).
type FileDeleter interface {
	Delete(path string) error
}

// LinkRecomputer recomputes graph links after a cluster merge (REQ-138).
type LinkRecomputer interface {
	RecomputeAfterMerge(survivorPath, absorbedPath string) error
}

// MemoryLister loads all existing memories from the data directory.
type MemoryLister interface {
	ListAll(ctx context.Context) ([]*memory.Stored, error)
}

// MergeExecutor performs a merge of one memory into another.
type MergeExecutor interface {
	Merge(ctx context.Context, survivor, absorbed *memory.Stored) error
}

// MergePlan describes what would happen for one cluster in a dry run.
type MergePlan struct {
	Survivor   string   `json:"survivor"`
	Absorbed   []string `json:"absorbed"`
	Confidence float64  `json:"confidence"`
}

// PrincipleSynthesizer synthesizes a merged principle from all cluster members' principles (REQ-139).
type PrincipleSynthesizer interface {
	SynthesizePrinciples(ctx context.Context, principles []string) (string, error)
}

// TextSimilarityScorer computes pairwise text similarity within a cluster (ARCH-82).
// Returns a confidence score in [0,1] where 1 = identical content.
type TextSimilarityScorer interface {
	ClusterConfidence(texts []string) float64
}

// WithBackupWriter sets the backup writer and backup directory (REQ-135).
func WithBackupWriter(w BackupWriter, backupDir string) ConsolidatorOption {
	return func(c *Consolidator) {
		c.backupWriter = w
		c.backupDir = backupDir
	}
}

// WithEffectiveness sets the effectiveness reader.
func WithEffectiveness(e EffectivenessReader) ConsolidatorOption {
	return func(c *Consolidator) {
		c.effectiveness = e
	}
}

// WithEntryRemover sets the entry removal function for absorbed memories (REQ-136).
func WithEntryRemover(fn func(path string) error) ConsolidatorOption {
	return func(c *Consolidator) {
		c.entryRemover = fn
	}
}

// WithFileDeleter sets the file deleter for absorbed memories (REQ-136).
func WithFileDeleter(d FileDeleter) ConsolidatorOption {
	return func(c *Consolidator) {
		c.fileDeleter = d
	}
}

// WithFileWriter sets the file writer for the merged survivor memory (REQ-136).
func WithFileWriter(w MemoryWriter) ConsolidatorOption {
	return func(c *Consolidator) {
		c.fileWriter = w
	}
}

// WithLinkRecomputer sets the link recomputer for post-merge link updates (REQ-138).
func WithLinkRecomputer(r LinkRecomputer) ConsolidatorOption {
	return func(c *Consolidator) {
		c.linkRecomputer = r
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

// WithPrincipleSynthesizer sets the LLM principle synthesizer (REQ-139).
func WithPrincipleSynthesizer(s PrincipleSynthesizer) ConsolidatorOption {
	return func(c *Consolidator) {
		c.synthesizer = s
	}
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

func collectPrinciples(cluster []*memory.Stored) []string {
	principles := make([]string, 0, len(cluster))

	for _, mem := range cluster {
		if mem.Principle != "" {
			principles = append(principles, mem.Principle)
		}
	}

	return principles
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
