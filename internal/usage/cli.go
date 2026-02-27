package usage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/config"
	"github.com/toejough/projctl/internal/log"
)

// CheckArgs holds arguments for the usage check command.
type CheckArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

// ReportArgs holds arguments for the usage report command.
type ReportArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (use this or --project)"`
	Project string `targ:"flag,short=p,desc=Project name (looks up in ~/.projctl/projects/)"`
	Session string `targ:"flag,desc=Filter by session ID"`
	Model   string `targ:"flag,desc=Filter by model (haiku|sonnet|opus)"`
	Format  string `targ:"flag,short=f,desc=Output format: text (default) or json"`
}

// RunCheck checks token usage against budget thresholds.
func RunCheck(args CheckArgs) error {
	return RunCheckCore(args, os.Exit)
}

// RunCheckCore is the testable core of RunCheck, accepting an injectable exit function.
func RunCheckCore(args CheckArgs, exit func(int)) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cfg, err := config.Load(args.Dir, homeDir, config.RealFS{})
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	result := Check(args.Dir, BudgetConfig{
		WarningTokens: cfg.Budget.WarningTokens,
		LimitTokens:   cfg.Budget.LimitTokens,
	}, log.RealFS{})

	switch result.Status {
	case StatusOK:
		fmt.Printf("Token usage: %d (within budget)\n", result.TotalTokens)
		return nil
	case StatusWarning:
		fmt.Printf("⚠️  Token usage: %d (over warning threshold: %d)\n", result.TotalTokens, cfg.Budget.WarningTokens)
		fmt.Printf("Recommendation: %s\n", result.Recommendation)
		exit(1)

		return nil
	case StatusLimit:
		fmt.Printf("❌ Token usage: %d (over limit: %d)\n", result.TotalTokens, cfg.Budget.LimitTokens)
		fmt.Printf("Recommendation: %s\n", result.Recommendation)
		exit(2)

		return nil
	}

	return nil
}

// RunReport generates a token usage report.
func RunReport(args ReportArgs) error {
	var (
		report UsageReport
		err    error
	)

	opts := ReportOpts{
		Model:   args.Model,
		Session: args.Session,
	}

	if args.Project != "" {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("failed to get home directory: %w", homeErr)
		}

		projctlDir := filepath.Join(homeDir, ".projctl")
		report, err = ReportByProject(args.Project, projctlDir, opts, log.RealFS{})
	} else if args.Dir != "" {
		report, err = Report(args.Dir, opts, log.RealFS{})
	} else {
		return errors.New("either --dir or --project is required")
	}

	if err != nil {
		return err
	}

	if args.Format == "json" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal report: %w", err)
		}

		fmt.Println(string(data))

		return nil
	}

	fmt.Printf("Token Usage Report\n")
	fmt.Printf("==================\n\n")
	fmt.Printf("Total tokens:  %d\n", report.TotalTokens)
	fmt.Printf("Entry count:   %d\n", report.EntryCount)

	if len(report.ByModel) > 0 {
		fmt.Printf("\nBy Model:\n")

		for model, tokens := range report.ByModel {
			if model == "" {
				model = "(unspecified)"
			}

			fmt.Printf("  %-15s %d\n", model, tokens)
		}
	}

	return nil
}
