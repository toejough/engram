package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryExtract(args memory.ExtractArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunExtract(args, home)
}
