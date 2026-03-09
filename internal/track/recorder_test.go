package track_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/track"
)

// TestEmptyFilePathSkipped verifies that memories with empty FilePath are skipped.
func TestEmptyFilePathSkipped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	readCalled := false

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			readCalled = true

			return nil, errors.New("should not be called")
		}),
	)

	memories := []*memory.Stored{
		{FilePath: ""},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(readCalled).To(BeFalse())
}

// T-76: RecordSurfacing re-writes TOML preserving content fields and stripping
// old tracking fields (surfaced_count, last_surfaced, surfacing_contexts).
func TestT76_RecordSurfacingPreservesContentFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	capture := &writeCapture{}

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(baseTOML), nil
		}),
		track.WithCreateTemp(capture.createTemp(t)),
		track.WithRename(func(_, _ string) error { return nil }),
		track.WithRemove(func(_ string) error { return nil }),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "session-start")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	record := capture.decodeTOML(g)

	// Content fields preserved.
	g.Expect(record.Title).To(Equal("Test Memory"))
	g.Expect(record.Content).To(Equal("some content"))
	g.Expect(record.ObservationType).To(Equal("correction"))
	g.Expect(record.Concepts).To(Equal([]string{"testing"}))
	g.Expect(record.Keywords).To(Equal([]string{"test"}))
	g.Expect(record.Principle).To(Equal("test principle"))
	g.Expect(record.AntiPattern).To(Equal("test anti-pattern"))
	g.Expect(record.Rationale).To(Equal("test rationale"))
	g.Expect(record.Confidence).To(Equal("B"))
	g.Expect(record.CreatedAt).To(Equal("2025-01-01T00:00:00Z"))
	g.Expect(record.UpdatedAt).To(Equal("2025-06-01T00:00:00Z"))
}

// T-77: Existing tracking fields in TOML are stripped on re-write.
func TestT77_RecordSurfacingStripsTrackingFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	capture := &writeCapture{}

	// TOML with old tracking fields that should be stripped.
	existingTOML := baseTOML + "surfaced_count = 3\n" +
		"last_surfaced = \"2026-02-01T00:00:00Z\"\n" +
		"surfacing_contexts = [\"prompt\", \"tool\", \"session-start\"]\n"

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(existingTOML), nil
		}),
		track.WithCreateTemp(capture.createTemp(t)),
		track.WithRename(func(_, _ string) error { return nil }),
		track.WithRemove(func(_ string) error { return nil }),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "tool")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Read raw output to verify tracking fields are absent.
	data, readErr := os.ReadFile(capture.tmpPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	raw := string(data)
	g.Expect(raw).NotTo(ContainSubstring("surfaced_count"))
	g.Expect(raw).NotTo(ContainSubstring("last_surfaced"))
	g.Expect(raw).NotTo(ContainSubstring("surfacing_contexts"))

	// Content fields still present.
	record := capture.decodeTOML(g)
	g.Expect(record.Title).To(Equal("Test Memory"))
}

// T-78: Two memories, first has unreadable path → first skipped, second updated.
func TestT78_PartialFailureContinues(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	capture := &writeCapture{}

	readFile := func(path string) ([]byte, error) {
		if path == "/nonexistent/bad.toml" {
			return nil, errors.New("file not found")
		}

		return []byte(baseTOML), nil
	}

	recorder := track.NewRecorder(
		track.WithReadFile(readFile),
		track.WithCreateTemp(capture.createTemp(t)),
		track.WithRename(func(_, _ string) error { return nil }),
		track.WithRemove(func(_ string) error { return nil }),
	)

	memories := []*memory.Stored{
		{FilePath: "/nonexistent/bad.toml"},
		{FilePath: "/fake/valid.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")

	// Should return an error for the first memory but still process the second.
	g.Expect(err).To(HaveOccurred())

	// Second file should be re-written with content preserved.
	record := capture.decodeTOML(g)
	g.Expect(record.Title).To(Equal("Test Memory"))
}

// TestWithNow verifies the WithNow option injects a custom time provider.
func TestWithNow(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	customTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	nowCalled := false

	recorder := track.NewRecorder(
		track.WithNow(func() time.Time {
			nowCalled = true

			return customTime
		}),
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(baseTOML), nil
		}),
		track.WithCreateTemp((&writeCapture{}).createTemp(t)),
		track.WithRename(func(_, _ string) error { return nil }),
		track.WithRemove(func(_ string) error { return nil }),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(nowCalled).To(BeTrue())
}

// TestWriteAtomicCreateTempFailure verifies writeAtomic error path when
// createTemp fails.
func TestWriteAtomicCreateTempFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(baseTOML), nil
		}),
		track.WithCreateTemp(func(_, _ string) (*os.File, error) {
			return nil, errors.New("disk full")
		}),
		track.WithRename(func(_, _ string) error { return nil }),
		track.WithRemove(func(_ string) error { return nil }),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("disk full"))
}

// TestWriteAtomicEncodeFailure verifies writeAtomic error path when
// the temp file is not writable (encode fails on write).
func TestWriteAtomicEncodeFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(baseTOML), nil
		}),
		track.WithCreateTemp(func(_, pattern string) (*os.File, error) {
			f, createErr := os.CreateTemp(t.TempDir(), pattern)
			if createErr != nil {
				return nil, createErr
			}

			// Close immediately so the encoder write fails.
			_ = f.Close()

			return f, nil
		}),
		track.WithRename(func(_, _ string) error { return nil }),
		track.WithRemove(func(_ string) error { return nil }),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).To(HaveOccurred())
}

// TestWriteAtomicRenameFailure verifies writeAtomic error path when
// rename fails — temp file should be cleaned up.
func TestWriteAtomicRenameFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var removedPath string

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(baseTOML), nil
		}),
		track.WithCreateTemp(func(_, pattern string) (*os.File, error) {
			return os.CreateTemp(t.TempDir(), pattern)
		}),
		track.WithRename(func(_, _ string) error {
			return errors.New("permission denied")
		}),
		track.WithRemove(func(path string) error {
			removedPath = path

			return nil
		}),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("permission denied"))

	// Verify temp file cleanup was attempted.
	g.Expect(removedPath).NotTo(BeEmpty())
}

// unexported constants.
const (
	baseTOML = "title = \"Test Memory\"\n" +
		"content = \"some content\"\n" +
		"observation_type = \"correction\"\n" +
		"concepts = [\"testing\"]\n" +
		"keywords = [\"test\"]\n" +
		"principle = \"test principle\"\n" +
		"anti_pattern = \"test anti-pattern\"\n" +
		"rationale = \"test rationale\"\n" +
		"confidence = \"B\"\n" +
		"created_at = \"2025-01-01T00:00:00Z\"\n" +
		"updated_at = \"2025-06-01T00:00:00Z\"\n"
)

// fullTOMLRecord mirrors content TOML fields for test verification.
// Tracking fields are no longer included (UC-23).
type fullTOMLRecord struct {
	Title           string   `toml:"title"`
	Content         string   `toml:"content"`
	ObservationType string   `toml:"observation_type"`
	Concepts        []string `toml:"concepts"`
	Keywords        []string `toml:"keywords"`
	Principle       string   `toml:"principle"`
	AntiPattern     string   `toml:"anti_pattern"`
	Rationale       string   `toml:"rationale"`
	Confidence      string   `toml:"confidence"`
	CreatedAt       string   `toml:"created_at"`
	UpdatedAt       string   `toml:"updated_at"`
}

// writeCapture tracks the temp file path so its content can be read back
// after the recorder closes it. This avoids real filesystem I/O for setup
// and verification — only the createTemp DI seam uses a real temp file.
type writeCapture struct {
	tmpPath string
}

func (w *writeCapture) createTemp(
	t *testing.T,
) func(string, string) (*os.File, error) {
	t.Helper()

	return func(_, pattern string) (*os.File, error) {
		f, err := os.CreateTemp(t.TempDir(), pattern)
		if err != nil {
			return nil, err
		}

		w.tmpPath = f.Name()

		return f, nil
	}
}

func (w *writeCapture) decodeTOML(g Gomega) fullTOMLRecord {
	g.Expect(w.tmpPath).NotTo(BeEmpty(), "no temp file was written")

	data, err := os.ReadFile(w.tmpPath)
	g.Expect(err).NotTo(HaveOccurred())

	var record fullTOMLRecord

	_, decodeErr := toml.Decode(string(data), &record)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	return record
}
