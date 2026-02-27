package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryScoreSession(args memory.ScoreSessionArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunScoreSession(args, home, os.Stdin)
}
