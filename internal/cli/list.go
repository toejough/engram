package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"engram/internal/memory"
)

func runList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("list: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("list: %w", defaultErr)
	}

	lister := memory.NewLister()

	memories, err := lister.ListAllMemories(*dataDir)
	if err != nil {
		// Empty data dir (no memory directories) is not an error for list.
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
