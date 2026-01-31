package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/toejough/projctl/internal/parser"
	"github.com/toejough/projctl/internal/trace"
)

type traceAddArgs struct {
	Dir  string `targ:"flag,short=d,required,desc=Project directory"`
	From string `targ:"flag,short=f,required,desc=Source ID (e.g. ISSUE-001 or REQ-001)"`
	To   string `targ:"flag,short=t,required,desc=Target ID(s) comma-separated (e.g. REQ-001 or DES-001,ARCH-001)"`
}

func traceAdd(args traceAddArgs) error {
	targets := strings.Split(args.To, ",")
	for i := range targets {
		targets[i] = strings.TrimSpace(targets[i])
	}

	err := trace.Add(args.Dir, args.From, targets)
	if err != nil {
		return err
	}

	fmt.Printf("Trace link added: %s → %s\n", args.From, args.To)

	return nil
}

type traceValidateArgs struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func traceValidate(args traceValidateArgs) error {
	result, err := trace.Validate(args.Dir)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	fmt.Println(string(data))

	if !result.Pass {
		os.Exit(1)
	}

	return nil
}

type traceImpactArgs struct {
	Dir     string `targ:"flag,short=d,required,desc=Project directory"`
	ID      string `targ:"flag,short=i,required,desc=Traceability ID (e.g. ISSUE-001 or REQ-003)"`
	Reverse bool   `targ:"flag,short=r,desc=Backward (upstream) analysis"`
}

func traceImpact(args traceImpactArgs) error {
	result, err := trace.Impact(args.Dir, args.ID, args.Reverse)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	fmt.Println(string(data))

	return nil
}

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

type traceValidateV2Args struct {
	Dir string `targ:"flag,short=d,required,desc=Project directory"`
}

func traceValidateV2(args traceValidateV2Args) error {
	// Collect trace items from docs and test files
	fs := parser.NewRealFS()
	collectResult, err := parser.CollectTraceItems(args.Dir, fs)
	if err != nil {
		return fmt.Errorf("failed to collect trace items: %w", err)
	}

	// Validate using the graph-based system
	result, err := trace.ValidateV2(collectResult.Items)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	fmt.Println(string(data))

	if !result.Pass {
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

		fmt.Println("\nRun 'projctl trace add --from <source> --to <target>' to fix.")
	}

	if !result.Pass {
		os.Exit(1)
	}

	return nil
}
