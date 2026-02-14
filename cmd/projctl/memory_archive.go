package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/toejough/projctl/internal/memory"
)

type memoryArchiveListArgs struct {
	MemoryRoot string `targ:"flag,desc=Path to memory root directory"`
	Limit      int    `targ:"flag,short=n,desc=Maximum entries to show (default 50)"`
}

func memoryArchiveList(args memoryArchiveListArgs) error {
	// Default memory root
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 50
	}

	db, err := memory.InitDBForTest(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	entries, err := memory.ListArchive(db, limit)
	if err != nil {
		return fmt.Errorf("failed to list archive: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No archived entries")
		return nil
	}

	for _, e := range entries {
		fmt.Printf("[%s] %s (ID:%d) — %s\n  %s\n\n", e.ArchivedAt, e.Action, e.EmbeddingID, e.Reason, truncate(e.Content, 100))
	}

	return nil
}
