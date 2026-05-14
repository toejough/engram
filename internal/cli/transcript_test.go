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

func TestAdvanceAndReportMarker_StatusLineContainsBothFromAndEffectiveEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	markerPath := filepath.Join(tmp, "marker.txt")

	fromTime := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	lastIncluded := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 30, 0, 0, time.UTC)

	var stdout bytes.Buffer

	err := cli.AdvanceAndReportMarkerForTest(markerPath, fromTime, lastIncluded, true, now, &stdout)
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

	err := cli.AdvanceAndReportMarkerForTest(markerPath, fromTime, lastIncluded, true, now, &stdout)
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
	err := cli.AdvanceAndReportMarkerForTest(markerPath, fromTime, now, true, now, &stdout)
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

	err := cli.AdvanceAndReportMarkerForTest(markerPath, fromTime, time.Time{}, false, now, &stdout)
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

		err := cli.RunTranscriptForTest(context.Background(), cli.TranscriptArgs{}, finder, reader, &stdout)

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

func TestEmitTranscripts_AlwaysIncludesFirstEntryEvenWhenOversized(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Single entry larger than cap — progress guarantee includes it anyway,
	// otherwise the marker would never advance past it.
	mkContent := func(prefix string) string { return prefix + strings.Repeat("x", 999) }
	reader := &fakeReader{contents: map[string]string{
		"/a": mkContent("A"),
	}}
	may1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	entries := []transcript.FileEntry{{Path: "/a", Mtime: may1, Source: "claude"}}

	var buf bytes.Buffer

	lastIncluded, hadEntries, err := cli.EmitTranscriptsForTest(reader, entries, 100, &buf)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(hadEntries["claude"]).To(BeTrue())
	g.Expect(lastIncluded["claude"].Equal(may1)).To(BeTrue())
	g.Expect(buf.String()).To(ContainSubstring("A"))
}

func TestEmitTranscripts_NoEntriesReturnsZeroAndFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reader := &fakeReader{contents: map[string]string{}}

	var buf bytes.Buffer

	lastIncluded, hadEntries, err := cli.EmitTranscriptsForTest(reader, nil, 1000, &buf)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(hadEntries).To(BeEmpty())
	g.Expect(lastIncluded).To(BeEmpty())
	g.Expect(buf.Len()).To(Equal(0))
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

	lastIncluded, hadEntries, err := cli.EmitTranscriptsForTest(reader, entries, 150, &buf)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(hadEntries["claude"]).To(BeTrue())
	g.Expect(lastIncluded["claude"].Equal(may1)).To(BeTrue())

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
		line := `{"type":"user","message":{"content":"hello from transcript"}}`
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
		line := `{"type":"user","message":{"content":"afternoon message"}}`
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
		line := `{"type":"assistant","message":{"content":"rfc3339 message"}}`
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

// failReader is a test-local Reader that always returns an error.
type failReader struct{}

func (r *failReader) Read(_ string, _ int) (string, int, error) {
	return "", 0, errors.New("read failed")
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
type fakeReader struct{ contents map[string]string }

func (f *fakeReader) Read(path string, _ int) (string, int, error) {
	c, ok := f.contents[path]
	if !ok {
		return "", 0, fmt.Errorf("fakeReader: no content for %s", path)
	}

	return c, len(c), nil
}

// writeTranscriptFixture writes a JSONL line to dir/<name> and sets its mtime.
// Fails the test immediately via g.Expect if any step fails.
func writeTranscriptFixture(g Gomega, dir, name, line string, mtime time.Time) {
	filePath := filepath.Join(dir, name)

	g.Expect(os.WriteFile(filePath, []byte(line+"\n"), 0o600)).To(Succeed())
	g.Expect(os.Chtimes(filePath, mtime, mtime)).To(Succeed())
}
