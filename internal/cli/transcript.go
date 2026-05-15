package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strings"
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
	From          string `targ:"flag,name=from,desc=start date (YYYY-MM-DD or RFC3339 or 'all' for epoch)"`
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
	errTranscriptFirstRun = errors.New(
		"transcript: no progress marker (pass --from <YYYY-MM-DD> or --from all)",
	)
	errTranscriptInvalidDate = errors.New(
		"transcript: invalid date: expected YYYY-MM-DD, RFC3339, or 'all'",
	)
)

// emitResult bundles per-source outputs from emitTranscripts so callers can
// both advance markers (lastIncluded + hadEntries) and emit continuation
// warnings (firstUnincluded — Mtime of the earliest entry the byte cap
// excluded, per source, zero time if everything fit).
type emitResult struct {
	lastIncluded    map[string]time.Time
	hadEntries      map[string]bool
	firstUnincluded map[string]time.Time
}

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

// advanceMarkers writes each per-source marker forward and emits the status
// line. Sources whose marker is missing AND have no entries in this run AND
// no explicit --from were guarded out earlier — for those, the marker still
// advances to `now` so the next run starts cleanly.
func advanceMarkers(
	markers map[string]harnessMarker,
	result emitResult,
	explicitFrom, now time.Time,
	stdout io.Writer,
) error {
	for src, marker := range markers {
		markErr := advanceAndReportMarker(
			marker.path,
			fromForMarkerReport(marker, explicitFrom, now),
			result.lastIncluded[src],
			result.hadEntries[src],
			now,
			stdout,
		)
		if markErr != nil {
			return markErr
		}
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

// checkFirstRun hard-fails when --mark is set, --from is empty, and any
// source has both a missing marker and at least one session entry. The error
// names every offending source plus its earliest entry's Mtime so the caller
// can pass --from <date> or --from all on the next invocation.
func checkFirstRun(
	args TranscriptArgs,
	markers map[string]harnessMarker,
	entries []transcript.FileEntry,
	explicitFrom time.Time,
) error {
	if !args.Mark || !explicitFrom.IsZero() {
		return nil
	}

	earliest := earliestBySource(entries)

	var offenders []string

	for src, marker := range markers {
		if marker.found {
			continue
		}

		earliestForSrc, hasEntries := earliest[src]
		if !hasEntries {
			continue
		}

		offenders = append(offenders, fmt.Sprintf("%s (earliest session: %s)",
			src, earliestForSrc.UTC().Format(dateFormat)))
	}

	if len(offenders) == 0 {
		return nil
	}

	slices.Sort(offenders)

	return fmt.Errorf("%w: %s", errTranscriptFirstRun, strings.Join(offenders, ", "))
}

// earliestBySource returns the earliest Mtime per source across entries.
func earliestBySource(entries []transcript.FileEntry) map[string]time.Time {
	earliest := make(map[string]time.Time)

	for _, entry := range entries {
		current, seen := earliest[entry.Source]
		if !seen || entry.Mtime.Before(current) {
			earliest[entry.Source] = entry.Mtime
		}
	}

	return earliest
}

// emitContinuationWarnings prints one line per source whose scan ended at
// the byte cap, telling the user when older sessions remain.
func emitContinuationWarnings(firstUnincluded map[string]time.Time, stdout io.Writer) {
	sources := make([]string, 0, len(firstUnincluded))
	for src := range firstUnincluded {
		sources = append(sources, src)
	}

	slices.Sort(sources)

	for _, src := range sources {
		_, _ = fmt.Fprintf(
			stdout,
			"[engram transcript: byte cap hit; %s sessions from %s onward not yet scanned; run again to continue]\n",
			src, firstUnincluded[src].UTC().Format(time.RFC3339Nano),
		)
	}
}

// emitTranscripts emits content chronologically (oldest first), stopping when
// the next entry's content would push total bytes over maxBytes. The first
// entry is always included even if it alone exceeds the cap — a single
// oversized entry must not stall marker progress forever. The returned
// firstUnincluded map records, per source, the Mtime of the earliest entry
// the byte cap excluded; callers use it to warn the user that more remains.
func emitTranscripts(
	reader transcript.Reader,
	entries []transcript.FileEntry,
	maxBytes int,
	stdout io.Writer,
) (emitResult, error) {
	result := emitResult{
		lastIncluded:    make(map[string]time.Time),
		hadEntries:      make(map[string]bool),
		firstUnincluded: make(map[string]time.Time),
	}

	var total int

	for index, entry := range entries {
		content, _, readErr := reader.Read(entry.Path, math.MaxInt32)
		if readErr != nil {
			return emitResult{}, fmt.Errorf("transcript: reading %s: %w", entry.Path, readErr)
		}

		// First entry is always included (progress guarantee). Subsequent entries
		// stop the scan when their content would push total over maxBytes. Record
		// the first such excluded entry per source so the caller can warn.
		if index > 0 && total+len(content) > maxBytes {
			recordUnincluded(result.firstUnincluded, entries[index:])

			break
		}

		_, writeErr := io.WriteString(stdout, content)
		if writeErr != nil {
			return emitResult{}, fmt.Errorf("transcript: writing output: %w", writeErr)
		}

		total += len(content)
		result.lastIncluded[entry.Source] = entry.Mtime
		result.hadEntries[entry.Source] = true
	}

	return result, nil
}

// filterBySourceMarkers returns entries whose Mtime falls within each source's
// [from, to] range. When explicitFrom is non-zero it overrides every source's
// marker-derived from; otherwise from is derived from each source's marker.
// Entries from sources without a marker are dropped when explicitFrom is zero
// — the caller is expected to gate this case via the first-run check before
// reaching the filter.
func filterBySourceMarkers(
	entries []transcript.FileEntry,
	markers map[string]harnessMarker,
	explicitFrom, now time.Time,
) []transcript.FileEntry {
	filtered := make([]transcript.FileEntry, 0, len(entries))

	for _, entry := range entries {
		marker, ok := markers[entry.Source]
		if !ok {
			continue
		}

		from := fromForSource(marker, explicitFrom, now)
		if from.IsZero() {
			continue
		}

		if !entry.Mtime.Before(from) && !entry.Mtime.After(now) {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

// fromForMarkerReport mirrors fromForSource but falls back to `now` (the
// scan-end) when no marker exists and no explicit --from was given. This
// only happens on sources with no entries — the first-run check would have
// failed earlier otherwise.
func fromForMarkerReport(marker harnessMarker, explicitFrom, now time.Time) time.Time {
	from := fromForSource(marker, explicitFrom, now)
	if from.IsZero() {
		return now
	}

	return from
}

// fromForSource returns the effective from time for a given source's marker.
// explicitFrom (non-zero) wins; otherwise the marker's mtime is used; if the
// marker is missing and explicitFrom is zero, returns the zero time so the
// caller can skip the source.
func fromForSource(marker harnessMarker, explicitFrom, _ time.Time) time.Time {
	if !explicitFrom.IsZero() {
		return explicitFrom
	}

	if marker.found {
		return marker.mtime
	}

	return time.Time{}
}

// parseDate parses a date string, accepting RFC3339, YYYY-MM-DD, or the
// sentinel "all" (which resolves to the Unix epoch, i.e. scan from the
// beginning of history).
func parseDate(s string) (time.Time, error) {
	if s == "all" {
		return time.Unix(0, 0).UTC(), nil
	}

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

// recordUnincluded captures the earliest excluded entry per source (entries
// are oldest-first, so the first occurrence of each source is the earliest).
func recordUnincluded(into map[string]time.Time, excluded []transcript.FileEntry) {
	for _, entry := range excluded {
		if _, seen := into[entry.Source]; seen {
			continue
		}

		into[entry.Source] = entry.Mtime
	}
}

// resolveExplicitFrom parses the --from flag value, returning the zero time
// when --from was not set. The "all" sentinel is recognized by parseDate.
func resolveExplicitFrom(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}

	parsed, err := parseDate(raw)
	if err != nil {
		return time.Time{}, err
	}

	return parsed, nil
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

	explicitFrom, fromErr := resolveExplicitFrom(args.From)
	if fromErr != nil {
		return fromErr
	}

	firstRunErr := checkFirstRun(args, state.markers, entries, explicitFrom)
	if firstRunErr != nil {
		return firstRunErr
	}

	filtered := filterBySourceMarkers(entries, state.markers, explicitFrom, now)
	slices.Reverse(filtered)

	result, emitErr := emitTranscripts(
		reader, filtered, resolveMaxBytes(args.MaxBytes), stdout,
	)
	if emitErr != nil {
		return emitErr
	}

	if args.Mark {
		markErr := advanceMarkers(state.markers, result, explicitFrom, now, stdout)
		if markErr != nil {
			return markErr
		}
	}

	emitContinuationWarnings(result.firstUnincluded, stdout)

	return nil
}
