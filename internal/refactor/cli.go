package refactor

import (
	"encoding/json"
	"fmt"
)

// CapabilitiesArgs holds arguments for the refactor capabilities command.
type CapabilitiesArgs struct {
	Format string `targ:"flag,short=f,desc=Output format: text (default) or json"`
}

// ExtractFunctionArgs holds arguments for the refactor extract-function command.
type ExtractFunctionArgs struct {
	File  string `targ:"flag,short=f,required,desc=File containing code to extract"`
	Lines string `targ:"flag,short=l,required,desc=Line range to extract (e.g. 10-15)"`
	Name  string `targ:"flag,short=n,required,desc=Name for the extracted function"`
}

// RenameArgs holds arguments for the refactor rename command.
type RenameArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Directory containing Go code"`
	Symbol string `targ:"flag,short=s,required,desc=Symbol to rename"`
	To     string `targ:"flag,short=t,required,desc=New name for the symbol"`
}

// RunCapabilities checks available refactoring capabilities.
func RunCapabilities(args CapabilitiesArgs) error {
	caps := CheckCapabilities()

	if args.Format == "json" {
		data, err := json.MarshalIndent(caps, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(data))

		return nil
	}

	fmt.Println("Refactoring Capabilities")
	fmt.Println("========================")
	fmt.Printf("gopls available: %v\n", caps.GoplsAvailable)

	if caps.GoplsVersion != "" {
		fmt.Printf("gopls version:   %s\n", caps.GoplsVersion)
	}

	fmt.Printf("rename support:  %v\n", caps.RenameSupport)

	if !caps.GoplsAvailable {
		fmt.Printf("\n%s\n", GoplsInstallInstructions())
	}

	return nil
}

// RunExtractFunction extracts code into a new function using LSP.
func RunExtractFunction(args ExtractFunctionArgs) error {
	var startLine, endLine int

	_, err := fmt.Sscanf(args.Lines, "%d-%d", &startLine, &endLine)
	if err != nil {
		return fmt.Errorf("invalid line range format %q (expected START-END)", args.Lines)
	}

	result, err := ExtractFunction(ExtractOpts{
		File:      args.File,
		StartLine: startLine,
		EndLine:   endLine,
		Name:      args.Name,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Extracted function %s\n", result.ExtractedFunction)

	return nil
}

// RunRename renames a symbol using LSP.
func RunRename(args RenameArgs) error {
	result, err := Rename(RenameOpts(args))
	if err != nil {
		return err
	}

	fmt.Printf("Renamed %s in %d files\n", args.Symbol, result.FilesChanged)

	return nil
}
