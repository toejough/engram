package signal

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"engram/internal/memory"
)

// Exported variables.
var (
	// ErrNilExtractor is returned when consolidateCluster is called without an extractor.
	ErrNilExtractor = errors.New("consolidating cluster: extractor is nil")
)

// BeforeRemove checks if a memory slated for removal belongs to a cluster.
// Called by maintain pipeline before generating a removal proposal.
func (c *Consolidator) BeforeRemove(
	ctx context.Context,
	mem *memory.MemoryRecord,
) (Action, error) {
	cluster := c.findCluster(ctx, mem, nil)
	if cluster == nil {
		return Action{Type: ProceedWithRemoval}, nil
	}

	action, err := c.consolidateCluster(ctx, cluster)
	if err != nil {
		c.logStderrf("[engram] consolidation failed before remove: %v\n", err)

		return Action{Type: ProceedWithRemoval}, nil
	}

	return action, nil
}

// BeforeStore checks if a candidate memory belongs to an existing cluster.
// Called by learn/correct pipeline before writing a new memory to disk.
func (c *Consolidator) BeforeStore(
	ctx context.Context,
	candidate *memory.MemoryRecord,
) (Action, error) {
	cluster := c.findCluster(ctx, candidate, nil)
	if cluster == nil {
		return Action{Type: StoreAsIs}, nil
	}

	action, err := c.consolidateCluster(ctx, cluster)
	if err != nil {
		c.logStderrf("[engram] consolidation failed, storing as-is: %v\n", err)

		return Action{Type: StoreAsIs}, nil
	}

	return action, nil
}

// OnIrrelevant checks if an irrelevantly-surfaced memory belongs to a cluster.
// Called by feedback pipeline after recording irrelevant feedback.
func (c *Consolidator) OnIrrelevant(
	ctx context.Context,
	input OnIrrelevantInput,
) (Action, error) {
	refinement := Action{
		Type: RefineKeywords,
		RefinementContext: &RefinementContext{
			Memory:         input.Memory,
			SurfacingQuery: input.SurfacingQuery,
			ToolName:       input.ToolName,
			ToolInput:      input.ToolInput,
		},
	}

	cluster := c.findCluster(ctx, input.Memory, nil)
	if cluster == nil {
		return refinement, nil
	}

	action, err := c.consolidateCluster(ctx, cluster)
	if err != nil {
		c.logStderrf("[engram] consolidation failed on irrelevant: %v\n", err)

		return refinement, nil
	}

	return action, nil
}

// consolidateCluster executes consolidation for a confirmed cluster.
// Shared by BeforeStore, OnIrrelevant, and BeforeRemove.
func (c *Consolidator) consolidateCluster(
	ctx context.Context,
	cluster *ConfirmedCluster,
) (Action, error) {
	if c.extractor == nil {
		return Action{}, ErrNilExtractor
	}

	// Check if any member is already consolidated (has Absorbed records).
	existing := findExistingConsolidated(cluster.Members)

	consolidated, err := c.extractor.ExtractPrinciple(ctx, *cluster)
	if err != nil {
		return Action{}, fmt.Errorf("consolidating cluster: %w", err)
	}

	// Determine which members to transfer counters from and archive.
	// If updating an existing consolidated memory, only transfer from non-existing members.
	originals := cluster.Members
	if existing != nil {
		originals = excludeExisting(cluster.Members, existing)
		// Copy existing absorbed records into the new consolidated memory.
		consolidated.Absorbed = append(consolidated.Absorbed, existing.Absorbed...)
	}

	TransferFields(consolidated, originals, time.Now())

	// Archive originals (skip the existing consolidated if updating).
	archived := make([]string, 0, len(originals))

	for _, orig := range originals {
		if c.archiver != nil {
			archErr := c.archiver.Archive(orig.SourcePath)
			if archErr != nil {
				c.logStderrf("[engram] archive failed for %q: %v\n", orig.Title, archErr)
			}
		}

		archived = append(archived, orig.Title)
	}

	return Action{
		Type:            Consolidated,
		ConsolidatedMem: consolidated,
		Archived:        archived,
	}, nil
}

// findCluster attempts to find a semantic cluster for the given memory.
// Returns nil if no cluster is found or if the pipeline is unavailable.
func (c *Consolidator) findCluster(
	ctx context.Context,
	mem *memory.MemoryRecord,
	exclude []string,
) *ConfirmedCluster {
	if c.scorer == nil || c.confirmer == nil {
		return nil
	}

	candidates, err := c.scorer.FindSimilar(ctx, mem, exclude)
	if err != nil || len(candidates) < minSemanticClusterSize-1 {
		return nil
	}

	clusters, err := c.confirmer.ConfirmClusters(ctx, mem, candidates)
	if err != nil || len(clusters) == 0 {
		return nil
	}

	// Sort smallest first — protect fragile clusters (per spec).
	sort.Slice(clusters, func(i, j int) bool {
		return len(clusters[i].Members) < len(clusters[j].Members)
	})

	for idx := range clusters {
		if len(clusters[idx].Members) >= minSemanticClusterSize {
			return &clusters[idx]
		}
	}

	return nil
}

// unexported constants.
const (
	minSemanticClusterSize = 3
)

// excludeExisting returns members that are NOT the existing consolidated memory.
func excludeExisting(
	members []*memory.MemoryRecord,
	existing *memory.MemoryRecord,
) []*memory.MemoryRecord {
	result := make([]*memory.MemoryRecord, 0, len(members)-1)

	for _, mem := range members {
		if mem != existing {
			result = append(result, mem)
		}
	}

	return result
}

// findExistingConsolidated returns the first member that is already
// a consolidated memory (non-empty Absorbed), or nil.
func findExistingConsolidated(members []*memory.MemoryRecord) *memory.MemoryRecord {
	for _, mem := range members {
		if len(mem.Absorbed) > 0 {
			return mem
		}
	}

	return nil
}
