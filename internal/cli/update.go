package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

// applyUpdateArgs sets only non-empty field values on the record.
func applyUpdateArgs(record *memory.MemoryRecord, args UpdateArgs) {
	if args.Situation != "" {
		record.Situation = args.Situation
	}

	if args.Behavior != "" {
		record.Content.Behavior = args.Behavior
	}

	if args.Impact != "" {
		record.Content.Impact = args.Impact
	}

	if args.Action != "" {
		record.Content.Action = args.Action
	}

	if args.Subject != "" {
		record.Content.Subject = args.Subject
	}

	if args.Predicate != "" {
		record.Content.Predicate = args.Predicate
	}

	if args.Object != "" {
		record.Content.Object = args.Object
	}

	if args.Source != "" {
		record.Source = args.Source
	}
}

// runUpdate implements the update subcommand: modifies individual fields of an existing memory.
func runUpdate(ctx context.Context, args UpdateArgs, stdout io.Writer) error {
	_ = ctx

	if args.Source != "" {
		srcErr := validateSource(args.Source)
		if srcErr != nil {
			return fmt.Errorf("update: %w", srcErr)
		}
	}

	dataDir := args.DataDir

	defaultErr := applyDataDirDefault(&dataDir)
	if defaultErr != nil {
		return fmt.Errorf("update: %w", defaultErr)
	}

	memPath := memory.ResolveMemoryPath(dataDir, args.Name, fileExists)

	record, loadErr := loadMemoryTOML(memPath)
	if loadErr != nil {
		return fmt.Errorf("update: loading %s: %w", args.Name, loadErr)
	}

	applyUpdateArgs(record, args)

	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	writer := tomlwriter.New()

	writeErr := writer.AtomicWrite(memPath, record)
	if writeErr != nil {
		return fmt.Errorf("update: writing %s: %w", args.Name, writeErr)
	}

	name := memory.NameFromPath(memPath)

	_, printErr := fmt.Fprintf(stdout, "UPDATED: %s\n", name)
	if printErr != nil {
		return fmt.Errorf("update: %w", printErr)
	}

	return nil
}
