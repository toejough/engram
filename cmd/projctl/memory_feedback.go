package main

import (
	"fmt"
	"os"

	"github.com/toejough/projctl/internal/memory"
)

type memoryFeedbackArgs struct {
	ID           int64  `targ:"flag,name=id,desc=Memory ID to give feedback on"`
	Helpful      bool   `targ:"flag,desc=Mark memory as helpful"`
	Wrong        bool   `targ:"flag,desc=Mark memory as wrong or not useful"`
	Unclear      bool   `targ:"flag,desc=Mark memory as unclear"`
	MemoryRoot   string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

func memoryFeedback(args memoryFeedbackArgs) error {
	// Determine memory root
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = home + "/.claude/memory"
	}

	// Mutual exclusivity: exactly one of --helpful/--wrong/--unclear
	feedbackCount := 0
	if args.Helpful {
		feedbackCount++
	}
	if args.Wrong {
		feedbackCount++
	}
	if args.Unclear {
		feedbackCount++
	}
	if feedbackCount != 1 {
		return fmt.Errorf("exactly one of --helpful, --wrong, or --unclear must be set")
	}

	// Determine feedback type
	var feedbackType memory.FeedbackType
	if args.Helpful {
		feedbackType = memory.FeedbackHelpful
	} else if args.Wrong {
		feedbackType = memory.FeedbackWrong
	} else {
		feedbackType = memory.FeedbackUnclear
	}

	// Resolve ID: must use --id flag
	if args.ID <= 0 {
		return fmt.Errorf("--id must be provided")
	}
	embeddingID := args.ID

	// Open database
	db, err := memory.InitDBForTest(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Record feedback
	if err := memory.RecordFeedback(db, embeddingID, feedbackType); err != nil {
		return fmt.Errorf("failed to record feedback: %w", err)
	}

	fmt.Printf("Recorded %s feedback for memory ID %d\n", feedbackType, embeddingID)
	return nil
}
