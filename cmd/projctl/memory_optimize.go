package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryOptimize(args memory.OptimizeArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunOptimize(args, home, os.Stdin)
}
