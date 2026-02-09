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
	ClaudeMD           bool   `targ:"flag,name=claude-md,desc=Analyze CLAUDE.md for redundancy and propose maintenance"`
	Synthesize         bool   `targ:"flag,name=synthesize,desc=Identify patterns from repeated similar memories"`
	SynthesisThreshold string `targ:"flag,name=synthesis-threshold,desc=Similarity threshold for pattern clustering (default 0.8)"`
	MinClusterSize     int    `targ:"flag,name=min-cluster-size,desc=Minimum cluster size for patterns (default 3)"`
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

	// Handle --claude-md flag: analyze CLAUDE.md for redundancy
	if args.ClaudeMD {
		claudeMDResult, err := memory.ConsolidateClaudeMD(memory.ConsolidateClaudeMDOpts{
			MemoryRoot: memoryRoot,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("CLAUDE.md analysis complete")
		fmt.Printf("Redundant entries: %d\n", claudeMDResult.RedundantCount)
		fmt.Printf("Promotion candidates: %d\n", claudeMDResult.PromoteCount)

		for _, p := range claudeMDResult.Proposals {
			fmt.Printf("\n[%s] %s\n", p.Type, p.Content)
			fmt.Printf("  Reason: %s\n", p.Reason)
			fmt.Printf("  Action: %s\n", p.Action)
		}

		return nil
	}

	// Parse synthesis threshold
	var synthesisThreshold float64
	if args.SynthesisThreshold != "" {
		var err error
		synthesisThreshold, err = strconv.ParseFloat(args.SynthesisThreshold, 64)
		if err != nil {
			return fmt.Errorf("invalid synthesis-threshold: %w", err)
		}
	}

	opts := memory.ConsolidateOpts{
		MemoryRoot:         memoryRoot,
		DecayFactor:        decayFactor,
		PruneThreshold:     pruneThreshold,
		DuplicateThreshold: duplicateThreshold,
		MinRetrievals:      args.MinRetrievals,
		MinProjects:        args.MinProjects,
		EnableSynthesis:    args.Synthesize,
		SynthesisThreshold: synthesisThreshold,
		MinClusterSize:     args.MinClusterSize,
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

	if args.Synthesize {
		fmt.Printf("Patterns identified: %d\n", result.PatternsIdentified)

		if result.PatternsIdentified > 0 {
			// Run synthesis separately to get full details
			synthResult, err := memory.SynthesizePatterns(memoryRoot, synthesisThreshold, args.MinClusterSize)
			if err == nil {
				for _, p := range synthResult.Patterns {
					fmt.Printf("\n[pattern] %s\n", p.Theme)
					fmt.Printf("  %s\n", p.Synthesis)
				}
			}
		}
	}

	return nil
}
