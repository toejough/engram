package main

import (
	"fmt"
	"os"

	"github.com/toejough/projctl/internal/territory"
)

type mapGenerateArgs struct {
	Dir    string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Output string `targ:"flag,short=o,desc=Output file path (default: stdout)"`
}

func mapGenerate(args mapGenerateArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	m, err := territory.Generate(dir)
	if err != nil {
		return fmt.Errorf("failed to generate territory map: %w", err)
	}

	data, err := territory.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal territory map: %w", err)
	}

	if args.Output != "" {
		if err := os.WriteFile(args.Output, data, 0o644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Territory map written to: %s\n", args.Output)
		return nil
	}

	fmt.Print(string(data))
	return nil
}
