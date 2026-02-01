package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/toejough/projctl/internal/result"
)

type resultValidateArgs struct {
	File   string `targ:"--file,-f,Path to result.toml file (required)"`
	Format string `targ:"--format,Output format: text (default) or json"`
}

// ValidationResult is the JSON output format for validation results.
type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	File    string   `json:"file"`
	Status  bool     `json:"status,omitempty"`
	Outputs int      `json:"outputs_count,omitempty"`
}

func resultValidate(args resultValidateArgs) error {
	if args.File == "" {
		return fmt.Errorf("--file is required")
	}

	data, err := os.ReadFile(args.File)
	if err != nil {
		if args.Format == "json" {
			out := ValidationResult{
				Valid:  false,
				Errors: []string{fmt.Sprintf("failed to read file: %v", err)},
				File:   args.File,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(out)
			os.Exit(1)
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	r, err := result.Parse(data)
	if err != nil {
		if args.Format == "json" {
			out := ValidationResult{
				Valid:  false,
				Errors: []string{err.Error()},
				File:   args.File,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(out)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Validation failed: %s\n", err)
		os.Exit(1)
	}

	if args.Format == "json" {
		out := ValidationResult{
			Valid:        true,
			File:         args.File,
			Status:       r.Status.Success,
			Outputs:      len(r.Outputs.FilesModified),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("Valid result file: %s\n", args.File)
	fmt.Printf("  Status: success=%v\n", r.Status.Success)
	fmt.Printf("  Outputs: %d files modified\n", len(r.Outputs.FilesModified))
	fmt.Printf("  Decisions: %d\n", len(r.Decisions))
	fmt.Printf("  Learnings: %d\n", len(r.Learnings))

	return nil
}
