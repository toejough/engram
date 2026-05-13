package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/learnmarker"
	"github.com/toejough/engram/internal/transcript"
)

func TestApplyTranscriptDirDefault(t *testing.T) {
	t.Parallel()

	t.Run("uses provided dir when non-empty", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		out, err := runTranscript(context.Background(), cli.TranscriptArgs{
			From:          "2026-05-10",
			To:            "2026-05-10",
			TranscriptDir: t.TempDir(),
		})

		// Empty dir, no files — no error, empty output.
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(BeEmpty())
	})

	t.Run("derives dir from cwd when transcript-dir empty and no slug", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// No TranscriptDir and no ProjectSlug — derives from os.Getwd().
		// The derived path will not exist, but the lister silently skips missing dirs.
		out, err := runTranscript(context.Background(), cli.TranscriptArgs{
			From: "2026-05-10",
			To:   "2026-05-10",
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(BeEmpty())
	})

	t.Run("uses project-slug when transcript-dir empty", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Non-existent slug-derived path; lister skips it silently.
		out, err := runTranscript(context.Background(), cli.TranscriptArgs{
			From:        "2026-05-10",
			To:          "2026-05-10",
			ProjectSlug: "-test-project",
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(BeEmpty())
	})
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

func TestRunTranscript_Errors(t *testing.T) {
	t.Parallel()

	t.Run("invalid --from date returns error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		_, err := runTranscript(context.Background(), cli.TranscriptArgs{
			From:          "not-a-date",
			To:            "2026-05-10",
			TranscriptDir: t.TempDir(),
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid date"))
	})

	t.Run("invalid --to date returns error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		_, err := runTranscript(context.Background(), cli.TranscriptArgs{
			From:          "2026-05-10",
			To:            "not-a-date",
			TranscriptDir: t.TempDir(),
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("invalid date"))
	})
}

func TestRunTranscript_Filtering(t *testing.T) {
	t.Parallel()

	t.Run("out-of-range mtime produces no output", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		dir := t.TempDir()
		line := `{"type":"user","message":{"content":"old message"}}`
		mtime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		writeTranscriptFixture(g, dir, "old.jsonl", line, mtime)

		out, err := runTranscript(context.Background(), cli.TranscriptArgs{
			From:          "2026-05-10",
			To:            "2026-05-10",
			TranscriptDir: dir,
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(BeEmpty())
	})

	t.Run("empty transcript dir produces no output", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		out, err := runTranscript(context.Background(), cli.TranscriptArgs{
			From:          "2026-05-10",
			To:            "2026-05-10",
			TranscriptDir: t.TempDir(),
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(BeEmpty())
	})
}

func TestRunTranscript_HappyPath(t *testing.T) {
	t.Parallel()

	t.Run("emits stripped content for in-range file", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		dir := t.TempDir()
		line := `{"type":"user","message":{"content":"hello from transcript"}}`
		mtime := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
		writeTranscriptFixture(g, dir, "session.jsonl", line, mtime)

		out, err := runTranscript(context.Background(), cli.TranscriptArgs{
			From:          "2026-05-10",
			To:            "2026-05-10",
			TranscriptDir: dir,
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(ContainSubstring("hello from transcript"))
	})

	t.Run("inclusive: file at 15:00 on --to date is included", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		dir := t.TempDir()
		line := `{"type":"user","message":{"content":"afternoon message"}}`
		mtime := time.Date(2026, 5, 11, 15, 0, 0, 0, time.UTC)
		writeTranscriptFixture(g, dir, "afternoon.jsonl", line, mtime)

		out, err := runTranscript(context.Background(), cli.TranscriptArgs{
			From:          "2026-05-11",
			To:            "2026-05-11",
			TranscriptDir: dir,
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(ContainSubstring("afternoon message"))
	})

	t.Run("RFC3339 from/to accepted with assistant message", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		dir := t.TempDir()
		line := `{"type":"assistant","message":{"content":"rfc3339 message"}}`
		mtime := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
		writeTranscriptFixture(g, dir, "rfc.jsonl", line, mtime)

		out, err := runTranscript(context.Background(), cli.TranscriptArgs{
			From:          "2026-05-10T00:00:00Z",
			To:            "2026-05-10T23:59:59Z",
			TranscriptDir: dir,
		})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(out).To(ContainSubstring("rfc3339 message"))
	})
}

func TestResolveTimeWindow_UsesMarkerWhenFromMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	markerTime := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	from, to, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{From: "", To: "", Marker: markerTime, MarkerFound: true, Now: now},
	)

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(from.Equal(markerTime)).To(BeTrue())
	g.Expect(to.Equal(now)).To(BeTrue())
}

func TestResolveTimeWindow_FallsBackTo24hWhenNoMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC)

	from, to, err := cli.ResolveTimeWindow(
		cli.TimeWindowInputs{From: "", To: "", MarkerFound: false, Now: now},
	)

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(from.Equal(now.Add(-24 * time.Hour))).To(BeTrue())
	g.Expect(to.Equal(now)).To(BeTrue())
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

// failReader is a test-local Reader that always returns an error.
type failReader struct{}

func (r *failReader) Read(_ string, _ int) (string, int, error) {
	return "", 0, errors.New("read failed")
}

// runTranscript is a test-local shorthand.
func runTranscript(_ context.Context, args cli.TranscriptArgs) (string, error) {
	var stdout bytes.Buffer

	err := cli.RunTranscriptForTest(args, &stdout)

	return stdout.String(), err
}

func TestRunTranscript_AcceptsEmptyFromAndToWhenMarkerExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"
	markerPath := learnmarker.MarkerPath(stateDir, slug)
	g.Expect(os.MkdirAll(filepath.Dir(markerPath), 0o755)).To(Succeed())
	markerTime := time.Now().Add(-2 * time.Hour).UTC()
	g.Expect(os.WriteFile(markerPath, []byte(markerTime.Format(time.RFC3339Nano)), 0o644)).
		To(Succeed())

	var stdout bytes.Buffer
	err := cli.RunTranscriptForTest(cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(), // empty dir; we only care that flags resolved
	}, &stdout)

	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunTranscript_MarkFlagAdvancesMarkerToNow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, ".local", "state", "engram")
	slug := "Users-joe-repos-test"

	var stdout bytes.Buffer
	before := time.Now().UTC()
	err := cli.RunTranscriptForTest(cli.TranscriptArgs{
		ProjectSlug:   slug,
		StateDir:      stateDir,
		TranscriptDir: t.TempDir(),
		Mark:          true,
	}, &stdout)
	after := time.Now().UTC()

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	got, _ := os.ReadFile(learnmarker.MarkerPath(stateDir, slug))
	parsed, parseErr := time.Parse(time.RFC3339Nano, string(got))
	g.Expect(parseErr).NotTo(HaveOccurred())
	if parseErr != nil {
		return
	}
	g.Expect(parsed.After(before.Add(-time.Second)) && parsed.Before(after.Add(time.Second))).
		To(BeTrue())
}

// writeTranscriptFixture writes a JSONL line to dir/<name> and sets its mtime.
// Fails the test immediately via g.Expect if any step fails.
func writeTranscriptFixture(g Gomega, dir, name, line string, mtime time.Time) {
	filePath := filepath.Join(dir, name)

	g.Expect(os.WriteFile(filePath, []byte(line+"\n"), 0o600)).To(Succeed())
	g.Expect(os.Chtimes(filePath, mtime, mtime)).To(Succeed())
}
