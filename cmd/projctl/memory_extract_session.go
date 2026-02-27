package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryExtractSession(args memory.ExtractSessionArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunExtractSession(args, home, os.Stdin)
}
