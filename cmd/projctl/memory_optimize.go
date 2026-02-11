package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/toejough/projctl/internal/memory"
)

type memoryOptimizeArgs struct {
	Review                   bool    `targ:"flag,desc=Use interactive proposal review mode (new ISSUE-212 workflow)"`
	Yes                      bool    `targ:"flag,short=y,desc=Auto-approve all interactive prompts"`
	ClaudeMD                 string  `targ:"flag,name=claude-md,desc=Path to CLAUDE.md (defaults to ~/.claude/CLAUDE.md)"`
	MemoryRoot               string  `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
	SkillPromotionThreshold  float64 `targ:"flag,desc=Minimum utility threshold for skill promotion to CLAUDE.md (default 0.8)"`
	SkillDemotionThreshold   float64 `targ:"flag,desc=Utility threshold for skill demotion/pruning (default 0.4)"`
	MinSkillProjects         int     `targ:"flag,desc=Minimum number of projects for skill promotion (default 3)"`
	MinSkillConfidenceThresh float64 `targ:"flag,name=min-skill-confidence,desc=Minimum confidence for skill promotion (default 0.8)"`
	ForceReorg               bool    `targ:"flag,desc=Force full skill reorganization regardless of last run time (normally runs every 30 days)"`
	NoLLM                    bool    `targ:"flag,desc=Disable all LLM-based features (extractor, specificity detector, skill compiler)"`
}

func memoryOptimize(args memoryOptimizeArgs) error {
	// Set up context with signal cancellation (ctrl-c / SIGINT)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	claudeMDPath := args.ClaudeMD
	if claudeMDPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		claudeMDPath = filepath.Join(home, ".claude", "CLAUDE.md")
	}

	// Set up skills directory
	skillsDir := ""
	{
		home, err := os.UserHomeDir()
		if err == nil {
			skillsDir = filepath.Join(home, ".claude", "skills")
		}
	}

	// Use new interactive review workflow if --review flag is set
	if args.Review {
		return runInteractiveOptimize(ctx, memoryRoot, claudeMDPath, skillsDir, args)
	}

	// Otherwise use legacy optimize workflow

	opts := memory.OptimizeOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
		AutoApprove:  args.Yes,
		SkillsDir:    skillsDir,
		ForceReorg:   args.ForceReorg,
		Context:      ctx,
	}

	// Wire LLM interfaces via shared instance (unless --no-llm is set)
	if !args.NoLLM {
		extractor := memory.NewClaudeCLIExtractor()
		opts.SkillCompiler = extractor
		opts.SpecificDetector = extractor
		opts.Extractor = extractor
	}

	// Apply threshold overrides from CLI flags
	if args.SkillPromotionThreshold > 0 {
		opts.MinSkillUtility = args.SkillPromotionThreshold
	}
	if args.SkillDemotionThreshold > 0 {
		opts.AutoDemoteUtility = args.SkillDemotionThreshold
	}
	if args.MinSkillProjects > 0 {
		opts.MinSkillProjects = args.MinSkillProjects
	}
	if args.MinSkillConfidenceThresh > 0 {
		opts.MinSkillConfidence = args.MinSkillConfidenceThresh
	}

	// If not auto-approving, set up interactive review
	if !args.Yes {
		opts.ReviewFunc = func(action, description string) (bool, error) {
			fmt.Printf("\n[%s]\n%s\n", action, description)
			fmt.Print("Approve? (y/n): ")

			var response string
			_, err := fmt.Scanln(&response)
			if err != nil {
				return false, fmt.Errorf("failed to read input: %w", err)
			}

			return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes", nil
		}
	}

	result, err := memory.Optimize(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Println("Memory optimization complete")

	if result.DecayApplied {
		fmt.Printf("Decay applied: %d entries (factor: %.4f non-promoted, %.4f promoted, %.1f days elapsed)\n",
			result.EntriesDecayed, result.DecayFactor, result.PromotedDecayFactor, result.DaysSinceLastOptimize)
	} else {
		fmt.Printf("Decay skipped (last optimized <1h ago)\n")
	}

	if result.ContradictionsFound > 0 {
		fmt.Printf("Contradictions found: %d\n", result.ContradictionsFound)
	}
	if result.AutoDemoted > 0 {
		fmt.Printf("Auto-demoted from CLAUDE.md: %d\n", result.AutoDemoted)
	}
	if result.EntriesPruned > 0 {
		fmt.Printf("Entries pruned: %d\n", result.EntriesPruned)
	}
	if result.BoilerplatePurged > 0 {
		fmt.Printf("Boilerplate purged: %d\n", result.BoilerplatePurged)
	}
	if result.LegacySessionPurged > 0 {
		fmt.Printf("Legacy session embeddings purged: %d\n", result.LegacySessionPurged)
	}
	if result.DuplicatesMerged > 0 {
		fmt.Printf("Duplicates merged: %d\n", result.DuplicatesMerged)
	}
	if result.PatternsFound > 0 {
		fmt.Printf("Patterns found: %d (approved: %d)\n", result.PatternsFound, result.PatternsApproved)
	}
	if result.PromotionCandidates > 0 {
		fmt.Printf("Promotion candidates: %d (approved: %d)\n", result.PromotionCandidates, result.PromotionsApproved)
	}
	if result.SkillsCompiled > 0 {
		fmt.Printf("Skills compiled: %d\n", result.SkillsCompiled)
	}
	if result.SkillsMerged > 0 {
		fmt.Printf("Skills merged: %d\n", result.SkillsMerged)
	}
	if result.SkillsPruned > 0 {
		fmt.Printf("Skills pruned: %d\n", result.SkillsPruned)
	}
	if result.SkillsReorganized > 0 {
		fmt.Printf("Skills reorganized: %d\n", result.SkillsReorganized)
	}
	if result.ClaudeMDDeduped > 0 {
		fmt.Printf("CLAUDE.md entries deduped: %d\n", result.ClaudeMDDeduped)
	}
	if result.ClaudeMDDemoted > 0 {
		fmt.Printf("ClaudeMDDemoted: %d\n", result.ClaudeMDDemoted)
	}
	if result.SkillsPromoted > 0 {
		fmt.Printf("SkillsPromoted: %d\n", result.SkillsPromoted)
	}

	return nil
}

// runInteractiveOptimize executes the new ISSUE-212 interactive review workflow.
func runInteractiveOptimize(ctx context.Context, memoryRoot, claudeMDPath, skillsDir string, args memoryOptimizeArgs) error {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")

	// Set up review function
	var reviewFunc memory.MaintenanceReviewFunc
	if args.Yes {
		// Auto-approve all
		reviewFunc = func(p memory.MaintenanceProposal) bool {
			return true
		}
	}
	// If no --yes flag, reviewFunc is nil and OptimizeInteractive will use interactive reviewProposal

	// Run interactive optimization
	opts := memory.OptimizeInteractiveOpts{
		DBPath:       dbPath,
		ClaudeMDPath: claudeMDPath,
		SkillsDir:    skillsDir,
		ReviewFunc:   reviewFunc,
		Input:        os.Stdin,
		Output:       os.Stdout,
		Context:      ctx,
	}

	result, err := memory.OptimizeInteractive(opts)
	if err != nil {
		return fmt.Errorf("interactive optimization failed: %w", err)
	}

	// Print summary
	fmt.Println("\n=== Memory Optimization Summary ===")
	fmt.Printf("Total proposals: %d\n", result.Total)
	fmt.Printf("  Approved: %d\n", result.Approved)
	fmt.Printf("  Rejected: %d\n", result.Rejected)
	fmt.Printf("  Applied: %d\n", result.Applied)
	if result.Failed > 0 {
		fmt.Printf("  Failed: %d\n", result.Failed)
	}

	// Print tier-by-tier summary
	if len(result.TierSummary) > 0 {
		fmt.Println("\nBy tier:")
		for tier, stats := range result.TierSummary {
			if stats.Total > 0 {
				fmt.Printf("  %s: %d proposals", tier, stats.Total)
				if stats.Applied > 0 {
					fmt.Printf(" (%d applied", stats.Applied)
					if len(stats.Actions) > 0 {
						var actionSummary []string
						for action, count := range stats.Actions {
							if count > 0 {
								actionSummary = append(actionSummary, fmt.Sprintf("%s=%d", action, count))
							}
						}
						if len(actionSummary) > 0 {
							fmt.Printf(": %s", strings.Join(actionSummary, ", "))
						}
					}
					fmt.Print(")")
				}
				fmt.Println()
			}
		}
	}

	return nil
}
