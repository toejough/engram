package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"time"

	"github.com/toejough/engram/internal/transcript"
)

// TranscriptArgs holds parsed flags for the transcript subcommand.
type TranscriptArgs struct {
	From          string `targ:"flag,name=from,required,desc=start date (YYYY-MM-DD or RFC3339 inclusive)"`
	To            string `targ:"flag,name=to,required,desc=end date (YYYY-MM-DD or RFC3339 inclusive)"`
	TranscriptDir string `targ:"flag,name=transcript-dir,env=ENGRAM_TRANSCRIPT_DIR,desc=path to transcript directory"`
	ProjectSlug   string `targ:"flag,name=project-slug,desc=project slug for default transcript-dir derivation"`
}

// unexported variables.
var (
	errTranscriptFromRequired = errors.New("transcript: --from is required")
	errTranscriptInvalidDate  = errors.New(
		"transcript: invalid date: expected YYYY-MM-DD or RFC3339",
	)
	errTranscriptToRequired = errors.New("transcript: --to is required")
)

// applyTranscriptDirDefault sets *dir to the ~/.claude/projects/<slug> path when empty.
// slug is derived from ProjectSlug or from PWD when ProjectSlug is empty.
func applyTranscriptDirDefault(dir *string, slug string, getwd func() (string, error)) error {
	if *dir != "" {
		return nil
	}

	if slug == "" {
		cwd, err := getwd()
		if err != nil {
			return fmt.Errorf("transcript: resolving working directory: %w", err)
		}

		slug = ProjectSlugFromPath(cwd)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("transcript: resolving home directory: %w", err)
	}

	*dir = home + "/.claude/projects/" + slug

	return nil
}

// emitTranscripts reads and writes stripped content for the given entries.
func emitTranscripts(
	reader transcript.Reader,
	entries []transcript.FileEntry,
	stdout io.Writer,
) error {
	const budgetBytes = math.MaxInt32

	for _, entry := range entries {
		content, _, readErr := reader.Read(entry.Path, budgetBytes)
		if readErr != nil {
			return fmt.Errorf("transcript: reading %s: %w", entry.Path, readErr)
		}

		_, writeErr := io.WriteString(stdout, content)
		if writeErr != nil {
			return fmt.Errorf("transcript: writing output: %w", writeErr)
		}
	}

	return nil
}

// filterByDateRange returns entries whose Mtime falls in [from, to] inclusive.
func filterByDateRange(entries []transcript.FileEntry, from, to time.Time) []transcript.FileEntry {
	filtered := make([]transcript.FileEntry, 0, len(entries))

	for _, entry := range entries {
		if !entry.Mtime.Before(from) && !entry.Mtime.After(to) {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

// TimeWindowInputs is the resolution input for ResolveTimeWindow. From/To are
// raw CLI strings (may be empty); Marker is the marker timestamp; MarkerFound
// distinguishes a missing-marker first-run from a zero-time marker. Now is the
// current time, injected for testability.
type TimeWindowInputs struct {
	From, To    string
	Marker      time.Time
	MarkerFound bool
	Now         time.Time
}

const defaultLookback = 24 * time.Hour

// ResolveTimeWindow returns the effective (from, to) time range for a
// transcript scan. Precedence: explicit --from > marker > now - 24h.
// Explicit --to > now. A date-only To ("YYYY-MM-DD") is extended to
// end-of-day for inclusive semantics; a date-only From parses as
// midnight start-of-day (no extension applied).
func ResolveTimeWindow(in TimeWindowInputs) (time.Time, time.Time, error) {
	from := in.Now.Add(-defaultLookback)
	if in.MarkerFound {
		from = in.Marker
	}

	if in.From != "" {
		parsed, err := parseDate(in.From)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}

		from = parsed
	}

	to := in.Now
	if in.To != "" {
		parsed, err := parseDate(in.To)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}

		if len(in.To) == len("2006-01-02") {
			parsed = parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}

		to = parsed
	}

	return from, to, nil
}

// parseDate parses a date string, accepting RFC3339 or YYYY-MM-DD layout.
func parseDate(s string) (time.Time, error) {
	// Try RFC3339 first.
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	// Fall back to YYYY-MM-DD.
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("%w: %q", errTranscriptInvalidDate, s)
}

// runTranscript reads session transcripts in [from, to] (inclusive) and emits stripped content.
func runTranscript(_ context.Context, args TranscriptArgs, stdout io.Writer) error {
	if args.From == "" {
		return errTranscriptFromRequired
	}

	if args.To == "" {
		return errTranscriptToRequired
	}

	fromTime, err := parseDate(args.From)
	if err != nil {
		return err
	}

	toTime, err := parseDate(args.To)
	if err != nil {
		return err
	}

	// If --to was given as YYYY-MM-DD, extend to end-of-day for inclusive semantics.
	if len(args.To) == len("2006-01-02") {
		toTime = toTime.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}

	transcriptDir := args.TranscriptDir

	dirErr := applyTranscriptDirDefault(&transcriptDir, args.ProjectSlug, os.Getwd)
	if dirErr != nil {
		return dirErr
	}

	lister := &osDirLister{}
	finder := transcript.NewCompositeSessionFinder(transcript.NewSessionFinder(lister))
	fileReader := &osFileReader{}
	reader := transcript.NewCompositeTranscriptReader(transcript.NewJSONLReader(fileReader))

	entries, findErr := finder.Find(transcriptDir)
	if findErr != nil {
		return fmt.Errorf("transcript: finding sessions: %w", findErr)
	}

	filtered := filterByDateRange(entries, fromTime, toTime)

	// Finder returns newest-first; reverse for chronological output.
	slices.Reverse(filtered)

	return emitTranscripts(reader, filtered, stdout)
}
