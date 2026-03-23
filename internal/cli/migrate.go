package cli

import (
	"flag"
	"fmt"
	"io"
)

func runMigrateScores(args []string, stdout, _ io.Writer) error {
	fs := flag.NewFlagSet("migrate-scores", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	apply := fs.Bool("apply", false, "apply consolidations instead of dry-run")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("migrate-scores: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("migrate-scores: %w", defaultErr)
	}

	// For now, output a placeholder message.
	// Full wiring with API token + MigrationRunner construction
	// will be done when the binary entry point is updated.
	dryRunLabel := "dry-run"
	if *apply {
		dryRunLabel = "apply"
	}

	_, _ = fmt.Fprintf(
		stdout,
		"[engram] migrate-scores (%s) from %s\n",
		dryRunLabel,
		*dataDir,
	)

	return nil
}
