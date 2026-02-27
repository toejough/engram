package memory

import (
	"errors"
	"fmt"
	"path/filepath"
)

// RunFeedback records feedback for a memory entry or session.
func RunFeedback(args FeedbackArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	if args.SessionID != "" {
		feedbackType := args.Type
		switch feedbackType {
		case "helpful", "wrong", "unclear":
		default:
			return errors.New("--type must be one of: helpful, wrong, unclear")
		}

		db, err := InitDBForTest(memoryRoot)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}

		defer func() { _ = db.Close() }()

		if err := UpdateSurfacingFeedback(db, args.SessionID, feedbackType); err != nil {
			return fmt.Errorf("failed to update surfacing feedback: %w", err)
		}

		if err := RecordMemoryFeedback(db, args.SessionID, feedbackType); err != nil {
			return fmt.Errorf("failed to record memory feedback: %w", err)
		}

		fmt.Printf("Recorded %s feedback for session %s\n", feedbackType, args.SessionID)

		return nil
	}

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
		return errors.New("exactly one of --helpful, --wrong, or --unclear must be set")
	}

	var feedbackType FeedbackType
	if args.Helpful {
		feedbackType = FeedbackHelpful
	} else if args.Wrong {
		feedbackType = FeedbackWrong
	} else {
		feedbackType = FeedbackUnclear
	}

	if args.ID <= 0 {
		return errors.New("--id must be provided")
	}

	db, err := InitDBForTest(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	if err := RecordFeedback(db, args.ID, feedbackType); err != nil {
		return fmt.Errorf("failed to record feedback: %w", err)
	}

	fmt.Printf("Recorded %s feedback for memory ID %d\n", feedbackType, args.ID)

	return nil
}
