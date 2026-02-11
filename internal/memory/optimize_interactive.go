package memory

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// OptimizeInteractiveOpts holds options for interactive memory optimization.
type OptimizeInteractiveOpts struct {
	DBPath       string                 // Path to embeddings.db
	ClaudeMDPath string                 // Path to CLAUDE.md
	SkillsDir    string                 // Path to skills directory
	ReviewFunc   MaintenanceReviewFunc  // Function to review proposals
	Input        io.Reader              // Input stream for interactive prompts (default os.Stdin)
	Output       io.Writer              // Output stream for display (default os.Stdout)
	Context      context.Context        // Context for cancellation
}

// OptimizeInteractiveResult holds the results of interactive optimization.
type OptimizeInteractiveResult struct {
	Total       int                          // Total proposals generated
	Approved    int                          // Proposals approved
	Rejected    int                          // Proposals rejected
	Applied     int                          // Proposals successfully applied
	Failed      int                          // Proposals that failed to apply
	TierSummary map[string]TierSummaryStats  // Per-tier statistics
}

// TierSummaryStats holds statistics for a single tier.
type TierSummaryStats struct {
	Total    int            // Total proposals from this tier
	Approved int            // Approved from this tier
	Rejected int            // Rejected from this tier
	Applied  int            // Successfully applied from this tier
	Failed   int            // Failed to apply from this tier
	Actions  map[string]int // Count by action type
}

// OptimizeInteractive runs interactive memory optimization with proposal-by-proposal review.
// Workflow:
// 1. Create transaction/backups
// 2. Scan embeddings tier → proposals
// 3. Scan skills tier → proposals
// 4. Scan CLAUDE.md tier → proposals
// 5. For each proposal: review → if approved, apply
// 6. On success: cleanup backups
// 7. On error: rollback all changes
// 8. Report summary
func OptimizeInteractive(opts OptimizeInteractiveOpts) (*OptimizeInteractiveResult, error) {
	// Check context
	if opts.Context != nil {
		if err := opts.Context.Err(); err != nil {
			return nil, err
		}
	}

	// Set defaults
	if opts.DBPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.DBPath = filepath.Join(home, ".claude", "memory", "embeddings.db")
	}
	if opts.ClaudeMDPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.ClaudeMDPath = filepath.Join(home, ".claude", "CLAUDE.md")
	}
	if opts.Input == nil {
		opts.Input = os.Stdin
	}
	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	// Initialize result
	result := &OptimizeInteractiveResult{
		TierSummary: make(map[string]TierSummaryStats),
	}

	// Create transaction for backup/rollback support
	txn := NewTransaction(TransactionOpts{
		DBPath:       opts.DBPath,
		ClaudeMDPath: opts.ClaudeMDPath,
		SkillsDir:    opts.SkillsDir,
	})

	// Create backups
	fmt.Fprintln(opts.Output, "Creating backups...")
	if err := txn.CreateBackups(); err != nil {
		return nil, fmt.Errorf("failed to create backups: %w", err)
	}

	// Ensure cleanup happens even on error
	defer func() {
		_ = txn.Cleanup()
	}()

	// Open database
	db, err := sql.Open("sqlite3", opts.DBPath)
	if err != nil {
		// Rollback on error
		_ = txn.Rollback()
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Scan all tiers and collect proposals
	fmt.Fprintln(opts.Output, "\nScanning memory tiers for maintenance opportunities...")

	allProposals := make([]MaintenanceProposal, 0)

	// Scan embeddings tier
	if err := checkContextErr(opts.Context); err != nil {
		_ = txn.Rollback()
		return nil, err
	}
	fmt.Fprintln(opts.Output, "- Scanning embeddings tier...")
	embeddingsProposals, err := scanEmbeddings(db, filepath.Dir(opts.DBPath), opts.SkillsDir)
	if err != nil {
		_ = txn.Rollback()
		return nil, fmt.Errorf("failed to scan embeddings tier: %w", err)
	}
	allProposals = append(allProposals, embeddingsProposals...)
	updateTierStats(result, embeddingsProposals, "embeddings")

	// Scan skills tier
	if opts.SkillsDir != "" {
		if err := checkContextErr(opts.Context); err != nil {
			_ = txn.Rollback()
			return nil, err
		}
		fmt.Fprintln(opts.Output, "- Scanning skills tier...")
		skillsScanner := NewSkillsScanner(db, SkillsScannerOpts{})
		skillsProposals, err := skillsScanner.Scan()
		if err != nil {
			_ = txn.Rollback()
			return nil, fmt.Errorf("failed to scan skills tier: %w", err)
		}
		allProposals = append(allProposals, skillsProposals...)
		updateTierStats(result, skillsProposals, "skills")
	}

	// Scan CLAUDE.md tier
	if err := checkContextErr(opts.Context); err != nil {
		_ = txn.Rollback()
		return nil, err
	}
	fmt.Fprintln(opts.Output, "- Scanning CLAUDE.md tier...")
	claudeMDProposals, err := ScanClaudeMD(RealFS{}, opts.ClaudeMDPath, 0.9)
	if err != nil {
		_ = txn.Rollback()
		return nil, fmt.Errorf("failed to scan CLAUDE.md tier: %w", err)
	}
	allProposals = append(allProposals, claudeMDProposals...)
	updateTierStats(result, claudeMDProposals, "claude-md")

	result.Total = len(allProposals)

	if len(allProposals) == 0 {
		fmt.Fprintln(opts.Output, "\nNo maintenance opportunities found.")
		return result, nil
	}

	fmt.Fprintf(opts.Output, "\nFound %d maintenance opportunities.\n\n", len(allProposals))

	// Review and apply each proposal
	for i, proposal := range allProposals {
		// Check context
		if err := checkContextErr(opts.Context); err != nil {
			_ = txn.Rollback()
			return nil, err
		}

		// Review proposal
		var approved bool
		if opts.ReviewFunc != nil {
			// Use MaintenanceReviewFunc (simple boolean)
			approved = opts.ReviewFunc(proposal)
		} else {
			// Use interactive reviewProposal (io-based)
			var err error
			approved, err = reviewProposal(proposal, opts.Input, opts.Output)
			if err != nil {
				_ = txn.Rollback()
				return nil, fmt.Errorf("review failed: %w", err)
			}
		}

		tierStats := result.TierSummary[proposal.Tier]
		if approved {
			result.Approved++
			tierStats.Approved++

			// Apply proposal
			if err := applyProposal(db, opts.DBPath, opts.ClaudeMDPath, opts.SkillsDir, proposal); err != nil {
				fmt.Fprintf(opts.Output, "Failed to apply proposal: %v\n", err)
				result.Failed++
				tierStats.Failed++
			} else {
				result.Applied++
				tierStats.Applied++

				// Record change in transaction log
				txn.RecordChange(ChangeRecord{
					Type:   proposal.Action,
					Tier:   proposal.Tier,
					Target: proposal.Target,
					Before: "",
					After:  proposal.Preview,
				})
			}
		} else {
			result.Rejected++
			tierStats.Rejected++
		}
		result.TierSummary[proposal.Tier] = tierStats

		// Progress indicator
		if (i+1)%10 == 0 {
			fmt.Fprintf(opts.Output, "Processed %d/%d proposals...\n", i+1, len(allProposals))
		}
	}

	// Success - cleanup backups
	if err := txn.Cleanup(); err != nil {
		fmt.Fprintf(opts.Output, "Warning: failed to cleanup backups: %v\n", err)
	}

	return result, nil
}

// checkContextErr checks if context is cancelled and returns the error.
func checkContextErr(ctx context.Context) error {
	if ctx != nil {
		return ctx.Err()
	}
	return nil
}

// updateTierStats updates the tier summary statistics.
func updateTierStats(result *OptimizeInteractiveResult, proposals []MaintenanceProposal, tier string) {
	stats, exists := result.TierSummary[tier]
	if !exists {
		stats = TierSummaryStats{
			Actions: make(map[string]int),
		}
	}

	stats.Total += len(proposals)
	for _, p := range proposals {
		stats.Actions[p.Action]++
	}

	result.TierSummary[tier] = stats
}

// applyProposal applies a maintenance proposal to the appropriate tier.
func applyProposal(db *sql.DB, dbPath, claudeMDPath, skillsDir string, proposal MaintenanceProposal) error {
	switch proposal.Tier {
	case "embeddings":
		memoryRoot := filepath.Dir(dbPath)
		return applyEmbeddingsProposal(db, memoryRoot, skillsDir, proposal)
	case "skills":
		applier := NewSkillsApplier(db, SkillsApplierOpts{
			SkillsDir:    skillsDir,
			ClaudeMDPath: claudeMDPath,
		})
		return applier.Apply(proposal)
	case "claude-md":
		return ApplyClaudeMDProposal(RealFS{}, claudeMDPath, proposal)
	default:
		return fmt.Errorf("unknown tier: %s", proposal.Tier)
	}
}
