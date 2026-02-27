package main

import (
	"os"

	"github.com/toejough/projctl/internal/memory"
)

func memoryFeedback(args memory.FeedbackArgs) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	return memory.RunFeedback(args, home)
}
