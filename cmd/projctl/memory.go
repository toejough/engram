package main

import (
	"fmt"
	"os"

	"github.com/toejough/projctl/internal/memory"
)

type memoryLearnArgs struct {
	Message    string `targ:"flag,short=m,required,desc=Learning message to store"`
	Project    string `targ:"flag,short=p,desc=Project to tag the learning with"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memoryLearn(args memoryLearnArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	opts := memory.LearnOpts{
		Message:    args.Message,
		Project:    args.Project,
		MemoryRoot: memoryRoot,
	}

	if err := memory.Learn(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Learned: " + args.Message)
	return nil
}
