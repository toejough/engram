package main

import (
	"fmt"
	"os"

	"github.com/toejough/projctl/internal/memory"
)

type memoryFeedbackArgs struct {
	Index        int    `targ:"positional,desc=Index from last query results (1-based)"`
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

	// Resolve ID: either direct via --id or from positional index
	var embeddingID int64
	if args.ID > 0 {
		embeddingID = args.ID
	} else if args.Index > 0 {
		// Load last query results
		results, _, err := memory.LoadLastQueryResults(memoryRoot)
		if err != nil {
			return fmt.Errorf("failed to load last query results: %w", err)
		}
		if len(results) == 0 {
			return fmt.Errorf("no previous query results found - run a query first")
		}
		if args.Index > len(results) {
			return fmt.Errorf("index %d out of range (only %d results available)", args.Index, len(results))
		}
		embeddingID = results[args.Index-1].ID // Convert 1-based to 0-based
	} else {
		return fmt.Errorf("either positional index or --id must be provided")
	}

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
