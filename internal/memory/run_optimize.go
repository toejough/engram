package memory

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

// RunOptimize runs memory optimization (non-interactive or interactive).
func RunOptimize(args OptimizeArgs, homeDir string, stdin io.Reader) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	claudeMDPath := args.ClaudeMD
	if claudeMDPath == "" {
		claudeMDPath = filepath.Join(homeDir, ".claude", "CLAUDE.md")
	}

	skillsDir := filepath.Join(homeDir, ".claude", "skills")

	if args.Review {
		return runInteractiveOptimize(ctx, memoryRoot, claudeMDPath, skillsDir, args, stdin)
	}

	opts := OptimizeOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
		AutoApprove:  args.Yes,
		SkillsDir:    skillsDir,
		ForceReorg:   args.ForceReorg,
		Context:      ctx,
		TestSkills:   !args.NoTestSkills,
		TestRuns:     args.TestRuns,
	}

	if !args.NoLLM {
		extractor := NewLLMExtractor()
		if extractor == nil {
			return errors.New("LLM extractor unavailable (keychain auth failed); use --no-llm to run without LLM features")
		}

		opts.SkillCompiler = extractor
		opts.SpecificDetector = extractor
		opts.Extractor = extractor
	}

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

	if !args.Yes {
		opts.ReviewFunc = func(action, description string) (bool, error) {
			fmt.Printf("\n[%s]\n%s\n", action, description)
			fmt.Print("Approve? (y/n): ")

			var response string

			_, err := fmt.Fscanln(stdin, &response)
			if err != nil {
				return false, fmt.Errorf("failed to read input: %w", err)
			}

			return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes", nil
		}
	}

	result, err := Optimize(opts)
	if err != nil {
		return err
	}

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

	if db, dbErr := InitEmbeddingsDB(memoryRoot); dbErr == nil {
		budgetProposals, budgetErr := EnforceClaudeMDBudget(claudeMDPath, db, RealFS{})
		_ = db.Close()

		if budgetErr == nil && len(budgetProposals) > 0 {
			var recs []Recommendation
			for _, p := range budgetProposals {
				recs = append(recs, p.Recommendation)
			}

			printOptimizeRecommendationsSummary(recs)

			if err2 := offerSaveOptimizeRecommendations(recs, memoryRoot, args.Yes); err2 != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save recommendations: %v\n", err2)
			}
		}
	}

	return nil
}

func offerSaveOptimizeRecommendations(recs []Recommendation, memoryRoot string, yes bool) error {
	if len(recs) == 0 {
		return nil
	}

	filename := fmt.Sprintf("memory-recommendations-%s.md", time.Now().Format("2006-01-02"))
	filePath := filepath.Join(memoryRoot, filename)

	ptrs := make([]*Recommendation, len(recs))
	for i := range recs {
		ptrs[i] = &recs[i]
	}

	if yes {
		if err := saveOptimizeRecommendations(filePath, ptrs); err != nil {
			return err
		}

		fmt.Printf("Recommendations saved to: %s\n", filePath)

		return nil
	}

	fmt.Printf("To save recommendations, re-run with --yes or review: %s\n", filePath)

	return nil
}

func printOptimizeRecommendationsSummary(recs []Recommendation) {
	if len(recs) == 0 {
		return
	}

	fmt.Printf("\n%d recommendation(s):\n", len(recs))

	for i, rec := range recs {
		fmt.Printf("  %d. [%s] %s\n", i+1, rec.Category, rec.Text)
	}
}

func runInteractiveOptimize(ctx context.Context, memoryRoot, claudeMDPath, skillsDir string, args OptimizeArgs, stdin io.Reader) error {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")

	var reviewFunc MaintenanceReviewFunc
	if args.Yes {
		reviewFunc = func(_ MaintenanceProposal) bool {
			return true
		}
	}

	var extractor LLMExtractor

	if !args.NoLLM {
		ext := NewLLMExtractor()
		if ext == nil {
			return errors.New("LLM extractor unavailable (keychain auth failed); use --no-llm to run without LLM features")
		}

		extractor = ext
	}

	var contextAssembler ContextAssembler

	if !args.NoLLMEval && !args.NoLLM {
		claudeMDContent, _ := os.ReadFile(claudeMDPath)

		var skillDescs []string

		if skillsDir != "" {
			entries, _ := os.ReadDir(skillsDir)
			for _, e := range entries {
				if e.IsDir() {
					skillPath := filepath.Join(skillsDir, e.Name(), "SKILL.md")
					if data, err := os.ReadFile(skillPath); err == nil {
						lines := strings.SplitN(string(data), "\n", 2)
						if len(lines) > 0 {
							skillDescs = append(skillDescs, e.Name()+": "+lines[0])
						}
					}
				}
			}
		}

		var embeddingTexts []string

		adb, dbErr := InitEmbeddingsDB(memoryRoot)
		if dbErr == nil {
			rows, qErr := adb.Query(`SELECT content FROM embeddings ORDER BY confidence DESC, retrieval_count DESC LIMIT 50`)
			if qErr == nil {
				for rows.Next() {
					var content string
					if rows.Scan(&content) == nil {
						embeddingTexts = append(embeddingTexts, content)
					}
				}

				_ = rows.Close()
			}

			_ = adb.Close()
		}

		contextAssembler = &MemoryContextAssembler{
			ClaudeMDContent:   string(claudeMDContent),
			SkillDescriptions: skillDescs,
			Embeddings:        embeddingTexts,
		}
	}

	if db, dbErr := InitEmbeddingsDB(memoryRoot); dbErr == nil {
		score, scoreErr := ScoreClaudeMD(claudeMDPath, db, RealFS{}, extractor)
		if scoreErr == nil && score != nil {
			fmt.Printf("\n=== CLAUDE.md Quality Score ===\n")
			fmt.Printf("Overall: %s (%.1f)\n", score.OverallGrade, score.OverallScore)
			fmt.Printf("  Context Precision: %.1f  Faithfulness: %.1f  Currency: %.1f\n",
				score.ContextPrecision, score.Faithfulness, score.Currency)
			fmt.Printf("  Conciseness: %.1f  Coverage: %.1f\n",
				score.Conciseness, score.Coverage)

			if len(score.Issues) > 0 {
				fmt.Println("  Issues:")

				for _, issue := range score.Issues {
					fmt.Printf("    - %s\n", issue)
				}
			}
		}

		_ = db.Close()
	}

	opts := OptimizeInteractiveOpts{
		DBPath:           dbPath,
		ClaudeMDPath:     claudeMDPath,
		SkillsDir:        skillsDir,
		ReviewFunc:       reviewFunc,
		Input:            stdin,
		Output:           os.Stdout,
		Context:          ctx,
		Extractor:        extractor,
		TierFilter:       args.Tier,
		NoLLMEval:        args.NoLLMEval,
		ContextAssembler: contextAssembler,
	}

	result, err := OptimizeInteractive(opts)
	if err != nil {
		return fmt.Errorf("interactive optimization failed: %w", err)
	}

	fmt.Println("\n=== Memory Optimization Summary ===")
	fmt.Printf("Total proposals: %d\n", result.Total)
	fmt.Printf("  Approved: %d\n", result.Approved)
	fmt.Printf("  Rejected: %d\n", result.Rejected)
	fmt.Printf("  Applied: %d\n", result.Applied)

	if result.Failed > 0 {
		fmt.Printf("  Failed: %d\n", result.Failed)
	}

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

	if db2, err2 := InitEmbeddingsDB(memoryRoot); err2 == nil {
		budgetProposals, budgetErr := EnforceClaudeMDBudget(claudeMDPath, db2, RealFS{})
		_ = db2.Close()

		if budgetErr == nil && len(budgetProposals) > 0 {
			var recs []Recommendation
			for _, p := range budgetProposals {
				recs = append(recs, p.Recommendation)
			}

			printOptimizeRecommendationsSummary(recs)

			if err3 := offerSaveOptimizeRecommendations(recs, memoryRoot, args.Yes); err3 != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save recommendations: %v\n", err3)
			}
		}
	}

	return nil
}

func saveOptimizeRecommendations(path string, recommendations []*Recommendation) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Memory Recommendations\n\nGenerated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	for i, rec := range recommendations {
		fmt.Fprintf(&sb, "## Recommendation %d: %s\n\n", i+1, rec.Category)

		if rec.Description != "" {
			sb.WriteString("**Action**: " + rec.Description + "\n\n")
		}

		if rec.Evidence != "" {
			sb.WriteString("**Evidence**: " + rec.Evidence + "\n\n")
		}

		if rec.Text != "" {
			sb.WriteString(rec.Text + "\n\n")
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
