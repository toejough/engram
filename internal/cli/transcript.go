package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	Segments      bool   `targ:"flag,name=segments,desc=emit one segment line per user turn"`
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
	defaultLookback   = 24 * time.Hour
	defaultMaxBytes   = 200_000
	segmentLineFormat = time.RFC3339
)

// unexported variables.
var (
	errSegmentsNotSupported = errors.New("transcript: --segments not supported by this reader")
	errTranscriptFirstRun   = errors.New(
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
	hadEntries, pending bool,
	now time.Time,
	stdout io.Writer,
) error {
	// Invariant: never advance the marker past the earliest row not actually
	// read. When the source has pending (budget-skipped) entries, hold the
	// marker at lastIncluded — with seeding this equals the marker when nothing
	// was read, so budget-starved sources stay put instead of jumping to now.
	effectiveEnd := now

	switch {
	case pending:
		effectiveEnd = lastIncluded
	case hadEntries && lastIncluded.Before(now):
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
		_, pending := result.firstUnincluded[src]

		markErr := advanceAndReportMarker(
			marker.path,
			fromForMarkerReport(marker, explicitFrom, now),
			result.lastIncluded[src],
			result.hadEntries[src],
			pending,
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

// dispatchEmit routes to emitSegments or emitTranscripts based on args.Segments.
func dispatchEmit(
	args TranscriptArgs,
	reader transcript.Reader,
	filtered []transcript.FileEntry,
	maxBytes int,
	seed map[string]time.Time,
	stdout io.Writer,
) (emitResult, error) {
	if args.Segments {
		segReader, ok := reader.(transcript.SegmentsReader)
		if !ok {
			return emitResult{}, errSegmentsNotSupported
		}

		return emitSegments(segReader, filtered, maxBytes, seed, stdout)
	}

	return emitTranscripts(reader, filtered, maxBytes, seed, stdout)
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
			src,
			firstUnincluded[src].UTC().Format(time.RFC3339Nano),
		)
	}
}

// emitOneEntry reads one transcript entry through the reader, writes its
// content to stdout, and updates result.lastIncluded / result.hadEntries
// in place. Returns bytesUsed, whether the read was partial, and any error.
// The marker rule: LastTimestamp wins when non-zero; a full (non-partial)
// read additionally advances to the entry's file Mtime to cover trailing
// untimestamped rows.
func emitOneEntry(
	reader transcript.Reader,
	entry transcript.FileEntry,
	remaining int,
	result *emitResult,
	stdout io.Writer,
) (int, bool, error) {
	if remaining < 1 {
		remaining = 1
	}

	fromTime := result.lastIncluded[entry.Source]

	readResult, readErr := reader.ReadFrom(entry.Path, fromTime, remaining)
	if readErr != nil {
		return 0, false, fmt.Errorf("transcript: reading %s: %w", entry.Path, readErr)
	}

	_, writeErr := io.WriteString(stdout, readResult.Content)
	if writeErr != nil {
		return 0, false, fmt.Errorf("transcript: writing output: %w", writeErr)
	}

	if !readResult.LastTimestamp.IsZero() {
		result.lastIncluded[entry.Source] = readResult.LastTimestamp
	}

	if !readResult.Partial && !entry.Mtime.IsZero() {
		result.lastIncluded[entry.Source] = entry.Mtime
	}

	result.hadEntries[entry.Source] = true

	return readResult.BytesUsed, readResult.Partial, nil
}

// emitOneSegmentEntry reads one entry's user-turn segments through the reader,
// writes each as a line to stdout, and updates result.lastIncluded /
// result.hadEntries in place. It mirrors emitOneEntry's marker rule: each
// segment's timestamp advances lastIncluded, and a full (non-partial) read
// additionally advances to the entry's file Mtime to cover trailing rows.
// Returns the bytes written, whether the scan was budget-truncated, and any
// error. The caller halts the outer loop on a truncated (partial) read.
func emitOneSegmentEntry(
	reader transcript.SegmentsReader,
	entry transcript.FileEntry,
	remaining int,
	result *emitResult,
	stdout io.Writer,
) (int, bool, error) {
	fromTime := result.lastIncluded[entry.Source]

	segResult, readErr := reader.SegmentsFrom(entry.Path, fromTime, remaining)
	if readErr != nil {
		return 0, false, fmt.Errorf("transcript: reading %s: %w", entry.Path, readErr)
	}

	bytesWritten := 0

	for _, seg := range segResult.Segments {
		line := seg.Timestamp.UTC().Format(segmentLineFormat) + "\t" + seg.Preview + "\n"

		_, writeErr := io.WriteString(stdout, line)
		if writeErr != nil {
			return 0, false, fmt.Errorf("transcript: writing output: %w", writeErr)
		}

		bytesWritten += len(line)

		if !seg.Timestamp.IsZero() {
			result.lastIncluded[entry.Source] = seg.Timestamp
		}
	}

	result.hadEntries[entry.Source] = true

	if !segResult.Partial && !entry.Mtime.IsZero() {
		result.lastIncluded[entry.Source] = entry.Mtime
	}

	return bytesWritten, segResult.Partial, nil
}

// emitSegments emits one line per genuine user turn across all entries, using
// the same byte-budget and per-source marker logic as emitTranscripts. Output
// format per line: <RFC3339-timestamp>\tUSER: <first 100 chars of text>.
// The per-source bookmark (lastIncluded) is updated exactly as in emitTranscripts
// so --mark and continuation warnings work identically whether --segments is set
// or not.
func emitSegments(
	reader transcript.SegmentsReader,
	entries []transcript.FileEntry,
	maxBytes int,
	seed map[string]time.Time,
	stdout io.Writer,
) (emitResult, error) {
	result := newSeededEmitResult(seed)

	total := 0

	for index, entry := range entries {
		remaining := maxBytes - total
		if remaining <= 0 && index > 0 {
			recordUnincluded(result.firstUnincluded, entries[index:])

			break
		}

		if remaining < 1 {
			remaining = 1
		}

		bytesWritten, partial, emitErr := emitOneSegmentEntry(reader, entry, remaining, &result, stdout)
		if emitErr != nil {
			return emitResult{}, emitErr
		}

		total += bytesWritten

		// Mirror emitTranscripts: when the segments scan was truncated by the
		// byte budget, hold the marker at the last fully-included segment (set
		// inside emitOneSegmentEntry) rather than advancing past this entry,
		// then halt — advancing would skip its unread tail and (for same-source
		// files, oldest-first) step the marker onto a newer file's Mtime,
		// dropping rows (M2 invariant).
		if partial {
			recordUnincluded(result.firstUnincluded, entries[index:])

			break
		}
	}

	return result, nil
}

// emitTranscripts emits content chronologically (oldest first), advancing
// the per-source marker incrementally. For each entry the reader takes
// the per-source marker as fromTime and a remaining byte budget; the
// reader returns ReadResult{Content, BytesUsed, LastTimestamp, Partial}.
// When Partial is true, the scan halts after that entry; the marker
// advances to LastTimestamp so the next run resumes mid-session. When
// Partial is false, the marker advances to the entry's file Mtime
// (covering trailing rows without parseable timestamps).
//
// The first entry's read is granted at least 1 byte of budget so a
// pathological session larger than maxBytes still makes progress.
//
// firstUnincluded records, per source, the Mtime of the next entry the
// byte cap excluded (or the partially-emitted entry's own Mtime), so
// callers can warn the user that more remains.
func emitTranscripts(
	reader transcript.Reader,
	entries []transcript.FileEntry,
	maxBytes int,
	seed map[string]time.Time,
	stdout io.Writer,
) (emitResult, error) {
	result := newSeededEmitResult(seed)

	total := 0

	for index, entry := range entries {
		remaining := maxBytes - total
		if remaining <= 0 && index > 0 {
			recordUnincluded(result.firstUnincluded, entries[index:])

			break
		}

		bytesUsed, partial, emitErr := emitOneEntry(reader, entry, remaining, &result, stdout)
		if emitErr != nil {
			return emitResult{}, emitErr
		}

		total += bytesUsed

		if partial {
			recordUnincluded(result.firstUnincluded, entries[index:])

			break
		}
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

		if entry.Mtime.After(from) && !entry.Mtime.After(now) {
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

// newSeededEmitResult builds a fresh emitResult with lastIncluded pre-seeded
// from each source's effective-from. Seeding makes the first ReadFrom/
// SegmentsFrom of each source resume from its marker (so an oversized session
// resumes mid-file instead of restarting), and makes a budget-starved source's
// lastIncluded equal its marker so the marker never advances past unread rows.
// Zero-time seeds are skipped (those sources are guarded by the first-run check).
func newSeededEmitResult(seed map[string]time.Time) emitResult {
	result := emitResult{
		lastIncluded:    make(map[string]time.Time, len(seed)),
		hadEntries:      make(map[string]bool),
		firstUnincluded: make(map[string]time.Time),
	}

	for src, from := range seed {
		if from.IsZero() {
			continue
		}

		result.lastIncluded[src] = from
	}

	return result
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

	seed := seedFromMarkers(state.markers, explicitFrom, now)

	result, emitErr := dispatchEmit(args, reader, filtered, resolveMaxBytes(args.MaxBytes), seed, stdout)
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

// seedFromMarkers returns each source's effective-from (the same value
// filterBySourceMarkers uses) so the emit loop can seed lastIncluded per
// source. Sources whose from is zero (missing marker, no explicit --from)
// are omitted — they are guarded by the first-run check and have no resume
// point. The result threads the per-source marker into the reader as the
// resume fromTime AND keeps a budget-starved source's lastIncluded equal to
// its marker so the marker never advances past unread rows.
func seedFromMarkers(
	markers map[string]harnessMarker,
	explicitFrom, now time.Time,
) map[string]time.Time {
	seed := make(map[string]time.Time, len(markers))

	for src, marker := range markers {
		from := fromForSource(marker, explicitFrom, now)
		if from.IsZero() {
			continue
		}

		seed[src] = from
	}

	return seed
}
