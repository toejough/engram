package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

// unexported variables.
var (
	errUpdateMissingName = errors.New("update: --name is required")
)

// unexported types.

// updateFlags holds parsed flags for the update command.
type updateFlags struct {
	name      *string
	dataDir   *string
	situation *string
	behavior  *string
	impact    *string
	action    *string
	subject   *string
	predicate *string
	object    *string
	source    *string
}

// applyUpdateFields sets only non-empty field values on the record.
func applyUpdateFields(record *memory.MemoryRecord, flags updateFlags) {
	if *flags.situation != "" {
		record.Situation = *flags.situation
	}

	if *flags.behavior != "" {
		record.Content.Behavior = *flags.behavior
	}

	if *flags.impact != "" {
		record.Content.Impact = *flags.impact
	}

	if *flags.action != "" {
		record.Content.Action = *flags.action
	}

	if *flags.subject != "" {
		record.Content.Subject = *flags.subject
	}

	if *flags.predicate != "" {
		record.Content.Predicate = *flags.predicate
	}

	if *flags.object != "" {
		record.Content.Object = *flags.object
	}

	if *flags.source != "" {
		record.Source = *flags.source
	}
}

func registerUpdateFlags(fs *flag.FlagSet) updateFlags {
	return updateFlags{
		name:      fs.String("name", "", "memory slug (required)"),
		dataDir:   fs.String("data-dir", "", "path to data directory"),
		situation: fs.String("situation", "", "context when this applies"),
		behavior:  fs.String("behavior", "", "observed behavior"),
		impact:    fs.String("impact", "", "impact of the behavior"),
		action:    fs.String("action", "", "recommended action"),
		subject:   fs.String("subject", "", "subject of the fact"),
		predicate: fs.String("predicate", "", "relationship or verb"),
		object:    fs.String("object", "", "object of the fact"),
		source:    fs.String("source", "", "human or agent"),
	}
}

// runUpdate implements the update subcommand: modifies individual fields of an existing memory.
func runUpdate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	flags := registerUpdateFlags(fs)

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("update: %w", parseErr)
	}

	if *flags.name == "" {
		return errUpdateMissingName
	}

	if *flags.source != "" {
		srcErr := validateSource(*flags.source)
		if srcErr != nil {
			return fmt.Errorf("update: %w", srcErr)
		}
	}

	defaultErr := applyDataDirDefault(flags.dataDir)
	if defaultErr != nil {
		return fmt.Errorf("update: %w", defaultErr)
	}

	memPath := memory.ResolveMemoryPath(*flags.dataDir, *flags.name, fileExists)

	record, loadErr := loadMemoryTOML(memPath)
	if loadErr != nil {
		return fmt.Errorf("update: loading %s: %w", *flags.name, loadErr)
	}

	applyUpdateFields(record, flags)

	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	writer := tomlwriter.New()

	writeErr := writer.AtomicWrite(memPath, record)
	if writeErr != nil {
		return fmt.Errorf("update: writing %s: %w", *flags.name, writeErr)
	}

	name := memory.NameFromPath(memPath)

	_, printErr := fmt.Fprintf(stdout, "UPDATED: %s\n", name)
	if printErr != nil {
		return fmt.Errorf("update: %w", printErr)
	}

	return nil
}
