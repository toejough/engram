package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/learnmarker"
	"github.com/toejough/engram/internal/transcript"
)

func TestAdvanceAndReportMarker_HoldsAtMarkerWhenPending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Bug 1: a source with pending (budget-skipped) entries must hold its
	// marker at lastIncluded (which seeding sets to the marker when nothing
	// was read this run) rather than jumping forward to now — otherwise its
	// pending sessions are skipped forever.
	tmp := t.TempDir()
	markerPath := filepath.Join(tmp, "marker.txt")

	fromTime := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	marker := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 30, 0, 0, time.UTC)

	var stdout bytes.Buffer

	// pending=true, hadEntries=false, lastIncluded=marker (the seeded value).
	err := cli.AdvanceAndReportMarkerForTest(markerPath, fromTime, marker, false, true, now, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	markerBytes, readErr := os.ReadFile(markerPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	parsed, parseErr := time.Parse(time.RFC3339Nano, string(markerBytes))
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(parsed.Equal(marker)).To(BeTrue())
	g.Expect(parsed.Equal(now)).To(BeFalse())
}

func TestAdvanceAndReportMarker_StatusLineContainsBothFromAndEffectiveEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	markerPath := filepath.Join(tmp, "marker.txt")

	fromTime := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	lastIncluded := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 30, 0, 0, time.UTC)

	var stdout bytes.Buffer

	err := cli.AdvanceAndReportMarkerForTest(markerPath, fromTime, lastIncluded, true, false, now, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	statusLine := stdout.String()
	expectedFrom := fromTime.UTC().Format(time.RFC3339Nano)
	expectedEnd := lastIncluded.UTC().Format(time.RFC3339Nano)

	g.Expect(statusLine).To(ContainSubstring(expectedFrom))
	g.Expect(statusLine).To(ContainSubstring(expectedEnd))
	g.Expect(statusLine).To(ContainSubstring("[engram transcript: scanned ["))
	g.Expect(statusLine).To(ContainSubstring("]; marker advanced to "))
}

func TestAdvanceAndReportMarker_UsesLastIncludedWhenCapHit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	markerPath := filepath.Join(tmp, "marker.txt")

	fromTime := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	lastIncluded := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 30, 0, 0, time.UTC)

	var stdout bytes.Buffer

	err := cli.AdvanceAndReportMarkerForTest(markerPath, fromTime, lastIncluded, true, false, now, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	markerBytes, readErr := os.ReadFile(markerPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	parsed, parseErr := time.Parse(time.RFC3339Nano, string(markerBytes))
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(parsed.Equal(lastIncluded)).To(BeTrue())
}

func TestAdvanceAndReportMarker_UsesNowWhenEverythingFit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	markerPath := filepath.Join(tmp, "marker.txt")

	fromTime := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 30, 0, 0, time.UTC)

	var stdout bytes.Buffer

	// When lastIncluded == now (not Before now), use now
	err := cli.AdvanceAndReportMarkerForTest(markerPath, fromTime, now, true, false, now, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	markerBytes, readErr := os.ReadFile(markerPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	parsed, parseErr := time.Parse(time.RFC3339Nano, string(markerBytes))
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(parsed.Equal(now)).To(BeTrue())
}

func TestAdvanceAndReportMarker_UsesNowWhenNoEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	markerPath := filepath.Join(tmp, "marker.txt")

	fromTime := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 30, 0, 0, time.UTC)

	var stdout bytes.Buffer

	err := cli.AdvanceAndReportMarkerForTest(markerPath, fromTime, time.Time{}, false, false, now, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	markerBytes, readErr := os.ReadFile(markerPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	parsed, parseErr := time.Parse(time.RFC3339Nano, string(markerBytes))
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(parsed.Equal(now)).To(BeTrue())
}

func TestApplyTranscriptDirDefault(t *testing.T) {
	t.Parallel()

	t.Run("uses provided dir when non-empty", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{entries: []transcript.FileEntry{}}
		reader := &fakeReader{contents: map[string]string{}}

		var stdout bytes.Buffer

		err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
			TranscriptDir: t.TempDir(),
		}, finder, reader, &stdout)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stdout.String()).To(BeEmpty())
	})

	t.Run("derives dir from cwd when transcript-dir empty and no slug", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{entries: []transcript.FileEntry{}}
		reader := &fakeReader{contents: map[string]string{}}

		var stdout bytes.Buffer

		err := cli.RunTranscriptForTest(
			context.Background(),
			cli.TranscriptArgs{},
			finder,
			reader,
			&stdout,
		)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stdout.String()).To(BeEmpty())
	})

	t.Run("uses project-slug when transcript-dir empty", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		finder := &fakeFinder{entries: []transcript.FileEntry{}}
		reader := &fakeReader{contents: map[string]string{}}

		var stdout bytes.Buffer

		err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
			ProjectSlug: "-test-project",
		}, finder, reader, &stdout)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stdout.String()).To(BeEmpty())
	})
}

func TestEmitSegments_EmitsOneLinePerUserTurn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts1 := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 5, 1, 11, 0, 0, 0, time.UTC)

	reader := &fakeSegmentsReader{segments: map[string][]transcript.Segment{
		"/a.jsonl": {
			{Timestamp: ts1, Preview: "USER: first ask"},
			{Timestamp: ts2, Preview: "USER: second ask"},
		},
	}}

	entry := transcript.FileEntry{Path: "/a.jsonl", Mtime: ts2.Add(time.Hour), Source: "claude"}

	var buf bytes.Buffer

	lastIncluded, hadEntries, _, err := cli.EmitSegmentsForTest(
		reader,
		[]transcript.FileEntry{entry},
		1<<20,
		nil,
		&buf,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	g.Expect(lines).To(HaveLen(2))
	g.Expect(lines[0]).To(ContainSubstring(ts1.UTC().Format(time.RFC3339)))
	g.Expect(lines[0]).To(ContainSubstring("USER: first ask"))
	g.Expect(lines[1]).To(ContainSubstring(ts2.UTC().Format(time.RFC3339)))
	g.Expect(lines[1]).To(ContainSubstring("USER: second ask"))
	g.Expect(hadEntries["claude"]).To(BeTrue())
	g.Expect(lastIncluded["claude"]).NotTo(BeZero())
}

func TestEmitSegments_ResumesReaderFromSeed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Bug 2 (segments path): the per-source seed (effective-from) must be
	// threaded into SegmentsFrom as the resume point on the FIRST read of
	// that source, mirroring emitTranscripts.
	markerTime := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	may20 := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

	reader := &fromTimeRecordingSegmentsReader{segments: map[string][]transcript.Segment{
		"/session": {{Timestamp: may20, Preview: "USER: resumed"}},
	}}
	entry := transcript.FileEntry{Path: "/session", Mtime: may20.Add(time.Hour), Source: "claude"}
	seed := map[string]time.Time{"claude": markerTime}

	var buf bytes.Buffer

	_, _, _, err := cli.EmitSegmentsForTest(reader, []transcript.FileEntry{entry}, 1<<20, seed, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(reader.fromTimes).NotTo(BeEmpty())
	g.Expect(reader.fromTimes[0].Equal(markerTime)).To(BeTrue())
}

func TestEmitTranscripts_NoEntriesReturnsZeroAndFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reader := &fakeReader{contents: map[string]string{}}

	var buf bytes.Buffer

	lastIncluded, hadEntries, _, err := cli.EmitTranscriptsForTest(reader, nil, 1000, nil, &buf)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(hadEntries).To(BeEmpty())
	g.Expect(lastIncluded).To(BeEmpty())
	g.Expect(buf.Len()).To(Equal(0))
}

func TestEmitTranscripts_OversizedFirstEntrySignalsPartialAndContinuation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Single entry larger than cap — under the new architecture the reader
	// is responsible for emitting at least one row (progress guarantee) and
	// returning Partial=true so the next run resumes mid-session. The fake
	// reader here models the "decline" outcome (content > budget); the real
	// readers emit at least one row of the file. The behavioral contract
	// tested at this layer is: hadEntries is set, firstUnincluded records
	// the partial entry's Mtime so the continuation warning fires.
	mkContent := func(prefix string) string { return prefix + strings.Repeat("x", 999) }
	reader := &fakeReader{contents: map[string]string{
		"/a": mkContent("A"),
	}}
	may1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	entries := []transcript.FileEntry{{Path: "/a", Mtime: may1, Source: "claude"}}

	var buf bytes.Buffer

	_, hadEntries, firstUnincluded, err := cli.EmitTranscriptsForTest(reader, entries, 100, nil, &buf)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(hadEntries["claude"]).To(BeTrue())
	g.Expect(firstUnincluded["claude"].Equal(may1)).To(BeTrue())
}

func TestEmitTranscripts_ReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entries := []transcript.FileEntry{{Path: "fake.jsonl"}}

	var stdout bytes.Buffer

	err := cli.ExportEmitTranscripts(&failReader{}, entries, &stdout)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("transcript: reading"))
}

func TestEmitTranscripts_ScansForwardAndStopsAtCap(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Three 100-byte entries (chronological); cap at 150 — first entry is always
	// included (progress guarantee), second would push total to 200 > 150 so the
	// scan stops. Effective end = first entry's Mtime.
	mkContent := func(prefix string) string { return prefix + strings.Repeat("x", 99) }
	reader := &fakeReader{contents: map[string]string{
		"/a": mkContent("A"),
		"/b": mkContent("B"),
		"/c": mkContent("C"),
	}}
	may1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	entries := []transcript.FileEntry{
		{Path: "/a", Mtime: may1, Source: "claude"},
		{Path: "/b", Mtime: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC), Source: "claude"},
		{Path: "/c", Mtime: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC), Source: "claude"},
	}

	var buf bytes.Buffer

	lastIncluded, hadEntries, firstUnincluded, err := cli.EmitTranscriptsForTest(
		reader,
		entries,
		150,
		nil,
		&buf,
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(hadEntries["claude"]).To(BeTrue())
	g.Expect(lastIncluded["claude"].Equal(may1)).To(BeTrue())
	// First excluded entry was /b (2026-05-02), so callers can warn the user
	// that older sessions remain through that timestamp.
	g.Expect(firstUnincluded["claude"].Equal(time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC))).
		To(BeTrue())

	out := buf.String()
	g.Expect(out).To(ContainSubstring("A"))
	g.Expect(out).NotTo(ContainSubstring("B"))
	g.Expect(out).NotTo(ContainSubstring("C"))
}

func TestParseDate(t *testing.T) {
	t.Parallel()

	t.Run("accepts YYYY-MM-DD", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		stderr := executeForTest(t, []string{
			"engram", "transcript",
			"--from", "2026-05-10",
			"--to", "2026-05-10",
			"--transcript-dir", t.TempDir(),
		})

		// No parse error expected for valid dates.
		g.Expect(stderr).NotTo(ContainSubstring("invalid date"))
	})

	t.Run("accepts RFC3339", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		stderr := executeForTest(t, []string{
			"engram", "transcript",
			"--from", "2026-05-10T00:00:00Z",
			"--to", "2026-05-10T23:59:59Z",
			"--transcript-dir", t.TempDir(),
		})
		g.Expect(stderr).NotTo(ContainSubstring("invalid date"))
	})
}

func TestParseDate_AcceptsAllSentinel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// "all" must round-trip through ResolveTimeWindow as an early date so
	// downstream filters see "scan everything".
	from, _, err := cli.ResolveTimeWindow(cli.TimeWindowInputs{
		From: "all",
		Now:  time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(from.Year()).To(BeNumerically("<=", 1970))
}

func TestResolveMaxBytes_ReturnsDefaultWhenZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := cli.ResolveMaxBytesForTest(0)

	g.Expect(result).To(Equal(200_000))
}

func TestResolveMaxBytes_ReturnsValueWhenPositive(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := cli.ResolveMaxBytesForTest(500)

	g.Expect(result).To(Equal(500))
}

func TestResolveProjectSlug_DerivesCwdWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	slug, err := cli.ResolveProjectSlugForTest(cli.TranscriptArgs{})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(slug).NotTo(BeEmpty())
}

func TestResolveProjectSlug_ReturnsSlugWhenProvided(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	slug, err := cli.ResolveProjectSlugForTest(cli.TranscriptArgs{ProjectSlug: "my-project"})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(slug).To(Equal("my-project"))
}

func TestResolveStateDir_DefaultsToXDGWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, err := cli.ResolveStateDirForTest(cli.TranscriptArgs{})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(dir).NotTo(BeEmpty())
}

func TestResolveStateDir_ReturnsDirWhenProvided(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir, err := cli.ResolveStateDirForTest(cli.TranscriptArgs{StateDir: "/custom/state"})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(dir).To(Equal("/custom/state"))
}

func TestResolveTimeWindow_DateOnlyToExtendedToEOD(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	_, toTime, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{To: "2026-05-11", MarkerFound: false, Now: now},
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Date-only "2026-05-11" should be extended to end of day.
	expected := time.Date(2026, 5, 11, 23, 59, 59, 999999999, time.UTC)
	g.Expect(toTime.Equal(expected)).To(BeTrue())
}

func TestResolveTimeWindow_ExplicitFromOverridesMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	explicit := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	markerTime := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	from, _, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{From: "2026-05-10", Marker: markerTime, MarkerFound: true, Now: now},
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(from.Equal(explicit)).To(BeTrue())
}

func TestResolveTimeWindow_FallsBackTo24hWhenNoMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	from, toTime, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{From: "", To: "", MarkerFound: false, Now: now},
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(from.Equal(now.Add(-24 * time.Hour))).To(BeTrue())
	g.Expect(toTime.Equal(now)).To(BeTrue())
}

func TestResolveTimeWindow_InvalidFromReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	_, _, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{From: "not-a-date", MarkerFound: false, Now: now},
	)

	g.Expect(err).To(HaveOccurred())
}

func TestResolveTimeWindow_InvalidToReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	_, _, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{To: "not-a-date", MarkerFound: false, Now: now},
	)

	g.Expect(err).To(HaveOccurred())
}

func TestResolveTimeWindow_RFC3339ToNoExtension(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	_, toTime, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{To: "2026-05-11T15:30:00Z", MarkerFound: false, Now: now},
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	expected := time.Date(2026, 5, 11, 15, 30, 0, 0, time.UTC)
	g.Expect(toTime.Equal(expected)).To(BeTrue())
}

func TestResolveTimeWindow_UsesMarkerWhenFromMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	markerTime := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	from, toTime, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{From: "", To: "", Marker: markerTime, MarkerFound: true, Now: now},
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(from.Equal(markerTime)).To(BeTrue())
	g.Expect(toTime.Equal(now)).To(BeTrue())
}

func TestRunTranscript_AcceptsEmptyFromAndToWhenMarkerExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	markerTime := time.Now().Add(-2 * time.Hour).UTC()
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	finder := &fakeFinder{entries: []transcript.FileEntry{}}
	reader := &fakeReader{contents: map[string]string{}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunTranscript_BudgetStarvedSourceMarkerStaysPut(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Bug 1: when one source's chronologically-earlier session consumes the
	// whole byte budget, the OTHER source processes no entries this run. Its
	// marker must NOT jump to now (which would skip its pending sessions
	// forever) — it must stay at its existing marker.
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"

	claudeMarkerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
	opencodeMarkerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "opencode")

	g.Expect(os.MkdirAll(filepath.Dir(claudeMarkerPath), 0o755)).To(Succeed())

	claudeMarker := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	opencodeMarker := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)

	g.Expect(os.WriteFile(claudeMarkerPath, []byte(claudeMarker.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())
	g.Expect(os.WriteFile(opencodeMarkerPath, []byte(opencodeMarker.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	// claude's session is chronologically earlier (May 15) than opencode's
	// (May 20), so after reverse-to-oldest-first claude is emitted first and
	// its content (exactly MaxBytes long) consumes the entire budget. opencode
	// then sees remaining<=0 and is recorded unread.
	claudeEntryMtime := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	opencodeEntryMtime := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

	const maxBytes = 50

	claudeContent := strings.Repeat("c", maxBytes)

	// Finder order is newest-first (production sorts this way); runTranscript
	// reverses the filtered slice to oldest-first before emitting.
	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "/opencode", Mtime: opencodeEntryMtime, Source: "opencode"},
		{Path: "/claude", Mtime: claudeEntryMtime, Source: "claude"},
	}}
	reader := &fakeReader{contents: map[string]string{
		"/claude":   claudeContent,
		"/opencode": "unread opencode content",
	}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          true,
		MaxBytes:      maxBytes,
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// opencode processed nothing this run; its marker must stay put.
	opencodeBytes, readErr := os.ReadFile(opencodeMarkerPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	parsedOpencode, parseErr := time.Parse(time.RFC3339Nano, string(opencodeBytes))
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(parsedOpencode.Equal(opencodeMarker)).To(BeTrue())

	// claude was fully read; its marker advances to the entry's Mtime.
	claudeBytes, claudeReadErr := os.ReadFile(claudeMarkerPath)
	g.Expect(claudeReadErr).NotTo(HaveOccurred())

	if claudeReadErr != nil {
		return
	}

	parsedClaude, claudeParseErr := time.Parse(time.RFC3339Nano, string(claudeBytes))
	g.Expect(claudeParseErr).NotTo(HaveOccurred())

	if claudeParseErr != nil {
		return
	}

	g.Expect(parsedClaude.Equal(claudeEntryMtime)).To(BeTrue())
}

func TestRunTranscript_CallsFinderWithTranscriptDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "opencode")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	markerTime := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	dir := t.TempDir()
	may10 := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "opencode://ses1", Mtime: may10, Source: "opencode"},
	}}
	reader := &fakeReader{contents: map[string]string{
		"opencode://ses1": "hello",
	}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: dir,
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("hello"))
	g.Expect(finder.findCalledWith).To(ContainElement(dir))
}

func TestRunTranscript_DateFilterWithMocks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "opencode")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	// Marker set to May 9, 2026 — old entry from 2025 should be filtered out.
	markerTime := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	oldEntry := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "opencode://ses_old", Mtime: oldEntry, Source: "opencode"},
	}}
	reader := &fakeReader{contents: map[string]string{
		"opencode://ses_old": "should not appear",
	}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(BeEmpty())
}

func TestRunTranscript_EmptyFinderResult(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	finder := &fakeFinder{entries: []transcript.FileEntry{}}
	reader := &fakeReader{contents: map[string]string{}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		TranscriptDir: t.TempDir(),
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(BeEmpty())
}

func TestRunTranscript_FinderErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	finder := &fakeFinder{findErr: errors.New("finder broke")}
	reader := &fakeReader{contents: map[string]string{}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		TranscriptDir: t.TempDir(),
	}, finder, reader, &stdout)

	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("transcript: finding sessions"))
}

func TestRunTranscript_FirstRunHardFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"

	// No marker files written → both sources first-run. Finder reports one
	// claude entry from a real date so the error message can quote it.
	earliest := time.Date(2025, 8, 21, 0, 0, 0, 0, time.UTC)
	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "/old", Mtime: earliest, Source: "claude"},
	}}
	reader := &fakeReader{contents: map[string]string{}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          true,
	}, finder, reader, &stdout)

	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("no progress marker"))
	g.Expect(err.Error()).To(ContainSubstring("claude"))
	g.Expect(err.Error()).To(ContainSubstring("2025-08-21"))
}

func TestRunTranscript_FirstRunWithNoEntriesAdvancesMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// No markers, no entries → first-run check is a no-op; marker advances
	// to now so the next run starts cleanly. Matches existing
	// TestRunTranscript_MarkFlagAdvancesMarkerToNow behavior — guard against
	// regression as the first-run check is added.
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"

	finder := &fakeFinder{entries: []transcript.FileEntry{}}
	reader := &fakeReader{contents: map[string]string{}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          true,
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunTranscript_FromAllSkipsFirstRunGate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"

	earliest := time.Date(2025, 8, 21, 0, 0, 0, 0, time.UTC)
	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "/old", Mtime: earliest, Source: "claude"},
	}}
	reader := &fakeReader{contents: map[string]string{
		"/old": "ancient content",
	}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          true,
		From:          "all",
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("ancient content"))
	g.Expect(out).To(ContainSubstring("[engram transcript: scanned ["))
}

func TestRunTranscript_HappyPath(t *testing.T) {
	t.Parallel()

	t.Run("emits stripped content for in-range file", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		tmp := t.TempDir()
		stateDir := filepath.Join(tmp, ".local", "state", "engram")
		slug := "Users-joe-repos-test"
		markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
		g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

		markerTime := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
		g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
			To(Succeed())

		dir := t.TempDir()
		line := `{"type":"user","timestamp":"2026-05-10T12:00:00Z",` +
			`"message":{"content":"hello from transcript"}}`
		mtime := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
		writeTranscriptFixture(g, dir, "session.jsonl", line, mtime)

		finder, reader := cli.NewTranscriptDepsForTest("")

		var stdout bytes.Buffer

		err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
			ProjectSlug:   slug,
			StateDir:      stateDir,
			TranscriptDir: dir,
		}, finder, reader, &stdout)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stdout.String()).To(ContainSubstring("hello from transcript"))
	})

	t.Run("inclusive: file later same day as marker is included", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		tmp := t.TempDir()
		stateDir := filepath.Join(tmp, ".local", "state", "engram")
		slug := "Users-joe-repos-test"
		markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
		g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

		markerTime := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
		g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
			To(Succeed())

		dir := t.TempDir()
		line := `{"type":"user","timestamp":"2026-05-11T15:00:00Z",` +
			`"message":{"content":"afternoon message"}}`
		mtime := time.Date(2026, 5, 11, 15, 0, 0, 0, time.UTC)
		writeTranscriptFixture(g, dir, "afternoon.jsonl", line, mtime)

		finder, reader := cli.NewTranscriptDepsForTest("")

		var stdout bytes.Buffer

		err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
			ProjectSlug:   slug,
			StateDir:      stateDir,
			TranscriptDir: dir,
		}, finder, reader, &stdout)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stdout.String()).To(ContainSubstring("afternoon message"))
	})

	t.Run("RFC3339 marker accepted with assistant message", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		tmp := t.TempDir()
		stateDir := filepath.Join(tmp, ".local", "state", "engram")
		slug := "Users-joe-repos-test"
		markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
		g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

		markerTime := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
		g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
			To(Succeed())

		dir := t.TempDir()
		line := `{"type":"assistant","timestamp":"2026-05-10T10:00:00Z",` +
			`"message":{"content":"rfc3339 message"}}`
		mtime := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
		writeTranscriptFixture(g, dir, "rfc.jsonl", line, mtime)

		finder, reader := cli.NewTranscriptDepsForTest("")

		var stdout bytes.Buffer

		err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
			ProjectSlug:   slug,
			StateDir:      stateDir,
			TranscriptDir: dir,
		}, finder, reader, &stdout)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stdout.String()).To(ContainSubstring("rfc3339 message"))
	})
}

func TestRunTranscript_MarkEmitsStatusLine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	markerTime := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	may10 := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "opencode://ses1", Mtime: may10, Source: "claude"},
	}}
	reader := &fakeReader{contents: map[string]string{
		"opencode://ses1": "status line content",
	}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          true,
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("[engram transcript: scanned ["))
	g.Expect(out).To(ContainSubstring("]; marker advanced to "))
}

func TestRunTranscript_MarkFlagAdvancesMarkerToNow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"

	var stdout bytes.Buffer

	finder := &fakeFinder{entries: []transcript.FileEntry{}}
	reader := &fakeReader{contents: map[string]string{}}

	before := time.Now().UTC()
	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          true,
	}, finder, reader, &stdout)
	after := time.Now().UTC()

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	for _, src := range []string{"claude", "opencode"} {
		got, _ := os.ReadFile(learnmarker.MarkerPathWithSuffix(stateDir, slug, src))
		parsed, parseErr := time.Parse(time.RFC3339Nano, string(got))
		g.Expect(parseErr).NotTo(HaveOccurred())

		if parseErr != nil {
			return
		}

		g.Expect(parsed.After(before.Add(-time.Second)) && parsed.Before(after.Add(time.Second))).
			To(BeTrue())
	}
}

func TestRunTranscript_MarkWithEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "opencode")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	// Marker set before the entry's mtime so it passes the filter.
	markerTime := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	may10 := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "opencode://ses1", Mtime: may10, Source: "opencode"},
	}}
	reader := &fakeReader{contents: map[string]string{
		"opencode://ses1": "marked content",
	}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          true,
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("marked content"))
	g.Expect(stdout.String()).To(ContainSubstring("[engram transcript: scanned ["))
}

func TestRunTranscript_MarkerParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	g.Expect(os.WriteFile(markerPath, []byte("not-a-timestamp"), 0o644)).To(Succeed())

	finder := &fakeFinder{entries: []transcript.FileEntry{}}
	reader := &fakeReader{contents: map[string]string{}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
	}, finder, reader, &stdout)

	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("transcript: reading marker"))
}

func TestRunTranscript_NoMarkOmitsStatusLine(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	markerTime := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	may10 := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "opencode://ses1", Mtime: may10, Source: "claude"},
	}}
	reader := &fakeReader{contents: map[string]string{
		"opencode://ses1": "no mark content",
	}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          false,
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).NotTo(ContainSubstring("marker advanced to"))
}

func TestRunTranscript_PartialScanEmitsContinuationWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	markerTime := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	// Three entries, each ~100 bytes; with a 150 cap, only the first is
	// emitted and the byte cap fires before /b is included. The continuation
	// warning should name /b's mtime.
	mkContent := func(prefix string) string { return prefix + strings.Repeat("x", 99) }
	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "/a", Mtime: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Source: "claude"},
		{Path: "/b", Mtime: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC), Source: "claude"},
		{Path: "/c", Mtime: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC), Source: "claude"},
	}}
	reader := &fakeReader{contents: map[string]string{
		"/a": mkContent("A"),
		"/b": mkContent("B"),
		"/c": mkContent("C"),
	}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          true,
		MaxBytes:      150,
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("byte cap hit"))
	g.Expect(out).To(ContainSubstring("claude sessions from"))
	g.Expect(out).To(ContainSubstring("onward not yet scanned"))
	g.Expect(out).To(ContainSubstring("2026-05-02"))
}

func TestRunTranscript_PropagatesEmitError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	markerTime := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	may10 := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "opencode://ses1", Mtime: may10, Source: "claude"},
	}}
	reader := &fakeReader{contents: map[string]string{
		"opencode://ses1": "trigger emit",
	}}

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
	}, finder, reader, &failWriter{})

	g.Expect(err).To(HaveOccurred())
}

func TestRunTranscript_ReaderErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "opencode")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	markerTime := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	may10 := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "opencode://ses1", Mtime: may10, Source: "opencode"},
	}}
	reader := &failReader{}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
	}, finder, reader, &stdout)

	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("transcript: reading"))
}

func TestRunTranscript_RespectsMaxBytesFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	markerTime := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	may10 := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "opencode://ses1", Mtime: may10, Source: "claude"},
	}}
	reader := &fakeReader{contents: map[string]string{
		"opencode://ses1": `{"type":"user","message":{"content":"hello"}}`,
	}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		MaxBytes:      1000,
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("hello"))
}

func TestRunTranscript_ResumesReaderFromMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Bug 2: the per-source marker must be threaded into the reader as the
	// resume point (fromTime) on the FIRST read of that source. Otherwise a
	// session too large for one byte budget is re-read from the start every
	// run and the marker never advances past the first chunk.
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	// Marker mid-session at a fixed past date; entry Mtime is after the marker
	// but well before real now, so it passes the [from, now] filter.
	markerTime := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	may20 := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

	finder := &fakeFinder{entries: []transcript.FileEntry{
		{Path: "/session", Mtime: may20, Source: "claude"},
	}}
	reader := &fromTimeRecordingReader{contents: map[string]string{
		"/session": "resumed content",
	}}

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
	}, finder, reader, &stdout)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(reader.fromTimes).NotTo(BeEmpty())
	g.Expect(reader.fromTimes[0].Equal(markerTime)).To(BeTrue())
}

func TestRunTranscript_SegmentsFlag_EmitsSegmentLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPathWithSuffix(stateDir, slug, "claude")
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())

	markerTime := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).To(Succeed())

	dir := t.TempDir()
	// A JSONL with two real user asks and one assistant — segments output should
	// have exactly two lines.
	ts1 := "2026-05-10T10:00:00Z"
	ts2 := "2026-05-10T10:02:00Z"
	lines := []string{
		`{"type":"user","timestamp":"` + ts1 + `","message":{"content":"first real ask"}}`,
		`{"type":"assistant","timestamp":"2026-05-10T10:01:00Z","message":{"content":"reply"}}`,
		`{"type":"user","timestamp":"` + ts2 + `","message":{"content":"second real ask"}}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	mtime := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	writeTranscriptFixture(g, dir, "session.jsonl", content, mtime)

	finder, reader := cli.NewTranscriptDepsForTest("")

	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: dir,
		Segments:      true,
	}, finder, reader, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	out := stdout.String()
	outLines := strings.Split(strings.TrimSpace(out), "\n")
	g.Expect(outLines).To(HaveLen(2))
	g.Expect(outLines[0]).To(ContainSubstring(ts1))
	g.Expect(outLines[0]).To(ContainSubstring("first real ask"))
	g.Expect(outLines[1]).To(ContainSubstring(ts2))
	g.Expect(outLines[1]).To(ContainSubstring("second real ask"))
}

// failReader is a test-local Reader that always returns an error.
type failReader struct{}

func (r *failReader) ReadFrom(_ string, _ time.Time, _ int) (transcript.ReadResult, error) {
	return transcript.ReadResult{}, errors.New("read failed")
}

// failWriter is an io.Writer that always returns an error.
type failWriter struct{}

func (f *failWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

// fakeFinder is a test-local Finder that returns configurable entries.
type fakeFinder struct {
	entries        []transcript.FileEntry
	findErr        error
	findCalledWith []string
}

func (f *fakeFinder) Find(dirs ...string) ([]transcript.FileEntry, error) {
	f.findCalledWith = dirs
	return f.entries, f.findErr
}

// fakeReader is a test-local Reader that returns content from a map.
// Honors the budget at file-level granularity: if the mapped content
// is larger than budget, returns empty + Partial=true (declines to
// emit a partial file because the stub has no per-row knowledge).
// LastTimestamp is left zero — tests of marker mechanics rely on the
// emit logic's fallback to entry.Mtime for full reads.
type fakeReader struct{ contents map[string]string }

func (f *fakeReader) ReadFrom(path string, _ time.Time, budget int) (transcript.ReadResult, error) {
	content, ok := f.contents[path]
	if !ok {
		return transcript.ReadResult{}, fmt.Errorf("fakeReader: no content for %s", path)
	}

	if len(content) > budget {
		return transcript.ReadResult{Partial: true}, nil
	}

	return transcript.ReadResult{Content: content, BytesUsed: len(content)}, nil
}

// fakeSegmentsReader is a test-local SegmentsReader that returns pre-configured segments.
type fakeSegmentsReader struct {
	segments map[string][]transcript.Segment
}

func (f *fakeSegmentsReader) SegmentsFrom(
	path string,
	_ time.Time,
	_ int,
) ([]transcript.Segment, error) {
	segs, ok := f.segments[path]
	if !ok {
		return []transcript.Segment{}, nil
	}

	return segs, nil
}

// fromTimeRecordingReader is a test-local Reader that records the fromTime it
// receives on each ReadFrom call (in call order) so tests can assert the
// per-source marker is threaded into the reader as the resume point.
type fromTimeRecordingReader struct {
	contents  map[string]string
	fromTimes []time.Time
}

func (r *fromTimeRecordingReader) ReadFrom(
	path string,
	fromTime time.Time,
	_ int,
) (transcript.ReadResult, error) {
	r.fromTimes = append(r.fromTimes, fromTime)

	content, ok := r.contents[path]
	if !ok {
		return transcript.ReadResult{}, fmt.Errorf("fromTimeRecordingReader: no content for %s", path)
	}

	return transcript.ReadResult{Content: content, BytesUsed: len(content)}, nil
}

// fromTimeRecordingSegmentsReader is a test-local SegmentsReader that records
// the fromTime it receives on each SegmentsFrom call so tests can assert the
// per-source marker is threaded into the segments reader as the resume point.
type fromTimeRecordingSegmentsReader struct {
	segments  map[string][]transcript.Segment
	fromTimes []time.Time
}

func (r *fromTimeRecordingSegmentsReader) SegmentsFrom(
	path string,
	fromTime time.Time,
	_ int,
) ([]transcript.Segment, error) {
	r.fromTimes = append(r.fromTimes, fromTime)

	segs, ok := r.segments[path]
	if !ok {
		return []transcript.Segment{}, nil
	}

	return segs, nil
}

// writeTranscriptFixture writes a JSONL line to dir/<name> and sets its mtime.
// Fails the test immediately via g.Expect if any step fails.
func writeTranscriptFixture(g Gomega, dir, name, line string, mtime time.Time) {
	filePath := filepath.Join(dir, name)

	g.Expect(os.WriteFile(filePath, []byte(line+"\n"), 0o600)).To(Succeed())
	g.Expect(os.Chtimes(filePath, mtime, mtime)).To(Succeed())
}
