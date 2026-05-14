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

// TranscriptArgs holds parsed flags for the transcript subcommand.
type TranscriptArgs struct {
	From          string `targ:"flag,name=from,desc=start date (YYYY-MM-DD or RFC3339); defaults to marker or 24h ago"`
	To            string `targ:"flag,name=to,desc=end date (YYYY-MM-DD or RFC3339); defaults to now"`
	TranscriptDir string `targ:"flag,name=transcript-dir,env=ENGRAM_TRANSCRIPT_DIR,desc=path to transcript directory"`
	ProjectSlug   string `targ:"flag,name=project-slug,desc=project slug for transcript-dir and marker derivation"`
	StateDir      string `targ:"flag,name=state-dir,env=ENGRAM_STATE_DIR,desc=state dir (XDG_STATE_HOME/engram default)"`
	Mark          bool   `targ:"flag,name=mark,desc=advance the last-learn marker to now after reading"`
	MaxBytes      int    `targ:"flag,name=max-bytes,desc=byte cap for transcript output (default 200000)"`
}

// ResolveTimeWindow returns the effective (from, to) time range for a
// transcript scan. Precedence: explicit --from > marker > now - 24h.
// Explicit --to > now. A date-only To ("YYYY-MM-DD") is extended to
// end-of-day for inclusive semantics; a date-only From parses as
// midnight start-of-day (no extension applied).
func ResolveTimeWindow(inputs TimeWindowInputs) (time.Time, time.Time, error) {
	from := inputs.Now.Add(-defaultLookback)
	if inputs.MarkerFound {
		from = inputs.Marker
	}

	if inputs.From != "" {
		parsed, err := parseDate(inputs.From)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}

		from = parsed
	}

	toTime := inputs.Now
	if inputs.To != "" {
		parsed, err := parseDate(inputs.To)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}

		if len(inputs.To) == len(dateFormat) {
			parsed = parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}

		toTime = parsed
	}

	return from, toTime, nil
}

// unexported constants.
const (
	defaultLookback = 24 * time.Hour
	defaultMaxBytes = 200_000
)

// unexported variables.
var (
	errTranscriptInvalidDate = errors.New(
		"transcript: invalid date: expected YYYY-MM-DD or RFC3339",
	)
)

type harnessMarker struct {
	path  string
	mtime time.Time
	found bool
}

type transcriptState struct {
	dir     string
	markers map[string]harnessMarker
}

// advanceAndReportMarker computes the effective scan end (Mtime of last
// fully-included entry if the cap was hit, otherwise `now`), writes the marker,
// and emits the status line to stdout.
func advanceAndReportMarker(
	markerPath string,
	fromTime, lastIncluded time.Time,
	hadEntries bool,
	now time.Time,
	stdout io.Writer,
) error {
	effectiveEnd := now
	if hadEntries && lastIncluded.Before(now) {
		effectiveEnd = lastIncluded
	}

	writeErr := learnmarker.Write(learnmarker.OSFS{}, markerPath, effectiveEnd)
	if writeErr != nil {
		return fmt.Errorf("transcript: advancing marker: %w", writeErr)
	}

	_, statusErr := fmt.Fprintf(
		stdout,
		"[engram transcript: scanned [%s, %s]; marker advanced to %s]\n",
		fromTime.UTC().Format(time.RFC3339Nano),
		effectiveEnd.UTC().Format(time.RFC3339Nano),
		effectiveEnd.UTC().Format(time.RFC3339Nano),
	)
	if statusErr != nil {
		return fmt.Errorf("transcript: writing status: %w", statusErr)
	}

	return nil
}

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

// emitTranscripts emits content chronologically (oldest first), stopping when
// the next entry's content would push total bytes over maxBytes. The first
// entry is always included even if it alone exceeds the cap — a single
// oversized entry must not stall marker progress forever. Returns the Mtime of
// the last fully-included entry per source and a hadEntries bool per source;
// callers use these to advance the per-source markers to the actual scan boundary.
func emitTranscripts(
	reader transcript.Reader,
	entries []transcript.FileEntry,
	maxBytes int,
	stdout io.Writer,
) (map[string]time.Time, map[string]bool, error) {
	lastIncluded := make(map[string]time.Time)
	hadEntries := make(map[string]bool)

	var total int

	for index, entry := range entries {
		content, _, readErr := reader.Read(entry.Path, math.MaxInt32)
		if readErr != nil {
			return nil, nil, fmt.Errorf("transcript: reading %s: %w", entry.Path, readErr)
		}

		// First entry is always included (progress guarantee). Subsequent entries
		// stop the scan when their content would push total over maxBytes.
		if index > 0 && total+len(content) > maxBytes {
			break
		}

		_, writeErr := io.WriteString(stdout, content)
		if writeErr != nil {
			return nil, nil, fmt.Errorf("transcript: writing output: %w", writeErr)
		}

		total += len(content)
		lastIncluded[entry.Source] = entry.Mtime
		hadEntries[entry.Source] = true
	}

	return lastIncluded, hadEntries, nil
}

// filterBySourceMarkers returns entries whose Mtime falls within each source's
// [from, to] range, where from is derived from that source's marker.
func filterBySourceMarkers(
	entries []transcript.FileEntry,
	markers map[string]harnessMarker,
	now time.Time,
) []transcript.FileEntry {
	filtered := make([]transcript.FileEntry, 0, len(entries))

	for _, entry := range entries {
		marker, ok := markers[entry.Source]
		if !ok {
			continue
		}

		from := now.Add(-defaultLookback)
		if marker.found {
			from = marker.mtime
		}

		if !entry.Mtime.Before(from) && !entry.Mtime.After(now) {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

// fromForSource returns the effective from time for a given source's marker.
func fromForSource(_ string, marker harnessMarker, now time.Time) time.Time {
	if marker.found {
		return marker.mtime
	}

	return now.Add(-defaultLookback)
}

// parseDate parses a date string, accepting RFC3339 or YYYY-MM-DD layout.
func parseDate(s string) (time.Time, error) {
	// Try RFC3339 first.
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	// Fall back to YYYY-MM-DD.
	t, err = time.Parse(dateFormat, s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("%w: %q", errTranscriptInvalidDate, s)
}

// resolveMaxBytes returns the effective byte cap, defaulting to defaultMaxBytes when zero or negative.
func resolveMaxBytes(maxBytes int) int {
	if maxBytes <= 0 {
		return defaultMaxBytes
	}

	return maxBytes
}

// resolveProjectSlug returns the effective project slug for the given args.
func resolveProjectSlug(args TranscriptArgs) (string, error) {
	if args.ProjectSlug != "" {
		return args.ProjectSlug, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("transcript: resolving working directory: %w", err)
	}

	return ProjectSlugFromPath(cwd), nil
}

func resolveStateDir(args TranscriptArgs) (string, error) {
	if args.StateDir != "" {
		return args.StateDir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("transcript: resolving home dir: %w", err)
	}

	return learnmarker.StateDirFromHome(home, os.Getenv), nil
}

func resolveTranscriptState(args TranscriptArgs) (transcriptState, error) {
	transcriptDir := args.TranscriptDir

	err := applyTranscriptDirDefault(&transcriptDir, args.ProjectSlug, os.Getwd)
	if err != nil {
		return transcriptState{}, err
	}

	stateDir, err := resolveStateDir(args)
	if err != nil {
		return transcriptState{}, err
	}

	slug, err := resolveProjectSlug(args)
	if err != nil {
		return transcriptState{}, err
	}

	sources := []string{"claude", "opencode"}
	markers := make(map[string]harnessMarker, len(sources))

	for _, src := range sources {
		mPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, src)

		mTime, mFound, mErr := learnmarker.Read(learnmarker.OSFS{}, mPath)
		if mErr != nil {
			return transcriptState{}, fmt.Errorf("transcript: reading marker: %w", mErr)
		}

		markers[src] = harnessMarker{path: mPath, mtime: mTime, found: mFound}
	}

	return transcriptState{dir: transcriptDir, markers: markers}, nil
}

// runTranscript reads session transcripts in [from, to] (inclusive) and emits stripped content.
func runTranscript(
	_ context.Context,
	args TranscriptArgs,
	finder transcript.Finder,
	reader transcript.Reader,
	stdout io.Writer,
) error {
	state, err := resolveTranscriptState(args)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	entries, findErr := finder.Find(state.dir)
	if findErr != nil {
		return fmt.Errorf("transcript: finding sessions: %w", findErr)
	}

	filtered := filterBySourceMarkers(entries, state.markers, now)
	slices.Reverse(filtered)

	lastIncluded, hadEntries, emitErr := emitTranscripts(
		reader, filtered, resolveMaxBytes(args.MaxBytes), stdout,
	)
	if emitErr != nil {
		return emitErr
	}

	if args.Mark {
		for src, marker := range state.markers {
			markErr := advanceAndReportMarker(
				marker.path,
				fromForSource(src, marker, now),
				lastIncluded[src],
				hadEntries[src],
				now,
				stdout,
			)
			if markErr != nil {
				return markErr
			}
		}
	}

	return nil
}
