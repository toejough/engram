package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/toejough/projctl/internal/memory"
)

// memoryConsolidateArgs holds the command-line arguments for consolidate.
type memoryConsolidateArgs struct {
	DecayFactor        string `targ:"flag,name=decay-factor,desc=Decay factor (default 0.9)"`
	PruneThreshold     string `targ:"flag,name=prune-threshold,desc=Confidence threshold for pruning (default 0.1)"`
	DuplicateThreshold string `targ:"flag,name=duplicate-threshold,desc=Similarity threshold for duplicates (default 0.95)"`
	MinRetrievals      int    `targ:"flag,name=min-retrievals,desc=Minimum retrieval count for promotion (default 3)"`
	MinProjects        int    `targ:"flag,name=min-projects,desc=Minimum unique projects for promotion (default 2)"`
	MemoryRoot         string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

// memoryConsolidate performs periodic memory maintenance.
func memoryConsolidate(args memoryConsolidateArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	// Parse float parameters
	var decayFactor float64
	if args.DecayFactor != "" {
		var err error
		decayFactor, err = strconv.ParseFloat(args.DecayFactor, 64)
		if err != nil {
			return fmt.Errorf("invalid decay-factor: %w", err)
		}
	}

	var pruneThreshold float64
	if args.PruneThreshold != "" {
		var err error
		pruneThreshold, err = strconv.ParseFloat(args.PruneThreshold, 64)
		if err != nil {
			return fmt.Errorf("invalid prune-threshold: %w", err)
		}
	}

	var duplicateThreshold float64
	if args.DuplicateThreshold != "" {
		var err error
		duplicateThreshold, err = strconv.ParseFloat(args.DuplicateThreshold, 64)
		if err != nil {
			return fmt.Errorf("invalid duplicate-threshold: %w", err)
		}
	}

	opts := memory.ConsolidateOpts{
		MemoryRoot:         memoryRoot,
		DecayFactor:        decayFactor,
		PruneThreshold:     pruneThreshold,
		DuplicateThreshold: duplicateThreshold,
		MinRetrievals:      args.MinRetrievals,
		MinProjects:        args.MinProjects,
	}

	result, err := memory.Consolidate(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Println("Memory consolidation complete")
	fmt.Printf("Entries decayed: %d\n", result.EntriesDecayed)
	fmt.Printf("Entries pruned: %d\n", result.EntriesPruned)
	fmt.Printf("Duplicates merged: %d\n", result.DuplicatesMerged)
	fmt.Printf("Promotion candidates: %d\n", result.PromotionCandidates)

	return nil
}
