package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"engram/internal/memory"
)

// backfillSlugs iterates memories in dir, counting (and optionally writing) those with an empty
// project_slug. Returns the count of memories that were (or would be) updated.
func backfillSlugs(dir, slug string, apply bool) (int, error) {
	records, listErr := memory.ListAll(dir)
	if listErr != nil {
		return 0, fmt.Errorf("migrate-slugs: %w", listErr)
	}

	var count int

	for _, stored := range records {
		if stored.Record.ProjectSlug != "" {
			continue
		}

		count++

		if !apply {
			continue
		}

		storedPath := stored.Path

		writeErr := memory.ReadModifyWrite(storedPath, func(record *memory.MemoryRecord) {
			record.ProjectSlug = slug
		})
		if writeErr != nil {
			return count, fmt.Errorf("migrate-slugs: writing %s: %w", storedPath, writeErr)
		}
	}

	return count, nil
}

func runMigrateSlugs(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("migrate-slugs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	slug := fs.String("slug", "", "project slug to backfill (defaults to PWD-derived slug)")
	apply := fs.Bool("apply", false, "apply changes instead of dry-run")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("migrate-slugs: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("migrate-slugs: %w", defaultErr)
	}

	slugErr := applyProjectSlugDefault(slug)
	if slugErr != nil {
		return fmt.Errorf("migrate-slugs: %w", slugErr)
	}

	memoriesDir := filepath.Join(*dataDir, "memories")

	count, err := backfillSlugs(memoriesDir, *slug, *apply)
	if err != nil {
		return err
	}

	if *apply {
		_, _ = fmt.Fprintf(stdout, "[engram] backfilled project_slug=%s on %d memories\n", *slug, count)
	} else {
		_, _ = fmt.Fprintf(stdout, "[engram] would set project_slug=%s on %d memories\n", *slug, count)
	}

	return nil
}
