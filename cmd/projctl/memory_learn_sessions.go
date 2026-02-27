package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryLearnSessions(args memory.LearnSessionsArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunLearnSessions(args, home)
}
