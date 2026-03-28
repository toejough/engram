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
	"engram/internal/tomlwriter"
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

// TestREQ22AC2_RecordSurfacingPreservesNonTrackingFields verifies REQ-22 AC(2):
// all non-tracking TOML fields are preserved exactly after RecordSurfacing.
// surfaced_count is preserved; legacy fields (last_surfaced, surfacing_contexts) are stripped.
func TestREQ22AC2_RecordSurfacingPreservesNonTrackingFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	capture := &writeCapture{}

	// fullFieldTOML includes every non-tracking field with distinct test values
	// plus old tracking fields that must be stripped.
	fullFieldTOML := "title = \"REQ-22 Title\"\n" +
		"content = \"REQ-22 content body\"\n" +
		"observation_type = \"pattern\"\n" +
		"concepts = [\"alpha\", \"beta\", \"gamma\"]\n" +
		"keywords = [\"req22\", \"preservation\", \"round-trip\"]\n" +
		"principle = \"REQ-22 principle text\"\n" +
		"anti_pattern = \"REQ-22 anti-pattern text\"\n" +
		"rationale = \"REQ-22 rationale text\"\n" +
		"confidence = \"A\"\n" +
		"created_at = \"2024-03-01T09:00:00Z\"\n" +
		"updated_at = \"2024-06-15T14:30:00Z\"\n" +
		"surfaced_count = 7\n" +
		"last_surfaced = \"2026-01-10T08:00:00Z\"\n" +
		"surfacing_contexts = [\"prompt\", \"session-start\"]\n"

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(fullFieldTOML), nil
		}),
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(capture.createTemp(t)),
			tomlwriter.WithRename(func(_, _ string) error { return nil }),
			tomlwriter.WithRemove(func(_ string) error { return nil }),
		)),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/req22.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	record := capture.decodeTOML(g)

	// Non-tracking fields must be preserved exactly.
	g.Expect(record.Title).To(Equal("REQ-22 Title"))
	g.Expect(record.Content).To(Equal("REQ-22 content body"))
	g.Expect(record.ObservationType).To(Equal("pattern"))
	g.Expect(record.Concepts).To(Equal([]string{"alpha", "beta", "gamma"}))
	g.Expect(record.Keywords).To(Equal([]string{"req22", "preservation", "round-trip"}))
	g.Expect(record.Principle).To(Equal("REQ-22 principle text"))
	g.Expect(record.AntiPattern).To(Equal("REQ-22 anti-pattern text"))
	g.Expect(record.Rationale).To(Equal("REQ-22 rationale text"))
	g.Expect(record.Confidence).To(Equal("A"))
	g.Expect(record.CreatedAt).To(Equal("2024-03-01T09:00:00Z"))
	g.Expect(record.UpdatedAt).To(Equal("2024-06-15T14:30:00Z"))

	// surfaced_count is preserved (it is a real tracking field in MemoryRecord).
	g.Expect(record.SurfacedCount).To(Equal(7))

	// Legacy fields (last_surfaced, surfacing_contexts) must be stripped.
	data, readErr := os.ReadFile(capture.tmpPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	raw := string(data)
	g.Expect(raw).To(ContainSubstring("surfaced_count"))
	g.Expect(raw).NotTo(ContainSubstring("last_surfaced ="))
	g.Expect(raw).NotTo(ContainSubstring("surfacing_contexts"))
}

// TestT353_RecordSurfacingPreservesTrackingFields verifies that feedback
// counters survive a RecordSurfacing cycle (#353).
func TestT353_RecordSurfacingPreservesTrackingFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	capture := &writeCapture{}

	tomlWithTracking := baseTOML +
		"surfaced_count = 5\n" +
		"followed_count = 3\n" +
		"contradicted_count = 1\n" +
		"ignored_count = 2\n" +
		"irrelevant_count = 4\n" +
		"last_surfaced_at = \"2026-01-03T00:00:00Z\"\n"

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(tomlWithTracking), nil
		}),
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(capture.createTemp(t)),
			tomlwriter.WithRename(func(_, _ string) error { return nil }),
			tomlwriter.WithRemove(func(_ string) error { return nil }),
		)),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "tool")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	record := capture.decodeTOML(g)

	g.Expect(record.FollowedCount).To(Equal(3))
	g.Expect(record.ContradictedCount).To(Equal(1))
	g.Expect(record.IgnoredCount).To(Equal(2))
	g.Expect(record.IrrelevantCount).To(Equal(4))
	g.Expect(record.SurfacedCount).To(Equal(5))
	g.Expect(record.LastSurfacedAt).To(Equal("2026-01-03T00:00:00Z"))

	// Also verify raw TOML contains tracking keys.
	data, readErr := os.ReadFile(capture.tmpPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	raw := string(data)
	g.Expect(raw).To(ContainSubstring("followed_count"))
	g.Expect(raw).To(ContainSubstring("contradicted_count"))
	g.Expect(raw).To(ContainSubstring("ignored_count"))
	g.Expect(raw).To(ContainSubstring("irrelevant_count"))
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
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(capture.createTemp(t)),
			tomlwriter.WithRename(func(_, _ string) error { return nil }),
			tomlwriter.WithRemove(func(_ string) error { return nil }),
		)),
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

// T-77: Tracking fields are preserved; legacy-only fields are stripped on re-write.
func TestT77_RecordSurfacingPreservesTrackingFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	capture := &writeCapture{}

	// TOML with surfaced_count (real field) and legacy fields that should be stripped.
	existingTOML := baseTOML + "surfaced_count = 3\n" +
		"last_surfaced = \"2026-02-01T00:00:00Z\"\n" +
		"surfacing_contexts = [\"prompt\", \"tool\", \"session-start\"]\n"

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(existingTOML), nil
		}),
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(capture.createTemp(t)),
			tomlwriter.WithRename(func(_, _ string) error { return nil }),
			tomlwriter.WithRemove(func(_ string) error { return nil }),
		)),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "tool")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Read raw output to verify real tracking field is present, legacy fields absent.
	data, readErr := os.ReadFile(capture.tmpPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	raw := string(data)
	g.Expect(raw).To(ContainSubstring("surfaced_count"))
	g.Expect(raw).NotTo(ContainSubstring("surfacing_contexts"))

	// Content fields still present.
	record := capture.decodeTOML(g)
	g.Expect(record.Title).To(Equal("Test Memory"))
	g.Expect(record.SurfacedCount).To(Equal(3))
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
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(capture.createTemp(t)),
			tomlwriter.WithRename(func(_, _ string) error { return nil }),
			tomlwriter.WithRemove(func(_ string) error { return nil }),
		)),
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
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp((&writeCapture{}).createTemp(t)),
			tomlwriter.WithRename(func(_, _ string) error { return nil }),
			tomlwriter.WithRemove(func(_ string) error { return nil }),
		)),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(nowCalled).To(BeTrue())
}

// TestWriteAtomicCreateTempFailure verifies error path when createTemp fails.
func TestWriteAtomicCreateTempFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(baseTOML), nil
		}),
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(func(_, _ string) (*os.File, error) {
				return nil, errors.New("disk full")
			}),
		)),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("disk full"))
}

// TestWriteAtomicEncodeFailure verifies error path when the temp file
// is not writable (encode fails on write).
func TestWriteAtomicEncodeFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(baseTOML), nil
		}),
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(func(_, pattern string) (*os.File, error) {
				f, createErr := os.CreateTemp(t.TempDir(), pattern)
				if createErr != nil {
					return nil, createErr
				}

				// Close immediately so the encoder write fails.
				_ = f.Close()

				return f, nil
			}),
			tomlwriter.WithRemove(func(_ string) error { return nil }),
		)),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).To(HaveOccurred())
}

// TestWriteAtomicRenameFailure verifies error path when rename fails —
// temp file should be cleaned up.
func TestWriteAtomicRenameFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	removeCalled := false

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(baseTOML), nil
		}),
		track.WithWriter(tomlwriter.New(
			tomlwriter.WithCreateTemp(func(_, pattern string) (*os.File, error) {
				return os.CreateTemp(t.TempDir(), pattern)
			}),
			tomlwriter.WithRename(func(_, _ string) error {
				return errors.New("permission denied")
			}),
			tomlwriter.WithRemove(func(_ string) error {
				removeCalled = true

				return nil
			}),
		)),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "prompt")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("permission denied"))

	// Verify temp file cleanup was attempted.
	g.Expect(removeCalled).To(BeTrue())
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

// fullTOMLRecord deleted — replaced by memory.MemoryRecord.

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

func (w *writeCapture) decodeTOML(g Gomega) memory.MemoryRecord {
	g.Expect(w.tmpPath).NotTo(BeEmpty(), "no temp file was written")

	data, err := os.ReadFile(w.tmpPath)
	g.Expect(err).NotTo(HaveOccurred())

	var record memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	return record
}
