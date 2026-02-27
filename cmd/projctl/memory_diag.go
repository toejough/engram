package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryDiag(args memory.DiagArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunDiag(args, home)
}
