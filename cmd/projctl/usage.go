package main

import (
	"encoding/json"
	"fmt"

	"github.com/toejough/projctl/internal/usage"
)

type usageReportArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Model  string `targ:"flag,desc=Filter by model (haiku|sonnet|opus)"`
	Format string `targ:"flag,short=f,desc=Output format: text (default) or json"`
}

func usageReport(args usageReportArgs) error {
	report, err := usage.Report(args.Dir, usage.ReportOpts{
		Model: args.Model,
	})
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
