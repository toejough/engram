package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

// unexported variables.
var (
	errFeedbackMissingName = errors.New("feedback: --name is required")
)

// RunFeedback parses feedback flags and increments the appropriate SBIA counters
// on the named memory TOML file.
func RunFeedback(args []string) error {
	fs := flag.NewFlagSet("feedback", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	name := fs.String("name", "", "memory slug")
	dataDir := fs.String("data-dir", "", "path to data directory")
	relevant := fs.Bool("relevant", false, "memory was relevant")
	irrelevant := fs.Bool("irrelevant", false, "memory was irrelevant")
	used := fs.Bool("used", false, "memory advice was followed")
	notused := fs.Bool("notused", false, "memory advice was not followed")

	if parseErr := fs.Parse(args); parseErr != nil {
		return fmt.Errorf("feedback: %w", parseErr)
	}

	if *name == "" {
		return errFeedbackMissingName
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("feedback: %w", defaultErr)
	}

	memPath := filepath.Join(*dataDir, "memories", *name+".toml")

	modifier := memory.NewModifier(memory.WithModifierWriter(tomlwriter.New()))

	return modifier.ReadModifyWrite(memPath, func(record *memory.MemoryRecord) {
		applyFeedbackCounters(record, *relevant, *irrelevant, *used, *notused)
		record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	})
}

// applyFeedbackCounters increments the appropriate counter on record based on
// the feedback flags, following the specified precedence rules.
func applyFeedbackCounters(record *memory.MemoryRecord, relevant, irrelevant, used, notused bool) {
	switch {
	case irrelevant:
		record.IrrelevantCount++
	case relevant && used:
		record.FollowedCount++
	case relevant && notused:
		record.NotFollowedCount++
	case used:
		record.FollowedCount++
	case notused:
		record.NotFollowedCount++
	}
}
