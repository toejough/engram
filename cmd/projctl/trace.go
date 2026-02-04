package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/toejough/projctl/internal/trace"
)

type traceRepairArgs struct {
	Dir  string `targ:"flag,short=d,required,desc=Project directory"`
	JSON bool   `targ:"flag,short=j,desc=Output as JSON"`
}

func traceRepair(args traceRepairArgs) error {
	result, err := trace.Repair(args.Dir)
	if err != nil {
		return err
	}

	if args.JSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode result: %w", err)
		}
		fmt.Println(string(data))
	} else {
		if len(result.DanglingRefs) == 0 && len(result.DuplicateIDs) == 0 {
			fmt.Println("No traceability issues found")
			return nil
		}

		fmt.Println("Traceability Issues Found")
		fmt.Println("=========================")

		if len(result.DanglingRefs) > 0 {
			fmt.Printf("\nDangling references (%d):\n", len(result.DanglingRefs))
			for _, ref := range result.DanglingRefs {
				fmt.Printf("  - %s (referenced but not defined)\n", ref)
			}
		}

		if len(result.DuplicateIDs) > 0 {
			fmt.Printf("\nDuplicate IDs (%d):\n", len(result.DuplicateIDs))
			for _, id := range result.DuplicateIDs {
				fmt.Printf("  - %s (defined multiple times)\n", id)
			}
		}

		os.Exit(1)
	}

	return nil
}

type traceValidateArtifactsArgs struct {
	Dir   string `targ:"flag,short=d,required,desc=Project directory"`
	Phase string `targ:"flag,short=p,desc=Workflow phase for phase-aware validation (e.g. architect-complete)"`
	JSON  bool   `targ:"flag,short=j,desc=Output as JSON"`
}

func traceValidateArtifacts(args traceValidateArtifactsArgs) error {
	var result trace.ValidateV2ArtifactsResult
	var err error

	if args.Phase != "" {
		result, err = trace.ValidateV2Artifacts(args.Dir, args.Phase)
	} else {
		result, err = trace.ValidateV2Artifacts(args.Dir)
	}
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if args.JSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode result: %w", err)
		}
		fmt.Println(string(data))
	} else {
		if result.Pass {
			fmt.Println("Validation passed: all IDs properly linked")
			return nil
		}

		fmt.Println("Validation Failed")
		fmt.Println("=================")

		if len(result.OrphanIDs) > 0 {
			fmt.Printf("\nOrphan IDs (%d) - referenced in **Traces to:** but not defined:\n", len(result.OrphanIDs))
			for _, id := range result.OrphanIDs {
				fmt.Printf("  - %s\n", id)
			}
		}

		if len(result.UnlinkedIDs) > 0 {
			fmt.Printf("\nUnlinked IDs (%d) - defined but not connected to chain:\n", len(result.UnlinkedIDs))
			for _, id := range result.UnlinkedIDs {
				fmt.Printf("  - %s\n", id)
			}
		}

		fmt.Println("\nFix by adding **Traces to:** fields in the artifact markdown files.")
	}

	if !result.Pass {
		os.Exit(1)
	}

	return nil
}

type traceShowArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	Format string `targ:"flag,short=f,desc=Output format: ascii (default) or json"`
}

func traceShow(args traceShowArgs) error {
	format := args.Format
	if format == "" {
		format = "ascii"
	}

	output, err := trace.Show(args.Dir, format)
	if err != nil {
		return err
	}

	fmt.Print(output)

	return nil
}

type tracePromoteArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Project directory"`
	DryRun bool   `targ:"flag,short=n,desc=Show what would be changed without modifying files"`
	JSON   bool   `targ:"flag,short=j,desc=Output as JSON"`
}

func tracePromote(args tracePromoteArgs) error {
	result, err := trace.Promote(args.Dir, args.DryRun)
	if err != nil {
		return err
	}

	if args.JSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode result: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Count unique files modified
	filesModified := make(map[string]bool)
	for _, p := range result.Promotions {
		filesModified[p.File] = true
	}

	if args.DryRun {
		fmt.Println("Trace promotion (dry run)")
		fmt.Println("=========================")
	} else {
		fmt.Println("Trace promotion complete")
		fmt.Println("========================")
	}

	if len(result.Promotions) > 0 {
		fmt.Printf("\nPromotions (%d):\n", len(result.Promotions))
		for _, p := range result.Promotions {
			fmt.Printf("  %s:%d: %s -> %s\n", p.File, p.Line, p.OldTrace, p.NewTrace)
		}
	}

	if len(result.Skipped) > 0 {
		fmt.Printf("\nSkipped (%d):\n", len(result.Skipped))
		for _, s := range result.Skipped {
			fmt.Printf("  %s:%d: %s - %s\n", s.File, s.Line, s.TaskID, s.Reason)
		}
	}

	if len(result.Promotions) == 0 && len(result.Skipped) == 0 {
		fmt.Println("\nNo TASK traces found in test files.")
	} else {
		fileWord := "file"
		if len(filesModified) != 1 {
			fileWord = "files"
		}
		if args.DryRun {
			fmt.Printf("\nWould modify %d %s.\n", len(filesModified), fileWord)
		} else {
			fmt.Printf("\nModified %d %s.\n", len(filesModified), fileWord)
		}
	}

	return nil
}
