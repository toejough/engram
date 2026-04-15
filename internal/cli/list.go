package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"engram/internal/memory"
)

func runList(ctx context.Context, args ListArgs, stdout io.Writer) error {
	_ = ctx

	dataDir := args.DataDir

	defaultErr := applyDataDirDefault(&dataDir)
	if defaultErr != nil {
		return fmt.Errorf("list: %w", defaultErr)
	}

	lister := memory.NewLister()

	memories, err := lister.ListAllMemories(dataDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("list: %w", err)
	}

	for _, mem := range memories {
		name := memory.NameFromPath(mem.FilePath)

		_, writeErr := fmt.Fprintf(stdout, "%s | %s | %s\n", mem.Type, name, mem.Situation)
		if writeErr != nil {
			return fmt.Errorf("list: %w", writeErr)
		}
	}

	return nil
}
