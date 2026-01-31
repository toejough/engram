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
	Dir  string `targ:"flag,short=d,required,desc=Project directory"`
	JSON bool   `targ:"flag,short=j,desc=Output as JSON"`
}

func traceValidateArtifacts(args traceValidateArtifactsArgs) error {
	result, err := trace.ValidateV2Artifacts(args.Dir)
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
