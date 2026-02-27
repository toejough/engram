package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryArchiveList(args memory.ArchiveListArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunArchiveList(args, home)
}
