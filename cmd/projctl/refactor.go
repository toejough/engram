package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/toejough/projctl/internal/refactor"
)

type refactorCapabilitiesArgs struct {
	Format string `targ:"flag,short=f,desc=Output format: text (default) or json"`
}

func refactorCapabilities(args refactorCapabilitiesArgs) error {
	caps := refactor.CheckCapabilities()

	if args.Format == "json" {
		data, err := json.MarshalIndent(caps, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Text output
	fmt.Println("Refactoring Capabilities")
	fmt.Println("========================")
	fmt.Printf("gopls available: %v\n", caps.GoplsAvailable)
	if caps.GoplsVersion != "" {
		fmt.Printf("gopls version:   %s\n", caps.GoplsVersion)
	}
	fmt.Printf("rename support:  %v\n", caps.RenameSupport)

	if !caps.GoplsAvailable {
		fmt.Printf("\n%s\n", refactor.GoplsInstallInstructions())
	}

	return nil
}

type refactorRenameArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Directory containing Go code"`
	Symbol string `targ:"flag,short=s,required,desc=Symbol to rename"`
	To     string `targ:"flag,short=t,required,desc=New name for the symbol"`
}

func refactorRename(args refactorRenameArgs) error {
	opts := refactor.RenameOpts{
		Dir:    args.Dir,
		Symbol: args.Symbol,
		To:     args.To,
	}

	result, err := refactor.Rename(opts)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "symbol not found") {
			fmt.Fprintf(os.Stderr, "Error: symbol not found: %s\n", args.Symbol)
			os.Exit(1)
		}
		if strings.Contains(errStr, "conflict") {
			fmt.Fprintf(os.Stderr, "Error: conflict: symbol %q already exists\n", args.To)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Renamed %s in %d files\n", args.Symbol, result.FilesChanged)
	return nil
}

type refactorExtractFunctionArgs struct {
	File  string `targ:"flag,short=f,required,desc=File containing code to extract"`
	Lines string `targ:"flag,short=l,required,desc=Line range to extract (e.g. 10-15)"`
	Name  string `targ:"flag,short=n,required,desc=Name for the extracted function"`
}

func refactorExtractFunction(args refactorExtractFunctionArgs) error {
	// Parse line range
	var startLine, endLine int
	_, err := fmt.Sscanf(args.Lines, "%d-%d", &startLine, &endLine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid line range format %q (expected START-END)\n", args.Lines)
		os.Exit(1)
	}

	opts := refactor.ExtractOpts{
		File:      args.File,
		StartLine: startLine,
		EndLine:   endLine,
		Name:      args.Name,
	}

	result, err := refactor.ExtractFunction(opts)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "file not found") {
			fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", args.File)
			os.Exit(1)
		}
		if strings.Contains(errStr, "invalid line range") {
			fmt.Fprintf(os.Stderr, "Error: %s\n", errStr)
			os.Exit(1)
		}
		if strings.Contains(errStr, "invalid function name") {
			fmt.Fprintf(os.Stderr, "Error: %s\n", errStr)
			os.Exit(1)
		}
		if strings.Contains(errStr, "function name conflict") {
			fmt.Fprintf(os.Stderr, "Error: %s\n", errStr)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Extracted function %s\n", result.ExtractedFunction)
	return nil
}
