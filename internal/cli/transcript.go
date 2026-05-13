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

	"github.com/toejough/engram/internal/learnmarker"
	"github.com/toejough/engram/internal/transcript"
)

// TranscriptArgs holds parsed flags for the transcript subcommand.
type TranscriptArgs struct {
	From          string `targ:"flag,name=from,desc=start date (YYYY-MM-DD or RFC3339); defaults to marker or 24h ago"`
	To            string `targ:"flag,name=to,desc=end date (YYYY-MM-DD or RFC3339); defaults to now"`
	TranscriptDir string `targ:"flag,name=transcript-dir,env=ENGRAM_TRANSCRIPT_DIR,desc=path to transcript directory"`
	ProjectSlug   string `targ:"flag,name=project-slug,desc=project slug for transcript-dir and marker derivation"`
	StateDir      string `targ:"flag,name=state-dir,env=ENGRAM_STATE_DIR,desc=state directory (defaults to XDG_STATE_HOME/engram)"`
	Mark          bool   `targ:"flag,name=mark,desc=advance the last-learn marker to now after reading"`
	MaxBytes      int    `targ:"flag,name=max-bytes,desc=byte cap for transcript output (default 200000)"`
}

const defaultMaxBytes = 200_000

// unexported variables.
var (
	errTranscriptInvalidDate = errors.New(
		"transcript: invalid date: expected YYYY-MM-DD or RFC3339",
	)
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

// emitTranscripts writes entries (oldest first) to stdout, capped at maxBytes
// of total content. When the cap is reached, oldest content is dropped —
// the most recent transcript wins. A one-line truncation notice is emitted
// when content was dropped.
func emitTranscripts(
	reader transcript.Reader,
	entries []transcript.FileEntry,
	maxBytes int,
	stdout io.Writer,
) error {
	contents := make([]string, 0, len(entries))
	total := 0
	for _, entry := range entries {
		content, _, readErr := reader.Read(entry.Path, math.MaxInt32)
		if readErr != nil {
			return fmt.Errorf("transcript: reading %s: %w", entry.Path, readErr)
		}
		contents = append(contents, content)
		total += len(content)
	}

	dropped := 0
	for total > maxBytes && len(contents) > 1 {
		total -= len(contents[0])
		dropped++
		contents = contents[1:]
	}

	if dropped > 0 {
		notice := fmt.Sprintf(
			"[engram transcript: dropped %d oldest session(s) to fit %d-byte cap]\n",
			dropped, maxBytes,
		)
		if _, err := io.WriteString(stdout, notice); err != nil {
			return fmt.Errorf("transcript: writing output: %w", err)
		}
	}

	for _, content := range contents {
		if _, err := io.WriteString(stdout, content); err != nil {
			return fmt.Errorf("transcript: writing output: %w", err)
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
	transcriptDir := args.TranscriptDir
	if err := applyTranscriptDirDefault(&transcriptDir, args.ProjectSlug, os.Getwd); err != nil {
		return err
	}

	stateDir := args.StateDir
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("transcript: resolving home dir: %w", err)
		}
		stateDir = learnmarker.StateDirFromHome(home, os.Getenv)
	}

	slug := args.ProjectSlug
	if slug == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("transcript: resolving working directory: %w", err)
		}
		slug = ProjectSlugFromPath(cwd)
	}

	markerPath := learnmarker.MarkerPath(stateDir, slug)
	markerTime, markerFound, err := learnmarker.Read(learnmarker.OSFS{}, markerPath)
	if err != nil {
		return fmt.Errorf("transcript: reading marker: %w", err)
	}

	now := time.Now().UTC()
	fromTime, toTime, err := ResolveTimeWindow(TimeWindowInputs{
		From: args.From, To: args.To,
		Marker: markerTime, MarkerFound: markerFound, Now: now,
	})
	if err != nil {
		return err
	}

	maxBytes := args.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
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
	slices.Reverse(filtered) // chronological for output

	if emitErr := emitTranscripts(reader, filtered, maxBytes, stdout); emitErr != nil {
		return emitErr
	}

	if args.Mark {
		if writeErr := learnmarker.Write(learnmarker.OSFS{}, markerPath, now); writeErr != nil {
			return fmt.Errorf("transcript: advancing marker: %w", writeErr)
		}
	}

	return nil
}
