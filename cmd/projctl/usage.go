package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/config"
	"github.com/toejough/projctl/internal/log"
	"github.com/toejough/projctl/internal/usage"
)

type usageReportArgs struct {
	Dir     string `targ:"flag,short=d,desc=Project directory (use this or --project)"`
	Project string `targ:"flag,short=p,desc=Project name (looks up in ~/.projctl/projects/)"`
	Session string `targ:"flag,desc=Filter by session ID"`
	Model   string `targ:"flag,desc=Filter by model (haiku|sonnet|opus)"`
	Format  string `targ:"flag,short=f,desc=Output format: text (default) or json"`
}

func usageReport(args usageReportArgs) error {
	var report usage.UsageReport
	var err error

	opts := usage.ReportOpts{
		Model:   args.Model,
		Session: args.Session,
	}

	if args.Project != "" {
		// Use project-based lookup
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("failed to get home directory: %w", homeErr)
		}
		projctlDir := filepath.Join(homeDir, ".projctl")
		report, err = usage.ReportByProject(args.Project, projctlDir, opts, log.RealFS{})
	} else if args.Dir != "" {
		report, err = usage.Report(args.Dir, opts, log.RealFS{})
	} else {
		return fmt.Errorf("either --dir or --project is required")
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

	// Human-readable format
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

type usageCheckArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func usageCheck(args usageCheckArgs) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cfg, err := config.Load(args.Dir, homeDir, &osConfigFS{})
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	result := usage.Check(args.Dir, usage.BudgetConfig{
		WarningTokens: cfg.Budget.WarningTokens,
		LimitTokens:   cfg.Budget.LimitTokens,
	}, log.RealFS{})

	switch result.Status {
	case usage.StatusOK:
		fmt.Printf("Token usage: %d (within budget)\n", result.TotalTokens)
		return nil
	case usage.StatusWarning:
		fmt.Printf("⚠️  Token usage: %d (over warning threshold: %d)\n", result.TotalTokens, cfg.Budget.WarningTokens)
		fmt.Printf("Recommendation: %s\n", result.Recommendation)
		os.Exit(1)
	case usage.StatusLimit:
		fmt.Printf("❌ Token usage: %d (over limit: %d)\n", result.TotalTokens, cfg.Budget.LimitTokens)
		fmt.Printf("Recommendation: %s\n", result.Recommendation)
		os.Exit(2)
	}

	return nil
}
