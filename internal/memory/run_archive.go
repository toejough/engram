package memory

import (
	"fmt"
	"path/filepath"
)

// RunArchiveList lists archived memory entries.
func RunArchiveList(args ArchiveListArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	limit := args.Limit
	if limit == 0 {
		limit = 50
	}

	db, err := InitDBForTest(memoryRoot)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	entries, err := ListArchive(db, limit)
	if err != nil {
		return fmt.Errorf("failed to list archive: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No archived entries")
		return nil
	}

	for _, e := range entries {
		fmt.Printf("[%s] %s (ID:%d) \u2014 %s\n  %s\n\n", e.ArchivedAt, e.Action, e.EmbeddingID, e.Reason, archiveTruncateContent(e.Content, 100))
	}

	return nil
}

func archiveTruncateContent(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen] + "..."
}
